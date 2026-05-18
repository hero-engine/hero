---
title: Hero Work Home — Spec + Delivery Surface
type: feature
status: planning
tags: [serve, surface, work, home, specs, roadmap, kanban, drift, ci, web-app]
created: 2026-05-17
relations:
  - target: hero-surface-architecture
    kind: parent
  - target: hero-surface-shell
    kind: depends-on
  - target: hero-surface-deployment-and-rendering
    kind: depends-on
  - target: spec-drift-detection
    kind: relates-to
  - target: spec-prioritization
    kind: relates-to
  - target: sprint-planner
    kind: relates-to
  - target: sprint-from-tracker
    kind: relates-to
  - target: acceptance-criteria-graph
    kind: relates-to
  - target: impact-analysis
    kind: relates-to
horizon: now
---

## Context

The Work home is the spec-and-delivery slice of the Hero surface. It is
where engineers live during a working day: looking at the roadmap,
picking up the next thing, watching active specs make progress, and
spotting drift, CI failures, and blocked work before they snowball.

Hero serve is a **web app companion to the CLI** — you open it in a
browser tab to deep-dive across projects. The surface initiative
([hero-surface-architecture](../../initiatives/hero-surface-architecture/spec.md))
pins this down explicitly: the chrome is a thin top nav, content
scrolls inside a centered ~1200px column, and the layout grammar is
the same one [hero-now-home](../hero-now-home/spec.md) uses (page hero
+ tabbed metric strip + sectioned content + quiet footer). The shell
([hero-surface-shell](../hero-surface-shell/spec.md)) owns the top
nav, footer, and ⌘K search. The rendering decision
([hero-surface-deployment-and-rendering](../hero-surface-deployment-and-rendering/spec.md))
fixed Go templates + SSE for chrome, lists, and feeds, with
hand-rolled vanilla web component islands only where interactivity
genuinely needs them.

**An earlier draft of this spec described a desktop-app layout** (fixed
left rail, VS Code tab strip, bottom verb strip, 380px ambient panel)
and registered eight "view records" against a workspace shell. That
layout was rejected. Work is now a scrolling web app page with an
in-page view-toggle row, a centerpiece Now/Next/Later roadmap, and a
small set of supporting sections below. The mock at
[mockups/01-work-roadmap.html](mockups/01-work-roadmap.html) is the
source of truth.

The v1 dashboard ([hero-dashboard-ui](../../../specs/hero-dashboard-ui/spec.md))
contributed four pages — Overview, Board (kanban), Bugs, Spec detail —
that are absorbed here. Overview moves to Now; Knowledge and Search
move to their own homes / the shell.

Much of the data the Work home needs already exists. The specs index
walks `.hero/planning/` and `.hero/specs/`. Drift comes from the
existing `spec-drift-detection` pipeline. Sprint comes from the
`sprint-planner` + `sprint-from-tracker` machinery. CI/commit linkage
already lands via `hero claim` and commit-message spec slug
references. This spec composes those sources; it does not invent new
ones beyond `PUT /api/specs/:slug` for the spec edit round-trip
(deferred — see Boundaries).

## Goal

Replace the v1 dashboard's Board, Bugs, and Spec-detail pages with a
single scrolling **Work home** at `/work` that opens to a three-column
Now/Next/Later roadmap, exposes an in-page view toggle to flip to
Kanban / Graph / Blocked, and surfaces drift, contract coverage, CI,
and agent-in-flight signals inline on every spec card.

When this lands, an engineer opening `/work` sees within one second:

1. **A page hero** stating workspace state in one sentence (e.g. "31
   specs · 14 delivering · 4 blocked · Sprint 17 ends Wed") with
   primary `New spec` and ghost `Import from tracker` / `Plan sprint`
   actions.
2. **A tabbed metric strip** (This sprint / Throughput / Quality) with
   four tiles per tab — the same idiom Now uses.
3. **A view-toggle row** of text-link tabs (Horizons · Kanban · Graph
   · Blocked) plus filter chips (All types · All owners). Clicking a
   tab navigates to that view's route.
4. **The centerpiece: Now / Next / Later roadmap.** Three quiet
   columns of spec cards. Each card carries spec slug, type chip,
   human title, status / owner / methodology meta, a dual progress bar
   (acceptance criteria + contract coverage) for delivering specs, and
   signal chips (drift, CI, agent-in-flight pulse, proposal count).
   The active initiative gets a subtle hero-blue gradient strip and a
   mini progress list of its child specs.
5. **A Blocked section** below the roadmap — a flat GitHub/Linear-style
   list of specs that can't move, mirroring Now's inbox idiom.
6. **A Recently shipped section** — a flat timeline of the last
   handful of shipped specs.
7. **The shell's footer.**

Done means: the v1 `/board`, `/bugs`, and `/spec/:slug` templates are
deleted, the page at `/work` renders the layout above, and the three
sibling routes (`/work/kanban`, `/work/graph`, `/work/blocked`) swap
the centerpiece while leaving the hero, metric strip, view toggle, and
bottom sections in place.

## Approach

### The page is a scrolling document, not a workspace

Work is not a multi-pane app with its own chrome. It is a page inside
the shell. The shell owns the top nav and footer; everything between
is this spec. The layout grammar matches Now exactly so users moving
between homes don't relearn the page:

```
┌── shell top nav (Now Work Knowledge Agents People · ⌘K · avatar) ──┐
│                                                                     │
│  ── page hero ────────────────────────────────────────────────────  │
│  eyebrow: hero · main                                               │
│  Work                                                               │
│  31 specs · 14 delivering · 4 blocked · Sprint 17 ends Wed          │
│  [New spec]  Import from tracker · Plan sprint                      │
│                                                                     │
│  ── tabbed metric strip ─────────────────────────────────────────   │
│  This sprint · Throughput · Quality                  View all →     │
│  [tile] [tile] [tile] [tile]                                        │
│                                                                     │
│  ── view toolbar ────────────────────────────────────────────────   │
│  Horizons · Kanban · Graph · Blocked (4)        All types ▾ Owners ▾│
│                                                                     │
│  ── view content (default: Horizons → 3-column roadmap) ─────────   │
│  ┌─ Now ─────────┐ ┌─ Next ────────┐ ┌─ Later ───────┐             │
│  │ initiative    │ │ spec card     │ │ spec card     │             │
│  │ spec card     │ │ spec card     │ │ spec card     │             │
│  │ spec card     │ │ spec card     │ │ ...           │             │
│  │ ...           │ │ ...           │ │               │             │
│  └───────────────┘ └───────────────┘ └───────────────┘             │
│                                                                     │
│  ── Blocked ─────────────────────────────────  View all (4) →       │
│  ● slug · reason                              Open  Reassign        │
│  ● slug · reason                              Open  Snooze          │
│  ...                                                                │
│                                                                     │
│  ── Recently shipped ────────────────────────  View all →           │
│  2h  slug — summary · actor                                         │
│  1d  slug — summary · actor                                         │
│  ...                                                                │
│                                                                     │
│  ── shell footer ────────────────────────────────────────────────   │
└─────────────────────────────────────────────────────────────────────┘
```

Max content width: 1200px. Column gap on the roadmap: 24px. Cards
have quiet backgrounds (`--bg-soft`), no boxed borders. Card
typography, chips, dot colors, and animations are locked by the
mockup CSS at [mockups/01-work-roadmap.html](mockups/01-work-roadmap.html);
the implementation extracts those tokens into the engineering pack's
shared CSS, it does not invent new ones.

### Views are routes, not client-side panes

The view toggle navigates between four sibling routes:

| Toggle label | Route | Content swapped in place of the roadmap |
|---|---|---|
| Horizons (default) | `/work` | 3-column Now/Next/Later roadmap |
| Kanban | `/work/kanban` | 4-column status board (planning / in-review / delivering / completed) |
| Graph | `/work/graph` | Dependency graph (inline SVG island) |
| Blocked | `/work/blocked` | Full Blocked list (the bottom section, expanded) |

**Why separate routes, not client-side swapping:** URLs are
deep-linkable and shareable (a manager can drop `/work/blocked` in
chat); back/forward works out of the box; each route ships only the
HTML it needs; no SPA state management. The view toggle is rendered
identically on every route with the active tab pinned by handler. The
page hero, metric strip, Blocked section, and Recently shipped section
are shared partials present on every Work route — the only thing that
changes between routes is the centerpiece block between the view
toolbar and the Blocked section.

Each route renders a complete HTML document (no client hydration
required for navigation). SSE on each route is scoped to the data that
route shows; navigating swaps subscriptions naturally.

### Page hero

One line of eyebrow text (`hero · main` — workspace name + active
branch, served from the shell context), the page title `Work`, a
subhead summarizing workspace state in plain prose, and an inline
action row.

Subhead format: `<n> specs · <n> delivering · <n> blocked · <sprint
status>`. The blocked count is colored warn-amber when ≥ 1. Sprint
status reads `Sprint <n> ends <weekday>` while a sprint is active and
collapses to `No active sprint` when none is configured.

Actions:

- **`New spec`** (primary button) — opens the shell's command palette
  pre-filled with `/design`. The full spec-creation form UI is **out
  of scope** for this spec (see Boundaries).
- **`Import from tracker`** (ghost link) — triggers `hero import` for
  the configured tracker; success swaps the page hero subhead via SSE
  to reflect new counts.
- **`Plan sprint`** (ghost link) — links to the sprint planner
  surface (lives under `/work/sprint`, owned by `sprint-planner`).

### Tabbed metric strip — same idiom as Now

Three text-link tabs share a single 4-tile row. Switching tabs is a
client-side pane swap (the tab content is rendered server-side into
hidden DOM panes; the toggle just flips an `active` class). This is
the lightest possible interactive idiom — no island, no fetch, no
SSE for tab swapping itself.

Tab data:

| Tab | Tile 1 | Tile 2 | Tile 3 | Tile 4 |
|---|---|---|---|---|
| **This sprint** (default) | Sprint progress `9/14` with segmented bar (done / in-review / at risk) | Days remaining `2`, sub: `ends Wed May 20` | Specs at risk `2` (warn), mono sub: `tripwire-system / graph-conflict-detection` | Your slice `3/4` with progress bar, sub: `1 in-review` |
| **Throughput** | Specs shipped this week `5`, sub: `↑ 2 vs prior week` | Lead time `3.2d`, sub: `spec → ship · 7d avg` | Cycle time `4.1d`, sub: `claimed → shipped` | Flow efficiency `62%`, sub: `active vs wait time` |
| **Quality** | Drift detected `3` (warn), mono sub: drift'd spec slugs | Contract coverage avg `74%` with bar | Re-review rate `3.1%`, sub: `2 of 64 PRs · 14d` | CI pass rate `96%`, sub: `main · last 100 runs` |

Tile geometry, segmented bar, and progress bar are the same primitives
Now's metric strip uses; share the partials.

### View toolbar — text-link tabs + filter chips

Sits inside its own section directly above the centerpiece. Same
text-link tab idiom as the metric strip. Active route's tab has a
hero-blue bottom border. `Blocked` carries a `(4)` badge when there
are blocked specs.

Filter chips on the right (`All types ▾`, `All owners ▾`) open a small
dropdown each. The chip state is encoded in the URL query string
(`?type=feature&owner=ben`) so deep links preserve filtering. The
filters apply to the centerpiece of the active view — they do not
filter the bottom Blocked or Recently shipped sections.

### Horizons centerpiece — the 3-column roadmap

Three columns: **Now / Next / Later**. Each column has a small head
(uppercase label + count), and below it a stack of cards with 16px
vertical spacing. The Now column head includes a pulsing hero-blue dot
to signal the live column.

Columns are populated by the `horizon` frontmatter field on each spec
(`now` / `next` / `later`; quarter-suffixed values like `q3-2026`
group into Later). Initiatives appear in the column matching their
`horizon`; their child specs do **not** also appear standalone if they
share the parent's column — they appear nested inside the initiative
card's mini progress list.

#### Spec card anatomy

```
┌─────────────────────────────────────────────────────┐
│ slug-of-the-spec                       [TYPE-CHIP]  │
│ Human-readable title of the spec                    │
│ ● status  · owner-avatar name  · methodology·tag    │
│ CRITERIA  ▰▰▰▰▰▰▰▱▱▱▱▱▱▱  10 / 14                   │
│ CONTRACT  ▰▰▰▰▰▰▰▰▰▰▰▰▱▱  88%                       │
│ [drift · minor] [CI pass] [● claude-opus] [1 prop]  │
└─────────────────────────────────────────────────────┘
```

- **Slug** — mono, hero-blue, linked. Clicking the slug navigates to
  `/work/spec/<slug>` (the per-item route owned by this spec's "Spec
  detail" extension; see route table below).
- **Type chip** — `FEATURE` (hero-blue), `BUG` (red), `INITIATIVE`
  (purple), `DECISION` (teal). Pull from the spec frontmatter `type`.
- **Title** — the spec's `title` frontmatter, lightly weighted, 14px.
- **Meta row** — colored status dot + label, owner avatar + name (or
  dashed "unclaimed" placeholder), methodology tag (`methodology ·
  sprint`, `methodology · phased`, etc.) when the spec carries one.
- **Dual bars** — only rendered for specs with `status:
  delivering`/`in-review`. First bar: acceptance criteria progress
  (count of EARS-passing criteria / total). Second bar: contract
  coverage percentage (hero-blue, success-green when ≥ 80%). Bar data
  comes from `hero coverage --spec <slug>` (already exists) and the
  AC graph (`acceptance-criteria-graph`).
- **Signal chips** — only render when present. Order: drift, CI,
  agent-in-flight, proposals.
  - `drift · minor` (amber) / `drift · major` (red) when the
    `spec-drift-detection` pipeline reports findings.
  - `CI pass` (green) / `CI fail` (red) from the most recent CI run
    associated with the spec.
  - `● <agent-id>` (hero-blue with pulsing live dot) when an agent
    session is currently in flight against this spec.
  - `<n> proposals` when the proposals queue has unresolved items for
    this spec.

The card hover lifts the background by one tone and underlines the
slug. Clicking anywhere on the card (except links) routes to
`/work/spec/<slug>`.

#### Initiative card variant

Initiatives use the same card grid but with three differences:

1. White background + a 1px `--border-strong` outline + a 3px
   hero-blue gradient strip across the top.
2. Title weight bumped to 600 / 15px.
3. Below the meta row, a `rm-children` mini-list of the child specs:
   colored status dot + child slug (mono) + `N / M` AC progress per
   child. A `Expand initiative →` link at the bottom navigates to the
   initiative's detail route.

The only initiative card rendered as `Now` in the seed data is
`hero-surface-architecture`; later initiatives like
`hero-team-experience` render as quieter cards in Next or Later.

#### Card stack for the seed mock

This is what the page renders against the current workspace; the
templates and data fetchers must produce this content when the
underlying specs and signals exist.

**Now column (6 cards):**

1. `hero-surface-architecture` (initiative, delivering, 14/24 criteria,
   5 children listed: `hero-surface-shell` 12/12 done, `hero-now-home`
   8/10 delivering, `hero-work-home` 5/16 delivering, `hero-knowledge-home`
   0/14 planning, `hero-agents-home` 0/12 planning).
2. `per-feature-smoke-coverage` (feature, delivering, 10/14, 88%
   contract, agent `claude-opus` in flight, CI pass, 1 proposal).
3. `tripwire-system` (feature, at-risk in-review, 6/12, 42% contract,
   drift minor, CI pass).
4. `graph-conflict-detection` (feature, at-risk blocked, 3/9, drift
   major, CI fail).
5. `unified-search` (feature, delivering, unclaimed, 8/12, 79%
   contract, CI pass).
6. `scan-enrichment-unbounded-loop` (bug, planning, sev-major, drift
   minor, repro confirmed).

**Next column (5 cards):**

1. `hero-team-experience` (initiative, planning, 4 children,
   methodology · phased, quiet note "Kickoff after surface-architecture
   lands. Q3 target.").
2. `cross-repo-peering` (feature, in-review, 9/9 criteria, 96%
   contract, signal `PR #421`, CI pass).
3. `e2e-traversal` (feature, planning, unclaimed, quiet note "Depends
   on traversal-queries shipping.").
4. `master-ingest-restore` (feature, planning, quiet note "Picked up
   after scan-enrichment bug closes.").
5. `hero-landing-page` (feature, planning, quiet note "Copy review
   pending. heroengine.ai target.").

**Later column (4 cards):**

1. `monorepo-satellite-installs` (feature, backlog, `Q3-2026`).
2. `e2e-discovery` (feature, backlog, `Q3-2026`).
3. `spec-type-registry` (decision, backlog, awaiting-decision, quiet
   note "Owner: Ben. Decision doc draft started.").
4. `hero-workspace-not-self-describing` (bug, backlog, sev-minor,
   unclaimed).

The quiet note (`rm-quiet` in the mock CSS) replaces the dual bars on
backlog/planning cards where progress is not yet meaningful.

### Kanban centerpiece (`/work/kanban`)

Four columns: `planning`, `in-review`, `delivering`, `completed`.
Cards are the same `spec-card.html` partial as Horizons. Column heads
show count. No drag-and-drop in v1 — status transitions happen via
CLI (`hero spec status`) or via the spec detail route. The Kanban view
is **read-only** in v1; the explicit deferral is in Boundaries.

Filter chips above the board (the same chips as the view toolbar) are
the only interactive surface. Filtering is server-side: the route
reads query-string params and re-renders.

### Graph centerpiece (`/work/graph`)

Inline SVG dependency graph rendered by a single island
(`work-graph.js`). Nodes are specs, edges are `depends-on` / `parent`
/ `supersedes` / `relates-to` relations. Node click opens
`/work/spec/<slug>`. Drift status colors the node border (clean / minor
/ major). The graph is **read-only** — no edge editing, no node
dragging beyond layout panning.

Data comes from `GET /api/work/graph` returning nodes (slug, type,
status, drift, contract %) and edges (from, to, kind). Layout is
client-side force-directed; per-user node positions persist via `PUT
/api/work/graph/layout` (server-side session state per the shell
spec).

The island is the only real JS work in this spec. It is bounded by the
parent rendering decision: no general-purpose graph editor, click-
through only.

### Blocked centerpiece (`/work/blocked`)

The full version of the bottom Blocked section. Same row template as
the bottom-section partial, but with no row cap and with sort controls
(by age / severity / owner). Source: `hero blocked` exposed as `GET
/api/work/blocked`.

### Blocked bottom section (always present)

On every Work route, a `Blocked` section appears between the
centerpiece and Recently shipped, capped at 3 rows with a `View all
(N) →` link to `/work/blocked`. Row anatomy (matches Now's inbox idiom):

```
● slug-of-the-blocked-spec — one-line reason
  [awaiting-decision]  depends on other-slug  · 3d blocked
                                            Open   Reassign
```

- Dot: red for hard blockers, amber for soft.
- Mono slug on the summary line, ink-grey reason after an em-dash.
- Reason chip: colored by category (`awaiting-decision` teal,
  `unmet-dep` neutral, `missing-data` warn).
- Trailing actions: text-link `Open` (routes to the spec detail) plus
  a category-appropriate second verb (`Reassign`, `Reprioritize`,
  `Snooze`).

The section is omitted entirely when nothing is blocked. The section
header always reads `Blocked` followed by a quiet meta count.

### Recently shipped bottom section (always present)

Flat timeline list. 5-6 rows max. Each row: relative-time chip
(right-aligned 56px column), a small green check icon, then the row
text: spec slug (mono, hero-blue), em-dash, one-line summary
(bold-ish), trailing actor in muted ink. Source: graph queries for
specs that transitioned to `completed` in the trailing window (~7
days); ordered most-recent first.

The section header reads `Recently shipped` with a `View all →` link
to the future shipped archive route (not built in this spec; the link
target falls back to `/work?status=completed` until the archive lands).

### Per-spec detail route (`/work/spec/<slug>`)

The Work home owns the per-item detail route as a sibling of the four
view routes. The detail page is **out of scope as a polished spec**
for this delivery — this spec stops at a minimum-viable detail page
(rendered markdown body + frontmatter sidebar + the existing v1
`/spec/:slug` content folded into the new shell). A follow-on spec
(`hero-work-spec-detail`) will replace it with the rich, EARS-chipped,
activity-tabbed version. The minimum-viable page is included here so
card clicks have a working target; the rich detail page is **not**
required for this spec to ship.

### SSE updates

A single `/api/work/stream` SSE channel multiplexes Work-page updates.
Fragment-id events the page subscribes to on mount:

- `event: page-hero` — counts / sprint state changed.
- `event: metric-strip` — sprint progress, throughput, quality
  numbers changed.
- `event: roadmap-card-<slug>` — one card's state changed (status,
  drift, CI, agent activity, proposal count).
- `event: blocked-section` — blocked list changed (add / remove /
  reason update).
- `event: recently-shipped` — a spec transitioned to completed.

Publishers are the existing subsystems (drift detector, CI ingest,
proposals subsystem, status transition pipeline, sprint planner).
Clients subscribe on Work mount, unsubscribe on Work unmount, and
re-render only the fragment that changed.

### Methodology and edition variance

Methodology hooks (preset machinery owned by the shell spec):

- Default methodology (`agile`/`sprint`/`shape-up`/`continuous`) is
  read from `hero.json` `engineering.methodology`. The first metric
  tab label flips accordingly:
  - `sprint` → `This sprint` (default).
  - `shape-up` → `This cycle`.
  - `continuous` → `This week`.
  - `phased` → `This phase`.
- The Horizons column labels stay `Now / Next / Later` across
  methodologies. Cycle-table or phased layouts are **out of scope** for
  this spec.

Edition gating:

| Capability | `local` | `team` | `cloud` | `enterprise` |
|---|---|---|---|---|
| Roadmap, Kanban, Graph, Blocked, Recently shipped views | ✓ | ✓ | ✓ | ✓ |
| Owner avatars on cards | self only | all teammates | + cross-repo | + RBAC redaction |
| Blocked queue (bottom + `/work/blocked`) | self-owned blockers only | team-wide blockers | + cross-repo | + RBAC redaction |
| `Your slice` metric tile | shown | shown | shown | shown |
| Throughput tile values | self only | team-wide | + cross-repo | + RBAC |
| Quality tab signals | self repo | team repo | + cross-repo | + RBAC |
| Agent-in-flight signal on cards | self sessions | + teammates' sessions | + cross-repo | + audit-respecting |

Edition gating lives in the data fetchers (`internal/serve/pages/work/data/`).
Templates do not gate; they render whatever the fetcher returns.

## Changes

### Page handlers and routing

1. `internal/serve/pages/work/page.go` — new file. Registers four
   handlers under the shell's router:
   - `GET /work` → `horizons`
   - `GET /work/kanban` → `kanban`
   - `GET /work/graph` → `graph`
   - `GET /work/blocked` → `blocked`
   - `GET /work/spec/{slug}` → minimum-viable spec detail (see step 13)
   Each handler resolves the active edition + methodology, calls the
   matching data fetcher, and renders the shared page chrome (hero +
   metric strip + view toolbar + bottom sections) with the route-
   specific centerpiece partial.

2. `internal/serve/pages/work/routes_test.go` — handler tests covering
   route registration, query-param filters (`?type=`, `?owner=`),
   active-tab pinning per route, and edition gating.

### Templates

3. `internal/serve/pages/work/templates/work-page.html` — outer page
   layout. Includes the page hero partial, metric strip partial, view
   toolbar partial, a `{{ block "centerpiece" }}` slot, the Blocked
   bottom section partial, and the Recently shipped partial.

4. `internal/serve/pages/work/templates/page-hero.html` — page hero
   partial. Reads workspace name, branch, counts (specs, delivering,
   blocked), and sprint state from the handler context.

5. `internal/serve/pages/work/templates/view-toolbar.html` — view
   toggle tabs + filter chips. Receives active route key from handler;
   reads filter state from query-string.

6. `internal/serve/pages/work/templates/metric-strip.html` — outer
   strip with three tabs and pane wrappers. Includes the three pane
   partials below.

7. `internal/serve/pages/work/templates/metric-tiles-sprint.html`,
   `metric-tiles-throughput.html`, `metric-tiles-quality.html` —
   tile-row partials per tab.

8. `internal/serve/pages/work/templates/horizons.html` — Horizons
   centerpiece. Three columns of cards. Receives a `Horizons` struct
   with `Now []SpecCard`, `Next []SpecCard`, `Later []SpecCard`.

9. `internal/serve/pages/work/templates/spec-card.html` — shared
   partial for a single card. Renders standard + initiative variants
   driven by `Card.Kind`. Includes meta row, dual-bar block (rendered
   only when `Card.Status` is `delivering` or `in-review`), signal
   chips block, and (for initiatives) the `rm-children` mini progress
   list.

10. `internal/serve/pages/work/templates/kanban.html` — Kanban
    centerpiece. Four status columns, each rendering the same
    `spec-card.html` partial.

11. `internal/serve/pages/work/templates/graph.html` — Graph
    centerpiece. Renders the island mount point (`<div
    data-island="work-graph" data-src="/api/work/graph"></div>`) plus
    an SSR fallback `<noscript>` block listing dependency edges as a
    plain list (so the page degrades to a usable view without JS).

12. `internal/serve/pages/work/templates/blocked.html` — Blocked
    centerpiece (full list). Reuses `blocked-row.html`.

13. `internal/serve/pages/work/templates/spec-detail.html` — minimum-
    viable per-item detail page. Renders the spec frontmatter sidebar
    (slug, type, status, owner, priority, severity, tracker link) and
    the rendered markdown body. **No** inline editor, **no** EARS
    chips, **no** activity feed — those land in the follow-on
    `hero-work-spec-detail` spec. This page exists only so card clicks
    have a working target during this delivery.

14. `internal/serve/pages/work/templates/blocked-section.html` —
    bottom-section partial (capped at 3 rows + `View all` link). Same
    row template as the full Blocked view.

15. `internal/serve/pages/work/templates/blocked-row.html` — single
    blocked row partial (dot + summary + reason chip + actions).

16. `internal/serve/pages/work/templates/recently-shipped.html` —
    flat timeline list of recently shipped specs.

### CSS

17. `internal/serve/pages/work/static/work.css` — extract the mockup
    CSS at `mockups/01-work-roadmap.html` (the `:root` token block is
    already in the shell's shared stylesheet; this file owns only Work-
    specific classes: `.page-hero`, `.metric-*`, `.view-*`,
    `.roadmap*`, `.rm-*`, `.blocked-*`, `.feed-*`). Reuse shell CSS
    tokens — do not add new tokens here.

### Data layer (fetchers)

18. `internal/serve/pages/work/data/horizons.go` — composes the
    Horizons centerpiece struct. Walks the specs index, groups by
    `horizon` field, nests initiative children, and joins drift /
    contract coverage / CI / agent-in-flight signals onto each card.
    Edition-aware via the request context.

19. `internal/serve/pages/work/data/kanban.go` — same composition
    grouped by `status` instead of `horizon`. Honors `?type=` and
    `?owner=` filters.

20. `internal/serve/pages/work/data/graph.go` — graph node/edge
    payload for the Graph view. Edges from frontmatter relations.

21. `internal/serve/pages/work/data/blocked.go` — wraps the existing
    `hero blocked` traversal. Returns rows with reason categories.

22. `internal/serve/pages/work/data/recently_shipped.go` — graph
    query for status transitions to `completed` in the last 7 days.

23. `internal/serve/pages/work/data/metrics.go` — composes the three
    metric strip panes. Sprint pane reads from the sprint planner
    state; Throughput from the existing velocity calculation; Quality
    joins drift / contract coverage / CI subsystems.

24. `internal/serve/pages/work/data/signals.go` — shared helper that
    joins drift status, contract coverage %, CI status, agent-in-
    flight presence, and proposal count to a list of spec slugs in one
    pass. Used by `horizons.go` and `kanban.go` to keep card
    rendering cheap.

25. `internal/serve/pages/work/data/edition.go` — edition gating
    helpers (filter to self vs team vs cross-repo).

26. `internal/serve/pages/work/data/*_test.go` — unit tests per
    fetcher covering: empty workspace, fully populated workspace, each
    edition value, and graceful degradation when an upstream signal
    source is unavailable.

### API + SSE

27. `internal/serve/api/work.go` — new file. Endpoints:
    - `GET /api/work/graph` — returns nodes + edges JSON for the Graph
      island.
    - `PUT /api/work/graph/layout` — persists per-user node positions
      to session state.
    - `GET /api/work/blocked` — JSON form of the Blocked list
      (consumed by the bottom-section SSE fragment refresh).
    - `GET /api/work/stream` — SSE channel multiplexing the events
      listed in Approach §SSE.

### Island

28. `internal/serve/pages/work/islands/work-graph.js` — single
    hand-rolled vanilla web component `<hero-work-graph>`. Fetches
    `/api/work/graph` on mount, runs a force-directed layout pass,
    renders inline SVG nodes and edges, handles click-through to
    `/work/spec/<slug>`, and PUTs layout positions on drag end. No
    external libraries; if the layout pass gets unwieldy, the
    rendering decision permits embedding a small (~50KB) helper —
    justify in the PR.

### Deletions / migrations

29. **Delete** `internal/serve/ui/board.html` (v1 kanban),
    `internal/serve/ui/bugs.html` (v1 bug inventory), and
    `internal/serve/ui/spec.html` (v1 spec detail) — their content is
    now served by the new Work templates. The `//go:embed` directives
    in `internal/serve/embed.go` are updated to embed
    `internal/serve/pages/work/templates/` and
    `internal/serve/pages/work/static/` in their place.

30. **Remove** route registrations for `/board`, `/bugs`, and
    `/spec/:slug` from the v1 router; add HTTP 301 redirects from
    `/board` → `/work/kanban`, `/bugs` → `/work?type=bug`, and
    `/spec/:slug` → `/work/spec/:slug` so any saved links keep
    working.

### Documentation

31. Update `hero.json` schema notes to document the
    `engineering.methodology` field's effect on the first metric tab
    label (sprint / cycle / week / phase). Schema enforcement of the
    field lives with the shell spec; this spec only documents the
    Work-side rendering effect.

## Boundaries

- **Spec creation UI is out of scope.** `New spec` opens the shell
  command palette pre-armed with `/design`; building an inline form
  for creating a spec is a separate item.
- **Inline spec editing is out of scope.** The minimum-viable spec
  detail page (step 13) renders the spec read-only. Inline editing
  with an `spec-editor` island and `PUT /api/specs/:slug` is deferred
  to the follow-on `hero-work-spec-detail` spec.
- **Polished spec detail is out of scope.** The activity feed, EARS-
  chipped acceptance criteria, validation tab, and proposals queue
  on the detail page all belong to `hero-work-spec-detail`. This
  spec ships only the bare detail page needed for card click-through.
- **Kanban drag-and-drop is out of scope.** Status transitions
  remain CLI-driven in v1. A drag-and-drop island can land later if
  usage warrants.
- **Graph editing is out of scope.** The dependency graph is read-
  only; edges are edited by changing frontmatter relation fields, not
  by manipulating the SVG.
- **Sprint planner detail is out of scope.** The `Plan sprint` ghost
  link routes to the sprint planner surface owned by `sprint-planner`;
  this spec does not implement the planner UI.
- **Methodology preset variants (cycle table, phased columns) are out
  of scope.** The first-tab label flips per methodology; the Horizons
  column layout stays Now/Next/Later regardless.
- **Agents and automations belong to [hero-agents-home](../hero-agents-home/spec.md).**
  The card-level `agent-in-flight` signal links into Agents; the
  running session UI lives there.
- **People rollups and ROI charts belong to [hero-people-and-roi-home](../hero-people-and-roi-home/spec.md).**
  The metric strip is a quick read; deep velocity / cycle / autonomy
  charts live in People.
- **Knowledge browsing belongs to [hero-knowledge-home](../hero-knowledge-home/spec.md).**
- **The cold-start landing surface belongs to [hero-now-home](../hero-now-home/spec.md).**
  Work assumes the user already knows what they came to look at.
- **Search is at the shell level.** ⌘K federated search belongs to
  the shell spec. The Work view toolbar's filter chips are per-page
  filters, not search.
- **Mobile is out of scope.** Desktop-first; the mockup degrades to a
  single-column layout below 1000px width but is not optimized for
  touch.

## Acceptance Criteria

- WHEN the user opens `/work` THE SYSTEM SHALL render the Horizons
  centerpiece with three columns labeled `Now`, `Next`, `Later`.
- WHEN the user opens `/work` THE SYSTEM SHALL display the page hero
  with workspace name, branch, spec counts, and current sprint state.
- WHEN the user clicks the `Kanban` view-toggle tab THE SYSTEM SHALL
  navigate to `/work/kanban` and pin the `Kanban` tab as active.
- WHEN the user clicks the `Graph` view-toggle tab THE SYSTEM SHALL
  navigate to `/work/graph` and mount the `work-graph` island.
- WHEN the user clicks the `Blocked` view-toggle tab THE SYSTEM SHALL
  navigate to `/work/blocked` and render the full blocked list.
- WHEN a spec card is clicked THE SYSTEM SHALL navigate to
  `/work/spec/<slug>` while preserving the source view's query string.
- WHEN the `This sprint` tab is active THE SYSTEM SHALL render four
  tiles: sprint progress with segmented bar, days remaining, at-risk
  spec slugs, and the user's slice.
- WHEN the user clicks the `Throughput` metric tab THE SYSTEM SHALL
  swap the tile row to show shipped count, lead time, cycle time, and
  flow efficiency without a full-page reload.
- WHEN the user clicks the `Quality` metric tab THE SYSTEM SHALL swap
  the tile row to show drift count, contract coverage average, re-
  review rate, and CI pass rate.
- WHEN a spec's status transitions to `completed` THE SYSTEM SHALL
  push an SSE `recently-shipped` event so the bottom timeline updates
  without a reload.
- WHEN a spec's drift state changes THE SYSTEM SHALL push an SSE
  `roadmap-card-<slug>` event and the card's drift signal chip SHALL
  update in place.
- WHEN a spec's CI run completes THE SYSTEM SHALL update the card's
  CI signal chip via SSE without re-rendering the rest of the page.
- WHEN the user applies a filter chip (`?type=` or `?owner=`) THE
  SYSTEM SHALL re-render the active centerpiece scoped to that filter
  and preserve the filter in the URL.
- WHILE a spec has drift detected THE SYSTEM SHALL render a drift
  signal chip on its card colored by severity (amber for minor, red
  for major).
- WHILE an agent session is in flight against a spec THE SYSTEM SHALL
  render a pulsing live-dot agent chip on that spec's card.
- WHILE a spec is `delivering` or `in-review` THE SYSTEM SHALL render
  the acceptance-criteria and contract-coverage dual bars on its
  card.
- WHILE no sprint is configured THE SYSTEM SHALL replace the sprint
  state phrase in the page hero with `No active sprint` and the first
  metric tab tiles SHALL render an empty-state.
- IF the specs index cannot be loaded THEN THE SYSTEM SHALL render
  the page chrome with an inline error placeholder in the centerpiece
  and continue to render the metric strip and bottom sections.
- IF a spec referenced by the graph cannot be loaded THEN THE SYSTEM
  SHALL render a greyed-out placeholder node with an inline error
  chip rather than failing the whole graph render.
- IF the JavaScript for the Graph island fails to load THEN THE
  SYSTEM SHALL render the `<noscript>` fallback list of dependency
  edges so the route remains useful.
- WHERE `engineering.methodology` IS `shape-up` THE SYSTEM SHALL
  relabel the first metric tab from `This sprint` to `This cycle`.
- WHERE `engineering.methodology` IS `continuous` THE SYSTEM SHALL
  relabel the first metric tab from `This sprint` to `This week`.
- WHERE `HERO_EDITION` IS `team` OR higher THE SYSTEM SHALL include
  teammate-owned blockers in the Blocked section and teammates' agent
  sessions in the agent-in-flight chip.
- WHERE `HERO_EDITION` IS `cloud` OR higher THE SYSTEM SHALL include
  cross-repo specs in the Recently shipped timeline.
- THE SYSTEM SHALL render the legacy v1 routes `/board`, `/bugs`,
  and `/spec/:slug` as HTTP 301 redirects to their `/work/*`
  equivalents.
- THE SYSTEM SHALL render the same page hero, metric strip, view
  toolbar, Blocked section, and Recently shipped section across all
  four view routes; only the centerpiece varies between routes.
- THE SYSTEM SHALL collapse the Blocked bottom section entirely when
  no specs are blocked, and collapse Recently shipped entirely when
  no specs have shipped in the trailing window.

## Risks

- **Card density.** Six cards in the Now column, each carrying meta,
  dual bars, and up to four signal chips, can become visually noisy.
  Mitigation: signals are render-only-when-present; the quiet-note
  variant replaces bars on backlog cards; tile typography and spacing
  are locked to the mock and must not creep.
- **Signal join cost.** Joining drift / contract / CI / agent / proposals
  to every card requires hitting five subsystems. `signals.go` joins
  in one pass against a slug set; if even that is too expensive on a
  large workspace, cache per-render with a short (~5s) TTL.
- **SSE volume on a busy workspace.** Many small fragment updates per
  minute is fine; bursts of dozens per second are not. Coalesce on
  the publisher side per spec slug with a 250ms debounce, same bound
  Now uses.
- **Initiative nesting confusion.** If an initiative and its children
  both have `horizon: now`, the children render only inside the
  initiative's mini progress list, not standalone. Renderer must
  enforce this; tests assert no duplicate rendering.
- **Detail page is a placeholder.** Shipping a minimum-viable detail
  page risks regressions vs v1's `/spec/:slug` (which had richer
  rendering). Mitigation: the minimum-viable page folds in the v1
  content verbatim; the rich version lands in `hero-work-spec-detail`
  before any v1 user notices the gap.
- **Filter chip dropdowns are untyped.** `?type=` and `?owner=` are
  free-form query params; an unknown value should render the
  centerpiece empty, not 500. Tests cover this explicitly.
- **Graph island scope creep.** The Graph view is the first Work
  island that does real work. Explicit boundary: read-only, click-
  through only, no edge editing, no general-purpose graph editor.
  Justify any additional island UI against the parent rendering
  decision.
- **Methodology label drift.** The first metric tab label switches on
  `engineering.methodology`; if the underlying data fetcher does not
  also flip (e.g. shows sprint progress under `shape-up`), the label
  becomes a lie. The tile-row partial selection must key off the same
  methodology value as the label.
- **Mock-to-template fidelity.** The mockup CSS is the source of
  truth; mismatches between the extracted `work.css` and the mock
  produce subtle visual regressions. Validation requires a side-by-
  side visual diff against the mock.

## Validation

### Tests

- Handler tests covering each Work route — assert the right
  centerpiece template renders, the right view-toggle tab is active,
  and query-string filters are honored.
- Data fetcher tests in `data/` per file — assert empty workspace,
  populated workspace, each `HERO_EDITION` value, and graceful
  degradation when an upstream subsystem (drift, CI, sprint) is
  unavailable.
- SSE channel test — publish each fragment event type, assert the
  client receives correct `event:` name and payload.
- Redirect tests for the three legacy routes (`/board`, `/bugs`,
  `/spec/:slug`).
- Filter test: `GET /work?type=bug` returns only bug-type cards in
  the centerpiece; the bottom sections are unaffected.
- Edition gating tests: under `local`, Blocked excludes teammate-
  owned rows; under `team`, includes them; under `cloud`, Recently
  shipped includes cross-repo specs.
- Graph render test: nodes carry drift status, edges carry kinds,
  unresolvable referenced specs render as placeholder nodes with
  error chips.
- Methodology label test: cycle through `sprint` / `shape-up` /
  `continuous` / `phased` and assert the first metric tab label
  matches.

### Manual

- Open `/work` on a workspace with the seed-mock spec set; verify the
  centerpiece, page hero, metric strip, bottom Blocked, and Recently
  shipped all render with the expected content within one second of
  first paint.
- Click each view-toggle tab; verify the route changes, the
  centerpiece swaps, and the active tab indicator moves. Verify the
  page hero, metric strip, and bottom sections remain identical.
- Click the `Throughput` and `Quality` metric tabs; verify tile-row
  swap is instant (no fetch) and the active tab indicator moves.
- Trigger a drift event externally (`hero check --drift` on a known
  spec); verify the corresponding card's drift chip updates without
  a reload.
- Trigger an agent session start against a spec; verify the agent-in-
  flight chip appears with the pulsing dot.
- Apply `?type=bug` to `/work`; verify only bug cards render in the
  centerpiece.
- Open `/work/graph`; verify the SVG graph renders, nodes are
  clickable, and click-through navigates to the spec detail.
- Disable JS in the browser; verify `/work/graph` falls back to the
  `<noscript>` edge list and the rest of `/work` still renders.
- Visit `/board`, `/bugs`, and `/spec/some-slug` from a saved link;
  verify each 301-redirects to the matching `/work/*` route.
- Flip `engineering.methodology` from `sprint` to `shape-up`; verify
  the first metric tab label switches to `This cycle`.
- Flip `HERO_EDITION` from `local` to `team`; verify teammate-owned
  blockers appear in the Blocked section.
- Visually compare `/work` against
  [mockups/01-work-roadmap.html](mockups/01-work-roadmap.html) side
  by side; verify column widths, card density, chip colors, dual
  bars, signal chip styling, and typography all match.

### Lints / smoke

- Run `hero spec lint hero-work-home`; freeform ratio is expected to
  be low — most criteria are WHEN/WHILE/IF/WHERE/THE shaped.
- After delivery, the per-feature smoke suite (`per-feature-smoke-coverage`)
  runs a headless visit of each Work route and asserts the
  centerpiece, page hero, metric strip, and bottom sections all
  appear.

## Kickoff

```
Deliver hero-work-home. Spec at
.hero/planning/features/hero-work-home/spec.md. Mock at
.hero/planning/features/hero-work-home/mockups/01-work-roadmap.html
is the visual source of truth — extract `work.css` from it; do not
invent new tokens.

Scope: replace the v1 dashboard's /board, /bugs, and /spec/:slug
pages with a scrolling Work home at /work with three sibling routes
(/work/kanban, /work/graph, /work/blocked) and a minimum-viable
detail page at /work/spec/:slug. Five shared sections render on
every route: page hero, tabbed metric strip (This sprint /
Throughput / Quality), view toolbar, Blocked, Recently shipped. Only
the centerpiece varies between routes.

Build under internal/serve/pages/work/. Handlers in page.go,
templates under templates/, data fetchers under data/, the single
Graph island under islands/work-graph.js, API + SSE under
internal/serve/api/work.go. Delete v1's board.html, bugs.html,
spec.html templates and update //go:embed directives. Wire 301
redirects from the legacy routes.

Do NOT build: spec creation UI, inline spec editing, rich spec
detail (EARS chips / activity feed / proposals — those land in the
follow-on hero-work-spec-detail spec), Kanban drag-and-drop, graph
editing, sprint planner UI, methodology preset layout variants
beyond the metric-tab label flip.

Acceptance criteria in the spec are the test plan. Use the seed
spec slug list from the spec's Horizons centerpiece section as
fixture data for handler / fetcher tests. Validate against the
mock side-by-side before calling it done.
```
