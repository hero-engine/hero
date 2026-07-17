---
name: epic-framing
description: Frame an epic as a coherent bet, not a bag of stories — a Why that names the shared outcome, a rollup acceptance line the child features add up to, and sequenced children with a real ordering rationale. If the children don't share an outcome, it isn't an epic.
metadata:
  audience: epic-framer (Wave-3)
  purpose: framework-guidance
---

## What I do

Supply the discipline that keeps an epic from degenerating into a label over a pile of unrelated work. An epic is the mid-tier grouping between an initiative (a strategic bet) and a feature (a dev-ready unit) — and its whole value is *coherence*: the child features share one outcome, add up to a rollup that's more than their sum, and sequence in an order that reflects real dependency and value. The common failure is a "bag of stories" — a grab-bag tagged with an epic name because they're vaguely related, with no shared outcome and no rollup. This skill gives the three-part frame (Why / rollup AC / sequenced children) that makes an epic a coherent bet. Forward-authored for the Wave-3 `epic-framer` agent.

## When to use me

- framing a new epic from an initiative or a large PRD
- deciding whether a set of features is genuinely one epic or should split
- reviewing an epic for coherence (does it earn the grouping?)
- sequencing an epic's child features with a real ordering rationale
- a story-writer flags "this is really an epic" on an over-large feature

## The coherence test

Before writing anything, apply the test: **do these child features share one outcome such that delivering some-but-not-all still moves that outcome?** If yes, it's an epic. If the features are related only by topic ("all the billing stuff") with no shared outcome, it's a *label*, not an epic — either find the real shared outcome or split it into separate initiatives/features.

An epic is a **coherent bet at mid-altitude**: coarser than a feature (it needs decomposition to deliver), finer than an initiative (it's a specific slice of a strategic bet, not the whole bet).

## The three-part frame

### 1. Why — the shared outcome

The epic opens with the outcome its children collectively move, in outcome terms (see `outcomes-over-outputs`), not a feature list:

> **Why:** New accounts abandon during data import because our importer only handles one format and gives no progress feedback. This epic makes first-import succeed unattended for the top 3 formats — targeting a lift in activation (import-completed within 24h of signup) from 54% to 75%.

The Why is the coherence anchor: every child feature must plausibly move *this* outcome. A child that doesn't belong to this Why belongs to a different epic.

### 2. Rollup acceptance — what the children add up to

The epic carries a **rollup acceptance line**: the epic-level "done" that the child features' individual AC sum to. It's not a copy of the children's AC — it's the aggregate condition that means the *bet* landed:

> **Rollup AC:** A new-account user can import CSV, JSON, and Excel files up to 500k rows, sees live progress, and recovers from a failed row without restarting — end to end, unattended, in under 5 minutes for a typical file.

The rollup is how you know the epic is complete beyond "all child stories closed" — it names the integrated behavior. If the children can all be `done` while the rollup is still false, the decomposition is wrong (a child is missing).

### 3. Sequenced children with a rationale

List the child features **in delivery order**, each with a one-line reason for its position — dependency, value-first, or risk-first:

> 1. `import-csv-core` — the format 80% of users need; ships value first and de-risks the pipeline.
> 2. `import-progress-feedback` — depends on the core pipeline existing; addresses the top abandonment cause.
> 3. `import-json` / `import-excel` — parallel once the pipeline generalizes; lower-volume formats.
> 4. `import-row-recovery` — hardening; sequenced last because it's only valuable once real files flow.

The sequencing rationale is what distinguishes an epic from a checklist. "Ships value first," "depends on #1," "hardening, do last" are real reasons. A children list with no ordering logic is a bag of stories with numbers.

## A worked contrast — bag of stories vs. coherent epic

**Bag of stories (fails the coherence test):**
> **Epic: Billing improvements**
> - Add annual billing option
> - Fix invoice PDF layout bug
> - Refactor the payment retry logic
> - Add usage-based pricing
> - Update the billing settings copy

These share a *topic* (billing), not an *outcome*. Delivering "fix invoice PDF" and "add annual billing" moves nothing in common — there's no rollup they sum to, no Why that constrains what belongs (why not "add a payments dashboard" too?). This is a label. It should split: the PDF bug is a standalone `bug`; the retry refactor is a `chore`; annual + usage-based pricing might be their *own* coherent epic if they share a revenue outcome.

**Coherent epic (passes):**
> **Epic: Self-serve plan upgrades**
> **Why:** SMB accounts that outgrow their plan today have to email sales to upgrade, and ~40% never do — they churn or stay under-served. This epic makes upgrading self-serve, targeting SMB expansion revenue (self-serve upgrades/quarter) from ~0 to a measurable baseline.
> **Rollup AC:** an SMB admin can compare plans, upgrade mid-cycle with prorated billing, and see the new limits take effect immediately — with no sales touch.
> Children (sequenced): 1. `plan-comparison-view` (value-first, no deps) → 2. `prorated-upgrade-flow` (the core bet; depends on #1) → 3. `immediate-limit-application` (depends on #2) → 4. `upgrade-confirmation-comms` (hardening, last).

Every child moves the *same* outcome; the rollup names an integrated behavior none delivers alone; the sequence has real rationale. That's an epic.

## Splitting: when an epic is too big or incoherent

- **Too big** — if the epic would take more than a couple of cycles or has 15+ children, it's probably two epics or a small initiative. Split on the *sub-outcome* seam, not arbitrarily.
- **Incoherent** — if you can't write one Why that all children serve, the grouping is topical, not outcome-based. Split into features under their real parents.
- **Rollup can't be written** — if there's no integrated behavior the children sum to, they're independent features that happen to share a tag; don't force an epic over them.

## Anti-patterns

- **Bag of stories.** An epic name over unrelated features sharing only a topic. If there's no shared outcome, it's a label — split it.
- **Missing Why.** An epic that opens with a feature list and no outcome. The Why is the coherence anchor; without it, nothing constrains what belongs.
- **No rollup AC.** "Done when all child stories are done" — that's not an epic acceptance line, it's a status check. Name the integrated behavior the children sum to.
- **Unsequenced children.** A children list with no ordering rationale. Sequence on dependency / value-first / risk-first, and say which.
- **Epic as a phase label.** "Q3 Epic" / "Phase 2" — that's a bucket, not a bet. Epics are outcome-coherent, not calendar-coherent.
- **Epic that's really an initiative.** If the "epic" is a whole strategic bet needing its own evidence and tradeoffs, it's an initiative; frame it with `roadmap-framing` instead.

## Cross-references

- `outcomes-over-outputs` — the Why is stated at the outcome rung; the epic is a coherent bet, not a feature bundle.
- `roadmap-framing` — how the parent initiative frames the strategic bet the epic slices; use it when the "epic" is really initiative-sized.
- `story-writing-invest` — each child feature must still pass INVEST independently; the epic sequences them, it doesn't excuse un-INVEST children.
- `dependency-mapping` — the sequencing rationale draws on the real dependency graph among the children.
