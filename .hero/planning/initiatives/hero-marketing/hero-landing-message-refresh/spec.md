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
depends-on: [hero-root-docs-remediation, hero-hosted-docs-remediation, hero-continuity-proof-demo]
relations:
  - target: hero-continuity-proof-demo
    kind: conflicts-with
  - target: hero-hosted-docs-remediation
    kind: conflicts-with
---

# Hero Landing Message and Proof Refresh

## Goal

Replace the stale v0.9/spec-tool-scale landing story with a truthful supervision/continuity/trust message backed by current product evidence and honest availability labels.

## Kickoff

Publishes the repaired story only after root docs, hosted docs, and the continuity proof agree.

**Status:** planning — blocked on documentation remediation and the proof demo.

**Pick up at:** translate the approved positioning house and completed evidence into the smallest coherent landing hierarchy.

→ `.hero/planning/initiatives/hero-marketing/hero-landing-message-refresh/spec.md`

**Files:** `web/landing/site/`, shared `web/` metadata/navigation, real product captures, docs destinations
**Skip:** broad brand redesign, campaign calendar, pricing, or unsupported roadmap promises.

## Changes

1. Lead with reduced supervision and durable project intelligence; make evidence-backed completion and cross-session continuity the proof, with specs as mechanism.
2. Remove hardcoded v0.9 and mutable roster-scale proof; derive any displayed release/build metadata from the public release authority.
3. Replace fictional `hero status` output with real `hero status`/`hero snapshot` evidence and an actual `hero serve` view, or clearly label illustrative data.
4. Separate `available`, `optional`, `preview`, and `planned` capabilities; correct harness-native workflow, peering, cloud/team/outpost, domain, and code-host language.
5. Route each proof pillar to current hosted documentation and publish a source revision/deployment marker.

## Acceptance Criteria

- **AC-1:** THE SYSTEM SHALL present Hero as the operating layer for AI-assisted engineering whose sessions inherit the project and finish against evidence, subject to approved positioning.
- **AC-2:** WHEN a product outcome is claimed THE SYSTEM SHALL link it to real evidence or explicitly label the output illustrative.
- **AC-3:** THE SYSTEM SHALL NOT present hardcoded stale version/count claims, universal slash commands, one cross-repo graph, or unverified roadmap work as shipped.
- **AC-4:** WHEN a capability requires setup or is not generally available THE SYSTEM SHALL show the approved availability label and prerequisites.
- **AC-5:** THE SYSTEM SHALL link every proof pillar to an accurate docs destination and identify the deployed source revision.
- **AC-6:** THE SYSTEM SHALL pass responsive, accessibility, asset, link, and production smoke checks with zero unresolved assigned claims.

## Boundaries

- No pricing, enterprise collateral, competitor pages, launch campaign, or broad visual identity replacement.
- No new product capability added solely to support copy.

## Validation

- Compare every landing assertion with the claim registry and positioning authority.
- Validate real captures/revision markers, links, responsive behavior, accessibility, assets, production deployment, Hero lint/score, and index refresh.
