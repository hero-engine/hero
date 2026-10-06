---
title: "Work-item model — one WorkItem for every read tool"
slug: work-item-model
type: feature
status: planning
priority: critical
domain: engineering
created: 2026-10-06
parent: hero-read-contract
depends-on: [read-contract-v1]
---

# Work-item model

## Kickoff

Build the shared Go model behind the read contract: `WorkItem` for every work spec. It carries slug, unquoted title, type, status, priority, severity, size, path, parent, progress (children verified / total), lane, timestamps, tracker, verify state, and a per-item revision. Use `read-contract-v1`'s rules and reuse `internal/spec`, `internal/acceptance` and `internal/projection`.

→ `/design work-item-model`

## Scope

- One package (e.g. `internal/workmodel`) that builds `[]WorkItem` from the corpus deterministically.
- Lane and verify derivation per the decision. Revision = a hash of the file plus its relation targets' statuses.
- No MCP surface here; the tools consume it.
