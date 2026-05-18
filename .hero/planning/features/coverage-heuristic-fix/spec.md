---
title: Coverage Heuristic Fix — Rust Discovery and Test-Name Matching
slug: coverage-heuristic-fix
type: feature
status: completed
status_verified: "2026-05-10 by go test ./internal/coverage/... — all 11 tests pass; smoke-checked against test-coverage-map (7/7 strong) and self (4 strong/5 weak)"
priority: P1
tags: [coverage, testing, rust, heuristic, mcp]
created: 2026-05-10
relations:
  - target: test-coverage-map
    kind: enhances
horizon: now
mission_alignment: |
  `hero coverage` exists so an agent can answer "which criteria are
  tested?" without reading every test file. Today it lies about Rust
  (zero discovery → 0/N covered even when `cargo test` passes) and is
  noisy in every other language (any single keyword substring counts as
  coverage). Agents currently second-guess the tool and report it as
  unreliable in delivery summaries. Fixing this makes the next session
  start smarter — the corpus stops disagreeing with itself.
principles_check: |
  Serves "the model only knows what someone thinks to inject": coverage
  is a piece of context the agent cannot reconstruct cheaply, so it
  must be trustworthy. Risks "it just works" if the new matcher becomes
  too strict and starts under-reporting; mitigated by keeping the
  one-keyword fallback when test-name extraction yields nothing, plus
  surfacing matched test names so the agent can sanity-check.
---

## Goal

Make `hero coverage` produce signal an agent can trust:

1. **Discover Rust tests** — `tests/*.rs` and `*.rs` files containing
   `#[test]` or `#[cfg(test)]` are recognized as test files.
2. **Match against test names, not whole-file substrings** — extract
   per-language test identifiers (Go `func Test...`, Rust `#[test] fn
   ...`, JS/TS `it(...)` / `test(...)` / `describe(...)`, Python `def
   test_...`) and require ≥2 keyword overlaps with a single test name
   to call a criterion covered. Fall back to the existing one-keyword
   substring match against the full file only when no test-name match
   succeeds, and label the verdict as `weak` so the agent knows.
3. **Show the matched test name(s)** in the report so a human or agent
   can see *why* a criterion was called covered.

## Problem

Two failure modes, both observed in real delivery sessions:

**Rust invisibility.** `internal/coverage/coverage.go:17-26` only matches
`_test.go`, `.test.{ts,js,tsx,jsx}`, `.spec.{ts,js,tsx,jsx}`, `_test.py`,
`test_*`. Rust unit tests live inside source `.rs` files in a
`#[cfg(test)] mod tests { ... }` block; integration tests live at
`tests/*.rs` with no naming convention. Neither matches. Result on a
Rust-touching delivery just now: `cargo test` passed 12/12, but
`hero_coverage` reported 0/12 and the delivering agent had to
hand-annotate "the tool is heuristic and missed the actual tests."

**Shallow matcher.** `keywordsMatch` at line 187 returns true if *any
single* keyword from the criterion appears as a lowercase substring
anywhere in a test file. A criterion mentioning "config" is "covered"
by every test file that touches config; a criterion phrased "stale
warning when index outdated" misses a test named
`test_warns_on_outdated_index` because no single keyword like
"stale-warning" appears verbatim. Both directions of error are common.

## Design

### Rust test discovery

In `discoverTestFiles`, after the existing suffix/prefix check, add a
content-sniff branch for `.rs` files:

```go
if strings.HasSuffix(name, ".rs") {
    // Always include anything under a tests/ directory.
    if pathContainsDir(path, "tests") {
        files = append(files, path)
        return nil
    }
    // Otherwise sniff the file head for inline test markers.
    if rustHasTestMarker(path) {
        files = append(files, path)
    }
    return nil
}
```

`rustHasTestMarker` reads up to the first 64KiB of the file and returns
true if it contains `#[test]`, `#[cfg(test)]`, or `#[tokio::test]`.
This is cheap — most `.rs` files are well under that size and the
sniff is one pass. Files that grow past 64KiB without a top-of-file
test marker are extremely unlikely to be test files.

`tests/` directory matching is path-segment based, not substring (so
`src/integration_tests/foo.rs` is *not* auto-included; it must contain
a marker).

### Per-language test-name extraction

Introduce `extractTestNames(path, content) []string` that returns the
identifier and any string-literal label associated with each test in
the file:

| Language | Pattern |
|---|---|
| Go (`*_test.go`) | `func (Test\|Benchmark\|Example)\w+` → captured name |
| Rust (`*.rs` with markers) | `#[test]\|#[tokio::test]\|#[cfg(test)]` followed by `fn (\w+)` within ~3 lines → captured name |
| JS/TS (`*.{test,spec}.{ts,tsx,js,jsx}`) | `(it\|test\|describe)\s*\(\s*["'\`]([^"'\`]+)` → captured label, plus any `function (\w+)` / `const (\w+) =` adjacent |
| Python (`*_test.py`, `test_*.py`) | `def (test_\w+)` → captured name |

Names are normalized: lowercased, snake-case and kebab-case both
accepted, stop-words stripped through the existing
`drift.ExtractKeywords` pipeline so matching is symmetric with criterion
keywords.

The extractors are intentionally regex-based, not full parsers. This
keeps the dependency surface zero and matches the "all analysis is
local and heuristic" boundary of the parent spec. Edge cases (macro
attributes split across lines, dynamically generated test names) are
acceptable misses — the file-level fallback (below) catches them.

### Two-tier matching

For each criterion:

1. **Strong match** — for each extracted test name in the project,
   count how many criterion keywords appear in the test name (after
   the same normalization). If any single test name has ≥2 keyword
   overlaps, the criterion is `covered: true` with `match_strength:
   "strong"` and the matching test name(s) are recorded in
   `MatchTests`.

2. **Weak match (fallback)** — only if no strong match was found, fall
   back to the existing whole-file ≥1-keyword substring check. If it
   matches, the criterion is `covered: true` with `match_strength:
   "weak"` and the matching file(s) appear in `MatchFiles`. The text
   report flags weak matches with `(weak)`.

3. **No match** — `covered: false`, listed under Gaps as today.

When a criterion has only one extracted keyword (which happens for
short EARS triggers), the strong-match threshold drops to 1 — otherwise
no criterion with a single keyword could ever match strongly.

### Output changes

`CriterionCoverage` gains:

```go
MatchStrength string   `json:"match_strength,omitempty"` // "strong" | "weak"
MatchTests    []string `json:"match_tests,omitempty"`    // test names that matched (strong only)
```

Text rendering:

```
csv-export — CSV Export with Streaming Support
  6/7 criteria have test coverage (5 strong, 1 weak)
  Gaps:
    criterion 6: "WHEN rate limiting is active THE SYSTEM SHALL return 429"
      -> no test mentions "rate_limiting" or "429"
  Weak matches (filename only, test names didn't match):
    criterion 4: matched export_test.go on keyword "export"
```

### Boundaries

- Does **not** parse Rust/Go/TS/Python — regex extraction only. Macros,
  dynamic test generation, table-driven test names that come from
  variables — all accepted as misses, caught by the weak fallback when
  the keywords appear in file body.
- Does **not** run any tests. Same as parent spec.
- Does **not** replace real coverage tools. A strong match is a
  plausible correspondence, not a proof of correctness.
- Does **not** support Rust doctests (`/// ```` blocks) — out of scope.
- Does **not** change the MCP tool surface. The JSON shape gains two
  optional fields; existing consumers ignore them.

## Acceptance Criteria

- WHEN `hero coverage` runs in a project containing Rust integration
  tests under `tests/*.rs` THE SYSTEM SHALL include those files in
  test discovery
- WHEN `hero coverage` runs in a project containing `.rs` source files
  with `#[test]` or `#[cfg(test)]` markers THE SYSTEM SHALL include
  those files in test discovery
- WHEN a `.rs` file under a non-`tests/` directory contains no test
  markers THE SYSTEM SHALL NOT include it in test discovery
- WHEN a criterion's keywords have ≥2 overlaps with the normalized
  identifier or label of a single discovered test THE SYSTEM SHALL
  mark the criterion `covered` with `match_strength: "strong"` and
  list the matching test name(s) in `match_tests`
- WHEN a criterion has only one extracted keyword and that keyword
  appears in a test name THE SYSTEM SHALL mark the criterion strongly
  covered (single-keyword threshold)
- IF no test name matches a criterion's keywords strongly THEN THE
  SYSTEM SHALL fall back to the existing one-keyword whole-file
  substring check and mark a positive result as `match_strength:
  "weak"`
- WHEN a criterion has no strong and no weak match THE SYSTEM SHALL
  list it under Gaps with the keywords that were searched, exactly
  as today
- WHEN the text report contains weak matches THE SYSTEM SHALL display
  them in a separate "Weak matches" section beneath Gaps so they are
  visually distinct from strong coverage
- THE SYSTEM SHALL preserve the existing JSON schema for
  `CoverageReport` and `CriterionCoverage`, adding only the optional
  `match_strength` and `match_tests` fields

## Changes

- `internal/coverage/coverage.go` — extend `discoverTestFiles` with
  Rust handling; add `rustHasTestMarker`; add `extractTestNames`
  dispatching by file extension; replace `keywordsMatch` call site in
  `analyzeSpec` with the two-tier match; add `MatchStrength` and
  `MatchTests` to `CriterionCoverage`; update `FormatText` to render
  weak matches in their own section
- `internal/coverage/coverage_test.go` — add cases: Rust integration
  test discovered, Rust source with `#[test]` discovered, Rust source
  without markers skipped, strong match against Go `func TestFoo`,
  strong match against Rust `#[test] fn foo`, strong match against
  TS `it("...")` / `describe("...")`, strong match against Python
  `def test_foo`, weak fallback when keywords only appear in file
  body, single-keyword criterion strong-matches on one overlap, no
  match path unchanged
- `internal/coverage/testdata/` — fixture files for the language
  matrix (one per language, all under the package's test data so
  the discovery walker can be pointed at them via `testDir`)

## Mission Fit

> "Does this make the next agent session start smarter than the last
> one ended — and does it raise the floor for everyone?"

Yes. Today the agent finishes a Rust delivery and has to manually warn
the user "the coverage tool missed it, but I really did test it."
After this fix, `hero_coverage` returns the correct verdict and the
agent's summary doesn't need a workaround paragraph. The floor rises
specifically for anyone working in a Rust repo (currently 0% useful)
and rises modestly for everyone else (fewer false-positive coverage
claims).
