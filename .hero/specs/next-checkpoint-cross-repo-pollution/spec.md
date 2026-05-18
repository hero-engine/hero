---
title: "hero next checkpoint accumulates cross-repo narrative sections in .local.md and bleeds user-graph reads across repos"
slug: next-checkpoint-cross-repo-pollution
type: bug
status: completed
severity: high
priority: P1
created: 2026-05-18
tags: [next, checkpoint, handoff, projection, cross-repo, layer-2, peer-call]
received_from:
  peer_id: ad027c2f-7f74-4a09-bf1d-6515cc906074
  peer_alias_display: hero-code
  originator_slug: hero-context-layer2-optimize
  handed_off_at: 2026-05-18T13:57:54Z
  at_commit: "6835176"
  call_id: 18b0ad8d3cd1c3c0adf6bdd5cad5a716
  mode: spec-out
  reason: "Discovered while optimizing hero-code's Layer 2 — the per-user NEXT file the engine writes has drifted with stale concatenated sections from unrelated repos, polluting any consumer's eager context."
relations:
  - target: hero-context-layer2-optimize
    kind: surfaced-by
---

# hero next checkpoint accumulates cross-repo narrative sections in .local.md and bleeds user-graph reads across repos

## Problem

The next-handoff pipeline is producing per-user files (`.hero/next/<user>.local.md`) with **multiple** `## Just finished` and `## Next` sections, each describing work from a different repo. Observed in hero-code's `.hero/next/chet-bellows.local.md` (call_id `18b0ad8…`):

- Section 1 (`## Just finished`): ROCm/burn/paperboy engine work — from a *different* repo (paperboy-engines/paperboy-rocm).
- Section 2 (`## Just finished`): pluggable-terminal-backends / alacritty_terminal work — from a third repo.
- Two `## Next` sections similarly stratified by repo.
- A `## Blocked on (omit if clear)` placeholder header even when no blocker applies — the skill spec (`skills/next-md/SKILL.md`) says to omit when clear.
- The marker-bounded machine state block at the top of the file is correct (commits and dirty tree match hero-code's repo).

Per `skills/next-handoff-emit/SKILL.md` and `skills/next-md/SKILL.md`, the file should carry **exactly one** of each narrative section — scoped to the writing repo and **replaced** on every checkpoint, not appended.

The bug pollutes Layer 2 (project context) for any consumer that reads the per-user next file. hero-code's `hero-context-layer2-optimize` work is about to lean on this file as a primary context source. Fixing it here unblocks that spec.

## Investigation

### Code trace

The relevant pipeline lives in `internal/cli/checkpoint.go`. `writeCheckpoint` produces two files each turn:

1. The durable next file (`.hero/NEXT.md` in solo mode, `.hero/next/<user>.md` in team mode).
2. The local machine-state file (`.hero/next/<user>.local.md`, always gitignored).

`rebuildLocalState(existing, machineBlock)` at `internal/cli/checkpoint.go:291-300` is the writer for the local file:

```go
func rebuildLocalState(existing, machineBlock string) string {
    if strings.TrimSpace(existing) == "" {
        return machineBlock + "\n"
    }
    hand := strings.TrimSpace(stripMachineBlock(existing))
    if hand == "" {
        return machineBlock + "\n"
    }
    return machineBlock + "\n\n" + hand + "\n"
}
```

It **preserves verbatim** any content outside the `<!-- BEGIN HERO MACHINE STATE --> … <!-- END HERO MACHINE STATE -->` markers — by design. The doc comment on `nextCheckpointCmd` (`internal/cli/checkpoint.go:46-48`) makes this explicit: *"Hand-written content in <user>.local.md outside the marker block is preserved across regens — drop reminders, scratch notes, anything ad-hoc into that file and it survives every checkpoint."*

That preservation has no upper bound, no de-duplication, no scoping check, and no provenance tracking. Any process — an older hero binary that wrote agent narrative there, a misbehaving agent following stale guidance, a cross-machine sync, or an editor save — that drops `## Just finished` / `## Next` content outside the markers leaves it in the file forever. Two checkpoints from two different working directories that landed content outside the markers produce the exact accumulated-from-multiple-repos shape observed.

### Repo-scope review (separate but adjacent issue)

The reporter also flagged "no repo-scoping filter on the emit source." Confirmed in `internal/handoff/handoff.go` and `internal/graph/node.go`:

- `LatestAsk(store, user)` (`handoff.go:172`) calls `store.GetNode(NodeUserAsk, user)` → `internal/graph/node.go:177-186` queries by `(type, key)` only, with no repo filter.
- `LatestSuggestion(store, user)` (`handoff.go:185`) has the same shape.
- `RecentReflections(store, user, limit)` (`handoff.go:198`) calls `store.ListNodesByType(NodeSessionReflection)` (`internal/graph/node.go:223-258`) which lists all current rows of the type regardless of repo, then filters only on the `user:` key prefix.

`UpsertNode` (`internal/graph/node.go:43-151`) treats `repo` as part of the partition key — when `existingRepo != n.Repo`, it invalidates and re-inserts. So for the singletons `UserAsk` and `NextSuggestion` (keyed by `user` alone), `hero next ask "X"` in repo A and then `hero next ask "Y"` in repo B leaves repo A's row invalidated and only repo B's current — i.e. the singleton is **global across repos**, last-write-wins. For `SessionReflection` (multi-row), all reflections accumulate and `RecentReflections` returns them regardless of which repo recorded them.

These reads feed `projection.UserHandoffMD` (`internal/projection/user_handoff.go:30-136`), which renders the durable `.hero/next/<user>.md` file. So in solo mode (where `writeUserHandoffFile` actually runs — `checkpoint.go:183-186` no-ops in team mode), an ask/suggestion recorded in repo A appears in repo B's `<user>.md`. This is *not* what produces the symptom in `.local.md` (that file isn't graph-driven), but it is a real second leak that the hero-context-layer2-optimize work will hit when it reads `<user>.md`.

### Why the placeholder `## Blocked on (omit if clear)` header sticks

`nextPlaceholder()` in `internal/cli/checkpoint.go:531-555` always emits all six section headers verbatim including `## Blocked on (omit if clear)` as a template the agent is expected to delete-when-clear. The placeholder writes to `nextPath` (NEXT.md or `<user>.md`), not to `.local.md`. The header appearing in the reporter's `.local.md` means either (a) `nextPlaceholder`-shaped content was written into `.local.md` by some external process and then preserved by `rebuildLocalState`, or (b) the same file is being shared between paths somewhere. Either reading reinforces the primary fix: `.local.md` must stop preserving narrative content.

### Root cause

**Primary** — `.hero/next/<user>.local.md` accumulates whatever lands outside its marker block, with no bound or scoping. The "preserve hand-written content" contract baked into `rebuildLocalState` is the load-bearing source of the cross-repo concatenation symptom: once `## Just finished` text reaches this file by any means, the checkpoint command rewrites it back on every run.

**Secondary** — `LatestAsk` / `LatestSuggestion` / `RecentReflections` read user-graph singletons and lists without filtering on `repo`. `UpsertNode`'s partition-key semantics mean the singletons are effectively *global per user*; reflections leak across repos when read. The durable `<user>.md` projection inherits the leak.

### Severity

High. The bug:

- Pollutes the per-user handoff with stale, mis-scoped narrative that misleads any fresh session reading it as load-bearing project context.
- Will land squarely in front of hero-code's `hero-context-layer2-optimize` work, which is preparing to use the per-user handoff as a primary Layer-2 source. Cannot ship that confidently while the artifact is corrupt.
- No workaround for users beyond manually deleting `.local.md` after each session — which defeats the purpose.
- Caused by our code (the `rebuildLocalState` preservation contract and the missing repo filter on handoff reads), not by external systems.

## Goal

`hero next checkpoint` produces a per-user `.local.md` containing **only** the marker-bounded machine-state block — no preserved narrative sections, no accumulation, replaced wholesale on each run. The user-graph reads that drive the durable `<user>.md` projection filter by the current repo so an ask or suggestion recorded in repo A never appears in repo B's handoff. A one-time cleanup step backs up any non-empty pre-existing `.local.md` hand-content before discarding it.

## Approach

Two coordinated changes, plus a cleanup migration:

1. **Make `.local.md` machine-state-only.** Drop the preservation behavior from `rebuildLocalState`. The doc comment that promised scratch preservation is replaced with a `### Scratch` section *inside* the marker block (rebuilt every turn from a dedicated graph source if needed) or, simpler for v1, removed entirely with a pointer to a separately-named scratch file users can hand-edit. The contract becomes: "this file is rebuilt every checkpoint; do not hand-edit, anything you write here will be lost."

2. **Repo-scope user-graph reads.** Add a `repoKey` parameter to `LatestAsk`, `LatestSuggestion`, and `RecentReflections`, and add `AND repo = ?` to the corresponding SQL. Update callers (`projection/user_handoff.go`, `internal/cli/next_handoff.go`, `internal/handoff/ingest.go`, `internal/cli/next_migrate.go`). Existing tests cover the (user,) case; new tests cover the (user, repo) case.

   Optional but recommended for the singletons: also widen the upsert key so the per-repo singleton survives concurrent emission from multiple repos rather than supersede-clobbering. Concretely, change the singleton key for `UserAsk` and `NextSuggestion` from `user` to `user + ":" + repoKey`. This requires a one-shot migration of existing rows (re-key by current repo column) and updates everywhere the key shape is assumed.

3. **One-time cleanup of stale `.local.md` accumulation.** On the first checkpoint after this fix lands, detect any non-trivial hand-content outside the marker block. If present, write it once to `.hero/next/<user>.local.md.bak.<timestamp>` (also gitignored) before the rebuild discards it. Print a one-line notice on stderr pointing the user at the backup so accidental data loss is recoverable for one cycle.

## Changes

1. `internal/cli/checkpoint.go` — replace `rebuildLocalState`'s preservation behavior with a wholesale rewrite.
   - Remove the `hand := strings.TrimSpace(stripMachineBlock(existing))` preservation branch; the function returns `machineBlock + "\n"` unconditionally.
   - Before discarding the existing content, if `existing` has non-empty content outside the marker block, write that content once to `<localPath>.bak.<UTC timestamp>` and emit a one-line stderr notice with the backup path. Idempotent: subsequent checkpoints (with no further hand-content) are a no-op for backup.
   - Update the `nextCheckpointCmd.Long` doc comment to drop the "scratch notes preserved across regens" promise and replace it with the new contract: "`.local.md` is fully rebuilt every checkpoint; do not hand-edit."
   - Add tests in `internal/cli/checkpoint_test.go`: `Test_rebuildLocalState_DiscardsHandContent`, `Test_writeCheckpoint_BacksUpPreExistingHandContent`, `Test_writeCheckpoint_NoBackupWhenAlreadyClean`.

2. `internal/handoff/handoff.go` — repo-scope the read API.
   - Change signatures to `LatestAsk(store *graph.Store, user, repoKey string)`, `LatestSuggestion(store *graph.Store, user, repoKey string)`, `RecentReflections(store *graph.Store, user, repoKey string, limit int)`.
   - For singletons, query directly via `store.DB()` with `WHERE type = ? AND key = ? AND repo = ? AND valid_to IS NULL`. (Keeping the singleton key shape unchanged for v1; the partition filter alone is enough to fix the read-side bleed since `UpsertNode` already invalidates the prior repo's row.)
   - For reflections, switch off `ListNodesByType` to a direct query that filters by both the `user:` key prefix and `repo = ?`.
   - Update callers in the same PR:
     - `internal/projection/user_handoff.go` — pass `opts.RepoKey` to all three handoff reads.
     - `internal/cli/next_handoff.go` — `runNextSuggest`, `runNextAsk`, `runNextReflection` already have `repoKey` from `openHandoffStore`; thread it through.
     - `internal/handoff/ingest.go` — `IngestUserFile` already has `repoKey`; thread it through to `RecentReflections` (dedup against this repo's reflections only).
     - `internal/cli/next_migrate.go` — `runNextMigrateProjection` already has `repoKey`; thread it through to `extractAskAndSuggestion`'s downstream reads if any (none today, but the snapshot capture should remain repo-stamped).
   - Update tests in `internal/handoff/handoff_test.go` and `internal/handoff/ingest_test.go` for the new signatures; add `TestRecordAsk_PerRepoIsolation`, `TestRecordSuggestion_PerRepoIsolation`, `TestRecentReflections_PerRepoIsolation` covering the cross-repo bleed regression.

3. `internal/projection/user_handoff.go` — confirm the projection consults `opts.RepoKey` for handoff reads (it already does for commits and staleness; this change brings asks/suggestions/reflections into the same partition discipline). Add a test in `internal/projection/user_handoff_test.go` (or create that file): `TestUserHandoffMD_DoesNotLeakCrossRepoAsk`.

4. `internal/cli/checkpoint_test.go` — add an end-to-end test that records `hero next ask "A"` against repo A, opens a second store with `repoKey=B`, runs `writeCheckpoint` and asserts the rendered `<user>.md` does **not** contain "A" in the ask section.

5. `skills/next-md/SKILL.md` — update §"What I do" and §"What not to do" to state explicitly that `.local.md` is machine-state-only, never hand-edited, never used for scratch. Point users who want preserved per-machine notes at a clearly-named hand-only file (e.g. `.hero/notes/<user>.md`) that no automated tool touches. Mirror the change in `skills/next-handoff-emit/SKILL.md`.

6. `internal/cli/checkpoint.go` — adjust `nextPlaceholder` so the `## Blocked on (omit if clear)` and `## Tried and failed (omit if N/A)` placeholder headers are **omitted by default** from the initial template (their guidance lives in the skill; the placeholder shouldn't ship the "(omit if clear)" instruction as a literal header that agents then forget to remove). Keep the agent contract: "if there's a real blocker, the agent adds `## Blocked on`; otherwise no header at all."

## Boundaries

- **Not** changing the durable `<user>.md` projection structure or content sections — only the read-path repo filter.
- **Not** moving `UserAsk` / `NextSuggestion` keys to `(user, repoKey)` in v1. The read-path repo filter is sufficient to fix the symptom; the key change is a follow-up if multi-repo singleton coexistence becomes necessary.
- **Not** introducing a new scratch-notes file format. The skill update just states what `.local.md` is and isn't; users can use any plain markdown file under `.hero/notes/` if they want a preserved hand-edited briefing.
- **Not** touching `writeProjectedNextMD` (the `next.projected=true` path that rebuilds `.hero/NEXT.md` itself from the graph). That code path is already total-rewrite and not implicated.
- **Not** changing `findProjectRoot` / `workspace.LocateFromCWD`. Even if a cross-repo path-resolution issue contributed to the original accumulation, fixing the preservation contract makes the file robust to that class of source as well — no path-resolution change required for this fix.
- **Not** dropping `.hero/next/<user>.local.md` from disk. The file remains, just becomes machine-state-only.

## Risks

- **Backup file proliferation.** If a user runs many checkpoints with new hand-content between each, `.local.md.bak.<timestamp>` files accumulate. Mitigation: only write the backup when the hand-content is **non-empty** and **differs from the most recent backup** (cheap hash comparison). Acceptable to leave them around for the user to clean up — they're gitignored.
- **Behavior change for users who relied on `.local.md` scratch preservation.** The promise was documented in the command help. Mitigation: the one-time backup on first run after the upgrade gives them a recoverable copy; the stderr notice tells them where; the skill update points at a clean alternative.
- **Repo-scope filter could mis-fire if `repoKey` is empty.** `gitutil.RepoKey` returns empty in non-git contexts. The new query `WHERE repo = ''` matches only rows recorded without a repo (which today is rare but possible). Mitigation: an empty `repoKey` returns no row, treating handoff state as not-yet-recorded — same UX as a brand-new workspace. Verify behavior in tests.
- **Migration of existing graph rows.** No schema change is needed (the `repo` column already exists and is populated for nodes written by current code). Rows written by older code without a `repo` value will become invisible to repo-scoped reads. Acceptable: those rows are *exactly* the cross-repo-bleed source we want to stop reading; surfacing them on a per-repo basis would require attribution we don't have.
- **Cross-machine round-trip via `.hero/next/<user>.md`.** That file is the federation medium for solo-no-Cloud users. The change does not affect that round-trip — the durable file is still rendered and ingested. Only `.local.md` (gitignored, per-machine) loses preservation.

## Validation

- Unit tests as listed in Changes (#1, #2, #3) all pass.
- `go build ./...` and `go test ./...` clean.
- Manual repro in a workspace with a polluted `.local.md`:
  1. Stage a `.hero/next/chet-bellows.local.md` containing the marker block plus a `## Just finished` section outside it.
  2. Run `hero next checkpoint`.
  3. Verify the resulting `.local.md` contains only the marker block.
  4. Verify a `.local.md.bak.<timestamp>` file exists with the prior hand-content.
  5. Verify a stderr notice points at the backup.
  6. Run `hero next checkpoint` again immediately.
  7. Verify no second backup is written.
- Manual cross-repo bleed repro:
  1. In repo A: `hero next ask "A-context"`.
  2. In repo B: `hero next ask` (no args) — confirm output is *not* `A-context`.
  3. In repo B: `hero next checkpoint`; cat `.hero/next/<user>.md` — confirm the "Last user ask" section does not contain `A-context`.

## Acceptance Criteria

- WHEN `hero next checkpoint` rebuilds `.hero/next/<user>.local.md` THE SYSTEM SHALL produce a file containing only the marker-bounded machine-state block — no preserved narrative content from prior runs.
- IF the existing `.local.md` has non-empty content outside the marker block THEN THE SYSTEM SHALL write that content once to `.hero/next/<user>.local.md.bak.<UTC-timestamp>` and emit a one-line stderr notice naming the backup path before discarding it.
- WHEN the same `.local.md` is rebuilt again with no further hand-content drift THE SYSTEM SHALL NOT create an additional backup file.
- WHEN `LatestAsk`, `LatestSuggestion`, or `RecentReflections` is called with a `repoKey` THE SYSTEM SHALL return only rows whose `repo` column equals that key.
- WHEN `projection.UserHandoffMD` renders `.hero/next/<user>.md` in repo B THE SYSTEM SHALL NOT include `UserAsk`, `NextSuggestion`, or `SessionReflection` content recorded in any other repo.
- THE SYSTEM SHALL preserve `.hero/next/<user>.md` round-trip ingest behavior — `hero next ingest` still reads asks, suggestions, and reflections from the durable file into the local graph, scoped to the current repo.
- THE SYSTEM SHALL document the new `.local.md` contract in `skills/next-md/SKILL.md` and `skills/next-handoff-emit/SKILL.md` so the agent never writes scratch into the file expecting preservation.
- WHERE the initial `nextPlaceholder` template would otherwise emit "(omit if clear)" or "(omit if N/A)" placeholder headers THE SYSTEM SHALL omit those headers entirely; an agent that has a real blocker or failed attempt adds the heading at write time.

## Kickoff

> Pick up at: **verify and ship**. Implementation is complete on this branch — `rebuildLocalState` is total-rewrite, `backupHandContentIfNeeded` writes a single hash-deduped `.bak.<RFC3339>` next to `.local.md` with a stderr notice, the three handoff reads (`LatestAsk`, `LatestSuggestion`, `RecentReflections`) take `repoKey` and are repo-scoped at the SQL layer, all callers thread `repoKey`, the placeholder template no longer emits "(omit if clear)" / "(omit if N/A)" headers, and both skill docs document `.local.md` as machine-state-only. Tests added: `Test_rebuildLocalState_DiscardsHandContent`, `Test_writeCheckpoint_BacksUpPreExistingHandContent`, `Test_writeCheckpoint_NoBackupWhenAlreadyClean`, `Test_writeCheckpoint_BackupIdempotentOnRerun`, `Test_writeCheckpoint_CrossRepoAskDoesNotLeakIntoUserHandoff`, `TestRecordAsk_PerRepoIsolation`, `TestRecordSuggestion_PerRepoIsolation`, `TestRecentReflections_PerRepoIsolation`, `TestUserHandoffMD_DoesNotLeakCrossRepoAsk`. `go build ./...` and `go test ./...` are clean. Manual repro per §Validation in `/tmp/herotest-checkpoint` confirmed: backup written on first run, no second backup on rerun, `.local.md` contains only the marker block. To ship: review the diff (status: `delivering`), run `hero spec complete next-checkpoint-cross-repo-pollution`, then commit. The `repo: ''` empty-key behavior in non-git contexts (Risks bullet) is unchanged — out of scope; key-shape migration to `(user, repoKey)` deferred per Boundaries.
## Handoff Trail

- 2026-05-18T13:57:54Z — in ← hero-code (peer_id: ad027c2f-7f74-4a09-bf1d-6515cc906074)
  mode: spec-out
  originating_spec: hero-context-layer2-optimize (on hero-code)
  peer_spec: next-checkpoint-cross-repo-pollution (this spec)
  at_commit: 6835176
  reason: "Per-user NEXT file accumulating stale narrative sections from unrelated repos, polluting Layer 2 context that hero-context-layer2-optimize is preparing to lean on."

- 2026-05-18T18:13:21Z — out → hero-code (peer_id: ad027c2f-7f74-4a09-bf1d-6515cc906074)
  mode: advisory
  originating_spec: next-checkpoint-cross-repo-pollution
  at_commit: 9f988e5
  result_ref: 18b0bb7358d24f88e1a3eb58a7d6f86d
  reason: "Notify peer that the cross-repo pollution bug they handed off is fixed, so their Layer-2 optimization work can proceed leaning on .hero/next/<user>.md as a trusted context source."

