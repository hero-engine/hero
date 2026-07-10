# Delivery audit — flat-tripwire-trigger-parity

**Audited:** `git diff HEAD -- internal/` (uncommitted working tree)
**Verdict:** SHIP
**Surface:** clean

Cold second-pass audit. Verified against on-disk code and a fresh `go test` run — the
Completion Ledger was not trusted.

## Acceptance criteria

- [✓] AC-1 flat tripwire trigger-highlights in anchor — `knowledge_triggers` table + slug
  index (index.go:243-250), `matchKnowledgeTripwires` union appended to results
  (index.go:1300-1385), called from `FindTripwiresByTrigger` (index.go:1293-1297).
  `TestFlatTripwireHighlightsByTrigger` asserts the flat slug `tripwires/no-globalstate`
  is present in the matched set — PASS.
- [✓] AC-2 flat / spec.md parity — both scans call the same `triggerMatches(query, tokens,
  trigger)` helper (index.go:1225 for the specs branch, index.go:1350 inside
  `matchKnowledgeTripwires`); no divergent/duplicated match logic. Parity fixture drives a
  flat tripwire (knowledge table) and a spec.md-shaped tripwire (`no-globalstate-spec`,
  specs table) with identical `triggers:`; test asserts BOTH highlight for the same
  context, on both a single-token match ("globalstate") and a multi-word substring match
  ("mutable singleton") — PASS.
- [✓] AC-3 tripwire + MCP surfaces, no call-site edits — `git status` shows only
  `internal/index/index.go`, `internal/index/knowledge_discover.go`, and the test file
  modified. All four consumers are untouched: `internal/cli/anchor.go:51`,
  `internal/cli/tripwire.go:82`, `internal/serve/mcp_tools.go:890`,
  `internal/serve/mcp_tools.go:942` — each still calls `FindTripwiresByTrigger` unchanged.
  The seam was drawn correctly.
- [✓] AC-4 no triggers → lists, never highlights — `TestFlatTripwireNoTriggersNever
  Highlights` indexes a flat tripwire with no `triggers:`, asserts it appears in
  `FindAllTripwires` but is absent from `FindTripwiresByTrigger`. Non-vacuous: the same
  slug demonstrably lists via `FindAllTripwires`, proving it is indexed — it simply never
  highlights — PASS.
- [✓] AC-5 self-heal on edit/remove — delete-then-insert in `IndexKnowledge`
  (index.go:465-475), delete in `RemoveKnowledge` (index.go:493-495), `knowledge_triggers`
  added to the `Rebuild` clear-list (index.go:1899). `TestFlatTripwireTriggerSelfHeals`
  asserts actual `SELECT COUNT(*) FROM knowledge_triggers` row counts: 1 after index → 0
  after editing triggers away via the real ingest path → 0 after file removal +
  `RefreshIfStale`, and confirms the removed tripwire no longer highlights — PASS.
- [✓] Validation — `go test ./internal/index/...` green (0.318s); `go build` of the three
  consumer packages clean.

## Changes

- [✓] index.go: `knowledge_triggers` table + `idx_knowledge_triggers_slug` index in
  `migrate`; `KnowledgeEntry.Triggers` field — present, mirrors the `knowledge_scopes`
  pattern exactly (parallel table, no FK, keyed by knowledge slug).
- [✓] index.go: `IndexKnowledge` delete-then-insert of trigger rows; `RemoveKnowledge`
  cleanup; `Rebuild` clear-list includes `knowledge_triggers` — all present and mirror the
  `knowledge_scopes` maintenance.
- [✓] index.go: `FindTripwiresByTrigger` unions flat tripwires via new
  `matchKnowledgeTripwires`; shared `triggerMatches` helper — present. The specs-branch
  load and knowledge-branch load are separate table reads, results concatenated.
- [✓] knowledge_discover.go: `parseKnowledgeFile` captures `s.Triggers` into
  `KnowledgeEntry.Triggers` (knowledge_discover.go:103,110,134) — present.
- [✓] Tests: highlight/parity, negative, self-heal in knowledge_discover_test.go — all
  three present and non-vacuous.

## Gap analysis (auditor-initiated, beyond the ledger)

- **Double-counting (can a flat tripwire appear twice?)** — No. Flat tripwires carry
  `type: tripwire`, which is in `nonWorkFlatTypes` (spec.go:1236), so spec discovery never
  turns them into work specs → they never reach the `specs` / `tripwire_triggers` tables.
  The two branches read disjoint tables (`tripwire_triggers` vs `knowledge_triggers`) over
  disjoint slug namespaces (knowledge slugs are path-relative and carry a `tripwires/`
  prefix, e.g. `tripwires/no-globalstate`; spec slugs are bare). No dedup needed.
- **Slug collision (knowledge slug == spec slug)** — Not reachable in practice. The
  specs-branch load only pulls slugs present in `tripwire_triggers` (`matchedSlugs`), so a
  knowledge-only slug can never leak into the specs load even if the strings matched. The
  `tripwires/` path prefix on knowledge slugs makes a textual collision with a bare spec
  slug effectively impossible.
- **Graceful degradation `return nil, nil` on absent table** — The swallow is narrow: it
  fires ONLY on the initial `idx.db.Query` error (old schema without the table). `Scan`
  and `rows.Err()` errors both propagate normally. Since `migrate` always creates
  `knowledge_triggers` before any query runs, on a current schema the swallow is dead in
  practice, and it is consistent with the existing surface (2 of 4 consumers already
  discard the `FindTripwiresByTrigger` error with `matched, _ :=`). Residual: a transient
  DB error (locked/corrupt) here would silently drop flat-tripwire highlighting rather than
  surface. Low severity, matches the documented design intent ("Graceful on an absent
  knowledge_triggers table"). Not a blocker.

## Audit notes

- No performative rows. Every `DONE` in the ledger maps to real diff + a passing,
  behavior-asserting test.
- The self-heal test asserts real DB row counts (`SELECT COUNT(*)`), not just surface
  behavior — the strongest form of evidence for AC-5.
- Diff is well-scoped: only the three named files changed. No scope drift. The seam design
  constraint (no call-site edits) held.
- The kind filter `k.type = 'tripwire' OR k.kind = 'tripwires'` (index.go:1330) is
  belt-and-suspenders: `kind` is derived from the directory prefix (`rel[:i]`,
  knowledge_discover.go:95), so a flat tripwire under `tripwires/` matches even without an
  explicit `type: tripwire` line.
