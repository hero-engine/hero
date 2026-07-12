# Delivery audit — scan-clobbers-authored-created-slug-frontmatter

**Audited:** `git diff -- internal/scan/merge.go internal/scan/merge_test.go` (working tree, branch `main`)
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria (from ## Goal)
- [✓] On `MergeUpdate`/`MergeForce`, existing `created:` and `slug:` are preserved (read-merge) — `decideMerge` calls `preservedFields(existingContent)` and attaches it to the decision for both `force` (merge.go:76-82) and the auto-generated/imported `MergeUpdate` branches (merge.go:92-105); `writeEntry` re-applies them via `spec.SetFrontmatterField` after the fresh stamp (merge.go:272-280). Verified by `TestScanPreservesCreatedAndSlugOnUpdate` + fault injection.
- [✓] New entries (`MergeCreate`) keep today's `created:` and gain a `slug:` line — `MergeCreate` decisions carry `Preserve == nil`; `writeEntry` emits `slug: <entry.Slug>` unconditionally (guarded on frontmatter present), so `ci-fresh` lands `slug: ci-fresh` + today's `created:`. Asserted in the test (merge_test.go:409-410).

## Changes
- [✓] Change 1 — emit `slug:` in generated content, done centrally in `writeEntry` (not per-template): `content = spec.SetFrontmatterField(content, "slug", entry.Slug)`, guarded on `entry.Slug != "" && strings.HasPrefix(strings.TrimSpace(content), "---")` (merge.go:266-268). Covers generate/enrich/import uniformly with no template edits, as claimed.
- [✓] Change 2 — preserve authored `created:`/`slug:` at the merge seam: `MergeDecision.Preserve map[string]string` (merge.go:31), `frontmatterValue` + `preservedFields` helpers (merge.go:115-150), captured in `decideMerge`, re-applied in `writeEntry` before `os.WriteFile` (merge.go:272-280). Uses `spec.SetFrontmatterField` (internal/spec/spec.go:1749), which both replaces an existing key and inserts a missing one. New `internal/spec` import added; no cycle (build is green).
- [✓] Change 3 — customization heuristic untouched: `isUserCustomized`/`MergeSkipCustomized` do not appear in the diff. `MergeSkipCustomized` never writes, so preservation is correctly moot for it.

## Ordering / composition check
- Slug emission runs FIRST (from `entry.Slug`), preserved values applied SECOND (merge.go:266-280). On `MergeUpdate` the disk-read `slug`/`created` in `preserve` override the fresh stamp → authored values win. On `MergeCreate` `preserve` is nil → the emitted `entry.Slug` survives. Ordering is correct; no path writes today's `created` on update or loses a slug on create.

## Guard check
- The slug-injection guard (`entry.Slug != ""` AND content starts with `---`) prevents fabricating frontmatter around non-frontmatter fixtures, and does not skip real scan entries (all scan templates emit `---` frontmatter and set `Slug`). Confirmed correct in both directions.

## Regression test
- [✓] `TestScanPreservesCreatedAndSlugOnUpdate` (merge_test.go:293-420) is genuine, not tautological:
  - Existing on-disk fixture has `created: 2020-01-01` + `slug: ci-github-actions`; regenerated `GeneratedEntry` carries today's `created:` and NO `slug:` — the two differ, so a passing assert can only come from real preservation.
  - Asserts `PlanMerge` → `MergeUpdate` (not Skip/Create) before executing.
  - `assertField` reads the file back **from disk** (`os.ReadFile`) — not in-memory content — and also checks the literal `slug: <want>` line is present.
  - Executes the scan **twice**, re-asserting both fields (idempotency).
  - Separate `MergeCreate` case (`ci-fresh`) asserts today's `created:` + emitted `slug: ci-fresh`.
- **Fault injection (run this session):** reverting `writeEntry` to the blind `os.WriteFile(entry.Content)` makes the test FAIL — `slug = "", want "ci-fresh"` and `created` shows today (`2026-07-12`) instead of `2020-01-01`. Restoring the fix makes it PASS. The engineer's exercise-the-feature claim holds.

## Boundaries
- [✓] `isUserCustomized` unchanged (not in diff).
- [✓] `internal/codescan` untouched (not in `git status`).
- [✓] No attempt to backfill already-clobbered dates.

## Scope
- Diff limited to `internal/scan/merge.go` + `internal/scan/merge_test.go`. The `MergeResult` struct field realignment is gofmt whitespace normalization (`gofmt -l` reports clean). No scope creep. (Non-code: spec.md, `.hero/NEXT.md`, `.hero/next/chet-bellows.md`, `.hero/SNAPSHOT.md`, and the new planning dir are expected.)

## Validation results (independently re-run)
- `go test ./internal/scan/ -run TestScanPreservesCreatedAndSlugOnUpdate -v` → `--- PASS`.
- `go test ./internal/scan/...` → `ok`.
- `go build ./cmd/hero` → BUILD OK.
- `gofmt -l internal/scan/merge.go internal/scan/merge_test.go` → clean.
- Fault injection: fix reverted → FAIL; restored → PASS.

## Audit notes
- Every ledger claim verified against source or a re-run. No performative rows. No downgrades.
- Secondary defects called out in the spec (`internal/codescan` code-scan writer) are explicitly out of scope per `## Boundaries`; the central-injection approach means any strategy-based generator gets preservation for free, but code-scan slug emission remains a documented follow-up, not a gap in this delivery.
