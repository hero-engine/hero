---
title: Dashboard Project URL Scheme — /p/<slug>/<page>
type: decision
status: proposed
created: 2026-05-19
tags: [dashboard, serve, url-routing, multi-project, decision]
relations:
  - target: hero-serve-multi-project
    kind: decided-in
---

# Dashboard Project URL Scheme — `/p/<slug>/<page>`

## Decision

The hero dashboard scopes per-project pages under the `/p/<slug>/...`
namespace. The aggregate "across all projects" view uses the reserved
slug `all`: `/p/all/<page>`. Legacy unprefixed routes (`/now`, `/work`,
`/knowledge`, `/people`, `/agents`) 302-redirect to
`/p/<default>/<page>` where `<default>` is the last-used project (from
`hero_active_project` cookie / `localStorage.heroActiveProject`) or the
first registered project in stable sort order.

## Context

`hero serve` is a global daemon serving every project in
`~/.hero/projects.json`, but historically its UI hardcoded the launching
project. Adding multi-project awareness to the dashboard forced a URL
scheme choice.

## Options considered

- **`?project=<slug>` query param.** Bookmark-fragile, makes the active
  project feel transient, ugly with deep links.
- **`/<slug>/<page>` (no prefix).** Collides with future top-level
  routes; risks slug/page name collisions (a project literally named
  `now`).
- **`/p/<slug>/<page>`.** Clear namespace; leaves `/projects`,
  `/dashboard`, `/settings`, etc. free for future top-level surfaces;
  matches the per-entity routing pattern used by GitHub and Linear.

## Why this matters going forward

- A future "Projects section" index can live at `/projects` without
  conflict.
- A future cross-project "Dashboard" can live at `/dashboard` without
  conflict.
- The `all` slug is a stable seam for aggregate views; page Deps structs
  carry an optional `MultiProject []ProjectContext` field when rendering
  under `/p/all/...`.

## Consequences

- Page handlers must resolve the project per-request from the URL slug,
  not at handler construction time.
- All legacy bookmarks need a redirect handler. Acceptable: one redirect
  on first load, then stable.
- The "All projects" People page is intentionally a "pick a project"
  empty state — aggregating people/ROI across projects would need team-
  membership semantics we don't have yet.
