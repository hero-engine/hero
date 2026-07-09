---
title: "Flat tripwires never trigger-highlight — wire knowledge triggers into FindTripwiresByTrigger"
slug: flat-tripwire-trigger-parity
type: enhancement
status: planning
priority: P3
size: small
domain: engineering
created: 2026-07-08
tags: [knowledge, tripwires, trigger-highlighting, anchor, parity, follow-on]
relations:
  - target: knowledge-surfacing
    kind: parent
  - target: knowledge-context-injection
    kind: depends-on
---

# Flat tripwires never trigger-highlight — wire knowledge triggers into FindTripwiresByTrigger

Follow-on to [[knowledge-context-injection]]. The drift/impact/anchor
surfacing follow-on (delivered) made flat tripwires in the isolated `knowledge`
table *appear* in `FindAllTripwires`, so they list wherever tripwires list. This
spec closes the remaining gap: flat tripwires never get **trigger-highlighted**
into the "relevant to your context" section, because the trigger matcher reads a
specs-only table.

## Goal

WHERE a flat `.hero/knowledge/tripwires/*.md` declares `triggers:` in its
frontmatter, WHEN a context string matches one of those triggers, THE SYSTEM
SHALL promote that tripwire into the trigger-matched output of every surface —
identically to `<slug>/spec.md`-shaped tripwires — with no per-surface patch.

## Kickoff

`FindTripwiresByTrigger` (`internal/index/index.go:1175`) matches context tokens
against the `tripwire_triggers` table, which `IndexSpec` populates from spec
`Triggers` and which flat knowledge never enters — so a flat tripwire's triggers
are invisible to it. It has **four** consumers, all equally blind:
`internal/cli/anchor.go:51`, `internal/cli/tripwire.go:82`, and
`internal/serve/mcp_tools.go:889` and `:941`.

The correct fix is a single DB seam, not four in-memory patches (an anchor-only
patch would leave `hero tripwire` and both MCP surfaces inconsistent — the exact
per-surface drift this initiative exists to eliminate). Mirror the existing
`knowledge_scopes` pattern:

- **Migration** (`internal/index/index.go` `migrate`): add
  `knowledge_triggers (id, knowledge_slug, trigger)` + a slug index, parallel to
  `knowledge_scopes`.
- **Capture** the triggers: `KnowledgeEntry` has no `Triggers` field today.
  Add one and populate it in `parseKnowledgeFile`
  (`internal/index/knowledge_discover.go`) from `spec.ParseFile`'s `s.Triggers`
  (already parsed, just not carried).
- **Ingest** (`IndexKnowledge`): delete-then-insert `knowledge_triggers` for the
  slug, exactly like the `knowledge_scopes` block already does; clean up in
  `RemoveKnowledge`.
- **Query** (`FindTripwiresByTrigger`): union matched flat-tripwire slugs from
  `knowledge_triggers` and build their `TripwireResult`s from the `knowledge`
  table + `spec.ParseFile(path)`, reusing the same section-parsing tail that
  `FindAllTripwires` now uses for flat tripwires. Match semantics must stay
  identical (case-insensitive token equality OR substring containment).

Because all four consumers call `FindTripwiresByTrigger`, they light up
together.

## Scope

**In scope**
- `knowledge_triggers` table + migration + slug index.
- `KnowledgeEntry.Triggers` field, captured in `parseKnowledgeFile`.
- `IndexKnowledge` / `RemoveKnowledge` maintain `knowledge_triggers` (add /
  modify / remove parity with `knowledge_scopes`).
- `FindTripwiresByTrigger` unions knowledge-table tripwire triggers and returns
  their `TripwireResult`s with parsed sections.

**Out of scope**
- Auto-inferring triggers — a flat tripwire highlights only if its author wrote
  a `triggers:` list.
- Any change to the four call sites — the seam fix must make them work
  unchanged. If a call site needs editing, the seam was drawn wrong.
- Free-form knowledge trigger matching — only `tripwires`-kind / `type:
  tripwire` entries participate.

## Acceptance Criteria

- WHERE a flat `.hero/knowledge/tripwires/*.md` declares `triggers: [foo, bar]`,
  WHEN `hero anchor "... foo ..."` runs, THE SYSTEM SHALL list that tripwire in
  the "Relevant to your context" section — today it appears only under "All
  active tripwires", never highlighted.
- THE SYSTEM SHALL highlight flat and `<slug>/spec.md` tripwires identically for
  the same context and trigger set (layout-agnostic parity).
- WHEN `hero tripwire <text>` and the MCP tripwire surfaces
  (`internal/serve/mcp_tools.go`) run with matching context, THE SYSTEM SHALL
  surface the flat tripwire too — no call-site edits required.
- IF a flat tripwire declares no `triggers:`, THEN THE SYSTEM SHALL still list it
  (via `FindAllTripwires`) but never trigger-highlight it.
- Removing or editing a flat tripwire's `triggers:` SHALL self-heal on the next
  `RefreshIfStale` (orphan cleanup parity with `knowledge_scopes`).

## Validation

- Repro-first: a flat tripwire with `triggers:` does **not** appear in
  `FindTripwiresByTrigger(ctx)` today; does after the change.
- Parity fixture: one flat and one `<slug>/spec.md` tripwire with identical
  `triggers:` — both highlight for the same context.
- Negative: a flat tripwire with no `triggers:` lists but never highlights.
- Self-heal: index a flat tripwire, remove its `triggers:`, `RefreshIfStale`,
  assert no stale `knowledge_triggers` rows remain.
- `go test ./...` green; existing `FindAllTripwires` / `FindTripwiresByTrigger`
  tests still pass.

## Notes

The surfacing half of this follow-on is already delivered: `FindAllTripwires`
(`internal/index/index.go`) unions flat tripwires from the `knowledge` table, and
`drift`/`impact` see flat code-scoped knowledge via `FindKnowledgeForFiles` at
their call sites. This spec is the last remaining slice of the
[[knowledge-context-injection]] "Out of scope" list — pure ranking/promotion
parity, no correctness gap. Deferred because no flat tripwire with a `triggers:`
list exists in the corpus yet; build it the day one is authored.
