---
title: "Multi-Project Serve — Single Daemon for All Local Projects"
slug: multi-project-serve
type: feature
status: completed
tags: [serve, daemon, multi-project, mcp]
created: 2026-04-12
parent: hero-serve-daemon
horizon: now
completed_at: 2026-05-18T19:25:38Z
---

## Progress

- [x] Spec written
- [x] Project registry (~/.hero/projects.json) — read/write/list (11 tests)
- [x] Server refactored to load multiple project indexes (7 tests)
- [x] HTTP API routes namespaced to /api/{project}/... (17 tests)
- [x] MCP unchanged — stdio mode is already single-project by cwd (correct design)
- [x] hero serve --add / --remove / --list subcommands (4 CLI tests)
- [x] hero install auto-registers project in registry (6 tests)
- [x] SSE events include project field, ?project= filter (8 tests)
- [x] Tests passing — 70 serve tests, all green
- [x] Full suite green — all 10 packages pass

Started: 2026-04-12
Completed: 2026-04-12

## Problem

`hero serve` currently manages one project per process. On a machine with multiple Hero-enabled repos, the user would need to start multiple daemons, manage port conflicts, and configure each agent to talk to the right port. This is unacceptable UX — the daemon should be invisible infrastructure, not something you babysit.

## Goal

A single `hero serve` process manages all registered projects on the machine. One port, one process, all projects. MCP stdio mode continues to work per-project (cwd-based). The HTTP API namespaces routes by project. Events include project identity.

## Approach

### Project Registry

A global registry at `~/.hero/projects.json` tracks all known projects:

```json
{
  "projects": {
    "hero": {
      "path": "/Users/chet-bellows/projects/personal/repository/hero",
      "registered": "2026-04-12T10:00:00Z"
    },
    "webapp": {
      "path": "/Users/chet-bellows/projects/work/webapp",
      "registered": "2026-04-12T10:05:00Z"
    }
  }
}
```

Project slug is derived from the directory name (or configurable in `hero.json`). The registry is append-only during normal operation.

**Registration methods:**
- `hero serve --add .` — register current directory
- `hero serve --add /path/to/repo` — register any directory
- `hero serve --remove hero` — unregister by slug
- `hero serve --list` — show all registered projects
- `hero install` — auto-registers the project if not already present

### HTTP API Changes

Routes gain a `{project}` segment:

| Before | After |
|---|---|
| `GET /api/status` | `GET /api/{project}/status` |
| `GET /api/specs` | `GET /api/{project}/specs` |
| `GET /api/specs/:slug` | `GET /api/{project}/specs/:slug` |
| `GET /api/search?q=...` | `GET /api/{project}/search?q=...` |
| `GET /api/context?files=...` | `GET /api/{project}/context?files=...` |
| `GET /api/check` | `GET /api/{project}/check` |
| `GET /api/knowledge` | `GET /api/{project}/knowledge` |
| `GET /api/events` | `GET /api/events` (all projects, filtered by ?project=) |

New endpoints:
- `GET /api/projects` — list all registered projects with path, spec count, last indexed
- `GET /health` — unchanged, daemon-level

### MCP Changes

MCP stdio mode already works per-project: the agent launches `hero mcp` from the repo directory, so cwd determines the project. No routing needed.

For HTTP-transported MCP (future), tool calls would include an optional `project` parameter. If omitted, the server infers from context or returns an error asking which project.

Each MCP tool gains an optional `project` string input:
```json
{
  "name": "hero_search",
  "inputSchema": {
    "properties": {
      "query": { "type": "string" },
      "project": { "type": "string", "description": "Project slug (optional in stdio mode)" }
    }
  }
}
```

### Server Internals

The `Server` struct holds a map of project contexts:

```go
type ProjectContext struct {
    Slug    string
    Path    string
    DB      *index.DB
    Watcher *watch.Watcher
}

type Server struct {
    projects map[string]*ProjectContext
    bus      *EventBus
    // ...
}
```

Each project gets its own index and watcher. The event bus is shared — events carry a `Project` field. On startup, the server loads all projects from the registry, opens their indexes, and starts their watchers.

### SSE Events

Events gain a `project` field:

```json
{
  "type": "spec.modified",
  "project": "hero",
  "data": { "slug": "multi-project-serve", "path": "..." },
  "timestamp": "2026-04-12T10:30:00Z"
}
```

The `/api/events` endpoint accepts `?project=hero` to filter events to a single project.

### Configuration

Global daemon config at `~/.hero/config.json`:
```json
{
  "serve": {
    "port": 7437
  }
}
```

Per-project config remains in `{repo}/.hero/hero.json` (auto_watch, etc.).

## Changes

- `internal/serve/server.go` — refactor to multi-project (ProjectContext map, per-project indexes)
- `internal/serve/api.go` — namespaced routes, /api/projects endpoint
- `internal/serve/mcp.go` — optional project parameter on tools
- `internal/serve/events.go` — add Project field to Event
- `internal/serve/registry.go` — new: project registry read/write
- `internal/install/install.go` — auto-register project in registry
- `internal/cli/serve.go` — --add, --remove, --list flags

## Boundaries

- No cross-project queries in this phase (e.g., "search across all projects"). Each project is isolated. Cross-project is a cloud feature.
- No daemon auto-start / launchd / systemd integration yet.
- No auth on the HTTP API — it's localhost-only.
- MCP stdio mode remains single-project (one process per agent session).

## Validation

- Start daemon, register 3 projects, verify all 3 are served on namespaced routes
- MCP stdio mode still works correctly (cwd-based, single project)
- Port override works via flag and config
- SSE events from different projects are distinguishable
- Registering a project that's already registered is idempotent
- Removing a project stops its watcher and closes its index
- hero install in a new repo auto-registers it
