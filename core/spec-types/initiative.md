---
title: Initiative
type: initiative
domain: core
category: work
bucket: initiatives
location: .hero/planning/initiatives/{slug}/spec.md
lifecycle:
  states: [planning, refined, ready, delivering, in-review, completed]
  initial: planning
  terminal: [completed]
  transitions:
    - { from: planning, to: refined, gate: "scope and outcome articulated" }
    - { from: refined, to: ready, gate: "child PRDs/epics drafted" }
    - { from: ready, to: delivering, gate: "first child feature picked up", owner_flip: { to: engineering } }
    - { from: delivering, to: in-review, gate: "child work completing" }
    - { from: in-review, to: completed, gate: "outcome achieved or accepted dropped" }
owner:
  values: [pm, engineering, qa, devops, design, docs]
  default: pm
  classification: org-state
sections:
  required: [Goal]
  optional: [Outcome, Bets, Risks, Notes]
accepting_commands: [/design, /refine, /handoff]
default_agents:
  authoring: spec-writer
  review: pm-reviewer
  handoff: handoff-coordinator
relations:
  - { kind: child, target_type: prd, cardinality: many }
  - { kind: child, target_type: epic, cardinality: many }
  - { kind: child, target_type: feature, cardinality: many }
frontmatter:
  required:
    - { name: title, type: string, required: true, classification: content, description: "One-line human title for the initiative." }
    - { name: type, type: enum, required: true, values: [initiative], default: initiative, classification: content, description: "Spec type discriminator; always 'initiative'." }
    - { name: status, type: enum, required: true, values: [planning, refined, ready, delivering, in-review, completed], default: planning, classification: org-state, description: "Lifecycle position." }
  optional:
    - { name: created, type: date, format: "YYYY-MM-DD", classification: content, description: "Authoring date." }
    - { name: priority, type: enum, values: [P0, P1, P2, P3], classification: org-state, description: "Hero-level priority." }
    - { name: horizon, type: enum, values: [now, next, someday, parking], default: now, classification: content, description: "Temporal segmentation per spec-prioritization. Initiatives commonly use 'next' or 'someday' for roadmap items." }
    - { name: tags, type: "list[string]", classification: content, description: "Free-form labels for grouping and search." }
    - { name: claimed_by, type: string, classification: org-state, description: "Who is actively shaping this initiative." }
    - { name: tracker_id, type: string, classification: org-state, description: "External tracker issue ID (e.g. PROJ-123)." }
    - { name: owner, type: enum, values: [pm, engineering, qa, devops, design, docs], default: pm, classification: org-state, description: "Current owning role; defaults to PM since initiatives are scope-side artifacts." }
    - { name: relations, type: "list[relation]", classification: content, description: "Child PRDs/epics/features that decompose this initiative." }
    - { name: smoke, type: "object|enum", values: [deferred, none], classification: content, description: "Smoke-test wiring; usually 'deferred' for initiatives since delivery lives on children." }
---

# Initiative spec-type

An **initiative** is the top-level aspirational strategic bet. It is the
coarsest work artifact in Hero — multi-quarter or cycle-spanning, and the
unit that anchors PRDs. Engineering uses it today for multi-spec strategic
themes; PM uses it for roadmap-shaped bets. Same artifact, same shape.

Initiatives carry **rationale and evidence**, not implementation detail.
Children (epics, features, PRDs) decompose the bet into deliverable work.

## When to use

- A multi-quarter strategic bet that needs explicit prioritization.
- Anything you'd want to show on a roadmap.
- An umbrella for a multi-spec coordinated effort that doesn't fit a single
  epic.

## When NOT to use

- A single-spec improvement — write the feature directly.
- A coherent mid-tier delivery grouping that ships together — that's an
  **epic**.
- An inbound signal — that's an **intake** item; triage first.

## Lifecycle

States: `candidate → committed → in-flight → shipped` (terminal); plus
`dropped` (terminal) reachable from `candidate` or `committed`.

- `candidate → committed` — gate: promoted to active (evidence threshold
  met, priority ranked, PRD optional).
- `committed → in-flight` — gate: first child epic/feature moves to
  delivering.
- `in-flight → shipped` — gate: all linked child epics/features completed,
  or explicit close-with-residual decision.
- `candidate → dropped` — gate: rejected with reason.
- `committed → dropped` — gate: deprioritized with reason (rare; logged).

Horizon assignment (`horizon: now | next | later`, or quarter string e.g.
`q3-2026`) is the standard portfolio filter. No `kind` field v1.

## Kind

No `kind` values v1. Initiatives use `horizon: now | next | later` or a
quarter string for time-based teams. The Roadmap board's default grouping
is by horizon.

## Tasks Schema

- Section heading: `Tasks`
- Required: false
- History: bitemporal

Item shape:

- `id` — string, required, format `T-<int>`
- `text` — string, required
- `status` — enum [todo, doing, done], default `todo`
- `kind` — optional, free string (e.g. `coordination`, `evidence`,
  `launch-readiness`)
- `assignee` — optional string
- `discovered_against` — optional ref to another spec
- `started` — optional date
- `done` — optional date

Initiative-level tasks are coordination shaped (stakeholder alignment,
success-metric instrumentation, launch readiness), not delivery work.
Delivery lives on child features.

## Owner

- Values: [pm, engineering, qa, devops, design, docs]
- Default: `pm`
- Classification: org-state
- Lifecycle triggers: none v1 — initiatives stay owned by PM through their
  life; child features carry per-spec owner flips on the standard
  `ready → delivering` transition.

## Sections

- Required: `Goal`
- Optional: `Outcome`, `Bets`, `Risks`, `Notes`
- Suggested (not required by the registry): `Bet`, `Evidence`, `Tradeoffs`,
  `Open Questions`, `Dependencies`, `Tasks`, `Linked delivery`

The registry treats only `Goal` as required so initiatives stay
methodology-neutral. Shape Up–flavored workspaces typically also author
`Bet`, `Evidence`, and `Tradeoffs`; those are surfaced here as suggested
sections and can be promoted to required for a team via a methodology
profile's `sections_overrides:` block (same pattern as `lifecycle_overrides`)
when that feature lands.

## Accepting Commands

- `/design` — draft or refine an initiative
- `/refine` — sharpen rationale and evidence
- `/decide` — sub-decision lands as a decision spec linked back
- `/handoff` — cross-repo or cross-domain handoff

## Default Agents

- authoring: `product-strategist`
- curator: `roadmap-curator`
- prioritization: `prioritization-strategist`
- review: `pm-reviewer`

## Relations

- `children → epic` (cardinality: many)
- `children → feature` (cardinality: many; when no epic intermediates)
- `children → prd` (cardinality: zero-or-one; one PRD per initiative
  canonical, optional)
- `links → intake` (cardinality: many)
