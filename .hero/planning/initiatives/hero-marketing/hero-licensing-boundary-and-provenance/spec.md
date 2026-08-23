---
title: "Hero Licensing Boundary and Open-Source Inventory"
slug: hero-licensing-boundary-and-provenance
type: feature
status: planning
domain: engineering
size: small
priority: critical
horizon: now
created: 2026-08-21
tags: [licensing, apache-2, ownership, dependencies, assets]
parent: hero-marketing
depends-on: [hero-public-truth-baseline]
---

# Hero Licensing Boundary and Open-Source Inventory

## Goal

Produce the bounded packet needed to add Apache-2.0 to this `hero` repository: record the sole owner's authorization, define the exact repository boundary, inventory third-party dependencies and bundled assets, and name any notice or compatibility blocker.

## Kickoff

Records owner authorization and clears third-party licensing before any surface says “open source” or any license file is added.

**Status:** planning — the sole owner has authorized Apache-2.0 preparation; third-party dependencies and bundled assets still need review before the grant mutation.

**Pick up at:** record the owner's authorization and inventory tracked source, generated material, vendored content, docs assets, and dependency manifests without adding the license or changing visibility.

→ `.hero/planning/initiatives/hero-marketing/hero-licensing-boundary-and-provenance/spec.md`

**Files/components:** `go.mod`, `go.sum`, documentation dependency manifests, web dependency manifests, tracked media/fonts/logos/screenshots, generated distributions, and a licensing inventory beside this spec
**Skip:** adding `LICENSE`, changing repository visibility, editing Sprout, or inspecting proprietary repositories beyond recording their exclusion.

## Changes

1. Write a repository-boundary report that identifies this `hero` CLI/repository as the proposed Apache-2.0 grant and explicitly excludes `hero-code` and `hero-cloud` as proprietary.
2. Record Sprout as a separately owned, public MIT-licensed dependency at `astroville/sprout`; verify that the exact version consumed by Hero carries the expected license metadata.
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
