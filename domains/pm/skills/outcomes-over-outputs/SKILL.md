---
name: outcomes-over-outputs
description: The spine framework for reading a roadmap, PRD, or bet honestly — the outcome ladder (input → output → outcome → impact), the outcome-vs-output-vs-input distinction, and the ~60/30/10 framing ratio that keeps a roadmap from being a build list wearing a strategy costume.
metadata:
  audience: product-strategist, pm-reviewer, and the deferred spine-loaders authored in later waves (metrics-analyst, portfolio-curator, roadmap-reviewer, stakeholder-communicator)
  purpose: framework-guidance
---

## What I do

Carry the single most load-bearing framing discipline in product management: **the thing we bet on is an outcome that changes, not an output we ship.** The external best-practice scan names this as the #1 framework the trusted PM tools ground in and the shallow ones ignore. It's the frame behind Marty Cagan / SVPG's *Inspired* and *Empowered*, and it's the difference between a roadmap leadership can trust and a roadmap that is a build queue with quarterly labels.

I give agents three tools: the **outcome ladder** for placing any statement at the right altitude, the **outcome-vs-output-vs-input distinction** for catching the most common framing failure, and the **~60/30/10 ratio** for auditing whether a whole roadmap or PRD is framed at the right level. When `product-strategist` frames a bet or `pm-reviewer` reviews an initiative or PRD, this is the lens.

## When to use me

- framing an initiative or bet (`product-strategist`) — before writing the Outcome/Bet sections
- reviewing an initiative, PRD, or roadmap (`pm-reviewer`) — as the first-pass framing check
- auditing a whole roadmap for drift toward output-framing (the deferred `roadmap-reviewer` drift critic)
- translating a bet for an exec audience (the deferred `stakeholder-communicator`) — execs want the outcome, not the output
- any time a stakeholder says "let's just build X" and you need to surface what outcome X is supposed to move

## The outcome ladder

Every product statement sits on a ladder. Naming the rung is how you catch framing that's pitched too low.

```
IMPACT     ← the business/mission result the outcome ladders to
  ↑          (revenue, retention, market position, mission metric)
OUTCOME    ← the change in user/customer BEHAVIOR we bet on
  ↑          (measurable, the unit of a real bet)
OUTPUT     ← the thing we ship
  ↑          (a feature, a flow, a release — necessary, not the bet)
INPUT      ← the work/effort/resource we spend
             (person-weeks, stories closed, tickets, activity)
```

Read the ladder bottom-up as *cause* and top-down as *why*:

- **Input** — what we spend. "Two engineers for a cycle." "14 stories." Necessary to track, useless as a goal — measuring inputs rewards motion, not results.
- **Output** — what we ship. "One-click CSV export." Real and shippable, but shipping it is not winning. Plenty of shipped outputs move nothing.
- **Outcome** — the **behavior change** we're actually betting on. "New accounts export within 30 minutes of signup instead of filing a support ticket." Measurable, and the honest unit of a bet.
- **Impact** — the business/mission result the outcome ladders up to. "Ops escalations down 40%, freeing ~6 ops-hours/week." The reason the outcome matters.

The discipline: **a bet is stated at the outcome rung, justified by the impact rung, delivered via the output rung, and costed at the input rung.** A "bet" stated at the output rung ("we're betting on CSV export") has skipped the question that matters — *what behavior do we expect to change, and how will we know?*

## Outcome vs output vs input — the distinction that catches the failure

The single most common framing failure: a statement framed as an **output** (a thing we'd build) or an **input** (effort we'd spend) when it should be framed as an **outcome** (a behavior that would change).

Run any roadmap-item, goal, or bet through this pass/fail table:

| Statement | Rung | Verdict | Reframed as outcome |
|---|---|---|---|
| "Build CSV export." | output | ❌ | "Cut manual-data-pull support escalations 40%." |
| "Ship 14 stories this quarter." | input | ❌ | *(inputs are never a goal — name the behavior the stories should move)* |
| "Redesign onboarding." | output | ❌ | "Lift D7 retention for new signups from 22% to 30%." |
| "Increase engagement." | vanity | ❌ | "Raise weekly active rate in the SMB segment from 41% to 50%." |
| "Add SSO." | output | ❌ | "Unblock enterprise deals stalled on SSO/MFA requirements." |
| "Reduce time-to-first-export to under 30 min." | outcome | ✅ | *(already an outcome — measurable behavior change)* |
| "Lift SMB quarterly retention 78% → 84%." | outcome | ✅ | *(already an outcome)* |

Two tells that a statement is misframed:

1. **It can only be *shipped*, not *measured*.** "Build X" has a done-state (shipped) but no success-state. An outcome has a success-state you can observe after shipping. Cagan's point (principle #5, learn from what shipped) requires outcomes — you can't learn from an output whose only metric is "it exists."
2. **It forecloses engineering's contribution.** An output-framed item tells engineering *what to build*; an outcome-framed item tells them *what to move* and leaves them room to propose a better build. Output framing throws away the team's best ideas.

The output is always *implicit* in a good outcome — the team will build *something* to move it — but the bet rides on the outcome, which survives scope changes the output doesn't. If discovery reshapes the build, an outcome-framed item still makes sense; an output-framed one is now betting on the wrong thing.

## The ~60/30/10 ratio

Use the **60/30/10** ratio to audit a whole roadmap, PRD Goals section, or planning doc — not just one item. A healthy product plan is framed roughly:

- **~60% outcomes** — behavior/impact changes we're betting on. The majority of what a roadmap communicates should be *what will be different*, not what will be built.
- **~30% outputs** — the concrete things we'll ship to move those outcomes. Outputs belong on the plan — they're how the work becomes real — but they should be the minority, hung *under* an outcome.
- **~10% inputs** — capacity, effort, resourcing notes. Present for planning realism, never the headline.

The numbers are a heuristic, not a gate — the point is the *shape*. When you review a roadmap and find it's **60% outputs** (a list of features with quarters attached) and near-zero outcomes, that's the drift signal: the roadmap has become a build queue, and no one can tell whether shipping it will change anything. When you review a Goals section and every "goal" is a feature name, apply the reframe table item by item.

**How to apply it in a review:** tally each line by rung. If outputs dominate, the finding is "this roadmap is output-framed — for the top items, what behavior are we betting will change, and how will we know?" Don't demand every output be reframed (some maintenance and compliance work is legitimately output-shaped) — demand that the *bets* ride on outcomes and that outputs hang under an outcome that justifies them.

## A worked roadmap audit

A Q3 roadmap lands for review with six items:

> 1. Ship CSV export · 2. Redesign settings page · 3. Add SSO · 4. Migrate to new billing provider · 5. Build mobile notifications · 6. Refactor the export service

Tally by rung: all six are **outputs**. Zero outcomes. Input/output/outcome ratio ≈ 0/100/0 — the exact drift the 60/30/10 heuristic exists to catch. This roadmap is a build queue; nobody reading it can tell whether shipping it changes anything.

The review finding isn't "reframe all six" — it's targeted:

- Items 1, 3, 5 are **bets** and must ride on outcomes: *"CSV export → cut manual-pull escalations 40%"; "SSO → unblock the two enterprise deals stalled in security review"; "mobile notifications → lift D7 re-engagement for the 31% who bounce after first session."* Now each has a success-state, not just a done-state.
- Item 4 (billing migration) and item 6 (export refactor) are legitimately output/infra-shaped — but they should **hang under the outcome they enable**, not float as peer "bets." Billing migration enables the pricing changes that serve a growth outcome; the refactor de-risks item 1. Framed that way, their priority becomes arguable on outcome grounds.
- Item 2 ("redesign settings") is the dangerous one: an output with *no* obvious outcome. The review question is blunt — *what behavior are we betting the redesign changes, and how will we know?* If there's no answer, it's motion, not a bet.

Reframed, the roadmap reads ~60% outcomes with outputs hung underneath — and every top item is now challengeable on whether it'll actually move what it claims.

## Leading vs lagging outcomes

An outcome is only useful if it feeds a learning loop, which means it must be *measurable within a reasonable window*. Distinguish:

- **Lagging outcome** — the real result, measurable only after time (quarterly retention, annual revenue). The honest target, but too slow to steer by alone.
- **Leading outcome** — an earlier behavior change that predicts the lagging one (first-week activation, feature-adoption rate, time-to-first-value). Fast enough to learn from inside a cycle.

Frame the bet on the lagging outcome, but instrument a **leading** indicator so you get signal before the quarter ends. "Lift quarterly retention (lagging) — leading indicator: D7 activation, measured weekly." An outcome whose *only* measure lands a year out isn't wrong, but it's a long-horizon bet with no near-term learning — say so explicitly rather than pretending it has a fast feedback loop. (See `metrics-design` for making the indicator observable.)

## Anti-patterns

- **Output-framed bets.** "We're betting on the redesign." A redesign is an output; the bet is the retention lift you expect it to produce. Refuse the framing until the outcome is named.
- **Vanity outputs quoted as outcomes.** "We shipped 40 features" / "we increased engagement." Shipped-count is an input-in-disguise; undefined "engagement" is a vanity counter. Neither is an outcome — an outcome names *which* behavior moved *how much* against *what baseline*.
- **Input goals.** Velocity, stories-closed, tickets-resolved as objectives. These reward motion. Track them as capacity signals; never set them as the goal.
- **Ladder-skipping.** Jumping from impact ("grow revenue") straight to output ("build billing v2") with no outcome rung. The missing rung — *what customer behavior changes* — is exactly where the bet should be interrogated.
- **Outcome metrics measurable only a year out.** An "outcome" whose only measure lands 12 months later can't feed a learning loop. Reframe to a leading indicator or accept it's a long-horizon bet with no near-term signal.
- **60% outputs, 0% outcomes.** The roadmap-as-build-queue. The drift the ratio exists to catch.

## Cross-references

- `roadmap-framing` — where the outcome discipline becomes required Bet/Evidence/Tradeoffs sections; its output-vs-outcome table is this skill applied to initiative authoring.
- `metrics-design` — an outcome needs an observable metric with a baseline and target; this is where you make the outcome measurable.
- `pm-agent-doctrine` — corpus-grounding is the other half: an outcome bet must cite the evidence that the behavior is worth moving.
- `opportunity-solution-trees-torres` — the OST hangs opportunities and solutions *under* an outcome; same altitude discipline, discovery-shaped.
- Prior art: Marty Cagan / SVPG (*Inspired*, *Empowered*); Josh Seiden, *Outcomes Over Output*; ProdPad's Now-Next-Later outcome framing.
