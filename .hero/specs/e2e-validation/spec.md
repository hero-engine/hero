---
title: E2E Validation — Repeatable Take-It-For-A-Spin Smoke Test
slug: e2e-validation
type: feature
status: completed
priority: P1
tags: [testing, validation, dx, polish, observability]
created: 2026-04-27
relations:
  - target: hero-killer-features
    kind: related
horizon: now
smoke: deferred
completed_at: 2026-06-09T19:18:11Z
---

## Goal

A repeatable end-to-end test that exercises the workflows a new user
would actually touch in their first session with hero in an unfamiliar
codebase. Run produces a markdown observation log with timing, exit
codes, output volume, and a curated UX assessment. Re-run on every
significant release to catch regressions and surface polish targets.

## What's shipped

- [`scripts/e2e_smoke.sh`](../../../../scripts/e2e_smoke.sh) — the
  runnable script. Defaults to `go-task/task` but takes any
  `<owner>/<repo>` as a positional arg. Captures per-step timing,
  exit code, output size, and an output snippet into a markdown
  observation log under `tmp/e2e-smoke/<repo>-<ts>/`.

- [Run-1 observations](../../../../tmp/e2e-smoke/go-task-task-20260428T011642Z/observations.md)
  for `go-task/task` — first baseline of how hero feels to a new user.

## Findings from run 1 (2026-04-27)

### Real bugs

- **`hero ask "anything?"` crashes** on FTS5 syntax error when the
  query contains `?`. Sanitize at the query layer.
- **`hero ask` returns "No knowledge found"** even when the
  ContextDoc clearly exists in the workspace. `search` finds it,
  `ask` doesn't — they're hitting different indexes.
- **`hero relevant` returns 0 bytes** when nothing matches. Silent
  success is bad UX.

### UX rough edges

- **Search drowns in commit messages.** Searching "Task" on
  `go-task/task` returned 100 results, 99 of which were git commit
  messages. Default search ranking should weight commits below
  spec/knowledge/code nodes, or commits should be opt-in via flag.
- **Package context entries inflate `hero status`.** A 33-package
  repo gets 33 auto-generated `"Package: X"` Context entries that
  drown the real Context entries (`architecture-overview`,
  `dev-workflow`, `project-overview`). Either tag them as a
  separate node type or hide from `status` by default.
- **No new-user "where am I?" tour.** After init+scan a user has
  41 specs, 974 graph nodes, and 7 generated knowledge stubs with
  no obvious entry point. Consider a `hero tour` or
  `hero status --intro` mode.
- **`hero graph` with no args errors.** `hero graph stats` is a
  great overview — make it the default.
- **`hero relevant` requires `--files`** instead of accepting
  positional paths like other file-arg commands.
- **`hero init` non-interactive nature is undocumented.** It's
  already non-interactive but the `--help` doesn't confirm that.
  Users writing CI scripts will guess at a `--no-input` or `--ci`
  flag (and we should probably add one as a no-op for clarity).

### What worked well

- `scan` output — substantive, fast, well-organized
- `suggest` — high-churn-no-coverage list is genuinely actionable
- `graph stats` — clean type breakdown, right level of detail
- Empty-state copy on `next` / `blocked` / `check conflicts`
- Speed — every command <100ms

## How to run

```bash
# Default: go-task/task
scripts/e2e_smoke.sh

# Any other public repo
scripts/e2e_smoke.sh charmbracelet/bubbletea
scripts/e2e_smoke.sh httpie/httpie

# Preserve the clone across runs (faster re-runs)
KEEP=1 scripts/e2e_smoke.sh

# Alternate hero binary
HERO_BIN=/tmp/hero-cloud-new scripts/e2e_smoke.sh
```

Each run drops a markdown log under `tmp/e2e-smoke/<repo>-<ts>/`
that contains the per-step results plus a stub for operator
observations. The operator section is filled in by hand after
reviewing the captured outputs.

## Future runs to do

- **Cross-language**: re-run on a Python project (`httpie/httpie`)
  and a TypeScript project (`vadimdemedes/ink`) to validate the
  scan quality outside hero's native Go ecosystem.
- **After the polish work above lands**, re-run on `go-task/task`
  and confirm the same step list produces a cleaner observation log.
- **Larger repo** (~100k LOC) to stress-test scan + graph DB at
  realistic enterprise scale.

## Out of scope

- Federation flows (push/pull) — covered by the
  [Phase 7c live test](../graph-memory-7c-live-test/spec.md)
- LLM-driven flows (`hero ask` with model invocation, agent
  integration) — needs a credentialed run, separate harness
- Interactive flows (`hero spec new -i`, `hero hooks setup`) —
  smoke tests are by design non-interactive

## Files

| File | Purpose |
|---|---|
| `scripts/e2e_smoke.sh` | The runnable script |
| `tmp/e2e-smoke/<repo>-<ts>/observations.md` | Per-run log (generated) |
| `tmp/e2e-smoke/<repo>-<ts>/<step>.txt` | Captured stdout/stderr per step |

## Completion Ledger

| # | Item | Status | Evidence |
|---|------|--------|----------|
| SC-1 | `scripts/e2e_smoke.sh` runnable script exists, accepts `<owner/repo>` arg, captures per-step timing + exit code + output into markdown log | DONE | `scripts/e2e_smoke.sh` — bash script with `run_step` helper that times, captures, and appends per-step markdown entries. Supports `HERO_BIN`, `KEEP`, positional `<owner/repo>` arg. |
| SC-2 | Script produces markdown observation log under `tmp/e2e-smoke/<repo>-<ts>/` | DONE | Script creates `${RUN_DIR}/observations.md` with `section` and `run_step` helpers. Per-step `.txt` files captured alongside. |
| SC-3 | Run-1 executed on `go-task/task`, findings documented | DONE | Run-1 findings documented in `## Findings from run 1` section: 3 real bugs (FTS5 crash on `?`, `ask` hitting wrong index, `relevant` silent on zero matches) and 6 UX rough edges captured. |
| SC-4 | Future run plan documented for cross-language and post-polish re-runs | DONE | `## Future runs to do` section names Python (`httpie/httpie`), TypeScript (`vadimdemedes/ink`), and post-polish re-run as next steps. |

### Exercise-the-feature check

- [x] Exercised: `scripts/e2e_smoke.sh` exists and is executable. Script structure verified — `run_step`, `section`, timing, markdown log, per-step txt capture all present. Run-1 findings documented in spec. `go build ./...` clean.
