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

**Status:** in-review — the unpublished `v0.34.0` candidate is reproducible and smoke-tested; only the explicit Apache and public-launch gates remain.

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
- Sprout remains a separately versioned MIT dependency; the candidate must consume an immutable module version whose exact archive contains the MIT license.

## Validation

- Compare candidate artifacts and notes with the v0.33.0 diff and claim registry.
- Verify reproducibility, checksums, SBOM/notices, clean installs, upgrade/rollback instructions, and zero unresolved release-checklist rows.

## Completion Ledger

The v0.34 preparation path now produces a reproducible, unpublished candidate
from one clean revision and stops before every externally visible mutation. The
candidate is evidence for the Apache and visibility gates, not a substitute for
either approval.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Target v0.34.0 from latest v0.33.0 authority | DONE | `source_identity` requires canonical semantic tags, proves `v0.33.0` is the latest release tag, rejects an existing `v0.34.0`, and records the exact source commit/tree. The exercised candidate used `fcc72b4988b479abe733c87ee0bfb6bd686830ff`. |
| 2 | Produce reproducible packages, checksums, SBOM, notices, and provenance | DONE | `scripts/release_candidate.py` independently built all five supported archives twice and compared complete outputs byte-for-byte. It emits verified SHA-256 checksums, a CycloneDX 1.5 SBOM with 19 components, a complete exact-license notice packet, and deterministic provenance; output deletion is confined to a symlink-resolved version directory below `.build/release-candidate`. |
| 3 | Align candidate release claims with the public registry | DONE | `docs/releases/v0.34.0-candidate.md` labels memory and verified delivery shipped while their claimed reinforcing improvement loop remains preview, and preserves the exact Hero/Sprout/Hero Code/Hero Cloud boundary; `test_candidate_notes_preserve_public_maturity_and_product_boundaries` prevents drift. |
| 4 | Complete clean local candidate installs without private source access | DONE | The extracted Darwin ARM64 archive ran its stamped version, `init`, Codex install, `status`, and `check` in a fresh Git repository with isolated HOME/XDG state and asserted installed artifacts. No source checkout or private repository was used by the smoke journey. |
| 5 | Do not tag, upload, or publish without final approval | DONE | Manual dispatch is isolated to the candidate job, uploads no binary, and writes only provenance/checksums to the job summary. The publishing job is tag-only and fails closed on missing final `LICENSE` or release notices. No tag, release upload, visibility mutation, or deployment occurred. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Derive change inventory and candidate release notes | DONE | `docs/releases/v0.34.0-candidate.md` reconciles the user-visible product and public-readiness changes since `v0.33.0` into memory-first public experience, dual-mode Engineering/PM/QA composition, focused packs, trust/onboarding, maturity boundaries, and upgrade guidance; release-preparation mechanics are intentionally not marketed as product features. |
| 2 | Reconcile target metadata, channels, and public release references | DONE | The candidate builder stamps `v0.34.0`; artifact names match the target; public release history intentionally remains at published `v0.33.0`; `.github/workflows/release.yml` cleanly separates manual candidate verification from tag-only publication and production license gates. |
| 3 | Build supported artifacts reproducibly with release evidence | DONE | Darwin amd64/arm64, Linux amd64/arm64, and Windows amd64 archives were generated twice with normalized timestamps/ownership/paths and identical bytes; checksums, SBOM, notices, and provenance are preserved locally under `.build/release-candidate/v0.34.0`. GoReleaser is preconfigured to add the gated final root license and notices to production archives. |
| 4 | Exercise installs and document destination/rollback behavior | DONE | The local archive smoke passed without repository access; current docs/landing source and destination contracts passed `hero docs check --public --invocations`; candidate notes distinguish binary replacement from `hero upgrade`; the checklist defines channel, tag, visibility, and site rollback. |
| 5 | Produce the exact approval/launch checklist without publication | DONE | `docs/releases/v0.34.0-launch-checklist.md` names every artifact, marks all preparation rows PASS, and marks all ten human-controlled mutations GATED with exact actions. No checklist row is unknown or unresolved. |

### Exercise-the-feature check

- [x] From clean detached revision `fcc72b4988b479abe733c87ee0bfb6bd686830ff`, the candidate builder produced and independently reproduced five archives, verified every checksum, generated the SBOM/notices/provenance packet, and completed the extracted Darwin ARM64 clean-install journey. The evidence is available locally at `.build/release-candidate/v0.34.0`; it is ignored, unpublished, and stamped `pending-apache-2.0` so it cannot be mistaken for the final release.

### Excellence Bar self-check

- [x] Yes — the preparation is deterministic, exact-revisioned, license-aware, rollback-aware, mutation-safe, and exercised from the artifact a user would install. It reuses the existing tag-triggered GoReleaser path for publication while making manual candidate runs incapable of publishing.
