---
type: bug
status: completed
severity: high
root_cause_class: design
tags: [init, gitignore, index, install-hygiene, drift]
relates-to:
  - context-files-flag-drift
  - recap-unregister-stale-and-empty-repo
  - hero-import-directory-unsupported
completed_at: 2026-05-18T19:25:38Z
---

# `hero init`'s managed-gitignore block omits `.hero/index.db`, causing projects to commit the binary SQLite search index

## Kickoff

`hero init` now gitignores `.hero/index.db` (and the `-wal`/`-shm` sidecars,
defensively, matching the graph.db treatment). Existing projects pick the
new entries up the next time they run `hero init` because the managed block
is re-spliced.

**Status:** completed — three lines added to `managedGitignoreEntries` and
the gitignore test's want-list expanded to exhaustively cover every entry
in the canonical manifest.

**Shipped:**
- `internal/cli/init.go` — added `# Search index cache (regenerable via `hero index`)` stanza with `.hero/index.db`, `.hero/index.db-wal`, `.hero/index.db-shm`.
- `internal/cli/init_gitignore_test.go` — extended `TestEnsureManagedGitignoreBlock_CreatesWhenMissing` want-list to assert all entries (including the previously unasserted `graph.db-wal`, `graph.db-shm`, `satellites.local.json`).

**Verified:**
- `go build ./...` clean.
- `go test ./internal/cli/... -run Gitignore -v` — all 4 tests pass.
- `go test ./...` — full suite clean.
- Manual: built binary, ran `hero init` in fresh `git init`'d temp dir, confirmed rendered `.gitignore` block contains the three new entries inside the marker block.

**Recovery for already-affected projects:** users with `.hero/index.db` already
tracked in git need a one-time `git rm --cached .hero/index.db .hero/index.db-wal .hero/index.db-shm`
after upgrading + re-running `hero init`. The new gitignore entry keeps it
untracked from then on.

**Files:** `internal/cli/init.go:445-470`, `internal/cli/init_gitignore_test.go:22-34`

## Issue

No tracker. Reported by user from direct observation: a project they encountered
had `.hero/index.db` committed to git and was experiencing "havoc" — exactly
the failure mode this gap produces. Binary SQLite in git → constant diffs on
every reindex, merge conflicts that can't be resolved by humans, repo bloat,
and accidental leakage of content extracted from files the user thought were
gitignored.

Affects: every project ever initialised with `hero init`. The mechanism that
*should* have prevented this — the managed-gitignore marker block in
`internal/cli/init.go` — exists and works correctly; it simply doesn't list
`index.db`.

## Investigation

### Confirmed observations

1. **`index.db` is the active SQLite spec/corpus search index.**
   `internal/index/index.go:62` declares `IndexFileName = "index.db"`.
   `internal/index/index.go:72-97` (`Open`) creates the file at
   `<heroDir>/index.db` and opens it via `modernc.org/sqlite`. It's the
   FTS5-backed corpus index used by `hero search`, `hero ask`, and the
   unified retrieval layer. It is **fully regenerable** from sources of
   truth via `hero index` — there is no reason it should ever be in git.

2. **`managedGitignoreEntries` does not list `index.db`.**
   `internal/cli/init.go:445-465` — the canonical list. Current contents:
   - `.hero/hero.local.json`
   - `.hero/graph.db`, `.hero/graph.db-wal`, `.hero/graph.db-shm`
   - `.hero/next/*.local.md`
   - `.hero/knowledge/code/`
   - `.hero/satellites.local.json`

   No `index.db`. No `index.db-wal`. No `index.db-shm`.

3. **The existing test does not assert `index.db`.**
   `internal/cli/init_gitignore_test.go:22-29` —
   `TestEnsureManagedGitignoreBlock_CreatesWhenMissing` asserts the block
   contains `.hero/hero.local.json`, `.hero/graph.db`, `.hero/next/*.local.md`,
   `.hero/knowledge/code/`. `index.db` is absent from the want-list, which
   is why the gap survived. The test perfectly mirrors the code: both forgot
   the same line.

4. **`index.db` does NOT currently enable WAL mode.**
   `internal/index/index.go:79-88` opens the database and sets only
   `PRAGMA foreign_keys = ON`. Compare `internal/graph/graph.go:69-77` which
   sets `PRAGMA journal_mode = WAL` (hence why graph.db has `-wal`/`-shm`
   sidecars listed). So under today's `index.db` open path, no `-wal`/`-shm`
   sidecars are created during normal operation — SQLite uses default
   rollback journals which only briefly produce a `-journal` file mid-write.

   However: including the `-wal`/`-shm` sidecars in the gitignore anyway
   is the right call. They are zero-cost extra lines that:
   - cover any future migration to WAL on `index.db` (which is plausible
     given the corpus grows and concurrent readers exist);
   - cover any tooling or ad-hoc user session that opens `index.db`
     with WAL enabled (the SQLite CLI defaults to it in many builds);
   - match the symmetry of how `graph.db` is handled, which keeps the
     managed-block readable and future-proof.

### Code flow — how the gap reaches users

1. User runs `hero init` in a fresh project.
2. `internal/cli/init.go` eventually calls `ensureManagedGitignoreBlock(gitignorePath)`
   (`internal/cli/init.go:470-483`).
3. That function writes the marker-bounded block using
   `managedGitignoreEntries` (`internal/cli/init.go:448-465`) verbatim.
4. The block correctly ignores `hero.local.json`, `graph.db*`, `next/*.local.md`,
   `knowledge/code/`, `satellites.local.json` — but never mentions `index.db`.
5. Later, the first `hero` command that needs FTS (`hero search`, `hero ask`,
   `hero relevant`, or the auto-index hooks) calls `index.Open(heroDir)`
   (`internal/index/index.go:72`), which materialises `.hero/index.db`.
6. `git status` shows `.hero/index.db` as an untracked binary file.
7. User `git add .` (or runs any tool that does), commits the binary,
   pushes. From now on every reindex creates a huge binary diff. Merges
   between branches collide unrecoverably. Anyone working on the repo
   pulls megabytes of churn per session.

The fact that `graph.db` *is* listed proves the mechanism works — the
slip is purely that nobody added a second three-line stanza when the
search-index file was introduced.

### Root cause

**Process / design drift, not a code logic bug.**

`managedGitignoreEntries` is intentionally designed so that adding a
new regenerable artifact to gitignore is a one-line change — the
comment on `internal/cli/init.go:445-447` says so explicitly:
*"Adding a new entry here is the only change required to roll it out
— `hero init` (and re-runs) splice it in automatically."*

The bug is that when `internal/index/index.go` introduced `index.db`
as the canonical spec corpus index, the author did not make that
one-line update. The supporting surface (the gitignore manifest)
drifted from the runtime surface (the files actually written under
`.hero/`).

This is the same structural failure as the sibling bugs in this
batch:

- **`context-files-flag-drift`** — a CLI flag was removed/renamed
  but help text/docs didn't update.
- **`recap-unregister-stale-and-empty-repo`** — a command was added/removed
  but the slash-command body didn't update.
- **`hero-import-directory-unsupported`** — import gained a code path
  but the supported-targets list didn't update.

Different mechanism, identical shape: **"thing was added/removed/changed
in the runtime, supporting surface didn't update."** All four are
process bugs, not code bugs.

### Severity

**High.** Justification:

- **Blast radius:** every Hero-installed project on every machine.
  Every user who runs `hero init` is affected the moment any FTS-touching
  command runs.
- **Silent:** no error, no warning. The corruption of git history happens
  through normal-looking workflow ("git add .", "git commit").
- **Hard to undo:** once `index.db` is committed, `git rm --cached` only
  removes it from future commits — the binary blob remains in repo
  history and continues to bloat clones. Full rewrite (`git filter-repo`
  or BFG) is required to truly purge, and that's a coordinated
  branch-rewrite operation most teams won't do.
- **Discovery is delayed:** users notice only when the first cross-branch
  merge produces an unresolvable binary conflict, or when CI repo-size
  budgets trip. By then the damage is already in shared history.
- **Mission-relevant:** Hero's pitch is *"agents make the floor rise."*
  A footgun that silently corrupts every installed project's git history
  is anti-mission.

Caused by our codebase: **Yes.**

## Goal

`hero init` (and every re-run) gitignores `.hero/index.db` and its WAL
sidecars, so no Hero-installed project commits the binary search index.
Existing projects pick up the new entries the next time they run
`hero init` because the managed block is re-spliced.

## Suggested Fix Approach

### Change 1 — Add the entries to the managed gitignore manifest

**File:** `internal/cli/init.go` — `managedGitignoreEntries` (lines 448-465).

**Before:**
```go
var managedGitignoreEntries = []string{
	"# Per-project local overrides (tokens, personal preferences)",
	".hero/hero.local.json",
	"",
	"# Knowledge graph cache (regenerable from sources of truth)",
	".hero/graph.db",
	".hero/graph.db-wal",
	".hero/graph.db-shm",
	"",
	"# Per-machine NEXT state (rewritten every Stop hook + agent scratch)",
	".hero/next/*.local.md",
	"",
	"# Auto-generated code intelligence (re-scan to regenerate)",
	".hero/knowledge/code/",
	"",
	"# Per-machine satellite manifest (which subprojects are symlinked locally)",
	".hero/satellites.local.json",
}
```

**After:**
```go
var managedGitignoreEntries = []string{
	"# Per-project local overrides (tokens, personal preferences)",
	".hero/hero.local.json",
	"",
	"# Knowledge graph cache (regenerable from sources of truth)",
	".hero/graph.db",
	".hero/graph.db-wal",
	".hero/graph.db-shm",
	"",
	"# Spec corpus search index (regenerable via `hero index`)",
	".hero/index.db",
	".hero/index.db-wal",
	".hero/index.db-shm",
	"",
	"# Per-machine NEXT state (rewritten every Stop hook + agent scratch)",
	".hero/next/*.local.md",
	"",
	"# Auto-generated code intelligence (re-scan to regenerate)",
	".hero/knowledge/code/",
	"",
	"# Per-machine satellite manifest (which subprojects are symlinked locally)",
	".hero/satellites.local.json",
}
```

**Why:** This is the canonical manifest. `hero init` and every re-run splice
this list into the marker-bounded block in the root `.gitignore` via
`ensureManagedGitignoreBlock`. Adding the three lines is the entire production
fix. Sidecars are included defensively to match the `graph.db` stanza — see
investigation note in §"Confirmed observations" point 4.

### Change 2 — Extend the test to assert the new entries

**File:** `internal/cli/init_gitignore_test.go` — `TestEnsureManagedGitignoreBlock_CreatesWhenMissing` (lines 22-29).

**Before:**
```go
for _, want := range []string{
    gitignoreMarkerStart,
    gitignoreMarkerEnd,
    ".hero/hero.local.json",
    ".hero/graph.db",
    ".hero/next/*.local.md",
    ".hero/knowledge/code/",
} {
```

**After:**
```go
for _, want := range []string{
    gitignoreMarkerStart,
    gitignoreMarkerEnd,
    ".hero/hero.local.json",
    ".hero/graph.db",
    ".hero/graph.db-wal",
    ".hero/graph.db-shm",
    ".hero/index.db",
    ".hero/index.db-wal",
    ".hero/index.db-shm",
    ".hero/next/*.local.md",
    ".hero/knowledge/code/",
    ".hero/satellites.local.json",
} {
```

**Why:** Locks in coverage for the new entries *and* closes the secondary
gap that the existing test didn't assert `.hero/graph.db-wal`,
`.hero/graph.db-shm`, or `.hero/satellites.local.json` either — meaning a
future careless refactor of `managedGitignoreEntries` could silently drop
any of these without the test catching it. While we're touching the
assertion list, make it exhaustive.

### Migration / recovery for existing projects

**Automatic for the gitignore:** `ensureManagedGitignoreBlock` is idempotent
and re-splices the canonical entries on every `hero init` run
(`internal/cli/init.go:470-483` + `mergeGitignoreBlock` at lines 488-510).
The next `hero init` (or `hero upgrade`) on an existing project will replace
the old marker block with the new one — no migration code needed.

**NOT automatic — affected projects must clean up tracked files manually.**
If a project already has `.hero/index.db` (and possibly `-wal`/`-shm`)
*tracked* in git, simply updating `.gitignore` will not untrack them.
Affected users need to run, once, after upgrading Hero and re-running
`hero init`:

```bash
git rm --cached .hero/index.db .hero/index.db-wal .hero/index.db-shm 2>/dev/null
git commit -m "stop tracking regenerable hero index.db"
```

This removes the files from the index without deleting them on disk.
Subsequent commits will exclude them by virtue of the new gitignore
entry. Note: prior commits in git history still contain the binary blob
— full removal requires `git filter-repo` / BFG, which is the
project's call and out of scope for this fix.

**Suggested:** mention this recovery snippet in the spec's
acceptance-criteria validation step and in any release notes for the
Hero version that ships this fix. Consider surfacing it from `hero init`
when it detects `.hero/index.db` is currently tracked in git — but that
proactive nudge is a follow-up, not part of this fix.

## Boundaries

- **Not** enabling WAL on `index.db`. The sidecars are added defensively;
  whether to switch `index.db` to WAL is a separate concurrency/performance
  decision and should have its own spec if anyone proposes it.
- **Not** retroactively rewriting affected projects' git history.
  `git filter-repo` / BFG is a project-level operation; Hero can document
  the recovery snippet but won't execute it.
- **Not** adding a startup detector that warns when `.hero/index.db` is
  tracked in git. That's the structural mitigation suggested below — it's
  worth doing, but as a follow-up spec, not bundled into this fix.
- **Not** auditing every other file written under `.hero/` to see what else
  should be gitignored. That's also the structural mitigation — it deserves
  a deliberate sweep, not a tack-on.

## Risks

- **Low.** The change is additive — three lines added to a list and an
  enriched test assertion. No behavioural change for users whose `index.db`
  is already untracked (the vast majority on new installs going forward).
  Existing affected projects need the one-time `git rm --cached` step
  documented above.
- **Regression scope:** very narrow. The managed-block mechanism has
  four existing tests (create/preserve/idempotent/refresh) all of which
  will continue to pass — they don't enumerate entry names except in
  Change 2.
- **Marker-block re-splice on re-run** is already tested
  (`TestEnsureManagedGitignoreBlock_RefreshesUpdatedEntries`,
  `internal/cli/init_gitignore_test.go:89-111`), so existing projects
  re-running `hero init` will correctly pick up the new entries without
  duplicating or corrupting the block.

## Test Plan

### Existing test review

`internal/cli/init_gitignore_test.go` already covers the mechanism:

- `TestEnsureManagedGitignoreBlock_CreatesWhenMissing` (lines 10-34) —
  asserts entries land when no `.gitignore` exists. **This is the test
  to extend in Change 2.**
- `TestEnsureManagedGitignoreBlock_PreservesUserContent` (lines 36-61) —
  unaffected; verifies user content above/below the block survives.
- `TestEnsureManagedGitignoreBlock_IdempotentOnReRun` (lines 63-87) —
  unaffected; verifies double-run produces identical output.
- `TestEnsureManagedGitignoreBlock_RefreshesUpdatedEntries` (lines 89-111) —
  unaffected; verifies an old/stale block is replaced (this is the test
  that proves existing projects re-running `hero init` will pick up the
  new index.db lines automatically).

### Test changes needed

1. **Extend the want-list** in
   `TestEnsureManagedGitignoreBlock_CreatesWhenMissing` per Change 2 above —
   exhaustively assert every entry in `managedGitignoreEntries`. This
   prevents this exact class of bug recurring: any future entry added
   without a corresponding test line will fail on the next `go test`.

2. **(Optional, recommended)** Add a small new test
   `TestManagedGitignoreEntries_IncludesAllRegenerableArtifacts` that
   reads `managedGitignoreEntries` directly and asserts both `graph.db*`
   and `index.db*` triples appear. This makes the intent ("all SQLite
   artifacts under .hero/ must be ignored") explicit at the test level
   rather than purely positional. Skip if you'd rather keep the surface
   minimal — Change 2 alone covers the regression.

### Regression scope

Run the full `internal/cli/...` package tests. Manually verify in a
scratch repo:

```bash
cd /tmp && mkdir gitignore-test && cd gitignore-test && git init
hero init
cat .gitignore | grep index.db    # expect: .hero/index.db, .hero/index.db-wal, .hero/index.db-shm
hero index                          # creates .hero/index.db
git status --porcelain .hero/       # expect: no .hero/index.db in untracked output
```

## Notes — structural mitigation (follow-up, not in scope)

All four bugs in this diagnose batch
(`gitignore-missing-index-db`, `context-files-flag-drift`,
`recap-unregister-stale-and-empty-repo`, `hero-import-directory-unsupported`)
share the same shape: a runtime surface changed (file written, flag
renamed, command added, target supported) and a parallel supporting
surface (gitignore manifest, help text, slash-command body, docs
list) wasn't updated in lockstep.

**Suggested follow-up spec:** add a `hero check`-time self-audit that
compares files actually written under `.hero/` against
`managedGitignoreEntries` and warns when an unignored regenerable
file appears. Cheap to write, catches this exact class of drift on
every workspace, and fits the mission ("raise the floor for everyone
— including the maintainer who forgot the one-line update").

That sweep should also consider:

- Auditing `--help` output against the actual command/flag surface
  (would catch `context-files-flag-drift`).
- Auditing slash-command bodies against the canonical CLI verb surface
  (would catch `recap-unregister-stale-and-empty-repo` style drift).
- Auditing docs' "supported X" lists against the actual registered
  implementations (would catch `hero-import-directory-unsupported`).

Worth a `discover` session, not a single bug spec.

## Recap

`hero init`'s managed-gitignore block lists `.hero/graph.db*` but forgot
`.hero/index.db*` when the search-index file was introduced, so every
Hero-installed project silently commits the binary SQLite corpus index
the first time `git add .` runs after `hero search`/`hero ask`/`hero index`.
Fix is three lines added to `managedGitignoreEntries` plus an extended
assertion in the existing gitignore test; existing projects pick up the
new entries the next time they run `hero init` and need a one-time
`git rm --cached .hero/index.db*` to stop tracking the already-committed
file.
