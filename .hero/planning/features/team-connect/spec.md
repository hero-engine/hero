---
title: Team Connect — CLI Registration with Team Server
type: feature
status: planning
priority: P1
tags: [team, connect, cli, registration]
created: 2026-04-25
relations:
  - target: hero-team-server
    kind: parent
  - target: team-oauth
    kind: related
horizon: next
smoke: deferred
---

## Goal

Let developers register their local Hero installation with a team
server so jobs, sessions, and events flow through the shared
infrastructure. `hero connect team <url>` is the one-time setup
command that stores the server URL and credentials locally.

## Problem

The team server runs, but individual developers need a way to:
1. Discover and connect to it
2. Authenticate (token or OAuth)
3. Have their `hero run` commands routed through the team queue
4. Have their MCP sessions register with the team server
5. See team-wide job status from their local CLI

Today `hero run` executes locally. After connecting, it should submit
to the team queue instead (when the server is reachable) so jobs get
tracked centrally, usage is attributed, and the team dashboard shows
everything.

## Design

### `hero connect team`

```bash
hero connect team https://hero.internal:7437
# If token auth: prompts for token
# If OAuth: opens browser for login
# Stores connection in ~/.hero/team.json
# Tests connection: "Connected to hero-team at hero.internal:7437 (3 jobs running)"
```

### Stored config

`~/.hero/team.json`:
```json
{
  "url": "https://hero.internal:7437",
  "token": "jwt-or-bearer-token-here",
  "user": "alice",
  "connected_at": "2026-04-25T12:00:00Z"
}
```

### `hero disconnect team`

```bash
hero disconnect team
# Removes ~/.hero/team.json
# "Disconnected from hero-team"
```

### `hero team status`

```bash
hero team status
# Connected to: https://hero.internal:7437
# User: alice
# Running jobs: 2
# Queued jobs: 1
# Active sessions: 3
# Your usage today: $4.50
```

### hero run integration

When a team connection exists, `hero run` behavior changes:

1. Check if team server is reachable
2. If yes: submit job to team queue via POST /api/jobs
3. If no: fall back to local execution (current behavior)

This is transparent — the user runs the same command, the routing
is automatic.

### MCP session auto-registration

When `hero mcp` starts and a team connection exists, it POST to
`/api/sessions` to register the active session. Heartbeats every
5 minutes update `last_seen`. On exit, DELETE the session.

### CLI changes

| Command | What it does |
|---|---|
| `hero connect team <url>` | Register with team server |
| `hero disconnect team` | Remove team connection |
| `hero team status` | Show team server status and your usage |
| `hero team usage` | Detailed usage breakdown |

## Changes

- `internal/cli/connect_team.go` — connect/disconnect team subcommands
- `internal/cli/team.go` — `hero team status`, `hero team usage`
- `internal/config/team.go` — load/save ~/.hero/team.json
- `internal/runner/runner.go` — check for team connection, submit to server if available
- `internal/cli/mcp.go` — register MCP session with team server on start

## Acceptance Criteria

- WHEN `hero connect team <url>` is called with a valid server THE SYSTEM SHALL store the connection config and confirm with server status
- WHEN `hero connect team <url>` is called with an unreachable server THE SYSTEM SHALL exit with an error
- WHEN `hero disconnect team` is called THE SYSTEM SHALL remove the stored connection
- WHEN `hero run` is called and a team connection exists THE SYSTEM SHALL submit the job to the team server queue instead of running locally
- WHEN `hero run` is called and the team server is unreachable THE SYSTEM SHALL fall back to local execution with a warning
- WHEN `hero mcp` starts and a team connection exists THE SYSTEM SHALL register the session with the team server
- WHEN `hero team status` is called THE SYSTEM SHALL display connection info, running jobs, active sessions, and user usage

## Boundaries

- Does **not** require a team connection to function — Hero works fully solo
- Does **not** sync specs or knowledge through the team server — git handles that
- Does **not** proxy MCP tool calls through the team server — only sessions and jobs
- Does **not** support multiple team server connections — one at a time
