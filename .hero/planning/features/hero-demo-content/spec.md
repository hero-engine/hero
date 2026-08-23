---
title: Hero Interactive Install Terminal Demo
slug: hero-demo-content
type: feature
status: planning
priority: P2
tags: [marketing, demo, install, terminal, animation]
created: 2026-04-25
relations:
  - target: hero-positioning
    kind: depends-on
horizon: someday
smoke: deferred
---

## Goal

Produce one short, lightweight terminal animation showing Hero's real interactive project-install flow. The output may be generated or staged, but it must be visibly labeled illustrative and must use valid Hero commands and prompts.

## Problem

The public site explains Hero in words but does not quickly show how little effort it takes to add Hero to a project. A concise install animation can make the onboarding path tangible without pretending to prove product outcomes.

## Deliverable

- A 6–12 second looping GIF, WebM, or CSS terminal animation.
- Show the real interactive `hero install` flow selecting the normal project setup and ending at a clear ready state.
- Use synthetic timing and generated terminal output when useful.
- Label the animation “Illustrative install flow” in the visible page or caption.
- Provide a reduced-motion static fallback and accessible text transcript.
- Keep the artifact small enough not to delay the first meaningful page render.

## Acceptance Criteria

- **AC-1:** THE SYSTEM SHALL show only commands, questions, choices, and outcomes supported by the current interactive install flow.
- **AC-2:** IF output or timing is generated, staged, abbreviated, or edited THEN THE SYSTEM SHALL label the presentation illustrative rather than recorded evidence.
- **AC-3:** THE SYSTEM SHALL provide an accessible static fallback and respect reduced-motion preferences.
- **AC-4:** THE SYSTEM SHALL add no release, license, or public-visibility dependency; this asset may ship later without blocking v0.34.

## Boundaries

- No cross-tool continuity proof, cold-resume fixture, external harness authentication, or evidence claim.
- No full screencast library, feature-by-feature GIF catalog, social campaign, or video hosting dependency.
- No fabricated product capability or generated output presented as a real transcript.
