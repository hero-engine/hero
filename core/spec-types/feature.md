---
title: Feature
type: feature
domain: core
category: work
bucket: features
location: .hero/planning/features/{slug}/spec.md
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
