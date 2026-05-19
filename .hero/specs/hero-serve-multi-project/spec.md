---
type: feature
status: completed
tags: [serve, daemon, dashboard, multi-project, lifecycle]
relates-to: [hero-team-server, monorepo-satellite-installs]
created: 2026-05-19
---
# hero serve — multi-project lifecycle and dashboard awareness

## Context

`hero serve` is already a **global, single-instance daemon** that serves every
project registered in `~/.hero/projects.json`. The plumbing is right:

- Registry at `internal/serve/registry.go:44` (`~/.hero/projects.json`).
- Multi-project load via `loadRegistryProjects` in `internal/serve/server.go:231`.
- Project-namespaced API: `/api/projects` and `/api/{project}/...` in
  `internal/serve/api.go:51-132`.

But three things make the daemon feel single-project to the human operator:

1. **No lifecycle controls.** There's no `hero serve stop`, no `--force`, no
   `hero serve status`. Re-running `hero serve` after a crash or hung process
   produces a raw `address already in use` error with no guidance.
2. **No visibility into what it's actually serving.** From any directory, the
   user can't see whether a daemon is running, on what port, with which PID,
   or how many projects it knows about.
3. **The dashboard hides multi-project reality.** Page handlers at
   `internal/serve/server.go:308-370` hardcode the launching project's
   `s.projectRoot` and `s.heroDir` into every page (Now, Work, Knowledge,
   People, Agents). The top nav has no project selector. The URL has no slug.
   The command bar search hits `/api/search?q=...` with no project param. The
   API is multi-project; the UI is not.

This spec aligns with Hero's mission — *make the next session start smarter
than the last one ended, and raise the floor for everyone* — by making
cross-project awareness the default experience. The daemon already knows
about every project; the operator should too.

This is **groundwork** for two future specs the user has signaled: a broader
"Projects section" (per-project info, health, switching, utilities) and a
cross-project Dashboard view. URL routing and the selector decided here must
extend cleanly to both. Don't over-build now; don't box us in either.

## Kickoff

Adds `hero serve stop`, `--force`, `hero serve status`, and a working project
selector in the dashboard so the global daemon stops pretending it only serves
one project.

**Status:** completed — Phase 1 (lifecycle) shipped in `b288086`; Phase 2
(routing + selector) shipped in `d987b99` (commit message names another item
but contains this spec's Phase 2 source — see scope notes below).

**Items 7/8 plug into:** `nowpage.Deps`, `workpage.Deps`, `knowledgepage.Deps`,
`agentspage.Deps`, `peoplepage.Deps`, `projectpage.Deps` already carry
per-project `ProjectRoot` + `HeroDir`. The aggregate-view seam is the
`/p/all/<page>` stub renderer in `internal/serve/routing.go:130-160`
(`allProjectsHandler`); items 7/8 replace those stubs with per-page
aggregate renderers.

→ `.hero/planning/features/hero-serve-multi-project/spec.md`

**Files:** `internal/cli/serve.go`, `internal/cli/serve_lifecycle.go`,
`internal/serve/server.go`, `internal/serve/api.go`,
`internal/serve/pidfile.go`, `internal/serve/lifecycle.go`,
`internal/serve/routing.go`, `internal/serve/shell/shell.go`,
`internal/serve/shell/types.go`,
`internal/serve/shell/templates/top-nav.html`,
`internal/serve/shell/static/islands/project-selector.js`.

**Skip:** per-project daemons or port-per-project — global daemon is intentional; team-server work is a separate spec.

## Goal

After this ships:

- `hero serve` writes a PID file on startup and cleans it on shutdown.
- `hero serve stop` terminates a running daemon idempotently from any
  directory.
- `hero serve --force` replaces an existing daemon without manual cleanup.
- `hero serve status` prints a friendly summary of the running daemon
  (PID, port, uptime, project count, project list) and exits non-zero when no
  daemon is running.
- Address-in-use errors are diagnosed: hero daemon vs. foreign process, with
  actionable next-step text.
- The dashboard top nav has a project selector. URLs carry an active project
  slug (`/p/<slug>/now`, etc.). Page handlers read the project from the URL,
  not from `s.projectRoot`. An `/p/all/...` aggregate mode renders cross-
  project views for the pages where aggregation is meaningful.
- Old bookmarks (`/now`, `/work`, etc.) keep working via redirect to the
  default project.
- Command bar search is project-scoped from the URL, and supports a "search
  across all projects" mode in the All Projects view.

## Approach

### Phase 1 — Daemon lifecycle (stop, status, --force)

**PID file.** Write `~/.hero/serve.pid` at daemon start, immediately after
`net.Listen` succeeds in `internal/serve/server.go` (around line 387). File
contents are JSON, not a bare PID, so `status` can read port and start time
without an HTTP round trip:

```json
{"pid": 12345, "port": 7437, "started_at": "2026-05-19T10:30:00Z", "version": "0.x.y"}
```

Rationale for JSON over bare PID: a status probe should be able to print
something useful even if the daemon is wedged and not answering HTTP.

**PID file path per port.** When `--port` is set to a non-default, write
`~/.hero/serve-<port>.pid` instead. This lets two daemons on different ports
coexist (e.g. a dev daemon on 7437 and a scratch daemon on 7438). The default
port keeps the bare filename for backwards-friendliness.

**`hero serve stop`.** Subcommand in `internal/cli/serve.go`:
1. Read PID file. If absent → print "no hero daemon is running" and exit 0
   (idempotent).
2. Probe `/api/status` on the recorded port first to confirm the process is
   actually a hero daemon — guards against PID reuse.
3. Send SIGTERM. Wait up to 5 seconds (poll every 200ms) for the process to
   exit and the listener to free.
4. If still alive after timeout, send SIGKILL.
5. Remove the PID file on success.
6. If the PID is dead but the file is stale, remove it and exit 0 ("cleaned
   up stale PID file at <path>").
7. Accept `--port` to target a non-default daemon's PID file.

**`hero serve --force`.** Flag on the existing start command. Implementation:
call the same stop logic as a precondition; only error out if stop itself
fails. Race-on-force: if two `hero serve --force` run simultaneously, both
try to stop, both try to listen — one wins the listen, the other fails with
the improved bind error. Acceptable; we won't double-lock.

**`hero serve status`.** Subcommand in `internal/cli/serve.go` and a matching
`/api/status` HTTP endpoint in `internal/serve/api.go`.

The HTTP endpoint returns:
```json
{
  "running": true,
  "pid": 12345,
  "port": 7437,
  "started_at": "2026-05-19T10:30:00Z",
  "uptime_seconds": 3600,
  "version": "0.x.y",
  "project_count": 3,
  "projects": [
    {"slug": "hero", "name": "Hero", "path": "/Users/developer/projects/hero"},
    {"slug": "hero-cloud", "name": "Hero Cloud", "path": "/Users/developer/projects/hero-cloud"}
  ]
}
```

Note: `/health` already exists at `internal/serve/api.go:48`. Keep it as a
lightweight liveness probe; `/api/status` is a richer, project-aware view.
Don't conflate them.

The CLI:
1. Reads PID file (try default path, then `--port` override).
2. If no file → "no hero daemon is running" → exit 1.
3. If file present → GET `http://127.0.0.1:<port>/api/status` with a short
   timeout (2s).
4. If HTTP succeeds → print friendly multi-line summary.
5. If HTTP fails → file is stale or daemon is wedged. Probe the PID with
   `os.FindProcess` + signal 0. If dead, say so and suggest `hero serve stop`
   to clean up. If alive but unresponsive, say so and suggest `--force`.

**Improved bind-collision error.** In `Run()` at the `net.Listen` failure
site (around line 388), detect `EADDRINUSE` (use `errors.Is` with
`syscall.EADDRINUSE`). On detection, probe `http://127.0.0.1:<port>/api/status`:
- If it returns a hero status response → print: *"a hero daemon is already
  running on :7437 (PID NNN) and serves all your projects — you don't need
  a second one. Use `hero serve status` to inspect, or `hero serve stop`
  (or `hero serve --force`) to terminate."*
- If it does not respond, or responds with a non-hero payload → *"port 7437
  is in use by another process (not a hero daemon). Try `hero serve --port
  <other>` or free the port."*

### Phase 2 — Dashboard project awareness

**URL shape: `/p/<slug>/<page>`.**

Considered alternatives:
- `?project=<slug>` query param — bad: breaks bookmarking, hard to make the
  active project feel persistent, ugly with deep links.
- `/<slug>/<page>` with no prefix — bad: collides with future top-level
  routes and with reserved page names if a project is ever named `now`.
- `/p/<slug>/<page>` — chosen. The `/p/` prefix is a clear namespace,
  extends cleanly to a future `/projects` index, and matches how GitHub
  and Linear do per-entity routing. The `/p/all/...` aggregate slot is
  naturally reserved.

**Migration path.** Existing handlers at `/now`, `/work`, `/knowledge`,
`/people`, `/agents` redirect (302) to `/p/<default>/<page>` where
`<default>` is, in order of precedence:
1. The last-used project slug from `localStorage` (set by client JS on
   navigation).
2. The first project in the registry's stable sort order.

The default-project lookup runs server-side for the initial redirect (cookie-
free; reads a JS-set cookie if present, else falls back to alphabetical
first). Client JS keeps the cookie/localStorage in sync as the user switches.

**Page handlers refactor.** The five page handler blocks in
`internal/serve/server.go:308-370` currently capture `s.projectRoot` and
`s.heroDir` at registration time. They must:
1. Be mounted on the `/p/{slug}/<page>` route pattern.
2. Look up the project from the URL slug via `s.GetProject(slug)`.
3. 404 if the slug is unknown.
4. Pass the project's `ProjectRoot` and `HeroDir` to the page Deps struct
   per-request, not at handler construction time.

This is the largest refactor in this spec. Each `Deps` struct (in
`internal/serve/pages/*page/`) currently expects `ProjectRoot` and `HeroDir`
at construction. Two options:
- (A) Make Deps accept a `ProjectResolver` callback that returns the active
  project per-request.
- (B) Keep Deps construction per-request — build a fresh Deps in the handler
  wrapper after slug lookup.

Recommend (B) — simpler, no resolver indirection, matches how multi-tenant
HTTP handlers typically resolve context. The Deps structs are small; the
allocation cost is negligible.

**Top nav selector.** Add to `internal/serve/shell/templates/top-nav.html`:
- A dropdown component showing the active project name with a chevron.
- On open, lists all projects from `/api/projects` plus "All projects" as
  the first item.
- Selecting a project navigates to the same page under the new slug
  (`/p/<new-slug>/<current-page>`).
- Selecting "All projects" navigates to `/p/all/<current-page>`.
- Active project name persists in `localStorage` as `hero.activeProject` and
  in a cookie `hero_active_project` for server-side default resolution.

**All-projects aggregate mode (`/p/all/...`).** Per page:
- **Now** — merged activity feed across projects, each item tagged with
  project slug. The "what to work on next" section shows the top suggestion
  per project, capped at N projects.
- **Work** — combined spec list, filterable by project. Shows the same
  columns plus a project column.
- **Knowledge** — combined knowledge index, filterable by project. Source
  project shown on every card.
- **People** — *skipped*. People/ROI is per-project enough that an aggregate
  is misleading without thinking through team membership semantics. The
  page renders an empty state with a "pick a project" prompt.
- **Agents** — combined live session list and proposals across projects.
  Project slug shown on every row.

Page-level Deps structs need an optional `MultiProject []ProjectContext`
field for the aggregate mode. The aggregate code paths read from the slice;
single-project paths read from the legacy single-project fields. Phase 2
can stub aggregate handlers with "coming soon" content for the first
iteration and fill them in over follow-up turns — the spec does not require
all five aggregate views to land at once.

**Command bar search.**
- Client JS reads the active project slug from the URL path (`/p/<slug>/...`).
- Single-project view → calls `/api/<slug>/search?q=...`.
- All-projects view → calls a new `/api/search?q=...` endpoint that fans
  out across registered projects and merges results, each tagged with its
  project slug.

### Forward-compatibility notes

- The `/p/<slug>/...` namespace leaves `/projects` free for the future
  "Projects section" index page.
- A future top-level `/dashboard` is unblocked — it's neither a project
  page nor inside `/p/`.
- The `MultiProject` Deps field becomes the natural seam for cross-project
  Dashboard data later.

## Acceptance Criteria

### Phase 1 — Lifecycle

- WHEN `hero serve` starts successfully THE SYSTEM SHALL write a PID file at
  `~/.hero/serve.pid` (or `~/.hero/serve-<port>.pid` for non-default ports)
  containing pid, port, started_at, and version as JSON.
- WHEN `hero serve` exits gracefully THE SYSTEM SHALL remove its PID file.
- WHEN `hero serve stop` runs and no PID file exists THE SYSTEM SHALL print
  "no hero daemon is running" and exit 0.
- WHEN `hero serve stop` runs and a PID file exists for a live daemon THE
  SYSTEM SHALL send SIGTERM, wait up to 5 seconds, escalate to SIGKILL if
  needed, remove the PID file, and exit 0.
- IF `hero serve stop` finds a PID file whose process is already dead THEN
  THE SYSTEM SHALL remove the stale file and exit 0 with a "cleaned up
  stale PID file" message.
- WHEN `hero serve --force` runs and a daemon is already running THE SYSTEM
  SHALL stop the existing daemon before starting fresh.
- WHEN `hero serve status` runs and no daemon is running THE SYSTEM SHALL
  print "no hero daemon is running" and exit 1.
- WHEN `hero serve status` runs against a healthy daemon THE SYSTEM SHALL
  print pid, port, uptime, version, and the list of served projects.
- IF `hero serve status` finds a PID file whose process is dead THEN THE
  SYSTEM SHALL report the stale state and suggest `hero serve stop`.
- WHEN `hero serve` fails to bind because a hero daemon already holds the
  port THE SYSTEM SHALL print a message naming the running PID and pointing
  the user at `hero serve stop` or `--force`.
- IF `hero serve` fails to bind because a non-hero process holds the port
  THEN THE SYSTEM SHALL print a message identifying the port as foreign and
  suggest `--port` or freeing the port.
- THE SYSTEM SHALL expose `/api/status` returning daemon pid, port,
  uptime, version, project count, and project list as JSON.

### Phase 2 — Dashboard

- WHEN a user loads `/now`, `/work`, `/knowledge`, `/people`, or `/agents`
  THE SYSTEM SHALL redirect (302) to `/p/<default>/<page>` where
  `<default>` is the last-used project or the first registered project.
- WHEN a user loads `/p/<slug>/<page>` for a registered project THE SYSTEM
  SHALL render that page using the named project's `ProjectRoot` and
  `HeroDir`.
- IF `/p/<slug>/<page>` names an unknown project THEN THE SYSTEM SHALL
  return a 404 with a list of registered projects.
- THE SYSTEM SHALL render a project selector dropdown in the top nav on
  every dashboard page, populated from `/api/projects`.
- WHEN a user selects a project from the dropdown THE SYSTEM SHALL navigate
  to the same page under the new slug and persist the new active project to
  `localStorage` and a cookie.
- WHEN a user selects "All projects" THE SYSTEM SHALL navigate to
  `/p/all/<current-page>`.
- WHEN a user loads `/p/all/<page>` THE SYSTEM SHALL render an aggregate
  view across all registered projects for Now, Work, Knowledge, and Agents.
- WHERE the page is People AND the mode is all-projects THE SYSTEM SHALL
  render an empty state prompting the user to pick a project.
- WHEN the command bar issues a search from a project-scoped page THE
  SYSTEM SHALL scope the query to that project.
- WHEN the command bar issues a search from `/p/all/...` THE SYSTEM SHALL
  query across all projects and tag each result with its project slug.

## Changes

Phase 1 and Phase 2 can ship as separate PRs. Phase 1 is self-contained and
delivers value alone; Phase 2 depends on the `/api/status` endpoint from
Phase 1 only insofar as it shares the multi-project framing.

### Phase 1 — Daemon lifecycle

1. **Add PID file write/cleanup in `internal/serve/server.go`.**
   - New helper `pidFilePath(port int) string` returns
     `~/.hero/serve.pid` for the default port, `~/.hero/serve-<port>.pid`
     otherwise. Reuse `registryDir()` from `internal/serve/registry.go:30`.
   - After successful `net.Listen` (around line 387), write JSON
     `{pid, port, started_at, version}` to the path.
   - In the shutdown path (after `httpServer.Shutdown` returns), remove
     the file. Tolerate already-removed.

2. **Add `/api/status` HTTP endpoint in `internal/serve/api.go`.**
   - Register at the top of `Handler()` alongside `/api/projects` (line 51).
   - Returns running=true, pid (os.Getpid), port, started_at, uptime,
     version, project_count, and a `projects` array with `{slug, name, path}`
     entries. Read project list via `s.Projects()` (add a snapshot accessor
     if one doesn't exist).
   - Keep `/health` untouched.

3. **Add `hero serve stop` subcommand in `internal/cli/serve.go`.**
   - New cobra subcommand registered under `serveCmd`.
   - `--port` flag mirrors the start command.
   - Logic: read PID file → probe `/api/status` to confirm hero identity →
     SIGTERM with 5s wait → SIGKILL → cleanup. Idempotent on missing file
     or stale PID.

4. **Add `hero serve status` subcommand in `internal/cli/serve.go`.**
   - New cobra subcommand. `--port` flag.
   - `--json` flag passes through the HTTP response as-is for scripting.
   - Default output is multi-line human-readable.

5. **Add `--force` flag to `hero serve` start in `internal/cli/serve.go`.**
   - `serveCmd.Flags().BoolVar(&serveForce, "force", false, ...)`.
   - When set, run the stop logic before starting. Surface stop errors but
     proceed if stop reports "no daemon running."

6. **Improve bind-collision error in `internal/serve/server.go`.**
   - Detect `errors.Is(err, syscall.EADDRINUSE)` at the `net.Listen` failure
     site (line 388).
   - Probe `/api/status` on the port via `http.Get` with a 1s timeout.
   - Branch on hero vs. foreign response to print the appropriate guidance.

7. **Tests** in `internal/serve/server_test.go` and a new
   `internal/cli/serve_test.go`:
   - PID file written on start, removed on stop.
   - `/api/status` returns the expected shape.
   - Stop handles missing file, stale PID, and live daemon.
   - `--force` replaces a running daemon.
   - Bind-collision branches print the right message (mock the probe).

### Phase 2 — Dashboard project awareness

8. **Add `/p/{slug}/{page}` routing in `internal/serve/server.go`.**
   - Modify the shell router setup (around line 370-380) to register page
     routes under `/p/{slug}/...` instead of `/{page}`.
   - Add legacy redirect handlers for `/now`, `/work`, `/knowledge`,
     `/people`, `/agents` that 302 to `/p/<default>/<page>`.
   - Default project resolution: read `hero_active_project` cookie; fall
     back to first project in stable sort order.

9. **Refactor page handler construction to per-request project resolution.**
   - For each page (Now, Work, Knowledge, People, Agents) in
     `internal/serve/server.go:308-370`, change the handler to a thin
     wrapper that:
     - Parses the slug from the URL.
     - Looks up the project via `s.GetProject(slug)`; 404 on miss.
     - Builds the page's `Deps` struct with the project's `ProjectRoot`,
       `HeroDir`, etc.
     - Dispatches to the same handler that exists today.
   - Touches each `pages/*page/` package only if the Deps struct needs
     adjustment for `MultiProject` mode (see step 11).

10. **Add `/p/all/{page}` aggregate routing.**
    - Special-case `slug == "all"` in the wrapper from step 9.
    - For Now, Work, Knowledge, Agents: build a `MultiProject` slice of
      project contexts and pass to the page.
    - For People: render the "pick a project" empty state via the existing
      `empty-state-notice.html` partial.

11. **Add `MultiProject` field to page Deps structs.**
    - Touch `internal/serve/pages/nowpage`, `workpage`, `knowledgepage`,
      `agentspage`.
    - When set, page rendering reads from the slice; tags every output
      item with its project slug. First iteration may render a placeholder
      summary card per project, with full merging following in subsequent
      turns. The spec accepts a staged rollout for the aggregate views.

12. **Add project selector to `internal/serve/shell/templates/top-nav.html`.**
    - Dropdown component showing active project name.
    - Populated client-side from `/api/projects` on page load (cached in
      `localStorage`, refreshed every 30s).
    - "All projects" item at the top.
    - On selection: navigate to `/p/<slug>/<current-page>`; set
      `localStorage.heroActiveProject` and `hero_active_project` cookie.

13. **Add `/api/search?q=...` cross-project endpoint in
    `internal/serve/api.go`.**
    - New mux entry alongside `/api/projects`.
    - Fans out the search across all projects via `s.Projects()`, merges
      results with a `project_slug` field per hit.

14. **Update command bar JS in `internal/serve/shell/` to scope by URL.**
    - Parse `/p/<slug>/...` from `window.location.pathname`.
    - Single-project: hit `/api/<slug>/search`.
    - All-projects: hit `/api/search`.

15. **Tests:**
    - Integration test for `/p/<slug>/<page>` resolution and 404 on unknown
      slug.
    - Test for legacy redirect from `/now` to `/p/<default>/now`.
    - Test for `/p/all/now`, `/p/all/work`, `/p/all/knowledge`,
      `/p/all/agents` rendering without error.
    - Test for `/p/all/people` rendering the empty state.
    - Test for `/api/search` fan-out across two projects.

## Boundaries

- **Team-server / multi-human work** is out of scope. That's tracked in
  `hero-team-server` and involves shared queues, approval gates, and
  cross-human coordination. This spec is about a single-human, single-
  machine daemon serving multiple projects.
- **Per-project daemons / port-per-project** are not the direction. The
  global daemon model is intentional and this spec reinforces it.
- **Auth and multi-user permissions** on the dashboard are out of scope. The
  daemon binds to localhost; this work doesn't change that posture.
- **The broader "Projects section"** (per-project info, health, switching,
  utilities) is a separate forthcoming spec. This spec lays the URL routing
  and selector groundwork but does not build the section itself.
- **The high-level cross-project Dashboard view** (a top-level summary
  across projects) is a separate forthcoming spec. The `MultiProject` seam
  is the natural place for that to plug in later.
- **Refactoring page handlers beyond what slug-aware routing requires** is
  out of scope. Don't restructure `pages/*page/` packages, don't rename
  Deps fields, don't extract new abstractions. Match the existing style.
- **Renaming `/health`** is out of scope. Keep it as the lightweight
  liveness probe; `/api/status` is the rich, project-aware endpoint.

## Risks

- **Stale PID files.** A crashed daemon leaves the file behind. Mitigated
  by always probing the PID and the `/api/status` endpoint before trusting
  the file's contents. The `status` and `stop` commands both handle this.
- **Port held with no PID file** (older daemon predating this change, or
  PID file deleted manually). Mitigated by the improved bind-collision
  error: probe `/api/status` and infer hero vs. foreign.
- **Race on `--force`.** Two `--force` invocations could both try to stop
  and both try to listen. Net effect: at least one listener wins; the loser
  surfaces the improved bind error. Acceptable; no extra locking needed.
- **Default-project resolution flicker.** First page load without a cookie
  picks alphabetical first, then client JS reads `localStorage` and may
  navigate again. Mitigated by setting the cookie on every dropdown
  selection so subsequent loads are stable. One redirect is acceptable.
- **Aggregate views are expensive.** Loading "all projects" Now or Work
  could touch many indexes. Mitigate by capping the merged item count per
  project (e.g. top 10 per project, paginate the rest). Spec the cap during
  delivery rather than baking a number in here.
- **Page Deps refactor blast radius.** The five page handlers touch shared
  Deps structs in `pages/*page/`. The refactor is mechanical but wide.
  Land Phase 2 changes per page in separate commits so each can be reverted
  independently.
- **localStorage / cookie sync edge cases.** A user clearing one but not
  the other could see inconsistent default-project behavior. Mitigate by
  having client JS always rewrite the cookie from localStorage on page
  load.

## Delivery notes (stubbed / deferred for items 7 & 8)

These seams are in place; items 7/8 fill them in without further refactor.

- **`/p/all/<page>` aggregate renderers** — `internal/serve/routing.go:130-160`
  (`allProjectsHandler`) returns plain-HTML placeholders for now/work/
  knowledge/agents and a "Pick a project" empty state for people. Item 7/8
  replace the placeholder branches with calls into per-page renderers that
  read from a `MultiProject []ProjectContext` Deps field.
- **`MultiProject []ProjectContext` Deps field** — not added yet. Items 7/8
  add it to the page packages that need cross-project rendering. The
  `ProjectContext` shape exported from `internal/serve/server.go:36` is the
  intended seam.
- **Command bar search project-scoping** — current command bar JS still
  calls `/api/search?q=...` unprefixed. Spec calls this out in Phase 2;
  the routing+selector groundwork is in place but the JS rewrite to read
  `/p/<slug>/` from `window.location.pathname` and dispatch to
  `/api/<slug>/search` or `/api/search` is deferred (no impact on items
  7/8 — they don't depend on it).
- **Registry-mutating endpoints** — `/api/projects` already lists; add /
  remove / set-default are explicitly deferred per the autopilot brief
  ("only ship what items 7 and 8 actually need").

## Validation

- **Phase 1:**
  - Manual: start the daemon, confirm `~/.hero/serve.pid` exists with valid
    JSON. Run `hero serve status` from another directory; confirm output.
    Run `hero serve stop`; confirm clean exit and file removal.
  - Manual: start the daemon, `kill -9` it, run `hero serve status` and
    `hero serve stop`; confirm stale-file detection and cleanup.
  - Manual: start daemon, run `hero serve` again from another shell;
    confirm the improved bind error names the running PID.
  - Manual: start daemon, run `hero serve --force`; confirm the original
    exits and the new one comes up.
  - Manual: bind port 7437 with `nc -l 7437`, run `hero serve`; confirm
    the foreign-process branch of the bind error.
  - Automated: unit tests per the test list in step 7.

- **Phase 2:**
  - Manual: register two projects, start the daemon, open `/now`, confirm
    redirect to `/p/<first>/now`. Use the dropdown to switch; confirm URL
    updates and content reflects the new project.
  - Manual: navigate to `/p/all/now`, `/p/all/work`, `/p/all/knowledge`,
    `/p/all/agents`; confirm cross-project content with project tags.
  - Manual: navigate to `/p/all/people`; confirm empty state.
  - Manual: type in the command bar from a project page; confirm results
    are scoped. Repeat from `/p/all/now`; confirm cross-project results.
  - Manual: visit `/p/unknown-slug/now`; confirm 404 with the registered
    project list.
  - Automated: integration tests per step 15.

- **Cross-cutting:**
  - `go test ./internal/serve/... ./internal/cli/...` passes.
  - `hero check` reports no new convention violations.
  - The existing `/health` endpoint still works unchanged.
