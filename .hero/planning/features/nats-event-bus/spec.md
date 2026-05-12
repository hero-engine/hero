---
title: "NATS JetStream Event Bus"
type: feature
status: planning
tags: [cloud, events, infrastructure]
created: 2026-04-19
parent: hero-cloud
depends-on: cloud-api
priority: medium
horizon: next
smoke: deferred
---

## Goal

Add NATS JetStream as a real-time event bus alongside CockroachDB. CockroachDB stores state, NATS moves events. This enables real-time push notifications, async job dispatch, and fan-out event patterns that HTTP polling can't handle efficiently.

## When to Build

Phase 3 (Async Delivery) — when we need:
- Real-time push from cloud to connected CLIs
- Async job queuing for agent execution
- Fan-out events to multiple subscribers (e.g. "spec approved" notifies dashboard + CLI + webhook)

## Design

### Architecture

```
hero-cloud
  ├── CockroachDB (state — what things ARE)
  │     users, orgs, specs, knowledge, audit_log
  │
  └── NATS JetStream (motion — what HAPPENED)
        sync events, notifications, job queue, activity streams
```

### Streams

```
hero.sync.{org_id}        — CLI sync push/pull events
hero.activity.{org_id}    — spec lifecycle events (created, approved, delivered)
hero.jobs.{org_id}        — async delivery job queue
hero.notify.{user_id}     — per-user notification channel
```

### Deployment Modes

Same pattern as CockroachDB:

- **Bundled:** `hero cloud start` manages an embedded NATS server
- **External:** `hero cloud start --nats=nats://your-cluster:4222`

NATS nodes cluster the same way as CockroachDB — run the binary, point at peers, they form a cluster. No special nodes.

### Integration Points

- `cloud/events/bus.go` — thin interface: `Publish(subject, payload)`, `Subscribe(subject, handler)`
- `cloud/events/nats.go` — NATS JetStream implementation
- `cloud/events/noop.go` — no-op for when NATS is unavailable (graceful degradation)
- API handlers publish events after state changes
- CockroachDB remains source of truth — NATS is best-effort delivery

### Why Not CockroachDB Changefeeds Alone?

Changefeeds watch table rows — good for "this row changed" but not for:
- Arbitrary event routing (user X wants notifications for repo Y)
- Job queuing with consumer groups
- Cross-service fan-out
- Backpressure and replay from a point in time

Changefeeds might be enough for simple cases. NATS is for when we outgrow them.

## Acceptance Criteria

- NATS JetStream connects and creates streams on startup
- Events published after spec status changes, sync events, user actions
- CLI can subscribe to real-time updates via `hero cloud watch` or similar
- Graceful degradation: if NATS is unavailable, hero-cloud continues working (events are lost, not queued)
- Bundled mode: `hero cloud init` sets up NATS alongside CockroachDB
- Cluster mode: NATS nodes join CockroachDB nodes on the same boxes
