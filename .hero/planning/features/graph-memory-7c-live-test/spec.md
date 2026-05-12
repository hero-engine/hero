---
title: Graph Memory Phase 7c — Live Multi-Dev Sync Test
type: feature
status: planning
priority: P0
tags: [graph-memory, federation, sync, integration-test, ops]
created: 2026-04-26
relations:
  - target: graph-memory
    kind: parent
  - target: graph-memory-federation
    kind: parent
horizon: next
smoke: deferred
---

## Goal

Prove the federation sync protocol works end-to-end against real
infrastructure: a running cloud server with a real Postgres-compatible
DB, two separate developer workspaces that push and pull the unified
knowledge graph, and verification that one dev's writes show up in
the other's brief on the next `/resume`.

This is the validation step gating phase 8 (cross-repo impact) from
moving into production use. Phases 7a (client) and 7b (server) are
already shipped with unit + httptest coverage; this spec walks through
the live integration.

## What's already done (no code work needed)

- ✅ `internal/graph/sync.go` — client-side push/pull with sync_state
  table, idempotent re-push, scope filtering, edge endpoint
  hydration. Tested via httptest (`TestPullAndApply_RoundTripsWithEdges`).
- ✅ `cloud/store/graph.go` — server-side store methods
  (`PushGraphDelta`, `PullGraphDelta`, `ImpactCrossRepo`).
- ✅ `cloud/api/graph.go` — HTTP handlers for `/push`, `/pull`,
  `/impact` with auth via existing JWT middleware.
- ✅ `cloud/store/migrations.go` — schema v5 adds `graph_nodes` and
  `graph_edges` tables (auto-applied on first server start).
- ✅ `internal/cli/sync_graph.go` — `hero sync graph push|pull|status`
  CLI subverbs.

## Prerequisites

| Item | Why | How to get |
|---|---|---|
| Go 1.26+ | Build hero | `brew install go` (Mac) or `apt install golang-go` (Linux) |
| Docker Desktop | Run CockroachDB locally | https://docs.docker.com/get-docker/ |
| Hero repo checked out | Source code | `git clone <repo> hero && cd hero` |
| GitHub OAuth app | `hero login` flow | See "GitHub OAuth setup" below |
| `ANTHROPIC_API_KEY` (optional) | Phase 10 extraction | https://console.anthropic.com/settings/keys |
| Two terminal windows | Simulate two devs | — |

## Step-by-step setup

### 1. Build the binaries

```bash
cd ~/projects/personal/repository/hero
go build -o /usr/local/bin/hero ./cmd/hero
go build -o /usr/local/bin/hero-cloud ./cmd/hero-cloud
hero --version          # confirm install
hero-cloud --help       # confirm install
```

Both binaries should print version + usage. If `/usr/local/bin` isn't
writable, use `~/bin` or any directory on `$PATH`.

### 2. Start CockroachDB locally (single-node, Docker)

```bash
docker run -d --name hero-cockroach \
  -p 26257:26257 \
  -p 8089:8080 \
  cockroachdb/cockroach:v23.2.0 \
  start-single-node --insecure
```

Wait ~5 seconds for it to boot, then create the database:

```bash
docker exec -it hero-cockroach \
  ./cockroach sql --insecure --execute="CREATE DATABASE hero; CREATE USER hero WITH PASSWORD 'hero'; GRANT ALL ON DATABASE hero TO hero;"
```

**Verify:** http://localhost:8089 should load the CockroachDB admin UI.

### 3. Run migrations

```bash
hero-cloud --migrate
```

This applies all 5 schema migrations (orgs, users, repos, specs,
intelligence, graph). On success it prints "migrations applied"
and exits.

**Verify:**
```bash
docker exec -it hero-cockroach ./cockroach sql --insecure -d hero \
  --execute="\dt"
```
Should list tables including `graph_nodes` and `graph_edges`.

### 4. GitHub OAuth setup (one-time)

Hero Cloud uses GitHub OAuth for `hero login`. Create an OAuth app:

1. https://github.com/settings/developers → "New OAuth App"
2. **Application name:** `hero-cloud-local`
3. **Homepage URL:** `http://localhost:8080`
4. **Authorization callback URL:** `http://localhost:8080/api/v1/auth/github/callback`
5. Click "Register application"
6. Copy the **Client ID**
7. Click "Generate a new client secret" → copy the **secret**

### 5. Start the cloud server

```bash
export HERO_JWT_SECRET="$(openssl rand -hex 32)"
export GITHUB_CLIENT_ID="<from step 4>"
export GITHUB_CLIENT_SECRET="<from step 4>"
export GITHUB_REDIRECT_URL="http://localhost:8080/api/v1/auth/github/callback"
export HERO_DB_URL="postgresql://hero:hero@localhost:26257/hero?sslmode=disable"

hero-cloud
```

Should print `hero-cloud dev starting`, `connected to database`,
`listening on :8080`. Leave running in this terminal.

**Verify:** in another terminal,
```bash
curl http://localhost:8080/healthz
```
Returns `ok`.

### 6. Login + create org

```bash
export HERO_CLOUD_URL="http://localhost:8080"
hero login
```

Opens a browser to GitHub OAuth. Approve. Returns to terminal with
`Credentials saved to ~/.hero/credentials.json`.

```bash
hero admin team list   # should be empty
```

Create an org via the API (no CLI yet for this in local dev):

```bash
TOKEN=$(jq -r .access_token ~/.hero/credentials.json)
curl -X POST http://localhost:8080/api/v1/orgs \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "Test Org", "slug": "test-org"}'
```

Note the returned `id` field — that's your `org_id`.

### 7. Set up two developer workspaces

Simulate two devs by checking out the hero repo into two separate
directories. (For a real test you'd use two separate machines or
VMs; for the smoke test, two clones is enough.)

```bash
mkdir -p ~/hero-test
cd ~/hero-test
git clone ~/projects/personal/repository/hero alice-workspace
git clone ~/projects/personal/repository/hero bob-workspace
```

Configure each with the same org:

```bash
# Alice
cd ~/hero-test/alice-workspace
cat > .hero/hero.json.local <<EOF
{
  "cloud": {
    "org_id": "<org_id from step 6>",
    "repo_id": ""
  }
}
EOF
# Merge into existing hero.json or use the existing field
hero scan --code        # populates Alice's local graph

# Bob (in a new terminal)
cd ~/hero-test/bob-workspace
# same hero.json edit
hero scan --code        # populates Bob's local graph (same data, different IDs)
```

Both should now have populated `.hero/graph.db` files.

## The actual test

### Round 1: Alice pushes, Bob pulls (basic round-trip)

**As Alice:**
```bash
cd ~/hero-test/alice-workspace
hero sync graph status        # cursor empty, pending push high
hero sync graph push          # ships ~1300 nodes + 2500 edges
hero sync graph status        # pending push now 0
```

Expected output:
```
Pushed: 4007 rows accepted (server time 2026-04-26T...)
```

**As Bob:**
```bash
cd ~/hero-test/bob-workspace
hero sync graph pull          # pulls everything Alice pushed
```

Expected output:
```
Pulled: 1306 nodes, 2489 edges applied (0 deferred)
```

**Verify both stores converged:**
```bash
# Alice
hero graph stats              # note total_nodes / total_edges

# Bob
hero graph stats              # should match Alice's totals
```

### Round 2: Alice writes a new note, Bob sees it

**As Alice:**
```bash
hero note "round-trip-test-from-alice"
hero scan                     # ingests the note into the graph
hero sync graph push          # pushes the new node + edges
```

**As Bob:**
```bash
hero sync graph pull
hero search "round-trip-test"
```

Expected: `round-trip-test-from-alice` appears in search results.

**Verify it shows in the brief:**
```bash
hero resume                   # the new Note should appear under
                              # "Nearby" or via "search results"
```

### Round 3: Conflict detection

**As Alice:**
```bash
hero spec new conflict-test --type feature
# Edit the spec, set status=delivering
hero scan
hero sync graph push
```

**As Bob (without pulling first):**
```bash
hero spec new conflict-test --type feature   # same slug
# Edit differently, set status=in-review
hero scan
hero sync graph push
```

Expected: server returns 1 conflict in the response. Both versions
land in the graph as bitemporal-history rows; the latest-wins for
`(type, key)` lookups but `hero check conflicts` surfaces the
divergence.

**Verify:**
```bash
hero check conflicts          # should list conflict-test as conflicted
```

### Round 4: Cross-repo impact (phase 8 dogfood)

This requires two repos pushing data, not just two clones of the
same one. For a quick smoke test:

```bash
cd ~/hero-test/alice-workspace
hero impact --cross-repo internal/cli  # should query the cloud
```

Expected: returns whatever incoming edges to `Package: internal/cli`
exist across all pushed graphs. With just one repo's data, you'll
see only same-repo callers — but the API path is exercised.

## Success criteria

- ✅ Alice and Bob's `hero graph stats` show identical totals after
  Alice pushes + Bob pulls
- ✅ A new Note created on Alice's side surfaces in Bob's
  `hero search` after pull
- ✅ `hero resume` on Bob's side, after pull, includes Alice's
  changes in the brief
- ✅ Concurrent edits to the same `(type, key)` are surfaced as
  conflicts via `hero check conflicts`
- ✅ Re-running `hero sync graph push` with no new local changes is
  a no-op (idempotency holds against real DB)
- ✅ `hero impact --cross-repo` returns data from the cloud (even if
  empty / single-repo)
- ✅ `hero sync graph status` accurately tracks cursor and pending
  push count

## Common failure modes + debugging

| Symptom | Likely cause | Fix |
|---|---|---|
| `hero login` opens browser but never returns | Callback URL mismatch | Verify `GITHUB_REDIRECT_URL` in env exactly matches what's in the OAuth app |
| Cloud server: `database connection failed` | CockroachDB not running, or wrong DSN | `docker ps` to check; `docker logs hero-cockroach` for errors |
| Cloud server: `invalid token` on auth | JWT secret changed between issue and validate | Restart server with same `HERO_JWT_SECRET`, re-login |
| `hero sync graph push` returns 403 | Org membership check fails | Verify `cloud.org_id` in `hero.json` matches a real org you're a member of |
| Push succeeds but pull returns nothing | Cursor mismatch / time-skew | `hero sync graph status` to see the cursor; manually clear via SQL: `DELETE FROM sync_state WHERE server_url='...'` |
| `Pulled: N nodes, 0 edges, M deferred` | Edge endpoints not in graph yet | Run pull again — endpoints often come in a later batch; or push from origin in dependency order (nodes before edges depending on them) |
| Sync includes unexpected `local`-scope rows | Bug | Should not happen — `internal/graph/sync.go:nodesSince` filters scope. File issue with reproduction. |

## Cleanup

When done testing:

```bash
docker stop hero-cockroach
docker rm hero-cockroach
rm -rf ~/hero-test
rm ~/.hero/credentials.json
```

The cloud server can stay running between sessions or be killed
(`Ctrl+C` in its terminal).

## Open questions to validate

These are things the test should answer that we don't yet know
empirically:

1. **Push payload size at scale.** A medium repo (~5k nodes) per
   push — is the JSON serialization fast enough? Watch for
   second+ push latency; if so, add streaming or chunking.
2. **Pull latency budget.** Goal: pull deltas fresh enough that a
   teammate's work is visible within 30 seconds of their push.
   Real test: push from Alice, immediately pull from Bob, time it.
3. **Conflict surface ergonomics.** `hero check conflicts` exists
   from phase 6a — does the graph-conflict surface plug into it
   cleanly, or do we need a dedicated `hero check graph-conflicts`?
4. **Large pull pagination.** PullGraphDelta caps at 5000 rows. If
   a fresh client onboards to a project with 10k+ nodes, does the
   client correctly paginate via `next_cursor`?

## What this unlocks once green

- **Phase 8 in production.** Cross-repo impact queries answer real
  questions ("who imports this symbol across the org?") with real
  data.
- **Multi-dev hot sessions.** A new dev cloning a repo gets the
  team's accumulated brief on first `hero resume` after pull —
  the original "cold session goes warm" promise.
- **Confidence to roll out the team server.** Until 7c is green,
  the team server is theoretically correct but unverified at the
  protocol level.

## References

- [graph-memory/spec.md](../graph-memory/spec.md) — overall design
- [graph-memory-federation/spec.md](../graph-memory-federation/spec.md)
  — multi-repo / multi-team architecture
- `internal/graph/sync.go` — client-side push/pull
- `cloud/api/graph.go` — server-side handlers
- `cloud/store/graph.go` — server-side store methods
- `cloud/store/migrations.go` (v5) — graph table schema

## Suggested first session prompt for a new agent picking this up

> "Read .hero/planning/features/graph-memory-7c-live-test/spec.md.
> I want to do the live phase 7c verification. Walk me through it
> step by step starting with prerequisites — pause after each
> major step (build binaries, start DB, migrations, OAuth,
> server start, login, dual workspaces, round 1) so I can confirm
> before moving on. If anything fails, debug it before continuing."
