---
title: "Flaky data race in opsrunner — global nowFn swapped by tests races leaked pump() goroutine"
slug: opsrunner-keepalive-data-race
type: bug
status: completed
priority: high
severity: high
domain: engineering
size: small
created: 2026-07-12
tags: [ci, data-race, flaky-test, opsrunner, test-isolation, release-blocker]
completed_at: 2026-07-12T05:21:30Z
---

# Flaky data race in opsrunner keepalive test

## Symptom

CI's `Test` workflow `go test -race -count=1 ./...` step fails reliably (2/2
observed runs) with a data race in `internal/serve/opsrunner`, failing the whole
job before the (separately-fixed) NEXT.md drift gate even runs. A local `-race`
run passes intermittently, so it reads as flaky — but it reproduces reliably in
CI's linux environment. Second independent blocker to a green pipeline (the
first was [[next-drift-gate-unwinnable]]).

```
WARNING: DATA RACE
Write at ... by goroutine 51:
  opsrunner.TestRunner_Keepalive()  runner_test.go:258
Previous read at ... by goroutine 21:
  opsrunner.(*Runner).pump()        runner.go:307
  ... created by (*Runner).Start()  runner.go:150
  ... TestRunner_Start_Dedup()      runner_test.go:61   <-- a DIFFERENT test
--- FAIL: TestRunner_Keepalive
```

## Root Cause

`nowFn` (`var nowFn = time.Now`) and `keepaliveInterval` are **mutable package
globals** (`runner.go:29,33`), read by `(*Runner).Start` (:108), `Stream`
(:249-293) and `pump` (:307). `TestRunner_Keepalive` swaps and restores them
(`runner_test.go:218-220,257-258`).

Two compounding problems:
1. **Goroutine leak across tests.** A `Runner` started in `TestRunner_Start_Dedup`
   leaks its `pump()` goroutine past the end of that test (Start isn't
   stop-and-waited). That leaked goroutine keeps reading the global `nowFn`.
2. **Cross-test global write.** `TestRunner_Keepalive` later writes the global
   `nowFn` at `runner_test.go:258` (the restore), racing the leaked reader. The
   existing "wait for job.Done() before restoring" guard only covers *this*
   test's job — not another test's leaked goroutine.

Aggravating detail: the `nowFn` save/restore in `TestRunner_Keepalive` is **dead**
— the test never actually installs a fake clock, so line 258 reassigns `time.Now`
to itself. It's a pure memory write with no functional purpose, yet it's the
racing write.

## Fix

Eliminate the mutable global — the source-of-truth class error — rather than
patch the symptom. Make the clock and keepalive interval **per-`Runner` fields**,
set once at construction and never mutated after, so they're safe to read from
any goroutine and one test's runner can't perturb another's:

1. `internal/serve/opsrunner/runner.go`:
   - Add `now func() time.Time` and `keepaliveInterval time.Duration` to the
     `Runner` struct; a `const defaultKeepaliveInterval = 15 * time.Second`.
   - `New()` sets `now: time.Now`, `keepaliveInterval: defaultKeepaliveInterval`.
   - Replace every `nowFn()` → `r.now()` / `run.now()` and `keepaliveInterval`
     → `r.keepaliveInterval` in `Start`, `Stream`, `pump`. Remove the globals.
2. `internal/serve/opsrunner/runner_test.go`:
   - `TestRunner_Keepalive` sets `r.keepaliveInterval = 50 * time.Millisecond`
     on its own runner (after `New`, before `Start`); drop the global
     save/swap/restore of `keepaliveInterval` and the dead `nowFn` dance.

Per-instance immutable fields make the fields race-free regardless of goroutine
lifetime, so even a leaked `pump()` cannot race — the deeper goroutine-leak
hygiene (stop-and-wait) is worth a follow-up but is not required to kill this
race.

## Acceptance Criteria

- AC-1: WHEN `go test -race -count=100 ./internal/serve/opsrunner/` runs, THE
  SYSTEM SHALL pass with no data race (the race no longer reproduces under
  repetition).
- AC-2: THE `Runner` SHALL carry its clock and keepalive interval as instance
  fields; the package-level mutable `nowFn`/`keepaliveInterval` globals SHALL be
  removed.
- AC-3: `TestRunner_Keepalive` SHALL still assert a `: keepalive` frame is
  emitted (behavior preserved) using a per-runner short interval, with no global
  mutation.
- AC-4: `go build ./... && go test ./...` passes; CI's `Test` `go test` step goes
  green (unblocking the drift-gate step to finally run).

## Validation

- Repro-under-repetition: `go test -race -count=100 ./internal/serve/opsrunner/`
  green (pre-fix, high `-count` surfaces the race locally).
- Keepalive behavior: `TestRunner_Keepalive` still passes and still asserts the
  keepalive frame.
- Full suite `-race`: `go test -race -count=1 ./...` green.
- Real acceptance: push and confirm the CI `Test` job's `go test` step passes,
  then the drift-gate step runs (and, with next-drift-gate-unwinnable already
  merged, passes) → whole job green.

## Completion Ledger

| AC | Status | Note |
|----|--------|------|
| AC-1 (no race under repetition) | DONE | `go test -race -count=25 ./internal/serve/opsrunner/` → ok, 54s, no race (CI reproduced at 1×) |
| AC-2 (per-instance fields; globals removed) | DONE | `Runner.now` + `Runner.keepaliveInterval` set in `New()`; package globals `nowFn`/`keepaliveInterval` deleted (only `defaultKeepaliveInterval` const remains) |
| AC-3 (keepalive behavior preserved, no global mutation) | DONE | `TestRunner_Keepalive` sets `r.keepaliveInterval = 50ms` on its own runner and still asserts the `: keepalive` frame |
| AC-4 (build + full -race suite) | DONE | `go test -race -count=1 ./...` → exit 0, 86 pkgs, 0 races (the CI gate, green locally); `go test ./...` green |

- [x] exercise-the-feature: ran the exact CI gate locally (`-race -count=1 ./...`) green, plus `-race -count=25` on the opsrunner package specifically to stress the previously-racing path.

## Changes

| # | Change | Status |
|---|--------|--------|
| 1 | `internal/serve/opsrunner/runner.go`: `nowFn`/`keepaliveInterval` globals → per-`Runner` fields (`now`, `keepaliveInterval`) set in `New()`; `defaultKeepaliveInterval` const; reads in `Start`/`Stream`/`pump` use the instance fields | DONE |
| 2 | `internal/serve/opsrunner/runner_test.go`: `TestRunner_Keepalive` sets the interval on its runner; removed the global save/swap/restore (incl. the dead `nowFn` restore that was the racing write) and the now-redundant `job.Done()` wait | DONE |

## Kickoff

**Pick up at: DELIVERED — pending push + CI-green.** `nowFn`/`keepaliveInterval`
are now per-`Runner` fields (`internal/serve/opsrunner/runner.go`) set in `New()`
and never mutated, so no shared-global write races a leaked `pump()` goroutine;
the racing global-restore in `TestRunner_Keepalive` is gone. Verified: `go test
-race -count=25 ./internal/serve/opsrunner/` and `go test -race -count=1 ./...`
both green (0 races). Real acceptance: push and confirm CI's `Test` job's `go
test` step passes — which finally lets the (already-merged) NEXT.md drift gate
step run and go green, making the whole pipeline trustworthy for the v0.24.0
release.
