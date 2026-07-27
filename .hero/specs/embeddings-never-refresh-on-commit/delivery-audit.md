# Delivery audit — embeddings-never-refresh-on-commit

**Audited:** `git diff 592cb99 -- .hero/planning/initiatives/continuous-code-index-freshness/embeddings-never-refresh-on-commit/spec.md internal/cli/embeddings.go internal/cli/scan.go internal/embeddings/chunker.go internal/embeddings/chunker_test.go internal/embeddings/refresh.go internal/embeddings/refresh_test.go internal/embeddings/storage.go internal/embeddings/storage_test.go`; untracked `internal/cli/embeddings_test.go` inspected with `git diff --no-index /dev/null internal/cli/embeddings_test.go`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] AC-1: matching hashes skip embedding and count as skipped — `internal/embeddings/refresh.go:124-171` reads stored hashes before calling `Embed`; `TestRefresh_HashMatchSkipsEmbed` asserts an unchanged rerun makes no additional embed calls and reports one skipped chunk.
- [✓] AC-2: the shared incremental command refreshes configured scope on changed and structurally unchanged code — `internal/cli/scan.go:453-465` refreshes embeddings on unchanged retries, and `internal/cli/scan.go:508-550` orders graph projection before configured-scope embedding and state commit; `TestIncrementalCodeRefreshEmbeddingsConfiguredScopeUnchangedAndDelete` covers `code` plus `knowledge` and asserts unchanged code has zero adds, updates, and prunes.
- [✓] AC-4: authoritative complete extraction prunes atomically while unsafe extraction does not prune — `internal/embeddings/refresh.go:97-179` reconciles only complete authoritative results; `internal/embeddings/storage.go:183-257` performs writes, prune, and generation update in one transaction; `TestRefresh_AuthoritativeEmptyAndDeletedCodePrune`, `TestRefresh_UnavailableGraphPreservesCodeCorpus`, and `TestReconcileCorpus_DeadlineLockPreservesGeneration` cover deletion, unavailable-source preservation, and rollback.
- [✓] AC-5: quiet deadline/model/lock/extraction failures preserve prior state and remain non-blocking — `internal/cli/embeddings.go:277-319` normalizes quiet command errors without output; `internal/cli/scan.go:107-120` and `internal/cli/scan.go:281-310` apply the same non-blocking quiet contract to the shared incremental command; `TestEmbeddingsRefreshCLIQuietDeadlineIsSilentNonBlockingAndPreservesCorpus`, `TestEmbeddingPhaseUnavailableModelIsTruthful`, `TestReconcileCorpus_DeadlineLockPreservesGeneration`, and `TestIncrementalScanCLIQuietIsSilentAndNonBlocking` exercise the relevant paths.
- [✓] AC-6: nil models return a descriptive error — `internal/embeddings/refresh.go:45-56` rejects nil with `embedding model is unavailable`; `TestRefresh_NilModelReturnsUnavailableError` exercises the guard.
- [✓] AC-7: hash reads are bounded and changed writes plus prune commit once per corpus — `internal/embeddings/storage.go:129-179` batches hash reads in groups of 500; `internal/embeddings/storage.go:181-257` uses one corpus transaction; `TestStoredHashes_BatchesLargeKeepSet` and `TestReconcileCorpus_RollsBackAllWritesOnBatchFailure` verify batching and atomic rollback.
- [✓] AC-9: storage stats include count and newest embedding timestamp per corpus — `internal/embeddings/storage.go:387-441` returns per-corpus counts, newest `embedded_at`, and successful extraction metadata; `TestStats_PerCorpusNewestAndGeneration` asserts populated and empty known-corpus entries.
- [✓] AC-10: unavailable or incomplete extraction preserves existing corpus chunks — `internal/embeddings/chunker.go:35-85` distinguishes unavailable, errored/partial, and authoritative-complete extraction; `internal/embeddings/refresh.go:97-121` avoids reconciliation for unsafe results; `TestExtractCorpus_DistinguishesUnavailableAndAuthoritativeEmpty`, `TestExtractCorpus_CanceledTraversalIsPartial`, and `TestRefresh_UnavailableGraphPreservesCodeCorpus` verify the distinction and preservation.

## Changes

- [✓] `internal/embeddings/refresh.go` — adds context and nil guards, pre-embed hash lookup, configured-scope de-duplication, per-corpus outcomes, and authoritative reconciliation.
- [✓] `internal/embeddings/storage.go` — adds bounded hash reads, context-aware one-transaction corpus reconciliation, compatibility `Upsert`, generation metadata, and extended per-corpus stats.
- [✓] `internal/embeddings/chunker.go` — adds explicit extraction authority/completion/unavailability and context-aware graph queries; the event query property keys remain unchanged.
- [✓] `internal/cli/embeddings.go` — adds the reusable configured-scope phase and `embeddings refresh --if-stale --deadline <duration> -q` with byte-silent quiet normalization.
- [✓] `internal/cli/scan.go` — invokes embeddings after graph/FTS reconciliation, retries coverage for structurally unchanged code, and commits scan state only after the embedding phase succeeds or is intentionally disabled.
- [✓] Focused tests — `internal/embeddings/refresh_test.go`, `internal/embeddings/storage_test.go`, `internal/embeddings/chunker_test.go`, and new `internal/cli/embeddings_test.go` assert hash skips, configured scope, rollback, extraction safety, deletion pruning, deadline/lock behavior, storage stats, and Cobra silence.

## Audit notes

- The supplied validation evidence reports focused and full Go suites passing, an independent uncached focused run passing, `git diff --check` passing, 8/8 drift and EARS checks, and compiled CLI exercises for unchanged refresh, deletion pruning, unavailable graph preservation, and quiet lock timeout.
- The implementation diff is confined to the spec and its named engine, storage, chunker, coordinator, and test files.
