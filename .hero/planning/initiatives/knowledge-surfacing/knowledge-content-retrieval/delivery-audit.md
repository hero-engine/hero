# Delivery audit — knowledge-content-retrieval (P1)

**Delivered via:** `/drive knowledge-surfacing` (guided autonomy), 2026-07-07
**Verdict:** SHIP
**Surface:** noteworthy

Layout-agnostic knowledge ingest into isolated `knowledge` + `fts_knowledge`
tables (ADR option B), reachable by `hero ask` and `hero search --knowledge`,
with the work/knowledge boundary structural (knowledge never touches the `specs`
table). Full suite green (`go test ./...` = ALL PASS).

## Acceptance criteria

- [✓] **Any layout retrievable** — `DiscoverKnowledge` walks `.hero/knowledge/**`
  and captures flat files; spec.md/three-file stay with `spec.Discover`. Verified:
  the original failing case `hero ask "what is the peer manifest publish boundary"`
  now answers from `decisions/peer-manifest-publish-boundary.md` (was "No knowledge
  found"). `TestDiscoverKnowledge_FlatDecision`.
- [✓] **Untyped captured** — untyped battlecard indexed with `kind` from subdir,
  title from H1. `TestDiscoverKnowledge_UntypedInvented`; live: `hero search
  --knowledge "RivalCorp pricing"` returns the untyped file.
- [✓] **Unknown subdir captured** — `kind` = raw subdir name, no allow-list.
- [✓] **No double-index** — raw/ skipped (own ingest); spec-owned sidecars
  skipped; dedup against `spec.Discover` by path. `TestDiscoverKnowledge_SkipsRawAndSpecOwned`.
- [✓] **Default search excludes knowledge** — `hero search "peer manifest publish
  boundary"` does NOT return the flat decision (0 hits in work search); knowledge
  lives in isolated tables so no `specs` query changed.
- [✓] **`hero search --knowledge`** — ranked FTS over the knowledge corpus with
  kind + path. Live-verified.
- [✓] **No leak into hero list/queue/verify** — `specs` table has 0 knowledge rows
  (`TestRefreshKnowledge_SelfHeals` asserts count==0); `hero list` shows no
  knowledge.
- [✓] **Self-healing** — `RefreshIfStale` gained `refreshKnowledge` (new/changed/
  removed parity). `TestRefreshKnowledge_SelfHeals` covers add + orphan-remove.
- [✓] **`--type <kind>`** — `SearchKnowledge` matches `kind OR type`, so
  `--type decision` (frontmatter type) and `--type battlecards` (subdir kind) both
  resolve. `TestSearchKnowledge_ByContentAndKind`.
- [✓] **Uniform across domains** — one code path; kind is data (subdir name), no
  domain branch.

## Deviations / notes

- **Ranking fix during delivery.** First cut merged knowledge into `hero ask`
  results by score, but graph/FTS/knowledge scores are on incomparable scales, so
  knowledge starved out below the limit in repos with a graph store (real-repo
  repro still failed while the temp workspace passed). Switched to round-robin
  interleave (`interleave()`), knowledge-first since ask is a knowledge question.
  Re-verified against the real repo.
- **Passage quality.** `hero ask` passage extraction (shared with specs) can pull
  frontmatter lines on very short files. Cosmetic; not a P1 AC. Candidate polish:
  strip frontmatter before passage extraction.
- **Sales pack repointed** — AGENTS.md + competitive-intel + deal-strategist now
  lead with `hero search --knowledge` / `hero ask`, keeping `ls` as a browse
  fallback and write paths intact. Closes the `sales-pack-reality-sync` AC#6
  path-lookup deviation.

## Files

- `internal/index/index.go` — knowledge + fts_knowledge tables (additive
  `CREATE TABLE`, no migration of existing data); `KnowledgeEntry`,
  `IndexKnowledge`, `RemoveKnowledge`, `KnowledgeModifiedAt`, `SearchKnowledge`;
  full-reindex clears+rebuilds knowledge.
- `internal/index/knowledge_discover.go` — `DiscoverKnowledge`, `parseKnowledgeFile`.
- `internal/index/refresh.go` — `refreshKnowledge` self-heal pass.
- `internal/retrieval/retrieval.go` — `Query.IncludeKnowledge` / `KnowledgeOnly`;
  `retrieveKnowledge`, `retrieveBase` extraction, `interleave`.
- `internal/cli/ask.go` — `IncludeKnowledge: true` + index self-heal.
- `internal/cli/search.go` — `--knowledge` flag.
- `internal/index/knowledge_discover_test.go` — 5 tests.
- `domains/sales/{AGENTS.md,agents/competitive-intel.md,agents/deal-strategist.md}`
  — repointed lookups.
