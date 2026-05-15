---
title: Cross-Repo Peering is Local-First; Cloud Augments
type: decision
status: proposed
created: 2026-05-15
tags: [peering, federation, local-first, architecture, decision]
relations:
  - target: cross-repo-peering
    kind: decided-in
  - target: graph-memory-federation
    kind: related
  - target: hero-cloud-split
    kind: related
---

# Cross-Repo Peering is Local-First; Cloud Augments

## Decision

The cross-repo peering primitives (peer manifest, handoff record,
peer events) are designed and delivered as **local-filesystem
operations against sibling `.hero/` directories** first. The cloud
transport (graph-memory-federation push API) is strictly additive
and gated on cloud configuration.

The seam: if a peer is reachable in the local `hero repos` registry,
operations go through the filesystem. If the peer is not local but
cloud is configured, operations go through the cloud. If neither,
the operation fails with a clear error.

## Context

A Hero workspace can register sibling repos via `hero repos add`
(already shipped). The dominant developer setup for the project's
own developer is three sibling repos on one laptop — backend, web
client, desktop client — with no cloud login required for daily
work.

A cloud-mediated federation already has a planned home in
`graph-memory-federation` (per-repo team graphs, unit-level join
graphs, partitioned sync API). That spec assumes cloud is the
substrate.

The risk: designing peering as a cloud feature would either (a) lock
out the laptop-only case or (b) require a local-only fallback bolted
on later, where the wire shapes don't match.

## Options considered

1. **Cloud-first, local fallback later.** Push the design through
   the cloud federation. Solo laptop devs would need either a local
   cloud daemon or to live without peering.
   - Pros: aligns with the cloud feature ambitions; one transport
     to maintain.
   - Cons: punishes the dominant solo case; lock-in to cloud for a
     feature that doesn't need it; contradicts hero's "useful
     without an account" stance.

2. **Local-first, cloud augments.** All v1 primitives operate on
   sibling `.hero/` dirs through `hero repos`-resolved paths. Cloud
   transport is additive and uses the same wire shapes (defined in
   `contracts/peering/`).
   - Pros: zero ceremony for the dominant case; the contracts seam
     means cloud later is a transport swap, not a redesign;
     dogfooding happens immediately on the developer's own three-
     repo setup.
   - Cons: two write paths to maintain (filesystem + cloud) once
     cloud lands; trust model is "anyone with write access to peer's
     `.hero/` can fake provenance" — accepted explicitly for v1.

3. **Hybrid daemon.** A long-running local process that watches
   sibling `.hero/` dirs and proxies events.
   - Pros: faster fan-out on receive.
   - Cons: install/uninstall/lifecycle complexity for a problem
     the user can solve by checking `hero queue` at session start.

## Decision

Option 2. Local-first, cloud augments.

## Consequences

- Wire shapes (manifest, handoff record, peer events) live in
  `contracts/peering/` so they're identical across transports.
- `hero handoff` performs two filesystem writes (receiver-side
  spec first, then originator-side status) when both repos are
  local. No daemon, no network.
- Cloud transport is delivered as a strictly additive Phase 4 of
  the cross-repo-peering spec, after the local primitives are
  proven.
- Multi-machine peering is *not* supported via SSH or LAN; if the
  peer isn't on the same disk, use cloud or wait.
- Provenance forgery is accepted as a local-only risk. Cloud mode
  can sign events using the same GitHub App identity the
  federation sync uses.
