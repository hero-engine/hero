---
title: Hero Agents Home — Sessions, Proposals, Scheduled, Automations, Health
slug: hero-agents-home
type: feature
status: completed
tags: [serve, surface, agents, sessions, proposals, scheduled, automations, web-app]
created: 2026-05-17
relations:
  - target: hero-surface-architecture
    kind: parent
  - target: hero-surface-shell
    kind: depends-on
  - target: hero-surface-deployment-and-rendering
    kind: depends-on
  - target: hero-chat-and-model
    kind: depends-on
  - target: hero-automations
    kind: absorbs
  - target: inline-propose-output-mode
    kind: relates-to
  - target: agent-contribution-tracking
    kind: relates-to
  - target: cost-calibration
    kind: relates-to
horizon: now
completed_at: 2026-05-18T19:25:38Z
---

## Context

`hero serve` is a **web app companion to the CLI**
([hero-surface-deployment-and-rendering](../hero-surface-deployment-and-rendering/spec.md)).
The Agents home is the page where everything autonomous lives:
the user opens a browser tab to `/agents` and can see every live
session, every pending proposal, every scheduled run, every
automation rule, and per-agent health stats — all on one scrolling
web-app page with a slim sub-nav, not a desktop shell.

Hero has the plumbing for all of this already — `internal/serve/jobs.go`
(534 lines) holds the job queue and session register;
`internal/serve/workers.go` (165 lines) runs the worker pool;
`internal/serve/scheduled.go` (101 lines) wraps the `scheduled-tasks`
MCP and the cron tick loop; `internal/serve/proposals.go` (285 lines)
ships the pending proposal store and the accept/edit/reject endpoints
from [inline-propose-output-mode](../inline-propose-output-mode/spec.md).
What it doesn't have is the **automations engine** (absorbed from the
standalone [hero-automations](../hero-automations/spec.md) spec) and
the **UI** that surfaces all of this to a human in a browser.

This spec owns that UI. The visual source of truth is
[mockups/01-agents-sessions.html](mockups/01-agents-sessions.html):
a scrolling web-app page (~1200px centered), slim sub-nav row, a
tabbed metric strip, live-session blocks separated by hairline
borders (not cards, not a desktop grid), light-grey transcript
panels (`#f7f8fa`) with dark text and a hero-blue blinking cursor —
**no dark terminal**.

Boundaries with the surrounding subsystems are sharp.
[hero-chat-and-model](../hero-chat-and-model/spec.md) owns chat
dispatch, the Hero adapter abstraction, the deferred-run queue, the
`miss_policy` semantics, and the slash command registry. The Agents
home **renders state** from that subsystem — it does not implement
dispatch, deferred-run logic, or adapter selection. The
[hero-surface-shell](../hero-surface-shell/spec.md) owns the top nav,
search pill, avatar, footer, and per-pack routing. Agents owns the
page body at `/agents` and everything below the sub-nav row.

## Goal

A `/agents` page in `hero serve` that:

1. Lands on a Sessions view by default, showing live and recent
   agent sessions as a vertical list of separator-line **blocks**
   (not cards), with light-grey transcript previews that stream
   token-by-token via SSE.
2. Carries a slim sub-nav row with six tabs:
   `Sessions · Proposals · Scheduled · Automations · Health · Credentials`.
   The `Credentials` tab renders faded under `HERO_EDITION=local`
   with the meta `team server only`.
3. Shows a tabbed metric strip near the top with three tabs —
   **Right now**, **Today**, **Health (7d)** — that swap tile
   content in place.
4. Routes each sub-nav tab to its own page with its own list view
   and per-item detail tab; new automations get a dedicated builder
   page.
5. Subscribes to `session.*`, `proposal.*`, `scheduled.*`,
   `deferred.*`, and `automation.*` SSE events to update the page
   without reload.
6. Renders the proposals queue as flat rows with inline
   approve / view-diff / reject actions, plus a richer diff-viewer
   island for `/agents/proposals/<id>`.
7. Renders scheduled agents as a cron-aware list with deferred-run
   visibility per [hero-chat-and-model](../hero-chat-and-model/spec.md).
8. Renders automation rules (trigger → action) absorbed from
   [hero-automations](../hero-automations/spec.md) with a per-rule
   detail tab, dry-run button, and a dedicated builder island at
   `/agents/automations/new`.
9. Renders per-agent reliability stats on `/agents/health`.

When this lands, a developer opens `/agents`, sees three live
session blocks (with the prominent one streaming text into a
light-grey panel), spots two pending proposals on the metric strip,
clicks `Approve` inline on the lighter proposal queue below, watches
the row fade out via SSE, and clicks through to a scheduled-runs
list that shows tomorrow's `daily-executive-report` and an
in-flight `tracker-import-sync`.

## Approach

### Routes and views

Hero serve is a Go-template-and-SSE web app; islands hydrate only
where templates can't carry the interaction
([decision spec §Rendering model](../hero-surface-deployment-and-rendering/spec.md)).
The Agents home owns one top-level route, six sub-routes, and four
per-item detail routes:

| Route | Renderer | Purpose |
|---|---|---|
| `/agents` | template + SSE | Sessions view (default) — metric strip + live blocks + awaiting-approval + completed-today + scheduled/automations preview |
| `/agents/session/<id>` | template + island (`agent-session.js`) | Full live transcript, full tool inventory, full action set |
| `/agents/proposals` | template + SSE | Proposals queue (flat row list) |
| `/agents/proposals/<id>` | template + island (`diff-viewer.js`) | Diff viewer + approve/edit/reject |
| `/agents/scheduled` | template + SSE | Scheduled cron-style agents with next-run / last-run / deferred-runs |
| `/agents/scheduled/<id>` | template + SSE | Run history + edit form |
| `/agents/automations` | template + SSE | Automation rules list |
| `/agents/automations/<id>` | template + SSE | Execution log + dry-run button |
| `/agents/automations/new` | template + island (`automation-builder.js`) | Trigger/action rule builder |
| `/agents/health` | template | Per-agent scoreboard |
| `/agents/credentials` | template | Faded under `local`; broker-state read-only under `team`+ |

The five existing view-registry slugs from the absorbed-on-paper
shell record (`agents-sessions`, `agents-proposals`, …) are kept
verbatim so the shell's openables manifest already wires this home
in; the Sessions slug is `default_view`.

### The Sessions view (default `/agents`)

The visual source of truth is
[mockups/01-agents-sessions.html](mockups/01-agents-sessions.html).
Top-down:

**Sub-nav row** (44px, white background, hairline bottom border)

`Sessions 3` (active, hero-blue underline) · `Proposals 2`
(amber-tinted badge) · `Scheduled 7` · `Automations 5` · `Health` ·
`Credentials` (faded, lock icon, meta `· team server only`).

The sub-nav is a shared template partial
(`internal/serve/pages/agents/templates/sub-nav.html`) that all
agent pages reuse with the active tab parameterized. The active tab
gets the hero-blue 2px bottom border and `font-weight: 600` per the
mock; the inactive `Credentials` tab is rendered with `color: var(--ink-5)`,
a lock SVG, and no underline.

**Page hero** (32px page title, 40px top padding)

- Eyebrow: `hero · main · agents` with a small bolt SVG.
- Title: `Sessions`.
- Subhead: one-liner combining four numbers in plain text — `3 live · 12 today · $4.27 spent today · 2 awaiting your approval`. Live / spend / pending are bolded.
- Inline action row (right side, wraps under at narrow viewports):
  - Primary: `+ Start session` (hero-blue fill, white text).
  - Ghost link (hero-blue text): `Approve all pending (2)`.
  - Ghost link (muted ink): `Pause my agents`.

**Tabbed metric strip** (24px top padding, hairline bottom border under tabs)

Three text-link tabs above a 4-column grid of metric tiles. Clicking
a tab swaps the tile content in place (CSS class toggle, server
renders all three panes; only the active one is visible). A
`View all metrics →` link sits at the right of the tab row.

- **Right now** (default active):
  1. `live sessions` — value `3` with a pulsing hero-blue dot; sub `opus · sonnet · engineer (paused)`.
  2. `awaiting your approval` — value `2` in amber with a pulsing amber dot; sub lists the affected spec slugs (`tripwire-system`, `per-feature-smoke-coverage`) in mono.
  3. `queue depth` — value `0`; sub `no jobs queued`.
  4. `spend today · budget $20.00` — value `$4.27`; thin hero-blue progress bar showing 21.3% fill.
- **Today** (DOM-rendered but hidden by default `display:none`):
  1. `sessions completed` — `12`; sub `8 merged · 3 review · 1 interrupted`.
  2. `autonomy today` — `84%`; mini sparkline (inline SVG, no chart library).
  3. `top tool · 142 calls` — `Edit`; sub `Read 198 · Bash 64 · Grep 41`.
  4. `total cost today` — `$4.27`; sparkline.
- **Health (7d)** (hidden by default):
  1. `interrupt rate` — `12%`; sub `8 of 67 sessions`.
  2. `approval rate` — `84%`; sub `42 of 50 proposals merged`.
  3. `failure rate` — `3.1%`; sub `2 of 65 completed`.
  4. `cost per merged proposal` — `$0.34`; sparkline.

A small inline `<script>` toggles `.metric-tab.active` and
`.metric-pane.active` on click. The pane content is rendered
server-side so first paint is correct without JS.

**Live sessions section**

Section header: `Live sessions <small>3 running</small>` on the left,
four filter chips on the right (`All` active, `Mine`, `Team`,
`Scheduled`). Filter chips swap visibility via a small inline script;
no XHR round-trip in v1.

The list is a `<div class="sessions-list">` containing stacked
`<article class="session-block">` partials. Blocks are separated by
a top hairline (`border-top: 1px solid var(--border)`) — they are
**not cards**. The first block has no top border and slightly
reduced top padding.

Each block is rendered from a shared partial
(`internal/serve/pages/agents/templates/session-block.html`) that
takes a session record + a `variant` (`prominent` | `compact` |
`amber`).

For the **prominent** variant (Block 1 in the mock — `claude-opus`
on `per-feature-smoke-coverage`):

- **Head row** — circular gradient avatar (32px, opus has
  purple-indigo gradient) + name + on-clause (`delivering <a>spec-slug</a>`)
  + status tag (hero-blue `LIVE` chip with pulsing dot).
- **Meta row** — `started 12m ago · branch <code>feat/...</code> · <span class="cost">$0.42</span> so far · <code>opus-4.7-1m</code> · 14 tool calls · 1 proposal pending` (amber). Indented 44px to align with the avatar gutter.
- **Transcript panel** (`<div class="transcript">`) — **light-grey** `background: #f7f8fa`, dark `color: var(--ink-2)`, monospace, indented 44px, `min-height: 172px`. Renders the last 10-12 streamed lines server-side, with each line carrying a `role` class (`assistant` — hero-blue ink, otherwise muted), inline `tool` spans, and status classes (`ok` green, `pending` amber italic, `danger` red). The final line carries `.streaming` which adds a hero-blue blinking cursor via CSS keyframes. **No dark terminal background — `#f7f8fa` light panel.**
- **Action row** — text-link verbs indented 44px: `Open transcript` (hero-blue primary) · `Open spec` · `Approve all (N)` (hero-blue) · `Reject all` · `Interrupt` (red).
- **Tool inventory strip** — small uppercase `TOOLS` label + mono pills (`Read · 67`, `Edit · 14`, `Bash · 6`, `Grep · 9`) indented 44px.

For the **compact** variant (Block 2 — `claude-sonnet` diagnosing
`scan-enrichment-unbounded-loop`):

Same head/meta, transcript clipped to 4 lines with
`.transcript.compact` (tighter line-height), action row with only
`Open transcript` + `Interrupt`. No tool strip.

For the **amber** variant (Block 3 — `engineer` subagent paused on
`tripwire-system`):

- Head row carries the amber status tag (`AWAITING YOUR APPROVAL`)
  with a pulsing amber dot.
- Meta row includes `1 proposal pending` in amber.
- In place of the transcript, a **proposal-preview panel**
  (`<div class="proposal-preview">`) on `#fffaf2` with an amber
  border. Header inside: bold `Proposal awaiting review` + file
  summary (`2 files · <code>...</code> · +47 / −0`). Body is an
  inline diff snippet (5-8 lines, monospace, light grey `#f7f8fa`
  background per hunk) with `.diff-add` (green) / `.diff-rem` (red)
  / `.diff-ctx` (muted) line classes.
- Action row: `Review proposal` (hero-blue) · `Resume` ·
  `Open transcript` · `Stop` (red).

**Awaiting your approval section**

Section header: `Awaiting your approval <small>2 proposals</small>`
with `View all (2) →` link on the right (links to `/agents/proposals`).

Flat list of `<div class="approval-row">` rows — **not cards**, a
24px amber dot in the left gutter, the proposal summary in the
center (`engineer proposes guard in <code>internal/tripwires/registry.go</code>`),
a metadata line below (`2 files · +47 / −0 · 4m ago · from /deliver on <spec>`),
and inline text-link actions on the right (`Approve`, `View diff`,
`Reject` in red). Hover state: `background: var(--bg-soft)`.

**Completed today section**

Section header: `Completed today <small>12 sessions</small>` with
`View all (12) →` link.

Timeline list of `<div class="timeline-row">` rows in a 4-column
grid: `[relative time | status icon | agent + spec link + outcome | duration · cost]`.
Status icon colors map to outcome: green check for merged, blue
clock for awaiting review, amber warn for interrupted, grey
calendar for scheduled-fired. Hover row: `background: var(--bg-soft)`.

**Scheduled & automations preview section**

Two-column split (`grid-template-columns: 1fr 1fr; gap: 28px`).

Left column: `Next 3 scheduled runs` (small uppercase title) →
compact list of three rows showing name, cron expression in
`<code>` + human description, and right-aligned `when` (relative +
absolute).

Right column: `Top 3 active automations` → compact list of three
rows showing rule name, trigger source + filter + mode, and
right-aligned `fired Nh ago` + `N runs / 7d`.

Each column header carries a section-meta count (`7 total`,
`5 total`). Above the split, two right-aligned links:
`Open Scheduled →` and `Open Automations →`.

### `/agents/session/<id>` (per-session detail)

Reuses the sub-nav (with `Sessions` still active) and page-hero
pattern, but the page body becomes a full-height streaming
transcript driven by the `agent-session.js` island.

- **Run header band** — agent avatar + name, spec link, model,
  branch, started, current turn / max-turns, cost so far / budget,
  status chip.
- **Transcript** — same **light-grey** `#f7f8fa` panel as the preview,
  but full-height (`max-height: 70vh`, scrollable, autoscroll on
  new tokens unless the user scrolls up). Each Claude API turn
  renders as a block: assistant text streams via `session.token`
  events; tool calls render inline with a collapsible argument
  JSON; tool results render under the call with a status icon
  (`ok` / `pending` / `danger`); reasoning blocks render as a quiet
  collapsed accordion. The cursor lives on whichever line is
  currently streaming.
- **Pending action banner** — when status is `awaiting_approval`,
  an amber strip appears above the transcript with inline
  `Approve` / `Reject` / `Edit & approve` actions wired to
  `/api/agents/proposals/<id>/{approve,reject,edit-approve}`.
- **Action row** at the bottom of the run header: `Interrupt` (red)
  · `Approve all pending (N)` · `Reject all pending` · `Open spec`
  · `Copy transcript`.
- **Tool inventory strip** beneath the action row.

The island subscribes to
`GET /api/agents/sessions/<id>/events?cursor=<last_event_id>` (SSE).
On reconnect it re-plays events from the cursor; the daemon
maintains a per-session ring buffer of the last N events.

### `/agents/proposals`

Sub-nav `Proposals` active. Page-hero title `Proposals` with
subhead `<n> awaiting · <m> resolved today`. Inline actions:
`Approve all (n)` and a filter chip row (`All`, `Mine`, `By agent`).

Body is a single flat list reusing the `<div class="approval-row">`
partial from the Sessions view. Each row carries one batch
(`batch_id` from [inline-propose-output-mode](../inline-propose-output-mode/spec.md))
with file count, ± line totals, age, originating slash + spec, and
inline `Approve` / `View diff` / `Reject` text-link actions. SSE on
`proposal.created` slides a new row in; `proposal.resolved` fades the
row.

### `/agents/proposals/<id>` (proposal detail island)

`diff-viewer.js` island hosts:

- **Left rail** — list of proposals in the batch with status pills
  (`pending` / `accepted` / `edited` / `rejected`).
- **Center pane** — for each proposal, render anchor-aware:
  - `section-append` / `list-item-append`: dotted-border insertion
    block at the anchor location, matching the locked visual in
    the inline-propose mockup.
  - `frontmatter-field`: side-by-side before/after.
  - `section-replace`: split diff (no three-way merge in v1).
- **Per-proposal actions**: `Accept`, `Edit & accept` (opens inline
  markdown editor), `Reject` (with optional reason input).
- **Batch actions** at the bottom: `Accept all pending`,
  `Reject all pending`. Wired to the existing
  `/api/{project}/sessions/{sid}/proposals/{pid}/{accept,edit-accept,reject}`
  endpoints from inline-propose-output-mode.

### `/agents/scheduled`

Sub-nav `Scheduled` active. Page-hero title `Scheduled` with
subhead `<n> recurring · next in <relative> · <m> deferred`. Inline
action: `+ New scheduled` (opens a small builder modal sharing
form fields with the automation builder, with the trigger fixed to
`schedule`).

Body is a table of rows, each carrying: name, cron expression
(`<code>`), action (`hero <command> <args>`), mode
(`autopilot` / `gated`), next run (computed), last run (relative,
with status icon), last result, 30d cost. Selecting a row exposes
a verb strip below: `Run now · Edit · Pause · Delete`.

**Deferred-run integration** — the page also surfaces deferred
runs from [hero-chat-and-model](../hero-chat-and-model/spec.md)'s
deferred-run queue. A row representing a deferred run gets an
amber `deferred` chip + an inline `Run now` action; when
`deferred.fired` arrives via SSE, the chip flips to `running` and
the row updates in place; on `session.done`, the chip flips to
`done`. Past missed runs (under `miss_policy: skip`) appear with a
grey `missed` chip.

### `/agents/scheduled/<id>`

Reuses sub-nav. Page-hero title `<scheduled name>` + run schedule
sub. Body: a paginated `scheduled_runs` table (one row per fire,
columns: fired-at, status, cost, exit-code, click-through to the
originating session in `/agents/session/<id>`). Below the table,
an edit form for cron expression, action command, mode, budget,
and `miss_policy`.

### `/agents/automations`

Sub-nav `Automations` active. Page-hero title `Automations` with
subhead `<n> rules · <m> fired today`. Inline action: `+ New automation`
(navigates to `/agents/automations/new`).

Body is a table of rules: name, trigger type
(`tracker` / `webhook` / `schedule` / `file` / `feed`), action
command, approval required toggle, enabled toggle, last fired
(relative), recent activity sparkline (inline SVG, 30d). Selecting
a row exposes the verb strip: `Run dry · Edit · Enable/Disable ·
Open execution log · Delete`.

### `/agents/automations/<id>`

Per-rule page. Reuses sub-nav. Page-hero shows rule name + summary
of trigger + action. Body has two stacked blocks:

1. **Execution log** — paginated `automation_runs` rows; each row
   click-throughs to the produced session.
2. **Dry-run form** — text area for a sample event payload + a
   `Dry run` button that POSTs to `/api/agents/automations/<id>/dry-run`
   and renders the evaluated filter + the action that would
   execute (without enqueueing a job).

### `/agents/automations/new` (builder island)

The `automation-builder.js` island. Three stacked vertical
sections inside a single form:

1. **Trigger** — picker (`tracker` / `webhook` / `schedule` /
   `file` / `feed`). Each choice reveals type-specific fields
   (tracker: event + filter expression builder; webhook: path +
   filter; schedule: cron expression + timezone; file: glob;
   feed: event type).
2. **Action** — Hero subcommand picker (populated from the
   command registry exposed by `hero-chat-and-model`), args
   input with `{{template variable}}` autocomplete from the
   trigger's payload shape, model picker (proxied through the
   adapter abstraction), budget input, mode (`autopilot` /
   `gated`).
3. **Approval** — required toggle; if on, reviewer picker and
   notification target. Under `local` edition, the reviewer
   picker is just the local user.

Right-side **Preview pane** renders the generated YAML matching
the absorbed [hero-automations](../hero-automations/spec.md)
format. On save, the island POSTs to `/api/agents/automations` and
the daemon writes both `.hero/automations/<slug>.yaml` (canonical,
version-controlled) and an `automations` SQLite row (query cache).

### `/agents/health`

Sub-nav `Health` active. Page-hero title `Agent health` with
subhead `<window>` and a time-range chip row (`24h` / `7d` /
`30d` / `all`). Body has three blocks:

1. **Leaderboard** — table of agents with columns: runs,
   win rate, interrupt rate, average cost per merged proposal,
   p95 turn count, recent failure count. Sortable per column.
2. **Cost over time** — inline SVG line chart, total agent cost
   per day, stacked by agent (server-rendered; no chart library).
3. **Recent failures** — list of the last 20 failed sessions with
   agent, spec, failure reason, click-through to the session.

Data sourced from `agent-contribution-tracking` events + `jobs.go`
records + `proposals.go` outcomes; aggregated server-side and
cached for 60s.

### `/agents/credentials`

Sub-nav `Credentials` active **only** under `team` / `cloud` /
`enterprise`; under `local` the sub-nav tab is faded with the
meta `· team server only` and the link is a no-op (or 404 on
direct hit).

Under `team`+, the page is a read-only window into hero-code's
credential broker (per the chat-and-model spec, credentials live
on the adapter, never on hero serve). Two halves:

1. **Shared keys** — list of broker-held provider keys (Anthropic,
   OpenAI, Azure, tracker tokens). Status (`active` / `rotated` /
   `missing`), last-used. **Server-side authority for what to
   render — the page only ever sees `sk-…last4`, never raw key
   material.**
2. **Per-user usage** — table of team members with today's /
   week's / month's spend, daily budget, percentage of budget,
   status. Inline budget editor (admin role only) posts to
   hero-code via `/api/agents/credentials/budget`.
3. **Budget alerts** — list of recent `budget_warning_80` /
   `budget_exceeded` events from the broker.

### Data sources

This home composes existing plumbing plus one new package and two
new SQLite tables. No new data domain ownership.

| Surface | Backing |
|---|---|
| Sessions list, session detail | `internal/serve/jobs.go` (queue + running) + `internal/serve/workers.go` (per-token streaming) + `internal/serve/team_coordination.go` (session registration) |
| Live transcript | Per-session SSE ring buffer in `jobs.go`, fanned out via new `session.*` event types |
| Proposals queue, proposal detail | `internal/serve/proposals.go` (pending store + existing accept/edit/reject endpoints) |
| Scheduled list, scheduled detail, deferred-run rows | `internal/serve/scheduled.go` (cron tick loop, already shipped) + new `scheduled_runs` table + deferred-run queue from [hero-chat-and-model](../hero-chat-and-model/spec.md) |
| Automations list, automation detail, automation builder | New `internal/automations/` package per the absorbed spec; two new SQLite tables (`automations`, `automation_runs`); on-disk `.hero/automations/*.yaml` is canonical |
| Health stats | `agent-contribution-tracking` events + `jobs.go` records + `proposals.go` outcomes; cached aggregation |
| Credentials | Read-only proxy of hero-code's credential broker per [hero-chat-and-model](../hero-chat-and-model/spec.md) |

### SSE event types

This page subscribes to a topic-multiplexed SSE feed at
`/api/agents/events`. New event types this home needs the daemon to
emit (extending `internal/serve/events.go`):

| Event | Payload |
|---|---|
| `session.started` | `{session_id, agent, spec, user, model, branch, started_at}` |
| `session.token` | `{session_id, turn, delta_text}` — per-session subscription only, not on the global stream |
| `session.tool_call` | `{session_id, turn, tool_name, args}` |
| `session.tool_result` | `{session_id, turn, tool_call_id, status, summary}` |
| `session.cost` | `{session_id, cost_total, tokens_in, tokens_out}` — debounced to 2s |
| `session.status_changed` | `{session_id, status}` |
| `session.done` | `{session_id, status, cost_total, turns}` |
| `proposal.created` | `{proposal_id, batch_id, session_id, spec, files, adds, dels}` |
| `proposal.resolved` | `{proposal_id, batch_id, outcome}` |
| `scheduled.fired` | `{scheduled_id, run_id, session_id}` |
| `scheduled.done` | `{run_id, status, cost}` |
| `deferred.queued` | `{run_id, scheduled_id, reason}` — emitted by the deferred-run queue in hero-chat-and-model when no headless adapter is reachable |
| `deferred.fired` | `{run_id, session_id}` — emitted when a deferred run dequeues and dispatches |
| `automation.fired` | `{automation_id, run_id, trigger_payload, session_id?}` |
| `automation.done` | `{run_id, status, session_id?}` |
| `credentials.budget_alert` | `{user, threshold, current}` — `team`+ only |

The high-frequency `session.token` events are fanned out **only** to
clients subscribed to a specific session; they do not appear on the
global agents stream. `session.cost` is debounced to 2s. The
deferred-run events are emitted by hero-chat-and-model's queue — this
home subscribes to them.

### Edition gating

The shell's view-registry edition filter (per
[hero-surface-shell §View registry](../hero-surface-shell/spec.md))
hides the `Credentials` sub-nav tab and the `/agents/credentials`
route entirely under `HERO_EDITION=local` at the data layer (404
on direct hit) and renders the tab as faded in the sub-nav partial
based on the same edition value. Everything else is visible at every
edition; data scoping differs (e.g. team-mate sessions appear under
`team`+ in the Sessions list).

## Changes

### Delivered (Sprint 1 — Sessions surface + live ledger hook)

The first delivery lands the `/agents` Sessions view on top of the
shell, exports the canonical `SessionRow` snapshot type that the Now
home will consume read-only in a follow-up, and wires page-level SSE
+ four section fragment endpoints. Sub-routes (`/agents/session/<id>`,
`/agents/proposals/<id>`, `/agents/scheduled`, `/agents/automations`,
`/agents/health`, `/agents/credentials`), the islands (`agent-session.js`,
`diff-viewer.js`, `automation-builder.js`), the automations engine,
and the per-session SSE topic are left to subsequent sprints per the
Kickoff build order.

- `internal/serve/pages/agentspage/page.go` — Agents home registration
  on the shell router; composes page-hero, sub-nav (six tabs, locked
  Credentials under local), tabbed metric strip, and the four section
  partials; exposes `SectionFragment(deps, section)` for SSE fragment
  swap
- `internal/serve/pages/agentspage/styles.go` — Agents-home stylesheet
  (light-grey transcript panel `#f7f8fa`, separator-line session
  blocks, amber proposal-preview, sub-nav tints, approval/timeline/
  compact rows) and inline filter-chip + SSE refresh script
- `internal/serve/pages/agentspage/data/types.go` — exports
  `SessionRow` (canonical live-session snapshot for cross-home
  consumption), `Sessions`/`SessionBlock`/`ApprovalRow`/`CompletedRow`/
  `CompactRow`/`MetricStrip` payload shapes
- `internal/serve/pages/agentspage/data/sessions.go` — composes the
  sessions-view payload from injected `LiveSessions` snapshot,
  builds prominent/compact/amber blocks, assembles the three-tab
  metric strip
- `internal/serve/pages/agentspage/data/proposals.go` — flat
  approval-row builder driven by injected `Proposals` snapshot
- `internal/serve/pages/agentspage/data/scheduled.go` /
  `automations.go` — table + preview-row builders, nil-safe
- `internal/serve/pages/agentspage/data/health.go` — placeholder
  empty-shape Health payload
- `internal/serve/pages/agentspage/templates/{page,sessions,session-block,approvals,completed,scheduled-preview}.html`
  — all four section partials + session-block partial with
  variant-driven shape (`prominent`/`compact`/`amber`)
- `internal/serve/pages/agentspage/{import,page}_test.go` — forbids
  chat / runner imports; asserts each section id + sub-nav render
- `internal/serve/api/agents.go` — `/api/agents/events` SSE channel
  + `/api/agents/{sessions,approvals,completed,scheduled-preview}`
  fragment endpoints, debounced per-section
- `internal/serve/shell/stubs.go` — drop the `agents` stub entry
  (real home takes the `/agents` slot)
- `internal/serve/shell/shell_test.go` — adjust the stub-render test
  to reflect the now-empty stub list
- `internal/serve/server.go` — mount the Agents handler before the
  `/api/` catch-all; register the real Agents home on the shell
  router; expose `snapshotLiveSessions()` reading the team-mode
  job-queue session ledger and `snapshotAgentsProposals()` reading
  the propose store

### Templates and handlers

1. **`internal/serve/pages/agents/page.go`** — new package; registers
   handlers for all eleven routes above and wires them to the
   data fetchers. Each handler renders a Go template with a
   data struct it populates from the fetchers.
2. **`internal/serve/pages/agents/templates/sub-nav.html`** — shared
   sub-nav partial (six tabs + faded-credentials variant under
   `local`). Takes `Active` parameter.
3. **`internal/serve/pages/agents/templates/sessions.html`** —
   `/agents` page template: page-hero, metric strip, live-sessions
   list, awaiting-approval list, completed-today timeline,
   scheduled/automations preview split.
4. **`internal/serve/pages/agents/templates/session-block.html`** —
   shared partial for one session in the live list. Takes a
   `Variant` parameter (`prominent` | `compact` | `amber`).
   Renders the **light-grey** transcript panel (`#f7f8fa`) when not
   the amber variant; renders the proposal-preview block when
   amber.
5. **`internal/serve/pages/agents/templates/session-detail.html`** —
   `/agents/session/<id>` template; hosts the `agent-session.js`
   island.
6. **`internal/serve/pages/agents/templates/proposals.html`** —
   `/agents/proposals` flat list.
7. **`internal/serve/pages/agents/templates/proposal-detail.html`** —
   `/agents/proposals/<id>`; hosts the `diff-viewer.js` island.
8. **`internal/serve/pages/agents/templates/scheduled.html`** —
   `/agents/scheduled` table with deferred-run rows interleaved.
9. **`internal/serve/pages/agents/templates/scheduled-detail.html`** —
   `/agents/scheduled/<id>` run history + edit form.
10. **`internal/serve/pages/agents/templates/automations.html`** —
    `/agents/automations` rules list.
11. **`internal/serve/pages/agents/templates/automation-detail.html`** —
    `/agents/automations/<id>` execution log + dry-run.
12. **`internal/serve/pages/agents/templates/automation-builder.html`** —
    `/agents/automations/new` host for the builder island.
13. **`internal/serve/pages/agents/templates/health.html`** —
    `/agents/health` leaderboard + cost chart (inline SVG) +
    recent failures.
14. **`internal/serve/pages/agents/templates/credentials.html`** —
    `/agents/credentials`; renders the broker view under `team`+
    only; under `local` returns the page-hero with a short
    "available on the team server" notice and a CTA, or 404.
15. **`internal/serve/pages/agents/static/agents.css`** — CSS lifted
    from the mock, including:
    - `.transcript { background: #f7f8fa; color: var(--ink-2); }`
      (light-grey panel — **not** a dark terminal)
    - `.proposal-preview { background: #fffaf2; border: 1px solid var(--warn-border); }`
    - `.session-block { border-top: 1px solid var(--border); }`
      (separator-line idiom — not cards)
    - Metric tab + filter chip styles per mock
    - Light-grey transcript cursor: `.transcript .streaming::after`
      with hero-blue blink animation
16. **`internal/serve/pages/agents/static/metric-tabs.js`** — inline
    script (also packaged as a small `.js`) that toggles
    `.metric-tab.active` / `.metric-pane.active` and `.filter-chip.active`
    on click.

### Data fetchers

17. **`internal/serve/pages/agents/data/sessions.go`** — reads from
    `internal/serve/jobs.go` + `workers.go`; returns the page's
    live-sessions / awaiting-approval / completed-today / metric-strip
    data. Edition-aware (own-sessions-only under `local`; team-wide
    under `team`+).
18. **`internal/serve/pages/agents/data/proposals.go`** — reads from
    `internal/serve/proposals.go`'s pending store; groups by
    `batch_id`.
19. **`internal/serve/pages/agents/data/scheduled.go`** — reads from
    `internal/serve/scheduled.go` for definitions, the new
    `scheduled_runs` table for run history, and the deferred-run
    queue from `hero-chat-and-model` for in-flight deferred rows.
20. **`internal/serve/pages/agents/data/automations.go`** — reads from
    the new `internal/automations/` engine + the `automations` /
    `automation_runs` SQLite tables.
21. **`internal/serve/pages/agents/data/health.go`** — aggregation
    queries; 60s memo.

### SSE / API

22. **`internal/serve/api/agents.go`** — new package; REST + SSE:
    - `GET /api/agents/sessions` — filterable session list (JSON)
    - `GET /api/agents/sessions/<id>` — session metadata + transcript
    - `POST /api/agents/sessions/<id>/interrupt`
    - `POST /api/agents/sessions/<id>/approve-all`
    - `POST /api/agents/sessions/<id>/reject-all`
    - `GET /api/agents/sessions/<id>/events` — per-session SSE (replays from `cursor`)
    - `GET /api/agents/events` — page-level multiplexed SSE
      (`proposal.*`, `scheduled.*`, `deferred.*`, `automation.*`,
      `session.started`, `session.status_changed`, `session.done`,
      `session.cost`, `credentials.budget_alert`)
    - `POST /api/agents/proposals/<id>/approve`
    - `POST /api/agents/proposals/<id>/reject`
    - `POST /api/agents/proposals/<id>/edit-approve`
    - `GET /api/agents/scheduled` / `POST` / `PUT/<id>` / `DELETE/<id>`
    - `POST /api/agents/scheduled/<id>/run-now`
    - `GET /api/agents/automations` / `POST` / `PUT/<id>` / `DELETE/<id>`
    - `POST /api/agents/automations/<id>/dry-run`
    - `POST /api/agents/automations/<id>/toggle`
    - `GET /api/agents/health?range=…`
    - `GET /api/agents/credentials` (team+ only; 404 under local)
23. **`internal/serve/jobs.go`** (extend) — emit new `session.*`
    events on state transitions; maintain per-session ring buffer
    of last N events for cursor replay.
24. **`internal/serve/workers.go`** (extend) — fan out
    `session.token` events on streaming deltas to per-session
    subscribers; debounce `session.cost` to 2s.
25. **`internal/serve/scheduled.go`** (extend) — expose CRUD
    endpoints (wired through `api/agents.go`); insert
    `scheduled_runs` row on each fire; emit `scheduled.fired` and
    `scheduled.done`. When no headless adapter reachable, hand off
    to the deferred-run queue from `hero-chat-and-model` (the
    queue lives there per spec boundary; `scheduled.go` calls
    into it).
26. **`internal/serve/events.go`** (extend) — register the new
    event types and the per-session subscription topic.
27. **`internal/serve/scheduled_runs.go`** (new) — `scheduled_runs`
    SQLite table (columns: `scheduled_id`, `fired_at`, `status`,
    `exit_code`, `cost`, `session_id`, `log_path`) + insert/update/list
    queries.

### Islands

28. **`internal/serve/islands/agent-session.js`** — vanilla web
    component, no bundler. Subscribes to
    `/api/agents/sessions/<id>/events`, renders streaming transcript
    into a host fragment that uses the same light-grey transcript
    CSS as the preview, handles autoscroll, renders the pending-approval
    banner when `session.status_changed` carries `awaiting_approval`,
    and wires the inline approve/reject actions.
29. **`internal/serve/islands/diff-viewer.js`** — multi-proposal
    diff viewer; calls existing proposal endpoints; renders
    anchor-aware proposal visualization per
    [inline-propose-output-mode](../inline-propose-output-mode/spec.md).
30. **`internal/serve/islands/automation-builder.js`** — trigger /
    action / approval form with live YAML preview pane; POSTs to
    `/api/agents/automations` on save.

### Automations engine (absorbs hero-automations)

31. **`internal/automations/engine.go`** (new) — rule loader,
    trigger registry, filter evaluator, action dispatcher (calls
    into hero-chat-and-model's adapter abstraction; never invokes
    a model directly).
32. **`internal/automations/triggers.go`** (new) — tracker poller,
    webhook handler, cron scheduler (shares the `scheduled.go`
    tick loop), file watcher, feed listener.
33. **`internal/automations/approval.go`** (new) — approval gate
    + reviewer notification.
34. **`internal/automations/types.go`** (new) — `AutomationConfig`,
    `TriggerConfig`, `ActionConfig`, `ApprovalConfig`.
35. **`internal/automations/store.go`** (new) — `automations` and
    `automation_runs` SQLite tables + sync with on-disk
    `.hero/automations/*.yaml`. Disk is canonical (for version
    control); SQLite is a query cache.
36. **`internal/cli/automations.go`** (new) — preserves the
    absorbed spec's CLI surface (`hero automations list / test /
    enable / disable / log`).

### View registry registration

37. **`internal/serve/packs/engineering/registry.go`** (extend) —
    register all eleven routes as view records with the existing
    `agents-*` slugs (default view `agents-sessions`), edition
    gating per the matrix above, and renderer kind per the table.

## Boundaries

- **Not in this spec:** chat dispatch, adapter selection, deferred-run
  queue logic, `miss_policy` semantics, slash command registry,
  ⌘K command bar. Owned by
  [hero-chat-and-model](../hero-chat-and-model/spec.md). Agents
  home consumes the deferred-run queue read-only and renders its
  rows on `/agents/scheduled`.
- **Not in this spec:** workflow implementations (`/design`,
  `/diagnose`, `/deliver`, …). Those run inside the adapter
  (hero-code by default).
- **Not in this spec:** the runner internals (provider abstraction,
  agent loop, sandbox). hero-code is a sibling repo.
- **Not in this spec:** the credential broker mechanism itself
  (per-user budgets, key storage, OAuth, rotation). hero-code owns
  the broker per [hero-chat-and-model](../hero-chat-and-model/spec.md);
  this home renders a read-only window into broker state.
- **Not in this spec:** the inline-propose wire contract (envelope,
  endpoints, `proposal.*` event types). Owned by
  [inline-propose-output-mode](../inline-propose-output-mode/spec.md);
  this home consumes them.
- **Not in this spec:** people presence, activity feed, handoff
  stream. Owned by hero-people-and-roi-home.
- **Not in this spec:** the shell chrome (top nav, search pill,
  avatar, footer, role-pack routing). Owned by
  [hero-surface-shell](../hero-surface-shell/spec.md).
- **Not in this spec:** three-way merge conflict resolution in the
  diff viewer (deferred to v2).
- **Not in this spec:** cross-org proposal review (cloud
  edition; deferred).

## Acceptance Criteria

- WHEN the user opens `/agents` THE SYSTEM SHALL render the Sessions
  view as the default with the sub-nav showing `Sessions` active.
- WHEN the Sessions view renders THE SYSTEM SHALL place each live
  session in a separator-line block (not a card) with a light-grey
  transcript preview panel.
- THE SYSTEM SHALL render the transcript preview panel on a light-grey
  background (`#f7f8fa`) with dark text, not a dark terminal.
- WHEN a session is in flight THE SYSTEM SHALL stream `session.token`
  events into the per-session SSE topic within 100ms of the runner
  emitting the delta.
- WHILE `/agents/session/<id>` is open THE SYSTEM SHALL append
  streamed tokens to the transcript without a page reload.
- WHEN the user clicks `Open transcript` on a Sessions-view block
  THE SYSTEM SHALL navigate to `/agents/session/<id>` and the
  `agent-session.js` island SHALL mount and subscribe to the
  per-session SSE stream.
- WHEN the user clicks `Interrupt` on a running session THE SYSTEM
  SHALL POST to `/api/agents/sessions/<id>/interrupt` and the
  runner SHALL halt the agent loop at the next turn boundary.
- WHEN the user clicks a metric-strip tab (`Right now` / `Today` /
  `Health (7d)`) THE SYSTEM SHALL toggle the active pane in place
  without a page reload.
- WHEN the user clicks `Approve` on a pending proposal in the
  awaiting-approval list THE SYSTEM SHALL POST to
  `/api/agents/proposals/<id>/approve` and remove the row from the
  list via SSE.
- WHEN the user clicks `Reject` on a pending proposal THE SYSTEM
  SHALL POST to `/api/agents/proposals/<id>/reject` and emit a
  `proposal.resolved` event with outcome `rejected`.
- WHEN a scheduled run fires AND no headless-capable adapter is
  reachable THE SYSTEM SHALL render the run in the
  `/agents/scheduled` view with a `deferred` chip via the
  `deferred.queued` event.
- WHEN a deferred run later fires THE SYSTEM SHALL update its row's
  chip to `done` (or `failed`) via the `deferred.fired` then
  `session.done` events without a page reload.
- WHEN the user saves a new automation in
  `/agents/automations/new` THE SYSTEM SHALL write
  `.hero/automations/<slug>.yaml` AND insert a row in the
  `automations` SQLite table.
- WHEN the user clicks `Dry run` on `/agents/automations/<id>`
  with a sample event payload THE SYSTEM SHALL evaluate the filter
  and render the action that would execute without enqueueing a
  job.
- WHEN an automation rule's trigger matches an incoming event THE
  SYSTEM SHALL evaluate the filter, dispatch the action through
  the adapter abstraction, insert an `automation_runs` row, and
  emit `automation.fired`.
- WHILE the daily spend exceeds the configured budget THE SYSTEM
  SHALL render the `spend today` metric tile's progress bar in
  amber when spend exceeds 80% and in red when spend exceeds
  100% of budget.
- WHERE the active edition is `local` THE SYSTEM SHALL render the
  `Credentials` sub-nav tab as faded with the meta `· team server
  only` AND SHALL return 404 from `GET /api/agents/credentials`
  AND `GET /agents/credentials`.
- WHERE the active edition is `team`, `cloud`, or `enterprise` THE
  SYSTEM SHALL render `/agents/credentials` with the broker's
  shared keys (last4 only) and per-user budgets.
- WHEN a session SSE client reconnects with a `cursor` query
  parameter THE SYSTEM SHALL re-play events from the cursor
  forward from the per-session ring buffer.
- IF the `.hero/automations/<slug>.yaml` file is edited externally
  THEN THE SYSTEM SHALL re-load the rule on the next file-watcher
  tick and update the `automations` table row.
- THE SYSTEM SHALL persist scheduled-run history, automation-run
  history, and session transcripts across daemon restarts.
- THE SYSTEM SHALL NOT render raw provider key material on
  `/agents/credentials`; the server SHALL pass only `sk-…last4`
  to the template.

## Risks

- **SSE volume on a busy day.** Twelve sessions streaming tokens
  in parallel plus debounced cost ticks plus proposal / scheduled /
  automation events can saturate the event bus. Mitigation: the
  high-frequency `session.token` event is per-session-subscription
  only, never on the global stream; `session.cost` is debounced
  to 2s; the page-level `/api/agents/events` stream excludes
  `session.token` entirely.
- **Transcript preview length.** Long agent runs produce a lot of
  transcript text. The preview panel is capped at the last 12
  lines server-side; the panel is `overflow: hidden` with a fixed
  `min-height`. The full transcript is only loaded on
  `/agents/session/<id>` and the island virtualizes older turns
  beyond a threshold.
- **Deferred-run notification loop.** A flapping adapter could
  cause rapid `deferred.queued` / `deferred.fired` cycles. The
  deferred-run queue lives in `hero-chat-and-model` and is
  expected to debounce; the Agents UI must tolerate rapid SSE
  updates without flicker (use CSS transitions, not full
  re-renders).
- **Automation builder complexity.** Trigger / action / approval
  with a YAML preview and template-variable autocomplete is the
  most complex island in this spec. Risk of scope creep. Mitigation:
  v1 ships the five trigger types and a basic AND/OR filter
  builder; advanced filter expressions and reusable templates
  are deferred.
- **Interrupt semantics.** Halt must occur at a turn boundary, not
  mid-tool-call, to avoid leaving the workspace in a partially
  edited state. The runner already loops turn-by-turn; the
  interrupt flag is checked at the top of each turn. This home's
  UI must convey that "interrupt" may not be immediate.
- **Cross-pack click-through.** Clicking `Open spec` on a session
  block must route into the Work home via the shell's canonical
  URL router, not `window.location.href`, so the Agents tab is
  preserved for back-navigation. Per-item tab persistence is
  shell-owned.
- **Credential broker leak.** A bug where the API returns more
  than `last4` to the page would be a silent privacy regression.
  Server-side authority: the `/api/agents/credentials` handler
  redacts before serializing; integration tests assert the
  serialized JSON never contains a full key string.
- **Edition-data scoping.** The Sessions list under `local` must
  filter to own-user sessions; under `team`+ it widens. The data
  fetcher (`data/sessions.go`) is the single enforcement point;
  integration tests must cover both editions.
- **Schedule clock skew.** "Next run in 14h 22m" uses server
  time; the UI also shows the absolute time (`Mon 9:00am`) so a
  user on a different timezone has both anchors.

## Validation

- **Unit tests.**
  - SSE event emission for each new event type
    (`session.token`, `session.tool_call`, `session.tool_result`,
    `session.cost`, `session.done`, `scheduled.fired`,
    `scheduled.done`, `deferred.queued`, `deferred.fired`,
    `automation.fired`, `automation.done`,
    `credentials.budget_alert`).
  - View-registry filter under each edition; assert
    `agents-credentials` is present/absent as expected.
  - Automation filter evaluator (per the absorbed spec).
  - `scheduled_runs` insert and update.
  - Health aggregation queries against a fixture dataset.
  - Credentials redaction: serialized response never contains a
    raw key string.
- **Integration tests.**
  - Start `hero serve` with a mock adapter; enqueue an agent job;
    assert `session.started`, several `session.token`, then
    `session.done` arrive in order on the per-session SSE stream.
  - Emit an inline-propose batch; assert
    `/agents/proposals` re-renders the new row via SSE; click
    `Approve` and assert the row disappears.
  - Create an automation rule via `POST /api/agents/automations`,
    fire a matching event, assert a session is dispatched and
    the `automation_runs` row records the link.
  - Create a scheduled rule; stop the mock adapter; advance the
    clock past the next run; assert the row appears with
    `deferred` chip and `deferred.queued` event fires;
    reconnect adapter; assert `deferred.fired` and the chip
    flips to `done`.
  - Under each `HERO_EDITION`, `GET /api/agents/credentials`;
    assert 404 under `local` and 200 with no raw key material
    under `team`+.
- **Manual.**
  - Open `/agents` against a real `hero serve` with hero-code
    connected; visually compare against
    [mockups/01-agents-sessions.html](mockups/01-agents-sessions.html):
    sub-nav layout, page-hero copy, metric strip tabs, three
    session blocks (prominent / compact / amber), light-grey
    transcript panel, awaiting-approval flat list, completed
    timeline, scheduled/automations split preview.
  - Open `/agents/session/<id>` against a live run; confirm
    token-by-token streaming feels live (no perceptible chunking)
    on the **light-grey** transcript panel — not a dark terminal.
  - Approve and reject proposals from the diff viewer; confirm
    the spec file on disk matches expected after-state.
  - Build a new automation via the builder, dry-run with a sample
    payload, save, fire the real trigger, confirm a session row
    appears in `/agents` Live sessions.
  - Confirm the `Credentials` sub-nav tab is faded under `local`
    and clicking it does not navigate.
- **Visual.**
  - Diff the rendered `/agents` against the mock at 1280px and
    900px viewports; verify spacing, colors, typography, sub-nav
    underline, metric tile layout, transcript panel background
    (`#f7f8fa`), and proposal-preview background (`#fffaf2`)
    match.
  - Confirm the transcript cursor is hero-blue (`var(--hero-blue-700)`)
    and blinks at ~0.95s.
- **Performance.**
  - With 12 concurrent sessions emitting tokens, daemon CPU stays
    below 30% on a developer laptop (per the risk mitigation:
    per-session subscription only).
  - The `/agents` page first paint completes in under 300ms with
    50 historical sessions in `completed-today`.
  - The `/agents/health` aggregation completes in under 500ms with
    10,000 historical sessions (cached for 60s).

## Kickoff

```
/deliver hero-agents-home
```

Prerequisites:

- [hero-surface-shell](../hero-surface-shell/spec.md) has at least
  the top nav, the page routing skeleton, and edition-gated view
  registration landed. If not, stub the shell chrome in the
  Agents page template and deliver standalone.
- [hero-chat-and-model](../hero-chat-and-model/spec.md) has at
  least the adapter abstraction and the deferred-run queue
  landed; otherwise stub the deferred-run rows on
  `/agents/scheduled` and defer that acceptance criterion.
- [inline-propose-output-mode](../planning/features/inline-propose-output-mode/spec.md)
  is shipped (it is) and its accept/edit/reject endpoints are
  consumed verbatim.

Build order:

1. Page handlers + templates for `/agents` (sessions view) +
   sub-nav partial + `session-block.html` partial (all three
   variants). Static data first to validate the visual.
2. SSE event types + per-session subscription topic +
   `data/sessions.go` fetcher; wire the metric strip and live
   blocks to real data.
3. `/agents/session/<id>` + `agent-session.js` island (light-grey
   transcript panel + token streaming + pending-approval banner).
4. `/agents/proposals` + `/agents/proposals/<id>` +
   `diff-viewer.js`.
5. `/agents/scheduled` + `/agents/scheduled/<id>` +
   `scheduled_runs` table + deferred-run row integration.
6. `internal/automations/` engine + `/agents/automations` +
   `/agents/automations/<id>` + dry-run.
7. `/agents/automations/new` + `automation-builder.js`.
8. `/agents/health` + cost-over-time inline SVG.
9. `/agents/credentials` (team+ only; faded sub-nav under local).
10. CLI surface preserved from absorbed spec (`hero automations …`).

**Files:**
.hero/planning/features/hero-agents-home/spec.md,
.hero/planning/features/hero-automations/spec.md (mark
superseded-partial on delivery),
internal/serve/pages/agents/page.go (new),
internal/serve/pages/agents/templates/{sub-nav,sessions,session-block,session-detail,proposals,proposal-detail,scheduled,scheduled-detail,automations,automation-detail,automation-builder,health,credentials}.html (new),
internal/serve/pages/agents/static/agents.css (new),
internal/serve/pages/agents/static/metric-tabs.js (new),
internal/serve/pages/agents/data/{sessions,proposals,scheduled,automations,health}.go (new),
internal/serve/api/agents.go (new),
internal/serve/islands/{agent-session,diff-viewer,automation-builder}.js (new),
internal/serve/jobs.go (extend — session.* SSE + ring buffer),
internal/serve/workers.go (extend — token streaming + cost debounce),
internal/serve/scheduled.go (extend — CRUD + deferred-queue handoff),
internal/serve/events.go (extend — new event types + per-session topic),
internal/serve/scheduled_runs.go (new),
internal/automations/{engine,triggers,approval,types,store}.go (new package; absorbs hero-automations),
internal/cli/automations.go (new — CLI from absorbed spec),
internal/serve/packs/engineering/registry.go (extend — register agent views).

**Skip:** the shell chrome (owned by hero-surface-shell); chat
dispatch / adapter selection / deferred-run queue logic (owned by
hero-chat-and-model — this home consumes the queue); the inline-propose
wire contract (already shipped); hero-code internals; three-way
merge conflict resolution in the diff viewer (deferred to v2);
cross-org proposal review (deferred); raw provider key handling
on the credentials page (server redacts to last4 only).
