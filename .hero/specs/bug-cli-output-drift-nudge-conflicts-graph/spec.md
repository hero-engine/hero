---
title: CLI Output Drift — Nudge, Conflicts, and Graph Behavior Diverged from Tests
slug: bug-cli-output-drift-nudge-conflicts-graph
type: bug
status: completed
tags: [cli, output, regression, nudge, conflicts, graph]
created: 2026-05-01
horizon: now
---

## Symptom

Four tests in `internal/cli` fail because command output and behavior have drifted from the contracts encoded in tests:

1. **`TestNudgeNoContext`** — `hero relevant --files src/unrelated.go` prints `"No relevant context found for these files."` when the test expects empty output (silent on no-context).
2. **`TestNudgeNoWorkspace`** — `hero relevant --files src/main.go` outside a workspace prints `"No hero workspace here — nothing to surface."` when the test expects empty output (silent on no-workspace).
3. **`TestConflictsFound`** — `hero check conflicts feat-c` output no longer contains the `"1 conflict"` summary count line. Test expects a count line so callers can detect "any conflicts at all" without parsing the body.
4. **`TestGraphRequiresArg`** — `hero graph` (no slug) returns nil error when the test expects an error. Validation regressed.

## Root Cause

These are independent drifts in three CLI commands. Either tests were aspirational and the implementations regressed, or the implementations evolved (added explanatory output, relaxed validation) without updating the contracts. The tests encode user-facing expectations:

- **Silent nudge** — nudge runs in pre-commit hooks and ambient watch contexts. Loud output every time there's nothing relevant is noise. The test's `"should be silent"` rationale matches the design intent.
- **Conflicts count line** — scriptable detection of "any conflicts" needs a stable count token. Removing it forces callers to parse markup.
- **Graph requires slug** — `hero graph <slug>` is the documented usage; bare `hero graph` should error so users see the right thing.

## Suggested Fix Approach

Three small, independent edits — sequenced together because they all live in `internal/cli`:

1. **Make nudge silent on empty / no-workspace.** In [internal/cli/relevant.go](internal/cli/relevant.go) (or wherever the nudge/relevant command formats output), suppress the `"No relevant context found..."` and `"No hero workspace here..."` strings — return empty output and `nil` error in both cases. Keep the loud message only when there *is* relevant context to print.

2. **Restore the conflicts count line.** In `hero check conflicts` output formatting, append `"N spec conflict(s) — coordinate before proceeding."` (or the exact phrasing the test still expects: `"1 conflict"`). Read the test at [internal/cli/conflicts_test.go:87](internal/cli/conflicts_test.go:87) for the exact substring required.

3. **Validate graph slug.** In [internal/cli/graph.go](internal/cli/graph.go) (or its `RunE`), return an error when `len(args) == 0`. Use the same pattern as other slug-required commands in the package.

After each edit, run the matching `-run` filter to confirm green, then run the full `internal/cli` package tests to confirm no regression elsewhere.

## Acceptance Criteria

- WHEN `hero relevant --files <files>` runs and no context applies THE SYSTEM SHALL produce empty stdout and exit 0
- WHEN `hero relevant --files <files>` runs without a hero workspace THE SYSTEM SHALL produce empty stdout and exit 0
- WHEN `hero check conflicts <slug>` finds N>0 conflicts THE SYSTEM SHALL include a count line containing `"N conflict"` in stdout
- WHEN `hero graph` runs without a slug argument THE SYSTEM SHALL exit non-zero with a clear usage error
- WHEN the `internal/cli` package tests run THE SYSTEM SHALL pass `TestNudgeNoContext`, `TestNudgeNoWorkspace`, `TestConflictsFound`, and `TestGraphRequiresArg`

## Out of Scope

- Broader review of CLI output consistency or formatting standards (separate spec if needed)
- Adding new flags or behaviors to any of these commands
- Changing the conflicts detection logic itself — only the count-line in the output formatter
- Changing nudge's positive-output formatting (when there *is* context, the output stays as-is)

## Related

These failures predate the two-tier-mcp-responses and mcp-server-refactor work — they were observed on the baseline before that work landed. Tracked here so the test suite reaches green and future regressions are easier to spot.
