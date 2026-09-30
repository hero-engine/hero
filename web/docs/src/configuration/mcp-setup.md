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

The install target is `deepseek`; the harness executable is `dsh`. This is
harness integration, not model-provider configuration. Compatibility is based on
DeepSeek Harness commit `477b4f420553e8a52c2fbccc464d7561b239c443`.

```bash
hero install project . --target deepseek
dsh --profile headless --patch '/absolute/project/.dsh/hero.cordis.patch.yml' 'Resume this Hero workspace'
```

Run `dsh` from the intended workspace. The interactive profile can use the same
`--patch` argument. Installation reports **overlay generated; activation
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
