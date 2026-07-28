---
title: "GitHub pull-request merge broker"
slug: github-pull-request-merge-broker
type: feature
status: delivering
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
delivery_method: manual
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

**Status:** in review — guarded direct merge, exact base-branch queue policy,
runtime capability material, and journal reconciliation are implemented and
validated.

**Pick up at:** run the cold delivery audit against the committed
implementation, remediate any hold, then verify the spec.

→ `/deliver github-pull-request-merge-broker`

**Files:** `internal/codehost/github_merge.go`, `internal/codehost/github.go`, `internal/codehost/reconcile.go`, `internal/mockcodehost/server.go`, `contracts/codehostbroker/`
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
force push races safely. For repositories requiring a queue, advertise merge
unavailable unless the adapter implements an exact provider queue mutation and
receipt/read-back contract. An adapter that does implement and advertise queue
support returns `queued` with provider receipt and queue identity; queued is
never reported as merged. GitHub queue submission is intentionally unavailable
in this v1 child until that proof exists.

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
5. Advertise queue submission unavailable and fail closed for queue-required
   repositories until the adapter can prove and reconcile exact support.
6. Extend journal receipts and reconciliation with expected head/base and
   merge commit identity; require queue identity and a distinct `queued` state
   if a future adapter advertises queue support.
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
- **AC-6:** WHEN repository policy requires a merge queue AND the adapter does not advertise `queue_supported` THE SYSTEM SHALL advertise merge unavailable and perform no direct merge effect; IF a future adapter advertises queue support THEN THE SYSTEM SHALL return a typed `queued` result with an exact queue receipt and SHALL NOT report it as merged.
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
- Exercise all direct merge methods, queue-only fail-closed flow, force-push race, changed
  protection, stale review/check, external completion, conflicting head,
  duplicate, cancellation, rate limit, and lost response through the fake.
- Verify merge service policy rejects missing explicit acceptance on every
  transport path.
- Run code-host/fake tests, then `go test ./...` and `go vet ./...`.

## Completion Ledger

### Acceptance Criteria

| # | Acceptance criterion | Status | Evidence |
|---|---|---|---|
| 1 | Advertise only current permitted merge methods and queue behavior | DONE | `mergeRuntimeCapability` combines GitHub repository method/permission metadata with the exact base branch's GraphQL merge queue and binds it into `MergeCapability.revision`; unavailable policy fails closed. |
| 2 | Require qualified identity, head/base, method, intent, keys, revisions, and acceptance | DONE | The contract validator and strict merge payload/scope decoder require every field before provider access. |
| 3 | Enforce commitment plus explicit acceptance inside the broker | DONE | The authoritative policy registry classifies `merge` as `commitment`/`explicit_acceptance`; `ValidateRequest` rejects weaker consent before adapter resolution. |
| 4 | Dispatch nothing when required evidence is stale, partial, unknown, or contradictory | DONE | Merge preflight requires complete/current PR, head/base, checks, reviews, protection, permission, mergeability, queue, method, and capability evidence; focused tests cover every fail-closed state. |
| 5 | Supply expected head SHA and fail safely on force-push races | DONE | Direct merge PUT includes `sha`; preflight and provider-race tests prove stale heads perform no accepted merge. |
| 6 | Fail closed for unsupported queue submission | DONE | `queue_supported:false`; exact base-branch queue requirement makes merge unavailable with no direct merge write. Future queue-capable adapters remain contractually required to return a distinct queued receipt. |
| 7 | Recognize exact externally completed merges without another write | DONE | Preflight proves merged lifecycle, expected head/base, and exact merge commit identity before returning `externally_completed`; conflicting heads fail. |
| 8 | Reconcile lost responses into proven applied, conflict, or ambiguous state | DONE | Detached read-back proves exact merged head/base/commit identity; unavailable evidence remains `ambiguous`. |
| 9 | Never describe a different-head merge as the requested effect | DONE | Preflight and reconciliation normalize different head/base evidence to conflict and return no applied receipt. |
| 10 | Retry the same canonical key at most once | DONE | The durable mutation journal replays/reconciles before new runtime capability access; duplicate and post-permission-change retries keep one provider merge attempt. |
| 11 | Reject key reuse with changed merge material | DONE | Canonical digest binds PR, head/base, method, title/message, intent, consent, and reconciliation key; mutation tests assert `idempotency_conflict`. |
| 12 | Honor cancellation boundaries | DONE | Pre-dispatch cancellation performs no merge; post-dispatch cancellation enters detached reconciliation and proves the applied effect when possible. |
| 13 | Exclude adjacent branch, tracker, deployment, and auto-merge effects | DONE | The adapter exposes only the PR merge endpoint; no update/delete/auto-merge/tracker/deployment code path was added. |
| 14 | Fake- and fixture-test declared hazards | DONE | The fake covers permissions, methods, queue policy, rate limits, force pushes, partial/stale readiness, protection, duplicates, external completion, conflicting heads, cancellation, and ambiguity. |

### Changes

| # | Planned change | Status | Evidence |
|---|---|---|---|
| 1 | Runtime merge capability material | DONE | Added validated `MergeCapability` methods, queue flags, and revision to the canonical contract and fixture. |
| 2 | Fail-closed merge proof | DONE | Added exact preflight across runtime policy and complete merge-readiness evidence. |
| 3 | Explicit acceptance | DONE | Enforced through the authoritative operation policy before provider access. |
| 4 | Direct GitHub merge | DONE | Added merge/squash/rebase PUT with expected-head protection and bounded normalized results. |
| 5 | Queue unsupported boundary | DONE | Exact GraphQL queue discovery advertises unavailable and performs zero direct writes for queue-required bases. |
| 6 | Durable receipts and reconciliation | DONE | Journal receipts retain exact head/base and merge commit identity; queue receipt/state stays conditional on future advertised support. |
| 7 | Deterministic fake | DONE | Extended `internal/mockcodehost` with merge methods, policy/races, lost responses, external effects, and cancellation. |
| 8 | Contract, broker, and safety tests | DONE | Focused, race, full-suite, vet, fixture generation, and diff checks pass. |

### Exercise-the-feature check

- [x] Prepared and executed all three advertised direct methods against the
  fake and confirmed exact head SHA, selected method, optional commit text,
  one provider write, typed actor/receipt, and readiness invalidations.
- [x] Exercised queue-required and partially unavailable queue policy and
  confirmed unavailable capability/preflight with zero direct merge writes.
- [x] Exercised force pushes, readiness/protection/permission races, duplicate
  retries, lost responses, external completion, conflicting heads,
  cancellation, rate limits, and ambiguous read-back.
- [x] Regenerated the canonical consumer fixture at SHA-256
  `c8918b94c9b8debefd933eff5f53ca721504a27b95a4134e53dc8fc81252e987`.
- [x] Passed `go test ./... -count=1`, `go vet ./...`, and
  `go test -race ./internal/codehost ./internal/mockcodehost ./contracts/codehostbroker`.

### Excellence Bar self-check

Yes. Merge is implemented as a narrow provider boundary over the existing
credential resolver and mutation journal. It uses real GitHub base-branch
queue evidence, fails closed on incomplete truth, emits only bounded typed
results, and adds no queue guess, credential path, tracker coupling, or
adjacent automation.

### Design amendment

The original queue change assumed exact queue submission was already
available. Delivery proved that the v1 adapter did not yet have a sufficient
queue receipt/read-back contract, so AC-6 and Changes 5–6 were narrowed:
queue-required bases are now explicitly unavailable with zero direct merge
effect; typed queued state remains mandatory if a future adapter advertises
queue support.
