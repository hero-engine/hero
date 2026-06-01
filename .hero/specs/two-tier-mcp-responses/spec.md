---
title: Two-Tier MCP Responses — Compact Summaries with `hero_expand` for Full Content
slug: two-tier-mcp-responses
type: feature
status: completed
tags: [mcp, context, breadcrumbs, tokens, hero-expand, response-shape]
created: 2026-05-01
relations:
  - target: active-context-management
    kind: child-of
  - target: hero-context-pipe
    kind: related
  - target: context-injection
    kind: related
horizon: now
completed_at: 2026-05-18T19:25:38Z
---

## Kickoff

Make every Hero read-side MCP tool return a compact summary plus a `ref_id` by default. Full content is fetched on demand via a new `hero_expand <ref_id>` tool. The model gets the gist for free; it pays the token cost only for content it actually needs.

**Status:** planning — design captured, no code yet.

**Pick up at:** decide ref_id format and storage backing (in-memory session cache vs SQLite session table), then prototype on a single read-side tool (`hero_read_spec`) before fanning out.

## Goal

Cut tokens-per-task on Hero MCP calls by 40–70% on the read-heavy workflows (spec exploration, search, why/blocked traversal) without losing the ability to drill in. Establish the breadcrumb/expand pattern as the **default shape** for all Hero read-side tools so the rest of the active-context-management work has a substrate to build on.

## Problem

Today, read-side Hero MCP tools dump full content because there's no way for the model to ask later. `hero_read_spec` returns 4–8k tokens of markdown; `hero_search` returns full bodies for every hit; `hero_context` returns every applicable convention verbatim. Most of that content informs a single decision and then sits in conversation history wasting tokens for the rest of the session.

The model often only needed the title and a one-line summary to know whether the spec was relevant. The full body was an over-fetch.

There's no single primitive — and no shared envelope shape — that lets a tool say "here's the gist; here's how to ask for more if you need it."

## Design

### Response envelope — additive, backwards-compatible

Every Hero read-side MCP tool gains two new fields in its response, alongside whatever it already returns:

```json
{
  "summary": "...short token-efficient gist...",
  "ref_id": "spec:hero-ask:full",
  "expand_via": "hero_expand",

  // existing fields preserved
  "spec_id": "hero-ask",
  "title": "...",
  "body": "...full markdown..."
}
```

**Default behavior changes by mode (see "Rollout phases" below):**

- **Phase 1 (opt-in):** new fields added; full content still returned. Clients opting in via `compact: true` parameter get the trimmed shape.
- **Phase 2 (default-on):** trimmed shape becomes default; full content omitted unless `compact: false` or explicit expansion.
- **Phase 3 (legacy retired):** full-content-by-default removed; clients must use `hero_expand` for verbatim bodies.

Old clients keep working through Phase 2. Phase 3 only happens after measurement confirms the win.

### `hero_expand <ref_id>` — the rehydration tool

A new MCP tool with a single string parameter:

```
hero_expand(ref_id: string) → { content: string, source: string, fetched_at: string }
```

Resolution path:

1. Look up `ref_id` in the session ref-store (in-memory or SQLite-backed; see "Storage" below)
2. If found and fresh → return cached content
3. If found but stale (file edited since cache, source changed) → re-fetch from source, update cache, return
4. If not found → structured error with re-fetch hint: `{"error": "expired", "rehydrate_via": "hero_read_spec", "args": {"id": "hero-ask"}}`

The model treats `hero_expand` as the universal "give me the full thing" tool. It does not need to know which original tool produced the ref_id.

### ref_id format

`<kind>:<slug>:<scope>` — predictable, parseable, debuggable.

Examples:
- `spec:hero-ask:full` — full body of spec hero-ask
- `spec:hero-ask:section-design` — just the design section
- `search:7f3a:results` — full result set for search query 7f3a
- `convention:api-handlers:full`
- `context:internal-api-auth:bundle` — the full context bundle for a file group
- `recap:2026-05-01:full` — full activity digest for a date

Kinds map to resolvers — each tool registers a resolver for the kinds it produces. Adding a new tool means registering one resolver, not adding cases to a switch.

### Per-tool summary contracts

Each tool defines what its summary contains. Keep these tight — the value is in being aggressively short.

**`hero_read_spec`:**
```
title + status + 2-sentence essence + key decisions (1-line each, max 3)
```
Target: 80–200 tokens vs current ~4000.

**`hero_search`:**
```
hit count + per-hit: { title, kind, status, 1-line snippet, ref_id }
```
Target: ~50 tokens per hit vs current ~500.

**`hero_context`:**
```
file group + applicable convention/decision/rule titles with ref_ids (no bodies)
```
Target: 200–400 tokens vs current ~2000–6000.

**`hero_why` / `hero_blocked` / `hero_feed` / `hero_recap`:**
```
top-level structure + node titles + ref_ids; bodies omitted
```

Each contract lives in the tool's MCP description so the model knows what it's getting and how to ask for more.

### Storage — session ref-store

**Recommendation: SQLite-backed session table** (`.hero/sessions/<id>/refs.db`).

Why not pure in-memory:
- Sessions can span MCP server restarts
- The ledger work in `active-context-management` will want the same store
- Already have SQLite infra (`graph.db`, `index.db`) — incremental cost is low

Schema sketch:
```sql
CREATE TABLE refs (
  ref_id TEXT PRIMARY KEY,
  kind TEXT NOT NULL,
  slug TEXT NOT NULL,
  scope TEXT NOT NULL,
  source_args_json TEXT NOT NULL,  -- args to re-run the producer
  cached_content TEXT,              -- nullable; populated on first expand
  source_fingerprint TEXT,          -- file hash, query hash, etc — for staleness check
  created_at INTEGER NOT NULL,
  expires_at INTEGER                 -- nullable
);
```

**Eviction:** TTL per kind (e.g., spec refs: 1 day; search refs: 1 hour). Background prune on session start.

### Backwards compatibility

- Phase 1 ships purely additive. Existing callers see new fields and ignore them. New `compact: true` parameter opts into trimmed shape. Zero risk of regression.
- Phase 2 flips default after measurement on real sessions. Callers needing legacy shape pass `compact: false`.
- Phase 3 retires the legacy field. Only ship after telemetry shows reconstitution rate is in target range and quality didn't regress.

### What about non-Hero MCP tools?

Out of scope. This spec is Hero MCP only. Other servers' tools keep doing whatever they do; the Curator (in active-context-management) will treat them as opaque verbatim sources.

## Open Questions

1. **ref_id stability across sessions** — should `spec:hero-ask:full` mean the same content in session A and session B? Recommend yes for shareable kinds (spec, convention, decision); session-scoped for query-result kinds (`search:*`, `context:*`).
2. **Section-level refs** — should we support `spec:hero-ask:section-design`? Recommend defer to v2; ship full-body refs first, add section parsing if telemetry shows demand.
3. **`hero_expand` batch mode** — should `hero_expand` accept multiple ref_ids in one call? Recommend yes from day one — `hero_expand([ref1, ref2])` is one tool call vs N. Cheap to add.
4. **Cache hit observability** — surface ref-store hit/miss to `hero pulse`? Recommend yes; this is the measurement that tells us if Phase 2 is safe.

## Acceptance Criteria

- WHEN a Hero read-side MCP tool is invoked with `compact: true` THE SYSTEM SHALL return only `summary`, `ref_id`, `expand_via`, and minimal identity fields (no verbatim body)
- WHEN a Hero read-side MCP tool is invoked without `compact` parameter (Phase 1) THE SYSTEM SHALL include `summary`, `ref_id`, and `expand_via` fields alongside legacy full content
- WHEN `hero_expand` is invoked with a known fresh `ref_id` THE SYSTEM SHALL return the cached full content
- WHEN `hero_expand` is invoked with a known stale `ref_id` THE SYSTEM SHALL re-fetch from source, update cache, and return current content
- WHEN `hero_expand` is invoked with an unknown or expired `ref_id` THE SYSTEM SHALL return a structured error including a `rehydrate_via` hint naming the producer tool and original args
- WHEN `hero_expand` is invoked with an array of ref_ids THE SYSTEM SHALL resolve each and return results in array order
- WHILE a session is active THE SYSTEM SHALL persist ref-store entries to `.hero/sessions/<id>/refs.db`
- WHILE the ref-store contains entries past their TTL THE SYSTEM SHALL prune them on next session start or on-demand prune call
- IF a ref_id's source fingerprint differs from cache at expand time THEN THE SYSTEM SHALL re-fetch and update cache before returning
- IF a tool produces a ref_id THEN THE SYSTEM SHALL register a resolver for that kind so `hero_expand` can rehydrate it
- WHERE the kind is shareable across sessions (spec, convention, decision, rule) THE SYSTEM SHALL use stable ref_ids that resolve identically across sessions
- WHERE the kind is query-scoped (search, context, recap) THE SYSTEM SHALL use session-scoped ref_ids
- THE SYSTEM SHALL document each tool's summary contract in its MCP tool description
- THE SYSTEM SHALL surface ref-store hit rate, miss rate, and re-fetch rate via `hero pulse --refs`

## Rollout Phases

**Phase 1 — Opt-in (ship first):**
- New fields added to all read-side Hero MCP tool responses
- `compact: true` parameter trims to summary-only
- `hero_expand` shipped
- Default behavior unchanged
- Acceptance: agents that opt in see no quality regression; baseline tokens-per-task measured

**Phase 2 — Default-on (after measurement):**
- Trim becomes default; `compact: false` returns legacy shape
- Engineer agent and delivery leads updated to use `hero_expand` reflexively
- Acceptance: reconstitution rate in 5–25% target window; tokens-per-task down ≥30% on representative tasks

**Phase 3 — Legacy retired (optional, opt-in by user):**
- `compact: false` becomes a no-op or deprecated
- Verbatim bodies only via `hero_expand`
- Acceptance: ≥4 weeks of clean Phase 2 telemetry; user opt-in via `hero.json`

## Measurement

Captured per session, exposed via `hero pulse --refs`:

- **tokens_saved_estimate** — sum of (full_body_tokens − summary_tokens) for refs not subsequently expanded
- **expand_rate** — fraction of ref_ids that get expanded back. Target window: 5–25%.
- **expand_latency** — P50/P95 wall-clock for `hero_expand` calls
- **stale_refetch_rate** — fraction of expands that triggered re-fetch due to fingerprint change
- **quality_regression_signal** — track agent error rates and re-asks per task; compare Phase 1 (opt-in) cohort vs control

If `expand_rate < 5%`, summaries may be too rich — trim further. If `expand_rate > 25%`, summaries are too thin — expand contracts.

## Out of Scope / Boundaries

- Non-Hero MCP tools — left untouched in this spec
- Mutation tools (`hero_claim`, `hero_event`, `hero_verify`, etc.) — they don't return content for context; no change
- Section-level refs — deferred until full-body refs prove out
- Cross-tool ref aliasing — each tool registers its own kinds; no shared aliases yet
- Direct integration with the Curator (active-context-management) — that work consumes this primitive, doesn't get implemented here
- Format negotiation per harness — `hero-context-pipe` already covers that orthogonally

## Changes (anticipated)

- `internal/mcp/envelope.go` — shared response envelope helpers (`summary`, `ref_id`, `expand_via`)
- `internal/mcp/refs.go` — ref-store backed by SQLite (`refs.db`); register/lookup/expire
- `internal/mcp/expand.go` — `hero_expand` tool implementation; resolver registry
- `internal/mcp/tools/read_spec.go` — emit summary + register resolver for `spec:*`
- `internal/mcp/tools/search.go` — emit summary + register resolver for `search:*`
- `internal/mcp/tools/context.go` — emit summary + register resolver for `context:*`
- `internal/mcp/tools/why.go`, `blocked.go`, `feed.go`, `recap.go` — same pattern
- `internal/cli/pulse.go` — `--refs` flag surfacing hit/miss/refetch metrics
- MCP tool descriptions updated with per-tool summary contract
- `agents/engineer.md` and delivery-lead agents — note that `hero_expand` is the canonical drill-in
