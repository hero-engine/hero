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
| [Spec Management](spec-management.md) | Create, verify, complete, and visualize specs | `spec new`, `spec complete`, `diff`, `drift`, `spec claim`, `graph`, `spec contract`, `spec plan` |
| [Search & Context](search-and-context.md) | Query the corpus and generate context for agents | `index`, `search`, `ask`, `relevant`, `resume`, `status`, `dashboard`, `check`, `impact`, `recap`, `suggest` |
| [Tracker Integration](tracker-integration.md) | Sync specs with GitHub Issues, Jira, and Linear | `sync connect`, `sync import`, `sync spec`, `sync link`, `sync pull`, `sprint load` |
| [Testing & Demos](testing-and-demos.md) | Generate acceptance tests and record demos from specs | `test generate`, `test run`, `test list`, `spec demo`, `coverage`, `smoke` |
| [Server & MCP](server-and-mcp.md) | Run the MCP server, HTTP daemon, and manage installations | `mcp`, `serve`, `install`, `upgrade`, `uninstall`, `agent run`, `agent jobs` |

## Global Flags

```bash
hero --help          # Show all commands
hero --version       # Print version
hero --smoke         # Run a command smoke check when supported
```

## Quick Reference

```bash
# Check workspace health
hero status
hero check

# Search specs
hero search "authentication"
hero search --list --type bug

# Get context for files you're editing
hero relevant src/auth.go src/session.go

# Import issues from your tracker
hero sync import --type bug --dry-run
```
