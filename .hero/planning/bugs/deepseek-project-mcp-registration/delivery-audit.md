# Delivery audit: deepseek-project-mcp-registration (round 2)

**Audited:** `git diff 3859755d...b15cd0a9` (commits `1ceaef1c` and `b15cd0a9`), `.hero` projection files excluded
**Verdict:** HOLD
**Surface:** noteworthy

The round-1 blocker is fixed: every file Hero writes on install now parses in the harness. Round 2 found a new defect, introduced by the round-1 trailing-newline fix. When a later block follows a block flagged `(added-newline)`, uninstall can orphan another project's entry or produce a home patch the harness cannot parse. See the blocker in Audit notes.

## Acceptance criteria
- [✓] AC-1: one entry per project, with an id, a valid serverName (at most 20 name characters plus a 6-hex hash, 32 total), an absolute command and a pinned root. Evidence: `internal/install/deepseek_home.go:58-81,161-177`. Tests: `TestDeepSeekHomeEntryCreateAndNoop`, `TestDeepSeekServerNameShapeAndCase`. On darwin and windows the root is case-folded for the hash (`rootIdentity`).
- [~] AC-2: foreign bytes preserved and repeat install is a no-op. The round-1 gaps are fixed and tested:
  - Markers now match only as whole lines, with CRLF tolerated (`cutDeepSeekHomeBlock`, `:198-232`).
  - A separator newline Hero adds is flagged on the start marker and removed on uninstall.
  - A CRLF-converted block is no longer duplicated.

  Auditor probes restored the original file exactly for these inputs: no trailing newline, a trailing comment, CRLF, CRLF with no trailing newline, a `---` start, a block scalar at end of file (with and without a trailing newline), anchors and aliases, and a two-project install removed in reverse order.

  One regression: when the flagged block is not the last thing in the file, removing it glues the next line onto the foreign line. See the blocker.
- [✓] AC-3: two projects keep distinct, independent entries (`TestDeepSeekHomeEntryTwoProjects`). The one exception is the removal-order case in the blocker.
- [✓] AC-4: unsafe or unextendable home patches are refused before any mutation, and dry-run writes nothing.
  - `PreflightDeepSeekHome` is now a dry-run of the same upsert (`:291-294`), run by `planDeepSeek` at the top of `install.Run`.
  - The re-parse guard (`:335-344`) requires the result to be a list with exactly one more element containing Hero's id.
  - Tests: `TestDeepSeekHomePatchUnextendableLayoutsRefused` (flow `[]`, indented list and `...` end, each left byte-identical), `TestDeepSeekHomePatchRefusedBeforeMutation`, `TestDeepSeekWorkspaceCollisionFailsBeforeRootInstall`.
  - Auditor probe: the three round-1 layouts are refused and the files are unchanged. A tab-indented comment, which js-yaml accepts, is also refused because yaml.v3 rejects it. That is conservative and safe.
- [✓] AC-5: PATH `hero` and `os.Executable()` both go through `transientExecutable`, which also checks the resolved path against `/tmp/`, `/private/tmp/`, `/var/folders/`, `/private/var/folders/` and `os.TempDir()` in both raw and resolved form (`:85-132`). The env pin must be absolute and is an explicit override by design. Test: `TestDeepSeekHeroCommandResolution`.
- [✓] AC-6: legacy overlay behavior is unchanged from round 1: pruned when unmodified, kept with a warning when modified (`TestDeepSeekLegacyOverlayPruning`, `TestDeepSeekLegacyOverlayPreservedAndRemovable`).
- [✓] AC-7: install output, AGENTS.md pattern and docs are unchanged from round 1. The doc examples are updated to 6-hex names (`MCP-SETUP.md`, `web/docs/src/configuration/mcp-setup.md`).
- [✓] AC-8: doctor reports registration status. Roots are now compared through `rootIdentity` (`:426`). Tests: `TestDeepSeekRegistrationProblems`, `TestDoctorDeepSeekRegistrationProblemNeedsRepair`.
- [~] AC-9: uninstall now removes the home entry before any project file (`internal/cli/uninstall.go:353-372`; test `TestDeepSeekUninstallFailsBeforeRemovingFilesOnBadHomePatch`). Dry-run does not write, and the Hero-created file is removed only when empty. However, `RemoveDeepSeekHomeEntry` does not re-validate its output, and the added-newline strip is unconditional. See the blocker.
- [✓] AC-10: `native-compatibility-{engineering,pm,qa}.log` pass against harness `477b4f42`, with 6-hex server names, 86 tools, desktop-path composition with no overlays, and `hero_status` executed.
- [✓] AC-11: the round-2 changes touch only `deepseek_home.go`, `cli/uninstall.go` (DeepSeek function), tests and DeepSeek docs. `go-tests.log` has 0 failures. The auditor re-ran `go test ./internal/install ./internal/cli -count=1` on `b15cd0a9`, and both pass.

## Changes
- [~] 1. Home-patch upsert and removal, naming, binary resolution. The upsert is now validated. Removal is not, and has the defect below.
- [✓] 2. State, target and legacy prune: ownership via the created-header line (deviation disclosed in the ledger).
- [✓] 3. `agents_md.go` DeepSeek section.
- [✓] 4. `doctor.go` and `uninstall.go`, with tests. Uninstall ordering is fixed.
- [✓] 5. `scripts/deepseek-compatibility.mjs`.
- [✓] 6. Docs. `docs-tests.log` and `docs-build.log` pass.
- [✓] 7. Supersede note on `deepseek-harness-install-target`.

## Open items
- None. The ledger has no PARTIAL, SKIPPED or BLOCKED rows.

## Harness parse check (js-yaml from the pinned harness)
The auditor copied `deepseek-home.go` at `b15cd0a9` into a scratch program and ran install, second project, optional edit, and uninstall over 17 starting layouts. Each resulting file was loaded with the harness's `js-yaml` using the `parsePatchList` rules (a top-level array of mappings).

- **Every file Hero wrote on install or second install parses as a list with the expected Hero entry count.**
- Every refusal left the file unchanged.
- Uninstalls parse, with two exceptions:
  - An empty or comment-only file that was already there is restored to its original bytes, which the harness already rejected before Hero touched it. This is correct.
  - The two cases below.

## Audit notes
- **BLOCKER: removing a block flagged `(added-newline)` joins whatever follows it onto the foreign last line.** Cause: `cutDeepSeekHomeBlock` strips the flagged newline unconditionally (`deepseek_home.go:224-226`), even when content follows the block. `RemoveDeepSeekHomeEntry` (`:356-377`) writes without re-parsing. Reproduced:
  1. **Two projects, remove the first.** Start with a home patch `- insert: []` that has no trailing newline. Install project A, then B, then uninstall A. The result is `- insert: []# hero:managed deepseek-mcp hero-b…`. B's start marker is no longer a whole line, so:
     - doctor reports B as "no Hero MCP entry";
     - reinstalling B is refused (a duplicate id makes the count 2);
     - uninstalling B cannot find it.

     B's live entry is orphaned permanently in the home patch. js-yaml happens to still load it.
  2. **User appends after Hero's block.** Same starting file. Install A, then the user appends `- remove: [y]`, then uninstall A. The result is `- insert: []- remove: [y]`, and js-yaml throws `bad indentation of a mapping entry`. Every DeepSeek profile, desktop included, then fails to boot: the same failure class as the round-1 blocker, now on the uninstall path.

  Suggested fix:
  - Strip the flagged newline only when the block ends at end of file (`lineEnd == len(rest)`). Otherwise leave it, since the newline now separates real content.
  - Give `RemoveDeepSeekHomeEntry` the same guard as upsert: re-parse and require exactly one fewer element with zero Hero ids for this server, or refuse.
  - Add both cases to `TestDeepSeekHomeEntryRestoresMissingTrailingNewline`.
- Non-blocking:
  - The `DeepSeekServerName` doc comment still says `<hash4>` (`deepseek_home.go:55`).
  - Case-folding on darwin means two projects on a case-sensitive APFS volume whose paths differ only in case would share one server name. This is extremely rare and accepted.
  - The `HERO_DEEPSEEK_MCP_COMMAND` pin is not checked for temporary paths, by design.
  - The evidence logs were produced on the `1ceaef1c-dirty` working tree. The auditor re-ran the install and CLI packages on `b15cd0a9`, and both pass.
