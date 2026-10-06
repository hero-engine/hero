---
title: "Next-step engine — one deterministic primary action per work item"
slug: next-step-engine
type: feature
status: planning
priority: high
domain: engineering
created: 2026-10-06
parent: hero-read-contract
depends-on: [work-item-model]
---

# Next-step engine

## Kickoff

Compute `NextStep` for each `WorkItem`: action, label, slash command, phase {label, state}, enabled, reason, and extras. Base it on `read-contract-v1`'s table. Keep the requester's rules:
- one primary action, never Diagnose and Deliver at once;
- `Delivered` only after a passed `hero spec verify`;
- completed with weak or missing verify → Verify, never Deliver.

`reason` names unmet dependencies ("waits on X"). Align with `internal/drive`'s judge.

→ `/design next-step-engine`

## Scope

- **In:** a pure function over `WorkItem` plus relations, table-tested per type × status × verify. Commands are chat-sendable slash commands, never shell.
- **Out:** polish and suggested.
