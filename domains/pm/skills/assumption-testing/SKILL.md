---
name: assumption-testing
description: Design fast Torres-style assumption tests — desirability / viability / feasibility / usability — with pre-registered pass/fail criteria that resolve in days, not weeks. An MVP is a delivery, not a test.
metadata:
  audience: discovery-researcher, and the deferred risk-curator / pm-delivery-lead loaders
  purpose: framework-guidance
---

## What I do

Give agents the discipline to test the riskiest belief behind a bet *before* building on it — cheaply and fast. Teresa Torres' core insight: every solution rests on assumptions, and the job of discovery is to find the assumption whose failure would most hurt the bet and resolve it in days. This skill supplies the four assumption categories, the pre-registration discipline that keeps a test honest, and the fast-test shapes that resolve before the work they'd inform begins. When `discovery-researcher` picks what to de-risk, or a delivery lead asks "do we actually know this?", this is the method.

## When to use me

- a bet or PRD rests on a belief that, if wrong, sinks it — and no one has checked
- `discovery-researcher` is choosing what to test before authoring proceeds
- a stakeholder wants to "build an MVP to see if people want it" (usually the wrong move — see below)
- a Risks section names a scenario you can't yet picture because you're unsure a belief holds
- planning a discovery cycle and sequencing which assumptions to resolve first

## The four assumption categories

Every solution carries assumptions in four families. Name which kind you're testing — the test shape differs by family:

- **Desirability** — *do users actually want this?* Will they change behavior to adopt it? (The most common silent bet, and the most expensive when wrong.)
- **Viability** — *does it work for the business?* Segment economics, pricing, cost-to-serve, cannibalization, channel fit.
- **Feasibility** — *can we build it within the appetite?* Often co-tested with engineering; a spike answers it faster than a debate.
- **Usability** — *can users actually accomplish the job with this shape?* Distinct from desirability — they may want the outcome but fail at your flow.

Torres' move: list the assumptions across all four, then **rank by risk** — for each, ask "if this is false, how badly does the bet fail, and how sure are we it's true?" Test the highest-risk, lowest-confidence assumption first. Don't test assumptions that can only confirm; test the one whose *failure* you most need to catch early.

## Pre-registration — the discipline that keeps a test honest

Before you run anything, write down what result means what. A test whose pass/fail is decided *after* seeing the data isn't a test — it's a search for confirmation.

```
## Assumption test: <name>
Assumption under test: <one sentence — the belief, stated so it can be false>
Category:              <desirability | viability | feasibility | usability>
Disconfirming signal:  <what we'd see if the assumption is FALSE>
Method:                <interview | concept test | wizard-of-oz | smoke test | data pull | spike>
Sample:                <segment, N, recruit channel>
Pass/fail criteria:    <concrete, pre-registered threshold — decided BEFORE running>
Resolution time:       <days>
Owner:                 <who runs it>
What the result changes: <proceed / reframe / kill — decided in advance>
```

The two non-negotiable lines are **disconfirming signal** (name what failure looks like before you run) and **pass/fail criteria** (a concrete threshold set in advance). If you can't state what would prove you wrong, you're not testing — you're rationalizing.

## Fast-test shapes (resolve in days)

Pick the cheapest test that can move your confidence. Rough speed order:

- **Data pull** (hours) — the assumption is about *current* behavior; instrument or query existing analytics before designing anything richer. Fastest possible answer; do this first when it applies.
- **5-user interview** (days) — fastest signal on desirability and usability; story-based, past-behavior questions (see `discovery-interview-design`).
- **Concept test** (days) — show a mock, ask about a specific past situation; weight hard against social-desirability bias ("nice mock" ≠ "would use").
- **Smoke test** (days–week) — a landing page or fake door measuring real intent (signup, click-through) instead of stated intent.
- **Wizard-of-oz** (days–week) — fake the backend by hand, measure whether people actually use the front.
- **Spike** (days) — a timeboxed engineering probe for a feasibility assumption.

The rule: **the test must resolve before the work it would inform begins.** A test that takes longer than the cycle it's meant to de-risk can't de-risk it — redesign it smaller or pick a faster shape.

## Why an MVP is not a test

The most common anti-pattern: "let's build an MVP to test whether people want it." An MVP is a *delivery* — weeks of build, a real launch, sunk cost, and a result that arrives too late and too entangled to isolate the assumption. If you're willing to build the MVP regardless, you haven't tested anything; you've decided. A real assumption test is *cheaper than the thing it protects* and resolves *before* you commit. Build the MVP after the assumption clears, not as the way to check it.

## Worked example

> **Assumption:** SMB users will export data themselves if we give them a one-click button (desirability + usability).
> **Disconfirming signal:** in a 5-user test, fewer than 3 of 5 complete the export without help, OR they say they'd still rather ask support.
> **Method:** 5-user moderated concept test on a clickable mock.
> **Sample:** 5 SMB admins, recruited from last month's export-related support tickets.
> **Pass/fail:** ≥4 of 5 complete unaided AND ≥3 say they'd use it over filing a ticket.
> **Resolution:** 4 days.
> **Changes:** pass → proceed to PRD; fail on completion → usability reshape; fail on preference → the real opportunity isn't self-serve export, re-frame the bet.

## Anti-patterns

- **"Build an MVP to test it."** An MVP is a delivery, not a test. Test cheaper and earlier.
- **No pre-registered pass/fail.** Deciding what the data means after seeing it is confirmation-seeking, not testing.
- **No disconfirming signal.** A test that can only confirm the assumption is theater — name what failure looks like first.
- **Tests that outlast the work.** A quarter-long test can't inform a six-week cycle. Redesign smaller.
- **Confirmation-biased samples.** Recruiting only happy users tests the wrong thing; mix in churned, never-converted, and competitor-users.
- **Testing solutions before opportunities.** A/B-ing two solution shapes for an unconfirmed need optimizes the wrong layer.

## Cross-references

- `discovery-interview-design` — how to design the interview when the test is a 5-user interview.
- `opportunity-solution-trees-torres` — assumption tests hang off solutions in the OST; this is the "test" layer of the tree.
- `continuous-discovery-cadence` — assumption tests are a weekly habit, not a one-off project.
- `risk-surfacing` — an untested assumption is what a speculative "risk" really is; resolve it here instead of writing a made-up response.
- `pm-agent-doctrine` — compare-don't-replace: synthesize the test result alongside the PM's read, with traceability to the raw signal.
