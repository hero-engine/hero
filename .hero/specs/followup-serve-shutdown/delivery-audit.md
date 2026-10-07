# Delivery audit — followup-serve-shutdown

**Audited:** `git diff 27add446...3925c865` on top of `git show 27add446` (clean detached worktree at 3925c865), round 2
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria
- [✓] AC-1: prompt, error-free exit with an idle never-used connection — `drainHTTP` closes once `inFlight == 0` and treats `ErrServerClosed`/`DeadlineExceeded` as success (`internal/serve/server.go`). `TestDrainClosesIdleNewConnectionsPromptly` passes (0.09s).
  - `real-exercise.log` is now captured script output: origin/main exited after 5.03s with exit 1 and `context deadline exceeded`; the fix exited after 0.05s with exit 0.
  - The auditor independently reproduced the fix half in round 1: 0.08s, exit 0, PID file removed. `server.go` is unchanged in 27add446...3925c865, so that result still applies.
- [✓] AC-2: in-flight request completes — `TestDrainLetsInFlightRequestsFinish` passes. The ledger now states the scope: requests that honour `r.Context()` get a 1s grace. No handler depends on a longer one: `opsRunner.Start` ignores the request ctx, and chat adapter dispatch uses `context.Background()`.
- [✓] AC-3: streams end after the grace, no error — `TestDrainEndsStreamingHandlersAfterGrace` passes (1.08s, nil error).

## Changes
- [✓] `internal/serve/server.go`: `inFlight`, `requestCancel`, `BaseContext`, `countInFlight`, `drainHTTP`, wired into `shutdown()`.
- [✓] `internal/serve/server_drain_test.go`: three tests asserting on timing, error and body.

## Open items
- None.

## Audit notes
- None. `go test ./internal/serve -count=1 -race` passes with no race reported. The only round-1 observation left is that a request arriving mid-shutdown in the gap between the `inFlight` check and `Close()` is dropped. That is inherent to shutdown and needs no action.
