---
title: "Story Detail + Intake Scrubber Backing — Dependency Mapper, Duplicate Intake Scrubber"
slug: story-detail-and-intake-scrubber-backing
type: feature
status: planning
domain: pm
priority: high
size: small
created: 2026-07-17
tags: [pm, story-detail, intake, scrub, wave-1]
relations:
  - target: pm-pack-completion
    kind: parent
  - target: pm-doctrine-and-skill-backfill
    kind: depends-on
  - target: remaining-roles-scrubbers-and-launch
    kind: conflicts-with
---

# Story Detail + Intake Scrubber Backing — Dependency Mapper, Duplicate Intake Scrubber

Backs the Story Detail "Show dependencies" and Intake Funnel "Cluster recent"
buttons.

## Scope (stub — materialize with `/design`)

- Agents: `dependency-mapper`, `duplicate-intake-scrubber`.
- Commands: `commands/scrub` — the **intake** concern (scaffolds the shared
  `domains/pm/commands/scrub.md` that Child #11 later extends).

## Depends on

- #1 (`pm-doctrine-and-skill-backfill`) — doctrine spine + AGENTS.md routing.

## Seams

- Reciprocal `conflicts-with` with #11 (`remaining-roles-scrubbers-and-launch`):
  both edit `domains/pm/commands/scrub.md`. This child scaffolds it (intake
  concern); #11 extends it (roadmap + stories concerns). See the initiative's
  "Intake scrubber ↔ launch roles same-file seam".
