---
description: Trace where a fact, decision, or piece of code came from.
---
Multi-hop traversal across the hero graph: knowledge → decisions →
specs → commits. Surface the chain that led to the user's target
existing today, so "why does X work this way" gets a real answer.

Steps:

1. Identify the target — file path, decision name, spec slug, or
   freeform topic from `$ARGUMENTS`.
2. Use `hero_why` (MCP tool) when available — it walks the graph
   natively.
3. When the tool isn't available, fall back to: `hero_search` for
   related specs/notes, `git log --follow` for commits that touched
   the target, and `grep` over `.hero/knowledge/decisions/` for
   rationale entries.
4. Render the chain as a bulleted timeline: oldest decision → newest
   change. Each row names the source (spec / decision / commit /
   note) with a clickable path.

Don't invent provenance. If the chain peters out — say so.

Request: $ARGUMENTS
