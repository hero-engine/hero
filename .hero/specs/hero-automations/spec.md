---
title: Hero Automations — Event-Driven Trigger-to-Action Engine
type: feature
status: planning
priority: P1
tags: [automations, triggers, events, platform, jira, github]
created: 2026-04-23
relations:
  - target: hero-platform
    kind: parent
  - target: hero-runner
    kind: depends-on
horizon: next
smoke: deferred
---

## Goal

Let teams define declarative automations that fire agent work in response
to external events — new Jira bug filed, GitHub PR opened, schedule tick,
or webhook received. Automations run via `hero run` and produce specs,
commits, and PRs without anyone sitting in a chat session.

## Problem

Today, someone must open a chat session, notice work needs doing, and type
a command. For a single developer this is fine. For a team, it means bugs
sit undiagnosed until someone looks at Jira, PRs wait for context until
someone runs `/review`, and routine maintenance (dependency updates, doc
freshness) happens only when remembered.

Hero already has the tools to diagnose, deliver, and review. What's missing
is the trigger that says "when X happens, do Y" — and a persistent process
to watch for X.

## Design

### Automation definition

Automations are defined in `.hero/automations/` as YAML files:

```yaml
# .hero/automations/auto-diagnose-bugs.yaml
name: Auto-diagnose new bugs
trigger:
  type: tracker
  event: issue_created
  filter:
    type: Bug
    status: Open
action:
  command: diagnose
  args: "{{issue.tracker_id}}"
  mode: autopilot
  model: claude-sonnet-4-6
  budget: 2.00
approval:
  required: false          # diagnose only, no code changes
  notify: "#bugs-channel"  # optional notification
```

```yaml
# .hero/automations/auto-fix-critical-bugs.yaml
name: Auto-fix critical bugs
trigger:
  type: tracker
  event: issue_updated
  filter:
    type: Bug
    priority: Critical
    hero_status: planning  # has been diagnosed, has a fix plan
action:
  command: deliver
  args: "{{spec.slug}} --autopilot"
  model: claude-sonnet-4-6
  budget: 5.00
approval:
  required: true           # code changes need human review
  gate: in-review          # parks spec at in-review until approved
  notify: "#engineering"
  reviewers: ["@alice", "@bob"]
```

```yaml
# .hero/automations/weekly-health.yaml
name: Weekly health check
trigger:
  type: schedule
  cron: "0 9 * * 1"       # Monday 9am
action:
  command: check
  args: "--reconcile"
approval:
  required: false
```

```yaml
# .hero/automations/pr-context.yaml
name: Auto-context on PR
trigger:
  type: webhook
  event: pull_request.opened
action:
  command: context
  args: "--files {{pr.changed_files}}"
  post_to: pr_comment
```

### Trigger types

| Type | Source | Events |
|---|---|---|
| `tracker` | Jira, GitHub Issues, Linear | issue_created, issue_updated, issue_closed |
| `webhook` | Any HTTP POST | pull_request.*, push, custom |
| `schedule` | Cron expression | periodic |
| `file` | File system watcher | spec_created, spec_completed, knowledge_updated |
| `feed` | Hero event feed | delivery_complete, drift_detected, regression_found |

### Approval gates

When `approval.required: true`:
1. The automation runs the action but stops at the approval gate
2. The spec is parked at the gate status (e.g., `in-review`)
3. Reviewers are notified (Slack, email, or just the Hero feed)
4. A reviewer runs `hero approve <job-id>` or reviews in the dashboard
5. On approval, the automation continues (e.g., deliver the fix)
6. On rejection, the automation stops and logs the reason

### Automation engine

The engine runs inside `hero serve --team` (or `hero serve` in solo mode):

```
hero serve --automations          # enable automation engine
hero automations list             # show active automations
hero automations test <file>      # dry-run an automation against sample data
hero automations enable/disable   # toggle individual automations
hero automations log              # recent automation activity
```

The engine:
1. Loads `.hero/automations/*.yaml` on startup
2. Registers trigger listeners (webhook server, tracker poller, cron scheduler, file watcher)
3. When a trigger fires, evaluates the filter
4. If matched, enqueues a `hero run` job with the action config
5. Tracks job status and handles approval gates

### Template variables

Actions use `{{mustache}}` templates for dynamic values:

| Variable | Source |
|---|---|
| `{{issue.tracker_id}}` | Tracker event payload |
| `{{issue.title}}` | Issue title |
| `{{spec.slug}}` | Matched spec slug |
| `{{pr.changed_files}}` | PR file list (comma-separated) |
| `{{event.type}}` | Event type name |

## Changes

- `internal/automations/engine.go` — automation loader, trigger registry, filter evaluator, job dispatcher
- `internal/automations/triggers.go` — tracker poller, webhook handler, cron scheduler, file watcher, feed listener
- `internal/automations/approval.go` — approval gate logic, reviewer notification, approve/reject commands
- `internal/automations/types.go` — AutomationConfig, TriggerConfig, ActionConfig, ApprovalConfig structs
- `internal/automations/automations_test.go` — unit tests for filter matching, template rendering, gate logic
- `internal/cli/automations.go` — `hero automations` command (list, test, enable, disable, log)
- `internal/cli/approve.go` — `hero approve <job-id>` command
- `internal/cli/root.go` — register commands
- `internal/serve/automations.go` — HTTP endpoints for webhook ingress, automation status API

## Acceptance Criteria

- WHEN a tracker automation is configured and a matching issue is created THE SYSTEM SHALL enqueue a `hero run` job with the specified command and arguments
- WHEN a webhook automation is configured and a matching HTTP POST is received THE SYSTEM SHALL evaluate the filter and enqueue a job if matched
- WHEN a schedule automation is configured THE SYSTEM SHALL fire the action at the specified cron interval
- WHEN an automation has `approval.required: true` THE SYSTEM SHALL park the job at the gate status and notify reviewers
- WHEN `hero approve <job-id>` is called THE SYSTEM SHALL resume the parked job
- WHEN `hero automations list` runs THE SYSTEM SHALL display all configured automations with their trigger type, status, and last-fired time
- WHEN `hero automations test <file>` runs THE SYSTEM SHALL simulate the trigger with sample data and show what would execute without actually running
- THE SYSTEM SHALL log all automation activity to `.hero/automations/log.jsonl`

## Boundaries

- Does **not** replace the tracker integration — uses it for polling
- Does **not** require Hero Cloud — runs in `hero serve` locally
- Does **not** auto-approve anything marked `approval.required: true`
- Does **not** support complex workflow DAGs — automations are single trigger → single action
- Does **not** manage secrets — uses existing `hero.json` tracker config and env vars for API keys
