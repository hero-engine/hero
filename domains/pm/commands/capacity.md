---
description: Reconcile committed work against team capacity under the active preset and place the Story Queue velocity cut-line — a proposal, never an auto-commit.
---
Route to the `capacity-planner` agent, loading `capacity-planning` (plus `sprint-planning` / `cycle-planning` for the per-preset lens and `pm-preset-detection` to select it). The planner reads the honest capacity signal for the active preset — velocity **distribution** (sprint), appetite budget (cycle), WIP + aging (kanban), release scope (phased) — walks the prioritized Story Queue, and draws the **velocity cut-line**: what fits above the line, what falls below, and why.

## Required argument

A cycle/sprint context or the Story Queue to reconcile. Without one, ask which iteration or queue to read capacity against — don't infer from session context. A cut-line belongs to a specific queue and a specific capacity window.

## What lands

- The active preset detected and the capacity lens it selects.
- The capacity signal shown with its source (the distribution / appetite / WIP / release scope, never a bare headline number).
- The **cut-line** placement — above-line vs below-line stories, each with a one-line reason, plus any dependency promotions.
- Any **over-capacity warning** naming the specific overcommit and what would have to come out.

This is a **proposal the team accepts**, not an auto-commit: the planner recommends the cut-line; it never writes the commit into Story fields on its own (decision gate — `pm-agent-doctrine`).

Request: $ARGUMENTS
