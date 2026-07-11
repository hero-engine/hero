# Delivery audit — created-field-stamp-and-surface

**Audited:** `git diff HEAD -- internal/` (uncommitted working tree) + 2 untracked files
**Verdict:** SHIP
**Surface:** clean

Cold audit — verified against on-disk code and tests, not the Completion Ledger.
All tests were run and read for non-vacuousness.

## Acceptance criteria

- [✓] AC-1 (list JSON emits `created`) — `internal/cli/list.go:314-317` adds
  `Created string json:"created,omitempty"` and populates
  `s.CreatedAt.Format("2006-01-02")`. Test `TestListJSONFormatIncludesCreated`
  (`list_test.go:219`) asserts the literal `"created": "2026-06-29"`. Non-vacuous.
- [✓] AC-2 (`CreatedFromFrontmatter` signal) — set `true` in **both** parse
  branches of `case "created":` (`spec.go:533,536`), left `false` otherwise.
  `TestParseSpec` asserts `true` when present; `TestParseCreatedFallsBackToMtime`
  asserts `false` + `CreatedAt == mtime` when absent. Both assert the field directly.
- [✓] AC-3 (backfill stamps missing / skips present) — `runBackfillCreated`
  filters `!CreatedFromFrontmatter`, stamps via `writeCreatedStamp`.
  `gitFirstCommitDate` uses `git log --follow --reverse --format=%aI` and takes
  `lines[0]` = OLDEST commit. `TestBackfillCreated_HappyPath` asserts
  `created: 2025-01-15` (the first-commit author date); `_SkipsAlreadyStamped`
  asserts skip count 1 and that the original `created: 2024-06-01` is preserved.
- [✓] AC-4 (uncommitted → today) — `createdDate` falls back to
  `time.Now().UTC().Truncate(24h)` when git returns no history.
  `TestBackfillCreated_UncommittedStampsToday` asserts `created: <today>`.
- [✓] AC-5 (`--dry-run` writes nothing) — dry-run branch counts but continues
  before `writeCreatedStamp`. `TestBackfillCreated_DryRun` asserts the file
  contains no `created:` after the run.
- [✓] AC-6 (`check --reconcile` self-heals) — `check.go:263-294` adds a dedicated
  missing-created section that stamps under `--reconcile`.
  `TestCheckReconcileStampsCreated` asserts the file gets `created: 2025-01-15`
  written; `TestCheckReportsMissingCreatedWithoutReconcile` asserts plain `check`
  reports but does not write.
- [✓] AC-7 (idempotent) — `TestBackfillCreated_Idempotent` runs backfill twice,
  asserts second run reports `Stamped: 0` / `Skipped: 1`. `resetFlags`
  (`helpers_test.go:219-224`) now resets the four backfill bool flags so the
  dry-run test cannot leak into the idempotent test via cobra's persisted bools.
- [✓] AC-8 (build + tests) — verified by the auditor: `go build ./...` clean;
  `go test ./internal/cli/... ./internal/spec/... ./internal/reconcile/...` all `ok`.

## Changes

- [✓] `internal/spec/spec.go` — `CreatedFromFrontmatter bool` on `Spec`, set in
  both `case "created":` branches.
- [✓] `internal/cli/list.go` — `created` field in `renderSpecsJSON` row.
- [✓] `internal/cli/admin_backfill_created.go` (new) — command + `createdDate` /
  `gitFirstCommitDate` / `writeCreatedStamp` / `workSpecsMissingCreated` helpers.
- [✓] `internal/cli/admin.go` — registers `backfillCreatedCmd`.
- [✓] `internal/cli/check.go` — report-only missing-created section, stamps under `--reconcile`.
- [✓] `internal/cli/helpers_test.go` — resets backfill flags in `resetFlags`.
- [✓] Tests — `admin_backfill_created_test.go` (new, 7 tests), `list_test.go`,
  `spec_test.go`.

## Targeted verification (per audit brief)

1. **Backfill takes OLDEST commit** — `gitFirstCommitDate` uses `--reverse` and
   `lines[0]`, i.e. the oldest commit = the creation date. Confirmed distinct from
   the sibling `backfill-completed-at`, which correctly uses `-1` (most recent)
   for a different field. Uncommitted → today is intentional and tested (AC-4).
2. **Decoupling from reconcile** — the missing-created detector is
   `workSpecsMissingCreated` (`admin_backfill_created.go:118`), a direct
   `spec.Discover` + `!CreatedFromFrontmatter` filter in the CLI. `git diff HEAD --
   internal/reconcile` is **empty** — the reconcile package is unchanged from HEAD.
3. **No scope creep** — `git diff HEAD -- internal/` touches exactly the 7 named
   files (+2 new). `team.go`, `velocity.go`, `graph_ingest.go`, `install.go`
   confirmed unchanged from HEAD; no residual directory-wide gofmt churn.
4. **Overwrite safety** — `writeCreatedStamp` is only reached when
   `!CreatedFromFrontmatter` (i.e. no authored `created:`), so an authored date is
   never overwritten. `SetFrontmatterField` replaces in place if the key exists,
   else inserts before any tracker comment / closing `---` — correct with other
   fields present. `_SkipsAlreadyStamped` proves the authored value survives.

## Audit notes

- **`CreatedAt` is never zero for a file-parsed spec.** `Parse` sets
  `CreatedAt: modTime` at construction (`spec.go:407`), so the `!IsZero()` guard in
  `renderSpecsJSON` never actually elides `created` for a real on-disk spec — every
  row carries the field, satisfying AC-1's "every spec row." The design note says
  the JSON emits `created` "unconditionally"; the implementation adds a harmless
  defensive `IsZero` guard. No behavioral gap.
- `gitFirstCommitDate` parses lines with `strings.Fields`; safe because
  `%aI` (RFC 3339) timestamps contain no internal whitespace, so each line is one
  field and `lines[0]` is reliably the oldest commit.
- No performative rows: every DONE in the Completion Ledger maps to real code and a
  test whose body asserts the specific new behavior. No downgrades.
