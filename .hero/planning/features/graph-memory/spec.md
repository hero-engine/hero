---
title: Graph Memory — Unified Substrate for Hero's Knowledge Corpus
type: feature
status: planning
priority: P0
tags: [memory, knowledge-graph, architecture, foundational]
created: 2026-04-26
relations:
  - target: cross-org-intelligence
    kind: sibling
  - target: cross-spec-awareness
    kind: sibling
  - target: cloud-dashboard
    kind: sibling
horizon: next
smoke: deferred
---

## Thesis

> **We already capture the right signals. We just store them flat and
> read them naively. Hero v.next stores them as a graph and reads them
> with traversal — the human/agent flow on top stays exactly the same.**

Hero today produces graph-shaped data in three places (codescan,
phase 6 work intelligence, knowledge/memory files) and stores it as
disconnected markdown trees. Each is locally useful and globally
disconnected. This feature unifies all three into one bitemporal
knowledge graph with markdown as the projection layer — preserving
the natural-language UX while unlocking traversal-based capabilities
that flat files structurally cannot provide.

## Background — why graph

Industry research (Microsoft GraphRAG, Zep, Cognee, Memento, A-MEM,
Graphiti) consistently shows graph memory outperforms vector and flat
memory for agentic systems on every relational dimension:

| Capability | Vector / flat | Graph |
|---|---|---|
| Multi-hop reasoning ("what depends on X?") | ✗ | ✓ |
| Temporal queries ("what was true at T?") | ✗ | ✓ via bitemporal |
| Entity resolution (renames, merges) | ✗ | ✓ |
| Provenance preservation | ✗ | ✓ via source edges |
| Contradiction handling (fact evolution) | ✗ | ✓ via valid_to |
| Cross-source joins | ✗ | ✓ |

Reported gains: Zep 18.5% accuracy improvement and 70× context-token
reduction over vector baselines on LongMemEval; GraphRAG 35% precision
gain over vector-only RAG; Memento 90.8% end-to-end accuracy vs. ~41%
for vector baselines.

For hero — which is fundamentally a context-engineering system across
sessions, tools, and teams — graph is the right substrate.

## North star — what users see

Nothing visibly changes about the daily flow. `hero scan`,
`hero next`, `hero handoff`, `/resume`, `hero note` all still work
the same way. Markdown is still the surface. The delta is invisible:

- Things users used to grep for now appear in the right place.
- Stale facts auto-invalidate instead of accumulating.
- Cross-subgraph questions get answered (`hero why`, `hero blocked`).
- Multi-dev teams share one consistent picture instead of merging
  per-developer NEXT.md files.
- Wiki projection grows from "spec sync" to a living team narrative.

Harness-agnostic stays a hard constraint. Every harness (Claude Code,
Cursor, Codex, Aider, opencode, plain humans, CI bots) talks only to
the `hero` CLI. Markdown out by default; `--json` for tool consumers.
The graph is implementation detail no harness needs to know about.

## Source-of-truth split

The single most important architectural call: which artifacts are
hand-authored markdown (humans write, graph ingests) vs. graph-sourced
(graph is canonical, markdown is regenerable projection).

| Artifact | Hand-authored | Source of truth |
|---|---|---|
| `planning/features/*/spec.md` | ✅ | **markdown** — humans iterate, reviewed in PRs |
| `planning/initiatives/*` | ✅ | **markdown** |
| `knowledge/notes/*/spec.md` | ✅ | **markdown** — `hero note`, brainstorms, design thinking |
| `knowledge/raw/*.md` | ❌ but immutable | **filesystem (audit)** — bytes never change |
| `knowledge/context/<ingested>/spec.md` | ❌ (stub then enriched) | **graph** — projection over Document + extracted entities |
| `knowledge/context/project-overview/spec.md` | ❌ (auto) | **graph** — projection over project-scan nodes |
| `knowledge/code/*/spec.md` | ❌ (auto via codescan) | **graph** — pure projection |
| `NEXT.md`, `MEMORY.md`, sprint todos, attempt logs | ❌ (auto) | **graph** — projection |
| External tracker (Jira, GitHub Issues) | ❌ | **tracker** — graph mirrors |
| External wiki (GitHub Pages, Confluence) | ❌ | **graph** — wiki is projection |
| Team-scope nodes (shared across devs) | ❌ | **team server** — authoritative for `scope=team` |

**Only two places are hand-authored: `plans/` and `notes/`.** Notes
graduate to plans when formal enough; plans become specs when shipped.
Two phases of human-driven thought, everything else generated.

## The four knowledge subdirectories — clarified

Investigation of `internal/cli/note.go`, `internal/cli/ingest.go`, and
`internal/codescan/generate.go` made these explicit:

- **`notes/`** — written via `hero note [slug]`. "Brainstorms,
  conversation captures, stream-of-consciousness, anything that isn't
  ready to be a spec yet." Same `<slug>/spec.md` shape as plans.
  Hand-authored. Rich extraction targets — current notes contain
  decisions, alternatives, references, risks, verdicts.
- **`context/`** — mixed: auto-generated `project-overview/` from
  codescan + ingested external content (`hero ingest --type=context`).
- **`raw/`** — immutable audit copy. `hero ingest` writes the
  unmodified fetched bytes here with a header (source, ingested-at,
  title). The processed entry goes to `context/`/`convention/`/`decision/`
  per `--type`. Bytes never change; re-extraction is always possible.
- **`code/`** — pure codescan output. Per-package `spec.md` with
  symbols, imports, line counts, plus checksum + enrichment caches.
  100% derivable from the codebase.

`raw/` has a special property worth preserving: **it's the only place
where bytes are permanent**. Everything else is regeneratable. Every
`Document` graph node has an edge `derived_from → raw/<slug>.md` with
content hash. If extraction logic improves, re-run against raw, get
better nodes, history preserved.

## The three subgraphs that already exist (unified)

| Subgraph | What's in it | Currently lives in |
|---|---|---|
| **Code** | Repos, packages, files, symbols, imports, cross-repo deps | `knowledge/code/*/spec.md` |
| **Work** | Features, sessions, commits, conflicts (6a), sequencing (6b), patterns, cross-org intel (6c) | scattered — planning md + computed at runtime |
| **Knowledge** | Decisions, attempts, learnings, memories, references, notes, ingested docs | `knowledge/notes/`, `~/.claude/.../memory/`, NEXT.md, prose in specs |

Today they don't talk to each other. Unifying enables cross-subgraph
queries that flat files structurally cannot answer:

- "I'm changing `internal/cli` — what features depend on it?" →
  code graph (touches) ⨝ work graph (feature touches package)
- "Why does `cli/handoff.go` exist?" → code graph (file) → work graph
  (commit) → knowledge graph (decision/spec that motivated it)
- "Show me everywhere this Jira ticket touched code" →
  tracker (issue) → work graph (commits) → code graph (symbols)
- "Trace this idea from notepad to shipped feature" →
  Note → proposes → Decision → decided_in → Spec → Feature → Commits

## Schema (SQLite, JSON1, recursive CTEs)

Storage: SQLite at `.hero/graph.db`. Embedded, zero-install, plays
with Go, recursive CTEs handle multi-hop traversal at hero's scale
(<1M nodes for years). Schema maps cleanly to a property graph DB if
scale ever forces a swap.

```sql
-- nodes: every entity in the graph
CREATE TABLE nodes (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    type          TEXT    NOT NULL,        -- 'Package', 'Feature', 'Note', ...
    key           TEXT    NOT NULL,        -- stable logical id, e.g. 'internal/cli'
    props         TEXT    NOT NULL,        -- JSON blob, type-specific fields
    scope         TEXT    NOT NULL,        -- 'local' | 'team' | 'org' | 'public'
    content_hash  TEXT,                    -- hash of source content
    source        TEXT    NOT NULL,        -- JSON: {kind, path, commit, session, ...}
    valid_from    TEXT    NOT NULL,        -- RFC3339 (event time)
    valid_to      TEXT,                    -- NULL = still current
    ingested_at   TEXT    NOT NULL         -- RFC3339 (when hero learned)
);

CREATE UNIQUE INDEX idx_nodes_current
    ON nodes(type, key) WHERE valid_to IS NULL;
CREATE INDEX idx_nodes_type     ON nodes(type);
CREATE INDEX idx_nodes_scope    ON nodes(scope);
CREATE INDEX idx_nodes_ingested ON nodes(ingested_at);

-- edges: typed relationships
CREATE TABLE edges (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    from_id       INTEGER NOT NULL REFERENCES nodes(id),
    to_id         INTEGER NOT NULL REFERENCES nodes(id),
    type          TEXT    NOT NULL,        -- 'imports', 'blocks', 'mentions', ...
    props         TEXT    NOT NULL DEFAULT '{}',
    scope         TEXT    NOT NULL,
    source        TEXT    NOT NULL,
    valid_from    TEXT    NOT NULL,
    valid_to      TEXT,
    ingested_at   TEXT    NOT NULL
);

CREATE UNIQUE INDEX idx_edges_current
    ON edges(from_id, type, to_id) WHERE valid_to IS NULL;
CREATE INDEX idx_edges_from     ON edges(from_id, type);
CREATE INDEX idx_edges_to       ON edges(to_id, type);
CREATE INDEX idx_edges_ingested ON edges(ingested_at);

-- sync_state: team server replication bookkeeping
CREATE TABLE sync_state (
    server_url       TEXT PRIMARY KEY,
    last_push_at     TEXT,
    last_pull_at     TEXT,
    last_pull_cursor TEXT,
    org              TEXT NOT NULL
);

-- meta: schema version, install id
CREATE TABLE meta (key TEXT PRIMARY KEY, value TEXT NOT NULL);
INSERT INTO meta(key, value) VALUES
    ('schema_version', '1'),
    ('install_id', lower(hex(randomblob(8))));
```

**Design rationale:**

- **Bitemporal from day one.** `valid_from`/`valid_to` for world-time;
  `ingested_at` for when hero learned. Updates never overwrite —
  they invalidate (`valid_to = now`) and insert new rows. Free history,
  free "what did we know at T?" queries, conflict-detection-friendly.
- **`(type, key)` is the logical identity.** `id` is internal.
  Sync uses `(type, key)` so it's deterministic across clients and
  upserts are idempotent.
- **`scope` is a column, not a separate table.** Every fact carries
  its own visibility. Sync filters on this; `local` never leaves the
  machine.
- **`source` is JSON.** Sources are heterogeneous (commit sha, file
  path, Jira issue, session id, raw doc hash). JSON keeps the schema
  universal; common patterns get partial JSON1 indexes later.
- **Append-only.** No UPDATEs on existing rows except setting
  `valid_to`. Audit + sync conflict resolution both depend on this.

## Node + edge catalog (v1)

**Code subgraph:**

| Node | Key |
|---|---|
| `Repo` | repo name |
| `Package` | `<repo>:<path>` |
| `File` | `<repo>:<path>` |
| `Symbol` | `<repo>:<pkg>:<name>` |

Edges: `belongs_to` (Symbol→Package, Package→Repo), `imports`
(Package→Package, cross-repo aware), `defines` (File→Symbol),
`references` (Symbol→Symbol, Tier 2 extraction).

**Work subgraph:**

| Node | Key | Source |
|---|---|---|
| `Initiative` | slug | `planning/initiatives/*/` |
| `Feature` | slug | `planning/features/*/spec.md` |
| `Phase` | name | extracted from specs |
| `Sprint` | id | Jira sprint |
| `Issue` | Jira key (PROJ-123) | Jira API |
| `Decision` | slug | extracted from specs/notes |
| `Attempt` | `<session>:<slug>` | NEXT.md "Tried and failed" |
| `Conflict` | hash | Phase 6a |
| `Pattern` | slug | Phase 6b |
| `Commit` | sha | git log |
| `Session` | handoff-id | `hero handoff` |
| `Person` | git-author email | git log + Jira |

Edges: `belongs_to`, `blocks`, `depends_on`, `decided_in`,
`attempted_in`, `fixes`, `breaks`, `authored_by`, `touches`
(Commit→File), `closes` (Commit→Issue), `evolved_into`,
`superseded_by`, `mentioned_in`.

**Knowledge subgraph:**

| Node | Key | Source |
|---|---|---|
| `Note` | slug | `knowledge/notes/*/spec.md` |
| `Document` | content-hash | `knowledge/raw/*.md` |
| `Memory` | `<type>:<slug>` | `~/.claude/.../memory/*.md` |
| `Concept` | normalized-term | Tier 2 extraction |

Edges: `mentions`, `references`, `summarizes`, `extracted_from`,
`proposes` (Note→Decision), `originated_in` (Decision→Note),
`derived_from` (Document→raw file).

**Cross-subgraph edges (the unlock):**

- `Feature → touches → Package`
- `Issue → resolved_by → Commit → touches → File`
- `Note → proposes → Decision → decided_in → Spec → belongs_to → Feature`
- `Memory → references → anything`

## Ingest tiers

Three tiers, layered by cost and frequency:

**Tier 1 — deterministic (every event):**
- git hooks → `Commit`, `touches`, `authored_by`
- Planning file watcher → `Feature`, `Spec`, `belongs_to`
- `hero scan` → code subgraph (Repo/Package/File/Symbol/imports)
- `hero next checkpoint` → `Session` + state edges
- `hero ingest` → `Document` + `derived_from`
- Memory file writes → `Memory` nodes

**Tier 2 — light extraction (small LLM call, async per event):**
- Notes prose → `Decision`, `Alternative`, `Concept` nodes via
  extraction; `proposes`/`mentions`/`references` edges
- Spec doc decisions sections → `Decision` nodes with rationale
- NEXT.md "Tried and failed" → `Attempt` nodes with outcome
- Commit messages → `fixes`/`closes`/`breaks` edges
- Ingested raw docs → entity extraction for `context/` projection

**Tier 3 — heavy extraction (larger LLM, periodic):**
- Cross-document entity resolution (this `cloud-billing` and that
  `cloud-billing` are the same node)
- Pattern detection across attempts (already partly built — Phase 6b)
- Narrative synthesis (turn linked subgraph into a paragraph for the
  GitHub Pages "this week" view)

Tier 1 is enough for the system to be useful day one. Tiers 2 and 3
enrich it over time without blocking anything. Default models:
Haiku 4.5 for ingest, Sonnet 4.6 for periodic synthesis.

## Projection layer

NEXT.md is no longer source — it's `SELECT … FROM graph`. Same for
sprint todos, MEMORY.md, the per-package code specs, and external
wiki output. The projection engine takes a query + a render template
and emits markdown (or JSON for tool consumers).

**Render targets:**

```
graph.db
   │
   ▼
projection engine
   │
   ├── local md (NEXT, MEMORY, sprint, code/<pkg>/spec.md, etc.)
   ├── GitHub Pages site  ← living team doc, replaces wiki-sync
   ├── GitHub Wiki         ← legacy alt
   ├── Confluence          ← enterprise
   └── Jira comments       ← back-write commit links into tickets
```

**Performance constraint:** projections are deterministic and fast
(<100ms). LLM calls happen in ingest, never in projection. Otherwise
the magic stops feeling magic.

**User-edits-the-projection rule:** users do not hand-edit NEXT.md,
MEMORY.md, or any other projection. Hero writes them. If a user does
edit one, the next projection wins (the projection is regenerated, not
merged). Hand-authored content lives in `plans/` or `notes/`.

## Wiki feature → GitHub Pages live view

Today's `hero wiki-sync` (`internal/wiki/`) pushes completed specs
outbound to GitHub Wiki / Confluence. One direction, one node type.

The graph design generalizes it: wiki sync becomes one of several
render targets of the projection engine. Default new target is
**GitHub Pages** — a static site pushed to `gh-pages` (or a configured
branch), live, indexable, link-shareable.

**Page types the graph naturally produces:**

- `Features/<name>` — spec body (from md) + live status + recent
  commits + attempts + linked decisions
- `Initiatives/<name>` — initiative md + child features rolled up
- `Decisions/<id>` — what we decided, why, what we tried first
- `This week` — Tier 3 narrative synthesis over recent graph deltas
- `Learnings` — accumulated `Attempt` + `Pattern` nodes, deduped
- `Code map` — package graph visualization

**Bidirectional:** comments/edits in Confluence and GitHub Wiki
reverse-ingest as `Comment` nodes attributed to their author. Closes
the loop on team narrative.

## Team server fit

One team server, not two. The Phase 5–7 cloud server already has
auth (GitHub App), multi-org, governance scaffolding. Repurpose the
in-progress server as the team graph host:

```
  Dev A's repo                Team Server                Dev B's repo
  ┌────────────┐              ┌────────────┐             ┌────────────┐
  │ .hero/     │              │ authoritative            │ .hero/     │
  │  graph.db  │ ←─ sync ──→  │ team graph  │ ←─ sync ─→ │  graph.db  │
  │  (local)   │              │ (shared)    │            │  (local)   │
  └────────────┘              └──────┬──────┘            └────────────┘
                                     │
                              ┌──────▼──────┐
                              │ cross-org   │
                              │ intel (6c)  │
                              └─────────────┘
```

**Three layers of visibility** (every node carries `scope`):

- `local` — your sessions, scratch attempts, personal memory.
  Never leaves the machine.
- `team` — features, decisions, blockers, sprint state, conflicts.
  Syncs to team server; visible to teammates.
- `org` / `public` — cross-org patterns, shared learnings (Phase 6c).

**Sync API (extends existing in-progress server):**

```
POST /v1/graph/push
  body: { since: <client_last_push_at>,
          nodes: [...], edges: [...] }   // scope in {team, org, public}
  resp: { accepted: N, conflicts: [...], server_time: ... }

GET  /v1/graph/pull?since=<cursor>&scope=team
  resp: { nodes: [...], edges: [...], next_cursor: ... }
```

**Semantics:**

- Push is idempotent — resending `(type, key, ingested_at)` is a no-op
- Conflict = same `(type, key)` invalidated by two clients with
  overlapping `valid_from` → both rows kept; surfaced via existing
  Phase 6a conflict UI
- Pull is cursor-based; client merges by upsert
- Local-only nodes (`scope='local'`) never leave the machine
- Sync triggers: any `hero scan` or `hero handoff` opportunistically
  pushes pending + pulls deltas. No new commands. Offline-friendly.

**Multi-dev wins this enables:**

- Dev A's "tried and failed: bcrypt rounds=12 too slow" visible to
  Dev B before they try the same thing.
- "Who owns this feature? When did they last touch it?" — one query.
- Sprint plan is one shared graph view — no merging individual NEXT.md.
- Onboarding: new dev's `/resume` returns team's current focus
  subgraph, not a stale doc.

## External trackers — Jira first

Jira is the priority external tracker (heavy real-world use; richer
fields than GitHub Issues map better to graph). Auth pattern reuses
`hero.json` config + env-var token, mirroring existing wiki token
resolution. Read-only first; write-back (commit links into Jira
comments) is a follow-up.

**Jira mapping:**

| Jira | Graph |
|---|---|
| Issue | `Issue` node |
| Epic / Story / Subtask | edge type `belongs_to` between `Issue` nodes |
| Sprint | `Sprint` node + `belongs_to` edges from `Issue`s |
| Component | `Component` node (label-like) |
| Custom field | JSON in `props` |
| Status | `props.status` (with valid_from/to invalidation on change) |
| Comment | `Comment` node + `mentioned_in` edge |
| Assignee | `assigned_to → Person` edge |

GitHub Issues support follows the same shape. Linear later.

## Command surface (simplified)

Every command is a verb that does one thing. No sub-verb sprawl.

| Command | Does |
|---|---|
| `hero scan` | Master ingest: code, planning, notes, raw, git, tracker, sync |
| `hero next` | Project current focus → markdown |
| `hero handoff` | Capture session subgraph → portable artifact |
| `hero resume` | Load handoff + relevant subgraph |
| `hero note [slug]` | Hand-authored brainstorm capture (unchanged) |
| `hero ingest <url-or-file>` | External content → raw + Document node |
| `hero why <thing>` | Traverse backward: decisions → discussions → outcomes |
| `hero recall <topic>` | Return relevant subgraph as markdown |
| `hero blocked` | Show blocker chains across open work |
| `hero plan <feature>` | Open or create a plan md (graph-linked) |
| `hero query <…>` | Power-user escape hatch, JSON out |
| `hero graph stats` | Node/edge counts by type, scope |
| `hero graph reingest <subgraph>` | Manual rebuild trigger |

`hero handoff dump/load/status` collapses to just `hero handoff`
(single verb, one thing). Other multi-verb commands re-evaluated
during build.

## Hard problems — decisions captured

| Question | Decision |
|---|---|
| Plans (`spec.md`) ingested into graph? | Yes — Tier 2 extraction with `source=extracted-from-prose`. Spec md remains authoritative; edges are inferential hints. |
| Per-dev local graph vs. team-server first | Local-first; `scope` column in schema from day one. Sync layer week ~7 of plan. |
| External trackers first | Jira (priority); GitHub Issues next; Linear later. |
| Source-of-truth conflicts (user edits NEXT.md) | Projections always win — not merged. Hand-authored content lives in `plans/` or `notes/`. |
| Granularity of `Session` | Per-handoff for now. `ToolCall` as optional finer child. |
| Entity resolution aggressiveness | Propose-and-confirm by default; auto-merge above high-confidence threshold. Drift worse than over-merging. |
| `~/.claude/.../memory/` files | Stay separate (user-scoped graph), join at query time. Memory is personal. |
| LLM choice for extraction | Haiku 4.5 for Tier 1/2; Sonnet 4.6 for Tier 3. |
| `.hero/graph.db` in git? | **TBD** — open question. Current lean: not committed (regenerable cache); team server is the canonical share path. |
| Wiki target | GitHub Pages default; GitHub Wiki + Confluence available. |
| Team server | Reuse Phase 5–7 in-progress server, extend with `/v1/graph/push|pull`. |

## Phased plan (revised — federation pulled forward)

See [graph-memory-federation/spec.md](../graph-memory-federation/spec.md)
for the multi-repo / multi-team / unit-join design that is now
threaded through phases 5/7/8. Schema contracts for federation land
in **3b** so the local graph stays forward-compatible.

| Phase | Goal | Status |
|---|---|---|
| 1 | Schema + graph store + code-subgraph round-trip | ✅ shipped |
| 2 | Work-subgraph ingest (commits, planning md, sessions) | ✅ shipped |
| 3a | Raw docs, NEXT.md attempts, commit issue-refs | ✅ shipped |
| **3b** | **Federation schema contracts** — repo/unit columns, scope levels (local/team/unit/public), alias_of edge type, depth-cap helper | **next** |
| 4 | NEXT.md + sprint todos as projections | |
| 5 | `hero why`, `hero blocked`, `hero impact` (cross-repo from day 1) | |
| 6 | Jira ingest (read-only) | |
| 7 | Team-server sync (per-repo partitioned API) | |
| 8 | Unit-level join layer (cross-repo dogfood) | |
| 9 | GitHub Pages projection | |
| 10 | Tier-2 LLM extraction (notes → Decision/Concept) | |

## Phase 1 — today's scope

Real, shippable, behavior-preserving foundation. Four steps. End-of-day
state: graph exists, holds code subgraph, round-trips bit-identical
markdown, no user-visible change.

**Step 1 — schema + store package** (~1 hr)

- New package `internal/graph/`
- `graph.go`: `Open(path) → *Store`, `UpsertNode(n)`,
  `UpsertEdge(from, type, to, props)`, `Get(type, key)`,
  `Query(...)`, `Invalidate(type, key, validTo)`
- Migrations runner from the SQL above
- Tests: open empty db, upsert + retrieve, invalidate-and-update,
  idempotency, schema version check

**Step 2 — codescan → graph ingest** (~1 hr)

- Extend `internal/codescan/` with `WriteGraph(*Result, *graph.Store)`
- Maps `Result` → `Repo`/`Package`/`File`/`Symbol` nodes +
  `imports`/`defines`/`belongs_to` edges
- Existing `GenerateKnowledge` (md output) keeps running unchanged

**Step 3 — graph → md projection** (~1 hr)

- `internal/codescan/project.go`: read graph, write
  `code/<pkg>/spec.md`
- Verify: scan → graph → project → diff against
  `GenerateKnowledge` output. Goal: byte-identical (modulo timestamps).

**Step 4 — `hero graph` command** (~30 min)

- `hero graph stats` — node/edge counts by type, scope
- `hero graph reingest code` — manual round-trip trigger
- Wire `hero scan` to also call step 2 (additive — no removal yet)

**Definition of done:**

- `.hero/graph.db` exists, populated with full code subgraph
- `hero graph stats` shows counts matching codescan output
- `hero graph reingest code` round-trips with bit-identical md
- `hero scan` populates graph as side effect
- All existing tests pass; one new test file proves round-trip

**Explicitly NOT in phase 1:**

- Work or knowledge subgraphs (phase 2/3)
- Sync (phase 7)
- Any user-visible projection swap (phase 4)
- Jira (phase 6)
- Tier 2/3 extraction (phase 3+)

## Open questions to revisit

1. Commit `.hero/graph.db` to git? (current lean: no)
2. Cross-repo / cross-org memory in v1 sync, or v2?
3. `hero query` CLI surface — power-user escape hatch in v1, or wait?
4. `~/.claude/.../memory/` migration path — stay separate forever, or
   migrate into per-user graph file at some point?

## Success metrics (when phase 8 lands)

- Multi-dev team using shared team graph in production for ≥ 2 weeks
- Median latency for `hero why`, `hero blocked`, `/resume`: < 200 ms
- GitHub Pages site auto-updates on every `hero scan` push
- Zero behavior regressions vs. flat-file hero
- One published case study showing graph-traversal-driven capability
  that flat hero structurally couldn't provide

## Why this is the right move now

Hero already produces graph-shaped data. Storing it flat is the
limiting constraint on every "smart" feature on the roadmap (cross-spec
awareness, drift detection, cross-org intel, dashboard, why/blocked
queries). Replacing the substrate now — additively, with no UX change —
unlocks all of those features as queries instead of bespoke
implementations. Every later phase ships faster because the foundation
is right.
