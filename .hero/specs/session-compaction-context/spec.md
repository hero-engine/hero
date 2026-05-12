---
title: Session Compaction Context — Active Spec Survival After Context Loss
type: feature
status: completed
priority: P0
tags: [context, compaction, active-spec, session, mcp]
created: 2026-04-22
relations:
  - target: hero-killer-features
    kind: parent
  - target: agent-cold-start
    kind: related
horizon: now
---

## Goal

When an AI agent session compacts or a new session starts, ensure the agent
immediately knows which spec it's working on and what conventions apply —
without re-reading the chat or asking the user.

## Problem

When OpenCode/Claude Code compacts a long session, the LLM loses nuanced
context about what spec is active, what decisions were made, and what
conventions apply. Hero already stores all this structured knowledge but
doesn't automatically re-inject it post-compaction. The NEXT.md handoff
helps but requires the agent to manually read it and is lossy about which
spec is actively being worked on.

## Design

### Active session registry

A JSON file at `.hero/.active-sessions.json` maps session IDs to the spec
being worked on:

```json
{
  "sessions": {
    "session-abc": {
      "spec": "csv-export",
      "command": "/deliver",
      "started": "2026-04-22T14:00:00Z"
    }
  }
}
```

- Multi-session aware — multiple sessions can work on different specs
- Auto-pruned after 24h of inactivity
- Gitignored — ephemeral local state

### Context injection enrichment

`hero context` and `hero_context` (MCP) check for active sessions and
include the full spec content as the highest-priority context entry,
tagged as `[ACTIVE SPEC]`. This means post-compaction context calls
automatically surface the right spec without any agent logic changes.

### CLI and MCP tools

- `hero active` — list active sessions
- `hero active register <session-id> <slug> <command>` — mark a spec active
- `hero active unregister <session-id>` — clear when done
- `hero active prune` — remove stale entries
- `hero_active` MCP tool — same operations via MCP

## Changes

- `internal/active/active.go` — registry load/save, register/unregister, prune, active spec listing
- `internal/active/active_test.go` — table-driven tests for register, unregister, prune, duplicates
- `internal/cli/active.go` — `hero active` CLI with register/unregister/prune subcommands
- `internal/cli/context.go` — enrich context output with active spec content
- `internal/cli/root.go` — register activeCmd
- `internal/serve/mcp.go` — `hero_active` MCP tool, active spec enrichment in `hero_context`
- `.hero/.gitignore` — exclude `.active-sessions.json`

## Acceptance Criteria

- WHEN a session registers an active spec via `hero active register` THE SYSTEM SHALL persist the session-to-spec mapping in `.hero/.active-sessions.json`
- WHEN `hero context` or `hero_context` runs with active sessions THE SYSTEM SHALL include the full spec content as the highest-priority context entry tagged `[ACTIVE SPEC]`
- WHEN `hero active prune` runs THE SYSTEM SHALL remove entries older than 24 hours
- WHEN multiple sessions are active on different specs THE SYSTEM SHALL include all active specs in context output
- WHEN a session unregisters THE SYSTEM SHALL remove only that session's entry without affecting others
- WHEN no sessions are active THE SYSTEM SHALL NOT modify context output (no empty active-spec section)

## Boundaries

- Does NOT replace OpenCode's conversation summarization
- Does NOT track file edits — git handles that
- Does NOT provide a "better summary" — just ensures the active spec survives compaction
- Does NOT require the agent to manually call register — the `/deliver` and `/diagnose` commands will eventually call it automatically, but for now the agent or MCP tool does it explicitly
