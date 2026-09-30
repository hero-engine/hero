# Delivery audit: deepseek-project-mcp-registration

**Audited:** `git diff 3859755d...1ceaef1c` (commit `1ceaef1c`), `.hero` projection files excluded
**Verdict:** HOLD
**Surface:** noteworthy

## Acceptance criteria
- [✓] AC-1: one entry per project, with id `hero-<key>-mcp`, a valid serverName, an absolute command and a pinned root. Evidence: `internal/install/deepseek_home.go:57-71,123-144`, called from `mcp_deepseek.go` `registerMCPDeepSeek` in project mode. `TestDeepSeekHomeEntryCreateAndNoop` asserts id, name, serverName charset and length, command, cwd and `--project-root`. Auditor probe: odd folder names (`é漢字`, `---`, a 58-character name with spaces and uppercase, `.hidden`, `under_score`) all produced valid names of 32 characters or fewer, and the name was the same with and without a trailing slash.
- [~] AC-2: foreign bytes preserved, and a repeat install is a no-op. The repeat no-op and the golden round-trip (`TestDeepSeekHomeEntryPreservesForeignContent`) hold, and a CRLF foreign file is restored exactly. Two exceptions:
  - A foreign file with no trailing newline gets a `\n` appended (`deepseek_home.go:247-248`). Uninstall does not remove it, so the file is not restored byte-for-byte.
  - The splice can produce YAML that the harness rejects. See the blocker in Audit notes.
- [✓] AC-3: two projects keep distinct, independent entries. Evidence: `TestDeepSeekHomeEntryTwoProjects`.
- [✓] AC-4: a home patch that is not a list, is a symlink, is not a regular file, or has an unterminated block is refused before any mutation, and dry-run writes nothing.
  - `planDeepSeek` → `PreflightDeepSeekHome` (`target_deepseek.go:153-161`) runs at the top of `install.Run` (`install.go:160-165`), before the legacy migration and before any renderer.
  - The CLI `--workspace` path preflights through `RegisterMCP` with DryRun forced (`cli/install.go:345-351`) before `install.Run`.
  - Upsert returns before `MkdirAll` when running dry (`deepseek_home.go:252`).
  - Tests: `TestDeepSeekHomePatchRefusedBeforeMutation` (3 cases, each with and without dry-run), `TestDeepSeekWorkspaceCollisionFailsBeforeRootInstall`.
- [✓] AC-5: binary resolution. Order is the env pin (absolute only), then `exec.LookPath` made absolute without following links, then `os.Executable()` unless transient, else an error naming `make install` (`deepseek_home.go:75-99`). Test: `TestDeepSeekHeroCommandResolution`. See the notes on the limits of the transient-path check.
- [✓] AC-6: project installs no longer write the overlay (`target_deepseek.go:110-118`, global mode only). An unmodified legacy overlay is pruned; a modified one is kept with a warning (`deepseek_state.go:150-154`). Tests: `TestDeepSeekLegacyOverlayPruning`, `TestDeepSeekLegacyOverlayPreservedAndRemovable`.
- [✓] AC-7: install prints the server name and home-patch path (`mcp_deepseek.go` `registerMCPDeepSeek`). The project message has no `--patch` (`target_deepseek.go:205-209`). The AGENTS.md DeepSeek section describes the `hero-<project>-<hash>` pattern and the `hero_status` check (`agents_md.go:743`). Docs were updated.
- [✓] AC-8: doctor reports "registered", or names the missing, malformed, stale-command or wrong-root problem, with a NEEDS REPAIR verdict (`deepseek_home.go:296-337`, `cli/doctor.go`). Tests: `TestDeepSeekRegistrationProblems`, `TestDoctorDeepSeekRegistrationProblemNeedsRepair`.
- [✓] AC-9: uninstall removes only this project's block. The file is removed only when it starts with Hero's created-header and nothing but whitespace remains (`deepseek_home.go:263-284`). Dry-run does not write. Tests: `TestDeepSeekUninstallOwnershipAndDryRun`, `TestDeepSeekHomeEntryTwoProjects`, `TestDeepSeekHomeEntryPreservesForeignContent`.
- [✓] AC-10: `native-compatibility-{engineering,pm,qa}.log` all pass against harness `477b4f42`. The runs compose `web` and `headless` with the home patch and no `--patch`, run the desktop-path `readProfilePatches` with `overlays: []`, get one Hero client, list 86 tools and execute `hero_status` from outside the project. Premises confirmed in the harness source:
  - `apps/desktop-host/src/index.ts:29` has `patchFiles: []`.
  - `profile-context.ts:63-75` composes `join(context.home, PROFILE_PATCH_FILENAME)`.
  - `mcp-client/src/index.ts:40` has `SERVER_NAME_PATTERN = /^[A-Za-z0-9_-]{1,32}$/`.
- [✓] AC-11: the non-DeepSeek code paths are untouched. Doctor, inventory and satellite changes are all inside `TargetDeepSeek` branches, and the AGENTS.md edit is DeepSeek-only. `go-tests.log` has 0 FAIL, and the auditor re-ran `go test ./internal/install ./internal/cli -count=1`, which passed. DeepSeek was deliberately removed from `TestRegisterMCP_CommandIsPortable_AllTargets`, and the other 6 MCP targets remain covered there.

## Changes
- [~] 1. Home-patch upsert and removal, naming, binary resolution. Implemented, in the new `deepseek_home.go` (text splice instead of a yaml Node; the spec's Risks section allowed this fallback). The splice does not check that the result is still a list the harness can load. See the blocker below.
- [✓] 2. State, target and legacy prune. Ownership is recorded as a created-header line rather than in install state. The deviation is stated in the ledger and the behavior is sound.
- [✓] 3. `agents_md.go` DeepSeek section.
- [✓] 4. `doctor.go`, `uninstall.go`, with tests.
- [✓] 5. `scripts/deepseek-compatibility.mjs`: home patch, desktop-path composition, start from outside the project, no duplicate on workspace reinstall.
- [✓] 6. Docs: `MCP-SETUP.md`, `web/docs/src/configuration/mcp-setup.md`, `GETTING-STARTED.md`, `README.md`, `project-setup.md`, `server-and-mcp.md`. `docs-tests.log` and `docs-build.log` pass.
- [✓] 7. Dated supersede note in `.hero/specs/deepseek-harness-install-target/spec.md`.

## Open items
- None. The ledger has no PARTIAL, SKIPPED or BLOCKED rows.

## Audit notes
- **BLOCKER: the append splice can turn a valid, harness-accepted home patch into one that makes every DeepSeek profile fail to boot.** `readDeepSeekHomePatch` accepts any YAML list. `UpsertDeepSeekHomeEntry` then appends a column-0 block sequence (`deepseek_home.go:244-250`), but nothing checks that the combined text still parses. Auditor probe using the copied Go logic plus the harness's own `js-yaml`:

  | Home patch before install | Harness before | Harness after Hero install |
  |---|---|---|
  | `[]` (flow-style empty list) | loads `[]` | throws "end of the stream or a document separator is expected" |
  | `  - insert: []` (indented block list) | loads OK | throws, same error |
  | `- insert: []` followed by `...` (explicit document end) | loads OK | throws "expected a single document" |

  The harness's `loadOptionalPatches` → `parsePatchList` throws on a parse error, so every profile is affected, including the desktop app. That is the opposite of this spec's goal. yaml.v3 does not catch these cases the same way: it parsed some of the results silently with Hero's entry dropped.

  Suggested fix, one function: after composing `next`, unmarshal it and require a list with exactly one more element than before, containing Hero's entry. Otherwise refuse before writing. This also covers the marker-anchoring case below. Add these three inputs to `TestDeepSeekHomePatchRefusedBeforeMutation` or to the upsert tests.
- Markers are matched as substrings, not anchored to the start of a line (`cutDeepSeekHomeBlock`, `deepseek_home.go:150-154`). In a contrived case, a foreign comment containing the start and end marker text causes foreign lines to be cut, and the result was invalid YAML. A Hero block converted to CRLF by an editor is not found (the search is for `start+"\n"`). Reinstall then appends a duplicate block with the same serverName, and doctor reports "missing". Both are low-probability. Anchoring to the start of a line and accepting `\r\n` would close them.
- A foreign file with no trailing newline gains a newline, and uninstall does not remove it (AC-2, 1 byte).
- Binary resolution:
  - The `exec.LookPath` result is not filtered for temporary or go-build paths. The spec intends this ("as found"), so a `hero` that is on PATH inside a temp directory would be written.
  - `transientExecutable` compares against `os.TempDir()` only. On macOS it does not catch the `/tmp` or `/private/var/folders` aliases. The `/go-build` check does catch `go run` and `go test` binaries.
  - The `HERO_DEEPSEEK_MCP_COMMAND` pin is not checked for temp paths. That is deliberate as a developer override, but it is undocumented in the spec.
- Uninstall is not atomic. If the home patch is invalid, `RemoveDeepSeekHomeEntry` errors after `.dsh/skills` has already been removed (`cli/uninstall.go:353-368`).
- serverName stability:
  - It depends on the exact path casing. On case-insensitive macOS volumes, `~/Projects/Hero` and `~/projects/hero` hash differently, and `EvalSymlinks` does not normalize case, so that produces a second entry.
  - The 4-hex suffix (16 bits) makes collisions between same-named folders unlikely but possible. A collision would make two projects overwrite each other's entry.
- `DeepSeekLaunchCommand` still emits `--patch`, but it is now used only on the global-mode path. The project path prints `dsh --profile web` without it, so the intent of Design 7 is met.
- `internal/install/mcp_test.go` has a stray blank line in its import block (`gofmt -l` flags it). This is cosmetic, and the repo has many other unformatted files.
- The evidence logs report `heroVersion ... g3757f58e-dirty`, meaning they were run on the pre-commit working tree. The auditor re-ran the install and CLI package tests on `1ceaef1c`, and they pass.
