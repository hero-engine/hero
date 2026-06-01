---
title: "Sprint Planner — Agent-Driven Sprint Planning from Backlog"
slug: sprint-planner
type: feature
status: completed
tags: [agent, planning]
created: 2026-04-12
horizon: now
completed_at: 2026-05-18T19:25:38Z
---

## Goal

The `/sprint` agent command plans a sprint by selecting and sequencing specs from the backlog, producing a prioritized work plan that respects dependencies and capacity constraints.

## Design

When an agent invokes `/sprint`, Hero:

1. Reads all specs in backlog/draft status
2. Analyzes dependency relationships between specs (via `relations` frontmatter)
3. Considers priority, effort signals, and tags to rank candidates
4. Selects a set of specs that fit within the sprint scope
5. Sequences them respecting dependency order
6. Outputs a sprint plan with rationale for inclusion/exclusion

The agent uses project context (velocity history, in-flight work, team conventions) to make informed selection decisions. The output is a structured sprint document listing the selected specs in execution order with brief justification.

## Changes

- `commands/sprint.md` — `/sprint` agent command definition and prompt

## Acceptance Criteria

- `/sprint` produces a prioritized, sequenced sprint plan from backlog specs
- Dependency ordering is respected (blockers come before dependents)
- The plan includes rationale for why specs were included or excluded
- Sprint scope is bounded (not an unbounded wishlist)
- The command works with the existing spec and relations model
