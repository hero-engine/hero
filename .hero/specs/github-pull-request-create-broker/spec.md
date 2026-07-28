---
title: "GitHub pull-request creation broker"
slug: github-pull-request-create-broker
type: feature
status: completed
domain: engineering
priority: high
size: medium
created: 2026-07-27
parent: hero-code-host-broker-capabilities
depends-on:
  - github-code-host-read-adapter
conflicts-with:
  - github-code-host-read-adapter
  - github-pull-request-collaboration-broker
tags: [github, pull-request, create, idempotency, reconciliation]
delivery_method: manual
completed_at: 2026-07-28T02:24:46Z
---

# GitHub pull-request creation broker

## Context

Hero Code currently has a narrow direct `gh`-based PR creation path. The read
adapter provides repository, branch, existing-PR, permission, and freshness
evidence, but GitHub's create API has no native idempotency key. A timeout after
dispatch can therefore create a PR even when the client receives no response.

## Goal

Add credential-safe `create_pull_request` to the in-process broker with exact
repository/head/base targeting, explicit user intent, durable duplicate
suppression, provider read-back reconciliation, and typed ambiguous outcomes.

## Kickoff

Adds one guarded Hero-owned GitHub PR creation path with durable duplicate
suppression and provider read-back after uncertain outcomes.

**Status:** in-review — implementation, deterministic provider scenarios,
stable-key regressions, full tests, race detection, and vet are green.

**Pick up at:** cold-audit the committed creation broker against all 12
criteria, then run `hero spec verify github-pull-request-create-broker`.

→ `.hero/planning/initiatives/hero-code-host-broker-capabilities/github-pull-request-create-broker/spec.md`

**Files:** `internal/codehost/github_create.go`, `internal/codehost/idempotency.go`, `internal/codehost/mutations.go`, `internal/codehost/github_create_test.go`
**Skip:** do not push commits, choose branches, invoke `gh`, or create tracker work.

## Problem

Creation has no PR identity before it succeeds, and matching only branch names
is unsafe for forks. Blind retry can create duplicates; treating every matching
open PR as the same request can attach the wrong work; storing full request
bodies in a journal leaks user content. Cancellation after dispatch and an
external actor creating the intended PR both require reconciliation rather
than a second write.

## Approach

Require the selected base repository, owner-qualified head repository/ref,
base ref, title, bounded body, draft state, capability revision, preflight
observation revision, `intent_source: user`, and idempotency key. Hero validates
that the repositories are in configured scope, refs resolve, the head SHA is
current, the actor can create, and no exact open PR already links the same base
repository/ref and head repository/ref.

Add a private replay journal beneath
`.hero/cache/code-host-broker/v1/`. It uses the existing cross-platform file
locking and atomic-write patterns, owner-only permissions, bounded entry count
and age, and stores only operation identity, canonical target, payload digest,
state, safe provider receipt, and reconciliation timestamps. It never stores
credentials, title/body text, authorization, or raw provider responses.

The same key plus the same payload digest returns the recorded or reconciled
receipt. The same key with a different digest returns
`idempotency_conflict`. Only one caller may own an in-progress key.

Before the write, query for an exact repository-qualified head/base match. If
the intended PR already exists and satisfies the request, return
`externally_completed` without writing. Otherwise dispatch exactly one GitHub
create request. On timeout, cancellation after dispatch, decode failure, or
other ambiguous response, query the exact target again and record
`reconciled_applied`, `not_applied`, or `ambiguous`. A retry uses the same key
and resumes reconciliation; it never blindly dispatches again while outcome is
unknown.

## Changes

1. Add the durable mutation journal and bounded retention/locking behavior.
2. Add creation request validation and preflight observation generation.
3. Add exact existing-PR reconciliation keyed by base repository/ref plus head
   repository/ref, with provider IDs and head SHA retained.
4. Add one typed GitHub creation request and normalize its receipt into the v1
   identity/result contract.
5. Integrate cancellation boundaries before dispatch, during dispatch, and
   after an ambiguous response.
6. Extend `internal/mockcodehost` with permission denial, duplicate retry,
   concurrent retry, lost response, externally completed creation, fork head,
   stale head, and conflicting idempotency scenarios.
7. Add journal recovery, crash-window, redaction, retention, and conformance
   tests.

## Acceptance Criteria

- **AC-1:** WHEN creation is requested THE SYSTEM SHALL require explicit base repository/ref, head repository/ref, current head SHA, title, draft state, user intent, idempotency key, capability revision, and preflight observation revision.
- **AC-2:** WHEN the head repository differs from the base repository THE SYSTEM SHALL create and reconcile using the owner-qualified fork identity rather than branch name alone.
- **AC-3:** IF permissions, repository scope, refs, capability revision, or preflight observation are invalid or stale THEN THE SYSTEM SHALL return a typed error before dispatching a provider write.
- **AC-4:** WHEN the same idempotency key and payload are submitted concurrently or repeatedly THE SYSTEM SHALL perform at most one provider create attempt and return the same recorded/reconciled result.
- **AC-5:** IF the same idempotency key is reused with a different canonical payload THEN THE SYSTEM SHALL return `idempotency_conflict` and perform no provider write.
- **AC-6:** WHEN an exact matching PR already exists before dispatch THE SYSTEM SHALL return its repository-qualified identity as `externally_completed` without creating another PR.
- **AC-7:** WHEN GitHub applies creation but its response is lost or cancelled after dispatch THE SYSTEM SHALL read back the exact base/head target and return `reconciled_applied` when provable.
- **AC-8:** WHEN post-dispatch read-back cannot prove applied or not applied THE SYSTEM SHALL return `ambiguous`, preserve the journal state, and allow only same-key reconciliation.
- **AC-9:** WHEN cancellation occurs before dispatch THE SYSTEM SHALL create no provider effect and record `not_applied`; cancellation after dispatch SHALL follow ambiguous-response recovery.
- **AC-10:** THE SYSTEM SHALL store the journal with owner-only permissions and atomic locking while excluding credentials, authorization, title/body text, and raw provider responses.
- **AC-11:** WHEN journal retention limits are exceeded THE SYSTEM SHALL remove only terminal expired entries and SHALL preserve in-progress or ambiguous entries needed for safe reconciliation.
- **AC-12:** THE SYSTEM SHALL return the v1 effect/consent policy, typed receipt, observation revision, rate-limit metadata, bounds, and reconciliation outcome for every creation result.

## Boundaries

- No commit push, branch creation, automatic reviewer assignment, comment,
  review, state transition, merge, tracker mutation, or Hero Code UI.
- No `gh` subprocess or provider credential outside Hero.
- No request-body persistence for debugging or reconciliation.
- No assumption that delivery tracker and code-host connection are identical.

## Risks

- Exact matching can still encounter two externally created PRs. Return
  conflict/ambiguous rather than selecting one.
- A local journal alone cannot close every crash window. Provider read-back is
  mandatory before any same-key redispatch.
- Network cancellation does not prove provider cancellation. Dispatch state
  must be journaled before the request leaves Hero.
- Journal cleanup can destroy safety evidence. Retention policy must favor
  unresolved entries and remain bounded by explicit limits.

## Validation

- Exercise successful, duplicate, concurrent, conflicting-key, lost-response,
  externally completed, ambiguous, cancelled, stale-head, fork, and permission
  scenarios against the fake.
- Crash at each journal transition and verify safe recovery.
- Inspect journal and error output with credential and body canaries.
- Run code-host and file-lock tests, then `go test ./...` and `go vet ./...`.

## Completion Ledger

Implemented on the existing in-process broker, GitHub transport, connection
resolver, file lock, and durable atomic writer. Validation included focused
creation/fake/file-lock tests, stable-key regressions, focused race detection,
the full repository suite, full `go vet`, formatting, and diff hygiene.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Require complete target, intent, idempotency, revisions, and explicit draft | DONE | `decodeCreatePayload` requires the raw `draft` boolean; v1 validation and `PrepareCreatePullRequest` require the remaining target and policy material before mutation. |
| 2 | Preserve owner-qualified fork identity | DONE | Preflight, reconciliation, and create use distinct repository identities plus `owner:ref`; `TestCreateExternallyCompletedForkAndAmbiguousRecovery` exercises a configured fork. |
| 3 | Reject permission, scope, ref, capability, and observation failures before write | DONE | `TestCreateRejectsStalePermissionScopeAndCancellationBeforeDispatch` verifies typed failures and zero create attempts across the preflight gates. |
| 4 | Same key/payload performs at most one provider create | DONE | The connection-scoped key digest and existing cross-platform journal lock serialize ownership; duplicate, concurrent, canonical-identity, and rejected-provider retries remain one attempt. |
| 5 | Different payload under the same key returns `idempotency_conflict` | DONE | Canonical payload digests detect title and cross-repository target changes before provider write; the regression asserts only the first target is created. |
| 6 | Existing exact PR returns `externally_completed` | DONE | Exact base/head repository, ref, SHA, content, draft, and open state produce the qualified existing identity with zero create attempts. |
| 7 | Lost or cancelled applied write reconciles to `reconciled_applied` | DONE | Detached bounded read-back proves both malformed/lost response and post-apply cancellation scenarios without redispatch. |
| 8 | Unprovable result remains `ambiguous` and same-key-only | DONE | Ambiguous journal state persists with `ambiguous_result`; repeated calls reconcile and never issue another create. |
| 9 | Pre-dispatch cancellation records `not_applied`; post-dispatch reconciles | DONE | Tests exercise a cancelled context before any effect and a response cancellation after the fake records the effect. |
| 10 | Journal is private, atomic, locked, and content/credential-free | DONE | The journal uses 0700/0600 permissions, `filelock.Acquire`, `spec.AtomicWriteFile`, key/payload digests, safe target/receipt material, and content/token canary inspection. |
| 11 | Retention removes only expired terminal entries | DONE | Retention tests preserve year-old ambiguous/in-progress records while expiring only terminal records; unresolved capacity fails closed. |
| 12 | Return v1 policy, receipt, revisions, rate limits, bounds, and outcome | DONE | Every scenario passes `codehostbroker.ValidateResponse`; success asserts effect, consent, receipt, reconciliation, rate metadata, bounds, and journal count. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Add durable mutation journal and bounded retention/locking | DONE | `internal/codehost/idempotency.go` stores private bounded digest/safety material using existing lock and atomic-write primitives. |
| 2 | Add creation validation and preflight observation | DONE | `internal/codehost/mutations.go` and `PrepareCreatePullRequest` provide strict payload validation and provider-backed revisions. |
| 3 | Add exact existing-PR reconciliation | DONE | `findExactPullRequests` and reconciliation state handling retain qualified identity and current head evidence. |
| 4 | Add one typed GitHub create and normalized receipt | DONE | `createPullRequest` sends one REST request through the credential-safe transport and returns a v1 mutation result/receipt. |
| 5 | Integrate cancellation and ambiguous recovery | DONE | Pre-dispatch cancellation is terminally not-applied; post-dispatch uncertainty uses detached bounded read-back. |
| 6 | Extend the deterministic fake | DONE | `internal/mockcodehost` covers denial, retries, lost/cancelled responses, external completion, forks, stale heads, mutable created PRs, and attempt accounting. |
| 7 | Add recovery, redaction, retention, and conformance tests | DONE | `github_create_test.go` covers successful and failed contract responses, concurrency, conflicts, crash states, canaries, retention, and stable canonical keys. |

### Exercise-the-feature check

- [x] The write lifecycle was exercised end-to-end against the deterministic GitHub HTTP fake with `go test ./internal/codehost -run '^TestCreate' -count=1 -v`; focused packages, file locks, race detection, `go test ./... -count=1`, and `go vet ./...` also passed.

### Excellence Bar self-check

- [x] Yes — the path is fail-closed, contract-valid, credential-safe, crash-aware, connection-scoped for stable idempotency, and reuses Hero's existing integration, lock, atomic-write, transport, and fake boundaries.
