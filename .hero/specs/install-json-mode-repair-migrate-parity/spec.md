---
title: "`hero install --repair`/`--migrate` ignore `--json` — stdout contract broken for programmatic consumers"
slug: install-json-mode-repair-migrate-parity
type: bug
status: completed
created: 2026-07-06
tags: [install, json, cli-contract]
completed_at: 2026-07-10T06:21:02Z
---
# `hero install --repair`/`--migrate` ignore `--json`

## Problem

`hero install --json` documents a contract: exactly one JSON result object
(`install.InstallJSONOutput`) on stdout, with an `error` field on failure, so
programmatic consumers (Hero-native clients) can subprocess-invoke and parse.
Two install entry points violate it:

- **`--repair --json`**: the repair short-circuit in `runInstall`
  (internal/cli/install.go, `if installRepair` block) prints human progress
  ("Repairing satellites for workspace at ...") to stdout and delegates to
  `runSatelliteRepair`, which emits its own human report. No JSON object is
  ever produced — on success or failure.
- **`--migrate --json`**: the migrate block prints its two "Note: `--migrate`
  is now equivalent..." lines to stdout *before* falling through to the
  regular install body. The install body then emits proper JSON, but stdout
  now contains note text + JSON — not a single parseable object. The
  `--migrate` detect-first-target failure path also returns an error without
  emitting any JSON.

Exit codes are correct (errors propagate to `cmd/hero/main.go` which exits 1);
the breakage is the stdout contract only.

This is the same class of bug as the satellite short-circuit fixed on
2026-07-06 (satellite path in `runInstall` bypassed all `--json` handling; see
`TestInstallJSON_SatelliteFailureEmitsJSONAndErrors` in
internal/cli/install_json_test.go). `--repair` and `--migrate` are the two
remaining pre-JSON short-circuits.

## Steps to Reproduce

```sh
# In a workspace with a satellite:
hero install project . --repair --json     # → human text on stdout, no JSON
# In a project with an installed harness:
hero install project . --migrate --json    # → note lines precede the JSON object
```

Pipe either through `jq .` to see the parse failure.

## Expected Behavior

With `--json`, both flags emit exactly one `InstallJSONOutput` object on
stdout — `error` populated (code `install_failed`, or a mode-specific stable
code) and nonzero exit on failure; no interactive prompts; no stray human
text on stdout (cobra usage already silenced via `cmd.SilenceUsage` when
`installJSON` is set).

## Root Cause

Both blocks are short-circuits at the top of `runInstall` that predate the
`--json` flag; the JSON handling (silence + `emitJSON`) lives only in the
main install body, so any path that returns before reaching it skips the
contract entirely.

## Fix

Reuse the pattern established for the satellite path: wrap the short-circuit
body in `silenceStdout`, then `emitJSON(install.InstallJSONOutput{...}, err)`.
Consider extracting a small helper (`runJSONWrapped(mode, targetDir, fn)`)
since this is now three call sites with identical shape. Suppress the
`--migrate` note lines in JSON mode (or route them to stderr).

## Changes

- internal/cli/install.go — `installRepair` and `installMigrate` blocks in
  `runInstall`; possibly a shared JSON-wrapping helper.
- internal/cli/install_json_test.go — regression tests: `--repair --json` and
  `--migrate --json`, asserting single-object stdout parse, `error` field on
  failure, and non-nil returned error.

## Completion Ledger

| AC / Change | Status | Note |
|-------------|--------|------|
| `--repair --json` emits one `InstallJSONOutput` (no human text) | DONE | both repair returns (workspace-locate failure + delegate to `runSatelliteRepair`) wrapped in `emitInstallJSON`; code `repair_failed`; `TestInstallJSON_RepairEmitsJSONAndErrors` |
| `--migrate --json` emits one object; note lines off stdout | DONE | note routed to stderr in JSON mode; both migrate early-error returns (mode/target validation + detect-first-target failure) wrapped; code `migrate_failed`; `TestInstallJSON_MigrateEmitsJSONAndErrors` |
| Nonzero exit preserved on failure | DONE | `emitInstallJSON` returns fn's error unchanged → `cmd/hero/main.go` exits 1; tests assert non-nil returned error |
| Shared JSON-wrapping helper (3 call sites) | DONE | extracted `emitInstallJSON(mode, targetDir, version, code, fn)`; satellite path refactored onto it too (behavior-preserving — `TestInstallJSON_SatelliteFailureEmitsJSONAndErrors` still green) |
| Test-isolation fix | DONE | `resetFlags` now resets `installRepair`/`installMigrate` (cobra persists bool flags across in-process runs) |
| `go build ./... && go test ./...` | DONE | 86 packages ok, 0 failed |

- [x] exercise-the-feature: driven through the real cobra command via `rootCmd.Execute()` (runCmd); a proof harness confirmed stdout for `--repair --json` is a single `{...}` parsing to `Error.Code="repair_failed"` with a non-nil returned error. NOTE: standalone-binary manual exec was blocked by the session sandbox (it refuses to exec any freshly-built Go binary; the only PATH `hero` is a stale Homebrew v0.8.1), so behavioral verification is via the in-process command tests, which drive the identical `runInstall` path.

## Judgment calls

- **Mode-specific error codes.** The spec allowed `install_failed` "or a
  mode-specific stable code". Chose `repair_failed` / `migrate_failed` so
  programmatic consumers can distinguish which mode failed. The satellite path
  keeps `install_failed` (unchanged contract for that surface).
- **Migrate note → stderr** (not suppressed): preserves the human breadcrumb for
  interactive `--json` users watching stderr, while keeping stdout a pure single
  object.
