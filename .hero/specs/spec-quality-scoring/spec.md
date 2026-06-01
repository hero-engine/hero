---
title: "Spec Quality Scoring"
slug: spec-quality-scoring
type: feature
status: completed
priority: high
horizon: now
completed_at: 2026-05-18T19:25:38Z
---

# Spec Quality Scoring

## Problem
The #1 failure mode is bad specs producing bad code. No guardrail exists between spec approval and delivery.

## Proposed Solution
Before `/deliver`, automatically assess spec quality and flag issues.

### Scoring dimensions:
- Acceptance criteria: present, measurable, testable?
- Scope clarity: bounded or open-ended?
- Technical specificity: references concrete files/packages/APIs?
- Ambiguity detection: vague language, conflicting requirements?
- Dependency awareness: does it reference other specs or features that aren't complete?
- Test strategy: does the spec indicate how to verify?

### Output:
- Score (0-100) with breakdown
- Specific warnings ("acceptance criteria #3 is not testable")
- Suggestions ("consider specifying which API endpoint this affects")
- Hard block if score below threshold (configurable)

### Value:
- Prevents async delivery from failing on ambiguous specs
- Teaches developers to write better specs over time
- Feedback loop: track score vs delivery success rate
