---
title: "Deferred Work Suggestion Contract — Explicitly Accepted Focus"
slug: deferred-work-suggestion-contract
type: feature
status: planning
domain: engineering
priority: medium
size: medium
horizon: next
created: 2026-07-20
parent: durable-attention
depends-on: [personal-focus-core]
tags: [focus, harness, suggestions, prompts]
---

# Deferred Work Suggestion Contract — Explicitly Accepted Focus

## Goal

Give models a harness-neutral way to propose meaningful work that is outside the
current loop, while keeping the user as the only authority that turns the
proposal into a durable Focus Item.

## Design inputs

- Suggested shape: kind, title, reason, executable prompt, optional project
  reference, and source conversation/run provenance.
- Consumer actions are `do_next`, `today`, `later`, and `dismiss`.
- `today`/`later` explicitly create Focus; dismissal leaves no durable
  commitment.
- `do_next` defaults to durable acceptance into `today` plus a returned launch
  intent. Hero owns acceptance; the client owns session creation. A failed
  client launch leaves the accepted Focus item in Today for safe retry.
- Unfinished harness checklist steps never leak automatically into Focus.
- Author shared guidance once and propagate self-contained behavior to
  `opencode`, `cursor`, `claude`, `copilot`, `codex`, and `generic`.

## Boundaries

No Claude-only hook as the mechanism, no client-specific chip UI, no silent
auto-capture, no model-managed personal list, and no synchronization with
harness task APIs.

## Acceptance shape

The `/design` pass must define the structured proposal contract, acceptance and
idempotency behavior, conversation provenance, prompt validation, generic
fallback rendering, all-target installation, and tests proving a suggestion is
not durable before acceptance. It must also define structured success,
idempotent replay, stale/unsupported/validation failures, and the authoritative
Focus row returned after acceptance.

## Kickoff

Inspect existing suggested-next actions, intake auto-capture guidance, install
rendering for all six targets, and any structured assistant-output conventions.
Keep suggestion production best-effort but acceptance deterministic.
