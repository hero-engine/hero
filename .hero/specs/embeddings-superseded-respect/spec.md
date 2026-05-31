---
title: Embeddings & Vector Retrieval Respect `superseded_by`
slug: embeddings-superseded-respect
type: feature
status: completed
priority: high
tags: [retrieval, embeddings, vector-search, hybrid, supersede, context-injection]
created: 2026-05-30
related:
  - kind: follows
    target: superseded-specs-soft-archive
---

# Embeddings & Vector Retrieval Respect `superseded_by`

## Goal

The semantic / vector retrieval path treats a spec carrying `superseded_by:` the same way the lexical path does: de-weighted in the ranked list, annotated with a `[SUPERSEDED → <slug>]` redirect on the returned snippet, and never out-ranking its replacement at equivalent similarity. After this ships, no surface — lexical, vector, or hybrid — can hand a stateless agent a v1 spec at full weight when a current v2 exists.

## Kickoff

Parent spec `superseded-specs-soft-archive` is in planning and de-weights superseded specs in lexical FTS only. The vector path (`internal/retrieval/retrieval.go:474` `retrieveHybrid` plus `internal/embeddings/storage.go` `QuerySimilar`) doesn't carry the `superseded_by` signal at all, so semantic hits will keep surfacing v1 noise after the parent ships.

**Status:** planning — design just landed; depends on parent spec's frontmatter + indexer changes being merged first.

**Pick up at:** Phase 1 — add a `SupersededSlugs map[string]string` overlay built once per `Retrieve` call from the `specs` table (cheap query: `SELECT slug, superseded_by FROM specs WHERE superseded_by != ''`). Apply in `retrieveHybrid` after `QuerySimilar` returns, before `fuseRRF`. Read this spec, then `retrieveHybrid` / `fuseRRF` in `internal/retrieval/retrieval.go` and `QuerySimilar` in `internal/embeddings/storage.go` before writing code.

→ `.hero/planning/features/embeddings-superseded-respect.md`

## Problem

The parent spec [`superseded-specs-soft-archive`](./superseded-specs-soft-archive.md) closes the supersede gap for lexical search (FTS5 BM25, graph node index) by:

1. Adding a `superseded_by` column to the `specs` table.
2. Multiplying the final score by `supersededDeweight = 0.3` in `retrieveViaNodeIndex` and `retrieveViaFTS`.
3. Prefixing the snippet with `[SUPERSEDED → <slug>]`.

That parent spec calls out one explicit leak in its own **Out of Scope** section (line 267):

> **Embedding-layer de-weight.** Vector retrieval (`embeddings.QuerySimilar`) is not modified in this spec — superseded specs in the embeddings corpus still come back at full similarity. Follow-up spec once we have a calibrated way to attenuate embedding hits.

This is that follow-up.

Concretely, after the parent ships:

- `retrieveHybrid` (`internal/retrieval/retrieval.go:474`) runs the lexical path (now supersede-aware) and the vector path (still naive), then fuses via Reciprocal Rank Fusion in `fuseRRF` (line 503).
- `fuseRRF` matches lexical and vector results by key (spec slug for `corpus="spec"` chunks). A vector-only hit on a superseded spec enters the fused ranking with full RRF weight and no annotation.
- Worse: a superseded spec that *also* hits in the lexical path gets de-weighted lexically (its `Score` multiplied by 0.3), but RRF in this implementation uses **rank position**, not raw score — so the lexical de-weight has no effect on the fused rank. The vector hit re-promotes it.

Net effect: the parent spec ships, the user runs `hero search --semantic "surface polish"`, and `hero-surface-polish-v1` comes back at the top with no `[SUPERSEDED]` marker. The exact failure mode the parent was built to prevent.

This is mission-critical: Hero's job is making the right context land. Vector hits returning v1 noise defeats that.

## Design

Five decisions, each with an explicit reason.

### Decision 1 — Signal lives at query time, not in the embedding row

**Choice:** option (b) — join at query time against an in-memory map of `{superseded_slug → replacement_slug}` built once per `Retrieve` call from the existing `specs` table.

The embedding row schema (`vec_chunks` in `internal/embeddings/storage.go:55-69`) holds: `chunk_id, corpus, source_id, section, text_hash, vector, embedded_at`. It deliberately carries no frontmatter — only a `text_hash` for invalidation. `specMetadataPrefix` (chunker.go:274) folds title/type/status/tags into the **embedded text** itself, but that's input to the model, not a queryable column.

Adding a `superseded_by` column to `vec_chunks` would require re-embedding every chunk of a spec the moment `hero supersede` runs. That's expensive (10–50 chunks per spec × N specs in a v2 backfill pass) and wasteful — only metadata changed, the vector is unchanged. A re-embed pass also makes `hero supersede` slow enough to discourage use.

The cheap path is the right path: at the top of `retrieveHybrid`, run

```sql
SELECT slug, superseded_by FROM specs WHERE superseded_by != ''
```

into a `map[string]string`. The parent spec already adds the `superseded_by` column to `specs` and populates it on `UpsertSpec`. Cardinality is small (tens, low hundreds even in a mature workspace), the query hits an index on a small table, and the map is built once per query — negligible cost. The vector store stays a pure similarity engine; supersede is a retrieval-layer overlay.

**Cold-start corollary (covered by Decision 4):** because the signal is read from `specs` (which the indexer updates on every `hero index` and `hero supersede` re-runs inline), the overlay is correct the instant the supersede edit lands. No re-embed step, no staleness window.

### Decision 2 — De-weight is the **same 0.3 multiplier** applied to the per-spec RRF score, after fusion

Cosine similarity ([0, 1] post-normalize) and BM25 (unbounded negative) are on different scales — applying 0.3 to a raw cosine score would behave differently than applying it to a BM25 score. But `fuseRRF` already normalizes that away: every result, lexical or vector, is reduced to a **Reciprocal Rank Fusion score** of `1/(k+rank+1)` where rank is 0-indexed and k=60. RRF scores are bounded, additive, and dimensionless.

The cleanest hook is therefore: **after `fuseRRF` produces the merged map but before sorting**, multiply each fused result's `rrfScore` by 0.3 when its `Key` is in the superseded overlay. This:

- Matches the parent spec's `supersededDeweight = 0.3` constant exactly (one knob, one place, one meaning).
- Operates on the same dimensionless RRF score regardless of which path(s) the spec came from.
- Penalizes the spec **once** even if it appears in both lexical and vector hits (Decision 5).
- Leaves the underlying vector similarity untouched — `QuerySimilar` stays a pure cosine engine.

Rejected alternatives:

- **Multiplicative 0.3 on raw cosine inside `QuerySimilar`:** would require `QuerySimilar` to know about specs and the `specs` table. Layering violation; also wrong for non-spec corpora that don't have supersede semantics.
- **Rank-position penalty (e.g. push rank down by 10):** brittle when result set is small; a single-result page would push the superseded entry off-list silently.
- **Drop-below-threshold:** violates the "annotate, don't drop" principle the parent spec already set.
- **Additive offset:** RRF scores at k=60 are tiny (~0.016 at rank 0); an additive offset would either dominate or vanish depending on the constant. Multiplicative is the right shape.

The same `supersededDeweight` constant in `internal/retrieval/retrieval.go` is reused — no second constant to drift.

### Decision 3 — Annotation marker prepended to the `Snippet` field, same wording as parent

When a fused result's `Key` is in the superseded overlay, the result's `Snippet` is prefixed with `[SUPERSEDED → <replacement-slug>] ` — character-identical to the parent spec's lexical-path marker (`internal/retrieval/retrieval.go` after the parent ships). The replacement slug comes from the overlay map value.

Reasoning:

- Identical marker across paths means the model sees the same redirect string whether the spec surfaced via FTS, the node index, or the vector path. No per-path training in the agent's head.
- Prepended to `Snippet` (not a separate field) means every caller that prints results gets the redirect for free — `hero search` output, MCP tool responses, context-injection bullets — without each caller needing to know about a new field.
- The annotation is applied **regardless of `IncludeSuperseded`** — the de-weight is the rank effect; the annotation is the visibility effect. They're independent, same as the parent spec's rule (parent AC: "WHEN `hero search --include-superseded` is passed THE SYSTEM SHALL skip the de-weight multiplier but still emit the `[SUPERSEDED → <slug>]` annotation").
- The annotation lives in `fuseRRF` (right where the de-weight is applied) — one mutation site, no risk of de-weighting without annotating or vice versa.

For non-spec corpora (knowledge, convention, code, event) the overlay never matches (overlay is built from `specs.superseded_by`, which only specs have), so no annotation, no de-weight. Conventions have their own supersede model that's out of scope.

### Decision 4 — No re-embed required; supersede is observable on the next query

Because the signal is a query-time overlay (Decision 1), `hero supersede <old> --by <new>` does not touch `vec_chunks` at all. The flow:

1. `hero supersede` writes `superseded_by: <new>` into the old spec's frontmatter (parent spec's flow).
2. `hero supersede` calls `hero index --if-stale -q`, which updates the `specs` table (parent spec's flow).
3. The very next `Retrieve` call builds its overlay from the now-updated `specs` table and applies de-weight + annotation to any vector hit on the superseded slug.

No re-embed. No vector store migration. No background job. The `embedded_at` timestamp on the chunk is unaffected — the text hash is unchanged because the underlying spec body didn't change (the parent's "render-time banner, on-disk file untouched" decision composes perfectly here).

The migration story is therefore: **nothing to migrate.** Existing installs pick up the behavior the next time they query, the instant the parent spec's indexer change populates `specs.superseded_by`.

### Decision 5 — Hybrid double-penalty avoidance: de-weight applied exactly once, after fusion

A superseded spec can hit in both rankings:

- Lexical path (parent spec): score already multiplied by 0.3 before lexical results are handed to `fuseRRF`.
- Vector path (this spec): chunk comes back from `QuerySimilar` at full similarity, enters `fuseRRF` keyed by `vc.SourceID` (the spec slug).

Today's `fuseRRF` uses rank position, not raw score — so the lexical 0.3 multiplier has **already been thrown away** by the time RRF runs. That makes the answer simple:

**Rule:** the de-weight is applied **once, in `fuseRRF`, after merge, before sort**, against the final `rrfScore`. The lexical path's pre-fusion 0.3 multiplier on `Score` is preserved as informational (so callers that bypass RRF still see the de-weighted score), but RRF re-applies the de-weight against the fused score because RRF discards `Score` and operates on rank position.

This means:

- Specs that hit only lexically: de-weighted in lexical path (parent), then de-weighted again in `fuseRRF` overlay. **This is wrong** — would be double penalty.
- Specs that hit only via vector: de-weighted once in `fuseRRF` overlay. Correct.
- Specs that hit in both: de-weighted once in lexical path (parent), then once in `fuseRRF` overlay. Wrong by the same logic.

To make it exactly-once **regardless of path combination**, the rule must be: **`fuseRRF` is the single application point for the supersede de-weight in the hybrid path.** The lexical de-weight from the parent spec is **skipped** when the lexical path is being called as a sub-step of `retrieveHybrid`.

Implementation: pass a `skipSupersedeDeweight bool` flag down the lexical sub-call from `retrieveHybrid`. The lexical path still adds the annotation marker (so non-hybrid callers see the redirect), but the score multiplier is gated on this flag.

```
non-hybrid lexical:   apply de-weight + annotation in lexical path ✓
hybrid lexical step:  apply annotation only; de-weight applied later in fuseRRF
hybrid vector step:   no per-path treatment; de-weight + annotation in fuseRRF
fuseRRF:              de-weight + annotation applied once per fused result
```

Equivalent alternative considered: apply de-weight in both lexical and vector paths, then *don't* re-apply in `fuseRRF`. Rejected because (a) the lexical path's multiplier vanishes through RRF's rank-position step anyway, so the lexical-path de-weight would do nothing for hybrid callers, and (b) it splits the rule across three sites instead of one.

The chosen rule keeps `fuseRRF` as the single source of truth for hybrid supersede behavior. Non-hybrid lexical callers (the parent spec's behavior) are unchanged.

### Implementation surface (file-level)

- `internal/retrieval/retrieval.go`:
  - Add `loadSupersededOverlay(db *sql.DB) (map[string]string, error)` helper — single SQL query against `specs`.
  - Add `skipSupersedeDeweight bool` field to `Query` (unexported documentation comment: "internal flag set by retrieveHybrid when running the lexical sub-call; do not set from external callers").
  - In `retrieveHybrid`: load overlay once; pass `skipSupersedeDeweight=true` into the lexical sub-call; pass the overlay into `fuseRRF`.
  - Extend `fuseRRF` signature: `fuseRRF(lexical []Result, vector []embeddings.ScoredChunk, supersededBy map[string]string, k int, limit int) []Result`. After fusion, walk the result map: when `Key` is in `supersededBy`, multiply `rrfScore` by `supersededDeweight` (the same constant the parent introduces) and prefix `Snippet` with `[SUPERSEDED → <replacement>] `.
  - In `retrieveViaFTS` / `retrieveViaNodeIndex` (the parent spec's de-weight site): wrap the multiplier in `if !q.skipSupersedeDeweight { score *= supersededDeweight }`. Annotation marker is always applied.
- `internal/embeddings/storage.go`: **no changes.** `QuerySimilar` stays a pure cosine engine.
- `internal/embeddings/refresh.go`: **no changes.** No re-embed on supersede.
- `internal/cli/search.go`: no flag changes — `--include-superseded` from the parent spec already routes through `Query.IncludeSuperseded`; when true, `retrieveHybrid` clears the overlay before calling `fuseRRF`, so neither de-weight nor annotation fires for hybrid+include-superseded calls **except** the annotation, which is re-applied unconditionally per parent rule. Concretely: build a separate annotation-only overlay path (the marker always applies; only the score multiplier is gated).
- `internal/retrieval/retrieval_test.go`: add hybrid-with-superseded scenarios — vector-only hit, lexical-only hit, both-paths hit, plus an `IncludeSuperseded=true` hybrid case.

### Feature flag for rollback

Wrap the entire vector-path overlay (load + apply in `fuseRRF`) in a check of `cfg.RetrievalSupersedeRespect` (default `true`). When `false`, `fuseRRF` runs unchanged and behavior is exactly today's. One env-overridable knob in `internal/config` so an operator can disable if score calibration turns out wrong in production. The lexical path's de-weight is governed by the parent spec's own flag (whatever shape that takes) — this flag controls only the hybrid/vector overlay, so they can be rolled back independently.

## Acceptance Criteria

- THE SYSTEM SHALL build a query-time `superseded_by` overlay from the `specs` table at the start of every `retrieveHybrid` call.
- THE SYSTEM SHALL NOT add a `superseded_by` column to `vec_chunks` and SHALL NOT re-embed any chunk in response to `hero supersede`.
- WHEN a vector result's `SourceID` is in the superseded overlay AND `IncludeSuperseded` is false THE SYSTEM SHALL multiply that result's fused RRF score by `supersededDeweight` (the same 0.3 constant the parent spec defines).
- WHEN any fused result's `Key` is in the superseded overlay THE SYSTEM SHALL prefix its `Snippet` with `[SUPERSEDED → <replacement-slug>] ` regardless of the `IncludeSuperseded` setting.
- WHEN a superseded spec hits in both the lexical and vector paths of `retrieveHybrid` THE SYSTEM SHALL apply the de-weight multiplier exactly once (in `fuseRRF`) and the annotation marker exactly once.
- WHEN `retrieveHybrid` invokes the lexical sub-path THE SYSTEM SHALL signal the lexical path to skip its supersede score multiplier (annotation marker still applied) so the de-weight is applied only by `fuseRRF`.
- WHEN a non-hybrid lexical call is made (`SemanticOK=false`) THE SYSTEM SHALL apply the supersede de-weight in the lexical path exactly as the parent spec defines (no change to parent behavior).
- WHEN `hero supersede <old> --by <new>` runs THE SYSTEM SHALL make the new annotation and de-weight observable on the very next `Retrieve` call without any embeddings refresh.
- WHEN the `RetrievalSupersedeRespect` config flag is `false` THE SYSTEM SHALL skip the vector-path overlay entirely and `fuseRRF` SHALL behave exactly as it does today.
- WHILE a vector hit is on a non-spec corpus (`knowledge`, `convention`, `code`, `event`) THE SYSTEM SHALL NOT apply the supersede overlay (overlay only contains spec slugs).
- WHILE no specs in the workspace carry `superseded_by` THE SYSTEM SHALL build an empty overlay and SHALL NOT mutate any result's score or snippet.

## Risks

- **Score calibration drift.** RRF scores at k=60 are small (~0.016 at rank 0; ~0.014 at rank 5). A 0.3× multiplier moves a top vector hit from rank 0 to roughly rank 9 in the fused list when competing against a clean lexical hit at rank 0 — measured against the parent spec's intent, that's the right shape. But if real workloads show superseded specs sinking too far (off the default 20-result page) or not far enough (still in the top 3), the multiplier needs re-tuning. Mitigation: the `supersededDeweight` constant lives in one place and is shared with the parent — tune both together. Add a hybrid-path test that asserts a superseded vector hit ranks below a non-superseded lexical hit of comparable quality.
- **Vector store performance.** The overlay query is `SELECT slug, superseded_by FROM specs WHERE superseded_by != ''` — sub-millisecond on the `specs` table even at thousands of rows; result is a small map. No measurable overhead. Risk is near-zero but worth a benchmark assertion (under 1ms on a 1000-spec workspace).
- **Cold start when `hero supersede` runs before next index.** The parent spec's `hero supersede` flow already inlines `hero index --if-stale -q` (parent design section "`hero supersede` command", step 6). The overlay reads from `specs` immediately after, so there is no staleness window. If the parent's inline index step is dropped, this overlay falls back to whatever `specs.superseded_by` currently holds — which is "the previous state" for as long as the user delays reindexing. Documented in the parent spec; this spec inherits that behavior.
- **Annotation idempotency.** A snippet that already starts with `[SUPERSEDED → ` (e.g. the lexical path annotated it before the result was handed to `fuseRRF`) must not be double-prefixed. Mitigation: in `fuseRRF`'s annotation pass, `strings.HasPrefix(snippet, "[SUPERSEDED → ")` guard.
- **`IncludeSuperseded=true` ambiguity in hybrid.** Parent spec defines `IncludeSuperseded` as "skip de-weight, keep annotation." This spec applies the same rule in `fuseRRF`. The risk is the user expects "include" to mean "show me even the dropped ones" — but parent already chose annotate-don't-drop. We match parent's choice exactly. If parent revisits, this spec follows.
- **RRF score interpretation drift.** `fuseRRF` currently writes the RRF score into `Result.Score` (line 555). After this spec, the post-fusion score is "RRF score × supersede multiplier" — still dimensionless, still rank-meaningful. Callers that print scores see a smaller number for superseded entries, which matches intent. Test that compares numeric scores before/after.
- **Score-distribution surprise for `--semantic` without lexical hits.** If the lexical path returns zero results and only vector hits make it to `fuseRRF`, the overlay still de-weights superseded entries. A workspace where every match is via vector still gets supersede-aware ranking. This is the desired behavior, but it means a user typing a query that only matches semantically will see superseded specs ranked below their replacements even when the replacement only weakly matches. Tested explicitly.

## Changes

- `internal/retrieval/retrieval.go` — add `skipSupersedeDeweight` internal `Query` field; add `Retriever.supersedeRespect` knob loaded from config in `New`; add `loadSupersededOverlay`, `applySupersedeDeweight`, `annotateSuperseded` helpers; gate the per-path de-weight in `retrieveViaNodeIndex` and `retrieveViaFTS` on the new flag (annotation always fires); rewrite `retrieveHybrid` to load the overlay once per query and pass it (plus `IncludeSuperseded`) into `fuseRRF`; extend `fuseRRF` signature to accept the overlay + include flag and apply de-weight + annotation exactly once after merging.
- `internal/config/config.go` — add `EmbeddingsConfig.RetrievalSupersedeRespect *bool` rollback knob + `Config.RetrievalSupersedeRespect()` accessor (default `true`).
- `internal/retrieval/retrieval_test.go` — update existing `fuseRRF` call sites for the new signature; add `TestFuseRRF_SupersedeVectorOnlyHit`, `TestFuseRRF_SupersedeBothPathsNoDoubleAnnotation`, `TestFuseRRF_IncludeSupersededSkipsDeweightKeepsAnnotation`, `TestFuseRRF_NonSpecCorpusUnaffected`, `TestFuseRRF_EmptyOverlayNoMutation`, `TestLoadSupersededOverlay_ReadsFromSpecsTable`, `TestLoadSupersededOverlay_NilDB`, `TestRetrieveHybrid_VectorPathSupersedeAware`, `TestRetrieveHybrid_SupersedeRespectDisabled`.

## Out of Scope

- **Redesigning retrieval, hybrid fusion, or RRF.** This spec adds one overlay; it does not change the fusion algorithm, the k constant, the lexical/vector balance, or the routing logic in `Retrieve`.
- **Changing the embedding model, dimensionality, or training data.** Vectors are unchanged; only ranking on top of them shifts.
- **Re-embedding any spec.** `vec_chunks` rows are untouched by this feature. No background job, no migration.
- **Adding supersede semantics to non-spec corpora.** Conventions, knowledge, code, and event corpora do not carry `superseded_by` today; expanding the overlay to other corpora is a separate decision and likely a separate spec per corpus.
- **Semantic-search ranking improvements unrelated to supersede.** Cross-encoder reranking, MMR diversification, score-distribution normalization, and similar ranking work are out of scope.
- **Vector-store backend changes.** Brute-force cosine scan over `vec_chunks` stays; sqlite-vec / FAISS / etc. migrations are unrelated.
- **Cross-repo supersede in the vector path.** Parent spec defers cross-repo supersede entirely; this spec inherits that deferral.
- **UI surfacing of supersede in `hero search --semantic` output.** The annotation arrives via the snippet (Decision 3), which is enough for both terminal and MCP consumers. Bespoke rendering tweaks per surface are out of scope.
