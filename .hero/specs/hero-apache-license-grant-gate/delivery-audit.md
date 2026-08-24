# Delivery audit — hero-apache-license-grant-gate

**Audited:** `git diff c2124bea7cf9704a15e6092072495d6076dcfc72...HEAD` (`HEAD` `b0eecd46360f947c0f6d5f8e2f288e0f485f7fee`)
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria
- [✓] Halt on unresolved third-party or repository-boundary blockers — `scripts/release_candidate.py:91-97,177-215` rejects a missing/non-canonical Hero license, unmapped runtime dependencies, missing module directories, and missing license/notice files; `scripts/test_release_candidate.py:108-116,130-133` covers canonical-license and unknown-dependency rejection, and the supplied candidate evidence was produced from the audited revision.
- [✓] Require explicit approval for this repository and revision — `spec.md:36-43` records owner authorization for `hero-engine/hero`, Apache-2.0, approved base `c2124bea7cf9704a15e6092072495d6076dcfc72`, authorized files, and explicit exclusions.
- [✓] Add the canonical Apache grant and required notice/metadata — root and candidate `LICENSE` hash to `cfc7749b96f63bd31c3c42b5c471bf756814053e847c10f3eb003417bc523d30`; root and candidate `THIRD_PARTY_NOTICES.txt` hash to `03ad2d8a4d70a98a1628131526f9419c5713dcadbb9331a18d861047dd64080c`; `scripts/release_candidate.py:244-303,384-435` emits Apache-2.0 SBOM/provenance metadata and packages both files.
- [✓] Keep Hero Code and Hero Cloud outside the Apache grant — `README.md:241-248`, `CONTRIBUTING.md:55-62`, `web/docs/src/index.md:69-75`, and `web/landing/site/index.html:506-508` retain the proprietary and visibility boundaries; `internal/cli/public_docs_check.go:67-83,371-419` enforces them and `internal/cli/public_docs_check_test.go:48-75,95-126` exercises the contract.
- [✓] State Sprout is separately MIT-licensed and verify the consumed release — `go.mod:7` pins `github.com/bdwheeler/sprout/go@v0.1.1-0.20260822024445-cd3f0c4a2208`; the supplied module directory contains an MIT `LICENSE`; `README.md:241-248`, `web/docs/src/index.md:69-75`, and `internal/cli/public_docs_check_test.go:48-75,95-103` enforce the separate boundary.

## Changes
- [✓] Verify inventory, ownership, readiness, and release packets — the supplied evidence reports the full Go/docs/landing/release-validation suite passing at `b0eecd46360f947c0f6d5f8e2f288e0f485f7fee`; candidate provenance records that exact revision, its source tree, five targets, reproducibility, Apache-2.0, and unpublished status.
- [✓] Record exact owner approval and mutation boundary — `spec.md:36-43` records the repository, approved base, license, authorized files, and explicit exclusions without cross-repository authority.
- [✓] Add canonical license and inventory-required notices/metadata — `LICENSE`, `THIRD_PARTY_NOTICES.txt`, public claims, package archives, CycloneDX metadata, provenance, release notes/checklist, and fail-closed validation have concrete diff evidence.
- [✓] Verify package wording, proprietary siblings, and Sprout MIT — each inspected candidate archive contains `LICENSE` and `THIRD_PARTY_NOTICES.txt`; candidate SBOM/provenance identify Apache-2.0 and the exact unpublished revision; current marketing, landing, README, contributing, and hosted-docs wording preserve sibling/Sprout boundaries, with passing boundary and build evidence supplied.

## Open items (if any)
- None; every ledger row is `DONE` with concrete evidence.

## Audit notes
- The prior HOLD gaps are committed in `HEAD`: `scripts/test_release_candidate.py:176` expects the checklist's six gated rows, `.hero/marketing/positioning.md:145-147` states the current grant, and `web/landing/site/index.html:507` states the Apache-2.0 license while preserving public-visibility and proprietary-product boundaries.
- The audited range also contains meaningful unrelated Hero Sales mockups/spec updates, Mail intake/feature scaffolds, and Mail/peering guidance in `CLAUDE.md`; these are outside this spec's license-grant scope and make the surface noteworthy, but they do not block the evidenced delivery.
