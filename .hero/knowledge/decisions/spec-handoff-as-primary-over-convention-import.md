---
title: Spec Handoff is the Primary Cross-Repo Path; Convention Import Supports
type: decision
status: proposed
created: 2026-05-15
tags: [peering, handoff, conventions, mission-alignment, decision]
relations:
  - target: cross-repo-peering
    kind: decided-in
---

# Spec Handoff is the Primary Cross-Repo Path; Convention Import Supports

## Decision

When work in repo A touches repo B's surface, the **primary path** is
to hand the spec off to B's Hero so that B's native context
(conventions, decisions, knowledge, stack skills) drives the work.
Importing B's conventions into A's session is a **supporting
capability** for the case where the work genuinely lives in A.

This is the central design fork the user asked us to pick.

## Context

The user posed it as: "do you ask the peer to *create* its own spec
so its conventions kick in natively when it picks up the work, or
do you load the peer's conventions into the current session? Or
both, as different modes?"

Both have legitimate use cases. Picking a primary matters because it
sets where the design effort goes and what the user reaches for by
default.

## Options considered

1. **Convention import as primary.** A's session pulls B's
   conventions and acts on them. B sees the merged change later.
   - Pros: zero context switch for the developer; one session
     completes the work.
   - Cons: A's session has a degraded snapshot of B's context
     (conventions but no decisions, no broader patterns); leaky
     abstraction — A may still get details wrong; doesn't compound
     for B (B's next session learns nothing).

2. **Spec handoff as primary.** A's session concludes "this is B's
   work," routes the spec to B, and B's Hero picks it up with full
   native context.
   - Pros: mission-aligned (compounds for B); B operates in its
     real context, not a snapshot; provenance is preserved as
     part of the handoff.
   - Cons: requires a context switch for solo devs (mitigated by
     `hero queue` surfacing incoming handoffs and a fast `/resume`).

3. **Two equal modes.** Both supported as first-class with no
   default.
   - Pros: maximum flexibility.
   - Cons: design effort split; user has to choose every time;
     CLI ergonomics suffer.

## Decision

Option 2. Spec handoff is primary; convention import is the
supporting move for "this genuinely belongs in A but I need to
respect B's surface."

The deciding factor is mission-fit. Hero's mission is "make the next
agent session start smarter than the last one ended." A handoff
with provenance produces exactly that on B's side. A
convention-import-into-A's-session helps A finish faster but does
nothing for B's future sessions.

## Consequences

- The CLI headline command is `hero handoff <slug> <peer>`. The
  peer-convention flag (`hero relevant --peer <alias>`) is
  available but not the headline.
- The peer manifest is engineered so that *both* paths consume the
  same data: handoff scaffolding can copy relevant peer
  conventions into the new spec's context, and `hero relevant
  --peer` reads the same manifest.
- Documentation, slash commands, and agent skills are written so
  that "consider handing off" is the question delivery leads ask
  first when a change crosses a peer boundary.
- Convention import remains useful and is not deprecated. It's
  positioned as the answer when the work genuinely lives in A
  (e.g., a serializer in A that produces a payload B parses).
- Open follow-on: how should `/diagnose` and `/design` *suggest*
  handoff automatically when a spec's Changes section names files
  the peer publishes? Deferred to v2 auto-detection (cross-repo-
  peering Phase 5).
