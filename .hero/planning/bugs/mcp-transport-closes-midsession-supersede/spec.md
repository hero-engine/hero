---
title: "Hero MCP transport closes mid-session in Codex — singleton supersede / process-lifecycle guards kill the live daemon"
slug: mcp-transport-closes-midsession-supersede
type: bug
status: planning
domain: engineering
size: small
priority: high
severity: high
root_cause_class: design
created: 2026-07-13
tags: [mcp, stdio, transport, codex, singleton, watchdog, reliability, serve]
tracker_id: null
---

# Hero MCP transport closes mid-session in Codex

## Summary

### Categorization
| Attribute | Assessment |
|-----------|------------|
| **Criticality** | high — Hero MCP tools become unavailable mid-session in Codex (one of six install targets); the agent silently falls back to the CLI, losing two-tier refs, session context, and the MCP surface. Core value ("agent leans on Hero MCP") degrades without warning. |
| **Ease of Fix** | moderate — the mechanism is small and localized to `internal/serve`, but the correct fix requires tightening the singleton identity/policy without reintroducing the orphan-daemon leak the guard was built to solve. |
| **Caused by our codebase?** | Yes — the death is Hero's own process-lifecycle guards (singleton supersede + parent-liveness watchdog), introduced in commit `bcb9424`. Not Codex, not the search path. |
| **Needs more research?** | Yes (for the *trigger* only) — the death *mechanism* and the over-aggressive supersede *design* are proven. What is NOT yet captured is the exact event inside Codex's MCP host that spawns/reconnects a second `hero mcp` during comparison queries. Enable `HERO_MCP_DEBUG` to catch the next occurrence. A second failing session added an index stale-lock / schema-mismatch observation; that was **run down and reproduced** (see Trigger B) — it produces graceful tool errors, **not** a transport close, so it does not change the transport-close root cause. |

### Background
During a Codex chat session the agent narrated: *"Hero's in-process search transport
closed on the comparison queries, so I'm switching to the local Hero CLI for this
read-only inventory."* There is **no separate "search transport"** that can close on its
own — the Hero MCP server is a single stdio process (`hero mcp`) that Codex spawns once per
session. "Transport closed" is the client's view of **that process dying**. The question is
therefore not "why did the transport close" but "why did the `hero mcp` process die
mid-session," and specifically around repeated/comparison `hero_search` calls.

### Analysis
Two independent, proven facts narrow it:

1. **The search handler does not crash on repeated/comparison queries.** Driving the
   compiled server with a batch of seven back-to-back `hero_search` calls (plain + compact,
   varied filters) returns all seven responses and exits cleanly (`EXIT=0`, no stderr). Each
   `toolSearch` opens and `defer`-closes its own retrieval layer, so there is no
   "closed-after-first-use" handle shared across calls. The search path is exonerated as the
   crash source.

2. **Hero's process-lifecycle guards kill a live, serving daemon.** Commit `bcb9424`
   ("reap orphaned hero mcp processes with a parent-liveness watchdog") added two guards in
   `Run()`:
   - a **singleton** keyed on `(workspace, parent-pid)` that **SIGTERMs the incumbent** the
     instant a second `hero mcp` starts for the same key (`acquireMCPSingleton`, policy:
     *supersede*), and
   - a **parent-liveness watchdog** that calls `os.Exit(0)` if `os.Getppid()` changes from
     its startup value.

   A direct reproduction of the supersede path (two daemons sharing a parent shell + the same
   workspace) shows the **first daemon dies within 1 second** of the second starting — SIGTERM,
   no graceful shutdown, pidfile handed to the successor. From the first daemon's client
   (Codex), that is exactly "the transport closed."

### Root Cause
The **singleton supersede design** (primary) is too coarse and too aggressive. Its identity
key `(workspace, parent-pid)` and its unconditional, immediate SIGTERM cannot distinguish:
- a **reconnect** — the old stdio pipe is dead, the incumbent *should* be reaped (the case
  the guard was built for), from
- a **legitimate second server** that shares the same parent and workspace while the first
  connection is **still live and in use** — in which case killing the incumbent tears the
  transport out from under the agent mid-work.

Under Codex, any event that causes a *second* `hero mcp` to be spawned/reconnected for the
same workspace under the same parent process (a host-side reconnect, a health-check-triggered
restart while the server is busy servicing a burst of comparison queries, or a second
window/pane on the same repo under the same Codex parent) trips the supersede and kills the
daemon the agent is actively querying. The parent-pid key is a proxy for "same client," but
it does not actually verify that the incumbent's *connection* is dead before killing it.

Secondary contributing factors (see Secondary Defects): a parent-liveness watchdog that
`os.Exit`s on any ppid change (false-positive on legitimate reparenting), and no panic
recovery in the request loop (any handler panic crashes the whole server).

### Source
- `internal/serve/mcp_singleton.go` — `acquireMCPSingleton` / supersede policy (primary).
- `internal/serve/mcp_lifecycle.go` — `Run()` wires the singleton + watchdog; scanner loop
  has no `recover()`.
- `internal/serve/mcp_watchdog.go` — `startParentWatchdog` exits on ppid change.
- `internal/cli/mcp.go` — `runMCP` → `mcpSrv.Run()` (entrypoint; no signal handling).

### Fix Direction
Make the incumbent-reaping decision *verify the incumbent is actually dead/disconnected*
before killing it, rather than assuming a same-parent second spawn means "reconnect." At
minimum: gate supersede so a demonstrably-live-and-serving incumbent is not killed, or
narrow the identity so genuinely-concurrent servers coexist. Add graceful SIGTERM handling
(clean pidfile release, flush) and wrap request dispatch in a `recover()` so a single tool
panic can't take down the whole session. Keep the orphan-reaping benefit intact — do not
simply revert `bcb9424`.

---

## Problem Statement

Reported symptom (verbatim agent narration in a Codex session):

> "Hero's in-process search transport closed on the comparison queries, so I'm switching to
> the local Hero CLI for this read-only inventory."

Observed:
- The drop happens while running **multiple / comparison** `hero_search` queries in sequence.
- On the drop, Codex falls back to the local `hero` CLI for the same read-only work.
- It happened in **Codex**. Codex launches the server via `.codex/config.toml`:
  ```toml
  [mcp_servers.hero]
  command = "/Users/bwheeler/go/bin/hero"
  args = ["mcp"]
  ```
  i.e. a single persistent stdio process per session, parented to Codex's MCP host.

The installed binary Codex runs (`/Users/bwheeler/go/bin/hero`, `v0.25.0-dirty`, built the
same day) **contains** the singleton and watchdog code (`strings` confirms `mcp-%d.pid`,
`hero mcp: singleton lock`, `startParentWatchdog`). So the running server is affected.

### Reproduction

**Search path is NOT the cause (clean):**
```
# 7 back-to-back hero_search calls (plain + compact + filters) piped to `hero mcp`
EXIT=0
ids seen: 1 2 3 4 5 6 7   (all responded)
stderr: (empty)
```

**Singleton supersede DOES kill a live daemon (reproduced):**
```
# two `hero mcp` daemons, same parent shell, same workspace, stdin held open
first hero mcp pid=35528
started second daemon; watching first for death...
  t=+1s first(35528)=DEAD | live hero mcp pids: 35654
  ... stays DEAD ...
final pidfile: {"pid":35654,"ppid":35525,...}
```
The incumbent is SIGTERM'd and dead within 1s; the newcomer owns the pidfile. This is the
client-visible "transport closed."

**Leaked pidfiles observed:** `.hero/mcp-<ppid>.pid` files from prior sessions remained on
disk with the recorded pid **dead** — daemons died without running the deferred `release()`
cleanup (SIGTERM and `os.Exit(0)` both skip defers). Benign (treated as stale next acquire)
but corroborates that daemons die via these non-graceful paths.

### What is confirmed vs. inferred
- **Confirmed:** the search handler does not crash on repeated queries; the singleton
  supersede kills a live incumbent within 1s; the installed Codex binary contains the guard;
  no SIGTERM handler exists (default disposition = immediate death, verified by the repro);
  no `recover()` guards the request loop.
- **Inferred (needs capture):** the precise Codex-host event that spawns/reconnects the
  second `hero mcp` during comparison queries. The "comparison queries" correlation may be
  causal (a busy server slow to answer a host ping → host restarts it) or incidental (that's
  simply when the agent noticed). Not yet observed with a debug log.

## Environment Details
- Harness: **Codex** (stdio MCP client), macOS (darwin 25.5.0).
- Server: `hero mcp` v0.25.0, single stdio process per session, parent = Codex MCP host.
- macOS has **no** `PR_SET_PDEATHSIG`; the watchdog relies solely on the portable ppid poll.
- Relevant config: `.codex/config.toml` `[mcp_servers.hero] args=["mcp"]`.
- Guards gated on `s.input == os.Stdin` (real stdio mode) — they run in Codex, not in unit
  tests that drive `Run()` with a buffer.

---

## Root Cause Analysis

### Primary — singleton supersede over-kills (`root_cause_class: design`)

`internal/serve/mcp_lifecycle.go` `Run()` (real-stdio mode only):

```go
if s.input == os.Stdin {
    ppid := os.Getppid()
    if release, err := acquireMCPSingleton(mcpPIDFilePath(s.heroDir, ppid), os.Getpid(), ppid); err != nil {
        ...
    } else {
        defer release()
    }
    done := make(chan struct{})
    defer close(done)
    startParentWatchdog(done)
}
```

`internal/serve/mcp_singleton.go` `acquireMCPSingleton`:

```go
if rec := readMCPPIDRecord(path); rec != nil {
    if rec.PID != self && singletonIsAlive(rec.PID) {
        // Live incumbent for this client+workspace. Supersede it:
        // a reconnect should win. Best-effort ...
        _ = singletonSignal(rec.PID)   // SIGTERM, immediate
    }
}
```

The decision rests entirely on `singletonIsAlive(rec.PID)` — *is the incumbent process
alive?* It never checks *is the incumbent's client connection dead?* The design **assumes**
that a second daemon starting for the same `(workspace, ppid)` means the client dropped the
old connection and reconnected. That assumption fails whenever the incumbent's connection is
still live. Result: SIGTERM to a daemon that is mid-conversation with the agent.

`singletonSignal` sends `SIGTERM`; the server installs **no** handler (confirmed: no
`signal.Notify` anywhere in `internal/serve` or `cmd`/`internal/cli`), so the process dies
immediately with default disposition — no flush, no pidfile cleanup. Reproduced: incumbent
dead in <1s.

Why `(workspace, ppid)` is the wrong key for "same client": ppid identifies the *spawning
process*, not the *connection*. Two live connections can share a parent (concurrent servers,
multi-pane, host that pre-spawns), and a reconnect is not the only way a same-parent second
daemon appears. The key conflates "same parent" with "old connection is dead."

### Why "comparison queries" plausibly triggers it (hypothesis)
A burst of comparison `hero_search` calls keeps the single-threaded `Run()` loop busy
(each call runs `ensureFreshIndex()` → `index.RefreshIfStale`, then `retrieval.New` +
`Retrieve`, serially). If Codex's MCP host pings or health-checks the server on a separate
expectation and the busy server is slow to answer, the host may treat it as unhealthy and
**spawn a replacement** — which then supersedes (SIGTERMs) the incumbent the agent is still
using. This is consistent with all observed facts but is **not yet captured** in a debug log.
See "Needs more research."

### Trigger B — index stale-lock / graph schema-mismatch (a SECOND session; run down and reproduced)

A second failing session narrated:

> "The decision anchor call failed because Hero's local MCP transport closed (the same
> local index is also showing a stale lock/schema mismatch). I'm reading the authoritative
> mission and tripwire files directly and will note this tooling failure as part of the
> evidence."

Two candidate mechanisms were tested. **Both were reproduced, and both produce graceful
`isError` tool results — neither closes the transport.** The transport-close in this session
is still Trigger A (process death); the "stale lock / schema mismatch" is a co-occurring,
distinct condition the agent observed when it fell back to reading directly (note its own
wording: the index is "**also** showing" a lock — co-occurrence, not causation).

**B1 — Index concurrency: no busy-timeout, no WAL (`root_cause_class: code`/`env`).**
`internal/index/index.go` `Open` calls `sql.Open("sqlite", dbPath)` with **no** connection
params — no `_busy_timeout`, no `_journal_mode=WAL`, no `_txlock` (confirmed: zero
busy_timeout/WAL config in the package). SQLite therefore fails **immediately** with
`database is locked (SQLITE_BUSY)` on any contended access instead of briefly waiting. The
graph, by contrast, opens with `PRAGMA journal_mode = WAL` (`internal/graph/graph.go:73`) and
tolerates concurrent readers — the index does not.

Concurrency is real and routine under Codex: `.codex/hooks.json` fires `hero next ingest
--quiet` and `hero next checkpoint --quiet` around turns, and `hero next ingest` **writes**
(`internal/nextdoc/graph_ingest.go` `store.UpsertNode`/`UpsertEdge`). These are separate
`hero` processes opening the workspace DBs while the persistent `hero mcp` daemon queries
them. On the index (no busy_timeout) the loser gets `database is locked`.

**Reproduced (decisive):** with an `EXCLUSIVE` lock held on `.hero/index.db`, firing
`initialize` + `hero_anchor` + `hero_search` + `hero_status` at the compiled server:
```
SERVER_EXIT=0            # server did NOT crash
ids answered: 1 2 3 4    # transport stayed fully alive
id=2 hero_anchor  isError=True  "Error: opening index: migrating index schema: ... database is locked (5)"
id=3 hero_search  isError=True  "Error: opening retrieval layer: ... database is locked"
id=4 hero_status  isError=False (succeeded — reads specs from disk, not the locked index)
```
A locked index degrades individual tool calls to graceful errors; it does **not** tear down
the transport. This **disproves** the hypothesis that an index-lock panics the handler and
(absent `recover()`) crashes the server — there is no panic. Grep confirms **no**
`panic`/`log.Fatal`/`Must(` anywhere in the index or retrieval read paths, and no
off-main-goroutine panic in retrieval.

**B2 — Graph schema mismatch (`root_cause_class: env` — version skew).** "Schema mismatch"
maps to `checkSchemaMismatch` (`internal/graph/graph.go:339`): when the running binary's
compiled graph schema is **newer** than the on-disk graph, `graph.Open` **returns an error**
(when the graph is newer than the binary it only prints a stderr warning). This is the exact
stray-/duplicate-binary-on-PATH condition commit `7dba572` added the `initialize`
`Schema`/`GraphSchema` fields to surface. It affects only the two graph-using read tools —
`hero_why` (`mcp_tools.go:3134`) and `hero_blocked` (`mcp_tools.go:3172`) — **not**
`hero_anchor` or `hero_search`, which are index-only. Like B1 it is a **returned error →
`isError` tool result**, not a transport close. (Locally the schemas match, 4=4, so this
session's own workspace is clean; the mismatch is an environment/PATH condition, not a code
defect in the open path.)

### Reconciliation — are A and B the same bug?
**No — they are independent, and only A closes the transport.** Evidence:
- The transport closing == the `hero mcp` process dying. Proven in every repro that the sole
  way the stdio transport closes is process death (Trigger A: supersede SIGTERM / watchdog
  exit). Index lock (B1) and schema mismatch (B2) were both reproduced as graceful `isError`
  results with the server alive and exiting 0.
- The second session's narration conflates a Trigger-A transport close with separately-noticed
  DB conditions ("**also** showing a stale lock"). The anchor call "failed" because the whole
  transport was gone (A), not because the index was locked (which would have returned a
  visible `isError`, not a closed pipe).

**One real coupling:** Trigger A's *non-graceful death* aggravates B1. A `hero mcp` (or a
`hero next` hook) SIGTERM'd mid-write — no signal handler, defers skipped, the same root as
the leaked `.hero/mcp-<ppid>.pid` files already documented — can abandon an in-flight index
transaction and leave a hot `.hero/index.db-journal`; combined with the missing busy_timeout,
the next opener is more likely to see a lock. So the graceful-shutdown fix (below) is now
load-bearing for **three** symptoms: killed live sessions, leaked pidfiles, and leaked DB
locks. B1's primary root is still the missing busy_timeout/WAL under ordinary concurrency,
independent of A.

**Net effect on framing:** the transport-close primary root cause is unchanged (Trigger A,
`design`). Trigger B is a **distinct reliability defect** (index opened without
busy_timeout/WAL → `database is locked` tool failures under concurrent `hero` processes) that
explains the *degraded tooling* the second session saw but is a separate fix. The coordinator's
hypothesis that panic-recovery is *load-bearing for B* is **not supported** — B does not
panic; panic-recovery remains cheap defense-in-depth with no confirmed trigger found.

### Secondary — parent-liveness watchdog false-positive (`code`/`race`)
`startParentWatchdog` (`internal/serve/mcp_watchdog.go`) calls `watchdogExit(0)` (=
`os.Exit(0)`) when `os.Getppid() != startPpid`. On macOS there is no PDEATHSIG; the ppid poll
is the only mechanism. If Codex spawns the server through an intermediate process that later
exits (leaving the server reparented while the *session* is still alive), the watchdog reads
the ppid change as "parent died" and exits mid-session. This is a second, independent
mid-session death vector. Harder to tie to searches specifically, but present.

### Secondary — no panic recovery in the request loop (`code`)
`Run()`'s scanner loop calls `s.handleRequest(&req)` with no `recover()`, and neither
`handleToolsCall` nor the handlers recover. Any panic in any tool handler crashes the entire
`hero mcp` process → transport close for *all* tools, indistinguishable from the above. Not
the proven cause here (search batch didn't panic), but the same client-visible symptom and
worth hardening.

---

## Code Flow (End to End)

1. `.codex/config.toml` — Codex spawns `/Users/bwheeler/go/bin/hero mcp`, parent = Codex host.
2. `internal/cli/mcp.go:35` `runMCP` — resolves workspace, builds server, calls `mcpSrv.Run()`.
3. `internal/serve/mcp_lifecycle.go:46-59` `Run()` (real-stdio) — `acquireMCPSingleton(...)`
   writes `.hero/mcp-<ppid>.pid`; `startParentWatchdog(done)` begins polling ppid.
4. `internal/serve/mcp_lifecycle.go:65-84` — scanner loop reads JSON-RPC lines, dispatches
   `hero_search` calls serially via `handleRequest` → `handleToolsCall` → `toolSearch`.
5. `internal/serve/mcp_tools.go:213` `toolSearch` — `ensureFreshIndex()`, `retrieval.New`,
   `Retrieve`, format; opens/closes its own resources per call (no cross-call handle).
6. **Death event (primary):** a second `hero mcp` starts for the same `(workspace, ppid)` →
   `internal/serve/mcp_singleton.go:71-75` sends SIGTERM to the incumbent (pid in step 3).
7. Incumbent has no SIGTERM handler → process dies immediately (deferred `release()` and
   watchdog `done` never run; pidfile left behind, now owned by successor).
8. Codex observes the incumbent's stdio pipe close → "in-process search transport closed" →
   falls back to the `hero` CLI.

Alternate death events reaching the same step 8: watchdog `os.Exit(0)` on ppid change
(`mcp_watchdog.go:47-49`); unrecovered panic in any handler (`mcp_lifecycle.go:83`).

---

## Key Files

### MCP process lifecycle (primary)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/serve/mcp_singleton.go` | 62–91 | `acquireMCPSingleton` — supersede policy; SIGTERMs live incumbent on same `(workspace, ppid)`. **Primary root cause.** |
| `internal/serve/mcp_singleton.go` | 36–45 | `singletonSignal` sends SIGTERM; no graceful handshake. |
| `internal/serve/mcp_lifecycle.go` | 46–59 | `Run()` wires singleton + watchdog (real-stdio gate). |
| `internal/serve/mcp_lifecycle.go` | 65–87 | scanner loop; dispatch with **no `recover()`**. |
| `internal/cli/mcp.go` | 35–63 | `runMCP` entrypoint → `Run()`; no signal handling. |

### Secondary vectors
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/serve/mcp_watchdog.go` | 34–54 | `startParentWatchdog` — `os.Exit(0)` on ppid change (reparent false-positive). |
| `internal/serve/mcp_watchdog_other.go` | 1–8 | darwin no-op for PDEATHSIG — ppid poll is the only mechanism on macOS. |
| `internal/serve/mcp_tools.go` | 213–278 | `toolSearch` — exonerated (opens/closes own resources; batch repro clean). |
| `internal/serve/mcp_expand.go` | 11–48 | shared refs store (`ensureRefsStore`/`registerRef`) — degrades gracefully on error, does not tear down transport. |

### Trigger B — index concurrency / schema mismatch (degrades tools, does NOT close transport)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/index/index.go` | 72–97 | `Open` — `sql.Open("sqlite", dbPath)` with **no busy-timeout / no WAL**. Trigger B1: concurrent access → immediate `database is locked`. **Fix target (Item 1b).** |
| `internal/serve/mcp_tools.go` | 862–931 | `toolAnchor` — opens the **index** (not the graph); returns error on lock. Reproduced: graceful `isError`, transport stays up. |
| `internal/nextdoc/graph_ingest.go` | 65–98 | `UpsertNode`/`UpsertEdge` — the writer behind the Codex `hero next ingest` hook; the concurrent-writer that contends with the daemon. |
| `.codex/hooks.json` | 1–20 | fires `hero next ingest`/`checkpoint` around turns → separate `hero` procs opening the workspace DBs concurrently. |
| `internal/graph/graph.go` | 60–88, 339–364 | Graph uses WAL (good); `checkSchemaMismatch` returns an **error** on binary-newer-than-graph (Trigger B2). Affects `hero_why`/`hero_blocked` only; graceful `isError`. |

### Prior art / related specs
| Spec | Relevance |
|------|-----------|
| `hihcp-mcp-auto-reconnect` (bug, handed_off) | Confirms the failure class: "Hero MCP server process dies unexpectedly mid-session." Client-side (hero-code Swift) recovery; Codex has no such recovery and just falls back to CLI. |
| commit `bcb9424` | Introduced the singleton + watchdog this bug lives in. Fix must preserve its orphan-reaping benefit. |
| `mcp-server-lifecycle`, `hero-embedded-mcp` | Background on the MCP server lifecycle design. |

---

## Secondary Defects
1. **No graceful SIGTERM shutdown.** The server dies on SIGTERM with default disposition —
   no pidfile cleanup, no flush. Causes leaked `.hero/mcp-<ppid>.pid` files (observed) and
   makes supersede a hard kill instead of a clean handoff.
2. **Index opened without busy-timeout or WAL** (`internal/index/index.go` `Open`) — Trigger
   B1. `sql.Open("sqlite", dbPath)` has no `_busy_timeout`/`_journal_mode`, so concurrent
   access (the `hero mcp` daemon vs. the Codex `hero next ingest`/`checkpoint` hook processes,
   a superseding second daemon, or `hero index`) fails immediately with `database is locked`
   instead of waiting. Reproduced. Degrades any index tool to `isError` under contention; does
   **not** close the transport. The graph already uses WAL; the index should too.
3. **Parent-liveness watchdog false-positive on reparent** (`mcp_watchdog.go`) — `os.Exit(0)`
   on any ppid change; a legitimate intermediate-parent exit kills a live session on macOS.
4. **No panic recovery in the request loop** (`mcp_lifecycle.go` `Run()`) — a tool-handler
   panic would crash the whole server with the same "transport closed" symptom. **No panic
   trigger was found**: the 7-search batch and the locked-index repro both returned gracefully.
   Kept as cheap defense-in-depth, explicitly **not** the fix for the observed index-lock
   errors.
5. **Leaked pidfiles / stale DB journals** — a direct consequence of the non-graceful death
   paths (defect 1). Stale pidfiles are benign (treated as free next acquire); a stale index
   journal + no busy_timeout makes the next tool call more likely to hit `database is locked`.
6. **Graph schema-mismatch surfaces as a raw tool error** (Trigger B2) — on a
   binary-newer-than-graph PATH skew, `hero_why`/`hero_blocked` return the `checkSchemaMismatch`
   error verbatim (already points at `hero doctor`). Environment condition, not a code defect.

---

## Notes
- The report's phrase "in-process search transport" is a Codex/agent characterization; there
  is no distinct search transport in Hero. All `hero_*` tools share the one stdio process, so
  the same death would drop *every* Hero tool, not just search — consistent with "fell back
  to the CLI for the whole inventory."
- Do **not** fix this by reverting `bcb9424`. The orphan/duplicate-daemon leak it solves is
  real (long-lived Codex clients accumulate one daemon per reconnect). The fix must keep
  reaping genuinely-dead/duplicate daemons while never killing a live, connected incumbent.
- The version/schema commit on this branch (`7dba572`) only adds `Schema`/`GraphSchema` to
  the `initialize` result — benign, additive, not a mid-session death vector. Ruled out.

---

## Recap
Codex's "in-process search transport closed" is the `hero mcp` **process dying mid-session**,
not a search-layer failure (the search path is clean under repeated/comparison queries). The
proven mechanism is Hero's own singleton **supersede** guard (commit `bcb9424`): keyed on
`(workspace, parent-pid)` with an unconditional immediate SIGTERM, it kills a live, serving
incumbent the moment a second `hero mcp` spawns for the same key — reproduced dead-in-1s. The
design flaw is that it assumes "same-parent second spawn == reconnect" without verifying the
incumbent's connection is actually dead. A second session's "index stale lock / schema
mismatch" was run down and **reproduced as graceful `isError` tool errors, not a transport
close** — it is a *separate* reliability defect (the index opens SQLite with no
busy-timeout/WAL, so concurrent Codex `hero next` hook processes trigger `database is
locked`), fixed independently (Item 1b). Severity **high** (Hero MCP silently drops in Codex);
the transport-close *mechanism and over-kill design are confirmed*; the exact Codex event that
spawns the second daemon during comparison queries still needs a debug-log capture.

---

## Suggested Fix Approach

> Design intent: keep reaping genuinely-dead / duplicate daemons (the `bcb9424` benefit) while
> **never** killing a live, connected incumbent. Prefer "verify the incumbent is dead before
> superseding" over "assume reconnect."

### 1. Gracefully handle SIGTERM (clean release, and make supersede a clean handoff)
**File:** `internal/serve/mcp_lifecycle.go` — `Run()`, real-stdio block.

**Before** (no signal handling; SIGTERM = hard kill, defers skipped):
```go
if s.input == os.Stdin {
    ppid := os.Getppid()
    if release, err := acquireMCPSingleton(mcpPIDFilePath(s.heroDir, ppid), os.Getpid(), ppid); err != nil {
        fmt.Fprintf(os.Stderr, "hero mcp: singleton lock: %v\n", err)
    } else {
        defer release()
    }
    done := make(chan struct{})
    defer close(done)
    startParentWatchdog(done)
}
```

**After** (install a SIGTERM handler that releases the pidfile and exits cleanly):
```go
if s.input == os.Stdin {
    ppid := os.Getppid()
    var release func()
    if r, err := acquireMCPSingleton(mcpPIDFilePath(s.heroDir, ppid), os.Getpid(), ppid); err != nil {
        fmt.Fprintf(os.Stderr, "hero mcp: singleton lock: %v\n", err)
    } else {
        release = r
        defer release()
    }

    // A superseding daemon (or the OS) sends SIGTERM. Run the pidfile
    // release before exiting so we don't leak .hero/mcp-<ppid>.pid, and
    // exit non-zero so the client can distinguish supersede from EOF.
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGTERM)
    go func() {
        <-sigCh
        if release != nil {
            release()
        }
        os.Exit(0)
    }()

    done := make(chan struct{})
    defer close(done)
    startParentWatchdog(done)
}
```
**Why:** Removes the leaked-pidfile secondary defect and gives the supersede path a clean
exit. This is necessary hygiene but is **not sufficient** — it still kills a live incumbent.
Item 2 is the load-bearing fix for the transport close. (Add `os/signal` and `syscall`
imports.) This item is also load-bearing for Trigger B1: a clean shutdown winds down in-flight
DB transactions instead of abandoning a hot `.hero/index.db-journal` lock.

### 1b. Give the index a busy-timeout and WAL (fixes Trigger B1 — the second session's `database is locked`)
**File:** `internal/index/index.go` — `Open`.

**Before:**
```go
db, err := sql.Open("sqlite", dbPath)
if err != nil {
    return nil, fmt.Errorf("opening index database: %w", err)
}
// Enable foreign keys
if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil { ... }
```

**After (match the graph's concurrency posture — WAL + a real busy-timeout):**
```go
// _busy_timeout makes contended opens/queries WAIT (up to 5s) instead of
// failing instantly with SQLITE_BUSY; WAL lets readers proceed while a
// writer (e.g. a `hero next ingest` hook, or `hero index`) holds the DB.
// The graph already does this (internal/graph/graph.go); the index did not.
db, err := sql.Open("sqlite", dbPath+"?_busy_timeout=5000&_journal_mode=WAL")
if err != nil {
    return nil, fmt.Errorf("opening index database: %w", err)
}
for _, pragma := range []string{
    "PRAGMA foreign_keys = ON",
    "PRAGMA journal_mode = WAL",   // belt-and-suspenders; DSN sets it too
    "PRAGMA busy_timeout = 5000",
} {
    if _, err := db.Exec(pragma); err != nil {
        db.Close()
        return nil, fmt.Errorf("setting pragma %q: %w", pragma, err)
    }
}
```
**Why:** This is the direct fix for the *degraded tooling* the second session saw — the index
was opened without any busy-timeout, so the moment a Codex `hero next ingest`/`checkpoint`
hook (or a second daemon) touched the DB, `hero_anchor`/`hero_search` returned `database is
locked`. Reproduced. WAL + a 5s busy-timeout make concurrent `hero` processes coexist. Note
this does **not** address the transport close (that's Trigger A) — it fixes the separate
reliability defect. Verify WAL sidecar files (`index.db-wal`, `index.db-shm`) are acceptable
in the workspace (the graph already creates them). Confirm the `modernc.org/sqlite` DSN param
spelling against the driver in `go.mod`.

### 2. Do not supersede a live, *connected* incumbent — verify liveness ≠ "should die"
**File:** `internal/serve/mcp_singleton.go` — `acquireMCPSingleton`.

**Before:**
```go
if rec := readMCPPIDRecord(path); rec != nil {
    if rec.PID != self && singletonIsAlive(rec.PID) {
        // Live incumbent ... Supersede it: a reconnect should win.
        _ = singletonSignal(rec.PID)
    }
}
```

**After (recommended — coexist instead of supersede when the incumbent is genuinely alive):**
```go
if rec := readMCPPIDRecord(path); rec != nil {
    if rec.PID != self && singletonIsAlive(rec.PID) {
        // A live incumbent for this (workspace, parent) already serves a
        // connection. We CANNOT tell from the pidfile whether the client
        // dropped the old pipe (reconnect → incumbent is a zombie that
        // should die) or whether both connections are live (concurrent
        // server → killing the incumbent breaks an active session).
        //
        // Killing a live-and-connected incumbent is the reported Codex
        // bug. Choose the safe policy: DO NOT signal it. Claim a distinct
        // pidfile keyed by our own pid so both daemons coexist; the
        // orphan/parent-liveness watchdog still reaps whichever daemon's
        // client actually dies. Duplicate-but-connected daemons are far
        // cheaper than a killed live session.
        return coexistRelease(path, self, ppid)
    }
}
```
plus a helper that writes a pid-suffixed pidfile so records don't collide:
```go
// coexistRelease claims a per-pid pidfile so a second live daemon on the
// same (workspace, parent) does not disturb the incumbent's record.
func coexistRelease(path string, self, ppid int) (func(), error) {
    alt := fmt.Sprintf("%s.%d", path, self)
    if err := writeMCPPIDRecord(alt, self, ppid); err != nil {
        return nil, err
    }
    return func() {
        if rec := readMCPPIDRecord(alt); rec != nil && rec.PID == self {
            _ = os.Remove(alt)
        }
    }, nil
}
```
**Why:** This is the direct fix for the reported bug — a live, connected incumbent is never
SIGTERM'd, so the agent's transport is never torn out mid-comparison. Reaping of genuinely
dead/orphaned daemons is retained by the existing parent-liveness watchdog (each daemon exits
when *its own* parent/connection dies) and by the stale-holder branch (dead incumbents are
still overwritten as free).

**Alternative (if strict single-daemon is required):** keep supersede, but **only** after a
liveness *and connection* check — e.g. probe the incumbent (signal 0 is not enough; add a
lightweight "am I still connected?" check such as a lock the incumbent holds only while its
stdin is open) and supersede solely when the incumbent's connection is provably gone. Heavier;
prefer the coexist policy above unless duplicate daemons prove costly.

### 3. Harden the parent-liveness watchdog against reparent false-positives
**File:** `internal/serve/mcp_watchdog.go` — `startParentWatchdog`.

**Before:**
```go
case <-ticker.C:
    if watchdogGetppid() != startPpid {
        watchdogExit(0)
    }
```
**After (exit only when reparented to init AND the original parent is actually gone):**
```go
case <-ticker.C:
    now := watchdogGetppid()
    if now != startPpid && (now == 1 || !singletonIsAlive(startPpid)) {
        // Reparented to launchd/init, or the original parent is confirmed
        // dead. A mere ppid change (intermediate wrapper exited, session
        // still live) is NOT sufficient to exit.
        watchdogExit(0)
    }
```
**Why:** Closes the secondary mid-session death vector where a legitimate reparent (parent
wrapper exits, session still alive) triggers `os.Exit(0)`. Requires `now == 1` (adopted by
init) or a confirmed-dead original parent, not just "ppid differs."

### 4. Recover panics in the request loop (defense in depth)
**File:** `internal/serve/mcp_lifecycle.go` — `handleRequest` (or wrap the call in `Run()`).

**Before:**
```go
s.handleRequest(&req)
```
**After:**
```go
func (s *MCPServer) handleRequest(req *JSONRPCRequest) {
    defer func() {
        if r := recover(); r != nil {
            s.logDebug("PANIC in %s: %v", req.Method, r)
            s.sendError(req.ID, ErrCodeInternal, fmt.Sprintf("internal error: %v", r))
        }
    }()
    switch req.Method {
    // ... unchanged ...
    }
}
```
**Why:** A tool-handler panic *would* crash the whole server → identical "transport closed"
symptom for every tool. Recovering turns it into one JSON-RPC error and keeps the session
alive. **Important — this is NOT the fix for either reported trigger:** the search batch and
the locked-index repro both returned gracefully (no panic path was found in the index /
retrieval / anchor handlers). Ship it as cheap, low-risk defense-in-depth, but do not present
it as the fix for the "transport closed" reports — that is Item 2 (Trigger A) and the
`database is locked` errors are already handled by Item 1b (Trigger B1).

---

## Test Plan

### Existing test review
| File | Coverage today |
|------|----------------|
| `internal/serve/mcp_singleton_test.go` | Supersede/stale-holder branches via `singletonIsAlive`/`singletonSignal` seams. **Add:** a "live-and-connected incumbent is NOT signaled" case for the new policy. |
| `internal/serve/mcp_watchdog_test.go` | Watchdog reparent-decision via `watchdogGetppid`/`watchdogExit` seams. **Extend:** ppid-changed-but-parent-alive (and not init) must **not** exit; reparent-to-1 or parent-dead must exit. |
| `internal/serve/mcp_test.go` | Drives `Run()` with buffers (guards are gated off there). Good for handler behavior; does not exercise the real-stdio guards. |
| `internal/index/index_test.go` | Index open/migrate/query coverage. **Add:** a concurrency test — a second connection holds a write lock; a query on the primary must **succeed within the busy-timeout** (not fail instantly with `database is locked`). |

### Test changes needed
1. **Singleton coexist (primary fix):** with a live incumbent record (`singletonIsAlive`
   returns true), assert `singletonSignal` is **never called** and `acquireMCPSingleton`
   returns a release that removes a pid-suffixed pidfile — not the incumbent's. Assert the
   incumbent's record is untouched.
2. **Singleton stale holder unchanged:** dead incumbent → still treated as free (overwrite),
   no signal. Regression guard for the retained reaping behavior.
3. **Watchdog false-positive:** `watchdogGetppid` returns a *new non-1* value while
   `singletonIsAlive(startPpid)` is true → `watchdogExit` **not** called. And: returns `1`
   (or original parent dead) → `watchdogExit(0)` called once.
4. **SIGTERM graceful release (integration):** start a real `hero mcp` (stdin held open),
   send SIGTERM, assert process exits promptly **and** `.hero/mcp-<ppid>.pid` is removed
   (no leak). Mirrors the manual repro that showed leaked pidfiles.
5. **Panic recovery:** register a temporary handler that panics, dispatch a `tools/call` to
   it, assert a JSON-RPC `ErrCodeInternal` error is returned and the loop continues to serve
   a subsequent request (process stays alive).
6. **Regression — two live daemons coexist:** the manual repro (two daemons, same parent,
   same workspace, both stdin held open) must now show the **incumbent staying alive** after
   the second starts (inverse of the reproduced failure).
7. **Index concurrency (Trigger B1):** hold an `EXCLUSIVE` lock on `.hero/index.db` from a
   second connection, then run `hero_search`/`hero_anchor` against the server. Before the fix
   they return `database is locked`; after (busy-timeout + WAL) they either succeed once the
   lock releases within 5s, or coexist via WAL. **Invariant either way:** the server never
   crashes and the transport stays open (the pre-fix behavior for this is *already* graceful —
   this test locks in that the tool now *succeeds* under contention, not just fails cleanly).

### Regression scope
- The whole point of `bcb9424` (reap orphaned/duplicate daemons) must still hold: verify
  orphaned daemons (parent dead → reparented) still exit via the watchdog, and that dead
  incumbents are still overwritten. The coexist policy trades "at most one daemon" for "never
  kill a live one" — confirm this does not reintroduce unbounded daemon accumulation (each
  daemon still self-exits when its own connection/parent dies).
- Cross-harness: the fix is harness-agnostic Go (`internal/serve`), so it applies to all six
  install targets uniformly. No `hero install` / instruction-file surface is touched, so the
  `harness-changes-cover-all-targets` tripwire does not apply — but validate the reconnect
  behavior in **Codex specifically** (the reporting harness) and in Claude Code (hook-driven
  reconnect) before closing.

---

## Kickoff

Paste-ready cold-start prompt for the fix session:

> Fix the Hero MCP server dying mid-session in Codex (spec:
> `.hero/planning/bugs/mcp-transport-closes-midsession-supersede/spec.md`).
>
> **Root cause (proven):** the singleton guard in `internal/serve/mcp_singleton.go`
> (`acquireMCPSingleton`, from commit `bcb9424`) SIGTERMs a live, serving `hero mcp` daemon
> the instant a second daemon starts for the same `(workspace, parent-pid)`. Reproduced: the
> incumbent dies within 1s. Codex sees this as "in-process search transport closed" and falls
> back to the CLI. The search path itself is clean (a 7-call `hero_search` batch responds and
> exits 0). This is a **design** flaw: the guard assumes "same-parent second spawn ==
> reconnect" without verifying the incumbent's connection is dead.
>
> **Do NOT revert `bcb9424`** — its orphan/duplicate-daemon reaping is real. Keep reaping
> genuinely-dead daemons; never kill a live, connected one.
>
> **A second session** reported the transport closing *and* an "index stale lock / schema
> mismatch." That was run down: a locked index and a graph schema mismatch both produce
> graceful `isError` tool results (reproduced) — they do **not** close the transport. So they
> are a **separate reliability defect** (Trigger B1: the index opens SQLite with no
> busy-timeout/WAL, so concurrent `hero next ingest`/`checkpoint` hook processes make tool
> calls fail with `database is locked`), not a second transport-close cause.
>
> **Implement the changes in `## Suggested Fix Approach`:**
> 1. `internal/serve/mcp_lifecycle.go` `Run()` — graceful SIGTERM handler that runs the
>    pidfile `release()` before exiting (stops leaked pidfiles AND leaked index journals).
> 1b. `internal/index/index.go` `Open` — add `_busy_timeout=5000` + `_journal_mode=WAL`
>    (match the graph). **This is the fix for the second session's `database is locked`.**
> 2. `internal/serve/mcp_singleton.go` — **coexist instead of supersede** when the incumbent
>    is genuinely alive (per-pid pidfile via a `coexistRelease` helper); **the load-bearing
>    fix for the transport close.**
> 3. `internal/serve/mcp_watchdog.go` — only `os.Exit(0)` when reparented to init (ppid==1)
>    or the original parent is confirmed dead, not on any ppid change.
> 4. `internal/serve/mcp_lifecycle.go` `handleRequest` — wrap dispatch in `recover()` as
>    defense-in-depth. **No panic path was found; this is not the fix for either report** —
>    ship it, but don't rely on it.
>
> **Then follow `## Test Plan`** — especially the inverse-repro regression (two live daemons
> sharing a parent + workspace must now **coexist**) and the index-concurrency test (a held
> lock must no longer make `hero_search`/`hero_anchor` fail).
>
> **Still open (capture, don't block the fix):** the exact Codex-host event that spawns the
> second daemon during comparison queries is not yet observed. Set `HERO_MCP_DEBUG=1` and
> reproduce in Codex to capture `.hero/mcp-debug.log` and confirm the trigger. The coexist
> fix is correct regardless of which event spawns the second daemon.
>
> Build target: `go build ./cmd/hero` (the CLI); the installed `~/go/bin/hero` that Codex
> runs must be rebuilt/reinstalled to pick up the fix.
