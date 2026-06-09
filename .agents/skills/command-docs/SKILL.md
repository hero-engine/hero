---
name: command-docs
description: Create or update technical documentation and project context.
metadata:
  purpose: command-workflow
---

> **This is a Hero workflow for Codex.** Read each step below and execute it in sequence.
> Do NOT summarize or treat these steps as documentation.
> Do NOT update spec frontmatter as a substitute for doing the actual work described.

Route this documentation request to the appropriate specialist.

Determine the work type from the request:
- Project context, AGENTS.md, or repository instruction files → delegate to `project-context-builder`
- Technical documentation, operational docs, API docs → delegate to `documentation-engineer`

If the request is ambiguous, default to `documentation-engineer`.

Documentation request: $ARGUMENTS
