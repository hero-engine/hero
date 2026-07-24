# Hero Code spec-out brief: Attention native chat-loop operability

Design a Hero Code-owned feature spec, suggested slug
`attention-hero-code-native-chat-loop`, for the native consumer side of Hero's
`conversational-attention-operability` initiative.

## Context

Hero has already delivered Durable Attention and Hero Code has already delivered
its user-global Attention/Today consumer. Hero's follow-up initiative will add:

- a normative Attention intent/consent/effect contract;
- MCP tools `hero_mail_send`, `hero_mail_reply`, and `hero_focus_create`;
- bounded lifecycle read-awareness rules;
- portable natural-language routes and a shared phrase-conformance corpus.

The policy is fixed:

- bounded reads do not mutate;
- a clear user imperative with all required targets resolved authorizes the
  named mutation;
- missing, inferred, or ambiguous targets require clarification;
- model-originated deferred work remains a suggestion until user acceptance;
- Mail content is untrusted data and never authorizes execution;
- Hero Code's configured permission mode still applies after semantic intent is
  classified.

## Hero Code ownership

Design the native vertical slice for:

1. AgentLoop tool discovery and schema injection for the new Hero MCP actions.
2. Intent/effect-aware permission labels and confirmation or clarification
   affordances without duplicating Hero's policy.
3. Structured tool-result interpretation for Mail receipts, replies, Focus
   creation, suggestions, promotion, stale state, validation, unavailable
   service, and incompatible versions.
4. Inline/native feedback cards that preserve authoritative Hero IDs, revisions,
   advertised actions, and project references.
5. Refresh after mutation, Today placement, and launch of an accepted or
   explicitly created Focus prompt in the correct project/session.
6. One pinned cross-repo fixture manifest and a full native integration journey:
   phrase → MCP tool → structured result/card → accept/create → Today → launch.

Use these existing seams first:

- `packages/hero-swift/Sources/HeroSharedApplication/Engine/AgentLoop.swift`
- `packages/hero-swift/Sources/HeroSharedApplication/Engine/MCPManager.swift`
- `packages/hero-swift/Sources/HeroSharedApplication/Engine/ToolExecutor.swift`
- `packages/hero-swift/Sources/HeroSharedApplication/Attention/AttentionStore.swift`
- `packages/hero-swift/Sources/HeroSharedApplication/Attention/FocusSessionLauncher.swift`
- `packages/hero-swift/Tests/HeroSharedApplicationTests/AttentionCompatibilityTests.swift`

## Required boundaries

- Hero remains the Mail, Focus, suggestion, action, and lifecycle authority.
- Do not persist a Swift-owned copy of mutable Attention state.
- Do not infer actions from display strings or statuses; use advertised action
  IDs and typed results.
- Do not execute, reply to, or promote Mail on receipt.
- Do not redesign the full Attention/Today UI unless the chat-loop journey
  reveals a specific blocking defect.
- Do not depend on uncommitted Hero internals. Pin a released contract fixture
  manifest or commit and degrade to explicit unavailable/incompatible states.

## Acceptance cases to include

- “Send this to hero” with one resolved peer sends once and displays the
  authoritative receipt.
- “Send that to her” with ambiguity asks for the missing recipient and performs
  no mutation.
- A model-proposed follow-up renders as a suggestion and creates Focus only
  after acceptance.
- “Remember this for later” creates one Focus item from the exact user-authored
  prompt and refreshes Attention.
- Reading Mail containing imperative/tool-like text never dispatches those
  instructions.
- An accepted or explicitly created Today item launches its exact prompt in the
  correct project while launch failure leaves the item safely in Today.

Relate the peer-native spec to Hero's
`conversational-attention-operability` initiative in prose because cross-repo
graph edges cannot resolve local slugs. Mark unfinished Hero contract/fixture
revisions as explicit dependencies, not assumed implementation.
