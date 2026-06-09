---
title: "Add Workspace-Relative Path Fuzzy Resolution"
slug: hihcp-fuzzy-path-resolution
type: feature
status: planning
domain: engineering
size: small
priority: medium
created: 2026-06-09
tags: [hero-code, swift, path-resolution, dx, p2]
parent: hero-in-hero-code-parity
---

# Add Workspace-Relative Path Fuzzy Resolution

## Context

The model frequently passes bare filenames (`AgentLoop.swift`) to tools like
`Read` and `Edit`, but `resolvePath()` in `ToolExecutor.swift` expects absolute
or workspace-relative paths. Every `Read` call with a bare filename fails,
forcing the model to waste a turn discovering the full path via `find` or `rg`.

Parent initiative: `hero-in-hero-code-parity`.

## Goal

`Read(path: "AgentLoop.swift")` resolves correctly when the filename is unique
within the workspace. When multiple matches exist, the model receives a
disambiguation error listing all candidates with their full paths.

## Scope -- design inputs for `/design`

Enhance `resolvePath()` in `ToolExecutor.swift`:

1. If the path is already absolute or workspace-relative and resolves, use it
   (existing behavior, unchanged)
2. If the path looks like a bare filename (no `/` separators, or a short relative
   path), search the workspace tree for matches
3. If exactly one match is found, return that path
4. If multiple matches are found, return an error listing all candidates so the
   model can pick the right one
5. If no matches are found, return the existing "file not found" error

The workspace search should respect `.gitignore` and `.rgignore` to avoid
scanning build artifacts.

**Files to touch:**
- `Engine/ToolExecutor.swift` (`resolvePath()` method)

## Boundaries

- Do not change how absolute or workspace-relative paths resolve (existing
  behavior is correct for those cases)
- Do not add caching of the workspace file tree (keep it simple, search on demand)
- Do not apply fuzzy matching to directory paths (only bare filenames)

## Risks

- Performance: searching the full workspace tree on every bare filename could be
  slow. Mitigate by respecting ignore files and using a fast directory walk.
- False positives: a common filename like `index.ts` will have many matches.
  The disambiguation error handles this correctly.
- Behavior change: existing tool calls that fail on bare filenames will now
  succeed. This is the desired outcome but could surface unexpected behavior if
  the model was working around the limitation.

## Validation

- `Read(path: "AgentLoop.swift")` resolves to the correct file when unique
- `Read(path: "index.ts")` returns a disambiguation error with candidates
- `Read(path: "/absolute/path/to/file.swift")` continues to work unchanged
- `Read(path: "Engine/AgentLoop.swift")` (relative) continues to work unchanged
- Search performance is under 500ms for a typical hero-code workspace
