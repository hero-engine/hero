---
title: "Hero Public Visibility and v0.34 Launch Gate"
slug: hero-public-visibility-launch-gate
type: feature
status: delivering
domain: engineering
size: large
priority: critical
horizon: now
created: 2026-08-21
tags: [github, visibility, launch, v0-34, approval-gate]
parent: hero-marketing
depends-on: [hero-apache-license-grant-gate, hero-v034-release-prep]
delivery_method: manual
---

# Hero Public Visibility and v0.34 Launch Gate

## Goal

After a second explicit owner approval, change only the `hero` repository to public visibility, publish the approved `v0.34.0` release, and verify the complete anonymous user journey across source, install, docs, landing, issues, support, and artifacts.

## Kickoff

This is the final human-controlled exposure gate; green dependencies do not authorize a visibility change.

**Status:** delivering — approval, revision parity, artifacts, DNS, docs, and landing are evidenced, but the public-exposure gate still finds proprietary Cloud source and session databases in reachable Git history.

**Pick up at:** obtain owner authorization to expand scope for the recoverable, bounded history rewrite described by the readiness audit; the current initiative excludes that rewrite, so visibility and `v0.34.0` publication remain halted.

→ `.hero/planning/initiatives/hero-marketing/hero-public-visibility-launch-gate/spec.md`

**Files:** `scripts/public-readiness-scan.sh`, `.hero/specs/hero-public-repo-readiness/exposure-audit.md`, `.build/release-candidate/v0.34.0/provenance.json`, `docs/releases/v0.34.0-launch-checklist.md`, this spec
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

## Launch Gate Evidence

Evidence collected on 2026-08-23/24 immediately before the intended public mutation:

- The owner authorization passed to this delivery names only `hero-engine/hero`, public visibility, publication of `v0.34.0`, and source revision `89b0b021c85cc48e2976fa1ef08cefe062da1fa0`; it explicitly keeps Hero Code and Hero Cloud private and Sprout separate under MIT. The passed authorization does not enumerate the approved artifact hashes or exact GitHub host-setting changes.
- Both declared dependencies are completed. The Apache gate records the canonical Apache-2.0 license and notices; the release-preparation gate records a reproducible unpublished five-target candidate.
- Local `HEAD` is the approved revision with tree `0956005a818b81694e68347f6cbedb61396183a7`. Candidate provenance names the same revision/tree, `v0.34.0`, five target archives, SBOM, license, notices, and exact SHA-256 hashes. `shasum -a 256 -c checksums.txt` passed for every candidate output.
- Cloudflare and Google DNS-over-HTTPS both resolve `heroengine.ai` to the proxied Cloudflare addresses. Requests forced through those authoritative answers returned HTTPS 200 for the landing page and docs. The landing metadata and docs `/revision.json` both identify the approved revision; docs identify `v0.34.0` as the current release target.
- The anonymous GitHub API returned 404 for `hero-engine/hero`, and no local `v0.34.0` tag exists. The repository and release therefore remain unavailable anonymously, as required before the gate opens.
- The current-tree exposure scan passes. `git rev-list --objects --all` still reports 55 forbidden reachable-history paths: 52 under `cloud/**` or `cmd/hero-cloud/**`, plus three `.hero/sessions/*/refs.db` files. These are the unresolved PR-1/PR-2 blockers recorded by the completed public-readiness audit, so the public-exposure gate is not green and AC-2 requires a halt. The current initiative explicitly excludes a history rewrite; clearing this blocker requires owner-authorized scope expansion for a recoverable, bounded rewrite.
- Validation passed: `scripts/test-public-readiness-scan.sh`; `go test ./...`; `go run ./cmd/hero docs check --public --invocations`; 28 docs tests; 8 landing tests; 14 release-candidate tests; and candidate checksum verification.
- Anonymous GitHub API checks returned 404 for `hero-engine/hero-code` and `hero-engine/hero-cloud`. The Sprout repository redirect resolves to public `astroville/sprout` with an MIT license.

## Completion Ledger

Delivery is deliberately halted before external mutation. The local, DNS, deployment, licensing, and candidate evidence is green, but the repository's reachable history still contains material the readiness contract classifies as a public-exposure blocker.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Leave visibility and release unchanged without approval | DONE | Before the passed approval, and still at this preflight, anonymous `GET /repos/hero-engine/hero` returns 404 and `v0.34.0` is absent. DNS and site deployment were prepared after approval, but no repository visibility, tag, release, or GitHub host-setting mutation occurred. |
| 2 | Halt if any launch prerequisite is not green | BLOCKED | The gate halted correctly: the current-tree scan passes, but 55 forbidden reachable-history paths remain (52 proprietary Cloud paths and 3 session databases), matching public-readiness blockers PR-1/PR-2. The initiative excludes history rewriting, so the owner must authorize scope expansion before these paths can become unreachable and the scan can be rerun. Do not add `[signed-off]`. |
| 3 | Expose only Hero and publish only approved v0.34 revision/artifacts | BLOCKED | The approved revision and candidate hashes are identified, but AC-2 prevents visibility/tag/release mutation. Root owns the external action after a clean rewritten revision is approved. |
| 4 | Complete anonymous source/install/docs/support/artifact journeys | BLOCKED | DNS, landing, and docs return HTTPS 200 at the approved revision, but the Hero repository and `v0.34.0` release remain anonymous 404 and cannot be exercised until AC-2 clears and publication occurs. |
| 5 | Do not change Sprout, Hero Code, or Hero Cloud | DONE | The authorization and execution boundary exclude all three. Anonymous APIs leave Hero Code/Cloud unavailable; Sprout remains a separate public MIT repository. No mutation was dispatched for any sibling repository. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Re-run exposure, licensing, candidate, DNS, deployment, link, support, and security gates | BLOCKED | Current-tree, dependency, candidate, checksum, test, DNS, landing, and docs evidence passed. The history exposure check found the 55 PR-1/PR-2 paths and correctly stopped the launch. |
| 2 | Record exact approval and mutation boundary | BLOCKED | Repository, visibility, version, revision, and sibling exclusions are explicit. Root must supply an approval record enumerating the exact artifact hashes and exact GitHub host-setting changes before mutation. |
| 3 | Change visibility/settings and publish signed/verified v0.34.0 | BLOCKED | Not attempted because the exposure gate failed; external mutations are owned by root after the history blocker is cleared. |
| 4 | Enable and verify public destinations and anonymous journeys | BLOCKED | Landing/docs/DNS are live at the approved revision, but source, clone, issues/security, release download, integrity, and package-channel journeys remain gated by private repository/release state. |
| 5 | Capture immutable evidence and execute rollback on failure | BLOCKED | Preflight evidence and the existing rollback plan are recorded, but immutable post-launch evidence cannot exist until a compliant launch occurs. |

### Exercise-the-feature check

- [ ] Cannot be exercised end-to-end because the reachable-history exposure gate failed before the authorized visibility and release mutations; the remediation is outside the initiative's current scope and requires owner-authorized expansion.

### Excellence Bar self-check

- [ ] No — shipping while proprietary Cloud source and machine-local session databases remain reachable would violate the spec's fail-closed public-exposure contract.
