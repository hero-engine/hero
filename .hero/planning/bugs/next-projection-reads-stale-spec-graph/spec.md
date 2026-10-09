---
title: "NEXT.md and hero_handoff show stale Next / Blocked on — the projection reads a spec graph nothing keeps current"
slug: next-projection-reads-stale-spec-graph
type: bug
status: delivering
priority: P1
severity: high
root_cause_class: logic
domain: engineering
size: medium
created: 2026-10-09
tags: [next, handoff, projection, graph, read-contract, hero-harness]
---

# NEXT.md shows stale Next / Blocked on

## Goal

`NEXT.md` (and so `hero_handoff`) reflects current spec status, the same status `hero_work` and `hero list` show, after every checkpoint. `hero_handoff` returns a clean briefing.

## Kickoff

Reported by hero-harness (Mail `mail_64ea08bb994ea32d1adfacbd`, spec-out, related `mvp-3d-project-home`) on bundled v0.35.3. Their `NEXT.md` was regenerated on 2026-10-09 but still showed problems:
- an archived, completed spec as Next;
- 16 "blocked" rows whose blockers were all completed;
- a placeholder "Just finished";
- a managed block and frontmatter ahead of the briefing.

The user's direction: "we work hard to get that accurate and fresh and something got broken along the way — find it and fix it." Verify with `go test ./...` and by checkpointing a copy of hero-harness's `.hero`.

## Root Cause

1. **The projection reads a cache that nothing keeps current.**
   - `NEXT.md` and the per-user handoff are projected from `graph.db`. Spec nodes there are written only by:
     - a cold-graph rebuild at session start;
     - `hero scan` / `hero graph reingest`;
     - Mail promotion and peer receive;
     - the read-side heal `reconcileSpecGraph`, which `hero why` and `hero blocked` gained in `55f6cfd3` (Jul 18).
   - Spec writes never touch the graph: verify, status edits and archiving change only the files. The checkpoint and `hero next project` never called the heal.
   - **Effect in hero-harness:** its graph's spec nodes were last ingested on Oct 4. Of 99 specs, 14 carried a wrong status and 69 were missing. Next and Blocked on were computed from that.
   - **Why it hid here:** this repo looked fresh only because `hero why` / `hero blocked` / `hero scan` run often as a side effect.
   - **Reproduced:** on a copy of hero-harness's `.hero`, `hero next checkpoint` stays stale. After `hero blocked` (which heals), the next checkpoint is correct.
2. **Next ranked priorities as plain strings.**
   - `ORDER BY priority` sorted text, so `low` beat `medium`, and every `P0`–`P3` beat `critical`. Slot 2 accepted only literal `P0`/`P1` bugs.
   - Its date tie-break read `$.created`, which no writer emitted. The tests seeded that prop by hand, so the gap was invisible.
   - `hero_work`'s Suggested ranking also ignored `P0`–`P3`.
3. **"Just finished" was a fixed git-log pointer.** It was made so by `30dbfc32` to avoid commit-list churn in the byte-gated file.
4. **`hero_handoff` returned `NEXT.md` verbatim**, managed snapshot block and frontmatter included.

## Fix

1. **Re-sync before projecting.** `writeCheckpoint` reconciles the spec subgraph from frontmatter (`reconcileCheckpointSpecGraph` → `reconcileSpecGraph`) before projecting `NEXT.md`, the per-user handoff and the snapshot. `hero next project` does the same. This is the same idempotent heal `hero why`/`hero blocked` use; it measured about 0.4 s for 550 specs.
2. **Rank priorities correctly.**
   - `priorityRank`/`priorityRankSQL` rank `P0`/critical > `P1`/high > `P2`/medium > `P3`/low > other, in Next, slot 2 and Context to carry forward.
   - Spec nodes now carry `created` (the authored `created:` date only, never the mtime fallback, so ordering stays deterministic across clones) and `completed_at`.
   - `hashSpec` now covers the node's full props. Without that, unchanged specs kept their old nodes and never gained the new props, so a warm local graph and CI's fresh scan would project different files.
   - `hero_work`'s `priorityRank` also maps `P0`–`P3`.
3. **Real "Just finished".** It lists the 5 most recently completed work items and initiatives by recorded `completed_at` (newest first, committed data, so the CI drift gate stays deterministic), then the git-log pointer.
5. **Checkpoint hang (found while delivering).**
   - `hero next checkpoint` reads a hook payload from stdin (`resolveSessionContext` via `autoEmitUserAsk`). The read was bounded in size but not in time, so it blocked forever on an open stdin that never sends anything. That happens with a git pre-commit hook run from an IDE or agent tool, and it hung this delivery's checkpoint twice; a goroutine dump showed it waiting in `syscall.read`.
   - `readHookPayload` now waits at most `hookPayloadWait` (1 s) and recovers a panicking reader inside its goroutine, keeping the existing never-fail contract.
4. **Clean `hero_handoff`.** `markdown` is the briefing body only. `updated_at` and a new `repo` field carry the metadata. This is additive within `schema_version: 1`, and the golden schema gained `hero_handoff.repo` (null/string).

## Acceptance Criteria

- **AC-1:** WHEN a spec's status changes after the graph last ingested it THE SYSTEM SHALL project Next, Blocked on and Just finished from its current frontmatter status at the next checkpoint (and `hero next project`).
- **AC-2:** THE SYSTEM SHALL rank `P0`/critical > `P1`/high > `P2`/medium > `P3`/low in Next (both slots) and in `hero_work`'s Suggested list. Ties are broken by the authored `created:` date.
- **AC-3:** THE SYSTEM SHALL list the most recently completed specs (by `completed_at`, at most 5) under Just finished, deterministically from committed data.
- **AC-5:** WHEN a hook's stdin is open but sends nothing THE SYSTEM SHALL finish `hero next checkpoint` without waiting on it. A payload written promptly is still read.
- **AC-4:** THE SYSTEM SHALL return from `hero_handoff` only the briefing body in `markdown`, with `updated_at` and `repo` as fields (additive to schema v1).

## Changes

1. `internal/cli/checkpoint.go` (`reconcileCheckpointSpecGraph`, called in `writeCheckpoint`), `internal/cli/next_project.go`, `internal/cli/checkpoint_reconcile_test.go`.
2. `internal/projection/projection.go` (`priorityRank`, `priorityRankSQL`, slot 2, `recentlyCompleted`, Just finished) and `projection_test.go`; `internal/spec/graph_ingest.go` (`created`, `completed_at` props) and `graph_ingest_test.go`; `internal/workmodel/polish.go` and `polish_test.go`.
4. `internal/cli/next_compact_handoff.go` (`readHookPayload`, `hookPayloadWait`) and `internal/cli/hook_payload_test.go`.
3. `internal/serve/mcp_tools_read_contract.go` (`HeroHandoff.Repo`, `handoffBody`), its tests, `read_contract_schema_test.go` and `testdata/read_contract_v1.golden`.

## Not changed (noted)

- `gitutil.RepoKey` falls back to the folder name when `git remote get-url origin` fails. hero-harness's graph and `NEXT.md` are keyed `hero-harness`, but the same checkout gives `hero-engine/hero-harness` when git works, so an environment without a working `git` writes a separate partition. This is reported to hero-harness (how is the bundled hero launched?). Making the key durable is its own design question.

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | AC-1: checkpoint projects current status | DONE | `TestCheckpointProjectsCurrentSpecStatus`: the graph ingests `engine` while it is delivering, then `engine` is verified and archived with no graph write. `writeCheckpoint` now shows Next `/deliver ui`, no `waiting on engine`, and Engine under Just finished. Without the reconcile call, all three assertions fail. End to end, the new binary's plain `hero next checkpoint` on a copy of hero-harness's `.hero` shows the 5 latest completions, a real ready Next and only genuine blockers |
| 2 | AC-2: priority ranking | DONE | `TestNextMD_PriorityConventionsRankTogether` and `TestNextMD_CriticalBugInSlot2` both fail with the old `projection.go`. `TestSuggestedRanksPLevels` fails with the old `polish.go`. `TestSpecPropsCreatedAndCompleted` (authored `created` only, plus `completed_at`) fails with the old `graph_ingest.go`. `TestSpecHashCoversProps` fails without `Props` in the hash. The existing tie-break and carry-forward fixtures were corrected from a hand-seeded prop to the one the writer emits |
| 3 | AC-3: Just finished | DONE | `TestNextMD_JustFinishedListsRecentlyCompleted`: newest 5 by `completed_at`, undated and open specs excluded, git-log pointer kept. It fails with the old code |
| 5 | AC-5: no hang on silent stdin | DONE | `TestResolveSessionContextDoesNotBlockOnSilentStdin` (open `io.Pipe`, nothing written) fails after its 5 s guard with the old code. `TestResolveSessionContextReadsPromptPayload` passes, and so does the existing panicking-reader test. The real checkpoint, run from this tool shell (open stdin), now takes 2 s instead of hanging |
| 4 | AC-4: handoff shape | DONE | `TestToolHandoff`: the managed block and frontmatter are stripped, and `updated_at` and `repo` are returned; the empty reply is `{"markdown":"","updated_at":null,"repo":null}`. The golden schema gained `hero_handoff.repo:null` and `:string` (additive-only test passes) |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | checkpoint reconcile | DONE | — |
| 2 | projection, props, ranking | DONE | — |
| 3 | hero_handoff shape | DONE | — |
| 4 | hook payload wait | DONE | — |

### Exercise-the-feature check

- [x] Ran the CI drift gate's steps (`hero scan` + `hero next checkpoint`) on a fresh clone: its `NEXT.md` matches the locally projected one byte for byte, apart from `updated:`. Ran the new binary's `hero next checkpoint` on a copy of hero-harness's real `.hero` (stale since Oct 4): Next, Blocked on and Just finished are now current. Each fix's test was falsified against the old code.

### Excellence Bar self-check

- [x] Yes. The root cause is traced through history: no write path ever refreshed specs in the graph, and the July read-side heal never reached the projection. Every section hero-harness flagged is fixed, plus the adjacent ranking defects in the same section.
