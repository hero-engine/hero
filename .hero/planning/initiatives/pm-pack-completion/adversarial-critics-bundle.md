---
title: "Adversarial Critics Bundle — Drift, Prioritization, Doc, and Readout Critics"
slug: adversarial-critics-bundle
type: feature
status: planning
domain: pm
priority: high
size: large
created: 2026-07-17
tags: [pm, critics, differentiation, wave-2]
relations:
  - target: pm-pack-completion
    kind: parent
  - target: pm-doctrine-and-skill-backfill
    kind: depends-on
  - target: pm-doctrine-and-skill-backfill
    kind: conflicts-with
  - target: experiment-stage-and-metric-rca
    kind: conflicts-with
---

# Adversarial Critics Bundle — Drift, Prioritization, Doc, and Readout Critics

**THE differentiation thesis** — critics over generators. Kept as one bundle so
the doctrine (adversarial, corpus-grounded, decision-gated) is authored
consistently across all four critics rather than drifting per-agent.

## Scope (stub — materialize with `/design`)

- `roadmap-reviewer` authored directly as a drift-critic + `skills/outcome-drift`
  (outcome/output/input ratio, stale-item flagging).
- `prioritization-challenger` agent + `skills/evidence-forcing`
  ("confidence needs named evidence or it's 50%").
- Sharpen `pm-reviewer` into `pm-critic` (premortem, "5 reasons this won't work").
- `experiment-readout-reviewer` agent (adversarial readout: SRM, no
  early-stopping, guardrails).

## Depends on

- #1 (`pm-doctrine-and-skill-backfill`) — critics must load `pm-agent-doctrine`.

## Seams

- Reciprocal `conflicts-with` with #1: registers net-new critic routes in
  `domains/pm/AGENTS.md` (marked Wave-2 region).
- Reciprocal `conflicts-with` with #7 (`experiment-stage-and-metric-rca`):
  `experiment-readout-reviewer` (here) reviews the experiment brief that #7
  defines — shared experiment-artifact contract. See open question (c) re:
  hardening this to a `depends-on`.
