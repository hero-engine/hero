---
name: capacity-planning
description: Per-preset capacity math — velocity distribution (sprint), appetite budget (cycle), WIP + aging (kanban), release scope (phased) — honest velocity, WIP limits, and how the Story Queue cut-line is drawn.
metadata:
  audience: capacity-planner, cycle-planner, prioritization-strategist, portfolio-curator
  purpose: process-guidance
---

## What I do

Give the mechanics of answering one question honestly across every delivery preset: **how much can this team actually take on?** Capacity is read differently under each preset — velocity under sprint, appetite under cycle, WIP + aging under kanban, release scope under phased — but the discipline is the same everywhere: read the honest signal, never a flattering point estimate, and draw the cut-line where the committed queue crosses the line. This skill is the home for the cut-line and the per-preset math; `capacity-planner` is the agent that runs it, and `cycle-planner` draws on it for the capacity read behind a commit.

## When to use me

- Placing the Story Queue cut-line under any preset.
- Reading a team's velocity history to size a sprint commit.
- Turning a Shape Up appetite into a capacity budget for a cycle.
- Setting or interpreting a WIP limit under kanban, and reading card aging.
- Diagnosing chronic over- or under-commitment.
- Feeding an effort-vs-capacity input into a prioritization pass.

## Per-preset capacity math

### Sprint — velocity is a distribution, not a number

The single most common capacity failure is treating velocity as a point estimate. "Our velocity is 32" sounds precise; it isn't. Velocity is a noisy distribution shaped by team composition, story-typing quality, holidays, on-call load, and surprise incidents.

```
Last 6 sprints:  28, 41, 19, 35, 30, 33
Median: 31.5   Min/Max: 19 / 41
```

Plan the **commit** against the median or slightly below; reserve room for the variance. Committing to 35 because last sprint was a 35, when a recent sprint hit 19, is gambling. Distinguish a **commit** (the bottom-half promise stakeholders can bank on) from a **forecast** (the median-to-top range you *might* reach). A commit is a contract; a forecast is a probability. Collapsing them is how teams train stakeholders to discount every commitment.

### Cycle — appetite is a budget, not an estimate

Under the shape-up preset you don't estimate the work, you constrain it. The **appetite** — small-batch (≈ 2 weeks) or big-batch (≈ 6 weeks) — is the budget the PM sets before the betting table. Capacity math here is counting how many small-batch and big-batch bets the team's cells can hold in a cycle, not summing hour estimates. Engineering's job is to ship the most valuable version *within* the appetite by cutting scope — not to validate whether the appetite is "enough." Appetite treated as an estimate to defend is the anti-pattern; see `cycle-planning`.

### Kanban — WIP limits and aging

Continuous flow has no timebox, so capacity is expressed as a **WIP limit** — the most items allowed in flight at once — plus the **aging** of what's already there (cycle-time distribution, oldest card first). A team at its WIP limit is at capacity by definition; the queue behind it waits. Aging is the early-warning signal: a card far past the median cycle time is stuck, and stuck work consumes capacity without producing throughput. Read aging oldest-first before you promote anything new.

A worked pass (kanban): WIP limit 4, three cards in flight (ages 2d, 3d, 11d against a 4-day median cycle time). Capacity headroom is one slot — but the 11-day card is the real signal: it's nearly 3× the median and stuck. The honest read is "one slot free, but finish the aged card before pulling — the free slot is masking a bottleneck." Pulling a fourth card here just widens the flow problem instead of exposing it.

### Phased — release scope and gate capacity

Under the phased preset, capacity is the scope a release can hold and the throughput of the current phase gate (discovery / design / build / launch). A build phase has a finite intake; the gate into it is an explicit entry criterion, not a date. Capacity here means: does the committed release scope fit the phase's throughput before the gate closes? See `iteration-planning` for phase-gate semantics.

The trap is a downstream phase that silently becomes the bottleneck — a build queue sized to the team's build throughput while the launch gate can only absorb half that per release. Read capacity gate-by-gate, not just at the phase the work is entering; the cut-line for a phased release sits at the *tightest* gate, not the widest.

## Honest velocity

Whatever the preset, the honest signal beats the flattering one:

- **Show the distribution, not a headline number.** The median plus the min/max carries the signal a single average erases. If a capacity model shows one velocity number with no spread, push back — the distribution is the artifact.
- **Show the uncertainty band.** Plan the commit against the low-to-median band; let the stretch/forecast reach toward the top.
- **Separate a commit from a forecast.** Missing a commit is a real miss surfaced in retro; missing a forecast is normal. Never publish the optimistic forecast *as* the commit.

## WIP limits — a tool, not a wall

A WIP limit is a lever for exposing flow problems, not a barrier that hides them. When work piles up against a limit, that's the signal to swarm and finish, not to raise the limit. A breach is information — it means something upstream is producing faster than the team can finish. Raising the limit to "stop the breach" just re-hides the bottleneck. See `iteration-planning` for the operational detail.

## The cut-line on the Story Queue

The cut-line is where the committed queue crosses capacity. Drawing it:

1. **Sort the queue by priority** (the prioritization ranking, with any explicit overrides).
2. **Walk down, summing against capacity** — median velocity (sprint), appetite budget (cycle), the WIP limit (kanban), or remaining release/phase scope (phased).
3. **Promote dependencies upward.** If a story above the line depends on one below it, pull the dependency up — or drop the dependent below.
4. **Stop at the capacity boundary.** Everything above is proposed in; everything below is proposed out, each with a one-line reason.
5. **The cut-line is non-negotiable for promotion.** Stories below the line don't get "stretch-committed under the table." If they're not above the line, they're not in the iteration.

A worked pass (sprint): median velocity 31; queue sorted A(5) B(8) C(3) D(13) E(5) F(5). Summing: A+B+C+D = 29 (in). Adding E → 34 crosses the median — E and F fall below the line unless a dependency pulls one up. The cut-line sits between D and E; the commit is 29, with E/F as forecast, not promise. The cut-line is a **proposal the team accepts**, never a commit written on their behalf (decision gate — `pm-agent-doctrine`).

## Anti-patterns

- **Sandbag-then-overcommit.** Padding estimates to build buffer, then committing past the real line anyway. The audit trail is the loyalty, not a flattering plan.
- **Last sprint's velocity as next sprint's commit, no bands.** One data point is not a distribution; a recent low sprint is exactly the variance you must plan around.
- **A headline velocity number that erases the distribution.** The average hides the spread that carries the signal.
- **WIP as a wall.** Raising the limit to stop a breach re-hides the bottleneck the breach exposed.
- **Auto-committing the cut-line.** Writing the commit into Story fields. Capacity work recommends; the team decides.
- **Appetite-as-estimate.** Setting the cycle appetite to whatever engineering says it'll take, instead of the budget the work is cut to fit.

## Cross-references

- `sprint-planning` — the sprint model this skill supplies the velocity-distribution and cut-line math for.
- `cycle-planning` — appetite as constraint and the betting table; capacity under the cycle preset counts bets, not hours.
- `iteration-planning` — WIP limits and phase-gate capacity under kanban / phased presets.
- `pm-preset-detection` — read `hero.json` `pm.presets` to pick which capacity lens applies before summing a single point.
- `feature` and `initiative` spec types — `points`/`sprint`, `appetite`/`cycle`, and `phase` are the preset-conditional fields a capacity read reasons over.
