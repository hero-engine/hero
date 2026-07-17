---
name: outcome-drift
description: Turn a whole roadmap into a drift signal — the ratio tally that reads the realized input/output/outcome shape against ~60/30/10, and the stale-item taxonomy that catches items a roadmap is quietly lying about. The method behind the roadmap drift critic.
metadata:
  audience: roadmap-reviewer, and the deferred stale-roadmap-scrubber
  purpose: curation
---

## What I do

Turn a roadmap into a **drift signal**. `outcomes-over-outputs` gives you the lens for reading *one* item at the right altitude; this skill scales that lens to a *whole board* and adds the second half of drift — **staleness**. A roadmap drifts in two directions at once: its framing slides down the ladder (bets become a build queue with quarters attached), and its items rot in place (shipped things still marked active, "Later" items older than the horizon, outputs floating with no outcome above them). I supply the two instruments that make both drifts measurable rather than felt: the **ratio tally** and the **stale-item taxonomy**.

The point is not to reframe every line. Maintenance and compliance work is legitimately output-shaped, and a healthy roadmap carries some. The point is to produce a *checkable* verdict — "this board is 62% outputs against a ~30% healthy share, and 4 of its 11 items are stale" — that a PM can act on, challenge, or overturn by reading the same board.

## When to use me

- a whole-roadmap critique pass (the `roadmap-reviewer` drift critic's core loop)
- a periodic board hygiene sweep (the deferred `stale-roadmap-scrubber`)
- before a quarterly/cycle planning review, to surface what the current board is silently claiming
- when a stakeholder says "the roadmap feels like a feature list" and you need to prove or disprove it with a number

This is a *board-level* instrument. For framing a single bet, load `outcomes-over-outputs` directly; for auditing one Risks section, load `risk-surfacing`. Reach for me when the unit of analysis is the roadmap itself.

## The ratio tally

The tally operationalizes the `outcomes-over-outputs` ladder — **load that skill for the rung definitions; do not restate the ladder here.** The method:

1. **Bucket every roadmap item by its top rung.** Read each item as written and place it: does it name a behavior/impact change (outcome), a thing to ship (output), or effort/capacity (input)? Score the item by the *highest* rung its framing actually reaches, not the rung it aspires to — "Redesign onboarding to lift D7 retention 22%→30%" is an outcome; "Redesign onboarding" is an output.
2. **Compute the realized shape.** Count the buckets and read the percentages: `outcome% / output% / input%`. That triple *is* the drift signal.
3. **Compare against ~60/30/10.** A healthy board reads roughly 60% outcomes / 30% outputs / 10% inputs (per `outcomes-over-outputs` — a heuristic for *shape*, not a gate). The drift finding is the delta: `realized 15/80/5 vs. healthy 60/30/10 → the board is a build queue`. Name the number; don't just assert "output-heavy."
4. **Localize the drift.** A ratio is an aggregate — say *where* it concentrates. "All five Now-horizon items are outputs; the three outcome-framed items are all parked in Later" is a more useful finding than a single board-wide percentage, because it tells the PM which part of the board stopped being a set of bets.

**Worked tally.** A board of eight items buckets as: 1 outcome, 6 outputs, 1 input → `12/75/13`. Against ~60/30/10 that is heavy output drift. The finding is *not* "reframe all six outputs" — it's targeted: the items that are **bets** (can be prioritized on expected behavior change) must ride on outcomes; the items that are legitimately infra/maintenance should **hang under** the outcome they enable rather than float as peer line-items. See the worked audit in `outcomes-over-outputs` for the item-by-item reframe pattern.

## The stale-item taxonomy

Framing drift is only half. The other half is items that have quietly stopped being true. Four stale types, each with the **observable that triggers it** and the **recommended action** (the critic *recommends* — a human acts; doctrine 2):

| Type | Observable trigger | What it means | Recommended action |
|---|---|---|---|
| **No-movement** | No status/edit/graph activity in N planning cycles (default N = 2) while still marked active | The item is parked but presenting as in-flight | **Refresh or defer** — confirm it's still a live bet, or move it to Later with a reason |
| **Lying-shipped** | Linked delivery state shows shipped/done, but the roadmap item still reads active/committed | The roadmap is claiming future work that already happened — the most corrosive drift, because it makes the whole board untrustworthy | **Archive** — reconcile to shipped; if an outcome was attached, open the measurement follow-up |
| **Over-horizon** | A `Later` (or equivalent far-horizon) item older than the planning horizon itself | It's been "someday" longer than "someday" is defined to be — it's not deferred, it's abandoned | **Drop-with-reason or re-commit** — decide explicitly; a permanent Later is a graveyard pretending to be a plan |
| **Orphan-output** | An output-framed item with no outcome above it in the graph and no outcome named in its own framing | Nobody can say what shipping it would change — motion, not a bet | **Re-hang under an outcome** — name the behavior it's betting to move, or drop it |

**How to read the triggers honestly.** Every trigger is an *observable*, not a calendar guess. "No-movement" checks the graph for activity, not the created-date. "Lying-shipped" reads live delivery state, not the PM's memory. Staleness asserted from the calendar alone ("this is old, kill it") is drift theater — it flags healthy long-horizon bets as rot. Always cite the observable that fired: *"item X: linked feature `csv-export` shows `status: completed` 2026-05-11, roadmap item still `committed` — lying-shipped."* That citation is what makes the finding checkable rather than contrarian (doctrine 1).

## The honest-roadmap question

After the tally and the taxonomy, run one synthesizing pass: **what does this roadmap claim that reality contradicts?** Read the board as a skeptical exec would — not "is each item fine" but "if I trusted this board completely, what would I believe that isn't true?" The lying-shipped items are the sharpest version, but the question is broader: an outcome with no baseline, a Now horizon that can't fit in the cycle's capacity, a bet whose disconfirming signal already fired. The output is a **drift verdict** — `honest` / `drifting` / `build-queue` — backed by the ratio number and the stale count, never a bare adjective.

## Anti-patterns

- **Drift theater.** Flagging every output as drift. Maintenance, compliance, and infra work is legitimately output-shaped; the discipline is that *bets* ride on outcomes and outputs hang under one — not that the word "build" is banned.
- **Staleness by calendar alone.** Marking an item stale because it's old, without checking the graph for movement or the delivery state for lying-shipped. A long-horizon bet is not rot. Check the observable, not the date.
- **Ratio without localization.** "This board is 70% outputs" with no note on *where* the drift concentrates. An aggregate the PM can't act on is a statistic, not a finding.
- **Recommendations with no action.** "This item is stale" is half a finding. Name the recommended action (refresh / drop-with-reason / archive / re-hang) so the PM has something to accept or reject.
- **Ungrounded drift claims.** "This feels like a feature list" with no tally behind it. The whole value of the drift signal is that it's a number someone can recompute; an unbacked vibe is the free-association `pm-agent-doctrine` forbids.
- **Auto-correcting the board.** Silently refiling horizons or archiving items. The critic surfaces; the human decides (doctrine 2).

## Cross-references

- `outcomes-over-outputs` — the ladder and the ~60/30/10 heuristic this skill tallies against; load it for the rung definitions rather than restating them.
- `roadmap-framing` — where a single initiative's Bet/Evidence/Tradeoffs get framed; a stale item's refresh routes back through this.
- `risk-surfacing` — an aging bet whose disconfirming signal is overdue is a risk, not just a stale line; frame it in scenario/indicator/response terms.
- `pm-agent-doctrine` — every drift finding cites its observable (doctrine 1) and recommends rather than decides (doctrine 2).
