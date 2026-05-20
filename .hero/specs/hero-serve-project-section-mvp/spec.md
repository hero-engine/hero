---
title: hero serve Project Section — Phase 1 MVP (Read-Only Per-Project Page)
slug: hero-serve-project-section-mvp
type: feature
status: completed
priority: P1
tags: [hero-serve, dashboard, ui, project, mvp]
created: 2026-05-19
relations:
  - target: hero-serve-project-section
    kind: parent
  - target: hero-serve-multi-project
    kind: depends-on
horizon: now
---

# `hero serve` Project Section — Phase 1 MVP (Read-Only Per-Project Page)

## Context

This is **Phase 1 of 5** for the
[`hero-serve-project-section`](../../initiatives/hero-serve-project-section/spec.md)
initiative. See the parent for the full product context, design rationale,
and boundaries.

Phase 1 lays the skeleton every later phase mounts onto: the
`projectpage` package, the per-project handler, the eight-section
template, the section data loaders, and the `/project` single-project
fallback for pre-multi-project mode. It ships **read-only** — no live
`hero check` runs, no peer probes, no lifecycle ops dispatch, no
destructive actions, no aggregate view. Sections that need cached data
(`Health`, `Peers`) render whatever is on disk with an "as of" timestamp
or an "as of: never" placeholder.

The point of Phase 1 is to land the seam. Phases 2–5 each plug into
this seam without re-architecting it.

## Kickoff

**Status:** delivered — `internal/serve/projectpage/` package shipped, eight read-only sections render, single-project `/project` fallback and multi-project `/p/<slug>/project` both wired, top-nav active-state pins Project on both routes.

**Shipped:**
- `internal/serve/projectpage/` package with `deps.go`, `handler.go`, `handler_test.go`, plus 8 section data loaders (identity / health / stack / registry / peers / trackers / knowledge / config) each with happy-path + missing-input unit tests.
- Templates under `internal/serve/projectpage/templates/` (one per section + composing `page.html`) — diverges from the spec's `shell/templates/project/` path to match the established `pages/now/templates/` convention, keeping `//go:embed` co-located with the package.
- `internal/serve/projectpage/static/project.js` — collapse/expand toggle with `localStorage` persistence keyed by `<slug>:<section>`. No SSE, no fetch.
- Rollup rename: `internal/serve/pages/project/` → `internal/serve/pages/rollup/`. Package, slug, label, template, breadcrumbs, and tests all moved. Top-nav order now: Now · Work · Knowledge · People · Agents · Project · Rollup.
- Server wiring in `internal/serve/server.go`: per-project `/p/{slug}/project` registration plus a single-project `/project` fallback. Both routes pass top-nav active-state tests.
- New `RegistryEntry` projection in the projectpage package and a `projectpageRegistryEntry` server adapter that surfaces the daemon-side `ProjectEntry` shape.

**Decisions called out during delivery:**
- **Stack detection** — no persistent producer yet. Implemented a lightweight marker-file probe (go.mod / package.json / Cargo.toml / …) that's cheap enough for per-request render. Richer detection is a follow-up Phase 2 (cached scan artifact) — flagged but not in scope.
- **Health artifact schema** — Phase 5 owns producing it; defined a tiny on-disk JSON shape (`captured_at` + `rows[name, status, message]`) at `.hero/cache/health.json` so the loader has something concrete to read. Phase 5 can override the path via the package-level `healthArtifactPath` var.
- **Template path** — `internal/serve/projectpage/templates/` instead of `internal/serve/shell/templates/project/` (the spec's suggestion). Matches the Now/Work convention exactly.

**Verified:** `go build ./...`, `go test ./...`, `go vet ./...` all clean. All new section loaders covered by tests. Handler integration test exercises every section against a fixture project with deliberately-missing inputs.

**Pick up at:** done — archive with `hero spec complete`. Phase 2 (`hero-serve-project-section-aggregate`) is the next phase.

## Goal

Loading `/p/<slug>/project` (or `/project` in single-project mode)
renders all eight read-only sections from whatever on-disk data exists,
without crashing on missing inputs and without running expensive
operations. The Project tab in the top nav is highlighted. The page is
the foundation Phases 2–5 build on.

## Acceptance Criteria

- WHEN a user loads `/p/<slug>/project` for a registered project THE
  SYSTEM SHALL render an Identity section showing project name, root
  path (clickable as a `file://` link), git remote, default branch,
  last-touched-at, spec/notes/decisions counts, and peer count.
- WHEN the page loads THE SYSTEM SHALL render a Stack and Conventions
  section listing the detected stack (from the `stack-detection`
  skill's persisted output), active conventions under
  `.hero/knowledge/conventions/`, and the three most-recently-added
  conventions, each linked to its spec.
- WHEN the page loads THE SYSTEM SHALL render a Registry Membership
  section reporting whether the project is present in
  `~/.hero/projects.json`, its slug, registration timestamp, and
  whether it is set as default. No remove affordance yet.
- WHEN the page loads THE SYSTEM SHALL render a Peers section listing
  registered sibling repos from `peer-manifest.yaml`, each with the
  most recent reachability value found on disk and last successful call
  timestamp. The page SHALL NOT perform a live probe; if no cached
  value exists the row SHALL show "as of: never".
- WHEN the page loads THE SYSTEM SHALL render a Trackers and
  Integrations section showing the configured tracker, last successful
  sync timestamp, queued import count, and any sync errors.
- IF no tracker is configured THEN THE SYSTEM SHALL display a single
  opt-in card linking to the integration setup docs.
- WHEN the page loads THE SYSTEM SHALL render a Knowledge Stats
  section showing counts of decisions, conventions, notes, captures
  and the last-captured-at timestamp, linking to the Knowledge page.
- WHEN the page loads THE SYSTEM SHALL render a read-only Config
  section displaying the merged contents of `hero.json` (and
  `hero.local.json` if present) with an "Open in editor" link
  resolving to a local `file://` path; the section SHALL be collapsed
  by default.
- WHEN the page loads THE SYSTEM SHALL render a Health section
  populated from the most recent on-disk `hero check` artifact if one
  exists, with a visible "as of" timestamp. IF no cached artifact
  exists THEN the section SHALL render "as of: never" with no error.
- WHEN every Health row passes THE SYSTEM SHALL collapse the Health
  section to a single green "All clear" row.
- WHEN the Project tab is the active page THE SYSTEM SHALL mark it
  active in the top nav.
- IF `hero-serve-multi-project` routing is not yet active THEN THE
  SYSTEM SHALL serve the same Project page at `/project` using
  `s.projectRoot` as a temporary fallback.
- THE SYSTEM SHALL build a fresh `projectpage.Deps` per request from
  the URL slug rather than reading `s.projectRoot` once at startup.
- WHEN a section's underlying data is unavailable (missing file,
  parse error) THE SYSTEM SHALL render the section with an
  unobtrusive empty-state row rather than 500ing the page.

## Approach

Mirror the layout of `internal/serve/pages/now/`: a `data/` directory
of section loaders, a `templates/` directory of partials, a top-level
`handler.go` that composes them. Loaders are pure functions over a
`Deps` struct so they unit-test cleanly. The shared template
(`internal/serve/shell/templates/project.html`) extends the existing
`page-layout.html` and includes the section partials in declared order.

Collapse state is persisted in `localStorage` keyed by
`<slug>:<section>`. The JS lives in
`internal/serve/shell/static/js/project.js` and is intentionally tiny:
only collapse-toggle wiring and a no-op stub for future section-refresh
calls (real refresh wiring lands in Phase 5).

## Changes

1. **Create `internal/serve/projectpage/` package**
   - `deps.go` — `Deps` struct with `ProjectRoot`, `HeroDir`, `Slug`,
     `RegistryEntry`. Health/Peer/OpsRunner fields are declared but
     nil-tolerant in Phase 1 (filled in later phases).
   - `handler.go` — `Handler` for `/p/{slug}/project`. Resolves slug
     to registry entry, builds `Deps`, calls each section loader,
     renders the template. 404 if the slug is unknown.
   - `data/identity.go`, `data/stack.go`, `data/registry.go`,
     `data/peers.go`, `data/trackers.go`, `data/knowledge.go`,
     `data/config.go`, `data/health.go` — one loader per section with
     a typed result struct and unit tests.

2. **Templates**
   - `internal/serve/shell/templates/project.html` — page composition
     extending `page-layout.html`, including section partials in
     fixed order: Identity, Health, Stack, Registry, Peers, Trackers,
     Knowledge, Config.
   - `internal/serve/shell/templates/project/` — one partial per
     section (`identity.html`, `health.html`, etc.).

3. **Register handlers in `internal/serve/server.go`**
   - In the shell-page handler block (currently lines 308-370), wire
     `/p/{slug}/project` to `projectpage.Handler`.
   - Add `/project` fallback handler for single-project mode using
     `s.projectRoot` to fabricate a `Deps`.
   - Extend top-nav active-state logic in `top-nav.html` to highlight
     Project for `/p/<slug>/project` and `/project`.

4. **Static assets**
   - `internal/serve/shell/static/js/project.js` — collapse/expand
     wiring with `localStorage` persistence keyed by
     `<slug>:<section>`. No SSE, no fetch yet.
   - `internal/serve/shell/static/css/project.css` (or inline in the
     template, matching the existing convention used by Now/Work) —
     section header + collapsed/expanded styles.

5. **Tests**
   - Unit tests for each data loader: happy path + missing-input
     branch returning empty-state result.
   - Handler test for `/p/{slug}/project` rendering all sections
     against a fixture project root.
   - Handler test for `/project` fallback in single-project mode.
   - 404 test for unknown slug.

6. **Knowledge stub**
   - Leave `internal/serve/healthcache/` and `internal/serve/opsrunner/`
     unbuilt — those land in Phase 5 and Phase 3. Phase 1 reads
     whatever is on disk and shows "as of: never" placeholders where
     there is nothing yet.

## Boundaries

- **No live `hero check` invocation**. Phase 5 owns that.
- **No peer probes**. Phase 5 owns that.
- **No ops dispatch buttons wired to anything**. Phase 3 owns the
  runner; Phase 1 may render the Operations section heading with
  disabled buttons or omit the section entirely (omit is preferred —
  no UI affordance until it does something).
- **No aggregate `/p/all/project` view**. Phase 2 owns that.
- **No registry removal or Danger Zone**. Phase 4 owns those.
- **No `hero.json` editing**. Read-only with "Open in editor" link is
  the permanent boundary, not a phase deferral.

## Risks

- **Routing dependency**. If `hero-serve-multi-project` hasn't landed
  the per-page `Deps` refactor when Phase 1 ships, the `/p/<slug>/`
  prefix won't route. Mitigation: ship the `/project` fallback first;
  it works on `s.projectRoot` and is independent of multi-project
  routing.
- **Section loader fragility**. If a missing `peer-manifest.yaml` or a
  malformed `hero.json` panics a loader, the whole page 500s.
  Mitigation: every loader returns an empty-state result on missing
  input; explicit tests for each branch.
- **Top-nav active-state regression**. The active-state branch in
  `top-nav.html` is shared with Now/Work/Knowledge; a wrong branch
  flips active to the wrong tab on every page. Mitigation: handler
  test for active-state CSS class on every project URL plus a smoke
  check against other home URLs.

## Validation

- All unit tests for `internal/serve/projectpage/data/` pass.
- Handler test renders all eight sections without panic on a fixture
  project with deliberately-missing inputs (no peer manifest, no
  tracker config, no cached health artifact).
- Manual: open `/p/<slug>/project` on a real project; every section
  renders; collapse/expand persists across reload.
- Manual: in single-project mode, `/project` serves the same content.
- Manual: Project tab is highlighted for `/p/<slug>/project`,
  `/project`, and (in a later phase) `/p/all/project`; Now/Work tabs
  are NOT highlighted on any project URL.

## Changes (files touched on completion)

Rollup rename (chunk a):
- `internal/serve/pages/rollup/` (renamed from `pages/project/`, `git mv`) — package, route, slug, label, breadcrumbs, template name all updated to "rollup"
- `internal/serve/server.go` — import + register call switched to `rolluppage`; new `/rollup` slot

Project section page (chunk b):
- `internal/serve/projectpage/deps.go` (new) — `Deps` + nil-tolerant `RegistryEntry`
- `internal/serve/projectpage/handler.go` (new) — `Handler`, page composition, inline collapse JS
- `internal/serve/projectpage/data/identity.go` + `_test.go` (new)
- `internal/serve/projectpage/data/health.go` + `_test.go` (new) — reads cached artifact only
- `internal/serve/projectpage/data/stack.go` + `_test.go` (new) — lightweight marker-file probe
- `internal/serve/projectpage/data/registry.go` + `_test.go` (new)
- `internal/serve/projectpage/data/peers.go` + `_test.go` (new) — reads `cfg.Repos`, never probes
- `internal/serve/projectpage/data/trackers.go` + `_test.go` (new)
- `internal/serve/projectpage/data/knowledge.go` + `_test.go` (new)
- `internal/serve/projectpage/data/config.go` + `_test.go` (new) — read-only `hero.json` dump
- `internal/serve/projectpage/templates/page.html` + 8 section partials (new)
- `internal/serve/projectpage/static/project.js` (new, reference copy; live JS is inlined by handler)
- `internal/serve/projectpage/handler_test.go` (new) — renders-all-sections, single-project fallback, top-nav active-state, 404

Wiring (chunk c):
- `internal/serve/server.go` — register `projectpage` at `/project` (per-project shell router) before `rolluppage`
- `internal/serve/routing.go` — `legacyPagePaths` now includes both `/project` and `/rollup`

Validation:
- `go build ./...` clean
- `go test ./...` clean (every package passes)
- `go vet ./...` clean

## Deferred to follow-up phases (per spec boundaries)

- Live `hero check` runs (Phase 5)
- Live peer probes / `peer-manifest.yaml` reachability (Phase 5)
- Operations section / ops dispatch (Phase 3) — section omitted entirely in Phase 1, preferred per spec
- Registry removal + Danger Zone (Phase 4)
- Aggregate `/p/all/project` view (Phase 2)
- SSE section-refresh + extracted static JS asset (Phase 5)
