---
title: Unified Retrieval Layer — Cross-Type Ranking via Faceted Search Index
slug: unified-retrieval-layer
type: feature
status: delivering
status_verified: 2026-04-29
horizon: next
priority: P1
tags: [search, retrieval, indexing, architecture, ranking, ingest-projection]
created: 2026-04-28
relations:
  - target: spec-prioritization
    kind: tagged-by
  - target: master-ingest-restore
    kind: depends-on
  - target: get-back-on-track
    kind: candidate-child
mission_alignment: |
  Mission is "AI gets the right context at the right moment." Today
  retrieval ranks across heterogeneous node types using hand-rolled
  type weights (per polish-round-1's commit: "specs/knowledge/code
  rank above commits/issues/people") because SQL doesn't naturally
  do cross-type relevance scoring. Each new ranking imbalance accrues
  another hack. A faceted unified search index — projecting from the
  graph at ingest time — gives us BM25 ranking with type boosts as
  scoring multipliers, so retrieval delivers the right slice without
  workarounds.
principles_check: |
  Serves #1 (it just works — search ranking is consistent across all
  types without hand-tuning per query). Serves #5 (graph stays for
  traversal queries, search index for ranked retrieval — each tool
  for its job). Doesn't risk any principle.
smoke: deferred
---

## Captured + reframed

User asked during the get-back-on-track recovery work
(2026-04-28): *"should we be indexing specs as well as leaving them
on the filesystem? do we need a lightweight embedded search library
so we aren't always fighting search on diverse db tables?"* And on
pushback: *"we defered it - i know - but it still nags at me because
we have disparate stuff and hit the how do you rank and organize it
together right - something that search libraries are very good at
yeah?"*

Initially captured at `horizon: someday`. **Promoted to `next` after
re-examination** — the user's instinct is correct and the
architectural smell is already in the codebase.

## The architectural smell (already in the code)

From `d9997ea` (polish round 1) commit message, verbatim:

> *"Graph search now weights node types: specs/knowledge/code rank
> above commits/issues/people. The 'Task' search on go-task/task
> used to return 99 commit messages and 1 useful result; now returns
> ContextDocs / Symbols at the top."*

That's a **hand-rolled type-weight workaround** for a missing
cross-type ranking layer. It fixed one query's symptom. The next
type-imbalance query (sales notes drowning decisions, code symbols
drowning architecture docs, vertical X drowning vertical Y) needs
the same hack again. Each accrues hand-tuned cutoffs that don't
generalize.

A proper search index would have indexed every node type in one
inverted index, ranked globally with BM25, applied type boosts as
*scoring multipliers* (not crude cutoffs), and avoided the
"99 commits drowning 1 useful result" failure from day one — and
every variant of it that shows up next.

## The corrected shape

Keeping markdown on disk for human iteration:

| Layer | Job |
|---|---|
| Markdown files | Source of truth — humans author, git reviews. Untouched. |
| Graph DB | Structural truth — nodes, edges, traversal, provenance, bitemporal history. Best at relational queries (`hero why`, `hero blocked`, `hero impact`). |
| Unified search index | Cross-type ranking — every node's content goes in, with `type` / `tags` / `valid_from` / `repo` as facets. Best at *"find me the N most relevant things across all node types."* |

Both indexes project from the same source (graph nodes + markdown
content). They serve different question shapes:
- Relational (*"why"*, *"depends on"*, *"blocked by"*) → graph
- Cross-cut ranked retrieval (*"things about X"*) → search index

This isn't "replace FTS5 with Bleve" — it's "add a properly-
faceted unified index alongside the graph so retrieval doesn't
keep accruing hand-rolled weight hacks."

## Current state (for context)

Specs already live in three places:
- **Filesystem** (`*.md`) — source of truth, human-authored, git-reviewed
- **FTS5** (`internal/index/`) — tokenized text, keyword search
- **Graph** (`internal/graph/`) — Feature/Initiative/Decision nodes + edges

Each does what it's best at. `hero search` queries graph first, falls
through to FTS5 (per `c4251de`). `hero ask` does FTS5-only retrieval
+ extractive answer.

The v2 charter explicitly chose FTS5 over semantic search:
> *"FTS5 with porter stemming is sufficient at project scale —
> semantic search deferred."*

## What might trigger promotion to `next`

Any of:
- An audit shows `hero ask` / `hero search` / `hero relevant` are
  systematically returning poor matches on real workspaces (not just
  edge cases)
- A vertical (e.g., Hero Sales) has a corpus shape FTS5 ranks badly
  (long-form documents, conversational content)
- A user report: "I knew the answer was in the corpus but search
  didn't find it"
- The graph populates fully (post `master-ingest-restore`) and we
  see retrieval is *still* the bottleneck

## Sketch (when promoted)

### Phase A — Unified retrieval facade (no new engine) ✓ SHIPPED

Shipped 2026-04-29. `internal/retrieval/` package with Query/Result/
Retriever types. All four callers migrated (search, ask, relevant,
MCP tools). Type-boost multipliers fix the d9997ea regression.
Graph-first routing with FTS5 fallback. 7 tests passing.

### Phase B — Enriched FTS5 with graph node projection ✓ SHIPPED

**Engine decision (2026-04-29):** Stay on SQLite FTS5 with a richer
schema. Zero new dependencies. BM25 ranking via FTS5's built-in
`rank` function. Bleve rejected (unnecessary dep; FTS5's query
language is sufficient at project scale).

Implementation:
- `fts_nodes` FTS5 virtual table + `node_index` metadata table
  added to `index.go` migrations
- `ProjectGraphNodes(graphDB *sql.DB)` in `internal/index/project.go`
  reads all current graph nodes and projects title/body into fts_nodes
- Projection wired into `hero scan` (after graph population) and
  `index.Rebuild` (when graph.db exists)
- `retrieveViaNodeIndex` in retrieval.go: BM25 MATCH on fts_nodes
  joined with node_index, scores = -bm25Rank × typeBoost(nodeType)
- Routing: fts_nodes BM25 → graph LIKE fallback (AC-5) → spec FTS5
- 4 new tests: projection populates, BM25 ranking, d9997ea BM25
  regression, AC-5 fallback

### Phase C — Semantic / vector search (when LLM cost makes sense)

Add `sqlite-vec` (or `USearch` Go binding) for embedding storage.
Tier-2 extraction generates embeddings during ingest. Queries hit
both keyword and vector indexes; results are fused with reciprocal
rank fusion or weighted score.

Hard prerequisites:
- Local embedding model (or cheap cached cloud embeddings) — must
  not violate the "no LLM in binary's hot path" anti-pattern unless
  it's clearly substrate work
- Tier-2 extraction already running automatically (per
  `master-ingest-restore`)

This is the biggest jump. Worth it when *"query intent ≠ literal
terms"* is the dominant failure mode.

## Acceptance criteria (skeleton — accrete as we go)

**AC-1:** ✓ Every graph node (post-`master-ingest-restore`) has a
corresponding entry in the unified search index, projected at
ingest time. `hero search "Task"` doesn't return 99 commits drowning
the 1 useful result — type boosts apply as scoring multipliers, not
hard cutoffs. *Passing: ProjectGraphNodes projects at scan+rebuild;
TestBM25RegressionD9997ea confirms Feature ranks above 50 Commits.*

**AC-2:** ✓ Search results across types are ranked globally with BM25
(or comparable proper-relevance algorithm), not per-type SQL
windowing. *Passing: fts_nodes BM25 rank × typeBoost; tested in
TestBM25RankingViaNodeIndex.*

**AC-3:** Partial. Type facet available via `node_index.node_type`.
Tag/repo/date facets are projected into `node_index` but not yet
exposed in the retrieval Query API. Next step when a caller needs
them.

**AC-4:** ✓ A single retrieval interface (`internal/retrieval/`) wraps
the search index and the graph, so every caller (`hero search`,
`hero ask`, `hero relevant`, MCP tools) goes through one path. No
caller wires raw SQL anymore. *Passing since Phase A.*

**AC-5:** ✓ When the search index has zero matches, retrieval falls
through to graph node-key matching (current `hero search` behavior
preserved for the "I know exactly what I want" case). *Passing:
TestBM25FallbackToGraphLIKE.*

ACs accrete as integration surfaces.

## Sequencing

This depends on `master-ingest-restore` shipping first. Reasoning:
without the corpus being complete, the search index would just
expose the same incompleteness with a fancier ranker. Build the
substrate; project from it.

After `master-ingest-restore` lands → this becomes a clean next
move (probably 2-3 days of work depending on engine choice).

## Engine decision (resolved 2026-04-29)

**Chose: SQLite FTS5 with enriched schema** (fts_nodes + node_index).

Rationale: zero new dependencies, BM25 ranking via FTS5's built-in
`rank` function, type/tag/repo facets via the `node_index` join
table. FTS5's query language (porter stemming, phrase queries, OR/AND
operators) is sufficient at project scale. Bleve rejected — would add
a Go dependency and separate index file for capabilities we don't
need yet. sqlite-vec deferred to Phase C (requires embedding model).
