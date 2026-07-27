# Delivery audit — hook-driven-index-freshness

**Audited:** original delivery at `0859771` plus `git diff 0859771 -- internal/cli/scan.go internal/cli/scan_test.go .hero/specs/incremental-code-graph-refresh/spec.md .hero/specs/hook-driven-index-freshness/spec.md .hero/specs/continuous-code-index-freshness/spec.md`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] AC-1: pre-commit runs lexical refresh, aggregate code refresh, NEXT checkpoint, queue write, and staging in order — `internal/cli/next_hooks.go:317`; exact command forms and order are asserted by `TestHookScript_IndexFreshnessPipelineOrder`.
- [✓] AC-2: post-merge runs lexical refresh, aggregate code refresh, and NEXT checkpoint without queue write or staging — `internal/cli/next_hooks.go:338`; exact order and exclusions are asserted by `TestHookScript_IndexFreshnessPipelineOrder`.
- [✓] AC-3: refresh failures remain silent/non-blocking and health-visible — every managed Hero command redirects stdout and stderr before `|| true` at `internal/cli/next_hooks.go:322` and `:343-345`; `TestManagedPreCommitSuppressesFailingHeroOutputAndAllowsCommit` proves the installed hook suppresses a failing stub while Git succeeds, and `TestManagedHooksConvergeCodeIndexesAcrossCommitAndMerge` proves a held real lock preserves Git success and leaves missing coverage visible to `hero check`. The amended `internal/cli/scan.go:565-581` retains the same non-blocking `flock` and busy result while placing the lock under ignored `.hero/cache/`.
- [✓] AC-4: freshness is activated only through the existing repository pre-commit/post-merge installer — `internal/cli/next_hooks.go:89`, `:278`, and the scoped diff show no post-commit, session-hook, target, or harness files. The repository Git-hook surface is shared by all six targets.
- [✓] AC-5: old managed content is detected as stale and the existing refresh path rewrites both hooks — `TestPreCommitHookStale_DetectsContentDrift`, `TestPreCommitHookStale_CurrentBlockIsFresh`, and `TestRefreshHooksIfPresent_RefreshesStale`; `installNextHooksQuiet` writes the current template for pre-commit and post-merge.
- [✓] AC-6: code-index health reports changed, missing, and deleted sources from inventory/checksum state rather than HEAD time — `internal/cli/check.go:736` and `TestInspectCodeIndexFreshnessReportsActualChangedMissingDeletedSources`. The disclosed changed/new-source parsing limitation does not mutate state; unchanged sources use cached parse products with `reparsed=0`.
- [✓] AC-7: embedding health reports per-corpus missing, mismatched, orphaned, and unavailable coverage without vectors or writes — `internal/cli/check.go:797`, `TestInspectEmbeddingFreshnessReportsCoverageWithoutMutatingVectors`, and `TestInspectEmbeddingFreshnessReportsUnavailableGraphCorpus`. `TestInspectEmbeddingFreshnessSkipsCodeWhenCodeScanDisabled` additionally proves disabled/nil code scanning omits code-vector demand while other configured corpora remain checked.
- [✓] AC-8: embeddings status renders count and newest timestamp for every corpus in stable order without claiming freshness — `internal/cli/embeddings.go:116` and `TestEmbeddingsStatusRendersEveryCorpusInStableOrderWithNewestTimestamp`.
- [✓] AC-9: the installed pre-commit hook converges add/change/delete across graph, FTS, vector, and hybrid retrieval without a post-bootstrap scan — `TestManagedHooksConvergeCodeIndexesAcrossCommitAndMerge`, especially `internal/cli/index_freshness_integration_test.go:42-65` and `:145-223`.
- [✓] AC-10: the installed post-merge hook converges add/change/delete across graph, FTS, vector, and hybrid retrieval without a post-bootstrap scan — `mergeLifecycle` commits branches with `--no-verify`, merges through the installed hook, and the same four-surface assertions run at `internal/cli/index_freshness_integration_test.go:67-73`.

## Changes

- [✓] Update `internal/cli/next_hooks.go` managed blocks — both hooks contain the ordered, silent, best-effort pipeline and preserve pre-commit-only queue/staging behavior.
- [✓] Extend hook template/currency tests — exact order and redirection, real failing-command silence, post-merge exclusions, marker preservation, staleness, regeneration, idempotence, and opt-out paths have concrete coverage.
- [✓] Add human/JSON code and embedding freshness rows — categorized rows use actual inventory and hashes, remain read-only, and intentionally skip code-vector demand when code scanning is disabled while checking other configured corpora.
- [✓] Extend embeddings status observability — stable canonical corpus ordering, counts, newest timestamps, and empty-corpus rendering are implemented and asserted.
- [✓] Add Git lifecycle end-to-end coverage — the dedicated test builds Hero, installs real hooks, exercises commit/merge add/change/delete, checks all four retrieval surfaces, and covers a held refresh lock.
- [✓] Update existing CLI help — hook help documents automatic best-effort refresh and the cache-rebuilding `hero scan --code` recovery command; `TestInstallHooksHelpUsesFullScanForManualRecovery` rejects the ineffective incremental recovery form, and embeddings status help distinguishes timestamps from freshness.

## Post-rollout amendment

- [✓] Move the refresh lock off the tracked workspace root — `internal/cli/scan.go:565-570` creates and opens `.hero/cache/code-refresh.lock`; no production behavior beyond the parent and lock path changed from `0859771`.
- [✓] Preserve lock exclusivity and fail-fast behavior — `LOCK_EX|LOCK_NB`, `EWOULDBLOCK`/`EAGAIN` classification, structured busy skip before state load, unlock, and close are unchanged at `internal/cli/scan.go:574-589`. `TestManagedHooksConvergeCodeIndexesAcrossCommitAndMerge` still covers cross-process contention through a compiled hook binary.
- [✓] Create the ignored parent safely — `os.MkdirAll(cacheDir, 0o755)` is idempotent and runs before `OpenFile`; `TestIncrementalCodeRefreshSkipsUnusableCacheAndBusyLock` starts without a cache directory and proves acquisition creates the lock successfully.
- [✓] Use the canonical ignored seam — the current workspace ignores `.hero/cache/` at `.gitignore:23`; fresh and reinitialized workspaces receive the same entry from `internal/cli/init.go:429-466`, covered by `TestEnsureManagedGitignoreBlock_CreatesWhenMissing` and `TestEnsureManagedGitignoreBlock_RefreshesUpdatedEntries`.
- [✓] Fence the original regression — `internal/cli/scan_test.go:563-568` requires the cache-local lock to exist and the former `.hero/code-refresh.lock` path not to exist before exercising the existing contention assertion. Reverting to the root path fails both checks.
- [✓] Require no legacy cleanup path — current `git status --short --ignored` and filesystem inspection find neither `.hero/code-refresh.lock` nor `.hero/knowledge/tracker/config.json`. The faulty lock path exists only in local commit `0859771`, which the branch reports ahead of `origin/main`; there is no released root-lock population requiring migration, stale-lock deletion, or dual-path coordination.

## Validation evidence

- Focused lifecycle and lock tests: `go test ./internal/cli -run 'TestIncrementalCodeRefreshSkipsUnusableCacheAndBusyLock|TestManagedHooksConvergeCodeIndexesAcrossCommitAndMerge' -count=1 -v` — PASS.
- Affected packages: `go test ./internal/cli ./internal/codescan ./internal/embeddings ./internal/index ./internal/retrieval -count=1` — PASS.
- Full repository: `go test -p 1 ./... -count=1` — PASS.
- Spec integrity: `hero spec lint` passed 10/10 hook criteria and 18/18 incremental criteria; `hero drift` reported no drift for both specs.
- Scope and formatting: `git diff --check` passed; the amendment adds no dependency, schema, configuration, harness, or engine change outside `internal/cli/scan.go`.

## Open items (if any)

- None.

## Audit notes

- None.
