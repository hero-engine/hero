---
description: Generate a weighted pipeline forecast grouped by stage, rep, and time period.
---
Route this forecast request to the `forecast-analyst` agent.

**Before starting**, load the `forecast-methodology` skill. It contains the
weighted pipeline formula, coverage ratio targets, commit/best-case/upside
definitions, and slippage risk signals.

**Parse the period argument**:
- `--period Q3 2026` — forecast for the named quarter
- `--period "July 2026"` — single-month forecast
- No argument — default to the current quarter

**Gather pipeline data** by reading all open deal specs:
```
hero search --type deal --status "qualifying,demo,proposal,negotiation"
```
For each deal, extract: `arr`, `probability`, `close_date`, `stage`,
`owner`, `meddpicc_score`, `company`.

**Delegate to `forecast-analyst`** with:
- The list of open deals and their fields
- The period requested
- The forecast methodology from `hero.json` (default: weighted)
- Stage probability weights from `hero.json`

The agent will produce:

### 1. Executive Summary

| Metric | Value |
|---|---|
| Commit | $X (deals at 80%+) |
| Best Case | $Y (commit + upside deals) |
| Weighted Pipeline | $Z (sum of ARR × probability) |
| Coverage Ratio | X.Xx (pipeline / quota) |
| Deals at Risk | N deals with slippage signals |

### 2. Pipeline by Stage

For each stage: deal count, total ARR, weighted ARR, avg days in stage,
stage conversion rate (from historical data if available).

### 3. Forecast by Rep

For each owner: commit, best case, weighted forecast, open pipeline,
quota attainment projection.

### 4. Forecast by Time Period

Week-by-week or month-by-month close date distribution within the period.

### 5. At-Risk Deals

Deals flagged for slippage risk (see `forecast-methodology` skill for
signals: close date pushed, no activity in 14+ days, low MEDDPICC score,
single-threaded with departing champion, etc.).

| Deal | ARR | Risk Signal | Recommended Action |
|---|---|---|---|

### 6. Deals Closing This Week

High-priority list for the current week with next actions.

**Write the forecast** as a file at `.hero/planning/forecasts/<period>.md`
so it can be referenced later and compared to actuals.

**If a CRM is configured**, optionally pull the latest stage data before
computing:
```
hero sync import --type deal --status open
```

**Surface the coverage ratio** prominently — below 3x is a risk signal that
should trigger pipeline generation activity.

---

## Flags

- `--period <Q# YYYY | month YYYY>` — forecast period (default: current quarter)
- `--rep <name>` — filter to one rep's pipeline
- `--stage <stage>` — filter to deals in a specific stage
- `--commit-only` — show only deals at 80%+ probability

---

## Session Title

Set the session title to: `forecast: <period>`

---

Period: $ARGUMENTS
