---
title: "`hero install --repair`/`--migrate` ignore `--json` — stdout contract broken for programmatic consumers"
slug: install-json-mode-repair-migrate-parity
type: bug
status: planning
created: 2026-07-06
tags: [install, json, cli-contract]
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
