---
title: "events.log churn fix — keep it tracked, stop it dirtying the tree, let interesting events travel into commits"
slug: eventslog-churn-fix-tracked-not-dirty
type: enhancement
status: completed
size: small
priority: medium
domain: engineering
tags: [events-log, hooks, git, churn, handoff, merge-driver, dx]
created: 2026-07-09
relates-to: [cst-gitignore-events-log]
completed_at: 2026-07-10T03:36:57Z
---

# events.log churn fix — keep it tracked, stop it dirtying the tree, let interesting events travel into commits

## Context

`.hero/events.log` is a tracked, append-only JSONL activity log. It has one
persistent annoyance: **the working tree is dirty the instant every commit
finishes.** The post-commit git hook appends an `{event: post-commit, sha}`
marker to `.hero/events.log` right after the commit lands, so `git status`
immediately reports the file modified — a fresh diff created by the very act
of committing.

A prior spec, **`cst-gitignore-events-log`**, looked at this churn and resolved
to "leave it tracked, by design" — i.e. it accepted the churn rather than
fixing it. That resolution is being **superseded** here: events.log stays
tracked (that decision holds and is reaffirmed as a hard boundary below), but
the churn itself is now fixable with two small, surgical changes. This spec
does the churn fix that the earlier spec declined to do. It does **not**
reopen the gitignore question — gitignoring events.log is off the table.

Two distinct problems, two prongs:

1. **The commit itself dirties events.log.** A write-only `post-commit` marker
   is appended after every commit. Nothing reads it. Its only observable
   effect is the post-commit dirty tree.
2. **Session-accrued events don't travel with commits, and aren't merge-safe.**
   Interesting events (claims, completions, delivery start/complete, decisions)
   accrue in events.log during work but are not auto-staged, so they lag behind
   the code they describe. And because events.log is not declared `merge=union`,
   concurrent appends from two branches collide as textual conflicts (we
   hand-resolved one such conflict earlier this session).

Together the two prongs realize the intent: **interesting activity goes into
the commit; events.log is not left dirty afterward.**

## Goal

After a `git commit` completes, `.hero/events.log` is **clean** — the commit
no longer appends anything to it. Session-accrued events (claims, completions,
etc.) are **auto-staged by the pre-commit hook** so they travel with the next
commit, exactly the way `.hero/NEXT.md` and `.hero/SNAPSHOT.md` already do, and
events.log is declared **`merge=union`** so concurrent appends auto-resolve
instead of conflicting. events.log **remains tracked** — no gitignore. No
consumer of events.log loses data, because nothing reads the removed
`post-commit` marker.

## Kickoff

Stops `.hero/events.log` from being dirty right after every commit, and makes
its interesting events auto-travel into commits — while keeping it tracked (no
gitignore).

**Status:** planning — spec just landed, no code yet. Two surgical prongs.

**Pick up at:** Prong 1 — in `internal/cli/hook.go` `case "post-commit":`
delete only the `hooks.LogEvent(...post-commit...)` append (lines 96-99),
keep `writeCheckpoint()`. Prong 2 — add `".hero/events.log"` to the
`handoffFilePaths` slice in `internal/cli/next_hooks.go:43`. Then update the
two tests that assert exact pathspec/gitattributes strings.

→ `.hero/planning/features/eventslog-churn-fix-tracked-not-dirty/spec.md`

**Files:** `internal/cli/hook.go:94-104`, `internal/cli/next_hooks.go:43`, `internal/cli/next_hooks_test.go`, `internal/cli/hooks_staging_integration_test.go`
**Skip:** gitignoring events.log — decided off the table; this spec supersedes `cst-gitignore-events-log`'s "leave it, by design" resolution with an actual churn fix.

## Approach

The fix is two independent, small edits plus their test updates. Both prongs
lean on structure that already exists — a single-source slice that drives both
the auto-stage list and the merge-driver block — so the changes are minimal and
can't drift.

**Prong 1 — remove the write-only post-commit event.** In `hook.go`'s
`case "post-commit":` block (lines 94-104), the hook does two things:
(a) appends `{event: post-commit, sha}` to events.log, and (b) calls
`writeCheckpoint()` to refresh NEXT.md's machine block. Only (a) causes the
churn. Delete only the `hooks.LogEvent(...)` append (lines 96-99, plus the now
unused `sha := gitRevParse("HEAD")` on line 95). **Keep `writeCheckpoint()`
(line 104) exactly as-is** — it is the cross-session handoff refresh and is
unrelated to the events.log churn.

The one honest behavioral change: `post-commit` events stop being recorded.
This is a pure churn removal with **no feature loss**, because no consumer reads
`post-commit` events. Verified this session: feed / velocity / metrics / since
all key off named event types (`claimed`, `completed`, `delivery_start`,
`delivery_complete`, `decision_made`, handoff types) —
`internal/serve/pages/now/data/since.go` uses `countEventsSince` with explicit
type filters, and nothing reads "most-recent-event-of-any-type" as a heartbeat.
The commit SHA the marker recorded is already in git. So the append is a
write-only marker whose only effect was dirtying the tree.

**Prong 2 — add `.hero/events.log` to `handoffFilePaths`.** The slice at
`next_hooks.go:43` is the documented single source of truth (see the comment at
lines 27-42) that drives BOTH:
- the pre-commit hook's per-path `git add` staging loop (via `hookScript()`,
  which joins `handoffFilePaths` into the staging loop at lines 327-333), and
- the managed `.gitattributes` `merge=union` block (via `updateGitAttributes()`,
  which iterates `handoffFilePaths` at lines 613-616).

Adding one line — `".hero/events.log"` — to the slice makes events.log
auto-staged AND `merge=union` in one edit. Both effects are intentional and are
exactly why the slice is the right (and only) edit point: they can't drift
because they're derived from the same source.

**Merge safety.** events.log is append-only, timestamped JSONL, and every
consumer sorts by timestamp on read. `merge=union` concatenates both sides of a
conflict; the possible out-of-order concatenation is harmless because readers
re-sort. (The events.log conflict we hand-resolved earlier this session would
have auto-resolved under union.)

**Slice ordering.** The comment at lines 34-35 says order is significant only
for stable test output. Place `".hero/events.log"` at the **end** of the slice
(after `.hero/peer-manifest.yaml`) — it groups the newest addition last and
keeps the existing entries' positions stable, minimizing test churn. Update the
assertions accordingly.

**Desired residual behavior (not a bug).** Even with both prongs, a `hero`
command that appends an event mid-session (e.g. `hero spec claim`) will
correctly dirty events.log until the next commit sweeps it in. That is
uncommitted activity, which *should* show as a pending change — it is not churn.
The churn we are eliminating is the one created by the commit itself.

## Changes

1. **`internal/cli/hook.go` — remove the post-commit events.log append**
   (`case "post-commit":`, lines 94-104).
   - Delete the `hooks.LogEvent(eventsLogPath, map[string]string{"event":
     "post-commit", "sha": sha})` call (lines 96-99).
   - Delete the now-unused `sha := gitRevParse("HEAD")` (line 95). Confirm
     `gitRevParse` is still referenced elsewhere before assuming any import/helper
     cleanup — if it is unused after this edit, that is out of scope to remove
     unless it produces a compile error (Go will flag an unused local, not an
     unused func).
   - **Keep `_, _ = writeCheckpoint()` (line 104)** and its comment. State in
     the commit/PR that the NEXT.md checkpoint refresh is deliberately preserved.
   - Result: after a commit, nothing appends to events.log — the file is clean.

2. **`internal/cli/next_hooks.go` — add events.log to the single-source slice**
   (`handoffFilePaths`, lines 43-49).
   - Add `".hero/events.log",` as the final entry (after
     `".hero/peer-manifest.yaml"`).
   - This automatically (a) auto-stages events.log in the pre-commit hook and
     (b) declares it `merge=union` in the managed `.gitattributes` block. No
     other edit is needed for either effect.
   - Optionally extend the slice's doc comment (lines 27-42) with a one-line note
     that events.log is append-only JSONL and union-safe because readers sort by
     timestamp — so a future maintainer understands why it belongs here.

3. **`internal/cli/next_hooks_test.go` — update pathspec / gitattributes assertions.**
   - Any test asserting the exact `handoffFilePaths` contents, the joined
     `git add` pathspec string in `hookScript("pre-commit")`, or the
     `updateGitAttributes` managed-block output must now include
     `.hero/events.log` (at the end, matching the chosen slice position).
   - Grep for the existing pathspec strings (`.hero/peer-manifest.yaml`,
     `merge=union`, `.hero/SNAPSHOT.md`) to locate every assertion that needs the
     new line.

4. **`internal/cli/hooks_staging_integration_test.go` — update the staging integration expectations.**
   - The integration test that exercises the pre-commit staging loop must expect
     `.hero/events.log` to be staged alongside the other projected handoff files.
   - If a `post-commit`-marker assertion exists anywhere in the hook tests (an
     assertion that committing appends a `post-commit` event), remove/adjust it —
     Prong 1 deletes that behavior.

5. **Run the full `internal/cli` package test suite** and fix any remaining
   assertions that referenced the removed `post-commit` event or the old
   pathspec/gitattributes strings.

## Boundaries

- **events.log STAYS TRACKED. Gitignoring is off the table** — decided and
  recorded this session; this spec supersedes `cst-gitignore-events-log`'s
  "leave it, by design" resolution with an actual churn fix. Do **not** propose,
  add, or suggest a `.gitignore` entry for events.log.
- **No change to the tracking writer** (`internal/tracking/tracking.go`
  `AppendEvent`) or to what events.log contains, beyond removing the
  `post-commit` marker append in the hook.
- **Do not touch the other hook cases** — `post-checkout`, `post-merge`,
  `prepare-commit-msg`, and the `pre-commit` status-truth gate stay exactly as
  they are. Only the events.log append inside `case "post-commit":` is removed.
- **Do not touch `writeCheckpoint()`** or its post-commit invocation.
- **Do not widen `handoffFilePaths`** beyond the single `.hero/events.log`
  addition.
- `writeCheckpoint()` re-dirtying `.hero/NEXT.md` after commit (if it does) is a
  **separate, pre-existing pattern** for a projection file that is *already*
  auto-staged — explicitly out of scope here. Mentioned only as a non-goal.

## Risks

- **Migration / propagation nuance (the important one).** The change touches two
  surfaces with different propagation, both derived from `handoffFilePaths`:
  1. **`.gitattributes` `merge=union` line** — the `.gitattributes` file is
     *committed* and `union` is a git built-in (no per-clone `.git/config`
     registration). Once *any* clone regenerates `.gitattributes` (via upgrade /
     scan / install-hooks) and commits it, the events.log `merge=union` line
     propagates to every clone on its next pull. **No per-clone action** is
     needed for the merge behavior once the regenerated `.gitattributes` is
     committed.
  2. **Pre-commit auto-staging of events.log** — this lives in the
     *uncommitted, per-clone* `.git/hooks/pre-commit` script. It does **not**
     travel with commits. Each clone's local hook only starts staging events.log
     after that clone refreshes its hook (see self-heal answer below).
- **Removing `post-commit` events is irreversible for historical analytics.**
  If any *future* consumer wanted a per-commit heartbeat, it's gone. Verified no
  *current* consumer reads it, so this is acceptable — but note it so a future
  reader doesn't reintroduce a reader expecting the marker.
- **Test brittleness.** The two named test files assert exact pathspec and
  gitattributes strings by design (the slice comment calls this out). Missing one
  assertion will fail the suite — that's the safety net working. Grep
  exhaustively for the old strings.
- **Low blast radius overall.** Four deleted lines in one hook case, one added
  slice entry, and test updates. No schema, no data migration, no API surface.

## Validation

**Does the merge-driver line self-heal on `hero upgrade`, or does an existing
repo need a manual `hero next install-hooks`?**

**Yes — it self-heals on `hero upgrade`, for any repo that already has the Hero
managed hook block installed.** Traced this session:
- `hookScript("pre-commit")` embeds `strings.Join(handoffFilePaths, " ")` in its
  staging loop (`next_hooks.go:327-333`). Adding events.log changes the hook
  script's content.
- `preCommitHookStale()` (`next_hooks.go:201-207`) compares the *installed*
  managed block against `hookScript("pre-commit")`. Because the pathspec now
  differs, the installed block is detected **stale**.
- `hero upgrade` (without `--no-hooks`) calls `refreshHooksIfPresent()`
  (`upgrade.go:232-236`), which — when a managed block is present and stale —
  runs `installNextHooksQuiet()` (`next_hooks.go:250-267`). That rewrites the
  pre-commit hook **and** calls `updateGitAttributes()`, so the events.log
  `merge=union` line lands and the local hook starts staging events.log — **with
  no manual step.**

Two caveats to state honestly:
1. `hero upgrade --no-hooks` skips the refresh (by design).
2. Repos with **no** managed block at all (never installed, or the user removed
   it) are intentionally *not* touched by `upgrade` — `refreshHooksIfPresent`
   respects explicit removal; `hero scan` / `hero init` / `hero next
   install-hooks` are the install paths. Those repos need one
   `hero next install-hooks` (or `hero scan`) to opt in — but they also had **no
   auto-staging to begin with**, so nothing regresses; they simply don't gain the
   new staging until they install.

Net: existing installed repos get both effects automatically on their next
`hero upgrade`; the committed `.gitattributes` then propagates the merge=union
line to teammates on pull, while each teammate's local pre-commit staging
activates on their own upgrade/reinstall.

**Verification steps:**

1. **Prong 1 — clean tree after commit.** In a repo with the managed hook
   installed, make and commit any change; run `git status --porcelain`. Assert
   `.hero/events.log` is **not** listed as modified. (Before this fix it would
   be dirty immediately.)
2. **Prong 2 — auto-stage.** Append an event to events.log mid-session (e.g. via
   a `hero` command that logs a `claimed`/`completed` event), then commit an
   unrelated change. Assert the committed tree includes the events.log update —
   i.e. it was swept in by the pre-commit staging loop.
3. **Prong 2 — gitattributes.** After `hero next install-hooks` (or `hero
   upgrade` on an installed repo), assert `.gitattributes` contains
   `.hero/events.log merge=union` inside the managed marker block.
4. **Merge safety.** Create two branches that each append a distinct event to
   events.log; merge one into the other. Assert the merge auto-resolves (union
   concatenation) with no conflict markers, and that a reader still sees both
   events after timestamp sort.
5. **No consumer regression.** Run whatever tests cover feed / velocity /
   metrics / since; confirm none depended on `post-commit` events.
6. **Full `internal/cli` suite passes**, including the updated
   `next_hooks_test.go` and `hooks_staging_integration_test.go`.

## Acceptance Criteria

- WHEN a git commit completes THE SYSTEM SHALL NOT append a `post-commit` event
  to `.hero/events.log` (the commit itself does not dirty the file).
- THE SYSTEM SHALL preserve the post-commit `writeCheckpoint()` NEXT.md refresh —
  only the events.log append is removed.
- WHEN the pre-commit hook runs THE SYSTEM SHALL stage `.hero/events.log`
  alongside the other projected handoff files so session-accrued events travel
  with the commit.
- THE SYSTEM SHALL declare `.hero/events.log` as `merge=union` in the managed
  `.gitattributes` block, derived from the same single-source `handoffFilePaths`
  slice, so concurrent appends auto-resolve.
- `.hero/events.log` SHALL remain tracked (NOT gitignored).
- THE SYSTEM SHALL keep the `handoffFilePaths`-derived git-add pathspecs and the
  `.gitattributes` block in sync from the single source, with the tests asserting
  the exact strings updated to include `.hero/events.log`.

## Completion Ledger

Delivered 2026-07-09. `go build ./...`, `go vet ./internal/cli/...`, and
`go test ./...` (86 packages) green. Exercised end-to-end in a throwaway `/tmp`
git repo (this repo's real hooks untouched): after `hero next install-hooks` +
commit, `git status --porcelain` does NOT list `.hero/events.log` (churn gone),
and `.gitattributes` carries `.hero/events.log merge=union`. Cold audit: see
`delivery-audit.md`.

### Acceptance Criteria

| Criterion | Status | Evidence |
|---|---|---|
| Commit no longer appends post-commit event to events.log | DONE | `hook.go` `case "post-commit":` append removed; `TestPostCommitHook_DoesNotAppendEventsLog`; scratch-repo `git status` clean |
| `writeCheckpoint()` preserved | DONE | `hook.go:92-97` — checkpoint call + comment unchanged |
| Pre-commit stages events.log with the handoff files | DONE | `.hero/events.log` in `handoffFilePaths` → staging loop; `TestIntegration_DefaultInstall_StagesHandoffFiles` |
| events.log declared `merge=union` (single-source) | DONE | derived via `updateGitAttributes`; `TestUpdateGitAttributes_BindsAllFourPathsToUnion`; scratch `.gitattributes` line present |
| events.log stays tracked (no gitignore) | DONE | no `.gitignore` change (diff confirms); boundary honored |
| pathspec + gitattributes stay in sync from single source, tests updated | DONE | `TestHandoffFileList_SingleSourceOfTruth` + updated assertions in `next_hooks_test.go`, `hooks_staging_integration_test.go` |

### Changes

| # | Change | Status | Evidence |
|---|---|---|---|
| 1 | `hook.go` — remove post-commit events.log append, keep writeCheckpoint | DONE | 5 lines deleted (`sha` + `LogEvent`); checkpoint preserved |
| 2 | `next_hooks.go` — add `.hero/events.log` (final) to `handoffFilePaths` + doc note | DONE | `next_hooks.go:43-56` |
| 3 | `next_hooks_test.go` — pathspec/gitattributes assertions | DONE | union-directive list + single-source test cover new entry |
| 4 | `hooks_staging_integration_test.go` — staging expectations + Prong-1 guard | DONE | `TestPostCommitHook_DoesNotAppendEventsLog`, default-install staging assert |
| 5 | Full `internal/cli` suite green | DONE | `go test ./internal/cli/...` ok; `go test ./...` 86 pkg ok |

### Exercise-the-feature check

- [x] Real commit in a scratch `/tmp` repo: `hero next install-hooks` → stage → commit.
  Post-commit `git status --porcelain` output did NOT include `.hero/events.log`
  (churn eliminated); pre-commit staging loop included `.hero/events.log`;
  `.gitattributes` managed block contained `.hero/events.log merge=union`. This
  repo's real `.git/hooks`/`.gitattributes` confirmed unchanged afterward.
- Deviation: no dedicated two-branch merge test — `merge=union` is a git built-in
  and the directive emission is test-covered; judged low-ROI per the surgical
  boundary.
