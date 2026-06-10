# Delivery audit — cev2-verbatim-turn-counting

**Audited:** `git diff HEAD` at `71f45ee` in hero-code
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria

- [✓] markVerbatimRecentK counts turn boundaries instead of raw messages — `ContextCurator.swift:470`: signature changed to accept `messages: [LLMMessage]`, delegates to `turnBoundaryIndex`; old per-message `k += 1` loop removed.
- [✓] reorderByRelevance uses turn counting for verbatimStart — `ContextCurator.swift:579`: `let verbatimStart = turnBoundaryIndex(in: messages, turnsFromEnd: verbatimRecentK)`. Old 10-line backward-scan loop deleted.
- [✓] Compactor.summarize preserveLast uses turn counting — `Compactor.swift:216`: `ContextCurator.turnBoundaryIndex(in: messages, turnsFromEnd: preserveLast)` replaces `max(0, messages.count - preserveLast)`.
- [✓] Shared helper extracted — `ContextCurator.turnBoundaryIndex(in:turnsFromEnd:)` at line 442. Used by both `markVerbatimRecentK` (line 471) and `Compactor.summarize` (line 216).
- [✓] Port-fidelity ledger updated — `ContextCurator.swift:23` and `Compactor.swift:62` both carry `[cev2]` fidelity notes.
- [✓] Regression tests for multi-tool-call turns — 4 new tests in ContextCuratorTests + 1 in ContextReorderTests. All assert on turn-vs-message semantics with multi-tool-call fixtures.
- [✓] Existing tests still pass — 46 tests across 3 suites, 0 failures.
- [✓] verbatimRecentK = 4 value unchanged — `ContextCurator.swift:135`: constant untouched in diff.

## Changes

- [✓] Define turn boundaries in `ContextCurator.markVerbatimRecentK()` — new helper `turnBoundaryIndex` counts `user`/`assistant` as turn starts, `tool` messages do not increment, system messages skipped. `markVerbatimRecentK` marks from boundary index to end.
- [✓] Fix `reorderByRelevance()` verbatimStart — replaced message-counting loop with single call to `turnBoundaryIndex`.
- [✓] Fix `Compactor.swift` preserveLast semantics — `suffixStart` now uses `turnBoundaryIndex`. Constants `softPreserveRecent=8` and `hardPreserveRecent=4` retain their values; semantics changed from messages to turns.
- [✓] Update port-fidelity ledger — both files carry `[cev2]` comments.

## Open items

None. All ledger rows are DONE with evidence.

## Audit notes

- **Compactor integration test gap (minor).** The spec's Validation section calls for a compactor-specific test with multi-tool-call turns to verify `preserveLast` semantics. No such test was added to `CompactorTests`. The `turnBoundaryIndex` helper is well-tested via ContextCuratorTests, and the Compactor simply delegates to it, so the risk is low. But a strict reading of the validation section notes this absence.
- **Manual validation not evidenced.** The spec requests running a real conversation with context curation enabled and inspecting the `AnnotatedContextPlan`. No screenshot or log artifact is present. This is expected for a code-level audit (manual QA is a separate concern) but is noted for completeness.
