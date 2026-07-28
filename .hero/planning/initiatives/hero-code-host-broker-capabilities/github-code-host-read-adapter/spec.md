---
title: "GitHub code-host read adapter"
slug: github-code-host-read-adapter
type: feature
status: delivering
domain: engineering
priority: high
size: large
created: 2026-07-27
parent: hero-code-host-broker-capabilities
depends-on:
  - code-host-broker-v1-contract
conflicts-with:
  - github-pull-request-create-broker
tags: [github, code-host, pull-request, reads, pagination, fake-adapter]
delivery_method: manual
---

# GitHub code-host read adapter

## Context

The v1 contract defines provider-neutral PR reads, but Hero's current GitHub
adapter is an issue-tracker implementation under `internal/tracker`. Its
GitHub Projects importer and mock tracker intentionally do not expose pull
requests. Code-host reads need a separate domain adapter that reuses the
credential-safe connection boundary and generic transport safety without
conflating issues with repositories.

## Goal

Implement the in-process GitHub read side of `code-host-broker/v1`, including
capability discovery, list/search/detail, commits, bounded diffs, checks,
reviews, comments, merge readiness, pagination, freshness, rate limits,
partial failure, cancellation, forks, and force-push detection.

## Kickoff

Adds Hero's credential-safe GitHub read adapter and deterministic provider fake
without coupling pull requests to issue tracking.

**Status:** in-review — all ten read/discovery operations, safety boundaries,
fake scenarios, and conformance tests are implemented and green.

**Pick up at:** cold-audit the committed adapter against all 14 criteria, then
run `hero spec verify github-code-host-read-adapter`.

→ `.hero/planning/initiatives/hero-code-host-broker-capabilities/github-code-host-read-adapter/spec.md`

**Files:** `internal/codehost/broker.go`, `internal/codehost/github.go`, `internal/codehost/broker_test.go`, `internal/mockcodehost/server.go`
**Skip:** do not add PR routes to `internal/tracker` or `internal/mocktracker`.

## Problem

GitHub splits PR data across REST and GraphQL, returns permission-dependent
sections, exposes transient mergeability, and paginates nested collections.
Fork heads make branch-only identity unsafe, force pushes invalidate prior
reviews and diffs, and rate-limited or unauthorized sections must not appear as
empty. Reusing the issue adapter would couple tracker selection to code-host
selection and encourage transport DTO leakage.

## Approach

Create a distinct `internal/codehost` broker and GitHub adapter. It consumes the
resolved `CodeHostConnection`, validates repository scope, dispatches typed
REST/GraphQL requests, normalizes provider responses into contract types, and
returns no provider DTO. Narrow origin, redirect, redaction, bounded-read, and
HTTP error helpers may move to an integration-level internal package only when
the tracker broker delegates to the same helper and its behavior stays intact.

Repository access is always explicit: the request names one configured
repository or a bounded configured repository set. The adapter never enumerates
an account or organization implicitly. GitHub Enterprise uses the resolved
connection origin and every pagination/redirect target must remain same-origin.

The adapter implements `capabilities`, `list_pull_requests`,
`search_pull_requests`, `get_pull_request`, `get_commits`, `get_diff`,
`get_checks`, `get_reviews`, `get_comments`, and `get_merge_readiness`.
Cursors bind the normalized operation, connection, repository set, filters,
sort, and page position. Nested collections report their own completeness and
next cursor where needed.

Observation revisions hash canonical mutable inputs appropriate to each view:
provider PR identity, lifecycle state, base/head repositories and refs, head
SHA, updated timestamp, and relevant section cursors/status. A new head SHA
after force push creates a new revision even when the PR number and branch names
are unchanged. Merge readiness represents `ready`, `blocked`, `unknown`, or
`unavailable`, with checks, reviews, branch protection, permission, draft,
mergeability, queue, and stale-head evidence kept distinct.

The adapter reports GitHub rate-limit resource, limit, remaining, reset,
retry-after, and observed time when available. It does not sleep or perform a
hidden retry. A read may return useful bounded sections plus typed partial
failures; `403`, `404`, `429`, GraphQL errors, truncation, cancellation, and
decode failures are never normalized to empty.

Build `internal/mockcodehost` as a deterministic GitHub protocol fake with
fixtures for permissions, pagination, rate limits, forks, force pushes,
partial GraphQL errors, changing mergeability, and oversized diffs. It is not
an extension of the issue mock.

## Changes

1. Add the in-process code-host broker, operation dispatch, request validation,
   capability registry integration, bounds, and cancellation handling.
2. Add a typed GitHub transport under `internal/codehost`.
   - Resolve only the configured connection and repositories.
   - Support GitHub.com and configured Enterprise origins with same-origin
     pagination and redirects.
   - Keep authorization headers inside the transport.
3. Implement and normalize all ten read/discovery operations.
   - Preserve repository-qualified base/head identity and provider IDs.
   - Model section completeness and partial failures explicitly.
   - Bound items, pages, text, commit lists, diff files, hunks, lines, and
     bytes.
4. Add opaque cursor encoding/validation and operation-specific observation
   revision generation.
5. Normalize rate limits, permissions, checks, reviews, draft state, branch
   protection, merge queue, and mergeability without inventing certainty.
6. Add `internal/mockcodehost` and scenario builders independent from
   `internal/mocktracker`.
7. Add broker, adapter, contract-conformance, cancellation, bounds, origin,
   redaction, and fake-provider tests.

## Acceptance Criteria

- **AC-1:** WHEN the selected GitHub connection is ready THE SYSTEM SHALL advertise exactly the read operations implemented by the current adapter plus their authoritative policies and bounds.
- **AC-2:** WHEN list or search spans configured repositories THE SYSTEM SHALL return repository-qualified PR identities and an opaque bounded cursor without implicit organization enumeration.
- **AC-3:** IF a cursor is reused with a different connection, repository set, operation, filter, query, sort, or contract version THEN THE SYSTEM SHALL return `cursor_mismatch` and perform no provider request.
- **AC-4:** WHEN a PR head originates from a fork THE SYSTEM SHALL preserve distinct base/head repository IDs, canonical names, refs, and head SHA.
- **AC-5:** WHEN a force push changes the head SHA THE SYSTEM SHALL emit a different observation revision and SHALL NOT report prior diff, review, check, or readiness state as current.
- **AC-6:** WHEN commits or diffs exceed a declared bound THE SYSTEM SHALL return the bounded prefix plus explicit truncation/completeness metadata rather than silently dropping data.
- **AC-7:** WHEN one GitHub section is unauthorized, rate-limited, unavailable, or malformed THE SYSTEM SHALL preserve valid sections and emit a bounded typed partial failure for the affected section.
- **AC-8:** WHEN GitHub reports mergeability as pending or required evidence cannot be fetched THE SYSTEM SHALL report readiness as `unknown` or `unavailable`, never `ready` or `blocked` by guess.
- **AC-9:** THE SYSTEM SHALL normalize rate-limit resource, limit, remaining, reset, retry-after, and observation time from REST and GraphQL responses when present.
- **AC-10:** WHEN the request context is cancelled before dispatch THE SYSTEM SHALL make no provider request; WHEN cancellation occurs during reads it SHALL stop further pages and return typed cancellation with any explicitly partial result.
- **AC-11:** THE SYSTEM SHALL reject same-request repository names outside the selected connection's configured scope before sending authorization to the provider.
- **AC-12:** WHEN GitHub Enterprise pagination or redirects leave the configured origin THE SYSTEM SHALL fail closed without forwarding authorization.
- **AC-13:** THE SYSTEM SHALL implement the fake GitHub scenarios for permissions, pagination, rate limits, forks, force pushes, partial failures, changing mergeability, and oversized diffs without modifying the tracker mock.
- **AC-14:** THE SYSTEM SHALL keep credentials and raw authorization data out of normalized results, errors, logs, fixture snapshots, and test failure output.

## Boundaries

- No create, comment, review, state transition, or merge provider writes.
- No CLI or MCP exposure; this child is the authoritative in-process read path.
- No tracker selection, issue DTO, GitHub Projects importer, `gh` subprocess,
  webhook, background poller, or local PR cache.
- No GitLab adapter despite GitLab's valid code-host capability declaration.

## Risks

- REST and GraphQL may disagree during propagation. Preserve observed source
  and time and use partial/unknown states rather than precedence guesses.
- Nested pagination can create unbounded fan-out. Enforce total request,
  duration, page, item, and byte budgets across the whole operation.
- Extracting security helpers can accidentally alter the tracker broker. Keep
  extraction narrow and run existing tracker tests unchanged.
- Search syntax is provider-specific. Normalize only the documented v1 filter
  subset and keep arbitrary GitHub query strings out of the provider-neutral
  contract.

## Validation

- Run unit and fake-provider tests for every read operation and normalized
  response state.
- Exercise multi-page, cursor mismatch, nested pagination, rate limit, partial
  GraphQL error, cancellation, excessive diff, fork, and force-push scenarios.
- Run credential canary, hostile redirect, Enterprise origin, and repository
  scope tests.
- Run `go test ./internal/codehost/... ./internal/mockcodehost/...`, existing
  tracker broker tests, then `go test ./...` and `go vet ./...`.

## Completion Ledger

Implemented as a separate Go broker and GitHub adapter over the existing
`CodeHostConnection` credential boundary. Validation included focused
adapter/fake and tracker regression tests, the full repository suite, full
`go vet`, the race detector, contract response validation, and diff hygiene.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Advertise exactly implemented reads with authoritative policies and bounds | DONE | `TestCapabilitiesAdvertiseExactlyImplementedReads` verifies the ten-operation registry against `codehostbroker.Policy` without a provider request. |
| 2 | List/search configured repositories with qualified identity and opaque cursor | DONE | `TestListAndSearchUseConfiguredRepositoryScopeAndOpaqueCursor` exercises bounded multi-repository list/search and decodes only the contract cursor envelope. |
| 3 | Reject cursor material mismatches before provider I/O | DONE | Cursor validation binds version, connection, repository set, operation, query, and order; the mismatch test asserts provider request count is unchanged. |
| 4 | Preserve fork-qualified base/head identity | DONE | `TestForkAndForcePushPreserveIdentityAndFreshness` verifies distinct provider IDs, canonical repositories, refs, and SHAs. |
| 5 | Invalidate prior head-dependent state after force push | DONE | Freshness rechecks discard stale diff, checks, reviews, and readiness, emit `stale_observation`, and change observation revisions with head SHA. |
| 6 | Return explicit bounded commit/diff prefixes | DONE | `TestCommitAndDiffBoundsAreExplicit` verifies item/file/hunk/byte limits, truncation metadata, and commit continuation cursors. |
| 7 | Preserve useful data with typed section failures | DONE | Partial collection reads and GraphQL readiness retain normalized results while emitting bounded typed failures; malformed or unavailable sole sections remain typed top-level errors. |
| 8 | Never invent merge readiness certainty | DONE | `TestPartialGraphQLAndMergeabilityNeverInventReadiness` covers pending mergeability and unavailable required-check evidence. |
| 9 | Normalize REST and GraphQL rate limits | DONE | Tests assert resource, limit, remaining, reset, retry-after, and observation time for both transport styles. |
| 10 | Cancel before dispatch or during pagination | DONE | `TestCancellationBeforeAndDuringPagination` verifies zero pre-dispatch calls and a typed partial prefix during a paginated read. |
| 11 | Reject repositories outside connection scope before authorization | DONE | Scope validation precedes token resolution and transport construction; the scope test asserts zero provider calls. |
| 12 | Fail closed on cross-origin Enterprise pagination/redirects | DONE | Hostile Link and redirect tests assert no target request or authorization forwarding; Enterprise base paths are exercised separately. |
| 13 | Provide independent deterministic GitHub fake scenarios | DONE | `internal/mockcodehost` covers permissions, pagination, rate limits, forks, force pushes, partial GraphQL data, changing mergeability, oversized diffs, and cancellation without tracker-mock edits. |
| 14 | Keep credentials and raw authorization out of outputs | DONE | Canary tests cover redirects, provider/config errors, normalized responses, and the fake's credential-free request inventory. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Add in-process broker, dispatch, validation, bounds, cancellation | DONE | `internal/codehost/broker.go` owns the provider-neutral read boundary and contract response envelope. |
| 2 | Add typed GitHub transport | DONE | `internal/codehost/github.go` resolves configured origins only, contains authorization, bounds responses, and validates pagination/redirect origins. |
| 3 | Implement and normalize all ten reads/discovery operations | DONE | `internal/codehost/github.go` and `internal/codehost/normalize.go` return contract DTOs only with explicit completeness. |
| 4 | Add opaque cursors and observation revisions | DONE | Cursor material uses the v1 encoder/validator; revisions hash normalized mutable result and head state. |
| 5 | Normalize readiness, rate limits, permissions, and transient state | DONE | REST headers and GraphQL readiness are mapped conservatively with typed availability and reasons. |
| 6 | Add independent `internal/mockcodehost` | DONE | `server.go` and `server_test.go` implement and verify the required deterministic scenarios and redacted request inventory. |
| 7 | Add adapter, conformance, cancellation, bounds, origin, and redaction tests | DONE | `broker_test.go` validates all ten successful responses plus typed failure paths against `ValidateResponse`; focused race and full-repository checks pass. |

### Exercise-the-feature check

- [x] In-process broker behavior was exercised end-to-end through the deterministic GitHub HTTP fake with `go test ./internal/codehost/... ./internal/mockcodehost/... -count=1`; the complete repository passed `go test ./... -count=1`, `go vet ./...`, and focused race detection.

### Excellence Bar self-check

- [x] Yes — the adapter is provider-neutral at its public boundary, keeps credentials inside Hero, fails closed at origin/scope boundaries, returns bounded conservative state, and has deterministic coverage for the specified provider hazards.
