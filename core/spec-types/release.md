---
title: Release
type: release
domain: core
category: work
bucket: releases
location: .hero/planning/releases/{slug}/spec.md
lifecycle:
  states: [planning, committed, in-flight, shipped]
  initial: planning
  terminal: [shipped]
  transitions:
    - { from: planning, to: committed, gate: "scope locked" }
    - { from: committed, to: in-flight, gate: "release window opens" }
    - { from: in-flight, to: shipped, gate: "release shipped" }
owner:
  values: [pm, engineering, qa, devops, design, docs]
  default: engineering
  classification: org-state
sections:
  required: [Goal]
  optional: [Scope, Dates, Risks, Notes]
accepting_commands: [/release, /refine, /handoff]
default_agents:
  authoring: spec-writer
  review: pm-reviewer
  handoff: handoff-coordinator
relations:
  - { kind: child, target_type: sprint, cardinality: many }
  - { kind: contains, target_type: feature, cardinality: many }
  - { kind: contains, target_type: epic, cardinality: many }
frontmatter:
  required:
    - { name: title, type: string, required: true, classification: content, description: "One-line human title for the release." }
    - { name: type, type: enum, required: true, values: [release], default: release, classification: content, description: "Spec type discriminator; always 'release'." }
    - { name: status, type: enum, required: true, values: [planning, committed, in-flight, shipped], default: planning, classification: org-state, description: "Lifecycle position." }
  optional:
    - { name: created, type: date, format: "YYYY-MM-DD", classification: content, description: "Authoring date." }
    - { name: priority, type: enum, values: [P0, P1, P2, P3], classification: org-state, description: "Hero-level priority." }
    - { name: tags, type: "list[string]", classification: content, description: "Free-form labels." }
    - { name: tracker_id, type: string, classification: org-state, description: "External tracker issue ID." }
    - { name: owner, type: enum, values: [pm, engineering, qa, devops, design, docs], default: engineering, classification: org-state, description: "Current owning role." }
    - { name: relations, type: "list[relation]", classification: content, description: "Contains/child edges." }
---

# Release spec-type

A **release** is a large time-box — the unit that gets shipped
(quarterly, monthly, Program Increment, project phase). It groups
features, epics, and initiatives whose delivery converges on a single
shipping event. Distinct from `sprint` (iteration-scale time-box that
checkpoints inside a release).

Methodology profiles decide whether release is required, optional, or
unused. The registry registers the type unconditionally; the active
methodology declares whether a workspace actually authors release
artifacts.

## When to use

- A coordinated ship date across multiple features / epics.
- A quarterly or monthly release cadence.
- A SAFe Program Increment.
- A waterfall phase-gated milestone.

## When NOT to use

- A short iteration time-box — that's a **sprint**.
- A continuous-deployment workspace with no release artifact in the
  methodology — skip.

## Lifecycle

States: `planning → committed → shipping → shipped` (terminal); plus
`cancelled` (terminal) reachable from `planning` or `committed`.

- `planning → committed` — gate: scope locked; features/epics committed
  to the release.
- `committed → shipping` — gate: code freeze or release branch cut.
- `shipping → shipped` — gate: release deployed; success criteria met.
- `planning → cancelled` — gate: release abandoned before commit.
- `committed → cancelled` — gate: rare; logged with reason.

Methodology profiles overlay alternate state machines (e.g. waterfall's
phase-gated states); this is the methodology-neutral default.

## Kind

No `kind` values v1. Release shape varies by methodology — the
methodology profile defines duration, gating, and rollup metrics rather
than a sub-type enum.

## Tasks Schema

- Section heading: `Tasks`
- Required: false
- History: bitemporal

Item shape:

- `id` — string, required, format `T-<int>`
- `text` — string, required
- `status` — enum [todo, doing, done], default `todo`
- `kind` — optional string (e.g. `release-readiness`, `comms`, `qa-sign-off`)
- `assignee` — optional string
- `discovered_against` — optional ref to another spec
- `started` — optional date
- `done` — optional date

Release-level tasks are coordination shaped (release-readiness checks,
release-comms drafts, QA sign-off, rollback drills).

## Owner

- Values: [pm, engineering, qa, devops, design, docs]
- Default: `pm`
- Classification: org-state
- Lifecycle triggers: none v1. Releases are typically PM- or
  release-manager-owned; ownership may flip to `devops` during the
  `shipping` state in some workflows. Bitemporally tracked.

## Sections

- Required: `Scope`, `Success Criteria`
- Optional: `Schedule`, `Risks`, `Tasks`, `Rollback Plan`, `Comms`,
  `Notes`

## Accepting Commands

- `/design` — draft a release
- `/sprint` — plan iteration time-boxes inside the release
- `/release` — assess release readiness
- `/handoff` — cross-repo coordination

## Default Agents

- authoring: `release-manager`
- review: `pm-reviewer`

## Relations

- `contains → feature` (cardinality: many)
- `contains → epic` (cardinality: many)
- `contains → sprint` (cardinality: many; sprints inside this release)
- `parent → initiative` (cardinality: zero-or-one)
