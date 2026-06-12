# Delivery Audit — spec-completion-loop

**Audited:** `git diff HEAD -- internal/cli/verify.go internal/cli/verify_test.go internal/async/runner.go domains/engineering/commands/deliver.md .hero/knowledge/spec-lifecycle-last-mile-gap.md`
**Verdict:** SHIP
**Surface:** noteworthy
**Confidence:** high

---

## Acceptance Criteria

### AC-1: Ledger gate PASS -> Record() flips Criterion to passing — PASS

**Evidence:** `verify.go:142-149` — after Gate 1 passes, `recordLedgerToGraph()` is called. The function at `verify.go:569-598` converts DONE ledger rows into `acceptance.RunResult` entries with `Status: "pass"` and AC keys formatted as `<slug>:AC-<N>`, then calls `acceptance.Record()` via a freshly opened graph store.

**Test:** `TestVerify_LedgerWritebackToGraph` (verify_test.go:384-455) seeds 3 Criterion nodes with status "proposed", runs verify with all-DONE ledger, asserts all 3 are "passing" in the graph and `ACStatusUpdates == 3`. Verified passing.

### AC-2: Parent initiative auto-complete when all siblings completed — PASS

**Evidence:** `verify.go:604-655` — `autoCompleteParentIfReady()` iterates the target's relations for `parent` or `child-of` kinds, loads all specs via `spec.Discover()`, finds the parent initiative, counts children, and calls `completeAndArchive()` when all children are completed. Called from `runVerify()` at line 162 after `completeAndArchive` succeeds on the target spec.

**Test:** `TestVerify_InitiativeAutoComplete` (verify_test.go:458-534) sets up a parent initiative with 2 children (one already completed, one delivering), verifies child-two, asserts `InitiativeCompleted == "parent-init"` and parent is archived to `specs/`. Verified passing.

### AC-3: Initiative auto-complete prints message naming the initiative — PASS

**Evidence:** `verify.go:175-177` — human output prints `Initiative %q auto-completed — all children delivered`. JSON output includes `InitiativeCompleted` field (line 64). Both paths confirmed in `TestVerify_InitiativeAutoComplete` which checks the JSON `initiative_completed` field.

### AC-4: Manual deliver chains verify automatically — SKIPPED [signed-off]

**Evidence:** The spec's Changes section 3 acknowledges the deliver flow is agent-instruction-driven, not Go code. The skill instruction path was strengthened instead: `deliver.md:248-258` now reads "MUST run `hero spec verify <slug>` before reporting delivery as complete. This is not optional." This is a reasonable pragmatic decision — the manual deliver path is inherently interactive and instruction-driven.

### AC-5: Async runner invokes verify instead of spec complete — PASS

**Evidence:** `runner.go:177-196` — the async runner now calls `hero verify --skip-tests <slug>` instead of `hero spec complete`. If verify fails, falls back to `hero spec complete` (lines 183-192), preserving the prior "always archives" behavior.

**Design note:** The spec proposed `--force` on the verify invocation. The implementation drops `--force` and uses a fallback instead — this is strictly better because it lets gate failures surface in logs rather than being silently forced.

**Gap:** The spec's Test Plan item 3 called for `internal/async/runner_test.go` to assert the runner calls verify with correct flags. No such test was written. The code change is correct and the fallback logic is sound, but this path has zero automated test coverage. This is the only material gap in the delivery.

### AC-6: Record() preserves existing satisfied_by edges — PASS

**Evidence:** Verified by reading `internal/acceptance/record.go:60-161`. The `Record()` function works by `UpsertNode` on Criterion nodes (line 114) and only creates `satisfied_by` edges when a SHA is present (lines 128-148). Edge creation uses `UpsertEdge` which is additive — there is no edge deletion path in the function. The verify integration passes `RunResult` entries with empty SHA fields (no commit context), so `Record()` flips status only without touching edges. Existing `satisfied_by` edges from prior run-result ingests are untouched.

### AC-7: Verify output reports AC status changes — PASS

**Evidence:** `verify.go:172-173` — human output: `"  AC graph: %d criteria flipped to passing"`. `verify.go:64` — JSON output: `ACStatusUpdates int` field with `json:"ac_status_updates,omitempty"`. Both paths exercised in `TestVerify_LedgerWritebackToGraph` which asserts `ACStatusUpdates == 3` from JSON output.

### AC-8: Skip auto-complete if children were force-completed — SKIPPED [signed-off]

**Evidence:** The spec's Risks section 5 documents that `--force` is not recorded in frontmatter today. The `autoCompleteParentIfReady` function has no force-detection code (confirmed by reading lines 604-655). The `hasForced` variable from the spec's pseudocode was not implemented. This is correctly scoped as a follow-up — tracking `--force` in frontmatter is a prerequisite.

### AC-9: Exercise-the-feature demoted to advisory — PASS

**Evidence:** `verify.go:250-260` — the three exercise branches. The two failure paths (unchecked, checked-without-detail) no longer set `allDone = false`. They emit `"ADVISORY: ..."` messages with regression test nudges instead.

**Test:** `TestVerify_ExerciseDemotedToAdvisory` (verify_test.go:536-600) creates a spec with all ACs DONE but exercise NOT checked. Asserts Gate 1 result is PASS (not FAIL), and that ADVISORY text mentioning exercise appears in gate details. Verified passing.

---

## Modified Files

| File | Lines changed | Purpose |
|---|---|---|
| `internal/cli/verify.go` | +102 | Ledger writeback, initiative auto-complete, exercise demotion, output updates |
| `internal/cli/verify_test.go` | +337 | 4 new tests: writeback, initiative auto-complete, exercise demotion, AC key mismatch |
| `internal/async/runner.go` | +13 / -6 | Replace `hero spec complete` with `hero verify --skip-tests` + fallback |
| `domains/engineering/commands/deliver.md` | +10 / -7 | Strengthen verify instruction from suggestion to hard MUST |

## New Files

| File | Purpose |
|---|---|
| `.hero/knowledge/spec-lifecycle-last-mile-gap.md` | Knowledge capture: architectural insight on detection-without-prevention anti-pattern |

## Tests

- **18 verify tests pass** (14 pre-existing + 4 new), confirmed via `go test ./internal/cli/ -run TestVerify -v`
- **4 new tests added:**
  - `TestVerify_LedgerWritebackToGraph` — AC writeback with graph verification
  - `TestVerify_InitiativeAutoComplete` — parent initiative auto-complete + archive
  - `TestVerify_ExerciseDemotedToAdvisory` — exercise no longer blocks gate
  - `TestVerify_ACKeyMismatchResilience` — partial graph coverage doesn't error
- **Missing test:** `internal/async/runner_test.go` for verify invocation (spec Test Plan item 3). The runner change at `runner.go:177-196` has no automated coverage. The code is straightforward (exec + fallback) but the fallback path is untested.
- **Pre-existing failure:** `TestMarkdownInvocationsResolveAgainstRootCmd` in `web/docs/` — not touched by this delivery, pre-existing drift.

## Noteworthy Observations

1. **Missing runner test.** The spec's Test Plan explicitly called for `internal/async/runner_test.go` (item 3: "Assert the async runner now calls `hero verify` instead of `hero spec complete`"). This test was not written. The code change is correct, but the async verify-fallback path — which is the safety net for the entire async delivery flow — has zero automated coverage. Low risk because the fallback preserves prior behavior, but this should be tracked.

2. **`--force` dropped from async verify.** The spec said `hero verify --skip-tests --force`; the implementation uses `hero verify --skip-tests` with a fallback to `hero spec complete`. This is a deliberate improvement — gate failures now surface in logs instead of being silently masked. The fallback achieves the same "always archives" safety property. No concern.

3. **`autoCompleteParentIfReady` returns only the first parent.** If a spec has multiple parent relations (unusual but structurally possible), only the first auto-completed parent is reported in `InitiativeCompleted`. Subsequent parents would still be completed (the function continues the loop), but only the first slug is returned and reported. Edge case, unlikely to matter in practice.
