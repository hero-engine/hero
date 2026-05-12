---
title: Impact Analysis — What Breaks If I Touch This?
type: feature
status: completed
priority: P0
tags: [impact, context, codescan, mcp, agent-effectiveness]
created: 2026-04-22
relations:
  - target: hero-killer-features
    kind: parent
  - target: context-injection
    kind: related
  - target: spec-drift-detection
    kind: related
horizon: now
---

## Goal

Give agents and humans a single command — `hero impact <file-path>` — that
answers "which specs, conventions, decisions, and upstream dependents are
affected if I change this file?" Surface the answer as a structured report
via CLI and MCP tool, and auto-inject it into `hero_context` output so
agents get impact awareness before editing any file.

## Problem

Hero already knows a lot about the codebase: specs list their
`FilesTouched`, conventions have scope globs, decisions mention files, and
`internal/codescan/` builds a package-level dependency graph. But none of
that knowledge is assembled into a single "blast radius" view. Today, an
agent editing `internal/index/index.go` has no idea that three specs depend
on it, a convention governs its error handling, and two upstream packages
import it. The agent discovers this after breaking something, not before.

competitor surfaces "related specs" in its steering panel; Copilot Workspace
shows affected tests. Hero can do better by combining spec, convention,
decision, and dependency data into one pre-edit impact report — something
no other tool does.

## Design

### Data sources

Impact analysis draws from four existing data stores, all local:

| Source | Lookup | What it answers |
|---|---|---|
| `files_touched` table (SQLite) | Exact path match | Which specs list this file in `## Changes`? |
| `convention_scopes` table (SQLite) | Glob match via `FindConventionsForFiles` | Which conventions govern this file? |
| `decisions` table (FTS5) | File path as search term | Which decisions mention this file? |
| `DepGraph` (codescan `[]DepEdge`) | Reverse edge traversal from file's package | Which packages import this file's package? |

### `hero impact` command

```
hero impact <file-path>            # impact report for one file
hero impact <file-path> ...        # multiple files
hero impact --format json          # machine-readable output
hero impact --depth <n>            # limit dep graph traversal depth (default: 2)
hero impact --no-deps              # skip dependency graph, specs/conventions only
```

Default human output:

```
internal/index/index.go

  Specs (3):
    - context-injection (delivering) — lists internal/index/index.go in ## Changes
    - fts-search (complete) — lists internal/index/index.go in ## Changes
    - spec-drift-detection (delivering) — lists internal/index/index.go in ## Changes

  Conventions (1):
    - error-handling — scope: internal/**/*.go

  Decisions (1):
    - use-fts5-over-bleve — mentions internal/index/index.go

  Upstream dependents (depth 2):
    - internal/serve (direct) — imports internal/index
    - internal/cli (transitive via internal/serve) — imports internal/serve

  Summary: changing this file may affect 3 specs, 1 convention, 1 decision,
  and 2 upstream packages.
```

Exit code: 0 = report generated, 1 = file not found or not in project tree.

### Impact report struct

```go
type Report struct {
    FilePath     string        `json:"file_path"`
    Specs        []SpecRef     `json:"specs"`
    Conventions  []ConvRef     `json:"conventions"`
    Decisions    []DecisionRef `json:"decisions"`
    Dependents   []DepRef      `json:"dependents"`
}

type SpecRef struct {
    Slug   string `json:"slug"`
    Status string `json:"status"`
    Title  string `json:"title"`
}

type ConvRef struct {
    Slug  string `json:"slug"`
    Title string `json:"title"`
    Scope string `json:"scope"` // the matching glob
}

type DecisionRef struct {
    Slug  string `json:"slug"`
    Title string `json:"title"`
}

type DepRef struct {
    Package  string `json:"package"`
    Relation string `json:"relation"` // "direct" or "transitive"
    Via      string `json:"via,omitempty"` // intermediate package for transitive
    Depth    int    `json:"depth"`
}
```

### Dependency graph traversal

The existing `codescan.Result.DepGraph` is a flat `[]DepEdge` where each
edge is `{From, To}` (From imports To). To find upstream dependents of a
file, the algorithm:

1. Resolves the file path to its package (strip filename, match against
   `DepGraph` package paths).
2. Builds a reverse adjacency map: `To -> []From`.
3. BFS from the target package up to `--depth` levels.
4. Returns each discovered package with its depth and the intermediate path.

This reuses the existing `DepEdge` type — no new scanner work needed.

### MCP tool — `hero_impact`

```json
{
  "name": "hero_impact",
  "description": "Analyze the blast radius of changing a file: which specs, conventions, decisions, and upstream dependents are affected",
  "inputSchema": {
    "type": "object",
    "properties": {
      "file_paths": {
        "type": "array",
        "items": { "type": "string" },
        "description": "File paths to analyze (relative to project root)"
      },
      "depth": {
        "type": "integer",
        "description": "Max dependency graph traversal depth (default 2)"
      },
      "include_deps": {
        "type": "boolean",
        "description": "Include upstream dependency analysis (default true)"
      }
    },
    "required": ["file_paths"]
  }
}
```

Returns the same JSON as `hero impact --format json`.

### Integration with `hero_context`

The existing `hero_context` tool accepts file paths and returns conventions,
decisions, and risks. Impact analysis extends this: when `hero_context` is
called with file paths, the response gains an `## Impact` section showing
which specs reference those files and how many upstream dependents exist.
This is a lightweight summary (spec slugs + dependent count), not the full
report — agents call `hero_impact` for the complete picture.

The integration point is `internal/context/format.go`, which already groups
`ContextEntry` items by type. A new entry type `"impact"` is added,
populated by calling the impact analyzer from the `toolContext` handler in
`internal/serve/mcp.go`.

### What this does NOT do

Impact analysis is a static, pre-edit signal. It does not:

- Watch for file changes at runtime
- Block or gate edits based on impact
- Trigger notifications to other agents or users
- Require the MCP server to be running (CLI works standalone)

## Changes

- `internal/impact/impact.go` — `Analyze(filePaths, db, depGraph, depth)` function, `Report` struct, reverse dep BFS, JSON/text formatters
- `internal/impact/impact_test.go` — table-driven tests: single file, multiple files, no matches, transitive deps, depth limiting
- `internal/cli/impact.go` — `hero impact` command with `--format`, `--depth`, `--no-deps` flags
- `internal/cli/root.go` — register `impactCmd`
- `internal/serve/mcp.go` — register `hero_impact` tool, wire `toolImpact` handler
- `internal/context/format.go` — add `"impact"` entry type, render `## Impact` section in all format variants
- `internal/serve/mcp.go` — update `toolContext` to call impact analyzer when file paths are provided and append impact entries

## Acceptance Criteria

- WHEN `hero impact <file-path>` is called with a file listed in one or more specs' `## Changes` THE SYSTEM SHALL return those specs with their slugs, titles, and statuses
- WHEN `hero impact <file-path>` is called with a file matching a convention's scope glob THE SYSTEM SHALL return those conventions with their slugs, titles, and matching scope patterns
- WHEN `hero impact <file-path>` is called with a file mentioned in one or more decisions THE SYSTEM SHALL return those decisions with their slugs and titles
- WHEN `hero impact <file-path>` is called and the file's package has upstream dependents in the dep graph THE SYSTEM SHALL traverse the reverse dependency graph up to `--depth` levels (default 2) and return each dependent package with its relation type and depth
- WHEN `hero impact --no-deps <file-path>` is called THE SYSTEM SHALL skip dependency graph traversal and return only spec, convention, and decision matches
- WHEN `hero impact <file-path>` is called with a file that has no spec references, no matching conventions, no decision mentions, and no upstream dependents THE SYSTEM SHALL return an empty report and exit 0
- WHEN the `hero_impact` MCP tool is called with `file_paths` THE SYSTEM SHALL return the same structured JSON payload as `hero impact --format json`
- WHEN `hero_context` is called with file paths THE SYSTEM SHALL include a lightweight impact summary (affected spec slugs and upstream dependent count) in the response
- WHEN `hero impact` is called THE SYSTEM SHALL compute all results from local data (SQLite index and codescan dep graph) without making any network or LLM API calls

## Boundaries

- Does **not** replace `hero_context` — extends it with an impact summary; full reports come from `hero_impact`
- Does **not** add runtime or production awareness (no file watchers, no CI integration, no deployment tracking)
- Does **not** require a running MCP server — the CLI command works standalone against the local SQLite index and cached codescan data
- Does **not** block or gate edits — impact is advisory, not a gate
- Does **not** perform source-level analysis within files (e.g., which functions are affected) — operates at file and package granularity
