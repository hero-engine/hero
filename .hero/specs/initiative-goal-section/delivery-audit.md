# Delivery audit — initiative-goal-section

**Audited:** `git diff -- internal/` (uncommitted working tree)
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria

- [✓] Initiative parses/exposes `## Goal` via `GoalSection()` — `internal/spec/spec.go:1232` (initiative-only guard; reads `Sections["goal"]`, populated by `parseSections` at `spec.go:1019`). Test `TestGoalSectionInitiativeOnly` (`spec_test.go`) asserts initiative body non-empty, no section bleed into `## Problem`, and empty on a `type: feature`.
- [✓] Derive canonical default condition from children when unauthored — `RunCondition()`/`ChildSlugs()`/`authoredRunCondition()` at `spec.go:1245-1295`. Tests `TestRunConditionDerivedFromChildren`, `TestRunConditionPrefersAuthored`, `TestRunConditionNonInitiativeEmpty`, `TestChildSlugsSortedAndScoped` all pass. See audit note 1: this is library-only, no non-test caller.
- [✓] `hero queue` renders initiative `## Goal` ("Run"), no Kickoff nag — `internal/cli/list.go:343-352` branches on `TypeInitiative`, emits `Run opener — arm with /drive <slug>` then the Goal body, and `continue`s past the Kickoff path. Test `TestQueueRendersInitiativeGoalOpener` asserts the `/drive` hint, the Goal body, and the *absence* of the Kickoff nag.
- [✓] Leaf specs render `## Kickoff` exactly as today (no regression) — leaf path at `list.go:354-359` is unchanged and only reached when `Type != TypeInitiative`. Pre-existing `TestQueueRendersKickoffBody` (not in the diff) still passes.
- [✓] `hero check` advisory when initiative lacks a Goal (non-blocking) — `internal/cli/check.go:326-339` adds an `initiative-goal-coverage` `warn` row and **does not** increment `issues` (contrast `kickoff-coverage` at `check.go:308` which does `issues += len(missing)`). Helper `missingGoalInitiatives()` at `check.go:617-642` skips non-initiatives and completed/superseded statuses. Test `TestMissingGoalInitiatives` verifies only the goalless initiative is flagged, and that a goalless leaf feature is *not*. See audit note 2.

## Changes

- [✓] Add `GoalSection()`, `RunCondition()`, `ChildSlugs()`, `authoredRunCondition()` + `sort` import to `internal/spec/spec.go` — all four present (`spec.go:1232,1245,1262,1282`); `sort` import added (`spec.go:8`). `bufio` was already imported, so `authoredRunCondition`'s scanner compiles.
- [✓] Branch `renderSpecsKickoff` in `internal/cli/list.go` — `list.go:343-352`, initiative → Goal opener with `/drive` hint.
- [✓] Advisory in `internal/cli/check.go` — non-blocking `warn` row, no issue-count bump.
- [✓] Tests across spec/list/check — 7 new tests (`spec_test.go` x5, `list_test.go` x1, `check_test.go` x1). Ledger says "7 new tests"; exact count is 5+1+1 = 7. All pass.

## Open items

None. No PARTIAL / SKIPPED / BLOCKED rows in the ledger; none warranted.

## Audit notes

1. **`RunCondition`/`ChildSlugs`/`authoredRunCondition` have no non-test callers.** They are exercised only by `spec_test.go`. AC#2 ("derive the canonical default condition from children") is satisfied as a *library capability* with unit tests, and the spec's Changes section explicitly names all four functions — so this is delivered as specified, not scope creep. But the derivation is not yet surfaced anywhere a user sees it (the queue renders the authored `## Goal` body verbatim via `GoalSection()`, not the derived condition). The wiring presumably lands in a sibling spec (`/drive` / `--check`). Flagging so the orchestrator knows the derivation is a staged primitive, not yet live.

2. **Non-blocking clause of AC#5 is verified by code inspection, not by a direct test assertion.** `TestMissingGoalInitiatives` asserts the helper returns the correct specs, but no test asserts that a workspace whose *only* finding is a missing Goal still reports "No issues found" / exit-clean. The code is unambiguous (`issues` is never touched in the goal-coverage block), so confidence is high, but the test gap is worth noting.

3. **Scope is clean — no predicate/`--check` leakage.** The only `needs_me` / `PASS` occurrences are inside the generated condition *string* (`spec.go:1254,1256`); there is no verify-running, PASS/FAIL evaluation, or judging logic. The spec's stated boundary ("container and surface only") is respected.

4. **Diff is well-scoped.** Exactly the 6 files named in the Changes section were touched (+321 lines, no deletions). No drift into unrelated files.

5. **Build/vet/test all clean.** `go build ./...` clean; `go vet ./internal/spec/ ./internal/cli/` clean; `go test ./internal/spec/ ./internal/cli/` both pass (cli ~22.7s). Targeted runs green: `TestGoalSection*`, `TestRunCondition*`, `TestChildSlugsSortedAndScoped`, `TestMissingGoalInitiatives`, `TestQueueRendersInitiativeGoalOpener`, `TestQueueRendersKickoffBody`.
