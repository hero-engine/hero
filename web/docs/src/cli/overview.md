# CLI Overview

The `hero` binary manages the spec corpus — the collection of design specs, bug reports, conventions, and decisions that drive your project. Agents call these commands automatically during workflows like `/design` and `/deliver`, but they're equally useful for manual inspection and scripting.

## Installation

```bash
# macOS / Linux
brew install hero-engine/tap/hero
```

See the [installation guide](../getting-started/installation.md) for Linux
install scripts and Windows (Scoop / PowerShell) options.

## Command Groups

| Group | Purpose | Commands |
|---|---|---|
| [Spec Management](spec-management.md) | Create, verify, complete, and visualize specs | `spec new`, `diagnose`, `spec complete`, `diff`, `drift`, `spec claim`, `size`, `supersede`, `graph`, `spec contract`, `spec plan` |
| [Search & Context](search-and-context.md) | Query the corpus and generate context for agents | `index`, `search`, `search --hybrid`, `ask`, `relevant`, `resume`, `status`, `dashboard`, `check`, `impact`, `recap`, `suggest`, `snapshot`, `synthesize`, `embeddings` |
| [Import](import.md) | Ingest external content (URL, file, directory) into the knowledge base | `import` |
| [Tracker Integration](tracker-integration.md) | Sync specs with GitHub Issues, Jira, Linear, and GitLab | `sync connect`, `sync import`, `sync spec`, `sync link`, `sync pull`, `sync push`, `sync jira`, `sprint load` |
| [Cross-Repo Peering](peering.md) | Talk to sibling Hero workspaces — advisory, spec-out, async handoff | `peer list`, `peer show`, `peer call`, `handoff`, `handoff status`, `handoff accept`, `context imports` |
| [Testing & Demos](testing-and-demos.md) | Generate acceptance tests and record demos from specs | `test generate`, `test run`, `test list`, `spec demo`, `coverage`, `smoke` |
| [Server & MCP](server-and-mcp.md) | Run the MCP server, HTTP daemon, and manage installations | `mcp`, `serve`, `install`, `upgrade`, `uninstall`, `agent run`, `agent jobs` |

## Not Sure Which Command?

```bash
hero do fix the login bug          # routes plain English to the right workflow
hero do check if my specs are healthy
```

`hero do "<request>"` is the "I don't know which command to use" front
door: describe what you want in plain English and Hero tells you which
command or slash-command workflow to run.

## Global Flags

```bash
hero --help          # Show all commands
hero --version       # Print version
hero --smoke         # Run a command smoke check when supported
```

## Workspace Utilities

A few smaller commands manage workspace-level configuration:

| Command | Purpose |
|---|---|
| `hero models` | Show or validate the model-role mapping (`design` / `execution` / `review`). Configure under `models.roles` in [hero.json](../configuration/hero-json.md). |
| `hero domain` | Show, `list`, or `switch` the active domain (e.g. `engineering`, `sales`). |
| `hero trust` | Apply one-time harness permission setup — `hero trust <claude\|codex> [project\|global]`. |
| `hero doctor` | Reconcile the running binary against the workspace graph, and confirm each installed harness target is complete (see [Troubleshooting](#troubleshooting)). |

## Troubleshooting

```bash
hero doctor          # which hero binary is running, and does its schema agree with the graph?
```

Run `hero doctor` when a tool reports a schema or version mismatch. The
usual cause is a **stale `hero` binary on `PATH`** — for example a
GUI-launched harness resolving a different binary than your login shell —
reading a current graph. `doctor` flags when the `PATH`-resolved `hero`
differs from the running one. Note that `hero upgrade` does **not** fix
this: it updates workspace files, not the binary.

`doctor` reports five sections:

```text
hero doctor

Running binary
  os.Executable(): /Users/you/go/bin/hero
  version:         v0.25.0
  binary schema:   4

PATH resolution
  `hero` on PATH:  /Users/you/go/bin/hero

Workspace graph
  workspace:       /Users/you/projects/app/.hero
  graph schema:    4

Installed harness targets
  TARGET   AGENTS   COMMANDS   SKILLS   ROOT FILE
  claude    35/35      29/29    55/55   CLAUDE.md
  codex     35/35          —    84/84   AGENTS.md
  not installed: copilot, cursor, generic, opencode

  codex has no command loader — its 29 commands install as skills under
  .agents/skills/command-<name>/ (55 canonical + 29 commands = 84).

Verdict: OK — binary and graph agree on schema 4.
```

- **Running binary / PATH resolution / Workspace graph** answer *which binary
  is running and does its schema agree with the graph.*
- **Installed harness targets** answers *did my install land?* Each row shows
  expected-vs-actual counts of agents, commands, and skills for one installed
  target. Absent targets collapse to a single `not installed:` line.
  Codex's commands cell shows `—`, not a number: Codex has no command loader,
  so Hero installs its commands as skills (which is why codex skills roll the
  two together). A `—` there is expected, not a broken install.
- If an installed target is **short on content**, its row is marked `!` and an
  in-section `WARNING:` recommends `hero upgrade` to re-materialize the missing
  files. The `Verdict:` line only ever answers the schema question, so an
  incomplete install never changes it.

## Quick Reference

```bash
# Check workspace health
hero status
hero check

# Search specs
hero search "authentication"
hero search --hybrid "retry logic for failed logins"
hero search --list --type bug

# Get context for files you're editing
hero relevant src/auth.go src/session.go

# Import issues from your tracker
hero sync import --type bug --dry-run
```
