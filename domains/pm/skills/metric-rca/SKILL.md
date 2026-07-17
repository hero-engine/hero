---
name: metric-rca
description: "Why did the metric move" — the disciplined method for localizing a metric change before naming a cause. Metric-tree decomposition, a five-class drift taxonomy, and a causality-before-asserting guard that treats correlation as a hypothesis, not a cause.
metadata:
  audience: metrics-analyst
  purpose: framework-guidance
---

## What I do

Answer the question that gets asked in every metrics review — **"why did the metric move?"** — without free-associating a cause. The distrusted version is "conversion dropped, probably the new checkout" asserted from a hunch. The trusted version localizes the move to a component of a metric tree, classifies it against a known drift taxonomy, and names the cut that would confirm or kill each candidate cause *before* asserting any of them. RCA is a reasoning method, not a query engine: it produces ranked, grounded, disconfirmable hypotheses, not a single confident verdict.

## When to use me

- "Why did <metric> drop / spike last week."
- "Run RCA on the funnel" / "the activation number moved, what happened."
- A principle-#5 retrospective where a shipped bet's metric moved and the team needs to know whether the bet caused it.
- Any time a headline number changed and the reflex is to name a cause from memory — that reflex is exactly what this skill interrupts.

## Step 1 — Metric-tree decomposition

Never reason about a top-line metric directly. **Decompose** it into the components that multiply or add up to it, so a move can be *localized to a component before a cause is named*. A move you can't localize is a move you can't explain.

Common decompositions:

- `Conversions = Sessions × Conversion-rate` (and `Conversion-rate = Σ step-through rates` down the funnel)
- `Revenue = Users × ARPU` (and `ARPU = Paid-rate × Price × Retention`)
- `Activation = Signups × Activation-rate`
- `Engagement = Active-users × Actions-per-user`

Build the tree, then attribute the top-line move to the component(s) that moved. "Revenue fell 8%" is not a finding; "revenue fell 8% and the whole move is in ARPU while user count held" *is* — it points the rest of the investigation at ARPU and rules out an acquisition story. Decompose until the move sits on a leaf.

### Decomposition table

Localize the move with a table before classifying it:

```markdown
| Component | Prior | Current | Δ | Carries the move? |
|---|---|---|---|---|
| Sessions | 120k | 121k | +0.8% | no |
| Conversion-rate | 4.2% | 3.6% | −14% | yes |
| … | | | | |
```

The "carries the move" column is what turns a top-line panic into a located question.

## Step 2 — Drift taxonomy

Once the move is localized to a component, classify *what kind* of drift it is. Every metric move is one (or a combination) of these **five drift classes** — naming the class narrows the confirming cut:

1. **Component** — a sub-metric genuinely moved (a funnel step's rate fell, a price changed). The move is real and localized within the tree. Confirming cut: the decomposition table already shows which leaf.
2. **Temporal** — seasonality, day-of-week, holiday, or an underlying trend, not a discrete event. A Monday-vs-Sunday comparison or a month-over-month against last year's same week is the tell. Confirming cut: overlay the prior-period / prior-year curve before believing anything changed at all.
3. **Influence** — a **mix / segment shift**: the overall metric moved because the *composition* of the population changed, not because any segment did. This is the **Simpson's-paradox** class — every segment can be flat or up while the blended number falls, because a lower-performing segment grew as a share. Confirming cut: hold the mix constant (segment the metric and weight by prior-period shares).
4. **Dimension** — a slice **appeared or disappeared**: a new geo launched, a platform (a new app version, a browser) entered or dropped out, a traffic source turned on or off. The population gained or lost a dimension value. Confirming cut: group-by the suspect dimension and look for a slice that is new, missing, or newly dominant.
5. **Event-shock** — a discrete event: a launch, an outage, a pricing change, a marketing push, a tracking/logging bug, or an external event (a competitor move, a news cycle). The move aligns in *time* with a known event. Confirming cut: line the change up against the deploy log / incident timeline / release calendar and check the timestamp alignment.

A single move can be more than one class (an event-shock that also shifted the mix). Classify all that apply; each class carries its own confirming cut.

## Step 3 — Causality before asserting

**Correlation is a hypothesis, not a cause.** A component moved, a class fits, an event lines up in time — none of that is a cause until the confirming cut is run. The guard, stated as a rule:

> Every candidate cause names the specific cut, segment, or time-window that would confirm or kill it — and that cut is run (or explicitly flagged as the next step) *before* the cause is asserted.

"Checkout redesign shipped Tuesday and conversion fell Tuesday" is a hypothesis; the confirming cut is "conversion for users who *saw* the redesign vs. those who didn't, same window." If the drop is equal in both arms, the redesign is exonerated and the real cause is elsewhere. Ranked hypotheses, never a single asserted cause without its cut — and each hypothesis grounded in a corpus number (doctrine 1: the actual deploy log, the actual segment table, the actual incident record), not model memory.

## Produces — the `## Metric RCA` section

The analyst writes an RCA section carrying:

- the **decomposition table** localizing the move to its component(s);
- the **drift class(es)** the move is classified as, with the reason;
- **ranked candidate causes**, each with the **confirming cut** that would validate or kill it and the corpus number it cites;
- an explicit statement of which cuts have been run vs. which are the recommended next step.

Suggest-don't-decide (doctrine 2): the RCA names the likely causes and the data that would confirm each. It does **not** assert a single cause as settled fact without the confirming cut in hand — an unconfirmed top hypothesis is labelled as such, not laundered into "the reason."

## Anti-patterns

- **Naming a cause from a hunch.** "Probably the new checkout" with no decomposition and no cut. The signature RCA failure.
- **Skipping decomposition.** Reasoning about the top-line number directly, so the move is never localized and every cause is equally (un)supported.
- **Confusing correlation with cause.** An event that lines up in time asserted as the cause without the saw-it-vs-didn't cut. Temporal coincidence is a hypothesis.
- **Ignoring the mix (Simpson's paradox).** Reporting a blended drop as a real decline when every segment is flat and only the composition changed — the influence class, missed.
- **One cause, no ranking.** Asserting a single explanation when the evidence supports several. Rank them and name each one's confirming cut.
- **Ungrounded numbers.** Citing "traffic shifted" without the actual segment table, or "the deploy caused it" without the deploy log. Grounding theater fails doctrine 1.

## Cross-references

- `metrics-design` — a metric worth doing RCA on is one defined to these standards (observable, baseline-anchored, segmentable).
- `evidence-synthesis` — the mechanics of grounding candidate causes in corpus sources and preserving attribution.
- `pm-agent-doctrine` — correlation-is-a-hypothesis is doctrine 1 (ground the claim) and doctrine 2 (suggest the cause, don't decide it) applied to metric movement.
- `experiment-design` — when RCA can't resolve cause from observational cuts, the answer is a pre-registered experiment, not a stronger assertion.
</content>
