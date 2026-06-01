---
title: Hero Dashboard V2 — Visual Interface for Jobs, Automations, and Team State
slug: hero-dashboard-v2
type: feature
status: completed
priority: P1
tags: [dashboard, ui, web, team, platform, visualization, replaced]
created: 2026-04-23
relations:
  - target: hero-platform
    kind: parent
  - target: hero-team-server
    kind: depends-on
  - target: hero-runner
    kind: related
  - target: cto-dashboard
    kind: related
  - target: hero-surface-architecture
    kind: superseded-by
horizon: next
smoke: deferred
completed_at: 2026-05-18T19:25:38Z
---

> **Replaced by the [Hero Surface Architecture](../../initiatives/hero-surface-architecture/spec.md) initiative.** The v2 page catalog is split across the five new homes: Jobs/Automations → [hero-agents-home](../hero-agents-home/spec.md), Specs/Bugs → [hero-work-home](../hero-work-home/spec.md), Team/Velocity → [hero-people-and-roi-home](../hero-people-and-roi-home/spec.md), Knowledge → [hero-knowledge-home](../hero-knowledge-home/spec.md), Overview → [hero-now-home](../hero-now-home/spec.md). Visual grammar reframed as a web app (top nav + scrolling content, not a desktop dashboard). This spec is preserved for history.


## Goal

Build a visual web UI served by `hero serve` that shows jobs, automations,
team activity, and project health — the thing you open in a browser to see
what Hero is doing across your team. Replaces the basic dashboard built in
Phase 5 with a full operational view.

## Problem

Hero has rich data — specs, drift reports, velocity metrics, job logs,
automation status, team sessions — but today it's all CLI output. For a
solo developer, CLI is fine. For a team, you need a screen on the wall
or a browser tab that answers: "what are our agents doing right now?"

The Phase 5 dashboard shows spec counts and activity. It doesn't show
running jobs, automation triggers, approval queues, or cross-repo state.
As Hero becomes a platform with headless execution and automations, the
dashboard must grow to match.

## Design

### Pages

#### 1. Overview (landing page)

The "glanceable" view — everything a team lead needs in one screen:

```
┌─────────────────────────────────────────────┐
│  Hero Dashboard                    team: 3  │
├──────────┬──────────┬───────────┬───────────┤
│ Jobs     │ Specs    │ Drift     │ Health    │
│ 2 running│ 72 done  │ 0 issues  │ ✓ clean   │
│ 1 queued │ 5 in-flt │           │           │
│ 1 review │ 4 plan   │           │           │
├──────────┴──────────┴───────────┴───────────┤
│  Live Activity Feed                          │
│  • Alice delivering auth-flow (turn 14)      │
│  • Automation diagnosed PROJ-456 (2m ago)    │
│  • Bob approved job #23 (5m ago)             │
│  • Automation: weekly health check clean     │
└──────────────────────────────────────────────┘
```

#### 2. Jobs

List of all jobs (running, queued, completed, failed, awaiting approval):

- Real-time status updates via SSE
- Click a job to see its full log (turns, tool calls, files changed)
- Approve/reject buttons for gated jobs
- Cancel button for running/queued jobs
- Filter by status, user, spec, date range

#### 3. Automations

Configuration and status of all automations:

- List of `.hero/automations/*.yaml` with trigger type, status, last fired
- Enable/disable toggles
- Execution log per automation
- "Test" button to dry-run against sample data
- Visual editor for creating new automations (stretch goal)

#### 4. Team

Who's doing what right now:

- Active sessions (user, tool, spec, started at)
- Claims (who has what claimed)
- Recent handoffs (NEXT.md summaries)
- Velocity chart (specs/week per engineer)

#### 5. Specs

Enhanced version of the existing specs page:

- Kanban board view (planning → delivering → in-review → completed)
- Dependency graph (Mermaid rendering)
- Drift indicators inline
- Contract coverage percentage
- Click-through to full spec content

#### 6. Project Health

The executive report as a live page:

- Pulse data (done, in-flight, at-risk)
- Drift summary
- Convention coverage
- Suggestions (high-churn uncovered areas)
- Contract regressions
- Velocity trends (chart)

### Technical approach

The existing dashboard (Phase 5) is built with:
- Go `html/template` for server-rendered HTML
- Embedded static assets via `embed.FS`
- SSE for live updates
- Vanilla JS + CSS (no framework)

V2 continues this approach — no React/Vue build step, no node_modules.
The dashboard stays embeddable in the Go binary. New pages are added as
Go templates with SSE-powered live data.

For charts (velocity trends), use a lightweight JS charting library
embedded as a static asset (e.g., Chart.js or uPlot, ~50KB).

### API integration

All dashboard pages consume the existing + new HTTP API endpoints:

| Page | API endpoints |
|---|---|
| Overview | `/api/status`, `/api/jobs`, `/api/team/status`, `/api/events/stream` |
| Jobs | `/api/jobs`, `/api/jobs/:id`, `/api/jobs/:id/approve` |
| Automations | `/api/automations`, `/api/automations/:id/log` |
| Team | `/api/team/status`, `/api/claims`, `/api/velocity` |
| Specs | `/api/specs`, `/api/specs/:slug`, `/api/drift` |
| Health | `/api/pulse`, `/api/drift`, `/api/suggestions`, `/api/velocity` |

### Configuration

```json
{
  "serve": {
    "dashboard": {
      "enabled": true,
      "theme": "auto",
      "refresh_interval": 5
    }
  }
}
```

## Changes

- `internal/serve/dashboard_v2.go` — new page handlers (jobs, automations, team, health)
- `internal/serve/templates/` — Go HTML templates for each page
- `internal/serve/static/` — CSS, JS, chart library
- `internal/serve/api_drift.go` — new endpoint for drift summary data
- `internal/serve/api_suggestions.go` — new endpoint for suggestions data
- Update existing dashboard routes to V2

## Acceptance Criteria

- WHEN the dashboard loads THE SYSTEM SHALL display the overview page with live job counts, spec counts, drift status, and activity feed
- WHEN a job is running THE SYSTEM SHALL update the jobs page in real-time via SSE showing current turn and status
- WHEN a job is awaiting approval THE SYSTEM SHALL show an approve/reject button on the jobs page
- WHEN the automations page loads THE SYSTEM SHALL list all configured automations with their trigger type, enabled status, and last execution time
- WHEN the team page loads THE SYSTEM SHALL show active sessions, claims, and velocity metrics
- WHEN the specs page loads THE SYSTEM SHALL render specs in a kanban-style board with drift indicators and contract coverage
- WHEN the health page loads THE SYSTEM SHALL display pulse, drift, convention coverage, suggestions, and velocity charts
- THE SYSTEM SHALL serve the dashboard as embedded HTML/CSS/JS with no external build step or CDN dependencies

## Boundaries

- Does **not** require a frontend build toolchain — Go templates + vanilla JS
- Does **not** support creating or editing specs via the UI (read-only + approvals)
- Does **not** replace the CLI — the dashboard is a view layer, not a new interface
- Does **not** require authentication (optional auth-token for team server)
- Does **not** support mobile-specific layouts in V1 (responsive but desktop-first)
