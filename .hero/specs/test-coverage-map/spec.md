---
title: Test Coverage Map — Criterion-to-Test Traceability
slug: test-coverage-map
type: feature
status: completed
priority: P1
tags: [testing, coverage, traceability, specs, mcp]
created: 2026-04-22
relations:
  - target: hero-killer-features
    kind: parent
  - target: living-contract
    kind: related
horizon: now
---

## Goal

Map which acceptance criteria in a spec have test coverage and which do not,
so that agents and engineers can answer "which criteria are tested?" without
manually cross-referencing spec bullets against test files.

## Problem

Hero can generate test scaffolds from acceptance criteria (`hero test generate`)
and detect when code drifts from a spec (`hero drift`), but there is no bridge
between the two: nothing answers "does criterion 3 actually have a test?" The
agent's biggest friction at delivery time is not knowing which criteria already
have tests versus which are untested. Today the only option is to read every
test file and mentally map assertions back to spec bullets. This is slow,
error-prone, and invisible to tooling.

## Design

### `hero coverage <slug>`

Analyzes a spec's acceptance criteria against test files in the project. For
each criterion, extracts keywords (reusing the keyword extraction from
`internal/drift/`) and searches test files for matching test names, assertion
text, and describe-block labels. Produces a per-criterion coverage verdict.

```
hero coverage csv-export
```

Default human output:

```
csv-export — CSV Export with Streaming Support
  5/7 criteria have test coverage
  Gaps:
    criterion 3: "WHEN export size exceeds 10MB THE SYSTEM SHALL stream
                   rather than buffer"
      -> no test mentions "streaming" or "buffer"
    criterion 6: "WHEN rate limiting is active THE SYSTEM SHALL return 429"
      -> no test for "rate_limiting" or "429"
```

```
hero coverage <slug>              # coverage report for one spec
hero coverage --all               # all specs with acceptance criteria
hero coverage --format json       # machine-readable output
hero coverage --test-dir <path>   # override test file discovery root
```

Exit code: 0 = all criteria covered, 1 = gaps exist.

### How coverage is computed

All analysis is local and heuristic -- no LLM calls, no test execution.

1. **Load spec** -- parse the spec file via `spec.ParseFile`, extract
   acceptance criteria via `AcceptanceCriteria()`.
2. **Discover test files** -- walk the project tree for files matching
   `*_test.go`, `*.test.ts`, `*.test.js`, `*.spec.ts`, `*.spec.js`,
   `*_test.py`, `test_*.py`. Respects `.gitignore`.
3. **Extract keywords per criterion** -- reuse `internal/drift.extractKeywords`
   (identifier-shaped tokens, stop-word filtered). For EARS criteria, keywords
   come from both the Trigger and Behavior clauses.
4. **Search test files** -- for each criterion's keywords, scan test file
   contents (function/method names, describe/it block labels, assertion
   strings). A criterion is "covered" when at least one keyword matches in
   at least one test file.
5. **Build report** -- aggregate per-criterion verdicts into a `CoverageReport`
   with counts, gap list, and per-criterion detail.

### Data structures

```go
// CriterionCoverage tracks test coverage for a single acceptance criterion.
type CriterionCoverage struct {
    Index      int      `json:"index"`
    Raw        string   `json:"raw"`
    Kind       string   `json:"kind"`       // EARS kind or "freeform"
    Covered    bool     `json:"covered"`
    MatchFiles []string `json:"match_files,omitempty"` // test files that matched
    Keywords   []string `json:"keywords"`              // extracted search terms
    Detail     string   `json:"detail,omitempty"`      // explanation when uncovered
}

// CoverageReport is the coverage analysis for a single spec.
type CoverageReport struct {
    Slug       string              `json:"slug"`
    Title      string              `json:"title"`
    Total      int                 `json:"total"`
    Covered    int                 `json:"covered"`
    Gaps       int                 `json:"gaps"`
    Criteria   []CriterionCoverage `json:"criteria"`
    ExitCode   int                 `json:"exit_code"`
}
```

### MCP tool

```json
{
  "name": "hero_coverage",
  "description": "Report which acceptance criteria have test coverage and which are untested",
  "inputSchema": {
    "type": "object",
    "properties": {
      "slug": { "type": "string", "description": "Spec slug to analyze" },
      "all": { "type": "boolean", "description": "Analyze all specs with acceptance criteria" },
      "test_dir": { "type": "string", "description": "Override test file discovery root" }
    }
  }
}
```

Returns the same structured payload as `--format json`. Agents can call this
at session start to know which criteria still need tests, or at delivery end to
verify completeness.

### Keyword extraction reuse

The `extractKeywords` function in `internal/drift/drift.go` already does
identifier-shaped token extraction with stop-word filtering. The coverage
package imports and reuses this logic rather than duplicating it. If the
function is currently unexported, it is exported as `ExtractKeywords` and the
drift package's own callers are updated to use the new name.

### Test file discovery

Test files are discovered by walking the project root (or `--test-dir` if
specified) and matching common test file naming conventions across Go,
TypeScript/JavaScript, and Python. Files inside `node_modules`, `vendor`, and
`.git` directories are excluded.

## Changes

- `internal/coverage/coverage.go` -- `CoverageReport`, `CriterionCoverage` structs, `Analyze` and `AnalyzeAll` functions, keyword-to-test matching, test file discovery, text and JSON renderers
- `internal/coverage/coverage_test.go` -- table-driven tests: full coverage, partial gaps, no tests found, EARS vs freeform criteria, keyword extraction edge cases
- `internal/cli/coverage.go` -- `hero coverage` command with `--all`, `--format`, `--test-dir` flags
- `internal/cli/root.go` -- register `coverageCmd`
- `internal/serve/mcp.go` -- register `hero_coverage` tool

## Acceptance Criteria

- WHEN `hero coverage <slug>` runs against a spec whose criteria all have matching test files THE SYSTEM SHALL report full coverage with exit code 0
- WHEN one or more criteria have no keyword matches in any test file THE SYSTEM SHALL list each uncovered criterion with its index, text, and the keywords that were searched, and exit code 1
- WHEN an EARS criterion has both Trigger and Behavior clauses THE SYSTEM SHALL extract keywords from both clauses for matching
- WHEN a criterion is freeform (non-EARS) THE SYSTEM SHALL fall back to extracting keywords from the full criterion text
- WHEN `hero coverage --all` runs THE SYSTEM SHALL evaluate every spec that has acceptance criteria and produce an aggregate report
- WHEN the `hero_coverage` MCP tool is called with a slug THE SYSTEM SHALL return the same JSON payload as `hero coverage <slug> --format json`
- THE SYSTEM SHALL discover test files by filename convention without requiring configuration, supporting at minimum `*_test.go`, `*.test.ts`, `*.spec.ts`, and `*_test.py` patterns

## Boundaries

- Does **not** replace actual code-coverage tools -- this is spec-criteria coverage (which criteria have any related test), not line-level or branch-level code coverage
- Does **not** require running tests -- all analysis is static, based on test file content
- Does **not** modify test files -- read-only analysis
- Does **not** call an LLM -- all matching is keyword/heuristic based
- Does **not** persist coverage state -- recomputed on demand
- Does **not** guarantee that a matched test actually validates the criterion -- a keyword match is a heuristic signal, not a proof of correctness
