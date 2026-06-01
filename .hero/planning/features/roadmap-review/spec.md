---
title: Roadmap Review — Command, Agent, and Skill for On-Demand Shape Detection
slug: roadmap-review
type: feature
domain: engineering
status: planning
size: medium
priority: P1
tags: [roadmap, sizing, command, agent, skill]
created: 2026-06-01
relations:
  - target: roadmap-shape
    kind: parent
---

## Goal

Introduce a `/roadmap-review` slash command, a `roadmap-reviewer` agent,
and a `roadmap-review` skill that together perform an on-demand pass
over the current planning corpus to detect roadmap-shape drift —
specifically *sizing drift* in v1 — and emit a triage list the user
can act on. The capability is structured around a Lenses extension
model so future analytical perspectives (horizons, releases, sprint-shape)
can be added by extending a list, not redesigning the surface.

## Context

This is the main capability in the [`roadmap-shape`](../../initiatives/roadmap-shape/spec.md)
initiative. It must land first because every other child in the
initiative references its command name, agent name, or skill
structure. Without this, the surfacing layer (#2) has nothing to
project, and the routing nudge (#4) has nowhere to point.

The capability is intentionally sizing-only in v1. The skill's Lenses
section names horizons / releases / sprint-shape as placeholders so
the structure is future-proof, but the agent explicitly refuses to
act on non-sizing drift in v1 and the command surfaces nothing for
other lenses.

## Kickoff

`/roadmap-review` command + `roadmap-reviewer` agent + skill that
detects sizing drift across the planning corpus and emits a triage
list. Sizing-only lens in v1; Lenses scaffolding for the rest.

**Status:** planning — stub only; needs full `/design` pass.

**Pick up at:** run `/design roadmap-review` to flesh out acceptance
criteria, agent prompt, skill layout, and the Lenses scaffolding model.
Lock the command name and agent name first — three sibling specs
depend on them.

→ `/design roadmap-review`

**Files:** `.claude/commands/`, `.claude/agents/`, `.claude/skills/spec-sizing/SKILL.md`, `internal/cli/size.go`
**Skip:** implementing horizons / releases / sprint-shape lenses in v1 — they're named placeholders, not behavior.

## Scope-creep watch

Sizing lens only in v1. Horizons, releases, and sprint-shape are
**scaffolding placeholders** in the skill's Lenses section, NOT
implemented behavior. The `roadmap-reviewer` agent must explicitly
refuse to act on non-sizing drift in v1 — including pushing back if
the user asks. Adding even one more lens doubles the surface area
the initiative has to support, and v1 hasn't proven the model yet.

## Notes for design

Decisions already made in the composition session that `/design`
should honor:

- **Command name is locked as `/roadmap-review`.** Not `/triage`, not
  `/shape`, not `/review` (taken). Three sibling specs reference this
  name; do not change it without coordinating across the initiative.
- **Agent name is locked as `roadmap-reviewer`.** Same rationale.
- **Skill name is locked as `roadmap-review`.** Lives alongside
  `spec-sizing` in `.claude/skills/`.
- **Sizing-only lens in v1.** Hard scope boundary.
- **Lenses scaffolding model.** The skill's `## Lenses` section names
  the future lenses (horizons, releases, sprint-shape) as placeholders
  so the structure can extend by appending list items rather than
  reworking the surface.
- **One sentence on `hero check` vs `/roadmap-review`** belongs in
  this spec and in the user-facing docs: `hero check` is *workspace
  hygiene* (stale specs, missing fields, convention drift);
  `/roadmap-review` is *roadmap shape* (oversized specs,
  related-but-orphaned specs, unfinished composition). Don't let
  them collide.
- **Shared "multiple related specs" phrasing** with child #4
  (`multi-spec-design-routing`). Capture once — either as a new
  shared note or appended to the `spec-sizing` skill — and quote
  from both surfaces. Don't reinvent.
- **Triage output shape.** The command should produce a triage list,
  not a wall of prose. Each finding: which spec, what drift, suggested
  next-step command. Aligns with the inline-next-step format being
  established in child #3.
- **No tracker writes in v1.** The command reads the local corpus and
  surfaces findings. It does not push labels, comments, or tickets to
  the tracker. Tracker integration is future-lens work.
