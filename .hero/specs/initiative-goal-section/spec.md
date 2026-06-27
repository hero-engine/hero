---
title: "Initiative `## Goal` section — the loop opener, parallel to a spec's Kickoff"
slug: initiative-goal-section
type: feature
status: completed
priority: high
horizon: now
tags: [drive, goal, initiative, queue, spec-sections]
created: 2026-06-27
relations:
  - target: drive-autonomous-initiative-execution
    kind: parent
delivery_method: manual
completed_at: 2026-06-27T21:43:20Z
---

# Initiative `## Goal` section — the loop opener, parallel to a spec's Kickoff

## Goal

Give initiatives a first-class **`## Goal`** section that is both the
human-readable objective **and** the machine-checkable stop-condition for a
Drive run — authored once, pasted straight into the harness `/goal`. It is
to an initiative what `## Kickoff` is to a leaf spec: where a Kickoff opens
*one session on one spec*, a Goal opens *the loop over the whole
initiative*. The queue learns to surface an initiative's `## Goal` (start a
run) where it surfaces a leaf spec's `## Kickoff` (start a session).

## Kickoff

Add the `## Goal` section as a recognized, parseable section on initiative
specs. Mirror the existing `Kickoff()` accessor in
[internal/spec/spec.go:1218](../../../../../internal/spec/spec.go) with a
`GoalSection()` accessor. Define the canonical condition shape: "every
child spec reports `hero verify` PASS — OR a `needs_me` pause is raised."
Then teach the queue ([internal/cli/queue.go](../../../../../internal/cli/queue.go),
`renderSpecsKickoff`) to render an initiative's Goal instead of nagging it
for a Kickoff. Leaf specs are unchanged. No predicate logic here — this
spec is the *container and surface* only; `needs_me` and `--check` live in
sibling specs.

## Problem

Today initiatives are structurally just "one more spec": they are not
excluded from Kickoff handling ([internal/spec/select.go:117](../../../../../internal/spec/select.go)
only excludes knowledge and pre-commitment), so an initiative is expected
to carry a Kickoff — a *start-one-session* prompt — when what it actually
needs is a *start-the-run* condition. There is no place to author the
objective-as-stop-condition that Drive consumes. Without it, the `/goal`
condition has to be hand-written every run and lives nowhere durable.

## Design

### The section

A new `## Goal` section on `type: initiative` specs, authored between the
title and `## Problem` (the slot a leaf spec uses for `## Kickoff`). Two
halves in one section:

- **Objective** — prose: what "done" means for this initiative.
- **Condition** — the machine-checkable line(s) Drive feeds `/goal`.
  Canonical default, auto-derivable from children:

  > Run until **every child spec of this initiative reports `hero verify`
  > PASS**, OR a **`needs_me` pause** is raised. On pause, stop and surface
  > the question.

Authors may tighten the condition (e.g. "children A, B, C only"; "and
`hero check` is clean") but the default requires no authoring — Drive can
synthesize it from the parent/child graph if the section is absent, and
`initiative-goal-section` provides the writer that materializes it into the
file so it is visible and editable.

### Accessor

```go
// GoalSection returns the initiative's `## Goal` section body — the
// paste-ready run opener. Empty for non-initiative specs.
func (s *Spec) GoalSection() string
```

Symmetric with `Kickoff()`. A helper `IsRunnable()` (initiative + has or
can-derive a Goal condition) parallels the Kickoff-readiness check.

### Queue rendering

In `queue.go`, when a ready item is an initiative, render its `## Goal`
(labeled "Run") instead of routing it through `renderSpecsKickoff`. Leaf
specs continue to render `## Kickoff` ("Session"). `hero check` stops
flagging initiatives for a missing Kickoff and instead (advisory) notes a
missing/underspecified Goal.

## Acceptance Criteria

- WHERE a spec is `type: initiative`, THE SYSTEM SHALL parse and expose its
  `## Goal` section via `GoalSection()`.
- WHEN an initiative has no authored `## Goal` condition, THE SYSTEM SHALL
  derive the canonical default condition from its child specs.
- WHEN `hero queue` surfaces an initiative, THE SYSTEM SHALL render its
  `## Goal` ("Run") rather than nag it for a `## Kickoff`.
- WHILE a spec is a leaf (non-initiative), THE SYSTEM SHALL render
  `## Kickoff` exactly as today (no regression).
- IF an initiative lacks both an authored and a derivable Goal condition,
  THEN `hero check` SHALL flag it advisory (not block).

## Test Plan

- Unit: `GoalSection()` on initiative with/without the section; on a leaf
  spec returns empty.
- Unit: default-condition derivation from a parent with N children.
- Golden: `hero queue` output renders an initiative's Goal block and a leaf
  spec's Kickoff block, side by side, no cross-contamination.
- Regression: existing queue golden tests for leaf specs unchanged.

## Risks

- **Section-slot collision** — initiatives that already carry a `## Goal`
  used as plain prose. Mitigation: the section *is* prose plus an optional
  condition block; existing prose-only Goals still parse, condition derives
  by default.
- **Scope creep into `--check`** — keep all judging out of this spec; it
  only surfaces and stores the condition.

## Changes

- `internal/spec/spec.go` — add `GoalSection()`, `RunCondition(bySlug)`,
  `ChildSlugs(bySlug)`, `authoredRunCondition()`; add `sort` import.
- `internal/cli/list.go` — `renderSpecsKickoff` branches on
  `TypeInitiative` to surface the `## Goal` run-opener (with a `/drive`
  arm hint) instead of a per-session Kickoff.
- `internal/cli/check.go` — add `missingGoalInitiatives()` and an
  advisory `initiative-goal-coverage` row (does not bump the issue count).
- Tests: `internal/spec/spec_test.go`, `internal/cli/list_test.go`,
  `internal/cli/check_test.go`.

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Initiative parses/exposes `## Goal` via `GoalSection()` | DONE | `internal/spec/spec.go` `GoalSection()`; `TestGoalSectionInitiativeOnly` (initiative non-empty, feature empty) |
| 2 | Derive canonical default condition from children when unauthored | DONE | `RunCondition()`/`ChildSlugs()`; `TestRunConditionDerivedFromChildren`, `TestRunConditionPrefersAuthored` |
| 3 | `hero queue` renders initiative `## Goal` ("Run"), no Kickoff nag | DONE | `internal/cli/list.go` branch; `TestQueueRendersInitiativeGoalOpener`; exercised live |
| 4 | Leaf specs render `## Kickoff` exactly as today (no regression) | DONE | leaf path untouched; `TestQueueRendersKickoffBody` still passes |
| 5 | `hero check` advisory when initiative lacks a Goal (non-blocking) | DONE | `internal/cli/check.go` `missingGoalInitiatives()` + advisory row; `TestMissingGoalInitiatives` |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Add accessors to `internal/spec/spec.go` | DONE | 4 funcs + `sort` import |
| 2 | Branch `renderSpecsKickoff` in `internal/cli/list.go` | DONE | initiative → Goal opener |
| 3 | Advisory in `internal/cli/check.go` | DONE | non-blocking row |
| 4 | Tests across spec/list/check | DONE | 7 new tests, all passing |

### Exercise-the-feature check

- [x] User-visible behavior was exercised end-to-end: built the binary and ran `hero queue --format kickoff` against the real workspace — the `drive-autonomous-initiative-execution` initiative renders ``_Run opener — arm with `/drive drive-autonomous-initiative-execution`_`` followed by its `## Goal` body, not a Kickoff nag.

### Excellence Bar self-check

- [x] yes — surgical change, scoped to the spec's container/surface role (no predicate or `--check` logic leaked in), 7 tests covering every AC, no regressions in `internal/spec` or `internal/cli`.
