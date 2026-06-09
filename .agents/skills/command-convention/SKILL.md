---
name: command-convention
description: Analyze a codebase pattern and produce a convention spec documenting the canonical approach.
metadata:
  purpose: command-workflow
---

> **This is a Hero workflow for Codex.** Read each step below and execute it in sequence.
> Do NOT summarize or treat these steps as documentation.
> Do NOT update spec frontmatter as a substitute for doing the actual work described.

**Before creating any file**, check whether the user is working in a sub-folder workspace. If so, preserve that workspace's `subproject:` frontmatter and write the spec under the workspace's `.hero/` root; otherwise write under the project `.hero/` root.

Route this convention request to the `convention-author` agent.

The convention author will:
1. Search the codebase for existing instances of the pattern
2. Assess whether current usage is consistent or fragmented
3. Identify or synthesize the canonical approach
4. Produce a convention spec with examples, anti-patterns, and adoption notes
5. Save it to `.hero/conventions/{slug}/spec.md`

Pattern to standardize: $ARGUMENTS
