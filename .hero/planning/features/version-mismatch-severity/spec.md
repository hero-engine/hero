---
title: Version mismatch severity
slug: version-mismatch-severity
type: feature
status: planning
priority: P1
tags: [cli, version, reliability]
horizon: now
smoke: false
delivery_method: supervised
created: 2026-05-12
---

# Version mismatch severity

## Context

The codebase already has `CompareVersions()` and direction-aware `Mismatch()` in `internal/version/version.go` (recently restored from a revert). `Mismatch()` currently returns a plain `string` — a warning message if versions differ, or empty string if they match. `PersistentPreRun` in `internal/cli/root.go` calls `Mismatch()` and prints any non-empty result to stderr, but never blocks execution regardless of how severe the mismatch is.

This means a major version mismatch (e.g., 1.x binary on a 2.x workspace) silently allows commands to run, which can cause data corruption or unexpected behavior when the binary and workspace schemas diverge significantly.

Related: `.hero/specs/upgrade-downgrade-workspace-churn/spec.md` (completed) — established the direction-aware mismatch messaging.

## Goal

Change `Mismatch()` to return a typed result with severity (none/warning/error) so that minor/patch version differences produce a stderr warning but allow execution to continue, while major version differences block all commands (except exempt ones) and exit non-zero with a clear message telling the user to update the binary.

## Kickoff

Change `Mismatch()` in `internal/version/version.go` to return a `MismatchResult` struct with `Severity` (none/warning/error) and `Message` fields instead of a bare string. Severity is `error` when the major version component differs, `warning` when only minor/patch differ, and `none` when versions match or checks are skipped. Update `PersistentPreRun` in `internal/cli/root.go` to exit non-zero on error severity (keeping the existing exempt-command list: init, install, trust, upgrade, mcp, version, help, scan). Update all tests in `internal/version/version_test.go` to assert on `MismatchResult.Severity` and `MismatchResult.Message`. Dev builds ("dev") and empty versions continue to skip all checks.

## Approach

### Design decisions

1. **Typed return over sentinel values.** `Mismatch()` returns a `MismatchResult` struct rather than changing the string return to use sentinel values. This is explicit, testable, and extensible if more severity levels are needed later.

2. **Severity determined by major component only.** If the major version number differs between binary and workspace, severity is `error`. If major matches but minor or patch differ, severity is `warning`. This is simple and aligns with semver conventions — major changes signal breaking incompatibility.

3. **Exempt commands unchanged except for `scan`.** The existing exempt list in `PersistentPreRun` (init, install, trust, upgrade, mcp, version, help) is preserved. `scan` is added per requirements — it should be usable even with a major mismatch so users can assess workspace state before upgrading.

4. **No behavioral change for dev builds.** Binary version `"dev"` or `""` continues to return `SeverityNone` with an empty message, exactly as today.

5. **No behavioral change when version file is missing.** Pre-version workspaces (no `version.json`) continue to return `SeverityNone` — no warning, no block.

### Severity determination logic

```
if binaryVersion == "" or "dev"          → none
if workspace version == "" or "dev"      → none
if binaryVersion == workspace version    → none
if major(binary) != major(workspace)     → error
otherwise (minor or patch differs)       → warning
```

## Changes

1. **Add `MismatchResult` type and severity constants to `internal/version/version.go`.**
   - Define `MismatchSeverity` as an `int` type with constants: `SeverityNone`, `SeverityWarning`, `SeverityError`.
   - Define `MismatchResult` struct with `Severity MismatchSeverity` and `Message string` fields.
   - Place these above the `Mismatch()` function, after the existing type definitions.

2. **Rewrite `Mismatch()` in `internal/version/version.go` to return `MismatchResult`.**
   - Change signature from `func Mismatch(heroDir, binaryVersion string) string` to `func Mismatch(heroDir, binaryVersion string) MismatchResult`.
   - Early-return `MismatchResult{Severity: SeverityNone, Message: ""}` for: dev/empty binary version, missing/unreadable version file, dev/empty workspace version, exact version match.
   - After determining versions differ, compare major components using `parseVersionParts()`:
     - If `aParts[0] != bParts[0]` → return `SeverityError` with message: `"workspace is v{wsVer}, binary is v{binVer} — major version mismatch, run 'hero upgrade' before continuing"`.
     - Otherwise → return `SeverityWarning` with the existing direction-aware messages (binary newer suggests `hero upgrade`, binary older says "downgrade detected, no action needed").

3. **Update `PersistentPreRun` in `internal/cli/root.go` to handle severity.**
   - Change the call site from `if msg := version.Mismatch(...); msg != ""` to `result := version.Mismatch(...)` followed by a switch on `result.Severity`.
   - For `SeverityWarning`: print to stderr (same as today).
   - For `SeverityError`: print to stderr and call `os.Exit(1)` to block execution.
   - Add `"scan"` to the exempt command list on line 37.
   - For `SeverityNone`: do nothing (implicit).

4. **Update tests in `internal/version/version_test.go`.**
   - `TestMismatch_Match`: assert `result.Severity == SeverityNone` and `result.Message == ""`.
   - `TestMismatch_Different`: assert `result.Severity == SeverityWarning` (0.2.0 vs 0.3.0 is minor diff) and message contains `"hero upgrade"`.
   - `TestMismatch_DevBuild`: assert `SeverityNone` for both `"dev"` and `""`.
   - `TestMismatch_NoVersionFile`: assert `SeverityNone`.
   - `TestMismatch_BinaryNewer`: assert `SeverityWarning` and message contains `"hero upgrade"`.
   - `TestMismatch_BinaryOlder`: assert `SeverityWarning` and message contains `"downgrade detected"`.
   - Add `TestMismatch_MajorVersionMismatch`: workspace `1.0.0`, binary `2.0.0` → assert `SeverityError` and message contains `"major version mismatch"`.
   - Add `TestMismatch_MajorVersionMismatch_BinaryOlder`: workspace `2.0.0`, binary `1.5.0` → assert `SeverityError` and message contains `"major version mismatch"`.
   - Add `TestMismatch_PatchDifference`: workspace `0.2.0`, binary `0.2.3` → assert `SeverityWarning`.

5. **Add CLI-level integration test (if integration test infrastructure exists).**
   - Test that a major mismatch causes `hero status` (or another non-exempt command) to exit non-zero.
   - Test that `hero version` and `hero upgrade` still work during a major mismatch (exempt commands).
   - If no integration test framework exists, skip this and rely on manual verification.

## Boundaries

- Does NOT change the `upgrade` command's existing downgrade-rejection logic.
- Does NOT change how `version.json` is written or stamped.
- Does NOT add any new commands or flags.
- Does NOT change behavior for dev builds or pre-version workspaces.
- Does NOT address version mismatches for subcommands that bypass `PersistentPreRun` (if any exist — none are known).

## Risks

1. **Breaking change for scripts.** Any external script that parses stderr output from `Mismatch()` will see a slightly different message format for major mismatches. The direction-aware messages for minor/patch differences are unchanged.

2. **`os.Exit(1)` in `PersistentPreRun`.** Calling `os.Exit()` in a cobra `PersistentPreRun` is abrupt but is the correct pattern for blocking all subcommands. The exempt list ensures users can still run `hero upgrade` to fix the problem.

3. **Test migration.** All existing `Mismatch()` tests must be updated to use the new return type. Missing one will cause a compile error, so the compiler will catch this.

## Validation

- `go test ./internal/version/` — all existing tests pass with updated assertions, plus new major-mismatch and patch-difference tests.
- `go build ./...` — compiles without errors (all callers updated).
- Manual verification:
  - Create a workspace with `hero init`, note the version in `.hero/version.json`.
  - Simulate a major mismatch by editing `version.json` to a different major version.
  - Run `hero status` — should print error to stderr and exit non-zero.
  - Run `hero version` — should still work (exempt).
  - Run `hero upgrade` — should still work (exempt).
  - Simulate a minor mismatch — `hero status` should print warning to stderr but continue normally.

## Acceptance Criteria

- WHEN the binary major version differs from the workspace major version THEN THE SYSTEM SHALL print an error message to stderr and exit non-zero, blocking command execution.
- WHEN the binary minor or patch version differs from the workspace version (major matches) THEN THE SYSTEM SHALL print a warning message to stderr and continue executing the command.
- WHEN the binary version matches the workspace version exactly THEN THE SYSTEM SHALL produce no output and continue normally.
- WHEN the binary version is "dev" or empty THEN THE SYSTEM SHALL skip all version checks and continue normally.
- WHEN a major version mismatch occurs and the user runs an exempt command (init, install, trust, upgrade, mcp, version, help, scan) THEN THE SYSTEM SHALL execute the command without blocking.
- THE SYSTEM SHALL determine severity by comparing only the major version component — major difference means error, otherwise warning.
- THE SYSTEM SHALL return a typed `MismatchResult` with `Severity` and `Message` fields from the `Mismatch()` function.
