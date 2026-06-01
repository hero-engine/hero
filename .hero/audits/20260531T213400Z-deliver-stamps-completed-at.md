# Delivery audit — deliver-stamps-completed-at

**Audited:** working-tree diff vs `HEAD` (commit `e7304a2`).
**Verdict:** SHIP
**Surface:** noteworthy
**Confidence:** high

This is a cross-repo inbound spec-out call from peer `hero-code`. The
contract is the field shape (`completed_at:` snake_case, RFC 3339 UTC,
no fractional seconds) and writer coverage so the peer can retire its
`git log -1 --format=%aI` fallback in the Sprint Dashboard resolver.

## Contract check (peer-visible shape)

- [✓] **Field name** — `completed_at` (snake_case) — `internal/spec/spec.go:1211`
  writes the canonical key; idempotency check accepts both forms but
  only emits snake_case.
- [✓] **Format** — `time.RFC3339` (not `time.RFC3339Nano`) —
  `internal/spec/spec.go:1212` (`nowFn().Format(time.RFC3339)`).
- [✓] **UTC** — `internal/spec/spec.go:1187` (`time.Now().UTC()`); test
  assertions confirm the `Z` suffix is present in stamped output
  (`spec_test.go:1032`, `complete_test.go:640`).
- [✓] **No fractional seconds** — RFC3339 has none; verified by
  `TestStampCompletedAt_Idempotent` asserting exactly
  `2026-05-31T19:42:08Z`.
- [✓] **Reader tolerance** — `parseFrontmatter` case
  `"completed_at", "completedAt"` at `spec.go:445`; covered by
  `TestParseFrontmatter_CompletedAt_CamelCase`.
- [✓] **JSON wire shape** — `internal/snapshot/render_json.go:47`
  exposes `completed_at` on `recently_done[*]`; populated from
  `s.CompletedAt` with `ModifiedAt` fallback in
  `rollup.go:391-393` and `rollup.go:357-365`.

## Acceptance criteria

Spec lacks an explicit `## Acceptance Criteria` section; the
"acceptance bar" in Kickoff and the 12 Changes items together form
the deliverable contract. Each row below maps to a Changes item.

- [✓] **C1: `CompletedAt time.Time` on `Spec` struct + `parseFrontmatter` case** —
  `internal/spec/spec.go:110` (struct field with comment) and
  `:445-450` (switch case with RFC3339 + date-only fallback, accepts both
  `completed_at` and `completedAt`). Tests: `TestParseFrontmatter_CompletedAt_RFC3339`,
  `_DateOnly`, `_CamelCase` (`spec_test.go:1058-1130`).
- [✓] **C2: `nowFn`, `StampCompletedAt`, `frontmatterHasField`, `SwapNowFnForTest`** —
  `internal/spec/spec.go:1187, 1195-1199, 1206-1213, 1220-1243`. `nowFn`
  defaults to `time.Now().UTC()` so callers do not need to remember
  to UTC-normalize. Tests: `TestStampCompletedAt_Idempotent`,
  `_RespectsCamelCase` (`spec_test.go:1026-1056`).
- [✓] **C3: `updateFrontmatterStatus` stamps on `completed` transition** —
  `internal/cli/complete.go:264-267` gates the stamp on
  `newStatus == "completed"` and chains the stamp onto the same buffer
  before write. Test: `TestRunComplete_StampsCompletedAt`
  (`complete_test.go:613`).
- [✓] **C4: `autoArchiveIfCompleted` stamps via `stampCompletedAtFile` in BOTH branches** —
  `internal/cli/complete.go:210` calls `stampCompletedAtFile` BEFORE the
  `isAlreadyInSpecsDir` branch split, so both the already-in-specs path
  (line 211-217) and the move path (line 218-228) end with a stamped
  file. Helper at `:236-250`. Tests: `TestAutoArchiveIfCompleted_StampsWhenMissing`,
  `_LeavesExistingStamp` (`complete_test.go:684-740`).
- [✓] **C5: `UpdateSpecFrontmatter` case `"complete"` stamps** —
  `internal/tracking/tracking.go:127` chains `spec.StampCompletedAt`
  between status flip and claim-field removal. Test:
  `TestUpdateSpecFrontmatter_CompleteStampsTimestamp`
  (`tracking_test.go:17`).
- [✓] **C6: `backfillCompletedAtCmd` registered** —
  `internal/cli/admin.go:26` (`adminCmd.AddCommand(backfillCompletedAtCmd)`).
- [✓] **C7: New file `admin_backfill_completed_at.go`** — 158 LOC; behavior:
  - Walks `spec.Discover` and filters to
    `Status==StatusCompleted && CompletedAt.IsZero()` (line 68-72).
  - Calls `git log -1 --format=%aI -- <path>` from project root (line 128).
  - **Skips on empty output** (`raw == ""` returns `(zero, false)`) —
    never fabricates a timestamp (line 134-136).
  - `--dry-run` and `--quiet` flags wired (line 44-49).
  - **`index.Rebuild` gated on `!dryRun && stamped > 0`** (line 108-112).
  - Uses `SetFrontmatterField` directly with the historical timestamp
    rather than `StampCompletedAt` (which would use `time.Now()`)
    (line 149-156).
  Tests: 4/4 (`admin_backfill_completed_at_test.go`).
- [✓] **C8: Snapshot rollup wires `CompletedAt` with `ModifiedAt` fallback** —
  `internal/snapshot/rollup.go:357-365` (initiatives) and
  `:391-393` (`rollupRecent`). JSON projection in
  `render_json.go:90-97` formats as RFC3339 UTC.
- [✓] **C9: `shippedFromSpecs` prefers `CompletedAt` via `shippedCompletionTime`** —
  `internal/serve/pages/work/data/shipped.go:72-74` (sort) and `:81`
  (display); helper at `:93-101` centralizes the
  `CompletedAt || ModifiedAt` rule.
- [✓] **C10: spec-format SKILL.md documents the field** —
  `domains/engineering/skills/spec-format/SKILL.md:266` adds the row to
  the General-conventions table covering name, format, write-side
  (Hero auto-stamps; do not hand-write), reader tolerance, and the
  backfill command reference.
- [✓] **C11: deliver.md notes auto-stamp** —
  `domains/engineering/commands/deliver.md:254-258` calls out that
  `hero spec verify` stamps automatically and hand-writing is
  unnecessary.
- [✓] **C12: tests** — covered alongside each row above. 14 tests
  total; all 5 spec_test, 4 complete_test, 1 tracking_test, 4 backfill_test.

## Validation evidence

- `go test ./internal/spec/... ./internal/cli/... ./internal/tracking/... ./internal/snapshot/...` →
  all `ok` (cached for spec/cli/tracking, 0.302s for snapshot).
- Lead independently re-ran `hero spec complete` on a fresh
  `in-review` spec — resulting frontmatter contains
  `status: completed` and `completed_at: 2026-06-01T03:31:18Z`.
- Engineer reported `hero spec verify` stamps via auto-archive on a
  pre-existing `status: completed` spec without the field.
- `hero admin backfill-completed-at --dry-run` against this repo
  reports 165 specs ready to stamp.

## Boundaries — verified not crossed

- [✓] **No retroactive overwrite.** `StampCompletedAt` short-circuits
  when the field exists (`spec.go:1207-1209`); backfill skips when
  `s.CompletedAt.IsZero()` is false (`admin_backfill_completed_at.go:72`).
- [✓] **No status-demotion handling.** No code path clears
  `completed_at`. Confirmed by reading every `completed_at` and
  `CompletedAt` reference; only writes are the two stamping sites and
  the backfill.
- [✓] **No stamping on non-work specs.** Stamping is gated either on
  `newStatus == "completed"` (`complete.go:265`) or on
  `s.Status == StatusCompleted` discovered through
  `autoArchiveIfCompleted` / backfill — and conventions, decisions,
  notes do not transition to `completed`.
- [✓] **No cross-repo `completed_at` projection.** No edits to
  `internal/peering/` or contracts; peer manifest unchanged in shape.
- [✓] **No event-log emitter changes.** `feed.AppendEvent` and the
  delivery-complete emission path are untouched.

## Audit notes

### One coverage gap worth surfacing (does not block ship)

**`internal/serve/refresh.go:136`** is a fourth writer site the spec
did not enumerate. The tracker-refresh loop calls
`updateSpecFrontmatterField(s.Path, "status", string(spec.StatusCompleted))`
when a tracker issue resolves to "Done" or equivalent. That helper
(`refresh.go:199`) only calls `spec.SetFrontmatterField` for the
status — it does NOT call `StampCompletedAt`.

Implications:

1. The post-condition safety net catches this in practice: the next
   `hero spec verify` (or the backfill command) will stamp the field
   on these specs. So no completed spec is ever permanently
   un-stamped.
2. The peer's contract is on field shape, not write latency. The
   gap means tracker-auto-resolved specs lack the stamp until the
   next verify pass — which may be never if no one runs verify on
   that slug. The dashboard would then fall back to `ModifiedAt` in
   `shippedCompletionTime` (which the engineer wired).
3. The fix is one line, and the spec's stated post-condition
   ("every Go writer that flips status to completed stamps in the
   same write") arguably requires it.

This is not a HOLD because:
- The spec explicitly named the three writer classes and didn't list
  refresh.go among them — the engineer matched the spec.
- The post-condition is preserved end-to-end via auto-archive +
  backfill.
- The peer's contract holds.

Recommend a follow-up spec or one-line fix-up in this same PR.
Filing as a flagged item for the orchestrator to decide.

### Other observations

- `nowFn` is `time.Now().UTC()` rather than `time.Now()` — slightly
  defensive vs. the spec snippet, but safer (a caller swapping in a
  local-tz `nowFn` can't break the UTC guarantee at the format
  site). Welcome variance, not a defect.
- `stampCompletedAtFile` in `complete.go:236` is non-exported and
  best-effort; matches the rest of `autoArchiveIfCompleted`'s
  stderr-log-and-continue stance.
- The 165-count from the lead's `--dry-run` run is good independent
  evidence the backfill walker hits both `.hero/specs/` and
  `.hero/planning/`.

## Open items

None — every Changes row landed with concrete evidence and an
assertive test. One coverage gap (refresh.go) is noted above as a
follow-up consideration, not a blocker.
