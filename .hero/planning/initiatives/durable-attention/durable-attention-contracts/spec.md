---
title: "Durable Attention Contracts — Ownership, Storage, Compatibility, and Trust"
slug: durable-attention-contracts
type: feature
status: planning
domain: engineering
priority: critical
size: medium
horizon: next
created: 2026-07-20
parent: durable-attention
tags: [attention, contract, storage, security]
---

# Durable Attention Contracts — Ownership, Storage, Compatibility, and Trust

## Goal

Define the versioned, provider-neutral contracts and invariants that let Project
Mail and Personal Focus evolve independently while supporting one consumer read
projection. Lock storage authority, compatibility behavior, identity references,
and the untrusted-input boundary before either durable store is implemented.

## Design inputs

- Mail is project-addressed and Focus is user-owned/global.
- There is no common Attention write model or common lifecycle.
- A combined read projection may share stable IDs, provenance references,
  project references, timestamps, display fields, and advertised actions.
- Incoming mail must not dirty the recipient project's tracked source tree.
- Unknown additive fields are tolerated; contract versions remain explicit.
- Credentials, tracked attachments, and executable payload semantics are
  forbidden.
- Hero Code requires raw-string source/action values, stable project and
  provenance references, row revisions/preconditions, structured action
  failures, and a canonical fixture manifest with checksums.

## Design questions

- Choose the Hero-owned local storage roots and authority for Mail and Focus.
- Define stable IDs, schema/version fields, timestamps, project/peer references,
  provenance references, and forward-compatible decoding.
- Define trust levels, sender validation, size limits, and what can be committed
  or exported safely.
- Define how JSON Schema and golden fixtures are generated and versioned for
  Hero Code and other clients.
- Settle the canonical user-global service/API/MCP boundary that works before a
  project is open; publish one transport contract, not parallel client paths.
- Decide whether incremental events are justified for v1. The minimum is a
  versioned authoritative snapshot with an opaque revision and explicit refresh
  behavior; streaming adds ordering, replay, expiry, and gap-recovery obligations.

## Boundaries

No store implementation, transport, UI, model suggestion rendering, promotion,
or runtime dispatch. Do not create a generic mutable `AttentionItem`.

## Acceptance shape

The `/design` pass must leave two distinct write contracts, one shared read
vocabulary, explicit storage/trust decisions, additive-field compatibility, and
fixture ownership clear enough that both core implementations can proceed
without reopening the seam.

## Kickoff

Inspect existing contracts, local-state conventions, peer identity, graph/event
IDs, JSON Schema generation, and client fixture patterns. Design the smallest
contract foundation that preserves separate Mail and Focus ownership.
