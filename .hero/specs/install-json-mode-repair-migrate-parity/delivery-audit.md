# Delivery audit — install-json-mode-repair-migrate-parity

**Audited:** `git diff HEAD -- internal/cli/` (uncommitted working tree)
**Verdict:** SHIP
**Surface:** clean

Cold audit — verified against on-disk code and a fresh test run, not the
Completion Ledger. Every ledger claim holds under inspection.

## Acceptance criteria

- [✓] `--repair --json` emits exactly one `InstallJSONOutput`, no human text —
  install.go:190-209. Both return points wrapped: workspace-locate failure
  (196-203) and the `runSatelliteRepair` delegate (204-207) call
  `emitInstallJSON(..., "repair_failed", ...)`. The human "Repairing satellites"
  `fmt.Printf` (208) is reachable only in the non-JSON branch. Asserted by
  `TestInstallJSON_RepairEmitsJSONAndErrors` (install_json_test.go:73-93).
- [✓] `--migrate --json` emits one object; note lines off stdout —
  install.go:153-186. The two note lines route to `os.Stderr` in JSON mode
  (164-166) and stdout only in the human branch (167-170). Both early-error
  returns emit JSON: mode/target validation (155-160) and detect-first-target
  failure (178-184), code `migrate_failed`. Success path falls through to the
  main install body unchanged. Asserted by
  `TestInstallJSON_MigrateEmitsJSONAndErrors` (install_json_test.go:102-131).
- [✓] Nonzero exit preserved on failure — `emitInstallJSON` returns fn's error
  unchanged: `emitJSON(payload, err)` returns `originalErr` (install.go:29,
  50). Both regression tests assert `cmdErr != nil`.
- [✓] Shared JSON-wrapping helper across 3 call sites — `emitInstallJSON`
  extracted (install.go:39-51). Satellite path refactored onto it (223-224),
  replacing the previous inline `time.Now()/silenceStdout/emitJSON` block with
  an identical-shape call; keeps code `install_failed`. Behavior-preserving —
  `TestInstallJSON_SatelliteFailureEmitsJSONAndErrors` still passes.
- [✓] Test-isolation fix — `resetFlags` now zeroes `installRepair` and
  `installMigrate` (helpers_test.go:201-202). Without this, cobra's persisted
  bool flags leak across in-process `runCmd` calls.

## Changes

- [✓] internal/cli/install.go — `installMigrate` and `installRepair` blocks
  now honor `--json`; `emitInstallJSON` helper added; satellite path refactored
  onto it. Matches the spec's `## Changes`.
- [✓] internal/cli/install_json_test.go — two regression tests added, exactly
  as the spec's `## Changes` prescribes (single-object stdout parse, error
  field on failure, non-nil returned error).
- [✓] internal/cli/helpers_test.go — the test-isolation fix (not in the spec's
  `## Changes` list but claimed in the ledger and genuinely needed).

## Test evidence

`go test ./internal/cli/ -run TestInstallJSON -count=1 -v` — 3/3 pass:

- `TestInstallJSON_SatelliteFailureEmitsJSONAndErrors` — PASS (pre-existing;
  confirms the refactor onto `emitInstallJSON` did not regress the satellite
  contract).
- `TestInstallJSON_RepairEmitsJSONAndErrors` — PASS.
- `TestInstallJSON_MigrateEmitsJSONAndErrors` — PASS.

`go build ./internal/cli/` — clean.

**Non-vacuity confirmed.** Each new test asserts (a) `json.Unmarshal` of
captured stdout succeeds, (b) `Error.Code` == `repair_failed` / `migrate_failed`,
(c) `cmdErr != nil`, (d) no human text on stdout (`"Repairing satellites"` /
`"Note:"`). Both would FAIL against the unfixed code:
- Repair: old code returned the locate error with empty stdout →
  `json.Unmarshal("")` fails.
- Migrate: old code printed the two note lines to stdout then returned the
  detect error with no JSON → both the `Contains("Note:")` assertion and
  `json.Unmarshal` fail.

The note-to-stderr claim is directly corroborated by the run: the "Note:" and
cobra "Error:" lines appear in the terminal (stderr passthrough) yet the tests'
stdout-only capture passes the "not on stdout" assertions.

## Behavioral-evidence adequacy

Standalone-binary manual exec is blocked by the environment sandbox. Verification
is via in-process tests driving `rootCmd.Execute()` through the real cobra
command (`runCmd` → `resetFlags` + `SetArgs` + `Execute`). This exercises the
identical `runInstall` code path a real invocation hits, including flag parsing,
`SilenceUsage`, and error propagation. For a stdout-contract bug this is adequate
evidence — the contract is "what bytes land on stdout," which the tests capture
and parse directly. The one thing tests cannot observe is the process exit code
itself, but that is mechanical: `main.go` exits 1 on any non-nil error, and the
tests assert the error is non-nil.

## Audit notes

- **Pre-existing gap (out of THIS spec's scope, not counted against delivery):**
  the top-of-`runInstall` arg validation still returns plain errors in `--json`
  mode — `mode must be 'project' or 'global'` (install.go:130-131) and
  `project mode requires a target path` (137-138) bypass the JSON contract.
  These predate the `--json` short-circuit work and are outside this spec's two
  named blocks (`installRepair` / `installMigrate`). Worth a follow-up if full
  arg-validation JSON parity is desired.
- **No double-emission risk.** `installJSON` sets `cmd.SilenceUsage = true`
  (install.go:125-126), so cobra emits no usage dump. Cobra still prints the
  returned error to **stderr** (observed as "Error: ..." in the run), which does
  not pollute stdout — stdout carries exactly the one JSON object. Contract
  holds.
- Spec `## Changes` named only install.go and install_json_test.go; the third
  changed file (helpers_test.go) is the test-isolation fix, correctly claimed in
  the ledger and legitimately in-scope.
