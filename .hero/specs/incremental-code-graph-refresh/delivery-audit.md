# Delivery audit — incremental-code-graph-refresh

**Audited:** working-tree diff for the spec-scoped files, including `git diff --no-index /dev/null internal/graph/transaction.go`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] Unchanged incremental scan parses zero files and performs no structural writes — `internal/cli/scan.go:453`; `TestIncrementalCodeRefreshNoChangeAndAddDeleteConvergence` asserts zero reparses, the post-structure result, and unchanged generated/graph/index/cache artifact mtimes; the compiled CLI exercise additionally compared artifact hashes.
- [✓] Existing source configuration, parser resolution, scanner, and excludes are reused — `internal/cli/scan.go:300`, `internal/cli/scan.go:441`; `TestIncrementalRefreshReusesConfiguredExcludes`.
- [✓] Added and changed sources converge generated, graph, and lexical state — `internal/cli/scan.go:470`; `TestIncrementalCodeRefreshNoChangeAndAddDeleteConvergence` asserts generated symbol content plus current graph, `node_index`, and `fts_nodes` counts.
- [✓] Deleted packages, files, symbols, incident edges, and lexical rows retire — `internal/codescan/graph_ingest.go:235`; `TestWriteGraphRetiresDeletedCodeNodesAndIncidentEdges` and `TestIncrementalCodeRefreshNoChangeAndAddDeleteConvergence`.
- [✓] Reconciliation isolates repo partitions and non-codescan sources — `internal/codescan/graph_ingest.go:235`; `TestWriteGraphRetirementIsolatesRepoAndSourceKind`.
- [✓] Scan state advances only after graph, projection, and knowledge phases — `internal/cli/scan.go:470`, `internal/cli/scan.go:500`, `internal/cli/scan.go:514`, `internal/cli/scan.go:546`; `TestGenerateKnowledgeContextCommitsStateSeparately`.
- [✓] Failures and incomplete snapshots do not advance state or authorize retirement — `internal/cli/scan.go:450`, `internal/codescan/graph_ingest.go:45`; `TestIncrementalCodeRefreshDeadlineDoesNotAdvanceState`, `TestScanContextMarksParseFailureIncomplete`, and `TestWriteGraphRejectsIncompleteResultWithoutRetirement`.
- [✓] Quiet incremental mode is silent, deadline-bound, and non-blocking — `internal/cli/scan.go:115`, `internal/cli/scan.go:290`, `internal/cli/scan.go:589`; `TestIncrementalCodeRefreshDeadlineDoesNotAdvanceState`, `TestIncrementalScanCLIQuietIsSilentAndNonBlocking`, and the byte-silent compiled CLI exercise.
- [✓] Manual scan uses the factored coordinator while preserving its output and artifacts — `internal/cli/scan.go:278`; existing manual scan tests and the compiled manual bootstrap exercise passed.
- [✓] Incremental mode remains structural-only — `internal/cli/scan.go:488`, `internal/cli/scan.go:522`; work/sibling and embedding phases are gated to manual mode, and the compiled incremental exercise observed only structural convergence.
- [✓] Walk, read, and parse skips make the result incomplete and non-authoritative — `internal/codescan/scanner.go:168`, `internal/codescan/scanner.go:214`, `internal/codescan/scanner.go:256`; `TestScanContextMarksParseFailureIncomplete` verifies diagnostics and generation/state rejection, while `TestWriteGraphRejectsIncompleteResultWithoutRetirement` verifies graph/history preservation.
- [✓] Graph upsert and retirement are atomic — `internal/graph/transaction.go:19`, `internal/codescan/graph_ingest.go:49`; `TestWriteGraphRollsBackUpsertsWhenReconciliationFails`.
- [✓] Split checksum/cache state is rejected — `internal/codescan/scancache.go:117`; `TestScanStateRejectsSplitGenerationAndParserMismatch` verifies the checksum manifest mismatch is unusable.
- [✓] A busy refresh lock skips before mutation — `internal/cli/scan.go:403`, `internal/cli/scan.go:563`; `TestIncrementalCodeRefreshSkipsUnusableCacheAndBusyLock`.
- [✓] Endpoint-only sources participate in change accounting — `internal/codescan/scanner.go:192`, `internal/codescan/scanner.go:226`; `TestScanContextAccountsForChangesAndEndpointOnlySources` verifies proto deletion and GraphQL addition.
- [✓] Parser identity mismatch rejects cached products — `internal/codescan/scancache.go:126`; `TestScanStateRejectsSplitGenerationAndParserMismatch`.
- [✓] Import-only package changes rebuild package identity and imports edges — `internal/codescan/graph_ingest.go:332`; `TestWriteGraphImportOnlyChangeRebuildsImportsEdge`.
- [✓] Structurally unchanged runs preserve the child-2 post-structure seam — `internal/cli/scan.go:453`; `TestIncrementalCodeRefreshNoChangeAndAddDeleteConvergence` asserts `PostStructureReady` without artifact rewrites.

## Changes

- [✓] Factor the CLI coordinator and add incremental/deadline/quiet behavior — `internal/cli/scan.go`, `internal/cli/scan_test.go`, and `internal/cli/helpers_test.go` contain the shared coordinator, flags, fail-fast lock, structured result, reset coverage, and end-to-end assertions.
- [✓] Add authoritative scanner accounting and paired cache state — `internal/codescan/scanner.go`, `internal/codescan/types.go`, `internal/codescan/scancache.go`, and `internal/codescan/codescan_test.go` contain context-aware inventory, completeness diagnostics, endpoint accounting, parser identity, checksum manifests, and rejection tests.
- [✓] Add transactional graph reconciliation — `internal/codescan/graph_ingest.go`, `internal/codescan/graph_ingest_test.go`, `internal/graph/node.go`, `internal/graph/edge.go`, and new `internal/graph/transaction.go` reuse canonical upserts inside one transaction and scope retirement by repo/type/source.
- [✓] Propagate cancellation through lexical projection — `internal/index/project.go` adds context-aware graph reads and transactional FTS operations; CLI convergence tests verify deletion from both lexical tables.
- [✓] Separate generation from final state commit — `internal/codescan/generate.go` writes/prunes generated knowledge without advancing state; `internal/codescan/scancache.go` owns the final context-aware paired commit.

## Audit notes

None.
