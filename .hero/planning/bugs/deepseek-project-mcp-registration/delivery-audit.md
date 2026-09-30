# Delivery audit: deepseek-project-mcp-registration (round 3)

**Audited:** `git diff 3859755d...3deea556` (commits `1ceaef1c`, `b15cd0a9`, `3deea556`), `.hero` projection files excluded
**Verdict:** SHIP
**Surface:** noteworthy

The round-1 blocker (install could write YAML the harness rejects) and the round-2 blocker (uninstalling a newline-flagged block that was not last) are both fixed, and both have regression tests. In the auditor's multi-project, multi-order probe, every file Hero writes parses in the pinned harness's js-yaml. One cosmetic byte-exactness gap remains (see Audit notes).

## Acceptance criteria
- [✓] AC-1: one entry per project, with id `hero-<name≤20>-<hash6>-mcp`, an absolute command and a pinned root. Evidence: `internal/install/deepseek_home.go:55-81,161-177`. Tests: `TestDeepSeekHomeEntryCreateAndNoop`, `TestDeepSeekServerNameShapeAndCase`.
- [~] AC-2: foreign bytes preserved, repeat install a no-op.
  - Markers match only as whole lines, with CRLF tolerated.
  - An added newline is flagged. On removal it is stripped only when the block is at end of file, handed to the next Hero block, or kept when foreign content follows (`:223-245`).
  - Tests: `TestDeepSeekHomeEntryPreservesForeignContent`, `TestDeepSeekHomeEntryRestoresMissingTrailingNewline`, `TestDeepSeekHomeMarkersAreWholeLines`, `TestDeepSeekHomeAddedNewlineRemovalKeepsFileValid`.
  - The auditor probe restored the exact original in every clean scenario except one. When an earlier project is reinstalled with a different binary while a later block exists, and the file had no trailing newline, one extra trailing `\n` is left behind. The file stays valid. See Audit notes.
- [✓] AC-3: projects stay independent in every probed order (`TestDeepSeekHomeEntryTwoProjects`, `TestDeepSeekHomeAddedNewlineRemovalKeepsFileValid`). After every step, `InspectDeepSeekRegistration` reported every still-installed project as registered.
- [✓] AC-4: refusal before any mutation, and dry-run is safe. `PreflightDeepSeekHome` is a dry-run of the upsert, run from `planDeepSeek` at the top of `install.Run`. The upsert re-parses and requires exactly +1 element carrying Hero's id. Flow `[]`, an indented list and a `...` end are refused with the file unchanged. Tests: `TestDeepSeekHomePatchUnextendableLayoutsRefused`, `TestDeepSeekHomePatchRefusedBeforeMutation`, `TestDeepSeekWorkspaceCollisionFailsBeforeRootInstall`.
- [✓] AC-5: the PATH binary and `os.Executable()` are both checked by `transientExecutable` (go-build, `os.TempDir()`, and the macOS `/tmp` and `/var/folders` aliases, raw and resolved). The env pin is an explicit absolute override. Test: `TestDeepSeekHeroCommandResolution`.
- [✓] AC-6: the legacy overlay is pruned when unmodified and kept with a warning when modified (`TestDeepSeekLegacyOverlayPruning`, `TestDeepSeekLegacyOverlayPreservedAndRemovable`).
- [✓] AC-7: install output names the server and path, the AGENTS.md section describes the naming pattern and the `hero_status` check, there is no `--patch`, and the docs use 6-hex examples.
- [✓] AC-8: doctor reports registered, missing, malformed, stale-command or wrong-root. Roots are compared via `rootIdentity`. Tests: `TestDeepSeekRegistrationProblems`, `TestDoctorDeepSeekRegistrationProblemNeedsRepair`.
- [✓] AC-9: uninstall removes the home entry first (`internal/cli/uninstall.go:353-372`). `RemoveDeepSeekHomeEntry` re-parses and requires -1 element with no remaining id for this server, refusing otherwise; dry-run gets the same check (`deepseek_home.go:383-396`). The file is removed only when Hero created it and it is empty. Tests: `TestDeepSeekUninstallFailsBeforeRemovingFilesOnBadHomePatch`, `TestDeepSeekUninstallOwnershipAndDryRun`, `TestDeepSeekHomeAddedNewlineRemovalKeepsFileValid`.
- [✓] AC-10: `native-compatibility-{engineering,pm,qa}.log` pass against harness `477b4f42`: web and headless composition plus the desktop-path `readProfilePatches` with no overlays, one Hero client, 86 tools, `hero_status` executed.
- [✓] AC-11: across all three commits, the non-DeepSeek code is untouched outside DeepSeek branches. `go-tests.log` has 0 failures. The auditor re-ran `go test ./internal/install ./internal/cli -count=1` on `3deea556`, and both pass. The 23 DeepSeek install tests pass individually.

## Changes
- [✓] 1. Home-patch upsert and removal (both validated by re-parsing), naming, binary resolution.
- [✓] 2. Ownership via the created-header line (deviation disclosed in the ledger); legacy prune.
- [✓] 3. `agents_md.go` DeepSeek section.
- [✓] 4. `doctor.go` and `uninstall.go`, with tests.
- [✓] 5. `scripts/deepseek-compatibility.mjs`.
- [✓] 6. Docs (`docs-tests.log` and `docs-build.log` pass; no docs changed in round 3).
- [✓] 7. Supersede note on `deepseek-harness-install-target`.

## Open items
- None. The ledger has no PARTIAL, SKIPPED or BLOCKED rows.

## Harness parse check (auditor probe)
`deepseek_home.go` at `3deea556` was copied into a scratch program.

**Coverage.** 17 starting layouts × 11 scenarios. The scenarios cover:
- two projects removed in either order;
- three projects removed middle-first and first-first;
- reinstall with a changed binary, for the first and for the later project;
- remove then re-add;
- user appends after Hero's block, with one and two projects.

That produced 738 intermediate files. Each was loaded with the pinned harness's `js-yaml` using the `parsePatchList` rule (a top-level array of mappings).

**Results:**
- 722 files parse as valid patch lists.
- The other 16 are exact restorations of originals that the harness already rejected before Hero touched them: an empty file, or a comments-only file.
- 0 invalid outputs.
- No project that was still installed was ever reported unregistered.
- Unextendable layouts were refused with the file unchanged.

## Audit notes
- **Cosmetic, non-blocking.** Take a home patch with no trailing newline and two installed projects. Reinstall the first with a different `hero` binary path; the newline flag has already moved to the second block. Then uninstall both. The file ends with one extra `\n` (for example `- insert: []` becomes `- insert: []\n`). It stays valid, and all entries are preserved. Only the missing-trailing-newline byte is not restored. Reproduced for 4 of the 17 layouts, in the `+A,+B,+A',-B,-A` order only.
- The evidence logs report `v0.34.2-17-gb15cd0a9-dirty`, meaning they were produced on the working tree before commit `3deea556`. The auditor re-ran the install and CLI package tests on the committed `3deea556`, and they pass. `gofmt -l` is clean for the touched files.
- The `HERO_DEEPSEEK_MCP_COMMAND` pin is not checked for temporary paths, by design (developer override).
