---
title: Public Quality Assurance Capability Pack
slug: qa-public-pack
type: feature
status: completed
domain: engineering
size: large
priority: P0
created: 2026-08-22
tags: [domains, qa, open-source, content-pack, harnesses]
relations:
  - target: hero-domains
    kind: parent
  - target: hero-qa
    kind: related
  - target: hybrid-content-packs-and-workflow-providers
    kind: related
blocks:
  - dual-mode-pm-qa-capability-packs
horizon: now
smoke: deferred
delivery_method: manual
completed_at: 2026-08-22T23:00:11Z
---

# Public Quality Assurance Capability Pack

## Goal

Ship a complete, Apache-ready Quality Assurance content pack in the public Hero
repository. A QA practitioner must be able to install QA as the primary Hero
experience in any supported harness and use it offline for test planning, case
authoring, exploratory work, defect triage, regression and flake curation,
coverage analysis, and release gates.

## Kickoff

Create `domains/qa/` from the locked public content design in `hero-qa` and its
agent-pack design, embed it as a first-class domain, and prove substantive
content plus native installation for every harness.

**Status:** delivering — public pack content, registry wiring, all-harness tests,
and offline Codex smoke are complete; cold audit and Hero verification remain.

**Pick up at:** cold-audit the completed public pack against this spec, address
any HOLD finding, then run `hero spec verify qa-public-pack --skip-tests`.

→ `/deliver qa-public-pack`

**Files:** new `domains/qa/`, `content.go`, pack/install tests, this spec.
**Skip:** proprietary Hero Code dashboards, hosted run-state services, TestRail
or Xray connector implementations, and duplicated lightweight/full trees.

## Problem

The QA product and agent-pack design exists, but no public `domains/qa/` pack
exists today. That makes `qa` impossible to initialize or install and leaves
the proposed engineering extension pointing at absent content. The broad
`hero-qa` design also mixes public workflow knowledge with private Hero Code UI
and integration surfaces, so the open-source pack needs its own verifiable
delivery boundary.

## Design

The first public QA pack is a complete practitioner experience, not a directory
of thin role-name stubs. It includes a domain routing body and mission; lead,
strategy, authoring, investigation, curation, review, release-gate, handoff, and
scrub roles; reusable testing-method skills; QA workflow commands; and artifact
definitions for test plans, test cases, regression suites, release gates, and
defects. Shared verbs may have QA-specific primary-pack instructions here;
dual-mode router composition is handled by the follow-on composition spec.

The pack is embedded in the binary and offline. Integration-aware guidance may
describe how adapters should expose evidence, but the pack cannot require an
external test-management service. Existing Core testing guidance is referenced
rather than copied. Every advertised artifact must resolve and contain useful
operational instructions.

## Acceptance Criteria

- THE SYSTEM SHALL expose `qa` through `DomainFS` and `AvailableDomains` as an embedded pack that requires no network access.
- THE QA PACK SHALL contain substantive `AGENTS.md`, mission, agent, skill, command, and spec-type content with no empty capability files or delivery placeholders.
- THE QA PACK SHALL provide workflows for coverage planning, test-plan and test-case authoring, exploratory charters, test-issue and flake triage, regression promotion, release gating, seam requests, and QA corpus scrubbing.
- THE QA PACK SHALL provide specialist roles spanning coordination, strategy, authoring, investigation, curation, review, release readiness, handoff, and hygiene.
- THE QA PACK SHALL declare canonical QA artifact guidance for test plans, test cases, regression suites, release gates, and defects with explicit lifecycle and evidence expectations.
- WHEN the QA pack is installed as the primary domain THE SYSTEM SHALL include Core plus QA content and SHALL exclude engineering-only pack content.
- WHEN Hero installs the QA pack for Claude, Codex, Cursor, Copilot, OpenCode, Grok, or Generic THE SYSTEM SHALL render the harness-native agents, skills, and command representation expected by that target.
- IF a QA agent, skill, command, or spec-type reference names a missing pack artifact THEN automated validation SHALL fail with the missing reference identified.
- THE SYSTEM SHALL provide automated inventory coverage that detects a removed advertised QA capability or a reintroduced placeholder.
- THE PUBLIC QA DELIVERY SHALL NOT add or require Hero Code, Hero Cloud, a hosted run-state service, TestRail, Xray, or another proprietary connector.

## Changes

- `domains/qa/` — add the public QA routing body, mission, agents, skills, commands, and artifact-type guidance.
- `content.go` — embed QA and advertise it as a bundled domain.
- `internal/install/` and content tests — add pack-integrity and seven-harness primary-install coverage.
- `.hero/planning/features/hero-qa/spec.md` — record that the public content slice is delivered separately while private surfaces remain open.

## Test Plan

- Run content inventory and internal-reference validation over the QA pack.
- Install QA into isolated workspaces for every supported harness and inspect the rendered native paths.
- Assert Core + QA resolution and the absence of engineering-only roles.
- Run `go test ./...` before the closing audit.

## Out of Scope

- Hero Code QA dashboards, widgets, or proprietary application behavior.
- Hero Cloud collaboration, run history, synchronization, or billing behavior.
- TestRail, Xray, or other external adapter implementations.
- Dual-mode extension activation, which is delivered by `dual-mode-pm-qa-capability-packs`.

## Completion Ledger

Stack: Go with embedded Markdown capability packs. Validation: focused content,
spec-type, and installer tests; full `go test ./...`; fresh binary build; offline
QA initialization plus Codex installation in an isolated Git workspace.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Expose embedded offline `qa` through `DomainFS` and `AvailableDomains` | DONE | `content.go` embeds QA, resolves `DomainFS("qa")`, loads QA spec types, and advertises the domain; `TestQACapabilityPackInventory` and `TestAvailableDomains` pass. |
| 2 | Ship substantive routing, mission, agents, skills, commands, and spec types without placeholders | DONE | `domains/qa/` contains a practitioner-grade body plus exact inventories of 23 agents, 31 skills, 18 commands, and 5 spec types; semantic-placeholder and minimum-content validation pass. |
| 3 | Cover planning, authoring, charters, triage, flakes, regression, gates, seams, and scrubbing | DONE | The 12 QA-specific routers in `domains/qa/commands/` exercise each named workflow; six primary and six cross-domain routes are also present. |
| 4 | Span coordination, strategy, authoring, investigation, curation, review, readiness, handoff, hygiene | DONE | `qaAgentInventory` in `qa_content_test.go` locks the full 23-agent roster across all required tiers. |
| 5 | Declare QA artifact guidance with lifecycle and evidence expectations | DONE | `domains/qa/spec-types/` defines test-plan, test-case, regression-suite, release-gate, and optional defect records with lifecycles, required sections, evidence fields, and QA agents. |
| 6 | Primary QA includes Core plus QA and excludes engineering-only content | DONE | `TestQAOverlay_AllTargetsRenderNativeCoreAndQASurfaces` asserts Core + QA and absence of `feature-delivery-lead` for every target. |
| 7 | Render native agents, skills, and commands for all seven harnesses | DONE | The all-target installer test verifies Claude, Codex, Cursor, Copilot, OpenCode, Grok, and Generic native paths. |
| 8 | Fail when a QA cross-reference names a missing artifact | DONE | Content-wide `validateAllArtifactReferences` resolves every backticked capability and every spec-type `ref(...)` target independent of sentence shape; a negative fixture proves freeform agent and spec-type failures name both source and target. |
| 9 | Detect removed advertised capabilities and semantic placeholders | DONE | Exact agent, command, skill, and spec-type inventory checks plus `validateRequiredPackFiles` guard removals, empty content, and semantic placeholders. |
| 10 | Do not require proprietary clients, hosted state, or connectors | DONE | The pack reads local artifacts and explicitly treats integrations as optional; build, tests, init, and install succeeded offline with no Hero Code, Hero Cloud, TestRail, or Xray dependency. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Add public QA body under `domains/qa/` | DONE | Added `AGENTS.md`, mission, 23 agents, 31 testing and QA-operation skills, 18 commands, and 5 canonical QA spec types. |
| 2 | Embed and advertise QA in `content.go` | DONE | Added `qaContent`, QA domain and spec-type resolution, and `AvailableDomains` registration. |
| 3 | Add integrity, reference, registry, and seven-harness tests | DONE | Added exact inventory, semantic-placeholder, content-wide capability/spec-reference, runtime registry, embedded-frontmatter, and seven-target native install coverage. |
| 4 | Record the public/private boundary in broad `hero-qa` | DONE | `.hero/planning/features/hero-qa/spec.md` now identifies `qa-public-pack` as the offline public slice while proprietary views and connectors remain open. |

### Exercise-the-feature check

- [x] User-visible behavior was exercised end-to-end: built a fresh `hero` binary, initialized an isolated Git project with `hero init --domain qa`, installed project-scoped Codex content, observed 27 Core+QA agent files and 79 Core+QA workflow skills, verified the QA lead/skill/command files, and confirmed the engineering delivery lead was absent.

### Excellence Bar self-check

- [x] Yes — the locked roster and workflow surface are complete, each capability has operational guidance rather than a thin role stub, and exact inventory, cross-reference, registry, native-install, full-suite, and real-binary smoke evidence protect the delivery.
