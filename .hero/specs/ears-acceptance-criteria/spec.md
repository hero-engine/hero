---
title: EARS Acceptance Criteria — Structured Grammar for Spec Validation
type: feature
status: completed
tags: [specs, ears, testing, parser, validation]
created: 2026-04-22
relations:
  - target: competitor-parity
    kind: parent
  - target: playwright-test-generation
    kind: related
  - target: spec-drift-detection
    kind: enables
horizon: now
---

## Goal

Let spec authors optionally write acceptance criteria in EARS notation
(Easy Approach to Requirements Syntax) so the criteria become machine-checkable
and feed `hero test generate` and `hero drift` more reliably than freeform prose.

## Problem

Today acceptance criteria are freeform bullet lists. The Playwright autonomous
generator falls back to `test.skip()` whenever its keyword heuristic can't map a
criterion to an assertion. Criteria like *"the system handles errors gracefully"*
generate nothing useful. Authors have no shared grammar, so two engineers writing
criteria for the same feature can produce wildly different shapes.

EARS gives us a small, learnable grammar that maps cleanly to assertions.

## Design

### EARS templates Hero will recognize

| Pattern | Example |
|---|---|
| **Ubiquitous** — `THE SYSTEM SHALL <behavior>` | `THE SYSTEM SHALL log every failed login attempt` |
| **Event-driven** — `WHEN <trigger> THE SYSTEM SHALL <behavior>` | `WHEN a user submits invalid form data THE SYSTEM SHALL display field-level validation errors` |
| **State-driven** — `WHILE <state> THE SYSTEM SHALL <behavior>` | `WHILE a sync is in flight THE SYSTEM SHALL block concurrent sync attempts` |
| **Optional** — `WHERE <feature> IS ENABLED THE SYSTEM SHALL <behavior>` | `WHERE auto_capture IS ENABLED THE SYSTEM SHALL persist learnings after /deliver` |
| **Unwanted** — `IF <trigger> THEN THE SYSTEM SHALL <behavior>` | `IF the tracker token is missing THEN THE SYSTEM SHALL print a setup hint and exit non-zero` |

Case-insensitive matching on the keywords. Trailing period optional.

### Parser changes

`internal/spec/spec.go` gains a `Criterion` struct:

```go
type Criterion struct {
    Raw      string         // original bullet text
    Pattern  CriterionKind  // ubiquitous | event | state | optional | unwanted | freeform
    Trigger  string         // WHEN/WHILE/IF/WHERE clause, empty for ubiquitous
    Behavior string         // SHALL clause
}

type CriterionKind int
const (
    CriterionFreeform CriterionKind = iota
    CriterionUbiquitous
    CriterionEvent
    CriterionState
    CriterionOptional
    CriterionUnwanted
)
```

`Spec.AcceptanceCriteria() []Criterion` parses the existing
`s.Sections["acceptance criteria"]` block, classifies each bullet, and returns
the list. Bullets that don't match any pattern fall through as `CriterionFreeform`
— full backwards compatibility.

### `/design` template update

The `/design` command emits a hint block at the top of the criteria section:

```markdown
## Acceptance Criteria

<!-- Prefer EARS patterns where they fit:
     WHEN <event> THE SYSTEM SHALL <behavior>
     WHILE <state> THE SYSTEM SHALL <behavior>
     IF <trigger> THEN THE SYSTEM SHALL <behavior>
     WHERE <feature> IS ENABLED THE SYSTEM SHALL <behavior>
     THE SYSTEM SHALL <behavior>
     Freeform bullets are allowed for criteria that don't fit. -->

- ...
```

Hint is HTML-comment so it doesn't render in the wiki sync output.

### `hero test generate` upgrade

When a criterion is structured, the autonomous adapter has explicit fields to
work with:

- `Trigger` clause containing "submit" / "click" / "navigate" → `page.click` /
  `page.goto` / `page.fill` setup
- `Behavior` clause containing "display" / "show" / "render" + a quoted string →
  `expect(page.locator(...)).toContainText(...)`
- `Behavior` clause containing "redirect" / "navigate to" + URL → `toHaveURL`
- `Behavior` clause containing "exit non-zero" / "fail" → CLI-style assertion
  (not browser)

The mapping table lives in `internal/testing/playwright.go`. Freeform criteria
keep the existing keyword-fallback path.

### `design-reviewer` agent update

`agents/design-reviewer.md` gets a new check: count EARS-shaped vs freeform
criteria, and flag specs where >50% are freeform with a suggestion to tighten
them. Advisory only — never blocks.

### CLI helper

```
hero spec lint <slug>     # report criteria classification + freeform ratio
hero spec lint --all      # workspace-wide report
```

Output:

```
csv-export (5 criteria)
  ✓ 4 EARS  (3 event, 1 state)
  ⚠ 1 freeform — "the system should be fast enough"
```

## Changes

- `internal/spec/spec.go` — `Criterion` struct, `CriterionKind`, parser
- `internal/spec/spec_test.go` — parser unit tests covering all five EARS patterns + freeform
- `internal/testing/playwright.go` — structured-criterion path in autonomous generator
- `internal/cli/spec.go` — new `hero spec lint` subcommand
- `commands/design.md` — emit EARS hint block in generated criteria section
- `agents/design-reviewer.md` — add EARS-coverage check to review heuristics
- `skills/spec-format.md` — document EARS as the preferred shape

## Acceptance Criteria

- WHEN a spec author writes `WHEN <x> THE SYSTEM SHALL <y>` THE SYSTEM SHALL parse it as `CriterionEvent` with the trigger and behavior split correctly
- WHEN a spec author writes a freeform bullet THE SYSTEM SHALL parse it as `CriterionFreeform` without erroring
- WHEN `hero test generate` runs against a spec with EARS criteria THE SYSTEM SHALL emit assertions derived from the parsed `Trigger` and `Behavior` fields rather than the raw bullet text
- WHEN `/design` produces a new spec THE SYSTEM SHALL include the EARS hint block in the criteria section
- WHEN `hero spec lint <slug>` runs THE SYSTEM SHALL print a per-criterion classification and the freeform ratio
- WHEN the `design-reviewer` agent reviews a spec where more than half of the criteria are freeform THE SYSTEM SHALL surface a suggestion to tighten them
- THE SYSTEM SHALL keep all existing specs (which use freeform criteria) working unchanged

## Boundaries

- Does **not** make EARS mandatory — freeform stays valid forever
- Does **not** invoke an LLM to convert freeform → EARS
- Does **not** change the `## Acceptance Criteria` section name or location
- Does **not** validate trigger/behavior semantics — only the structural shape
