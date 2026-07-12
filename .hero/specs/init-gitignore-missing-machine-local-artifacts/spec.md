---
title: "hero init's managed .gitignore omits cache/, sessions/, and install-state.json — machine-local artifacts leak into fresh projects"
slug: init-gitignore-missing-machine-local-artifacts
type: bug
status: completed
severity: medium
priority: medium
size: small
domain: engineering
created: 2026-07-12
origin: session
root_cause_class: design
completed_at: 2026-07-12T18:08:27Z
---

# hero init's managed .gitignore omits cache/, sessions/, and install-state.json

## Summary

### Categorization
| Attribute | Assessment |
|-----------|------------|
| **Criticality** | medium — no data loss or crash, but every fresh `hero init` project leaks per-machine artifacts and confuses first-push onboarding. Compounding second defect: 3 binary `refs.db` files already committed into *this* repo. |
| **Ease of Fix** | easy — append three grouped entries to a static string slice, extend one test's expected list, and run one `git rm --cached` for the leaked files. No logic change. |
| **Caused by our codebase?** | Yes — `managedGitignoreEntries` in `internal/cli/init.go` is an incomplete canonical list. |
| **Needs more research?** | No — root cause, fix, rollout mechanism, and remediation are all confirmed against source and git state. |

### Background
`hero init` splices a marker-bounded managed block into the root `.gitignore` from a single static slice, `managedGitignoreEntries` (`internal/cli/init.go:432`). That slice covers `hero.local.json`, `graph.db*`, `index.db*`, `next/*.local.md`, `knowledge/code/`, and `satellites.local.json` — but omits three artifact types that are equally regenerable or per-machine: `.hero/cache/`, `.hero/sessions/`, and `.hero/install-state.json`. Fresh projects therefore never get these ignored, and the untracked files surface confusingly on first push.

### Analysis
Hero's own repo does **not** use the generated managed block — its root `.gitignore` is hand-maintained and has no `# >>> hero gitignore (managed) >>>` markers. That hand-maintained file already ignores all three missing artifact types (lines 8, 20, 61). The canonical generated list drifted away from what the maintainers knew to ignore by hand: as `cache/` (spec-types export), `sessions/` (per-session refs.db), and `install-state.json` (host capabilities) were added to the `.hero/` layout over time, each was added to Hero's own hand-maintained `.gitignore` but never back-ported into `managedGitignoreEntries`. A fresh `hero init` user gets the stale list.

### Root Cause
Design/completeness gap in a static list. `managedGitignoreEntries` was never updated as new machine-local artifact types were introduced into the `.hero/` directory. This is not a runtime logic bug — the splice-and-refresh mechanism works correctly; the *data* it splices is incomplete.

### Source
- `internal/cli/init.go:432-454` — `managedGitignoreEntries` (the incomplete canonical list).
- `internal/cli/init.go:459-472` — `ensureManagedGitignoreBlock` (correct; refreshes the marker-bounded block on every run).
- `internal/cli/init_gitignore_test.go:22-34` — test's expected-entries list (must be extended in lockstep).

### Fix Direction
Add the three missing entries to `managedGitignoreEntries`, grouped with section comments matching the existing style; extend the test's expected list; and separately untrack the three already-committed `refs.db` files with `git rm --cached`. Do **not** add `.hero/events.log` — it is a committed source-of-truth ledger, not a cache.

---

## Problem Statement

Two related defects.

**Defect 1 — incomplete canonical ignore list (the reported bug).**
`hero init` writes `managedGitignoreEntries` into the root `.gitignore` via `ensureManagedGitignoreBlock`. The list currently covers six artifact classes but omits three machine-local / regenerable ones:

1. `.hero/cache/` — regenerable cross-language cache (`cache/spec-types.json`). `internal/spectypes/export.go:238` describes it as "perpetually dirty" because `ExportTo` runs in the CLI's `PersistentPreRun` on every `hero` invocation. Same class as the already-ignored `graph.db`.
2. `.hero/sessions/` — per-session SQLite `refs.db` files at `sessions/<id>/refs.db` (`internal/refs/refs.go:104-113`). Session-scoped and ephemeral. Same class as `graph.db`/`index.db`. On this machine there are 24+ session dirs totaling ~920K, none meant to be shared.
3. `.hero/install-state.json` — per-machine host capabilities, described as "informational" (`internal/install/linking.go:13`) and "records what Hero did during install/upgrade" (`internal/install/state.go:16`). Same class as the already-ignored `satellites.local.json`.

Impact: on a first push in a fresh Hero project, none of these are ignored, so they appear as untracked files with no guidance on commit-vs-ignore. Confusing onboarding.

**Defect 2 — leaked refs.db files in this repo.**
Three binary `refs.db` files are committed into git tracking despite this repo's hand-maintained `.gitignore` now ignoring `.hero/sessions/` (line 8):

```
.hero/sessions/30f03a503761d2fd/refs.db
.hero/sessions/638463b8eb8ad320/refs.db
.hero/sessions/9f842f314f777d27/refs.db
```

They were committed in `70265f7` before (or independent of) the `.hero/sessions/` ignore line — and a `.gitignore` rule never untracks an already-tracked path. `git check-ignore` confirms they *would* be ignored today (`sessions/aaa/refs.db → .gitignore:8`), yet `git ls-files` still lists them. They must be untracked via `git rm --cached` (documented below; do not run during diagnosis).

### Reproduction
1. `hero init` a fresh project.
2. Run any `hero` command (populates `.hero/cache/spec-types.json` via `PersistentPreRun`; a session creates `.hero/sessions/<id>/refs.db`; install writes `.hero/install-state.json`).
3. `git status` → the three artifacts show as untracked, not covered by the generated managed block.

## Environment Details
Not environment-specific. Reproduces on any platform where `hero init` runs. Hero's own repo masks Defect 1 because it hand-maintains its `.gitignore` rather than using the generated block — so this is invisible to maintainers but hits every fresh-init user.

---

## Root Cause Analysis

**Confirmed (read this session):**
- `internal/cli/init.go:432-454` — the exact contents of `managedGitignoreEntries`. Six groups, none referencing `cache/`, `sessions/`, or `install-state.json`.
- `internal/cli/init.go:459-472` — `ensureManagedGitignoreBlock` rebuilds the block from the slice on every call and merges via `mergeGitignoreBlock`, which replaces the marker-bounded region (`:477-502`). **Rollout mechanism confirmed:** existing installs pick up new entries the next time they run `hero init` — the stale managed block is replaced, not appended. `TestEnsureManagedGitignoreBlock_RefreshesUpdatedEntries` (`init_gitignore_test.go:95-117`) already proves refresh-replaces-stale behavior.
- `internal/spectypes/export.go:230-259` — `.hero/cache/spec-types.json` is written on every invocation, guarded only against timestamp-only rewrites; explicitly called "perpetually dirty." Regenerable → ignore.
- `internal/refs/refs.go:104-117` — refs DB path is `heroDir/sessions/<id>/refs.db`; session-scoped. Ephemeral → ignore.
- `internal/install/state.go:14-53` + `internal/install/linking.go:13` — `install-state.json` is per-machine host/install metadata, "informational," "mostly forward-looking." Per-machine → ignore.
- Root `.gitignore` (hand-maintained, no managed markers) already ignores `.hero/sessions/` (:8), `.hero/cache/` (:20), `.hero/install-state.json` (:61). This is the reference set the canonical list drifted from.
- `git ls-files .hero/sessions/` → 3 tracked `refs.db`. `git log 70265f7` is where they landed.

**events.log is correctly excluded — must stay committed (confirmed):**
- `.hero/events.log` is tracked (`git ls-files` confirms) with 109 commits of history.
- `internal/feed/feed.go:54` `AppendEvent` appends workspace facts to it; it is the append-only source-of-truth ledger (spec status changes, claims, decisions, deliveries, peer_id mints, cross-repo handoffs across `internal/feed`, `internal/peering`, `internal/tracking`).
- `internal/serve/metrics/metrics.go:119` `LoadCounts` *reads* `.hero/events.log` — `hero velocity`/`hero pulse`/metrics dashboards are computed **from** it.
- `graph.db` is a regenerable cache *of* this ledger; the ledger itself is not regenerable. Adding `events.log` to the ignore list would risk irreversible loss of workspace history. **The fix must not touch it**, and the section comments should make the exclusion intentional and legible.

**Classification:** `design` (completeness gap in a static canonical list), severity `medium` (onboarding confusion + a committed-binary leak; no runtime failure, no data loss from the bug itself).

---

## Code Flow (End to End)

1. `internal/cli/init.go` (init command) → calls `ensureManagedGitignoreBlock(gitignorePath)`.
2. `internal/cli/init.go:459` — reads existing `.gitignore`, builds the managed block by iterating `managedGitignoreEntries` (`:464`).
3. `internal/cli/init.go:432-454` — **defect surfaces here**: the slice omits `cache/`, `sessions/`, `install-state.json`.
4. `internal/cli/init.go:470` — `mergeGitignoreBlock` splices the block between `gitignoreMarkerStart`/`End`, replacing any prior managed region.
5. Result: the generated `.gitignore` lacks the three entries; on a fresh project those artifacts (produced by `spectypes/export.go`, `refs/refs.go`, `install/state.go`) are never ignored.

---

## Key Files

### Managed gitignore generation
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/init.go` | 432–454 | `managedGitignoreEntries` — the incomplete canonical list (fix target). |
| `internal/cli/init.go` | 459–502 | `ensureManagedGitignoreBlock` / `mergeGitignoreBlock` — correct refresh mechanism; unchanged by fix. |
| `internal/cli/init_gitignore_test.go` | 22–34 | Expected-entries list — must gain the three new entries. |

### Artifacts being ignored (evidence of machine-local nature)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/spectypes/export.go` | 230–259 | `.hero/cache/spec-types.json` — "perpetually dirty," regenerable. |
| `internal/refs/refs.go` | 104–117 | `sessions/<id>/refs.db` — per-session, ephemeral. |
| `internal/install/state.go` | 14–53 | `install-state.json` — per-machine, informational. |
| `internal/install/linking.go` | 13 | Confirms install-state is "informational." |

### Must-NOT-ignore (guardrail)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/feed/feed.go` | 54–91 | `AppendEvent`/`ReadEvents` — events.log is the append-only ledger. |
| `internal/serve/metrics/metrics.go` | 119–133 | `LoadCounts` reads events.log for velocity/pulse. |

---

## Secondary Defects
- **Leaked binary refs.db files (Defect 2, above).** Already-tracked; the ignore rule alone won't remove them. Remediation is `git rm --cached` (see Suggested Fix Approach step 3). Not caused by the fix, but should ride along with it so the ignore rule and the tracked state agree.
- No other defects found in the flow. The splice/merge logic is correct and well-tested.

---

## Notes
- Trailing-slash vs. glob semantics: prefer directory ignores `.hero/cache/` and `.hero/sessions/` over a narrower glob like `.hero/sessions/*/refs.db`. Both directories are entirely machine-local (verified: nothing under them is meant to be shared), so ignoring the whole directory is simplest and matches what Hero's own hand-maintained `.gitignore` already does (lines 8, 20). `install-state.json` is a single file → plain path.
- The fix is self-rolling-out: because `ensureManagedGitignoreBlock` replaces the marker block on every `hero init`, existing users who re-run init get the three new entries with no migration step. New users get them immediately.

---

## Recap
Fresh `hero init` projects leak three machine-local artifacts (`.hero/cache/`, `.hero/sessions/`, `.hero/install-state.json`) because the canonical `managedGitignoreEntries` slice was never updated as those artifact types were added — a design/completeness gap, not a runtime bug. Hero's own repo hides this by hand-maintaining its `.gitignore`, which has already ignored all three; the fix back-ports them into the generated list. A related leak — three committed `refs.db` binaries — must be untracked with `git rm --cached`. `.hero/events.log` must stay committed; it is the source-of-truth ledger, not a cache.

---

## Suggested Fix Approach

### 1. Add the three missing entries to `managedGitignoreEntries`

**File:** `internal/cli/init.go`, function-level `var managedGitignoreEntries` (lines 432–454).

**Before** (tail of the slice):
```go
	"# Auto-generated code intelligence (re-scan to regenerate)",
	".hero/knowledge/code/",
	"",
	"# Per-machine satellite manifest (which subprojects are symlinked locally)",
	".hero/satellites.local.json",
}
```

**After:**
```go
	"# Auto-generated code intelligence (re-scan to regenerate)",
	".hero/knowledge/code/",
	"",
	"# Per-machine satellite manifest (which subprojects are symlinked locally)",
	".hero/satellites.local.json",
	"",
	"# Regenerable cross-language cache (rewritten every hero invocation)",
	".hero/cache/",
	"",
	"# Per-session ref store (ephemeral, session-scoped)",
	".hero/sessions/",
	"",
	"# Per-machine install state (host capabilities, informational)",
	".hero/install-state.json",
}
```

**Why:** back-ports the three machine-local/regenerable artifact types that Hero's own hand-maintained `.gitignore` already ignores into the canonical list `hero init` ships. Section comments mirror the existing per-group comment style so intent is legible. Directory forms (`cache/`, `sessions/`) are used deliberately — both trees are entirely machine-local. **`.hero/events.log` is intentionally NOT added:** it is the committed append-only ledger that `hero velocity`/`hero pulse`/metrics read from and that `graph.db` is merely a regenerable cache of.

### 2. Extend the test's expected-entries list

**File:** `internal/cli/init_gitignore_test.go`, `TestEnsureManagedGitignoreBlock_CreatesWhenMissing` (lines 22–34).

**Before:**
```go
		".hero/next/*.local.md",
		".hero/knowledge/code/",
		".hero/satellites.local.json",
	} {
```

**After:**
```go
		".hero/next/*.local.md",
		".hero/knowledge/code/",
		".hero/satellites.local.json",
		".hero/cache/",
		".hero/sessions/",
		".hero/install-state.json",
	} {
```

**Why:** the test asserts each canonical entry is present in the generated block. Without this, the new entries are untested; with it, the test locks in the completeness fix and guards against future drift.

### 3. Untrack the three leaked refs.db files (remediation — do NOT delete working files)

Run (a human/delivery step, **not** part of diagnosis):
```sh
git rm --cached \
  .hero/sessions/30f03a503761d2fd/refs.db \
  .hero/sessions/638463b8eb8ad320/refs.db \
  .hero/sessions/9f842f314f777d27/refs.db
```

**Why:** these binaries are already tracked, so the `.hero/sessions/` ignore rule cannot remove them on its own. `--cached` unstages them from git tracking while leaving the working-tree files intact. After this commit, `git ls-files .hero/sessions/` returns empty and the ignore rule holds going forward. Do not use plain `git rm` (would delete the local files) and do not run this during diagnosis.

---

## Test Plan

### Existing test review
- `internal/cli/init_gitignore_test.go`
  - `TestEnsureManagedGitignoreBlock_CreatesWhenMissing` (:10) — asserts each canonical entry is present. **Extend** its expected list (fix step 2).
  - `TestEnsureManagedGitignoreBlock_PreservesUserContent` (:42) — unaffected; still passes.
  - `TestEnsureManagedGitignoreBlock_IdempotentOnReRun` (:69) — unaffected; the larger block stays idempotent.
  - `TestEnsureManagedGitignoreBlock_RefreshesUpdatedEntries` (:95) — already proves stale blocks are replaced with the current list; this is the rollout guarantee for existing installs. No change needed.

### Test changes needed
1. Add the three new entries to `CreatesWhenMissing`'s expected slice (fix step 2). Minimum viable and consistent with the existing test's structure.
2. (Optional, defensive) Add an assertion in `RefreshesUpdatedEntries` that `.hero/cache/`, `.hero/sessions/`, and `.hero/install-state.json` all appear after refresh — proves an existing install re-running `hero init` gains the new entries. Low cost, directly encodes the rollout claim.
3. (Optional, guardrail) Assert the generated block does **not** contain `.hero/events.log`. Cheap regression fence that encodes the CRITICAL constraint so a future edit can't silently start ignoring the ledger.

### Regression scope
- Only `internal/cli/init.go` (data-only change to a static slice) and the test file change. No control flow touched.
- Verify no code path depends on `.hero/cache/`, `.hero/sessions/`, or `.hero/install-state.json` being *tracked* — confirmed none are (all are regenerated/per-machine, and Hero's own repo already ignores them).
- Run `go test ./internal/cli/...` — expect all four gitignore tests green.
- After the `git rm --cached` step (delivery), confirm `git ls-files .hero/sessions/` is empty and `git status` shows the three files staged for deletion-from-index only (working files retained).

---

## Kickoff

Fresh `hero init` projects don't ignore `.hero/cache/`, `.hero/sessions/`, or
`.hero/install-state.json` — the canonical managed-gitignore list drifted from
what Hero's own repo already ignores by hand. Plus 3 `refs.db` binaries leaked
into git tracking.

**Status:** planning — diagnosis complete, fix is data-only + one test + one `git rm --cached`.

**Pick up at:** append the three grouped entries (with section comments) to
`managedGitignoreEntries`, extend the expected list in
`TestEnsureManagedGitignoreBlock_CreatesWhenMissing`, then `git rm --cached` the
three tracked `refs.db` files. Do NOT add `.hero/events.log` — it's the committed ledger.

→ `/deliver init-gitignore-missing-machine-local-artifacts`

**Files:** `internal/cli/init.go:432-454`, `internal/cli/init_gitignore_test.go:22-34`
**Skip:** the narrow glob `.hero/sessions/*/refs.db` — use the whole-directory form `.hero/sessions/`; both dirs are entirely machine-local.

---

## Completion Ledger

| Item | Status | Evidence |
|------|--------|----------|
| Add `.hero/cache/` to `managedGitignoreEntries` | DONE | `internal/cli/init.go` — grouped entry with section comment |
| Add `.hero/sessions/` to `managedGitignoreEntries` | DONE | `internal/cli/init.go` — grouped entry with section comment |
| Add `.hero/install-state.json` to `managedGitignoreEntries` | DONE | `internal/cli/init.go` — grouped entry with section comment |
| Do NOT add `.hero/events.log` (guardrail) | DONE | absent from slice; regression fence asserts it's never ignored |
| Extend `CreatesWhenMissing` expected list | DONE | `internal/cli/init_gitignore_test.go` — 3 entries added |
| Rollout coverage assertion in `RefreshesUpdatedEntries` | DONE | asserts existing installs gain new entries on re-init |
| Untrack 3 leaked `refs.db` binaries | DONE | `git rm --cached`; `git ls-files .hero/sessions/` empty; working files retained |
| Tests green + build clean | DONE | 4/4 gitignore tests PASS; `go build ./...` exit 0 |

**Cold audit:** SHIP / clean / high confidence — see [delivery-audit.md](delivery-audit.md).
