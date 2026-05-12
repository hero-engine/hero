---
title: "Cost/Effort Calibration \u2014 Estimated vs Actual Effort Tracking"
type: feature
status: completed
priority: P2
tags: [cost, calibration, estimation, velocity, metrics]
created: 2026-04-22
relations:
  - target: hero-killer-features
    kind: parent
  - target: hero-velocity
    kind: related
horizon: now
---

## Goal

Make `hero cost` learn from this project's delivery history so estimates
converge on reality over time. After enough specs are completed, the cost
heuristic stops guessing and starts reflecting how THIS team actually
delivers in THIS codebase.

## Problem

`hero cost` today uses a fixed heuristic: file count, section count,
word count, dependency count, type multiplier. It already does a basic
calibration blend against completed spec averages, but it treats all spec
types the same, ignores wall-clock duration, ignores commit volume, and
has no memory between runs. A feature spec that lists 6 files could take
an afternoon or a week depending on the codebase, the team, and the
problem domain. The current estimate cannot distinguish these cases.

Teams that have delivered 30+ specs have a rich signal sitting in
`.hero/specs/` (completed specs with timestamps), git history (commits,
file churn, time ranges), and `events.log` (session counts, agent
claims). None of that feeds back into the estimator.

## Design

### Calibration data model

A new `calibration.json` file in `.hero/knowledge/` stores derived
metrics from the completed spec corpus. It is rebuilt on demand, not
maintained incrementally.

```json
{
  "generated": "2026-04-22T14:00:00Z",
  "spec_count": 42,
  "global_ratio": 1.25,
  "by_type": {
    "feature": { "count": 28, "avg_ratio": 1.45, "avg_days": 3.2, "avg_commits": 12 },
    "bug":     { "count": 10, "avg_ratio": 0.78, "avg_days": 0.8, "avg_commits": 3 },
    "chore":   { "count": 4,  "avg_ratio": 1.02, "avg_days": 1.1, "avg_commits": 5 }
  },
  "entries": [
    {
      "slug": "csv-export",
      "type": "feature",
      "estimated_points": 8.5,
      "actual_signals": {
        "days_elapsed": 4,
        "commits": 14,
        "files_changed": 9,
        "criteria_count": 7,
        "sessions": 3
      },
      "actual_points": 12.2,
      "ratio": 1.44
    }
  ]
}
```

**`ratio`** = actual_points / estimated_points. Values > 1.0 mean the
estimate was optimistic; < 1.0 means it was pessimistic.

### Actual effort derivation

For each completed spec, actual effort is computed from:

| Signal | Source | Weight |
|---|---|---|
| Days elapsed | Spec `created` date to completion timestamp in git | 0.25 |
| Commit count | `git log --oneline` for files listed in `## Changes` | 0.25 |
| Files actually changed | `git diff --stat` between creation and completion | 0.25 |
| Criteria addressed | Count of acceptance criteria in completed spec | 0.15 |
| Session count | `events.log` entries with matching slug | 0.10 |

These are normalized against the project corpus (z-score style) and
combined into an `actual_points` value on the same scale as the existing
estimated points.

### `hero cost calibrate` subcommand

```
hero cost calibrate              # rebuild calibration.json from completed specs + git
hero cost calibrate --verbose    # print per-spec breakdown during rebuild
hero cost calibrate --min-specs 5  # override the 10-spec default minimum
```

Reads every completed spec in `.hero/specs/`, queries git history for
each, computes actual effort, compares to the estimate that would have
been produced by the raw heuristic, and writes
`.hero/knowledge/calibration.json`.

### `hero cost --calibrated` flag

```
hero cost csv-export                # uses calibration if available (default on)
hero cost csv-export --no-calibrated  # raw heuristic only
hero cost csv-export --calibrated   # explicit: fail if calibration data missing
```

When calibration data exists and the completed spec count meets the
minimum threshold (default 10), `--calibrated` is the default behavior.
The estimate is adjusted by the type-specific ratio from
`calibration.json`:

```
calibrated_points = raw_points * by_type[spec.type].avg_ratio
```

If the spec type has fewer than 3 completed examples, falls back to
`global_ratio`.

### `hero cost --history` flag

```
hero cost --history             # show calibration summary
hero cost --history --format json  # machine-readable
```

Human output:

```
Calibration data (42 completed specs)
─────────────────────────────────────
Feature specs:  1.45x estimated (28 specs, avg 3.2 days, avg 12 commits)
Bug fixes:      0.78x estimated (10 specs, avg 0.8 days, avg 3 commits)
Chores:         1.02x estimated (4 specs, avg 1.1 days, avg 5 commits)

Overall:        1.25x estimated

Features in this repo take ~45% longer than raw estimates suggest.
Bug fixes are consistently overestimated.
```

### Integration with existing `hero cost`

The current `calibrate()` function in `internal/cli/cost.go` does a
simple average-files blend. The new calibration replaces this with the
richer `calibration.json`-backed approach when the file exists and has
enough data. The existing blend remains as the fallback when calibration
data is insufficient.

### Threshold behavior

Calibration activates only when:
- `.hero/knowledge/calibration.json` exists
- It contains at least 10 completed specs (configurable via
  `cost.min_calibration_specs` in `hero.json`)
- The file was generated within the last 30 days (staleness guard)

Below the threshold, `hero cost` behaves exactly as today with a note:
"Calibration requires 10 completed specs (currently 3). Run `hero cost
calibrate` after completing more specs."

## Changes

- `internal/cost/calibration.go` -- calibration data model, actual-effort
  derivation from git history, calibration.json read/write, ratio
  computation per type
- `internal/cost/calibration_test.go` -- table-driven tests for effort
  derivation, ratio computation, threshold gating, staleness checks
- `internal/cli/cost.go` -- add `--calibrated`/`--no-calibrated`,
  `--history` flags, `calibrate` subcommand, integrate calibration into
  `estimateSpec`
- `internal/cli/root.go` -- register `calibrate` subcommand under `cost`
  if needed

## Acceptance Criteria

- WHEN `hero cost calibrate` runs with at least 10 completed specs in `.hero/specs/` THE SYSTEM SHALL generate `.hero/knowledge/calibration.json` containing per-type ratios derived from estimated vs actual effort signals
- WHEN `hero cost <slug>` runs and valid calibration data exists with at least 10 completed specs THE SYSTEM SHALL apply the type-specific ratio from calibration.json to adjust the raw heuristic estimate
- WHEN `hero cost <slug>` runs and calibration data contains fewer than 3 completed specs of the target type THE SYSTEM SHALL fall back to the global ratio rather than the type-specific ratio
- WHEN `hero cost <slug> --no-calibrated` runs THE SYSTEM SHALL produce the raw heuristic estimate without any calibration adjustment, regardless of available calibration data
- WHEN `hero cost --history` runs THE SYSTEM SHALL display per-type calibration ratios, average durations, average commit counts, and a plain-English summary of estimation accuracy
- WHEN fewer than 10 completed specs exist and `hero cost <slug>` runs THE SYSTEM SHALL use the raw heuristic and display a message indicating how many more completed specs are needed before calibration activates
- WHEN `hero cost calibrate` computes actual effort THE SYSTEM SHALL derive it from git history and event logs without requiring any external time-tracking tool or manual input

## Boundaries

- Does **not** require external time-tracking tools -- all signals come from git history, spec metadata, and events.log
- Does **not** track individual developer performance -- calibration ratios are team aggregates only
- Does **not** block delivery based on cost estimates -- estimates are advisory, never gatekeeping
- Does **not** activate calibration below 10 completed specs -- insufficient data produces misleading ratios
- Does **not** call an LLM -- all computation is deterministic from local data
