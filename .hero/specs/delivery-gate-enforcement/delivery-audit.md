# Delivery Audit — delivery-gate-enforcement

**Audited:** uncommitted working-tree changes (git diff + git status), verified against spec.md
**Verdict:** SHIP
**Surface:** clean

---

## Acceptance Criteria

| AC | Spec requirement (abbreviated) | Verdict | Evidence |
|---|---|---|---|
| 1 | verify with valid ledger+audit+tests -> PASS, flip, archive | PASS | `TestVerify_AllGatesPass` — creates spec with ledger + audit, runs verify --skip-tests, asserts PASS and archived path exists. Independently confirmed test passes. |
| 2 | verify with missing ledger -> FAIL, name gate | PASS | `TestVerify_MissingLedger` — spec has no Completion Ledger section, verify returns "verification failed" error. Gate 1 message confirmed in test output: "no Completion Ledger section found". |
| 3 | verify with PARTIAL rows -> FAIL, list rows | PASS | `TestVerify_PartialRows` — ledger row with PARTIAL status, verify fails, spec NOT archived. `checkLedger` appends "AC-%d is PARTIAL: %s" for each non-DONE row. |
| 4 | signed-off SKIPPED/BLOCKED pass gate | PASS | `TestVerify_SignedOffPassesGate` — one DONE row + one SKIPPED with `[signed-off]`, verify returns PASS. `TestParseLedger_SignedOff` also validates the `[signed off]` (space variant). |
| 5 | verify with no audit file -> FAIL | PASS | `TestVerify_MissingAudit` — spec with ledger but no delivery-audit.md, verify returns "verification failed" error. |
| 6 | verify with HOLD verdict -> FAIL | PASS | `TestVerify_HoldAudit` — audit file has `**Verdict:** HOLD`, verify returns "verification failed". Gate message: "verdict: HOLD". |
| 7 | --json output with slug, result, gates array | PASS | `TestVerify_JSON` — parses output as JSON into `VerifyResult` struct, checks `Slug`, `Result`, and `len(Gates) >= 4`. Struct has correct json tags. |
| 8 | --skip-tests -> Gate 4 SKIPPED | PASS | `TestVerify_SkipTests` — runs with `--skip-tests --json`, finds Gate 4 "Build & Tests" with Result "SKIPPED". |
| 9 | --force -> bypass gates, FORCED, archive with warning | PASS | `TestVerify_Force` — spec with no ledger/audit (would normally fail), runs with `--force`, asserts no error and "FORCED" in output. `completeAndArchive` called on FORCED path. |
| 10 | parse ledger tables (pipe-delimited, case-insensitive) | PASS | `TestParseLedger_CaseInsensitiveStatus` — tests "done", "Done", "DONE" all parse to LedgerDone. `TestParseLedger_BoldStatus` — `**DONE**` stripped to DONE. |
| 11 | exercise checkbox without detail -> fails gate | PASS | `TestParseLedger_ExerciseNoDetail` — checkbox `[x]` with trailing colon but no text, ExerciseDetail is empty. `checkLedger` reports "checked but no detail provided" and sets FAIL. |
| 12 | detect test command from stack | PASS | `detectTestCommand` in verify.go checks go.mod, package.json, pyproject.toml, Cargo.toml, build.gradle, pom.xml in that order. Config override checked first via `cfg.Verify.TestCommandOrDefault()`. |

## Changes Verification

| # | Claimed change | Verdict | Evidence |
|---|---|---|---|
| 1 | `internal/spec/ledger.go` (new) | LANDED | 301 lines. ParseLedger, parseTable, splitTableRow, parseDataRow, parseStatus, parseCheckbox, stripBold. Types: LedgerStatus, LedgerRow, LedgerResult. Tolerant parser handles bold, case variations, missing pipes. |
| 2 | `internal/spec/ledger_test.go` (new) | LANDED | 8 test functions: AllDone, MixedStatuses, SignedOff, Missing, CaseInsensitiveStatus, ExerciseNoDetail, BoldStatus, NilSpec. All pass. |
| 3 | `internal/spec/audit.go` (new) | LANDED | 118 lines. FindAuditReport (Spec-based), FindAuditReportInDir (dir-based), parseAuditHeader, extractHeaderValue. Types: AuditResult. |
| 4 | `internal/spec/audit_test.go` (new) | LANDED | 7 test functions: ShipClean, ShipNoteworthy, Hold, Missing, NilSpec, MalformedHeader, FindAuditReportInDir. All pass. |
| 5 | `internal/cli/verify.go` (rewrite) | LANDED | Complete rewrite from cosmetic AC checklist to four-gate enforcement. Old code (score-based cosmetic report + AI prompt generation) fully replaced. New: GateResult/VerifyResult structs, checkLedger, checkAudit, checkCoverage, checkTests, detectTestCommand, completeAndArchive, printVerifyReport, outputJSON. Flags: --json, --skip-tests, --force. |
| 6 | `internal/cli/verify_test.go` (new) | LANDED | 10 test functions covering all gate paths. All pass. |
| 7 | `domains/engineering/commands/deliver.md` (edit) | LANDED | Closeout flow rewritten: removes "set status: completed in frontmatter" instruction, replaces with "run hero spec verify" as the only path. Documents --skip-tests, --json, --force. |
| 8 | `domains/engineering/agents/engineer.md` (edit) | LANDED | Exercise-the-feature format tightened: requires "command run + what was observed", warns bare `[x]` will fail Gate 1. |
| 9 | `domains/engineering/agents/feature-delivery-lead.md` (edit) | LANDED | Step 19 changed from "move spec + update status" to "run hero spec verify --skip-tests", with explicit "do not edit status: completed directly" instruction. |

## Additional changes (not in spec Changes section but necessary)

| File | What | Verdict |
|---|---|---|
| `internal/config/config.go` | Added `VerifyConfig` struct with `RunTests`, `TestCommand`, helper methods, and `Verify` field on `Config`. | APPROPRIATE — needed for verify.test_command and verify.run_tests config support (spec mentions hero.json fields). |
| `internal/cli/deliver_test.go` | Existing tests updated to match gated behavior: TestVerify, TestVerifyNoAcceptanceCriteria, TestDeliverManualAutoArchivesOnVerify, TestVerifyDoesNotArchiveIncomplete all rewritten. | APPROPRIATE — old tests asserted the cosmetic report behavior that was replaced. |
| `internal/cli/helpers_test.go` | `resetFlags()` updated to reset verifyJSON, verifySkipTests, verifyForce between test runs. | APPROPRIATE — standard test hygiene for new cobra flags. |
| `.hero/knowledge/decisions/verify-gates-the-flip.md` | Decision record capturing the design inversion rationale. | APPROPRIATE — novel architectural decision properly recorded. |

## Build & Test

- `go build ./...` — clean, no errors.
- `go test ./internal/spec/...` — all pass (0.212s). 8 ledger tests + 7 audit tests + pre-existing spec tests.
- `go test ./internal/cli/...` — all pass (12.999s). 10 new verify tests + all existing deliver/complete tests.
- No regressions detected. Old tests adapted to new gated behavior (not deleted).

## Design Integrity Check

The core design inversion is correctly implemented: `hero verify` gates the status flip. The old flow (agent flips status, verify rubber-stamps) is gone. The new flow (agent calls verify, verify checks gates, verify flips status) is enforced in code and instructions.

Key design points verified:
- **Gate 1 (ledger)** is a hard gate — FAIL blocks archive.
- **Gate 2 (audit)** is a hard gate — FAIL blocks archive.
- **Gate 3 (coverage)** is advisory — reports gaps but does not fail. Matches spec Boundaries section.
- **Gate 4 (tests)** is optional — --skip-tests and config disable it. When enabled, runs auto-detected or configured test command.
- **--force** bypass is logged as FORCED, visible in output. Escape valve, not workflow.
- **Already-completed specs** handled idempotently (TestVerify_AlreadyCompleted).
- **extractCriteriaItems** retained for backward compatibility with deliver_test.go — not orphaned.

## Noteworthy observations

None. The delivery is clean — code matches spec, tests cover all ACs, instruction changes are consistent, no dead code left behind, no orphaned imports.

---

**Verdict: SHIP**
