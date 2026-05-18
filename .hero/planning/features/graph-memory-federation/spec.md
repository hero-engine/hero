---
title: Graph Memory Federation — Multi-Repo, Multi-Team, Cross-Unit Topology
slug: graph-memory-federation
type: feature
status: planning
priority: P0
tags: [memory, federation, multi-repo, scaling, foundational]
created: 2026-04-26
relations:
  - target: graph-memory
    kind: parent
horizon: next
smoke: deferred
---

## Goal

Make hero's knowledge graph work for the real shape of large
engineering orgs: 100+ developers, hundreds to thousands of repos,
multiple teams per repo, multiple business units per org, devs
"flipping around" between repos all day. Federate per-repo team
graphs into unit-level join graphs that surface cross-repo signal
without becoming noisy, org-wide soup.

The ambition is small but pointed: deliver three "magic moments"
that are otherwise structurally impossible — at-commit blast-radius,
session-start cross-repo context, block-time cross-repo blocker
visibility. Everything else is plumbing in service of those three.

## The topology — many + join, NOT org-wide

```
                ┌──────── public layer ────────┐
                │ open-source patterns,        │
                │ shared learnings, OSS        │
                └─────────────┬────────────────┘
                              │
       ┌──────────────────────▼────────────────────────┐
       │       unit graphs (one per business unit /    │
       │       product line — many per org)            │
       │  cross-repo Features, Decisions, Persons,     │
       │  Issues, Sprints, cross-repo edges            │
       └────┬─────────────┬──────────────┬─────────────┘
            │             │              │
    ┌───────▼──┐   ┌──────▼──┐    ┌──────▼──┐
    │ repo A   │   │ repo B  │    │ repo C  │  (per-repo team graphs;
    │ team     │   │ team    │    │ team    │   live on team server,
    │ graph    │   │ graph   │    │ graph   │   partitioned by repo)
    └─────┬────┘   └─────┬───┘    └─────┬───┘
          │              │              │
          └──────── per-dev local ──────┘
                  (.hero/graph.db on each
                   developer's machine)
```

**Why not org-wide:** at 1000+ repos with varied BUs and unrelated
products, an org-wide graph is structurally noisy. Most of those
repos have no business knowing about each other. The unit
(business unit / product line) is the right join scope — typically
5–50 related repos that genuinely share Features, Decisions, and
cross-repo dependencies.

Each org has many unit graphs. Repos belong to one unit. Cross-unit
links are unusual and are flagged for human review (they often
indicate a process problem more than a graph one).

## Four scope levels

`scope` is the partition key for sync routing:

| Scope | Lives in | Visible to | Sync direction |
|---|---|---|---|
| `local` | dev's `.hero/graph.db` only | the dev | never leaves machine |
| `team` | per-repo team graph | repo collaborators | local ↔ repo team server |
| `unit` | unit-level join graph | unit members | repo team server ↔ unit join graph |
| `public` | global registry | everyone | pull-only from registry |

Default scope: `team`. Most ingest produces team-scope nodes. A node
gets promoted to `unit` when (a) it represents a cross-repo entity
(multi-repo Initiative, Person who works across repos), or (b) a
cross-repo edge has at least one endpoint in another repo.

## Partition columns on every node and edge

Beyond `scope`, every team-scope-or-higher row carries:

- **`repo`** — which repo this came from (always set when not pure
  local agent state; empty for `local` scope or pure agent state)
- **`unit`** — which business unit / product line (set on
  team-scope+ nodes once unit identity is known; empty until then)

These are columns, not props, because:
- They're partition keys for sync filtering (`POST /v1/graph/push?repo=…&unit=…`)
- They're indexed for fast scoped queries
- They lock the federation contract — every binary that writes the
  graph stamps the right partition or fails to write

## Identity across repos

Three classes of node identity:

**Globally stable** (same key across all repos):
- `Person` — keyed by lowercase email
- `Commit` — keyed by SHA (already global)
- `Issue` — keyed by tracker key (Jira `PROJ-123`, GitHub `GH#42`)

**Repo-scoped** (key is `<repo>:<localKey>`):
- `Package`, `File`, `Symbol` — these are repo-internal entities

**Unit-scoped, possibly aliased**:
- `Feature`, `Initiative`, `Decision` — usually repo-scoped, but
  multi-repo features (e.g. "drop log4j across all 14 services")
  exist as a single unit-scope node with `alias_of` edges from each
  repo's local feature node.

## Aliases — `alias_of` as a graph-native concept

When two nodes turn out to be the same logical entity (renamed
feature, multi-repo initiative split per-repo, two specs that
crystallized into one), an `alias_of` edge marks the non-canonical
node as pointing at the canonical:

```
(Feature: payments-rewrite@repo-a) ── alias_of ──▶ (Feature: payments-rewrite@unit-X)
(Feature: payments-rewrite@repo-b) ── alias_of ──┤
```

**Resolution rule:** when a query returns a node that has an outgoing
`alias_of` edge, hero follows the chain (capped at 5 hops to handle
bad data) and returns the canonical node. Aliases are bitemporal
like any edge — renaming history is preserved.

This is one new edge type, no schema change. The store gets a
`ResolveAlias(typ, key) → canonicalID` helper that walks chains.

## Traversal: pragmatic, not perfect

Per the user's direction: complex graphs at scale will produce deep
chains and cycles. We do **not** try to solve them perfectly.

- **Default depth cap: 4 hops.** Configurable per query.
- **Bounded results: top 50 by recency by default.** No "give me
  every transitive caller across 20 repos" by accident.
- **Cycle detection: visited-set tracked per traversal**, just enough
  to avoid infinite loops. Not exhaustive cycle analysis.
- **Stale conflicts may sit forever.** Phase 6a surfaces them; humans
  resolve; the graph doesn't auto-merge prose. Last-writer-wins for
  enum/status fields. Tag arrays merge as set unions.

Better to give a useful 80% answer in 50ms than a perfect answer in
3 seconds that nobody waits for.

## Per-team scoping inside a repo

Monorepos with multiple teams: every team gets a `Team` node. Work
items (Feature, Initiative, Issue) get a `team_owns` edge from the
owning team:

```
(Team: billing) ── team_owns ──▶ (Feature: subscription-tiers)
(Team: auth)    ── team_owns ──▶ (Package: services/auth)
```

`hero blocked` / `hero next` / sprint projections filter on team
membership of the current dev (resolved via `Person → member_of → Team`
edges). No new scope needed — just data discipline.

## Cross-repo edges

The killer capability. When repo A's `internal/auth.NewToken` is
imported by repos B, C, D:

- Repo A's codescan creates: `Symbol: a:internal/auth.NewToken` (scope=team, repo=a)
- Repos B/C/D's codescan creates a `cross_imports` edge whose `to_id` references the canonical symbol key, even though the row doesn't exist in B/C/D's local graph
- The unit join graph holds a `Symbol: a:internal/auth.NewToken` shadow row (replicated from A) plus all the cross-repo `cross_imports` edges
- Querying the unit graph: "who imports A's NewToken?" → instant answer with downstream owners, recent activity, open features that touch them

**Cross-repo edge storage rule:** cross-repo edges live at the
**lowest scope that contains both endpoints**. Same-unit edges live
in that unit's join graph. Cross-unit edges (rare and noteworthy)
get promoted to `public` or flagged.

## The three magic moments

These are the tests for whether federation feels like value.

**1. At commit time:**
```
$ git commit -m "rename NewToken to MintToken"
hero pre-commit:
  ⚠ This change touches a public symbol imported by 3 other repos:
    - service-billing (last touched: yesterday, owner: alice)
    - service-orders  (last touched: 3 days ago, owner: bob)
    - lib-shared      (last touched: 2 weeks ago, owner: carol)
  → Continue, or run `hero impact` for full blast radius?
```

**2. At session start:**
```
$ hero resume
Welcome back. In repo `service-billing`. Last session: 6 days ago.

Since you last worked here:
  • 4 cross-repo decisions affect this repo (run `hero why` for each)
  • 2 attempts in repo `lib-shared` are relevant to your in-flight
    work — check before re-trying the same approach
  • 1 blocker resolved upstream — `subscription-tiers` is unblocked

Open work:
  • subscription-tiers (claimed by you, in-progress)
```

**3. At block time:**
```
$ hero blocked
This repo is not blocked locally.

Cross-repo blockers affecting your work:
  • subscription-tiers ← waiting on `auth-token-rotation` decision
    (lives in repo `service-auth`, owned by dave, scheduled for sprint
    2026-05-04). Last activity: 4 days ago.
```

If those three moments work and feel snappy, the federation succeeds.
If they don't, the rest of the architecture is plumbing nobody sees.

## Sync API — partitioned by repo from day one

Phase 7 ships per-repo sync. The contract locked in phase 3b:

```
POST /v1/graph/push?repo=<repo>&unit=<unit>
  body: { since: <last_push_at>,
          nodes: [...], edges: [...] }
  resp: { accepted: N, conflicts: [...], server_time }

GET  /v1/graph/pull?repo=<repo>&since=<cursor>&include=team,unit
  resp: { nodes, edges, next_cursor }
```

Server shards storage internally per-repo. Unit-level joins are
materialized via background indexers that watch repo-graph deltas
and update the unit graph's cross-repo edge tables.

Auth: each repo's GitHub App identity governs read/write permissions
for its own partition. Unit join graphs require unit-membership.

## Conflict + drift handling at scale

We already have:
- Phase 6a — conflict detection (graph-level conflicts reuse the
  same machinery as code conflicts: same logical key, divergent
  edits, surface both versions)
- Phase 6b — sequencing + pattern mining (proposes resolution
  ordering, finds repeated dead ends across teams)

What we add for federation:
- **Last-writer-wins as default merge** for enum/status props
  (status, priority, claimed_by)
- **Set-union merge** for tag arrays
- **Manual conflict-flag for prose** — surface both versions, let
  humans pick or merge
- **Per-edge merge strategy** as a node-type config (decisions:
  last-writer-wins; design notes: manual)
- **No automatic deep-cycle resolution** — accept that some chains
  will be loopy; depth caps prevent infinite traversal

Acceptable failure mode: 0.1% of nodes have unresolved conflicts
at any time, surfaced in `hero conflicts` (already exists). Devs
resolve as they touch the area. The graph remains useful with
imperfections.

## Game-changing capabilities (locked in)

These are the queries the federation specifically enables. Phases
5/8 implement them:

- `hero impact <symbol|file|package>` — blast radius across repos
  (cross-repo `cross_imports` traversal + ownership + recency)
- `hero blocked --cross-repo` — blocker chains spanning repos
- `hero who-touches <thing>` — active sessions touching same area,
  pre-emptive collision avoidance
- `hero learnings <topic>` — `Attempt` + `Pattern` nodes across
  repos, deduped, ranked by recency
- `hero migration-status <initiative>` — multi-repo Initiative
  rollup with per-repo Feature progress
- `hero recall <topic> --unit` — searches unit graph + relevant
  per-repo data, not just current repo

## Schema contracts — locked NOW (phase 3b)

To keep the local graph forward-compatible, phase 3b adds:

1. **`repo` TEXT NOT NULL DEFAULT ''** column on `nodes` and `edges`
2. **`unit` TEXT NOT NULL DEFAULT ''** column on `nodes` and `edges`
3. **Indexes**: `idx_nodes_repo`, `idx_nodes_unit`, `idx_edges_repo`
4. **Scope constants**: `ScopeLocal`, `ScopeTeam`, `ScopeUnit`,
   `ScopePublic`. Drop `ScopeOrg` (never built but retained as a
   concept earlier). Default remains `ScopeTeam`.
5. **`alias_of` edge type** — documented, no code beyond a
   `ResolveAlias` helper with depth-cap=5
6. **`Person` keys** stay lowercase email globally
7. **Schema bumps to v2** — clean migration on next open

After 3b, every existing ingest path stamps `repo` on every node
it writes. `unit` stays empty until phase 8 (when unit identity is
configured per-repo).

## Phased plan revision

| Phase | Scope | Status |
|---|---|---|
| 1 | Schema + code subgraph | ✅ |
| 2 | Work subgraph (specs, sessions, git) | ✅ |
| 3a | Raw docs, attempts, issue refs | ✅ |
| **3b** | **Federation schema contracts** | **next** |
| 4 | NEXT.md + sprint todos as projections | |
| 5 | `hero why`, `hero blocked`, `hero impact` (cross-repo from day 1) | |
| 6 | Jira ingest (read-only) | |
| 7 | Team-server sync (per-repo partitioned API) | |
| 8 | Unit-level join layer (cross-repo dogfood) | |
| 9 | GitHub Pages projection | |
| 10 | Tier-2 LLM extraction (notes → Decision/Concept) | |

Federation is no longer a "later" concern — it's threaded through
phases 5/7/8 with the schema locked early so nothing rewrites.

## Open questions

- **Unit identity configuration**: where does a repo declare its
  unit? Likely `hero.json: { "unit": "platform-payments" }`. Lock
  in phase 6 or 7.
- **Cross-unit edges**: rare. Promote to `public`, or always flag for
  manual review? (Probably both — flag, then offer promotion.)
- **`Team` membership data source**: GitHub teams API, Jira groups,
  manual config? Likely a hybrid — pluggable in phase 7.
- **Replication latency budget**: how stale can the unit join graph
  be before it stops being magic? Soft target: <5 min for code-graph
  cross-repo edges, <30 min for work-graph deltas.

## Success criteria

- Local hero binary continues to work standalone (no regressions)
- Phase 8 dogfood: 3 related repos in one unit, cross-repo `hero
  impact` answers "who imports this" in <500ms
- Phase 8 magic-moment 1 (commit-time impact warning) demoable
- Zero schema changes between 3b and phase 8 — contract holds
