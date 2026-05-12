---
title: "Root Cause Classification — Structured Diagnosis Taxonomy"
type: feature
status: completed
tags: [agent, diagnosis]
created: 2026-04-12
horizon: now
---

## Goal

Enhanced `/diagnose` command with a 7-category root cause taxonomy and structured frontmatter fields, enabling systematic tracking and analysis of failure patterns across the project.

## Design

The `/diagnose` command is extended to classify issues into one of seven root cause categories:

1. **code** — logic errors, off-by-one, wrong algorithm
2. **data** — bad input, schema mismatch, corrupt state
3. **env** — configuration, dependency version, platform difference
4. **user** — incorrect usage, missing prerequisite
5. **external** — third-party service failure, API change
6. **race** — concurrency, timing, ordering issue
7. **design** — architectural flaw, missing abstraction

Two new frontmatter fields are added to diagnosis specs:

- `root_cause_class` — one of the seven categories above
- `severity` — low / medium / high / critical

The classification skill provides structured reasoning guidance so the agent evaluates each category before selecting the best fit, avoiding premature anchoring.

## Changes

- `commands/diagnose.md` — updated `/diagnose` command with taxonomy integration and frontmatter fields
- `skills/root-cause-classification.md` — classification skill with category definitions, decision heuristics, and examples

## Acceptance Criteria

- `/diagnose` assigns a `root_cause_class` from the 7-category taxonomy
- `/diagnose` assigns a `severity` level (low/medium/high/critical)
- Both fields appear in the diagnosis spec frontmatter
- The classification skill provides clear definitions and examples for each category
- Agent considers all categories before selecting the best fit
