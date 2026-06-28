---
title: "Pause-as-question + resume — a precise question to disk, resumable cold"
slug: drive-pause-resume
type: feature
status: completed
priority: high
horizon: next
tags: [drive, pause, resume, next, checkpoint, human-in-loop]
created: 2026-06-27
relations:
  - target: drive-autonomous-initiative-execution
    kind: parent
  - target: needs-me-predicate
    kind: depends-on
  - target: hero-goal-command
    kind: depends-on
delivery_method: manual
completed_at: 2026-06-28T00:02:07Z
---

# Pause-as-question + resume — a precise question to disk, resumable cold

## Goal

When a Drive run pauses, make the stop a **clean, self-contained question**
persisted to disk — not a bare "blocked" status — so the human can answer
with full context and the run resumes from exactly that point, even in a
fresh session or on another machine. This is the difference between "the
loop stopped" and "the loop handed me a decision and is waiting."

## Kickoff

When `hero goal --check` returns `pause`
([hero-goal-command](../hero-goal-command/spec.md)), write a structured
question to the user's handoff file (`.hero/NEXT.md` solo, `.hero/next/<user>.md`
team) and to a run-ledger that records where the run is. Enrich the existing
`hero next` / checkpoint machinery
([internal/cli/next.go](../../../../../internal/cli/next.go)) rather than
inventing a new file. Resume = human answers, run is re-armed, `--check`
reads ledger + answer and continues. All state on disk so a cold process
resumes identically. Reuse the `next` projection so the question travels
with commits like other handoff state.

## Problem

A pause is only useful if the human can act on it without re-deriving
context, and if answering it actually continues the run. Today's `next`
captures "Next / Blocked on / Context" as *status*, not as a *decision
request*: it doesn't carry the options, the recommendation, or the work
already done up to the stopping point, and there's no defined "answer →
resume" handshake.

## Design

### The pause question (written on `pause`)

A structured block in the handoff file:

```
## Drive paused — needs you
Initiative: drive-autonomous-initiative-execution
Stopped at: needs-me-predicate  (category: DesignFork)

**Decision:** <the specific question>
**Options:**
  A) <option> — <tradeoff>
  B) <option> — <tradeoff>
**Recommendation:** <A/B + one-line why>
**Done so far:** <children completed, what was just built>
**To resume:** answer above, then re-run /drive (or it auto-resumes on
next turn if the run is still armed).
```

The block is composed from the `--check` pause payload (category, reason)
plus the run-ledger. It is a *question*, not a log line.

### Run-ledger

A small on-disk record per active Drive run (e.g.
`.hero/drive/<initiative>.json`): armed mode, completed children, current
child, consecutive-proceed count (for the hard cap), pause state +
unanswered question, last answer. `--check` is stateless *because* this
ledger exists — it reads the ledger, not memory.

### Resume handshake

1. Human edits/answers the question in the handoff file (or replies in
   session).
2. The answer is recorded to the ledger (captured by the `/drive` skill or a
   `hero goal <init> --answer` write path).
3. Next `--check` sees the answered pause, clears it, and returns
   `continue`. Consecutive-proceed count resets at the pause.

Because the ledger + question live in `.hero/` and travel with commits, a
brand-new session, or a different machine, resumes the run with full
fidelity — the cold-start property Hero already guarantees for specs and
handoffs.

## Acceptance Criteria

- WHEN a Drive run pauses, THE SYSTEM SHALL write a structured question
  (decision, options, recommendation, work-done, resume instructions) to the
  user's handoff file.
- WHEN the human answers a pause and the run is re-armed, THE SYSTEM SHALL
  resume from the paused transition (not restart the initiative).
- THE SYSTEM SHALL persist run state (mode, progress, pause, answer) to disk
  such that a cold process resumes identically.
- WHILE a pause is unanswered, `hero goal --check` SHALL keep returning the
  same `pause` (idempotent — no progress past an open question).
- WHERE team mode is active, THE SYSTEM SHALL write the question to the
  per-user handoff file, not the shared one.

## Test Plan

- Unit: pause-question composition from a `--check` pause payload +
  run-ledger fixture (golden output).
- Unit: ledger round-trip (arm → progress → pause → answer → resume).
- Cold-resume: serialize ledger, start a fresh process, assert `--check`
  yields `continue` from the paused point.
- Idempotency: repeated `--check` over an unanswered pause is stable.
- Projection: the pause question travels via the `next` projection (commit
  carries it).

## Risks

- **Question quality** — a vague question is as bad as no pause. The
  composition must force options + recommendation; if `--check` can't supply
  them, the pause says so explicitly rather than emitting an empty shell.
- **Ledger/spec divergence** — the ledger must never contradict spec status
  on disk; treat spec status as source of truth and the ledger as run
  metadata only.
- **Team-mode races** — two users answering the same run. Out of scope for
  v1 (single-owner runs); note for the team-server follow-on.

## Changes

- `internal/drive/ledger.go` — `RunLedger` (load/save `.hero/drive/<init>.json`),
  `PendingPause`, `RecordAnswer`/`IsAnswered`/`SetPause`/`ClearPause`.
- `internal/drive/question.go` — `ComposeQuestion` (structured pause question)
  + `MergeQuestion`/`StripQuestion` (idempotent in-place block in the handoff file).
- `internal/cli/goal.go` — `--answer` flag; `reconcilePause` integrates the
  ledger + question into `--check` (resume-if-answered, else surface);
  `writeDriveQuestion`/`clearDriveQuestion` via the team-aware `resolveNextPath`.
- Tests: `internal/drive/ledger_test.go`, `internal/drive/question_test.go`,
  `internal/cli/goal_test.go`, `internal/cli/helpers_test.go` (flag reset).

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Pause writes a structured question to the handoff file | DONE | `ComposeQuestion` + `writeDriveQuestion`; `TestComposeMergeStripQuestion`; **exercised live** (NEXT.md got 1 question block) |
| 2 | Answer + re-arm resumes from the paused transition (not restart) | DONE | `RecordAnswer` + `reconcilePause` override pause→continue for the answered spec; `TestGoalPauseWritesQuestionThenResumesOnAnswer` step 4; **live** (--check → continue after --answer) |
| 3 | Run state persisted to disk; cold process resumes identically | DONE | `RunLedger` JSON at `.hero/drive/<init>.json`; `TestRunLedgerRoundTripAndAnswer` (reload preserves answer) |
| 4 | Unanswered pause → `--check` returns the same pause (idempotent) | DONE | `MergeQuestion` replaces (no duplicate); `TestGoal...` step 2 asserts exactly 1 block on repeat; verdict recomputed from disk |
| 5 | Team mode → write to the per-user handoff file | DONE | `writeDriveQuestion` writes via `resolveNextPath`, the existing team-aware resolver (`.hero/next/<user>.md` in team mode) already covered by `hero next` tests — reused, not reimplemented |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | `RunLedger` persistence | DONE | load/save/answer/pause |
| 2 | Question compose/merge/strip | DONE | idempotent block |
| 3 | `--answer` + `--check` pause reconciliation | DONE | resume handshake |
| 4 | Tests (drive + cli) + flag reset | DONE | 5 new tests, all passing |

### Exercise-the-feature check

- [x] User-visible behavior was exercised end-to-end: against the real workspace, `hero goal <init> --check` (supervised) paused and wrote the "Drive paused — needs you" block to `.hero/NEXT.md` and created `.hero/drive/<init>.json`; a second `--check` left exactly one block (idempotent); `hero goal <init> --answer "yes, proceed"` cleared the block; and `--check` then returned `verdict: continue`. Workspace restored afterward.

### Excellence Bar self-check

- [x] yes — verdict stays recomputed from spec status (ledger holds only *answers*, never contradicts disk truth); question is structured (decision/done/remaining/resume), not a status line; idempotent in-place merge; team-mode path reuses the proven `resolveNextPath` rather than a parallel implementation. AC#5's team branch is delegated (honestly noted), not separately unit-tested.
