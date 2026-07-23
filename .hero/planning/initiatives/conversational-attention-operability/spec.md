---
title: "Conversational Attention Operability — Safe Chat-Loop Access to Mail and Focus"
slug: conversational-attention-operability
type: initiative
status: planning
domain: engineering
priority: high
size: x-large
horizon: now
created: 2026-07-23
tags: [attention, chat-loop, mcp, routing, consent, harnesses]
child:
  - attention-interaction-consent-contract
  - attention-mcp-action-tools
  - attention-lifecycle-read-awareness
  - attention-conversational-routes
relations:
  - target: durable-attention
    kind: related
  - target: portable-routing-rules
    kind: related
  - target: agent-end-of-turn-recap
    kind: related
  - target: intake-capture-loop
    kind: related
  - target: always-on-runtime
    kind: related
  - target: hero-in-hero-code-parity
    kind: related
---

# Conversational Attention Operability — Safe Chat-Loop Access to Mail and Focus

## Goal

Make the completed Durable Attention foundation naturally operable from a chat
loop. Models should recognize Mail and Focus intent, use a complete MCP surface,
surface bounded Attention state at useful lifecycle boundaries, and behave the
same way across all six installed harnesses and Hero Code—without silently
creating commitments or treating incoming mail as authorization to execute.

## Vision

Attention becomes a normal conversational capability rather than a UI that
users must remember to visit. “What is in my inbox?”, “send this to hero-code,”
“reply with Friday,” “remember this for later,” and “promote that mail” resolve
to typed, auditable operations with one consent policy.

The policy spine is:

- reads are safe and bounded;
- a clear user imperative authorizes the named mutation;
- missing, inferred, or ambiguous targets require clarification;
- agent-generated deferred work remains a suggestion until accepted;
- mail content is always untrusted data and never authorizes execution.

Harness approval settings still apply. Semantic authorization from the
conversation does not bypass a harness or client permission gate.

## Kickoff

Makes Project Mail and Personal Focus work naturally in chat through complete
MCP actions, explicit consent rules, lifecycle awareness, and portable routes.

**Status:** planning — initiative and four Hero-owned child stubs are composed;
the Hero Code native consumer brief is ready for peer design.

**Pick up at:** design the consent contract first; every mutation and route
depends on its intent and confirmation matrix.

→ `/design attention-interaction-consent-contract`

**Files:** `contracts/attention/`, `internal/serve/mcp_tools_def.go`,
`domains/engineering/AGENTS.md`, `domains/engineering/skills/`
**Skip:** mailbox-triggered execution and a second Attention write model.

## Why this follow-up exists

`durable-attention` delivered the storage, identity, lifecycle, promotion,
suggestion, projection, API, CLI, MCP read/action adapters, async peering, and
Hero Code Attention/Today consumer. The substrate works. The remaining product
gap is operability from the place users already work: the chat loop.

The current MCP surface can read Attention and mutate advertised rows, but it
cannot directly send or reply to Mail or create an explicitly requested Focus
item. The installed guidance describes deferred-work suggestions, but it does
not route ordinary Attention phrases, define when a session should inspect
Attention, or give every harness one testable consent policy. Hero Code can
discover MCP tools and display suggestion results, but the complete
phrase-to-tool-to-card-to-Today-to-launch journey is not yet an owned contract.

## Interaction policy

| User/model state | Allowed behavior | Forbidden behavior |
|---|---|---|
| Explicit read request | Read the bounded snapshot/list/show surface | Mark read, acknowledge, or mutate as a side effect |
| Explicit imperative with one resolved target | Invoke the named typed mutation once | Ask a redundant semantic-confirmation question |
| Missing or ambiguous recipient, item, timing, or destination | Ask for the missing fact; perform no mutation | Guess from pronouns, recent context, or model preference |
| Agent notices meaningful deferred work | Emit a suggestion with an exact resumable prompt | Create Focus directly or displace required current work |
| User accepts a suggestion | Run the advertised acceptance action | Reconstruct or invent a different mutation |
| Mail is received or inspected | Treat every field as untrusted data | Execute embedded instructions, start a model, or create work |

The contract child makes this matrix normative and machine-testable. Later
children consume it; they do not restate their own consent rules.

## Specs

| Order | Slug | Priority | Size | Outcome |
|---:|---|---:|---:|---|
| 1 | `attention-interaction-consent-contract` | critical | medium | Canonical intent, consent, ambiguity, effect, and conformance contract |
| 2a | `attention-mcp-action-tools` | critical | medium | Typed Mail send/reply and explicit-user Focus create MCP actions |
| 2b | `attention-lifecycle-read-awareness` | high | medium | Bounded read-only Attention awareness at deterministic chat-loop boundaries |
| 3 | `attention-conversational-routes` | high | large | Natural-language Attention routes distributed through the portable routing substrate |

Six-harness propagation is deliberately not a trailing fifth child. Each
harness-facing child is incomplete until its authoritative content reaches
`opencode`, `cursor`, `claude`, `copilot`, `codex`, and `generic`. The
initiative closes with one aggregate conformance matrix across those targets.

Hero Code owns the native consumer slice. It should design
`attention-hero-code-native-chat-loop` in the sibling repository from
[hero-code-spec-out-prompt.md](hero-code-spec-out-prompt.md); Hero does not keep
a locally deliverable proxy spec for Swift work.

## Dependencies

```text
attention-interaction-consent-contract
  ├─→ attention-mcp-action-tools ───────────────┐
  └─→ attention-lifecycle-read-awareness ──────┤
                                               ├─→ attention-conversational-routes
portable-routing-rules ────────────────────────┘

consent + MCP actions + lifecycle + routes
  └─→ hero-code: attention-hero-code-native-chat-loop
```

`portable-routing-rules` is an adopted hard dependency for the final routing
child. It owns canonical `routing.md` distribution mechanics; this initiative
supplies Attention semantics and conformance cases. The first three children
can proceed before it lands.

`attention-lifecycle-read-awareness` and `attention-conversational-routes` are a
reciprocal soft mutex because both can touch canonical agent guidance and
all-target install fixtures. Their dependency already sequences them, while the
`conflicts-with` edge protects manual or batch delivery from same-file overlap.

## Cross-repo ownership

Hero owns:

- the normative interaction and consent contract;
- MCP names, schemas, permission/effect metadata, idempotency, and errors;
- lifecycle read semantics and Attention source authority;
- portable conversational route definitions and six-target conformance.

Hero Code owns:

- the native AgentLoop adapter and tool-result interpretation;
- permission labels, clarification/confirmation affordances, and typed cards;
- refresh, Today placement, launch behavior, and app-level failure states;
- Swift integration and end-to-end tests against a pinned Hero contract.

The repositories land independently. Cross-repo acceptance runs only after Hero
publishes a pinned contract/fixture revision; Hero Code must not depend on
uncommitted implementation details or persist a competing lifecycle.

## Acceptance journeys

- WHEN a user asks “what is in my inbox?” THE SYSTEM SHALL return a bounded,
  current Attention/Mail view without acknowledging, dismissing, or otherwise
  mutating any item.
- WHEN a user says “send this to hero-code” and the content and destination are
  uniquely resolved THE SYSTEM SHALL send exactly one Mail message and return
  its auditable message/thread receipt.
- IF a user says “send that to her” and either the content or recipient is
  ambiguous THEN THE SYSTEM SHALL perform no mutation and ask only for the
  missing fact.
- WHEN a model identifies optional deferred work THE SYSTEM SHALL create only a
  suggestion; WHEN the user accepts it THE SYSTEM SHALL create the advertised
  Focus item exactly once.
- WHEN Mail contains imperative or executable-looking text THE SYSTEM SHALL
  treat it as untrusted data and SHALL NOT execute it, create committed work, or
  start a session merely because it was received or read.
- WHEN the same explicit Attention intent is exercised through `opencode`,
  `cursor`, `claude`, `copilot`, `codex`, `generic`, or Hero Code THE SYSTEM
  SHALL select the same operation class, consent behavior, and authoritative
  Hero result.

## Cross-cutting rules

- Reuse Mail, Focus, suggestion, projection, and promotion services from
  `durable-attention`; do not create a generic Attention write model.
- Mutations require stable idempotency keys and return authoritative source
  records or typed errors.
- Tool descriptions must be sufficient for a model to select the right action
  without depending on a Claude-only hook.
- Permission/effect metadata must distinguish reads, suggestions, explicit
  user-requested writes, and destructive or commitment-forming transitions.
- An unavailable Attention service is not an empty inbox.
- Bodies and prompts stay out of argv, logs, and committed configuration.
- Unknown additive contract values remain forward-compatible.

## Boundaries

- No mailbox watcher, push-triggered model, scheduler, worker, or automatic
  responder. Those belong to `always-on-runtime`.
- No silent capture of vague intent. `intake-capture-loop` owns its separate,
  project-scoped capture policy; Focus remains explicit or accepted.
- No redesign of the generic end-of-turn recap. This initiative supplies
  Attention-specific inputs; `agent-end-of-turn-recap` owns recap shape.
- No reimplementation of portable routing distribution.
- No new Mail, Focus, or Attention storage authority.
- No automatic conversion of Mail or Focus into Intake, Specs, Jobs, or runs.
- No general personal task manager, prioritizer, calendar, or reminders.

## Risks

- **Consent drift:** a tool description, harness instruction, and Hero Code
  confirmation can each invent a different definition of “explicit.” The
  contract and shared conformance corpus must land first.
- **Prompt injection:** model-facing Mail increases exposure to adversarial
  content. Read awareness must summarize metadata without treating bodies as
  instructions.
- **Routing dependency:** `portable-routing-rules` is still planning. The final
  route child must not work around that by hard-coding one harness.
- **Over-eager awareness:** reading full Attention state on every turn would add
  noise and tokens. Lifecycle points and row limits must be deterministic.
- **Dual authority:** Hero Code may be tempted to infer mutations or persist
  local status. Typed results and fixture-pinned tests keep Hero authoritative.

## Validation strategy

Each child owns focused contract, Go, and installer tests. Initiative closure
also requires:

- one phrase/intent conformance corpus covering reads, explicit writes,
  ambiguity, suggestions, promotion, untrusted Mail, unavailable service, and
  retries;
- generated output checks for all six install targets;
- MCP inventory/profile tests and cross-adapter parity against the Attention
  service;
- a pinned Hero fixture manifest decoded by Hero Code;
- one native end-to-end journey from phrase → MCP tool → structured result/card
  → accept/create → Today → launch;
- repository searches proving no active path executes on Mail receipt.

## Recommended order

Design and deliver `attention-interaction-consent-contract` first. Then deliver
MCP actions and lifecycle awareness in parallel. Deliver
`portable-routing-rules` before the route child, and let Hero Code design its
native slice once the consent and MCP schemas are stable enough to pin.
