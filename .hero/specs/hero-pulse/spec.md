---
title: hero pulse — AI Narrative Summary of Sprint and Week State
type: feature
status: completed
milestone: v0.4
tags: [pulse, summary, sprint, narrative, status, reporting]
created: 2026-04-13
relations:
  - target: sprint-from-tracker
    kind: extends
  - target: agent-contribution-tracking
    kind: related
  - target: git-hook-integration
    kind: related
horizon: now
---

## Goal

Give engineers and teams a single command that produces a coherent, human-readable narrative of where the sprint stands — what's done, what's in flight, what's blocked, and what's at risk — without requiring anyone to manually pull status from Jira, GitHub, and Hero's spec corpus.

## Problem

Sprint standups and status updates are a tax on engineering time. The information already exists — in Hero's spec statuses, git commit history, linked tracker issues, and knowledge base decisions — but synthesizing it into a readable summary requires manual work. Teams either skip it (flying blind) or do it slowly (standup theater).

Linear's AI summaries give teams a real-time narrative of what changed. Hero can do the same thing, grounded in the spec corpus and git history, without requiring a cloud connection.

## Design

### CLI Interface

```
hero pulse                     # current sprint status narrative
hero pulse --week              # rolling 7-day summary
hero pulse --since <date>      # summary since a specific date
hero pulse --spec <slug>       # narrative focused on a single spec
hero pulse --json              # structured JSON output
hero pulse --md                # markdown output (for copy-paste into docs/Slack)
```

### Input Sources

`hero pulse` synthesizes from:

1. **Hero spec corpus** — status distribution (planning/delivering/done/stale), recent status changes
2. **Git log** — commits since `--since` date (or sprint start), parsed for conventional commit messages
3. **Linked tracker** — if `hero.json` has a tracker configured, pulls issue status from Linear/Jira/GitHub
4. **Agent contribution data** — if `agent-contribution-tracking` is active, includes agent velocity stats
5. **Knowledge base changes** — new or updated conventions and decisions since the period start

Hero does not make LLM calls. The narrative is assembled by template rendering from structured data — the "AI" framing refers to the agent-friendly output format and the synthesis from multiple structured sources.

### Output Format

**Plaintext narrative (default):**
```
Sprint Pulse — Week of Apr 7, 2026

Done this sprint (3):
  ✓ mcp-tool-filtering — MCP tool routing and profiles fully implemented
  ✓ second-model-review — Review model integration shipping
  ✓ knowledge-contradiction-detection — Contradiction detection live

In flight (2):
  ↻ hero-ask — Semantic query pipeline, 60% complete (last commit: 2 days ago)
  ↻ spec-triage — Intake validation, started Apr 10

At risk (1):
  ⚠ hero-pulse — No commits in 5 days, stale

Knowledge updates (1):
  + conventions/go-code-style updated Apr 11

No blockers detected.
```

**JSON (`--json`):**
```json
{
  "period": { "from": "2026-04-07", "to": "2026-04-13" },
  "done": [...],
  "in_flight": [...],
  "at_risk": [...],
  "knowledge_updates": [...],
  "blockers": [],
  "agent_contributions": {...}
}
```

**Markdown (`--md`):**
Formatted for pasting into Slack, Notion, or a PR description — uses `##` headers, checkboxes, and bullet lists.

### Staleness Detection

A spec is "at risk" if:
- Status is `delivering` but no git commits touch related paths for > N days (default: 3)
- Status is `planning` for > M days without moving (default: 7)
- Linked tracker issue is marked blocked or stalled

Thresholds are configurable in `hero.json`:
```json
{
  "pulse": {
    "stale_delivering_days": 3,
    "stale_planning_days": 7
  }
}
```

### Sprint Boundary Detection

If a tracker is configured, Hero uses the tracker's sprint dates. Otherwise:

- `--week` uses a rolling 7-day window
- `hero.json` can declare a sprint cadence:
  ```json
  {
    "pulse": {
      "sprint_days": 14,
      "sprint_start_day": "monday"
    }
  }
  ```

### MCP Tool

```json
{
  "name": "hero_pulse",
  "description": "Get a narrative summary of current sprint/week state",
  "inputSchema": {
    "type": "object",
    "properties": {
      "since": { "type": "string", "description": "ISO date" },
      "format": { "type": "string", "enum": ["text", "json", "markdown"] }
    }
  }
}
```

This is valuable for agents that need project velocity context before suggesting work: "what's already done, what's in flight, what should I not re-implement?"

## Changes

- `internal/cli/pulse.go` — `hero pulse` command, source aggregation, narrative rendering
- `internal/pulse/pulse.go` — data model, period calculation, staleness detection
- `internal/pulse/render.go` — plaintext, JSON, and markdown renderers
- `internal/serve/mcp.go` — add `hero_pulse` tool
- `internal/config/config.go` — `pulse` config section

## Acceptance Criteria

- `hero pulse` produces a readable narrative with done/in-flight/at-risk sections
- Stale specs (delivering, no commits > N days) appear in "at risk"
- `--week` uses a rolling 7-day window
- `--since <date>` filters to commits and status changes since that date
- `--json` output matches the schema above
- `--md` output is valid GitHub-flavored markdown
- `hero_pulse` MCP tool returns JSON output
- Works with no tracker configured (git log only)
- Works with no git history (spec status only)
- Staleness thresholds are configurable in `hero.json`

## Boundaries

- Does **not** make LLM API calls — narrative is template-rendered from structured data
- Does **not** send notifications — output only; push delivery is a cloud/webhook concern
- Tracker integration uses the same adapter as `sprint-from-tracker` — no new tracker clients
- Does **not** predict future completion dates — observational only
