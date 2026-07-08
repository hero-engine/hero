---
name: prioritization-frameworks
description: RICE, ICE, WSJF, and value-vs-effort prioritization — formulas, when each fits, and how soft inputs make the scores lie.
metadata:
  audience: prioritization-strategist, roadmap-curator, pm-delivery-lead
  purpose: framework-guidance
---
## What I do

Provide the mechanics of the four common prioritization frameworks (RICE, ICE, WSJF, value-vs-effort), name the conditions under which each is the right tool, and call out the soft inputs that quietly turn a score into theater. The output of a framework is an argument, not a verdict — this skill exists to keep it honest.

## When to use me

- Ranking a backlog of initiatives where the team needs a defensible ordering.
- Triaging fresh intake when the team needs fast, cheap "do we keep looking at this" calls.
- Sequencing a quarter or cycle when cost-of-delay matters and dependencies are real.
- Comparing 5–15 candidates visually for a stakeholder conversation.
- Anytime a stakeholder says "but framework X says we should do Y" and you need to push back without dismissing the framework.

## The frameworks

### RICE

Designed for high-volume, like-for-like comparison across a backlog. Intercom's framework.

**Formula:** `RICE = (Reach × Impact × Confidence) / Effort`

**Inputs:**
- **Reach** — number of users/events affected in a defined time window (e.g. "customers/quarter"). Concrete count when possible; an honest guess otherwise.
- **Impact** — per-user value, on a fixed scale: `3 = massive, 2 = high, 1 = medium, 0.5 = low, 0.25 = minimal`.
- **Confidence** — your faith in Reach and Impact combined: `100% = strong evidence, 80% = some evidence, 50% = a hunch`. Below 50% means do discovery, not prioritization.
- **Effort** — person-months (or person-weeks; pick one and stay consistent).

**When it fits:** You have ≥10 candidates, you want a sortable column, and you're comparing apples to apples (all initiatives, all stories, etc.).

**Where it lies:** Reach and Impact are usually guesses, but the RICE number prints to two decimals and looks like data. See "How scores lie" below.

### ICE

Sean Ellis' lightweight cousin of RICE. Designed for triage speed, not defensibility.

**Formula:** `ICE = Impact × Confidence × Ease` (each scored 1–10)

**When it fits:** Fast intake triage. "Should we even look at this further?" Twenty items in twenty minutes.

**Where it lies:** Three subjective 1–10s multiplied together produce a 1–1000 range that *feels* precise. It isn't. Use ICE to bucket (top 20% / middle / bottom), not to rank-order.

### WSJF (Weighted Shortest Job First)

SAFe's cost-of-delay framework. Designed for sequencing when finishing sooner has measurable value.

**Formula:** `WSJF = Cost of Delay / Job Size`

where `Cost of Delay = User/Business Value + Time Criticality + Risk Reduction & Opportunity Enablement`

Each input scored on a relative scale (Fibonacci: 1, 2, 3, 5, 8, 13, 21). Relative, not absolute — pick a reference item and score others against it.

**When it fits:** Sequencing already-committed work where the team understands cost-of-delay, you have a real dependency graph, and items differ meaningfully in size. Common under cycle or sprint preset for stories already past triage.

**Where it lies:** Time Criticality is the input most easily inflated. If everything is "time critical," nothing is. Score Time Criticality relative to a baseline item the team agrees is *not* time-critical.

### Value-vs-effort 2x2

The visual, conversational framework. No formula — a 2x2 grid.

**Axes:** Value (low/high) × Effort (low/high). Items drop into one of four quadrants: Quick wins (high value, low effort), Big bets (high value, high effort), Fill-ins (low value, low effort), Time sinks (low value, high effort).

**When it fits:** Stakeholder conversations. Quarterly planning where you want a picture, not a spreadsheet. ≤15 items.

**Where it lies:** "Value" and "effort" are unanchored. Without explicit definitions for the axes, every team member places items differently. Define both axes in concrete terms (e.g. "value = projected lift in activation rate; effort = person-weeks") before the exercise.

## Show the math, always

Whenever this skill produces a ranking, the output must include:

```
| Item | R | I | C | E | RICE | Notes |
|---|---|---|---|---|---|---|
| CSV export | 1200/qtr | 2 | 80% | 3 | 640 | Reach from segment-A account count |
| SSO for SMB | 400/qtr | 3 | 50% | 8 | 75 | Confidence low — need 3 customer interviews |
```

The Notes column is non-optional. It records the basis for each estimate (where did Reach come from, what's the Impact reasoning, what does Effort assume) so the team can challenge the inputs instead of arguing about the output. A score without notes is a number without an argument.

## Segment-weighting — reach is not all customers

The default `Reach = number of users` formulation treats every user equally. Often wrong. An initiative that affects 200 enterprise customers paying $100k/year is not equivalent to one affecting 200 free-tier users.

Weight reach by segment importance when segments differ materially:

```
Reach_weighted = Σ (users_in_segment × segment_weight)
```

Segment weights should come from a team decision (recorded once, reused), not be invented per-scoring. Common weights: `enterprise = 5, mid-market = 3, SMB = 1, free = 0.3`. The exact numbers matter less than consistency.

When you apply segment weighting, say so in the Notes column. A score that quietly uses weighted reach without disclosure is worse than one that doesn't weight at all.

## How scores lie

The score is only as good as its inputs. The common failure modes:

- **Reach as a guess.** "Probably about half our users" becomes "Reach = 5000" becomes a number that gets quoted in roadmap reviews. If you don't have analytics or evidence, score Confidence ≤50% and flag the gap as "blocked on instrumentation."
- **Impact as a guess.** A 3 vs 2 distinction is rarely defensible without research. When Impact comes from "the team's intuition," say so in Notes.
- **Confidence as a polite filler.** Many teams default everything to 80% because higher numbers feel optimistic. Force a discipline: items with no customer evidence cannot exceed 50%.
- **Effort sandbagging.** PMs estimate Effort. Engineering didn't see the number. Always have engineering size Effort before the score goes into a roadmap review — otherwise you're prioritizing on a fiction.
- **Confidence-pumping pet projects.** Watch for items where Confidence climbs as the score needs to win. If the same input keeps moving to defend a ranking, the framework is being abused.

## When the framework says X but you think Y

The framework is a sanity check, not the decision-maker. Strategic context, market timing, regulatory deadlines, founder conviction, and customer relationships routinely override scores — that's fine, *as long as the override is logged*.

How to surface a conflict without overriding the team:

1. **Show the score result first.** "RICE ranks this 7th out of 15."
2. **Name the override factor explicitly.** "We're proposing rank 2 because the customer churn risk on Acme is not captured in Reach."
3. **Record the override in the initiative's `rationale` field.** "Promoted above RICE rank due to Q2 retention risk; revisit if Acme renewal closes."
4. **Don't hide the score.** Showing the framework result alongside the override is more credible than dropping the framework when it's inconvenient.

Principle: the team owns the call. The framework owns the audit trail.

## Anti-patterns

- **RICE-theater.** Inputs are guesses; output is quoted as data. Always show Notes and Confidence.
- **Single-framework worship.** Picking RICE because "we use RICE" when the situation calls for value-vs-effort (or vice versa). The framework should fit the decision, not the other way around.
- **Confidence creep.** Confidence values that mysteriously rise when the team likes an item. Anchor Confidence to evidence, not vibes.
- **Effort-by-PM.** Effort estimated without engineering input. Worthless. Always co-sign Effort with engineering before scoring matters.
- **Equal-weight reach.** Treating 1000 free-tier users as equivalent to 100 enterprise customers when the segment economics are 10x different.
- **Re-scoring to win.** Adjusting inputs after the ranking comes out to make a favored item win. If you do this, log it and explain why — otherwise the framework is just decoration.
- **Framework-as-decision.** "RICE says do this" without strategic context. The framework is one input. The team owns the call.

## Cross-references

- `metrics-design` — Impact often refers to a metric movement; metric definitions should match.
- `opportunity-solution-trees-torres` — items emerging from an OST should carry inherited Confidence from the assumption tests that produced them.
- `initiative` spec type — RICE/ICE/WSJF scores live in `rice_score`, `ice_score`, `wsjf_score` frontmatter fields; `rationale` records framework overrides.
- PM principle #3 (make tradeoffs visible) — every override deserves a recorded reason.
