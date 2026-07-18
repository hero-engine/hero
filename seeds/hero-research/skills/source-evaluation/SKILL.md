---
name: source-evaluation
description: How to triage a source before its claims are used — credibility, recency, primary-vs-secondary, bias, and corroboration — and when to discard one outright. Keeps a research run from laundering a weak source into a confident claim.
metadata:
  audience: researcher, document-analyst
  purpose: source-triage
---

## What I do

I give the rules for deciding whether a source is trustworthy enough to build a
claim on, and how much weight it carries once it is. Retrieving a source is not
the same as trusting it; this skill is the gate between the two. Assembling
evaluated sources into cited claims is `evidence-and-citation`'s job — I stop at
the verdict on each source.

## When to use me

Load me whenever a research or document-analysis run has retrieved sources and is
about to use them — in `research-workflow`'s round loop, every retained source
passes through here before its content is used.

## The five triage dimensions

Evaluate every retained source on all five. A source can fail on one dimension
and still be usable if you carry the caveat; it can fail on several and be
discarded.

1. **Credibility.** Who produced this, and what is their basis for knowing? A
   named author or institution with domain standing and a methodology you can
   inspect outranks an anonymous post asserting a conclusion. For a corpus item,
   credibility is provenance: a decision record or a spec outranks an offhand
   note.

2. **Recency.** Is it current enough for the question? Recency matters enormously
   for fast-moving topics (prices, versions, standings) and barely at all for
   stable ones (definitions, history). State the source's date and judge it
   against the question's volatility — do not treat "recent" as universally
   better.

3. **Primary vs secondary.** Is this the origin of the claim or a retelling? A
   primary source (the original study, the spec, the announcement) outranks a
   secondary one (an article summarizing it) for the same claim. When you have
   only a secondary source, say so — and prefer to trace it back to its primary.

4. **Bias and incentive.** Does the source have a stake in the conclusion? A
   vendor describing its own product, an advocate for a position, a summary
   selected to support a thesis — none are disqualifying, but each is weighted
   with its incentive named.

5. **Corroboration.** Is the claim independently supported? One source is a lead;
   two independent sources that agree is evidence. Independence matters — three
   articles all citing the same origin are one source wearing three coats, not
   three sources.

## When to discard

Discard a source, rather than caveat it, when:

- You cannot establish who produced it or on what basis (no credibility floor).
- It is contradicted by a clearly stronger source and adds nothing but noise.
- It is a circular citation — it traces back to a source already in your set.
- It is stale on a question where recency is decisive and a current source exists.

Discarding is not censorship; it is refusing to let a weak source lend false
weight to a claim. Note in the `evaluation` checkpoint what you discarded and why,
so the reasoning is visible.

## Weighting, not just pass/fail

Most sources are neither pristine nor worthless. The output of triage is a
*weight*: how much this source can support on its own, and what caveat travels
with it. A biased-but-primary source carries its claim with the incentive named.
A credible-but-secondary source carries its claim flagged as second-hand. That
weight is what `evidence-and-citation` uses when it decides whether a claim is
solid enough to state plainly or must be hedged.

## Anti-patterns

- **Retrieval as endorsement.** Treating "it came back in the search" as "it is
  true." The search finds relevance, not reliability.
- **Counting coats, not sources.** Mistaking many retellings of one origin for
  independent corroboration.
- **Recency as a proxy for quality.** The newest source is not the best source
  unless the question is time-sensitive.
- **Silent discard.** Dropping a source without noting it, so a reader cannot
  tell whether it was considered and rejected or never seen.
- **Uncaveated bias.** Using a source with an obvious incentive as if it were
  disinterested.
