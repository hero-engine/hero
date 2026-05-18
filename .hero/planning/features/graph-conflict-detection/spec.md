---
title: Graph Conflict Detection — Detect and Surface Concurrent Node Divergence
slug: graph-conflict-detection
type: feature
status: delivering
priority: P1
tags: [graph-memory, federation, sync, conflicts]
created: 2026-04-27
relations:
  - target: graph-memory-7c-live-test
    kind: sibling
  - target: graph-memory-federation
    kind: child
horizon: now
smoke: deferred
---

## Problem

When two developers push different versions of the same `(type, key)`
node without coordinating, the server silently applies last-write-wins.
The "losing" version survives in bitemporal history but neither client
is told a divergence occurred. There is no signal to the developer
that their teammate's version was overwritten, or vice versa.

`hero check conflicts <slug>` exists but checks local spec-file state
(FTS5 index), not graph-level divergence.

## Resolution strategy: last-write-wins + detect + surface

**No automatic merge.** Specs are source-of-truth as files on disk;
git handles content merge when branches meet. The graph is a derived
representation. Merging `status: delivering` with `status: in-review`
has no safe semantic — the intent belongs to the developer, not the
sync layer.

The right workflow:
1. Bob's push wins (last-write-wins, current behavior). Alice's
   version is closed in bitemporal history — nothing is lost.
2. The push response tells Bob his version overwrote Alice's, and
   which fields differed.
3. Alice's next pull surfaces the divergence so she can re-scan and
   re-push if she disagrees.
4. Actual text conflicts are resolved in git when branches merge —
   the graph reflects whatever state the spec file is in after that.

**v2 (out of scope here):** field-level merge for non-conflicting
fields (Alice changed `status`, Bob changed `description` — both
changes can survive). Requires per-field diff/merge semantics.

## What counts as a conflict

A conflict is when:
- An incoming node has the same `(org_id, type, key)` as an existing
  current node (`valid_to IS NULL`)
- The `content_hash` differs (it's not an idempotent re-push)
- The existing row's `client_id` differs from the incoming node's
  `client_id` (same client re-pushing is not a conflict — it's a
  legitimate update)

Same-client re-pushes are intentional updates, not conflicts.

## Implementation

### 1. Server — populate conflicts in `PushGraphDelta`

In `cloud/store/graph.go:PushGraphDelta`, extend the SELECT to also
fetch `client_id` from the existing row. When invalidating, if the
existing `client_id` differs from the incoming node's `client_id`,
append a `GraphConflict` to the return slice.

```go
// existing SELECT extended:
SELECT content_hash, COALESCE(client_id, '') FROM graph_nodes
  WHERE org_id=$1 AND type=$2 AND key=$3 AND valid_to IS NULL

// on mismatch with different client_id:
conflicts = append(conflicts, GraphConflict{
    NodeType: n.Type,
    NodeKey:  n.Key,
    Reason:   fmt.Sprintf("overwritten: prior version from client %s", existingClientID),
})
```

The `conflicts` slice is already wired through `handlePush` into the
`PushResponse` JSON (`"conflicts"` field). No handler changes needed.

### 2. Client — print conflicts in `hero sync graph push`

In `internal/cli/sync_graph.go:runSyncGraphPush`, after a successful
push, check `pr.Conflicts`. If non-empty, print each one as a warning:

```
Pushed: 42 rows accepted (server time ...)
Warning: 1 conflict — your version won, but a teammate's version was overwritten:
  Feature conflict-test  (prior client: abc123)
Run 'hero sync graph pull && hero scan' if you want to reconcile.
```

### 3. Client — extend `hero check conflicts` for graph divergence

`hero check conflicts <slug>` currently checks the FTS5 spec index.
Extend it to also query the local `graph.db` for nodes where:
- `key` matches the slug (exact or suffix `:<slug>`)
- Multiple rows exist (any `valid_to`) within the same `(type, key)`
  from different `client_id`s

This detects "two clients pushed different versions" from the
bitemporal history, without needing a server round-trip.

```sql
SELECT type, key, client_id, valid_from, valid_to,
       json_extract(props, '$.status') as status
FROM nodes
WHERE (key = ? OR key LIKE '%:' || ?)
  AND valid_from > datetime('now', '-30 days')
ORDER BY type, key, valid_from DESC
```

If multiple `client_id` values appear for the same `(type, key)`,
surface the divergence with both versions' statuses.

## Success criteria

- `hero sync graph push` prints a warning when it overwrites a
  different client's node
- `hero check conflicts conflict-test` reports the divergence after
  Alice and Bob both push the same slug with different statuses
- Re-running push with no local changes after a pull is a no-op
  (idempotency — same client_id, same hash → no conflict)
- Same-client re-push of changed content is NOT flagged as a conflict

## Files

| File | Change |
|---|---|
| `cloud/store/graph.go` | Extend SELECT in `PushGraphDelta` to read `client_id`; populate `conflicts` |
| `internal/cli/sync_graph.go` | Print conflicts from push response |
| `internal/cli/check.go` | Extend `runCheckConflicts` to query `graph.db` |

## Out of scope

- Server-side conflict storage (`graph_conflicts` table) — bitemporal
  history is sufficient for v1; a dedicated table is a v2 concern
- Field-level merge for non-conflicting props
- Conflict resolution UI
