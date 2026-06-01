---
title: Spec Types and Lifecycle — Expanded Type System and Status States
slug: spec-types-lifecycle
type: feature
status: completed
tags: [spec, types, lifecycle, index]
created: 2026-04-12
relations:
  - target: hero-v2-system-design
    kind: parent
horizon: now
completed_at: 2026-05-18T19:25:38Z
---

## Goal

Expand the Hero spec type system from 2 types (feature, bug) to 9 types and extend the lifecycle state set to cover the full spec workflow, including convention and decision lifecycles.

## What Was Built

**Spec types:** `feature`, `bug`, `convention`, `decision`, `initiative`, `rule`, `external`, `context`, `note`

**Lifecycle states:**
- Work specs: `planning → in-review → delivering → completed`
- Conventions: `draft → active → superseded`
- Decisions: `proposed → accepted → superseded`
- Shared: `superseded` (terminal for conventions and decisions)

**Folder structure:**
- `.hero/specs/` — completed work specs
- `.hero/planning/features/`, `.hero/planning/bugs/`, `.hero/planning/initiatives/` — active work
- `.hero/knowledge/conventions/` — convention specs
- `.hero/knowledge/decisions/` — decision specs
- `.hero/knowledge/notes/` — knowledge notes

All types and statuses handled in `internal/spec/spec.go` with graceful degradation for unknown values.

## Changes

- `internal/spec/spec.go` — full type and status enum, parser handles all types
- `internal/index/index.go` — index extended for type, status, tags, scope, relations, claims
- `internal/spec/discover.go` — walks all spec directories including knowledge subdirectories
