---
title: "Satellite Scope Finishers — Queue Default and MCP Subproject Param"
slug: satellite-scope-finishers
type: feature
status: planning
priority: medium
horizon: now
tags: [monorepo, satellites, scoping, queue, mcp]
relations:
  - target: monorepo-satellite-installs
    kind: parent
  - target: satellite-corpus-integration
    kind: parent
  - target: satellite-scope-extras
    kind: parent
created: 2026-05-12
---

# Satellite Scope Finishers — Queue Default and MCP Subproject Param

## Problem

After three rounds of satellite work the corpus knows about scope, the dashboard filters by scope, and `hero list / search / recap / feed / why` all respect scope when run in a scoped cwd. Two surfaces still leak:

1. **`hero queue` is scope-blind.** It's the everyday "what should I work on" front door — the most-used scope-relevant surface — and a developer running `hero queue` from inside `engines/mlx` today sees the entire workspace's ready set instead of just the mlx-ready set. The internals support it (`spec.Filter.Subproject` works, the selector applies it); the CLI just doesn't pass it through.

2. **MCP tools don't accept a `subproject` argument.** `hero_list / hero_queue / hero_search / hero_recap / hero_feed` are how the *model* talks to the corpus. The slash-command preamble (`hero context scope`) tells the model "you're in scope X — stamp it" — but when the model wants to *query* the corpus from inside that satellite chat, the MCP tool schemas don't expose `subproject` as a parameter, so the model can't push the scope hint into its queries. It either gets the whole repo back, or has to filter results client-side, which is wasteful and noisy.

The MCP server runs at the workspace root, not in the user's cwd. So the right shape isn't "default from cwd" (that has no meaning here) — it's "expose subproject as an explicit tool parameter and tell the model to use it when active scope is known."

## Goal

Two small surface-level additions that close the daily-use gaps without touching anything below the CLI/serve layer:

- `hero queue` reads `--subproject <name|all>` with the same default-from-cwd rule that `hero list` already uses.
- The five spec-querying MCP tools (`hero_list`, `hero_queue`, `hero_search`, `hero_recap`, `hero_feed`) accept a `subproject` parameter, document it clearly, and apply it. Tool descriptions tell the model *when* to pass it ("set this when the user is working in a satellite or you've been told the active scope").

**Mission-fit.** Queue is the high-frequency surface a scoped developer hits every working day. MCP is the model's only path into the corpus — without it, every scoped session has to either accept noisier results or do client-side filtering. Both are about making the in-harness experience consistent with what the slash-command preamble has already promised.

The non-goal is broader integration: `hero blocked`, dashboard pipeline cards, per-scope dashboard views — all surveyed and skipped intentionally as either low-frequency (blocked) or wrong-default (pipeline cards).

## Design

### 1. `hero queue` cwd default

`runQueue` currently builds `spec.Selector{Filter: spec.Filter{Horizons, Ready: true, ExcludeClosedDefault: true}}`. The `Subproject` field on Filter already exists and the selector applies it. The change is:

1. Add `--subproject <name|all>` flag (string, default empty) parallel to `hero list`.
2. Reuse the existing `resolveSubprojectFilter` helper from `internal/cli/list.go` — it implements the explicit-flag-wins, else-cwd-default rule.
3. Reuse `maybePrintScopeHint` — same one-time-per-machine "showing scope: X" nudge as `hero list`.
4. Pass the resolved value into `Filter.Subproject`.

That's it. No new config, no new persistence, no test scaffolding beyond what list/search already have.

### 2. MCP tool subproject parameter

Five tools need the parameter:

| Tool | Path |
|---|---|
| `hero_list` | `selectorFromMCPArgs` (already builds a `spec.Selector` — needs one new line) |
| `hero_queue` | uses its own selector hand-rolled in `toolQueue` (not via `selectorFromMCPArgs`) — needs one line |
| `hero_search` | calls into retrieval which already accepts `subproject` filter — needs `args["subproject"]` plumbing |
| `hero_recap` | filters results post-build — needs `args["subproject"]` plumbing |
| `hero_feed` | already has `feed.Filter` — needs `args["subproject"]` plumbing |

For each, the tool *definition* (`mcp_tools_def.go`) gets an additional `subproject` property in `InputSchema`. Description on each follows a consistent template:

> Filter to a specific subproject scope (e.g. "engines/mlx"). When the user is working in a satellite or you have been told the active subproject, pass it here so results are scope-relevant. Pass "all" to disable when the model is asking a workspace-wide question.

That last clause matters: without "pass `all` to disable", a model that has memorized the active scope from the slash-command preamble will keep filtering by it even when the user later asks a wide question. The escape hatch needs to be visible in the description.

The `selectorFromMCPArgs` change is one line:

```go
Subproject: stringOr(args["subproject"], ""),
```

`hero_search` already routes through retrieval's `Filters` map, so the change is `q.Filters["subproject"] = sp` when args has it.

`hero_recap` and `hero_feed` already filter post-build / via `feed.Filter` — the CLI patterns from the previous spec port directly.

**Why no MCP-side cwd default?** Because the MCP server has no concept of the harness session's cwd. The MCP server's cwd is wherever the harness spawned it from — typically the workspace root. The hint about scope flows model-side via the slash-command preamble, not through MCP. So the parameter is opt-in per call, with the description telling the model when to opt in.

**Why not derive scope from a session-level "active scope" set by the harness?** Two reasons. First, MCP doesn't have a session-state surface that's portable across harnesses. Second, the model already knows the active scope from the preamble — making it pass that hint in is a much simpler, more transparent design than encoding it in some session blackbox.

### Design decisions

**Why ship both pieces in one spec instead of two?** They're symmetric and tiny. Splitting would double the ceremony for the same change. Each is essentially "thread one parameter through one already-built mechanism."

**Why does `hero queue` get cwd-default but the MCP tools don't?** Different runtime model. CLI commands run *in* the user's cwd, so cwd-default is meaningful — and matches what `hero list` already does, so users don't have to remember per-command rules. MCP tools run in the server's cwd, which has no relationship to the user's harness session — the model is the one with the scope context and pushes it explicitly.

**Why is the parameter description consistent across all five MCP tools instead of tailored per tool?** Because the model reads them all and consistency makes the pattern learnable. A model that sees "subproject" in `hero_list` and parses out the right behavior should not have to re-parse a different phrasing in `hero_queue`.

**Why include the "all" escape hatch prominently in the description?** Because models that pick up a scope from the preamble tend to over-apply it. Telling them "here's how to opt out for workspace-wide questions" prevents the failure mode where a scoped session can never see anything outside its scope.

**Why not also document the parameter on the slash-command preamble side?** The preamble already says "stamp `subproject:` on artifacts." Adding "and pass `subproject` to MCP tools" would balloon it. The MCP tool description is the right place — it's the contract for the tool, not for the preamble.

## Acceptance Criteria

- THE SYSTEM SHALL accept a `--subproject <name|all>` flag on `hero queue` matching the semantics of the same flag on `hero list`.
- WHEN `hero queue` is invoked from a satellite or scoped cwd without `--subproject` THE SYSTEM SHALL default the filter to the active scope from `.hero/subprojects.json`.
- WHEN `hero queue` applies a default-scope filter THE SYSTEM SHALL print the same one-time per-machine hint surfaced by `hero list` ("showing scope: X — pass `--subproject all` for the full workspace").
- THE SYSTEM SHALL accept a `subproject` parameter on `hero_list`, `hero_queue`, `hero_search`, `hero_recap`, and `hero_feed` MCP tools.
- WHEN any of those tools is invoked with `subproject` set to a non-empty value other than "all" THE SYSTEM SHALL filter results to that scope.
- WHEN any of those tools is invoked with `subproject="all"` or empty THE SYSTEM SHALL apply no subproject filter.
- THE SYSTEM SHALL document the `subproject` parameter on each of those tools with a description telling the model to set it when an active scope is known and to pass "all" to opt out for workspace-wide queries.

## Changes

### Modified files
- `internal/cli/queue.go` — `--subproject` flag + cwd default + scope hint
- `internal/serve/mcp_tools_def.go` — add `subproject` property to the five tools' input schemas
- `internal/serve/mcp_tools.go` — wire `args["subproject"]` through `selectorFromMCPArgs`, `toolQueue`, `toolSearch`, `toolRecap`, `toolFeed`

## Phasing

Single phase — all changes ship together.

## Kickoff

Resume by reading the spec at `.hero/planning/features/satellite-scope-finishers/spec.md`. Two surface additions: `hero queue --subproject` flag (mirrors `hero list`), and `subproject` parameter on five MCP tools (`hero_list`, `hero_queue`, `hero_search`, `hero_recap`, `hero_feed`) with consistent description telling the model when to pass it. Parent specs in the satellite arc: monorepo-satellite-installs, satellite-corpus-integration, satellite-scope-extras — all shipped.
