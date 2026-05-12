---
title: Team Activity Feed — Cross-Session Event Log
type: feature
status: completed
priority: P1
tags: [feed, events, multi-agent, coordination, mcp]
created: 2026-04-22
relations:
  - target: hero-killer-features
    kind: parent
horizon: now
---

## Goal

Give every agent working in a Hero repo instant visibility into what other
agents have done recently, without reading full session transcripts. A
lightweight, append-only event log captures significant actions across all
sessions; `hero feed` surfaces them; the `hero_feed` MCP tool lets agents
query them programmatically. Two agents on the same repo stop stepping on
each other because each can see the other's work at a glance.

## Problem

Hero already tracks per-session reasoning via JSONL session logs
(`internal/sessions/`) and sprint-level narrative via `hero pulse`. Neither
solves the "what happened in the last hour across all agents" question.
When multiple agents work concurrently — different Claude Code sessions,
different terminals, even different machines sharing a repo via git — each
is blind to the others. The result: duplicate work, conflicting edits,
specs claimed by one agent getting unknowingly modified by another, and
decisions made in one session that contradict decisions in another.

`.hero/events.log` already exists for claim events (`internal/tracking/`),
but its schema is claim-specific and nothing reads it as a cross-session
activity feed. The infrastructure is half-built; the feature is not.

## Design

### Event schema

Each event is one JSON line appended to `.hero/events.log`, extending the
existing `ClaimEvent` format with new fields:

```jsonl
{"ts":"2026-04-22T14:32:00Z","type":"spec_created","agent":"opencode/claude","session":"a1b2c3","slug":"csv-export","message":"Created spec for CSV export feature"}
{"ts":"2026-04-22T14:33:12Z","type":"files_modified","agent":"opencode/claude","session":"a1b2c3","slug":"csv-export","message":"Modified internal/api/export.go, internal/api/handlers/export.go"}
{"ts":"2026-04-22T14:35:44Z","type":"decision_made","agent":"cursor/claude","session":"d4e5f6","slug":"csv-export","message":"Chose streaming CSV generation over buffered — memory stays O(1) for large exports"}
{"ts":"2026-04-22T14:40:01Z","type":"blocker_hit","agent":"opencode/claude","session":"a1b2c3","slug":"csv-export","message":"Test failure in auth middleware — export route requires auth token but test fixtures lack one"}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `ts` | RFC 3339 timestamp | yes | When the event occurred |
| `type` | enum string | yes | One of the defined event types |
| `agent` | string | yes | Agent identity (e.g. `opencode/claude`, `cursor/sonnet`) |
| `session` | string | no | Session ID from `hero session`, if available |
| `slug` | string | no | Spec slug, when the event relates to a spec |
| `message` | string | yes | Human-readable description of what happened |

### Event types

| Type | When emitted |
|---|---|
| `spec_created` | Agent creates a new spec via `hero spec new` or manually |
| `spec_updated` | Agent changes spec status, content, or metadata |
| `files_modified` | Agent modifies source files (summary, not full diff) |
| `decision_made` | Agent makes a notable architectural or design choice |
| `blocker_hit` | Agent encounters a blocking issue |
| `delivery_complete` | Agent finishes delivering a spec |

### `hero event` command

```
hero event <type> <message>              # append an event
hero event decision_made "Chose Redis over Postgres for session cache"
hero event files_modified "Updated internal/api/export.go, internal/api/handlers/export.go" --slug csv-export
hero event blocker_hit "Auth middleware rejects test tokens" --slug csv-export --session a1b2c3
```

Flags:
- `--slug <slug>` — associate with a spec (optional)
- `--session <id>` — override session ID (defaults to current `hero session` if active)
- `--agent <name>` — override agent identity (defaults to `$HERO_AGENT` or hostname)

The command validates `type` against the known enum and rejects unknown types
with a non-zero exit code and a list of valid types.

### `hero feed` command

```
hero feed                                # last 20 events (default)
hero feed --since 1h                     # events from the last hour
hero feed --since 2026-04-22T14:00:00Z   # events since a specific timestamp
hero feed --type decision_made           # filter by event type
hero feed --slug csv-export              # filter by spec
hero feed --agent opencode/claude        # filter by agent
hero feed --limit 50                     # control output count
hero feed --format json                  # machine-readable output
```

Default human output:

```
14:40 [blocker]    opencode/claude  csv-export  Test failure in auth middleware — export route requires auth token but test fixtures lack one
14:35 [decision]   cursor/claude    csv-export  Chose streaming CSV generation over buffered — memory stays O(1) for large exports
14:33 [files]      opencode/claude  csv-export  Modified internal/api/export.go, internal/api/handlers/export.go
14:32 [spec]       opencode/claude  csv-export  Created spec for CSV export feature
```

Events are displayed newest-first. `--since` parses both duration strings
(`1h`, `30m`, `2h30m`) and RFC 3339 timestamps.

### MCP tools

Two new MCP tools:

**`hero_feed`** — read the activity feed:

```json
{
  "name": "hero_feed",
  "description": "Query the cross-session activity feed. Returns recent significant events from all agents working in this repo.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "since": { "type": "string", "description": "Duration (1h, 30m) or RFC 3339 timestamp" },
      "type": { "type": "string", "description": "Filter by event type" },
      "slug": { "type": "string", "description": "Filter by spec slug" },
      "limit": { "type": "integer", "description": "Max events to return (default 20)" }
    }
  }
}
```

**`hero_event`** — append an event:

```json
{
  "name": "hero_event",
  "description": "Log a significant event to the cross-session activity feed. Other agents will see this.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "type": { "type": "string", "enum": ["spec_created", "spec_updated", "files_modified", "decision_made", "blocker_hit", "delivery_complete"] },
      "message": { "type": "string" },
      "slug": { "type": "string" }
    },
    "required": ["type", "message"]
  }
}
```

### Backward compatibility with existing events.log

The existing `ClaimEvent` struct in `internal/tracking/tracking.go` writes
events with `event`, `slug`, `agent`, `at` fields. The new `FeedEvent` struct
uses `type` and `ts` instead. The feed reader handles both schemas: if a line
has `event` + `at` (old format), it maps `event` to `type` and `at` to `ts`.
Old claim events appear in the feed as `spec_updated` type. No migration
needed — the log is append-only and both formats coexist.

### AGENTS.md integration

Add a section to `AGENTS.md` instructing agents to log significant events:

> After creating or updating a spec, modifying files, making a notable
> decision, or hitting a blocker, run `hero event <type> <message>` to
> log it. Before starting work, run `hero feed --since 1h` to see what
> other agents have done recently.

## Changes

- `internal/feed/feed.go` — `FeedEvent` struct, `AppendEvent`, `ReadEvents` with filtering (since, type, slug, agent, limit), backward-compatible reader for old `ClaimEvent` format
- `internal/feed/feed_test.go` — table-driven tests: append, read, filter by each dimension, backward compat with old format, malformed line handling
- `internal/cli/feed.go` — `hero feed` command with `--since`, `--type`, `--slug`, `--agent`, `--limit`, `--format` flags
- `internal/cli/event.go` — `hero event` command with type validation and `--slug`, `--session`, `--agent` flags
- `internal/cli/root.go` — register `feedCmd` and `eventCmd`
- `internal/serve/mcp.go` — register `hero_feed` and `hero_event` tools
- `AGENTS.md` — add event logging instructions to agent workflow section

## Acceptance Criteria

- WHEN an agent runs `hero event <type> <message>` with a valid type THE SYSTEM SHALL append a JSON line to `.hero/events.log` containing the timestamp, type, agent identity, and message
- WHEN an agent runs `hero event` with an invalid type THE SYSTEM SHALL reject the command with a non-zero exit code and print the list of valid event types
- WHEN `hero feed` runs without flags THE SYSTEM SHALL display the 20 most recent events from `.hero/events.log` in reverse chronological order
- WHEN `hero feed --since 1h` runs THE SYSTEM SHALL display only events whose timestamp falls within the last hour
- WHEN `hero feed --type decision_made --slug csv-export` runs THE SYSTEM SHALL display only events matching both the specified type and slug
- WHEN the `hero_feed` MCP tool is called with a `since` parameter THE SYSTEM SHALL return the same filtered results as the equivalent `hero feed --since` command in JSON format
- WHEN the `hero_event` MCP tool is called with a valid type and message THE SYSTEM SHALL append the event to `.hero/events.log` identically to the CLI command
- WHEN `.hero/events.log` contains old-format claim events (with `event` and `at` fields) THE SYSTEM SHALL display them in the feed by mapping to the new schema without requiring migration

## Boundaries

- Does **not** replace session logs — complements them with a curated summary layer
- Does **not** require a server or daemon — purely file-based, uses OS-level append semantics
- Does **not** sync across machines — git handles cross-machine propagation
- Events are **append-only** — no editing, deleting, or compacting the log
- Does **not** auto-generate events from session transcripts — agents emit events explicitly
- Does **not** enforce structured formats in the message field — free-text is fine
