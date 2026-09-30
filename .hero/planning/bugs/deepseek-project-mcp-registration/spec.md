---
title: "DeepSeek project install leaves MCP inactive — register Hero in the DeepSeek home patch"
slug: deepseek-project-mcp-registration
type: bug
status: delivering
priority: P1
severity: high
root_cause_class: design
domain: engineering
size: medium
created: 2026-09-30
tags: [deepseek, install, mcp, desktop, harness]
relations:
  - target: deepseek-harness-install-target
    kind: related
  - target: sept-review-cleanup
    kind: related
delivery_method: manual
---

# DeepSeek project install leaves MCP inactive

## Goal

After `hero install project <dir> --target deepseek`, every DeepSeek Harness
surface loads Hero's MCP server for that project with no extra launch flags.
That includes the desktop app, which never passes `--patch`. This is the
same bar every other install target meets.

## Kickoff

DeepSeek project installs now register Hero MCP in `$DSH_HOME/cordis.patch.yml`, which the desktop app and every `dsh` profile load:
- one marker-delimited entry per project, serverName `hero-<project>-<hash>`, absolute `hero` command, pinned `cwd`/`--project-root`
- doctor reports the registration; uninstall removes only that project's entry

**Status:** implemented; full suite, docs, and native desktop-path checks green. Closing gates: cold audit, then `hero spec verify deepseek-project-mcp-registration`.

**Pick up at:** run the audit on `git diff 3757f58e...HEAD -- ':!.hero'` if verify has not passed.

**Files:** `internal/install/deepseek_home.go`, `mcp_deepseek.go`, `target_deepseek.go`, `internal/cli/doctor.go`, `internal/cli/uninstall.go`, `scripts/deepseek-compatibility.mjs`

## Problem

Today a project install writes `.dsh/hero.cordis.patch.yml` and prints a
`dsh --profile web --patch '<path>'` launch command. That file loads only when
it's passed on the command line:
- The **DeepSeek Harness desktop app** (0.2.0-rc.2, profile `desktop`) always
  boots with `patchFiles: []` (`apps/desktop-host/src/index.ts`), so Hero MCP
  never loads there.
- CLI users must remember `--patch` on every launch.

The completed `deepseek-harness-install-target` spec chose this on purpose
("No silent global Cordis activation or edits to user profiles", line 142).
It only considered the `dsh` CLI. Every other Hero target registers MCP where
its harness reads it, and Codex already merges an entry into the user's own
`~/.codex/config.toml`.

## Source evidence (pinned harness `477b4f42`)

| Fact | Evidence | Consequence |
|---|---|---|
| Every profile, desktop included, composes bundle layers → profile patch → **home patch `$DSH_HOME/cordis.patch.yml`** → `--patch` overlays | `packages/boot/app-boot/src/profile-context.ts` `readProfilePatches`; desktop via `apps/desktop-host/src/index.ts` → `runProfile` | The home patch is the one layer that is loaded everywhere and not app-managed |
| The desktop app writes the profile patch (UI/model settings) but not the home patch | `~/.dsh/profiles/desktop/cordis.patch.yml` content; recovery logs `homePatch: 'unchanged'` (`apps/desktop/src/main.ts`) | Write the home patch, never the profile patch |
| The MCP client is a profile-level plugin spawned once per activation with a fixed `cwd` (default `''` = host cwd), not per session | `packages/mcp/mcp-client/src/index.ts` `StdioConfig`, `transport.ts` | One entry cannot follow a session's project; each project needs its own pinned entry |
| `serverName` must match `[A-Za-z0-9_-]{1,32}`, be unique across live clients, and namespaces tools as `mcp__<serverName>__*` | `StdioConfig.serverName` doc | Needs unique, stable per-project names, which can never be renamed after install |
| GUI apps do not inherit the shell PATH | macOS launchd environment | `command` must be an absolute path |

## Design

1. **Entry.** A project install upserts one element into the home patch's
   top-level list:
   `- insert: [{ id: hero-<key>-mcp, name: '@deepseek-ai/dsh-mcp-client', config: { serverName: hero-<key>, transport: stdio, command: <abs hero>, args: [mcp, --project-root, <abs root>], cwd: <abs root> } }]`.
   - `<key>` is the project folder name, lowercased and sanitized to
     `[a-z0-9-]`, truncated so `hero-<key>` stays ≤ 32 characters, plus `-`
     and the first 4 hex characters of the SHA-256 of the absolute project
     root. Stable and unique.
2. **Merge.** Hero writes only its own text block between
   `# hero:managed deepseek-mcp <server>` / `# end:hero:managed deepseek-mcp <server>`
   markers, the same contract as the Codex/Grok TOML upsert. Bytes outside
   the markers are never rewritten, so foreign entries and comments survive
   byte-for-byte.
   - A new file starts with a one-line "Created by hero install" header.
   - Before any mutation, refuse a file that isn't a YAML list, isn't a
     regular file, or is a symlink, or has an unterminated Hero block.
   - Reinstall replaces the block in place.
3. **Binary.** Use `exec.LookPath("hero")` as found (absolute, not following
   links, so upgrades via Homebrew or `make install` carry over). Fall back to
   `os.Executable()` only if it isn't under a temp or `go-build` directory.
   Otherwise fail with an actionable error naming `make install`.
4. **Project overlay.** Stop generating `.dsh/hero.cordis.patch.yml`. On
   reinstall or upgrade, prune it through the existing checksum-owned prune
   path. If it was modified, preserve it and warn: passing it via `--patch`
   now adds a second Hero server.
5. **Scope of other install modes.**
   - `--workspace` subfolder installs register the same project-root entry
     (idempotent).
   - Global install keeps its skills/AGENTS.md and `$DSH_HOME/hero.cordis.patch.yml`
     behavior unchanged. An unpinned home entry would start with no workspace.
   - Satellites are unchanged.
6. **Instructions.** `AGENTS.md` is committed and shared, but `<hash>`
   depends on the machine-local project path (and differs per worktree). So
   the DeepSeek section describes the `mcp__hero-<project>-<hash>__*` pattern
   and tells the model to confirm with that server's `hero_status`. The exact
   name appears in install output and `hero doctor`.
7. **Install output and doctor.**
   - Install prints the server name, the home patch path, and "restart the
     DeepSeek desktop app, or start `dsh web`; no `--patch` needed".
   - Doctor reads the home patch and reports this project's entry as
     `registered` (entry present, command executable, root matches) or names
     the problem: missing, stale command path, or wrong root.
   - `DeepSeekLaunchCommand` becomes `dsh --profile web`, with no patch.
8. **Uninstall.** `hero uninstall --target deepseek` removes only this
   project's element; `--dry-run` previews. If that leaves the home patch as
   an empty list, remove the file only when Hero created it; Hero records
   creation in the project's install state.
9. **Tripwire scope.** This is a DeepSeek-only change. The other seven
   targets' MCP registration is untouched, and existing matrices must stay
   green.
10. **Supersede.** Amend `deepseek-harness-install-target` line 142 with a
    dated note. Its "no automatic activation" boundary is superseded for
    project installs by this spec, because the desktop app has no `--patch`
    path.

## Acceptance Criteria

- **AC-1:** WHEN `hero install project <dir> --target deepseek` runs THE SYSTEM SHALL upsert exactly one Hero entry for that project into `$DSH_HOME/cordis.patch.yml`, with id `hero-<key>-mcp`, serverName `hero-<key>` (≤ 32 chars, `[A-Za-z0-9_-]`), an absolute `command`, and `args`/`cwd` pinned to the absolute project root.
- **AC-2:** IF the home patch contains foreign entries or comments THEN THE SYSTEM SHALL preserve them byte-for-byte outside Hero's own element, and a repeat install SHALL be a no-op.
- **AC-3:** WHEN two different projects are installed THE SYSTEM SHALL keep two entries with distinct ids and serverNames, and neither install SHALL alter the other's entry.
- **AC-4:** IF the home patch is not a YAML sequence, is a symlink, or is not a regular file THEN THE SYSTEM SHALL fail before any file mutation with an actionable error; `--dry-run` SHALL write nothing.
- **AC-5:** WHEN install resolves the Hero binary THE SYSTEM SHALL use the PATH `hero` (absolute, not following links), else a non-temporary `os.Executable()`, and SHALL otherwise fail naming `make install`.
- **AC-6:** THE SYSTEM SHALL stop generating `.dsh/hero.cordis.patch.yml` for project installs; an unmodified previously generated one SHALL be pruned and a modified one preserved with a warning.
- **AC-7:** WHEN DeepSeek is installed THE SYSTEM SHALL print the project's exact server name and home-patch path in install output, describe the `hero-<project>-<hash>` naming pattern (with a `hero_status` check) in the DeepSeek AGENTS.md section, and give launch guidance without `--patch`.
- **AC-8:** WHEN `hero doctor` runs in a project with DeepSeek installed THE SYSTEM SHALL report the home-patch entry as registered, or name the missing, stale-command or wrong-root problem, instead of "activation unverified".
- **AC-9:** WHEN `hero uninstall --target deepseek` runs THE SYSTEM SHALL remove only this project's element, and remove the file only if Hero created it and it is now empty; dry-run SHALL not mutate.
- **AC-10:** WHEN the pinned harness composes the `web` profile with the generated home patch and no `--patch` THE SYSTEM SHALL yield exactly one Hero MCP client for the project, and the native client SHALL initialize, list tools and execute `hero_status` against that project.
- **AC-11:** THE SYSTEM SHALL leave MCP registration for the other seven targets unchanged (existing target matrices pass).

## Boundaries

- Never write `~/.dsh/profiles/*/cordis.patch.yml`; the app manages it.
- No global-install home entry, no per-session project routing, no DeepSeek
  source changes, no dsh or app installation.
- No edits to foreign entries in the home patch, including other tools' MCP
  servers.
- No change to skills layout, role rendering or satellites beyond the AGENTS.md
  server-name line.

## Risks

- A home patch is shared by every profile, so each installed project's server
  starts in every DeepSeek session. That is acceptable for a handful of
  projects. The per-project server name plus AGENTS.md guidance keeps the model
  on the right one. Document the cost and `hero uninstall` as the cleanup.
- yaml.v3 round-trips can reflow foreign formatting. Mitigate with a golden
  test on a hand-written file with comments. If the re-encode is not
  byte-stable outside Hero's element, fall back to text-splicing only Hero's
  element.
- The harness contract can change after `477b4f42`. The compatibility script
  remains the check, and it records the harness revision.

## Validation

- **Unit tests** (`internal/install`, temp `HOME`/`DSH_HOME`): create, upsert,
  no-op repeat, foreign preservation (golden file with comments), two
  projects, malformed/symlink/non-regular refusal before mutation, dry-run,
  binary resolution (PATH, fallback, temp refusal), overlay prune and
  preserve, uninstall only-ours and created-file cleanup.
- **CLI tests:** doctor registered, missing, stale-command and wrong-root
  output; uninstall dry-run.
- **Native:** update `scripts/deepseek-compatibility.mjs` to set
  `DSH_HOME` to a temp home and install through the real CLI. Compose `web`
  (the desktop profile's bundles) plus `headless` with **no overlays**. Assert
  exactly one Hero client with the expected serverName and cwd, then
  initialize, list tools and call `hero_status`, for engineering, pm and qa.
- **Full:** `go test ./... -count=1`, `go vet ./...`, docs tests, strict
  MkDocs.
- **Manual (user):** restart the desktop app in this repo and confirm the
  `mcp__hero-hero-<hash>__*` tools appear.

## Changes

1. `internal/install/mcp_deepseek.go` — home-patch upsert and removal (yaml Node), key/serverName derivation, binary resolution, `DeepSeekLaunchCommand` without patch; stop the project overlay.
2. `internal/install/deepseek_state.go`, `target_deepseek.go` — record home-patch ownership (entry id, created-file flag) in install state; prune the legacy overlay.
3. `internal/install/agents_md.go` — DeepSeek section names the project's server.
4. `internal/cli/doctor.go`, `internal/cli/uninstall.go` (+ tests) — registration status; entry removal.
5. `scripts/deepseek-compatibility.mjs` — home-patch composition with no overlays.
6. Docs: `MCP-SETUP.md`, `web/docs/src/configuration/mcp-setup.md`, `GETTING-STARTED.md` DeepSeek sections — home-patch registration, desktop restart, uninstall cleanup.
7. `.hero/specs/deepseek-harness-install-target/spec.md` — dated supersede note on the activation boundary.

## Later: native DeepSeek plugin

This spec is the short-term fix. The longer-term direction, deferred by the
user on 2026-09-30, is a Hero DeepSeek plugin bundle, installable from the
desktop app's Plugins page. It would provide:
- native tools that route by each session's workspace
- a skills provider, slash commands and agent presets
- a live system-prompt section
- session lifecycle hooks

See the "Desktop app and plugin model" section of
`.hero/knowledge/context/deepseek-harness-native-surfaces/spec.md`. When
picked up, it should go through `/compose` as an initiative.

## Completion Ledger

Implemented inline (implementation-principles, go-stack, testing-and-validation). Base `3757f58e`. Evidence logs are in this spec directory.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | AC-1: one entry per project, id/serverName/absolute command/pinned root | DONE | `deepseek_home.go` `UpsertDeepSeekHomeEntry`, `DeepSeekServerName`, called from `registerMCPDeepSeek` for project mode. Tests: `TestDeepSeekHomeEntryCreateAndNoop` asserts id, name, serverName (≤32, charset), command, cwd, `--project-root`; `TestDeepSeekNativeLayoutAndInventory` |
| 2 | AC-2: foreign content preserved byte-for-byte; repeat is a no-op | DONE | Marker-delimited text splice. Tests: `TestDeepSeekHomeEntryPreservesForeignContent` (original with comments restored exactly after uninstall); `TestDeepSeekHomeEntryCreateAndNoop` (repeat `changed=false`, bytes identical) |
| 3 | AC-3: two projects, distinct entries, independent | DONE | `TestDeepSeekHomeEntryTwoProjects` |
| 4 | AC-4: non-list, invalid, symlink, or non-regular fails before mutation, including dry-run | DONE | `PreflightDeepSeekHome` in `planDeepSeek`, which runs before the legacy migration. Tests: `TestDeepSeekHomePatchRefusedBeforeMutation` (3 cases × dry/non-dry, asserts no AGENTS.md/.dsh); CLI `TestDeepSeekWorkspaceCollisionFailsBeforeRootInstall` (malformed home patch, root untouched) |
| 5 | AC-5: PATH hero (absolute, links kept), else non-temporary executable, else actionable error | DONE | `resolveDeepSeekHeroCommand`, plus `HERO_DEEPSEEK_MCP_COMMAND` pin (absolute required). `TestDeepSeekHeroCommandResolution`. Both TestMains pin a fake executable so tests don't depend on the host PATH |
| 6 | AC-6: no project overlay; unmodified legacy overlay pruned, modified kept with warning | DONE | `deepseekFiles` writes the overlay only in global mode; `pruneDeepSeek` warning. Tests: `TestDeepSeekLegacyOverlayPruning` (both cases), `TestDeepSeekLegacyOverlayPreservedAndRemovable` |
| 7 | AC-7: install output names server and path; AGENTS.md naming pattern plus hero_status check; no `--patch` | DONE | `registerMCPDeepSeek` prints server and path; `runDeepSeek` project message; `renderDeepSeekWorkflowSection` updated (asserted in `TestDeepSeekLegacyOverlayPreservedAndRemovable`); satellite guidance (`TestDeepSeekSatelliteLayout`); docs rewritten |
| 8 | AC-8: doctor reports registered, or names missing/stale/wrong-root with a repair verdict | DONE | `InspectDeepSeekRegistration`; `doctor.go` footnote and `deepseekMCPBroken` verdict. Tests: `TestDeepSeekRegistrationProblems`, `TestDoctorDeepSeekRegistrationProblemNeedsRepair`, `TestDoctorDeepSeekNamesMissingArtifactsDespiteFullCounts`, `TestDeepSeekUpgradeSelectionAndDoctor` |
| 9 | AC-9: uninstall removes only this entry; file removed only if Hero-created and empty; dry-run safe | DONE | `RemoveDeepSeekHomeEntry` wired in `uninstallDeepSeek`. Tests: `TestDeepSeekHomeEntryPreservesForeignContent` (dry-run no-op, exact restore), `TestDeepSeekHomeEntryTwoProjects` (file removed when empty), `TestDeepSeekUninstallOwnershipAndDryRun`, `TestDeepSeekInstallJSONWorkspace` |
| 10 | AC-10: pinned harness composes web/headless plus home patch with no --patch, one Hero client; initialize, list, hero_status | DONE | `scripts/deepseek-compatibility.mjs` also calls the desktop host's `readProfilePatches` with no overlays, and starts MCP from outside the project. `native-compatibility-{engineering,pm,qa}.log`: 86 tools, zero model calls |
| 11 | AC-11: other seven targets unchanged | DONE | Only DeepSeek code paths changed; full `go test ./...` passes, including the all-target contract, native, smoke, routing, composition, PM and QA matrices (`go-tests.log`). `mcp_test` portable-command matrix still covers the other 6 MCP targets |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | mcp_deepseek.go home upsert/removal, naming, binary resolution | DONE | Core logic lives in the new `deepseek_home.go`; `mcp_deepseek.go` dispatches by mode. `DeepSeekOverlayPath` became `DeepSeekMCPConfigPath` (CLI `--workspace` JSON reports the home patch) |
| 2 | state/target: ownership and legacy overlay prune | DONE | Ownership is the per-project marker plus the Hero-created header line, not install state (simpler, survives state loss). Legacy prune reuses the checksum prune |
| 3 | agents_md.go DeepSeek section | DONE | Naming pattern and `hero_status` instruction; see AC-7 for why not the exact name |
| 4 | doctor.go, uninstall.go (+ tests) | DONE | See AC-8/9 |
| 5 | compatibility script | DONE | Home patch, desktop-path `readProfilePatches`, outside-project MCP start, workspace no-duplicate |
| 6 | Docs | DONE | `MCP-SETUP.md`, `web/docs/src/configuration/mcp-setup.md` (full section), `README.md`, `GETTING-STARTED.md`, `web/docs/src/getting-started/project-setup.md`, `web/docs/src/cli/server-and-mcp.md` |
| 7 | Supersede note on deepseek-harness-install-target | DONE | Dated sub-bullet under its line-142 boundary |

### Exercise-the-feature check

- [x] `node scripts/deepseek-compatibility.mjs` against pinned harness `477b4f42` passes for engineering, pm and qa. The real CLI installs into a temp workspace with an isolated `DSH_HOME`. The desktop host's own `readProfilePatches` with `overlays: []` then yields exactly one Hero client. The real MCP client starts `hero` from outside the project, lists 86 tools as `hero-<project>-<hash>` and executes `hero_status`. A `--workspace` reinstall adds no duplicate.

### Excellence Bar self-check

- [x] Yes. The fix targets the layer the desktop app actually loads, verified with its own composition function. It follows Hero's existing marked-span ownership contract. Tests cover every safety rule.

## Final validation

`make build` (`build.log`), `go vet ./...` (`vet.log`) and `go test ./... -count=1` (`go-tests.log`) pass. Docs unit tests (`docs-tests.log`) and `mkdocs build --strict` (`docs-build.log`) pass. `git diff --check` is clean.
