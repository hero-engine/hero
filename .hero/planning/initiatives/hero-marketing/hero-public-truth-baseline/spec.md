---
title: "Hero Shipped Surface Inventory"
slug: hero-public-truth-baseline
type: feature
status: planning
domain: engineering
size: medium
priority: critical
horizon: now
created: 2026-08-04
tags: [documentation, claims, evidence, onboarding]
parent: hero-marketing
relations:
  - target: hero-public-docs-drift-guard
    kind: conflicts-with
---

# Hero Shipped Surface Inventory

## Goal

Inventory the product as it exists now, classify capability maturity, and package the known P0 onboarding corrections before downstream copy is rewritten.

## Kickoff

Maps public descriptions to shipped behavior and produces executable replacements for stale or unsafe instructions.

**Status:** planning — seeded by `../content-truth-audit.md`; the exhaustive inventory and correction packet remain to be produced.

**Pick up at:** enumerate behavioral, architectural, compatibility, availability, version, count, and prerequisite claims across root docs, hosted docs, landing content, metadata, and generated guidance.

→ `.hero/planning/initiatives/hero-marketing/hero-public-truth-baseline/spec.md`

**Files:** `../content-truth-audit.md`, `README.md`, `GETTING-STARTED.md`, `MCP-SETUP.md`, `web/docs/src/`, `web/landing/site/`
**Skip:** publishing corrections or changing product behavior.

## Changes

1. Materialize an exhaustive claim registry with surface/location, claim family, evidence authority, availability label, prerequisites, owner child, last verification, and resolution state.
2. Resolve a P0 correction packet for satellites/monorepos, install repair syntax, verify/complete semantics, configuration decoding, Go prerequisite, and the dead `hero verify-install` command.
3. Derive current command, agent, skill, and runtime MCP inventories from implementation authorities; recommend removal of mutable counts outside generated reference pages.
4. Classify continuity, audit/verify, Attention/Mail/Focus, `hero serve`, code-host, tracker, peering, headless runtime, PM/Sales packs, and cloud/team/outposts as `shipped`, `optional`, `preview`, or `planned` with prerequisites.
5. Record the settled repository boundary—this `hero` repository as the Apache-2.0 candidate, Sprout separately MIT-licensed, and `hero-code`/`hero-cloud` proprietary—and keep actual Hero licensing/public claims prohibited until their evidence and approvals exist.

## Acceptance Criteria

- **AC-1:** THE SYSTEM SHALL inventory every public behavior, architecture, compatibility, availability, count, version, and prerequisite claim with an owner and source location.
- **AC-2:** WHEN a claim is marked `shipped` THE SYSTEM SHALL attach executable or delivered evidence sufficient to reproduce it.
- **AC-3:** WHEN a capability requires setup or a provider THE SYSTEM SHALL label it `optional` and state the prerequisite.
- **AC-4:** IF evidence is unresolved or planning-only THEN THE SYSTEM SHALL label the claim `preview`, `planned`, or prohibited rather than `shipped`.
- **AC-5:** THE SYSTEM SHALL provide authoritative replacements and executable validation for every P0 row in `content-truth-audit.md`.
- **AC-6:** THE SYSTEM SHALL identify one owner child and resolution state for every registered claim.

## Boundaries

- No public copy edits, landing redesign, or product implementation.
- No inference from a planning spec as evidence of shipped behavior.
- No unconditional licensing, cloud/team/outpost, domain-pack, or code-host readiness claims.
- No license or visibility mutation and no inference that licensing this repository licenses Sprout, `hero-code`, or `hero-cloud`.

## Validation

- Exercise the P0 command/config paths in disposable workspaces against the current binary and production decoder.
- Compare registries with installed manifests and runtime `tools/list`.
- Run `hero spec lint hero-public-truth-baseline`, `hero spec score hero-public-truth-baseline`, and `hero index`.
