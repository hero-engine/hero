---
title: "Hero Public Documentation Drift Guard"
slug: hero-public-docs-drift-guard
type: feature
status: completed
domain: engineering
size: medium
priority: medium
horizon: now
created: 2026-08-04
tags: [documentation, ci, validation, deployment]
parent: hero-marketing
depends-on: [hero-root-docs-remediation, hero-hosted-docs-remediation, hero-landing-message-refresh, hero-public-repo-readiness]
relations:
  - target: hero-public-truth-baseline
    kind: conflicts-with
  - target: hero-root-docs-remediation
    kind: conflicts-with
  - target: hero-hosted-docs-remediation
    kind: conflicts-with
  - target: generated-command-refs-validated
    kind: related
delivery_method: manual
completed_at: 2026-08-23T21:31:07Z
---

# Hero Public Documentation Drift Guard

## Goal

Turn the repaired public truth into derived, executable, deployment-aware validation so structurally green builds cannot hide false product content again.

## Kickoff

Adds one executable public-truth gate across root docs, hosted docs, the landing page, and approved production deploys.

**Status:** in-review — the shared gate, CI wiring, production parity crawl, and mutation coverage are implemented and green locally.

**Pick up at:** cold-audit the implementation and Completion Ledger, then run `hero spec verify`; do not deploy either public surface.

→ `.hero/planning/initiatives/hero-marketing/hero-public-docs-drift-guard/spec.md`

**Files:** `internal/cli/docs_check.go`, `internal/cli/public_docs_check.go`, `internal/cli/public_docs_check_test.go`, `internal/serve/mcp_tools_def.go`, `.github/workflows/{docs,landing}.yml`
**Skip:** rewriting product copy or duplicating `generated-command-refs-validated` command-reference scope.

## Changes

1. Derive command/agent/skill inventories from install manifests and total/client-filtered MCP inventories from runtime registries.
2. Validate nested command semantics, required arguments, flags, harness workflow forms, configuration examples through the production decoder, and clean-workspace quickstarts.
3. Scan root docs, all `web/docs/src` content/navigation/releases, landing HTML/meta/OG/output, and generated install guidance for high-risk contradictions.
4. Build and link-check public surfaces; validate anchors, redirects, accessibility, assets, release freshness, and real-versus-illustrative output labels.
5. Tie deployments to source revision/time and run a post-deploy production crawl that fails the initiative gate on unresolved P0/P1 claims or stale public content.
6. Assert compatibility-bounded docs dependencies, resolvable `heroengine.ai` DNS, anonymous-source-link gating, and the exact Hero/Sprout/proprietary repository boundary.

Extend the existing `internal/cli/docs_check.go` surface instead of creating a parallel truth checker. Decode configuration fixtures through `internal/config`, derive the full MCP inventory from the same runtime definitions asserted in `internal/serve/mcp_test.go`, and add source-revision parity to the existing docs and landing workflows.

## Acceptance Criteria

- **AC-1:** WHEN public reference counts are emitted THE SYSTEM SHALL derive them from canonical install/runtime registries and distinguish total from filtered MCP surfaces.
- **AC-2:** WHEN documentation contains an executable command, quickstart, or full configuration example THE SYSTEM SHALL validate its semantics in an isolated environment or production decoder.
- **AC-3:** WHEN source docs or landing content changes THE SYSTEM SHALL scan every public surface for high-risk contradictions in versions, maturity, harnesses, install methods, closing gates, peering, and repository architecture.
- **AC-4:** WHEN public content deploys THE SYSTEM SHALL expose and verify a source revision and deployment timestamp.
- **AC-5:** WHEN the initiative closes THE SYSTEM SHALL crawl production landing/docs URLs and report zero unresolved P0/P1 claims with reviewed-source parity.
- **AC-6:** THE SYSTEM SHALL preserve the distinct scope of `generated-command-refs-validated` and integrate or extend it without duplicating contradictory checkers.
- **AC-7:** WHEN public-readiness validation runs THE SYSTEM SHALL fail on incompatible docs dependency drift, non-resolving production DNS, dead anonymous source links, or a claim that `hero-code`/`hero-cloud` is open source.

## Boundaries

- No hardcoded duplicate registry, version, or count authority.
- No claim that a build/link pass proves semantic truth without the claim and executable checks.
- No silent deploy failure or source-only success reported as public correction.
- No license or visibility mutation; this child validates the boundary but does not cross either approval gate.

## Validation

- Mutation tests for stale counts/versions, invalid nested commands, undecodable config, maturity contradictions, fictional output labels, and deployment revision mismatch.
- Strict docs build, landing validation, production URL crawl, Hero lint/score, index refresh, and `git diff --check`.

## Completion Ledger

The delivery extends the existing `hero docs check` command with a separate public-contract mode and a gated production-parity mode. It reuses the canonical install enumerator, runtime MCP registry and filter, production config decoder, existing invocation resolver, docs/landing builders, and their revision metadata rather than introducing parallel authorities. No site was deployed and no network crawl was run against production.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Derive public inventory counts from canonical registries | DONE | `internal/cli/docs_check.go` and `canonicalMCPInventory`; `TestCanonicalMCPInventoryUsesRuntimeRegistryAndConfiguredProfiles` proves total and filtered counts come from `serve.MCPToolDefinitions` and `ToolFilter`. |
| 2 | Validate commands, quickstarts, and configuration examples | DONE | `--invocations` retains broad Cobra resolution; fenced executable commands additionally validate positional arguments and flag values; three marked install quickstarts run against fresh Git workspaces; marked public JSON uses `config.Load`. This caught and corrected the stale code-host broker example. |
| 3 | Scan every public surface for high-risk contradictions | DONE | `publicNarrativeSurfaces` walks every text source under hosted docs and landing, including navigation, releases, attributions, metadata/assets, plus root docs and rendered install guidance. Surface-discovery and mutation tests cover the complete set. |
| 4 | Expose and verify source revision and deployment timestamp | DONE | Existing docs/landing build metadata is now checked by `revisionTemplateIssues`; `productionPublicIssues` requires exact revision parity and RFC3339 `generated_at`; revision-template and production mutation tests pass. |
| 5 | Crawl production landing/docs at initiative close | DONE | `--production docs|landing|all --expected-revision <sha>` checks canonical roots, key docs paths, revision endpoints, availability, and exact source parity; both deploy workflows invoke their surface crawl only after the approved deploy step. |
| 6 | Preserve generated-command reference scope | DONE | The existing `--invocations` checker is unchanged and composed with `--public`; `go run ./cmd/hero docs check --public --invocations` passed with all command references resolving. |
| 7 | Fail on dependency drift, DNS/unavailability, source links, or open-source boundary drift | DONE | Exact MkDocs major pins, unavailable production surfaces, private source links, required Hero Code/Cloud proprietary wording, required separate Sprout MIT wording, forbidden Sprout/Apache coupling, and revision mismatch all have biting checks and mutation coverage. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Derive install and MCP inventories | DONE | Existing `canonicalInstallCounts` remains the install authority; exported `serve.MCPToolDefinitions` plus the configured runtime `ToolFilter` produces total/default/profile MCP counts. |
| 2 | Validate command semantics, config, and quickstarts | DONE | Public config blocks use the production decoder; fenced executable commands enforce arguments and flag values; README, Getting Started, and hosted-doc quickstarts execute `init → install → status/check` in isolated repositories and assert installed artifacts. |
| 3 | Scan public sources for contradictions | DONE | `internal/cli/public_docs_check.go` covers all named source families, every hosted/landing text source, navigation, releases, attributions, metadata/assets, and rendered AGENTS guidance; discovery coverage prevents silent omissions. |
| 4 | Build/link/revision/illustrative checks | DONE | Existing strict docs and landing validators remain authoritative; both Python suites, strict MkDocs build, JavaScript/link checks, landing source/build/artifact checks, and workflow YAML parsing passed. |
| 5 | Tie deployments to revision and crawl after deploy | DONE | Both workflows stamp `${{ github.sha }}`, deploy only behind existing gates, then invoke the matching production crawl against that exact revision. |
| 6 | Enforce dependency, DNS, source-link, and repository boundaries | DONE | `docsDependencyIssues`, production HTTP failures, private-source gating, explicit Hero Code/Cloud proprietary requirements, explicit separate Sprout MIT requirements, and forbidden Sprout grant claims are wired into `--public`/`--production` and tested. |

### Exercise-the-feature check

- [x] User-visible behavior was exercised end-to-end: a freshly built binary ran `hero docs check --public --invocations`, derived 35 agents, 29 commands, 57 skills, and 82 canonical MCP tools, resolved every documented invocation, executed all three marked install quickstarts in clean Git workspaces, and reported no public-contract issues. The production mode was exercised against local HTTP fixtures for both success and fail-closed paths; no live deployment was authorized.

### Excellence Bar self-check

Honest answer to "would a senior engineer who cares about this codebase be proud to ship this?" — yes. The gate composes existing authorities, fails closed at the deploy boundary, carries mutation coverage for the risky claims, and does not turn an optional marketing asset into release engineering.
