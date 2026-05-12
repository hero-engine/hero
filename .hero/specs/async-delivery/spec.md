---
title: "Async Delivery"
type: feature
status: completed
priority: high
horizon: now
---

# Async Delivery

## Problem
Hero is synchronous — the developer sits and watches. The spec-driven model is naturally suited for async: approve a spec, hand off to an agent, come back to a PR.

This is especially painful for batch workflows. Today, diagnosing 30 bugs means sitting in a session and prompting "diagnose the next 10" over and over. Delivering a backlog of specs means babysitting each one sequentially.

## Proposed Solution
After spec approval, Hero kicks off background agent execution that delivers the spec, runs tests, and opens a PR. Developer reviews when ready.

### Batch operations — the primary use case:

**Batch diagnosis:**
- `hero diagnose --batch` or `hero diagnose --all` — agent works through all imported bug specs
- For each bug: investigate, produce a fix spec, update status
- Developer comes back to a queue of fix specs ready for review/approval
- No more "diagnose 10 bugs" prompting loop

**Batch delivery:**
- `hero deliver --batch` — deliver all approved specs
- Each spec gets its own branch and PR
- Agent works them sequentially or in parallel (configurable)
- Progress visible: `hero status` shows "3/12 delivered, 2 failed, 7 pending"

**The full pipeline:**
- `hero import` pulls 30 bugs from Jira
- `hero diagnose --batch` investigates all 30, produces fix specs
- Developer reviews fix specs, approves the good ones
- `hero deliver --batch` implements all approved fixes
- Developer reviews PRs

This turns a week of tedious prompting into: import, wait, review specs, approve, wait, review PRs.

### Key considerations:
- How does this integrate with OpenClaw / open agent orchestration frameworks?
- Agent needs sandbox or branch isolation
- Progress reporting (webhook, polling, CLI status)
- Failure handling — partial delivery, test failures, spec ambiguity detected mid-flight
- Could run locally (background process) or remotely (cloud agent)
- Spec quality gate before async handoff (see spec-quality-scoring)
- Batch diagnosis needs rate limiting for tracker API calls
- Each batch item should be independent — one failure doesn't block the rest

### Open questions:
- Local-first (background process) vs cloud-first (remote agent)?
- How to handle specs that need human clarification mid-delivery?
- Integration with CI/CD — should async delivery run in CI?
- Relationship to OpenClaw and similar open agent orchestration efforts
- Parallelism: how many specs can an agent work simultaneously? Branch conflicts?
