---
title: Epic
type: epic
domain: core
category: work
bucket: epics
location: .hero/planning/epics/{slug}/spec.md
lifecycle:
  states: [planning, refined, ready, delivering, in-review, completed]
  initial: planning
  terminal: [completed]
  transitions:
    - { from: planning, to: refined, gate: "scope and child features sketched" }
    - { from: refined, to: ready, gate: "child features drafted" }
    - { from: ready, to: delivering, gate: "first child feature picked up", owner_flip: { to: engineering } }
    - { from: delivering, to: in-review, gate: "child features delivering" }
    - { from: in-review, to: completed, gate: "all children completed" }
kind:
  values: [theme, delivery, bet, milestone]
  default: theme
  required: false
owner:
  values: [pm, engineering, qa, devops, design, docs]
  default: pm
  classification: org-state
sections:
  required: [Goal]
  optional: [Scope, Features, Risks, Notes]
accepting_commands: [/design, /refine, /compose, /handoff]
default_agents:
  authoring: spec-writer
  review: pm-reviewer
  handoff: handoff-coordinator
relations:
  - { kind: parent, target_type: initiative, cardinality: zero-or-one }
  - { kind: parent, target_type: prd, cardinality: zero-or-one }
  - { kind: child, target_type: feature, cardinality: many }
  - { kind: child, target_type: bug, cardinality: many }
frontmatter:
  required:
    - { name: title, type: string, required: true, classification: content, description: "One-line human title for the epic." }
    - { name: type, type: enum, required: true, values: [epic], default: epic, classification: content, description: "Spec type discriminator; always 'epic'." }
    - { name: status, type: enum, required: true, values: [planning, refined, ready, delivering, in-review, completed], default: planning, classification: org-state, description: "Lifecycle position." }
  optional:
    - { name: created, type: date, format: "YYYY-MM-DD", classification: content, description: "Authoring date." }
    - { name: priority, type: enum, values: [P0, P1, P2, P3], classification: org-state, description: "Hero-level priority." }
    - { name: horizon, type: enum, values: [now, next, someday, parking], default: now, classification: content, description: "Temporal segmentation." }
    - { name: tags, type: "list[string]", classification: content, description: "Free-form labels." }
    - { name: claimed_by, type: string, classification: org-state, description: "Who is actively working this spec." }
    - { name: tracker_id, type: string, classification: org-state, description: "External tracker issue ID." }
    - { name: kind, type: enum, values: [theme, delivery, bet, milestone], default: theme, classification: content, description: "Epic sub-category." }
    - { name: owner, type: enum, values: [pm, engineering, qa, devops, design, docs], default: pm, classification: org-state, description: "Current owning role." }
    - { name: relations, type: "list[relation]", classification: content, description: "Parent/child edges to other specs." }
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
