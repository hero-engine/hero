---
title: Hero PM — Product Management Domain Pack
type: feature
status: planning
priority: P0
tags: [platform, domains, product-management, roadmap, content-pack]
created: 2026-05-15
relations:
  - target: hero-domains
    kind: parent
depends-on:
  - domain-plugin-architecture
  - spec-type-registry
  - domain-routing-and-agents
  - dashboard-view-registry
  - scan-pluggability
  - domain-scoped-knowledge-graph
horizon: next
smoke: deferred
---

> **Status: awaiting platform primitives.** This stub is a `/design`-ready
> brief, not a complete design. Implementation is blocked on work items
> 1–5 of the parent `hero-domains` initiative
> (`domain-plugin-architecture`, `spec-type-registry`,
> `domain-routing-and-agents`, `dashboard-view-registry`,
> `scan-pluggability`). `/design hero-pm` can resolve the open questions
> below and produce the full design in parallel with primitive delivery,
> but no code lands until the primitives are in place.

## Kickoff

First non-engineering Hero domain pack. PM-shaped spec types (PRD, story, epic, roadmap-item), PM agents and workflows, PM dashboard views. Reuses existing tracker integrations (Jira/Linear/GitHub). Killer demo: a Jira epic becomes a Hero story, `/design` turns it into an engineering `feature` spec, and the handoff edge appears in the graph.

**Status:** planning — `/design`-ready brief written 2026-05-15. Implementation blocked on parent initiative's platform primitives (work items 1–5: domain-plugin-architecture, spec-type-registry, domain-routing-and-agents, dashboard-view-registry, scan-pluggability).

**Pick up at:** Either wait for the platform primitives, or run `/design hero-pm` ahead in parallel to resolve the 5 open questions (OKRs in/out, cross-tracker handoff, single- vs multi-domain projects, acceptance-criteria format, roadmap horizon model) so design lands the moment primitives are ready.

→ `/design hero-pm`

**Files:** .hero/planning/features/hero-pm/spec.md, .hero/planning/initiatives/hero-domains/spec.md, .hero/planning/features/domain-plugin-architecture/spec.md
**Skip:** New tracker integrations in v1 (reuse Jira/Linear/GitHub only). OKRs as a PM spec type (defer to v2; may belong in a separate `strategy` domain).

## Goal

Ship the first non-engineering Hero domain pack: Product Management.
Provides PM-shaped spec types (PRD, story, epic, roadmap-item), PM
agents and workflows, PM-specific dashboard views, and onboarding that
imports an existing roadmap or tracker into a queryable spec corpus.
Reuses existing tracker integrations (Jira/Linear/GitHub) — no new
integrations in v1. Success means a PM can take a Jira epic into Hero,
refine it into stories with acceptance criteria, hand a story off to
engineering as a `feature` spec, and see the handoff edge appear in the
knowledge graph.

## Artifact types

The pack declares the following spec types via the spec-type registry.
Each type has its own lifecycle and frontmatter schema.

| Type | Purpose | Lifecycle | Notes |
|---|---|---|---|
| `prd` | Product requirement doc | draft → review → approved → delivered | Largest artifact; references stories and roadmap-item |
| `story` | User story / dev-ready unit | drafted → refined → ready → in-flight → done | Handoff atom into `/design` and `/deliver` |
| `epic` | Container for related stories | proposed → committed → in-flight → done | Maps to tracker epic |
| `roadmap-item` | Coarse-grained future bet | candidate → committed → shipped → dropped | Quarterly horizon |
| `okr` _(open)_ | Objectives + KRs | active → closed | Deferred to v2; possibly a separate `strategy` domain |

## Workflows

1. **Intake** — capture inbound (customer feedback, internal asks,
   competitive signals); triage into a roadmap-item or reject with reason.
2. **Refinement** — turn a roadmap-item into a PRD and child stories;
   apply INVEST shape and explicit acceptance criteria on each story.
3. **Prioritization** — sequence roadmap-items against capacity,
   dependencies, and OKRs.
4. **Roadmap maintenance** — keep roadmap status fresh from delivery
   signal (in-flight, shipped, dropped).
5. **Handoff to engineering** — `/design` on a `story` produces an
   engineering `feature` spec; the knowledge graph records the handoff
   edge. This is the killer demo of the multi-domain platform.

## Agents

- **product-strategist** — roadmap framing, opportunity sizing,
  prioritization tradeoffs.
- **story-writer** — produces INVEST-shaped stories with acceptance
  criteria.
- **roadmap-curator** — maintains the roadmap view, surfaces stale
  items, reconciles delivered work against the roadmap.
- **intake-triager** — classifies inbound; routes to roadmap or
  rejects with reasoning.
- **prd-author** _(maybe)_ — could be a skill on `product-strategist`
  rather than a standalone agent. Resolve in design.

## Commands

- **New commands** — `/refine`, `/triage`, `/roadmap`.
- **Reused commands** — `/design`, `/deliver`, `/diagnose` remain
  cross-domain workflows that accept PM spec types (notably `/design`
  accepting `story` to produce an engineering `feature`).

## Integrations

**v1 reuses existing tracker integrations (Jira, Linear, GitHub) —
no new integrations.** Importing Jira epics as `epic` specs and Jira
stories as `story` specs is the v1 onboarding moment.

This is an explicit decision, not a punt: it lets PM ship on the same
integration surface engineering already has, and stress-tests whether
the `DomainIntegration` interface tolerates two domains sharing one
provider before we add roadmap-shaped providers (Productboard, Aha) in
a follow-up.

## Dashboard views

The pack registers four views via the dashboard view registry. The
default landing page is Roadmap.

- **Roadmap** _(default landing)_ — committed and candidate
  roadmap-items by quarter or now/next/later.
- **Story queue** — refined and ready stories awaiting engineering
  handoff.
- **Intake funnel** — raw inbound → triaged → roadmap-item.
- **Handoff stream** — recent story→feature handoffs and their
  delivery status.

## Unknowns for design pass

These must be resolved during `/design hero-pm` before delivery:

1. **OKRs — PM domain or separate `strategy` domain?** Defer
   recommendation; design decides whether `okr` lives here or in a
   future strategy pack.
2. **Cross-tracker handoff** — does the platform need to support a PM
   project on Jira handing off to an engineering project on Linear?
   Plausible, adds scope. Decide in/out for v1.
3. **Domain coexistence model** — should PM and engineering live in
   the same workspace (one `.hero/` dir, two active packs) or as
   separate projects sharing a knowledge graph? Affects whether the
   graph namespace tags from `domain-scoped-knowledge-graph` (item #6)
   are required for v1.
4. **Story acceptance criteria format** — free-text, Gherkin, or
   structured frontmatter? Engineering uses EARS today; PM stories
   need a compatible-enough shape that `/design` on a story produces a
   useful feature spec.
5. **Roadmap horizon model** — time-based (Q1/Q2/Q3) or ordered-bets
   (now/next/later)? Affects the Roadmap view, the roadmap-item
   lifecycle, and `roadmap-curator`'s prioritization signal.

## Boundaries

- **Not** designing OKR support in v1 — see unknown #1.
- **Not** adding new integration providers in v1 — Jira/Linear/GitHub
  reuse only.
- **Not** building a PM-only Hero binary — same `hero` binary, domain
  selected via `hero.json`.
- **Not** building cross-domain reporting (PM-eng combined dashboards)
  — that's a follow-up after both domains ship.
- **Not** modeling product analytics, experiment results, or metrics
  pipelines — those belong to a future `hero-data-analytics`
  initiative.

## Risks

- **PM-as-first-domain feels like incremental engineering.** Mitigated
  by making the handoff edge (story→feature) a hero demo, and by
  planning `hero-qa` (item #8 in parent) on a real cadence.
- **Spec-type registry surface area surprises.** If item #2 of the
  parent initiative underestimates the audit, PM design and delivery
  slip with it.
- **Tracker reuse hides design tension.** Reusing Jira for PM and
  engineering is right for v1 but masks integration-shape questions
  that the QA pack and later domains will surface.
