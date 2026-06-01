---
title: v2 Commands — /compose, /convention, /decide, /retro, /check
slug: v2-commands
type: feature
status: completed
tags: [commands, workflow, retro, compose, check]
created: 2026-04-12
relations:
  - target: hero-v2-system-design
    kind: parent
horizon: now
completed_at: 2026-05-18T19:25:38Z
---

## Goal

Add 5 new workflow commands to cover initiative decomposition, convention capture, decision recording, post-delivery retrospectives, and workspace health.

## What Was Built

**`/compose`** — breaks a large initiative into ordered spec stubs. `product-ideator` + `feature-delivery-lead` collaborate to decompose and sequence the work. Produces an initiative spec with child spec references.

**`/convention`** — convention-author agent analyzes codebase, produces a convention spec in `.hero/knowledge/conventions/`. Detects existing instances, documents the pattern with examples and anti-patterns.

**`/decide`** — records an architectural decision in ADR format via `architecture-reviewer`. Saved to `.hero/knowledge/decisions/`.

**`/retro`** — post-delivery retrospective comparing spec to actual git diff. Identifies deviations, what was harder than expected, and convention/decision updates needed.

**`/check`** — workspace health report: stale specs, spec drift, uncovered files, convention violations, unclaimed work. Runs on demand; `hero check` CLI runs the same checks.

**All existing commands unchanged** — `/discover`, `/design`, `/diagnose`, `/deliver`, `/review`, `/release`, `/docs` all continue to work as before.

## Changes

- `commands/compose.md`
- `commands/convention.md`
- `commands/decide.md`
- `commands/retro.md`
- `commands/check.md`
