---
title: "Hero Positioning — Reduce Supervision Through Durable Project Intelligence"
slug: hero-positioning
type: feature
status: planning
domain: engineering
size: medium
priority: high
horizon: now
created: 2026-04-25
tags: [marketing, positioning, messaging, continuity, trust]
parent: hero-marketing
depends-on: [hero-public-truth-baseline]
---

# Hero Positioning — Reduce Supervision Through Durable Project Intelligence

## Goal

Preserve the original positioning work's narrative, ICP, messaging-house, vocabulary, fair comparison, and boilerplate intent while replacing its “spec layer” headline and count-based proof with one evidence-backed source of truth about reduced supervision, continuity, and trusted completion.

## Kickoff

Defines the message hierarchy every repaired Hero surface inherits: less re-explaining and supervision because project intent, decisions, corrections, and evidence survive.

**Status:** planning — relocated from the original top-level feature and reframed; the truth baseline must land first.

**Pick up at:** use the baseline registry to choose the lead audience and draft the messaging house without promoting unverified capabilities.

→ `.hero/planning/initiatives/hero-marketing/hero-positioning/spec.md`

**Files:** `.hero/planning/initiatives/hero-marketing/content-truth-audit.md`, `.hero/marketing/positioning.md`, `README.md`, `web/docs/src/what-is-hero.md`, `web/landing/site/index.html`
**Skip:** “spec layer” as category, mutable roster counts as proof, and “Correct your AI once” as an absolute promise.

## Context and provenance

This is the canonical continuation of the feature originally created at `.hero/planning/features/hero-positioning/spec.md` on 2026-04-25. The original semantics are retained: a shared one-liner/elevator pitch, primary and secondary audiences, what Hero is/is not, jobs to be done, objections, vocabulary, comparison guidance, audience→pain→message→proof mapping, candidate taglines, and reusable boilerplate.

The direction changes because the shipped product and public evidence changed. Specs are a mechanism for trustworthy execution, not the market category. The strongest truthful story is reduced supervision through durable project intelligence and evidence-backed completion.

## Changes

1. Rewrite `.hero/marketing/positioning.md` as the public messaging authority.
   - Define category, outcome, mechanism, primary/secondary audience, jobs, objections, and what Hero is/is not.
   - Map every proof pillar to entries in the public-truth baseline and availability labels.
2. Define the messaging house and proof pillars.
   - Lead with continuity across sessions/tools, corrections that survive, and completion against cold-audit/verify evidence.
   - Explain specs as the forcing function and engineering as the lead wedge.
3. Define vocabulary and prohibited claims consumed by root docs, hosted docs, and landing work.
   - Include harness-native workflow terminology, one-graph-per-project peering, and maturity language.
   - Prohibit unsupported licensing, cloud/team/outpost readiness, raw counts as differentiation, and fictional output presented as real.
4. Produce candidate taglines and boilerplate without publishing them yet.
   - Keep “Correct your AI once” only as a test candidate with a named proof threshold.
   - Provide 25-, 50-, and 150-word blocks after the lead audience is chosen.
5. Write fair comparison guidance focused on category boundaries rather than a broad competitor campaign.
   - Explain how Hero complements coding harnesses, rule files, wikis, and task trackers.
   - Defer competitor-specific public pages until separate current research exists.

## Acceptance Criteria

- **AC-1:** THE SYSTEM SHALL define “operating layer for AI-assisted engineering” as the category and “AI sessions inherit the project and finish against evidence” as the outcome, unless documented user research justifies a revision.
- **AC-2:** WHEN any proof point appears in the positioning source THE SYSTEM SHALL link it to a baseline claim with an evidence class and availability label.
- **AC-3:** WHEN audience messaging is finalized THE SYSTEM SHALL choose one lead audience and label all secondary/expansion audiences explicitly.
- **AC-4:** IF “Correct your AI once” is retained THEN THE SYSTEM SHALL label it as a candidate and name the repeatable proof required before publication.
- **AC-5:** THE SYSTEM SHALL provide preferred vocabulary, prohibited claims, jobs to be done, objections, fair comparison guidance, and 25/50/150-word boilerplate.
- **AC-6:** IF PM or Sales appears in public positioning THEN THE SYSTEM SHALL describe it as maturity-labeled expansion after the engineering wedge.

## Boundaries

- No landing, README, hosted-docs, metadata, or repository-description publication in this child.
- No pricing, visual brand system, launch campaign, or sales collateral.
- No competitor-specific public pages without fresh sourced research.
- No unconditional open-source, cloud/team/outpost, domain-readiness, or code-host-readiness claims.

## Validation

- Review every positioning proof against `content-truth-audit.md` and the materialized baseline registry.
- Scan downstream child specs for vocabulary/prohibited-claim compatibility before they enter delivery.
- Test candidate copy for comprehension with a small engineering cohort; record evidence rather than treating preference as fact.
- Run `hero spec lint hero-positioning`, `hero spec score hero-positioning`, and `hero index`.

