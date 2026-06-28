---
title: "Native Drive — autonomous initiative execution for the hero-code Swift app"
slug: native-drive-for-hero-code
type: feature
status: handed_off
priority: high
horizon: now
tags: [drive, goal, loop, autonomy, hero-code, swift, cross-repo]
created: 2026-06-27
---

# Native Drive — autonomous initiative execution for the hero-code Swift app

## Goal

Bring "Drive" — autonomous initiative execution — to the hero-code Swift
desktop app, mirroring the Go engine's just-shipped `drive-autonomous-
initiative-execution` initiative (6 child specs + `drive-progressive-design`,
all under `internal/drive`). Design it **natively** for the Swift app's
SpecStore / SpecWriteService model — not a port of the Go code.

## Kickoff

Design native Drive for hero-code. The Go engine just shipped it; build the
Swift-native equivalent on your side via `/design`. Read the brief below; the
Go specs (archived under `.hero/specs/` on the originator: `needs-me-predicate`,
`hero-goal-command`, `drive-progressive-design`) are the reference design, not
code to translate.

## Brief (the short prompt)

Build native "Drive" for hero-code, mirroring the Go engine's design:

1. An initiative **`## Goal` run-opener** (objective + machine stop-condition),
   parallel to a spec's Kickoff.
2. A deterministic **needs_me() autonomy boundary** with modes
   supervised | guided | autonomous; irreversible / outward-facing actions
   ALWAYS pause; a hard cap so the run is never unbounded.
3. A **per-turn judge** (continue | pause | done) that ANDs each child's
   verify-status with needs_me, from on-disk state. **Hero SURROUNDS the loop;
   it does NOT rebuild a loop engine or a transcript-based completion
   evaluator** — those belong to the harness/app.
4. **Progressive design from day one** (this was our follow-on gap): an
   initiative's children are partially specified on purpose. Drive must DESIGN
   a child if it isn't designed yet, THEN deliver it — never hand an undesigned
   child to delivery, and never declare an initiative done while intended-but-
   unspecced children remain. Per-child stage (needs-design | ready-to-deliver
   | done); routine design is autonomous; only a genuine design fork pauses.
5. **Pause-as-question + resume** (a precise question persisted, resumable
   cold) and **rubber-stamp learning** (promote approved pauses to auto-proceed;
   guardrail categories never promotable).

Design natively for SpecStore / SpecWriteService (the way you shipped the
intake/idea primitive). Return a spec on your side.

## Acceptance Criteria

- THE SYSTEM SHALL let a user run an initiative autonomously in the hero-code
  app, pausing only when a decision genuinely needs the human.
- THE SYSTEM SHALL design undesigned children before delivering them and SHALL
  NOT short-circuit on intended-but-unspecced children.
- THE SYSTEM SHALL NOT reimplement the harness loop or a transcript-based
  completion evaluator.

## Handoff Trail

- 2026-06-28T07:09:55Z — out → hero-code (peer_id: cd8dd06d-3df1-4878-a88f-24593dcbb4b3)
  mode: async-drop
  originating_spec: native-drive-for-hero-code
  peer_spec: hero-code/native-drive-for-hero-code
  at_commit: c0167d0
  reason: "The Go engine shipped Drive (autonomous initiative execution, 6 specs + progressive design under internal/drive). hero-code should build the Swift-native equivalent. Async-drop rather than a live spec-out (which timed out at the 10-min cap on a request this size)."

