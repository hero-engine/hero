---
title: "Surface `created` in list JSON, and make the created date reliable (stamp + backfill)"
slug: created-field-stamp-and-surface
type: enhancement
status: completed
priority: medium
domain: engineering
size: small
created: 2026-07-11
tags: [list, json, cli-contract, frontmatter, created, reconcile, backfill]
completed_at: 2026-07-11T19:32:58Z
---

# Surface `created` in list JSON, and make the created date reliable

## Goal

`hero list --format json` SHALL emit each spec's creation date, and that date
SHALL be trustworthy — sourced from an authored `created:` frontmatter field
rather than a file mtime that drifts on every edit.

## Kickoff

**Pick up at: DELIVERED — pending `hero spec verify`.** Both parts landed:
(A) `renderSpecsJSON` (`internal/cli/list.go`) emits `created` (date-only);
(B) `Spec.CreatedFromFrontmatter` distinguishes an authored `created:` from the
mtime fallback, `hero admin backfill-created` stamps missing dates from the
oldest git commit (today if uncommitted), and `hero check` reports/`--reconcile`
self-heals them. Tests: parse signal, list JSON field, backfill
(happy/skip/uncommitted/dry-run/idempotent), reconcile self-heal — `go test ./...`
green (86 pkgs). Verified on the real corpus: `--dry-run` reports 42 to stamp /
309 already stamped. Next: cold audit → `hero spec verify
created-field-stamp-and-surface`, then optionally run `hero admin
backfill-created` to stamp the 42 mtime-only specs for real.

## Problem

`renderSpecsJSON` serializes `slug, title, type, status, horizon, tags, pinned,
kickoff, path` — but not `created`. So "what specs were created in the last two
weeks?" cannot be answered from the machine-readable surface; it requires
scanning frontmatter (misses fieldless specs) or `git log` per file.

Naively surfacing `Spec.CreatedAt` would *look* authoritative while being wrong
for the 14 open specs (of 69) with no `created:` field, because `CreatedAt`
silently falls back to file mtime. mtime is updated on every edit, so a spec
authored 2026-06-29 but edited today reports today. This is exactly why a
frontmatter scan returned "0 created in the last 2 weeks" while a git
first-commit scan correctly found `intake-capture-loop` (created 2026-06-29).

## Root Cause

Two gaps compound:
1. **Surface gap** — `renderSpecsJSON` never emitted `created`.
2. **Data gap** — `created:` is not reliably present in frontmatter (only the
   authoring agent writes it, inconsistently), and the mtime fallback masks the
   absence with a plausible-but-wrong value. There is no code seam that
   guarantees `created:` gets stamped.

## Design

Resolve the "truthful vs mtime-guess" question by **making `CreatedAt` reliably
truthful**, not by marking the JSON as untrustworthy. Once every committed spec
carries an authored `created:`, the only remaining mtime-fallback case is a
brand-new uncommitted spec — where mtime ≈ now ≈ the real creation time, so the
value is correct. The JSON can then emit `created` unconditionally.

**1. Truthfulness signal (`internal/spec/spec.go`).** Add
`CreatedFromFrontmatter bool` to `Spec`; set it `true` in the `case "created":`
parse branch. This is the idempotent detector for "this spec has no authored
created date" that both the backfill and the reconcile pass filter on — needed
because `CreatedAt` itself is always non-zero (mtime fallback).

**2. Surface (`internal/cli/list.go` `renderSpecsJSON`).** Add
`Created string \`json:"created,omitempty"\`` to the row and populate it with
`s.CreatedAt.Format("2006-01-02")` — date-only, matching the authored
frontmatter convention (existing `created:` values are `YYYY-MM-DD`).

**3. Stamping mechanism.** A shared helper derives the creation date for a spec
missing `created:` from the **oldest** commit that touched the file (following
renames), truncated to date; falls back to today for an uncommitted spec.
Exposed two ways, mirroring the `completed_at` precedent:
- **`hero admin backfill-created`** — one-shot, re-runnable, `--dry-run` /
  `--quiet`, structurally identical to `backfill-completed-at`. Stamps every
  work spec with `!CreatedFromFrontmatter`, writes `created: YYYY-MM-DD` via
  `spec.SetFrontmatterField`, re-indexes.
- **`hero check --reconcile`** — the self-heal path. During reconcile, any work
  spec missing `created:` is stamped the same way, so newly-authored specs get a
  real date without depending on the authoring agent's discipline (consistent
  with how reconcile already self-heals status drift and completed_at).

Git source is the oldest-commit author-date (`git log --follow --reverse
--format=%aI -- <path>`, first line; or equivalent), distinct from
`backfill-completed-at`'s most-recent-commit (`-1`). Uncommitted specs get
today's date (unlike `completed_at`, which refuses to synthesize — here today
*is* the true creation date for a just-authored spec, so synthesizing is
correct).

## Acceptance Criteria

- AC-1: WHEN `hero list --format json` runs, THE SYSTEM SHALL include a
  `created` field (format `YYYY-MM-DD`) on every spec row, sourced from
  `Spec.CreatedAt`.
- AC-2: WHERE a spec's frontmatter declares `created:`, THE SYSTEM SHALL set
  `Spec.CreatedFromFrontmatter = true`; WHERE it does not, THE SYSTEM SHALL
  leave it `false` (even though `CreatedAt` is populated from mtime).
- AC-3: WHEN `hero admin backfill-created` runs, THE SYSTEM SHALL stamp
  `created: YYYY-MM-DD` (from the oldest commit touching the file, following
  renames) into the frontmatter of every work spec where
  `CreatedFromFrontmatter` is false, and SHALL skip specs that already have it.
- AC-4: IF a spec has no git history (uncommitted), THEN `hero admin
  backfill-created` SHALL stamp today's date (the true creation date for a
  just-authored file).
- AC-5: WHEN `hero admin backfill-created --dry-run` runs, THE SYSTEM SHALL
  report what it would stamp and write nothing.
- AC-6: WHEN `hero check --reconcile` runs, THE SYSTEM SHALL stamp `created:`
  into any work spec missing it (same source rule), so the field self-heals
  without a manual backfill.
- AC-7: THE SYSTEM SHALL be idempotent — a second `backfill-created` or
  `--reconcile` run stamps nothing new and reports all specs skipped.
- AC-8: `go build ./... && go test ./internal/cli/... ./internal/spec/...`
  passes.

## Validation

- List JSON: build a spec fixture with `created: 2026-06-29`, assert the JSON
  row has `"created": "2026-06-29"`; a fixture with no `created:` reports its
  mtime date and is a candidate for backfill.
- Parse: `CreatedFromFrontmatter` is true only when the frontmatter key is
  present.
- Backfill: a committed spec missing `created:` is stamped from its first-commit
  date; an already-stamped spec is skipped; an uncommitted spec is stamped with
  today; `--dry-run` writes nothing; re-run is a no-op.
- Reconcile: `hero check --reconcile` on a workspace with a fieldless committed
  spec leaves it carrying an authored `created:`.
- Regression: existing `backfill-completed-at`, list, and reconcile tests stay
  green.

## Scope

**In scope**
- `Spec.CreatedFromFrontmatter` field + parse wiring.
- `created` in `renderSpecsJSON`.
- `hero admin backfill-created` (mirror of `backfill-completed-at`).
- Reconcile self-heal stamping in `hero check --reconcile`.
- Shared oldest-commit-date helper.

**Out of scope**
- Changing the human (`text`/`kickoff`) list formats — JSON only for the new
  field (add to text later if wanted).
- Reworking `CreatedAt`'s mtime fallback itself — it stays as the last-resort
  default; the fix is ensuring `created:` is present so the fallback rarely
  fires.
- Backfilling `created` into a tracker.

## Changes

- `internal/spec/spec.go` — `CreatedFromFrontmatter bool` on `Spec`; set in
  `case "created":`.
- `internal/cli/list.go` — `Created` field in `renderSpecsJSON`'s row.
- `internal/cli/admin_backfill_created.go` — new command + oldest-commit-date
  helper; register in `internal/cli/admin.go`.
- `internal/cli/check.go` (or the reconcile implementation) — self-heal stamping
  pass.
- Tests: list JSON field, parse signal, backfill (stamp/skip/no-git/dry-run/
  idempotent), reconcile self-heal.

## Completion Ledger

| AC | Status | Note |
|----|--------|------|
| AC-1 (list JSON has `created`) | DONE | `Created` field in `renderSpecsJSON` (`internal/cli/list.go`), `s.CreatedAt.Format("2006-01-02")`; `TestListJSONFormatIncludesCreated` |
| AC-2 (`CreatedFromFrontmatter` signal) | DONE | field on `Spec`, set in `case "created":`; `TestParseSpec`/`TestParseCreatedFallsBackToMtime` cover present + mtime-fallback |
| AC-3 (backfill stamps missing, skips present) | DONE | `hero admin backfill-created` from oldest commit (`git log --follow --reverse`); `TestBackfillCreated_HappyPath` / `_SkipsAlreadyStamped` |
| AC-4 (uncommitted → today) | DONE | `createdDate` synthesizes today when no git history; `TestBackfillCreated_UncommittedStampsToday` |
| AC-5 (`--dry-run` writes nothing) | DONE | `TestBackfillCreated_DryRun` |
| AC-6 (`check --reconcile` self-heals) | DONE | dedicated missing-created section in `runCheck`; `TestCheckReconcileStampsCreated` (+ `TestCheckReportsMissingCreatedWithoutReconcile` for the report-only path) |
| AC-7 (idempotent) | DONE | `TestBackfillCreated_Idempotent`; also fixed `resetFlags` to reset the backfill bool flags |
| AC-8 (build + tests) | DONE | `go test ./...` 86 packages ok, 0 failed |

- [x] exercise-the-feature: fresh binary — `hero list --format json` carries `created` on every row (our spec shows `2026-07-11`); `hero admin backfill-created --dry-run` on the real corpus reports 42 to stamp / 309 already stamped, no writes.

## Design note (fork resolved)

The "truthful date vs mtime-guess" fork was resolved by **making `CreatedAt`
reliable** rather than marking the JSON untrustworthy: after backfill, every
committed spec carries an authored `created:`, so the only remaining
mtime-fallback case is a brand-new uncommitted spec — where mtime ≈ now ≈ the
true creation time. The JSON therefore emits `created` unconditionally.

Implementation note: the missing-created detector lives in the CLI (a direct
`spec.Discover` filter on `!CreatedFromFrontmatter`), **not** in the
`reconcile` package's Finding stream — folding a data-quality check into the
git-status-drift findings polluted every existing reconcile test and conflated
two concerns. Kept as its own `hero check` section.

## Changes

| # | Change | Status |
|---|--------|--------|
| 1 | `internal/spec/spec.go`: `CreatedFromFrontmatter` field + set in `case "created":` | DONE |
| 2 | `internal/cli/list.go`: `created` in `renderSpecsJSON` | DONE |
| 3 | `internal/cli/admin_backfill_created.go`: `hero admin backfill-created` + `createdDate`/`gitFirstCommitDate`/`writeCreatedStamp`/`workSpecsMissingCreated` helpers | DONE |
| 4 | `internal/cli/admin.go`: register command | DONE |
| 5 | `internal/cli/check.go`: missing-created self-heal section (report-only, stamps under `--reconcile`) | DONE |
| 6 | `internal/cli/helpers_test.go`: reset backfill flags in `resetFlags` | DONE |
| 7 | Tests across spec + cli packages | DONE |
