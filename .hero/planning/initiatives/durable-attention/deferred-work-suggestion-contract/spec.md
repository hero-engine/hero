---
title: "Deferred Work Suggestion Contract — Explicitly Accepted Focus"
slug: deferred-work-suggestion-contract
type: feature
status: planning
domain: engineering
priority: medium
size: large
horizon: next
created: 2026-07-20
parent: durable-attention
depends-on: [personal-focus-core]
conflicts-with: [project-mail-triage-and-provenance]
tags: [focus, harness, suggestions, prompts]
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

Implement deferred suggestions as a distinct proposal store and service, not as
Focus state. Expose propose/accept/dismiss through CLI and MCP, then add one
canonical harness skill that installs to all six targets. Acceptance alone may
create Focus; `do_next` atomically accepts into Today and returns a launch intent
without starting a session.

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
