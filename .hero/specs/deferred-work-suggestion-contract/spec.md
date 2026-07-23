---
title: "Deferred Work Suggestion Contract — Explicitly Accepted Focus"
slug: deferred-work-suggestion-contract
type: feature
status: completed
domain: engineering
priority: medium
size: large
horizon: next
created: 2026-07-20
parent: durable-attention
depends-on: [personal-focus-core]
conflicts-with: [project-mail-triage-and-provenance]
tags: [focus, harness, suggestions, prompts]
delivery_method: manual
completed_at: 2026-07-23T02:25:22Z
---

# Deferred Work Suggestion Contract — Explicitly Accepted Focus

## Context

Agents often discover useful work that should not interrupt the current loop.
Today they can mention it in prose or leave it in a harness-local todo, but
neither creates a reliable, cross-session choice for the user. Automatically
writing Personal Focus would be worse: a model would silently create personal
commitments.

Hero installs behavior into six harness targets. The mechanism must therefore
be a Hero-owned structured operation, not a Claude hook or assistant-prose
parser, and all generated guidance must teach the same authority boundary.

## Goal

Let an agent create a structured, non-commitment deferred-work proposal and let
the user explicitly dismiss it or accept it into Focus as Today, Later, or Do
Next, with deterministic replay and a safe launch intent.

## Kickoff

Adds consent-bound deferred suggestions as advisory records distinct from Focus,
with explicit user acceptance through CLI and MCP.

**Status:** completed — proposal storage, explicit Focus acceptance, CLI/MCP
surfaces, six-harness guidance, and all delivery gates passed.

**Pick up at:** consume the structured suggestion actions from a client while
preserving Hero's user-consent and client-owned session-launch boundaries.

→ `.hero/specs/deferred-work-suggestion-contract/spec.md`

**Files:** `internal/attention/suggestion/service.go`, `internal/cli/focus.go`, `internal/serve/mcp_tools_focus.go`, `domains/engineering/skills/deferred-work-suggestions/SKILL.md`
**Skip:** direct model-created Focus, prose parsing, and target-specific hooks are forbidden.

## Problem

Prose suggestions cannot be consumed reliably by Hero Code, and harness todos
are run-owned rather than user-owned. A direct `focus add` tool would let a model
bypass consent. `do_next` also has a two-authority boundary: Hero can durably
accept the intention, but only the client can create a correctly rooted session.

## Design

### Proposal authority

Add `internal/attention/suggestion` with a private global proposal store under
`focus/suggestions/<suggestion-id>.json`. A proposal carries kind, title,
reason, exact executable prompt, optional project ref, typed conversation/run
provenance, creation/expiry timestamps, revision, and state
`pending | accepted | dismissed | expired`.

Persisting a pending proposal is allowed because it is explicitly not a Focus
item or commitment. Pending proposals expire from active projection after seven
days but remain inspectable for replay/audit for 30 days; cleanup is opportunistic
on store access and never runs a watcher. Repeating a proposal idempotency key
with the same normalized payload returns the same proposal. Models cannot set
accepted state.

### Agent-facing proposal operation

Expose `hero focus suggest` and MCP tool `hero_focus_suggest`. Inputs are title,
reason, prompt, optional project ref, source run/session ref, and idempotency
key. Prompt/body inputs use stdin or structured tool fields, not shell argv.
Success returns a `DeferredWorkSuggestion` with advertised actions. It does not
create Focus.

The operation is best-effort: failure to persist a proposal must not fail the
agent's primary task. The structured error may be reported, but guidance must
not tell the agent to simulate success with prose.

### User actions

Expose dismiss and three acceptance modes:

- `today`: atomically `CreateOrGet` Focus in `today`, then mark proposal
  accepted with authoritative Focus ID/revision.
- `later`: same, with Focus state `later`.
- `do_next`: same durable acceptance into `today`, then return a
  `FocusLaunchIntent` containing the exact prompt and project resolution.
- `dismiss`: mark proposal dismissed; create no Focus item.

Every action requires suggestion revision and idempotency key. The store records
the action result before returning. If marking the proposal fails after Focus
creation, retry resolves through the Focus origin key and completes the receipt.
No client retries a stale/unsupported/missing action automatically.

Hero owns the atomic semantic result “accepted Focus exists”; Hero Code owns
session creation. A failed/cancelled launch leaves the Focus item in Today. V1
stores no Focus-session correlation and never infers completion from a harness
todo.

### Harness-neutral guidance

Add a canonical `domains/engineering/skills/deferred-work-suggestions/SKILL.md`
that defines when a suggestion is appropriate: meaningful work, outside the
current accepted scope, concrete enough to resume, and not merely an unfinished
required step. It instructs agents to invoke the structured Hero command/tool
once, never create Focus directly, and continue/finish the current task.

Hero's existing render-direct install copies this self-contained skill to
`opencode`, `cursor`, `claude`, `copilot`, `codex`, and `generic`. Extend install
fixtures for every target and the generated Skills Reference. Do not add
target-specific hooks.

### Fallback UX

CLI users list proposals with `hero focus suggestions [--pending] [--json]` and
act with `hero focus suggestion <id> today|later|do-next|dismiss --revision ...`.
Human output presents title, reason, project, and action commands. Structured
consumers use DTOs; no surface parses assistant prose.

## Changes

1. Add `internal/attention/suggestion/store.go`, `service.go`, and tests for
   pending proposals, expiry, cleanup, idempotency, and private persistence.
2. Extend `internal/cli/focus.go` with suggest/list/action commands and stable
   JSON responses.
3. Add `hero_focus_suggest`, `hero_focus_suggestions`, and
   `hero_focus_suggestion_action` in `internal/serve/mcp_tools_def.go`, dispatch,
   and `internal/serve/mcp_tools_focus.go`.
4. Add `domains/engineering/skills/deferred-work-suggestions/SKILL.md` and update
   the managed Skills Reference source.
5. Extend `internal/install` target fixture/golden tests for `opencode`,
   `cursor`, `claude`, `copilot`, `codex`, and `generic`, proving identical
   semantic guidance is installed natively for each harness.
6. Add failure-injection tests around Focus creation and proposal receipt
   commit to prove idempotent recovery.
7. Document that deferred suggestion is advisory output and not part of the
   Completion Ledger or a replacement for required delivery steps.

## Acceptance Criteria

- **AC-1:** WHEN an agent submits a valid structured deferred-work proposal, THE SYSTEM SHALL persist one pending suggestion and SHALL NOT create a Focus item.
- **AC-2:** WHEN the same proposal idempotency key and payload are replayed, THE SYSTEM SHALL return the original suggestion without duplication.
- **AC-3:** WHEN a user accepts a suggestion as Today or Later, THE SYSTEM SHALL create or reuse one source-linked Focus item in the chosen state and return its authoritative row.
- **AC-4:** WHEN a user chooses Do Next, THE SYSTEM SHALL durably accept one Focus item into Today and return its launch intent without creating a session.
- **AC-5:** WHEN a client fails or cancels after a successful Do Next response, THE SYSTEM SHALL leave the accepted Focus item in Today for safe retry.
- **AC-6:** WHEN a user dismisses a pending suggestion, THE SYSTEM SHALL create no Focus item and SHALL return an authoritative dismissed suggestion.
- **AC-7:** IF an action is stale, unsupported, missing, expired, or invalid THEN THE SYSTEM SHALL return a structured error and make no new commitment.
- **AC-8:** WHEN suggestion guidance is installed for any supported harness target, THE SYSTEM SHALL provide the same consent boundary and structured invocation behavior without a target-specific hook.
- **AC-9:** WHILE an agent is completing accepted work, THE SYSTEM SHALL NOT convert unfinished required steps, checklist items, or harness todos into deferred suggestions automatically.
- **AC-10:** WHEN Hero Code or another client consumes suggestions, THE SYSTEM SHALL provide structured records and advertised actions without requiring assistant-prose parsing.

## Boundaries

- No client chip/card UI, notification worker, auto-capture, or direct model
  authority to create Focus.
- No harness task synchronization, session correlation, or completion inference.
- No Claude-only hook or hidden fallback that writes a commitment.
- No global Attention HTTP projection; that is the read-model child.

## Risks

- Models may over-suggest; narrow guidance and expiry limit clutter, while the
  user remains the only commitment authority.
- Persisted proposal prompts can be sensitive; they inherit Focus private
  storage and prompt limits.
- Atomicity spans two files; deterministic Focus origin keys and resumable
  proposal receipts are required.
- Install behavior can drift by harness; six-target golden tests are a release
  gate, not a documentation promise.

## Validation

- `go test ./internal/attention/suggestion/... ./internal/attention/focus/...`
- MCP and CLI tests for propose, every action, exact replay, stale/missing/
  expired errors, and failed launch separation.
- Failure-injection test proving Focus exists once after a mid-acceptance crash.
- Install/golden tests for all six harness targets.
- Repository search assertion that no target-specific deferred-work hook or
  assistant-prose parser was added.
- `go test ./...` for installer, MCP, Focus, and CLI regressions.

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Structured proposal persists pending without Focus | DONE | `TestProposalPersistsPrivatelyWithoutCreatingFocusAndReplays` and `TestMCPFocusSuggestionToolsAreStructuredAndConsentBounded` assert private proposal persistence and zero Focus rows. |
| 2 | Same proposal key and normalized payload replays exactly once | DONE | `Store.CreateOrGet` compares normalized proposal content; service and CLI replay tests assert the original suggestion ID/JSON. |
| 3 | Today/Later acceptance creates or reuses one source-linked Focus row | DONE | `TestAcceptTodayLaterDoNextAndDismiss` covers both states and exact action replay; Focus uses deterministic `deferred_suggestion:<id>` origin keys. |
| 4 | Do Next accepts into Today and returns launch intent without session creation | DONE | Service, CLI, and MCP end-to-end tests assert Today plus exact prompt/project/path launch intent; no session API is imported or invoked. |
| 5 | Failed/cancelled client leaves accepted Do Next Focus in Today | DONE | Do Next tests stop after the response and verify the durable Focus row remains Today; the action has no rollback or completion inference. |
| 6 | Dismiss creates no Focus and returns authoritative dismissed proposal | DONE | Dismiss subtest asserts dismissed state and an empty Focus store. |
| 7 | Stale, unsupported, missing, expired, and invalid actions are structured and create no commitment | DONE | `TestActionErrorsMakeNoCommitment`, CLI error assertions, and MCP JSON error assertion cover all named error classes with zero Focus rows. |
| 8 | Every supported harness gets the same consent/invocation guidance with no hook | DONE | `TestAllTargetsInstallIdenticalDeferredWorkConsentGuidance` verifies byte-identical native installs for opencode, cursor, claude, copilot, codex, and generic. |
| 9 | Required current work is never auto-converted into a suggestion | DONE | Canonical skill explicitly forbids unfinished required steps, ACs, ledgers, and harness todos; implementation exposes only explicit invocation and adds no hook/watcher. |
| 10 | Consumers receive structured records and advertised actions without prose parsing | DONE | DTOs include state/revision/actions; CLI JSON and all three MCP tools are exercised as structured data, with no assistant-prose parser. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Add suggestion store/service and persistence tests | DONE | Added `internal/attention/suggestion/store.go`, `service.go`, and comprehensive service/store-backed tests for pending, expiry, retention, replay, permissions, and actions. |
| 2 | Extend Focus CLI with suggest/list/action and JSON | DONE | Added `focus suggest`, `focus suggestions`, and `focus suggestion`; prompt/reason bodies stay out of argv, legacy `registered: auto` project registries remain readable, and CLI tests cover JSON/replay/errors. |
| 3 | Add three MCP suggestion tools and dispatch | DONE | Added canonical definitions, read/mutate dispatch, `mcp_tools_focus.go`, decimal-string revisions, and structured MCP tests. |
| 4 | Add canonical harness skill and Skills Reference | DONE | Added `domains/engineering/skills/deferred-work-suggestions/SKILL.md` plus managed source/generated Skills Reference entries. |
| 5 | Verify all six harness target installations | DONE | Six-target fixture test asserts native paths, identical bytes, and semantic consent boundary phrases. |
| 6 | Add Focus-create / receipt-commit failure injection recovery | DONE | `TestReceiptWriteFailureRecoversIdempotently` proves one Focus row remains after injected receipt failure and retry completes the proposal receipt without duplication. |
| 7 | Document advisory/ledger boundary | DONE | Canonical skill states suggestions are advisory, not Focus, not a Completion Ledger item, and never a replacement for required delivery work. |

### Exercise-the-feature check

- [x] User-visible behavior was exercised end-to-end: focused CLI/MCP tests passed, then real `go run ./cmd/hero focus suggest ...` and `focus suggestion <id> do-next ... --json` commands against an isolated state root returned a pending proposal, authoritative Today Focus row, and exact project launch intent.

### Excellence Bar self-check

- [x] Yes — the implementation keeps proposals and commitments separate, makes every mutation replay-safe, preserves exact prompts and revisions across JSON, and proves identical consent guidance across every supported harness.
