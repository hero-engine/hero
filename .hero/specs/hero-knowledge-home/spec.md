---
title: Hero Knowledge Home — The Corpus, Visible
type: feature
status: completed
tags: [serve, surface, knowledge, home, search, traversal, corpus, web-app]
created: 2026-05-17
relations:
  - target: hero-surface-architecture
    kind: parent
  - target: hero-surface-shell
    kind: depends-on
  - target: hero-surface-deployment-and-rendering
    kind: depends-on
  - target: unified-search
    kind: consumes
  - target: unified-retrieval-layer
    kind: consumes
  - target: traversal-queries
    kind: consumes
  - target: knowledge-contradiction-detection
    kind: consumes
horizon: now
---

## Context

Hero's knowledge base is the moat: conventions captured from real work,
decisions with their reasoning intact, learnings persisted by `/capture`,
and the provenance edges that tie every line of code back to the spec →
decision → note → discussion that produced it. Today that moat is
mostly invisible from the surface. The legacy dashboard had a flat
"Knowledge" page that listed entries by directory — no facets, no
provenance, no staleness, no in-surface writer, no traversal.
`hero why` prints a gorgeous chain in the terminal that no one ever
sees in a browser tab.

Hero serve is now a **web app companion to the CLI** opened in a browser
tab — slim top nav, scrolling content, no fixed rails or VS Code tabs
(see [hero-surface-deployment-and-rendering](../hero-surface-deployment-and-rendering/spec.md)).
The previous Knowledge home draft assumed the older desktop-shell
grammar (fixed left rail, center tab strip, bottom verb strip, right
ambient panel). That grammar has been rejected for hero serve. The
**source of truth** for this home is the `01-knowledge-why.html` mock,
which renders the Why view as a single scrolling page with a slim
sub-nav row, a target-input page hero, and a two-column graph+detail
section inline in the content stream.

The substrates that make this home cheap to build have already landed
or are landing:

- [unified-search](../unified-search/spec.md) merges federation graph
  and on-disk index into a single ranked result set across knowledge,
  specs, commits, and code symbols.
- [unified-retrieval-layer](../unified-retrieval-layer/spec.md)
  exposes a single `internal/retrieval/` façade with BM25 ranking and
  type/tag/repo facets via `node_index`.
- [traversal-queries](../traversal-queries/spec.md) ships
  `hero why` (multi-hop origin chain) as a real recursive-CTE
  traversal returning structured JSON.
- [knowledge-contradiction-detection](../../../specs/knowledge-contradiction-detection/spec.md)
  produces the staleness/conflict signals stored in
  `.hero/knowledge/contradictions.json`.

What's missing is the **surface**. This spec is the visual home for
all of the above, plus a writer for entries that today only get
created by capture or by hand-editing markdown.

## Goal

A scrolling web-app page at `/knowledge` (and per-view siblings)
that renders the corpus as a first-class home in hero serve:

1. A slim **sub-nav row** of text-link tabs below the top nav —
   `Browse · Search · Why · Staleness <n> · Recent · Write` —
   matching the Knowledge mock.
2. Seven routes, each its own scrolling page: `/knowledge` (Browse,
   default), `/knowledge/search`, `/knowledge/why`,
   `/knowledge/staleness`, `/knowledge/recent`, `/knowledge/write`,
   and per-entry detail at `/knowledge/<slug>`.
3. The **Why view** is the signature: a target-input page hero, a
   compact metric strip, a 2-column graph + detail centerpiece, a
   plain-English summary, suggested neighbors, and a staleness flag.
   It is the visual home for `hero why`.
4. Search results federate across knowledge entries, specs, commits,
   and code symbols, grouped by kind, via `unified-search`.
5. Staleness lists contradictions and broken references with inline
   resolve actions; the count appears as a badge on the sub-nav tab.
6. Recent is a reverse-chronological feed of capture / edit /
   supersession events.
7. The Writer is the only knowledge writer in the surface — an
   island that edits markdown frontmatter + body and routes save
   through `/api/knowledge/entries`.

Everything else (contradiction detection itself, retrieval ranking,
chat dispatch, cross-repo write-back) lives in its consumed spec.
This home **renders** what those substrates produce.

## Approach

### Grammar

The Knowledge home obeys the web-app grammar locked by
[hero-surface-deployment-and-rendering](../hero-surface-deployment-and-rendering/spec.md):

- Slim top nav owned by [hero-surface-shell](../hero-surface-shell/spec.md)
  (Hero brand mark, top-nav tabs, ⌘K pill, avatar) — 56px tall,
  sticky, hero-blue underline on active.
- Knowledge-owned **sub-nav row** (~42px tall, also sticky to the
  underside of the top nav) renders the six in-home tabs.
- Scrolling content at `max-width: 1200px` centered, 32px horizontal
  padding, sections stacked with ~32–48px breathing room.
- No fixed left rail. No fixed right rail. No fixed bottom strip.
  No VS Code-style tab strip.
- Actions live inline with the content they affect.

The mock `mockups/01-knowledge-why.html` is the byte-for-byte
visual reference for spacing, typography, color tokens, and the
graph+detail layout. CSS lifts color/spacing tokens from the
shell's locked grammar — no new tokens.

### Renderer kind per view

Per the rendering decision: Go templates for pages, lists, feeds,
metric strips, and SSE-driven fragment swaps. Islands only where
genuine interactivity is required (the why graph's pan/zoom/click
behavior, the writer's frontmatter form + body editor).

| View | Route | Renderer | Justification |
|---|---|---|---|
| Browse | `/knowledge` | template + SSE | static card grid + facet swap |
| Search | `/knowledge/search` | template + SSE | grouped result list |
| Why (graph) | `/knowledge/why` | template **+ island** | graph pan/zoom/click-to-expand |
| Staleness | `/knowledge/staleness` | template + SSE | actionable list |
| Recent | `/knowledge/recent` | template + SSE | reverse-chronological feed |
| Writer | `/knowledge/write` | **island** | frontmatter form + body editor |
| Per-entry detail | `/knowledge/<slug>` | template | body + provenance + usages |

### The seven views

#### 1. Browse — `/knowledge` (default, template + SSE)

Faceted browse of the corpus. Page hero (eyebrow `hero · main ·
knowledge`, title "Browse the corpus", short subtitle). Below the
hero: a filter row of chip-style facets across the top of the
content column (not a sidebar — the locked grammar has no rails).
Below the filter row: a responsive card grid (3-col at desktop,
1-col at narrow).

**Data source:** `internal/retrieval/.Query(type_facet=knowledge)`
with no query string returns the recency-sorted full corpus.
Facet aggregations come from `node_index`.

**Facets exposed (chip row):**
- **Type** — `convention`, `decision`, `learning`, `pattern`,
  `note`, `rule`
- **Domain** — derived from frontmatter `domain` or the parent
  directory under `.hero/knowledge/`
- **Status** — `active`, `draft`, `superseded`, `deprecated`
- **Tag** — multi-select; from `node_index.tags`
- **Recency** — 7d / 30d / 90d / all
- **Has contradictions** — boolean toggle

SSE channel `/api/knowledge/stream` pushes a card-grid fragment
swap when the corpus changes (new capture, status change, edit).

Each card: kind chip (color-coded per the mock), slug (mono),
title, one-line description, footer with last-edited date and
author dot. Clicking a card opens `/knowledge/<slug>`.

#### 2. Search — `/knowledge/search?q=...` (template + SSE)

The federated unified-search results page. The shell chrome's ⌘K
search bar submits here when the user picks a knowledge or
all-types search.

**Data source:** `internal/retrieval/.Query` with the user's text;
results grouped by `node_type`. Type-boost multipliers already
land knowledge and spec results above commits per
[unified-retrieval-layer](../unified-retrieval-layer/spec.md).

Page hero with a single rounded text input pre-filled with the
query, then a results list **grouped by kind**: Knowledge entries
first, then Specs, then Commits, then Code symbols. Each group
has a heading and a count; result rows show a kind chip + slug +
title + one-line excerpt with query terms highlighted.

Contradiction warnings from retrieval-time checks render inline
on each result row as an amber "may be stale" pill.

#### 3. Why — `/knowledge/why?target=...` (template + island)

The signature view. Matches `mockups/01-knowledge-why.html`
exactly. Top-down:

1. **Page hero** — eyebrow (`hero · main · knowledge`), title
   "Why does this exist?" at 32px, then a 60px-tall rounded
   target input (mono font, search icon left, submit button
   right) prefilled with the current target (e.g.,
   `internal/serve/proposals.go :: approveProposal() · line 147`).
   Below the input, a quiet preset-chip row:
   `Depth: 3 hops ▾` · `Include: specs + decisions + conventions + commits ▾` · `Direction: upstream ▾`,
   with a quiet `Open in browse view →` text link on the right.
2. **Tiny metric strip** — 4 quiet tiles
   (Corpus entries · Stale flags · Reuse rate 7d · New this week).
   No sparklines, no tabbing. Pure orientation context.
3. **Provenance chain section** — section header
   ("Provenance chain") with a `Copy provenance.json` ghost button.
   2-column grid (`1.5fr / 1fr`):
   - **Graph pane (~60%)** — bordered rounded panel with a dot-grid
     background, the `<knowledge-why-graph>` custom element host,
     a small "Tracing upstream from <pill>" header, a legend pinned
     bottom-left (solid = explicit edge, dashed = inferred), and a
     zoom control cluster pinned bottom-right (`−`, `+`, `fit`,
     `reset`).
   - **Detail pane (~40%)** — chain breadcrumb at the top
     (`Learning → Decision (selected) → Spec → Convention → Target`),
     then a type-chip row (kind, status, slug, date), then the
     detail title, then rendered sections (Context, Options
     considered, Consequences as appropriate), then **Cited by**
     and **Cites** chip rows, a made-by line with avatar, and a
     bottom action bar (`Open full · Find usages · Supersede · Edit`).
4. **Plain-English summary** — section header, then a single
   prose paragraph at `max-width: 760px` attributed to
   `knowledge-narrator` with a regeneration timestamp.
5. **You might also explore** — section header with a
   `Browse all related →` link, then 3-4 neighbor rows
   (kind chip + mono slug + description).
6. **Worth re-checking** — staleness flag block in amber tint
   (icon + body + inline actions: `View supersession →`,
   `Open re-evaluation draft`, `Dismiss`).

**Data source:** `/api/knowledge/why?target=<id>&depth=N&direction=upstream`
returns the trace as a node/edge JSON payload (the same data
`hero why` formats as markdown — exposed structured). Wraps
`traversal.Why()` from [traversal-queries](../traversal-queries/spec.md).

**Island behavior:** `internal/serve/islands/knowledge-why-graph.js`
hydrates the `<knowledge-why-graph>` element. Hand-rolled web
component. Renders nodes as shape-coded rectangles/hexagons
(per the mock: learning = amber-tinted rect, decision = purple
hexagon, spec = blue rect, convention = green rect, target =
solid hero-blue rect with white text). Curved bezier edges
with labels. 1–2 dashed inferred edges. Clicking a node updates
the right-side detail pane **without a full page reload**
(island fetches `/api/knowledge/entries/<slug>` and swaps the
detail pane innerHTML). Click-to-expand on a node fetches
additional hops from `/api/knowledge/why?from=<node>&depth=1`.
Force-directed layout for ≤30 nodes; vertical-chain fallback
above 30.

#### 4. Staleness — `/knowledge/staleness` (template + SSE)

Page hero ("Worth re-checking", subtitle counting flagged items).
Below: a single actionable list grouped into three sections, each
with a "resolve" verb on every row:

- **Contradictions** — from `.hero/knowledge/contradictions.json`
  produced by [knowledge-contradiction-detection](../../../specs/knowledge-contradiction-detection/spec.md).
  Each row: conflict type chip + two conflicting slugs + one-line
  summary + inline actions (`Open merge draft`, `Mark intentional`).
- **Broken references** — entries whose `relates-to` / `supersedes`
  / `depends-on` slugs no longer resolve in the graph or on disk.
  Inline actions: `Re-link` (opens relations picker in the writer)
  or `Remove reference`.
- **Drift** — entries flagged by drift checks (a convention whose
  scope no longer matches any files, a learning older than 12
  months with zero references). Inline actions: `Mark superseded`
  or `Confirm still relevant`.

SSE channel pushes a count update so the sub-nav `Staleness <n>`
badge updates live without a reload.

#### 5. Recent — `/knowledge/recent` (template + SSE)

Page hero ("Recently captured"). Below: a vertical timeline list,
reverse-chronological. Each row: timestamp, actor (avatar + name,
human or agent), event kind chip (`captured`, `edited`,
`superseded`, `deprecated`), entry slug as a link, one-line summary
of what changed.

**Data source:** graph history rows for knowledge node types
(`valid_from` / `valid_to` pairs) joined with the capture event
log. SSE channel `/api/knowledge/recent/stream` pushes new rows
at the top of the feed.

#### 6. Writer — `/knowledge/write` (island)

The only writer in the home. Two modes: `?new=1` (empty draft) or
`?slug=<slug>` (edit existing entry). Page hero ("Write knowledge
entry") followed by the island host.

**Island:** `internal/serve/islands/knowledge-writer.js`. Two-pane
layout inside the page (not fixed; scrolls with the content):
- **Left** — frontmatter form: `type` dropdown (convention /
  decision / learning / note / pattern / rule), status, domain,
  tag chips with autocomplete, scope-glob picker (conventions
  only), `relates-to` / `supersedes` / `depends-on` picker with
  autocomplete from `/api/knowledge/entries` slugs, `applies-to`
  for cross-domain entries.
- **Right** — markdown body editor with heading shortcuts, list
  formatting, link insertion, code-block fencing. No live collab.
  Draft auto-save to `sessionStorage` keyed by slug.

**Save path:** `POST /api/knowledge/entries` (new) or
`PUT /api/knowledge/entries/<slug>` (update) writes the markdown
file to `.hero/knowledge/<type>/<slug>.md`, triggers a `hero scan`
projection so the entry appears in `node_index` in the same
session, and runs the on-write contradiction check from
[knowledge-contradiction-detection](../../../specs/knowledge-contradiction-detection/spec.md).
A warning surfaces inline if a conflict was detected.

Chat dispatch (if the user invokes ⌘K from inside the writer) is
brokered by [hero-chat-and-model](../hero-chat-and-model/spec.md);
this home does not run inference and does not embed an LLM in the
editor itself.

#### 7. Per-entry detail — `/knowledge/<slug>` (template)

Page hero (eyebrow = breadcrumb to corpus, title = entry title,
type/status/domain chips below). Sections:

- **Body** — rendered markdown of the entry.
- **Provenance** — compact rendering of the top-3 hops of
  `hero why <slug>`; "See full chain →" links to
  `/knowledge/why?target=<slug>`.
- **Usages** — back-references: specs citing this entry, commits
  referencing it (via `node_index`), other knowledge entries
  linking to it.
- **Supersession** — `supersedes` / `superseded-by` chain.
- **Recent edits** — last 5 edits (git blame + capture events).

Inline action row at the top (right of the title): `Edit · Supersede ·
Find usages · Mark stale · Open provenance →`. These are anchor
links / form posts to existing CLI-backed endpoints — no separate
verb strip.

### API surface

New endpoints under `/api/knowledge/*`, all served by
`internal/serve/api/knowledge.go`:

| Endpoint | Method | Purpose |
|---|---|---|
| `/api/knowledge/browse` | GET | Faceted browse query; returns card-grid fragment or JSON |
| `/api/knowledge/entries/{slug}` | GET | Single entry: body + frontmatter + recent edits |
| `/api/knowledge/entries` | POST | Create new entry (writer) |
| `/api/knowledge/entries/{slug}` | PUT | Update existing entry (writer) |
| `/api/knowledge/search` | GET | Federated search results (delegates to unified-search) |
| `/api/knowledge/why` | GET | Trace JSON for the graph view (wraps `traversal.Why()`) |
| `/api/knowledge/staleness` | GET | Contradictions + broken refs + drift |
| `/api/knowledge/recent` | GET | Recent events feed |
| `/api/knowledge/stream` | SSE | Live fragment updates for browse / recent / staleness count |

Each handler is a thin wrapper over the existing substrate — no
new corpus shape, no new query language. Edition gating applies
through the shell registry: every view in this home ships at
`HERO_EDITION=local`. Team/cloud/enterprise overlays (team
corpus rollup, peer-repo search results) come from the underlying
substrate, not this surface.

### CLI commands that land here as visual views

| CLI | Visual surface |
|---|---|
| `hero why <target>` | `/knowledge/why?target=<target>` |
| `hero search <query>` | `/knowledge/search?q=<query>` |
| `hero knowledge check` | `/knowledge/staleness` (contradictions section) |
| `hero contradictions [slug]` | `/knowledge/staleness?filter=<slug>` |

(`hero blocked` is a Work-home view, not Knowledge.)

### Cross-home navigation

Every knowledge entry has a canonical URL `/knowledge/<slug>`.
Deep links from a Work spec ("originating decision"), from Now
("recent capture"), from Agents (proposal touching a convention)
all open the same scrolling page per the shell router. The
"Why does this exist?" verb on a Work spec opens
`/knowledge/why?target=<spec-slug>`.

## Changes

**First-pass autopilot delivery** — replicates the Now home pattern
one-for-one. Browse is the default sub-state. Sections that depend on
not-yet-shipped substrates (Why-traversal, narrator summary, neighbor
retrieval) render as empty-state notices per the kickoff instruction.
Search / Recent / Write / per-entry routes and the two islands
(`knowledge-why-graph`, `knowledge-writer`) remain on the original
plan list below — they were not in this delivery's scope.

**Delivered in this pass:**
- `internal/serve/pages/knowledge/page.go` — Register + handle + buildPage + SectionFragment, mirrors `pages/now/page.go`
- `internal/serve/pages/knowledge/styles.go` — per-page CSS + JS inlined through HeadExtra, mirrors `pages/now/styles.go`
- `internal/serve/pages/knowledge/templates/page.html` — outer composition
- `internal/serve/pages/knowledge/templates/browse.html` — card-grid section (default sub-state)
- `internal/serve/pages/knowledge/templates/provenance.html` — Why-view two-column scaffold with empty-state fallback
- `internal/serve/pages/knowledge/templates/summary.html` — narrator paragraph block with empty-state fallback
- `internal/serve/pages/knowledge/templates/neighbors.html` — "You might also explore" rows
- `internal/serve/pages/knowledge/templates/staleness.html` — Worth-re-checking amber block with empty-state fallback
- `internal/serve/pages/knowledge/data/types.go` — value-typed payload structs (Corpus, Why, Summary, Neighbors, Staleness)
- `internal/serve/pages/knowledge/data/corpus.go` — `.hero/knowledge/<kind>/` walker, returns entries + recency
- `internal/serve/pages/knowledge/data/staleness.go` — reads `.hero/knowledge/contradictions.json`
- `internal/serve/pages/knowledge/data/why.go` — stub fetcher (returns Available=false; the spec's empty-state notice covers it)
- `internal/serve/pages/knowledge/data/summary.go` — stub fetcher (Available=false)
- `internal/serve/pages/knowledge/data/neighbors.go` — stub fetcher (empty rows)
- `internal/serve/pages/knowledge/data/events.go` — events.log helper for the "new this week" tile
- `internal/serve/pages/knowledge/import_test.go` — boundary guard: forbids chat/runner imports
- `internal/serve/pages/knowledge/page_test.go` — renders all five section ids + sub-nav
- `internal/serve/api/knowledge.go` — SSE channel + per-section fragment endpoints, mirrors `api/now.go`
- `internal/serve/shell/stubs.go` — removed the `knowledge` stub entry
- `internal/serve/shell/shell_test.go` — updated `TestStubHomes_AllRender` to drop the `knowledge` slug
- `internal/serve/server.go` — added `knowledgepage` import, mounted `KnowledgeHandler` on the top mux, registered the real Knowledge home in `buildShellRouter`

**Still to deliver (not in scope for this autopilot pass):**

All paths under `internal/serve/`. The Knowledge home lives in
`internal/serve/pages/knowledge/` per the repository's
templates-as-pages layout.

1. **Route handlers** — `internal/serve/pages/knowledge/page.go`
   - `GET /knowledge` → Browse handler; queries
     `internal/retrieval/` with `type_facet=knowledge`; renders
     `templates/browse.html`.
   - `GET /knowledge/search` → Search handler; reads `q`,
     delegates to `internal/retrieval/.Query`; renders
     `templates/search.html`.
   - `GET /knowledge/why` → Why handler; reads `target`, `depth`,
     `direction`; renders `templates/why.html` (the island fetches
     graph data client-side from `/api/knowledge/why`).
   - `GET /knowledge/staleness` → Staleness handler; reads
     `contradictions.json` + dangling-edge query + drift query;
     renders `templates/staleness.html`.
   - `GET /knowledge/recent` → Recent handler; reads graph history
     for knowledge nodes; renders `templates/recent.html`.
   - `GET /knowledge/write` → Writer host handler; renders
     `templates/write.html` (the island loads on hydrate).
   - `GET /knowledge/<slug>` → Per-entry detail handler; reads the
     markdown file + provenance + usages; renders
     `templates/detail.html`.
   - All handlers compose the shell chrome + Knowledge sub-nav
     via shared layout partials.

2. **Templates** — `internal/serve/pages/knowledge/templates/`
   - `sub-nav.html` — the Knowledge-owned sub-nav row (six tabs:
     `Browse · Search · Why · Staleness <n> · Recent · Write`).
     Receives the staleness count as a template data field.
   - `browse.html` — page hero + facet chip row + card grid.
   - `search.html` — page hero with rounded input + grouped
     result list (Knowledge / Specs / Commits / Symbols sections).
   - `why.html` — full Why-view scaffold matching the mock:
     page hero with target input + preset chips, metric strip,
     `Provenance chain` section with graph pane (hosting the
     island) and detail pane (server-rendered for initial load),
     `Plain-English summary` section, `You might also explore`
     section, `Worth re-checking` section.
   - `staleness.html` — three-section list (contradictions,
     broken references, drift) with inline resolve actions.
   - `recent.html` — reverse-chronological timeline list.
   - `write.html` — host page for the writer island.
   - `detail.html` — per-entry page (body, provenance summary,
     usages, supersession, recent edits) with inline action row.
   - `entry-card.html` — shared card partial used by Browse and
     by SSE fragment swaps.

3. **API handlers** — `internal/serve/api/knowledge.go`
   - Implements the table in §"API surface".
   - `browse` → `internal/retrieval/.Query(type_facet=knowledge)`
   - `entries GET` → markdown file read + frontmatter parse +
     graph history join for recent edits
   - `entries POST/PUT` → markdown write + projection trigger +
     on-write contradiction check
   - `search` → `internal/retrieval/.Query` with full facets
   - `why` → `traversal.Why()` returning structured JSON
   - `staleness` → reads `contradictions.json` + dangling-edge
     query + drift query
   - `recent` → graph history for knowledge node types joined
     with capture event log

4. **SSE fragment renderer** —
   `internal/serve/api/knowledge_stream.go` pushes pre-rendered
   HTML fragments (`<article>` cards for browse, `<li>` rows for
   recent, `<span>` count for the staleness badge) onto the SSE
   channel when underlying data changes. Triggered by
   `internal/index/` projection events and the capture event hook.

5. **Why-graph island** —
   `internal/serve/islands/knowledge-why-graph.js`. Hand-rolled
   `<knowledge-why-graph>` custom element. Reads `target`, `depth`,
   `direction` attributes; fetches `/api/knowledge/why`; renders
   inline SVG with shape-coded nodes (rect for learning/spec/
   convention, hexagon for decision, solid hero-blue rect for
   target), curved bezier edges with labels, 1–2 dashed inferred
   edges, legend bottom-left, zoom cluster bottom-right.
   Click-on-node fires a `node-selected` CustomEvent; the host
   page listens and swaps `templates/detail.html` content via
   `/api/knowledge/entries/<slug>` fetch. Click-to-expand fetches
   additional hops. Force-directed layout ≤30 nodes; vertical
   chain above. No D3 — tiny in-house spring simulator.

6. **Writer island** —
   `internal/serve/islands/knowledge-writer.js`. Hand-rolled
   `<knowledge-writer>` custom element. Two-pane layout:
   frontmatter form (left) + markdown body editor (right).
   Tag autocomplete from `/api/knowledge/tags`; slug autocomplete
   from `/api/knowledge/entries?prefix=...`. Save via `POST` /
   `PUT` to `/api/knowledge/entries`. Draft auto-save to
   `sessionStorage`. On-write contradiction warning renders as
   an inline banner above the body editor.

7. **Data fetchers** — `internal/serve/pages/knowledge/data/`
   - `browse.go` — facet query helpers
   - `detail.go` — entry + provenance + usages loader
   - `staleness.go` — contradictions + dangling-edge + drift loader
   - `recent.go` — recent events loader
   - All reuse `internal/retrieval/`, `internal/traversal/`, and
     existing contradiction-detection data — no new substrate.

8. **CSS** — `internal/serve/pages/knowledge/static/knowledge.css`.
   Card grid, facet chip row, target input, preset chips, metric
   strip, two-column graph+detail grid, neighbor rows, staleness
   block. Lifts color/spacing tokens from the shell's locked
   grammar — no new tokens. Matches `mockups/01-knowledge-why.html`
   class names and spacing.

9. **Island wire-up** — `templates/why.html` includes
   `<script type="module" src="/islands/knowledge-why-graph.js"></script>`
   and hosts `<knowledge-why-graph target="..." depth="3" direction="upstream"></knowledge-why-graph>`
   inside the graph pane. `templates/write.html` includes
   `<script type="module" src="/islands/knowledge-writer.js"></script>`
   and hosts `<knowledge-writer slug="..." mode="new|edit"></knowledge-writer>`.

10. **Router registration** — `internal/serve/router.go` (or the
    equivalent shell wire-up) mounts the seven Knowledge routes
    and the `/api/knowledge/*` API endpoints. The shell's top-nav
    `Knowledge` tab links to `/knowledge`.

11. **Migration of the legacy knowledge page** — the existing
    `internal/serve/ui/` flat directory list (if still mounted)
    is removed as part of this spec's landing. The seven new
    views replace it.

## Boundaries

- **Not in this spec:** the contradiction-detection algorithm.
  Lives in [knowledge-contradiction-detection](../../../specs/knowledge-contradiction-detection/spec.md).
  This home renders the flags it produces.
- **Not in this spec:** retrieval ranking or facet aggregation.
  Lives in [unified-retrieval-layer](../unified-retrieval-layer/spec.md)
  and [unified-search](../unified-search/spec.md). This home calls
  the existing façade.
- **Not in this spec:** the multi-hop traversal logic itself.
  Lives in [traversal-queries](../traversal-queries/spec.md). This
  home wraps `traversal.Why()` and renders the result.
- **Not in this spec:** writing specs (features, bugs, decisions
  as work items). That is a Work-home concern. This writer is for
  knowledge entries (conventions, decisions-as-ADR, learnings,
  patterns, notes, rules).
- **Not in this spec:** PM PRD editor or QA test-plan editor.
  Those packs register their own homes.
- **Not in this spec:** cross-repo knowledge write-back. Federation
  writes go through `hero handoff`, not this surface.
- **Not in this spec:** an embedded LLM in the writer suggesting
  content. Chat dispatch is brokered by
  [hero-chat-and-model](../hero-chat-and-model/spec.md); the
  writer here is human-driven editing of capture output.
- **Not in this spec:** the top-nav search bar UI itself. Owned by
  [hero-surface-shell](../hero-surface-shell/spec.md). This spec
  owns the Search results destination view.

## Acceptance Criteria

- WHEN the user opens `/knowledge` THE SYSTEM SHALL render the
  Browse view with the corpus filtered by the user's currently
  applied facet selection.
- WHEN the user selects a type / domain / status / tag / recency /
  has-contradictions facet THE SYSTEM SHALL update the card grid
  via SSE fragment swap without a full page reload.
- WHEN the user clicks a card in the Browse view THE SYSTEM SHALL
  navigate to `/knowledge/<slug>` and render the per-entry detail
  page with body, provenance summary, usages, supersession, and
  recent edits populated.
- WHEN the user clicks `Why` in the sub-nav THE SYSTEM SHALL
  navigate to `/knowledge/why`, or to `/knowledge/why?target=<id>`
  if a target was just selected from another view.
- WHEN the user enters a target in the Why view input and submits
  THE SYSTEM SHALL render the provenance graph for that target
  driven by `traversal.Why()`.
- WHEN the user clicks a node in the Why graph THE SYSTEM SHALL
  update the detail panel to show that node's content without a
  full page reload.
- WHEN the user clicks the click-to-expand affordance on a node
  THE SYSTEM SHALL fetch and render that node's additional origin
  edges in place.
- WHEN the user submits a query in the shell's ⌘K bar with a
  knowledge or all-types scope THE SYSTEM SHALL open
  `/knowledge/search?q=<query>` with results grouped by kind
  (Knowledge / Specs / Commits / Symbols) and ranked via the
  unified retrieval layer.
- WHERE the search query crosses into specs / commits / code
  symbols THE SYSTEM SHALL group results by kind in the Search
  view rather than interleaving them.
- WHEN the user opens `/knowledge/staleness` THE SYSTEM SHALL
  render contradictions from `contradictions.json`, broken
  references from dangling-edge queries, and drift flags, each
  with inline resolve actions.
- WHEN the user clicks `Open merge draft` on a contradiction in
  `/knowledge/staleness` THE SYSTEM SHALL open `/knowledge/write`
  pre-populated with a merged-frontmatter draft for one of the
  conflicting entries.
- WHILE knowledge entries have detected contradictions THE SYSTEM
  SHALL show a numeric staleness count badge on the sub-nav
  `Staleness` tab.
- WHILE the `/knowledge/recent` view is open THE SYSTEM SHALL push
  new capture / edit / supersession events as SSE fragments at the
  top of the feed within 2 seconds of the underlying event.
- WHEN the user saves an entry from the writer THE SYSTEM SHALL
  write the markdown file to `.hero/knowledge/<type>/<slug>.md`,
  trigger projection so the entry appears in the unified index in
  the same session, and surface any on-write contradiction warning
  inline above the body editor.
- IF the unified retrieval layer returns zero results for a search
  query THEN THE SYSTEM SHALL fall through to graph node-key
  matching per [unified-retrieval-layer](../unified-retrieval-layer/spec.md)
  and render the fallback results with a "matched by exact key"
  tag.
- IF a `/knowledge/<slug>` deep link references a slug not present
  in the corpus THEN THE SYSTEM SHALL render an empty-state page
  with a `Create this entry` link opening `/knowledge/write?new=1&slug=<slug>`.
- THE SYSTEM SHALL render the Why view as a scrolling page with
  the graph pane and detail pane laid out inline as a two-column
  grid inside the content stream (no fixed right rail, no fixed
  bottom strip).
- THE SYSTEM SHALL render the sub-nav row beneath the shell's top
  nav on every Knowledge route with `Browse · Search · Why ·
  Staleness <n> · Recent · Write` and a hero-blue underline on
  the active tab.
- THE SYSTEM SHALL render the Why graph and the Writer as islands
  (`<knowledge-why-graph>` and `<knowledge-writer>`); THE SYSTEM
  SHALL render every other Knowledge view as a Go template
  fragment with SSE updates where listed.
- THE SYSTEM SHALL expose every Knowledge view at a canonical URL
  under `/knowledge` that renders the same page state whether
  reached by in-app navigation or by direct deep link.

## Risks

- **Browse performance with thousands of entries.** A workspace
  with thousands of knowledge entries plus cross-repo entries can
  make the unfiltered browse slow. Mitigation: cap default browse
  at 200 most-recent cards; facets narrow before expansion; SSE
  delivers incremental loads.
- **Why-graph blowup.** A target with 50+ origin hops produces an
  unreadable force-directed graph. Mitigation: fall back to
  vertical chain layout above 30 nodes; cap initial depth at 3
  (matching the mock's preset); user expands on demand.
- **Writer contradiction-check false positives.** The on-write LLM
  check may produce noisy warnings on benign edits. Mitigation:
  reuse the false-positive marking flow from
  [knowledge-contradiction-detection](../../../specs/knowledge-contradiction-detection/spec.md)
  inside the writer's inline warning.
- **SSE channel multiplexing.** Three views in this home want SSE
  (browse fragment, recent feed, staleness count). The shell's
  session SSE transport must support multiple named channels per
  session. If it does not yet, this spec depends on the shell
  adding that capability — cross-spec coordination needed.
- **Sub-nav grammar conflict with the older shell spec.** The
  current shell spec ([hero-surface-shell](../hero-surface-shell/spec.md))
  still references the older desktop grammar (left nav, VS Code
  tabs). The deployment-and-rendering decision supersedes that.
  This spec aligns with the decision (slim top nav + sub-nav row).
  Shell spec should be updated separately; this home is built to
  the decision spec's grammar regardless.
- **Provenance completeness.** `hero why` is only as good as the
  edges in the graph. If ingestion of commits / notes / decisions
  is incomplete (per `master-ingest-restore`), the visual chain
  will be incomplete and may mislead. Mitigation: the Why graph
  shows a "graph incomplete" banner when the active scan is
  older than 24 hours or when the corpus has unindexed files.
- **Cross-pack search ownership.** Engineering owns
  `/knowledge/search`; PM and QA may want their own scoped views
  later. The shell's ⌘K bar routes to the active pack's search
  view. Engineering's Knowledge Search is the canonical
  federated view today.

## Validation

- **Manual:** open `/knowledge`; verify all six sub-nav tabs
  navigate to their routes; verify `Browse` is the default.
- **Manual:** apply each facet (type, domain, status, tag,
  recency, has-contradictions) and verify the card grid filters
  via SSE without a full reload.
- **Manual:** open `/knowledge/<slug>` for a known entry; verify
  body, provenance summary, usages, supersession, and recent edits
  populate.
- **Manual:** open `/knowledge/why?target=master-ingest-restore`;
  verify the graph renders with shape-coded nodes, curved edges,
  labels, legend, zoom cluster; verify clicking a node updates
  the detail pane without a full reload; verify click-to-expand
  fetches additional hops.
- **Manual:** layout match against `mockups/01-knowledge-why.html`
  byte-for-byte at desktop width — page hero spacing, target
  input size (60px), preset chip row, metric strip, two-column
  graph+detail grid (1.5fr / 1fr), neighbor rows, staleness
  block.
- **Manual:** submit a query from the shell ⌘K bar; verify the
  Search view opens at `/knowledge/search?q=<query>` with results
  grouped by kind.
- **Manual:** open `/knowledge/staleness`; verify contradictions
  from `contradictions.json` render with correct resolve actions;
  verify a deliberately broken `relates-to` link appears under
  Broken references.
- **Manual:** trigger `/capture` in another session; verify
  `/knowledge/recent` shows the new event within 2 seconds via
  SSE; verify the sub-nav `Staleness` badge updates if the new
  entry triggered a contradiction.
- **Manual:** open `/knowledge/write?new=1`; create a new
  convention with scope/tags/relations; verify the markdown file
  lands at `.hero/knowledge/conventions/<slug>.md`, the entry
  appears in `/knowledge` on next render, and on-write
  contradiction detection runs.
- **Test:** handler tests for each `/api/knowledge/*` endpoint —
  round-trip a fixture entry through browse, detail, search, why,
  staleness, recent, and writer save.
- **Test:** SSE stream test — write a fixture knowledge file;
  assert a stream subscriber sees a fragment within 1 second.
- **Test:** Why-graph island unit test — given a fixed trace JSON,
  assert the rendered SVG has the expected node count, edge count,
  and shape coding; assert click-on-node fires the
  `node-selected` event.
- **Test:** Writer island round-trip — frontmatter form
  serialization produces identical markdown to a manual write for
  the same input.
- **Visual:** side-by-side compare the Why view against
  `mockups/01-knowledge-why.html`; verify no drift in tokens
  (colors, spacing, typography) or layout.

## Kickoff

Paste this into a fresh `/deliver` session to start work:

> Deliver `hero-knowledge-home`. The visual source of truth is
> `.hero/planning/features/hero-knowledge-home/mockups/01-knowledge-why.html`
> (the Why view). Build the scrolling web-app page at `/knowledge`
> with a slim sub-nav row (`Browse · Search · Why · Staleness <n> ·
> Recent · Write`) and seven routes: `/knowledge` (Browse default),
> `/knowledge/search`, `/knowledge/why`, `/knowledge/staleness`,
> `/knowledge/recent`, `/knowledge/write`, `/knowledge/<slug>`.
> Templates for all views; islands only for `<knowledge-why-graph>`
> (pan/zoom/click-to-expand provenance graph) and `<knowledge-writer>`
> (frontmatter form + markdown body editor). Reuse
> `internal/retrieval/` for browse + search, `internal/traversal/`
> for the Why view, `.hero/knowledge/contradictions.json` for
> Staleness. No new corpus shape, no new query language, no React,
> no top-level build step. Grammar is the web-app grammar from
> `hero-surface-deployment-and-rendering` (slim top nav, scrolling
> content, no fixed rails). Files land under
> `internal/serve/pages/knowledge/` (templates + handlers + data
> fetchers), `internal/serve/api/knowledge.go` (API), and
> `internal/serve/islands/knowledge-why-graph.js` +
> `internal/serve/islands/knowledge-writer.js` (islands). Start by
> reading the mock and the four parent/dep specs listed in
> frontmatter, then scaffold the seven routes with stub templates
> before filling in the Why view to match the mock byte-for-byte.
