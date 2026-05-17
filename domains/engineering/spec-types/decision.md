---
title: Decision
type: decision
domain: engineering
category: knowledge
bucket: decisions
location: .hero/knowledge/decisions/{slug}.md
lifecycle:
  states: [proposed, accepted, superseded]
  initial: proposed
  terminal: [accepted, superseded]
  transitions:
    - { from: proposed, to: accepted, gate: "decision finalized" }
    - { from: accepted, to: superseded, gate: "replaced by another decision" }
owner:
  values: [pm, engineering, qa, devops, design, docs]
  default: engineering
  classification: org-state
sections:
  required: [Decision, Rationale]
  optional: [Context, Alternatives, Consequences, Notes]
accepting_commands: [/decide, /refine]
default_agents:
  authoring: spec-writer
  review: engineering-reviewer
relations:
  - { kind: supersedes, target_type: decision, cardinality: zero-or-one }
  - { kind: related, target_type: feature, cardinality: many }
---

# Decision spec-type

A **decision** is an architectural or process choice with explicit
rationale — Hero's ADR (Architecture Decision Record) shape.
Engineering-led but cross-domain-visible: PM, design, ops can all
author and read decisions when they constrain engineering.

## When to use

- Choosing between concrete alternatives with non-obvious tradeoffs.
- Locking in a constraint future code must respect.
- Recording why an existing approach was kept when re-evaluated.

## When NOT to use

- A coding style or pattern that recurs in the codebase — that's a
  **convention**.
- A vague intent without committed tradeoff — that's a **note**.

## Lifecycle

States: `proposed → accepted → superseded`.

- `proposed → accepted` — gate: review pass.
- `accepted → superseded` — gate: a newer decision replaces this one.

## Sections

- Required: `Decision`, `Rationale`
- Optional: `Context`, `Alternatives`, `Consequences`, `Notes`

## Accepting Commands

- `/decide` — author a new decision
- `/refine` — revise an existing decision

## Default Agents

- authoring: `spec-writer`
- review: `engineering-reviewer`
