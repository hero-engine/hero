---
title: Dashboard View Registry — Pluggable Dashboard Pages per Domain
type: feature
status: completed
priority: P0
tags: [platform, domains, dashboard, ui, registry, refactor, replaced]
created: 2026-05-15
relations:
  - target: hero-domains
    kind: parent
  - target: hero-surface-shell
    kind: superseded-by
  - target: hero-surface-architecture
    kind: superseded-by
depends-on:
  - domain-plugin-architecture
horizon: next
smoke: deferred
---

> **Absorbed by [hero-surface-shell](../hero-surface-shell/spec.md) under the [Hero Surface Architecture](../../initiatives/hero-surface-architecture/spec.md) initiative.** The view registry primitive is retained as the shell's pluggability mechanism; the cross-domain router risk flagged here is treated as a first-class concern in the shell spec. This spec is preserved for history.


> **Status: awaiting `domain-plugin-architecture`.** This stub is a
> `/design`-ready brief, not a complete design. The work cannot land
> until the domain pack layout exists. `/design dashboard-view-registry`
> can resolve the open questions below in parallel with primitive #1
> delivery.

## Kickoff

Make dashboard pages pluggable per domain. Today every page (spec kanban, drift, velocity, CI status) is fixed in the dashboard code. Move to a config-driven page registry where each domain pack ships its own `views/` manifest. Engineering's existing pages become the reference registration under `domains/engineering/views/`. PM ships Roadmap, Story queue, Intake funnel, Handoff stream.

**Status:** planning — stub written 2026-05-15. Blocked on `domain-plugin-architecture`.

**Pick up at:** Run `/design dashboard-view-registry`. First decision: server-rendered registration (Go code in the pack) vs client-side registration (manifest + a known view-component catalog). Then design the domain router — cross-domain navigation is first-class, not bolted on.

→ `/design dashboard-view-registry`

**Files:** .hero/planning/features/dashboard-view-registry/spec.md, .hero/planning/initiatives/hero-domains/spec.md, internal/serve/
**Skip:** A generic "drop your React app here" plugin system. Views register declaratively with a fixed component catalog; full custom HTML is out of v1 scope.

## Goal

Replace the dashboard's hardcoded page list with a per-domain view
registry. Each domain pack declares its views via a `views/` manifest
that registers pages with a route, title, default-landing flag, and the
data sources or widgets it consumes. Engineering's existing pages (spec
kanban, drift report, CI status, velocity, etc.) become the reference
registration under `domains/engineering/views/`. PM will register
Roadmap (default landing), Story queue, Intake funnel, and Handoff
stream from its own pack.

Shared chrome — top nav, global search, sessions list, settings — stays
in the dashboard and is reused across domains.

## Why now

PM's dashboard is fundamentally different in shape: Roadmap is the
default landing page, not a spec kanban. Without a view registry, the
PM pack would need to fork the dashboard or ship its pages as a
separate UI. Both are bad outcomes.

The parent initiative flags a cross-cutting risk worth budgeting for in
design: **the dashboard needs a domain *router*, not just a page
registry.** Cross-domain navigation — e.g. "I'm in PM, show me the
engineering features delivering this story" — is first-class. Building
only a page registry without thinking about cross-domain links produces
a UI that visibly fractures at the seams.

## Scope outline

The design pass should cover:

1. **View manifest schema.** What a `domains/<name>/views/` directory
   looks like. Per-view: id, title, route, default-landing flag,
   widgets or data sources consumed, optional access flags.
2. **View registration mechanism.** Whether registration is
   declarative (manifest only) or programmatic (Go code registering
   routes/handlers). Likely both: manifest is the source of truth,
   handlers are looked up in a known catalog.
3. **Domain router.** When the dashboard loads, it picks the active
   pack's default landing. Switching domains is a first-class
   navigation action, not a config edit-and-restart.
4. **Cross-domain links.** Going from a PM story page to the
   engineering feature it handed off to should be a hyperlink, not a
   manual context switch. The router preserves URL semantics across
   domain boundaries.
5. **Shared widget reuse.** Spec search, spec list, kanban — these are
   widgets multiple domains want. Catalog them as shared components
   that any pack's view can include.
6. **Engineering reference registration.** Each existing page —
   spec kanban, drift report, CI status, velocity, blocked queue, etc.
   — gets a manifest entry under `domains/engineering/views/`. Behavior
   bit-identical to today.

## Touchpoints (sketch — confirm during design)

- `internal/serve/` — current dashboard server, page routes, handlers
- `domains/engineering/views/` — new directory, reference manifest
- Shared widgets — likely lives under `internal/serve/widgets/` or
  similar after refactor
- `hero.json` — domain selection (already covered by primitive #1)

## Unknowns for design pass

1. **Server-rendered vs client-side view registration.** Today the
   dashboard is server-rendered (HTMX-style? full SSR?). A client-side
   view registry buys flexibility but doubles the cognitive surface.
   Pin the rendering model first; the registry shape follows.
2. **Shared widget reuse contract.** When a PM view embeds the spec
   search widget, what does it pass and what does the widget return?
   Need a widget interface — probably a small typed contract per
   widget kind.
3. **Cross-domain link semantics.** A handoff from PM story to
   engineering feature: is the link a stable graph edge that the
   router knows how to follow, or a URL with a domain-qualified
   path? Probably the latter, but pin it down.
4. **Default landing per domain — config or convention?** Each pack
   declares its default landing in the views manifest. Whether the
   user can override per-project (in `hero.json`) is a question for
   design.
5. **Auth/access flags.** Some views may be role-gated later
   (per `cloud-admin`). The manifest should leave room for an
   `access:` field without committing to roles yet.

## Boundaries

- **Not** building a generic "third-party HTML plugin" surface. Views
  are declarative manifests + components from a known catalog.
- **Not** designing PM's views in detail — those land in `hero-pm`.
- **Not** addressing multi-active-domain coexistence in the
  dashboard — single-active-domain v1, multi-active deferred to
  primitive #6.
- **Not** rebuilding the dashboard. Refactor in place.

## Risks

- **Page-registry-only is a trap.** If we skip the domain router, the
  cross-domain UX fractures: every navigation between PM and
  engineering looks like a hard context switch. Parent initiative
  flagged this explicitly.
- **Shared widget contract drifts.** Without a tight widget interface,
  PM views will fork their own variants of spec list / search and the
  reuse promise dies. Spend design time on the widget contract even
  though it feels secondary.
- **Engineering-reference parity is expensive.** Each existing page
  becomes a manifest entry plus a reviewed migration; drift from
  current behavior is a regression. Diff before/after.
