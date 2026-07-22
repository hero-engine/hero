# Delivery audit — broker-issue-id-validator-raw-string-regression

**Audited:** `git diff -- internal/tracker/broker.go internal/tracker/broker_test.go internal/cli/tracker_broker_test.go` at `b4a8cc5`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] Valid IDs containing `r`, `n`, `x`, or `0` pass validation and dispatch — `internal/tracker/broker.go:201-210` uses actual control runes; `TestValidateBrokerIssueIDAcceptsOrdinaryProviderIDsAndRejectsPathControls` covers all formerly accidental runes, and the direct broker and CLI `ACME-103` tests prove provider dispatch.
- [✓] `ACME-103` evidence returns structured evidence — `TestBrokerGetIssueEvidenceAcceptsACME103` and `TestTrackerBrokerCommandGetsACME103EvidenceThroughConfiguredConnection` send `detail: evidence`, reach the Jira issue/comment handlers, and assert the structured result.
- [✓] Path, traversal, CR, LF, and NUL fail before any provider request — the validator table covers every required unsafe class; `TestBrokerRejectsIssueIDPathConfusionBeforeProviderRequest` sends actual CR/LF/NUL through `Broker.GetIssue` and asserts zero provider calls; the CLI regression sends actual CR/LF/NUL through JSON decoding and asserts its provider-call counter is unchanged.
- [✓] GitHub/GitLab numeric IDs containing `0` remain valid while non-numeric IDs remain rejected — `TestValidateBrokerIssueIDAcceptsOrdinaryProviderIDsAndRejectsPathControls` accepts `10` for both providers and rejects `not-10`.
- [✓] In-process, CLI, and MCP use the corrected shared guard — direct broker and CLI boundary tests exercise `Broker.GetIssue`; `internal/serve/mcp_tools_tracker.go:12-17` continues to delegate MCP `get_issue` requests to the same method.
- [✓] Preserve v1 schema, effects, redaction, bounds, and connection selection — the product diff changes only the validator literal; no contract or adapter code changed. The CLI test asserts the v1 response and credential-canary absence, and the recorded source-qualification exercise reports `effect: read` with no credential exposure.
- [✓] Focused tests cover valid accidental runes and actual unsafe controls — `go test ./internal/tracker -run 'Test(BrokerGetIssue|ValidateBrokerIssueID)' -count=1` and `go test ./internal/cli -run 'TestTrackerBrokerCommand' -count=1` both pass, including exact `ACME-103` JSON and CR/LF/NUL boundary cases.

## Changes

- [✓] Correct shared issue-ID character check — `internal/tracker/broker.go:201-203` replaces the raw literal with an interpreted literal containing actual CR, LF, and NUL runes while retaining all existing path, traversal, and numeric checks.
- [✓] Add table-driven validator regression — `internal/tracker/broker_test.go:113-160` covers ordinary Jira/Linear IDs, numeric GitHub/GitLab IDs, every required unsafe rune, traversal, and non-numeric provider IDs; `internal/tracker/broker_test.go:284-310` proves unsafe boundary requests do not reach the provider.
- [✓] Add successful `ACME-103` evidence regression — `internal/tracker/broker_test.go:72-111` asserts issue/comment endpoints and structured evidence.
- [✓] Add command-level exact JSON regression — `internal/cli/tracker_broker_test.go:55-153` creates an isolated configured connection, sends the exact evidence request, verifies structured success and credential redaction, then sends CR/LF/NUL JSON inputs and proves provider calls do not increase.

## Open items

- None.

## Audit notes

- None.
