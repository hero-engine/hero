---
title: "`hero scan` clobbers authored `created:` and drops `slug:` on regenerated knowledge entries"
slug: scan-clobbers-authored-created-slug-frontmatter
type: bug
status: completed
priority: high
severity: medium
domain: engineering
size: small
created: 2026-07-12
root_cause_class: code
tags: [scan, knowledge, frontmatter, created, slug, regression, merge, idempotency]
completed_at: 2026-07-12T15:23:03Z
---

# `hero scan` clobbers authored `created:` and drops `slug:` on regenerated knowledge entries

## Issue

Running `hero scan` in a repo that already has auto-generated knowledge entries
(tagged `auto-generated, project-scan`, and also `imported`) rewrites their
frontmatter on regeneration:

- (a) **DROPS** the `slug:` field entirely.
- (b) **FLIPS** `created:` to today's date, overwriting the authored/original value.

Reproduces on this repo (hero-engine/hero). Example:
`.hero/knowledge/rules/ci-github-actions/spec.md` loses `slug: ci-github-actions`
and has `created: 2026-04-29` rewritten to today.

Affected entries observed (each `MergeUpdate` on re-scan):
- `.hero/knowledge/rules/ci-github-actions/spec.md` — `auto-generated, project-scan, ci`
- `.hero/knowledge/context/project-overview/spec.md` — `auto-generated, project-scan`
- `.hero/knowledge/context/architecture-overview/spec.md` — `imported, architecture`
- `dev-workflow`, `rules/project-rules` (same generator families)

**Why it matters.** This directly UNDOES the `created-field-stamp-and-surface`
feature (shipped 2026-07-11), which backfilled authored `created:` dates from
each spec's first git commit. Every `hero scan` re-clobbers those dates, so
`hero list --format json`'s `created` field and any age-based query silently
drift back to "today" for scan-regenerated knowledge, and the stable `slug:` is
lost. It bit twice during the CI pipeline work — scan runs left knowledge-spec
noise in the working tree that had to be reverted.

## Investigation

### Load-bearing claims (all `read` — verified against source this session)

- `read` — Every scan generator template stamps `created: %s` with a
  freshly-computed today's date and **no** template anywhere in `internal/scan`
  emits a `slug:` field.
- `read` — On re-scan, an existing `auto-generated`/`imported`, uncustomized
  entry is classified `MergeUpdate`, and `MergeUpdate` writes the file via a
  full `os.WriteFile` overwrite — the existing frontmatter is never read back
  or merged.
- `read` — The existing files' `slug:`/`created:` came from an out-of-band
  backfill (`created-field-stamp-and-surface` + `hero admin backfill-created`),
  not from scan — so scan has no knowledge of them and destroys them on rewrite.
- `read` — `internal/spec` does not import `internal/scan`; `internal/scan` may
  safely import `internal/spec` (no cycle), so `spec.SetFrontmatterField` is
  available for the fix.

### Code flow (end to end)

1. `internal/cli/scan.go:174` — `entries := scan.Generate(result, heroDir)`
   builds the `[]GeneratedEntry` to write.
2. `internal/scan/generate.go:24` — `date := time.Now().Format("2006-01-02")`.
   This single "today" value is threaded into every generator.
3. Generator templates stamp that date as `created:` and **omit `slug:`**:
   - `internal/scan/generate.go:89` (linter conventions), `:136` (CI rules —
     the `ci-github-actions` example), `:182` (multi-module), `:255` (test
     conventions).
   - `internal/scan/enrich.go:571` (rich project-overview).
   - `internal/scan/import.go:371` (architecture-overview), `:404`, `:438`,
     `:472`, `:536`, `:566` (conventions/rules/commands/module entries).
   - `GeneratedEntry` carries a `Slug` field (`generate.go:13`) but it is used
     only for the file path and console output — never written into frontmatter.
4. `internal/cli/scan.go:185` — `decisions := scan.PlanMerge(entries, scanForce)`.
5. `internal/scan/merge.go:56` — `decideMerge` reads the existing file. For an
   `auto-generated` (`merge.go:77`) or `imported` (`merge.go:86`) entry that is
   not user-customized, it returns `MergeUpdate`. The existing content is read
   here but only inspected for classification — its `created:`/`slug:` are
   discarded.
6. `internal/cli/scan.go:202` → `internal/scan/merge.go:184-188` — `MergeUpdate`
   calls `writeEntry`.
7. `internal/scan/merge.go:204-210` — `writeEntry` does
   `os.WriteFile(entry.Path, []byte(entry.Content), 0o644)` — a full overwrite
   with the freshly-generated content (today's `created:`, no `slug:`).

### Root cause

Two compounding gaps, both in `internal/scan`:

1. **`created:` flip** — the generator templates unconditionally stamp
   `created: <today>` (`generate.go:24` feeds `generate.go:136` et al.), and the
   merge path (`merge.go:184-210`) overwrites the whole file. On regeneration of
   an existing entry the authored `created:` is never preserved — it is
   overwritten with today's date. Confirmed root cause of (b).

2. **`slug:` drop** — no scan generator template ever emits a `slug:` line
   (grep of `internal/scan/*.go` for `slug:` returns nothing). The `slug:` in
   existing files was added out-of-band (the `created-field-stamp-and-surface`
   backfill). Because the merge overwrites the whole file with content that has
   no `slug:` line, the field is dropped. Confirmed root cause of (a).

The architectural defect is that `hero scan` treats regeneration as blind
overwrite rather than a read-merge: the merge seam already reads the existing
file (`merge.go:56`) but throws away the authored fields instead of carrying
them forward.

### Severity

Medium. Not a crash or data-loss of user prose (customized entries are still
skipped via `MergeSkipCustomized`), but it silently corrupts authored metadata
on every scan, defeating a just-shipped feature and polluting the working tree
with churn that must be manually reverted. High-ish annoyance/regression, low
blast radius. Caused entirely by our code. Workaround today: `git checkout --
.hero/knowledge/` after every scan.

## Goal

When `hero scan` regenerates an **existing** knowledge entry (strategy
`MergeUpdate` or `MergeForce`), it SHALL preserve the entry's existing `created:`
and `slug:` frontmatter values (read-merge), rather than emitting a fresh
`created:` date or dropping `slug:`. Newly created entries (`MergeCreate`) keep
today's `created:` and gain a `slug:` line from the known `GeneratedEntry.Slug`.

## Changes

1. Emit `slug:` in generated content so new entries are stable and re-scans have
   a slug to preserve.
   - In `internal/scan/generate.go`, `enrich.go`, and `import.go`, add a
     `slug: <slug>` line to each frontmatter template (immediately after
     `type:` / before `created:`), using the value already computed for
     `GeneratedEntry.Slug`. The slug is in scope at every generator
     (`ci-<name>`, `use-<name>`, `architecture-overview`, etc.).
   - Simpler alternative if touching every template is undesirable: inject the
     slug centrally in `writeEntry` via
     `content = spec.SetFrontmatterField(content, "slug", entry.Slug)` before
     writing (applies to all strategies uniformly).

2. Preserve authored `created:` and `slug:` on regeneration of an existing
   entry. Do the read-merge at the merge seam where the existing file is already
   read.
   - In `internal/scan/merge.go`, add a small helper to read a top-level
     frontmatter scalar from existing content:
     ```go
     func frontmatterValue(content, key string) string {
         lines := strings.Split(content, "\n")
         if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
             return ""
         }
         for i := 1; i < len(lines); i++ {
             t := strings.TrimSpace(lines[i])
             if t == "---" {
                 break
             }
             if strings.HasPrefix(t, key+":") {
                 return strings.TrimSpace(strings.TrimPrefix(t, key+":"))
             }
         }
         return ""
     }
     ```
   - Extend `MergeDecision` with a carry-over field, e.g.
     `Preserve map[string]string`. In `decideMerge`, when the strategy is
     `MergeUpdate` (and for `MergeForce` where an existing file was read),
     capture `created` and `slug` from `existingContent` into
     `decision.Preserve` if non-empty.
   - In `ExecuteMerge`/`writeEntry`, before the `os.WriteFile`, apply the
     preserved values to `entry.Content`:
     ```go
     for k, v := range d.Preserve {
         if v != "" {
             entry.Content = spec.SetFrontmatterField(entry.Content, k, v)
         }
     }
     ```
     `spec.SetFrontmatterField` (`internal/spec/spec.go:1749`) already handles
     both "replace existing key" and "insert missing key" — so it both restores
     a dropped `slug:` and overwrites the fresh `created:` with the authored
     one. No import cycle (`internal/spec` does not import `internal/scan`).
   - Leave `MergeCreate` untouched: brand-new entries correctly get today's
     `created:` and the emitted `slug:`.

3. Do not alter the customization heuristic — `MergeSkipCustomized` entries are
   already untouched; this fix only changes what gets written on `MergeUpdate`/
   `MergeForce`.

### Delivered (files actually changed)

- `internal/scan/merge.go` — added `Preserve map[string]string` to
  `MergeDecision`; added `preservedFields` + `frontmatterValue` helpers; capture
  `created`/`slug` in `decideMerge` for `MergeUpdate`/`MergeForce`; `writeEntry`
  now emits `slug: <entry.Slug>` (Change 1, central injection — covers
  generate/enrich/import uniformly) then re-applies preserved provenance via
  `spec.SetFrontmatterField` before `os.WriteFile` (Change 2). No template edits
  needed. Added `internal/spec` import (no cycle).
- `internal/scan/merge_test.go` — added `TestScanPreservesCreatedAndSlugOnUpdate`
  (twice-scan idempotency + `MergeCreate` fresh-stamp guard) and `assertField`
  helper.

## Regression test

Add a test in `internal/scan/merge_test.go` (or `scan_test.go`):

- **`TestScanPreservesCreatedAndSlugOnUpdate`**:
  1. Write an existing auto-generated entry to a temp `.hero/knowledge/...`
     path with `slug: ci-github-actions` and `created: 2020-01-01` and the
     `auto-generated, project-scan` tags (uncustomized — placeholders intact).
  2. Build a `GeneratedEntry` for the same path/slug whose content carries a
     different (today's) `created:` and, per the current bug, no `slug:`.
  3. `PlanMerge` → assert strategy is `MergeUpdate`; `ExecuteMerge`.
  4. Read the file back and assert `created: 2020-01-01` is unchanged and
     `slug: ci-github-actions` is present.
  5. Run the same scan-and-write a **second** time and re-assert both fields are
     still `2020-01-01` / `ci-github-actions` (idempotency — the core "scan an
     existing entry twice, assert unchanged" check).
- Also assert the `MergeCreate` path still stamps today's `created:` and a
  `slug:` for a brand-new entry, so the preservation logic doesn't leak into
  fresh creation.

## Boundaries

- Does not change the customization detection heuristics in `isUserCustomized`.
- Does not touch the code-scan knowledge writer
  (`internal/codescan.GenerateKnowledge`, `.hero/knowledge/code/*`); those
  entries are a separate generator (`code-scan` tag) and out of scope unless the
  same test surfaces the same defect there — noted below as a secondary concern.
- Does not attempt to backfill/repair already-clobbered dates; the
  `created-field-stamp-and-surface` backfill (`hero admin backfill-created`)
  remains the tool for restoring authored dates from git.

## Risks

- `spec.SetFrontmatterField` inserts a missing hero-level key before any tracker
  comment block; knowledge entries have no such block, so insertion lands inside
  the frontmatter as expected. Low risk, but the regression test asserts exact
  placement by reading the field back.
- If `MergeForce` is meant to fully reset an entry, preserving `created:`/`slug:`
  on force is a judgement call. Recommendation: still preserve — `--force` is
  about overwriting user *customizations*, not resetting provenance metadata.
  Call this out for the reviewer.

## Secondary defects

- The **same defect exists for `imported`-tagged entries** (architecture-overview,
  and the `import.go` conventions/rules/commands/module builders at lines 371,
  404, 438, 472, 536, 566): they stamp fresh `created:`, emit no `slug:`, and
  merge via the same `MergeUpdate` overwrite. The fix at the merge seam (Change
  2) covers them automatically since it is strategy-based, not
  template-specific — but the `slug:` emission (Change 1) must be applied to the
  import templates too, or done centrally in `writeEntry`.
- The code-scan writer (`internal/codescan`, `.hero/knowledge/code/*`, tag
  `code-scan`) likely has the identical pattern (`created: 2026-07-11`, no
  `slug:`). Not confirmed this session — flag for follow-up if age-based queries
  matter for code-scan entries.

## Validation

- `go test ./internal/scan/...` green, including the new regression test.
- `go test ./...` green (no cross-package fallout from the new `internal/spec`
  import in `internal/scan`).
- Manual: in a scratch checkout, note `created:`/`slug:` on
  `.hero/knowledge/rules/ci-github-actions/spec.md`, run `hero scan`, then
  `git diff .hero/knowledge/` — assert the file is unchanged (no `created:` flip,
  no `slug:` drop). Revert any scratch changes.

## Completion Ledger

Delivered in supervised mode (user pre-authorized implementation + testing).
Cold audit: SHIP — clean (see `delivery-audit.md`).

| Item | Status | Evidence |
|---|---|---|
| Goal — preserve `created:`/`slug:` on `MergeUpdate`/`MergeForce`; fresh stamp on `MergeCreate` | DONE | `internal/scan/merge.go` `decideMerge` captures existing `created`/`slug` into `MergeDecision.Preserve` for update/force; `writeEntry` re-applies before `os.WriteFile`; `MergeCreate` (`Preserve == nil`) keeps today's `created:` + emitted slug. Regression test asserts both paths. |
| Change 1 — emit `slug:` in generated content | DONE | Central injection in `writeEntry`: `spec.SetFrontmatterField(content, "slug", entry.Slug)`, guarded on `entry.Slug != ""` and frontmatter present. Covers `generate.go`/`enrich.go`/`import.go` without per-template edits. |
| Change 2 — preserve `created:`/`slug:` at merge seam via `SetFrontmatterField` | DONE | `MergeDecision.Preserve map[string]string`; `frontmatterValue` + `preservedFields` helpers (`merge.go` ~L115-149); captured in `decideMerge`; re-applied in `writeEntry` after slug emit so disk value wins on update. |
| Change 3 — customization heuristic untouched | DONE | `isUserCustomized` / `MergeSkipCustomized` unchanged; skipped entries carry no `Preserve` and never write. |
| Regression test `TestScanPreservesCreatedAndSlugOnUpdate` (twice-scan idempotency) | DONE | `internal/scan/merge_test.go`: writes existing entry `created: 2020-01-01` / `slug: ci-github-actions`, asserts strategy `MergeUpdate`, executes twice, asserts both fields unchanged both runs via disk readback. `-v` → PASS. |
| `MergeCreate` still stamps today's `created:` + `slug:` (no leak) | DONE | Same test, `ci-fresh` case asserts `created: <today>` and `slug: ci-fresh`. |
| `go test ./internal/scan/...` green | DONE | `ok github.com/hero-engine/hero/internal/scan` (lead re-ran, cached). |
| `go test ./...` green | DONE | Full suite PASS (engineer); no cross-package fallout from new `internal/spec` import. |
| `go build ./cmd/hero` green | DONE | `BUILD OK` (lead re-ran). |
| Exercise-the-feature: test fails on unpatched code, passes on patched | DONE | Independently reproduced by cold auditor: reverting `writeEntry` to blind `os.WriteFile` → test FAILS (`slug = ""`, `created = 2026-07-12` want `2020-01-01`); fix restored → PASS. Genuine guard, not a tautology. |

**Files changed:** `internal/scan/merge.go` (+`internal/spec` import; `MergeDecision.Preserve`; `frontmatterValue`/`preservedFields` helpers; `decideMerge` capture; `writeEntry` slug-emit + preserve-apply), `internal/scan/merge_test.go` (`TestScanPreservesCreatedAndSlugOnUpdate` + `assertField` helper). 217 insertions, 12 deletions across the two files.

**Out of scope (per Boundaries / Secondary defects):** `internal/codescan` writer (`.hero/knowledge/code/*`, `code-scan` tag) likely shares the pattern — untouched; flag for a follow-up spec if age-based queries matter there.

## Kickoff

Fixes `hero scan` clobbering authored `created:` and dropping `slug:` on
knowledge entries it regenerates — every re-scan currently resets those to
"today"/nothing and undoes the created-date backfill.

**Status:** planning — root cause confirmed, fix is read-merge at the scan merge seam.

**Pick up at:** in `internal/scan/merge.go`, capture existing `created:`/`slug:`
into `MergeDecision` during `decideMerge` (strategy `MergeUpdate`/`MergeForce`),
then re-apply them via `spec.SetFrontmatterField` in `writeEntry` before the
`os.WriteFile`. Also emit `slug:` for new entries (templates or centrally in
`writeEntry`). Then add the twice-scan idempotency regression test.

→ `.hero/planning/bugs/scan-clobbers-authored-created-slug-frontmatter/spec.md`

**Files:** `internal/scan/merge.go:56,184-210`, `internal/scan/generate.go:24,136`, `internal/scan/import.go:371`, `internal/spec/spec.go:1749`, `internal/scan/merge_test.go`
**Skip:** editing every generator template for `created:` — preservation belongs at the merge seam, not the templates; the templates only need the `slug:` line.
