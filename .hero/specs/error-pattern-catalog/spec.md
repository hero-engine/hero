---
title: Error Pattern Catalog — Stack-Specific Error Knowledge Base
slug: error-pattern-catalog
type: feature
status: completed
priority: medium
tags: [code-intelligence, diagnose, knowledge, error-patterns]
created: 2026-04-18
horizon: now
---

## Goal

Build a catalog of common error patterns specific to the project's stack, accumulated from `/diagnose` sessions. When a user encounters an error, surface relevant patterns via `hero_context` so the agent starts with institutional knowledge rather than reasoning from scratch.

## Problem

Diagnose sessions often rediscover the same root causes. "Connection refused on port 5432" always means the local Postgres container isn't running. "Cannot read property of undefined" in a specific module always traces back to a missing null check on the API response. This knowledge exists in engineers' heads but isn't captured anywhere structured.

## Design

### Storage

Error patterns live in `.hero/knowledge/error-patterns/` as individual YAML or Markdown files:

```
.hero/knowledge/error-patterns/
  postgres-connection-refused.md
  nil-pointer-config-load.md
  ...
```

Each pattern file:

```yaml
---
pattern: "connection refused.*:5432"
stack: [go, postgres]
severity: common
files: ["internal/db/connect.go"]
---

## Symptom
Connection refused errors when starting the service locally.

## Root Cause
Local Postgres container is not running or is on a non-default port.

## Fix
Run `docker compose up -d postgres` or check `DATABASE_URL` in `.env`.
```

### Accumulation

At the end of a `/diagnose` session that identifies a root cause, the agent should check if the pattern already exists. If not, prompt to save it. The pattern should capture: regex match for the error, relevant files, root cause, and fix.

### Surfacing

The `hero_context` MCP tool should, when queried about files involved in a known error pattern, include the pattern in its response. This requires the context formatter (`internal/context/format.go`) to load and match patterns against the queried file paths.

### Matching

Pattern matching should be simple: regex on error text, plus file-path overlap. No need for fuzzy matching initially — exact regex and file-path intersection is sufficient.

## Boundaries

- Patterns are local to the project, not global across workspaces.
- No automatic pattern detection from logs — patterns are only created through diagnose sessions.
- No deduplication logic beyond the agent checking before saving.
- V1 is read/write of flat files plus context integration. No database, no indexing.
