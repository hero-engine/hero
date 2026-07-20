---
title: "Attention Read Model v1 — Consumer-Safe Mail and Focus Projection"
slug: attention-read-model-v1
type: feature
status: planning
domain: engineering
priority: medium
size: medium
horizon: next
created: 2026-07-20
parent: durable-attention
depends-on: [personal-focus-core, project-mail-triage-and-provenance]
tags: [attention, projection, api, schema, hero-code]
---

# Attention Read Model v1 — Consumer-Safe Mail and Focus Projection

## Goal

Publish one provider-neutral read projection through which Hero Code and other
clients can render Mail and Focus together without sharing their write models or
reimplementing lifecycle rules.

## Design inputs

- Rows expose stable source ID/kind, project reference, display fields,
  timestamps, unread/Today state, provenance summary, and explicitly supported
  actions.
- Snapshots expose contract version, snapshot identity/revision, server
  timestamp, deterministic ordering, and an opaque refresh token if needed.
- Action descriptors expose stable ID, label/style, confirmation, supported
  input schema, row precondition, and idempotency requirements.
- Action results expose authoritative updated state or invalidation plus
  structured stale, unsupported, missing, validation, incompatible-version, and
  unavailable errors.
- Promotion results expose artifact identity and a client-safe navigation
  reference.
- Clients act only through advertised capabilities and never infer actions from
  status strings.
- Publish versioned JSON Schema and checked-in golden fixtures.
- Unknown additive fields decode safely.
- Define refresh semantics and stale-action responses. Snapshot refresh on
  mount/foreground/reconnect/mutation is the v1 floor; streaming events are
  optional and must justify their ordering, replay, duplicate, expiry, and gap
  recovery contract.
- Reuse existing projection/source patterns rather than creating a second
  dashboard inbox architecture.

## Boundaries

No client UI, direct store access, generic Attention write endpoint, remote push
notifications, or Swift-only contract variants.

## Acceptance shape

The `/design` pass must define row and action schemas, ordering/pagination,
version negotiation, errors, golden fixtures, refresh behavior, and
compatibility tests that Hero Code can consume before the whole initiative is
delivered. Golden coverage must include empty and mixed projections, threaded
unread Mail, every Focus bucket, missing projects, absent and unknown actions,
unknown additive fields/source kinds, structured action failures, promotion
results, and deferred suggestions before/after acceptance.

## Kickoff

Inspect current Hero projection and MCP/API contracts, schema/golden generation,
and existing inbox providers. Coordinate field needs through
`hero-code-spec-out-prompt.md`; do not let the consumer become contract
authority.
