---
title: "Context engine test harness"
slug: cev2-context-engine-test-harness
type: feature
status: superseded
superseded_by: context-fixture-test-harness
superseded_by_peer:
  peer_id: cd8dd06d-3df1-4878-a88f-24593dcbb4b3
  peer_alias_display: hero-code
  peer_slug: context-fixture-test-harness
  successor_initiative: context-engine-v3
  reason: "Scope moved to hero-code's context-engine-v3 initiative; successor slug renamed to context-fixture-test-harness."
priority: high
size: medium
parent: context-engine-v2
created: 2026-06-09
relations:
  - target: context-fixture-test-harness
    kind: superseded-by
tags: [hero-code, swift, context-engine, testing, fixtures, regression]
---

# Context engine test harness

## Context

The existing test suite (`ContextCuratorTests.swift`, 389 lines) covers the
core curator algorithm with simple fixtures: single-tool-call turns,
user-only prose sequences, basic dedup/supersede scenarios. These tests are
faithful ports of the Rust v1 test suite and were sufficient when each
`SessionMessage` mapped 1:1 to one `LLMMessage`.

After the `toLLMMessages()` fix, the message structure changed fundamentally:
one assistant turn with N tool calls now expands to N+1 `LLMMessage`s. None
of the existing tests exercise this pattern. The test suite also lacks:

- **Multi-tool-call assistant turns** -- the dominant pattern in real
  sessions.
- **Bash-heavy conversation fixtures** -- Bash outputs are the most common
  tool output in practice.
- **Compaction summary messages** -- testing that summaries survive pruning.
- **`AgentLoop.outgoing()` integration tests** -- the production entry
  point, currently zero tests. This function wires `CurationOptions`,
  calls `ContextCurator.curate()`, and returns the curated list. Testing
  the curator in isolation doesn't catch wiring bugs.
- **End-to-end token reduction measurement** -- no test verifies that
  the curator actually reduces token count on a realistic conversation.

## Goal

Build a test harness with realistic conversation fixtures that exercises
every context engine component end-to-end. The harness provides reusable
fixture builders, standardized assertion helpers, and integration tests for
the production path (`AgentLoop.outgoing()`). Each child spec in the
context-engine-v2 initiative ships its own regression tests using this
harness.

## Approach

The harness is additive -- it extends the existing `ContextCuratorTests.swift`
and `CompactorTests.swift` files (or creates new focused test files alongside
them). It does not replace existing tests.

Three layers:

1. **Fixture builders** -- functions that construct realistic `[LLMMessage]`
   conversations. Parameterized by: number of turns, tool calls per turn,
   tool output size, whether to include compaction summaries, etc.

2. **Assertion helpers** -- functions that check common properties of
   curation results: verbatim window coverage, supersede counts, tag
   distributions, token reduction ratios.

3. **Integration tests** -- tests that call `AgentLoop.outgoing()` with
   realistic fixtures and verify the end-to-end behavior including option
   wiring.

## Changes

All files are in `../hero-code/apps/hero-desktop-mac/Tests/HeroDesktopTests/`.

1. **Create fixture builder utilities**
   (new file: `ContextEngineFixtures.swift` or extend existing test files)
   - `func buildMultiToolTurn(toolCount: Int, toolName: String, outputSize: Int) -> [LLMMessage]`
     -- Builds one assistant turn with N tool calls and N tool results,
     mimicking the `toLLMMessages()` expansion. Each tool call gets a
     unique `id`, the arguments field contains a realistic payload, and
     each tool result has `outputSize` characters of content.
   - `func buildRealisticConversation(turns: Int, avgToolsPerTurn: Int, bashFraction: Double) -> [LLMMessage]`
     -- Builds a multi-turn conversation mixing user messages, assistant
     messages with tool calls, and tool results. Parameterized so tests
     can control the shape.
   - `func buildConversationWithCompactionSummary() -> [LLMMessage]`
     -- Builds a conversation that includes a compaction summary user
     message (the "This session is being continued..." pattern).
   - `func buildBashHeavyConversation(uniqueCommands: Int, repeats: Int) -> [LLMMessage]`
     -- Builds a conversation where the same Bash commands run multiple
     times (for testing supersede).

2. **Create assertion helpers**
   (same file or a `ContextEngineAssertions.swift`)
   - `func assertVerbatimCovers(plan: AnnotatedContextPlan, lastTurns: Int, messages: [LLMMessage])`
     -- Asserts that the verbatim window covers exactly the last N turns
     (counting turn boundaries, not raw messages).
   - `func assertTokenReduction(plan: AnnotatedContextPlan, minReductionPct: Double)`
     -- Asserts that `estimatedTokens < rawTokens * (1 - minReductionPct)`.
   - `func assertSupersededCount(plan: AnnotatedContextPlan, expected: Int)`
     -- Wrapper for readability.
   - `func assertNoPrunedLoadBearing(plan: AnnotatedContextPlan)`
     -- Asserts that no `.loadBearing` message was pruned.

3. **Add AgentLoop.outgoing() integration tests**
   (new file: `AgentLoopOutgoingTests.swift` or extend existing)
   - Test that `AgentLoop.outgoing()` with `enabled: true` produces a
     curated list shorter than the input.
   - Test that `AgentLoop.outgoing()` with `enabled: false` returns the
     input unchanged.
   - Test that `verbatimK` override is threaded through to the curator.
   - Test that `overrides` (forceKeep/forceDrop) are applied correctly.
   - Test that `toolRetention` exempts tools from dedup/supersede.
   - Test that semantic scoring and reorder flags wire through without
     crash (functional verification requires a real scorer, but wiring
     correctness can be tested with a mock).

4. **Add end-to-end token reduction tests**
   - Build a realistic 50K-token conversation fixture.
   - Run the curator with default options.
   - Assert token reduction is meaningful (>10% for a clean conversation,
     >30% for a conversation with repeated Bash commands and stale reads).
   - This test serves as a regression gate: if future changes reduce
     curation effectiveness, the test fails.

5. **Add regression tests for each child spec's fix**
   - Each child spec (cev2-verbatim-turn-counting, cev2-protect-compaction-
     summaries, etc.) defines its own regression tests in its Validation
     section. This harness provides the fixture builders and assertion
     helpers those tests use. The tests themselves ship with the child spec,
     not with the harness.

## Boundaries

- Do NOT refactor or rewrite existing tests. The existing tests are
  faithful ports of v1 and must continue to pass unchanged.
- Do NOT mock the provider for curator tests. The curator is stateless and
  pure -- it takes `[LLMMessage]` and returns curated messages + plan.
  No mocking needed. Provider mocks are only needed for Compactor tests
  (which involve LLM summarization calls).
- Do NOT test the `Session.toLLMMessages()` expansion here. That function
  has its own test scope. This harness tests the curator and agent loop
  with pre-expanded `[LLMMessage]` arrays.
- Do NOT introduce SwiftUI/AppKit test dependencies.

## Risks

- **Fixture drift.** If the `LLMMessage` struct or `toLLMMessages()`
  expansion logic changes, the fixtures become unrealistic. Mitigate by
  documenting the expected message structure in fixture builder comments
  and cross-referencing `Session.swift` line 231.
- **Test execution time.** A 50K-token fixture is large. The curator is
  fast (all in-memory, no I/O), but the test must not become a CI
  bottleneck. Measure execution time; if >1s, reduce the fixture size
  while keeping the test meaningful.
- **AgentLoop.outgoing() is `nonisolated static`.** It's directly callable
  in tests without instantiating an `AgentLoop`. No concurrency concerns
  for testing.

## Validation

- **All existing tests pass unchanged** after the harness is added.
- **Fixture builders produce valid message arrays:** Each builder's output
  passes a structural sanity check (tool results follow their assistant
  turns, tool_call IDs match, no orphaned tool results).
- **Assertion helpers catch known violations:** Test each helper against a
  known-bad plan to verify it correctly fails.
- **AgentLoop.outgoing() tests pass:** At least 5 integration tests
  covering enabled/disabled, verbatim override, force keep/drop, tool
  retention, and the default path.
- **CI green:** The full test suite runs in CI without flakiness or
  timeout.
