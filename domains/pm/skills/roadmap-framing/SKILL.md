---
name: roadmap-framing
description: How to write initiatives that read honestly — bet-shaped framing, evidence-grounded rationale, named tradeoffs, and horizon assignments that reconcile against engineering reality.
metadata:
  audience: product-strategist, roadmap-curator, pm-reviewer
  purpose: roadmap-authoring
---

## What I do

Provide the authoring discipline for initiatives — the coarsest PM artifact, the unit of the Roadmap board, and the surface where principle #3 (tradeoffs visible) and principle #4 (alignment maintained) live or die. Initiatives that read honestly let leadership trust the roadmap; initiatives that fudge horizons or omit tradeoffs make the roadmap a fiction.

## When to use me

Load this skill when:

- drafting a new `initiative` spec (`product-strategist`)
- curating the roadmap board (`roadmap-curator`) — horizon assignments, stale-item surfacing, shipped-vs-claimed reconciliation
- reviewing the roadmap as a whole (`roadmap-reviewer`, `pm-reviewer`)
- coaching a team on how to keep their roadmap from lying
- reconciling claimed delivery state against engineering reality via the cross-domain graph

## Initiatives as bets, not projects

An initiative is a **bet** — a hypothesis about an outcome, with the evidence behind it and the tradeoffs it implies.

The three required sections (Bet, Evidence, Tradeoffs) operationalize this. Each is load-bearing.

### Bet — hypothesis → expected outcome → evidence

The Bet section names what we believe will happen if we ship, and by how much. Outcome-shaped, not output-shaped.

**Output-shaped (fails):** "Build CSV export."

**Outcome-shaped (passes):** "We believe a one-click CSV export will reduce ops-team escalations about manual data pulls by at least 40% within one quarter of launch. The 14 escalations we logged in Q1 represent ~6 hours of ops time per week."

The pattern:

```
[hypothesis] → [expected outcome, quantified where possible] →
[evidence grounding the hypothesis]
```

This is the bet-shaping discipline. Read more in Marty Cagan / SVPG's outcomes-over-outputs framing (*Inspired*, *Empowered*).

### Evidence — without source attribution, the bet is opinion

The Evidence section grounds the bet in something other than the PM's intuition. Sources include:

- **Linked intakes** — customer requests, support escalations, sales feedback, with source attribution preserved (which customer? which segment? what date?).
- **Research notes** — discovery interviews, usability studies, opportunity-tree synthesis (`opportunity-solution-trees-torres`).
- **Competitive signals** — what comparable products ship, with the source of the observation.
- **Quantitative data** — usage metrics, funnel analysis, support ticket volume.
- **Strategic context** — board-level direction, regulatory deadlines, contractual obligations.

Without source attribution, the bet is the PM's opinion in a frame. With sources, others can challenge or strengthen the bet.

### Tradeoffs — what we're deferring to do this

The Tradeoffs section names what doesn't get built because this gets built. **This is where principle #3 (tradeoffs visible) becomes concrete.** An initiative without Tradeoffs implies the work has no opportunity cost — which is never true.

**What passes:**

```
## Tradeoffs

Committing this cycle defers the SSO improvements (rmI-sso-mfa)
that the enterprise segment has been waiting on. We believe the
ops time savings here are larger than the deal-acceleration impact
of SSO this quarter, but it's a real tradeoff and we should
revisit if a deal blocks on SSO.

Changing course mid-build would cost ~2 weeks of partial work
and re-introduce the manual-pull friction the bet is designed
to remove.
```

**What fails:**

- Empty Tradeoffs (most common failure — leaves the cost invisible).
- "We'll do both eventually" (not a tradeoff — that's a deferral with no opportunity cost named).
- Tradeoffs that name the wrong axis (the cost of *building* is not the tradeoff; the cost of *not building something else* is).

## Horizon assignment — what makes something honestly "Now"

Under the horizon preset (almost always on), every initiative carries a `horizon` value: **now / next / later** (or a specific quarter for time-based teams).

The horizon discipline is what keeps the roadmap honest. Common rot pattern: items drift into `now` aspirationally, sit there for a quarter without movement, and the roadmap becomes a wish list.

### `now` — committed, in or starting this period

An item is honestly `now` when **all** of these hold:

- The bet has a PRD (or a pitch) drafted and approved.
- At least one child story is `in-review` or `delivering` (engine statuses; see the lifecycle table in `pm-preset-detection`).
- The team has explicitly committed in the current planning cycle (sprint plan, cycle bet, release plan).
- An engineering feature exists (cross-domain edge populated) OR the work is single-domain PM.

An item is *not* honestly `now` if:

- It has no child stories.
- Its child stories are all `planning` (none yet `in-review`).
- The team hasn't committed — it's "what we hope to do next."
- It's been `now` for >2 planning cycles without delivery progress (use `roadmap-curator` to surface).

### `next` — committed for the period after now

- The bet is shaped (PRD or pitch exists, possibly in draft).
- Discovery is sufficient — we believe we know the shape of the work.
- Capacity is reserved (not necessarily named-engineers, but planning has assumed it'll happen).
- We could plausibly move it to `now` at the next planning boundary.

### `later` — candidate or committed for future; not currently being shaped

- The bet exists as an initiative but isn't being actively shaped.
- Evidence supports the bet's importance but timing isn't urgent.
- Could be reordered as priorities shift.

### Time-based horizons (e.g. `q3-2026`)

Some teams use quarter strings instead of now/next/later. Same discipline applies: items in the current quarter must satisfy the `now` criteria; items in future quarters must satisfy `next` or `later`.

### Reconciling against engineering reality

`roadmap-curator` runs weekly and walks the cross-domain graph:

- If all child stories are `done` and the cross-domain feature is `delivered`, status → `shipped` automatically, `shipped_at` populated.
- If a `now` item has no committed child stories after 2+ cycles, surface as stale; recommend drop with reason or refresh.
- If a `committed` item has cross-domain features in `delivering` state but no PM-side movement (no new linked stories, no PRD updates), surface in standup.
- If "shipped" is claimed in the roadmap but no satisfying cross-domain edge exists, surface as a lie — the roadmap claims completion the graph can't substantiate.

This is principle #4 (alignment maintained) operationalized. The roadmap reconciles against reality; if it doesn't, the curator surfaces the drift.

## Rationale — "why this and not that"

The optional `rationale` frontmatter field is the strategic counterpart to Tradeoffs. Where Tradeoffs names *what's deferred*, Rationale names *why this bet won the slot*.

**What passes:**

> Rationale: We picked CSV export over SSO/MFA improvements because (a) the
> ops time savings affect every team daily, while SSO is a deal-acceleration
> tool that surfaces episodically; (b) export is single-cycle work and SSO
> is multi-cycle; (c) two enterprise prospects in pipeline are SSO-blockers
> but neither is closing this quarter. Revisit if pipeline shifts.

**What fails:**

- Missing Rationale on a `committed` item. (Reviewers can't challenge a bet whose reasoning is invisible.)
- Rationale that restates the Bet section. (Tautology.)
- Rationale that's actually a Tradeoff. (Different field, different purpose.)

Rationale is first-class on every `committed` initiative. `pm-reviewer` flags missing Rationale as a blocking finding on the `candidate → committed` transition.

## Output vs outcome — the framing distinction

The single most common roadmap-framing failure: items framed as outputs (things we'd build) instead of outcomes (things that would change).

| Output-framed (fails) | Outcome-framed (passes) |
|---|---|
| "Build CSV export feature" | "Reduce ops escalations for manual data pulls" |
| "Redesign onboarding" | "Lift D7 retention for new signups from 22% to 30%" |
| "Add SSO" | "Unblock enterprise deals stalled on SSO/MFA requirements" |
| "Mobile app v2" | "Capture the 31% of returning users who currently bounce on mobile" |

The output is implicit in the outcome — the team will build *something* to move the outcome. But the bet is on the outcome, not the build. This matters because:

- Outcomes can be measured; outputs can only be shipped. Principle #5 (learn from what shipped) requires outcomes.
- Outcomes leave engineering room to propose better outputs. An initiative framed as output forecloses engineering's contribution.
- Outcomes survive scope changes; outputs don't. If the build morphs during discovery, an outcome-framed item still makes sense.

## How `roadmap-curator` reads the graph weekly

The curator's weekly sweep walks every initiative and applies the reconciliation rules above. It produces:

- **Auto-state-changes** (with notification, not silent): `committed → shipped` when graph confirms delivery.
- **Stale-now warnings**: items in `now` for >2 cycles without movement.
- **Lying-shipped warnings**: items in `shipped` without satisfying graph edges.
- **Lonely-committed warnings**: `committed` items with no child stories or PRDs.
- **Orphan-feature warnings**: cross-domain features with no parent initiative (engineering shipped something the roadmap doesn't claim).

These are not auto-corrected. The curator surfaces them; the PM decides. (Auto-correcting state would let the system silently rewrite the PM's claims, which violates the "PM owns the decisions" principle from the mission.)

## Anti-patterns

- **Items that are project names.** "Mobile App v2." "Q3 Initiatives." Not bets — labels. Refuse the draft; ask "what outcome are we betting on?"
- **`Now` lists that never explicitly defer anything.** Tradeoffs section empty across every `now` item. The roadmap reads as a wish list of equal-priority items; the PM hasn't done the prioritization work.
- **"Quarterly priorities" as items.** "Q3 Priorities" isn't an initiative — it's a label. Initiatives are bets; labels are tags.
- **Themes as items.** "Performance." "Reliability." "AI." Themes are lanes (use the `themes` frontmatter field); items are bets within lanes.
- **Items with no Evidence.** The bet has no grounding — it's opinion in a frame. Refuse the `candidate → committed` transition.
- **"Shipped" without graph edge.** Status claims completion the cross-domain graph can't substantiate. The roadmap is lying.
- **Items stuck in `committed` for 2+ cycles with no child stories.** Either drop with reason or escalate; don't let stale items rot in `committed`.
- **Rationale = "because the team agreed."** Not a rationale — that's restated agreement. Rationale names the *why* behind the agreement.
- **Outcome metrics that can only be measured a year out.** The bet can't be evaluated within a reasonable feedback loop. Either reframe to a leading indicator or accept that this is a long-horizon bet with no near-term learning.

## Cross-references

- `prioritization-frameworks` — RICE / ICE / WSJF mechanics for picking between initiatives.
- `cross-domain-graph-query` — how `roadmap-curator` reads engineering delivery state.
- `evidence-synthesis` — how to ground the Evidence section.
- `opportunity-solution-trees-torres` — discovery framework that often feeds initiatives.
- `prd-structure` and `pitch-writing-shape-up` — once an initiative promotes to `committed`, a PRD or pitch follows.
- PM domain mission — principle #3 (tradeoffs visible), principle #4 (alignment maintained), principle #5 (learn from what shipped) all live here.
- Prior art: Marty Cagan / SVPG (*Inspired*, *Empowered*) on outcome-shaped roadmaps; Teresa Torres on opportunity-grounded bets.
