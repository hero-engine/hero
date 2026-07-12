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
`hero_expand`, `hero_snapshot`, `hero_synthesize`.

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

### Harness-native root instruction files

`hero install` is **harness-native**: each target gets only the root
instruction file it natively reads — nothing else.

| Target | Root instruction file |
|---|---|
| `claude` | `CLAUDE.md` |
| `codex`, `opencode`, `cursor`, `copilot`, `generic` | `AGENTS.md` |

- `hero install --target claude` writes `CLAUDE.md` only — it does **not**
  litter an `AGENTS.md` no Claude session reads.
- `hero install --target <non-claude>` writes `AGENTS.md` only.
- Installing multiple targets where one is Claude produces **both**
  `CLAUDE.md` and `AGENTS.md`, each carrying the same Hero-managed block.

Both files use the same versioned managed region (`<!-- hero:managed-start
… -->` / `<!-- hero:managed-end -->`); content you write **outside** the
markers is preserved byte-for-byte on every re-install.

### Persisted target set

Every project-mode install records the installed target set in
`.hero/install-state.json` (`targets`). `hero upgrade` reads it and
regenerates the managed region **only** in the native instruction files of
previously-installed targets:

- If Claude was never a target, upgrade never creates a `CLAUDE.md`.
- If Claude was a target, upgrade regenerates `CLAUDE.md`'s managed region.

A repo installed before this state existed is **backfilled** on the next
upgrade: Hero infers the prior target set from the harness content
directories (`.claude/`, `.codex/`, …) plus any Hero-managed instruction
file, persists it, and proceeds.

### Migration safety and pruning orphans

Upgrading a repo installed under the old "always both files" model is
non-destructive: **Hero never deletes your `AGENTS.md` or `CLAUDE.md` by
default.** An instruction file whose target is not in the resolved set has
its managed region kept current (so it doesn't rot), but is never removed.

To remove a leftover phantom file, opt in explicitly:

```bash
hero install project . --target claude --prune-orphaned-instruction-files
hero upgrade --prune-orphaned-instruction-files
```

Even with the flag, a file is deleted **only** when its target is not in the
resolved set **and** its entire content is Hero-managed. Any user content
outside the markers means the file is always preserved. `hero check`
surfaces an informational note when an orphaned instruction file is present.

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

See also: [Web UI Homes](../serve/homes.md) for the route inventory of
the top-level pages `hero serve` exposes in the browser.

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
