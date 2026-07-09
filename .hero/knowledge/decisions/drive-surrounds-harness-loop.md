---
title: "Drive surrounds the harness /goal loop — Hero does not build its own loop engine"
type: decision
status: accepted
created: 2026-06-27
related_specs:
  - drive-autonomous-initiative-execution
  - hero-goal-command
  - needs-me-predicate
---

# Drive surrounds the harness /goal loop

## Decision

For autonomous initiative execution ("Drive"), Hero does **not** implement a
turn-after-turn loop driver or a completion evaluator. Both already ship as
the harness `/goal` command (Claude Code v2.1.139, Codex GA) — the
productized "Ralph loop." Hero's role is to (a) supply the objective via an
initiative `## Goal` section and (b) be the authoritative per-turn judge,
`hero goal <init> --check`, that the harness loop consults via a Stop hook.

Corollaries, also settled:
- The autonomy boundary is a **deterministic predicate** (`needs_me()`), not
  a babysitter agent — decisions must be inspectable and reproducible.
- The user-facing verb is **`/drive`**, not `/autopilot` ("autopilot" is a
  natural-language synonym only). `/deliver` is **not** overloaded: deliver =
  one spec/step; drive = whole initiative/autonomous.

## Context

The loop pattern (Geoffrey Huntley's Ralph loop, mid-2025) is now a harness
primitive, so rebuilding it Hero-side is wasted duplication. Practitioner
evidence is consistent: agentic loops live or die on (1) spec quality and
(2) a real completion gate — both already Hero's strengths (rigorous specs,
deterministic `hero verify`). The harness's own `/goal` evaluator judges
completion from the *transcript* (a vibe-check); Hero's `verify` is an
authoritative gate. So the highest-leverage Hero contribution is the parts
the harness can't do: a structured objective, an authoritative gate, and a
human-boundary predicate it can pause at and resume from.

## Why this matters

It scopes the whole initiative *down*: no loop engine, no evaluator, no
daemon. It also resists the recurring temptation to "just build our own
autopilot," which would re-implement harness infrastructure and couple Hero
to one execution model instead of riding whatever loop the harness ships.

See [[verify-gates-the-flip]] — the same `hero verify` gate that flips status
is the "done" half of Drive's stop-condition.
