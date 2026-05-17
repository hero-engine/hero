---
name: dependency-mapping
description: Walk the cross-artifact dependency graph forward and backward, distinguish hard blockers from soft sequencing, and surface cross-domain chains.
compatibility: opencode
metadata:
  audience: dependency-mapper, roadmap-curator, prioritization-strategist, cycle-planner
  purpose: curation
---

## What I do

Provide the rules for analyzing dependencies across PM artifacts (and across the PM-engineering boundary). Distinguish **hard blockers** (A cannot start until B ships) from **soft sequencing** (A is more efficient if it follows B) from **coupling** (A and B touch the same code or contract but neither blocks the other). Walk the graph forward ("what does this block?") and backward ("what blocks this?") through transitive chains, including cross-domain edges.

## When to use me

Load this skill when:

- planning a cycle / sprint / release and need to know what can actually start (`cycle-planner`, `capacity-planner`)
- curating the roadmap and reconciling which `committed` items are actually unblocked (`roadmap-curator`)
- prioritizing — a high-value item blocked by a P3 prerequisite needs its real start date surfaced (`prioritization-strategist`)
- a PM asks "what's blocking story X?" or "what does initiative Y unblock?"

## Hard blocker vs soft sequencing vs coupling

### Hard blocker

A cannot start until B ships. If you try, A halts partway and produces nothing usable.

Examples:
- Story A needs the `payments-v2` API. Story B is the engineering feature delivering `payments-v2`. A is hard-blocked on B.
- Initiative "EU launch" is hard-blocked on initiative "data residency compliance."

Hard blockers are first-class — they constrain start dates and capacity math directly.

### Soft sequencing

A *can* start before B, but it's more efficient to wait. Doing A first means rework when B lands.

Examples:
- Story A is "redesign the checkout button." Story B is "refactor the checkout component." A could happen first, but you'd redo the button placement after the refactor.
- Initiative "marketing site refresh" is soft-sequenced after "new brand guidelines" — you can launch with the old brand and refresh later, but it's wasted effort.

Soft sequencing is a *recommendation*, not a constraint. Surface it but don't enforce it. The PM may have a reason to take the rework cost (e.g. a deadline).

### Coupling (not a dependency)

A and B touch the same code, contract, or workflow but neither blocks the other. They can ship in either order; doing them together may be cheaper than doing them apart.

Examples:
- Two stories both modifying the dashboard layout — coordinate to avoid merge conflict, but neither blocks the other.
- Two initiatives both touching pricing — share research, but distinct decisions.

**Do not record coupling as a dependency.** Coupling is a coordination concern, not a sequencing constraint. Mislabeling coupling as dependency creates phantom blockers and slows the roadmap.

## Walking the graph

### Forward walk — "what does this block?"

Start at the target node and traverse outbound `blocks` and `blocked-by` edges (inverted), plus `parent` edges (a blocked epic blocks its child stories).

Forward walks answer "if this slips, what slips with it?" Critical for risk analysis on `committed` initiatives. If the slipping item has a long forward chain, the roadmap impact is amplified.

### Backward walk — "what blocks this?"

Start at the target and traverse inbound `blocks` edges. Stop when you hit either a `done` / `shipped` node (resolved) or a `dropped` node (resolved differently — the chain may need re-evaluation).

Backward walks answer "can this actually start now?" Critical before flipping a story to `in-flight` or an initiative to `committed`. A `ready` story with an open hard blocker is not actually ready.

Use `hero_why` to walk back from any node — it traces the full chain of decisions, specs, and edges that produced the current state.

### Transitive chains

Dependencies chain. A blocks B; B blocks C; therefore A blocks C transitively. Surface the full chain, not just the direct edge — a story whose direct blocker is "ready" but whose transitive blocker is "in-flight engineering work" has a real start date weeks out.

The rule: **walk to terminal state.** Stop only at `done` / `shipped` / `dropped` or at a node with no outbound blockers.

## Cross-domain dependencies

The most common chain in Hero:

> PM `story` → blocked by → engineering `feature` → blocked by → another engineering `feature`

This chain crosses the domain boundary twice. Use the `cross-domain-graph-query` skill to traverse cross-namespace edges. The chain renders with the domain boundary visible:

```
story:enable-saml [refined]
  blocked-by feature:saml-provider [delivering]
    blocked-by feature:auth-refactor [planning]
```

The PM-side reading of "this story is blocked" requires reading engineering-side delivery state — there is no other source of truth. The tracker silos by project; the graph is the cross-domain truth surface.

### Hard blocker that's unblocked but slow

A common case: the blocker (engineering feature) is `delivering` — not done, but unblocked itself. The dependent story is technically "still blocked," but the blocker has a real ETA.

Surface this as **"blocker in progress, ETA <date>"** rather than as a binary blocked/unblocked. `cycle-planner` and `capacity-planner` use the ETA to decide whether the dependent can be committed to the next cycle.

If the blocker has been `delivering` for multiple cycles with no progress signal (no commits referencing it, hill chart stuck), flag it as a `roadmap-curator` finding — the chain is at risk.

## What to do when a hard blocker is dropped

The dependent's dependency edge needs re-evaluation. Three outcomes:

1. **The need still exists** — find a different way to unblock it (new feature, scoped-down alternative, accept partial capability).
2. **The need evaporated** — the dependent should likely be dropped too. Don't carry zombie work.
3. **The need transforms** — the dependent gets rewritten against the new context.

`roadmap-curator` should surface dropped blockers as a weekly finding so the dependent doesn't quietly stay on the roadmap with a broken upstream.

## Anti-patterns

- **One-hop-only dependency analysis.** Walking the direct edges misses transitive chains; a story whose direct blocker is "ready" can still be effectively blocked weeks out.
- **Mislabeling coupling as dependency.** Creates phantom blockers; slows the roadmap.
- **Ignoring soft sequencing entirely.** Soft sequencing missed becomes rework taken; surface it.
- **Treating hard blockers as binary.** A blocker that's `delivering` has an ETA — surface the ETA, not just "blocked."
- **Dependencies that skip the cross-domain edge.** A PM story blocked by an engineering feature must be modeled as a cross-domain dependency — not a vague "waiting on engineering" note in the body.
- **Leaving zombie dependencies after a blocker is dropped.** Re-evaluate dependents whenever a blocker terminates.
- **Trusting tracker state over graph state for dependencies.** The tracker silos by project and cannot express cross-domain edges. The graph is the truth surface.
