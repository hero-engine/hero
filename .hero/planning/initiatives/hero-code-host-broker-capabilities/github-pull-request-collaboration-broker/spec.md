---
title: "GitHub pull-request collaboration broker"
slug: github-pull-request-collaboration-broker
type: feature
status: planning
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

**Status:** planning — GitHub review states, comment identities, mutation
journal, and ambiguous-response requirements are mapped.

**Pick up at:** define one collaboration reconciliation marker and normalized
review receipt, then extend the mutation state machine operation by operation.

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
