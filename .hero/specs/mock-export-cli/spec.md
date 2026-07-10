---
title: Mock Export CLI
slug: mock-export-cli
type: feature
status: completed
domain: engineering
size: medium
created: 2026-06-24
priority: P2
tags: [cli, mocks, export, platform]
relates-to: [knowledge-export-cli]
delivery_method: manual
completed_at: 2026-06-24T23:32:26Z
---

# Mock Export CLI

## Context

Hero now has a completed `hero export knowledge <destination>` command that copies `.hero/knowledge/**` with safe, explicit conflict handling. Mock artifacts under `.hero/mocks/**` are the same kind of local-first handoff asset: they help the next agent or teammate understand intended UI behavior, but today sharing them requires manual directory copies.

This follow-up applies the completed export pattern to mocks. It should reuse the existing export semantics where possible, while respecting that mocks are mostly generated HTML, screenshots, and assets rather than semantically mergeable knowledge entries.

Inspection of `internal/knowledge/export.go`, `internal/cli/export.go`, and `internal/cli/mock.go` shows this is `medium`, not `small`: the CLI addition is small, but safely avoiding duplicated export/conflict machinery requires a narrow refactor of the existing knowledge-named exporter plus focused tests for unsupported mock merge behavior.

## Goal

Add a Hero CLI command that exports the current workspace's `.hero/mocks/**` tree to a caller-provided destination path, preserving the mock tree shape and using the same conflict strategy vocabulary as knowledge export: `fail`, `skip`, `overwrite`, `merge`, and `interactive`. `merge` must remain accepted for CLI parity, but when a mock artifact conflicts it must fail clearly because generated mock artifacts have no safe semantic merge.

## Kickoff

Adds `hero export mocks <destination>` by reusing the knowledge export conflict engine without letting mock artifacts use markdown merge semantics.

**Status:** completed — implementation and verification are finished.

**Pick up at:** inspect `internal/cli/export.go` and `internal/knowledge/export.go` for the completed mock export command and shared export core.

→ `.hero/planning/features/mock-export-cli/spec.md`

**Files:** `internal/knowledge/export.go`, `internal/cli/export.go`, `internal/cli/export_test.go`, `internal/cli/mock.go`, `internal/config/config.go`

## Problem

Mock artifacts can include `index.html`, generated CSS/JS, images, screenshots, and renderer-specific files. The existing mock listing command in `internal/cli/mock.go` intentionally filters to mock directories with `index.html`, but export must copy the raw `.hero/mocks/**` tree exactly, including nested assets and files that are not discoverable mock entries.

The sharp edge is conflict handling. Knowledge export supports deterministic markdown merge for compatible knowledge entries. Mocks do not have comparable semantics: even HTML files are generated artifacts, and merging screenshots or assets is unsafe. The command still needs to accept `--conflict merge` and offer `merge` in interactive mode so the export surface stays consistent, but choosing merge for a differing mock artifact must produce a path-specific unsupported-merge error instead of attempting a markdown merge or silently falling back to another strategy.

## Approach

Expose `hero export mocks <destination>` under the existing `export` umbrella in `internal/cli/export.go`. The destination path represents the exported mocks root: `.hero/mocks/foo/index.html` writes to `<destination>/foo/index.html`, not `<destination>/.hero/mocks/foo/index.html`.

Reuse the completed knowledge export behavior by extracting or parameterizing only the content-agnostic tree export machinery: tree walking, relative path preservation, symlink/type rejection, byte-identical detection, fail preflight, skip/overwrite behavior, atomic writes, interactive callback handling, conflict errors, and summary counters. Keep knowledge markdown merge as a knowledge-specific merge policy, and add a mocks merge policy that always returns a clear unsupported-merge conflict for differing files. Preserve the current ordering where byte-identical files are detected before the selected conflict strategy, so identical mock files count as `identical` even when `--conflict merge` is selected.

Do not create a broad export plugin framework. This is the second tree export surface, so a small shared helper or policy seam is justified; a generalized registry or dynamic export architecture is not.

## Changes

1. Refactor the reusable export core currently in `internal/knowledge/export.go` without changing existing knowledge behavior.
   - Keep the public `knowledge.Export(srcKnowledgeDir, destination, opts)` API working for `hero export knowledge`.
   - Extract or parameterize the shared filesystem steps: `filepath.WalkDir`, regular file/directory copying, relative path preservation, symlink rejection, destination-inside-source rejection, preflight planning for conflict-detecting strategies, atomic temp-file-and-rename writes, `Summary` counters, and `ConflictError` reporting.
   - Add an artifact label or source description so mock export errors say `mock`/`mocks` instead of `knowledge export` or `source knowledge dir` where user-facing output would otherwise be misleading.
   - Keep the markdown/frontmatter merge helper available only to knowledge export.
   - Ensure the byte-identical fast path runs before any strategy-specific handling for both knowledge and mocks.

2. Add mock-specific merge handling in the shared export path.
   - For mock export, accept `ConflictMerge` as a valid strategy value.
   - If source and destination bytes differ, return a conflict/error reason such as `merge is not supported for mock artifacts` or `unsupported merge for mock artifact` with the relative path.
   - Do not invoke `mergeMarkdown` for any mock export file, including `.html`, `.md`, or `.markdown` files under `.hero/mocks/**`.
   - In interactive mode, if the prompt callback or terminal selection returns `merge` for a conflicting mock file, fail with the same unsupported-merge error rather than re-prompting indefinitely or converting the choice to `skip`/`overwrite`.

3. Add the CLI surface in `internal/cli/export.go`.
   - Define `exportMocksCmd` as `mocks <destination>` under the existing `exportCmd` umbrella.
   - Add the same `--conflict fail|skip|overwrite|merge|interactive` vocabulary and default `fail` used by `exportKnowledgeCmd`.
   - In `runExportMocks`, call `findProjectRoot()`, `config.Load(projectRoot)`, and `cfg.MocksDir(projectRoot)` from `internal/config/config.go`.
   - Reuse the existing terminal guard and prompt vocabulary for `--conflict interactive`; the prompt should offer `fail`, `skip`, `overwrite`, and `merge`.
   - Print a concise summary equivalent to knowledge export, using a mocks-specific destination label and counts for copied, skipped, overwritten, merged, identical, and conflicts. For mocks, `merged` is expected to stay `0` unless future work adds safe semantics.

4. Keep `internal/cli/mock.go` unchanged except for any documentation comments needed to avoid confusion.
   - Do not use `discoverMocks` for export because it only returns directories containing `index.html`.
   - Export must walk `Config.MocksDir(projectRoot)` directly and preserve every regular file under `.hero/mocks/**`, including nested assets, screenshots, and renderer-specific files.

5. Update CLI test support in `internal/cli/helpers_test.go` if new flag variables are introduced.
   - If knowledge and mocks use separate conflict flag variables, reset both to `fail` in `resetFlags()`.
   - If the existing `exportConflict` variable remains shared, keep the existing reset and add regression coverage that knowledge and mocks invocations do not leak flag state.

6. Add core export tests in `internal/knowledge/export_test.go` or the new shared exporter package's test file if the core is extracted.
   - Preserve nested mock tree shape with representative HTML, nested asset, and screenshot fixture files.
   - Verify default `fail` reports all differing mock conflicts and writes nothing for planned conflict-detecting runs.
   - Verify `skip` leaves existing destination files unchanged and copies only missing mock files.
   - Verify `overwrite` atomically replaces differing destination mock files and does not prune extra destination files.
   - Verify `merge` accepts the strategy but fails clearly for differing mock files.
   - Verify `merge` counts byte-identical mock files as identical and does not report them as conflicts.
   - Verify interactive callback selection of `merge` on a differing mock file returns the same unsupported-merge error.
   - Verify symlinks, file/directory mismatches, and destination-inside-source remain rejected for mocks.

7. Add CLI tests in `internal/cli/export_test.go`.
   - Exercise `hero export mocks <destination>` from `newTestEnv` using `.hero/mocks/**` fixtures.
   - Exercise `--conflict skip`, `--conflict overwrite`, `--conflict merge`, and `--conflict interactive` routing for mocks.
   - Verify invalid `--conflict` values and missing source mocks directory errors are user-readable and mocks-specific.
   - Verify `--conflict interactive` rejects non-interactive stdin/stdout before writing conflicting mock files.
   - Verify command output includes the resolved destination and summary counts.

## Acceptance Criteria

- WHEN an engineer runs `hero export mocks <destination>` THE SYSTEM SHALL copy every regular file under `.hero/mocks/**` to the matching relative path under `<destination>`.
  verified_by: internal/cli/export_test.go::TestExportMocksCommandCopiesTreeAndPrintsSummary
- WHEN the destination does not exist THE SYSTEM SHALL create the destination directory and any needed child directories.
  verified_by: internal/knowledge/export_test.go::TestExportMocksPreservesTreeShape
- WHEN a destination mock file already exists with different bytes and no `--conflict` flag is provided THE SYSTEM SHALL fail before writing any files and print the conflicting relative paths.
  verified_by: internal/knowledge/export_test.go::TestExportMocksConflictStrategies/fail_reports_all_conflicts_and_writes_nothing
- WHEN `--conflict skip` is provided THE SYSTEM SHALL leave existing destination paths unchanged and copy only missing source mock files.
  verified_by: internal/knowledge/export_test.go::TestExportMocksConflictStrategies/skip_leaves_existing_and_copies_missing
- WHEN `--conflict overwrite` is provided THE SYSTEM SHALL atomically replace conflicting destination mock files without deleting unrelated extra files in the destination tree.
  verified_by: internal/knowledge/export_test.go::TestExportMocksConflictStrategies/overwrite_replaces_conflicting_files_and_keeps_extras
- WHEN `--conflict merge` is provided for byte-identical source and destination mock files THE SYSTEM SHALL count those files as identical and continue without reporting a merge conflict.
  verified_by: internal/knowledge/export_test.go::TestExportMocksMergeBehavior/identical_files_count_as_identical
- IF `--conflict merge` encounters a differing mock artifact THEN THE SYSTEM SHALL fail with a path-specific error that clearly says merge is unsupported for mock artifacts.
  verified_by: internal/knowledge/export_test.go::TestExportMocksMergeBehavior/differing_artifacts_fail_clearly
- WHEN `--conflict interactive` is provided in an attached terminal THE SYSTEM SHALL prompt for each mock conflict with `fail`, `skip`, `overwrite`, and `merge` choices.
  verified_by: internal/cli/export.go::promptConflictStrategy
- IF an interactive user chooses `merge` for a differing mock artifact THEN THE SYSTEM SHALL surface the same unsupported-merge error used by non-interactive `--conflict merge`.
  verified_by: internal/knowledge/export_test.go::TestExportMocksMergeBehavior/interactive_merge_choice_returns_unsupported_merge
- IF `--conflict interactive` is provided without an attached terminal THEN THE SYSTEM SHALL fail before writing conflicting files with a clear non-interactive error.
  verified_by: internal/cli/export_test.go::TestExportMocksCommandRejectsInvalidAndInteractiveInputs/interactive_requires_terminal
- IF the source `.hero/mocks` directory is missing THEN THE SYSTEM SHALL fail clearly without creating a partial destination export.
  verified_by: internal/cli/export_test.go::TestExportMocksCommandReportsWorkspaceErrors
- THE SYSTEM SHALL preserve existing `hero export knowledge` behavior, including markdown merge semantics and summary counters.
  verified_by: internal/cli/export_test.go::TestExportKnowledgeCommandConflictStrategies

## Boundaries

- Do not implement semantic, HTML-aware, image-aware, or LLM-assisted mock merging.
- Do not use `internal/cli/mock.go::discoverMocks` as the export source of truth; it intentionally filters the mocks tree.
- Do not export `.hero/knowledge/**`, specs, graph files, queue files, snapshots, or peer manifests as part of this command.
- Do not write destination paths under `<destination>/.hero/mocks`; the destination is the mocks root itself.
- Do not refresh or mutate the destination workspace's index, graph, queue, or mock listing state.
- Do not prune destination files that are absent from the source tree.
- Do not add cloud sync, tracker integration, peer handoff, or bidirectional import/apply behavior.
- Do not introduce a generalized export plugin registry for this feature.

## Risks

- A careless reuse of knowledge export could apply markdown merge semantics to mock artifacts. The implementation must make mock merge unsupported by policy, not by file extension.
- Interactive handling can accidentally loop if `merge` returns a conflict and the code re-prompts. For mocks, selecting `merge` on a differing artifact should terminate with the unsupported-merge error.
- Partial writes can confuse the receiving folder. Keep fail/merge preflight behavior for conflict-detecting strategies and keep atomic per-file writes for overwrite/copy operations.
- Large screenshots or generated assets are copied by reading whole files into memory in the current exporter. This is acceptable for current mock artifacts but should not be extended to large media bundles without revisiting streaming behavior.
- Symlinks may escape the workspace or copy unintended bytes. Preserve the existing symlink rejection behavior.
- Rollback is local: because the source `.hero/mocks/**` tree is never mutated, rollback means deleting the exported destination or restoring overwritten destination files from version control/backups. There is no schema, graph, or migration rollback.
- `internal/cli/export.go` and `internal/knowledge/export.go` are recent platform surfaces from `knowledge-export-cli`; delivery should check active conflicts before implementation.

## Validation

- Run `go test ./internal/knowledge ./internal/cli` after implementation.
- If a shared exporter package is introduced, run its targeted tests as well, for example `go test ./internal/exporttree ./internal/knowledge ./internal/cli`.
- Run targeted tests: `go test ./internal/cli -run Export` and `go test ./internal/knowledge -run Export`.
- Manually verify from a temporary workspace:
  1. Create a demo mock with an index file, nested asset file, and screenshot file under `.hero/mocks/demo/`.
  2. Run `hero export mocks /tmp/hero-mocks-export` and confirm the destination contains the same demo index, nested asset, and screenshot relative paths.
  3. Re-run without flags and confirm identical files count as identical.
  4. Change one destination file and confirm default `fail` reports the relative path before writes.
  5. Re-run with `--conflict skip`, `--conflict overwrite`, and `--conflict merge`; confirm merge fails clearly only for the differing artifact while identical files remain identical.
  6. Re-run with `--conflict interactive`, choose `merge` for a differing artifact, and confirm the unsupported-merge error is surfaced.

## Completion Ledger

- Refactored `internal/knowledge/export.go` to route knowledge and mock tree export through a shared filesystem planner while keeping markdown merge knowledge-specific.
- Added `knowledge.ExportMocks` with mocks-specific source/error labels and unsupported merge behavior for differing mock artifacts.
- Added `hero export mocks <destination>` with the same conflict vocabulary, terminal guard, and summary counters as knowledge export.
- Covered mock export tree preservation, fail/skip/overwrite/merge behavior, unsupported interactive merge, unsafe paths, missing source, and CLI routing in tests.
- Verified existing knowledge export behavior still passes with `go test ./internal/knowledge ./internal/cli`.
