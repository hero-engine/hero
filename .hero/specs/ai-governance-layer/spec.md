---
title: "AI Governance Layer"
slug: ai-governance-layer
type: feature
status: completed
priority: high
tags: [cloud, enterprise, billion-dollar]
horizon: now
completed_at: 2026-05-18T19:25:38Z
created: 2026-05-12
---

# AI Governance Layer — "The Spec Is The Law"

## Problem
Large orgs gave 200+ developers AI coding tools with no shared understanding of what's being built, no consistency, no audit trail, and no way to measure velocity vs debt. CTOs have no visibility. Architects are drowning. Security teams are terrified.

## Proposed Solution
Every AI-generated change must trace back to an approved spec. No spec, no merge.

### Core capabilities:
- GitHub/GitLab integration that gates PRs on spec linkage and convention compliance
- Intent-aware policy engine: "This PR claims to implement feature-xyz but also modified the auth module which isn't in the spec — flag for review"
- Convention enforcement: not a linter but a policy engine that understands architectural intent
- Audit trail: every change traces from spec → approval → delivery → merge
- Configurable strictness: advisory mode (warnings) → enforcement mode (blocks)

### Why this sells to enterprises:
Not "your developers will be faster" but "your developers will be faster and you won't lose control." CISO and CTO sale simultaneously.

### Implementation phases:
1. PR bot that checks for spec linkage (GitHub App)
2. Convention compliance checking against .hero/knowledge/conventions
3. Scope drift detection (PR touches files outside spec scope)
4. Full policy engine with org-configurable rules

### The grumpy engineer escape hatch:
Specs can be hand-delivered. A developer can take any spec and implement it themselves without Hero's agent. The governance layer doesn't care HOW the spec was delivered — only that it WAS delivered against a spec. This preserves developer autonomy while maintaining organizational visibility. The spec is the contract, not the tool.
