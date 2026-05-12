---
title: Cloud Notifications and Activity Alerts
type: feature
status: planning
tags: [cloud, notifications, integrations]
created: 2026-04-12
parent: hero-cloud
depends-on: [cloud-api, cloud-sync]
horizon: next
smoke: deferred
---

## Goal

Notify team members of relevant spec activity — status changes, new specs,
stale warnings, sync events. Notifications go to Slack, email, and the
cloud dashboard.

## Design

### Event Types

| Event | Description | Default Channel |
|---|---|---|
| `spec.created` | New spec synced | Dashboard, Slack |
| `spec.status_changed` | Status transition (planning → delivering, etc.) | Dashboard, Slack |
| `spec.claimed` | Someone claimed a spec | Dashboard |
| `spec.stale` | Spec hasn't been modified in N days | Email (weekly digest) |
| `spec.completed` | Spec marked completed | Dashboard, Slack |
| `sync.completed` | Repo sync finished | Dashboard |
| `member.joined` | New member added to org | Dashboard |

### Channels

1. **Dashboard** — in-app notification bell + activity feed (always on)
2. **Slack** — webhook integration, configurable per event type
3. **Email** — digest (daily or weekly), configurable per user

### Configuration

Org admins configure Slack webhook URL and default notification preferences.
Individual users can override preferences for their own notifications.

### Rate Limiting

- Slack: max 1 message per event per channel (deduplicated)
- Email digest: batched, never more than 1/day per user
- Dashboard: real-time but paginated

## Changes

- Cloud service: `notifications/` package — event processing and dispatch
- Cloud service: `notifications/slack.go` — Slack webhook sender
- Cloud service: `notifications/email.go` — email digest builder
- Cloud dashboard: notification preferences UI, notification bell

## Acceptance Criteria

- Dashboard shows real-time activity feed for the org
- Slack notifications work via webhook for configurable event types
- Email digest sends daily or weekly summary of spec activity
- Users can configure their notification preferences
- Org admins can set default Slack webhook and preferences
- Notifications are deduplicated (no duplicate Slack messages for same event)
