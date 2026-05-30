---
title: "Embeddings Index — Semantic Retrieval Over Hero's Project Corpus"
slug: embeddings-index
type: feature
status: superseded
priority: medium
horizon: next
tags: [retrieval, semantic-search, embeddings, ai-services, foundational]
superseded_by: embedded-inference
relations:
  - target: embedded-inference
    kind: superseded-by
  - target: next-compact-handoff
    kind: related
  - target: compact-handoff-summarizer
    kind: related
  - target: local-project-model
    kind: related
  - target: master-ingest-restore
    kind: related
---

# Embeddings Index — Semantic Retrieval Over Hero's Project Corpus

## Problem

Hero accumulates a structured project corpus: specs (planning + completed), knowledge files, decisions, events (UserAsks, NextSuggestions, SessionReflections), conventions, peer-call answers, and (with master-ingest restored) code symbols with descriptions. Today, retrieval against that corpus is **lexical** — `hero search` and `hero_search` use BM25/TF-IDF. That works well for known-keyword queries ("find the spec about auth"), poorly for intent-driven queries ("find decisions related to the user-onboarding question I'm about to ask").

Five concrete cases where lexical retrieval falls short:

1. **Compact handoff context.** When the [compact-handoff-summarizer](../compact-handoff-summarizer/spec.md) reads a transcript, it should ground summaries in the most relevant existing specs/decisions — without us hand-coding "find by slug." The model thinks in concepts; retrieval needs to match concepts.
2. **`/resume` quality.** Today's resume pulls graph-ranked context. Embedding similarity adds a parallel "semantically close to the current branch / recent edits" axis that catches connections lexical can't see.
3. **`/why` synthesis.** Already does multi-hop graph traversal. Embeddings add: "show me decisions and notes semantically related to this function, not just connected by edges."
4. **Peer-call prompt composition.** When the user says "ask hero-code about X," Hero should auto-attach the most relevant local specs/decisions as `--reason` context. That's a retrieval task.
5. **Spec authoring.** When a new spec is being written, "this is 0.84 similar to `auth-refactor` — link as related?" is a real ergonomic upgrade. Lexical can't do it without exact term overlap.

Lexical search also misses **soft matches across vocabulary**: "noob-prompter" / "non-expert user" / "junior dev" are the same concept; BM25 sees three different bags of words.

## Goal

Build a local semantic-retrieval layer over Hero's corpus that downstream features can query via a simple API. Concretely:

- A persistent local index (`.hero/index/embeddings.db`, gitignored) keyed on chunk_id → vector, with metadata for type/source/timestamp.
- A small local embedding model (no network calls for retrieval) producing 384-dim vectors.
- A `hero search --semantic <query>` CLI exposing the layer for humans + a `Retriever` Go interface for downstream features.
- Hybrid retrieval (BM25 + vector similarity via reciprocal-rank fusion) as the default — pure vector misses exact-string lookups for file paths, slugs, type names.
- Incremental updates triggered by content changes, not full rebuilds.

This is **infrastructure**, not a user-visible feature. The win shows up in every downstream surface that queries it. The spec includes one concrete consumer (compact handoff summarizer's prompt augmentation) so the seam is exercised end-to-end at delivery; other consumers come incrementally.

## Design

### Corpora — what gets indexed

Five corpora, each with a different chunk shape:

| Corpus | Chunk unit | Source | Metadata |
|---|---|---|---|
| **Specs** | One chunk per section (`## Problem`, `## Goal`, `## Design`, `## Decisions`, `## Risks`, etc.) | `.hero/planning/**/spec.md`, `.hero/specs/**/spec.md` | spec_slug, section_name, type, status, mtime |
| **Knowledge** | One chunk per file (already written tight) | `.hero/knowledge/**/*.md` | category, mtime |
| **Events** | One chunk per event | Graph `events.log` / `events` table | event_type, session_id, spec_slug, timestamp |
| **Conventions** | One chunk per convention spec | `.hero/knowledge/conventions/**` | convention_slug, scope |
| **Code symbols** *(gated on master-ingest)* | One chunk per function/struct + its docstring | code-scan extraction | path, symbol_name, kind |

Code symbol indexing is gated on the [master-ingest-restore](../master-ingest-restore/spec.md) work — that's where symbol-with-description extraction lives. Until that ships, code is excluded; the index is still useful from the other four corpora.

### Embedding model — local-first

Default: **`bge-small-en-v1.5`** (~130MB, 384-dim, runs on CPU comfortably). Sweet spot of quality vs. footprint vs. speed.

Loaded via [ONNX Runtime](https://onnxruntime.ai/) or a pure-Go alternative ([github.com/sugarme/transformer](https://github.com/sugarme/transformer)). Decision deferred to implementation phase — pick what installs cleanest across macOS/Linux without requiring a system Python.

Configurable via `hero.json` to swap to a different model or point at an HTTP embedding endpoint (Ollama, vLLM, hero-cloud) for users who want bigger/better:

```json
{
  "embeddings": {
    "provider": "local | hero-cloud | http",
    "model": "bge-small-en-v1.5",
    "endpoint": "http://localhost:11434/v1/embeddings",
    "dimension": 384
  }
}
```

Model artifact downloaded on first use by `hero embeddings init`; cached in `~/.hero/models/embeddings/`.

### Storage — sqlite-vec

`.hero/index/embeddings.db` is a SQLite database with the [`sqlite-vec`](https://github.com/asg017/sqlite-vec) extension for vector ops. Why SQLite:

- Hero already has SQLite in `internal/graph/`. No new database dependency.
- `sqlite-vec` is small, fast for the scale Hero operates at (tens of thousands of chunks per project), portable.
- Schema is debuggable with regular SQL.

Schema sketch:

```sql
CREATE TABLE chunks (
  chunk_id TEXT PRIMARY KEY,
  corpus TEXT NOT NULL,          -- 'spec', 'knowledge', 'event', 'convention', 'code'
  source_path TEXT,              -- file path or 'graph:<event_id>'
  source_id TEXT,                -- spec_slug, event_id, convention_slug, symbol_id
  section TEXT,                  -- '## Problem' for specs; null otherwise
  text TEXT NOT NULL,
  text_hash TEXT NOT NULL,       -- sha256 of text for invalidation
  metadata JSON,                 -- corpus-specific fields
  embedded_at TIMESTAMP NOT NULL
);

CREATE VIRTUAL TABLE chunk_vec USING vec0(
  chunk_id TEXT PRIMARY KEY,
  embedding FLOAT[384]
);

CREATE INDEX idx_chunks_corpus ON chunks(corpus);
CREATE INDEX idx_chunks_source ON chunks(source_path, source_id);
```

Storage estimate: ~1.5KB per chunk (text + 384 floats + metadata). 10K chunks ≈ 15MB. 100K chunks ≈ 150MB. Acceptable for `.gitignored` local cache.

### Invalidation — content-hash driven

A chunk is re-embedded when its `text_hash` changes (text edited) or when it's missing from the index (newly created). Stale chunks (source deleted) are pruned.

Triggers for incremental update:

- **Git pre-commit hook** (extends the existing `internal/cli/next_hooks.go` pre-commit) calls `hero embeddings refresh --if-stale`. Cheap, runs in the background of every commit.
- **On `hero check`** — full sweep to catch chunks that drifted between commits.
- **On `hero embeddings rebuild`** — manual full rebuild for recovery / model upgrade.

The "if-stale" path walks the corpus, computes text_hashes, and embeds only what's changed or new. On a project with no changes, this completes in tens of milliseconds.

### Retriever interface (Go)

```go
// Package retrieval is the query-side seam for embeddings + lexical search.
package retrieval

type Retriever interface {
    // Search returns the top-N chunks for a query, ranked by hybrid
    // (BM25 + vector similarity via reciprocal-rank fusion).
    Search(ctx context.Context, q Query) ([]Result, error)
}

type Query struct {
    Text     string
    Corpora  []string          // empty = all
    Filters  map[string]string // e.g. {"spec_slug": "auth-refactor"}
    TopK     int               // default 10
    Strategy Strategy          // hybrid (default) | vector | lexical
}

type Result struct {
    ChunkID     string
    Corpus      string
    SourcePath  string
    SourceID    string
    Section     string
    Text        string
    Score       float64
    Metadata    map[string]any
}
```

Downstream features call this; they don't reach into the embeddings DB directly. The interface stays stable even when the index implementation evolves.

### CLI surface

```
hero embeddings init                   # one-time setup: download model, create DB
hero embeddings refresh [--if-stale]   # incremental update (default for hook)
hero embeddings rebuild                # full re-embed (manual; for model upgrade)
hero embeddings status                 # DB size, chunk counts per corpus, last refresh
hero embeddings query "<question>"     # debug retrieval; prints top-N results
hero search --semantic "<query>"       # user-facing semantic search alongside lexical
```

`hero search` (existing) gains a `--semantic` flag and a `--hybrid` flag (default). Pure-lexical `hero search` keeps working unchanged.

### Hybrid ranking — RRF

[Reciprocal Rank Fusion](https://plg.uwaterloo.ca/~gvcormac/cormacksigir09-rrf.pdf): for each result, score = `1 / (k + rank_in_each_ranking)`, summed across the two rankings. Default `k=60` (the original paper's recommendation, robust across domains).

This is a well-established, no-tuning-required fusion technique. Better than learned re-rankers for our scale and our zero-training-data starting position.

### One concrete downstream consumer at delivery

To exercise the seam end-to-end, the [compact-handoff-summarizer](../compact-handoff-summarizer/spec.md)'s summarization prompt gains a retrieval step:

> Given the transcript, retrieve the top-5 most relevant chunks (specs, decisions, conventions) from the embeddings index. Include them in the user message above the transcript as "Relevant project context: ...".

This validates the interface and demonstrably improves summarization quality on real transcripts where the conversation alludes to decisions or conventions the model has otherwise never seen.

Other consumers (peer-call drafting, `/why` augmentation, spec-relation auto-suggest) come incrementally as separate small specs that just call the `Retriever`.

## Out of scope

- **Training or fine-tuning the embedding model.** We use a pre-trained model as-is. Project-specific adaptation is a separate research question, see [local-project-model](../local-project-model/spec.md).
- **Federated / cross-project embeddings.** Each project has its own index. No global search across projects.
- **Replacing the graph.** The graph is still the source of truth for typed relations (depends_on, blocks, derived_from). Embeddings add a parallel similarity axis, not a replacement.
- **Replacing lexical search.** Hybrid (BM25 + vector) is the default; pure lexical stays available. We don't want to lose exact-string lookup quality.
- **A general LLM gateway.** Embeddings via `hero-cloud` is one path through the existing AIServices seam (see [compact-handoff-summarizer](../compact-handoff-summarizer/spec.md)). We don't build a separate gateway.
- **Re-ranking with a cross-encoder.** Reciprocal-rank fusion is sufficient at our scale. Cross-encoder re-rank is a possible future upgrade if quality drives it.
- **Vector-DB alternatives (Pinecone, Weaviate, Qdrant).** SQLite + sqlite-vec covers the scale we operate at without adding a service dependency.

## Risks

- **Model artifact distribution.** A ~130MB download on first use is a real onboarding tax. Mitigations: lazy-load on first use (not on `hero init`), surface a single-line "downloading embedding model (~130MB, one-time)" message, support `--no-embeddings` flag to skip.
- **Cross-platform inference runtime.** ONNX Runtime needs native libs. Pure-Go transformer libs are slower and have spottier model coverage. Either choice has tradeoffs; needs implementation-phase decision with a small benchmark.
- **Index drift.** If the pre-commit refresh fails silently, the index gradually goes stale. Mitigations: `hero check` includes an embeddings-staleness probe; staleness shown in `hero embeddings status`.
- **Storage growth on large projects.** Code-symbol corpus on a 100K-LOC codebase could push the DB past a few hundred MB. Mitigations: per-corpus opt-out in config, automatic pruning of chunks whose source files were deleted.
- **Embedding model upgrade is a full rebuild.** Switching `bge-small` for a different model invalidates all existing vectors. Mitigations: `hero embeddings rebuild` exists and is fast (re-embedding 10K chunks is minutes, not hours); the `dimension` is in config so mismatched DBs error early instead of silently producing junk results.
- **Quality plateau.** Vector retrieval is genuinely good but not magic. Some queries will return weird results. Hybrid (BM25 + vector) buffers this; pure-vector mode is for debugging/research.
- **Memory footprint of inference.** Embedding model loaded into the CLI process. ~150–250MB resident when active. For short-lived CLI calls this is fine; long-running processes (the MCP server) should load on demand and unload after N seconds idle.
- **Stale embeddings between sessions.** Two sessions on the same project both think they have a current index but one just committed changes; the other's hook hasn't fired yet. Mitigation: `Retriever.Search` checks last-refresh mtime against corpus mtimes and triggers a quick `--if-stale` refresh inline when more than a few seconds out of date.

## Acceptance criteria

- [ ] `hero embeddings init` downloads the configured embedding model to `~/.hero/models/embeddings/<model>/` and creates `.hero/index/embeddings.db` with the schema.
- [ ] `hero embeddings refresh --if-stale` embeds new + changed chunks from all enabled corpora (specs, knowledge, events, conventions) in <2 seconds on a project with no changes.
- [ ] `hero embeddings status` reports DB size, chunk counts per corpus, last refresh timestamp.
- [ ] `hero embeddings query "<text>"` prints top-10 results with corpus, source, snippet, score.
- [ ] `hero search --semantic "<text>"` and `hero search --hybrid "<text>"` return ranked results; `--hybrid` is the default behavior of `hero search`.
- [ ] `retrieval.Retriever` interface exists and is callable from Go; pre-commit hook for `hero embeddings refresh --if-stale` is installed by `hero hooks install`.
- [ ] Reciprocal-rank fusion implementation correct (testable with synthetic dual rankings).
- [ ] Storage estimate within an order of magnitude of spec (~1.5KB/chunk).
- [ ] `.hero/index/embeddings.db` is gitignored.
- [ ] `hero embeddings rebuild` does a full re-embed and can recover from a deleted/corrupted index.
- [ ] Provider abstraction: swapping `local` → `http` (Ollama) in config works without code changes.
- [ ] Code-symbol corpus is gated on `master-ingest-restore` shipping; embeddings work without it from the other four corpora.
- [ ] [compact-handoff-summarizer](../compact-handoff-summarizer/spec.md)'s prompt augmentation includes top-5 retrieved chunks (validates the seam end-to-end).
- [ ] Memory footprint: embedding model lazy-loaded; CLI commands that don't need retrieval don't pay the cost.

## Changes

- `internal/retrieval/` (new package) — `Retriever` interface, hybrid ranking (RRF), provider abstraction.
- `internal/embeddings/` (new package) — model loading (ONNX or pure-Go), chunk extraction per corpus, refresh/rebuild engine.
- `internal/embeddings/storage.go` — sqlite-vec schema, insert/upsert/prune, vector + metadata queries.
- `internal/cli/embeddings.go` — `hero embeddings init/refresh/rebuild/status/query` subcommands.
- `internal/cli/search.go` — `--semantic` / `--hybrid` flags, route to `retrieval.Retriever`.
- `internal/cli/next_hooks.go` — pre-commit hook adds `hero embeddings refresh --if-stale` alongside checkpoint/queue/index.
- `internal/config/embeddings.go` — `embeddings.*` config schema.
- `.gitignore` updates for `.hero/index/embeddings.db`.
- Tests: refresh idempotency, hybrid ranking correctness, schema migrations, provider swap, pre-commit hook integration.

## Kickoff

Build the embeddings index as foundational infrastructure for Hero's "knows your project" features. Start with specs + knowledge + events + conventions; code-symbol corpus comes when master-ingest-restore lands. Use `bge-small-en-v1.5` via the simplest runtime that installs cleanly across macOS/Linux. SQLite + sqlite-vec for storage. Reciprocal-rank fusion for hybrid ranking.

Read first:
- This spec end-to-end.
- [master-ingest-restore](../master-ingest-restore/spec.md) for the code-symbol corpus dependency.
- [compact-handoff-summarizer](../compact-handoff-summarizer/spec.md) — the concrete first consumer.
- `internal/graph/` for SQLite patterns already in use.
- `internal/cli/search.go` (current `hero search`) to understand the existing BM25 path.

Implement in order:

1. Decide ONNX-Runtime vs. pure-Go inference. Bench both with `bge-small` on a representative chunk count; pick the one that installs cleanly across platforms.
2. `internal/embeddings/storage.go` — sqlite-vec schema + insert/upsert/prune. Test with synthetic vectors first; no model dependency yet.
3. Chunk extractors per corpus (specs, knowledge, events, conventions). Test independently.
4. `hero embeddings init/refresh/status/query` CLI; manual smoke against this Hero repo's own corpus.
5. `retrieval.Retriever` interface + RRF hybrid ranking; unit-test ranking with synthetic dual rankings.
6. `hero search --semantic/--hybrid` flag wiring.
7. Pre-commit hook addition for `--if-stale` refresh.
8. Wire the first consumer (compact handoff summarizer prompt augmentation) once that spec ships.

The win is cumulative: every downstream feature that retrieves project context gets sharper. The MVP delivery is ~600–900 LOC plus the model-loading infrastructure (likely the trickiest piece). Start small — get the seam right, then bolt features onto it.
