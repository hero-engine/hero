---
title: "Always-On Runtime"
slug: always-on-runtime
type: initiative
status: planning
domain: engineering
size: x-large
priority: P0
parent: hero-platform
child:
  - job-run-contract-v1
tags: [runtime, jobs, workers, scheduling, automation]
created: 2026-07-13
relates-to: [hero-runner, hero-team-server, hero-automations, team-connect, hero-team-experience]
---

# Always-On Runtime

## Vision

Hero work can continue after the initiating chat closes, wake on engineering events or schedules, pause for human input or approval, and return a durable, reviewable result. Local, self-hosted, desktop, and Cloud modes use one lifecycle contract rather than separate schedulers and agent runtimes.

## Goal

Deliver the versioned runtime contracts and reliable local/self-hosted execution spine that hero-code and Hero Cloud can consume. The first milestone proves a scheduled local workflow can run while the desktop is closed and later replay its complete history.

## Kickoff

Builds the shared contracts and execution spine for durable Hero jobs, schedules, workers, approvals, and reconnectable events.

**Status:** planning — runner authority is proposed; the v1 contract is ready to design behind that gate.

**Pick up at:** resolve `always-on-runner-authority`, then deliver `job-run-contract-v1`.

→ `.hero/planning/initiatives/always-on-runtime/job-run-contract-v1/spec.md`

**Files:** `contracts/`, `internal/serve/jobs.go`, `internal/serve/workers.go`, `internal/automations/engine.go`, `internal/runner/runner.go`

## Context

The existing source already includes a headless runner, SQLite job queue, local worker pool, approvals, sessions, usage records, and automation parsing/matching. The missing product is the reliable contract and wiring between those pieces:

- current job and runner records have competing lifecycle semantics;
- workers have no durable identity, lease, heartbeat, retry, or stale-run recovery;
- automation matches do not enqueue production jobs;
- scheduled maintenance is hard-coded rather than user-defined;
- notification and client surfaces have no replayable canonical event vocabulary.

This initiative is a child of `hero-platform`. It narrows that broad initiative to the runtime foundation and leaves live team UX in `hero-team-experience`.

## Specs

| Order | Slug | Priority | Size | State | Design kickoff |
|---:|---|---:|---:|---|---|
| 0 | `always-on-runner-authority` | P0 | decision | proposed | Decide which process owns inference/tools and the role of the Go compatibility runner. |
| 1 | `job-run-contract-v1` | P0 | medium | designed | Version portable job snapshots, lifecycle events, outcomes, controls, targets, and compatibility rules. |
| 2 | `worker-adapter-protocol` | P0 | large | stub | Design worker registration, capabilities, leases, heartbeats, event streaming, cancellation, and resumable input/approval. |
| 3 | `hero-team-server` | P0 | medium remaining | reuse | Audit the delivered source against the existing spec, then harden CAS transitions, cancellation, recovery, retries, and logs. |
| 4 | `automation-trigger-execution-wiring` | P0 | medium | stub | Connect existing trigger matching to idempotent job submission and durable automation-run history. |
| 5 | `user-scheduled-workflows` | P1 | medium | stub | Persist cron/timezone schedules with pause, resume, run-now, overlap and missed-run policies. |
| 6 | `job-notification-event-contract` | P1 | small | stub | Define delivery-neutral envelopes for completion, failure, approval, input, budget, and worker-loss events. |

Each stub should be materialized under this initiative before delivery. Do not reopen the completed `hero-automations` history; the wiring gap gets its own spec.

## Dependencies

```text
always-on-runner-authority
  → job-run-contract-v1
      ├─→ worker-adapter-protocol → hero-team-server hardening
      └─→ automation-trigger-execution-wiring → user-scheduled-workflows
      └─→ job-notification-event-contract
```

hero-code's `always-on-agent-experience` and hero-cloud's `automated-workflows` consume this initiative's contracts. Cross-repo dependencies are named in their bodies rather than local `depends-on` fields because local graphs cannot resolve peer slugs.

## Cross-cutting rules

- Durable database state is authoritative; SSE, webhooks, and NATS are projections/transports.
- Submission, event processing, cancellation, and approval must be idempotent.
- No provider credentials or secret material appear in job contracts or events.
- Existing stores migrate additively; old and new shapes coexist until all consumers have moved.
- Local Hero remains useful without Cloud.
- A worker may disappear at any point; leases and replay must make recovery observable and safe.

## Acceptance journeys

- WHEN a local schedule fires while Hero Code is closed THE SYSTEM SHALL execute or explicitly queue the job and preserve replayable history.
- WHEN a worker disappears during a run THE SYSTEM SHALL expire its lease and move the job through a deterministic recovery policy.
- WHEN a client reconnects with its last event cursor THE SYSTEM SHALL replay every later event once in sequence order.
- WHEN cancellation or approval is retried THE SYSTEM SHALL produce one legal state transition and no duplicate execution.
- THE SYSTEM SHALL support local/self-hosted execution without Hero Cloud.

## Boundaries

- No consumer messaging gateway, voice/TTS, or general personal-assistant surface.
- No mandatory NATS dependency.
- No managed cloud sandbox fleet in the near-term program.
- No second independent full inference/provider/tool roadmap.
- No generic workflow DAG engine; hero-code's workflow orchestration remains separate.
- No automatic skill or knowledge mutation without review and audit.

## Risks

- The runner-authority decision can invalidate downstream implementation details if skipped.
- Existing source and specs have drifted; each reused spec needs an as-built audit before delivery.
- A premature distributed design would add operational cost before local scheduling is proven.
- Cross-repo contract changes can strand older clients; fixtures and compatibility tests are release gates.

## Progress

Program composed on 2026-07-13. Runner authority is proposed. `job-run-contract-v1` is the first fully designed child; later children remain actionable stubs.

