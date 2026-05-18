---
title: Hero Surface Polish v2 — Per-Item Detail Routes, Filter Reconcile, CSS + Verb Cleanup
slug: hero-surface-polish-v2
type: feature
status: completed
tags: [serve, surface, polish, bug, routing, detail, web-app]
created: 2026-05-18
relations:
  - target: hero-surface-polish
    kind: parent
  - target: hero-surface-polish-v1
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

[hero-surface-polish-v1](../../../specs/hero-surface-polish-v1/spec.md)
fixed the 22 sub-nav 404s, the default-view bugs, the Now data
fetcher event vocabulary, and the Work firehose. A fresh triage on
2026-05-18 surfaced the next set:

### 1. Every per-item detail link is a dead end

The home pages render cards and rows that link to per-item detail
routes that were never registered. Click any:

- Knowledge entry card → `/knowledge/<slug>` → 404
- Work spec card → `/work/spec/<slug>` → 404
- Agents session row → `/agents/session/<id>` → 404
- People profile pill → `/people/profiles/<user>` → 404

Confirmed via curl: all four return HTTP 404. Browse and Roadmap
look populated but every click leads nowhere.

This is now the biggest visible UX gap on the surface — the home
pages set up an expectation of "browse → drill in" and the drill-in
is broken.

### 2. Work's filter UI is in limbo

V1 spec called for a small filter row above the Horizons grid
(`All types / Features / Bugs / Initiatives`, `All ages / Active
this week`). The v1 delivery added the data-fetcher hooks (and the
roadmap_test.go asserts them) but the rendered HTML has zero
`rm-filter` / `filter-tab` markers; only the OLD inert
`view-toolbar` element from the original Work delivery remains.

The user-visible state: the old `view-toolbar` chips are still
there with `href="#"` (do nothing), and no new filter row exists.
Need to either delete the old inert chips and add the new row, or
wire the old chips to the new filter behavior (one UI, not two).

### 3. `coming-soon` template carries inline styles

V1's shared `coming-soon` stub renders 12 sub-route stubs with
~5 distinct inline `style="…"` blocks per render. Should move to
shared CSS so the stubs respect future theme changes and don't
fight with home-owned styles.

Confirmed: every stubbed sub-route includes `style="border:1px
solid var(--rule); …"`, `style="display:flex; …"`, etc. inline.

### 4. Verb settlement: `spec.complete` is a dead branch

Inspection of `.hero/events.log` over the last ~7 days shows:

```
4 peer.call.invoked
4 peer.call.completed
3 delivery_complete
1 workspace.peer_id_minted
1 spec_created
1 files_modified
1 decision_made
```

**Zero `spec.complete` events.** V1 expanded the consumer code in
`pages/now/data/metrics.go` and `changes.go` to count both
`spec.complete` and `delivery_complete`. The `spec.complete` arm is
dead branch — no code path emits it. The canonical verb is
`delivery_complete`.

Cleanup is small: either drop the dead-branch consumer references
to `spec.complete`, OR confirm a future emitter is planned and
document why both are kept. Inspecting code: nothing in
`internal/feed/`, `internal/serve/api/work.go`, or anywhere else
emits `spec.complete`. Drop the references.

### 5. Chat input only on Now (carry-over from earlier polish)

The Now home embeds the shell's `chat-input` fragment in its Quick
launch section. The other four homes — Work, Knowledge, Agents,
People — do not. The ⌘K overlay works everywhere (top-nav pill),
but the inline `chat-input` is Now-only.

Spec'd as a follow-up after polish-v1; mock parity is fine (the
home mocks didn't show inline chat), but consistency-of-affordance
argues for embedding it in a small ambient position on every home
(e.g., a thin row at the bottom of the page above the footer, or
floating in the page header). Pick a placement and apply
uniformly.

## Goal

After this polish pass:

1. **Every per-item detail link returns 200.** Knowledge entries,
   Work specs, Agents sessions, People profiles all have a real
   detail route. Each renders the home's shell + sub-nav (with the
   parent home tab active) + a per-item detail body. Where the
   underlying data exists on disk (specs in `.hero/specs/` and
   `.hero/planning/`, knowledge entries in `.hero/knowledge/`),
   render the real content. Where it doesn't (sessions on a workspace
   with no live session store; profiles on a solo workspace), render
   a clearly-marked stub that includes the requested slug.
2. **Work has one filter UI.** The old inert `view-toolbar` is gone
   or wired; the new filter row drives the actual roadmap data
   fetcher.
3. **`coming-soon` stubs are styled via shared CSS.** Zero inline
   `style="…"` attributes in the rendered stub HTML.
4. **`spec.complete` references are gone from consumer code.**
   `delivery_complete` is the canonical verb; comments document why.
5. **The four non-Now home pages include the chat-input fragment** in
   a consistent position so ⌘K is not the only way to chat across
   the surface.

## Approach

### Fix 1: Per-item detail routes (P0)

Four new item routes, one per home. Pattern (Knowledge example):

```go
return r.RegisterHome(shell.Home{
    // ... existing fields ...
    Items: []shell.ItemRoute{
        // existing v1 sub-routes ...
        {Pattern: "GET /knowledge/{slug}", Render: renderEntryDetail(deps)},
    },
})
```

**Important**: route patterns with `{slug}` capture must NOT collide
with named sub-routes (`/knowledge/search`, `/knowledge/why`, etc.).
Go 1.22+ `http.ServeMux` patterns specifically registered (e.g.,
`GET /knowledge/search`) win over the wildcard pattern; verify with
a test that hitting a named sub-route doesn't fall through to the
wildcard handler.

#### Knowledge entry detail (`/knowledge/{slug}`)

`renderEntryDetail` loads the entry from `.hero/knowledge/notes/<slug>.md`
(or wherever the entry lives in the workspace — use existing
`internal/knowledge` helpers if available; otherwise read the
markdown directly from the workspace `.hero/knowledge/` tree).

Render:
- Page hero: eyebrow `hero · main · knowledge`, title = entry's
  `title` from frontmatter, subhead = entry's `type · status ·
  created date · last touched`.
- Inline action row: `Edit ↗` (links to file path), `Open Why →`
  (deep-link to `/knowledge/why?target=<slug>` if it exists),
  `Copy reference` (just a no-op chip for now).
- Body: rendered markdown of the entry's content (the body after
  the frontmatter). Use whatever markdown renderer is already in the
  codebase (`internal/spec/` or `internal/knowledge/` likely has
  one; grep for `goldmark` or `markdown.Render`).
- Footer card: relations from frontmatter (`relates-to`,
  `depends-on`, `supersedes`) as linked chips that route to other
  knowledge / spec slugs.

If the slug doesn't resolve, render a 404-style "Entry not found"
page using the existing 404 template (not a stub card — this is a
genuine "doesn't exist" situation).

#### Spec detail (`/work/spec/{slug}`)

Same pattern, sourcing from `.hero/specs/<slug>/spec.md` (completed)
or `.hero/planning/{features,bugs,initiatives}/<slug>/spec.md`
(in-flight). The existing `internal/spec` package has loaders.

Render:
- Page hero: eyebrow `hero · main · work`, title = spec's `title`,
  subhead = `<type> · <status> · <horizon>` plus a one-liner from
  the spec's Goal section if it's short.
- Inline action row varies by status (consider it nice-to-have):
  - `planning` → `Move to delivering ↗` (CLI hint), `Edit ↗`
  - `delivering` → `Verify ↗`, `Complete ↗`, `Drift check ↗`
  - `completed` → `Open file ↗`, `View on GitHub ↗` (if remote
    configured)
- Body: rendered markdown of the spec. The Acceptance Criteria
  section can render with EARS-classification chips already used in
  the spec lint output; if too much work, skip the chips and render
  raw markdown.
- Right side: mockup links (if `mockups/` dir exists), child specs
  (if initiative), parent / depends-on / relates-to relations as
  linked pills.

404 fallback if slug unknown.

#### Session detail (`/agents/session/{id}`)

Workspace has no live session ledger yet. Render a stub:
- Page hero with title = `Session <id>`, subhead = `No live session
  store connected — sessions render here once hero-code emits live
  events.`.
- Body: a single coming-soon card explaining what this view will
  render once the ledger is wired (live transcript, tool calls, cost
  ticker), with a link to the relevant section of the chat-and-model
  spec.

#### Profile detail (`/people/profiles/{user}`)

Workspace likely has no team identity store. Render a stub:
- Page hero with title = `<user>`, subhead = `No team identity store
  connected.`.
- Body: coming-soon card; mention this view will populate from the
  team-coordination subsystem once team edition is configured.

### Fix 2: Work filter UI reconcile

The old `view-toolbar` element comes from the original Work delivery
template — likely a `<div class="view-toolbar">` row with inert
`<a href="#">` chips at the top of the page. The new filter row that
v1 added was supposed to live above the Horizons grid but didn't
make it into the rendered output.

Pick one:

- **Option A (drop the old)**: remove `view-toolbar` from
  `pages/work/templates/page.html` (or wherever it's rendered).
  Render the new filter row from `roadmap.html` only when on the
  Horizons view. Other views (Kanban / Graph / Blocked) get no
  filter (they're separate routes with different concerns).
- **Option B (rewire the old)**: keep `view-toolbar` as the shell
  for the filter row, replace its inert `<a href="#">` chips with
  the type/age filters that drive the data fetcher.

**Pick Option A.** It's cleaner — the filter row is roadmap-
specific, lives next to the data it filters, doesn't pretend to
exist on views where it doesn't apply.

Concretely:
- Delete the `view-toolbar` block from `pages/work/templates/page.html`.
- Confirm `roadmap.html` includes the filter row markup
  (`<div class="rm-filters">`); if not, add it.
- The filter chip hrefs are query-param links:
  `<a href="/work?type=feature">Features</a>` etc.
- Active state styled with hero-blue underline (same idiom as
  metric strip / sub-nav).

### Fix 3: `coming-soon` CSS extraction

In `internal/serve/shell/templates/coming-soon.html`, replace every
inline `style="…"` with a class name. Add the corresponding rules
to `internal/serve/shell/static/shell.css` (or a new `coming-soon.css`
served alongside).

Suggested class names:
- `.cs-card` — outer card
- `.cs-header` — flex row for title + meta
- `.cs-title` — heading
- `.cs-meta` — eyebrow text
- `.cs-body` — body paragraph
- `.cs-link` — track-follow-up link

Verify by curling a stubbed sub-route after the change: zero
`style="…"` attributes in the rendered HTML.

### Fix 4: Drop `spec.complete` dead branches

Grep across `internal/serve/pages/now/data/` for `"spec.complete"`.
Hits in `metrics.go`, `changes.go`, and possibly tests.

For each consumer arm:
- Remove the `spec.complete` case from the event-type filter.
- Leave a one-line comment: `// canonical completion verb is delivery_complete; spec.complete was a draft that never landed`
- Drop the corresponding test cases that asserted `spec.complete`
  counted.

Other code paths that mention `spec.complete` (per the v1 grep):
- `internal/feed/feed.go` — check whether this is an emitter or
  another consumer. If consumer: clean up. If it's actually about
  to emit one day, document why and keep.
- `internal/serve/api/work.go` — check + clean accordingly.

### Fix 5: Chat input on the four non-Now homes

Add the shell's `chat-input` fragment to each home page in a
consistent position. Recommendation: **a thin ambient row at the
top of each home, just below the page hero**, with the input
collapsed-by-default and expanding on focus.

Concretely, in each home's `page.html` (Work, Knowledge, Agents,
People), add:

```html
{{ template "chat-input" (chatInputForHome .) }}
```

just after the page-hero render. Use the `inline` variant of the
chat-input fragment (40px tall — already in the shell's variant
catalog). Pass page context (home + active sub-nav slug) so the
chat is scoped to that home.

The empty-state notice (when no adapter) is NOT shown on these
homes — keep that to Now where it's the primary affordance. The
inline chat just goes disabled with a placeholder "Connect a chat
adapter to enable" when there's no adapter.

### What is OUT of scope for v2

- **Building real content for stub sub-routes** (Knowledge Search /
  Recent / Write; Agents Automations / Health / Credentials;
  People Activity / Handoffs / Profiles non-detail). Those are
  per-home spec work, not polish.
- **`cost_usd` payload on delivery events.** Needs producer-side
  changes in the runner. Defer to a separate spec when hero-code
  adapter wire-up lands.
- **Populating Pulse metric tiles.** Depends on team/presence
  pipeline that doesn't exist.
- **EARS chip rendering on the spec detail view** — nice-to-have;
  use raw markdown for v2.

## Changes

### Routing (Fix 1)

- `internal/serve/pages/knowledge/page.go` — registered `GET /knowledge/{slug}` item route → `renderEntryDetail`; added `buildEntryPageHero` and `renderHeroAndChat`.
- `internal/serve/pages/knowledge/data/entry.go` (new) — `LoadEntry(heroDir, slug)` walks `.hero/knowledge/` (typed subdirs + loose .md files), parses frontmatter + relations, renders markdown via `internal/serve/mdrender`.
- `internal/serve/pages/knowledge/templates/detail.html` (new) — detail body (action row + rendered markdown + relations footer).
- `internal/serve/pages/work/page.go` — registered `GET /work/spec/{slug}` item route → `renderSpecDetail`; added `buildSpecDetailPageHero`.
- `internal/serve/pages/work/data/spec.go` (new) — `LoadSpec(heroDir, slug)` searches `.hero/specs/`, `planning/features/`, `planning/bugs/`, `planning/initiatives/` and the three-file layout; reuses `internal/spec.ParseFile`/`ParseThreeFile`.
- `internal/serve/pages/work/templates/spec-detail.html` (new) — detail body (action row + rendered markdown + relations footer).
- `internal/serve/pages/agentspage/page.go` — registered `GET /agents/session/{id}` → coming-soon stub renderer that surfaces the requested id in the page hero + stub card.
- `internal/serve/pages/people/page.go` — registered `GET /people/profiles/{user}` → coming-soon stub renderer that surfaces the requested user.
- `internal/serve/mdrender/mdrender.go` (new) — tiny in-house markdown renderer (headings / lists / fenced + inline code / bold / italic / autolinks / safe-href links) so spec + knowledge body rendering needs no third-party dependency.

### Work filter (Fix 2)

- `internal/serve/pages/work/templates/page.html` — removed the `view-toolbar` include from the default Horizons view; updated doc comment.
- `internal/serve/pages/work/templates/roadmap.html` — replaced the inline-styled `roadmap-filter-row` with the canonical `rm-filters` class structure (`rm-filter-group`, `rm-filter-label`, `rm-filter-tab` with `.active`); same query-param hrefs as before.
- `internal/serve/shell/static/shell.css` — added `.rm-filters` / `.rm-filter-tab` rules with the hero-blue underline active idiom.
- `internal/serve/pages/work/page.go` — dropped the unused `Toolbar` field from `pageData`.

### Stub CSS (Fix 3)

- `internal/serve/shell/templates/coming-soon.html` — replaced every inline `style="…"` with class hooks (`cs-card`, `cs-header`, `cs-title`, `cs-meta`, `cs-body`, `cs-note`, `cs-link`).
- `internal/serve/shell/static/shell.css` — added the corresponding `.cs-*` rules.

### Verb cleanup (Fix 4)

- `internal/serve/pages/now/data/metrics.go` — `isDeliveryCompleteEvent` and `isDeliveryEvent` no longer accept `spec.complete`; comment documents why.
- `internal/serve/pages/now/data/changes.go` — `kindFromEventType` and `renderEventText` drop the `spec.complete` arm.
- `internal/serve/pages/now/data/agents.go` — `shortenEventType` drops the `spec.complete` arm.
- `internal/serve/pages/now/data/metrics_test.go` — test renamed and seeded with `spec.complete` to assert it is NOT counted.
- `internal/serve/pages/now/data/changes_test.go` — case for `spec.complete` now expects the `pulse` default.
- `internal/serve/pages/now/data/agents_test.go` — case for `spec.complete` now expects the raw fall-through.

(Audit confirmed: no producer in `internal/feed/`, `internal/serve/api/work.go`, or anywhere else emits `spec.complete` — it was always only a consumer arm.)

### Chat input on other homes (Fix 5)

- `internal/serve/pages/knowledge/page.go` — added `chatInputFor(activeSlug)` + `renderHeroAndChat` helpers; every content closure (Browse, Why, Staleness, Stub, EntryDetail) now renders `chat-input` immediately after `page-hero`.
- `internal/serve/pages/work/page.go` — same helpers; applied in every content closure (Horizons, Blocked, Stub, SpecDetail).
- `internal/serve/pages/agentspage/page.go` — same helpers; centralized via `buildPageWith` + the new SessionDetail handler.
- `internal/serve/pages/people/page.go` — same helpers; applied in Pulse, ROI overview, Stub, and ProfileDetail closures.

(The Now home already renders `chat-input` via its `quicklaunch.html`; no template changes needed there.)

### Tests

- `internal/serve/mdrender/mdrender_test.go` (new) — covers headings, lists, fenced code, inline code, javascript: link sanitization, raw-HTML escaping, autolinks.
- `internal/serve/pages/knowledge/detail_test.go` (new) — EntryDetail 200 / 404 cases, slug-vs-named-route collision (seeding `search`/`why` slugs that collide with sub-routes), inline chat-input render.
- `internal/serve/pages/work/detail_test.go` (new) — SpecDetail 200 from `.hero/specs/` and `.hero/planning/features/`, 404, sibling sub-route collision, rm-filters-only-no-view-toolbar, inline chat-input render.
- `internal/serve/pages/agentspage/detail_test.go` (new) — SessionDetail stub render, sibling sub-route collision, chat-input render.
- `internal/serve/pages/people/detail_test.go` (new) — ProfileDetail stub render, sibling sub-route collision, chat-input render.
- `internal/serve/shell/coming_soon_test.go` (new) — guards Fix 3: the rendered `coming-soon` fragment contains zero `style="…"` and includes every `.cs-*` class hook.
- `internal/serve/pages/work/page_test.go` — updated `TestRegister_RendersAllSections` to assert `rm-filters` is present and `view-toolbar`/`work-toolbar` are absent on the default `/work` view.

## Boundaries

- **No new home content** beyond what's needed to wire the detail
  routes.
- **No new event types or emitters.** Verb cleanup is purely
  consumer-side.
- **No data-source new builds.** Reuse existing spec + knowledge
  loaders.
- **No design pivot** on the chat-input placement — pick the
  thin-row-below-page-hero pattern; iterate later if it feels
  wrong.

## Acceptance Criteria

- WHEN the user clicks ANY knowledge entry link from the Browse
  view THE SYSTEM SHALL return HTTP 200 and render the entry detail
  view with the entry's rendered markdown content.
- WHEN the user clicks ANY spec card on the Work Horizons view THE
  SYSTEM SHALL return HTTP 200 and render the spec detail view with
  the spec's rendered markdown content.
- WHEN the user requests `/agents/session/<id>` THE SYSTEM SHALL
  return HTTP 200 and render a clearly-marked coming-soon stub
  including the requested id in the page title.
- WHEN the user requests `/people/profiles/<user>` THE SYSTEM SHALL
  return HTTP 200 and render a coming-soon stub including the
  requested user in the page title.
- WHEN the user requests a per-item detail route with a slug that
  does NOT exist THE SYSTEM SHALL return HTTP 404 with the existing
  not-found template.
- WHEN the user requests `/knowledge/search` (a named sub-route)
  THE SYSTEM SHALL invoke the search renderer, NOT the slug detail
  renderer. (Same for every other named sub-route.)
- WHEN the user opens `/work` THE SYSTEM SHALL render exactly one
  filter UI — the `rm-filters` row above the roadmap grid — and
  SHALL NOT render the legacy `view-toolbar` element.
- WHEN the user clicks a filter chip in the new row THE SYSTEM
  SHALL re-render the page with the filter applied via query param.
- THE SYSTEM SHALL render the `coming-soon` stub with zero inline
  `style="…"` attributes in the HTML.
- THE SYSTEM SHALL NOT count `spec.complete` events in any metric
  tile, changes feed, or activity computation.
- WHEN the user opens any of `/work`, `/knowledge`, `/agents`, or
  `/people` THE SYSTEM SHALL render the shell's chat-input fragment
  in a consistent position (below the page hero) using the inline
  variant.
- WHEN no chat adapter is connected THE SYSTEM SHALL render the
  inline chat-input on the four non-Now homes as disabled with a
  short placeholder, WITHOUT rendering the full empty-state notice
  (that stays on Now only).

## Risks

- **Slug routes can collide with sub-routes.** Go 1.22 `ServeMux`
  prefers specifically-registered patterns over wildcards, but the
  test must verify this — easy to mis-register and have
  `/knowledge/search` fall through to the slug handler.
- **Knowledge entry loader needs to walk the right paths.** Notes
  live under `.hero/knowledge/notes/`; conventions / decisions
  elsewhere. Use existing helpers if any; otherwise grep the
  workspace's `.hero/knowledge/` for the right naming convention.
- **Spec detail loader has three possible locations** (specs/,
  planning/features/, planning/bugs/, planning/initiatives/). Search
  in order; the first match wins.
- **Chat-input placement on other homes is a UX bet.** The mocks
  don't show it. If users find the row distracting, we hide it via
  a per-home opt-out flag in v3. For now, ship it everywhere with a
  single placement.
- **Markdown rendering varies in quality.** If the existing renderer
  doesn't handle code blocks / tables nicely, the spec detail view
  could look off. Acceptable for v2; render quality is a separate
  polish later.
- **Verb cleanup might miss a hidden emitter.** Grep before
  removing; if any path emits `spec.complete`, keep the consumer
  branch and add a note.

## Validation

- Manual: open `/knowledge`, click any entry — verify a real detail
  page with rendered markdown.
- Manual: open `/work`, click any spec card — verify spec detail.
- Manual: visit `/agents/session/fake-id` — verify coming-soon stub
  with the id visible.
- Manual: visit `/people/profiles/fake-user` — verify coming-soon
  stub with the user visible.
- Manual: open `/work` — verify exactly one filter UI (the new
  `rm-filters` row); no `view-toolbar` element.
- Manual: open `/knowledge/recent` (a stubbed route) — verify zero
  inline styles in the rendered HTML.
- Manual: open `/work`, `/knowledge`, `/agents`, `/people` — verify
  the chat-input fragment appears below the page hero on each.
- Tests per the Changes list (25-29).
- `go build ./... && go test ./...` must pass.

## Kickoff

**Status: delivered 2026-05-18.** All five fix classes shipped.

**What works today (verified live on port 7457):**
- Per-item detail routes return **200** for real slugs:
  `/knowledge/<slug>`, `/work/spec/<slug>`, `/agents/session/<id>`,
  `/people/profiles/<user>`. Unknown knowledge/spec slugs return
  **404**. Sessions/profiles always render a coming-soon stub with
  the requested id/user.
- Named sub-routes win over the `{slug}` capture (all 12 verified
  — `/knowledge/search`, `/work/kanban`, `/agents/proposals`, etc.
  still invoke the named handlers).
- Knowledge detail renders real markdown (loader + new
  `internal/serve/mdrender` package).
- Spec detail renders real markdown sourced from `.hero/specs/`
  first, then `.hero/planning/{features,bugs,initiatives}/`.
- Work filter UI is the new `rm-filters` row (9 markers); the
  `view-toolbar` markup is removed.
- `coming-soon` stubs render with **0 inline styles** (extracted
  to `.cs-*` CSS rules in shell.css).
- `spec.complete` is gone from all consumer code paths.
- Inline chat-input fragment now embedded on all 5 homes (Go-level
  injection via `chatInputFor` / `renderHeroAndChat` helpers so
  every sub-route + detail route gets it consistently).

**Pick up at: file v3 follow-ups** in
[hero-surface-polish](../../initiatives/hero-surface-polish/spec.md)
as they're discovered. Known carry-overs from this delivery:

- Inline chat-input on the four non-Now homes has no visual disabled
  state when no adapter is connected — looks identical regardless.
  v3 could thin-style it muted when `ChatRegistry` is unset.
- `internal/serve/mdrender` doesn't handle markdown tables,
  blockquotes, or nested lists. Expand or switch to goldmark when
  we hit a spec/note that needs them.
- Detail views (`/knowledge/{slug}`, `/work/spec/{slug}`) render no
  active sub-nav tab — there's no tab corresponding to "an
  individual entry/spec." Consider a breadcrumb or "Detail"
  pseudo-tab in v3.
- Dead CSS rule `.view-toolbar { … }` remains in `shell.css` after
  the markup was removed. Cosmetic; one-line cleanup.

The detail-route tests must include the slug-vs-named-route
collision case; easy to mis-register patterns.
