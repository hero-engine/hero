# Delivery audit — github-pull-request-state-transition-broker

**Audited:** `git diff 07395c5...8dc0993`
**Verdict:** HOLD
**Surface:** noteworthy

## Acceptance criteria

- [✓] Expose four explicit external-write transitions — `internal/codehost/broker.go:37` advertises all four operations; `TestStateTransitionOperationsAdvertisedAppliedAndReplayed` asserts `external_write` and `explicit_user`.
- [✓] Require qualified PR, head, intent, idempotency, and revisions — `internal/codehost/github_state.go:63` strictly decodes operation payloads; the v1 request validator and `TestStateTransitionRequiredFieldsAndScope` cover the required mutation fields.
- [✓] Retarget requires observed current and explicit existing new base — `internal/codehost/github_state.go:98` binds both bases to the request repository; `internal/codehost/github_state.go:183` exact-matches the current base and resolves the requested target ref.
- [✓] Reject incompatible permission, revision, head, lifecycle, and base changes before dispatch — `internal/codehost/github_state.go:154` implements the preflight gates; `TestStateTransitionStalePermissionAndTargetGates` asserts zero writes for force pushes, permission changes, stale current/target bases, missing branches, and stale revisions.
- [✗] Exact desired state returns external completion — the retarget comparator at `internal/codehost/github_state.go:368` compares repository and branch name but omits `desiredBase.SHA`. A same-named target branch at a different SHA can therefore be accepted as exact and recorded `externally_completed`.
- [✗] Lost responses reconcile from the exact authoritative desired state — dispatch and reconciliation both call the incomplete comparator (`internal/codehost/github_state.go:276` and `internal/codehost/github_state.go:301`). Because GitHub's retarget write carries only the branch name, a target branch can move after preflight and the broker can still report `applied` or `reconciled_applied` for the wrong requested SHA.
- [✓] Preserve merged as an immutable terminal state — `internal/codehost/github_state.go:162`, `internal/codehost/github_state.go:276`, and `internal/codehost/github_state.go:298` reject merged state before dispatch and during read-back; the normalized lifecycle matrix and merged fake scenarios exercise all four operations.
- [✓] Same-key retries perform at most one provider transition — `internal/codehost/reconcile.go` uses the existing locked journal; `TestStateTransitionOperationsAdvertisedAppliedAndReplayed` and `TestStateTransitionDuplicateConcurrentConflictAndProviderRejection` assert one provider attempt for sequential and concurrent retries.
- [✓] Changed operation, PR, head, base, or desired target conflicts — `internal/codehost/github_state.go:412` binds those dimensions into the payload digest; conflict cases in `TestStateTransitionDuplicateConcurrentConflictAndProviderRejection` assert `idempotency_conflict` without a second write.
- [✗] Return a new observation revision and mark prior readiness observations stale — `internal/codehost/github_create.go:307` returns a new transition observation hash, but no response field, shared revision domain, or cache marker identifies prior readiness evidence as stale. The assertion at `internal/codehost/github_state_test.go:31` only compares hashes from different operations; `internal/codehost/broker.go:368` includes operation in ordinary read revision material, so inequality does not prove invalidation.
- [✓] Respect pre- and post-dispatch cancellation boundaries — the shared state machine checks cancellation before dispatch and uses detached bounded reconciliation after an ambiguous write; `TestStateTransitionLostAmbiguousDelayedAndCancellation` covers both boundaries for all four operations.
- [✓] Exercise required transition hazards through the fake — `internal/mockcodehost/server.go:208` adds deterministic state scenarios, and `internal/codehost/github_state_test.go` covers external completion, stale head/base, missing/moving target, permission races, delayed visibility, force push, merged state, duplicates, cancellation, denial, and lost response. The missing post-preflight target-SHA race assertion is captured under AC-6 rather than credited as exact reconciliation.

## Changes

- [✗] Add validation and desired-state comparators — validation and lifecycle modeling landed, but the retarget desired-state comparator omits the requested target SHA (`internal/codehost/github_state.go:368`).
- [✓] Add lifecycle and target preflight — `internal/codehost/github_state.go:154` checks identity, lifecycle, head, current base, target ref/SHA, permissions, actor, capability revision, and observation revision.
- [✓] Implement typed GitHub transition routes — `internal/codehost/github_state.go:238` uses GraphQL for mark-ready and bounded REST patches for retarget, close, and reopen, followed by authoritative PR reads.
- [✗] Extend journal and desired-state reconciliation — the journal extension is content-safe and the shared state machine is reused, but retarget reconciliation is not exact because it ignores desired-base SHA.
- [✓] Distinguish normalized lifecycle states — `internal/codehost/github_state.go:351` separates merged, closed-unmerged, open-draft, open-ready, and unknown.
- [✓] Extend the deterministic GitHub fake — `internal/mockcodehost/server.go:208` and `internal/mockcodehost/server.go:758` add mutable state, transition routes, delays, races, merged state, permission sequences, and attempt accounting.
- [✗] Add effect, stale-state, receipt, and conformance tests — broad transition coverage landed, but `TestStateTransitionDesiredStateMatrix` has no same-repository/name with mismatched-SHA case, and the readiness assertion does not demonstrate invalidation.

## Audit notes

- AC-5, AC-6, and Change 1 are performative as written: the implementation calls the retarget comparison “exact” while deliberately omitting the SHA carried by `RetargetPayload.NewBase`.
- AC-10 is performative as written: returning a different hash is not evidence that an earlier readiness observation has been marked stale. The current test passes because read revisions are operation-specific even without any invalidation behavior.
- The ledger reports focused lifecycle tests, focused package tests, race detection, `go test ./... -count=1`, `go vet ./...`, formatting, and diff hygiene as passing. Per the cold-audit contract, these commands were not rerun.
