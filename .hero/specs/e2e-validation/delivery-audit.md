# Delivery audit — e2e-validation

**Audited:** `git log --oneline --follow scripts/e2e_smoke.sh` → commit `982742d` (hero v0.8.0 initial public release)
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria

- [✓] **SC-1: `scripts/e2e_smoke.sh` exists, accepts `<owner/repo>`, captures per-step timing + exit code + output** — Script is present at `scripts/e2e_smoke.sh`, executable (`-rwxr-xr-x`). Positional arg wired at line 21 (`REPO_SLUG="${1:-go-task/task}"`). `run_step` helper (lines 42–74) times with nanosecond `perl` calls (lines 49, 52), captures exit code (`rc=$?`, line 51), counts lines and bytes (lines 56–57), appends per-step markdown entry to `${LOG}` (lines 59–70), writes stdout+stderr to `${RUN_DIR}/${label// /_}.txt` (line 44, 50). All claimed behaviors present and consistent.
- [✓] **SC-2: Script produces markdown observation log under `tmp/e2e-smoke/<repo>-<ts>/`** — `RUN_DIR` is set to `${TMP_BASE}/${SAFE_SLUG}-${RUN_TS}` (line 27), where `TMP_BASE="${ROOT}/tmp/e2e-smoke"` (line 24). `LOG="${RUN_DIR}/observations.md"` (line 29). `section` helper appends `## …` headers (lines 36–38). `run_step` appends `### …` step headers, exit code, timing, byte/line counts, and output snippets. Per-step `.txt` files written alongside (line 44). Operator-fill stub appended at wrap-up (lines 162–175). Fully implemented.
- [✓] **SC-3: Run-1 executed on `go-task/task`, findings documented** — Spec `## Findings from run 1` section documents 3 real bugs (FTS5 crash on `?`, `ask` hitting wrong index, `relevant` silent on zero results) and 6 UX rough edges with specific command names and observed behaviors. Evidence is substantive and internally consistent. Note: `tmp/` is gitignored (`.gitignore` line 46) so the `tmp/e2e-smoke/go-task-task-20260428T011642Z/observations.md` artifact referenced in spec is not on disk — this is expected and not a defect; the findings are durably documented in the spec body.
- [✓] **SC-4: Future run plan documented** — `## Future runs to do` section names three concrete next runs: Python (`httpie/httpie`), TypeScript (`vadimdemedes/ink`), and post-polish re-run on `go-task/task`. Concrete targets, not vague intent.

## Changes

- [✓] `scripts/e2e_smoke.sh` — New 179-line bash script committed in `982742d`. `run_step` helper implements timing, exit-code capture, per-step `.txt` output, and markdown log appending exactly as specified. `section` helper writes `##`-level headers. Script handles `HERO_BIN` and `KEEP` env vars, clones with `--depth=50`, and emits an operator observation stub at the end.

## Open items (if any)

None. No PARTIAL, SKIPPED, or BLOCKED rows in the Completion Ledger.

## Audit notes

- The `tmp/e2e-smoke/go-task-task-20260428T011642Z/observations.md` file referenced in the spec (`## What's shipped`) is not present on disk because `tmp/` is in `.gitignore`. This is the correct and expected state — smoke run artifacts are ephemeral by design. The spec body captures findings durably, which satisfies SC-3. This is noteworthy for clarity but not a defect.
- SC-1 ledger entry claims "output size" capture. The script captures output in bytes (`wc -c`, line 57) and lines (`wc -l`, line 56) and writes both to the log (line 59). The claim is accurate.
- The `run_step` helper captures both stdout and stderr (line 50: `> "${out_file}" 2>&1`), which is correct for a smoke harness — test output and error messages are both relevant.
- One minor implementation nuance: timing uses `perl -MTime::HiRes=time` rather than Bash's `$SECONDS` or `date +%s%N`, which means the script requires `perl` to be on `PATH`. This is an undocumented dependency but not a spec requirement and perl is standard on macOS/Linux.

**Verdict:** SHIP
