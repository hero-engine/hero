---
title: Capture Intent at Altitude — Don't Spec-ify Every Change
type: decision
status: proposed
created: 2026-06-29
tags: [intake, workflow, capture, spec-discipline, decision, architecture]
relations:
  - target: intake-capture-loop
    kind: decided-in
  - target: hero-idea-primitive-core
    kind: related
---

# Capture Intent at Altitude — Don't Spec-ify Every Change

## Decision

Hero does **not** route every code change through `spec → deliver`. Forcing all
changes into full specs has two failure modes that outweigh the capture benefit:
(1) **friction** — the value of a small ask is its speed; full ceremony makes
users route around Hero entirely, losing the capture; (2) **spec spam** — a spec
titled "fix typo" captures nothing and degrades the signal of `hero_search` /
`hero_why`. The cleanliness of the spec corpus is itself a feature.

The thing worth capturing is **intent and rationale**, at the right altitude —
not the mechanical change. The mechanism is the `intake` pre-commitment primitive
(see [[hero-idea-primitive-core]]): intent-bearing loose asks are captured as
intakes **retroactively and only when they clear a threshold** — the same
silent, non-gating discipline `auto_capture` already uses for knowledge, not a
ceremony on every edit. The human is the **manual promote gate** (the human is
the altitude classifier at triage time, so no up-front auto-classifier is built).
Captured intakes stay out of all committed-work rollups until promoted, so
capturing freely never pollutes in-flight views. Commit↔intake provenance is a
deferred nice-to-have, not part of the lightweight core.

## Why

The mission is to capture "the stuff nobody told the next session." Loose inline
asks are exactly where intent evaporates today. But capture must be near-zero
friction and must not drown the corpus — so capture freely, gate manually, and
exclude un-promoted captures from rollups.

## How to apply

When designing capture/workflow features: prefer **retroactive or capture-then-
edit + manual triage** over up-front gating; default to *capture freely, promote
deliberately*; reject "every X becomes a spec" framings in the spec's Goal the
way [[intake-capture-loop]] does. Mirrors the type-proliferation discipline in
[[hero-idea-primitive-core]] (don't add a primitive when an existing one fits).
