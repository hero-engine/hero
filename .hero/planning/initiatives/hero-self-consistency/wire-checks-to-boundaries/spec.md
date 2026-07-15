---
title: "Wire Checks to Boundaries — Run the Validator and Reconciler Where Hero Already Looks"
slug: wire-checks-to-boundaries
type: feature
status: planning
domain: engineering
priority: high
size: medium
horizon: now
created: 2026-07-14
parent: hero-self-consistency
depends-on: spec-contract-enums-unified
tags: [validation, gates, reconcile, hooks, self-consistency]
---

# Wire Checks to Boundaries — Run the Validator and Reconciler Where Hero Already Looks

## Goal

The validator and reconciler that already exist run automatically at a boundary Hero already observes, instead of printing a reminder for a human to run them. Every issue class they report is deliberately classified warn or error, so the gate fires on defects and stays quiet on correct-but-uncalibrated state — and therefore stays enabled.

## Kickoff

Hero has a validator and a reconciler that nothing runs. `hero check` never calls `validate`; `--reconcile` defaults off and prints "run this yourself" — the exact human-memory lossiness `status-reconciliation` was built to delete.

**Status:** planning — **blocked on `spec-contract-enums-unified`**. Wiring a validator that disagrees with its own contract would gate the corpus against a definition known to be wrong.

**Pick up at:** the warn-vs-error policy for the 781 uncalibrated issues (504 file-not-found + 277 missing-smoke) — decide that before writing any wiring. The policy is the spec; the hook is an afternoon.

→ `hero check validate 2>&1 | rg -o 'file not found|missing smoke|invalid type|invalid status' | sort | uniq -c`

**Files:** `internal/cli/check.go:47`, `internal/cli/check.go:54`, `internal/cli/check.go:295`, `internal/reconcile/`, `internal/integrity/regression.go`
**Skip:** don't treat the 781 as defects — a planning spec naming unbuilt files is correct.

## Context

Parent initiative: `hero-self-consistency`. This child addresses finding (D) and is governed by finding (E).

The checks exist. Nothing runs them:

- `hero check validate` is a subcommand that plain `hero check` never invokes (`internal/cli/check.go:54` registers it as a subcommand only).
- `hero check --reconcile` defaults to false (`internal/cli/check.go:47`).
- `internal/cli/check.go:295` prints `Run 'hero check --reconcile' to auto-fix eligible items.` — a reminder to a human. This is verbatim the lossiness that `status-reconciliation` (completed) was built to eliminate: a correct automation that only runs when someone remembers it.

A check nobody runs is a check that doesn't exist.

**But wiring it today would fire 1019 issues** on the maintainer's own repo:

| Class | Count | Nature |
|---|---|---|
| `file not found` | 504 | **Uncalibrated policy** — a planning spec whose `files:` names unbuilt code is *correct* |
| `missing smoke` | 277 | **Uncalibrated policy** |
| `invalid type` | 150 | Defect — fixed by `spec-contract-enums-unified` |
| `invalid status` | 14 | Defect — fixed by `spec-contract-enums-unified` |

**781 of 1019 are not defects.** A gate that fires 1019 times is not a gate — it is a thing people disable. Calibration is the work here, and it is why this spec is `medium` rather than `small`.

## Approach

**The policy decision comes first. The wiring is the easy part.**

1. **Classify every issue class warn or error.** For each class the validator emits, decide deliberately:
   - `invalid type` / `invalid status` → **error** once #2 lands. These are unambiguous contract violations with a target of 0.
   - `file not found` (504) → almost certainly **warn**, and possibly not an issue at all for `planning` specs. A spec that names code it intends to create is doing its job. Consider making the check status-aware — error for `completed`, silent for `planning` — rather than a flat warn.
   - `missing smoke` (277) → **warn**. Smoke coverage is an aspiration, not a contract.
   - The status-aware refinement is where the real value is: `file not found` on a *completed* spec is a genuine signal that the spec lies about what shipped, which is exactly this initiative's thesis. Flattening it to "warn" everywhere discards that.
2. **Wire to a boundary Hero already observes.** The commit hook exists — `hero next install-hooks` already installs a pre-commit hook. Prefer extending an observed boundary over inventing a new one. Do not build a new daemon, watcher, or scheduler.
3. **Turn the reminder into an action.** `check.go:295`'s "run this yourself" line should become either the reconcile actually running, or nothing. A reminder is the failure mode, not the fix.
4. **Audit `internal/reconcile/` for the same axis collapse as `regression.go`.** `internal/integrity/regression.go` overwrites delivery lifecycle with verification health — the defect `spec-state-axes` (child #5) exists to correct. If the reconciler makes the same mistake, wiring it to a boundary *automates* the corruption at higher frequency. Audit before wiring, and treat any finding as a regression to fix here.

## Changes

1. Classify every validator issue class
   - Enumerate the classes `hero check validate` emits.
   - Assign each warn or error, with recorded rationale.
   - Implement severity in the validator's output so callers can gate on error and report warn.
2. Make `file not found` status-aware
   - Evaluate erroring for `completed` specs and staying silent for `planning`.
   - This converts 504 noise items into a small number of real signals about specs that misreport what shipped.
3. Wire `validate` into the default `hero check` run
   - `internal/cli/check.go:54` — invoke validation as part of the default path, reporting warns and failing on errors.
4. Flip the reconcile default or remove the reminder
   - `internal/cli/check.go:47` — decide whether `--reconcile` defaults true at the boundary.
   - `internal/cli/check.go:295` — delete the human reminder either way. It is the anti-pattern.
5. Wire to the commit boundary
   - Extend the existing `hero next install-hooks` pre-commit hook.
   - Errors block; warns report. Must be fast enough not to be bypassed — a slow hook gets `--no-verify`'d, which is the same failure as a check nobody runs.
6. Audit `internal/reconcile/` for delivery/verification axis collapse
   - Compare against the defect documented in `internal/integrity/regression.go` and analyzed in `spec-state-axes`.
   - If present, fix as a regression before wiring — automating a corrupting write is worse than not running it.

## Acceptance Criteria

- THE SYSTEM SHALL classify every validator issue class as warn or error, with no class left unclassified.
- WHEN `hero check` runs without subcommands THE SYSTEM SHALL invoke the spec validator.
- WHEN a commit is made in a repo with Hero's hooks installed THE SYSTEM SHALL run the validator and block the commit on error-severity issues.
- WHILE a spec has status `planning` THE SYSTEM SHALL NOT report `file not found` as an error for files the spec intends to create.
- IF a `completed` spec names files that do not exist THEN THE SYSTEM SHALL report an error, because the spec misreports what shipped.
- THE SYSTEM SHALL NOT print a reminder instructing a human to run `hero check --reconcile`.
- IF `internal/reconcile/` overwrites delivery lifecycle state with verification health THEN THE SYSTEM SHALL be corrected before the reconciler is wired to any boundary.
- WHEN the validator runs at the commit boundary THE SYSTEM SHALL complete fast enough that bypassing it is not tempting.

## Boundaries

- **Do not start before `spec-contract-enums-unified` lands.** Hard dependency, not a preference.
- Do not migrate corpus specs to satisfy the validator — that is #2's job, per its fate decisions.
- Do not build new observation infrastructure. Use boundaries Hero already has.
- Do not redesign the status field. `spec-state-axes` (#5) owns that; this spec only ensures the reconciler doesn't corrupt it.
- Do not add command-ref validation to the gate — `generated-command-refs-validated` (#3) runs as its own test.

## Risks

- **This is the child where the initiative can fail.** The 781 uncalibrated issues are a policy question wearing a bug's clothing. Treating them as defects produces a gate that fires 1019 times, gets disabled within a week, and ships negative value — Hero would then have a check that exists, runs at a boundary, and is universally bypassed. That is strictly worse than today.
- **Hard dependency on #2 is real, not procedural.** Wiring a validator whose type contract is known wrong in three directions would gate every commit against a definition this initiative already established is broken.
- **A slow hook is a bypassed hook.** `--no-verify` is one flag away. Performance is a correctness requirement here.
- **The `internal/reconcile/` audit may find a real defect**, which would grow this spec. If the reconciler collapses the same axes as `regression.go`, fixing it may belong with `spec-state-axes` instead — coordinate rather than duplicating. If it lands here and is substantial, bump `size:` to `large`.
- **Blocking commits is a behavior change for every Hero user.** Consider whether errors block or report-only in the first release, and whether the boundary is opt-in initially.

## Validation

- `hero check` on this repo reports 0 error-severity issues and a finite, defensible list of warns.
- The 781 uncalibrated issues no longer surface as failures; spot-check that a `planning` spec naming unbuilt files passes clean.
- Deliberately break a spec's type and confirm the commit hook blocks.
- Deliberately create a `completed` spec naming a nonexistent file and confirm it errors — the status-aware refinement's real payoff.
- Time the hook on a full-corpus run; confirm it is fast enough to leave enabled.
- Confirm the `internal/reconcile/` audit is documented, with its finding stated either way — "audited, no collapse found" is a valid and necessary result to record.
