---
description: Plan the next generic iteration — the kanban/phased entry point into the one preset-adaptive cycle-planner (WIP + rolling commit + phase gates). Recommends a commit, never auto-commits.
---
Route to the `cycle-planner` agent. This is the **generic kanban/phased** entry point into the single preset-adaptive planner: it plans the next iteration using the rolling-pull + WIP model (kanban) or the phase-gate model (phased), from `iteration-planning`, with the capacity read delegated to `capacity-planner`.

`cycle-planner` is **one agent** across sprint / cycle / iteration, not three. This command is a hint that you want the generic iteration model; the agent still reads the active preset via `pm-preset-detection`, and **the preset is authoritative** — if `hero.json` `pm.presets.delivery` names a different preset, the agent plans the active preset's model and says so.

## What lands

A **recommended** plan for the next iteration — the pulled/committed Story set against the WIP limit or the recommended phase-gate crossings, what stays below the line and why, and the preset-conditional Story field proposed (`phase` under phased). It is never auto-committed: the planner recommends, the team decides (decision gate — `pm-agent-doctrine`).

Request: $ARGUMENTS
