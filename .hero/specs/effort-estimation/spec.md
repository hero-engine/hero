---
title: "Effort Estimation — Complexity-Based Cost Prediction"
type: feature
status: completed
tags: [cli, analytics, estimation]
created: 2026-04-12
horizon: now
---

## Goal

The `hero cost` command estimates implementation effort for a spec based on complexity signals, calibrating predictions against historical data from completed specs in the project.

## Design

Effort estimation analyzes a spec and produces an effort score using multiple complexity signals:

- **File count** — number of files listed in the Changes section
- **Section count** — number of major sections in the spec (design, constraints, phases, etc.)
- **Dependency count** — number of relations/dependencies in frontmatter
- **Word count** — total word count as a proxy for scope breadth
- **Spec type** — feature vs bugfix vs refactor (different baseline effort)

### Calibration

The estimator calibrates against completed specs in the project. For each completed spec, it computes the same complexity signals and correlates them with actual duration (time from in-progress to completed). This calibration produces project-specific weights, so estimates improve as more specs are completed.

### Usage

```
hero cost <slug>
hero cost --all          # estimate all draft/backlog specs
```

Output includes the effort estimate (e.g., "~3 days"), confidence level (low/medium/high based on calibration data), and the top contributing complexity factors.

## Changes

- `internal/cli/cost.go` — `hero cost` command implementation, signal extraction, calibration engine
- `internal/cli/cost_test.go` — tests for signal extraction, calibration, estimation accuracy

## Acceptance Criteria

- `hero cost <slug>` produces an effort estimate for a given spec
- Estimate is based on measurable complexity signals (file count, sections, dependencies, word count, type)
- Calibration uses completed specs to improve accuracy over time
- `hero cost --all` estimates all draft/backlog specs in batch
- Confidence level reflects the amount of calibration data available
- Tests cover signal extraction, calibration logic, and edge cases
