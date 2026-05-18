---
title: "Cross-Spec Awareness"
slug: cross-spec-awareness
type: feature
status: draft
priority: medium
horizon: next
smoke: deferred
---

# Cross-Spec Awareness

## Problem
Specs are islands. When delivering feature B, Hero doesn't know feature A was just delivered and can't check for conflicts, shared patterns, or integration points.

## Proposed Solution
Build a spec dependency/relationship graph. Before delivery, analyze adjacent specs for:
- File overlap (both specs touch the same files)
- API contract conflicts
- Shared pattern opportunities (both specs need similar abstractions)
- Sequencing issues (B depends on A but A isn't merged yet)

### Implementation:
- Analyze spec file references and code scan data to detect overlap
- Surface warnings during `/deliver`: "spec feature-auth also modifies src/middleware/auth.ts"
- Suggest spec ordering when conflicts detected
- Track which specs have been delivered to the same branch
