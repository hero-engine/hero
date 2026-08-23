---
title: "Hero Hosted Documentation Truth and Capability Refresh"
slug: hero-hosted-docs-remediation
type: feature
status: planning
domain: engineering
size: large
priority: high
horizon: now
created: 2026-08-04
tags: [documentation, website, information-architecture, truth]
parent: hero-marketing
depends-on: [hero-public-truth-baseline, hero-positioning]
relations:
  - target: hero-public-docs-drift-guard
    kind: conflicts-with
  - target: hero-landing-message-refresh
    kind: conflicts-with
supersedes: [hero-docs-site]
---

# Hero Hosted Documentation Truth and Capability Refresh

## Goal

Audit every hosted documentation page against implementation evidence, correct false or stale content, organize the product around durable project memory plus its connected delivery system, bound documentation dependencies, and restore deployment parity between current source and the stale June 11 live site.

## Kickoff

Turns a structurally green but factually drifting docs site into the authoritative product reference.

**Status:** planning — blocked on the claim baseline and positioning source.

**Pick up at:** map every `web/docs/src` page and navigation entry to claim-registry rows, identify why newer source pages are absent from the June 11 live deployment, then resolve P0/P1 content before adding missing capability coverage.

→ `.hero/planning/initiatives/hero-marketing/hero-hosted-docs-remediation/spec.md`

**Files:** `web/docs/src/`, `web/docs/mkdocs.yml`, documentation generation scripts and navigation
**Skip:** landing-page copy and broad visual rebranding.

## Changes

1. Perform a page-by-page claim audit; repair satellite architecture, configuration, closing gates, harness behavior, peering, repository layout, versions, counts, and domain-pack maturity.
2. Make release/current-version framing derived and current; reconcile locally present Focus/Mail/Releases coverage with deployed navigation.
3. Add discoverable, evidence-backed coverage for continuity, cold audit/verify, Attention/Mail/Focus, `hero serve`, guarded code-host operations, tracker evidence, peering/handoffs, and approval-aware runtime behavior.
   - Create a clear project-memory path covering capture, structure, retrieval, continuity, corrections, decisions, evidence, and cross-session/tool use.
   - Create a distinct delivery-system path covering design, specs, agents, implementation, cold audit, verification, and knowledge capture.
   - Explain how the two paths reinforce one another without positioning Hero as a generic spec kit.
4. Rewrite server/MCP and CLI references from current registries and actual subcommands; state provider/setup/approval prerequisites.
5. Bound MkDocs/theme/plugin dependencies away from the warned incompatible MkDocs 2.0 transition, repair deployment automation, and expose a deployed revision marker proving source/live parity.
6. Align terminology, links, metadata, availability labels, and the Hero/Sprout/proprietary repository boundary with positioning and the landing refresh.

## Acceptance Criteria

- **AC-1:** THE SYSTEM SHALL resolve every hosted-doc P0/P1 claim in the public claim registry.
- **AC-2:** WHEN hosted docs describe a command or configuration THE SYSTEM SHALL validate it against the current command tree or production decoder.
- **AC-3:** WHEN a differentiated capability is documented THE SYSTEM SHALL state availability, prerequisites, safe action boundaries, and evidence.
- **AC-4:** THE SYSTEM SHALL provide discoverable navigation for continuity, verification, Attention/Mail/Focus, `hero serve`, peering, code-host, and tracker capabilities.
- **AC-5:** WHEN release or version information appears THE SYSTEM SHALL derive it from the agreed public release authority and identify source revision/freshness.
- **AC-6:** THE SYSTEM SHALL build the docs in strict mode with valid links, navigation, anchors, and no unresolved assigned claims.
- **AC-7:** WHEN hosted docs deploy THE SYSTEM SHALL use compatibility-bounded dependencies and expose a revision proving the newer source, not the stale June 11 build, is live.
- **AC-8:** THE SYSTEM SHALL provide distinct, linked navigation for project memory and the spec-and-agent delivery system, with memory presented as Hero's primary differentiated value.

## Boundaries

- No claims based only on planning specs or sibling/private implementation.
- No broad tutorial/video library, landing rewrite, or competitor campaign.
- If the work becomes x-large, split factual remediation before capability discoverability.
- No license or visibility mutation and no open-source claim for `hero-code` or `hero-cloud`.

## Validation

- Strict MkDocs build, internal/external link and anchor checks, executable example/config tests, registry reconciliation, Hero lint/score, and index refresh.
