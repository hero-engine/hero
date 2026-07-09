---
title: Delivery Closing Gate Terminal Contract
slug: delivery-closing-gate-terminal-contract
type: feature
status: completed
created: 2026-06-29
tags: [deliver, codex, audit, agent-reliability]
delivery_method: manual
completed_at: 2026-06-29T18:38:19Z
---
# Delivery Closing Gate Terminal Contract

## Goal

Stop delivery agents (observed acutely in Codex) from finishing the
implementation, *recognizing* the audit/verify gate is required, and then
**yielding to the user instead of running it** — leaving the spec stuck in
`planning`/`delivering` with the cold audit unrun.

## Background

A real Codex session in a downstream project ended with:

> "I did not mark the Hero spec formally complete/archive because the final
> delivery audit gate still needs to run."

The agent **understood** the audit was mandatory and correctly refused to flip
status without it — but it stopped and handed back rather than running the
gate. So this is not a comprehension gap; tightening the *explanation* of why
audits matter would not have helped. The failure is structural, in three places:

1. **No terminal-state contract in the always-loaded context.** In Codex the
   only persistent instruction surface is `AGENTS.md`. Its "Running Hero
   Workflows in Codex" block says "read the skill, follow the steps, don't
   skip" but never states what *done* means for a delivery. The enforcement
   language (`MUST run hero spec verify`, "do not skip this step in supervised
   mode") lives 250+ lines deep in `command-deliver/SKILL.md`. The agent reads
   the long skill once, then drifts off its tail.

2. **The gates are trailing steps after the satisfying part.** The cold audit
   (step 6) and `hero spec verify` sit at the very end of a ~290-line workflow,
   after "code written + tests green" — exactly where an agent feels the task
   is essentially complete and treats the rest as optional cleanup. The
   agent's phrase "*still needs to run*" is the tell of long-workflow drift:
   it reframed an in-workflow step as a future external gate.

3. **The persistence rule is scoped to autopilot only.** "Do not yield between
   phases unless a true blocker fires" appears only under `### Autopilot mode`.
   The default mode is **supervised**, defined as "Pause at handoffs, surface
   decisions" — which an agent can read as license to stop before the audit and
   hand back. Nothing tells supervised mode that the closing gates are not a
   handoff.

`/diagnose` was checked and does **not** share this shape: its done-state is "a
fix spec written to disk," not a delivery with an audit gate. Scope is limited
to the deliver/audit path.

## Design

Three surgical edits, all to canonical engine sources (the downstream copies in
`.agents/` / `AGENTS.md` are generated and would be overwritten otherwise):

1. **Terminal-state contract in the always-loaded Codex block.** Add a "not
   finished until the closing gate runs" paragraph to
   `renderCodexWorkflowSection()` in `internal/install/agents_md.go`, so it
   lands in every Codex project's `AGENTS.md` managed region — the one surface
   guaranteed to stay in context. It names the gate (`hero spec verify`),
   states the audit must run first, and says explicitly: if you are about to
   say "the audit still needs to run," run it now instead.

2. **Hoist a "Definition of done" to the top of `deliver.md`.** Add a short
   `## Definition of done` callout immediately after the pre-flight intro and
   *before* `## Delivery modes`, so the terminal contract is seen before the
   long body rather than 250 lines in. It restates: not done until verify
   passes; closing gates run in the same turn as implementation; holds in every
   mode.

3. **Broaden the persistence rule out of autopilot-only.** Annotate the
   `--supervised` row in the modes table to state the closing gates (audit →
   verify) are NOT handoffs and must run before yielding, and have the
   Definition-of-done callout assert the persistence rule applies in all modes,
   not just autopilot.

No behavior change to the gates themselves — the audit and `hero spec verify`
already exist and already block `completed`. This change only makes the agent
actually reach them instead of stopping one step short.

## Changes

- `internal/install/agents_md.go` — `renderCodexWorkflowSection()`: append the
  terminal-state contract paragraph after the existing 3-step "read and follow"
  list.
- `internal/install/harness_smoke_test.go` — extend the AGENTS.md content
  assertion to cover the new contract text so the generated block can't
  silently lose it.
- `domains/engineering/commands/deliver.md` — add `## Definition of done`
  callout near the top; annotate the `--supervised` modes-table row with the
  closing-gates-are-not-handoffs clause.
- `internal/install/agents_md.go` — `generateEngineeringAgentsMdBody()`: add a
  "Finish the closing gate before yielding" item to the shared **Key Workflow**
  section so the always-loaded contract lands in **both CLAUDE.md and every
  target's AGENTS.md** (not just Codex). The Codex-only deeper version stays.
- `domains/engineering/AGENTS.md` — regenerated from the Go fallback
  (`HERO_REGEN_PACK_AGENTS=1`) to stay byte-equal per
  `TestEngineeringPackBodyMatchesGoFallback`.
- `internal/install/harness_smoke_test.go` — assert the shared contract renders
  into both `CLAUDE.md` and `AGENTS.md` on the Claude target.

## Kickoff

Pick up at: make the three edits in the Changes section. Start with
`renderCodexWorkflowSection()` in `internal/install/agents_md.go` (the
always-loaded contract), then `domains/engineering/commands/deliver.md`
(Definition-of-done callout + supervised-row annotation), then extend
`internal/install/harness_smoke_test.go`. Validate with
`go test ./internal/install/...` and `go build ./...`.

## Completion Ledger

| # | Item | Status | Evidence |
|---|------|--------|----------|
| AC1 | Generated Codex AGENTS.md contains terminal-state contract (not done until `hero spec verify` passes; audit first; run the gate rather than yield) | DONE | `renderCodexWorkflowSection()` in `internal/install/agents_md.go` appends the "A Hero workflow is not finished until its closing gate runs." paragraph; smoke test asserts it renders |
| AC2 | `deliver.md` has `## Definition of done` before `## Delivery modes` (same-turn, every mode incl. supervised) | DONE | `domains/engineering/commands/deliver.md` — new section at line 20, precedes `## Delivery modes` at line 40 (verified by audit) |
| AC3 | `--supervised` row states closing gates are not handoffs, run before yielding | DONE | Supervised row annotated in `deliver.md` modes table |
| AC4 | `harness_smoke_test.go` asserts new contract text; install tests pass | DONE | Two `mustContain` assertions added; `go test ./internal/install/...` → ok |
| AC5 | `go build ./...` clean | DONE | `go build ./...` exit 0; `go test ./...` → 85 packages ok |
| AC6 | Always-loaded contract reaches **all targets** via the shared body — CLAUDE.md and every target's AGENTS.md, not just Codex | DONE | `generateEngineeringAgentsMdBody()` Key Workflow item 5; pack `domains/engineering/AGENTS.md` regenerated; smoke test asserts on both `CLAUDE.md` and `AGENTS.md` |
| C1 | `agents_md.go` `renderCodexWorkflowSection` edit | DONE | see AC1 |
| C2 | `harness_smoke_test.go` assertion edit | DONE | see AC4 |
| C3 | `deliver.md` Definition-of-done + supervised-row edits | DONE | see AC2/AC3 |
| C4 | `agents_md.go` shared-body Key Workflow item + pack regen | DONE | see AC6; `TestEngineeringPackBodyMatchesGoFallback` passes |
| C5 | `harness_smoke_test.go` CLAUDE.md + AGENTS.md shared-contract assertions | DONE | see AC6 |

## Acceptance Criteria

- The generated Codex `AGENTS.md` managed region contains an explicit
  terminal-state contract: a delivery is not finished until `hero spec verify`
  passes (audit first), and the agent must run the gate rather than yield with
  it unrun.
- `deliver.md` carries a `## Definition of done` section positioned before
  `## Delivery modes` that states the closing gates run in the same turn and
  hold in every mode (including supervised).
- The `--supervised` row of the modes table states the closing gates are not
  handoffs and must run before yielding.
- `harness_smoke_test.go` asserts on the new contract text; `go test ./internal/install/...` passes.
- The always-loaded contract reaches **all six install targets** (claude, codex,
  opencode, cursor, copilot, generic) via the shared body — emitted
  unconditionally by `Render` for every target, with the Codex-only deeper
  version appended on top. The Claude smoke test asserts the shared line on both
  CLAUDE.md and AGENTS.md; the other five targets' reach is guaranteed by the
  shared-body code path plus `TestEngineeringPackBodyMatchesGoFallback` keeping
  the Go fallback and pack `domains/engineering/AGENTS.md` byte-equal.
- `go build ./...` is clean.
