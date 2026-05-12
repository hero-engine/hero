---
title: Agent Contribution Tracking — claimed_by Slug and Velocity Metrics
type: feature
status: completed
milestone: v0.3
tags: [agents, tracking, velocity, metrics, attribution, workflow]
created: 2026-04-13
relations:
  - target: hero-pulse
    kind: related
  - target: git-hook-integration
    kind: related
  - target: reasoning-log
    kind: related
horizon: now
---

## Goal

Track which agent (or human) is working on a spec, record completion velocity, and surface per-agent contribution metrics — so teams can understand how agent-assisted work flows through the system and where bottlenecks are.

## Problem

As agent-assisted engineering matures, teams lose visibility into who (or what) is doing the work. A spec goes from `planning` to `done` but there's no record of whether a human wrote it, an agent wrote it with human review, or an agent completed it end-to-end. Without this data, teams can't:

- Optimize agent configurations (which model actually finishes specs?)
- Detect stalls (spec claimed by an agent 3 days ago, nothing happened)
- Build trust in agent contributions (where did this come from?)
- Measure velocity per role/model/agent

## Design

### `claimed_by` Frontmatter Field

Specs gain an optional `claimed_by` field:

```yaml
---
title: Semantic Query Against Knowledge and Specs
slug: hero-ask
status: delivering
claimed_by: opencode/claude-sonnet-4-5
claimed_at: 2026-04-13T14:22:00Z
---
```

`claimed_by` format: `<agent-id>/<model-slug>` or `human/<username>`.

Examples:
- `opencode/claude-sonnet-4-5` — OpenCode agent running Claude Sonnet
- `cursor/gpt-4o` — Cursor agent running GPT-4o
- `human/chet-bellows` — human engineer
- `agent/custom-pipeline` — custom automation

### CLI Interface

```
hero claim <slug>                        # claim a spec for the current agent/user
hero claim <slug> --agent opencode       # claim on behalf of a named agent
hero claim <slug> --release              # release claim (set claimed_by to null)
hero claim <slug> --complete             # mark done and record completion
hero claims                              # list all active claims
hero claims --agent <agent-id>           # filter by agent
hero claims --stale                      # claims with no activity > N days
```

`hero claim` writes `claimed_by` and `claimed_at` to the spec's frontmatter and records the claim event to `.hero/events.log`.

### Agent Identity

The current agent identity is resolved from:

1. `--agent <id>` flag (explicit)
2. `HERO_AGENT` environment variable (set by the AI tool's shell integration)
3. `hero.json` `default_agent` config field
4. `human/<git-config-user>` (fallback)

AI tools that integrate with Hero (OpenCode, Cursor, Claude Code) can set `HERO_AGENT` in their environment to identify themselves.

### Velocity Metrics

`hero velocity` shows contribution metrics:

```
$ hero velocity

Agent Velocity — Last 30 days

  opencode/claude-sonnet-4-5    7 specs done    avg 1.2 days/spec
  human/chet-bellows               4 specs done    avg 3.1 days/spec
  cursor/gpt-4o                 2 specs done    avg 2.4 days/spec

Fastest spec: hero-ask (0.4 days, opencode/claude-sonnet-4-5)
Slowest spec: confluence-wiki-sync (6.1 days, human/chet-bellows)
Currently claimed: 3 specs (2 by agents, 1 by human)
```

Velocity is calculated from `claimed_at` → spec status transition to `done` (via git hook or manual `hero claim --complete`).

```
hero velocity --json           # JSON output
hero velocity --since <date>   # filter window
hero velocity --agent <id>     # single agent breakdown
```

### Stale Claim Detection

A claim is "stale" if `claimed_at` is older than `N` days with no status change or git activity touching spec-related paths:

```json
{
  "tracking": {
    "stale_claim_days": 2
  }
}
```

`hero claims --stale` lists stale claims. `hero pulse` and `hero check` include stale claims in their output.

### Events Log

All claim events append to `.hero/events.log` (newline-delimited JSON):

```json
{"event":"claimed","slug":"hero-ask","agent":"opencode/claude-sonnet-4-5","at":"2026-04-13T14:22:00Z"}
{"event":"completed","slug":"hero-ask","agent":"opencode/claude-sonnet-4-5","at":"2026-04-13T14:58:00Z","duration_minutes":36}
{"event":"released","slug":"hero-pulse","agent":"opencode/claude-sonnet-4-5","at":"2026-04-13T15:10:00Z"}
```

This log is the source of truth for velocity metrics. It's committed to git, so history is preserved.

### MCP Integration

`hero_claim` and `hero_velocity` are exposed as MCP tools:

```json
{
  "name": "hero_claim",
  "description": "Claim a spec for the current agent or release an existing claim",
  "inputSchema": {
    "type": "object",
    "properties": {
      "slug": { "type": "string" },
      "action": { "type": "string", "enum": ["claim", "release", "complete"] }
    },
    "required": ["slug", "action"]
  }
}
```

This allows an agent to self-register as working on a spec before it begins — so Hero can detect stalls and surface them via `hero pulse`.

## Changes

- `internal/tracking/tracking.go` — claim resolution, identity detection, events log writer
- `internal/tracking/velocity.go` — velocity calculation from events log
- `internal/cli/claim.go` — `hero claim` and `hero claims` commands
- `internal/cli/velocity.go` — `hero velocity` command
- `internal/serve/mcp.go` — add `hero_claim` and `hero_velocity` tools
- `internal/config/config.go` — `tracking` config section
- `internal/spec/spec.go` — `claimed_by`, `claimed_at` frontmatter fields

## Acceptance Criteria

- `hero claim <slug>` writes `claimed_by` and `claimed_at` to spec frontmatter
- Agent identity resolution: `--agent` flag > `HERO_AGENT` env > `hero.json` default > git user fallback
- All claim events are appended to `.hero/events.log` as newline-delimited JSON
- `hero velocity` shows per-agent completion counts and average days/spec
- `hero claims --stale` lists specs claimed but with no activity > configured threshold
- `hero pulse` includes stale claims in its "at risk" section
- `hero_claim` MCP tool works end-to-end, allowing agents to self-register
- `hero check` reports stale claims as a health issue
- Velocity is calculated correctly from events log (not from spec file mtimes)
- Multiple concurrent claims on different specs are tracked independently

## Boundaries

- Does **not** enforce exclusive claiming — two agents can claim the same spec; this is a warning, not an error
- Does **not** make network calls — all tracking is local, in git
- Does **not** authenticate agent identity — `claimed_by` is trusted metadata
- Velocity metrics are observational only — no SLA enforcement or alerting in v1
