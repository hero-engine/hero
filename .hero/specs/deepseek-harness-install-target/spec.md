---
title: DeepSeek Harness Install Target
slug: deepseek-harness-install-target
type: feature
status: completed
domain: engineering
size: medium
priority: medium
created: 2026-09-28
tags: [install, deepseek, harness, mcp]
relations:
  - target: grok-build-harness-target
    kind: related
  - target: harness-native-install-target-aware-upgrade
    kind: related
delivery_method: manual
completed_at: 2026-09-30T16:58:16Z
---

# DeepSeek Harness Install Target

## Context

The user wants Hero available in the DeepSeek harness, whose source is checked out at `/Users/bwheeler/projects/public/repository/deepseek-harness`. This design inspected that checkout at [commit `477b4f420553e8a52c2fbccc464d7561b239c443`](https://github.com/deepseek-ai/deepseek-harness/tree/477b4f420553e8a52c2fbccc464d7561b239c443) on 2026-09-28. Its executable is `dsh`; this is a harness integration, not a DeepSeek model/provider integration.

Hero currently supports seven install targets. Its completed Grok integration provides the lifecycle pattern, but DeepSeek has a different loader contract: skills and root instructions are discovered from files, whereas MCP is composed through explicit Cordis configuration patches. Agent presets are plugin declarations, not Markdown agent files.

Mission fit: a DeepSeek session should discover the same Hero knowledge and executable workflows as other harnesses, without making the user recreate context or pretending unsupported integration surfaces work. The active `harness-changes-cover-all-targets` tripwire is satisfied by adding the new native adapter and checking propagation/regressions for all seven existing targets.

## Goal

Make `hero install --target deepseek` produce instructions and skills that DeepSeek actually loads, plus an explicitly activated Hero MCP overlay. Include project lifecycle support, global installation, accurate diagnostics, and satellite handling. Preserve user files and existing harness behavior. Native role presets and automatic patch activation are not required for this target.

## Kickoff

Add DeepSeek (`dsh`) as a Hero install target, using native skills and an explicit MCP patch.

**Status:** completed — native compatibility, regression checks, independent cold audit, and `hero spec verify` passed on 2026-09-30.

**Pick up at:** use the local build to install DeepSeek support in the intended application workspace, then activate its generated Cordis overlay with `dsh --patch`. The machine's default Hero binary was not replaced by this delivery.

→ Review `delivery-audit.md` and `delivery-evidence.md` for the completed delivery; do not redeliver this closed spec.

**Files:** `internal/install/install.go`, `internal/install/target_grok.go`, `internal/install/render.go`, `internal/install/inventory.go`, `internal/cli/uninstall.go`
**Skip:** `.dsh/agents`, project `.mcp.json`, automatic global profile edits.

## Problem

There is no `deepseek` target. Installing Claude or Codex output happens to expose some instructions or skills to DeepSeek, but mixes ownership with those harnesses and does not provide native MCP activation. Copying the Grok adapter and renaming directories would write agent files DeepSeek never discovers and configure MCP in a format it does not read.

## Source evidence

Paths below are relative to the inspected harness checkout; implementation must recheck these contracts if its revision changes.

| Surface | Evidence | Consequence |
|---|---|---|
| Instructions | `packages/context/agent-instructions/src/config.ts` defaults to `AGENTS.md` and `CLAUDE.md`; global instructions use `$DSH_HOME/AGENTS.md` | Write the existing managed `AGENTS.md` surface. Do not create another `CLAUDE.md` or assume existing Claude instructions are ignored. |
| Skills | `packages/skill/skill-filesystem/src/index.ts` discovers project `.dsh/skills` and `.agents/skills`, plus user skill roots | Own `.dsh/skills` exclusively. Explain discovery collisions rather than modifying Codex-owned `.agents/skills`. |
| Agents | `packages/preset/agent-preset-registry/README.md:46` states that the registry does not scan files; presets come from plugin declarations | Render Hero roles as instruction skills, not nonexistent `.dsh/agents` definitions. |
| MCP | `packages/mcp/mcp-client/src/index.ts:51` defines `serverName`, `transport`, `command`, `args`, `cwd`, and `env`; `apps/cli/package.json:51` includes the client plugin | Generate the native `@deepseek-ai/dsh-mcp-client` plugin configuration. |
| Configuration activation | `apps/cli/config/examples/mcp-memory/mcp-reference-memory.cordis.yml` demonstrates an `insert` patch; CLI profile composition loads bundles, profile patches, home patch, then repeatable `--patch` inputs | A project `.dsh` patch is **not automatically loaded**. Installation must report the launch command. |

These are source findings, not a claim that a live DeepSeek session has already been exercised. Delivery must validate the generated output with the real loader.

## Approach

### Native surfaces and ownership

| Content | Project install | Global install |
|---|---|---|
| Root instructions | `AGENTS.md` managed region | `$DSH_HOME/AGENTS.md` managed region |
| Canonical skills | `.dsh/skills/<name>/SKILL.md` | `$DSH_HOME/skills/<name>/SKILL.md` |
| Hero workflows | `.dsh/skills/command-<name>/SKILL.md` | `$DSH_HOME/skills/command-<name>/SKILL.md` |
| Hero role instructions | `.dsh/skills/role-<name>/SKILL.md` | `$DSH_HOME/skills/role-<name>/SKILL.md` |
| MCP overlay | `.dsh/hero.cordis.patch.yml` | `$DSH_HOME/hero.cordis.patch.yml` |

Resolve `DSH_HOME` as `packages/util/home-paths/src/index.ts` does: treat whitespace-only/unset values as `~/.dsh`, preserve nonblank path values, expand `~`, `~/`, and `~\`, and resolve relative paths against the invocation directory. It only redirects global installation. Project outputs remain below the selected workspace. No new dependency on a DeepSeek executable is required to install.

Use the existing command-as-skill renderer with a DeepSeek label. Add a small role-as-skill renderer with `name: role-<name>`, `user-invocable: false`, the canonical description, and the canonical role body. Leave model invocation enabled. DeepSeek accepts kebab-case `user-invocable` and `disable-model-invocation`, not legacy camel-case aliases (`skill-filesystem/src/index.ts:1001`). A short preamble must explain that this is role guidance: load it when adopting the role or pass the role instructions to a supported native delegation tool. It does not register a named subagent or grant tools, permissions, models, or hooks. Preserve source model/tool constraints as instructional metadata without claiming runtime enforcement. Preserve source instructions without hardcoding a Codex/Claude tool API. Validate rendered YAML scalars safely, including quoted/colon-containing descriptions.

DeepSeek's instruction section explains natural-language workflow routing to `command-*` skills, role guidance to `role-*` skills, and fallback to the current agent when no compatible delegation tool is available. Use the native `skill` tool with `{name: "command-design"}` or `{name: "role-engineer"}` when available (`packages/skill/tool-skill/src/index.ts:82`), with the installed file as a read fallback. The native subagent tool accepts configured persona guidance, not an `agent_type` lookup into files (`packages/subagent/tool-subagent/src/index.ts:50`). Do not advertise Hero workflow names as built-in slash commands. Keep canonical content authored once; target differences belong to the renderer/adapter. Shared `AGENTS.md` must retain this guidance after installing another target, and retain that target's guidance after DeepSeek installation, regardless of order. Existing user text remains unchanged.

Adopting a role locally is not independent review. If a workflow requires a fresh reviewer or cold audit and the selected profile cannot provide one, stop at that named gate and report the unavailable capability; never self-grade or mark delivery verified.

### Explicit MCP activation

Generate a standalone, Hero-owned Cordis patch using the inspected example's schema:

```yaml
- insert:
    - id: hero-mcp
      name: '@deepseek-ai/dsh-mcp-client'
      config:
        serverName: hero
        transport: stdio
        command: hero
        args: [mcp]
```

Use a YAML encoder, not interpolated shell or JavaScript. For the existing `--workspace` case, carry `Options.ProjectRoot` through as `args: [mcp, --project-root, <project-root>]`, as other MCP adapters do. Otherwise inherit the launch working directory and let Hero resolve its workspace; require users to launch from the intended workspace and test that binding. The reusable global overlay must not pin the repository from which installation ran. Keep the portable command `hero`; do not install binaries, alter PATH, store credentials, or emit `!!js` expressions.

The MCP child binds to its launch workspace, not to later DeepSeek session-directory changes. A GUI/server that hosts multiple repositories cannot share this single MCP child safely as a per-session Hero connection. Document one process/overlay activation per intended workspace; multi-workspace connection routing is out of scope. A globally installed overlay is reusable configuration, not a multi-project MCP router.

Install output and docs show an explicit launch from the desired workspace, for example `dsh --profile headless --patch /absolute/path/to/project/.dsh/hero.cordis.patch.yml 'Resume this Hero workspace'`. Quote the displayed path safely. Harness-specific arguments belong after `--patch` according to the inspected CLI. The normal interactive profile can use the same overlay. Report “overlay generated; activation required,” never “MCP connected.” `--json` output remains valid JSON; expose the generated file through existing result records without unsolicited stdout text.

Do not modify `$DSH_HOME/cordis.patch.yml`, profile patches, model selection, or allowlists. Treat an unknown existing overlay as user-owned: refuse replacement unless explicitly forced; identical generated bytes are a no-op, and prior Hero-owned bytes can refresh using recorded checksums. Report the collision before target mutation. The same ownership rule governs removal. No generic JSON/TOML MCP merger is appropriate here.

### Lifecycle and scope

Project install state records `deepseek` and all three skill families; upgrade, sibling auto-sync, pruning, inventory, doctor, and uninstall use those same paths. Agent and command counts are `NotApplicable` as independent native registries; expected skill count is canonical skills + workflows + roles. Missing-artifact checks enumerate actual skills and the MCP overlay. Doctor distinguishes generated files from runtime activation; it cannot infer a live MCP connection from disk.

Satellites link only `.dsh/skills`, use the shared `AGENTS.md` marker, and preserve existing satellite content. DeepSeek discovers skills at the first ancestor containing `.git`, otherwise at cwd (`skill-filesystem/src/index.ts:945`): inside the same Git tree it reads root skills, not nested satellite links; links support separately rooted/non-Git satellites. Test these cases rather than claiming every local link is discovered. Do not symlink settings or install fake agent/command directories. Satellite guidance points at the parent's explicit MCP overlay while preserving the satellite working directory; an explicitly bound `--workspace` overlay continues to use its declared root. The existing satellite cleanup path must remove only DeepSeek-owned links/markers.

Project uninstall removes owned generated skills, the owned overlay, and target state. Preserve shared `AGENTS.md` while another non-Claude target needs it; otherwise remove only its managed content. Preserve foreign skills, modified files according to existing uninstall ownership semantics, and all DeepSeek user configuration. Global lifecycle stays at the existing product boundary: support global install/reinstall and document removal of the generated global artifacts; do not add a new global-uninstall command. Record the global target's generated paths and checksums in `$DSH_HOME/hero-install-manifest.json` (versioned, paths relative to that home), excluding whole-file ownership of shared `AGENTS.md`. The manifest is target-local bookkeeping, not a Hero workspace. Only manifest-matching unchanged files may be refreshed/pruned automatically; unknown or modified files require explicit force or remain preserved. Reject malformed manifests before writes, honor dry-run, and document checksum-aware manual cleanup plus managed-region-only instruction removal. Reuse existing checksum primitives; do not create a general global lifecycle framework.

## Changes

1. **Target and native renderer:** add `TargetDeepSeek`/dispatch in `internal/install/install.go` and implement `internal/install/target_deepseek.go`. Extend `internal/install/render.go` for role skills and skill-family enumeration; reuse nested-skill writes and manifest pruning. Implement the target-local global manifest in `internal/install/deepseek_state.go` using existing checksum primitives. Reject namespace collisions and invalid global manifests before writes.
2. **Instructions and contracts:** extend `internal/install/agents_md.go` for global home resolution and DeepSeek workflow/role/activation guidance; ensure mixed-target order does not erase applicable routing. Add the actual skill-only contract in `internal/install/contracts.go`. Update the canonical eighth-target roster in `.hero/knowledge/tripwires/harness-changes-cover-all-targets/spec.md` (preserving its rule) and canonical harness descriptions in `domains/engineering/AGENTS.md` only where needed; regenerate derived root guidance through existing tests/install machinery, never hand-edit installed copies.
3. **MCP overlay:** add `internal/install/mcp_deepseek.go` and register it in `internal/install/mcp.go`. Implement owned YAML generation, dry-run and collision checks, workspace argument binding, and safe removal. Propagate target-specific registration failure rather than reporting a successful complete install. Do not broaden unrelated MCP merger behavior.
4. **Lifecycle integration:** extend `internal/install/auto_sync.go`, `inventory.go`, `satellite.go`, `satellite_detect.go`, and applicable state/ownership helpers. Add target acceptance and accurate messaging in `internal/cli/install.go`, `upgrade.go`, `uninstall.go`, and `doctor.go`; include overlay activation guidance without breaking JSON output. Include `.dsh` in harness-directory exclusions.
5. **Verification:** add `internal/install/deepseek_test.go` and `internal/cli/deepseek_uninstall_test.go`. Extend existing contract, inventory, pruning, native-root, routing, smoke, composition, attention, overlay, QA, satellite, doctor, upgrade, and prompt-target matrices. Keep tests data-driven for all eight targets; update changed prompt snapshots intentionally. Add an opt-in loader compatibility check against the supplied harness checkout without vendoring it or making normal Go tests depend on a sibling path.
6. **Documentation:** update `README.md`, `GETTING-STARTED.md`, `MCP-SETUP.md`, and the corresponding `web/docs/src/getting-started/project-setup.md`, `configuration/mcp-setup.md`, `concepts/agents-and-skills.md`, and `cli/server-and-mcp.md`. Explain `deepseek` vs `dsh`, native paths, explicit patch activation, roles-as-skills, `DSH_HOME`, discovery precedence, global removal, PATH/runtime selection, and the pinned compatibility baseline. Do not update historical release entries as if released.

## Acceptance Criteria

- **AC-1:** WHEN the user selects `deepseek` through the install flag or target picker THE SYSTEM SHALL accept it and install the native project layout above, without `.dsh/agents`, `.dsh/commands`, or writes to another harness's owned directories.
- **AC-2:** WHEN DeepSeek loads generated skills THE SYSTEM SHALL expose canonical skills, `command-*` workflows, and `role-*` guidance with valid names/descriptions and intact canonical bodies; instructions SHALL describe role/delegation and workflow limits accurately.
- **AC-3:** WHEN project or global installation generates its MCP overlay THE SYSTEM SHALL emit a loader-valid stdio Hero plugin patch and explicit activation instructions, preserve portable binary selection, and bind `--workspace` roots without pinning ordinary global installs.
- **AC-4:** WHEN installation uses global mode THE SYSTEM SHALL honor the harness's `DSH_HOME` rules, write global native paths only, and leave project files and shared Cordis/profile configuration unchanged.
- **AC-5:** WHEN installation or upgrade is repeated THE SYSTEM SHALL maintain correct DeepSeek state, refresh owned files, prune dropped owned skills across all three families, and preserve unrelated user content; `--dry-run` SHALL make no filesystem changes and overlay collisions SHALL fail before mutation unless forced.
- **AC-6:** WHEN DeepSeek is installed or missing generated artifacts THE SYSTEM SHALL include it in inventory/doctor, count role and workflow skills accurately, name missing output paths, and describe MCP activation as unverified rather than connected.
- **AC-7:** WHEN a DeepSeek satellite is materialized, repaired, or removed THE SYSTEM SHALL manage only the declared skills link and shared instruction marker, preserve foreign content, and provide usable parent-overlay activation guidance for that working directory.
- **AC-8:** WHEN DeepSeek is uninstalled from a project THE SYSTEM SHALL remove its owned output/state, preserve other harnesses and foreign content, and retain the shared root instruction region while a remaining target needs it; uninstall dry-run SHALL not mutate files.
- **AC-9:** WHEN DeepSeek coexists with Claude, Codex, or another supported target THE SYSTEM SHALL retain each installed target's routing under either install order and sibling synchronization, preserve the seven existing native layouts, and document unavoidable cross-discovery of user/other-harness instructions or skills.
- **AC-10:** WHEN generated output is checked against the pinned DeepSeek source THE SYSTEM SHALL pass real skill/instruction loading and Cordis-patch composition checks; a delivery smoke test SHALL start Hero through the generated MCP configuration and observe initialization plus tool listing without requiring a paid model call.

## Boundaries

- No DeepSeek model provider, credentials, binary installation, shell/PATH edits, or default application selection.
- No native preset/plugin authoring for Hero roles, new delegation runtime, custom slash commands, lifecycle hooks, or changes to the DeepSeek repository.
- No silent global Cordis activation or edits to user profiles; no claim that a generated overlay is automatically active.
  - **Superseded 2026-09-30 for project installs** by `deepseek-project-mcp-registration`. The DeepSeek desktop app never passes `--patch`, so project installs now register a Hero-owned, marker-delimited entry in `$DSH_HOME/cordis.patch.yml`. App-managed profile patches are still never edited.
- No redesign of all harness registries, global uninstall, or fixes to unrelated lifecycle defects. Existing targets change only for shared instructions, enumeration, and regression coverage necessary for the eighth target.
- One medium feature: a bounded adapter and its existing lifecycle joins. No independent platform deliverables warrant an initiative.

## Risks

- DeepSeek is source-verified at one revision. Runtime APIs and discovery precedence can change; retain compatibility evidence and stop to revise the spec if the pinned assumptions no longer hold.
- DeepSeek also loads `CLAUDE.md` and `.agents/skills`; mixed installations can duplicate instructions or collide on skill names. Hero owns `.dsh` but cannot suppress foreign loaders. Test observed precedence and document it; never delete other targets' files to compensate.
- The user must activate the overlay. Clear install/doctor guidance is part of correctness, not optional documentation. An unavailable `hero` on the harness's PATH remains an executable setup issue for `hero doctor`.
- Native delegation depends on the selected profile. Role skills provide guidance, not executable presets; a workflow must not claim to have delegated when it merely adopted a role.
- Existing local edits touch `inventory.go`, `doctor.go`, and `upgrade.go`. Delivery must inspect current changes and merge surgically; this planning task changes no implementation.

## Validation

Run `go test ./internal/install ./internal/cli`, then `go test ./...` and `make build`. Tests use temporary project/home directories and no model credentials. Include project/global/custom-home cases; malformed or foreign existing overlay; namespace collisions; path-with-spaces quoting; dry-run; no-op reinstall; dropped role/workflow skills; user-file retention; narrowed upgrade; sibling sync; missing output; JSON install output; uninstall with/without another AGENTS consumer; and satellite cleanup. Exercise engineering, PM, and QA overlays rather than hardcoding engineering-only inventory.

The compatibility check takes an explicit external harness path and records its commit. Use the real skill parser/discovery and instruction loader against generated temporary fixtures; compose the emitted Cordis patch with an installed CLI profile. Boot just the MCP client/service path, start `hero mcp`, and assert initialization and tool listing. Capture the commands/results in delivery evidence. If harness dependencies or runtime are unavailable, report that named obstacle and do not substitute a YAML parse for AC-10.

Run docs validation with the repository's documented docs commands. Review generated guidance across all eight targets, including DeepSeek-only, DeepSeek+Codex, and DeepSeek+Claude in both orders. Finish delivery through the normal Completion Ledger, cold audit, and `hero spec verify` gates.

## Completion Ledger

Implemented the Go installer and CLI lifecycle with the implementation-principles, go-stack, and testing-and-validation skills. Native compatibility uses the pinned DeepSeek checkout, not a simulated loader. Existing unrelated workspace changes remain outside this delivery.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Accept DeepSeek and install native layout | DONE | `internal/install/target_deepseek.go`, CLI picker; TestDeepSeekNativeLayoutAndInventory and prompt snapshots verify skill-only layout. |
| 2 | Valid canonical/workflow/role skills and accurate guidance | DONE | `internal/install/render.go`, `agents_md.go`; TestDeepSeekRenderingYAML and native loaders validate bodies, YAML, role metadata and invocation flags across engineering/PM/QA. |
| 3 | Native MCP overlay, explicit activation, workspace binding | DONE | `internal/install/mcp_deepseek.go`; TestDeepSeekInstallJSONWorkspace, TestDeepSeekWorkspaceOverlayRefresh, and native MCP initialize/list/hero_status exercise portable command and bound/unbound roots. |
| 4 | Global DSH_HOME and native paths | DONE | `target_deepseek.go`, `deepseek_state.go`; TestDeepSeekGlobalOwnershipAndHome verifies blank/relative/tilde/custom paths, isolated project/profile files and malformed manifests. |
| 5 | Ownership, repeat install, pruning, dry-run, preflight | DONE | `deepseek_state.go`, installer preflight; TestDeepSeekNamespaceCollisionAndPruning, TestDeepSeekGlobalRefreshPruneAndDryRun, TestDeepSeekWorkspaceCollisionFailsBeforeRootInstall and overlay collision tests assert preservation and no mutation. |
| 6 | Inventory and doctor reflect artifacts and activation limits | DONE |  `inventory.go`, `internal/cli/doctor.go`; TestDeepSeekNativeLayoutAndInventory and TestDeepSeekUpgradeSelectionAndDoctor verify counts and unverified activation; TestDoctorDeepSeekNamesMissingArtifactsDespiteFullCounts verifies exact missing paths and repair warnings even at full counts. |
| 7 | Satellite skills link, marker and parent activation | DONE | `satellite.go`, `satellite_detect.go`; TestDeepSeekSatelliteLayout and satellite coverage matrix assert owned links/cleanup; native Git-root discovery test checks nested behavior. |
| 8 | Uninstall owned output/state, preserve shared/user files | DONE | `internal/cli/uninstall.go`; TestDeepSeekUninstallOwnershipAndDryRun, TestDeepSeekUninstallKeepsSharedInstructionsAndClearsState and TestDeepSeekUninstallPreservesModifiedOverlay verify removal/preservation. |
| 9 | Mixed-target guidance and existing native layouts | DONE | `agents_md.go`, `auto_sync.go`; TestDeepSeekMixedRoutingAndSiblingSync and eight-target contract/native/smoke/composition/routing/PM/QA matrices pass; docs describe cross-discovery. |
| 10 | Real loaders, Cordis composition, MCP initialization/list | DONE | `scripts/deepseek-compatibility.mjs`; native-compatibility-{engineering,pm,qa}.log records 121/133/106 skills, 86 tools each, hero_status execution and explicit workspace binding, zero model calls. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Target, renderer, global manifest and collision preflight | DONE | `install.go`, `target_deepseek.go`, `render.go`, `deepseek_state.go`; new focused ownership/rendering tests. |
| 2 | Instructions, contracts, eighth-target roster | DONE | `agents_md.go`, `contracts.go`, canonical engineering AGENTS and tripwire; generated-content parity test passes. |
| 3 | Owned YAML MCP overlay and workspace integration | DONE | `mcp_deepseek.go`, `mcp.go`, CLI JSON/collision handling; real plugin client exercised. |
| 4 | Lifecycle, state, satellites, CLI and exclusions | DONE | Install/inventory/state/auto-sync/satellite paths and CLI install/upgrade/uninstall/doctor updated; `.gitignore` ignores generated DeepSeek output. |
| 5 | Focused tests, eight-target matrices, native runner | DONE | `deepseek_test.go`, `deepseek_uninstall_test.go`, existing data-driven matrices/snapshots and opt-in `scripts/deepseek-compatibility.mjs`. |
| 6 | User-facing documentation | DONE | README, GETTING-STARTED, MCP-SETUP and four web docs cover paths, roles, explicit activation, DSH_HOME, cleanup, discovery and PATH; 28 docs tests and strict MkDocs build pass. |

### Exercise-the-feature check

- [x] User-visible behavior was exercised end-to-end: `node scripts/deepseek-compatibility.mjs --harness /Users/bwheeler/projects/public/repository/deepseek-harness --hero /Users/bwheeler/projects/hero-engine/repository/hero/.build/hero-deepseek --domain <engineering|pm|qa>` installed temporary workspaces with the real CLI, loaded generated instructions/skills through DeepSeek, composed its headless profile and overlay, launched Hero through the native MCP client, listed 86 tools, and invoked hero_status from nested cwd and from outside an explicitly bound workspace.

### Excellence Bar self-check

- [x] Yes — native runtime evidence, safe ownership checks, all-domain/eight-target regression coverage and explicit activation documentation support the completion claim.
