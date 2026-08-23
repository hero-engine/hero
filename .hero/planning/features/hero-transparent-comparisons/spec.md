---
title: "Transparent Comparisons — Help People Choose Hero or an Alternative Honestly"
slug: hero-transparent-comparisons
type: feature
status: planning
domain: engineering
size: medium
priority: medium
horizon: next
created: 2026-08-23
tags: [positioning, comparisons, alternatives, memory, spec-systems]
relations:
  - target: hero-positioning
    kind: depends-on
  - target: spec-kit-and-swappable-workflow-providers
    kind: related
---

# Transparent Comparisons — Help People Choose Hero or an Alternative Honestly

## Goal

Add a durable, evidence-backed comparison section that gives adjacent tools full credit for what they do well, identifies when they are the better choice, and makes Hero's distinct value unmistakable: a compounding project-memory system that delivers the right knowledge to agents at the moment of work, plus a verified delivery system that can coexist with external workflow providers.

## Kickoff

Adds a HashiCorp-style comparison section that helps readers honestly choose between Hero and adjacent spec, agent, memory, and context tools.

**Status:** planning — comparison posture and initial candidate set are defined; no public copy exists yet.

**Pick up at:** refresh every candidate from first-party sources, then write the shared comparison matrix and the Hero vs. Spec Kit page first.

→ `.hero/planning/features/hero-transparent-comparisons/spec.md`

**Files:** `.hero/marketing/positioning.md`, `web/docs/src/`, `web/landing/site/index.html`, `.hero/knowledge/notes/spec-kit-and-swappable-workflow-providers/`
**Skip:** attack copy, unverifiable feature tables, and framing every adjacent tool as a direct replacement.

## Problem

Hero overlaps several visible categories without fitting neatly inside any one of them. Spec-driven systems can make Hero look like another workflow kit. Harness-native rules and auto-memory can make its memory system look redundant. Context engines can sound similar while solving a narrower retrieval problem. Without a careful comparison surface, readers must infer the distinction themselves and may either dismiss Hero too early or receive an unfair picture of the alternatives.

The comparison must also remain current. These products change quickly, so a static feature-war table would become misleading. The right model is an editorial comparison contract: first-party evidence, explicit dates and maturity, generous “choose this when” guidance, and a small set of differences that actually matter.

## Positioning posture

- Lead with the alternative's genuine strength before describing Hero's difference.
- State when the alternative is the better fit.
- Compare product models and operating tradeoffs, not popularity, personalities, or speculative motives.
- Separate workflow systems, harness-native memory, and context engines instead of flattening them into one competitor category.
- Prefer “different center of gravity” over “missing feature” when products optimize for different jobs.
- Keep coexistence visible: Hero can use external spec systems as pluggable workflow providers while retaining project memory, retrieval, evidence, and continuity.
- Use plain language. “More ceremony” must be tied to observable phases or artifacts; “lighter weight” must be tied to what the user does not have to maintain.

## Initial comparison set

| Candidate | Category | Give it full credit for | Hero distinction to test |
|---|---|---|---|
| GitHub Spec Kit | Structured workflow system | Broad agent support, explicit spec→plan→tasks→implementation flow, extensibility, presets, and organizational process packaging | Spec Kit standardizes and sequences the work; Hero's center is the temporal project corpus, retrieval, continuity, and verified knowledge that compounds across work |
| OpenSpec | Lightweight spec/change system | A fluid brownfield-friendly agreement layer, approachable change artifacts, and shared spec stores | Hero owns more than agreement artifacts: decisions, failed attempts, relationships, evidence, handoffs, and context selection across sessions and tools |
| Kiro | Integrated agentic development environment | A cohesive prompt→spec→code/test experience plus steering files in one product | Hero is harness-independent and makes project memory portable across tools rather than binding continuity to one development environment |
| Claude Code memory / Cursor rules and memories | Harness-native memory | Low-friction persistent instructions, preferences, and automatically captured notes inside a chosen coding harness | Hero provides a shared, typed, relational, evidence-aware project corpus and cross-harness retrieval rather than only harness-local instructions or notes |
| Augment Context Engine | Code/context engine | Deep codebase understanding and retrieval across large repositories and services | Hero's corpus includes intent, decisions, failed attempts, work state, provenance, and delivery evidence—not only code and connected context—and explicitly closes the capture/retrieval loop after work |

The candidate set is provisional until delivery-time research. Add or remove products based on category relevance and public evidence, not name recognition. BMAD or another workflow framework may replace a weaker candidate if it materially improves the reader's decision.

## Changes

1. Create a comparison research registry.
   - Record each product's official description, documented capabilities, license/deployment model where relevant, source URL, evidence date, and last editorial review.
   - Distinguish facts, Hero's interpretation, and user-reported tradeoffs; public claims may use only facts and clearly labeled interpretation.
   - Require first-party sources for product capabilities and link directly to them.
2. Define a reusable public comparison template.
   - “What it is,” “What it does especially well,” “Choose it when,” “Choose Hero when,” “How they can work together,” and “Evidence reviewed on.”
   - Include a compact dimension matrix covering primary job, memory scope, workflow structure, harness portability, retrieval/capture loop, verification, and interoperability.
   - Avoid universal winners and aggregate scores.
3. Publish the first comparison: Hero vs. GitHub Spec Kit.
   - Describe Spec Kit as a strong way to structure and streamline agent workflows through explicit phases and extensible process assets.
   - Explain that this structure can be more ceremony than a small or fast-moving project wants without treating ceremony as inherently bad.
   - Explain Hero's different center: capturing durable project knowledge, selecting and delivering it when an agent needs it, using that context to make better decisions, and recording verified outcomes so later work starts smarter.
   - Make clear that Hero may run its own delivery system or integrate Spec Kit as a pluggable provider; the products are not mutually exclusive.
4. Add concise comparisons for the remaining validated candidates.
   - Group the navigation by workflow systems, harness-native memory, and context engines.
   - Prefer one useful comparison hub plus focused detail pages only where reader questions justify them.
5. Add editorial freshness and correction paths.
   - Display the evidence-review date and link to a correction issue/template.
   - Fail the comparison freshness check when a page exceeds the agreed review window or a cited primary source disappears.

## Acceptance Criteria

- **AC-1:** THE SYSTEM SHALL organize alternatives into workflow systems, harness-native memory, and context engines rather than presenting one undifferentiated competitor list.
- **AC-2:** WHEN a comparison names an alternative's capability THE SYSTEM SHALL support it with a dated first-party source and distinguish fact from Hero interpretation.
- **AC-3:** THE SYSTEM SHALL state what each alternative does especially well and when a reader should choose it instead of Hero.
- **AC-4:** WHEN Hero is compared with a spec-driven system THE SYSTEM SHALL explain that Hero's primary distinction is durable, relational project memory and timely context delivery, while its spec-and-agent workflow is a connected execution layer.
- **AC-5:** WHEN the Spec Kit comparison is published THE SYSTEM SHALL describe its structured, extensible workflow fairly, explain the observable ceremony tradeoff without ridicule, and describe the optional provider/coexistence path.
- **AC-6:** THE SYSTEM SHALL compare primary job, memory scope, workflow structure, harness portability, retrieval/capture loop, verification, and interoperability without an aggregate score or universal winner.
- **AC-7:** IF a comparison lacks current evidence, exceeds the review window, or cites a missing source THEN THE SYSTEM SHALL withhold or flag the affected claims rather than silently publish them as current.
- **AC-8:** THE SYSTEM SHALL provide a visible correction path and record the last evidence-review date on each published comparison.

## Validation

- Re-run delivery-time research against official product documentation and repositories; do not rely on this spec's 2026-08 snapshot as publication evidence.
- Test the Hero vs. Spec Kit copy with readers familiar with Spec Kit and revise any passage they identify as factually wrong or unfair.
- Ask readers unfamiliar with Hero to explain its distinction after reading the page; they should identify project memory and timely context delivery before spec ceremony.
- Crawl cited links and validate evidence dates in the docs build.
- Run `hero spec lint hero-transparent-comparisons`, `hero spec score hero-transparent-comparisons`, and `hero index`.

## Boundaries

- Not a v0.34 release blocker and not part of the current landing/docs remediation scope.
- No attack copy, anonymous user anecdotes presented as evidence, popularity ranking, or speculative claims about another team's intent.
- No claim that external workflow providers are shipped until the pluggable-provider architecture is implemented and verified.
- No broad SEO page farm; start with a small decision-useful surface.

## Evidence snapshot for design

- GitHub describes Spec Kit as an extensible spec-driven process with a default spec→plan→tasks→implementation flow, optional analysis/quality steps, workflows, presets, extensions, and role-oriented bundles.
- OpenSpec describes itself as a lightweight agreement layer with fluid, iterative, brownfield-oriented change artifacts and optional shared stores.
- AWS describes Kiro as an agentic coding service that turns prompts into specs and then code, docs, and tests, with steering files for persistent project knowledge.
- Anthropic documents project instructions and machine-local auto memory loaded into Claude Code sessions; Cursor documents project rules and automatically generated memories as persistent prompt context.
- Augment describes its Context Engine as maintaining live understanding across repositories, services, history, docs, tickets, and decisions.

These notes establish comparison candidates only. Delivery must refresh and cite the current primary sources in public copy.
