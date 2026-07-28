---
title: "Code-host broker v1 contract"
slug: code-host-broker-v1-contract
type: feature
status: planning
domain: engineering
priority: critical
size: large
created: 2026-07-27
parent: hero-code-host-broker-capabilities
depends-on:
  - code-host-integration-capability-model
relates-to:
  - brokered-tracker-agent-access
  - integration-config-uses-stable-ids
tags: [code-host, contract, pull-request, versioning, fixtures, permissions]
---

# Code-host broker v1 contract

## Context

Hero Code needs a stable pull-request boundary it can decode without learning
GitHub payloads, receiving credentials, or relying on a subprocess-specific
shape. Hero already has useful `tracker-broker/v1` envelope, bounds,
cancellation, redaction, and error-normalization patterns, but that contract is
issue-shaped and does not define mutation replay or reconciliation.

This child freezes the provider-neutral wire contract before the GitHub adapter
and transports are built. Hero owns the canonical types and fixture; Hero Code
owns its Swift decoder.

## Goal

Define an additive `code-host-broker/v1` contract for repository-qualified PR
identity, capability discovery, bounded reads, guarded writes, typed effects,
pagination, freshness, rate limits, partial failure, cancellation,
idempotency, reconciliation, and normalized errors.

## Kickoff

Freeze the Hero-owned wire boundary and canonical fixture that every code-host
adapter and Hero Code decoder must share.

**Status:** planning — existing broker, permission, credential, pagination, and
Hero Code decoder conventions have been mapped.

**Pick up at:** write the operation/policy registry and canonical identity
types, then generate a fixture containing every response and error variant.

→ `/deliver code-host-broker-v1-contract`

**Files:** `contracts/codehostbroker/contract.go`, `contracts/codehostbroker/testdata/v1/consumer-fixture.json`, `docs/contracts/code-host-broker-v1.md`
**Skip:** do not add transport handlers or reuse tracker issue DTOs.

## Problem

Without a frozen contract, the Go adapter, CLI, MCP tools, and Swift decoder can
independently invent identity, nullability, pagination, permission, and retry
semantics. A PR number is only repository-local; branch names are not stable
across forks; checks and mergeability can be unknown; and most GitHub mutations
lack provider-native idempotency. A generic “success/error” envelope would
encourage clients to retry ambiguous writes or treat unavailable state as
empty.

## Approach

Create `contracts/codehostbroker` as a dependency-light package containing the
version constant, operation registry, policy descriptors, request/response
types, bounds, normalized errors, and canonical fixture generator. It contains
no HTTP client, configuration lookup, credential, CLI, MCP, or persistence
logic.

The v1 operation inventory is:

- discovery and reads: `capabilities`, `list_pull_requests`,
  `search_pull_requests`, `get_pull_request`, `get_commits`, `get_diff`,
  `get_checks`, `get_reviews`, `get_comments`, and `get_merge_readiness`;
- mutations: `create_pull_request`, `comment`, `submit_review`, `approve`,
  `request_changes`, `mark_ready`, `retarget`, `close`, `reopen`, and `merge`.

Canonical identity is the tuple of stable `connection_id`, repository identity,
and provider PR identity. Repository identity includes provider repository ID
when known, host, owner/namespace, name, and canonical full name. PR identity
includes provider node/opaque ID when known plus repository-local number.
Repository and ref identities distinguish base and head repositories, ref
names, and commit SHAs so forks and force pushes are explicit. A PR number,
branch name, or URL alone is invalid mutation identity.

Every response uses the same envelope:

- `version`, `operation`, `provider`, `connection_id`, and canonical
  `repository`;
- authoritative operation `policy` with effect, consent, replay, revision,
  idempotency, and input-bound requirements;
- `capability_revision`, `observation_revision`, `observed_at`, freshness, and
  rate-limit metadata;
- typed `result`, pagination cursor and completeness, bounded/truncation
  metadata, section-level partial failures, duration, receipt, and
  reconciliation outcome;
- one normalized error with retry guidance.

Read operations are `read` effects requiring no consent. All non-merge
mutations are `external_write` effects requiring `explicit_user`; merge is a
`commitment` requiring `explicit_acceptance`. Mutation requests also require
`intent_source: user`, a stable idempotency key, the capability revision, a
fresh operation-specific observation revision, and reconciliation identity.
Creation uses a repository/head/base preflight revision because no PR identity
exists yet.

Normalized errors include `invalid_input`, `incompatible_version`,
`connection_not_found`, `code_host_role_missing`,
`wrong_connection_capability`, `credential_unavailable`, `unauthorized`,
`forbidden`, `unsupported_provider`, `unsupported_operation`, `not_found`,
`stale_observation`, `capability_changed`, `conflict`, `rate_limited`,
`cursor_mismatch`, `idempotency_conflict`, `operation_in_progress`,
`ambiguous_result`, `provider_unavailable`, `provider_error`,
`partial_failure`, `input_too_large`, `output_too_large`, `cancelled`, and
`encoding_error`. Retry guidance is a closed enum: `none`, `same_key`,
`refresh_then_retry`, `retry_after`, or `reconcile`.

V1 evolves additively: consumers ignore unknown fields and unknown advertised
capabilities, while unknown requested operations and incompatible major
versions fail closed. Removing or changing the meaning of a field requires v2.

## Changes

1. Add canonical contract types and validation in
   `contracts/codehostbroker`.
   - Define repository, ref, PR, actor, commit, diff, check, review, comment,
     readiness, capability, page, rate-limit, error, receipt, and
     reconciliation types.
   - Use explicit availability/completeness states rather than overloaded
     zero values.
   - Reject unqualified identities and unbounded strings, arrays, diffs, and
     page sizes.
2. Add a single operation-policy registry.
   - Declare effect, consent, idempotency, revision, freshness, target,
     replay-safety, and request-bound requirements per operation.
   - Validate requests and later derive CLI/MCP metadata from this registry.
3. Define cursor and revision material.
   - Cursors are opaque to clients and bind version, provider, connection,
     repository scope, operation, normalized query, and ordering.
   - Capability and observation revisions are opaque stable values with
     documented invalidation inputs.
4. Define mutation receipt and reconciliation outcomes.
   - Distinguish `applied`, `replayed`, `reconciled_applied`,
     `externally_completed`, `not_applied`, `in_progress`, and `ambiguous`.
   - Include provider receipt IDs and safe reconciliation keys, never request
     bodies, credentials, or provider tokens.
5. Publish `docs/contracts/code-host-broker-v1.md`.
   - Document every operation, policy, field, bound, nullability rule, error,
     retry rule, and compatibility promise.
   - Include lifecycle examples for forks, force pushes, partial checks,
     duplicate retries, ambiguous provider responses, and merge queues.
6. Generate
   `contracts/codehostbroker/testdata/v1/consumer-fixture.json`.
   - Cover every operation, availability state, pagination state, effect,
     reconciliation result, and normalized error.
   - Make generation deterministic and publish a SHA-256 digest.
   - Include unknown additive fields for consumer-forward-compatibility tests.
7. Add contract validation, round-trip, fuzz, bounds, golden fixture, and
   redaction-canary tests.

## Acceptance Criteria

- **AC-1:** THE SYSTEM SHALL define exactly the twenty v1 operations listed in this spec and advertise each with one authoritative effect, consent, idempotency, revision, freshness, replay, and bounds policy.
- **AC-2:** WHEN a PR identity is encoded THE SYSTEM SHALL include stable connection, repository, and provider PR identity; IF only a PR number, branch, or URL is supplied THEN validation SHALL fail.
- **AC-3:** WHEN base and head refs belong to different repositories THE SYSTEM SHALL preserve both repository identities, ref names, and commit SHAs without collapsing the fork.
- **AC-4:** THE SYSTEM SHALL include version, operation, provider, connection, policy, capability revision, observation/freshness metadata, rate-limit metadata, bounds, completeness, and typed result-or-error state in every v1 response.
- **AC-5:** WHEN a result is partially available THE SYSTEM SHALL preserve returned sections and report bounded section-level failures without translating the result to empty or success-complete.
- **AC-6:** WHEN pagination is used THE SYSTEM SHALL expose only opaque cursors bound to the originating operation, connection, repository scope, query, and ordering; mismatched reuse SHALL return `cursor_mismatch`.
- **AC-7:** THE SYSTEM SHALL require `intent_source: user`, a stable idempotency key, capability revision, operation-specific observation revision, and reconciliation material in every mutation request.
- **AC-8:** THE SYSTEM SHALL classify non-merge mutations as `external_write` plus `explicit_user`, merge as `commitment` plus `explicit_acceptance`, and all reads as `read` plus no consent.
- **AC-9:** WHEN cancellation or transport loss leaves provider outcome unknown THE SYSTEM SHALL represent `ambiguous` or a proven reconciliation result and SHALL NOT describe the request as safely retryable with a new key.
- **AC-10:** THE SYSTEM SHALL publish the closed normalized-error and retry-guidance enums in Go types, documentation, tests, and the canonical fixture with no credential-bearing raw provider body.
- **AC-11:** WHEN a v1 consumer receives unknown additive fields or capability names THE SYSTEM SHALL document and fixture-test that they are ignored; WHEN the major version is unsupported it SHALL fail closed.
- **AC-12:** THE SYSTEM SHALL enforce explicit bounds for repository scopes, page sizes, item counts, text fields, diff bytes/files/hunks, partial failures, error detail, duration, redirects, and mutation journal material.
- **AC-13:** WHEN the canonical fixture is regenerated with unchanged source cases THE SYSTEM SHALL produce byte-stable JSON and the same SHA-256 digest.
- **AC-14:** THE SYSTEM SHALL exclude credentials, authorization headers, token-derived values, unredacted provider bodies, and mutation text bodies from errors, receipts, reconciliation material, and fixtures.

## Boundaries

- No GitHub/GitLab transport, configuration resolver, CLI command, MCP tool, or
  durable journal implementation.
- No tracker issue, GitHub Projects, local Swift cache, UI, or webhook schema.
- No generic arbitrary provider-request operation.
- No v2 compatibility shim; v1 additive evolution is the only compatibility
  mechanism in this child.

## Risks

- A single envelope can become a bag of nullable fields. Operation-specific
  result types and explicit availability states must keep invalid states
  unrepresentable.
- Capability policy duplicated into transport definitions will drift. Later
  surfaces must derive metadata from the registry and test the full inventory.
- Provider IDs differ across REST and GraphQL. The contract must permit opaque
  provider IDs while never using them without repository identity.
- Overly generous fixture values can normalize accidental unbounded behavior.
  Fixture generation must exercise maximums and rejection cases separately.

## Validation

- Run contract unit, validation, fuzz, JSON round-trip, golden fixture,
  additive-field, and redaction-canary tests.
- Generate the fixture twice and compare bytes and SHA-256 digest.
- Decode every fixture case into a strict independent test decoder and verify
  unknown additive fields do not affect known values.
- Run `go test ./contracts/codehostbroker/...`, then `go test ./...` and
  `go vet ./...`.
