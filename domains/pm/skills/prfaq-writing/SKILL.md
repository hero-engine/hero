---
name: prfaq-writing
description: Amazon PR/FAQ working-backwards — draft a mock press release plus an anticipated FAQ *before* building, as if the thing launched today, to surface the "dragons" (the hard customer and internal questions) while the bet is still cheap to change. The value is the reasoning that surfaces the dragons, not the copy.
metadata:
  audience: stakeholder-communicator
  purpose: working-backwards
---

## What I do

Provide the discipline for the **PR/FAQ** — Amazon's working-backwards artifact: a one-page mock **press release** describing the launched product from the customer's point of view, plus an **anticipated FAQ** answering the hard questions a launch would raise. Both are written *before* a line of code, as if the thing shipped today. The point is not marketing copy. The point is that committing to the customer-visible promise *first* forces every hidden assumption into the open while the bet is still cheap to change.

**The deliverable is the reasoning that surfaces the dragons, not polished prose.** A PR/FAQ that reads beautifully but never names a hard question has failed; a rough one that drags three deal-breaking questions into the light has succeeded. Judge it on the argument, not the wordsmithing.

## When to use me

Load this skill when:

- a bet needs a full working-backwards case before commit — a major launch, a new product surface, a platform pivot
- `stakeholder-communication` flagged a moment as "reach for working-backwards" and deferred the format here
- a team is arguing about *what to build* and keeps talking in features instead of customer outcomes
- you want to pressure-test whether a proposed thing is worth building at all — the FAQ half is a cheap kill-switch

## The press release — the customer-visible promise

One page, past tense, written the day of launch. The shape:

- **Headline** — the customer benefit in a sentence a real customer would care about, not the feature name.
- **Sub-head** — who it's for and the one-line why-now.
- **Problem** — the customer pain, in the customer's terms, grounded in real evidence (link intake / interviews — never invent the pain).
- **Solution** — what the customer can now do, described as an experience, not an architecture.
- **Quotes** — a leader quote and a customer quote. **These are placeholders to be sourced, never fabricated.** Write `[customer quote — to source from a design-partner interview]`, not an invented testimonial. A fabricated quote is the cardinal sin (`pm-agent-doctrine` doctrine 1).
- **Call to action** — how a customer starts.

If you cannot write the headline as a benefit a customer would repeat, the bet may not have a customer. That is a finding, surfaced now for the cost of a page.

## The FAQ — the load-bearing half, where the dragons live

The press release sells the dream; the FAQ is where you **hunt the dragons** — the hard customer and internal questions you would rather not answer. This is the half that earns the artifact. Split it:

- **Customer FAQ** — the questions a skeptical customer asks. "How is this different from the workaround I already use?" "What does it cost me to switch?" "What happens to my existing data?" Answer honestly or name the gap.
- **Internal / stakeholder FAQ** — the questions leadership, eng, legal, and finance ask. "Why us, why now?" "What's the hardest technical unknown?" "What has to be true for this to work, and is it?" "What are we *not* doing?" "How could this fail?"

**A dragon is any question whose honest answer threatens the bet.** The discipline: write the questions you're avoiding *first*, then answer them. A FAQ that only asks soft questions is selling a conclusion — the same failure `evidence-synthesis` names as sanding off the outlier. If a dragon has no good answer yet, say so and name the assumption test that would resolve it (`assumption-testing`) rather than papering over it.

## When the answer is "don't build it"

The PR/FAQ is a cheap kill-switch, and killing a bet on one page is a *win*, not a failure of the exercise. If the headline won't land as a customer benefit, or a customer-FAQ dragon has no answer, the artifact did its job — it surfaced the problem for the price of a page instead of a quarter. Working-backwards that never kills anything isn't being used as a filter.

## Anti-patterns

- **Marketing copy over argument.** Polishing the prose while the FAQ dodges every hard question. The reasoning is the deliverable; pretty is not.
- **A soft-ball FAQ.** Only asking questions with easy answers. The dragons are the ones you're avoiding — write those first.
- **Fabricated quotes.** Inventing a customer or leader testimonial to make the release read real. Placeholders to source, always (doctrine 1).
- **Building anyway.** Running the exercise, hitting an unanswerable dragon, and shipping the plan unchanged. Then it was theater, not a filter.
- **Feature-list release.** A press release that enumerates capabilities instead of describing one customer outcome. If it reads like a changelog, it's working *forwards*.
- **PR/FAQ as after-the-fact justification.** Writing it to rationalize a decision already made. Working-backwards only works *before* commit.

## Cross-references

- `exec-narrative` — the sibling working-backwards format (the 6-pager); reach for it when the case is a strategy narrative rather than a launch. Same argument-over-prose discipline.
- `stakeholder-communication` — names the working-backwards pattern and defers the full format here; audience-shaped *cuts* are its job, the *full artifact* is this one.
- `pm-agent-doctrine` — quotes are placeholders until sourced (doctrine 1, never fabricate); the PR/FAQ is a *proposal* the human commits, not a decision the agent makes (doctrine 2).
- `assumption-testing` — an unanswered dragon becomes a pre-registered assumption test, not a hand-wave.
- `outcomes-over-outputs` — the headline is a customer outcome, not a feature; if it isn't, the bet needs reframing.
