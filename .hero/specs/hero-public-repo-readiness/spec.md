---
title: "Hero Public Repository Readiness"
slug: hero-public-repo-readiness
type: feature
status: completed
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
delivery_method: manual
completed_at: 2026-08-23T20:05:46Z
---

# Hero Public Repository Readiness

## Goal

Make the repository safe and understandable for anonymous visitors by resolving public-exposure risks and adding the minimum contribution, security, conduct, issue, and support surfaces, while leaving both licensing and repository visibility unchanged for their explicit final gates.

## Kickoff

Prepares the private repository for safe public exposure without performing the exposure.

**Status:** in-review — public policies/templates and redacted exposure evidence are complete; reachable proprietary Cloud source and session databases correctly block visibility.

**Pick up at:** cold-audit this delivery against the fail-closed exposure report; do not rewrite history or change visibility inside this spec.

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

## Completion Ledger

Prepared the public repository policy surfaces, a redacting current/history
scanner with mutation-sensitive reviewed baselines, a host-settings launch
checklist, and a durable fail-closed exposure report. Validation deliberately
returns a blocked readiness decision because proprietary Cloud source and local
session databases remain reachable in Git history.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Report public-exposure risks with disposition evidence | DONE | `exposure-audit.md` records tracked/history secrets review, personal/internal data, proprietary source, unsuitable databases, large/generated assets, and third-party reconciliation using redacted fingerprints only. |
| 2 | Provide contribution, security, conduct, support, issue, and PR surfaces | DONE | Root policy files plus `.github/ISSUE_TEMPLATE/*.yml` and `.github/PULL_REQUEST_TEMPLATE.md` provide bounded routes; YAML and all relative policy links were exercised. |
| 3 | Hide or label dead anonymous source links while private | DONE | Public-facing root/docs/landing/policy surfaces contain no direct `github.com/hero-engine/hero` links; policy text explicitly labels issue/security routes unavailable until their gates land. |
| 4 | Detect Code/Cloud content and block rather than grant it | DONE | Current tree has no Cloud source path; 45 historical source files / 52 object-list path names including 7 tree prefixes under `cloud/**` and `cmd/hero-cloud/**` are recorded as blocker PR-1, distinct from name-only public contracts/specs. |
| 5 | Leave visibility unchanged without explicit approval | DONE | No repository-settings mutation was dispatched; anonymous repository access still returns `404`, and the settings checklist retains an explicit HOLD. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Audit tracked content and reachable history | DONE | `scripts/public-readiness-scan.sh`, its self-test/baseline, and `exposure-audit.md` cover current and reachable objects without printing matched values; blockers have exact remediation ownership. |
| 2 | Add minimum policy and GitHub contribution surfaces | DONE | Added `CONTRIBUTING.md`, `SECURITY.md`, `CODE_OF_CONDUCT.md`, `SUPPORT.md`, two issue forms/config, and a PR template with truthful pre-license/private boundaries. |
| 3 | Repair root navigation and gate private source routes | DONE | `README.md` links every policy surface and states that preparation grants no current redistribution rights; direct private source URLs are absent from public-facing sources. |
| 4 | Prepare host-settings checklist | DONE | `repository-settings-checklist.md` covers identity/homepage, issues/discussions, default branch, protection/checks, vulnerability reporting, secret scanning, dependency alerts, and the explicit visibility hold. |
| 5 | Reconcile owner/licensing packet and fail on proprietary material | DONE | `exposure-audit.md` consumes the completed licensing boundary, preserves Sprout's separate MIT status, excludes Code/Cloud, and blocks on historical Cloud source plus remaining grant/notice gates. |

### Exercise-the-feature check

- [x] `scripts/test-public-readiness-scan.sh` proved current/history detection, output redaction, exact fingerprint baselining, and changed-match reblocking; the authoritative 26,027-row history scan plus exact baseline replay left only the 45 proprietary-path and 3 machine-local blockers; issue-form YAML parsed; policy links resolved; public surfaces had zero direct private-source URLs; anonymous repository/docs requests returned `404`/`200`; `git diff --check` passed; and `go test ./...` passed.

### Excellence Bar self-check

- [x] Yes — the result is operational rather than ceremonial: public routes are honest, known false positives cannot hide mutations, sensitive values never enter the report, and repository exposure fails closed on exact recoverable history evidence without performing a destructive rewrite.
