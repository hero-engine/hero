---
title: Hero People + ROI Home — Team Pulse and Hero Value Surface
type: feature
status: planning
tags: [serve, surface, people, roi, team, presence, activity, velocity, metrics, web-app]
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
  - target: team-coordination
    kind: consumes
  - target: team-activity-feed
    kind: consumes
  - target: agent-contribution-tracking
    kind: consumes
  - target: effort-estimation
    kind: consumes
  - target: cost-calibration
    kind: consumes
  - target: hero-team-experience
    kind: relates-to
  - target: executive-report
    kind: relates-to
horizon: now
---

## Context

The fifth top-level home in the engineering pack covers two audiences
that share infrastructure but rarely share screens. **People** is the
team-coordination surface: who is working on what, who handed what
off, what an individual has been focused on this week. **ROI** is the
executive surface: Hero's value to the team and to the business,
framed in numbers that survive a CFO conversation — hours saved,
dollars saved, net value, ROI multiple, autonomy ratio, knowledge
reuse.

Both pull from the same primitives — `.hero/events.log`
([team-activity-feed](../../../specs/team-activity-feed/spec.md)),
spec frontmatter (`claimed_by`, `claimed_at`, status timestamps from
[agent-contribution-tracking](../../../specs/agent-contribution-tracking/spec.md)),
the inline-proposals store, the adapter-reported cost events
([hero-chat-and-model](../hero-chat-and-model/spec.md)), and the
saved-hours estimator and calibration data from
[effort-estimation](../../../specs/effort-estimation/spec.md) and
[cost-calibration](../../../specs/cost-calibration/spec.md). Today
those primitives are scattered across CLI commands (`hero feed`,
`hero velocity`, `hero claim`, `hero check`). No surface stitches
them together for either audience.

The ROI piece in particular has been undersold. It currently lives
nowhere visible: `hero pulse` shows status, `hero velocity` shows raw
counts, and the
[executive-report](../../../skills/executive-report/SKILL.md) skill
emits a markdown file once on demand. None of this is the view a VP
of Engineering pins in a tab and shows their boss. **ROI is the
surface that gets Hero adopted past the lead developer.** This spec
treats it as a first-class executive home, not a footnote to People.

The parent initiative
([hero-surface-architecture](../../initiatives/hero-surface-architecture/spec.md))
locks this as one of five top-level homes and the
[deployment-and-rendering](../hero-surface-deployment-and-rendering/spec.md)
decision pins the shape: **a web app companion to the CLI** with a
slim ~56px top nav as the only fixed chrome, a scrolling content
column ~1200px wide, sections that stack vertically with breathing
room, optional sub-nav row per home, Go templates + SSE for chrome
and lists, hand-rolled vanilla web component islands only where
templates fall short. **No fixed left rail, no VS Code tab strip, no
fixed bottom verb strip, no fixed right ambient panel.** The visual
source of truth is
[mockups/01-roi-overview.html](mockups/01-roi-overview.html); when
this spec and that mockup disagree, the mockup wins.

An earlier draft of this spec described a desktop-shell layout
(left-nav openables, VS Code tabs, bottom verb strip, right ambient
panel). That layout has been **rejected** and is superseded by the
web-app grammar described here.

## Goal

A scrolling web-app surface mounted at `/people`, with a slim
sub-nav row that distinguishes People sub-views from ROI sub-views,
that:

1. Opens by default to **Pulse** — a live view of who is working on
   what (presence + claims + active sessions strip + recent activity
   feed).
2. Provides four People sub-views (Pulse, Activity, Handoffs,
   Profiles) and five ROI sub-views (ROI Overview, Velocity,
   Autonomy, Knowledge reuse, My productivity), plus one
   enterprise-only **Export ↗** sub-view.
3. Lands the **Money chain** — hours saved, dollars saved, net
   value, ROI multiple — as four equal-weight tiles on the ROI
   Overview's default **Money** metric tab, with the chain visible
   on each tile's sub-line so a reader can verify the math without
   opening the methodology modal.
4. Computes every ROI metric server-side from existing primitives
   with explicit, documented formulas published at
   `/api/roi/methodology`. The Money chain runs off configurable
   coefficients in `hero.json` under `roi.coefficients`.
5. Lets managers, leads, and ICs each see something useful — without
   any IC's productivity data leaking unless they opted in, and
   without ever rendering a per-engineer dollar breakdown.
6. Renders mostly via Go templates + inline SVG (no client chart
   library); reserves at most one chart island
   (`roi-chart.js`) for the 12-week trend chart's hover/zoom/metric
   switch and two tiny control islands (`feed-filters.js`,
   `window-picker.js`).
7. Gates views by `HERO_EDITION` and role through the view
   registry: `local` users see only **My productivity** (self-only);
   `team` / `cloud` users see the full People + ROI set; only
   `enterprise` exposes **Export ↗**.

When this lands, a developer opens `/people` and sees who else is in
the codebase right now. A lead clicks `ROI Overview` and sees how
the team is moving. A VP opens the same URL and sees *"Last 4 weeks
· 10 engineers · 1 repo · 142 specs delivered · ~$49.9K net value ·
44× ROI"* — and every number is one click from its formula.

## Approach

### Page structure

A single `/people` page tree with a slim sub-nav row. The sub-nav
contains, in order:

`Pulse` · `Activity` · `Handoffs` · `Profiles` · `|` · `ROI
Overview` · `Velocity` · `Autonomy` · `Knowledge reuse` · `My
productivity` · `Export ↗`

The `|` is a visual divider between the People cluster and the ROI
cluster, rendered as a 1px tall border-strong divider. The
**Export ↗** tab is right-aligned via `margin-left: auto`, faded
(`color: ink-4`), and carries an `Enterprise` pill badge. The faded
treatment + badge is the affordance — it is visually present in
every edition so users can see what exists in higher tiers, but
hidden entirely when `HERO_EDITION=local` (because at the local
edition there is no team to roll up at all).

### Routes

| Route | View | Default? |
|---|---|---|
| `/people` | Pulse | Default landing for the home |
| `/people/activity` | Full activity feed (filterable) | |
| `/people/handoffs` | Cross-repo handoff stream | |
| `/people/profiles/<user>` | Per-person profile | |
| `/people/roi` | ROI Overview — **the mock** | Default ROI landing |
| `/people/roi/velocity` | Velocity / cycle time / throughput | |
| `/people/roi/autonomy` | Agent autonomy deep-dive | |
| `/people/roi/knowledge` | Knowledge reuse | |
| `/people/roi/individual` | Personal productivity (opt-in to share) | |
| `/people/roi/export` | Enterprise export bundle | Enterprise only |

Every route is a real URL (deep-linkable, back-button-friendly,
shareable). Each is served by a Go template under
`internal/serve/pages/people/templates/` or `…/people/roi/templates/`.

### Sub-nav row (shared)

A single shared template `internal/serve/pages/people/templates/sub-nav.html`
renders the row below the shell's top nav. It is a horizontal flex
container with sub-nav tabs styled per the mock: pill-shaped
(`border-radius: 6px`), 6×12 padding, 13px font, `ink-3` resting /
`hero-blue-700` active on `hero-blue-50` background. The People→ROI
divider is a `<span class="subnav-divider">`. The Export tab uses
the faded variant with the enterprise badge. The active tab is
determined server-side from the request path.

### ROI Overview view (the headline, per the mock)

This is the page a VP pins. Top-down structure:

#### 1. Top nav — shell-owned

Rendered by `hero-surface-shell`. `People` is the active top-level
tab. Nothing in this spec touches that chrome.

#### 2. Sub-nav row

`ROI Overview` is the active sub-nav tab.

#### 3. Page hero

- **Eyebrow** — `hero · main · roi`, prefixed with the small bolt
  SVG icon (`ink-4`, 12px font, 0.02em letter-spacing).
- **Title** — `ROI Overview`. 34px, weight 600, `hero-ink`, `-0.02em`
  letter-spacing, 1.15 line-height.
- **Subhead** — one line summarising the window. Format:
  `Last 4 weeks · 10 engineers · 1 repo · 142 specs delivered ·
  ~$49.9K net value · 44× ROI`. The dollar/net value/ROI multiple
  segment is wrapped in `<span class="headline">` and rendered in
  `hero-ink` weight 700 so it pops against the rest of the line.
- **Inline action row** — ghost `Last 4 weeks ▾` (opens the window
  picker), ghost `Compare prior 4w`, ghost `Schedule weekly report`,
  faded `Export PDF` with an `Enterprise` pill badge. The action
  row flex-wraps and right-aligns on wide viewports; on narrow it
  stacks under the subhead.

#### 4. Tabbed metric strip

Three text-link tabs along a 1px bottom border:
**Money** (default) · **Throughput** · **Quality**.

Active tab gets a `hero-blue-700` 2px underline overlapping the
bottom border. On the right of the tab row sits a small
`Window: Apr 19 – May 17, 2026` label (12px, `ink-4`) so the
displayed numbers always have an explicit window stamp.

**Money pane** (the active default — four tiles, equal weight,
`grid-template-columns: repeat(4, 1fr)`, 18px gap):

| Tile | Headline | Sub-line | Trend pill | Visual |
|---|---|---|---|---|
| **Hours saved** | `~340h` | "vs unassisted baseline" | `▲ +22% vs prior 4w` | Inline-SVG area sparkline (200×36) |
| **Dollars saved** | `~$51K` | "≈340h × $150/hr loaded" | `▲ +22% vs prior 4w` | Inline-SVG area sparkline |
| **Net value** | `$49.9K` | "$51K saved − $1.14K Hero API spend" | `▲ +24% vs prior 4w` | Thin 8px segmented bar (savings vs spend split) with legend underneath |
| **ROI multiple** | `44×` | "net value ÷ API spend" | `▲ +3.2× vs prior 4w` | Inline-SVG area sparkline |

Tile chrome: `bg-secondary` (#f5f7fa) background, 10px radius, 22px
padding, min-height 168px, flex column, value uses
`font-variant-numeric: tabular-nums`. The `~` prefix on
approximations is rendered in a smaller weight (22px, `ink-3`) so
the precision claim matches reality.

**Trend pill rules**: `success` color + `#e8f6ec` background when the
delta is positive (positive in the "good direction" sense — for
ROI-relevant metrics, all four Money tiles consider "up" good). A
`.flat` variant renders in `ink-3` on `bg-softer` when the delta is
within ±1%.

**Net value tile segmented bar**: replaces the sparkline because the
two scalars (savings vs spend) tell the story more directly. Bar is
8px tall, two segments, widths proportional to `dollars_saved` and
`hero_api_spend`. Legend underneath shows the swatches with values.

**Throughput pane** (tab content, hidden until selected): four
tiles using identical chrome. Headlines and subs:

| Tile | Headline | Sub-line |
|---|---|---|
| Specs delivered | `142` | "in window" |
| Autonomy ratio 7d | `71%` | "without-edit merges ÷ all merges" |
| Cycle time | `4.1d` | "median: claimed → completed" |
| Time-to-spec | `2.4d` | "median: imported → designed" |

**Quality pane** (tab content, hidden until selected):

| Tile | Headline | Sub-line |
|---|---|---|
| Spec coverage 7d | `84%` | "merged commits linked to a spec" |
| Re-review rate | `3.1%` | "completed → reopened" |
| Knowledge reuse | `84%` | "agent sessions injecting ≥1 entry" |
| Drift catches | `11` | "spec/code mismatches surfaced" |

Tab switching is a tiny piece of inline JS on the page (~20 lines)
that toggles `aria-selected` + `.active` and shows/hides the panes.
Not an island — the panes are all server-rendered HTML and switching
is local visual state only.

#### 5. Methodology footnote

Centered block immediately under the metric strip. Max-width 720px,
14px font (`ink-3`), 1.6 line-height, `bg-soft` background with 1px
`border` border, 8px radius. Text reads:

> ⓘ Numbers use configurable coefficients (**loaded hourly cost:
> $150** · time-per-edit: 4min · auto-import time saved: 12min · …).
> View methodology → · Tune coefficients →

The two links open `/people/roi/methodology` (modal, served as a
fragment) and `/people/roi/coefficients` (form for editing
`hero.json`, gated by the `roi-admin` role). **This block is the
credibility anchor.** It renders on every ROI sub-view, not only
Overview. A user looking at any ROI number can reach the formula in
one click.

#### 6. Section: How time was spent

Section header `How time was spent` (20px, weight 600).

Body: a full-width 56px stacked bar (3 segments) with each segment
labeled inline:

- 61% Agent autonomous (`hero-blue-700`)
- 24% Agent + human review (`hero-blue-500`)
- 15% Human authored (`ink-5`, dark text)

Legend underneath repeats the three categories with their swatches
and percentages. Below the legend, a max-width 720px caption:
*"Agents wrote **61% of merged work fully autonomously** this period
— up from 53% in the prior 4-week window."* The bold span uses
`hero-ink` weight 600. The comparison number comes from the
prior-period delta computation.

#### 7. Section: Where the savings came from

Two-column grid (`grid-template-columns: 320px 1fr`, gap 56px,
align center). Collapses to single column under 1000px.

**Left**: an inline SVG donut chart (280×280) with 5 slices summing
to 100%. The slices, in order (largest first per the mock), use
`stroke-dasharray` on rotated `<circle>` elements at radius 100 (so
the circumference is `2π × 100 ≈ 628.3185`):

| Slice | % | Stroke | Approx hours | Approx $ |
|---|---|---|---|---|
| Agent-authored proposals merged without edits | 41% | `hero-blue-700` | ~140h | ~$21.0K |
| Auto-imported & triaged specs | 22% | `hero-blue-500` | ~75h | ~$11.2K |
| Knowledge re-injection into agent context | 18% | `hero-blue-300` | ~61h | ~$9.2K |
| Cross-spec drift catches | 11% | `warn` (#d97706) | ~37h | ~$5.6K |
| Automated reviews | 8% | `success` (#16a249) | ~27h | ~$4.0K |

Center of the donut shows three stacked labels: small `Total saved`
in `ink-4`, large `~$51K` in `hero-ink`, smaller `≈340h` in
`ink-4`.

**Right**: a vertical breakdown list. Each row is `14px 1fr auto
auto` grid: a 10px colored dot matching the slice, the label +
"% of savings" subtext, right-aligned hours ("~140h saved"),
right-aligned bold dollars ("~$21.0K"). 1px `border` divider between
rows. Rows ordered to match the donut (largest first).

#### 8. Section: 12-week trend

Section header `12-week trend` with the toggle chips
right-aligned in the header row:

- `Net value` (default active)
- `Hours saved`
- `Autonomy ratio`
- `Specs delivered`

Chip styling per the mock: 999px radius, 12px font, `bg-softer`
resting / `hero-blue-50` + `hero-blue-300` border active.

Body: a wide inline-SVG chart (960×300 viewBox) with the active
metric drawn as one solid line over a gradient area fill, plus two
auxiliary lines (`Agent autonomous` dashed `hero-blue-500`, `Agent +
human review` dashed `ink-5`). Y-axis grid lines (5 ticks), Y-axis
labels in `ink-4`, X-axis week labels (12 ticks: Feb 23 → May 11
in the mock — generated server-side from the current window).

Data points on the active metric line are 3px circles, white fill,
2px `hero-blue-700` stroke. One highlighted week (configurable;
defaults to the largest delta week) renders as a 5px filled circle
with a vertical dashed leader line up to a small dark tooltip rect
naming the week, the value, and the named anchor event (e.g.
`propose-shim landed`).

Hover behavior is the **one chart-interaction island**:
`internal/serve/islands/roi-chart.js`. The chip toggle and hover
tooltip + drag-range hand off to the island. The island reads its
data from a sibling `<script type="application/json">` block
emitted by the template — no fetch.

Legend below the chart: a horizontal row showing the three line
styles and what they mean.

#### 9. Section: Top contributors

Section header `Top contributors` with `last 4 weeks` in `ink-4`
12px as a metadata suffix.

A flat table (NOT cards) with 7 rows mixing humans and agents.
Columns:

| Column | Notes |
|---|---|
| **Contributor** | 28px circular gradient avatar (`av-h*` for human, `av-a*` for agent), name in `hero-ink` weight 500, AGENT/HUMAN badge underneath |
| **Specs touched** | Right-aligned, tabular-nums |
| **Autonomy %** | Right-aligned, tabular-nums |
| **Hours saved** | Right-aligned. Cell content: an 80×5 horizontal bar (`bg-softer` track, `hero-blue-500` fill, width proportional to row's share of the top contributor's hours) + numeric value to the right |
| **$ saved** | Right-aligned, tabular-nums, `hero-ink` weight 500 |

Badge styling: `AGENT` in `hero-blue-50` background / `hero-blue-700`
text; `HUMAN` in `bg-softer` background / `ink-3` text. Both 9.5px
weight 700 uppercase with 0.06em letter-spacing.

The dollar column is the **only** per-row dollar display the
surface ever renders, and it represents a contributor's **share of
team hours saved × the team `c_hourly_cost`** — never an individual
loaded cost. This is reinforced in the methodology modal.

Default sort: hours saved, descending. Mixes humans + agents in the
same list deliberately — the message is "your team includes
agents."

#### 10. Section: What changed this period

Section header `What changed this period`. Body: four one-line
highlights in a stacked list. Each row is `18px 1fr auto`: an 8px
colored status dot (`success` green or `warn` amber), the body
sentence (with bold spans on the load-bearing numbers and inline
links to specs), and a right-aligned date hint (e.g. `May 5`,
`last 4w`) in `ink-4` 12px.

Source: the period-over-period delta computation surfaces the
biggest deltas (positive: green dot; negative or watch-item: amber)
plus pinned narratives derived from event-log markers (e.g.
`anchor` events captured in this period).

#### 11. Footer

Rendered by `hero-surface-shell`. Not in this spec.

### People sub-views

#### Pulse (`/people`) — default landing

Three stacked blocks:

- **"Right now" pill** — single horizontal pill summarising the
  workspace: *"4 humans active · 2 agent sessions running · 1
  awaiting your approval"*. Numbers come from the presence + claims
  + proposals stores. The pill is its own slim card; the awaiting-
  approval count is a `hero-blue` chip when > 0.
- **Presence + claims grid** — one card per active team member.
  Each card: avatar, name, current active spec slug (resolved from
  `claimed_by` + recent events), session length, "agent attached"
  badge if a proposal-producing agent session is bound to that
  user. Cards with awaiting-your-approval proposals get a
  `hero-blue` badge ribbon. Empty in `local` edition.
- **Recent activity feed** — newest-first stream from
  `.hero/events.log` and the proposals queue. Each row carries an
  actor (human or agent), an event type, a target (spec slug or
  file group), and a relative timestamp. Updates via SSE fragment
  swap; new events arrive without a page reload. Server-side
  coalescing: at most one feed fragment per 500ms per subscriber to
  prevent SSE storms.

In `local` edition the Pulse view collapses to "no team
configured — see Activity for your own events" and the presence
grid disappears.

#### Activity (`/people/activity`)

Full-page version of the Pulse feed. Filter chips strip across the
top: actor type (humans only / agents only / both), specific
person, specific spec, event type, date range. Filter state is URL-
encoded (so a filtered view is shareable). Server-rendered list
with SSE updates. The filter chip strip is a small island
(`internal/serve/islands/feed-filters.js`) for instant client-side
state changes without a roundtrip; URL updates trigger
server-rendered list refresh.

#### Handoffs (`/people/handoffs`)

Cross-repo handoff stream consuming the peer-handoff store from
[cross-repo-peering](../../../specs/cross-repo-peering/spec.md).
Two columns: **Incoming** (awaiting accept) and **Outgoing**
(awaiting peer response). Each row: spec slug, peer alias, reason,
status, age. Inline `Accept`/`Decline`/`Comment` buttons per
incoming row. Empty state in `local` edition with no peers
configured: "No peers configured — see `hero peer add` to register
a sibling repo."

#### Profiles (`/people/profiles/<user>`)

Per-person profile, opened by clicking a presence card or a feed
actor. Three sections:

- **Header**: identity (avatar, name, role), current focus, session
  count window, claim count window.
- **Recent work**: last 30 days of specs touched, with role on each
  (`claimed` / `delivered` / `reviewed` / `paired-with-agent`).
- **Expertise heatmap**: server-rendered inline SVG. Rows are
  convention scopes / domain tags; columns are weeks. Cell shade =
  number of events the user produced touching that domain that
  week. Read-only signal — never used to rank or score.

### ROI sub-views (deep-dives)

#### Velocity (`/people/roi/velocity`)

Deep-dive on throughput and timing.

- **Velocity** — specs/week (server SVG line chart) split by spec
  type (feature / bug / chore / decision / convention).
- **Cycle time distribution** — histogram of
  `completed_at - claimed_at`, per spec type. Optional log Y-axis
  (island toggle).
- **Throughput** — count of specs reaching `completed` per week
  with a trendline.
- **Lead time** — distribution of `completed_at - created_at`, per
  spec type.

All charts share the same `roi-chart.js` island for range zoom +
hover tooltip.

#### Autonomy (`/people/roi/autonomy`)

Agent autonomy deep-dive.

- **Proposal funnel** — stacked bar by week: proposals submitted →
  reviewed → merged-without-edits / merged-with-edits / rejected.
- **Edits per merged proposal** — box plot per agent identity.
- **Cost per merged proposal** — sourced from the adapter-reported
  `chat.cost` events filtered by proposal correlation. Falls back
  to "—" when adapter cost tracking is unavailable.

Caption above the section: *"Autonomy ratio is going up while
edits-per-proposal is going down → the agent layer is maturing."*
Or a measured neutral statement when the trend is flat/negative.

#### Knowledge reuse (`/people/roi/knowledge`)

- **Reuse rate** — `unique_knowledge_entries_injected / total_agent_sessions`
  over the window. Single number + sparkline.
- **Top injected entries** — ranked list of conventions / decisions
  / notes most often pulled into agent context.
- **Stale knowledge** — entries never injected in the active window;
  candidates for review or archive.

Data source: a new context-injection log (see Changes §6) written
whenever Hero serves a `relevant`-style response. When the log is
absent (early-stage workspaces) the view shows an empty state with
a hint to enable context-injection tracking. The view does not
vanish.

#### My productivity (`/people/roi/individual`)

Personal productivity, **default private to you**, visible in every
edition (including `local` — this is the only ROI view a solo user
sees).

- **Your shipped specs** in the active window — list with one-line
  outcomes.
- **Your agent-assist ratio** — % of your delivered specs where an
  agent claimed and produced at least one merged proposal.
- **Your Hero hours saved (est.)** — saved-hours estimator scoped
  to specs you authored or claimed.
- **Your knowledge contributions** — conventions, decisions, notes
  you authored; how often they were injected into others' context.
- **Sharing controls** — toggle: *"Include my stats in team rollups
  by name."* Default **off**. When off, your work is still counted
  in aggregate totals but you appear as *"anonymous IC"* in the
  ROI Overview contributors table.

The view never compares your numbers to another named individual.
Sharing is opt-in and revocable without losing your own
visibility.

#### Export ↗ (`/people/roi/export`) — Enterprise only

A page that bundles the active ROI window into an exportable
artifact suitable for SOC2 / leadership review. Outputs:

- **JSON snapshot** — every metric + the methodology coefficients +
  the input-data hashes; signed when audit-chain is configured.
- **Markdown report** — same data, formatted like the
  [executive-report](../../../skills/executive-report/SKILL.md)
  skill output. The skill's runtime becomes a live render of this
  endpoint.
- **CSV** of contributor and per-spec records for analyst handoff.

Edition-gated via `editions: [enterprise]` on the view registration.
Audit retention applies (the export is recorded with timestamp +
requester identity).

### Metric definitions

Every metric has a single, written formula. Coefficients live in
`hero.json` under `roi.coefficients` so they are calibrated
per-workspace without code changes. The `/api/roi/methodology`
endpoint emits the exact formula and coefficient set used for the
most recent computation.

| Metric | Formula |
|---|---|
| **Spec coverage of merged commits** | `count(merged_commits where commit_message links a spec slug OR commit touched files in a spec's Changes section) / count(total_merged_commits)` over the active window. |
| **Time-to-spec** | `median(designed_at - imported_at)` per spec where both timestamps exist. `imported_at` is the `created` frontmatter date for imported specs; `designed_at` is the first `status: in-review` transition in `.hero/events.log`. |
| **Autonomy ratio** | `count(proposals_merged_without_edits) / count(proposals_merged_total)` over the window. "Without edits" means the merged commit's diff equals the proposal's diff byte-for-byte (whitespace-normalised). |
| **Re-review rate** | `count(specs that transitioned from completed back to in-review or planning) / count(specs that ever reached completed)` over the window. From status-transition events in `.hero/events.log`. |
| **Knowledge reuse rate** | `count(unique knowledge slugs injected into agent context) / count(distinct agent sessions)` over the window. Source: context-injection log. |
| **Velocity** | Specs reaching `completed` per week, segmented by `type`. |
| **Cycle time** | Per-spec `completed_at - claimed_at`; aggregated as median + p90 per type. |
| **Lead time** | Per-spec `completed_at - created_at`; aggregated identically. |
| **Drift catches** | Count of spec/code mismatches surfaced by `hero check` or `graph-conflict-detection` during the window. |

### Hours saved — the estimator

The headline ROI substrate. **Treat it as an estimate, label it as
one, and make the coefficients visible.** A confident-looking number
with opaque math destroys trust faster than no number at all.

```
hours_saved(window) =
    agent_proposals_merged_without_edits * c_no_edit
  + agent_proposals_merged_with_edits     * c_with_edit
  + auto_imported_specs                   * c_import_triage
  + auto_diagnosed_bugs                   * c_diagnosis
  + auto_reviewed_specs                   * c_design_review
  + knowledge_entries_injected            * c_context_lookup
  + scheduled_jobs_executed               * c_scheduled_run
```

Coefficient defaults (all in hours) in `hero.json`:

```json
{
  "roi": {
    "coefficients": {
      "c_no_edit":        1.5,
      "c_with_edit":      0.5,
      "c_import_triage":  0.1,
      "c_diagnosis":      1.0,
      "c_design_review":  0.5,
      "c_context_lookup": 0.05,
      "c_scheduled_run":  0.25
    }
  }
}
```

Defaults are conservative on purpose. The methodology modal
explains every coefficient and links to the source data. Workspaces
are encouraged to calibrate against their own measurements; the
[cost-calibration](../../../specs/cost-calibration/spec.md) work
already provides the pattern (estimated vs actual). Auto-piping
calibration output into these coefficients is out of scope here —
a future companion spec owns it.

### The Money chain — dollar saved, net value, ROI multiple

Hours saved is the substrate; managers think in dollars. Three
derived scalars close the loop and render as equal-weight tiles on
the ROI Overview **Money** tab alongside hours saved:

```
dollars_saved(window) = hours_saved(window) * c_hourly_cost
hero_api_spend(window) = sum(chat.cost events in window)   // from hero-chat-and-model
net_value(window)     = dollars_saved(window) - hero_api_spend(window)
roi_multiple(window)  = net_value(window) / hero_api_spend(window)
```

A new coefficient is added under `roi.coefficients`:

```json
{
  "roi": {
    "coefficients": {
      "c_hourly_cost": 150.0
    }
  }
}
```

- **Loaded hourly cost** (`c_hourly_cost`) — fully-loaded developer
  cost per hour (salary + benefits + overhead). Default $150.
  Configurable per workspace. Methodology modal exposes the value
  and lets `roi-admin` change it.
- **Hero API spend** (`hero_api_spend`) — sum of `cost_usd` from
  adapter-reported `chat.cost` events in the window. The
  [hero-chat-and-model](../hero-chat-and-model/spec.md) spec is
  the source of truth for those events; this surface aggregates
  them but does not enforce or display per-session budgets. If no
  adapter has been reporting costs, the field renders as `$0` with
  an inline hint *"adapter cost reporting not configured"* — the
  Net value tile then equals Dollars saved and the ROI multiple
  tile shows `—` with a tooltip explaining why.
- **ROI multiple** — render as `Nx` rounded to one decimal place
  when ≥10×, two decimals otherwise. Caps display at `999×` to
  avoid silly numbers in tiny windows where API spend rounds to
  zero.

**Headline framing.** On the ROI Overview page-hero subhead,
surface the net value and ROI multiple together: *"Last 4 weeks ·
142 specs delivered · ~$49.9K net value · 44× ROI"*. This is the
line that gets screenshotted into a budget review.

**Tile order on the Money tab.** Hours saved · Dollars saved · Net
value · ROI multiple — same tile size, same prominence, in that
left-to-right order. Sub-lines on each tile show the chain so a
reader can verify the math without opening the methodology modal:

- `Hours saved ~340h` — sub: "vs unassisted baseline"
- `Dollars saved ~$51K` — sub: "≈340h × $150/hr loaded"
- `Net value ~$49.9K` — sub: "$51K saved − $1.14K Hero API spend"
- `ROI multiple 44×` — sub: "net value ÷ API spend"

**Privacy of $ display.** Dollar figures are visible to anyone who
can see the ROI views in that edition. There is **no per-engineer
$ breakdown** — only team-level rollups and the contributor table's
share-of-savings column (which converts a contributor's hours-saved
share to dollars at the team `c_hourly_cost`, never that
individual's loaded cost). The methodology modal says this
explicitly.

### Rendering model

Per the parent decision spec: Go templates + SSE for everything,
with exactly three islands.

- **Templates.** `internal/serve/pages/people/templates/*.html` and
  `internal/serve/pages/people/roi/templates/*.html`. One template
  per view. A shared `sub-nav.html` partial is included by every
  template. A shared `methodology-footnote.html` partial is
  included by every ROI template.
- **Inline SVG.** All sparklines, the donut, the segmented bar,
  the trend chart, and the heatmap render as inline SVG emitted
  directly from the template — no client library. A Go template
  helper file (`internal/serve/pages/people/svgcharts.go`) provides
  `svgSparkline`, `svgArea`, `svgHistogram`, `svgDonut`,
  `svgHeatmap`, `svgSegBar`, `svgBoxPlot`. Each helper is one
  function, under 120 lines, no dependencies.
- **Chart island: `roi-chart.js`.** A hand-rolled vanilla web
  component hydrating any chart marked with `data-roi-chart`.
  Handles hover tooltip, range zoom, and active-metric switch for
  the 12-week trend. Reads its data from a sibling
  `<script type="application/json">` block emitted by the template.
  No fetch from the island. Used only on ROI Overview and the
  ROI deep-dive views.
- **Control islands: `feed-filters.js` and `window-picker.js`.**
  Each one is a thin reactive control that emits URL state
  changes; the shell handles navigation and the server re-renders
  the affected fragment.
- **Tab strip switching** (Money / Throughput / Quality) is **not**
  an island. It's ~20 lines of inline JS on the page that toggles
  `.active` and `aria-selected` on pre-rendered panes.
- **SSE.** People views (Pulse, Activity, Handoffs) subscribe to
  `/api/people/stream` for new events. ROI views refresh on a slow
  timer (default 60s); the SSE channel is shared but ROI
  subscribers receive only window-tick events.

### Data sources

| Surface need | Source |
|---|---|
| Presence | `team-coordination` claims + recent events |
| Activity feed | `.hero/events.log` (humans) + proposals store + agent session events (agents) |
| Handoffs | `cross-repo-peering` handoff store |
| Profiles | events.log joined to frontmatter `tags` + convention `scope` globs |
| Hours saved counters | proposals store, imported-spec records, scheduled-job log, context-injection log |
| Hero API spend | `chat.cost` events from `hero-chat-and-model` |
| Spec coverage | git commit log joined to spec frontmatter + `Changes` parse |
| Cycle / lead / time-to-spec | events.log status-transition records |

### Computation layer

Metric computation lives under `internal/serve/metrics/` — separate
from view templates so the same code drives the surface, the
executive-report skill, and any future `hero roi` CLI.

- One file per metric family: `autonomy.go`, `velocity.go`,
  `coverage.go`, `hours_saved.go`, `dollars.go`, `roi.go`,
  `knowledge.go`, `individual.go`, `methodology.go`.
- Each function is window-parameterised and pure given the
  underlying data; results are cacheable.
- A small in-memory cache with a 60s TTL is acceptable for v1;
  persistent caching is out of scope.

### API surface

People endpoints under `/api/people/*`:

- `GET /api/people/presence` — current presence + claims.
- `GET /api/people/feed?since=…&type=…&actor=…&slug=…&limit=…`
- `GET /api/people/handoffs`
- `GET /api/people/profile/:id`
- `GET /api/people/stream` — SSE channel.

ROI endpoints under `/api/roi/*`:

- `GET /api/roi/overview?window=…` — Money/Throughput/Quality tile
  values + sparklines + sections (time spent, savings breakdown,
  trend series, contributors, what changed).
- `GET /api/roi/velocity?window=…`
- `GET /api/roi/autonomy?window=…`
- `GET /api/roi/knowledge?window=…`
- `GET /api/roi/individual?window=…` — self-only; respects opt-in
  state when invoked by another user.
- `GET /api/roi/methodology` — current coefficients + formula for
  every metric + source-data hashes for the most recent
  computation.
- `GET /api/roi/export?window=…&format=json|md|csv` — enterprise-
  only; gated by edition tag.

### Identity, privacy, and gating

- **`/people/roi/individual` is self-only by default.** A user can
  view their own page in every edition. A user can view another
  user's individual page only if that user opted in to team-visible
  stats.
- **Aggregate rollups anonymise non-opted-in users.** Their work
  still contributes to totals but they render as *"anonymous IC"*
  in any contributors table or strip.
- **No per-engineer dollar breakdowns**, ever. The contributor
  table's `$ saved` column is hours-share × team `c_hourly_cost` —
  not an individual loaded cost.
- **Edition gating** via view-registry `editions:`:
  - `local` — only `/people/roi/individual` (self-only). All
    People sub-views and all team ROI sub-views hidden. The
    sub-nav row collapses to a single tab.
  - `team` — all sub-views except `Export ↗` visible; data scope is
    the configured LAN team.
  - `cloud` — all sub-views except `Export ↗` visible; data scope
    is the org.
  - `enterprise` — all sub-views visible including `Export ↗`;
    audit retention enabled.
- **RBAC (enterprise).** A `roi-viewer` role gates ROI Overview,
  Velocity, Autonomy, Knowledge reuse, Export. A `roi-admin` role
  can change coefficients. Defaults: every team member is
  `roi-viewer`; only workspace admins are `roi-admin`.

## Changes

1. **Page handlers** —
   `internal/serve/pages/people/page.go`
   - Register handlers for every route in the routes table above.
   - Each handler resolves the active window, calls the
     computation layer, and renders the appropriate template with
     the `methodology-footnote.html` partial included for ROI
     views and the `sub-nav.html` partial included for every view.

2. **People templates** —
   `internal/serve/pages/people/templates/`
   - `sub-nav.html` — shared sub-nav row partial.
   - `pulse.html` — Pulse landing view.
   - `activity.html` — Filterable activity feed.
   - `handoffs.html` — Handoff stream.
   - `profile.html` — Per-person profile.

3. **ROI templates** —
   `internal/serve/pages/people/roi/templates/`
   - `overview.html` — the headline view (matches
     `mockups/01-roi-overview.html`).
   - `velocity.html`
   - `autonomy.html`
   - `knowledge.html`
   - `individual.html`
   - `export.html` (enterprise-gated)
   - `methodology-footnote.html` — shared partial included by every
     ROI template.
   - `methodology-modal.html` — fragment served at
     `/people/roi/methodology` and opened from the footnote link.

4. **Data fetchers** —
   `internal/serve/pages/people/data/`
   - `presence.go`, `feed.go`, `handoffs.go`, `profile.go` for
     People sub-views.
   - `overview.go`, `velocity.go`, `autonomy.go`, `knowledge.go`,
     `individual.go`, `export.go` for ROI sub-views. Each one
     composes the metrics package outputs into the template's
     view-model struct.

5. **Metrics package** —
   `internal/serve/metrics/` (new)
   - `autonomy.go` — autonomy ratio + funnel.
   - `velocity.go` — velocity + cycle time + lead time + throughput.
   - `coverage.go` — spec coverage of merged commits.
   - `hours_saved.go` — the saved-hours estimator with the seven
     coefficient knobs.
   - `dollars.go` — `dollars_saved` = `hours_saved * c_hourly_cost`.
   - `roi.go` — `hero_api_spend`, `net_value`, `roi_multiple`,
     with explicit edge-case behavior (zero spend → ROI multiple
     renders `—`, capped at `999×`).
   - `knowledge.go` — reuse rate + top-injected + stale.
   - `individual.go` — per-user scoped saved-hours + share view.
   - `methodology.go` — emits the methodology record consumed by
     the modal and `/api/roi/methodology`.
   - `metrics_test.go` — fixture-driven assertions for every
     formula.

6. **Context-injection log** —
   New write path under `internal/serve/relevant/` (and wherever
   `relevant`-style responses are emitted) that records
   `{session_id, ts, slugs_injected[]}` to
   `.hero/context-injection.log` as append-only JSONL. Required by
   `knowledge.go`; absence triggers the Knowledge reuse view's
   empty state.

7. **API handlers** —
   - `internal/serve/api/people.go` — `/api/people/*` endpoints.
   - `internal/serve/api/roi.go` — `/api/roi/*` endpoints.
   - SSE stream wired into existing events.log infrastructure with
     500ms-per-subscriber coalescing.

8. **Islands** —
   `internal/serve/islands/`
   - `roi-chart.js` — chart hover / range zoom / metric switch for
     the 12-week trend (and the velocity/autonomy chart blocks).
   - `feed-filters.js` — Activity feed filter chips.
   - `window-picker.js` — ROI window selector dropdown.
   - Each authored as a single ES module, served directly, no
     bundler.

9. **SVG chart helpers** —
   `internal/serve/pages/people/svgcharts.go`
   - `svgSparkline`, `svgArea`, `svgHistogram`, `svgDonut`,
     `svgHeatmap`, `svgSegBar`, `svgBoxPlot`. Each under 120
     lines, no dependencies.

10. **Configuration** —
    `internal/config/config.go`
    - Add `roi.coefficients` block carrying all 8 coefficients
      (the 7 saved-hours coefficients + `c_hourly_cost`).
    - Add `roi.individual_sharing` keyed by user identity for the
      opt-in record.
    - Add `roi.attribution` switch (`agent` default, `reviewer`
      alternative) for autonomy-ratio attribution.

11. **View registry registrations** —
    `internal/serve/packs/engineering/pack.go`
    - Register `people-pulse` (default), `people-activity`,
      `people-handoffs`, `people-profile`, `roi-overview`,
      `roi-velocity`, `roi-autonomy`, `roi-knowledge`,
      `roi-individual`, `roi-export`.
    - Each registration carries `editions`, `roles`, and renderer
      metadata per the shell view-registry contract.

12. **Executive-report skill integration** —
    `skills/executive-report/SKILL.md`
    - Update to point at `/api/roi/export?format=md` as the
      authoritative renderer; CLI fallback remains for offline use
      but the surface is canonical.

13. **Edition gating tests** —
    `internal/serve/registry/registry_test.go`
    - Assert the per-edition sub-view visibility matrix matches the
      Identity/privacy/gating section.

## Boundaries

- **Not a per-spec drift or contract view.** That belongs to
  `hero-work-home`. People + ROI shows rollups, not per-spec
  delivery state.
- **Not the live agent session viewer.** That belongs to
  `hero-agents-home`. People views deep-link into a session but the
  live transcript itself lives on the Agents home.
- **Not the knowledge browser.** `roi-knowledge` shows which
  entries are load-bearing, not the entries themselves. Click-
  through opens the entry in `hero-knowledge-home`.
- **Not a coefficient calibration system.** Coefficients have
  defaults and are editable in `hero.json` (or via the
  `roi-admin`-gated Tune coefficients form); auto-calibrating them
  against measured outcomes is a separate spec.
- **Not a budget enforcement surface.** `chat.cost` events are
  aggregated for display; per-session budgets, throttling, or
  blocking belong to `hero-chat-and-model`.
- **Not a federation surface.** Cross-repo aggregation is mentioned
  for cloud + enterprise but the wire protocol lives with
  [cross-repo-peering](../../../specs/cross-repo-peering/spec.md).
- **Not a per-engineer scoreboard.** Every design choice here
  defaults toward personal feedback and aggregate visibility, never
  ranked individual comparison. There is no per-engineer dollar
  breakdown anywhere on the surface.
- **Not a desktop-shell view.** No fixed left nav, VS Code tabs,
  bottom verb strip, or right ambient panel. The earlier draft
  describing that layout is explicitly superseded.

## Acceptance Criteria

- WHEN the user opens `/people` THE SYSTEM SHALL render the Pulse
  view with the sub-nav row, the "Right now" pill, the presence +
  claims grid, and the recent activity feed.
- WHEN a team event is appended to `.hero/events.log` WHILE Pulse
  or Activity is open THE SYSTEM SHALL stream the new row into the
  feed via SSE without a full page reload.
- WHEN the user filters the Activity feed by actor type, spec, or
  date range THE SYSTEM SHALL update the filter chips, the URL, and
  the rendered list to match.
- WHEN the user clicks a presence card or a feed actor THE SYSTEM
  SHALL navigate to the corresponding `/people/profiles/<user>`
  page.
- WHEN the user opens `/people/roi` THE SYSTEM SHALL render the ROI
  Overview with the Money metric tab active by default.
- WHEN the Money tab is active THE SYSTEM SHALL render four
  equal-weight tiles (Hours saved, Dollars saved, Net value, ROI
  multiple) in that left-to-right order, each with the chain-
  exposing sub-line documented in the Approach section.
- WHEN the user clicks the Throughput or Quality metric tab THE
  SYSTEM SHALL show the corresponding pre-rendered pane and hide
  the others without a network roundtrip.
- WHEN the user changes the time window via the window picker THE
  SYSTEM SHALL recompute every metric tile, the savings breakdown,
  the 12-week chart, the contributors table, and the page-hero
  subhead for the new window.
- WHEN the user changes the `roi.coefficients.c_hourly_cost` value
  in `hero.json` THE SYSTEM SHALL recompute the Dollars saved, Net
  value, and ROI multiple tiles on next page load (or via SSE on
  save when the surface is open).
- WHEN the chart island loads THE SYSTEM SHALL hydrate it from a
  sibling `<script type="application/json">` block and SHALL NOT
  fetch its data from the network.
- WHEN the user clicks `View methodology →` on any ROI view THE
  SYSTEM SHALL open a modal listing every metric's formula and the
  current coefficient values.
- WHILE a user has not opted in to team-visible stats THE SYSTEM
  SHALL render that user as *"anonymous IC"* in every aggregate
  ROI view and SHALL still include their work in totals.
- WHILE a user is viewing `/people/roi/individual` for themselves
  THE SYSTEM SHALL render their personal metrics regardless of
  opt-in state.
- WHILE the edition is `local` THE SYSTEM SHALL only render
  `/people/roi/individual` (self-only) and SHALL hide the People
  sub-views and all team ROI sub-views.
- WHERE the edition IS `team` OR `cloud` THE SYSTEM SHALL render
  every sub-view except `Export ↗`.
- WHERE the edition IS `enterprise` THE SYSTEM SHALL enable the
  `/people/roi/export` route and the Export ↗ sub-nav tab, gated
  by the `roi-admin` role.
- WHERE the context-injection log is absent THE SYSTEM SHALL render
  Knowledge reuse with an empty state and a hint to enable
  context-injection tracking.
- WHERE no adapter has reported `chat.cost` events in the active
  window THE SYSTEM SHALL render Hero API spend as `$0`, Net value
  equal to Dollars saved, and ROI multiple as `—` with a tooltip
  explaining that adapter cost reporting is not configured.
- IF a user attempts to view another user's My productivity AND
  that user has not opted in THEN THE SYSTEM SHALL render a privacy
  notice instead of metrics.
- IF an event source is unreachable (e.g. proposals store
  unavailable) THEN THE SYSTEM SHALL render the affected tile with
  an "unavailable" indicator and SHALL continue rendering the rest.
- IF the ROI multiple exceeds 999 in the active window THEN THE
  SYSTEM SHALL cap the displayed value at `999×`.
- THE SYSTEM SHALL render every sparkline, the donut, the
  segmented bar, the heatmap, and the 12-week trend chart as
  inline SVG emitted from a Go template helper without requiring a
  client-side chart library.
- THE SYSTEM SHALL render the methodology footnote on every ROI
  sub-view with the loaded hourly cost value inlined and links to
  `View methodology →` and `Tune coefficients →`.
- THE SYSTEM SHALL expose `/api/roi/methodology` returning a JSON
  document with the formula and coefficient for every metric.
- THE SYSTEM SHALL NOT render per-engineer dollar breakdowns; the
  contributor table's `$ saved` column SHALL be hours-share ×
  team `c_hourly_cost` only.
- THE SYSTEM SHALL register all ten sub-views with the shell view
  registry under `pack: engineering, home: people`.

## Risks

- **Coefficient credibility.** A manager who can't verify the
  number stops believing the surface. Defaults could be challenged
  as optimistic. Mitigation: every metric has a published formula;
  every coefficient is editable in `hero.json`; the methodology
  modal is one click from every ROI view; defaults err
  conservative; the Tune coefficients form is wired into the
  same surface.
- **Privacy regression in My productivity.** A leaked individual
  number turns Hero into a panopticon. Mitigation:
  `/people/roi/individual` is self-only by default; aggregate
  rollups anonymise non-opted-in users; default state for sharing
  is off; an explicit privacy test in `metrics_test.go` asserts
  non-opted-in users never appear by name in any rollup payload.
- **Demoralising scoreboarding.** Even with anonymisation, a
  visible ranked contributors table can feel like surveillance.
  Mitigation: framing matters — the section is titled "Top
  contributors (window)" not "Leaderboard"; mixes humans and
  agents in the same list; opt-in users can hide their bar from
  the chart while remaining counted in totals.
- **Metric drift across editions.** The same metric computed on a
  solo machine vs a team server vs a cloud rollup must produce
  comparable numbers. Mitigation: the `internal/serve/metrics/`
  package is the single source of truth and runs identically in
  every edition; the data scope changes, the formula does not.
- **SSE storms on the activity feed.** A noisy events.log on a
  busy team could spam the Pulse and Activity views. Mitigation:
  server-side coalescing — at most one feed fragment update per
  500ms per subscriber.
- **Empty context-injection log.** Until the log accumulates
  data, Knowledge reuse is uninformative. Mitigation: explicit
  empty state with an actionable hint; the sub-view does not
  vanish.
- **Attribution ambiguity for agent work.** A proposal merged
  without edits credits the agent in the autonomy funnel; some
  teams may prefer to credit the reviewer. Mitigation: both
  attribution modes are available via a `roi.attribution` config
  switch (`agent` default, `reviewer` alternative); the
  methodology modal discloses the active mode.
- **Adapter cost reporting gaps.** If `chat.cost` events stop
  flowing mid-window, Net value and ROI multiple can swing
  artificially. Mitigation: render `$0` spend with the
  "adapter cost reporting not configured" hint when the window's
  spend record count is zero or anomalously low; surface this
  state in the What changed section.

## Validation

- **Manual: edition matrix.** Start the surface under each
  `HERO_EDITION` value (`local`, `team`, `cloud`, `enterprise`,
  `ce`). Verify the sub-nav row shows the expected tabs and every
  hidden route returns 404.
- **Manual: Money tab parity with the mock.** Open
  `/people/roi` and compare against
  `mockups/01-roi-overview.html` side by side: tile values, sub-
  lines, trend pills, segmented bar split, methodology footnote
  text, donut slice order and colors, contributors table layout,
  what-changed dot colors. Spacing, typography, and hero-blue
  accents must match.
- **Manual: methodology modal.** Open every ROI sub-view, click
  the methodology link, verify every metric and coefficient is
  rendered. Edit `c_hourly_cost` in `hero.json`, reload, verify
  the modal reflects the new value and the Dollars saved + Net
  value + ROI multiple tiles recompute.
- **Manual: opt-in privacy.** Configure two users — one opted
  in, one not. Open `/people/roi` as a third user. Verify the
  opted-in user appears by name in the contributors table and the
  non-opted-in user appears as *"anonymous IC"* but contributes
  to the team totals.
- **Manual: window selector.** Change the active window on
  `/people/roi` from `Last 4 weeks` to `Last 12 weeks`; verify
  every tile, the donut, the trend chart, the contributors table,
  and the URL update.
- **Manual: live feed.** Open `/people` in two browsers. Run
  `hero event decision_made "test"` in a terminal. Verify both
  browsers receive the event via SSE within 1s and that no more
  than two SSE fragments arrive even under burst event load.
- **Manual: cross-repo handoff.** Configure a peer; receive a
  handoff; verify it appears in `/people/handoffs` and the Accept
  button produces the expected peer state change.
- **Test: metric formulas.** `internal/serve/metrics/metrics_test.go`
  — fixture-based assertions for autonomy ratio, spec coverage,
  time-to-spec, re-review rate, knowledge reuse, hours saved,
  dollars saved, net value, ROI multiple. Each test pins the
  expected output for a hand-built event log + proposals store +
  chat.cost fixture.
- **Test: ROI multiple edge cases.** Zero API spend → `—`. ROI
  multiple ≥1000 → capped at `999×`. ROI multiple between 10 and
  999 → one decimal. ROI multiple < 10 → two decimals.
- **Test: privacy invariant.** Fixture with three users, two
  opted out. Assert that the `/api/roi/overview` payload contains
  the totals but no opted-out user appears in any name field.
- **Test: per-engineer dollar invariant.** Fixture asserts no
  individual loaded cost appears anywhere in any ROI payload;
  contributor `$ saved` values match `hours_share *
  c_hourly_cost`.
- **Test: edition gating.** `internal/serve/registry/registry_test.go`
  — for each edition, enumerate the visible sub-views and assert
  the expected set.
- **Test: SVG helpers.** Snapshot tests for `svgSparkline`,
  `svgDonut`, `svgSegBar`, `svgHistogram`, `svgHeatmap`,
  `svgBoxPlot` — deterministic output for fixture inputs.
- **Performance: ROI overview computation.** With a 30-day window
  over an events.log of 100k events and a proposals store of 5k
  proposals, `/api/roi/overview` must respond within 500ms warm
  and 2s cold on a developer laptop.

## Kickoff

```
You are picking up the People + ROI home delivery.

Read first, in this order:
1. .hero/planning/features/hero-people-and-roi-home/spec.md (this file)
2. .hero/planning/features/hero-people-and-roi-home/mockups/01-roi-overview.html
3. .hero/planning/features/hero-surface-shell/spec.md
4. .hero/planning/features/hero-surface-deployment-and-rendering/spec.md
5. .hero/planning/features/hero-chat-and-model/spec.md

The mockup is the visual source of truth. When the spec and the
mockup disagree, the mockup wins for layout, color, spacing, and
typography; the spec wins for routing, formulas, and gating.

Start order:
1. internal/serve/metrics/ — land hours_saved.go, dollars.go,
   roi.go with fixture tests first. The Money chain math has to be
   right before any pixel renders.
2. internal/serve/pages/people/svgcharts.go — the SVG helpers.
   Snapshot tests pin the output.
3. internal/serve/pages/people/roi/templates/overview.html — the
   headline view. Match the mockup pixel-for-pixel before moving
   on.
4. methodology footnote + modal — wire on the Overview, then
   include in every other ROI template.
5. Sub-nav row partial + remaining People templates.
6. ROI deep-dive templates (velocity, autonomy, knowledge,
   individual).
7. roi-chart.js island for the 12-week trend hover/zoom/switch.
8. feed-filters.js and window-picker.js islands.
9. Edition gating + view registry registrations + tests.
10. Export ↗ (enterprise only) last.

Hard rules:
- No client-side chart library. Inline SVG only.
- No per-engineer dollar breakdown. Anywhere.
- No fixed left rail, VS Code tabs, bottom verb strip, or right
  ambient panel. This is a scrolling web app.
- Coefficients live in hero.json under roi.coefficients. Never
  hardcode the loaded hourly cost in templates or computations.
- Every ROI view renders the methodology footnote.
- /people/roi/individual defaults to self-only opt-in.

Verify before claiming done:
- `hero spec lint hero-people-and-roi-home` passes.
- All acceptance criteria have at least one corresponding test or
  manual validation step exercised.
- `/people/roi` rendered in a browser matches
  mockups/01-roi-overview.html within reasonable visual tolerance.
- The Money chain values on the Overview tile are mathematically
  consistent: Net value = Dollars saved − Hero API spend; ROI
  multiple = Net value ÷ Hero API spend.
```
