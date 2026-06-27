---
title: "`needs_me()` predicate — the autonomy boundary + `autonomy:` policy field"
slug: needs-me-predicate
type: feature
status: planning
priority: high
horizon: now
tags: [drive, needs-me, predicate, autonomy, safety, boundary]
created: 2026-06-27
relations:
  - target: drive-autonomous-initiative-execution
    kind: parent
  - target: hero-idea-primitive-core
    kind: relates-to
---

# `needs_me()` predicate — the autonomy boundary + `autonomy:` policy field

## Goal

Ship the single conservative predicate that decides, at every transition in
a Drive run, **proceed autonomously or pause for the human** — plus the
per-initiative `autonomy:` policy field that sets how aggressively it
proceeds. This is the hard part of the whole initiative: distinguishing a
decision only the human can make from a step the agent can take that the
human would merely rubber-stamp.

## Kickoff

Build `needs_me(spec, ctx) -> Decision` as a shared, deterministic Go
predicate — a sibling, on a different axis, to the existing
`is_committed_work()` predicate from the intake primitive (see
[internal/handoff](../../../../../internal/handoff) and
[hero-idea-primitive-core](../../../features/hero-idea-primitive-core/spec.md)). It
returns `proceed` or `pause{reason, category}`, and is **conservative:
unknown → pause**. Add an `autonomy: supervised | guided | autonomous`
frontmatter field (default `supervised` = today's behavior) that tunes the
thresholds. Encode the pause taxonomy and the hard-pause guardrails below.
No loop logic, no `/goal` wiring — those are in `hero-goal-command`. This
predicate is pure and unit-testable in isolation.

## Problem

The loop driver (`/goal`) and resume (disk state) already exist; the only
reason the user hand-approves every spec boundary is that nothing decides
*whether this boundary needs a human*. A naive "just keep going" is unsafe
(autonomous wrong turns compound); "always ask" is what we have today. We
need a predicate that is right often enough to remove the rubber-stamp
stops while never silently taking a decision that was the human's to make.

## Design

### Signature

```go
type AutonomyMode int // Supervised, Guided, Autonomous

type Decision struct {
    Proceed  bool
    Category PauseCategory // why, when !Proceed
    Reason   string        // human-readable, fed to the pause question
}

// needs_me reports whether advancing past `at` needs the human, given the
// run's autonomy mode and observable context. Conservative: unknown→pause.
func NeedsMe(at *spec.Spec, ctx RunContext, mode AutonomyMode) Decision
```

`RunContext` carries the cheap, already-computed signals: the verify
verdict for the just-finished spec, `hero score` for the candidate next
spec, the blocked set, the dependency graph, and a classification of the
pending action (does it touch irreversible/outward-facing surfaces).

### Pause taxonomy (PauseCategory)

| Category | Pause when… | Signal source |
|---|---|---|
| `DesignFork` | the next spec's design surfaced ≥2 viable approaches with material tradeoffs (a `/decide` moment) | design output / spec markers |
| `Underspecified` | candidate next spec's `hero score` below threshold | `hero score` |
| `Irreversible` | pending action touches migration/delete/deploy/external-send | action classifier (hard rule) |
| `VerifyStuck` | `hero verify` still FAILs after N rework passes | verify history |
| `Blocked` | next work is dependency/externally blocked | `hero blocked` |
| `AmbiguousPick` | queue near-tie between candidate next specs of different intent | queue ranking |

### Proceed-silently set (no pause)

- next ready child, deps satisfied, score ≥ threshold
- cold-audit → rework → re-verify cycles *within* a spec
- `hero verify` PASS → mark complete → advance
- well-specified mechanical delivery

### `autonomy:` policy field

Per-initiative frontmatter knob consumed by `NeedsMe`:

- `supervised` (default) — pause at every spec boundary; today's behavior.
- `guided` — pause on every taxonomy category, proceed only on the
  proceed-silently set.
- `autonomous` — proceed on everything in the proceed-silently set AND on
  categories the learning layer has promoted (see
  [drive-autonomy-learning](../drive-autonomy-learning/spec.md)); **but the
  hard-pause guardrails below are never relaxed.**

### Hard-pause guardrails (mode-independent — never relaxed)

1. `Irreversible` actions ALWAYS pause, even in `autonomous`.
2. Always pause at initiative boundaries and after N consecutive specs
   (hard cap; N configurable, conservative default).
3. Unknown / unclassifiable transition → pause.

`NeedsMe` is pure given `RunContext`; all I/O (running score, reading
blocked set) happens in the caller and is passed in, so the predicate is
trivially testable and reproducible.

## Acceptance Criteria

- WHEN advancing past a spec whose pending action is classified
  irreversible/outward-facing, THE SYSTEM SHALL return `pause{Irreversible}`
  regardless of `autonomy` mode.
- WHILE `autonomy: supervised`, THE SYSTEM SHALL pause at every spec
  boundary (behavioral parity with today).
- WHEN the candidate next spec scores below the readiness threshold, THE
  SYSTEM SHALL return `pause{Underspecified}`.
- WHEN `hero verify` has failed N times on the current spec, THE SYSTEM
  SHALL return `pause{VerifyStuck}` rather than retry indefinitely.
- IF a transition cannot be classified, THEN THE SYSTEM SHALL pause
  (conservative default).
- WHERE the transition is a verify-PASS → advance to a ready, sufficiently
  scored next child, THE SYSTEM SHALL proceed without pausing (in `guided`
  or `autonomous`).

## Test Plan

- Table-driven unit tests: one row per PauseCategory × autonomy mode,
  asserting Proceed/Category. The irreversible-always-pauses row runs
  across all three modes.
- Property test: `NeedsMe` with an empty/unknown `RunContext` always pauses.
- Parity test: `supervised` mode pauses on every boundary fixture.
- Hard-cap test: after N consecutive proceeds, the (N+1)th pauses even when
  every other signal says proceed.

## Risks

- **Miscalibration** (R1 from the initiative). The taxonomy thresholds will
  be wrong at first. Mitigation: ship conservative, make thresholds config,
  and let `drive-autonomy-learning` tune per-category from real outcomes.
- **Action classification accuracy** — deciding "is this irreversible?" is
  itself a judgment. Mitigation: start from a conservative static
  allow/deny of action shapes (shell verbs, file ops, network/deploy
  markers); unknown action shape = irreversible = pause.
- **Predicate drift from `is_committed_work()`** — keep them as distinct
  axes (classification vs. autonomy); do not fold one into the other.
