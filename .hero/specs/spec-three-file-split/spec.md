---
title: Three-File Spec Layout — Optional requirements/design/tasks Split
slug: spec-three-file-split
type: feature
status: completed
tags: [specs, structure, parser, backwards-compat]
created: 2026-04-22
relations:
  - target: competitor-parity
    kind: parent
  - target: ears-acceptance-criteria
    kind: related
horizon: now
---

## Goal

Allow a spec to be authored as three files — `requirements.md`, `design.md`,
`tasks.md` — instead of one monolithic `spec.md`, while keeping single-file
specs as the default and fully supported shape.

## Problem

Hero specs cram goal, problem, design, criteria, changes, and boundaries into
a single `spec.md`. For large initiatives this file grows to hundreds of lines
and becomes hard to review section-by-section. competitor three-file split makes
each phase reviewable and regeneratable independently, and lets trackers map
the right file to the right field (requirements → "story", design → "design
doc attachment", tasks → "subtasks").

## Design

### File layout

A spec may be a directory containing either:

- **Single-file**: `spec.md` (current shape, unchanged) — frontmatter + all sections
- **Three-file**: `requirements.md` + `design.md` + `tasks.md` — frontmatter only on `requirements.md`

```
.hero/planning/features/csv-export/
├── requirements.md      # frontmatter + Goal + Problem + Acceptance Criteria + Boundaries
├── design.md            # Design (architecture, tradeoffs, diagrams)
└── tasks.md             # Changes (file-level task list, can be checked off)
```

Lifecycle, status, claims, and tracker links all live in the `requirements.md`
frontmatter. The other two files have no frontmatter.

### Discovery

`internal/spec/spec.go` `Discover()` already walks `planning/` and `specs/`.
Update the rule:

- If a directory contains `spec.md` → load as single-file (current behavior)
- Else if it contains `requirements.md` → load as three-file, concatenate
  the three files in order into the in-memory `Spec.RawContent`

The in-memory `Spec` struct stays identical. Section parsing already keys off
H2 headings and is layout-agnostic. Tools that iterate sections (test gen,
drift, design-reviewer) work without changes.

### Conversion CLI

```
hero spec split <slug>       # convert single-file spec.md to three-file layout
hero spec join <slug>        # convert three-file layout back to single spec.md
hero spec split --all        # convert every planning spec to three-file
```

Both are pure file moves — no parsing surprises, deterministic output, idempotent.

### `/design` config

`hero.json` gains:

```json
{
  "specs": {
    "layout": "single"      // "single" (default) | "three-file"
  }
}
```

`/design` reads `specs.layout` and writes accordingly. Existing specs keep
their original layout — no implicit migration.

### Tracker mapping

`internal/tracker/jira.go` (and Linear, GitHub) currently push `s.RawContent`
to the issue body. With three-file layout, push:

- Issue body ← `requirements.md` body
- Comment 1 ← `design.md` ("Design doc attached below")
- Comment 2 ← `tasks.md` (or as a checklist on tracker types that support it)

Single-file specs continue pushing the whole body.

### Wiki sync

`internal/wiki/` already paginates large specs. Three-file specs sync as three
sibling pages with cross-links: `csv-export-requirements`, `csv-export-design`,
`csv-export-tasks`.

## Changes

- `internal/spec/spec.go` — `Discover()` recognizes three-file layout, concatenates into `RawContent`
- `internal/spec/spec_test.go` — fixtures for both layouts; round-trip tests
- `internal/cli/spec.go` — `hero spec split` and `hero spec join` subcommands
- `internal/config/config.go` — `Specs.Layout` field
- `internal/tracker/jira.go` (+linear.go, +github.go) — three-file push branching
- `internal/wiki/sync.go` — three-page sibling sync for three-file specs
- `commands/design.md` — read `specs.layout`, write accordingly
- `commands/split.md` — clarify it splits scope (multi-spec) vs `hero spec split` (file layout)

## Acceptance Criteria

- WHEN a spec directory contains `spec.md` THE SYSTEM SHALL load it as a single-file spec exactly as today
- WHEN a spec directory contains `requirements.md` + `design.md` + `tasks.md` THE SYSTEM SHALL load it as a three-file spec with the same in-memory `Spec` structure
- WHEN `hero spec split <slug>` runs against a single-file spec THE SYSTEM SHALL produce the three-file layout and remove the original `spec.md`
- WHEN `hero spec join <slug>` runs against a three-file spec THE SYSTEM SHALL produce a single `spec.md` and remove the three split files
- WHEN `/design` runs with `specs.layout: "three-file"` THE SYSTEM SHALL emit the three-file layout
- WHEN a three-file spec is pushed to Jira THE SYSTEM SHALL post requirements as the issue body and design + tasks as comments
- THE SYSTEM SHALL leave `specs.layout` defaulted to `"single"` so existing workflows are unchanged

## Boundaries

- Does **not** force migration — single-file specs stay valid forever
- Does **not** support arbitrary file names or counts — only the documented three
- Does **not** allow frontmatter in `design.md` or `tasks.md` (single source of truth)
- Does **not** change H2 section names or parsing semantics
