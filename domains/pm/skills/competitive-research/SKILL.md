---
name: competitive-research
description: Retrieval-augmented competitive teardown — never model-memory. Ground every claim about what a competitor ships in a retrieved, dated, linkable source, separate parity from differentiation, and treat the model's training-data recollection as a lead to verify, not a fact to state.
metadata:
  audience: competitive-analyst (Wave-2)
  purpose: framework-guidance
---

## What I do

Supply the discipline for competitive analysis that a PM can actually trust: **retrieval-augmented, never model-memory.** This is the highest-stakes place for the corpus-grounding doctrine, because a model's training-data recollection of a competitor's feature set is stale by construction and confidently wrong often enough to be dangerous — a fabricated "Competitor X already has this" can kill a bet or greenlight a redundant one. The rule is absolute: every claim about what a competitor ships is grounded in a **retrieved, dated, linkable source** (their live product, docs, changelog, pricing page, a review), or it is marked as an unverified lead. This skill is forward-authored for the Wave-2 `competitive-analyst` agent; it exists now so agents that reference it don't dangle.

## When to use me

- a competitive teardown of what a rival actually ships today
- checking whether a proposed feature is parity (table stakes) or differentiation (a wedge)
- grounding an initiative's "competitive signals" evidence line
- responding to "Competitor X already does this" — verify before it propagates

## The retrieval rule

**Model memory is a starting point for *where to look*, never a source for *what is true*.** When you recall that "Competitor X has SSO," that recollection is a lead: it tells you to go check X's docs/pricing/changelog. It is not evidence. State it only after you've retrieved a current source that confirms it, and cite that source.

Every competitive claim carries:

- **what** — the specific capability, described concretely
- **source** — a linkable, retrieved reference (product page, docs, changelog entry, pricing tier, dated review)
- **date** — when the source was observed (competitors ship; a claim without a date rots)

```
Claim:  Competitor X offers SAML SSO on their Business tier and above.
Source: x.com/pricing (Business tier feature list), x.com/docs/sso
Observed: 2026-07-15
```

A claim you can't source, you don't make. You write instead: *"Model recollection suggests X may have SSO — unverified, no current source retrieved. Recommend checking x.com/pricing before relying on this."* A named unverified lead is honest and useful; a confident unsourced assertion is the exact failure mode that makes AI competitive analysis distrusted.

## Teardown of what they actually ship

A teardown documents *observed behavior*, not marketing claims and not what you assume they do:

- **Walk the actual product** where possible — trial account, docs, demo videos, screenshots. What the feature *does*, not what the landing page says it does.
- **Read the changelog / release notes** for recency — what shipped lately signals where they're investing.
- **Read pricing** — what's gated behind which tier tells you what they consider premium vs table-stakes.
- **Read third-party signal** — G2/review complaints reveal the gaps behind the marketing.

Record the teardown as observed facts with sources, then interpret. Keep the two separate: "X's export is CSV-only, no scheduling (observed, x.com/docs/export, 2026-07-15)" is a fact; "so scheduled export is a differentiation wedge for us" is your interpretation of it.

## Parity vs differentiation

The output of competitive research is a judgment on each capability:

- **Parity (table stakes)** — everyone credible has it; not having it is a *liability*, not having a *plan to have it* loses deals. Parity work is defensive: match, don't over-invest.
- **Differentiation (a wedge)** — something you'd do materially better or that no credible competitor does well. This is where investment compounds. A differentiation claim requires evidence that the competitors *don't* do it well — which means retrieval, not assumption.
- **White space** — a need no one in the category serves. The highest-leverage finding, and the one most often imagined rather than verified — hold it to the strictest sourcing bar.

Feed this parity/differentiation/white-space split into `feature-comparison-framing` to build the matrix.

## A worked teardown

The task: *"Does Competitor X support SSO? We need to know before we scope our own."*

**The wrong move (model-memory):**
> "Yes, X has had SAML SSO for years — it's standard for tools at their scale."

That's a training-data recollection stated as fact. It might be true; it might be a year stale; it might confuse X with a competitor. Either way it's the failure mode that gets a bet mis-scoped.

**The right move (retrieval-augmented):**
1. **Lead from memory, verify from source.** Recollection says "X probably has SSO" → go check `x.com/pricing` and `x.com/docs`.
2. **Retrieve and record with date + link.**
   > SAML SSO: available on X's "Business" tier and above. SCIM provisioning: "Enterprise" tier only. No OIDC support found in docs.
   > Source: x.com/pricing, x.com/docs/security/sso · Observed 2026-07-15.
3. **Note what you couldn't verify.** "Could not confirm whether X supports IdP-initiated flows — docs don't say; would need a trial account."
4. **Interpret, kept separate from fact.** *"So SSO is table-stakes parity (X gates it at Business tier, matching the category); our wedge isn't SSO itself but making it self-serve on lower tiers — unverified whether X does, worth a trial."*

The output is a claim a PM can act on *and challenge*, with a link to click and a date to age-check — not an assertion they have to take on faith.

## Anti-patterns

- **Model-memory as fact.** "Competitor X has feature Y" stated from training-data recollection with no retrieved source. The cardinal sin — stale by construction, confidently wrong often enough to be dangerous.
- **Undated claims.** A competitive fact with no observation date rots silently; the competitor shipped something last week and your teardown is now wrong.
- **Marketing claims as behavior.** Repeating the competitor's landing-page copy as if it were observed capability. Walk the product; read the docs.
- **Feature-list arms race.** Cataloguing every checkbox a competitor has and treating each gap as a mandate. Distinguish parity worth matching from noise.
- **Imagined white space.** "No one does this" asserted without checking. White-space claims get the *strictest* sourcing bar precisely because they're the most tempting to imagine.
- **Positioning laundered as fact.** Presenting your strategic interpretation ("we win on X") as an observed truth. Keep observed facts and your interpretation visibly separate.

## Cross-references

- `feature-comparison-framing` — turns the sourced parity/differentiation judgments into a decision-useful feature matrix.
- `pm-agent-doctrine` — corpus-grounding doctrine at its strictest; competitive claims are the place fabrication does the most damage.
- `evidence-synthesis` — competitive signals are one evidence source feeding an initiative's Evidence section.
- `roadmap-framing` — "competitive signals" is a named evidence type for a bet; it must carry sources like any other.
