# Delivery audit — codex-command-workflow-surface-split-brain

**Audited:** `git diff -- internal/cli/upgrade.go internal/cli/upgrade_test.go internal/cli/doctor.go internal/cli/doctor_test.go internal/install/inventory.go internal/install/inventory_test.go internal/serve/mcp_tools.go internal/serve/mcp_tools_def.go internal/serve/mcp_test.go internal/serve/mcp_tool_metadata_test.go`
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria

- [✓] Same-version incomplete harnesses run integrity repair — `internal/cli/upgrade.go:127-180` resolves targets, inventories canonical artifacts, and bypasses the equal-version no-op when incomplete; `TestUpgradeSameVersionRepairsIncompleteDiscoveredCodex` and `TestUpgradeSameVersionRepairsLegacyManagedRootWithoutContentTree` assert restoration.
- [✓] Newer-written incomplete workspaces remain downgrade-safe and name the minimum binary — `internal/cli/upgrade.go:159-169` rejects before repair and names the workspace-writing version; `TestUpgradeOlderBinaryNamesRequiredVersionWithoutRepairing` asserts the guidance and that the missing artifact remains absent.
- [✓] Doctor gives a non-OK, feasible remediation for incomplete targets — `internal/cli/doctor.go:155-172` keeps inventory actionable without a graph, while `internal/cli/doctor.go:341-394` emits `NEEDS REPAIR` or `NEEDS NEWER HERO`; `TestBuildDoctorReport` covers compatible, older-binary, and no-graph cases.
- [✓] Filesystem-discovered Codex is reconciled through native command skills — `internal/cli/upgrade.go:127-147` inventories the filesystem-resolved target before the no-op; `internal/install/inventory.go:186-190` identifies commands at `.agents/skills/command-<name>/SKILL.md`; `TestUpgradeSameVersionRepairsIncompleteDiscoveredCodex` removes install-state and restores a deleted native command skill.
- [✓] Complete same-version targets remain no-ops — `internal/cli/upgrade.go:172-175` retains the no-op only when integrity is complete; `TestUpgradeSameVersionCompleteInstalledTargetRemainsNoOp` asserts the existing message and behavior.
- [✓] Saved-skill misses identify `.hero/skills` and direct commands to the native surface — `internal/serve/mcp_tools.go:1308-1329` returns the explicit namespace guidance and `internal/serve/mcp_tools_def.go:490-493` documents it; `TestToolSkillRunDistinguishesSavedSkillsFromBuiltInCommands` verifies both the miss and a real saved-skill success.

## Changes

- [✓] Evaluate target and root-instruction integrity before the equal-version no-op — implemented in `internal/cli/upgrade.go:127-180` with filesystem resolution, legacy inference, canonical inventory, and root integrity findings.
- [✓] Preserve the downgrade guard with exact compatible-version guidance — implemented in `internal/cli/upgrade.go:159-169` and `internal/cli/doctor.go:248-257`; regressions cover both CLI surfaces.
- [✓] Include completeness and repair feasibility in the doctor verdict — implemented in `internal/cli/doctor.go:155-172` and `internal/cli/doctor.go:341-394`; report tests cover non-OK outcomes with and without a graph.
- [✓] Detect exact missing Hero-owned artifacts across all seven targets — `internal/install/inventory.go:83-219` inventories canonical paths instead of relying only on counts; `TestInventory_AllSevenTargets` and `TestInventoryForTargets_ReportsUnpersistedCodexShortfall` cover fresh installs and equal-count substitution.
- [✓] Clarify the `hero_skill_run` saved-project namespace — `internal/serve/mcp_tools_def.go:490-493` and `internal/serve/mcp_tools.go:1328-1329` distinguish saved workflows from native commands; metadata and behavior tests assert the contract.

## Audit notes

- The audited diff is confined to the five named production files and their tests.
- The authoritative full test rerun passed `./internal/install/...` and `./internal/serve/...`, but `./internal/cli/...` is not currently green. Its only failures are `TestConnectProviderPickerStaysSilentWithoutATerminal/{pipe,closed}`: concurrent out-of-scope `internal/cli/connect.go` now includes the `aha` provider while `internal/cli/prompt_setup_commands_test.go:133` still expects the prior provider list.
- The focused in-scope CLI regressions pass: `go test ./internal/cli -run 'TestUpgrade(SameVersion|Older)|TestBuildDoctorReport' -count=1`. The reported seven-target inventory gate, focused install/serve regressions, build, diff check, drift check, and spec score also pass. The unrelated dirty-worktree failure does not downgrade any acceptance criterion, but it prevents claiming a clean full CLI package run.
