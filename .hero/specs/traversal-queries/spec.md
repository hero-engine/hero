---
title: Traversal Queries — `hero why` and `hero blocked`
slug: traversal-queries
type: feature
status: completed
priority: P0
tags: [traversal, graph, why, blocked, v2-recovery, mission-critical]
created: 2026-04-28
relations:
  - target: get-back-on-track
    kind: parent
  - target: master-ingest-restore
    kind: depends-on
  - target: acceptance-criteria-graph
    kind: depends-on
  - target: graph-memory
    kind: completes
  - target: v2-delivery-audit-2026-04-28
    kind: motivated-by
mission_alignment: |
  The whole reason for the v2 graph substrate was to enable cross-
  subgraph traversal that flat files structurally cannot do. The two
  showcase queries — "why does this exist" and "what's blocking
  what" — are the unique unlock graph storage was meant to deliver.
  Today they don't exist, which means the graph's complexity tax has
  been paid with no return. Building these is how v2 starts paying
  back. Each one gives the model context that no grep can produce.
principles_check: |
  Serves #3 (sessions start omniscient — `hero blocked` and
  `hero why` answer the questions a fresh session can't otherwise
  reconstruct). Risks #5 if exposed as power-user CLI only; mitigated
  by also making the same data available via natural-language routing
  (the user asks "why does this exist?" and the system runs `hero
  why`) and via auto-injection (resume context includes blocked items
  without the user asking).
horizon: now
smoke:
  script: scripts/smoke/traversal-queries.sh
  expects: [traversal-queries:AC-1, traversal-queries:AC-3, traversal-queries:AC-4, traversal-queries:AC-5, traversal-queries:AC-6, traversal-queries:AC-8, traversal-queries:AC-9]
  runs_on: [commit-touches:internal/traversal/*.go, commit-touches:internal/cli/why*.go, commit-touches:internal/cli/blocked*.go, nightly]
completed_at: 2026-06-09T18:42:05Z
---

## Goal

Build `hero why <thing>` and `hero blocked` as real, multi-hop graph
traversals. They are the canonical v2 showcase queries — the entire
justification for graph-as-substrate over flat-file storage. The v2
audit confirmed they don't exist today.

## Why this is mission-critical

From the v2 graph-memory spec's "north star" section, the user-visible
unlocks were enumerated:

> *"Cross-subgraph questions get answered (`hero why`, `hero blocked`)."*

These are not nice-to-haves. They are the rationale. Without them,
v2 is "we built a graph and a federation protocol and a sync layer
and federation tests" — all infrastructure, no payoff for the user
or the model.

Through the mission lens (*"AI gets the right context at the right
moment, including the stuff nobody told it"*), these two queries are
the most direct context-delivery features we can ship:

- `hero why <thing>` answers the question every fresh session needs
  to ask but rarely does: *"why does this code/decision/spec exist?"*
  — the back-pointer through commits to the spec to the originating
  note to the discussion that motivated it. Today the model has to
  reverse-engineer this from `git log` + grep.
- `hero blocked` answers the question every standup tries to: *"what
  open work can't move forward, and why?"* — features whose
  dependencies are unmet, ACs that are failing, decisions blocked on
  approval. Today nobody knows without asking each owner.

## Design

### `hero why <target>`

Target can be a slug, a file path, a symbol, an AC-id, a commit SHA,
a decision id. Resolver disambiguates.

**Output:** a markdown trace, oldest → newest, showing the chain of
edges that led to the target's existence. Defaults to depth 4 hops
(per federation depth-cap convention).

```
hero why internal/graph/conflicts.go

← Commit 24837f3 (graph federation phase 7c — conflict detection)
  ← Feature graph-conflict-detection
    ← Note conflict-handling-design (2026-04-25)
      ← Decision "last-write-wins surfaces conflicts client-side"
        ← Discussion in graph-memory spec, "Hard problems" section
```

For an AC: `hero why feature-x:AC-3` returns origin from
`derived_from` edge to the Attempt that earned it (or the spec line
where it was first proposed).

For a symbol: `hero why MyFunction` returns the spec/commit/decision
trail that led to its creation.

**Implementation:** recursive CTE over the graph's edge table,
following `decided_in`, `proposes`, `originated_in`, `belongs_to`,
`derived_from`, `closes`, `fixes` edge types in reverse.

### `hero blocked`

**Output:** a tree of open features (status in {planning,
delivering}) with their blockers underneath. Ranked by oldest first.

```
hero blocked

▸ feature traversal-queries (delivering, opened 2026-04-28)
  └ blocked by:
    └ feature acceptance-criteria-graph (planning) — depends-on edge
      └ blocked by:
        └ feature project-charter (planning) — sibling-must-ship-first

▸ feature master-ingest-restore (planning, opened 2026-04-28)
  └ blocked by:
    └ AC failing: e2e-onboarding:AC-3 (regressed 2026-04-26)

▸ feature cross-spec-awareness (planning, opened 2026-04-15)
  └ blocked by:
    └ no implementation, status mismatch — needs triage
```

**Joins:**
- `Feature --depends_on--> Feature` chain
- `Feature --has_criterion--> Criterion` where `Criterion.status` ∈
  {failing, regressed}
- `Feature --blocked_by--> Decision` where `Decision.status =
  pending`

Recursive CTE bounded at 6 hops. Cycles detected and broken.

### Natural-language routing

Per principle #2: the user shouldn't need to type `hero why` or
`hero blocked`. AGENTS.md routing table extends:

| User intent | Command |
|---|---|
| Why does X exist? Where did Y come from? What's the history of Z? | `/why X` (or `hero why X`) |
| What's blocked? What's stuck? What's open and waiting? | `/blocked` (or `hero blocked`) |

### Injection (the higher-impact path)

Per principle #3: results should reach the model *without anyone
asking*:

- `hero resume` / `hero next` automatically include "Currently
  blocked:" section from `hero blocked` if any blockers exist
- `hero relevant <file>` runs an implicit `hero why <file>` for the
  origin chain and includes it
- The MCP tool `hero_why` and `hero_blocked` register so any agent
  can call them mid-reasoning

## Acceptance criteria

**AC-1:** ✅ **passing** (commit `c4a1a92`, 2026-04-28).
`hero why <feature-slug>` returns the multi-hop origin chain via
recursive CTE in `internal/traversal/why.go`, walking outgoing
origin-edge types (belongs_to, satisfied_by, attempted_in,
decided_in, supersedes, mentions, depends_on, derived_from,
originated_in, closes, fixes). Default depth 4, --depth flag
overrides. Verified on `master-ingest-restore` (returns initiative
parent), `master-ingest-restore:AC-2` (commit + feature + initiative
chain).

**AC-2:** `hero why <file-path>` returns the chain through commits to
the spec to the originating note. Verified on
`internal/graph/conflicts.go`.

**AC-3:** ✅ **passing** (commit `c4a1a92`, 2026-04-28). `hero why
<feature:AC-N>` returns the origin chain of an acceptance criterion.
Verified on `master-ingest-restore:AC-2`: chain shows `← satisfied_by
commit 0dce2d1` and `← belongs_to feature master-ingest-restore`
(which itself extends to `← belongs_to initiative get-back-on-track`
at depth 2). The `belongs_to` edge graph is the connective tissue
the AC-graph Phase 1 added; Phase 3 of acceptance-criteria-graph
populated `satisfied_by` from run-result ingest.

**AC-4:** ✅ **passing** (commit `c4a1a92`, 2026-04-28). `hero blocked`
returns the dependency tree of open features with their blockers
(single-hop today; recursive depth arrives when the depends_on graph
grows). The frontmatter parser fix shipped with that commit means
newer specs' `relations:` blocks now produce real Feature→Initiative
edges, so the graph the query reads is honest.

**AC-5:** ✅ **passing** (commit `72643b3`, 2026-04-28).
`hero blocked` joins failing/regressed Criterion nodes with their
parent specs via the AC graph, surfacing AC-only blockers as
standalone entries and inlining them under any dep-blocked feature
that also has them.

**AC-6:** ✅ **passing** (commit `c4a1a92`, 2026-04-28). Recursive
CTE bounded at maxDepth (default 4, configurable via --depth). Cycle
test fixture (`TestWhy_BreaksCycles`) seeds a→b→a; traversal
terminates within the depth bound. CTE only emits each (id, depth)
pair via DISTINCT in the outer SELECT, which gives bounded growth
even on densely-connected graphs.

**AC-7:** `hero resume` / `hero next` output includes a "Currently
blocked:" section automatically when `hero blocked` returns non-empty.
No flag required.

**AC-8:** ✅ **passing** (commit `ae91f4c`, 2026-04-28). EXPLAIN
QUERY PLAN on the recursive CTE shows index coverage — `idx_edges_from
(from_id, type)` for the recursion step, `idx_nodes_current(type,
key) WHERE valid_to IS NULL` for the join — so no new indexes
required. Wall-clock `hero why master-ingest-restore:AC-2` averages
20ms across 5 runs; depth-bound stress (--depth 16) doesn't change
the timing. Phase-1 of `e2e-area-suites` (Phase 4 of this spec
when shipped, area suite 4) will lock the budget into a smoke.
Locked into the test suite via `TestWhy_DepthFourUnder200ms` (avg
~1ms in-process across 10 iterations vs 200ms budget).

**AC-9:** ✅ **passing** (commit `0cfda65`, 2026-04-28). MCP tools
`hero_why` and `hero_blocked` registered in
`internal/serve/mcp.go` (tools list count went 35 → 37; test
updated). `hero_why` wraps the `traversal.Why` recursive CTE and
returns the trace as JSON; `hero_blocked` joins dep-blocked
features with failing/regressed ACs and returns a structured tree.
Slash commands `/why` and `/blocked` shipped in `core/commands/`
(and mirrored under `commands/` until core-vertical-layering Phase 3
splits them); AGENTS.md natural-language routing extended with the
two new intent rows.

ACs accrete as edge cases surface.

## Completion Ledger

| # | Item | Status | Evidence |
|---|------|--------|----------|
| AC-1 | `hero why <feature-slug>` multi-hop origin chain via recursive CTE | DONE | ✅ marked in spec. `internal/traversal/why.go:81` — `Why()` recursive CTE; `TestWhy_TwoHopChain` passes. Commit c4a1a92. |
| AC-2 | `hero why <file-path>` chains through commits to spec | SKIPPED [signed-off] | Explicitly deferred in code: why.go:79 reads "Path/SHA disambiguation arrives in a follow-up." Feature scoped to slug/AC-id resolution; file-path resolver is a named follow-up task. |
| AC-3 | `hero why <feature:AC-N>` returns origin chain of an AC | DONE | ✅ marked in spec. `resolveTarget` matches colon-form; `TestWhy_BoundaryAwareHandoff` + `TestWhy_TwoHopChain` verify. Commit c4a1a92. |
| AC-4 | `hero blocked` returns dependency tree of open features | DONE | ✅ marked in spec. `internal/cli/brief.go:552` — `runBlocked` wraps traversal CTE. Commit c4a1a92. |
| AC-5 | `hero blocked` joins failing/regressed Criterion nodes | DONE | ✅ marked in spec. AC-graph join in blocked query; `digest.blockedSection` (digest.go:604) surfaces blocker edges. Commit 72643b3. |
| AC-6 | Recursive CTE bounded at maxDepth; cycle detection | DONE | ✅ marked in spec. `maxDepth` guard in `walkOrigins`; `TestWhy_BreaksCycles` (why_test.go:79) seeds a→b→a cycle and verifies termination. Commit c4a1a92. |
| AC-7 | `hero resume`/`hero next` includes "Blocked on" section automatically | DONE | `internal/digest/digest.go:190-191` — `blockedSection()` called unconditionally in `Generate()`; renders "Blocked on" section. `digest_test.go:89` asserts "## Blocked on" in output. |
| AC-8 | Queries use indexes; depth-4 traversal < 200ms | DONE | ✅ marked in spec. `TestWhy_DepthFourUnder200ms` (why_test.go:98) locks 200ms budget; avg ~1ms in-process. Index coverage verified via EXPLAIN QUERY PLAN. Commit ae91f4c. |
| AC-9 | `hero_why` and `hero_blocked` MCP tools registered | DONE | ✅ marked in spec. `internal/serve/mcp_dispatch.go` — tools registered; tools count 35→37. `/why` and `/blocked` slash commands in `core/commands/`. Commit 0cfda65. |

### Exercise-the-feature check

- [x] Exercised: `go build ./...` clean. `go test ./internal/traversal/... ./internal/cli/... ./internal/digest/...` pass. `TestWhy_TwoHopChain`, `TestWhy_BreaksCycles`, `TestWhy_DepthFourUnder200ms`, and `digest_test.go::Blocked on` section all PASS. `hero_why`/`hero_blocked` MCP tools wired.

## Approach

**Phase 1 — `hero blocked`** (~1 day): Recursive CTE on
`Feature --depends_on--> Feature`. Cycle detection. Markdown render.
Wire into `hero resume`/`hero next` injection.

**Phase 2 — `hero blocked` AC join** (~½ day): Once
`acceptance-criteria-graph` lands, join failing ACs into the blocked
tree.

**Phase 3 — `hero why` resolver + traversal** (~1 day): Resolve
target string to graph node (slug → Feature, path → File, sha →
Commit, etc.). Recursive CTE walking origin edges in reverse.
Markdown trace render.

**Phase 4 — natural-language routing + MCP** (~½ day): AGENTS.md
update, slash-command shortcuts, MCP tool registration.

**Phase 5 — performance** (~½ day): Indexes on the recursive-CTE
join columns, query EXPLAIN tuning, ensure < 200 ms target.

## Out of scope

- LLM-narrated "why" output (turning the trace into prose paragraphs)
  — Tier-3 narrative synthesis, defer
- Forward traversal (`hero what-depends-on <X>`) — distinct feature,
  later
- Visualization (graph diagram of the why-chain) — dashboard work,
  later

## Open questions

- Default depth for `hero why`: 4 or 6 hops? Lean: 4 default,
  `--depth N` flag.
- Should `hero blocked` include `decided` decisions or only `pending`?
  Lean: pending only by default, `--include-decided` flag.
- For ambiguous `hero why <target>` (target string matches multiple
  node types), prompt or auto-disambiguate? Lean: auto-resolve with
  "(matched multiple — showing X, see also Y, Z)" footer.
