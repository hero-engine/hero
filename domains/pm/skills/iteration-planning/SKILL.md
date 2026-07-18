---
name: iteration-planning
description: The generic iteration shape for kanban and phased presets — WIP limits as a tool not a wall, rolling commitment, and phase-gate semantics — explicitly distinct from fixed-sprint and Shape-Up-cycle planning.
metadata:
  audience: cycle-planner, capacity-planner
  purpose: process-guidance
---

## What I do

Provide the planning mechanics for the delivery presets that are **neither a fixed sprint nor a Shape Up cycle**: `kanban` (continuous flow) and `phased` (discovery / design / build / launch). These presets have no timebox to plan against — flow is continuous, or work moves through gates — so the planning discipline is different. This skill is where `cycle-planner` gets its model when the active preset is kanban or phased, and where `capacity-planner` gets the WIP and phase-gate reading. For the sprint model see `sprint-planning`; for the cycle model see `cycle-planning`.

## When to use me

- Planning the next iteration under the `continuous` (kanban) or `phased` delivery preset.
- Setting or interpreting a WIP limit and reasoning about what a breach means.
- Deciding how the Story Queue behaves when there's no fixed sprint boundary.
- Defining or reviewing a phase gate for the phased preset.
- Diagnosing why "we do kanban" has quietly become "we do mini-waterfall with dates."
- Explaining to a sprint-trained team why there's no commit meeting and no velocity number.

## Why generic iteration is its own model

Sprint planning assumes a timebox and a velocity distribution. Cycle planning assumes an appetite and a betting table. Neither fits flow-based or gated work:

- **Kanban has no timebox.** There is nothing to "commit for the next two weeks"; work is pulled continuously as capacity frees up. Planning is about the *pull policy* and the WIP limit, not a batch commitment.
- **Phased has gates, not sprints.** Work moves discovery → design → build → launch through explicit entry/exit criteria. Planning is about what's ready to cross the next gate, not what fits in a box.

Forcing either into a sprint or cycle model produces broken artifacts — velocity math on unestimated flow work, or a betting table for a team that pulls continuously.

## WIP limits as a tool, not a wall

A WIP limit is the number of items allowed in flight at a given stage at once. Its purpose is to **expose flow problems**, not to block work:

- **What a limit is for.** Limiting WIP forces the team to *finish* before starting. It surfaces bottlenecks (work piling up in front of a stage), shortens cycle time, and keeps context-switching down.
- **How to set one.** Start near the team's natural in-flight count, then tighten until the queue in front of the slowest stage becomes visible. Too high and it's decorative; too low and the team starves.
- **What a breach signals.** Hitting the limit is information: something upstream is producing faster than this stage can finish. The response is to swarm and finish, or fix the bottleneck — **not** to raise the limit. Raising the limit to stop a breach re-hides the problem the breach revealed. See `capacity-planning` for how the WIP limit reads as capacity.

## Rolling commitment

Without a fixed boundary, the Story Queue doesn't get a batch commit — it gets a **pull policy** and a rolling commitment:

- **Pull, don't push.** When a work slot frees up, the top-priority `ready` story is pulled in. There's no Monday-morning "here's the sprint"; the queue is always live and the next item is always the current top of the ranked queue.
- **The commitment is the next item, not the batch.** The team commits to *finishing what's pulled*, one item deep, rather than promising a fortnight of scope. This is what makes flow predictable — cycle time per item, not velocity per sprint.
- **How the queue behaves.** The Story Queue stays sorted by priority; the cut-line is soft and continuous — it's simply "everything above the current WIP capacity is next, in order." Re-ranking is cheap because nothing was batch-committed.

"Rolling commitment" is not "no commitment." The team still finishes what it pulls; it just doesn't pretend to know two weeks of scope in advance.

### A worked pass (kanban)

WIP limit 3, two cards in flight, one slot free. Ranked `ready` queue: X, Y, Z. Planning the "next iteration" here is a single decision: pull X (the top of the ranked queue) into the free slot, commit to *finishing* it, and leave Y and Z in the queue untouched. There is no batch to size and no velocity to sum — the plan is the pull policy plus the WIP limit. When the next slot frees, re-read the ranked queue (it may have re-sorted) and pull its current top. The cut-line is simply "everything above current WIP capacity, in priority order," and it moves continuously as work finishes.

## Phase-gate semantics (phased preset)

Under the phased preset, a **gate is an explicit entry/exit criterion, not a date**:

- Each phase (discovery / design / build / launch / post-launch) has a defined *done* — the criteria a story must meet to cross into the next phase. "Design gate: the flows are reviewed and the AC are written" is a gate; "design ends Friday" is a deadline masquerading as one.
- Planning the iteration means asking, per story: **which gate is it at, and what does it need to cross the next one?** The `phase` Story field carries the current phase; planning populates it and names the exit criterion.
- A gate that has become a date is the phased anti-pattern — it turns iterative discovery into a waterfall with a countdown. Gates are about readiness, not the calendar.

### A worked pass (phased)

Queue: Story P at `discovery` (needs 3 more interviews to exit), Story Q at `design` (flows reviewed — ready to cross into `build`), Story R at `build` (in progress). Planning the next iteration: Q crosses its gate and its `phase` moves to `build`; P stays at `discovery` with the interview criterion named as its blocker; R continues. The plan is a set of gate-crossings recommended for the team to accept, not a dated schedule — populate `phase` as a proposal, never auto-flip it (decision gate — `pm-agent-doctrine`).

## How it populates Story fields

Iteration planning under these presets touches a narrow set of preset-conditional fields, and only the ones the active preset requires (per `pm-preset-detection`):

- **Kanban (`continuous`)** — no batch commit fields. The dashboard computes `wip_age` from the in-flight timestamp; planning does not author it. There is deliberately no `points` and no `sprint` — estimating flow work produces meaningless numbers.
- **Phased** — the `phase` field carries the current gate (`discovery / design / build / launch / post-launch`), and planning proposes the crossing that moves it forward. The `release` field (org-state, tracker-authoritative) names the release the story is committed to.

The rule is the same as everywhere in the pack: propose the field value with its reason; the team accepts it. A planner that silently flips `phase` from `design` to `build` has skipped the readiness judgment the gate exists to force.

## When neither model fits

If the work is genuinely time-boxed and estimated, you're running sprints — use `sprint-planning`. If it's shaped to an appetite and bet at a table, you're running Shape Up — use `cycle-planning` and `shape-up-cadence`. This skill is for the middle ground: continuous pull, or gated phases, where the planning question is "what's ready to move next?" rather than "what fits in the box?"

## Anti-patterns

- **WIP limit ignored under pressure.** "We're slammed, just start it" — raising or ignoring the limit exactly when it's telling you the most. The breach is the signal; honor it.
- **Phased treated as mini-waterfall with date-gates.** Turning readiness criteria into calendar deadlines. A gate is what the work must *be*, not *when* it's due.
- **"Rolling commit" as an excuse for no commitment.** Pulling continuously but never finishing, so nothing is ever actually done. Rolling commitment still commits to finishing the pulled item.
- **Velocity math on flow work.** Estimating points and summing a "sprint velocity" for a team that pulls continuously — the numbers are meaningless without a timebox.
- **Auto-committing the plan.** Writing phase or pull decisions into Story fields as settled. Iteration planning recommends; the team decides.

## Cross-references

- `sprint-planning` — the fixed-sprint model; iteration planning is explicitly *not* this.
- `cycle-planning` — the Shape Up cycle model; iteration planning is explicitly *not* this either.
- `capacity-planning` — WIP limits and phase-gate throughput read as capacity; where the cut-line is drawn.
- `pm-preset-detection` — read `hero.json` `pm.presets` to confirm the active preset is kanban or phased before applying this model.
