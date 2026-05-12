---
title: Graph Schema Simplification — Upsert + History Table
type: feature
status: planning
priority: P1
tags: [graph-memory, federation, performance, database]
created: 2026-04-27
relations:
  - target: graph-memory-federation
    kind: child
  - target: graph-conflict-detection
    kind: sibling
horizon: next
smoke: deferred
---

## Problem

The current server-side graph schema uses a **bitemporal model**: every
node and edge update closes the existing row (`valid_to = now`) and
inserts a new one. This means every push requires a read-modify-write
cycle per row — even in the bulk UNNEST implementation, each batch
is three SQL statements (SELECT existing, UPDATE to close, INSERT new).

CockroachDB's serializable isolation makes this expensive: large
transactions accumulate write spans and fail with
`TransactionRetryError: can't refresh txn spans` before we even hit
the data volume limit. We worked around it with batching and the UNNEST
bulk pattern, but the root cause is the schema, not the database.

The bitemporal model was chosen for:
1. Conflict detection — know which `client_id` wrote last
2. Audit history — see what changed and when

Both can be achieved with a simpler schema that decouples current state
from history and lets pushes use a single `INSERT ... ON CONFLICT DO
UPDATE` statement.

## Resolution

Replace the bitemporal `valid_to IS NULL` pattern with:

- **`graph_nodes` / `graph_edges`** — current state only. One row per
  `(org_id, type, key)`. Upserted on every push. No `valid_to`.
- **`graph_node_history` / `graph_edge_history`** — append-only audit
  log. A row is inserted here whenever current state changes. Never
  updated. Indexed by `(org_id, type, key, updated_at)` for history
  queries.

Push becomes one SQL statement per batch regardless of batch size.
Pull becomes `WHERE updated_at > $cursor`. Conflict detection reads
the displaced `client_id` from the upsert's RETURNING clause.

## Schema (migration 6)

```sql
-- Drop the bitemporal current-state tables.
DROP TABLE IF EXISTS graph_edges;
DROP TABLE IF EXISTS graph_nodes;

-- Current state — simple upsert target, one row per logical entity.
CREATE TABLE graph_nodes (
  org_id      UUID        NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
  repo        TEXT        NOT NULL DEFAULT '',
  unit        TEXT        NOT NULL DEFAULT '',
  type        TEXT        NOT NULL,
  key         TEXT        NOT NULL,
  props       JSONB       NOT NULL DEFAULT '{}',
  scope       TEXT        NOT NULL DEFAULT '',
  hash        TEXT        NOT NULL DEFAULT '',
  source      JSONB       NOT NULL DEFAULT '{}',
  client_id   TEXT        NOT NULL DEFAULT '',
  server_time TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (org_id, type, key)
);

CREATE INDEX idx_graph_nodes_cursor ON graph_nodes (org_id, server_time);
CREATE INDEX idx_graph_nodes_repo   ON graph_nodes (org_id, repo);
CREATE INDEX idx_graph_nodes_unit   ON graph_nodes (org_id, unit) WHERE unit != '';

CREATE TABLE graph_edges (
  org_id      UUID        NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
  repo        TEXT        NOT NULL DEFAULT '',
  unit        TEXT        NOT NULL DEFAULT '',
  from_type   TEXT        NOT NULL,
  from_key    TEXT        NOT NULL,
  to_type     TEXT        NOT NULL,
  to_key      TEXT        NOT NULL,
  type        TEXT        NOT NULL,
  props       JSONB       NOT NULL DEFAULT '{}',
  scope       TEXT        NOT NULL DEFAULT '',
  source      JSONB       NOT NULL DEFAULT '{}',
  client_id   TEXT        NOT NULL DEFAULT '',
  server_time TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (org_id, from_type, from_key, type, to_type, to_key)
);

CREATE INDEX idx_graph_edges_cursor ON graph_edges (org_id, server_time);
CREATE INDEX idx_graph_edges_repo   ON graph_edges (org_id, repo);

-- Append-only audit log — written when current state changes.
CREATE TABLE graph_node_history (
  id          BIGSERIAL   PRIMARY KEY,
  org_id      UUID        NOT NULL,
  type        TEXT        NOT NULL,
  key         TEXT        NOT NULL,
  props       JSONB       NOT NULL DEFAULT '{}',
  hash        TEXT        NOT NULL DEFAULT '',
  client_id   TEXT        NOT NULL DEFAULT '',
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_graph_node_history_entity
  ON graph_node_history (org_id, type, key, updated_at DESC);

CREATE TABLE graph_edge_history (
  id          BIGSERIAL   PRIMARY KEY,
  org_id      UUID        NOT NULL,
  from_type   TEXT        NOT NULL,
  from_key    TEXT        NOT NULL,
  to_type     TEXT        NOT NULL,
  to_key      TEXT        NOT NULL,
  type        TEXT        NOT NULL,
  props       JSONB       NOT NULL DEFAULT '{}',
  client_id   TEXT        NOT NULL DEFAULT '',
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_graph_edge_history_entity
  ON graph_edge_history (org_id, from_type, from_key, type, updated_at DESC);
```

## Push implementation (`cloud/store/graph.go`)

Replace `PushGraphDelta` with a two-statement batch per entity type.

### Nodes

```sql
-- Statement 1: upsert current state, get back what changed.
INSERT INTO graph_nodes
  (org_id, repo, unit, type, key, props, scope, hash, source, client_id, server_time)
SELECT $1,
  unnest($2::text[]), unnest($3::text[]), unnest($4::text[]),
  unnest($5::text[]), unnest($6::jsonb[]), unnest($7::text[]),
  unnest($8::text[]), unnest($9::jsonb[]), unnest($10::text[]), now()
ON CONFLICT (org_id, type, key) DO UPDATE
  SET repo        = excluded.repo,
      unit        = excluded.unit,
      props       = excluded.props,
      scope       = excluded.scope,
      hash        = excluded.hash,
      source      = excluded.source,
      client_id   = excluded.client_id,
      server_time = excluded.server_time
  WHERE graph_nodes.hash != excluded.hash
RETURNING type, key, client_id,
          (xmax != 0)                        AS was_updated,
          (xmax != 0 AND
           graph_nodes.client_id != excluded.client_id) AS is_conflict,
          graph_nodes.client_id              AS prior_client_id;
```

From the RETURNING rows:
- `was_updated = true` → row changed → write a history record and count as accepted
- `is_conflict = true` → different client wrote it → add to conflicts list
- `was_updated = false` (no row in RETURNING, or xmax=0) → idempotent, skip

```sql
-- Statement 2: append history for changed rows only.
INSERT INTO graph_node_history (org_id, type, key, props, hash, client_id)
SELECT $1, unnest($2::text[]), unnest($3::text[]),
           unnest($4::jsonb[]), unnest($5::text[]), unnest($6::text[]);
```

### Edges — same pattern with its PRIMARY KEY.

## Pull implementation

The `server_time` column on current-state rows acts as the pull cursor,
exactly as before. The query simplifies to:

```sql
SELECT org_id, repo, unit, type, key, props, scope, hash, source,
       client_id, server_time
  FROM graph_nodes
 WHERE org_id = $1
   AND server_time > $2
 ORDER BY server_time
 LIMIT $3;
```

No `valid_to IS NULL` filter needed — every row is current state.

## Conflict detection

The RETURNING clause from the upsert tells us:
- `is_conflict = true` → the row was overwritten by a different client
- `prior_client_id` → who owned it before

This replaces the existing SELECT-before-write conflict check entirely.

## Wire format changes

None. The client-side `PushRequest` / `PullResponse` structs and the
sync protocol are unchanged. `valid_from` / `valid_to` fields in the
wire format are dropped from the server store but can remain in the
wire struct for backward compatibility (server ignores them).

## Files changed

| File | Change |
|---|---|
| `cloud/store/migrations.go` | Add migration 6 with new schema |
| `cloud/store/graph.go` | Rewrite `PushGraphDelta`, `PullGraphDelta`, `ImpactCrossRepo` for new schema |
| `cloud/api/graph.go` | Remove `valid_from`/`valid_to` from `wireNode`/`wireEdge` mapping |
| `cloud/store/db.go` | No change needed |

## Migration strategy

Migration 6 drops and recreates `graph_nodes` and `graph_edges`. Any
data in those tables is lost. This is acceptable for v1 (no production
data yet). A data-preserving migration would be:

1. Create new tables under temp names
2. `INSERT INTO graph_nodes_new SELECT DISTINCT ON (org_id, type, key) ... FROM graph_nodes WHERE valid_to IS NULL ORDER BY server_time DESC`
3. Rename
4. Drop old tables

But given we're pre-launch, the destructive migration is fine.

## Out of scope

- Soft-delete / tombstone support (can be added as a `deleted_at`
  column if needed; for now, a missing row means deleted)
- History table pruning / TTL (add a background job later; history
  rows are cheap and grow slowly)
- Client-side local `graph.db` schema — SQLite bitemporal stays as-is;
  the simplification is server-only

## Success criteria

- Alice push of 5700 nodes: single transaction, under 500ms
- Bob push of 5700 nodes against existing data: single transaction,
  conflict detection via RETURNING, under 500ms
- `hero check conflicts` still works (conflicts come from push response
  → `push_conflicts.json`, no schema dependency)
- Pull cursor still works (server_time column unchanged)
- No serialization errors under load
