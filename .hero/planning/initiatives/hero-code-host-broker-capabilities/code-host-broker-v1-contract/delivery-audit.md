# Delivery audit — code-host-broker-v1-contract

**Audited:** `git diff 1e3fa83...HEAD` at `693638d`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] AC-1: exactly twenty authoritative operation policies — `contracts/codehostbroker/policy.go:37-58,121-155`; `TestOperationRegistryIsCompleteAndAuthoritative` checks the exact inventory, bounds, effects, consent, freshness, idempotency, reconciliation, and replay policy.
- [✓] AC-2: repository-qualified PR identity — `contracts/codehostbroker/contract.go:103-122`, `contracts/codehostbroker/validate.go:18-48,99-108`, and `TestRepositoryQualifiedPullRequestIdentity` require the connection, complete repository, provider PR ID, and positive number.
- [✓] AC-3: distinct fork base/head repositories, refs, and SHAs — `contracts/codehostbroker/contract.go:111-122` and `TestForkRefsRoundTripWithoutCollapse`.
- [✓] AC-4: complete versioned response envelope with typed result-or-error validation — `contracts/codehostbroker/contract.go:282-339`, `contracts/codehostbroker/validate.go:139-254,364-527`, and `TestOperationSpecificResultsRejectInvalidShapesAndBounds`.
- [✓] AC-5: partial results retain returned sections and bounded failures — `contracts/codehostbroker/contract.go:220-224`, `contracts/codehostbroker/fixture.go:94-100`, and `TestPartialResultAndBoundsTruth`.
- [✓] AC-6: opaque cursors bind version, provider, connection, complete normalized repository identities, operation, query, order, and position — `contracts/codehostbroker/contract.go:398-412`, `contracts/codehostbroker/validate.go:264-321,567-635`, and `TestCursorAndRevisionFingerprintsBindMutableMaterial`. Negative cases reject provider mismatch and a same-name repository on another host; request and response checks compare complete `RepositoryIdentity` values.
- [✓] AC-7: every mutation requires user intent, exact consent, stable idempotency, capability and observation revisions, reconciliation, and a typed payload — `contracts/codehostbroker/validate.go:110-135,644-713` and `TestMutationRequestsRequirePolicyMaterial`.
- [✓] AC-8: reads, non-merge writes, and merge have the required effect and consent classes — `contracts/codehostbroker/policy.go:121-155` and `TestOperationRegistryIsCompleteAndAuthoritative`.
- [✓] AC-9: ambiguous mutation outcomes cannot advertise unsafe replay — `contracts/codehostbroker/validate.go:218-253,349-361`, `TestMutationResponsesRequireReconciliationAndExactRetry`, and `docs/contracts/code-host-broker-v1.md:256-290`.
- [✓] AC-10: all 26 normalized errors and five retry values are closed, documented, tested, and fixture-covered — `contracts/codehostbroker/policy.go:157-220`, `TestErrorAndRetryEnumsAreClosedAndFixtureComplete`, and `docs/contracts/code-host-broker-v1.md:270-286`.
- [✓] AC-11: independent consumers tolerate additive fields and capabilities while unsupported major versions fail closed — `TestUnknownAdditiveFieldsDecodeAndMajorVersionFailsClosed`, `TestFixtureDecodesWithIndependentConsumerShapes`, and `contracts/codehostbroker/validate.go:65-71,139-146`.
- [✓] AC-12: every published bound is enforced — `contracts/codehostbroker/policy.go:3-35`, `contracts/codehostbroker/validate.go:65-135,139-254,364-527,541-745`, and `TestEveryPublishedBoundHasEnforcementEvidence`. The 100-scope maximum includes the primary repository; revisions, rate-limit resource/reset time, error field/retry time, and RFC3339 time fields are bounded and negatively tested.
- [✓] AC-13: fixture generation is byte-stable and its SHA-256 matches the sidecar — `contracts/codehostbroker/fixture.go:13-146`, `contracts/codehostbroker/cmd/generate/main.go:11-30`, and `TestFixtureIsByteStableAndMatchesPublishedDigest`; both committed artifacts contain digest `14421657e12d8b1bc31587f0dcb7a179fd17c0671467a16cb8c30ab30998e5d3`.
- [✓] AC-14: safe material excludes credential/body fields and fixture mutation text is redacted — `contracts/codehostbroker/contract.go:226-243`, `contracts/codehostbroker/fixture.go:11,148-195`, and `TestFixtureAndErrorsContainNoCredentialCanaries`.

## Changes

- [✓] Add canonical contract types and validation — `contracts/codehostbroker/contract.go` and `contracts/codehostbroker/validate.go` define and validate the provider-neutral identities, typed results and payloads, envelope, availability states, and bounds.
- [✓] Add a single operation-policy registry — `contracts/codehostbroker/policy.go` is the sole ordered policy, bounds, normalized-error, and retry inventory.
- [✓] Define cursor and revision material — `contracts/codehostbroker/contract.go:398-423` and `contracts/codehostbroker/validate.go:257-347` provide bounded opaque cursors, complete normalized repository scope, integrity fingerprints, request/response binding, and deterministic revisions.
- [✓] Define mutation receipt and reconciliation outcomes — `contracts/codehostbroker/contract.go:226-243`, `contracts/codehostbroker/validate.go:218-253`, and the fixture cover safe receipts and all seven outcomes.
- [✓] Publish the contract documentation — `docs/contracts/code-host-broker-v1.md` documents the complete v1 wire contract and `docs/contracts/README.md` registers it in the cross-language contract index.
- [✓] Generate the canonical consumer fixture and digest — `contracts/codehostbroker/fixture.go`, `contracts/codehostbroker/cmd/generate/main.go`, and `contracts/codehostbroker/testdata/v1/` cover all operations, policy/state variants, additive material, redacted writes, errors, and the published digest.
- [✓] Add validation, round-trip, fuzz, bounds, golden, and redaction tests — `contracts/codehostbroker/contract_test.go` contains the asserted coverage; both committed `FuzzCursorRoundTrip` regression corpora are valid Go fuzz corpus files and exercise non-UTF-8 repository material, which `tooLong`/`ValidateRepository` reject.

## Audit notes

- No ledger `DONE` claim was downgraded. The prior AC-6 and AC-12 blockers are remediated without regressing the other criteria.
- Recorded evidence includes deterministic generation, focused and full Go tests, identity and cursor fuzzing, full vet, diff checking, 14/14 EARS lint, and all drift items addressed. Tests were not rerun during this cold artifact audit.
