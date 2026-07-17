---
title: "PRD Editor + Comms Backing — Pitch Author, Stakeholder Communicator"
slug: prd-editor-comms-backing
type: feature
status: planning
domain: pm
priority: high
size: medium
created: 2026-07-17
tags: [pm, prd, comms, wave-1]
relations:
  - target: pm-pack-completion
    kind: parent
  - target: pm-doctrine-and-skill-backfill
    kind: depends-on
  - target: pm-doctrine-and-skill-backfill
    kind: conflicts-with
---

# PRD Editor + Comms Backing — Pitch Author, Stakeholder Communicator

Un-dangles the shipped `/pitch` and `/release-notes` commands and backs the
PRD Editor view's "Convert to pitch" / "Summarize for standup" buttons.

## Scope (stub — materialize with `/design`)

- Agents: `pitch-author` (split from `prd-author`), `stakeholder-communicator`.
- Skills: `stakeholder-communication`, `release-notes-writing`.
- Commands: `/standup`, `/interview`.

## Depends on

- #1 (`pm-doctrine-and-skill-backfill`) — doctrine spine + AGENTS.md routing.

## Seams

- Reciprocal `conflicts-with` with #1: registers the two net-new agents' routes
  in `domains/pm/AGENTS.md` (marked Wave-2 region only). See the initiative's
  "AGENTS.md routing-table hotspot".
