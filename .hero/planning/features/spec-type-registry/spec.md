---
title: Spec Type Registry — Domain-Declared Spec Types and Lifecycles
type: feature
status: planning
priority: P0
tags: [platform, domains, spec-types, registry, refactor]
created: 2026-05-15
relations:
  - target: hero-domains
    kind: parent
depends-on:
  - domain-plugin-architecture
horizon: next
smoke: deferred
---

> **Status: awaiting `domain-plugin-architecture`.** This stub is a
> `/design`-ready brief, not a complete design. The work cannot land
> until the domain pack layout exists. `/design spec-type-registry` can
> resolve the open questions below in parallel with primitive #1
> delivery so the registry design is ready the moment the pack layout
> lands.

## Kickoff

Make spec types pluggable per domain. Today `feature`, `bug`, `convention`, `decision` are hardcoded across the parser, lint, status filters, importers, dashboard, and command routing. Each domain pack should declare its own spec types, lifecycle states, frontmatter schema, and which commands accept them. Blocking for `hero-pm` (PRD, story, epic, roadmap-item don't exist yet).

**Status:** planning — stub written 2026-05-15. Blocked on `domain-plugin-architecture`.

**Pick up at:** Run `/design spec-type-registry`. First job is the audit — grep `internal/spec/`, `internal/lint/`, importers, and dashboard for hardcoded type literals to size the surface area honestly. Then resolve open question #1: registry declared in Go code, manifest file, or both.

→ `/design spec-type-registry`

**Files:** .hero/planning/features/spec-type-registry/spec.md, .hero/planning/initiatives/hero-domains/spec.md, internal/spec/, internal/lint/
**Skip:** Adding PM spec types here — those live in `hero-pm`. This spec only ships the registry mechanism plus engineering's reference declaration.

## Goal

Replace hardcoded spec types with a per-domain registry. Each domain
pack declares the types it accepts, their lifecycles, frontmatter
schema, and which commands route to them. Engineering's existing four
types (`feature`, `bug`, `convention`, `decision`) become the reference
registration shipped inside `domains/engineering/`. The parser, lint,
status filters, importers, dashboard type filters, and command router
all read from the registry rather than enumerated literals.

## Why now

Blocking for `hero-pm`. PM needs `prd`, `story`, `epic`, and
`roadmap-item` as first-class spec types — they cannot be modeled as
engineering specs with different tags. Until the registry exists, the
PM pack has no clean way to introduce new types without forking core.

The parent initiative flags this as the deepest cross-cutting risk:
spec-type hardcoding is referenced from the parser, lint, status
filters, importers, and the dashboard. Budget for a full audit during
design — don't discover the surface area mid-delivery.

## Scope outline

The design pass should cover:

1. **Registry shape** — how a domain declares its types. Whether the
   declaration is Go code (compile-time), a manifest file (data-driven),
   or both (manifest validated against Go interfaces).
2. **Type record** — name, label, lifecycle states, allowed status
   transitions, frontmatter schema (required + optional fields),
   accepting commands, default folder under `.hero/planning/`.
3. **Parser refactor** — `internal/spec/` reads from the registry to
   validate type, status, and required frontmatter. No more `switch`
   on hardcoded type strings.
4. **Lint refactor** — `hero spec lint` uses the registry's
   frontmatter schema for required-field checks and the lifecycle
   states for status validation.
5. **Status filters** — every CLI command that filters by status
   (`hero list`, `hero active`, `hero blocked`, dashboard) accepts
   types from the active pack's registry.
6. **Importers** — tracker importers (Jira, GitHub, Linear) take a
   target type from the registry rather than hardcoding `feature` or
   `bug`. PM's epic-import path needs to land an `epic` spec, not a
   `feature`.
7. **Command routing** — `/design`, `/diagnose`, `/deliver` (and PM's
   `/refine`, `/triage`) check the registry for which types they
   accept. `/diagnose` on a `story` should fail clearly, not silently
   write a bug-shaped spec.
8. **Dashboard type filters** — type pills, kanban swimlanes, and the
   type filter in spec lists all enumerate the active pack's types.
9. **Engineering reference registration** — `domains/engineering/`
   ships a registry declaration that reproduces today's exact behavior.

## Touchpoints (sketch — confirm during audit)

- `internal/spec/` — parser, type/status validation
- `internal/lint/` — required-field and status-transition rules
- `internal/cli/` — list, active, blocked, queue, status filters
- `internal/serve/` — dashboard type filters, kanban, swimlanes
- Tracker importer modules — Jira, GitHub, Linear
- `commands/` and `domains/*/commands/` — command-accepts-type metadata

The audit is the first deliverable of `/design`. Treat the list above
as a starting point, not a contract.

## Unknowns for design pass

1. **Registry mechanism — Go code, manifest, or both?** Go code is
   type-safe and refactor-friendly; manifests (JSON/YAML) are
   declarative and editable without rebuilding. Both is plausible:
   manifests as the source of truth, Go interfaces for type-safe
   accessors. Pick during design.
2. **Frontmatter schema language.** Plain Go struct tags, JSON Schema,
   a Hero-specific mini-DSL? The schema needs to drive lint, importer
   defaults, and possibly form fields in the dashboard.
3. **Migration story.** Existing specs use the four engineering types
   today. Re-registering those in `domains/engineering/` should be a
   no-op — but how do we prove that? A pre/post lint diff is the
   obvious test, name it explicitly.
4. **Cross-domain type references.** `hero-pm` will need to express
   "this story handed off to that engineering feature." Does the
   registry need to know about cross-domain edges, or is that the
   knowledge graph's problem? Probably the graph's — but flag it
   here so design doesn't get surprised.
5. **Lifecycle transitions per type.** Engineering specs go
   `planning → in-review → delivering → completed`. PM types have
   different lifecycles (e.g. `roadmap-item`: candidate → committed
   → shipped → dropped). Should transitions be enforced (the
   registry rejects invalid jumps) or just declared (the dashboard
   uses them for ordering)? Pick during design.

## Boundaries

- **Not** declaring PM spec types here — those land in `hero-pm`.
- **Not** changing the on-disk spec format. Frontmatter shape stays;
  what changes is who validates it.
- **Not** introducing per-spec custom types declared inline. Types
  are a domain-pack concern, not a per-project concern.
- **Not** addressing graph node namespacing — that's
  `domain-scoped-knowledge-graph` (item #6).

## Risks

- **Audit underestimates surface area.** Parent initiative explicitly
  calls this out. If the audit misses a touchpoint, the registry
  silently disagrees with that code path and bugs land in PM
  delivery instead of here.
- **Engineering reference registration changes behavior.** Re-encoding
  today's four types via the registry must be bit-identical to current
  behavior. Any drift is a regression. Diff lint output before/after.
- **Importers are the messiest path.** Tracker importers have
  type-mapping logic that pre-dates this work; expect surprises.
