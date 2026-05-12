---
title: Reasoning Log Per Session — Structured Agent Decision Trail
type: feature
status: completed
milestone: v0.4
tags: [reasoning, log, sessions, decisions, audit, agents]
created: 2026-04-13
relations:
  - target: agent-contribution-tracking
    kind: related
  - target: spec-triage
    kind: related
  - target: knowledge-contradiction-detection
    kind: related
horizon: now
---

## Goal

Capture a structured, queryable log of agent reasoning decisions during a session — which context was retrieved, what conventions were applied, what tradeoffs were considered, and what was ultimately done — so the team can audit, replay, and learn from agent work.

## Problem

Agents produce outputs (code, docs, specs), but the reasoning that led to those outputs is lost. When something goes wrong — or more importantly, when something goes right — there's no way to understand why the agent made the choices it did. This makes it impossible to:

- Debug agent mistakes ("why did it use the wrong pattern?")
- Reproduce successful approaches ("what context did it have when it wrote that handler?")
- Build institutional knowledge from agent decisions
- Train the next generation of conventions from observed agent behavior

Linear's approach: every AI-assisted action is logged with its inputs and rationale. Hero's equivalent: a session reasoning log that captures the Hero context calls, knowledge lookups, and conventions the agent acted on.

## Design

### Session Model

A session is a bounded unit of agent work, identified by:
- A session ID (UUID, auto-generated)
- An optional name (`--session my-feature-work`)
- A start time and optional end time
- The spec slug being worked on (if known)
- The agent identity (from `HERO_AGENT` or `--agent`)

Sessions are started explicitly or implicitly:

```
hero session start                        # start a new session
hero session start --name "add-handler"   # named session
hero session end                          # end current session
hero session log                          # show log for current session
hero session log <session-id>             # show log for a specific session
hero session list                         # list sessions
```

Implicit start: if `HERO_SESSION` is not set, the first Hero command in a process tree starts a session automatically (a new UUID is generated and exported to `HERO_SESSION`).

### Reasoning Log Format

Each entry in the reasoning log is newline-delimited JSON, appended to `.hero/sessions/<session-id>.jsonl`:

```json
{"t":"2026-04-13T14:22:01Z","event":"context_retrieved","files":["internal/api/user_handler.go"],"knowledge_entries":["conventions/api-handlers","conventions/go-code-style"],"session":"abc123"}
{"t":"2026-04-13T14:22:05Z","event":"knowledge_applied","slug":"conventions/api-handlers","excerpt":"Handlers return structured JSON errors with code and message fields","session":"abc123"}
{"t":"2026-04-13T14:22:12Z","event":"convention_checked","slug":"conventions/go-code-style","result":"pass","session":"abc123"}
{"t":"2026-04-13T14:22:40Z","event":"spec_claimed","slug":"hero-ask","session":"abc123"}
{"t":"2026-04-13T14:58:00Z","event":"spec_completed","slug":"hero-ask","session":"abc123","duration_minutes":36}
{"t":"2026-04-13T14:58:01Z","event":"session_end","session":"abc123","specs_completed":1,"hero_calls":12}
```

### Event Types

| Event | Description |
|---|---|
| `session_start` | Session initiated |
| `session_end` | Session ended, summary stats |
| `context_retrieved` | `hero context` called — which files, which knowledge returned |
| `knowledge_applied` | A specific knowledge entry was retrieved and presumably read |
| `convention_checked` | A convention was explicitly checked (via `hero check`) |
| `ask_answered` | `hero ask` was called — question and answer recorded |
| `spec_claimed` | A spec was claimed in this session |
| `spec_completed` | A spec was marked done in this session |
| `contradiction_found` | A knowledge contradiction was surfaced |
| `triage_warning` | A triage warning was raised during this session |

### Session Replay

```
hero session replay <session-id>
```

Renders a human-readable summary of a session:

```
Session: abc123 — add-handler (2026-04-13 14:22 → 14:58, 36 min)
Agent: opencode/claude-sonnet-4-5

Context retrieved (2 calls):
  - internal/api/user_handler.go
    → conventions/api-handlers, conventions/go-code-style

Knowledge applied:
  - conventions/api-handlers: "Handlers return structured JSON errors..."
  - conventions/go-code-style: passed check

Specs:
  - hero-ask: claimed → completed (36 min)

12 total Hero calls.
```

### Knowledge Distillation

`hero session distill <session-id>` analyzes a session log and suggests new knowledge entries based on observed patterns:

```
$ hero session distill abc123

Observed patterns not in knowledge base:

  ? The agent consistently retrieved conventions/api-handlers before editing handlers.
    Suggest: add "always load api-handlers context when editing internal/api/" as a rule.

  ? The agent applied a nil-check pattern 3 times not documented in go-code-style.
    Suggest: review and add to conventions/go-code-style.
```

These are suggestions only — the engineer reviews and manually adds to the knowledge base (or uses `hero add`).

### Session Index

`.hero/sessions/index.jsonl` maintains a lightweight session index (one line per session) for fast `hero session list`:

```json
{"id":"abc123","name":"add-handler","agent":"opencode/claude-sonnet-4-5","start":"2026-04-13T14:22:00Z","end":"2026-04-13T14:58:00Z","specs_completed":1}
```

Sessions older than `session_retention_days` (default: 30) are pruned on `hero session list` or explicit `hero session prune`.

## Changes

- `internal/sessions/sessions.go` — session model, ID generation, lifecycle management
- `internal/sessions/log.go` — reasoning log writer (append-only JSONL)
- `internal/sessions/replay.go` — session replay renderer
- `internal/sessions/distill.go` — pattern suggestion from session log
- `internal/cli/session.go` — `hero session` command group
- `internal/cli/context.go` — emit `context_retrieved` events when `hero context` runs
- `internal/cli/ask.go` — emit `ask_answered` events when `hero ask` runs
- `internal/config/config.go` — `sessions` config section
- `.hero/sessions/` — new directory, gitignored by default (configurable)

## Acceptance Criteria

- `hero session start` creates a new session and sets `HERO_SESSION` in environment
- All Hero commands within a session emit appropriate log events
- `hero session log` renders a human-readable session timeline
- `hero session replay <id>` renders a full session summary
- `hero session list` shows sessions from the index with agent, duration, specs completed
- `hero session distill <id>` suggests knowledge entries from observed patterns
- Session files are stored in `.hero/sessions/` as `<session-id>.jsonl`
- Sessions are pruned after `session_retention_days` days
- `.hero/sessions/` is gitignored by default (personal/ephemeral data)
- Session logging does not meaningfully slow down Hero commands (< 1ms overhead per event)

## Boundaries

- Does **not** record agent prompts or model outputs — only Hero's own inputs/outputs
- Does **not** require a session to be active — all Hero commands work normally without one
- Does **not** automatically create knowledge entries — `distill` output is suggestions only
- `.hero/sessions/` is gitignored by default; teams can opt in to committing sessions
