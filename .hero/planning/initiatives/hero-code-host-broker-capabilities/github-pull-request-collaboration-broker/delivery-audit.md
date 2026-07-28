# Delivery audit — github-pull-request-collaboration-broker

**Audited:** `git diff dfe781d...162e171`
**Verdict:** HOLD
**Surface:** noteworthy

## Acceptance criteria

- [✓] AC-1 — `codehostbroker.ValidateRequest`, `decodeCollaborationPayload`, and `PrepareCollaboration` require the qualified PR, head SHA, explicit user intent, stable keys, and both revisions; `TestCollaborationPreflightPermissionsStateAndFreshness` exercises the required fields.
- [✓] AC-2 — `internal/codehost/broker.go:37` advertises the four distinct operations, and `TestCollaborationOperationsAdvertisedAndApplied` checks each operation's `external_write`/`explicit_user` policy.
- [✓] AC-3 — `internal/codehost/github_collaboration.go:109` performs the open-state, provider-ID, head, permission, and actor preflight before `executeMutation` increments provider attempts; `TestCollaborationPreflightPermissionsStateAndFreshness` covers unsupported, closed, stale, and forbidden cases.
- [✓] AC-4 — `internal/codehost/collaboration.go:80` derives one fixed-size marker from operation identity, `validateCollaborationPayload` reserves its bytes, and `TestCollaborationOperationsAdvertisedAndApplied` checks one exact marker in every dispatched operation.
- [✗] AC-5 — provider receipt ID, qualified PR, head, outcome, and a structured actor are persisted, but the public mutation result has no actor field. `createReceipt` instead embeds only the actor login in the undocumented string `target_revision = "head:<sha>;actor:<login>"` (`internal/codehost/mutations.go:113`), dropping the actor provider ID. `TestCollaborationOperationsAdvertisedAndApplied` asserts string containment rather than a typed actor. This does not deliver the criterion's returned actor identity through the provider-neutral v1 contract.
- [✓] AC-6 — `internal/codehost/github_collaboration.go:301` performs bounded marker/actor/head/state read-back, and `TestCollaborationLostDelayedAmbiguousAndCancelledResponses` proves all four lost-response paths reconcile without a second write.
- [✓] AC-7 — `internal/codehost/reconcile.go:148` records ambiguous outcomes when exact proof is absent; delayed, unavailable, marker-collision, mismatched-state, and partial-read-back tests assert ambiguity and one provider attempt.
- [✓] AC-8 — `internal/codehost/github_collaboration.go:134` recognizes the authenticated actor's current-head `APPROVED` or `CHANGES_REQUESTED` review, and `TestCollaborationExternallyCompletedReviewRules` asserts zero writes.
- [✓] AC-9 — `authoritativeCurrentHeadReview` binds actor/head and rejects a terminal dismissed review; old-head, dismissed, and other-actor scenarios each assert one new requested effect.
- [✓] AC-10 — the shared locked journal in `internal/codehost/reconcile.go` serializes retries; sequential and concurrent tests cover all four operations and assert one provider effect plus a stable receipt.
- [✓] AC-11 — `canonicalCollaborationDigest` binds operation, qualified target, head, body digest, intent, consent, and reconciliation key beneath a connection-scoped journal key; conflict tests exercise operation/review-state, target, head, and body changes before another write.
- [✓] AC-12 — `stripHeroMarkers` removes only the exact bounded grammar, and normalization, malformed-lookalike, and fuzz tests preserve all other content.
- [✓] AC-13 — `executeMutation` checks cancellation before preflight/dispatch and uses a detached bounded reconciliation context after ambiguous dispatch; tests cover cancellation before and after every operation.
- [✓] AC-14 — `internal/mockcodehost` and `github_collaboration_test.go` provide asserted scenarios for permissions and permission changes, duplicate/concurrent retries, delayed visibility, dismissed/old-head reviews, collision, external completion, force push, cancellation, lost response, and partial read-back.

## Changes

- [✓] Extend the mutation journal with safe collaboration identity — `internal/codehost/idempotency.go:64` adds PR/head/marker target material and structured receipt actor/provider ID/head fields; journal canaries reject body and credential persistence.
- [✓] Add common collaboration validation and preflight — `internal/codehost/collaboration.go` and `internal/codehost/github_collaboration.go:32` add bounded decoding plus provider-backed state, head, actor, permission, and revision observation.
- [✓] Implement GitHub issue-comment creation — `internal/codehost/github_collaboration.go:253` posts a marked comment and validates the returned actor and marker.
- [✓] Implement neutral and decision review submission — `internal/codehost/github_collaboration.go:275` maps the three review operations to `COMMENT`, `APPROVE`, and `REQUEST_CHANGES`, pins `commit_id`, and validates state/head/actor/marker.
- [✓] Add exact marker recovery and normalization — `internal/codehost/collaboration.go:80`, `internal/codehost/github_collaboration.go:301`, and `internal/codehost/normalize.go:244` implement injection, bounded exact recovery, and strict stripping.
- [✓] Extend the deterministic GitHub fake — `internal/mockcodehost/server.go` adds collaboration routes and all named response/state scenarios.
- [✓] Add conformance, concurrency, cancellation, and redaction tests — `internal/codehost/github_collaboration_test.go` covers the declared behavior, response validation, retries, cancellation, redaction, reconciliation, and marker fuzzing.

## Open items

- None in the Completion Ledger; every row was marked DONE.

## Audit notes

- AC-5 is performative as written: the durable journal has a typed actor, but the public v1 response does not. Packing a login into opaque `target_revision` is neither the documented fixture shape nor a typed actor identity and is not a reliable cross-repo contract for the Swift consumer.
- The ledger reports focused and race-enabled code-host tests, 130,125 marker-fuzz executions, `go test ./... -count=1`, `go vet ./...`, formatting, and diff hygiene as passing. The corresponding asserted test bodies are present in the audited commit; the cold audit did not rerun tests.
