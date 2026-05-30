---
title: "Static Embeddings Over Transformer Runtime"
type: decision
status: accepted
created: 2026-05-28
tags: [embeddings, inference, architecture, pure-go]
relations:
  - target: embedded-inference
    kind: informs
  - target: embeddings-index
    kind: supersedes-approach-in
---

# Static Embeddings Over Transformer Runtime

## Context

Hero needs semantic retrieval over the project corpus (specs, knowledge, code symbols). The original `embeddings-index` spec assumed a transformer model (bge-small-en-v1.5, 130MB) via ONNX Runtime or a pure-Go transformer library.

## Decision

Use Model2Vec-style static embeddings implemented in pure Go instead of running transformer inference.

## Rationale

We evaluated eight runtime options: ONNX Runtime (purego), llama.cpp (kelindar/search), Ollama, Apple MLX, Candle (Rust FFI), TFLite, Hugot/GoMLX (pure Go transformer), WASM, and Model2Vec (static lookup).

Key factors:

1. **CGo kills cross-compilation.** Hero ships as a single binary across 6+ platform targets. ONNX and llama.cpp bindings either require CGo (platform-specific builds) or ship large shared libraries per platform. Model2Vec is pure Go — `CGO_ENABLED=0` works.

2. **The algorithm is trivially simple.** Model2Vec is a vocabulary lookup table + mean pooling. ~200 lines of Go. Taking on a dependency for this would add more complexity than writing it.

3. **Quality is sufficient in hybrid ranking.** When embeddings are one signal among four (BM25 + vector + graph distance + recency), the per-embedding quality gap between Model2Vec and a full transformer shrinks to near-zero in practice. BM25 catches exact matches; vectors catch conceptual matches; the combination covers both.

4. **Performance is dramatically better.** Sub-millisecond per embedding vs. 20-50ms for a transformer. This makes incremental re-embedding on every file save feasible, not just on commit.

5. **Install story stays clean.** No model download step (8MB weights embedded in binary). No server process. No API key. It just works.

## Tradeoffs accepted

- Lower per-embedding quality than transformer models. Mitigated by hybrid ranking.
- No contextual understanding of word order within a chunk. Mitigated by chunking strategy (function-level, not file-level).
- Model2Vec has no fine-tuning path. Accepted — project-specific adaptation is the `local-project-model` spec's domain.

## Alternatives rejected

- **ONNX Runtime** — best quality, but CGo or platform-specific shared libs
- **Ollama** — 500MB+ install, 2-5 second cold start, process management
- **Pure-Go transformer (Hugot/GoMLX)** — 5x slower than ONNX, still heavy
- **External dependency (kelindar/search)** — purego llama.cpp bindings; good but adding a dependency for something that's 200 lines of Go is wrong

## Conditions that would reverse this

- Evidence that Model2Vec quality is insufficient for code retrieval (measured by the 10-query quality test in the spec)
- Hero projects routinely exceeding 500K chunks where brute-force scan is too slow
- A mature, maintained, CGo-free, cross-platform Go embedding library emerges that's clearly better than owning 200 lines
