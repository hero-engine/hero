---
title: "Protect compaction summaries from pruning"
slug: cev2-protect-compaction-summaries
type: bug
status: completed
priority: high
size: small
parent: context-engine-v2
created: 2026-06-09
tags: [hero-code, swift, context-engine, curator, compaction, pruning]
---

# Protect compaction summaries from pruning

## Issue

When a conversation is continued from a prior session, the compaction summary
("This session is being continued from a previous conversation...") enters
the message list as a `role: "user"` message. The curator's `tagMessage()`
tags it `.prose` because none of the 8 `looksLoadBearing()` markers match.
Budget pruning drops `.prose` messages lowest-score-first, and the compaction
summary -- being early in the conversation -- is among the first to go.

This is the most valuable message in a continued session. It IS the
pre-compaction history: all decisions, file paths, and context from the prior
session compressed into one message. Pruning it throws away the entire
continuation context.

Discovered during a deep analysis of a real ~50K token conversation showing
~43% token waste (context-engine-v2 initiative).

## Investigation

### Code trace

1. **`tagMessage()`** (`ContextCurator.swift` line 402-419): User messages
   are tagged `.prose` unless `looksLoadBearing()` returns true.

2. **`looksLoadBearing()`** (`ContextCurator.swift` line 422-435): Checks
   for 8 markers: "let's", "now i will", "plan:", "decision:", "approved:",
   "request_plan_approval", "ask_user_question", "i'll". The compaction
   summary text does not contain any of these markers.

3. **Budget prune pass** (`ContextCurator.swift` curate function): Drops
   `.prose` messages lowest-score-first until under target. The compaction
   summary, being early and low-scored, is pruned first.

### The compaction summary patterns

Two patterns to detect, from `Compactor.swift` lines 279-285:

- `"[compacted summary of N prior messages]"` (Layer 2 soft-summarize)
- `"[hard-compacted summary of N prior messages]"` (Layer 3 hard-compact)

Additionally, the continuation prompt injected by the session restore flow:

- `"This session is being continued from a previous conversation"`

### Root cause

`tagMessage()` has no awareness of compaction summaries. They enter as plain
user messages and pass through the load-bearing check without matching any
marker. The pruning pass then treats them as disposable prose.

### Severity

High. Every continued session that hits budget pressure loses its entire
continuation context. The model proceeds without knowing what happened in
the prior session, leading to repeated work, contradictory decisions, and
confused output.

## Goal

Compaction summaries and session continuation messages are tagged
`.loadBearing` so they survive budget pruning. The fix is minimal: detect
the known patterns and protect them.

## Changes

All files are in `../hero-code/apps/hero-desktop-mac/Sources/HeroDesktop/Engine/`.

1. **Add compaction summary detection to `tagMessage()` or `looksLoadBearing()`**
   (`ContextCurator.swift`)
   - Option A (recommended): Add the detection to `tagMessage()` directly,
     before the `looksLoadBearing()` call. Check if the user message content
     starts with any of the known compaction prefixes. If so, return
     `.loadBearing` immediately.
   - Option B: Add the prefixes to `looksLoadBearing()`. This is simpler
     but mixes structural detection (compaction format) with semantic
     detection (human language markers).
   - Patterns to detect:
     - `"[compacted summary of"` (covers both soft and hard variants)
     - `"[hard-compacted summary of"` (explicit match for hard variant)
     - `"This session is being continued from a previous conversation"`
   - Use `hasPrefix()` or `contains()` on lowercased content. Prefer
     `hasPrefix()` for the bracket-prefixed patterns (more precise).

2. **Update port-fidelity ledger** (`ContextCurator.swift` lines 17-22)
   - Add a fidelity note:
     `//  - [cev2] Compaction summaries tagged loadBearing (v1 has no compaction summaries).`

## Boundaries

- Do NOT change how the Compactor generates summaries. The fix is in
  detection/tagging, not in summary generation.
- Do NOT inject compaction summaries as `role: "system"` messages. While
  that would make them immune to pruning, it changes the API contract and
  may confuse the model about who is speaking. The `.loadBearing` tag
  approach keeps the message structure unchanged.
- Do NOT modify the pruning priority order. `.loadBearing` messages are
  already protected from budget pruning -- the existing priority order
  is correct.

## Risks

- **Pattern fragility.** If the compaction summary format changes in the
  future, the detection will break. Mitigate by using prefix matching
  rather than exact string matching, and by adding a comment cross-
  referencing `Compactor.swift` lines 279-285 where the format is defined.
- **False positives.** A user message that happens to start with
  `"[compacted summary of"` would be tagged load-bearing. This is
  extremely unlikely in practice and the consequence (retaining an extra
  message) is benign.

## Validation

- **Unit test:** Create a conversation with a compaction summary user
  message, set a tight budget, and verify the summary is tagged
  `.loadBearing` and survives pruning while other prose is pruned.
- **Unit test:** Verify all three pattern variants are detected:
  `[compacted summary of ...]`, `[hard-compacted summary of ...]`, and
  `This session is being continued...`.
- **Existing tests must still pass.** No existing test uses compaction
  summary text, so this is a safe additive change.
