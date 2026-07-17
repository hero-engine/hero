---
name: risk-surfacing
description: Name risks concretely — scenario + indicator + response — so a Risks section is decision-useful instead of a generic worry list. "Might not scale" is not a risk; "if usage exceeds 10×, the cron digest misses its window" is.
metadata:
  audience: product-strategist, roadmap-curator, pm-reviewer, and the deferred risk-curator / roadmap-reviewer agents
  purpose: curation
---

## What I do

Turn vague unease into risks a team can actually act on. Most Risks sections are theater — a bulleted list of generic worries ("technical complexity", "timeline risk", "adoption risk") that no one can plan against because none of them says *what would happen, how we'd see it coming, or what we'd do*. This skill supplies the shape that makes a risk useful: a **concrete scenario**, an **early indicator**, and a **planned response**. When `product-strategist` writes an initiative's disconfirming-signal, when `pm-reviewer` audits a PRD's Risks section, or when `roadmap-curator` flags a bet that's aging badly, this is the bar.

## When to use me

- writing the Risks section of a PRD or pitch
- writing an initiative's "what we'd see if we're wrong" / disconfirming-signal
- reviewing a Risks section for substance (`pm-reviewer`)
- a premortem pass on a bet ("what are the top ways this fails?")
- flagging delivery risk on a roadmap item that's committed but stalling

## The three-part risk

A useful risk names all three. Drop any one and it stops being actionable:

1. **Scenario** — the specific thing that goes wrong, concrete enough to picture. Not "performance risk" — *"if the nightly digest job runs longer than its 2-hour window, it collides with the morning backup and one of them fails."*
2. **Indicator** — the early signal that the scenario is starting to happen, ideally observable *before* the failure. *"Digest runtime crosses 90 minutes"* or *"we see the first collision warning in the job log."* The indicator is what turns a risk from a surprise into a managed event.
3. **Response** — what we do when the indicator fires. *"Shard the digest by segment; we've scoped it at ~3 days and can start the day the indicator trips."* A risk with no response is a worry; a risk with a pre-decided response is a plan.

**Template:**

```
Risk: <one-line name>
  Scenario:  <the specific failure, concrete enough to picture>
  Indicator: <the observable early signal>
  Response:  <the pre-decided action + rough cost/timing>
  Likelihood/Impact: <low/med/high × low/med/high, if the team ranks>
```

## Worked examples

**Weak (worthless):**
> - Scalability risk
> - The timeline might slip
> - Users might not adopt it

None of these can be planned against. "Scalability risk" — of what, at what threshold, seen how? "Timeline might slip" — every project's timeline might slip; this says nothing.

**Strong (actionable):**
> **Risk: digest misses its delivery window under load**
> Scenario: if active accounts exceed ~10× today's volume, the cron-based digest won't finish inside its 2-hour window and users get stale or missing digests.
> Indicator: digest job runtime crosses 90 minutes (alert already wired).
> Response: switch to per-segment sharding — scoped at ~3 days, can start the day the alert trips. We accept the risk now rather than pre-building the shard.

> **Risk: enterprise adoption stalls on SSO gap**
> Scenario: the two pipeline enterprise deals require SAML SSO; if we ship without it, both stall in procurement.
> Indicator: either deal's security review returns SSO as a blocker (expected within 3 weeks).
> Response: SSO is already scoped as the fast-follow; we sequence it next if the indicator fires, and we've told sales not to commit a date.

## Naming vs. discovering risk

Two moves, don't confuse them:

- **Surfacing a *known* risk** — you can already picture the scenario; write it in the three-part shape.
- **Surfacing an *assumption* whose failure is a risk** — you're not sure the scenario happens because you're not sure a belief holds. That's an assumption to test, not a risk to plan. Route it to `assumption-testing` (design a fast test) rather than writing a speculative response for a scenario you can't yet picture.

A Risks section should hold the first kind. The second kind belongs in an assumption test — dressing an untested assumption as a "risk" with a made-up response is false confidence.

## The premortem procedure

When you need to *generate* risks (not just format known ones), run a premortem — the most reliable way to surface the scenarios optimism hides. Gary Klein's move: imagine it's six months out and the bet **failed**, then work backwards.

1. **Assume failure.** "It's Q4. The bulk-import bet shipped and flopped. Activation didn't move." State it as fact, not possibility — this licenses honesty a "what might go wrong?" question suppresses.
2. **Enumerate causes.** Ask each stakeholder to independently write *why* it failed. Independent-first beats group brainstorm — it avoids anchoring on the first (usually loudest) cause.
3. **Convert each cause to a three-part risk.** "Users didn't trust auto-import" → scenario (they abandon after one bad import), indicator (first-import error rate > X%, or drop-off after first import), response (add a dry-run preview). A premortem cause with no indicator/response is just a fear — finish the shape.
4. **Rank by likelihood × impact**, keep the top few, and put them in the artifact's Risks section. A premortem that surfaces 15 risks and plans for none is theater; triage to the vital few.

The premortem's value is psychological as much as analytical: "why did this fail?" gives people permission to voice the doubt that "any concerns?" shuts down.

## Anti-patterns

- **Generic risk catalogs.** "Technical complexity, adoption risk, timeline risk" pasted into every PRD. If the same three risks fit every project, they describe none of them.
- **Risks without indicators.** A scenario and a response but no early signal means you find out at failure time, when the response is a fire drill instead of a plan.
- **Risks named after the technical-debt-of-the-week.** The engineer's current annoyance is not automatically a product risk. A risk is a threat to the *outcome*, framed in scenario/indicator/response terms.
- **Untested assumptions dressed as risks.** A speculative response for a scenario you can't picture is theater. If you don't know whether the scenario happens, that's an assumption test (`assumption-testing`), not a risk.
- **Likelihood/impact scores with no scenario.** A 2×3 heat map is decoration if the cells don't name concrete scenarios. Score the scenario, not a category.
- **Response = "monitor it."** "We'll keep an eye on it" is not a response. Name the action and its rough cost so the team can decide now whether to pre-invest or accept-and-watch.

## Cross-references

- `assumption-testing` — when the risk is really an untested belief, design a fast test instead of writing a speculative response.
- `roadmap-framing` — an initiative's disconfirming-signal is a risk in outcome terms ("what we'd see if the bet is wrong").
- `prd-anti-patterns` — empty or generic Risks sections are a catalogued PRD smell.
- `pm-agent-doctrine` — a named risk still grounds in the corpus; "if usage exceeds 10×" should cite the current-volume number it's 10×-ing.
