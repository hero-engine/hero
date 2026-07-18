---
description: Plan the next scrum sprint — the sprint entry point into the one preset-adaptive cycle-planner (velocity + commit/stretch). Recommends a commit, never auto-commits.
---
Route to the `cycle-planner` agent. This is the **scrum sprint** entry point into the single preset-adaptive planner: it plans the next sprint using the velocity-distribution + commit/stretch model, with the cut-line and capacity read delegated to `capacity-planner`.

`cycle-planner` is **one agent** across sprint / cycle / iteration, not three. This command is a hint that you want the sprint model; the agent still reads the active preset via `pm-preset-detection`, and **the preset is authoritative** — if `hero.json` `pm.presets.delivery` is not `sprint`, the agent plans the active preset's model and says so.

## What lands

A **recommended** commit for the next sprint — the commit/stretch split with per-Story cycle-fit, what falls below the cut-line and why, and the preset-conditional Story fields proposed (`points` / `sprint`). It is never auto-committed: the planner recommends, the team decides (decision gate — `pm-agent-doctrine`).

Request: $ARGUMENTS
