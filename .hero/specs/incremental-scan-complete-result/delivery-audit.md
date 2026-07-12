# Delivery audit — incremental-scan-complete-result

**Audited:** `git diff b5e7652 -- internal/codescan/{scancache.go,scanner.go,generate.go,codescan_test.go} internal/cli/{scan.go,graph_memory.go} internal/serve/mcp_tools.go` (scancache.go is new/untracked)
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria
- [✓] Partially-changed package `spec.md` gets COMPLETE symbols — `TestIncrementalScanEqualsFullScan` reads `internal-a/spec.md` and asserts `DBURL` (unchanged-file symbol context), `CacheURL` (new in changed a1.go), and `RegisterHealth` (unchanged a2.go) all present. This is the deepest defect; test passes.
- [✓] Incremental `Packages`/`ConfigVars`/`Endpoints` equal full scan (order-insensitive) — `TestIncrementalScanEqualsFullScan` compares `r2` against a fresh no-cache full scan via `canonPackages/canonConfigVars/canonEndpoints`. Helpers are genuinely order-insensitive (sort files, sort symbols, sort the outer slice) AND compare real content (symbol `name|kind|file|line`, configvar `name|source|file|line|required`, endpoint `method|path|handler|file|line|protocol`) — not counts. codescan_test.go:702-742.
- [✓] Deleted file drops + prunes emptied package — `TestIncrementalScanDeletedFileDropsAndPrunes` removes B's only file, asserts B absent from `r2.Packages`, `internal-b` dir pruned, and `r2.Packages` matches a fresh full scan. Mechanism verified: deleted file is absent from the walk, never carried forward (carry-forward only fires inside the walk callback), never written to the new cache.
- [✓] Missing/corrupt cache → full parse → complete result — `TestIncrementalScanMissingCacheFallback` covers (a) nil cache and (b) a corrupted `.scan-cache.json`; asserts `LoadScanCache` returns `(nil, err)` on corruption and both paths equal a full scan.
- [✓] Full scan unchanged + writes complete cache — `TestFullScanWritesCompleteCache` asserts `.scan-cache.json` exists, `Version == scanCacheVersion`, and every `r1.Files` entry round-trips into the loaded cache.
- [✓] Shipped prune behavior unchanged — `TestGenerateKnowledgeIncrementalPreservesUnchangedPackages` passes unchanged; keep-set logic in generate.go not touched (only a `BuildScanCache(result).Save` call added after `SaveChecksums`).

## Changes
- [✓] `scancache.go` (new) — `ScanCache` type + `LoadScanCache`/`Save`/`BuildScanCache`. `LoadScanCache` returns `nil,nil` for missing file and version mismatch (the "unusable" signal), `nil,err` for unmarshal failure. `BuildScanCache` reads `result.Files` and groups configvars/endpoints by file.
- [✓] `scanner.go` — signature now `Scan(prevChecksums, prevCache)`; carry-forward at scanner.go:210-224 appends cached `FileInfo`/`ConfigVars`/`Endpoints` and `return nil` only when the cache has the file's entry, else falls through to parse fresh. `result.Files = files` (scanner.go:254) holds the complete merged set (carried-forward appends at :219, fresh parse at :234).
- [✓] `generate.go` — `BuildScanCache(result).Save(codeDir)` added immediately after `SaveChecksums`; keep-set logic untouched.
- [✓] `scan.go` — loads `prevCache` via `LoadScanCache(codeDir)`, warns+nil on error, passes to `Scan`.
- [✓] `graph_memory.go:171` and `mcp_tools.go:2196` — both load the cache from the correct code dir and pass it. `graph_memory` uses `codeDir := cfg.CodeDir(projectRoot)`; `mcp_tools` reuses the same `codeDir` already load-bearing for `LoadChecksums`/enrichment save. 6 test call sites updated to `Scan(..., nil)` / `Scan(..., BuildScanCache(...))`.

## Open items
None. No PARTIAL / SKIPPED / BLOCKED rows.

## Audit notes
- **Order-insensitivity (spec's #1 risk) is genuine, not accidental.** The three canon helpers sort at every level and compare structural symbol content, so a pass cannot be explained by walk order coincidentally matching. Independently confirmed by reading the helper bodies.
- **Per-file guard is truly per-file.** A nil/corrupt/partial cache cannot silently drop a file: reuse requires `prevCache != nil` AND `prevCache.Files[relPath]` present; every miss falls through to a fresh parse. Corrupt path is exercised end-to-end.
- **`result.Files` is the complete merged set on incremental**, so `BuildScanCache` persists the full corpus (not just changed files) — the invariant the next incremental scan depends on.
- **Scope is clean.** Diff touches only the 7 named files (+ new scancache.go). `graph_ingest.go`, `types.go`, the prune keep-set, the checksum format, and all harness-facing surfaces are untouched (verified by empty diffs). Other files in the wider `b5e7652..HEAD` range (opsrunner, server.go, go.mod/go.sum) belong to unrelated intervening commits, not this delivery.
- Build clean (`go build ./cmd/hero`), all 5 target codescan tests PASS on independent re-run; full suite reported 86 packages ok / 0 FAIL.
