---
title: "Embedded Inference — Zero-Dependency Semantic Retrieval for Hero"
slug: embedded-inference
type: feature
status: delivering
priority: P1
horizon: next
tags: [retrieval, semantic-search, embeddings, pure-go, foundational]
created: 2026-05-28
relations:
  - target: embeddings-index
    kind: supersedes
  - target: unified-retrieval-layer
    kind: extends
  - target: compact-handoff-summarizer
    kind: enables
  - target: local-project-model
    kind: related
  - target: master-ingest-restore
    kind: depends-on
mission_alignment: |
  Mission is "AI gets the right context at the right moment." BM25 fails
  when the spec says "authentication flow" but the code calls it
  `validateSessionToken`. Semantic retrieval closes that vocabulary gap so
  every agent session starts with the right code, not just the right
  keywords. Raises the floor for everyone — you don't need to know the
  exact function name to find it.
principles_check: |
  Serves #1 (it just works — no API key, no server, no config to get
  semantic search). Serves #5 (each tool for its job — BM25 for exact
  keywords, vectors for intent, graph for relationships). Pure Go, zero
  deps, single binary — keeps the install story clean.
claimed_by: mcp-agent
claimed_at: 2026-05-29T23:48:12-06:00
---

# Embedded Inference — Zero-Dependency Semantic Retrieval for Hero

## Goal

Add semantic vector search to Hero's retrieval pipeline — implemented as a pure Go static-embedding engine with zero external dependencies, zero CGo, and zero runtime servers. Ships inside the existing binary. No API key required. Works offline.

The embedding layer becomes one signal in a multi-signal ranking function alongside BM25, graph distance, and recency. Every downstream surface that queries project context (`hero context`, `hero relevant`, `/deliver`, `/resume`, `/why`, peer-call prompt composition) gets sharper retrieval without any of them changing their call sites — they already go through the `internal/retrieval/` `Retriever` interface.

## Kickoff

Pick up at: All four phases implemented and tested. The engine, storage, chunking, refresh, RRF fusion, CLI, and scan integration are shipped. Remaining: prepare a real Model2Vec weight file (Python distillation of a base model, export as vocab.txt + weights.bin), install to `~/.hero/models/embeddings/hero-embed-v1/`, and run the 10-query quality validation on this repo.

Read first:
- `internal/embeddings/` — the full package: model.go, storage.go, chunker.go, refresh.go
- `internal/retrieval/retrieval.go` — `retrieveHybrid()` and `fuseRRF()` at the bottom
- `internal/cli/embeddings.go` — `hero embeddings status` and `hero embeddings rebuild`
- `internal/cli/scan.go` ~line 395 — embedding refresh wired into `hero scan`

Next steps:
1. Run Model2Vec distillation (Python, one-time) on `all-MiniLM-L6-v2` → export vocab.txt + weights.bin to `~/.hero/models/embeddings/hero-embed-v1/`
2. Run `hero embeddings rebuild` against this repo to populate vec_chunks
3. Run the 10-query quality validation (AC-10)
4. If quality passes, decide on `//go:embed` vs download-on-first-use for model distribution

## Problem

Hero's retrieval today is lexical — BM25 via FTS5 (Phase A/B of unified-retrieval-layer, shipped). This works well for known-keyword queries but fails on vocabulary gaps:

1. **Code ↔ spec mismatch.** A spec says "retry with exponential backoff" but the code calls it `withExponentialDelay`. A spec says "authentication flow" but the implementation is `validateSessionToken`. BM25 sees zero overlap.

2. **Concept synonyms across the corpus.** "Non-expert user" / "junior dev" / "noob-prompter" are the same concept. "Compact handoff" / "session resume context" / "compaction summary" refer to the same feature. BM25 requires exact term overlap.

3. **Cross-type conceptual links.** A convention about error handling is relevant to a spec about retry logic, but they share no keywords. Graph edges help when someone manually linked them; semantic similarity finds the link automatically.

4. **Code retrieval at scale.** When a project has thousands of functions, `hero context` during `/deliver` needs to find the 5-10 most relevant implementation files for a spec. Grep and BM25 find files that mention the right words. Embeddings find files that *do* the right thing.

The existing `Retriever` already has a `SemanticOK` field that stubs through to BM25. This spec fills in that stub.

## Design

### Why build it vs. use a library

We evaluated ONNX Runtime (purego), llama.cpp (`kelindar/search`), Ollama, Apple MLX, Candle (Rust FFI), TFLite, Hugot/GoMLX, WASM, and Model2Vec. Full analysis in the design session notes.

The decision: **implement Model2Vec-style static embeddings in pure Go.**

Rationale:
- The core algorithm is a vocabulary lookup table + mean pooling — ~200 lines of Go, not a transformer forward pass
- Zero CGo, zero shared libraries, zero platform-specific builds — keeps Hero's single-binary cross-compilation story intact
- Sub-millisecond per embedding — faster than any library option
- Zero runtime memory overhead when not in use (weight matrix is memory-mapped)
- No model download step for users — weights (~30MB) can be embedded in the binary via `//go:embed` or downloaded on first use
- Quality is sufficient when embeddings are one signal in a four-signal ranking function (BM25 + vector + graph + recency)
- The interface (`func Embed(text string) []float32`) is swappable — if transformer-quality embeddings are ever needed, the backend changes without touching callers

A library dependency would be heavier and more complex than the code it replaces. When the thing you need is a lookup table, shipping a runtime is the wrong abstraction.

### Model2Vec — how it works

[Model2Vec](https://github.com/MinishLab/model2vec) distills a sentence transformer into a static embedding table:

1. Run a full transformer model on each token in the vocabulary once
2. Store the resulting vectors as a matrix: `vocab_size × embedding_dim`
3. At inference time: tokenize → look up each token's vector → mean-pool → normalize

No attention layers, no layer norms, no matrix multiplication. The quality comes from the distillation — each token's vector already encodes contextual information baked in during the one-time distillation pass.

Reported quality: outperforms GloVe and BM25 on retrieval benchmarks. Competitive with full transformers for search/similarity within bounded corpora. 50x smaller than the source model, 500x faster.

### Implementation: `internal/embeddings/`

New package. Four files, each independently testable:

#### `model.go` — the embedding engine

```go
package embeddings

// Model holds the static embedding weights and vocabulary.
type Model struct {
    vocab   map[string]int    // token → row index
    weights []float32         // flat matrix: vocab_size × dim (row-major)
    dim     int               // embedding dimension (e.g. 256 or 384)
}

// Load reads the weight matrix and vocabulary from the model directory.
// Weights are memory-mapped for zero-copy access.
func Load(modelDir string) (*Model, error)

// Embed produces a normalized vector for the input text.
// Steps: tokenize → lookup → mean-pool → L2-normalize.
func (m *Model) Embed(text string) []float32

// EmbedBatch embeds multiple texts, reusing tokenization buffers.
func (m *Model) EmbedBatch(texts []string) [][]float32
```

The tokenizer is a WordPiece or unigram tokenizer (depending on the distilled model's vocabulary format). Model2Vec models typically use a simple whitespace + subword split. Implementation: ~150 lines of Go for tokenization, ~50 lines for lookup + pool + normalize.

**Model artifact options (decide during implementation):**

| Option | Size | Install story |
|---|---|---|
| `//go:embed` the weights into the binary | +30MB binary size | Zero download. Just works. |
| Download on first `hero scan` | +0 binary, 30MB download | Smaller binary. One-time fetch to `~/.hero/models/`. |
| Both: ship a small default, download larger on opt-in | Best of both | Recommended. Ship `potion-base-8M` (8MB) embedded; offer `potion-base-32M` as upgrade. |

**Recommended: ship the 8MB model embedded in the binary.** This keeps the download-free install story. 8MB added to a Go binary that's already 30-50MB is acceptable. Users who want higher quality can configure a larger model via `hero.json`.

#### `storage.go` — vector index in SQLite

Vectors are stored alongside the existing index.db (not a separate database). New tables added to the existing migration chain in `internal/index/index.go`:

```sql
-- Chunk metadata
CREATE TABLE IF NOT EXISTS vec_chunks (
    chunk_id   TEXT PRIMARY KEY,
    corpus     TEXT NOT NULL,        -- 'spec', 'knowledge', 'event', 'convention', 'code'
    source_id  TEXT NOT NULL,        -- spec_slug, file path, event_id, symbol_id
    section    TEXT NOT NULL DEFAULT '',  -- '## Problem' for spec chunks; '' otherwise
    text_hash  TEXT NOT NULL,        -- sha256 for invalidation
    vector     BLOB NOT NULL,        -- float32 array as raw bytes
    embedded_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_vec_chunks_corpus ON vec_chunks(corpus);
CREATE INDEX IF NOT EXISTS idx_vec_chunks_source ON vec_chunks(source_id);
CREATE INDEX IF NOT EXISTS idx_vec_chunks_hash ON vec_chunks(text_hash);
```

**Why raw BLOB instead of sqlite-vec?** sqlite-vec would add a CGo dependency (it's a C extension). Storing vectors as BLOBs and computing cosine similarity in Go keeps the zero-dependency story. At Hero's scale (10K-100K chunks), a brute-force scan with SIMD-friendly Go code completes in <10ms. If scale ever demands it, we can add approximate nearest neighbor (HNSW) in pure Go later.

Vector query flow:
1. Embed the query text → query vector
2. `SELECT chunk_id, corpus, source_id, section, vector FROM vec_chunks WHERE corpus IN (?)`
3. Compute cosine similarity in Go (dot product of normalized vectors)
4. Sort by similarity, return top-K

For 10K chunks × 256-dim vectors = ~10MB of vector data. Scanning all of it with a dot product is ~2ms on an M-series Mac. No index needed at this scale.

#### `chunker.go` — corpus-specific chunk extraction

Each corpus type has a chunk function:

| Corpus | Chunk unit | Source |
|---|---|---|
| **Specs** | One chunk per section (`## Problem`, `## Goal`, `## Design`, etc.) | `.hero/planning/**/spec.md`, `.hero/specs/**/spec.md` |
| **Knowledge** | One chunk per file (already written tight) | `.hero/knowledge/**/*.md` |
| **Conventions** | One chunk per convention file | `.hero/knowledge/conventions/**` |
| **Events** | One chunk per event (UserAsk, SessionReflection, NextSuggestion) | Graph events table |
| **Code symbols** | One chunk per function/type + its docstring | `codescan` tree-sitter extraction |

Code symbol chunking reuses `internal/codescan`'s existing `FileInfo.Symbols` — each symbol's signature + docstring + first N lines of body becomes a chunk. The tree-sitter infrastructure already extracts this; we just format it for embedding.

Chunk text format (for code):
```
// Package: internal/retrieval
// File: retrieval.go
// Function: Retrieve
// Signature: func (r *Retriever) Retrieve(q Query) ([]Result, error)
//
// Returns ranked results for q. Routes based on query fields:
// filters/types → FTS5, plain text → node index BM25, semantic → vector.
```

This gives the embedding model both structural context (package, file, signature) and semantic content (what the function does).

#### `refresh.go` — incremental update engine

```go
// Refresh walks the enabled corpora, computes text hashes, and
// re-embeds only chunks whose content changed or are missing.
// Prunes chunks whose source was deleted.
func Refresh(heroDir string, model *Model, db *index.DB) (*RefreshStats, error)

type RefreshStats struct {
    Added   int
    Updated int
    Pruned  int
    Skipped int // unchanged
    Elapsed time.Duration
}
```

Triggers:
- **`hero scan`** — after graph population and FTS5 projection, run `Refresh`. This is the primary indexing path.
- **Pre-commit hook** — `hero embeddings refresh --if-stale` (checks mtimes against last refresh timestamp; skips entirely if nothing changed).
- **`hero embeddings rebuild`** — manual full re-embed for recovery.

The `--if-stale` path is designed to be <50ms when nothing changed (mtime check only, no embedding calls).

### Hybrid ranking — wiring into `Retriever`

The existing `Retriever.Retrieve()` method in `internal/retrieval/retrieval.go` already has a `SemanticOK` field on `Query`. Today it's a no-op stub (line 100: "SemanticOK stub: vector search is not yet implemented (Phase C). Fall through to graph/FTS5.").

Change: when `SemanticOK=true` and the vector index has chunks:

1. Run the BM25 path (existing) → lexical results with ranks
2. Embed the query → query vector
3. Scan `vec_chunks` → vector results with cosine similarity scores
4. Fuse via **Reciprocal Rank Fusion**:

```go
// RRF: for each result, score = sum over rankings of 1/(k + rank)
// k=60 per the original Cormack et al. paper.
func fuseRRF(lexical, vector []Result, k int) []Result
```

5. Return fused results with `Source: "hybrid"`

Default behavior: `hero search` gains `--hybrid` (default when embeddings are available) and `--semantic` (vector-only, for debugging). `hero context` and `hero relevant` set `SemanticOK=true` automatically.

**Four-signal scoring** (future refinement, not in MVP):

```
Score = w1×BM25 + w2×vectorSim + w3×graphProximity + w4×recency
```

MVP ships with RRF (rank-based fusion) which doesn't require weight tuning. Signal-weighted scoring is an optimization pass once we have retrieval quality telemetry.

### Configuration

New section in `hero.json`:

```json
{
  "embeddings": {
    "enabled": true,
    "scope": ["spec", "knowledge", "convention", "event", "code"],
    "model": "potion-base-8M"
  }
}
```

- `enabled`: defaults to `true` once the feature ships. Set `false` to disable.
- `scope`: which corpora to embed. `"code"` is gated on tree-sitter/codescan being populated.
- `model`: which Model2Vec distilled model to use. `potion-base-8M` is embedded in the binary. Larger models require a one-time download.

No provider abstraction in MVP. The pure-Go engine is the only backend. If someone later wants cloud or Ollama embeddings, we add a `provider` field behind the same `Embed()` interface — but we don't build that plumbing until there's demand.

### CLI surface

```
hero embeddings status              # chunk counts per corpus, index size, last refresh
hero embeddings rebuild             # full re-embed (model upgrade, recovery)
hero search --semantic "<query>"    # vector-only search (debugging)
hero search --hybrid "<query>"      # BM25 + vector fused (default when available)
```

`hero embeddings refresh` is not a user-facing command — it's called internally by `hero scan` and the pre-commit hook. No `hero embeddings init` — the model is embedded in the binary, so there's no setup step.

### Model artifact preparation (one-time, pre-release)

Before this ships, we need a Model2Vec weight file in a Go-consumable format:

1. Run Model2Vec distillation (Python, one-time) on a good base model (e.g., `all-MiniLM-L6-v2` → `potion-base-8M`)
2. Export: vocabulary as newline-delimited text, weights as raw `float32` binary blob
3. Commit both to the repo under `models/embeddings/potion-base-8M/` with `//go:embed` directives
4. Write a test that loads the model, embeds a known sentence, and asserts the vector matches the Python reference output within floating-point tolerance

This is a build-time step, not a runtime step. Users never touch it.

## Supersedes

This spec supersedes [`embeddings-index`](../embeddings-index/spec.md) (draft). Key differences:

| Aspect | `embeddings-index` (old) | `embedded-inference` (this) |
|---|---|---|
| Model | `bge-small-en-v1.5` (full transformer, 130MB) | Model2Vec static lookup (8-30MB) |
| Runtime | ONNX Runtime or pure-Go transformer lib (TBD) | Pure Go, ~200 lines, no external dep |
| CGo | Likely (ONNX) or slow (pure-Go transformer) | None |
| Storage | Separate `embeddings.db` + `sqlite-vec` | Vectors in existing `index.db` as BLOBs |
| Binary impact | External model download required | 8MB model embedded in binary |
| Cold start | 200-500ms model load | <5ms (memory-mapped weight matrix) |
| Inference speed | ~50 embeddings/sec (transformer) | Thousands/sec (table lookup) |
| Quality | Higher per-embedding | Lower per-embedding, mitigated by hybrid ranking |
| Install story | `hero embeddings init` + 130MB download | Zero steps. It just works. |

The old spec's Retriever interface, chunking strategy, hybrid ranking approach, and acceptance criteria structure are preserved — the change is purely in the embedding engine underneath.

## Out of scope

- **Local generation/summarization.** Cloud LLM is already in the workflow for these tasks. If local generation is ever needed, it's a separate spec building on `local-project-model`.
- **Transformer-quality embeddings.** Model2Vec quality is sufficient for hybrid retrieval at project scale. If evidence shows otherwise, we add an optional transformer backend behind the same interface.
- **Cross-project embeddings / federation.** Each project has its own index. Cross-project search is a graph-federation problem, not an embedding problem.
- **Approximate nearest neighbor (HNSW/IVF).** Brute-force scan is <10ms at 100K chunks. ANN is premature optimization.
- **Fine-tuning the embedding model on project data.** Static weights, no training. See `local-project-model` for the research-phase exploration.
- **Re-ranking with a cross-encoder.** RRF is sufficient at our scale.
- **Provider abstraction (Ollama, cloud).** Ship the pure-Go engine. Add provider swapping when there's demand.

## Risks

| Risk | Impact | Likelihood | Mitigation |
|---|---|---|---|
| Model2Vec quality insufficient for code retrieval | Semantic search returns poor results; users disable it | Medium | Hybrid ranking (BM25 + vector) ensures worst case = same as today. Quality is measurable — test against 10 known queries. If bad, swap to a larger distilled model before shipping. |
| 8MB binary size increase | Packaging/download concerns | Low | Go binaries are already 30-50MB. 8MB is <20% increase. Acceptable. |
| Tokenizer edge cases | Mismatched tokenization vs. reference → wrong vectors | Medium | Port the exact tokenizer from the Python Model2Vec reference. Test against known input→output pairs. |
| Brute-force scan too slow at 100K+ chunks | Query latency exceeds 50ms | Low | At 256-dim, 100K dot products = ~4ms. 500K is the realistic ceiling where ANN becomes needed. Hero projects rarely reach 500K chunks. |
| Weight matrix format changes upstream | Need to re-export weights | Low | We commit a frozen snapshot. Upstream changes don't affect us. |
| Index corruption on crash during refresh | Stale or partial vectors | Medium | Wrap refresh in a SQLite transaction. Incomplete refresh rolls back. `hero embeddings rebuild` recovers from any state. |

## Acceptance Criteria

**AC-1 — Pure Go embedding engine:**
WHEN `embeddings.Model.Embed("authentication retry logic")` is called THE SYSTEM SHALL return a normalized float32 vector of the configured dimension, matching the Python Model2Vec reference output within ±1e-5 per element.

**AC-2 — Zero external dependencies:**
THE SYSTEM SHALL build the embedding engine with `CGO_ENABLED=0`. No shared libraries, no ONNX runtime, no model download step for the default model.

**AC-3 — Vector storage in existing index.db:**
WHEN `hero scan` completes on a project with `embeddings.enabled=true` THE SYSTEM SHALL populate `vec_chunks` with one row per chunk, including corpus type, source ID, text hash, and vector blob.

**AC-4 — Incremental refresh:**
WHEN a spec's `## Problem` section is edited and `hero scan` runs THE SYSTEM SHALL re-embed only the changed chunk (matched by text_hash), leaving all other chunks untouched. Refresh on a project with no changes SHALL complete in <100ms.

**AC-5 — Hybrid retrieval via RRF:**
WHEN `Retriever.Retrieve(Query{Text: "retry with backoff", SemanticOK: true})` is called THE SYSTEM SHALL return results fused from BM25 and vector rankings via Reciprocal Rank Fusion, with `Source: "hybrid"` on fused results.

**AC-6 — Graceful degradation:**
IF `embeddings.enabled` is false or `vec_chunks` is empty THEN THE SYSTEM SHALL fall through to BM25-only retrieval (existing behavior). No error, no user-visible change.

**AC-7 — CLI surface:**
WHEN `hero search --semantic "query"` is run THE SYSTEM SHALL return vector-only results. WHEN `hero search --hybrid "query"` is run THE SYSTEM SHALL return RRF-fused results. WHEN `hero embeddings status` is run THE SYSTEM SHALL print chunk counts per corpus, total index size, and last refresh timestamp.

**AC-8 — Code symbol embedding (gated):**
WHERE `master-ingest-restore` has populated codescan symbols AND `"code"` is in `embeddings.scope` THE SYSTEM SHALL embed function-level chunks with signature + docstring + body prefix as chunk text.

**AC-9 — Pre-commit refresh:**
WHEN the pre-commit hook fires THE SYSTEM SHALL run `embeddings refresh --if-stale`, completing in <100ms when no files changed.

**AC-10 — Retrieval quality validation:**
THE SYSTEM SHALL include a test that runs 10 real queries from this repo's history against both BM25-only and hybrid retrieval, asserting that hybrid returns the expected target spec/code in the top-5 results for at least 8 of 10 queries.

## Changes

| File | Change |
|---|---|
| `internal/embeddings/model.go` | **New.** Model2Vec engine: Load, Embed, EmbedBatch. Tokenizer. Weight matrix memory-map. |
| `internal/embeddings/model_test.go` | **New.** Reference vector comparison against Python output. Tokenizer edge cases. |
| `internal/embeddings/storage.go` | **New.** vec_chunks schema, upsert, prune, cosine-similarity query. |
| `internal/embeddings/storage_test.go` | **New.** Round-trip insert/query, hash-based skip, prune orphans. |
| `internal/embeddings/chunker.go` | **New.** Corpus-specific chunk extraction (specs, knowledge, conventions, events, code). |
| `internal/embeddings/chunker_test.go` | **New.** Chunk extraction per corpus type. |
| `internal/embeddings/refresh.go` | **New.** Incremental refresh engine with RefreshStats. |
| `internal/embeddings/refresh_test.go` | **New.** Idempotency, changed-only re-embed, prune deleted. |
| `internal/index/index.go` | Add `vec_chunks` table + indexes to migration chain. |
| `internal/retrieval/retrieval.go` | Fill `SemanticOK` stub: vector query + RRF fusion. Add `fuseRRF()`. |
| `internal/retrieval/retrieval_test.go` | RRF correctness with synthetic rankings. Hybrid vs. BM25-only on real queries. |
| `internal/cli/search.go` | `--semantic` and `--hybrid` flags. |
| `internal/cli/embeddings.go` | **New.** `hero embeddings status` and `hero embeddings rebuild` commands. |
| `internal/scan/scan.go` | Call `embeddings.Refresh()` after graph + FTS5 projection. |
| `internal/config/config.go` | `Embeddings` struct: Enabled, Scope, Model. |
| `models/embeddings/potion-base-8M/` | **New.** `vocab.txt` + `weights.bin` with `//go:embed`. |
| `.gitignore` | (No change needed — vec_chunks lives in index.db which is already gitignored.) |

## Phasing

### Phase 1 — Engine + storage (~3 days)

Build `internal/embeddings/model.go` and `storage.go`. Test against Python reference vectors. No integration with the rest of Hero yet — just the pure embedding engine and SQLite storage, proven correct in isolation.

### Phase 2 — Chunking + refresh (~2 days)

Build `chunker.go` and `refresh.go`. Wire into `hero scan`. Verify incremental refresh works: add a spec, edit a spec, delete a spec — check that vec_chunks reflects each change correctly.

### Phase 3 — Retrieval integration (~2 days)

Fill the `SemanticOK` stub in `retrieval.go`. Implement `fuseRRF`. Wire `--semantic` and `--hybrid` flags into `hero search`. Run the 10-query quality validation test.

### Phase 4 — Polish + pre-commit hook (~1 day)

`hero embeddings status` command. Pre-commit hook wiring. Config schema. Documentation in help text.

**Total estimate: ~8 days.**

Code symbol embedding (AC-8) ships whenever `master-ingest-restore` populates the symbols — no additional work beyond the chunker already handling the `"code"` corpus type.
