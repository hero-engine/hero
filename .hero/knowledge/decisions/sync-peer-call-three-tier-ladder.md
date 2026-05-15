---
title: Sync Peer Call is a First-Class Mode with a Three-Tier Ladder
type: decision
status: proposed
created: 2026-05-15
tags: [peering, sync-peer-call, autonomy, mission-alignment, decision]
relations:
  - target: cross-repo-peering
    kind: decided-in
  - target: spec-handoff-as-primary-over-convention-import
    kind: refines
---

# Sync Peer Call is a First-Class Mode with a Three-Tier Ladder

## Decision

Add a top-level interaction mode alongside async handoff and
convention import: **sync peer call** (also "peer RPC"). A session
in repo A can invoke a subagent session in repo B's workspace with
B's full Hero context loaded, run a prompt, and receive a structured
result.

Three tiers of increasing autonomy:

1. **Advisory (v1)** — B investigates, returns findings text.
   **Never writes.**
2. **Spec-out (v1)** — B runs its `/design` flow, saves a spec to
   B's `.hero/planning/`, returns the slug. Integrates with the
   handoff lifecycle.
3. **Full delivery (v2)** — B designs and delivers, returns
   commit/PR ref. Gated by approval prompt + turn/token budget.

CLI shape (final naming flexible):

```
hero peer call <alias> --mode=<advisory|spec-out|full> "<prompt>"
               [--budget-turns N] [--budget-tokens N]
               [--related-spec <slug>] [--reason <text>]
```

## Context

The original cross-repo-peering draft had two modes: async handoff
and convention import. Real usage surfaced a third pattern that
neither covers:

- The user wants an **authoritative answer right now** from B about
  B's code, without committing to a handoff and without trusting a
  convention snapshot.
- Examples: "is there an existing endpoint for this?", "does this
  API change break you?", "what's your convention for X?"
- Today these require a human to switch repos and start a new
  session.

A second observation: spec-out (mode 2) **resolves the original
convention-loading fork**. The draft pondered whether A should load
B's conventions or ask B to design. Asking B to design (via a sync
call) is the cleanest answer — B's rules apply natively, and the
output persists as a real spec on B's side for B's future sessions.
Convention import drops to a fallback for the genuinely-stays-in-A
case.

## Options considered

1. **Don't add sync calls; rely on handoff + convention import.**
   - Pros: smaller surface.
   - Cons: forces handoff even when A just wants a fact; misses
     the spec-out solution to the convention-loading fork; the
     pattern stays manual.

2. **One sync mode (advisory only).**
   - Pros: lowest-risk slice.
   - Cons: misses the durable artifact benefit of spec-out; doesn't
     resolve the convention-loading fork.

3. **Two sync tiers (advisory + spec-out), full-delivery deferred.**
   - Pros: covers fact-finding and the headline answer to
     conventions-vs-handoff; full-delivery is genuinely high-risk
     and benefits from real budget primitives.
   - Cons: more design surface in v1.

4. **All three tiers in v1.**
   - Pros: completes the ladder up-front.
   - Cons: full-delivery needs approval-prompt and budget UX that
     hasn't been dogfooded; risk of half-baked v1.

## Decision

Option 3. Advisory + spec-out in v1; full delivery deferred to v2
behind a gate.

## Consequences

- `hero peer call` is a top-level command alongside `hero handoff`.
- Spec-out is the **headline** answer to the original
  convention-loading fork. The cross-repo-peering spec reframes
  convention import as a fallback.
- Sync calls integrate with the handoff lifecycle:
  - Advisory with `--related-spec` appends a trail entry; no
    status change.
  - Spec-out with `--related-spec` moves the calling spec to
    `awaiting_peer` and writes both sides of the trail.
- Hard budget enforcement on every call. Advisory's no-write
  promise is enforced at the filesystem boundary, not just by
  prompt.
- A structured result block is appended to the calling session's
  context and to `--related-spec`'s trail.
- Full delivery (v2) requires: explicit approval prompt at call
  time, non-defaulted budget, post-call commit/PR review surfaced
  on the calling side.
- The subagent invocation API is the main implementation question
  deferred to delivery. Likely path: shell-out to a peer-side
  `hero peer-call-server` invocation that operates in the peer's
  cwd and emits structured stdout.
