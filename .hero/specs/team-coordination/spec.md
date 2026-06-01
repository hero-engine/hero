---
title: Team Coordination — Claiming, Conflict Detection, and Spec Review
slug: team-coordination
type: feature
status: completed
tags: [team, claiming, conflicts, review, coordination]
created: 2026-04-12
relations:
  - target: hero-v2-system-design
    kind: parent
horizon: now
completed_at: 2026-05-18T19:25:38Z
---

## Goal

Multiple developers using AI agents on the same codebase need lightweight coordination primitives: claiming specs to signal active work, detecting file overlap between concurrent specs, and reviewing designs before execution.

## What Was Built

**Spec claiming** — `hero claim <slug>` sets `claimed_by` and `status: delivering` in frontmatter. `hero status` shows all claims. Claiming is advisory (git-committed frontmatter, no lock server). Agents check claim status and warn when a spec is already claimed.

**Conflict detection** — `hero conflicts <spec-path>` queries the index for in-flight specs with overlapping `files_touched`. Delivery leads run this before starting work.

**`in-review` status** — new lifecycle state between `planning` and `delivering`. `/review spec <path>` reviews the design document using `architecture-reviewer`. `team.require_review: true` in hero.json prevents delivery on unreviewed specs.

**`hero check`** — workspace health report: stale specs (planning > N days), spec drift (files heavily modified post-completion), uncovered files (no spec history), unclaimed planning specs.

## Changes

- `internal/cli/claim.go` — `hero claim` / `hero unclaim` commands
- `internal/cli/conflicts.go` — `hero conflicts` command
- `internal/cli/check.go` — `hero check` health report
- `internal/index/index.go` — `claims` table, `spec_relations` table, stale/drift queries
- `commands/check.md` — `/check` command definition
- `commands/review.md` — updated with `spec` review target type
