---
name: evidence-forcing
description: Force every prioritization input to name its evidence or default to neutral — the discipline that keeps a RICE/ICE/WSJF score from being a soft opinion wearing two decimal places. Confidence with no named evidence is 50%, not the optimistic value; recompute at the honest default and show the swing.
metadata:
  audience: prioritization-challenger, and the deferred prioritization-strategist critic-mode
  purpose: framework-guidance
---

## What I do

Make soft prioritization inputs honest. A prioritization framework's output is only as good as its inputs, but the output *prints like data* — RICE scores to two decimals, ICE lands on a crisp 1–1000, WSJF sorts a column — while the inputs behind them are frequently guesses. This skill supplies the discipline that draws the line between **RICE-as-data** (inputs each carry named evidence) and **RICE-theater** (inputs are vibes, the score is quoted as fact in a roadmap review). `prioritization-frameworks` names the abuse modes; this skill is the method for *hunting* them — the operationalized challenge a prioritization critic runs against an existing ranking.

The move is not to demand certainty. It's to demand **named evidence or an honest default**. An input that can cite its basis stays; an input that can't drops to neutral and the score gets recomputed at the honest value so the team can see how much of the ranking was resting on an unsupported number.

## When to use me

- challenging any framework score (RICE / ICE / WSJF / value-vs-effort) whose inputs a critic must stress-test
- auditing a backlog ranking before it drives a roadmap commitment
- when a stakeholder says "but RICE ranks this #1" and you need to check whether the #1 is earned or manufactured
- reviewing a re-scored item whose position changed without new evidence

This is the *challenger's* half of prioritization. For the framework mechanics and when each fits, load `prioritization-frameworks`; for the outcome-framing of Impact, load `outcomes-over-outputs`. Reach for me when the job is to interrogate inputs, not to produce a ranking.

## The default-to-neutral rule

The core discipline, stated as a rule:

> An input with no named evidence does not get its optimistic value. It defaults to neutral, and the score is recomputed at the default.

- **Confidence** is the canonical case. A Confidence with no named evidence is **50%** (a hunch, per `prioritization-frameworks`), not the 80% that "feels right." Below 50% the item belongs in discovery, not prioritization.
- The same principle applies to any unsupported multiplier: an **Impact** rating with no outcome tie defaults toward the middle of the scale, not the top; a **Reach** with no source defaults to the defensible floor (the segment you can actually name), not the aspirational ceiling.
- **Named evidence** means a citation the team can follow: an intake item, a support-ticket tag with a count, an analytics query, an interview finding, a segment-size number from the corpus. "The team thinks it's high" is not named evidence — it's the absence of it, honestly labeled.

**Recompute and show the swing.** The rule has teeth only when you *recompute*. Take the ranking as scored, reset every unsupported input to its neutral default, and re-run the formula. The delta is the finding:

```
As scored:   Bulk import — Reach 5000, Impact 3, Confidence 80%, Effort 2 → RICE 6000  (rank #1)
Defaulted:   Reach 5000 (no source → floor 1200), Impact 3 (no outcome tie → 2),
             Confidence 80% (no evidence → 50%), Effort 2 → RICE 750  (rank #6)
Finding:     #1 → #6 once unsupported inputs default. The top ranking rested on
             three un-cited numbers. Which of the three can the team source?
```

The recomputed score is a *suggestion*, not a rewrite of the board (doctrine 2). The point is to show the ranking's **sensitivity** to its softest inputs, so the team can decide whether to source the evidence or accept the lower rank.

## Inflated-input detection

Beyond missing evidence, hunt inputs that are *present but inflated*:

- **Reach that exceeds the addressable segment.** A Reach of 5,000/quarter when the corpus says the relevant segment holds 1,800 accounts is inflated on its face. Anchor Reach to a corpus number — segment size, active-account count, event volume from analytics — and flag any Reach that outruns it.
- **Impact with no outcome tie.** An Impact of 3 ("massive") that can't name the behavior it would move is a vibe. Force the tie: *what outcome does this Impact assume, and against what baseline?* (This is `outcomes-over-outputs` applied to the Impact input.) No tie → default toward the middle.
- **Effort that ignores known dependencies.** An Effort estimate that doesn't account for a blocking dependency already in the graph is sandbagged low, inflating the score. Effort should be engineering-co-signed; a PM-only Effort is a fiction (per `prioritization-frameworks`).

Every inflation finding **cites the corpus number it contradicts** — "Reach 5000 vs. segment `enterprise` = 1800 accounts (corpus)." An inflation claim with no anchor is exactly the ungrounded free-association the doctrine forbids; the challenge must be checkable.

## Confidence-pumping detection

The most corrosive abuse is not a single soft input — it's an input that *moves to win*. Track an input across revisions:

- Pull the item's prior scores from history where available.
- A **Confidence that rises across revisions with no new named evidence** is the tell — the framework is being tuned to defend a pet ranking rather than to reflect what's known. (Doctrine 2's anti-gaming corollary: agents — and the humans they audit for — must not quietly tune inputs to steer a decision.)
- The finding names the movement and demands the evidence that would justify it: *"Confidence went 50% → 65% → 80% across three revisions; no new intake, interview, or analytics landed between them. What evidence supports the climb?"*

## Show-the-math discipline

Every challenge re-states the score at defensible inputs. Never "this feels inflated" — always the recomputed number and the delta, in a table the team can read and argue with. The Notes column carries the *basis* of every retained input and the *reason* every defaulted one dropped. A challenge without the recomputed math is an opinion; a challenge with it is a checkable claim the team can accept, source-around, or overturn.

## Anti-patterns

- **Challenging the order, not the inputs.** The human owns the output order (strategic overrides are legitimate and logged, per `prioritization-frameworks`). The critic owns *input integrity*. Attacking the rank instead of the inputs oversteps the seat.
- **Demanding certainty.** The goal is *named evidence or an honest default*, not perfect data. A defensible 50% Confidence with a flagged gap is a fine input; a fabricated 80% is not.
- **Black-box "this feels inflated."** Any inflation claim with no corpus number behind it. The anchor is the whole point — an unanchored challenge is the free-association the doctrine forbids.
- **Recompute without showing it.** Asserting "the real rank is #6" without the table. The swing is the finding; hide the math and the team can't check you.
- **Treating the score as truth to defend.** A framework score is a claim to interrogate, not a verdict to protect. The critic's loyalty is to the audit trail, not to any item's rank.

## Cross-references

- `prioritization-frameworks` — the formulas, the "how scores lie" catalog, and the show-the-math table this skill enforces; load it for the mechanics.
- `pm-agent-doctrine` — doctrine 2's anti-gaming corollary ("don't quietly tune inputs to steer a decision") is the discipline this skill hunts for; every challenge grounds in the corpus (doctrine 1).
- `outcomes-over-outputs` — an Impact input with no outcome tie is this framework's version of output-framing; force the behavior-and-baseline tie.
- `evidence-synthesis` — the mechanics of what counts as named evidence and how to weight it across sources.
