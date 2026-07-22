---
title: "Project Mail Triage and Provenance — From Signal to Explicit Work"
slug: project-mail-triage-and-provenance
type: feature
status: planning
domain: engineering
priority: high
size: large
horizon: next
created: 2026-07-20
parent: durable-attention
depends-on: [project-mail-core, personal-focus-core, hero-idea-primitive-core]
conflicts-with: [deferred-work-suggestion-contract]
tags: [mail, intake, provenance, graph, mcp]
---

# Project Mail Triage and Provenance — From Signal to Explicit Work

## Context

Project Mail deliberately delivers only an untrusted signal. Useful work begins
when the recipient reads, acknowledges, dismisses, adds it to Personal Focus, or
promotes it through Hero's existing Intake/Spec path. Those decisions need
idempotency and provenance without turning Mail receipt state into spec status.

`internal/cli/intake.go` currently owns both Cobra presentation and the capture/
promotion filesystem logic. This feature needs the same authority from CLI,
MCP, and the later Attention API, so the domain operation must be extracted
without changing Intake behavior.

## Goal

Add explicit Mail triage actions, reuse Intake promotion as the sole work-
creation authority, preserve message/thread provenance, and surface unread Mail
through existing Hero CLI and MCP read paths.

## Kickoff

Implement triage as service calls over immutable Mail and separate receipts.
First extract Intake capture/promotion from Cobra into a reusable internal
service with parity tests. Make promotion and Add to Today idempotent, return
authoritative artifact/Focus references, emit provenance, and expose only
project-scoped MCP capabilities here; the global Hero Code API belongs to the
read-model child.

## Problem

If each surface writes specs or Focus directly, duplicate retries and partial
failures will create conflicting commitments. If Mail state is used as spec
state, communication and work ownership become inseparable. If source identity
is copied only into prose, `hero why` cannot trace why work exists.

## Design

### Receipt state machine

Extend Mail receipts with independent fields: `read_at`, `acknowledged_at`,
`dismissed_at`, `promoted_artifact`, and `focus_item_id`. Envelopes remain
immutable. Read and acknowledge are monotonic. Dismiss hides Mail from the
active inbox but does not delete it, create work, or imply acknowledgement.
Promote and Add to Today may be performed on read/acknowledged/dismissed Mail;
they do not silently alter other receipt fields.

Each action takes message ID, expected row revision, and idempotency key. A stale
revision returns the authoritative current row. Exact replay returns the prior
result; conflicting replay fails. Missing messages and unsupported actions are
structured errors.

### Intake authority extraction

Move storage logic from `internal/cli/intake.go` into a new `internal/intake`
package with `Capture`, `Promote`, `Reject`, and `Resolve` operations. Cobra
becomes an adapter. Preserve existing directories, templates, statuses,
`promoted_to`, and `derived_from` behavior. Add optional source provenance to
capture so Mail promotion creates an Intake carrying a typed `mail:<message-id>`
source reference, then calls the same promote operation when the user requests
a Feature/Bug directly.

Promotion is a transaction with resumable steps recorded in the Mail receipt:
reserve deterministic intake slug from message ID/idempotency key, capture or
find the same Intake, promote or find the same roadmap artifact, write graph/
event provenance, then commit the authoritative `promoted_artifact` receipt.
Retry resumes from the first missing step. It never creates a second artifact.

The result includes artifact slug/type/status, source message/thread IDs,
project reference, and navigation reference `{kind:"spec", project_peer_id,
slug}`. No client constructs filesystem paths.

### Add to Today

`add_to_today` calls Focus `CreateOrGet` using origin key
`mail:<recipient-peer-id>:<message-id>`, state `today`, subject-derived title,
and a saved prompt containing the message's user-visible request plus typed
provenance. It creates a separately owned Focus item. It does not acknowledge,
dismiss, promote, or rewrite Mail. Replay returns the same Focus row.

### Provenance and events

Emit feed events for read, acknowledge, dismiss, promotion, and Add to Today.
Register Mail source nodes/relations through the existing graph ingestion path
so `hero why <artifact>` traverses artifact → Intake → Mail source, including
sender/recipient peer IDs and thread identity but not the full body. The
`intake-capture-loop` must recognize the source key and skip re-capturing mail
already represented by Intake or a promoted artifact.

### Existing read surfaces

- `hero resume` and `hero status` show the local project's unread count and up
  to five oldest unread subjects with stable message IDs; `--json` includes a
  typed summary.
- Add project-scoped MCP tools `hero_mail_list`, `hero_mail_show`, and
  `hero_mail_action`. The action tool accepts only advertised action IDs and
  structured inputs; it delegates to the same triage service.
- The global combined snapshot/API is intentionally deferred to
  `attention-read-model-v1`.

## Changes

1. Extract `generateIntakeContent`, resolve/capture/promote/reject operations
   from `internal/cli/intake.go` into `internal/intake/service.go` and
   `repository.go`, retaining Cobra output compatibility.
2. Extend `internal/attention/mail/store.go` receipt persistence and add
   `internal/attention/mail/triage.go` for revisioned, idempotent actions.
3. Add `internal/attention/mail/promotion.go` adapters over `internal/intake`,
   Focus `CreateOrGet`, graph ingestion, and feed events.
4. Extend `internal/cli/mail.go` with `read`, `dismiss`, `promote --type
   intake|feature|bug`, and `add-to-today`; keep `reply` and `ack` delegated to
   the same service.
5. Extend resume/status view-data builders and JSON types at their existing
   aggregation points with bounded unread Mail summaries.
6. Add Mail tools to `internal/serve/mcp_tools_def.go`, route them through
   `internal/serve/mcp_dispatch.go`, and implement handlers in
   `internal/serve/mcp_tools_mail.go`.
7. Extend graph/event vocabulary and `hero why` fixture coverage for typed Mail
   source provenance without storing message bodies in graph nodes.
8. Update `intake-capture-loop` guidance/implementation to deduplicate by the
   canonical Mail source reference.

## Acceptance Criteria

- **AC-1:** WHEN Mail is read, acknowledged, or dismissed, THE SYSTEM SHALL atomically update only its receipt, preserve the immutable envelope, and return the authoritative revisioned state.
- **AC-2:** WHEN promotion is requested with a new idempotency key, THE SYSTEM SHALL create or reuse one Intake/roadmap artifact through `internal/intake` and persist typed Mail provenance.
- **AC-3:** WHEN a partially completed promotion or exact request is retried, THE SYSTEM SHALL resume or return the same artifact without creating a duplicate Intake or spec.
- **AC-4:** WHEN promotion succeeds, THE SYSTEM SHALL return artifact identity, project reference, source message/thread identity, and a client-safe navigation reference.
- **AC-5:** WHEN Add to Today is requested repeatedly for one Mail message, THE SYSTEM SHALL return one linked Focus item in `today` without changing acknowledgement, dismissal, or promotion state.
- **AC-6:** WHEN `hero why` traverses a promoted Mail artifact, THE SYSTEM SHALL reach its Intake and Mail source identity without exposing the Mail body in graph metadata.
- **AC-7:** WHEN resume or status is rendered for a project, THE SYSTEM SHALL include a bounded deterministic unread Mail summary and preserve all existing output fields.
- **AC-8:** WHEN an MCP client lists Mail, reads Mail, or dispatches an advertised triage action, THE SYSTEM SHALL delegate to the shared service and return contract-shaped success or structured failure.
- **AC-9:** IF a row revision is stale or the message/action is missing or unsupported THEN THE SYSTEM SHALL make no mutation and return the current state when available.
- **AC-10:** WHILE triaging Mail, THE SYSTEM SHALL NOT execute message content, infer acceptance, alter spec status directly, or create work outside Intake and Focus authorities.

## Boundaries

- No Hero Code UI or global Attention HTTP snapshot.
- No automatic promotion, auto-response, runtime execution, policy engine, or
  mail-driven spec status transitions.
- No new request/idea spec type and no duplication of Intake templates.
- No Mail/Focus lifecycle merger.

## Risks

- Promotion spans file, graph, event, and receipt authorities; deterministic
  IDs and resumable steps are required because the filesystem has no global
  transaction.
- Refactoring Intake risks CLI drift; characterization tests must land before
  moving the implementation.
- Resume/status output can become noisy; summaries are bounded and bodies are
  excluded.
- Dismiss plus later promotion may surprise consumers; the contract keeps these
  orthogonal and returns each field explicitly.

## Validation

- Characterization tests for every existing Intake CLI operation before and
  after extraction.
- `go test ./internal/intake/... ./internal/attention/mail/... ./internal/cli/... ./internal/serve/...`
- Failure-injection tests at every promotion step followed by idempotent retry.
- Graph traversal test for artifact → Intake → Mail and feed event assertions.
- MCP tests for advertised actions, stale revisions, missing sources, and exact
  promotion/Add-to-Today replay.
- `go test ./...` for resume, status, graph, feed, Intake, Focus, and MCP regressions.
