---
name: command-capture
description: Extract and persist learnings from the current session into the Hero knowledge base.
metadata:
  purpose: command-workflow
---

> **This is a Hero workflow for Codex.** Read each step below and execute it in sequence.
> Do NOT summarize or treat these steps as documentation.
> Do NOT update spec frontmatter as a substitute for doing the actual work described.

**Before creating any file**, check whether the user is working in a sub-folder workspace. If so, preserve that workspace's `subproject:` frontmatter and write the note under the workspace's `.hero/` root; otherwise write under the project `.hero/` root.

Review this session and capture any knowledge worth persisting.

Load the `auto-knowledge-capture` skill — it contains the full methodology for what to capture, what to skip, how to classify learnings, and the entry templates.

If nothing meets the capture threshold, say so: "No new knowledge to capture from this session."

After capturing, run `hero index` to make new entries searchable.

Session context: $ARGUMENTS
