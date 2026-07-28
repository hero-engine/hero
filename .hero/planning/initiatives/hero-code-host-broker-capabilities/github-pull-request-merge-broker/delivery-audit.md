# Delivery audit — github-pull-request-merge-broker

**Audited:** `git diff 81cd8af..6e804ac`
**Verdict:** HOLD
**Surface:** noteworthy

## Acceptance criteria

- [✓] AC-1 — repository-specific methods, queue policy, permission state, and a runtime revision are advertised by `mergeRuntimeCapability`; `TestMergeCapabilitiesAreRepositorySpecificAndQueueFailsClosed` asserts the direct and queue-required forms.
- [✓] AC-2 — `MergePayload`, request validation, strict decoding, and `validateMergeScope` require the qualified PR, exact head/base, method, intent, keys, revisions, and acceptance material before dispatch.
- [✓] AC-3 — the operation registry classifies merge as `commitment` / `explicit_acceptance`, and `TestMergeRequiresExplicitAcceptanceBeforeProviderAccess` proves rejection before provider access.
- [✓] AC-4 — `observeMergePreflight` fails closed across lifecycle, draft, exact head/base, complete readiness, queue, permissions, checks, reviews, protection, mergeability, and method support; the readiness and race table asserts zero merge attempts for rejected states.
- [✓] AC-5 — `dispatchMerge` sends the expected head as `sha`; `TestMergeDispatchesExactHeadAndReturnsTypedReceipt` and the provider-force-push case assert the request and safe conflict behavior.
- [✗] AC-6 — the current GitHub adapter correctly advertises `queue_supported:false` and performs no direct write for queue-required bases, but the claimed future queue contract is absent. `MergeCapability` permits an available `queue_supported:true` capability (`contracts/codehostbroker/validate.go:398-402`), while `MutationResult` has only a free-form `outcome` and the response has only a generic receipt (`contracts/codehostbroker/contract.go:318-323`); no typed queued state or required queue identity/receipt is defined or validated.
- [✗] AC-7 — the clean first-attempt external-completion path is tested, but retries after a proven no-dispatch failure can misclassify a later external merge as `replayed`. `reconcileExistingMutation` reconciles every existing entry before checking `journalNotApplied` / `ProviderAttempts == 0` and unconditionally emits `ReconciliationReplayed` for an exact effect (`internal/codehost/reconcile.go:216-229`). That violates the required `externally_completed` classification for a merge that occurred before Hero ever dispatched.
- [✓] AC-8 — ambiguous PUT responses enter detached exact head/base/commit read-back; the lost-response and unavailable-read-back tests assert reconciled-applied versus ambiguous.
- [✓] AC-9 — `reconcileMerge` returns conflict for different head/base evidence, and the conflicting-head test asserts no merge attempt.
- [✓] AC-10 — the journal gates dispatch and the lost-response, ambiguity, and replay tests assert one provider merge attempt across duplicate retries.
- [✓] AC-11 — `canonicalMergeDigest` binds PR, head/base, method, title/message, intent, consent, and reconciliation key; the changed-payload test asserts `idempotency_conflict`.
- [✓] AC-12 — pre-dispatch cancellation and cancellation after an applied write are both asserted, with the latter entering detached reconciliation.
- [✓] AC-13 — the committed adapter adds only repository policy/readiness reads and GitHub's pull-request merge PUT; no branch, tracker, deployment, release, or auto-merge effect was added.
- [✓] AC-14 — the fake and focused tests cover method policy, queue-only behavior, rate limiting, force pushes, stale/partial readiness, protection and permission changes, duplicate retries, external completion, conflicting heads, cancellation, and ambiguous responses; the canonical fixture includes merge policy, payload, result, receipt, and runtime capability material.

## Changes

- [✓] Runtime merge capability material — `MergeCapability`, validation, canonical fixture, docs, and repository-specific discovery landed.
- [✓] Fail-closed merge proof — `observeMergePreflight` and its readiness/race tests cover the required evidence.
- [✓] Explicit acceptance — authoritative policy and broker validation reject weaker consent before provider access.
- [✓] Direct GitHub merge — merge/squash/rebase dispatch with expected-head protection and bounded normalized output landed.
- [✓] Queue unsupported boundary — queue-required repositories advertise merge unavailable and `PrepareMerge` performs no direct write.
- [✗] Durable receipts and reconciliation — direct-merge receipts retain exact head/base/commit identity, but the future typed queued receipt/state is not modeled or validated, and a no-dispatch journal retry can label a later external merge as replayed rather than externally completed.
- [✓] Deterministic fake — merge policies, races, response loss, external effects, and cancellation scenarios landed.
- [✓] Contract, broker, and safety tests — focused contract/broker/fake tests and canonical fixture assertions landed.

## Open items

- None were declared in the Completion Ledger.

## Audit notes

- AC-6 and Change 6 were marked `DONE`, but the code only implements the fail-closed unsupported-queue half. Add a closed typed queued result and exact queue identity/receipt contract (plus validation/fixture tests), or remove the future-supported branch from this spec and contract until that child is delivered.
- AC-7 was marked `DONE`, but the journal replay path does not distinguish a zero-attempt `not_applied` entry from a previously applied entry when later reconciliation observes an exact merge. Classify that state as external completion and add a test covering no-dispatch failure → external merge → same-key retry.
- Recorded validation evidence says `go test ./... -count=1`, `go vet ./...`, and race tests for codehost/mock/contract packages passed. Per the cold-audit instruction, tests were not rerun.
