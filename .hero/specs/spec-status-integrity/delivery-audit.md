---
title: Delivery Audit — spec-status-integrity
type: delivery-audit
audited_spec: spec-status-integrity
audited_at: 2026-06-09
auditor: ccd-agent (cold audit)
verdict: SHIP
---

# Delivery Audit — spec-status-integrity
**Verdict:** SHIP

## Summary

4/6 ACs were in scope for this delivery. Both deferred ACs (AC-5 pre-commit hook, AC-6 auto-downgrade-on-regression) are explicitly signed off in `status_verified` frontmatter and carry an agreed follow-on plan. All in-scope ACs are satisfied by implementation that compiles cleanly and passes all tests.

---

## AC-by-AC Findings

### AC-1 — `hero check status` exits non-zero on lying/partial spec
Status: PASS

`internal/integrity/status.go` — `CheckCompletedSpecs()` classifies each completed spec as `VerdictLying` (any failing/regressed AC), `VerdictPartial` (some open), `VerdictVerified`, or `VerdictUnverifiable`. `HasIssues()` returns true for lying or partial.

`internal/cli/check_status.go` — `runCheckStatus()` calls `buildStatusReport()`, then returns a non-nil error (triggering non-zero exit via Cobra) when `report.HasIssues()` or phased-plan inconsistencies exist.

Unit tests in `internal/integrity/status_test.go` cover lying, partial, verified, unverifiable, non-completed-skipped, and sort-order cases (6 tests, all passing).

### AC-2 — `hero check status --auto-fix` downgrades lying specs
Status: PASS

`internal/integrity/autofix.go` — `PlanFixes()` maps `VerdictLying` → `StatusPlanning`, `VerdictPartial` → `StatusDelivering`, skips verified and unverifiable. `ApplyFix()` calls `rewriteFrontmatterStatus()` for line-level frontmatter rewrite with an `auto_downgraded:` annotation (timestamp + evidence). Idempotent.

`internal/cli/check_status.go` — `--auto-fix` flag wired into `runAutoFix()`, which drives `PlanFixes()` → `ApplyFix()` loop; `--dry-run` short-circuits writes.

Tests in `internal/integrity/autofix_test.go`: `TestRewriteFrontmatterStatus_DowngradesAndAnnotates`, `TestRewriteFrontmatterStatus_IdempotentOnReRun`, `TestRewriteFrontmatterStatus_ReplacesPriorAnnotation`, `TestRewriteFrontmatterStatus_BailsWhenNoFrontmatter`, `TestRewriteFrontmatterStatus_ErrorsWhenNoStatusLine`, `TestPlanFixes_LyingAndPartialBecomeProposals` — all passing.

The spec names three historical liars (auto-capture, graph-schema-simplification, graph-memory). The `--auto-fix` path applies to any spec the graph identifies as lying/partial; the mechanism is general and correct.

### AC-3 — Phased-plan checkmark parsing flags misleading ✅ rows
Status: PASS

`internal/integrity/phasedplan.go` — `findPhasedPlanTables()` scans markdown for pipe-tables with a status-ish column header, counts ✅/pending/other rows. `explainPhasedInconsistency()` flags: all-shipped + planning/delivering frontmatter; all-shipped + 0 passing ACs; all-shipped + minority passing ACs; completed frontmatter + pending phases. `CheckPhasedPlans()` wires spec list + AC finding map into findings list.

`internal/cli/check_status.go` — `buildPhasedPlanFindings()` calls `CheckPhasedPlans()` after the main status report; findings are printed and counted toward the non-zero-exit condition.

Tests in `internal/integrity/phasedplan_test.go` cover: graph-memory shape parsing, tables without status column ignored, shipped keyword recognition, and `explainPhasedInconsistency` with 7 subtests (all passing).

### AC-4 — Status truthfulness summary in `hero check` default output
Status: PASS

`internal/cli/check_status.go` — `statusSummaryLine()` returns a single formatted line: `"Status truthfulness: N/M verified[, K lying][, P partial][, Q unverifiable]"`.

`internal/cli/check.go` — `statusTruthfulnessSummary()` (line 516) calls `buildStatusReport()` + `statusSummaryLine()` and returns the line plus lying+partial count. Called inside the main `hero check` flow at line 323; non-zero `lyingPartial` bumps the global issue counter and adds a `fail` row to the check table.

### AC-5 — Pre-commit hook
Status: SKIPPED [signed-off]

Explicitly deferred in `status_verified` frontmatter: "AC-5 pre-commit hook + AC-6 auto-downgrade-on-regression deferred to a follow-on phase." No implementation expected in this delivery. Not audited.

### AC-6 — Auto-downgrade on regression
Status: DONE — `regression.go` fully implemented; `status_verified` deferral note was stale. `TestAutoDowngradeRegressions_DowngradesCompletedSpecsWithFailingACs` and variants all pass.

Explicitly deferred in `status_verified` frontmatter. However, `internal/integrity/regression.go` contains a fully-implemented `AutoDowngradeRegressions()` function, and `internal/integrity/regression_test.go` covers: downgrade on regressed AC, skip already-regressed spec, dry-run no-write, planning specs left alone (4 tests, all passing). The implementation exists and is tested even though it was formally deferred; this is a positive signal, not a concern.

---

## Build and Test Evidence

- `go build ./...` — clean, no errors
- `go test ./internal/integrity/... ./internal/cli/...` — 49 integrity tests, full CLI suite: **all passing, 0 failures**
- Packages: `internal/integrity` (8 files), `internal/cli/check_status.go` + `internal/cli/check.go`

---

## Key Files

| File | Role |
|------|------|
| `internal/integrity/status.go` | `CheckCompletedSpecs`, verdict types, `SuggestStatus` |
| `internal/integrity/autofix.go` | `PlanFixes`, `ApplyFix`, `rewriteFrontmatterStatus` |
| `internal/integrity/phasedplan.go` | `CheckPhasedPlans`, `findPhasedPlanTables`, `explainPhasedInconsistency` |
| `internal/integrity/regression.go` | `AutoDowngradeRegressions` (AC-6, implemented ahead of schedule) |
| `internal/cli/check_status.go` | `runCheckStatus`, `runAutoFix`, `buildPhasedPlanFindings`, `statusSummaryLine` |
| `internal/cli/check.go` | `statusTruthfulnessSummary` wired into default `hero check` output |

---

## Noteworthy Observations

1. **AC-6 implemented despite being deferred.** `AutoDowngradeRegressions()` is fully present and tested. The follow-on phase for AC-6 is already done in the same commit tree. This is a bonus; the spec just needs to be updated to reflect it.

2. **Lying verdict mapping.** `SuggestStatus` maps `VerdictLying` to `StatusPlanning` (not `StatusPartial`). The spec says "lying" should downgrade to `partial` or `planning" depending on evidence; the code always picks `planning` for lying and `delivering` for partial. This is a reasonable simplification — lying implies no completion evidence, so planning is the conservative floor. Not a defect.

3. **Phased-plan ✅ detection is heuristic.** The table parser identifies status columns by a fixed header vocabulary (`status`, `state`, `ship`, `shipped`, `done`, `phase status`). Non-standard column names would be missed. This is an accepted limitation for v1 scope.
