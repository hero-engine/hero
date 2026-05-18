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
frontmatter:
  required:
    - { name: title, type: string, required: true, classification: content, description: "One-line human title for the decision." }
    - { name: type, type: enum, required: true, values: [decision], default: decision, classification: content, description: "Spec type discriminator; always 'decision'." }
    - { name: status, type: enum, required: true, values: [proposed, accepted, superseded], default: proposed, classification: org-state, description: "Lifecycle position." }
  optional:
    - { name: created, type: date, format: "YYYY-MM-DD", classification: content, description: "Authoring date." }
    - { name: tags, type: "list[string]", classification: content, description: "Free-form labels." }
    - { name: subproject, type: string, classification: content, description: "Monorepo subproject scope identifier; empty = workspace root." }
    - { name: owner, type: enum, values: [pm, engineering, qa, devops, design, docs], default: engineering, classification: org-state, description: "Current owning role." }
    - { name: relations, type: "list[relation]", classification: content, description: "Supersedes/related edges." }
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
