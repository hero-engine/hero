# Delivery audit — github-pull-request-collaboration-broker

**Audited:** `git diff dfe781d...9653ce4`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] AC-1 — `contracts/codehostbroker/validate.go:99`, `internal/codehost/collaboration.go:37`, and `internal/codehost/github_collaboration.go:34` require repository-qualified PR identity, expected head SHA, explicit user intent, stable keys, and capability/observation revisions; `TestCollaborationPreflightPermissionsStateAndFreshness` asserts the required material fails before dispatch.
- [✓] AC-2 — `internal/codehost/broker.go:37` advertises `comment`, `submit_review`, `approve`, and `request_changes` separately, while `TestCollaborationOperationsAdvertisedAndApplied` asserts each operation's `external_write` and `explicit_user` policy.
- [✓] AC-3 — `internal/codehost/github_collaboration.go:109` checks provider PR identity, open state, exact head, actor, and operation-specific permission before the shared state machine reaches dispatch; `TestCollaborationPreflightPermissionsStateAndFreshness` covers unsupported, closed, stale, capability-changed, observation-changed, and forbidden paths with zero writes.
- [✓] AC-4 — `internal/codehost/collaboration.go:70` reserves marker bytes and rejects caller-supplied valid markers, while `collaborationMarker` derives one fixed-size lowercase-hex marker only from the stable operation identity; `TestCollaborationOperationsAdvertisedAndApplied` asserts exactly one marker for each operation.
- [✓] AC-5 — `internal/codehost/idempotency.go:78` persists provider receipt, qualified PR identity, typed actor, and head SHA without body text; `internal/codehost/github_create.go:307` returns the actor through additive `MutationResult.actor`, defined at `contracts/codehostbroker/contract.go:310` and strictly validated at `contracts/codehostbroker/validate.go:517`. `TestCollaborationOperationsAdvertisedAndApplied` asserts actor login and provider ID for all four operations. The canonical consumer fixture contains typed actors in all four collaboration results, the independent consumer shape decodes them, and the published sidecar matches SHA-256 `49156d4a15aacb64f09c5a64a107e8fafc323db4441aa6fb15a8f87b39c00903`, providing the versioned Hero Code handoff artifact.
- [✓] AC-6 — `internal/codehost/github_collaboration.go:301` reconciles by exact marker and actor, additionally binding review state and expected head; `TestCollaborationLostDelayedAmbiguousAndCancelledResponses` proves each operation recovers a lost response without a second write.
- [✓] AC-7 — `internal/codehost/reconcile.go:148` records `ambiguous` when read-back cannot prove the exact effect, and same-key replay remains in reconciliation; delayed, unavailable, marker-collision, mismatched-state, and partial-read-back tests assert one provider attempt.
- [✓] AC-8 — `internal/codehost/github_collaboration.go:134` recognizes only the authenticated actor's authoritative current-head `APPROVED` or `CHANGES_REQUESTED` state; `TestCollaborationExternallyCompletedReviewRules` asserts `externally_completed` and zero writes.
- [✓] AC-9 — `internal/codehost/github_collaboration.go:386` binds the authoritative review to actor and head and treats a latest dismissed review as non-authoritative; old-head, dismissed, and other-actor scenarios each produce one requested effect rather than external completion.
- [✓] AC-10 — `internal/codehost/reconcile.go:34` runs every mutation under the durable journal lock, and `TestCollaborationDuplicateConcurrentAndConflictSemantics` covers sequential and concurrent retries for all four operations with one provider effect and a stable receipt.
- [✓] AC-11 — `internal/codehost/collaboration.go:112` binds operation, qualified PR target, expected head, body digest, intent, consent, and reconciliation key beneath the connection-scoped idempotency key; conflict tests vary operation/review state, target, head, and body and assert no additional write.
- [✓] AC-12 — `internal/codehost/collaboration.go:90` strips only the exact bounded marker grammar, and `internal/codehost/normalize.go:242` applies it to normalized reviews/comments; direct malformed-lookalike assertions and `FuzzHeroMarkerStripping` preserve all other content.
- [✓] AC-13 — `internal/codehost/reconcile.go:89` stops cancelled requests before preflight/dispatch, while `internal/codehost/reconcile.go:148` uses detached bounded reconciliation after an ambiguous dispatch; tests cover cancellation before dispatch and after application for all four operations.
- [✓] AC-14 — `internal/mockcodehost/server.go` and `internal/codehost/github_collaboration_test.go` exercise permission denial/change, duplicate and concurrent retry, delayed/unavailable read-back, old/dismissed/other-actor reviews, marker collision, external completion, force push, cancellation, lost response, mismatched state, and partial read-back.

## Changes

- [✓] Extend the mutation journal with safe collaboration identity — `internal/codehost/idempotency.go:64` adds content-free qualified target/head/marker material and typed receipt actor/provider ID/head fields; `assertCollaborationJournal` verifies required fields and rejects body and credential canaries.
- [✓] Add common collaboration validation and preflight — `internal/codehost/collaboration.go` supplies strict payload bounds and canonical identity, and `internal/codehost/github_collaboration.go:32` adds provider-backed PR/head/actor/permission revision observation.
- [✓] Implement GitHub issue-comment creation — `internal/codehost/github_collaboration.go:253` posts one marked issue comment and validates its exact marker and authenticated actor before recording success.
- [✓] Implement neutral and decision review submission — `internal/codehost/github_collaboration.go:275` maps only the explicit operation to `COMMENT`, `APPROVE`, or `REQUEST_CHANGES`, pins `commit_id`, and validates returned marker, state, head, and actor.
- [✓] Add exact marker recovery and normalization — `internal/codehost/collaboration.go:80`, `internal/codehost/github_collaboration.go:301`, and `internal/codehost/normalize.go:242` implement injection, bounded actor/head/state-aware recovery, and strict normalized stripping.
- [✓] Extend the deterministic GitHub fake — `internal/mockcodehost/server.go` adds collaboration routes and the named write, visibility, permission, review-state, collision, external-completion, force-push, and cancellation scenarios.
- [✓] Add conformance, concurrency, cancellation, and redaction tests — `internal/codehost/github_collaboration_test.go` validates every response contract, all four operations, stable-key behavior, reconciliation, actor output, cancellation, body bounds, journal redaction, and marker fuzzing; `contracts/codehostbroker/contract_test.go` adds actor validation plus canonical and independent-consumer fixture coverage.

## Open items

- None in the Completion Ledger; every row is DONE with concrete implementation and test evidence.

## Audit notes

- None.
