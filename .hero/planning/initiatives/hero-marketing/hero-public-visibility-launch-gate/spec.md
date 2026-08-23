---
title: "Hero Public Visibility and v0.34 Launch Gate"
slug: hero-public-visibility-launch-gate
type: feature
status: planning
domain: engineering
size: medium
priority: critical
horizon: now
created: 2026-08-21
tags: [github, visibility, launch, v0-34, approval-gate]
parent: hero-marketing
depends-on: [hero-apache-license-grant-gate, hero-v034-release-prep]
---

# Hero Public Visibility and v0.34 Launch Gate

## Goal

After a second explicit owner approval, change only the `hero` repository to public visibility, publish the approved `v0.34.0` release, and verify the complete anonymous user journey across source, install, docs, landing, issues, support, and artifacts.

## Kickoff

This is the final human-controlled exposure gate; green dependencies do not authorize a visibility change.

**Status:** planning — the repository remains private, public source links are dead, `heroengine.ai` does not resolve, and no visibility approval has been given.

**Pick up at:** present the final anonymous-exposure scan, Apache grant, DNS/deployment state, artifact identifiers, host-setting changes, and rollback plan for explicit approval.

→ `.hero/planning/initiatives/hero-marketing/hero-public-visibility-launch-gate/spec.md`

**Files/components:** repository visibility/settings, `v0.34.0` tag/release, release artifacts/checksums, public source links, docs/landing/DNS, anonymous clone/install checks, launch evidence
**Skip:** broad campaign execution and any visibility/license change to Sprout, `hero-code`, or `hero-cloud`.

## Changes

1. Re-run the public-exposure, owner-authorization/licensing-inventory, release-candidate, DNS, deployed-revision, source-link, support, and security gates immediately before approval.
2. Record explicit approval naming the repository, public visibility, `v0.34.0` publication, approved revision/artifacts, and host-setting changes; halt without mutation if any element is absent.
3. Change this repository's visibility to public, apply the approved host settings, create the signed/verified `v0.34.0` tag and release, and upload only the approved artifacts/checksums/SBOM/notices.
4. Enable public source destinations and verify anonymous clone, build/install, upgrade, docs, landing, DNS, issue templates, security/support routes, release download, checksums, and license visibility.
5. Capture immutable launch evidence and execute the rollback/incident path if any high-severity anonymous journey fails.

## Acceptance Criteria

- **AC-1:** IF explicit owner approval for public visibility and v0.34 publication is absent THEN THE SYSTEM SHALL leave visibility and release state unchanged.
- **AC-2:** IF DNS, deployment revision, Apache license, third-party licensing inventory, exposure scan, or release-candidate validation is not green THEN THE SYSTEM SHALL halt before public exposure.
- **AC-3:** WHEN approval is recorded THE SYSTEM SHALL expose only this `hero` repository and publish only the approved `v0.34.0` revision and artifacts.
- **AC-4:** WHEN the repository becomes public THE SYSTEM SHALL complete anonymous clone, install, docs, landing, source, issue, support, security, and artifact-integrity journeys.
- **AC-5:** THE SYSTEM SHALL NOT change the visibility or license of Sprout, `hero-code`, or `hero-cloud`.

## Boundaries

- No inferred approval from prior license approval; visibility/publication requires its own explicit authorization.
- No broad social, community, pricing, or launch campaign.
- No mutation or publication of Sprout, `hero-code`, or `hero-cloud`.

## Validation

- Verify approval evidence before mutation and record exact external changes afterward.
- Exercise all anonymous journeys from a credential-free environment and verify `heroengine.ai` resolves to the approved deployment revision.
- Confirm `v0.34.0` artifacts, checksums, SBOM/notices, release notes, and license are mutually consistent.
