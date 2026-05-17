---
name: cycle-planning
description: Shape Up cycle planning — six-week build + two-week cooldown rhythm, betting table mechanics, appetite as constraint, hill chart as inflight signal.
compatibility: opencode
metadata:
  audience: cycle-planner, pitch-author, pm-delivery-lead
  purpose: process-guidance
---
## What I do

Provide the mechanics of Shape Up cycle planning under the `cycle` preset — the cycle rhythm, what a betting table is and how it runs, why appetite is the constraint instead of estimates, how the hill chart works as an inflight signal, what cooldown is (and isn't) for, and what to do when a bet busts. Source: Singer, *Shape Up* (Basecamp).

## When to use me

- Authoring a pitch and feeding it into the next betting table.
- Running or attending a betting table.
- Replanning when a bet appears to be busting mid-cycle.
- Setting up the cycle preset on a team adopting Shape Up.
- Defending cooldown when leadership wants to "skip it this cycle to catch up."
- Reading a hill chart and asking the right question about a stuck dot.

## The cycle rhythm

Shape Up's default is **6 weeks of build + 2 weeks of cooldown** as one cycle. Variations:

- **4 + 1** — smaller teams, faster product, tighter feedback. Common in early-stage startups.
- **5 + 1** — slightly longer build with a single cooldown week.
- **8 + 2** — larger teams or surfaces where 6 weeks routinely runs short. Rare; consider whether the work needs decomposing instead.

The ratio and the rhythm matter more than the exact numbers. The cycle is the cadence — bets get made at fixed intervals, not on demand. This is the load-bearing constraint of the methodology.

A team that runs "rolling 6-week cycles starting whenever the last bet finishes" is not running Shape Up — they're running open-ended waterfall. The discipline of the cooldown is what creates the betting table; without cooldown there's no synchronization point.

## Appetite is the constraint, not estimates

The single biggest mental shift from sprint/scrum to Shape Up: you don't estimate the work, you constrain it.

- **Sprint thinking:** "How long will this take? 3 weeks? OK, commit to it."
- **Cycle thinking:** "How much time is this worth? 2 weeks? Then the team has 2 weeks. Whatever doesn't fit gets cut."

Appetite is a number set by the PM (with stakeholder input) before the betting table. The two appetites:

- **Small batch** — about 2 weeks. Fits multiple bets per cycle for one team. Smaller scope, tighter problem.
- **Big batch** — full 6 weeks. One bet per team per cycle. Substantial scope.

The pitch declares the appetite up front. Engineering's job during the bet is to deliver the most valuable version of the bet **within** the appetite, cutting scope as needed. Engineering's job is *not* to estimate whether it fits — the appetite is a constraint, not an estimate to be validated.

When engineering says "this will take 8 weeks," the right response is rarely "let's bet 8 weeks." It's "what's the 6-week version, what's the 4-week version, what would we cut?" If no smaller version exists, the bet doesn't fit the cycle — defer it or reshape it.

## Betting table mechanics

The betting table is a fixed meeting near the end of cooldown. It produces the bet list for the next cycle.

### Who's at the table

- **Founders / CEO** in small companies; **senior leadership / product head** at larger ones.
- **Tech lead / head of engineering.**
- **Head of design.**
- **PM lead.**

Not at the table: individual contributors. The team that builds the bet isn't in the room — they get briefed after the bet is placed. This is intentional; it prevents the room from becoming a negotiation about who likes which idea.

### What gets bet

- **Pitches.** A pitch is a written artifact (problem, appetite, solution sketch, rabbit holes, no-gos). Authored by `pitch-author` (or `prd-author` in pitch shape).
- **Not ideas.** Ideas without pitches don't enter the betting table. The discipline of writing the pitch is the qualifying gate.
- **Not committed work.** Last cycle's bets that didn't finish don't automatically continue. They have to be re-pitched and re-bet.

### What stays unbet

- Pitches that don't make the cut this cycle. They sit on the shelf. They may be re-pitched next cycle, or they may quietly age out — both outcomes are valid.
- There is no backlog in the traditional sense. Pitches accumulate; only the ones bet on become work.

### The decision

The table walks through pitches, discusses tradeoffs, and produces the list. Calibrate to the team capacity (how many teams, how many small-batch vs big-batch bets fit). The output is short: "Team A bets on pitch X (big batch). Team B bets on pitch Y (small batch) plus pitch Z (small batch)."

The decision is recorded. The unbet pitches and the reasons (or lack of reason) are also recorded — visibility is what makes the bet a bet and not a wish.

## The hill chart as inflight signal

Shape Up uses a hill chart as the primary status visualization during a cycle. The hill has two slopes:

```
         apex
          /\
   uphill /  \ downhill
         /    \
     figuring  executing
       out      it
```

- **Uphill** — you're working out *what* the solution is and how it'll fit. Unknowns are being resolved.
- **Apex (top)** — the unknowns are resolved. You know exactly what to build, and you know it fits the appetite.
- **Downhill** — execution. Knowns being turned into shipped code.

Each scope (a sub-piece of the bet) is a dot on the hill. Movement is the data. The signal:

- **Dots moving up** — discovery is producing answers.
- **Dots at the apex moving downhill** — execution is progressing.
- **A dot stuck on uphill for >1 week** — something is wrong with the scope. Unknowns aren't resolving. This is the most important signal.
- **A dot that slides backward** — discovery surfaced something that invalidates earlier assumptions. Rare but informative.

The hill chart is **not a progress percentage**. A dot at apex isn't "50% done" — it's "halfway through; we don't yet know whether the downhill is steep or shallow." Reading it as percentage breaks the methodology's most useful signal.

The `hill_position` field on a `story` (or scope, depending on how the team models them) carries the hill chart state in Hero. Values: `unknown / uphill / top / downhill / done`.

## Cooldown — what it's for, what it isn't

Cooldown is 2 weeks (or 1, in smaller cycles) of no-bet time after each cycle. Its purposes:

- **Housekeeping.** Bugs, small improvements, technical debt the team didn't get to.
- **Exploration.** Trying ideas that might become pitches.
- **Pitch authoring and review.** Pitches for the next betting table are written and refined during cooldown.
- **Rest.** Genuine recovery from cycle pressure.

What cooldown is **not** for:

- **Extending the previous cycle.** "We're 80% done, let's just use cooldown to finish." This is the single most common Shape Up failure mode. Once you do it twice, cycles become continuous and the methodology collapses.
- **Starting next cycle's bets early.** No. The betting table hasn't happened yet.
- **Cross-team coordination work the cycle didn't have room for.** That's a sign the cycle scope was wrong.

The rule: **the bet ends when the cycle ends.** If the bet didn't finish, it either ships in its incomplete form (because the team shaped what to cut throughout) or the bet busts. Cooldown is not a soft extension.

## How cycle preset interacts with spec fields

Under cycle preset:

- `initiative.appetite` — `small` or `big`. Set when the bet is shaped.
- `story.cycle` — which cycle the story is part of. Populated when the cycle starts.
- `story.hill_position` — `unknown / uphill / top / downhill / done`. Updated by the team during the cycle, typically at standup or weekly check-ins.

Together these fields are what the cycle dashboard reads to render the betting table and the hill chart. An initiative without `appetite` can't be bet on; a story without `hill_position` can't appear on the hill chart.

## What to do when a bet busts

A bet busts when the team realizes mid-cycle that the work cannot ship in the remaining time, even with aggressive scope cuts. This is rare with good shaping but it happens.

The Shape Up answer: **the bet ends, the work stops, the cycle continues without it.** No automatic extension into the next cycle. If the bet is still worth pursuing, it must be re-shaped and re-pitched for a future betting table.

This is harsh but load-bearing. The threat of a busted bet is what disciplines pitch authors to shape work that fits the appetite. Without the threat, every pitch creeps and every cycle becomes an extension of the last.

Recovery:

1. **Declare the bust.** Surface it; don't quietly keep grinding.
2. **Capture what was learned.** What didn't we know that would have made us shape it differently? This feeds the next pitch.
3. **Free the team.** The remaining cycle time goes to cooldown-style work (if late in the cycle) or to a small-batch bet (if early, and one is shelf-ready and pre-shaped).
4. **Don't punish.** A busted bet is usually a shaping failure, not an execution failure. The lesson goes to the people who shaped, not the people who built.

The frequency of busts is a diagnostic. If bets bust often, shaping is too optimistic. If bets never bust, appetites are probably too generous (or the team is silently working past cycle end).

## Cycle preset vs other layers

Cycle preset is one of three delivery layers (with sprint and continuous flow). It is the right choice when:

- The team wants substantial uninterrupted build time.
- Work can be meaningfully scoped to 2 or 6 weeks.
- Leadership can defer most input to the betting-table cadence rather than reprioritizing weekly.

It's the wrong choice when:

- Work routinely needs <1 week turnaround (use continuous flow).
- Leadership cannot leave teams alone for 6 weeks at a time (the methodology collapses).
- The team needs to coordinate tightly with another team running sprints (the rhythms don't sync).

## Anti-patterns

- **Estimating inside cycles.** "How long will this take?" is a sprint question. In cycles, the answer is "the cycle." Scope flexes, time doesn't.
- **Skipping cooldown.** Most common failure. Once or twice and the methodology is dead.
- **Rolling cycles into each other without a betting table.** No cadence = no Shape Up. Just open-ended work.
- **Betting outside the betting table.** Asks landing mid-cycle that displace bets. Defer to the next table; the cycle is protected.
- **Hill charts as progress bars.** Reading "apex" as "50%" loses the signal. The dot's *movement* is the data, not its position.
- **Static hill charts.** Dots that never move = the team isn't engaging with the hill chart. Either drop it (use status instead) or fix the cadence of updates.
- **Hill charts on bug fixes or chore work.** Hill charts are for shaped bets with unknowns. Bugs and chores are status-tracked.
- **Pitches without no-gos.** Every pitch must name what's explicitly out of scope. Without no-gos, scope creep is invited.
- **Bets that automatically continue.** Last cycle's unfinished work that "just rolls over." Re-pitch and re-bet, or drop.
- **Appetite-as-estimate.** Setting appetite to whatever engineering says it'll take. The point of appetite is that it constrains, not validates.

## Cross-references

- `pitch-writing-shape-up` — how to author the pitches that enter the betting table.
- `shape-up-cadence` — the operational rhythm of the cycle preset (this skill focuses on planning; that skill focuses on the recurring cadence).
- `hill-chart-reasoning` — deeper guidance on reading and updating the hill chart.
- `sprint-planning` — the alternative delivery layer for teams running fixed sprints.
- `prioritization-frameworks` — produces the pitch ranking the betting table considers.
- `initiative` and `feature` spec types — `appetite`, `cycle`, and `hill_position` are the preset-conditional fields cycle planning populates.
- PM principle #3 (make tradeoffs visible) — the betting table's recorded decisions are the operating mechanism.
- Singer, *Shape Up* (basecamp.com/shapeup) — original source.
