# Delivery audit — doctor-install-target-table

**Audited:** `git diff main...HEAD` (commit `aba3fa4`, branch `feat/doctor-install-target-table`, one commit ahead of `main`)
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria
- [✓] AC-1 section rendered between Workspace graph and Verdict, one row per installed target with expected/actual + root file — `doctor.go:161` (call site after graph block, before `doctorVerdict` at :163); test `healthy install table renders rows` asserts ordering `gi < si < vi` (`doctor_test.go:147-153`).
- [✓] AC-2 expected counts from `install.EnumerateContent` over active-domain overlay FS — `inventory.go:76,86`; `buildTargetInventory` uses `m.Agents/Commands/Skills` (`inventory.go:105-116`). No parallel count map.
- [✓] AC-3 domain via `activeDomainForRoot`, falls back to engineering — `doctor.go:103`; `Inventory` defaults `""`→`engineering` (`inventory.go:69-71`).
- [✓] AC-4 correct row per target using real dest paths — `targetInstallPaths` (`inventory.go:142-179`) with per-target dirs incl. cursor flat skills (`countFlatMD` at :160), codex TOML (`countFlatTOML` at :164), copilot `.prompt.md` (`countFlatPromptMD` at :169-170), `.agents/skills` (:166). Test `TestInventory_AllSixTargets` exercises all six.
- [✓] AC-5 codex commands `—` never `0`, skills expected = `len(Skills)+len(Commands)` — `inventory.go:112-113`; `NotApplicable` modeled in `KindCount` type (`inventory.go:33-37`), never `Expected:0`. Tests `TestInventory_CodexCommandsNotApplicable`, `TestInventory_CodexSkillsMatchInstalledDirs`.
- [✓] AC-6 codex footnote present only when codex present — `doctor.go:239-245` gated on `hasCodex`. Tests `healthy install table` (present) + `codex footnote omitted when codex absent`.
- [✓] AC-7 copilot detected via instructions file / prompt dirs, NOT `DetectInstalledTargets` — `targetInstalledOnDisk` (`inventory.go:255-272`) uses own dest paths + `.github/copilot-instructions.md` marker (:265-270). `DetectInstalledTargets` never referenced. Test `TestInventory_CopilotDetectedFromInstructionsFile` (real `runCopilot` install + marker-in-isolation).
- [✓] AC-8 row set = `UnionTargets(PreviouslyInstalledTargets, detected)` — `inventory.go:90`. Both directions tested: `TestInventory_UnionSurvivesMissingInstallState` (fresh clone), `TestInventory_PersistedTargetWithMissingTreeIsZero` (missing tree → 0/N).
- [✓] AC-9 empty state = single neutral `hero install --target` line, no warning/verdict change — `doctor.go:188-191`. Test `empty install state renders neutral line`.
- [✓] AC-10 shortfall marks row `!` + WARNING recommending `hero upgrade` — `kindCell` `!` at `doctor.go:265-267`; WARNING at :228-237 recommends `hero upgrade`. Test `shortfall_recommends_hero_upgrade` asserts contains `hero upgrade` AND NOT `hero install`.
- [✓] AC-11 not-installed target: no `!`, no WARNING, no upgrade nudge — `incomplete` counter only increments over rows in `info.inventory` (`doctor.go:204`); not-installed targets never become rows. Test `not_installed_target_no_warning`.
- [✓] AC-12 Verdict line unchanged — `doctorVerdict` untouched; install section carries its own WARNING. Test `verdict_unchanged_under_shortfall` (byte-identical Verdict with/without shortfall).
- [✓] AC-13 outside workspace / no graph preserves early-return, section skipped — two early returns at `doctor.go:148-149,154-155` return before `buildInventorySection` (:161). Test `section skipped with no graph or workspace`.
- [✓] AC-14 brief — not-installed collapse to one line (`doctor.go:224-226`), no per-file names. Test asserts single `not installed:` line.

## Changes
- [✓] 1. New `internal/install/inventory.go` — `TargetInventory`/`KindCount` types with `NotApplicable` (not `Expected:0`), `Inventory(projectRoot, domain)`, `targetInstallPaths`, union-based row set, `RootFile` from `nativeInstructionFile` (:104), README skip via `isContentReadme` (:210). Present, matches spec §Changes-1.
- [✓] 2. New `internal/install/inventory_test.go` — six-row `TestInventory_AllSixTargets` over `integrityTargets` (no `t.Skip`), plus all five named guard tests (CodexCommandsNotApplicable, CodexSkillsMatchInstalledDirs, CopilotDetectedFromInstructionsFile, UnionSurvivesMissingInstallState, PersistedTargetWithMissingTreeIsZero). Present.
- [✓] 3. `internal/cli/doctor.go` — `doctorInfo` extended with `inventory`/`inventoryErr` (:58-59); populated in `runDoctor` (:102-108) with non-fatal error handling; `buildInventorySection` inserted after graph, before verdict; upgrade-not-install WARNING string. Present.
- [✓] 4. `internal/cli/doctor_test.go` — subtests for healthy table, codex em-dash, footnote presence/absence, shortfall→upgrade, not-installed silence, verdict-unchanged, empty state, section-skip, introspection-error-non-fatal. Present.
- [✓] 5. `web/docs/src/cli/overview.md` — Troubleshooting `text` sample block (all five sections), codex `—` convention explained, `hero doctor` row added to Workspace Utilities table. Edits `src/` only. `TestMarkdownInvocationsResolveAgainstRootCmd` green after edit.

## Boundaries (all respected)
- No `--json` flag added to doctor (`doctor.go` has no `Flags()`/json). ✓
- No `DetectLegacyDrift` surfacing. ✓
- No content validation (counts files only). ✓
- No `isEngineSourceRepo` special-case. ✓
- No changes to `install.DetectInstalledTargets` / `targetLayouts`. ✓
- No new `hero install` behavior — read-only introspection. ✓

## Test evidence
- `go build ./...` — clean (exit 0).
- `go vet ./internal/...` — clean.
- `go test ./internal/cli/... ./internal/install/...` — both packages `ok`.
- `go test ./internal/cli/ -run TestMarkdownInvocationsResolveAgainstRootCmd -count=1` — `ok`.

## Import layering (claim #8)
`internal/install/inventory.go` imports root `hero` (embedded FS via `DomainFS`/`OverlayFS`/`CoreFS`). Direction is sane: root `hero` does NOT import `internal/install` (no non-test import found; a cycle would fail the build, which passed). Consistent with existing `internal/install/*_test.go` files that already import root `hero`. Acceptable layering, not a smell.

## Audit notes
- **Minor test gap (not a blocker):** the `actual > expected` (over-count, e.g. `30/29`) non-shortfall case is handled correctly — `kindShort`/`kindCell` use strict `<` (`doctor.go:255,265`), so an over-count renders `30/29` with no `!` and no WARNING — but no test pins this behavior. A future refactor to `!=` or `<=` would silently regress it. Worth a one-line subtest; does not hold the ship.
- No Completion Ledger was provided with the audit inputs; the self-reported "all 14 ACs + all Changes DONE" served as the claim set. Every claim was verified independently against on-disk code and tests run in this audit — no performative DONEs found.
- Diff is well-scoped: only the two new `internal/install` files, `doctor.go`/`doctor_test.go`, the docs file, and projected handoff files (`.hero/NEXT.md`, `QUEUE.md`, `SNAPSHOT.md`, `next/chet-bellows.md`). No scope drift.
