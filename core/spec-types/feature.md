---
title: Feature
type: feature
domain: core
category: work
bucket: features
location: .hero/planning/features/{slug}/spec.md
lifecycle:
  states: [planning, refined, ready, delivering, in-review, completed]
  initial: planning
  terminal: [completed]
  transitions:
    - { from: planning, to: refined, gate: "spec-writer pass with AC drafted" }
    - { from: refined, to: ready, gate: "pm-reviewer pass; preset-required fields populated" }
    - { from: ready, to: delivering, gate: "engineering claim", owner_flip: { to: engineering } }
    - { from: delivering, to: in-review, gate: "implementation complete; PR open" }
    - { from: in-review, to: completed, gate: "merged + acceptance criteria satisfied" }
    - { from: ready, to: planning, gate: "engineering hands back", owner_flip: { to: pm } }
kind:
  values: [new, refactor, perf, infra, security, ux]
  default: new
  required: false
  description: "Sub-category for feature work; methodology-neutral."
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
  required: [Goal, Acceptance Criteria]
  optional: [Tasks, Boundaries, Risks, User Value, Out of Scope, Dependencies, Notes]
accepting_commands: [/refine, /design, /deliver, /diagnose, /handoff]
extension_points: [lifecycle]
default_agents:
  authoring: story-writer
  review: pm-reviewer
  delivery: engineer
  handoff: handoff-coordinator
relations:
  - { kind: parent, target_type: epic, cardinality: zero-or-one }
  - { kind: parent, target_type: prd, cardinality: zero-or-one }
  - { kind: parent, target_type: initiative, cardinality: zero-or-one }
  - { kind: blocks, target_type: feature, cardinality: many }
  - { kind: blocked_by, target_type: feature, cardinality: many }
frontmatter:
  required:
    - { name: title, type: string, required: true, classification: content, description: "One-line human title for the feature." }
    - { name: type, type: enum, required: true, values: [feature], default: feature, classification: content, description: "Spec type discriminator; always 'feature'." }
    - { name: status, type: enum, required: true, values: [planning, refined, ready, delivering, in-review, completed], default: planning, classification: org-state, description: "Lifecycle position." }
  optional:
    - { name: created, type: date, format: "YYYY-MM-DD", classification: content, description: "Authoring date." }
    - { name: priority, type: enum, values: [P0, P1, P2, P3], classification: org-state, description: "Hero-level priority." }
    - { name: severity, type: enum, values: [critical, high, medium, low], classification: org-state, description: "Hero-level severity (rare on features; use bug)." }
    - { name: horizon, type: enum, values: [now, next, someday, parking], default: now, classification: content, description: "Temporal segmentation per spec-prioritization." }
    - { name: pinned, type: bool, default: "false", classification: content, description: "Float to top of `hero queue` regardless of ranking." }
    - { name: tags, type: "list[string]", classification: content, description: "Free-form labels for grouping and search." }
    - { name: claimed_by, type: string, classification: org-state, description: "Who is actively working this spec." }
    - { name: delivery_method, type: enum, values: [agent, manual], classification: content, description: "How delivery is being executed." }
    - { name: tracker_id, type: string, classification: org-state, description: "External tracker issue ID (e.g. PROJ-123)." }
    - { name: kind, type: enum, values: [new, refactor, perf, infra, security, ux], default: new, classification: content, description: "Sub-category for feature work." }
    - { name: owner, type: enum, values: [pm, engineering, qa, devops, design, docs], default: engineering, classification: org-state, description: "Current owning role." }
    - { name: relations, type: "list[relation]", classification: content, description: "Parent/child/blocks edges to other specs." }
    - { name: smoke, type: "object|enum", values: [deferred, none], classification: content, description: "Smoke-test wiring or escape-hatch sentinel." }
    - { name: size, type: enum, values: [trivial, small, medium, large, x-large, giant], classification: content, description: "Declared effort tier (shared 6-tier ladder). Comfortable band for features: trivial..medium; large triggers a soft promotion nudge, x-large/giant recommend `/split` or promotion to an initiative." }
---

# Feature spec-type

A **feature** is THE unit of work — a user-facing capability change.
Shared between PM and engineering: PM authors and refines; engineering
delivers. The cross-domain handoff is an owner flip on the same artifact,
not a new spec creation.

This is the most common artifact in the workspace. Engineering's existing
137 features stay as authored; PM-authored features land here too. The
vocabulary preset renders the display name ("Story" under agile-scrum,
"Scope" under shape-up, "Card" under kanban) — the frontmatter always
says `type: feature`.

## When to use

- A unit of work that fits one engineering effort (single capability,
  single delivery cycle).
- A customer-asked capability that doesn't need a full PRD to scope.
- Decomposing an epic into deliverable chunks.

## When NOT to use

- Multi-effort work — that's an **epic** with child features.
- Vague signal — that's an **intake**; triage first.
- A defect — that's a **bug** (diagnose-fix lifecycle).
- Operational / maintenance work — that's a **chore**.

## Lifecycle

States (default work lifecycle): `planning → refined → ready → delivering
→ in-review → completed` (terminal).

- `planning → refined` — gate: spec-writer pass with AC drafted.
- `refined → ready` — gate: pm-reviewer pass; preset-required fields
  populated. Ready for owner flip to engineering.
- `ready → delivering` — gate: engineering claim. **owner_flip: to
  engineering.** engineer agent picks up.
- `delivering → in-review` — gate: implementation complete; PR open.
- `in-review → completed` — gate: merged + acceptance criteria satisfied.
- `ready → planning` — gate: engineering hands back with
  `handed_back_reason` (under-specified).

Methodology profiles overlay alternate state machines (e.g. scrum's
`backlog → ready → in_progress → in_review → done`); this lifecycle is
the methodology-neutral default.

## Kind

Values: `[new, refactor, perf, infra, security, ux]`

- `new` — net-new user-facing capability
- `refactor` — internal restructuring; no user-visible change
- `perf` — performance work with measurable target
- `infra` — platform/infrastructure work supporting features
- `security` — security hardening
- `ux` — UX-only iteration (visual, copy, interaction)

Default: `new`. Required: false. Back-fillable on existing features.

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
- `discovered_against` — optional ref to another spec (captures lineage
  when a task surfaces against this feature while working on another)
- `started` — optional date
- `done` — optional date

Tasks are the *next thing to do*. Acceptance Criteria are the *bar to
pass*. Don't conflate them.

## Owner

- Values: [pm, engineering, qa, devops, design, docs]
- Default: `engineering`
- Classification: org-state
- Lifecycle triggers:
  - `ready → delivering`: flip owner to `engineering`
  - `ready → planning` (hand-back): flip owner to `pm`

## Sections

- Required: `Goal`, `Acceptance Criteria`
- Optional: `Tasks`, `Boundaries`, `Risks`, `User Value`, `Out of Scope`,
  `Dependencies`, `Notes`

## Accepting Commands

- `/refine` — sharpen scope or AC
- `/design` — draft a new feature
- `/deliver` — engineering pickup
- `/diagnose` — convert to or pair with a bug investigation
- `/handoff` — owner flip or cross-repo handoff

## Default Agents

- authoring: `story-writer` (canonical pack name; vocabulary preset may
  render as "spec-writer")
- review: `pm-reviewer`
- delivery: `engineer`
- handoff: `handoff-coordinator`

## Relations

- `parent → epic` (cardinality: zero-or-one)
- `parent → prd` (cardinality: zero-or-one)
- `parent → initiative` (cardinality: zero-or-one)
- `blocks → feature` (cardinality: many)
- `blocked_by → feature` (cardinality: many)
