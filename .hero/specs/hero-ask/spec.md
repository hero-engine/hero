---
title: hero ask — Semantic Query Against Knowledge and Specs
slug: hero-ask
type: feature
status: completed
milestone: v0.3
tags: [query, knowledge, search, semantic, cli]
created: 2026-04-13
relations:
  - target: knowledge-contradiction-detection
    kind: extends
  - target: hero-serve-daemon
    kind: related
horizon: now
---

## Goal

Give engineers and agents a single command to ask a natural-language question against Hero's knowledge base and specs, getting a direct, grounded answer with citations — not a list of search results to manually sift through.

## Problem

Hero currently has two ways to retrieve knowledge:

1. `hero context <files>` — file-path-driven, returns everything relevant to given paths
2. `hero search <query>` — keyword search, returns matching spec/knowledge titles and excerpts

Neither answers a question. If an agent wants to know "what key prefix does badger use for peer records?" or "what did we decide about central servers?", it has to do a search, retrieve N docs, read them all, and synthesize. That's model work that Hero should handle — the answer already exists in the knowledge base.

`hero ask` closes this gap: one command, one answer, grounded in the corpus.

## Design

### CLI Interface

```
hero ask "what key prefix does badger use for peer records?"
hero ask "why did we choose badger over sqlite?"
hero ask "what are the rules for gin handler error responses?"
hero ask --json "what CI checks are required?"
```

**Flags:**
- `--json` — output structured JSON with answer + citations
- `--citations` — always show source slugs/paths (default: shown when answer references multiple docs)
- `--type <type>` — restrict search to knowledge type: `convention`, `decision`, `context`, `rule`
- `--limit <n>` — max number of knowledge entries to search (default: 20)

### Answer Pipeline

`hero ask` is intentionally **not an LLM call**. It operates through a structured retrieval + extraction pipeline:

1. **Tokenize the question** — extract key nouns, verbs, identifiers (package names, file names, quoted strings)
2. **Retrieve candidates** — ranked full-text search across `.hero/` knowledge + specs (same engine as `hero search`)
3. **Extract answer sentences** — for each candidate, score sentences that contain the question tokens; surface the top-scoring passage
4. **Compose answer** — stitch the top passages into a short answer with citation slugs
5. **Return** — plaintext answer to stdout, or JSON if `--json`

This is deterministic, fast, and requires no API key. The quality of the answer is directly proportional to the quality of the knowledge base — which is the right incentive.

### Output Format

**Plaintext (default):**
```
$ hero ask "what key prefix does badger use for peer records?"

Peer records use the prefix "peer:" followed by the peer ID.
GC records use "gc:peers". Change records use "changes:" + peer_id + ":" + seq.

Sources: conventions/badger-storage
```

**JSON (`--json`):**
```json
{
  "question": "what key prefix does badger use for peer records?",
  "answer": "Peer records use the prefix \"peer:\" followed by the peer ID. GC records use \"gc:peers\". Change records use \"changes:\" + peer_id + \"+\" + seq.",
  "citations": [
    {
      "slug": "conventions/badger-storage",
      "path": ".hero/conventions/badger-storage/spec.md",
      "excerpt": "peer: + peer_id — full peer record"
    }
  ],
  "confidence": "high"
}
```

**Confidence levels:**
- `high` — answer passage directly contains the exact question tokens
- `medium` — answer is inferred from related passages
- `low` — no strong match; answer is best-effort from top result

When confidence is `low`, the output notes it and suggests `hero search` for deeper exploration.

### MCP Tool Integration

`hero ask` is also exposed as MCP tool `hero_ask` via `hero mcp`:

```json
{
  "name": "hero_ask",
  "description": "Answer a natural-language question using the project knowledge base",
  "inputSchema": {
    "type": "object",
    "properties": {
      "question": { "type": "string" },
      "type": { "type": "string", "enum": ["convention", "decision", "context", "rule"] }
    },
    "required": ["question"]
  }
}
```

This is the highest-value MCP tool for agents: instead of retrieving and synthesizing multiple docs, the agent asks one question and gets a grounded answer.

### Failure Modes

- **No strong match** — returns `"No knowledge found for this question."` with confidence `low` and a hint to use `hero search`
- **Multiple conflicting answers** — returns all passages and notes the conflict (integrates with knowledge-contradiction-detection)
- **Empty knowledge base** — returns a clear error message directing user to `hero scan` or manual knowledge authoring

## Changes

- `internal/cli/ask.go` — new `ask` command, question tokenization, pipeline orchestration
- `internal/search/extractor.go` — passage extraction + scoring logic (new, shared with search)
- `internal/serve/mcp.go` — add `hero_ask` tool to MCP registry
- `internal/search/` — answer composition, confidence scoring

## Acceptance Criteria

- `hero ask "<question>"` returns a readable answer with source citations
- `--json` returns structured output matching the schema above
- Answers are grounded — every claim maps to a cited knowledge entry
- Confidence is reported accurately: `high` only when question tokens appear in the answer passage
- `hero_ask` MCP tool works end-to-end via `hero mcp`
- Conflicting knowledge entries surface the conflict rather than silently picking one
- No LLM API calls — fully deterministic, offline-capable
- Works with zero knowledge base (graceful error, not a panic)

## Boundaries

- Does **not** make LLM API calls — Hero is a context layer, not a model orchestrator
- Does **not** synthesize across more than the top-ranked passages — answer is extractive, not generative
- Does **not** update or write knowledge entries based on the question
- `--type` filter restricts search scope but does not change the extraction algorithm
