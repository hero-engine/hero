---
title: "Code-scan knowledge writer (`internal/codescan`) — created/slug frontmatter clobber follow-up"
slug: codescan-created-slug-frontmatter-followup
type: bug
status: completed
resolution: wont-fix
priority: low
severity: none
domain: engineering
size: small
created: 2026-07-12
root_cause_class: code
tags: [codescan, knowledge, frontmatter, created, slug, ephemeral, wont-fix, followup]
completed_at: 2026-07-12
related: [scan-clobbers-authored-created-slug-frontmatter]
---

# Code-scan knowledge writer — created/slug frontmatter clobber follow-up

## Issue

The completed bug `scan-clobbers-authored-created-slug-frontmatter` fixed
`hero scan`'s `internal/scan` knowledge writer, which was regenerating committed
`.hero/knowledge/` entries as a blind full-file overwrite that dropped authored
`slug:` and flipped `created:` to today. Its ## Secondary defects section flagged
the **code-scan** writer — `internal/codescan.GenerateKnowledge`, writing
`.hero/knowledge/code/*` with the `code-scan` tag — as OUT OF SCOPE and "likely
has the identical pattern (`created: 2026-07-11`, no `slug:`)," conditioned on
"if age-based queries matter for code-scan entries."

This spec confirms the investigation and resolves it: **the mechanical pattern is
present, but it is immaterial. Won't-fix.**

## Investigation

### Load-bearing claims (all `read` — verified against source this session)

- `read` — The mechanical pattern IS present in `internal/codescan/generate.go`:
  - `writePackageSpec` stamps `created: <today>` via `time.Now()`
    (`generate.go:78`) and emits **no** `slug:` line.
  - `writeCodeIndex` does the same (`generate.go:175`).
  - Both write via a blind `os.WriteFile` (`generate.go:160`, `:299`) — no
    read-merge of existing frontmatter.
  - No `slug:` literal appears anywhere in `internal/codescan/*.go`.
- `read` — codescan does **not** flow through `internal/scan/merge.go`. It has no
  `PlanMerge`/`decideMerge`/customization-preservation concept at all. So the
  merge-seam fix from the sibling spec does **not** cover it, and there is no
  shared path to reuse.
- `read` — `.hero/knowledge/code/` is **gitignored** (`.gitignore:37`:
  `.hero/knowledge/code/`) and has **0 files tracked** in git. Every other
  knowledge dir the sibling spec fixed (`context/`, `conventions/`, `decisions/`,
  `explainers/`, `notes/`, `rules/`) IS committed. `code/` is deliberately
  excluded.
- `read` — Code-scan entries are `type: context`, **not** work specs. The
  `created:` backfiller (`admin_backfill_created.go:68-69`) skips anything where
  `!IsWorkSpec() && Type != TypeInitiative`, and only walks `.hero/specs/` /
  `.hero/planning/`. It never touches code entries. Even if it did, they are
  gitignored, so `gitFirstCommitDate` returns no history.
- `read` — `GenerateKnowledge` calls `pruneStaleDirectories`
  (`generate.go:37,49`), which deletes any package directory not written in the
  current run. Combined with incremental scanning (`scanner.go:210-214` skips
  re-parsing unchanged files, so `result.Packages` holds only changed packages),
  code-scan output is a **pure, disposable projection** of current code
  structure — directories are created and destroyed by what changed since the
  last run.

### Root cause

Same shape as the sibling bug (fresh `created:`, no `slug:`, blind overwrite),
but in a writer whose output is by-design ephemeral rather than authored.

### Why it does NOT matter (the close-out reasoning)

The entire motivation for fixing `internal/scan` was two problems, **neither of
which exists for code-scan**:

1. **Working-tree git churn on committed files.** `internal/scan` writes
   committed entries, so every re-scan produced a `git diff` that had to be
   reverted. `.hero/knowledge/code/` is gitignored — there is no diff, no churn,
   nothing to revert.

2. **Undoing the `created-field-stamp-and-surface` backfill surfaced by
   `hero list`.** That backfill stamped authored `created:` on committed **work
   specs**, and `hero list --format json` reports their age. Code-scan entries
   are `type: context`, gitignored, never backfilled, and not work specs — no
   age-based query over them carries authored meaning. Their `created:`
   legitimately means "when this projection was last regenerated," which is
   exactly what an ephemeral, auto-pruned projection should report.

There is no authored `created:` to preserve (nothing backfills it), no authored
`slug:` to drop (nothing emits or backfills one, and a human does not hand-author
frontmatter into gitignored files that `pruneStaleDirectories` wholesale
deletes), and no committed-file churn. The sibling spec's condition — "if
age-based queries matter for code-scan entries" — is **not met**.

### Root cause classification

`code` (mechanically), but resolves as **wont-fix / by-design**: the writer's
disposable-projection contract is intentional and the frontmatter behavior is
correct for that contract.

## Decision — Won't fix

Applying the read-merge preservation (a `slug:` emission + read-existing-frontmatter
+ `spec.SetFrontmatterField` re-apply) to `internal/codescan` would add
provenance-preservation machinery for provenance that does not exist and is not
wanted, and would fight the deliberate prune-and-regenerate design. That violates
"Simplicity first" and "Surgical changes" — speculative complexity with no
motivating symptom.

**No regression test is added.** A test mirroring
`TestScanPreservesCreatedAndSlugOnUpdate` would assert preservation that we are
deliberately not implementing; a test asserting the current fresh-stamp behavior
would be a change-detector with no value. The correct artifact is this close-out
record so the follow-up is not re-investigated.

## Boundaries / out of scope (noted, not fixed here)

- **Incremental prune deletes unchanged package dirs.** Because incremental scan
  omits unchanged packages from `result.Packages` and `pruneStaleDirectories`
  removes any dir not written this run, an incremental `hero scan` appears to
  delete the knowledge dir of any package with no changed files. This is a
  distinct behavior from the created/slug question and out of scope for this
  follow-up — flagged for a separate look if it proves to matter.
- **AIDesc / LLM-enrichment content preservation** across scans is likewise a
  separate content-preservation concern, not a frontmatter-provenance one.

## Validation

- No code changed → no new tests. Baseline suite confirmed green to establish the
  tree was not left broken:
  - `go build ./cmd/hero` — BUILD OK.
  - `go test ./...` — PASS.
- Did **not** run `hero scan` against the working tree (clobbers
  `.hero/knowledge/`), per task constraint.

## Completion Ledger

| Item | Status | Evidence |
|---|---|---|
| Confirm codescan stamps fresh `created:` + emits no `slug:` | DONE | `generate.go:78,175` (`time.Now()`), no `slug:` literal in `internal/codescan/*.go` |
| Confirm overwrite vs. read-merge | DONE | blind `os.WriteFile` (`generate.go:160,299`); no `merge.go` path; `pruneStaleDirectories` (`generate.go:49`) |
| Determine materiality | DONE | `.hero/knowledge/code/` gitignored (`.gitignore:37`), 0 tracked files, `type: context` (excluded from backfill `admin_backfill_created.go:68`), pure disposable projection |
| Fix decision | DONE | **wont-fix** — no committed churn, no backfill, no authored provenance; read-merge would be speculative complexity |
| Regression test | N/A | preservation deliberately not implemented; no meaningful test to add |
| `go build ./cmd/hero` | DONE | BUILD OK |
| `go test ./...` | DONE | PASS |

## Kickoff

Follow-up to `scan-clobbers-authored-created-slug-frontmatter`: confirmed the
code-scan knowledge writer (`internal/codescan`, `.hero/knowledge/code/*`) shares
the mechanical created/slug clobber pattern but resolves **wont-fix** — those
entries are gitignored, `type: context`, never backfilled, and wholesale
regenerated + pruned every scan, so their `created:` correctly means
"regeneration time" and there is no authored `slug:`/`created:` to preserve.

**Status:** completed (wont-fix) — no code change; sibling merge-seam fix does not
and should not extend here.

**Files:** `internal/codescan/generate.go:78,175,160,299,37,49`,
`internal/codescan/scanner.go:210`, `.gitignore:37`,
`internal/cli/admin_backfill_created.go:68`
