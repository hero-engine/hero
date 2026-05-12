---
title: v2 Agents and Skills — New Specialists and Capabilities
type: feature
status: completed
tags: [agents, skills, specialists]
created: 2026-04-12
relations:
  - target: hero-v2-system-design
    kind: parent
horizon: now
---

## Goal

Add 4 new specialist agents and 7 new skills to cover capabilities missing in v1: conventions authoring, data migrations, test strategy design, dependency analysis, incident response, Python stack, and context injection.

## What Was Built

**New agents:**
- `convention-author` — analyzes codebase patterns, produces convention specs
- `migration-engineer` — plans and executes data/API/library migrations
- `test-architect` — designs test strategies (unit/integration/E2E/property testing)
- `dependency-analyst` — evaluates library choices, vulnerability and maintenance health

**Incident response as skill, not agent** — `incident-response` skill added to `debug-investigator` rather than a separate agent. Activated via `/diagnose --incident`.

**New skills:**
- `python-stack` — Python packaging, async, type hints, testing patterns
- `convention-writing` — how to analyze patterns and write effective convention specs
- `migration-safety` — migration patterns, rollback strategies, zero-downtime techniques
- `test-strategy` — test pyramid, coverage strategies, property testing
- `dependency-analysis` — library health evaluation, license risks, supply chain
- `incident-response` — production triage, evidence gathering, post-mortem format
- `context-injection` — how to use `hero context` output in agent workflows

**Architecture agents updated** — `brownfield-architect` and `greenfield-architect` now load the `architecture-principles` skill instead of duplicating content inline.

**`debug-investigator` updated** — checks spec corpus for past bugs in the same files before investigating.

## Changes

- `agents/convention-author.md`
- `agents/migration-engineer.md`
- `agents/test-architect.md`
- `agents/dependency-analyst.md`
- `skills/python-stack.md`
- `skills/convention-writing.md`
- `skills/migration-safety.md`
- `skills/test-strategy.md`
- `skills/dependency-analysis.md`
- `skills/incident-response.md`
- `skills/context-injection.md`
- `agents/brownfield-architect.md` — now loads architecture-principles skill
- `agents/greenfield-architect.md` — now loads architecture-principles skill
- `agents/debug-investigator.md` — corpus-aware investigation
