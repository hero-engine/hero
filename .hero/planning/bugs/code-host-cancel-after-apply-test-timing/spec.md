---
title: "Code-host cancel-after-apply tests race provider dispatch"
slug: code-host-cancel-after-apply-test-timing
type: bug
status: delivering
diagnosis_status: diagnosed
domain: engineering
priority: high
severity: high
size: small
root_cause_class: code
created: 2026-07-30
tags: [code-host, tests, cancellation, release-blocker]
relations:
  - target: code-host-broker-github-adapter
    kind: related
delivery_method: manual
---

# Code-host cancel-after-apply tests race provider dispatch

## Kickoff

Makes code-host cancel-after-apply coverage synchronize on the fake provider's
observed write instead of assuming preflight completes within 100–150 ms.

**Status:** delivering — deterministic synchronization is implemented and all
validation passes; the cold delivery audit is next.

**Pick up at:** audit the isolated test-only implementation, then run Hero's
verification gate.

→ `/deliver code-host-cancel-after-apply-test-timing`

**Files:** `internal/codehost/github_create_test.go`,
`internal/codehost/github_collaboration_test.go`,
`internal/codehost/github_state_test.go`,
`internal/codehost/github_merge_test.go`, and a shared
`internal/codehost/*_test.go` helper if useful.

**Skip:** do not weaken cancellation/reconciliation assertions or change
production broker behavior; the broker correctly returns a pre-dispatch
deadline error when the context expires before a provider write.

## Summary

The `v0.31.0` release workflow failed in GoReleaser's `go test ./...` before
publishing artifacts. The failing case was
`TestCollaborationLostDelayedAmbiguousAndCancelledResponses/cancel_after_approve`.
Its context deadline expired before the fake GitHub provider observed a write,
so Hero truthfully returned `deadline_exceeded` with
`ReconciliationNotApplied` and the fake recorded zero attempts. The test
expected a write followed by cancellation and reconciliation.

The test tries to establish "cancel after apply" with a 150 ms timeout and a
500 ms fake response delay. That only orders cancellation after provider apply
when all broker preflight work completes inside 150 ms. GitHub Actions release
load violated that unstated timing assumption. The same pattern appears in
collaboration, lifecycle-state, and merge tests.

## Root Cause

The tests use elapsed wall-clock time as a proxy for a causal event:

1. start `Broker.Execute`;
2. hope preflight and provider dispatch finish within 100–150 ms;
3. let the context deadline cancel the request;
4. expect the provider attempt to equal one and reconciliation to recover the
   externally applied action.

Preflight includes capability, identity, pull-request, permission, freshness,
and journal work. Its duration is intentionally not bounded by the test's
100–150 ms assumption. Under CI load, the deadline may fire before dispatch.
That is a valid pre-dispatch cancellation result, not the post-apply scenario
the test claims to exercise.

The fake provider already exposes thread-safe attempt counters. Tests can
therefore establish the required happens-before relationship explicitly:
execute asynchronously, wait with a bounded test deadline until the attempt
counter reaches one, cancel the context, and assert reconciled-applied output.
A long fake response delay keeps the provider response pending until that
explicit cancellation.

## Evidence

1. Production release run `30568093164` failed only
   `cancel_after_approve`.
2. The failure returned `deadline_exceeded`,
   `ReconciliationNotApplied`, and `attempts=0`, proving cancellation preceded
   provider dispatch.
3. Required Test and Smoke workflows for the same commit passed, as did two
   local full suites and two GoReleaser snapshots.
4. The focused case passed 100 local repetitions in 29.7 seconds, confirming
   that ordinary local timing hides the race.
5. Create, collaboration, state-transition, and merge tests all use the same
   short-deadline/long-response-delay construction.

## Goal

Make post-apply cancellation coverage causally deterministic across local,
loaded CI, race-enabled, and release test environments without changing
production behavior.

## Acceptance Criteria

- **AC-1:** WHEN a cancel-after-apply test cancels its request THE SYSTEM SHALL first observe exactly one fake-provider mutation attempt and SHALL NOT use elapsed time alone to infer provider dispatch.
- **AC-2:** WHEN the explicitly synchronized request is cancelled while the fake response is pending THE SYSTEM SHALL return no contract error and SHALL report `reconciled_applied` with exactly one provider attempt.
- **AC-3:** THE SYSTEM SHALL apply the deterministic synchronization pattern to create, collaboration, lifecycle-state, and merge cancel-after-apply coverage while preserving pre-dispatch cancellation tests.
- **AC-4:** WHEN release qualification runs THE SYSTEM SHALL pass high-count focused cancellation tests, `go test ./internal/codehost -count=1`, `go test ./... -count=1`, and the GoReleaser snapshot prerequisite without timing-dependent retries.

## Suggested Fix Approach

1. Add a test-only bounded wait helper that polls a thread-safe attempt
   accessor until it reaches the expected value, failing with a useful message
   after a generous fixed test deadline.
2. Start each post-apply `Broker.Execute` in a buffered result goroutine under
   `context.WithCancel`.
3. Configure the fake response delay long enough that the explicit
   attempt-observation and cancel establish ordering independent of CI load.
4. Wait for exactly one attempt, cancel, collect the response, and retain the
   existing reconciliation and attempt-count assertions.
5. Keep the existing already-cancelled-context tests unchanged to preserve
   pre-dispatch behavior.

## Changes

1. Add shared test-only synchronization helpers for provider-attempt
   observation and bounded response collection.
2. Convert create and collaboration cancel-after-apply subtests to explicit
   post-dispatch cancellation.
3. Convert lifecycle-state and merge cancel-after-apply tests to the same
   pattern.
4. Run high-count focused, package, repository, race-focused, and snapshot
   release validation.

## Boundaries

- Do not change production cancellation, idempotency, reconciliation, or
  provider transport behavior.
- Do not replace meaningful causal assertions with longer arbitrary sleeps.
- Do not remove pre-dispatch cancellation coverage.
- Do not publish, delete, or reuse `v0.31.0`; recovery is a forward patch
  release after this fix passes.
- Do not change harness installation surfaces.

## Validation

- `go test ./internal/codehost -run 'Cancel|cancel_after' -count=100`
- `go test -race ./internal/codehost -run 'Cancel|cancel_after' -count=10`
- `go test ./internal/codehost -count=1`
- `go test ./... -count=1`
- `go vet ./...`
- `goreleaser release --snapshot --clean`
- `hero spec lint code-host-cancel-after-apply-test-timing`

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Cancel only after one observed provider attempt | DONE | `executeThenCancelAfterAttempt` starts execution asynchronously, observes the thread-safe attempt accessor reach exactly one, then cancels; it fails boundedly if dispatch or reconciliation never occurs |
| 2 | Synchronized cancellation reconciles applied work | DONE | Existing create, collaboration, lifecycle, and merge assertions all retain error-free `reconciled_applied` and exactly-one-attempt requirements |
| 3 | Cover every mutation family and preserve pre-dispatch tests | DONE | All four cancel-after-apply constructions use the shared helper; already-cancelled-context tests remain unchanged |
| 4 | Release qualification is stable | DONE | Exact release-failing subtest passed 100 repetitions, all cancellation tests passed 25 repetitions and race-enabled 10 repetitions, package/full repository tests and vet passed, and GoReleaser snapshot built all six targets |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Add bounded attempt/response synchronization helper | DONE | Added `internal/codehost/cancellation_test.go` with a test-only causal helper and generous failure bounds |
| 2 | Convert create and collaboration tests | DONE | Both now use a 30-second pending fake response and explicit cancellation after the fake records one attempt |
| 3 | Convert lifecycle and merge tests | DONE | Both use the same shared helper; no production file changed |
| 4 | Run focused, race, repository, and snapshot validation | DONE | 100x exact failure, 25x all cancellation, 10x race-enabled cancellation, package/full suites, vet, and GoReleaser snapshot pass |

### Exercise-the-feature check

- [x] The exact CI-failing `cancel_after_approve` case was exercised 100 times,
  while all four mutation families were repeatedly cancelled only after the
  fake provider confirmed the external write.

### Excellence Bar self-check

- [x] Yes — the tests now encode the causal state they claim to cover, retain
  strong cancellation/reconciliation assertions, and avoid production changes
  or arbitrary deadline inflation.
