---
title: "hero serve takes 5s to stop and exits with an error when a client holds an idle connection"
slug: followup-serve-shutdown
type: bug
status: completed
priority: P3
severity: low
root_cause_class: code
domain: engineering
size: small
created: 2026-10-07
tags: [serve, shutdown, http]
completed_at: 2026-10-07T02:08:59Z
---

# hero serve shutdown: drain promptly

## Goal

Ctrl+C on `hero serve` should exit promptly and cleanly while still letting real in-flight requests finish.

## Kickoff

`http.Server.Shutdown` waits up to 5s on connections that never sent a request (a browser preconnect), then `Run` returns `context deadline exceeded` (exit 1). `drainHTTP` fixes this:
- it tracks in-flight requests and closes leftover idle connections once none remain;
- it cancels request contexts after a 1s grace so SSE streams end;
- reaching the deadline is not an error.

Test with `go test ./internal/serve -run 'TestDrain|RunAndShutdown'`.

## Root Cause

`shutdown()` called `httpServer.Shutdown(ctx)` alone. The stdlib treats a never-used connection as active for 5s, and SSE handlers only end when their request context ends, which `Shutdown` never cancels.

## Fix

- `countInFlight` middleware.
- `BaseContext` from a cancellable request context.
- `drainHTTP`: stop keep-alives, run `Shutdown` concurrently, and close as soon as nothing is in flight. After 1s, cancel request contexts. Treat a deadline or `ErrServerClosed` as success.

## Acceptance Criteria

- **AC-1:** WHEN `hero serve` is stopped while a client holds a connection that has sent no request THE SYSTEM SHALL exit promptly (well under 1s) with no error.
- **AC-2:** WHILE a request is in flight at shutdown THE SYSTEM SHALL let it complete.
- **AC-3:** WHEN a streaming handler is open at shutdown THE SYSTEM SHALL end it after a short grace (about 1s, before the 5s deadline), and shutdown SHALL report no error.

## Changes

1. `internal/serve/server.go`: `inFlight`, `requestCancel`, `BaseContext`, `countInFlight`, `drainHTTP`.
2. `internal/serve/server_drain_test.go`.

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | AC-1: prompt exit with idle connection | DONE | `TestDrainClosesIdleNewConnectionsPromptly` (0.09s). Round 1: `real-exercise.log` is now captured script output. A real `hero serve --no-ui` with an idle TCP connection then SIGINT: origin/main 5.03s, exit 1, `context deadline exceeded`; fix 0.05s, exit 0 |
| 2 | AC-2: in-flight completes | DONE | `TestDrainLetsInFlightRequestsFinish`: a 300ms handler returns its body. Scope: requests honoring `r.Context()` get a 1s grace (previously up to 5s); the audit found no handler that depends on longer |
| 3 | AC-3: streams end after grace | DONE | `TestDrainEndsStreamingHandlersAfterGrace`: 1.08s, nil error |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | server.go drain | DONE | `TestServer_RunAndShutdown` still passes |
| 2 | tests | DONE | — |

### Exercise-the-feature check

- [x] Real `hero serve` process with a browser-style idle connection, old and new builds, timed (`real-exercise.log`).

### Excellence Bar self-check

- [x] Yes. It keeps graceful draining for real work and removes the dead wait.
