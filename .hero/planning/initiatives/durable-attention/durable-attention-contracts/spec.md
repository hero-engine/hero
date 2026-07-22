---
title: "Durable Attention Contracts — Ownership, Storage, Compatibility, and Trust"
slug: durable-attention-contracts
type: feature
status: planning
domain: engineering
priority: critical
size: large
horizon: next
created: 2026-07-20
parent: durable-attention
tags: [attention, contract, storage, security]
---

# Durable Attention Contracts — Ownership, Storage, Compatibility, and Trust

## Context

Hero needs two durable concepts that cross project and session boundaries without
collapsing into another work tracker. Project Mail is project-addressed inbound
communication. Personal Focus is a user's private list of prompt-backed
intentions. They have different owners, lifecycles, and mutation authorities,
but Hero Code needs one stable projection over both.

The repository already has a leaf `contracts/` package, stable peering IDs and
manifests, a user-global project registry, and a multi-project Hero Serve API.
This spec establishes the contract seam before either store is built.

## Goal

Define versioned Mail, Focus, suggestion, and read-projection contracts; choose
their storage and service authorities; and publish fixtures that Hero and Hero
Code can implement independently without sharing storage models.

## Kickoff

Implement the v1 durable-attention contract exactly as designed here. Keep
`contracts/attention` leaf-only, validate untrusted Mail at the boundary, and
preserve separate Mail and Focus write models. Publish schemas and checksumed
golden fixtures before store work. Treat Hero Serve's global HTTP endpoint as
the Hero Code boundary, with snapshot refresh required and events explicitly
optional.

## Problem

Without a provider-neutral contract, the first store or client would become the
de facto schema owner. That would invite a generic mutable Attention record,
project-tree storage, inferred actions, or Swift-specific DTOs. It would also
leave compatibility, untrusted content, row revisions, and action replay
undefined at the most expensive point to change them.

## Design

### Separate authorities

- `MailEnvelope` is immutable communication owned by the recipient project's
  mailbox. Read, acknowledgement, dismissal, and promotion receipts are
  separate records; they never rewrite the envelope.
- `FocusItem` is mutable user-owned state with exactly
  `inbox | today | later | done` lifecycle values.
- `DeferredWorkSuggestion` is a proposal, not a Focus item. Acceptance creates
  Focus through the Focus authority; dismissal creates no commitment.
- `AttentionRow` and `AttentionSnapshot` are read-only projections. No API may
  accept an Attention row as a write model.

### Identity and compatibility

- Every record carries `schema_version: 1` and an opaque, stable source ID.
- Project references contain canonical `peer_id`, optional local registry slug,
  and display name. Paths are local resolution detail and never serialized in
  portable contracts.
- Source kinds, action IDs, error codes, styles, and provenance kinds are raw
  strings. Consumers preserve and tolerate unknown values.
- Timestamps are UTC RFC3339Nano strings. Required timestamps that fail parsing
  reject writes; optional malformed timestamps are surfaced as contract errors,
  never replaced with local time.
- Additive optional fields are compatible. Removing/renaming fields, changing
  meaning, or narrowing accepted values requires a new attention schema
  version. The repository-wide `ContractsVersion` is bumped only when its
  existing breaking-change policy requires it.

### Storage authority

Add `internal/attention/state` as the single resolver for the Hero-owned user
state root. It uses `$XDG_STATE_HOME/hero` when set and otherwise
`~/.local/state/hero`; tests inject an explicit root. Mail and Focus receive
separate subdirectories and stores. No durable-attention data is written inside
a recipient repository, `.hero/`, configuration directory, or credentials
file. Directories are `0700`; state files are `0600`.

### Trust and limits

Mail subject, body, and provenance are untrusted display data. V1 permits UTF-8
text only: 200-character subject, 64 KiB body, 128 KiB encoded envelope, at most
32 provenance references, and no attachment, credential, environment, command,
or executable-payload fields. Receipt-side sender identity is taken from the
resolved local peer manifest, not trusted from user-supplied JSON. Delivery
never executes content or invokes a model.

Focus prompts are user-authored text, capped at 64 KiB. Returning a launch
intent does not execute it. Clients must display the target project and retain
normal confirmation/trust behavior before opening a session.

### Consumer boundary

The canonical Hero Code v1 transport is Hero Serve HTTP under
`/api/attention/v1`. It is user-global and therefore available before a project
is selected. CLI and project-scoped MCP tools may call the same Go services, but
Hero Code implements only the HTTP contract. Core CLI commands remain fully
usable when Hero Serve is not running. V1 requires authoritative snapshot
refresh on mount, foreground, reconnect, and successful mutation; SSE/event
delivery is optional and is not a compatibility dependency.

### Schemas and fixtures

`contracts/attention` owns Go DTOs and hand-maintained JSON Schemas under
`contracts/attention/schema/v1`. `contracts/attention/testdata/v1` contains
canonical request, response, row, error, and unknown-field fixtures. A manifest
records relative path, purpose, schema version, and SHA-256 for every fixture.
Contract tests validate schemas, decode every fixture into Go DTOs, tolerate
unknown additive fields, and verify manifest checksums. Hero Code vendors this
directory as a unit and verifies the manifest rather than copying examples from
spec prose.

## Changes

1. Add `contracts/attention/version.go` with the v1 schema constant and
   compatibility policy comments.
2. Add `contracts/attention/mail.go`, `focus.go`, `suggestion.go`, and
   `projection.go` for separate write DTOs and shared read DTOs.
3. Add `contracts/attention/action.go` for advertised action descriptors,
   preconditions, idempotency keys, authoritative results, launch intents,
   navigation references, and structured errors.
4. Add `contracts/attention/validate.go` for enum-independent structural limits,
   RFC3339 validation, IDs, and untrusted text limits.
5. Add `contracts/attention/schema/v1/*.schema.json` and
   `contracts/attention/testdata/v1/{manifest.json,*.json}`.
6. Extend `contracts/contracts_boundary_test.go` and add
   `contracts/attention/contract_test.go` to enforce the leaf boundary, schema
   parity, fixtures, checksums, and forward decoding.
7. Add `internal/attention/state/root.go` and tests for XDG/default resolution,
   injected roots, permissions, and the prohibition on project-tree storage.
8. Document `/api/attention/v1` as the one desktop consumer transport in
   `docs/serve.md`; implementation of its handlers belongs to
   `attention-read-model-v1`.

## Acceptance Criteria

- **AC-1:** WHEN Mail, Focus, suggestion, and projection records are encoded THE SYSTEM SHALL include schema version, stable opaque IDs, exact UTC timestamps,
and typed project/provenance references without local filesystem paths.

- **AC-2:** WHEN a consumer decodes an additive field, unknown source kind, or unknown action ID THE SYSTEM SHALL preserve compatibility without converting
the value to a known enum or failing the entire snapshot.

- **AC-3:** IF a write violates the v1 validation contract THEN THE SYSTEM SHALL reject it with a stable structured validation code and no partial state.

- **AC-4:** WHEN durable-attention state roots are resolved, THE SYSTEM SHALL use
the injected root or XDG user-state location with private permissions and SHALL
NOT choose a project repository, `.hero/`, config, or credentials directory.

- **AC-5:** WHEN an action is described, THE SYSTEM SHALL expose its raw ID,
label/style/confirmation metadata, input schema, row revision precondition, and
idempotency requirement.

- **AC-6:** WHEN an action succeeds or fails, THE SYSTEM SHALL return either an
authoritative updated row/result or one of the structured validation, stale,
unsupported, missing, incompatible-version, or unavailable errors.

- **AC-7:** WHEN the fixture suite runs, THE SYSTEM SHALL validate every v1 golden
fixture against its schema and Go DTO and verify every manifest SHA-256.

- **AC-8:** WHEN Hero Code integrates the feature, THE SYSTEM SHALL provide one
documented global HTTP contract under `/api/attention/v1`; streaming events
SHALL NOT be required for v1 correctness.

- **AC-9:** WHILE Mail and Focus share the read projection, THE SYSTEM SHALL keep
their mutation contracts and lifecycles separate and SHALL NOT expose a generic
Attention write endpoint.

- **AC-10:** WHEN contracts are used from `contracts/attention` THE SYSTEM SHALL compile without imports from `internal/` or any non-standard Hero package.

## Boundaries

- No Mail or Focus store implementation beyond the shared state-root resolver.
- No transport delivery, triage, UI, model execution, attachments, cloud sync,
  notification worker, or generic mutable Attention item.
- No required event cursor/replay protocol in v1.
- No Swift-only DTOs; Hero Code consumes the checked-in JSON contract.

## Risks

- Hand-maintained schema drift is controlled by parity and fixture tests.
- Global state can expose sensitive prompts or messages; private permissions
  and no project-tree writes are mandatory.
- Raw extensible strings shift strictness to action advertisement; clients must
  render only actions present in each row.
- Hero Serve availability is a client runtime concern, not a prerequisite for
  CLI use; unavailable responses must not be mistaken for empty state.

## Validation

- `go test ./contracts/... ./internal/attention/state/...`
- Contract fixture/schema/checksum test with unknown additive fields and kinds.
- Permission and path tests under injected XDG and default-home layouts.
- Boundary test proving `contracts/attention` remains leaf-only.
- `go test ./...` after downstream specs implement the shared DTOs.
