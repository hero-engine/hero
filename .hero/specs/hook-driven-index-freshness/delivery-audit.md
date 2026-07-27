# Delivery audit — hook-driven-index-freshness

**Audited:** `git diff 189e288 -- .hero/planning/initiatives/continuous-code-index-freshness/hook-driven-index-freshness/spec.md internal/cli/check.go internal/cli/check_json_test.go internal/cli/check_test.go internal/cli/embeddings.go internal/cli/embeddings_test.go internal/cli/next_hooks.go internal/cli/next_hooks_test.go` plus `git diff --no-index /dev/null internal/cli/index_freshness_integration_test.go`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] AC-1: pre-commit runs lexical refresh, aggregate code refresh, NEXT checkpoint, queue write, and staging in order — `internal/cli/next_hooks.go:317`; exact command forms and order are asserted by `TestHookScript_IndexFreshnessPipelineOrder`.
- [✓] AC-2: post-merge runs lexical refresh, aggregate code refresh, and NEXT checkpoint without queue write or staging — `internal/cli/next_hooks.go:338`; exact order and exclusions are asserted by `TestHookScript_IndexFreshnessPipelineOrder`.
- [✓] AC-3: refresh failures remain silent/non-blocking and health-visible — every managed Hero command redirects stdout and stderr before `|| true` at `internal/cli/next_hooks.go:322` and `:343-345`; `TestManagedPreCommitSuppressesFailingHeroOutputAndAllowsCommit` proves the installed hook suppresses a failing stub while Git succeeds, and `TestManagedHooksConvergeCodeIndexesAcrossCommitAndMerge` proves a held real lock preserves Git success and leaves missing coverage visible to `hero check`.
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

## Open items (if any)

- None.

## Audit notes

- None.
