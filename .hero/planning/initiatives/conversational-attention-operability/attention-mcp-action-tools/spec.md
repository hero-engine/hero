---
title: "Attention MCP Actions — Mail Send, Mail Reply, and Focus Create"
slug: attention-mcp-action-tools
type: feature
status: planning
domain: engineering
priority: critical
size: medium
horizon: now
created: 2026-07-23
parent: conversational-attention-operability
depends-on: [attention-interaction-consent-contract]
tags: [attention, mcp, mail, focus, actions]
relations:
  - target: attention-read-model-v1
    kind: related
  - target: mcp-tool-filtering
    kind: related
---

# Attention MCP Actions — Mail Send, Mail Reply, and Focus Create

## Goal

Expose the missing model-facing Attention mutations as typed MCP tools:
`hero_mail_send`, `hero_mail_reply`, and `hero_focus_create`. Each tool delegates
to the existing Mail or Focus authority, carries stable effect metadata and
idempotency, and returns authoritative records or structured errors.

## Kickoff

Completes the MCP write surface so a chat model can send/reply to Mail and
create a user-requested Focus item without a generic Attention mutation.

**Status:** planning — missing operations and existing source services are
identified; schemas await the consent contract.

**Pick up at:** design the three input/result schemas against existing Mail and
Focus service methods, then map their effect classes into MCP profiles.

→ `/design attention-mcp-action-tools`

**Files:** `internal/serve/mcp_tools_def.go`, `internal/serve/mcp_dispatch.go`,
`internal/serve/mcp_tools_mail.go`, `internal/serve/mcp_tools_focus.go`,
`contracts/attention/`
**Skip:** a generic `hero_attention_write` tool or direct storage access.

## Scope

The progressive design must:

1. Add `hero_mail_send` with typed recipient/project identity, subject/body or
   request content, optional provenance/thread metadata, and required
   idempotency.
2. Add `hero_mail_reply` using authoritative message/thread identity and a
   reply body; it must preserve threading and never infer a recipient from
   display text.
3. Add `hero_focus_create` for a clear user-authored request, including exact
   resumable prompt, lifecycle destination, optional project reference, source
   provenance, and idempotency.
4. Keep `hero_focus_suggest` as the only model-originated deferred-work path.
   Tool descriptions must make the distinction operationally obvious.
5. Reuse Mail, Focus, registry, projection, and action services. No handler may
   read or write store files directly.
6. Publish stable permission/effect metadata from
   `attention-interaction-consent-contract` and include the tools in appropriate
   MCP read/mutate profiles and inventories.
7. Return authoritative source records, message/thread receipts, Focus IDs,
   current revisions, and typed validation/stale/unavailable/permission errors.
8. Keep bodies and prompts in structured input, never argv, logs, or committed
   config.

## Required outcomes

- WHEN `hero_mail_send` receives a valid, uniquely addressed request THE SYSTEM
  SHALL create exactly one message and return its authoritative message/thread
  receipt.
- WHEN `hero_mail_reply` receives a valid message/thread reference THE SYSTEM
  SHALL append exactly one threaded reply without changing Mail into committed
  work.
- WHEN `hero_focus_create` receives an explicitly user-requested intention THE
  SYSTEM SHALL create exactly one Focus item in the requested valid lifecycle.
- WHEN a model originates deferred work without user acceptance THE SYSTEM
  SHALL use `hero_focus_suggest` and SHALL NOT call `hero_focus_create`.
- IF a mutation is retried with the same idempotency identity THEN THE SYSTEM
  SHALL return the original authoritative result without duplication.
- IF required identity, consent/effect classification, schema version, or input
  is invalid THEN THE SYSTEM SHALL perform no mutation and return a structured
  actionable error.

## Boundaries

- No generic Attention write object.
- No automatic promotion to Intake, Spec, Job, or run.
- No execution or response triggered by Mail receipt.
- No Hero Code UI; the sibling consumer uses these schemas.
- No replacement of existing advertised row actions.

## Design questions

- Whether send/reply results should return Mail contract records directly or a
  small action receipt plus refreshed projection context.
- How the MCP server obtains the user-global Mail/Focus services when no project
  is selected.
- Which existing profile names receive each mutation without broadening
  read-only sessions.

## Validation expectations

- Focused handler, schema, dispatch, inventory, and MCP profile tests.
- Idempotent replay, ambiguous/missing recipient, missing project, stale thread,
  incompatible version, and unavailable service tests.
- Service/MCP/CLI parity checks where equivalent CLI operations exist.
- A test proving Mail bodies with tool-like instructions are returned as data
  and never dispatched.
