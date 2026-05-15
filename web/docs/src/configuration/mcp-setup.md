# MCP Setup

Hero exposes its corpus through `hero mcp`, a stdio Model Context
Protocol server launched by AI coding tools.

## Automatic Setup

```bash
hero install project . --target opencode
hero install project . --target cursor
hero install project . --target claude
hero install project . --target codex
hero install project . --target copilot
hero install project . --target generic
```

For sub-folder workspaces:

```bash
hero install project . --target cursor --workspace services/auth
```

The installer preserves user-owned files where possible and writes only
Hero-managed MCP blocks/config.

## Manual Config

OpenCode:

```json
{
  "mcp": {
    "hero": {
      "type": "local",
      "command": ["hero", "mcp"]
    }
  }
}
```

Cursor or Claude-style MCP config:

```json
{
  "mcpServers": {
    "hero": {
      "command": "hero",
      "args": ["mcp"]
    }
  }
}
```

Codex config:

```toml
[mcp_servers.hero]
command = "hero"
args = ["mcp"]
```

If the harness runs from a sub-folder, include the project root:

```json
{
  "mcpServers": {
    "hero": {
      "command": "hero",
      "args": ["mcp", "--project-root", "/path/to/project"]
    }
  }
}
```

## Available Tools

Hero currently exposes 41 MCP tools. The most common are:

| Tool | Purpose |
|---|---|
| `hero_resume` | Not an MCP tool; use CLI/slash `/resume`. |
| `hero_context` | File-aware conventions, past work, risks, and decisions. |
| `hero_search` | Full-text search over specs and knowledge. |
| `hero_ask` | Extractive Q&A. |
| `hero_list` / `hero_queue` | Spec lists and ready-work queue. |
| `hero_kickoff` | Return a spec's `## Kickoff` prompt. |
| `hero_read_spec` | Read full spec content. |
| `hero_claim` | Claim, release, or complete a spec. |
| `hero_plan` | Persist an execution plan. |
| `hero_code` | Code symbol/package intelligence. |
| `hero_why` / `hero_blocked` | Graph traversal queries. |
| `hero_expand` | Rehydrate compact tool responses. |

The full list is documented in [Server and MCP](../cli/server-and-mcp.md)
and in `internal/serve/mcp_tools_def.go`.

## Tool Filtering

Use `serve.tool_filter` in `.hero/hero.json`:

```json
{
  "serve": {
    "tool_filter": {
      "allow": ["hero_context", "hero_search", "hero_status", "hero_read_spec"],
      "deny": ["hero_demo_record"]
    }
  }
}
```

An `allow` list hides everything not listed. `deny` always wins.

## Verification

```bash
hero --version
hero mcp --help
hero status
```

Inside the AI tool, ask it to call `hero_status` or `hero_search`.

## Troubleshooting

| Symptom | Check |
|---|---|
| `hero: command not found` | Use an absolute binary path in MCP config or fix `PATH`. |
| No tools appear | Restart the harness and validate the config file location. |
| Wrong project | Add `--project-root /path/to/project`. |
| Expected tool hidden | Check `serve.tool_filter` in `.hero/hero.json`. |
