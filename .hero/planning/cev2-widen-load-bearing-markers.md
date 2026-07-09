---
title: "Widen looksLoadBearing() marker set"
slug: cev2-widen-load-bearing-markers
type: feature
status: completed
completed_at: 2026-06-09
priority: medium
size: small
parent: context-engine-v2
created: 2026-06-09
delivered_in_peer:
  peer_id: cd8dd06d-3df1-4878-a88f-24593dcbb4b3
  peer_alias_display: hero-code
  code_path: apps/hero-desktop-mac/Sources/HeroDesktop/Engine/ContextCurator.swift
  function: looksLoadBearing
  summary: "Marker set widened 8 → 22 with [cev2]-tagged additions plus code-fence and pipe-table structural signals."
  audit_local: .hero/planning/audits/cev2-widen-load-bearing-markers-audit.md
tags: [hero-code, swift, context-engine, curator, tagging, heuristics]
---

# Widen looksLoadBearing() marker set

## Context

`looksLoadBearing()` in `ContextCurator.swift` (line 422-435) determines
whether a user or assistant message is operationally important enough to
survive budget pruning. It currently checks for exactly 8 hardcoded phrases:

```swift
let markers = [
    "let's",
    "now i will",
    "plan:",
    "decision:",
    "approved:",
    "request_plan_approval",
    "ask_user_question",
    "i'll",
]
```

A deep analysis of a real ~50K token conversation revealed that many
operationally important messages are tagged `.prose` and pruned because they
don't match any marker. Missed categories include:

- **User questions** -- messages containing "?" that direct the model's work
- **Diagnostic conclusions** -- "root cause", "the bug is", "the issue is"
- **Explicit instructions** -- "please", "make sure", "do not", "don't"
- **Code fences** -- messages with triple-backtick code blocks
- **Summary tables** -- messages with pipe-separated markdown tables

## Goal

Expand the marker set to catch the most common load-bearing patterns missed
by the current 8 markers, without over-tagging to the point where the
prose-pruning lever becomes useless.

## Approach

Two categories of new detection:

1. **Semantic markers** -- additional phrases that reliably indicate
   operational importance. These follow the same `contains()` pattern as
   the existing markers.

2. **Structural signals** -- patterns like "?" in user messages, code
   fences, and pipe tables that indicate importance regardless of specific
   word choice. These need their own check logic, not just string contains.

The risk of over-tagging is real but bounded: in tool-heavy sessions (which
are the majority of hero-code usage), prose messages are a small fraction
of the total. The pruning lever is already tiny. Widening load-bearing
detection shifts the balance toward retaining useful prose at the cost of
slightly less aggressive pruning -- a net win.

## Changes

All files are in `../hero-code/apps/hero-desktop-mac/Sources/HeroDesktop/Engine/`.

1. **Add semantic markers to `looksLoadBearing()`**
   (`ContextCurator.swift` line 422-435)
   - Add the following markers to the existing array:
     - `"root cause"` -- diagnostic conclusions
     - `"the bug is"` -- diagnostic conclusions
     - `"the issue is"` -- diagnostic conclusions
     - `"the fix is"` -- diagnostic conclusions
     - `"please "` -- explicit instructions (trailing space avoids "pleasant")
     - `"make sure"` -- explicit instructions
     - `"do not "` -- explicit instructions (trailing space avoids "done")
     - `"don't "` -- explicit instructions
     - `"must "` -- explicit instructions
     - `"important:"` -- explicit callouts
     - `"note:"` -- explicit callouts
     - `"summary:"` -- recap markers
     - `"conclusion:"` -- recap markers
     - `"next step"` -- plan markers

2. **Add structural signal detection to `looksLoadBearing()`**
   (`ContextCurator.swift`)
   - **User questions:** If the message role is `user` (caller must pass
     role or check before calling), and the content contains "?", tag as
     load-bearing. This is aggressive but user questions are almost always
     operationally important -- they direct the model's work.
     - Note: `looksLoadBearing()` currently takes only `String`, not role.
       Either: (a) add a `role` parameter, or (b) handle the "?" check
       in `tagMessage()` directly before calling `looksLoadBearing()`.
       Option (b) is cleaner -- keep `looksLoadBearing()` as a pure text
       check and add the role-aware check at the call site.
   - **Code fences:** If the content contains "```", tag as load-bearing.
     Messages with code blocks typically contain important code snippets,
     examples, or instructions.
   - **Pipe tables:** If the content contains "| " followed later by
     another "| ", tag as load-bearing. Markdown tables are summary
     artifacts that should survive pruning.

3. **Update port-fidelity ledger** (`ContextCurator.swift` lines 17-22)
   - Add a fidelity note:
     `//  - [cev2] looksLoadBearing() expanded from 8 to ~22 markers + structural signals (v1 has 8 markers).`

## Boundaries

- Do NOT make all user messages load-bearing. Only user messages with "?"
  get the upgrade. Plain affirmations ("yes", "ok", "sure") and
  acknowledgments should remain `.prose`.
- Do NOT add markers for every possible phrase. The goal is the 80/20 --
  catch the most common patterns, not achieve perfect recall.
- Do NOT change the pruning priority order. This spec only changes which
  messages get tagged `.loadBearing`; the pruning logic itself is
  unchanged.

## Risks

- **Over-tagging reduces the prose-pruning lever.** If too many messages
  become load-bearing, budget pruning has fewer candidates to drop and
  the context window fills up faster. Monitor: after the change, run the
  curator on a representative conversation and compare the load-bearing
  count before and after. If the load-bearing ratio exceeds 80% of
  non-system messages, the markers are too aggressive.
- **"please" false positives.** "please" appears in polite prose that
  isn't operationally important. The trailing space (`"please "`) reduces
  but doesn't eliminate false positives. Acceptable: a few extra retained
  messages are better than losing explicit instructions.
- **Structural signals are language-agnostic.** Code fences and pipe
  tables work regardless of natural language. The semantic markers are
  English-only. This is acceptable for now -- hero-code's primary user
  base communicates in English.

## Validation

- **Unit test for each new marker:** Verify that messages containing each
  new semantic marker are tagged `.loadBearing`.
- **Unit test for structural signals:** Verify user messages with "?" are
  load-bearing, messages with code fences are load-bearing, messages with
  pipe tables are load-bearing.
- **Negative tests:** Verify that plain prose without any markers remains
  tagged `.prose`. Verify "pleasant" (contains "pleas") is not tagged
  load-bearing (the trailing space in `"please "` prevents this).
- **Existing tests must still pass.** The existing `test_loadBearingMarkers`
  test checks the 8 original markers and a negative case -- all should
  still pass unchanged.
- **Integration check:** Run the curator on a representative ~50K token
  conversation and compare annotations before/after. The load-bearing
  count should increase but remain below 80% of non-system messages.
