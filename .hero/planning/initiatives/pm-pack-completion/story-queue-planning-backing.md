---
title: "Story Queue Backing — Capacity + Cycle Planning Agents, Skills, Commands"
slug: story-queue-planning-backing
type: feature
status: planning
domain: pm
priority: high
size: medium
created: 2026-07-17
tags: [pm, planning, story-queue, wave-1]
relations:
  - target: pm-pack-completion
    kind: parent
  - target: pm-doctrine-and-skill-backfill
    kind: depends-on
  - target: pm-foundation-delivery
    kind: depends-on
---

# Story Queue Backing — Capacity + Cycle Planning Agents, Skills, Commands

Backs the Story Queue hero-code view, which has **zero** backing agents today.

## Scope (stub — materialize with `/design`)

- Agents: `capacity-planner` (velocity cut-line), `cycle-planner`
  (cycle-fit marker; one preset-adaptive agent for sprint/cycle/iteration).
- Skills: `capacity-planning`, `iteration-planning`, `shape-up-cadence`.
- Commands: `/capacity`, `/plan-cycle`, `/plan-sprint`, `/plan-iteration`.

## Depends on

- #1 (`pm-doctrine-and-skill-backfill`) — doctrine spine + AGENTS.md routing.
- `pm-foundation-delivery` (delivered) — planning operates over the canonical
  `feature` type rendered as "Story" via the `agile-scrum` vocabulary + `scrum`
  methodology profile, all already on disk in `core/spec-types/`,
  `core/vocabularies/`, `core/methodologies/`. No new spec type is authored here.
