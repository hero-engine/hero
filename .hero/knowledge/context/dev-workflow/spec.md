---
title: Development Workflow & Commands
type: context
status: active
created: 2026-04-29
tags: [imported, commands, dev-workflow]
slug: dev-workflow
---

## CLI Commands

These are run in the terminal, not as slash commands:
- `hero status` — workspace state and active specs
- `hero search <query>` — find specs by keyword
- `hero snapshot` — render the project-shape rollup (surfaces, stages, recent activity, risks)
- `hero sync import` — import issues from tracker as spec scaffolds
- `hero sync pull <slug>` — sync spec status from tracker
- `hero note <slug>` — quick note capture
- `hero check` — health check
- `hero peer list` — list registered sibling repos with reachability + manifest status
- `hero peer show <alias>` — inspect one peer (manifest contents, in-flight handoffs)
- `hero peer call <alias> --mode=advisory "..."` — ask peer's Hero a question (no writes on peer)
- `hero peer call <alias> --mode=spec-out "..."` — have peer's Hero design a spec natively on its side
- `hero handoff <spec> <alias>` — async-drop a local spec on peer's queue
- `hero handoff status` / `hero handoff accept <spec>` — track handoffs across the boundary
- `hero admin repos add <alias> <path>` — register a sibling repo as a peer (one-time setup)

## Running Hero Workflows in Codex

Hero's workflow commands are **not slash commands in Codex** — they are skill files you read and follow step-by-step.

**When the user asks you to deliver, diagnose, design, or run any Hero workflow:**

1. Read the workflow skill file at `.agents/skills/command-<name>/SKILL.md`
   (e.g. `.agents/skills/command-deliver/SKILL.md` when the user says "deliver")
2. Follow each step in the file as your workflow. These are **instructions to execute**, not documentation.
3. **Do NOT** skip steps, flip spec frontmatter as a shortcut, or treat the workflow as informational.

**Workflow routing table for Codex:**

| User intent | Skill file to read and follow |
|---|---|
| Deliver, implement, ship, execute | `.agents/skills/command-deliver/SKILL.md` |
| Diagnose, investigate, debug, fix | `.agents/skills/command-diagnose/SKILL.md` |
| Design, plan, spec, add feature | `.agents/skills/command-design/SKILL.md` |
| Review, PR, pull request | `.agents/skills/command-review/SKILL.md` |
| Check, health, validate workspace | `.agents/skills/command-check/SKILL.md` |
| Note, capture, remember | `.agents/skills/command-note/SKILL.md` |
| Compose, break down, epic | `.agents/skills/command-compose/SKILL.md` |
| Discover, brainstorm, explore | `.agents/skills/command-discover/SKILL.md` |

If the skill file doesn't exist, fall back to reading `.claude/commands/<name>.md` directly.

**A Hero workflow is not finished until its closing gate runs.** For `/deliver`, that gate is `hero spec verify <slug>` passing — and verify requires the cold delivery audit to have run first. Do NOT yield back to the user with a spec still in `planning` or `delivering` and the audit unrun. The audit and verify run in the **same turn** as the implementation — they are not a follow-up step the user triggers later. If you find yourself about to say "the audit still needs to run" or "I did not mark the spec complete because the gate still needs to run" — **run it now instead.** Stopping one step short of the closing gate is an unfinished delivery, not a handoff. This holds in every delivery mode, including the default supervised mode: "pause at handoffs" does not include the closing gates.

## CLI Commands

These are run in the terminal, not as slash commands:
- `hero status` — workspace state and active specs
- `hero search <query>` — find specs by keyword
- `hero snapshot` — render the project-shape rollup (surfaces, stages, recent activity, risks)
- `hero sync import` — import issues from tracker as spec scaffolds
- `hero sync pull <slug>` — sync spec status from tracker
- `hero note <slug>` — quick note capture
- `hero check` — health check
- `hero peer list` — list registered sibling repos with reachability + manifest status
- `hero peer show <alias>` — inspect one peer (manifest contents, in-flight handoffs)
- `hero peer call <alias> --mode=advisory "..."` — ask peer's Hero a question (no writes on peer)
- `hero peer call <alias> --mode=spec-out "..."` — have peer's Hero design a spec natively on its side
- `hero handoff <spec> <alias>` — async-drop a local spec on peer's queue
- `hero handoff status` / `hero handoff accept <spec>` — track handoffs across the boundary
- `hero admin repos add <alias> <path>` — register a sibling repo as a peer (one-time setup)

<!-- Imported from: AGENTS.md, CLAUDE.md -->
