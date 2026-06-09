---
name: command-roadmap-review
description: On-demand triage of roadmap-shape drift across the planning corpus. Walks one finding at a time and resolves on confirm.
metadata:
  purpose: command-workflow
---

> **This is a Hero workflow for Codex.** Read each step below and execute it in sequence.
> Do NOT summarize or treat these steps as documentation.
> Do NOT update spec frontmatter as a substitute for doing the actual work described.

Route this triage request to the `roadmap-reviewer` agent.

The agent loads the `roadmap-review` and `spec-composition` skills,
surveys the workspace (`hero size --check`, `hero_warnings`,
`hero_list`, `hero_search`), prioritizes findings, walks them one at
a time, and executes the chosen resolution CLI on confirm. Sizing is
the only active lens in v1; horizons / releases / sprint-shape are
refused with a scaffolded phrase.

If `$ARGUMENTS` is non-empty, treat it as a focus slug — the agent
should scope its survey and triage to that one spec instead of the
whole corpus.

Focus: $ARGUMENTS
