---
title: "Personal Focus Core — Prompt-Backed Work Across Projects"
slug: personal-focus-core
type: feature
status: planning
domain: engineering
priority: high
size: medium
horizon: next
created: 2026-07-20
parent: durable-attention
depends-on: [durable-attention-contracts]
tags: [focus, personal, cross-project, prompts]
---

# Personal Focus Core — Prompt-Backed Work Across Projects

## Goal

Provide a user-global, durable list of prompt-backed intentions that can be
captured manually, linked to projects, moved through
`inbox/today/later/done`, and launched later without becoming a project manager
or synchronizing harness task lists.

## Design inputs

- Focus belongs to a user, not a repository or agent run.
- Minimum record: stable ID, title, executable prompt, optional project
  reference, origin/provenance, lifecycle state, and timestamps.
- Manual creation is immediately durable; model suggestions require explicit
  acceptance before persistence.
- Provide CLI and consumer-safe API operations independent of Hero Code.
- Completion assistance may be proposed, but the durable list remains
  user-authoritative.
- Publish the saved executable prompt through an advertised launch capability
  or authoritative launch result without exposing store internals.
- Treat a source/session correlation as optional typed launch context, not a
  spec slug and not an inferred link to a harness checklist.

## Boundaries

No due dates, estimates, priority numbers, assignments, subtasks, calendars,
notifications, harness todo synchronization, or automatic import from specs and
trackers.

## Acceptance shape

The `/design` pass must choose global storage and user identity behavior, define
CRUD and state-transition semantics, project resolution and missing-project
behavior, prompt safety, JSON output, and tests for cross-project use and
concurrent updates.

The design must also define idempotent creation of a Focus item from a Mail
source so `add_to_today` cannot produce duplicates while leaving Mail ownership
and lifecycle unchanged.

## Kickoff

Inspect existing user-global config/state conventions and project registry
before choosing storage. Preserve the product rule: models may help a user
remember commitments but may not create commitments on the user's behalf.
