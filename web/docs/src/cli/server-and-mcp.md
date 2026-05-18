# Server and MCP

Hero has two server surfaces:

| Surface | Command | Use |
|---|---|---|
| MCP stdio | `hero mcp` | Launched by AI tools after `hero install`. |
| HTTP daemon | `hero serve` | Dashboard, API, file watcher, team mode, multi-project registry. |

## MCP

```bash
hero mcp
hero mcp --project-root /path/to/project
```

`hero mcp` is hidden from the default command list because users
normally do not run it manually. `hero install` writes MCP config for
the target harness.

Current MCP tools: `hero_context`, `hero_search`, `hero_status`,
`hero_check`, `hero_nudge`, `hero_list`, `hero_queue`, `hero_kickoff`,
`hero_knowledge`, `hero_read_spec`, `hero_ask`, `hero_anchor`,
`hero_pulse`, `hero_skill_run`, `hero_claim`, `hero_velocity`,
`hero_test_generate`, `hero_demo_record`, `hero_code`,
`hero_error_pattern`, `hero_enrich`, `hero_score`, `hero_diagnose`,
`hero_verify`, `hero_conflicts`, `hero_sequence`, `hero_warnings`,
`hero_insights`, `hero_contract`, `hero_plan`, `hero_impact`,
`hero_recap`, `hero_drift`, `hero_ci`, `hero_feed`, `hero_event`,
`hero_active`, `hero_coverage`, `hero_why`, `hero_blocked`,
`hero_expand`, `hero_snapshot`.

## Install MCP Config

```bash
hero install project . --target opencode
hero install project . --target cursor
hero install project . --target claude
hero install project . --target codex
hero install project . --target copilot
hero install project . --target generic
hero install project . --target cursor --workspace services/auth
```

Use `--dry-run` to preview and `hero upgrade` to refresh installed
agent/command/skill files and MCP registration after upgrading Hero.

## HTTP Daemon

```bash
hero serve
hero serve --port 8080
hero serve --no-ui
hero serve --no-watch
hero serve --add .
hero serve --add /path/to/project
hero serve --remove my-project
hero serve --list
```

Default address: `http://localhost:7437`.

Useful endpoints:

| Method | Path |
|---|---|
| `GET` | `/health` |
| `GET` | `/api/projects` |
| `GET` | `/api/{project}/status` |
| `GET` | `/api/{project}/specs` |
| `GET` | `/api/{project}/specs/:slug` |
| `GET` | `/api/{project}/search?q=` |
| `GET` | `/api/{project}/context?f=` |
| `GET` | `/api/{project}/check` |
| `GET` | `/api/{project}/knowledge` |
| `GET` | `/api/events` |

## Team Mode

```bash
hero serve --team --workers 2
hero serve --team --auth-token "$HERO_TEAM_TOKEN"
hero admin team status
hero admin users list
hero admin users add alice
```

Team mode enables job queue workers, shared activity, and server-side
coordination features.
