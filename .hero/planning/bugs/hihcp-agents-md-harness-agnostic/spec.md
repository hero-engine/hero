---
title: "Produce Harness-Agnostic AGENTS.md, Demote CLAUDE.md"
slug: hihcp-agents-md-harness-agnostic
type: bug
status: planning
domain: engineering
size: small
priority: critical
created: 2026-06-09
tags: [hero-code, agents-md, claude-md, system-prompt, harness-agnostic, p0]
parent: hero-in-hero-code-parity
depends-on:
  - hihcp-skill-run-tool
---

# Produce Harness-Agnostic AGENTS.md, Demote CLAUDE.md

## Issue

hero-code's system prompt is assembled from `CLAUDE.md`, which contains Claude
Code-specific concepts: ToolSearch for deferred tool loading, Explore agent
references, skill invocation via a `Skill` tool (not `skill_run`). None of these
exist in hero-code. The model follows instructions it cannot satisfy, producing
confused output and wasted turns.

Parent initiative: `hero-in-hero-code-parity`.
Depends on: `hihcp-skill-run-tool` (AGENTS.md should reference the tool that
now exists).

## Scope -- design inputs for `/design`

**In hero-code repo:**
- Rewrite `AGENTS.md` to be the authoritative, harness-agnostic source of Hero
  workflow routing instructions
- Reference `skill_run` tool (from item 1) instead of Claude Code's `Skill` tool
- Remove references to ToolSearch, deferred tool loading, Explore agent
- Strip or delete the hero:managed block from `CLAUDE.md`

**In hero repo:**
- Teach the snapshot emitter / `hero install` to detect the active harness
- When the harness is not Claude Code, emit to `AGENTS.md` instead of `CLAUDE.md`
- Use the same source data for both renderings to prevent drift

## Boundaries

- Do not delete CLAUDE.md entirely -- Claude Code users still need it
- Claude Code-specific features (ToolSearch, deferred tools, Explore agent)
  remain in CLAUDE.md for Claude Code users
- Harness-agnostic content (routing table, Hero rules, workflow instructions)
  moves to AGENTS.md, which both harnesses read

## Risks

- Cross-repo coordination: hero-code AGENTS.md and hero repo emitter must land
  together or in the right order
- Regression risk for Claude Code users if CLAUDE.md is over-stripped
- AGENTS.md drift: if the hero repo evolves without updating the emitter,
  hero-code falls behind

## Validation

- hero-code system prompt contains only instructions the model can follow
- No references to Claude Code-specific concepts in the hero-code prompt
- `hero install` on a hero-code project emits AGENTS.md with correct content
- Claude Code users are unaffected (CLAUDE.md still works for them)
