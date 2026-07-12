# Delivery audit — incremental-scan-prunes-unchanged-packages

**Audited:** `git diff -- internal/codescan/generate.go internal/codescan/codescan_test.go` (uncommitted working tree)
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria
- [✓] Incremental scan preserves every current package's `spec.md` (Goal) — keep-set now unions `slugify(filepath.Dir(relPath))` over all `result.Checksums` keys (`generate.go:47-49`); regression test asserts `internal-b/spec.md` survives an incremental scan that only changed package A (`codescan_test.go:663-665`).
- [✓] Still prunes genuinely-deleted packages (Goal) — deleted package absent from `result.Checksums` so its dir is still removed; test deletes B's file, re-scans incrementally, asserts `internal-b/` is pruned via `os.IsNotExist` (`codescan_test.go:679-691`).

## Changes
- [✓] Derive prune keep-set from complete `result.Checksums` instead of changed-only `writtenSlugs` (`generate.go:43-52`) — matches spec's suggested shape exactly; `filepath` already imported. `pruneStaleDirectories(codeDir, keep)` receives the union of `writtenSlugs` + `"index"` + `slugify(filepath.Dir(relPath))` over every checksum key.

## Slug-match verification (spec Risk #1)
- `writePackageSpec` writes to `slugify(pkg.Path)` (`generate.go:80`); `pkg.Path = filepath.Dir(f.Path)` where `f.Path == relPath` (`scanner.go:265,273`). Keep-set uses `slugify(filepath.Dir(relPath))` — **identical derivation**. No divergence.
- Root-file case confirmed: `filepath.Dir("main.go") == "."`; `slugify(".")` collapses `.`→`-` then the `s == "-"` guard maps to `"root"` (`generate.go:398,401-403`), matching `aggregatePackages`' `pkg.Path == "."` → `slugify(".") == "root"`.
- `result.Checksums` is recorded for every file *before* the incremental skip (`scanner.go:207` precedes the `prevChecksums` early-return at `210-214`), so the keep-set is complete on both full and incremental runs.

## Boundaries (no overreach)
- [✓] Scanner incremental-skip untouched — `scanner.go` not in diff.
- [✓] Partial index/ConfigVars/Endpoints (secondary defect #1) not touched — documented as follow-up per spec, concrete reason (needs full prior `Result` carry-forward).
- [✓] Graph write path untouched — already idempotent (`UpsertNode`), correctly left alone.
- [✓] `created:`/`slug:` frontmatter (wont-fix followup) not reopened — no frontmatter changes in diff.

## Test evidence (independently reproduced)
- Regression test is a real guard, not performative: reverting only the keep-set to `writtenSlugs` (pre-fix) makes it **FAIL** ("package B spec deleted by incremental scan (regression)"); the fixed `keep` set **PASSES**. All assertions read on-disk state via `os.Stat` / `os.IsNotExist`.
- `go test ./internal/codescan/` — ok.
- `go test ./...` — ALL_PASS (no failures across the suite).
- `go build ./cmd/hero` — BUILD_OK.

## Audit notes
- Diff is well-scoped to the two spec-named files. The working tree carries unrelated parallel changes (`internal/serve/opsrunner/*`, `internal/serve/server.go`, `go.mod`/`go.sum`) that are outside this delivery's audited diff and out of scope for this spec.
- Secondary defects were documented (not fixed) exactly as the spec's Boundaries direct — concrete deferral reasons, not soft skips.
