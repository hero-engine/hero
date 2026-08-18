---
title: "Grok Build Harness Target — Native Agents, Workflow Skills, MCP, and Lifecycle Parity"
slug: grok-build-harness-target
type: feature
status: completed
domain: engineering
priority: P1
size: large
tags: [install, grok, harness, mcp, agents, skills, upgrade]
created: 2026-08-17
relations:
  - target: install-upgrade-contract-coverage
    kind: parent
  - target: harness-native-install-target-aware-upgrade
    kind: related
  - target: install-contract-registry-foundation
    kind: related
  - target: harness-install-paths-match-loaders
    kind: related
  - target: doctor-install-target-table
    kind: related
  - target: satellite-harness-coverage
    kind: related
delivery_method: manual
completed_at: 2026-08-18T03:44:31Z
---

# Grok Build Harness Target — Native Agents, Workflow Skills, MCP, and Lifecycle Parity

## Goal

Make `grok` a first-class Hero install target with the same lifecycle guarantees
as Hero's existing harnesses. `hero install project . --target grok` and
`hero install global --target grok` must put agents, skills, Hero workflow
commands, root instructions, and MCP configuration where Grok Build natively
loads them. Upgrade, uninstall, auto-sync, workspace satellites, diagnostics,
contracts, dry-run, and pruning must all understand the target.

Grok's Claude and AGENTS compatibility loaders are useful migration surfaces,
but they are not the implementation. A Grok target must be represented as
`grok` in install state and must write `.grok` artifacts; it must not masquerade
as `claude` or fall back to Generic's loaderless `.ai` layout.

## Kickoff

Grok is now Hero's seventh native harness target, covering project/global
install, workflow skills, MCP, state, upgrade, doctor, satellites, and safe
uninstall. **Status:** completed; 16/16 criteria passed, the cold audit returned
SHIP, and Hero verification archived the spec. **Pick up at:** build the current
Hero binary and run a fresh `hero install project . --target grok` for the
user's hands-on Grok session. **Files:** `internal/install/target_grok.go`,
`internal/install/mcp.go`, `internal/cli/uninstall.go`, and Grok tests. **Skip:**
compatibility `.claude`/`.ai` paths and `.grok/commands`; this target uses native
agents and commands-as-skills.

## Context and Evidence

### Current Hero behavior

- `internal/install/install.go` defines six targets and rejects `grok` as
  unknown.
- `internal/cli/install.go` advertises the same six values in help, validation,
  and the interactive picker.
- `generic` writes `.ai/{agents,commands,skills}`. Its own implementation notes
  that `.ai` has no consuming loader, and `RegisterMCP` does not register MCP
  for Generic. Generic is therefore not functional Grok support.
- Installing the Claude target happens to create content Grok can load through
  compatibility, but it records and manages the install as Claude and may add
  Claude-only hooks/settings. That breaks honest detection, upgrade, uninstall,
  diagnostics, and native configuration.
- Target awareness is intentionally distributed across the target dispatcher,
  CLI surfaces, detection/state, native instruction mapping, inventory,
  contracts, auto-sync, prune/uninstall, satellites, and all-target tests. Grok
  must join each applicable surface.

### Grok Build's native contract

The current xAI documentation and the installed Grok Build 1.0.4 guide establish:

- project rules: root or nested `AGENTS.md`, with global rules under `~/.grok/`;
- project skills: `.grok/skills/<name>/SKILL.md`; global skills:
  `~/.grok/skills/<name>/SKILL.md`;
- user-invocable skills appear as slash commands;
- project agents: `.grok/agents/*.md`; global agents: `~/.grok/agents/*.md`;
- project MCP: `.grok/config.toml`; global MCP: `~/.grok/config.toml`, using
  `[mcp_servers.<name>]` TOML tables;
- `grok inspect --json` reports discovered instructions, agents, skills, and MCP
  origins;
- Grok also loads Claude, Cursor, `.mcp.json`, and legacy flat command layouts
  for compatibility.

References:

- https://docs.x.ai/build/features/project-rules
- https://docs.x.ai/build/features/skills-plugins-marketplaces
- https://docs.x.ai/build/features/subagents
- https://docs.x.ai/build/features/mcp-servers
- https://docs.x.ai/build/cli/headless-scripting

## Design Decisions

### 1. Grok is a native target, not an alias

Add `TargetGrok = "grok"` and a dedicated `runGrok`. The target owns `.grok`
paths, `AGENTS.md`, `.grok/config.toml`, state under the `grok` key, and its own
contract/inventory row. Do not dispatch through `runClaude`, `runCodex`, or
`runGeneric`.

This keeps vendor compatibility as a safety net without coupling Hero's install
state to a different harness. A Grok-only install must create neither `.claude`
nor `.ai` artifacts.

### 2. Commands use the shared commands-as-skills model

Grok still recognizes flat `.grok/commands/*.md` as legacy command markdown,
but its current extension model is user-invocable skills. Hero will therefore
install workflows as:

```text
.grok/skills/command-design/SKILL.md
.grok/skills/command-deliver/SKILL.md
.grok/skills/command-diagnose/SKILL.md
...
```

This matches the Codex target's model and gives Hero one durable strategy for
harnesses without a preferred standalone commands primitive. Generalize the
Codex renderer into a harness-neutral command-as-skill renderer. It may accept
a harness label for the execution preamble, but source selection, directory
naming, frontmatter, and pruning ownership stay shared. Do not copy the current
Codex-specific sentence into Grok output.

Grok receives no separate `.grok/commands` tree. In inventory and contracts,
Commands is `NotApplicable`; Skills counts canonical skills plus rendered
commands, like Codex.

### 3. Render to native Grok files

For project mode:

```text
AGENTS.md
.grok/agents/<name>.md
.grok/skills/<skill>/SKILL.md
.grok/skills/command-<command>/SKILL.md
.grok/config.toml
```

For global mode:

```text
~/.grok/AGENTS.md
~/.grok/agents/<name>.md
~/.grok/skills/<skill>/SKILL.md
~/.grok/skills/command-<command>/SKILL.md
~/.grok/config.toml
```

Agents remain Markdown with YAML frontmatter. Preserve canonical fields that
Grok accepts and add no speculative model, permission, or tool restrictions.
Skills retain the canonical `SKILL.md` contract. Workflow skills have
`name: command-<name>`, a useful description, and the execution preamble used
by the shared commands-as-skills renderer.

`installNativeInstructionFile` maps Grok to `AGENTS.md`. Project mode uses the
root managed block; global mode writes the same managed block to
`~/.grok/AGENTS.md`. User content outside Hero's markers remains byte-identical.

### 4. Register MCP in Grok's TOML without owning the file

`RegisterMCP(TargetGrok, opts)` upserts only this block:

```toml
# hero:managed
[mcp_servers.hero]
command = "hero"
args = ["mcp"]
# end:hero:managed
```

Workspace/satellite installs append `"--project-root", "<root>"` to `args`,
matching other targets. Use the portable bare `hero` command. Project mode
writes `<target>/.grok/config.toml`; global mode writes
`~/.grok/config.toml`.

Reuse a harness-neutral TOML managed-block helper extracted from Codex where
possible. The helper must preserve every byte outside Hero's markers, replace
an existing managed block idempotently, and remove or replace an old unmanaged
`[mcp_servers.hero]` table so the result is valid TOML with exactly one Hero
server. A malformed existing config is an error, not permission to erase user
settings. Never copy secrets into config or place credentials in argv.

### 5. Lifecycle parity is part of target support

Add Grok to every applicable canonical registry and switch:

- CLI help, validation, prompt order, JSON output, and unknown-target messages;
- install dispatch and project/global path resolution;
- persisted target state, filesystem detection, stable union order, auto-sync,
  upgrade resolution/narrowing, and version reporting;
- uninstall paths, trusted manifest cleanup, stale-file and stale-skill pruning,
  dry-run, `--force`, and `--prune-orphaned-instruction-files` behavior;
- doctor inventory, expected/actual counts, root file, health status, and repair;
- install contract registry and validation;
- domain overlays and canonical content enumeration;
- workspace `--workspace` MCP routing and satellite layout/markers;
- install guidance and every explicit all-target matrix.

Grok's satellite layout is `.grok` with `AGENTS.md` as the scoped marker. Its
`agents` and `skills` directories are satellite-able; there is no commands
directory. Satellite code must not require all targets to expose the same
subdirectory set: either make the layout declare its linkable directories or
skip absent `commands` cleanly while preserving existing target behavior.

### 6. Preserve the other targets and update the tripwire

The high-priority `harness-changes-cover-all-targets` tripwire remains binding.
Update generated guidance from “six targets” to “seven targets” and include
`grok` in the non-Claude `AGENTS.md` set. All cross-target propagation,
routing, Attention guidance, integrity, overlay, smoke, contract, inventory,
cleanup, and satellite tests must add Grok where the behavior applies.

The existing six targets must remain byte-compatible except for a necessary
rename/generalization of the command-as-skill helper. In particular, Codex's
rendered skill bytes must not change accidentally when the helper is shared.

## Changes

1. **Add the native target and renderer** — implemented in
   `internal/install/install.go`, `target_grok.go`, `render.go`,
   `target_codex.go`, and `agents_md.go`: Grok uses native Markdown agents,
   nested canonical/workflow skills, and the shared commands-as-skills model.
2. **Add native MCP merging** — implemented in `internal/install/mcp.go`, with
   shared byte-preserving TOML management and Grok project/global registration;
   covered by `mcp_test.go` and `grok_test.go` for malformed, duplicate,
   idempotent, user-content, workspace-root, and dry-run behavior.
3. **Carry Grok through the lifecycle** — implemented across
   `internal/cli/install.go`, `internal/cli/upgrade.go`,
   `internal/cli/uninstall.go`, `internal/cli/doctor.go`,
   `internal/install/state.go`, `internal/install/auto_sync.go`,
   `internal/install/inventory.go`, `internal/install/contracts.go`,
   `internal/install/satellite.go`, and `internal/install/satellite_detect.go`.
4. **Expand contract and regression coverage** — added `grok_test.go` and
   `grok_uninstall_test.go`; expanded install/CLI target, content, integrity,
   inventory, overlay, prune, routing, Attention, satellite, smoke, prompt,
   doctor, and upgrade test matrices.
5. **Update generated guidance and docs** — updated `README.md`,
   `GETTING-STARTED.md`, `domains/engineering/AGENTS.md`, CLI prompt baselines,
   and `.hero/knowledge/tripwires/harness-changes-cover-all-targets/spec.md`.
6. **Qualify the real loader** — built the current Hero CLI and exercised an
   isolated Grok-only workspace with Grok Build 1.0.4, two installs,
   `grok inspect --json`, and uninstall with planted user fixtures.

## Acceptance Criteria

- WHEN `hero install project <repo> --target grok` runs in a clean Hero workspace THE SYSTEM SHALL create the Grok-native project artifacts `.grok/agents/*.md`, `.grok/skills/*/SKILL.md`, root `AGENTS.md`, and `.grok/config.toml`.
- WHEN a canonical Hero command is installed for Grok THE SYSTEM SHALL render it as `.grok/skills/command-<name>/SKILL.md` with user-invocable skill frontmatter and executable workflow guidance, and SHALL NOT require a `.grok/commands` directory.
- WHEN the shared command-as-skill renderer serves Codex and Grok THE SYSTEM SHALL preserve Codex's existing output contract while producing a Grok-appropriate preamble without Codex-specific wording.
- WHEN `hero install global --target grok` runs THE SYSTEM SHALL install agents, canonical skills, command workflow skills, managed instructions, and MCP configuration under `~/.grok` without writing project-local artifacts.
- WHEN Hero registers MCP for a Grok project or user THE SYSTEM SHALL write exactly one valid `[mcp_servers.hero]` entry using the portable `hero mcp` command and SHALL preserve all non-Hero TOML bytes and settings.
- WHEN Grok MCP registration includes a workspace project root THE SYSTEM SHALL include `--project-root <root>` in the managed `args` and SHALL leave sibling MCP servers and unrelated Grok configuration unchanged.
- IF an existing `.grok/config.toml` is malformed THEN THE SYSTEM SHALL return a useful error and SHALL NOT replace the file with a fresh configuration.
- WHEN Grok installation succeeds THE SYSTEM SHALL record `grok` in `.hero/install-state.json`, detect it from native on-disk artifacts, show it in `hero doctor` inventory with commands not applicable and commands rolled into skills, and include it in stable auto-sync/upgrade resolution.
- WHEN `hero upgrade` or target auto-sync runs for an installed Grok target THE SYSTEM SHALL refresh only Hero-owned Grok artifacts, prune only files proven by the prior manifest, and preserve user-authored agents, skills, config, and instruction content.
- WHEN `hero uninstall --target grok` runs THE SYSTEM SHALL remove only manifest-owned Grok artifacts and target state, preserve user-owned `.grok` content and shared `AGENTS.md` user content, and leave every other installed harness operational.
- WHEN a Grok target is installed for a `--workspace` or satellite scope THE SYSTEM SHALL route MCP to the workspace root and materialize Grok's declared satellite directories plus an `AGENTS.md` marker without assuming a physical commands directory.
- WHEN `--dry-run`, `--force`, domain overlays, repair, or orphan pruning is used with `--target grok` THE SYSTEM SHALL honor the same safety and reporting contracts as the existing targets.
- WHEN `--target grok` runs THE SYSTEM SHALL NOT create `.claude`, `.ai`, or other harness-owned content as a substitute for native Grok output.
- WHEN harness-wide guidance or install content changes THE SYSTEM SHALL cover all seven targets (`opencode`, `cursor`, `claude`, `copilot`, `codex`, `generic`, `grok`) in applicable propagation and contract tests, with no regression in the existing six targets.
- WHERE an installed Grok Build binary is available WHEN a fresh Grok-only fixture is inspected with `grok inspect --json` THE SYSTEM SHALL report the Hero root instruction, agent, canonical skill, command workflow skill, and Hero MCP server from their native Grok origins without requiring a model request.
- THE SYSTEM SHALL pass focused `internal/install` and `internal/cli` tests, `go test ./... -count=1`, `git diff --check`, Hero drift validation, and the completion-ledger/audit/verify gates.

## Verification Strategy

### Automated

- Extend table-driven target registries so a missing Grok lifecycle cell fails
  loudly.
- Add `runGrok` project/global fixtures that assert exact paths, valid
  frontmatter, command-as-skill names, root instruction mapping, state, and no
  `.claude`/`.ai` writes.
- Add TOML golden tests for empty, existing-user-content, existing-managed,
  unmanaged duplicate Hero table, malformed input, workspace root, dry-run, and
  second-run idempotency cases.
- Add inventory/contract tests that model Grok like Codex for the Commands
  column while counting `.grok/skills` as canonical skills plus commands.
- Add upgrade, auto-sync, uninstall, prune, overlay, satellite, repair, CLI help,
  and JSON contract tests.
- Run focused packages first, then `go test ./... -count=1` and
  `git diff --check`.

### Local qualification

Use the installed Grok Build binary without sending a model prompt:

1. install Hero into an isolated temporary Hero workspace with `--target grok`;
2. run `grok inspect --json` with that workspace as cwd and auto-update disabled
   where supported;
3. assert the native origins for `AGENTS.md`, one Hero agent, one canonical
   skill, one `command-*` skill, and `mcp_servers.hero`;
4. re-run install to prove idempotency, then uninstall and inspect preserved
   user fixtures.

If the installed Grok version lacks a stable JSON field needed for an automated
assertion, preserve the filesystem tests as the release gate and record the
human-readable `grok inspect` evidence in the Completion Ledger. Do not weaken
the native-path requirements.

## Risks and Mitigations

- **Target-list drift:** Grok could be added to install but omitted from upgrade,
  doctor, or uninstall. Mitigation: update canonical registries and meta-tests,
  then search for all existing six-target matrices before completion.
- **Shared renderer regression:** generalizing the Codex renderer could change
  its output or prune namespace. Mitigation: retain Codex golden coverage and
  make harness wording an explicit parameter.
- **TOML data loss:** decode/re-encode would erase comments and formatting.
  Mitigation: marker-based byte-preserving replacement with malformed-input
  refusal and duplicate-table tests.
- **User-file deletion:** `.grok/skills` and `.grok/agents` are user-extensible.
  Mitigation: reuse manifest/provenance pruning; never wholesale-remove those
  directories.
- **Legacy command ambiguity:** Grok loads `.grok/commands`, but current product
  direction favors skills. Mitigation: use commands-as-skills, qualify with
  `grok inspect`, and avoid dual-installing the same workflow.
- **Satellite shape mismatch:** current satellites assume `agents`, `commands`,
  and `skills` for every target. Mitigation: declare linkable dirs per layout or
  make absent dirs a first-class no-op, covered across all targets.

## Boundaries

- Do not install, upgrade, authenticate, or configure the Grok Build binary.
- Do not add Grok as a model provider or execution backend for Hero Runner or
  peer calls.
- Do not install Grok plugins, hooks, personas, roles, permissions, or
  marketplaces. Those require separate product decisions and trust semantics.
- Do not redesign canonical Hero agent, command, or skill content.
- Do not remove Grok's ability to consume Claude, Cursor, `.agents`, or legacy
  command compatibility files; Hero simply does not depend on those fallbacks.
- Do not refactor unrelated target code except where a small shared helper is
  necessary to keep Codex and Grok on one commands-as-skills/TOML model.

## Completion Ledger

Delivered as a Go CLI/install change using the `go-stack`,
`implementation-principles`, `testing-and-validation`, `agent-reliability`, and
`completion-ledger` contracts. Validation included focused install/CLI tests,
the full repository suite, diff hygiene, Hero drift, and a real Grok Build 1.0.4
native-loader exercise in an isolated Hero workspace.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Project install creates native Grok artifacts | DONE | `internal/install/target_grok.go` and `TestGrokProjectInstallUsesNativeLayout` verify `.grok/agents`, canonical/workflow skills, `AGENTS.md`, and `.grok/config.toml`. |
| 2 | Commands install as user-invocable skills without `.grok/commands` | DONE | `commandAsSkillRenderer("Grok Build")` writes `command-*/SKILL.md`; project, satellite, and real-filesystem checks prove `.grok/commands` is absent. |
| 3 | Shared renderer preserves Codex bytes and uses Grok wording | DONE | `TestCommandAsSkillRendererPreservesCodexBytes` asserts Codex's exact byte contract; the Grok project test asserts the Grok preamble and rejects Codex wording. |
| 4 | Global install uses only `~/.grok` | DONE | `TestGrokGlobalInstallUsesUserNativeLayout` covers global agents, skills, workflows, instructions, MCP, and absence of project-local output. |
| 5 | MCP is one valid portable entry and preserves non-Hero bytes | DONE | `upsertManagedTOMLMCPConfig` validates TOML and owns only its marked block; preserve/convergence and duplicate-table tests pass. |
| 6 | Workspace-root MCP args preserve sibling configuration | DONE | `TestManagedTOMLMCPUpsertPreservesUserBytesAndConverges` asserts `--project-root /workspace/root`, exact outer bytes, valid TOML, and second-run identity. |
| 7 | Malformed Grok TOML is refused without mutation | DONE | `TestGrokMalformedConfigIsRefusedWithoutMutation` asserts the useful parse error and byte-identical file after failure. |
| 8 | State, detection, doctor, auto-sync, and upgrade include Grok | DONE | Target/state/inventory registries plus doctor and upgrade tests cover the seventh target and commands-not-applicable skills rollup. |
| 9 | Upgrade/auto-sync refresh owned files and preserve user files | DONE | `TestAutoSyncRefreshesGrokAndPreservesForeignContent`, prune matrices, and upgrade-resolution tests cover refresh and provenance-safe preservation. |
| 10 | Uninstall removes only owned artifacts and Grok state | DONE | `internal/cli/grok_uninstall_test.go` covers user content, managed TOML, shared instructions, other-target preservation, and state; the isolated real uninstall removed 123 Hero files and preserved exactly two planted user fixtures. |
| 11 | Workspace/satellite uses declared Grok dirs and root routing | DONE | `TargetLayout.LinkableDirs` restricts Grok to agents/skills; `TestGrokSatelliteLinksOnlyDeclaredDirectories` verifies links, marker, and no commands dir; MCP tests cover root args. |
| 12 | Dry-run, force, overlays, repair, and orphan safety match contracts | DONE | Grok is included in dry-run, overlay, integrity/repair, prune, contract, and all-target matrices; focused install and CLI packages pass. |
| 13 | Grok target creates no substitute harness content | DONE | Project test and real isolated exercise assert `.claude`, `.ai`, and `.grok/commands` are absent. |
| 14 | Applicable propagation/contracts cover all seven targets | DONE | All-target matrices and the active harness tripwire now enumerate Grok; the existing six targets pass the full repository suite. |
| 15 | Installed Grok loader reports native Hero origins | DONE | Grok Build 1.0.4 `inspect --json` found project `Agents.md`, Hero `.grok/agents`, `spec-format`, user-invocable `command-design`, and `hero` from `configToml` at a `.grok/config.toml` origin without a model request. |
| 16 | Tests, diff, drift, ledger, audit, and verify gates pass | DONE | `go test ./internal/install ./internal/cli -count=1`, `go test ./... -count=1`, and `git diff --check` pass; drift and ledger are clean, with the independent audit and verify closing pass assigned to the root delivery lead. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Native Grok target and shared workflow renderer | DONE | Added `internal/install/target_grok.go`; updated target dispatch, native guidance, renderer, and Codex shared helper without changing Codex bytes. |
| 2 | Byte-preserving native TOML MCP management | DONE | Updated `internal/install/mcp.go` and tests for project/global Grok paths, exact preservation, duplicates, malformed input, root args, removal, and dry-run. |
| 3 | Full install lifecycle parity | DONE | Updated CLI install/upgrade/uninstall/doctor and install state, detection, auto-sync, inventory, contracts, pruning, and satellites. |
| 4 | Contract and regression coverage | DONE | Added focused Grok install/uninstall suites and expanded applicable all-target test matrices and prompt baselines. |
| 5 | Guidance, docs, and tripwire | DONE | Updated root docs, generated engineering guidance, CLI text, and the seven-target harness tripwire source. |
| 6 | Real Grok loader qualification | DONE | Current Hero binary installed twice into an isolated `.hero` workspace; Grok inspect assertions and provenance-safe uninstall assertions passed. |

### Exercise-the-feature check

- [x] User-visible behavior was exercised end-to-end: built `./cmd/hero`, installed Grok into an isolated Hero workspace twice (122 files, then a no-op), ran Grok Build 1.0.4 `inspect --json` and observed native instruction/agent/canonical-skill/workflow-skill/MCP origins, then uninstalled and observed 123 Hero files removed while exactly two planted user files remained.

### Excellence Bar self-check

- [x] Yes — the target uses Grok's current native surfaces, shares established lifecycle machinery, preserves user-owned bytes/files, has exact regression coverage for Codex, and passed both hermetic and real-loader validation.
