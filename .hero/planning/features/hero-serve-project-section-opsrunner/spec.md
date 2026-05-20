---
title: hero serve Project Section — Phase 3 Operations Runner
slug: hero-serve-project-section-opsrunner
type: feature
status: planning
priority: P2
tags: [hero-serve, dashboard, ui, project, operations, sse]
created: 2026-05-19
relations:
  - target: hero-serve-project-section
    kind: parent
  - target: hero-serve-project-section-mvp
    kind: depends-on
horizon: now
---

# `hero serve` Project Section — Phase 3 Operations Runner

## Context

This is **Phase 3 of 5** for the
[`hero-serve-project-section`](../../initiatives/hero-serve-project-section/spec.md)
initiative. See the parent for the full design rationale and boundaries.

Phase 3 lands the `opsrunner` package and wires the Operations section
on the per-project page from "disabled placeholder" to "real button
that runs a `hero` command and streams progress." The runner is also
the dispatch target for Phase 4's "Stop daemon" button.

The verb allowlist is fixed (no shell pass-through), each verb maps to
a known `hero` CLI invocation, and in-flight ops per `slug+verb` are
deduplicated.

## Kickoff

`opsrunner` package + Operations section wiring. Lets the user run
`hero check`, `hero index`, `hero scan`, etc. from the Project page
with SSE progress.

**Status:** planning — Phase 3 of 5; gated on Phase 1's `projectpage`
package landing.

**Pick up at:** scaffold `internal/serve/opsrunner/` with the verb
allowlist map, in-memory job registry keyed by `slug+verb`, subprocess
launcher, and SSE writer. Register `POST /api/{slug}/ops/{verb}` and
`GET /api/{slug}/ops/{job_id}/stream` in `internal/serve/api.go`.

→ `.hero/planning/features/hero-serve-project-section-opsrunner/spec.md`

**Files:** `internal/serve/projectpage/data/operations.go` (new),
`internal/serve/api.go:51-132`,
`internal/serve/shell/templates/project/operations.html` (new),
`internal/serve/shell/static/js/project.js` (extend with SSE wiring)

**Skip:** verbs outside the fixed allowlist; "Stop daemon" wiring
(Phase 4 plumbs through this runner); persistence of job history across
daemon restarts (in-memory is fine for v1).

## Goal

Clicking any of the seven Operations buttons on `/p/<slug>/project`
starts the corresponding `hero` command server-side, returns a job id
immediately, and streams progress to the page via SSE until the
subprocess exits. Duplicate-start affordances are disabled while a job
of the same verb is in flight for the same project. The Operations
section becomes a usable replacement for typing the CLI command in a
terminal.

## Acceptance Criteria

- WHEN a user clicks a Lifecycle / Operations button on
  `/p/<slug>/project` for any verb in the fixed allowlist THE SYSTEM
  SHALL start the corresponding `hero` command server-side, return a
  job id immediately, and stream progress via server-sent events to
  the page.
- THE SYSTEM SHALL maintain a fixed verb allowlist of: `re-scan`,
  `re-index`, `run-check`, `refresh-queue`, `capture-knowledge`,
  `snapshot`, `export`.
- IF a verb is not in the allowlist THEN `POST /api/{slug}/ops/{verb}`
  SHALL return 400 without launching anything.
- WHILE a lifecycle operation is in flight THE SYSTEM SHALL disable
  duplicate-start affordances for that verb on that project and show
  inline progress.
- IF a user POSTs to start the same `slug+verb` while one is already
  in flight THEN THE SYSTEM SHALL return the existing job id rather
  than spawning a second subprocess.
- WHEN the underlying subprocess exits (success or failure) THE
  SYSTEM SHALL emit a final SSE event carrying the exit code and
  close the stream.
- IF the subprocess exits non-zero THEN THE SYSTEM SHALL render the
  Operations row in an error state with the last 200 bytes of stderr
  visible in the inline progress.
- THE SYSTEM SHALL scope every subprocess invocation to the project's
  root via `cmd.Dir = projectRoot` rather than relying on process-wide
  cwd.
- WHEN a client disconnects from the SSE stream THE SYSTEM SHALL
  continue running the underlying subprocess (the job is not tied to
  the stream) but SHALL release the SSE writer cleanly.

## Approach

`internal/serve/opsrunner/` has three pieces:

- `runner.go` — public API: `Start(ctx, slug, verb) (jobID, error)`,
  `Stream(ctx, slug, jobID, w http.ResponseWriter) error`,
  `Lookup(slug, verb) (jobID, ok)`.
- `registry.go` — in-memory `map[string]*Job` keyed by
  `"<slug>:<verb>"` with sync primitives. A `Job` carries
  `JobID`, `Slug`, `Verb`, `StartedAt`, `Done chan struct{}`,
  `ExitCode`, and a ring buffer of recent output lines so a late
  subscriber gets backfill.
- `allowlist.go` — `var Verbs = map[string][]string{ "re-scan":
  {"scan"}, "re-index": {"index"}, "run-check": {"check"},
  "refresh-queue": {"queue", "write"}, "capture-knowledge":
  {"capture"}, "snapshot": {"snapshot"}, "export": {"export"} }` —
  the canonical verb→arg mapping. The `hero` binary path resolves
  from `os.Executable()` so the subprocess invokes the same binary
  the daemon is running from.

The SSE writer emits one JSON event per output line plus a final
`{"type":"exit","code":N}` event. The handler sets the standard SSE
headers (`Content-Type: text/event-stream`, `Cache-Control: no-cache`,
`Connection: keep-alive`) and flushes after every event.

## Changes

1. **Create `internal/serve/opsrunner/` package**
   - `runner.go`, `registry.go`, `allowlist.go` as above.
   - Unit tests for the allowlist mapping (every verb maps to a
     concrete arg list), the dedup behavior, and SSE termination on
     subprocess exit.

2. **API endpoints in `internal/serve/api.go`**
   - `POST /api/{slug}/ops/{verb}` → resolve slug to a project,
     validate verb, call `opsrunner.Start`, return `{job_id}`.
   - `GET /api/{slug}/ops/{job_id}/stream` → call `opsrunner.Stream`
     to write SSE to the response.

3. **Wire Operations section in `internal/serve/projectpage/`**
   - `data/operations.go` — section loader reading allowlist verbs
     and current in-flight jobs for the slug. Returns a struct the
     template renders.
   - `internal/serve/shell/templates/project/operations.html` — one
     button per allowlist verb with a slot for inline progress.
   - Include `operations.html` in the Phase 1 `project.html` section
     order (it sat as a placeholder or omitted in Phase 1).

4. **Client behavior**
   - Extend `internal/serve/shell/static/js/project.js` with:
     - Click handler that POSTs `/api/{slug}/ops/{verb}` and opens an
       `EventSource` on the returned stream URL.
     - Renders each progress event into the row.
     - Re-enables the button on the final exit event.

5. **Dep wiring**
   - Add `*opsrunner.Runner` to `projectpage.Deps` (the Phase 1 stub
     field gets filled in).
   - Construct the runner once in `internal/serve/server.go` startup
     and inject it into the handler's per-request `Deps` builder.

6. **Tests**
   - `opsrunner` unit tests as listed above.
   - Handler test for `POST /api/{slug}/ops/{verb}` validating the
     allowlist enforcement (400 for unknown verb).
   - Integration test: start `run-check` via the API, observe SSE
     events, assert final exit event arrives, assert second
     `Start("<slug>", "run-check")` returns the same job id while the
     first is in flight.

## Boundaries

- **Verbs outside the fixed allowlist** are out of scope. No
  shell-pass-through, no user-provided commands.
- **Cancellation of in-flight jobs** is out of scope for v1.
  (Subprocess survives client disconnect; killing it is a follow-up.)
- **Job history persistence across daemon restarts** is out of scope.
  In-memory only.
- **The Stop-daemon button** is NOT in this phase. Phase 4 adds it on
  the aggregate page and dispatches through this runner.
- **A team-server job queue backend** is out of scope. Phase 3's
  runner is a swap-in seam for the future `hero-team-server` queue.

## Risks

- **Subprocess exit not flushed to SSE**. If the writer is closed
  before the final event flushes, the client sees the row stuck "in
  flight" forever. Mitigation: explicit final event + flush before
  returning from the stream handler; integration test asserts the
  event arrives.
- **Client disconnect leaks goroutines**. The SSE goroutine must exit
  when the request context is canceled. Mitigation: thread `ctx`
  through; select on `ctx.Done()` alongside the event channel; unit
  test that disconnects mid-stream.
- **`hero` binary not on PATH**. Resolve via `os.Executable()` rather
  than searching PATH so the subprocess always runs the same binary
  the daemon does.
- **Long-running ops with no output for minutes**. SSE clients may
  disconnect if the daemon's reverse proxy times the stream out.
  Mitigation: emit a keepalive comment line every 15s if no real
  output has been emitted; documented as a known-good behavior.

## Validation

- All `opsrunner` unit tests pass.
- Handler test for the allowlist enforcement returns 400 for an
  unknown verb.
- Integration test: starting `run-check` returns a job id, SSE
  streams progress, the final exit event arrives, the registry
  releases the slot on exit.
- Manual: open `/p/<slug>/project`, click each of the seven
  Operations buttons against a real project; progress streams; the
  button re-enables after exit; clicking the same button while in
  flight reuses the existing job rather than spawning another.

## Changes (files touched on completion)

- `internal/serve/opsrunner/runner.go` (new)
- `internal/serve/opsrunner/registry.go` (new)
- `internal/serve/opsrunner/allowlist.go` (new)
- `internal/serve/opsrunner/*_test.go` (new)
- `internal/serve/projectpage/data/operations.go` (new)
- `internal/serve/projectpage/deps.go` (`Runner` field filled in)
- `internal/serve/shell/templates/project/operations.html` (new)
- `internal/serve/shell/templates/project.html` (include operations
  partial)
- `internal/serve/shell/static/js/project.js` (SSE client wiring)
- `internal/serve/api.go` (`/api/{slug}/ops/{verb}` and
  `/api/{slug}/ops/{job_id}/stream`)
- `internal/serve/server.go` (runner construction + injection)
