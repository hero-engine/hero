---
title: "Retrieval Quality — Reranking, Expansion & Feedback Loop"
slug: retrieval-quality
type: initiative
status: planning
tags: [embeddings, retrieval, search, reranking, quality]
created: 2026-06-09
horizon: next
size: large
relations:
  - target: embedded-inference
    kind: extends
  - target: embeddings-index
    kind: supersedes-scope-of
---

## Goal

Raise retrieval precision from "good enough for hybrid RRF" to "reliably
surfaces the right spec on the first try." The current system — Model2Vec static
embeddings + BM25 via FTS5 + Reciprocal Rank Fusion — works at project scale
but has known ceilings: bag-of-words can't distinguish word order, section-level
chunking dilutes long sections, and there's no feedback loop to measure misses.

This initiative is a **menu, not a commitment**. Each child spec is independently
designable and deliverable. The team picks based on measured pain and available
time.

## Prior art

- `embedded-inference` spec (delivered) — current Model2Vec + hybrid RRF system
- `static-embeddings-over-transformer-runtime` decision — why we chose Model2Vec
  over transformers; lists conditions that would reverse it
- `embeddings-superseded-respect` spec (delivered) — supersede de-weighting in
  fuseRRF
- `embeddings-index` spec (superseded) — original transformer-based design;
  explicitly deferred cross-encoder reranking as "possible future"

## Children

| Slug | Title | Priority | Effort | Impact |
|---|---|---|---|---|
| configurable-reranking | Configurable reranking — local model, LLM, or off | P1 | M | ★★★★★ |
| query-expansion | Query expansion — stemming & synonym mapping | P1 | S | ★★★ |
| retrieval-miss-logging | Retrieval miss logging — implicit failure signals | P1 | S | ★★★ |
| weighted-field-embeddings | Weighted multi-field embeddings — title boost | P2 | S | ★★ |
| overlapping-chunks | Overlapping chunk strategy for long sections | P2 | S | ★★ |
| optional-transformer-model | Optional transformer embedding model (ONNX) | P3 | L | ★★★★ |

## Sequencing

```
Phase 1 (parallel, no dependencies):
  ├── configurable-reranking   — highest impact; opens the reranker interface
  ├── query-expansion          — low effort, immediate quality bump
  └── retrieval-miss-logging   — instruments the system for data-driven decisions

Phase 2 (informed by Phase 1 miss data):
  ├── weighted-field-embeddings — if miss data shows title-mismatch pattern
  └── overlapping-chunks        — if miss data shows long-section dilution

Phase 3 (if quality ceiling still reached):
  └── optional-transformer-model — plugs into reranker interface from Phase 1
```

Phase 1 items are independent. Phase 2 items should be prioritized based on
what retrieval-miss-logging reveals. Phase 3 is the escape hatch if Model2Vec
quality becomes the bottleneck — the `static-embeddings-over-transformer-runtime`
decision lists the reversal conditions.

## Cross-cutting concerns

### Config surface

All children extend `hero.json` under the existing `embeddings` key. The config
schema must remain backward-compatible — new fields get sensible defaults that
preserve current behavior (reranker: off, expansion: off, chunking: section).

Current config:
```json
{
  "embeddings": {
    "enabled": true,
    "scope": ["spec", "knowledge", "convention", "event", "code"],
    "model": "hero-embed-v1",
    "retrieval_supersede_respect": true
  }
}
```

New fields will nest under this. Example target state:
```json
{
  "embeddings": {
    "enabled": true,
    "scope": ["spec", "knowledge", "convention", "event", "code"],
    "model": "hero-embed-v1",
    "retrieval_supersede_respect": true,
    "reranker": {
      "provider": "off",
      "model": "",
      "top_k_input": 25
    },
    "query_expansion": {
      "enabled": false,
      "stemming": true,
      "synonyms": true
    },
    "chunking": {
      "strategy": "section",
      "overlap_ratio": 0.5,
      "window_tokens": 200
    }
  }
}
```

### Extension points

All children touch the retrieval pipeline at predictable points:

```
Query text
  → [query-expansion]        new: expand before embedding
  → Embed query (model.go)
  → BM25 + QuerySimilar      existing hybrid paths
  → fuseRRF                  existing fusion
  → [configurable-reranking]  new: rerank fused top-K
  → Result[]
```

Chunking changes (weighted-field-embeddings, overlapping-chunks) affect the
indexing side (chunker.go, refresh.go) and are invisible to the retrieval path.

### Testing strategy

Each child should include a **retrieval quality test**: a set of 10+ query/expected
pairs that assert the correct spec appears in the top-3 results. The embedded-inference
spec established this pattern. New specs should extend the test set, not replace it.

### Binary size budget

The current model adds ~23 MB. The optional-transformer-model spec needs to be
opt-in and downloadable, not embedded, to avoid ballooning the binary. All other
children add negligible binary size (code only).

## Risks

- **Over-tuning to small corpus.** Most Hero projects have <1000 chunks. Quality
  improvements that matter at 100K chunks may be unmeasurable at 200. Miss logging
  (Phase 1) is the check on this.
- **Config complexity creep.** Six new config knobs across all children. Each must
  default to "off" or "current behavior" so zero-config users see no change.
- **Reranker token cost surprise.** LLM reranking burns tokens on every search.
  The spec must surface cost estimates in `hero embeddings status` and warn on
  first enable.

---

## Child spec stubs

### configurable-reranking

```yaml
title: "Configurable Reranking — Local Model, LLM, or Off"
type: feature
status: draft
parent: retrieval-quality
tags: [retrieval, reranking, embeddings]
size: medium
```

**Problem:** Hybrid RRF fusion produces a reasonable ranking, but RRF is a
position-based heuristic — it doesn't understand whether a result actually answers
the query. A cross-encoder or LLM reranker can score (query, document) pairs
directly, dramatically improving precision in the top-3.

**Design direction:**
- Add `Reranker` interface to `internal/retrieval/` with a single method:
  `Rerank(query string, candidates []Result, topK int) ([]Result, error)`
- Three implementations behind config:
  - `OffReranker` — passthrough, current behavior (default)
  - `LocalReranker` — loads a small ONNX cross-encoder model from
    `~/.hero/models/rerankers/<name>/`. Pure Go inference if feasible,
    otherwise document the platform constraint.
  - `LLMReranker` — formats candidates as a prompt, asks the configured LLM to
    rank them. Model configurable (`embeddings.reranker.model`). Must work with
    any model that supports the chat API Hero already uses.
- Config: `embeddings.reranker.provider` ("off" | "local" | "llm"),
  `embeddings.reranker.model`, `embeddings.reranker.top_k_input` (default 25)
- Integration point: called in `retrieveHybrid()` after `fuseRRF()`, before
  returning results. Also callable from `retrieveViaFTS()` when SemanticOK.
- `hero embeddings status` shows reranker config and estimated cost per query
  (for LLM provider).

**Key code paths:** `internal/retrieval/retrieval.go` (retrieveHybrid, fuseRRF),
`internal/config/config.go` (EmbeddingsConfig), `internal/embeddings/storage.go`
(QuerySimilar).

**Relates to:** `static-embeddings-over-transformer-runtime` decision (lists ONNX
as rejected for embeddings but a cross-encoder is a different cost profile —
runs once per query, not once per chunk).

---

### query-expansion

```yaml
title: "Query Expansion — Stemming & Synonym Mapping"
type: feature
status: draft
parent: retrieval-quality
tags: [retrieval, search, embeddings]
size: small
```

**Problem:** Model2Vec is bag-of-words. "auth" and "authentication" produce
different token lookups. Users searching for "config" won't match specs about
"configuration." This is the most common class of near-miss in semantic search.

**Design direction:**
- Add `ExpandQuery(text string) string` to `internal/embeddings/` or
  `internal/retrieval/`.
- Two expansion strategies, both enabled by default:
  - **Porter stemming** — reduce terms to stems ("authentication" → "authent",
    "configured" → "configur"). Append stems alongside originals so both
    signals contribute to the embedding.
  - **Synonym map** — small built-in map of common programming synonyms
    (auth/authentication, config/configuration, db/database, repo/repository,
    err/error, etc.). Expand query with synonyms before embedding.
- Applied before `model.Embed()` in the hybrid retrieval path.
- Config: `embeddings.query_expansion.enabled` (default true once shipped),
  `embeddings.query_expansion.stemming`, `embeddings.query_expansion.synonyms`.
- No impact on indexing side — only query-time expansion.

**Key code paths:** `internal/retrieval/retrieval.go` (retrieveHybrid, before
embed), `internal/embeddings/model.go` (Embed).

---

### retrieval-miss-logging

```yaml
title: "Retrieval Miss Logging — Implicit Failure Signals"
type: feature
status: draft
parent: retrieval-quality
tags: [retrieval, observability, embeddings]
size: small
```

**Problem:** We have no data on where retrieval fails. Without measurement,
improvement prioritization is guesswork.

**Design direction:**
- New table in index.db: `retrieval_events(id, query, results_json, action,
  timestamp)` where action is "selected" | "re-searched" | "abandoned".
- When a search result is opened (via `hero_read_spec` or file open after
  search), log a "selected" event with the query and result key.
- When a user searches again within N seconds of a previous search without
  opening a result, log a "re-searched" event pairing the two queries.
- `hero embeddings diagnostics` — new subcommand that reports: miss rate,
  most common re-search pairs, queries with no selections.
- Privacy: all data local, in index.db. Purged on `hero embeddings rebuild`.
- No telemetry, no remote reporting.

**Key code paths:** `internal/retrieval/retrieval.go` (Retrieve — log query),
`internal/embeddings/storage.go` (new table), `internal/cli/embeddings.go`
(new diagnostics subcommand).

---

### weighted-field-embeddings

```yaml
title: "Weighted Multi-Field Embeddings — Title Boost"
type: feature
status: draft
parent: retrieval-quality
tags: [embeddings, chunking]
size: small
```

**Problem:** Chunk embeddings are computed by prepending metadata as text
("Title: X\nType: Y\nStatus: Z\n\n<body>") and mean-pooling everything equally.
In a long body, the title signal gets diluted. Title is the highest-density
signal for relevance.

**Design direction:**
- Embed title and body separately in `chunker.go` or `refresh.go`.
- Store both vectors, or store a weighted combination:
  `final_vec = normalize(alpha * title_vec + (1 - alpha) * body_vec)`
  where alpha defaults to 0.6 (title-heavy).
- Alpha configurable in hero.json for tuning.
- Schema change: either add a `title_vector` column to vec_chunks, or compute
  the weighted combination at index time (simpler, no schema change).
- Recommend: weighted combination at index time (no query-time change needed).

**Key code paths:** `internal/embeddings/chunker.go` (specMetadataPrefix),
`internal/embeddings/refresh.go` (Refresh — embed step),
`internal/embeddings/model.go` (Embed).

---

### overlapping-chunks

```yaml
title: "Overlapping Chunk Strategy for Long Sections"
type: feature
status: draft
parent: retrieval-quality
tags: [embeddings, chunking]
size: small
```

**Problem:** One vector per spec section. A long `## Design` section gets
compressed into a single mean-pooled vector that averages all topics. If the
section discusses both "auth flow" and "rate limiting," a query about either
topic gets a diluted match.

**Design direction:**
- For sections exceeding a token threshold (e.g., 200 tokens), split into
  overlapping windows: window_size=200 tokens, overlap=50%.
- Each window becomes its own chunk with a section suffix:
  `spec:auth-flow:Design:0`, `spec:auth-flow:Design:1`, etc.
- Short sections (<= window_size) stay as single chunks (no change).
- Config: `embeddings.chunking.strategy` ("section" | "sliding"),
  `embeddings.chunking.window_tokens`, `embeddings.chunking.overlap_ratio`.
- Default: "section" (current behavior) — opt-in to sliding.
- Storage impact: ~2-3x more chunks for long specs. Negligible at project scale.

**Key code paths:** `internal/embeddings/chunker.go` (ChunkSpecs, all corpus
extractors), `internal/embeddings/refresh.go` (Refresh).

---

### optional-transformer-model

```yaml
title: "Optional Transformer Embedding Model (ONNX)"
type: feature
status: draft
parent: retrieval-quality
tags: [embeddings, inference, onnx]
size: large
```

**Problem:** Model2Vec's static embeddings are fast and dependency-free but
fundamentally limited — no word order, no contextual understanding. For projects
that need higher retrieval quality and are willing to accept a dependency, a
small transformer model is a step-change improvement.

**Design direction:**
- Ship as an **opt-in downloadable model**, not embedded in binary.
  `hero embeddings install hero-embed-v2-onnx` downloads to
  `~/.hero/models/embeddings/hero-embed-v2-onnx/`.
- Config: `embeddings.model: "hero-embed-v2-onnx"` — `LoadModelFromConfig`
  already checks filesystem before falling back to embedded.
- Inference: evaluate pure-Go ONNX options (if mature by then) or accept
  platform-specific binaries in `~/.hero/models/`.
- Automatic fallback: if configured model isn't available, warn and fall back
  to hero-embed-v1.
- Also serves as a high-quality local reranker (cross-encoder variant) for
  the configurable-reranking spec.
- **Must not break CGO_ENABLED=0 builds.** The base binary stays pure Go.
  Transformer inference is loaded dynamically or runs as a subprocess.

**Key decisions needed:**
- Which model? (bge-small-en-v1.5, all-MiniLM-L6-v2, or custom distilled)
- Pure Go inference vs. platform binary vs. subprocess?
- Cross-encoder (reranking) vs. bi-encoder (embedding) vs. both?

**Relates to:** `static-embeddings-over-transformer-runtime` decision — its
"conditions that would reverse this" section is the gate for this spec.

**Key code paths:** `internal/embeddings/model.go` (LoadModelFromConfig,
Model interface), `internal/config/config.go` (EmbeddingsConfig).
