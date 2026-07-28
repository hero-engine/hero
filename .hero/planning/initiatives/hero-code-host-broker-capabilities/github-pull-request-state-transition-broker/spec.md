---
title: "GitHub pull-request state-transition broker"
slug: github-pull-request-state-transition-broker
type: feature
status: delivering
domain: engineering
priority: high
size: medium
created: 2026-07-27
parent: hero-code-host-broker-capabilities
depends-on:
  - github-pull-request-collaboration-broker
conflicts-with:
  - github-pull-request-collaboration-broker
  - github-pull-request-merge-broker
tags: [github, pull-request, lifecycle, retarget, idempotency, stale-state]
delivery_method: manual
---

# GitHub pull-request state-transition broker

## Context

Mark-ready, retarget, close, and reopen mutate a PR's lifecycle without merging
code. Their desired states are directly observable, making externally
completed and ambiguous-response reconciliation stronger than body-based
collaboration matching. They still require fresh state because a retarget or
close based on an old head/base can invalidate reviews, checks, and downstream
workflow.

## Goal

Implement guarded `mark_ready`, `retarget`, `close`, and `reopen` operations
with fresh lifecycle evidence, exact desired-state reconciliation, stable
idempotency, typed effects, permissions, cancellation, and partial/ambiguous
outcomes.

## Kickoff

Add idempotent PR lifecycle transitions while keeping them distinct from
review collaboration and merge commitment.

**Status:** in review — all four lifecycle transitions, desired-state
reconciliation, and deterministic provider hazards are implemented and
validated.

**Pick up at:** cold-audit the completed delivery and verify the spec.

→ `/deliver github-pull-request-state-transition-broker`

**Files:** `internal/codehost/github_state.go`, `internal/codehost/reconcile.go`, `internal/mockcodehost/state.go`
**Skip:** do not merge, enable auto-merge, update branches, or complete tracker/spec work.

## Problem

A generic “update PR” operation would hide materially different permissions and
stale-state rules. Retarget changes the base branch and can invalidate merge
readiness; mark-ready is one-way for this contract; close/reopen race with
external actors and merge. Treating “already in desired state” as failure
causes unsafe retries, while treating an old observation as permission to write
can overwrite a newer decision.

## Approach

Expose four explicit operations, each classified as `external_write` requiring
`explicit_user`. Require repository-qualified PR identity, current head SHA,
capability revision, lifecycle observation revision, user intent, stable
idempotency key, and operation-specific desired state. Retarget also requires
the current base ref and explicit new base ref.

Preflight reads authoritative PR lifecycle state, target refs, permissions, and
operation availability. If the provider is already in the exact desired state
and the target is otherwise valid, record `externally_completed` without a
write. If lifecycle/head/base changed incompatibly, return
`stale_observation` or `conflict` and require refresh.

Use GitHub's typed REST state/base updates and GraphQL ready-for-review mutation
behind the existing journal. Mark the journal dispatched before the provider
call. After ambiguous response, read the exact PR and compare only the
operation's desired state: draft false for mark-ready, exact base repository/ref
for retarget, closed-unmerged for close, or open for reopen. Preserve merged as
a distinct terminal state; close/reopen never rewrites it.

Successful or reconciled transitions return the new observation revision and
invalidate readiness-sensitive cached observations. Same-key retries return the
stored result; changed targets conflict.

## Changes

1. Add operation-specific state-transition request validation and desired-state
   comparators.
2. Add preflight for current lifecycle, draft, merge state, head SHA, base
   repository/ref, target ref existence, permissions, and capability revision.
3. Implement typed GitHub mark-ready, retarget, close, and reopen routes.
4. Extend the shared journal and reconciliation state machine with
   desired-state read-back and new observation revisions.
5. Make merged, closed-unmerged, open-draft, and open-ready distinct normalized
   lifecycle states.
6. Extend the fake with external transitions, races, missing branches,
   permission changes, delayed visibility, force pushes, merged PRs, and lost
   responses.
7. Add effect, permission, idempotency, stale-state, cancellation, receipt,
   readiness invalidation, and conformance tests.

## Acceptance Criteria

- **AC-1:** THE SYSTEM SHALL expose `mark_ready`, `retarget`, `close`, and `reopen` as separate `external_write` operations requiring `explicit_user`.
- **AC-2:** THE SYSTEM SHALL require repository-qualified PR identity, current head SHA, user intent, idempotency key, capability revision, and lifecycle observation revision in every transition request.
- **AC-3:** WHEN retarget is requested THE SYSTEM SHALL additionally require the observed current base and an explicit existing new base within the configured repository scope.
- **AC-4:** IF permissions, capability revision, head SHA, lifecycle, or observed base changed incompatibly THEN THE SYSTEM SHALL return a typed stale/conflict error before provider dispatch.
- **AC-5:** WHEN the PR is already in the exact requested state THE SYSTEM SHALL return `externally_completed` and perform no provider write.
- **AC-6:** WHEN a provider response is lost after a transition THE SYSTEM SHALL read the exact PR, compare the operation-specific desired state, and return `reconciled_applied`, `not_applied`, or `ambiguous`.
- **AC-7:** WHEN a PR is merged THE SYSTEM SHALL NOT close, reopen, retarget, or mark it ready and SHALL preserve merged as a distinct terminal result.
- **AC-8:** WHEN the same key and canonical request are retried THE SYSTEM SHALL perform at most one provider transition and return the same receipt/reconciliation result.
- **AC-9:** IF the same key is reused with another operation, PR, head, current base, or desired target THEN THE SYSTEM SHALL return `idempotency_conflict`.
- **AC-10:** WHEN a transition succeeds or reconciles THE SYSTEM SHALL return a new observation revision and mark prior readiness observations stale.
- **AC-11:** WHEN cancellation occurs before dispatch THE SYSTEM SHALL perform no write; cancellation after dispatch SHALL enter desired-state reconciliation.
- **AC-12:** THE SYSTEM SHALL fake- and fixture-test external completion, stale head/base, missing target branch, permission race, delayed visibility, force push, merged terminal state, duplicate retry, and lost response.

## Boundaries

- No merge, merge queue, auto-merge, update-branch, branch delete, branch push,
  tracker transition, spec completion, or Hero Code cache mutation.
- No generic patch operation accepting arbitrary GitHub fields.
- No draft conversion from ready back to draft in v1.
- No implicit default-base retargeting.

## Risks

- Retarget can invalidate provider review/check state asynchronously. Return the
  new observation and require consumers to refresh rather than predicting it.
- “Already closed” may mean merged. Lifecycle comparison must never collapse
  those states.
- A target branch can move between preflight and update. Provider conflicts and
  returned base/head evidence must remain visible.
- GraphQL and REST transition responses differ. Normalize through authoritative
  post-state, not response-shape convenience.

## Validation

- Table-test all source/desired lifecycle combinations and invalid transitions.
- Exercise stale base/head, force push, branch movement, external completion,
  permission race, duplicate, cancellation, and lost response through the fake.
- Verify new revisions invalidate prior readiness evidence.
- Run code-host/fake tests, then `go test ./...` and `go vet ./...`.

## Completion Ledger

Implemented on the existing provider-neutral broker, GitHub transport,
credential resolver, durable mutation journal, exact reconciliation state
machine, and deterministic provider fake. Validation included focused lifecycle
and regression tests, race detection, the full repository suite, full `go vet`,
formatting, and diff hygiene.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Expose four explicit external-write transitions | DONE | The capability registry advertises `mark_ready`, `retarget`, `close`, and `reopen` separately with authoritative `external_write` and `explicit_user` policies. |
| 2 | Require qualified PR, head, intent, idempotency, and revisions | DONE | V1 validation, strict operation payload decoding, and `PrepareStateTransition` require every safety field and provider-backed lifecycle evidence before dispatch. |
| 3 | Retarget requires observed current and explicit existing new base | DONE | Scope validation binds both refs to the PR repository; preflight exact-matches the current base and resolves the named target branch at its supplied SHA. |
| 4 | Reject incompatible permission, revision, head, lifecycle, and base changes | DONE | Force pushes, base changes, target movement, permission races, capability drift, observation drift, invalid lifecycle sources, and missing branches return typed errors with zero writes. |
| 5 | Exact desired state returns external completion | DONE | Operation-specific comparators recognize ready, repository/ref/SHA-exact retarget, closed-unmerged, and reopened states and return `externally_completed` with no provider transition. |
| 6 | Lost responses reconcile from authoritative desired state | DONE | Detached bounded read-back returns `reconciled_applied` only for the exact desired state, including retarget target SHA; delayed, moved-target, or unavailable proof remains `ambiguous`, and definite provider rejection remains `not_applied`. |
| 7 | Preserve merged as an immutable terminal state | DONE | Normalization distinguishes merged from closed-unmerged, and every transition rejects a merged PR before dispatch and during reconciliation. |
| 8 | Same-key retries perform at most one provider transition | DONE | Sequential and concurrent retries for all four operations share the existing durable connection-scoped journal, file lock, receipt, and one provider attempt. |
| 9 | Changed operation, PR, head, base, or desired target conflicts | DONE | The canonical lifecycle payload digest binds every required dimension beneath the stable idempotency key and returns `idempotency_conflict` before another write. |
| 10 | Success returns a new observation and invalidates prior readiness evidence | DONE | Post-transition results return a new lifecycle observation plus typed `invalidated_operations: [get_merge_readiness]`; the v1 validator and canonical consumer fixture preserve that provider-neutral invalidation signal. |
| 11 | Respect pre- and post-dispatch cancellation boundaries | DONE | Pre-dispatch cancellation records `not_applied` with zero writes; cancellation after application enters exact desired-state reconciliation for all four operations. |
| 12 | Exercise required transition hazards through the fake | DONE | `internal/mockcodehost` covers external completion, stale head/base, missing and moving branches, permission races, delayed visibility, force pushes, merged terminal state, duplicate retry, provider denial, cancellation, and lost response. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Add validation and desired-state comparators | DONE | `github_state.go` strictly decodes lifecycle payloads and models open-draft, open-ready, closed-unmerged, and merged source/desired combinations, with repository/ref/SHA-exact retarget matching. |
| 2 | Add lifecycle and target preflight | DONE | Common preflight verifies identity, head, lifecycle, current base, target ref/SHA, actor, permission, capability, and observation revisions. |
| 3 | Implement typed GitHub transition routes | DONE | Mark-ready uses the typed GraphQL mutation; retarget, close, and reopen use bounded REST patches, all followed by authoritative PR read-back. |
| 4 | Extend journal and desired-state reconciliation | DONE | Safe base/state material extends the existing locked journal and shared dispatch/retry/ambiguous-response state machine; post-write target movement remains ambiguous, and no provider bodies or credentials are stored. |
| 5 | Distinguish normalized lifecycle states | DONE | `normalizedLifecycleState` keeps merged, closed-unmerged, open-draft, and open-ready separate and comparators reject unknown or invalid sources. |
| 6 | Extend the deterministic GitHub fake | DONE | Mutable fake state models typed routes, branch resolution/movement, permissions, delayed visibility, merged state, lost/cancelled responses, and attempt accounting. |
| 7 | Add effect, stale-state, receipt, and conformance tests | DONE | `github_state_test.go` covers policies, SHA-exact state matrices, pre/post-write branch movement, typed readiness invalidation, preflight gates, external completion, concurrency, conflicts, cancellation, recovery, journal safety, and provider rejection. |

### Exercise-the-feature check

- [x] All four transitions were exercised end-to-end through the deterministic GitHub HTTP fake with `go test ./internal/codehost -run '^TestStateTransition' -count=1 -v`; focused packages, race detection, `go test ./... -count=1`, and `go vet ./...` also passed.

### Excellence Bar self-check

- [x] Yes — the implementation is fail-closed, provider-neutral at the public boundary, credential-safe, exact-state- and head-bound, merged-aware, crash-aware, bounded, and built by extending Hero's existing transport, revisions, journal, lock, and fake paths.
