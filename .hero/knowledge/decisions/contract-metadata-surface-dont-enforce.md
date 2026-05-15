---
title: Contract Metadata Surfaces, Does Not Enforce, in v1
type: decision
status: proposed
created: 2026-05-15
tags: [peering, contracts, ux, signal-not-guard, decision]
relations:
  - target: cross-repo-peering
    kind: decided-in
  - target: hero-cloud-split
    kind: related
---

# Contract Metadata Surfaces, Does Not Enforce, in v1

## Decision

The peer manifest's `contracts:` section is consumed as a **context
signal**, not a guardrail. Three-step ladder of increasing
intervention:

1. **v1 — Passive surfacing.** When the session edits files that
   import a peer-owned contract symbol, `/resume` and `hero context`
   add a one-liner: *"You're touching `<symbol>` — owned by peer
   `<alias>`. Convention: `<slug>`. Last changed: `<commit>`."* No
   blocking, no prompts.
2. **v2 — Boundary nudge.** When the touch looks structural
   (changing a contract's shape, not just consuming it), `hero
   nudge` suggests an advisory call. User can accept, ignore, or
   silence per-spec.
3. **v3 — Auto-trigger.** Contract-shape edits auto-open an
   advisory or spec-out call. Defer unless v2 dogfooding proves the
   nudge is friction.

## Context

The cross-repo-peering spec needs a **boundary detector**: a way to
tell that a change in repo A crosses into repo B's territory. The
original draft contemplated path heuristics, commit-message
scanning, and AST analysis.

A cleaner signal exists: **"this file imports a peer-owned contract
symbol"** is unambiguous, cheap to compute (Go import scan), and
points directly at the convention that governs the boundary.
Contract metadata becomes the detector for free.

The risk: if v1 turns the detector into a guard (blocking commits,
auto-opening sessions), it becomes friction before we know whether
the signal is high-quality. Better to start with surfacing only.

## Options considered

1. **No contract detection in v1.** Rely on the user to remember
   when they're touching a peer surface.
   - Pros: simplest.
   - Cons: misses the lowest-cost win; contradicts the
     seamlessness goal of the spec.

2. **Detect and surface (passive) in v1; enforce later.**
   - Pros: tests the signal in real use without committing to
     UX that may turn out friction-shaped; the detector is the
     same regardless of intervention level.
   - Cons: requires discipline to defer the "block on contract
     change" temptation.

3. **Detect and enforce (block / auto-prompt) in v1.**
   - Pros: maximum guardrail.
   - Cons: untested UX; high friction risk; if the signal is
     noisy in early use, blocks land in the model's way; rollback
     is awkward.

## Decision

Option 2. Passive surface in v1; ladder up only when dogfooding
proves the next step adds value.

## Consequences

- v1 ships the Go import scanner and the one-liner surfacing in
  `/resume` and `hero context`. Nothing blocks.
- The peer manifest's `contracts:` section is the source of truth
  for "what symbols are peer-owned" — populated by the peer when it
  generates its manifest.
- The scanner runs on changed files only (cheap; signal-rich).
  Untouched-but-importing files are ignored. If real usage shows
  this misses important cases, revisit at v2.
- v2 boundary nudge UX requires distinguishing structural edits
  (changing a contract type or signature) from consumption (calling
  a function the contract exposes). v1 doesn't need this
  classification; v2 design will lock the heuristic.
- v3 auto-trigger is deferred indefinitely until the nudge UX is
  proven friction-free.
- The contract metadata in the manifest does double duty:
  - Lookup for the surfacing one-liner.
  - Versioning signal (manifest's `contracts.version` informs the
    consumer of which contract version it's pinned against).
