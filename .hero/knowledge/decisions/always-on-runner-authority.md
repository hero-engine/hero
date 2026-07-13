---
title: "Always-on execution authority"
slug: always-on-runner-authority
type: decision
status: proposed
domain: engineering
created: 2026-07-13
tags: [runtime, workers, scheduling, cross-repo]
relates-to: [hero-platform, hero-runner, hero-team-server, hero-chat-and-model]
---

# Always-on execution authority

## Kickoff

Reconciles Hero's two competing headless-runtime stories before scheduling, remote workers, or Cloud execution are expanded.

**Status:** proposed — source and the completed adapter decision disagree; an explicit owner choice is required.

**Pick up at:** review the three options and accept or revise the proposed worker-adapter boundary.

→ `.hero/knowledge/decisions/always-on-runner-authority.md`

**Files:** `internal/runner/runner.go`, `internal/serve/workers.go`, `internal/serve/chat/adapter.go`, `../hero-code/crates/hero-core/src/session_runtime.rs`

## Context

Hero currently has two incompatible architectural claims:

- The completed `hero-chat-and-model` decision says `hero serve` does not run inference. It dispatches work to an adapter, with hero-code as the canonical interactive and headless adapter.
- Current source contains a Go agent loop in `internal/runner` and `hero serve --team` workers invoke it directly.
- hero-code separately owns mature Swift and Rust provider, tool, permission, memory, and session runtimes.
- Hero Cloud's charter describes governed workflow coordination, but does not currently host an inference loop or worker fleet.

Expanding schedules or remote execution without resolving this would create two full runtimes, two credential paths, and divergent cancellation, transcript, permission, and tool semantics.

## Options considered

### A. Make the Go runner canonical

`hero` owns inference, providers, tools, transcripts, and workers. hero-code becomes a UI client.

This centralizes execution, but requires replatforming hero-code's richer native runtime and duplicates much of its existing work during migration.

### B. Make registered worker adapters authoritative

`hero` owns versioned job, event, schedule, execution-target, and worker-registration contracts plus local/self-hosted coordination. A registered worker owns inference and native tool execution. hero-code is the first supported full worker. The Go runner becomes a minimal compatibility worker behind the same contract.

This preserves local-first operation and existing source while preventing the compatibility runner from becoming a second independent product runtime.

### C. Permanently support two full runtimes

The Go runner and hero-code evolve independently but accept common jobs.

This minimizes near-term migration, but permanently doubles provider, tool, permission, transcript, and behavioral parity costs.

## Proposed decision

Choose **Option B**.

Ownership becomes:

| Concern | Authority |
|---|---|
| Job and lifecycle-event schemas | `hero/contracts/runtime` |
| Local/self-hosted queue, schedules, leases and coordination | `hero` |
| Inference, provider credentials and native tools | Registered worker adapter |
| First supported full worker | `hero-code` |
| Minimal CLI/local compatibility worker | `hero/internal/runner`, behind the worker contract |
| Multi-tenant persistence, governance, triggers and worker coordination | `hero-cloud` |
| Schedule editing and run presentation | `hero-code` |
| Durable cron clock | `hero serve` locally; `hero-cloud` for hosted teams |

The compatibility runner may remain for `hero run` and bootstrap use, but it does not gain an independent provider/tool roadmap. Managed Hero Cloud compute is deferred; Cloud initially coordinates customer-hosted workers.

## Required call graphs

### Solo/local

`hero-code or hero CLI → local hero serve job API → registered local worker → lifecycle events → clients`

The compatibility worker may be in-process, but it must consume and emit the same contract as an external worker.

### Self-hosted team

`clients → hero serve --team → durable queue/leases → registered customer workers → event log/approvals`

### Hero Cloud

`tracker/webhook/schedule → governed Cloud control plane → registered customer worker → audited events/results`

Cloud does not execute repository tools in the initial program.

## Migration

1. Add the executor-neutral v1 job contract without changing current runtime behavior.
2. Add translation adapters for the existing `serve.Job`, `runner.JobRecord`, and hero-code session runtime.
3. Route `WorkerPool` execution through a worker interface; keep the current Go runner as the first compatibility implementation.
4. Register hero-code as a full worker and prove parity for cancellation, approval, budgets, events, and results.
5. After a soak period, decide whether the compatibility worker remains supported or is deprecated. Removal requires a separate decision and migration spec.

## Consequences

- Hero can coordinate multiple execution implementations without encoding providers or native tools in its control-plane contract.
- hero-code gains a service/headless responsibility in addition to its desktop UX.
- The existing Go runner needs an adapter boundary and parity tests rather than continued standalone expansion.
- Cloud can deliver an always-on product before taking on arbitrary customer-code sandbox operations.
- Contract evolution must be additive and versioned because three repositories consume it.

## Acceptance checklist

- [ ] Confirm hero-code is the supported full worker rather than only a desktop client.
- [ ] Confirm the Go runner is a compatibility worker, not the canonical provider/tool roadmap.
- [ ] Confirm Cloud coordinates customer-hosted workers before managed execution.
- [ ] Confirm interactive and headless runs share job/event semantics even when their transports differ.
- [ ] Confirm credentials never cross the job contract; workers resolve them within their trust boundary.

