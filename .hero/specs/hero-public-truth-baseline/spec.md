---
title: "Hero Shipped Surface Inventory"
slug: hero-public-truth-baseline
type: feature
status: completed
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
delivery_method: manual
completed_at: 2026-08-23T16:07:18Z
---

# Hero Shipped Surface Inventory

## Goal

Inventory the product as it exists now, classify capability maturity, and package the known P0 onboarding corrections before downstream copy is rewritten.

## Kickoff

Maps public descriptions to shipped behavior and produces executable replacements for stale or unsafe instructions.

**Status:** completed — 34 claims, all 35 hosted Markdown pages, root guides, landing/exposure surfaces, and every P0 correction passed cold audit and Hero verification.

**Pick up at:** consume the registry and correction packet in `hero-licensing-boundary-and-provenance`, `hero-positioning`, and the root/hosted documentation remediation children.

→ `.hero/specs/hero-public-truth-baseline/spec.md`

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
  verified_by: internal/config/public_example_test.go::TestPublicHeroConfigFixtureLoadsThroughProductionDecoder
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

## Completion Ledger

Delivered a source-of-record for the v0.34 public refresh without changing public copy: 34 claim rows, complete hosted-page ownership, seven executable P0 replacements, derived install/runtime inventory, and a production-decoder configuration fixture. Validation included the full Go test suite, documentation checks, registry/schema checks, surface completeness, and the user-visible CLI help/error paths cited by the correction packet.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Inventory every public claim with owner and source location | DONE | `public-claim-registry.yaml` records 34 behavior, architecture, compatibility, availability, count, version, prerequisite, and licensing claims; `public-surface-inventory.md` maps 35/35 hosted Markdown pages plus root/landing surfaces. |
| 2 | Shipped claims carry executable or delivered evidence | DONE | Every `shipped` registry row names Class A/B evidence; registry validation confirmed required evidence fields on all rows. |
| 3 | Optional capabilities state prerequisites | DONE | Optional domain-pack, tracker, code-host, and peering rows name setup/provider/consent prerequisites in `public-claim-registry.yaml`. |
| 4 | Unresolved or planning-only claims are bounded | DONE | The two-system outcome and Sprout consumed-version boundary are `preview`/`bounded`; external spec providers and Apache status are `planned`/`prohibited`; deployment and DNS are `preview`/`blocked`; headless runtime is `preview`/`bounded`. |
| 5 | Provide executable replacements for every P0 audit row | DONE | `p0-correction-packet.md` covers all seven P0 rows; `internal/config/public_example_test.go` proves the canonical public config fixture loads through `config.Load`. |
| 6 | Every claim has one owner and resolution state | DONE | YAML validation confirmed owner and allowed resolution values for all 34 claim rows. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Materialize exhaustive claim registry | DONE | Added `public-claim-registry.yaml` and `public-surface-inventory.md`, including evidence, availability, prerequisites, surfaces, owner, verification date, and resolution. |
| 2 | Resolve the seven-row P0 correction packet | DONE | Added `p0-correction-packet.md` with affected locations, exact replacements, observed failures, owner, and executable checks. |
| 3 | Derive command, agent, skill, and MCP inventories | DONE | Recorded 29 commands, 35 agents, 57 canonical skills, 86 Codex/Grok installed skills, 82 MCP tools, and seven targets from runtime/install authorities; narrative-count removal is assigned downstream. |
| 4 | Classify capability maturity and prerequisites | DONE | Registry covers continuity, audit/verify, Attention/Mail/Focus, serve, code-host, tracker, peering, headless runtime, domain packs, cloud boundaries, and external provider direction. |
| 5 | Record repository licensing boundary | DONE | Registry records Apache preparation as authorized but not granted, Sprout as separate MIT, and `hero-code`/`hero-cloud` as proprietary and excluded. |

### Exercise-the-feature check

- [x] User-visible behavior was exercised end-to-end: current CLI help and stale invocation paths were run; `hero doctor` derived install counts; `hero docs check` and `hero docs check --invocations` passed; the registry mapped 35/35 hosted Markdown pages; and the decoder-backed public config test passed.

### Excellence Bar self-check

- [x] Yes — the artifacts are bounded, machine-validated where appropriate, explicit about uncertainty, and give each downstream public edit one authoritative owner and replacement contract.
