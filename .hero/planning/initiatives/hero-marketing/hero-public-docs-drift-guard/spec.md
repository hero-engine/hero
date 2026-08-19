---
title: "Hero Public Documentation Drift Guard"
slug: hero-public-docs-drift-guard
type: feature
status: planning
domain: engineering
size: medium
priority: medium
horizon: now
created: 2026-08-04
tags: [documentation, ci, validation, deployment]
parent: hero-marketing
depends-on: [hero-root-docs-remediation, hero-hosted-docs-remediation, hero-landing-message-refresh]
relations:
  - target: hero-public-truth-baseline
    kind: conflicts-with
  - target: hero-root-docs-remediation
    kind: conflicts-with
  - target: hero-hosted-docs-remediation
    kind: conflicts-with
  - target: generated-command-refs-validated
    kind: related
---

# Hero Public Documentation Drift Guard

## Goal

Turn the repaired public truth into derived, executable, deployment-aware validation so structurally green builds cannot hide false product content again.

## Kickoff

Closes the initiative with source-to-production truth gates rather than another manual audit.

**Status:** planning — intentionally last; depends on corrected root docs, hosted docs, and landing content.

**Pick up at:** convert the claim registry into machine-checkable authorities and classify assertions that still require a human evidence review.

→ `.hero/planning/initiatives/hero-marketing/hero-public-docs-drift-guard/spec.md`

**Files:** `internal/cli/docs_check.go`, `internal/config/config.go`, `internal/serve/mcp_test.go`, `.github/workflows/docs.yml`, `.github/workflows/landing.yml`, generated reference fragments, MkDocs/landing validation, production crawl evidence
**Skip:** rewriting product copy or duplicating `generated-command-refs-validated` command-reference scope.

## Changes

1. Derive command/agent/skill inventories from install manifests and total/client-filtered MCP inventories from runtime registries.
2. Validate nested command semantics, required arguments, flags, harness workflow forms, configuration examples through the production decoder, and clean-workspace quickstarts.
3. Scan root docs, all `web/docs/src` content/navigation/releases, landing HTML/meta/OG/output, and generated install guidance for high-risk contradictions.
4. Build and link-check public surfaces; validate anchors, redirects, accessibility, assets, release freshness, and real-versus-illustrative output labels.
5. Tie deployments to source revision/time and run a post-deploy production crawl that fails the initiative gate on unresolved P0/P1 claims or stale public content.

Extend the existing `internal/cli/docs_check.go` surface instead of creating a parallel truth checker. Decode configuration fixtures through `internal/config`, derive the full MCP inventory from the same runtime definitions asserted in `internal/serve/mcp_test.go`, and add source-revision parity to the existing docs and landing workflows.

## Acceptance Criteria

- **AC-1:** WHEN public reference counts are emitted THE SYSTEM SHALL derive them from canonical install/runtime registries and distinguish total from filtered MCP surfaces.
- **AC-2:** WHEN documentation contains an executable command, quickstart, or full configuration example THE SYSTEM SHALL validate its semantics in an isolated environment or production decoder.
- **AC-3:** WHEN source docs or landing content changes THE SYSTEM SHALL scan every public surface for high-risk contradictions in versions, maturity, harnesses, install methods, closing gates, peering, and repository architecture.
- **AC-4:** WHEN public content deploys THE SYSTEM SHALL expose and verify a source revision and deployment timestamp.
- **AC-5:** WHEN the initiative closes THE SYSTEM SHALL crawl production landing/docs URLs and report zero unresolved P0/P1 claims with reviewed-source parity.
- **AC-6:** THE SYSTEM SHALL preserve the distinct scope of `generated-command-refs-validated` and integrate or extend it without duplicating contradictory checkers.

## Boundaries

- No hardcoded duplicate registry, version, or count authority.
- No claim that a build/link pass proves semantic truth without the claim and executable checks.
- No silent deploy failure or source-only success reported as public correction.

## Validation

- Mutation tests for stale counts/versions, invalid nested commands, undecodable config, maturity contradictions, fictional output labels, and deployment revision mismatch.
- Strict docs build, landing validation, production URL crawl, Hero lint/score, index refresh, and `git diff --check`.
