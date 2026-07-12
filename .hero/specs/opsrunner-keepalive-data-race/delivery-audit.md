# Delivery audit — opsrunner-keepalive-data-race

**Audited:** `git diff HEAD -- internal/serve/opsrunner/`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria
- [✓] AC-1 (no race under repetition) — `go test -race -count=20 ./internal/serve/opsrunner/` → ok, 42.5s, no race reported.
- [✓] AC-2 (per-instance fields; globals removed) — `runner.go:58-59` adds `now func() time.Time` and `keepaliveInterval time.Duration` fields; `runner.go:30` keeps only `const defaultKeepaliveInterval`. `grep nowFn` = 0 hits; no package-level `var` for either global remains.
- [✓] AC-3 (keepalive behavior preserved, no global mutation) — `runner_test.go:222` sets `r.keepaliveInterval = 50ms` on its own runner; `runner_test.go:242` still asserts `": keepalive"`. No global save/swap/restore, no `nowFn` dance.
- [✓] AC-4 (build + full -race suite) — `go build ./...` exit 0; targeted `go test -run 'Keepalive|Dedup' -count=1` ok.

## Changes
- [✓] `internal/serve/opsrunner/runner.go` — globals `nowFn`/`keepaliveInterval` deleted; per-`Runner` fields added and set in `New()` (`now: time.Now`, `keepaliveInterval: defaultKeepaliveInterval`, lines 76-77). All read sites converted: `Start` StartedAt `r.now()` (:118); `Stream` `r.now()` (:259,296,303) + `r.keepaliveInterval` (:260,298); `pump` `run.now()` (:317, receiver is `run`). No missed `nowFn()` call.
- [✓] `internal/serve/opsrunner/runner_test.go` — `TestRunner_Keepalive` sets interval on its runner before `Start`; removed the global save/swap/restore (incl. the dead `nowFn` self-restore that was the racing write) and the now-redundant `job.Done()` wait. Stream goroutine still drained via `<-done` after `cancel()` (:238-239).

## Audit notes
- **Construction sites:** only one production caller — `internal/serve/server.go:170` (`opsrunner.New(...)`) — and one `&Runner{}` literal (inside `New` itself). No path constructs a `Runner` outside `New`, so `now` is never nil; no panic risk.
- **Immutability confirmed:** repo-wide grep for `.now =` / `.keepaliveInterval =` writes yields exactly one hit — `runner_test.go:222`, executed before `r.Start()` (line 223) and thus before any pump/Stream goroutine is spawned. No write after construction anywhere. Concurrent reads by leaked goroutines are race-free.
- **Race genuinely eliminated, not masked:** the two previously-racing tests now touch disjoint state — `TestRunner_Keepalive` mutates only its own runner's field (pre-Start), and a leaked `pump()` from `TestRunner_Start_Dedup` reads only its own runner's `now` (`time.Now`, never written by anyone). No shared mutable global remains for them to collide on. The leaked-goroutine hygiene issue persists (test doesn't stop-and-wait the subprocess) but is now harmless and correctly scoped as a follow-up in the spec.
- **Verification run by auditor:** `go build ./...` (exit 0); `go test ./internal/serve/opsrunner/ -run 'Keepalive|Dedup' -count=1` (ok, 0.74s); `go test -race -count=20 ./internal/serve/opsrunner/` (ok, 42.5s, 0 races).
- Diff is well-scoped to the two spec-named files; no scope drift.
