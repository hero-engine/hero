---
title: Intake
type: intake
domain: core
category: work
bucket: intake
location: .hero/planning/intake/{slug}/spec.md
lifecycle:
  states: [planning, triaged, promoted, rejected, merged]
  initial: planning
  terminal: [promoted, rejected, merged]
  transitions:
    - { from: planning, to: triaged, gate: "intake reviewed" }
    - { from: triaged, to: promoted, gate: "promoted to feature/epic/bug" }
    - { from: triaged, to: rejected, gate: "rejected with reason" }
    - { from: triaged, to: merged, gate: "merged into another intake" }
kind:
  values: [customer, support, sales, internal, competitive]
  default: customer
  required: false
owner:
  values: [pm, engineering, qa, devops, design, docs]
  default: pm
  classification: org-state
sections:
  required: [Signal]
  optional: [Source, Notes]
accepting_commands: [/intake, /refine, /handoff]
default_agents:
  authoring: spec-writer
  review: pm-reviewer
  handoff: handoff-coordinator
relations:
  - { kind: promotes_to, target_type: feature, cardinality: zero-or-one }
  - { kind: promotes_to, target_type: bug, cardinality: zero-or-one }
  - { kind: promotes_to, target_type: epic, cardinality: zero-or-one }
frontmatter:
  required:
    - { name: title, type: string, required: true, classification: content, description: "One-line human title for the intake." }
    - { name: type, type: enum, required: true, values: [intake], default: intake, classification: content, description: "Spec type discriminator; always 'intake'." }
    - { name: status, type: enum, required: true, values: [planning, triaged, promoted, rejected, merged], default: planning, classification: org-state, description: "Lifecycle position." }
  optional:
    - { name: created, type: date, format: "YYYY-MM-DD", classification: content, description: "Authoring date." }
    - { name: priority, type: enum, values: [P0, P1, P2, P3], classification: org-state, description: "Hero-level priority." }
    - { name: tags, type: "list[string]", classification: content, description: "Free-form labels." }
    - { name: tracker_id, type: string, classification: org-state, description: "External tracker issue ID." }
    - { name: kind, type: enum, values: [customer, support, sales, internal, competitive], default: customer, classification: content, description: "Intake source category." }
    - { name: owner, type: enum, values: [pm, engineering, qa, devops, design, docs], default: pm, classification: org-state, description: "Current owning role." }
    - { name: relations, type: "list[relation]", classification: content, description: "Promotes-to edges into work specs." }
---

# Intake spec-type

An **intake** is an inbound signal — a customer asking for something, a
support escalation, a sales note, a competitive observation. It's the
funnel from raw signal to roadmap decision: triaged, then promoted to
an initiative / epic / feature, merged into another intake, or rejected.

The defining property: **source attribution is the trust signal**. The
customer's own words, the link back to the ticket, the segment they're
in — these are what make intake usable for prioritization later.
Paraphrasing erases trust.

## When to use

- Anything inbound that could shape the roadmap but hasn't been evaluated
  yet.
- Customer feedback in any form (ticket, call, NPS comment, sales note).
- Internal asks from sales, support, leadership, engineering.
- Competitive signals worth weighing.

## When NOT to use

- A clear, well-scoped customer request that maps obviously to an existing
  initiative — link it directly via the initiative's evidence section.
- Bug reports with reproduction steps — those route to `/diagnose` as a
  bug.

## Lifecycle

States: `planning → triaged → promoted` (terminal); plus `rejected` and
`merged` (terminal) reachable from `planning` or `triaged`. This mirrors
the authoritative frontmatter `lifecycle:` block above and the engine
`Status` constants (planning/triaged/promoted/rejected/merged).

- `planning → triaged` — gate: intake-triager classified and clustered
  (target SLA: 24h).
- `triaged → promoted` — gate: promoted to a feature / bug / epic
  (`hero intake promote <slug>`), writing the `derived_from` provenance edge.
- `triaged → rejected` — gate: rejected with reason
  (`hero intake reject <slug>`).
- `triaged → merged` — gate: merged into another intake.
- `planning → rejected` — gate: obvious reject at intake (spam,
  off-product, duplicate of recently rejected).

## Kind

Values: `[customer, support, sales, internal, competitive]`

- `customer` — direct customer feedback
- `support` — support escalation
- `sales` — sales-originated request or signal
- `internal` — internal stakeholder ask (leadership, ops, engineering)
- `competitive` — competitive observation worth weighing

Default: `customer`. Required: false.

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

Tasks on intake items capture triage follow-ups (additional outreach to
the customer, evidence to gather before linking, internal stakeholders
to consult).

## Owner

- Values: [pm, engineering, qa, devops, design, docs]
- Default: `pm`
- Classification: org-state
- Lifecycle triggers: none v1. Intake is PM-owned through its life.

## Sections

- Required: `Signal`
- Optional: `Investigation`, `Tasks`, `Linked decision`, `Notes`

## Accepting Commands

- `/import` — funnel from tracker as intake
- `/design` — promote intake to a higher-tier artifact
- `/note` — quick capture as intake

## Default Agents

- authoring: `intake-triager`
- investigation: `pm-investigator`
- duplicate-detection: `duplicate-detector`

## Relations

- `links → initiative` (cardinality: zero-or-one)
- `links → epic` (cardinality: zero-or-one)
- `links → feature` (cardinality: zero-or-one)
- `merged-into → intake` (cardinality: zero-or-one)
