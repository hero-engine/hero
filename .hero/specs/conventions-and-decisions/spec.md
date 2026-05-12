---
title: Conventions and Decisions — Knowledge Specs for How and Why
type: feature
status: completed
tags: [conventions, decisions, knowledge, context]
created: 2026-04-12
relations:
  - target: hero-v2-system-design
    kind: parent
horizon: now
---

## Goal

Introduce conventions and decisions as first-class spec types. Conventions document repeatable patterns ("how we do X"); decisions record architectural choices with rationale ("why we chose X"). Both are loaded by agents before implementation to ensure consistent code.

## What Was Built

**Convention specs** live in `.hero/knowledge/conventions/` with lifecycle `draft → active → superseded`. They declare a `scope` frontmatter field (file globs) controlling when they apply.

**Decision specs** live in `.hero/knowledge/decisions/` with lifecycle `proposed → accepted → superseded`. They record context, alternatives, chosen approach, and consequences.

**`/convention` command** — `convention-author` agent analyzes codebase for patterns and produces convention specs.

**`/decide` command** — `architecture-reviewer` records architectural decisions in ADR format.

**Convention scope matching** — `hero context --files` finds conventions whose scope globs match the files being worked on.

**`convention-writing` skill** — loaded by `convention-author` to produce well-structured conventions.

## Changes

- `commands/convention.md` — `/convention` command definition
- `commands/decide.md` — `/decide` command definition
- `agents/convention-author.md` — convention author agent
- `skills/convention-writing.md` — how to analyze and write effective conventions
- `internal/spec/spec.go` — `TypeConvention`, `TypeDecision`, `StatusDraft`, `StatusProposed`, `StatusAccepted` states
- `internal/index/index.go` — `convention_scopes` table, scope-glob matching
