---
title: "Incremental code-scan produces a complete Result — scan cache carries unchanged files forward"
slug: incremental-scan-complete-result
type: enhancement
status: completed
domain: engineering
size: medium
priority: medium
created: 2026-07-12
relates-to: [incremental-scan-prunes-unchanged-packages]
tags: [codescan, incremental, scan-cache, knowledge-writer, correctness]
completed_at: 2026-07-12T17:05:08Z
---

# Incremental code-scan produces a complete Result — scan cache carries unchanged files forward

## Kickoff

Makes incremental `hero scan` produce a COMPLETE `Result` — persist a per-file
`.scan-cache.json` next to `.checksums.json`, carry unchanged files' parse
results forward, and aggregate packages/configvars/endpoints over the full set
instead of only changed files.

**Status:** delivered — implemented, full suite green (86 pkgs), cold audit SHIP (clean). Pending `hero spec verify`.

**Pick up at:** nothing to implement. `internal/codescan/scancache.go`
(`ScanCache` + `LoadScanCache`/`Save`/`BuildScanCache`) is in; the `Scan`
skip point (`scanner.go:211-224`) now carries cached parse products forward
per-file; `GenerateKnowledge` saves the refreshed cache; `runCodeScan`,
`graph_memory.go`, and `mcp_tools.go` all load + pass it. Five tests cover
full-vs-incremental equivalence (order-insensitive), partial-package,
deleted-file, and missing/corrupt-cache fallback. If revisiting, the only
open adjacent item is graph pruning of genuinely-deleted packages (explicitly
out of scope — see Boundaries).

→ `.hero/planning/features/incremental-scan-complete-result/spec.md`

**Files:** `internal/codescan/scanner.go:205-244`, `internal/codescan/generate.go:14-59`, `internal/codescan/types.go:34-95`, `internal/cli/scan.go:258-268`, `internal/codescan/enrich.go:36-66`
**Skip:** directory-level re-parse (re-parses clean files in changed dirs, still needs a persisted cache) — rejected in favor of the file-level cache. Do NOT touch the shipped prune keep-set logic in `generate.go` — it stays correct once `result.Packages` is complete.

## Context

This is the documented follow-up to
`incremental-scan-prunes-unchanged-packages` (shipped, archived at
`.hero/specs/incremental-scan-prunes-unchanged-packages/spec.md`). That fix
stopped incremental `hero scan` from **deleting** unchanged package directories
by deriving the prune keep-set from the always-complete `result.Checksums`. It
explicitly deferred a deeper problem (Secondary Defect #1): on an incremental
scan the `Result` **itself** is partial, because the scanner parses only changed
files.

`internal/codescan/scanner.go` records `result.Checksums[relPath]` for every
parseable file (`scanner.go:205-207`) but then `return nil`s for files whose
checksum matches `prevChecksums` (`scanner.go:210-214`), so only changed files
are appended to `files` (`scanner.go:221`), and `ConfigVars`/`Endpoints` are
extracted only for those changed files (`scanner.go:224-233`).
`result.Packages = s.aggregatePackages(files)` (`scanner.go:244`) therefore runs
over the changed-only slice. Everything downstream inherits that partiality.

Three consequences, one root cause (partial `Result` on incremental):

1. **Partial index sections.** `writeCodeIndex` (`generate.go:178-315`)
   regenerates `index/spec.md`'s package table, `## Environment Variables`
   section, and `## API Endpoints` section from `result.Packages` /
   `result.ConfigVars` / `result.Endpoints`. On incremental these list only the
   changed packages/vars/endpoints.
2. **Partial `ConfigVars` / `Endpoints`.** Same root — extracted only for
   changed files.
3. **DEEPEST — active corruption of partially-changed packages.** A package
   where only *some* files changed is `aggregatePackages`'d from just its
   changed files, so `writePackageSpec` (`generate.go:79-176`) REWRITES that
   package's `spec.md` with a SUBSET of its symbols/files/line-counts. The
   shipped prune fix protects *fully-unchanged* packages from deletion; it does
   nothing for a *partially-changed* package's content — its `spec.md` is
   overwritten with a lie.

**Graph path.** `internal/codescan/graph_ingest.go` `WriteGraph`
(`graph_ingest.go:33-205`) upserts `result.Packages` via `store.UpsertNode`
(additive, no prune). On incremental it currently re-asserts only changed
packages; a partially-changed package is upserted with subset symbols and
`hashPackage` (`graph_ingest.go:245-271`) computed over that subset — so the
graph carries a truncated Package node until a full scan. Making
`result.Packages` complete fixes the graph write for free (verified: no change
needed in `graph_ingest.go`). Deleted-package pruning *from the graph* remains a
separate pre-existing concern and stays out of scope (see Boundaries).

**Mission fit.** `hero_code`, retrieval, and the DSKG graph all read this
corpus. A silently-partial incremental result means every second-or-later scan
degrades what the next agent session sees. Completing the `Result` is directly
"the next session starts as smart as the last one ended."

## Goal

An incremental `hero scan` produces `result.Packages`, `result.ConfigVars`, and
`result.Endpoints` that are equal (by content) to what a full scan of the same
tree would produce — so every downstream surface (per-package `spec.md`, the
`index/spec.md` table + env-var + endpoint sections, the dep-graph, and the
graph ingest) is complete and correct after any scan. Incremental stays cheap:
unchanged files are never re-parsed; their prior parse results are carried
forward from a persisted cache. A missing, unreadable, or legacy cache degrades
gracefully to a full re-parse (never a crash, never a result worse than today).
Full-scan behavior and the shipped prune keep-set behavior are unchanged.

## Approach

### Persist a scan cache alongside the checksums

Add a per-file **scan cache** artifact `.scan-cache.json` in the (gitignored)
code knowledge dir, next to `.checksums.json`, `.enrichments.json`, and `.l.json`.
The cache holds, attributed by file path, the intermediate parse products the
incremental skip currently throws away: the per-file `FileInfo`, plus the
`ConfigVar`s and `Endpoint`s extracted from each file.

This is a **separate artifact, not a change to `Result`'s wire form.**
`Result.Files` is `json:"-"` (intermediate; `types.go:93`). `FileInfo`,
`ConfigVar`, and `Endpoint` already carry their file path and already have JSON
tags (`types.go:35-42`, `types.go:75-82`, `types.go:115-122`), so the cache
serializes cleanly with no type changes.

Keying by relative file path makes the three merge operations trivial:
- **unchanged file** → checksum matches AND the cache has an entry for it → reuse
  the cached `FileInfo`/`ConfigVars`/`Endpoints`;
- **changed file** → parse fresh (as today);
- **deleted file** → absent from the current walk → never added to the merged set
  and never written to the fresh cache → dropped.

### Keep `Scan` pure; caller does the disk IO (mirror the checksum split)

The codebase already splits scan IO cleanly: `Scanner.Scan` is pure over
in-memory `prevChecksums` in / `result.Checksums` out; `runCodeScan` does
`LoadChecksums` (`scan.go:259`) and `GenerateKnowledge` does `SaveChecksums`
(`generate.go:55`). Follow that exact seam for the cache:

- `runCodeScan` (`scan.go`) loads the prior cache and passes it into `Scan`
  alongside `prevChecksums`.
- `Scan` performs a **pure** merge (no disk IO): for each unchanged file, pull
  from the passed-in cache; otherwise parse. Produces the complete `files` /
  `ConfigVars` / `Endpoints`.
- `GenerateKnowledge` (`generate.go`) builds the refreshed cache from the now-
  complete `result` and saves it, right where it already calls `SaveChecksums`.

This means `Scan`'s signature grows one parameter. That is the honest, explicit
Go move (per `go-stack`: explicit over clever); the alternative — giving the
Scanner a `codeDir` and doing disk IO inside `Scan` — would break the existing
pure-`Scan` separation and is rejected.

### Graceful degradation is a per-file guard, not a mode flag

The reuse decision is made **per file**: reuse the cache for an unchanged file
only if the cache actually has an entry for that file. Consequences fall out for
free and cover every failure mode:

- **No cache (missing/corrupt/legacy):** caller passes `nil`. Every unchanged
  file misses the per-file guard and is parsed fresh → equivalent to a full scan
  → complete result. Never worse than today.
- **Partial cache (some entries missing):** only the missing files are re-parsed;
  the rest carry forward. Still complete.

A `Version` field on the cache gives forward/backward tolerance: a version
mismatch is treated as "no usable cache" (→ full re-parse), same as corrupt.

### Full scan is unchanged

When `prevChecksums == nil` (full scan), no file is ever skipped, the passed
cache is irrelevant, and behavior is byte-for-byte as today — plus a fresh
complete cache is written for the next run.

## Changes

1. **New file `internal/codescan/scancache.go`** — the cache type and its IO,
   modeled on `enrich.go:36-66` (`LoadEnrichmentCache`/`Save`).
   - Define the cache keyed by relative file path:
     ```go
     const scanCacheVersion = 1

     // ScanCache carries per-file parse products forward across incremental
     // scans so unchanged files need not be re-parsed while the merged Result
     // stays complete. Keyed by relative file path.
     type ScanCache struct {
         Version    int                     `json:"version"`
         Files      map[string]FileInfo     `json:"files"`
         ConfigVars map[string][]ConfigVar  `json:"config_vars,omitempty"`
         Endpoints  map[string][]Endpoint   `json:"endpoints,omitempty"`
     }
     ```
   - `LoadScanCache(codeDir string) (*ScanCache, error)`:
     - path `filepath.Join(codeDir, ".scan-cache.json")`.
     - missing file (`os.IsNotExist`) → return `nil, nil` (no cache; caller will
       full-parse). Do NOT return an empty-but-non-nil cache for the missing
       case — `nil` is the "unusable" signal the merge guard keys on.
     - unmarshal error → return `nil, err` (caller warns and proceeds with `nil`).
     - `cache.Version != scanCacheVersion` → return `nil, nil` (legacy/unusable →
       full-parse; not an error).
     - success → ensure the three maps are non-nil, return `&cache, nil`.
   - `(*ScanCache) Save(codeDir string) error`: `MkdirAll` then
     `json.MarshalIndent` to `.scan-cache.json`, `0o644` — identical shape to
     `EnrichmentCache.Save` and `SaveChecksums`.
   - `BuildScanCache(result *Result) *ScanCache`: construct a fresh cache from
     the **complete** merged `result`:
     - `Version: scanCacheVersion`.
     - `Files`: one entry per `result.Files` keyed by `FileInfo.Path`.
     - `ConfigVars`: group `result.ConfigVars` by `ConfigVar.File`.
     - `Endpoints`: group `result.Endpoints` by `Endpoint.File`.
     - Note: `result.Endpoints` includes endpoint-only files (`.proto`/`.graphql`,
       extracted at `scanner.go:177-190`). Grouping them into the cache is
       harmless — those files are never in `result.Checksums`, so the merge reuse
       guard (which is gated on `prevChecksums` membership) never fires for them;
       they are re-extracted every run. No double-count.

2. **`internal/codescan/scanner.go` — merge cached parse products at the skip
   point.**
   - Change `Scan`'s signature to accept the prior cache:
     ```go
     func (s *Scanner) Scan(prevChecksums map[string]string, prevCache *ScanCache) (*Result, error)
     ```
   - Replace the current unchanged-file early return (`scanner.go:210-214`):
     ```go
     // Skip unchanged files if we have previous checksums
     if prevChecksums != nil {
         if prev, ok := prevChecksums[relPath]; ok && prev == sum {
             return nil
         }
     }
     ```
     with a carry-forward that keeps the merged set complete:
     ```go
     // Unchanged file: carry its prior parse result forward from the cache so
     // the merged Result stays complete without re-parsing. If the cache lacks
     // an entry (missing/partial/legacy cache), fall through and parse fresh —
     // never drop the file.
     if prevChecksums != nil {
         if prev, ok := prevChecksums[relPath]; ok && prev == sum {
             if prevCache != nil {
                 if fi, ok := prevCache.Files[relPath]; ok {
                     files = append(files, fi)
                     result.ConfigVars = append(result.ConfigVars, prevCache.ConfigVars[relPath]...)
                     result.Endpoints = append(result.Endpoints, prevCache.Endpoints[relPath]...)
                     return nil
                 }
             }
             // no cached entry — fall through to parse fresh
         }
     }
     ```
   - Everything from `scanner.go:216` down (`ParseFile`, `aggregatePackages`,
     `buildDepGraph`, `computeImportedBy`, `mapModules`) is unchanged — it now
     runs over the **complete** merged `files`, which is the whole point. No edits
     needed below the skip point.
   - `result.Files = files` (`scanner.go:241`) now holds the complete set on
     incremental too — this is what `BuildScanCache` reads.

3. **`internal/codescan/generate.go` — save the refreshed cache in
   `GenerateKnowledge`.**
   - After the existing `SaveChecksums(codeDir, result.Checksums)` call
     (`generate.go:55`), save the fresh cache built from the complete result:
     ```go
     if err := BuildScanCache(result).Save(codeDir); err != nil {
         return fmt.Errorf("saving scan cache: %w", err)
     }
     ```
   - Do NOT touch the prune keep-set logic (`generate.go:43-52`). It already
     derives from `result.Checksums` and stays correct; with a complete
     `result.Packages` the `writtenSlugs` half now also covers every current
     package, but the checksum-derived half is still the load-bearing guard and
     must remain.

4. **`internal/cli/scan.go` — load the prior cache and pass it to `Scan`.**
   - Alongside the existing `LoadChecksums` (`scan.go:259`), load the cache:
     ```go
     prevCache, err := codescan.LoadScanCache(codeDir)
     if err != nil {
         fmt.Fprintf(os.Stderr, "Warning: could not load scan cache, re-parsing all files: %v\n", err)
         prevCache = nil
     }
     ```
   - Update the `Scan` call (`scan.go:268`) to
     `scanner.Scan(prevChecksums, prevCache)`.

5. **Update all other `Scan` callers to the new signature.**
   - Grep for `.Scan(` across the repo (`internal/codescan` tests, any other
     CLI/serve call sites) and pass the second argument. Full-scan call sites
     pass `nil` for `prevCache`. This is the only cross-file blast radius of the
     signature change; it is mechanical but must be complete for the build to
     pass.

## Acceptance Criteria

- WHEN an incremental scan runs on a package where only some files changed, THE
  SYSTEM SHALL write that package's `spec.md` with the COMPLETE set of its
  symbols and files, not just the changed files'. *(regression test required)*
- WHEN an incremental scan runs, THE SYSTEM SHALL produce `result.Packages`,
  `result.ConfigVars`, and `result.Endpoints` equal by content to what a full
  scan of the same tree would produce. *(full-vs-incremental equivalence test on
  a fixture containing an unchanged file, a changed file, and a partially-changed
  package)*
- WHEN a file is deleted between scans, THE SYSTEM SHALL drop its symbols,
  configvars, and endpoints from the incremental result and prune a package that
  is thereby emptied. *(test)*
- IF the scan cache is missing or unreadable, THEN THE SYSTEM SHALL parse all
  files and still produce a complete result. *(test: run incremental with the
  cache file absent and with it corrupted; assert the result equals a full scan)*
- WHERE `prevChecksums` is nil (full scan) THE SYSTEM SHALL behave exactly as
  today AND write a fresh complete `.scan-cache.json`. *(test)*
- THE SYSTEM SHALL leave full-scan behavior and the shipped prune keep-set
  behavior unchanged. *(existing tests, including
  `TestGenerateKnowledgeIncrementalPreservesUnchangedPackages`, stay green)*

## Boundaries

- Does NOT change the incremental checksum mechanism, the checksum file format,
  or `Load/SaveChecksums`.
- Does NOT change the shipped prune keep-set logic (`generate.go:43-52`) — that
  fix stays as-is and must keep passing its regression test.
- Does NOT add deleted-package pruning to the graph write path
  (`graph_ingest.go`). The graph is additive by design; complete
  `result.Packages` fixes *partial* package upserts, but removing stale Package
  nodes for genuinely-deleted packages from the graph is a separate, pre-existing
  concern out of scope here (as already noted in the shipped spec).
- Does NOT touch any harness-facing surface (instruction files, install targets,
  slash commands, agents, skills). Confirmed against the
  `harness-changes-cover-all-targets` tripwire — this is purely internal
  `internal/codescan` + one CLI call site.
- Does NOT alter the deep-enrichment cache (`.enrichments.json` / `.l.json`) or
  its flow; the new cache is a sibling artifact with its own file.
- Does NOT run `hero scan` against the real working tree (it clobbers
  `.hero/knowledge/`). Validation is `go test` + `go build` only.

## Risks

- **Signature-change blast radius.** Changing `Scan(prevChecksums)` to
  `Scan(prevChecksums, prevCache)` breaks every caller until updated (Change #5).
  Mitigation: grep `.Scan(` before building; the compiler catches misses. Full-
  scan callers pass `nil`.
- **Stale/corrupt cache producing a subset.** The whole safety argument rests on
  the per-file guard: an unchanged file is only carried forward if the cache has
  its entry. If that guard were weakened to a mode flag (cache present → skip
  all unchanged), a partial/corrupt cache would silently drop files. Keep the
  guard per-file; the corrupt-cache test locks this in.
- **`FileInfo` round-trip fidelity.** The cached `FileInfo` must deserialize to
  exactly what a fresh parse produces, or the equivalence test fails. All
  `FileInfo`/`Symbol` fields that feed `aggregatePackages` and `writePackageSpec`
  carry JSON tags (`types.go:22-42`); `Symbol.File` is set during aggregation
  (`scanner.go:288-294`), not parse, so it is reconstructed identically on the
  carried-forward path because aggregation re-runs over the merged files. Verify
  the equivalence test compares symbol `File`/`Line` too.
- **ConfigVar/Endpoint ordering.** Full and incremental scans may append these in
  different file-walk order, so equivalence assertions must be order-insensitive
  (sort or compare as multisets), not slice-equal. `writeCodeIndex` already sorts
  env-var names (`generate.go:254`), so on-disk output is order-stable; the test,
  not the code, must account for in-memory ordering.
- **Endpoint-only files in the cache.** `.proto`/`.graphql` endpoints are stored
  in the cache but never reused (not in `result.Checksums`); they are re-extracted
  every run. Confirm no double-count by asserting endpoint counts match full scan.

## Validation

- `go build ./cmd/hero` — compiles.
- `go test ./...` — all pass, including the new tests and the existing codescan
  suite (`TestScanner`, `TestGenerateKnowledge`,
  `TestGenerateKnowledgeIncrementalPreservesUnchangedPackages`).
- Do NOT run `hero scan` against the real working tree.

### Tests to add (`internal/codescan/codescan_test.go`)

Model on the existing `TestScanner` incremental re-scan and `TestGenerateKnowledge`.

1. **Full-vs-incremental equivalence (the core test).** Temp project with three
   shapes: package A with two files where only one will change (partially-changed
   package), package B untouched, package C's single file changed. Include at
   least one `os.Getenv` call (→ ConfigVar) and one route registration
   (→ Endpoint) split across a changed and an unchanged file.
   - Full scan: `r1, _ := scanner.Scan(nil, nil)`; `GenerateKnowledge(r1, dir)`;
     capture `r1.Packages`, `r1.ConfigVars`, `r1.Endpoints`.
   - Mutate A's one file and C's file on disk.
   - Incremental: build the prior cache from r1 (`BuildScanCache(r1)`), then
     `r2, _ := scanner.Scan(r1.Checksums, BuildScanCache(r1))`.
   - Assert `r2.Packages` ≡ a full re-scan's packages (order-insensitive; compare
     path, file list, symbol names+lines+files, line counts). Assert
     `r2.ConfigVars` and `r2.Endpoints` ≡ full re-scan (as multisets).
   - Assert package A's on-disk `spec.md` lists BOTH files' symbols (the deepest
     defect — fails today because A is aggregated from its one changed file).
2. **Deleted file drops from result and prunes emptied package.** From r1's tree,
   `os.Remove` B's only file, incremental scan, `GenerateKnowledge`; assert B is
   absent from `r2.Packages` and B's dir is pruned; assert a deleted symbol's
   ConfigVar/Endpoint is gone.
3. **Missing-cache fallback.** Incremental scan passing `r1.Checksums` but
   `nil` cache (and separately, a cache loaded from a corrupted `.scan-cache.json`
   → `LoadScanCache` returns `nil`); assert the result equals a full scan.
4. **Full scan writes a complete cache.** After a full `Scan(nil,nil)` +
   `GenerateKnowledge`, assert `.scan-cache.json` exists, `Version ==
   scanCacheVersion`, and `LoadScanCache` round-trips every current file.
5. **Prune regression stays green.** Confirm
   `TestGenerateKnowledgeIncrementalPreservesUnchangedPackages` still passes
   unchanged.

## Completion Ledger

### Acceptance Criteria

| # | Criterion | Status | Evidence |
|---|-----------|--------|----------|
| 1 | Partially-changed package's `spec.md` gets COMPLETE symbols | DONE | `TestIncrementalScanEqualsFullScan` asserts package A's on-disk `internal-a/spec.md` lists both the changed and unchanged files' symbols. Carry-forward at `scanner.go:211-224`. |
| 2 | Incremental `Packages`/`ConfigVars`/`Endpoints` ≡ full scan (content, order-insensitive) | DONE | `TestIncrementalScanEqualsFullScan` compares via `canonPackages`/`canonConfigVars`/`canonEndpoints` (sort-then-compare on name+kind+file+line, not counts). |
| 3 | Deleted file drops from result + prunes emptied package | DONE | `TestIncrementalScanDeletedFileDropsAndPrunes`. |
| 4 | Missing/unreadable cache → full parse, complete result | DONE | `TestIncrementalScanMissingCacheFallback` (nil cache + corrupt `.scan-cache.json`). |
| 5 | Full scan unchanged + writes complete versioned cache | DONE | `TestFullScanWritesCompleteCache`. |
| 6 | Shipped prune keep-set behavior unchanged | DONE | `TestGenerateKnowledgeIncrementalPreservesUnchangedPackages`, `TestScanner`, `TestGenerateKnowledge` all green; prune logic untouched. |

### Changes

| # | Item | Status | Evidence |
|---|------|--------|----------|
| 1 | New `internal/codescan/scancache.go` (`ScanCache` + `LoadScanCache`/`Save`/`BuildScanCache`) | DONE | `nil` = unusable→full-parse for missing/version-mismatch; error only on corrupt unmarshal; builds from complete merged result. |
| 2 | `scanner.go` — new signature + per-file carry-forward at skip point | DONE | `Scan(prevChecksums, prevCache *ScanCache)`; carry-forward `scanner.go:211-224`; `result.Files` now complete on incremental. |
| 3 | `generate.go` — save refreshed cache after `SaveChecksums` | DONE | `BuildScanCache(result).Save(codeDir)`; prune keep-set untouched. |
| 4 | `internal/cli/scan.go` — load + pass cache | DONE | warn-and-nil fallback; `Scan(prevChecksums, prevCache)`. |
| 5 | Update all other callers | DONE | `graph_memory.go` (`cfg.CodeDir`), `mcp_tools.go` (existing `codeDir`) both load+pass; 6 test call sites updated. |

**Validation:** `go build ./cmd/hero` → BUILD_OK · `go test ./...` → 86 packages ok, 0 FAIL · cold audit `delivery-audit.md` → SHIP (clean), high confidence.

**Exercise-the-feature:** driven end-to-end via the test harness (`Scan` → `GenerateKnowledge` → on-disk `spec.md` + `.scan-cache.json` inspection). Running `hero scan` on the real tree is forbidden by the spec (clobbers `.hero/knowledge/`), so the test harness is the mandated exercise surface.
