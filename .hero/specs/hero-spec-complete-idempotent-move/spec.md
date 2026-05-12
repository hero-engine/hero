---
title: "`hero spec complete` Idempotent Move — Don't Strand Specs Whose Status Was Already Flipped"
type: bug
status: completed
severity: high
tags: [spec, lifecycle, complete, idempotency, move]
created: 2026-05-01
relations:
  - target: kickoff-prompts-queue
    kind: related
horizon: now
---

## Kickoff

Fix `hero spec complete` so that a spec whose frontmatter status was already flipped to `completed` (by `/deliver`, by an agent edit, by `hero check --reconcile`) still gets *moved* from `planning/` to `specs/` instead of erroring out with "already completed."

**Status:** completed — `runComplete` is now idempotent across status / move / reindex steps. Self-tested by running `hero spec complete` against this spec's own file.

**Pick up at:** lived-experience pass on the user's new project — flip a stranded spec's status without using the command, then run `hero spec complete` against it and verify it gets moved instead of erroring.

→ `hero spec complete <stranded-spec-path>`

**Files:** [internal/cli/complete.go:31](internal/cli/complete.go:31), [internal/cli/complete.go:111](internal/cli/complete.go:111), [internal/cli/complete_test.go](internal/cli/complete_test.go)

## Goal

Make `hero spec complete <path>` idempotent across its three steps (status flip, planning→specs move, reindex). Today the function early-exits the moment it sees `status: completed` in frontmatter, even when the spec is still sitting in `planning/` and the move never ran. As a result, feature specs accumulate in `planning/` indefinitely on any project where status flips happen outside this command.

## Problem

User dogfooding on a new project: *"feature specs never get moved when complete."* Reproducible in this codebase too — every spec we shipped this session ended up either pre-positioned in `specs/` (because we wrote them there directly) or got rescued by `hero check --reconcile`'s alternate move path. The `hero spec complete` command itself is broken.

The bug is at [internal/cli/complete.go:55-57](internal/cli/complete.go:55):

```go
if s.Status == spec.StatusCompleted {
    return fmt.Errorf("spec %s is already completed", s.Slug)
}
```

This early-returns *before* the move call on line 66. So any spec whose status got set to `completed` by some other path — `/deliver` writing the file, an agent editing frontmatter directly, `hero check --reconcile` flipping the status field — can never be moved by `hero spec complete`. The user runs the command, gets an "already completed" error, and the file stays in `planning/`.

The function is internally inconsistent: it's *trying* to be a multi-step lifecycle operation (flip status, move, reindex, sync tracker), but it gates the entire pipeline on a single signal that's been satisfied — even when downstream steps are pending.

## Design

Rewrite `runComplete` to evaluate **what work remains** rather than refuse on status alone. Each of the four steps becomes individually idempotent:

1. **Status flip** — only writes if the current status isn't already `completed`. Today this part already works; the early-exit just hides it.
2. **Move from `planning/` to `specs/`** — `moveToSpecs` is already idempotent: it returns `(path, false, nil)` when the spec isn't under `planning/` ([internal/cli/complete.go:131-134](internal/cli/complete.go:131)). No change needed; just stop blocking the call.
3. **Reindex** — always safe to re-run; `index.Rebuild` is idempotent.
4. **Tracker sync** — already gated on `tracker_id` and config; safe to re-run if both are present.

Replace the early-return with a "nothing to do" detector that fires only when *all* steps are no-ops:

```go
alreadyCompleted := s.Status == spec.StatusCompleted
alreadyMoved := !strings.Contains(specPath, "planning/")  // simplified
if alreadyCompleted && alreadyMoved {
    fmt.Printf("Spec %s is already completed and in specs/ — nothing to do.\n", s.Slug)
    return nil  // exit 0, not an error
}
```

Then run each step. The status flip becomes conditional; the move and reindex run unconditionally (they self-no-op when nothing's needed); the tracker sync stays conditional on its own preconditions.

### Output messaging

- If status flips and move happens: same as today.
- If status was already completed but the move ran: `"Status was already completed. Moved <src> → <dest>. Re-indexed."` — make clear what work was performed.
- If both no-ops: friendly "nothing to do" + exit 0. Not an error.
- Tracker sync output unchanged.

### Why this isn't a behavior break

- Callers expecting an error exit when status is already completed: change the contract — the goal is "make sure this spec is fully completed," not "fail if you've already started." Idempotency is the safer contract for a lifecycle command.
- Callers expecting a successful move: this *fixes* their case, currently silently broken.
- `hero check --reconcile` already does the move via its own path — this brings `hero spec complete` to parity rather than introducing a new code path.

### What's NOT in scope

- Changing where `/deliver` flips status. The fix is to make `hero spec complete` resilient to whatever order things happen in, not to dictate one canonical order.
- Detecting half-moved specs (file exists in both `planning/` and `specs/`). `moveToSpecs` already errors on collision; preserve that.
- Backfilling already-stranded specs. Out of scope for this fix; users can run `hero check --reconcile` to clean up history. Could spec a one-shot migration command separately if the volume warrants it.
- The `runComplete` call signature or flag set. No new flags needed.

## Changes

- `internal/cli/complete.go` — rewrite the early-exit guard. Keep the existing parse / type-validity check. Replace the unconditional refusal-on-status with: (a) detect already-fully-complete (status + location), exit 0 with friendly message; (b) otherwise run each step idempotently. Status flip becomes conditional on `s.Status != StatusCompleted`. Move and reindex always run.
- `internal/cli/complete_test.go` — three new tests:
  - `TestComplete_StatusFlippedNoMove`: spec in `planning/` with status already `completed` → command moves it, doesn't error.
  - `TestComplete_FullyComplete_NoOp`: spec in `specs/` with status `completed` → command exits 0 without error or modification.
  - `TestComplete_StandardFlow`: spec in `planning/` with status `delivering` → command flips status AND moves (regression coverage on the existing happy path).

No CLI flag changes, no new commands, no MCP changes.

## Acceptance Criteria

- WHEN `hero spec complete <path>` runs against a spec in `planning/` whose frontmatter status is already `completed` THE SYSTEM SHALL move the spec to `specs/<slug>/spec.md` and re-index, without erroring.
- WHEN `hero spec complete <path>` runs against a spec already in `specs/` whose status is already `completed` THE SYSTEM SHALL exit 0 with a "nothing to do" message and not modify any files.
- WHEN `hero spec complete <path>` runs against a spec in `planning/` whose status is `delivering` THE SYSTEM SHALL flip status to `completed`, move the spec, and re-index — preserving today's happy-path behavior.
- WHEN the move would collide with an existing spec at the destination THE SYSTEM SHALL error with the existing collision message — collision detection unchanged.
- THE SYSTEM SHALL not introduce new flags, subcommands, or MCP surface.
- THE SYSTEM SHALL keep `moveToSpecs`'s existing idempotent contract (returns false / no error when the spec isn't under `planning/`).
- IF the tracker is configured AND the spec has a `tracker_id` THEN the tracker sync step SHALL run on every invocation that produces real work, exactly as today.

## Boundaries

- Does **not** change `/deliver`, `/diagnose`, or any other workflow that flips spec status. They can keep doing what they do; the fix is in the receiver.
- Does **not** introduce a `--force` flag or a separate "move only" subcommand. Idempotent by default is the right shape.
- Does **not** address half-moved specs (file in both locations). The existing collision check stays.
- Does **not** retroactively migrate specs that have been stranded by the bug. `hero check --reconcile` already covers that.

## Mission Fit

> "Does this make the next agent session start smarter than the last one ended — and does it raise the floor for everyone, not just the senior dev who already knows what to ask?"

Floor-raising. Today the workaround is "know that `hero check --reconcile` will rescue your stranded specs" — esoteric knowledge. With this fix, `hero spec complete` does what its name promises regardless of what touched the file first. The user shouldn't need to understand which path flipped the status before running the canonical complete command.
