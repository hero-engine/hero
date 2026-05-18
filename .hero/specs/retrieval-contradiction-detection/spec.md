---
title: Retrieval Contradiction Detection — Surface Stale Facts at Read Time
slug: retrieval-contradiction-detection
type: feature
status: planning
tags: [retrieval, graph, temporal, context, quality]
created: 2026-05-06
relations:
  - target: context-injection
    kind: related
  - target: spec-drift-detection
    kind: related
  - target: spec-status-integrity
    kind: related
horizon: now
---

## Goal

When the retrieval layer returns a node — a decision, spec, knowledge entry,
or convention — check whether a newer node on the same topic exists that
supersedes or contradicts it. If so, attach a staleness warning to the result
so consumers (MCP tools, CLI, context injection) can surface it. The agent
session starts with correct context instead of silently trusting outdated
facts.

## Problem

Hero's bitemporal graph preserves every version of every node (`valid_from` /
`valid_to`), but the retrieval layer only returns the current row
(`valid_to IS NULL`). This means:

1. **Silent supersession** — A decision node gets revised, the old row is
   invalidated, but a context injection that cached the old title/summary
   still presents the stale version. Nothing warns that the ground shifted.

2. **Status divergence** — A spec moves from `delivering` to `completed` or
   `regressed`, but a search result returned moments before the re-index
   still shows the prior status. Graph and FTS5 index can be out of sync
   within a session.

3. **Concurrent edit blindspots** — `FindGraphConflicts` catches concurrent
   edits at write/push time, but a reader querying the graph between pushes
   sees whichever version won last-write-wins, with no indication that
   another client submitted a competing version.

4. **Topical overlap without explicit edges** — Two knowledge entries or
   decisions may cover the same topic without a `supersedes` edge. Today
   nothing connects them; a reader can consume both without knowing they
   disagree.

These failure modes compound in multi-agent workflows: one agent reads a
decision, another agent revises it, and a third agent delivers against the
stale version.

## Design

### Contradiction signals

Five signals, all computable from existing graph + FTS5 data with no LLM
calls:

| Signal | Detection method | Severity |
|---|---|---|
| **Explicit supersession** | `supersedes` edge exists from a newer node to the retrieved node | High |
| **Bitemporal revision** | Same `(type, key)` has a newer `valid_from` row (the retrieved row has `valid_to IS NOT NULL` but was returned from a stale cache/index) | High |
| **Concurrent edit** | Same `(type, key)` has versions from multiple `client_id` values in the last 30 days | Medium |
| **Topical overlap** | A different node of the same type scores ≥ 0.8 BM25 similarity against the retrieved node's title+body, with a newer `valid_from` | Medium |
| **Status mismatch** | FTS5 index `status` differs from the graph node's `props.status` for the same key | Low |

### Where detection runs

A new package `internal/contradict` provides the detection logic. It is
called from three points in the retrieval pipeline:

1. **`retrieval.Retrieve()`** — After results are scored and ranked, run
   contradiction checks against the top-N results (not the full candidate
   set). Attach warnings to each `Result`.

2. **`index.BuildContext()`** — After assembling the `ContextBlock`, check
   each entry in Decisions, Conventions, PastWork, and InFlight for
   contradictions. Attach warnings to a new `Warnings` field on
   `ContextBlock`.

3. **`traversal.Why()`** — After building the trace chain, check each hop
   for supersession. Attach warnings to a new `Warnings` field on `Trace`.

### Data flow

```
  Retrieve(query)
       │
       ▼
  score + rank (existing)
       │
       ▼
  contradict.Check(store, results)   ◄── new
       │
       ├── explicit supersession (edge query)
       ├── bitemporal revision (history query)
       ├── concurrent edit (client_id check)
       ├── topical overlap (BM25 cross-match)
       └── status mismatch (graph vs index)
       │
       ▼
  results with Warnings attached
```

### Result and ContextBlock changes

```go
// In retrieval package:
type Result struct {
    // ... existing fields ...
    Warnings []Warning  // NEW: contradictions detected at retrieval time
}

// In index package:
type ContextBlock struct {
    // ... existing fields ...
    Warnings []Warning  // NEW: contradictions across context entries
}

// In contradict package:
type Warning struct {
    Signal      string // "superseded", "revised", "concurrent_edit",
                       // "topical_overlap", "status_mismatch"
    Severity    string // "high", "medium", "low"
    AffectedKey string // the retrieved node's key
    NewerKey    string // the node that contradicts/supersedes it (if any)
    NewerTitle  string // human-readable title of the newer node
    Message     string // one-line explanation
    ValidFrom   string // RFC3339: when the newer version became valid
}
```

### Query budget

Contradiction detection adds SQL queries per retrieval call. Budget:

- **Explicit supersession**: 1 query, `IN` clause over result keys → edges
  table. O(1) per call.
- **Bitemporal revision**: 1 query, `IN` clause over result `(type, key)`
  pairs → nodes table where `valid_to IS NOT NULL`. O(1) per call.
- **Concurrent edit**: Reuses the bitemporal query results, just counts
  distinct `client_id`. No extra query.
- **Topical overlap**: 1 FTS5 query per result in the top-N (default N=5).
  O(N) per call, capped.
- **Status mismatch**: 1 query, `IN` clause joining graph nodes and FTS5
  index. O(1) per call.

Total: 3 fixed queries + up to 5 FTS5 queries per retrieval. The FTS5
queries use the existing `fts_nodes` table and complete in <1ms each on a
typical workspace. The topical overlap cap (default 5) is configurable via
`contradict.MaxOverlapChecks`.

### MCP tool output

Warnings render as a `## Contradictions` section in `formatContextBlock()`
and as a `warnings` array in JSON output from `hero_search`, `hero_context`,
`hero_why`, and `hero_recap`:

```
## Contradictions

⚠ Decision `api-auth-strategy` may be stale:
  superseded by `api-auth-strategy-v2` (2026-05-01)
  → "Switched from JWT to session tokens per legal review"

⚠ Convention `error-handling` has concurrent edits:
  your version (claude-code/proj-a, 2026-04-28) vs.
  teammate's version (cursor/proj-b, 2026-04-30)
  → Run `hero conflicts error-handling` to resolve
```

### CLI surface

`hero search` and `hero context` print warnings inline. A new
`--no-contradictions` flag suppresses the check for scripted/CI use where
latency matters.

`hero contradictions [slug]` is a standalone command that runs all five
signals against a specific node or across all current nodes, useful for
workspace health checks.

## Changes

- `internal/contradict/contradict.go` — `Check()`, `CheckContext()`, `CheckTrace()` functions; `Warning` struct
- `internal/contradict/contradict_test.go` — table-driven tests per signal
- `internal/retrieval/retrieval.go` — call `contradict.Check()` after scoring in `Retrieve()`
- `internal/index/index.go` — add `Warnings []Warning` to `ContextBlock`; call `contradict.CheckContext()` in `BuildContext()`
- `internal/traversal/why.go` — add `Warnings []Warning` to `Trace`; call `contradict.CheckTrace()` after chain assembly
- `internal/serve/mcp_tools.go` — render warnings in `formatContextBlock()` and JSON tool outputs
- `internal/cli/search.go` — render warnings in CLI output; add `--no-contradictions` flag
- `internal/cli/context.go` — render warnings in CLI output; add `--no-contradictions` flag
- `internal/cli/contradictions.go` — `hero contradictions` standalone command
- `internal/cli/root.go` — register `contradictionsCmd`

## Acceptance Criteria

- WHEN `retrieval.Retrieve()` returns a node that has an inbound `supersedes` edge from a newer node THE SYSTEM SHALL attach a high-severity warning naming the superseding node
- WHEN `retrieval.Retrieve()` returns a node whose `(type, key)` has a newer `valid_from` row in the graph THE SYSTEM SHALL attach a high-severity warning indicating the node was revised
- WHEN a retrieved node's `(type, key)` has versions from two or more distinct `client_id` values in the last 30 days THE SYSTEM SHALL attach a medium-severity warning naming the competing clients
- WHEN a different node of the same type scores ≥ 0.8 BM25 similarity against the retrieved node's title+body and has a newer `valid_from` THE SYSTEM SHALL attach a medium-severity warning identifying the potentially superseding node
- WHEN the FTS5 index status for a key differs from the graph node's `props.status` for the same key THE SYSTEM SHALL attach a low-severity warning noting the mismatch
- WHEN `hero context` assembles a `ContextBlock` THE SYSTEM SHALL run contradiction checks on Decisions, Conventions, PastWork, and InFlight entries
- WHEN `hero why` builds a trace THE SYSTEM SHALL check each hop for supersession warnings
- WHEN `--no-contradictions` is passed to `hero search` or `hero context` THE SYSTEM SHALL skip all contradiction checks
- WHEN `hero contradictions <slug>` runs THE SYSTEM SHALL evaluate all five signals for the specified node and print results
- THE SYSTEM SHALL compute all contradiction signals without making any LLM API calls
- THE SYSTEM SHALL cap topical-overlap FTS5 queries at 5 per retrieval call by default

## Boundaries

- Does **not** auto-resolve contradictions — warnings are advisory, the human or agent decides
- Does **not** require vector embeddings (Phase C) — uses BM25 via existing FTS5 infrastructure
- Does **not** duplicate write-time conflict detection (`FindGraphConflicts`) — that catches concurrent pushes; this catches stale reads
- Does **not** check code symbol nodes — they change too frequently to produce useful contradiction signals
- Does **not** persist contradiction state — recomputed on demand like drift detection
- Does **not** block any operation — all warnings are informational
