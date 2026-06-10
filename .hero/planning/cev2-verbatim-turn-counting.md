---
title: "Fix verbatim window to count turns, not messages"
slug: cev2-verbatim-turn-counting
type: bug
status: completed
priority: critical
size: small
parent: context-engine-v2
created: 2026-06-09
tags: [hero-code, swift, context-engine, curator, verbatim, ship-blocking]
---

# Fix verbatim window to count turns, not messages

## Issue

Ship-blocking regression introduced by the `toLLMMessages()` fix in
`Session.swift` (line 231). Before that fix, the model lost all tool history
between turns. After it, one `SessionMessage` assistant turn with N tool
calls expands to N+1 `LLMMessage`s (1 assistant + N tool results).
`markVerbatimRecentK()` counts raw `LLMMessage`s backward, so K=4 now
covers approximately 1 actual turn instead of 4 turns. The model loses
recent work every turn.

Discovered during a deep analysis of a real ~50K token conversation showing
~43% token waste (context-engine-v2 initiative).

## Investigation

### Affected functions

1. **`ContextCurator.markVerbatimRecentK()`** -- lines 440-448 of
   `ContextCurator.swift`. Counts backward through raw `LLMMessage`s,
   skipping system messages, marking K=4 messages as verbatim. After
   `toLLMMessages()` expansion, a single assistant turn with 3 tool calls =
   4 messages (1 assistant + 3 tool results). K=4 covers just that one turn.

   ```swift
   static func markVerbatimRecentK(_ verbatim: inout [Bool], tags: [MsgTag], window: Int) {
       var k = 0
       for idx in stride(from: tags.count - 1, through: 0, by: -1) {
           if tags[idx] == .system { continue }
           if k < window {
               verbatim[idx] = true
               k += 1
           }
       }
   }
   ```

2. **`ContextCurator.reorderByRelevance()`** -- lines 551-557 of
   `ContextCurator.swift`. Same backward count for `verbatimStart`:

   ```swift
   var verbatimStart = count
   var nonSystemFromEnd = 0
   for i in stride(from: count - 1, through: 0, by: -1) {
       if messages[i].role != "system" {
           nonSystemFromEnd += 1
           if nonSystemFromEnd >= verbatimRecentK {
               verbatimStart = i
               break
           }
       }
   }
   ```

3. **`Compactor.swift`** -- lines 65-66. `softPreserveRecent = 8` and
   `hardPreserveRecent = 4` are message counts passed to `summarize()` as
   `preserveLast`. Line 215: `let suffixStart = max(0, messages.count - preserveLast)`.
   Same message-vs-turn bug: `preserveLast: 4` preserves 4 raw messages,
   which is ~1 turn post-expansion.

### Root cause

All three locations use raw `LLMMessage` counts as a proxy for "turns."
Before `toLLMMessages()`, each `SessionMessage` mapped 1:1 to one
`LLMMessage`, so message count = turn count. After `toLLMMessages()`, one
assistant turn with tool calls fans out to N+1 messages, breaking the
assumption.

### Severity

Critical. Without this fix, the verbatim window shrinks to ~1 turn. The
model loses context of its recent work on every turn, leading to repeated
file reads, contradictory edits, and wasted tokens. This is the P0 item in
the context-engine-v2 initiative and MUST ship with the `toLLMMessages()` fix.

## Goal

`markVerbatimRecentK()` counts turn boundaries instead of raw messages. K=4
means the last 4 assistant turns plus all their tool results are verbatim.
The same turn-counting logic applies to `reorderByRelevance()` and to the
`Compactor.swift` `preserveLast` constants.

## Changes

All files are in `../hero-code/apps/hero-desktop-mac/Sources/HeroDesktop/Engine/`.

1. **Define turn boundaries in `ContextCurator.markVerbatimRecentK()`**
   (`ContextCurator.swift` lines 440-448)
   - Change the backward scan to count turn starts instead of individual
     messages. A new turn starts at each `assistant` or `user` message.
     `tool` messages (role="tool") belong to the preceding assistant turn
     and do NOT increment the turn counter.
   - When a turn boundary is found, mark it AND all following tool results
     (up to the next turn boundary) as verbatim.
   - System messages continue to be skipped (neither counted nor marked).
   - The public constant `verbatimRecentK = 4` retains its name but its
     semantics change from "4 messages" to "4 turns."

2. **Fix `reorderByRelevance()` verbatimStart calculation**
   (`ContextCurator.swift` lines 551-557)
   - Replace the `nonSystemFromEnd` message counter with the same
     turn-counting logic from change 1. `verbatimStart` should point to
     the first message of the Kth-from-last turn, not the Kth-from-last
     raw message.

3. **Fix `Compactor.swift` preserveLast semantics** (lines 65-66, 215)
   - Change `softPreserveRecent` and `hardPreserveRecent` from message
     counts to turn counts. The `summarize()` function's `suffixStart`
     calculation (line 215) must use the same turn-counting logic to find
     the start of the preserved suffix.
   - Consider extracting a shared `turnBoundaryIndex(from:turnsFromEnd:)`
     helper in `ContextCurator` that both the curator and compactor can
     call, to avoid duplicating the logic.

4. **Update port-fidelity ledger** (`ContextCurator.swift` lines 17-22)
   - Add a fidelity note:
     `//  - [cev2] Verbatim window counts turns (user/assistant boundaries), not raw messages (v1 counts messages).`
   - Add a similar note in `Compactor.swift` line 61 block:
     `//  - [cev2] preserveRecent counts turns, not raw messages.`

## Boundaries

- Do NOT change the `CurationOptions.verbatimK` type or the
  `ContextCurator.verbatimRecentK` constant value (4). Only the semantics
  change: messages -> turns.
- Do NOT modify `toLLMMessages()` in `Session.swift`. That fix is correct;
  the bug is in the curator's counting, not the message expansion.
- Do NOT introduce SwiftUI/AppKit dependencies. All changes are pure
  Foundation Swift.

## Risks

- **Turn boundary definition.** The definition "assistant or user message
  starts a new turn" must handle edge cases: consecutive user messages
  (each is its own turn), assistant messages without tool calls (still a
  turn), orphaned tool results at the start of the conversation.
- **Compactor.swift is sensitive.** The `preserveLast` change affects which
  messages are fed to the summarization LLM call. If the suffix is too
  large, the summarization prompt exceeds the model's context. Test with
  realistic conversation lengths.
- **Existing tests assume message counting.** Tests like
  `test_isVerbatimFlagSetForTrailingK()` use simple user-only messages
  where message count = turn count. They will still pass, but new tests
  must exercise multi-tool-call turns to validate the fix.

## Validation

- **Regression test with multi-tool-call turns:** Build a fixture where an
  assistant turn has 3 tool calls (expanding to 4 LLMMessages via
  `toLLMMessages()`). With K=4, all 4 of the last 4 turns must be verbatim,
  NOT just the last 4 raw messages.
- **Regression test for reorderByRelevance:** Same fixture, verify
  `verbatimStart` points to the correct turn boundary.
- **Compactor test:** Verify `preserveLast` with multi-tool-call turns
  preserves the correct number of turns in the suffix.
- **Existing tests must still pass.** The change is backward-compatible for
  conversations where every turn is a single message.
- **Manual validation:** Run a real conversation in hero-code with context
  curation enabled. Inspect the `AnnotatedContextPlan` annotations to
  verify the last 4 turns (not messages) are marked verbatim.
