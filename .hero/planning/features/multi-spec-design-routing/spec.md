---
title: Multi-Spec Design Routing — Nudge from /design ×N to /compose
slug: multi-spec-design-routing
type: feature
domain: engineering
status: planning
size: small
priority: P2
tags: [design, compose, routing, ambient-context]
created: 2026-06-01
relations:
  - target: roadmap-shape
    kind: parent
---

## Goal

When the user runs `/design` against work that's clearly part of a
larger group — heuristic: 2+ topically related specs surfaced in the
current session — the design workflow surfaces a routing nudge
suggesting `/compose` to scaffold a parent initiative before
proceeding with another flat feature spec. The nudge is a soft
recommendation, not a block; the user can decline and ship the flat
spec.

## Context

This is the routing-nudge child in the [`roadmap-shape`](../../initiatives/roadmap-shape/spec.md)
initiative. It catches the "I'm `/design`-ing four related things"
anti-pattern at the *second or third* `/design` call — the moment
where promotion to an initiative is still cheap. By the fourth flat
spec, the user has already paid the orphan-grouping cost.

The nudge is *routing-only*: it does not redesign `/compose`'s UX, it
does not introduce new sequencing primitives, and it does not change
how the design workflow scaffolds individual specs. It only suggests
the better entry point at the right moment.

## Kickoff

When the user fires `/design` and 2+ related specs are already in
the session, nudge them toward `/compose` instead of scaffolding
another flat feature.

**Status:** planning — stub only; needs full `/design` pass.

**Pick up at:** run `/design multi-spec-design-routing` once the
"multiple related specs" phrasing is locked in `roadmap-review`'s
skill. The nudge text quotes from that canonical sentence.

→ `/design multi-spec-design-routing`

**Files:** `.claude/commands/design.md`, `.claude/agents/feature-delivery-lead.md`
**Skip:** /compose UX redesign, new heuristics beyond the 2+ related-specs trigger.

## Scope-creep watch

**Routing nudge only.** No `/compose` UX redesign, no new heuristics
beyond "2+ related specs surfaced in conversation," no new spec types,
no auto-promotion (the system suggests; the user runs the command).
The heuristic itself stays simple — "topically related specs in the
current session" — and does not grow into a learned classifier or a
cross-session model in v1. If the design conversation reaches for
"and we could also…," push back.

## Notes for design

Decisions already made in the composition session that `/design`
should honor:

- **Single trigger heuristic:** 2+ related specs surfaced in the
  current session. Not a learned model, not a cross-session signal,
  not a tracker query. In-session and simple.
- **Nudge, not block.** The user can decline and the flat spec
  scaffolds normally. The nudge surfaces; it does not gate.
- **Shared phrasing with child #1.** The "multiple related specs"
  detection logic in `roadmap-reviewer` and the routing nudge here
  describe the same condition. Use the canonical sentence #1
  establishes in the `spec-sizing` skill (or wherever #1 lands the
  shared note). Do not invent new wording.
- **Nudge text points at `/compose`.** Explicit command in the nudge,
  not prose. Match the style of the size-promotion nudges in the
  `spec-sizing` skill — concrete, runnable, no hedging.
- **No interaction with `roadmap-review-ambient-surfacing` (#2).**
  This child fires *inline during `/design`*; #2 fires *between
  sessions*. Same conceptual signal, different surfaces. They should
  not echo each other in a single session.
- **No new MCP tools.** Detection happens inside the `/design`
  command flow; the agent has the session context it needs.
