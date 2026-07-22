---
title: "Personal Focus Core — Prompt-Backed Work Across Projects"
slug: personal-focus-core
type: feature
status: planning
domain: engineering
priority: high
size: large
horizon: next
created: 2026-07-20
parent: durable-attention
depends-on: [durable-attention-contracts]
conflicts-with: [project-mail-core]
tags: [focus, personal, cross-project, prompts]
---

# Personal Focus Core — Prompt-Backed Work Across Projects

## Context

Hero has project specs, Intake signals, tracker work, and harness-local task
lists, but none represents a user's small cross-project list of intentions to
resume later with a saved prompt. Using specs would over-commit the work; using
harness todos loses it at the session boundary. Focus must remain a private,
user-authoritative primitive.

## Goal

Implement a user-global durable Focus store and CLI for capturing prompt-backed
intentions, moving them among `inbox`, `today`, `later`, and `done`, resolving
optional projects, and returning safe launch intents without executing them.

## Kickoff

Build Personal Focus on the contract state root with atomic per-item updates and
optimistic revisions. Keep it independent of project specs and harness todos.
Manual creation is durable; project-derived creation requires an idempotency
source key. Return launch intents containing the saved prompt and resolved
project, but never create a session or infer completion.

## Problem

A lightweight personal list becomes a second project manager if it acquires
priority, assignments, estimates, or spec synchronization. It becomes unsafe if
models can silently add commitments. It becomes unreliable if saved prompts
cannot survive concurrent clients or if a missing project causes launch in the
wrong repository.

## Design

### Store and item model

Add `internal/attention/focus`. The injected global state root stores
`focus/items/<focus-id>.json` plus a small lock file. Each contract `FocusItem`
contains title, executable prompt, lifecycle state, optional `ProjectRef`,
origin/provenance, `created_at`, `updated_at`, and opaque revision. Files are
private, written atomically, and sorted at read time rather than maintaining a
second index.

Revisions are content-derived opaque hashes over canonical persisted fields.
Every mutation accepts an expected revision; mismatch returns `stale`. Store
operations take an exclusive lock around compare-and-replace so two processes
cannot lose an update.

### Lifecycle

The only states are `inbox`, `today`, `later`, and `done`. Any active item may
move to any other active state or `done`; a done item may be reopened to an
active state. Repeating the same transition is idempotent and does not change
`updated_at`. V1 has no destructive delete; `done` is the terminal presentation
state and can be excluded by default listings.

### Creation authority and idempotency

`hero focus add` is an explicit user action and persists immediately. Service
callers creating from another Hero source must supply a typed provenance
reference and idempotency key. The store keeps `origin_key` on the item and
returns the existing item for exact replay. A different payload with the same
key fails. This is the bridge used later by Mail `add_to_today`.

Model-proposed work does not call `Create` directly; it goes through the
suggestion authority added by `deferred-work-suggestion-contract`.

### Project resolution and launch

A project reference is optional. On creation from a workspace, Hero captures
its canonical peer ID plus registry slug/display name. On list/show, resolution
consults the user-global registry and configured peers. Missing projects remain
visible with `availability: missing`.

`Service.LaunchIntent` returns item ID/revision, exact saved prompt, and either a
resolved absolute local project target or a structured missing-project error.
It never picks the current directory as a fallback, invokes a harness, creates a
session, marks the item done, or stores session correlation. The client owns
session creation and may retry safely.

### CLI

Add:

- `hero focus add --title ... --prompt-file -|path [--project ...] [--state ...]`
- `hero focus list [--state inbox|today|later|done|all] [--json]`
- `hero focus show <id> [--json]`
- `hero focus move <id> <state> --revision <revision> [--json]`
- `hero focus done <id> --revision <revision> [--json]`
- `hero focus launch <id> [--json]`

Prompts use stdin/files rather than argv. Human `launch` output identifies the
project and prints the prompt only after explicit invocation; `--json` returns
the typed launch intent.

## Changes

1. Add `internal/attention/focus/store.go`, `lock.go`, and tests for private
   per-item persistence, canonical revisions, atomic writes, and concurrency.
2. Add `internal/attention/focus/service.go` for validation, lifecycle,
   provenance/idempotency, project resolution, and launch intents.
3. Add `internal/attention/focus/project.go` as an adapter over
   `internal/serve.Registry` and peering identity without leaking registry paths
   into contracts.
4. Add `internal/cli/focus.go` and focused command tests; register `focusCmd` in
   `internal/cli/root.go`.
5. Add completion/reference documentation for Focus commands and clarify that
   Focus is not a spec, Intake item, tracker issue, or harness todo.
6. Add tests for the source-derived `CreateOrGet` operation that later Mail and
   deferred-suggestion features will use.

## Acceptance Criteria

- **AC-1:** WHEN a user explicitly adds a valid Focus item, THE SYSTEM SHALL persist one private item with an exact prompt, stable ID, initial revision, timestamps, and requested lifecycle state.
- **AC-2:** WHEN an item is moved among `inbox`, `today`, `later`, and `done`, THE SYSTEM SHALL enforce the supplied revision and return the authoritative updated item.
- **AC-3:** WHEN the same lifecycle transition is replayed, THE SYSTEM SHALL return the existing item without changing its revision or `updated_at`.
- **AC-4:** WHEN source-derived creation repeats the same origin key and payload, THE SYSTEM SHALL return the original Focus item without duplication.
- **AC-5:** IF a source origin key is reused with a different payload THEN THE SYSTEM SHALL return `idempotency_conflict` and preserve the original item.
- **AC-6:** WHEN a referenced project cannot be resolved, THE SYSTEM SHALL keep the Focus item visible as missing and SHALL NOT launch in the current or another project.
- **AC-7:** WHEN launch intent is requested for a resolvable item, THE SYSTEM SHALL return the exact saved prompt and resolved project target without invoking a harness, creating a session, or changing Focus state.
- **AC-8:** WHILE multiple processes mutate Focus, THE SYSTEM SHALL use optimistic revision checks plus exclusive atomic replacement so no successful update is silently lost.
- **AC-9:** WHEN Focus is listed, THE SYSTEM SHALL order active items deterministically and exclude `done` by default while supporting explicit state filters.
- **AC-10:** WHILE Personal Focus operates, THE SYSTEM SHALL NOT create or update specs, Intake items, tracker issues, harness task lists, due dates, assignments, estimates, or priorities.

## Boundaries

- No model suggestions, `do_next`, Hero Code UI, session launcher, notifications,
  calendar, subtasks, destructive deletion, or automatic completion.
- No import from specs/trackers and no synchronization with harness todos.
- No common Mail/Focus write model.

## Risks

- Saved prompts can contain sensitive text; private permissions and no argv
  prompt values are required.
- Content-derived revisions need canonical encoding; tests must prevent map
  ordering from changing revisions.
- Registry slugs can change; peer ID is canonical and missing projects remain
  recoverable rather than silently rebound.
- A large number of per-item files may eventually need indexing, but v1 favors
  inspectable atomic storage over premature database machinery.

## Validation

- `go test ./internal/attention/focus/... ./internal/cli/...`
- Lifecycle table tests, stale revisions, idempotency conflicts, exact prompt
  preservation, missing projects, reopen behavior, and deterministic ordering.
- Concurrent mutation test with two store instances on one injected root.
- CLI tests ensuring prompts are not required in argv and JSON is stable.
- `go test ./...` for registry, config, peering, and CLI regressions.
