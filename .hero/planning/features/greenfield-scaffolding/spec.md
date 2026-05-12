---
title: "Greenfield Scaffolding"
type: feature
status: draft
priority: medium
horizon: next
smoke: deferred
---

# Greenfield Scaffolding

## Problem
Hero shines on existing codebases but is weakest at "I have no code yet." `/design` + `/deliver` on a greenfield project should scaffold something running in minutes.

## Proposed Solution
Enhanced `/design` for greenfield:
- Detect empty/new project
- Ask high-level questions: what are you building, who uses it, what stack?
- Generate initial spec with architecture decisions, directory structure, dependency choices
- `/deliver` scaffolds running app with tests, CI config, conventions file
- Essentially: spec-driven `create-*-app` that produces a real project, not a template

### Differentiator vs v0/Bolt:
- Those produce throwaway prototypes. This produces production-ready structure with conventions, tests, and a knowledge base already seeded.
