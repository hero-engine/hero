---
title: "Team Knowledge Flywheel"
type: feature
status: draft
priority: medium
horizon: next
smoke: deferred
---

# Team Knowledge Flywheel

## Problem
`.hero/knowledge/` is per-repo but not shared across team members' sessions effectively. Conventions and decisions captured by one developer should benefit everyone.

## Proposed Solution
Make knowledge a shared team asset:
- Conventions, ADRs, and code structure committed to repo (already works via git)
- Cloud sync for cross-repo knowledge (org-level conventions)
- Knowledge quality: track which conventions are actually followed vs ignored
- Auto-suggest conventions based on observed patterns across the team

### Cloud tie-in:
- Org-level knowledge store (cloud)
- Convention compliance dashboard
- "Your team decided X on 2026-03-15" surfaced in relevant contexts
