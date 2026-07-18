---
name: prd
title: Product Requirement Document
domain: pm
description: Largest authoring artifact. Pitch-shaped by default under cycle preset; ten-section shape under sprint/flow/phased presets. References child features and a parent initiative.
location: .hero/planning/prds/{slug}/spec.md
kind:
  values: [pitch, ten-section, lightweight]
  default: ten-section
  note: |
    Canonical PRD shapes. `pitch` = Shape Up pitch (Problem / Appetite /
    Solution / Rabbit Holes / No-Gos); `ten-section` = the canonical agile
    PRD shape; `lightweight` = single-section problem framing for small
    bets. The vocabulary preset may rename "PRD" (e.g. shape-up renders
    `prd.pitch` as "Pitch Doc").
lifecycle:
  states: [draft, review, approved, delivered]
  initial: draft
  terminal: [approved, delivered]
  transitions:
    - from: draft
      to: review
      gate: "pm-reviewer pass"
    - from: review
      to: approved
      gate: "PM approval action"
    - from: review
      to: draft
      gate: "review findings require rework"
    - from: approved
      to: delivered
      gate: "all child specs completed"
frontmatter:
  required:
    - { name: title, type: string, classification: content }
    - { name: type, type: enum, values: [prd], classification: content }
    - { name: kind, type: enum, values: [pitch, ten-section, lightweight], classification: content }
    - { name: status, type: enum, values: [draft, review, approved, delivered], classification: org-state }
    - { name: priority, type: enum, values: [P0, P1, P2, P3], classification: content }
    - { name: owner, type: enum, values: [pm, engineering, qa, design, ...], classification: org-state, note: "PRDs are PM-led; owner is typically `pm` for the life of the doc. Bitemporally tracked when handoffs happen." }
  optional:
    - { name: tracker_id, type: string, classification: org-state }
    - { name: tags, type: list[string], classification: content }
    - { name: initiative, type: ref(initiative), classification: content }
    - { name: features, type: list[ref(feature)], classification: content, note: "Child features that decompose this PRD." }
    - { name: metrics, type: list[object], classification: content }
    - { name: created, type: date, classification: org-state }
    - { name: updated, type: date, classification: org-state }
  preset_conditional:
    cycle:
      - { name: appetite, type: enum, values: [small, big], classification: content, note: "Shape Up — 1-2w small / 6w big" }
sections:
  pitch_template:
    required: [Problem, Appetite, Solution, Rabbit Holes, No-Gos]
    optional: [Linked specs, Tasks, Risks, Goals & Success Metrics, Out of Scope]
    note: "Default under cycle preset. Pitch-shape is the Shape Up default."
  ten_section_template:
    required: [Problem, Goals & Success Metrics, Users & Personas, Solution, User Flows, Acceptance Criteria, Out of Scope, Risks, Open Questions, Timeline]
    optional: [Tasks]
    note: "Default under sprint / continuous / phased presets."
authoring_agent: prd-author
review_agent: pm-reviewer
relations:
  - { kind: parent, target_type: initiative, cardinality: one }
  - { kind: children, target_type: feature, cardinality: many }
---

# PRD spec type

A **PRD** is the largest PM authoring artifact. It captures the
*what* and *why* of a product change in enough detail that an
engineering team can pick it up, decompose it into stories, and
deliver against it without back-and-forth on intent.

## When to use

- An initiative has been **promoted to active** and needs detailed
  framing.
- The work spans **multiple features** or is large enough that a single
  feature alone won't capture the context.
- **Stakeholders need to align** on tradeoffs before engineering
  invests effort.
- Under **cycle preset**, when an idea graduates from a fat-marker
  sketch into a pitch the team will bet on.

## When NOT to use

- A single-feature improvement — write the feature directly.
- An ambiguous customer signal — `/triage` it as an intake first;
  PRD comes after the opportunity is understood.
- A strategy-level objective — that's an OKR (deferred to v2) or a
  decision spec.

## Default template — cycle preset (Shape Up pitch)

```
# {Title}

## Problem
What's broken or missing. Concrete, customer-grounded.

## Appetite
Small (1-2 weeks) or Big (6 weeks). The budget, not the estimate.

## Solution
Fat-marker sketches. Breadboards. What we'd build, at a level the
team can react to without overcommitting.

## Rabbit Holes
Specific traps to avoid. "Don't build configurable X." "Skip the
edge case where Y."

## No-Gos
Work explicitly excluded from this appetite. Defends against scope
creep.

## Linked features
Child features that decompose this pitch. Filled in during /refine.

## Tasks
PRD-level coordination tasks (alignment checkpoints, stakeholder reviews,
launch readiness). Parsed identically to AC; `hero task add | list | done`.

## Risks
What we'd discover too late if we didn't probe now.
```

## Default template — non-cycle presets (ten-section)

```
# {Title}

## Problem
## Goals & Success Metrics
## Users & Personas
## Solution
## User Flows
## Acceptance Criteria
## Out of Scope
## Risks
## Open Questions
## Timeline
```

## Authoring rules

- The **default agent** is `prd-author`. It loads `prd-structure` and
  `prd-anti-patterns` skills.
- Under cycle preset, the pitch template is enforced (Appetite and
  No-Gos must be non-empty before a PRD can move to `review`).
- Acceptance criteria, when present in a PRD, prefer EARS via the
  `acceptance-criteria-ears` skill.
- Metrics belong to `metrics-analyst`.

## Anti-patterns

- PRDs that read like engineering tickets (too prescriptive about
  *how*).
- PRDs with no Appetite (cycle preset) or no Timeline (phased preset).
- PRDs with empty No-Gos — that's where most scope creep enters.
- PRDs that duplicate intake content rather than synthesizing it.

See `prd-anti-patterns` skill for the full list.

## Conflict policy

Fields tagged `content` (title, problem statement, AC, metrics) —
**Hero wins** on conflict with tracker writes. Fields tagged
`org-state` (status, owner, tracker_id) — **tracker wins**. See
[tracker-fronting-and-local-first](../../../.hero/knowledge/decisions/tracker-fronting-and-local-first.md).
