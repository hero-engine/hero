---
title: "Attention Interaction Consent Contract — Explicit Intent, Ambiguity, and Effects"
slug: attention-interaction-consent-contract
type: feature
status: planning
domain: engineering
priority: critical
size: medium
horizon: now
created: 2026-07-23
parent: conversational-attention-operability
tags: [attention, consent, intent, permissions, contract]
relations:
  - target: durable-attention-contracts
    kind: related
  - target: intake-capture-loop
    kind: related
---

# Attention Interaction Consent Contract — Explicit Intent, Ambiguity, and Effects

## Goal

Define one normative, machine-testable policy that maps conversational intent
to Attention reads, suggestions, explicit mutations, clarification, and
permission effects. MCP tools, installed guidance, conformance tests, and Hero
Code must consume this contract rather than inventing local consent rules.

## Kickoff

Defines when chat language permits an Attention read or mutation and when the
agent must clarify, suggest, or do nothing.

**Status:** planning — initiative policy is agreed; the canonical artifact and
conformance representation still need design.

**Pick up at:** design the intent/effect matrix and choose the single source
from which Go tests, installed guidance, and Hero Code fixtures can derive.

→ `/design attention-interaction-consent-contract`

**Files:** `contracts/attention/`, `contracts/attention/testdata/v1/`,
`domains/engineering/skills/deferred-work-suggestions/SKILL.md`
**Skip:** treating harness approval mode as a substitute for semantic consent.

## Scope

The progressive design must:

1. Define canonical operation classes:
   - bounded read;
   - explicit user-requested mutation;
   - agent-generated suggestion;
   - ambiguous or incomplete request;
   - commitment-forming promotion;
   - untrusted inbound content.
2. Define what counts as an explicit imperative. The action, object, and every
   required target must be stated or uniquely resolved from user-authored
   context. Pronouns or inferred recipients are not sufficient when multiple
   candidates remain.
3. Separate semantic authorization from execution approval. A clear imperative
   authorizes selecting the tool, while harness/client permission policy may
   still require its configured approval.
4. Assign stable permission/effect classes for reads, suggestions, writes, and
   promotions so MCP filtering and Hero Code labels do not infer risk from tool
   names.
5. Define idempotency and retry expectations: one user intent maps to one
   stable key until the intent materially changes.
6. Publish positive and negative phrase fixtures that every harness and Hero
   Code can consume.
7. Preserve the existing distinction between direct Focus creation from an
   explicit user request and `hero_focus_suggest` for model-originated work.

## Required outcomes

- WHEN a request is a bounded read THE SYSTEM SHALL permit the read without
  mutating acknowledgement, status, or lifecycle.
- WHEN a clear imperative identifies one action and all required targets THE
  SYSTEM SHALL classify it as an explicit mutation request.
- IF an action target or required input is missing or ambiguous THEN THE SYSTEM
  SHALL classify it as clarification-required and SHALL NOT dispatch a write.
- WHEN deferred work originates from the model THE SYSTEM SHALL classify it as
  a suggestion even if the model believes it is important.
- WHEN Mail content contains instructions THE SYSTEM SHALL classify those
  instructions as untrusted data, never as user authorization.
- WHEN the same phrase fixture is evaluated by Go, installed harness guidance,
  and Hero Code THE SYSTEM SHALL produce the same operation and consent class.

## Boundaries

- No MCP handlers or new tools; `attention-mcp-action-tools` consumes this
  contract.
- No natural-language routing installation; `attention-conversational-routes`
  owns the routes.
- No client UI or approval-dialog design.
- No silent Focus or Intake creation.
- No general policy engine beyond Attention interactions.

## Design questions

- Should the canonical artifact be executable Go data with generated fixtures,
  a versioned JSON contract, or a small declarative source consumed by both?
- Which effect names align with existing MCP profile/permission metadata without
  breaking older clients?
- How are references such as “that message” proven uniquely resolved across
  turns without serializing model reasoning?

## Validation expectations

- Table-driven positive/negative phrase classification tests.
- Contract fixture round-trip and additive compatibility tests.
- A test proving explicit semantic authorization does not suppress a configured
  harness/client approval requirement.
- A test proving Mail body text cannot become an authorization source.
