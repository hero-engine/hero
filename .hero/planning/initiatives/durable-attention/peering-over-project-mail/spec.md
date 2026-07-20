---
title: "Peering over Project Mail — Async First, Execution Optional"
slug: peering-over-project-mail
type: feature
status: planning
domain: engineering
priority: medium
size: medium
horizon: next
created: 2026-07-20
parent: durable-attention
depends-on: [project-mail-triage-and-provenance]
tags: [peering, mail, handoff, compatibility]
---

# Peering over Project Mail — Async First, Execution Optional

## Goal

Reframe cross-project peering over durable Project Mail so communication works
without Hero spawning a harness, and a work transfer occurs only when the
receiver accepts and promotes a request.

## Design inputs

- Preserve peer identity, alias/path resolution, manifests, provenance, and
  compatible user-facing entry points.
- `hero handoff` becomes request delivery plus explicit receiver acceptance,
  not immediate receiver-side spec creation.
- A synchronous `hero peer call` may remain as an optional send/wait adapter
  when a harness or runtime is available.
- Remove Claude-specific subprocess execution and result-fence parsing from the
  architectural center.
- Define compatibility and deprecation behavior for existing handoff trails and
  statuses.
- When this design is materialized, supersede `peer-call-multi-cli` through the
  supported `hero supersede` command.

## Boundaries

No implementation of every harness CLI profile, no always-on worker, no cloud
transport, no full-delivery peer call, and no changes to NEXT/session handoff.

## Acceptance shape

The `/design` pass must trace current peer-call and handoff flows, define
compatibility commands and status translation, receiver acceptance, failure and
retry behavior, legacy trail reading, and tests proving peering works with no
model CLI installed.

## Kickoff

Inspect `internal/peering`, peering contracts, CLI wiring, and reconciliation.
Separate three overloaded concepts: session handoff, project communication, and
accepted work transfer. Retain the useful locator/provenance substrate.
