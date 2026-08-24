# Delivery audit — remove-invented-preview-marketing-copy

**Audited:** complete working-tree diff (`git diff`), cold re-audit
**Verdict:** SHIP
**Surface:** clean
**Confidence:** high

## Acceptance criteria

- [✓] AC-1 — The claim registry classifies `product-two-system-loop` as Class A/shipped, and the changed positioning, README, hosted docs, release notes, and landing copy describe the reinforcing model without the removed preview/proof/perfection qualifier (`.hero/specs/hero-public-truth-baseline/public-claim-registry.yaml:47`, `.hero/marketing/positioning.md:23`, `README.md:6`, `web/landing/site/index.html:380`).
- [✓] AC-2 — The landing source no longer contains `.preview-note`, `.trust-strip`, the repository-boundary block, or the artifact-revision footer; the remaining footer contains only `Hero Engine` (`web/landing/site/index.html:380`, `web/landing/site/index.html:492`, `web/landing/site/index.html:525`).
- [✓] AC-3 — The head retains `hero-source-revision` and `web/landing/site/revision.json` retains machine-readable provenance, while no build placeholder remains in visible landing markup (`web/landing/site/index.html:19`, `web/landing/site/revision.json`).
- [✓] AC-4 — The diff updates the claim registry, positioning authority, README, affected hosted-docs pages, and v0.34 candidate notes; the audited public narrative sources contain none of the removed qualifiers.
- [✓] AC-5 — `inventedContinuityRules` applies to public narrative surfaces and the dedicated `publicTruthAuthorityPaths`; mutation coverage exercises positioning, the claim registry, and v0.34 candidate notes, release-candidate tests reject the removed phrases, and visible landing provenance is independently rejected (`internal/cli/public_docs_check.go:86`, `internal/cli/public_docs_check.go:91`, `internal/cli/public_docs_check_test.go:97`, `scripts/test_release_candidate.py:159`).
- [✓] AC-6 — Legitimate headless-runtime preview labels remain in README, hosted docs, positioning authority, claim registry, and v0.34 candidate notes (`README.md:180`, `web/docs/src/index.md:41`, `docs/releases/v0.34.0-candidate.md:52`).

## Changes

- [✓] Correct the canonical product claim and positioning authority — both now use Class A/shipped language backed by implementation evidence.
- [✓] Refresh every affected public narrative surface — all named surfaces in the diff carry the corrected memory-and-delivery message.
- [✓] Remove internal launch/provenance UI from the landing page — the preview banner, trust strip, visible revision link, timestamp, and artifact-revision footer are removed while head and JSON provenance remain.
- [✓] Replace bad-copy tests with positive and negative regression coverage — landing checks reject invented qualifiers and visible provenance, Go mutation tests cover public narratives plus every authority path, and release-candidate tests assert the removed phrases stay absent.

## Open items

None. The Completion Ledger contains no PARTIAL, SKIPPED, or BLOCKED rows.

## Validation evidence

- `go test ./...` — PASS.
- Landing validation — PASS (8 tests, strict build, and artifact checks).
- Hosted-documentation validation — PASS (28 tests, strict build, and link check).
- Release-candidate validation — PASS (14 tests).
- Public-documentation contract and invocation validation — PASS.
- Focused authority-mutation tests and `git diff --check` — PASS.

## Audit notes

None.
