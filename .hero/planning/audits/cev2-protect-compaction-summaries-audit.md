---
title: "Audit: cev2-protect-compaction-summaries"
auditor: cold-review (independent)
date: 2026-06-09
verdict: SHIP
confidence: high
surface: noteworthy
spec: cev2-protect-compaction-summaries
---

# Audit: cev2-protect-compaction-summaries

## Acceptance Criteria Cross-Check

| # | AC | Verdict | Notes |
|---|-----|---------|-------|
| 1 | `looksLikeCompactionSummary()` detects 3 patterns | DONE | ContextCurator.swift line 449; lowercased hasPrefix for bracket patterns, contains for continuation prompt |
| 2 | `tagMessage()` calls detection before `looksLoadBearing()` for user messages | DONE | ContextCurator.swift line 412; early return on match |
| 3 | Port-fidelity ledger updated | DONE | ContextCurator.swift line 24 |
| 4 | Unit test: all 3 patterns detected | DONE | `test_compactionSummaryTaggedLoadBearing`, `test_hardCompactionSummaryTaggedLoadBearing`, `test_continuationPromptTaggedLoadBearing` |
| 5 | Unit test: summary survives budget pruning | DONE | `test_compactionSummarySurvivesBudgetPrune` |
| 6 | Unit test: normal prose still tagged `.prose` | DONE | `test_normalProseStillTaggedProse` |
| 7 | Existing tests still pass | DONE | 33/33 pass (verified independently) |

**AC: 7/7 DONE.**

## Findings

### F1 — Spec premise incorrect for 2 of 3 patterns (medium, informational)

The spec states:

> the compaction summary ("This session is being continued from a previous
> conversation...") enters the message list as a `role: "user"` message.

This is correct for the third pattern (session continuation prompt, injected by the external Claude Code harness). However, for the first two patterns:

- `[compacted summary of N prior messages]`
- `[hard-compacted summary of N prior messages]`

**`Compactor.swift` line 287 creates these as `LLMMessage.system(...)`, not `LLMMessage.user(...)`.**

System messages are tagged `.system` by `tagMessage()` (line 406-407) and are **never subject to budget pruning**. The `looksLikeCompactionSummary()` check only fires inside the `case "user"` branch (line 410-412), so for Compactor-produced summaries, the new detection code is unreachable.

**Impact: None at runtime.** The two bracket-pattern checks are defense-in-depth against a future change where the Compactor might emit user-role summaries, or against an external source injecting them as user messages. The protection is redundant but not harmful. The third pattern (continuation prompt) IS genuinely needed.

**Recommendation:** Add a comment in `looksLikeCompactionSummary` noting that the bracket patterns are defense-in-depth since the Compactor currently emits system-role messages. This prevents a future reader from believing the code is load-bearing for those patterns when it currently is not.

### F2 — End-to-end pruning test uses a synthetic scenario (low)

`test_compactionSummarySurvivesBudgetPrune` constructs a `[compacted summary of ...]` as a `user(...)` message. In production, the Compactor creates these as `system(...)`. The test proves the tagging code works correctly on user-role input, but the scenario it tests does not occur in production for the bracket patterns.

The test IS valid for the continuation prompt pattern (which does enter as user role). Consider adding a second end-to-end test using the continuation prompt text for production-realistic coverage.

### F3 — Diff bundles two specs (informational)

The diff includes substantial changes for `cev2-verbatim-turn-counting` (the `turnBoundaryIndex` function, rewritten `markVerbatimRecentK`, and 4 turn-counting tests). These are out of scope for this audit. The verbatim turn-counting changes appear clean and well-tested from a surface read, but they have their own audit at `cev2-verbatim-turn-counting-audit.md`.

## Boundary Check

| Boundary | Respected? |
|----------|-----------|
| Do NOT change Compactor summary generation | Yes |
| Do NOT inject as system role | Yes (detection only) |
| Do NOT modify pruning priority order | Yes |

## Detection Logic Review

```
looksLikeCompactionSummary(_ text: String) -> Bool
  let lower = text.lowercased()
  hasPrefix("[compacted summary of")       -- Layer 2 soft
  hasPrefix("[hard-compacted summary of")  -- Layer 3 hard
  contains("this session is being continued from a previous conversation")  -- session restore
```

- **Lowercasing:** Correct. Prevents case-sensitivity mismatches.
- **hasPrefix for brackets:** Appropriate. More precise than `contains`; avoids mid-message false positives.
- **contains for continuation:** Appropriate. The continuation prompt may be preceded by other text in some injection paths.
- **Pattern ordering:** The `[hard-compacted` check is redundant when `[compacted` uses `hasPrefix` -- a string starting with `[hard-compacted summary of` does NOT start with `[compacted summary of`, so both checks are necessary. Correct.
- **False positive risk:** Low. `[compacted summary of` is a structured format unlikely in organic user input. Tested with `[some other bracket thing]` as a negative case.
- **Cross-reference comment:** Present, referencing `Compactor.swift` lines 279-285. Correct and helpful.

## Test Quality

- **Pattern detection (3 tests):** Each pattern tested individually with realistic content. Good.
- **End-to-end pruning (1 test):** Tight budget, verifies summary survives while prose is pruned. Checks tag, action, and curated output presence. Thorough.
- **False-positive guard (1 test):** 4 negative examples including a bracket-prefixed non-match. Adequate.
- **Missing:** No test for case-insensitivity (e.g., `[Compacted Summary Of ...]`). The lowercasing logic is correct by inspection, but a test would lock it down.

## Risk Assessment

- **Pattern fragility:** Acknowledged in the spec. Mitigated by prefix matching and cross-reference comment.
- **Regression risk:** Zero. Additive change; no existing paths modified in the tagging logic (the `case "user"` branch just got an early-return gate).
- **Performance:** Negligible. Two `hasPrefix` calls and one `contains` on an already-lowercased string per user message.

## Verdict

**SHIP.** The implementation matches the spec. All acceptance criteria are met. The code is correct for what it does. The spec's premise about message roles is partially wrong (F1), but the implementation is still valuable: the continuation-prompt detection (pattern 3) is genuinely needed, and the bracket-pattern detection (patterns 1-2) provides harmless defense-in-depth. No blocking issues.
