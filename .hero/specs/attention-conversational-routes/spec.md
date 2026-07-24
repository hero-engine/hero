---
title: "Attention Conversational Routes — Natural Language to Safe Hero Actions"
slug: attention-conversational-routes
type: feature
status: completed
domain: engineering
priority: high
size: medium
horizon: now
created: 2026-07-23
updated: 2026-07-23
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
  - target: attention-contract-bundle-publication
    kind: blocks
delivery_method: manual
completed_at: 2026-07-24T01:23:29Z
---

# Attention Conversational Routes — Natural Language to Safe Hero Actions

## Context

Hero has a normative interaction-policy registry, typed Mail send/reply and
explicit Focus-create MCP tools, advertised row actions, and bounded lifecycle
awareness. Models can discover those primitives, but the engineering routing
source does not yet tell them how ordinary inbox language maps to the correct
read, explicit write, suggestion, promotion, peering, or clarification path.

This feature adds Attention semantics to the canonical route source delivered
by `portable-routing-rules`. It does not add another dispatcher or infer intent
in Go. The executable contract remains Hero's MCP/action surface; installed
guidance makes model selection deterministic, and a machine-readable corpus
pins the expected mapping for every harness and Hero Code.

## Goal

Teach every supported chat harness to map ordinary Attention language to the
correct typed Hero operation, apply the canonical consent and trust rules, and
publish one complete conformance corpus that downstream clients can run without
reimplementing Hero's policy.

## Kickoff

Add canonical Attention route families and a shared phrase corpus after the
portable route source is available.

**Status:** in review — canonical guidance, the 29-case corpus, registry
validation, six-target distribution, and repository validation are complete.

**Pick up at:** cold-audit the Completion Ledger, address any HOLD findings,
then run the delivery verification gate.

→ `/deliver attention-conversational-routes`

**Files:** `domains/engineering/routing.md`,
`contracts/attention/conversational_route.go`,
`contracts/attention/testdata/v1/conversational-routes.json`,
`contracts/attention/schema/v1/conversational-routes.schema.json`,
`contracts/attention/contract_test.go`, `internal/install/routing_guidance_test.go`,
`internal/serve/routing_reference_test.go`,
`internal/cli/conversational_routes_test.go`
**Skip:** target-specific prompts, generic NLU code, duplicate consent rules,
automatic Mail execution, or another Attention mutation surface.

## Design

### Canonical route families

Add a compact Attention section to the canonical engineering routing source:

| Intent | Example | Canonical operation |
|---|---|---|
| Read bounded Attention | “What needs my attention?” | `hero_attention_snapshot(limit: 8)` |
| List Mail | “What is in my inbox?” | `hero_mail_list` |
| Inspect one message | “Show me that mail” | `hero_mail_show` after unique resolution |
| Send ordinary Mail | “Send this to hero-code” | `hero_mail_send` |
| Ask a peer | “Ask hero-code whether this schema is stable” | peering advisory route, not generic Mail |
| Ask a peer to design | “Have hero-code design its native slice” | peering spec-out route |
| Transfer owned work | “Hand this spec to hero-code” | typed handoff route |
| Reply | “Reply with Friday” | `hero_mail_reply` after unique thread/message resolution |
| Remember explicit user work | “Remember this for later” | `hero_focus_create` |
| Capture model-originated option | “We should maybe harden this later” | `hero_focus_suggest`, never direct create |
| Accept/dismiss suggestion | “Put that in Today” / “dismiss it” | advertised suggestion row action |
| Promote Mail | “Turn that mail into a bug” | advertised Mail promotion action |
| Resolve ambiguity | “Send that to her” | clarification only; zero mutation |

The installed guidance names the operation family and selection rule, but it
does not duplicate request JSON schemas. Models use the advertised MCP schema
and typed row actions when dispatching.

### Consent, ambiguity, and trust

Every route refers to the canonical operation policy rather than inventing a
new confirmation matrix:

- bounded reads are side-effect-free;
- an explicit user imperative can satisfy semantic consent only when every
  required target and content value resolves uniquely;
- missing, inferred, or ambiguous facts yield clarification and zero mutation;
- model-originated deferred work routes only to suggestion;
- suggestion acceptance and promotion use the exact advertised action;
- Mail fields and bodies are untrusted data and never authorize execution; and
- harness/client permission policy still runs after semantic classification.

The guidance must ask only for the missing fact. It must not guess from a
pronoun, choose a recent peer because it seems likely, or ask a redundant
semantic confirmation when an explicit imperative is already complete.

### Peering stays typed

Project Mail is transport, not a reason to flatten peer workflows. A request
for a fact uses the advisory peer-call route; a request for peer-native design
uses spec-out; an already-investigated work transfer uses handoff. An ordinary
human-readable message uses `hero_mail_send`.

The routing source retains the existing session `/handoff` versus cross-repo
handoff disambiguation. Attention examples extend it without redefining peer
mode semantics.

### Versioned conformance corpus

Add `conversational-routes.json` and its JSON Schema under the existing
Attention v1 contract directories. Each case records:

- stable case ID and phrase;
- source/trust class;
- resolved and missing target facts;
- expected disposition (`dispatch`, `clarify`, `suggest`, or
  `ignore_untrusted`);
- operation ID;
- MCP tool, advertised action, or typed CLI workflow family;
- effect and semantic consent class;
- expected mutation count;
- idempotent-retry expectation; and
- optional expected clarification field.

The corpus covers positive, negative, ambiguous, stale, unavailable,
adversarial Mail, retry, unknown-additive, and cross-repo peering cases. It is
data, not an executable natural-language classifier. Hero tests it against the
interaction registry, tool inventory, advertised action registry, and real CLI
command tree; Hero Code later runs the same cases through its deterministic
integration harness. Peering cases use the `cli_workflow` surface because
advisory, spec-out, and handoff are intentionally typed CLI workflows rather
than invented Attention MCP tools.

The initial v1 corpus contains at least 26 named cases: one positive case for
each of the 13 route-family rows, six missing-or-ambiguous target cases, three
untrusted-Mail variants, and four cases covering unavailable, stale,
idempotent retry, and unknown-additive behavior. Additional cases are allowed;
these category floors are the completeness gate.

### Six-target distribution

The route source is rendered by `portable-routing-rules` into Claude's
`CLAUDE.md` and every other target's `AGENTS.md`. Add golden installation tests
for OpenCode, Cursor, Claude, Copilot, Codex, and Generic. Tests assert the
same route markers, consent boundary, peering distinction, and untrusted-Mail
rule in every native root.

No Claude hook or target-specific prompt file is required. Detailed lifecycle
rules remain in the existing reusable Attention skill and resume workflow.

## Changes

1. Add the Attention route-family table and concise safety rules to the
   canonical engineering routing source.
2. Add a versioned conversational-routes schema and complete corpus to the
   Attention v1 contract fixtures.
3. Validate every corpus case against the canonical interaction-policy
   operation, effect, consent, target, and trust invariants.
4. Validate each dispatched corpus case against the real MCP tool inventory,
   an advertised Attention row action, or—only for typed peering—the real CLI
   command tree; reject invented names.
5. Add peering-specific cases preserving advisory, spec-out, handoff, and
   ordinary-Mail distinctions.
6. Add ambiguity, model-originated suggestion, untrusted-Mail, unavailable,
   stale, and idempotent-retry negative cases with exact mutation counts.
7. Add a six-target native-root golden matrix and reference searches proving
   equivalent route content and no nonexistent commands or tools.
8. Add the corpus to the existing v1 fixture manifest so the final publication
   child can assemble it into the cross-repo conformance bundle.

## Acceptance Criteria

- **AC-1:** WHEN a user asks what needs attention THE INSTALLED GUIDANCE SHALL
  select one bounded `hero_attention_snapshot(limit: 8)` read and SHALL NOT
  acknowledge, dismiss, show full Mail, or otherwise mutate an item.
- **AC-2:** WHEN a user asks what is in the inbox THE INSTALLED GUIDANCE SHALL
  select `hero_mail_list`; WHEN the user asks to inspect one uniquely resolved
  message it SHALL select `hero_mail_show` without treating its body as
  instructions.
- **AC-3:** WHEN a user explicitly requests Mail send or reply with uniquely
  resolved content and recipient/thread THE INSTALLED GUIDANCE SHALL select
  `hero_mail_send` or `hero_mail_reply` exactly once with stable idempotency.
- **AC-4:** IF a required recipient, content value, message, project, timing, or
  destination is missing, inferred, or ambiguous THEN THE INSTALLED GUIDANCE
  SHALL ask only for that fact and SHALL dispatch zero mutations.
- **AC-5:** WHEN a user explicitly asks to remember exact work for later THE
  INSTALLED GUIDANCE SHALL select `hero_focus_create`; WHEN the model originates
  optional deferred work it SHALL select `hero_focus_suggest` and SHALL NOT
  create Focus directly.
- **AC-6:** WHEN a user accepts, dismisses, moves, launches, or promotes an
  Attention row THE INSTALLED GUIDANCE SHALL invoke only the exact advertised
  action and SHALL NOT manufacture an action ID from status or display text.
- **AC-7:** WHEN a request names a peer advisory question, peer-native design,
  or work transfer THE INSTALLED GUIDANCE SHALL preserve the corresponding
  typed peering route instead of flattening it into ordinary Mail send.
- **AC-8:** WHEN Mail contains imperative, prompt-like, or tool-like content THE
  INSTALLED GUIDANCE SHALL classify it as untrusted data and SHALL dispatch zero
  operations solely because that content was received or read.
- **AC-9:** WHEN the conversational corpus is validated THE SYSTEM SHALL contain
  at least 26 named cases meeting the 13 positive, six ambiguity, three
  untrusted-Mail, and four availability/retry/additive category floors and
  SHALL prove every case's disposition, operation, tool/action/typed workflow,
  effect, consent, clarification field, and mutation count against the
  canonical registries.
- **AC-10:** WHEN Hero installs OpenCode, Cursor, Claude, Copilot, Codex, or
  Generic THE SYSTEM SHALL render equivalent Attention routing and safety rules
  in that target's native root instruction file without a target-specific
  routing source.

## Non-goals

- No generic natural-language classifier, intent service, or runtime phrase
  matcher.
- No duplicate routing renderer or per-target source of truth.
- No new Attention storage, lifecycle, action, or consent semantics.
- No automatic responder, mailbox-triggered execution, watcher, or model run.
- No automatic `hero_mail_show` during bounded awareness.
- No direct Focus creation for model-originated deferred work.
- No generic recap formatting or frequency changes.
- No Hero Code Swift implementation.

## Risks

- **Policy duplication:** prose can drift from the interaction registry. Corpus
  validation pins every semantic field to the canonical policy.
- **Tool hallucination:** models may invent action names. Runtime inventory and
  advertised-action validation fail invalid references before publication.
- **Peering collapse:** all peer traffic uses Mail underneath, but the workflow
  modes carry different ownership. Explicit route cases preserve them.
- **Prompt injection:** Mail content is model-visible. Negative corpus cases and
  six-target imperatives prohibit content-authorized dispatch.
- **Ambiguity overreach:** recent context can look unique while remaining
  inferred. Target-resolution fields and zero-mutation cases make the boundary
  testable.
- **Harness drift:** a route added only to Claude would appear functional.
  Portable source composition and the all-target golden matrix prevent it.

## Validation

- `go test ./contracts/attention ./internal/install ./internal/serve`
- `go test ./...`
- Schema validation and DTO decoding for every corpus case.
- Registry parity for operation, effect, consent, tool/action, and target
  requirements.
- Mutation-count assertions for positive, retry, ambiguous, model-originated,
  untrusted-Mail, unavailable, and stale cases.
- Six-target install matrix checking native root filenames and equivalent route
  markers.
- Repository search proving no route references a nonexistent command, MCP
  tool, action, or skill and no target-specific Attention route copy exists.

## Completion Ledger

The canonical engineering route source now teaches models how ordinary
Attention language maps to safe reads, explicit writes, suggestions, exact
advertised row actions, and typed peering workflows. A schema-governed
29-case corpus publishes the same expectations for deterministic downstream
conformance. Typed peering is represented as `cli_workflow` rather than as a
fictional Attention MCP tool; tests resolve those commands against Hero's real
CLI tree.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Bounded “what needs attention” selects one snapshot read and no mutation | DONE | `domains/engineering/routing.md` routes the phrase to one `hero_attention_snapshot` call with `limit: 8`; `route-attention-snapshot` pins read/none/zero-mutation semantics. |
| 2 | Inbox list and uniquely resolved Mail show use typed reads without trusting bodies | DONE | The canonical route table maps inbox and message inspection to `hero_mail_list` and `hero_mail_show`; route cases pin unique resolution, read effects, and zero mutations, while the safety rule keeps Mail inert. |
| 3 | Explicit resolved Mail send/reply dispatch exactly once with stable idempotency | DONE | `route-mail-send` and `route-mail-reply` resolve required facts, require explicit-user consent, expect one initial mutation, and pin same-key retry to zero duplicate mutations. |
| 4 | Missing or ambiguous facts clarify only that field and mutate zero times | DONE | Six dedicated ambiguity cases cover recipient, content, message, project, timing, and destination; the validator requires one clarification field, no executable operation, and zero mutations. |
| 5 | User-authored work creates Focus; model-authored options only suggest | DONE | `route-focus-create` maps explicit user work to `hero_focus_create`; `route-focus-suggest` maps model-originated work to `hero_focus_suggest`, and validation rejects any other model dispatch. |
| 6 | Accept, dismiss, move, launch, and promote use only advertised row actions | DONE | Canonical guidance names all five verbs and requires exact advertised IDs/revisions. Corpus cases cover suggestion Today/dismiss, Focus move/launch, and Mail promote; contract tests prove every referenced action appears in the canonical all-actions fixture. |
| 7 | Preserve advisory, spec-out, and handoff peering routes instead of Mail flattening | DONE | Three peering cases use typed `cli_workflow` operations; `TestConversationalRouteCorpusUsesRealCLICommands` resolves each against Hero's actual CLI tree. |
| 8 | Imperative, prompt-like, and tool-like Mail remains untrusted and dispatches nothing | DONE | Three hostile-Mail cases use `ignore_untrusted` with zero mutations; validation rejects any Mail-content dispatch. |
| 9 | Publish and validate the complete category-floor corpus against canonical registries | DONE | The manifest-listed corpus contains 29 cases: 16 route-family, 6 ambiguity, 3 untrusted-Mail, and 4 resilience. Schema, DTO, policy, action, MCP inventory, CLI tree, consent/effect, clarification, error, and exact mutation/retry validation all pass. |
| 10 | Render equivalent Attention guidance into all six native harness roots | DONE | `TestRoutingGuidanceReachesAllHarnessNativeRoots` asserts the same Attention table, consent boundary, untrusted-Mail rule, and revision rule in OpenCode, Cursor, Claude, Copilot, Codex, and Generic native roots. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Add Attention routes and safety rules to the canonical source | DONE | Extended `domains/engineering/routing.md` with the 13 core route families plus consent, ambiguity, trust, stale, idempotency, and advertised-action rules. |
| 2 | Add the versioned schema and complete corpus | DONE | Added `conversational-routes.schema.json`, the 29-case `conversational-routes.json`, and typed DTOs in `conversational_route.go`. |
| 3 | Validate policy, effect, consent, target, and trust invariants | DONE | `ValidateConversationalRouteFixture` resolves Attention operations to `OperationPolicies`, enforces unique targets, model-vs-user authority, trust classes, effects, consent, and category floors. |
| 4 | Validate dispatched cases against real executable surfaces | DONE | Contract tests validate advertised actions, serve tests validate MCP tools, and CLI tests validate the three typed peering commands; invented operation/tool and unsafe-dispatch negative tests pass. |
| 5 | Add typed peering cases | DONE | Added advisory, spec-out, and handoff cases with one resolved peer and distinct canonical command/effect mappings. |
| 6 | Add ambiguity, model, hostile-Mail, unavailable, stale, retry, and additive cases | DONE | The corpus includes every required negative/resilience family with exact first-dispatch and retry mutation counts. |
| 7 | Add six-target native-root and reference validation | DONE | Expanded the existing six-target matrix with Attention markers; canonical route, real MCP, real CLI, and Markdown drift validation pass. |
| 8 | Add the corpus to the v1 fixture manifest | DONE | The fixture manifest now lists 22 artifacts including the route corpus with SHA-256; the HTTP checksum constant and Hero Code handoff pin the updated manifest hash. |

### Exercise-the-feature check

- [x] User-visible behavior was exercised end-to-end: all six installers
  rendered the same Attention chat-routing rules into their native root files;
  the 29-case fixture passed schema/DTO/policy/action/MCP/CLI validation;
  `go test ./contracts/attention ./internal/install ./internal/serve ./internal/cli`
  and `go test ./...` passed.

### Excellence Bar self-check

- [x] Yes — the guidance is portable and imperative, the corpus is
  machine-consumable and negative-capable, and every executable reference is
  checked against the surface that actually owns it.
