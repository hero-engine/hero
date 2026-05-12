---
description: Scan the codebase to detect the technology stack, extract code structure, and generate knowledge base entries.
---
Analyze the current project and populate the Hero knowledge base with initial entries plus code intelligence.

Load the `project-context-generation` skill for enrichment guidance.

1. Run `hero scan --dry-run` first to preview what will be detected
2. Review the output with the user — check that languages, frameworks, and tools are correctly identified
3. If the preview looks good, run `hero scan` to generate entries and code intelligence
4. Read each generated entry and enrich it using guidance from the `project-context-generation` skill

The scan automatically includes code intelligence (packages, symbols, dependencies, hot files) unless `code_scan.depth` is set to `"disabled"` in hero.json. Use `hero scan --code` to run only the code intelligence scan.

Use `--force` to overwrite existing entries when re-scanning after stack changes.

$ARGUMENTS
