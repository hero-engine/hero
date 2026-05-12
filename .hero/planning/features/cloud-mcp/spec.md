---
title: Cloud MCP — Cross-Repo Knowledge Federation
type: feature
status: planning
tags: [cloud, mcp, knowledge, flagship]
created: 2026-04-12
parent: hero-cloud
depends-on: [cloud-api, cloud-sync]
horizon: next
smoke: deferred
---

## Goal

The flagship paid feature. A cloud-hosted MCP server that federates knowledge
(specs, conventions, decisions, rules) across all repos in an org. An AI tool
connected to Cloud MCP can search and retrieve context from any repo the user
has access to — not just the one they're currently working in.

## Why This Matters

Today, each repo has its own local MCP server with its own knowledge base.
When a developer is working on a frontend repo and needs to understand the
API contract defined in the backend repo's specs, they have to manually go find it.

Cloud MCP solves this: the AI tool asks "what's the auth API spec?" and
Cloud MCP returns it from whichever repo it lives in.

## Design

### MCP Protocol

Same JSON-RPC 2.0 over SSE (Server-Sent Events) for cloud transport — the
standard MCP remote transport. The cloud MCP server implements the same
tool interface as the local MCP server, plus additional cross-repo capabilities:

**Tools (superset of local MCP):**

| Tool | Description |
|---|---|
| `search_specs` | Full-text search across all repos in org |
| `get_spec` | Retrieve a specific spec by repo + slug |
| `list_specs` | List specs with filters (repo, type, status, tags) |
| `get_conventions` | Retrieve conventions from any repo |
| `get_decisions` | Retrieve decisions from any repo |
| `search_knowledge` | Search all knowledge base entries across repos |
| `get_activity` | Recent spec changes across the org |
| `get_context` | Aggregate context for a task (pulls relevant specs from multiple repos) |

### Access Control

Cloud MCP respects org membership. A user can only search repos they have
access to. The MCP server validates the JWT on connection and scopes all
queries to the user's authorized repos.

### Connection

AI tools (Cursor, Claude Code, OpenCode) connect to Cloud MCP via:
```json
{
  "heroCloud": {
    "command": "hero",
    "args": ["mcp", "--cloud"],
    "env": {}
  }
}
```

`hero mcp --cloud` acts as a local proxy: it reads the stored JWT, opens
an SSE connection to the cloud MCP endpoint, and bridges the local
stdio MCP protocol to the remote SSE transport.

### Latency Budget

- Search: < 500ms p95
- Get spec: < 200ms p95
- Context aggregation: < 1s p95

### Indexing

Cloud maintains a Postgres-backed full-text search index (tsvector) over
synced spec content. Index is updated on each sync push. For large orgs,
an async indexing queue ensures sync latency isn't affected.

## Changes

- Cloud service: `mcp/` package — cloud MCP server implementation
- Cloud service: `mcp/tools/` — tool handlers for cross-repo operations
- Cloud service: `search/` — Postgres FTS indexing and query
- CLI: `hero mcp --cloud` flag — local-to-cloud MCP bridge
- CLI: `internal/cloud/mcp_proxy.go` — SSE transport adapter

## Acceptance Criteria

- Cloud MCP server implements the full MCP tool interface
- AI tools can connect via `hero mcp --cloud` and search cross-repo
- Search returns results from all repos the user has access to
- Access control prevents cross-org data access
- Latency targets met (search < 500ms, get < 200ms)
- Works with Cursor, Claude Code, and OpenCode
- Free tier users cannot connect to Cloud MCP (graceful error with upgrade prompt)
