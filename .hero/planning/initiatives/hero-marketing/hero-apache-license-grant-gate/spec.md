---
title: "Hero Apache-2.0 License Grant Gate"
slug: hero-apache-license-grant-gate
type: feature
status: delivering
domain: engineering
size: small
priority: critical
horizon: now
created: 2026-08-21
tags: [apache-2, licensing, approval-gate, legal]
parent: hero-marketing
depends-on: [hero-v034-release-prep, hero-licensing-boundary-and-provenance, hero-public-repo-readiness]
delivery_method: manual
---

# Hero Apache-2.0 License Grant Gate

## Goal

After owner authorization and all third-party inventory blockers are clear, obtain final mutation approval, then add the canonical Apache License 2.0 grant and required notices to this `hero` repository only.

## Kickoff

Adds Apache-2.0 to the `hero` repository while keeping visibility, publication, Hero Code, Hero Cloud, and Sprout outside the mutation.

**Status:** in-review — the licensed source and reproducible candidate are complete; cold audit and verification are the remaining closing gates.

**Pick up at:** cold-audit the ledger and exact candidate evidence, then close through `hero spec verify` if the verdict is SHIP.

→ `.hero/planning/initiatives/hero-marketing/hero-apache-license-grant-gate/spec.md`

**Files:** `LICENSE`, `THIRD_PARTY_NOTICES.txt`, `scripts/release_candidate.py`, `internal/cli/public_docs_check.go`, this spec
**Skip:** repository visibility, release publication, Sprout mutation, and any license change to `hero-code` or `hero-cloud`.

## Approval

- **Owner authorization:** the repository owner explicitly directed this task to add Apache-2.0 to the `hero` repository and deliver the prepared documentation, package, and license updates.
- **Approved repository:** `/Users/developer/projects/hero-engine/repository/hero` (`hero-engine/hero`) only.
- **Approved base revision:** `c2124bea7cf9704a15e6092072495d6076dcfc72`.
- **Approved license:** Apache License 2.0.
- **Authorized files:** root `LICENSE`, root `THIRD_PARTY_NOTICES.txt`, and the public license-status, validation, release-package, and spec evidence needed to make that grant accurate.
- **Explicit exclusions:** no license grant or mutation for Hero Code, Hero Cloud, or Sprout; no repository visibility change, tag, release publication, docs/landing deployment, or launch announcement.

## Changes

1. Verify the licensing inventory, owner authorization, repository-readiness, and v0.34 packets have zero unresolved grant blockers and present the exact proposed mutation to the owner.
2. Record explicit approval identifying this `hero` repository, Apache License 2.0, the approved revision, and the authorized files; halt without mutation if approval is absent or narrower.
3. Add the canonical Apache License 2.0 text at root and only the NOTICE/attribution and metadata changes demonstrated necessary by the inventory.
4. Verify generated packages and public wording identify this repository's license accurately, preserve the proprietary status of `hero-code` and `hero-cloud`, and identify Sprout's separate MIT license.

## Acceptance Criteria

- **AC-1:** IF any third-party asset, dependency, notice, or repository-boundary blocker is unresolved THEN THE SYSTEM SHALL halt without adding or changing license files.
- **AC-2:** IF explicit owner approval for Apache-2.0 on this exact repository/revision is absent THEN THE SYSTEM SHALL leave the repository license state unchanged.
- **AC-3:** WHEN approval is recorded THE SYSTEM SHALL add the canonical Apache License 2.0 grant and only inventory-required notice/metadata changes.
- **AC-4:** THE SYSTEM SHALL NOT license or imply an Apache-2.0 grant for `hero-code` or `hero-cloud`.
- **AC-5:** THE SYSTEM SHALL state that Sprout is MIT-licensed separately and verify the consumed Sprout module release contains that license.

## Boundaries

- No inferred, blanket, or cross-repository approval.
- No visibility change, public release, tag, or launch announcement.
- No mutation of Sprout, `hero-code`, or `hero-cloud`.

## Validation

- Compare license/notice text byte-for-byte with the canonical approved forms.
- Re-run the cleared dependency, asset, package-metadata, owner-authorization, and proprietary-boundary checks.
- Confirm visibility and release state remain unchanged.

## Completion Ledger

The approved Apache-2.0 grant is applied to this repository, enforced by the public-docs contract, represented in the reproducible v0.34 candidate, and bounded away from visibility and proprietary sibling products.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Halt on unresolved third-party or boundary blockers | DONE | The completed licensing inventory and repository-readiness specs report the grant blockers cleared; `scripts/release_candidate.py` still fails closed on unknown runtime dependency licenses, missing license files, or non-canonical Hero license text. |
| 2 | Require explicit approval for this repository and revision | DONE | `## Approval` records owner authorization for `hero-engine/hero`, Apache-2.0, approved base `c2124bea7cf9704a15e6092072495d6076dcfc72`, authorized files, and explicit exclusions. |
| 3 | Add canonical Apache grant and required notice/metadata only | DONE | Root `LICENSE` matches the ASF text at SHA-256 `cfc7749b96f63bd31c3c42b5c471bf756814053e847c10f3eb003417bc523d30`; root and candidate notices match at `03ad2d8a4d70a98a1628131526f9419c5713dcadbb9331a18d861047dd64080c`; SBOM and provenance report Apache-2.0. |
| 4 | Do not grant Apache-2.0 to Hero Code or Hero Cloud | DONE | `README.md`, `CONTRIBUTING.md`, `web/docs/src/index.md`, marketing positioning, and executable public-doc tests keep both sibling products separate and proprietary. |
| 5 | Keep Sprout separately MIT and verify consumed release | DONE | Hero consumes `github.com/bdwheeler/sprout/go@v0.1.1-0.20260822024445-cd3f0c4a2208`; its module directory contains the MIT license, while public wording and drift tests keep it outside Hero's grant. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Verify inventory, ownership, readiness, and release packets | DONE | Preconditions were reconciled against the completed licensing, repository-readiness, public-docs, and v0.34 release-prep specs; the full test/build suite and licensed candidate passed. |
| 2 | Record exact owner approval and mutation boundary | DONE | `## Approval` captures the repository, approved base revision, license, authorized files, and exclusions without inferring cross-repository authority. |
| 3 | Add canonical license and inventory-required notices/metadata | DONE | Added root `LICENSE` and `THIRD_PARTY_NOTICES.txt`; updated public claims, package archives, SBOM/provenance, release notes/checklist, and fail-closed validation. |
| 4 | Verify package wording, proprietary siblings, and Sprout MIT | DONE | The final clean licensed revision rebuilt reproducibly for five targets; every archive contains the license/notices, the SBOM is Apache-2.0, provenance records the exact source revision and unpublished state, and boundary tests pass. |

### Exercise-the-feature check

- [x] User-visible behavior was exercised end-to-end: `python3 scripts/release_candidate.py --version v0.34.0 --base v0.33.0 --output .build/release-candidate/v0.34.0` built twice byte-identically, smoked the native archive through version/init/Codex install/status/check, and emitted five unpublished archives whose provenance records the exact clean licensed revision; `/tmp/hero-apache-gate docs check --public --invocations`, `go test ./...`, 28 docs tests, 8 landing tests, and `goreleaser check` passed.

### Excellence Bar self-check

- [x] Yes — the grant is exact, machine-enforced, package-complete, evidence-linked, and deliberately stops before any irreversible public visibility or release action.
