---
name: market-sizing
description: Defensible TAM/SAM/SOM where every step down rests on ONE statable, challengeable assumption — no un-annotated multipliers — computed top-down AND bottom-up, with divergence flagged as a signal to investigate rather than averaged into a false midpoint.
metadata:
  audience: product-strategist
  purpose: framework-guidance
---

## What I do

Turn "how big is this?" from a number pulled from the air into a **defensible size a skeptic can attack step by step.** Market sizing is where bets go to look serious and stay wrong: a single confident "$4B market" hides every assumption that produced it. This skill supplies the two disciplines that make a size trustworthy — **TAM/SAM/SOM with one challengeable assumption per step**, and **top-down↔bottom-up convergence** where computing the number two independent ways and *flagging their divergence* is the real signal. A size that survives both is defensible; one un-annotated multiplier and it's theater.

## When to use me

- answering Q3 (how big) of an `opportunity-assessment` before a bet is committed
- pressure-testing a "$Xbn market" claim in a pitch or board deck — find the un-annotated multiplier
- deciding whether an opportunity clears the floor to be worth building for at all
- sizing a segment or wedge, not just a whole category

## TAM / SAM / SOM — one challengeable assumption per step

Three nested tiers, each a documented step *down* from the last:

- **TAM (Total Addressable Market)** — the whole market if you had 100% of everyone who could conceivably buy. The ceiling.
- **SAM (Serviceable Addressable Market)** — the slice your product and go-to-market can actually serve (geography, segment, channel, price point).
- **SOM (Serviceable Obtainable Market)** — the slice you can realistically win in a defined window, given competition and capacity. The number a plan is built on.

**The rule: every step down is a documented, ONE-sentence challengeable assumption. No un-annotated multipliers.**

> `SAM = TAM × {% reachable by our GTM}` — assumption: *we can reach English-speaking SMBs on self-serve, ~35% of the global TAM.* A skeptic can attack the 35%.
> `SOM = SAM × {realistic share in window}` — assumption: *we take ~4% of reachable SAM in 3 years against two incumbents.* A skeptic can attack the 4%.

A multiplier without its one-sentence assumption is the failure mode — it looks like arithmetic but hides a guess. If you can't state the assumption behind a step in one challengeable sentence, you haven't sized that step; you've decorated it. This is the doctrine-1 grounding contract applied to arithmetic: each factor is grounded in a corpus source or flagged as the assumption to test.

## Top-down ↔ bottom-up convergence

Compute the size **both ways, independently:**

- **Top-down** — start from a published market figure (analyst report, industry data) and narrow: `market $ × addressable % × obtainable %`. Fast, but inherits whatever the report assumed.
- **Bottom-up** — build from unit economics: `reachable accounts × conversion × ACV` (or `users × price × attach rate`). Grounded in your own numbers, but blind to market you can't yet see.

Then **compare them, and flag divergence — do not average.** The gap between the two is the finding, not a nuisance to smooth away:

> Averaging a top-down $800M and a bottom-up $120M into "$460M" manufactures a false midpoint and buries the disagreement. The 6.7× gap *is* the signal — it means one method rests on a wrong assumption. Name which one: usually an inflated top-down addressable % or an optimistic bottom-up conversion.

**Divergence > ~2–3× means an assumption is wrong — find it and say so.** Convergence within that band is genuine corroboration (two independent methods agreeing raises confidence); wide divergence is a flag to investigate, not a number to compromise. The honest output is either "both methods land near $X (converged)" or "top-down says $X, bottom-up says $Y, they diverge ~Nx because assumption Z — resolve Z before trusting either."

Per doctrine 2: the size is a *proposal* with its assumptions exposed for challenge; the human decides whether it clears the bar to commit.

## Market Sizing (copy-paste artifact)

```markdown
## Market Sizing — <opportunity>

| Tier | Value | Formula | The one challengeable assumption | Source |
|------|-------|---------|----------------------------------|--------|
| TAM  | | | | |
| SAM  | | TAM × {% reachable} | | |
| SOM  | | SAM × {% obtainable in window} | | |

**Top-down:** <market $ × addressable % × obtainable %> = <$>
**Bottom-up:** <reachable accounts × conversion × ACV> = <$>
**Convergence:** <converged within ~2–3× → corroborated> | <diverges ~Nx → assumption <Z> is wrong; resolve before trusting>
**The assumption that most moves the number:** <name it — test it first>
```

## Anti-patterns

- **Un-annotated multiplier.** A step-down factor with no one-sentence assumption behind it. Looks like arithmetic, hides a guess — the signature sizing failure.
- **Single-method size.** Only top-down (inherits the report's assumptions) or only bottom-up (blind to unseen market). Compute both or you can't spot a wrong assumption.
- **Averaging divergence.** Splitting a top-down and bottom-up that disagree into a midpoint. The gap is the finding; the average buries it.
- **TAM as the headline.** Leading with the biggest, least-defensible number. SOM is what a plan is built on; TAM is the ceiling, not the pitch.
- **Precision theater.** "$4.237B" implies a rigor the assumptions don't support. Round to the confidence the weakest assumption allows.
- **Deciding on the size.** The size is a proposal; the human decides if it clears the bar (doctrine 2).

## Cross-references

- `opportunity-assessment` — market-sizing answers its Q3 (how big); the two run together before a bet commits.
- `pm-agent-doctrine` — corpus-grounding applied to every factor; a multiplier is grounded or flagged as an assumption to test.
- `outcomes-over-outputs` — the size bounds how big an outcome the bet could plausibly move.
- `assumption-testing` — test the single assumption that most moves the number before committing capacity.
