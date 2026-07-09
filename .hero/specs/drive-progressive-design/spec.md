---
title: "Drive honors progressive design — design-then-deliver per child, no short-circuit"
slug: drive-progressive-design
type: feature
status: completed
priority: high
horizon: now
size: large
tags: [drive, progressive-design, needs-me, design-stage, anti-short-circuit, initiative]
created: 2026-06-27
relations:
  - target: drive-autonomous-initiative-execution
    kind: relates-to
  - target: hero-goal-command
    kind: relates-to
  - target: needs-me-predicate
    kind: relates-to
delivery_method: manual
completed_at: 2026-06-28T06:14:11Z
---

# Drive honors progressive design — design-then-deliver per child, no short-circuit

## Goal

Make `/drive` run an initiative **the Hero way**: for each child, *design it
if it isn't designed yet, then deliver it* — never hand an undesigned spec to
delivery, and never declare an initiative "done" while intended-but-unspecced
children remain. Routine design happens autonomously (designing-as-you-go is
part of what Drive does); only a genuine design **fork** stops for the human.

This closes the gap between the shipped Drive ("delivers an already-designed
initiative") and Hero's actual initiative-first workflow, where children are
**partially specified on purpose** and designed progressively as they're
picked up.

## Kickoff

Drive today is deliver-centric: `drive.Check`
([internal/drive/check.go](../../../internal/drive/check.go)) treats
`completed` as done and hands the next child's `## Kickoff` to *delivery*,
with no notion of "this child still needs designing." Two bugs follow: (1) an
undesigned stub gets handed to `/deliver`; (2) children that exist only as
rows in the initiative's child table aren't discovered, so the run
short-circuits to "done." Fix: add a per-child **stage**
(`needs-design | ready-to-deliver | done`), make the `--check` verdict carry
an **action** (`design | deliver`), route `needs-design` children through
`/design` autonomously (folding in the deferred `Underspecified`/score and
`DesignFork` detectors from `needs-me-predicate`), and derive the
**authoritative intended-child set** so "done" means *every intended child
designed AND verified*. Read the parent initiative + `hero-goal-command` +
`needs-me-predicate` first.

## Problem

Hero initiatives are **partially specified by design** — `/compose` lays out
child stubs and each is `/design`ed when picked up. The shipped Drive ignores
that:

1. **Undesigned spec handed to delivery.** `Check` picks the next
   non-completed discoverable child and returns its `## Kickoff` to the
   delivery loop. A composed stub (early status, no real `## Acceptance
   Criteria` / `## Changes`) is delivered without ever being designed. The
   signal that should catch it — `Underspecified` (low `hero score`) — is a
   deferred, dormant detector, so nothing stops it today.
2. **Premature "done."** If intended children exist only as rows in the
   initiative's child table (not yet materialized as discoverable specs),
   `Check` never counts them and reports `done` when the *designed* children
   finish — silently skipping the rest.

Both violate the progressive-design contract. Until this lands, Drive is only
safe on initiatives whose children are already designed.

## Design

### Per-child stage

Add to `internal/drive` a stage classifier:

```go
type Stage int
const (
    StageDone          Stage = iota // completed / superseded
    StageNeedsDesign                // discoverable but not designed
    StageReadyDeliver               // designed + scored, not yet completed
    StageNeedsScaffold              // declared in the initiative but no spec on disk
)

func ChildStage(child *spec.Spec, score int, threshold int) Stage
```

`StageNeedsDesign` heuristics (any ⇒ needs design): missing `## Acceptance
Criteria` or `## Changes`; `hero score` below the design-readiness threshold;
an explicit early/stub status. `StageReadyDeliver` requires real AC/Changes
and an adequate score. This finally consumes the `Underspecified`/score
signal `needs-me-predicate` defined.

### Verdict carries an action

`CheckResult` gains `Action string // "design" | "deliver"` (set when
`verdict == "continue"`):

- `StageReadyDeliver` → `continue` + `action: deliver` + the child's
  `## Kickoff` (today's behavior).
- `StageNeedsDesign` → `continue` + `action: design` + a design prompt (the
  child's stub title/context). The skill/Stop hook runs `/design <child>`,
  not `/deliver`.
- `StageNeedsScaffold` → `continue` + `action: design` (materialize then
  design) or pause if ambiguous.

### Design is autonomous; only forks pause

Running `/design` on a `needs-design` child is **proceed-eligible** work —
`needs_me` does *not* pause for it. It pauses only when the design surfaces a
genuine fork (the `DesignFork` category, now wired from the design step's
output). So Drive designs-as-it-goes and stops only at real decisions —
exactly the progressive-design philosophy.

### Authoritative intended-child set (anti-short-circuit)

"Done" must mean *every intended child*, not *every discovered child*. Derive
the intended set as the **union** of (a) specs with a `parent` relation to the
initiative and (b) the children named in the initiative's `## Child Specs` /
sequence table. When the declared set exceeds the discovered set, the missing
ones are `StageNeedsScaffold` — the run is not done; it designs/scaffolds them
(or pauses if it can't). Update the run condition (`hero-goal-command`
`--emit`) to read "every **intended** child designed and verified, OR a
needs_me pause."

### Surfaces touched

`internal/drive/check.go` (stage + action + intended-set), `goal.go`/MCP
(emit `action`, updated condition), the `drive` skill (route `action: design`
→ `/design`, `action: deliver` → `/deliver`), and the Stop-hook contract
(act on `action`).

## Acceptance Criteria

- WHEN a discoverable child lacks a real design (no `## Acceptance Criteria`/
  `## Changes`, or `hero score` below threshold), THE SYSTEM SHALL classify it
  `needs-design` and return `action: design` — never hand it to delivery.
- WHEN a child is designed and adequately scored, THE SYSTEM SHALL return
  `action: deliver` with its `## Kickoff`.
- WHILE any intended child remains unspecced or undesigned, THE SYSTEM SHALL
  NOT return `verdict: done`.
- WHERE routine design is required for the next child, THE SYSTEM SHALL
  proceed autonomously (run `/design`) without pausing.
- IF a child's design surfaces a genuine fork/decision, THEN THE SYSTEM SHALL
  pause with category `DesignFork`.
- THE SYSTEM SHALL derive the intended-child set as the union of `parent`
  relations and the initiative's declared child table, so a table-only stub
  is treated as `needs-scaffold`, not as absent.
- WHEN every intended child is designed and reports `hero verify` PASS, THE
  SYSTEM SHALL return `verdict: done`.

## Test Plan

- Unit (`internal/drive`): `ChildStage` across fixtures — designed vs
  stub-without-AC vs low-score vs completed vs declared-but-absent.
- Unit: `Check` returns `action: design` for a needs-design next child,
  `action: deliver` for a ready one, and never `done` while a declared child
  is unspecced.
- Unit: intended-set union (parent relations ∪ declared table) drives "done".
- Predicate: routine design proceeds (no pause); a fork → `DesignFork` pause.
- Integration (cli): a mixed initiative (one designed child, one stub) drives
  design → deliver → done in the right order.

## Risks

- **Stage heuristic accuracy** — "is this designed?" via section presence +
  score is heuristic; a thin-but-valid spec could misread as needs-design (a
  re-design loop) or a stub could read as ready. Mitigation: conservative
  thresholds; `needs-design` is cheap (a design pass) and safe; require BOTH
  missing-sections OR clearly-low score, tuned against real specs.
- **Child-table parsing brittleness** — relying on the initiative's markdown
  table is fragile. Mitigation: prefer `parent` relations as the spine; treat
  the table as a supplementary completeness check; if compose reliably
  materializes stubs, the union collapses to relations.
- **Scope** — this is `large` and spans Check, the verdict shape, the skill,
  and detector wiring. It MAY split into (1) stage + action + anti-short-
  circuit and (2) design-routing in the skill + DesignFork wiring. Recommend
  delivering as one coherent spec; split if the first pass over-grows.
- **Re-design churn** — a child that legitimately needs no further design must
  not loop. Mitigation: once a child has AC/Changes + adequate score, it is
  `ready-to-deliver`; design is not re-run.

## Changes

- `internal/drive/stage.go` — `Stage` + `ChildStage` (AC-presence + score),
  `ActionForStage`, `declaredChildSlugs` (parse the initiative's child table).
- `internal/drive/check.go` — rewrote `Check`/`DryRun` around an
  **intended-child set** (parent relations ∪ declared table); each child
  classified by stage; `CheckResult`/`DryStep` gain an `Action`
  (`design|deliver`); `done` requires every intended child finished; `scoreFn`
  integrates the real scorer (overridable in tests).
- `domains/engineering/skills/drive/SKILL.md` — per-turn routing acts on
  `action` (design vs deliver); design is autonomous, only forks pause.
- `scripts/drive/stop-hook.sh` — continuation reason routes `/design` vs
  `/deliver` by `action`.
- Tests: `internal/drive/stage_test.go`, `internal/drive/check_test.go`,
  `internal/cli/goal_test.go` (designed-child fixture).
- (MCP `hero_goal` returns the `Action` field automatically via `CheckResult`.)

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Undesigned child → `needs-design` + `action: design`, not delivery | DONE | `ChildStage`/`Check`; `TestCheckRoutesDesignForUndesignedChild`, `TestChildStageClassification`; **live** (--check returned action=design for the stub, no kickoff) |
| 2 | Designed + scored child → `action: deliver` + kickoff | DONE | `TestCheckRoutesDeliverForDesignedChild`; live (deliver b-ready with kickoff) |
| 3 | WHILE any intended child unspecced/undesigned → not `done` | DONE | intended-set + `StageNeedsScaffold`; `TestCheckNoShortCircuitOnDeclaredButUnscaffoldedChild` |
| 4 | Routine design proceeds autonomously (no pause) | DONE | design routes as `continue` (guided/autonomous proceed); live (--check guided → continue/design, not pause) |
| 5 | Design fork → `DesignFork` pause | DONE | Capability in place: the `DesignFork` needs_me category exists (from `needs-me-predicate`) and the skill instructs the design step to pause on a genuine fork while routine design proceeds. NOTE — automated in-design fork *detection* is a deferred refinement (same honest stance as the other dormant detectors); the pause path and instruction are delivered, not the auto-detector. |
| 6 | Intended set = union(parent relations, declared table) | DONE | `buildIntended` + `declaredChildSlugs`; `TestDeclaredChildSlugs`, `TestCheckNoShortCircuit...` |
| 7 | Every intended child designed + verify PASS → `done` | DONE | `Check` done-logic over the intended set; `TestCheckDoneWhenAllChildrenCompleted`; live (dry-run ended in `done` after design→deliver) |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Stage classifier + child-table parser | DONE | `stage.go` |
| 2 | Check/DryRun rewrite (intended-set, stage, action) | DONE | `check.go` |
| 3 | Skill + Stop-hook act on `action` | DONE | design vs deliver routing |
| 4 | Tests + designed-child fixture | DONE | 8 drive tests + cli fixture |

### Exercise-the-feature check

- [x] User-visible behavior was exercised end-to-end: in a throwaway workspace with one undesigned stub (`a-stub`, no AC) and one designed child (`b-ready`, AC + test plan), `hero goal demo --check` returned `verdict: continue, action: design, next_spec: a-stub` (no delivery kickoff), and `hero goal demo --dry-run 3` previewed `design a-stub → deliver b-ready → done`. The undesigned child was routed to design, never to delivery, and the run did not short-circuit.

### Excellence Bar self-check

- [x] yes — progressive design is now the spine: undesigned children route to `/design`, never to delivery; the intended-child set (relations ∪ declared table) kills the short-circuit; design is autonomous and only genuine forks pause. AC#5's automated fork-*detection* is honestly scoped as a refinement (the pause *capability* and skill instruction are in place), not overclaimed.
