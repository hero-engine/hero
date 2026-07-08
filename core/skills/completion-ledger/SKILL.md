---
name: completion-ledger
description: Format and validation contract for the Completion Ledger — the closing artifact hero spec verify Gate 1 parses
metadata:
  audience: engineer, delivery-leads, auditors
  purpose: delivery-gate-contract
---

## What I do

Define the Completion Ledger — the mandatory closing artifact of every delivery — as a single contract, both sides: the format an engineer authors and the rules `hero spec verify` Gate 1 and a delivery lead validate it against. Every other file that used to restate or fork this format cites this skill instead.

## Format

```markdown
## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | THE SYSTEM SHALL ... | DONE | `path/to/file.go:NN` — what was implemented |
| 2 | WHEN X THE SYSTEM SHALL ... | PARTIAL | What remains and why it's not yet done |
| 3 | ... | SKIPPED | Why skipped + needs explicit sign-off |
| 4 | ... | BLOCKED | What was tried + the specific obstacle |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Edit `path/to/file` — ... | DONE | Section(s) touched + summary |
| 2 | ... | PARTIAL | What remains |

### Exercise-the-feature check

- [x] User-visible behavior was exercised end-to-end: <command run + what was observed>
- [ ] OR: cannot be exercised in this environment because <specific reason>

### Excellence Bar self-check

Honest answer to "would a senior engineer who cares about this codebase be proud to ship this?" — yes / no + one-line justification. If no, list what would need to change for a yes.
```

### Preamble (optional but useful)

Before the tables, include: a brief understanding of the task as executed, the stack detected and skills loaded, validation performed (test runs, builds, exercise commands), and any risks/follow-ups/rough edges for the delivery lead.

## Status definitions

- **`DONE`** — implemented and verified. Has corresponding code, test evidence, and (for user-visible behavior) end-to-end exercise.
- **`PARTIAL`** — partially implemented. Must include what remains and why it isn't done.
- **`SKIPPED`** — explicitly not done. Must include why, and requires explicit user or delivery-lead sign-off before the spec can flip to `completed`.
- **`BLOCKED`** — attempted, hit a true blocker (see `agent-reliability` — "Persistence on continuous tasks"). Must include what was tried and the specific obstacle.

## Honesty rules

- Every acceptance criterion gets a row. Every `## Changes` item gets a row. No omissions.
- `DONE` is a high bar — code exists, tests exist, and the feature was exercised. If you only wrote code, the item is `PARTIAL` or you owe an exercise note.
- Do not mark items `DONE` to avoid friction. The delivery lead will challenge `DONE` rows that lack corresponding evidence; performative ledgers are worse than honest ones.
- If you ran out of context, time, or willingness mid-task, the affected items are `PARTIAL` — not `DONE`.

## What Gate 1 actually parses

Hero's engine parses this section mechanically to gate `hero spec verify`. Match the contract exactly:

- **Section shape**: a `## Completion Ledger` section (matched case-insensitively), containing `### Acceptance Criteria` and `### Changes` sub-tables, plus `### Exercise-the-feature check` and `### Excellence Bar self-check` checkbox blocks (these two are matched on "exercise" / "excellence" appearing anywhere in the sub-header — the exact wording above is not required, but keep it for consistency).
- **Table tolerance**: rows may have 3 or 4 columns. 4-column = `# | Summary | Status | Note`; 3-column (no index) = `Summary | Status | Note`. Bold text, backticks, and extra whitespace in cells are stripped before matching.
- **Statuses**: `DONE` / `PARTIAL` / `SKIPPED` / `BLOCKED`, matched case-insensitively with bold/backticks tolerated. Any other value parses as `UNKNOWN` and **fails Gate 1** — this is a real reason not to invent new statuses.
- **`[signed-off]` in the Note cell** is the machine-readable sign-off that lets a `SKIPPED` or `BLOCKED` row pass Gate 1 (matched case-insensitively; `[signed off]` without the hyphen also works). Without it, any `SKIPPED`/`BLOCKED` row fails the gate.
- **Exercise-the-feature is advisory at the gate.** A missing or bare (no detail text) exercise checkbox produces an `ADVISORY:` detail line in the gate output — it does **not** fail Gate 1. It is mandatory by convention (the delivery lead still requires it for a user-visible `DONE` row before validating the ledger), just not machine-enforced today. Hardening this into a blocking gate is a possible follow-up, tracked separately from this contract.
- **Gate 1 pass condition, stated plainly**: the ledger section is present, and every Acceptance Criteria row and every Changes row is `DONE`, or `SKIPPED`/`BLOCKED` with `[signed-off]` in the Note.

## Validating a ledger

This is the delivery lead's (or cold auditor's) side of the contract:

- **`DONE` evidence bar**: cross-check each `DONE` row against actual evidence — code on disk, test files, exercise notes. A `DONE` row with no corresponding diff or test evidence is performative; challenge and downgrade it. For user-visible behavior, confirm the Exercise-the-feature check is populated — unit tests alone are not sufficient evidence.
- **`PARTIAL` is not an end state.** Loop back to the engineer with the specific PARTIAL rows and explicit instructions to finish them — do not ask "ship as-is or chase it down?"; the standing answer is chase it down. Only escalate to the user after a second pass still returns PARTIAL on the same row with a concrete, specific obstacle (not "minor polish," not "low value").
- **`SKIPPED` / `BLOCKED` escalation**: these are legitimate human-judgment halts. Surface the row, the engineer's stated reason, and a concrete recommendation. Getting sign-off means writing `[signed-off]` into that row's Note cell — that is what lets Gate 1 pass; nothing else counts.
- **Closing rule**: `hero spec verify <slug>` is the only path to `completed`. Never hand-edit `status: completed` in a spec's frontmatter, and never run `hero spec complete` on a work spec (feature, bug, or initiative) to bypass the gates.
