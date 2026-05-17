---
title: Chore
type: chore
domain: core
category: work
bucket: chores
location: .hero/planning/chores/{slug}/spec.md
lifecycle:
  states: [planning, ready, delivering, completed]
  initial: planning
  terminal: [completed]
  transitions:
    - { from: planning, to: ready, gate: "scope confirmed" }
    - { from: ready, to: delivering, gate: "claimed", owner_flip: { to: engineering } }
    - { from: delivering, to: completed, gate: "done" }
owner:
  values: [pm, engineering, qa, devops, design, docs]
  default: engineering
  classification: org-state
tasks_schema:
  required: false
  section_heading: Tasks
  history: bitemporal
  item_shape:
    id: { type: string, required: true, format: "T-<int>" }
    text: { type: string, required: true }
    status: { type: enum, values: [todo, doing, done], default: todo }
    assignee: { type: string, required: false }
    started: { type: date, required: false }
    done: { type: date, required: false }
sections:
  required: [Goal]
  optional: [Tasks, Notes]
accepting_commands: [/deliver, /handoff]
default_agents:
  authoring: spec-writer
  delivery: engineer
  handoff: handoff-coordinator
relations:
  - { kind: parent, target_type: epic, cardinality: zero-or-one }
---

# Chore spec-type

A **chore** is operational or maintenance work — the small do-it-done
units that keep the system healthy. Dependency bumps, doc cleanups,
config rotations, dead-code removal, lint-fix sweeps. Distinct from
features (no user-visible capability change) and bugs (no defect being
fixed). Engineering-owned by default.

## When to use

- Dependency updates without behavior change.
- Doc fixes, README updates, comment cleanups.
- Config rotations, secret rotations, infra hygiene.
- Dead code removal, lint sweeps.
- Sub-tasks of larger work that don't merit their own feature.

## When NOT to use

- User-visible capability change — that's a **feature**.
- Fixing a defect — that's a **bug**.
- Infra work that supports a feature delivery — folds under the parent
  feature (`kind: infra`).

## Lifecycle

States (default work lifecycle, abbreviated): `planning → ready →
delivering → completed` (terminal).

- `planning → ready` — gate: scope clear; reviewer pass optional for
  trivial chores.
- `ready → delivering` — gate: engineering claim. **owner_flip: to
  engineering.**
- `delivering → completed` — gate: merged.

Chores skip `refined` and `in-review` by default — the do-it-done shape
doesn't need a refinement gate or a separate review phase. PRs land on
merge; AC is typically a single line ("dependency upgraded, tests pass").

## Kind

No `kind` values v1. The shape is intentionally minimal.

## Tasks Schema

- Section heading: `Tasks`
- Required: false
- History: bitemporal

Item shape:

- `id` — string, required, format `T-<int>`
- `text` — string, required
- `status` — enum [todo, doing, done], default `todo`
- `kind` — optional string
- `assignee` — optional string
- `discovered_against` — optional ref to another spec
- `started` — optional date
- `done` — optional date

Tasks rarely appear on chores — the whole chore is usually one task's
worth of work. When present, they typically capture sub-steps in a
multi-step rotation.

## Owner

- Values: [pm, engineering, qa, devops, design, docs]
- Default: `engineering`
- Classification: org-state
- Lifecycle triggers:
  - `ready → delivering`: flip owner to `engineering` (no-op when already
    engineering)

## Sections

- Required: `Description`
- Optional: `Acceptance Criteria`, `Tasks`, `Notes`

## Accepting Commands

- `/design` — author a chore (lightweight)
- `/deliver` — engineering pickup
- `/handoff` — cross-repo handoff if applicable

## Default Agents

- authoring: `engineer`
- delivery: `engineer`

## Relations

- `parent → feature` (cardinality: zero-or-one; chore done in support of
  a feature)
- `parent → epic` (cardinality: zero-or-one)
