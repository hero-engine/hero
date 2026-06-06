---
title: "Delivery Gate Enforcement — hero verify Becomes the Load-Bearing Checkpoint"
slug: delivery-gate-enforcement
type: feature
status: delivering
priority: P0
severity: high
size: large
domain: engineering
horizon: now
tags: [delivery, verification, quality, enforcement, gate, trust]
relations:
  - target: delivery-completion-discipline
    kind: supersedes
  - target: e2e-area-suites
    kind: related
  - target: per-feature-smoke-coverage
    kind: related
  - target: spec-lifecycle-hygiene-breakdown
    kind: related
created: 2026-06-06
---

# Delivery Gate Enforcement — hero verify Becomes the Load-Bearing Checkpoint

## Goal

Make `hero verify` the single enforcement point that prevents incomplete,
untested, or unaudited work from being marked completed. Today the delivery
pipeline assumes agent honesty — the agent self-reports via a Completion
Ledger, optionally spawns a cold audit, then flips status to `completed`
and calls `hero verify` which rubber-stamps the archive. The result: specs
get marked delivered that were never wired in, never exercised end-to-end,
never actually verified.

After this lands: `hero verify <slug>` checks four gates before it will
flip status and archive. If any gate fails, it refuses, names the specific
failures, and the delivery lead routes them back to the engineer. No agent
can skip the gates. The pipeline goes from honor system to enforcement.

## Kickoff

Pick up at: greenfield. Deliver `delivery-gate-enforcement` — make
`hero verify` the real enforcement checkpoint for delivery closeout. Four
gates: ledger present with all rows DONE, audit report on disk with SHIP
verdict, test coverage mapped to ACs, and build/test passing. Verify
flips status + archives only when all gates pass. Key files:
`internal/spec/ledger.go` (new parser), `internal/cli/verify.go` (rewrite
to gated flow), `internal/cli/verify_test.go` (new), `deliver.md` and
`engineer.md` instruction updates. Run `go test ./internal/spec/...
./internal/cli/... ./internal/coverage/...` to validate.

## Problem

The delivery pipeline has five steps specified in instructions but zero
enforced by tooling:

| Step | Specified in | Enforced by |
|---|---|---|
| Completion Ledger produced | engineer.md | Nothing |
| All AC rows DONE | deliver.md | Nothing |
| Cold audit run, verdict SHIP | deliver.md | Nothing |
| Tests exist for ACs | deliver.md | Nothing |
| Status flip to completed | deliver.md | Nothing — agent edits frontmatter directly |

`hero verify` today:
- Prints a cosmetic AC checklist
- Prints a "copy to agent" prompt
- Auto-archives if status is already `completed`
- Checks nothing

The `delivery-completion-discipline` spec (completed 2026-05-22) correctly
diagnosed this. Its fix: more instructions. No tooling. The architecture
still assumes agent honesty. Agents can skip the audit, write a
performative ledger, flip status, and `hero verify` rubber-stamps it.

This spec supersedes `delivery-completion-discipline` by turning its
instruction-only fixes into tooling-enforced gates.

## Design

### Inversion: verify gates the flip, not the other way around

Today's flow:
1. Agent flips `status: completed` in frontmatter
2. Agent calls `hero verify <slug>`
3. Verify sees status=completed, auto-archives. Done.

New flow:
1. Agent calls `hero verify <slug>` (status is still `delivering`)
2. Verify checks four gates
3. If all pass: verify flips status to `completed`, archives, returns PASS
4. If any fail: verify returns FAIL with structured failures, does not flip

The agent no longer touches `status: completed` directly. `hero verify` is
the only path to completed. This is the core design change.

### Gate 1: Completion Ledger present and clean

Parse the spec's `## Completion Ledger` section. Check:
- Section exists
- AC table has a row for every AC in the spec's `## Acceptance Criteria`
- Every AC row is `DONE` (or has a `[signed-off]` annotation for SKIPPED/BLOCKED)
- Exercise-the-feature checkbox is checked with a description (not just `[x]`)

New file: `internal/spec/ledger.go` — parser for the Completion Ledger
markdown format. Returns a `LedgerResult` struct:

```go
type LedgerStatus string

const (
    LedgerDone    LedgerStatus = "DONE"
    LedgerPartial LedgerStatus = "PARTIAL"
    LedgerSkipped LedgerStatus = "SKIPPED"
    LedgerBlocked LedgerStatus = "BLOCKED"
)

type LedgerRow struct {
    Index     int
    Summary   string
    Status    LedgerStatus
    Note      string
    SignedOff bool // true if [signed-off] annotation present
}

type LedgerResult struct {
    Found             bool
    ACRows            []LedgerRow
    ChangesRows       []LedgerRow
    ExerciseChecked   bool
    ExerciseDetail    string // the description after the checkbox
    ExcellenceChecked bool
    ExcellenceNote    string
}
```

The parser reads markdown tables from the `## Completion Ledger` section,
splitting on `### Acceptance Criteria` and `### Changes` sub-headers. It
extracts status from the `Status` column, matching case-insensitively
against the four values. The `[signed-off]` annotation is detected in the
Note column.

### Gate 2: Audit report exists with SHIP verdict

Look for `delivery-audit.md` in the spec's directory (both
`.hero/planning/{type}/{slug}/` and `.hero/specs/{slug}/`). Parse the
verdict from the file header:

```go
type AuditResult struct {
    Found   bool
    Path    string
    Verdict string // "SHIP" or "HOLD"
    Surface string // "clean" or "noteworthy"
}
```

The parser reads the file, finds `**Verdict:** SHIP|HOLD` in the header
lines. Simple string scanning — the audit report format is already
standardized in the `delivery-audit` skill.

### Gate 3: Test coverage for acceptance criteria

Use the existing `coverage.Analyze()` function to check AC-to-test mapping.
This already works — it maps acceptance criteria to test files using keyword
matching. The gate checks:
- Coverage report can be generated (spec has ACs, project has test files)
- Coverage gaps count is reported (advisory, not blocking — see Boundaries)

The coverage gate reports but does not block. Zero-gap coverage is
aspirational; the ledger + audit gates are the hard enforcement. Coverage
is the "you should know about these gaps" signal.

### Gate 4: Build passes

Run the project's test command (detected from stack or configured in
`hero.json`). For this project: `go build ./... && go test ./...`.

This gate is optional and configurable:
- `hero.json` field: `verify.run_tests: true|false` (default: true)
- `hero.json` field: `verify.test_command: "go test ./..."` (auto-detected if not set)
- `--skip-tests` flag on `hero verify` to bypass in CI or when tests were just run

When enabled, verify runs the command and checks exit code. On failure,
reports the output and refuses to archive.

### Verify output

`hero verify` produces structured output for both humans and agents:

```
hero verify <slug>

  Delivery Gate Report: <slug>
  ════════════════════════════════════════════

  Gate 1 — Completion Ledger           PASS
    ✓ Ledger found
    ✓ 5/5 AC rows DONE
    ✓ 3/3 Changes rows DONE
    ✓ Exercise-the-feature: checked with detail

  Gate 2 — Delivery Audit              PASS
    ✓ Audit report found at .hero/planning/features/<slug>/delivery-audit.md
    ✓ Verdict: SHIP (clean)

  Gate 3 — Test Coverage               ADVISORY
    ✓ 4/5 criteria have test coverage (1 gap)
    △ Gap: AC-3 "WHEN user exports CSV..." — no test mentions "export", "csv"

  Gate 4 — Build & Tests               PASS
    ✓ go test ./... — 79 packages, 0 failures

  ────────────────────────────────────────────
  Result: PASS — all gates satisfied
  → Status flipped to completed
  → Archived to specs/<slug>/
```

On failure:
```
  Gate 1 — Completion Ledger           FAIL
    ✓ Ledger found
    ✗ AC-3 is PARTIAL: "export handler not wired to router"
    ✗ Exercise-the-feature: not checked

  Result: FAIL — 2 gate failures
  → Status NOT changed. Fix the failures and re-run hero verify.
```

Also supports `--json` for machine consumption by the delivery lead agent.

### Instruction changes

**`deliver.md`** — change the closeout flow (steps 6-7 area):

Remove: "set the spec's `status: completed` in the frontmatter and run
`hero spec verify <slug>`"

Replace with: "Run `hero verify <slug>`. Verify checks four gates (ledger,
audit, coverage, tests) and flips status to `completed` only if all pass.
If verify returns FAIL, read the specific failures and route them back to
the engineer to address. Do not edit `status: completed` directly — verify
is the only path to completed."

**`engineer.md`** — tighten exercise-the-feature:

Add to the Exercise-the-feature check format: "The description must include
the actual command run and a one-line summary of what was observed. Example:
`- [x] Exercised: ran 'hero verify test-slug', confirmed 4 gates reported,
  FAIL returned for missing audit report as expected`"

Mark the old pattern as insufficient: "`- [x] Exercised end-to-end` with
no description is not valid — verify will reject it."

**`feature-delivery-lead.md`** — update step 19:

Change from: "move the spec from planning/ to specs/ and update its status
to completed"

To: "Run `hero verify <slug>`. If PASS, verify handles the status flip and
archive. If FAIL, route the specific gate failures back to the engineer."

## Changes

1. `internal/spec/ledger.go` (new) — Completion Ledger parser. Extracts
   AC rows, Changes rows, exercise check, excellence check from the
   `## Completion Ledger` section of a spec. Returns `LedgerResult`.

2. `internal/spec/ledger_test.go` (new) — Tests for ledger parsing:
   happy path (all DONE), mixed statuses, missing ledger, malformed table,
   signed-off annotations, exercise checkbox variants.

3. `internal/spec/audit.go` (new) — Audit report reader. Scans for
   `delivery-audit.md` in the spec directory, parses verdict and surface
   from the standardized header format.

4. `internal/spec/audit_test.go` (new) — Tests for audit report parsing:
   SHIP/clean, SHIP/noteworthy, HOLD, missing file, malformed header.

5. `internal/cli/verify.go` (rewrite) — Four-gate enforcement. Runs
   ledger check, audit check, coverage check, optional test run. Flips
   status and archives only on all-pass. Returns structured output.
   Supports `--json`, `--skip-tests`, `--force` (bypass gates for
   exceptional cases with a warning).

6. `internal/cli/verify_test.go` (new) — Integration tests for the
   gated verify flow: all-pass archives, ledger-fail blocks, audit-fail
   blocks, force-flag overrides with warning, JSON output format.

7. `domains/engineering/commands/deliver.md` (edit) — Update closeout
   steps to use `hero verify` as the gate. Remove instructions for agents
   to edit `status: completed` directly.

8. `domains/engineering/agents/engineer.md` (edit) — Tighten
   exercise-the-feature format. Require command + observation, not just
   a checkbox.

9. `domains/engineering/agents/feature-delivery-lead.md` (edit) — Update
   step 19 to route through `hero verify` instead of manual status flip.

## Acceptance Criteria

**AC-1:** WHEN `hero verify <slug>` is called on a spec with a valid
Completion Ledger (all DONE), a SHIP audit report, and passing tests,
THE SYSTEM SHALL flip status to `completed`, archive to `specs/`, and
print a PASS result with all four gates shown.

**AC-2:** WHEN `hero verify <slug>` is called on a spec missing a
Completion Ledger section, THE SYSTEM SHALL print a FAIL result naming
"Gate 1 — Completion Ledger: FAIL — no ledger section found" and SHALL
NOT change the spec status.

**AC-3:** WHEN `hero verify <slug>` is called on a spec with PARTIAL
AC rows in the ledger, THE SYSTEM SHALL print a FAIL result listing each
non-DONE row with its status and note, and SHALL NOT change the spec
status.

**AC-4:** WHEN `hero verify <slug>` is called on a spec with SKIPPED
or BLOCKED rows that have a `[signed-off]` annotation, THE SYSTEM SHALL
treat those rows as passing the gate.

**AC-5:** WHEN `hero verify <slug>` is called on a spec with no
`delivery-audit.md` file in its directory, THE SYSTEM SHALL print a FAIL
result naming "Gate 2 — Delivery Audit: FAIL — no audit report found"
and SHALL NOT change the spec status.

**AC-6:** WHEN `hero verify <slug>` is called on a spec with an audit
report containing `**Verdict:** HOLD`, THE SYSTEM SHALL print a FAIL
result and SHALL NOT change the spec status.

**AC-7:** WHEN `hero verify <slug>` is called with `--json`, THE SYSTEM
SHALL output a JSON object with fields: `slug`, `result` (PASS|FAIL),
`gates` (array of gate results each with `name`, `result`, `details`).

**AC-8:** WHEN `hero verify <slug>` is called with `--skip-tests`, THE
SYSTEM SHALL skip Gate 4 and report it as SKIPPED in the output.

**AC-9:** WHEN `hero verify <slug>` is called with `--force`, THE
SYSTEM SHALL print a warning, skip all failed gates, flip status, and
archive — but prefix the output with "FORCED — gates bypassed" so the
override is visible in logs.

**AC-10:** THE SYSTEM SHALL parse Completion Ledger tables in the format
documented in `engineer.md` — pipe-delimited markdown tables under
`### Acceptance Criteria` and `### Changes` sub-headers, with Status
column containing DONE/PARTIAL/SKIPPED/BLOCKED (case-insensitive).

**AC-11:** WHEN the Exercise-the-feature checkbox is checked but has no
description text after it (just `[x]` with no content), THE SYSTEM SHALL
treat it as unchecked for gate purposes and report "exercise checked but
no detail provided."

**AC-12:** THE SYSTEM SHALL detect test commands from the project stack
(Go → `go test ./...`, Node → `npm test`, Python → `pytest`) or from
`verify.test_command` in `hero.json`. When no test command can be
determined, Gate 4 reports SKIPPED with "no test command detected."

## Boundaries

- **Coverage is advisory, not blocking.** Gate 3 reports gaps but does not
  fail the verify. The ledger + audit gates are the hard enforcement. We
  may promote coverage to a blocking gate later, but starting advisory
  avoids false-negative friction on specs where keyword matching is weak.

- **`--force` exists for exceptional cases.** Sometimes a human decides
  to ship with known gaps. Force lets them, but the override is logged
  visibly. It's an escape valve, not a workflow.

- **No changes to the audit skill.** The `delivery-audit` skill is
  well-designed and working. This spec only adds detection of its output
  file by the tooling — it does not change audit behavior.

- **No `hero_completion_score` MCP tool.** The `delivery-completion-
  discipline` spec considered this and decided against it. If agents fill
  the ledger dishonestly, a score derived from a dishonest ledger doesn't
  help. The audit is the independent check; verify is the enforcement.
  If dishonest ledgers persist after this lands, revisit.

- **Ledger parser is tolerant.** Real ledgers have formatting variations
  (extra pipes, inconsistent spacing, bold text in cells). The parser
  should handle common markdown table variants, not require pixel-perfect
  formatting.

## Test Strategy

### Unit tests (spec package)

- `ledger_test.go`: Parse well-formed ledger, parse ledger with mixed
  statuses, parse ledger with signed-off annotations, handle missing
  section, handle malformed tables, handle case variations (done/Done/DONE),
  handle empty exercise checkbox, handle exercise with detail.

- `audit_test.go`: Parse SHIP/clean report, parse SHIP/noteworthy report,
  parse HOLD report, handle missing file, handle malformed header.

### Integration tests (cli package)

- `verify_test.go`: End-to-end verify on a temp workspace with:
  - All gates pass → status flipped, spec archived
  - Missing ledger → FAIL, status unchanged
  - PARTIAL rows → FAIL, status unchanged
  - Missing audit → FAIL, status unchanged
  - HOLD audit → FAIL, status unchanged
  - `--force` → archives with warning despite failures
  - `--skip-tests` → Gate 4 skipped
  - `--json` → valid JSON output with all fields
  - Signed-off SKIPPED rows → pass Gate 1
  - Already-completed spec → idempotent (no error, no double-archive)

### Manual verification

Run `hero verify` on a real recently-delivered spec and confirm the gates
report accurately against the actual spec state.

## Risks

- **Existing completed specs.** Specs already archived via the old flow
  won't have audit reports. This is fine — verify only runs on specs being
  completed now, not retroactively.

- **Agent learns to game the ledger.** An agent could write a performative
  ledger that passes parsing but lies about evidence. The cold audit is
  the check against this — it reads the diff independently. If both the
  ledger and the audit are gamed, we have a deeper problem. But making the
  audit non-optional (Gate 2) raises the bar significantly.

- **Markdown parsing fragility.** Real-world ledgers will have formatting
  quirks. The parser needs to be tolerant — strip whitespace, handle
  missing pipes, accept case variations. Test against actual ledgers from
  recent deliveries.

- **Test command detection.** Auto-detecting the test command will miss
  some projects. The `hero.json` override handles this. When detection
  fails, Gate 4 reports SKIPPED rather than FAIL.
