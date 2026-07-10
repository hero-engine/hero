# Delivery audit — drive-judge-findconflicts-backstop

**Audited:** `git diff -- internal/` + untracked `internal/driveio/` (working tree, 2026-07-09)
**Verdict:** SHIP
**Surface:** clean

Piece 3 of the conflict-aware-drive chain. Cold audit against artifacts on disk —
build/vet/test run fresh, every claim traced to real code + a real assertion.

## Gate results (run fresh by the auditor)
- `go build ./...` → exit 0
- `go vet ./internal/drive/... ./internal/index/... ./internal/driveio/... ./internal/cli/... ./internal/serve/...` → exit 0
- `go test ./...` → exit 0 (no FAIL/panic)
- Purity: `go list -deps ./internal/drive/ | grep internal/index` → empty (drive does NOT import index) ✓

## Acceptance criteria
- [✓] Undeclared delivering overlap → `SeamDetected` naming file(s) + in-flight spec — `check.go:337-355` detect+dedup, `needsme.go:232-235` branch + `seamDetectedReason` (`needsme.go:173`); unit `TestCheckSeamDetectedPauseNamesOverlap` (check_test.go:364) asserts category + names file+slug; CLI `TestGoalCheckDetectsUndeclaredSeam` (goal_test.go:168) drives the real cobra command against a live sqlite index and asserts `"category": "SeamDetected"` + names both.
- [✓] Autonomous+promoted proceeds past `SeamDetected` (promotable) — `Promotable()` true-set includes `CategorySeamDetected` (`needsme.go:82`); branch routes through `maybePromoted` (`needsme.go:233`); `TestCheckSeamDetectedPromotableAcrossModes` (check_test.go:418) asserts Autonomous+promoted→continue, Guided→pause, Supervised→pause; `TestNeedsMeSeamDetectedPromotable` (needsme_test.go:169).
- [✓] Authored `conflicts-with` → `SeamCollision` once, never `SeamDetected` — two mechanisms both real: (a) structural — authored+delivering target excluded from `ready` at `check.go:298`, fallback emits SeamCollision; NeedsMe checks `SeamBlocked` (needsme.go:225) BEFORE `SeamDetected` (needsme.go:232). (b) explicit subtraction — `authoredConflictTargets` (`check.go:137`) subtracted at `check.go:345-349`. `TestCheckAuthoredWinsOverDetected` (check_test.go:453) covers (a); `TestCheckSeamDetectedSkipsAuthoredPicksUndeclared` (check_test.go:389) covers (b) and WOULD fail if subtraction removed (authored-peer sorts before ghost-peer, would surface first); CLI `TestGoalCheckAuthoredConflictStaysSeamCollision` (goal_test.go:188) asserts SeamCollision and NOT SeamDetected end-to-end.
- [✓] nil detector → byte-for-byte piece-1 verdict — detect block guarded by `else if detect != nil` (`check.go:337`); `TestCheckNilDetectorMatchesPiece1` (check_test.go:474) re-invokes Check 10× and compares verdict+next_spec, plus asserts nil never emits SeamDetected and an empty detector == nil (not a trivial pass).
- [✓] Reuse `FindConflicts`; no second detector — `FindConflicts` and `FindDeliveringConflicts` both delegate to a shared private `findConflicts(slug, statuses...)` (`index.go:1270-...`); same SQL, only the status `IN` set varies.
- [✓] Scope to locally-delivering only — `FindDeliveringConflicts` = `findConflicts(slug, "delivering")`; `TestFindDeliveringConflicts` (index_test.go:275) asserts planning+in-review EXCLUDED and delivering INCLUDED (FindConflicts sees 3, delivering sees only feat-b).
- [✓] Both callers wired → identical verdicts (parity green) — both build `driveio.Detector(idx)` (goal.go:115, mcp_tools.go:1287); `TestMCP_ToolGoal_CheckParity` (mcp_test.go:1093) green; `TestMCP_ToolGoal_CheckDetectsUndeclaredSeam` (mcp_test.go:1148) asserts the same SeamDetected verdict through MCP dispatch on the same fixture shape as the CLI test.
- [✓] DryRun authored-only, no detected gate — `DryRun` signature unchanged (no detect param, `check.go:390`); boundary doc comment `check.go:385-389`; detector never wired into it.

## Changes
- [✓] 1. `needsme.go` — `CategorySeamDetected` const (`:66-73`), Promotable true-set (`:82`), 3 RunContext fields (`:124-134`), `seamDetectedReason` (`:173`), NeedsMe branch after SeamBlocked via maybePromoted (`:232`).
- [✓] 2. `check.go` — `DetectedConflict` struct with no index import (`:17-24`), final `detect` param (`:266`), post-selection dedup block (`:337-355`), `authoredConflictTargets` (`:137`), DryRun untouched.
- [✓] 3. `index.go` — `FindDeliveringConflicts` + shared `findConflicts`; ORDER BY `s.slug, ft2.file_path` preserved end-to-end.
- [✓] 4. `internal/driveio/detector.go` — new single-file package, `Detector(idx)` filter+map, nil on query error, doc comment states the stub-footprint limitation.
- [✓] 5. `goal.go` — opens index (+ RefreshIfStale self-heal), passes `driveio.Detector(idx)`; DryRun path unchanged.
- [✓] 6. `mcp_tools.go` — wires `driveio.Detector(idx)`; `nil` promoted left as-is (gap explicitly out of scope, commented); dryRun unchanged.
- [✓] 7. Tests — 5 drive Check cases + 2 NeedsMe cases + Promotable scope + `TestFindDeliveringConflicts` + 2 MCP tests + 2 CLI end-to-end.

## Boundaries held
- Purity: `drive` does not import `index` (verified via go list -deps). ✓
- No second overlap detector (shared `findConflicts`). ✓
- No auto-authoring — Check is a read-only verdict; no spec mutation in the judge path. ✓
- Local-delivering only; no wave ordinal / new sequencing hierarchy. ✓
- MCP `promoted: nil` gap left as-is (commented). ✓
- DryRun signature unchanged. ✓

## Audit notes
- Both callers add an index self-heal before opening (`index.RefreshIfStale` in goal.go, `s.ensureFreshIndex()` in mcp_tools.go). This is slightly beyond the literal Changes text (which said only "open an *index.DB") but is in-scope and directly serves the documented "Index freshness" risk — the detector must reason over current file footprints. Failure is warned/swallowed, not fatal. Not scope drift; noted for the record.
- `internal/driveio` ships with no dedicated test file — acceptable: it is a 1-function filter+map exercised end-to-end through both the CLI and MCP seam tests (live sqlite), which is stronger evidence than a unit test would be.
- No performative DONE rows found. No PARTIAL / SKIPPED / BLOCKED rows. Every criterion has code + an assertion that would fail if the behavior regressed.
