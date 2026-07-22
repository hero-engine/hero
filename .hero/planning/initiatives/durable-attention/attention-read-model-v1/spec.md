---
title: "Attention Read Model v1 — Consumer-Safe Mail and Focus Projection"
slug: attention-read-model-v1
type: feature
status: planning
domain: engineering
priority: medium
size: large
horizon: next
created: 2026-07-20
parent: durable-attention
depends-on: [personal-focus-core, project-mail-triage-and-provenance, deferred-work-suggestion-contract]
tags: [attention, projection, api, schema, hero-code]
---

# Attention Read Model v1 — Consumer-Safe Mail and Focus Projection

## Context

Mail, Focus, and deferred suggestions now have separate authorities. Hero Code
needs one user-global “Today” surface before a project is opened, but it must not
read Hero files or reproduce mutation rules. Hero Serve already exposes a
multi-project HTTP API and is the smallest existing process-global boundary.

The consumer spec `durable-attention-consumer` requires stable rows, advertised
actions, authoritative results, compatibility fixtures, and graceful
unavailability. It can deliver v1 from snapshot refresh; event cursors are not
needed to make the feature correct.

## Goal

Publish a deterministic Attention snapshot and capability-driven action API
over unread Mail, Today Focus, and pending deferred suggestions, backed solely
by their owning services and the v1 contracts.

## Kickoff

Implement one projection service and mount its global HTTP handlers before the
generic project router. Hero Code's canonical transport is
`/api/attention/v1`; CLI and MCP are adapters over the same service. Build rows
only from source authorities, advertise only valid actions, return
authoritative post-action state, and require snapshot refresh rather than event
streaming for v1 correctness.

## Problem

Without a projection boundary, a client must parse Mail/Focus storage or infer
actions from status strings. That creates duplicate domain logic, stale writes,
and a Swift-owned compatibility contract. A generic Attention mutation endpoint
would also erase the ownership rules established by the core specs.

## Design

### Projection contents and ordering

Add `internal/attention/projection.Service`. `Snapshot` loads:

1. non-dismissed unread Mail addressed to registered local peer IDs;
2. Personal Focus items in `today`;
3. pending, unexpired deferred-work suggestions.

Each becomes an `AttentionRow` with stable row ID `<source-kind>:<source-id>`,
raw source kind, source revision, project/provenance refs, title/summary,
created/updated/activity timestamps, presentation group
`mail | focus | suggestion`, unread/Today flags, availability, and advertised
actions. It never copies a source DTO into a mutable common record.

Ordering is group rank Mail, Focus, Suggestion; then activity timestamp oldest
first for unread Mail and newest first for Focus/Suggestion; then stable source
ID. Missing activity sorts last within its group. V1 returns the full active
projection without pagination; done/later Focus, read/dismissed Mail, and
accepted/dismissed/expired suggestions are source-history concerns, not Today
rows. Counts include active rows by group.

The snapshot revision is a SHA-256 over canonical ordered row identity,
revision, and advertised action IDs. `generated_at` is excluded so an unchanged
refresh has the same revision. `refresh_token` equals this opaque revision in
v1.

### Capability-driven actions

The projection asks each source service for applicable actions. Examples:

- Mail: `mark_read`, `acknowledge`, `reply`, `dismiss`, `promote`,
  `add_to_today`, only where supported.
- Focus: `launch`, `move_inbox`, `move_later`, `complete`.
- Suggestion: `do_next`, `today`, `later`, `dismiss`.

Descriptors carry label, style, confirmation hint, JSON input schema, source
revision precondition, and idempotency requirement. Unknown action IDs remain
decodable. Clients render and dispatch only advertised actions.

`Dispatch` validates schema version, row/source identity, advertised action,
revision, input, and idempotency key, then delegates to the owning service.
Success returns the authoritative source result, updated/removed projected row,
new snapshot revision, optional promotion navigation reference, or launch
intent. It never accepts an Attention row as input. Failures use the contract
codes and include current row when safe for refresh.

### Global HTTP API

Mount before `API.routeProject`:

- `GET /api/attention/v1/snapshot`
- `POST /api/attention/v1/actions`
- `GET /api/attention/v1/contract` returning supported version and fixture
  manifest checksum, not the fixture bodies.

Use existing Hero Serve JSON envelope/error conventions and local server
security posture. The API resolves projects through the global registry and
loads user-state stores independent of an open project. If state cannot be
loaded, return `unavailable`; do not return an empty snapshot.

Hero Code implements this HTTP boundary only. CLI exposes
`hero attention today --json` and `hero attention act ...`; project-scoped MCP
exposes `hero_attention_snapshot` and `hero_attention_action` for harness use.
All adapters share `projection.Service`.

### Refresh and compatibility

The authoritative v1 synchronization algorithm is fetch on mount, foreground,
server reconnect, and every successful mutation. A stale action causes one
refresh and user re-selection, not automatic mutation replay. Unsupported,
missing, validation, and incompatible-version responses also refresh without
retry. Unavailable clients may retain a labelled stale read-only snapshot.

No event cursor, ordering, replay, duplicate, expiry, or gap-recovery promise is
made in v1. Existing `/api/events` is not extended for Attention. Streaming may
be designed later as an additive capability.

## Changes

1. Add `internal/attention/projection/service.go`, `ordering.go`,
   `actions.go`, and focused tests with source service interfaces/fakes.
2. Add `internal/serve/api/attention.go` or the repository's equivalent handler
   package for snapshot, action, and contract endpoints.
3. Mount `/api/attention/v1/*` in `internal/serve/api.go` before the generic
   `/api/{project}` router and wire global state/registry dependencies from
   `internal/serve/server.go`.
4. Add `internal/cli/attention.go`, register it in `internal/cli/root.go`, and
   reuse the projection service for JSON/human output.
5. Add MCP definitions/dispatch/handlers for the two Attention operations,
   delegating to the same service rather than composing existing MCP tools.
6. Complete the v1 fixture set under `contracts/attention/testdata/v1` with
   empty/mixed snapshots, all source states/actions, missing projects, unknown
   fields/kinds/actions, every structured error, promotions, launch intents,
   and suggestions before/after acceptance.
7. Add a Hero Code handoff note beside the fixture manifest documenting the
   HTTP route, snapshot-only refresh policy, and exact manifest checksum.
8. Update `docs/serve.md` and CLI/MCP references with the canonical transport
   decision and unavailability semantics.

## Acceptance Criteria

- **AC-1:** WHEN a snapshot is requested, THE SYSTEM SHALL project unread active Mail, Today Focus, and pending unexpired suggestions with stable row IDs, revisions, provenance, project availability, display fields, and advertised actions.
- **AC-2:** WHEN source state is unchanged across refreshes, THE SYSTEM SHALL return the same snapshot revision regardless of `generated_at`.
- **AC-3:** WHEN rows are returned, THE SYSTEM SHALL order them deterministically by documented group, activity, and stable ID rules without reading client or project-tree state directly.
- **AC-4:** WHEN an action is dispatched, THE SYSTEM SHALL validate version, source identity, advertised capability, revision, input, and idempotency before delegating to the owning service.
- **AC-5:** WHEN an action succeeds, THE SYSTEM SHALL return authoritative source state, projected row/invalidation, snapshot revision, and any typed navigation or launch result.
- **AC-6:** IF an action is stale, unsupported, missing, invalid, incompatible, or unavailable THEN THE SYSTEM SHALL return the matching structured error and SHALL NOT infer or retry a mutation.
- **AC-7:** WHEN Hero Serve starts without an open project, THE SYSTEM SHALL serve the user-global snapshot through `/api/attention/v1/snapshot` using the registry and global stores.
- **AC-8:** WHEN CLI, MCP, and HTTP adapters are exercised with the same state, THE SYSTEM SHALL return semantically identical contract records from one projection service.
- **AC-9:** WHEN a v1 client reconnects or completes a mutation, THE SYSTEM SHALL converge through authoritative snapshot refresh without requiring an event stream.
- **AC-10:** WHEN every checked-in golden fixture is decoded, THE SYSTEM SHALL tolerate unknown additive fields/source kinds/action IDs and preserve all documented v1 fields exactly.
- **AC-11:** WHILE exposing Attention, THE SYSTEM SHALL NOT offer a generic Attention write operation or permit a client to manufacture source actions from statuses.

## Boundaries

- No Hero Code UI, direct Swift storage adapter, push notification, remote sync,
  pagination, or mandatory streaming events.
- No source lifecycle implementation; projection delegates to Mail, Focus, and
  suggestion services.
- No history view for read Mail, Later/Done Focus, or resolved suggestions.
- No second desktop transport; Hero Code uses the HTTP API only.

## Risks

- Full active snapshots could grow; v1 intentionally projects only current
  attention. Pagination is deferred until measured need.
- Mixed-store reads are not transactionally simultaneous; per-row revisions and
  deterministic refresh provide convergence without a shared write database.
- Canonical hash drift would cause needless refreshes; golden tests pin encoding
  and ordering.
- Local Hero Serve may be stopped; unavailable is distinct from empty and Hero
  Code already specifies stale read-only behavior.

## Validation

- `go test ./internal/attention/projection/... ./internal/serve/... ./internal/cli/...`
- Golden tests for empty/mixed snapshots, every action/result/error, unknown
  values, missing projects, exact timestamps, and deterministic revisions.
- Cross-adapter parity test invoking service, HTTP, CLI JSON, and MCP fixtures.
- Server routing test proving `/api/attention/v1` wins before project routing
  and works with no selected project.
- Refresh convergence tests after mutation/stale error and explicit assertion
  that no Attention SSE dependency exists.
- `go test ./...` for Serve, MCP, contracts, and source-service regressions.
