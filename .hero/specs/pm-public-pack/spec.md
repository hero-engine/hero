---
title: Public Product Management Capability Pack
slug: pm-public-pack
type: feature
status: completed
domain: engineering
size: medium
priority: P0
created: 2026-08-22
tags: [domains, pm, open-source, content-pack, harnesses]
relations:
  - target: hero-domains
    kind: parent
  - target: hero-pm
    kind: related
  - target: hybrid-content-packs-and-workflow-providers
    kind: related
blocks:
  - dual-mode-pm-qa-capability-packs
horizon: now
smoke: deferred
delivery_method: manual
completed_at: 2026-08-22T22:37:32Z
---

# Public Product Management Capability Pack

## Goal

Finish and verify the Apache-ready Product Management content pack that belongs
in the public Hero repository. A PM must be able to install the pack offline as
the primary Hero experience in every supported harness and receive substantive
agents, skills, workflows, routing guidance, and canonical PM artifact types.

## Kickoff

Audit the existing `domains/pm/` pack against its locked public content design,
fill its missing artifact-type overlays, remove accidental scaffolding, embed
it as a first-class available domain, and prove native installation across all
supported harnesses.

**Status:** planning — public content boundary and verification contract locked.

**Pick up at:** inventory `domains/pm/` against the roster and artifact table in
`hero-pm`; author the missing `feature`/story, epic, and roadmap-item overlays;
then run pack integrity and all-harness installation tests.

→ `/deliver pm-public-pack`

**Files:** `domains/pm/`, `content.go`, pack/install tests, this spec.
**Skip:** proprietary Hero Code dashboards, hosted collaboration, new tracker
connectors, and duplicate `lite` or `full` PM trees.

## Problem

Hero already carries a large PM content tree, but the canonical PM design still
describes unshipped artifact overlays and has never been closed as an
independently verified public pack. The larger `hero-pm` spec also combines
public content with proprietary Hero Code surfaces, so it cannot serve as the
delivery boundary for the open-source repository. Dual-mode setup must not
advertise PM until this public slice is complete and installable by itself.

## Design

The public pack is the complete model-facing PM capability layer: `AGENTS.md`,
mission, agents, skills, commands, and spec-type declarations. It is bundled in
the Hero binary and works without a network connection. Shared Core artifacts
remain canonical; PM declarations describe PM-owned types or explicit PM
overlays and never fork a second copy of a Core type. The broad `hero-pm` spec
continues to own the private UI design and is not completed by this delivery.

Pack quality is structural and semantic. Files must contain actionable prompt
or workflow content, internal references must resolve, the advertised roster
must match the installed files, and README-only directory markers do not count
as capabilities. Existing PM files that are already substantive should remain
unchanged unless needed to correct a verified mismatch.

## Acceptance Criteria

- THE SYSTEM SHALL expose `pm` through `DomainFS` and `AvailableDomains` as an embedded pack that requires no network access.
- THE PM PACK SHALL contain substantive `AGENTS.md`, mission, agent, skill, command, and spec-type content with no delivery placeholders or empty capability files.
- THE PM PACK SHALL provide PM authoring behavior for canonical Core feature/story and epic records without redefining them, and SHALL register PM-owned roadmap-item guidance in addition to its existing PRD and intake types.
- THE PM PACK SHALL keep Core-owned type identities canonical and SHALL NOT install a second shadow definition for a shared type at the same resolved path.
- WHEN the PM pack is installed as the primary domain THE SYSTEM SHALL include Core plus PM content and SHALL exclude engineering-only pack content.
- WHEN Hero installs the PM pack for Claude, Codex, Cursor, Copilot, OpenCode, Grok, or Generic THE SYSTEM SHALL render the harness-native agents, skills, and command representation expected by that target.
- IF a PM agent, skill, command, or spec-type reference names a missing pack artifact THEN automated validation SHALL fail with the missing reference identified.
- THE SYSTEM SHALL provide automated inventory coverage that detects a removed advertised PM capability or a reintroduced placeholder.
- THE PUBLIC PM DELIVERY SHALL NOT add or require Hero Code, Hero Cloud, hosted services, or a new tracker connector.

## Changes

- `domains/pm/` — complete the public PM artifact declarations and correct any inventory or reference drift.
- `content.go` — retain PM as a first-class embedded domain and cover its public availability contract.
- `internal/install/` and content tests — add pack-integrity and seven-harness primary-install coverage.
- `.hero/planning/features/hero-pm/spec.md` — record that the public content slice is delivered separately while private surfaces remain open.

## Test Plan

- Run content inventory and internal-reference validation over the PM pack.
- Install PM into isolated workspaces for every supported harness and inspect the rendered native paths.
- Assert Core + PM resolution and the absence of engineering-only roles.
- Run `go test ./...` before the closing audit.

## Out of Scope

- Hero Code PM dashboards, widgets, or proprietary application behavior.
- Hero Cloud collaboration, synchronization, or billing behavior.
- Productboard, Aha, or other new tracker/provider integrations.
- Dual-mode extension activation, which is delivered by `dual-mode-pm-qa-capability-packs`.

## Completion Ledger

Delivered 2026-08-22. The existing PM pack was already substantive; delivery
registered the missing PM-owned roadmap artifact, preserved canonical Core
feature/epic identities, and added executable inventory and native-install
contracts across agents, skills, commands, and spec types.

| Acceptance criterion | Status | Evidence |
|---|---|---|
| `pm` is embedded and advertised offline | DONE | Existing `pmContent`, `DomainFS("pm")`, and `AvailableDomains`; `TestAvailableDomains` and `TestDomainFS_KnownDomains` pass. |
| Substantive PM pack with no empty capabilities | DONE | Existing PM corpus retained; PM agents and skills now participate in required-frontmatter/body validation. |
| Canonical feature/epic authoring plus PM-owned roadmap item | DONE | Existing `story-writer`/`epic-framer` roles target Core types; loadable `roadmap-item.md` added and covered by `TestLoad_PMIncludesOwnedTypesWithoutShadowingCore`. |
| Core-owned identities remain canonical | DONE | PM registry loading proves `feature` and `epic` remain `domain: core`; no PM-local shadows exist. |
| Primary PM install resolves Core + PM without engineering-only content | DONE | Existing overlay contract and `TestOverlay_AllTargetsIncludeCoreAndDomain`; PM-specific `prd-author` and Core `session-primer` are asserted. |
| Seven harness-native PM installs | DONE | `TestOverlay_AllTargetsIncludeCoreAndDomain` passes for Claude, Codex, Cursor, Copilot, OpenCode, Grok, and Generic. |
| Missing PM references fail validation | DONE | Complete inventory plus cross-reference validation covers agent load declarations, command agent/skill routing, skill cross-references, and spec-type default agents; negative fixtures name each missing target. |
| Inventory detects removed capability or malformed placeholder | DONE | Required capability files are checked for presence, substantive size, and explicit TODO/TBD/stub/placeholder markers; a long-placeholder negative fixture proves semantic detection. |
| No proprietary dependency added | DONE | Changes are embedded Markdown and local Go tests only; no Hero Code, Hero Cloud, hosted service, or connector dependency changed. |

| Planned change | Status | Evidence |
|---|---|---|
| Complete `domains/pm/` declarations | DONE | Registered PM-owned `roadmap-item`, documented Core ownership for shared types, embedded mission, and corrected three malformed YAML descriptions. |
| Preserve `content.go` public availability | DONE | PM mission is now embedded with the existing pack; domain advertisement remains green. |
| Add content/install coverage | DONE | Complete 30/19/51 inventory, cross-artifact reference and placeholder validation, real `spectypes.Load("pm")`, and seven-target agent/skill/command assertions pass. |
| Record the separated public slice in `hero-pm` | DONE | Broad spec retains private UI scope and points to this focused public delivery. |

**Exercised:** focused PM content, registry, and seven-target install tests pass;
`go test ./...` passes. The first cold audit correctly rejected invalid shared
type shadows; the corrected registry test now proves the runtime surface loads.

**Excellence:** The delivery avoided rewriting dozens of already-useful prompts.
It closes the exact public-pack gap, turns latent malformed frontmatter into
enforced contracts, and keeps one canonical artifact identity across PM and
engineering instead of weakening the registry to admit shadows.
