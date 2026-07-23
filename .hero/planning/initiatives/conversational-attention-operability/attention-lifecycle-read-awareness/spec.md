---
title: "Attention Lifecycle Read Awareness — Bounded Context at Chat Boundaries"
slug: attention-lifecycle-read-awareness
type: feature
status: planning
domain: engineering
priority: high
size: medium
horizon: now
created: 2026-07-23
parent: conversational-attention-operability
depends-on: [attention-interaction-consent-contract]
conflicts-with: [attention-conversational-routes]
tags: [attention, lifecycle, resume, recap, context]
relations:
  - target: agent-end-of-turn-recap
    kind: related
  - target: active-context-management
    kind: related
---

# Attention Lifecycle Read Awareness — Bounded Context at Chat Boundaries

## Goal

Give chat loops deterministic, bounded, read-only awareness of Attention at
useful lifecycle points without loading full Mail bodies, marking items read,
creating commitments, or turning every model turn into an inbox poll.

## Kickoff

Defines when a Hero-aware chat inspects Attention and how it summarizes that
state without mutating or flooding context.

**Status:** planning — the existing snapshot tool is sufficient; lifecycle
points, limits, unavailable behavior, and all-target guidance need design.

**Pick up at:** choose the smallest deterministic boundary set and bounded
summary shape, anchored on the existing `hero_attention_snapshot` contract.

→ `/design attention-lifecycle-read-awareness`

**Files:** `domains/engineering/commands/resume.md`,
`domains/engineering/skills/`, `internal/install/content_test.go`,
`internal/serve/mcp_tools_attention.go`
**Skip:** per-turn polling, full Mail bodies, or acknowledgment on inspection.

## Scope

The progressive design must:

1. Define a small boundary set, expected to include:
   - fresh/resumed Hero-aware session;
   - successful Attention mutation;
   - end-of-turn recap only for Attention state changed or made relevant during
     that turn.
2. Use `hero_attention_snapshot` or its shared projection service as the read
   authority. Do not introduce a second digest or storage query.
3. Bound row counts and fields to actionable metadata. Full Mail body inspection
   remains an explicit `hero_mail_show` action.
4. Keep snapshot inspection side-effect free: no mark-read, acknowledge,
   dismissal, acceptance, promotion, or lifecycle mutation.
5. Distinguish unavailable, stale, and empty states so an offline service cannot
   masquerade as “nothing needs attention.”
6. Compose with `agent-end-of-turn-recap`: this child provides
   Attention-specific facts and trigger rules; that spec owns the generic recap
   structure.
7. Author guidance once and prove propagation across `opencode`, `cursor`,
   `claude`, `copilot`, `codex`, and `generic`.

## Required outcomes

- WHEN a Hero-aware session starts or resumes with Attention available THE
  SYSTEM SHALL expose one bounded read-only summary before the model claims
  there is nothing pending.
- WHEN Attention is unavailable THE SYSTEM SHALL say it is unavailable and
  SHALL NOT represent the state as empty.
- WHEN a mutation succeeds THE SYSTEM SHALL converge on authoritative current
  state without blindly replaying the write.
- WHEN no Attention item changed or became relevant during a turn THE SYSTEM
  SHALL NOT append a generic inbox dump to the end-of-turn recap.
- WHEN snapshot awareness runs THE SYSTEM SHALL NOT read full Mail bodies or
  mutate any source lifecycle.
- WHEN installed into any supported harness THE SYSTEM SHALL communicate the
  same boundaries and side-effect rules without relying on hooks.

## Boundaries

- No background watcher, timer, push notification, or mailbox-triggered model.
- No generic recap format changes.
- No ranking algorithm beyond the existing Attention projection.
- No automatic Mail show/read/acknowledge.
- No requirement that Attention be available for unrelated Hero workflows.

## Design questions

- Whether fresh-session awareness belongs directly in `/resume`, in a reusable
  lifecycle skill, or in both through one generated source.
- The exact row/count budget that remains useful without crowding session
  context.
- Whether post-mutation convergence consumes the tool result alone or performs
  one explicit snapshot refresh.

## Validation expectations

- Boundary tests for fresh, resumed, post-mutation, relevant end-turn, and
  irrelevant end-turn cases.
- Side-effect assertions around every awareness read.
- Empty/unavailable/stale distinction tests.
- Golden installed guidance checks across all six harnesses.
