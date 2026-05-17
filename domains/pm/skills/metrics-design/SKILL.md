---
name: metrics-design
description: Define success metrics for product work — leading vs lagging, observable, with named baseline and target before commit.
compatibility: opencode
metadata:
  audience: metrics-analyst, prd-author, roadmap-curator
  purpose: framework-guidance
---
## What I do

Provide rules for defining product success metrics that are actually measurable, actually leading, and actually tied to outcomes — not vanity counters dressed up as KPIs. Every metric this skill produces names a baseline before it proposes a target, and every metric is co-signed by engineering as observable in the current system.

## When to use me

- Authoring the `Goals & Success Metrics` section of a PRD.
- Defining the outcome at the top of an Opportunity Solution Tree.
- Setting `current → target` on an initiative before promotion to `committed`.
- Running a post-ship retrospective on a delivered initiative (principle #5 — learn from what shipped).
- Pushing back on a stakeholder request to "track engagement" without defining what that means.

## What makes a good metric

A metric earns a place on a spec when it passes all five tests:

1. **Observable** — the data exists in production today, or engineering has explicitly agreed to add the instrumentation as part of the work.
2. **Leading** — it moves within days or weeks of the change shipping, not months. Lagging metrics (revenue, retention 90d, NPS) confirm long-term but can't validate a single bet on a cycle timescale.
3. **Outcome-tied** — it measures user behavior or business state, not feature usage. "Activation rate" is an outcome; "clicks on the activation button" is feature usage.
4. **Baseline-anchored** — you know the current value. Without a baseline, a target is a wish.
5. **Targeted with rationale** — the target value has a defensible reason (benchmark, segment uplift assumption, parity with comparable surface).

If a metric fails any of these, it's a candidate, not a metric. Capture it as "instrumentation gap" or "needs baseline measurement" and treat establishing it as work.

## Leading vs lagging

Most teams default to lagging metrics because they're familiar (revenue, retention, NPS). Lagging metrics are still important — they're how the business measures itself — but they're poor signals for whether a specific bet worked.

| | Leading | Lagging |
|---|---|---|
| Timescale | Days to weeks | Months to quarters |
| Use for | Validating a bet, steering | Quarterly/annual goals |
| Risk | Can be a misleading proxy | Confounded by everything else shipping |
| Example | "% of new accounts reaching activation event in 7 days" | "90-day retention rate" |

An initiative should usually carry **both**: a leading metric to validate the bet within the cycle, and a lagging metric naming the eventual outcome the bet feeds. The leading metric is what the team watches; the lagging metric is what the business judges.

## Observable means engineering has seen it

A metric that engineering hasn't reviewed is almost certainly unmeasurable. Common failures:

- "Time to first value" — fine, but what event marks "first value"? Not defined in any product spec, not emitted by any analytics call.
- "Engagement per session" — what's a session? When does it end? Different teams answer differently.
- "Active users" — DAU? WAU? What counts as activity? "Logged in" is rarely what you mean.

**Rule:** every metric in a PRD or initiative must be reviewable by engineering before the spec promotes to `committed`. Engineering's check: can we compute this from the data we currently have, or do we need to ship instrumentation as part of this work? If the latter, the instrumentation is in-scope and named in the spec.

This is the "signed-up-by-engineering" gate. A metric without it is a guess.

## Baseline before target

Never propose a target without a baseline. The format on every spec:

```
| Metric | Current | Target | Window | Source |
|---|---|---|---|---|
| 7-day trial activation rate | 22% | 35% | Q3 2026 | Mixpanel: event `activation_complete` within 7d of signup |
| Trial → paid conversion (lagging) | 8% | 11% | Q4 2026 | Stripe: paid_subscription within 60d of trial start |
| Support tickets per activated account | 0.40/mo | 0.25/mo | end of Q3 | Zendesk: tickets joined to user_id |
```

The Source column is required. It says exactly how the metric is computed, from which data system, with which definition. This is what makes the metric verifiable months later when the retrospective runs.

If you don't know the current value, write `unknown — needs baseline measurement` and treat establishing it as a precursor work item. A spec with `unknown → 35%` is not ready for `committed`.

## Vanity counters and the proxy problem

Vanity counters look like metrics but don't carry decision weight. Common ones:

- **DAU / MAU as a top-line target.** Useful for trend monitoring; rarely the right metric for a single feature. Almost any feature touches DAU a little — almost none move it noticeably enough to attribute.
- **Pageviews / clicks / impressions.** Activity counts without quality dimensions. A feature that doubles clicks while halving conversion is worse, not better.
- **Number of users who used the feature.** Adoption is necessary but not sufficient. A feature 80% of users tried once and abandoned is failing, not winning.
- **"Engagement."** Almost always undefined. Force the team to name the specific behavior they want to see more of.

### Proxy metrics

A proxy metric stands in for an outcome that's hard to measure directly. Proxies are necessary but lie under stress:

- "Time on page" as a proxy for "value received" — broken when users leave the tab open or get stuck.
- "Search abandonment rate" as a proxy for "search quality" — broken when users abandon because they found the answer at-a-glance in the results.
- "Reduced support tickets" as a proxy for "feature works" — broken when users give up instead of asking for help.

When you use a proxy, **name it as a proxy** in the spec and note what would invalidate it. The proxy is allowed; the failure to acknowledge it is not.

```
| Metric | Note |
|---|---|
| Search abandonment rate | Proxy for search quality. Invalidates if abandonment drops because users stop searching entirely (check search volume separately). |
```

## Per-metric vs aggregate

A common temptation: combine several metrics into a single "north star." Resist this for single bets — composites hide which input moved and which didn't.

Composites have a place at the org level (`OMTM`, `north star metric`), but at the initiative / PRD level, prefer 1–3 independent metrics with their own current/target. You want to know which one moved.

## Segmentation

A metric averaged across all users often hides the bet. If the bet is "activation will improve for SMB segment," reporting "activation overall went from 22% to 24%" tells you almost nothing.

Define the segment in the spec when the bet is segment-specific:

```
| Metric | Segment | Current | Target | Window |
|---|---|---|---|---|
| 7-day activation rate | SMB (self-serve trials) | 22% | 35% | Q3 2026 |
| 7-day activation rate | All trials | 31% | (monitor — no regression) | Q3 2026 |
```

The second row is a guardrail: "we're targeting SMB, but we don't want enterprise activation to drop as a side effect." Guardrail metrics are good practice on any bet that intentionally optimizes for a subset.

## Tying back to principle #5 — learn from what shipped

The PM mission's fifth principle is *learn from what shipped*. Metrics design is the operating mechanism for that principle. The discipline:

1. **Set the metric, baseline, and target before commit.** Recorded in the spec's frontmatter or `Goals & Success Metrics` section.
2. **Re-measure at a fixed window after ship.** Default: 30 days post-launch for leading; 90 days for lagging.
3. **Record the actual vs target on the spec.** The `retrospective_note` field on a `story` or the `Shipped vs expected` section on a `initiative`.
4. **Feed the result back into prioritization.** A bet that missed its target reduces Confidence on similar future bets; a bet that beat its target raises it.

A team that ships without measuring is a team that prioritizes from intuition forever. The metric is the feedback loop.

## Anti-patterns

- **Targets without baselines.** `"Target: 35%"` with no current value. The number means nothing.
- **Unmeasurable metrics.** Engineering hasn't seen the definition, no instrumentation exists, no data system can produce the value. The metric is theater.
- **Vanity-counter targets.** DAU, pageviews, clicks promoted to KPIs without quality dimensions.
- **Lagging-only goals.** A PRD where every metric is 90-day retention. The cycle ends before any data arrives — you can't steer.
- **Composite-only reporting.** A single "engagement score" combining 5 inputs. Hides which input moved.
- **Targets without rationale.** "Target: 50%" with no benchmark, no segment analysis, no comparable surface. The number is invented.
- **Set-and-forget.** Metrics defined at commit, never re-measured at ship. The feedback loop is broken.
- **"Engagement" with no definition.** The single most common offender. Force the team to name the behavior.
- **Proxy without disclosure.** A proxy metric reported as if it were the outcome. The reader can't tell what's actually being measured.

## Cross-references

- `opportunity-solution-trees-torres` — the outcome at the top of an OST is a metric defined by these rules.
- `prioritization-frameworks` — Impact in RICE is shorthand for "expected movement on a defined metric"; without metrics design, Impact is a guess.
- `continuous-discovery-cadence` — discovery surfaces which opportunities matter; metrics confirm whether the bet on an opportunity worked.
- `prd-structure` — the `Goals & Success Metrics` section follows the table format above.
- `initiative` spec type — frontmatter carries the metric definitions; `Bet` references them.
- PM principle #5 (learn from what shipped) — metrics design is its operating mechanism.
- PM mission anti-pattern "a roadmap that lies" — applies equally to "a PRD whose metrics are unmeasurable."
