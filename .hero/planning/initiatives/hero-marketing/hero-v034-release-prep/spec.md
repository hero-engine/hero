---
title: "Hero v0.34 Release Preparation"
slug: hero-v034-release-prep
type: feature
status: delivering
domain: engineering
size: medium
priority: high
horizon: now
created: 2026-08-21
tags: [release, v0-34, artifacts, sbom, changelog]
parent: hero-marketing
depends-on: [hero-public-docs-drift-guard]
supersedes: [hero-distribution]
delivery_method: manual
---

# Hero v0.34 Release Preparation

## Goal

Prepare a reproducible `v0.34.0` release candidate from the current `v0.33.0` baseline, including accurate notes, artifacts, checksums, SBOM/notices, install verification, and a launch checklist, without tagging or publishing the release.

## Kickoff

Converts the repaired public product into a release candidate the final approval gates can inspect and publish.

**Status:** planning — `v0.33.0` is the latest tag and public-readiness work is not yet complete.

**Pick up at:** diff `v0.33.0` to the candidate revision, classify user-visible changes, and reconcile every release artifact and public version reference.

→ `.hero/planning/initiatives/hero-marketing/hero-v034-release-prep/spec.md`

**Files/components:** changelog/release notes, release automation, version metadata, package formulas/manifests, artifact build, checksums, SBOM/notice bundle, install verification, launch and rollback checklists
**Skip:** creating `v0.34.0`, publishing a release, changing visibility, or adding the Apache license.

## Changes

1. Derive the `v0.33.0..candidate` change inventory and write user-facing v0.34 release notes that match shipped behavior and public maturity labels.
2. Reconcile version metadata, generated references, install channels, package manifests, and docs/landing release references against the `v0.34.0` target.
3. Build release artifacts reproducibly for supported targets; generate checksums, SBOM, and required third-party notices from the cleared licensing inventory.
4. Exercise clean anonymous-style installs and smoke journeys from local candidate artifacts, including docs/landing destinations and upgrade/rollback instructions.
5. Produce the exact approval/launch checklist and artifact identifiers consumed by the Apache and visibility gates; do not tag or upload them.

## Acceptance Criteria

- **AC-1:** THE SYSTEM SHALL target `v0.34.0` from the latest `v0.33.0` release authority.
- **AC-2:** WHEN candidate artifacts are built THE SYSTEM SHALL produce reproducible binaries/packages, checksums, SBOM, third-party notices, and provenance tied to one revision.
- **AC-3:** WHEN release notes describe a capability THE SYSTEM SHALL align it with the public claim registry and v0.34 evidence.
- **AC-4:** WHEN candidate install paths are exercised THE SYSTEM SHALL complete supported clean installs and core smoke journeys without relying on private repository access.
- **AC-5:** IF final launch approval is absent THEN THE SYSTEM SHALL NOT create a tag, upload artifacts, or publish a v0.34 release.

## Boundaries

- No release publication, visibility change, Apache grant, or launch announcement.
- No inclusion or licensing of proprietary `hero-code` or `hero-cloud` artifacts.
- Sprout remains a separately versioned MIT dependency; the candidate must consume a module tag that contains the MIT license.

## Validation

- Compare candidate artifacts and notes with the v0.33.0 diff and claim registry.
- Verify reproducibility, checksums, SBOM/notices, clean installs, upgrade/rollback instructions, and zero unresolved release-checklist rows.
