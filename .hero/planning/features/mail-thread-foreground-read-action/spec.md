---
title: Mail Thread Foreground Read Action
slug: mail-thread-foreground-read-action
type: feature
status: delivering
created: 2026-08-22
domain: engineering
size: medium
priority: high
surface: hero-core
relates-to:
  - mail-da2727fd11615a9cafa5125c
  - mail-b7ca19966ac5041e6ff604dd
tags: [mail, lifecycle, receipts, contracts]
delivery_method: manual
---
# Mail Thread Foreground Read Action

## Goal

Expose one revisioned, idempotent thread-level `mark_read` action so a desktop
foreground selection can mark exactly the messages returned by its successful
authoritative detail read without treating a plain read as a mutation or
exchanging per-message receipt revisions with thread revisions.

## Background

The published thread projection carries authoritative thread identity, read
summary, and lifecycle actions, but the public thread action route currently
rejects the receipt actions already named by the portable contract. Hero Code's
consumer spec requires one explicit foreground-read mutation after detail
success. Replaying one message action per envelope would not be one thread
operation and could incorrectly swallow a newer reply.

## Design

- Advertise `mark_read` only while a projected thread has unread messages and
  `mark_unread` only while it is fully read, alongside independent lifecycle
  actions.
- Dispatch through the thread action route with the advertised thread revision
  and stable idempotency key. Mail-read v1 message actions continue to use
  receipt revisions; the two revision domains are never substituted.
- Snapshot current message IDs before the state CAS, record the semantic action
  once, and apply the receipt change only to that snapshot. A concurrent inbound
  message remains unread.
- Plain thread list/detail reads remain non-mutating.

## Changes

1. Extend thread action validation and descriptors with executable thread-level
   `mark_read` / `mark_unread` actions.
2. Add replay-safe Mail service receipt reconciliation for the captured thread
   message snapshot.
3. Publish the actions through thread summaries/details and the existing HTTP
   thread action endpoint without changing Mail-read v1.
4. Add contract, service, query, and HTTP regression coverage.

## Acceptance Criteria

- **AC-1:** WHEN a thread detail is unread THE SYSTEM SHALL advertise one
  revisioned thread `mark_read` action independently from lifecycle actions.
- **AC-2:** WHEN the current action and a new idempotency key are submitted THE
  SYSTEM SHALL mark the captured thread messages read exactly once and return
  authoritative thread state.
- **AC-3:** IF an inbound message arrives after the action snapshot THEN THE
  SYSTEM SHALL leave that newer message unread.
- **AC-4:** IF the request is stale, conflicting, unknown, or malformed THEN THE
  SYSTEM SHALL reject it without replaying or applying a different action.
- **AC-5:** WHEN a thread is fully read THE SYSTEM SHALL advertise `mark_unread`,
  and invoking it SHALL change only receipt state plus semantic action history,
  not lifecycle classification.
- **AC-6:** THE SYSTEM SHALL preserve Mail-read v1 request, response, revision,
  and route compatibility, and plain list/detail reads SHALL remain inert.

## Boundaries

- No desktop UI or local receipt cache.
- No content inference, automatic read on GET, or bulk action across identities.
- No physical deletion or lifecycle clock changes.

## Validation

- Contract closed-world action tests.
- Service tests for exact replay, stale/conflict, partial/concurrent inbound
  behavior, mark-unread, and lifecycle independence.
- Query and HTTP tests for advertised action/revision parity and inert GETs.
- Race-enabled focused suites, full Go tests, vet, and diff check.

## Kickoff

Add the service-level foreground-read regression first, then expose the exact
descriptor through the existing thread projection and HTTP action route.

## Completion Ledger

Validation performed:

- `go test -race ./contracts/attention/mailthread ./internal/attention/mail ./internal/attention/mailquery ./internal/attention/projection ./internal/serve`
- `go test ./...`
- `go vet ./...`
- `git diff --check`

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Unread detail advertises revisioned thread mark-read | DONE | `ThreadCapabilities` composes the receipt action from authoritative `ReadSummary` with the current thread revision; service and HTTP tests assert descriptor/revision parity. |
| 2 | Current action marks captured messages once | DONE | `ThreadAction` performs the thread CAS once, records the exact message snapshot, and reconciles each receipt idempotently; exact replay returns the same thread state. |
| 3 | Later inbound message remains unread | DONE | The replay regression delivers a later envelope after the initial action and proves replay uses stored `PriorMessageIDs`, leaving the new envelope without a read receipt. |
| 4 | Stale, conflicting, unknown, malformed fail closed | DONE | Tests cover stale-without-receipt, same-key/different-action conflict, unknown action rejection, and forbidden receipt-action input. |
| 5 | Fully read advertises mark-unread orthogonally | DONE | The service test round-trips mark-read to mark-unread and proves lifecycle remains open while only read summary/action history changes. |
| 6 | Mail-read v1 and inert GETs remain unchanged | DONE | Existing v1 routes and contracts are untouched; the HTTP test proves thread list/detail GETs create no receipt before the explicit thread action. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Executable thread receipt actions | DONE | Closed validation now accepts only the six canonical thread actions, and descriptors expose exactly the receipt action appropriate to current read summary. |
| 2 | Replay-safe receipt reconciliation | DONE | Mark-read/unread operate on the action's captured message IDs with stable per-message idempotency keys. |
| 3 | Projection and HTTP publication | DONE | Existing source-owned summaries/details inherit the authoritative descriptors, and the unchanged thread action endpoint executes them. |
| 4 | Contract, service, query, HTTP regressions | DONE | Focused race, full repository, vet, and compatibility suites pass unattended. |

### Exercise-the-feature check

- [x] The automated HTTP flow performed inert list/detail reads, executed one
  foreground thread mark-read, observed authoritative mark-unread state, then
  archived the same exact thread. Service coverage replayed the read after a
  later inbound message and proved that new content remained unread.

### Excellence Bar self-check

- [x] Yes — one source owns identity, action history, receipts, and lifecycle;
  executable actions are advertised and revisioned; replay scope is durable;
  and no read endpoint, content inference, client clock, or manual test gate was
  introduced.
