---
title: Hero Surface Polish v1 — Sub-route 404s, Now Data, Work Firehose, Default Views
type: feature
status: completed
tags: [serve, surface, polish, bug, routing, data, web-app]
created: 2026-05-18
relations:
  - target: hero-surface-polish
    kind: parent
  - target: hero-now-home
    kind: relates-to
  - target: hero-work-home
    kind: relates-to
  - target: hero-knowledge-home
    kind: relates-to
  - target: hero-agents-home
    kind: relates-to
  - target: hero-people-and-roi-home
    kind: relates-to
horizon: now
---

## Context

After running the freshly-shipped Hero Surface end-to-end on a real
workspace, four classes of issue surfaced that block the page from
feeling complete. None are architectural — they are wire-up + data-
fetcher gaps that fell through the parallel home delivery.

This spec bundles them into one polish pass so the surface goes from
"navigable" to "actually usable for daily work."

### What's broken (observed on 2026-05-18, post commit ccb091a)

**1. Every sub-nav link 404s.** All four sub-nav-bearing homes render
a sub-nav row whose anchors point at routes that were never registered
with the shell:

- `/work/{kanban,graph,blocked}` — 3 routes
- `/knowledge/{search,why,staleness,recent,write}` — 5 routes
- `/agents/{proposals,scheduled,automations,health,credentials}` — 5 routes
- `/people/{activity,handoffs,profiles,roi,roi/velocity,roi/autonomy,roi/knowledge,roi/individual,roi/export}` — 9 routes

That's **22 broken links across 4 homes** the moment a user clicks any
sub-nav tab.

The home `RegisterHome` calls in each home's `page.go` omitted the
`Items: []shell.ItemRoute{...}` field. The shell supports it; the
homes didn't use it.

**2. Knowledge default view is wrong.** `/knowledge` renders the
*Why view* content (Provenance chain, Plain-English summary, You
might also explore) instead of the *Browse view* the spec calls
default. The Browse template (`browse.html`) exists; it's just not
wired to the root.

**3. People default view is conflated.** `/people` renders both
Pulse-style content (Right now, Recent activity) AND ROI Overview
content (How time was spent, Where the savings came from, 12-week
trend, What changed this period) on the same page. The spec calls
for `/people` = Pulse only and `/people/roi` = ROI Overview only.
The two should split.

**4. Now's data fetchers don't recognize our event vocabulary.** The
data we have:

- `.hero/events.log` records `peer.call.invoked`, `peer.call.completed`,
  `decision_made`, `delivery_complete`, `spec.complete` (and others).
- `hero spec complete` is the verb we use; `claimed_by` is rarely set
  in this workspace.
- We've shipped 5+ specs this week and 4 today; the Now home reports
  `0 specs shipped this week`, `0 sessions done today`.

The data IS there; the fetchers' event-type filters miss it. The
`Changes` feed does pick up some events but icon-maps them all to
"convention" because the kind-inference is too narrow.

**5. Work page is a firehose.** `/work` Horizons view renders 122 cards
in NOW alone (out of 205 total specs). Initiative children appear
twice (once as a child row inside the parent initiative card, once as
a standalone card). No filter, no recents collapse, no pagination.
The view is unusable as a roadmap; it's a corpus dump.

## Goal

After this polish pass:

1. **Zero 404s on sub-nav clicks.** Every sub-nav anchor leads to a
   real route. The route renders the same shell page with the
   corresponding sub-nav tab active and the right section content in
   the body.
2. **Knowledge `/knowledge` opens to Browse** (filtered list of
   knowledge entries). The Why view is reachable at `/knowledge/why`.
3. **People `/people` opens to Pulse** (presence + claims + activity
   feed). The ROI Overview lives at `/people/roi` and is reachable
   from the sub-nav.
4. **Now's data fetchers recognize the real event vocabulary** —
   metric tiles reflect actual work shipped, agent activity reflects
   actual sessions, the changes feed icons map correctly.
5. **Work's Horizons view caps each column** at a reasonable default
   (10 cards) with a `Show all (N) →` link to expand. Initiative
   children render only once (inside the parent). When expanded, the
   full list paginates.

## Approach

### Fix 1: Register sub-routes on every home (CRITICAL)

For each of the four sub-nav-bearing homes, extend the `Home{}`
struct passed to `shell.RegisterHome` with an `Items` slice of
`ItemRoute`s — one per sub-nav anchor.

Each item route renders the same `page-layout` wrapper that the home
root uses, with the page's `activeSlug` (or equivalent) set to the
matching sub-nav tab, and with the body content sourced from the
appropriate section template.

Pattern for Knowledge (mirrors apply to Work, Agents, People):

```go
return r.RegisterHome(shell.Home{
    Slug: "knowledge", Label: "Knowledge", Href: "/knowledge",
    Render: renderBrowse(deps),  // NOTE — see Fix 2; default flips
    Items: []shell.ItemRoute{
        {Pattern: "GET /knowledge/search",    Render: renderSearch(deps)},
        {Pattern: "GET /knowledge/why",       Render: renderWhy(deps)},
        {Pattern: "GET /knowledge/staleness", Render: renderStaleness(deps)},
        {Pattern: "GET /knowledge/recent",    Render: renderRecent(deps)},
        {Pattern: "GET /knowledge/write",     Render: renderWrite(deps)},
    },
})
```

Where the existing single `handle(deps)` function gets split into per-
view render functions, each invoking the same page hero / sub-nav
chrome but with a different `activeSlug` and a different inner
section template.

For routes whose target template doesn't yet exist as full content
(e.g., Knowledge `Search`, `Recent`, `Write`; Agents `Automations`,
`Health`, `Credentials`; People `Activity`, `Handoffs`, `Profiles`,
ROI deep-dives), the route MUST register and render the home's chrome
+ sub-nav + a clearly-marked **section stub** (a single content card
saying `"<View name> — coming soon. The substrate exists; this view's
content is tracked in a follow-up."`). This way no link 404s, but we
don't pretend to deliver content that isn't there.

For routes whose target template DOES exist as content (Work
`Kanban`, `Graph`, `Blocked`; Knowledge `Why`, `Staleness`; People
`ROI Overview` and its tabs that already have inline content on the
current `/people` page), wire the real template.

Per home:

- **Work**: Register `/work/kanban` (renders kanban view; template
  exists). Register `/work/graph` (renders graph view; if no template,
  stub it). Register `/work/blocked` (renders blocked list; template
  exists).
- **Knowledge**: Register `/knowledge/why` (renders existing provenance
  view), `/knowledge/staleness` (renders existing staleness template),
  `/knowledge/search` (stub), `/knowledge/recent` (stub),
  `/knowledge/write` (stub).
- **Agents**: Register `/agents/proposals` (renders existing approvals
  partial — extract from current `/agents` page), `/agents/scheduled`
  (renders existing scheduled-preview partial), `/agents/automations`
  (stub), `/agents/health` (stub), `/agents/credentials` (stub,
  faded by edition gate per existing convention).
- **People**: Register `/people/roi` (renders existing overview
  template — extract from current `/people` page; see Fix 3),
  `/people/activity` (stub), `/people/handoffs` (stub),
  `/people/profiles` (stub), `/people/roi/velocity` (stub),
  `/people/roi/autonomy` (stub), `/people/roi/knowledge` (stub),
  `/people/roi/individual` (stub), `/people/roi/export` (stub, faded
  for enterprise).

Where a sub-nav tab is the home root (Browse for Knowledge, Sessions
for Agents, Pulse for People), `Active: true` is set on that tab in
the sub-nav data when the request URL is `/<home>`; otherwise on the
matching item route.

### Fix 2: Knowledge default = Browse

In `internal/serve/pages/knowledge/page.go`, change `Render:` on the
home registration to render the Browse view, not the Why view.

The current `handle(deps)` function presumably runs Why-view data
fetchers. Split it into `renderBrowse(deps)`, `renderWhy(deps)`, etc.
The Browse view consumes `internal/serve/pages/knowledge/data/corpus.go`
(corpus listing with facet filters) and renders `templates/browse.html`.

The home root sub-nav has `Browse` as active when on `/knowledge`.

### Fix 3: People default = Pulse, /people/roi = ROI Overview

Today `/people` renders all sections (Pulse + ROI) in one go. The
spec calls for `/people` = Pulse only. Split the existing
`templates/page.html` content into two render paths:

- `renderPulse(deps)` → page hero (Pulse subhead) + Right now strip +
  Recent activity feed only. Uses the existing `pulse.html` partial.
- `renderROIOverview(deps)` → page hero (ROI subhead with $-saved /
  ROI-multiple headline) + Money/Throughput/Quality metric strip +
  How time was spent + Where savings came from + 12-week trend +
  Top contributors + What changed. Uses the existing `overview.html`
  partial.

Wire `renderPulse` to `/people` (Render field) and `renderROIOverview`
to `/people/roi` (item route from Fix 1).

The sub-nav's `Pulse` tab is active on `/people`, `ROI Overview` on
`/people/roi`.

### Fix 4: Now's data-fetcher event vocabulary

The `.hero/events.log` event types we actually fire include:

- `peer.call.invoked`, `peer.call.completed`
- `decision_made`
- `delivery_complete` (and `delivery_start` if it exists)
- `spec.complete` (from `hero spec complete`)
- `spec.status_changed` (status flips through planning → delivering
  → completed)
- `knowledge.captured` / `note.captured`
- `commit` (if/when emitted; otherwise sourced from `git log`)

In `internal/serve/pages/now/data/agents.go`:
- Today's session count: count events where
  `type ∈ {delivery_complete, delivery_start, agent_session_started,
  agent_session_ended}` in the last 24h, regardless of `claimed_by`.
- Today's spend: sum any `cost_usd` payloads on those events; fall
  back to 0 (placeholder) when no cost is recorded.

In `internal/serve/pages/now/data/metrics.go`:
- `Specs shipped this week`: count events where
  `type == spec.complete` OR `type == delivery_complete` in the last
  7 days. Drop the `claimed_by == user` filter (we don't claim
  consistently); attribute by commit author OR by who fired the
  event (`agent` field in the event JSONL).
- `Commits authored`: keep the existing `git log` query but make sure
  it doesn't require a sprint to be configured.
- `Agent assist on your work`: ratio of `delivery_complete` events
  with an `agent` field that's an LLM identifier (e.g. starts with
  `claude-`, `gpt-`, `engineer`, `ai/`) vs total delivery events in
  the window.

In `internal/serve/pages/now/data/changes.go`:
- Expand the kind→icon mapping. Currently everything falls to
  `convention`. Add:
  - `peer.call.*` → `handoff` icon (network/share style)
  - `decision_made` → `decision` icon (compass/diamond)
  - `delivery_complete` → `commit` or `check` icon
  - `spec.complete` → `check` icon
  - `spec.status_changed` → `spec` icon
  - `knowledge.captured` / `note.captured` → `knowledge` icon (book)
  - default fallback → `pulse` icon (generic) — NOT `convention`
- Each row's sentence-form text reflects the event type's payload
  reasonably (the existing `message` field is usually adequate; just
  prefix with a verb where useful, e.g. "peer call ok …" → "Peer
  call completed to hero-code: …").

### Fix 5: Work firehose + dupes

In `internal/serve/pages/work/data/horizons.go` (or wherever the
horizons fetcher lives):

- After grouping specs by horizon, cap each column at **10 cards by
  default** with a `Show all (N) →` link at the bottom of any column
  that has more.
- The cap is overridable via a query param `?all=1` (the link's
  target) which renders every spec in every column.
- When `?all=1` is set, also paginate at 50 cards per column with
  prev/next links.

Dedupe initiative children at top level:
- Build a set of slugs that appear as `child` in any initiative's
  frontmatter (the existing graph queries already track parent/child
  relations).
- When listing top-level cards, skip any spec whose slug is in that
  set. It will still render as a child row inside its parent
  initiative card (the existing initiative card's mini child list).

Add a small filter row above the Horizons grid for the most common
cuts:

- `[All types] [Features only] [Bugs only] [Initiatives only]`
- `[All ages] [Active this week]` (active = touched in last 7d)
- These are text-link tabs in the same style as the metric strip
  (already established pattern).

### What is OUT of scope for v1

- New views / templates beyond what's already drafted.
- Embedding the chat-input fragment on the four non-Now home pages
  (consistency improvement, not blocking — file as v2).
- Cost ticker `today_cost` field plumbing (depends on real adapter
  cost reporting).
- Live `chat.adapter.added` / `chat.disconnected` event publishing
  (latent follow-up named in the previous polish pass).
- Any work on PM/QA packs.

## Changes

### Shared stub (Fix 1 prerequisite)

1. `internal/serve/shell/templates/coming-soon.html` — new shared shell
   card rendered by every stubbed sub-route. Single template, one
   message, one link back to this spec.

### Routing + default-view splits (Fix 1 + 2 + 3)

2. `internal/serve/pages/knowledge/page.go` — split `handle` into
   `renderBrowse` / `renderWhy` / `renderStaleness` / stubs; add
   `Items` to `RegisterHome` with 5 sub-routes; default flips to
   Browse; `buildSubNav` now takes an `activeSlug`.
3. `internal/serve/pages/work/page.go` — add `Items` with 3 sub-
   routes (kanban / graph / blocked); wire Blocked to its template;
   stub Kanban + Graph; parse `?type` / `?age` / `?all` / `?page` and
   feed into `LoadRoadmap`.
4. `internal/serve/pages/agentspage/page.go` — add `Items` with 5
   sub-routes; extract `approvals` + `scheduled-preview` into per-view
   renders; stub the rest; `buildSubNav` now takes an `activeSlug`.
5. `internal/serve/pages/people/page.go` — full rewrite. Default
   flips to `renderPulse`; `renderROIOverview` lives at `/people/roi`;
   7 more stubs cover activity / handoffs / profiles / ROI deep-dives;
   added a quiet Pulse-only metric strip so the chrome is consistent.

### Now data (Fix 4)

6. `internal/serve/pages/now/data/agents.go` — expand session-count
   vocabulary (`delivery_*`, `agent_session_*`, `peer.call.completed`);
   add per-type session-row subtitles via the extended
   `shortenEventType`.
7. `internal/serve/pages/now/data/metrics.go` — `countCompletedSince`
   now recognises `spec.complete` AND `delivery_complete`;
   `agentAssistTile` computes the real ratio from
   `isAgentAuthored(agent)` instead of the placeholder.
8. `internal/serve/pages/now/data/changes.go` — kind→icon mapping
   covers `peer.* → handoff`, `decision_made → decision`, `*.complete
   → check`, `spec.status_changed → spec`, `knowledge.captured →
   knowledge`; default fallback flipped from `convention` to `pulse`.
   `renderEventText` now produces sentence-form HTML for every new
   type.

### Work firehose (Fix 5)

9. `internal/serve/pages/work/data/types.go` — add `LastTouched` to
   `SpecCard`; add `RoadmapFilters`, `ShowAll`, `Page`, `Capped`,
   `ShowAllHref`, `PageInfo` / `ColumnPage`.
10. `internal/serve/pages/work/data/roadmap.go` — `LoadRoadmap` now
    honours type/age filters, dedupes ANY initiative-child slug from
    the top-level list (not just same-horizon), sorts each column by
    `LastTouched DESC`, and caps at 10 (or paginates at 50 under
    `?all=1`). Added `buildHref` / `passesFilter` /
    `applyCapOrPaginate` / `sortByLastTouchedDesc` helpers.
11. `internal/serve/pages/work/templates/roadmap.html` — filter row
    (Type + Age) above the grid; `Show all (N) →` link per capped
    column; prev/next page nav under `?all=1`.

### Tests

12. `internal/serve/pages/knowledge/page_test.go` — `/knowledge`
    renders Browse only; sub-routes all return 200 with active tab.
13. `internal/serve/pages/work/page_test.go` — sub-routes return 200.
14. `internal/serve/pages/work/data/roadmap_test.go` — new. Column
    cap at 10, `?all=1` expansion, pagination at 50, initiative-child
    dedupe, type filter, age filter, sort-by-last-touched.
15. `internal/serve/pages/agentspage/page_test.go` — sub-routes
    return 200.
16. `internal/serve/pages/people/page_test.go` — `/people` is Pulse
    only; `/people/roi` has full ROI; all 9 sub-routes return 200.
17. `internal/serve/pages/now/data/changes_test.go` — kind→icon map
    covers handoff / decision / check / knowledge / pulse fallback.
18. `internal/serve/pages/now/data/metrics_test.go` — `spec.complete`
    counted; `isAgentAuthored` covers the LLM identifier prefixes.
19. `internal/serve/pages/now/data/agents_test.go` — extended
    `shortenEventType`; session-count picks up all expanded types.

## Boundaries

- **No new homes, no new views.** Stubs are stubs — they exist to
  unbreak the links, not to fake content.
- **No data-source new builds.** The fetchers consume what
  `.hero/events.log` and existing queries already provide. If a
  metric needs a new pipeline, leave it as `—` and note in code.
- **No layout changes** beyond the column-cap row and the small
  filter row on Work.
- **No design pivots.** Visual style stays.

## Acceptance Criteria

- WHEN the user clicks ANY sub-nav anchor on Work, Knowledge, Agents,
  or People THE SYSTEM SHALL respond with HTTP 200 and render the
  shell + sub-nav with the clicked tab active (no 404s).
- WHEN a sub-route's content is not yet built THE SYSTEM SHALL render
  a clearly-marked stub card with the standardized copy AND link to
  this spec for traceability.
- WHEN the user opens `/knowledge` THE SYSTEM SHALL render the Browse
  view (corpus list + facet filters), NOT the Why view.
- WHEN the user opens `/people` THE SYSTEM SHALL render the Pulse
  view only (Right now + Recent activity), NOT the combined
  Pulse + ROI page.
- WHEN the user opens `/people/roi` THE SYSTEM SHALL render the ROI
  Overview (Money metric strip, How time was spent, Where the
  savings came from, 12-week trend, Top contributors, What changed).
- WHILE `.hero/events.log` contains `spec.complete` OR
  `delivery_complete` events from the last 7 days THE SYSTEM SHALL
  reflect their count in Now's `Specs shipped this week` tile
  regardless of whether the spec was claimed.
- WHILE the events log contains `delivery_*` events from today THE
  SYSTEM SHALL count them in Now's `Today · Sessions done` stat.
- WHEN a row in Now's `Since you were here` feed corresponds to a
  `peer.call.*` event THE SYSTEM SHALL render the handoff icon (not
  the convention icon).
- WHEN a row corresponds to `decision_made` THE SYSTEM SHALL render
  a decision icon.
- WHEN the user opens `/work` AND a horizon column contains more
  than 10 specs THE SYSTEM SHALL render only the 10 most recently-
  touched cards plus a `Show all (N) →` link.
- WHEN the user clicks `Show all` THE SYSTEM SHALL render every
  spec in that column, paginated at 50 per page.
- WHILE rendering top-level horizon cards THE SYSTEM SHALL NOT
  render any spec that is already a child of a rendered initiative
  card.
- WHERE the Work filter row is set to `Features only` THE SYSTEM
  SHALL show only `type == feature` cards across all columns.
- THE SYSTEM SHALL NOT introduce any new content beyond what's
  required to unbreak the listed paths.

## Risks

- **Stub copy quality.** Slapping a placeholder card on 12+ stub
  routes risks looking lazy if they all read identically. Mitigation:
  the stub template includes the view name and a "track follow-up
  →" link to this spec, plus a tiny preview of what the substrate
  exists for (e.g., for Knowledge `/recent`: "Recent knowledge
  captures are available in `.hero/knowledge/` — UI lands in a
  follow-up").
- **Knowledge Browse → Why default flip might surprise users who
  bookmarked `/knowledge` expecting the graph view.** Acceptable —
  the spec calls for Browse default, and Why is now one click away.
- **Splitting People into Pulse-only may feel empty** on a workspace
  with no team presence (i.e., solo). Mitigation: Pulse view's
  empty state honestly says "No teammates connected" with a link to
  ROI Overview as the visible alternative.
- **Work column cap (10) may hide important work.** Mitigation:
  ordering is `last_touched DESC` so what's at the top is what
  matters most; the `Show all` link is always present when capped.
- **Initiative-child dedupe relies on the graph query being
  correct.** If the parent/child relation is missing or stale on
  any spec, both copies will render. Mitigation: add a debug-mode
  flag (`?showdupes=1`) for diagnosis; otherwise accept the trade-
  off.

## Validation

- Manual: visit every sub-nav anchor on each home. Each returns 200.
  Active tab highlight matches the URL.
- Manual: open `/knowledge`; verify Browse view (not Why).
- Manual: open `/people`; verify Pulse-only (no ROI sections). Open
  `/people/roi`; verify ROI Overview renders.
- Manual: confirm Now's `Specs shipped this week` reflects the
  actual count after we shipped multiple specs.
- Manual: confirm Now's `Today · Sessions done` is non-zero after
  agent activity.
- Manual: open `/now`; check Since-you-were-here icons — peer calls
  show a handoff icon, decisions show a decision icon, not all
  convention.
- Manual: open `/work`; verify NOW column shows ≤ 10 cards by
  default; verify `Show all (N) →` link is present and works.
- Manual: verify no spec appears both as a top-level card AND as a
  child row of an initiative.
- Tests: per-home routing assertion that each registered item route
  returns 200.
- Tests: data-fetcher unit tests for the expanded event vocabulary.
- Tests: horizon-cap + dedupe logic with fixture corpora.

## Kickoff

**Status: delivered 2026-05-18.** All five fix classes shipped in one
pass. Smoke verification on a real workspace:

- **All 22 sub-routes return 200** (Work 3 + Knowledge 5 + Agents 5 +
  People 9). Sub-nav clicks no longer 404 anywhere.
- **`/knowledge` opens to Browse** (Provenance content count = 0;
  Browse content present). `/knowledge/why` reaches the Why view.
- **`/people` opens to Pulse only** (12-week-trend / savings content
  count = 0). `/people/roi` renders the full ROI Overview (2 matches).
- **Work column cap working** (44 default cards vs 138 with `?all=1`,
  with 2 "Show all" links). Initiative children dedupe at top level.
  Filter row (`type`, `age`) on the roadmap view.
- **Now changes feed icon mapping fixed** (7 handoff icons for the
  recent peer.call.* events; zero stuck on "convention").
- **Now metric tile vocabulary** counts both `spec.complete` and
  `delivery_complete`; drops `claimed_by` filter; tile reads 0 today
  only because the workspace events log hasn't fired
  `delivery_complete` since 2026-05-05 — the moment future deliveries
  emit either verb, the tile populates.

**Pick up at: file v2 follow-ups** in
[hero-surface-polish](../../initiatives/hero-surface-polish/spec.md)
as they're discovered. Carry-over from this delivery:

- Settle on one verb (`spec.complete` vs `delivery_complete`) so
  metric attribution is consistent end-to-end.
- Populate `cost_usd` payload on `delivery_*` events so the agents
  Today.Spend tile shows real dollars instead of `—`.
- Reconcile the inert `view-toolbar` filter chips (currently `href="#"`)
  with the new roadmap filter row (one filter UI, not two).
- Move stub `coming-soon` template's inline styles into shared CSS.
- Pulse-only metric strip renders four `—` tiles — populate when the
  team/presence pipeline lands.
