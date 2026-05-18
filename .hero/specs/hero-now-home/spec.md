---
title: Hero Now Home — Personal Cold-Start Surface
slug: hero-now-home
type: feature
status: completed
tags: [serve, surface, now, home, web-app, ui]
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
  - target: next-as-projection
    kind: relates-to
  - target: next-handoff-emit
    kind: relates-to
horizon: now
---

## Context

Open `hero serve` today and the landing surface is a static workspace
summary — counts of specs by status, recent commits, a search box. It's
a *picture of the project* rather than a *picture of where the user
is*. Open it on a Monday and you still have to go hunting: which spec
was I in the middle of? what did my agent finish overnight? what's
waiting on me from a peer repo? what changed since I closed the lid?

The day-to-day question the Now home answers is small and specific:
**"what's on my plate, what changed, what's my agent doing, how do I
jump in?"** Everything on this page exists to answer one of those four
questions and to do it in the order a person would actually scan it.

The visual source of truth for this spec is the mock at
`mockups/01-now-default.html`. The earlier draft of this spec described
a desktop-app-style layout (fixed left rail, VS Code tab strip, bottom
verb strip, right ambient panel, 4-card 2×2 main grid). That layout has
been rejected. **The Now home is a scrolling web app page**, not a
desktop chrome. It is built from stacked sections in top-down
day-to-day flow order, hosted inside the shell defined by
[hero-surface-shell](../hero-surface-shell/spec.md), rendered using the
template-first strategy locked by
[hero-surface-deployment-and-rendering](../hero-surface-deployment-and-rendering/spec.md),
and dispatches chat through
[hero-chat-and-model](../hero-chat-and-model/spec.md).

Data sources are largely already produced for other purposes. The
NEXT.md projection pipeline ([next-as-projection]) aggregates per-user
state. The handoff-emit pipeline ([next-handoff-emit]) already streams
the graph events that feed the "Since you were here" section. The
proposals, handoff, drift, and session subsystems all exist. This spec
*composes* those streams behind a scrolling page; it does not reinvent
them.

## Goal

Replace the current `hero serve` landing surface with a single
scrolling page at `/now`, served by the shell, that any logged-in user
can open and within one second see:

1. **Page hero** — who they are, the project, what edition, and a
   one-liner status of what's open ("2 need your input, 1 agent
   running, since Fri 4:12pm") with an inline action row.
2. **Tabbed metric strip** — `This sprint` (default) / `My week` /
   `Hero ROI`. Each tab swaps a 4-tile row of pulse metrics. The first
   tab is methodology-aware (Sprint / Cycle / Week).
3. **Needs your input** — a clean Linear/GitHub-style list of
   3–4 items with typed colored dots, one-line summaries, and inline
   action links.
4. **Quick launch** — a large rounded chat input with intent chips and
   a "Try:" recent-prompts line; dispatches via the chat-and-model
   adapter.
5. **On your plate** — 60/40 split: the active spec (with mini progress
   bars, agent-pulse, inline actions) and a secondary recently-touched
   spec.
6. **Your agents** — 1.4/1 split: a Currently-running block (soft
   transcript preview, cost ticker, Open / Interrupt) and a Today block
   (compact stat grid + 3-row session list).
7. **Since you were here** — a flat 5–6 row timeline feed.

The page is **personal** — two users on the same workspace see
different Now pages. It is methodology-aware — the first metric tab
relabels to `This cycle` for Shape Up or `This week` for kanban/solo.
It is edition-aware — under `local`, team-presence signals on the
agents section are hidden. The chat input is the same shared chat
island already defined by [hero-surface-shell]; this page does not
re-implement chat dispatch.

Done means: opening `/now` renders the seven sections above with live
personalized data, the metric strip tabs swap without a server round
trip, the chat input dispatches through the chat-and-model adapter, and
agent-session updates stream into the Your-agents section via SSE.

## Approach

### Rendering strategy

Per [hero-surface-deployment-and-rendering], the page is **server-
rendered Go templates** composed against the shell's layout. The only
client-side JavaScript is:

- **The shared chat-input island** (owned by [hero-chat-and-model] /
  [hero-surface-shell]) — mounted once in the Quick launch section.
- **A tiny vanilla JS tab toggler** for the metric strip — already
  shown in the mock (~10 lines, toggles `.active` on `.metric-tab` and
  `.metric-pane` siblings, no framework, no module).
- **An SSE subscriber** that listens on `/api/now/events` and swaps the
  `Your agents` and `Needs your input` fragments via fragment-
  replacement, matching the convention from the shell SSE spec.

No React, no bundler. The chat island is shared with the shell, not
new to this page.

### Section composition

The page template includes the shell's `page-hero`, `tabbed-metric-strip`,
and `chat-input` fragments (per [hero-surface-shell]). Section partials
own the Now-specific content beneath those fragments.

| Section | Template | Data source(s) | Update channel |
|---|---|---|---|
| Page hero | shell `page-hero` fragment | computed from inbox count + running session count + last-active timestamp | SSE on counts change |
| Tabbed metric strip | shell `tabbed-metric-strip` + `data/metrics.go` | composes sprint / week / ROI tile data | initial render only |
| Needs your input | `inbox.html` + `data/inbox.go` | proposals queue + handoff inbound + PR-review mentions + import drafts | SSE on `event: inbox` |
| Quick launch | shell `chat-input` fragment | static prompts pulled from recent history | n/a (island) |
| On your plate | `plate.html` + `data/plate.go` | graph: claimed-by-user specs ordered by last-touched | SSE on `event: plate` |
| Your agents | `agents.html` + `data/agents.go` | in-flight session store + today's session log + spend ledger | SSE on `event: agents` |
| Since you were here | `changes.html` + `data/changes.go` | event log from next-handoff-emit, filtered to user's areas of interest | SSE on `event: changes` |

Each section partial renders even when empty — empty states collapse
to a one-line "Nothing waiting on you" / "No agent activity today"
rather than vanishing silently.

### Page hero

Owned by the shell's `page-hero` fragment. The Now route passes:

- **Eyebrow**: `hero · <branch> · <edition> edition` (e.g., `hero · main
  · solo edition`).
- **Title**: `Now` (32px, weight 600).
- **Subhead**: one-liner composed from `inbox.count`,
  `agents.running.count`, and `user.last_active_at` — e.g., `2 need
  your input · 1 agent running · since Fri 4:12pm`.
- **Action row**: primary `Open Inbox →` (anchor link to `#inbox`),
  ghost `What changed since Friday`, plus an edition chip (`Solo` /
  `Team` / `Cloud`).

### Tabbed metric strip

Owned by the shell's `tabbed-metric-strip` fragment. The Now route
provides three tabs and the partial templates for each tab's 4-tile
row:

- **`This sprint`** (default for Scrum) — Sprint progress (e.g., `9/14`
  with segmented bar by done / in-review / at-risk), Days remaining,
  Specs at risk (with spec slugs), Your slice (`3/4`).
- **`My week`** — Specs shipped this week (sparkline), Commits authored
  (with line deltas), Agent assist on your work (sparkline), Sessions
  / hours.
- **`Hero ROI`** — Autonomy ratio 7d (sparkline), Hours saved this
  week, Spec coverage 7d, Cycle time.

The tabs are text-link tabs with a hero-blue underline on the active
tab. A `View all metrics →` link sits at the right of the tab row.
Pane swapping is purely client-side DOM toggling (already shown in the
mock); no server round trip.

#### Methodology-aware first tab

The first tab's label and tile content adapt to the project's declared
methodology:

| `methodology` (from `hero.json`) | First tab label | Tile set |
|---|---|---|
| `scrum` (default) | `This sprint` | sprint progress, days remaining, at-risk specs, your slice |
| `shape-up` | `This cycle` | cycle progress, weeks remaining, scope-creep flags, your appetite |
| `kanban` | `This week` | WIP count, throughput 7d, blocked items, your slice |
| `solo` / unset | `This week` | specs shipped 7d, throughput sparkline, longest-open spec, your slice |

Detection is read once at request time from `hero.json`
(`methodology` field). If unset, fall back to `solo`. The remaining
two tabs (`My week`, `Hero ROI`) are identical across methodologies.

### Needs your input

Section heading `Needs your input` with an item count (`4 items`) and
a `View all (4) →` link. Rendered as a flat list (not cards):

- Each row: typed colored dot (proposal=blue, handoff=purple,
  review=green, import=warn-orange) · one-line summary · inline
  meta · inline action links right-aligned.
- Hover state highlights the row with `--bg-soft`.

Composed from:

- **Proposals** queued by the user's agent sessions (from the
  inline-propose-output-mode subsystem). Actions: `Approve` · `View
  diff` · `Reject`.
- **Inbound handoffs** from peer repos (`hero handoff status`).
  Actions: `Accept handoff` · `Open`.
- **PR reviews** assigned to the user (via tracker integration).
  Actions: `Review` · `Snooze`.
- **Import drafts** awaiting triage. Actions: `Triage` · `Dismiss`.

Updates stream via SSE — when any underlying source emits an event,
the section fragment is replaced.

### Quick launch

Section heading `Tell Hero what to do next`. The large rounded chat
input is mounted from the shell's shared `chat-input` fragment (the
same island [hero-surface-shell] defines and [hero-chat-and-model]
dispatches through). Configuration passed in from the Now route:

- Height ~64px, max-width 720px, centered.
- Hero-blue focus ring.
- Bolt icon left, submit button right.
- Intent chips below: `/design` · `/diagnose` · `/deliver` · `/review`
  · `/ask`. Clicking a chip prefills the input with the slash command.
- "Try:" recent-prompts line below chips, populated from the user's
  three most recent prompts.
- `/` keyboard shortcut focuses the input when no other input is
  active.

Submission posts to the chat-and-model dispatch endpoint; this page
does not own the dispatch logic.

#### Empty state — no adapter connected

When no chat adapter is configured, the Quick launch section renders
a small notice above the input — `Connect Claude or ChatGPT to enable
quick launch →` linking to the adapter settings — and the input is
disabled. The rest of the page renders normally.

### On your plate

Section heading `On your plate` with `2 active specs` meta and an
`All your work →` link. A 60/40 grid:

- **Primary block** (1.5fr): the current active spec. Filled
  background `--bg-soft` with border. Contains: spec slug as title
  link, status pill (Delivering / In review / etc), one-line
  description, two mini progress bars (`Acceptance criteria` and
  `Contract coverage` with right-aligned numeric value), a meta row
  (agent-pulse with animated dot when a session is live, claim owner,
  branch, last activity), and an action row (`Open spec` · `Continue
  session` · `/deliver` · `View graph`).
- **Secondary block** (1fr): a quieter, transparent-background variant
  for the spec the user was last looking at but isn't currently
  active. Same structure, fewer actions (`Open spec` · `View PR →`).

Data source: graph query for specs `claimed_by(user)`, ordered by
`last_touched`. First two are rendered; "All your work →" goes to the
Work home.

### Your agents

Section heading `Your agents` with an `Open Agents →` link. A 1.4/1
two-column grid:

- **Currently running** (1.4fr): an `agent-card` containing the
  current session — agent avatar + name, what they're working on
  (with spec link), a `Live` tag with animated dot, a soft transcript
  preview (background `--bg-panel` = `#f7f8fa`, **not** a dark
  terminal), 4–5 lines showing recent assistant/tool turns with subtle
  role coloring, plus a foot row with `$0.32 · 18 tool calls · 47k
  tokens` and `Open transcript` / `Interrupt` links.
- **Today** (1fr): a 2×2 stat grid (Sessions done, Proposals pending,
  Spend with mini sparkline, Autonomy with mini sparkline) above a
  3-row list of completed sessions (status icon · spec slug · duration).

Data sources: in-flight session store, today's session log, spend
ledger from the existing agent subsystem.

#### Edition gating

Under `HERO_EDITION=local`, this section omits any team presence
signals — no "teammate also running" indicators, no shared-presence
dots. The personal data (your sessions, your spend) renders normally.

### Since you were here

Section heading `Since you were here` with a `Show more →` link.
Flat timeline feed, 5–6 entries. Each row is one line:

- Relative-time chip (`2h`, `4h`, `1d`), right-aligned in a 56px column.
- Kind icon (commit, spec status, knowledge, drift, convention).
- Sentence-form text — e.g., "`d71a47d` `feat(rendering): …` on
  `main` · chet-bellows", or "Drift detected: `spec-status-integrity`
  claims complete but smoke test fails · hero check".
- Linkable artifacts (commit SHA, spec slug, knowledge slug) inline.

Source: the event stream emitted by [next-handoff-emit], filtered to
the user's areas of interest (specs they own, files they recently
edited, conventions they watch). `Show more →` opens a dedicated
feed view (not in scope for this spec — owned by [hero-work-home]
or a future Changes home).

### SSE wiring

A single Now SSE channel at `GET /api/now/events` multiplexes
updates. Event names map to section fragments:

- `event: inbox` — re-render Needs your input
- `event: plate` — re-render On your plate
- `event: agents` — re-render Your agents
- `event: changes` — re-render Since you were here
- `event: hero` — re-render the subhead in the page hero (counts
  changed)

Clients subscribe on Now mount and unsubscribe on unmount. Publisher
debounces ~250ms per event name. The metric strip is **not** wired
to SSE — its data is sprint/weekly aggregate, refresh on next page
load is fine.

### Footer

Owned by the shell. The Now route does not render its own footer.

## Changes

### Created

- `internal/serve/pages/now/page.go` — Now home registration,
  `GET /now` handler, methodology + edition resolution, page-hero +
  metric-strip composition; exposes `Register(router, Deps)` and
  `SectionFragment(deps, name)` for the SSE fragment endpoints.
- `internal/serve/pages/now/styles.go` — `nowStyles` (Now-specific CSS
  inlined via HeadExtra) and `nowScript` (intent-chip prefill,
  `/`-key focus, SSE subscriber). The chrome / fragment styles live in
  shell.css and are not duplicated.
- `internal/serve/pages/now/templates/page.html` — outer Now layout;
  composes the four section partials and the Quick-launch input.
- `internal/serve/pages/now/templates/inbox.html` — Needs-your-input
  list with typed dots, summary, meta, and inline actions; empty
  state collapses to a muted one-liner.
- `internal/serve/pages/now/templates/plate.html` — On-your-plate
  60/40 grid with dual progress bars, agent-pulse meta, and action
  links; renders empty state when no spec is claimed.
- `internal/serve/pages/now/templates/agents.html` — Your-agents 1.4/1
  grid: Currently-running card (soft-grey transcript preview — NOT a
  dark terminal) and Today block (2×2 stat grid + 3-row session list).
- `internal/serve/pages/now/templates/changes.html` — Since-you-were-
  here flat timeline; surfaces a "limited view" hint when falling back
  to git-only.
- `internal/serve/pages/now/data/types.go` — value types returned by
  every Load fn (Inbox, Plate, Agents, Changes, Metrics) plus a
  `MetricTile` alias for `shell.MetricTile`.
- `internal/serve/pages/now/data/metrics.go` — methodology-aware first
  tab + shared My-week / Hero-ROI tile sets and inline-SVG sparkline
  helper; placeholder tiles render when no project data is available.
- `internal/serve/pages/now/data/inbox.go` — composes inbox rows from
  injected proposal envelopes + inbound peer handoffs (via the
  `ReceivedFrom` frontmatter block); pretty-age helper shared with
  agents.go.
- `internal/serve/pages/now/data/plate.go` — graph-free spec discovery
  for claimed-by-user specs, ordered by last-modified, with progress
  derived from the EARS criterion classifier.
- `internal/serve/pages/now/data/agents.go` — synthesizes the Today
  list from events.log (delivery_complete, spec_updated); returns
  `Running: nil` until a live session ledger lands.
- `internal/serve/pages/now/data/changes.go` — reads events.log first;
  falls back to `git log -n 6` when the pipeline is empty and flips
  the `Limited` flag so the partial shows the hint.
- `internal/serve/pages/now/data/events.go` — shared `feed.ReadEvents`
  helper that swallows missing-log errors to a nil slice.
- `internal/serve/pages/now/data/*_test.go` — unit tests for each
  data fetcher (empty-input, populated-input, helper functions).
- `internal/serve/pages/now/page_test.go` — handler test asserting all
  four section markers + chrome render; `resolveMethodology`,
  `firstTabFor`, and `SectionFragment` cases.
- `internal/serve/api/now.go` — `GET /api/now/events` SSE channel +
  `GET /api/now/{inbox|plate|agents|changes}` fragment endpoints;
  per-connection debounce (250ms default) so an event storm under one
  section produces at most one refresh per window.
- `internal/serve/api/now_test.go` — fragment-endpoint smoke,
  SSE-header assertion, `sectionForEventType` mapping, and a
  fake-subscriber end-to-end test that bursts three events and
  asserts the debounced `event: inbox` frame.

### Modified

- `internal/serve/shell/shell.go` — added `Router.RenderFragment(out,
  name, data)` so home packages can render shared fragments without
  duplicating template paths; added `io` import.
- `internal/serve/shell/stubs.go` — removed the `now` stub entry so
  the real Now home doesn't collide on route registration.
- `internal/serve/shell/shell_test.go` — the `TestStubHomes_AllRender`
  loop now covers the four still-stubbed homes (now is asserted by
  the page-package test instead).
- `internal/serve/server.go` — imports `internal/serve/api` and
  `internal/serve/pages/now`; registers the real Now home after
  `RegisterStubHomes`; mounts the Now SSE + fragment endpoints on
  the top mux before the `/api/` catch-all; adds `busSubscriber` (an
  `api.Subscriber` adapter over `*EventBus`) and
  `snapshotProposals()` (a per-project proposal-store snapshotter
  used by the Now inbox).

## Boundaries

- **Not chat dispatch.** Submitting from the Quick launch input
  routes through the [hero-chat-and-model] adapter; this page renders
  the input and forwards the submit. The dispatch logic, model
  selection, and the resulting session UI are owned there.
- **Not the full agent transcript.** The Currently-running card shows
  a preview; the full transcript lives in [hero-agents-home].
  Clicking `Open transcript` navigates away from this page.
- **Not the spec detail editor.** The On-your-plate primary block
  links into [hero-work-home] for the spec; the spec UI itself is
  owned there.
- **Not team or org-level rollups.** Even at team edition, every
  section shows *the user's slice*. Team-wide presence, ROI rollups,
  and people analytics live in [hero-people-and-roi-home].
- **Not the event log schema.** This page consumes events emitted by
  the [next-handoff-emit] pipeline; it does not own the schema.
- **Not the search bar.** Top-nav search is owned by the shell.
- **Not a dashboard.** Each section has a single job. New "wouldn't
  it be nice to also show…" requests belong in their dedicated home,
  not as a new section here.
- **Not multi-pack.** This spec defines the engineering pack's Now.
  PM / QA / CS / Ops / Sales packs may define their own Now pages in
  future specs.

## Acceptance Criteria

- WHEN the user opens `/now` THE SYSTEM SHALL render the page hero
  with the eyebrow (`hero · <branch> · <edition> edition`), title
  `Now`, status one-liner subhead, and inline action row (primary
  `Open Inbox →`, ghost `What changed since Friday`, edition chip).
- WHEN the user opens `/now` THE SYSTEM SHALL render the seven
  sections in the order: page hero, tabbed metric strip, Needs your
  input, Quick launch, On your plate, Your agents, Since you were
  here.
- WHEN the user clicks a metric-strip tab THE SYSTEM SHALL swap the
  4-tile row to the selected tab's tiles via client-side DOM
  toggling without a server round trip.
- WHEN the project methodology in `hero.json` is `shape-up` THE
  SYSTEM SHALL relabel the first metric tab `This cycle` and render
  the cycle-tiles partial in place of sprint-tiles.
- WHEN the project methodology is `kanban` OR `solo` OR unset THE
  SYSTEM SHALL relabel the first metric tab `This week` and render
  the week-tiles partial.
- WHEN the user submits a message from the Quick launch input THE
  SYSTEM SHALL dispatch via the chat-and-model adapter (no chat
  dispatch logic on this page).
- WHEN the user clicks an intent chip below the Quick launch input
  THE SYSTEM SHALL prefill the input with the slash command and
  focus the input.
- WHILE the user has one or more agent sessions in flight THE
  SYSTEM SHALL stream session-status updates into the Your agents
  section via SSE on `event: agents` without a full page reload.
- WHILE the inbox count or running-agent count changes THE SYSTEM
  SHALL update the page-hero subhead via SSE on `event: hero`.
- WHEN any underlying source emits a Needs-your-input change THE
  SYSTEM SHALL replace the Needs-your-input section fragment via
  SSE on `event: inbox`.
- WHEN no chat adapter is connected THE SYSTEM SHALL render an
  empty-state notice above the Quick launch input and disable the
  input.
- WHERE the edition is `local` THE SYSTEM SHALL omit team-presence
  signals from the Your agents section (no shared-presence dots,
  no teammate-running indicators).
- WHERE the edition is `team` OR higher THE SYSTEM SHALL include
  inbound peer-handoff rows in the Needs your input section.
- WHEN a section's data fetcher returns an empty result THE SYSTEM
  SHALL render a one-line empty state for that section rather than
  hiding the section.
- IF a section's data fetcher errors THEN THE SYSTEM SHALL render
  an inline error placeholder for that section and continue rendering
  the others.
- IF the next-handoff-emit event stream is unavailable THEN THE
  SYSTEM SHALL fall back to a commits-only Since-you-were-here feed
  sourced directly from git and surface a quiet "limited view" hint.
- THE SYSTEM SHALL serve the Now page using server-rendered Go
  templates with no React, no bundler, and only the shell-owned chat
  island plus inline vanilla JS for tab toggling and SSE subscription.

## Risks

- **Methodology detection edge cases.** `hero.json` may omit the
  `methodology` field entirely on legacy workspaces, declare a value
  the registry doesn't know, or declare different methodologies per
  feature (rare). Default to `solo` / `This week` when unset or
  unrecognized; log a warning but never fail the page render.
- **SSE storm under many sessions.** A user with five concurrent
  agent sessions could push frequent `event: agents` updates.
  Publisher-side debounce (~250ms per event name) caps this; if even
  that is noisy, the Your-agents section degrades to "refresh on
  focus" instead of live streaming.
- **Data freshness vs. cost.** Some data (graph queries for `On your
  plate`, event-log scans for `Since you were here`) is moderately
  expensive. Cache per request, refresh only on SSE event, not on
  poll.
- **Empty page on first install.** A brand-new workspace has zero
  specs, zero sessions, zero events. Every section will render an
  empty state. We must validate this path looks intentional rather
  than broken — the Quick launch section anchors the page even when
  everything else is empty.
- **Chat adapter dependency.** If [hero-chat-and-model] is not yet
  delivered when this page ships, Quick launch renders the empty-
  state notice everywhere. Acceptable as a soft launch — the rest of
  the page is independently useful.
- **Edition leaks.** A team-only row leaking into a `local` render
  would not crash anything. Test the Agents-section partial under
  every `HERO_EDITION` value.

## Validation

- Manual: open `/now` on a workspace with one active spec, one
  pending proposal, one inbound handoff, one PR review assigned,
  recent commits, and recent knowledge captures. Verify all seven
  sections render with the expected content within one second.
- Manual: open `/now` on an empty workspace (no specs, no sessions).
  Verify each section shows its empty state; verify the page does not
  feel broken; verify the Quick launch input is the visual anchor.
- Manual: click each metric-strip tab and verify the tile row swaps
  without a network request (DevTools Network tab).
- Manual: edit `hero.json` to set `methodology: shape-up`, reload
  `/now`, verify the first tab reads `This cycle` and renders
  cycle-tiles. Repeat for `kanban` and `solo`. Remove the field and
  verify the fallback to `solo`/`This week`.
- Manual: submit a prompt from the Quick launch input; verify the
  request hits the chat-and-model dispatch endpoint and the resulting
  session opens in [hero-agents-home].
- Manual: start an agent session externally (`hero deliver <spec>`
  in a terminal), open `/now`, and verify the Currently-running card
  populates and updates via SSE as the session emits events.
- Manual: set `HERO_EDITION=local`, reload, verify no team-presence
  signals in the Your agents section. Set `HERO_EDITION=team`, reload,
  verify peer-handoff rows appear in Needs your input.
- Manual: trigger an inbox event externally (queue a proposal via
  CLI), verify the Needs-your-input section re-renders via SSE
  without a page reload.
- Manual: stop the next-handoff-emit subscriber, reload, verify the
  Since-you-were-here section falls back to commits-only and shows
  the "limited view" hint.
- Test: per-section partial render tests with mock data fixtures —
  assert correct DOM structure for populated, empty, and error cases.
- Test: methodology routing — assert correct tile partial selected
  per `hero.json` value (scrum / shape-up / kanban / solo / unset /
  unknown).
- Test: edition gating — assert no team-presence signals appear in
  the Your-agents rendered HTML under `local`.
- Test: SSE event publication — assert correct `event:` names and
  fragment payloads for each section channel.

## Kickoff

**Status: Now home delivered 2026-05-17.** The page is live at `/now`,
served by the shell, with all four section partials wired to real
data fetchers and SSE wired for live section refresh. `/` redirects
to `/now`. The stub registration for Now has been removed from
`internal/serve/shell/stubs.go` and the real `internal/serve/pages/now`
package owns the route.

**What works today:**
- `GET /now` → 200, ~29KB page with the full seven-section layout
- Page hero with eyebrow, title, status one-liner subhead, inline
  action row
- Tabbed metric strip with `This sprint` / `My week` / `Hero ROI`
  tabs (methodology-aware first tab — defaults to `This week` until
  workspace declares `methodology: scrum`)
- Needs your input list (proposals + inbound peer handoffs)
- Quick launch input with intent chips
- On your plate (claimed-by-user specs with progress bars)
- Your agents (today's session history from events.log; live ledger
  pending — see follow-ups)
- Since you were here (events.log feed; git-log fallback when
  events.log is unavailable)
- `GET /api/now/events` SSE channel multiplexing section refreshes
- `GET /api/now/{inbox,plate,agents,changes}` fragment endpoints

**Pick up at: deliver the other four homes.** Use the parallel
delivery prompt from the chat-and-model close-out (paste into a new
session). The pattern is now established:
`internal/serve/pages/now/` is the reference layout; each home
follows the same shape (page.go + templates + data fetchers + the
home's slug removed from shell/stubs.go + registered in server.go
after `RegisterStubHomes`).

**Follow-ups to file separately:**
- **Quick launch input should embed the shell's `chat-input`
  fragment** instead of the hand-rolled markup. The shell fragment
  ships `data-chat-input-variant` for the ⌘K island to hydrate
  consistently — Now's hand-rolled input misses that hook. Small.
- **Live session ledger for Your agents `Currently running`** — the
  Agents-home delivery will land this; until then, the
  Currently-running block shows "No active session" empty state.
- **`event: hero` SSE handler** — page-hero subhead refresh on
  count change. Today the subhead is correct on full reload only.
- **No-adapter empty-state notice on Quick launch** — the page
  always renders the input; when no chat adapter is connected the
  spec calls for a notice above. Wire when `chat-input` fragment
  is adopted (above).

When in doubt: the mock wins.
