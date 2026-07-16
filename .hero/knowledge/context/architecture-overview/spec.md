---
title: Architecture Overview
type: context
status: active
created: 2026-04-29
tags: [imported, architecture]
slug: architecture-overview
---

## Project Structure

- `<harness>/commands/` — Slash command definitions (workflows like /design, /deliver, /diagnose)
- `<harness>/agents/` — Specialized agent roles (feature-delivery-lead, debug-investigator, etc.)
- `<harness>/skills/` — Domain-specific knowledge and patterns (each skill is a subdir with SKILL.md)
- `.hero/planning/` — Active specs being worked on
- `.hero/specs/` — Completed specs (archive)
- `.hero/knowledge/` — Project knowledge base (conventions, decisions, context)
- `.hero/hero.json` — Project configuration

`hero install` **writes** these into your harness's own directory in that harness's native format — e.g. `.claude/commands/`, `.claude/agents/`, and `.claude/skills/` for Claude; `.codex/agents/*.toml` (TOML) plus workflow skills under `.agents/skills/` for Codex (Codex has no commands directory — its slash commands are a built-in enum, so Hero commands install there as skills). They are generated copies, **not** symlinks or views: re-running `hero install` regenerates them, so hand-edits to the installed files are overwritten on the next install.

## Project Structure

- `<harness>/commands/` — Slash command definitions (workflows like /design, /deliver, /diagnose)
- `<harness>/agents/` — Specialized agent roles (feature-delivery-lead, debug-investigator, etc.)
- `<harness>/skills/` — Domain-specific knowledge and patterns (each skill is a subdir with SKILL.md)
- `.hero/planning/` — Active specs being worked on
- `.hero/specs/` — Completed specs (archive)
- `.hero/knowledge/` — Project knowledge base (conventions, decisions, context)
- `.hero/hero.json` — Project configuration

`hero install` **writes** these into your harness's own directory in that harness's native format — e.g. `.claude/commands/`, `.claude/agents/`, and `.claude/skills/` for Claude; `.codex/agents/*.toml` (TOML) plus workflow skills under `.agents/skills/` for Codex (Codex has no commands directory — its slash commands are a built-in enum, so Hero commands install there as skills). They are generated copies, **not** symlinks or views: re-running `hero install` regenerates them, so hand-edits to the installed files are overwritten on the next install.

<!-- Imported from: AGENTS.md, CLAUDE.md -->
