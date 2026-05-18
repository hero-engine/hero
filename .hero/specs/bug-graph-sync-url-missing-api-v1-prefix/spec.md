---
title: Graph Sync Client URL Missing `/api/v1` Prefix
slug: bug-graph-sync-url-missing-api-v1-prefix
type: bug
status: completed
tags: [graph, sync, cloud, urls, contract-drift]
created: 2026-05-01
horizon: now
---

## Symptom

All three sync round-trip tests in `internal/graph/sync_test.go` fail with `server 404: 404 page not found`:

- `TestPush_SendsTeamScopeNodesAndEdges`
- `TestPush_IsIdempotentViaSyncState`
- `TestPullAndApply_RoundTripsWithEdges`

## Root Cause

The sync client constructs request URLs without the documented `/api/v1` path prefix.

In [internal/graph/sync.go:18](internal/graph/sync.go:18) the contract is documented as:

```
//   POST /api/v1/graph/push?repo=<repo>
//   GET  /api/v1/graph/pull?repo=<repo>&since=<cursor>&include=team,unit
```

The test fixture at [internal/graph/sync_test.go:24](internal/graph/sync_test.go:24) registers handlers at the documented paths:

```go
mux.HandleFunc("POST /api/v1/graph/push", ...)
mux.HandleFunc("GET /api/v1/graph/pull", ...)
```

But the actual request construction at [internal/graph/sync.go:273](internal/graph/sync.go:273) and [internal/graph/sync.go:312](internal/graph/sync.go:312) uses the bare path:

```go
endpoint := c.ServerURL + "/graph/push?repo=" + url.QueryEscape(c.Repo)
endpoint := c.ServerURL + "/graph/pull?" + q.Encode()
```

Drift between code and contract; tests catch it but were left red.

## Suggested Fix Approach

In [internal/graph/sync.go](internal/graph/sync.go), prepend `/api/v1` to the two endpoint URL constructions:

```go
endpoint := c.ServerURL + "/api/v1/graph/push?repo=" + url.QueryEscape(c.Repo)
endpoint := c.ServerURL + "/api/v1/graph/pull?" + q.Encode()
```

Run `go test ./internal/graph/...` to confirm all three tests pass. No other call sites need to change — the `c.ServerURL` value is the bare host (e.g. `http://localhost:8080`).

## Acceptance Criteria

- WHEN `Push` is invoked with a configured `SyncClient` THE SYSTEM SHALL POST to `<server>/api/v1/graph/push?repo=<repo>`
- WHEN `Pull` is invoked with a configured `SyncClient` THE SYSTEM SHALL GET from `<server>/api/v1/graph/pull?...`
- WHEN the sync round-trip tests run THE SYSTEM SHALL pass `TestPush_SendsTeamScopeNodesAndEdges`, `TestPush_IsIdempotentViaSyncState`, and `TestPullAndApply_RoundTripsWithEdges`
- THE SYSTEM SHALL keep the documented endpoint contract and the actual request URLs in sync (a single source of truth would prevent recurrence — out of scope for this fix, worth a follow-up)

## Out of Scope

- Cloud-side handler implementation (these tests use a fake server)
- Authentication, rate limiting, or other endpoint behavior
- Centralising the URL construction (worth a follow-up but not required to close the bug)
