# Delivery audit — github-pull-request-merge-broker

**Audited:** `git diff 81cd8af..9316361`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] AC-1 — `mergeRuntimeCapability` derives repository-permitted methods, credential permission, and exact-base queue policy into a bounded runtime revision; `TestMergeCapabilitiesAreRepositorySpecificAndQueueFailsClosed` proves both executable and queue-required unavailable forms.
- [✓] AC-2 — `ValidateRequest`, `validateMutationPayload`, strict `decodeMergePayload`, and `validateMergeScope` require the repository-qualified PR, exact head/base, supported method, user intent, key material, revisions, and explicit acceptance before execution.
- [✓] AC-3 — the authoritative registry declares merge `commitment` / `explicit_acceptance`; `TestMergeRequiresExplicitAcceptanceBeforeProviderAccess` proves weaker consent reaches neither credentialed provider reads nor the merge endpoint.
- [✓] AC-4 — `observeMergePreflight` requires an open non-draft PR plus exact head/base and complete/current checks, reviews, protection, permission, mergeability, queue, and method evidence; `TestMergeReadinessUnknownBlockedPartialAndRateLimitedNeverDispatch` and the race table assert zero writes for every rejected class.
- [✓] AC-5 — `dispatchMerge` sends the expected head as GitHub's `sha`; the direct-request assertion and provider-force-push case prove exact dispatch and safe conflict handling.
- [✓] AC-6 — queue-required bases advertise merge unavailable with `queue_supported:false`, empty methods, and zero direct writes. `MergeMutationResult` is a closed `merged` / `queued` result, and `ValidateResponse` requires the merge-commit or queue identity to equal the safe provider receipt exactly; `TestMergeMutationResultAndReceiptAreClosedAndExact` exercises the future queued form and mismatch rejection.
- [✓] AC-7 — exact pre-dispatch external completion returns `externally_completed` without a write. `reconcileExistingMutation` also preserves that classification when `ProviderAttempts == 0`; `TestMergeRetryAfterNoDispatchClassifiesLaterExternalCompletion` proves stale-observation failure → external merge → same-key retry remains zero-dispatch and externally completed.
- [✓] AC-8 — ambiguous PUT responses use detached read-back that requires exact PR identity, head, base, merged state, and merge commit identity; lost-response and unavailable-read-back tests prove `reconciled_applied` versus `ambiguous`.
- [✓] AC-9 — both preflight and reconciliation return conflict for a merge at a different head or base; the conflicting-head scenario asserts no requested effect is reported as applied.
- [✓] AC-10 — the durable journal reconciles an existing key before runtime capability access or dispatch; lost-response, ambiguous, and permission-change retry tests each assert one provider merge attempt.
- [✓] AC-11 — `canonicalMergeDigest` binds PR, head/base, method, title/message, intent, consent, and reconciliation key; the changed-payload same-key test asserts `idempotency_conflict` and no second write.
- [✓] AC-12 — cancellation before dispatch records no provider effect, while cancellation after provider application enters detached reconciliation and returns the proven applied result.
- [✓] AC-13 — the delivery adds only repository/readiness reads and GitHub's PR merge PUT; no branch update/delete, auto-merge, tracker/spec transition, deployment, or release path appears in the delivery diff.
- [✓] AC-14 — `internal/mockcodehost` and focused broker tests cover permissions, methods, queue-only and partial queue policy, rate limits, force pushes, stale/partial checks and reviews, protection changes, duplicates, external completion, conflicting heads, cancellation, and ambiguous read-back; the canonical fixture covers the additive merge contract.

## Changes

- [✓] Runtime merge capability material — `MergeCapability`, response validation, GitHub policy discovery, canonical fixture material, and contract documentation landed.
- [✓] Fail-closed merge proof — exact provider identity, lifecycle, head/base, actor, readiness, queue, and runtime method evidence gate dispatch in `observeMergePreflight`.
- [✓] Explicit acceptance — registry policy and broker request validation enforce commitment consent before adapter resolution.
- [✓] Direct GitHub merge — merge, squash, and rebase dispatch through the bounded PUT transport with expected-head protection and post-write proof.
- [✓] Queue unsupported boundary — exact base-branch queue discovery makes the operation unavailable and performs no direct merge write.
- [✓] Durable receipts and reconciliation — journal targets retain exact head/base; safe receipts retain merge commit identity; typed merged/queued results must match that receipt; zero-dispatch external completion remains distinct from replay.
- [✓] Deterministic fake — merge policy, permission/readiness races, response loss, external effects, conflicting heads, cancellation, and rate limiting are controllable scenarios with recorded merge requests.
- [✓] Contract, broker, and safety tests — focused contract/broker/fake coverage, generated fixture digest, full suite, vet, race suite, spec lint, and diff checks are recorded as passing.

## Open items

None.

## Audit notes

The prior AC-6 and AC-7 HOLD findings are closed by commit `9316361`. Recorded post-fix validation passed `go test ./internal/codehost ./internal/mockcodehost ./contracts/codehostbroker`, `go test ./... -count=1`, `go vet ./...`, the focused race suite, deterministic fixture generation at SHA-256 `d32eb13200e9d36fd51b8c2240aa171f90ee9fc6c348f58ea35f695b469dde10`, 14/14 EARS lint, and `git diff --check`. Per the cold-audit contract, tests were inspected but not rerun.
