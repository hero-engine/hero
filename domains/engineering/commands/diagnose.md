---
description: Investigate a bug, classify the root cause, and produce a fix spec.
---
**Before creating any file**, check whether the user is working in a sub-folder workspace. If so, preserve that workspace's `subproject:` frontmatter and write the spec under the workspace's `.hero/` root; otherwise write under the project `.hero/` root.

**Stamp the active domain in frontmatter.** New bug specs MUST emit a `domain:` field reflecting the workspace's active domain — run `hero domain` (or read the `domain` key from `.hero/hero.json`; empty means engineering) and write that value. This keeps bug specs under the correct DSKG namespace partition.

Route this bug investigation to the `debug-investigator` agent.

Pass the bug report and any spec path or tracker ID to the agent. The `debug-investigator` handles the complete diagnosis workflow: pre-flight status check, investigation, root cause classification, fix planning, spec writing, and tracker posting.

**Before starting work**, emit `hero next ask` to capture the bug report
the user pasted in. This preserves session intent across compaction — see
the `next-handoff-emit` skill for the full pattern.

**The agent must write all findings into the spec file on disk.** The spec file is the deliverable, not chat output.

**Always include a `## Kickoff` section** in the fix spec. Follow the `kickoff-prompt` skill — the kickoff is a paste-ready cold-start prompt for picking the fix work back up in a fresh session. Skipping it excludes the spec from `hero queue` and triggers a `hero check` advisory.

**After writing the fix spec, run `hero index --if-stale -q`** so it surfaces in `hero search` / `hero list` / MCP tools without anyone needing to run a manual reindex.

**Before proposing a fix direction**, call `hero_anchor` to check project tripwires. If any proposed fix would reintroduce a forbidden dependency, pattern, or tool, eliminate that direction and find an alternative within project constraints.

If the `debug-investigator` returns with `Needs more research? → Yes`, that is an acceptable outcome — report it to the user and move on.

---

## Batch Mode: Diagnosing Multiple Bugs

When asked to diagnose multiple bugs (e.g. "diagnose 10 bugs", "work through the imported bugs"):

1. **Select from locally imported specs only** — `hero search --list --type bug` (filter with `--status planning` / `--since`). Never query the tracker to pick work items.
2. **Filter to open status** (`planning`, `draft`, `active`) — skip `completed` / `superseded`.
3. **Confirm the selection with the user** before starting.
4. **Default: one at a time**, sequentially through `debug-investigator`, each bug fully diagnosed (spec written, tracker posted) before the next.

If the user explicitly asks for parallel execution (e.g. "run each in an agent", "diagnose all of them at once"), launch one subagent per bug as the `debug-investigator` (load `debugging-investigation`), each with its own spec path, tracker ID, and bug description. Follow the `batch-discipline` skill for concurrency, per-item isolation, the after-completion verification loop, and the summary-table format.

---

## Session Title

On the **first interaction**, set a concise session title reflecting the bug being diagnosed (e.g. "diagnose: null pointer in cart total", "diagnose: PROJ-456 export timeout").

---

Bug report: $ARGUMENTS
