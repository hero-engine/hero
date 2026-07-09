# Delivery audit — cst-verify-lifecycle-scoping

**Audited:** `git diff -- internal/cli/verify.go internal/cli/verify_test.go` (working tree)
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] **AC-1** — No delivery gates for planning/draft; clear lifecycle message naming status and pointing to `/deliver`. Guard at `internal/cli/verify.go:118-126` runs `isPreDeliveryStatus(target.Status) && !verifyForce`, returns a `fmt.Errorf` with the spec slug, the literal status (`%s` of `target.Status`), and "Run /deliver to begin implementation". `isPreDeliveryStatus` at `verify.go:72-74` matches only `StatusPlanning`/`StatusDraft`. Test `TestVerify_PlanningStatusGuarded` (verify_test.go:308) asserts the error fires and contains "planning status", and additionally asserts the spec was **not** archived. PASS.
- [✓] **AC-2** — `--force` bypasses the guard. Condition `&& !verifyForce` (verify.go:118) lets the gates run when `--force` is set; `verifyForce` is the existing `--force` flag (verify.go:23, 48). Test `TestVerify_PlanningStatusForceBypass` (verify_test.go:332) runs `--force` on a planning spec with a complete ledger+audit, asserts no error, and asserts the spec **was** archived — proving the gates actually executed, not just that the error was suppressed. PASS.
- [✓] **AC-3** — `delivering`, `in-review`, `handed_back`, `completed` unaffected. `isPreDeliveryStatus` matches only planning/draft, so all four delivery-phase statuses fall through to the gate block unchanged (verify.go:128+). The pre-existing 16 verify tests all use `status: delivering` and stay green (full `TestVerify` run: all PASS). `handed_back` (`StatusHandedBack = "handed_back"`) is confirmed NOT in the helper. No regression. PASS.
- [✓] **AC-4** — JSON mode emits `SKIPPED` + `lifecycle` gate, exit 0. JSON branch at verify.go:120-123 builds `VerifyResult{Result: "SKIPPED"}` with one `GateResult{Name: "lifecycle", Result: "SKIPPED"}` and returns `outputJSON(r)` (nil error → exit 0). Test `TestVerify_PlanningStatusJSON` (verify_test.go:353) asserts no error, unmarshals output, asserts `Result == "SKIPPED"` and exactly one gate named `lifecycle`. PASS.
- [✓] **AC-5** — Regression tests for guard + force bypass. Three real tests added (verify_test.go:308, 332, 353), covering guard / force / JSON. PASS.

## Changes

- [✓] `internal/cli/verify.go` — `isPreDeliveryStatus` helper (verify.go:72-74) + lifecycle guard in `runVerify` (verify.go:113-126). Confirmed in diff.
- [✓] `internal/cli/verify_test.go` — 3 regression tests (verify_test.go:308-381). Confirmed in diff.

## Ledger verification

Every Completion Ledger row marked DONE has corresponding evidence in the diff/tests:

| Ledger row | Claim | Evidence | Verdict |
|---|---|---|---|
| AC-1 | guard + helper; `TestVerify_PlanningStatusGuarded` | verify.go:72-74,118-126; test passes | ✓ real |
| AC-2 | `&& !verifyForce`; `TestVerify_PlanningStatusForceBypass` | verify.go:118; test passes & asserts archival | ✓ real |
| AC-3 | matches only planning/draft; 16 existing tests green | helper body; full suite green | ✓ real |
| AC-4 | JSON branch; `TestVerify_PlanningStatusJSON` | verify.go:120-123; test passes | ✓ real |
| AC-5 | 3 tests added | verify_test.go:308-381 | ✓ real |

No performative DONE rows. No downgrades.

## Placement / regression check

- **Guard is correctly placed:** it sits after the already-completed-and-archived early-return (verify.go:103-111) and before `result := VerifyResult{}` and any gate execution (verify.go:128+). A planning spec returns at verify.go:125 long before any archive code runs.
- **No path leaks a planning spec into archival:** `TestVerify_PlanningStatusGuarded` explicitly stats the archive path and fails if the spec was moved. The non-force planning path returns an error before `result` is constructed.
- **Force bypass genuinely re-enables archival:** `TestVerify_PlanningStatusForceBypass` asserts the spec *is* archived under `--force`, so the test would catch a guard that merely swallowed the error without running gates.
- **JSON exit code:** SKIPPED path returns `outputJSON(r)` (nil error), mirroring the existing already-completed JSON convention (verify.go:104-107) — exit 0.

## Verification run

- `go build ./...` — clean, exit 0.
- `go test ./internal/cli/ -run TestVerify` — all PASS, including the 3 new tests (`TestVerify_PlanningStatusGuarded`, `TestVerify_PlanningStatusForceBypass`, `TestVerify_PlanningStatusJSON`) and all 16 pre-existing verify tests. Re-run with `-count=1` to defeat cache: still `ok`.

## Audit notes

- The spec references `StatusDraft` and `StatusHandedBack`; both are confirmed real constants (spec.go:56,59). Helper covers planning+draft only, as specified.
- Message text matches the Suggested Fix Approach near-verbatim, including the `/deliver` pointer and the `--force` escape hatch.
- Diff is tightly scoped to the two named files. No scope drift.
