---
title: hero serve Project Section — Phase 2 Aggregate (/p/all/project)
slug: hero-serve-project-section-aggregate
type: feature
status: completed
priority: P2
tags: [hero-serve, dashboard, ui, project, aggregate, daemon]
created: 2026-05-19
relations:
  - target: hero-serve-project-section
    kind: parent
  - target: hero-serve-project-section-mvp
    kind: depends-on
  - target: hero-serve-multi-project
    kind: depends-on
horizon: now
---

# `hero serve` Project Section — Phase 2 Aggregate (`/p/all/project`)

## Context

This is **Phase 2 of 5** for the
[`hero-serve-project-section`](../../initiatives/hero-serve-project-section/spec.md)
initiative. See the parent for the full design rationale and boundaries.

Phase 2 adds the cross-project rollup at `/p/all/project`: a sortable
Project Directory, daemon-wide ops metadata, a cross-project health
rollup, and a cross-project peers map (table form). It depends on
Phase 1 having landed the `projectpage` package and on
`hero-serve-multi-project` having shipped the `/p/all/<page>` aggregate
routing slot.

## Kickoff

Aggregate cross-project view at `/p/all/project`. Reads the project
registry, the existing `/api/status` daemon-status source, and the
Phase 1 data loaders fanned out across every registered project.

**Status:** delivered — Phase 2 of 5. Phase 1's `projectpage` package
extended with an aggregate sibling; the existing /p/all/<page> routing
slot now dispatches /p/all/project through a project-aware aggregate
router.

**Shipped:**

- `internal/serve/projectpage/aggregate.go` — `RegisterAggregate` +
  aggregate handler with per-loader panic isolation.
- Four new aggregate-only loaders under
  `internal/serve/projectpage/data/`: `directory.go`,
  `health_rollup.go`, `peers_map.go`, `daemon_ops.go` (each with
  `_test.go`). Health-rollup color rule documented at the top of
  `health_rollup.go`.
- Page-local templates at `internal/serve/projectpage/templates/`:
  `page_all.html`, `directory.html`, `daemon_ops.html`,
  `health_rollup.html`, `peers_map.html`.
- `internal/serve/projectpage/static/project_all.js` — embedded into
  the page via `//go:embed` (no /static/projectpage route added).
- `internal/serve/server.go` — extended `buildAggregateShellRouter` to
  also register the Project aggregate; added
  `aggregateProjectpageProjects()` + `daemonOpsSnapshot()` helpers that
  feed the new loaders directly (no HTTP round-trip).
- `internal/serve/routing.go` — `allProjectsHandler` now dispatches
  "project" through the same aggregate router as "now" / "work".
- `internal/serve/api.go` — `POST /api/daemon/registry/refresh`
  (re-reads `~/.hero/projects.json` and re-syncs `server.projects`).
  GET form returns current state without reloading.

**Deferred:** the "Stop daemon" button (Phase 4); live `hero check`
refresh on the rollup (Phase 5); a graph visualization library for the
peers map (table is sufficient for v1); JSON-side row hot-swap (the
refresh button currently does a full page reload after the POST
succeeds — adequate for the read-only operator surface).

**Validation:** `go build ./...` clean; `go test ./...` clean;
`go vet ./...` clean. Aggregate route handler test renders all four
sections against a three-project fixture including one deliberately-
broken project (page returns 200, broken row carries the degraded
indicator). Registry-refresh endpoint test verifies POST returns JSON
with the current project list.

## Goal

Loading `/p/all/project` renders a Project Directory of every entry in
`~/.hero/projects.json`, a Daemon Ops section showing PID/port/uptime/
version/served-project count from `/api/daemon/status`, a Cross-Project
Health Rollup aggregating cached health across registered projects, and
a Cross-Project Peers Map showing which projects peer with which. The
Project tab in the top nav is highlighted.

## Acceptance Criteria

- WHEN a user loads `/p/all/project` THE SYSTEM SHALL render a Project
  Directory table of every entry in `~/.hero/projects.json` with
  columns for slug, name, root path, last-touched, rolled-up health
  status (green / yellow / red), spec count, peer count, and active
  tracker adapter.
- THE SYSTEM SHALL make the Project Directory client-side sortable and
  text-searchable via a small JS module (`project_all.js`).
- WHEN a user clicks a row in the Project Directory THE SYSTEM SHALL
  navigate to that project's `/p/<slug>/project`.
- WHEN the page loads THE SYSTEM SHALL render a Daemon Ops section
  showing daemon PID, listening port, uptime, version, and served-
  project count, sourced from the daemon-status endpoint defined by
  `hero-serve-multi-project`.
- WHEN a user clicks "Refresh registry" THE SYSTEM SHALL re-read
  `~/.hero/projects.json` server-side and re-render the directory
  without a full page reload.
- WHEN the page loads THE SYSTEM SHALL render a Cross-Project Health
  Rollup aggregating stale-spec / missing-kickoff / broken-manifest
  counts across all registered projects, with each rolled-up category
  drillable into a per-project breakdown.
- WHEN the page loads THE SYSTEM SHALL render a Cross-Project Peers
  Map as a table showing source-project, peer-alias, peer-project,
  cached reachability, and last successful call timestamp.
- WHEN the Project tab is the active page THE SYSTEM SHALL mark it
  active in the top nav for `/p/all/project` as well as the per-
  project URLs.
- IF a per-project loader fails for one project in the rollup THEN
  THE SYSTEM SHALL render that project's row with a degraded
  indicator and continue rendering the rest of the page.

### Health rollup color rules

- Green: zero health items.
- Yellow: only stale-spec or missing-kickoff items.
- Red: drift detected OR broken peer manifest OR orphan files.

Document the rule in `internal/serve/projectpage/data/health_rollup.go`
so it can evolve in one place.

## Approach

Add an `AggregateHandler` to `internal/serve/projectpage/aggregate.go`.
The handler reads the registry, fans out across all projects, reuses
the Phase 1 loaders for per-project data needed by the rollup (health
summary, peer list, spec count), and composes the result into
`project_all.html`.

Daemon Ops data comes from the existing
`/api/daemon/status` (or `/api/status` — see the parent initiative's
endpoint-collision open question) endpoint defined by
`hero-serve-multi-project`. Phase 2 does NOT define a new daemon-status
endpoint; if `hero-serve-multi-project` has only `/api/status`, Phase 2
uses that and aliases through the same handler.

Add a new `/api/daemon/registry/refresh` endpoint that re-reads
`~/.hero/projects.json` and re-emits the project list as JSON for the
"Refresh registry" button to consume.

## Changes

1. **Add `internal/serve/projectpage/aggregate.go`**
   - `AggregateHandler` for `/p/all/project`. Fans out registry,
     gathers per-project rollup data, renders
     `project_all.html`.
   - One-shot per-project loader failure isolation — a single
     project's failure must not block the page.

2. **Add aggregate-only data loaders**
   - `data/directory.go` — Project Directory rows + tests.
   - `data/health_rollup.go` — aggregated counts + color rule + tests.
   - `data/peers_map.go` — cross-project peers table + tests.
   - `data/daemon_ops.go` — wraps the daemon-status endpoint into a
     view struct + tests.

3. **Templates**
   - `internal/serve/shell/templates/project_all.html` — page
     composition extending `page-layout.html`.
   - `internal/serve/shell/templates/project_all/directory.html`,
     `daemon_ops.html`, `health_rollup.html`, `peers_map.html` —
     section partials.

4. **Register handler**
   - In `internal/serve/server.go` shell-page handler block, register
     `/p/all/project` alongside the Phase 1 `/p/{slug}/project`.

5. **API endpoint**
   - `POST /api/daemon/registry/refresh` — re-read
     `~/.hero/projects.json` and return the refreshed list as JSON.
   - Coordinate with `hero-serve-multi-project` on the
     `/api/daemon/status` vs `/api/status` name; alias whichever it
     shipped.

6. **Client behavior**
   - `internal/serve/shell/static/js/project_all.js` — client-side
     sort + text search on the Project Directory; fetch handler for
     "Refresh registry".

7. **Top-nav active-state**
   - Extend the Phase 1 active-state branch in `top-nav.html` to also
     match `/p/all/project`.

8. **Tests**
   - Unit tests for `directory.go`, `health_rollup.go`,
     `peers_map.go`, `daemon_ops.go`.
   - Handler test for `/p/all/project` rendering all four sections
     across a fixture registry with N projects, including one
     deliberately-broken project to verify failure isolation.
   - Test for `/api/daemon/registry/refresh` returning JSON.
   - Active-state test asserting Project tab is active on
     `/p/all/project`.

## Boundaries

- **"Stop daemon" button** is NOT in this phase — Phase 4 adds it
  alongside the destructive-ops flow (it dispatches through the ops
  runner landing in Phase 3).
- **Live `hero check` refresh on the rollup** is NOT in this phase —
  Phase 5 adds it. Phase 2 reads whatever cached state Phase 1's
  health loader returned.
- **Graph visualization** for the peers map is out of scope; a clean
  sortable table is acceptable for v1.
- **Defining a daemon-status endpoint** is out of scope. Phase 2
  consumes whatever `hero-serve-multi-project` shipped.
- **Per-project drill-down beyond a link** is out of scope. Clicking a
  row navigates to `/p/<slug>/project`; that's the drill-down.

## Risks

- **Daemon-status endpoint name collision**. If
  `hero-serve-multi-project` ships `/api/status` and this phase ships
  `/api/daemon/status`, both should point at the same handler.
  Mitigation: coordinate during Phase 2 delivery; pick one canonical
  name and alias the other.
- **Per-project loader fan-out cost**. Reading every project's data
  on every page load is N×Phase-1-cost. Mitigation: keep loaders
  cheap; Phase 5 adds the cache that makes the truly expensive
  loaders (Health) effectively free.
- **One bad project poisons the page**. Mitigation: explicit
  per-project failure isolation in `AggregateHandler` plus a
  fixture-based test.

## Validation

- Unit tests for all aggregate loaders pass.
- Handler test renders `/p/all/project` with a fixture registry of 3+
  projects including one with deliberately-broken inputs; all rows
  render; the broken project shows a degraded indicator; the page
  itself returns 200.
- Manual: open `/p/all/project`; the directory shows every registry
  entry; sort works; text search filters rows; clicking a row
  navigates.
- Manual: hit `POST /api/daemon/registry/refresh` directly; the
  endpoint returns the current registry as JSON.
- Manual: Project tab is highlighted on `/p/all/project` and remains
  highlighted on `/p/<slug>/project` (Phase 1 regression check).

## Changes (files touched on completion)

- `internal/serve/projectpage/aggregate.go` — new; `RegisterAggregate`
  + `aggregateHandler`; per-loader panic isolation; embeds
  `static/project_all.js` directly.
- `internal/serve/projectpage/data/directory.go` (+ `_test.go`) — new;
  Project Directory rows with per-row degraded indicator.
- `internal/serve/projectpage/data/health_rollup.go` (+ `_test.go`) —
  new; cross-project health rollup with the canonical color rule
  (Green / Yellow / Red) documented at file top.
- `internal/serve/projectpage/data/peers_map.go` (+ `_test.go`) — new;
  cross-project peering table with path-based peer-project resolution.
- `internal/serve/projectpage/data/daemon_ops.go` (+ `_test.go`) — new;
  wraps the daemon snapshot into a view struct (formatUptime helper).
- `internal/serve/projectpage/templates/page_all.html` — new; outer
  composition for the aggregate page.
- `internal/serve/projectpage/templates/directory.html`,
  `daemon_ops.html`, `health_rollup.html`, `peers_map.html` — new;
  page-local section partials following the Phase 1 partial style.
- `internal/serve/projectpage/static/project_all.js` — new;
  client-side sort + filter + Refresh-registry POST.
- `internal/serve/server.go` — extended `buildAggregateShellRouter` to
  register the project aggregate; added
  `aggregateProjectpageProjects()` and `daemonOpsSnapshot()` so the
  loader pulls from the in-process server without an HTTP round-trip.
- `internal/serve/routing.go` — `allProjectsHandler` switch extended
  to dispatch "project" through the aggregate router.
- `internal/serve/api.go` — added
  `POST/GET /api/daemon/registry/refresh` handler.
- `internal/serve/projectpage_aggregate_test.go` — new; handler
  integration tests for `/p/all/project` (multi-project + broken
  project), top-nav active-state on the aggregate, and the
  registry-refresh endpoint.
