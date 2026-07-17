---
name: stakeholder-communication
description: Audience-shaped messaging — cut the same PM artifact differently for exec, customer, engineering, and sales; pressure-test every exec line against "so what"; know when to reach for a working-backwards (PR-FAQ / 6-pager) narrative without reproducing it.
metadata:
  audience: stakeholder-communicator
  purpose: audience-shaped-messaging
---

## What I do

Provide the discipline for turning one PM artifact into the right message for the audience who has to act on it. The facts don't change between cuts — the *lead*, the *omissions*, and the *frame* do. A PRD read by an exec, a customer, an engineer, and a sales rep is four different documents drawn from one source of truth. This skill packages the rules for making those cuts land without distorting the underlying truth.

## When to use me

Load this skill when:

- summarizing a PM artifact for a specific audience (exec review, customer announcement, standup, sales enablement)
- preparing a leadership update or a board / QBR narrative
- deciding what to lead with and what to omit for a given reader
- pressure-testing a draft update for filler that won't survive an executive's "so what?"
- deciding whether a moment calls for a full working-backwards narrative (and where that format actually lives)

## The four audience cuts

The same artifact, cut four ways. For each: what they want, what to lead with, what to omit.

| Audience | What they want | Lead with | Omit |
|---|---|---|---|
| **Executive** | Outcome and the tradeoff behind it | The measurable outcome, the bet, what you gave up to make it | Feature enumeration, implementation detail, process narration |
| **Customer** | Capability and timing | What they can now do, when it's available, what changes for their workflow | Internal spec slugs, org mechanics, roadmap hedging |
| **Engineering** | Context and acceptance criteria | Why the work matters, the AC, the constraints and dependencies | Marketing framing, exec-level outcome abstraction without the concrete "done" line |
| **Sales** | Talking points that map to a buyer need | The capability tied to the objection or need it answers, availability | Caveats phrased as engineering risk; anything that reads as an internal hedge |

The failure mode is one cut serving all four. A single blast that mixes exec outcome-talk, customer benefit-talk, and engineering AC serves none of them — each reader has to dig past three-quarters of noise to find their quarter.

## The "so what" pressure-test

**Every exec-facing line must survive the question "so what?" — tie the statement to an outcome, or cut it.**

Executives read for consequence. A line that states an activity without naming its result is filler that buries the one line that matters. Run each sentence through the test:

| Fails "so what?" | Survives "so what?" |
|---|---|
| "We shipped the export refactor." | "Export now completes in under 2s (was 11s); the three enterprise accounts that escalated latency in Q2 are unblocked." |
| "The team ran five customer interviews." | "Five interviews killed the configurable-export bet — users want a one-click dump, saving us a big-batch cycle." |
| "We're making good progress on onboarding." | "Onboarding activation moved 12% → 18% week-over-week; on track for the 25% target by cycle end." |

If a line can't be tied to an outcome, a tradeoff, or a decision the reader has to make, it doesn't belong in the exec cut. This is `outcomes-over-outputs` applied at the sentence level.

## Working-backwards awareness (PR-FAQ / 6-pager)

Amazon's working-backwards method — the **PR-FAQ** (a press release plus anticipated FAQ, written *before* building) and the **six-page narrative** (prose, not slides, read in silence at the start of the meeting) — is the right tool when a bet needs a full narrative case: a major launch, a strategic pivot, a leadership decision that turns on the story rather than a status line.

**This skill names the pattern and tells you when to reach for it. It does not reproduce the format.** The full PR-FAQ / narrative mechanics — the section-by-section structure, the FAQ discipline, the prose-over-bullets rule — live in `exec-narrative` (authored by child #9). When a moment calls for working-backwards, cross-reference `exec-narrative` as the home for the format; don't rebuild it inline here. Audience-shaped *cuts* (this skill) and the *full narrative artifact* (`exec-narrative`) are different jobs: reach for the narrative when a one-page exec cut can't carry the decision.

## Anti-patterns

- **One cut for all audiences.** A single summary sent to exec, customer, and engineering. Each needs a different lead; the blast serves none.
- **Sandbagging timing.** Quoting a conservative date to one audience and an aggressive one to another for the same work. Shape the framing, never the facts.
- **Marketing-flavor everything.** Turning a standup or an internal update into launch copy. Internal cuts are plain and specific.
- **Filler in the exec cut.** Activity lines that don't survive "so what?" — they bury the consequence that matters.
- **Reproducing the PR-FAQ format here.** Reach for `exec-narrative` when you need the full narrative; this skill points to it, it doesn't duplicate it.

## Cross-references

- `outcomes-over-outputs` — the "so what" test is outcome-framing at the sentence level; the exec cut leads with the outcome this skill defines.
- `release-notes-writing` — the customer and internal release-note shapes are two specific audience cuts with their own conventions.
- `pm-agent-doctrine` — no fabricated quotes or metrics in any cut; a cut is a proposal of how to say it, not a new fact.
- `exec-narrative` (child #9) — the home for the full PR-FAQ / 6-pager working-backwards format; this skill names the pattern and defers the mechanics there.
