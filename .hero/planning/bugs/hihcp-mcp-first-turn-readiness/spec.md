---
title: "Gate First Turn on Hero MCP Readiness"
slug: hihcp-mcp-first-turn-readiness
type: bug
status: handed_off
domain: engineering
size: small
priority: high
created: 2026-06-09
tags: [hero-code, swift, mcp, race-condition, first-turn, p1]
parent: hero-in-hero-code-parity
---

# Gate First Turn on Hero MCP Readiness

## Issue

`ensureReady` on MCP servers is only called from `callTool()`, meaning the tool
catalog constructed for the first turn may omit all Hero MCP tools
(`hero_search`, `hero_list`, `hero_read_spec`, etc.). The model's first turn
sees no Hero tools and falls back to grep/read against `.hero/` directories,
which is slow and unreliable.

Parent initiative: `hero-in-hero-code-parity`.

## Scope -- design inputs for `/design`

- Gate tool catalog construction on MCP readiness with a short timeout (3s
  recommended)
- Call `ensureReady` (or equivalent) on all configured MCP servers before
  building the tool catalog for the first turn
- On timeout, proceed with whatever tools are available and log a warning
- Do not block the UI -- this should be a background wait with timeout

**Files to touch:**
- `Engine/AgentLoop.swift` -- tool catalog construction path
- `Engine/MCPManager.swift` -- readiness check, possibly expose a
  `waitForReady(timeout:)` method

## Boundaries

- Do not change how MCP tools are added mid-session (that path already works via
  `callTool` -> `ensureReady`)
- Do not add retry logic here -- that belongs in item 4 (auto-reconnect)

## Risks

- Timeout too short: Hero MCP server may not be ready in 3s on slow machines
- Timeout too long: first-turn latency is visible to the user
- Edge case: MCP server configured but not installed (should degrade gracefully,
  not block)

## Validation

- First turn always includes Hero MCP tools when the Hero MCP server is
  configured and running
- Timeout degrades gracefully: first turn proceeds without Hero MCP tools and
  they appear on the second turn after `callTool` triggers `ensureReady`
- No visible delay for users who do not have a Hero MCP server configured

## Handoff Trail

- 2026-06-24T18:01:14Z — out → hero-code (peer_id: cd8dd06d-3df1-4878-a88f-24593dcbb4b3)
  mode: async-drop
  originating_spec: hihcp-mcp-first-turn-readiness
  peer_spec: hero-code/hihcp-mcp-first-turn-readiness
  at_commit: 2f774b7
  reason: "Targets the hero-code Swift agent loop (Engine/MCPManager.swift / ensureReady). Pre-release triage of the hero CLI confirmed no matching Go code exists here."

