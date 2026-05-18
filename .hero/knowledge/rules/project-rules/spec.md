---
title: Project Rules
slug: project-rules
type: rule
status: active
created: 2026-04-29
scope: ["*"]
tags: [imported, rules]
---

## Important Rules

- **Don't assume.** Surface tradeoffs and ask questions if anything is unclear. Present multiple interpretations instead of picking one silently.
- **Simplicity first.** Write the minimum code that solves the problem. No speculative features, no unnecessary abstractions, and no error handling for impossible scenarios.
- **Surgical changes.** Touch only what is strictly required. Do not "improve" nearby code or refactor unrelated sections. Match the existing style perfectly.
- **Verify before reporting done.** Define clear success criteria for every task. Run tests or validation scripts and iterate until the criteria are met before reporting completion.
- **Local specs first.** When asked to work on bugs, features, or any tracked items, ALWAYS check what's already imported locally before querying the tracker. Use `hero search --list --type <type>` to find local specs. Only go to the tracker if the local search comes up empty. When working on multiple items (e.g. "diagnose 10 bugs"), select from locally imported specs — never bulk-query the tracker to pick work items.
- Always check spec status before doing work — don't investigate closed bugs or deliver completed specs
- When a tracker is configured, sync status with `hero pull` before starting work
- **Auto-capture learnings.** At the end of major workflows (`/deliver`,
  `/diagnose`, `/design`, `/retro`), evaluate whether the session produced
  knowledge worth persisting — design decisions made, debugging techniques
  that worked, conventions discovered or reinforced, surprising findings.
  If so, write a short entry to `.hero/knowledge/notes/` without prompting.
  Skip if nothing non-obvious was learned. This is enabled by default via
  `knowledge.auto_capture` in `hero.json`.
- **File useful queries back.** When `hero_ask` or research produces a
  synthesis that would help future sessions (architecture explanations,
  debugging playbooks, integration guides), write it to
  `.hero/knowledge/context/` as a knowledge entry. Every exploration
  should add up — ephemeral Q&A becomes permanent institutional memory.
- Specs use YAML frontmatter with fields: title, type, status, tracker_id, priority, severity
- Imported specs include tracker-prefixed fields (e.g. jira_status, jira_priority, jira_assignee) under a `# Jira` / `# Github` / `# Linear` comment header in frontmatter

<!-- Imported from: AGENTS.md -->
