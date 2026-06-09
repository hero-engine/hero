---
name: forecast-methodology
description: Weighted pipeline formula, coverage ratio targets, commit vs. best-case vs. upside definitions, and slippage risk signals. Loaded by forecast-analyst.
metadata:
  audience: forecast-analyst
  purpose: forecast-framework
---

## What this skill covers

- The three forecast categories: commit, best case, upside
- Weighted pipeline calculation
- Coverage ratio and what it means
- Slippage risk signals
- Forecast confidence levels
- How to adjust for rep optimism bias

---

## The Three Forecast Categories

### Commit

**Definition:** Deals the rep is willing to stake their number on. If these
don't close, the rep missed — no excuses.

**Typical criteria:**
- Probability ≥ 80% (Negotiation stage or signed-off Proposal)
- Economic Buyer engaged and verbal intent received
- Legal review started or complete
- Specific close date confirmed by buyer

**What it means for the forecast:** Commit is the floor. CROs plan the
business around commit.

---

### Best Case

**Definition:** Commit plus deals that *could* close if everything goes
right. Not guaranteed, but realistic.

**Typical criteria:**
- Probability 40–79% (active Evaluation, Proposal submitted)
- Strong champion, EB identified, active process
- No major blockers outstanding

**What it means for the forecast:** Best case is the upside scenario.
A team consistently hitting best case is overperforming.

---

### Upside / Pipeline

**Definition:** All open deals in the pipeline, weighted by probability.
Includes early-stage deals that are real but uncertain.

**Typical criteria:**
- All open deals (Prospect through Negotiation)
- Weighted by stage probability

**What it means for the forecast:** Upside is the raw material. Low
upside relative to quota means the team is one bad quarter away from
a miss.

---

## Weighted Pipeline Calculation

### Per-deal calculation

```
weighted_arr = arr × (probability / 100)
```

### Stage probability defaults (from hero.json)

| Stage | Default Probability |
|---|---|
| Prospecting | 10% |
| Qualifying | 20% |
| Evaluation | 40% |
| Proposal | 60% |
| Negotiation | 80% |
| Closed Won | 100% |
| Closed Lost | 0% |

Override with deal-specific probability when MEDDPICC score or deal context
warrants a different estimate.

### Portfolio calculation

```
total_weighted = sum(arr × probability / 100) for all open deals
```

This is the expected value of the pipeline — what you'd expect to close
if you ran the same pipeline 100 times.

---

## Coverage Ratio

**Formula:**

```
coverage_ratio = total_open_pipeline_arr / quota
```

**Target: 3x**

This means for every $1 of quota, you should have $3 of open pipeline.
Why 3x? Because deals fall out, timing slips, and not every qualified
deal closes. 3x coverage gives the team enough cushion to hit quota.

**Interpretation:**

| Coverage | Interpretation | Action |
|---|---|---|
| ≥ 3.0x | Healthy — pipeline can absorb normal deal loss | Monitor |
| 2.5–3.0x | Borderline — one large deal loss threatens quota | Add pipeline now |
| 2.0–2.5x | At-risk — must hold every deal and add more | Urgent pipeline generation |
| < 2.0x | Critical — forecast miss likely without new large deals | Executive escalation |

**Surface coverage ratio prominently.** It is the most important leading
indicator of whether the team will hit quota. A team with 2x coverage in
week 4 of a quarter needs to act immediately, not at the end of the quarter.

---

## Slippage Risk Signals

Flag a deal as at-risk when 2+ of these signals are present:

### Activity-based signals

- **Stale deal:** No CRM activity or spec update in 14+ days (Prospect/Qualifying)
  or 10+ days (Proposal/Negotiation)
- **Unresponsive champion:** Champion has not responded to 2 outreach attempts
  in 7 days
- **Meeting canceled without rescheduling:** A key meeting was canceled and
  no new date is set within 48 hours

### Process-based signals

- **Close date pushed:** Close date moved back since last review (once is a
  signal; twice is a pattern)
- **Stage regression:** Deal moved backward in stage (Proposal → Evaluation)
- **Missing next step:** No next step with a date in the deal spec

### Qualification-based signals

- **Low MEDDPICC at late stage:** Score < 50 at Proposal or Negotiation
- **Missing Economic Buyer at Proposal:** EB not engaged when proposal submitted
- **Single-threaded:** Only one active contact in any deal above $50K ARR
- **No compelling event:** No "why now" has been documented

### Champion-based signals

- **Champion went quiet:** Champion no longer responsive or proactive
- **Champion departed:** Champion left the company or changed roles
- **Champion demoted:** Champion lost internal credibility or scope

### Competitive signals

- **New competitor appeared:** A new vendor entered the evaluation late
- **Competitor doing POC:** Competitor has been given a trial or POC
  opportunity we didn't know about
- **Price pressure intensified:** Buyer is pushing hard on price after
  previously not having price concerns

---

## Forecast Confidence Levels

Report forecast confidence in addition to the numbers:

**High confidence:**
- All commit deals have verbal EB intent, legal in flight, specific close date
- Weighted pipeline is 3x+ quota
- No deals with 2+ slippage signals

**Medium confidence:**
- Some commit deals have outstanding questions (legal not started, EB meeting
  pending)
- Weighted pipeline is 2.5–3x quota
- 1–2 deals with slippage signals

**Low confidence:**
- Multiple commit deals are dependent on events outside our control
- Weighted pipeline below 2.5x quota
- Multiple deals with 2+ slippage signals

---

## Adjusting for Rep Optimism Bias

Reps are optimistic by nature (that's part of what makes them good at
sales). The forecast should apply systematic adjustments:

### MEDDPICC-probability consistency check

If a deal is probability 80% (commit-level) but MEDDPICC score is < 40:
- Flag the mismatch
- Apply a risk-adjusted probability: `meddpicc_score × 0.8` as a ceiling
- Example: MEDDPICC 35 → probability ceiling of 28%, not 80%

### Stage-probability consistency check

If a deal's probability is above its stage default without documented
justification, flag it. A deal in "Qualifying" should not be at 70%
probability unless there's a documented reason.

### Historical win rate calibration

If historical data is available (`hero search --type retro`), compute
actual win rate by stage. Use this to calibrate stage default probabilities:

```
actual_win_rate_by_stage = won_deals_that_were_at_stage / deals_that_were_at_stage
```

If actual win rate at "Proposal" is 45% but the configured weight is 60%,
adjust the configured weight to match history.

---

## Forecast Output Format

The `forecast-analyst` should produce all sections in the format defined
in `forecast.md` (the command file). The key requirement is:

1. Executive summary first — the numbers a CRO reads in 30 seconds
2. Coverage ratio prominently and with interpretation
3. At-risk deals specifically named with the risk signal and recommended action
4. No hidden assumptions — state the methodology used

The forecast written to disk is the authoritative record. Chat output
summarizes; the file is the record.
