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

The install target is `deepseek`; the harness executable is `dsh`. This is
harness integration, not model-provider configuration. Compatibility is based on
DeepSeek Harness commit `477b4f420553e8a52c2fbccc464d7561b239c443`.

```bash
hero install project . --target deepseek
dsh --profile web --patch '/absolute/project/.dsh/hero.cordis.patch.yml'
```

Run `dsh` from the intended workspace; `--profile web` starts an interactive
session. `--profile headless` with a prompt argument is a one-shot model run for scripts. Installation reports **overlay generated; activation
required**. Neither a file on disk nor `hero doctor` proves a live connection.
The patch inserts `@deepseek-ai/dsh-mcp-client` with `serverName: hero`,
`transport: stdio`, `command: hero`, and `args: [mcp]`. Hero never edits your
shared `cordis.patch.yml`, profiles, models, or allowlists.

Project instructions live in the managed region of `AGENTS.md`; canonical,
`command-*`, and `role-*` skills live under `.dsh/skills/`. Roles are guidance,
not registered native agents. A profile without independent delegation cannot
complete a required fresh review or cold audit by adopting the reviewer role
locally. DeepSeek also reads `CLAUDE.md` and `.agents/skills`; mixed installs can
expose duplicates. At the pinned baseline, project skills beat global skills,
`.dsh/skills` beats `.agents/skills`, and provider registration/local ordering break remaining ties. Hero preserves other harnesses' files.

For global installation:

```bash
hero install global --target deepseek
```

Global files use `$DSH_HOME/AGENTS.md`, `$DSH_HOME/skills/`, and
`$DSH_HOME/hero.cordis.patch.yml`. Nonblank `DSH_HOME` values retain their whitespace; unset/whitespace-only means
`~/.dsh`, `~/` expands to your home, and relative values resolve against the
installation working directory. It does not redirect project installs. Global
installation neither creates a project nor pins the current repository. Pass
the absolute global overlay path with `--patch` each time you launch.

The MCP child binds once to its launch workspace. One global overlay is reusable
configuration, not a router for multiple repositories in a shared GUI/server
process. Launch one process per intended workspace. `hero install project .
--target deepseek --workspace services/api` writes a subfolder overlay explicitly
bound to the parent project root using `--project-root`.

Satellites link only `.dsh/skills` and retain the shared instruction marker.
Within a Git tree, DeepSeek finds the first `.git` ancestor and loads its root
skills; nested links are useful for separately rooted/non-Git satellites. Launch
from the satellite directory with the absolute **parent** overlay path, keeping
that working directory. An overlay created with `--workspace` keeps its explicit
project-root binding.

Generated patches keep the portable executable name `hero`. The harness's
`PATH` chooses the binary; GUI launches need not inherit shell startup files.
Use `hero doctor` to inspect binary drift. For a specific development/release
binary, keep a separate user-owned overlay with an absolute `command` path and
activate that copy. Restart the process after changing the binary/configuration.
The installer does not install executables or change `PATH`.

Project removal uses `hero uninstall --target deepseek`; other targets and
foreign files remain. Modified DeepSeek skills/patches are preserved. Unknown
existing overlays block install before mutation; use `--force` only when you
intend replacement. `--dry-run` previews without writing.

Global install/reinstall records relative paths and SHA-256 checksums in
`$DSH_HOME/hero-install-manifest.json`. There is no global-uninstall command.
For manual global cleanup, remove only files whose current SHA-256 matches their
manifest entry, preserve modified/unlisted files, and remove only the
`hero:managed-start` through `hero:managed-end` region from `AGENTS.md`. Never
remove the whole home or shared Cordis configuration. Remove the manifest after
finishing that inventory review; reinstall uses it to protect user changes.

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
