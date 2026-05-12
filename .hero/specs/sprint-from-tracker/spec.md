---
title: Sprint from Tracker — Load a Live Sprint into Hero Specs
type: feature
status: completed
milestone: v0.2
tags: [sprint, jira, tracker, planning, import]
created: 2026-04-12
relations:
  - target: jira-deep-integration
    kind: depends-on
  - target: sprint-planner
    kind: related
horizon: now
---

## Goal

Engineers arrive on Monday morning and their sprint is already in Jira (or Linear). Today they have to either manually recreate that work in Hero, or work without Hero's spec structure. This spec closes that gap: `hero sprint load` pulls a live sprint from the tracker, turns each work item into a Hero spec stub, and produces a sprint plan document — all in one step.

This is the complement to the existing `/sprint` command, which plans a sprint from an *existing local backlog*. This command pulls from an *external sprint* that product/project management has already assembled.

## Design

### Command surface

```bash
# Pull a specific Jira sprint by ID or name
hero sprint load --tracker jira --sprint "Sprint 42"
hero sprint load --tracker jira --sprint 42

# Pull the active sprint for a board
hero sprint load --tracker jira --board "Engineering"

# Pull a Linear iteration
hero sprint load --tracker linear --iteration "2026-04-14"

# Preview without writing
hero sprint load --tracker jira --sprint 42 --dry-run
```

### What it does

1. **Authenticates** to the tracker (same env-var based auth as `hero import`)
2. **Fetches the sprint** — all work items assigned to that sprint (stories, bugs, tasks, sub-tasks)
3. **Deduplicates** — if a spec already exists with a matching `source` field, skips it (or updates frontmatter if `--update` is passed)
4. **Creates spec stubs** — one per work item, placed in the right Hero directory:
   - Story/feature/task → `planning/features/<slug>/spec.md`
   - Bug → `planning/bugs/<slug>/spec.md`
   - Epic → `planning/initiatives/<slug>/spec.md` with child references
5. **Writes a sprint plan note** to `.hero/knowledge/notes/sprint-<date>/spec.md` listing all imported specs, their priorities, assignees, and dependencies
6. **Runs `hero index`** so everything is immediately searchable

### Spec stub format

Imported specs include enough frontmatter to be useful immediately:

```markdown
---
title: "Add CSV export to user data API"
type: feature
status: planning
source: https://yourorg.atlassian.net/browse/ENG-142
tracker_id: ENG-142
priority: high
claimed_by: alice
tags: [api, export, sprint-42]
sprint: "Sprint 42"
created: 2026-04-14
---

## Goal

<!-- Imported from Jira: Add CSV export to user data API -->
Add the ability to export user data as a CSV file from the API.

## Description

<!-- Original Jira description -->
As a data analyst, I want to export user records as CSV so I can analyze them in Excel.

## Acceptance Criteria

<!-- Imported from Jira acceptance criteria / story points -->
- [ ] GET /api/users/export returns CSV
- [ ] Supports date range filtering
- [ ] Handles up to 100k rows

## Notes

*Spec stub imported from Jira ENG-142. Run `/design` to flesh this out into a full Hero spec before delivery.*
```

The stub is intentionally incomplete — it's a starting point. The team can enrich it with `/design` before running `/deliver`.

### Epic → initiative mapping

Jira epics become Hero initiative specs with child relations pointing to their stories:

```markdown
---
title: "Q2 Export Features"
type: initiative
status: planning
source: https://yourorg.atlassian.net/browse/ENG-100
tracker_id: ENG-100
sprint: "Sprint 42"
relations:
  - target: add-csv-export
    kind: child
  - target: add-pdf-export
    kind: child
---
```

### Sprint plan note

After import, a sprint plan note is written to `.hero/knowledge/notes/sprint-2026-04-14/spec.md`:

```markdown
---
title: Sprint 42 — 2026-04-14
type: note
tags: [sprint, sprint-42]
created: 2026-04-14
---

# Sprint 42

Imported from Jira on 2026-04-14.

| Spec | Type | Assignee | Priority | Tracker |
|---|---|---|---|---|
| add-csv-export | feature | alice | high | ENG-142 |
| fix-login-timeout | bug | bob | critical | ENG-143 |
| q2-export-features | initiative | — | — | ENG-100 |

## Sprint Goal
<!-- Imported from Jira sprint goal if present -->

## Ready for delivery
Specs that already have a `/design` spec: none — run `/design` on each before `/deliver`.
```

### Integration with `/sprint`

The existing `/sprint` agent command works from local backlog specs. After `hero sprint load`, those imported stubs *are* local backlog specs — so `/sprint` can then be used to sequence them, check dependencies, and confirm the sprint scope. The two commands compose naturally:

```
hero sprint load --tracker jira --sprint 42
/sprint initiative: sprint-2026-04-14
```

## Changes

- `internal/cli/sprint.go` — `hero sprint load` subcommand with tracker selection, dry-run
- `internal/tracker/jira.go` — sprint query (JQL: `sprint = X`), epic/story/bug field mapping
- `internal/tracker/linear.go` — iteration query, issue type mapping
- `internal/spec/stub.go` — spec stub writer with deduplication
- `commands/sprint.md` — update to document both `/sprint` (local backlog) and the CLI `hero sprint load` flow

## Acceptance Criteria

- `hero sprint load --tracker jira --sprint 42` imports all sprint work items as Hero spec stubs
- Jira epics become Hero initiative specs with child relations to their stories
- Existing specs with matching `source` fields are not duplicated (dedup by `tracker_id`)
- A sprint plan note is written to `.hero/knowledge/notes/sprint-<date>/spec.md`
- `--dry-run` previews what would be created without writing
- `--update` refreshes frontmatter on existing specs from current tracker state
- Authentication uses the same env vars as `hero import jira`
- Works with both Jira and Linear trackers

## Boundaries

- Does **not** replace `/design` — imported stubs are starting points, not full specs
- Does **not** create the sprint in the tracker — Hero reads, it does not write sprints back
- Does **not** modify specs that have been enriched beyond stub format (detects via body content length)
