# Delivery audit — mcp-transport-closes-midsession-supersede

**Audited:** `git diff -- internal/serve internal/index` (uncommitted working tree, branch `fix/mcp-transport-closes-midsession-supersede`)
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria
- [✓] AC1 — two live daemons coexist (no supersede of a live incumbent) — `internal/serve/mcp_singleton.go:79-85` returns `coexistRelease` instead of `singletonSignal`; `TestMCPSingleton_CoexistsWithLiveIncumbent` PASS (asserts incumbent NOT signaled, incumbent keeps primary pidfile, newcomer gets `mcp-<ppid>.pid.<pid>`).
- [✓] AC2 — dead/orphaned daemons still reaped (bcb9424 benefit survives) — stale-holder branch intact at `mcp_singleton.go:78-89` (dead incumbent → `singletonIsAlive` false → overwrite as free); `TestMCPSingleton_StalePidfileTreatedAsFree` PASS; watchdog reaps orphans via `TestParentWatchdog_ExitsWhenOriginalParentDead` + `TestParentWatchdog_ExitsOnPpidChange` PASS.
- [✓] AC3 — reconnect leaves new server serving — `TestMCPSingleton_ReconnectLeavesNewServerServing` PASS (new daemon claims per-pid file, old release removes only primary, no leak).
- [✓] AC4 — no leaked pidfile after SIGTERM — `mcp_lifecycle.go:64-74` installs `signal.Notify(SIGTERM, SIGINT)` handler that runs `release()` before `os.Exit(0)`. Code evidence only; no automated test (spec Test Plan item 4 scoped this as an integration test requiring a spawned process). Mechanism present and correct.
- [✓] AC5 — index tool no longer instant-fails under contention — `internal/index/index.go:79` DSN `?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)`; `TestConcurrentWrite_WaitsForBusyTimeout` PASS (waited 0.35s for a 300ms-held write lock then succeeded — proves busy_timeout engaged on the pooled connection); `TestOpen_UsesWALJournalMode` PASS (`PRAGMA journal_mode` == "wal").
- [✓] AC6 — transport stays up / recovers on panic — `mcp_lifecycle.go:110-121` `recover()` in `handleRequest` → `ErrCodeInternal`; `TestHandleRequest_RecoversPanicAndKeepsServing` PASS (panic on request 1 becomes a JSON-RPC error, request 2 served normally, process survives).

## Changes (Suggested Fix Approach, 5 items)
- [✓] #1 Graceful SIGTERM release — `mcp_lifecycle.go:64-74`. Handler covers SIGTERM **and** SIGINT (engineer added SIGINT beyond the spec sketch — documented, benign, arguably better). Uses `os.Exit(0)` matching the spec's sketch code.
- [✓] #1b Index busy_timeout + WAL — `index.go:79`. Deviated from spec's `?_busy_timeout=…&_journal_mode=…` sketch to modernc's `?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)`. Deviation is CORRECT: driver is `modernc.org/sqlite v1.53.0` (confirmed in go.mod), which honors only `_pragma=`. Independently verified by both index tests passing.
- [✓] #2 Coexist instead of supersede — `mcp_singleton.go:79-85` + new `coexistRelease` helper (`:107-117`). Live incumbent never signaled; per-pid pidfile written; release removes only own file.
- [✓] #3 Watchdog reparent-hardening — `mcp_watchdog.go:48-54`. Exit only when `now == 1 || !singletonIsAlive(startPpid)`. Both branches tested (ignores live-parent reparent; exits on confirmed-dead parent / reparent-to-init).
- [✓] #4 Panic recovery — `mcp_lifecycle.go:110-121`. Defense-in-depth, tested.

## Build / test evidence (independently reproduced)
- `go build ./cmd/hero` — OK. `go build ./...` — OK.
- `go vet ./internal/serve/... ./internal/index/...` — clean.
- `go test ./internal/serve/ ./internal/index/` — both packages ok. All 9 named tests re-run with `-count=1` (no cache): PASS.

## Open items
None blocking. No PARTIAL / SKIPPED / BLOCKED rows in the ledger.

## Audit notes (four risk areas the orchestrator flagged — all cleared)
- **(a) Coexist vs. orphan reaping — no regression against spec intent.** The stale-holder branch (`:78-89`) still reaps genuinely-dead incumbents, and the watchdog still self-reaps orphans (parent dead). The ONE case no longer reaped is a *live duplicate daemon under a live parent* — which the spec deliberately chose to allow ("trades 'at most one daemon' for 'never kill a live one'", Fix Approach #2). This means cross-session daemon accumulation (spec Secondary Defect #7) is NOT fully solved by this fix — it is explicitly out of scope for the transport-close bug. Worth the user knowing; not a defect.
- **(b) modernc DSN — honored.** `_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)` verified against `modernc.org/sqlite v1.53.0`. WAL is a persistent DB-level property (applies to every connection inherently); busy_timeout is per-connection and modernc applies `_pragma` on every pooled connection — the concurrency test proves the write path's pooled connection waited out the lock. Solid.
- **(c) Watchdog — cannot exit a healthy session, still exits genuine orphans.** Condition is correct on both sides; only a negligible PID-reuse race (original ppid recycled to a live proc before reparent-to-init) could theoretically defer an exit, no worse than the prior status quo. Both branches unit-tested.
- **(d) SIGTERM double-release — benign.** The signal goroutine calls `release()` then `os.Exit(0)`, which skips the deferred `release()`, so normal paths release exactly once. A SIGTERM landing exactly as `Run()` returns yields two concurrent `release()` calls, but each is `readMCPPIDRecord` + guarded `os.Remove` with the error ignored — idempotent, no panic, no state corruption.
- **gofmt:** Engineer's flagged pre-existing drift (index.go structs ~1667/2259) confirmed pre-existing and repo-wide (also api.go, jobs.go, mcp_tools.go, etc.); it is OUTSIDE the changed `Open()` region. The engineer's edits are gofmt-clean. "Left untouched for surgical discipline" is honest.
- Ledger's end-to-end claims (incumbent stayed ALIVE 4s; SIGTERM leaves no pidfile) are manual reproductions not re-run here; the seam-level unit tests cover the same logic and pass.
