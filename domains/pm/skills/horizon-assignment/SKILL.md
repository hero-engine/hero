---
name: horizon-assignment
description: The now/next/later (or quarter) assignment discipline that keeps a roadmap honest — the concrete gates that make an item honestly "now," how to reconcile horizons against engineering reality, and how to catch aspirational drift.
metadata:
  audience: roadmap-curator, and the deferred roadmap-reviewer loader
  purpose: curation
---

## What I do

Supply the assignment discipline for roadmap horizons — the gate rules that decide whether an item is honestly `now`, `next`, or `later`, and the reconciliation policy that keeps those assignments true over time. `roadmap-framing` owns the *authoring* of an honest initiative (Bet / Evidence / Tradeoffs) and states the horizon criteria; this skill is the *curation* companion, focused specifically on assigning and re-assigning horizons as delivery reality moves. When `roadmap-curator` walks the board and decides what belongs in each lane, this is the rule set. It deliberately does **not** re-derive the horizon definitions — it cross-references `roadmap-framing` and adds the curation mechanics.

## When to use me

- assigning a horizon to a new or promoted initiative
- the weekly roadmap sweep — checking whether each item still earns its lane
- a delivery-state change, dependency unblock, or capacity event that should move a horizon
- catching `now`-lane bloat ("14 items in Now") and stale-now drift
- reconciling claimed horizon against what the cross-domain graph shows shipped

## What makes something honestly "now"

`roadmap-framing` states the full gate; the curation-critical version: an item is honestly `now` only when **all** hold —

- the bet has an approved PRD or pitch,
- at least one child spec is `in-review` or `delivering` (not all `planning`),
- the team has explicitly committed *this* planning cycle, and
- an engineering feature exists in the graph (or the work is single-domain PM).

It is **not** honestly `now` — and you should move it out — when it has no child specs, its children are all `planning`, the team hasn't actually committed (it's "what we hope to do next"), or it's sat in `now` for >2 planning cycles with no delivery movement.

The gate exists because `now` is the lane that costs trust. An aspirational `now` — an item parked there because someone *wants* it soon — turns the roadmap into a wish list, and once leadership learns `now` doesn't mean now, the whole board loses signal.

## `next` and `later` — the honest thresholds

- **`next`** — committed for the period after now. The bet is shaped (PRD/pitch exists, possibly draft), discovery is sufficient to know the shape of the work, capacity is assumed (not necessarily named engineers), and you could plausibly promote it to `now` at the next planning boundary. If you *couldn't* honestly move it up next cycle, it's `later`, not `next`.
- **`later`** — a real bet with supporting evidence, but not being actively shaped and not timing-urgent. Reorderable as priorities shift. `later` is not a graveyard — items with no evidence don't belong on the roadmap at all; they belong in intake or rejected-with-reason.

**Time-based variant (e.g. `q3-2026`):** same discipline. Current-quarter items must satisfy the `now` gate; future-quarter items must satisfy `next` or `later`. A quarter label is a horizon with a date, not an exemption from the gate.

## The assignment/reassignment rule

**Assign on evidence; reassign only on a grounded event.** Never shuffle horizons cosmetically. A horizon change is justified only by:

- a **delivery-state change** — a child spec moved to `in-review`/`delivering`/`completed`, or stalled;
- a **dependency event** — a blocker cleared or a new one landed;
- a **capacity event** — the team gained or lost the room the item assumed;
- a **strategic decision** — leadership re-bet, recorded in the initiative's rationale.

Every reassignment carries its grounding reason appended to the initiative. "Moved to Next — the SSO dependency it waited on shipped this week (`f-sso-saml` completed)" is grounded. "Moved to Now because it feels important" is cosmetic — refuse it.

## Reconciling against engineering reality

The curator's weekly sweep is where horizons stay honest. Walk the cross-domain graph and flag the mismatches (these are surfaced, never auto-corrected — the human decides, per `pm-agent-doctrine`'s decision-gate doctrine):

- **Stale-now** — `now` for >2 cycles with no committed child specs. Recommend demotion to `next` or a drop-with-reason.
- **Lying-now** — `now` but the graph shows no `in-review`/`delivering` child. The lane claims commitment the graph can't substantiate.
- **Ready-to-promote** — `next` whose children just moved to `in-review` and whose dependencies cleared. Recommend promotion to `now`.
- **Now-lane bloat** — more than ~5–7 items in `now`. A `now` lane of 14 means the team hasn't prioritized; recommend tightening.

## A worked reconciliation sweep

The curator walks a 5-item `now` lane against the graph. Findings, each surfaced as a proposal (never auto-applied):

| Item | Graph state | Gate check | Finding |
|---|---|---|---|
| `billing-self-serve` | 2 children `delivering`, PRD approved, committed this cycle | passes all 4 | ✅ honestly now — leave |
| `mobile-notifs` | children all `planning`, no commitment recorded | fails: no `in-review` child, not committed | ⚠️ **lying-now** — demote to `next` |
| `sso-saml` | 1 child `in-review`, dep `f-idp-config` just `completed` | now passes (dep cleared) | ✅ keep — note the grounded reason |
| `export-v2` | in `now` 3 cycles, zero children | fails: stale, no children | ⚠️ **stale-now** — demote or drop-with-reason |
| `dashboard-redesign` | children `planning`, no PRD | fails: no approved PRD | ⚠️ demote to `next` |

Plus a lane-level finding: **5 → but 3 don't earn `now`**, so the real committed lane is 2. That's healthy width, not bloat — the problem was aspirational parking, not over-commitment.

The curator writes each demotion as a proposal with its grounding ("`export-v2`: 3 cycles in `now`, no child specs — recommend drop-with-reason or demote to `later`"), logs the sweep, and lets the PM decide. Nothing moves silently.

## Anti-patterns

- **Aspirational now.** Parking an item in `now` because it's wanted soon, with no committed child spec. The core rot.
- **Cosmetic reshuffling.** Moving horizons without a delivery/dependency/capacity/strategy event behind it. Every move needs a grounded reason on the record.
- **Now-lane bloat.** A `now` lane nobody could actually deliver this cycle. If everything is now, nothing is.
- **Stale-now rot.** Letting an item sit in `now` for cycles with no movement. Demote or drop with reason — don't let it lie there.
- **`later` as a graveyard for un-evidenced ideas.** `later` still requires a real bet with evidence; un-evidenced ideas belong in intake, not on the roadmap.
- **Silent demotion.** Moving an item down without a reason the PM and stakeholders can see. Reassignments are proposals with rationale, not quiet edits.

## Cross-references

- `roadmap-framing` — owns the horizon *definitions* and the honest-initiative authoring bar; this skill is its curation companion (assign/reassign mechanics), not a duplicate.
- `cross-domain-graph-query` — how the curator reads live delivery state to reconcile horizons.
- `dependency-mapping` — dependency events are a primary grounded reason for reassignment.
- `pm-agent-doctrine` — horizon changes are surfaced suggestions with rationale, never silent auto-corrections.
- Prior art: ProdPad Now-Next-Later (Janna Bastow); Shape Up appetite/betting for cycle-preset teams.
