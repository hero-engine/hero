---
name: feature-comparison-framing
description: Build a competitive feature matrix that drives a decision instead of a checkbox arms race — separate must-match parity from optional differentiation from white space, ground every cell in a retrieved source, and frame the matrix around the outcome, not the feature count.
metadata:
  audience: competitive-analyst (Wave-2)
  purpose: framework-guidance
---

## What I do

Turn competitive research into a feature matrix that makes a *decision*, not a scoreboard. The default failure of a feature-comparison table is the checkbox arms race: a grid of ✓/✗ across ten competitors that implies "we're behind, catch up on everything." That grid rewards feature-matching over strategy and buries the two judgments that matter — *what we must match to stay credible* and *where we could win*. This skill supplies the matrix shape that keeps those judgments in front, grounds every cell in a source (inherited from `competitive-research`), and frames the whole thing around the outcome rather than the count. Forward-authored for the Wave-2 `competitive-analyst`.

## When to use me

- building a competitive feature matrix for a category or a specific bet
- deciding what a PRD must include to reach parity vs. what would differentiate
- a stakeholder wants a "how do we compare" table and you need it to drive a decision, not anxiety
- pairing with `competitive-research` to turn a sourced teardown into a comparison

## The three-band matrix

Don't build a flat ✓/✗ grid. Classify each capability into one of three bands, because each implies a *different decision*:

- **Must-match (parity)** — table stakes; credible competitors have it and its absence costs deals. Decision: *match it, don't over-invest.* Parity is defensive — reaching parity is enough; beating everyone on a parity feature is wasted effort.
- **Optional differentiation** — a capability you could do materially better, or that no credible competitor does well. Decision: *consider investing here — this is where advantage compounds.* Requires sourced evidence that competitors *don't* do it well.
- **White space** — a need no one in the category serves. Decision: *the highest-leverage bet, held to the strictest evidence bar* (it's the easiest band to imagine rather than verify).

```
| Capability          | Us  | Comp A | Comp B | Comp C | Band            | Source / date            |
|---------------------|-----|--------|--------|--------|-----------------|--------------------------|
| SAML SSO            | ✗   | ✓      | ✓      | ✓      | must-match      | *-pricing pages, 2026-07 |
| Scheduled export    | ✗   | ✗      | ✗      | partial| differentiation | docs teardown, 2026-07   |
| Export-to-warehouse | ✗   | ✗      | ✗      | ✗      | white space     | verified absent, 2026-07 |
```

The Band column is the point of the whole matrix — it converts a row of checkmarks into a decision. A matrix without it is a scoreboard.

## Every cell is sourced

A ✓ or ✗ is a competitive claim, and competitive claims are grounded in retrieved, dated sources — never model memory (this is `competitive-research`'s retrieval rule, and it applies to every cell). A ✗ ("Competitor B lacks this") is *especially* dangerous to assert from memory, because you're claiming an absence you may simply not have found. Prefer "verified absent (checked docs + pricing, 2026-07)" over a bare ✗, and mark unverified cells explicitly as unverified rather than guessing.

## Frame around the outcome, not the count

The matrix serves a bet, and the bet is an *outcome* (see `outcomes-over-outputs`). "We have 7 of their 10 features" is a count, and counts drive arms races. Reframe the read around what the customer is trying to accomplish:

- Group capabilities by the **job the customer is doing**, not by feature name. "Getting data out of the tool" (export, scheduling, formats, warehouse) is a job; four scattered checkboxes are not.
- Ask, per job: *are we credible (parity) and can we win (differentiation)?* — not "how many boxes do we have."
- A matrix that ends in "so the wedge is scheduled + warehouse export, because no one serves the data-out job well" is decision-useful. A matrix that ends in "we're behind by 3 features" is anxiety.

## A worked example — from grid to wedge

A team building a data-analytics tool wants to know "how do we compare on data export?" The naive output is a 12-checkbox grid across four competitors. The framed output groups by the customer's job ("get data out and keep it flowing") and lands a decision:

```
Job: get data out of the tool and keep it flowing
| Capability            | Us | A | B | C | Band            | Source / date (verified)     |
|-----------------------|----|---|---|---|-----------------|------------------------------|
| CSV export            | ✓  | ✓ | ✓ | ✓ | must-match      | product walk, 2026-07-14     |
| Excel / JSON formats  | ✓  | ✓ | ✓ | ✗ | must-match      | docs, 2026-07-14             |
| Scheduled / recurring | ✗  | ✗ | partial | ✗ | differentiation | docs + trial, 2026-07-15 |
| Export to warehouse   | ✗  | ✗ | ✗ | ✗ | white space     | verified absent, 2026-07-15  |
| API-driven export     | ✓  | ✓ | ✓ | ✓ | must-match      | api docs, 2026-07-14         |
```

Read: on the must-match rows we're at parity (fine — hold, don't over-invest; note we already lead C on formats). The two rows that matter are **scheduled export** (only B does it, partially — a differentiation wedge) and **export-to-warehouse** (nobody serves it — white space, held to the strictest evidence bar since a full ✗ column is the most tempting to imagine).

**Decision the matrix produces:** the bet isn't "catch up on export features" (we're already at parity) — it's *"own the data-out job by shipping scheduled + warehouse export, because no competitor serves the flow end-to-end."* That's a wedge framed on the customer's job, not a feature tally. Contrast the count-framing an unframed grid would have produced: "we have 3 of their 4 export features" — anxiety, no decision.

## Anti-patterns

- **Checkbox arms race.** A flat ✓/✗ grid with no Band column, implying every gap is a mandate. Classify parity vs differentiation vs white space or the matrix drives feature-matching instead of strategy.
- **Unsourced cells.** ✓/✗ from model memory. Every cell is a sourced, dated competitive claim; a ✗ especially needs "verified absent," not a guess.
- **Over-investing in parity.** Treating a must-match feature as a place to out-build everyone. Parity is defensive — match and move on.
- **Imagined white space.** A whole column of ✗ asserted without verifying each competitor actually lacks it. Strictest evidence bar of the three bands.
- **Count-framing.** "We're behind by N features" as the headline. Frame around the customer's job and the outcome, not the tally.
- **Matrix as the deliverable.** The matrix is an input to a bet, not the bet. It should end in a decision (the wedge), not a grid.

## Cross-references

- `competitive-research` — supplies the sourced, dated teardown that fills the cells; retrieval-not-memory applies to every one.
- `outcomes-over-outputs` — frame the comparison around the customer's job and the outcome, not the feature count.
- `roadmap-framing` — the matrix's "wedge" conclusion becomes the differentiation evidence for a bet.
- `prioritization-frameworks` — parity vs differentiation feeds Impact and strategic-context judgments when ranking the resulting work.
