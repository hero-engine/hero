---
title: "Harden Permission Bridge Payload Validation"
slug: hihcp-permission-bridge-validation
type: bug
status: handed_off
domain: engineering
size: small
priority: medium
created: 2026-06-09
tags: [hero-code, swift, permissions, crash, validation, p2]
parent: hero-in-hero-code-parity
---

# Harden Permission Bridge Payload Validation

## Issue

Malformed permission payloads sent to the agent loop's permission bridge methods
crash the process instead of producing a denial with a human-readable
explanation. This was observed multiple times during the audit period -- the model
constructs a tool call with unexpected parameters, the permission check receives
a payload it cannot parse, and the app crashes.

Parent initiative: `hero-in-hero-code-parity`.

## Scope -- design inputs for `/design`

- Add defensive validation (guard clauses) to the permission bridge methods in
  `AgentLoop.swift`
- Malformed payloads should be denied with a structured error message explaining
  what was wrong, not crash
- Cover all permission bridge call sites (estimated 1-2 methods)
- Validate payload structure (expected fields present, correct types) before
  processing

**Files to touch:**
- `Engine/AgentLoop.swift` (permission bridge methods, 1-2 call sites)

## Boundaries

- Do not change the permission model itself (what is allowed/denied)
- Do not add new permission types
- Do not change the permission UI -- only the validation of incoming payloads

## Risks

- Overly strict validation could reject legitimate permission requests that have
  optional fields missing. Validate only the required structure.
- Error message content: must be helpful enough for the model to self-correct on
  the next attempt, not just "invalid payload."

## Validation

- Malformed permission requests produce a visible error message, not a crash
- Well-formed permission requests continue to work exactly as before
- The error message includes enough detail for the model to construct a valid
  request on retry

## Handoff Trail

- 2026-06-24T18:01:15Z — out → hero-code (peer_id: cd8dd06d-3df1-4878-a88f-24593dcbb4b3)
  mode: async-drop
  originating_spec: hihcp-permission-bridge-validation
  peer_spec: hero-code/hihcp-permission-bridge-validation
  at_commit: 2f774b7
  reason: "Targets hero-code's Swift permission bridge in Engine/AgentLoop.swift. No Go equivalent in the hero CLI repo."

