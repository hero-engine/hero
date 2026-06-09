# Delivery Audit — unified-search

**Spec:** `.hero/planning/features/unified-search/spec.md`
**Audit date:** 2026-06-09
**Auditor:** cold delivery audit (re-audit after SC-4 targeted fix)
**Verdict:** SHIP

---

## Audited Success Criteria

| # | Criterion | Verdict | Evidence |
|---|-----------|---------|----------|
| SC-1 | `hero search` returns federation-pulled data with no flags | PASS | `retrieveViaGraph` and `retrieveViaNodeIndex` query all current nodes with no repo filter. Federation data lands in graph.db; default search returns it. |
| SC-2 | `hero search` returns results from on-disk sibling repos | PASS | `writeSiblingSubgraphs` (scan.go:464–508) iterates `cfg.Repos`, derives remote-origin key via `gitutil.RepoKey`, stamps each node with `Repo = siblingKey`, ingests via `spec.WriteGraph`. Sibling data in graph.db surfaces through SC-1 graph scan. `TestSmokeScan` passes. |
| SC-3 | `hero search --cross-repo` continues to work | PASS | `runSearch` (search.go:89) routes `--cross-repo` to `runSearchFTS` — unchanged legacy FTS5 path. No regression. |
| SC-4 | Search results include `repo` label for cross-repo results | PASS | `retrieval.Result.Repo string` field added (retrieval.go:85). `retrieveViaNodeIndex` SELECTs `ni.repo` (retrieval.go:239) and maps it to `Result.Repo` (retrieval.go:323). `retrieveViaGraph` SELECTs `repo` from nodes (retrieval.go:384) and maps it to `Result.Repo` (retrieval.go:465). `printGraphResults` (search.go:310–312) appends `[r.Repo]` when `r.Repo != "" && r.Repo != localRepo`. `TestPrintGraphResults_RepoLabelAppearsForCrossRepoResult` PASS — asserts label present for cross-repo, absent for local, absent for empty-repo. |
| SC-5 | `hero scan` ingests sibling repo specs with correct remote-origin key | PASS | `spec.WriteGraph` stamps `Repo: repoKey` on every upserted node; `writeSiblingSubgraphs` passes `gitutil.RepoKey(status.Path)`. `TestSmokeScan` passes. |
| Phase 2 FTS5 fallback | SKIPPED [signed-off] | Phase 3 (sibling ingest at scan time) makes the fallback unnecessary. Explicitly deferred in spec. Rationale is sound. |

---

## Detailed Findings

### SC-1 — Phase 1 delivery confirmed

`retrieveViaGraph` (retrieval.go:378) and `retrieveViaNodeIndex` (retrieval.go:223) both query across all current graph nodes with no repo filter. Federation data pulled into graph.db is visible on every default `hero search` run.

### SC-2 — Phase 3 delivery confirmed

`writeSiblingSubgraphs` (scan.go:473) correctly:
1. Skips if `cfg.Repos` is empty.
2. Resolves alias → path via `cfg.ResolveAllRepos(projectRoot)`.
3. Skips inaccessible repos.
4. Derives remote-origin key via `gitutil.RepoKey(status.Path)`.
5. Skips repos whose key matches the local repo (no double-ingest).
6. Calls `spec.WriteGraph(specs, siblingKey, ...)` — each node gets `Repo = siblingKey`.

The UNIQUE(type, key) collision caveat for same-slug specs across repos is documented in both spec and function comment; accepted as a known limitation for v1.

### SC-3 — Legacy path unchanged

`--cross-repo` flag routes through `runSearchFTS` (search.go:90). No regression.

### SC-4 — Repo label now implemented (PASS)

The fix adds the repo label end-to-end across all three layers:

**Struct** (`retrieval.go:85`):
```go
Repo string // remote-origin key; non-empty when result is from a sibling repo
```

**Node index path** (`retrieveViaNodeIndex`, retrieval.go:239): SQL now includes `ni.repo` in the SELECT and maps it via `Repo: c.repo` in the Result constructor (retrieval.go:323).

**Graph path** (`retrieveViaGraph`, retrieval.go:384): SQL now includes `repo` from nodes in the SELECT and maps it via `Repo: c.repo` in the Result constructor (retrieval.go:465).

**Printer** (`printGraphResults`, search.go:310–312):
```go
if r.Repo != "" && r.Repo != localRepo {
    line += fmt.Sprintf(" [%s]", r.Repo)
}
```

Three cases are correctly handled:
- Cross-repo result → label shown (`[alice/sibling-repo]`)
- Local result (Repo == localRepo) → no label
- Result with empty Repo → no label (guarded by `r.Repo != ""`)

**Test** (`TestPrintGraphResults_RepoLabelAppearsForCrossRepoResult`, search_test.go:353–376): exercises all three cases with direct assertions. PASS.

### SC-5 — Ingest key confirmed

Unchanged from previous audit. `spec.WriteGraph` stamps `Repo: repoKey` on every node; `writeSiblingSubgraphs` derives the key correctly.

---

## Test Coverage Assessment

| Test | Covers |
|------|--------|
| `TestPrintGraphResults_RepoLabelAppearsForCrossRepoResult` (cli/search_test.go:353) | SC-4: cross-repo label present, local label absent, empty-repo label absent. **New.** |
| `TestSmokeScan` (cli/smoke_test.go:31) | `hero scan` runs without error. |
| `TestSearchByQuery`, `TestSearchNoResults`, `TestSearchByFile`, `TestSearchListOnly`, `TestSearchListWithTypeFilter` | Search routing and result formatting. |
| `TestSearchJSONFTS5Path`, `TestSearchJSONListMode`, `TestSearchJSONFileMode`, `TestSearchJSONNoResults` | JSON output format. |
| `TestSpecWriteGraphInsertsNodesAndEdges`, `TestSpecWriteGraphIsIdempotent` | Graph write correctness. |

Full test run: `./internal/cli/...` and `./internal/retrieval/...` — all PASS.

---

## Phase 2 Skip

Phase 2 (FTS5 fallback for repos not in graph.db) is marked SKIPPED with sign-off in the Completion Ledger. Phase 3 makes the fallback unnecessary by ingesting sibling specs at scan time. The skip is legitimate and the rationale is correct.

---

## Files Audited

| File | Role |
|------|------|
| `internal/retrieval/retrieval.go:74–86` | `Result` struct — `Repo` field added |
| `internal/retrieval/retrieval.go:223–326` | `retrieveViaNodeIndex` — `ni.repo` selected, mapped to `Result.Repo` |
| `internal/retrieval/retrieval.go:378–467` | `retrieveViaGraph` — `repo` selected from nodes, mapped to `Result.Repo` |
| `internal/cli/search.go:286–328` | `printGraphResults` — `[repo]` label appended for cross-repo results |
| `internal/cli/search_test.go:353–376` | `TestPrintGraphResults_RepoLabelAppearsForCrossRepoResult` — new SC-4 test |
| `internal/cli/scan.go:464–508` | `writeSiblingSubgraphs` — Phase 3 ingest (unchanged) |
| `internal/spec/graph_ingest.go:32–74` | `WriteGraph` — stamps `Repo` on nodes (unchanged) |
