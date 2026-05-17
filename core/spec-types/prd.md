---
title: Product Requirement Document
type: prd
domain: core
category: work
bucket: prds
location: .hero/planning/prds/{slug}/spec.md
lifecycle:
  states: [planning, refined, ready, delivering, in-review, completed]
  initial: planning
  terminal: [completed]
  transitions:
    - { from: planning, to: refined, gate: "problem and scope articulated" }
    - { from: refined, to: ready, gate: "decomposed into features/epics" }
    - { from: ready, to: delivering, gate: "child work begins", owner_flip: { to: engineering } }
    - { from: delivering, to: in-review, gate: "child features delivered" }
    - { from: in-review, to: completed, gate: "PRD accepted" }
kind:
  values: [pitch, ten-section, lightweight]
  default: ten-section
  required: false
owner:
  values: [pm, engineering, qa, devops, design, docs]
  default: pm
  classification: org-state
sections:
  required: [Problem, Goal]
  optional: [Background, Users, Scope, Out of Scope, Risks, Notes]
accepting_commands: [/design, /refine, /handoff]
default_agents:
  authoring: spec-writer
  review: pm-reviewer
  handoff: handoff-coordinator
relations:
  - { kind: parent, target_type: initiative, cardinality: zero-or-one }
  - { kind: child, target_type: epic, cardinality: many }
  - { kind: child, target_type: feature, cardinality: many }
---

# PRD spec-type

A **PRD** is the heaviest authoring artifact — the flushed-out version of
an initiative. It captures the *what* and *why* of a product change in
enough detail that an engineering team can pick it up, decompose it into
features, and deliver against it without back-and-forth on intent. PM-led.

## When to use

- An initiative has been promoted to active and needs detailed framing.
- The work spans multiple features or is large enough that a single
  feature alone won't capture the context.
- Stakeholders need to align on tradeoffs before engineering invests
  effort.

## When NOT to use

- A single-feature improvement — write the feature directly.
- An ambiguous customer signal — that's an **intake**; triage first.
- A strategy-level objective without scope — that's an **initiative**.

## Lifecycle

States: `draft → review → approved → delivered` (terminal).

- `draft → review` — gate: pm-reviewer pass.
- `review → approved` — gate: PM approval action.
- `review → draft` — gate: review findings require rework.
- `approved → delivered` — gate: all child features/epics completed.

## Kind

Values: `[pitch, ten-section, lightweight]`

- `pitch` — Shape Up pitch shape (Problem / Appetite / Solution / Rabbit
  Holes / No-Gos)
- `ten-section` — the canonical agile PRD shape (Problem, Goals & Metrics,
  Users & Personas, Solution, User Flows, AC, Out of Scope, Risks, Open
  Questions, Timeline)
- `lightweight` — single-section problem framing for small bets

Default: `ten-section`. Required: false.

The vocabulary preset may rename the display name (e.g. shape-up renders
`prd.pitch` as "Pitch Doc"). Methodology profiles may enforce a particular
kind under their preset.

## Tasks Schema

- Section heading: `Tasks`
- Required: false
- History: bitemporal

Item shape:

- `id` — string, required, format `T-<int>`
- `text` — string, required
- `status` — enum [todo, doing, done], default `todo`
- `kind` — optional string (e.g. `alignment`, `review`, `launch-readiness`)
- `assignee` — optional string
- `discovered_against` — optional ref to another spec
- `started` — optional date
- `done` — optional date

PRD-level tasks are coordination shaped (alignment checkpoints,
stakeholder reviews, launch readiness). Delivery tasks live on child
features.

## Owner

- Values: [pm, engineering, qa, devops, design, docs]
- Default: `pm`
- Classification: org-state
- Lifecycle triggers: none v1 — PRDs are PM-led through their life.
  Bitemporally tracked when handoffs happen.

## Sections

- Required (pitch kind): `Problem`, `Appetite`, `Solution`, `Rabbit Holes`,
  `No-Gos`
- Required (ten-section kind): `Problem`, `Goals & Success Metrics`,
  `Users & Personas`, `Solution`, `User Flows`, `Acceptance Criteria`,
  `Out of Scope`, `Risks`, `Open Questions`, `Timeline`
- Required (lightweight kind): `Problem`, `Solution`
- Optional (all kinds): `Tasks`, `Linked specs`, `Notes`

## Accepting Commands

- `/design` — draft or refine a PRD
- `/refine` — sharpen problem framing or solution detail
- `/handoff` — flip to engineering when ready

## Default Agents

- authoring: `prd-author`
- review: `pm-reviewer`

## Relations

- `parent → initiative` (cardinality: zero-or-one)
- `children → feature` (cardinality: many)
- `children → epic` (cardinality: many)
