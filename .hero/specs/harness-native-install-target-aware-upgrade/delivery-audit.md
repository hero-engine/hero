# Delivery audit — harness-native-install-target-aware-upgrade

**Audited:** working tree vs `HEAD` (60169ac); delivery is uncommitted (13 modified files + 2 untracked test files). Base `3f272ed` per invocation; note the `init.go`/`init_gitignore_test.go` deltas in that base-diff belong to intervening commit `f1b5388` (gitignore fix), not this delivery.
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria

- [✓] `hero install --target claude` in a clean repo → CLAUDE.md, NOT AGENTS.md — `target_claude.go:40-44` drops `installAgentsMd`, routes through `installNativeInstructionFile`; asserted in `harness_native_test.go:48` (mustNotExist AGENTS.md), `install_test.go:747` (`AGENTS.md must NOT be created`), `harness_smoke_test.go:49`.
- [✓] `hero install --target <non-claude>` → AGENTS.md, NOT CLAUDE.md — cursor/copilot/generic add `installNativeInstructionFile` (`target_cursor.go:37`, `target_copilot.go:93`, `target_generic.go:38`); codex/opencode switched to it. `harness_native_test.go:49-53` asserts each of the six writes its native file and the other is absent.
- [✓] Multi-target incl. claude → both files, same managed body — `harness_native_test.go:70` (both present), `:87` (`TestHarnessNative_SameManagedBody` asserts byte-identical managed body for claude+cursor). See Audit note on the claude+codex case.
- [✓] Persist installed target set — `RecordTargetInstall` unchanged as writer; `harness_native_test.go:149` asserts `PreviouslyInstalledTargets` returns claude+codex after multi-install.
- [✓] Upgrade regenerates managed region only in native files of previously-installed targets — `resolveUpgradeTargets` (`upgrade.go:414`) = `PreviouslyInstalledTargets` ∪ `detectInstalledTargets`; `upgrade_target_aware_test.go:37,67`.
- [✓] claude ∉ prior targets → upgrade never creates CLAUDE.md — `TestUpgrade_ClaudeNeverInstalled_NoCLAUDEMd` (`upgrade_target_aware_test.go:37`) persists opencode-only, asserts no CLAUDE.md.
- [✓] claude ∈ prior targets → CLAUDE.md managed region regenerated — `TestUpgrade_ClaudeInstalled_RegeneratesCLAUDEMd` (`:67`) asserts STALE body replaced, markers present, no phantom AGENTS.md.
- [✓] No persisted state → infer + persist + no files for un-inferred targets — `InferInstalledTargets` (`state.go`), backfill wired at `upgrade.go:150-158`; `TestUpgrade_Backfill_StubOnly` (`:111`) asserts inferred {claude} persisted, CLAUDE.md regenerated, no AGENTS.md conjured.
- [✓] Preserve user content outside markers byte-for-byte — `TestOrphanPolicy_PreservesUserContentEvenWithProve` (`harness_native_test.go:277`) + `TestUpgrade_OrphanAgentsMd_MaintainedByDefault` (`upgrade_target_aware_test.go:145`, asserts `USER KEEP` retained).
- [✓] Idempotent (2nd run → zero changes) — `TestHarnessNative_Idempotent` (`:128`) + `TestHarnessNative_TwoNonClaudeTargetsShareAgentsMd` (`:104`, byte-for-byte no-op on 2nd non-claude write).
- [✓] No auto-delete of AGENTS.md/CLAUDE.md; deletion opt-in only + managed-only — `ApplyOrphanInstructionFilePolicy` (`agents_md.go:169`) requires `prune==true` AND `instructionFileIsHeroManagedOnly`; `TestOrphanPolicy_NoPruneNeverDeletes` (`:296`) → OrphanMaintained.
- [✓] Prune deletes managed-only orphan, preserves user-content file — `TestOrphanPolicy_PrunesManagedOnly` (`:258`) + `TestOrphanPolicy_PreservesUserContentEvenWithPrune` (`:277`); CLI-level `TestUpgrade_OrphanPrune_OptIn` (`:180`).
- [✓] Define behavior for all six targets — `nativeInstructionFile` mapping + per-target routing; `TestHarnessNative_PerTargetFileSet` table covers claude/codex/opencode/cursor/copilot/generic.

## Changes

- [✓] `agents_md.go` — `nativeInstructionFile`, `installNativeInstructionFile`, `instructionFileIsHeroManagedOnly`, `ApplyOrphanInstructionFilePolicy` (+ orphan-action constants); global-mode `resolveAgentsMdPath` returns "" for cursor/copilot/generic.
- [✓] `target_claude.go` — `installAgentsMd` block removed; CLAUDE.md via `installNativeInstructionFile`.
- [✓] `target_cursor.go`/`target_copilot.go`/`target_generic.go` — emit AGENTS.md via `installNativeInstructionFile`; copilot keeps separate `.github/copilot-instructions.md`; generic replaced legacy `installInstructionsMd` stub.
- [✓] `target_codex.go`/`target_opencode.go` — switched to `installNativeInstructionFile` (behavior unchanged: AGENTS.md).
- [✓] `state.go` — `PreviouslyInstalledTargets`, `InferInstalledTargets`, `claudeMdIsHeroManaged`, `PersistInferredTargets`; `RecordTargetInstall` unchanged. `ReadInstallState` always returns a non-nil `Targets` map (no nil-map panic in persist path).
- [✓] `upgrade.go` — union persisted+detected, backfill for empty state, `--prune-orphaned-instruction-files`, `handleOrphanedInstructionFiles`, `unionTargets`, help text.
- [✓] `install.go` — `--prune-orphaned-instruction-files`, harness-native `Long` help, post-install prune wiring (includes just-installed target in resolved set so it never prunes what it wrote).
- [✓] `check.go` — informational `orphan-instruction-files` note (`detectOrphanInstructionFiles`, `fileExists`); warns, does not fail.
- [✓] `web/docs/src/cli/server-and-mcp.md` — harness-native model, per-target mapping, persisted-target behavior, migration + prune flag (+54 lines).
- [~] `cmd/hero/...` help text — cmd/hero/main.go NOT touched; help surface lives in the cobra `Long` fields in `install.go`/`upgrade.go`, which WERE updated. Documented in the spec's Delivered section; the intent (updated help) is satisfied, just in the correct location.

## No-destructive-delete invariant (traced)

Confirmed. Two guards, both required for deletion:
- `ApplyOrphanInstructionFilePolicy` (`agents_md.go:169`): absent → OrphanAbsent (never creates); no managed region → OrphanPreserved (untouched); `prune && instructionFileIsHeroManagedOnly` → delete; else → maintain-in-place.
- `instructionFileIsHeroManagedOnly` (`agents_md.go:103`): requires a managed region, `TrimSpace(suffix) == ""`, and prefix empty or exactly the Hero default H1. Whitespace-tolerant (TrimSpace); a user H1 like `# My Project` fails the prefix check → not deletable. Any content after the markers → false. This is correct and conservative.
- No default path calls delete: `handleOrphanedInstructionFiles` is called with `prune` = the flag value; without the flag orphans are only maintained. Install prune path (`install.go:334`) is gated on `installPruneOrphans`.

## Backfill ambiguity (traced)

`InferInstalledTargets`: content-dir probe (`DetectInstalledTargets`) is authoritative for the SET; CLAUDE.md-with-managed-region only adds claude (never a non-claude target); a lone AGENTS.md never conjures a target. All four cases have dedicated tests: both-files-only-claude-content → {claude} (`:194`), stub-only → {claude} (`:210`), neither → {} (`:221`), agents-only-non-claude-content → {generic}, claude excluded (`:230`); plus user-CLAUDE.md-not-inferred (`:246`).

## Open items

- Cross-repo advisory to hero-code peer (`hihcp-agents-md-harness-agnostic`, "demote CLAUDE.md") — spec §"Cross-repo coordination (pending)": dry-run OK but live dispatch failed 401, NOT delivered. Owed follow-up when peer auth is available. Not an AC of this spec; flagged so it isn't forgotten. — concrete.

## Audit notes

- **claude+codex multi-target body is not byte-identical.** AC#3 says "same Hero-managed block body." The `SameManagedBody` test deliberately uses claude+cursor because codex appends a codex-specific workflow addendum to its managed body — a pre-existing P1 behavior explicitly out of scope ("same body generator as P1, unchanged"). The AC's spirit (shared generator, no phantom stripped file) holds; the codex addendum predates this work. Not a downgrade, but the literal "byte-identical for every multi-target combo" reading has this one documented exception.
- **`init.go` is not this delivery.** `hero init` independently generates a project-context AGENTS.md (`init.go:206 generateAgentsMD`, detected build/test/lint commands) — a separate command and surface from `hero install`. Working-tree diff for init.go vs HEAD is empty; the delta seen in the `3f272ed` base-diff is from commit `f1b5388`. Does not contradict the clean-repo `hero install --target claude` AC (which governs `hero install`, tested via the install harness with no init run).
- **Legacy tests flipped, not deleted.** `install_test.go` and `harness_smoke_test.go` old "both files for claude" assertions were rewritten to "CLAUDE.md only, AGENTS.md must NOT exist" with live assertions.
- **Verification:** `go build ./...` clean; `go test ./internal/install ./internal/cli` → ok. Matches the reported 86-package / 0-FAIL run. No codescan files in the delivery diff (no regression surface). Scope confined to internal/install, internal/cli, web/docs (+ .hero governance edits, expected).
