---
title: "Spec Decomposition — Split Large Specs into Child Specs"
slug: spec-decomposition
type: feature
status: completed
tags: [agent, planning]
created: 2026-04-12
horizon: now
---

## Goal

The `/split` agent command breaks a large spec into smaller, focused child specs — enabling incremental delivery and clearer scope boundaries.

## Design

When an agent invokes `/split`, Hero analyzes the target spec and produces a set of child specs that collectively cover the parent's scope. Each child spec:

- Has its own slug derived from the parent (e.g., `parent-slug/child-name`)
- Contains a `parent` relation pointing back to the original spec
- Covers a distinct, non-overlapping slice of the parent's scope
- Is independently implementable and testable

The agent uses the spec's sections, acceptance criteria, and change list to identify natural decomposition boundaries. The original spec is updated to reference its children.

## Changes

- `commands/split.md` — `/split` agent command definition and prompt

## Acceptance Criteria

- `/split` produces multiple child specs from a single large spec
- Child specs have parent relations linking back to the original
- Each child spec has clear, non-overlapping scope
- The original spec is updated to reference its children
- Child specs are independently actionable
