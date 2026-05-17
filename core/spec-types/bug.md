---
title: Bug
type: bug
domain: core
category: work
bucket: bugs
location: .hero/planning/bugs/{slug}/spec.md
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
