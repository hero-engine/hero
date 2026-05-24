---
title: hero search response tiering — max_results + pagination instead of compact boolean
slug: hero-search-tiered-response
type: feature
status: planning
priority: P2
created: 2026-05-24
tags: [cli, mcp, search, api, contract, ergonomics]
relations:
  - target: hero-search-json-flag-silently-ignored
    kind: related-to
---

# hero search response tiering — `max_results` + pagination instead of `compact` boolean

## Problem

`mcp__hero__hero_search` today is a binary: `compact: false` returns the full result set with snippets (verbose, expensive); `compact: true` returns just a one-line summary plus a `ref_id` that the caller must round-trip through `hero_expand` to actually see anything. Neither serves the middle ground, which is the most common case:

> "I want to know what specs are about X — show me the top few, and tell me if there are more."

Today an agent has to either:
- Pay the full snippet cost on every search (verbose default), or
- Make two MCP calls every time (`compact` → `expand`).

There is also no pagination. If a search returns 50 hits, you get all 50 inline (verbose) or zero inline (compact). No way to say "give me hits 11–20." `hero_expand` exists but is not a paging primitive; it's a "show me everything for this ref_id" hatch.

The CLI shape mirrors this. `hero search "<q>"` either dumps everything or — with `--budget` — silently truncates and prints a "+N more, refine your query" hint. No `--page`, no `--limit`, no consistent way to walk the result set.

The downstream effect on agents (and on humans): searches feel like a forced choice between "drown in context" and "round-trip dance." The result quality is fine; the response shape is wrong.

## Goals

1. Replace the `compact` boolean with a numeric `max_results` so callers ask for exactly the slice they want.
2. Add pagination so callers can walk the rest of the result set without re-running and de-duping.
3. Always return the total hit count, regardless of how many results are returned, so "does X exist?" is answerable in one call.
4. Keep the CLI and MCP shapes aligned — the same parameters and the same response keys, just rendered differently.
5. Preserve backward compatibility for one release: `compact: true` continues to work and maps to `max_results: 1`. Emit a deprecation note on stderr (CLI) or in the envelope (MCP).

## Non-Goals

- Changing the underlying ranking, the FTS5/graph routing, or the retrieval algorithm. Strictly about response shape.
- Adding cursor-based (opaque token) pagination. Offset+limit is sufficient because the underlying index recomputes the query each call; there's no streamed state to preserve.
- Reworking `hero_expand`. It remains the "show me the full record for a ref_id" hatch and is unaffected.

## Design

### Parameters

| Param | Type | Default | Meaning |
|---|---|---|---|
| `query` | string | (required) | search text |
| `max_results` | int | `5` | how many results to return in the response. `0` is reserved for "metadata only" (count + page info, no results array). Cap at `50` to prevent accidental fire-hose. |
| `page` | int | `1` | 1-indexed page number. `page * max_results` is the offset into the ranked result set. |
| `type`, `status`, `subproject` | string | — | unchanged filters |
| `compact` | bool | `false` | **deprecated**. If true, equivalent to `max_results: 1`. Emits a deprecation note. Removed in next minor. |

### Response shape (MCP)

```json
{
  "query": "domain swap",
  "total_hits": 23,
  "page": 1,
  "page_size": 5,
  "total_pages": 5,
  "has_next": true,
  "results": [
    {"type": "feature", "key": "domain-routing-and-agents", "title": "...", "status": "completed", "snippet": "..."},
    ...
  ]
}
```

Key choices:
- `total_hits` is always present — answers "does X exist?" in one call even when `max_results: 0`.
- `has_next` is a convenience boolean so consumers don't have to compute `page < total_pages`.
- `results` is empty (`[]`) when `max_results: 0` or `page > total_pages`.
- No `ref_id` / `expand_via` envelope — `page` is the paging primitive. `hero_expand` remains for non-search records.

### CLI shape

```
hero search "<query>" [--max-results N] [--page N] [--type X] [--status Y] [--json]
```

- Default human output: same compact markdown list it emits today, but with a footer line: `Showing 1–5 of 23 (page 1/5). Next: hero search "..." --page 2`.
- `--json` emits the response shape above verbatim.
- `--budget` (current token-budget knob) is **kept** as an orthogonal concern but applies only to graph-path verbose output; `max_results` takes precedence when both are set.
- `--max-results 0 --json` gives scripts a clean "just tell me how many" call: `{"total_hits": 23, "results": []}`.

### Backward compatibility

| Old call | Behavior after this change |
|---|---|
| `hero_search(query)` (MCP, default) | Returns 5 results in new envelope. Previously returned ~20 with full snippets. **Breaking visual change** but parseable consumers tolerate it (snippet key still present). |
| `hero_search(query, compact: true)` | Returns 1 result + `total_hits`. Emits `_deprecated: "use max_results: 1"` in envelope. |
| `hero search "<q>"` (CLI, default) | Shows top 5 + footer. Previously dumped up to ~20. |
| `hero search "<q>" --budget 800` | Honored, but only for the graph-path human format. |

Breaking the default verbosity is the explicit intent — the current default is unergonomic. A short release note in `CHANGELOG.md` and a one-line deprecation in MCP envelope output for `compact: true` covers the migration.

### Where `compact:true` lands semantically

The user's clarification — *"compact isn't count-only, it's top-hit only"* — settles a small ambiguity. After this change:
- "Top hit only" = `max_results: 1` (default `page: 1`). Returns the single best match plus `total_hits`.
- "Count only" = `max_results: 0`. Returns no results, just metadata.

Both are first-class. Neither requires the `compact` keyword anymore.

## Implementation Sketch

1. **`internal/retrieval/query.go`**: extend `Query` with `MaxResults` and `Page` (replacing/augmenting `Limit`). Plumb through `retrieval.Retrieve`. Compute `total_hits` independently of slicing — the underlying FTS5/graph calls already return a full ranked list internally; truncate in the formatter, not the index.
2. **`internal/mcp/.../search.go`** (locate the hero_search MCP handler): rebuild the envelope per the response shape above. Honor legacy `compact: true` → `MaxResults: 1` with deprecation marker.
3. **`internal/cli/search.go`**: add `--max-results` and `--page` flags. Update `printFTSResults`, `printGraphResults`, and the JSON path to emit the new envelope. The recently-fixed `--json` path (see `hero-search-json-flag-silently-ignored`) becomes the canonical shape — human output is a render of the same envelope.
4. **`internal/cli/search_test.go` + MCP test suite**: paging tests (request page 2 of a 12-result corpus, assert no overlap with page 1), `max_results: 0` test, `compact: true` backward-compat + deprecation marker test, default-output test (5 results, footer present).
5. **Docs**: update CLI `--help`, MCP tool description in the registry, and any agent-facing docs (`domains/engineering/AGENTS.md` routing table mentions `compact: true` — switch the wording to `max_results: 1` once shipped).

## Risks

- **Behavior change for existing CLI consumers.** Scripts that grep for `N result(s)` footer or count lines will need to switch to `--json` + parsing. Mitigated by: the JSON shape is now stable and worth scripting against; the human footer keeps a result count.
- **Cap at 50 might surprise heavy users.** Anyone genuinely wanting more should page or use `hero_expand` against a ref_id. The cap exists to prevent runaway costs in agent loops. Configurable via `hero.json` if needed (out of scope unless asked).
- **`--budget` (token budget) and `--max-results` interact awkwardly on the graph path.** Resolved by precedence: `--max-results` truncates after the budget calculation. Document explicitly.

## Acceptance Criteria

- `mcp__hero__hero_search(query, max_results: 1)` returns `{total_hits, page, page_size, has_next, results: [<single hit>]}`.
- `mcp__hero__hero_search(query, max_results: 0)` returns the envelope with `results: []` and the correct `total_hits`.
- `mcp__hero__hero_search(query, page: 2)` returns hits 6–10 (with default `max_results: 5`) and `has_next: true` if more exist.
- `mcp__hero__hero_search(query, compact: true)` still works, returns the `max_results: 1` envelope, and includes a deprecation marker.
- `hero search "<q>"` CLI default shows ≤ 5 results with a "Showing 1–5 of N, next: …" footer.
- `hero search "<q>" --max-results 0 --json` emits `{"total_hits": N, "results": []}` (or the agreed metadata-only shape).
- Tests cover pagination, max_results, `compact` backward-compat, and default-output footer.

## Open Questions

1. **Cap value.** `50` is a guess. Reasonable? Or higher (`100`) given the index is local?
2. **`compact` deprecation window.** One minor release, or two?
3. **MCP envelope structure.** The spec proposes a plain object. The existing MCP convention in this repo uses `[hero envelope] ... [/hero envelope]` framing (see `hero_search(compact:true)` response today). Should the new shape stay inside that frame, or move to native JSON? Recommend: native JSON, since this tool already returns structured data and frame-wrapping obscures the shape from MCP consumers that aren't parsing the frame.
