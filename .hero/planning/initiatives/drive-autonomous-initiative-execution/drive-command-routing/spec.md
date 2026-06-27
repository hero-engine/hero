---
title: "`/drive` command + natural-language routing + `/deliver`-on-initiative fallback"
slug: drive-command-routing
type: feature
status: planning
priority: high
horizon: now
tags: [drive, command, skill, routing, natural-language, ux]
created: 2026-06-27
relations:
  - target: drive-autonomous-initiative-execution
    kind: parent
  - target: hero-goal-command
    kind: depends-on
---

# `/drive` command + natural-language routing + `/deliver`-on-initiative fallback

## Goal

Give Drive its one-sentence front door: **`/drive <initiative>`**, plus
natural-language routing so *"autopilot this initiative"* / *"put X on
autopilot"* trigger the same thing, plus a graceful fallback that catches
*"deliver this initiative"* and offers the upgrade instead of silently doing
the wrong thing. This is the "say it in plain text and it goes" layer; it
arms the run and owns the pause/resume UX, delegating all judgment to
`hero goal`.

## Kickoff

Add a `/drive` command + a `drive` skill (NOT a dedicated agent — an agent
would duplicate the harness loop and we want the boundary deterministic in
`needs_me`). The skill: resolve the initiative, ensure/author its `## Goal`
([initiative-goal-section](../initiative-goal-section/spec.md)), confirm on
first arm, emit the condition via `hero goal --emit`, hand it to the harness
`/goal`, ensure the Stop hook calling `hero goal --check`
([hero-goal-command](../hero-goal-command/spec.md)) is wired, and render
pause questions / accept answers ([drive-pause-resume](../drive-pause-resume/spec.md)).
Add the routing-table rows. Mirror the command across harness command dirs
(`core/commands/`, `domains/engineering/commands/`) per existing convention.

## Problem

Everything underneath (goal section, predicate, judge, pause/resume) is
useless without a trivially-triggerable entry point. And the user's muscle
memory is natural language — "drive this initiative", "autopilot X",
sometimes "deliver this initiative." Without routing, the feature isn't
reachable; without the deliver-fallback, the most ambiguous phrasing
silently mis-fires into the one-spec flow.

## Design

### The command / skill

`/drive <initiative>` runs the `drive` skill:

1. Resolve the initiative (by slug or fuzzy title); error if it's not an
   initiative (offer `/deliver` for a leaf spec).
2. Ensure a `## Goal` condition exists (author the default if missing).
3. **Confirm on first arm** — show the condition, the autonomy mode, and the
   guardrails; require go-ahead.
4. `hero goal <init> --emit` → paste into the harness `/goal`.
5. Ensure the Stop hook (→ `hero goal <init> --check`) is installed/armed.
6. On `pause`: render the question, collect the answer, record it, resume.
   On `done`: report completion + what shipped.

Skill, not agent — explicitly. The loop is the harness's; the judgment is
the predicate's; the skill is thin orchestration + UX.

### Natural-language routing

Add to the CLAUDE.md / harness routing table:

| User intent | Command |
|---|---|
| autopilot this initiative, put X on autopilot, drive/run the initiative, keep working the whole initiative | `/drive <initiative>` |

`/drive` is canonical; "autopilot" and friends are synonyms that route to
it.

### `/deliver`-on-initiative fallback

When `/deliver` (or "deliver this …") resolves to a `type: initiative`,
do not run the one-spec flow. Instead offer:

> "That's an initiative — `/deliver` works one spec at a time. Want to
> `/drive` the whole thing (autonomous, pauses when it needs you) instead?"

The user picks; nothing is silently reinterpreted. (Implements the
initiative's "no `/deliver` overload" decision.)

## Acceptance Criteria

- WHEN a user runs `/drive <initiative>`, THE SYSTEM SHALL arm an autonomous
  run after a first-arm confirmation showing condition, mode, and guardrails.
- WHEN a user says "autopilot this initiative" (or close synonyms), THE
  SYSTEM SHALL route to `/drive`.
- IF `/drive` targets a non-initiative spec, THEN THE SYSTEM SHALL decline
  and point to `/deliver`.
- WHEN `/deliver` resolves to an initiative, THE SYSTEM SHALL offer to
  `/drive` it rather than silently delivering one child.
- WHILE a run is armed, THE SYSTEM SHALL surface pause questions and accept
  answers to resume, delegating verdicts to `hero goal --check`.
- THE SYSTEM SHALL be implemented as a skill, not a dedicated agent.

## Test Plan

- Routing: NL phrasings ("autopilot X", "drive the X initiative", "keep
  going on X") all resolve to `/drive <X>`.
- Fallback: "deliver this initiative" surfaces the upgrade offer, not a
  one-spec delivery.
- Guard: `/drive <leaf-spec>` declines with the `/deliver` pointer.
- Arm flow: first arm confirms; re-arm of an in-progress run resumes (does
  not restart).
- Command parity: `/drive` present and consistent across all harness command
  dirs.

## Risks

- **Synonym over-capture** — routing "drive" too broadly hijacks unrelated
  phrasing. Mitigation: require an initiative object in the utterance; when
  ambiguous, ask rather than assume.
- **Confirmation fatigue** — confirming every arm annoys. Mitigation:
  confirm on *first* arm per initiative; subsequent resumes don't re-confirm.
- **Skill/loop boundary blur** — keep the skill free of judgment logic; if
  it starts deciding proceed/pause, that belongs in `needs_me`.
