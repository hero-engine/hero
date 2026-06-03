---
title: Cross-harness agent rules belong in agents_md.go
type: rule
status: active
tags: [install, agents, claude-md, agents-md, harness-integration]
created: 2026-05-12
---

## Rule

When Hero needs to enforce a behavioral rule that agents on **multiple harnesses** (Claude Code, Codex, Cursor, OpenCode, Copilot, generic) must follow, the canonical insertion point is `generateAgentsMdBody()` in [internal/install/agents_md.go](../../../../internal/install/agents_md.go).

That function is the **single source** for the Hero-managed block written into:

- Project-root `CLAUDE.md` (read by Claude Code)
- Project-root `AGENTS.md` (read by Codex, Cursor, OpenCode, Copilot, others via the AGENTS.md convention)

Domain-pack-specific rules — guidance tied to engineering vs. some future pack — go in `domains/<pack>/AGENTS.md`.

## Why

Editing only `CLAUDE.md` leaves every non-Claude agent without the rule. This is the failure mode of the [pre-commit-auto-stage-next](../../../specs/pre-commit-auto-stage-next/spec.md) spec: AC #11 specified a CLAUDE.md backstop, but the rule never landed in the template, so agents on every harness — including Claude — fell back to wrong defaults on fresh clones and worktrees.

## How to apply

1. Open `internal/install/agents_md.go`, find `generateAgentsMdBody()`.
2. Add the rule as a bullet inside the appropriate section (usually "Important Rules"). One bullet, one rule.
3. Refresh the rendered output in this repo by editing the managed block of `CLAUDE.md`, `AGENTS.md`, and (if applicable) `domains/engineering/AGENTS.md`. Or run `hero install project . --force-managed` once tested.
4. No test pins the literal rules content — it's prose. Don't add a brittle string-match test; trust the generator.

## Anti-pattern

- Editing one harness's instruction file (`CLAUDE.md`) by hand and stopping there.
- Adding the rule to a domain pack when it's a workflow rule, not a domain rule.
- Adding the rule as a skill or command description; agents don't always load those.
