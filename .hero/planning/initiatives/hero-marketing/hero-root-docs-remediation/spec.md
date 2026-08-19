---
title: "Hero Root Documentation Truth Repair"
slug: hero-root-docs-remediation
type: feature
status: planning
domain: engineering
size: medium
priority: high
horizon: now
created: 2026-08-04
tags: [documentation, onboarding, configuration, truth]
parent: hero-marketing
depends-on: [hero-public-truth-baseline, hero-positioning]
relations:
  - target: hero-public-docs-drift-guard
    kind: conflicts-with
---

# Hero Root Documentation Truth Repair

## Goal

Make repository-level onboarding and reference documentation executable, current, and consistent with the approved positioning and claim baseline.

## Kickoff

Repairs the highest-risk public instructions before broader hosted-doc and landing publication.

**Status:** planning — blocked on the truth baseline and positioning authority.

**Pick up at:** apply the P0 correction packet to root docs, then exercise every changed quickstart in a clean workspace.

→ `.hero/planning/initiatives/hero-marketing/hero-root-docs-remediation/spec.md`

**Files:** `README.md`, `GETTING-STARTED.md`, `MCP-SETUP.md`, `CROSS-REPO-PEERING.md`, `TEAM-SERVER.md`, root metadata and generated public guidance
**Skip:** `web/docs/src/` and landing publication.

## Changes

1. Replace nonexistent satellite/list/add and invalid repair examples with the canonical one-root satellite workflow; explicitly prevent nested `hero init` guidance.
2. Make `hero spec verify` the normal closing path, correct the Go prerequisite, remove the dead `hero verify-install` command, and align all root quickstarts with current help.
3. Correct configuration, harness-native workflow, peering, team, repository-tree, install, and capability-maturity statements from the baseline.
4. Remove hand-maintained inventory counts from narrative pages or source them from generated reference data.
5. Exercise changed commands and configurations in disposable workspaces and record evidence.

## Acceptance Criteria

- **AC-1:** WHEN a user follows any root quickstart THE SYSTEM SHALL complete it against the current CLI without undocumented positional behavior.
- **AC-2:** THE SYSTEM SHALL document one root `.hero` corpus with thin satellite harness trees and SHALL NOT prescribe nested project initialization.
- **AC-3:** THE SYSTEM SHALL present `hero spec verify` as the normal evidence-backed closing gate and accurately bound any manual completion path.
- **AC-4:** WHEN root documentation includes configuration THE SYSTEM SHALL load its examples through the production decoder.
- **AC-5:** WHEN Hero workflow surfaces differ by harness THE SYSTEM SHALL use harness-native terminology rather than claiming universal slash commands.
- **AC-6:** THE SYSTEM SHALL resolve every root-doc claim assigned to this child in the claim registry with reproducible evidence.

## Boundaries

- No hosted-doc page edits, landing design, release campaign, or product compatibility shim for stale docs.
- No mutable counts in narrative copy unless they are generated from the canonical registry.

## Validation

- Execute root quickstarts in clean temporary workspaces and decode every complete JSON example.
- Run documentation invocation checks, link checks, `git diff --check`, Hero lint/score, and index refresh.
