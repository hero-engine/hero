---
title: Bug
type: bug
domain: core
category: work
bucket: bugs
location: .hero/planning/bugs/{slug}/spec.md
lifecycle:
  states: [planning, refined, ready, delivering, in-review, completed]
  initial: planning
  terminal: [completed]
  transitions:
    - { from: planning, to: refined, gate: "diagnosis written; root cause classified" }
    - { from: refined, to: ready, gate: "reviewer pass; reproduction confirmed" }
    - { from: ready, to: delivering, gate: "engineering claim", owner_flip: { to: engineering } }
    - { from: delivering, to: in-review, gate: "fix implemented; PR open" }
    - { from: in-review, to: completed, gate: "merged + regression test + AC passing" }
    - { from: ready, to: planning, gate: "engineering hands back; needs more investigation" }
kind:
  values: [regression, edge-case, security, data]
  default: regression
  required: false
  description: "Defect sub-category."
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
    kind: { type: enum, values: [feature, bug, chore, refactor, qa-blocker, perf, infra, security, ux], required: false }
    assignee: { type: string, required: false }
    discovered_against: { type: ref(spec), required: false }
    started: { type: date, required: false }
    done: { type: date, required: false }
sections:
  required: [Symptoms, Diagnosis, Acceptance Criteria]
  optional: [Root Cause, Reproduction, Tasks, Risks, Notes]
accepting_commands: [/diagnose, /challenge, /deliver, /handoff]
default_agents:
  authoring: debug-investigator
  review: engineering-reviewer
  delivery: engineer
  handoff: handoff-coordinator
relations:
  - { kind: parent, target_type: feature, cardinality: zero-or-one }
  - { kind: parent, target_type: epic, cardinality: zero-or-one }
  - { kind: discovered_against, target_type: feature, cardinality: zero-or-one }
  - { kind: blocks, target_type: feature, cardinality: many }
  - { kind: blocked_by, target_type: feature, cardinality: many }
frontmatter:
  required:
    - { name: title, type: string, required: true, classification: content, description: "One-line human title for the bug." }
    - { name: type, type: enum, required: true, values: [bug], default: bug, classification: content, description: "Spec type discriminator; always 'bug'." }
    - { name: status, type: enum, required: true, values: [planning, refined, ready, delivering, in-review, completed], default: planning, classification: org-state, description: "Lifecycle position." }
    - { name: severity, type: enum, required: true, values: [critical, high, medium, low], classification: org-state, description: "Defect impact." }
  optional:
    - { name: created, type: date, format: "YYYY-MM-DD", classification: content, description: "Authoring date." }
    - { name: priority, type: enum, values: [P0, P1, P2, P3], classification: org-state, description: "Hero-level priority." }
    - { name: horizon, type: enum, values: [now, next, someday, parking], default: now, classification: content, description: "Temporal segmentation." }
    - { name: pinned, type: bool, default: "false", classification: content, description: "Float to top of `hero queue`." }
    - { name: tags, type: "list[string]", classification: content, description: "Free-form labels." }
    - { name: claimed_by, type: string, classification: org-state, description: "Who is actively working this spec." }
    - { name: delivery_method, type: enum, values: [agent, manual], classification: content, description: "How delivery is executed." }
    - { name: tracker_id, type: string, classification: org-state, description: "External tracker issue ID." }
    - { name: kind, type: enum, values: [regression, edge-case, security, data], default: regression, classification: content, description: "Defect sub-category." }
    - { name: owner, type: enum, values: [pm, engineering, qa, devops, design, docs], default: engineering, classification: org-state, description: "Current owning role." }
    - { name: relations, type: "list[relation]", classification: content, description: "Parent/child/blocks edges." }
    - { name: smoke, type: "object|enum", values: [deferred, none], classification: content, description: "Smoke-test wiring." }
    - { name: size, type: enum, values: [trivial, small, medium, large, x-large, giant], classification: content, description: "Declared effort tier (shared 6-tier ladder). Comfortable band for bugs: trivial..medium; large triggers a soft promotion nudge, x-large/giant recommend `/split` or promotion to an initiative." }
---

# Bug spec-type

A **bug** is a defect — a deviation from intended behavior. It uses a
diagnose-fix lifecycle distinct from features: investigation precedes
implementation, and acceptance criteria typically express the
no-regression bar. Shared between engineering and QA; PM can file bugs
that surface from intake.

## When to use

- Observed defect with reproduction steps or strong signal.
- Regression after a recent ship.
- Edge case that escapes the current AC coverage.

## When NOT to use

- Vague customer complaint without reproduction — file as **intake** and
  triage.
- Requested capability ("can it do X?") — that's a **feature**, not a bug.
- Operational task — that's a **chore**.

## Lifecycle

States (default work lifecycle): `planning → refined → ready → delivering
→ in-review → completed` (terminal).

- `planning → refined` — gate: diagnosis written; root cause classified;
  fix scope sketched.
- `refined → ready` — gate: reviewer pass; reproduction confirmed.
- `ready → delivering` — gate: engineering claim. **owner_flip: to
  engineering.**
- `delivering → in-review` — gate: fix implemented; PR open.
- `in-review → completed` — gate: merged + regression test added + AC
  passing.
- `ready → planning` — gate: engineering hands back with
  `handed_back_reason` (under-diagnosed, needs more investigation).

## Kind

Values: `[regression, edge-case, security, data]`

- `regression` — broke after working previously
- `edge-case` — unexpected behavior under boundary conditions
- `security` — vulnerability or exposure
- `data` — data corruption, drift, or integrity issue

Default: `regression`. Required: false.

## Tasks Schema

- Section heading: `Tasks`
- Required: false
- History: bitemporal

Item shape:

- `id` — string, required, format `T-<int>`
- `text` — string, required
- `status` — enum [todo, doing, done], default `todo`
- `kind` — optional enum [feature, bug, chore, refactor, qa-blocker,
  perf, infra, security, ux]
- `assignee` — optional string
- `discovered_against` — optional ref to another spec
- `started` — optional date
- `done` — optional date

Tasks on bugs typically capture investigation follow-ups (reproduce on
staging, capture log dump, isolate reproducer), regression test
additions, and QA verification handoffs.

## Owner

- Values: [pm, engineering, qa, devops, design, docs]
- Default: `engineering`
- Classification: org-state
- Lifecycle triggers:
  - `ready → delivering`: flip owner to `engineering`
  - `ready → planning` (hand-back): flip owner to `qa` or `pm` depending
    on hand-back reason

## Sections

- Required: `Symptoms`, `Diagnosis`, `Acceptance Criteria`
- Optional: `Root Cause`, `Reproduction`, `Tasks`, `Risks`, `Notes`

## Accepting Commands

- `/diagnose` — investigate and classify root cause
- `/challenge` — push back on a diagnosis with new context
- `/deliver` — engineering fix pickup
- `/handoff` — owner flip or cross-repo handoff

## Default Agents

- authoring: `debug-investigator`
- review: `engineering-reviewer`
- delivery: `engineer`
- handoff: `handoff-coordinator`

## Relations

- `parent → feature` (cardinality: zero-or-one; bug that regresses a
  feature)
- `parent → epic` (cardinality: zero-or-one)
- `discovered_against → feature` (cardinality: zero-or-one)
- `blocks → feature` (cardinality: many)
- `blocked_by → feature` (cardinality: many)
