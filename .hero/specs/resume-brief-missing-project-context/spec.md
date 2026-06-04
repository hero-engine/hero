---
title: "Resume brief's project-context sections come up empty — commits never become graph nodes, and a fresh clone has no project graph at all"
slug: resume-brief-missing-project-context
type: bug
status: completed
severity: medium
priority: medium
domain: engineering
created: 2026-06-04
origin: session
root_cause_class: design
relates-to:
  - resume-brief-surfaces-handoff
  - e2e-handoff-continuity
  - next-as-projection
  - handoff-one-call-simplification
completed_at: 2026-06-04T18:09:06Z
---

# Resume brief's project-context sections come up empty

## Summary

### Categorization
| Attribute | Assessment |
|-----------|------------|
| **Criticality** | medium — the handoff *singletons* (ask / suggestion / reflection) DO survive both same-machine and cross-machine, so "what was I asked / what's next" is preserved. What's lost is the surrounding project-context richness (Just-changed commits, In-flight specs, Blocked, Tried, Nearby). A content-quality judge rated the resume brief 6/10 partly because of this — "an empty map." |
| **Ease of Fix** | moderate — the same-machine half is a small, well-scoped ingest wire-up (one `WriteGitLogGraph` call on the right hook). The cross-machine half is a slightly larger "rebuild project context on a cold graph" step plus a decision about where it fires. |
| **Caused by our codebase?** | Yes — both halves are gaps in how/when Hero records and rebuilds the project graph, not external factors. |
| **Needs more research?** | No — root cause is confirmed against source for both observations. The only open question is a design choice (where the cross-machine rebuild fires), not a missing fact. |

### Background
`hero resume` renders a brief whose project-context sections (`Just changed`, `In-flight`, `Tried`, `Blocked`, `Nearby`) are read out of the **graph database** (`.hero/graph.db`). Two observed failures during a content-quality eval:

1. **Same-machine:** a commit that just landed does NOT appear in the next `hero resume`'s `Just changed` section, even though the per-user handoff file's MACHINE-STATE block (which reads `git log` directly) DOES list it.
2. **Cross-machine:** a fresh `git clone` (graph.db is gitignored, doesn't travel) shows ALL project-context sections empty — the clone got the handoff singletons but none of the project graph.

### Analysis
The digest section `justChangedSection` queries graph `Commit` nodes. Those nodes are created **only** by `gitutil.WriteGitLogGraph`, which is called **only** from `hero scan` and `hero graph reingest`. Nothing on the normal commit path or session-start path creates them. So:
- Same-machine: the commit reaches git and gets a thin `post-commit` line in `events.log`, but no `Commit` graph node is ever written → `justChangedSection` returns empty.
- Cross-machine: `graph.db` is gitignored so the clone starts with an empty graph; `hero next ingest` (SessionStart) only rehydrates handoff singletons, never project-context nodes → every graph-backed section is empty.

### Root Cause
**This is one mechanism with two surfaces, not two independent bugs.** The single root cause: *the project graph (`Commit` nodes, and on a cold clone all project-context nodes) is only ever populated by `hero scan`/`hero graph reingest`, which run neither on commit nor on session start.* The digest reads a graph that the normal lifecycle never keeps current.

- Same-machine surface: the post-commit hook and `writeCheckpoint` never ingest the commit into the graph (no `WriteGitLogGraph` call).
- Cross-machine surface: a cold clone's empty graph is never rebuilt from the locally-available authoritative sources (`git log`, committed specs) on first resume; only handoff singletons are ingested.

The cross-machine half is **partly by-design**: `graph.db` is treated as a rebuildable local cache (it's gitignored precisely because `hero scan` can regenerate it from git + specs). The bug is that the rebuild step is never *auto-run* on a fresh clone, so the user lands on an empty map and isn't told to run `hero scan`. Per the mission anchor ("every session starts as smart as where the last one left off"), a fresh machine with an empty project map fails the bar regardless of design intent — so the cross-machine half is framed below as a design gap / enhancement, distinct from the clear same-machine bug.

### Source
- `internal/digest/digest.go:488` — `justChangedSection` reads `Commit` nodes from the graph.
- `internal/gitutil/graph_ingest.go:28` — `WriteGitLogGraph`, the only creator of `Commit` nodes.
- `internal/cli/hook.go:94` — `post-commit` hook logs an event line + `writeCheckpoint()`, never ingests the commit.
- `internal/cli/checkpoint.go:157` — `writeCheckpoint` projects from existing graph nodes; reads `git log` directly only for the machine block.
- `internal/cli/next_handoff.go:94` / `internal/handoff/ingest.go:29` — SessionStart `hero next ingest` rehydrates handoff singletons only.

### Fix Direction
Same-machine: ingest the commit into the graph at commit time (call `WriteGitLogGraph` from the post-commit path, or from `writeCheckpoint`). Cross-machine: on a cold/empty graph at resume or SessionStart, rebuild project context from the locally-available authoritative sources (`git log` + committed specs) — or, at minimum, detect the empty graph and tell the user to run `hero scan`. Both are low-risk because `WriteGitLogGraph` is idempotent.

---

## Problem Statement

`hero resume` produces a session-start brief. Its project-context sections are populated by `digest.Generate` reading the graph DB (`internal/cli/brief.go:77-104`). Two real reproductions from a content-quality eval expose that these sections come up empty when they should not.

### Reproduction — Observation 1 (same-machine "Just changed" empty)
1. `hero init` a fresh repo with hooks installed.
2. Make a real commit: `auth: add in-memory token-bucket rate limit on /login`.
3. Run `hero resume` on the **same** machine.
4. **Expected:** `## Just changed` lists the commit.
5. **Actual:** `## Just changed\n\n_(none)_`. The per-user file's MACHINE-STATE block (which reads `git log` directly) DID list the commit, proving the commit reached git. `.hero/events.log` had only ~1 line, confirming almost nothing was recorded to the graph during the session.

### Reproduction — Observation 2 (cross-machine fresh clone, no project context)
1. `git clone` the repo to a second folder. `graph.db` is gitignored → it does NOT travel.
2. On the clone, SessionStart fires `hero next ingest`, which rehydrates only the handoff singletons (ask / suggestion / reflection) from `.hero/next/<user>.md`.
3. Run `hero resume` on the clone.
4. **Expected:** project context (recent commits, in-flight specs, blockers) is available.
5. **Actual:** `In-flight`, `Just changed`, `Tried`, `Blocked`, `Nearby` are ALL empty. An independent judge called this "an empty map — I'm grepping before I'm typing," dragging helpfulness to 6/10.

## Environment Details
- Built `hero` binary, `hero init` repo with git hooks installed.
- `next.projected` mode (handoff is a graph projection).
- `.hero/graph.db` is gitignored (local cache); `.hero/events.log` and `.hero/next/<user>.md` ARE committed and travel.
- Tracker: `none` (session-originated diagnosis; no tracker post).

---

## Root Cause Analysis

### How a commit is *supposed* to become a graph `Commit` node — and why it doesn't on a normal commit

`Commit` nodes are created in exactly one function:

`internal/gitutil/graph_ingest.go:28` — `WriteGitLogGraph(repoDir, repoKey, limit, store)` walks `git log` and upserts a `Commit` node per SHA (type `"Commit"`, `Repo: repoKey`, props `sha/subject/date/author_name/author_email/file_count`), plus a `Person` node and `authored_by` edge. It is idempotent (graph_ingest.go:26-27).

The **only** callers (confirmed by grep across `internal/` and `cmd/`):
- `internal/cli/scan.go:557` — `hero scan`.
- `internal/cli/graph_memory.go:212` — `hero graph reingest`.

Neither runs on a normal commit or at session start. Walking the actual commit path:

- `internal/cli/hook.go:94-104` — the `post-commit` hook does two things: (a) `hooks.LogEvent(eventsLogPath, {"event":"post-commit","sha":sha})` — a thin event line, **not** a graph node; (b) `writeCheckpoint()`. Neither calls `WriteGitLogGraph`.
- `internal/cli/checkpoint.go:157` — `writeCheckpoint` projects NEXT.md/SNAPSHOT and the per-user handoff file **from already-recorded graph nodes**, and reads `git log` directly via `recentCommits(projectRoot, 5)` (checkpoint.go:732) only to build the MACHINE-STATE block. It never ingests commits into the graph. **This is why the machine block shows the commit but the graph-backed `Just changed` does not** — two different data paths reading two different sources.

So a commit lands in git and in `events.log` (as a bare `{event,sha,t}` line) but **never becomes a `Commit` graph node** until someone runs `hero scan`. That is Observation 1's exact mechanism.

### What `digest.justChangedSection` queries — confirming the empty result

`internal/digest/digest.go:488-523`:

```sql
SELECT json_extract(props,'$.sha'), json_extract(props,'$.subject'),
       json_extract(props,'$.author_name'), json_extract(props,'$.date')
  FROM nodes
 WHERE type = 'Commit' AND repo = ? AND valid_to IS NULL
 ORDER BY json_extract(props,'$.date') DESC
 LIMIT 30
```

- Filter is `type = 'Commit' AND repo = ?` where `? = opts.RepoKey = gitutil.RepoKey(projectRoot)` (`internal/cli/brief.go:94`).
- **No author/email filter** — so this is NOT the author-slug-mismatch class (cf. `cross-machine-handoff-slug-mismatch`). The section is empty purely because **no `Commit` nodes exist**, not because they're filtered out.
- **Repo key is consistent** between writer and reader: `WriteGitLogGraph` is called with `repoKey := gitutil.RepoKey(projectRoot)` (`scan.go:512`) and the digest filters on the same `gitutil.RepoKey(projectRoot)` (`brief.go:94`). I verified this to rule out a key-mismatch red herring. `RepoKey` prefers `git remote get-url origin` and falls back to the dir basename (`internal/gitutil/gitutil.go:209`), so a fresh clone with the same origin produces the same key — portability is fine; the problem is simply that nothing wrote the nodes.

### Cross-machine: does anything rebuild graph.db from the committed events.log?

No. Two facts confirm Observation 2:

1. `internal/graph/graph.go:59-87` — `graph.Open` just opens the SQLite file and runs schema migrations. **There is no replay of `events.log`** on open. A fresh clone (graph.db gitignored) gets an empty DB.
2. `internal/cli/next_handoff.go:94-146` → `internal/handoff/ingest.go:29` (`IngestUserFile`) — SessionStart `hero next ingest` parses `.hero/next/<user>.md` and re-creates only `UserAsk` / `NextSuggestion` / `SessionReflection` nodes (ingest.go:99-123). It never calls `WriteGitLogGraph`, `spec.WriteGraph`, or any project-context ingest.

Additionally, even if a replay existed, `events.log` is **not a sufficient source** for rich `Commit` nodes: the `post-commit` line is only `{event:"post-commit", sha, t}` (`internal/hooks/hooks.go:149-167` writes whatever map it's given; hook.go:96-99 gives it just event+sha). Subject / author / date / file_count are absent. The authoritative source for `Commit` nodes is **`git log` itself**, which is fully present in any clone. So the correct cross-machine rebuild reads `git log` + committed specs locally — not `events.log`.

**Conclusion on Observation 2's classification:** it is *partly by-design* (graph.db is a rebuildable local cache, deliberately gitignored) and *partly a gap* (the rebuild is never auto-run, and resume doesn't even tell the user to run `hero scan`). Framed below as a design gap / enhancement distinct from the same-machine bug.

### One root cause or two?
**One root cause, two surfaces.** The single underlying defect: *the project graph is only populated by `hero scan`/`hero graph reingest`, which run on neither commit nor session start, so the digest reads a graph the normal lifecycle never keeps current.* Same-machine and cross-machine are the same hole seen from two angles (a stale graph vs. an empty graph). **Recommendation: keep ONE spec** (this one) that diagnoses both, because the fix touches the same ingest function (`WriteGitLogGraph`) and the same "keep the project graph current" responsibility. The two *fixes* differ in trigger (commit-time vs. cold-clone-time) and can land as two changes within this spec, but they should not be split into two specs — splitting would fracture a single coherent responsibility and risk one half regressing the other.

---

## Code Flow (End to End)

### Observation 1 — same-machine commit never reaches the graph
1. User commits. Git fires the `post-commit` hook.
2. `internal/cli/hook.go:94-99` — `runHook` case `"post-commit"`: `hooks.LogEvent(eventsLogPath, {"event":"post-commit","sha":sha})` writes a thin line to `.hero/events.log`. **No graph write.**
3. `internal/cli/hook.go:104` — `writeCheckpoint()` runs.
4. `internal/cli/checkpoint.go:157-271` — projects NEXT.md / per-user handoff from existing graph nodes; `recentCommits(projectRoot,5)` (checkpoint.go:732) reads `git log` directly for the MACHINE-STATE block. **No `WriteGitLogGraph`.**
5. Later, `hero resume` → `internal/cli/brief.go:77-104` opens graph, calls `digest.Generate`.
6. `internal/digest/digest.go:488` — `justChangedSection` queries `Commit` nodes → **0 rows** (none were ever written) → `## Just changed` renders empty.

### Observation 2 — fresh clone has no project graph
1. `git clone`. `.hero/graph.db` is gitignored → not present; `.hero/events.log` and `.hero/next/<user>.md` travel.
2. SessionStart fires `hero next ingest` (`internal/install/claude_hooks.go:50-61`).
3. `internal/cli/next_handoff.go:94-146` → `internal/handoff/ingest.go:29` (`IngestUserFile`) rehydrates `UserAsk` / `NextSuggestion` / `SessionReflection` only.
4. `hero resume` → `internal/cli/brief.go:77-104` opens the (empty) graph, calls `digest.Generate`.
5. `internal/digest/digest.go` — every graph-backed section (`justChangedSection`, in-flight, tried, blocked, nearby) queries an empty `nodes` table → all empty.
6. Nothing has run `WriteGitLogGraph` / `spec.WriteGraph` against the clone's local `git log` + committed specs, so the project map stays blank until the user manually runs `hero scan` (which they aren't prompted to do).

---

## Key Files

### Digest / brief (read side)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/digest/digest.go` | 488–523 | `justChangedSection` — reads `Commit` nodes by `type+repo`; empty when no nodes exist. Confirms no author filter. |
| `internal/cli/brief.go` | 77–104 | `hero resume` opens graph, calls `digest.Generate` with `RepoKey = gitutil.RepoKey(projectRoot)`. |

### Commit → graph ingest (write side / the gap)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/gitutil/graph_ingest.go` | 28–220 | `WriteGitLogGraph` — sole creator of `Commit` nodes; idempotent. Stamps `Repo: repoKey`. |
| `internal/cli/scan.go` | 512, 557 | `hero scan` — one of two callers; computes `repoKey := gitutil.RepoKey(projectRoot)`. |
| `internal/cli/graph_memory.go` | 189–212 | `hero graph reingest` (`reingestWork`) — the other caller; rebuilds specs+sessions+git-log+next into the graph. |

### Commit / session-start lifecycle (where the wire-up is missing)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/hook.go` | 94–104 | `post-commit` hook — logs thin event + `writeCheckpoint`; never ingests the commit. |
| `internal/cli/checkpoint.go` | 157–271, 726–777, 926 | `writeCheckpoint` projects from graph + reads `git log` for machine block only. `recentCommits` is the direct-git read. |
| `internal/cli/next_handoff.go` | 94–146 | `hero next ingest` (SessionStart) — handoff singletons only. |
| `internal/handoff/ingest.go` | 29–123 | `IngestUserFile` — parses ask/suggestion/reflection; no project-context nodes. |
| `internal/install/claude_hooks.go` | 50–61 | Hook wiring: SessionStart→`hero next ingest`, Stop/PreCompact→`hero next checkpoint`. |
| `internal/graph/graph.go` | 59–87 | `graph.Open` — opens SQLite + migrates; no `events.log` replay. |
| `internal/hooks/hooks.go` | 149–167 | `LogEvent` — writes the thin `events.log` line; not a graph write, insufficient to rebuild Commit nodes. |

---

## Secondary Defects

- **Thin post-commit event line.** `internal/cli/hook.go:96-99` records only `{event, sha}` for `post-commit`. Even if a future `events.log`→graph replay were built, it could not reconstruct rich `Commit` nodes (no subject/author/date/file_count). Not the reported bug, but it forecloses "rebuild from events.log" as a path — the rebuild must read `git log`. Worth noting so a future fix doesn't chase the events.log angle.
- **Empty-graph silence.** `hero resume` on a cold graph produces empty sections with no hint to run `hero scan`. Even before a full auto-rebuild lands, resume should detect an empty/near-empty graph and surface a one-line "run `hero scan` to populate project context" nudge. Low-effort guardrail; prevents the "empty map" UX.

---

## Notes

### Load-bearing claims (read vs. assumed)
- `Commit` nodes created only by `WriteGitLogGraph` — **read** (grep across `internal/`+`cmd/`; only `scan.go:557`, `graph_memory.go:212`).
- post-commit hook does not ingest commits — **read** (`hook.go:94-104`).
- `writeCheckpoint` reads git directly for machine block, never ingests — **read** (`checkpoint.go:157-271,732`).
- `justChangedSection` filters `type='Commit' AND repo=?`, no author filter — **read** (`digest.go:488-500`).
- writer/reader repo keys match (`gitutil.RepoKey`) — **read** (`scan.go:512`, `brief.go:94`, `gitutil.go:209`).
- `graph.Open` does not replay events.log — **read** (`graph.go:59-87`).
- SessionStart ingest = handoff singletons only — **read** (`next_handoff.go:94-146`, `ingest.go:29-123`).
- events.log post-commit line is `{event,sha,t}` only — **read** (`hook.go:96-99`, `hooks.go:149-167`).

### Relation to prior bugs
- `cross-machine-handoff-slug-mismatch` (planning) is a *different* root cause: it's about the per-user handoff file loading empty when the local user slug differs between machines (an author/slug filter mismatch). This bug is about project-context nodes (`Commit`/spec/blocker) being absent entirely. They share the cross-machine surface but not the mechanism — keep them separate.
- `resume-brief-surfaces-handoff` (completed) closed the load half for handoff *singletons*; this bug is the project-context companion gap on the same brief.

---

## Goal
`hero resume` surfaces real project context in both situations: (1) a commit made during a session appears in the next resume's `Just changed`; (2) a fresh clone's first resume surfaces project context (commits, in-flight specs, blockers) rebuilt from the locally-available authoritative sources (`git log` + committed specs) — or, if the cold rebuild is deferred as a separate enhancement, resume at minimum detects the empty graph and tells the user the one command that restores context.

## Acceptance Criteria
- WHEN a commit is made during a session on a hooks-installed repo THE SYSTEM SHALL record it as a graph `Commit` node so the next `hero resume`'s `Just changed` section lists that commit.
- WHEN `hero resume` runs on a fresh clone whose `graph.db` is absent THE SYSTEM SHALL surface project context (recent commits and in-flight specs) rebuilt from the clone's local `git log` and committed specs.
- IF the cold-graph rebuild is deferred to a follow-up THEN `hero resume` SHALL detect the empty/near-empty graph and print a one-line nudge naming `hero scan` as the command that restores project context.
- THE SYSTEM SHALL keep commit ingest idempotent — repeated checkpoints/resumes SHALL NOT create duplicate `Commit` nodes (already guaranteed by `WriteGitLogGraph`).
- THE SYSTEM SHALL NOT regress the handoff-singleton continuity round-trip (A→B continuity stays green).

## Suggested Fix Approach

Two changes within this one spec. They share `WriteGitLogGraph` but differ in trigger. Recommended ordering: same-machine first (smaller, higher-confidence), then cross-machine.

### Change 1 — same-machine: ingest the just-made commit into the graph at checkpoint time

Wire a bounded `WriteGitLogGraph` call into the commit/checkpoint path so a commit becomes a `Commit` node before the next resume reads the graph. Preferred location is `writeCheckpoint` (runs on `post-commit` AND on every Stop/PreCompact, so it also backfills commits made outside the git hook), using a small `limit` to stay cheap.

In `internal/cli/checkpoint.go`, inside `writeCheckpoint` (after the graph is available / alongside the existing projection work), add an idempotent ingest of the most recent N commits:

**Before** (`writeCheckpoint` reads git only for the machine block; nothing ingests commits):
```go
// recentCommits(projectRoot, 5) feeds the MACHINE-STATE block only.
// No WriteGitLogGraph call anywhere in the checkpoint path.
```

**After** (sketch — ingest recent commits into the graph; idempotent, errors swallowed so checkpoint never fails):
```go
// Keep the project graph current so the next resume's `Just changed`
// reflects commits made this session. Bounded + idempotent.
if store, err := graph.Open(heroDir); err == nil {
    repoKey := gitutil.RepoKey(projectRoot)
    if _, gerr := gitutil.WriteGitLogGraph(projectRoot, repoKey, 50, store); gerr != nil {
        fmt.Fprintf(os.Stderr, "warning: commit graph ingest failed: %v\n", gerr)
    }
    store.Close()
}
```

**Why:** `justChangedSection` reads `Commit` nodes. Today they're written only by `hero scan`. Ingesting at checkpoint time closes the same-machine gap: the commit that just landed (and any made outside the hook) becomes a node before the next resume. `WriteGitLogGraph` is idempotent (graph_ingest.go:26-27), so repeated checkpoints don't duplicate. A `limit` of ~50 keeps `git log` cheap on every turn. (Implementer's call whether to place this in `writeCheckpoint` or directly in the `post-commit` case of `internal/cli/hook.go:94`; `writeCheckpoint` is preferred because it also covers non-hook commit flows.)

### Change 2 — cross-machine: rebuild project context on a cold graph (or nudge)

On a fresh clone the graph is empty and only handoff singletons get ingested. Two options; recommend 2a, with 2b as the minimum-viable guardrail if 2a is deferred.

**Option 2a (recommended): auto-rebuild project context on SessionStart / first resume when the graph is empty.** Extend the SessionStart path so that, after `IngestUserFile`, if the graph has no project-context nodes, it rebuilds them from local authoritative sources — the same work `reingestWork` already does (`internal/cli/graph_memory.go:189-212`): `spec.WriteGraph` + `sessions.WriteGraph` + `WriteGitLogGraph` + NEXT ingest. Gate it on an empty/near-empty graph so it only pays the cost on a cold clone, not every session.

Sketch (in the SessionStart ingest path, `internal/cli/next_handoff.go` or a helper it calls):
```go
// Cold-clone rebuild: graph.db is a gitignored local cache. On a fresh
// clone it's empty — rebuild project context from local git log + specs
// so the first resume isn't an empty map. Only fires when empty.
if projectGraphEmpty(store, repoKey) {
    _ = rebuildProjectContext(cfg, projectRoot, heroDir, store) // specs + sessions + git-log
}
```

**Option 2b (minimum guardrail if 2a is deferred): detect empty graph in `hero resume` and nudge.** In `internal/cli/brief.go` after opening the graph, if project-context node counts are ~0, print a one-line hint: `run \`hero scan\` to populate project context (commits, specs, blockers) for this clone`. This satisfies the deferred-path acceptance criterion and removes the silent "empty map."

**Why:** `graph.db` is deliberately gitignored as a rebuildable cache (so committing it would create cross-machine merge conflicts). The defect is that the rebuild never auto-fires on a cold clone. Reading `git log` + committed specs locally is the correct source (events.log is too thin to reconstruct rich Commit nodes — see Secondary Defects). 2a fulfills the mission anchor ("start as smart as where the last one left off"); 2b is the honest fallback that at least tells the user how to restore context.

### What NOT to do
- Do NOT attempt to rebuild `Commit` nodes from `events.log` — the `post-commit` line is `{event,sha}` only and cannot reconstruct subject/author/date. Rebuild from `git log`.
- Do NOT commit `graph.db` to make it travel — it's gitignored on purpose (merge-conflict avoidance); the rebuild-from-local-sources path is the intended design.
- Do NOT add an author/email filter to `justChangedSection` — it deliberately shows all authors' recent commits.

### Delivered (files changed)

- `internal/cli/checkpoint.go` — Change 1: `writeCheckpoint` now calls `ingestRecentCommits(projectRoot, heroDir)` after the snapshot projection; new helper opens the graph and runs `gitutil.WriteGitLogGraph(projectRoot, gitutil.RepoKey(projectRoot), 50, store)` (bounded, idempotent, errors swallowed to stderr).
- `internal/cli/next_handoff.go` — Change 2a: `runNextIngest` calls `rebuildProjectContextIfCold(...)` BEFORE the handoff-file loop (so it fires even on a cold clone with no per-user files). New helpers `rebuildProjectContextIfCold` (specs + sessions + git-log rebuild, keyed on `gitutil.RepoKey` to match the digest reader) and `projectGraphCold` (repo-scoped count of `Commit`/`Feature`/`Bug` nodes).
- `internal/cli/brief.go` — Change 2b: `runResume` prints a one-line `hero scan` nudge to stderr when `projectGraphCold` is true, so a cold graph never renders a silent empty map.
- `internal/cli/resume_project_context_test.go` (new) — same-machine commit-in-graph, idempotence, cross-machine cold rebuild, and cold-graph nudge tests.

## Test Plan

### Existing test review
- `internal/gitutil/graph_ingest_test.go` (if present) — covers `WriteGitLogGraph` node/edge creation and idempotency. Confirm the idempotency case before relying on it for repeated-checkpoint safety.
- `internal/cli/checkpoint_test.go` — covers `writeCheckpoint` projection behavior; the new ingest must not break existing projection assertions.
- `internal/cli/handoff_continuity_test.go` — the A→B continuity guardrail for handoff singletons; must stay green after both changes (the singleton round-trip is the protected "magic").
- `internal/digest/*_test.go` — digest section rendering; add/extend a `justChangedSection` case.

### Test changes needed
1. **Same-machine commit-in-graph (Change 1):** in a temp git repo, init Hero, make a commit, run `writeCheckpoint` (or invoke the `post-commit` hook path), then assert a `Commit` node with the commit's SHA/subject exists for the repo key, and that `justChangedSection` returns a non-empty line containing the subject.
2. **Idempotency:** run the checkpoint twice; assert the `Commit` node count for the SHA is exactly 1.
3. **Cross-machine cold rebuild (Change 2a):** simulate a clone — temp repo with `git log` + committed specs but NO `graph.db`. Run the SessionStart ingest path; assert project-context nodes (Commit + in-flight Spec) are rebuilt and `digest.Generate` returns non-empty `Just changed` and in-flight sections.
4. **Empty-graph nudge (Change 2b, if 2a deferred):** with an empty graph, assert `hero resume` output contains the `hero scan` nudge string.
5. **Continuity regression:** re-run the A→B handoff-singleton continuity test to confirm ask/suggestion/reflection still round-trip.

### Regression scope
- `writeCheckpoint` runs on every Stop/PreCompact/post-commit — adding a `git log` + graph write there touches the hottest path. Verify it stays cheap (bounded `limit`, swallowed errors, never fails the checkpoint) and that `writeFileIfChanged`/projection assertions are unaffected.
- The cold-rebuild gate must be a true "empty graph" check, not "every session," or it adds `git log` + spec-walk cost to every SessionStart on warm machines.
- Confirm no duplicate `Commit` nodes across `hero scan` + the new checkpoint ingest (both call the same idempotent function with the same repo key — verified consistent).

## Boundaries
- Does NOT change the digest's query semantics beyond what's needed (no new author filter, no new section).
- Does NOT address `cross-machine-handoff-slug-mismatch` (separate root cause — handoff-singleton slug filter, not project-context nodes).
- Does NOT make `graph.db` travel with git (out of scope and contrary to design).
- Does NOT redesign `events.log` into a graph-replay log; the thin-event-line observation is logged as a Secondary Defect, not fixed here.
- Whether Change 2a ships now or is deferred to a follow-up (with 2b as the interim) is a delivery-time scope decision; this spec diagnoses both and recommends 2a.

## Risks
- **Hot-path cost:** ingesting `git log` on every checkpoint adds a `git log` exec + upserts to the Stop/PreCompact path. Mitigate with a small `limit` and swallowed errors. Measure on a large-history repo.
- **Cold-rebuild over-firing:** if the "empty graph" gate is wrong, warm machines pay a full reingest every session. The gate must be precise.
- **Continuity regression:** any change near `writeCheckpoint` / SessionStart ingest risks the handoff-singleton round-trip. The continuity guardrail test is the gate — it must stay green.

## Validation
- Reproduce Observation 1: fresh repo, commit, `hero resume` → `Just changed` now lists the commit.
- Reproduce Observation 2: clone (no graph.db), open session, `hero resume` → project context present (2a) or the `hero scan` nudge appears (2b).
- Run `internal/cli/handoff_continuity_test.go` and the digest tests; all green.
- Confirm no duplicate `Commit` nodes after `hero scan` followed by several checkpoints.

## Kickoff

Fixes `hero resume`'s empty project-context sections — commits never become graph `Commit` nodes on a normal commit, and a fresh clone's graph is empty (graph.db is gitignored, never rebuilt).

**Status:** planning — root-caused to one mechanism, two surfaces: `WriteGitLogGraph` is the only creator of `Commit` nodes and runs only on `hero scan`/`graph reingest`, never on commit or session start.

**Pick up at:** start with Change 1 (same-machine) — wire a bounded, idempotent `gitutil.WriteGitLogGraph(projectRoot, repoKey, 50, store)` into `writeCheckpoint` so the just-made commit becomes a node before the next resume. Then Change 2 (cross-machine cold rebuild or `hero scan` nudge).

→ `.hero/planning/bugs/resume-brief-missing-project-context/spec.md`

**Files:** `internal/cli/checkpoint.go:157`, `internal/gitutil/graph_ingest.go:28`, `internal/digest/digest.go:488`, `internal/cli/next_handoff.go:94`, `internal/cli/brief.go:77`
**Skip:** rebuilding Commit nodes from `events.log` (too thin — `{event,sha}` only); committing graph.db; adding an author filter to `justChangedSection`.
