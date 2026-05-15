---
title: Peer Manifest Uses an Explicit Publish Boundary
type: decision
status: proposed
created: 2026-05-15
tags: [peering, conventions, least-authority, decision]
relations:
  - target: cross-repo-peering
    kind: decided-in
---

# Peer Manifest Uses an Explicit Publish Boundary

## Decision

A repo's peer manifest (`.hero/peer-manifest.yaml`) lists *only*
conventions explicitly marked peer-surface — either by `peer: true`
in the convention frontmatter or by a glob in
`hero.json: { "peering": { "publish_conventions": [...] } }`. The
default publish set is **empty**. Internal-only conventions never
appear in the manifest.

## Context

When a peer loads conventions across the boundary, it's loading
context the model will treat as authoritative. Dumping the whole
peer knowledge base is fast but produces "convention starvation by
flood" — the consumer drowns in irrelevant rules and the signal
gets lost.

The opposite extreme — manifest the peer queries on demand for any
named convention — punts the publish-boundary decision to the
consumer, which is the wrong owner. The peer author knows which of
their conventions govern public surfaces (API shapes, error
envelopes, auth flow) vs. which are internal style guides.

## Options considered

1. **All conventions are peer-visible.** Manifest lists everything.
   - Pros: simple; no flag needed.
   - Cons: drowns the consumer; leaks internal style as if it were
     contract; encourages the model to apply rules outside their
     intended scope.

2. **Consumer queries any convention on demand.** Manifest is just
   an index; consumer picks what to load.
   - Pros: maximum flexibility on the consumer side.
   - Cons: consumer doesn't know the peer author's intent; same
     drowning risk; no signal about which conventions are stable
     enough to publish as a contract.

3. **Explicit publish boundary, default-empty.** Peer author marks
   peer-surface conventions; everything else stays internal.
   - Pros: principle of least authority applied to context; peer
     author owns the "this is contract-grade" decision; consumer
     sees a focused, intentional set.
   - Cons: requires the peer author to mark conventions; default-
     empty means peering is inert until something is explicitly
     published (the right cost).

## Decision

Option 3. Explicit publish boundary, default-empty.

## Consequences

- A convention's `peer: true` frontmatter flag (or matching
  publish glob) is the gate. Convention-writing skill grows a
  "peer-surface" subsection covering when to flag and the quality
  bar that flagged conventions must meet.
- `hero check` lints peer-surface conventions for the higher
  quality bar (concrete `## Examples` and `## Anti-patterns`
  referencing real identifiers, not just prose).
- Adding a convention to the peer manifest is a deliberate act
  with a review obligation. Removing one is also deliberate and
  surfaceable as a contract-style breaking change.
- `hero relevant --peer` is the consumer-side reader; it never
  reaches into the peer's `knowledge/` dir beyond the manifest.
- Future v2 may add an `--include-unpublished` escape hatch with
  a warning. v1 has no escape hatch — the publish boundary is the
  only path.
