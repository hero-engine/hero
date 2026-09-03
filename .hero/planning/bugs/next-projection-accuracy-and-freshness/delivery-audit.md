# Delivery audit — next-projection-accuracy-and-freshness

**Audited:** `git diff HEAD` at `885889c5`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] AC-1: Blocked specs excluded from "## Next" — `readyWorkByPriority()` at projection.go:240-248 adds `NOT EXISTS` subquery matching `depends_on`/`blocks` edges to non-completed targets. Test: `TestNextMD_BlockedSpecExcludedFromNext` creates a P0 feature depending on a P1 feature, asserts blocked P0 absent from Next and unblocked P1 takes slot 1. Falsification confirmed: ledger states revert of NOT EXISTS clause causes test failure.
- [✓] AC-2: P0/P1 bug surfaces in slot 2 alongside feature in slot 1 — `NextMD()` at projection.go:126-145 scans remaining candidates after slot 1 pick; when slot 1 is Feature/Enhancement, selects first Bug with P0 or P1. Test: `TestNextMD_TwoSlotBugSurfacing` seeds a P1 bug, asserts it appears in Next alongside the P0 feature.
- [✓] AC-3: Bug in slot 1, feature in slot 2 — complement-type logic at projection.go:136-143 handles the `else` branch (slot 1 is Bug), scanning for first Feature/Enhancement. Test: `TestNextMD_BugInSlot1FeatureInSlot2` seeds a P0 bug and P1 feature in an otherwise empty repo, asserts bug in slot 1 with `/deliver` hint and feature in slot 2.
- [✓] AC-4: Slot 2 omitted when no qualifying complement exists — slot 2 only rendered at projection.go:145-147 when `slot2 != nil`. Test: `TestNextMD_LowPriorityBugNotInSlot2` seeds a P2 bug, asserts it does not appear. `TestNextMD_HappyPath` (pre-existing) has no bugs and confirms single-slot output.
- [✓] AC-5: `hero next checkpoint` refreshes QUEUE.md — `refreshQueueSnapshot(heroDir)` added to `writeCheckpoint()` at checkpoint.go:358, called between snapshot projection and commit ingest. Test: `Test_refreshQueueSnapshot_WritesQueueFile` calls `refreshQueueSnapshot` on a fresh heroDir and asserts QUEUE.md created with expected header.
- [✓] AC-6: Content-hash gating skips write on timestamp-only change — `normalizeQueueTimestamp()` at checkpoint.go:418-420 strips `_Generated: <timestamp>·` via regex before string comparison in `refreshQueueSnapshot()` at checkpoint.go:406. Tests: `Test_refreshQueueSnapshot_ContentHashGating` confirms mtime unchanged on second call. `Test_normalizeQueueTimestamp` confirms regex normalizes timestamps but preserves other content differences.
- [✓] AC-7: Rename `openFeaturesByPriority` to `readyWorkByPriority` — diff shows full rename of function, struct (`featureRow` to `workRow`), and addition of `nodeType` field. Caller in `user_handoff.go:188` updated to use `readyWorkByPriority` with widened type list. All tests reference new names.

## Changes

- [✓] `internal/projection/projection.go` — Renamed `openFeaturesByPriority` to `readyWorkByPriority` with parameterized type filter, added NOT EXISTS blocked-exclusion subquery, added `nodeType` field to `workRow`, implemented two-slot rendering in `NextMD()` with complement-type selection, extracted `writeWorkItem` helper, widened `blockedFeatures` type filter to include Bug/Enhancement.
- [✓] `internal/projection/projection_test.go` — 4 new tests (`BlockedSpecExcludedFromNext`, `TwoSlotBugSurfacing`, `LowPriorityBugNotInSlot2`, `BugInSlot1FeatureInSlot2`). Updated `PrioritiesOrdered` and `TieBreakDeterministic` for two-slot semantics. Updated `EmptyRepo` for "No ready" wording.
- [✓] `internal/projection/user_handoff.go` — Updated sole caller at line 188 from `openFeaturesByPriority` to `readyWorkByPriority` with widened type list; updated rationale string from "highest-priority open feature" to "highest-priority ready work".
- [✓] `internal/cli/checkpoint.go` — Added `refreshQueueSnapshot()` call inside `writeCheckpoint()`, new `refreshQueueSnapshot` function with content-hash gating, new `normalizeQueueTimestamp` function and regex.
- [✓] `internal/cli/checkpoint_test.go` — 3 new tests (`WritesQueueFile`, `ContentHashGating`, `normalizeQueueTimestamp`) covering queue refresh behavior and timestamp normalization.

## Open items

(none)

## Audit notes

(none)
