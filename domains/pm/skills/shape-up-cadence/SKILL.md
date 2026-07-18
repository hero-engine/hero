---
name: shape-up-cadence
description: The operational rhythm of the cycle preset — the 6-week build + cooldown cadence, betting-table timing, and hill-chart update cadence. Distinct from cycle-planning (the planning mechanics); this is the cadence that repeats.
metadata:
  audience: cycle-planner, pm-delivery-lead
  purpose: process-guidance
---

## What I do

Carry the recurring **operational rhythm** of the Shape Up cycle preset — *when* things happen, not *how* to plan them. `cycle-planning` covers the planning mechanics (how a betting table runs, how appetite constrains scope, how a bet busts). This skill covers the cadence those mechanics repeat on: the build-plus-cooldown rhythm, when the betting table fires, and how often the hill chart is updated. `cycle-planner` loads this to time the next cycle and to drive the cycle-fit marker's timing; `pm-delivery-lead` loads it to keep the team's rhythm honest. Source: Singer, *Shape Up* (Basecamp).

## When to use me

- Setting up or defending the cadence of a team running the cycle preset.
- Timing the next betting table (and pushing back when someone wants to bet mid-cycle).
- Deciding how often the hill chart should move and reading it for cadence health.
- Defending cooldown when leadership wants to "skip it this cycle to catch up."
- Diagnosing a team whose "cycles" have quietly become continuous open-ended work.

## The 6-week build + cooldown rhythm

The Shape Up default is **6 weeks of build + 2 weeks of cooldown** as one repeating cycle. Common variations:

- **4 + 1** — smaller teams, faster product, tighter feedback. Common in early-stage startups.
- **5 + 1** — a slightly longer build with a single cooldown week.

The ratio and the regularity matter more than the exact numbers. The load-bearing property is that **bets are made at fixed intervals, not on demand.** A team running "rolling 6-week cycles that start whenever the last bet finishes" is not running Shape Up — it's running open-ended waterfall. The cooldown is what creates the synchronization point; without it there is no betting table and no cadence.

**Cooldown is non-negotiable.** Its purposes are housekeeping (bugs, small debt), exploration (ideas that might become pitches), pitch authoring for the next table, and genuine rest. What cooldown is *not* for: extending the previous cycle ("we're 80% done, let's just finish in cooldown"). That is the single most common Shape Up failure — do it twice and cycles become continuous and the methodology collapses. The rule: **the bet ends when the cycle ends.**

The tell that a team has lost the cadence is a cooldown that keeps shrinking — two weeks becomes "we'll take a couple days this time" until it's gone entirely. When that happens the betting table loses its slot, bets start getting made ad hoc, and what's left is open-ended waterfall wearing Shape Up's vocabulary. Defend the cooldown as the load-bearing part of the rhythm, not the skippable one.

## Betting-table timing

The betting table is a **fixed meeting near the end of cooldown**, once per cycle — not a continuous intake:

- **When it fires.** During cooldown, after the previous cycle's work has ended and before the next build starts. Pitches authored during cooldown are ready; the table walks them and produces the next cycle's bet list.
- **Once per cycle, not on demand.** Asks that land mid-cycle do **not** get bet mid-cycle. They wait for the next table. This is the discipline that protects a running cycle from reprioritization churn — the cycle is a promise that the team gets left alone to build.
- **The output is the bet list, recorded.** "Team A bets pitch X (big batch); Team B bets pitch Y + pitch Z (small batch)." Unbet pitches sit on the shelf with the reason recorded — visibility is what makes a bet a bet and not a wish.

The betting table's timing is what `cycle-planner` reads here; the *mechanics* of running it (who's in the room, what qualifies as a pitch, how appetite is set) live in `cycle-planning`.

### A worked cadence (6 + 2)

```
Week 1–6   build cycle N        (team left alone; hill updates weekly)
Week 7–8   cooldown             (pitches authored; betting table in wk 8)
Week 9–14  build cycle N+1      (bets from the wk-8 table)
Week 15–16 cooldown             (…)
```

A mid-cycle ask landing in week 4 does not jump the queue — it waits for the week-8 table. A bet that's "80% done" in week 6 ends anyway; it either ships in its shaped-down form or busts. The cadence is what makes both of those calls automatic rather than a negotiation every time.

## Hill-chart update cadence

The hill chart earns its value only on a rhythm — a snapshot is nearly worthless; the **trajectory** across check-ins is the signal:

- **At least twice per cycle**, and in practice at every standup or weekly check-in. A hill chart updated once at cycle-end tells you nothing about where the risk was.
- **The person doing the work places the dot**, answering "what do you still not know?" — not a manager counting hours.
- **Read for movement, not position.** A scope that climbed, crested, or stalled is the story. A dot stuck uphill across two check-ins is the loudest alarm the chart produces — investigate now, don't wait for cycle-end.
- **Movement is the data.** A hill chart whose dots never move means the team isn't engaging with it; fix the cadence or drop the chart.

This cadence is what drives the **cycle-fit marker's timing** on the Story Queue: the marker is only as fresh as the last hill update, so a stale hill chart means a stale fit signal. For the deeper reading — *why* a scope's position means unknowns-remaining rather than percent-done, and how to interrogate a stuck scope — see `hill-chart-reasoning`.

## How the three cadences interlock

```
[ cooldown ]                 [ 6-week build ]                 [ cooldown ]
     |                              |                              |
 betting table              hill updates ≥2×                 betting table
 (fires here)            (standup / weekly)                  (fires here)
```

The betting table fires in cooldown and sets the cycle. The hill chart updates on a standup/weekly rhythm through the build and reports where the risk lives. Cooldown protects the rest and authors the next cycle's pitches. Break any one — bet on demand, skip the hill updates, burn the cooldown — and the cadence stops being Shape Up.

## Anti-patterns

- **Ad-hoc cycle starts.** "Start the next cycle whenever the last bet finishes." No fixed cadence means no betting table means no Shape Up — just open-ended work.
- **Betting outside the betting table.** Asks landing mid-cycle that displace running bets. Defer to the next table; the cycle is protected.
- **Cooldown used to extend the previous cycle.** "80% done, we'll finish in cooldown." The most common failure — once or twice and cycles become continuous.
- **Hill charts updated as progress bars.** Dragging a dot to "90% because most hours are spent." Position is unknowns-remaining; see `hill-chart-reasoning`.
- **Hill charts never updated.** A single cycle-end snapshot loses the trajectory that is the entire signal.

## Cross-references

- `hill-chart-reasoning` — the deeper unknowns-remaining reading of the hill chart (position ≠ percent-done, reading a stuck scope). This skill supplies the *cadence*; that skill supplies the *interpretation*.
- `cycle-planning` — the planning mechanics of the cycle preset (betting-table run, appetite, bet-bust). This skill is the recurring cadence; that skill is the planning.
- `pitch-writing-shape-up` — how the pitches authored during cooldown are written for the next betting table.
- `capacity-planning` — how many bets the team's cells hold per cycle; the appetite budget the cadence bets against.
