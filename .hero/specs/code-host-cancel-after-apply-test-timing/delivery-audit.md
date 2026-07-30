# Delivery audit — code-host-cancel-after-apply-test-timing

**Audited:** `git diff e8dce05ab2518aeebdc85b3c135587a0ea33e1d4..2e9470e`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] Cancel only after one observed provider attempt — `executeThenCancelAfterAttempt` waits on the fake provider's mutex-protected attempt accessor, cancels only when it equals one, and fails after five seconds rather than inferring dispatch from elapsed response time (`internal/codehost/cancellation_test.go:13`).
- [✓] Synchronized cancellation reconciles applied work — create, collaboration, lifecycle-state, and merge tests retain assertions for no contract error, `reconciled_applied`, and exactly one provider attempt (`internal/codehost/github_create_test.go:194`, `internal/codehost/github_collaboration_test.go:242`, `internal/codehost/github_state_test.go:327`, `internal/codehost/github_merge_test.go:311`).
- [✓] Cover every mutation family and preserve pre-dispatch tests — all four post-apply families call the shared helper, while their already-cancelled-context cases still assert cancellation before dispatch and zero attempts (`internal/codehost/github_create_test.go:322`, `internal/codehost/github_collaboration_test.go:230`, `internal/codehost/github_state_test.go:352`, `internal/codehost/github_merge_test.go:284`).
- [✓] Release qualification is stable — supplied evidence records 100 passing repetitions of the release-failing approve case, 25 passing repetitions of all cancellation tests, 10 race-enabled repetitions, passing package and repository suites, passing vet, 4/4 spec lint, and a GoReleaser snapshot for all six targets.

## Changes

- [✓] Add bounded attempt/response synchronization helper — `internal/codehost/cancellation_test.go` adds a test-only helper with a buffered result channel and separate five-second bounds for observing dispatch and collecting reconciliation.
- [✓] Convert create and collaboration tests — both replace short context deadlines with explicit attempt observation and cancellation while a cancellable 30-second fake response delay is pending.
- [✓] Convert lifecycle and merge tests — both use the same helper and retain their existing reconciliation and attempt-count assertions.
- [✓] Run focused, race, repository, and snapshot validation — the supplied test evidence covers the focused failure, all cancellation cases, the race detector, package and full suites, vet, lint, and all release targets without reported retries.

## Open items

- None.

## Audit notes

- None.
