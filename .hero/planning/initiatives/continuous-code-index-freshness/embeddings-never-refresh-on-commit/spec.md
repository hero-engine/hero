---
title: "Semantic embeddings never refresh on commit"
slug: embeddings-never-refresh-on-commit
type: bug
status: planning
domain: engineering
size: medium
priority: high
severity: high
horizon: now
created: 2026-07-25
parent: continuous-code-index-freshness
depends-on:
  - incremental-code-graph-refresh
conflicts-with:
  - incremental-code-graph-refresh
root_cause_class: code
tags: [embeddings, retrieval, semantic-search, staleness, incremental, code-index]
---

# Semantic embeddings never refresh on commit

## Issue

Hero's vector index updates only when a user runs `hero scan` or
`hero embeddings rebuild`. The automatic trigger designed by
`embedded-inference` AC-9 was never delivered, so hybrid retrieval combines a
fresh lexical leg with stale semantic chunks and gives no indication that
recent code or specs are absent.

This child repairs the embedding engine and CLI integration seam. Hook
activation, freshness rows, status rendering, and commit/merge end-to-end
tests belong exclusively to `hook-driven-index-freshness`.

## Investigation

### Reproduction evidence

The original 2026-07-25 investigation measured this repository:

```text
Embeddings enabled: true
Model: hero-embed-v1
Scope: [spec knowledge convention event code]

Index stats:
  Total chunks: 6912
  code:          2927
  knowledge:       45
  spec:          3940
```

Direct `vec_chunks` inspection showed every corpus capped at the same manual
scan timestamp:

```text
code       | 2927 | 2026-07-11T23:01:02Z | 2026-07-16T21:33:00Z
knowledge  |   45 | 2026-05-30T15:04:06Z | 2026-07-16T21:33:00Z
spec       | 3940 | 2026-05-30T15:04:04Z | 2026-07-16T21:33:00Z
```

HEAD was 2026-07-25: nine days, 76 commits, 615 changed files, and 230 changed
Go files later. Newly created Go files and specs had no vector chunks.
`graph.db` was equally stale: current `Symbol` rows had
`MAX(ingested_at)=2026-07-16T21:32:58Z`, and new source files produced no
symbols. Embedding refresh alone could therefore only re-embed an old code
graph. `incremental-code-graph-refresh` fixes that prerequisite.

### Confirmed code flow

`embeddings.Refresh` in `internal/embeddings/refresh.go` has exactly two
manual callers:

- `internal/cli/embeddings.go:runEmbeddingsRebuild`
- `internal/cli/scan.go:writeCodeSubgraph`

The engine then:

1. selects corpus extractors from configured scope;
2. extracts all chunks for a corpus;
3. computes `textHash`;
4. calls `model.Embed` unconditionally;
5. asks `Storage.Upsert` to compare the stored hash and discard the vector if
   unchanged;
6. calls `PruneCorpus` for every extracted corpus.

This ordering makes unchanged writes cheap but leaves unchanged embedding
computation expensive. It also conflates three different states:
authoritative empty, source unavailable (for example `graphDB == nil`), and a
partial/deadline-truncated pass.

### Measured cost

Measurements from the original diagnosis used a cloned workspace and a binary
built from `./cmd/hero`:

| Operation | Result |
|---|---|
| Full rebuild, 7,512 extracted chunks | 1.606s |
| First scan after clone, embedding step | 940ms |
| Steady-state full embedding pass | 571–578ms |
| Code-only unchanged pass | 102–103ms |
| Spec-only unchanged pass | 424–425ms |
| Convention-only extraction walk | 49ms |
| Warm whole-process embeddings status | ~23ms |

The model is not the steady-state bottleneck. Hashing/embedding unchanged spec
chunks accounted for roughly 375ms of the 424ms spec-only pass. The stored
hash comparison occurs too late.

### Prior requirement evidence

`.hero/specs/embedded-inference/spec.md` is completed even though its
Completion Ledger recorded:

> AC-9 | Pre-commit hook calls embeddings refresh --if-stale | PARTIAL

The ledger accurately stated that `internal/cli/next_hooks.go` contained no
embedding call. The predecessor `embeddings-index` spec described the same
automatic refresh. This bug is therefore a concrete delivery gap, not a newly
invented feature.

### Root cause

The direct code defect has four parts:

1. no automatic incremental orchestration calls embedding refresh;
2. `Refresh` embeds before consulting stored hashes;
3. storage writes are per-chunk rather than a corpus transaction;
4. extraction lacks an explicit authoritative/completion result, so an
   unavailable source can be mistaken for an empty keep-set and pruned.

The process defect—completion despite a PARTIAL ledger row—is real but belongs
to the separate ledger-governance work and is not changed here.

### Severity

Severity is high. Semantic retrieval is one of hybrid search's two ranking
legs, and stale chunks fail silently: results still look plausible while being
blind to recent and deleted code. Every agent relying on `hero ask`, hybrid
search, or injected retrieval context inherits the skew.

## Goal

Make configured embedding refresh safe and cheap enough for the shared
incremental code command: unchanged chunks are hash-skipped before
`model.Embed`, changed writes are batched transactionally, code refresh runs
only after child 1 has made `graph.db` authoritative, and pruning happens only
for a fully extracted authoritative corpus. Quiet/deadline failures never
turn a partial pass into deletion.

## Kickoff

Fixes the vector engine behind the continuous code-index pipeline: hash before
embed, batch transactional writes, explicit extraction authority, and
configured-scope refresh including `code`.

**Status:** planning — root cause and measurements are preserved; hook/check
ownership has moved to child 3.

**Pick up at:** change the storage/refresh contract so known hashes are read
before embedding and each corpus extraction reports complete, authoritative,
unavailable, or partial.

→ `.hero/planning/initiatives/continuous-code-index-freshness/embeddings-never-refresh-on-commit/spec.md`

**Files:** `internal/embeddings/refresh.go`,
`internal/embeddings/storage.go`, `internal/embeddings/chunker.go`,
`internal/cli/embeddings.go`, `internal/cli/scan.go`
**Skip:** hook templates, `hero check`, status rendering, event chunker fixes,
search-time writes, and file-level embedding scope.

## Problem

Child 1 makes `graph.db` authoritative for the configured source tree, but the
embedding layer can still waste most of its no-change budget and destroy valid
chunks when sources are unavailable. The previous bug design also hard-coded
`spec|knowledge|convention` and explicitly left `code` scan-only, contradicting
the initiative goal.

The corrected design keeps corpus-level configured scope. It does not add
file-level scope. The shared incremental command runs code graph
reconciliation and lexical projection first, then refreshes every enabled
configured embedding corpus, including `code`.

## Approach

### Hash before embed

Extend `internal/embeddings/storage.go` with a batched hash lookup by chunk ID.
For each fully extracted corpus, `Refresh` computes text hashes, compares them
with the stored map, and calls `model.Embed` only for missing or changed
chunks. It still builds the complete keep-set from every extracted chunk.

Keep `Storage.Upsert` as a one-chunk compatibility API implemented through the
same internal batch semantics. `RefreshStats` reports added, updated, skipped,
pruned, corpus outcomes, and elapsed time accurately.

### Transactional changed writes

Write the changed chunks for one corpus in one transaction using prepared
statements. A failed batch rolls back that corpus's additions/updates. The
deadline/context is used by SQL context APIs so a contended statement does not
silently continue mutating after the caller has returned.

Pruning is in the same corpus transaction and occurs after successful changed
writes only when that extraction is authoritative and complete. A rolled-back
corpus remains at its prior valid generation.

This child may chunk batched hash reads to remain below SQLite variable limits.
It does not redesign the existing general `PruneCorpus` keep-set SQL beyond
what transactional refresh requires; broader unbounded-parameter cleanup
remains deferred.

### Explicit extraction authority

Replace ambiguous `([]TextChunk, error)` use inside refresh with a narrow
result carrying:

- chunks;
- corpus;
- `Complete`;
- `Authoritative`;
- unavailable/partial reason.

Existing chunker functions remain the extraction implementations and may be
wrapped rather than duplicated. Filesystem corpora are authoritative only
after their discovery/read completes without ignored failures. `event` and
`code` are unavailable when `graphDB` cannot be opened. An authoritative empty
result is distinct from unavailable.

Pruning rules:

- complete + authoritative + within deadline: prune absent IDs, including an
  empty corpus;
- unavailable, errored, deadline-expired, source-generation-changed, or
  partial: no prune;
- a corpus whose changed-write transaction fails: rollback and no prune.

This replaces the old blanket “delta mode never prunes” rule. After child 1
reconciles deleted symbols from `graph.db`, a complete code extraction can
safely prune their vector chunks.

### Shared incremental command integration

Add a reusable embedding phase in `internal/cli/embeddings.go` and call it from
the child-1 incremental orchestration seam in `internal/cli/scan.go` after
`codescan.WriteGraph` and `ProjectGraphNodes` succeed but before the scan
cache/checksum generation commits.

The phase uses `cfg.EmbeddingsScope()`, not a hard-coded hook scope. When
embeddings are disabled or the model is unavailable, it returns an explicit
skipped/unavailable outcome without panicking. The code corpus runs only after
the same coordinator has completed the current code graph phase.

On a structurally unchanged source tree, reuse the already-authoritative graph
and still run hash/coverage checks for configured corpora. This lets a prior
deadline, unavailable model, or locked database catch up on the next
commit/merge instead of being hidden forever by equal code checksums.

Expose `hero embeddings refresh --if-stale --deadline <duration> -q` for
focused manual repair and tests, but keep hook activation on the shared
incremental code command so graph→embedding success ordering is preserved.
One aggregate deadline/context from the incremental coordinator is passed
through all phases rather than restarting a separate per-phase budget.

Quiet mode emits no stdout/stderr and is non-blocking for hook use. Internally,
phase outcomes remain truthful so the coordinator does not advance scan state
or run dependent destructive work after a failure. The Cobra layer may
normalize quiet hook exit to zero; child 3 additionally retains `|| true` in
the managed shell block.

### Per-corpus storage metadata

Extend `Storage.Stats` to return, for every corpus:

- count;
- newest `embedded_at`;
- successful extraction outcome/generation when available.

This child owns the storage/query contract. Child 3 owns
`hero embeddings status` rendering and `hero check` coverage diagnosis.
Timestamps are observability, not freshness proof; actual coverage uses source
IDs and text hashes.

## Changes

1. Update `internal/embeddings/refresh.go`.
   - Add nil-model and context/deadline guards.
   - Read hashes before embedding.
   - Track explicit corpus extraction outcomes.
   - Prune only authoritative, complete corpus transactions.
   - Return structured per-corpus stats.
2. Update `internal/embeddings/storage.go`.
   - Add bounded batched hash reads and transactional changed writes.
   - Keep single-chunk `Upsert` compatibility through shared semantics.
   - Keep writes and authoritative prune in one corpus transaction.
   - Extend stats data with per-corpus count/newest timestamp.
3. Update `internal/embeddings/chunker.go`.
   - Wrap current chunkers with explicit complete/authoritative/unavailable
     results.
   - Distinguish unavailable `graph.db` from authoritative empty code/event.
   - Do not fix event property keys in this child.
4. Update `internal/cli/embeddings.go`.
   - Add the focused refresh CLI and reusable phase function.
   - Honor configured embedding enablement, model, and scope.
   - Implement quiet/deadline behavior without hiding phase outcome from the
     shared coordinator.
5. Update the child-1 integration seam in `internal/cli/scan.go`.
   - Run embeddings only after authoritative code graph reconciliation and FTS
     projection.
   - Pass one aggregate deadline/context.
   - Commit scan cache/checksum state only after configured embedding phases
     required by the coordinator complete or return an intentional skipped
     outcome.
6. Extend `internal/embeddings/refresh_test.go`,
   `storage_test.go`, `chunker_test.go`, and add focused CLI tests.
   - Falsify unconditional embed calls with a counting model.
   - Cover transaction rollback, nil model, unavailable source, authoritative
     empty, deadline/lock contention, code deletion prune, configured scope,
     and per-corpus stats.

## Boundaries

- No edits to `internal/cli/next_hooks.go`, hook tests,
  `internal/cli/check.go`, or end-to-end git-hook fixtures.
- Child 3 owns `hero embeddings status` human/JSON rendering; this child only
  supplies storage stats.
- No hard-coded hook corpus set. Use configured corpus scope.
- No file-level extension to embedding `scope []string`.
- No inline refresh from retrieval/search/MCP read paths.
- No watcher, daemon, post-commit hook, or harness session hook.
- No event chunker key fix, chunk-ID collision investigation, general
  unbounded SQL cleanup, or ledger-gate fix.

## Risks

| Risk | Mitigation / rollback |
|---|---|
| A partial extraction prunes valid chunks. | Authority/completion is explicit and tested; prune shares the successful corpus transaction. Roll back the automatic call and run `hero embeddings rebuild` from authoritative sources. |
| Database contention exceeds the hook budget. | Use context-aware SQL inside one aggregate deadline and one transaction per corpus. Quiet failure leaves the prior corpus generation valid. |
| Graph refresh fails but code embedding still runs. | Invoke embeddings inside the same coordinator only after child 1 reports graph+FTS success; do not chain independent success-obscuring shell commands. |
| Configured code scope is empty/disabled. | Disabled corpora are skipped without pruning. Code is authoritative-empty only after a successful current graph extraction. |
| Initial catch-up exceeds the deadline. | Partial/deadline passes do not prune or advance the successful generation. A manual `hero scan`/rebuild remains the bootstrap and rollback path. |
| More precise stats change output assumptions. | Child 3 owns rendering compatibility tests; keep the storage API deterministic and sorted at the CLI boundary. |

## Validation

- Focused: `go test ./internal/embeddings ./internal/cli`
- Full: `go test ./...`
- Prove a second unchanged refresh makes zero model `Embed` calls.
- Hold an index write lock past the deadline and assert the corpus transaction
  rolls back, prints nothing under quiet mode, and performs no prune.
- Delete a code symbol, complete child-1 graph reconciliation, refresh
  configured code embeddings, and assert the deleted chunk is gone.
- Remove/unopen `graph.db` and assert existing code chunks remain untouched.
- Run `git diff --check`.

## Acceptance Criteria

- **AC-1:** WHEN `Refresh` processes a chunk whose stored `text_hash` matches current text THE SYSTEM SHALL skip `model.Embed` entirely and count the chunk as skipped
- **AC-2:** WHEN the shared incremental code command has a current authoritative graph, including a structurally unchanged retry, THE SYSTEM SHALL refresh every enabled corpus in `cfg.EmbeddingsScope()`, including `code`, and report `added=0 updated=0 pruned=0` for an unchanged current corpus
- **AC-4:** WHEN extraction finishes completely from an authoritative corpus within the deadline THE SYSTEM SHALL prune absent chunk IDs in the same corpus transaction, including deleted code-symbol chunks, and SHALL NOT prune on partial, unavailable, errored, or deadline-truncated extraction
- **AC-5:** IF the aggregate deadline expires, the model is unavailable, the index is locked, or extraction fails THEN THE SYSTEM SHALL emit no output under `-q`, preserve the prior corpus transaction and successful generation, and leave git-facing invocation non-blocking
- **AC-6:** IF `Refresh` is called with a nil model THEN THE SYSTEM SHALL return a descriptive unavailable/error outcome rather than dereferencing it
- **AC-7:** WHEN `Refresh` writes changed chunks THE SYSTEM SHALL batch stored-hash reads and commit additions, updates, and any authoritative prune in one transaction per corpus
- **AC-9:** WHEN embedding storage stats are queried THE SYSTEM SHALL return newest `embedded_at` and chunk count per corpus for child 3 to render
- **AC-10:** IF an extractor cannot reach its source or cannot prove a complete traversal THEN THE SYSTEM SHALL mark that corpus unavailable or partial and preserve every existing chunk in it
