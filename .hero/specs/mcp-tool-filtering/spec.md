---
title: MCP Tool Filtering — Route and Control Which Tools Agents Can Use
type: feature
status: completed
milestone: v0.2
tags: [mcp, tools, filtering, routing, config, security]
created: 2026-04-12
relations:
  - target: hero-serve-daemon
    kind: extends
  - target: cloud-mcp
    kind: related
horizon: now
---

## Goal

When `hero serve` exposes Hero as an MCP server, it currently offers all 8 tools to all agents unconditionally. As agent workflows get more complex — especially with multi-model setups, role-based tool access, and security-sensitive environments — teams need control over which tools are available to which agents or contexts.

This spec defines the MCP tool filtering, routing, and configuration layer: a way to declare which tools are enabled, scoped, or restricted per agent role, per environment, or per command.

## Problem

Unfiltered tool exposure causes several real problems:

1. **Security / blast radius** — a code-generation agent should not call `hero_check` or trigger health checks; a read-only research agent should not have access to tools that could cause side effects
2. **Context contamination** — if an agent can call all 8 Hero tools, it may over-query and flood its context window with irrelevant results
3. **Role coherence** — a `review` model should have different tool access than an `execution` model (see `model-role-config`)
4. **Multi-tenant serve** — when `hero serve` exposes an HTTP MCP endpoint, different clients may need different tool subsets

## Design

### Tool Manifest with Tags

Each MCP tool in `hero serve` gets a set of capability tags that describe what it does and what level of access it requires:

| Tool | Tags |
|---|---|
| `hero_context` | `read`, `context`, `core` |
| `hero_search` | `read`, `search`, `core` |
| `hero_check` | `read`, `health`, `diagnostic` |
| `hero_nudge` | `read`, `context`, `core` |
| `hero_status` | `read`, `status` |
| `hero_list` | `read`, `search` |
| `hero_knowledge` | `read`, `knowledge` |
| `hero_related` | `read`, `graph` |

All current tools are read-only — there are no write tools yet. Tags allow filtering by capability class rather than tool name, so adding new tools doesn't break existing configs.

### Filter Config in `hero.json`

```json
{
  "serve": {
    "mcp": {
      "tools": {
        "default": "all",
        "roles": {
          "execution": ["hero_context", "hero_nudge", "hero_search"],
          "review": ["hero_search", "hero_knowledge", "hero_related", "hero_check"],
          "research": "all",
          "design": ["hero_context", "hero_search", "hero_knowledge", "hero_related"]
        },
        "deny": []
      }
    }
  }
}
```

**`default`**: `"all"` | `"none"` | array of tool names — used when no role matches  
**`roles`**: per-role tool allowlists (integrates with `model-role-config` role names)  
**`deny`**: global blocklist that overrides role allowlists — useful for disabling a tool org-wide

### Tag-Based Filtering

Instead of naming tools explicitly, filter by capability tags:

```json
{
  "serve": {
    "mcp": {
      "tools": {
        "roles": {
          "execution": { "tags": ["core"] },
          "review": { "tags": ["read", "diagnostic"] },
          "research": { "tags": ["read"] }
        }
      }
    }
  }
}
```

Tag-based config is more stable — adding a new `core` tool automatically includes it for `execution` agents without config changes.

### Runtime Role Resolution

When an MCP client connects, the server needs to know which role it is to apply the right filter. Role is determined by:

1. **Connection parameter** — MCP client passes `x-hero-role` header (HTTP transport) or `role` init param (stdio transport)
2. **Client ID** — named clients in `hero.json` map to roles:
   ```json
   {
     "serve": {
       "mcp": {
         "clients": {
           "cursor": "execution",
           "opencode": "execution",
           "research-agent": "research"
         }
       }
     }
   }
   ```
3. **Default** — falls back to `tools.default` if no role can be determined

### Per-Tool Rate Limiting (Optional)

High-frequency tool calls can be expensive. Optional per-tool call rate limits:

```json
{
  "serve": {
    "mcp": {
      "rate_limits": {
        "hero_search": { "per_minute": 60 },
        "hero_context": { "per_minute": 120 },
        "hero_check": { "per_minute": 10 }
      }
    }
  }
}
```

Rate limiting is enforced per-connection, not globally. Exceeded limits return a structured MCP error rather than dropping the connection.

### Tool Introspection

The MCP `tools/list` response should include the tag metadata so clients can understand tool capabilities:

```json
{
  "tools": [
    {
      "name": "hero_context",
      "description": "Get conventions, rules, past work, decisions for given files",
      "tags": ["read", "context", "core"],
      "inputSchema": { ... }
    }
  ]
}
```

This allows MCP clients to do their own capability-based tool selection if needed.

### Environment-Based Profiles

Common environments get named profiles:

```json
{
  "serve": {
    "mcp": {
      "profile": "strict"
    }
  }
}
```

| Profile | Description |
|---|---|
| `full` | All tools, no restrictions (default for local dev) |
| `strict` | Only `core`-tagged tools |
| `readonly` | All `read`-tagged tools, no `diagnostic` |
| `ci` | `hero_check` and `hero_status` only |

Profiles are overridden by explicit `tools` config if both are present.

## Changes

- `internal/serve/mcp.go` — tool registry with tags, filter middleware applied at `tools/list` and `tools/call`
- `internal/serve/mcp_filter.go` — filter config parsing, role resolution, tag matching
- `internal/serve/mcp_ratelimit.go` — per-tool rate limiting (optional, configurable)
- `internal/config/config.go` — `serve.mcp.tools` config section

## Acceptance Criteria

- `tools.default: "all"` serves all tools when no role config is present (no regressions)
- Role-based tool lists restrict available tools for matching roles
- Tag-based config works as an alternative to explicit tool name lists
- Client ID → role mapping resolves correctly for named clients
- `tools/list` response includes tool tags
- `tools.deny` overrides role allowlists
- Named profiles (`full`, `strict`, `readonly`, `ci`) work out of the box
- Rate limiting (if enabled) returns structured MCP errors on excess, does not drop connections
- All filtering is applied without modifying the underlying tool implementations

## Boundaries

- Does **not** add write tools — all filtering here applies to the existing read-only tool set
- Does **not** implement authentication / JWT validation — that's a cloud concern (`cloud-mcp`)
- Role resolution depends on client cooperation (passing role header/param) — no cryptographic enforcement for local use
