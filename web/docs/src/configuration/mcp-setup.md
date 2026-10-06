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
hero install project . --target grok
hero install project . --target deepseek
```

For sub-folder workspaces:

```bash
hero install satellites
```

Run the satellite command from the repository root. It creates thin
harness-native trees that point to the one root `.hero` corpus.

The installer preserves user-owned files where possible and writes only
Hero-managed MCP blocks/config.

## DeepSeek Harness (`dsh`)

The install target is `deepseek`. It works with the DeepSeek Harness desktop
app and the `dsh` CLI. This is harness integration, not model-provider
configuration. Compatibility is based on DeepSeek Harness commit
`477b4f420553e8a52c2fbccc464d7561b239c443`.

```bash
hero install project . --target deepseek
```

That's the whole setup: restart the DeepSeek desktop app, or start
`dsh --profile web`. No `--patch` flag is needed.

**How MCP is registered.** Every DeepSeek profile, including the desktop
app's, loads the home patch `$DSH_HOME/cordis.patch.yml` (default
`~/.dsh/cordis.patch.yml`). A project install adds one Hero entry per project
to that file, between `# hero:managed deepseek-mcp <server>` markers:

- **Server name:** `hero-<project>-<hash>`, e.g. `hero-api-3f2a9c`. Hero tools
  appear as `mcp__hero-api-3f2a9c__hero_status` and so on.
- **Command:** the absolute `hero` path found on your `PATH` at install time.
  GUI apps don't inherit your shell `PATH`. If you reinstall Hero somewhere
  else, rerun the install. To pin a specific build, set
  `HERO_DEEPSEEK_MCP_COMMAND=/abs/path/to/hero` while installing.
- **Project binding:** `cwd` and `--project-root` are pinned to the project,
  so Hero serves the right project wherever DeepSeek was launched from.

Hero never touches anything outside its own markers. It refuses (before
writing any file) to edit a home patch that isn't a YAML list, isn't a
regular file, or is a symlink. It never edits the app-managed
`~/.dsh/profiles/*/cordis.patch.yml`, models, or allowlists.

DeepSeek starts MCP servers once per profile, not per session, so every
installed project's Hero server is loaded in every session. `AGENTS.md` tells
the model to use the server whose `hero_status` reports this repository.
`hero doctor` shows the project's server name and whether its entry is present,
points at an executable `hero`, and serves this project root. If not, it gives
the verdict `NEEDS REPAIR` with the install command that fixes it.

Project instructions live in the managed region of `AGENTS.md`; canonical,
`command-*`, and `role-*` skills live under `.dsh/skills/`:

- **Roles** are guidance, not registered native agents. A profile without
  independent delegation cannot complete a required fresh review or cold audit
  by adopting the reviewer role locally.
- **Discovery:** DeepSeek also reads `CLAUDE.md` and `.agents/skills`, so mixed
  installs can expose duplicates. At the pinned baseline, project skills beat
  global skills, `.dsh/skills` beats `.agents/skills`, and provider
  registration/local ordering break remaining ties. Hero preserves other
  harnesses' files.

`hero install project . --target deepseek --workspace services/api` registers
the same project-root entry, so there is still one server per project.
Satellites link only `.dsh/skills` and use the parent project's server. Within
a Git tree, DeepSeek finds the first `.git` ancestor and loads its root skills;
nested links are useful for separately rooted or non-Git satellites.

Earlier Hero versions generated `.dsh/hero.cordis.patch.yml` for use with
`--patch`. Reinstalling removes it if unmodified. A modified copy is kept with
a warning, because passing it with `--patch` now adds a second Hero server.

Project removal uses `hero uninstall --target deepseek`. It removes this
project's home-patch entry, and deletes the file only if Hero created it and
nothing else remains. Other targets, foreign entries, and modified skills are
preserved. `--dry-run` previews without writing.

For global installation:

```bash
hero install global --target deepseek
```

Global files use `$DSH_HOME/AGENTS.md`, `$DSH_HOME/skills/`, and a
`$DSH_HOME/hero.cordis.patch.yml` overlay:

- **Activation:** the global overlay is not pinned to a project and is not
  loaded automatically. Pass it with `dsh --profile web --patch
  '<absolute path>'` from the intended workspace. Prefer project installs for
  the desktop app.
- **`DSH_HOME`:** nonblank values retain their whitespace. Unset or
  whitespace-only means `~/.dsh`, `~/` expands to your home, and relative
  values resolve against the installation working directory.
- **Cleanup:** global install/reinstall records relative paths and SHA-256
  checksums in `$DSH_HOME/hero-install-manifest.json`. There is no
  global-uninstall command. For manual cleanup:
  - Remove only files whose current SHA-256 matches their manifest entry, and
    preserve modified or unlisted files.
  - From `AGENTS.md`, remove only the `hero:managed-start` through
    `hero:managed-end` region.
  - Never remove the whole home or shared Cordis configuration.
  - Remove the manifest after that review; reinstall uses it to protect your
    changes.

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

The authoritative tool inventory is the runtime `tools/list` response after
configured filtering. Common tools include:

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

Capability groups are documented in [Server and MCP](../cli/server-and-mcp.md).
Use `tools/list` when an exact revision-tied inventory is required.

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
