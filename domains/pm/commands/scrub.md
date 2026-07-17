---
description: Scrub a workspace concern for accumulated quality issues — concern-dispatched. The `intake` concern clusters recent intake to surface missed near-duplicates; `roadmap` sweeps the roadmap for stale / mislabeled items and `stories` sweeps `ready` stories for INVEST/EARS failures. All three are report-only.
---
Route this scrub request by its `<concern>` argument to the concern-specific agent.

`/scrub <concern>` runs a **background/batch sweep** for one concern and surfaces findings + recommended actions. It is **report-only**: scrub concerns never auto-apply state changes — no merge, no state flip, no roadmap rewrite. The decision stays with the human (decision-gate doctrine).

## Pre-flight: read the concern

Read the `<concern>` argument from `$ARGUMENTS`.

- **No argument** → list the available concerns (below) and ask which to run. Do not guess.
- **Unknown concern** → say so and list the available concerns.

<!-- SCRUB CONCERNS: intake (this child, story-detail-and-intake-scrubber-backing);
     roadmap + stories appended BELOW by remaining-roles-scrubbers-and-launch (#11).
     #11 APPENDS new concern blocks only — do not edit the intake block above. -->

### Concern: `intake`

`/scrub intake` → `duplicate-intake-scrubber`.

Cluster a window of recent intake (default: the `new` / untriaged queue, or the last N days when a window is given) to surface near-duplicates the write-time `duplicate-detector` structurally could not catch — dups that only become visible across a batch after vocabulary drift, cross-segment paraphrase, or accumulated volume. Emit the **cluster report**: ranked clusters, per-cluster confidence, the specific field overlap behind each, and a recommended canonical survivor per cluster. **No auto-merge** — merges are recommendations the human confirms; source attribution is preserved. Complements (does not replace) the live write-time detector.

<!-- #11 APPEND POINT — remaining-roles-scrubbers-and-launch adds new concern
     blocks (### Concern: `roadmap` → stale-roadmap-scrubber; ### Concern:
     `stories` → ambiguous-story-scrubber) BELOW this line. Append only;
     do not edit the intake block above. -->

### Concern: `roadmap`

`/scrub roadmap` → `stale-roadmap-scrubber`.

Sweep the roadmap for items that have gone stale or fallen out of sync with delivery reality: roadmap-items with **no movement in N weeks** (default N weeks; widen when a window is given), **shipped-but-active** items still marked `committed`/`now` while their child specs are all `completed` in the graph, and **over-horizon `later`** items parked past the planning horizon. Cross-check against live cross-domain delivery state, not the tracker. Emit the **flag report**: each flagged item with the specific age/movement/state signal behind it, its current horizon + status, and a recommended action (`archive` / `drop with reason` / `refresh`); shipped-but-active (roadmap-lying) flags called out first. **Report-only — no auto state flip**; the human confirms every state change. Explicit "no stale items found" when clean.

### Concern: `stories`

`/scrub stories` → `ambiguous-story-scrubber`.

Sweep stories at `status: ready` (the pool a planning cycle pulls from) for ones that **fail INVEST** or **lack EARS acceptance criteria** — the ambiguity that causes friction at handoff. Skip non-`ready` drafts (they're expected to be rough). Emit the **flag report**: each flagged `ready` story with its *specific* failure (missing/untestable AC, too large / not Small, not Independent, not Estimable) and the recommended refinement + the agent that owns it (`story-writer`, or `epic-framer` when it's really an unframed epic). **Report-only — no auto edit**; the refinement is human-gated authoring. Explicit "no ambiguous stories found" when clean.

## Dispatch

Dispatch the concern argument to its concern-specific agent (the mapping above), pass the scope through, and let that agent produce its report. Each concern owns one agent; the agent runs the sweep and surfaces findings + recommended actions read-only. The human decides what to act on.

Request: $ARGUMENTS
