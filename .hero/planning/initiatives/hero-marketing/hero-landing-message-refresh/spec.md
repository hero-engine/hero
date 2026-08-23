---
title: "Hero Landing Message and Proof Refresh"
slug: hero-landing-message-refresh
type: feature
status: planning
domain: engineering
size: medium
priority: high
horizon: now
created: 2026-08-04
tags: [landing-page, positioning, messaging, proof]
parent: hero-marketing
depends-on: [hero-root-docs-remediation, hero-hosted-docs-remediation]
relations:
  - target: hero-continuity-proof-demo
    kind: conflicts-with
  - target: hero-hosted-docs-remediation
    kind: conflicts-with
supersedes: [hero-landing-page]
---

# Hero Landing Message and Proof Refresh

## Goal

Replace the stale v0.9/spec-tool-scale landing story with a clear two-system message: durable project memory is the headline, while specs and specialized agents form the connected delivery system. Restore a resolvable `heroengine.ai` deployment and keep unproven continuity language bounded until the later proof demo lands.

## Kickoff

Publishes the repaired story only after root and hosted docs agree, while reserving final proof claims for the later continuity demonstration.

**Status:** planning — blocked on documentation remediation; `heroengine.ai` currently does not resolve.

**Pick up at:** translate the approved positioning house and completed evidence into the smallest coherent landing hierarchy, then restore DNS/deployment and gate any still-unproven claim.

→ `.hero/planning/initiatives/hero-marketing/hero-landing-message-refresh/spec.md`

**Files:** `web/landing/site/`, shared `web/` metadata/navigation, real product captures, docs destinations
**Skip:** broad brand redesign, campaign calendar, pricing, or unsupported roadmap promises.

## Changes

1. Lead with durable project memory across sessions, tools, and agents; immediately explain the spec-and-agent delivery system that uses that memory to implement, cold-audit, and verify work.
   - Show the reinforcing loop: memory informs delivery, verified delivery enriches memory, and the next session starts smarter.
   - Make the distinction understandable without prior knowledge and prevent the page from reading like another spec kit.
2. Remove hardcoded v0.9 and mutable roster-scale proof; derive any displayed release/build metadata from the public release authority.
3. Replace fictional `hero status` output with real `hero status`/`hero snapshot` evidence and an actual `hero serve` view, or clearly label illustrative data.
4. Separate `available`, `optional`, `preview`, and `planned` capabilities; correct harness-native workflow, peering, cloud/team/outpost, domain, and code-host language.
5. Route each proof pillar to current hosted documentation and publish a source revision/deployment marker.
6. Restore `heroengine.ai` DNS/hosting, verify the canonical domain and redirects, and hide or gate public source links until anonymous repository access exists.

## Acceptance Criteria

- **AC-1:** THE SYSTEM SHALL present Hero as two connected systems in one: durable project memory as the primary promise and a spec-and-agent delivery system as the execution layer that finishes against evidence.
- **AC-2:** WHEN a product outcome is claimed THE SYSTEM SHALL link it to real evidence or explicitly label the output illustrative.
- **AC-3:** THE SYSTEM SHALL NOT present hardcoded stale version/count claims, universal slash commands, one cross-repo graph, or unverified roadmap work as shipped.
- **AC-4:** WHEN a capability requires setup or is not generally available THE SYSTEM SHALL show the approved availability label and prerequisites.
- **AC-5:** THE SYSTEM SHALL link every proof pillar to an accurate docs destination and identify the deployed source revision.
- **AC-6:** THE SYSTEM SHALL pass responsive, accessibility, asset, link, and production smoke checks with zero unresolved assigned claims.
- **AC-7:** WHEN the landing page is ready for repository audit THE SYSTEM SHALL resolve at `heroengine.ai`, identify its deployed source revision, and avoid dead anonymous source links.
- **AC-8:** WHEN a new reader reviews the primary landing hierarchy THE SYSTEM SHALL understand Hero's memory distinction before encountering detailed workflow mechanics and SHALL NOT reasonably classify Hero as merely another spec kit.

## Boundaries

- No pricing, enterprise collateral, competitor pages, launch campaign, or broad visual identity replacement.
- No new product capability added solely to support copy.
- No license or repository visibility mutation, and no implication that `hero-code` or `hero-cloud` is open source.

## Validation

- Compare every landing assertion with the claim registry and positioning authority.
- Validate real captures/revision markers, links, responsive behavior, accessibility, assets, production deployment, Hero lint/score, and index refresh.
