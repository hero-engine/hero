---
title: "Audit: cev2-widen-load-bearing-markers"
spec: cev2-widen-load-bearing-markers
auditor: cold-audit (independent)
date: 2026-06-09
verdict: SHIP
confidence: high
---

# Audit: Widen looksLoadBearing() marker set

## Verdict

**SHIP** -- all acceptance criteria met, implementation matches spec, tests
cover every specified case, no correctness bugs found.

## Scope check

The spec prescribes three categories of change:

1. 14 new semantic markers added to `looksLoadBearing()`
2. Structural signal detection (code fences, pipe tables)
3. Role-aware "?" check for user messages in `tagMessage()`
4. Port-fidelity ledger update

All four are present in the diff. No changes outside the spec boundary were
made. The pruning logic is untouched -- only the tagging layer changed.

## Acceptance criteria cross-check

| # | Criterion | Status | Evidence |
|---|-----------|--------|----------|
| 1 | 14 new semantic markers in `looksLoadBearing()` | PASS | ContextCurator.swift L441-458: exactly 14 new markers with `[cev2]` comments |
| 2 | Structural signals: code fences (`\`\`\``) | PASS | ContextCurator.swift L463 |
| 3 | Structural signals: pipe tables (two `\| ` occurrences) | PASS | ContextCurator.swift L465-468 |
| 4 | User "?" check in `tagMessage()` (role-aware) | PASS | ContextCurator.swift L415 -- only fires for `role == "user"` |
| 5 | Port-fidelity ledger updated | PASS | ContextCurator.swift L25: "22 markers + structural signals" |
| 6 | Unit test: all 14 new semantic markers | PASS | `test_widenedSemanticMarkers` -- all 14 tested individually |
| 7 | Unit test: structural signals | PASS | `test_structuralSignalsLoadBearing` -- code fence, pipe table, single-pipe negative |
| 8 | Unit test: user questions with "?" | PASS | `test_userQuestionTaggedLoadBearing` -- user positive + assistant negative |
| 9 | Negative test: "pleasant" / "done" | PASS | `test_pleasantNotLoadBearing` |
| 10 | Original 8 markers still work | PASS | `test_originalMarkersStillWork` + original `test_loadBearingMarkers` |
| 11 | All existing tests pass | PASS | 38/38 ContextCuratorTests + 8/8 ContextReorderTests = 46/46 |

## Adversarial analysis

### Trailing-space markers -- false positive risk

The markers `"please "`, `"do not "`, `"don't "`, and `"must "` use trailing
spaces to avoid substring collisions ("pleasant", "done", "musty"). This
creates a recall gap: messages ending with these words at end-of-string
(e.g. "you must") will NOT match. The spec explicitly chose trailing spaces
as the approach, so this is by-design. The recall gap is acceptable -- these
phrases at end-of-string are uncommon in practice.

False positives from `"must "` (e.g. "I must admit") are possible but low-
frequency in coding conversations and result in retaining a message that
would otherwise be pruned -- a low-severity error in the safe direction.

### "note:" substring collision

`"note:"` matches substrings like "keynote:" -- an extremely rare occurrence
in coding-agent conversations. No action needed.

### "?" check aggressiveness

User messages containing "?" are tagged load-bearing. Edge cases:
- `"ok?"` -- tagged load-bearing. Defensible: this is a confirmation request.
- `"?"` alone -- tagged load-bearing. Defensible: bare question marks direct
  the model's attention.
- `"haha right?"` -- tagged load-bearing. Mild over-tag but acceptable.

The check is role-aware (assistant "?" messages go through `looksLoadBearing()`
instead), which is the spec-prescribed option (b). No false positives on the
assistant side.

### Pipe table detection -- shell pipes

The pattern `"| "` (pipe-space) repeated twice will match shell pipe chains
like `cat foo | grep bar | wc -l`. However, shell commands in coding
conversations are operationally important content, so this "false positive"
produces a correct retention outcome. No concern.

### Code fence detection

`text.contains("```")` on original-case text is correct (backticks are not
case-sensitive). No edge case issues found.

## Spec boundary compliance

| Boundary | Compliant? | Evidence |
|----------|-----------|----------|
| Do NOT make all user messages load-bearing | Yes | Only "?" triggers the upgrade |
| Do NOT add markers for every possible phrase | Yes | Exactly 14 new markers as specified |
| Do NOT change the pruning priority order | Yes | Pruning logic untouched in diff |

## Test quality assessment

- **Positive coverage:** All 14 markers individually tested with representative
  sentences, not just bare markers. Code fences, pipe tables, and user "?" all
  covered.
- **Negative coverage:** "pleasant"/"done" false-positive prevention tested.
  Single-pipe non-table tested. Assistant "?" non-promotion tested.
- **Regression coverage:** Original 8 markers re-verified. All pre-existing
  tests pass unmodified.
- **Missing:** No test for "must" at end-of-string (recall gap). No test for
  "note:" substring in "keynote:". These are by-design gaps per spec trade-offs,
  not omissions.

## Risk assessment

The over-tagging risk (spec section "Risks") is bounded by the nature of
hero-code sessions: prose messages are a small fraction of total messages
in tool-heavy sessions. The spec notes: "Widening load-bearing detection
shifts the balance toward retaining useful prose at the cost of slightly
less aggressive pruning -- a net win." This is a sound trade-off.

The integration check (run on a representative ~50K conversation) is listed
as a manual validation step, not a unit test. This is appropriate -- the
unit tests verify correctness, the integration check verifies the
over-tagging threshold (<80%) in practice.

## Files reviewed

- `apps/hero-desktop-mac/Sources/HeroDesktop/Engine/ContextCurator.swift` -- implementation
- `apps/hero-desktop-mac/Tests/HeroDesktopTests/ContextCuratorTests.swift` -- tests
- `.hero/planning/cev2-widen-load-bearing-markers.md` -- spec
