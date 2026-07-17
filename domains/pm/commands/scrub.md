---
description: Scrub a workspace concern for accumulated quality issues — concern-dispatched. The `intake` concern clusters recent intake to surface missed near-duplicates; `roadmap` and `stories` ship via child #11.
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

## Dispatch

Dispatch the concern argument to its concern-specific agent (the mapping above), pass the scope through, and let that agent produce its report. Each concern owns one agent; the agent runs the sweep and surfaces findings + recommended actions read-only. The human decides what to act on.

Request: $ARGUMENTS
