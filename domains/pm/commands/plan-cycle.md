---
description: Plan the next Shape Up cycle — the shape-up entry point into the one preset-adaptive cycle-planner (betting table + appetite + cooldown cadence). Recommends a commit, never auto-commits.
---
Route to the `cycle-planner` agent. This is the **shape-up cycle** entry point into the single preset-adaptive planner: it plans the next cycle using the betting-table + appetite + cooldown model, with the cadence read from `shape-up-cadence` and the capacity read delegated to `capacity-planner`.

`cycle-planner` is **one agent** across sprint / cycle / iteration, not three. This command is a hint that you want the cycle model; the agent still reads the active preset via `pm-preset-detection`, and **the preset is authoritative** — if `hero.json` `pm.presets.delivery` is not `cycle`, the agent plans the active preset's model and says so.

## What lands

A **recommended** commit for the next cycle — the bet list with per-Story cycle-fit, what's cut and why, and the preset-conditional Story fields proposed (`appetite` / `cycle` / `hill_position`). It is never auto-committed: the planner recommends, the team decides (decision gate — `pm-agent-doctrine`).

Request: $ARGUMENTS
