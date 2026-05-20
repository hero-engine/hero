---
type: feature
status: planning
tags: [hero-serve, dashboard, ui, project, operations, daemon, registry, peers]
created: 2026-05-19
relates-to:
  - hero-serve-multi-project
  - hero-serve-dashboard-redesign
  - hero-team-server
depends-on:
  - hero-serve-multi-project
---

# `hero serve` Project Section — Per-Project Info, Utilities, and Operations Page

## Context

The `hero serve` daemon runs once on the user's machine and is intentionally
multi-tenant: a single process manages many registered projects via
`~/.hero/projects.json` (`internal/serve/registry.go:44`,
`loadRegistryProjects` at `internal/serve/server.go:231`). The dashboard
top nav already includes a **Project** tab, but its content has never been
designed — today it is sparse, empty-state-heavy, and configuration-form-
dominated, with no way to see the project's own identity, health, peers,
or registry state at a glance.

Two adjacent specs reshape the surrounding surface:

- `.hero/planning/features/hero-serve-multi-project/spec.md` introduces
  `/p/<slug>/<page>` routing, a top-nav project selector, an
  `/p/all/<page>` aggregate slot, and per-page `Deps` that read the
  active project from the URL rather than `s.projectRoot`. This Project
  page **depends on** that routing landing.
- `.hero/planning/features/hero-serve-dashboard-redesign/spec.md`
  reworks **Now** (activity feed) and **Work** (spec catalog) into the
  primary surfaces. Project must NOT duplicate either — Now is "what
  happened lately," Work is "the spec catalog," Project is "this project
  itself."

Hero's mission is to make the next session start smarter than the last one
ended. The Project page is exactly the "stuff nobody told the agent"
surface: identity, health, registry membership, peers, lifecycle ops.
Today all of that lives in the CLI (`hero check`, `hero peer list`,
`hero status`, `hero snapshot`, `hero index`, `hero scan`) or in raw
JSON under `~/.hero/projects.json`. There is no UI home.

No active tripwires conflict with this design.

## Kickoff

A per-project home page in `hero serve` — identity, health, conventions,
peers, registry membership, lifecycle ops, danger zone — plus an
`/p/all/project` cross-project rollup with daemon-ops.

**Status:** planning — spec just landed; routing dependency
(`hero-serve-multi-project`) still delivering.

**Pick up at:** wait for `/p/<slug>/<page>` routing + per-page `Deps`
seam to land in `hero-serve-multi-project`, then scaffold a
`projectpage.Deps`, register `/p/{slug}/project` + `/p/all/project`
handlers in `internal/serve/server.go`, and create
`internal/serve/shell/templates/project.html` with collapsible sections.

→ `.hero/planning/features/hero-serve-project-section/spec.md`

**Files:** `internal/serve/server.go:308-370`, `internal/serve/api.go:51-132`,
`internal/serve/registry.go:44`, `internal/serve/shell/templates/page-layout.html`,
`.hero/planning/features/hero-serve-multi-project/spec.md`

**Skip:** in-browser `hero.json` editing (read-only + edit-in-editor link);
project creation wizard (separate spec); auth/multi-user (deferred to
`hero-team-server`).

## Goal

Opening `/p/<slug>/project` in the dashboard shows the project's identity,
health, stack, conventions, registry membership, peers, integrations,
knowledge stats, and lifecycle controls on a single scannable page — with
collapsed-by-default sections for items that are usually clean. From this
page, an operator can run the routine maintenance commands they would
otherwise type in a terminal (`hero check`, `hero index`, `hero scan`,
`hero peer list`, `hero snapshot`), and can manage registry membership
(remove, deregister). Opening `/p/all/project` shows the registry as a
directory plus daemon-wide ops (PID, port, uptime, version, served-
project count), with a cross-project health rollup and a peer-relationship
map. Neither view duplicates the Now activity feed or the Work spec
catalog; both cross-link to those pages.

## Acceptance Criteria

### Per-project view (`/p/<slug>/project`)

- WHEN a user loads `/p/<slug>/project` for a registered project THE
  SYSTEM SHALL render an identity header showing project name, root
  path (clickable to open in the OS file manager via a `file://` link),
  git remote, default branch, last-touched-at, spec/notes/decisions
  counts, and peer count.
- WHEN the page loads THE SYSTEM SHALL display a Health section
  populated from cached `hero check` output (TTL configurable; default
  5 minutes) showing drift, stale specs (status=`delivering`,
  untouched >30d), missing kickoffs, broken peer manifests, orphan
  files, and index staleness, with each row linked to the offending
  spec or file.
- WHEN every health check passes THE SYSTEM SHALL collapse the Health
  section to a single green "All clear" row.
- WHEN a user clicks "Refresh now" on the Health section THE SYSTEM
  SHALL re-run the check server-side and stream progress without a
  full page reload.
- WHEN the page loads THE SYSTEM SHALL render a Stack and Conventions
  section listing the detected stack (from the `stack-detection`
  skill's persisted output), active conventions under
  `.hero/knowledge/conventions/`, and the three most-recently-added
  conventions, each linked to its spec.
- WHEN the page loads THE SYSTEM SHALL render a Registry Membership
  section reporting whether the project is present in
  `~/.hero/projects.json`, its slug, registration timestamp, and
  whether it is set as default.
- WHEN a user clicks "Remove from registry" THE SYSTEM SHALL show a
  confirmation, call `RemoveProject` (`internal/serve/server.go:200`),
  and present an undo toast for 5 seconds before finalizing.
- WHEN the page loads THE SYSTEM SHALL render a Peers section listing
  registered sibling repos from `peer-manifest.yaml`, each with
  cached reachability status, last successful call timestamp, and
  in-flight handoff counts (in/out), with buttons to call peer
  (advisory mode), hand off a spec, or inspect the peer manifest.
- IF a peer's reachability cache is stale THEN THE SYSTEM SHALL show
  a "probe" affordance rather than blocking the page on a live probe.
- WHEN the page loads THE SYSTEM SHALL render a Trackers and
  Integrations section showing the configured tracker (Jira / GitHub
  / Linear), last successful sync timestamp, queued import count, and
  any sync errors.
- IF no tracker is configured THEN THE SYSTEM SHALL display a single
  opt-in card (not a recurring nag) linking to the integration setup
  docs.
- WHEN the page loads THE SYSTEM SHALL render a Knowledge Stats
  section showing counts of decisions, conventions, notes, captures
  and the last-captured-at timestamp, linking out to the Knowledge
  page.
- WHEN a user clicks a Lifecycle / Operations button (`re-scan`,
  `re-index`, `run health check`, `refresh queue`, `capture
  knowledge`, `snapshot`, `export`) THE SYSTEM SHALL start the
  corresponding `hero` command server-side, return a job id, and
  stream progress via server-sent events to the page.
- WHILE a lifecycle operation is in flight THE SYSTEM SHALL disable
  duplicate-start affordances for that operation and show inline
  progress.
- WHEN the page loads THE SYSTEM SHALL render a read-only Config
  section displaying the merged contents of `hero.json` (and
  `hero.local.json` if present) with an "Open in editor" link that
  resolves to a local `file://` path; the section SHALL be
  collapsed by default.
- WHEN a user expands the Danger Zone THE SYSTEM SHALL reveal
  destructive operations (remove from daemon, deregister, archive)
  each requiring a single typed-confirmation step.
- WHILE the Danger Zone is collapsed (the default) THE SYSTEM SHALL
  hide destructive operation buttons from the page.
- IF a registered project's root path no longer exists on disk THEN
  THE SYSTEM SHALL display a warning banner offering one-click
  deregistration.

### Cross-project rollup (`/p/all/project`)

- WHEN a user loads `/p/all/project` THE SYSTEM SHALL render a
  Project Directory table of every entry in `~/.hero/projects.json`
  with columns for slug, name, root path, last-touched, rolled-up
  health status (green / yellow / red), spec count, peer count, and
  active tracker adapter; sortable and text-searchable client-side.
- WHEN a user clicks a row in the Project Directory THE SYSTEM SHALL
  navigate to that project's `/p/<slug>/project`.
- WHEN the page loads THE SYSTEM SHALL render a Daemon Ops section
  showing daemon PID, listening port, uptime, version, and served-
  project count, sourced from the `/api/status` endpoint defined by
  `hero-serve-multi-project`.
- WHERE a future `hero serve stop` command is available THE SYSTEM
  SHALL surface a "Stop daemon" button in Daemon Ops with a
  confirmation explaining that stopping the daemon ends the current
  dashboard session.
- WHEN a user clicks "Refresh registry" THE SYSTEM SHALL re-read
  `~/.hero/projects.json` and re-render the directory without a full
  page reload.
- WHEN the page loads THE SYSTEM SHALL render a Cross-Project Health
  Rollup aggregating stale-spec / missing-kickoff / broken-manifest
  counts across all registered projects, with each rolled-up
  category drillable into a per-project breakdown.
- WHEN the page loads THE SYSTEM SHALL render a Cross-Project Peers
  Map (table or simple visual) showing which projects peer with
  which.

### Cross-cutting

- THE SYSTEM SHALL serve all per-project endpoints under
  `/api/{slug}/...` and all daemon-wide endpoints under
  `/api/daemon/...`, matching the namespacing established by
  `hero-serve-multi-project`.
- THE SYSTEM SHALL build a fresh `projectpage.Deps` per request from
  the URL slug rather than reading `s.projectRoot`.
- WHEN the Project tab is the active page THE SYSTEM SHALL mark it
  active in the top nav.
- IF `hero-serve-multi-project` routing is not yet active THEN THE
  SYSTEM SHALL serve a single-project Project page at `/project`
  using `s.projectRoot` as a temporary fallback.

## Approach

### Page architecture

A new `projectpage` package under `internal/serve/projectpage/` mirrors
the shape of `nowpage` and `workpage`. It exposes:

- `Deps` struct carrying `ProjectRoot`, `HeroDir`, `Slug`, a
  `RegistryEntry` snapshot, and shared services (health cache, peer
  cache, op-runner) — built per request by the handler from the URL
  slug, consistent with the per-page Deps refactor in
  `hero-serve-multi-project`.
- `Handler` for `/p/{slug}/project` rendering a single template,
  `internal/serve/shell/templates/project.html`.
- `AggregateHandler` for `/p/all/project` rendering
  `internal/serve/shell/templates/project_all.html`, fanning out across
  the registry using the same pattern Now and Work use in
  `hero-serve-dashboard-redesign`.

### Section model

Each section on the page is a small server-rendered partial. The page
arrives with everything pre-rendered for first-paint speed; sections
that have a "refresh" affordance re-fetch their own HTML via a small
JS helper rather than rebuilding the whole page. This keeps the page
fast without introducing a heavy reactive framework.

Sections are collapse-by-default when typically clean (Danger Zone,
Config, Knowledge Stats) and expand-by-default for sections that
demand operator attention (Identity, Health, Peers when reachability
is degraded). Collapse state is persisted in `localStorage` keyed by
project slug, so individual operators can shape their own default
view per project.

### Health cache

`hero check` is expensive enough that running it on every page load
is unacceptable. A `HealthCache` (`internal/serve/healthcache/`)
stores the most recent check result per project with a TTL (default
5 minutes, configurable in `hero.json` under `serve.health_ttl`). The
page renders the cached result with a visible "as of" timestamp and a
"Refresh now" button that invalidates the cache and re-runs the
check, streaming progress.

The cache lives in-process for now; persistence across daemon restarts
is not required (the cache repopulates on first request).

### Peer cache

Peer reachability is probed lazily. The cached reachability lives in
the same `HealthCache` keyed by `slug+peer-alias`. The page never
blocks on a live probe; instead, it renders cached values with an age
indicator and lets the operator trigger a probe manually. This avoids
turning page load into an N-peer round-trip.

### Operation runner

Lifecycle operations (`re-scan`, `re-index`, `run check`, `refresh
queue`, `capture knowledge`, `snapshot`, `export`) are dispatched
through a new `opsrunner` package (`internal/serve/opsrunner/`):

- The handler accepts `POST /api/{slug}/ops/<verb>`, validates the
  verb against an allowlist, and starts the underlying `hero`
  command as a subprocess scoped to the project's root.
- A job id is returned immediately.
- Progress streams to the page via SSE on
  `GET /api/{slug}/ops/<job_id>/stream` until the underlying process
  exits.
- The page disables duplicate-start affordances while a job is in
  flight (per project, per verb) using a small in-memory job
  registry.

The verb allowlist is fixed (no shell-pass-through), and each verb
maps to a known `hero` CLI invocation. This is forward-compatible with
the future `hero-team-server` job queue but does not depend on it —
the runner is a small local component that can be swapped for the
team-server queue later by changing the dispatch target.

### Daemon-ops endpoints

`/api/daemon/status` mirrors the `/api/status` endpoint introduced by
`hero-serve-multi-project` but is the canonical name for daemon-level
state used by this page. If the multi-project spec lands first with
`/api/status`, this page reuses that endpoint and does not introduce
a duplicate; the renames remain coordinated through the dependency.

Stop and restart are deferred to `hero serve stop` / `hero serve
restart` CLI commands; the page button calls those commands via the
ops runner and displays a "dashboard will disconnect" warning before
issuing the call.

### Templates

Two new templates:

- `internal/serve/shell/templates/project.html` — per-project page
  composed of section partials.
- `internal/serve/shell/templates/project_all.html` — aggregate page.

Both extend the shared `page-layout.html`. The top-nav rendering in
`top-nav.html` gets a Project tab active-state branch.

### Forward-compat seams

- All endpoints accept project context via URL slug → Deps, never via
  `s.projectRoot`. This matches the seam shape of
  `hero-serve-multi-project` and means the Project page works
  identically in single-project and multi-project modes.
- The cross-project peers map and the daemon-ops section are the
  natural extension points where `hero-team-server` will plug shared-
  job-queue and team-presence data in.

## Changes

1. **Routing dependency check**
   - Confirm `hero-serve-multi-project` has landed before merging this
     spec's per-project routing changes.
   - During an interim window, ship a single-project fallback at
     `/project` that uses `s.projectRoot` so the page is available
     even before multi-project routing lands.

2. **Create `internal/serve/projectpage/` package**
   - `deps.go` — `Deps` struct with `ProjectRoot`, `HeroDir`, `Slug`,
     `RegistryEntry`, `HealthCache`, `PeerCache`, `OpsRunner`.
   - `handler.go` — `Handler` for `/p/{slug}/project`.
   - `aggregate.go` — `AggregateHandler` for `/p/all/project`.
   - `sections/` — one file per section (`identity.go`, `health.go`,
     `stack.go`, `registry.go`, `peers.go`, `trackers.go`,
     `knowledge.go`, `operations.go`, `config.go`, `danger.go`).

3. **Create `internal/serve/healthcache/` package**
   - In-process per-project cache for `hero check` output.
   - TTL configurable via `hero.json` `serve.health_ttl`, default
     5m.
   - Cache key: `slug` → `{result, timestamp, ttl}`.

4. **Create `internal/serve/opsrunner/` package**
   - Allowlisted verb → `hero` CLI command map.
   - In-memory job registry keyed by `slug+verb` to deduplicate
     in-flight ops.
   - `POST /api/{slug}/ops/{verb}` → start, return job id.
   - `GET /api/{slug}/ops/{job_id}/stream` → SSE progress.

5. **Add per-project API endpoints**
   - `/api/{slug}/health` — cached health result + age.
   - `/api/{slug}/health/refresh` — invalidate + re-run.
   - `/api/{slug}/peers` — peer list with cached reachability.
   - `/api/{slug}/peers/{alias}/probe` — refresh reachability for
     one peer.
   - `/api/{slug}/stack` — detected stack + active conventions.
   - `/api/{slug}/registry` — registry entry view (GET, DELETE,
     PATCH for default-flag).
   - `/api/{slug}/config` — read-only merged `hero.json` view.
   - `/api/{slug}/knowledge/stats` — counts and last-captured-at.
   - All implemented in `internal/serve/api.go` near the existing
     `/api/projects` routing (currently `internal/serve/api.go:51-132`).

6. **Add daemon-wide API endpoints**
   - `/api/daemon/status` — PID, port, uptime, version, served
     project count. If `hero-serve-multi-project` already provides
     `/api/status`, alias or replace one consistently.
   - `/api/daemon/registry/refresh` — re-read
     `~/.hero/projects.json` and re-emit the project list.
   - Plumb deregister/remove operations through existing
     `RemoveProject` (`internal/serve/server.go:200`).

7. **Register page handlers**
   - In `internal/serve/server.go` shell-page handler block
     (currently lines 308-370), register `/p/{slug}/project` and
     `/p/all/project` alongside Now, Work, Knowledge.
   - Add `/project` fallback handler for single-project mode.

8. **Templates**
   - `internal/serve/shell/templates/project.html` — page composition
     with collapsible sections, per-project.
   - `internal/serve/shell/templates/project_all.html` — registry
     directory + daemon ops + cross-project rollup.
   - Extend `top-nav.html` to mark the Project tab active when the
     current page is a project page.

9. **Client behavior**
   - Small JS module under
     `internal/serve/shell/static/js/project.js` for:
     - Section collapse/expand with `localStorage` persistence keyed
       by `<slug>:<section>`.
     - SSE wiring for ops progress.
     - Per-section "refresh" partial fetches.
     - Undo toast for 5-second registry removal grace window.
   - No new framework. Use the existing JS conventions from
     `hero-serve-dashboard-redesign`.

10. **Health rollup color rules**
    - Green: zero health items.
    - Yellow: only stale-spec or missing-kickoff items.
    - Red: drift detected OR broken peer manifest OR orphan files.
    - Document the rule in `internal/serve/healthcache/README.md` so
      it can evolve in one place.

11. **Registry removal flow**
    - `POST /api/{slug}/registry/remove` → mark as pending-remove
      with a 5-second grace window; the server commits to
      `RemoveProject` after the grace expires unless the page
      issues `POST /api/{slug}/registry/remove/undo`.
    - The page renders a toast with an undo button during the grace
      window.

12. **Path-missing detection**
    - On Project page load, `stat(projectRoot)` — if absent, render
      the missing-path banner and offer one-click deregister.
    - Add a unit test for this branch.

13. **Update top-nav active-state logic**
    - Active for any URL matching `/p/<slug>/project` or
      `/p/all/project` or the `/project` fallback.

14. **Tests**
    - Unit tests for `healthcache` TTL behavior and invalidation.
    - Unit tests for `opsrunner` job dedup and SSE termination.
    - Handler tests for `/p/{slug}/project` rendering all sections.
    - Handler tests for `/p/all/project` aggregate rendering.
    - Integration test for the registry-removal grace window
      (request → undo → no removal; request → wait → removal).
    - Integration test for missing-path detection.

15. **Documentation**
    - Add a short page under `docs/cli/serve.md` (or equivalent) on
      the Project page sections and ops verbs.
    - Update `docs/dashboard/` (if present) with screenshots and the
      collapse-by-default rationale.

16. **Knowledge capture**
    - After delivery, capture a decision: "Project page is operator-
      home, not activity-feed-home — boundary between Now / Work /
      Project pages."
    - Capture a convention: "Collapse-by-default sections for usually-
      clean operator UI; expand-by-default for items demanding
      attention."

## Boundaries

- **In-browser `hero.json` editing** is out of scope. Config is
  read-only with an "Open in editor" link. Editing config in-browser
  is a separate spec.
- **Authentication and multi-user permissions** are out of scope.
  Destructive operations (remove project, stop daemon, archive)
  assume the single-operator local case and will be re-evaluated in
  the `hero-team-server` work.
- **Project creation wizard** is out of scope. This page is for
  projects that are already registered in `~/.hero/projects.json`.
  Onboarding flows belong in a separate spec.
- **Renaming a project / changing slug** is out of scope. The
  graph-wide identity story (across notes, decisions, NEXT.md
  references) is not settled.
- **Migrating specs between projects** is out of scope.
- **Cross-project search** is out of scope — covered by
  `hero-serve-multi-project` (per-project `/api/{slug}/search`) and
  the dashboard redesign command bar.
- **Activity feed on the Project page** is out of scope — Now owns
  that. The Project page cross-links to Now and Work but never
  duplicates them.
- **A heavy frontend framework** is out of scope. Server-rendered
  partials + small JS module, matching existing dashboard
  conventions.
- **Persistent health cache across daemon restarts** is out of
  scope; in-process cache is sufficient.
- **A real graph visualization library** for the peers map is out of
  scope. A clean table is acceptable for v1. Visual layout can come
  later.

## Risks

- **Routing dependency**. This spec depends on `hero-serve-multi-
  project` for `/p/<slug>/<page>` routing and the per-page `Deps`
  refactor. Until that lands, only the single-project `/project`
  fallback is available. Mitigation: ship the fallback first; gate
  the `/p/all/project` aggregate behind multi-project landing.
- **Health computation cost**. Running `hero check` on every load
  would be unusable. Mitigation: `HealthCache` with a default 5-
  minute TTL and an explicit "Refresh now" affordance.
- **Peer reachability noise**. Live-probing every peer on every
  page load is slow and noisy. Mitigation: cached reachability with
  manual refresh.
- **Daemon stop kills the dashboard**. Surfacing "Stop daemon" in
  the UI means the user's current dashboard session dies the moment
  they click it. Mitigation: clear confirmation copy explaining the
  consequence, plus restart instructions visible before the click.
- **Registry edits are destructive**. Removing a project from the
  registry should be reversible. Mitigation: 5-second grace window
  with undo toast; nothing irreversible happens in those 5 seconds.
- **Missing project root path**. If a user deletes a project
  directory off disk, the page must not crash or render garbage.
  Mitigation: `stat` the root on load and render a deregister
  banner.
- **SSE behind reverse proxies**. If a user fronts `hero serve`
  with a proxy that buffers, op progress streams will lag.
  Mitigation: document `serve` is local-only for now; revisit when
  `hero-team-server` introduces remote serving.
- **Endpoint name collision with `hero-serve-multi-project`**. If
  that spec ships `/api/status` and this spec ships
  `/api/daemon/status`, both should point at the same handler.
  Mitigation: coordinate during delivery; pick one canonical name
  and alias the other.
- **Information density tipping into overwhelm**. Ten sections is
  a lot. Mitigation: collapse-by-default for usually-clean
  sections; persist per-operator collapse state in `localStorage`.

## Validation

- **Unit tests** for `healthcache` (TTL, invalidation, concurrent
  reads), `opsrunner` (dedup, SSE termination on subprocess exit,
  verb allowlist enforcement), and section renderers (deterministic
  HTML given fixture inputs).
- **Handler integration tests** for `/p/{slug}/project` and
  `/p/all/project` rendering all sections without panic across:
  - registered project, clean state
  - registered project, drift / stale specs / broken peer manifest
  - registered project, root path missing on disk
  - unregistered slug (should 404 cleanly)
- **Integration test** for the registry-removal grace window with
  both undo-within-5s and let-it-elapse paths.
- **Integration test** for the ops runner: starting `hero check`
  via the UI returns a job id, streams progress, and terminates
  cleanly on completion. Repeat with cancellation mid-stream.
- **Manual verification**:
  - Open `/p/<slug>/project` on a real project; every section
    renders and links to the expected target.
  - Trigger each ops verb; progress streams; subsequent identical
    triggers are deduplicated until the first completes.
  - Remove a project from the registry, then undo within 5s, then
    confirm the registry is unchanged.
  - Delete a project's root directory on disk, reload the page,
    confirm the missing-path banner appears.
  - Open `/p/all/project`; the directory shows every registry
    entry; sort/search works; clicking a row navigates.
  - Confirm the Project tab is highlighted in the top nav for all
    project URLs (per-project, aggregate, and fallback).
- **Performance check**: page load with cached health and peer
  state completes in <200ms on a project with 100+ specs and 5
  peers.
- **Cross-link sanity**: the page links into Now and Work but does
  not duplicate their content (no activity feed, no spec catalog).

## Open questions

- **Daemon-status endpoint name**. Settle on `/api/status` vs
  `/api/daemon/status` during joint delivery with
  `hero-serve-multi-project`. Either is fine; pick one and alias.
- **Stop-daemon button placement**. Only on `/p/all/project`, or
  also on each `/p/<slug>/project`? Current design says aggregate-
  only, since stopping the daemon is not a per-project concern.
- **Knowledge stats time series**. The spec calls for "over time"
  counts; v1 may ship as totals only, with sparkline / time-series
  deferred to a follow-up.
