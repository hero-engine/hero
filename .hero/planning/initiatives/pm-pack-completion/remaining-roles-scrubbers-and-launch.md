---
title: "Remaining Roles, Scrubbers, and Launch/GTM"
slug: remaining-roles-scrubbers-and-launch
type: feature
status: planning
domain: pm
priority: low
size: medium
created: 2026-07-17
tags: [pm, roles, scrub, launch, coverage, wave-3]
relations:
  - target: pm-pack-completion
    kind: parent
  - target: pm-doctrine-and-skill-backfill
    kind: depends-on
  - target: story-detail-and-intake-scrubber-backing
    kind: depends-on
  - target: story-detail-and-intake-scrubber-backing
    kind: conflicts-with
---

# Remaining Roles, Scrubbers, and Launch/GTM

Rounds out the remaining designed P1/P2 roles plus launch/GTM.

## Scope (stub — materialize with `/design`)

- Agents: `epic-framer`, `risk-curator`, `portfolio-curator`,
  `discovery-reviewer`, `stale-roadmap-scrubber`, `ambiguous-story-scrubber`.
- Extend `commands/scrub` — the **roadmap + stories** concerns.
- `skills/launch-gtm-tiering` (tier 1/2/3, phased checklist) + `/launch` command.

## Depends on

- #1 (`pm-doctrine-and-skill-backfill`) — doctrine spine + AGENTS.md routing.
- #5 (`story-detail-and-intake-scrubber-backing`) — #5 scaffolds
  `domains/pm/commands/scrub.md` (intake concern); this child extends it.

## Seams

- Reciprocal `conflicts-with` with #5: both edit `domains/pm/commands/scrub.md`.
  The `depends-on` orders them (#5 first); the `conflicts-with` additionally
  guards against concurrent edits if the ordering slips. See the initiative's
  "Intake scrubber ↔ launch roles same-file seam".
