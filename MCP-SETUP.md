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
hero install project . --target deepseek
```

The installer writes the harness-native instruction/workflow surfaces and its
supported MCP configuration. If a session starts inside a monorepo subfolder,
run `hero install satellites` at the repository root; satellites are thin
harness trees pointing to the one root `.hero` corpus.

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

- **Server name:** `hero-<project>-<hash>`, e.g. `hero-api-3f2a`. Hero tools
  appear as `mcp__hero-api-3f2a__hero_status` and so on.
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
