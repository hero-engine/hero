---
title: "Durable Attention — Project Mail, Personal Focus, and Explicit Promotion"
slug: durable-attention
type: initiative
status: completed
autonomy: autonomous
domain: engineering
priority: high
size: giant
size_ack: giant
horizon: next
created: 2026-07-20
tags: [attention, mail, focus, peering, intake, cross-project]
child:
  - durable-attention-contracts
  - project-mail-core
  - personal-focus-core
  - project-mail-triage-and-provenance
  - deferred-work-suggestion-contract
  - attention-read-model-v1
  - peering-over-project-mail
relations:
  - target: hero-idea-primitive-core
    kind: related
  - target: intake-capture-loop
    kind: related
  - target: peer-call-multi-cli
    kind: related
  - target: always-on-runtime
    kind: related
  - target: job-run-contract-v1
    kind: related
completed_at: 2026-07-23T04:18:39Z
---

# Durable Attention — Project Mail, Personal Focus, and Explicit Promotion

## Goal

Deliver local-first Project Mail and user-global Personal Focus as distinct,
Hero-owned durable primitives; expose them through CLI/API and one stable
consumer read model; and reframe peering over asynchronous mail without making
model execution part of the communication contract.

## Vision

Hero gains a durable attention layer between a live agent run and committed
project work. Projects can receive and triage asynchronous requests without
launching a model or creating a spec on receipt. People can remember prompt-backed
work across projects without turning harness task lists into a personal planner.
Both surfaces remain useful from the CLI and API before any desktop UI or
always-on runtime exists.

## Design

### Ownership model

The initiative deliberately keeps six lifecycles distinct:

| Primitive | Owner | Meaning |
|---|---|---|
| Project Mail | Project | An addressed external signal awaiting project triage |
| Focus Item | User | A durable personal intention across sessions and projects |
| Harness task | Run/session | Execution steps for the current objective |
| Intake | Project | Pre-commitment intent awaiting promotion or rejection |
| Spec | Project/team | Committed work with a delivery lifecycle |
| Job/Run | Runtime | Authorized autonomous execution |

Mail and Focus may share provenance references and one combined read projection.
They do not share a write model, storage authority, statuses, or mutation API.

Hero core owns contracts, validation, identity and addressing, durable storage,
local delivery, threading and acknowledgement, explicit promotion, Focus
lifecycle, CLI/MCP/API surfaces, events, and trust enforcement.

Hero Code owns native presentation, the user-global Attention/Today experience,
triage controls, notifications, and launching a saved prompt into the correct
project and a new session. Harnesses continue to own their ephemeral task lists.
Always-On Runtime may later monitor and execute explicitly authorized requests;
that behavior is not part of mailbox core.

Incoming messages are untrusted data. Delivery, receipt, or acknowledgement
never authorizes execution.

## Specs

| Wave | Child | Priority | Size | Outcome |
|---:|---|---|---|---|
| 0 | `durable-attention-contracts` | critical | large | Versioned, separate Mail and Focus contracts plus ownership, storage, compatibility, and trust rules |
| 1 | `project-mail-core` | high | large | Project-addressed immutable mail, local transport, threading, acknowledgement, and CLI |
| 1 | `personal-focus-core` | high | large | User-global prompt-backed Focus lifecycle and CLI/API |
| 2 | `project-mail-triage-and-provenance` | high | large | Explicit promotion, provenance, unread surfaces, graph events, and MCP/API actions |
| 2 | `deferred-work-suggestion-contract` | medium | large | Harness-neutral model suggestion with explicit user acceptance into Focus |
| 2 | `attention-read-model-v1` | medium | large | Consumer-safe combined read projection, advertised actions, schema, and fixtures |
| 3 | `peering-over-project-mail` | medium | large | Existing peering identity and entry points reframed over asynchronous mail |

Every child now has a full implementation design, file-level change plan,
numbered acceptance criteria, risk boundaries, and validation strategy.

## Dependencies and delivery order

```text
durable-attention-contracts
  ├─ project-mail-core
  │    └─ project-mail-triage-and-provenance ─┐
  │          ├─ attention-read-model-v1
  │          └─ peering-over-project-mail
  └─ personal-focus-core
       ├─ project-mail-triage-and-provenance
       └─ deferred-work-suggestion-contract ──┤
                                              └─ attention-read-model-v1
```

`project-mail-triage-and-provenance` also depends on
`hero-idea-primitive-core`: Intake is the existing pre-commitment destination
and must not be reinvented.

Wave 1 must be usable without Hero Code, a daemon, a watcher, a model launcher,
or Always-On Runtime. Wave 2 publishes the stable consumer seam Hero Code can
implement against. Wave 3 changes peering only after mail and triage are real.

Two concrete same-file seams are declared as reciprocal `conflicts-with` soft
mutexes for `/drive`:

- `project-mail-core` and `personal-focus-core` both register new root CLI
  command families and may update shared completion/reference fixtures.
- `project-mail-triage-and-provenance` and
  `deferred-work-suggestion-contract` both extend MCP definitions/dispatch and
  install/reference surfaces.

## Existing work disposition

### `peer-call-multi-cli`

Do not deliver it as currently framed. Teaching Hero to spawn every current and
future harness solves the wrong layer. When `peering-over-project-mail` is
materialized, supersede `peer-call-multi-cli` through `hero supersede`; a
synchronous convenience command may survive only as an adapter over
send/claim/reply, not as the peering architecture.

### `handoff-one-call-simplification`

Leave it untouched. It simplifies NEXT and cross-session continuity. This
initiative distinguishes:

- session handoff: context for a future session;
- project mail: communication across project boundaries;
- work transfer: a mail request explicitly accepted and promoted.

### `hero-idea-primitive-core` and `intake-capture-loop`

Finish the current Intake review gate and reuse its capture/promotion path.
`intake-capture-loop` remains separate: it records intent behind loose work that
already happened, while Focus records future personal intent and Mail records an
external signal. Its later design must prevent one accepted suggestion or mail
promotion from being silently captured again as a duplicate Intake.

### `always-on-runtime` and `job-run-contract-v1`

Relate, do not merge. A future `mail-to-job-dispatch` child belongs under
`always-on-runtime` and depends on both `job-run-contract-v1` and
`project-mail-triage-and-provenance`. Mail does not gain its own worker,
scheduler, model, budget, or approval lifecycle.

## Cross-repo contract with Hero Code

Hero publishes a versioned Attention read model, JSON Schema, golden fixtures,
stable source IDs, project references, timestamps, unread state, and explicitly
advertised actions. Hero Code consumes those artifacts rather than reading
mailbox or global Focus storage directly, inferring capabilities from statuses,
or defining Swift-only lifecycle values.

The consumer design must account for two existing Hero Code names:

- `AdvisorViewData.FocusItem` is currently a derived, spec-backed advisor pick,
  not durable user-owned Focus storage.
- `NeedsAttention` currently represents backlog-health signals, not the combined
  Mail and Focus projection.

Those existing types may inform presentation, but they must not silently become
the new durable contracts. The existing `AdvisorSessionLauncher` is also
slug/action-oriented; a saved Focus prompt requires project identity plus an
arbitrary executable prompt.

The paste-ready coordination brief is
[hero-code-spec-out-prompt.md](hero-code-spec-out-prompt.md). Use it with a
`spec-out` peer call or paste it into an existing Hero Code task if the current
peer runner is unavailable.

After Hero Code produced its consumer design, Hero recorded the requested
contract surface and proposed v1 decisions below. The paste-ready return brief
is [hero-code-contract-response.md](hero-code-contract-response.md).

### Hero Code consumer design received

Hero Code designed `durable-attention-consumer` at
`../hero-code/.hero/planning/features/durable-attention-consumer/spec.md`. It is
a `large`, `high`-priority vertical feature and is correctly blocked on:

- `durable-attention-contracts`;
- `project-mail-triage-and-provenance`;
- `deferred-work-suggestion-contract`;
- `attention-read-model-v1`.

The peer design confirms the Swift presentation, process-global store,
project-routing, and session layers can absorb the feature. It also confirms the
existing Advisor data models cannot serve as the durable contract.

The upstream Hero contract must provide:

- versioned snapshots/rows, raw source kinds, stable project/provenance
  references, display fields, revisions, unread/Today grouping, and advertised
  actions;
- action input schemas, preconditions, idempotency, authoritative results, and
  structured stale/unsupported/missing/validation/version/unavailable errors;
- promotion artifact/navigation references;
- one user-global transport usable before a project is open;
- canonical JSON Schemas, fixture manifest/checksums, and golden fixtures.

Cross-repo dependency names remain prose here because the local graph cannot
resolve a peer repository's spec slug.

### Locked compatibility decisions

The child designs lock these v1 decisions:

1. `do_next` atomically accepts the suggestion into a durable `today` Focus
   item and returns a launch intent. Creating a desktop session is a separate
   client effect; launch failure leaves the Focus item safely in Today.
2. Mail `add_to_today` creates an idempotent, separately-owned Focus item linked
   to the Mail source. It does not mutate Mail into a Focus lifecycle or imply
   acknowledgement/dismissal.
3. Focus launch does not require persisted session correlation in v1. If
   observability needs it, return a typed launch context; never overload a spec
   slug or infer completion from harness todos.
4. Hero publishes exactly one user-global client boundary: Hero Serve HTTP at
   `/api/attention/v1`. CLI and MCP are Hero-side adapters over the same service;
   Hero Code implements only HTTP.
5. V1 requires an authoritative snapshot plus explicit refresh after foreground,
   reconnect, and mutations. A streaming event cursor is optional scope and
   must justify its daemon/service and recovery complexity before becoming a
   delivery dependency.

## Acceptance Criteria

- **AC-1:** THE SYSTEM SHALL keep Project Mail, Personal Focus, harness tasks, Intake, Specs, and Job/Run as distinct ownership and lifecycle contracts.
- **AC-2:** WHEN a peer delivers mail THE SYSTEM SHALL persist and expose it without launching a model, creating committed work, or dirtying the recipient project's tracked source tree.
- **AC-3:** WHEN a user accepts a deferred-work suggestion THE SYSTEM SHALL create a durable Focus Item, and no durable commitment SHALL exist before acceptance.
- **AC-4:** WHEN a project accepts a mail request as work THE SYSTEM SHALL explicitly promote it through the existing Intake or Spec path with source provenance.
- **AC-5:** THE SYSTEM SHALL expose Mail and Focus through CLI/API surfaces that do not require Hero Code or an always-on runtime.
- **AC-6:** THE SYSTEM SHALL publish a versioned consumer contract and golden fixtures that combine Mail and Focus for reading without merging their write models.
- **AC-7:** IF a mail body contains commands, prompts, attachments, or executable-looking content THEN THE SYSTEM SHALL continue to treat it as untrusted data and SHALL NOT execute it on receipt.
- **AC-8:** WHERE deferred-work guidance is installed THE SYSTEM SHALL provide self-contained, harness-neutral behavior across `opencode`, `cursor`, `claude`, `copilot`, `codex`, and `generic`.
- **AC-9:** WHEN peering is reframed over Project Mail THE SYSTEM SHALL preserve peer identity, alias/path resolution, manifests, provenance, and compatible user entry points.

## Non-Goals

- No calendar, due dates, estimates, assignments, priority scores, subtasks, or
  general-purpose personal task manager.
- No synchronization of Claude, Codex, Copilot, Cursor, Gemini, or OpenCode
  internal task graphs.
- No daemon, hosted message grid, email provider, A2A implementation, or cloud
  transport in v1.
- No mandatory streaming subscription, replay window, or gap-recovery protocol
  in v1; snapshot refresh is the minimum consumer contract.
- No automatic execution, auto-accept, scheduler, background worker, or approval
  engine in mailbox core.
- No direct Hero Code reads of Hero-owned storage.
- No automatic conversion of every message or Focus Item into an Intake or Spec.

## Risks

- **Semantic collapse:** one generic Attention write model would blur ownership
  and recreate task-system complexity. Keep only the read projection shared.
- **Prompt injection:** mail content may be adversarial. Treat it as data and
  require explicit acceptance before any transition toward execution.
- **Working-tree mutation:** cross-repo delivery must not create tracked files in
  a recipient repository behind its user.
- **Harness coupling:** Claude-only hooks or model launchers would reproduce the
  current peering failure. Core behavior must remain harness-neutral.
- **Dual storage:** Hero Code may be tempted to persist its own statuses. Golden
  contract tests and advertised actions keep Hero authoritative.
- **Name collision:** Hero Code's existing Focus/NeedsAttention presentation
  types could be mistaken for the new durable domain model.

## Test Strategy

Each child owns its detailed acceptance criteria and focused tests after its
progressive `/design` pass. The initiative closes only when all seven children
pass their delivery audit and `hero spec verify` gate.

Cross-child validation must cover these initiative invariants:

- **Contract fixtures:** generated JSON Schemas, manifests, checksums, and
  golden fixtures decode through both Hero's contract tests and Hero Code's
  consumer tests without direct storage access.
- **Ownership isolation:** Mail mutations never change Focus lifecycle directly;
  Focus mutations never change Mail lifecycle; harness task state never writes
  either store.
- **Local delivery:** repeated and interrupted sibling delivery is atomic and
  idempotent and does not dirty the recipient project's tracked worktree.
- **Explicit promotion:** Mail-to-Intake/Spec and Mail-to-Today operations are
  idempotent, retain source provenance, and return authoritative artifact/Focus
  references.
- **Untrusted input:** executable-looking Mail content is stored and displayed
  as data and cannot trigger a model, command, Job/Run, or client session on
  receipt.
- **Consumer compatibility:** snapshot refresh, unknown additive fields/source
  kinds/actions, absent capabilities, structured failures, stale data, and
  missing projects behave according to the published v1 fixtures.
- **Peering regression:** existing peer identity, manifests, aliases, and legacy
  trails remain readable while communication works with no model CLI installed.
- **Harness parity:** deferred-work guidance is generated and verified for all
  six install targets.

Before the initiative completes, run the full Go suite, focused CLI/MCP/peering
integration tests, fixture compatibility checks, all-target install tests, and
the Hero Code consumer suite against the released fixture manifest. Optional
streaming tests are required only if a child design deliberately includes the
streaming capability in v1.

## Progress

Composition and progressive design are complete. All seven children are
implementation-ready and score Grade A. Begin delivery with
`durable-attention-contracts`; `/drive` must honor the declared dependencies and
reciprocal same-file soft mutexes.

## Kickoff

Design is complete for `durable-attention`. Drive the initiative from
`durable-attention-contracts`, then follow the frontmatter dependencies and
`conflicts-with` soft mutexes. Each child contains concrete files, numbered EARS
criteria, failure boundaries, and validation commands; do not reopen locked v1
decisions unless implementation evidence contradicts them.

**Do not:** launch models from mailbox core, create receiver specs on delivery,
merge Mail and Focus writes, or let Hero Code invent parallel persistence.
