# Delivery audit — drive-pause-resume

**Audited:** `git diff -- internal/cli/goal.go internal/cli/goal_test.go internal/cli/helpers_test.go` + untracked `internal/drive/{ledger,question}.go` (+ tests)
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria

- [✓] **AC#1 — pause writes a structured question to the handoff file.** `ComposeQuestion` (question.go:18) emits decision/options-via-reason/done-so-far/remaining/resume-instructions, wrapped in `<!-- drive-pause -->` markers. Written via `writeDriveQuestion` → `reconcilePause` (goal.go:155, :96). Test `TestComposeMergeStripQuestion` (question_test.go:8) asserts the block carries the decision, category, next-spec, and `--answer`; integration test asserts NEXT.md gets the "Drive paused" block (goal_test.go:113).
- [✓] **AC#2 — answer + re-arm resumes from the paused transition (not restart).** `RecordAnswer` (ledger.go:78) records the answer keyed to the paused spec; `reconcilePause` flips `pause`→`continue` only when `led.IsAnswered(res.NextSpec)` (goal.go:124) and re-populates `res.Kickoff` from the *same* next spec — it does not reset progress. Test step 4 asserts `--check` returns `continue` after `--answer` (goal_test.go:142).
- [✓] **AC#3 — run state persisted to disk; cold process resumes identically.** `RunLedger` JSON at `.hero/drive/<init>.json` (ledger.go:32-68). `TestRunLedgerRoundTripAndAnswer` (ledger_test.go:5) saves, reloads in a fresh struct, and asserts both the pause and the answer survive reload — the cold-start property exercised at the unit level. The CLI integration test crosses real process invocations (separate `runCmd` calls reload from disk each time).
- [✓] **AC#4 — unanswered pause → `--check` returns the same pause (idempotent).** `MergeQuestion` (question.go:40) replaces an existing block in place rather than appending; verdict is recomputed from spec status every turn via `drive.Check` (goal.go:90), so the pause re-derives identically. Test step 2 asserts exactly one block after a repeat `--check` (goal_test.go:128); `TestComposeMergeStripQuestion` asserts `MergeQuestion` is idempotent at the unit level (question_test.go:27).
- [✓] **AC#5 — team mode → per-user handoff file.** Delegated (honestly) to the existing `resolveNextPath` (next.go:107), which returns `.hero/next/<user>.md` when `cfg.NextMode()=="team"`. Both `writeDriveQuestion` and `clearDriveQuestion` route exclusively through it (goal.go:156, :179) — no team-specific code path bypasses the tested resolver. Ledger states this is delegated and not separately unit-tested; that is accurate, not an overclaim. There is no new branch for team mode to leave untested.

## Changes

- [✓] `internal/drive/ledger.go` — `RunLedger` load/save + `PendingPause`, `RecordAnswer`/`IsAnswered`/`SetPause`/`ClearPause`. Present, matches spec.
- [✓] `internal/drive/question.go` — `ComposeQuestion` + `MergeQuestion`/`StripQuestion` (idempotent in-place block). Present, matches spec.
- [✓] `internal/cli/goal.go` — `--answer` flag (goal.go:44); `reconcilePause` integrates ledger+question into `--check` (resume-if-answered else surface); `writeDriveQuestion`/`clearDriveQuestion` via team-aware `resolveNextPath`. Present, matches spec.
- [✓] Tests — `ledger_test.go`, `question_test.go`, `goal_test.go`, `helpers_test.go` (goal flag reset added at helpers_test.go:186). Present.

## Correctness properties (read, not just trusted)

- **Ledger never contradicts spec status.** `reconcilePause` calls `drive.Check(init, all)` first to compute the verdict from on-disk spec status, then reconciles. The ledger stores only `Answered` (spec→answer) and a transient `Pause`; it stores no verdict. The resume override fires only inside the `Verdict=="pause" && Pause!=nil` branch, so the ledger can never manufacture a `continue` against a non-paused disk state. ✓
- **Idempotent.** Confirmed via `MergeQuestion` marker-based in-place replace + test step 2. ✓
- **Answer overrides only the answered transition.** Override gated on `IsAnswered(res.NextSpec)`, a per-spec map keyed by slug. A pause on a different next-spec is not in the map. ✓

## Open items

None blocking. (No PARTIAL / SKIPPED / BLOCKED rows in the ledger; all rows DONE with concrete evidence.)

## Audit notes

- **Sticky-answer behavior (note, not a blocker).** `Answered` entries are never removed — only the transient `Pause` is cleared on resume (`ClearPause` does not touch `Answered`; `RecordAnswer` sets the map permanently). Consequence: once spec A is answered, the resume override is permanent for spec A. A *future, distinct* pause that happens to land on the same `NextSpec` slug would auto-resume without re-asking the human. For v1 this is consistent with the design intent ("answer = I've made the call for this child, proceed") and the spec explicitly scopes multi-pause/team-races out of v1 (Risks section). Worth tracking for the team-server follow-on, but in scope it is correct.
- **AC#5 has no direct test, by design.** The team-mode branch lives entirely in the pre-existing, separately-tested `resolveNextPath`. The ledger discloses this. No new untested team code exists. Calling this out for surface honesty, not as a gap.
- **Scope is clean.** `git status` shows exactly the five spec-named files (3 modified + 4 new, all enumerated in `## Changes`). No drift.
- **Build/vet/test all green.** `go build ./...` clean; `go vet ./internal/drive/ ./internal/cli/` clean; `go test ./internal/drive/ ./internal/cli/` pass. Named tests verified non-skipped via `-v`.
- **Live-exercise claim (AC#1/#2/#4 "exercised live").** The ledger claims a real-workspace run against `.hero/NEXT.md`. I cannot verify the live run from artifacts (the workspace was restored afterward, per the ledger), but the equivalent behavior is covered by the cross-invocation CLI integration test, so the SHIP does not rest on the unverifiable live claim.
