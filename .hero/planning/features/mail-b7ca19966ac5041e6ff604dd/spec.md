---
title: "Mail thread lifecycle contract and state"
slug: mail-b7ca19966ac5041e6ff604dd
type: feature
status: delivering
created: 2026-08-22
domain: engineering
size: large
priority: critical
surface: hero-core
tags: [mail, lifecycle, contracts, persistence, migration]
size_ack: large

delivery_method: manual
---
# Mail thread lifecycle contract and state

## Goal

Introduce a backward-compatible, revisioned Project Mail thread lifecycle in
which message read state and thread `open / resolved / archived` state are
orthogonal, explicit actions are idempotent, old receipts migrate without data
loss, and unresolved threads can never become archive-eligible merely because
time passed.

## Kickoff

Adds a portable Mail thread contract and durable source-owned lifecycle state
without changing Mail-read v1 behavior.

**Status:** planning — accepted from Hero Code as Wave 1 of the Mail lifecycle initiative.

**Pick up at:** implement the contract package, then wire storage migration and revisioned lifecycle actions into Mail.

→ `/deliver mail-b7ca19966ac5041e6ff604dd`

**Files:** `contracts/attention/mailthread/contract.go`, `internal/attention/mail/store.go`, `internal/attention/mail/thread.go`
**Skip:** do not infer thread identity or completion from Mail content.

## Background

Hero owns immutable Mail envelopes and per-message receipts. This receiver-owned
slice adds authoritative thread state while leaving the existing Mail-read v1
contract compatible.

## Design

- Keep envelopes immutable and receipts per message.
- Store thread state beside existing mailbox data, keyed by project peer and
  explicit thread ID.
- Give legacy unthreaded Mail a stable message-backed identity.
- Expose revisioned, idempotent lifecycle actions and publish a vendorable
  conformance bundle.

## Changes

1. Add a versioned Mail thread contract, validation, schemas, and golden fixtures.
2. Add durable thread state beside existing Mail envelope/receipt storage.
3. Migrate existing mailboxes deterministically on read and write boundaries.
4. Extend Mail service capabilities with mark-unread and lifecycle actions while preserving Mail-read v1.

## Acceptance Criteria

- **AC-1:** THE SYSTEM SHALL persist message read state independently from thread lifecycle state.
- **AC-2:** WHEN Mail lacks an explicit thread ID THE SYSTEM SHALL assign a stable message-backed thread identity without inspecting subject or body text.
- **AC-3:** WHEN a lifecycle action carries the current revision and a new idempotency key THE SYSTEM SHALL apply it once and return the authoritative thread state.
- **AC-4:** IF a lifecycle action is stale, duplicated with different input, unknown, or malformed THEN THE SYSTEM SHALL reject it without partially mutating thread or receipt state.
- **AC-5:** WHEN an existing v1 mailbox is first read or mutated THE SYSTEM SHALL preserve its envelopes and receipts and SHALL migrate it deterministically to valid thread state.
- **AC-6:** WHILE a thread is open THE SYSTEM SHALL omit archive eligibility regardless of message age or read state.
- **AC-7:** THE SYSTEM SHALL persist only lifecycle identity, revision, provenance, outcome, and timestamps and SHALL NOT copy Mail content into thread state.
- **AC-8:** WHEN the contract bundle is published THE SYSTEM SHALL provide golden fixtures for every state, action, migration, error, and compatibility path.

## Boundaries

- No desktop UI, typed peering outcome reducer, timed reconciliation, projection,
  or physical Mail deletion.
- No subject/body/model inference and no user-state mutation from a plain read.

## Validation

- Contract and golden bundle tests.
- Store migration, stale write, idempotency, malformed input, and atomic-write tests.
- Existing Mail-read v1, Mail, query, Serve API, and MCP compatibility suites.
