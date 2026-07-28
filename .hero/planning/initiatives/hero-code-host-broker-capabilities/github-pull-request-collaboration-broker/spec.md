---
title: "GitHub pull-request collaboration broker"
slug: github-pull-request-collaboration-broker
type: feature
status: delivering
domain: engineering
priority: high
size: medium
created: 2026-07-27
parent: hero-code-host-broker-capabilities
depends-on:
  - github-pull-request-create-broker
conflicts-with:
  - github-pull-request-create-broker
  - github-pull-request-state-transition-broker
tags: [github, pull-request, comments, reviews, idempotency, permissions]
delivery_method: manual
---

# GitHub pull-request collaboration broker

## Context

PR comments and reviews are externally visible collaboration effects. GitHub
does not accept caller idempotency keys for them, a review is tied to a
particular head commit, and repeated approval/request-changes submissions can
change the visible review timeline. The creation broker provides a durable
replay journal and ambiguous-response state machine that this child can extend
with collaboration-specific reconciliation.

## Goal

Implement `comment`, `submit_review`, `approve`, and `request_changes` through
Hero's broker with explicit user intent, fresh PR/head evidence, stable
idempotency, exact provider receipts, safe read-back markers, permission
enforcement, and externally completed action recognition.

## Kickoff

Add brokered PR discussion and review writes without allowing retries to post
duplicate comments or reviews.

**Status:** in review — all four collaboration operations, shared mutation
reconciliation, and deterministic provider scenarios are implemented and
validated.

**Pick up at:** cold-audit the completed delivery and verify the spec.

→ `/deliver github-pull-request-collaboration-broker`

**Files:** `internal/codehost/github_collaboration.go`, `internal/codehost/reconcile.go`, `internal/mockcodehost/reviews.go`
**Skip:** do not change PR lifecycle state, retarget, merge, or infer user intent from message text.

## Problem

Matching a lost comment by body text and timestamp is ambiguous, while blindly
retrying duplicates a visible message. Reviews add current-head, author,
permission, and state semantics; a prior approval on an old head cannot prove
the current request was applied. Provider receipt IDs solve successful
responses but not the lost-response window.

## Approach

Require repository-qualified PR identity, current head SHA,
`capability_revision`, `observation_revision`, `intent_source: user`, a stable
idempotency key, and bounded body text where applicable. `submit_review`
supports a neutral comment review; `approve` and `request_changes` remain
explicit operations so permission surfaces and model clients cannot smuggle a
decision through an arbitrary enum.

Before dispatch, Hero verifies the PR is open, the head SHA/revision is
current, the operation is supported, and the authenticated actor has the
required permission. A stale head or changed capability fails before writing.

For exact recovery, Hero appends an inert provider-visible HTML comment marker
such as `<!-- hero-code-host-op:<opaque-id> -->` to the submitted body. The
opaque ID is derived from the stable journal operation identity, contains no
credential or user text, is bounded, and is returned as safe reconciliation
material. Normalized broker bodies remove only a syntactically valid
Hero-owned marker. The marker is used solely to locate a possibly applied
same-key request after loss; it is not accepted as authorization or intent.

On success, persist the provider comment/review receipt ID and current head
SHA. On ambiguous response, query the authenticated actor's bounded comments or
reviews for the exact marker and expected head, then return
`reconciled_applied`, `not_applied`, or `ambiguous`. A same-key retry continues
that reconciliation. For approval/request-changes, if the actor's authoritative
current-head review already has the requested state before dispatch, return
`externally_completed`; do not add another timeline event. A matching state on
an old head does not qualify. Comment bodies without the marker, including
identical human comments, are not claimed as this operation.

## Changes

1. Extend the mutation journal for collaboration operation identity, marker,
   provider receipt, actor identity, and expected head SHA without persisting
   body text.
2. Add common collaboration preflight for PR state, permissions, capability
   revision, observation revision, head SHA, body bounds, and user intent.
3. Implement GitHub issue-comment creation for `comment`.
4. Implement GitHub pull-review submission for neutral review, approval, and
   request-changes operations.
5. Add marker injection, normalized stripping, bounded read-back, exact receipt
   matching, and current-head externally completed recognition.
6. Extend the fake with lost responses, delayed visibility, permission
   changes, dismissed reviews, old-head reviews, duplicate retries, marker
   collisions, and externally completed actions.
7. Add contract, journal, concurrency, cancellation, partial failure,
   redaction, and reconciliation tests.

## Acceptance Criteria

- **AC-1:** WHEN collaboration is requested THE SYSTEM SHALL require repository-qualified PR identity, current head SHA, user intent, idempotency key, capability revision, and observation revision.
- **AC-2:** THE SYSTEM SHALL advertise `comment`, `submit_review`, `approve`, and `request_changes` as separate `external_write` operations requiring `explicit_user`.
- **AC-3:** IF the PR is closed, the head/revision is stale, the operation is unsupported, or the actor lacks permission THEN THE SYSTEM SHALL return a typed error before provider dispatch.
- **AC-4:** WHEN a comment or review is dispatched THE SYSTEM SHALL include one bounded opaque Hero marker containing no credential, user text, authorization, or secret-derived value.
- **AC-5:** WHEN GitHub returns success THE SYSTEM SHALL persist and return the provider receipt ID, repository-qualified PR identity, actor, head SHA, and `applied` outcome.
- **AC-6:** WHEN a response is lost after application THE SYSTEM SHALL locate the exact same-key marker and expected head and return `reconciled_applied` without posting again.
- **AC-7:** WHEN read-back finds no exact marker but provider application remains uncertain THE SYSTEM SHALL return `ambiguous` and permit only same-key reconciliation.
- **AC-8:** WHEN the same actor has already approved or requested changes on the current head before dispatch THE SYSTEM SHALL return `externally_completed` without creating another review event.
- **AC-9:** WHEN an equivalent review exists only on an older head, has been dismissed, or belongs to another actor THE SYSTEM SHALL NOT treat it as externally completed.
- **AC-10:** WHEN the same key and payload are retried concurrently or sequentially THE SYSTEM SHALL produce at most one provider collaboration effect and the same receipt/reconciliation result.
- **AC-11:** IF a key is reused with a different operation, target, head, review state, or body digest THEN THE SYSTEM SHALL return `idempotency_conflict`.
- **AC-12:** WHEN normalized comments/reviews are returned THE SYSTEM SHALL remove only valid Hero-owned markers while preserving all other user content and reporting malformed lookalikes unchanged.
- **AC-13:** WHEN cancellation occurs before dispatch THE SYSTEM SHALL perform no write; cancellation after dispatch SHALL use the same ambiguous-response reconciliation path.
- **AC-14:** THE SYSTEM SHALL fixture-test permissions, duplicate retries, delayed visibility, dismissed/old-head reviews, marker collision, externally completed action, and lost response through the fake adapter.

## Boundaries

- No mark-ready, retarget, close, reopen, merge, branch update, tracker
  transition, notification, or automatic reviewer assignment.
- No interpretation of received comment/review text as model instructions.
- No storage of comment/review body text in the replay journal.
- No claim that identical unmarked external comments were created by Hero.

## Risks

- Hidden markers are visible in raw provider markup. Document the marker,
  minimize it, and ensure normalized clients do not display it.
- Provider eventual consistency can delay read-back. Use bounded reconciliation
  attempts and return ambiguous rather than redispatching.
- GitHub permission and review dismissal rules can change between preflight and
  dispatch. Preserve provider conflicts as typed results and never reinterpret
  them as success.
- Review state can be current for an actor but obsolete after a force push.
  Head SHA is part of both preflight and reconciliation identity.

## Validation

- Exercise every operation through success, duplicate, concurrent, lost,
  ambiguous, cancelled, stale-head, permission-denied, and externally completed
  scenarios.
- Fuzz marker parsing and test body canaries, maximum lengths, malformed
  lookalikes, and provider round trips.
- Verify journal snapshots contain only digests and safe identifiers.
- Run code-host/fake tests, then `go test ./...` and `go vet ./...`.

## Completion Ledger

Implemented on the existing in-process broker, GitHub transport, connection
resolver, durable mutation journal, file lock, and deterministic provider fake.
Validation included focused broker/fake/file-lock tests, race detection, marker
fuzzing, the full repository suite, full `go vet`, formatting, and diff hygiene.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Require qualified PR, head, intent, idempotency, and revisions | DONE | Contract validation plus `decodeCollaborationPayload` and `PrepareCollaboration` require repository-qualified identity, expected head, explicit user intent/consent, stable keys, and capability/observation evidence. |
| 2 | Advertise four separate explicit-user external writes | DONE | The capability registry exposes `comment`, `submit_review`, `approve`, and `request_changes` using the authoritative v1 policies. |
| 3 | Reject unsupported, closed, stale, and forbidden writes before dispatch | DONE | Common preflight verifies operation, scope, open PR, provider identity, exact head, permission, capability revision, and observation revision before incrementing provider attempts. |
| 4 | Dispatch one bounded opaque content-free marker | DONE | `collaborationMarker` derives one fixed-size lowercase-hex marker from the stable operation identity; payload validation reserves the marker bytes and rejects caller-supplied valid markers. |
| 5 | Persist and return exact safe collaboration receipts | DONE | Success records the provider receipt ID, qualified PR identity, authenticated actor, and head SHA without body text, then returns an `applied` v1 mutation result with the actor's typed login and provider ID. |
| 6 | Reconcile lost responses by exact marker, actor, head, and review state | DONE | Detached bounded read-back matches the same marker and actor; reviews additionally require the expected head and requested state before returning `reconciled_applied`. |
| 7 | Preserve ambiguity when exact proof is absent | DONE | Missing, delayed, colliding, partial, or mismatched-state read-back records `ambiguous` and never redispatches under a different key. |
| 8 | Recognize current-actor/current-head approval decisions | DONE | Preflight returns `externally_completed` for the authenticated actor's authoritative current-head `APPROVED` or `CHANGES_REQUESTED` state with zero writes. |
| 9 | Exclude old-head, dismissed, and other-actor reviews | DONE | Deterministic scenarios prove those reviews do not qualify for external completion and produce one requested effect instead. |
| 10 | Same-key retries produce at most one provider effect | DONE | The existing cross-platform journal lock and shared reconciliation state machine serialize sequential and concurrent retries across all four operations. |
| 11 | Conflicting key reuse fails before another effect | DONE | The connection-scoped journal key plus canonical collaboration digest binds operation, target, head, body digest, intent, consent, and reconciliation key. |
| 12 | Strip only syntactically valid Hero markers | DONE | Normalized comment/review bodies remove the exact bounded marker grammar while preserving surrounding text and malformed lookalikes; fuzzing checks the parser invariant. |
| 13 | Respect pre- and post-dispatch cancellation boundaries | DONE | Pre-dispatch cancellation records `not_applied` with zero writes; cancellation after application uses detached exact reconciliation without replay. |
| 14 | Exercise required provider hazards through the fake | DONE | `internal/mockcodehost` covers permissions and changes, duplicates, delayed visibility, lost/ambiguous responses, marker collisions, mismatched review state, old/dismissed/other-actor reviews, external completion, force pushes, cancellation, and partial read-back. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Extend the mutation journal with safe collaboration identity | DONE | Journal targets and receipts retain only digests, marker, qualified identifiers, actor, provider receipt, and expected head; canary tests reject body and credential persistence. |
| 2 | Add common collaboration validation and preflight | DONE | `collaboration.go` and `PrepareCollaboration` provide strict payload bounds plus provider-backed PR/head/actor/permission revisions. |
| 3 | Implement GitHub issue-comment creation | DONE | `dispatchCollaboration` uses the existing authorized transport and validates the returned marker and authenticated actor. |
| 4 | Implement neutral and decision review submission | DONE | Review dispatch maps only the explicit operation to `COMMENT`, `APPROVE`, or `REQUEST_CHANGES`, pins `commit_id`, and validates the returned state. |
| 5 | Add exact marker recovery and normalization | DONE | Marker injection, read-back, strict stripping, safe receipts, and current-head external-completion logic share one bounded implementation. |
| 6 | Extend the deterministic GitHub fake | DONE | Fake scenarios model write denial, permission changes, response loss, delayed/failed visibility, collisions, state mismatch, external decisions, force pushes, and cancellation. |
| 7 | Add conformance, concurrency, cancellation, and redaction tests | DONE | Collaboration tests validate every response contract, all four operations, stable-key semantics, crash-safe recovery, body bounds, marker fuzzing, and journal canaries. |

### Exercise-the-feature check

- [x] All four writes were exercised end-to-end against the deterministic GitHub HTTP fake with focused and race-enabled code-host tests; marker parsing completed 130,125 fuzz executions, and `go test ./... -count=1` plus `go vet ./...` passed.

### Excellence Bar self-check

- [x] Yes — the delivery is fail-closed, provider-neutral at the broker boundary, credential-safe, head- and actor-bound, exact-state-aware, bounded, crash-aware, and built by extending Hero's existing integration, transport, journal, lock, and fake paths.
