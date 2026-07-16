# Delivery audit — parent-watchdog-race-ci-red

**Audited:** `git diff HEAD -- internal/serve/` (uncommitted, on top of HEAD `3410199`)
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria / verification commands
- [✓] Race fixed under repetition — `go test -race -count=20 ./internal/serve/ -run TestParentWatchdog` → `ok 5.677s`, no `WARNING: DATA RACE`. Run fresh by this auditor.
- [✓] Full `serve` package clean under `-race` — `go test -race -count=1 ./internal/serve/` → `ok 13.937s`.
- [✓] Full CI gate green — the exact workflow command (`.github/workflows/test.yml:32` = `go test -race -count=1 ./...`) → exit 0: 86 packages `ok`, 0 `FAIL`, 0 `DATA RACE`, 16 no-test-files. First-hand run, not trusted from ledger.
- [✓] Build clean across all call sites — `go build ./...` → exit 0. Signature change compiles with the return discarded.

## Changes
- [✓] `startParentWatchdog` returns `<-chan struct{}` — `internal/serve/mcp_watchdog.go`: signature now `func startParentWatchdog(done <-chan struct{}) <-chan struct{}`; goroutine has `defer close(stopped)` as its first statement; `return stopped` at the tail. Doc comment explains the happens-before-join rationale. Poll condition (`:48-49`) byte-for-byte unchanged.
- [✓] Racing test joins before seam restore — `mcp_watchdog_test.go:160-166`: `stopped := startParentWatchdog(done)` then `defer func(){ close(done); <-stopped }()`. This defer is registered *after* the seam-restore defer (`:133-138`), so LIFO orders it first: close → join → then restore. This is a genuine happens-before edge, not a poll.
- [✓] Production caller unchanged — `internal/serve/mcp_lifecycle.go:78` still `startParentWatchdog(done)`, return discarded. Exit-on-parent-death logic untouched.
- [✓] Exit-branch tests NOT modified — the sole diff hunk in the test file is at `@@ -157` inside `_IgnoresReparentWhileParentAlive`. `_ExitsOnPpidChange` (`:76-120`) and `_ExitsWhenOriginalParentDead` (`:181-224`) are verbatim unchanged, still `defer close(done)`, and pass under `-race` (part of the `-count=20` and full-package runs).

## Open items
None. Ledger is all-DONE; every row verified with first-hand evidence.

## Audit notes

**No suppression.** `grep` for `t.Skip` / `SkipNow` / `testing.Short()` in the test file → none. `-race` is present in the gate and in every verification command. The fix is real synchronization (a channel close/receive join), exactly as the spec prescribed.

**Deadlock claim (exit-branch tests) is correct.** The ledger asserts `_ExitsOnPpidChange` and `_ExitsWhenOriginalParentDead` were deliberately *not* given a `<-stopped` join because a join there would deadlock. Verified by reasoning: in those two tests the fake `watchdogExit` does `exited <- code; select{}`, so the poll goroutine parks forever inside `watchdogExit` and never reaches the loop `return` — therefore `defer close(stopped)` never fires and `<-stopped` would block indefinitely. Their existing `exited`-channel handshake supplies the happens-before edge (last seam read → `exited <-` send → test `<-exited` receive → return → restore), so they are race-free without a join. The "do NOT apply the join here" instruction is load-bearing and was honored.

**Production guarantee preserved.** The exit-on-genuine-orphan path is unchanged in the diff — only `stopped := make(chan struct{})`, `defer close(stopped)`, and `return stopped` were added around the identical poll loop. `_ExitsWhenOriginalParentDead` (parent confirmed dead → must reap) and `_ExitsOnPpidChange` both pass under `-race`, confirming the daemon still exits when its original parent dies.

**Design judgment — test-only join channel on a production signature: acceptable, endorsed.** The return value has zero production consumers today, which invites the "test needs shaping production API" objection. But returning a completion channel from a goroutine-spawning function is idiomatic Go, not a smell — a spawner that gives callers no way to observe termination is the more common defect. The doc comment is honest about why the channel exists. The heavier alternative the spec weighed (parameterize the four seam vars into a config struct) is legitimately rejected: `singletonIsAlive` is a *shared* seam also used by the singleton-acquire path (`mcp_singleton.go:79`), and `startParentWatchdog` is a free function with no per-instance home, so struct-homing the globals would ripple into another subsystem for a test-harness-only race. The channel-join is the surgical, correctly-scoped choice. Minor nit (not a defect, spec already flagged it as optional): production discards the return silently; `_ = startParentWatchdog(done)` would document intent. Not worth a change.

**Scope.** Diff touches exactly the two named files plus `.hero/NEXT.md` (a projected handoff file that travels with commits by design — not scope drift). No stray edits.
