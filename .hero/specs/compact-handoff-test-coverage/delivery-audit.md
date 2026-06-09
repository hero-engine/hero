# Delivery Audit — compact-handoff-test-coverage

**Date:** 2026-06-09
**Auditor:** claude-sonnet-4-6
**Verdict:** SHIP
**Surface:** clean
**Confidence:** high

---

## What was audited

Spot-checked 4 of 5 test files at source level, ran all 25 named tests, and
verified the delivery commit touched no production code.

## Files delivered

| File | Type | Description |
|---|---|---|
| `internal/cli/next_compact_handoff_test.go` | extended | 9 new tests: integration assembly, 5-step truncation cascade, panic recovery, bad-stdin exit-0, frontmatter strip, Goal extraction, body truncation suffix, transcript-fallback kickoff |
| `internal/cli/host_hooks_test.go` | new | 4 tests: `--host=all` installs git + Claude + Codex, Codex real install, status report, `--host=all` uninstall |
| `internal/cli/init_compact_hook_test.go` | new | 3 tests: default installs compact hook, `--no-hooks` skips it, idempotency |
| `internal/hooks/claude_settings_test.go` | extended | 6 new edge-case tests: missing file creates it, invalid JSON errors without mutation, SessionStart array + no compact entry, compact matcher with user entry, no-Hero uninstall is no-op, remove-then-reinstall idempotency |
| `internal/projection/compact_handoff_test.go` | extended | 3 new tests: spec-anchored carryover excludes UserAsk/NextSuggestion from other sessions, bidirectional carryover, file-touched dedup |

No production code modified (confirmed via `git show a5621a4 --stat`).

---

## AC-by-AC findings

### AC-1 — TestAssembleFullHandoff_PopulatedSession (full envelope)

PASS. The test builds a full fixture (git repo, active-session registry, spec on disk, UserAsk + 2 Decisions + 3 Attempts + NextSuggestion + 2 dirty files in working tree), calls `buildHandoff` end-to-end, and asserts:

- Header regex matches `**Session:** test-session · started <RFC3339> · <elapsed>`
- Active spec line contains `[fixture-spec]` link
- "What you were doing" contains Goal text, ≤300 chars
- "Active spec — full content" section: frontmatter absent, H1 present
- "Original kickoff" contains the UserAsk text
- "Files touched this session" names all 3 Attempt paths
- "Recent decisions" contains both Decision titles, ordered newest-first
- "Next concrete action" contains the NextSuggestion text
- "Working tree" contains both dirty file names
- Total approxTokens within compactHandoffTokenHardCap

Assessment: non-trivial, exercises the full pipeline.

### AC-2 — Truncation cascade (5 steps + invariants)

PASS. `TestEnforceTokenCap_FullCascade` has 5 subtests, each seeding a
precondition that exceeds the cap via a specific dominant section:

1. `step1_dropWorkingTree` — 1500-line working tree; asserts cap enforced, tree entries dropped, next action survives.
2. `step2_trimFilesTouched` — 40 deep paths via `makeFilesTouchedHeavy`; cap enforced.
3. `step3_trimDecisions` — 40 verbose decision titles via `makeDecisionsHeavy`; cap enforced.
4. `step4_shrinkSpecBody` — 4000-repeat spec body; cap enforced.
5. `step5_shrinkKickoff` — 2000-repeat kickoff text; cap enforced.

`TestEnforceTokenCap_PreservesInvariants` drives a massively over-budget input
(all sections simultaneously huge) and asserts `**Session:** sess-X`, the slug
`fixture-spec`, and the full next-action string all survive.

Assessment: genuine cascade coverage; each step independently exercises a truncation branch.

### AC-3 — Panic recovery + always-exit-0 safety contract

PASS. Two tests:

`TestRunCompactHandoff_PanicRecoveryReturnsValidEnvelope`:
- Uses a `panicReader` (`Read` panics) injected as cobra stdin.
- Sets `compactHandoffJSON = true`, calls `emitJSONEnvelope`.
- Asserts `emitJSONEnvelope` returns `nil` (no error propagation).
- Asserts `out.Len() > 0` and the output parses as valid JSON with `hookSpecificOutput.hookEventName == "SessionStart"`.

The test correctly exercises the production defer-recover path in `emitJSONEnvelope` — the panic propagates up through `resolveSessionID` → `emitJSONEnvelope`'s defer catches it, writes the fallback envelope, returns nil.

`TestRunCompactHandoff_AlwaysExitsZeroOnBadStdin`:
- Feeds `{ this is not valid json` as stdin.
- Asserts nil return, non-empty output, valid JSON envelope with `additionalContext` field present.

Assessment: the panic recovery test is real; it uses a genuine panic injection, not a stub assertion.

### AC-4 — Settings-file edge cases

PASS. Six tests covering all specified scenarios:

- `TestInstall_MissingSettingsFile_CreatesIt` — verifies `.claude/` dir absent as precondition, calls install, asserts dir + file created, asserts 1 SessionStart entry.
- `TestInstall_InvalidJSON_ErrorsCleanlyNoMutate` — seeds `{ this is not valid json`, asserts install returns `(false, non-nil error)`, asserts file byte-for-byte unchanged.
- `TestInstall_SessionStartArrayExistsNoCompact_AddsCompactEntry` — seeds a `startup` matcher entry, asserts 2 entries post-install with both `startup` and `compact` matchers present.
- `TestInstall_CompactMatcherHasUserEntry_HeroEntryAddedAlongside` — seeds a user-authored `compact` matcher (`my-custom-compact-tool`), asserts 2 entries with user's command intact and no Hero marker on it, Hero entry added with `added_by_hero=true`.
- `TestUninstall_NoHeroEntries_IsNoOp` — asserts `removed=false`, file byte-for-byte unchanged.
- `TestRemoveThenReinstall_PreservesIdempotency` — install → uninstall (assert `removed=true`) → install (assert `installed=true`) → install again (assert `installed=false`/no-op) → asserts exactly 1 entry survives.

Assessment: all four spec-specified scenarios covered, tests check meaningful file-state invariants, not trivial returns.

### AC-5 — hero init flow

PASS. Three tests in `init_compact_hook_test.go`:

- `TestInit_DefaultInstallsCompactHook` — runs `hero init` via `runCmd`, calls `hooks.ClaudeCompactHandoffStatus`, asserts true.
- `TestInit_NoHooksFlagSkipsCompactHook` — runs `hero init --no-hooks`, asserts status is false.
- `TestInit_IsIdempotentForCompactHook` — runs `hero init`, then calls `hooks.InstallClaudeCompactHandoff` directly, asserts `installed=false` (no duplicate).

### AC-6 — --host=all and Codex stub

PASS. `TestHostHooksInstall_AllInstallsGitAndClaude` verified via `runCmd("hooks", "install", "--host=all")`:
- Asserts git pre-commit hook file exists via `os.Stat`.
- Asserts `ClaudeCompactHandoffStatus=true`.
- Asserts `CodexCompactHandoffStatus=true`.
- Asserts output contains both `"claude SessionStart{compact}"` and `"codex SessionStart{compact}"`.

`TestHostHooksInstall_CodexInstallsToProjectFile` verified the Codex installer is real (not a stub — the file `.codex/hooks.json` must exist on disk, `CodexCompactHandoffStatus` returns true, output contains `"trust"` and `"codex_hooks"`).

Note: the spec named this test `TestHostHooksInstall_CodexPrintsUnsupportedAndExitsZero` but the delivered test is `TestHostHooksInstall_CodexInstallsToProjectFile`. The test name reflects that the Codex installer landed as a real installer (not a stub) in a follow-on commit (`956f8c2`). The behavior tested is correct and stronger than the stub version the spec planned.

### AC-7 — Content extraction

PASS. `TestStripFrontmatter_HandlesAllShapes` (3 subtests: with trailing newline, no frontmatter, unterminated frontmatter). `TestExtractGoalSection_PreservesIntent` verifies first-paragraph extraction and 300-char cap. `TestTruncateSpecBody_AppendsReadFullSuffix` exercises the 6KB body truncation path and asserts `"truncated — read full at"` + spec path in output.

### AC-8 — Transcript-fallback kickoff

PASS. `TestKickoffFromTranscript_WhenNoUserAsk` registers a session with no spec (no graph events), writes a real JSONL transcript, calls `buildHandoff` end-to-end, asserts kickoff text in output and no placeholder. Additional coverage from `TestKickoffForSession_*` family (8 more tests).

### AC-9 — Spec-anchored carryover bidirectionality + exclusions

PASS. Three new tests in `compact_handoff_test.go`:

- `TestCollectSessionEvents_SpecAnchoredCarryover_ExcludesUserAskFromOthers` — spec-anchored Decision from sess-OTHER appears; UserAsk and NextSuggestion from sess-OTHER do NOT contribute paths to FilesTouched.
- `TestCollectSessionEvents_BothDirectionsForSameSpec` — sess-A sees B's decision, sess-B sees A's decision, both via spec carryover.
- `TestFilesTouched_DeduplicatesAcrossEvents` — 3 Attempt nodes referencing same file → exactly 1 entry with count=3.

### AC-10 — Coverage targets (advisory)

Commit message reports: `buildHandoff 96%`, `enforceTokenCap 100%`, `InstallClaudeCompactHandoff 100%`, `CollectSessionEvents 94%`. All exceed spec advisory targets (≥60% cli, ≥80% claude_settings, ≥85% projection). Not re-measured in this audit (targets are advisory per spec).

### AC-11 — Runtime under 10s

PASS. Measured wall time: `internal/cli` 1.505s, `internal/hooks` 0.193s, `internal/projection` 0.411s. Total ~2.1s.

### AC-12 — No production code modified

CONFIRMED. `git show a5621a4 --stat` shows 5 `_test.go` files added and `.hero/` metadata updated only. Zero production code changes.

---

## Noteworthy observations

**Codex test name drift:** The spec planned `TestHostHooksInstall_CodexPrintsUnsupportedAndExitsZero` (a stub test). What landed is `TestHostHooksInstall_CodexInstallsToProjectFile` — a stronger real-installer test. This is an upgrade, not a gap; the spec's "codex stub" AC is satisfied by testing the actual installer that shipped in `956f8c2`.

**Coverage is exceptional:** The numbers cited in the commit message (`buildHandoff 96%`, `enforceTokenCap 100%`) significantly exceed the advisory spec targets. The test suite is denser than the spec required.

---

## Verdict

All 12 ACs satisfied. Tests are real (non-trivial assertions against actual behavior), pass in ~2s, and no production code was touched. The panic-recovery test uses genuine panic injection and validates the full envelope contract. The settings edge cases verify file-level invariants, not just return values.

**SHIP.**
