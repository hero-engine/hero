---
type: convention
slug: default-to-action-when-judgment-is-clear
status: active
domain: engineering
scope: ["**/agents/**", "**/skills/**"]
tags: [agent-reliability, decision-discipline, process]
created: 2026-06-01
---

# Default to action when judgment is clear; menus only for real ambiguity

## Rule

When the right next step is clear from the evidence already in front of the
agent, **execute it**. Do not chain (a/b/c) decision menus that force the
user to re-decide what the agent could have decided itself.

A menu is appropriate when:
- Multiple paths have meaningfully different trade-offs the agent can't
  resolve from context (taste, business priority, risk appetite)
- The user has stated they want to be in the loop on a category of
  decisions
- The action is destructive or irreversible

A menu is **inappropriate** when:
- One option is clearly correct from the evidence
- The "menu" is a re-pose of a question the agent already has enough
  context to answer
- The user already authorized this kind of action in this session

## Anti-pattern

```
Agent: Here are the 6 specs with drift. Options:
  (a) Bump 3 of them to medium
  (b) Bump everything to large
  (c) Bump only some
Which do you want?

User: a

Agent: OK done. Now 2 more drifts surfaced. Options:
  (a) Bump 2 more
  (b) Stop
  (c) Spawn a follow-up
Which?

User: we need our process to not end up with 2 people digressing back and forth
```

The first menu was right (real choice). The second menu was friction —
the agent had enough context to make the call (bump the 2 that are genuine
under-sizing, leave the rest, spawn the follow-up for the calibration
issue). Should have just done it and reported.

## Right pattern

```
Agent: 6 specs drifted. Three are real under-sizing, three are
       calibration noise (computed heuristic over-weights prose). My
       proposed action: bump these 3, leave these 3. Going?

User: yeah

Agent: [executes; reports outcome including a new finding]
       Done. Bumps revealed a cascade — roadmap-shape rollup now wants
       x-large. Bumping that too; it matches the actual scope. Also
       spawned a follow-up to tune the computed heuristic since the
       calibration noise is systemic, not per-spec. Session record
       written. 5 calibration-noise drifts left; ambient surface goes
       quiet for 24h.
```

One ask, one execute, one honest report (including new findings).
Not three round-trips.

## Source

Surfaced during the dogfooding `/roadmap-review` session on 2026-06-01.
User push: "we need our process to not end up with 2 people digressing
back and forth." Captured into the roadmap-review session record at
`.hero/knowledge/roadmap-review-sessions/2026-06-01-1530.md` and lifted
to a convention here for cross-agent applicability.

## Applies to

All coordination agents — feature-delivery-lead, platform-delivery-lead,
roadmap-reviewer, design-reviewer. Especially relevant when:
- Walking a queue of drift / findings / candidates
- Surfacing follow-on findings during execution
- Post-action validation
