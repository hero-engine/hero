---
title: Tenant Isolation — Postgres Row-Level Security
type: feature
status: planning
priority: P0
tags: [security, multi-tenancy, database, pre-launch]
created: 2026-04-27
relations:
  - target: pre-launch-hardening
    kind: child
  - target: cloud-api
    kind: related
horizon: someday
smoke: deferred
---

## Problem

Tenant isolation in Hero Cloud is currently enforced only in
**application code** — every handler reads `claims.UserID`, checks
`GetMemberRole(orgID, userID)`, and trusts that downstream queries
include `WHERE org_id = $orgID`. A single missing filter in a single
query path is enough to leak one customer's graph nodes, specs, or
knowledge entries to another customer.

This is the kind of bug that destroys SaaS companies. It's never the
core paths — those get reviewed. It's an analytics endpoint added in
a hurry, a support tool, an internal debug route. We need defense in
depth before any customer data lives in the cluster.

## Resolution

Enable PostgreSQL row-level security (RLS) on every per-tenant table.
The database itself rejects cross-tenant reads even if a query forgets
the `WHERE` clause. Application code stays in charge of *who is allowed
to see what*; RLS enforces the floor.

### Schema

**CockroachDB constraint discovered during implementation:** CRDB v26.1's
RLS does not support subqueries in policy expressions. So we cannot
write `repo_id IN (SELECT id FROM repos WHERE org_id = ...)`. Instead
we **denormalize org_id** onto every per-tenant table that previously
relied on a foreign key (specs, knowledge, conventions, pr_checks).
The migration adds the column, backfills it from the FK, indexes it,
and uses simple direct comparison in every policy.

Add migration 7:

```sql
-- Enable RLS on all tenant tables.
ALTER TABLE specs              ENABLE ROW LEVEL SECURITY;
ALTER TABLE knowledge          ENABLE ROW LEVEL SECURITY;
ALTER TABLE conventions        ENABLE ROW LEVEL SECURITY;
ALTER TABLE graph_nodes        ENABLE ROW LEVEL SECURITY;
ALTER TABLE graph_edges        ENABLE ROW LEVEL SECURITY;
ALTER TABLE graph_node_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE graph_edge_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE activity_events    ENABLE ROW LEVEL SECURITY;
ALTER TABLE pr_checks          ENABLE ROW LEVEL SECURITY;

-- Policy: app sets the current org via SET LOCAL on each request.
CREATE POLICY org_isolation_specs ON specs
  USING (repo_id IN (SELECT id FROM repos WHERE org_id = current_setting('app.org_id')::uuid));

CREATE POLICY org_isolation_knowledge ON knowledge
  USING (repo_id IN (SELECT id FROM repos WHERE org_id = current_setting('app.org_id')::uuid));

CREATE POLICY org_isolation_graph_nodes ON graph_nodes
  USING (org_id = current_setting('app.org_id')::uuid);

CREATE POLICY org_isolation_graph_edges ON graph_edges
  USING (org_id = current_setting('app.org_id')::uuid);

-- ... etc for each table

-- A service role bypasses RLS for migrations and admin work.
ALTER TABLE specs FORCE ROW LEVEL SECURITY;  -- applies to table owner too
```

The `current_setting('app.org_id', true)` returns NULL if unset, which
fails the policy → zero rows returned. Safe by default.

### Middleware

Wrap every authenticated request handler so the connection is bound
to the caller's org for the duration of the request:

```go
// in cloud/middleware/auth.go after JWT verification:
conn, err := db.Pool().Acquire(ctx)
defer conn.Release()
_, err = conn.Exec(ctx, "SET LOCAL app.org_id = $1", orgID)
// pass `conn` (not `pool`) into the handler context
```

This requires threading a request-scoped DB handle instead of using
the pool directly. The store layer already does `db.pool.Query(...)`
everywhere — we change it to accept a `pgx.Tx` or `pgx.Conn` argument.

### Carve-outs

Three areas legitimately read across orgs:

1. **Cross-org intelligence** (`stack_profiles`, `global_patterns`,
   `global_conventions`) — no RLS, anonymized by design
2. **GitHub installations** lookup at webhook receive time — uses a
   service role
3. **Auth flows** before the user has an org context — uses a service role

The service role is a separate Postgres user (`hero_service`) with
`BYPASSRLS`. The app role (`hero_app`) does not have it.

## Test plan

- Add an integration test that creates two orgs, two users, two
  workspaces of data, and asserts that a query under user A's session
  returns zero rows from user B's data even when the query has no
  `WHERE org_id` filter
- Add a fuzz/contract test that runs every read query in the codebase
  under a context where `app.org_id` is unset and asserts the result
  is empty
- Manual: try direct SQL via a leaked connection and confirm RLS denies
  cross-tenant reads

## Files

| File | Change |
|---|---|
| `cloud/store/migrations.go` | Migration 7 — enable RLS, create policies |
| `cloud/middleware/auth.go` | Acquire request-scoped conn, set `app.org_id` |
| `cloud/store/db.go` | Add `RequestConn` type, thread through handlers |
| `cloud/api/*.go` | Use request-scoped conn instead of pool |
| `cmd/hero-cloud/main.go` | Configure two roles: hero_app (RLS), hero_service (BYPASSRLS) |

## Success criteria

- All tenant tables have RLS enabled and a policy
- Every authenticated handler runs under `SET LOCAL app.org_id = <claim>`
- Cross-tenant integration test passes (zero leakage)
- No regression in existing federation push/pull tests
- Service role explicitly used for cross-org intelligence and auth paths

## Out of scope

- Per-row encryption (separate feature)
- Audit logging of denied queries (file as `rls-audit-trail` later)
- Customer-managed encryption keys (enterprise tier)
