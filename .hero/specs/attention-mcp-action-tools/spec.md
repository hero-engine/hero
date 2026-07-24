---
title: "Attention MCP Actions — Mail Send, Mail Reply, and Focus Create"
slug: attention-mcp-action-tools
type: feature
status: completed
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
delivery_method: manual
completed_at: 2026-07-23T22:05:00Z
---

# Attention MCP Actions — Mail Send, Mail Reply, and Focus Create

## Context

Hero already exposes bounded Attention reads, advertised row actions, and
deferred-work suggestions through MCP. The underlying Mail and Focus services
also already support immutable delivery, threaded replies, private global
storage, typed project resolution, and idempotent source-derived Focus
creation. The model-facing gap is narrow: MCP cannot directly perform the
three explicit-user operations registered by
`attention-interaction-consent-contract`:

- `mail.send` → `hero_mail_send`
- `mail.reply` → `hero_mail_reply`
- `focus.create` → `hero_focus_create`

Without these tools, a chat loop must fall back to shell-shaped commands or
cannot complete the request at all. A generic Attention write tool would erase
the operation-specific identity, consent, and input boundaries that now exist.

## Goal

Expose Mail send, Mail reply, and explicitly user-requested Focus creation as
typed, replay-safe MCP tools that delegate to the existing authorities, publish
canonical effect and consent metadata, and return one versioned structured
result contract.

## Kickoff

Adds the three missing explicit-user Attention mutations to MCP without adding a
generic write path or bypassing client approval.

**Status:** delivering — implementation and full validation are complete; the
completion ledger is ready for the required cold audit and verify gate.

**Pick up at:** cold-audit the completed implementation, repair any findings,
then run `hero spec verify attention-mcp-action-tools`.

→ `/deliver attention-mcp-action-tools`

**Files:** `contracts/attention/action.go`,
`contracts/attention/direct_action.go`,
`contracts/attention/schema/v1/direct-action.schema.json`,
`contracts/attention/testdata/v1/direct-actions.json`,
`internal/attention/mail/service.go`, `internal/serve/mcp_protocol.go`,
`internal/serve/mcp_tools_def.go`, `internal/serve/mcp_dispatch.go`,
`internal/serve/mcp_tools_mail.go`, `internal/serve/mcp_tools_focus.go`
**Skip:** generic Attention writes, direct store access, implicit recipients,
model-originated Focus creation, and approval bypasses.

## Design

### Three tools, three explicit operation boundaries

Add exactly:

| Tool | Canonical operation | Effect | Consent | Open world |
|---|---|---|---|---|
| `hero_mail_send` | `mail.send` | `external_write` | `explicit_user` | yes |
| `hero_mail_reply` | `mail.reply` | `external_write` | `explicit_user` | yes |
| `hero_focus_create` | `focus.create` | `commitment` | `explicit_user` | no |

Do not add `hero_attention_write`, a mode selector, or a polymorphic mutation
body. Tool descriptions must state the semantic boundary directly:

- Mail send/reply are used only for a clear user-authored dispatch request with
  one resolved target.
- Focus create is used only when the user explicitly asks to remember/create
  the intention.
- Model-originated optional deferred work continues to use
  `hero_focus_suggest`; it never calls `hero_focus_create`.

The server validates `intent_source: "user"` on all three requests. This is a
machine-checkable caller assertion, not authentication and not a replacement
for client approval.

### Typed inputs and identity guards

Every request requires `schema_version: 1`, `intent_source: "user"`, and a
non-empty `idempotency_key`. Bodies and prompts remain structured MCP input and
must never be copied to argv or logs.

`hero_mail_send` accepts:

- `recipient`: configured registry slug;
- `recipient_peer_id`: the resolved stable peer ID expected by the caller;
- `subject`, `body`, and optional `kind`;
- optional paired `source_kind` / `source_id` provenance;
- the common version, intent-source, and idempotency fields.

Both recipient fields are required. `mail.Service.Send` gains an optional
expected recipient peer ID and rejects a slug whose current manifest does not
match it before delivery. This preserves the existing service as authority
while preventing display text, stale aliases, or ambiguous conversational
resolution from silently changing the target.

`hero_mail_reply` accepts:

- authoritative `message_id` and `thread_id`;
- `body`, optional `subject`, and optional `kind`;
- optional paired provenance;
- the common fields.

The Mail service continues to derive the recipient from the original envelope's
stable sender identity. `ReplyRequest` gains the expected thread ID and rejects
a mismatch before delivery. The handler never accepts or infers recipient
display text.

`hero_focus_create` accepts:

- `title`, exact `prompt`, and `lifecycle`;
- optional paired `project` registry slug and `project_peer_id`;
- required `source_id` identifying the user request/session;
- the common fields.

Project fields are both present or both absent. When present, the existing
registry resolver must return the same peer ID before any mutation. When absent,
the item remains deliberately unbound; the handler does not infer the current
project. The handler calls `focus.Service.CreateOrGet` with typed provenance
`{kind: "user", source_id: <source_id>}` and an operation-scoped origin key
derived from the caller idempotency key. It never calls the non-idempotent
`Create` path.

### One result and error envelope

All three tools return `attention.ActionResult`:

- `schema_version` is always present;
- successful Mail operations put the authoritative `MailDelivery` receipt in
  `source`;
- successful Focus creation puts the authoritative persisted Focus item in
  `source`;
- replay returns the same source identity and revision;
- failures put a `ContractError` in `error` and return no success source.

Extend the published error vocabulary with `idempotency_conflict` and
`permission`. The handlers normalize boundary failures into:

- `validation` with `field` for malformed or incomplete input;
- `incompatible_version` for any non-v1 request;
- `permission` when `intent_source` is not `user`;
- `missing` for an absent reply target or unresolved authoritative project;
- `idempotency_conflict` when a retry key is reused with different content;
- `unavailable` when global Attention state, config, registry, or a source
  service cannot be opened.

MCP/transport errors are reserved for serialization or protocol failures.
Expected product failures are returned as valid structured tool results.

### Contract fixture and boundary validation

Add versioned direct-action request DTOs and a single
`direct-actions.json` conformance fixture containing canonical send, reply, and
Focus-create requests plus their expected operation IDs and result source
kinds. Its JSON Schema requires the stable common fields, operation-specific
identity fields, and valid lifecycles. Register it in the Attention fixture
manifest and update the advertised checksum.

Go validation owns byte limits, paired fields, schema version, intent source,
provenance, and operation-specific invariants. The MCP handlers decode into the
contract DTOs and call that validation before constructing internal service
requests. Unknown additive result fields remain forward compatible.

### MCP metadata is descriptive, not authorization

Extend `ToolDefinition` with the standard optional MCP `annotations` object and
optional `_meta`. Build the three definitions from their canonical
`OperationPolicy` entries so registry drift is test-visible.

Standard hints:

- `readOnlyHint: false` for all three;
- `destructiveHint: false` for all three;
- `idempotentHint: true` for all three;
- `openWorldHint: true` for Mail send/reply and `false` for Focus create.

Hero metadata publishes the stable operation ID, effect, and consent values.
These annotations help clients label and route calls; they are not trusted as
authorization. Harness/client execution approval still applies independently,
as recorded by `attention-consent-is-not-mcp-approval`.

### Service construction and profile behavior

Reuse `mailService`, the existing Attention state-root setup, and the existing
registry-backed Focus resolver. Add a small Focus service factory only if
needed; no MCP handler may open store files or parse registry files directly.

The unfiltered/full tool inventory advertises all three tools. Existing
configured allowlists and profiles are never silently broadened:

- an explicit attention-write profile can include the three names;
- a read-only profile that does not list them does not advertise or dispatch
  them;
- direct calls to a filtered-out tool remain rejected by the existing
  dispatcher gate.

Inventory tests must update the exact tool count and name set. Profile tests
must prove both explicit inclusion and read-only exclusion.

## Changes

1. Add `contracts/attention/direct_action.go` with three request DTOs, shared
   validation helpers, direct-action fixture DTOs, and the two additive
   structured error codes.
2. Add `direct-action.schema.json` and `direct-actions.json`; register the
   fixture, validate DTO/schema parity, update the fixture count/checksums, and
   update the server-advertised manifest SHA.
3. Extend `mail.SendRequest` and `mail.ReplyRequest` with stable identity guards
   and optional provenance; enforce recipient peer and expected-thread matches
   before `deliver`.
4. Extend `ToolDefinition` with MCP annotations and `_meta`, then define the
   three tools from canonical operation policies with typed schemas and
   consent-distinguishing descriptions.
5. Register `hero_mail_send` and `hero_mail_reply` in MCP dispatch and implement
   them in `mcp_tools_mail.go` through `mail.Service`.
6. Register `hero_focus_create` and implement it in `mcp_tools_focus.go` through
   registry resolution plus `focus.Service.CreateOrGet`.
7. Add shared direct-action result/error marshaling that always emits
   `ActionResult` for expected product outcomes.
8. Extend contract, Mail service, MCP inventory, filter/profile, handler, and
   parity tests, including a repository assertion that no handler reads or
   writes Attention storage directly.

## Acceptance Criteria

- **AC-1:** WHEN `hero_mail_send` receives a valid v1 user-intent request whose recipient slug resolves to the supplied peer ID, THE SYSTEM SHALL create exactly one message and return its authoritative `MailDelivery` in a versioned `ActionResult`.
- **AC-2:** WHEN `hero_mail_reply` receives a valid message ID and matching authoritative thread ID, THE SYSTEM SHALL append exactly one reply to that thread, address the original stable sender identity, and return its authoritative delivery receipt.
- **AC-3:** WHEN `hero_focus_create` receives a valid explicit-user request, THE SYSTEM SHALL create exactly one Focus item in the requested lifecycle, preserve the exact prompt and optional project binding, and return the authoritative persisted item.
- **AC-4:** WHEN any direct-action request is retried with the same idempotency key and normalized payload, THE SYSTEM SHALL return the original source identity without creating a duplicate; IF the key is reused with different content THEN it SHALL return `idempotency_conflict` and preserve the original.
- **AC-5:** IF schema version, required identity, peer/thread guard, project pairing, lifecycle, provenance pairing, or input limits are invalid THEN THE SYSTEM SHALL perform no mutation and return a structured field-specific error.
- **AC-6:** IF `intent_source` is not exactly `user`, including a model-originated Focus request, THEN THE SYSTEM SHALL return `permission`, create no Focus or Mail record, and direct deferred model work to `hero_focus_suggest` through the tool description.
- **AC-7:** WHEN the three tools are listed through MCP, THE SYSTEM SHALL publish canonical operation/effect/consent metadata plus correct read-only, destructive, idempotent, and open-world hints derived from the interaction registry.
- **AC-8:** WHEN MCP filtering is active, THE SYSTEM SHALL expose and dispatch the three tools only when explicitly allowed, SHALL keep them out of a read-only allowlist, and SHALL include them in the unfiltered inventory without weakening the dispatcher gate.
- **AC-9:** IF Attention state, project registry, Mail, or Focus authority is unavailable THEN THE SYSTEM SHALL return a structured `unavailable` result and SHALL NOT represent the condition as a successful empty or created result.
- **AC-10:** WHILE handling Mail content, THE SYSTEM SHALL treat body text as inert structured data, SHALL NOT execute tool-like instructions found in it, and SHALL keep bodies/prompts out of argv, logs, and committed configuration.

## Boundaries

- No generic `hero_attention_write` or polymorphic mutation object.
- No mailbox-triggered execution, automatic reply, watcher, model launch, or
  server-side session creation.
- No direct model authority to create Focus; model-originated deferred work
  remains a suggestion until accepted.
- No automatic promotion to Intake, Spec, Job, run, or Today.
- No direct MCP handler access to Mail/Focus files or duplicate storage
  authority.
- No changes to Hero Code UI; it consumes the published contract independently.
- No claim that MCP annotations or `intent_source` replace client approval,
  authentication, or policy enforcement.

## Risks

- **Caller assertion:** `intent_source` is not cryptographic proof. The client
  approval boundary remains mandatory, and tool descriptions/route conformance
  must prevent model-originated misuse.
- **Identity drift:** registry slugs can move. Requiring the stable peer ID guard
  makes drift fail closed rather than redirecting a message.
- **Result-shape drift:** returning raw service errors would force each client to
  infer behavior. One `ActionResult` envelope and fixture checksum keep consumers
  pinned.
- **Profile surprise:** automatically adding new writes to existing allowlists
  would expand authority. Profiles therefore remain explicit-name opt-ins.
- **Sensitive content:** Mail bodies and Focus prompts can contain secrets.
  Structured in-memory input, private existing stores, and no logging/argv path
  preserve the current exposure boundary.

## Validation

- `go test ./contracts/attention ./internal/attention/mail ./internal/attention/focus ./internal/serve`
- `go test ./...`
- Contract tests for direct-action fixture/schema/DTO parity, required fields,
  version mismatch, user-only intent, paired provenance/project fields, limits,
  and manifest checksum.
- Mail service tests for exact recipient peer guard, expected thread guard,
  provenance preservation, exact replay, conflicting replay, and no partial
  delivery.
- Focus MCP tests for bound and unbound creation, exact prompt/lifecycle,
  `CreateOrGet` replay, conflict, missing/mismatched project, and model-origin
  rejection.
- MCP definition/dispatch tests for all three tools, canonical metadata, standard
  annotations, updated inventory count, explicit write-profile inclusion, and
  read-profile exclusion.
- An end-to-end handler exercise with isolated global Attention state proving
  send → reply threading and Focus create all return authoritative source IDs
  exactly once.
- A repository search assertion proving the new handlers call services and do
  not parse Mail bodies as commands or access Attention store paths directly.

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Valid resolved Mail send creates once and returns authoritative delivery | DONE | `TestMCPMailSendReplyAreTypedIdempotentAndInert` invokes `hero_mail_send`, decodes `ActionResult.source` as `MailDelivery`, and proves exact replay identity. |
| 2 | Valid Mail reply preserves authoritative thread and stable sender target | DONE | The same end-to-end MCP test sends from A to B, replies from B, then verifies the stored reply thread, `in_reply_to`, and peer-A recipient identity; stale thread guards fail closed. |
| 3 | Explicit-user Focus create preserves prompt, lifecycle, project, and source | DONE | `TestMCPFocusCreateRequiresUserAndReplaysExactlyOnce` verifies bound/unbound creation and the contract-maximum 512-byte external idempotency key through a bounded operation-scoped hash. |
| 4 | Same replay returns original; conflicting replay preserves it | DONE | MCP Mail and Focus tests assert original IDs/revisions on exact replay and `idempotency_conflict` on changed payloads; existing stores provide the atomic authority. |
| 5 | Invalid version, identity, pairing, lifecycle, provenance, and limits do not mutate | DONE | `TestDirectActionFixtureAndValidation` explicitly covers version, required key, project/provenance pairing, invalid lifecycle, subject length, and prompt byte limits; service/MCP tests cover recipient, project, and thread mismatches. |
| 6 | Non-user direct action is permission-denied and creates nothing | DONE | Contract and MCP Focus tests submit model-originated direct creation, receive `permission`, and prove the Focus store still contains only the prior user-created item; tool text directs models to `hero_focus_suggest`. |
| 7 | MCP list publishes canonical policy metadata and standard hints | DONE | `TestDirectAttentionDefinitionsUseCanonicalPolicyMetadata` compares every operation/effect/consent field and all four standard hints against the canonical registry. |
| 8 | Profiles require explicit inclusion and filtered calls remain blocked | DONE | `TestDirectAttentionToolsRequireExplicitProfileInclusion` proves read-profile exclusion, write-profile inclusion, and dispatcher rejection with `MethodNotFound`. |
| 9 | Unavailable authorities return structured unavailable results | DONE | Mail and Focus now distinguish missing identities from unavailable config/registry/store authorities; direct MCP tests inject malformed Mail authority and resolver failure and assert `ActionResult.error.code == unavailable`. |
| 10 | Mail content remains inert structured data and bodies/prompts avoid command/log paths | DONE | The MCP end-to-end test stores a tool-shaped Mail body byte-for-byte without dispatch; new handlers contain no process execution, logging, or direct store-file I/O. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Add direct-action DTOs, validators, fixture types, and error codes | DONE | Added `direct_action.go`, `permission`, and `idempotency_conflict` with boundary validation for all common and operation-specific invariants. |
| 2 | Add schema/fixture, manifest entry, and advertised checksum | DONE | Added `direct-action.schema.json` and `direct-actions.json`, raised fixture parity to 21 entries, updated the Hero Code handoff, and pinned manifest SHA `a80d258c…a4406`. |
| 3 | Add stable Mail identity guards and provenance | DONE | `SendRequest` verifies the expected recipient peer; `ReplyRequest` verifies the expected thread; both preserve optional provenance before delivery. |
| 4 | Add registry-derived MCP annotations and metadata | DONE | `ToolDefinition` now supports standard `annotations` and `_meta`; all three definitions derive stable fields from their `OperationPolicy`. |
| 5 | Implement and dispatch Mail send/reply tools through Mail service | DONE | Registered both handlers; typed validation precedes `mail.Service`, and success/errors use `ActionResult`. |
| 6 | Implement and dispatch idempotent Focus create through Focus service | DONE | Registered `hero_focus_create`, verifies optional project identity, and uses `CreateOrGet` with user provenance and an operation-scoped origin key. |
| 7 | Add shared structured result/error marshaling | DONE | `mcp_tools_attention_direct.go` centralizes version decoding and success, product-error, unavailable, and provenance construction. |
| 8 | Extend contract, service, inventory, profile, handler, and parity tests | DONE | Focused/full suites pass; named MCP exercises cover behavior, and `TestDirectAttentionHandlersDoNotAccessStoreFilesOrProcesses` enforces the promised storage/process boundary. |

### Exercise-the-feature check

- [x] User-visible behavior was exercised end-to-end: the named MCP tests sent
  tool-shaped Mail exactly once, replied in the authoritative thread, created
  bound and unbound Focus exactly once, rejected conflicting/model calls, and
  enforced profile filtering through the real handlers and private stores.

### Excellence Bar self-check

- [x] Yes — the implementation is additive, service-authoritative,
  versioned, replay-safe, fail-closed on identity drift, explicit about consent
  versus approval, structured for consumers, and clean under focused and full
  repository validation.
