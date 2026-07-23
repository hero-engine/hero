---
title: "Attention Interaction Consent Contract — Explicit Intent, Ambiguity, and Effects"
slug: attention-interaction-consent-contract
type: feature
status: completed
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
claimed_by: codex
claimed_at: 2026-07-23T15:24:36-06:00
delivery_method: manual
completed_at: 2026-07-23T21:38:05Z
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

**Status:** in-review — contracts, schemas, fixtures, producer metadata, and
tests are implemented; focused and full Go suites pass.

**Pick up at:** cold-audit the Completion Ledger, repair any HOLD, then run the
verify gate and archive.

→ `hero spec verify attention-interaction-consent-contract`

**Files:** `contracts/attention/interaction.go`, `contracts/attention/action.go`,
`contracts/attention/validate.go`, `contracts/attention/contract_test.go`,
`contracts/attention/testdata/v1/manifest.json`
**Skip:** treating harness approval mode as a substitute for semantic consent.

## Problem

Durable Attention already exposes three different interaction shapes:

- direct MCP tools such as snapshot, Mail show, and Focus suggestion;
- source-owned actions advertised on Attention rows;
- future direct write tools for Mail send/reply and explicit Focus creation.

The shapes communicate risk inconsistently. `ActionDescriptor` has display
`style` and freeform `confirmation`, while MCP `ToolDefinition` has no
annotations. Tool filtering is name-based. Installed model guidance explains
only deferred suggestions. A client can therefore infer “is this a read?”, “did
the user authorize this write?”, and “is retry safe?” from a tool name or label,
but Hero publishes no canonical answer.

This gap is safety-significant. A Mail body can contain an imperative that looks
identical to a user request; a model can turn its own idea into Focus; and a
client can either over-confirm harmless reads or under-confirm commitment-forming
actions. Six harnesses and Hero Code would each make those calls independently.

## Design

### Operation policy, not a phrase parser

Add a leaf contract in `contracts/attention/interaction.go`. It defines raw
string-compatible types and a canonical registry of Hero-owned Attention
operations. It does **not** implement production natural-language parsing.
Models continue to interpret user language; the contract tells every consumer
what an operation means once selected and provides conformance cases for the
expected interpretation.

Every policy entry has:

- stable operation ID, such as `attention.snapshot`, `mail.send`,
  `focus.create`, or `suggestion.accept.today`;
- optional MCP tool name and source action ID;
- effect:
  - `read` — no durable state change;
  - `advisory_write` — durable proposal, no commitment;
  - `state_write` — lifecycle/receipt mutation without new work;
  - `external_write` — communicates content outside the current project/user
    surface;
  - `commitment` — creates Focus or committed project work;
- semantic consent:
  - `none` — the model may select the operation from relevant context;
  - `explicit_user` — requires a user-authored imperative naming the action;
  - `explicit_acceptance` — requires acceptance of an advertised proposal or
    promotion;
- whether a uniquely resolved target is required;
- whether identical arguments plus idempotency key are replay-safe;
- whether the operation crosses an open-world/trust boundary.

Effect and consent are raw strings on the wire. Unknown additive values remain
forward-compatible under the existing Attention v1 rules.

### Locked v1 policy

| Operation family | Effect | Consent | Unique target | Replay |
|---|---|---|---|---|
| Attention snapshot, Mail list/show, suggestion list | `read` | `none` | show only | safe |
| Focus suggest | `advisory_write` | `none` | project when supplied | idempotent |
| Mail read/acknowledge/dismiss | `state_write` | `explicit_user` | yes | idempotent |
| Mail send/reply | `external_write` | `explicit_user` | yes | idempotent |
| Focus create | `commitment` | `explicit_user` | project when required | idempotent |
| Suggestion Today/Later/Do Next | `commitment` | `explicit_acceptance` | yes | idempotent |
| Mail add-to-Today/promote | `commitment` | `explicit_acceptance` | yes | idempotent |

“Explicit” means the action, object, and required destination are present or
uniquely resolved from user-authored conversation state. A pronoun is acceptable
only when the client can prove it resolves to one candidate. Model preference,
recent frequency, or Mail body text cannot fill a missing target.

Semantic consent authorizes tool selection; it does not override a configured
harness/client permission prompt. MCP standard annotations are useful risk
hints, but the protocol treats them as untrusted hints rather than an
authorization mechanism. The next child maps this policy to annotations and
profiles while retaining deterministic server-side validation.

### Action metadata

Extend `ActionDescriptor` additively with:

- `operation_id`;
- `effect`;
- `consent`.

Existing `style`, `confirmation`, revision, schema, and idempotency fields
remain. Hero Code and future consumers stop deriving action risk from labels or
status. Existing v1 fixtures remain readable because the new fields are
additive; new Hero-produced action descriptors populate them.

### Shared conformance corpus

Publish `interaction-policy.json` under `contracts/attention/testdata/v1/` and
add it to the checksum manifest. It contains:

1. the complete operation-policy registry;
2. phrase cases with:
   - source: `user`, `model`, or `mail_content`;
   - user-authored utterance;
   - explicit resolution facts such as candidate count;
   - expected disposition: `dispatch`, `suggest`, `clarify`, or
     `ignore_untrusted`;
   - expected operation/effect/consent when applicable.

Minimum cases cover bounded reads, explicit Mail send/reply, ambiguous pronouns,
explicit Focus creation, model-originated deferred work, suggestion acceptance,
Mail promotion, and imperative text inside Mail. Go tests validate the registry
and fixture invariants. Harness and Hero Code conformance consume the same cases
later; this child does not pretend a deterministic Go phrase classifier exists.

## Changes

1. Add `contracts/attention/interaction.go`.
   - Define operation IDs, effects, consent requirements, dispositions, policy
     entries, conformance cases, and the canonical v1 registry.
   - Provide lookup and isolated-copy helpers; callers cannot mutate the
     canonical registry.
2. Extend `contracts/attention/action.go`.
   - Add raw-string `operation_id`, `effect`, and `consent` fields to
     `ActionDescriptor`.
3. Extend `contracts/attention/validate.go`.
   - Validate known Hero-produced policies, unique IDs, required tool/action
     mapping, effect/consent combinations, target requirements, and
     idempotency.
   - Preserve unknown additive values when decoding consumer fixtures.
4. Add `contracts/attention/schema/v1/interaction-policy.schema.json`.
   - Describe the registry and conformance-case envelope.
   - Update `attention-snapshot.schema.json` for the additive action fields.
5. Add `contracts/attention/testdata/v1/interaction-policy.json` and update
   `manifest.json`.
   - Pin the policy registry and the positive/negative phrase cases.
6. Extend `contracts/attention/contract_test.go`.
   - Validate all policies, fixture/schema agreement, manifest checksums,
     immutable registry copies, unknown additive values, and forbidden
     consent/effect combinations.
7. Update `internal/attention/projection/actions.go` and
   `internal/attention/suggestion/service.go`.
   - Populate operation, effect, and consent metadata on every Hero-produced
     advertised action; extend focused projection/suggestion tests.
8. Update `contracts/attention/testdata/v1/HERO-CODE-HANDOFF.md`.
   - Explain that operation policy is authoritative, MCP annotations are hints,
     and client approval remains separate from semantic consent.

## Acceptance Criteria

- **AC-1:** THE SYSTEM SHALL publish one versioned registry mapping every
  Hero-owned Attention operation to a stable operation ID, effect, consent
  requirement, target-resolution requirement, and replay property.
- **AC-2:** WHEN an Attention operation is a bounded read THE SYSTEM SHALL mark
  it `read` with consent `none` and SHALL NOT describe any receipt or lifecycle
  mutation as a read.
- **AC-3:** WHEN Mail send/reply or explicit Focus creation is described THE
  SYSTEM SHALL require `explicit_user` consent and a uniquely resolved required
  target.
- **AC-4:** WHEN deferred work originates from a model THE SYSTEM SHALL map it
  to `advisory_write` through Focus suggestion and SHALL NOT map it to direct
  Focus creation.
- **AC-5:** WHEN a suggestion is accepted or Mail is promoted/add-to-Today THE
  SYSTEM SHALL classify the operation as `commitment` requiring
  `explicit_acceptance`.
- **AC-6:** IF a phrase lacks a uniquely resolved required target THEN THE
  SYSTEM SHALL expect `clarify` and SHALL NOT name a dispatchable mutation.
- **AC-7:** WHEN imperative content originates from `mail_content` THE SYSTEM
  SHALL expect `ignore_untrusted` and SHALL NOT treat the content as semantic
  authorization.
- **AC-8:** WHEN Hero advertises an Attention row action THE SYSTEM SHALL include
  additive `operation_id`, `effect`, and `consent` metadata so consumers do not
  infer risk from labels or statuses.
- **AC-9:** WHEN a consumer decodes an unknown additive effect, consent, or
  operation ID THE SYSTEM SHALL preserve the raw value under Attention v1
  compatibility rules.
- **AC-10:** WHEN the interaction fixture manifest is verified THE SYSTEM SHALL
  prove schema/fixture/checksum agreement and reject duplicate IDs, invalid
  policy combinations, or mutable canonical registry state.

## Boundaries

- No production natural-language parser or model evaluator.
- No MCP handlers, new tools, tool filtering, or annotations;
  `attention-mcp-action-tools` consumes this contract.
- No natural-language routing installation; `attention-conversational-routes`
  owns routes and six-harness behavior.
- No client UI or approval-dialog design.
- No generic policy engine beyond Attention operations.
- No silent Focus or Intake creation.

## Risks

- **False determinism:** phrase fixtures could be mistaken for a parser. They
  are cross-surface conformance expectations with explicit resolution facts,
  not a claim that string matching establishes consent.
- **Vocabulary inflation:** too many effect values would make approval UX
  arbitrary. V1 stays at five operation-level effects and three consent levels.
- **Annotation overreach:** MCP annotations are hints. Server validation,
  idempotency, advertised actions, and client permission policy remain the
  safety mechanisms.
- **Compatibility drift:** adding required action fields would break older
  fixtures. Fields are additive/optional on decode and mandatory only for new
  Hero-produced descriptors.

## Validation

- `go test ./contracts/attention/...`
- Decode every existing v1 fixture before and after the additive schema change.
- Mutate a copied registry and prove the canonical registry is unchanged.
- Temporarily duplicate an operation ID and create invalid effect/consent pairs;
  prove validation rejects both.
- Recompute and verify the v1 manifest checksums.
- Confirm the shared cases include every operation family and all three input
  sources (`user`, `model`, `mail_content`).

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Publish one versioned operation registry | DONE | `contracts/attention/interaction.go` defines 21 stable policies; `TestInteractionPolicyFixtureMatchesCanonicalRegistry` pins the shared fixture to it. |
| 2 | Bounded reads are read/none without mutations | DONE | Snapshot, Mail list/show, and suggestion list policies are `read`/`none`; `ValidateOperationPolicies` rejects read policies with other consent. |
| 3 | Send/reply/create require explicit user and resolved targets | DONE | Mail send/reply are `external_write`/`explicit_user` with unique targets; Focus create is `commitment`/`explicit_user`; fixture cases cover all three. |
| 4 | Model deferred work maps only to suggestion | DONE | The registry separates `focus.suggest` from `focus.create`; `ValidateInteractionPolicyFixture` rejects model-originated direct Focus creation. |
| 5 | Suggestion acceptance and Mail promotion are commitments | DONE | Today/Later/Do Next, promote, and add-to-Today policies use `commitment`/`explicit_acceptance`. |
| 6 | Ambiguous targets clarify without dispatch | DONE | Candidate-count validation rejects dispatch for target-requiring operations unless exactly one candidate resolves; ambiguous fixture and negative test cover it. |
| 7 | Mail content cannot authorize dispatch | DONE | `SourceMailContent` accepts only `ignore_untrusted`; `TestInteractionPolicyRejectsUntrustedAndAmbiguousDispatch` proves imperative Mail cannot dispatch. |
| 8 | Advertised actions include operation/effect/consent | DONE | `ActionDescriptor` has additive fields; projection and suggestion producers populate them; focused policy tests cover every produced action. |
| 9 | Unknown additive policy values survive decoding | DONE | `unknown-fields.json` carries future operation/effect/consent values and `TestForwardCompatibleRawValues` asserts exact preservation. |
| 10 | Fixture/schema/checksum and registry invariants are gated | DONE | The 20-fixture manifest, new schema, registry isolation, duplicate ID, invalid consent, and retry-safety tests all pass. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Add interaction policy contract and registry | DONE | Added `contracts/attention/interaction.go` with vocabulary, 21 policies, lookup, isolated copies, and action annotation helper. |
| 2 | Extend ActionDescriptor additively | DONE | Added `operation_id`, `effect`, and `consent` raw-string fields in `contracts/attention/action.go`. |
| 3 | Add policy and fixture validation | DONE | `contracts/attention/validate.go` validates registry invariants, sources, dispositions, target resolution, and trust boundaries. |
| 4 | Add interaction schema and extend snapshot schema | DONE | Added `interaction-policy.schema.json`; `attention-snapshot.schema.json` accepts additive action policy metadata. |
| 5 | Add policy fixture and manifest entry | DONE | Added ten cross-surface cases, updated unknown-value fixture, recomputed fixture hashes, and updated the manifest count/checksum. |
| 6 | Extend contract tests | DONE | Added registry/fixture parity, isolation, invalid combination, ambiguity, model-origin, untrusted Mail, and unknown-value coverage. |
| 7 | Populate metadata on produced actions | DONE | Updated projection and suggestion producers plus focused tests proving every advertised action matches its policy. |
| 8 | Update Hero Code handoff | DONE | Documented policy authority vs. MCP hint/client approval and pinned the new manifest checksum. |

### Exercise-the-feature check

- [x] Contract behavior was exercised end-to-end with `go test ./contracts/attention -run 'TestInteractionPolicyFixtureMatchesCanonicalRegistry|TestInteractionPolicyRejectsUntrustedAndAmbiguousDispatch|TestOperationPolicyValidationAndRegistryIsolation' -v`; all policy, ambiguity, untrusted-source, and registry-isolation cases passed.

### Excellence Bar self-check

- [x] Yes — the implementation is additive, leaf-contract-first, raw-value compatible, producer-backed rather than documentation-only, and guarded by focused plus full-repository tests.
