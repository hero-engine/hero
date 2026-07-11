---
title: "Synthesis Maintenance — Write-Through Coherence for the Hero Graph"
slug: synthesis-maintenance
type: feature
status: planning
priority: medium
horizon: now
tags: [graph, integration, maintenance, knowledge, retrieval]
relations:
  - target: timely-briefs
    kind: related
  - target: retrieval-contradiction-detection
    kind: builds-on
  - target: graph-conflict-detection
    kind: complements
  - target: spec-drift-detection
    kind: complements
created: 2026-05-12
---

## Problem

Hero ingests new content constantly — specs land, knowledge entries are
captured, decisions are recorded, conventions are extracted, status flips —
and each write is treated as an isolated event. The new node enters the
graph, the FTS5 index updates, and that is it. Nothing **integrates** the
new node against the existing substrate.

The cost compounds. Two knowledge entries on the same topic accumulate
without anyone noticing they say the same thing in different words. A new
decision implicitly supersedes an older one, but no `supersedes` edge is
written, so retrieval keeps returning both as if they were equally current.
A new spec references three existing specs by slug in its prose, but no
`related` edges form, so traversal can't follow the trail. A new convention
gets extracted, but the older convention it modernizes sits unflagged.

`graph-conflict-detection` catches the **divergent-write** case (two pushes
of the same `(type, key)`). `retrieval-contradiction-detection` catches the
**stale-read** case (returning an outdated node). `spec-drift-detection`
catches the **code-vs-spec** case. None of them catch the case where a
*coherent* write lands but no integration happens — the substrate stays a
pile of valid-but-unconnected nodes, and retrieval quality silently
degrades over time.

The Karpathy "LLM Wiki" framing names this directly: a substrate that
accumulates without integration is a worse substrate every week, not a
better one. The maintenance overhead is what historically killed knowledge
systems. With LLMs the per-op cost of integration approaches zero, which
makes a write-through maintenance layer viable for the first time.

## Goal

Ship a write-through synthesis layer that, on every successful graph write,
runs a fixed pipeline of **cheap deterministic integrations** automatically
and queues a small set of **proposed expensive integrations** for human (or
agent) review. Cheap ops apply immediately, are reversible, and never
involve LLM calls. Expensive ops never auto-apply — they accumulate in a
maintenance queue surfaced via a dedicated CLI/MCP/brief surface, where
each proposal is one explicit accept/dismiss decision.

The substrate stays coherent without anyone scheduling cleanup work, and
the user keeps explicit control over every irreversible change.

**Mission-fit.** This is direct: "every session starts as smart as where
the last one left off" requires the substrate the next session reads to
actually be coherent. Without write-through integration, the corpus
accumulates noise faster than it accumulates signal, and the next session
inherits a worse map of the work, not a better one. Floor-raising because
maintenance is exactly the kind of disciplined hygiene work seniors do
intuitively and juniors skip — automating the cheap half and queueing the
expensive half means everyone's substrate stays clean by default.

## Design

### Two tiers, one trigger

Every successful write to the graph (any node insert/update via
`hero scan`, `hero note`, `hero capture`, `hero spec`, `hero deliver`, MCP
writes, etc.) fires a single hook: `maintenance.OnWrite(node, prev)`. The
hook fans out to two tiers.

**Tier 1 — Auto-integrations (apply immediately, no LLM):**

| Op | Trigger | Action |
|---|---|---|
| **Slug-reference edge** | New node body mentions an existing slug (regex over `(type:key)` patterns and bare slugs) | Insert `related` edge if not already present |
| **Tag co-occurrence edge** | New node shares ≥ 2 tags with an existing node and no edge exists | Insert `tag-related` edge with the shared tag set as edge props |
| **Contradiction warning refresh** | New node shares `(type, key)` or has high BM25 overlap with existing nodes | Recompute and cache contradiction warnings for the affected `(type, key)` set |
| **Status-transition edge** | Spec status flipped (`delivering` → `completed`, etc.) | Insert `transitioned` edge from the prior status row to the new one |
| **Files-touched extension** | New commit lands files mapped to a spec | Update `files_touched` index, increment touch count, refresh last-touched timestamp |
| **Convention rebind** | New code symbol matches a convention's pattern | Insert `instance-of` edge from symbol → convention |

All Tier 1 ops are deterministic, O(1) or O(log N) per write, idempotent,
and reversible by inverse op. No LLM calls. Bounded compute budget per
write (default 100ms; degrade gracefully if exceeded — log
`maintenance.budget_exceeded` and skip remaining Tier 1 ops for that
write).

**Tier 2 — Proposed integrations (queued, never auto-applied):**

| Op | Trigger | Proposal contents |
|---|---|---|
| **Merge candidate** | Two nodes of same type with BM25 similarity ≥ 0.85 and overlapping tag sets | Both node refs, similarity score, suggested merged title/body (LLM-generated, lazy) |
| **Supersede candidate** | New node of same `(type)` and high topical overlap with an existing node, newer `valid_from` | Old + new refs, suggested `supersedes` edge direction |
| **Convention extraction** | ≥ 3 code symbols or knowledge entries match a recurring pattern not yet captured as convention | Pattern signature, exemplar refs, suggested convention slug + body |
| **Summary rewrite** | A "rolling summary" knowledge entry (e.g. an architecture context doc) has had ≥ 5 referenced nodes change since its last `valid_from` | Summary ref, list of changed referenced nodes, suggested rewrite (LLM-generated, lazy) |
| **Stale-tag** | A knowledge entry's tags reference a tag no longer used anywhere else in the corpus | Entry ref, dead tag, suggested replacement tags |

Tier 2 ops produce `MaintenanceProposal` records persisted in a new SQLite
table `maintenance_queue`. The LLM-generated body of each proposal is
**lazy**: the proposal is queued with the cheap signal data, and the LLM
suggestion is generated on-demand the first time the proposal is viewed
(via `hero maintain show <id>`) — not at write time. This keeps write-time
cost bounded.

### `MaintenanceProposal` shape

```
type MaintenanceProposal struct {
    ID          string             // ULID
    Kind        string             // merge_candidate | supersede_candidate | convention_extraction | summary_rewrite | stale_tag
    Refs        []NodeRef          // nodes the proposal involves
    Signal      map[string]any     // BM25 score, tag overlap, etc.
    Body        *string            // LLM-suggested change, generated lazily
    BodyAt      *time.Time         // when Body was generated
    Status      string             // pending | accepted | dismissed | expired
    CreatedAt   time.Time
    DecidedAt   *time.Time
    DecidedBy   *string            // user / agent identifier
    Reason      *string            // optional dismissal reason
}
```

### Triggering rules

`maintenance.OnWrite` is called from a single chokepoint in the write
path: `internal/store/Write()`. This is the only place graph writes
land — Tier 1 fires synchronously inside the same transaction (so an
integration failure can roll back the write), Tier 2 fires asynchronously
on a worker after the transaction commits.

To prevent feedback loops:

- Writes performed by `maintenance.OnWrite` itself (Tier 1 edge inserts)
  are tagged `client_id = maintenance` and **do not** re-fire the hook.
- Accepting a Tier 2 proposal applies the change as a normal write — that
  write *can* fire the hook, but the resulting Tier 1 ops will be no-ops
  because the proposal was the integration.
- A debounce per `(type, key)`: if the same key is written more than 10
  times in a 60-second window, Tier 2 ops for that key are coalesced (only
  the most recent is queued, prior pending proposals on the same `(type,
  key)` are superseded).

### Surfacing the maintenance queue

Three surfaces:

**1. CLI: `hero maintain`**

```
hero maintain                    # list pending proposals, ranked by signal strength
hero maintain show <id>          # detail view; lazy-generates Body if not yet present
hero maintain accept <id>        # apply the proposal; marks decided, applies the write
hero maintain dismiss <id> [-r reason]  # mark dismissed; remembers the (refs, kind) tuple to suppress duplicates for 30 days
hero maintain stats              # counts by kind, accept/dismiss ratio, age distribution
hero maintain --since 7d         # filter by creation time
```

**2. MCP: `hero_maintenance_queue`**

```json
{
  "name": "hero_maintenance_queue",
  "description": "List or act on synthesis-maintenance proposals",
  "inputSchema": {
    "type": "object",
    "properties": {
      "action": { "type": "string", "enum": ["list", "show", "accept", "dismiss"] },
      "id":     { "type": "string" },
      "since":  { "type": "string" },
      "reason": { "type": "string" }
    }
  }
}
```

**3. Briefs integration (the seam with `timely-briefs`).**

The weekly synthesis brief surfaces the top N pending Tier 2 proposals as
a "Maintenance Opportunities" panel. The brief does **not** apply
proposals — it links to `hero maintain show <id>`. Proposal acceptance is
always an explicit user (or agent) decision through the maintain surface.

### Configuration

`hero.json`:

```json
{
  "maintenance": {
    "tier1_enabled": true,
    "tier1_budget_ms": 100,
    "tier2_enabled": true,
    "tier2_workers": 2,
    "tier2_queue_max": 500,
    "dismissed_ttl_days": 30,
    "debounce_writes": 10,
    "debounce_window_s": 60,
    "merge_candidate_threshold": 0.85,
    "supersede_candidate_threshold": 0.80,
    "convention_extraction_min_instances": 3
  }
}
```

Disabled tier-by-tier: setting `tier1_enabled: false` disables the hook
entirely (writes proceed as today). Setting `tier2_enabled: false` keeps
Tier 1 ops but stops queueing proposals — useful for environments where
review bandwidth is the bottleneck.

### Observability

Every Tier 1 op logs a `maintenance.applied` event to the feed with
`{op, refs, duration_ms}`. Every Tier 2 proposal logs
`maintenance.proposed` with `{kind, refs, signal}`. Every accept/dismiss
logs `maintenance.decided` with `{id, decision, by}`. These flow through
the standard feed so briefs can surface "you've dismissed 8 merge
candidates this week — is the threshold too low?" as a finding.

### Edge cases and explicit non-handling

- **Read-only operations** (`hero search`, `hero context`, etc.) never
  trigger maintenance. Only writes do.
- **Bulk import** (`hero scan` over a fresh repo) bypasses Tier 2 entirely
  via a `--no-maintenance` flag the scanner sets internally — otherwise a
  fresh scan would queue thousands of proposals at once. Tier 1 still
  runs because it's bounded per write. After the scan completes, a single
  one-shot `hero maintain --catchup` queues a single round of Tier 2 ops
  over the new corpus.
- **Federation pulls** trigger maintenance on the receiving side — pulling
  in a teammate's specs is a write and should integrate against the local
  substrate.
- **Failed Tier 1 ops** roll back the original write. This is intentional:
  if integration is corrupting the graph, the underlying write is also
  suspect.

## Changes

- `internal/maintenance/maintenance.go` — `OnWrite(node, prev)`,
  Tier 1 op implementations, dispatcher
- `internal/maintenance/tier1_*.go` — one file per Tier 1 op for clarity
- `internal/maintenance/tier2.go` — proposal generation, signal
  computation, lazy Body generation
- `internal/maintenance/queue.go` — `MaintenanceProposal` CRUD over the
  new `maintenance_queue` SQLite table
- `internal/maintenance/maintenance_test.go` — per-op tests, feedback
  loop prevention, debounce
- `internal/store/store.go` — call `maintenance.OnWrite` at the end of
  `Write()` inside the transaction
- `internal/cli/maintain.go` — `hero maintain` command + subcommands
- `internal/cli/root.go` — register `maintainCmd`
- `internal/serve/mcp_tools.go` — register `hero_maintenance_queue`
  MCP tool
- `internal/brief/render_html.go` — extend weekly template renderer to
  include the "Maintenance Opportunities" panel
- `internal/index/migrations/` — add `maintenance_queue` table migration
- `hero.json` schema — add `maintenance` section

## Acceptance Criteria

- WHEN any node is successfully written via `internal/store.Write()` AND
  `maintenance.tier1_enabled` is true THE SYSTEM SHALL invoke
  `maintenance.OnWrite(node, prev)` synchronously inside the same
  transaction
- WHEN Tier 1 ops complete THE SYSTEM SHALL log a `maintenance.applied`
  event per applied op with `{op, refs, duration_ms}`
- IF the cumulative Tier 1 budget for a single write exceeds
  `tier1_budget_ms` THEN THE SYSTEM SHALL skip remaining Tier 1 ops and
  log `maintenance.budget_exceeded`
- WHEN a Tier 1 op fails inside the transaction THE SYSTEM SHALL roll
  back the original write and surface the error to the caller
- WHEN a write tagged `client_id = maintenance` lands THE SYSTEM SHALL
  NOT re-invoke `maintenance.OnWrite` for that write
- WHEN Tier 2 op signals are met for a write AND `maintenance.tier2_enabled`
  is true THE SYSTEM SHALL queue a `MaintenanceProposal` asynchronously
  after the transaction commits
- WHEN a Tier 2 proposal is queued THE SYSTEM SHALL NOT generate the
  proposal `Body` until the proposal is first viewed via
  `hero maintain show` or `hero_maintenance_queue` action=show
- WHEN the same `(type, key)` is written more than `debounce_writes`
  times within `debounce_window_s` seconds THE SYSTEM SHALL coalesce
  Tier 2 proposals so only the most recent pending proposal per
  `(kind, refs)` survives
- WHEN `hero maintain` runs with no arguments THE SYSTEM SHALL list all
  pending proposals ranked by signal strength
- WHEN `hero maintain accept <id>` runs THE SYSTEM SHALL apply the
  proposal as a normal graph write, mark the proposal `accepted`, and
  record `decided_at` / `decided_by`
- WHEN `hero maintain dismiss <id>` runs THE SYSTEM SHALL mark the
  proposal `dismissed` and suppress duplicate `(refs, kind)` proposals
  for `dismissed_ttl_days` days
- WHEN `hero scan` runs a bulk import THE SYSTEM SHALL bypass Tier 2 ops
  for the duration of the scan and queue a single `hero maintain --catchup`
  round after completion
- WHEN a weekly synthesis brief is generated AND there are pending Tier 2
  proposals THE SYSTEM SHALL include a "Maintenance Opportunities" panel
  in the HTML output with the top N proposals
- THE SYSTEM SHALL NOT auto-apply any Tier 2 proposal under any
  configuration
- THE SYSTEM SHALL NOT make LLM API calls during Tier 1 op execution
- THE SYSTEM SHALL execute Tier 2 proposal generation on a bounded worker
  pool (`tier2_workers`, default 2) so proposal generation cannot starve
  the write path
- WHERE `maintenance.tier1_enabled` IS false THE SYSTEM SHALL skip the
  hook entirely and writes SHALL proceed as if the maintenance package
  were absent

## Boundaries

- Does **not** auto-apply any expensive op. Every Tier 2 proposal is one
  explicit accept/dismiss decision.
- Does **not** attempt content merge of two specs / two knowledge entries
  itself. The proposal *suggests* a merged body via LLM; the user reviews
  the suggested body before accepting.
- Does **not** replace `graph-conflict-detection` (concurrent-push case)
  or `retrieval-contradiction-detection` (stale-read case). It complements
  them by handling the no-conflict-but-no-integration case.
- Does **not** detect architectural drift — that is `architectural-drift-detection`.
  This spec is about substrate coherence, not architectural compliance.
- Does **not** generate briefs — that is `timely-briefs`. The brief
  *displays* maintenance opportunities; this spec produces them.
- Does **not** federate maintenance proposals across cloud / cross-repo —
  proposals are workspace-local. Cross-org maintenance is a downstream
  concern of `cloud-mcp`.
- Does **not** rewrite history. All maintenance ops are forward-only:
  Tier 1 ops insert new edges or refresh caches; accepted Tier 2 ops are
  normal graph writes that produce new bitemporal rows. Old rows survive.
- Does **not** support per-user proposal queues in v1 — single shared
  workspace queue. Multi-user routing is downstream.
- Does **not** require any new daemon. Tier 2 work runs on a worker pool
  inside the existing Hero process; if Hero is not running, proposals
  queue up next time it runs.
