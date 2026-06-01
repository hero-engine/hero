---
title: Roadmap-Review Ambient Surfacing — NEXT.md, Pulse, and Pre-Flight Hooks
slug: roadmap-review-ambient-surfacing
type: feature
domain: engineering
status: planning
size: small
priority: P1
tags: [roadmap, ambient-context, next-md, pulse, delivery-lead]
created: 2026-06-01
relations:
  - target: roadmap-shape
    kind: parent
  - target: roadmap-review
    kind: depends-on
---

## Goal

Wire `/roadmap-review` findings into three existing context surfaces —
the NEXT.md projection, `hero_pulse` / `hero_kickoff` output, and the
delivery-lead pre-flight — so users hear about roadmap-shape drift at
natural moments without needing to remember to run the command. Surface
messages stay lens-agnostic ("size drift," "shape concern") so the
format extends as future lenses come online.

## Context

This is the surfacing-layer child in the [`roadmap-shape`](../../initiatives/roadmap-shape/spec.md)
initiative. It ships last in the recommended sequence because it depends
on a locked command name (`/roadmap-review`) and agent name
(`roadmap-reviewer`) from child #1, and benefits from a stable
`hero size --check` row format from child #3.

Without this child, the detection work in #1 is largely wasted — users
won't run a command they don't know to run. This child closes the loop
between "the system knows there's drift" and "the user is told about
it at a moment where action is natural."

## Kickoff

Wires `/roadmap-review` findings into NEXT.md projection, hero_pulse /
hero_kickoff, and delivery-lead pre-flight so drift surfaces ambiently
without the user running the command.

**Status:** planning — stub only; needs full `/design` pass.

**Pick up at:** run `/design roadmap-review-ambient-surfacing` once
`roadmap-review` is delivered. Confirm command/agent names are stable
before referencing them in surface text.

→ `/design roadmap-review-ambient-surfacing`

**Files:** `internal/next/projection.go`, `internal/mcp/pulse.go`, `.claude/agents/feature-delivery-lead.md`
**Skip:** /prime, /resume, hero status, status bar — out of scope for v1; tune noise threshold first.

## Scope-creep watch

Wire into exactly three surfaces: **NEXT.md projection, hero_pulse /
hero_kickoff, and delivery-lead pre-flight.** Resist `/prime`,
`/resume`, `hero status`, the status bar, and any new surface that
emerges mid-design. Nudge fatigue is the biggest risk to the initiative
— every additional surface multiplies the chance the user mutes the
channel, which makes the detection in #1 wasted work. Three surfaces,
tuned for noise, before any expansion.

## Notes for design

Decisions already made in the composition session that `/design`
should honor:

- **Exactly three surfaces in v1.** NEXT.md projection, hero_pulse /
  hero_kickoff output, delivery-lead pre-flight. Not more.
- **Surface messages are lens-agnostic.** Say "size drift" or "shape
  concern," not "sizing-lens drift." The format must extend as new
  lenses come online without rewriting messaging.
- **Count-only by default; click-through for detail.** "3 specs have
  shape concerns — run `/roadmap-review` for details" is the default
  surface. Don't echo row excerpts in the surface text; that couples
  this child to #3's row format and inflates the surface.
- **Surface only on change.** Track the last-surfaced finding set and
  only re-fire when findings change. A user who saw "3 concerns"
  yesterday and the same "3 concerns" today doesn't need to hear it
  twice.
- **Soft dependency on child #3.** If this surface echoes any part of
  `hero size --check` row text, #3's CLI shape becomes load-bearing
  here. Recommended mitigation: surface counts only, so this child is
  decoupled from #3's format.
- **Adjacent to child #4's routing logic.** Both detect "multiple
  related specs." Shared phrasing pass, not a sequencing constraint —
  but use the same canonical sentence #1 establishes.
- **Nudge fatigue is the dominant risk.** Tune the surface threshold
  against real `/roadmap-review` usage from #1; do not ship until at
  least a one-week dogfood window confirms the cadence is sustainable.
- **No new MCP tools.** Use the existing surfaces. If a new tool is
  needed, that's a scope-creep signal.
