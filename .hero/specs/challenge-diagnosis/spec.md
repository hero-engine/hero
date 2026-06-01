---
title: Challenge/Revise a Bug Diagnosis — Engineer Feedback Loop
slug: challenge-diagnosis
type: feature
status: completed
priority: P1
tags: [diagnose, challenge, feedback, debug-investigator, dx]
created: 2026-04-22
relations:
  - target: hero-killer-features
    kind: parent
horizon: now
completed_at: 2026-05-18T19:25:38Z
---

## Goal

Let an engineer push back on a diagnosed bug spec — either layering a new
hypothesis on top of the existing analysis or rejecting it entirely — and
have the `debug-investigator` re-examine the bug with that feedback as
input. The spec's `## Investigation History` section preserves every round
so future readers see the evolution of understanding, not just the final
answer.

## Problem

`/diagnose` produces a spec with investigation findings, root cause, and
fix plan. Sometimes the engineer reads it and disagrees: the root cause is
wrong, the analysis missed a related subsystem, or the fix plan targets the
symptom instead of the cause. Today there is no structured way to feed that
disagreement back. The engineer either re-explains the bug from scratch
(losing the original investigation work) or manually edits the spec (losing
the reasoning chain). Neither option gives the `debug-investigator` the
chance to revise its analysis with new context.

## Design

### Two feedback modes

| Mode | Trigger | Behavior |
|---|---|---|
| **Layer** (default) | "I think it's ALSO related to X" | Adds the engineer's hypothesis without discarding the existing analysis. The `debug-investigator` re-examines with both the original findings AND the new hypothesis, producing a merged analysis. |
| **Reject** | "This analysis is wrong" / "re-diagnose this" | Triggers a full fresh re-investigation with the engineer's feedback as the starting hypothesis. The original analysis is archived into `## Investigation History` — never deleted. |

Mode detection is natural language: if the feedback contains negation or
replacement language ("wrong", "not correct", "off base", "re-diagnose",
"start over"), use reject. Otherwise default to layer.

### Command surface

```
/challenge <slug> <feedback>
/challenge auth-null-pointer I think this is actually a race condition in the session middleware
/challenge auth-null-pointer --reject The root cause is wrong — this is a config parsing issue, not a null pointer
```

Natural language routing in `AGENTS.md` catches phrases like:

- "I think this bug is actually about..."
- "the root cause is wrong"
- "this analysis is off"
- "re-diagnose this"
- "also consider..."
- "what about the possibility that..."

These route to `/challenge` with the slug inferred from context (active
spec, most recent diagnosis, or explicit mention).

### Re-investigation flow

1. `/challenge` routes to the `feature-delivery-lead`
2. The delivery lead reads the existing spec at `.hero/planning/bugs/<slug>/spec.md`
3. It determines the mode (layer vs reject) from the feedback language
4. **Layer mode**: passes the original analysis + engineer feedback to the
   `debug-investigator` with instructions to incorporate both
5. **Reject mode**: archives the current analysis into `## Investigation
   History`, then passes the engineer's feedback as the starting hypothesis
   to a fresh `debug-investigator` run
6. The `debug-investigator` writes its updated findings back into the spec
7. The `## Investigation History` section is updated with the new round

### Investigation History format

Appended to the bug spec after each challenge round:

```markdown
## Investigation History

### Round 1 — Initial diagnosis
- **Date**: 2026-04-22T14:00:00Z
- **Agent**: debug-investigator
- **Root cause**: Null pointer dereference in `cart.Total()` when items list is empty
- **Confidence**: High

### Round 2 — Challenged (layer)
- **Date**: 2026-04-22T15:30:00Z
- **Challenged by**: engineer
- **Feedback**: "I think this is also related to the session middleware resetting the cart on timeout"
- **Revised root cause**: Null pointer in `cart.Total()` is the immediate cause, but the session middleware's aggressive timeout (30s) causes premature cart resets that surface the bug more frequently
- **What changed**: Added session middleware timeout as contributing factor; fix plan now includes both the nil check and a timeout configuration change
- **Confidence**: High
```

### Skill: `challenge-diagnosis`

A new skill in `skills/challenge-diagnosis.md` that defines:

- The investigation history format (section structure, required fields per round)
- The re-diagnosis protocol (how to merge vs replace analysis)
- Layer mode instructions: "Treat the original analysis as valid evidence.
  Add the engineer's hypothesis as a new thread. Investigate both. The
  result should be a merged analysis, not two competing ones."
- Reject mode instructions: "Archive the entire current analysis into
  Investigation History. Start fresh with the engineer's feedback as your
  primary hypothesis. You may reference archived findings as evidence but
  do not assume they are correct."

### What this is NOT

This is instructions-in-markdown, not infrastructure. No new Go code. The
`debug-investigator` already does investigation work; `/challenge` feeds it
new context through the `feature-delivery-lead` coordination layer — the
same pattern as `agent-cold-start`. The delivery lead reads the mode,
manages the history section, and delegates to the investigator.

## Changes

- `commands/challenge.md` — new command definition for `/challenge <slug> <feedback>` with mode detection and routing to the delivery lead
- `agents/feature-delivery-lead.md` — add a "Challenge handling" section defining the layer/reject flow, history management, and `debug-investigator` delegation
- `AGENTS.md` — add routing entries for challenge/revise/reject language in the Natural Language Routing table
- `skills/challenge-diagnosis.md` — new skill defining the investigation history format, re-diagnosis protocol, and mode-specific instructions

## Acceptance Criteria

- WHEN an engineer runs `/challenge <slug> <feedback>` with additive language THE SYSTEM SHALL default to layer mode and pass both the original analysis and the new hypothesis to the `debug-investigator` for a merged re-examination
- WHEN an engineer runs `/challenge <slug> <feedback>` with negation or replacement language (e.g. "wrong", "not correct", "re-diagnose") THE SYSTEM SHALL use reject mode and archive the current analysis into `## Investigation History` before starting a fresh investigation
- WHEN the `debug-investigator` completes a challenge re-examination THE SYSTEM SHALL append a new round to the `## Investigation History` section recording the date, mode, engineer feedback, revised root cause, and what changed
- WHEN a spec has been challenged multiple times THE SYSTEM SHALL preserve all prior rounds in `## Investigation History` in chronological order so the full reasoning chain is visible to future readers
- WHEN a user says "I think this bug is actually about...", "the root cause is wrong", "this analysis is off", or "re-diagnose this" THE SYSTEM SHALL route to `/challenge` via the natural language routing table in AGENTS.md
- WHEN `/challenge` runs in reject mode THE SYSTEM SHALL NOT delete the original analysis — it must be archived in the history section, not discarded
- WHEN `/challenge` runs THE SYSTEM SHALL NOT require the original diagnosing agent session to be available — the spec file on disk contains all needed context
- WHEN `/challenge` runs against a slug that has no existing diagnosis spec THE SYSTEM SHALL report an error and suggest running `/diagnose` first

## Boundaries

- Does **not** auto-accept or auto-reject diagnoses — the engineer decides when to challenge and which mode to use
- Does **not** delete previous analysis — layers merge it, rejects archive it into Investigation History
- Does **not** require the original diagnosing agent session to be available — the spec on disk is the complete input
- Does **not** modify the `debug-investigator` agent's core logic — challenge context is passed as input, not as a behavioral change
- Does **not** add new Go code — follows the agent-cold-start pattern of instructions in markdown
- Does **not** auto-trigger on diagnosis completion — the engineer reads the spec and decides whether to challenge
