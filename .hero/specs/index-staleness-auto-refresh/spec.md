---
title: Index Staleness Auto-Refresh — Specs Created Outside the Workflow Always Surface
type: feature
status: completed
priority: high
tags: [index, staleness, search, list, mcp, self-healing]
created: 2026-05-01
relations:
  - target: hero-spec-complete-idempotent-move
    kind: related
  - target: kickoff-prompts-queue
    kind: related
horizon: now
---

## Kickoff

Make the search/list index self-heal so specs created on disk (via `/design`, manual edit, paste, third-party tools) always show up in `hero search` / `hero_list` / `hero_queue` / `hero_kickoff` without anyone having to remember to run `hero index`.

**Status:** completed — `index.RefreshIfStale` ships at [internal/index/refresh.go](internal/index/refresh.go), wired into all 6 MCP read tools, 3 CLI read commands, the 3 spec-authoring skills, and the pre-commit hook. `hero index --if-stale -q` is the canonical refresh entry point.

**Pick up at:** lived-experience pass on the new project — `/design` should now register specs in the index immediately; `hero list`/`hero search` should find them without a manual `hero index`.

→ `hero index --if-stale`

**Files:** [internal/index/index.go:251](internal/index/index.go:251), [internal/index/index.go:1389](internal/index/index.go:1389), [internal/serve/mcp.go](internal/serve/mcp.go), [internal/cli/list.go](internal/cli/list.go), [internal/cli/queue.go](internal/cli/queue.go)

## Goal

Eliminate the "index staleness" failure mode: a spec exists on disk but `hero search` / `hero_list` / `hero_queue` say it doesn't. Either by reactive auto-refresh on read (correctness floor) or by proactive refresh after writes (performance optimization), so that the user — and any agent calling these tools — always sees the current state.

## Problem

Reproduced in user testing on a new project: agent in a fresh session was asked to deliver a feature, called `hero list` and `hero search`, both returned no match. The spec existed on disk at `.hero/planning/features/<slug>/spec.md` but had never been indexed. The user had to manually point the agent at the file path.

The index is SQLite-backed full-text + structural data over the spec corpus. It's populated by `hero index` (full rebuild) or per-spec calls to `idx.IndexSpec(spec, content)`. Today nothing routes into the index automatically when a spec is written — `/design` writes the file and exits, `/deliver` mutates frontmatter and exits, manual edits and pastes never touch the index. So the index drifts the moment anyone authors a spec outside the rebuild path.

The result: every read-side tool that goes through the index can be silently wrong. `hero_search` (FTS-backed). `hero_search` results in `hero list` (when called via the legacy MCP path on older binaries). `hero_kickoff` and `hero_queue` use `spec.Discover` directly so they're already disk-fresh — but anything index-backed is suspect.

## Design

### Cheap stale detection per-spec

The `specs` table already carries `modified_at TEXT` (RFC3339, set from the file's mtime at index time — [internal/index/index.go:251](internal/index/index.go:251)). To detect drift:

1. Query `SELECT slug, modified_at, path FROM specs` once — the indexed view.
2. Walk disk via `spec.Discover(heroDir)` once — the truth.
3. For each disk spec:
   - **Not in index** → re-index (`IndexSpec`).
   - **In index, disk mtime > stored modified_at** → re-index.
   - **In index, mtime matches** → skip (most common case).
4. For each indexed slug not on disk → `RemoveSpec(slug)` (rare; happens after a `git pull` deletes a spec or `hero spec complete` moves it without reindexing).

Steady-state cost when nothing's changed: ~stat per spec.md on disk + one SELECT against the index. On 425 specs that's a few hundred microseconds total.

### `index.RefreshIfStale(heroDir) (RefreshStats, error)`

New top-level function in `internal/index/refresh.go`:

```go
type RefreshStats struct {
    Indexed int  // newly added (was on disk, not in DB)
    Updated int  // re-indexed (mtime newer than stored)
    Removed int  // removed orphans (was in DB, not on disk)
    Scanned int  // total specs walked on disk
    DurationMS int64
}

func RefreshIfStale(heroDir string) (RefreshStats, error)
```

Opens the DB, performs the diff, applies the changes, returns stats. Self-contained — callers don't need to manage transactions or worry about partial application; each spec mutation is its own transaction (matching `IndexSpec`'s existing pattern).

### Read-side: lazy-on-entry

Every tool/command that reads from the index calls `RefreshIfStale` once at the start, before querying. This is the **correctness floor** — query results are guaranteed current regardless of how the spec got onto disk.

Wired into:

- **MCP tools** — `hero_search`, `hero_list`, `hero_queue`, `hero_kickoff`, `hero_knowledge`, `hero_read_spec`. (Note: `hero_list` and `hero_queue` and `hero_kickoff` already use `spec.Discover` post-`kickoff-prompts-queue`, but they don't hurt by ensuring the index is current — other tools may chain through them, and `hero_search` going through the same self-heal makes consistency obvious.)
- **CLI read commands** — `hero search`, `hero list`, `hero queue`. The disk-scan-based commands self-heal too because `hero check` and `hero recap` chain through index data.

Errors from `RefreshIfStale` are logged to stderr but not fatal — the query still runs against whatever's in the index. Better to return slightly-stale data than fail the read.

### Write-side: eager triggers (performance optimization, not correctness)

These reduce how often the lazy path actually does work. Each is small:

- **`/design`, `/deliver`, `/diagnose`** skill steps — call `hero index --if-stale` after writing a spec. Surgical: only the just-written spec is dirty, so the call is microseconds.
- **Pre-commit hook** — extend the existing managed block to call `hero index --if-stale -q` before staging. Index stamp lands in the same commit as the spec change.
- **`hero serve` daemon** — already has a file watcher; it should call `RefreshIfStale` on `.hero/**/spec.md` events. (Out of scope for v1 if the watcher is already keeping the index hot — confirm during delivery.)

### `hero index --if-stale` CLI flag

Existing `hero index` rebuilds from scratch. Add a `--if-stale` flag (alias `-s`):

- `hero index` (current behavior preserved) — full rebuild.
- `hero index --if-stale` — runs `RefreshIfStale`, prints `Indexed N, Updated N, Removed N` summary.
- `hero index --if-stale -q` — same but suppresses output (for hooks).

This is the canonical surface for "refresh what's drifted" — used by skills, hooks, and any caller that wants to be defensive without paying for a full rebuild.

### Within-process caching (deferred, not v1)

If a single CLI invocation makes multiple index-reading calls in a row (e.g., a tool that calls `hero_list` then `hero_kickoff`), each currently pays the stale-check cost. A simple per-process cache — "we already checked at X timestamp, skip if X < 1s ago" — would dedupe. Not shipping in v1; trivial to add if profiling shows it matters.

### What's NOT shipping

- **Replacing `hero index`'s default behavior.** Full rebuild stays the default for backwards compatibility and disaster recovery. `--if-stale` is opt-in (and what hooks/skills use).
- **Daemon file-watcher integration.** `hero serve` already exists with a watcher; piggybacking it is its own small effort. Document the gap; don't expand scope.
- **Stop hook integration.** Per-harness Stop hook config is out of scope (different surface, different audit trail). The lazy-on-read path catches the gap regardless of hook coverage.
- **Reindexing the graph nodes (`fts_nodes` / `node_index`).** Those are projected from the graph DB, not from spec.md files. Different staleness story; out of scope here.
- **Schema changes.** Existing `modified_at` column does the job.

## Changes

- `internal/index/refresh.go` — new file. Defines `RefreshStats` and `func RefreshIfStale(heroDir string) (RefreshStats, error)`. Reuses `spec.Discover`, `Open`, `IndexSpec`, `RemoveSpec`. Per-spec mtime comparison drives the diff.
- `internal/index/refresh_test.go` — covers the four cases: no drift (no-op), new spec on disk (added), modified spec (updated), spec removed from disk (orphan deleted).
- `internal/cli/index.go` — add `--if-stale` (`-s`) and `--quiet` (`-q`) flags to the existing `hero index` command, route to `index.RefreshIfStale` when set.
- `internal/cli/index_test.go` — extend or add tests for the new flag.
- `internal/serve/mcp.go` — call `index.RefreshIfStale` at the top of `toolSearch`, `toolList`, `toolQueue`, `toolKickoff`, `toolKnowledge`, `toolReadSpec`. Errors logged, don't fail the tool call.
- `internal/cli/list.go`, `internal/cli/queue.go` — call `RefreshIfStale` early in `runList` / `runQueue`. (`hero search` uses index directly — wire it there too.)
- `internal/cli/search.go` (or wherever `hero search` lives) — call `RefreshIfStale` early in the run.
- `commands/design.md` — add a one-line step: after saving the spec, run `hero index --if-stale -q`.
- `commands/deliver.md` — same step after status flips / kickoff updates / chunks land.
- `commands/diagnose.md` — same step after writing the fix spec.
- `internal/cli/next_hooks.go` — extend the pre-commit managed block to call `hero index --if-stale -q || true` before staging projected files. Updates `hookScript("pre-commit")` and the existing test fixture.
- `internal/cli/next_hooks_test.go` — assert the new line is in the script.

No new commands, no MCP tool additions, no schema changes. Single-purpose feature.

## Acceptance Criteria

- THE SYSTEM SHALL provide an `index.RefreshIfStale(heroDir)` primitive that diff-syncs the index against disk truth.
- WHEN a spec.md exists on disk but is not in the index THE SYSTEM SHALL index it during refresh.
- WHEN a spec.md on disk has a newer mtime than the indexed `modified_at` THE SYSTEM SHALL re-index it during refresh.
- WHEN a slug exists in the index but no corresponding spec.md exists on disk THE SYSTEM SHALL remove the orphaned slug during refresh.
- WHEN every disk spec.md matches its indexed `modified_at` THE SYSTEM SHALL leave the index untouched and return a no-op summary.
- WHEN `hero index --if-stale` runs THE SYSTEM SHALL invoke `RefreshIfStale` and print a `Indexed N, Updated N, Removed N` summary to stdout.
- WHEN `hero index --if-stale -q` runs THE SYSTEM SHALL suppress non-error output (for hook callers).
- WHEN any of the MCP tools `hero_search`, `hero_list`, `hero_queue`, `hero_kickoff`, `hero_knowledge`, or `hero_read_spec` is invoked THE SYSTEM SHALL call `RefreshIfStale` before querying.
- WHEN any of the CLI commands `hero search`, `hero list`, or `hero queue` runs THE SYSTEM SHALL call `RefreshIfStale` before querying.
- IF `RefreshIfStale` returns an error during a read-side call THEN THE SYSTEM SHALL log the error to stderr and continue executing the original query against whatever index state is current.
- WHEN the pre-commit hook runs THE SYSTEM SHALL invoke `hero index --if-stale -q` so the index stamp travels with the commit alongside NEXT.md and QUEUE.md.
- THE SYSTEM SHALL preserve `hero index`'s default behavior (full rebuild) when called without `--if-stale`.
- THE SYSTEM SHALL not modify the existing index schema.

## Boundaries

- Does **not** replace `hero index`'s full-rebuild default. `--if-stale` is opt-in.
- Does **not** wire the `hero serve` daemon file watcher to `RefreshIfStale`. Watcher integration is its own small follow-up.
- Does **not** install or configure harness-side Stop hooks. Lazy-on-read covers the gap regardless.
- Does **not** add a within-process cache for repeated stale checks. Trivial to add later if profiling shows it matters.
- Does **not** touch `fts_nodes` / `node_index` (graph projection); those have their own refresh story.
- Does **not** add a schema migration. The existing `modified_at` column is the staleness signal.

## Mission Fit

> "Does this make the next agent session start smarter than the last one ended — and does it raise the floor for everyone, not just the senior dev who already knows what to ask?"

Floor-raising. Today, every agent that uses Hero MCP tools to find specs (the *correct* behavior, the one we're training for) gets silently-wrong results when the index is stale. The user has to know the workaround: "run `hero index` first." With this fix, the tools self-heal — *every* user, *every* agent, *every* session. The agent calls `hero list`, gets the truth, and proceeds. The senior who runs `hero index` manually doesn't notice; the noob never has to learn it exists.
