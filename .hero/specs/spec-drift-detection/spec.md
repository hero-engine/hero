---
title: Spec Drift Detection — Flag When Code Diverges From Its Spec
slug: spec-drift-detection
type: feature
status: completed
tags: [specs, drift, validation, retro, mcp]
created: 2026-04-22
relations:
  - target: competitor-parity
    kind: parent
  - target: ears-acceptance-criteria
    kind: depends-on
  - target: hero-pulse
    kind: related
horizon: now
---

## Goal

Detect — while a spec is in flight, not just retroactively — when the
implementation has diverged from what the spec promised. Surface drift through
a new `hero drift` command, an MCP tool, and a `pulse`-style report so delivery
leads can ask "is the code still doing what we said it would?"

## Problem

Hero has retroactive spec-vs-code comparison via `/retro` and `hero replay`,
but those run *after* completion. Mid-delivery, a spec can quietly become a
lie — files get renamed, criteria get silently dropped, new boundaries get
crossed. Today the only way to notice is to re-read the spec and the diff
manually. competitor re-syncs spec↔code in both directions; Hero can do better than
retroactive comparison without going as far as auto-rewriting either side.

## Design

### What "drift" means

Five concrete signals, all derivable from existing Hero data:

| Signal | Source | Example |
|---|---|---|
| **Missing files** | Spec `## Changes` lists `internal/foo.go`, file doesn't exist | Spec promised a new package that was never created |
| **Renamed/moved files** | Spec lists `internal/foo.go`, git log shows it was renamed to `internal/bar.go` | Refactor happened mid-delivery, spec wasn't updated |
| **Untouched criteria** | EARS `Behavior` clause references a symbol/file that has no commits since spec creation | Acceptance criterion was forgotten |
| **Crossed boundaries** | Spec `## Boundaries` says "does not touch X", git log shows commits to X on this branch | Scope crept |
| **Stale acceptance** | Spec criterion mentions `flag --foo`, no occurrence of `--foo` in code | Promised flag never landed |

### `hero drift` command

```
hero drift <slug>              # report drift for one spec
hero drift --in-flight         # all delivering specs in this workspace
hero drift --initiative <id>   # all child specs of an initiative
hero drift --since <ref>       # only count drift introduced since git ref
hero drift --format json       # machine-readable output
```

Default human output:

```
csv-export (status: delivering, claimed by opencode/claude)
  ✓ 4/5 acceptance criteria have related code changes
  ⚠ 1 criterion looks unaddressed:
      "WHEN export size exceeds 10MB THE SYSTEM SHALL stream rather than buffer"
      → no occurrences of "stream" or "buffer" in changed files
  ⚠ Boundary possibly crossed:
      Spec says "does not modify the auth middleware"
      → 3 commits to internal/auth/middleware.go on this branch
  ✓ All listed files in ## Changes exist
```

Exit code: 0 = no drift, 1 = warnings, 2 = boundary violation.

### How signals are computed

All five signals are local + heuristic — no LLM calls.

- **Files**: parse `## Changes` block (existing Markdown list parser), `os.Stat` each path, `git log --follow --diff-filter=R` to detect renames.
- **Boundaries**: parse `## Boundaries` block, extract paths/symbols mentioned with negative phrasing ("does not", "must not", "never"), grep changed files since spec creation date.
- **Criteria**: for each EARS `Behavior` clause, extract identifier-shaped tokens (CLI flags, function names, file paths) and grep the diff range. Freeform criteria contribute a softer warning.
- **Stale acceptance**: same as above, but evaluated against the entire branch since `git merge-base main HEAD`.

The criterion ↔ code mapping reuses the keyword extraction already present in
`internal/testing/playwright.go`.

### MCP tool

```json
{
  "name": "hero_drift",
  "description": "Report drift between spec and code for one or more specs",
  "inputSchema": {
    "type": "object",
    "properties": {
      "slug": { "type": "string" },
      "in_flight": { "type": "boolean" },
      "initiative": { "type": "string" },
      "since": { "type": "string", "description": "git ref" }
    }
  }
}
```

Returns the same structured payload as `--format json`. Lets delivery leads
inject drift status into handoffs the same way they already inject `hero context`.

### Integration with existing surfaces

- **`hero pulse`** — gain a "Drift" section listing in-flight specs with
  warnings, similar to the existing "At Risk" section.
- **`/check`** — run drift on all delivering specs as part of workspace health.
- **`/deliver`** — at the end of each agent loop, the delivery lead runs
  `hero drift <slug>` and surfaces any new warnings before declaring complete.
- **`functional-qa-engineer`** agent gets a new tool: read the drift report
  before validating behavior.

### Storage

Drift is computed on demand — nothing persisted. The SQLite index already has
file→spec mappings via the `## Changes` parser; that's the only required input.

## Changes

- `internal/drift/drift.go` — signal computation, report struct, JSON encoder
- `internal/drift/drift_test.go` — table-driven tests per signal
- `internal/cli/drift.go` — `hero drift` command + flags
- `internal/cli/root.go` — register `driftCmd`
- `internal/serve/mcp.go` — register `hero_drift` tool
- `internal/pulse/pulse.go` — add Drift section to pulse report
- `commands/check.md` — invoke drift as part of workspace health
- `commands/deliver.md` — invoke drift at end of delivery loop
- `agents/feature-delivery-lead.md` — surface drift warnings in handoff
- `agents/functional-qa-engineer.md` — include drift report in validation context

## Acceptance Criteria

- WHEN `hero drift <slug>` runs against a spec whose listed files all exist and whose criteria all have related diffs THE SYSTEM SHALL exit 0 with a green report
- WHEN a spec's `## Changes` lists a file that does not exist THE SYSTEM SHALL flag the missing file as a warning and exit non-zero
- WHEN a spec's `## Boundaries` says "does not touch X" and the branch has commits modifying X THE SYSTEM SHALL flag a boundary violation and exit code 2
- WHEN an EARS criterion mentions an identifier that has no occurrence in changed files THE SYSTEM SHALL flag the criterion as possibly unaddressed
- WHEN `hero drift --in-flight` runs THE SYSTEM SHALL evaluate every delivering spec in the workspace and aggregate the report
- WHEN `hero drift --initiative <id>` runs THE SYSTEM SHALL evaluate every child spec of the initiative
- WHEN the `hero_drift` MCP tool is called THE SYSTEM SHALL return the same payload as `--format json`
- WHEN `hero pulse` runs THE SYSTEM SHALL include a Drift section listing in-flight specs with warnings
- THE SYSTEM SHALL compute all signals locally without making any LLM API calls

## Boundaries

- Does **not** auto-rewrite the spec or the code to reconcile drift
- Does **not** require EARS criteria — degrades to keyword extraction on freeform
- Does **not** call an LLM
- Does **not** persist drift state — recomputed on demand
- Does **not** block commits, builds, or merges — advisory only
