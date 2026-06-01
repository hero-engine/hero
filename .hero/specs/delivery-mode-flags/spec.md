---
title: Delivery Mode Flags — Autopilot, Supervised, and Dry-Run
slug: delivery-mode-flags
type: feature
status: completed
tags: [deliver, modes, ux, claims, dx]
created: 2026-04-22
relations:
  - target: competitor-parity
    kind: parent
  - target: spec-drift-detection
    kind: related
horizon: now
completed_at: 2026-05-18T19:25:38Z
---

## Goal

Make `/deliver`'s execution intent explicit by supporting three named modes —
`--autopilot`, `--supervised`, and `--dry-run` — instead of one ambiguous loop.
Match competitor clarity around "run the list vs ask each step" without changing
how the underlying delivery lead and engineer agents work.

## Problem

Today `/deliver <slug>` runs one mode: the delivery lead coordinates specialists
and writes code. There's no way to say "do everything but don't commit," "run
straight through without pausing for confirmations," or "ask me before each
file change." Users either trust the agent fully or babysit every step. competitor
distinguishes autopilot from supervised explicitly, and users find that
clarifying.

## Design

### Three modes

| Flag | Behavior |
|---|---|
| `--supervised` (default) | Current behavior. Delivery lead pauses at handoffs, surfaces decisions, asks before destructive actions. |
| `--autopilot` | Runs to completion without intermediate confirmations. Stops only on test failure, drift warning ≥ severity 1, or boundary violation. |
| `--dry-run` | Runs analysis + planning. Produces a delivery plan (file list, agent assignments, estimated changes) but writes no code. |

### `/deliver` invocation

```
/deliver csv-export                      # supervised (default)
/deliver csv-export --autopilot          # straight-through
/deliver csv-export --dry-run            # plan only, no writes
/deliver csv-export --autopilot --halt-on drift,test
```

`--halt-on` (autopilot only) takes a comma-separated list of conditions that
escalate to a pause: `drift`, `test`, `boundary`, `lint`. Default: all of them.

### Configuration

```json
{
  "delivery": {
    "default_mode": "supervised",
    "autopilot_halt_on": ["drift", "test", "boundary"],
    "dry_run_writes_plan": true
  }
}
```

`dry_run_writes_plan: true` (default) writes the planned diff summary to
`.hero/planning/features/<slug>/plan.md` so the user can review without an
agent transcript.

### Delivery lead contract

`agents/feature-delivery-lead.md` and `agents/platform-delivery-lead.md` get a
new opening section that reads the mode from the invocation:

- **Supervised** — keep current behavior verbatim.
- **Autopilot** — suppress confirmation requests; check `hero drift`, run tests
  (if a runner is configured), and halt only on conditions in `--halt-on`.
- **Dry-run** — emit a plan: ordered task list, target files, specialist
  assignments, estimated complexity. No file writes.

### Plan output (dry-run)

```markdown
# Delivery plan — csv-export
Generated: 2026-04-22T14:00:00Z
Mode: dry-run

## Tasks (sequenced)
1. [api-engineer] Add `/api/export.csv` route → internal/api/export.go (~40 lines)
2. [database-engineer] Add `users.csv_export_count` migration → migrations/0042_*.sql (~10 lines)
3. [engineer] Wire route handler → internal/api/handlers/export.go (~80 lines)
4. [test-architect] Add e2e test (autonomous mode → playwright) → e2e/csv-export.spec.ts

## Risks
- Touches the auth middleware (boundary check): no — handler uses existing middleware
- Drift baseline: clean — no in-flight conflicts

## Estimated complexity
~130 lines, 3 specialist handoffs, ~15 minutes wall time
```

### Claims integration

`hero claim <slug>` already records the active agent. Mode is recorded too:

```jsonl
{"ts":"2026-04-22T14:00Z","slug":"csv-export","action":"claim","agent":"opencode/claude","mode":"autopilot"}
```

`hero claims --stale` surfaces autopilot claims that have been running >
`tracking.autopilot_max_minutes` (default 30) as a soft warning.

### Velocity tracking

`hero velocity` adds a column for mode breakdown so teams can see whether
autopilot deliveries are statistically slower or faster than supervised ones.

## Changes

- `internal/cli/deliver.go` — accept mode flags (currently just a `/deliver` command)
- `internal/config/config.go` — `DeliveryConfig` struct with the three fields
- `internal/tracking/claim.go` — write `mode` into events.log
- `internal/tracking/velocity.go` — group by mode in the velocity report
- `commands/deliver.md` — document the three flags + `--halt-on`
- `agents/feature-delivery-lead.md` — branching behavior per mode
- `agents/platform-delivery-lead.md` — same
- `agents/engineer.md` — note that `--dry-run` means no writes

## Acceptance Criteria

- WHEN `/deliver <slug>` runs without a mode flag THE SYSTEM SHALL behave identically to today (supervised)
- WHEN `/deliver <slug> --autopilot` runs THE SYSTEM SHALL suppress confirmation prompts during the delivery loop
- WHEN `/deliver <slug> --autopilot` runs and a halt condition fires THE SYSTEM SHALL pause and surface the condition to the user
- WHEN `/deliver <slug> --dry-run` runs THE SYSTEM SHALL produce a plan and write `.hero/planning/features/<slug>/plan.md` without modifying any source files
- WHEN `/deliver <slug> --autopilot --halt-on drift,test` runs THE SYSTEM SHALL only halt on drift or test failures, ignoring lint and boundary conditions
- WHEN a claim is recorded THE SYSTEM SHALL include the mode in `events.log`
- WHEN `hero claims --stale` runs THE SYSTEM SHALL flag autopilot claims older than `tracking.autopilot_max_minutes`
- WHEN `hero velocity` runs THE SYSTEM SHALL group results by delivery mode

## Boundaries

- Does **not** change supervised mode behavior (current default preserved)
- Does **not** add a new agent — modes are flags interpreted by existing delivery leads
- Does **not** auto-merge or auto-push — autopilot still respects all git boundaries
- Does **not** support a "fully autonomous" mode that ignores all halts
- Does **not** track per-mode cost — leave that to a future enhancement
