---
title: "Hero Positioning — Durable Project Memory and Verified Delivery"
slug: hero-positioning
type: feature
status: completed
domain: engineering
size: medium
priority: high
horizon: now
created: 2026-04-25
tags: [marketing, positioning, messaging, continuity, trust]
parent: hero-marketing
depends-on: [hero-public-truth-baseline, hero-licensing-boundary-and-provenance]
delivery_method: manual
completed_at: 2026-08-23T16:47:06Z
---

# Hero Positioning — Durable Project Memory and Verified Delivery

## Goal

Define Hero clearly as two connected systems in one: durable project memory as the differentiated headline, and a spec-and-agent delivery system as the execution layer that turns that memory into verified work. Preserve the original positioning work's narrative, ICP, messaging house, vocabulary, fair comparison, and boilerplate intent while removing the impression that Hero is merely another spec kit.

## Kickoff

Defines the message hierarchy every repaired Hero surface inherits: project memory first, delivery system second, and a reinforcing loop in which each verified delivery leaves better context for the next session.

**Status:** delivering — the baseline and licensing boundary are complete; the positioning authority is implemented and undergoing its closing review.

**Pick up at:** validate the messaging source against the baseline registry, run the cold audit, and close the spec.

→ `.hero/planning/initiatives/hero-marketing/hero-positioning/spec.md`

**Files:** `.hero/planning/initiatives/hero-marketing/content-truth-audit.md`, `.hero/marketing/positioning.md`, `README.md`, `web/docs/src/what-is-hero.md`, `web/landing/site/index.html`
**Skip:** “spec layer” as category, mutable roster counts as proof, and “Correct your AI once” as an absolute promise.

## Context and provenance

This is the canonical continuation of the feature originally created at `.hero/planning/features/hero-positioning/spec.md` on 2026-04-25. The original semantics are retained: a shared one-liner/elevator pitch, primary and secondary audiences, what Hero is/is not, jobs to be done, objections, vocabulary, comparison guidance, audience→pain→message→proof mapping, candidate taglines, and reusable boilerplate.

The direction changes because the shipped product and public evidence changed. Hero's memory system preserves project intent, decisions, corrections, conventions, evidence, and current state across sessions, tools, and agents. Its spec-and-agent system uses that memory to structure, implement, cold-audit, and verify work. Specs are a mechanism for trustworthy execution, not the market category.

## Changes

1. Rewrite `.hero/marketing/positioning.md` as the public messaging authority.
   - Define Hero as a project memory and delivery system for AI-assisted engineering, along with its outcome, primary/secondary audience, jobs, objections, and what Hero is/is not.
   - Make durable project memory the lead promise and explain the spec-and-agent loop as the connected execution system.
   - Show the reinforcing loop: delivery reads project memory, work is implemented and verified, and the result becomes better memory for later sessions.
   - Map every proof pillar to entries in the shipped-surface inventory and availability labels.
2. Define the messaging house and proof pillars.
   - Lead with continuity across sessions, tools, and agents; corrections and decisions that survive; and less repeated explanation.
   - Explain specs and agents as the forcing function for completion against cold-audit/verify evidence, with engineering as the lead wedge.
   - Prevent page structure or copy from collapsing the two systems into a generic “spec-driven development” pitch.
3. Define vocabulary and prohibited claims consumed by root docs, hosted docs, and landing work.
   - Include harness-native workflow terminology, one-graph-per-project peering, and maturity language.
   - State that only this `hero` repository is an Apache-2.0 candidate, Sprout is separately MIT-licensed, and `hero-code`/`hero-cloud` remain proprietary.
   - Prohibit unsupported licensing, cloud/team/outpost readiness, raw counts as differentiation, and fictional output presented as real.
4. Produce candidate taglines and boilerplate without publishing them yet.
   - Keep “Correct your AI once” only as a test candidate with a named proof threshold.
   - Provide 25-, 50-, and 150-word blocks after the lead audience is chosen.
5. Write fair comparison guidance focused on category boundaries rather than a broad competitor campaign.
   - Explain how Hero complements coding harnesses, rule files, wikis, and task trackers.
   - Defer competitor-specific public pages until separate current research exists.

## Acceptance Criteria

- **AC-1:** THE SYSTEM SHALL define Hero as a project memory and delivery system for AI-assisted engineering, with durable memory as the primary differentiated promise and verified delivery as its connected execution layer.
- **AC-2:** WHEN any proof point appears in the positioning source THE SYSTEM SHALL link it to a baseline claim with an evidence class and availability label.
- **AC-3:** WHEN audience messaging is finalized THE SYSTEM SHALL choose one lead audience and label all secondary/expansion audiences explicitly.
- **AC-4:** IF “Correct your AI once” is retained THEN THE SYSTEM SHALL label it as a candidate and name the repeatable proof required before publication.
- **AC-5:** THE SYSTEM SHALL provide preferred vocabulary, prohibited claims, jobs to be done, objections, fair comparison guidance, and 25/50/150-word boilerplate.
- **AC-6:** IF PM or Sales appears in public positioning THEN THE SYSTEM SHALL describe it as maturity-labeled expansion after the engineering wedge.
- **AC-7:** WHEN the positioning is read without surrounding context THE SYSTEM SHALL make clear that Hero is two reinforcing systems in one and SHALL NOT read as merely another spec kit.

## Boundaries

- No landing, README, hosted-docs, metadata, or repository-description publication in this child.
- No pricing, visual brand system, launch campaign, or sales collateral.
- No competitor-specific public pages without fresh sourced research.
- No unconditional open-source, cloud/team/outpost, domain-readiness, or code-host-readiness claims.
- No license or visibility mutation, and no language implying `hero-code` or `hero-cloud` is open source.

## Validation

- Review every positioning proof against `content-truth-audit.md` and the materialized baseline registry.
- Scan downstream child specs for vocabulary/prohibited-claim compatibility before they enter delivery.
- Before promoting any test-only candidate into published copy, test it for comprehension with a small engineering cohort and record evidence rather than treating preference as fact. This is a downstream publication experiment, not a gate for establishing the canonical positioning authority.
- Run `hero spec lint hero-positioning`, `hero spec score hero-positioning`, and `hero index`.

## Completion Ledger

The positioning authority was written against the executable public-claim baseline and the completed licensing boundary. The copy is intentionally not published by this child; root docs, hosted docs, landing, repository, and release children consume it after their own evidence gates.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Define memory as the primary promise and verified delivery as the connected layer | DONE | `.hero/marketing/positioning.md` defines the category, outcome, two-system model, messaging house, and reinforcing loop with memory first. |
| 2 | Link proof points to baseline claim, evidence class, and availability | DONE | The proof register maps every public proof to named baseline claims, evidence classes, availability, and required qualifiers; every registry ID referenced by the authority was validated against the registry. |
| 3 | Choose one lead audience and label all others | DONE | AI-native engineers and hands-on technical leads are the sole lead audience; secondary and expansion audiences are explicitly separated. |
| 4 | Bound “Correct your AI once” behind repeatable proof | DONE | The phrase is test-only and requires a revision-tied, cross-harness cold-session test to pass ten of ten runs across at least two harness pairings. |
| 5 | Supply the complete public messaging toolkit | DONE | The authority contains preferred vocabulary, prohibited claims, jobs, objections, comparison guidance, taglines, and exact 25/50/150-word boilerplates. |
| 6 | Treat PM and Sales as expansion after engineering | DONE | Lightweight PM/QA help is bounded to Engineering; focused PM, QA, and Sales are maturity-labeled expansion paths rather than the lead story. |
| 7 | Make the two reinforcing systems legible without surrounding context | DONE | The one-sentence position, category, system breakdown, messaging house, proof register, and spec-kit objection independently preserve the two-system hierarchy. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Rewrite the public messaging authority | DONE | Added `.hero/marketing/positioning.md` as the canonical source with category, audiences, jobs, proof, boundaries, and surface rules. |
| 2 | Define messaging house and proof pillars | DONE | Added the “Your project remembers. Your agents finish.” roof with memory, verified delivery, and harness-native/project-owned pillars. |
| 3 | Define vocabulary and prohibited claims | DONE | Added exact terminology plus licensing, proprietary-product, maturity, architecture, evidence, and count prohibitions. |
| 4 | Produce candidate taglines and exact boilerplate | DONE | Added bounded candidates and blocks validated as exactly 25, 50, and 150 words under both editorial-token and whitespace counting. |
| 5 | Write fair category comparison guidance | DONE | Added complementary boundaries for coding harnesses, rule files, wikis, trackers, and spec frameworks; named competitors remain deferred to researched comparison work. |

### Exercise-the-feature check

- [x] Read the positioning source as a standalone public authority, validated every referenced registry claim ID against the materialized registry, checked the three boilerplates under two word-count methods, scanned every downstream initiative child for incompatible vocabulary and boundary claims, and ran Hero lint (7/7 EARS), score (95/A), and index successfully. Candidate-tagline cohort testing remains a named pre-publication experiment rather than fabricated delivery evidence.

### Excellence Bar self-check

- [x] Yes — the authority tells one memorable story without flattening Hero into a spec kit, separates evidence from aspiration, names who the product is for, and gives every downstream surface enforceable language and product-boundary constraints.
