---
title: /prime Command — Explicit Session-Start Context Loading
slug: prime-command
type: feature
status: completed
milestone: v0.2
tags: [commands, context, session, onboarding, ux]
created: 2026-04-12
relations:
  - target: hero-serve-daemon
    kind: related
horizon: now
completed_at: 2026-05-18T19:25:38Z
---

## Goal

Engineers start AI tool sessions cold — the model has no memory of the last session, no idea what's in progress, and no knowledge of project conventions or recent decisions. Today they either manually invoke `hero context`, hope CLAUDE.md/AGENTS.md is complete enough, or just start working and get bad output because the model is flying blind.

The `/prime` command is a one-shot session-start ritual: run it at the start of a working session and get the model fully loaded with the right context — what's in progress, relevant conventions, active decisions, and recent changes — before writing a single line of code.

This is one of the most commonly missing commands in community Claude/Cursor setups (see `memory-tools-and-community-patterns` note). It's simple but it closes a real gap.

## Context

The community pattern research (awesome-claude-code, 38k stars) showed `/prime` or `/context` as the most commonly requested missing session-start command. The gap is real: `hero context imports --files` injects context for specific files, but there's no "load me up for this session" command. `/prime` is that command.

`hero serve` (MCP daemon) eventually makes `/prime` less critical by providing continuous context injection — but the MCP server is a larger effort, and `/prime` is simple and immediately valuable.

## Design

### Command Surface

```
/prime
/prime sprint-42
/prime spec: csv-export
/prime area: auth
```

With no arguments, `/prime` figures out the right context automatically based on:
- Specs in `delivering` status (what's in progress)
- Recent git activity (what files have changed in the last few days)
- Active spec claims (`claimed_by: current-user`)

With arguments, it loads context for a specific spec, sprint, or code area.

### What /prime Does

```
1. Identify current work context
   - Specs in "delivering" status
   - Specs claimed by the current user (git author from git config)
   - Recent git commits (last 5) to identify active files

2. Load conventions relevant to the active areas
   - Conventions scoped to files being worked on
   - Conventions tagged to the active spec's tags

3. Load active decisions
   - Decisions that haven't expired
   - Decisions relevant to the active spec's files/tags

4. Load spec context
   - Full spec for the active delivery (if one is found)
   - Related specs (blocked-by, depends-on)
   - Recent completions that may affect the area (last 30 days)

5. Summarize and present
   - Current sprint state (if a sprint note exists)
   - What the model should know before starting
   - Specific things to watch for (active decisions, known risks from spec)
```

### Example Output

```markdown
# Session Context — 2026-04-14

## What's in Progress

**csv-export** (delivering) — claimed by alice
Add CSV export to user data API (ENG-142)
Files: internal/api/export.go, internal/api/export_test.go

Related: user-data-api (completed), fix-export-timeout (blocked-by)

## Active Decisions Relevant to This Work

**ADR-007** — Use JWT, not sessions (all API endpoints)
**ADR-012** — All exports must support streaming for large datasets

## Conventions for This Area

- API handlers live in `internal/api/` and follow the middleware chain pattern
- All endpoints require authentication middleware
- Export endpoints must set `Content-Disposition: attachment` header
- Error responses use the shared `api.ErrorResponse` struct

## Recent Context

- **2 days ago**: fix-auth-middleware completed (alice) — auth middleware refactored, JWT validation moved to middleware layer
- **Sprint 42** (active): 8 items, 2 delivering, 3 completed

## Watch For

- The spec notes a known risk: export hangs on datasets > 50k rows (not yet resolved)
- fix-export-timeout is blocking this spec — check if it's resolved before proceeding
```

### Implementation

`/prime` is an agent command in `commands/prime.md`. It calls `hero context` and `hero status` under the hood to assemble the session brief. The agent does the synthesis — no new CLI commands needed.

```markdown
# /prime

Load session context for the current work.

## Steps

1. Run `hero status --delivering` to find in-progress specs
2. Run `hero status --claimed` to find specs assigned to the current user
3. Run `hero context imports --files <recent-git-files>` for conventions and decisions
4. If a specific spec or sprint is mentioned, run `hero context --spec <slug>`
5. Synthesize into a session brief covering: what's in progress, active decisions,
   relevant conventions, recent context, and things to watch for
6. Present the brief clearly. Do not begin any implementation — this is context only.

## When called with an argument
- `/prime <spec-slug>`: load context for that specific spec
- `/prime sprint-<name>`: load the sprint note + all delivering specs in the sprint
- `/prime area: <topic>`: run `hero search <topic>` + context for matching specs
```

### Integration with `hero serve`

When `hero serve` is running, `/prime` can query the MCP server instead of shelling out to the CLI — the context is always fresh (the daemon's watcher keeps the index current):

```
/prime → hero_context(files: <recent-files>) → hero_list(status: delivering) → synthesize
```

This is the preferred path when `hero serve` is running. The command works either way — CLI fallback if the MCP server is not running.

### Auto-Prime (Optional)

For teams that always want session priming, `hero.json` can enable auto-prime:

```json
{
  "prime": {
    "auto": false,
    "on_first_message": false
  }
}
```

`auto: true` + `on_first_message: true` would trigger `/prime` automatically on the first message of each session. This requires hook support from the AI tool (Claude Code `SessionStart` hook, etc.) — available only with hook-compatible tools.

## Changes

- `commands/prime.md` — `/prime` command definition
- (No CLI changes needed — `/prime` is an agent command that uses existing CLI tools)

## Acceptance Criteria

- `/prime` with no arguments identifies in-progress work and produces a session brief
- Brief includes: active specs, relevant conventions, active decisions, recent context, and risks
- `/prime <slug>` loads context for a specific spec
- `/prime sprint-<name>` loads sprint context
- Output is concise enough to fit comfortably in a session opening (~500-1000 tokens)
- The command does not start any implementation — it is context-only
- Works without `hero serve` running (CLI fallback)
- Works better with `hero serve` running (MCP query path)

## Boundaries

- Does **not** replace `hero context` — `/prime` is a synthesis command; `hero context` is the raw context tool
- Does **not** write anything — output only, no side effects
- Does **not** require manual file selection — it infers context from git + spec state automatically
- Auto-prime (`on_first_message`) is opt-in and hook-dependent — not available in all AI tools
