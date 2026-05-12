# Hero MCP Setup

Hero exposes project memory to AI coding tools through `hero mcp`, a
stdio MCP server. Most users should configure it with:

```bash
hero install project . --target opencode
hero install project . --target cursor
hero install project . --target claude
hero install project . --target codex
hero install project . --target copilot
hero install project . --target generic
```

For sub-folder harness sessions:

```bash
hero install project . --target cursor --workspace services/auth
```

`hero mcp` is launched by the tool. You usually do not run it manually,
though `hero mcp --help` is available for verification.

## Manual Config

Cursor/Claude-style:

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

Codex:

```toml
[mcp_servers.hero]
command = "hero"
args = ["mcp"]
```

Use `["mcp", "--project-root", "/path/to/project"]` when the tool's
working directory is not the project root.

## Current Tool Set

Hero currently advertises 41 MCP tools:

`hero_context`, `hero_search`, `hero_status`, `hero_check`,
`hero_nudge`, `hero_list`, `hero_queue`, `hero_kickoff`,
`hero_knowledge`, `hero_read_spec`, `hero_ask`, `hero_anchor`,
`hero_pulse`, `hero_skill_run`, `hero_claim`, `hero_velocity`,
`hero_test_generate`, `hero_demo_record`, `hero_code`,
`hero_error_pattern`, `hero_enrich`, `hero_score`, `hero_diagnose`,
`hero_verify`, `hero_conflicts`, `hero_sequence`, `hero_warnings`,
`hero_insights`, `hero_contract`, `hero_plan`, `hero_impact`,
`hero_recap`, `hero_drift`, `hero_ci`, `hero_feed`, `hero_event`,
`hero_active`, `hero_coverage`, `hero_why`, `hero_blocked`,
`hero_expand`.

## Troubleshooting

```bash
hero --version
hero mcp --help
hero status
```

If tools do not appear, restart the harness, check the config file
location, and confirm the project has been initialized with `hero init`.

Tool filtering is controlled by `serve.tool_filter` in `.hero/hero.json`.
