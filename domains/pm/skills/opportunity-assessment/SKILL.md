---
name: opportunity-assessment
description: Cagan's 10-question opportunity assessment, run under single-challengeable-assumption discipline — every answer rests on ONE statable assumption a skeptic can attack, or it's hand-waving. The gate a bet clears before it's committed.
metadata:
  audience: product-strategist
  purpose: framework-guidance
---

## What I do

Supply the discipline for deciding **whether an opportunity is worth pursuing** before a team commits build capacity to it. `outcomes-over-outputs` frames *what outcome* a bet moves; this skill asks the prior question — *is this bet worth making at all?* The instrument is Marty Cagan's (SVPG) **10-question opportunity assessment**: ten questions that force the case for a bet into the open, each answered under **single-challengeable-assumption discipline** so a skeptic can attack the weakest link instead of the whole thing being asserted at once. An assessment that survives the ten questions is a defensible go/no-go; one that hand-waves any of them is theater.

## When to use me

- before promoting an initiative from framing to committed — the go/no-go gate
- when a bet has a compelling outcome but an undefended case ("great idea, but *why us, why now, how big?*")
- pre-`/discover` scoping: which questions are already answered by the corpus and which need research
- sanity-checking someone else's pitch — walk the ten questions and find the hand-waved one

## The Cagan 10-question assessment

Answer each in one or two sentences, grounded in the team's corpus (`pm-agent-doctrine`) — not model-memory. The point is not ten paragraphs; it's ten *checkable claims*.

1. **Problem / value proposition** — exactly what problem does this solve, and what's the value of solving it? If you can't state the problem in one sentence, you don't have one yet.
2. **For whom / target market** — who specifically has this problem? A segment, not "users." Named accounts or a sized segment beat a persona sketch.
3. **How big / market size** — how large is the opportunity? Answered with a defensible **TAM/SAM/SOM**, not a number pulled from the air. **Cross-reference `market-sizing` for this question** — an unsized market fails the assessment.
4. **Competitive alternatives** — what do people do today, and who else is (or will be) in this space? Includes the do-nothing alternative and status-quo workarounds. **Cross-reference `competitive-research` here** — sourced, not recollected.
5. **Why us / our differentiator** — why are *we* the right team to win this? A right-to-win the competition can't easily copy, not a generic strength.
6. **Why now / market window** — why is this the moment? What changed (tech, regulation, behavior, cost curve) that makes now the window and not two years ago or two years from now?
7. **Go-to-market** — how do we reach and sell to the target market? A channel, a motion (self-serve / sales-led / PLG), a hook — not "we'll market it."
8. **How we measure success / revenue** — what's the success metric and the revenue/impact model? Baseline before target, per `metrics-design`.
9. **Critical success factors** — what has to be true for this to work? The dependencies, the risks, the "this only works if…" conditions. Name them so they can be watched.
10. **Recommendation / go-no-go** — the call: pursue, defer, or reject — with the *reasoning*, not just the verdict. A no carries its reason forward so the same opportunity isn't re-litigated next quarter.

## Single-challengeable-assumption discipline

Each of the ten answers rests on **ONE statable assumption a skeptic can attack**. If you can't compress the answer to one challengeable sentence, the answer is hand-waving dressed as analysis.

> For every question, write the load-bearing assumption as a single sentence a reasonable skeptic could disagree with — then it's testable. "The market is big" is not challengeable. "≥ 40% of mid-market SaaS teams run manual export weekly, so an automation wins meaningful time" *is* — a skeptic can attack the 40%.

This is the doctrine-1 grounding contract applied to a go/no-go: an answer is either grounded in the corpus, or flagged as the *one assumption* that would resolve it if tested. A confident un-challengeable answer is exactly the failure mode that gets a bad bet committed. The most valuable output of the assessment is often naming *which single assumption*, if wrong, sinks the whole opportunity — so the team can test that one first (see `assumption-testing`).

Per doctrine 2 (suggest-don't-decide): the assessment *surfaces* the go/no-go case; the human commits the bet. The strategist owns the audit trail, not the seat.

## Opportunity Assessment (copy-paste artifact)

```markdown
## Opportunity Assessment — <bet name>

| # | Question | Answer | The one challengeable assumption |
|---|----------|--------|----------------------------------|
| 1 | Problem / value proposition | | |
| 2 | For whom / target market | | |
| 3 | How big / market size (→ market-sizing) | | |
| 4 | Competitive alternatives (→ competitive-research) | | |
| 5 | Why us / differentiator | | |
| 6 | Why now / market window | | |
| 7 | Go-to-market | | |
| 8 | Success metric / revenue | | |
| 9 | Critical success factors | | |
| 10 | Recommendation / go-no-go | | |

**Weakest link:** <the single assumption that, if wrong, sinks the bet — test this first>
**Recommendation:** <pursue / defer / reject> — <reasoning; carried forward if rejected>
```

## Anti-patterns

- **Hand-waved question.** Any of the ten answered with an un-challengeable generality ("big market," "we're the best team"). If a skeptic can't disagree with it, it isn't an answer.
- **Skipping how-big.** An opportunity with no defensible size is a bet against an undefended market — send it to `market-sizing`.
- **Recollected competition.** Answering Q4 from model-memory instead of sourced `competitive-research`. The cardinal competitive sin.
- **Verdict without reasoning.** A go/no-go that states the call but not the case; the reasoning is the reusable part.
- **Ten paragraphs.** The assessment is ten checkable claims, not an essay. Compression is the discipline.
- **Deciding the bet.** The assessment surfaces the case; the human commits it (doctrine 2).

## Cross-references

- `market-sizing` — answers Q3 (how big) with defensible TAM/SAM/SOM under the same single-assumption discipline.
- `competitive-research` — answers Q4 (alternatives) with sourced, dated competitive facts, never recollection.
- `outcomes-over-outputs` — frames the bet's outcome once the opportunity clears the gate.
- `assumption-testing` — test the weakest-link assumption before committing build capacity.
- `pm-agent-doctrine` — corpus-grounding and suggest-don't-decide; the assessment is a proposal, the human commits.
