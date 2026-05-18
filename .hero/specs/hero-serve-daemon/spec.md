---
title: "hero serve — Local Daemon with MCP Server, File Watcher, and HTTP API"
slug: hero-serve-daemon
type: feature
status: completed
tags: [mcp, daemon, api, watch, serve]
created: 2026-04-12
relations:
  - target: hero-v2-system-design
    kind: parent
horizon: now
---

# hero serve — Local Daemon

## Problem

Hero's value depends on project knowledge reaching the AI model at the right time. Today this is manual — the user or agent must invoke `/context`, `/search`, `/check` explicitly. The model starts cold every session and only gets knowledge that someone thinks to inject upfront.

The "buddy model" vision (see note: buddy-model-architecture) requires continuous, automatic knowledge delivery. The model should be able to query project knowledge mid-reasoning, the index should always be fresh, and integrations should be live.

## Approach

A single `hero serve` command starts a local daemon that bundles four subsystems:

### 1. MCP Server (stdio + HTTP transport)

Exposes Hero's knowledge base as MCP tools that any AI agent can call during reasoning.

**Tools exposed:**

| Tool | Description | Maps to |
|---|---|---|
| `hero_context` | Get conventions, rules, past work, decisions for given files | `hero context imports --files` |
| `hero_search` | Full-text search across all specs and knowledge | `hero search` |
| `hero_check` | Run workspace health check | `hero check` |
| `hero_nudge` | Get nudge for files being worked on | `hero nudge --files` |
| `hero_status` | Get spec status by slug | `hero status <slug>` |
| `hero_list` | List specs with optional filters | `hero search --list` |
| `hero_knowledge` | List knowledge entries by type | `hero knowledge` |
| `hero_related` | Find specs related to a given spec | `hero graph <slug>` |

**Transport:**
- **stdio** — for agents that launch MCP servers as child processes (Claude Code, OpenCode, Cursor)
- **HTTP/SSE** — for agents that connect to running servers, and for the HTTP API

**Auto-registration:** `hero install` already configures agent tool files. It should also register the MCP server in the agent's MCP config:
- OpenCode: `mcp.json` or equivalent
- Cursor: `.cursor/mcp.json`
- Claude Code: `claude_desktop_config.json` / `.mcp.json`

The MCP server is auto-added when `hero install` runs. No manual MCP configuration needed.

### 2. File Watcher (replaces standalone `hero watch`)

The existing `watch` package runs as a subsystem of the daemon:
- Polls `.hero/` directory for changes (already built)
- Auto-reindexes changed specs immediately
- Emits events to the event stream
- No separate `hero watch` process needed (though `hero watch` remains as a standalone fallback)

### 3. HTTP API

Local REST API on a configurable port (default: `localhost:7437` — "HERO" on a phone keypad).

**Endpoints:**

| Method | Path | Description |
|---|---|---|
| GET | `/api/status` | Workspace summary (same as `hero dashboard`) |
| GET | `/api/specs` | List specs with query params for type, status, tag |
| GET | `/api/specs/:slug` | Get single spec detail |
| GET | `/api/search?q=...` | Full-text search |
| GET | `/api/context?files=...` | Context block for files |
| GET | `/api/check` | Health check results |
| GET | `/api/knowledge` | Knowledge entries |
| GET | `/api/events` | SSE event stream |
| GET | `/health` | Daemon health (for process managers) |

**No external dependencies.** Uses `net/http` from the standard library.

This powers:
- Future HTML dashboard (7H) — served directly by the daemon
- Editor extensions — VS Code, Neovim plugins can query the API
- Custom integrations — CI scripts, Slack bots, anything that speaks HTTP
- Hero Cloud sync — the cloud agent polls or receives pushes from the local API

### 4. Event Stream (SSE)

Server-Sent Events endpoint at `/api/events`:
- File change events (spec created/modified/deleted)
- Index rebuild events
- Health check results
- Spec status transitions

Consumers: editor extensions, cloud sync agent, HTML dashboard auto-refresh.

## Architecture

```
hero serve
├── MCP Server (stdio or HTTP transport)
│   ├── hero_context tool
│   ├── hero_search tool
│   ├── hero_check tool
│   ├── hero_nudge tool
│   ├── hero_status tool
│   ├── hero_list tool
│   ├── hero_knowledge tool
│   └── hero_related tool
├── File Watcher (polls .hero/, auto-reindexes)
├── HTTP API (localhost:7437)
│   ├── /api/status
│   ├── /api/specs[/:slug]
│   ├── /api/search
│   ├── /api/context
│   ├── /api/check
│   ├── /api/knowledge
│   ├── /api/events (SSE)
│   └── /health
└── Shared State
    ├── SQLite index (existing)
    └── Event bus (in-memory channel)
```

All subsystems share the same SQLite index. The event bus is a simple Go channel that the watcher publishes to and the SSE endpoint/MCP server subscribe to.

## Implementation Plan

### Phase 1: MCP Server (highest priority)
- `internal/serve/mcp.go` — MCP protocol handler (JSON-RPC 2.0 over stdio)
- Tool definitions mapping to existing Hero functions
- `hero mcp` — stdio mode for agent integration (hidden command, launched by AI tools)
- Update `hero install` to register MCP server in agent configs

### Phase 2: HTTP API
- `internal/serve/api.go` — HTTP handlers wrapping existing functions
- `internal/serve/server.go` — daemon lifecycle (start, graceful shutdown)
- `hero serve` — starts HTTP API + watcher

### Phase 3: Event Stream + Watch Integration
- `internal/serve/events.go` — SSE endpoint, in-memory event bus
- Wire watcher events into the event bus
- Auto-reindex on file changes

### Phase 4: Auto-registration
- Update `hero install` to add MCP server config for each agent
- `hero serve --background` — optional background/daemon mode

## Configuration

In `hero.json`:
```json
{
  "serve": {
    "port": 7437,
    "auto_watch": true,
    "mcp_transport": "stdio"
  }
}
```

## Constraints

- **No new Go dependencies.** MCP protocol is simple JSON-RPC 2.0 — implement with `encoding/json` and `net/http`.
- **No CGo.** Continue using `modernc.org/sqlite`.
- `hero serve` should start fast (<100ms) and use minimal resources when idle.
- Graceful shutdown on SIGINT/SIGTERM.
- Single process — no separate watcher or MCP processes.

## What This Enables

1. **Continuous knowledge injection** — models query Hero mid-reasoning, not just at session start
2. **Always-fresh index** — watcher keeps specs indexed in real-time
3. **Editor integration** — VS Code/Neovim plugins can show spec status, conventions, nudges
4. **Cloud foundation** — Hero Cloud syncs with the local daemon, not the filesystem
5. **HTML dashboard** — served directly by the daemon (phase 7H becomes trivial)
6. **Zero-config for users** — `hero install` sets up everything including MCP registration

## Cloud Implications

The local MCP server knows one project. Hero Cloud provides:
- **Cross-repo knowledge** — "how did team X solve this in repo Y?"
- **Team awareness** — "who else is working in this area across all repos?"
- **Shared conventions** — org-wide rules pushed to all local daemons
- **Analytics** — spec velocity, completion rates, convention compliance across the org

The local daemon is free forever. The cloud layer that aggregates across repos/teams is the paid product.

## Changes

- `internal/serve/mcp.go` — MCP protocol handler
- `internal/serve/api.go` — HTTP API handlers
- `internal/serve/server.go` — daemon lifecycle
- `internal/serve/events.go` — event bus and SSE
- `internal/cli/serve.go` — `hero serve` command
- `internal/cli/install.go` — MCP registration updates
- `internal/config/config.go` — serve config section
