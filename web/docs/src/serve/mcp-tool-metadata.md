# MCP Tool Metadata — Category & Tier (v1)

Hero's MCP server advertises ~60 `hero_*` tools. Their schemas are the single
largest fixed token cost at the start of a request, and most tools are not
needed on any given turn. To let a harness defer schemas it does not need
*without* each harness re-inventing which Hero tools matter, Hero labels every
tool in `tools/list` with two facets a client can read directly.

**This is a contract you can pin to.** The keys, the tier values, and the
backward-compatibility guarantee below are stable for v1. The category set is a
closed enum that may *gain* members in a later version but will not rename or
repurpose existing ones. Mirror these labels; do not hardcode your own.

## Where the labels live

Both facets are emitted in each tool's MCP `_meta`, under keys namespaced to
Hero so they never collide with another server's metadata:

```jsonc
{
  "name": "hero_search",
  "description": "...",
  "inputSchema": { "...": "..." },
  "annotations": { "readOnlyHint": true, "idempotentHint": true },
  "_meta": {
    "hero.dev/category": "search-and-knowledge",
    "hero.dev/tier": "eager"
  }
}
```

- `hero.dev/category` — the functional family (for grouping and lookup).
- `hero.dev/tier` — Hero's advisory recommendation on eager-vs-deferrable.

> **Namespace note.** `hero.dev/` is Hero's current `_meta` namespace (it is
> also where the Attention v1 contract emits `hero.dev/effect` / `hero.dev/consent`).
> It is a convention for collision-avoidance, not a claim to an owned domain, and
> the prefix may be renamed to an owned namespace in a future contract version —
> a versioned change with migration, not a silent break. Read the keys through a
> single constant so a future rename is a one-line change on your side.

## `hero.dev/tier` — what to broadcast, what to defer

| Value | Meaning |
|---|---|
| `eager` | Session-warmup tools a harness should broadcast up front (full schema sent every turn). |
| `deferrable` | Safe to list by name only and load the schema on first use. |

**Tier is advisory. A harness MAY override it.** Which tools are hot depends on
the harness and the workflow; Hero only supplies a defensible default so you do
not have to hand-pick an eager set. Hero currently marks exactly six tools
`eager` — the "orient, find, pick work" loop:

`hero_context`, `hero_anchor`, `hero_search`, `hero_status`, `hero_list`,
`hero_queue`.

Everything else is `deferrable`.

## `hero.dev/category` — the closed taxonomy (v1)

Every tool carries exactly one category from this set. Source of truth: the
`ToolCategory` enum in `internal/serve/mcp_protocol.go` — this table is derived
from it.

| Category | What belongs here |
|---|---|
| `search-and-knowledge` | Retrieval and Q&A over specs, knowledge, provenance, and saved skills (`hero_search`, `hero_ask`, `hero_knowledge`, `hero_why`, `hero_read_spec`, …). |
| `spec-lifecycle` | Author, claim, score, verify, plan, diagnose, and the test/demo artifacts of a spec (`hero_claim`, `hero_score`, `hero_verify`, `hero_plan`, `hero_diagnose`, …). |
| `planning-and-status` | Workspace status, ready-work queues, and initiative/session steering (`hero_status`, `hero_queue`, `hero_kickoff`, `hero_goal`, `hero_pulse`, …). |
| `coverage-and-quality` | Heuristic analysis of drift, coverage, conflicts, blockers, blast radius, and CI risk (`hero_drift`, `hero_coverage`, `hero_conflicts`, `hero_blocked`, `hero_impact`, `hero_ci`, …). |
| `activity-and-metrics` | Cross-session activity feed and sprint/velocity reporting (`hero_feed`, `hero_recap`, `hero_velocity`, `hero_insights`, …). |
| `code-intelligence` | Code-symbol search, enrichment, synthesis, and error-pattern capture (`hero_code`, `hero_enrich`, `hero_synthesize`, `hero_error_pattern`). |
| `attention-and-mail` | The Attention window, Project Mail, and Focus (`hero_attention_*`, `hero_mail_*`, `hero_focus_*`). |
| `external-integrations` | Credential-brokered tracker and code-host operations against external systems (`hero_tracker_*`, code-host tools). |

A discovery query like "spec lifecycle" or "cross-repo" should resolve against
these categories rather than tool-name prefixes.

## Safety class is a separate axis — read `annotations`, not `_meta`

Whether a tool is safe to call unprompted is **not** a category question. It
lives in standard MCP `annotations`:

- `readOnlyHint: true` — the tool only reads; safe to call without confirmation.
- `readOnlyHint: false` — the tool writes state; treat as an action.
- `destructiveHint` / `idempotentHint` / `openWorldHint` — as per the MCP spec.

Every Hero tool now carries annotations. Category answers *what family*;
annotations answer *is it safe to call*. Do not infer one from the other — a
`search-and-knowledge` tool is read-only, but a `spec-lifecycle` tool may read
(`hero_score`) or write (`hero_claim`).

## Backward-compatibility guarantee

The category/tier labels are **purely additive**. They appear only inside
`_meta`, never as top-level `ToolDefinition` fields. A client that ignores
`_meta` sees exactly today's `tools/list` and every tool remains callable
unchanged. Adopting the labels is opt-in; there is no breaking change and no
version negotiation required.

## What Hero does *not* provide

Hero emits **static labels** and nothing more. It does **not** ship a
`discover_tools` meta-tool, a tool-search endpoint, or any runtime deferral
mechanism. Managing an eager/deferred window and loading schemas on demand is
the harness's job, because only the harness owns its context window. Hero's
contribution ends at the labels; you build the broadcast-plus-lookup on top of
them.

Reference implementations of the harness side:

- **Claude Code** — `ToolSearch` over the deferred set.
- **hero-code** — `discover_tools` over the deferred set
  (spec `tool-deferred-schema-loading`, hero-code repo).

Both read the same labels above.
