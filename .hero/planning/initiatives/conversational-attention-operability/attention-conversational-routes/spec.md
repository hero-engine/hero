---
title: "Attention Conversational Routes — Natural Language to Safe Hero Actions"
slug: attention-conversational-routes
type: feature
status: planning
domain: engineering
priority: high
size: large
horizon: now
created: 2026-07-23
parent: conversational-attention-operability
depends-on:
  - attention-interaction-consent-contract
  - attention-mcp-action-tools
  - attention-lifecycle-read-awareness
  - portable-routing-rules
conflicts-with: [attention-lifecycle-read-awareness]
tags: [attention, routing, natural-language, harnesses, conformance]
relations:
  - target: agent-end-of-turn-recap
    kind: related
  - target: intake-capture-loop
    kind: related
---

# Attention Conversational Routes — Natural Language to Safe Hero Actions

## Goal

Teach every supported harness to map ordinary Attention language to the correct
Mail, Focus, suggestion, promotion, peering, clarification, or read action using
the canonical consent contract and complete MCP surface.

## Kickoff

Adds portable routes for ordinary inbox, Mail, Focus, suggestion, and promotion
phrases after the consent, MCP, lifecycle, and routing substrates are ready.

**Status:** planning — route families and dependencies are identified; portable
routing mechanics and upstream contracts must land first.

**Pick up at:** after dependencies land, design the canonical Attention route
table and shared phrase-conformance corpus in `routing.md`.

→ `/design attention-conversational-routes`

**Files:** `domains/engineering/routing.md`, `domains/engineering/AGENTS.md`,
`internal/install/agents_md.go`, `internal/install/content_test.go`
**Skip:** one-off Claude guidance or duplicating portable routing rendering.

## Scope

The progressive design must add routes for:

| Intent | Example | Canonical operation |
|---|---|---|
| Read Attention | “What needs my attention?” | `hero_attention_snapshot` |
| Read Mail | “What is in my inbox?” | `hero_mail_list`, then explicit show |
| Inspect one message | “Show me that mail” | `hero_mail_show` |
| Send Mail | “Send this to hero-code” | `hero_mail_send` or existing peer route when peer semantics are intended |
| Reply | “Reply with Friday” | `hero_mail_reply` |
| Remember user-authored work | “Remember this for later” | `hero_focus_create` |
| Model-originated optional work | “We should maybe harden this later” | `hero_focus_suggest`, never direct create |
| Accept/dismiss suggestion | “Put that in Today” / “dismiss it” | advertised suggestion action |
| Promote Mail | “Turn that mail into a bug” | advertised Mail/Attention promotion action |
| Ambiguous mutation | “Send that to her” | clarification, no tool mutation |

The routes must:

1. Reuse `portable-routing-rules` as the only distribution mechanism.
2. Preserve existing cross-repo peering distinctions. An advisory question,
   spec-out request, or work transfer uses the peering vocabulary over Project
   Mail rather than being flattened into an untyped send.
3. Choose only advertised row actions for acceptance, dismissal, Today, or
   promotion; models may not manufacture action IDs from status text.
4. Apply the canonical semantic-consent class before dispatch while respecting
   configured harness/client approval.
5. Include exact positive, negative, and ambiguous examples in one conformance
   corpus.
6. Render and verify equivalent imperative guidance in `opencode`, `cursor`,
   `claude`, `copilot`, `codex`, and `generic`.

## Required outcomes

- WHEN a user uses a supported Attention phrase THE SYSTEM SHALL select the
  canonical operation family and preserve the user-authored content.
- WHEN a phrase is an explicit mutation with resolved inputs THE SYSTEM SHALL
  invoke the typed action at most once with stable idempotency.
- IF a phrase is ambiguous or missing required input THEN THE SYSTEM SHALL ask
  only for the missing fact and SHALL NOT dispatch a mutation.
- WHEN the model originates deferred work THE SYSTEM SHALL route to suggestion,
  not direct Focus creation.
- WHEN a user requests promotion THE SYSTEM SHALL invoke only an advertised
  source action and preserve source provenance.
- WHEN the shared phrase corpus is rendered or exercised in any supported
  harness THE SYSTEM SHALL produce the same operation and consent class.

## Boundaries

- No duplicate routing generator or target-specific source of truth.
- No new storage, action, or lifecycle semantics.
- No generic natural-language understanding framework.
- No mailbox-triggered execution or automatic response.
- No changes to generic recap formatting.

## Conflict seam

This child and `attention-lifecycle-read-awareness` both touch canonical agent
guidance and all-target installation fixtures. Their reciprocal
`conflicts-with` edges are a soft mutex even though this child also depends on
the lifecycle child.

## Validation expectations

- Shared conformance cases for every route row, plus ambiguous pronouns,
  unavailable projects, stale rows, retries, and adversarial Mail bodies.
- Installer golden tests proving the same authoritative content lands in all
  six native instruction surfaces.
- MCP inventory checks proving every named tool/action exists.
- Repository search proving no generated instruction references a nonexistent
  command or tool.
