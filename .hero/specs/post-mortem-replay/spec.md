---
title: "Post-mortem Replay — Compare Plan vs Actual Outcome"
type: feature
status: completed
tags: [cli, analytics]
created: 2026-04-12
horizon: now
---

## Goal

The `hero replay` command compares a spec's original plan against the actual implementation outcome, providing a structured post-mortem with file analysis, timeline, and accuracy metrics.

## Design

Replay analyzes completed specs by comparing what was planned (the spec's changes list and acceptance criteria) against what actually happened (git history and file system state). The output includes:

- **File analysis** — which planned files were actually created/modified, which unexpected files were touched, which planned files were never changed
- **Timeline** — chronological view of commits and spec status transitions from start to completion
- **Accuracy metrics** — percentage of planned changes that matched actual changes, scope creep indicator, estimation accuracy
- **Divergence summary** — narrative explanation of where the implementation diverged from the plan and possible reasons

### Usage

```
hero replay <slug>
```

The command requires the spec to be in `completed` status. It cross-references the spec's `Changes` section with git log entries that touched those files during the spec's active period.

## Changes

- `internal/cli/replay.go` — `hero replay` command implementation, git log analysis, comparison logic
- `internal/cli/replay_test.go` — tests for replay analysis, file matching, accuracy calculation

## Acceptance Criteria

- `hero replay <slug>` produces a post-mortem for a completed spec
- Output includes planned vs actual file changes comparison
- Output includes a timeline of commits and status transitions
- Accuracy metrics quantify how closely implementation matched the plan
- Command errors gracefully if the spec is not in completed status
- Tests cover file matching, accuracy calculation, and edge cases
