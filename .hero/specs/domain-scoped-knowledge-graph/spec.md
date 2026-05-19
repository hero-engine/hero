---
title: Domain-Scoped Knowledge Graph — Namespace Tags on Graph Nodes
slug: domain-scoped-knowledge-graph
type: feature
status: completed
priority: P0
tags: [platform, domains, graph, knowledge, namespacing]
created: 2026-05-15
designed: 2026-05-19
relations:
  - target: hero-domains
    kind: parent
depends-on:
  - domain-plugin-architecture
horizon: next
smoke: deferred
received_from:
  peer_id: ad027c2f-7f74-4a09-bf1d-6515cc906074
  peer_alias: hero-code
  related_spec: hero-pm-ui-prework
  call_id: 18b0dc2024afa930345e09a22506e39e
  mode: spec-out
  at: 2026-05-19T04:11:23Z
  reason: |
    Kickoff B1 of the hero-pm-ui-prework initiative.
    domain-scoped-knowledge-graph is the longest pole and gates the brand
    demo. Hero ran /design natively on the existing spec to lock the
    cross-domain edge contract and query-shape audit before C1
    (handoff-coordinator agent) and D1/D2 (handoff-protocol,
    cross-domain-graph-query skills) start.
---

## Kickoff

Add domain namespace tags to every knowledge graph node and edge so that
PM and engineering content can coexist in one workspace without queries
silently mixing them. Tag at write time; filter or render the boundary
at read time. The killer demo this unblocks: `hero-pm-ui`'s **Hand off to
/design** button writes a cross-domain `story → feature` edge whose kind
is `handoff`, and downstream surfaces (`handoff_card.rs`, the Handoff
stream view, the linked-feature card on Story detail) read it back via
cross-domain graph queries.

**Status:** designed — 2026-05-19. Cross-domain edge schema, query-shape
audit, active-domain resolution, and migration plan all locked. Awaiting
`domain-plugin-architecture` cutover for write-side `Config.Domain` to
become non-default; the read-side and schema work can land first.

**Pick up at:** `/deliver domain-scoped-knowledge-graph`. Phase 1
(schema v3 migration) is one PR; Phase 2 (write-path stamping) is a
narrow patch across ingest packages; Phase 3 (read-path filtering) is
the long pole — one helper plus a call-site sweep keyed off the audit
table below.

→ `/deliver domain-scoped-knowledge-graph`

**Files:** .hero/planning/features/domain-scoped-knowledge-graph/spec.md,
internal/graph/graph.go, internal/graph/node.go, internal/graph/edge.go,
internal/traversal/why.go, internal/retrieval/retrieval.go,
internal/cli/brief.go, internal/serve/mcp_tools.go,
internal/handoff/handoff.go, internal/spec/graph_ingest.go,
internal/tracker/graph_ingest.go, internal/sessions/graph_ingest.go,
internal/codescan/graph_ingest.go, internal/memory/graph_ingest.go,
internal/nextdoc/graph_ingest.go, internal/knowledge/graph_ingest.go,
internal/gitutil/graph_ingest.go, internal/mission/mission.go,
internal/extract/decisions.go, internal/tasks/record.go,
internal/acceptance/record.go, internal/config/config.go.

**Skip:** Full multi-active-domain workspaces in v1 — the substrate
supports it, but `hero domain switch` remains a re-install rather than a
per-query domain swap. Third-party domain packs loaded from disk
(deferred). Cross-domain reporting / combined dashboards (separate spec
after PM ships).

## Goal

Add a `domain` namespace tag to every node and edge in the knowledge
graph. Teach every query path that consumes the graph (`hero why`,
`hero blocked`, `hero search`, `hero relevant`/`hero ask`, dashboard
widgets, MCP tools, sub-system queries inside `hero recap`, `hero pulse`,
etc.) to either filter by the active domain (default) or render
cross-domain boundaries explicitly.

V1 ships with single-active-domain workspaces — namespace tags are
present, every query path has an explicit stance, and the foundation is
in place so multi-active-domain workspaces can land later without a
graph re-index.

## Why now

The parent initiative makes a deliberate call: PM can ship in
single-domain mode without this, but adding namespace tags later forces
a re-index against a live graph. Even in single-domain v1, every graph
query path must tolerate a namespace tag — silently mixing domains in
shared queries is worse than blocking on the work upfront.

The cross-cutting risk in the parent initiative: queries (`hero why`,
`hero blocked`, `hero search`) currently operate on a flat namespace.
PM stories pointing at engineering features cross a domain boundary;
queries need to either filter by active domain (default) or render the
boundary explicitly (cross-domain traversal opt-in). `hero-pm-ui`'s
brand demo depends on this contract — handoff writes need to be visible
across both surfaces from day one.

P0 priority because:

1. `hero-pm-ui` Risk #3 (the cross-domain read audit) blocks on this
   spec landing the per-query-path stance enumeration.
2. The `hero-pm` content pack is already authored; the only thing that
   keeps PM stories from leaking into engineering queries is this work.
3. Adding tags later means a re-index against potentially large local
   graphs (tens of thousands of nodes after a `hero scan deep`).

## Design

### Schema change

Add `domain TEXT NOT NULL DEFAULT 'engineering'` to both `nodes` and
`edges` tables. Index on `domain` for fast partition filtering — the
same pattern that v2 used for the federation `repo`/`unit` columns.

Schema v3 migration (one new entry in `internal/graph/graph.go`
migrations slice):

```sql
ALTER TABLE nodes ADD COLUMN domain TEXT NOT NULL DEFAULT 'engineering';
ALTER TABLE edges ADD COLUMN domain TEXT NOT NULL DEFAULT 'engineering';
CREATE INDEX IF NOT EXISTS idx_nodes_domain ON nodes(domain);
CREATE INDEX IF NOT EXISTS idx_edges_domain ON edges(domain);
```

The `DEFAULT 'engineering'` clause does the row-by-row backfill in
place at ALTER time — no separate one-shot pass, no opportunistic
read-time backfill. SQLite's `ALTER TABLE ADD COLUMN ... DEFAULT <const>`
is O(1) (metadata-only); rows render the default at read time. This
makes the migration invisible to engineering-only users.

Why a literal `'engineering'` default rather than `NULL` or `''`:

- A literal string keeps every existing row trivially queryable as if it
  had always been tagged. `WHERE domain = ?` works from day one.
- A `NULL` sentinel forces every read path to write `COALESCE(domain,
  'engineering')` or risk silent boundary leaks. The cost is real and
  the bug class is dangerous.
- The empty string `''` is reserved for an explicit **workspace-wide /
  global** sentinel (Mission, Person — see write-path rules below).
  Making the new column default to `''` and treating `''` as both
  "untagged" and "global" would erase the very distinction we need.

### Node and edge struct fields

`internal/graph/node.go` `Node` struct adds:

```go
Domain string `json:"domain,omitempty"` // namespace tag — empty = global
```

`internal/graph/edge.go` `Edge` struct adds the same field.

Both `UpsertNode` and `UpsertEdge` thread the field through to the
column. The existing partition-unchanged short-circuit in `UpsertNode`
(node.go:98) and `UpsertEdge` (edge.go:79) is extended:

```go
partitionUnchanged := existingRepo == n.Repo &&
                      existingUnit == n.Unit &&
                      existingDomain == n.Domain
```

— so a domain change forces an invalidate-and-reassert, the same way
`repo`/`unit` changes do today. This gives bitemporal correctness for
"this node moved domains on date X" with zero new machinery.

### Cross-domain edge semantics

**Domain is computed, not declared.** An edge crosses a domain boundary
when `from-node.domain != to-node.domain`. The edge row's own
`domain` column is the partition tag — set at write time to the
**from-node's domain** — and exists strictly for fast single-domain
filtering. The boundary itself is a property of the endpoints.

This design means:

1. We do **not** introduce a `cross-domain-handoff` edge kind. Existing
   edge kinds keep their semantics; a `handoff` edge is a `handoff`
   regardless of whether the endpoints share a domain.
2. We do **not** mutate edge kind based on endpoint domains. The kind
   is the relationship; the boundary is a render concern.
3. Cross-domain queries are answered by JOINing both endpoint nodes and
   comparing their domains — cheap because the column is indexed on
   both ends.

**New v1 edge kinds explicitly allowed to cross domains:**

| Kind | From → To | Use |
|---|---|---|
| `handoff` | `Story` (pm) → `Feature` (engineering) | The killer demo. `/design` on a PM story creates an engineering feature and writes this edge. |
| `derived_from` | `Feature` (engineering) → `Story` (pm) | Engineering feature derived from a PM story — the reverse pointer for the same handoff link. |
| `realizes` | `Feature` (engineering) → `PRD` (pm) | Engineering feature realizes a PM PRD/initiative without a story-level handoff. |

These are the v1 set. Anything beyond — e.g. PM `intake → engineering
Bug`, PM `Epic → engineering Component` — is treated as a *valid but
unusual* cross-domain edge: it writes, but `hero warnings` surfaces
`cross_domain_unusual_kind` so the contract evolves visibly rather than
silently. Adding a new sanctioned kind is a one-line append to the
`crossDomainAllowedKinds` set in `internal/graph/edge.go`.

**Write-time invariants:**

1. `UpsertNode` rejects `domain == ""` with `ErrDomainRequired` **unless**
   the node type is in the global allow-list:
   - `Mission` (workspace-wide brief, no domain)
   - `Person` (people are shared across domains in v1)
   - `Org`, `Repo`, `Unit` (federation partitions, not domain partitions)
   The allow-list lives in `internal/graph/node.go` as
   `globalNodeTypes`.
2. `UpsertEdge` accepts an explicit `Domain` field. When unset, it
   inherits from the **from-node's** domain via a lookup at write time.
   If the from-node is a global type (`Domain == ""`), the edge writer
   MUST pass `Domain` explicitly or the call returns
   `ErrEdgeDomainRequired`. This catches the "Mission has an edge into
   PM, but the edge is now silently global" trap.
3. `UpsertNode` rejects a **domain change** on the current row — i.e.
   you cannot upsert a node with the same `(type, key)` and a different
   `domain`. The first write wins; relocating a node across domains is
   a v2 migration concern and gets `ErrDomainMutation` for now. This
   prevents content drift from silently retagging upstream specs.

### Active-domain resolution

The "active domain" at query time is resolved through a fixed
precedence chain:

```
1. Explicit override (highest priority)
   - CLI: --domain <name>   or  --all-domains
   - MCP: { "domain": "<name>" }  or  { "domain": "*" }
   - Dashboard: ?domain=<name>  query parameter
   The literal value "*" means "include all domains and render boundaries".
2. Workspace config
   - Config.Domain (from hero.json — the field exists today;
     internal/config/config.go:28)
3. Hardcoded fallback
   - "engineering"
   (Covers pre-migration workspaces with no `domain` key in hero.json.)
```

Resolved once per CLI invocation / per MCP request / per dashboard
request — never per-node, never per-query within a single call. The
helper lives at `internal/graph/scope.go` (new file):

```go
// DomainScope captures how a single CLI/MCP/dashboard call wants to
// interact with the domain partition.
type DomainScope struct {
    Active string // resolved active domain; "" only for global reads
    AllDomains bool // true when --all-domains / domain="*" was passed
}

// ResolveDomain returns the effective scope for one call. cfg.Domain is
// the workspace default; override is the explicit flag/argument (empty
// string = no override).
func ResolveDomain(cfg config.Config, override string) DomainScope { ... }

// Where returns a SQL WHERE clause fragment and arg list suitable for
// appending into existing queries on `nodes` and `edges`. Empty
// fragment when AllDomains is true.
func (d DomainScope) Where(tableAlias string) (string, []any) { ... }
```

Every read-path call site in the audit below either calls `Where()` or
explicitly opts out (e.g. `hero conflicts`, which must surface
cross-domain conflicts regardless).

### Query-shape audit — every read path that touches the graph

This is the contract D2 (cross-domain-graph-query skill in
`hero-pm-ui-prework`) consumes. It enumerates every existing query path
with its v1 stance. **Boundary-aware** means the query intentionally
crosses domains; **filtered** means the query scopes to the active
domain by default and accepts an opt-in widening flag; **single-target**
means the query resolves one node by slug and has no scope concern.

#### CLI / MCP tool query paths

| Query | Today | v1 stance | Override |
|---|---|---|---|
| `hero why <target>` / `hero_why` (`internal/traversal/why.go`) | Resolves target across repo, walks origin edges (belongs_to, satisfied_by, attempted_in, decided_in, supersedes, mentions, depends_on, derived_from, originated_in, closes, fixes) up to maxDepth. Already scope-aware via `MarkdownScoped`. | **Boundary-aware by default.** Why is the showcase traversal — a PM story's `why` trace must include the engineering feature it handed off to (and vice versa). Render boundary inline: `← _handoff (cross-domain pm → engineering)_`. The existing `[scope: ...]` marker is extended with `[domain: pm]`. Add `handoff`, `realizes` to `originEdgeTypes`. | `--domain <name>` filters to a single domain. |
| `hero blocked` / `hero_blocked` (`internal/cli/brief.go:394`) | Joins Features in current repo to incomplete dependencies via `depends_on`/`blocks` edges. Already filters `f.repo = ?`. | **Filtered by default.** PM features blocked on PM dependencies and engineering features blocked on engineering decisions are separate views. Filter `f.domain = ? AND b.domain = ?`. | `--all-domains` opt-in surfaces cross-domain blockers (engineering feature blocked on PM PRD). |
| `hero search <query>` / `hero_search` (`internal/retrieval/retrieval.go`) | FTS5 spec corpus path + graph node-key match path, ranked by type boosts. No domain awareness today. | **Filtered by default; cross-domain de-boosted.** In-domain results render at full score; cross-domain results render below with a `[domain: <name>]` snippet tag at a 0.5× score multiplier. | `--all-domains` removes the de-boost; the partition is left intact in the result rows. |
| `hero list <type>` / `hero_list` (`internal/cli/list.go`) | Spec list query (disk-driven, graph-mirrored). | **Filtered by default.** PM stories don't appear in an `engineering`-active list. | `--all-domains` surfaces both. |
| `hero queue` / `hero_queue` | Ranked ready-to-work specs (curated `hero list` with ready=true, sort=priority, format=kickoff). | **Filtered by default.** A PM user's queue must not surface engineering features. | `--all-domains` for cross-domain readiness rollup (oncall ops use case). |
| `hero kickoff <slug>` / `hero_kickoff` | Single-spec kickoff prompt render. | **Single-target.** Slug resolves uniquely; the node's domain is implicit in the resolved row. No filter needed. | — |
| `hero status` / `hero_status` | Workspace summary. | **Filtered by default.** PM and engineering status render separately. | `--all-domains` for unified rollup. |
| `hero recap` / `hero_recap` | Recent activity summary. | **Filtered by default.** Activity in the active domain only. | `--all-domains`. |
| `hero feed` / `hero_feed` | Cross-agent activity stream. | **Filtered by default.** A PM agent's feed must not see engineering Stop-hook checkpoints. | `--all-domains` for admin views. |
| `hero active` / `hero_active` | Active session list. | **Filtered by default.** Sessions stamp domain at start. | `--all-domains`. |
| `hero context` / `hero_context` | Auto-prime context payload. | **Filtered by default.** Domain hints tell the model which agents/skills/conventions to weight. | `--all-domains` (admin profile only). |
| `hero pulse` / `hero_pulse` | Sprint health rollup. | **Filtered by default.** A sprint is per-domain. | `--all-domains`. |
| `hero plan` / `hero_plan` | Read/write plan.md for a spec. | **Single-target.** | — |
| `hero impact` / `hero_impact` | Impact analysis for a change. | **Boundary-aware by default.** Impact must surface PM stories affected by an engineering change (and vice versa) or the link silently breaks. Render boundary in output. | `--domain <name>` narrows. |
| `hero contract` / `hero_contract` | Contract surface check. | **Filtered by default.** Contracts are inside-domain by convention. | `--all-domains`. |
| `hero coverage` / `hero_coverage` | AC coverage rollup. | **Filtered by default.** ACs belong to single-domain specs. | `--all-domains`. |
| `hero ci` / `hero_ci` | CI status surfacing. | **Filtered by default.** CI is engineering today; PM workspaces return empty unless widened. | `--all-domains`. |
| `hero drift` / `hero_drift` | Architecture drift detection. | **Filtered by default.** Drift is structural-within-domain. Cross-domain drift is a future concern. | `--all-domains`. |
| `hero score` / `hero_score` | Spec quality score. | **Single-target.** | — |
| `hero code` / `hero_code` | Code intelligence query (symbols, packages, hot files). | **Filtered by default.** Code symbols stamp engineering; a PM workspace returns empty unless widened. | `--all-domains`. |
| `hero conflicts` / `hero_conflicts` | Concurrent-write conflict detection. | **Boundary-aware (always).** Conflicts are storage-level and must surface regardless of domain. No filter applied. | — |
| `hero sequence` / `hero_sequence` | Spec sequencing analysis. | **Filtered by default.** | `--all-domains`. |
| `hero warnings` / `hero_warnings` | Workspace warnings (including the new `cross_domain_unusual_kind` warning). | **Boundary-aware (always).** All warnings surface; the `domain` column is rendered alongside. | — |
| `hero insights` / `hero_insights` | Insights analysis. | **Filtered by default.** | `--all-domains`. |
| `hero ask` / `hero_ask` | Knowledge synthesis. | **Filtered by default; cross-domain de-boosted.** Same rule as `hero search`. | `--all-domains`. |
| `hero anchor` / `hero_anchor` | Anchor / pin a spec. | **Single-target.** | — |
| `hero diagnose` / `hero_diagnose` | Run diagnosis. | **Single-target (writes a new spec).** New spec stamps active domain. | — |
| `hero verify` / `hero_verify` | Verify a spec's claim. | **Single-target.** | — |
| `hero velocity` / `hero_velocity` | Velocity rollup. | **Filtered by default.** Velocity is per-domain. | `--all-domains`. |
| `hero error_pattern` / `hero_error_pattern` | Error pattern analysis. | **Filtered by default.** | `--all-domains`. |
| `hero check` / `hero_check` | Health check. | **Filtered by default.** Multi-domain workspaces aren't v1 — but `--all-domains` is the v2 seam. | `--all-domains`. |
| `hero knowledge` / `hero_knowledge` | Knowledge entry retrieval. | **Filtered by default; cross-domain de-boosted.** | `--all-domains`. |
| `hero read_spec` / `hero_read_spec` | Read a spec by slug. | **Single-target.** | — |
| `hero snapshot` / `hero_snapshot` | Project shape rollup. | **Filtered by default.** Snapshot is per-domain. | `--all-domains` for unified rollup. |
| `hero nudge` / `hero_nudge` | Surface nudges. | **Filtered by default.** | `--all-domains`. |
| `hero expand` / `hero_expand` | Expand a referenced node. | **Single-target.** | — |
| `hero claim` / `hero_claim` | Claim a spec. | **Single-target (write).** Claim row inherits target spec's domain. | — |
| `hero event` / `hero_event` | Record an event. | **Write-only.** Event stamps active domain. | — |
| `hero skill_run` / `hero_skill_run` | Skill workflow preview/run. | **Single-target.** Skill node's domain is implicit; widens at run time if the skill writes cross-domain. | — |
| `hero test_generate` / `hero_test_generate` | Generate tests for a spec. | **Single-target.** | — |
| `hero demo_record` / `hero_demo_record` | Record a demo for a spec. | **Single-target.** | — |
| `hero enrich` / `hero_enrich` | Write code-symbol descriptions. | **Stamps engineering** (code is engineering-only). No read filter. | — |

#### Sub-system query paths inside packages

| Path | Today | v1 stance |
|---|---|---|
| `internal/handoff/handoff.go` — `LatestAsk`, `LatestSuggestion`, `RecentReflections` | Filters by `(user, repo)`. Singleton key is `user`. | **Add domain filter.** Singleton key becomes `user:<domain>`; per-(user, repo, domain) singleton. `LatestAsk(store, user, repoKey, domain)`. Engineering-only workspaces see no behavior change because domain resolves to `engineering` and the key just gains a suffix. |
| `internal/traversal/why.go` — `walkOrigins` recursive CTE | No domain filter. | Add optional `WHERE n.domain = ?` join on the traversed nodes (when DomainScope is filtered). Edge inclusion follows endpoint domains, not the edge's own column, to keep boundary rendering correct. |
| `internal/retrieval/retrieval.go` — graph node-key path + FTS5 path | No domain awareness. | Two paths:<br>• Graph node-key path: add `WHERE n.domain = ?` (or skip when AllDomains).<br>• FTS5 path: spec corpus rows already carry `domain` (sourced from frontmatter at index time — see Phase 2 below); apply the same filter at query time. Cross-domain results get a 0.5× score multiplier in the unified ranker. |
| `internal/extract/decisions.go` — decision extraction over commits | No domain filter. | Stamp decisions with the active domain at write time; read paths inherit domain from the parent spec. |
| `internal/acceptance/query.go` — AC participation join | No domain filter. | Add `WHERE feature.domain = ?` on the parent-spec join. Cross-domain ACs are not a v1 concept. |
| `internal/sessions/graph_ingest.go` — Session, Commit, Person ingest | No domain stamp. | Sessions stamp active domain at start. Commits stamp `engineering` (code is engineering). Persons stamp `""` (global). |
| `internal/spec/graph_ingest.go` — `WriteGraph(specs, repoKey, store)` | Threads `repoKey` already. | Add `domain string` parameter, threaded into every UpsertNode/UpsertEdge call. The spec's domain comes from its frontmatter `domain:` field (added by `/design`) or from `Config.Domain` as the fallback. |
| `internal/tracker/graph_ingest.go` — tracker issue ingest | No domain stamp. | Stamps active domain. A PM-active workspace pulling Jira stamps `pm`; an engineering-active workspace pulling GitHub stamps `engineering`. Cross-domain tracker integration (one workspace, two trackers, two domains) is out of v1 scope. |
| `internal/codescan/graph_ingest.go` — code symbol ingest | No domain stamp. | **Always stamps `engineering`.** Code is intrinsically engineering. |
| `internal/memory/graph_ingest.go` — memory ingest | No domain stamp. | Stamps active domain. |
| `internal/nextdoc/graph_ingest.go` — NEXT.md projection | No domain stamp. | Stamps active domain. |
| `internal/knowledge/graph_ingest.go` — knowledge ingest (notes, conventions, decisions) | No domain stamp. | Stamps active domain. Engineering conventions don't appear in a PM workspace's `--list` unless widened. |
| `internal/gitutil/graph_ingest.go` — git ingest | No domain stamp. | **Always stamps `engineering`** (Commits + Branches). |
| `internal/mission/mission.go` — Mission node | No domain stamp. | **Stamps `""` (global).** Mission is workspace-wide. |
| `internal/tasks/record.go` — task records | No domain stamp. | Stamps active domain. |

#### Dashboard widget query paths (read-side)

| Widget | Today | v1 stance |
|---|---|---|
| Specs kanban | Lists current specs from graph. | **Filtered to active domain (router-resolved).** Per-page override via `?domain=` query param. |
| Drift report | Drift rollup. | **Filtered to active domain.** |
| CI status | CI surface. | **Filtered to active domain** (returns empty for PM workspaces unless widened). |
| Velocity | Velocity chart. | **Filtered to active domain.** |
| Handoff stream (the brand-demo target) | Cross-domain handoff queue — reads `handoff` edges where `from.domain != to.domain`. | **Boundary-aware (always).** This widget exists to render the boundary. Always queries with `AllDomains = true` and JOINs both endpoint domains for the boundary label. |
| Roadmap (PM landing) | PM roadmap with linked engineering features. | **Boundary-aware.** PM roadmap shows engineering features it depends on; rendered with the boundary tag. |
| Story queue (PM) | PM story list. | **Filtered to `pm`.** |
| Linked-feature card on Story detail | Reads `handoff` edge from current story → feature. | **Boundary-aware (single-edge read).** Query the `handoff` edge by kind; resolve target node across domains; render its domain in the card. Pattern documented for D2 to consume. |
| Intake funnel (PM) | PM intake board. | **Filtered to `pm`.** |
| Knowledge browser | Notes/conventions/decisions. | **Filtered to active domain; cross-domain de-boosted.** Mirrors `hero ask` / `hero knowledge`. |

This list is the v1 contract for `D2 — cross-domain-graph-query skill`.
Any read site not listed here is a defect; if a downstream change adds
a new query path, it MUST extend this table.

### Migration plan

#### Phase 1 — Schema v3 (one PR)

`internal/graph/graph.go` migrations slice gains:

```go
{
    version: "3",
    statements: []string{
        `ALTER TABLE nodes ADD COLUMN domain TEXT NOT NULL DEFAULT 'engineering'`,
        `ALTER TABLE edges ADD COLUMN domain TEXT NOT NULL DEFAULT 'engineering'`,
        `CREATE INDEX IF NOT EXISTS idx_nodes_domain ON nodes(domain)`,
        `CREATE INDEX IF NOT EXISTS idx_edges_domain ON edges(domain)`,
    },
},
```

The existing migration runner is idempotent and version-bumps `meta.schema_version`. Bump the binary's `schemaVersion` constant from `"2"` to `"3"` in the same PR.

**Bitemporal safety:** ALTER ADD COLUMN with a constant DEFAULT does not invalidate any historical rows. Bitemporal time-travel queries (`GetNodeAt`) keep working unchanged — every prior row simply gets `domain = 'engineering'` from the default clause.

**Dry-run:** `hero admin schema dry-run` (new subcommand under existing `hero admin` family) prints the SQL it would execute. Implementation is a no-op wrapper around the migrations slice.

**Verification:** `hero domain verify` (new) reports:
- Node count by `(type, domain)`.
- Edge count by `(type, domain)`.
- Cross-domain edge count grouped by `(from-domain, to-domain, edge-kind)`.
- Warnings: nodes with `domain = ''` whose type is not in `globalNodeTypes`.

Run pre- and post-migration; the post-migration node count grouped by domain must equal the pre-migration total under `engineering` plus zero rows under any other domain. Any deviation is a migration bug.

#### Phase 2 — Write-path stamping (incremental PRs)

Each graph-ingest package gets a one-line stamp added to its
`UpsertNode`/`UpsertEdge` call sites. The package's public entrypoint
gains a `domain string` parameter (or reads it from `Config.Domain`
directly when the caller already passes `cfg`).

Helper: `internal/graph/stamp.go` (new) exposes
`DomainFor(cfg config.Config, hint NodeHint) string` returning:

- `"engineering"` for `NodeHint{ Intrinsic: "code" }` (codescan, gitutil)
- `""` for `NodeHint{ Intrinsic: "global" }` (mission, person)
- `cfg.Domain` (with `"engineering"` fallback) otherwise

The hint is a per-package constant chosen by the ingest implementation. This keeps the rule local to each package and prevents a future ingest path from forgetting to stamp.

**Lint:** add a CI check that fails when a new `graph.UpsertNode`/`UpsertEdge` call is added without setting `Domain` (regex check on the diff is sufficient; the call sites are bounded).

#### Phase 3 — Read-path filtering (long pole; ~30 call sites)

For each query path in the audit table:

1. Plumb `DomainScope` from the entrypoint (CLI flag parsing or MCP args) down to the graph query.
2. Replace the existing `WHERE n.repo = ?` (or equivalent) with `WHERE n.repo = ? AND ` + `scope.Where("n")`.
3. For boundary-aware queries, JOIN both endpoint nodes and render boundary inline.

Order of work: `hero why` and `hero blocked` first (the showcase paths), then `hero search` / `hero ask` (the retrieval paths), then dashboard widgets, then the long tail of single-target tools (mostly no-op for those).

#### Phase 4 — Spec frontmatter `domain:` field

`/design` (the command) starts writing `domain: <active-domain>` into spec frontmatter. The spec loader reads it and passes it into `WriteGraph`. The loader's fallback is the workspace `Config.Domain` for legacy specs that have no `domain:` field — guaranteeing that pre-migration specs all resolve to `engineering`.

The `domain:` field is also surfaced in `hero search --list` output so the user can tell at a glance which domain a result belongs to.

#### Rollback story

Two rollback layers:

1. **Binary rollback.** Revert the binary to a v2-aware build. The v3 database fails the existing `schema_version mismatch` check (`graph.go:230`), printing `graph schema version mismatch: db=3 binary=2`. This is the established federation cutover error model — operator sees the mismatch immediately, runs `hero admin schema rollback v3`.
2. **Schema rollback.** `hero admin schema rollback v3` (new) executes:
   ```sql
   DROP INDEX IF EXISTS idx_nodes_domain;
   DROP INDEX IF EXISTS idx_edges_domain;
   ALTER TABLE nodes DROP COLUMN domain;
   ALTER TABLE edges DROP COLUMN domain;
   UPDATE meta SET value = '2' WHERE key = 'schema_version';
   ```
   SQLite supports `DROP COLUMN` as of 3.35 (released 2021) — well below our minimum. The drop is non-data-corrupting: every row's other columns are preserved, and the column being dropped only carried a domain tag, not load-bearing relationship data.

Rollback safety: no row migrations, no JSON props edits, no edge rewrites. The only side effect is that any post-migration writes that stamped a non-engineering domain (PM, future packs) lose that tag. A `hero admin schema rollback v3 --dry-run` prints the count of non-engineering rows that would lose their tag, letting the operator decide.

### Backwards compatibility for MCP clients

Every MCP tool that previously had no `domain` argument gains an optional `domain` string field whose absence resolves to active domain (which is `engineering` for legacy workspaces). Net = no client breakage. The MCP tool definitions (`internal/serve/mcp_tools_def.go`) gain one entry per affected tool documenting the new field as `optional, defaults to workspace active domain. Pass "*" to include all domains.`.

External MCP clients that don't know about domains see today's behavior unchanged — engineering content only, just like today.

## Acceptance Criteria

- WHEN `hero` opens a v2 graph database THE SYSTEM SHALL apply schema v3 idempotently, defaulting every existing node and edge to `domain = 'engineering'`.
- WHEN a node is upserted without a `Domain` field set AND the node type is not in `globalNodeTypes` THE SYSTEM SHALL return `ErrDomainRequired`.
- WHEN an edge is upserted with both endpoints in the same domain THE SYSTEM SHALL stamp `edge.domain` to the endpoint domain.
- WHEN an edge is upserted with endpoints in different domains AND the edge kind is in `crossDomainAllowedKinds` THE SYSTEM SHALL write the edge and stamp `edge.domain` to the from-node's domain.
- WHEN an edge is upserted with endpoints in different domains AND the edge kind is NOT in `crossDomainAllowedKinds` THE SYSTEM SHALL write the edge AND surface a `cross_domain_unusual_kind` warning via `hero warnings`.
- WHEN `hero blocked` runs without `--all-domains` THE SYSTEM SHALL return only Features in the active domain whose blockers are also in the active domain.
- WHEN `hero blocked --all-domains` runs THE SYSTEM SHALL return Features in any domain and surface cross-domain blockers with a `[domain: <name>]` tag on the blocker line.
- WHEN `hero why <story-slug>` traces a PM story that handed off to an engineering feature THE SYSTEM SHALL include the engineering feature in the trace AND render the boundary as `← _handoff (cross-domain pm → engineering)_`.
- WHEN `hero search <query>` runs in a `pm`-active workspace without `--all-domains` THE SYSTEM SHALL return PM results at full score AND any matching engineering results at 0.5× score with a `[domain: engineering]` tag in the snippet.
- WHEN `hero search <query> --all-domains` runs THE SYSTEM SHALL return results from all domains at full score with domain tags in the snippet.
- WHEN the Handoff stream dashboard widget renders THE SYSTEM SHALL query all `handoff`-kind edges regardless of active domain AND label each card with both endpoint domains.
- WHEN `hero next ask` writes a UserAsk in a `pm`-active workspace AND another agent runs `hero next ask` in the same workspace with `engineering` active THE SYSTEM SHALL store two distinct singleton rows keyed `(user, repo, pm)` and `(user, repo, engineering)`.
- WHEN a `Mission` node is upserted THE SYSTEM SHALL accept `Domain == ""` and write the row.
- WHEN `hero admin schema rollback v3 --dry-run` runs THE SYSTEM SHALL print the count of nodes and edges currently tagged with a non-engineering domain.
- WHEN `hero domain verify` runs post-migration on a v2-upgraded database THE SYSTEM SHALL report all nodes under `domain = 'engineering'` with zero nodes in any other domain.

## Boundaries

- **Not** shipping multi-active-domain workspaces. Single-active v1 — `hero domain switch` remains a re-install, not a per-query domain swap.
- **Not** designing the PM domain pack's specific node types — that's `hero-pm` plus whatever `scan-pluggability` decides about scan output schema.
- **Not** introducing per-user or per-team namespacing — that's the `cloud-admin` initiative.
- **Not** changing graph storage backend.
- **Not** designing cross-domain reporting / combined dashboards — separate spec after PM ships.
- **Not** introducing third-party domain packs loaded from disk.
- **Not** renaming any existing graph entity types (Feature stays Feature; the `domain` tag is additive).
- **Not** redefining `repo`/`unit` partition columns — they coexist with `domain`. Domain is a content-vertical partition; repo/unit are federation partitions; they are orthogonal.

## Risks

1. **Query audit underestimates surface.** Many code paths read the graph; missing one means that path silently leaks cross-domain data. Mitigation: the audit table in this spec is now the v1 contract for D2 (cross-domain-graph-query skill); CI lint rejects new graph reads that don't pass a `DomainScope`. Treat the audit as the first deliverable.
2. **Edge inherit-from-source is wrong when from-node is global.** Mission/Person carry `domain = ""`; their outgoing edges would default to `""` and silently bypass the partition. Mitigation: `UpsertEdge` rejects an edge whose from-node is global unless `Edge.Domain` is set explicitly. Catches the trap at the write site.
3. **Cross-domain reads silently widen scope when active domain resolves to empty.** A workspace with a corrupted `hero.json` could resolve active domain to `""`, and the filter would treat that as workspace-wide. Mitigation: `ResolveDomain` enforces the `"engineering"` fallback at config-load time, not query time. Empty active domain is never a valid query state — it's only a valid stored value for the global-types allow-list.
4. **Default-filter surprise.** A user running `hero search` in a `pm`-active workspace and expecting engineering results will be surprised when results are scoped. Mitigation: the de-boost-with-tag approach for `hero search` / `hero ask` / `hero knowledge` keeps cross-domain results visible at lower rank with an inline `[domain: ...]` tag rather than silently hiding them. Help text and dashboard search affordance explicitly mention active-domain scoping.
5. **Handoff stream widget assumes `handoff` edge kind exists end-to-end.** Mitigation: register `handoff` in `originEdgeTypes` (`internal/traversal/why.go:51`), in `crossDomainAllowedKinds`, and in the dashboard widget registry as part of Phase 3. D1 (handoff-protocol skill in hero-pm-ui-prework) consumes the same edge-kind constant.
6. **MCP server's active domain is process-launch-time, not request-time.** A multi-tenant `hero serve` with one process per workspace is fine; running one process serving multiple workspace clients is not v1-supported. Mitigation: documented as a v1 boundary; the per-request `domain` override exists for future multi-tenant use.
7. **Code-scan ingest writing engineering tags into a PM-active workspace.** A user in a PM workspace running `hero scan` would write code symbols tagged `engineering`, then their `hero search` would filter them out by default. Mitigation: PM-active workspaces typically don't run `hero scan`; if they do, the tag is correct (code IS engineering content); `--all-domains` widens the view; documentation in `domain-plugin-architecture` notes the cross-domain code-symbol behavior.
8. **Spec frontmatter `domain:` field drift.** A user can hand-edit `domain:` in spec frontmatter. If the value changes from `engineering` to `pm` after the node has been graph-ingested, the next ingest hits `ErrDomainMutation`. Mitigation: `/deliver` and `/diagnose` warn loudly when reading a spec whose frontmatter `domain:` differs from the existing graph node's `domain`. `hero check` flags the drift. Manual remediation is `hero admin domain retag <slug> --to <name>` (deferred to v2).
9. **Federation sync (repo/unit partitions) and domain partitioning interact.** When a node syncs to Hero Cloud, its `domain` tag travels with it. Cross-org domain conventions ("our PM is your engineering") are deferred. Mitigation: documented; Cloud sync filters by `(repo, unit)` today and ignores domain; cross-domain sync rules are a future cloud-admin concern.
10. **D2 (cross-domain-graph-query skill) over-reads the audit table.** If D2 freezes the v1 stance literally, future per-query stance changes break the skill. Mitigation: D2 reads the audit as guidance, not a frozen contract; the table here is the v1 floor, not a ceiling. New widening flags or boundary-aware queries can be added in follow-ups without re-writing the skill.

## Resolved open questions

1. **Edge semantics for cross-domain handoff.** Resolved: the boundary is computed (`from.domain != to.domain`), not declared via a new edge kind. Existing edge kinds keep their semantics. `handoff`, `derived_from`, and `realizes` are explicitly registered as allowed-cross-domain edge kinds in v1; other kinds cross with a `cross_domain_unusual_kind` warning.
2. **Query default — filter or include.** Resolved per-query in the audit table above. Default is **filter to active domain** for most queries; **boundary-aware** for the showcase traversals (`hero why`, `hero impact`, `hero conflicts`, `hero warnings`) and the dashboard widgets whose job is to render the boundary (Handoff stream, Roadmap, Story-detail linked-feature card). De-boost-with-tag for the retrieval paths (`hero search`, `hero ask`, `hero knowledge`).
3. **Active domain at query time vs at write time.** Resolved: write time stamps from `Config.Domain` (or intrinsic for code/global types); read time resolves through the explicit precedence chain (override → `Config.Domain` → `engineering` fallback). Per-call override-capable for dashboard router and admin tooling; no env var, no per-user override in v1.
4. **Migration of the graph store.** Resolved: real schema migration to v3 — `ALTER TABLE ADD COLUMN ... DEFAULT 'engineering'`. Not opportunistic. Default clause does the backfill in place at migration time, making it invisible to engineering-only users. Reversible via `hero admin schema rollback v3`.
5. **Backwards compatibility for MCP tools.** Resolved: every affected MCP tool gains an optional `domain` argument whose absence resolves to active domain. Pass `"*"` to include all domains. External MCP clients without domain awareness see today's behavior — engineering content scoped to the active workspace.

## Touchpoints

- `internal/graph/graph.go` — schema v3 migration
- `internal/graph/node.go` — Node struct, UpsertNode, partition compare, globalNodeTypes
- `internal/graph/edge.go` — Edge struct, UpsertEdge, crossDomainAllowedKinds
- `internal/graph/scope.go` (new) — DomainScope, ResolveDomain, Where
- `internal/graph/stamp.go` (new) — DomainFor write-side helper
- `internal/config/config.go` — already has `Domain` field; no change required
- `internal/traversal/why.go` — boundary rendering, originEdgeTypes additions
- `internal/cli/brief.go` — `hero blocked` filter join
- `internal/retrieval/retrieval.go` — domain filter on both routing paths, cross-domain de-boost
- `internal/serve/mcp_tools.go` — every read tool plumbs DomainScope
- `internal/serve/mcp_tools_def.go` — document the optional `domain` arg
- `internal/handoff/handoff.go` — domain in singleton key
- `internal/spec/graph_ingest.go` — `WriteGraph(..., domain string)`
- `internal/tracker/graph_ingest.go` — domain stamp
- `internal/sessions/graph_ingest.go` — domain stamp (sessions=active, commits=engineering, persons=global)
- `internal/codescan/graph_ingest.go` — stamp engineering
- `internal/memory/graph_ingest.go`, `internal/nextdoc/graph_ingest.go`, `internal/knowledge/graph_ingest.go` — stamp active
- `internal/gitutil/graph_ingest.go` — stamp engineering
- `internal/mission/mission.go` — stamp `""` (global)
- `internal/extract/decisions.go` — inherit domain from parent spec
- `internal/tasks/record.go` — stamp active
- `internal/acceptance/record.go`, `internal/acceptance/query.go` — domain filter via parent-spec join
- `commands/design.md`, `commands/diagnose.md` — emit `domain:` into new-spec frontmatter
- `internal/cli/admin*.go` (new subcommands) — `hero admin schema rollback v3`, `hero domain verify`, `hero admin domain retag` (v2)
