# Delivery audit — followup-serve-shutdown

**Audited:** `git show 27add446` (worktree at 27add446)
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria
- [✓] AC-1: prompt, error-free exit with an idle never-used connection — `drainHTTP` closes the server once `inFlight == 0` (`internal/serve/server.go:1773-1779`) and treats `ErrServerClosed`/`DeadlineExceeded` as success. `TestDrainClosesIdleNewConnectionsPromptly` passes (0.09s). Real exercise was independently reproduced by the auditor: built `./cmd/hero` at 27add446, ran `hero serve --port 47391 --no-ui`, held a raw TCP connection, sent SIGINT. The process exited 0 in 0.08s, printed "shutting down...", and the PID file was removed. The old-build comparison (5.04s, exit 1) was not reproduced.
- [✓] AC-2: in-flight request completes — `countInFlight` keeps the server open while a handler runs. `TestDrainLetsInFlightRequestsFinish` (300ms handler returns "done") passes. Caveat in notes.
- [✓] AC-3: streams end after about 1s grace, no error — `BaseContext` plus `requestCancel` after a 1s timer (`server.go:1768-1771`). `TestDrainEndsStreamingHandlersAfterGrace` passes (1.09s, nil error).

## Changes
- [✓] `internal/serve/server.go`: `inFlight`, `requestCancel`, `BaseContext`, `countInFlight`, `drainHTTP`, and `shutdown()` now calls `drainHTTP`.
- [✓] `internal/serve/server_drain_test.go` — three tests asserting on timing, error and body.

## Open items
- None in the ledger.

## Audit notes
- `go test ./internal/serve -count=1 -race` passes, including `TestDrain*` and `TestServer_RunAndShutdown`. No race reported.
- AC-2 holds for requests that finish within the 1s grace or do not watch their context. A handler still running after 1s that honours `r.Context()` is now cancelled at 1s. Previously it had until the 5s deadline. No handler relies on `r.Context()` surviving: `opsRunner.Start` ignores the request ctx and the chat adapter dispatch uses `context.Background()`. The only ctx-bound work is the SSE loops (intended), `opsRunner.Stream`, the synchronous runner-free slash dispatch and the `shell.go` timeout.
- Known benign window: `inFlight` is counted only once a handler is entered. A request whose headers are mid-read, or one that arrives on an already-open connection between the `inFlight.Load()==0` check and `Close()`, is dropped. It is a request arriving mid-shutdown and cannot be fully avoided.
- `real-exercise.log` is a one-line prose summary, not captured output or timings. The auditor's own run substitutes for the new-build half.
- `requestCancel` is set once per `Run`. A `Server` reused across `Run` calls would rebind it. No such reuse was found.
