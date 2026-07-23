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

## Git Hooks

```bash
hero hooks status      # show which Hero git hooks are installed
hero hooks install     # install Hero git hooks into .git/hooks/
hero hooks uninstall   # remove all Hero git hooks and the merge driver
```

Hero's git hooks keep projected handoff files (`.hero/NEXT.md`,
`.hero/next/*.md`, `.hero/SNAPSHOT.md`, `.hero/QUEUE.md`) staged
automatically so they travel with every commit — without which the next
session (possibly on another machine) starts cold. Standard install paths
wire these for you; `hero check` warns if the staging block is missing.

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

### Durable attention contract

Hero Code's v1 desktop boundary is the user-global HTTP API rooted at
`/api/attention/v1`. It is available before a project is selected and exposes
authoritative snapshots plus advertised row actions. Mail and Focus retain
separate mutation contracts; there is no generic mutable Attention endpoint.

Clients refresh the authoritative snapshot on mount, foreground, reconnect,
and after every successful mutation. Streaming events are optional: v1 clients
must remain correct using snapshot refresh alone. The portable schemas and
checksum manifest live under `contracts/attention/{schema,testdata}/v1`.

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

## Hero Cloud

!!! note "Requires a Hero Cloud account"
    These commands talk to Hero Cloud (or a self-hosted instance via
    `HERO_CLOUD_URL`). They're independent of your issue-tracker setup.

```bash
hero login                 # authenticate via GitHub OAuth (stores ~/.hero/credentials.json)
hero cloud create-org      # create a Hero Cloud organization
hero sync cloud            # push specs to Hero Cloud
hero sync graph            # push / pull knowledge-graph deltas
hero logout                # revoke credentials and the server refresh token
```

`hero login` opens a browser to authenticate and stores a token locally;
`hero cloud` bootstraps an org and links a repo without opening the
dashboard. Once authenticated, cloud-aware commands — `hero sync cloud`,
`hero sync graph`, and cross-repo queries like `hero impact --cross-repo` —
become available.

## Async Agent Runtime

`hero agent` runs agent work *outside* the interactive session —
fire-and-forget jobs, scheduled automations, and approval gates for work
that needs human sign-off before it merges.

```bash
hero agent run deliver auth-flow       # run headlessly via the Claude/OpenAI API
hero agent jobs                        # list / inspect / cancel async jobs
hero agent automate                    # set up event-driven automations
hero agent approve <job-id>            # approve a gated job to continue
hero agent events                      # log / inspect cross-session events
```

`hero agent run` executes a workflow headlessly against a model provider
API (set the corresponding key, e.g. `ANTHROPIC_API_KEY`). `hero agent
jobs` inspects the queue; `hero agent automate` wires event-driven
triggers; and gated jobs pause for `hero agent approve` before completing.

## Publishing

`hero publish` is one-way output of Hero state to external surfaces —
the read-only counterpart to the bidirectional `hero sync`:

```bash
hero publish wiki      # push completed specs to Confluence / GitHub Wiki
hero publish pages     # render the knowledge graph as a static GitHub Pages site
```

Configure the Confluence target as a `docs`-role connection — see
[Tracker Setup → Confluence](../configuration/tracker-setup.md#confluence-wiki-publishing).

## Workspace Watcher

```bash
hero watch                   # local mode: poll for changes, auto-reindex (Ctrl+C to stop)
hero watch --interval 5      # local mode, 5s poll interval
hero watch --mode ci         # one-shot: validate all specs + health checks, non-zero on issues
```

`hero watch` keeps the index fresh as you work (local mode) or runs a
single validating pass for CI (ci mode).

## CI Status

```bash
hero ci                    # pipeline status for the current branch
hero ci --branch main      # a specific branch
hero ci --format json      # machine-readable
```

`hero ci` queries the configured CI provider for the latest run on the
branch — pass/fail, failed-step detail, and a link. Requires
`environment.ci` in `hero.json`, e.g.
`{ "environment": { "ci": { "provider": "github-actions" } } }`.

## Skills

Skills are reusable step-by-step workflows stored as
`.hero/skills/<name>.md`:

```bash
hero skill list            # list available skills
hero skill run <name>      # run a skill
hero skill save <name>     # scaffold a new skill and open it in $EDITOR
hero skill show <name>     # print the skill markdown
```

## Export

```bash
hero export knowledge <dir>   # export the knowledge base to a directory
hero export mocks <dir>       # export mock artifacts
```

## Uninstall

```bash
hero uninstall --target claude
hero uninstall --target codex
hero uninstall --target claude --dry-run    # preview what would be removed
```

`hero uninstall` removes only the agent, command, and skill files Hero
originally installed for a target (tracked in `.hero/version.json`);
user-created files in those directories are preserved. For Claude Code it
also strips the Hero-managed section from `CLAUDE.md`, leaving your own
content intact. Supported targets: `opencode`, `cursor`, `claude`, `codex`.
