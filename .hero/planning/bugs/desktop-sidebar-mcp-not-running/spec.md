---
title: "HeroDesktop sidebar shows MCP notRunning error when hero serve is absent"
slug: desktop-sidebar-mcp-not-running
type: bug
status: handed_off
priority: medium
severity: medium
domain: engineering
created: 2026-06-04
origin: session
root_cause_class: design
size: medium
tags: [mcp, hero-serve, desktop, sidebar, lifecycle, cross-codebase]
relates-to:
  - hero-mcp-orphan-no-parent-liveness
---

# HeroDesktop sidebar shows MCP notRunning error when hero serve is absent

## Summary

### Categorization
| Attribute | Assessment |
|-----------|------------|
| **Criticality** | medium -- raw error messages in the UI degrade the user experience; no data loss or correctness impact, but the sidebar becomes non-functional when `hero serve` is not running |
| **Ease of Fix** | moderate -- Go-side hardening is straightforward (health endpoint exists, HTTP API already exposes `/api/{project}/specs`); the harder half is on the Swift desktop side (lifecycle management, graceful degradation) which is in a separate, unpeered repo |
| **Caused by our codebase?** | Partially -- design gap spans two codebases. The Go side lacks an auto-start mechanism and the desktop side lacks graceful handling of server absence |
| **Needs more research?** | Yes -- the Swift HeroDesktop codebase is not available via cross-repo peering; the desktop's MCP client implementation, error-mapping logic, retry/recovery behavior, and lifecycle management of `hero serve` cannot be traced from this repo |

### Background
The HeroDesktop Swift companion app uses the `hero_list` MCP tool to populate a sidebar section. When `hero serve` (the MCP server process) is not running, the sidebar displays:
```
[SidebarSectionBody] hero_list MCP call failed: notRunning
```
(logged at 2026-06-04 13:58:31.433, PID 2317). The error occurs at the transport/connection layer -- it never reaches Hero's Go code.

This is the **inverse** of the recently-completed `hero-mcp-orphan-no-parent-liveness` spec, which addressed orphaned MCP processes that accumulated when they should have stopped. This bug is about the server being **absent when needed**, and the desktop not gracefully handling it.

### Analysis
The bug is a **design gap** at the boundary between two applications:

1. **`hero serve` has no auto-start mechanism.** It is a manually started daemon (`hero serve` in a terminal). There is no LaunchAgent plist, systemd unit, or any "start-if-not-running" mechanism in the Go codebase. If the user hasn't started it, or if it crashed/was-stopped, the MCP server is simply absent.

2. **The desktop has no graceful degradation.** When the MCP call fails, the raw error string `notRunning` surfaces directly in the sidebar UI rather than showing a helpful empty state ("Hero daemon not running -- start it with `hero serve`") or attempting to start the daemon itself.

3. **The MCP transport distinction matters.** `hero mcp` (stdio MCP server for AI tools like Claude Code) is a separate process from `hero serve` (HTTP daemon with MCP, dashboard, watcher). The desktop appears to be calling `hero_list` as an MCP tool -- but since it's a native macOS app (not an AI coding tool), it almost certainly connects to the HTTP daemon at `127.0.0.1:7437`, not the stdio pipe. The `notRunning` error is likely the desktop's own transport-layer check detecting that no daemon is listening.

### Root Cause
**Design gap at the system boundary.** Neither the Go daemon nor the Swift desktop was designed with the other's lifecycle in mind:
- The Go side assumes `hero serve` is started manually by the user and stays running.
- The desktop side assumes the MCP server is already running and surfaces a raw error when it is not.
- No protocol exists between the two for lifecycle coordination (health probes, auto-start, graceful empty states).

### Source
**Go side (this repo):**
- `internal/cli/serve.go` -- `runServe()` starts the daemon; no auto-start mechanism
- `internal/serve/server.go` -- `Server.Run()` listens on `127.0.0.1:7437`
- `internal/serve/pidfile.go` -- PID file at `~/.hero/serve.pid` for status/stop
- `internal/serve/lifecycle.go` -- `probeHeroDaemon()`, `IsProcessAlive()`, `PortListenerHeld()` utilities
- `internal/serve/api.go` -- `/health` endpoint returns `{"status":"ok"}`

**Swift side (not accessible):**
- Unknown MCP client implementation
- Unknown `notRunning` error mapping
- Unknown lifecycle management of `hero serve`

### Fix Direction
Two-pronged fix spanning both codebases:

**Go side:** (1) Add a `hero serve ensure` subcommand that starts the daemon if not running (idempotent, suitable for the desktop to call); (2) optionally generate a LaunchAgent plist via `hero serve install-launchagent` for auto-start on login.

**Swift side:** (1) Graceful degradation -- show a helpful empty state when the daemon is unreachable instead of a raw error; (2) probe `/health` before making MCP calls; (3) optionally call `hero serve ensure` to auto-start the daemon.

---

## Problem Statement

HeroDesktop's sidebar calls the `hero_list` MCP tool to populate a section with spec data. When `hero serve` is not running on `127.0.0.1:7437`, the call fails at the transport layer and the sidebar shows:

```
[SidebarSectionBody] hero_list MCP call failed: notRunning
```

This is the user-visible symptom. The underlying problem is a missing lifecycle contract between the desktop app and the daemon:

1. `hero serve` is a manually-started foreground process. There is no mechanism to auto-start it on login, on-demand when the desktop needs it, or to restart it after a crash.
2. The desktop has no graceful fallback when the daemon is absent -- the raw error propagates to the UI.
3. The Go codebase provides all the building blocks for lifecycle management (`PIDFile`, `probeHeroDaemon`, `IsProcessAlive`, `PortListenerHeld`) but they are only used by the CLI `serve stop` and `serve status` commands, not exposed for external consumers like the desktop.

**Reproduction:**
1. Ensure `hero serve` is NOT running (`hero serve stop` or simply never start it)
2. Open HeroDesktop
3. Observe `[SidebarSectionBody] hero_list MCP call failed: notRunning` in the sidebar

## Environment Details

- macOS (Darwin 25.5.0) -- HeroDesktop is a native macOS app
- `hero serve` daemon port: 7437 (default)
- PID file location: `~/.hero/serve.pid`
- Health endpoint: `GET http://127.0.0.1:7437/health`
- No LaunchAgent, systemd unit, or other auto-start mechanism exists in the Go codebase

---

## Root Cause Analysis

All Go-side claims below are **read** (verified against source in this session). Swift-side claims are **assumed** (the desktop repo is not peered and not accessible).

**Claim 1 -- `hero serve` is purely manual-start.** `read`
`internal/cli/serve.go:78-166`: `runServe()` is invoked by `hero serve` CLI command. It creates a `serve.Server`, sets up signal handling (`SIGINT`, `SIGTERM`), and calls `srv.Run(ctx)` which blocks until cancelled. There is no daemon fork, no background mode, no `launchd` integration, no socket activation. The user must run `hero serve` in a terminal (or a wrapper script) to start it.

**Claim 2 -- No auto-start mechanism exists anywhere in the Go codebase.** `read`
Searched the entire repo for `plist`, `launchd`, `LaunchAgent`, `LaunchDaemon`, `systemd`, `autostart`, `auto.start`, `start-if-not`, `ensure.*running`, `daemon.*launch`. Zero hits. There is no mechanism for automatic daemon startup.

**Claim 3 -- The PID file and lifecycle utilities exist but are CLI-only.** `read`
`internal/serve/pidfile.go` provides `WritePIDFile`, `ReadPIDFile`, `RemovePIDFile`. `internal/serve/lifecycle.go` provides `probeHeroDaemon`, `IsProcessAlive`, `PortListenerHeld`. These are consumed only by `internal/cli/serve_lifecycle.go` (`hero serve stop`, `hero serve status`) and `internal/serve/server.go` (`diagnoseBindError`). No external consumer (like a desktop app) has a pathway to trigger daemon start.

**Claim 4 -- The HTTP API already exposes spec data.** `read`
`internal/serve/api.go:168`: `GET /api/{project}/specs` returns the spec list -- the same data `hero_list` returns via MCP. The desktop could use this HTTP endpoint directly instead of (or as a fallback to) the MCP tool call, with the advantage of standard HTTP error handling (connection refused = server not running).

**Claim 5 -- The `/health` endpoint is lightweight and suitable for liveness probing.** `read`
`internal/serve/api.go:219-229`: `GET /health` returns `{"status":"ok","version":"...","projects":N}`. Sub-second response, no disk I/O, no computation. Ideal for a pre-flight check before making heavier MCP/API calls.

**Claim 6 -- `hero_list` in the MCP server reads from disk, not from daemon state.** `read`
`internal/serve/mcp_tools.go:344-363`: `toolList` calls `spec.Discover(s.heroDir)` which scans the `.hero/` directory on disk. It does not depend on the HTTP daemon being up, watchers running, or any in-memory state. However, the MCP tools are only accessible via the MCP protocol (stdio for AI tools, or whatever transport the desktop uses), not as standalone functions.

**Claim 7 (assumed) -- The desktop connects to `hero serve` over HTTP or a local transport, not stdio.** `assumed`
The desktop is a native macOS app, not an AI coding tool. It would not spawn `hero mcp` as a child process over stdio. The most likely transport is HTTP to `127.0.0.1:7437`. The `notRunning` error is likely generated by the desktop's own MCP client when the HTTP connection fails (connection refused). This needs verification on the Swift side.

**Claim 8 -- The orphan spec's watchdog does not help here.** `read`
The `hero-mcp-orphan-no-parent-liveness` spec added a parent-liveness watchdog to `internal/serve/mcp_lifecycle.go:40-44` that exits the stdio MCP server when its parent dies. This addresses orphaned processes. The current bug is the opposite: the server (whether stdio or HTTP daemon) is not running at all. The watchdog is irrelevant to this case -- it prevents accumulation of dead processes, not absence of live ones.

**Conclusion:** The root cause is a **design-level lifecycle gap** at the boundary between two applications. The Go daemon has no auto-start capability and the desktop has no graceful degradation for daemon absence. Classification: `design`. Neither codebase is "wrong" in isolation -- the gap is in the contract between them.

---

## Code Flow (End to End)

### Normal (happy) path -- daemon running
1. User starts `hero serve` in a terminal
2. `internal/cli/serve.go:78` -- `runServe()` loads config, creates `serve.Server`
3. `internal/serve/server.go:302` -- `Server.Run()` opens TCP listener on `127.0.0.1:7437`
4. `internal/serve/server.go:497` -- writes PID file to `~/.hero/serve.pid`
5. `internal/serve/api.go:75` -- registers `/health` handler
6. `internal/serve/api.go:99` -- registers `/api/` project router including `/api/{project}/specs`
7. HeroDesktop connects (transport unknown -- HTTP or MCP-over-HTTP)
8. Desktop calls `hero_list` MCP tool
9. `internal/serve/mcp_dispatch.go:32` -- dispatches to `s.toolList`
10. `internal/serve/mcp_tools.go:344` -- `toolList` reads specs from disk via `spec.Discover`
11. Result returned to desktop; sidebar renders the spec list

### Failure path -- daemon NOT running (the bug)
1. User has NOT started `hero serve` (or it was stopped/crashed)
2. No process listening on `127.0.0.1:7437`
3. No PID file at `~/.hero/serve.pid` (or stale PID file from a crashed daemon)
4. HeroDesktop attempts to connect to `127.0.0.1:7437`
5. Connection refused at the TCP layer
6. Desktop's MCP client maps this to `notRunning` error (Swift-side logic, not traceable from this repo)
7. Error propagates to `[SidebarSectionBody]` renderer
8. Raw error string displayed in the sidebar: `hero_list MCP call failed: notRunning`

---

## Key Files

### Go: Daemon lifecycle (where the auto-start gap is)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/serve.go` | 78-166 | `runServe()` -- daemon start; manual-only, no auto-start |
| `internal/cli/serve_lifecycle.go` | 54-127 | `stopDaemon()` -- stop logic; shows the lifecycle primitives available |
| `internal/cli/serve_lifecycle.go` | 129-188 | `runServeStatus()` -- status probe; template for an "ensure" command |
| `internal/serve/server.go` | 302-525 | `Server.Run()` -- HTTP listener, PID file write, graceful shutdown |
| `internal/serve/pidfile.go` | 1-101 | PID file read/write/remove; the daemon's on-disk presence signal |
| `internal/serve/lifecycle.go` | 52-118 | `probeHeroDaemon`, `IsProcessAlive`, `PortListenerHeld` -- all the building blocks for lifecycle management |

### Go: MCP tool dispatch (what the desktop calls)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/serve/mcp_dispatch.go` | 24-74 | `toolHandlers()` -- `hero_list` dispatches to `s.toolList` |
| `internal/serve/mcp_tools.go` | 344-363 | `toolList()` -- reads specs from disk; no daemon-state dependency |

### Go: HTTP API (alternative data path for the desktop)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/serve/api.go` | 71-101 | `Handler()` routes: `/health`, `/api/status`, `/api/{project}/specs` |
| `internal/serve/api.go` | 219-229 | `handleHealth()` -- lightweight liveness probe |

### Go: Related spec (inverse problem)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/serve/mcp_watchdog.go` | 1-54 | Parent-liveness watchdog -- addresses orphan accumulation (inverse of this bug) |
| `internal/serve/mcp_lifecycle.go` | 40-44 | Watchdog gate in `Run()` -- confirms the orphan fix is separate from this issue |

---

## Cross-Codebase Investigation Needed

The Swift HeroDesktop codebase is **not peered** with this repo (only `hero-cloud` and `hero-code` peers exist). The following questions need investigation on the desktop side:

### Critical questions (must answer to complete the fix)
1. **How does the desktop connect to `hero serve`?** HTTP to `127.0.0.1:7437`? A custom MCP-over-HTTP transport? Direct stdio spawn of `hero mcp`?
2. **What does `notRunning` map to?** Is it a connection-refused error? A process-check? A custom error code? Where is it defined in the Swift code?
3. **Does the desktop attempt to start `hero serve` itself?** If yes, how? If no, is it expected to?
4. **What retry/recovery logic does the desktop have?** Does it retry on failure? Backoff? Show a persistent error or poll for recovery?

### Design questions (inform the fix approach)
5. **Should the desktop own daemon lifecycle?** Should HeroDesktop start `hero serve` on launch and stop it on quit? Or should it rely on an external mechanism (LaunchAgent, user manually starting)?
6. **What is the desired UX when the daemon is absent?** Empty sidebar with a "Start Hero" button? A system notification? Auto-start with a spinner?
7. **Does the desktop use other `hero serve` HTTP endpoints?** If it only uses MCP tools, should it switch to the HTTP API for simpler error handling?

### Recommended approach to answer these
- Register the desktop repo as a peer: `hero admin repos add hero-desktop /path/to/hero-desktop`
- Then: `hero peer call hero-desktop --mode=advisory "How does HeroDesktop connect to hero serve? What transport does SidebarSectionBody use for MCP calls? What does the notRunning error map to? Does the desktop manage hero serve lifecycle (start/stop)?"`

---

## Secondary Defects

1. **No auto-start mechanism exists at all.** Beyond the desktop, any consumer of `hero serve` (the dashboard UI in a browser, the MCP tools over HTTP, future mobile/remote surfaces) faces the same problem: if the daemon isn't running, they get nothing. A LaunchAgent or `hero serve ensure` would benefit all consumers, not just the desktop.

2. **Stale PID file after crash.** If `hero serve` crashes (SIGKILL, OOM, panic), the PID file at `~/.hero/serve.pid` is not removed (the `defer RemovePIDFile` in the shutdown path never fires). A subsequent `hero serve status` detects this via `IsProcessAlive` and reports it, but the desktop (if it checks the PID file) might see a stale file and think the daemon is running. The Go code handles this in `serve_lifecycle.go:79-85` but the desktop may not.

---

## Notes

**Relationship to the orphan spec.** This bug and `hero-mcp-orphan-no-parent-liveness` are two sides of the same lifecycle coin:
- **Orphan spec:** server is present when it should be absent (processes accumulate after client death). Fixed by the parent-liveness watchdog.
- **This bug:** server is absent when it should be present (desktop gets `notRunning`). Needs lifecycle management and graceful degradation.

Together, they suggest the system needs a coherent lifecycle story for `hero serve`: auto-start on demand, graceful shutdown on idle/exit, and watchdog for crash recovery. The orphan spec solved the "stop" side; this spec addresses the "start" side.

**Why `hero mcp` (stdio) is not relevant here.** The stdio MCP server is designed for AI coding tools (Claude Code, Cursor, OpenCode) that spawn it as a child process. The desktop is a long-lived GUI app that connects to the HTTP daemon, not a tool that spawns stdio children. The stdio path, the orphan watchdog, and the parent-liveness mechanism are all irrelevant to this bug.

---

## Suggested Fix Approach

### Go side (this repo)

#### Change 1 -- Add `hero serve ensure` subcommand

**File:** `internal/cli/serve.go` (new subcommand registration)
**File (new):** `internal/cli/serve_ensure.go`

Add a new `hero serve ensure` command that:
1. Probes `127.0.0.1:7437` via `probeHeroDaemon(port)`
2. If the daemon is already running, prints status and exits 0
3. If not running, starts `hero serve` in the background (fork + detach), waits for the health endpoint to respond, then exits 0

This gives the desktop (and any other consumer) a single idempotent command to guarantee the daemon is running.

**Before:** No `ensure` subcommand exists.

**After (sketch):**
```go
var serveEnsureCmd = &cobra.Command{
    Use:   "ensure",
    Short: "Start the Hero daemon if not already running",
    RunE:  runServeEnsure,
}

func runServeEnsure(cmd *cobra.Command, args []string) error {
    port := servePort
    if port == 0 {
        port = serve.DefaultPort
    }
    // Already running?
    if info := serve.ProbeHeroDaemon(port); info != nil {
        fmt.Printf("hero daemon already running (pid %d, port %d)\n", info.PID, info.Port)
        return nil
    }
    // Start in background
    heroPath, _ := exec.LookPath("hero")
    child := exec.Command(heroPath, "serve")
    child.SysProcAttr = &syscall.SysProcAttr{Setsid: true} // detach
    if err := child.Start(); err != nil {
        return fmt.Errorf("starting hero serve: %w", err)
    }
    // Wait for health
    deadline := time.Now().Add(10 * time.Second)
    for time.Now().Before(deadline) {
        if info := serve.ProbeHeroDaemon(port); info != nil {
            fmt.Printf("hero daemon started (pid %d, port %d)\n", info.PID, info.Port)
            return nil
        }
        time.Sleep(200 * time.Millisecond)
    }
    return fmt.Errorf("hero serve started but not responding after 10s")
}
```

**Why:** Provides an idempotent, scriptable entry point for any consumer to ensure the daemon is running. The desktop can shell out to `hero serve ensure` on launch.

#### Change 2 -- Export `probeHeroDaemon` for external use

**File:** `internal/serve/lifecycle.go`

**Before (line 55):**
```go
func probeHeroDaemon(port int) *DaemonStatusResponse {
```

**After:**
```go
func ProbeHeroDaemon(port int) *DaemonStatusResponse {
```

**Why:** The function is currently unexported. `hero serve ensure` (in package `cli`) needs to call it. The existing caller in `server.go:624` (`diagnoseBindError`) updates to `ProbeHeroDaemon`.

#### Change 3 (optional) -- Add LaunchAgent plist generator

**File (new):** `internal/cli/serve_launchagent.go`

Add `hero serve install-launchagent` that writes a plist to `~/Library/LaunchAgents/com.hero-engine.serve.plist` configured to start `hero serve` on login and restart on crash. This is the macOS-native way to ensure a daemon stays running.

**Why:** Eliminates the "forgot to start hero serve" class of errors entirely. Optional because it requires user consent (LaunchAgent installation) and may not be appropriate for all users.

### Swift side (desktop repo -- recommendations only)

#### Desktop Change 1 -- Graceful degradation
When the MCP call to `hero_list` fails with `notRunning` (or any connection error):
- Show a helpful empty state in the sidebar: "Hero daemon not running"
- Include a "Start" button that runs `hero serve ensure`
- Do NOT show the raw error string `hero_list MCP call failed: notRunning`

#### Desktop Change 2 -- Health probe before MCP calls
Before making any MCP tool call, probe `GET http://127.0.0.1:7437/health`:
- If healthy, proceed with the MCP call
- If unreachable, show the empty state immediately (avoid the MCP call timeout)

#### Desktop Change 3 -- Periodic health polling
Poll `/health` periodically (every 30-60s). When the daemon comes back online, automatically refresh the sidebar. When it goes offline, switch to the empty state.

#### Desktop Change 4 -- Auto-start on app launch
On HeroDesktop launch, run `hero serve ensure` to guarantee the daemon is running before making any MCP calls. Show a brief spinner during startup.

---

## Acceptance Criteria

- WHEN `hero serve` is not running THE SYSTEM SHALL provide `hero serve ensure` that starts the daemon and exits 0 when healthy
- WHEN `hero serve ensure` is called and the daemon is already running THE SYSTEM SHALL exit 0 without starting a second instance
- IF the daemon fails to start within 10 seconds THEN THE SYSTEM SHALL exit non-zero with a diagnostic message
- THE SYSTEM SHALL export `ProbeHeroDaemon` so package `cli` can use it for the ensure command
- WHEN `hero serve ensure` starts the daemon THE SYSTEM SHALL detach the child process so it outlives the ensure command

### Desktop-side ACs (for the Swift repo)
- WHEN `hero_list` MCP call fails with a connection error THE SYSTEM SHALL display a helpful empty state instead of the raw error string
- WHEN the daemon becomes reachable after being absent THE SYSTEM SHALL automatically refresh the sidebar content

---

## Boundaries

- This spec does NOT fix the desktop's error handling (that is in the Swift repo)
- This spec does NOT add the LaunchAgent plist as a required change (optional, separate work)
- This spec does NOT change the MCP stdio server (`hero mcp`) -- that is a separate process for AI coding tools
- This spec does NOT address the stale PID file issue (secondary defect, separate work)

---

## Risks

- **Cross-codebase coordination.** The full fix requires changes in both repos. The Go-side changes (`hero serve ensure`, exported `ProbeHeroDaemon`) are independently valuable but the user-visible fix requires the desktop to use them.
- **Process detachment on macOS.** `child.SysProcAttr = &syscall.SysProcAttr{Setsid: true}` creates a new session, but macOS may still associate the child with the terminal. Testing on macOS is required.
- **Port conflicts.** `hero serve ensure` assumes the default port (7437). If the user runs on a custom port, the ensure command needs `--port` support or config awareness.

---

## Validation

### Go side
1. Run `hero serve ensure` when no daemon is running -- confirm it starts one and exits 0
2. Run `hero serve ensure` when a daemon IS running -- confirm it exits 0 without starting a second instance
3. Run `hero serve ensure` and then `hero serve status` -- confirm status reports the daemon started by ensure
4. Kill the daemon with SIGKILL, run `hero serve ensure` -- confirm it cleans up the stale PID file and starts a fresh daemon
5. Verify `go test ./internal/serve/...` passes (no regression from exporting `probeHeroDaemon`)

### Desktop side (for the Swift repo)
1. Stop `hero serve`, open HeroDesktop -- confirm sidebar shows a helpful empty state, not a raw error
2. Start `hero serve` while HeroDesktop is open -- confirm sidebar auto-populates
3. With `hero serve ensure` available: open HeroDesktop when daemon is stopped -- confirm it auto-starts the daemon

---

## Test Plan

### Existing test review
| File | Coverage | Relevance |
|------|----------|-----------|
| `internal/serve/pidfile_test.go` | PID file read/write/remove, `IsProcessAlive` | Directly relevant -- ensure command depends on these |
| `internal/serve/lifecycle_test.go` | `probeHeroDaemon` with a test HTTP server | Directly relevant -- ensure command uses the probe |
| `internal/cli/serve_test.go` | `--force` flag, port binding | Adjacent -- tests the serve startup path |

### Test changes needed
1. **`TestServeEnsure_AlreadyRunning` (new, `internal/cli/serve_ensure_test.go`):** Start a test HTTP server on a known port that responds to `/api/status`. Run `runServeEnsure` targeting that port. Assert it exits 0 without spawning a child process.

2. **`TestServeEnsure_StartsBackground` (integration, may be hard to unit-test):** Assert that `hero serve ensure` spawns a background `hero serve` process. Would need a test binary or mock.

3. **`TestProbeHeroDaemon_Exported` (rename in `internal/serve/lifecycle_test.go`):** Update existing test to use the exported `ProbeHeroDaemon` name.

### Regression scope
- Renaming `probeHeroDaemon` to `ProbeHeroDaemon` affects one call site in `internal/serve/server.go:624`. Update that reference.
- New `serve ensure` subcommand -- verify it doesn't break existing `serve`, `serve stop`, `serve status` command routing.
- Run the full `go test ./...` suite.

---

## Recap

HeroDesktop's sidebar shows a raw `notRunning` error when `hero serve` is not running because there is no lifecycle management between the two apps -- no auto-start, no graceful degradation, no health probing. The Go side lacks an idempotent "ensure daemon is running" mechanism, and the desktop side surfaces raw transport errors instead of helpful empty states. The fix spans both codebases: add `hero serve ensure` on the Go side, and graceful degradation + auto-start on the Swift side. Severity is medium -- the sidebar is non-functional but there's no data loss.

---

## Kickoff

Investigate and fix the HeroDesktop sidebar `notRunning` error when `hero serve` is absent.

**Go-side scope:** Add `hero serve ensure` subcommand (idempotent start-if-not-running), export `ProbeHeroDaemon`. Optionally add LaunchAgent plist generator.

**What you need to know:**
- `hero serve` is the HTTP daemon on port 7437 -- manually started, no auto-start mechanism
- The desktop calls `hero_list` via MCP; when the daemon is absent, the call fails at the transport layer
- All lifecycle primitives exist (`probeHeroDaemon`, `IsProcessAlive`, `PortListenerHeld`) -- they just need to be wired into an `ensure` command
- The orphan spec (`hero-mcp-orphan-no-parent-liveness`) solved the inverse problem (stop side); this solves the start side

**Start with:**
1. Export `probeHeroDaemon` -> `ProbeHeroDaemon` in `internal/serve/lifecycle.go`
2. Update the one caller in `internal/serve/server.go:624`
3. Add `internal/cli/serve_ensure.go` with the ensure subcommand
4. Register it in `internal/cli/serve.go` init()
5. Test manually: `hero serve ensure` when stopped, when running

**Skip:** Desktop-side changes (separate repo), LaunchAgent plist (optional/separate), stdio MCP changes (irrelevant).

-> `.hero/planning/bugs/desktop-sidebar-mcp-not-running/spec.md`

**Files:** `internal/cli/serve_ensure.go` (new), `internal/serve/lifecycle.go` (export rename), `internal/serve/server.go` (update caller)

## Handoff Trail

- 2026-06-24T18:01:15Z — out → hero-code (peer_id: cd8dd06d-3df1-4878-a88f-24593dcbb4b3)
  mode: async-drop
  originating_spec: desktop-sidebar-mcp-not-running
  peer_spec: hero-code/desktop-sidebar-mcp-not-running
  at_commit: 2f774b7
  reason: "The user-visible defect is in hero-code's Swift HeroDesktop sidebar (SidebarSectionBody notRunning string). The hero CLI only owns the optional 'hero serve ensure' enabler, tracked separately."

