---
title: "Project Mail Core — Local Asynchronous Project Messaging"
slug: project-mail-core
type: feature
status: planning
domain: engineering
priority: high
size: large
horizon: next
created: 2026-07-20
parent: durable-attention
depends-on: [durable-attention-contracts]
conflicts-with: [personal-focus-core]
tags: [mail, peering, local-first, cli]
---

# Project Mail Core — Local Asynchronous Project Messaging

## Context

Hero's current peering substrate already provides stable `peer_id` values,
aliases, sibling path resolution, and peer manifests, but communication is
coupled either to receiver-side spec writes or a Claude subprocess. A project
needs a lighter asynchronous inbox that works on one machine with no harness,
daemon, or recipient work-tree mutation.

## Goal

Implement a private user-state mailbox and local sibling transport with
immutable, versioned messages plus CLI operations to send, inspect, reply, and
acknowledge mail safely and idempotently.

## Kickoff

Implement Project Mail on the durable-attention contracts and injected global
state root. Reuse peer identity, manifest, and alias resolution, but do not call
the peer-call launcher or write into the recipient repository. Make delivery an
atomic immutable-envelope write, keep receipts separate, expose stable JSON
output, and prove retries cannot duplicate messages.

## Problem

`internal/peering.Handoff` writes a spec scaffold into the receiver repository,
while `internal/peering.Call` launches `claude` and parses a result fence.
Neither is a basic communication primitive. They conflate sending a request,
accepting work, and executing an agent, and they can dirty another checkout
before its owner has agreed to anything.

## Design

### Mailbox layout and ownership

Add `internal/attention/mail`. The store receives an explicit attention state
root and uses:

```text
mail/
  boxes/<recipient-peer-id>/messages/<message-id>.json
  boxes/<recipient-peer-id>/receipts/<message-id>.json
  outbound/<sender-peer-id>/<message-id>.json
```

An envelope is immutable. Read/ack state lives in a receipt guarded by the
envelope revision. Sender outbound records contain delivery identity and time,
not a second mutable copy of recipient state. Files are written to a sibling
temporary file, `fsync`ed, renamed atomically, and created with private
permissions. A store lock scoped to the mailbox protects read-modify-write
receipt updates; tests use an injected lock/root.

### Addressing and delivery

`Service.Send` accepts a recipient alias plus message input. It loads the local
config, resolves the alias through `Config.ResolveRepoPath`, reads the peer
manifest with `internal/peering.ReadPeerManifest`, and derives both sender and
recipient IDs from authoritative project configuration. The recipient path is
used only to verify identity/reachability; delivery writes to the current
user's global mailbox root keyed by recipient `peer_id`.

The caller supplies or receives a generated message ID and idempotency key.
Replaying the same key with byte-equivalent normalized input returns the
original receipt. Reusing it with different content fails with
`idempotency_conflict`. An existing message ID with different bytes is never
overwritten.

### Threads and replies

A root message has `thread_id == message_id` and no `in_reply_to`. A reply keeps
the original thread ID, addresses the original sender, and names a message that
exists in the responder's mailbox. Missing or cross-thread reply targets fail.
V1 supports `question`, `request`, `response`, and `notice` as documented values
while storing unknown kinds as raw strings for compatibility.

### Receipts

`show` marks a message read unless `--no-mark-read` is passed. `ack` records a
single acknowledgement timestamp and optional short note; replay is
idempotent. Core does not define dismiss, promote, or Add to Today. Those are
triage operations in the dependent spec.

### CLI

Add `hero mail` with:

- `send <peer> --subject ... [--body-file -|path] [--kind ...] [--json]`
- `inbox [--project <alias|peer-id>] [--unread] [--json]`
- `show <message-id> [--no-mark-read] [--json]`
- `reply <message-id> --body-file ... [--subject ...] [--json]`
- `ack <message-id> [--note ...] [--json]`

Interactive body text may come from stdin; secrets and message bodies are never
accepted as positional argv values. Human output prints IDs and subjects but
does not echo full bodies after send. JSON uses contract DTOs and stable error
codes.

## Changes

1. Add `internal/attention/mail/store.go`, `lock.go`, and `store_test.go` for
   private layout, atomic writes, immutable envelopes, receipts, listing, and
   injected roots/time.
2. Add `internal/attention/mail/service.go` and `service_test.go` for validation,
   peer addressing, delivery idempotency, threads, replies, and acknowledgement.
3. Reuse `internal/peering/identity.go`, `resolve.go`, and `manifest.go`; expose a
   small public identity resolver only if the existing functions cannot supply
   both peer IDs without duplicating parsing.
4. Add `internal/cli/mail.go` and `mail_test.go`, then register `mailCmd` in
   `internal/cli/root.go`.
5. Add CLI JSON result/error structs only where the contract DTOs do not already
   cover command metadata.
6. Add an end-to-end test with two temporary Hero projects proving delivery
   changes only the global state root and leaves both git work trees unchanged.
7. Update CLI reference documentation and generated shell completion snapshots
   for the new command family.

## Acceptance Criteria

- **AC-1:** WHEN a sender delivers valid mail to a configured local peer, THE SYSTEM SHALL atomically persist one immutable envelope in the recipient peer's global mailbox and one outbound receipt for the sender.
- **AC-2:** WHEN the same idempotency key and normalized input are retried, THE SYSTEM SHALL return the original delivery result without creating another message.
- **AC-3:** IF an idempotency key or message ID is reused with different content THEN THE SYSTEM SHALL return `idempotency_conflict` and preserve the original envelope.
- **AC-4:** WHEN a root message or reply is stored, THE SYSTEM SHALL preserve valid thread and reply identity and reject missing or cross-thread reply targets.
- **AC-5:** WHEN `hero mail inbox`, `show`, `reply`, or `ack` is used with `--json`, THE SYSTEM SHALL emit contract-shaped JSON with stable IDs and structured errors.
- **AC-6:** WHEN a message is shown or acknowledged, THE SYSTEM SHALL update a separate receipt atomically and SHALL NOT rewrite the envelope.
- **AC-7:** IF a recipient alias, peer manifest, peer ID, or local path cannot be resolved THEN THE SYSTEM SHALL fail before writing recipient or outbound state.
- **AC-8:** WHILE delivering or reading mail, THE SYSTEM SHALL NOT invoke a model, execute message content, require Hero Serve/MCP, or write within the recipient repository.
- **AC-9:** WHEN untrusted mail violates contract limits or contains forbidden payload fields, THE SYSTEM SHALL reject it with no partial delivery.
- **AC-10:** WHEN two temporary projects exchange mail in the end-to-end test, THE SYSTEM SHALL leave both tracked work trees byte-for-byte unchanged.

## Boundaries

- No Intake/Spec promotion, dismiss, Add to Today, unread dashboard projection,
  watcher, automatic response, cloud transport, attachments, or model runtime.
- No replacement of `hero peer call` or `hero handoff` in this spec.
- No message deletion in v1; receipts may express later triage state.

## Risks

- Local path reachability does not prove a remote identity; v1 is explicitly a
  same-user, same-machine transport.
- Filesystem concurrency must not silently lose receipt changes; lock and atomic
  replacement tests are required.
- Bodies may contain secrets despite warnings. Private permissions and avoiding
  argv/log echo reduce exposure but do not classify content.
- A peer ID change creates a new mailbox identity; recovery/alias UI is outside
  v1 and the old box remains readable by explicit peer ID.

## Validation

- `go test ./internal/attention/mail/... ./internal/cli/...`
- Table tests for validation limits, malformed envelopes, thread rules, missing
  peers, idempotent replay, conflicts, and concurrent receipt updates.
- End-to-end two-project test with no `claude` executable and no running server.
- Git-status assertion proving receiver-tree non-mutation.
- `go test ./...` for peering and CLI regressions.
