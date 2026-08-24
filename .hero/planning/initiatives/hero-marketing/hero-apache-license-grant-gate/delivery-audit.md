# Delivery audit — hero-apache-license-grant-gate

**Audited:** `git diff c2124bea7cf9704a15e6092072495d6076dcfc72...HEAD` (`HEAD` `2cb6f9c53386b8eb7d19701057f06ec2eb1876a5`)
**Verdict:** HOLD
**Surface:** noteworthy

## Acceptance criteria
- [✓] Halt on unresolved third-party or repository-boundary blockers — `scripts/release_candidate.py:177-215` rejects unmapped runtime dependencies and missing license/notice files; `scripts/release_candidate.py:91-97` rejects a missing or non-canonical Hero license; the supplied candidate evidence was produced from the audited revision.
- [✓] Require explicit approval for this repository and revision — approval records `hero-engine/hero`, Apache-2.0, base `c2124bea7cf9704a15e6092072495d6076dcfc72`, authorized files, and exclusions in the spec at `spec.md:36-43`.
- [✓] Add the canonical Apache grant and required notice/metadata — new root `LICENSE` hashes to `cfc7749b96f63bd31c3c42b5c471bf756814053e847c10f3eb003417bc523d30`; root and candidate `THIRD_PARTY_NOTICES.txt` hash to `03ad2d8a4d70a98a1628131526f9419c5713dcadbb9331a18d861047dd64080c`; `scripts/release_candidate.py:244-303,380-435` emits Apache-2.0 SBOM/provenance and packages both files.
- [✓] Keep Hero Code and Hero Cloud outside the Apache grant — `README.md:241-248`, `CONTRIBUTING.md:55-57`, and `web/docs/src/index.md:69-75` retain the proprietary boundary; `internal/cli/public_docs_check.go:371-390` enforces it and `internal/cli/public_docs_check_test.go:95-103` exercises it.
- [✓] State Sprout is separately MIT-licensed and verify the consumed release — `go.mod:7` pins `github.com/bdwheeler/sprout/go@v0.1.1-0.20260822024445-cd3f0c4a2208`; the supplied module directory contains an MIT `LICENSE`; `README.md:241-248`, `web/docs/src/index.md:69-75`, and `internal/cli/public_docs_check_test.go:48-75` enforce the separate boundary.

## Changes
- [✗] Verify inventory, ownership, readiness, and release packets — candidate evidence is concrete, but the audited revision's `scripts/test_release_candidate.py:176` expects 10 `| GATED |` rows while the audited checklist contains 6; the supplied 14-test pass therefore does not apply to `HEAD` without the uncommitted correction.
- [✓] Record exact owner approval and mutation boundary — `spec.md:36-43` records the repository, approved base, license, authorized files, and explicit exclusions.
- [✓] Add canonical license and inventory-required notices/metadata — `LICENSE`, `THIRD_PARTY_NOTICES.txt`, public claims, package archives, CycloneDX metadata, provenance, release notes, and launch checklist all have concrete diff evidence.
- [✗] Verify package wording, proprietary siblings, and Sprout MIT — package evidence and core public docs are present, but the audited `.hero/marketing/positioning.md` still says this repository has a "future grant," and the audited `web/landing/site/index.html` omits the now-current Apache-2.0 status; both corrections exist only as uncommitted working-tree changes.

## Open items (if any)
- None in the ledger; all rows were claimed `DONE`.

## Audit notes
- The `DONE` full-suite claim is downgraded because its passing evidence was produced with an uncommitted test correction, not from the exact audited revision.
- The `DONE` public-wording claim is downgraded because two wording corrections required for the current grant are outside `HEAD`.
- The diff also contains meaningful unrelated Hero Sales mockups, Mail intake/feature scaffolds, and Mail/peering guidance in `CLAUDE.md`; these are outside the spec's named licensing files and make the audit surface noteworthy.
