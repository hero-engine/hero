---
title: "GitHub pull-request merge broker"
slug: github-pull-request-merge-broker
type: feature
status: planning
domain: engineering
priority: high
size: medium
created: 2026-07-27
parent: hero-code-host-broker-capabilities
depends-on:
  - github-pull-request-state-transition-broker
conflicts-with:
  - github-pull-request-state-transition-broker
tags: [github, pull-request, merge, commitment, readiness, reconciliation]
---

# GitHub pull-request merge broker

## Context

Merge is the highest-risk PR effect: it changes the base branch and can trigger
deployments, releases, and downstream automation. GitHub may support merge,
squash, rebase, or a merge queue depending on repository policy, and its
mergeability/check/review signals can be pending or stale. The read adapter and
state-transition broker provide the exact identities, readiness evidence,
revisions, and mutation journal required to fail closed.

## Goal

Implement a provider-capability-aware `merge` operation classified as a
commitment, guarded by explicit acceptance, current head/base and readiness
evidence, supported merge method, provider-side SHA protection, stable
idempotency, and exact merged/queued/ambiguous reconciliation.

## Kickoff

Add the final GitHub write only when Hero can prove the requested PR revision
is mergeable under current repository policy and can reconcile every retry.

**Status:** planning — readiness, merge methods, queue state, SHA guards, and
mutation reconciliation requirements are mapped.

**Pick up at:** define the merge preflight proof and provider result states,
then implement one guarded dispatch using the existing journal.

→ `/deliver github-pull-request-merge-broker`

**Files:** `internal/codehost/github_merge.go`, `internal/codehost/readiness.go`, `internal/mockcodehost/merge.go`
**Skip:** do not auto-merge later, update branches, delete branches, deploy, release, or close Hero work.

## Problem

A boolean `mergeable` is insufficient: checks may be pending, reviews may be
dismissed by a force push, branch protection can change, permission can be
lost, and merge queues distinguish acceptance from completion. Retrying a
timed-out merge without checking provider state can act on a new head or return
a misleading failure after the PR already merged.

## Approach

Advertise merge methods and queue support as runtime capabilities per selected
repository. A request names the repository-qualified PR, exact expected head
SHA, observed base repository/ref/SHA, requested supported method, optional
bounded commit title/message as allowed by the method, capability revision,
fresh readiness observation revision, `intent_source: user`, stable
idempotency key, and explicit acceptance evidence.

Merge is `commitment` plus `explicit_acceptance`; service validation enforces
that policy regardless of CLI/MCP annotations. Preflight re-reads permissions,
open/ready state, current head/base, draft state, checks, reviews, branch
protection, mergeability, queue requirements, and method support. Any unknown,
unavailable, partial, pending, stale, or contradictory required evidence fails
closed without dispatch.

For direct merge, pass the expected head SHA to GitHub's merge endpoint so a
force push races safely. For repositories requiring a queue, use the supported
provider queue mutation and return `queued` with provider receipt and queue
identity; never report queued as merged.

After any ambiguous response, re-read the exact PR. If it is merged with the
expected head and base evidence, return `reconciled_applied`. If it is queued
with the exact provider queue/operation identity, return `reconciled_applied`
with queued state. If an external actor already merged the same expected head
before dispatch, return `externally_completed`. A merge of another head is a
conflict, never success. Same-key retries reconcile from the journal and never
blindly merge again.

## Changes

1. Extend runtime capabilities with repository-specific merge methods, queue
   support, input bounds, and capability revision inputs.
2. Add a fail-closed merge proof over permissions, lifecycle, draft, head/base,
   checks, reviews, protection, mergeability, queue, and requested method.
3. Add explicit-acceptance validation at the broker boundary.
4. Implement direct GitHub merge with expected-head SHA and supported
   merge/squash/rebase method.
5. Implement provider queue submission only where the adapter can prove and
   reconcile support.
6. Extend journal receipts and reconciliation with expected head/base, merge
   commit identity, queue identity, and distinct `merged`/`queued` states.
7. Extend the fake with method restrictions, queue-only repositories,
   force-push races, stale checks/reviews, changed protection, lost responses,
   externally completed merges, and conflicting-head merges.
8. Add policy, readiness, permission, idempotency, cancellation, rate-limit,
   partial-state, and conformance tests.

## Acceptance Criteria

- **AC-1:** THE SYSTEM SHALL advertise only merge methods and queue behavior implemented and permitted for the selected repository at the current capability revision.
- **AC-2:** THE SYSTEM SHALL require repository-qualified PR identity, exact head SHA, observed base identity/SHA, supported method, user intent, idempotency key, capability revision, readiness observation revision, and explicit acceptance in every merge request.
- **AC-3:** THE SYSTEM SHALL classify merge as `commitment` requiring `explicit_acceptance` and enforce that classification inside the broker independently of transport metadata.
- **AC-4:** IF permissions, lifecycle, draft state, head/base, checks, reviews, protection, mergeability, queue policy, or method support is stale, unknown, unavailable, partial, pending, or contradictory THEN THE SYSTEM SHALL perform no merge effect.
- **AC-5:** WHEN direct merge is dispatched THE SYSTEM SHALL supply the expected head SHA to the provider and fail safely if a force push wins the race.
- **AC-6:** WHEN repository policy requires a merge queue THE SYSTEM SHALL return a typed `queued` result with queue receipt and SHALL NOT report the PR as merged until authoritative state says merged.
- **AC-7:** WHEN the exact expected head was already merged before dispatch THE SYSTEM SHALL return `externally_completed` without another provider write.
- **AC-8:** WHEN a response is lost after merge or queue acceptance THE SYSTEM SHALL reconcile the exact expected head/base and provider receipt into `reconciled_applied`, conflict, or `ambiguous`.
- **AC-9:** WHEN the provider shows a merge of a different head THE SYSTEM SHALL return conflict and SHALL NOT describe the requested merge as applied.
- **AC-10:** WHEN the same key and canonical request are retried THE SYSTEM SHALL perform at most one merge/queue submission and return the same receipt/reconciliation result.
- **AC-11:** IF the same key is reused with a different PR, head/base, method, message digest, or acceptance material THEN THE SYSTEM SHALL return `idempotency_conflict`.
- **AC-12:** WHEN cancellation occurs before dispatch THE SYSTEM SHALL perform no effect; cancellation after dispatch SHALL enter merge/queue reconciliation.
- **AC-13:** THE SYSTEM SHALL not delete branches, update branches, enable future auto-merge, transition trackers/specs, or trigger deployments beyond provider-native consequences of the explicitly accepted merge.
- **AC-14:** THE SYSTEM SHALL fake- and fixture-test permissions, supported methods, queue-only policy, rate limits, force pushes, stale checks/reviews, protection changes, duplicates, external completion, conflicting heads, and ambiguous responses.

## Boundaries

- No automatic/future merge scheduler, update branch, conflict resolution,
  branch deletion, deployment, release, tracker transition, or spec completion.
- No override of required checks, reviews, branch protection, or provider queue.
- No optimistic merge when readiness is unknown or partially unavailable.
- No GitLab merge implementation in v1.

## Risks

- Merge queue APIs and permissions vary by GitHub deployment. Capability
  discovery must distinguish unsupported from temporarily unavailable.
- Mergeability is eventually computed. A pending result is not permission to
  poll without bounds or merge optimistically.
- Branch protection can change between preflight and dispatch. Provider-side
  enforcement remains authoritative and conflicts must be normalized.
- A merged PR may not expose enough evidence to prove which request caused it.
  If expected head/base cannot be proven, return ambiguous/conflict.

## Validation

- Build a readiness truth table covering every required evidence state and
  prove only the complete/current combination dispatches.
- Exercise all merge methods, queue-only flow, force-push race, changed
  protection, stale review/check, external completion, conflicting head,
  duplicate, cancellation, rate limit, and lost response through the fake.
- Verify merge service policy rejects missing explicit acceptance on every
  transport path.
- Run code-host/fake tests, then `go test ./...` and `go vet ./...`.
