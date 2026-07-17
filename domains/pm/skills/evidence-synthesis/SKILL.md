---
name: evidence-synthesis
description: Turn raw signals — customer quotes, interview synthesis, usage data, NPS comments — into a roadmap evidence trail that survives the "how do you know?" challenge.
metadata:
  audience: intake-triager, discovery-researcher, product-strategist
  purpose: discovery-curation
---

## What I do

Provide the rules for assembling an Evidence section on a `initiative`, `prd`, or `intake` Investigation. Evidence is what makes a bet defensible. The synthesis must aggregate across many intakes without losing source attribution, preserve customer quotes verbatim, distinguish evidence from interpretation, and surface counter-evidence honestly.

## When to use me

Load this skill when:

- promoting an intake to an initiative — the rolled-up evidence becomes the initiative's Evidence section (`intake-triager`)
- writing a discovery synthesis after interviews or research sessions (`discovery-researcher`)
- adding a competitive observation to an existing initiative (competitive-analyst is P1)
- attaching usage-data analysis to a PRD or initiative (metrics-analyst is P1)
- a `product-strategist` is framing a bet and needs to defend "why this and not that"
- `pm-investigator` is populating an intake Investigation section

## The evidence pyramid

In rough order of trust:

1. **Verbatim customer quote** — the customer's own words, captured at the time of the original signal. Highest trust. Defensible against challenge because the words came from the user, not from the team telling itself a story.
2. **Customer interview synthesis** — themes pulled from multiple recorded interviews. Trust is high when the synthesis preserves direct quotes; collapses fast when the interview synthesis becomes "the team's reading of what the customer meant."
3. **Usage-data signal** — concrete behavior in the product. High trust for measuring *what users do*; weak signal for *why* they do it.
4. **NPS comment** — open-ended feedback at scale. Useful for theme detection across the base; weak per-comment trust (NPS biases toward extremes).
5. **Internal hunch** — someone on the team thinks users would want this. Valid as a question to investigate, not as evidence supporting a bet.

**The synthesis rule:** every Evidence section names *which tier* its evidence comes from. "Three enterprise customers asked verbatim" carries different weight than "the sales team thinks prospects would want it." Both can appear; conflating them is dishonest.

## Writing an Evidence section that survives challenge

The Evidence section on a `initiative` must survive the standard challenge: **"how do you know?"** A reviewer should be able to:

1. Trace each evidence statement back to its source (intake, ticket, interview, dashboard).
2. See the customer's own words (or the interview quote, or the metric chart) without leaving the spec.
3. Distinguish facts from interpretation.

The shape that works:

```
## Evidence

**Verbatim customer signals** (3 enterprise, 2 SMB):
- "We can't roll out to legal without SAML." — acme-corp, Q1 expansion call (intake:saml-acme-q1)
- "Login complexity is the biggest blocker." — beta-llc, support ticket (intake:saml-beta-456)
- [...]

**Discovery synthesis** (5 interviews, 2026-04):
- 4 of 5 enterprise prospects cite SSO as a procurement gate.
- Synthesis: enterprise procurement requires SAML, not just SSO. (discovery:enterprise-auth-apr2026)

**Usage data**:
- 18% of enterprise accounts churned in last 6mo cite "authentication" in exit survey. (metric:enterprise-churn-q1)

**Counter-evidence**:
- 0 prosumer customers requested SAML. The bet is segment-specific.
- One enterprise customer (delta-co) explicitly prefers OIDC-only. Solution must accommodate.
```

Notice:

- Each evidence statement carries its source attribution as a link.
- Quotes are verbatim, not paraphrased.
- The synthesis is labeled "synthesis" — distinct from quotes.
- Counter-evidence has its own subsection. The bet is stronger when the team has examined its negation honestly.

## Aggregating across intakes without losing attribution

When an initiative rolls up evidence from N linked intakes, the temptation is to summarize: "12 enterprise customers asked for SAML." This loses the trust signal.

The rule: **summarize the count, preserve the sources.**

```
**Enterprise SAML requests**: 12 intakes
- acme-corp (intake:saml-acme-q1) — verbatim "can't roll out without SAML"
- beta-llc (intake:saml-beta-456) — verbatim "biggest blocker"
- [10 more, expandable]
```

The aggregate count makes the pattern visible; the per-source list makes the trust verifiable. `roadmap-curator` should generate the rollup automatically by walking the `linked_intake` edges.

## Quoting the customer's words

Verbatim quotes are non-negotiable. The `source_quote` field on each intake flows through to the initiative Evidence section unchanged. This preserves the trust signal that `intake-classification` calls out.

When the original signal isn't a quote (a support ticket summary, a sales note), label it as "paraphrased" and link the source. Never invent a quote.

## Distinguishing evidence from interpretation

Two clauses, distinguished:

- **Evidence**: "12 enterprise customers cited SAML in the last quarter."
- **Interpretation**: "Therefore enterprise expansion is gated on SAML."

The interpretation may be right, but it's the team's reading of the evidence, not the evidence itself. Keep them in different sentences. A reviewer who disagrees with the interpretation can still credit the evidence.

In the Evidence section, lead with facts. Save interpretation for the **Bet** or **Tradeoffs** section, where it belongs.

## Surfacing counter-evidence honestly

An initiative Evidence section with no counter-evidence subsection is suspicious. Almost every bet has data points that argue against it. Naming them does three things:

1. Forces the team to confront the strongest argument against the bet.
2. Signals to reviewers that the analysis was honest, not selective.
3. Surfaces the boundary conditions — "the bet doesn't hold for segment X" is critical context for downstream design.

If you genuinely cannot find counter-evidence, write that: "Searched 30 intakes and 5 interview transcripts; no signal contradicting the bet." That's a finding too.

## The "five whys" pattern when intake is vague

When an intake Investigation calls "the underlying need is unclear," apply five whys:

1. Why did the customer ask for X?
2. Why does X matter to them in their workflow?
3. Why is the current workflow inadequate?
4. Why hasn't a workaround sufficed?
5. Why now (vs. six months ago)?

Each answer becomes a quote-or-question pair. Quotes go in Evidence; unanswered questions become the input to `discovery-researcher`'s assumption tests.

Five whys is a pattern, not a script. Stop when you reach a defensible root or when further drilling requires new data.

## The Torres synthesize-then-compare discipline

Synthesis is a *second read*, not a replacement for the PM's own — this is `pm-agent-doctrine` doctrine 3 stated as synthesis mechanics. The distrusted pattern is "the agent read the interviews and here's the answer": it outsources judgment, and when it's subtly wrong (an over-weighted outlier, a fabricated theme), no one catches it because no one did the parallel read.

The **synthesize-then-compare** loop inverts that:

1. **Synthesize independently.** The agent (or the PM) pulls themes from the raw signal first, each theme carrying the verbatim quotes behind it.
2. **Read independently.** The PM forms their own reading of the same source, without anchoring on the agent's.
3. **Reconcile the two passes.** Diff them. Where they agree, confidence rises. Where they diverge, that gap is the finding — the agent may have over-weighted a vivid quote, or the PM may have skimmed a segment. Neither pass is authoritative until reconciled.

Present the synthesis as "here's what I see — does it match what you see?", name your confidence per theme, and name where you're unsure. A synthesized theme is "proposed synthesis pending PM review," never "the finding," until the human has reconciled it. This protects the one thing the pack can't regenerate: the PM's ownership of their own read.

## Surfacing the outlier, not just the modal answer

A synthesis that reports only the confirming pattern is selling a conclusion. **The outlier is often the high-value finding** — the churned user who wanted the opposite, the one segment where the pattern broke, the interview that contradicted the other four. Report it; do not sand it into the modal narrative to make the story tidy.

The rule: **the modal answer and the signal that broke it both get a line.** "4 of 5 enterprise prospects gate on SSO" is the pattern; "the 5th (delta-co) explicitly prefers OIDC-only and would churn on a SAML-first design" is the outlier that reshapes the solution. Dropping the 5th to keep "5 of 5" clean is the exact dishonesty that makes synthesis distrusted. An outlier belongs in the Evidence section's counter-evidence subsection, linked to its verbatim, not smoothed away.

## Verbatim attribution survives reconciliation

The synthesize-then-compare loop only works if the **theme → verbatim link survives it.** Every theme the agent proposes carries the exact quotes that produced it, so the PM can click from "users struggle to find export" to the three quotes behind it and judge whether the read is fair. Reconciliation *diffs the reads* — that diff is only possible when both passes trace to the same source words. A theme that loses its verbatim during synthesis becomes an unfalsifiable assertion: the PM can't check it, so they can't reconcile it, so the compare step collapses back into replace. Preserve the attribution through every step; a theme with no verbatim behind it is not a finding, it's an opinion.

## Anti-patterns

- **"100 customers asked for this" with no underlying data.** Aggregates without sources are not evidence; they're claims.
- **Cherry-picking one customer quote into a trend.** One quote is one quote. A trend requires recurrence across distinct sources.
- **Synthesis that strips source attribution.** The trust signal is the source. Strip it and the synthesis becomes opinion.
- **Synthesis-as-replacement.** Handing over "the answer" with no parallel read to reconcile against. Synthesize-then-compare, never replace (doctrine 3).
- **Sanding off the outlier.** Dropping the signal that broke the pattern to keep the modal narrative clean. The outlier is often the finding, not noise.
- **Conflating evidence with interpretation in the same sentence.** Lead with facts; interpret separately.
- **Evidence sections with no counter-evidence subsection.** Either the analysis was selective, or the team didn't look hard.
- **Quoting paraphrased text as if it were a direct quote.** Either preserve the verbatim or label as paraphrased.
- **Treating sales restatements as customer evidence.** Sales speaks for the prospect, not as the prospect. Trust accordingly (per `intake-classification`).
- **Using NPS scores as the evidence — ignoring the comment text.** The number is weak signal; the comment text is where the trust lives.
