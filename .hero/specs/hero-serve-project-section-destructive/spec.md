---
title: hero serve Project Section — Phase 4 Destructive Operations (Registry, Danger Zone, Stop Daemon)
slug: hero-serve-project-section-destructive
type: feature
status: completed
priority: P2
tags: [hero-serve, dashboard, ui, project, destructive, registry, danger]
created: 2026-05-19
relations:
  - target: hero-serve-project-section
    kind: parent
  - target: hero-serve-project-section-mvp
    kind: depends-on
  - target: hero-serve-project-section-aggregate
    kind: depends-on
  - target: hero-serve-project-section-opsrunner
    kind: depends-on
horizon: now
---

# `hero serve` Project Section — Phase 4 Destructive Operations

## Context

This is **Phase 4 of 5** for the
[`hero-serve-project-section`](../../initiatives/hero-serve-project-section/spec.md)
initiative. See the parent for the full design rationale and boundaries.

Phase 4 lands every destructive affordance on the Project surface:

- **Registry remove** with a 5-second grace window and undo toast.
- **Danger Zone** section with typed-confirmation flow for deregister
  and archive.
- **Missing-path detection** with a one-click deregister banner when a
  registered project's root no longer exists on disk.
- **Stop daemon** button — placed on `/p/all/project` only, dispatched
  through Phase 3's `opsrunner`.

Destructive flows are higher product risk than caching plumbing, so
they land before Phase 5's perf work. The 5-second grace window is the
safety net: nothing irreversible happens in those 5 seconds.

## Kickoff

Destructive affordances on the Project surface delivered: registry-
remove with 5-second undo toast, Danger Zone (deregister only) with
typed-confirm gate, missing-path banner, and aggregate-only Stop-
daemon button dispatched through Phase 3's opsrunner.

**Status:** delivered — implementation complete, tests passing (`go build ./...`, `go test ./...`, `go vet ./...` all clean), committed in feature commit.

**Done:**
- `internal/serve/pending_remove.go` (queue + race-safe goroutine)
- `internal/serve/api.go` (registry/remove + remove/undo +
  /api/daemon/ops/stop)
- `internal/serve/server.go` (`RemoveProject` now persists registry
  to disk; primary shell router flagged as fallback)
- `internal/serve/opsrunner/allowlist.go` (`stop` verb +
  `DaemonScopedSlug` constant)
- `internal/serve/projectpage/data/{registry,danger}.go` and
  templates (`registry.html`, `danger.html`, `page.html`,
  `daemon_ops.html`)
- `internal/serve/projectpage/handler.go` (Deps.IsFallbackProject,
  MissingPath, inline JS for remove + danger gate)
- `internal/serve/projectpage/static/project_all.js` (Stop-daemon
  client behaviour)

**Skip:** undo windows longer than 5 seconds; admin/multi-user
permission gates; restart-daemon; archive verb (no top-level
`hero archive` exists — Snapshot has its own subcommand and is out
of scope for Danger Zone today).

## Goal

Operators can safely remove a project from the registry with a
visible 5-second undo affordance. The Danger Zone exposes
deregister/archive behind a typed-confirmation step. When a project's
root path disappears off disk, the page renders a banner offering
one-click deregister. The aggregate `/p/all/project` view has a Stop
daemon button gated by confirmation copy that explains the consequence.
Nothing destructive happens by accident.

## Acceptance Criteria

### Registry remove with grace window

- WHEN a user clicks "Remove from registry" on
  `/p/<slug>/project` THE SYSTEM SHALL POST to
  `/api/{slug}/registry/remove`, enqueue a pending-remove entry, and
  return immediately with a grace-window deadline.
- WHILE a remove is pending THE SYSTEM SHALL show an undo toast on
  the page for 5 seconds.
- WHEN the grace window elapses without undo THE SYSTEM SHALL invoke
  `RemoveProject` (`internal/serve/server.go:200`) and commit the
  removal.
- IF the user POSTs to `/api/{slug}/registry/remove/undo` within the
  grace window THEN THE SYSTEM SHALL cancel the pending-remove
  without ever calling `RemoveProject`.
- IF the daemon restarts during the grace window THEN THE SYSTEM
  SHALL drop the pending-remove (no destructive action survives a
  restart unconfirmed).

### Danger Zone

- WHILE the Danger Zone section is collapsed (the default) THE
  SYSTEM SHALL hide destructive operation buttons from the page.
- WHEN a user expands the Danger Zone THE SYSTEM SHALL reveal
  destructive operations (deregister, archive) each requiring a
  single typed-confirmation step where the user types the project
  slug to enable the action button.
- IF the typed confirmation does not match the project slug THEN
  the action button SHALL remain disabled.

### Missing path detection

- IF a registered project's root path no longer exists on disk on
  page load THEN THE SYSTEM SHALL display a warning banner offering
  one-click deregistration.
- WHEN a user clicks the deregister banner THE SYSTEM SHALL POST to
  `/api/{slug}/registry/remove` and proceed through the same grace-
  window flow as a normal removal.

### Stop daemon (aggregate-only)

- WHERE the user is on `/p/all/project` THE SYSTEM SHALL render a
  "Stop daemon" button inside the Daemon Ops section with a
  confirmation modal that explains stopping the daemon ends the
  current dashboard session.
- WHEN a user confirms "Stop daemon" THE SYSTEM SHALL dispatch
  `hero serve stop` through the Phase 3 ops runner (NOT via direct
  `os.Exit`), so the same SSE-progress and dedup machinery applies.
- THE SYSTEM SHALL NOT render the "Stop daemon" button on any per-
  project `/p/<slug>/project` URL.

## Approach

### Pending-remove queue

A small `internal/serve/registry/pending.go` carries an in-memory
map keyed by slug. Each entry stores `Slug`, `Deadline`, and a
`Cancel chan struct{}`. A goroutine waits on `time.After(deadline)`
or `cancel`; on deadline it calls `RemoveProject`. The queue is
flushed on daemon restart (pending removals do NOT survive — that's
intentional).

### Danger Zone

A new section partial
(`internal/serve/shell/templates/project/danger.html`) renders the
section collapsed by default. The JS in `project.js` gates the
confirm-button enabled state on the typed input matching the slug.
The section is omitted entirely from the page if the project is not
in the registry (no point showing destructive actions for an
unregistered project).

### Missing path detection

`projectpage.Handler` calls `os.Stat(projectRoot)` during `Deps`
construction. If `ErrNotExist`, the handler sets a `MissingPath`
flag on the template data; the template renders a top-of-page banner
that links to the same `/api/{slug}/registry/remove` endpoint via a
single button.

### Stop daemon

`/p/all/project` Daemon Ops template adds a Stop-daemon button. The
client posts `/api/daemon/ops/stop` which is a slim wrapper that
dispatches `hero serve stop` through the Phase 3 ops runner
(registered with a daemon-scoped slug like `"_daemon"`). The
confirmation modal is part of `project_all.js`.

`hero serve stop` itself is assumed to exist; if it does not, this
spec depends on landing it first. (Note for the implementer: if
`hero serve stop` is not yet a CLI verb, add it as a small
`cmd/hero/serve_stop.go` that POSTs to `/api/daemon/shutdown`, and
add the shutdown endpoint as well.)

## Changes

1. **Pending-remove queue**
   - `internal/serve/registry/pending.go` — in-memory pending-remove
     map with `Enqueue(slug, deadline, onCommit)` and `Cancel(slug)`.
   - Unit tests for the elapsed-vs-canceled paths.

2. **API endpoints in `internal/serve/api.go`**
   - `POST /api/{slug}/registry/remove` → enqueue pending-remove,
     return deadline.
   - `POST /api/{slug}/registry/remove/undo` → cancel pending-remove
     if present; return 200 either way.

3. **Wire Registry section button**
   - `internal/serve/projectpage/data/registry.go` — the Phase 1
     loader returns the existing fields plus a `CanRemove` flag.
   - `internal/serve/shell/templates/project/registry.html` — add
     the "Remove from registry" button gated by `CanRemove`.

4. **Danger Zone section**
   - `internal/serve/projectpage/data/danger.go` — loader returns
     the verbs available (deregister, archive) and any guardrails.
   - `internal/serve/shell/templates/project/danger.html` — section
     partial; collapsed by default; typed-confirm input.
   - Include `danger.html` in `project.html` after Config (last
     section).

5. **Missing path detection**
   - In `projectpage.Handler`, `os.Stat(projectRoot)` once during
     `Deps` build. Set `MissingPath` flag.
   - `internal/serve/shell/templates/project.html` — top-of-page
     banner rendered when `MissingPath` is true.
   - Unit test covering the missing-path branch.

6. **Client behavior in `project.js`**
   - Undo toast wiring: render on remove click, count down 5
     seconds, post undo if clicked.
   - Danger Zone typed-confirm gate.
   - Missing-path banner click handler.

7. **Stop daemon (aggregate)**
   - Add `_daemon` slug entry or equivalent special-case to the
     `opsrunner` allowlist for a `stop` verb mapping to
     `hero serve stop`.
   - `POST /api/daemon/ops/stop` API wrapper.
   - Template: `internal/serve/shell/templates/project_all/daemon_ops.html`
     gets a Stop-daemon button + confirmation copy.
   - `project_all.js` confirmation modal.
   - If `hero serve stop` does not yet exist as a CLI verb, add it
     under `cmd/hero/serve_stop.go` posting to a new
     `POST /api/daemon/shutdown` that triggers a clean
     `server.Shutdown(ctx)`.

8. **Tests**
   - Pending-remove queue unit tests (elapsed → committed; canceled
     → not committed; restart → flushed).
   - Handler test for `POST /api/{slug}/registry/remove` followed by
     undo within 5s: registry unchanged.
   - Handler test for `POST /api/{slug}/registry/remove` followed by
     no undo: `RemoveProject` invoked after grace.
   - Handler test for missing-path detection: `Stat` returns
     `ErrNotExist`, banner renders, full page returns 200.
   - Danger Zone test asserting buttons are hidden when collapsed
     and the typed-confirm gate works.
   - Stop-daemon test asserting button is present on
     `/p/all/project` and absent on `/p/<slug>/project`.

## Boundaries

- **Multi-step undo / longer than 5 seconds** is out of scope.
  5s is the UX-tested grace window.
- **Multi-user / admin permission gates** are out of scope; deferred
  to `hero-team-server`.
- **Restart daemon** is out of scope; only Stop is wired in v1.
  Restart is a follow-up.
- **Archive implementation details** (where archived projects live,
  whether they're recoverable) are not redefined here; this phase
  surfaces the affordance against whatever archive verb exists. If
  no archive verb exists, the button is omitted from Danger Zone in
  this phase and added in a follow-up.

## Risks

- **Pending-remove queue lost on restart**. By design — destructive
  actions should not silently survive a daemon restart unconfirmed.
  Mitigation: documented behavior + restart-during-grace test.
- **`hero serve stop` CLI verb may not exist**. Mitigation: if
  absent, add it as part of this phase per the Changes list.
- **Stop daemon kills the dashboard tab**. The user clicks Stop and
  their browser session dies mid-request. Mitigation: confirmation
  modal explains the consequence with restart instructions visible
  before the click.
- **Missing-path on a project that is the current daemon's
  `s.projectRoot`**. Could trigger a banner on the user's own home
  project before they've moved off it. Mitigation: banner only
  fires for `/p/<slug>/project`, not for the `/project` fallback;
  test covers that branch.

## Validation

- All unit tests for the pending-remove queue, the Danger Zone gate,
  and the missing-path branch pass.
- Integration test: POST remove → undo within 5s → `RemoveProject`
  never called → registry unchanged.
- Integration test: POST remove → wait 6s → `RemoveProject` called
  → registry no longer contains the slug.
- Manual: open `/p/<slug>/project`, click "Remove from registry";
  undo toast appears; click undo; registry unchanged.
- Manual: same as above without clicking undo; toast disappears
  after 5s; reload — the project is gone from
  `~/.hero/projects.json`.
- Manual: rename a project's root dir on disk; reload page; missing-
  path banner renders; click deregister; same grace-window flow.
- Manual: open Danger Zone; buttons disabled; type slug; buttons
  enable; click; action runs.
- Manual: open `/p/all/project`; Stop daemon button is present.
  Open `/p/<slug>/project`; Stop daemon button is absent.

## Changes (files touched on completion)

- `internal/serve/pending_remove.go` (new) — pending-remove queue +
  race-safe goroutine + tests in `pending_remove_test.go`
- `internal/serve/projectpage/data/danger.go` (new) + `danger_test.go`
- `internal/serve/projectpage/data/registry.go` — added `CanRemove`
- `internal/serve/projectpage/handler.go` — `pageData.MissingPath` +
  `Slug`, `Deps.IsFallbackProject` honoured, inline JS extended
  for undo toast and Danger Zone gate, banner styles added
- `internal/serve/projectpage/deps.go` — `IsFallbackProject` field
- `internal/serve/projectpage/templates/registry.html` — Remove
  button gated by `{{ if .CanRemove }}`
- `internal/serve/projectpage/templates/danger.html` (new)
- `internal/serve/projectpage/templates/page.html` — Danger Zone
  include + top-of-page missing-path banner
- `internal/serve/projectpage/templates/daemon_ops.html` — Stop-
  daemon `<details>` block with confirmation copy
- `internal/serve/projectpage/static/project_all.js` — Stop-daemon
  POST + inline "Daemon stopped" landing on connection loss
- `internal/serve/projectpage/destructive_test.go` (new) — missing-
  path banner present/absent + Danger Zone presence/visibility +
  Remove button presence/visibility
- `internal/serve/projectpage_aggregate_test.go` — Stop-daemon
  presence assertion on aggregate, absence assertion on per-project
- `internal/serve/api.go` — `/api/{slug}/registry/remove`,
  `/api/{slug}/registry/remove/undo`, `/api/daemon/ops/stop`,
  `/api/daemon/ops/{job}/stream`
- `internal/serve/registry_remove_api_test.go` (new) — undo within
  window, commit after window, GET-not-allowed, daemon-ops verb
  rejection
- `internal/serve/server.go` — `RemoveProject` persists registry to
  disk (`reg.Remove` + `reg.Save`); `pendingRemove` field; primary
  shell router built with `IsFallbackProject=true`
- `internal/serve/opsrunner/allowlist.go` — added `stop` verb,
  `DaemonScopedSlug` const, `VerbLabel("stop")` label;
  `allowlist_test.go` updated to whitelist the daemon-scoped entry
