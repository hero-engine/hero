---
title: "Hero Public Repository Readiness"
slug: hero-public-repo-readiness
type: feature
status: planning
domain: engineering
size: medium
priority: high
horizon: now
created: 2026-08-21
tags: [public-repository, security, community, github, readiness]
parent: hero-marketing
depends-on: [hero-landing-message-refresh, hero-licensing-boundary-and-provenance]
relations:
  - target: hero-root-docs-remediation
    kind: conflicts-with
---

# Hero Public Repository Readiness

## Goal

Make the repository safe and understandable for anonymous visitors by resolving public-exposure risks and adding the minimum contribution, security, conduct, issue, and support surfaces, while leaving both licensing and repository visibility unchanged for their explicit final gates.

## Kickoff

Prepares the private repository for safe public exposure without performing the exposure.

**Status:** planning — public source links are currently dead and repository contents/settings have not passed a public-exposure audit.

**Pick up at:** inspect the complete public blast radius, then author the minimum repository policies and templates from verified support and security routes.

→ `.hero/planning/initiatives/hero-marketing/hero-public-repo-readiness/spec.md`

**Files/components:** `README.md`, `CONTRIBUTING.md`, `SECURITY.md`, `CODE_OF_CONDUCT.md`, `SUPPORT.md`, `.github/ISSUE_TEMPLATE/`, pull-request templates, repository metadata/settings checklist, secret/private-reference scan report
**Skip:** broad community programming, CLA infrastructure without a demonstrated need, license creation, and repository visibility changes.

## Changes

1. Audit tracked content and reachable history for credentials, personal data, internal-only URLs, proprietary repository content, unsafe examples, large/generated artifacts, and material unsuitable for public exposure; name every blocker and remediation owner.
2. Add minimum `CONTRIBUTING.md`, `SECURITY.md`, `CODE_OF_CONDUCT.md`, `SUPPORT.md`, issue templates/config, and pull-request guidance with real maintainership and response routes.
3. Update root repository navigation and metadata so install, docs, support, security, contribution, and issue paths are accurate; hide or gate public-source links until anonymous access exists.
4. Prepare a host-settings checklist covering description/homepage, issue/discussion availability, default branch, branch protection, required checks, vulnerability reporting, and secret scanning without changing visibility.
5. Reconcile all findings with the owner-authorization and licensing-inventory packet and fail readiness if proprietary `hero-code`/`hero-cloud` content or unresolved redistribution material is present.

## Acceptance Criteria

- **AC-1:** WHEN the repository is evaluated for public exposure THE SYSTEM SHALL report secrets, personal data, proprietary content, internal references, and unsuitable tracked/history material with disposition evidence.
- **AC-2:** THE SYSTEM SHALL provide current contribution, security, conduct, support, issue, and pull-request surfaces with valid destinations.
- **AC-3:** IF an anonymous source link would be dead while the repository is private THEN THE SYSTEM SHALL hide it or label it unavailable until the visibility gate succeeds.
- **AC-4:** IF content from `hero-code` or `hero-cloud` is detected THEN THE SYSTEM SHALL block public readiness and SHALL NOT treat it as Apache-licensed Hero content.
- **AC-5:** IF explicit visibility approval is absent THEN THE SYSTEM SHALL leave repository visibility unchanged.

## Boundaries

- No `LICENSE`, visibility flip, tag/release publication, or launch announcement.
- No licensing or publication of `hero-code` or `hero-cloud`; both remain proprietary.
- No mutation of Sprout or claim that this repository's eventual license grants rights to Sprout.
- No broad forum, ambassador, content-calendar, or growth-community program.

## Validation

- Run secret, private-reference, third-party-asset, generated-artifact, and anonymous-link scans over the public blast radius.
- Lint policy/template configuration and exercise every support/security/contribution destination.
- Confirm the repository remains private after readiness validation.
