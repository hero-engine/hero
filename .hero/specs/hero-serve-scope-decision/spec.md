---
title: "Decide `hero serve` Scope After the hero/hero-cloud Split"
slug: hero-serve-scope-decision
type: feature
status: completed
priority: high
tags: [architecture, decision, hero-serve, scope, post-split, replaced]
created: 2026-05-16
relations:
  - target: hero-cloud-split
    kind: depends-on
  - target: hero-team-server
    kind: blocks
  - target: hero-dashboard-v2
    kind: blocks
  - target: hero-automations
    kind: blocks
  - target: hero-surface-architecture
    kind: superseded-by
  - target: hero-surface-deployment-and-rendering
    kind: superseded-by
horizon: now
completed_at: 2026-05-18T19:25:38Z
---

> **Replaced by the [Hero Surface Architecture](../../initiatives/hero-surface-architecture/spec.md) initiative.** The scope question is resolved structurally rather than by adjudication: one binary, one surface, layer-gated features. See [hero-surface-deployment-and-rendering](../hero-surface-deployment-and-rendering/spec.md) for the accepted decision. This spec is preserved for history.


# Decide `hero serve` Scope After the hero/hero-cloud Split

## Goal

Land a clear, written decision on what `hero serve` (the CLI's local
serve command) keeps doing after the hero/hero-cloud split vs. what
moves to the hero-cloud server. Resolve the three deferred specs
(`hero-team-server`, `hero-dashboard-v2`, `hero-automations`) by
splitting, scoping, or relocating them based on that decision.

## Kickoff

Quick architectural decision blocked by lack of explicit scoping.
After the repo split landed (2026-05-16), 17 cloud-side specs moved
cleanly to hero-cloud. Three specs were deferred because they
straddle the line between "CLI convenience local server" and "team
coordination server (= hero-cloud)."

User note from the moment of deferral: *"hero serve [had] some stuff
that it might need to still do — and some that it should stop doing
— but not full blown get rid of hero serve — that was also some UI
elements for info."*

**Pick up at:** read the three deferred specs (`hero-team-server`,
`hero-dashboard-v2`, `hero-automations`), have a focused conversation
about what `hero serve` keeps (info UI, local-only convenience) vs.
what moves to hero-cloud (multi-dev job queue, approval gates, team
coordination). Capture the line as a decision spec. Then update the
three blocked specs accordingly — either split each into "local-side
in hero" + "server-side in hero-cloud" specs, narrow scope to one
side, or relocate.

→ `.hero/planning/features/hero-serve-scope-decision/spec.md`

## Problem

`hero serve` exists in the CLI today and does several things:

- Local web UI for browsing the workspace (info elements — likely
  stays in CLI for solo devs and small team users who want a quick
  local view of their own workspace)
- Job queue + approval gates for multi-dev coordination (likely
  moves to hero-cloud — this is exactly what the team server is)
- Automations / scheduled work (unclear — could be local cron-like
  per-dev convenience, or could be server-side org automation)

Without an explicit scope line, the three deferred specs are stuck
in limbo: they're all currently in hero's planning area but partially
or fully belong in hero-cloud. Pushing them through `/design` without
resolving the boundary first would entrench the ambiguity.

This is a small but load-bearing decision. The cost of not making it
is that future cloud features in this area get planned twice or
designed against the wrong repo's constraints.

## Suggested Approach

1. Read the three deferred specs end-to-end. Build a list of what
   each one assumes about where its server-side runs.
2. Sketch the line: what's the principle that decides "this lives
   in `hero serve`" vs. "this lives in hero-cloud"? Candidate
   principles:
   - Single-developer concerns (info UI, personal workspace browsing)
     stay in `hero serve`; multi-developer concerns (job queue,
     approvals, coordination) move to hero-cloud.
   - Anything that needs to be reachable by other people / processes
     beyond the dev's own machine = hero-cloud. Anything that's
     fundamentally about *my own work, on my machine* = `hero serve`.
3. Apply the principle to the three specs. For each: stays in hero
   as-is, moves to hero-cloud, or splits into a pair of specs.
4. Capture the principle itself as either a convention or a brief
   decision spec so future cloud/CLI work has a clear rule.
5. Update QUEUE / index, mark this spec completed.

## Acceptance Criteria

- THE SYSTEM SHALL produce a written scope rule that distinguishes
  `hero serve` (local CLI server) from hero-cloud (team server).
- WHEN the rule is applied to `hero-team-server`, `hero-dashboard-v2`,
  and `hero-automations`, each SHALL be resolved (kept, moved, or
  split) without remaining in the deferred state.
- WHERE a spec splits across both repos, the resulting child specs
  SHALL link to each other via `relations:` and reference the
  cross-repo-workflow convention.
- THE SYSTEM SHALL capture the scope rule somewhere durable
  (`.hero/knowledge/conventions/` or a decision spec) so subsequent
  `hero serve` vs. hero-cloud questions are answered without
  re-deriving the rule.

## Out of Scope

- Implementing any of the affected specs — this spec only decides
  where they live and what shape they take.
- Designing `hero serve`'s feature set beyond the boundary
  question.
- The broader hero-cloud roadmap — that's the hero-cloud initiative
  in hero-cloud's planning area.
