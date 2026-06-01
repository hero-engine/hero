---
title: hero serve Project Section — Per-Project Info, Utilities, and Operations Page
slug: hero-serve-project-section
type: initiative
status: completed
tags: [hero-serve, dashboard, ui, project, operations, daemon, registry, peers]
created: 2026-05-19
relations:
  - target: hero-serve-dashboard-redesign
    kind: follows
  - target: hero-serve-multi-project
    kind: depends-on
  - target: hero-team-server
    kind: related
horizon: now
completed_at: 2026-05-20T05:29:22Z
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
- `.hero/specs/hero-serve-dashboard-redesign/spec.md` (now archived)
  reworked **Now** (activity feed) and **Work** (spec catalog) into the
  primary surfaces. Project must NOT duplicate either — Now is "what
  happened lately," Work is "the spec catalog," Project is "this project
  itself."

Hero's mission is to make the next session start smarter than the last one
ended. The Project page is exactly the "stuff nobody told the agent"
surface: identity, health, registry membership, peers, lifecycle ops.
Today all of that lives in the CLI (`hero check`, `hero peer list`,
`hero status`, `hero snapshot`, `hero index`, `hero scan`) or in raw
JSON under `~/.hero/projects.json`. There is no UI home.

The original single-spec design for this surface was judged too large
for one delivery cycle. This initiative carries the shared Context,
Goal, Approach, Boundaries, and Risks, and is decomposed into five
phased child specs delivered in strict sequence.

No active tripwires conflict with this design.

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

## Approach

### Page architecture

A new `projectpage` package under `internal/serve/projectpage/` mirrors
the shape of `nowpage` and `workpage`. It exposes:

- `Deps` struct carrying `ProjectRoot`, `HeroDir`, `Slug`, a
  `RegistryEntry` snapshot, and shared services (health cache, peer
  cache, op-runner) — built per request by the handler from the URL
  slug, consistent with the per-page `Deps` refactor in
  `hero-serve-multi-project`.
- `Handler` for `/p/{slug}/project` rendering a single template,
  `internal/serve/shell/templates/project.html`.
- `AggregateHandler` for `/p/all/project` rendering
  `internal/serve/shell/templates/project_all.html`, fanning out across
  the registry using the same pattern Now and Work use in
  `hero-serve-dashboard-redesign`.

### Section model

Each section is a small server-rendered partial. The page arrives with
everything pre-rendered for first-paint speed; sections that have a
"refresh" affordance re-fetch their own HTML via a small JS helper.
No new framework. Sections are collapse-by-default when typically clean
(Danger Zone, Config, Knowledge Stats) and expand-by-default for sections
that demand operator attention (Identity, Health, Peers when degraded).
Collapse state persists in `localStorage` keyed by project slug.

### Health and peer caches

`hero check` is too expensive to run on every page load, and live-probing
every peer turns page load into an N-peer round-trip. Both states are
served from a shared in-process `HealthCache`
(`internal/serve/healthcache/`) keyed by `slug` and `slug+peer-alias`
with a configurable TTL (default 5 minutes; `serve.health_ttl` in
`hero.json`). The page renders cached values with an "as of" timestamp
and explicit refresh affordances.

### Operation runner

Lifecycle ops dispatch through a new `opsrunner` package
(`internal/serve/opsrunner/`) with an allowlisted verb→`hero` CLI
command map, an in-memory job registry keyed by `slug+verb` for dedup,
and SSE progress streaming. Verb allowlist:
`re-scan`, `re-index`, `run-check`, `refresh-queue`, `capture-knowledge`,
`snapshot`, `export`. The runner is forward-compatible with the future
`hero-team-server` job queue.

### Forward-compat seams

- All endpoints accept project context via URL slug → `Deps`, never via
  `s.projectRoot`. This matches `hero-serve-multi-project`.
- The cross-project peers map and daemon-ops section are the natural
  extension points where `hero-team-server` will plug shared-job-queue
  and team-presence data in.

## Boundaries

- **In-browser `hero.json` editing** is out of scope. Config is
  read-only with an "Open in editor" link.
- **Authentication and multi-user permissions** are out of scope.
  Destructive operations assume the single-operator local case and will
  be re-evaluated in `hero-team-server`.
- **Project creation wizard** is out of scope. This page assumes the
  project is already registered in `~/.hero/projects.json`.
- **Renaming a project / changing slug** is out of scope. Graph-wide
  identity story (notes, decisions, NEXT.md cross-refs) is unsettled.
- **Migrating specs between projects** is out of scope.
- **Cross-project search** is out of scope — covered by
  `hero-serve-multi-project` and the dashboard redesign command bar.
- **Activity feed on the Project page** is out of scope — Now owns
  that. Project page cross-links to Now and Work but never duplicates.
- **A heavy frontend framework** is out of scope. Server-rendered
  partials + small JS module, matching existing dashboard conventions.
- **Persistent health cache across daemon restarts** is out of scope;
  in-process cache repopulates on first request.
- **A real graph visualization library** for the peers map is out of
  scope. A clean table is acceptable for v1.

## Risks

- **Routing dependency**. Depends on `hero-serve-multi-project` for
  `/p/<slug>/<page>` routing and per-page `Deps`. Until that lands, only
  the single-project `/project` fallback is available. Mitigation: ship
  fallback in Phase 1; gate `/p/all/project` aggregate behind multi-
  project landing in Phase 2.
- **Health computation cost**. Running `hero check` on every load is
  unusable. Mitigation: `HealthCache` with default 5m TTL and explicit
  "Refresh now" affordance — landed in Phase 5; Phase 1 ships static
  cached-output rendering only.
- **Peer reachability noise**. Live-probing every peer per load is slow.
  Mitigation: cached reachability with manual refresh — Phase 5.
- **Daemon stop kills the dashboard**. Surfacing "Stop daemon" means the
  current dashboard session dies on click. Mitigation: clear confirmation
  copy + restart instructions — Phase 4.
- **Registry edits are destructive**. Mitigation: 5-second grace window
  with undo toast — Phase 4.
- **Missing project root path**. If a user deletes a project directory,
  the page must not crash. Mitigation: `stat` the root on load and render
  a deregister banner — Phase 4.
- **SSE behind reverse proxies**. Op progress streams may lag.
  Mitigation: document `serve` is local-only for now.
- **Endpoint name collision with `hero-serve-multi-project`**. If both
  ship `/api/status` vs `/api/daemon/status`, both should point at the
  same handler. Mitigation: coordinate during Phase 2 delivery.
- **Information density overwhelm**. Ten sections is a lot.
  Mitigation: collapse-by-default for usually-clean sections; persist
  per-operator collapse state in `localStorage`.

## Open questions

- **Daemon-status endpoint name**. Settle on `/api/status` vs
  `/api/daemon/status` during joint delivery with
  `hero-serve-multi-project`. Either is fine; pick one and alias.
- **Stop-daemon button placement**. Current design says aggregate-only
  (Phase 4), since stopping the daemon is not a per-project concern.
- **Knowledge stats time series**. The spec calls for "over time"
  counts; v1 ships totals only, with sparkline / time-series deferred
  to a follow-up.

## Child Specs

All five phases shipped. Initiative complete.

| Order | Spec | Status |
|---|---|---|
| Phase 1 | [hero-serve-project-section-mvp](../../specs/hero-serve-project-section-mvp/spec.md) | ✅ delivered |
| Phase 2 | [hero-serve-project-section-aggregate](../../specs/hero-serve-project-section-aggregate/spec.md) | ✅ delivered |
| Phase 3 | [hero-serve-project-section-opsrunner](../../specs/hero-serve-project-section-opsrunner/spec.md) | ✅ delivered |
| Phase 4 | [hero-serve-project-section-destructive](../../specs/hero-serve-project-section-destructive/spec.md) | ✅ delivered |
| Phase 5 | [hero-serve-project-section-healthcache](../../specs/hero-serve-project-section-healthcache/spec.md) | ✅ delivered |

### Sequencing rationale

- **Phase 1 first** because the package + template + section partials
  are load-bearing for every later phase; everything else mounts onto
  this skeleton.
- **Phase 2 before 3** because the aggregate view exercises the
  multi-project routing seam without requiring the ops runner, surfacing
  routing bugs early.
- **Phase 3 before 4** because the destructive Stop-daemon action in
  Phase 4 dispatches through the ops runner.
- **Phase 4 before 5** because destructive flows are higher product
  risk than caching plumbing; we'd rather ship the safety affordances
  ahead of the perf optimization.
- **Phase 5 last** because the health/peer caches are an optimization
  swap-in: Phase 1 renders whatever cached output already exists on
  disk; Phase 5 makes those caches live, refreshable, and probe-driven.
