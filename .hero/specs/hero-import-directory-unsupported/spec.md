---
title: hero import <directory> fails with "is a directory" despite --help advertising directory imports
type: bug
status: completed
severity: high
root_cause_class: design
tags: [cli, import, knowledge, help-drift]
created: 2026-05-18
relates-to: [context-files-flag-drift, recap-unregister-stale-and-empty-repo]
completed_at: 2026-05-18T19:25:38Z
---

# `hero import <directory>` fails with "is a directory" despite `--help` advertising directory imports

## Kickoff

`hero import <directory>` now walks the tree and ingests each text-ish file as its own knowledge entry. Single-file and URL branches were refactored through the same `writeSingleIngest` helper so all three paths emit raw + stub identically. Per-file slug is `<groupSlug>-<fileSlug>`, every entry carries the `groupSlug` tag, and skip-if-exists runs per file so re-running after agent enrichment is safe.

**Status:** delivered — sanity-verified against `.hero/specs/` (112 files in, 112 raw + 112 stubs out, all tagged `spec-archive-test`).

**Pick up at:** nothing — the bug is fixed. Follow-up structural mitigation (`cli-examples-smoke-test`) tracked separately per the Related concerns section below.

→ `.hero/planning/bugs/hero-import-directory-unsupported/spec.md`

**Files delivered:** `internal/cli/import.go` (refactor + directory branch + `--max-bytes` flag), `internal/cli/import_test.go` (new — 8 tests covering walk/filter/cap/skip/empty/tag/single-file/URL-signature), `internal/cli/helpers_test.go` (added flag resets).
**Skip:** concatenating all files into one entry — rejected; per-file entries give better `hero search` granularity.

## Issue

User reported running:
```
hero import ~/code/api-reviews/doc-portal/docs/greenlake/standards/ratified/ "GLP API Standards"
```
and getting:
```
Error: reading file: read /Users/developer/code/.../ratified/: is a directory
```

The `hero import --help` text explicitly advertises directory imports:
```
Examples:
  hero import https://docs.example.com/api-reference "API Reference"
  hero import ./docs/architecture.md "Architecture Overview"
  hero import ./vendor-docs/ "Vendor Documentation"   <-- directory example
```

No tracker; reported in-session.

## Investigation

### Code trace — end to end

1. `internal/cli/import.go:18` — `importCmd` declares `Use: "import <url-or-file> [title]"`. The `Long:` description and Examples block at lines 21–28 advertise three input shapes: URL, single file, and directory. Note the `Use:` line itself omits "directory" — the docstring is more aspirational than the synopsis.
2. `internal/cli/import.go:43` — `runImport(cmd, args)` receives the source path as `args[0]` (`source`).
3. `internal/cli/import.go:50–60` — project root resolution, config load, raw-dir creation. No issue here.
4. `internal/cli/import.go:66–73` — URL branch. Detected via `strings.HasPrefix(source, "http://")` or `"https://"`. Calls `fetchURL`. Not the failure path.
5. `internal/cli/import.go:74–83` — the **non-URL else branch**:
   ```go
   } else {
       content, err = os.ReadFile(source)
       if err != nil {
           return fmt.Errorf("reading file: %w", err)
       }
       if title == "" {
           title = filepath.Base(source)
       }
       sourceName = source
   }
   ```
   `os.ReadFile` is called unconditionally on the source path. There is **no** `os.Stat` check, no `f.IsDir()` branch, no `filepath.Walk`, no `filepath.WalkDir`. When `source` resolves to a directory, the underlying `read(2)` syscall returns `EISDIR`, which Go surfaces as the wrapped `is a directory` error the user sees.
6. `internal/cli/import.go:85–158` — slug generation, raw write, stub knowledge entry write. All single-blob logic. Even if the read succeeded, this code path produces exactly one slug, one raw file, one knowledge entry — there is no concept of per-file fan-out anywhere in this file.

### Confirmation

The bug is in our code, not an environment, data, or external factor. The help text was written aspirationally (or for a planned-but-never-shipped feature) and the implementation never caught up. Grep for `Walk`, `WalkDir`, `ReadDir`, or directory-related logic in `internal/cli/import.go` returns nothing — the directory path is genuinely unimplemented, not a regression.

### Root cause

**design / scope drift between documentation and implementation.** The `--help` `Long:` and `Examples:` block at `internal/cli/import.go:21–28` promises a feature (`hero import ./vendor-docs/`) that was never implemented. The `else` branch at line 74 unconditionally calls `os.ReadFile`, which fails with `EISDIR` for any directory input.

This is a **design** classification rather than **code** because the implementation is internally consistent — it correctly handles every shape it was written to handle. The defect is that the surface area (the advertised contract) is wider than the implementation. Calling it `code` would imply someone wrote buggy code; the more accurate framing is that the contract drifted from reality.

### Severity

**high.** Ingest is core to Hero's value proposition ("compounding knowledge"). A user trying to import a standards directory — the exact use case Hero is supposed to support best — hits a hard error with a misleading message ("reading file" when the user pointed at a directory) and no hint about how to proceed. There is no workaround other than scripting a manual per-file loop outside Hero, which defeats the point. Anyone following the documented example fails immediately.

## Goal

`hero import <directory> <title>` walks the directory, ingests each text-ish file as its own knowledge entry (slugged `<title-slug>-<filename-slug>`), tags them with a shared `<title-slug>` tag so the import group is retrievable via `hero search --tag`, and respects the existing "skip if entry already exists" semantics on a per-file basis so re-running after enrichment doesn't clobber agent work.

## Approach — fix design

Add a `os.Stat` branch before the existing `os.ReadFile` call in `runImport`. If the source is a directory, extract a helper (`ingestDirectory`) that walks it, filters, and loops the existing per-file ingest logic. If the source is a regular file, the existing single-file branch runs unchanged.

### Pseudo-code

```go
} else {
    info, err := os.Stat(source)
    if err != nil {
        return fmt.Errorf("stat: %w", err)
    }
    if info.IsDir() {
        return ingestDirectory(source, title, importType, importTag, heroDir, rawDir)
    }
    content, err = os.ReadFile(source)
    if err != nil {
        return fmt.Errorf("reading file: %w", err)
    }
    if title == "" {
        title = filepath.Base(source)
    }
    sourceName = source
}
```

```go
func ingestDirectory(dir, groupTitle, kType, userTag, heroDir, rawDir string) error {
    if groupTitle == "" {
        groupTitle = filepath.Base(filepath.Clean(dir))
    }
    groupSlug := slugify(groupTitle)

    var ingested, skipped int
    err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
        if walkErr != nil { return walkErr }
        if d.IsDir() {
            // Skip hidden dirs (.git, .obsidian, etc.)
            if path != dir && strings.HasPrefix(d.Name(), ".") {
                return fs.SkipDir
            }
            return nil
        }
        if strings.HasPrefix(d.Name(), ".") { return nil }      // hidden file
        if !isTextExt(d.Name())             { return nil }      // extension filter
        info, _ := d.Info()
        if info.Size() > maxIngestFileBytes { return nil }      // size cap

        content, err := os.ReadFile(path)
        if err != nil { return fmt.Errorf("reading %s: %w", path, err) }

        rel, _ := filepath.Rel(dir, path)
        fileSlug := slugify(strings.TrimSuffix(rel, filepath.Ext(rel)))
        slug     := groupSlug + "-" + fileSlug
        entryTitle := groupTitle + " — " + rel

        // Write raw + stub using existing logic, parameterized by slug/title/tag.
        // Tags: [groupSlug, ingested] (+ userTag if non-empty)
        // Skip if entry already exists (existing behavior at import.go:104).
        wrote, err := writeSingleIngest(writeArgs{
            slug: slug, title: entryTitle,
            content: content, sourceName: path,
            kType: kType, extraTags: []string{groupSlug, userTag},
            rawDir: rawDir, heroDir: heroDir,
        })
        if err != nil { return err }
        if wrote { ingested++ } else { skipped++ }
        return nil
    })
    if err != nil { return err }
    if ingested == 0 && skipped == 0 {
        return fmt.Errorf("no ingestable files found under %s (check extension filter and size cap)", dir)
    }
    fmt.Printf("Ingested %d files (%d skipped as already-present) from %s under tag %q\n",
        ingested, skipped, dir, groupSlug)
    return nil
}

func isTextExt(name string) bool {
    switch strings.ToLower(filepath.Ext(name)) {
    case ".md", ".markdown", ".txt", ".rst", ".adoc", ".mdx", ".org",
         ".json", ".yaml", ".yml":
        return true
    }
    return false
}

const defaultMaxIngestFileBytes = 1 << 20 // 1 MiB — override with --max-bytes
```

### Refactor of existing single-file path

Pull the lines `internal/cli/import.go:85–158` (slug-from-title, raw write, stub write, skip-if-exists) into a helper (`writeSingleIngest` above). The existing single-file and URL branches call it too, eliminating duplication and ensuring the per-file directory loop and the legacy single-file path behave identically.

### Slug grouping rationale

`<groupSlug>-<fileSlug>` means an import of `"GLP API Standards"` pointing at `naming.md` produces slug `glp-api-standards-naming`. All entries from one directory import share the `glp-api-standards-` prefix and the `glp-api-standards` tag, so:
- `hero search --tag glp-api-standards` retrieves the entire import group.
- File listings in `.hero/knowledge/<type>/` group naturally by prefix.
- The agent enriching one entry has a clear "sibling group" to cross-reference.

### Resolved decisions

1. **File-size cap.** Default **1 MiB**, **configurable via `--max-bytes` flag**. The flag overrides the default; the default is a constant `defaultMaxIngestFileBytes = 1 << 20`. Skipped files emit a per-file note so the user sees what dropped.
2. **Extension allowlist.** `.md`, `.markdown`, `.txt`, `.rst`, `.adoc`, `.mdx`, `.org`, `.json`, `.yaml`, `.yml`. Structured-data formats are included by default — API standards repos commonly include OpenAPI/JSON Schema docs alongside prose.
3. **`.gitignore` respect.** **No** for v1. Hidden-dir filter (`.git`, `.obsidian`, dotfiles) handles common cases. Revisit if a user reports real friction.
4. **Empty-after-filter behavior.** **Exit non-zero with a clear message**: `no ingestable files found under <dir> — check extension filter and size cap`.
5. **`Use:` line.** Update from `"import <url-or-file> [title]"` to `"import <url-or-file-or-directory> [title]"`.

## Changes

1. **`internal/cli/import.go`** — add directory branch.
   - Insert `os.Stat` check before `os.ReadFile` at line 75.
   - On `info.IsDir()`, delegate to new `ingestDirectory` helper.
   - Update `Use:` at line 19 to `"import <url-or-file-or-directory> [title]"`.
   - Add `--max-bytes` flag (uint, default `defaultMaxIngestFileBytes`) — applies to directory walk and is wired into the `ingestDirectory` size filter.

2. **`internal/cli/import.go`** — extract `writeSingleIngest` helper from current lines 85–158.
   - Parameters: slug, title, content, sourceName, kType, extraTags, rawDir, heroDir.
   - Returns `(wrote bool, err error)` — `wrote=false` when the skip-if-exists branch triggers.
   - Call from both single-file/URL branch and the new directory walk.

3. **`internal/cli/import.go`** — add `ingestDirectory`, `isTextExt`, and `maxIngestFileBytes` constant.
   - `filepath.WalkDir` traversal.
   - Hidden-dir skip (`fs.SkipDir` when `path != dir && strings.HasPrefix(d.Name(), ".")`).
   - Hidden-file skip, extension filter, size cap.
   - Per-file slug `<groupSlug>-<fileSlug>` (where `fileSlug` derives from the relative path with extension stripped).
   - Per-file tag set includes `groupSlug` so the import group is queryable.
   - Summary line at end: `Ingested N files (M skipped as already-present) from <dir> under tag <groupSlug>`.
   - Error out if zero files matched.

4. **`internal/cli/import_test.go`** (new file) — tests for the three branches:
   - URL branch unchanged (mock `http.Get` or skip behind a build tag).
   - Single-file branch unchanged behavior.
   - Directory branch:
     - Walks recursively.
     - Filters extensions, hidden files, hidden directories.
     - Skips oversize files.
     - Per-file slug derivation.
     - Group tag applied to all entries.
     - Re-run after one entry is hand-edited does NOT overwrite it (skip-if-exists per file).
     - Empty-after-filter directory returns a clear error.

## Boundaries

- Not touching the URL branch. `fetchURL` is unchanged.
- Not adding `--gitignore` or `--ext` flags in this fix. Defaults are baked in; flags are a follow-up if users ask.
- Not changing the knowledge entry template (lines 116–144). The stub format is fine; we just emit one per file.
- Not building a cross-cutting "help text validates against implementation" framework in this spec — that's the shared mitigation flagged below.

## Risks

- **Slug collisions.** If two files have the same slugified path (e.g. `auth/login.md` and `auth-login.md` both slugify to `auth-login`), the second silently skips via the "entry already exists" check. Acceptable for v1 — collisions are rare in real docs trees and the skip-message tells the user. Worth a comment in the helper.
- **Walking a giant tree.** Pointing at `~/` or `/` would walk the whole filesystem. The extension filter caps damage but the walk itself is unbounded. Mitigation: print a "scanning X..." line so the user notices and can `^C`. Not adding a depth limit for v1.
- **Symlinks.** `filepath.WalkDir` follows symlinks to files but not symlinked directories by default. Acceptable. Don't `os.Lstat`-dance unless a user reports an issue.
- **Existing `--tag` flag interaction.** The current `--tag` flag is a single user-supplied tag. The new code adds the auto-derived `groupSlug` tag. Both should be applied. The helper should de-dupe tags.

## Validation

- New unit tests under `internal/cli/import_test.go` cover the directory walk, filters, slug scheme, and skip-if-exists per file.
- Manual reproduction: build the binary, run `hero import ./docs/ "Docs Test"` against a temp tree, verify N `.hero/knowledge/raw/docs-test-*.md` files appear and N stubs are created under `.hero/knowledge/context/`.
- Manual regression: `hero import ./single-file.md "Single"` still works as before.
- Manual regression: `hero import https://example.com "Example"` still works.
- `hero search --tag docs-test` returns the import group after `hero index`.

## Related concerns — help-vs-reality drift pattern

This bug is one of three reported in the same batch where the user-facing surface (`--help`, skill instructions, command synopses) promised behavior the implementation didn't deliver:

- **This spec** (`hero-import-directory-unsupported`) — `--help` advertises directory imports; code only handles files and URLs.
- **`context-files-flag-drift`** — a `--context-files` (or similar) flag is documented but not wired through to the code path.
- **`recap-unregister-stale-and-empty-repo`** — recap behavior or registration flow promises something the implementation doesn't do correctly.

All three are the same defect class: **documented contract drifted past the implementation**. Each one in isolation is a small fix. As a pattern, it suggests we don't have a guard against this drift.

### Proposed shared mitigation

Add a CLI smoke test that:
1. Extracts every `Examples:` block from every `cobra.Command` in `internal/cli/`.
2. Runs each example invocation against a fixture project (`testdata/smoke-fixture/`) with a temp `.hero/` workspace.
3. Asserts non-zero-exit on any example, and asserts the documented side effect happened (file created, line written, etc. — where checkable).

This catches the `hero import ./vendor-docs/` case because the example would fail with `is a directory` and the test would fail. It catches future drift the same way: if you change help text to advertise something, the test forces you to either implement it or remove the example. If you change implementation in a way that breaks an example, the test catches it.

Scope this mitigation as its own follow-up spec (`cli-examples-smoke-test` or similar) — not in this fix. This bug should land first; the smoke test is the structural improvement that prevents the *next* one of these.
