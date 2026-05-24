---
title: hero search --json silently emits human text on the FTS5 path
slug: hero-search-json-flag-silently-ignored
type: bug
status: planning
severity: low
priority: P2
created: 2026-05-24
tags: [cli, search, output, contract, testing-gap]
---

# hero search --json silently emits human text on the FTS5 path

## Problem

`hero search "<query>" --json` is documented as "emit JSON (graph search only)" and is wired as a boolean flag in `internal/cli/search.go:58`. In practice, for plain-text queries that fall back to FTS5 (which is the path most queries land on today — see Repro), the `--json` flag is silently ignored. The command emits the regular human-readable tabular format and exits 0.

This is a contract violation in two directions:

1. Consumers that pass `--json` expecting machine output get human text and will fail to parse it. Worst-case they parse partial output and silently consume garbage.
2. There is no error, no warning to stderr, and no exit code change to signal the mismatch.

Surfaced during a session investigating whether `hero search` was returning a "default ranked list regardless of query." Filtering was actually working; the false-broken signal came from trying `--json` to inspect ranking and getting back human text that looked identical to the no-`--json` output, which deepened the distrust.

## Steps to Reproduce

```
hero search "domain swap" --json > /tmp/out.json
file /tmp/out.json
# Expected: ASCII/UTF-8 text containing valid JSON (e.g. `[{...}]`)
# Actual:   Unicode text — tabular human output with `>>>term<<<` highlights
#           and a trailing `20 result(s)` line. Not parseable as JSON.
```

## Expected Behavior

One of:

- **A.** `--json` honored on every output path (FTS5 + graph), emitting a stable JSON shape (`[{type, key, title, score, snippet?}, …]`). Consumers don't have to know which internal index served the query.
- **B.** `--json` rejects with a clear error when the resolved path can't honor it: `Error: --json is only supported for graph search results; this query routed through FTS5. Remove filter flags or upgrade the FTS5 path.` Exit non-zero.

Preferred: **A.** The whole point of `--json` is "I am a script, give me machine output" — silently picking which queries get to be machine-readable is a worse trap than not supporting it at all.

## Root Cause

`internal/cli/search.go:142-147`:

```go
// Graph results use the compact markdown format; FTS5 results use the
// tabular format. Source is homogeneous within a single Retrieve call.
if results[0].Source == "graph" {
    return printGraphResults(results, strings.Join(args, " "), searchBudget, searchJSON)
}
return printFTSResults(results)
```

`printFTSResults` does not accept the `asJSON` parameter and has no JSON branch. `searchJSON` is dropped on the FTS5 path. `runSearchFTS` (the `--file` / `--list` / `--cross-repo` path) likewise ignores `searchJSON`.

The empty-results path (`if len(results) == 0 { fmt.Println("No results found.") }`) also ignores `--json` — JSON consumers expect `[]`, not the string `"No results found."`.

## Why This Escaped Tests

`internal/cli/search_test.go` has **zero `--json` invocations.** The seven existing tests cover plain query, no results, `--file`, `--list`, `--list --type`, no-workspace error, and required-args. None exercise `--json` on any path.

Additionally, every test uses a 1–2 spec corpus, which means the graph-vs-FTS5 routing inside `retrieval.Retrieve` likely never engages the graph path during the suite — so even ordinary plain-text-search output formatting is exercised on a single path. The dual-path nature of the command is invisible to the test suite.

## Fix

1. **Plumb `asJSON` through both formatters.** `printFTSResults` and `runSearchFTS` accept `asJSON` and emit the same `[{type, key, title, score?, snippet?}, …]` shape as `printGraphResults`. Score may be `null` on the FTS5 path.
2. **Honor `--json` on empty results.** Emit `[]` instead of `No results found.` when `searchJSON` is set.
3. **Add tests:**
   - `TestSearchJSONFTS5Path` — query a small corpus, assert output parses as JSON array with expected keys.
   - `TestSearchJSONNoResults` — `--json` with a no-match query returns `[]`.
   - `TestSearchJSONListMode` — `--list --json` emits JSON.
   - Consider one test that builds a large enough corpus to engage the graph path and asserts JSON output there too. (May be deferred if graph-engagement threshold is unclear.)

## Out of Scope

- Redesigning the compact/full/expand tiering of `mcp__hero__hero_search` and the CLI default verbosity. Tracked separately; this bug is strictly about `--json` honoring its contract.
- Adding `--json` support to other Hero CLI commands that lack it.

## Acceptance Criteria

- `hero search "<any query>" --json` emits valid JSON on stdout for every routing path (FTS5, graph, list, file, cross-repo).
- `hero search "no-match-string" --json` emits `[]`.
- New tests cover at least the FTS5 path and the empty-results path; they fail against current `main` and pass after the fix.
