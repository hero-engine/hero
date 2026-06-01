---
title: Spec-as-Living-Contract — Continuous Criteria Validation
slug: living-contract
type: feature
status: completed
priority: P0
tags: [specs, ears, contract, verification, regression]
created: 2026-04-22
relations:
  - target: hero-killer-features
    kind: parent
  - target: ears-acceptance-criteria
    kind: depends-on
  - target: spec-drift-detection
    kind: related
  - target: hero-pulse
    kind: related
horizon: now
completed_at: 2026-05-18T19:25:38Z
---

## Goal

Keep acceptance criteria alive after a spec is delivered. Each EARS criterion
in a completed spec gets a `verified_by:` annotation linking it to the test
file and function that proves it. `hero contract` commands let engineers see
contract status, link criteria to tests, and detect regressions across the
entire completed-spec corpus without leaving the CLI.

## Problem

Today a spec's acceptance criteria are write-once artifacts. Once `hero deliver`
moves a spec from `planning/` to `specs/`, the criteria become inert prose --
nobody checks whether the tests that proved them still pass. A passing CI run
proves *something* works, but not *which criteria* are still satisfied. When a
refactor breaks the streaming behavior promised by criterion 3 of `csv-export`,
nobody notices until a customer files a bug.

Hero already has the pieces: EARS-parsed criteria with structured `Trigger` and
`Behavior` fields, `hero test generate` that creates test files from those
criteria, and `hero pulse` that reports sprint health. What's missing is the
link between a delivered criterion and the test that proves it -- and a way to
run just those linked tests to check for regressions.

## Design

### Storage model: the spec IS the contract

No new files, no separate database. When a spec is delivered and its criteria
are verified, the acceptance criteria section in `specs/<slug>/spec.md` gets
enriched with inline `verified_by:` annotations:

```markdown
## Acceptance Criteria

- WHEN export size exceeds 10MB THE SYSTEM SHALL stream rather than buffer
  verified_by: e2e/csv-export.spec.ts::streams large exports
- WHILE a sync is in flight THE SYSTEM SHALL block concurrent sync attempts
  verified_by: internal/sync/sync_test.go::TestBlockConcurrentSync
- THE SYSTEM SHALL log every failed login attempt
  verified_by: internal/auth/auth_test.go::TestFailedLoginLogging
```

Each `verified_by:` line is indented under its criterion bullet. The format is
`<test-file>::<test-name>`. A criterion may have zero annotations (unlinked),
one, or multiple (covered by several tests). The parser treats lines beginning
with `verified_by:` (after whitespace) as annotations belonging to the
preceding criterion.

### Parser changes

`internal/spec/spec.go` extends the existing `Criterion` struct:

```go
type Criterion struct {
    Raw        string
    Kind       CriterionKind
    Trigger    string
    Behavior   string
    VerifiedBy []TestLink   // NEW — populated from verified_by: annotations
}

type TestLink struct {
    File string // e.g. "e2e/csv-export.spec.ts"
    Name string // e.g. "streams large exports"
}
```

`AcceptanceCriteria()` already iterates the criteria section line by line. It
gains a look-ahead for `verified_by:` lines following each bullet.

### `hero contract` command

Three subcommands under a new `hero contract` top-level command:

```
hero contract <slug>                                    # show contract status
hero contract link <slug> <index> <file>::<test>        # link criterion to test
hero contract check [--slug <slug>]                     # run linked tests, report regressions
```

#### `hero contract <slug>`

Displays a table of all acceptance criteria for the spec, their EARS
classification, and link status:

```
csv-export (completed, 5 criteria)
  1. [event]  WHEN export exceeds 10MB ... SHALL stream
     linked: e2e/csv-export.spec.ts::streams large exports
  2. [state]  WHILE sync is in flight ... SHALL block
     linked: internal/sync/sync_test.go::TestBlockConcurrentSync
  3. [ubiq]   THE SYSTEM SHALL log failed logins
     UNLINKED
  4. [event]  WHEN user cancels export ... SHALL clean up temp files
     linked: e2e/csv-export.spec.ts::cleans up on cancel
  5. [unwant] IF export target is read-only ... SHALL fail with message
     linked: e2e/csv-export.spec.ts::rejects read-only target

Contract coverage: 4/5 (80%)
```

Exits 0 if all criteria are linked, 1 if any are unlinked.

#### `hero contract link <slug> <index> <file>::<test>`

Writes a `verified_by:` annotation into the spec file under criterion number
`<index>` (1-based). The command:

1. Parses the spec and locates criterion `<index>`.
2. Validates that `<file>` exists relative to the project root.
3. Appends `  verified_by: <file>::<test>` on the line after the criterion.
4. Writes the updated spec back to disk.

If the link already exists, exits with a message and no change.

#### `hero contract check [--slug <slug>]`

Runs all linked tests and reports regressions:

1. If `--slug` is given, collects `TestLink`s from that spec only. Otherwise
   collects from all completed specs in `specs/`.
2. Groups test links by runner type (Go tests detected by `_test.go` suffix,
   Playwright by `.spec.ts` / `.test.ts`, etc.).
3. Invokes each runner for its batch of tests.
4. Maps pass/fail results back to criteria and specs.

Output:

```
Contract check — 3 specs, 12 criteria, 12 linked tests

csv-export .............. 5/5 PASS
auth-middleware ......... 3/4 FAIL
  REGRESSION: criterion 2 — WHEN token expires THE SYSTEM SHALL redirect
    FAIL: e2e/auth.spec.ts::redirects on expired token
sync-engine ............. 3/3 PASS

Result: 1 regression in 1 spec
```

Exit code: 0 = all pass, 1 = regressions detected.

### `hero pulse` regression section

`internal/pulse/pulse.go` gains a `Regressions` field on `PulseData`:

```go
type RegressionEntry struct {
    Slug      string
    Title     string
    Criterion string
    TestFile  string
    TestName  string
}
```

When `hero pulse` runs, it checks for completed specs with linked tests and
reports any that are failing. The regressions section appears after the
existing "At Risk" section:

```
Regressions (1):
  ! auth-middleware — criterion 2 failing (e2e/auth.spec.ts::redirects on expired token)
```

### MCP tool

`internal/serve/mcp.go` registers a `hero_contract` tool:

```json
{
  "name": "hero_contract",
  "description": "Show contract status or run regression checks for completed specs",
  "inputSchema": {
    "type": "object",
    "properties": {
      "action": { "type": "string", "enum": ["status", "check", "link"] },
      "slug": { "type": "string" },
      "criterion_index": { "type": "integer" },
      "test_ref": { "type": "string", "description": "file::testname" }
    },
    "required": ["action"]
  }
}
```

This lets delivery agents check contract coverage and link tests
programmatically as part of the `/deliver` workflow.

### Test runner abstraction

`internal/contract/contract.go` detects the runner from the test file path:

| Pattern | Runner | Invocation |
|---|---|---|
| `*_test.go` | Go | `go test -run <TestName> ./<package>` |
| `*.spec.ts`, `*.test.ts` | Playwright | `npx playwright test <file> -g "<name>"` |
| `*.test.js`, `*.spec.js` | Jest/Vitest | `npx vitest run <file> -t "<name>"` |
| `*.py::*` | pytest | `pytest <file>::<name>` |

The runner table is extensible via `hero.json`:

```json
{
  "contract": {
    "runners": {
      "*.spec.ts": "npx playwright test {file} -g \"{name}\""
    }
  }
}
```

## Changes

- `internal/contract/contract.go` — `TestLink` type, spec parser for `verified_by:` annotations, test runner dispatch, regression reporting
- `internal/contract/contract_test.go` — unit tests for parsing, linking, runner detection, regression mapping
- `internal/cli/contract.go` — `hero contract` command with `status`, `link`, and `check` subcommands
- `internal/cli/root.go` — register `contractCmd`
- `internal/spec/spec.go` — extend `Criterion` with `VerifiedBy []TestLink`, update `AcceptanceCriteria()` parser
- `internal/serve/mcp.go` — register `hero_contract` tool
- `internal/pulse/pulse.go` — add `Regressions []RegressionEntry` to `PulseData`, populate during pulse collection

## Acceptance Criteria

- WHEN a completed spec contains `verified_by:` annotations under its acceptance criteria THE SYSTEM SHALL parse them into `TestLink` values on the corresponding `Criterion`
- WHEN `hero contract <slug>` runs against a completed spec THE SYSTEM SHALL display each criterion with its EARS classification, link status, and overall coverage percentage
- WHEN `hero contract link <slug> <index> <file>::<test>` runs THE SYSTEM SHALL write a `verified_by:` annotation into the spec file under the specified criterion
- WHEN `hero contract link` is called with a test file that does not exist THE SYSTEM SHALL exit non-zero with an error message
- WHEN `hero contract check` runs THE SYSTEM SHALL execute all linked tests grouped by runner type and report pass/fail results mapped back to criteria and specs
- WHEN `hero contract check` detects a failing linked test THE SYSTEM SHALL exit non-zero and identify the regressed criterion, spec, and test
- WHEN `hero pulse` runs and completed specs have failing linked tests THE SYSTEM SHALL include a Regressions section listing each failure
- WHEN the `hero_contract` MCP tool is called with action `status` THE SYSTEM SHALL return the same structured payload as `hero contract <slug> --format json`
- IF a criterion already has a `verified_by:` link for the same file and test name THEN THE SYSTEM SHALL skip the duplicate and exit cleanly

## Boundaries

- Does **not** auto-generate `verified_by:` links -- the delivery agent or engineer creates them via `hero contract link`
- Does **not** replace `hero test generate` -- that command creates test files; this feature links them back to criteria
- Does **not** require CI integration -- `hero contract check` runs locally; CI-aware contract checks belong to the environment-awareness feature
- Does **not** validate test correctness -- it only checks pass/fail; whether a test *actually* proves the criterion is a human/agent judgment
- Does **not** modify specs in `planning/` -- contract annotations only apply to completed specs in `specs/`
