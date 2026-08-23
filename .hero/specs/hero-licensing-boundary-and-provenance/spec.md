---
title: "Hero Licensing Boundary and Open-Source Inventory"
slug: hero-licensing-boundary-and-provenance
type: feature
status: completed
domain: engineering
size: small
priority: critical
horizon: now
created: 2026-08-21
tags: [licensing, apache-2, ownership, dependencies, assets]
parent: hero-marketing
depends-on: [hero-public-truth-baseline]
delivery_method: manual
completed_at: 2026-08-23T16:28:31Z
---

# Hero Licensing Boundary and Open-Source Inventory

## Goal

Produce the bounded packet needed to add Apache-2.0 to this `hero` repository: record the sole owner's authorization, define the exact repository boundary, inventory third-party dependencies and bundled assets, and name any notice or compatibility blocker.

## Kickoff

Records owner authorization and clears third-party licensing before any surface says “open source” or any license file is added.

**Status:** completed — owner authorization, repository boundaries, third-party compatibility, notice obligations, and fail-closed grant preconditions are recorded. The license and visibility mutations remain gated.

**Pick up at:** record the owner's authorization and inventory tracked source, generated material, vendored content, docs assets, and dependency manifests without adding the license or changing visibility.

→ `.hero/planning/initiatives/hero-marketing/hero-licensing-boundary-and-provenance/spec.md`

**Files/components:** `go.mod`, `go.sum`, documentation dependency manifests, web dependency manifests, tracked media/fonts/logos/screenshots, generated distributions, and a licensing inventory beside this spec
**Skip:** adding `LICENSE`, changing repository visibility, editing Sprout, or inspecting proprietary repositories beyond recording their exclusion.

## Changes

1. Write a repository-boundary report that identifies this `hero` CLI/repository as the proposed Apache-2.0 grant and explicitly excludes `hero-code` and `hero-cloud` as proprietary.
2. Record Sprout as a separately owned, public MIT-licensed dependency at `bdwheeler/sprout`; verify that the exact version consumed by Hero carries the expected license metadata.
3. Inventory direct/transitive source dependencies and bundled/generated assets, including documentation themes/plugins, JavaScript packages, fonts, logos, screenshots, examples, and vendored files; record license, source, redistribution obligations, and compatibility status.
4. Record the owner's explicit statement that he solely owns Hero repository content and authorizes preparation for an Apache-2.0 grant. Treat that as the ownership authority; do not perform a contributor-by-contributor consent investigation.
5. Produce the exact Apache-2.0 grant preconditions, any NOTICE/attribution work, and a fail-closed third-party blocker list consumed by repository readiness and the final license gate.

## Acceptance Criteria

- **AC-1:** THE SYSTEM SHALL identify every repository and material category included in or excluded from the proposed Apache-2.0 grant.
- **AC-2:** WHEN a dependency or bundled asset is redistributed THE SYSTEM SHALL record its license, source, compatibility conclusion, and notice obligations.
- **AC-3:** THE SYSTEM SHALL record the sole owner's authorization to prepare this repository for an Apache-2.0 grant without reopening contributor-rights analysis.
- **AC-4:** IF a third-party asset's provenance or a dependency's license compatibility cannot be demonstrated THEN THE SYSTEM SHALL mark the row unresolved and block the Apache grant gate.
- **AC-5:** THE SYSTEM SHALL verify Hero consumes a Sprout module release containing the MIT license and SHALL state that `hero-code` and `hero-cloud` remain proprietary.

## Boundaries

- No contributor-by-contributor rights review; the owner's explicit authorization is authoritative for Hero-owned content.
- No `LICENSE` or visibility mutation without the later explicit approval gates.
- No licensing, publication, or code transfer involving `hero-code` or `hero-cloud`.
- No mutation or relicensing of the Sprout repository; it remains a separate MIT project.

## Validation

- Reconcile manifest inventories with tracked files and built distributions.
- Verify owner authorization is present in the packet and review third-party material without turning repository history into an ownership investigation.
- Require zero unexplained redistributed assets and zero unresolved third-party licensing rows before marking the grant packet clear.

## Completion Ledger

Delivered the licensing packet needed for the v0.34 public-open-source preparation: a precise repository/product boundary, recorded owner authorization, package- and asset-level third-party review, a licensed exact Sprout dependency, corrected release-manager metadata, and fail-closed work required before the Apache grant and public release.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Identify every included and excluded repository/material category | DONE | `repository-boundary.md` covers CLI source, product content, tooling, docs/site, planning and generated content, the embedded model, user-local state, dependencies, Sprout, and the proprietary Hero Code/Cloud repositories. |
| 2 | Record license, source, compatibility, and notice obligations for redistributed material | DONE | `third-party-inventory.md` separates release-binary, model, Go test/tooling, hosted-doc build, generated site, and first-party visual-asset treatment. |
| 3 | Record sole-owner authorization without contributor-rights reopening | DONE | `repository-boundary.md` records the initiative authorization as the authority for Hero-owned content and explicitly avoids a contributor-by-contributor investigation. |
| 4 | Fail closed on unresolved provenance or incompatibility | DONE | `grant-readiness.md` blocks on Unknown/incompatible/unfulfilled-reciprocal licenses, missing notices, unbounded site dependencies, and blurred product boundaries. The current generated docs output is explicitly blocked on unused MPL-1.1/LGPL-3.0 bundles. |
| 5 | Consume a Sprout artifact containing MIT; keep Code/Cloud proprietary | DONE | Hero now references immutable version `v0.1.1-0.20260822024445-cd3f0c4a2208`; its module ZIP contains `LICENSE`, and the packet repeatedly excludes `hero-code` and `hero-cloud` as proprietary. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Write the repository-boundary report | DONE | Added `repository-boundary.md` with grant inclusions, exclusions, and the public-language rule. |
| 2 | Verify exact Sprout licensing | DONE | Replaced the pre-license `v0.1.0` module with the licensed commit pseudo-version; non-license module content is byte-identical to `v0.1.0`. |
| 3 | Inventory dependencies and assets | DONE | Added `third-party-inventory.md`; package closure was derived across all release targets and the full test graph, docs dependencies were resolved, generated reciprocal bundles were identified, and tracked plus generated binary/search/icon/font/visual/vendored assets were reconciled. Added pinned model source/license records. |
| 4 | Record owner authority | DONE | Owner preparation authorization is stated without implying that the later grant/publication gates have run. |
| 5 | Produce grant and notice preconditions | DONE | Added `grant-readiness.md`; also corrected Homebrew/Scoop license metadata from MIT to Apache-2.0 in `.goreleaser.yaml`. |

### Exercise-the-feature check

- [x] The exact Sprout module archive was inspected for its MIT license; `go mod verify`, mock-tracker tests, the full `go test ./...` suite, GoReleaser configuration validation, YAML parsing, cross-target package closure, package-level `go-licenses` reporting, hosted-doc dependency resolution, and tracked/generated asset scans all passed or produced the documented blocker. A clean export from the pinned upstream model revision reproduced every embedded-model hash byte-for-byte.

### Excellence Bar self-check

- [x] Yes — the packet is product-boundary-specific, distinguishes build inputs from redistributed artifacts, records the embedded model that a source-only scan would miss, and turns every remaining notice/deploy concern into an explicit fail-closed release precondition.
