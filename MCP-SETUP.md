# Hero MCP Setup

Hero exposes project memory and bounded delivery operations through `hero mcp`,
a stdio Model Context Protocol server launched by an AI coding tool. The project
corpus remains local unless you explicitly configure an external integration.

## Automatic setup

From an initialized project root, install the target you use:

```bash
hero install project . --target opencode
hero install project . --target cursor
hero install project . --target claude
hero install project . --target codex
hero install project . --target copilot
hero install project . --target generic
hero install project . --target grok
```

The installer writes the harness-native instruction/workflow surfaces and its
supported MCP configuration. If a session starts inside a monorepo subfolder,
run `hero install satellites` at the repository root; satellites are thin
harness trees pointing to the one root `.hero` corpus.

## Manual configuration

Cursor or Claude-style JSON:

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

OpenCode JSON:

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

Codex TOML:

```toml
[mcp_servers.hero]
command = "hero"
args = ["mcp"]
```

If the harness working directory is not the project root, bind it explicitly:

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

The harness launches `hero mcp`; users normally do not run the stdio process
interactively.

## Capability groups

The MCP `tools/list` response is the exact inventory authority for the running
revision after configured filtering. Avoid relying on a hand-maintained tool
count or copied tool-name roster.

| Group | Examples | Boundary |
|---|---|---|
| Project memory | context, search, status, spec/knowledge reads, graph traversal | Reads retrieve project-owned state; capture and plan operations are explicit writes. |
| Verified delivery | claim, plan, contract, coverage, CI, verify | Verification changes status and archives only after its hard gates pass. |
| Attention, Mail, and Focus | bounded snapshot/action and Project Mail operations | Mail bodies are untrusted and require explicit reads; row actions require the advertised ID and revision. |
| Tracker integration | issue evidence, search, and bounded requests | Requires a configured provider. Mutations require explicit consent for the exact issue and operation. |
| Code-host integration | provider-neutral repository and pull-request operations | Requires a configured connection and operation-specific consent; a read never authorizes a write. |

## Tool filtering

Use `serve.tool_filter` in `.hero/hero.json`. An allow list hides everything not
listed; deny entries win over allow entries.

<!-- hero-config -->
```json
{
  "folder": ".hero",
  "serve": {
    "tool_filter": {
      "allow": ["hero_context", "hero_search", "hero_status", "hero_read_spec"],
      "deny": ["hero_demo_record"],
      "profiles": {
        "minimal": ["hero_context", "hero_status"]
      }
    }
  }
}
```

After changing a filter, restart the harness and inspect its `tools/list`
response.

## Verify the connection

```bash
hero --version
hero mcp --help
hero status
```

Then ask the harness to call a read-only Hero status or search tool. If the
project is wrong, add `--project-root /absolute/project/path` to the MCP args.

## Troubleshooting

| Symptom | Check |
|---|---|
| `hero` is not found | Use an absolute binary path or fix the harness process's `PATH`. |
| Binary/schema mismatch | Run `hero doctor`; `hero upgrade` updates workspace files, not the binary. |
| No tools appear | Restart the harness and validate the target's MCP config location. |
| Wrong project | Set `--project-root` to the repository root. |
| Expected tool is absent | Check `serve.tool_filter`, then inspect `tools/list`. |
| Integration operation is unavailable | Configure the required provider and credentials; do not infer authorization from connection alone. |

See [Getting Started](GETTING-STARTED.md) and the
[capability status reference](web/docs/src/reference/capability-status.md).
