---
name: intake-classification
description: Classifying inbound intakes by theme, segment, source quality, and signal strength — without losing source attribution or treating one loud customer as a pattern.
metadata:
  audience: intake-triager, pm-investigator, duplicate-detector
  purpose: intake-curation
---

## What I do

Provide the classification logic that turns a raw inbound signal into a triaged intake. Classification has four axes — **theme**, **segment**, **source quality**, **signal strength** — plus the explicit handling of uncertainty (when to escalate to `pm-investigator` instead of guessing). The output is an intake with enough structure to be linked, merged, promoted, or rejected by `intake-triager` within the 24-hour SLA.

## When to use me

Load this skill when:

- triaging a new `intake` (default: `intake-triager`)
- investigating an ambiguous signal that's already on the queue (`pm-investigator`)
- comparing two intakes for duplicate consideration (`duplicate-detector` — overlap on theme + segment is one of the strongest signals)
- a PM is reviewing the `new` queue manually and wants to apply the same classification the agent would

## The four axes

### Theme

The substantive area the signal is about. Themes are how `roadmap-curator` later balance the roadmap.

**Let themes emerge.** Do not pre-build a 30-category taxonomy. Start by reading the signal and naming what it's *about* in 1-3 words ("csv export", "saml sso", "mobile offline"). Over time, the same themes recur; the recurring ones earn a place in the workspace glossary (P1, ships v1.5).

**Rules:**

- Multi-theme intake is fine — apply up to three tags.
- Reuse existing themes verbatim before inventing new ones. Check `hero search --list --tag <theme>` first.
- If you find yourself naming a 25th distinct theme in a month, the theme axis is too granular — collapse.

### Segment

Which customer segment the signal came from. Different segments have different ROI math:

- **Enterprise** — single signal can carry contractual / revenue weight; one customer at $500k ARR is a pattern of one.
- **SMB** — patterns matter more than individuals; ten SMB customers asking is real signal, one is anecdote.
- **Prosumer / individual** — volume is the signal; raw count matters.

When the source attribution names a specific customer, set `customer_segment` on the spec. When it's a synthesized "many users said this," ask which segment(s) — the answer changes the prioritization framework downstream (`prioritization-frameworks` skill).

### Source quality

A pyramid, in order of trust:

1. **Verbatim customer quote** — the customer's own words, captured in `source_quote`. Highest trust.
2. **Paraphrased ticket** — support summarizing the customer. Trust is contingent on the ticket linking back to the original.
3. **Sales restatement** — "the prospect needs X." Often shaped by deal-desire bias; treat as input to investigation, not as conclusion.
4. **Internal hunch** — "I think users would want…" Lowest trust. Valid as a question, not as evidence.

**Preserve source attribution as the trust signal.** This is the rule. The `source_quote` field stays verbatim through every downstream agent that touches the spec. Paraphrasing the customer's words erases the only thing that makes the item credible to a future skeptic asking "how do you know?"

### Signal strength

- **One customer asking once** — not signal. Log it; do not promote.
- **One enterprise customer asking with revenue attached** — signal *for that account*, not a market signal. Triage decision depends on the deal value vs. the build cost.
- **Repeated across a segment** — pattern. Worth promoting to an initiative or merging with an existing one.
- **Blocking deal / churn risk** — escalation. Use the `intake-triager`'s "escalated" annotation and route to PM leadership in the same pass.

Strength is not measured by *intensity of one quote* ("this is killing us!"). It's measured by *recurrence across distinct sources*. One emphatic ticket and ten quiet ones — the ten quiet ones are the pattern.

## Classification under uncertainty

You will hit intakes where:

- The signal is too vague to theme ("the product feels slow").
- The segment is unclear (no customer named on the ticket).
- The ask is buried under emotion ("just fix it!").
- Multiple distinct asks are bundled into one ticket.

**Do not guess.** Two options:

1. **Escalate to `pm-investigator`.** Set status to `triaged` with a note in the Investigation section: "Ambiguous theme — `pm-investigator` to interview ticket reporter or pull adjacent tickets." The investigator does the work; you don't fake confidence.
2. **Split.** Bundled intakes get split into multiple intakes, each individually classified. The originals link to the splits via `merged_into` semantics inverted (cross-link in the Notes section).

The 24-hour SLA covers *triage*, not *resolution*. "Escalated to investigation" is a complete triage outcome.

## The 24-hour triage SLA

Every intake gets a status (linked / merged / rejected / escalated-for-investigation) within 24 hours of landing in `new`. This is the operational standard from the intake spec type.

`intake-triager` runs against the `new` queue continuously. Items aging past 24h surface as a `/triage` sweep finding. The SLA exists because:

- Inbound moves fast — a customer who asked last Tuesday and got no acknowledgment defects.
- Stale intake breeds duplicate intake — the same signal comes in again because nobody acted on the first.
- Triage debt compounds; you can never catch up by working harder, only by sustaining the cadence.

## Anti-patterns

- **Pre-building a 30-category theme taxonomy.** It will be wrong. Let themes emerge from the signal.
- **Discarding source attribution on classification.** "Theme: csv export, segment: enterprise" without the ticket link is unverifiable; future agents cannot defend the priority.
- **Treating one loud customer as a pattern.** One enterprise account at risk is signal *for that account*; it is not the market.
- **Paraphrasing customer quotes during triage.** The verbatim quote is the trust signal. Preserve it.
- **Auto-rejecting without a reason.** Every reject carries `rejection_reason`. The customer-facing communication is a separate workflow, but the reason lives on the spec.
- **Guessing under uncertainty.** Escalate to `pm-investigator`. False classification compounds into bad prioritization.
- **Missing the 24h SLA without a workspace alarm.** Triage debt is the silent killer of an intake funnel.
