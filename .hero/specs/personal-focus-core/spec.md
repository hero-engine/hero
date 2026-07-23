---
title: "Personal Focus Core — Prompt-Backed Work Across Projects"
slug: personal-focus-core
type: feature
status: completed
domain: engineering
priority: high
size: large
horizon: next
created: 2026-07-20
parent: durable-attention
depends-on: [durable-attention-contracts]
conflicts-with: [project-mail-core]
tags: [focus, personal, cross-project, prompts]
delivery_method: manual
completed_at: 2026-07-23T01:33:53Z
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

Adds a private cross-project Focus list with exact saved prompts and safe,
side-effect-free launch intents.

**Status:** delivering — implementation and validation are complete; the cold
audit returned a clean SHIP with all 10 criteria evidenced.

**Pick up at:** run the closing `hero spec verify personal-focus-core` gate and
archive the completed delivery.

→ `.hero/planning/initiatives/durable-attention/personal-focus-core/spec.md`

**Files:** `internal/attention/focus/service.go`, `internal/attention/focus/store.go`, `internal/cli/focus.go`, `web/docs/src/cli/focus.md`, `.hero/planning/initiatives/durable-attention/personal-focus-core/spec.md`
**Skip:** changing the published v1 Focus DTO; optional project binding is handled by the local persisted item.

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

## Completion Ledger

Implemented in Go using the existing attention contracts and state-root seam.
Validation performed with focused unit/CLI tests, the race detector, `go vet`,
the complete repository test suite, and a strict MkDocs build. The published v1 `int64` revision is a
positive opaque value derived from SHA-256 over canonical persisted fields; the
local persisted item uses a pointer project so unbound items remain compatible
without changing the completed portable contract.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Explicit add persists a private exact-prompt item | DONE | `internal/attention/focus/store.go`, `service.go`, and `internal/cli/focus.go`; tests verify IDs, timestamps, exact prompt bytes, revision, state, `0600` persistence, automatic current-workspace binding when registered, and unbound creation otherwise. |
| 2 | Moves enforce supplied revision and return authority | DONE | `Service.Move` and locked `Store.Replace`; lifecycle and CLI tests verify authoritative results and stale rejection. |
| 3 | Same transition replay is idempotent | DONE | `Service.Move` returns the existing item before changing timestamps; `TestServiceLifecycleReplayAndListing` verifies stable revision and `updated_at`. |
| 4 | Same source key and payload returns original | DONE | `CreateOrGet` holds the store lock across lookup/create; store and service replay tests verify one stable item. |
| 5 | Reused key with different payload conflicts safely | DONE | `ErrIdempotencyConflict` maps to `idempotency_conflict`; tests verify the original prompt remains unchanged. |
| 6 | Missing project remains visible and cannot mislaunch | DONE | `RegistryResolver`, `ListedItem.Availability`, and `MissingProjectError`; tests cover missing and unbound targets with no current-directory fallback. |
| 7 | Resolvable launch returns exact prompt and target only | DONE | `Service.LaunchIntent` is side-effect free; service and CLI end-to-end tests verify prompt/path and unchanged revision/lifecycle. |
| 8 | Concurrent mutations cannot silently lose updates | DONE | `lock.go` uses an exclusive process lock and `store.go` uses compare-under-lock plus fsynced atomic rename; concurrent two-store test and race detector pass. |
| 9 | Lists are deterministic, filtered, and hide done by default | DONE | Store sorts by updated time then ID; service filters active/specific/all; service and CLI tests verify ordering and default done exclusion. |
| 10 | Focus does not mutate adjacent work systems | DONE | The package writes only the injected user state root; CLI/service paths contain no spec, Intake, tracker, todo, due-date, assignment, estimate, or priority integration. Full repository tests pass. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Add private atomic Focus store, lock, and tests | DONE | Added `internal/attention/focus/store.go`, `lock.go`, and `store_test.go` covering permissions, canonical revisions, atomic replacement, traversal rejection, source replay, and concurrency. |
| 2 | Add service validation, lifecycle, provenance, resolution, launch | DONE | Added `service.go` and `service_test.go` with boundary validation and full lifecycle/idempotency/launch coverage. |
| 3 | Add registry and peering identity adapter | DONE | Added `project.go` and `project_test.go`; resolution checks global registry and configured peers by canonical `peer_id`, auto-resolves the registered current workspace, and refreshes presentation slug without exposing registry storage paths in portable contracts. |
| 4 | Add and register complete Focus CLI | DONE | Added `internal/cli/focus.go`, `focus_test.go`, and root registration for add/list/show/move/done/launch, file/stdin prompts, human output, and stable JSON. |
| 5 | Add command/reference documentation and distinguish Focus | DONE | Added `web/docs/src/cli/focus.md` and registered it in `web/docs/mkdocs.yml`, with command examples, auto-binding, privacy/revision/launch semantics, and explicit separation from specs, Intake, trackers, and harness todos. |
| 6 | Test source-derived CreateOrGet seam | DONE | Store and service tests cover required typed provenance/key, exact replay, conflicting payload, and original-item preservation. |

### Exercise-the-feature check

- [x] User-visible behavior was exercised end-to-end: `go test ./internal/cli -run '^TestFocusCLI'` added an exact multiline stdin prompt, listed/shown/moved/launched/completed it through Cobra, verified stale revision handling and safe unbound failure, and observed stable JSON results.

### Excellence Bar self-check

- [x] Honest answer to "would a senior engineer who cares about this codebase be proud to ship this?" — yes: the implementation preserves the completed public contract, closes the optional-project gap locally, uses durable cross-process compare-and-replace, validates trust boundaries, and is covered through CLI, race, vet, and full-suite checks.
