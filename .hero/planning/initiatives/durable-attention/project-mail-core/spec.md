---
title: "Project Mail Core — Local Asynchronous Project Messaging"
slug: project-mail-core
type: feature
status: planning
domain: engineering
priority: high
size: medium
horizon: next
created: 2026-07-20
parent: durable-attention
depends-on: [durable-attention-contracts]
tags: [mail, peering, local-first, cli]
---

# Project Mail Core — Local Asynchronous Project Messaging

## Goal

Deliver useful project-addressed asynchronous mail without a daemon, model
launcher, MCP dependency, or receiver-side spec mutation.

## Design inputs

- Reuse `peer_id`, alias/path resolution, and peer manifests.
- Use immutable versioned envelopes with message/thread/reply IDs, sender and
  recipient project identity, kind, subject/body, timestamps, acknowledgement
  intent, and safe provenance references.
- Store recipient mail in Hero-owned local state, not tracked project files.
- Make local sibling delivery atomic and idempotent.
- Provide `hero mail send`, `inbox`, `show`, `reply`, and `ack`.

## Boundaries

No promotion to Intake/Spec, unread dashboard projection, hosted/cloud transport,
attachments, watcher, automatic response, or model execution.

## Acceptance shape

The `/design` pass must specify envelope validation, local store layout, atomic
delivery and duplicate behavior, thread/reply semantics, CLI output/JSON, error
handling, and focused tests for delivery, retries, malformed mail, and untrusted
content.

## Kickoff

Start from existing peering identity and path resolution, but do not inherit
`handed_off` spec statuses, two-repository spec writes, the Claude subprocess,
or result-fence parsing.
