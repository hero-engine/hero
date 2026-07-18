---
name: sprint-planning
description: Sprint commit, velocity reading, and cut-line decisions under the sprint preset — including what to do when plans drift.
metadata:
  audience: pm-delivery-lead
  purpose: process-guidance
---
## What I do

Provide the mechanics of sprint planning under the `sprint` preset — how to read velocity honestly, how to set a commit vs stretch, how to draw the cut-line on the story queue, and what to do when the plan drifts (which it will). Adopts the scrum/scrumban vocabulary because that's what teams running this preset already use.

## When to use me

- Authoring a sprint plan at the start of an iteration.
- Reviewing a team's velocity history when planning capacity.
- Triaging a queue of `ready` stories into a sprint commit.
- Diagnosing why the team consistently over- or under-commits.
- Replanning mid-sprint when scope changes (priority drop, blocker, surprise).

## Velocity is a distribution, not a number

The single most common sprint-planning failure is treating velocity as a point estimate. "Our velocity is 32" sounds precise; it isn't. Velocity is a noisy distribution shaped by team composition, story-typing quality, holidays, on-call load, surprise incidents, and the difficulty of the stories sampled in any given window.

The correct view:

```
Last 6 sprints:  28, 41, 19, 35, 30, 33
Median: 31.5
Min/Max: 19 / 41
```

Plan the **commit** against the median or slightly below. Reserve room for the variance — if the team has hit 19 in a recent sprint, committing to 35 because last sprint was a 35 is gambling.

If a capacity model shows a single velocity number with no distribution, push back. The distribution is the artifact; the headline number erases the signal.

## Commit vs stretch

A sprint has two queues:

- **Commit** — what the team is publicly promising to ship this sprint. Sized to the bottom half of the velocity distribution. Missing the commit is a real miss, surfaced in retro.
- **Stretch** — what the team will start if the commit completes early. Sized so the commit + stretch lands near the top of the velocity distribution. Missing stretch is normal; it isn't a promise.

This split exists because the commit is a contract with stakeholders and the stretch is a contract with the work. Without the split, every sprint is a stretch (and every miss is "we just didn't get to it"), which trains stakeholders to discount commitments.

The right shape:

```
Commit (29 pts): [story A 5pt] [story B 8pt] [story C 3pt] [story D 13pt]
Stretch (10 pts): [story E 5pt] [story F 5pt]
Total: 39 pts (top of recent range)
```

## The cut-line on the story queue

The story queue going into planning is the prioritized list of `ready` stories. The cut-line is where you stop committing.

Drawing the cut-line:

1. **Sort the queue by priority** (the prioritization frameworks output, with any explicit overrides).
2. **Walk down, summing points,** stopping at the median velocity for commit and at the top of the range for stretch.
3. **Promote dependencies upward.** If story #7 depends on story #15, either pull #15 up or drop #7 below the line.
4. **Sanity-check the commit's coherence.** A sprint of unrelated firefighting stories is usually worse than a sprint focused on a single epic or theme. If the commit is incoherent, ask whether re-ordering by theme would improve it without losing too much priority weight.
5. **Cut-line is non-negotiable for promotion.** Stories that didn't make the cut don't get "stretch-committed under the table." If they're not in commit or stretch, they're not in the sprint.

## How sprint preset interacts with story fields

The `story` spec type carries preset-conditional fields. Under sprint preset:

- `points: <int>` — the sized estimate the team places on the story. Set during refinement, not planning. Stories without points cannot be sprint-committed.
- `sprint: <string>` — which sprint the story is committed to. Populated at planning, cleared if carried over.

A story missing `points` at planning time is a signal that refinement hasn't happened. Do not size it in the planning meeting — that's where guess-points come from. Send it back to refinement.

A story with a hand-wave `points: 13` and no acceptance criteria is worse than one with no points at all, because it pretends to be ready when it isn't.

## When the team consistently over- or under-commits

### Over-commits (misses sprint > 50% of the time)

Likely causes:
- Planning against velocity max instead of median.
- Surprise work (incidents, urgent asks) eating committed capacity.
- Story-point inflation that masks complexity (everything is 5 or 8 with no 13s or 21s).
- Carry-over compounding — last sprint's incomplete work counts toward this sprint's load but doesn't appear in planning.

Remedy: drop commit to median minus one sigma. Carry-over enters the next sprint's commit explicitly, not as bonus capacity. Track interrupt load as a separate line item and subtract it from available capacity.

### Under-commits (finishes commit + stretch + extras every sprint)

Likely causes:
- Planning against a stale velocity (team has gotten faster).
- Points being inflated to create buffer.
- Stretch acting as the real commit; "commit" is sandbagging.

Remedy: raise commit toward median. Make the team explicit about why estimates are inflating. Surface the sandbagging in retro — chronic under-commitment hides capacity and erodes stakeholder trust.

### The right diagnostic

Look at the **hit rate on commit**, not the absolute throughput. A team that ships 30 points 90% of the time is more predictable than a team that ships 45 ± 25.

## The half-life of a sprint plan

Most sprint plans drift by day 3. The drift sources:

- A `committed` story turns out to be bigger than estimated.
- Discovery during implementation reveals a dependency that wasn't mapped.
- A production incident pulls capacity.
- A stakeholder request lands and the PM accepts it.

Plans that don't adapt to drift become fiction. The discipline:

- **Standup is a re-planning surface,** not just a status broadcast. When the team identifies a slip, the question "what comes out of commit?" is on the table.
- **Don't quietly stretch.** If the commit is in jeopardy, the team explicitly drops something — not silently absorbs the slip and misses everything by 10%.
- **Day-3 check-in.** Optional but useful: a 15-minute mid-sprint check where the commit's status is honestly reported. By day 3 the team usually knows whether the commit will hold.

The plan is a guess at Monday; by Wednesday the guess has aged. Treat replanning as normal, not as failure.

## Standup as a re-planning surface

The standup format (yesterday/today/blockers) is useful but incomplete. The PM should add:

- **Commit health check.** Are we still on track? If not, what's the proposal — drop a story, extend cycle, accept the miss?
- **New asks landed.** What got dropped on us since yesterday? Does it require a commit change?
- **Carry-over risk.** Which stories look unlikely to finish this sprint?

This isn't ceremony for ceremony's sake — it's the only forum where the plan can re-shape itself in time to matter.

## Coexistence with other delivery layers

The sprint preset is a delivery layer, not an exclusive operating model. It can coexist with:

- **Continuous discovery cadence** — discovery runs weekly regardless of sprint boundaries.
- **Roadmap horizon (now/next/later)** — sprint commits draw from `now` horizon items.
- **A milestone overlay** — multi-sprint milestones aggregate sprint output.

What sprint preset replaces:

- Ad-hoc "we'll get to it when we get to it" delivery — sprint forces explicit commits.
- Pure flow / WIP-limit kanban — sprint adds time-boxed commitments that flow doesn't.

What sprint preset does *not* fix:

- A bad roadmap. Sprint planning organizes execution; it doesn't decide what to execute. Bad inputs produce well-executed irrelevance.
- A team without refined stories. Sprint planning assumes a queue of `ready` stories; if refinement is broken, planning is a guessing exercise.

## Anti-patterns

- **Rolling-average velocity treated as ground truth.** Hides the variance. Always show the distribution.
- **Planning poker theater.** Estimating in a meeting under time pressure, with social anchoring to whoever speaks first. If you must use planning poker, do it on stories already refined, not as a way to refine.
- **Sprint goals that are "finish the stories."** A sprint goal is a *one-sentence outcome* the sprint serves, distinct from the story list. Without a goal, the sprint is a to-do batch.
- **Stretch as the real commit.** Sandbagging the commit so the team always "exceeds." Stakeholders learn to ignore the published commit.
- **Carry-over absorbed silently.** Incomplete stories from last sprint added to this sprint without subtracting from capacity. The team is now committed to 1.5x.
- **Sprint as a deadline forcing function.** "We have to finish X by end of sprint" pressure that produces brittle work. Sprints are for predictability, not for arbitrary deadlines.
- **No mid-sprint replanning.** Plan made Monday; reality drifts by Wednesday; the team misses the commit and is "surprised" Friday. The drift was visible Wednesday.
- **Unsized stories committed.** A story without `points` cannot be honestly committed. Send it back to refinement, don't guess-size in planning.
- **Sprint-as-methodology-cult.** Forcing sprint preset on work that's better served by flow or cycle. The preset should fit the team, not the other way around.

## Cross-references

- `cycle-planning` — the Shape Up alternative for teams running 6+2 instead of fixed sprints.
- `iteration-planning` — generic iteration shape for kanban / phased presets.
- `prioritization-frameworks` — produces the prioritized queue that sprint planning draws from.
- `story-writing-invest` — INVEST-shaped stories are what makes refinement (and therefore sizing) possible.
- `capacity-planning` — the cross-team / cross-sprint capacity view that sits above sprint planning.
- `story` spec type — `points` and `sprint` are the preset-conditional fields planning populates.
