---
title: Buddy Model Architecture — Two-Model Tag Team Vision
slug: buddy-model-architecture
type: note
status: active
tags: [architecture, ai-models, hero-cloud, vision]
created: 2026-04-12
horizon: next
---

# Buddy Model Architecture — Two-Model Tag Team Vision

## The Question

What if instead of crafting a perfect context window upfront for a stateless big model, you could pair it with a small continuously-retrained "buddy model" that natively knows the project? A tag-team where the buddy participates layer-by-layer during reasoning, not just at the start.

For a large team, multiple developers working on tangential things would share one buddy model — it knows the whole codebase, all the conventions, who's working on what, where the bugs cluster.

## The Analysis

### The core limitation is real
Big models like Claude are stateless. Every request starts cold. The only way to influence behavior is through context — system prompts, tool results, conversation history. Hero is fundamentally a context engineering system: it structures project knowledge so the right information lands in the window at the right time.

That IS a workaround for the inability to learn a project natively.

### Fine-tuning doesn't solve it (today)
- Fine-tuned small models learn statistical patterns, not semantic understanding
- Retraining on every commit is prohibitively expensive and slow (hours, not seconds)
- Fine-tuned models hallucinate confidently about stale knowledge
- A fine-tuned 7B model giving advice would be wrong often enough to be net-negative

### RAG is the practical middle ground
Retrieval-augmented generation — index the knowledge, retrieve relevant chunks at query time — is what Hero actually implements. FTS5 index, convention scopes, context builder: that's RAG without a vector database.

### MCP/tool-calling is the closest "side by side" mechanism today
The model calling tools mid-reasoning IS the buddy interaction pattern, just implemented as tool calls against a structured corpus rather than a second neural network. When the model calls `hero context imports --files src/auth.go`, it's asking the project knowledge base "what do I need to know?"

### The gap: continuous vs. one-shot
Today context injection happens primarily at the start of a session. The ideal is continuous querying during reasoning — at every decision point, not just the first one. This is technically possible with MCP. Hero could expose an MCP server queried mid-reasoning:

```
model: "I'm about to add a new auth endpoint"
→ tool call: hero.context("auth", "endpoint", "api")  
← "Convention: all API endpoints use middleware chain. Decision ADR-007: JWT not sessions. Active: Alice refactoring auth middleware. Risk: auth.go had 3 bugs recently."
model: *adjusts approach*
```

This is a UX/integration problem, not a fundamental technical limitation.

## Is Hero the right path?

**Yes, for now.** Hero is the best practical implementation given current model architecture. The "buddy model" intuition is correct about what's missing, but the implementation that works today is structured retrieval, not fine-tuning.

The gap narrows as models get:
- Persistent memory across sessions
- Better tool-use patterns (more frequent, lower-latency tool calls)
- Longer and cheaper context windows

Hero's knowledge structure will be the thing those future capabilities plug into. The investment in structuring project knowledge isn't wasted — it's infrastructure that becomes more valuable as models improve.

## The team dimension → Hero Cloud

For a large team sharing one "buddy":
- **CLI** = individual knowledge layer (free, local)
- **Cloud** = shared knowledge service (cross-repo knowledge graph, team activity awareness, shared conventions)
- Not a second model — a shared knowledge service that any model can query

## Key Insight

Hero is both the workaround AND the foundation. It solves the immediate problem (smart context injection) while building the knowledge infrastructure that future model capabilities will consume. The buddy model vision may eventually be achievable through persistent model memory or continuous fine-tuning, but the structured knowledge layer Hero provides will still be the source of truth.

---

## Follow-up: `hero serve` — The Buddy Model Realized (2026-04-12)

The conversation continued into what `hero serve` should be. The key realization: MCP is not a cloud feature — it's a **local CLI feature** that makes Hero the always-available knowledge layer.

### `hero serve` = MCP + Watcher + HTTP API + Event Stream

Instead of just an MCP server, `hero serve` becomes a local daemon bundling four subsystems:

1. **MCP Server** — AI agents query Hero tools mid-reasoning (context, search, check, nudge). Auto-registered by `hero install`. This IS the buddy model interaction pattern.
2. **File Watcher** — the watch mode we built runs as a subsystem, keeping the index always fresh. No separate process.
3. **HTTP API** — local REST endpoints for dashboards, editor extensions, CI scripts, cloud sync.
4. **Event Stream (SSE)** — live updates for spec changes, health checks, status transitions.

### Where does the money come from?

The local daemon is free forever — it knows one project. Hero Cloud aggregates across repos and teams:
- Cross-repo knowledge ("how did team X solve this in repo Y?")
- Team awareness ("who else is working in this area across all repos?")
- Shared conventions pushed to all local daemons
- Analytics (velocity, completion rates, convention compliance)

**The local MCP server is the adoption driver. The cloud knowledge graph is the business.**

See spec: `hero-serve-daemon` for full design.
