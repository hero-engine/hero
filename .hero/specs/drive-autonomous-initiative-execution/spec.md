---
title: "Drive — Autonomous Initiative Execution with a Human-Boundary Predicate"
slug: drive-autonomous-initiative-execution
type: initiative
status: completed
priority: high
horizon: now
tags: [drive, autonomy, goal, loop, initiative, needs-me, pause-resume, harness]
created: 2026-06-27
completed_at: 2026-06-28T00:13:51Z
---

# Drive — Autonomous Initiative Execution with a Human-Boundary Predicate

## Goal

Let a user say **`/drive <initiative>`** (natural-language synonyms:
*"autopilot this initiative"*, *"put X on autopilot"*) and have Hero run
the initiative's child specs autonomously — design, deliver, cold-audit,
rework, verify, advance to the next — **pausing only when a decision
genuinely needs the human, and resuming cleanly afterward.**

The bar: an initiative whose children are well-specified should run to
completion with the human touched only at real forks (design tradeoffs,
ambiguity, irreversible actions, stuck gates) — never at the routine
"yep, do the next one" boundaries that the user hand-approves today.

## Kickoff

Building "Drive": autonomous execution of a whole initiative's child
specs. The loop driver is the **harness's `/goal`** (Claude Code/Codex) —
Hero does NOT build a loop engine. Hero supplies (a) the objective via a
new initiative `## Goal` section and (b) an authoritative per-turn judge,
`hero goal <init> --check`, wired through a Claude Code **Stop hook**. The
judge ANDs `hero verify` over the children with a new conservative
`needs_me()` predicate that decides proceed-vs-pause-for-human. Pauses
write a *question* to NEXT.md and resume from disk. Start with child
`initiative-goal-section`, then `needs-me-predicate`. Read this initiative
spec and the six children for the full picture and the settled naming
(`/drive`, not `/autopilot`; `/deliver` is NOT overloaded).

## Problem

The user already runs this loop by hand: make an initiative, design and
deliver its specs, repeat — Hero drives the design, acceptance criteria,
cold audits, rework, and verification. But **it stops at the end of every
spec**, and the user "mostly just says yes, do the next." Those stops are
pure friction: the human is rubber-stamping, not deciding.

Three observations frame the work:

1. **The loop is already 2/3 built.** The *driver* ships as the harness
   `/goal` command (the productized "Ralph loop" — same shape in Claude
   Code v2.1.139 and Codex). *Resume* is nearly free because Hero already
   projects durable state to disk (`QUEUE.md`, `NEXT.md`, completion
   ledger, spec status) — exactly the discipline a context-reset loop
   needs. The missing third is the **stop/continue decision**.

2. **Loops live or die on two things — both already Hero's strengths.**
   Ralph-loop practitioners are unanimous: a loop turns a *good spec* into
   hours of correct autonomous work and a *vague spec* into confident
   garbage at scale, and it needs a *real completion gate*, not a
   transcript vibe-check. Hero already has rigorous specs and a
   deterministic 4-gate `hero verify`. Drive makes the loop trustworthy on
   real engineering work instead of greenfield demos.

3. **The hard part is the human boundary, not the loop.** Distinguishing
   "a decision only the human can make" from "a step the agent can take
   and the human would just approve" is the entire ballgame. This is a new
   predicate — `needs_me()` — a sibling on a different axis to the
   `is_committed_work()` predicate Hero already ships from the intake
   primitive ([hero-idea-primitive-core](../../features/hero-idea-primitive-core/spec.md)).

## Mission Fit

> *Does this make the next agent session start smarter than the last one
> ended, and raise the floor for everyone?*

Yes, on the substance not just the throughput:

- **Pause-as-question + resume** writes a precise, self-contained question
  to disk — so a fresh session (or another machine) resumes exactly where
  autonomy stopped, carrying the full "what I tried / what I need" state.
  That is context compounding across sessions.
- **Rubber-stamp learning** captures each user's approval patterns and
  raises the floor: over time the system stops asking what this user
  always says yes to, so a junior running Drive inherits the senior's
  "this is fine to auto-proceed" judgment without being told.

## Architecture — The Settled Shape

Surface at top, engine at bottom. Hero surrounds the harness loop; it does
not rebuild it.

| Layer | Piece | Child spec |
|---|---|---|
| **Trigger** | `/drive <initiative>` command + NL routing ("autopilot X"); graceful `/deliver`-on-initiative fallback | `drive-command-routing` |
| **Driver** | harness `/goal` runs the turn-after-turn loop | *exists (harness)* |
| **Opener** | initiative `## Goal` section (parallel to spec `## Kickoff`) — human objective AND machine stop-condition, pasted into `/goal`; queue renders it | `initiative-goal-section` |
| **Per-turn judge** | `hero goal <init>` (CLI+MCP): `emit` the condition, `--check` returns `continue \| pause \| done`; Stop-hook contract calls it each turn | `hero-goal-command` |
| **Gate** | `hero verify` over each child = the "done" half | *exists* |
| **Boundary** | `needs_me()` predicate + `autonomy:` policy field | `needs-me-predicate` |
| **Pause/resume** | pause writes a *question* to NEXT.md; human answers; re-arm; resume from disk | `drive-pause-resume` |
| **Learning** | promote rubber-stamped pause-types to auto-proceed | `drive-autonomy-learning` |

### Non-negotiable architecture decisions

- **Hero does not build a loop engine or a completion evaluator.** Both
  ship as the harness `/goal`. Rebuilding them is the explicit anti-goal.
- **`/deliver` is NOT overloaded.** `deliver` = one spec, one step, human
  at the boundary. `drive` = whole initiative, autonomous. Different
  altitude, different verb. "Deliver this initiative" is detected and
  *offered* an upgrade to `/drive`, never silently reinterpreted.
- **`/drive`, not `/autopilot`.** `drive` reads cleanest as a verb and is
  the word the user reached for unprompted ("have Hero drive a loop");
  "autopilot" survives as a recognized NL synonym for when the user wants
  to be explicit about the autonomy.
- **The boundary is deterministic, not agentic.** `needs_me()` is a Hero
  predicate, not a babysitter agent — autonomy decisions must be
  inspectable and reproducible.

## Guardrails (cross-cutting — every child inherits these)

1. **Irreversible / outward-facing actions are a HARD pause forever** —
   migrations, deletes, deploys, external sends — even in `autonomous`
   mode. "The agent decided" is never an acceptable answer here.
2. **Dry-run / preview** — Drive can show the next 3 transitions it *would*
   auto-take before running unattended.
3. **Hard cap, never truly unbounded** — always pause at initiative
   boundaries and after N consecutive specs, regardless of mode ("one
   refactor a morning beats fifty overnight").
4. **Confirm on first arm** — arming Drive on an initiative for the first
   time always confirms.
5. **Conservative when unsure** — `needs_me()` pauses on uncertainty.

## Child Specs & Sequence

1. **[initiative-goal-section](initiative-goal-section/spec.md)** — the `## Goal` section type + queue rendering. *(foundational, no deps)*
2. **[needs-me-predicate](needs-me-predicate/spec.md)** — the `needs_me()` autonomy boundary + `autonomy:` policy field. *(foundational, no deps)*
3. **[hero-goal-command](hero-goal-command/spec.md)** — `hero goal` emit + `--check` + Stop-hook contract. *(deps: 1, 2)*
4. **[drive-pause-resume](drive-pause-resume/spec.md)** — pause-as-question + resume protocol. *(deps: 2, 3)*
5. **[drive-autonomy-learning](drive-autonomy-learning/spec.md)** — rubber-stamp → auto-proceed learning. *(deps: 4)*
6. **[drive-command-routing](drive-command-routing/spec.md)** — `/drive` command, NL routing, deliver-fallback. *(deps: 3)*

The minimum end-to-end vertical slice is **1 → 2 → 3 → 6**: that alone
gives a user-triggerable Drive that runs an initiative, gates on verify,
and stops at a `needs_me` pause. Specs 4 and 5 make the pause *graceful*
and the boundary *self-tuning* — high value, but the slice ships without
them.

## Acceptance Criteria (initiative-level)

- WHEN every child spec above reports `hero verify` PASS, THE SYSTEM SHALL
  let a user run `/drive <initiative>` and have an initiative with
  well-specified children execute to completion with human interaction
  only at `needs_me` pauses.
- THE SYSTEM SHALL NOT contain a Hero-side turn-driver or completion
  evaluator that duplicates the harness `/goal` loop.
- IF an autonomous run encounters an irreversible/outward-facing action,
  THEN THE SYSTEM SHALL pause for the human regardless of autonomy mode.
- WHEN Drive pauses, THE SYSTEM SHALL persist enough state to disk that a
  cold session resumes the run from the pause point.

## Risks

- **R1 — The classifier is the whole risk surface.** Too eager to pause →
  no gain over today; too eager to proceed → autonomous wrong turns
  compound across an initiative. Mitigated by conservative default +
  rubber-stamp learning + dry-run + hard caps. Owned by `needs-me-predicate`.
- **R2 — Harness coupling.** The Stop-hook seam is Claude Code-specific;
  Codex `/goal` differs. Keep `hero goal --check` harness-agnostic (plain
  JSON in/out) so each harness needs only a thin adapter. Owned by
  `hero-goal-command`.
- **R3 — Self-approving audits.** Risk that the agent rubber-stamps its own
  work. Mitigated because the "done" half is deterministic `hero verify`
  gates, not transcript judgment — and irreversible actions never
  auto-pass.
- **R4 — Verb proliferation / overload temptation.** Settled: new verb
  `/drive`, no `/deliver` overload, `autopilot` as synonym only.
