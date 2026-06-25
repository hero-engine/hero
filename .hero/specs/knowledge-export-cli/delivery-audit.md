# Delivery audit — knowledge-export-cli

**Audited:** requested working-tree diffs for `internal/cli/root.go`, `internal/cli/helpers_test.go`, `.hero/planning/features/knowledge-export-cli/spec.md`, `internal/knowledge/export.go`, `internal/knowledge/export_test.go`, `internal/cli/export.go`, and `internal/cli/export_test.go`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria
- [✓] Copy every regular file under `.hero/knowledge/**` preserving relative paths — `internal/knowledge/export.go:163`, `internal/knowledge/export.go:184`, `internal/knowledge/export.go:223`; test `internal/knowledge/export_test.go:32`.
- [✓] Create missing destination and child directories — `internal/knowledge/export.go:274`, `internal/knowledge/export.go:359`; test `internal/knowledge/export_test.go:57`.
- [✓] Default `fail` preflights differing files, reports conflicts, and writes nothing — `internal/knowledge/export.go:198`, `internal/knowledge/export.go:216`, `internal/knowledge/export.go:257`; tests `internal/knowledge/export_test.go:84` and `internal/cli/export_test.go:111`.
- [✓] `--conflict skip` leaves existing paths unchanged and copies missing files — `internal/knowledge/export.go:259`; tests `internal/knowledge/export_test.go:107` and `internal/cli/export_test.go:52`.
- [✓] `--conflict overwrite` atomically replaces conflicting files without pruning extras — `internal/knowledge/export.go:261`, `internal/knowledge/export.go:359`; tests `internal/knowledge/export_test.go:129` and `internal/cli/export_test.go:52`.
- [✓] `--conflict merge` deterministically merges compatible markdown/frontmatter while preserving destination-owned content — `internal/knowledge/export.go:390`, `internal/knowledge/export.go:444`, `internal/knowledge/export.go:530`; tests `internal/knowledge/export_test.go:151` and `internal/cli/export_test.go:52`.
- [✓] `--conflict interactive` applies per-conflict `fail`, `skip`, `overwrite`, or `merge` choices — `internal/knowledge/export.go:291`, `internal/cli/export.go:77`; test `internal/knowledge/export_test.go:260`.
- [✓] Non-terminal `--conflict interactive` fails before export — `internal/cli/export.go:47`, `internal/cli/export.go:50`; test `internal/cli/export_test.go:136`.
- [✓] Merge rejects unsafe or ambiguous conflicts with path-specific errors — `internal/knowledge/export.go:173`, `internal/knowledge/export.go:237`, `internal/knowledge/export.go:242`, `internal/knowledge/export.go:390`, `internal/knowledge/export.go:460`; tests `internal/knowledge/export_test.go:196` and `internal/knowledge/export_test.go:213`.
- [✓] Destination inside source knowledge directory is rejected before walking files — `internal/knowledge/export.go:112`, `internal/knowledge/export.go:152`; test `internal/knowledge/export_test.go:214`.
- [✓] Source `.hero/knowledge/**` is never mutated — export writes only destination paths via plans, `internal/knowledge/export.go:327`; test `internal/knowledge/export_test.go:70`.
- [✓] CLI reports copied, skipped, overwritten, merged, identical, and conflicted counts — `internal/cli/export.go:67`; test `internal/cli/export_test.go:35`.

## Changes
- [✓] Added reusable export logic in `internal/knowledge/export.go` — strategies, options, summary, validation, walking, symlink rejection, planning, atomic writes, and interactive callback are present in `internal/knowledge/export.go:18`, `internal/knowledge/export.go:28`, `internal/knowledge/export.go:33`, `internal/knowledge/export.go:92`, `internal/knowledge/export.go:163`, `internal/knowledge/export.go:198`, `internal/knowledge/export.go:291`, and `internal/knowledge/export.go:359`.
- [✓] Implemented deterministic markdown/frontmatter merge helpers — `internal/knowledge/export.go:390`, `internal/knowledge/export.go:420`, `internal/knowledge/export.go:444`, `internal/knowledge/export.go:530`, and `internal/knowledge/export.go:588`.
- [✓] Added Cobra command `hero export knowledge <destination>` — `internal/cli/export.go:19`, `internal/cli/export.go:24`, `internal/cli/export.go:31`, `internal/cli/export.go:36`, `internal/cli/export.go:47`, `internal/cli/export.go:56`, `internal/cli/export.go:67`, and `internal/cli/export.go:77`.
- [✓] Registered `exportCmd` and reset export test flag state — `internal/cli/root.go:110` and `internal/cli/helpers_test.go:299`.
- [✓] Added core package tests — `internal/knowledge/export_test.go:32`, `internal/knowledge/export_test.go:57`, `internal/knowledge/export_test.go:70`, `internal/knowledge/export_test.go:84`, `internal/knowledge/export_test.go:107`, `internal/knowledge/export_test.go:129`, `internal/knowledge/export_test.go:151`, `internal/knowledge/export_test.go:196`, `internal/knowledge/export_test.go:213`, and `internal/knowledge/export_test.go:260`.
- [✓] Added CLI tests — `internal/cli/export_test.go:15`, `internal/cli/export_test.go:35`, `internal/cli/export_test.go:52`, `internal/cli/export_test.go:111`, and `internal/cli/export_test.go:152`.

## Open items
- None.

## Audit notes
- None.
