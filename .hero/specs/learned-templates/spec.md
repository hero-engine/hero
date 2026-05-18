---
title: Learned Spec Templates — Templates from Delivery Patterns
slug: learned-templates
type: feature
status: completed
tags: [specs, templates, patterns, knowledge, dx]
created: 2026-04-22
priority: P2
relations:
  - target: hero-killer-features
    kind: parent
horizon: now
---

## Goal

After enough specs have been delivered, `hero new` scaffolds specs that match
the team's actual writing patterns rather than a static template. A new
`hero templates` command shows discovered patterns, and `hero templates refresh`
re-analyzes the completed spec corpus on demand. The result: new specs arrive
pre-shaped with the sections, frontmatter fields, and acceptance-criteria
density that this particular team actually uses.

## Problem

Today `hero new <slug> --type feature` emits the same five-section skeleton
every time: Goal, Background, Design, Changes, Acceptance Criteria. But real
teams develop their own conventions organically. One team always adds a
Performance Considerations section to feature specs. Another writes 6-8 EARS
acceptance criteria per spec. Bug specs in this repo consistently include both
a Root Cause section and a Regression Test section. None of these patterns are
captured by the static templates in `internal/cli/new.go`.

The static templates are good enough for first contact, but after a team has
delivered 10-20 specs, the scaffolding should learn from history. Without this,
engineers either add the same boilerplate sections by hand every time or forget
them and get flagged in review. The knowledge is in the spec corpus already --
Hero just doesn't use it.

## Design

### Pattern extraction (structural, no LLM)

The analyzer reads all completed specs (status: `completed`) from
`.hero/specs/` and groups them by type. For each type with at least 5 completed
specs, it extracts:

| Signal | How extracted | Example output |
|---|---|---|
| **Section structure** | Parse `## ` headings from each spec, count frequency across the corpus | "Performance Considerations appears in 80% of feature specs" |
| **Section ordering** | Record the position of each heading, compute median order | Goal -> Problem -> Design -> Performance -> Changes -> AC -> Boundaries |
| **Frontmatter fields** | Collect all YAML keys used across specs of this type, count frequency | `priority` used in 90%, `depends-on` in 40% |
| **Acceptance criteria count** | Count bullet items under `## Acceptance Criteria`, compute mean/median/range | "Feature specs average 6.2 criteria (range 4-9)" |
| **EARS pattern ratio** | Classify each criterion via existing EARS parser, compute ratio | "72% EARS, 28% freeform" |
| **Common tags** | Aggregate tags across specs of this type, rank by frequency | Top 5 tags for feature specs |

All extraction uses the existing `internal/spec/` parser for frontmatter and
sections, plus simple heading/bullet counting. No LLM calls.

### Generated pattern files

Discovered patterns are written to `.hero/knowledge/templates/` as one
markdown file per spec type:

```
.hero/knowledge/templates/feature.learned.md
.hero/knowledge/templates/bug.learned.md
```

Each file contains:

```markdown
---
type: learned-template
spec_type: feature
corpus_size: 14
generated: 2026-04-22T18:30:00Z
---
# Learned Feature Template

## Discovered sections (by frequency)

1. Goal (100%)
2. Problem (100%)
3. Design (93%)
4. Changes (100%)
5. Acceptance Criteria (100%)
6. Boundaries (86%)
7. Performance Considerations (71%)

## Acceptance criteria profile

- Mean count: 6.2
- Median count: 6
- Range: 4-9
- EARS ratio: 72%

## Common frontmatter fields

- priority (90%)
- depends-on (40%)
- parent (35%)

## Scaffold body

# {{.Title}}

## Goal

## Problem

## Design

## Changes

## Acceptance Criteria

## Boundaries

## Performance Considerations
```

The `## Scaffold body` section is the actual template text that `hero new`
uses. Sections appearing in >= 60% of completed specs are included. Ordering
follows the median position across the corpus.

### Integration with `hero new`

When `hero new <slug> --type <type>` runs:

1. Check for `.hero/knowledge/templates/<type>.learned.md`
2. If found, extract the `## Scaffold body` section and use it as the template
   body (same placeholder mechanism as existing custom templates)
3. If not found, fall back to the static template in `generateSpecTemplate()`
4. In `--interactive` mode, user-provided values always win -- learned defaults
   populate the scaffold but never override explicit choices

The existing `loadCustomTemplate()` mechanism in `new.go` already checks
`.hero/knowledge/templates/<type>.md` for custom templates. Learned templates
use the `.learned.md` suffix to coexist: a hand-written `feature.md` custom
template takes priority over a learned `feature.learned.md`.

Priority order: custom template > learned template > static template.

### `hero templates` command

```
hero templates                  # list discovered patterns per type
hero templates show <type>      # show full pattern detail for one type
hero templates refresh          # re-analyze corpus and regenerate patterns
hero templates refresh --force  # regenerate even if corpus is below threshold
```

Default output of `hero templates`:

```
Learned templates (from 47 completed specs):

  feature   14 specs  7 sections  avg 6.2 criteria  72% EARS
  bug        9 specs  6 sections  avg 4.8 criteria  65% EARS
  decision   5 specs  4 sections  (no criteria)

  convention  3 specs  — below threshold (5), using static template
  initiative  2 specs  — below threshold (5), using static template
```

### Storage

Pattern files live in `.hero/knowledge/templates/` alongside any hand-written
custom templates. The `.learned.md` suffix distinguishes them. They are
plain markdown, checked into git, and human-editable (though edits will be
overwritten on the next `hero templates refresh`).

## Changes

1. `internal/templates/templates.go` -- pattern extraction engine
   - `AnalyzeCorpus(specsDir string) (map[spec.Type]*TypePattern, error)` -- reads completed specs, groups by type, extracts all signals
   - `TypePattern` struct holding section frequencies, ordering, criteria stats, frontmatter field frequencies
   - `GenerateLearnedTemplate(pattern *TypePattern) string` -- renders the `.learned.md` file content
   - `LoadLearnedTemplate(knowledgeDir string, specType string) (string, bool)` -- reads the scaffold body from a learned template file
   - Section frequency threshold constant (60%)
   - Minimum corpus size constant (5)

2. `internal/templates/templates_test.go` -- table-driven tests
   - Corpus with known section structure produces expected pattern
   - Corpus below threshold returns no pattern
   - Section ordering matches median position
   - Criteria count stats are accurate
   - EARS ratio computation
   - Generated scaffold body includes sections above threshold, excludes those below

3. `internal/cli/templates.go` -- `hero templates` command and subcommands
   - `templates` (list) -- show summary table of discovered patterns
   - `templates show <type>` -- print full learned template file
   - `templates refresh [--force]` -- re-analyze corpus and write pattern files
   - Register under root command

4. `internal/cli/new.go` -- integrate learned templates into scaffolding
   - After `loadCustomTemplate()` check, add `loadLearnedTemplate()` fallback
   - Priority: custom > learned > static
   - In interactive mode, merge learned defaults with user inputs (user wins)

5. `internal/cli/root.go` -- register `templatesCmd`

## Acceptance Criteria

- WHEN `hero templates refresh` runs against a specs directory containing 5 or more completed specs of a given type THE SYSTEM SHALL generate a `.learned.md` file in `.hero/knowledge/templates/` for that type with accurate section frequencies, criteria stats, and a scaffold body
- WHEN `hero templates refresh` runs and a spec type has fewer than 5 completed specs THE SYSTEM SHALL skip that type and report it as below threshold
- WHEN `hero new <slug> --type <type>` runs and a learned template exists for that type and no custom template exists THE SYSTEM SHALL use the learned template's scaffold body instead of the static template
- WHEN `hero new <slug> --type <type> --interactive` runs THE SYSTEM SHALL never override values the user explicitly provides, even if learned defaults differ
- WHEN `hero templates` runs THE SYSTEM SHALL display a summary table listing each spec type, its corpus size, discovered section count, average criteria count, and EARS ratio
- IF a hand-written custom template exists at `.hero/knowledge/templates/<type>.md` alongside a learned template THEN THE SYSTEM SHALL prefer the custom template over the learned template
- THE SYSTEM SHALL extract all patterns using structural analysis of markdown headings, frontmatter fields, and bullet counts without making any LLM API calls

## Boundaries

- Does **not** require an LLM -- pattern extraction is structural analysis of markdown headings, frontmatter YAML, and bullet lists
- Does **not** override explicit user choices in `hero new --interactive` -- user-provided values always take priority over learned defaults
- Does **not** modify existing specs -- only reads completed specs for analysis, never writes back to them
- Does **not** generate patterns until a spec type has at least 5 completed specs in the corpus
- Does **not** replace hand-written custom templates -- custom templates at `.hero/knowledge/templates/<type>.md` always take priority over learned `.learned.md` files
- Does **not** run automatically -- patterns are generated on explicit `hero templates refresh` only
