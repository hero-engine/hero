---
title: hero context Pipe — Explicit Tool-Aware Output Format for AI Tools
slug: hero-context-pipe
type: feature
status: completed
milestone: v0.3
tags: [context, pipe, output, format, tools, integration, mcp]
created: 2026-04-13
relations:
  - target: hero-serve-daemon
    kind: extends
  - target: skills-git-native
    kind: related
  - target: mcp-tool-filtering
    kind: related
horizon: now
---

## Goal

Give AI tools a reliable, structured way to receive Hero context via stdin/pipe — enabling `hero context` output to be composed into agent prompts, piped to tool CLIs, and formatted specifically for the target tool's context window constraints and format expectations.

## Problem

`hero context <files>` currently outputs a markdown document intended for human reading or pasting into a chat. When AI tools integrate Hero via the CLI (rather than MCP), they get this human-readable markdown — which is verbose, contains redundant headers, and isn't structured for token efficiency.

Different tools also have different expectations:
- **OpenCode** — expects context as a system prompt block or a specific `<context>` XML tag
- **Claude Code** — works best with a structured context document in its scratchpad format
- **Custom agent pipelines** — may need JSON, YAML, or a compact text format
- **Cursor** — rule injection via `.cursorrules` format

Without explicit format control, every tool that uses `hero context` via the CLI gets the same generic markdown and has to post-process it. This creates integration friction and reduces context quality.

## Design

### Output Format Flags

```
hero context <files>                    # default: human-readable markdown
hero context <files> --format json      # structured JSON
hero context <files> --format yaml      # YAML
hero context <files> --format compact   # compressed text, minimal whitespace
hero context <files> --format opencode  # OpenCode-specific system block format
hero context <files> --format claude    # Claude-specific XML context block
hero context <files> --format cursorrules  # .cursorrules injection format
hero context <files> --format pipe      # generic pipe format (newline-delimited JSON)
```

### Format Specifications

**`--format json`:**
```json
{
  "files": ["internal/api/user_handler.go"],
  "knowledge": [
    {
      "type": "convention",
      "slug": "conventions/api-handlers",
      "title": "API Handler Conventions",
      "body": "..."
    }
  ],
  "rules": [...],
  "decisions": [...],
  "generated_at": "2026-04-13T14:22:00Z"
}
```

**`--format compact`:**
Strips all markdown formatting, collapses whitespace, removes headers. Optimizes for token count. Useful when the agent needs to fit context into a tight window.

**`--format opencode`:**
Wraps context in OpenCode's expected format for programmatic injection via `opencode context inject`:
```
<hero_context>
[structured context block]
</hero_context>
```

**`--format claude`:**
Uses Claude's XML context tag convention:
```xml
<context>
<conventions>...</conventions>
<decisions>...</decisions>
<rules>...</rules>
</context>
```

**`--format cursorrules`:**
Outputs in `.cursorrules` file format — suitable for `cat >> .cursorrules` integration:
```
# Hero context — generated 2026-04-13
# Source: conventions/api-handlers, conventions/go-code-style

[rule block]
```

**`--format pipe`:**
Newline-delimited JSON — one entry per knowledge item, suitable for stream processing:
```
{"type":"convention","slug":"conventions/api-handlers","body":"..."}
{"type":"rule","slug":"rules/ci-github-actions","body":"..."}
```

### Token Budget Control

```
hero context <files> --max-tokens 2000    # truncate to ~2000 tokens
hero context <files> --max-tokens 4000 --priority conventions,decisions,rules
```

`--max-tokens` triggers a prioritized truncation strategy:
1. Keep all `rule` entries (mandatory)
2. Keep `convention` entries, longest first (most specific)
3. Keep `decision` entries by recency
4. Truncate remaining to fit budget

Priority order is configurable via `--priority`.

### Pipe Integration

`hero context` is designed for composition:

```sh
# Pipe context into a custom agent CLI
hero context internal/api/ --format pipe | my-agent run --context-stdin

# Inject context into opencode session
hero context internal/api/ --format opencode | opencode context inject

# Build a .cursorrules file from current context
hero context . --format cursorrules >> .cursorrules

# Use in a skill step
hero context {{files}} --format json --max-tokens 3000 > /tmp/hero-context.json
```

### Tool Config in `hero.json`

Default output format per target tool:

```json
{
  "context": {
    "default_format": "markdown",
    "tools": {
      "opencode": { "format": "opencode", "max_tokens": 4000 },
      "cursor": { "format": "cursorrules", "max_tokens": 2000 },
      "claude": { "format": "claude", "max_tokens": 8000 }
    }
  }
}
```

When `HERO_TOOL` env is set (by shell integration), Hero auto-selects the right format:
```sh
export HERO_TOOL=opencode
hero context internal/api/  # automatically uses opencode format + max_tokens 4000
```

### `--diff` Mode

For incremental context updates (agent already has context, only changed files need delta):

```
hero context <files> --diff --since <commit-sha>
```

Returns only knowledge entries that changed since the given commit, in the specified format.

## Changes

- `internal/context/format.go` — format registry, renderer implementations (json, yaml, compact, opencode, claude, cursorrules, pipe)
- `internal/context/truncate.go` — token budget calculation, priority-based truncation
- `internal/cli/context.go` — `--format`, `--max-tokens`, `--priority`, `--diff` flags
- `internal/config/config.go` — `context.tools` config section

## Acceptance Criteria

- `--format json` outputs valid JSON matching the schema above
- `--format compact` reduces output size by at least 30% vs default markdown (measured by byte count)
- `--format opencode`, `--format claude`, `--format cursorrules` produce tool-correct output
- `--format pipe` outputs valid newline-delimited JSON, one entry per line
- `--max-tokens N` truncates output to approximately N tokens using priority-based strategy
- `HERO_TOOL` env variable auto-selects format from `hero.json` tool config
- `hero context <files> --format pipe | wc -l` equals the number of knowledge entries
- `--diff --since <sha>` returns only changed entries since that commit
- All formats produce valid output for empty knowledge base (no panic, structured empty response)

## Boundaries

- Does **not** send context to tools directly — output goes to stdout for composition
- Does **not** implement tool-specific APIs (OpenCode API, Cursor API) — format only
- Token counting is approximate (whitespace-split word count × 1.3) — not exact BPE tokenization
- `--format cursorrules` does not modify `.cursorrules` directly — output goes to stdout for the user to redirect
