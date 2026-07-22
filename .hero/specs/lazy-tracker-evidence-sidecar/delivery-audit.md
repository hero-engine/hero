# Delivery audit — lazy-tracker-evidence-sidecar

**Audited:** `git diff --cached`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] AC-1: first explicit Jira load publishes the sidecar — `internal/tracker/evidence_store.go`; `TestEvidenceLoader_FirstLoadCurrentRefreshAndForce`; `TestSyncEvidence_JiraSidecarEndToEnd`; existing paginated Jira evidence test.
- [✓] AC-2: the five-part cache check returns `current` without full work or rewriting files — `validateEvidenceSnapshot`; `TestEvidenceLoader_FirstLoadCurrentRefreshAndForce`; isolated CLI exercise call counters.
- [✓] AC-3: stale identity, timestamp, payload, and attachment state refetch atomically — `TestEvidenceLoader_CorruptionMissingAttachmentAndUnknownTimestampRefetch`, `TestEvidenceLoader_ManifestIdentityMismatchRefetches`, and publication failure tests.
- [✓] AC-4: cache hits preserve bytes and mtimes — loader unit assertions plus the isolated Jira-like second-load exercise.
- [✓] AC-5: evidence remains private and manifest/status fields are allowlisted — contract shape tests, credential canaries, private-mode assertions, and managed ignore tests.
- [✓] AC-6: restrictive writes, cancellation, and rollback are exercised — `ContextEvidenceTracker`; Jira context-aware HTTP methods; CLI/MCP process contexts; `TestEvidenceLoader_CancelsEveryInFlightProviderStage`; `TestJiraEvidenceContextCancelsInFlightRequests`; `TestPublishEvidenceSnapshotFailuresPreservePriorCommit`.
- [✓] AC-7: attachment paths/bytes participate in the hash and omissions remain private — mutated/missing attachment tests and `TestEvidenceLoader_DownloadOmissionRemainsPrivateAndNeverCurrent`.
- [✓] AC-8: missing or malformed timestamps never become current and remain exact — timestamp branch in `EvidenceLoader.Load`; malformed-source repeated-refresh assertions.
- [✓] AC-9: provider unavailability preserves validated stale state without claiming a hit — `TestEvidenceLoader_UnavailablePreservesValidatedSnapshot`.
- [✓] AC-10: unsupported providers create nothing — loader and MCP unsupported-provider tests.
- [✓] AC-11: no implicit path loads evidence — `NewEvidenceLoader` remains limited to explicit CLI/MCP handlers; import, refresh, discovery, queue, serve startup, and broker paths do not call it.
- [✓] AC-12: in-process, CLI, and MCP use the same `tracker-evidence/v1` request/status — shared contract package and adapter tests, including MCP cancellation context.
- [✓] AC-13: default CLI output remains full `IssueEvidence` — unit compatibility test and isolated authenticated Jira-like legacy-output exercise.
- [✓] AC-14: Hero Code fixtures cover every state/error and tolerate unknown fields — contract fixture tests and `hero tracker contract tracker-evidence` binary surface.
- [✓] AC-15: broker, Jira authentication/same-origin handling, comments, normalized/raw evidence, and credential safety remain intact — isolated authenticated exercise, context-aware Jira tests, and supplied full repository validation.

## Changes

- [✓] Shared contract and fixtures — `contracts/trackerevidence/**`, documentation, and released-binary fixture selection.
- [✓] Shared loader/store — freshness validation, private hashing/storage, context-aware provider calls, atomic publication/recovery, and trusted snapshot reading.
- [✓] CLI compatibility and status adapters — shared loader, `--status`, `--force`, `--no-attachments`, connection selection, and signal-derived context.
- [✓] MCP adapter — tool definition/dispatch, status-only output, and process cancellation context.
- [✓] Managed ignore rule — recursive private-sidecar ignore with init/refresh/idempotency coverage while the manifest remains trackable.
- [✓] End-to-end and failure coverage — real Jira-like CLI flow, attachment mutation/omission, in-flight HTTP cancellation, publication-stage failures, recovery, concurrency, corruption, and surface tests.

## Open items

- None.

## Audit notes

- The previous HOLD findings are closed: provider HTTP calls now propagate context; MCP and CLI supply process contexts; publication failures are injected at backup/candidate/manifest stages; attachment mutation and omissions are asserted; and the isolated Jira-like CLI scenario covers fetched/current/refreshed plus legacy output.
- Focused contract/tracker/CLI/MCP tests and focused race tests passed independently during this re-audit. The supplied full test, vet, docs, lint, fixture-command, and gitignore evidence is also clean.
