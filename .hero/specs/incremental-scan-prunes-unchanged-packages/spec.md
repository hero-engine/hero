---
title: "Incremental code-scan prunes unchanged package dirs — data loss in .hero/knowledge/code/"
slug: incremental-scan-prunes-unchanged-packages
type: bug
status: completed
severity: medium
priority: medium
domain: engineering
size: small
created: 2026-07-12
root_cause_class: code
relates-to: [codescan-created-slug-frontmatter-followup]
tags: [codescan, incremental, knowledge-writer, data-loss]
completed_at: 2026-07-12T16:37:13Z
---

# Incremental code-scan prunes unchanged package dirs — data loss in `.hero/knowledge/code/`

## Kickoff

Incremental `hero scan` wipes every unchanged package's `spec.md` under
`.hero/knowledge/code/`, leaving a misleadingly-partial corpus.

**Status:** delivered — fix landed and verified; cold audit SHIP (clean). Pending `hero spec verify` to flip to completed.

**Pick up at:** nothing to implement — `GenerateKnowledge`
(`internal/codescan/generate.go`) now builds the prune keep-set from
`result.Checksums` (union of `writtenSlugs` + `slugify(filepath.Dir(relPath))`
over every checksum key), and `TestGenerateKnowledgeIncrementalPreservesUnchangedPackages`
in `internal/codescan/codescan_test.go` guards it (fails on the old
`writtenSlugs` keep-set, passes on the fix). `go test ./...` green (86 pkgs),
`go build ./cmd/hero` OK. If revisiting, the open follow-up is secondary
defect #1 (partial index/ConfigVars/Endpoints on incremental scans), which
needs the full prior `Result` carried forward.

→ `.hero/planning/bugs/incremental-scan-prunes-unchanged-packages/spec.md`

**Files:** `internal/codescan/generate.go:14`, `internal/codescan/generate.go:49`, `internal/codescan/scanner.go:205`, `internal/codescan/codescan_test.go:551`
**Skip:** "prune only on full scans" — leaves genuinely-deleted packages lingering; the Checksums-derived keep-set is strictly more correct for the same code cost. Do NOT reopen the `codescan-created-slug-frontmatter-followup` (wont-fix) frontmatter concern.

## Issue

No tracker ID. Found on this repo: after an incremental `hero scan`,
`ls .hero/knowledge/code/` shows only `index/`, `internal-projection/`,
`internal-serve-opsrunner/` (plus `.checksums.json`) despite ~70 Go packages.
The prior incremental scan deleted every package directory whose files did not
change since the previous scan.

Environment: any repo where a second-or-later `hero scan` runs (incremental is
the default once `.checksums.json` exists). Reproduces on the hero repo itself.

## Investigation

### Categorization
| Attribute | Assessment |
|-----------|------------|
| **Criticality** | medium — `.hero/knowledge/code/` is gitignored and regenerable via a full scan, but the incremental default silently produces a partial corpus that downstream readers treat as authoritative. |
| **Ease of Fix** | easy — one-function change to the prune keep-set + one regression test. |
| **Caused by our codebase?** | Yes — logic error in `internal/codescan/generate.go`. |
| **Needs more research?** | No — root cause traced and confirmed against source. |

### Root cause
The knowledge writer prunes package directories using a keep-set built from the
**changed-only** packages, but on an incremental scan `result.Packages` contains
only packages whose files changed. Every unchanged package is therefore absent
from the keep-set and gets `os.RemoveAll`'d.

Confirmed source trace:

1. `internal/codescan/scanner.go:206-207` — for EVERY parseable file, the scanner
   records `result.Checksums[relPath] = sum` **before** any incremental skip. So
   `result.Checksums` is always the COMPLETE current file set.
2. `internal/codescan/scanner.go:210-214` — in incremental mode
   (`prevChecksums != nil`), an unchanged file (`prev == sum`) `return nil`s early
   and is NOT appended to the `files` slice.
3. `internal/codescan/scanner.go:244` — `result.Packages = s.aggregatePackages(files)`
   is built from the changed-only `files` slice → partial on incremental runs.
4. `internal/codescan/generate.go:23-27` — `GenerateKnowledge` writes only
   `result.Packages` and records their slugs (`slugify(pkg.Path)`) into
   `writtenSlugs`.
5. `internal/codescan/generate.go:34` — adds `"index"` to `writtenSlugs`.
6. `internal/codescan/generate.go:37` → `pruneStaleDirectories(codeDir, writtenSlugs)`.
7. `internal/codescan/generate.go:49-62` — `pruneStaleDirectories` `os.RemoveAll`s
   every subdirectory of `.hero/knowledge/code/` whose name is not in the keep-set.
   On incremental, that's every unchanged package. **Data loss.**
8. `internal/cli/scan.go:258-268` — `runCodeScan` loads `prevChecksums` via
   `LoadChecksums(codeDir)` on every run and passes them to `scanner.Scan`, so
   incremental mode is the default on any second+ run → deletion fires by default.

`.checksums.json` itself is NOT lost: it is written from the complete
`result.Checksums` (`generate.go:40` → `SaveChecksums`, `scanner.go:417-425`).

### Severity
Medium. Blast radius is confined to the markdown corpus under
`.hero/knowledge/code/`, which is gitignored and fully regenerable by a
non-incremental scan. No source or graph data is lost. But the failure is
silent and fires on the default path, so the on-disk corpus is routinely
incomplete and downstream readers (agents, `hero_code`, retrieval) treat the
partial corpus as authoritative. Root cause is CODE (logic error — wrong data
source for the prune keep-set).

## Code Flow (End to End)

1. `internal/cli/scan.go:259` — `LoadChecksums(codeDir)` loads prior `.checksums.json`.
2. `internal/cli/scan.go:268` — `scanner.Scan(prevChecksums)` runs incremental.
3. `internal/codescan/scanner.go:207` — every file's checksum recorded (complete).
4. `internal/codescan/scanner.go:210-214` — unchanged files skipped from `files`.
5. `internal/codescan/scanner.go:244` — `result.Packages` built from changed-only `files`.
6. `internal/codescan/generate.go:23-28` — writes changed packages, fills `writtenSlugs`.
7. `internal/codescan/generate.go:37,49-62` — prunes every dir not in `writtenSlugs` → deletes unchanged packages.
8. `internal/codescan/generate.go:40` — `.checksums.json` rewritten (complete, so it survives).

## Key Files

### Scanner
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/codescan/scanner.go` | 205-214 | Checksums recorded for all files; unchanged files skipped from `files`. |
| `internal/codescan/scanner.go` | 241-244 | `result.Packages` built from changed-only slice. |
| `internal/codescan/scanner.go` | 400-425 | `LoadChecksums` / `SaveChecksums`. |

### Knowledge writer (defect site)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/codescan/generate.go` | 14-45 | `GenerateKnowledge` — builds keep-set from changed packages only. |
| `internal/codescan/generate.go` | 49-62 | `pruneStaleDirectories` — `os.RemoveAll`s dirs not in keep-set. |
| `internal/codescan/generate.go` | 64-66 | `writePackageSpec` slug derivation `slugify(pkg.Path)` — keep-set must match. |
| `internal/codescan/generate.go` | 380-390 | `slugify` — including the root case (`"."` → `"root"`). |

### CLI entry
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/scan.go` | 250-274 | `runCodeScan` — loads prevChecksums, drives incremental by default. |

## Goal

An incremental `hero scan` preserves every current package's `spec.md` under
`.hero/knowledge/code/`, while still pruning directories for packages that were
genuinely deleted (their files no longer exist). The on-disk corpus after an
incremental scan matches the corpus a full scan would produce for the set of
current package directories.

## Changes

1. Derive `pruneStaleDirectories`'s keep-set from the COMPLETE current file set
   (`result.Checksums`), not the changed-only `writtenSlugs`.
   - In `GenerateKnowledge` (`internal/codescan/generate.go:14`), after writing
     packages and the index, build the keep-set as the union of:
     - `"index"`,
     - the written package slugs (`slugify(pkg.Path)` for each `result.Packages`), and
     - `slugify(filepath.Dir(relPath))` for **every** `relPath` key in
       `result.Checksums`.
   - Pass that union to `pruneStaleDirectories`.
   - Rationale: `result.Checksums` is always complete (full AND incremental), so
     the keep-set covers all current packages regardless of scan mode. Packages
     whose files were all deleted are absent from `result.Checksums`, so they are
     still pruned — on both full and incremental scans.
   - Verify the slug derivation matches `writePackageSpec`: it uses
     `slugify(pkg.Path)` where `pkg.Path == filepath.Dir(file)`
     (`scanner.go:265` sets `dir = filepath.Dir(f.Path)`), so
     `slugify(filepath.Dir(relPath))` reproduces the exact written slug —
     including the root case: `filepath.Dir("main.go") == "."`,
     `slugify(".") == "root"`, matching `aggregatePackages`' `pkg.Path == "."`
     → `slugify(".") == "root"`.

   Suggested shape (illustrative — engineer to place cleanly):

   **Before** (`internal/codescan/generate.go:19-37`):
   ```go
   	// Track which slug directories we write this run
   	writtenSlugs := make(map[string]bool)

   	// Write per-package files
   	for _, pkg := range result.Packages {
   		if err := writePackageSpec(pkg, codeDir); err != nil {
   			return fmt.Errorf("writing package %s: %w", pkg.Name, err)
   		}
   		writtenSlugs[slugify(pkg.Path)] = true
   	}

   	// Write the index/overview
   	if err := writeCodeIndex(result, codeDir); err != nil {
   		return fmt.Errorf("writing index: %w", err)
   	}
   	writtenSlugs["index"] = true

   	// Remove stale directories from previous runs
   	pruneStaleDirectories(codeDir, writtenSlugs)
   ```

   **After**:
   ```go
   	// Track which slug directories we write this run
   	writtenSlugs := make(map[string]bool)

   	// Write per-package files
   	for _, pkg := range result.Packages {
   		if err := writePackageSpec(pkg, codeDir); err != nil {
   			return fmt.Errorf("writing package %s: %w", pkg.Name, err)
   		}
   		writtenSlugs[slugify(pkg.Path)] = true
   	}

   	// Write the index/overview
   	if err := writeCodeIndex(result, codeDir); err != nil {
   		return fmt.Errorf("writing index: %w", err)
   	}
   	writtenSlugs["index"] = true

   	// Build the prune keep-set from the COMPLETE current file set so an
   	// incremental scan (which only rebuilds changed packages) does not delete
   	// unchanged packages' directories. result.Checksums covers every current
   	// file on both full and incremental runs; genuinely-deleted packages are
   	// absent from it and are still pruned.
   	keep := make(map[string]bool, len(writtenSlugs)+len(result.Checksums))
   	for slug := range writtenSlugs {
   		keep[slug] = true
   	}
   	for relPath := range result.Checksums {
   		keep[slugify(filepath.Dir(relPath))] = true
   	}

   	// Remove stale directories from previous runs
   	pruneStaleDirectories(codeDir, keep)
   ```
   - `filepath` is already imported in `generate.go`.
   - Why: the keep-set now reflects all current packages, so unchanged package
     dirs survive an incremental scan while deleted packages are still removed.

## Secondary Defects

Documented as noted follow-ups — NOT fixed here.

1. **Partial index/env/endpoint sections on incremental scans.** The regenerated
   `index/spec.md` package table (`generate.go:191-198` `writeCodeIndex`), plus
   `result.ConfigVars` (`scanner.go:224-227`) and `result.Endpoints`
   (`scanner.go:230-233`), are all built only from parsed (changed) files. On an
   incremental scan the index summary table and the env-var/endpoint sections are
   therefore partial. Fixing this fully requires carrying forward the complete
   prior `Result` (a larger change) and is not required to stop the directory
   data loss. Record as a follow-up; do not fix in this spec.

2. **Graph write path is safe — no parallel fix needed.**
   `internal/codescan/graph_ingest.go:33` `WriteGraph` uses `store.UpsertNode`
   (`graph_ingest.go:52,83,109,154`) which is idempotent/additive. There is no
   prune in the graph write path, so unchanged packages persist in the graph from
   prior runs. The deletion is confined to the markdown corpus under
   `.hero/knowledge/code/`.

## Boundaries

- Does NOT change the incremental-skip logic in the scanner (`scanner.go:210-214`)
  or attempt to make `result.Packages` complete on incremental runs.
- Does NOT fix the partial index/env-var/endpoint sections (secondary defect #1).
- Does NOT touch the graph write path (already safe).
- Does NOT reopen the `codescan-created-slug-frontmatter-followup` (wont-fix)
  `created:`/`slug:` frontmatter concern — that is a separate, closed issue.

## Risks

- **Slug-derivation mismatch.** If `slugify(filepath.Dir(relPath))` diverges from
  `writePackageSpec`'s `slugify(pkg.Path)`, either a live package dir gets pruned
  or a stale dir is retained. Mitigation: both derive from `filepath.Dir(file)`
  then `slugify`; the root case (`"."` → `"root"`) is covered by `slugify`'s
  empty/`"-"` guard. The regression test's genuinely-deleted assertion guards the
  retain-too-much direction; the unchanged-package assertion guards the
  prune-too-much direction.
- **Non-parseable files in Checksums.** Only files that reach `scanner.go:207` are
  in `result.Checksums` (parseable languages with a registered parser). Their
  directories map exactly to package dirs, so no spurious keep entries for
  non-package dirs.

## Validation

- `go build ./cmd/hero` — compiles.
- `go test ./...` — all pass, including the new regression test.
- Do NOT run `hero scan` against the real working tree (it clobbers
  `.hero/knowledge/`). Validation is `go test` + `go build` only.

### Regression test (add to `internal/codescan/codescan_test.go`)

Model on `TestGenerateKnowledge` (line 551) and `TestScanner`'s incremental
re-scan (lines 540-548):

1. Create a temp project with TWO packages, A (`internal/a/a.go`) and
   B (`internal/b/b.go`).
2. `scanner.Scan(nil)` → full scan; `GenerateKnowledge(result, codeDir)`.
   Assert both `internal-a/spec.md` and `internal-b/spec.md` exist.
3. Modify ONLY package A's file on disk (change its content so its checksum
   differs).
4. `scanner.Scan(result.Checksums)` → incremental (pass the first scan's
   checksums as prevChecksums); `GenerateKnowledge(result2, codeDir)` again.
5. Assert `internal-b/spec.md` STILL exists (the core regression — currently
   fails, pruned by the bug).
6. Assert `internal-a/spec.md` still exists (was rewritten).
7. Deleted-package case: `os.Remove` package B's file, `scanner.Scan(...)`
   incremental again, `GenerateKnowledge`, and assert `internal-b/` IS pruned
   (genuinely-deleted packages must still be removed).

## Completion Ledger

| Item | Status | Evidence |
|------|--------|----------|
| **Changes #1** — derive `pruneStaleDirectories` keep-set from `result.Checksums` | DONE | `internal/codescan/generate.go` — `GenerateKnowledge` builds `keep` as the union of `writtenSlugs` (`"index"` + written package slugs) and `slugify(filepath.Dir(relPath))` for every `result.Checksums` key, then passes `keep` (not `writtenSlugs`) to `pruneStaleDirectories`. |
| **Regression test** — two-package fixture, incremental A-only change, assert B survives; deleted-package assertion | DONE | `internal/codescan/codescan_test.go` — `TestGenerateKnowledgeIncrementalPreservesUnchangedPackages`. Verified FAILS on the pre-fix `writtenSlugs` keep-set (B's `spec.md` deleted) and PASSES on the fix. Third phase confirms a genuinely-deleted package IS pruned. |
| **`go build ./cmd/hero`** | DONE | Compiles (BUILD_OK). `go build ./...` also exits 0. |
| **`go test ./...`** | DONE | All packages pass (86 ok, 0 fail); codescan suite deterministic. |
| **Cold audit** | DONE | `delivery-audit.md` — verdict SHIP (clean), high confidence; slug derivation, deletion-still-prunes, and test-is-real-guard all independently verified. |
| **Secondary defect #1** — partial index/`ConfigVars`/`Endpoints` on incremental | SKIPPED (deferred) | Per spec Boundaries — requires carrying the full prior `Result` forward; out of scope for this data-loss fix. Recorded as a follow-up. |
| **Secondary defect #2** — graph write path | DONE (verified safe) | `graph_ingest.go` `WriteGraph` uses only `store.UpsertNode` (additive, no prune); unchanged packages persist. No fix needed. |

**Exercise-the-feature:** the regression test drives the real code path end
to end — `scanner.Scan` (incremental) → `GenerateKnowledge` →
`pruneStaleDirectories` → on-disk `os.Stat` assertions on the `.hero/knowledge/code/`
corpus. Running `hero scan` against the real tree is explicitly prohibited by
the spec (it clobbers `.hero/knowledge/`), so the test IS the exercise.
