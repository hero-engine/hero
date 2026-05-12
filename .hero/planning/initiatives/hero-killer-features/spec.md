---
title: Hero Killer Features — Agent Effectiveness, Team Power, Living Specs
type: initiative
status: planning
tags: [agent-effectiveness, team, intelligence, dx]
created: 2026-04-22
horizon: next
---

## Goal

Make Hero indispensable for teams and dramatically more effective for AI agents.
Three themes: (1) make agents smarter by giving them better data at session start
and during delivery, (2) make specs living documents that stay connected to code
after delivery, (3) give teams visibility and coordination tools that no other
spec-driven system offers.

## Children

| Slug | Title | Priority | Theme |
|---|---|---|---|
| impact-analysis | Impact analysis — what breaks if I touch this? | P0 | agent |
| living-contract | Spec-as-living-contract — continuous criteria validation | P0 | team |
| activity-digest | Recent activity digest for session start | P0 | agent |
| team-activity-feed | Session replay + team-wide activity feed | P1 | team |
| test-coverage-map | Criterion-to-test coverage mapping | P1 | agent |
| challenge-diagnosis | Challenge/revise a bug diagnosis with engineer feedback | P1 | team |
| multi-repo-specs | Cross-repo spec references and drift detection | P1 | team |
| learned-templates | Spec templates learned from delivery patterns | P2 | dx |
| cost-calibration | Estimated vs actual effort calibration | P2 | team |
| environment-awareness | CI/deployment/runtime visibility (sub-initiative) | P2 | agent |

## Sequencing

Start with the P0 trio — impact-analysis, living-contract, activity-digest.
These are independent and can be delivered in parallel. Impact-analysis builds
on existing `hero_code` and `hero_context`; living-contract extends the spec
lifecycle; activity-digest is a small new command.

P1 items depend on P0 patterns. Challenge-diagnosis is standalone.
Multi-repo-specs is the biggest architectural lift — needs design time.

## Context

Born from a brainstorm session comparing Hero to competitors and identifying
where Hero's unique position (spec corpus + code graph + agent workflow) creates
capabilities nobody else can build. The user has a real 20+ repo project that
would immediately benefit from multi-repo-specs.
