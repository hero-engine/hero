---
title: "Mock Tracker Server — Offline Multi-Tracker HTTP Fake with Drift Injection"
slug: mock-tracker-server
type: feature
status: planning
priority: high
horizon: next
tags: [tracker, mock-server, fixtures, testing, round-trip]
created: 2026-06-27
received_from:
  peer_id: cd8dd06d-3df1-4878-a88f-24593dcbb4b3
  peer_alias_display: hero-code
  originator_slug: pm-workbench-tracker-validation
  call_id: 18bd0775ec35d74042d241cd94a2fd37
  mode: spec-out
  at: 2026-06-27T19:40:30Z
  at_commit: f1d4c1b
  reason: "Offline server lets the originator's round-trip harness run in CI without depending on a real GitHub/Jira/Linear/GitLab project."
relations:
  - target: tracker-fixtures
    kind: parent
  - target: gitlab-tracker-support
    kind: sibling
---

# Mock Tracker Server — Offline Multi-Tracker HTTP Fake with Drift Injection

## Provenance

Designed in response to `hero peer call --mode=spec-out` from peer
`hero-code` (peer_id `cd8dd06d-3df1-4878-a88f-24593dcbb4b3`),
related to originator spec `pm-workbench-tracker-validation`.

## Goal

Ship a single-binary, in-memory HTTP server that speaks the *subset*
of GitHub Issues, Jira Cloud, Linear, and GitLab APIs that hero's
tracker adapters actually call — enough to drive `hero sync import`,
`hero sync push --field`, `hero sync pull --field`, and
`hero spec set-owner` to completion without any real network
dependency. It is the offline twin of the four real trackers, and
also the test-double Hero's own integration tests run against.

## Non-Goal

Not a general-purpose mock for arbitrary clients. The contract is:
**every endpoint hero's adapters call** is implemented; **everything
else** returns 404. This keeps the surface honest — if we ship a new
adapter capability that calls a new endpoint, the mock fails fast and
we extend it.

## Design

### Binary and Package Layout

```
cmd/mock-tracker-server/
  main.go                # flag parsing, server bootstrap
internal/mocktracker/
  server.go              # http.Handler, mux, mode dispatch
  state.go               # in-memory data model (tracker-neutral)
  seed.go                # seed-file loader and validator
  github.go              # GitHub Issues API subset
  jira.go                # Jira Cloud API subset
  linear.go              # Linear GraphQL subset
  gitlab.go              # GitLab API subset (depends on
                         # gitlab-tracker-support's wire shape)
  admin.go               # /__admin endpoints (mutate, drift, rotate)
  pagination.go          # shared pagination + Link header helpers
  ratelimit.go           # configurable 429 / Retry-After injector
  fixtures/
    default-seed.json    # tiny default for hero's own tests
  *_test.go              # contract tests per tracker mode
```

The `cmd/` placement ensures `go build ./...` for hero proper does
not link it into the main hero binary. CI builds the mock server as
a separate artifact.

### State Model (Tracker-Neutral)

The server's internal state is one canonical model that each tracker
handler projects into its own wire shape:

```go
type State struct {
    Issues      map[string]*Issue      // keyed by global ID
    Epics       map[string]*Epic       // includes Jira Epic, Linear Project, GitHub sub-issue parent, GitLab Epic
    Milestones  map[string]*Milestone  // Jira fixVersion, GitHub milestone, GitLab milestone, Linear cycle (when used as release)
    Iterations  map[string]*Iteration  // Linear cycle, Jira sprint, GitLab iteration
    Labels      map[string]*Label
    Users       map[string]*User
    Comments    []*Comment
    // ID-rotation table — see /__admin/rotate-ids
    IDAliases   map[string]string
}
```

Each handler maps requests into State by global ID, and projects
State back into the requested wire format. This is what makes one
seed file work across all four tracker modes: the dataset is
authored once in the neutral shape and re-emitted as
`{"number": 42, ...}` for GitHub, `{"key": "ACME-42", ...}` for
Jira, `{"id": "ACME-42", ...}` for Linear, and
`{"iid": 42, "web_url": "..."}` for GitLab.

### Seed Format

Versioned JSON, owned-by-contract:

```jsonc
{
  "schema_version": 1,
  "project": {
    "name": "Acme Checkout",
    "slug": "acme-checkout"
  },
  "epics": [
    {"id": "epic:guest-checkout", "title": "Guest Checkout", ...},
    {"id": "epic:express-pay",    "title": "Express Pay",    ...},
    {"id": "epic:fraud",          "title": "Fraud Controls", ...},
    {"id": "epic:refunds",        "title": "Refunds",        ...},
    {"id": "epic:wallet",         "title": "Wallet",         ...}
  ],
  "milestones": [
    {"id": "release:v1.0", "title": "Acme Checkout v1.0", "due": "2026-09-01"}
  ],
  "iterations": [
    {"id": "sprint:2026-06-w4", "name": "Sprint 26.6.4", "start": "...", "end": "..."}
  ],
  "issues": [
    {"id": "ACME-100", "type": "story", "title": "Add Apple Pay sheet",
     "epic": "epic:express-pay", "milestone": "release:v1.0",
     "iteration": "sprint:2026-06-w4", "labels": ["frontend"],
     "assignee": "alice", "status": "open", "weight": 3,
     "description": "..."},
    {"id": "ACME-200", "type": "bug", "severity": "high",
     "title": "Refund webhook drops idempotency key on retry", ...}
  ],
  "users": [
    {"username": "alice", "email": "alice@acme.test", "display": "Alice K."}
  ]
}
```

Unknown top-level keys are rejected (forward-incompatible by
default) so seed-file drift is caught early. `schema_version: 1` is
the only accepted value in this feature; bumping it is a coordinated
change with hero-code.

The bundled `fixtures/default-seed.json` is tiny — 2 epics, 6
issues, 1 milestone — purely so hero's `go test ./internal/...` can
spin up the server without depending on hero-code's larger Acme
dataset.

### Mode and Routing

Server starts in **multi-mode**: a single binary serves four URL
prefixes, one per tracker:

```
http://localhost:PORT/github/...   → GitHub Issues API subset
http://localhost:PORT/jira/...     → Jira Cloud API subset
http://localhost:PORT/linear/...   → Linear GraphQL endpoint
http://localhost:PORT/gitlab/...   → GitLab API subset
http://localhost:PORT/__admin/...  → admin/control plane
```

Adapters point at the right prefix via `TrackerConfig.BaseURL`. One
running server can back integration tests for all four modes
concurrently because state is shared and projections are stateless.

A `--single-mode <github|jira|linear|gitlab>` flag is available for
debugging — same routing minus the prefix.

### Endpoints (Per Mode)

**GitHub mode** — subset called by `internal/tracker/github.go`:
- `GET /repos/:owner/:repo/issues` — list with `state`, `labels`,
  `per_page`, `page`. `Link` header for pagination.
- `GET /repos/:owner/:repo/issues/:id`
- `POST /repos/:owner/:repo/issues`
- `PATCH /repos/:owner/:repo/issues/:id` — title, body, labels,
  state, assignees
- `POST /repos/:owner/:repo/issues/:id/comments`
- `GET /search/issues?q=…` — minimal `repo:` + free-text matching

**Jira mode** — subset called by `internal/tracker/jira.go`:
- `GET /rest/api/3/search` — JQL parsing is a hand-rolled mini-parser
  for the clauses hero actually emits (`project = X`, `status = …`,
  `assignee = …`, `sprint in openSprints()`, `ORDER BY`). Unknown
  clauses return 400 with a clear error.
- `GET /rest/api/3/issue/:key`
- `POST /rest/api/3/issue`
- `PUT /rest/api/3/issue/:key` — fields + transitions
- `GET /rest/api/3/field` — for the field cache discovery flow
- `POST /rest/api/3/issue/:key/transitions`
- `POST /rest/api/3/issue/:key/comment`
- `GET /rest/agile/1.0/board/:id/sprint`, `/sprint/:id/issue`

**Linear mode** — subset called by `internal/tracker/linear.go`:
- `POST /graphql` — a hand-rolled GraphQL handler that matches the
  exact query shapes the adapter sends (issue queries, mutations,
  cycle queries). Not a general GraphQL engine; the handler dispatches
  on `operationName` + a hash of the query string for stability.

**GitLab mode** — subset called by the new
`internal/tracker/gitlab.go` (sibling spec):
- `GET /api/v4/projects/:id/issues` — with `Link` headers
- `GET/POST/PUT /api/v4/projects/:id/issues/:iid`
- `GET /api/v4/groups/:id/epics`, `/epics/:id`
- `GET /api/v4/projects/:id/milestones`
- `GET /api/v4/projects/:id/iterations`
- `POST /api/v4/projects/:id/issues/:iid/notes`

### Pagination Contract

All list endpoints honor `per_page` (default 20, max 100) and emit
the next-page link in the form each tracker expects:

| Tracker | Pagination signal |
|---|---|
| GitHub | `Link: <…?page=2>; rel="next"` |
| Jira | response body `startAt` / `total` / `isLast` |
| Linear | GraphQL `pageInfo { hasNextPage, endCursor }` |
| GitLab | `Link: <…?page=2>; rel="next"` |

Tests cover **pagination to exhaustion**: seed with >100 issues,
request `per_page=20`, confirm the client follows pagination until
the last page returns no `next` link.

### 429 / Retry-After Injection

Admin endpoint sets a 429-on-next-request rule:

```
POST /__admin/inject-429
  {"path_glob": "/github/repos/*/issues*", "retry_after_seconds": 1, "count": 1}
```

The configured count of matching requests is answered with
`429 Too Many Requests` and `Retry-After: 1`, after which normal
serving resumes. This drives the `doWithRetry` path in
`internal/tracker/fielderror.go`.

### Drift Injection

```
POST /__admin/mutate
  {"id": "ACME-100", "field": "title", "value": "Mutated externally"}
```

Mutates state out-of-band — no API call from the adapter, just a
direct State write. Subsequent `GET` requests reflect the new value,
which is exactly the shape `hero sync pull` needs to detect drift.

```
POST /__admin/rotate-ids
  {"id": "ACME-100", "new_id": "ACME-100"}   # same global ID, new IIDs
```

Rotates the per-tracker IIDs (GitHub number, GitLab iid, Linear
identifier) while keeping the global ID stable, validating the
stable-external-ID contract.

### Authentication

Accepts any non-empty `Authorization` header. The server has a
`--require-token <token>` flag for tests that want to assert
401-on-bad-token behavior; without the flag, any token is accepted.
This keeps the production OAuth surface out of the mock entirely.

### Logging and Diagnostics

`--log-requests` emits each request as one JSON line on stderr,
including the matched route and the response status. Hero's
integration tests assert against this stream to confirm the right
endpoints were called in the right order.

## Acceptance Criteria

1. **AC-1: Binary builds and runs** — `go build
   ./cmd/mock-tracker-server/...` produces a binary; `mock-tracker-server
   --port 0 --seed fixtures/default-seed.json` starts and reports the
   chosen port on stdout.
2. **AC-2: GitHub round-trip** — pointing `hero` at
   `http://…/github` with a github tracker config completes
   `hero sync import`, `hero sync push --field title`, `hero sync
   pull --field title`, `hero spec set-owner` against the seeded
   data.
3. **AC-3: Jira round-trip** — same as AC-2 against `/jira`. JQL
   subset accepts the queries hero emits today (verified by grep over
   `internal/tracker/jira.go` for `JQL` constants).
4. **AC-4: Linear round-trip** — same as AC-2 against `/linear`.
   GraphQL handler dispatches all operations
   `internal/tracker/linear.go` actually sends.
5. **AC-5: GitLab round-trip** — same as AC-2 against `/gitlab`.
   Depends on `gitlab-tracker-support` shipping.
6. **AC-6: Pagination to exhaustion** — seed with 150 issues,
   request `per_page=20`, confirm hero's adapter retrieves all 150
   across 8 pages with no duplicates and no missing items. Test exists
   for each tracker mode.
7. **AC-7: 429 backoff** — `POST /__admin/inject-429` then a `hero
   sync push --field`; observe a single 429 followed by a successful
   retry. `Retry-After` is honored (capped by `maxRetryAfter`).
8. **AC-8: Drift detection** — `hero sync import` seeds local spec;
   `POST /__admin/mutate` rewrites the title server-side; `hero sync
   pull --field title` overwrites local and reports drift in a
   `--dry-run` precheck.
9. **AC-9: ID rotation** — `POST /__admin/rotate-ids` changes IIDs;
   subsequent `hero sync pull` still finds the issue by global ID
   without re-importing. Tests this for github/gitlab (IID-based) and
   jira/linear (key-based).
10. **AC-10: Seed-format strictness** — server rejects
    `schema_version: 2`, unknown top-level keys, and unknown issue
    fields with descriptive errors. Test fixtures cover all three.
11. **AC-11: Test isolation** — `go test ./internal/mocktracker/...`
    starts one server per test on `:0`, no port conflicts, no
    process leaks. Standard `t.Cleanup` discipline.
12. **AC-12: No prod-path contamination** — `grep -r
    'mocktracker\|mock-tracker-server' internal/cli internal/tracker
    internal/serve` returns no hits. The mock server is invisible
    from the hero binary.

## Open Questions

- **GraphQL hand-rolling vs library.** Linear's API is GraphQL. A
  third-party server library would be more "correct" but adds a
  dependency just for testing. **Decision:** hand-roll an
  `operationName`-dispatched handler. The set of queries hero issues
  is small and stable; a library is a hammer for a screw.
- **Concurrency.** Multi-mode means parallel requests from
  hero-code's CI lanes could hit the same State. **Decision:** one
  `sync.RWMutex` around State. Writes are admin-driven and rare;
  reads dominate. Contention is a non-issue at fixture scale (~50
  issues).
- **Persistence between runs.** **Decision:** none. Always start
  from seed. Determinism is the whole point.

## Completion Ledger

| AC | File | Status |
|----|------|--------|
| AC-1 | `cmd/mock-tracker-server/main.go`, `internal/mocktracker/server.go` | — |
| AC-2 | `internal/mocktracker/github.go`, `internal/mocktracker/github_test.go` | — |
| AC-3 | `internal/mocktracker/jira.go`, `internal/mocktracker/jira_test.go` | — |
| AC-4 | `internal/mocktracker/linear.go`, `internal/mocktracker/linear_test.go` | — |
| AC-5 | `internal/mocktracker/gitlab.go`, `internal/mocktracker/gitlab_test.go` | — |
| AC-6 | `internal/mocktracker/pagination.go`, `internal/mocktracker/pagination_test.go` | — |
| AC-7 | `internal/mocktracker/ratelimit.go`, `internal/mocktracker/admin.go` | — |
| AC-8 | `internal/mocktracker/admin.go` | — |
| AC-9 | `internal/mocktracker/admin.go`, `internal/mocktracker/state.go` | — |
| AC-10 | `internal/mocktracker/seed.go`, `internal/mocktracker/seed_test.go` | — |
| AC-11 | (cross-cutting) | — |
| AC-12 | (cross-cutting) | — |

## Dependencies

- `gitlab-tracker-support` (sibling) — required for AC-5. AC-1
  through AC-4 and AC-6 through AC-12 can be developed and tested in
  parallel; AC-5 is the last to land.
- No other production-code dependencies on the hero side.
- Originator-side dependency: hero-code owns the Acme Checkout seed
  file and the live-validation harness. This spec ships the binary
  and the seed-format contract; hero-code authors the data.
