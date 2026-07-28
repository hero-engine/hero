# Delivery audit — code-host-broker-v1-contract

**Audited:** `git diff 1e3fa83...HEAD` at `2483699`
**Verdict:** HOLD
**Surface:** noteworthy

## Acceptance criteria

- [✓] AC-1: exactly twenty authoritative operation policies — `contracts/codehostbroker/policy.go:37-58`, `TestOperationRegistryIsCompleteAndAuthoritative`.
- [✓] AC-2: repository-qualified PR identity — `contracts/codehostbroker/validate.go:34-48`, `TestRepositoryQualifiedPullRequestIdentity`.
- [✓] AC-3: distinct fork base/head repositories, refs, and SHAs — `contracts/codehostbroker/contract.go:111-122`, `TestForkRefsRoundTripWithoutCollapse`.
- [✓] AC-4: complete envelope with operation-specific result validation — `contracts/codehostbroker/contract.go:315-339`, `contracts/codehostbroker/validate.go:139-237,346-509`, `TestOperationSpecificResultsRejectInvalidShapesAndBounds`.
- [✓] AC-5: partial results retain data and bounded section failures — `contracts/codehostbroker/fixture.go:94-100`, `TestPartialResultAndBoundsTruth`.
- [✗] AC-6: opaque cursors bind every originating dimension — request validation never compares `CursorMaterial.Provider`, and repository scope is reduced to `RepositoryIdentity.FullName`, dropping host and provider repository ID (`contracts/codehostbroker/validate.go:549-575,578-585`). A cursor from another provider, or from a same-named repository on another host, can therefore pass the claimed binding. `TestCursorAndRevisionFingerprintsBindMutableMaterial` checks query mismatch and token corruption but not either omitted identity dimension.
- [✓] AC-7: every mutation requires intent, consent, idempotency, revisions, reconciliation, and typed payload — `contracts/codehostbroker/validate.go:110-135`, `TestMutationRequestsRequirePolicyMaterial`.
- [✓] AC-8: reads, non-merge writes, and merge have the required effect/consent classes — `contracts/codehostbroker/policy.go:121-155`, `TestOperationRegistryIsCompleteAndAuthoritative`.
- [✓] AC-9: unknown mutation outcomes and retry guidance are represented safely — `contracts/codehostbroker/validate.go:209-237,331-344`, `TestMutationResponsesRequireReconciliationAndExactRetry`, and the lifecycle rules at `docs/contracts/code-host-broker-v1.md:256-290`.
- [✓] AC-10: all 26 normalized errors and five retry values are closed and fixture-covered — `contracts/codehostbroker/policy.go:157-220`, `TestErrorAndRetryEnumsAreClosedAndFixtureComplete`.
- [✓] AC-11: additive fields/capabilities are consumer-tolerated and incompatible major versions fail closed — `TestUnknownAdditiveFieldsDecodeAndMajorVersionFailsClosed`, `TestFixtureDecodesWithIndependentConsumerShapes`, and `contracts/codehostbroker/validate.go:66-71,139-146`.
- [✗] AC-12: every published bound is enforced — `ValidateRequest` bounds only the additional `repositories` slice, although the primary repository is also part of the scope, so 100 additional repositories plus the primary (101 total) is accepted (`contracts/codehostbroker/validate.go:79-87`; `docs/contracts/code-host-broker-v1.md:91-95,296-315`). Response revisions are checked only for non-emptiness, and rate-limit `resource`/`reset_at` plus error `field`/`retry_at` have no length bounds (`contracts/codehostbroker/validate.go:156-163,177-180,209-218`). `TestEveryPublishedBoundHasEnforcementEvidence` does not exercise these gaps.
- [✓] AC-13: deterministic fixture bytes and SHA-256 sidecar — `TestFixtureIsByteStableAndMatchesPublishedDigest`; the on-disk fixture and sidecar both read `e96d4ea5e60c8707698188db3096477a320ec4ccf3417dd8faf4983f37640bb5`.
- [✓] AC-14: safe material excludes body-bearing fields and fixture mutation text is redacted — `contracts/codehostbroker/contract.go:226-243`, `contracts/codehostbroker/fixture.go:11,148-195`, `TestFixtureAndErrorsContainNoCredentialCanaries`.

## Changes

- [✗] Add canonical contract types and validation — operation-specific validation landed, but the AC-12 bound gaps remain.
- [✓] Add a single operation-policy registry — `contracts/codehostbroker/policy.go`.
- [✗] Define cursor and revision material — cursor encoding/fingerprinting landed, but request-side provider and canonical repository identity binding remain incomplete.
- [✓] Define mutation receipt and reconciliation outcomes — `contracts/codehostbroker/contract.go:226-243`, `contracts/codehostbroker/validate.go:219-235`.
- [✓] Publish the contract documentation — `docs/contracts/code-host-broker-v1.md` documents the inventory, DTO/nullability catalog, lifecycle cases, errors, compatibility, and bounds.
- [✓] Generate canonical consumer fixture and digest — the fixture covers 20 known operations plus one unknown advertised operation, all availability/completeness/reconciliation states, terminal/non-terminal pages, redacted mutation fields, and all errors.
- [✗] Add validation, round-trip, fuzz, bounds, golden, and redaction tests — these test classes landed, including the committed invalid-UTF-8 cursor corpus, but cursor identity and total-scope/metadata bound negative cases are absent.

## Audit notes

- All ledger rows are marked `DONE`; AC-6, AC-12, and Changes 1, 3, and 7 are downgraded because the on-disk implementation does not support those completion claims.
- The prior AC-4, AC-9, AC-11, and AC-14 blockers were independently rechecked and are remediated: typed producer decoding, exact retry/reconciliation checks, independent consumer shapes with additive tolerance, full fixture state coverage, and redacted fixture text are present.
- Recorded validation evidence reports focused tests, both fuzz targets, the final full Go test suite, vet, diff check, EARS lint, and drift checks passing. Per the cold-audit instruction, those commands were not rerun.
