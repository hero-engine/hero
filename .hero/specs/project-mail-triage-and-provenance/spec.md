---
title: "Project Mail Triage and Provenance — From Signal to Explicit Work"
slug: project-mail-triage-and-provenance
type: feature
status: completed
domain: engineering
priority: high
size: large
horizon: next
created: 2026-07-20
parent: durable-attention
depends-on: [project-mail-core, personal-focus-core, hero-idea-primitive-core]
conflicts-with: [deferred-work-suggestion-contract]
tags: [mail, intake, provenance, graph, mcp]
delivery_method: manual
completed_at: 2026-07-23T03:23:06Z
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

Adds explicit Project Mail triage, promotion, and Add-to-Today actions with
retry-safe receipts and traceable provenance.

**Status:** completed — all 10 criteria passed, the cold audit returned SHIP,
and Hero archived the verified delivery.

**Pick up at:** use this completed contract as the project-scoped foundation
when building the combined Attention read model.

→ `.hero/planning/initiatives/durable-attention/attention-read-model-v1/spec.md`

**Files:** `internal/attention/mail/triage.go`,
`internal/attention/mail/promotion.go`, `internal/intake/service.go`,
`internal/serve/mcp_tools_mail.go`

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

## Completion Ledger

Implementation keeps Mail envelopes immutable, moves all work creation through
the extracted Intake and existing Focus authorities, and records every
cross-authority promotion step in the revisioned receipt. Validation included
the full Go suite, focused failure-injection and CLI/MCP exercises, graph
traversal, and the six-target harness propagation matrix.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Read, acknowledge, and dismiss update only revisioned receipts | DONE | `internal/attention/mail/triage.go:51` and `triage_test.go:43` — CAS receipt actions preserve the immutable envelope and return authoritative rows |
| 2 | Promotion reuses Intake/roadmap authority with typed Mail provenance | DONE | `internal/attention/mail/promotion.go:20` and `internal/intake/service.go:66` — deterministic source-aware capture and promotion use the extracted authority |
| 3 | Partial and exact promotion retries never duplicate work | DONE | `internal/attention/mail/promotion.go:61` and `triage_test.go:138` — receipt-recorded steps resume after injected failures at every boundary |
| 4 | Promotion returns artifact, project, source, and navigation identity | DONE | `internal/attention/mail/triage.go:40` and `promotion.go:153` — result carries artifact, project, message/thread, and client-safe spec navigation |
| 5 | Add to Today is idempotent and orthogonal | DONE | `internal/attention/mail/promotion.go:162` and `triage_test.go:71` — Focus `CreateOrGet` returns one Today item without changing other triage fields |
| 6 | Why traverses artifact → Intake → Mail without body metadata | DONE | `internal/spec/graph_ingest.go:175` and `triage_test.go:177` — body-free `MailSource` plus `derived_from`/`mail_source` traversal is asserted |
| 7 | Resume/status expose bounded deterministic unread summaries | DONE | `internal/cli/brief.go:137`, `status.go:288`, and `triage_test.go:110` — text/JSON paths preserve existing data and add five oldest IDs/subjects |
| 8 | Project-scoped MCP Mail tools return shaped success/failure | DONE | `internal/serve/mcp_tools_mail.go:51` and `mcp_tools_mail_test.go` — advertised action descriptors and JSON failures cover list/show/action |
| 9 | Stale, missing, and unsupported requests do not mutate | DONE | `internal/attention/mail/store.go` CAS stale result, `triage.go:58`, and MCP structured-failure tests cover authoritative current rows |
| 10 | Mail content is never executed and work stays in Intake/Focus | DONE | `internal/intake/service.go:173` quotes untrusted frontmatter; promotion and Today adapters call only Intake/Focus, with no spec-status side channel |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Extract Intake storage authority from Cobra | DONE | `internal/intake/service.go`, `repository.go`, and retained `internal/cli/intake_test.go` characterization coverage |
| 2 | Add revisioned receipt persistence and triage service | DONE | `contracts/attention/mail.go`, `internal/attention/mail/store.go`, and `triage.go` |
| 3 | Add promotion, Focus, graph, and feed adapters | DONE | `internal/attention/mail/promotion.go`, resumable event handling, and body-free graph ingestion |
| 4 | Add Mail CLI triage commands and shared ack/read paths | DONE | `internal/cli/mail.go` plus end-to-end command exercise in `mail_test.go:16` |
| 5 | Extend resume/status text and JSON aggregation | DONE | `internal/cli/brief.go`, `status.go`, and bounded summary service |
| 6 | Add MCP definitions, dispatch, and Mail handlers | DONE | `internal/serve/mcp_tools_def.go`, `mcp_dispatch.go`, `mcp_tools_mail.go`, and tool inventory coverage |
| 7 | Extend graph/event vocabulary and why coverage | DONE | `internal/feed/feed.go`, `internal/graph/edge.go`, `internal/traversal/why.go`, and traversal assertions |
| 8 | Deduplicate canonical Mail sources in capture guidance/implementation | DONE | `internal/intake/service.go:70`, `core/skills/auto-knowledge-capture/SKILL.md`, and six-target propagation test `internal/install/content_test.go:103` |

### Exercise-the-feature check

- [x] User-visible behavior was exercised end-to-end: `go test ./internal/cli -run '^(TestIntake|TestMail)' -count=1` sent Mail, rendered status/resume JSON, acknowledged, dismissed, promoted, added to Today, replied, and checked structured errors.
- [x] Full regression suite passed: `GOCACHE=/private/tmp/hero-mail-gocache go test ./...`.
- [x] Cross-harness guidance propagation passed for `opencode|cursor|claude|copilot|codex|generic`: `go test ./internal/install -run '^TestAllTargetsInstallMailSourceDedupGuidance$' -count=1`.

### Excellence Bar self-check

- [x] Yes — the implementation is authority-preserving, retry-safe at every external write boundary, explicit about structured failures, hardened against untrusted Mail frontmatter injection, fully traversable through provenance, and covered end-to-end plus by the full repository suite.
