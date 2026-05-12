---
title: Memory Tools, Community Patterns, and Where Hero Stands
type: note
created: 2026-04-12
milestone: v0.2
tags: [ai-memory, session-continuity, community, architecture, hero-features, v0.2]
horizon: someday
---

# Memory Tools, Community Patterns, and Where Hero Stands

## Context

Research question: what can Hero learn from tools like claude-mem, supermemory, mem0, and the broader claude-code community (awesome-claude-code, etc.)? Are there genuine gaps, or does Hero already cover the important ground?

## The Memory Tool Landscape

### claude-mem (thedotmack/claude-mem)

Solves the cold-start problem: every session starts with zero context. Uses Claude Code lifecycle hooks — `SessionStart` injects prior context, `PostToolUse` captures tool calls in real time, `Stop`/`SessionEnd` triggers compression. Storage is SQLite + FTS5 (source of truth) plus ChromaDB for vector/semantic search. Four MCP tools let Claude query its own memory mid-session.

The clever part: **progressive disclosure** — search returns a compact index first (~50-100 tokens), then you fetch full details for specific IDs (~500-1000 tokens). Claims ~10x token savings vs. naive injection.

Credibility concern: AGPL-3.0 + PolyForm Noncommercial on the RAG layer, 123 open issues, creator tied it to a Solana memecoin. Interesting architecture, questionable project health.

### Supermemory (supermemory.ai)

Targets *evolving* understanding vs. static RAG. Key differentiator: a **memory graph** that resolves contradictions — when a new memory conflicts with an existing one, it merges/updates/marks stale rather than appending. Standard RAG just accumulates; Supermemory maintains. Claims 85.2% on LongMemEval (#1 on MemoryBench). Fully hosted, $399/mo for Scale tier, embedding model proprietary.

### Mem0 (mem0.ai)

LLM-mediated fact extraction before storage. When you add a conversation, it passes it through a small LLM (`gpt-4.1-nano`) to extract atomic facts ("User is vegetarian", "User prefers TypeScript"). Stores facts, not raw text. On retrieval, vector similarity over *extracted facts*, not chunks. Claims 80% token reduction, 26% better response quality. Open source (Qdrant + Neo4j optional).

The risk: if the extraction LLM misses something important, it's silently lost. Designed for user preference memory, not code-semantic memory.

### The CLAUDE.md / AGENTS.md Pattern

The dominant community pattern for session persistence. A structured markdown file loaded at session start. Two modes:
- **Static**: project overview, tech stack, code style rules, forbidden operations, MCP config
- **Dynamic/auto-maintained**: hooks append decisions, gotchas, WIP state after each session

Community consensus: **~2,000 token ceiling** before the file consumes too much context budget without proportional benefit. Beyond that, performance degrades. No community tool implements TTL or relevance decay — everything is append-only.

`Rulesync` CLI converts between `CLAUDE.md`, `.cursorrules`, `.windsurfrules` bidirectionally. Branch-specific files are an emerging pattern.

### The Karpathy/coleam insight

`claude-memory-compiler` implements what Karpathy calls the "LLM Knowledge Base" architecture: at personal scale (50-500 articles), **a structured index.md + LLM reading outperforms vector similarity**. No RAG needed until ~2,000+ articles. Hooks capture session transcripts → `flush.py` extracts decisions → `compile.py` builds cross-referenced knowledge articles.

This is essentially what Hero does. The spec corpus + knowledge base is a structured index. FTS5 with porter stemming is the right retrieval mechanism at this scale. Hero is architecturally correct.

## Community Agent/Command Patterns

From awesome-claude-code (38k stars) and awesome-claude-agents, the patterns that appear everywhere:

**Commands that appear in nearly every serious workflow:**
- `/plan` or `/design` — spec before code (Hero has this)
- `/review` — code/PR review (Hero has this)
- `/context` or `/prime` — explicitly load context for a new session (Hero has `hero context`, could be a command)
- `/memory` or `/save` — explicit knowledge save (Hero has `/capture` and `/note`)
- `/debug` — structured debugging (Hero has `/diagnose`)
- `/commit` — AI-generated commit message (Hero does NOT have this)
- `/test` — test generation (Hero does NOT have a dedicated one)

**Patterns Hero already covers well:**
- Spec-driven workflow (design → deliver) — this is Hero's core
- Agent specialization and orchestration — 25 agents, all specialized
- Convention/decision knowledge base — `.hero/conventions/`, `.hero/decisions/`
- Context injection before execution — `hero context`
- Nudge awareness when working without a spec

**Patterns Hero does NOT cover:**
1. **Session transcript capture / hook-based auto-logging** — claude-mem does this; Hero has `/capture` but it's manual and agent-driven, not hook-based
2. **Explicit `/context` or `/prime` session start command** — loading relevant context at the *start* of a session is a common pain point; Hero has `hero context` CLI but no agent command for "prime me for this session"
3. **`/commit` command** — AI-generated commit messages with context awareness; extremely common in community repos
4. **`/test` command** — dedicated test generation workflow; Hero has `test-architect` agent but no slash command
5. **Cross-session memory with contradiction resolution** — Hero's knowledge base is append-only; no mechanism to detect when a new note contradicts an old convention

## What Hero Already Does That Most Tools Don't

- **Code-semantic structure** — specs have typed metadata (files_touched, status, relations, tags). Memory tools treat everything as undifferentiated text.
- **Lifecycle management** — specs have states (planning → delivering → completed). Memory tools have no lifecycle.
- **Team coordination** — claiming, conflict detection, convention scopes. All memory tools are single-user.
- **Git-native** — everything committed, diffable, reviewable. Memory tools use databases that live outside the repo.
- **Tool-agnostic** — works with OpenCode, Cursor, Claude Code. Memory tools are mostly Claude Code-specific.

## The MCP Angle

`hero serve` (planned) would expose Hero as an MCP server — meaning agents could query the spec corpus, knowledge base, and context injection mid-reasoning, not just at session start. This is the "buddy model realized" pattern from the earlier note.

The community is doing this manually via tool calls / skill invocations. An MCP server is the right infrastructure because:
- Agents can query mid-session (not just at start)
- Any MCP-compatible tool benefits, not just the ones Hero explicitly supports
- Continuous context, not one-shot injection

This should be mentioned in the README as a planned capability, since it directly answers "how does Hero compare to claude-mem?"

## Key Takeaways

- **Hero is architecturally correct** — structured index at project scale beats RAG. Karpathy's insight validates the approach.
- **The gaps are commands, not architecture** — `/commit`, `/test`, `/prime` (session start context loading) are the most commonly requested things Hero doesn't have
- **Hook-based capture is the one genuine mechanism gap** — claude-mem's real-time `PostToolUse` capture is something Hero can't do today because it requires hooks; `/capture` is manual by comparison
- **Contradiction resolution in the knowledge base is a v0.2 idea** — append-only is fine for now, but as the knowledge base grows, detecting when a new note contradicts an old convention becomes valuable
- **MCP server = the bridge** — `hero serve` as an MCP server makes Hero the always-available knowledge layer, queryable mid-reasoning, which closes the gap with hook-based tools without requiring hooks

## Open Questions

- Should Hero add a `/prime` command — explicit session-start context loading for a given spec or area?
- Should `/commit` be a Hero command, or is that too far from the spec workflow?
- What's the right hook strategy if/when Claude Code hooks are available in a Hero target tool?
- At what knowledge base size does FTS5 become insufficient and semantic search worth adding?
