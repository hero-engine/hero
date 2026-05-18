---
title: Deep Mode — LLM-Enhanced Symbol Descriptions
slug: deep-mode-llm-descriptions
type: feature
status: completed
priority: medium
tags: [code-intelligence, codescan, llm, deep-mode]
created: 2026-04-18
horizon: now
---

## Goal

Implement the `code_scan.depth: "deep"` mode that runs an LLM pass over scanned symbols to generate richer one-line descriptions than heuristic parsing can extract. The config field already exists but the LLM pass is not implemented.

## Problem

Heuristic parsing extracts function signatures and doc comments, but many symbols lack useful documentation. A function named `processItems` with no doc comment gets a generic description. An LLM can read the function body and produce something like "Filters expired items and batches the rest into groups of 50 for upstream processing."

## Design

### Trigger

When `code_scan.depth` is set to `"deep"` in `hero.json`, the scan pipeline adds an LLM enrichment pass after heuristic parsing completes.

### Flow

1. Heuristic scan runs as normal, producing the symbol index.
2. For each symbol that has a weak or missing description, extract the symbol's source code (function body, type definition, etc.).
3. Batch symbols and send to the configured LLM with a prompt requesting a concise one-line description.
4. Merge LLM descriptions into the index, flagging them as `source: "llm"` so they can be distinguished from doc-comment-derived descriptions.

### Batching and Cost Control

- Group symbols by file to provide file-level context.
- Set a configurable max token budget (`code_scan.deep_budget`) to cap LLM spending.
- Skip symbols that already have good descriptions (doc comments > 10 words).
- Cache LLM descriptions keyed by symbol content hash — only re-run when source changes.

### LLM Integration

Use the same LLM provider configuration the rest of Hero uses (via `hero.json` provider settings). The scan command already has access to config; it needs to instantiate an LLM client.

### Cache

Store LLM-generated descriptions in `.hero/cache/deep-descriptions.json` keyed by `file:symbol:contentHash`. Invalidate on source change.

## Boundaries

- Deep mode is opt-in and off by default. The `"standard"` depth remains the default.
- No LLM calls during `hero_context` or `hero_code` — enrichment only happens at scan time.
- If the LLM is unavailable or errors, fall back to heuristic descriptions silently.
- V1 targets function/method descriptions only. Type and package-level summaries are future work.
- No streaming — batch requests are fine since this runs as a background scan.
