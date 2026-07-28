---
title: "GitHub pull-request state-transition broker"
slug: github-pull-request-state-transition-broker
type: feature
status: planning
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

**Status:** planning — lifecycle fields, observation revisions, GitHub
REST/GraphQL routes, and shared mutation state are mapped.

**Pick up at:** define operation-specific desired-state comparators, then wire
preflight and one provider transition per operation.

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
