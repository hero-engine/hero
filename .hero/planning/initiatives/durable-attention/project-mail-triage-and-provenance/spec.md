---
title: "Project Mail Triage and Provenance — From Signal to Explicit Work"
slug: project-mail-triage-and-provenance
type: feature
status: planning
domain: engineering
priority: high
size: medium
horizon: next
created: 2026-07-20
parent: durable-attention
depends-on: [project-mail-core, hero-idea-primitive-core]
tags: [mail, intake, provenance, graph, mcp]
---

# Project Mail Triage and Provenance — From Signal to Explicit Work

## Goal

Let a project explicitly triage received mail—reply, acknowledge, dismiss, or
promote it to existing Intake/Spec workflows—while preserving source/thread
provenance and surfacing unread mail through Hero's standard read paths.

## Design inputs

- Promotion is explicit; delivery never creates committed work.
- Reuse the shipped Intake capture/promote path and graph relations rather than
  adding a request/idea work type.
- Preserve mail message and thread identity on promoted artifacts so `hero why`
  can trace the source.
- Emit durable events for read/ack/dismiss/promote outcomes.
- Surface unread counts/items in `hero resume`, `hero status`, and MCP/API
  actions without coupling to Hero Code.
- Prevent accepted mail from being recaptured as a duplicate Intake by
  `intake-capture-loop`.
- Return authoritative promotion results with artifact identity, provenance,
  and a client-safe navigation reference.
- Define `add_to_today` as an idempotent bridge that creates a linked Personal
  Focus item rather than mutating the Mail lifecycle.

## Boundaries

No desktop UI, automatic acceptance, runtime execution, scheduler, or policy
engine. Do not push mail lifecycle states into spec status.

## Acceptance shape

The `/design` pass must define triage transitions, idempotent promotion,
provenance edges, ownership of dismiss/read receipts, stale-action handling,
resume/status/MCP output, and tests across promotion, duplication, missing source
mail, and graph traversal.

## Kickoff

Inspect `hero-idea-primitive-core`, Intake CLI paths, graph/event conventions,
resume/status projections, and MCP write boundaries. Design promotion as a call
into those existing authorities, not a parallel work-creation subsystem.
