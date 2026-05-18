---
title: Domain-Scoped Knowledge Graph — Namespace Tags on Graph Nodes
slug: domain-scoped-knowledge-graph
type: feature
status: planning
priority: P0
tags: [platform, domains, graph, knowledge, namespacing]
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
> until the domain pack layout exists. P1 priority — `hero-pm` can ship
> in single-domain mode without this — but the parent initiative requires
> namespace tags from day one so multi-domain coexistence can land later
> without re-indexing.

## Kickoff

Add domain namespace tags to every knowledge graph node and edge. Today the graph is flat: a `Spec`, a `Component`, a `Decision` lives in one undifferentiated namespace. When PM and engineering coexist (later), PM stories and engineering features collide unless the graph knows which domain a node belongs to. Tag every node now; teach queries (`hero why`, `hero blocked`, `hero search`) to filter by active domain or render the boundary.

**Status:** planning — stub written 2026-05-15. Blocked on `domain-plugin-architecture`.

**Pick up at:** Run `/design domain-scoped-knowledge-graph`. First decision: edge semantics for cross-domain handoff (`story → feature` from `/design`) — does this need a new edge kind, or is it a regular edge that crosses a namespace boundary? Then audit every query path that touches the graph and decide its v1 stance on filtering.

→ `/design domain-scoped-knowledge-graph`

**Files:** .hero/planning/features/domain-scoped-knowledge-graph/spec.md, .hero/planning/initiatives/hero-domains/spec.md, internal/index/, internal/graph/
**Skip:** Full multi-active-domain runtime in v1 — single-active-domain is enough. Namespace tags now so v2 can flip the switch without re-indexing.

## Goal

Add a `domain` namespace tag to every node and edge in the knowledge
graph. Teach every query path that consumes the graph (`hero why`,
`hero blocked`, `hero search`, dashboard widgets, MCP tools) to filter
by the active domain or render cross-domain boundaries explicitly.

V1 ships with single-active-domain workspaces — namespace tags are
present, all queries default to the active domain — but the foundation
is in place so multi-active-domain workspaces can land later without a
graph re-index.

## Why now

The parent initiative makes a deliberate call: PM can ship in
single-domain mode without this, but adding namespace tags later is
painful. Even in single-domain v1, every graph query path must tolerate
a namespace tag — silently mixing domains in shared queries is worse
than blocking on the work upfront.

The cross-cutting risk in the parent initiative: queries (`hero why`,
`hero blocked`, `hero search`) currently operate on a flat namespace.
PM stories pointing at engineering features cross a domain boundary;
queries need to either filter by active domain (default) or render the
boundary explicitly (cross-domain traversal opt-in).

P1 priority because PM ships without coexistence; P0 timing because
without it, the multi-domain story is one re-index away from rotting.

## Scope outline

The design pass should cover:

1. **Graph schema change.** Every node and edge gets a `domain`
   field. Default is the active domain at write time. Persisted in
   whatever store the graph uses today.
2. **Backfill / migration.** Existing nodes (all engineering) get
   `domain: engineering` on first read. No-op for users today.
3. **Query layer.** Every query that touches the graph accepts an
   optional domain filter. Default behavior: filter to the active
   domain. Opt-in: cross-domain queries.
4. **Cross-domain edges.** Edges that span domains (`story →
   feature` handoff) are first-class. Likely a new edge kind, or a
   regular edge with both endpoints' domains exposed in the result.
5. **Query path audit.** Enumerate every consumer of the graph —
   `hero why`, `hero blocked`, `hero search`, `hero relevant`,
   dashboard widgets, MCP tools, drift detection. Each gets a
   per-query stance: filter, traverse, or render-boundary.
6. **Cross-domain rendering.** When a query crosses a domain
   boundary (e.g. `hero why` traces a story to a feature), the
   output must make the crossing visible — not silently hop.
7. **Search ranking.** `hero search` should rank in-domain results
   above cross-domain by default. Opt-in `--all-domains` flag.

## Touchpoints (sketch — confirm during design)

- `internal/index/` — graph storage, node/edge schema
- `internal/graph/` (if it exists) — query layer
- `hero why`, `hero blocked`, `hero search`, `hero relevant`
- MCP tools that touch the graph (`hero_search`, `hero_why`,
  `hero_blocked`, `hero_code`, `hero_drift`)
- Dashboard widgets reading the graph (spec list, kanban, drift,
  handoff stream)

## Unknowns for design pass

1. **Edge semantics for cross-domain handoff.** When `/design` on a
   PM story produces an engineering feature, the resulting edge
   crosses a domain boundary. Is this a regular `parent` or
   `handoff` edge whose endpoints happen to be in different
   domains? Or does it need its own kind (e.g. `cross-domain-handoff`)
   that queries treat specially?
2. **Query default — filter or include.** Default to active-domain
   filtering, with an opt-in flag to traverse cross-domain? Or
   default to inclusive with an opt-in filter? Filtering-by-default
   is safer (less surprising); inclusivity-by-default is more
   honest about cross-domain links.
3. **Active domain at query time vs at write time.** A node's
   `domain` is set at write time (active pack writes it). But what
   reads the "current domain" for queries — `hero.json` only, or
   can a query override it? Probably override-capable for the
   dashboard router (cross-domain views), but pin it down.
4. **Migration of the graph store.** Whether the on-disk graph
   format needs a real migration or can the field be opportunistic
   (read missing-domain as `engineering`).
5. **Backwards compatibility for MCP tools.** External MCP clients
   call `hero_search` etc. today. Adding a domain filter must not
   break clients that don't pass one — default to active domain.

## Boundaries

- **Not** shipping multi-active-domain workspaces. Single-active v1.
- **Not** designing the PM domain pack's specific node types — that's
  `hero-pm` plus whatever `scan-pluggability` decides about scan
  output schema.
- **Not** introducing per-user or per-team namespacing — that's the
  `cloud-admin` initiative.
- **Not** changing graph storage backend.

## Risks

- **Query audit underestimates surface.** Many code paths read the
  graph; missing one means that path silently leaks cross-domain
  data. Treat the audit as the first deliverable, like the
  `spec-type-registry` audit.
- **Cross-domain edges become a liability.** If the edge kind
  decision is wrong, the killer demo (story → feature) either
  doesn't render correctly or breaks engineering's existing graph
  queries.
- **Default-filter surprise.** A user running `hero search` and
  expecting to see everything will be surprised when results are
  scoped to their domain. The CLI/MCP help text and the dashboard
  search affordance need to make active-domain scoping obvious.
- **Backfill of engineering nodes.** All existing nodes need to
  land as `engineering`. Opportunistic read-time backfill is
  cheaper than a one-shot migration but is forever — pin the
  trade-off down.
