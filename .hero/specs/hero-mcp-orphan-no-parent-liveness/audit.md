# Delivery audit — hero-mcp-orphan-no-parent-liveness

**Audited:** working tree on `fix/mcp-orphan-parent-liveness` vs `main` (`git diff main -- internal/ go.mod go.sum` + 4 untracked new files read directly)
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria
This spec is a bug with no explicit "Acceptance Criteria" section; the contract is the `## Suggested Fix Approach` (Changes 1–3) plus the `## Test Plan`. Audited against those.

- [✓] Change 1 — watchdog gated + torn down in `Run()` — `internal/serve/mcp_lifecycle.go:40-44`. `if s.input == os.Stdin { done := make(chan struct{}); defer close(done); startParentWatchdog(done) }` matches the spec's After block byte-for-byte; inserted above the unchanged scanner loop. No new import needed (`os` already imported, line 8).
- [✓] Change 2 — portable ppid-poll watchdog — `internal/serve/mcp_watchdog.go:1-43`. Matches spec verbatim: `parentWatchdogInterval = 30s`, `armParentDeathSignal()` first, `startPpid := os.Getppid()`, goroutine with `select { <-done: return; <-ticker.C: if Getppid() != startPpid { os.Exit(0) } }`, `defer ticker.Stop()`.
- [✓] Change 3 — Linux fast path + non-linux no-op — `internal/serve/mcp_watchdog_linux.go` (`//go:build linux`, `unix.Prctl(unix.PR_SET_PDEATHSIG, SIGTERM, …)`) and `internal/serve/mcp_watchdog_other.go` (`//go:build !linux`, empty `armParentDeathSignal`). Match spec verbatim. Tags are mutually exclusive + exhaustive — exactly one symbol per build, no duplicate-symbol risk (confirmed by `grep` + both cross builds passing).
- [✓] Test 1 — `TestParentWatchdog_NotStartedUnderSetIO` — `mcp_watchdog_test.go:17-45`. Drives `Run()` with a `bytes.Buffer` ("ping" request), asserts it returns within 5s. Because `SetIO` sets `s.input != os.Stdin`, the gate branch is skipped — directly guards the test-suite-safety invariant the ledger claims.
- [✓] Test 2 — `TestStartParentWatchdog_StopsOnDone` — `mcp_watchdog_test.go:52-67`. Calls `startParentWatchdog(done)`, `close(done)`, polls `runtime.NumGoroutine()` back to baseline. Asserts clean teardown of the `defer close(done)` path. Does not (and by design cannot) exercise `os.Exit`.
- [~] Test 3 — `armParentDeathSignal` Linux smoke (build-tagged) — NOT delivered. Spec listed it as a third test item. Absent. Low impact: the function is a single non-panicking `unix.Prctl` call whose return is intentionally discarded, and both cross-platform builds compile it. Genuine minor gap, not performative.

## Changes
- [✓] `internal/serve/mcp_lifecycle.go` — gate block added (lines 33-44). Surgical; only the documented insertion, scanner loop untouched.
- [✓] `internal/serve/mcp_watchdog.go` — new, portable watchdog.
- [✓] `internal/serve/mcp_watchdog_linux.go` — new, Linux PR_SET_PDEATHSIG.
- [✓] `internal/serve/mcp_watchdog_other.go` — new, non-linux no-op.
- [✓] `internal/serve/mcp_watchdog_test.go` — new, 2 tests + `waitForGoroutinesAtMost` helper.
- [✓] `go.mod` — `golang.org/x/sys v0.43.0` promoted indirect→direct (single line in/out). `go.sum` unchanged (`git diff main -- go.sum` empty). Matches ledger.

## Verification gates (re-run independently from repo root)
- `go build ./...` — PASS (exit 0)
- `GOOS=linux go build ./...` — PASS (exit 0)
- `GOOS=darwin go build ./...` — PASS (exit 0)
- `go test ./internal/serve/...` — PASS (`ok internal/serve`)
- `go test -race -v -run 'TestParentWatchdog_NotStartedUnderSetIO|TestStartParentWatchdog_StopsOnDone'` — both PASS, no race
- `go vet ./internal/serve/...` — PASS (exit 0)
- `gofmt -l internal/serve/mcp_watchdog*.go internal/serve/mcp_lifecycle.go` — flags ONLY `mcp_lifecycle.go`; the 4 new files are clean.

## Open items
- Test 3 (Linux `armParentDeathSignal` smoke) — NOT delivered — reason not stated in ledger — assessment: **soft skip but immaterial**. The function is trivial and compiles on both targets; a no-panic assertion adds near-zero coverage value. Not worth a HOLD.
- E2E reparent→exit test — SKIPPED — engineer cites need for a multi-process harness (build binary, spawn child, SIGKILL parent, poll pid) — assessment: **concrete and correctly deferred**. The spec itself (Test Plan, "What is hard to unit-test") defers this to the `e2e-validation` / `per-feature-smoke-coverage` family. Honest gap, not performative.

## Audit notes
- **gofmt-on-mcp_lifecycle.go claim is TRUE.** `git show main:internal/serve/mcp_lifecycle.go | gofmt -l` emits `<standard input>` (drift exists on main). `gofmt -d` on the current file shows the only diff is method-receiver alignment in the `mcpClientAdapter` block (lines 242-248: `Name()`/`Version()`/`Close()` need one more space to align with `Kinds()`/`Stream()`) — pre-existing, entirely unrelated to the watchdog gate at lines 33-44. The engineer did NOT introduce new gofmt drift. Their edit is gofmt-clean in isolation.
- **`os.Exit(0)` skips deferred cleanup** — specifically the debug-log `defer f.Close()` in `Run()` (mcp_lifecycle.go:28). Acceptable: this path runs ONLY after the parent is already dead (orphan teardown), losing a buffered debug-log tail on a backstop exit is harmless, and the spec's Secondary Defects #1 explicitly judged this acceptable. Noting per audit-catch checklist; not a defect.
- **No goroutine leak on the non-stdio path.** When `s.input != os.Stdin` the watchdog is never started, so there is nothing to leak; `done` is only created inside the gated branch. On the stdio path, `defer close(done)` ties the goroutine's life to `Run()` returning. Test 2 proves teardown.
- **Cheaper seam for the os.Exit branch was available but skipped (quality observation, not a blocker).** The `os.Exit` and `os.Getppid` calls are package-level, not injected. A small seam (package var `exitFunc = os.Exit` / `getppid = os.Getppid`) would have let a unit test drive the reparent→exit branch in-process without an e2e harness. The spec did defer the e2e test, and the engineer's framing ("single-process Go test cannot fake a reparent") is accurate *as written*, but it slightly overstates the constraint: the *exit decision logic* (ppid changed ⇒ exit) is testable behind a seam even if a true OS reparent is not. Low-cost, skipped. Flagging as a quality note, not a HOLD reason — severity is low and the gate test already protects the suite.
- **Scope is clean.** Diff touches only the spec's named files plus the expected one-line go.mod promotion. No drift.
