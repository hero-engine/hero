---
title: "Compact Handoff Test Coverage — Close MVP Coverage Gaps"
slug: compact-handoff-test-coverage
type: feature
status: delivering
priority: medium
horizon: now
tags: [tests, quality, compact-handoff, regression-prevention]
relations:
  - target: next-compact-handoff
    kind: hardens
---

# Compact Handoff Test Coverage — Close MVP Coverage Gaps

## Problem

The [next-compact-handoff](../next-compact-handoff/spec.md) MVP shipped with 21 tests that cover the safety-critical paths: session-id filtering correctness, settings-file preservation, JSON validity, idempotent install, and the most common graceful-degradation cases. Build and test suites are green.

But the coverage has real gaps that will hurt later. Concretely:

1. **No end-to-end "assembly with content" test.** Every unit test exercises one section or one helper. Nothing verifies that a populated session (with active spec, decisions, files touched, kickoff, next suggestion) actually assembles into the correct full markdown shape. A regression that breaks section ordering, header text, or inter-section spacing would land green.
2. **Truncation cascade is one-step deep.** The spec defines a five-step truncation order (Working tree → Files touched tail → Recent decisions tail → Active spec body → Original kickoff). Only the first step is tested. The other four can silently drift.
3. **`hero init` auto-install path has no test.** The new behavior — `hero init` installs SessionStart{compact} by default unless `--no-hooks` — isn't covered. A change to init flow could silently disable the feature for new installs.
4. **Defer-panic + always-exit-0 safety contract is not tested.** This is the most safety-critical property: the hook must never block compaction. A simulated panic with verification that the process still exits 0 with a valid envelope is missing.
5. **Settings-file edge cases.** What if `.claude/settings.json` doesn't exist? Has invalid JSON? Has a `SessionStart` array but no `compact` matcher? Has a `compact` matcher with non-Hero entries already? Each is a real scenario; only the "fresh repo" and "existing entries preserved" cases are tested today.
6. **Active spec body extraction edges.** Frontmatter stripping, 6KB body truncation with the `… (truncated — read full at <path>)` suffix, and Goal-section extraction for "What you were doing" all run without dedicated tests.
7. **Original-kickoff transcript fallback.** When no UserAsk is in the graph for this session, the spec says fall back to the first user prompt from `transcript_path`. The transcript-read fallback isn't exercised.
8. **Codex stub behavior.** `--host=codex` should print a clear unsupported notice and exit 0; there's no test asserting the message shape or exit code.
9. **`--host=all` semantics.** Should install git hooks AND host-tool hooks; current tests cover each independently.
10. **Remove-then-reinstall.** Marker idempotency should survive an uninstall+reinstall cycle. Currently tested only "install twice."

The MVP is shippable as-is — these are quality gaps, not correctness defects. But the next person touching this code (or a future refactor) needs a denser test surface to land changes safely.

## Goal

Add a focused set of tests that close the above gaps. Specifically:

- One **integration-style assembly test** that exercises the full handoff path from a populated graph + active-session registry → JSON envelope, asserting structure and content.
- **Full truncation cascade coverage** — one test per step, plus an "extreme over-budget" test that verifies the final preserved-content invariant (header + active spec slug + next concrete action).
- **Safety-contract tests** for defer-panic + always-exit-0.
- **Settings-file edge case suite** covering missing file, invalid JSON, partial matchers, and non-Hero entries inside a `compact` matcher.
- **Codex stub + `--host=all` tests.**
- **Content extraction tests** for frontmatter stripping, body truncation, Goal extraction, transcript-fallback kickoff.
- **`hero init --no-hooks` and `hero init` default behavior tests.**

Stretch goal: package coverage on the touched packages moves from 25–35% (cli/hooks) and 73% (projection) to ≥60% on cli/hooks (for the new files specifically) and ≥85% on the new projection file. Coverage isn't the goal — these gaps are — but tracking the number gives a concrete signal that the work landed.

## Design

### Test files to add or extend

```
internal/cli/next_compact_handoff_test.go   (extend)
  + TestAssembleFullHandoff_PopulatedSession
  + TestEnforceTokenCap_FullCascade
  + TestEnforceTokenCap_PreservesInvariants
  + TestRunCompactHandoff_PanicRecoveryReturnsValidEnvelope
  + TestRunCompactHandoff_AlwaysExitsZeroOnBadStdin
  + TestExtractGoalSection_PreservesIntent
  + TestStripFrontmatter_HandlesAllShapes
  + TestTruncateSpecBody_AppendsReadFullSuffix
  + TestKickoffFromTranscript_WhenNoUserAsk

internal/cli/host_hooks_test.go             (new)
  + TestHostHooksInstall_AllInstallsGitAndClaude
  + TestHostHooksInstall_CodexPrintsUnsupportedAndExitsZero
  + TestHostHooksStatus_ReportsPerHostState
  + TestHostHooksUninstall_AllRemovesGitAndClaude

internal/cli/init_compact_hook_test.go      (new)
  + TestInit_DefaultInstallsCompactHook
  + TestInit_NoHooksFlagSkipsCompactHook
  + TestInit_IsIdempotentForCompactHook

internal/hooks/claude_settings_test.go      (extend)
  + TestInstall_MissingSettingsFile_CreatesIt
  + TestInstall_InvalidJSON_ErrorsCleanlyNoMutate
  + TestInstall_SessionStartArrayExistsNoCompact_AddsCompactEntry
  + TestInstall_CompactMatcherHasUserEntry_HeroEntryAddedAlongside
  + TestUninstall_NoHeroEntries_IsNoOp
  + TestRemoveThenReinstall_PreservesIdempotency

internal/projection/compact_handoff_test.go (extend)
  + TestCollectSessionEvents_SpecAnchoredCarryover_ExcludesUserAskFromOthers
  + TestCollectSessionEvents_BothDirectionsForSameSpec
  + TestFilesTouched_DeduplicatesAcrossEvents
```

### Integration assembly test — the most important addition

`TestAssembleFullHandoff_PopulatedSession` builds a fixture with:

- A temp project root with `.hero/` and a real graph store
- An active-session registry entry: session_id "test-session" → spec "fixture-spec"
- A fixture spec at `.hero/planning/features/fixture-spec/spec.md` with frontmatter + Goal + Design sections
- Graph events tagged with "test-session": one UserAsk (the kickoff), two Decisions, three Attempts touching different files, one NextSuggestion
- Working tree with two dirty files

Then calls the full assembly path and asserts:

- Header line matches `**Session:** test-session · started <RFC3339> · <elapsed>` (parse tolerant on the timestamp)
- Active spec line names the fixture spec slug and links to its path
- "What you were doing" excerpts the Goal section, ≤300 chars
- "Active spec — full content" contains the spec body with frontmatter stripped
- "Original kickoff" contains the UserAsk text
- "Files touched this session" lists all three Attempt files, deduplicated, with event counts
- "Recent decisions" lists both Decision events newest-first
- "Next concrete action" reflects the NextSuggestion
- "Working tree" lists the two dirty files
- Total length within token cap

This is the test that catches accidental section reordering, missing blank lines, or wrong header text — the kind of regression that breaks the model's parsing without breaking any unit test.

### Truncation cascade

Five tests, each starting from a known-budget overage that forces exactly one step of truncation:

1. Overage 50 lines → only Working tree dropped
2. Overage 200 lines → Working tree dropped, Files touched tail trimmed
3. Overage 500 lines → previous + Recent decisions tail trimmed
4. Overage 2000 lines → previous + Active spec body re-truncated to a smaller cap
5. Overage 4000 lines → previous + Original kickoff truncated harder

Plus a single `TestEnforceTokenCap_PreservesInvariants` that takes an absurdly long input and asserts that the final output still contains the session header, the active spec slug, and the Next concrete action line — the three invariants the spec promises never to drop.

### Safety contract — panic recovery + exit code

`TestRunCompactHandoff_PanicRecoveryReturnsValidEnvelope`: inject a panic-inducing condition (e.g. malformed transcript path that crashes deep in assembly) and assert that:

- Process exits 0
- Stdout contains a valid JSON envelope (parses as the expected schema)
- `additionalContext` is either empty or a one-line "handoff unavailable" message — not garbage

`TestRunCompactHandoff_AlwaysExitsZeroOnBadStdin`: feed malformed JSON on stdin and assert exit 0 + valid envelope. Today only "no stdin → error" is tested.

These are the most consequential tests in the spec. A non-zero exit from this command would block compaction in real users' sessions. Has to be untestably-safe.

### Settings-file edge cases

The four scenarios in [Problem](#problem) — missing file, invalid JSON, partial matcher arrays, non-Hero entries alongside Hero entries — each get a dedicated test. Idempotency across remove-then-reinstall gets one more.

### Init flow

Three tests for the `hero init` auto-install path: default behavior, `--no-hooks` flag, idempotent re-run. These can share a temp-dir fixture.

### Coverage measurement

Add a `make test-coverage` target (or extend the existing one if present) that emits per-package and per-file coverage for the touched files. Useful for the next person reviewing changes here; not load-bearing.

## Out of scope

- **Refactoring the production code for testability.** If `runCompactHandoff` is hard to test because of tight coupling, file a separate spec for the refactor — don't conflate it with coverage work.
- **Codex live install tests.** Stub-only. Once the Codex installer lands for real (see next-compact-handoff follow-ups), add tests then.
- **End-to-end test through a real Claude Code compaction.** Requires running Claude Code; out of scope for unit/integration tests. A manual smoke checklist in `docs/` is acceptable instead.
- **Test the LLM-curated middle section.** That's part of [compact-handoff-summarizer](../compact-handoff-summarizer/spec.md).
- **Performance benchmarks.** Test the compact handoff completes fast enough not to delay compaction — useful but separate.

## Risks

- **Test fixtures get unwieldy.** The integration test needs a populated graph + spec corpus. Use a shared `setupCompactHandoffFixture(t)` helper so each test gets a clean copy without 200 lines of setup. Borrow patterns from existing `internal/projection/*_test.go` setups.
- **Flaky tests if the integration test depends on real time.** The "elapsed since started" line needs a tolerant matcher. Use a clock-injection pattern or regex match against `Xh Ym` shape rather than exact strings.
- **Coverage targets become a goal in themselves.** Avoid writing tests just to hit a number. Each test in this spec exists because of a real gap. If the coverage moves less than the target but the gap list is closed, that's a pass.
- **Overlapping with the follow-up summarizer spec.** The LLM-curated middle will eventually splice into the same assembly path. Tests written here should focus on the deterministic skeleton only; assertions on the splice point come with the summarizer work.
- **Test-only helpers leaking into production code.** Keep test fixtures in `_test.go` files. If a test needs to inspect internal state, prefer making the relevant function exported-for-testing rather than building a test-only escape hatch.

## Acceptance Criteria

- [ ] `TestAssembleFullHandoff_PopulatedSession` exists and validates the full envelope structure from a populated fixture (active spec, kickoff, files touched, decisions, next action, working tree).
- [ ] Truncation cascade has one test per step (5 total) plus `TestEnforceTokenCap_PreservesInvariants` asserting header + spec slug + next-action are never dropped.
- [ ] `TestRunCompactHandoff_PanicRecoveryReturnsValidEnvelope` and `TestRunCompactHandoff_AlwaysExitsZeroOnBadStdin` verify the always-exit-0 + valid-envelope contract.
- [ ] Settings-file edge cases covered: missing file, invalid JSON (errors cleanly without mutating), `SessionStart` array exists without `compact` matcher, `compact` matcher exists with non-Hero entries.
- [ ] `hero init` default behavior, `--no-hooks` skip, and idempotency are tested.
- [ ] `--host=all` install/uninstall and Codex stub messaging covered.
- [ ] Content-extraction tests: frontmatter stripping, 6KB body truncation with proper suffix, Goal-section extraction.
- [ ] Original-kickoff transcript fallback test exists (when no UserAsk in graph, reads from `transcript_path`).
- [ ] Spec-anchored carryover test covers both directions (session A sees session B's spec-anchored events and vice versa) and excludes UserAsk/NextSuggestion event types from other sessions.
- [ ] Per-file coverage on the new code reaches: `next_compact_handoff.go` ≥60%, `claude_settings.go` ≥80%, `compact_handoff.go` (projection) ≥85%. (Targets, not gates — gap closure is the real bar.)
- [ ] All new tests pass on `go test ./...` and run in <10s combined.
- [ ] No production code modified except where strictly required for testability (and any such change documented in the spec change log).

## Changes

- `internal/cli/next_compact_handoff_test.go` — extended with 9 new tests (integration fixture co-located here per existing test style; no separate fixture file)
- `internal/cli/host_hooks_test.go` — new file, 4 tests covering `--host=all`, `--host=codex` stub, status, uninstall
- `internal/cli/init_compact_hook_test.go` — new file, 3 tests covering `hero init` auto-install path
- `internal/hooks/claude_settings_test.go` — extended with 6 new tests covering settings-file edge cases
- `internal/projection/compact_handoff_test.go` — extended with 3 new tests covering spec-anchored carryover bidirectionality and file dedup
- Makefile `test-coverage` target deferred — not needed to land the gap closures; can be added later as a small follow-up

## Kickoff

Add the test coverage that the MVP shipped without. The most consequential additions are: (1) the full-assembly integration test that exercises a populated session end-to-end, and (2) the panic-recovery + always-exit-0 safety tests for the hook command.

Read first:
- [next-compact-handoff](../next-compact-handoff/spec.md) for the full content shape and truncation order being verified.
- `internal/cli/next_compact_handoff.go` to understand the assembly path being exercised.
- `internal/projection/compact_handoff.go` for the graph query.
- `internal/hooks/claude_settings.go` for the settings-file mutation surface.

Then build in roughly this order:

1. Shared test fixture helper for populated graph + active session registry + fixture spec on disk. Keep it small and reusable.
2. `TestAssembleFullHandoff_PopulatedSession` — the integration test. This will surface any small assembly bugs that the unit tests missed; expect to iterate once before it passes.
3. Truncation cascade (5 + invariant test).
4. Panic recovery + bad-stdin safety contract tests.
5. Settings-file edge cases (six tests).
6. Init flow (three tests).
7. Codex stub + `--host=all` tests.
8. Projection-side spec-anchored carryover bidirectionality.

Run `go test -cover ./internal/cli/... ./internal/projection/... ./internal/hooks/...` before and after to confirm the coverage move. Do not gate on the percentage; gate on the gap list above being closed.
