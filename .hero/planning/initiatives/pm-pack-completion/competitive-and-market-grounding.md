---
title: "Competitive + Market Grounding — Retrieval-Only Competitive Analyst"
slug: competitive-and-market-grounding
type: feature
status: planning
domain: pm
priority: medium
size: small
created: 2026-07-17
tags: [pm, competitive, market-sizing, differentiation, wave-2]
relations:
  - target: pm-pack-completion
    kind: parent
  - target: pm-doctrine-and-skill-backfill
    kind: depends-on
  - target: pm-doctrine-and-skill-backfill
    kind: conflicts-with
---

# Competitive + Market Grounding — Retrieval-Only Competitive Analyst

## Scope (stub — materialize with `/design`)

- `competitive-analyst` agent — **retrieval-augmented, never model-memory**
  (teardown + feature matrix + positioning).
- Sharpen `product-strategist`.
- Skills: `opportunity-assessment` (Cagan 10-Q, single-challengeable-assumption),
  `market-sizing` (TAM/SAM/SOM).

## Depends on

- #1 (`pm-doctrine-and-skill-backfill`) — corpus-grounding doctrine is the whole
  point of the retrieval-only constraint.

## Seams

- Reciprocal `conflicts-with` with #1: registers the `competitive-analyst` route
  in `domains/pm/AGENTS.md` (marked Wave-2 region).
