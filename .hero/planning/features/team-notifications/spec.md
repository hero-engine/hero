---
title: Team Notifications — Webhook Alerts for Job Events
type: feature
status: planning
priority: P1
tags: [team, notifications, webhooks, slack, events]
created: 2026-04-25
relations:
  - target: hero-team-server
    kind: parent
horizon: next
smoke: deferred
---

## Goal

Notify the team when important job events happen — approvals needed,
jobs failed, automations completed, budget exceeded. Deliver via
configurable webhooks (Slack, Teams, Discord, or any HTTP endpoint).

## Problem

The team server queues and executes jobs, but nobody knows what's
happening unless they poll `hero jobs` or watch the dashboard. When a
job hits an approval gate at 2am, nobody sees it until morning. When
an automation fails, the error sits in a log. Notifications push events
to where the team already is.

## Design

### Configuration

```json
{
  "serve": {
    "team": {
      "notifications": {
        "webhook_url": "https://hooks.slack.com/services/xxx",
        "events": [
          "approval_needed",
          "job_completed",
          "job_failed",
          "budget_exceeded",
          "automation_error"
        ],
        "format": "slack"
      }
    }
  }
}
```

### Webhook formats

| Format | Description |
|---|---|
| `slack` | Slack Block Kit payload with action buttons |
| `teams` | Microsoft Teams Adaptive Card |
| `discord` | Discord embed |
| `generic` | Plain JSON POST to any endpoint |

### Slack example

```json
{
  "blocks": [
    {
      "type": "header",
      "text": {"type": "plain_text", "text": "⏸ Approval needed"}
    },
    {
      "type": "section",
      "text": {
        "type": "mrkdwn",
        "text": "*Job:* deliver csv-export\n*Submitted by:* alice\n*Budget:* $5.00"
      }
    },
    {
      "type": "actions",
      "elements": [
        {"type": "button", "text": {"type": "plain_text", "text": "Approve"}, "url": "http://hero:7437/api/jobs/123/approve"},
        {"type": "button", "text": {"type": "plain_text", "text": "Reject"}, "url": "http://hero:7437/api/jobs/123/reject"}
      ]
    }
  ]
}
```

### Event types

| Event | Trigger | Payload |
|---|---|---|
| `approval_needed` | Job moves to `awaiting_approval` | Job ID, command, args, submitted_by |
| `job_completed` | Job finishes successfully | Job ID, turns, cost, duration |
| `job_failed` | Job fails or errors out | Job ID, error message |
| `budget_exceeded` | Job or user hits budget limit | Job ID, user, cost, budget |
| `automation_error` | Automation trigger fails | Automation name, error |

### Notification dispatch

A `Notifier` runs in the server and listens for job state changes.
When a matching event occurs, it formats the payload and POSTs to the
webhook URL. Failed deliveries are retried once after 30 seconds.

```go
type Notifier struct {
    webhookURL string
    events     map[string]bool
    format     string
}

func (n *Notifier) Notify(event string, data map[string]interface{}) error
```

### Integration points

- `JobQueue.Update()` — check for state transitions that trigger notifications
- `WorkerPool.executeJob()` — notify on completion/failure
- Automations engine — notify on automation errors

## Changes

- `internal/serve/notifications.go` — Notifier, webhook formatting (Slack/Teams/Discord/generic), retry logic
- `internal/serve/workers.go` — notify on job completion/failure
- `internal/serve/jobs.go` — notify on approval gate transitions
- `internal/config/config.go` — NotificationConfig struct

## Acceptance Criteria

- WHEN a job moves to `awaiting_approval` and `approval_needed` is in the events list THE SYSTEM SHALL POST a notification to the configured webhook URL
- WHEN a job completes and `job_completed` is in the events list THE SYSTEM SHALL POST a notification with job summary
- WHEN a job fails and `job_failed` is in the events list THE SYSTEM SHALL POST a notification with the error
- WHEN the webhook format is `slack` THE SYSTEM SHALL format the payload as Slack Block Kit with action buttons
- WHEN the webhook format is `generic` THE SYSTEM SHALL POST a plain JSON payload
- WHEN a webhook delivery fails THE SYSTEM SHALL retry once after 30 seconds
- WHEN no webhook is configured THE SYSTEM SHALL skip notification dispatch silently

## Boundaries

- Does **not** implement a Slack bot (bidirectional) — webhook is one-way push
- Does **not** support email notifications — webhooks only
- Does **not** queue notifications for offline delivery — fire and forget with one retry
- Does **not** filter notifications per user — all configured events go to one webhook
