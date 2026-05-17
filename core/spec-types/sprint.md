---
title: Sprint
type: sprint
domain: core
category: work
bucket: sprints
location: .hero/planning/sprints/{slug}/spec.md
lifecycle:
  states: [planning, committed, in-flight, completed]
  initial: planning
  terminal: [completed]
  transitions:
    - { from: planning, to: committed, gate: "scope chosen" }
    - { from: committed, to: in-flight, gate: "sprint starts" }
    - { from: in-flight, to: completed, gate: "sprint ends" }
owner:
  values: [pm, engineering, qa, devops, design, docs]
  default: engineering
  classification: org-state
sections:
  required: [Goal]
  optional: [Scope, Dates, Risks, Notes]
accepting_commands: [/sprint, /refine, /handoff]
default_agents:
  authoring: spec-writer
  review: pm-reviewer
  handoff: handoff-coordinator
relations:
  - { kind: parent, target_type: release, cardinality: zero-or-one }
  - { kind: contains, target_type: feature, cardinality: many }
  - { kind: contains, target_type: bug, cardinality: many }
---

# Sprint spec-type

A **sprint** is an iteration time-box — the checkpoint inside a release.
It groups features and bugs committed to a single iteration with a
defined start, end, and goal. The vocabulary preset renders the display
name ("Sprint" under agile-scrum, "Cycle" under shape-up, "Iteration"
under generic agile).

Methodology profiles decide whether sprint is required, optional, or
unused. Kanban-style workspaces typically don't author sprint artifacts;
scrum and shape-up workspaces always do.

## When to use

- A fixed iteration (1-4 weeks, often 2) with a committed scope.
- A Shape Up cycle (typically 6 weeks) plus its cooldown.
- A planning checkpoint that aggregates committed features and bugs.

## When NOT to use

- A continuous-flow workspace (kanban) — no sprint artifact needed.
- A long release-scale time-box — that's a **release**.

## Lifecycle

States: `planning → active → completed → retrospected` (terminal); plus
`cancelled` (terminal) reachable from `planning` or `active`.

- `planning → active` — gate: sprint commitment locked; start date hit.
- `active → completed` — gate: end date hit; scope reconciled (shipped vs
  carried-over).
- `completed → retrospected` — gate: retro held; learnings captured.
- `planning → cancelled` — gate: rare; sprint dropped before start.
- `active → cancelled` — gate: rare; mid-sprint cancellation logged.

Methodology profiles overlay alternate state machines (e.g. shape-up's
`betting → building → cooldown → shipped`); this is the methodology-
neutral default.

## Kind

No `kind` values v1. Sprint shape varies by methodology — the methodology
profile defines duration, rituals (planning / standup / review / retro),
and estimation field rather than a sub-type enum.

## Tasks Schema

- Section heading: `Tasks`
- Required: false
- History: bitemporal

Item shape:

- `id` — string, required, format `T-<int>`
- `text` — string, required
- `status` — enum [todo, doing, done], default `todo`
- `kind` — optional string (e.g. `planning`, `ceremony`, `retro-action`)
- `assignee` — optional string
- `discovered_against` — optional ref to another spec
- `started` — optional date
- `done` — optional date

Sprint-level tasks capture ceremonies and retro action items, not
delivery work (delivery lives on the features and bugs committed into
the sprint).

## Owner

- Values: [pm, engineering, qa, devops, design, docs]
- Default: `pm`
- Classification: org-state
- Lifecycle triggers: none v1. Sprint ownership is typically the
  scrum-master / cycle-lead role; expressed via the `pm` owner in v1.
  Bitemporally tracked.

## Sections

- Required: `Goal`, `Committed`
- Optional: `Capacity`, `Risks`, `Tasks`, `Retro`, `Notes`

## Accepting Commands

- `/sprint` — plan a sprint by selecting and sequencing specs
- `/design` — draft a sprint
- `/retro` — run a post-sprint retrospective

## Default Agents

- authoring: `sprint-planner`
- review: `pm-reviewer`
- retro: `retro-facilitator`

## Relations

- `parent → release` (cardinality: zero-or-one; sprint inside a release)
- `contains → feature` (cardinality: many)
- `contains → bug` (cardinality: many)
