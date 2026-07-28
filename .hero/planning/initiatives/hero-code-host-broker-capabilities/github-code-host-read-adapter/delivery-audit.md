# Delivery audit — github-code-host-read-adapter

**Audited:** `git diff f9bd1a4...HEAD` (`a084b29eee31fa8fd8e09357b891729ba359acfe`)
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria
- [✓] Advertise exactly the implemented read operations with authoritative policies and bounds — `internal/codehost/broker.go:24`, `internal/codehost/github.go:354`; `TestCapabilitiesAdvertiseExactlyImplementedReads`.
- [✓] List and search configured repositories with qualified identities and opaque bounded cursors — `internal/codehost/github.go:405`, `internal/codehost/github.go:453`, `internal/codehost/github.go:492`; `TestListAndSearchUseConfiguredRepositoryScopeAndOpaqueCursor`.
- [✓] Reject cursor material mismatches before provider I/O — `internal/codehost/broker.go:72`, `internal/codehost/broker.go:83`; the contract validator binds connection, repository scope, operation, query, and order, and `TestListAndSearchUseConfiguredRepositoryScopeAndOpaqueCursor` asserts the provider request count is unchanged.
- [✓] Preserve fork-qualified base/head repository identity, refs, and SHAs — `internal/codehost/normalize.go:163`; `TestForkAndForcePushPreserveIdentityAndFreshness`.
- [✓] Invalidate head-dependent observations after a force push — `internal/codehost/github.go:937`; `TestForkAndForcePushPreserveIdentityAndFreshness` verifies new revisions and discarded diff, check, review, and readiness state.
- [✓] Return explicit bounded commit and diff prefixes — `internal/codehost/github.go:619`, `internal/codehost/github.go:664`, `internal/codehost/github.go:754`; `TestCommitAndDiffBoundsAreExplicit` verifies truncation/completeness, output bounds, and commit continuation.
- [✓] Preserve useful sections with bounded typed partial failures — `internal/codehost/broker.go:158`, `internal/codehost/github.go:510`, `internal/codehost/github.go:1026`; `TestPartialGraphQLAndMergeabilityNeverInventReadiness` verifies retained readiness data and a typed failed checks section.
- [✓] Avoid invented merge-readiness certainty — `internal/codehost/github.go:1040`; `TestPartialGraphQLAndMergeabilityNeverInventReadiness` verifies pending and unavailable evidence remains unknown or unavailable.
- [✓] Normalize REST and GraphQL rate-limit state — `internal/codehost/github.go:199`, `internal/codehost/github.go:218`; `TestRateLimitsPermissionsAndNormalizedErrors` and `TestPartialGraphQLAndMergeabilityNeverInventReadiness`.
- [✓] Cancel before dispatch and during pagination — `internal/codehost/broker.go:91`, `internal/codehost/github.go:430`; `TestCancellationBeforeAndDuringPagination` verifies zero pre-dispatch calls and a typed partial prefix with no extra page.
- [✓] Reject out-of-scope repositories before credential resolution or provider dispatch — `internal/codehost/broker.go:117`; `TestRepositoryScopeAndCrossOriginPaginationFailClosed` verifies a forbidden response and zero provider calls.
- [✓] Fail closed on cross-origin Enterprise pagination and redirects — `internal/codehost/github.go:99`, `internal/codehost/github.go:271`; `TestRepositoryScopeAndCrossOriginPaginationFailClosed`, `TestCrossOriginRedirectDoesNotForwardAuthorization`, and `TestEnterpriseOriginAndCredentialRedaction`.
- [✓] Provide an independent deterministic GitHub fake for the required scenarios — `internal/mockcodehost/server.go:22`; `TestScenarioBuildersCoverDeclaredBehaviors`, `TestPaginationForkAndForcePushAreDeterministic`, `TestRateLimitPermissionsAndGraphQLPartialFailures`, and `TestOversizedDiffAndChangingMergeability`.
- [✓] Keep credentials and raw authorization out of outputs and fixture inventory — `internal/codehost/github.go:128`, `internal/mockcodehost/server.go:81`; `TestCrossOriginRedirectDoesNotForwardAuthorization`, `TestEnterpriseOriginAndCredentialRedaction`, `TestRequestInventoryNeverStoresAuthorization`, and `TestConfigurationFailureDoesNotLeakUnderlyingError`.

## Changes
- [✓] Add the in-process broker, dispatch, validation, bounds, and cancellation handling — `internal/codehost/broker.go`.
- [✓] Add a typed GitHub transport with configured-origin and authorization containment — `internal/codehost/github.go:51`.
- [✓] Implement and normalize all ten read/discovery operations — dispatch is complete at `internal/codehost/github.go:320`; provider DTO normalization is isolated in `internal/codehost/normalize.go`.
- [✓] Add opaque cursors and operation-specific observation revisions — `internal/codehost/github.go:363`, `internal/codehost/broker.go:273`.
- [✓] Normalize readiness, rate limits, permissions, and transient state conservatively — `internal/codehost/github.go:199`, `internal/codehost/github.go:1040`.
- [✓] Add an independent deterministic `internal/mockcodehost` — `internal/mockcodehost/server.go`, `internal/mockcodehost/server_test.go`; no `internal/mocktracker` change appears in the diff.
- [✓] Add adapter, conformance, cancellation, bounds, origin, redaction, and fake-provider tests — `internal/codehost/broker_test.go`, `internal/mockcodehost/server_test.go`.

## Audit notes
- No Completion Ledger rows were missing, partial, skipped, blocked, or downgraded.
- The implementation diff is scoped to the adapter, fake, tests, spec, and generated Hero handoff projections.
- Provided validation evidence: focused code-host/fake and tracker tests passed; the full repository test suite passed; `go vet`, focused race detection, `git diff --check`, 14/14 drift checks, and 14 EARS lint checks passed.
