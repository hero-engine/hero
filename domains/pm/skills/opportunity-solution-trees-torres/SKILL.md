---
name: opportunity-solution-trees-torres
description: Teresa Torres' Opportunity Solution Tree — map a clear outcome to opportunities, solutions, and assumption tests before authoring.
compatibility: opencode
metadata:
  audience: discovery-researcher, pm-investigator, product-strategist
  purpose: framework-guidance
---
## What I do

Provide the mechanics of Teresa Torres' Opportunity Solution Tree (OST) — the four-level map from outcome down to assumption tests — and the discipline of mapping the opportunity space before authoring solutions. The OST is a thinking tool, not a deliverable; it survives across weeks of continuous discovery and feeds the `Evidence` section of any initiative or PRD that emerges from it. Source: Torres, *Continuous Discovery Habits* (2021).

## When to use me

- Starting on a fuzzy problem where the solution isn't obvious.
- Triaging a flood of feature requests that all seem to point at the same underlying pain.
- Investigating a metric that's flat or declining and you don't yet know why.
- Authoring an `Evidence` section on an initiative where the bet needs explicit grounding.
- Pushing back on "let's just build X" when the team has skipped the opportunity space.

## The four levels

The tree is read top-down. Each level constrains what's valid at the level below.

```
                      [ Outcome ]
                          |
        +-----------------+-----------------+
        |                 |                 |
   [ Opportunity ]   [ Opportunity ]   [ Opportunity ]
        |                 |
   +----+----+            |
   |         |            |
[ Sol ]   [ Sol ]      [ Sol ]
   |
+--+--+
|     |
[AT] [AT]   <-- assumption tests
```

### Level 1: Outcome

A single, leading metric that the team is trying to move. The constraint that everything below must serve.

**Good outcomes:**
- "Increase the proportion of trial users who reach the activation event within 7 days from 22% to 35% by end of Q3."
- "Reduce monthly support contacts per active account from 0.40 to 0.25 by year-end."

**Bad outcomes:**
- "Revenue" (lagging — moves too slowly to validate experiments against; everything you do affects it).
- "Ship the redesign" (output, not outcome — names a deliverable, not a movement).
- "Better engagement" (unmeasurable — no baseline, no target, no definition).

The outcome should be a **leading** metric where possible (something that moves within weeks, not quarters), and **observable** (the team has instrumentation or can add it). See `metrics-design` for the full rubric.

### Level 2: Opportunities

User needs, pain points, or desires that, if addressed, would plausibly move the outcome. Opportunities are **user-shaped**, not solution-shaped.

**Good opportunities:**
- "New users can't tell which feature to try first."
- "I lose track of which conversations I've already responded to."
- "I'm not sure if my export actually worked."

**Bad opportunities:**
- "We need an onboarding wizard." (Solution-shaped — predetermines the answer.)
- "Add a notification center." (Solution-shaped.)
- "Users want more features." (Too vague to test against.)

Opportunities come from research — interviews, support tickets, session recordings, sales calls. Each opportunity should be traceable to specific evidence ("3 of 8 trial users in Oct interviews said this verbatim"). Opportunities are written in the user's voice, not the team's framing.

A healthy outcome has 5–20 mapped opportunities. Fewer means you haven't done the discovery; more means you haven't synthesized.

### Level 3: Solutions

Specific things you could build to address an opportunity. One opportunity typically has 2–5 candidate solutions worth considering — the discipline of generating multiple solutions per opportunity is what prevents the team from anchoring on the first idea.

**Good solutions:**
- "Surface a 'recommended first action' card on the empty home screen."
- "Show inline 'last replied at' timestamps in the conversation list."

**Bad solutions:**
- "Redesign the home screen." (Too broad — not one solution, an initiative.)
- "Improve discoverability." (Solution-as-aspiration — name the thing you'd actually build.)

The tree's value here is *forcing solution divergence*. Without an OST, teams commonly take the first solution that sounds plausible and start authoring. With one, you compare 2–5 candidates against the same opportunity before picking.

### Level 4: Assumption tests

Every solution rests on assumptions. The tree's job is to surface them and assign each one a test that resolves quickly.

For a given solution, ask: "What has to be true for this to work?" Categories:
- **Desirability** — will users actually want this?
- **Viability** — does it serve the business?
- **Feasibility** — can engineering build it in the appetite?
- **Usability** — can users figure it out?
- **Ethical** — should we build this?

For each load-bearing assumption, design a test:

```
Assumption: "Trial users will engage with a 'recommended first action' card."
Test: Add a faked card to 50 trial accounts (Wizard of Oz). Measure click-through over 5 days.
Pass: >25% click-through.
Fail: <10% click-through. Revisit the opportunity.
Resolves in: 5 days.
```

Tests that take a quarter to resolve are not assumption tests — they're projects. Good assumption tests resolve in **days, not weeks**, using techniques like prototypes, fake doors, concierge tests, and small-batch interviews.

## How to start a tree

1. **Anchor the outcome.** One sentence. Leading metric, baseline, target, deadline. If the team can't agree on the outcome, the tree won't help — fix that first.
2. **Empty the opportunity space.** Read existing research, support tickets, NPS comments, sales call notes. List every distinct user need you can attribute to evidence. Don't filter yet.
3. **Synthesize and prune.** Cluster duplicates. Drop opportunities you can't trace to evidence. Aim for 5–20 well-scoped opportunities, each in the user's voice.
4. **Pick one opportunity to attack first.** Use a `prioritization-frameworks` scoring or a value-vs-effort 2x2. The chosen opportunity is the tree's active branch.
5. **Generate 2–5 solutions for the active opportunity.** Don't author them yet — generate, then compare.
6. **For the chosen solution, list its assumptions.** Categorize (desirability / viability / feasibility / usability / ethical). Rank by load-bearing-ness.
7. **Design tests for the top 1–3 assumptions.** Each test names its pass/fail criteria and resolves in days.
8. **Run the tests. Update the tree.** Solutions get killed, opportunities get re-ranked, sometimes the outcome itself gets revised. The tree is alive.

## The discipline of mapping before authoring

The most common failure of teams new to OST is **jumping from outcome to solution**. Someone in a meeting says "let's build an onboarding wizard" and the team starts authoring a PRD without ever naming the opportunity that the wizard supposedly addresses.

The rule: **no initiative gets `committed` status until its OST branch exists.** The branch doesn't have to be elaborate — outcome → 1 opportunity → 1 solution → 1 named assumption is the minimum — but it has to be there, traceable to evidence, and recorded in the `Evidence` section of the initiative.

This is the gate that prevents solution-anchored backlogs. When a stakeholder says "we should build X," the response is "what opportunity does X address?" If the answer is shaky, the OST work hasn't happened yet.

## How an OST becomes Evidence on an initiative

When an OST branch matures (assumption tests run, solution survives), it gets distilled into the initiative's `Evidence` section:

```markdown
## Evidence

### Outcome served
Increase 7-day trial activation from 22% to 35% by end of Q3.

### Opportunity
"New users can't tell which feature to try first."
— 6/12 trial interviews in Oct 2026 named this verbatim
— 23 support tickets in last 30 days asked "where do I start?"

### Solution shortlisted
Recommended-first-action card on empty home screen.
(Alternatives considered: full onboarding wizard — too big appetite;
checklist sidebar — user testing showed it was ignored.)

### Assumption tests run
- Wizard of Oz test, 5 days, 50 accounts, 31% click-through (PASS threshold 25%)
- Concierge test with 8 users — 6 reached activation within 7 days

### Confidence
Medium-high. Two passing tests; remaining risk is feasibility under
the chosen appetite.
```

The initiative's `Bet` section names the bet; the `Evidence` section shows the OST work that backs it. An initiative without an OST-backed Evidence section is an initiative that hasn't earned its `committed` status.

## Continuous discovery cadence

The OST is not authored once. It's revisited weekly as part of continuous discovery (see `continuous-discovery-cadence`). The cadence:

- **Weekly:** Three customer touchpoints feed new evidence into the tree. The active opportunity gets re-examined. One assumption test resolves (or progresses).
- **Per cycle / sprint:** The tree's active branch informs what gets authored as a story or PRD.
- **Per outcome milestone:** The whole tree is revisited — are we still pointed at the right outcome?

A tree that hasn't changed in a month is either an outcome that's been solved or a team that's stopped doing discovery. Either way, surface it.

## Anti-patterns

- **Trees that skip the opportunity layer.** Outcome → solution → done. The opportunity space is the whole point — collapsing it is just disguised solution-anchoring.
- **Solution-shaped opportunities.** "We need a notification center" is not an opportunity. The opportunity is the user need that a notification center might address.
- **Assumption tests that take a quarter.** If the test takes longer than the cycle, it's a project, not a test. Find a smaller test that resolves in days.
- **Trees that ossify into roadmaps.** The tree is a living thinking artifact. Once you turn it into a Gantt chart with deadlines, you've killed it.
- **OST as a deliverable.** The OST is *thinking*, not a document to ship to stakeholders. What you ship is the `Evidence` section of the initiative it produced.
- **Single-opportunity trees.** One opportunity under an outcome means you haven't done discovery yet. Even a thin tree should have 3+ opportunities so the team can choose where to attack.
- **No assumption tests.** Solutions that go from tree to author without testing any assumption. The tree without tests is just decoration.

## Cross-references

- `metrics-design` — outcome definitions follow the same leading/observable/baseline rules.
- `continuous-discovery-cadence` — the weekly rhythm that keeps the tree alive.
- `discovery-interview-design` — how to run the interviews that populate the opportunity space.
- `assumption-testing` — designing tests that resolve in days.
- `evidence-synthesis` — clustering raw research into mapped opportunities.
- `initiative` spec type — the `Evidence` section is where the tree's output lives.
- PM principle #1 (decide what's worth building) and #5 (learn from what shipped) — the OST is the operating system for both.
