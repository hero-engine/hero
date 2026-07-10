---
title: "Flat tripwires never trigger-highlight — wire knowledge triggers into FindTripwiresByTrigger"
slug: flat-tripwire-trigger-parity
type: enhancement
status: completed
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
completed_at: 2026-07-10T06:05:05Z
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

**Pick up at: DELIVERED — pending `hero spec verify`.** The `knowledge_triggers`
DB seam is implemented in `internal/index/index.go` (migration + slug index,
`KnowledgeEntry.Triggers`, `IndexKnowledge`/`RemoveKnowledge`/`Rebuild`
maintenance, and a `matchKnowledgeTripwires` union inside `FindTripwiresByTrigger`
sharing a `triggerMatches` helper for identical semantics), with capture in
`internal/index/knowledge_discover.go`. All four consumers
(`anchor.go`, `tripwire.go`, `mcp_tools.go` ×2) call `FindTripwiresByTrigger`
unchanged — no call-site edits, as the seam design requires. Tests
(highlight/parity, negative, self-heal) are in
`internal/index/knowledge_discover_test.go`; `go test ./...` is green (86 pkgs).
See the Completion Ledger below. Next action: cold audit → `hero spec verify
flat-tripwire-trigger-parity` (last open child of the knowledge-surfacing
initiative; on completion the initiative auto-completes via `hero check
--reconcile`).

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

## Completion Ledger

| AC | Status | Note |
|----|--------|------|
| AC-1 (flat tripwire trigger-highlights in anchor) | DONE | `knowledge_triggers` seam + `matchKnowledgeTripwires` union in `FindTripwiresByTrigger`; `TestFlatTripwireHighlightsByTrigger` |
| AC-2 (flat/spec.md parity) | DONE | shared `triggerMatches` helper drives both scans; parity fixture (flat + spec.md, identical triggers) both highlight in `TestFlatTripwireHighlightsByTrigger` |
| AC-3 (tripwire + MCP surfaces, no call-site edits) | DONE | all four consumers call `FindTripwiresByTrigger` unchanged; `git diff` touches no call site |
| AC-4 (no triggers → lists, never highlights) | DONE | `TestFlatTripwireNoTriggersNeverHighlights` |
| AC-5 (self-heal on edit/remove) | DONE | delete-then-insert in `IndexKnowledge`, delete in `RemoveKnowledge`, `knowledge_triggers` added to `Rebuild` clear-list; `TestFlatTripwireTriggerSelfHeals` |
| Validation (`go test ./...` green) | DONE | 86 packages ok, 0 failed |

- [x] exercise-the-feature: parity fixture drives a flat and a spec.md-shaped tripwire through the real `RefreshIfStale` → `FindTripwiresByTrigger` path; both highlight, untriggered flat never does, self-heal verified against the DB rows.

## Changes

| # | Change | Status |
|---|--------|--------|
| 1 | `internal/index/index.go`: `knowledge_triggers` table + slug index (migration); `KnowledgeEntry.Triggers` field | DONE |
| 2 | `internal/index/index.go`: `IndexKnowledge`/`RemoveKnowledge` maintain `knowledge_triggers` (delete-then-insert + cleanup); `Rebuild` clear-list | DONE |
| 3 | `internal/index/index.go`: `FindTripwiresByTrigger` unions flat tripwires via new `matchKnowledgeTripwires`; shared `triggerMatches` helper for identical semantics | DONE |
| 4 | `internal/index/knowledge_discover.go`: capture `s.Triggers` in `parseKnowledgeFile` | DONE |
| 5 | Tests: highlight/parity, negative, self-heal (`internal/index/knowledge_discover_test.go`) | DONE |

## Notes

The surfacing half of this follow-on is already delivered: `FindAllTripwires`
(`internal/index/index.go`) unions flat tripwires from the `knowledge` table, and
`drift`/`impact` see flat code-scoped knowledge via `FindKnowledgeForFiles` at
their call sites. This spec is the last remaining slice of the
[[knowledge-context-injection]] "Out of scope" list — pure ranking/promotion
parity, no correctness gap. Deferred because no flat tripwire with a `triggers:`
list exists in the corpus yet; build it the day one is authored.
