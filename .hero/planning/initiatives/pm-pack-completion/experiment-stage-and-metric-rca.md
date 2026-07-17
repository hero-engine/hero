---
title: "Experiment Stage + Metric RCA — Experiment Designer, Metrics Analyst"
slug: experiment-stage-and-metric-rca
type: feature
status: planning
domain: pm
priority: high
size: medium
created: 2026-07-17
tags: [pm, experiment, metrics, differentiation, wave-2]
relations:
  - target: pm-pack-completion
    kind: parent
  - target: pm-doctrine-and-skill-backfill
    kind: depends-on
  - target: pm-doctrine-and-skill-backfill
    kind: conflicts-with
  - target: adversarial-critics-bundle
    kind: conflicts-with
---

# Experiment Stage + Metric RCA — Experiment Designer, Metrics Analyst

A whole stage absent from the original design. Un-dangles the shipped
`/metrics` command.

## Scope (stub — materialize with `/design`)

- `experiment-designer` agent + `skills/experiment-design` + `/experiment`
  command (pre-registration, MDE, guardrails, SRM, no early-stopping).
- `metrics-analyst` agent + `skills/metric-rca` (metric-tree decomposition,
  drift taxonomy — "why did the metric move").

## Depends on

- #1 (`pm-doctrine-and-skill-backfill`) — doctrine spine + AGENTS.md routing.

## Seams

- Reciprocal `conflicts-with` with #1: registers net-new agent routes in
  `domains/pm/AGENTS.md` (marked Wave-2 region).
- Reciprocal `conflicts-with` with #6 (`adversarial-critics-bundle`): defines
  the experiment brief format that #6's `experiment-readout-reviewer` asserts
  against. See open question (c).
