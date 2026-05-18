---
title: Hero Team Server — Shared Job Queue, Approval Gates, and Team Coordination
slug: hero-team-server
type: feature
status: planning
priority: P1
tags: [team, server, jobs, approval, coordination, platform]
created: 2026-04-23
relations:
  - target: hero-platform
    kind: parent
  - target: hero-runner
    kind: depends-on
  - target: hero-automations
    kind: related
horizon: next
smoke: deferred
---

## Goal

Upgrade `hero serve` with a shared job queue, approval gate system, and
team coordination layer so multiple engineers and agents can see each
other's work, queue jobs, and manage approvals — without Hero Cloud.

## Problem

When a team uses Hero, each person works in their own session bubble.
Alice delivers auth-flow on her laptop. Bob queues 3 bug fixes on his.
An automation diagnoses 2 bugs overnight on a CI box. Nobody sees the
full picture. There's no way to say "don't start on this, Alice is already
delivering it" or "this diagnosis needs a human review before we ship."

Hero already has per-session events (hero feed) and claims (hero claim).
What's missing is a central coordinator that aggregates state across all
sessions and manages work handoffs.

## Design

### `hero serve --team`

Extends the existing `hero serve` daemon with team capabilities:

```bash
hero serve --team                    # enable team mode
hero serve --team --port 7437        # custom port (default: 7437)
hero serve --team --auth-token xxx   # require auth for API access
```

### Job queue

The server maintains a job queue backed by SQLite (same pattern as the
spec index). Jobs are created by:

- `hero run` invocations (local or remote)
- Automation triggers
- Team members via the dashboard or CLI

```
hero jobs                          # list all jobs (local + team)
hero jobs submit deliver csv-export --autopilot  # submit to team queue
hero jobs cancel <id>              # cancel a queued/running job
hero jobs <id>                     # show job details and log
```

Job states: `queued → running → completed | failed | cancelled | awaiting_approval`

### Runner workers

The team server spawns runner workers (configurable count) that pull jobs
from the queue and execute them via `hero run` internally:

```json
{
  "serve": {
    "team": {
      "workers": 2,
      "default_model": "claude-sonnet-4-6",
      "default_budget": 5.00,
      "max_concurrent": 3
    }
  }
}
```

### Approval gates

When a job or automation hits an approval gate:

1. Job status moves to `awaiting_approval`
2. The server emits a notification (configurable: webhook, feed event)
3. Engineers review via:
   - `hero approve <job-id>` (CLI)
   - Dashboard approve button (web UI)
   - Slack bot (future integration)
4. On approval: job resumes from where it paused
5. On rejection: job is cancelled, reason logged

### Team state API

New HTTP endpoints on `hero serve --team`:

| Method | Path | Description |
|---|---|---|
| GET | `/api/jobs` | List all jobs with status |
| POST | `/api/jobs` | Submit a new job |
| GET | `/api/jobs/:id` | Job details + log |
| POST | `/api/jobs/:id/approve` | Approve a gated job |
| POST | `/api/jobs/:id/reject` | Reject a gated job |
| POST | `/api/jobs/:id/cancel` | Cancel a job |
| GET | `/api/team/status` | Who's working on what |
| GET | `/api/automations` | Automation status |
| GET | `/api/automations/:id/log` | Automation execution log |

### Session registration

When a team member starts a session (Claude Code, Cursor, etc.), the MCP
server registers with the team server:

```
POST /api/sessions { agent: "claude-code", user: "alice", spec: "auth-flow" }
```

This is an extension of the `hero active register` feature. The team server
aggregates active sessions so the dashboard can show "Alice is delivering
auth-flow, Bob is diagnosing login-crash."

### Team authentication

Three tiers, same server code:

| Tier | Config | How it works |
|---|---|---|
| None | `auth: "none"` | No auth — solo use, trusted network |
| Token | `auth: "token"` | Shared secret in `HERO_AUTH_TOKEN` env var |
| OAuth | `auth: "github-oauth"` | GitHub/Google/Okta OAuth — ties into existing identity |

```bash
# Admin sets up (once)
hero serve --team --auth github-oauth
export HERO_OAUTH_CLIENT_ID=...
export HERO_OAUTH_CLIENT_SECRET=...
export HERO_OAUTH_ORG=your-github-org   # restrict to org members

# Developer connects (once per machine)
hero connect team https://hero.internal:7437
# → opens browser → GitHub OAuth → stores token in ~/.hero/credentials
# → from now on, MCP sessions and hero run auto-authenticate
```

OAuth flow:
1. `hero connect team <url>` opens a browser to the team server's `/auth/login`
2. Server redirects to GitHub/Google OAuth consent screen
3. On callback, server issues a Hero session token (JWT, 30-day expiry)
4. CLI stores the token in `~/.hero/credentials`
5. All subsequent API calls include the token in `Authorization: Bearer`

### Credential brokering

The team server holds org-level API keys so individual devs don't need them:

```json
{
  "serve": {
    "team": {
      "api_keys": {
        "anthropic": "${ANTHROPIC_API_KEY}",
        "openai": "${OPENAI_API_KEY}",
        "azure_openai": "${AZURE_OPENAI_KEY}"
      },
      "azure_endpoint": "${AZURE_OPENAI_ENDPOINT}"
    }
  }
}
```

When a developer runs `hero run deliver csv-export`, the CLI checks:
1. Local API key? Use it (solo mode).
2. Connected to team server? Submit job to server, which uses the org key.

The server tracks per-user usage: API calls, tokens, cost. This enables:
- Per-user daily budgets (`budget_per_user_day: 20.00`)
- Usage dashboards ("Alice: $14 this week, automations: $8")
- Org-level cost caps

### Tracker credentials

Same broker pattern — the server holds service account tokens for
Jira/GitHub/Linear. Individual devs don't need tracker admin access.
When hero needs to post a comment or update an issue, the request goes
through the team server.

### Configuration

```json
{
  "serve": {
    "team": {
      "enabled": true,
      "auth": "github-oauth",
      "oauth_org": "your-github-org",
      "workers": 2,
      "default_model": "claude-sonnet-4-6",
      "default_budget": 5.00,
      "budget_per_user_day": 20.00,
      "usage_tracking": true,
      "api_keys": {
        "anthropic": "${ANTHROPIC_API_KEY}",
        "openai": "${OPENAI_API_KEY}"
      },
      "notifications": {
        "webhook": "https://hooks.slack.com/xxx",
        "events": ["approval_needed", "job_failed", "automation_complete"]
      }
    }
  }
}
```

## Changes

- `internal/serve/jobs.go` — job queue (SQLite-backed), job state machine, worker pool
- `internal/serve/approval.go` — approval gate logic, notification dispatch
- `internal/serve/team.go` — team state aggregation, session registration
- `internal/serve/auth.go` — OAuth flow (GitHub/Google), token auth, JWT issuance
- `internal/serve/credentials.go` — credential brokering, per-user usage tracking, budget enforcement
- `internal/serve/api_jobs.go` — HTTP API handlers for jobs, approvals, team status
- `internal/serve/api_auth.go` — auth endpoints (/auth/login, /auth/callback, /auth/me)
- `internal/cli/jobs.go` — `hero jobs` command (list, submit, cancel, inspect, approve)
- `internal/cli/connect.go` — `hero connect team <url>` command, token storage
- `internal/cli/root.go` — register commands
- `internal/config/config.go` — team server configuration structs

## Acceptance Criteria

- WHEN `hero serve --team` runs THE SYSTEM SHALL start the job queue, worker pool, and team API endpoints
- WHEN `hero jobs submit deliver <slug>` is called THE SYSTEM SHALL enqueue a job and return a job ID
- WHEN a worker is available THE SYSTEM SHALL dequeue the next job and execute it via hero run
- WHEN a job hits an approval gate THE SYSTEM SHALL move it to `awaiting_approval` and emit a notification
- WHEN `hero approve <job-id>` is called THE SYSTEM SHALL resume the gated job
- WHEN an MCP session starts THE SYSTEM SHALL register itself with the team server if `team.enabled` is true
- WHEN `GET /api/team/status` is called THE SYSTEM SHALL return all active sessions and running jobs
- WHEN `hero connect team <url>` is called THE SYSTEM SHALL authenticate via the configured auth method and store the session token locally
- WHEN a job is submitted and the server holds an org API key THE SYSTEM SHALL use the org key for LLM calls instead of requiring a per-user key
- WHEN `auth: "github-oauth"` is configured THE SYSTEM SHALL authenticate users via GitHub OAuth and restrict access to the configured org
- WHEN `budget_per_user_day` is set and a user exceeds it THE SYSTEM SHALL reject new job submissions with a budget-exceeded error
- WHEN `usage_tracking` is enabled THE SYSTEM SHALL record per-user API call counts, token usage, and estimated cost
- THE SYSTEM SHALL persist job state across server restarts via SQLite

## Boundaries

- Does **not** replace Hero Cloud — this is the self-hosted version
- Does **not** manage user authentication (uses optional auth token)
- Does **not** support cross-network job execution (jobs run on the server machine)
- Does **not** provide Slack/Teams bot integration in this feature — uses webhooks
- Does **not** require internet access — works on a local network
