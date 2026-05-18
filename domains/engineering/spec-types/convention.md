---
title: Convention
type: convention
domain: engineering
category: knowledge
bucket: conventions
location: .hero/knowledge/conventions/{slug}.md
lifecycle:
  states: [draft, active, superseded]
  initial: draft
  terminal: [active, superseded]
  transitions:
    - { from: draft, to: active, gate: "convention adopted" }
    - { from: active, to: superseded, gate: "replaced by another convention" }
owner:
  values: [pm, engineering, qa, devops, design, docs]
  default: engineering
  classification: org-state
sections:
  required: [Convention, Scope]
  optional: [Rationale, Examples, Anti-patterns, Notes]
accepting_commands: [/convention, /refine]
default_agents:
  authoring: spec-writer
  review: engineering-reviewer
relations:
  - { kind: supersedes, target_type: convention, cardinality: zero-or-one }
  - { kind: related, target_type: decision, cardinality: many }
frontmatter:
  required:
    - { name: title, type: string, required: true, classification: content, description: "One-line human title for the convention." }
    - { name: type, type: enum, required: true, values: [convention], default: convention, classification: content, description: "Spec type discriminator; always 'convention'." }
    - { name: status, type: enum, required: true, values: [draft, active, superseded], default: draft, classification: org-state, description: "Lifecycle position." }
  optional:
    - { name: created, type: date, format: "YYYY-MM-DD", classification: content, description: "Authoring date." }
    - { name: tags, type: "list[string]", classification: content, description: "Free-form labels." }
    - { name: scope, type: "list[string]", classification: content, description: "Glob patterns the convention applies to." }
    - { name: subproject, type: string, classification: content, description: "Monorepo subproject scope identifier; empty = workspace root." }
    - { name: owner, type: enum, values: [pm, engineering, qa, devops, design, docs], default: engineering, classification: org-state, description: "Current owning role." }
    - { name: relations, type: "list[relation]", classification: content, description: "Supersedes/related edges." }
---

# Convention spec-type

A **convention** is a recurring pattern or norm the codebase follows —
a standard for how a class of problem gets solved. Engineering-led and
cross-domain-visible.

## When to use

- A pattern that already appears multiple times and should be
  standardized.
- A team norm (commit message shape, file naming, error handling
  style) that should be discoverable by new contributors.

## When NOT to use

- A one-time architectural choice — that's a **decision**.
- A loose observation — that's a **note**.

## Lifecycle

States: `draft → active → superseded`.

## Sections

- Required: `Convention`, `Scope`
- Optional: `Rationale`, `Examples`, `Anti-patterns`, `Notes`

## Accepting Commands

- `/convention` — author a new convention
- `/refine` — revise an existing convention

## Default Agents

- authoring: `spec-writer`
- review: `engineering-reviewer`
