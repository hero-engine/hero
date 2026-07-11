---
title: "Stamp completed_at into spec frontmatter on status transition"
type: feature
status: completed
priority: high
horizon: now
received_from:
  peer_id: cd8dd06d-3df1-4878-a88f-24593dcbb4b3
  peer_alias_display: hero-code
  originator_slug: hero-app-sprint-dashboard
  handed_off_at: 2026-06-01T02:59:55Z
  at_commit: bee38d6
  reason: "Sprint dashboard depends on completedAt timestamps; stamping them in /deliver removes our git fallback path."
completed_at: 2026-06-01T03:39:24Z
created: 2026-06-01
---

# Stamp `completed_at` into spec frontmatter on status transition

## Provenance

Spec-out call from peer `hero-code` (peer_id `cd8dd06d-3df1-4878-a88f-24593dcbb4b3`).
Originator spec: `hero-app-sprint-dashboard` — Sprint Dashboard surface in
`apps/hero-desktop-mac` that needs to know when each spec was completed.
The desktop side currently uses a three-tier fallback to resolve completion
time: frontmatter `completed_at:` (authoritative) → `git log -1 --format=%aI`
(backfill) → file mtime (low-confidence). The git fallback works but is
slow at scale and gets noisy once specs are edited post-completion.

**Reason for the call:** make frontmatter the single source of truth so the
git fallback can be retired.

## Context

Hero specs are markdown files with YAML frontmatter. A work spec
(`type: feature` / `type: bug`) progresses through a lifecycle:
`planning → in-review → delivering → completed`. The `status` field carries
the lifecycle position; nothing today carries the *moment* the spec
crossed into `completed`.

Several consumers want that moment:
- **Sprint Dashboard (hero-code, `apps/hero-desktop-mac`)** — progress
  ring, velocity sparkline, recently-shipped list, weekly velocity buckets.
  All week-boundary-aware, so a date-only field is insufficient.
- **Hero `internal/serve/pages/work/data/shipped.go` (Recently shipped
  tile)** — currently falls back to `s.ModifiedAt` (file mtime) when the
  events log is empty.
- **Hero `internal/tracking/velocity.go`** — computes per-agent cycle time
  from claim/complete events in the events log. A frontmatter timestamp
  would survive log compaction and let cycle-time recompute from specs
  alone.
- **Hero people-and-roi-home spec** (planned) — explicitly references
  `completed_at - claimed_at` and `completed_at - created_at` as primary
  metric inputs.

There is no Hero-side migration of `completedAt` to `completed_at` because
the field has never been written. snake_case `completed_at:` is the
canonical name per Hero's existing frontmatter convention (`claimed_by`,
`claimed_at`, `delivery_method`, `tracker_id`, `at_commit`,
`handed_off_at`).

## Goal

Whenever a spec's `status` flips to `completed`, its frontmatter gains a
canonical `completed_at:` line with an RFC 3339 / ISO 8601 UTC timestamp.
The write is idempotent (re-runs don't churn the value), covers every
code path Hero owns that performs the transition, and is documented for
the model-driven `/deliver` flow that edits frontmatter directly. A
one-shot backfill command derives historical values from
`git log -1 --format=%aI` so the desktop app's git fallback can be retired.

## Approach

There are three classes of writer that flip status to `completed`:

| Writer | Where | Trigger |
|---|---|---|
| `internal/cli/complete.go` → `updateFrontmatterStatus` | `hero spec complete <path>` | Human or script invocation |
| `internal/tracking/tracking.go` → `UpdateSpecFrontmatter(action="complete")` | claim/release/complete bookkeeping path | Programmatic claim lifecycle |
| The `/deliver` model agent (markdown rewrite) | model edits the spec file with the rewrite tool, then runs `hero spec verify <slug>` | Async + supervised delivery |

Of these, only the first two are Go writers we can patch directly. The
third path mutates frontmatter from inside the agent's prompt and then
calls `hero spec verify <slug>`. To cover that path without
re-architecting the agent flow, we add stamping into
`autoArchiveIfCompleted` (the auto-archive hook `hero spec verify` runs):
if it observes `status: completed` and no `completed_at:`, it stamps one
right then. This makes the stamping invariant *post-condition driven*
("a completed spec on disk has `completed_at`"), not *write-site driven*
("every writer remembers to stamp"). The post-condition wins because
future writers (and humans editing by hand) can't forget.

Concretely:

1. Add a helper `spec.SetCompletedAt(content) string` that:
   - Returns content unchanged if `completed_at:` is already present.
   - Otherwise calls `spec.SetFrontmatterField(content, "completed_at", time.Now().UTC().Format(time.RFC3339))`.
   The current-time formatter lives at the helper, not the call site, so
   tests can override via a package-level `nowFn` like other timestamp
   sites do.
2. Patch every Go writer that flips status to `completed` to call the
   helper on the same content buffer, in the same write, before
   `os.WriteFile`.
3. Patch `autoArchiveIfCompleted` to re-read the spec after parse and
   stamp `completed_at:` if the status is `completed` and the field is
   missing. This catches the model-driven path that bypasses the Go
   writers.
4. Extend the `spec.Spec` struct + `parseFrontmatter` with a
   `CompletedAt time.Time` field, so downstream consumers (web dashboard,
   tracking/velocity, future ROI tile) can read it without re-parsing.
5. Document the field in `domains/engineering/skills/spec-format/SKILL.md`
   and the `/deliver` agent prompt, calling out that Hero stamps it
   automatically — agents and humans should not write it by hand.
6. Add `hero admin backfill-completed-at` (nice-to-have, ship in same
   PR) — walks every `status: completed` spec under `.hero/specs/` and
   `.hero/planning/` missing the field, runs
   `git log -1 --format=%aI -- <spec-path>` against the spec file, and
   stamps the result. Reports the count of stamped, skipped, and
   git-empty (specs with no git history).

### Idempotency

The post-condition is: *if status is completed and completed_at is
missing, set it.* Reverse: *if status is completed and completed_at is
present, do not touch it.* This means:
- `/deliver` re-runs on an already-completed spec are no-ops.
- `hero check --reconcile` or any other status fiddler that bounces a
  spec out of and back into `completed` does not change `completed_at`.
- The backfill command is rerunnable — already-stamped specs get
  skipped.

If a spec is *demoted* from `completed` (e.g. status flipped back to
`delivering` for a re-open), `completed_at` is left alone. This is
intentional: the original completion still happened, and if the spec
later re-completes, the historical value is more useful than an
overwrite. If the user wants to clear it, they can hand-edit.

### Field shape

- Canonical name: `completed_at` (snake_case, matching every other
  Hero-side timestamp: `claimed_at`, `handed_off_at`).
- Value: RFC 3339 with timezone, e.g. `2026-05-31T19:42:08Z`. UTC.
  Always written without fractional seconds for diff cleanliness.
- Reader compatibility: the parser also accepts `completedAt:`
  (camelCase) for tolerance with anyone hand-rolling the field; only
  `completed_at:` is ever written.

### Why `autoArchiveIfCompleted` is the safety net (not a hook on every writer)

The `/deliver` agent edits the spec markdown directly with its file
edit tool, then runs `hero spec verify <slug>` to trigger the auto-
archive. We do not control that markdown edit from Go. We *do* control
the verify step. Stamping there guarantees that no `completed` spec
ever lands in `.hero/specs/` without a `completed_at`, regardless of
who flipped the status.

The Go writers still get the in-place stamp because they're the path
of least latency — stamping at write time means the field is present
in the *same commit* as the status change, not "during the next verify
run." That matters for `hero deliver --async` flows that batch
commits.

## Changes

1. **`internal/spec/spec.go`** — add `CompletedAt time.Time` to the
   `Spec` struct (lines ~102–172 region with the other time fields).
   In `parseFrontmatter` (lines ~407–530 switch block) add a
   `completed_at` case (also accept `completedAt`):
   ```go
   case "completed_at", "completedAt":
       if t, err := time.Parse(time.RFC3339, val); err == nil {
           s.CompletedAt = t
       } else if t, err := time.Parse("2006-01-02", val); err == nil {
           s.CompletedAt = t
       }
   ```
   The YYYY-MM-DD fallback is read-only tolerance for anyone who
   hand-writes a date — never produced by Hero.

2. **`internal/spec/spec.go`** — add the canonical stamping helper:
   ```go
   // nowFn is overridable in tests.
   var nowFn = func() time.Time { return time.Now().UTC() }

   // StampCompletedAt sets completed_at to nowFn() in the spec content
   // unless the field is already present. Idempotent — returns the
   // content unchanged when completed_at exists.
   func StampCompletedAt(content string) string {
       if frontmatterHasField(content, "completed_at") ||
           frontmatterHasField(content, "completedAt") {
           return content
       }
       return SetFrontmatterField(content, "completed_at",
           nowFn().Format(time.RFC3339))
   }
   ```
   `frontmatterHasField` is a small private helper that walks the
   frontmatter range and returns true if any line trims to
   `<key>:` — cheap and avoids re-parsing.

3. **`internal/cli/complete.go`** — in `updateFrontmatterStatus`
   (line 224), when the new status is `"completed"`, also stamp:
   ```go
   updated := spec.SetFrontmatterField(content, "status", newStatus)
   if newStatus == "completed" {
       updated = spec.StampCompletedAt(updated)
   }
   ```

4. **`internal/cli/complete.go`** — in `autoArchiveIfCompleted`
   (line 195), after the early `s.Status != spec.StatusCompleted`
   guard and before the move/index, re-read the file, stamp if
   missing, write back:
   ```go
   data, err := os.ReadFile(specPath)
   if err == nil {
       stamped := spec.StampCompletedAt(string(data))
       if stamped != string(data) {
           _ = os.WriteFile(specPath, []byte(stamped), 0o644)
       }
   }
   ```
   Do this regardless of the `isAlreadyInSpecsDir` branch — both
   paths should end with a stamped file. Failures are best-effort
   logged to stderr (consistent with the rest of the function),
   not fatal.

5. **`internal/tracking/tracking.go`** — in
   `UpdateSpecFrontmatter` (line 125, the `case "complete":`),
   stamp on the same content buffer:
   ```go
   case "complete":
       content = spec.SetFrontmatterField(content, "status", "completed")
       content = spec.StampCompletedAt(content)
       content = removeFrontmatterField(content, "claimed_by")
       content = removeFrontmatterField(content, "claimed_at")
   ```

6. **`internal/cli/admin.go`** — register a new subverb:
   ```go
   adminCmd.AddCommand(backfillCompletedAtCmd)
   ```

7. **`internal/cli/admin_backfill_completed_at.go`** (new file) —
   one-shot backfiller. Behavior:
   - Walk all specs under `.hero/specs/` and `.hero/planning/`
     using `spec.Discover`.
   - Filter to `s.Status == spec.StatusCompleted` and
     `s.CompletedAt.IsZero()`.
   - For each, run `git log -1 --format=%aI -- <path>` from the
     project root.
   - On non-empty output, parse as RFC 3339, set
     `completed_at:` via `spec.SetFrontmatterField` (not
     `StampCompletedAt`, which would use `time.Now()`), and
     re-write.
   - Report: `Stamped: N, Skipped (already stamped): N, No git history: N`.
   - Supports `--dry-run` (preview without writing) and
     `--quiet` (suppress per-spec output, only print the summary).
   - Re-indexes once at the end via `index.Rebuild`.

8. **`internal/snapshot/render_json.go`** — the existing
   `CompletedAt string \`json:"completed_at"\`` field at line 47
   currently has no producer for spec-derived completion time;
   wire `spec.CompletedAt` into the JSON projection so dashboard
   consumers can read it without re-parsing the file. Verify the
   field is populated for shipped specs in
   `internal/snapshot/projector.go`.

9. **`internal/serve/pages/work/data/shipped.go`** — in
   `shippedFromSpecs` (lines 61–84), prefer
   `s.CompletedAt` over `s.ModifiedAt` for both the sort key and
   the `prettyAgeSince` call. Fall back to `ModifiedAt` only when
   `CompletedAt.IsZero()` (legacy specs pre-backfill).

10. **`domains/engineering/skills/spec-format/SKILL.md`** — add
    a short subsection under "General conventions" documenting
    the `completed_at:` field: name, format, who writes it (Hero,
    automatically — do not hand-write), reader tolerance for
    `completedAt:`. Cross-reference the backfill command.

11. **`domains/engineering/commands/deliver.md`** — at the
    section that describes flipping status to `completed`
    (around lines 248–253), add a one-liner note that
    `hero spec verify` stamps `completed_at:` automatically, so
    the agent does not need to add it when rewriting the
    frontmatter.

12. **Tests** — see Validation.

## Mockups

None. Backend-only change. The dashboard mockups already live in the
caller workspace.

## Boundaries

- **No retroactive overwrite.** Existing specs with a
  `completed_at:` value (whatever its provenance) are not touched
  by the writer paths or by backfill. The backfill is for
  *missing* values only.
- **No status-demotion handling.** If a spec gets bumped out of
  `completed`, the existing `completed_at` stays. We are not
  designing a "completion history" — only a most-recent timestamp.
- **No `completed_at` for non-work specs.** Conventions,
  decisions, rules, notes — these don't have a `delivering →
  completed` transition, and stamping them would just add noise.
  The writer paths already gate on the `complete` action being
  invoked on a work spec.
- **Not designing a cross-repo `completed_at` projection.** The
  cross-repo peering layer has its own handoff/handed-back states
  with their own timestamps (`handed_off_at`, `handed_back_at`).
  This spec only stamps the local spec's local completion.
- **Not touching the events log emitter.** `delivery_complete`
  events already carry their own timestamp from
  `feed.AppendEvent`. The dashboard's event-log path is
  unaffected.

## Risks

- **Race between Go writer and model rewrite.** If the model
  rewrites frontmatter and the Go writer also rewrites
  concurrently (e.g. `hero spec complete` invoked while an agent
  is mid-edit), one write overwrites the other. The race already
  exists for `status` itself — `completed_at` rides the same risk
  envelope without amplifying it. Document but do not solve.
- **`git log` returns empty for specs added in the working tree
  but never committed.** Backfill should treat empty git output
  as "no git history" and skip rather than stamping a fake time.
  Reported in the summary count.
- **Time-source drift in tests.** Several tests construct specs
  with hand-written frontmatter. They will now also produce a
  `completed_at:` line on transition. Test fixtures that diff
  full frontmatter need updating, and the `nowFn` override gives
  them a deterministic value.
- **Subsecond precision loss.** RFC 3339 without fractional
  seconds rounds to the nearest second. Acceptable for sprint
  dashboard week-bucketing; named as a known limitation for any
  future high-frequency telemetry consumer.

## Validation

### Unit tests

- `internal/spec/spec_test.go`
  - `TestStampCompletedAt_Idempotent` — calling twice does not
    change the value.
  - `TestStampCompletedAt_RespectsCamelCase` — content with
    `completedAt:` is left untouched.
  - `TestParseFrontmatter_CompletedAt_RFC3339` —
    `2026-05-31T19:42:08Z` parses into `s.CompletedAt`.
  - `TestParseFrontmatter_CompletedAt_DateOnly` —
    `2026-05-31` parses into `s.CompletedAt` at UTC midnight.
  - `TestParseFrontmatter_CompletedAt_CamelCase` —
    `completedAt:` is read into the same struct field.

- `internal/cli/complete_test.go`
  - `TestRunComplete_StampsCompletedAt` — fresh planning spec
    with `status: approved`, run `hero spec complete <path>`,
    assert resulting frontmatter contains both `status:
    completed` and a parseable `completed_at:`.
  - `TestRunComplete_IdempotentCompletedAt` — re-running on an
    already-completed-and-stamped spec preserves the original
    timestamp.
  - `TestAutoArchiveIfCompleted_StampsWhenMissing` — write a
    spec with `status: completed` and no `completed_at:`,
    invoke `autoArchiveIfCompleted`, assert the stamp landed.
  - `TestAutoArchiveIfCompleted_LeavesExistingStamp` — pre-stamp
    the spec, assert auto-archive does not bump.

- `internal/tracking/tracking_test.go` (or wherever
  `UpdateSpecFrontmatter` is tested)
  - `TestUpdateSpecFrontmatter_CompleteStampsTimestamp` —
    action `complete` writes both `status: completed` and
    `completed_at:`, and removes `claimed_by` / `claimed_at`.

- `internal/cli/admin_backfill_completed_at_test.go` (new)
  - `TestBackfill_HappyPath` — git repo with one committed
    completed spec missing the field; backfill stamps it from
    `git log`.
  - `TestBackfill_SkipsAlreadyStamped` — pre-stamped spec is
    skipped and counted under "skipped."
  - `TestBackfill_SkipsNoGitHistory` — uncommitted spec is
    counted under "no git history," not stamped.
  - `TestBackfill_DryRun` — `--dry-run` produces the same
    counts but writes no files.

### Manual verification

1. `git checkout` a fresh branch.
2. Create a planning spec with `hero spec scaffold`.
3. Move it through `hero spec deliver --manual <slug>` and then
   `hero spec verify <slug>` (the canonical local-deliver path).
4. Open the resulting `.hero/specs/<slug>/spec.md`; confirm both
   `status: completed` and `completed_at: <recent ISO 8601 UTC>`.
5. Run `hero spec verify <slug>` a second time; confirm
   `completed_at:` does not change.
6. Pick an old completed spec in `.hero/specs/` that predates
   this change. Confirm `completed_at:` is absent. Run
   `hero admin backfill-completed-at --dry-run`; confirm the
   spec appears in the preview. Run without `--dry-run`; confirm
   the field is stamped with the git committed-at time, not
   `time.Now()`.
7. Confirm with `git log -1 --format=%aI -- .hero/specs/<slug>/spec.md`
   that the stamped value matches.

### Cross-repo verification (with caller)

Once shipped, ping `hero-code` (the calling workspace) to:
- Update the desktop Sprint Dashboard's resolver to drop the
  git-log fallback tier and treat `completed_at:` as
  authoritative.
- Confirm the velocity sparkline buckets resolve correctly for
  a mix of newly-completed (stamped at write time) and
  backfilled-historical (stamped at `git log` time) specs.

## Kickoff

Delivered. 12/12 Changes items DONE, 14/14 spec-mandated tests green,
plus 3 extra tests covering a fourth writer site the auditor surfaced
(`internal/serve/refresh.go:199` — tracker auto-resolve to "completed"
now also stamps in the same write). Cold audit: SHIP / noteworthy /
high confidence — the noteworthy bullet was the refresh.go gap, fixed
inline.

**Status:** completed. Ready to hand back to `hero-code` so they can
retire the desktop Sprint Dashboard's git-log fallback tier.

**Pick up at:** nothing on this spec. Once shipped, `hero handoff
<spec> hero-code` returns the work and signals they can drop the
`git log -1 --format=%aI` resolver.

**Files shipped:**
- `internal/spec/spec.go` — `CompletedAt` field, `nowFn`,
  `StampCompletedAt`, `SwapNowFnForTest`, `frontmatterHasField`,
  parser case for both `completed_at` and `completedAt`.
- `internal/cli/complete.go` — `updateFrontmatterStatus` and
  `autoArchiveIfCompleted` (both branches) stamp.
- `internal/tracking/tracking.go` — `case "complete"` stamps.
- `internal/serve/refresh.go` — `updateSpecFrontmatterField`
  stamps when key=status / value=completed (audit-driven fix).
- `internal/cli/admin_backfill_completed_at.go` — new subcommand
  `hero admin backfill-completed-at` (--dry-run, --quiet, never
  fabricates a timestamp on empty `git log` output).
- `internal/snapshot/rollup.go` + `internal/serve/pages/work/data/shipped.go`
  — readers prefer `CompletedAt`, fall back to `ModifiedAt` for legacy.
- `domains/engineering/skills/spec-format/SKILL.md` + `commands/deliver.md`
  — docs.

**Backfill ready on this repo:** 165 historical completed specs
under `.hero/specs/` are missing the field. Run
`hero admin backfill-completed-at` to stamp them from `git log`
in one pass.

**Acceptance bar — held:** every Go writer that flips status to
completed now stamps in the same write (verified post-audit at four
writer sites: `complete.go`, `tracking.go`, the auto-archive safety
net, and `refresh.go`). Model-driven `/deliver` is covered by the
auto-archive post-condition. Idempotent on re-runs (proven via clock-
advance test). Reader-tolerant of `completedAt` camelCase.
## Handoff Trail

- 2026-06-01T03:42:53Z — out → hero-code (peer_id: cd8dd06d-3df1-4878-a88f-24593dcbb4b3)
  mode: advisory
  originating_spec: deliver-stamps-completed-at
  at_commit: 9ea7bf2
  result_ref: .hero/peer-calls/18b4d80d3fadb640c7ec6e9c1ea5d992.md
  reason: "Notify peer that the spec they spec-out'd is delivered so they can retire the desktop Sprint Dashboard's git-log fallback."

- 2026-06-01T03:56:53Z — out → hero-code (peer_id: cd8dd06d-3df1-4878-a88f-24593dcbb4b3)
  mode: advisory
  originating_spec: deliver-stamps-completed-at
  at_commit: 9c7fb9b
  result_ref: .hero/peer-calls/18b4d8da1254fb90233fc02645fc71df.md
  reason: "Answer the two non-blocking technical questions from the prior ack so the desktop team has full context before they cut to v0.14.7."

