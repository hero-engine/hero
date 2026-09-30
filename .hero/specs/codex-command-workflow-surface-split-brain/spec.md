---
title: "Codex advertises Hero workflows that are absent from its native skill surface"
slug: codex-command-workflow-surface-split-brain
type: bug
status: completed
priority: high
severity: medium
root_cause_class: design
domain: engineering
created: 2026-09-21
tags: [codex, install, skills, commands, mcp, routing]
relates-to: [codex-install-broken, doctor-install-target-table, install-integrity-self-check]
delivery_method: manual
completed_at: 2026-09-23T00:07:21Z
---

# Codex advertises Hero workflows that are absent from its native skill surface

## Summary

### Categorization

| Attribute | Assessment |
|-----------|------------|
| **Criticality** | high — an incomplete native harness can be diagnosed correctly but remain unrepairable through the recommended normal update path |
| **Ease of Fix** | moderate — rendering is correct; version gating, health verdicts, and repair routing must be made coherent |
| **Caused by our codebase?** | Yes — Hero recommends `hero upgrade` for an incomplete target even when version gating rejects that command before it can inspect or repair the target |
| **Needs more research?** | No — the on-disk timestamps, version ledgers, CLI behavior, and current-source dry run establish the failure chain |

### Background

In a Codex session on Candy, the agent reported that the installed `/decide`
workflow was unavailable through Hero's current skill surface and fell back to
the checked-in Claude command definition. The decision work could continue, but
the message incorrectly suggested that Hero lacks `/decide` rather than that the
project's Codex install is incomplete.

### Analysis

Candy's root `AGENTS.md` advertises `/decide`, and `.claude/commands/decide.md`
exists, but `.agents/skills/command-decide/SKILL.md` does not. The current Candy
clone began on 2026-07-12. Its Hero ledgers record only a Claude project install
at v0.19.0 and a last successful workspace upgrade on 2026-08-27 to
`v0.34.1-15-g329c9b60`. The Codex-native files appeared later, in one batch at
2026-09-01 19:54:57 MDT. That batch contains all agents and canonical skills but
only three migrated `source-command-*` skills.

The September 1 batch is consistent with Codex's external-agent migration
history, but it does not prove user intent and is not the repair failure. Codex
skill import does not overwrite an existing skill directory. Therefore it could
not have deleted a pre-existing set of Hero `command-*` skills. The durable fact
is that this clone had no complete Hero-rendered Codex command surface when that
batch landed. An explicit Codex install is not present in this clone's Hero
ledger; that says nothing about an earlier checkout or another clone.

Hero's target resolution is not the culprit. It unions persisted install state
with filesystem detection, and current source detects both Claude and Codex in
Candy. A current-source Codex dry run renders all 29 `command-*` workflow skills,
including `command-decide`.

The repair failed at version gating. Candy was already stamped with the newer
`v0.34.1-15-g329c9b60` workspace version before the incomplete Codex files
appeared. The available PATH binary is `v0.32.0-2-ge09f3ee-dirty`, and the
Homebrew installation is `0.25.1`. `hero upgrade` rejects both as downgrades
before target resolution or integrity repair. A same-version binary would also
return “nothing to upgrade” before checking target completeness. Meanwhile,
`hero doctor` detects Codex at `60/86`, recommends `hero upgrade`, and still ends
with `Verdict: OK` because its verdict covers schema compatibility rather than
the incomplete install or whether the recommended repair can run.

The misleading `hero_skill_run` namespace remains a secondary defect: it reads
project-authored `.hero/skills`, not harness-native `command-*` workflow skills.

### Root Cause

Hero's health and update contracts disagree. Health detects an incomplete Codex
target and recommends `hero upgrade`, but upgrade applies workspace-version
short-circuits before target resolution and repair. In Candy, the only locally
available Hero binaries are older than the workspace stamp, so every normal
upgrade attempt is rejected before it can use the filesystem evidence that
correctly identifies Codex as installed. Even a same-version binary can skip an
incomplete target as “nothing to upgrade.” Doctor then reports `Verdict: OK`,
making an actionable harness failure look healthy.

The September 1 Codex migration explains when the partial surface appeared; it
is not the root cause of why Hero failed to converge it. The prior diagnosis was
wrong to claim that target-aware upgrade ignored an unregistered target.

### Source

The primary defect is the early version exit and downgrade rejection in
`internal/cli/upgrade.go`, which run before `resolveUpgradeTargets` and install
integrity evaluation. The contradictory diagnosis/remediation is in
`internal/cli/doctor.go`. Target resolution in `internal/cli/upgrade.go` and
`internal/install/state.go` is correct: it unions persisted targets with the
filesystem probe. Native Codex command rendering in
`internal/install/target_codex.go` is also correct.

### Fix Direction

1. Make `hero upgrade` evaluate installed-target integrity before treating an
   equal workspace version as a no-op; missing managed artifacts must be
   repairable without a version change.
2. When the running binary is older than the workspace, report the incomplete
   targets together with the exact required/newer Hero version. Do not recommend
   `hero init` as the normal repair for generated harness drift.
3. Make `hero doctor` verdicts include harness completeness and repair
   feasibility. `60/86 !` cannot end in an unqualified `Verdict: OK`.
4. Make doctor distinguish “upgrade can repair now” from “running binary is too
   old; install/use version X or newer first.”
5. Retain the secondary tool-contract cleanup: describe `hero_skill_run` as the
   `.hero/skills` saved-workflow surface, not the built-in command surface.

---

## Problem Statement

### Reproduction

1. Open `/Users/bwheeler/projects/astroville/repository/candy` in Codex.
2. Observe that `AGENTS.md` routes decision/tradeoff requests to `/decide`.
3. Observe that `.claude/commands/decide.md` exists while
   `.agents/skills/command-decide/SKILL.md` does not.
4. Run `hero doctor` in Candy. It reports Codex agents `35/35`, skills
   `60/86 !`, and says the 29 commands should be native `command-*` skills.
5. Call `hero_skill_run` with either `decide` or `command-decide`. It returns
   `Skill ... not found in .../.hero/skills`.
6. Run `hero install project . --target codex --dry-run`. The plan includes
   `.agents/skills/command-decide/SKILL.md (rendered)` and the other 28 command
   workflows.

### Expected

Codex either has every workflow advertised by its root instructions or receives
a direct incomplete-install diagnosis and repair instruction. The MCP saved-skill
tool must not imply that it is the lookup surface for built-in Hero commands.

### Actual

Codex sees `/decide` in routing, lacks `command-decide`, and receives a generic
“skill not found” result from an unrelated saved-skill store. The model reports
that Hero's current skill surface lacks the workflow and silently uses a
cross-harness fallback.

## Environment Details

- Hero repository: `/Users/bwheeler/projects/hero-engine/repository/hero`
- Observed consumer: `/Users/bwheeler/projects/astroville/repository/candy`
- Harness: Codex desktop
- Candy install inventory: Claude complete; Codex `60/86` skills, 29 command
  workflow skills absent
- Candy install state records only the Claude target, while Codex-looking
  artifacts are present and discoverable as an incomplete target
- Codex external-agent import ID:
  `93785f9f-415a-4aa9-a9fd-364e78b2af73`; provider `claude-code`; completed
  2026-09-01 19:54:57 MDT; no recorded failures
- The Codex import wrote the `.agents` and `.codex` artifacts in one batch, and
  Git commit `1e5d7d97cb576a7f23833bdb2c0914d2af52367a` later committed them as part
  of a general workspace checkpoint
- Running CLI: `v0.32.0-2-ge09f3ee-dirty`; Candy was last used by
  `v0.34.1-15-g329c9b60`; Homebrew resolves an even older `0.25.1` binary.
  Both reject `hero upgrade` as a downgrade before install reconciliation.

## Root Cause Analysis

### Confirmed sequence

1. Candy's Hero-managed files and ledgers were last successfully refreshed on
   2026-08-27. The target ledger contained Claude only.
2. On 2026-09-01 at 19:54:57 MDT, Codex wrote `.codex` and `.agents` in one
   batch. The recorded migration contains 35 agents, 57 ordinary skills, and
   only three commands. This establishes the partial surface but not that the
   user requested an import.
3. Hero's filesystem probe now recognizes that surface as Codex. A current
   source dry run resolves `claude, codex` and plans all 29 missing
   `command-*` skills, then removal of the three obsolete `source-command-*`
   directories.
4. The locally resolved `hero` binary is older than Candy's workspace stamp.
   `hero upgrade --dry-run` exits with `cannot downgrade workspace ...` before
   `resolveUpgradeTargets` runs. The Homebrew binary fails the same way.
5. `hero doctor` independently reports Codex `60/86 !`, prescribes
   `hero upgrade`, and then prints `Verdict: OK` because its final verdict only
   compares graph schema versions.

### Why repeated upgrades did not protect the surface

The successful upgrades predated the September 1 partial Codex surface. Once
that surface existed, version gating could prevent reconciliation in two ways:

- an older binary rejects the workspace as a downgrade; and
- an equal-version binary returns “nothing to upgrade” before checking whether
  generated harness files are complete.

Hero therefore treats the workspace version stamp as a proxy for install
integrity even though harness artifacts can be added, deleted, or partially
created without changing that stamp. The filesystem detector and renderer are
healthy; they are simply reached too late.

### Explicit Codex install evidence

Available shell history contains explicit
`hero install project . --target codex` commands, consistent with the user's
recollection. The surrounding directory history places one in Boxy, and the
other is tied to the `adding codex hero` commit in an archived Hero Code
checkout. Candy's own target ledger never records a completed Codex install.
That is not treated as user error: the partial Codex surface looked installed,
doctor recognized it as installed, and the prescribed convergence command
could not run.

## Code Flow (End to End)

1. `domains/engineering/routing.md` — tells any harness to route decision and
   tradeoff language to `/decide`.
2. `internal/install/agents_md.go:546` — appends the Codex-specific workflow
   explanation, whose static table at lines 557–564 lists only eight workflows.
3. Codex external-agent migration writes selected Claude artifacts to `.agents`
   and `.codex`; Candy's recorded batch contains only three commands.
4. `internal/cli/upgrade.go:108–119` — equal-version and older-binary exits run
   before target discovery or install-integrity repair.
5. `internal/cli/upgrade.go:405–415` — when reached, correctly unions persisted
   target state with filesystem detection and resolves Candy's Codex target.
6. `internal/install/target_codex.go:82` — writes ordinary skills to
   `.agents/skills`; lines 90–96 render every command as a `command-*` skill.
7. In Candy, that complete Codex render has not landed, so the model cannot load
   `.agents/skills/command-decide/SKILL.md` despite seeing `/decide` in routing.
8. `internal/cli/doctor.go:175–263` — reports the incomplete target and
   recommends upgrade.
9. `internal/cli/doctor.go:163, doctorVerdict` — appends a schema-only `OK`
   verdict without considering incomplete inventory or repair feasibility.
10. `internal/serve/mcp_tools_def.go:490` — advertises `hero_skill_run` with a
   generic saved-workflow description.
11. `internal/serve/mcp_tools.go:1308` — handles the call; line 1315 hard-codes
   `.hero/skills`, and line 1329 returns a non-error “not found” message.
12. The model interprets this saved-skill miss as evidence about the native Hero
   command surface, then reads `.claude/commands/decide.md` as a fallback.

## Key Files

### Harness installation and routing

| File | Lines | Relevance |
|------|-------|-----------|
| `internal/install/target_codex.go` | 82–119 | Canonical implementation that renders command definitions as native Codex skills |
| `internal/cli/upgrade.go` | 108–145, 405–415 | Gates on workspace version before correctly resolving filesystem-detected targets |
| `internal/install/agents_md.go` | 540–567 | Codex workflow explanation and incomplete static routing table |
| `domains/engineering/routing.md` | decision row | Advertises `/decide` through natural-language routing |
| `internal/cli/doctor.go` | 120–163, 175–263 | Detects incomplete inventory but appends a schema-only OK verdict and infeasible repair |
| `internal/version/version.go` | 192–226 | Knows the binary is older than the workspace but labels the downgrade as needing no action |

### MCP saved-skill surface

| File | Lines | Relevance |
|------|-------|-----------|
| `internal/serve/mcp_tools_def.go` | 490–500 | Ambiguous `hero_skill_run` metadata |
| `internal/serve/mcp_tools.go` | 1308–1330 | Restricts discovery to `.hero/skills` and returns the misleading miss |
| `internal/skills/skills.go` | 10–39 | Defines saved project skills parsed from `.hero/skills` |

## Secondary Defects

1. `hero_skill_run` returns a successful text response for an unknown skill
   rather than a structured miss containing the searched namespace and the
   appropriate next action. That makes tool misuse hard for a model to detect.
2. The Codex and Grok workflow tables are independently hard-coded subsets, so
   their instruction surfaces can drift from the installed command inventory.
3. Candy contains obsolete `source-command-*` skills that a full Codex install
   would prune, further obscuring whether the target is current.
4. Upgrade writes its version record through multiple paths; the cached
   `version.Info` write after `StampUpgrade` can erase the newly written
   `last_upgrade` record, weakening historical diagnosis.

## Goal

An incomplete generated harness is never treated as healthy merely because its
workspace version matches or exceeds the running binary. Hero either repairs
the surface with a compatible binary or gives one feasible, exact path to the
required binary and repair command. Codex workflow lookup then resolves native
`command-*` skills without conflating them with `.hero/skills`.

## Changes

1. `internal/cli/upgrade.go` — resolve legacy and filesystem-detected targets,
   then evaluate generated-file and root-instruction integrity before the
   equal-version no-op.
2. `internal/cli/upgrade.go` and `internal/cli/doctor.go` — preserve downgrade
   safety while naming the minimum compatible Hero binary and a feasible repair
   sequence.
3. `internal/cli/doctor.go` — fold incomplete target inventory and repair
   feasibility into the final verdict, including workspaces without a graph.
4. `internal/install/inventory.go` and tests — identify missing canonical
   artifacts by path across all seven targets so stale or user extras cannot
   mask a missing workflow by keeping aggregate counts equal.
5. `internal/serve/mcp_tools.go`, `internal/serve/mcp_tools_def.go`, and tests —
   identify `hero_skill_run` as the `.hero/skills` saved-project-skill surface
   and route built-in commands to harness-native `command-*` workflows.

## Boundaries

- Do not merge saved project skills and installed command workflows into one
  filesystem namespace.
- Do not remove the cross-harness fallback until all supported upgrade paths
  reliably materialize native workflow skills; make it an explicit degraded
  path, not the normal resolution route.
- Do not change `/decide`'s decision methodology; its workflow content is not
  the defect.
- Do not special-case only Candy. The fix belongs in Hero's generated surfaces
  and applies to every consumer repository.

## Risks

- Generating routing from command inventory can accidentally expose commands a
  domain pack intentionally omits; tests must use the active pack's resolved
  inventory.
- Changing MCP miss semantics can affect clients that treat the current text as
  a successful preview; retain protocol compatibility while making the result
  unmistakable.
- An older binary must not rewrite newer generated content; the fix must improve
  diagnosis and binary remediation without weakening downgrade safety.
- Same-version repair is a normal explicit `hero upgrade` mutation and should
  not require a second authorization prompt.

## Acceptance Criteria

- **AC-1:** WHEN an installed harness is incomplete and the workspace version equals the running Hero binary THE SYSTEM SHALL run install-integrity repair instead of returning “nothing to upgrade.”
- **AC-2:** WHEN an incomplete workspace was last written by a newer Hero binary THE SYSTEM SHALL preserve downgrade safety and report the minimum compatible binary version before prescribing a repair command.
- **AC-3:** WHEN `hero doctor` detects any incomplete installed target THE SYSTEM SHALL return a non-OK overall verdict and SHALL NOT recommend a command the running binary will reject.
- **AC-4:** WHEN a Codex target is discoverable from `.codex` or `.agents` but absent from persisted install-state THE SYSTEM SHALL include Codex in compatible upgrade reconciliation and render every active command as `command-<name>/SKILL.md`.
- **AC-5:** WHEN every installed target is complete and the workspace version equals the running binary THE SYSTEM SHALL preserve the existing no-op upgrade behavior.
- **AC-6:** WHEN `hero_skill_run` cannot resolve a slug from `.hero/skills` THE SYSTEM SHALL identify that namespace explicitly and direct built-in command workflows to the harness-native `command-*` surface.

## Validation

1. Install a clean Codex fixture, delete `command-decide`, retain the same
   workspace/binary version, and verify plain `hero upgrade` restores it rather
   than returning “nothing to upgrade.”
2. Run an older binary against the incomplete fixture and verify it refuses to
   rewrite content while doctor returns a non-OK verdict naming the minimum
   compatible Hero version.
3. Create a filesystem-detected Codex target absent from install-state and
   verify compatible upgrade discovers and fully renders it.
4. Verify a complete same-version fixture remains a true no-op.
5. Call `hero_skill_run` for a real `.hero/skills` slug and verify it succeeds;
   call it for `decide` and verify the response explicitly says command
   workflows use the native harness surface.
6. Run `go test ./internal/install/... ./internal/serve/... ./internal/cli/...`.
7. Run the seven-target harness propagation tests required by the
   `harness-changes-cover-all-targets` tripwire.

## Kickoff

Repairs incomplete generated harness files even when the workspace version is
already current, without letting an older binary rewrite newer content.

**Status:** completed — implementation, regression coverage, the independent
delivery audit, and the verification gate all passed.

**Pick up at:** no delivery work remains; publish the repaired Hero binary and
then update affected projects so their incomplete Codex surfaces converge.

→ `.hero/specs/codex-command-workflow-surface-split-brain/spec.md`

**Files:** `internal/cli/upgrade.go:127`, `internal/install/inventory.go:83`, `internal/cli/doctor.go:149`, `internal/serve/mcp_tools.go:1308`

## Completion Ledger

Implementation is in Go. Loaded `agent-reliability`,
`implementation-principles`, `go-stack`, `context-injection`, and
`completion-ledger`. The affected install/serve package trees, focused CLI
regressions, and explicit seven-target inventory gate pass. The full CLI
package currently has one unrelated dirty-worktree failure in
`TestConnectProviderPickerStaysSilentWithoutATerminal`: concurrent `connect`
work advertises `aha`, while its existing expected usage string does not.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Same-version incomplete harness runs integrity repair | DONE | `internal/cli/upgrade.go:127` evaluates targets and integrity before the no-op; `TestUpgradeSameVersionRepairsIncompleteDiscoveredCodex` and `TestUpgradeSameVersionRepairsLegacyManagedRootWithoutContentTree` restore deleted workflows. |
| 2 | Newer-written incomplete workspace preserves downgrade safety and names minimum binary | DONE | `internal/cli/upgrade.go:159` rejects before writes and names the workspace version; `TestUpgradeOlderBinaryNamesRequiredVersionWithoutRepairing` verifies no repair occurs. |
| 3 | Doctor returns non-OK and only feasible remediation for incomplete targets | DONE | `internal/cli/doctor.go:149` keeps inventory visible without a graph and `doctorVerdict` emits `NEEDS REPAIR` or `NEEDS NEWER HERO`; `TestBuildDoctorReport` covers both paths. |
| 4 | Filesystem-discovered Codex is reconciled and commands render as native skills | DONE | `internal/cli/upgrade.go:127` resolves filesystem targets before integrity evaluation; `TestUpgradeSameVersionRepairsIncompleteDiscoveredCodex` removes install-state and proves `command-design/SKILL.md` is restored. |
| 5 | Complete same-version target remains a no-op | DONE | `internal/cli/upgrade.go:172` retains the no-op only for complete targets; `TestUpgradeSameVersionCompleteInstalledTargetRemainsNoOp` pins the behavior. |
| 6 | Saved-skill miss names `.hero/skills` and routes commands to native surface | DONE | `internal/serve/mcp_tools.go:1308` returns an explicit namespace miss; `TestToolSkillRunDistinguishesSavedSkillsFromBuiltInCommands` verifies the miss and a real saved-skill success. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Evaluate target integrity before equal-version no-op | DONE | `internal/cli/upgrade.go:127-181`; includes legacy managed-root inference and root instruction integrity. |
| 2 | Preserve downgrade guard with exact compatible-version guidance | DONE | `internal/cli/upgrade.go:159-170` and `internal/cli/doctor.go:248-259`. |
| 3 | Make doctor verdict include completeness and feasibility | DONE | `internal/cli/doctor.go:149-173` and `internal/cli/doctor.go:341-395`. |
| 4 | Repair exact missing Hero-owned artifacts at same version | DONE | `internal/install/inventory.go:83-220` tracks canonical missing paths; all-seven-target and equal-count substitution regressions pass. |
| 5 | Clarify `hero_skill_run` saved-project namespace | DONE | `internal/serve/mcp_tools_def.go:490` and `internal/serve/mcp_tools.go:1308`; metadata and behavior regressions pass. |

### Exercise-the-feature check

- [x] User-visible behavior was exercised end-to-end through CLI/MCP entry points: focused `runCmd("upgrade")` fixtures restored missing Codex and legacy Claude artifacts, refused an older-binary repair, preserved the complete same-version no-op, and direct `toolSkillRun` calls distinguished saved skills from built-in commands.

### Excellence Bar self-check

- [x] Yes — repair is target-generic, downgrade-safe, exact-artifact aware, preserves unrelated user skills, covers all seven harness targets, and has package-wide regression evidence.

## Notes

The observed agent's fallback was operationally reasonable given Candy's disk
state. The bad part was the diagnosis wording: the workflow was not absent from
Hero. Candy's partial surface arrived after its last successful Hero refresh,
and the available binaries could not pass the version gate to repair it. The
generic MCP tool was also the wrong namespace to test.

## Recap

Hero now treats workspace version and generated harness integrity as separate
gates. Compatible same-version upgrades repair exact missing artifacts across
filesystem-detected and legacy-inferred targets; older binaries still refuse to
rewrite newer workspaces and name the required version first. Doctor reports
incomplete targets as non-OK with an executable remediation, and
`hero_skill_run` explicitly identifies `.hero/skills` as distinct from native
`command-*` workflows. All in-scope regressions and the seven-target propagation
gate pass; the unrelated dirty `connect` expectation remains outside this
delivery.

## Investigation History

### Round 1 — Initial diagnosis (superseded)
- **Date**: 2026-09-22T21:03:00Z
- **Agent**: debug-investigator
- **Root cause**: An external Codex import allegedly created a partial, unregistered Codex target that normal Hero upgrades then ignored.
- **Confidence**: Low — rejected by the repository owner because Codex was explicitly installed through Hero and repeatedly upgraded normally.
- **Key evidence**: Candy had a Claude-only `.hero/install-state.json`, an incomplete `.agents/skills` tree, and a Codex external-agent import record listing only three command artifacts. Those facts established the broken state, but did not establish that the import created it or explain why explicit Hero install and later upgrades failed to preserve Codex.
- **Superseded root cause**: The prior analysis attributed the state to a cross-owner migration gap: Codex imported selected Claude artifacts into Codex-native locations without registering the Codex target, after which Hero's target-aware updater followed the persisted Claude-only registry and did not repair the partial Codex tree. It also identified ambiguity between harness-native `command-*` workflow skills and project-authored `.hero/skills` used by `hero_skill_run`.
- **Superseded fix plan**: Clarify the `hero_skill_run` namespace; generate complete Codex workflow routing; detect imported/unregistered targets; surface missing `command-*` skills in ordinary health checks; and add an external-import regression fixture.

### Round 2 — Challenged (reject)
- **Date**: 2026-09-22T21:03:00Z
- **Challenged by**: engineer
- **Feedback**: "don't make sense - never ran an import - installed codex specifically - did multiple upgrades on hero upgrades - no way codex should have ever been missing anything. so - figure it out"
- **Revised root cause**: Hero detects the incomplete Codex target but upgrade checks workspace-version equality/downgrade before target integrity. Candy's available binaries are older than its workspace stamp, so the recommended repair cannot run; an equal-version binary would also skip integrity. Doctor compounds this by recommending upgrade and returning a schema-only `Verdict: OK`.
- **What changed**: The import record is retained only as evidence for when the partial files appeared. Filesystem target detection is proven healthy, and the claim that Hero ignored an unregistered target is rejected. The explicit install history is reconciled across checkouts without treating the user's recollection as false.
- **Confidence**: High — reproduced against Candy with both installed binaries, confirmed by source ordering, and counter-tested with a current-source dry run that detects both targets and plans all 29 missing workflows.
