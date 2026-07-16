---
title: "Test CI red for days — data race between the parent-watchdog goroutine and a test's deferred seam-var restore"
slug: parent-watchdog-race-ci-red
type: bug
status: completed
severity: high
priority: high
domain: engineering
root_cause_class: race
created: 2026-07-16
tags: [ci, data-race, watchdog, serve, test-reliability]
relations:
  - target: agents-md-erased-by-snapshot-pointer-writer
    kind: related
  - target: opsrunner-keepalive-data-race
    kind: related
  - target: hero-mcp-orphan-no-parent-liveness
    kind: related
completed_at: 2026-07-16T20:46:17Z
---

# Test CI red for days — parent-watchdog goroutine races a test's deferred seam-var restore

## Kickoff

The `Test` CI job (`go test -race -count=1 ./...`) has been RED since 2026-07-13.
It is **not** a process-model problem — it is a genuine `-race` DATA RACE in
`internal/serve`. The parent-watchdog poll goroutine reads the package-level seam
vars `watchdogGetppid` / `singletonIsAlive` (`mcp_watchdog.go:48-49`) while
`TestParentWatchdog_IgnoresReparentWhileParentAlive`'s deferred cleanup restores
those same vars (`mcp_watchdog_test.go:135-136`) with no happens-before edge
between them. It "passed locally" only because the local run omitted `-race`; the
gate uses `-race`.

**Reproduce:** `go test -race -count=1 ./internal/serve/ -run TestParentWatchdog`
(fails reliably on `..._IgnoresReparentWhileParentAlive`).

**Root cause confirmed & reproduced.** The race lives ONLY in the test harness —
production never concurrently writes those vars. But the fix needs a minimal
production touch: `startParentWatchdog` must return a `stopped <-chan struct{}`
it closes on goroutine exit, so the one racing test can `close(done); <-stopped`
(join) before restoring the seam vars.

**Pick up at:** apply the fix in `## Suggested Fix Approach` — (1) add the
`stopped` return to `startParentWatchdog` (`internal/serve/mcp_watchdog.go`); the
production caller `mcp_lifecycle.go:78` needs no change (return discarded);
(2) in `_IgnoresReparentWhileParentAlive` only, capture `stopped` and join before
restore. Then verify with
`go test -race -count=10 ./internal/serve/ -run TestParentWatchdog` and
`go test -race -count=1 ./internal/serve/`.

**Do NOT:** add the `<-stopped` join to `..._ExitsOnPpidChange` or
`..._ExitsWhenOriginalParentDead` — their fake `watchdogExit` parks the goroutine
in `select {}` forever, so `stopped` never closes and the join would deadlock.
Those two are already race-free via their `exited` channel handshake. And do NOT
"fix" this with `t.Skip` or by dropping `-race` — the race is real and `-race` is
doing its job.

→ `.hero/planning/bugs/parent-watchdog-race-ci-red/spec.md`

**Files:** `internal/serve/mcp_watchdog.go:35-60`,
`internal/serve/mcp_watchdog_test.go:128-170`, `internal/serve/mcp_lifecycle.go:76-78`,
`.github/workflows/test.yml:31-32`

---

## Summary

### Categorization
| Attribute | Assessment |
|-----------|------------|
| **Criticality** | **high** — the race itself is test-harness-only, but it has kept the `Test` gate RED for days, so `go test -race -count=1 ./...` has stopped catching every other regression. The perma-red gate is the severity multiplier, independent of the race's tiny blast radius. |
| **Ease of Fix** | **easy** — add a `stopped` return channel to `startParentWatchdog`; join on it in one test before its cleanup restores the seam vars. ~10 lines, no behavior change. |
| **Caused by our codebase?** | **Yes** — a test-isolation defect introduced with the reparent-hardening in commit `3c38bcc` (2026-07-13). |
| **Needs more research?** | **No** — root cause confirmed and reproduced locally under `-race`; exact read/write sites and the missing happens-before edge are pinned. |

### Background
CI's `Test` workflow runs `go test -race -count=1 ./...` (`.github/workflows/test.yml:32`).
Since 2026-07-13 it fails in `internal/serve` with two `WARNING: DATA RACE` reports.
The failure was initially mis-attributed to a process-model / reparenting problem;
the CI logs show it is a `-race` detector finding, not a runtime crash or an
orphaned process.

### Analysis
`startParentWatchdog` spawns a poll goroutine that, on every ticker tick, reads
the package-level seam vars `watchdogGetppid` (`mcp_watchdog.go:48`) and
`singletonIsAlive` (`mcp_watchdog.go:49`). `TestParentWatchdog_IgnoresReparentWhileParentAlive`
shrinks the tick interval to 1ms, swaps those seam vars to simulate a false-positive
reparent, waits 200ms, and returns. On return its deferred cleanup **writes** the
same seam vars back to their originals (`mcp_watchdog_test.go:135-136`). Because
that test's branch (parent still alive → ignore) never calls `watchdogExit`, the
goroutine keeps ticking and reading right up to the moment the test returns — and
nothing synchronizes the goroutine's final read with the cleanup's write.

### Root Cause
The poll goroutine **outlives** the test's `defer close(done)` signal. `close(done)`
only *signals* the goroutine to stop; it does not *wait* for it to actually exit.
So the deferred seam-var restore runs concurrently with a goroutine still in its
`case <-ticker.C` branch reading those vars. Two vars race (two distinct addresses
in the report), matching `watchdogGetppid` and `singletonIsAlive`.

### Source
- `internal/serve/mcp_watchdog.go:40-59` — the poll goroutine; reads at `:48-49`.
- `internal/serve/mcp_watchdog_test.go:128-170` — `_IgnoresReparentWhileParentAlive`;
  the deferred writes at `:135-136`, and `defer close(done)` (signal-without-join)
  at `:159-160`.

### Fix Direction
Give the test a real happens-before join: `startParentWatchdog` returns a channel
it closes when the goroutine exits; the racing test closes `done` and receives on
that channel before its cleanup restores the seam vars. No `t.Skip`, no dropping
`-race`, no weakening of the watchdog's exit-on-parent-death guarantee.

---

## Problem Statement

CI `Test` job (`go test -race -count=1 ./...`) is RED. Exact evidence from the
failed run (gh run 29531318368), reproduced locally verbatim with
`go test -race -count=1 ./internal/serve/ -run TestParentWatchdog`:

```
WARNING: DATA RACE
Write at 0x000106008e60 by goroutine 29:
  ...TestParentWatchdog_IgnoresReparentWhileParentAlive.func1()
      internal/serve/mcp_watchdog_test.go:135 +0x7c
  runtime.deferreturn()
Previous read at 0x000106008e60 by goroutine 30:
  ...startParentWatchdog.func1()
      internal/serve/mcp_watchdog.go:48 +0x100
...
Goroutine 30 (finished) created at:
  ...startParentWatchdog()  internal/serve/mcp_watchdog.go:40
  ...TestParentWatchdog_IgnoresReparentWhileParentAlive()  internal/serve/mcp_watchdog_test.go:161
==================
WARNING: DATA RACE
Write at 0x000106008e48 by goroutine 29:
  ...func1()  internal/serve/mcp_watchdog_test.go:136 +0xb8
Previous read at 0x000106008e48 by goroutine 30:
  ...startParentWatchdog.func1()  internal/serve/mcp_watchdog.go:49 +0x138
--- FAIL: TestParentWatchdog_IgnoresReparentWhileParentAlive (0.20s)
FAIL
```

Mapping the addresses to source:
- `mcp_watchdog_test.go:135` writes `watchdogGetppid = origGetppid`; `mcp_watchdog.go:48`
  reads `now := watchdogGetppid()`. → race #1.
- `mcp_watchdog_test.go:136` writes `singletonIsAlive = origAlive`; `mcp_watchdog.go:49`
  reads `!singletonIsAlive(startPpid)`. → race #2.

**Why it "passed locally":** a plain `go test` (no `-race`) has no race detector,
so the concurrent read/write is invisible. The gate uses `-race`. Local
verification that does not match the gate's flags is false confidence — a
diagnostic trap this class of bug repeats. Confirmed here: `go test ./internal/serve/`
(no `-race`) is green; `go test -race ./internal/serve/` fails.

## Environment Details

- Gate: `.github/workflows/test.yml:31-32` → `go test -race -count=1 ./...`.
- Platform-relevant: the watchdog's portable ppid poll is the sole mechanism on
  darwin (`mcp_watchdog_other.go` — `armParentDeathSignal` is a no-op off Linux).
  The race is platform-independent (it is in shared Go code, not the OS path) and
  reproduces on darwin locally and on the CI linux runner.
- Reproduced locally: `go test -race -count=1 ./internal/serve/ -run TestParentWatchdog`
  → `--- FAIL: TestParentWatchdog_IgnoresReparentWhileParentAlive`.

---

## Root Cause Analysis

**Confirmed and reproduced.** Concurrency (test-isolation) defect.

`startParentWatchdog` (`mcp_watchdog.go:35`) launches a goroutine that loops on a
ticker and, each tick, reads the package-level seam vars `watchdogGetppid` and
`singletonIsAlive`:

```go
case <-ticker.C:
    now := watchdogGetppid()                                   // :48  read
    if now != startPpid && (now == 1 || !singletonIsAlive(startPpid)) {  // :49  read
        watchdogExit(0)
    }
```

These vars are package globals (`mcp_watchdog.go:20-23`, `mcp_singleton.go:44`),
documented as *"Production code never reassigns these."* In production that holds:
they are set once at init and only ever read, so the goroutine's unsynchronized
reads are safe — **there is no concurrent writer in production**.

The concurrent writer exists only in the **test harness**.
`TestParentWatchdog_IgnoresReparentWhileParentAlive` (`mcp_watchdog_test.go:128`)
uses those globals as a seam: it swaps them for fakes, runs the goroutine with a
1ms interval, then restores them in a deferred cleanup:

```go
defer func() {
    watchdogExit = origExit
    watchdogGetppid = origGetppid   // :135  write  ← races :48
    singletonIsAlive = origAlive    // :136  write  ← races :49
    parentWatchdogInterval = origInterval
}()
...
done := make(chan struct{})
defer close(done)                   // :159-160  signal-only, no join
startParentWatchdog(done)
```

The teardown sequence is the defect. Defers run LIFO, so `close(done)` (registered
last) runs first, then the restore (registered first) runs last. But `close(done)`
only **signals** the goroutine — it does not **wait** for it to exit. The goroutine
may still be inside `case <-ticker.C` executing `:48-49` when the restore defer
writes `:135-136`. No channel handshake, mutex, or join orders the goroutine's
final read against the restore write → data race.

**Why only this test races (the exit-branch tests do not):**
`_ExitsOnPpidChange` and `_ExitsWhenOriginalParentDead` drive the goroutine to
call the fake `watchdogExit`, which sends on an `exited` channel and then parks in
`select {}`. The test receives on `exited` before returning — and that receive
*happens-after* the goroutine's `:48-49` read that triggered it, and the parked
goroutine never touches the seam vars again. That `exited` handshake supplies the
happens-before edge, so their restore writes are ordered and race-free. Verified:
`go test -race -count=20 ./internal/serve/ -run 'TestParentWatchdog_ExitsOnPpidChange|TestParentWatchdog_ExitsWhenOriginalParentDead|TestStartParentWatchdog_StopsOnDone'`
→ `ok`, no race.

`_IgnoresReparentWhileParentAlive` is precisely the case with **no** such
handshake: its whole point is that the false-positive branch never fires
`watchdogExit`, so the goroutine loops until the test returns, with nothing to
order its last read against the cleanup. That is why it — and only it — trips the
detector. This is consistent with the race report naming only
`..._IgnoresReparentWhileParentAlive.func1` as the writer.

**Introduced by:** commit `3c38bcc` (2026-07-13, *"fix(serve): stop hero mcp
daemon dying mid-session; coexist instead of supersede"*), which added the
reparent-hardening (`singletonIsAlive` check at `:49`) and this false-positive
test. The original watchdog (`bcb9424`, 2026-06-02) had no such non-exiting,
still-looping test, so no race. The gate has been red since `3c38bcc` landed on
main, matching the observed 2026-07-13 start.

### Test-only race, production-touching fix — stated plainly

- **The race is a test-harness artifact.** Production `startParentWatchdog` reads
  seam vars that production never writes concurrently; there is no production race
  and no production correctness bug. The watchdog's real guarantee (exit when the
  original parent dies) is intact and is not what fails.
- **The fix is not purely test-file-local.** To kill the race, the test needs a
  *real* happens-before edge from the goroutine back to itself. Goroutine-count
  polling (as `waitForGoroutinesAtMost` does) does **not** establish one — the
  `-race` detector requires actual synchronization (a channel close/receive), not
  an observed count drop. The only place that edge can originate is the goroutine
  itself, so `startParentWatchdog` must expose a "stopped" signal. That is a
  minimal production signature change (add a return value) with **zero** production
  behavior change — the caller discards it.

---

## Code Flow (End to End)

1. `internal/serve/mcp_lifecycle.go:76-78` — production start path: `done := make(chan struct{}); defer close(done); startParentWatchdog(done)`. Runs only when `s.input == os.Stdin` (real daemon), never under the buffer-driven tests.
2. `internal/serve/mcp_watchdog.go:35-39` — `startParentWatchdog` arms the OS path, captures `startPpid`, snapshots `interval` — all on the caller goroutine.
3. `internal/serve/mcp_watchdog.go:40-59` — spawns the poll goroutine; each tick reads `watchdogGetppid()` (`:48`) and `singletonIsAlive(startPpid)` (`:49`).
4. `internal/serve/mcp_watchdog_test.go:143-157` — test swaps the seam globals and sets `parentWatchdogInterval = 1ms` (before the goroutine starts — safe, sequential).
5. `internal/serve/mcp_watchdog_test.go:159-161` — `defer close(done)`, then `startParentWatchdog(done)`; goroutine begins looping every 1ms.
6. `internal/serve/mcp_watchdog_test.go:164-169` — test waits 200ms (expecting no exit), then returns.
7. **Teardown (the race):** `close(done)` fires (signal only) while the goroutine may still be mid-tick at `mcp_watchdog.go:48-49`; the restore defer then writes `mcp_watchdog_test.go:135-136`. Reads (:48-49) and writes (:135-136) overlap with no happens-before → `WARNING: DATA RACE`.

---

## Key Files

### Production watchdog
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/serve/mcp_watchdog.go` | 35-60 | `startParentWatchdog` + poll goroutine; the racing **reads** at `:48-49`. Fix target: add a `stopped` return channel. |
| `internal/serve/mcp_watchdog.go` | 15, 20-23 | The `parentWatchdogInterval`, `watchdogExit`, `watchdogGetppid` seam globals — read here, never written in production. |
| `internal/serve/mcp_singleton.go` | 44, 79 | `singletonIsAlive` seam global — read by the watchdog (`:49`) *and* by the singleton-acquire path (`:79`). It is a genuinely **shared** seam, not watchdog-private. |
| `internal/serve/mcp_lifecycle.go` | 76-78 | The only production caller. Discards the return today; compiles unchanged after the signature change. |
| `internal/serve/mcp_watchdog_other.go` | 1-8 | `armParentDeathSignal` no-op off Linux; establishes the poll is the sole darwin mechanism. |

### Test harness
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/serve/mcp_watchdog_test.go` | 128-170 | `_IgnoresReparentWhileParentAlive` — the **only** racing test; deferred **writes** at `:135-136`, signal-only `defer close(done)` at `:159-160`. Fix target. |
| `internal/serve/mcp_watchdog_test.go` | 76-120, 176-219 | `_ExitsOnPpidChange`, `_ExitsWhenOriginalParentDead` — race-free via their `exited` handshake. **Must NOT** get the `<-stopped` join (their goroutine parks in `select {}` and never closes `stopped`). |
| `internal/serve/mcp_watchdog_test.go` | 53-68, 221-234 | `_StopsOnDone` + `waitForGoroutinesAtMost` — no seam restore, so not racing, but relies on noisy goroutine-count polling; a deterministic secondary improvement. |

### CI
| File | Lines | Relevance |
|------|-------|-----------|
| `.github/workflows/test.yml` | 31-32 | `go test -race -count=1 ./...` — the gate that has been red. |

---

## Secondary Defects

- **Noisy goroutine-count assertion.** `TestStartParentWatchdog_StopsOnDone`
  (`:53`) verifies goroutine teardown via `runtime.NumGoroutine()` polling
  (`waitForGoroutinesAtMost`, `:223`). It is not racing (it never restores seam
  vars), but goroutine counts are inherently noisy and could flake independently.
  The same `stopped` channel this fix adds converts it to a deterministic
  `<-stopped` join. Optional but recommended (see Test Plan).
- **No goroutine-leak join in the production caller.** `mcp_lifecycle.go:76-78`
  also only `close(done)`s without waiting. That is acceptable in production (the
  process is exiting anyway) and is explicitly out of scope — noted so a future
  reader does not mistake it for part of this fix.

---

## Notes

Themed finding — a broken verification signal, same class as
[[agents-md-erased-by-snapshot-pointer-writer]] (a guard that silently stopped
guarding) and a near-exact structural twin of [[opsrunner-keepalive-data-race]]
(a global fn var swapped by a test racing a leaked goroutine, on the same
`-race` CI gate). The `Test` gate being perma-red for days is itself the severity
multiplier: while red, `go test -race -count=1 ./...` catches **no** regressions
in any package — the signal is not just wrong, it is off. Restoring the gate
matters independent of this race's (tiny) blast radius.

**Divergence from the opsrunner precedent — why not "eliminate the global."**
`opsrunner-keepalive-data-race` was fixed by moving the mutable globals onto the
`Runner` struct as per-instance immutable fields. That is the ideal fix when a
struct exists to home them. It does **not** transfer cleanly here: (1)
`startParentWatchdog` is a free function with no per-instance home; (2)
`singletonIsAlive` is a *shared* seam used by the singleton-acquire path
(`mcp_singleton.go:79`), so it is legitimately package-scoped and cannot simply
become a watchdog field without rippling into that subsystem. Parameterizing all
four seams into a config struct is possible but is a larger, more invasive change
than the bug warrants. The join fix is the surgical option and directly closes the
specific defect (a goroutine outliving cleanup with no happens-before edge). The
"parameterize the seams" alternative is recorded below for the deliverer.

---

## Root Cause Classification

```yaml
root_cause_class: race
severity: high
```

- **race** — timing-dependent unsynchronized read/write between the poll goroutine
  and the test's deferred cleanup; surfaces only under `-race`.
- **high** — test-harness-scoped race, but it has held the `Test` gate red for
  days, disabling regression detection across the whole module.

---

## Anchor Check

`hero_anchor` (context: the channel-join fix, production behavior unchanged,
watchdog exit-on-parent-death guarantee preserved) returned the mission plus the
`harness-changes-cover-all-targets` tripwire. That tripwire governs harness-facing
install content (CLAUDE.md/AGENTS.md, commands, agents, skills) and does **not**
apply to this Go test/runtime fix — no install target, instruction file, or
routing surface is touched. No tripwire forbids the proposed direction.

---

## Suggested Fix Approach

### 1. `internal/serve/mcp_watchdog.go` — `startParentWatchdog` returns a "stopped" channel

**Before:**
```go
func startParentWatchdog(done <-chan struct{}) {
	armParentDeathSignal() // platform-specific; no-op on darwin

	startPpid := watchdogGetppid()
	interval := parentWatchdogInterval
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				now := watchdogGetppid()
				if now != startPpid && (now == 1 || !singletonIsAlive(startPpid)) {
					watchdogExit(0)
				}
			}
		}
	}()
}
```

**After:**
```go
// startParentWatchdog ... (existing doc comment retained) ...
//
// It returns a channel that is closed once the poll goroutine has fully
// exited (after done is closed). Production ignores it — the process is
// tearing down anyway — but any test that swaps the package-level seam vars
// (parentWatchdogInterval/watchdogGetppid/singletonIsAlive/watchdogExit) MUST
// receive from it before restoring those vars, so the goroutine's final seam
// read happens-before the restore write. Without that join, -race reports a
// data race between the poll goroutine and the test's deferred cleanup.
func startParentWatchdog(done <-chan struct{}) <-chan struct{} {
	armParentDeathSignal() // platform-specific; no-op on darwin

	startPpid := watchdogGetppid()
	interval := parentWatchdogInterval
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				now := watchdogGetppid()
				if now != startPpid && (now == 1 || !singletonIsAlive(startPpid)) {
					watchdogExit(0)
				}
			}
		}
	}()
	return stopped
}
```

**Why:** the goroutine now publishes a completion signal (`close(stopped)` via
`defer`, fired only after the loop returns on `<-done`). A test can receive on it
to establish a genuine happens-before edge between the goroutine's last seam read
and its own cleanup. No behavior change: same arming, same poll condition, same
`watchdogExit(0)` on genuine orphan.

### 2. `internal/serve/mcp_lifecycle.go` — no change required

**Current (`:76-78`), unchanged:**
```go
done := make(chan struct{})
defer close(done)
startParentWatchdog(done)
```

**Why:** Go permits discarding a function's return value in an expression
statement, so this compiles as-is against the new signature. The production path
deliberately does not join — the process is exiting and never rewrites the seam
vars, so there is nothing to order. (Optional: assign to `_ = startParentWatchdog(done)`
for explicitness; not required.)

### 3. `internal/serve/mcp_watchdog_test.go` — join before restore in `_IgnoresReparentWhileParentAlive` ONLY

**Before (`:159-161`):**
```go
	done := make(chan struct{})
	defer close(done)
	startParentWatchdog(done)
```

**After:**
```go
	done := make(chan struct{})
	stopped := startParentWatchdog(done)
	// Stop the goroutine AND wait for it to exit before the restore defer
	// (registered earlier in this function, so it runs later) rewrites the
	// seam vars. This receive is the happens-before edge that orders the
	// goroutine's final read (mcp_watchdog.go:48-49) ahead of the restore
	// write (:135-136) — the -race failure this closes. Signalling with
	// close(done) alone does not wait, which is exactly the original bug.
	defer func() {
		close(done)
		<-stopped
	}()
```

**Why:** defers run LIFO. The seam-restore `defer func(){...}()` is registered at
the top of the function, so it runs **last**; this new `defer func(){ close(done); <-stopped }()`
is registered later, so it runs **first** — the goroutine is fully stopped and
joined before any seam var is restored. Race eliminated at its root (the goroutine
no longer outlives the cleanup). The watchdog's guarantee is untouched; the test
still asserts no exit fires on a bare reparent while the parent is alive.

**Do NOT** apply this to `_ExitsOnPpidChange` (`:76`) or
`_ExitsWhenOriginalParentDead` (`:176`): their fake `watchdogExit` sends on
`exited` then blocks in `select {}` forever, so the goroutine never returns and
`stopped` never closes — a `<-stopped` there would deadlock the test. They are
already race-free via the `exited` handshake; leave their `defer close(done)` as-is.

### 4. (Secondary, recommended) `TestStartParentWatchdog_StopsOnDone` — deterministic join

**Before (`:53-68`):**
```go
func TestStartParentWatchdog_StopsOnDone(t *testing.T) {
	before := runtime.NumGoroutine()

	done := make(chan struct{})
	startParentWatchdog(done)

	close(done)

	if got := waitForGoroutinesAtMost(before, 2*time.Second); got > before {
		t.Fatalf("watchdog goroutine did not exit after done closed: before=%d now=%d", before, got)
	}
}
```

**After:**
```go
func TestStartParentWatchdog_StopsOnDone(t *testing.T) {
	done := make(chan struct{})
	stopped := startParentWatchdog(done)

	close(done)

	select {
	case <-stopped:
		// goroutine exited cleanly
	case <-time.After(2 * time.Second):
		t.Fatal("watchdog goroutine did not exit after done closed")
	}
}
```

**Why:** replaces noisy `runtime.NumGoroutine()` polling with a deterministic wait
on the goroutine's own completion signal, and directly exercises the new `stopped`
channel. If nothing else references `waitForGoroutinesAtMost` (`:223`) or the
`runtime` import after this change, remove them to avoid dead code. This is a
test-quality improvement, not required to kill the race — apply if in scope.

---

## Test Plan

### Existing test review
- `internal/serve/mcp_watchdog_test.go:128` — `TestParentWatchdog_IgnoresReparentWhileParentAlive`:
  the failing test. Fixed by the join in change 3; still asserts no exit on a bare
  reparent while the parent is alive.
- `internal/serve/mcp_watchdog_test.go:76` / `:176` — `_ExitsOnPpidChange` /
  `_ExitsWhenOriginalParentDead`: exercise the exit branch; race-free via `exited`
  handshake. **No change** (and must not receive the join).
- `internal/serve/mcp_watchdog_test.go:53` — `_StopsOnDone`: goroutine-teardown
  test; optionally upgraded to a deterministic join (change 4).
- `internal/serve/mcp_watchdog_test.go:18` — `_NotStartedUnderSetIO`: gate that the
  watchdog stays off under buffer-driven tests. Unaffected.

### Test changes needed
- Change 3 (required): join-before-restore in `_IgnoresReparentWhileParentAlive`.
- Change 4 (recommended): deterministic `<-stopped` join in `_StopsOnDone`; drop
  `waitForGoroutinesAtMost` / `runtime` import if unused afterward.
- No new test file needed — the fix is validated by the existing tests now passing
  under `-race`, which is the exact signal that was red. The reparent-hardening
  behavior (`3c38bcc`) already has dedicated coverage in the four tests above.

### Verification commands
1. `go test -race -count=10 ./internal/serve/ -run TestParentWatchdog` → all pass,
   no `WARNING: DATA RACE` (repetition guards against a nondeterministic residual).
2. `go test -race -count=1 ./internal/serve/` → green (the whole `serve` package
   under the detector).
3. `go test -race -count=1 ./...` → green (the exact CI gate) — the real
   acceptance: push and confirm the `Test` job goes green.
4. `go build ./...` → clean (signature change compiles across all call sites).

### Regression scope
- `startParentWatchdog` signature change touches exactly one production caller
  (`mcp_lifecycle.go:78`, return discarded) and the test call sites. `go build ./...`
  covers compile-completeness.
- Production runtime behavior is unchanged: arming, poll condition, and
  exit-on-genuine-orphan are byte-for-byte identical; only a completion channel is
  added. The exit-on-parent-death guarantee is preserved.
- Risk of deadlock if the join is mis-applied to the exit-branch tests — explicitly
  called out in changes 3 and the fix notes; those tests must keep `defer close(done)`.

---

## Recap
The `Test` CI gate has been red since 2026-07-13 due to a `-race` data race: the
parent-watchdog poll goroutine reads the package seam vars `watchdogGetppid` /
`singletonIsAlive` (`mcp_watchdog.go:48-49`) while `_IgnoresReparentWhileParentAlive`'s
deferred cleanup restores them (`mcp_watchdog_test.go:135-136`), because
`close(done)` signals the goroutine without waiting for it to exit. The race is
test-harness-only, but the fix is minimally production-touching: `startParentWatchdog`
returns a `stopped` channel the racing test joins on before restoring the seam
vars. Severity is high not from blast radius but because a perma-red `-race` gate
has stopped catching every other regression for days.

## Completion Ledger

Delivered 2026-07-16. Stack: Go. The fix matches the spec's Suggested Fix Approach
(surgical channel-join), test-only in effect — production behavior unchanged.

| Item | Status | Evidence |
|---|---|---|
| Root cause reproduced under `-race` | DONE | `go test -race ./internal/serve/ -run TestParentWatchdog` → `WARNING: DATA RACE` + `FAIL`; same command without `-race` → `ok`. The "passes locally" trap confirmed. |
| `startParentWatchdog` returns a `stopped` join channel | DONE | `internal/serve/mcp_watchdog.go`: signature now `func startParentWatchdog(done <-chan struct{}) <-chan struct{}`; goroutine does `defer close(stopped)`; returns `stopped`. Doc comment explains the happens-before-join rationale. |
| Production caller unchanged | DONE | `internal/serve/mcp_lifecycle.go:78` still calls `startParentWatchdog(done)` and discards the return — compiles unchanged, zero behavior change. Watchdog's exit-on-parent-death guarantee untouched. |
| Racing test joins before seam-restore | DONE | `mcp_watchdog_test.go` `TestParentWatchdog_IgnoresReparentWhileParentAlive`: `stopped := startParentWatchdog(done)` then `defer func(){ close(done); <-stopped }()`, registered after the seam-restore defer so LIFO orders close → join → restore. |
| Exit-branch tests left alone | DONE | They block in `watchdogExit`'s `select{}` so the goroutine never returns; a `<-stopped` join would deadlock. Their `exited`-channel handshake already orders them race-free (spec §Root Cause). Not modified. |
| Race fixed (repeated) | DONE | `go test -race -count=10 ./internal/serve/ -run TestParentWatchdog` → `ok`; `go test -race -count=1 ./internal/serve/` → `ok`. |
| Full CI gate green | DONE | `go test -race -count=1 ./...` (the exact `.github/workflows/test.yml` command) → all packages `ok`, zero failures. First green run of the gate since it went red ~2026-07-13. |

### Exercise-the-feature check

- [x] Exercised: reproduced the CI-red condition locally under `-race` (FAIL), applied the fix, re-ran the exact CI command `go test -race -count=1 ./...` → fully green. The gate that was red for days now passes.
