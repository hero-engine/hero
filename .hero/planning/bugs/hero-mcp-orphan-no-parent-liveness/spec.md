---
title: "hero mcp stdio servers orphan and accumulate — no parent-liveness backstop"
type: bug
status: completed
priority: medium
severity: low
domain: engineering
root_cause_class: design
tags: [mcp, stdio, process-lifecycle, orphan, watchdog, serve]
created: 2026-06-02
---

# hero mcp stdio servers orphan and accumulate — no parent-liveness backstop

## Summary

### Categorization
| Attribute | Assessment |
|-----------|------------|
| **Criticality** | low — operational annoyance / resource leak. No correctness, data, or security impact. Each idle server holds a goroutine and a few MB RSS; they accumulate over days. |
| **Ease of Fix** | moderate — a self-contained watchdog goroutine in one function plus a small build-tagged file. The conceptual care (test-mode gating, clean shutdown, darwin/linux split) is what makes it moderate rather than easy. |
| **Caused by our codebase?** | Yes — design gap. hero's stdio server has no mechanism to detect a dead parent; it relies solely on stdin EOF, which the client controls and does not always deliver. |
| **Needs more research?** | No — root cause is confirmed against source. The only un-unit-testable part (true reparent → exit) is called out honestly in the Test Plan. |

### Background
`hero mcp` runs as a JSON-RPC stdio server, launched as a child of the AI client
(Claude Code). On the user's machine, **14 orphaned `hero mcp` processes** had
accumulated, the oldest **2 days old**, each from a past client session. They never
exit on their own. The user must `kill $(pgrep -f 'hero mcp')` manually. Worse, each
new build (`go install`) leaves the stale child running the old binary, forcing a
manual `/mcp` reconnect to pick up the new one.

### Analysis
The mcp server's `Run()` loop has exactly one exit condition: `bufio.Scanner.Scan()`
returning false, which happens only on **stdin EOF or a read error**. EOF on hero's
stdin fires only when *all write ends of the stdin pipe close*. hero never holds a
write end — the write end lives only in the client process. So if the client exits
without closing the pipe (crash, SIGKILL, or handing the fd to a process that
outlives the session), hero's stdin never reaches EOF. hero gets reparented to
`launchd` (ppid → 1 on macOS) and blocks forever in `scanner.Scan()`.

### Root Cause
**Design / process-lifecycle gap.** The stdio server has no parent-liveness
detection. It depends entirely on a shutdown signal (stdin EOF) that hero cannot
generate itself and that the client does not reliably deliver on session teardown.
This is not a code typo — every line of `Run()` is individually correct. The flaw is
the *absence* of a backstop: the server has no way to notice "my parent is gone, I
should exit."

### Source
- `internal/serve/mcp_lifecycle.go` — `Run()` (lines 19–59). The scanner loop; the
  sole shutdown path.
- `internal/serve/mcp.go` — `MCPServer` struct + `SetIO` (lines 29–87). `input`
  defaults to `os.Stdin` in production; tests override it via `SetIO`.
- `internal/cli/mcp.go` — `runMCP` (lines 35–63). Calls `mcpSrv.Run()` with no
  signal/context wrapping.

### Fix Direction
Add a **parent-liveness watchdog** to the stdio `Run()` path — the only lever hero
owns. A goroutine watches `os.Getppid()`; when the parent changes (reparented to
init/launchd ⇒ parent died), it exits the process. EOF stays the fast path; the
watchdog is the backstop that only fires when the parent is already dead. Gate it to
real stdio mode so in-process tests never trip it. Optionally add a build-tagged
Linux `prctl(PR_SET_PDEATHSIG)` fast path for poll-free delivery, with a darwin
no-op so the poll covers macOS.

---

## Problem Statement

`hero mcp` is configured by `hero install` as a direct exec (`command: "hero",
args: ["mcp"]`) — no shell wrapper — so the mcp process's parent **is** the client
(Claude Code). The client launches it at session start, talks JSON-RPC over the
pipe, and is expected to close the pipe (and thus deliver stdin EOF) at session end.

When the client does **not** close the pipe — process crash, `SIGKILL`, OS teardown
that reaps the client without flushing fd closes, or an fd handed to a surviving
grandchild — hero's stdin never reaches EOF. The server sits in `scanner.Scan()`
indefinitely after being reparented to `launchd`. Over days of sessions, these pile
up: 14 orphans, oldest 2 days old, on the reporter's machine.

Reproduction (conceptual; macOS):
1. Launch `hero mcp` as a child with a pipe on its stdin.
2. `SIGKILL` the parent (simulating a client crash) without closing the pipe.
3. Observe `hero mcp` reparented to pid 1, still alive, blocked in `Scan()`.

Symptom chain reported by the user:
- `kill $(pgrep -f 'hero mcp')` is needed to clean up.
- After `go install`, the stale child keeps running the old binary; a manual `/mcp`
  reconnect is required to switch to the freshly built one. (Claude Code does not
  auto-respawn a dead stdio server, which is also why a time-based idle timeout would
  be the *wrong* fix — see Notes.)

---

## Environment Details

- macOS (Darwin 25.5.0). On macOS, orphaned children reparent to `launchd` (pid 1).
  Linux behaves analogously (reparent to init / a subreaper).
- `hero mcp` is installed as a direct exec per `internal/install/mcp.go` — relevant
  because it guarantees `os.Getppid()` returns the *client's* pid, making ppid a
  reliable liveness signal. (If a shell wrapper sat in between, ppid would track the
  wrapper, not the client, and the signal would be weaker.)
- `golang.org/x/sys v0.43.0` is already a dependency (currently `// indirect`), and
  provides `unix.Prctl` for the optional Linux fast path.

---

## Root Cause Analysis

All claims below are **read** (verified against source in this session), not assumed.

**Claim 1 — The only shutdown path is stdin EOF.** `read`
`internal/serve/mcp_lifecycle.go:33-58`: `scanner := bufio.NewScanner(s.input)`
then `for scanner.Scan() { ... }`, `return scanner.Err()`. `Scan()` returns false
only on EOF or error. No `select`, no context, no signal handler, no ppid check
anywhere in the loop or the function.

**Claim 2 — `input` is `os.Stdin` in production.** `read` `internal/serve/mcp.go:59`:
`NewMCPServer` sets `input: os.Stdin`. `runMCP` (`internal/cli/mcp.go:61-62`) builds
the server and calls `Run()` directly — no `signal.NotifyContext`, no wrapper. So in
production, `s.input` is the read end of the client→hero pipe and EOF is the only way
out.

**Claim 3 — EOF is entirely client-controlled; hero cannot generate it.** This is
the crux. hero's fd 0 is the *read* end of the stdin pipe. EOF fires only when all
*write* ends close. The write end lives in the client; hero never holds one.
Therefore hero has no mechanism to make EOF happen — if the client doesn't close the
pipe, hero blocks forever. This is an OS-level fact about pipes, consistent with the
code: nothing in the mcp path opens, dups, or closes a write end of fd 0.

**Claim 4 — The "leak stdin to subprocesses" theory is FALSE.** `read`. The mcp
server does spawn subprocesses:
- `internal/serve/mcp_tools.go:2783` — `exec.Command("git", "-C", ..., "rev-parse", ...).Output()`
- `internal/serve/workers.go:139,146,153` — `exec.Command("hero", "drift"/"coverage"/"contract", ...)`
- `internal/runner/tools.go:158,181,212` — `exec.Command("sh","-c",...)`, `grep`, `hero`

A grep for `.Stdin =` across `internal/` (verified) shows **none of these sites set
`cmd.Stdin`**. Go's `os/exec` therefore wires their stdin to `/dev/null` and sets
`O_CLOEXEC` on inherited fds, so children never inherit hero's stdin pipe. The sites
that *do* set `cmd.Stdin = os.Stdin` (`internal/cli/skill.go:312`,
`internal/cli/test.go:176,229`, `internal/skills/runner.go:79,88`,
`internal/cli/propose_shim.go:100`) are all **CLI-foreground** commands, not on the
mcp server path. And even if a child *did* inherit fd 0, it would inherit the *read*
end — which cannot prevent EOF (only an open *write* end can). So subprocess fd
hygiene is **not** the bug and **not** the fix. Documented here so it isn't
re-investigated.

**Claim 5 — No parent-liveness or signal handling exists in the mcp path.** `read`.
A grep for `Getppid|PDEATHSIG|Pdeathsig|prctl|Prctl|signal\.|SIGTERM|NotifyContext`
across `internal/serve`, `internal/cli`, `internal/runner`: every hit is in the
**HTTP `serve` daemon** (`internal/cli/serve.go:162` `signal.NotifyContext(...,
SIGINT, SIGTERM)`, `internal/cli/serve_lifecycle.go` SIGTERM/SIGKILL stop logic) —
a *separate* long-lived daemon, not the stdio mcp server. The stdio path has zero
liveness handling.

**Claim 6 — Direct exec makes ppid a reliable signal.** `read`
`internal/install/mcp.go:144-150` (Claude/generic), `:256-262` (Codex toml): all
write `command: "hero", args: ["mcp"]` with no shell wrapper. So the mcp process's
parent is the client; `os.Getppid()` changing away from its startup value is a
reliable "parent died" signal.

**Conclusion.** Confirmed root cause: a **design-level process-lifecycle gap**. The
stdio server has no backstop for a dead parent and depends solely on a client-driven
EOF that is not reliably delivered. Classification: `design`. Severity: `low`
(resource leak / operational friction, no correctness impact).

---

## Code Flow (End to End)

1. `internal/install/mcp.go:144-150` — installer writes the client config
   `command: "hero", args: ["mcp"]` (direct exec, no wrapper).
2. Client (Claude Code) spawns `hero mcp` at session start; the mcp process's parent
   is the client; its stdin is the read end of a client→hero pipe.
3. `internal/cli/mcp.go:35-62` — `runMCP` resolves project root, loads config, builds
   `MCPServer` via `NewMCPServerWithFilter`, calls `mcpSrv.Run()`. No signal/context
   wrapping.
4. `internal/serve/mcp.go:54-66` — `NewMCPServer` sets `input: os.Stdin`,
   `output: os.Stdout`.
5. `internal/serve/mcp_lifecycle.go:33-37` — `Run()` builds `bufio.NewScanner(s.input)`
   and enters `for scanner.Scan()`.
6. Steady state: client writes JSON-RPC lines; `Scan()` returns true each time;
   `handleRequest` dispatches. Normal.
7. **Normal teardown (works):** client closes the pipe on session end → write end
   gone → `Scan()` returns false → `Run()` returns → process exits. Correct path.
8. **Buggy teardown (the bug):** client dies *without* closing the pipe (crash,
   SIGKILL, fd handed to a survivor). Write end is never closed. `Scan()` blocks
   forever. Process is reparented to launchd (`ppid → 1`). It never exits.
9. Result: orphan accumulates. Next `go install` builds a new binary, but the orphan
   keeps serving the old one until the user manually kills it and `/mcp` reconnects.

---

## Key Files

### MCP stdio server (the bug surface)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/serve/mcp_lifecycle.go` | 19–59 | `Run()` — the scanner loop; sole shutdown path; where the watchdog must be started and stopped. |
| `internal/serve/mcp.go` | 29–87 | `MCPServer` struct; `NewMCPServer` defaults `input` to `os.Stdin`; `SetIO` overrides it for tests. The `s.input == os.Stdin` gate keys off this. |
| `internal/cli/mcp.go` | 35–63 | `runMCP` entry point; calls `Run()` with no signal handling. |

### Install (proves direct-exec ppid reliability)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/install/mcp.go` | 144–150, 256–262 | Writes `command: "hero", args: ["mcp"]` — no shell wrapper. ppid == client pid. |

### Subprocess sites (the FALSE theory — documented, not the fix)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/serve/mcp_tools.go` | 2783 | `git rev-parse` — no `cmd.Stdin`; child stdin is `/dev/null`. |
| `internal/serve/workers.go` | 139, 146, 153 | `hero drift/coverage/contract` — no `cmd.Stdin`. |
| `internal/runner/tools.go` | 158, 181, 212 | `sh -c`, `grep`, `hero` — no `cmd.Stdin`. |

### Contrast: HTTP daemon (where signal handling DOES live)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/serve.go` | 162 | `signal.NotifyContext(..., SIGINT, SIGTERM)` — separate daemon, not the stdio mcp server. |
| `internal/cli/serve_lifecycle.go` | 22–93 | SIGTERM→SIGKILL stop logic for the daemon. Confirms the pattern is absent from mcp. |

### Test harness (gating constraint)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/serve/mcp_test.go` | 22–60 | `sendRecv`/`sendMulti` call `srv.SetIO(bytes.Buffer, ...)` then `srv.Run()` in the test process. The watchdog MUST NOT fire here, or it would `os.Exit` the test runner. |

---

## Root Cause Classification

```yaml
root_cause_class: design
severity: low
```

- **design** — the bug is the *absence* of a parent-liveness backstop, not a logic
  error in any existing line. Every line of `Run()` is individually correct; the spec
  (implicit) never accounted for "client dies without closing the pipe."
- **severity: low** — resource leak + operational friction. No correctness, data, or
  security impact. A single orphan is harmless; the problem is accumulation and the
  stale-binary-after-rebuild annoyance.
- **priority: medium** (argued) — above `low` because it hits the *developer's own
  inner loop daily* (every `go install` during Hero development leaves a stale child
  needing a manual `/mcp`), and Hero's whole value proposition is friction-free agent
  sessions — an accumulating orphan pile is exactly the kind of papercut that erodes
  trust in the tool. Not `high` because there's a trivial manual workaround
  (`pkill -f 'hero mcp'`) and zero data/correctness risk.

---

## Suggested Fix Approach

The fix is a parent-liveness watchdog started inside `Run()`, gated to real stdio
mode, torn down when `Run()` returns. Below, *Before* is copied from source; *After*
shows the exact change.

### Change 1 — start/stop the watchdog in `Run()`

**File:** `internal/serve/mcp_lifecycle.go`, function `Run()`.

**Before** (lines 33–58):
```go
	scanner := bufio.NewScanner(s.input)
	// Allow large messages (1MB)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			s.logDebug("→ PARSE ERROR: %s", line)
			s.sendError(nil, ErrCodeParse, "Parse error")
			continue
		}

		s.logDebug("→ %s (id=%s)", req.Method, string(req.ID))
		if req.Params != nil {
			s.logDebug("  params: %s", string(req.Params))
		}

		s.handleRequest(&req)
	}

	return scanner.Err()
```

**After:**
```go
	// Parent-liveness backstop. Stdin EOF is the fast, correct shutdown
	// path, but the client controls EOF and may die without closing the
	// pipe (crash, SIGKILL, fd handed to a survivor), leaving us blocked
	// in Scan() forever and reparented to launchd/init. The watchdog
	// notices the reparent and exits. Gated to real stdio mode so tests
	// that drive Run() with a bytes.Buffer never start it. Stopped when
	// Run() returns via the done channel.
	if s.input == os.Stdin {
		done := make(chan struct{})
		defer close(done)
		startParentWatchdog(done)
	}

	scanner := bufio.NewScanner(s.input)
	// Allow large messages (1MB)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			s.logDebug("→ PARSE ERROR: %s", line)
			s.sendError(nil, ErrCodeParse, "Parse error")
			continue
		}

		s.logDebug("→ %s (id=%s)", req.Method, string(req.ID))
		if req.Params != nil {
			s.logDebug("  params: %s", string(req.Params))
		}

		s.handleRequest(&req)
	}

	return scanner.Err()
```

**Why:** Introduces the only lever hero owns (watch its own ppid) without changing the
normal EOF path. The `s.input == os.Stdin` gate ensures the watchdog only runs when
hero is a real stdio child; tests that call `SetIO(bytes.Buffer, ...)` never enter
this branch. `defer close(done)` ties the watchdog's lifetime to `Run()` returning,
so the normal EOF exit cleanly stops the goroutine (no leaked ticker).

### Change 2 — portable watchdog (new file)

**File (new):** `internal/serve/mcp_watchdog.go`

```go
package serve

import (
	"os"
	"time"
)

// parentWatchdogInterval is how often the portable poll checks ppid.
// 30s is a deliberate tradeoff: an orphan lives at most ~30s past its
// parent's death, which is invisible operationally, while the poll cost
// is one getppid() syscall per tick — negligible.
const parentWatchdogInterval = 30 * time.Second

// startParentWatchdog launches the parent-liveness backstop. It first
// arms the OS-native fast path (Linux: PR_SET_PDEATHSIG; darwin: no-op),
// then runs a portable ppid poll that covers platforms without a native
// signal (notably macOS). The poll exits the process when our parent
// changes from its startup value — i.e. we've been reparented to
// launchd/init because the original parent died.
//
// It fires ONLY when the parent is already dead, so it can never disrupt
// a live session.
func startParentWatchdog(done <-chan struct{}) {
	armParentDeathSignal() // platform-specific; no-op on darwin

	startPpid := os.Getppid()
	go func() {
		ticker := time.NewTicker(parentWatchdogInterval)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if os.Getppid() != startPpid {
					// Parent reparented away (died). Exit cleanly.
					os.Exit(0)
				}
			}
		}
	}()
}
```

**Why:** Keying on *change* of ppid (not a hardcoded `== 1`) is robust across macOS
(launchd) and Linux (init or a subreaper). The poll is the baseline that covers
darwin where there's no native parent-death signal. `done` gives the goroutine a
clean exit when `Run()` returns normally on EOF.

### Change 3 — Linux fast path (build-tagged)

**File (new):** `internal/serve/mcp_watchdog_linux.go`

```go
//go:build linux

package serve

import (
	"golang.org/x/sys/unix"
	"syscall"
)

// armParentDeathSignal requests SIGTERM when our parent dies. This is
// instant and poll-free on Linux. The portable ticker still runs as a
// backstop (and to cover the race where the parent dies between fork and
// this call). Errors are non-fatal — the poll covers us either way.
func armParentDeathSignal() {
	_ = unix.Prctl(unix.PR_SET_PDEATHSIG, uintptr(syscall.SIGTERM), 0, 0, 0)
}
```

**File (new):** `internal/serve/mcp_watchdog_other.go`

```go
//go:build !linux

package serve

// armParentDeathSignal is a no-op on non-Linux platforms (notably
// darwin), which have no equivalent of PR_SET_PDEATHSIG. The portable
// ppid poll in startParentWatchdog is the sole mechanism there.
func armParentDeathSignal() {}
```

**Why:** On Linux, `PR_SET_PDEATHSIG` delivers `SIGTERM` the instant the parent dies
— no poll latency. Default Go behavior terminates the process on an unhandled
SIGTERM, which is the desired outcome. macOS has no equivalent, so the no-op variant
keeps the build green and the poll does the work. `golang.org/x/sys` is already a
dependency; this promotes it from `// indirect` to direct (a one-line `go mod tidy`
change).

> **Note on `SysProcAttr.Pdeathsig`:** Go's `SysProcAttr.Pdeathsig` sets the death
> signal for a *child you spawn*, not for the current process watching its *own*
> parent. hero is the child here, so it must call `prctl` on itself directly — hence
> `unix.Prctl`, not `SysProcAttr`. (The proposed fix direction flagged this; it is
> correct.)

### Imports
`mcp_lifecycle.go` already imports `os` (line 8) — no new import needed for Change 1.

---

## Test Plan

### Existing test review
| File | Coverage | Relevance |
|------|----------|-----------|
| `internal/serve/mcp_test.go:22-60` (`sendRecv`, `sendMulti`) | Drive `Run()` with a `bytes.Buffer` via `SetIO`, then assert on JSON-RPC responses. | These are the tests the watchdog must NOT disturb. Because `s.input` is a buffer (not `os.Stdin`), the gate in Change 1 keeps the watchdog off. Run the full `internal/serve` suite to confirm no regression. |
| `internal/serve/mcp_test.go:339-390` and siblings | Many call `srv.Run()` in-process. | Same gating guarantee applies. |

### Test changes needed

1. **`TestParentWatchdog_NotStartedUnderSetIO` (new, `internal/serve/mcp_watchdog_test.go`).**
   The honest, deterministic test. Construct an `MCPServer`, `SetIO` a `bytes.Buffer`
   with one valid request, run `Run()`, and assert it returns promptly without
   exiting the process. Since `SetIO` makes `s.input != os.Stdin`, the watchdog branch
   is never entered. This directly verifies the gate that protects the entire test
   suite. (If the gate regressed, the watchdog goroutine would either leak or, worse,
   a future bug could `os.Exit` the runner — so this test is the guardrail.)

2. **`TestStartParentWatchdog_StopsOnDone` (new).** Call `startParentWatchdog(done)`
   directly with a `done` channel, then `close(done)`, and assert (via a short sleep +
   goroutine-count or a leak detector like `goleak`, if adopted) that the ticker
   goroutine has exited. Verifies clean teardown — the `defer close(done)` path. This
   does not exercise the `os.Exit` branch (ppid is stable in-process), which is the
   point: it proves the goroutine is well-behaved when the parent is alive.

3. **`armParentDeathSignal` smoke (Linux only, build-tagged test).** Assert
   `armParentDeathSignal()` does not panic and returns. We cannot meaningfully assert
   the signal *fires* without forking and killing a parent — see below.

### What is hard to unit-test (called out honestly)
The actual payoff — "parent dies → orphan exits" — requires a real reparent, which a
single-process Go test cannot fake: `os.Getppid()` is stable within the test binary,
and we will not `os.Exit` the runner to prove it. Faithful coverage would need an
**integration/e2e harness** that:
1. Builds the `hero` binary.
2. Spawns `hero mcp` as a child via a pipe, from an intermediate parent process.
3. `SIGKILL`s the intermediate parent without closing the pipe.
4. Polls (up to ~35s, > one watchdog tick) that the `hero mcp` pid is gone.

This belongs in the repo's e2e suite (the `e2e-validation` / `per-feature-smoke-coverage`
family), not a unit test. If that harness isn't built as part of this fix, the unit
tests above + manual verification (reproduction steps in Problem Statement) are the
honest coverage line, and this gap should be noted in the delivery.

### Regression scope
- **Test suite safety:** the only way this fix breaks existing tests is if the
  `s.input == os.Stdin` gate is wrong and the watchdog starts under `SetIO`. Test 1
  above guards exactly that. Run `go test ./internal/serve/...`.
- **Live sessions:** the watchdog fires only when ppid has already changed (parent
  dead). It cannot terminate a live session — there is no time-based component that
  could misfire on a quiet-but-alive session.
- **Build matrix:** the new build-tagged files must compile on both `darwin` and
  `linux`. Verify `GOOS=linux go build ./...` and `GOOS=darwin go build ./...`.
- **go.mod:** promoting `golang.org/x/sys` to a direct dependency — run `go mod tidy`
  and confirm no other version churn.

---

## Secondary Defects

**None that cause incorrect behavior**, but two adjacent observations worth recording:

1. **No SIGTERM/SIGINT handling in the stdio mcp path either.** `runMCP`
   (`internal/cli/mcp.go:35-62`) calls `Run()` with no `signal.NotifyContext`. If a
   user (or a supervisor) sends SIGTERM to an orphan, default Go behavior terminates
   it — which is fine — but there's no graceful flush of the debug log
   (`s.debugLog`). Low impact; not worth a separate fix, but the watchdog's
   `os.Exit(0)` path likewise skips the `defer f.Close()` on the debug log. Acceptable
   for a backstop that only runs when the parent is already gone.

2. **The 1MB scanner buffer cap** (`mcp_lifecycle.go:35`) means a single JSON-RPC
   line over 1MB makes `Scan()` return an error and `Run()` exit. Unrelated to this
   bug, but worth a separate look if large tool payloads ever appear — flagging, not
   fixing here.

---

## Notes

**Why a watchdog and not an idle timeout.** A time-based idle timeout ("exit if no
request in N minutes") cannot distinguish "abandoned" from "quiet but alive." Claude
Code does not auto-respawn a dead stdio server (that's precisely the `/mcp`-reconnect
symptom), so an idle timeout would *break live sessions* — the user steps away, the
server self-destructs, and tools silently stop working until a manual reconnect. The
ppid watchdog has no such failure mode: it fires only after the parent is provably
dead. EOF remains the fast, correct path; the watchdog is a pure backstop.

**Why key on ppid *change* not `== 1`.** Hardcoding `os.Getppid() == 1` assumes the
orphan always reparents to pid 1. On Linux, a subreaper (set via
`PR_SET_CHILD_SUBREAPER`) can become the new parent instead, so the ppid would be the
subreaper's pid, not 1. Comparing against the *startup* ppid catches both cases.

**Dependency posture.** `golang.org/x/sys` is already vendored/cached
(`v0.43.0`), so the Linux fast path adds no new external dependency — only promotes an
existing indirect one to direct.

---

## Recap
`hero mcp` stdio servers orphan because `Run()`'s only exit is stdin EOF, which the
client controls and doesn't always deliver on teardown — leaving hero blocked in
`scanner.Scan()` and reparented to launchd. The fix is a parent-liveness watchdog in
`Run()` (poll `os.Getppid()` for a change, optional Linux `prctl` fast path), gated to
real stdio mode and torn down on `Run()` return. Severity is low (resource leak, no
correctness impact); the "subprocess inherits stdin" theory was checked and is false.

---

## Kickoff

Fixed `hero mcp` orphan processes piling up — they never exited when Claude Code died
without closing the stdin pipe, so 14 stale servers accumulated over 2 days.

**Status:** completed — parent-liveness watchdog landed on branch
`fix/mcp-orphan-parent-liveness`. Audit verdict SHIP. All gates green.

**What landed:** `Run()` now starts a ppid-watching backstop (gated to real stdio via
`s.input == os.Stdin`, torn down on `Run()` return) that `os.Exit(0)`s when the parent
reparents away. Linux gets a `PR_SET_PDEATHSIG` fast path; darwin uses the 30s poll.
Seam vars (`watchdogExit`/`watchdogGetppid`) make the reparent-decision branch
unit-tested in-process; the true OS-reparent path is deferred to e2e (see below).

**Pick up at (only remaining work):** the real "parent SIGKILL → orphan exits within
~30s" path is not unit-testable in-process. Add an e2e harness to the
`e2e-validation` / `per-feature-smoke-coverage` family: build the binary, spawn
`hero mcp` via a pipe from an intermediate parent, SIGKILL that parent without closing
the pipe, poll that the child pid is gone past one watchdog tick.

→ `.hero/planning/bugs/hero-mcp-orphan-no-parent-liveness/spec.md`

**Files:** `internal/serve/mcp_lifecycle.go` (gate block), `internal/serve/mcp_watchdog.go`, `mcp_watchdog_linux.go`, `mcp_watchdog_other.go`, `mcp_watchdog_test.go`, `mcp_watchdog_linux_test.go`
**Skip:** "subprocesses leak stdin" (false — no mcp-path child sets `cmd.Stdin`); idle timeout (would kill live sessions — Claude Code doesn't auto-respawn).
