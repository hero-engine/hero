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

This gate has the owner's explicit authorization and is being delivered only for the approved repository boundary.

**Status:** delivering — apply and verify the approved Apache-2.0 grant, then stop before visibility or publication.

**Pick up at:** finish the licensed candidate validation, cold-audit the evidence, and close through `hero spec verify`.

→ `.hero/planning/initiatives/hero-marketing/hero-apache-license-grant-gate/spec.md`

**Files/components:** root `LICENSE`, `NOTICE` only when required, package/repository license metadata, public license references, approval evidence
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
