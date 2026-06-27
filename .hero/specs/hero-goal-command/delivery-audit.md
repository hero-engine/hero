# Delivery audit — hero-goal-command

**Audited:** uncommitted working tree (`git diff` over the spec's named files + new untracked `internal/drive/check.go`, `internal/drive/check_test.go`, `internal/cli/goal.go`, `internal/cli/goal_test.go`, `scripts/drive/`)
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] AC#1 — `hero goal <init>` prints the run condition — `internal/cli/goal.go:72-82` (emit path prints `GoalSection()` then `RunCondition()`); test `TestGoalEmit` (`internal/cli/goal_test.go:38`) asserts both the objective text and the derived condition listing children.
- [✓] AC#2 — `--check` all children pass → `done` — `internal/drive/check.go:109-111` (no pending → `done`); test `TestCheckDoneWhenAllChildrenCompleted` (`internal/drive/check_test.go:25`).
- [✓] AC#3 — ready next child → `continue` + Kickoff — `internal/drive/check.go:113-120` (continue carries `next.Kickoff()`); tests `TestCheckContinueGuided` (`check_test.go:40`) asserts `next_spec` + kickoff text, `TestGoalCheckContinue` (`goal_test.go:55`) asserts the CLI JSON carries `next_spec` and the kickoff.
- [✓] AC#4 — `NeedsMe` pause → `pause` with category/reason — `internal/drive/check.go:148-152` (pause carries `dec.Category`/`dec.Reason`); tests `TestCheckPauseSupervised` (`check_test.go:58`, Supervised category) and `TestCheckBlockedWhenDepsUnmet` (`check_test.go:67`, Blocked category).
- [✓] AC#5 — verdict derived from on-disk state only (cold-stable) — `Check`/`DryRun` (`check.go:84`, `:167`) are pure functions over `spec.Discover()` output; no in-memory session, no cache, no clock, no transcript input. Cold re-run is structurally identical. Tests run green under `-count=1`.
- [✓] AC#6 — does NOT run turns or judge completion from a transcript — no turn-execution or transcript-reading code exists in `check.go`/`goal.go`/`toolGoal`. Boundary asserted in `stop-hook-contract.md:42-46`. The Stop hook (`scripts/drive/stop-hook.sh`) only relays the verdict; the harness owns block/allow.

## Changes

- [✓] `internal/drive/check.go` — `Check()`, `DryRun()`, `Children()`, `depsMet()`, `completedSet()`, `step()` helpers. Verdict engine present and pure.
- [✓] `internal/cli/goal.go` + `internal/cli/root.go` — `hero goal` command with `--check` / `--dry-run` / emit default; initiative-type guard (`goal.go:62-64`); registered at `root.go:100`.
- [✓] `internal/serve/{mcp_tools.go,mcp_dispatch.go,mcp_tools_def.go}` — `hero_goal` MCP tool. Handler registered (`mcp_dispatch.go:34`), tool def added (`mcp_tools_def.go:414`), tool-count assertion bumped 43→44 (`mcp_test.go:253`, passes).
- [✓] `scripts/drive/stop-hook.sh` + `stop-hook-contract.md` — reference Claude Code Stop hook and the harness-agnostic contract doc.
- [✓] Tests across drive/cli/serve — 8 drive (`check_test.go`), 3 cli (`goal_test.go`), 1 serve parity (`mcp_test.go:1093`). All pass.

## Adversarial checks (requested)

1. **Scope creep (loop-driving / transcript-evaluation):** None found. `check.go` and `goal.go` contain no turn execution, no completion-from-transcript logic. The hook script relays the verdict; the harness drives. NOTE: `TestGoalMarkerFromTranscript` / `goalMarkerFromTranscript` exist in the cli package but live in pre-existing, out-of-scope files (`goal_capture_test.go`, `checkpoint.go`, `next_compact_handoff.go`) — a different "goal handoff marker" feature, not part of this delivery and not touched by it.

2. **CLI/MCP parity:** Genuine. Both surfaces call the identical `drive.Check(init, all)` / `drive.DryRun(...)` and serialize the same `drive.CheckResult` struct (CLI: `goal.go:69`; MCP: `mcp_tools.go:1274`). Parity is structural, not just test-asserted. The MCP emit default returns an `{initiative, objective, condition}` object rather than the bare emit text, but `--check` and `--dry-run` — the loop-consumed contract — are byte-identical shapes. Parity test `TestMCP_ToolGoal_CheckParity` confirms `verdict`/`next_spec`.

3. **`--check` statelessness:** Confirmed. Pure over `spec.Discover()`; no hidden session, cache, or mutable global. The "run-ledger" referenced in the spec design is a *future* dependency (drive-pause-resume) and is correctly NOT relied upon here.

4. **Honest v1 signal scope (unknown→safe):** Confirmed honest, no fake detectors. `step()` (`check.go:143-147`) passes `ActionClassified: true` and `NextScore: -1`. Tracing `NeedsMe` (`needsme.go:135-180`): `NextScore: -1` fails the `NextScore >= 0` guard (line 171) so Underspecified never fires; `ActionClassified: true` skips the Unknown pause (line 145); `VerifyStuckThreshold`/`DesignFork`/`AmbiguousPick`/`Irreversible` are zero/false and their guards never fire. The only live signals are mode (Supervised always pauses) and dependency-readiness (`Blocked`, derived from on-disk `depends-on` relations via `depsMet()`). That matches the settled v1 scope exactly. The genuine hard guardrail (Irreversible) is documented as living at the per-turn layer, not this between-spec verdict (`stop-hook-contract.md:43-46`).

5. **Every AC has code + test evidence:** Yes — see the AC table above. No performative DONE rows; no ledger row downgraded.

## Build & test evidence

- `go build ./...` — clean (exit 0).
- `go vet ./internal/drive/ ./internal/cli/ ./internal/serve/` — clean (exit 0).
- `go test ./internal/drive/ ./internal/cli/ ./internal/serve/` — all pass. Targeted, cold (`-count=1`): 8 drive Check/DryRun/Children tests PASS, 3 cli goal tests PASS, `TestMCP_ToolGoal_CheckParity` PASS, tool-count assertion (44) PASS.

## Audit notes

- The ledger's "exercised live" claims for AC#1/#3/#5 are not independently reproducible from artifacts alone, but each has a backing unit test that asserts the same behavior, so the criteria stand on test evidence regardless. No downgrade.
- Diff is tightly scoped to the spec's named files. No drift into unrelated code.
