---
name: customer-segment-weighting
description: Weight reach and impact by segment economics instead of counting every user equally — team-decided weights recorded once and reused, applied consistently, and always disclosed in the notes so a weighted score never quietly outranks an unweighted one.
metadata:
  audience: intake-triager, prioritization-strategist
  purpose: curation
---

## What I do

Give agents the discipline to treat customers as economically different when they are — without turning that into a hidden thumb on the scale. The default "reach = number of users" formulation treats 200 free-tier users as equal to 200 enterprise accounts paying $100k/year. Often that's wrong, and a naive count buries the signal. But segment weighting is also the easiest place to quietly rig a ranking, so the discipline is as much about *disclosure and consistency* as about the math. `prioritization-frameworks` owns the RICE/ICE/WSJF mechanics and has a segment-weighting section; this skill is the focused companion on *how to weight honestly*. When `intake-triager` judges how much a signal counts, or `prioritization-strategist` computes weighted reach, this is the method.

## When to use me

- computing RICE Reach (or any reach-based score) where segments differ materially in value
- triaging intake — deciding how much weight one customer's ask carries
- a stakeholder argues "but this is our biggest customer" and you need to make the weighting explicit rather than ad hoc
- setting up (or auditing) a team's standing segment-weight table

## Weight by economics, decided once

When segments differ materially in value, weight reach by segment importance:

```
Reach_weighted = Σ (users_in_segment × segment_weight)
```

The weights come from a **team decision, recorded once and reused** — not invented per scoring. A representative table:

```
enterprise  = 5
mid-market  = 3
smb         = 1
free        = 0.3
```

Two rules make this honest:

1. **Recorded once, applied consistently.** The weight table lives in a durable place (a decision note in `.hero/knowledge/decisions/`, or team config), and every score uses the *same* table. The moment weights change per-item to help a favored bet, the framework is theater. The exact numbers matter far less than using one agreed set everywhere.
2. **Grounded in real economics.** A weight should trace to something — ARPU by segment, retention value, strategic account status, cost-to-serve. "Enterprise = 5" because enterprise ARPU is ~5× SMB is defensible; "enterprise = 5" because the loudest stakeholder is enterprise is not.

## Disclosure is non-negotiable

**A weighted score must say so, in the notes, every time.** A RICE number that quietly uses weighted reach without disclosure is *worse* than one that doesn't weight at all — because a reader comparing it against an unweighted score is comparing apples to oranges without knowing it, and the weighting silently decided the ranking.

The disclosure rule, concretely:

> Any score using segment weighting names the weight table and shows the weighted-vs-raw reach in the Notes column.

```
| Item        | Raw reach | Weighted reach | RICE | Notes                                      |
|-------------|-----------|----------------|------|--------------------------------------------|
| SSO for SMB | 400/qtr   | 400            | 75   | SMB weight 1.0 — no weighting effect       |
| Bulk import | 120/qtr   | 600            | 210  | 120 enterprise × weight 5 (std table v2)   |
```

A reader can now see *why* bulk import outranks SSO — the enterprise weighting did it — and can challenge the weight if they disagree. That is the whole point: weighting makes the economic judgment *visible and contestable*, not hidden.

## When NOT to weight

Weighting is a tool, not a default. Don't reach for it when:

- **segments don't differ materially** — if the economics are within ~2× across segments, weighting adds noise and false precision; count straight.
- **the bet is strategic, not reach-driven** — a compliance requirement or a strategic-account unblock is decided by strategic context (logged in `rationale`), not by a weighted reach number. Don't launder a strategic override into a weighting tweak.
- **you'd have to invent weights on the spot** — no standing table means no honest weighting. Use raw reach and flag that the team should decide weights, rather than fabricating them to win an argument.

## A worked example — the disclosure that changes the call

Two bets, ranked by RICE, weights from the team's standing table (`enterprise=5, mid-market=3, smb=1, free=0.3`, decision note `seg-weights-v2`):

**Naive (unweighted, raw counts):**
> - "Free-tier onboarding polish" — Reach 4,000 free users → RICE 320 → **rank 1**
> - "Enterprise audit log" — Reach 80 enterprise accounts → RICE 90 → **rank 2**

By headcount, onboarding polish wins in a landslide. But the segment economics are ~17× apart, and the raw count buries that.

**Weighted, disclosed:**
> - "Enterprise audit log" — raw 80, **weighted 400** (80 × 5) → RICE 210 → **rank 1** · *Notes: enterprise weight 5 (seg-weights-v2); 3 of the 80 are renewal-risk this quarter.*
> - "Free-tier onboarding polish" — raw 4,000, **weighted 1,200** (4,000 × 0.3) → RICE 190 → **rank 2** · *Notes: free weight 0.3 (seg-weights-v2).*

The ranking flips — and crucially, a reader can *see why*: the Notes column shows the weighting did it, names the table version, and shows raw-vs-weighted for both. Anyone who thinks free-tier growth matters more than the weight implies can challenge `seg-weights-v2` directly, rather than discovering months later that a hidden multiplier decided the roadmap. That visibility *is* the deliverable — the weighting isn't the point, the contestable transparency is.

## Anti-patterns

- **Equal-weight reach when economics are 10× apart.** Treating 1000 free users as equal to 100 enterprise accounts buries the real signal.
- **Per-scoring weights.** Inventing or nudging weights item-by-item to steer a ranking. Weights are decided once and reused; anything else is confidence-pumping with extra steps.
- **Silent weighting.** Using weighted reach without disclosing it in the notes. The cardinal sin — it makes weighted and unweighted scores falsely comparable.
- **Weights with no economic basis.** "Enterprise = 5 because the VP said so." A weight should trace to ARPU, retention value, or a recorded strategic decision.
- **Weighting a strategic override.** Dressing "we must do this for the Acme renewal" as a reach-weighting bump. Strategic overrides go in `rationale`, transparently, not into the reach math.
- **False precision.** Three-decimal weighted scores implying the weights are exact. They're a team judgment; keep the numbers coarse and the reasoning visible.

## Cross-references

- `prioritization-frameworks` — owns RICE/ICE/WSJF mechanics; its segment-weighting section is where this plugs in. Read it for the Reach/Impact/Confidence/Effort definitions.
- `intake-classification` — segment is captured at triage; consistent segment tags are what make weighting possible downstream.
- `evidence-synthesis` — weighting by segment economics is one axis; recency and pain-intensity are others.
- `pm-agent-doctrine` — decision-gate doctrine: the weighted ranking is a proposal with visible math, and the team owns the call; disclosure is the audit trail.
