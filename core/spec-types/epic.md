---
title: Epic
type: epic
domain: core
category: work
bucket: epics
location: .hero/planning/epics/{slug}/spec.md
---

# Epic spec-type

An **epic** is a mid-tier grouping — a coherent bucket of features that go
together to achieve a large capability. It sits between `initiative`
(strategic bet) and `feature` (deliverable unit). Shared across PM and
engineering: PM-led for strategic epics, engineering-led for delivery
epics. May exist standalone with no parent initiative (Jira-board style).

## When to use

- A coherent large feature that needs multiple features to deliver.
- A grouping that maps to a tracker epic.
- A Shape Up cycle bet (`kind: bet`) — one cycle, fixed appetite.
- A date-anchored delivery milestone (`kind: milestone`).

## When NOT to use

- Single-feature scope — write the feature directly.
- A coarse-grained future bet without committed features — that's an
  **initiative**.

## Lifecycle

States: `proposed → committed → in-flight → done` (terminal).

- `proposed → committed` — gate: first child feature moves to ready.
- `committed → in-flight` — gate: first child feature moves to delivering.
- `in-flight → done` — gate: all child features completed, or explicit
  close-with-residual decision.

## Kind

Values: `[theme, delivery, bet, milestone]`

- `theme` — thematic grouping for portfolio views; long-running, may span
  multiple cycles
- `delivery` — mid-flight set of features being shipped together; the
  default engineering-side shape
- `bet` — Shape Up cycle bet; one cycle, fixed appetite
- `milestone` — date-anchored cluster (e.g. "EU launch Q3")

Default: `theme`. Required: false.

## Tasks Schema

- Section heading: `Tasks`
- Required: false
- History: bitemporal

Item shape:

- `id` — string, required, format `T-<int>`
- `text` — string, required
- `status` — enum [todo, doing, done], default `todo`
- `kind` — optional string (e.g. `coordination`, `infra-prep`,
  `release-readiness`)
- `assignee` — optional string
- `discovered_against` — optional ref to another spec
- `started` — optional date
- `done` — optional date

Epic-level tasks are cross-spec coordination shaped, not feature-level
delivery work.

## Owner

- Values: [pm, engineering, qa, devops, design, docs]
- Default: `pm`
- Classification: org-state
- Lifecycle triggers: none v1. Epics typically stay owned by PM while
  their child features flip to engineering individually. Delivery-shaped
  epics may flip wholesale at `proposed → committed`; the flip is
  authored, not declared.

## Sections

- Required: `Why`, `Rollup Acceptance Criteria`, `Child specs`
- Optional: `Sequencing`, `Dependencies`, `Tasks`, `Risks`, `Notes`

## Accepting Commands

- `/design` — draft or refine an epic
- `/compose` — decompose into child features
- `/refine` — sharpen rollup AC or sequencing

## Default Agents

- authoring: `epic-framer`
- review: `pm-reviewer`

## Relations

- `parent → initiative` (cardinality: zero-or-one; epics may exist
  standalone)
- `parent → prd` (cardinality: zero-or-one)
- `children → feature` (cardinality: many)
