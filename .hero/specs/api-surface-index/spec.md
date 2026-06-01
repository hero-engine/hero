---
title: API Surface Index — Endpoint-to-Handler Mapping
slug: api-surface-index
type: feature
status: completed
priority: low
tags: [code-intelligence, codescan, api, endpoints]
created: 2026-04-18
horizon: now
completed_at: 2026-05-18T19:25:38Z
---

## Goal

Index API endpoints (HTTP routes, gRPC services, GraphQL resolvers) with their handler functions so that bugs reported as "the `/api/foo` endpoint returns wrong data" can be immediately mapped to the relevant source code.

## Problem

When a bug is reported against an API endpoint, the first step is always finding which handler serves that route. In large codebases with middleware, routers, and indirection, this is non-trivial. The codescan package already parses symbols but doesn't understand routing patterns.

## Design

### Parser Extension

Add an additional parser pass in `internal/codescan/` that runs after symbol extraction. Each language parser (or a dedicated route parser) looks for common routing patterns:

- **Go**: `http.HandleFunc`, `mux.Handle`, `r.GET/POST/...` (chi, gin, echo, gorilla)
- **JS/TS**: `app.get/post/...` (Express), `router.get/...`, Next.js file-based routes
- **Python**: `@app.route`, `@router.get`, Django `urlpatterns`

### Output

Endpoints are stored in the scan index alongside symbols:

```go
type Endpoint struct {
    Method  string   // GET, POST, etc.
    Path    string   // /api/users/:id
    Handler string   // handleGetUser
    File    string   // internal/api/users.go
    Line    int
}
```

### Integration

The `hero_context` tool should accept endpoint path queries (e.g., `/api/users`) and return the handler chain. The `hero_code` tool could also resolve endpoint paths.

## Boundaries

- V1 covers HTTP REST endpoints only. gRPC and GraphQL are future work.
- Only static route definitions are indexed — dynamic route registration (e.g., routes built in a loop) won't be detected.
- Framework detection should be best-effort based on import analysis already done by codescan.
- No runtime verification — this is purely static analysis.
