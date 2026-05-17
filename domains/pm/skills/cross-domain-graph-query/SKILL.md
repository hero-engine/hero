---
name: cross-domain-graph-query
description: Walking the knowledge graph across domain boundaries — from PM artifacts into engineering reality and back. The graph, not the tracker, is the cross-domain truth surface.
compatibility: opencode
metadata:
  audience: roadmap-curator, handoff-coordinator, dependency-mapper, duplicate-detector, pm-investigator
  purpose: cross-domain
---

## What I do

Provide the query patterns for walking the Hero knowledge graph across the PM-engineering boundary. The graph is the only place where cross-domain relationships are first-class — trackers silo by project and cannot represent `spec → commit → PR` chains, much less the bitemporal `owner_history` rows that record PM ↔ engineering ownership transitions. PM agents that need to know "what is engineering actually delivering against this spec?" must query the graph, not the tracker.

This skill covers query patterns, ranking rules (active-domain first, cross-domain visible), the staleness pitfalls, the namespace boundaries, and the cross-repo peering case where the graph spans repositories.

## When to use me

Load this skill when:

- `roadmap-curator` is reconciling weekly roadmap status against engineering delivery
- `handoff-coordinator` is verifying that the bitemporal `owner_history` row landed and engineering picked up the spec
- `dependency-mapper` is walking a `spec → spec → spec` chain across owner-flip boundaries
- `duplicate-detector` is checking whether an incoming intake duplicates an existing in-flight or completed spec
- `pm-investigator` is asking "has engineering already shipped something for this?"

## The graph as the cross-domain truth surface

Trackers (Jira / Linear / GitHub) are organized by project. The PM project and the engineering project are separate; cross-project edges in the tracker are weak (a link in a description field at best). The tracker cannot answer "which engineering feature delivers this PM story?" without manual upkeep.

The Hero knowledge graph spans domains. Under the **unified type model**, the same graph holds shared artifacts (`spec`, `epic`, `initiative`), PM-led artifacts (`prd`, `intake`), and engineering-led knowledge artifacts (`decision`, `convention`), plus commits, PRs, and releases via deep-code enrichment. **Cross-domain transitions are first-class via bitemporal `owner_history` on the shared types** — a spec's `owner` field flips between `pm` and `engineering` (and back), and the history records every transition. There is no separate `kind: handoff` edge under the unified model; the ownership history is the edge.

**The rule:** when a PM agent needs cross-domain truth, query the graph. The tracker is read-mostly for org-state inside its own domain (per the tracker-fronting decision in `.hero/knowledge/decisions/tracker-fronting-and-local-first.md`); it is not authoritative for cross-domain relationships.

## Query patterns

### spec → commit → PR (with ownership transition annotated)

The most common forward walk. Answers: "what's the live delivery state of this spec?"

```
spec:enable-saml  [owner: engineering, status: delivering]
  owner_history:
    pm → engineering @ 2026-05-17T14:33:00Z
  has-commit commit:abc123 (2 days ago)
  has-commit commit:def456 (5 hours ago)
  has-pr pr:1234 [open, 2 reviewers approved]
```

`roadmap-curator` runs this query against every `committed` initiative's child specs on its weekly reconciliation pass. Specs whose engineering owners are stale (no commits in 14d after the owner flip) surface as risks; specs with merged PRs surface as candidates for status transition.

### initiative → spec → release

The release-tracking walk. Answers: "what shipped under this initiative, in which release?"

```
roadmap:saml-rollout
  child spec:enable-saml [owner: engineering, status: completed]
    shipped-in release:2026-Q2-1
  child spec:saml-admin-ui [owner: engineering, status: completed]
    shipped-in release:2026-Q2-2
```

When all child specs are `completed` (and the latest `owner_history` shows the engineering-side close-out), the initiative is ready to transition to `shipped` (with `shipped_at` set to the latest release date).

### spec ← epic ← initiative (backward / "why")

The backward walk, from any node, to find the PM origin. This is what `hero_why spec-X` does. Answers: "why does this spec exist?"

```
spec:enable-saml [owner: engineering]
  owner_history:
    null → pm           @ 2026-05-01T09:12:00Z
    pm → engineering    @ 2026-05-17T14:33:00Z
  parent epic:enterprise-auth
    parent roadmap:saml-rollout
      evidence intake:saml-acme-q1, intake:saml-beta-456, ... [11 more]
```

The chain of decisions is contiguous — there is no domain *boundary* in the spec hierarchy itself (the spec is the same artifact); the cross-domain transition is captured in `owner_history` on the spec node. An engineer on `spec:enable-saml` runs `hero why` and sees the full lineage, including the verbatim customer quotes that motivated the initiative and the timestamp when PM handed the spec off.

### Cross-domain search

`hero search "csv export"` from a PM session returns:

- `spec:csv-export` (`owner: pm` — ranks first; PM is the active domain)
- `intake:csv-export-acme` (PM-led — ranks first)
- `spec:csv-export-fix-line-endings` (`owner: engineering, kind: bug` — same spec type, different owner; clearly marked by owner)
- `spec:csv-export-v1-perf` (`owner: engineering, kind: perf` — same type, different owner; clearly marked)

Under the unified type model, the *type* is the same (`spec`) across domains — what distinguishes "PM-led" from "engineering-led" is the `owner` field. Search renders the `owner` prominently so the user sees which side currently has the spec. There is no longer a separate `feature` type to mark visually; the boundary is the `owner` field, not the type.

## Using `hero_why` to walk back

`hero_why <node>` is the general-purpose backward walker. Pass any node (PM or engineering) and it returns the multi-hop chain of decisions, specs, and edges that led to the node's current state.

Use cases:

- PM asks "why is this feature in engineering?" → `hero_why feature:X` returns the cross-domain handoff edge and the upstream story / epic / initiative.
- PM asks "why did we drop this initiative?" → `hero_why roadmap:X` returns the rejection_reason, the dropped-on date, and any intakes that were re-routed.
- Engineering asks "why is this feature scoped this way?" → `hero_why feature:X` returns the originating story's AC and the linked PRD's Goals.

`hero_why` respects the domain boundary in output rendering — cross-domain nodes are visually distinguished.

## Ranking — active-domain first, cross-domain visible

When a query returns mixed results:

1. **Active-domain results rank first.** A PM session searching for "csv export" sees PM artifacts at the top.
2. **Cross-domain results are visible, not hidden.** They appear with a domain marker (engineering badge, color treatment per the locked UX).
3. **Cross-domain matches above a confidence threshold can interrupt active-domain ranking.** A high-confidence engineering feature match for an incoming intake should be surfaced prominently — the duplicate may need to resolve cross-domain.

The principle: respect the user's working context (active domain) without hiding the broader truth (cross-domain). Silent merging breaks the silo-tearing thesis.

## How `roadmap-curator` reads live engineering delivery state

The weekly reconciliation pass under the unified type model:

1. For each `committed` initiative, walk to all child specs (direct or via epics).
2. For each child spec, read `owner`, `status`, latest `owner_history` row, latest commit timestamp, open PR state.
3. Apply transition rules:
   - All child specs `completed` + shipped in a release → initiative `shipped`.
   - Any child spec engineering-owned but stale (no commits in 14d since the `pm → engineering` flip) → surface as a risk in the Roadmap board.
   - Any child spec PM-owned long after the initiative was committed → surface as not-yet-handed-off.
   - Any child spec recently flipped back to PM (`engineering → pm` in the last week) with a `handed_back_reason` → surface for follow-up.
4. Write the reconciled status to the initiative; update the `shipped_at` field when transitioning.

The reconciliation is the live wire between PM and engineering reality. Without it, the roadmap drifts; with it, the roadmap reconciles itself weekly. The bitemporal `owner_history` is what makes the cross-domain reconciliation queryable without a dedicated edge.

## Pitfalls

### Graph staleness

The graph is only as fresh as the last index pass. After a burst of activity (handoff, PR merges, release), there may be a window before the graph reflects reality. PM agents querying the graph should either:

- Trigger an index refresh (`hero index --if-stale`) before the query for high-stakes reads (initiative status transition).
- Accept staleness for low-stakes reads (background analytics).

The staleness window is typically minutes, not hours — but a PM doing a `committed → shipped` transition should not rely on a graph that's hours stale.

### Namespace boundaries (per primitive #6)

Cross-domain edges live in the graph with domain namespaces preserved. Queries that walk across namespaces must respect the boundary:

- Edge rendering distinguishes cross-domain edges visually.
- Edge metadata includes `target_domain: engineering` (or `target_domain: pm`).
- Search rankings honor active-domain-first.
- Aggregations that conflate counts across domains are usually wrong — "12 specs" is meaningless without saying which domains.

Mishandling the namespace breaks the silo-tearing semantics. The user should always be able to tell which side of the boundary a node lives on.

### Cross-repo edges via peering

When the PM domain and engineering domain live in different Hero workspaces (cross-repo peering), the handoff edge spans repositories. The graph supports this via the peering layer, but queries must:

- Tolerate peer unreachability — a `roadmap-curator` reconciliation pass cannot block forever on an offline peer.
- Cache the last-known cross-repo edge state and timestamp it.
- Surface "peer unavailable; last sync: <time>" rather than silently treating missing data as "no engineering work exists."

The cross-repo case is real (a PM workspace and an engineering workspace on separate repos). Queries should not break when the peer is offline; they should degrade gracefully.

### Trusting tracker state over graph state when they disagree

If a tracker says story `done` but the graph says feature is still `delivering`, **the graph wins for in-session views.** The tracker's `done` is org-state and may reflect a manual update that doesn't correspond to actual delivery. The graph reflects the engineering evidence (commits, PRs, releases).

For PM display, render the graph state. The tracker org-state can appear as a secondary chip ("Jira says done") but should not override the graph reading.

## Anti-patterns

- **Querying the tracker for cross-domain truth.** The tracker silos by project; cross-domain edges are at best a weak text link. Use the graph.
- **Silently merging cross-domain results into active-domain output.** Loses the boundary signal; breaks the silo-tearing thesis.
- **Trusting tracker state over graph state.** Graph wins for cross-domain in-session views.
- **Blocking forever on an unreachable cross-repo peer.** Degrade gracefully; cache last-known.
- **Reading from a stale graph for high-stakes status transitions.** Refresh first.
- **Aggregating counts across domain namespaces without labeling.** "12 specs" is meaningless; "8 PM stories + 4 engineering features" is correct.
- **Hiding cross-domain results to keep PM views "clean."** The whole point is that PM and engineering are visible across the boundary. Hiding it breaks the platform.
