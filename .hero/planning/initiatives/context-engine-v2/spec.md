---
title: "Context Engine v2 — Fix and Optimize hero-code Desktop Context Curation"
slug: context-engine-v2
type: initiative
status: planning
domain: engineering
size: x-large
priority: critical
created: 2026-06-09
tags: [hero-code, swift, context-engine, curator, compactor, performance, token-efficiency]
child:
  - cev2-verbatim-turn-counting
  - cev2-protect-compaction-summaries
  - cev2-widen-load-bearing-markers
  - cev2-bash-output-supersede
  - cev2-tool-input-compression
  - cev2-context-engine-test-harness
  - cev2-system-prompt-curator
---

# Context Engine v2 — Fix and Optimize hero-code Desktop Context Curation

## Vision

The hero-code desktop app's context engine correctly and efficiently curates
conversation history before every model call, eliminating the ~43% token waste
identified in a deep analysis of real production sessions. The verbatim window
counts turns (not raw messages), compaction summaries survive pruning, Bash
outputs participate in supersede, tool inputs are compressed after execution,
and load-bearing detection catches the patterns that matter. When all items
ship, a 50K-token conversation sends ~28K tokens instead of ~50K, the model
retains its last 4 turns of work (not ~1 turn), and the test harness prevents
regressions.

## Problem

A deep analysis of a real ~50K token conversation revealed 7 issues causing
approximately 43% token waste in the context engine. The architecture is sound
-- the `ContextCurator` (stateless heuristic curator, `enum ContextCurator`
with all-static functions) and `Compactor` (3-layer: L1=curator,
L2=soft-summarize@60%, L3=hard-compact@80%) are faithful ports of the Rust v1
`crates/hero-core/src/context_curator.rs`. But critical gaps exist:

1. **Verbatim window counts messages, not turns.** After the `toLLMMessages()`
   fix (which expands one `SessionMessage` assistant turn with 3 tool calls
   into 4 `LLMMessage`s: 1 assistant + 3 tool results), K=4 raw messages
   covers ~1 actual turn instead of 4 turns. The model loses recent work
   every turn. This is ship-blocking.

2. **Compaction summaries are pruned.** "This session is being continued from
   a previous conversation..." enters as a user message, gets tagged `.prose`,
   and is the first thing pruned. It IS the pre-compaction history.

3. **Load-bearing detection is too narrow.** Only 8 hardcoded phrases. Misses
   user questions, diagnostic conclusions, explicit instructions, code fences,
   and summary tables.

4. **Bash outputs never supersede.** `pathWhitelist` only covers
   Read/Write/Edit/grep/glob. Bash tool outputs -- which dominate real sessions
   -- accumulate without bound.

5. **Tool inputs stay at full size forever.** The `arguments` field on
   `tool_use` messages (especially long Bash scripts) keeps full size after
   the tool result is back. Value drops near zero once the result exists.

6. **No realistic test fixtures.** Existing tests use simple single-tool-call
   turns. No tests exercise multi-tool-call assistant turns, Bash-heavy
   sessions, or compaction summary survival.

7. **System prompt is uncurated.** ~24K tokens, ~14K wasted on unused tool
   schemas, irrelevant skill lists, and computer-use instructions for
   non-screen tasks. (Separate phase -- highest risk.)

## Architecture reference

All code lives in the hero-code repo at
`/Users/bwheeler/projects/hero-engine/repository/hero-code/apps/hero-desktop-mac/Sources/HeroDesktop/Engine/`:

- **`ContextCurator.swift`** (616 lines) -- stateless heuristic curator,
  `enum ContextCurator` with all-static functions. Pipeline:
  `tagMessage()` -> `markVerbatimRecentK()` -> `estimateMessageTokens()` ->
  dedup/supersede -> compress (Wave 6 hook) -> score (Wave 6 hook) ->
  budget prune -> annotate -> assemble -> optional U-shaped reorder.

- **`Compactor.swift`** (317 lines) -- 3-layer compactor.
  `CompactionOptions.production`: 200K window, 60% soft, 80% hard.
  Constants: `softPreserveRecent = 8`, `hardPreserveRecent = 4` (message
  counts, NOT turn counts -- same bug as curator).

- **`AgentLoop.swift`** -- runs curator via `outgoing()` before every
  `provider.stream()`. The testable seam is `static func outgoing()`.

- **`Session.swift`** -- `toLLMMessages()` extension on `[SessionMessage]`
  that expands assistant turns with tool calls into N+1 LLMMessages.

- **`ChatProvider.swift`** -- `LLMMessage` struct definition.

Tests at `Tests/HeroDesktopTests/ContextCuratorTests.swift` (389 lines) and
`CompactorTests.swift`. Existing tests use simple single-tool-call turns only.

The curator is a faithful port of Rust v1. Changes need documentation against
the port-fidelity ledger referenced in the ContextCurator.swift header comment.
All code is pure Foundation Swift -- no SwiftUI/AppKit dependencies.

## Goal

Eliminate the 7 identified token-waste issues, bringing a representative 50K
session from ~43% waste to under 15%. The verbatim window reliably preserves
the last 4 turns of work regardless of tool-call fan-out. Every change is
covered by regression tests, and the port-fidelity ledger is updated to
document intentional divergences from v1.

## Specs

Seven sequenced child specs across four phases. Phase 0 is ship-blocking.
Phases 1-2 are independent quick wins and improvements. Phase 3 is test
infrastructure that starts in Phase 0 and grows with each phase. Phase 4 is
the largest and riskiest item (system prompt curation), designed and delivered
separately.

---

### Phase 0 -- Ship-blocking (must land WITH the toLLMMessages fix)

#### 1. Fix verbatim window to count turns, not messages
**Slug:** `cev2-verbatim-turn-counting` | **Type:** bug | **Size:** small |
**Deps:** none (foundation) | **SHIPS FIRST**

`markVerbatimRecentK()` at line 440-448 counts raw `LLMMessage`s backward.
After `toLLMMessages()`, one assistant turn with 3 tool calls = 4 messages
(1 assistant + 3 tool results). K=4 now covers ~1 turn instead of 4 turns.
Same bug in `reorderByRelevance()` at line 551-557 and `Compactor.swift`
lines 65-66 (`softPreserveRecent = 8`, `hardPreserveRecent = 4`).

This is the only item that MUST ship with the toLLMMessages fix. Without it,
the model loses recent work every turn.

---

### Phase 1 -- Quick wins (independent, parallelizable)

These two items are independent of each other and of Phase 0 (though they
benefit from the test harness started in Phase 0). Either can ship first.

#### 2. Protect compaction summaries from pruning
**Slug:** `cev2-protect-compaction-summaries` | **Type:** bug | **Size:** small |
**Deps:** none (independent)

The "This session is being continued..." message enters as a user message,
gets tagged `.prose`, and is the first thing pruned. It is the most valuable
message -- it IS the pre-compaction history. Fix: detect the pattern in
`tagMessage()` or `looksLoadBearing()` and tag as `.loadBearing`.

#### 3. Widen looksLoadBearing() marker set
**Slug:** `cev2-widen-load-bearing-markers` | **Type:** feature | **Size:** small |
**Deps:** none (independent)

Only 8 hardcoded phrases. Misses user questions (messages with "?"),
diagnostic conclusions ("root cause", "the bug is"), explicit instructions
("please", "make sure", "do not"), code fences, summary tables. Add ~10-15
markers and consider structural signals.

---

### Phase 2 -- Supersede/compress improvements (independent)

These two items are independent of each other. Both benefit from the test
harness. They can be parallelized with Phase 1 if capacity allows.

#### 4. Bash output supersede
**Slug:** `cev2-bash-output-supersede` | **Type:** feature | **Size:** medium |
**Deps:** none (independent)

`pathWhitelist` only covers Read/Write/Edit/grep/glob. Bash outputs -- which
dominate real sessions -- never supersede. Key on the command itself: if the
same/similar Bash command runs again, the older output is superseded.

#### 5. Tool input compression
**Slug:** `cev2-tool-input-compression` | **Type:** feature | **Size:** small |
**Deps:** none (independent)

`tool_use` `arguments` fields (especially long Bash scripts) stay at full
size forever. Once the tool result is back, the input body's value drops
near zero. Truncate `arguments` on non-verbatim assistant messages to a
summary (tool name + first line). Preserve structural fields for API
contract.

---

### Phase 3 -- Test infrastructure (cross-cutting, starts Phase 0)

#### 6. Context engine test harness
**Slug:** `cev2-context-engine-test-harness` | **Type:** feature | **Size:** medium |
**Deps:** grows with each phase

Build realistic conversation fixtures with multi-tool-call turns, Bash
outputs, compaction summaries, prose. Tests cover: tagging correctness,
verbatim window turn counting, dedup/supersede across tool types, budget
pruning priority order, end-to-end token reduction. Also test
`AgentLoop.outgoing()` -- the production entry point, currently zero tests.

Some tests ship with each phase (e.g., turn-counting regression tests ship
with P0). This spec covers the overall harness and cross-cutting fixtures.

---

### Phase 4 -- System prompt curation (separate phase, largest risk)

#### 7. System prompt curator
**Slug:** `cev2-system-prompt-curator` | **Type:** feature | **Size:** large |
**Deps:** cev2-context-engine-test-harness

The curator only sees conversation messages. System prompt is ~24K tokens,
~14K wasted on unused tool schemas, irrelevant skill lists, computer-use
instructions for non-screen tasks, stale cached file reads. Needs design
first: which sections are safe to prune? How to detect relevance?
Feature-flag gated rollout. Highest risk of behavioral regression.

---

## Dependencies

```
Phase 0 (ship-blocking):
  cev2-verbatim-turn-counting ──── must ship WITH toLLMMessages() fix

Phase 1 (quick wins, parallel):
  cev2-protect-compaction-summaries ── independent
  cev2-widen-load-bearing-markers ──── independent

Phase 2 (improvements, parallel):
  cev2-bash-output-supersede ───────── independent
  cev2-tool-input-compression ──────── independent

Phase 3 (cross-cutting):
  cev2-context-engine-test-harness ─── grows with each phase, no blocker

Phase 4 (separate):
  cev2-system-prompt-curator ──────── depends on test harness
```

No spec in Phases 1-2 blocks another. Phase 0 is the only hard gate: the
verbatim turn-counting fix must ship with (or before) the `toLLMMessages()`
fix, because without it the model loses recent work on every turn.

The test harness (Phase 3) is not blocking -- it grows incrementally as
regression tests ship alongside each fix. It has its own spec to track the
cross-cutting fixtures and `AgentLoop.outgoing()` integration tests that
don't belong to any single fix.

Phase 4 (system prompt curator) depends on the test harness being mature
enough to catch behavioral regressions from system prompt changes.

## Cross-cutting concerns

### Port-fidelity ledger

The curator is a faithful port of Rust v1. Several of these changes are
intentional divergences (v1 counts messages, v2 will count turns; v1 has
no Bash supersede). Each child spec's delivery MUST update the port-fidelity
ledger comment block at the top of `ContextCurator.swift` (lines 17-22) to
document the divergence. Format:

```
//  - [cev2] Verbatim window counts turns, not raw messages (v1 counts messages).
//  - [cev2] Bash outputs supersede by command key (v1 skips non-whitelisted tools).
```

### toLLMMessages dependency

The `toLLMMessages()` extension on `[SessionMessage]` (Session.swift line 231)
is the fix that CAUSES the verbatim window bug. It expands one assistant
`SessionMessage` with N tool calls into N+1 `LLMMessage`s. Before this fix,
the model lost all tool history; after it, the verbatim window is broken.
These two changes must ship together.

### Test coverage

Each child spec includes its own acceptance test requirements. The test harness
spec (Phase 3) covers cross-cutting fixtures and integration tests that don't
belong to any single fix. Target: every changed function has at least one
regression test exercising the fixed behavior with realistic multi-tool-call
fixtures.

### All changes are Foundation Swift

No SwiftUI/AppKit. The context engine must port again to WinUI, so it stays
framework-free. Delivery engineers must not introduce UI-framework
dependencies.

## Delivery order

Recommended delivery sequence with parallelization:

```
Week 1:
  [P0] cev2-verbatim-turn-counting     (ship-blocking, ~1 day)
  [P3] cev2-context-engine-test-harness (start fixtures, ongoing)

Week 1-2 (parallel):
  [P1] cev2-protect-compaction-summaries (~0.5 day)
  [P1] cev2-widen-load-bearing-markers   (~1 day)
  [P2] cev2-tool-input-compression       (~1 day)

Week 2-3:
  [P2] cev2-bash-output-supersede        (~3 days)

Week 4-5 (after harness is mature):
  [P4] cev2-system-prompt-curator        (~2 weeks, design + implement)
```

Total estimated effort: 4-5 weeks with one engineer. Phases 1-2 are
parallelizable across two engineers if capacity exists.

## Progress

| Spec | Phase | Status |
|------|-------|--------|
| cev2-verbatim-turn-counting | P0 | planning |
| cev2-protect-compaction-summaries | P1 | planning |
| cev2-widen-load-bearing-markers | P1 | planning |
| cev2-bash-output-supersede | P2 | planning |
| cev2-tool-input-compression | P2 | planning |
| cev2-context-engine-test-harness | P3 | planning |
| cev2-system-prompt-curator | P4 | planning |
