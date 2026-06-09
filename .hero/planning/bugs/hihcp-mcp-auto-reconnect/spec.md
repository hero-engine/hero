---
title: "Auto-Recover from MCP Server Disconnect Mid-Session"
slug: hihcp-mcp-auto-reconnect
type: bug
status: planning
domain: engineering
size: small
priority: high
created: 2026-06-09
tags: [hero-code, swift, mcp, reconnect, reliability, p1]
parent: hero-in-hero-code-parity
---

# Auto-Recover from MCP Server Disconnect Mid-Session

## Issue

When the Hero MCP server process dies unexpectedly mid-session,
`handleDisconnect` in `MCPManager.swift` sets the server state to `.errored` but
makes no attempt to restart the server. The session is permanently degraded for
the remainder of its lifetime -- all Hero MCP tool calls fail until the user
manually restarts.

Parent initiative: `hero-in-hero-code-parity`.

## Scope -- design inputs for `/design`

- On unexpected disconnect (server process died, not user-initiated shutdown),
  attempt automatic reconnection
- Use exponential backoff: first retry immediately, then 1s, 2s, 4s
- Three consecutive failures within a 30-second window trigger a cooldown period
  (e.g., 60 seconds)
- Surface the degradation state to the user: show a notification or status
  indicator when Hero MCP tools are unavailable due to disconnect
- On successful reconnection, re-register tools into the catalog
- Distinguish between expected disconnect (user stopped the server, app shutting
  down) and unexpected disconnect (process crash, SIGKILL)

**Files to touch:**
- `Engine/MCPManager.swift` -- reconnection logic, state machine extension

## Boundaries

- Do not change the first-turn readiness path (that is item 3)
- Do not add health-check pinging -- reconnect on disconnect is sufficient for v1
- Do not attempt to replay in-flight tool calls after reconnect

## Risks

- Reconnect storm: if the server has a startup bug, rapid reconnection attempts
  waste resources. The cooldown mitigates this.
- State leak: after reconnection, the tool catalog must be cleanly rebuilt, not
  appended to
- Race condition: a tool call arrives between disconnect detection and reconnect
  completion. Must return a clear "MCP server reconnecting" error, not hang.

## Validation

- Hero MCP server auto-restarts within 5 seconds after unexpected death
- Three consecutive failures trigger cooldown; no further reconnect attempts for
  the cooldown period
- User sees a notification when MCP tools are degraded
- After successful reconnect, Hero MCP tools work normally on the next tool call
- Expected disconnects (app shutdown) do not trigger reconnect attempts
