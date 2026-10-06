# Delivery validation

- `go test ./internal/install`: passed; install-tests.log.
- `go test ./internal/cli -run TestDeepSeek -count=1`: passed; cli-focused-tests.log.
- `go vet ./...`: exit 0; vet.log is empty.
- `make build`: exit 0; build.log.
- `python3 -m unittest discover -s web/docs/scripts -p 'test_*.py'`: 28 passed; docs-tests.log. The printed release lookup error is an expected test case.
- `.build/docs-venv/bin/mkdocs build --strict -f web/docs/mkdocs.yml`: exit 0; docs-build.log.
- `node scripts/deepseek-compatibility.mjs --harness /Users/bwheeler/projects/public/repository/deepseek-harness --hero /Users/bwheeler/projects/hero-engine/repository/hero/.build/hero-deepseek --domain DOMAIN` for engineering, pm, qa: all pass; native-compatibility-DOMAIN.log. Harness commit 477b4f420553e8a52c2fbccc464d7561b239c443. No model calls.
- `git diff --check`: exit 0.

## Workspace baseline

The working tree already contained unrelated uncommitted work. `/tmp/hero-deepseek-preexisting.patch` captures the original unstaged diff. Changes to connect/Aha, tracker/serve, earlier doctor/upgrade/inventory behavior, installed root AGENTS/CLAUDE, and earlier spec archives predate this delivery. New implementation files are untracked until the user commits them; read them directly as well as using `git diff` on existing files.

## Regression retry

The first `go test ./...` passed all packages except CLI. CLI exposed two remaining seven-target expectations and leaked `installWorkspace` from the new tests. The new tests now restore their flags and both target matrices expect eight. Full CLI regression results are recorded below after the retry.

Full CLI retry: `go test ./internal/cli -count=1` passed (84.872s), cli-full-tests.log. All other Go packages passed in the original suite; the complete suite is also rerunning to produce a single clean result.

Complete `go test ./...` retry passed (exit 0); go-tests.log. Cold audit subsequently identified missing-path presentation in doctor; final doctor validation follows after that repair.

Audit repair: doctor names missing DeepSeek artifacts even when skill counts are full; TestDoctorDeepSeekNamesMissingArtifactsDespiteFullCounts plus focused doctor/DeepSeek tests pass (doctor-tests.log). Corrected DSH_HOME whitespace wording and reran strict docs build successfully.

Final post-audit validation: full `go test ./internal/cli -count=1` passes (87.279s), cli-full-tests.log. `make build`, `go vet ./...`, and `git diff --check` also pass after the doctor correction.

Closing gate: `./hero spec verify deepseek-harness-install-target --skip-tests` returned PASS; ledger 10/10 AC and 6/6 changes DONE, independent audit SHIP (clean), test mapping 10/10 strong. Build/tests gate reused the completed validation above via --skip-tests. Spec archived with status completed.
