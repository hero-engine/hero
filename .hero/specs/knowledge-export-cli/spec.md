---
title: Knowledge Export CLI
slug: knowledge-export-cli
type: feature
status: completed
domain: engineering
size: medium
created: 2026-06-24
priority: P2
tags: [cli, knowledge, export, platform]
claimed_by: chet-bellows
claimed_at: 2026-06-24T10:32:40-07:00
delivery_method: manual
completed_at: 2026-06-24T19:03:04Z
---

# Knowledge Export CLI

## Context

Hero's knowledge corpus is valuable because it makes the next agent session start smarter than the last one ended. Today the corpus is easy to query locally through `hero ask`, `hero search`, `hero relevant`, and the dashboard, but there is no direct CLI path to hand a workspace's `.hero/knowledge/**` tree to another engineer. Engineers can copy the directory manually, but manual copies do not give a safe default for collisions or a consistent way to converge with a destination that already has entries.

This is platform/tooling work in the Hero CLI. It should stay local-first: export filesystem knowledge files without introducing cloud sync, graph reconciliation, or tracker behavior.

## Goal

Add a Hero CLI command that exports the current workspace's `.hero/knowledge/**` contents to a caller-provided destination path, preserving the knowledge tree shape and using an explicit conflict strategy. The default strategy must fail safely before writing conflicting files. Operators must also be able to skip existing destination files, overwrite destination files, merge compatible existing markdown knowledge entries, or interactively choose a strategy for each conflict.

## Kickoff

Adds `hero export knowledge <destination>` so engineers can hand off `.hero/knowledge/**` with safe conflict handling.

**Status:** completed — verified and archived after `hero spec verify knowledge-export-cli` passed all gates.

**Pick up at:** review `internal/knowledge/export.go` and `internal/cli/export.go` if extending export beyond knowledge.

→ `.hero/specs/knowledge-export-cli/spec.md`

**Files:** `internal/knowledge/export.go`, `internal/knowledge/export_test.go`, `internal/cli/export.go`, `internal/cli/export_test.go`, `internal/cli/root.go`

## Problem

Hero knowledge entries are files under `.hero/knowledge/` with multiple shapes: flat markdown files, directory-backed `spec.md` entries, raw imported artifacts, code intelligence cache files, and generated session records. The export command must copy the whole tree without assuming every entry is a parsed spec. The riskiest part is destination convergence: a destination may already contain human-edited entries, generated code knowledge, or raw files. The command needs a default that cannot silently corrupt the destination, plus explicit strategies for users who choose a different convergence behavior.

## Approach

Expose a new CLI surface as `hero export knowledge <destination>` with a `--conflict` flag accepting `fail`, `skip`, `overwrite`, `merge`, or `interactive`; default to `fail`. Use an `export` umbrella because future one-way exports may include specs, snapshots, or graph bundles, while this feature only implements the `knowledge` subcommand.

The destination path represents the exported knowledge root. For example, `.hero/knowledge/conventions/foo.md` writes to `<destination>/conventions/foo.md`, not `<destination>/.hero/knowledge/conventions/foo.md`. This lets another engineer pass their workspace's `.hero/knowledge` directory directly, or pass any standalone folder they want to archive or inspect.

Keep the CLI thin and put reusable filesystem behavior in `internal/knowledge`. The exporter should walk `config.Config.KnowledgeDir(projectRoot)` with `filepath.WalkDir`, copy regular files and directories, and preserve relative paths exactly. Do not use `spec.Discover` because it misses flat markdown files, raw artifacts, and code intelligence files.

Conflict strategies:

- `fail`: preflight the whole tree; if any destination path exists with differing bytes or incompatible file type, report all conflicts and write nothing. Identical files are not conflicts.
- `skip`: leave every existing destination path untouched; copy only missing paths and report skipped counts.
- `overwrite`: replace conflicting regular files atomically; create missing directories; do not delete extra destination files.
- `merge`: merge only compatible markdown knowledge files. Parse YAML frontmatter when present; union list fields such as `tags`, `scope`, and `triggers`; copy source scalar fields only when the destination field is missing; fail on scalar field conflicts. Merge markdown bodies by preserving destination sections and appending source top-level sections that are absent. If a conflicting non-markdown file, directory/file type mismatch, malformed merge candidate, or unsafe symlink is encountered, fail with a clear path-specific error rather than guessing.
- `interactive`: when a conflict is encountered in an attached terminal, prompt for that path and let the operator choose `fail`, `skip`, `overwrite`, or `merge`. Apply the selected strategy only to that conflict. If stdin is non-interactive, fail with a clear message that `--conflict interactive` requires a terminal.

All writes should use a temp file in the destination directory followed by atomic rename. The command never mutates the source `.hero/knowledge` tree and does not refresh the destination workspace's index or graph; the receiver can run `hero index` in their workspace if needed.

## Changes

1. Add reusable export logic in `internal/knowledge/export.go`.
   - Define `ConflictStrategy` constants for `fail`, `skip`, `overwrite`, `merge`, and `interactive`.
   - Define `Options` with the selected strategy and `Summary` counters for copied, skipped, overwritten, merged, identical, and conflicts.
   - Implement `Export(srcKnowledgeDir, destination string, opts Options) (*Summary, error)`.
   - Resolve absolute source and destination paths and reject any destination nested inside the source knowledge directory.
   - Walk the source with `filepath.WalkDir`, preserve relative paths, copy only regular files/directories, and fail on symlinks or unsupported filesystem entries.
   - Preflight `fail` and `merge` enough to avoid writing after a known conflict; for `overwrite`, protect each individual write with temp-file-and-rename so existing files stay intact if that file write fails.
   - For `interactive`, accept a prompt callback in `Options` so the core package can stay testable and CLI-free while the Cobra command handles terminal I/O.

2. Implement deterministic markdown merge helpers in `internal/knowledge/export.go`.
   - Parse leading YAML frontmatter using the same dependency already used elsewhere (`gopkg.in/yaml.v3`) without adding a new dependency.
   - Preserve destination scalar fields when present; copy missing source scalar fields; union known list fields in stable destination-first order.
   - Preserve destination markdown body sections; append source sections whose top-level headings are absent from the destination.
   - Return a conflict error for scalar frontmatter conflicts, non-markdown conflicts, invalid merge candidates, and directory/file type mismatches.

3. Add the Cobra command in `internal/cli/export.go`.
   - Define `exportCmd` as the top-level umbrella and `exportKnowledgeCmd` as `knowledge <destination>`.
   - Add `--conflict fail|skip|overwrite|merge|interactive` with default `fail`.
   - In `runExportKnowledge`, call `findProjectRoot()`, `config.Load(projectRoot)`, `cfg.KnowledgeDir(projectRoot)`, and `knowledge.Export(...)`.
   - When `--conflict interactive` is selected, require an attached stdin/stdout terminal, print the relative conflict path plus concise source/destination facts, and prompt for `fail`, `skip`, `overwrite`, or `merge`.
   - Print a concise summary including destination path and counts for copied, skipped, overwritten, merged, identical, and conflicts.
   - Return non-zero with path-specific errors for invalid strategy, missing workspace, missing source knowledge directory, unsafe destination, or unresolved conflicts.

4. Register the command and test flag reset behavior.
   - Add `rootCmd.AddCommand(exportCmd)` in `internal/cli/root.go` near other top-level command registrations.
   - Add the export conflict flag state to `resetFlags()` in `internal/cli/helpers_test.go` so CLI tests do not leak flag values across runs.

5. Add core package tests in `internal/knowledge/export_test.go`.
   - Preserve tree shape for flat markdown, directory-backed `spec.md`, raw files, and generated-ish files such as `code/.checksums.json`.
   - Verify default `fail` reports all conflicting paths and writes nothing.
   - Verify `skip` leaves existing destination files untouched and copies missing files.
   - Verify `overwrite` replaces conflicting files atomically without pruning extra destination files.
   - Verify `merge` combines compatible markdown/frontmatter entries deterministically and reports each merged file.
   - Verify `interactive` applies callback-selected strategies per conflict and stops immediately when the callback selects `fail`.
   - Verify `merge` fails on conflicting non-markdown files, scalar frontmatter conflicts, symlinks, file/directory mismatches, and destination-inside-source.

6. Add CLI tests in `internal/cli/export_test.go`.
   - Exercise `hero export knowledge <dest>` from `newTestEnv`.
   - Exercise `--conflict skip`, `--conflict overwrite`, `--conflict merge`, and `--conflict interactive` routing into the package behavior.
   - Verify `--conflict interactive` rejects non-interactive stdin with a clear error.
   - Verify invalid `--conflict` values and missing workspace/source errors are user-readable.
   - Verify command output includes the resolved destination and summary counts.

## Acceptance Criteria

- WHEN an engineer runs `hero export knowledge <destination>` THE SYSTEM SHALL copy every regular file under `.hero/knowledge/**` to the matching relative path under `<destination>`.
  verified_by: internal/knowledge/export_test.go::TestExportPreservesKnowledgeTreeShape
- WHEN the destination does not exist THE SYSTEM SHALL create the destination directory and any needed child directories.
  verified_by: internal/knowledge/export_test.go::TestExportCreatesDestinationDirectoryAndNeededChildDirectories
- WHEN a destination file already exists with different bytes and no `--conflict` flag is provided THE SYSTEM SHALL fail before writing any files and print the conflicting relative paths.
  verified_by: internal/knowledge/export_test.go::TestExportFailReportsConflictsAndWritesNothing
- WHEN `--conflict skip` is provided THE SYSTEM SHALL leave existing destination paths unchanged and copy only missing source files.
  verified_by: internal/knowledge/export_test.go::TestExportSkipLeavesExistingAndCopiesMissing
- WHEN `--conflict overwrite` is provided THE SYSTEM SHALL atomically replace conflicting destination files without deleting unrelated extra files in the destination tree.
  verified_by: internal/knowledge/export_test.go::TestExportOverwriteReplacesFilesAndKeepsExtras
- WHEN `--conflict merge` is provided and an existing markdown knowledge entry is compatible THE SYSTEM SHALL merge frontmatter and body sections deterministically while preserving destination-owned content.
  verified_by: internal/knowledge/export_test.go::TestExportMergeMarkdownKnowledgeEntries
- WHEN `--conflict interactive` is provided in an attached terminal THE SYSTEM SHALL prompt for each conflict and apply the selected `fail`, `skip`, `overwrite`, or `merge` behavior to that conflict only.
  verified_by: internal/knowledge/export_test.go::TestExportInteractiveAppliesPerConflictChoices
- IF `--conflict interactive` is provided without an attached terminal THEN THE SYSTEM SHALL fail before writing conflicting files with a clear non-interactive error.
  verified_by: internal/cli/export_test.go::TestExportKnowledgeCommandRejectsInvalidAndInteractiveInputs
- IF `--conflict merge` encounters a non-markdown conflict, scalar frontmatter conflict, symlink, or file/directory type mismatch THEN THE SYSTEM SHALL fail with a path-specific error instead of producing a guessed merge.
  verified_by: internal/knowledge/export_test.go::TestExportRejectsUnsafePathsAndTypes
- IF the destination path is inside the source `.hero/knowledge` directory THEN THE SYSTEM SHALL reject the export before walking files.
  verified_by: internal/knowledge/export_test.go::TestExportRejectsUnsafePathsAndTypes
- THE SYSTEM SHALL never mutate the source `.hero/knowledge/**` tree during export.
  verified_by: internal/knowledge/export_test.go::TestExportNeverMutatesSourceKnowledgeTree
- THE SYSTEM SHALL report copied, skipped, overwritten, merged, identical, and conflicted counts after each attempted export.
  verified_by: internal/cli/export_test.go::TestExportKnowledgeCommandReportsCopiedSkippedOverwrittenMergedIdenticalAndConflictedCounts

## Boundaries

- Do not implement semantic or LLM-assisted knowledge deduplication.
- Do not refresh or mutate the destination workspace's `index.db`, `graph.db`, `.hero/QUEUE.md`, or peer manifest.
- Do not add cloud sync, peer handoff, tracker integration, or cross-repo transport.
- Do not prune destination files that are absent from the source tree.
- Do not exclude `.hero/knowledge/code/` or `.hero/knowledge/raw/`; the request is for `.hero/knowledge/**`.
- Do not add a bidirectional import/apply workflow; this command is one-way export to a filesystem path.

## Risks

- Merge semantics can surprise users if they expect semantic reconciliation. Keep merge constrained, deterministic, and explicit in CLI help/output; fail instead of guessing on ambiguous conflicts.
- Partial writes could leave destination state confusing. Use preflight for conflict-detecting strategies and atomic per-file writes for strategies that intentionally modify existing files.
- Symlinks can escape the workspace or copy unintended bytes. V1 should fail on symlinks with a clear error.
- Large generated code knowledge trees may make exports noisy or slow, but excluding them would violate the requested `.hero/knowledge/**` behavior.
- Rollback is local: because the source tree is never mutated, rollback means deleting the exported destination or restoring overwritten destination files from the receiver's version control/backups. There is no schema, graph, or migration rollback.
- `internal/cli/root.go` is touched by other platform work; delivery should check active conflicts before implementation.

## Validation

- Run `go test ./internal/knowledge ./internal/cli` after implementation.
- Run targeted tests: `go test ./internal/knowledge -run Export` and `go test ./internal/cli -run Export`.
- Manually verify from a temporary workspace:
  1. Create sample files under `.hero/knowledge/conventions`, `.hero/knowledge/notes/example/spec.md`, `.hero/knowledge/raw`, and `.hero/knowledge/code`.
  2. Run `hero export knowledge /tmp/hero-knowledge-export` and confirm relative paths are preserved.
  3. Re-run without flags against the same destination and confirm identical files do not fail while changed destination files do fail before writes.
   4. Re-run with each non-interactive conflict strategy and inspect destination contents plus summary counts.
   5. Re-run with `--conflict interactive`, choose different strategies for different conflicts, and confirm each selected behavior applies only to that path.

## Completion Ledger

### Acceptance Criteria

- DONE — AC1: `knowledge.Export` walks the source knowledge dir and copies regular files preserving relative paths; verified by `internal/knowledge/export_test.go::TestExportPreservesKnowledgeTreeShape`.
- DONE — AC2: missing destination and child directories are created; verified by `internal/knowledge/export_test.go::TestExportCreatesDestinationDirectoryAndNeededChildDirectories`.
- DONE — AC3: default `fail` preflights conflicts and returns differing paths without applying plans; verified by `internal/knowledge/export_test.go::TestExportFailReportsConflictsAndWritesNothing`.
- DONE — AC4: `skip` leaves existing destination files untouched and copies missing files; verified by `internal/knowledge/export_test.go::TestExportSkipLeavesExistingAndCopiesMissing`.
- DONE — AC5: `overwrite` writes via temp file plus rename and does not prune destination extras; verified by `internal/knowledge/export_test.go::TestExportOverwriteReplacesFilesAndKeepsExtras`.
- DONE — AC6: `merge` supports deterministic markdown/frontmatter merge preserving destination-owned content; verified by `internal/knowledge/export_test.go::TestExportMergeMarkdownKnowledgeEntries`.
- DONE — AC7: `interactive` applies fail/skip/overwrite/merge per selected conflict; verified by `internal/knowledge/export_test.go::TestExportInteractiveAppliesPerConflictChoices`.
- DONE — AC8: CLI rejects `--conflict interactive` without an attached terminal; verified by `internal/cli/export_test.go::TestExportKnowledgeCommandRejectsInvalidAndInteractiveInputs`.
- DONE — AC9: merge rejects ambiguous conflicts and unsafe filesystem entries; verified by `internal/knowledge/export_test.go::TestExportRejectsUnsafePathsAndTypes`.
- DONE — AC10: destination-inside-source is rejected before walking files; verified by `internal/knowledge/export_test.go::TestExportRejectsUnsafePathsAndTypes`.
- DONE — AC11: source knowledge tree is never mutated; verified by `internal/knowledge/export_test.go::TestExportNeverMutatesSourceKnowledgeTree`.
- DONE — AC12: CLI reports copied, skipped, overwritten, merged, identical, and conflicted counts; verified by `internal/cli/export_test.go::TestExportKnowledgeCommandReportsCopiedSkippedOverwrittenMergedIdenticalAndConflictedCounts`.

### Changes

- DONE — Change1: added reusable export logic in `internal/knowledge/export.go` with strategies, options, summary, validation, walking, symlink rejection, preflight planning, atomic writes, and interactive callback.
- DONE — Change2: implemented deterministic markdown/frontmatter merge helpers in `internal/knowledge/export.go`.
- DONE — Change3: added Cobra command in `internal/cli/export.go` with `hero export knowledge <destination>`, conflict flag, config loading, terminal guard, interactive prompt, and summary output.
- DONE — Change4: registered `exportCmd` in `internal/cli/root.go` and reset `exportConflict` in `internal/cli/helpers_test.go`.
- DONE — Change5: added core package tests in `internal/knowledge/export_test.go`.
- DONE — Change6: added CLI tests in `internal/cli/export_test.go`.

### Validation Notes

- DONE — Hero drift passed: 12/12 acceptance criteria have related code changes and no drift detected.
- DONE — Hero lint passed: 12 EARS criteria.
- DONE — Hero contract coverage passed: 12/12 criteria linked.
- DONE — Hero coverage passed: 12/12 criteria covered, 12 strong.
- DONE — `gofmt` ran on touched Go files and `go test ./internal/knowledge ./internal/cli` passed.
- DONE — Manual CLI exercise passed: `go run ./cmd/hero export knowledge /var/folders/vn/r401wc5925vb6j16lk4mn9rc0000gn/T/opencode/hero-knowledge-export-manual` copied 154 files and produced the expected exported `context/project-overview/spec.md` file.
