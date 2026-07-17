---
name: product-vision-writing
description: The one-page product vision that ladders strategy → roadmap — rooted at the OST outcome, filling a who / problem / for-whom / unlike / approach / how-we'll-know template, and constraining the initiatives below it.
metadata:
  audience: product-strategist, roadmap-curator
  purpose: framework-guidance
---

## What I do

Give agents the discipline for writing a **one-page product vision** — the top of the strategy ladder that every initiative below it must serve. A vision names the future you're trying to create and *why it matters*, concretely enough that it can settle an argument about what to build next. It is not a slogan and not a feature list; it's the durable statement that outlives quarterly reshuffles and gives the roadmap a spine. My job is to keep the vision rooted in a real outcome and measurable, so it constrains decisions instead of decorating a slide.

## When to use me

- setting or refreshing product direction above the roadmap (`product-strategist`)
- a roadmap that's a pile of features with no through-line, and you need the "why" that orders it (`roadmap-curator`)
- an exec asks "what's the vision?" and the honest answer is a sentence nobody can act on
- new initiatives keep getting proposed with no way to say yes or no on strategy grounds

## The ladder — vision → strategy → roadmap → initiatives

The vision is one rung of a ladder, and its value is that each rung *constrains* the one below:

```
VISION       the future we're creating and why it matters (durable, 1 page)
  ↓ constrains
STRATEGY     the approach + the bets we're making to get there
  ↓ constrains
ROADMAP      the sequenced initiatives (now / next / later)
  ↓ constrains
INITIATIVES  the outcome-shaped bets (see roadmap-framing)
```

Read downward, each rung answers "how?"; read upward, each answers "why?". A roadmap item you can't ladder *up* to the vision is an orphan bet — it may be worth doing, but the vision doesn't yet justify it, and that's the conversation to have. This is why the vision is worth writing: it's the test that tells strategy drift from strategy.

The rungs also differ in **shelf life**, and getting that right is half the discipline. The vision is the slowest-changing rung — measured in years — because it names an enduring outcome, not a plan. Strategy shifts as you learn what works. The roadmap re-sequences every planning cycle. Initiatives turn over fastest of all. When a lower, faster rung starts dictating an upper, slower one — when this quarter's roadmap rewrites the vision — the ladder is upside down and the "vision" was really just a plan wearing the wrong label.

## Durable but not frozen — the right altitude

The hardest judgment in vision-writing is altitude. Pitch it too high and it's a mission statement ("empower every team") that never rules anything out; pitch it too low and it's a roadmap ("ship SSO by Q3") that expires next quarter. The vision sits in between: **specific about the future world you're creating, silent about the exact features that get you there.** A good test of altitude — the vision should survive a pivot in *how* you build while dying if the *outcome you're betting on* turns out wrong. If a feature-level change forces a vision rewrite, it was too low; if nothing could ever falsify it, it was too high.

## Rooted at the OST outcome

A vision floats unless it's anchored to an outcome the organization is actually trying to move. The **root of the vision is the top-level outcome** — the same altitude as the root of an Opportunity Solution Tree (`opportunity-solution-trees-torres`). The OST hangs opportunities and solutions under that outcome for *discovery*; the vision expresses the same outcome as *direction*, in language a stakeholder reads rather than a metric tree. Same anchor, two faces: the tree explores how to move it, the vision explains why moving it matters and what the world looks like once you have.

## The one-page template

Six clauses. Each is one or two sentences; the whole thing fits on a page.

- **Who** — the user/customer the vision serves. A specific segment, not "everyone."
- **Problem** — the pain or unmet need in their terms, today.
- **For whom (it matters most)** — the segment feeling that problem most acutely; where the value lands hardest.
- **Unlike** — the current alternative and why it falls short. (This clause is the `positioning-canvas` compressed to a line — anchor on the real fallback, not an ideal.)
- **Our approach** — the distinctive way we solve it. The bet, stated at the level of *how we're different*, not a feature list.
- **How we'll know** — the measurable outcome that tells us the vision is being realized. This is the clause that keeps the vision honest.

**Worked one-liner assembled from the template:**

> For **eng orgs whose roadmaps drift from reality** (who), status meetings run on claims nobody can verify (problem), and it hurts most **where leadership has stopped trusting delivery updates** (for-whom). **Unlike** a wiki-plus-tracker kept in sync by hand, our approach **graph-links specs to the commits and issues that satisfy them**, so "shipped" is substantiated, not asserted. **We'll know it's working when** teams stop reconciling status by hand and roadmap-vs-reality drift trends to zero.

## Writing it — the process

1. **Start from the outcome, not the product.** Name the top-level behavior/impact you're trying to move (`outcomes-over-outputs`). If you can't state it, you're not ready to write a vision — you have a feature idea.
2. **Draft the six clauses** — who / problem / for-whom / unlike / approach / how-we'll-know. Keep each to a sentence or two; brevity forces the choices a vague vision dodges.
3. **Assemble into prose.** The clauses should read as one paragraph, not a form. If it doesn't flow, a clause is probably fighting another one — usually "who" is too broad for the "problem" you named.
4. **Test it against a decision.** Take a real proposed initiative and ask "does the vision tell me yes or no?" If the vision is compatible with *any* initiative, it's too vague to steer.
5. **Test it against the roadmap.** Every `now`/`next` item should ladder up. Orphans are either mis-prioritized work or a gap the vision doesn't yet cover — surface which.

## The "would this change a decision?" test

The single test that separates a real vision from wallpaper: **can it settle an argument about what to build next?** A vision that every stakeholder nods at but that never rules anything out is decoration. A good vision is specific enough that some genuinely appealing feature ideas fail it — and that's the point. If nothing fails the vision, it's directing nothing.

## Anti-patterns

- **Vision as a slogan.** "Delight every customer, every day." Inspirational wallpaper — it can't settle a single decision about what to build. A vision has to be specific enough to say no.
- **Vision that's really a feature list.** "We'll ship SSO, mobile, and an API." That's a roadmap wearing a vision's title. The vision is the *why* those might matter; features are three rungs down.
- **No measurable "how we'll know."** A vision with no outcome clause can never be evaluated, so it can never be wrong — which means it's steering nothing. Root it in a real outcome (`outcomes-over-outputs`).
- **The everyone-vision.** "Who" is all users, "problem" is all problems. A vision that excludes no one directs no one. Narrow to the segment that feels the problem most.
- **Vision divorced from the roadmap.** A grand statement on a slide that no initiative actually ladders up to. If the roadmap doesn't serve the vision, one of them is fiction — reconcile them (`roadmap-framing`).
- **Vision that never changes vs. vision that changes quarterly.** A vision should be durable but not frozen; if it flips every planning cycle it's a strategy, and if it hasn't been revisited in two years it's probably stale. Revisit deliberately.
- **Borrowed vision.** A competitor's vision with the names swapped, or a generic "best-in-class platform" that any company could claim. If the vision doesn't encode *your* specific bet about *your* users, it steers nothing that a rival's wouldn't.
- **Vision the team can't recite.** If nobody on the team can state the vision from memory, it's not doing its job as a shared north star — it's a document. A real vision is short and sharp enough to repeat.

## Cross-references

- `outcomes-over-outputs` — the "how we'll know" clause must be outcome-shaped; the vision is this framework at the top of the ladder.
- `roadmap-framing` — the rung below; initiatives are the outcome-shaped bets that must ladder up to the vision.
- `opportunity-solution-trees-torres` — the vision and the OST share the same outcome root; the tree explores it, the vision expresses it.
- `positioning-canvas` — the "unlike" clause is positioning compressed to a line.
- `product-vision-writing` pairs with strategy docs; prior art: Marty Cagan / SVPG on product vision, Roman Pichler's Product Vision Board, Geoffrey Moore's "For / Who / Unlike" positioning template.
