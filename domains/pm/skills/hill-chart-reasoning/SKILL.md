---
name: hill-chart-reasoning
description: Basecamp's hill chart read correctly — position signals unknowns-remaining, not percent-done; uphill is figuring-out, downhill is executing, and a scope stuck uphill is the signal that matters.
metadata:
  audience: pm-delivery-lead, cycle-planner, roadmap-curator
  purpose: framework-guidance
---

## What I do

Give agents the correct mental model for Basecamp's **hill chart** (Shape Up) — and correct the near-universal misreading of it as a progress bar. A hill chart plots each scope on a curve: **up the left side = still figuring out the approach, over the top and down the right = the approach is known and you're grinding it out.** Position encodes **how many unknowns remain**, not what fraction of the work is done. Read that way, the chart tells you where the *risk* is, which a percent-complete number never can.

## When to use me

- reporting where a cycle's work actually stands, honestly (`pm-delivery-lead`)
- planning a cycle and reasoning about which scopes carry the unknowns (`cycle-planner`)
- a status update says "80% done" and you need to know if that 80% includes the hard part (`roadmap-curator`)
- a scope hasn't moved on the hill in two check-ins and you need to know why

## The hill has two sides

```
                    ● over the top: figured out
                  ╱   ╲
   uphill        ╱     ╲     downhill
   (figuring    ╱       ╲    (executing —
    out —      ╱         ╲    approach known,
    unknowns  ╱           ╲   just work left)
    dominate)●             ●─────────→ done
```

- **Uphill** — you're still discovering *how* to solve it. Open questions, unproven approaches, "we think this will work." Effort spent here buys **knowledge**, not visible output. Being uphill isn't behind schedule; it's the phase where the unknowns get resolved.
- **Over the top** — the moment the approach is proven. You now *know* what building it takes. Nothing surprising remains, even if lots of work does. Crossing the top is a real event worth naming out loud: it's when a scope's estimate becomes trustworthy.
- **Downhill** — execution. The unknowns are gone; what's left is grinding out the known work. Estimates here are trustworthy in a way uphill estimates never are.

## Position = unknowns remaining, NOT percent done

This is the whole point, and the thing everyone gets wrong. A scope can be **80% of the effort complete and still uphill** — if the remaining 20% contains the unproven part everything depends on. Conversely a scope can be **20% of the effort complete but already over the top** — the hard question is answered, the rest is typing.

"How much is left?" is the wrong question because it can't distinguish *known* work from *unknown* work. "What don't we know yet?" is the right one. The hill answers it: distance from the top is unresolved-unknowns, and only the downhill distance is trustworthy remaining effort.

## Reading a stuck scope

Movement on the hill is the signal — not position alone.

- **Stuck uphill across check-ins** is the loudest alarm the chart produces. It means the team keeps working but isn't *resolving* the unknown — the approach still isn't proven. That's when to intervene: pair someone in, cut the scope, or spike the risky part directly. A burndown would show hours ticking down and hide this completely.
- **Jumped to the top fast** — either genuinely easy or the team hasn't hit the hard part yet. Ask what unknown they proved.
- **Downhill but slow** — a scheduling/capacity issue, not a risk issue. Different problem, different fix.
- **Slid back uphill** — rare but important: the team thought the approach was proven and discovered it wasn't. Honest and valuable to show; a chart that only ever moves rightward is being gamed.

## Scopes, not tasks — what sits on the hill

The dots on a hill chart are **scopes**, not individual tasks. A scope is a meaningful slice of the work that can be talked about as a unit — "expense submission," "approval routing" — not "write the validation function." This granularity is deliberate: tasks are binary (done or not) and belong on a to-do list, while scopes have a *knowledge trajectory* worth tracking. If your hill chart has thirty dots, you're plotting tasks; collapse them into the handful of scopes that actually carry distinct unknowns.

Good scopes for a hill chart are **vertical** (end-to-end through the layers, like a story-map release slice — see `story-mapping`) rather than **horizontal** (a whole layer). A horizontal scope like "the backend" can be 90% built and still hide whether the thing *works*, so it never crests the hill honestly. A vertical scope proves its unknown as soon as the thin path works end to end.

## How it differs from a burndown

A burndown chart plots *remaining effort over time* and implicitly assumes all remaining work is equally knowable — it treats an unproven spike and a known chore as interchangeable hours. The hill chart plots *confidence/unknowns* and deliberately separates them. A burndown trending down looks healthy even when the entire remaining effort is one unproven approach that could blow up. The hill makes that scope visibly stuck uphill. Use a burndown for capacity pacing; use the hill for *where the risk lives*.

## The check-in ritual

The hill chart earns its value only when it's updated on a rhythm and read for *movement*. A useful cadence:

- **Each check-in, the person doing the work places the dot** — not a manager, and not by counting hours. The question they answer is "what do you still not know?", and the dot's distance from the top is their honest answer.
- **Compare against last check-in.** The story is in the *delta*: did this scope climb? crest? stall? A snapshot is nearly worthless; a trajectory is the whole signal.
- **Interrogate the stalls, celebrate the crests.** A dot that crossed the top means an unknown got resolved — worth noting *what* got proven. A dot that didn't move is the conversation to have now.

Because the placement is subjective, the chart only works in a blame-free setting. If being uphill is treated as "behind," people will report optimistic positions to avoid scrutiny, and the chart becomes the very progress-theater it was meant to replace.

## What a hill chart can't tell you

The hill shows *risk resolution*, not calendar. A scope can crest the hill and still take three weeks of known downhill work — the hill says "no surprises left," not "almost shipped." Pair it with a capacity/date view (a burndown or the cycle's appetite in `cycle-planning`) when the question is *when*, and use the hill when the question is *how confident*. Confusing the two — reading a crest as "nearly done" — is how teams get blindsided by a long downhill.

## Anti-patterns

- **Hill position as %-done.** Dragging a dot to 90% "because most of the hours are spent." Position is unknowns-remaining; if the hard question is still open, the dot stays uphill regardless of hours burned.
- **Everything-at-the-top optimism.** Every scope parked near the summit on day two. Either the work was trivial or — far more likely — the team is reporting hope, not resolved unknowns. Ask what got *proven* to earn each position.
- **No movement between check-ins.** Dots that sit in the same place update after update. Not a neutral status — it means unknowns aren't being resolved. Investigate, don't wait.
- **Big-batch scopes that can't move.** One giant scope that can only be "uphill" or "done" with nothing in between. Break it so the hill can actually show progress through the unknowns.
- **Treating uphill as "behind."** Uphill is the *normal* early phase, not a delay. Pressuring a team off the hill before the approach is proven just relocates the unknown into downhill where it detonates as a surprise.
- **Reporting the hill only at the end.** The chart's value is the *trajectory* across check-ins. A single snapshot at cycle-end tells you nothing about where the risk was.

## Cross-references

- `cycle-planning` — the hill is how you narrate a cycle's progress mid-flight; scopes are the units it tracks.
- `pitch-writing-shape-up` — pitches name the rabbit holes and no-gos; those are exactly the unknowns a scope has to climb past to crest the hill.
- `roadmap-framing` — a scope stuck uphill is upstream signal that a `now` initiative may not land when claimed.
- `story-mapping` — release slices give you scopes small enough that the hill can show real movement.
- Prior art: Basecamp / Ryan Singer, *Shape Up* — the hill chart and uphill/downhill framing.
