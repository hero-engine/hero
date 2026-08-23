---
title: "Hero Apache-2.0 License Grant Gate"
slug: hero-apache-license-grant-gate
type: feature
status: planning
domain: engineering
size: small
priority: critical
horizon: now
created: 2026-08-21
tags: [apache-2, licensing, approval-gate, legal]
parent: hero-marketing
depends-on: [hero-v034-release-prep, hero-licensing-boundary-and-provenance, hero-public-repo-readiness]
---

# Hero Apache-2.0 License Grant Gate

## Goal

After owner authorization and all third-party inventory blockers are clear, obtain final mutation approval, then add the canonical Apache License 2.0 grant and required notices to this `hero` repository only.

## Kickoff

This is a human-controlled mutation gate, not an automatic continuation of release preparation.

**Status:** planning — do not deliver until the user explicitly approves adding Apache-2.0 after reviewing the final grant packet.

**Pick up at:** present the exact included repository boundary, recorded owner authorization, cleared third-party obligations, proposed license files, and resulting public claim for final mutation approval.

→ `.hero/planning/initiatives/hero-marketing/hero-apache-license-grant-gate/spec.md`

**Files/components:** root `LICENSE`, `NOTICE` only when required, package/repository license metadata, public license references, approval evidence
**Skip:** repository visibility, release publication, Sprout mutation, and any license change to `hero-code` or `hero-cloud`.

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
