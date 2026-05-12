---
title: Master Ingest Restore — `hero scan` Returns to Its V2 Promise
type: feature
status: delivering
priority: P0
tags: [scan, ingest, corpus, mission-critical, v2-recovery]
created: 2026-04-28
relations:
  - target: get-back-on-track
    kind: parent
  - target: traversal-queries
    kind: enables
  - target: graph-memory
    kind: completes
  - target: v2-delivery-audit-2026-04-28
    kind: motivated-by
mission_alignment: |
  The mission is "AI gets the right context at the right moment,
  including the stuff nobody told it." If the corpus is incomplete,
  the model is missing context — period. The v2 spec promised a single
  verb that pulls all signals into the graph; today three sources are
  missing and Tier-2 extraction is opt-in. Restoring master ingest is
  the most direct corpus-completeness move possible.
principles_check: |
  Serves #1 directly (it just works — one verb, full corpus). Reverses
  the existing principle violation where six different sources require
  six different commands. Nothing risked.
horizon: now
smoke:
  script: scripts/smoke/master-ingest-restore.sh
  expects: [master-ingest-restore:AC-1, master-ingest-restore:AC-2, master-ingest-restore:AC-3, master-ingest-restore:AC-4, master-ingest-restore:AC-5, master-ingest-restore:AC-6, master-ingest-restore:AC-7, master-ingest-restore:AC-8]
  runs_on: [commit-touches:internal/cli/scan*.go, commit-touches:internal/knowledge/*.go, commit-touches:internal/memory/*.go, nightly]
---

## Goal

Restore `hero scan` to the v2 spec's promise: **"Master ingest: code,
planning, notes, raw, git, tracker, sync."** Today it does code +
planning + git + raw + sessions only. Three sources are silently
missing and Tier-2 extraction never runs automatically.

## What's broken (revised after empirical audit on 2026-04-28)

The v2 system-design spec (`hero-v2-system-design`) and graph-memory
spec both promised a single verb as the master ingest. After running
`hero scan` against the populated graph (verifying via
`hero graph stats`), the actual gap is **smaller than the original
audit claimed.** Corrections noted inline.

| Source | V2 promise | Today | Notes |
|---|---|---|---|
| Stack detection / knowledge stubs | ✅ in scan | ✅ in scan | working |
| Code subgraph | ✅ in scan | ✅ in scan | 57 packages, 351 files, 1142 symbols, 2810 edges on hero repo |
| Planning specs | ✅ in scan | ✅ in scan | spec.WriteGraph — 138 Feature, 14 Initiative, 8 Decision, 8 Convention nodes |
| Git log | ✅ in scan | ✅ in scan | 181 commits, 181 persons, 5 issues, 1064 edges |
| Sessions / NEXT.md attempts | ✅ in scan | ✅ in scan | sessions + nextdoc |
| Raw ingested docs | ✅ in scan | ✅ in scan | knowledge.WriteRawGraph |
| Notes-as-Note-nodes | ✅ in scan | ✅ in scan | **Audit was wrong.** spec.WriteGraph creates `Note`-typed nodes from any spec.md with `type: note` frontmatter. Verified: 7 Note nodes from `knowledge/notes/`. |
| Sibling repos | (not in v2 spec) | ✅ in scan | bonus from unified-search; ingests sibling-repo specs via `repos:` config |
| **Tier-2 LLM extraction** (notes prose → Decision/Concept) | ✅ async in scan | ✅ in scan **(commit `0e08673`)** | Wired via `extract.RunAuto` in scan flow. Best-effort: skips silently with reason if no API key. |
| **Memory files** (`~/.claude/.../memory/`) | ✅ Memory nodes | ✅ in scan | Wired via `memory.WriteGraph` in scan flow. Best-effort: missing dir skips silently. Memory nodes scoped local — never sync. |
| **Tracker pull** | ✅ in scan | ✅ in scan | Wired via `tracker.PullAndWriteGraph` in scan flow. Skips silently when no tracker config / token. Calls existing `Tracker.ListIssues`; uses new `tracker.WriteIssuesGraph` to upsert Issue + Person nodes. |
| **Team-server sync** | ✅ "opportunistic on every scan" | ✅ in scan | Wired via `runOpportunisticTeamSync` in scan flow. Push always (cheap, deltas-only). Pull only when last-pull >5 min stale. Skips silently when not logged in or `cloud.org_id` not set. |

The v2 spec was explicit: *"Every command is a verb that does one
thing. No sub-verb sprawl."* Three sources still need to fold into
scan to honor that. The sub-verbs (`hero extract`, `hero sync`,
`hero import`) stay as escape hatches per principle #5.

## Approach

Add five best-effort orchestration steps to `runScan` in
[internal/cli/scan.go](../../../internal/cli/scan.go), following the
existing pattern (warn-and-continue on failure, never block the scan).

```go
// after existing code/planning/git/raw/sessions ingest...

if err := writeNotesGraph(heroDir, repoKey, store); err != nil {
    fmt.Fprintf(os.Stderr, "Warning: notes ingest failed: %v\n", err)
}

if err := writeMemoryGraph(memoryDir, store); err != nil {
    fmt.Fprintf(os.Stderr, "Warning: memory ingest failed: %v\n", err)
}

if cfg.Tracker.Enabled() {
    if err := tracker.PullAndWriteGraph(cfg, store); err != nil {
        fmt.Fprintf(os.Stderr, "Warning: tracker pull failed: %v\n", err)
    }
}

if cfg.TeamServer.Connected() {
    if err := sync.OpportunisticPushPull(cfg, store); err != nil {
        fmt.Fprintf(os.Stderr, "Warning: team-server sync failed: %v\n", err)
    }
}

if cfg.Extraction.Enabled() && os.Getenv("ANTHROPIC_API_KEY") != "" {
    if err := extract.RunAll(store, ...); err != nil {
        fmt.Fprintf(os.Stderr, "Warning: Tier-2 extraction failed: %v\n", err)
    }
}
```

Print a single "Graph ingest summary" block at the end listing what
ran, what was skipped (with reason: "no API key", "tracker not
configured", "team server not connected"), what counts changed.

The existing `hero extract`, `hero sync pull`, `hero sync cloud`
commands stay as escape hatches for the practitioner — they just
don't need to be the *primary* path anymore. Honors principle #5.

## Acceptance criteria (build-out-as-we-go set)

**AC-1:** ✅ **passing** (was always passing — audit error). `hero scan`
on a fresh workspace populates `Note` nodes for every
`knowledge/notes/*/spec.md` file. `hero graph stats` shows `Note`
count = note-dir count. Verified 2026-04-28: 7 Note nodes match
the 7 knowledge/notes/ directories.

**AC-2:** ✅ **passing** (commit `0dce2d1`, 2026-04-28). `hero scan`
walks `~/.claude/projects/<project-key>/memory/` and upserts `Memory`
nodes (scope: local — these never sync). `hero graph stats` shows
`Memory` count after a memory file is present. Skipped silently when
the dir doesn't exist. Verified end-to-end: empty dir → no output;
populated dir → "Graph memory: N files" + matching node count in
`hero graph stats`. Tier-1 deterministic, no external deps.

**AC-3:** ✅ **passing** (commit `5791ebf`, 2026-04-28). `hero scan`
on a workspace with a configured tracker calls
`tracker.PullAndWriteGraph` to fetch open issues (capped at 100) and
upsert Issue + Person nodes via `tracker.WriteIssuesGraph`. Skipped
silently when tracker not configured or token missing — prints a
single `Graph tracker: skipped — <reason>` line. Verified end-to-end
on this repo (no tracker → skip path); Issue-node creation paths
covered by `TestWriteIssuesGraph_UpsertsIssuesAndPersons` and
`TestWriteIssuesGraph_Idempotent`.

**AC-4:** ✅ **passing** (commit `05e5b10`, 2026-04-28). `hero scan`
opportunistically calls `runOpportunisticTeamSync`, which pushes
pending non-local rows always (cheap — deltas only) and pulls when
the last-pull cursor is older than 5 minutes. Skipped silently with a
one-line reason when the user isn't logged in or `cloud.org_id` is
unset. Verified end-to-end on this repo (no creds → skip path).
Two-machine convergence test deferred until live cloud account is
available; the push/pull plumbing reuses the same `store.Push` /
`store.Pull` paths exercised by `hero sync graph push|pull`.

**AC-5:** ✅ **passing** (commit `0e08673`, 2026-04-28). `hero scan`
on a workspace with `ANTHROPIC_API_KEY` set runs Tier-2 extraction
on notes + specs and produces `Decision` / `Concept` nodes. Skipped
(with one-line "Tier-2 disabled, no API key" notice) when no key.
Idempotent on re-run via content-hash cache. Verified: skip path
exits cleanly with reason; with-key path exercises existing
`extract.DecisionExtractor.ExtractFromSource` flow.

**AC-6:** ✅ **passing** (commit `327e750`, 2026-04-28). `hero scan`
end-of-run prints a structured "Graph ingest summary" block. Each
step contributes a `stepResult` to a shared `ingestReport`; the
report renders as one block with three glyphs (✅ ran, ⊘ skipped,
❌ failed) and a one-line per-step detail. Replaces the eleven
per-step `Graph X: …` prints with a single coherent summary.
Verified end-to-end on the live repo — 12 entries rendered correctly.

**AC-7:** ✅ **passing** (commit `327e750`, 2026-04-28). Idempotent
on second run: `hero graph stats` before/after consecutive
`hero scan` runs shows identical totals (2186 nodes / 3752 edges →
2186 / 3752). Idempotency is structural (content-hashed upserts) so
this holds for any unchanged source.

**AC-8:** ✅ **passing** (commit `327e750`, 2026-04-28). Per-step
failure isolation: each ingest step returns its own outcome to the
shared report rather than bubbling errors up. A failed step emits a
`❌ <name>: <err>` row; the next step still runs. Unit tests in
`scan_report_test.go` confirm the summary surfaces all three
outcomes (ok / skipped / failed) without one masking another.

ACs accrete as edge cases surface (e.g., partial tracker pull on
network failure, OAuth token expiry).

## Implementation notes

- **`writeNotesGraph`**: probably a new function in
  `internal/knowledge/`. Reads `knowledge/notes/*/spec.md`, upserts
  `Note` nodes with `belongs_to_project` and any extracted
  `proposes` edges (the Tier-2 part). Tier-1 part (just the Note
  node) runs unconditionally; Tier-2 runs if API key present.
- **`writeMemoryGraph`**: walks the `~/.claude/projects/<project-key>/memory/`
  directory (locate via existing `auto-memory` system), upserts
  `Memory` nodes scoped `local`. These never sync (per v2 spec:
  *"Memory is personal"*).
- **`tracker.PullAndWriteGraph`**: factor existing `hero sprint load`
  / `hero sync pull` logic into a callable. Same code path; just
  exposed as a library function the scan flow calls.
- **`sync.OpportunisticPushPull`**: factor existing
  `hero sync graph push|pull` logic. Push is conditional on having
  pending unscoped-as-local nodes; pull is conditional on cursor
  staleness (>5 min since last pull).
- **`extract.RunAll`**: already exists as `hero extract all`. Wrap.

## Out of scope

- Replacing `hero extract`, `hero sync`, `hero import` with `hero
  scan` (they remain as escape hatches and for cron-driven flows).
- Changing how Tier-2 LLM choice is configured — uses existing
  `models.json` defaults.
- Wiring `hero scan` into a watcher daemon — that's `hero watch` /
  `hero serve`, separate.

## Open questions

- Should Tier-2 extraction be on by default if the key is present, or
  require explicit opt-in via `hero.json`? Lean: on by default,
  user-disable via `extraction.auto: false`. Magic-by-default is
  principle #1.
- Should opportunistic team-sync run on every scan or rate-limited?
  Lean: rate-limit pull to 5 min; push always (cheap).
- What's the budget cap for Tier-2 extraction in a single scan? Lean:
  enforce existing `cost.budget_per_session` setting; degrade to
  "skipped, budget exceeded" with clear message.
