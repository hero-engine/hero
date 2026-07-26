---
title: "Semantic embeddings never refresh on commit — the vector index only moves when someone runs `hero scan` by hand"
slug: embeddings-never-refresh-on-commit
type: bug
status: planning
domain: engineering
priority: high
severity: high
root_cause_class: code
tags: [embeddings, retrieval, semantic-search, git-hooks, staleness, harness-agnostic]
created: 2026-07-25
---

# Semantic embeddings never refresh on commit

## Kickoff

Paste into a fresh session to start delivery:

> Deliver `embeddings-never-refresh-on-commit`. The vector index in
> `vec_chunks` only updates when a human runs `hero scan` or
> `hero embeddings rebuild` — no git hook, no automation. This repo's
> embeddings are frozen at 2026-07-16 while HEAD is 2026-07-25 (76 commits,
> 615 files). `embedded-inference` AC-9 designed the pre-commit trigger and
> its own ledger marks it PARTIAL; the spec was closed anyway. Fix: make the
> no-op refresh cheap (hash-check *before* `model.Embed`, currently line 73-74
> of `refresh.go` embeds every chunk unconditionally), add
> `hero embeddings refresh --if-stale --deadline`, wire it into the managed
> **git** pre-commit block, and add a `hero check` staleness row. Start by
> reading **Key Files**, then work the Acceptance Criteria in order. Close
> with the cold delivery audit and `hero spec verify`.

**Status:** planning — investigation complete, root cause confirmed with
measurements; no code written.

**Pick up at:** AC-1 first (move the `textHash` comparison ahead of
`model.Embed` in `internal/embeddings/refresh.go`). Everything else depends
on the no-op refresh being cheap enough to sit in a git hook.

→ `.hero/planning/bugs/embeddings-never-refresh-on-commit/spec.md`

**Files:** `internal/embeddings/refresh.go:69`, `internal/embeddings/storage.go:80`,
`internal/cli/next_hooks.go:314`, `internal/cli/embeddings.go:39`,
`internal/cli/check.go:223`
**Skip:** a Claude Stop/PreCompact hook (4 of 6 harness targets have no session
hook at all — tripwire `harness-changes-cover-all-targets`); a watcher daemon;
extending `Refresh`'s `scope []string` to file-level scoping.

---

## Summary

### Categorization

| Attribute | Assessment |
|-----------|------------|
| **Criticality** | high — semantic retrieval is one of Hero's two ranking legs. A frozen vector index silently degrades every `hero ask`, `hero search --hybrid`, and every context injection built on them. It fails *quietly*: results still come back, they're just blind to everything recent. |
| **Ease of Fix** | moderate — the trigger itself is one line in a hook template, but it is only safe to add after the no-op refresh path is made cheap and the prune-on-empty hazard is closed. Roughly 4 files, no schema change. |
| **Caused by our codebase?** | Yes — `embeddings.Refresh` has exactly two call sites, both manual commands. The automatic trigger was designed, acknowledged as undelivered, and never built. |
| **Needs more research?** | No — root cause confirmed by code reading, live database inspection, and timing measurements against a cloned workspace. One minor observation (chunk-ID discrepancy, Secondary Defect 8) is unexplained and flagged as such. |

### Background

Hero embeds specs, knowledge, and code symbols into `vec_chunks` inside
`index.db` and uses them for the semantic half of hybrid retrieval. The
embedding refresh is supposed to run automatically so the index tracks the
repository. It does not. It runs only when a human types `hero scan` or
`hero embeddings rebuild`.

In this repository the consequence is severe and measurable: every row in
`vec_chunks` carries `embedded_at` no later than **2026-07-16T21:33:00Z**,
while HEAD is **2026-07-25** — nine days, 76 commits, 615 files changed, 230
of them `.go`. Files created today have zero chunks. Specs delivered today
have zero chunks. Semantic search cannot see any of it.

### Analysis

`embeddings.Refresh` (`internal/embeddings/refresh.go:32`) is called from
exactly two places, both of which require a human to type a command:

- `internal/cli/embeddings.go:141` — `hero embeddings rebuild`
- `internal/cli/scan.go:412` — `hero scan`

The installed git pre-commit hook runs `hero next checkpoint -q`,
`hero index --if-stale -q`, `hero queue write -q`, then re-stages the
projected handoff files. It never touches embeddings. There is no
post-commit hook at all. No other automation — no daemon, no watcher, no
harness hook — invokes `Refresh`.

So the FTS5/lexical index *is* kept current on every commit (that is what
`hero index --if-stale` does), and the vector index sitting in the same
`index.db` file is not. Hybrid retrieval therefore fuses a fresh lexical leg
with a nine-day-stale semantic leg, and nothing anywhere reports the skew.

### Root Cause

**The automatic refresh trigger was specified, was known at delivery time to
be missing, and the spec was closed as completed anyway.**

`.hero/specs/embedded-inference/spec.md` is `status: completed`. Its own
Completion Ledger contains:

> | AC-9 | Pre-commit hook calls embeddings refresh --if-stale | **PARTIAL** |
> `internal/cli/scan.go` calls `embeddings.Refresh()` in `hero scan`.
> Pre-commit hook template (`next_hooks.go` line 316) does NOT include
> `hero embeddings refresh --if-stale` — hook only runs `hero index --if-stale`
> + `hero queue write` + NEXT staging. The `--if-stale` fast-path for
> pre-commit is spec'd but not wired into the hook script. |

and AC-9 itself reads:

> WHEN the pre-commit hook fires THE SYSTEM SHALL run
> `embeddings refresh --if-stale`, completing in <100ms when no files changed.

The predecessor spec `embeddings-index` (now `superseded`) says the same
thing at line 129: *"Git pre-commit hook (extends the existing
`internal/cli/next_hooks.go` pre-commit) calls `hero embeddings refresh
--if-stale`."*

So the missing code is not an oversight of design. It is a delivery gap that
was written down, shipped as PARTIAL, and lost when the spec closed. Two
concrete artifacts are absent today:

1. `hero embeddings refresh` — the subcommand does not exist. `embeddingsCmd`
   registers only `status` and `rebuild` (`internal/cli/embeddings.go:39-42`).
2. The hook line — `hookScript("pre-commit")`
   (`internal/cli/next_hooks.go:314`) emits `indexRefresh` and `queueRefresh`
   and nothing for embeddings.

Classified `code` (a specific, nameable line is absent from a specific
function). The **contributing cause is process**: `hero spec verify` allowed
`embedded-inference` to close with an acceptance criterion self-reported as
PARTIAL. That is out of scope here but is the reason a designed requirement
evaporated — see Notes.

### Source

`internal/embeddings/refresh.go` (the engine, and the unconditional-embed
cost problem), `internal/cli/next_hooks.go` (the hook template that should
call it), `internal/cli/embeddings.go` (the missing subcommand),
`internal/embeddings/storage.go` (the prune-on-empty hazard and per-chunk
transactions), `internal/cli/check.go` (where the missing staleness signal
belongs).

### Fix Direction

Make the incremental refresh genuinely cheap when nothing changed, expose it
as `hero embeddings refresh --if-stale` with a self-enforced wall-clock
deadline, call it from the **git** pre-commit hook (harness-agnostic, so it
covers all six install targets), and add a `hero check` row that reports when
the vector index lags HEAD so this can never again rot invisibly for nine
days.

---

## Problem Statement

### Reported symptom

The semantic embedding index does not track the repository. New code and new
specs are invisible to semantic search until someone manually re-scans.

### Reproduction — this repository, as of 2026-07-25

`hero embeddings status` against the live workspace:

```
Embeddings enabled: true
Model: hero-embed-v1
Scope: [spec knowledge convention event code]
Model status: loaded (dim=256)

Index stats:
  Total chunks: 6912
  code:          2927
  knowledge:     45
  spec:          3940
```

Direct query against `.hero/index.db`:

```sql
SELECT corpus, COUNT(*), MIN(embedded_at), MAX(embedded_at)
FROM vec_chunks GROUP BY corpus;
```

```
code       | 2927 | 2026-07-11T23:01:02Z | 2026-07-16T21:33:00Z
knowledge  |   45 | 2026-05-30T15:04:06Z | 2026-07-16T21:33:00Z
spec       | 3940 | 2026-05-30T15:04:04Z | 2026-07-16T21:33:00Z
```

All three corpora share the identical `MAX(embedded_at)` of
**2026-07-16T21:33:00Z** — the signature of one manual `hero scan` and
nothing since. (`MIN` is older because `Upsert` only rewrites
`embedded_at` when the text hash changes, so untouched chunks keep their
original stamp. See the staleness-guard design, which depends on this.)

Since that timestamp: **9 days, 76 commits, 615 files changed, 230 `.go`
files.**

Files created today have **zero** chunks:

- `internal/spec/declared_children.go`
- `internal/spec/relation_keys.go`
- `internal/graph/node_identity_test.go`
- `internal/peering/receive_graph_test.go`

Specs *delivered* today have **zero** chunks:

- `graph-node-identity-repo-scoped`
- `initiative-autocomplete-ignores-declared-children`

The source data is stale too, not just its embeddings. `Symbol` nodes in
`graph.db` — which is where the `code` corpus is extracted from — carry
`MAX(ingested_at) = 2026-07-16T21:32:58Z`, and a query for symbols whose
`$.path` matches `declared_children` returns **0**. So refreshing embeddings
alone would re-embed the same nine-day-old symbols. This matters for the
design and is addressed in Delta Scoping below.

### Why nothing warns

`hero check` has no embeddings row. `hero embeddings status` prints chunk
counts but never prints `embedded_at` or compares it to anything.
`retrieveHybrid` (`internal/retrieval/retrieval.go:683`) embeds the query and
calls `QuerySimilar` with no freshness consideration at all — a stale index
returns confident, well-ranked, wrong-era results.

---

## Environment Details

- Repo: `/Users/developer/projects/hero-engine/repository/hero`, Go, `main` @ 2026-07-25.
- `.hero/hero.json` has **no** `embeddings` block, so all defaults apply:
  enabled = `true`, model = `hero-embed-v1`, scope =
  `[spec knowledge convention event code]` (`internal/config/config.go:225-243`).
- Model is embedded in the binary via `//go:embed`
  (`internal/embeddings/defaultmodel/`): `weights.bin` 24,263,680 bytes,
  `vocab.txt` 180,644 bytes, dim 256 → vocab ≈ 23,695 rows.
- `index.db` opens with `busy_timeout(5000)` and `journal_mode(WAL)`
  (`internal/index/index.go:88`). The 5-second busy timeout is load-bearing
  for the failure design below.
- Installed `.git/hooks/pre-commit` matches `hookScript("pre-commit")`; no
  `.git/hooks/post-commit` exists.

### Measurements

All timings taken against a clone of `.hero` plus the Go sources in a
scratch directory, using a binary built from `./cmd/hero` at HEAD. Machine:
darwin 25.5.0.

| Measurement | Result |
|---|---|
| Full rebuild, 7,512 chunks from empty (`hero embeddings rebuild`) | **1.606s** |
| First `hero scan` after clone (embeddings step: added=3310, skipped=7420, pruned=19) | **940ms** |
| Steady-state `hero scan`, no changes (added=20, skipped=10710) | **571ms / 578ms** |
| Same, scope=`[code]` only (skipped=6113) | **102ms / 103ms** |
| Same, scope=`[spec]` only (added=20, skipped=4549) | **424ms / 425ms** |
| Same, scope=`[convention]` only (0 chunks — pure `spec.Discover` walk) | **49ms** |
| `hero embeddings status` warm, whole process incl. model load, ×20 | **0.468s total ≈ 23ms each** |
| `hero embeddings status` first exec (cold page cache, 58MB binary) | **0.60s** |
| Whole `hero scan`, steady state | **2.81s / 2.92s** |

**Model load is not the per-commit bottleneck.** The entire
`hero embeddings status` process — config load, model load (23,695 × 256
float32 decode), index open, stats query, print — completes in ~23ms warm.
The only cold-start cost is paging in the 58MB binary on first exec after a
reboot or binary upgrade (~0.6s), which is a one-time cost per boot and fits
inside the deadline proposed below. This directly answers the concern that
model load would blow the sub-second budget: it will not.

**What *does* cost is embedding text that did not change.** Subtracting the
49ms `spec.Discover` walk from the 424ms spec-only pass leaves ~375ms spent
hashing, embedding and hash-checking 4,569 unchanged spec chunks — roughly
82µs per chunk, of which the `Embed` call is the dominant term (code chunks,
truncated to 500 chars of body, cost only ~17µs each). That work is 100%
discarded. See Secondary Defect 1 — this is the single change that gets the
no-op refresh comfortably under the hook budget.

---

## Root Cause Analysis

### Confirmed

**C1. `Refresh` has no automatic caller.** `grep` over the tree returns
exactly two call sites, `internal/cli/embeddings.go:141` (`hero embeddings
rebuild`) and `internal/cli/scan.go:412` (`hero scan`). Both are
user-initiated commands.

**C2. No git hook mentions embeddings.** The managed block is generated by
`hookScript(kind)` at `internal/cli/next_hooks.go:314-349`. It emits
`indexRefresh` (`hero index --if-stale -q`) and `queueRefresh`
(`hero queue write -q`) for `pre-commit`, and nothing for embeddings.
`installHooks` writes only `pre-commit` and `post-merge`
(`next_hooks.go:260-267`) — there is no post-commit hook to write to.

**C3. `hero embeddings refresh` does not exist.** `init()` at
`internal/cli/embeddings.go:39-42` registers `status` and `rebuild` only.
The hook line specified by `embedded-inference` AC-9 could not have been
added even if someone had tried — the command it calls was never built.

**C4. The requirement was known-missing at close.** `embedded-inference` is
`status: completed` with AC-9 marked `PARTIAL` in its own Completion Ledger,
with an accurate description of exactly this gap. (Full quote in Root Cause
above.)

**C5. `Refresh`'s `scope` is corpus-scoped, not file-scoped.** At
`refresh.go:41-44` the `scope []string` is turned into a `map[string]bool`
and checked at line 60 against `ext.name` — one of the five literal corpus
names `spec | knowledge | convention | event | code`. There is no path-level
filtering anywhere in the function or in any extractor. `ChunkSpecs` calls
`spec.Discover(heroDir)` and walks everything; `ChunkCodeSymbols` runs
`SELECT ... FROM nodes WHERE valid_to IS NULL AND type = 'Symbol'` with no
predicate. The API as it stands cannot express "only these changed files."

**C6. The `code` corpus is stale at its source, not only in its embeddings.**
`ChunkCodeSymbols` reads `Symbol` nodes from `graph.db`, which are written by
`internal/codescan/graph_ingest.go:155` during `hero scan`. Live graph:
2,927 symbols, `MAX(ingested_at) = 2026-07-16T21:32:58Z`, zero symbols for
files created today. Re-running only the embedding pass would faithfully
re-embed nine-day-old symbols.

### Hypothesis, not yet confirmed

**H1.** A full rebuild reported `added=7512` while the resulting table held
7,502 rows, implying ~10 chunk IDs collide and are upserted twice. Duplicate
spec slugs were ruled out (0 duplicates across `.hero/planning` and
`.hero/specs`) and `Symbol` keys are unique (2,927 of 2,927 distinct). Most
likely candidates are sibling/satellite spec ingest or knowledge relpath
collisions. Recorded as Secondary Defect 8; not load-bearing for this fix.

---

## Code Flow (End to End)

The path that *should* fire on every commit, and where it stops:

1. `git commit` → `.git/hooks/pre-commit` runs the hero-managed block.
2. Hook body, generated by `internal/cli/next_hooks.go:314` `hookScript("pre-commit")` — runs `hero next checkpoint -q`, `hero index --if-stale -q`, `hero queue write -q`, then per-path `git add` of the projected handoff files. **This is where the flow ends. There is no embeddings call.**
3. `internal/cli/index.go:52` — `hero index --if-stale` reaches `index.RefreshIfStale(heroDir)`, which syncs the FTS5/lexical tables in `index.db`. It does not touch `vec_chunks`.
4. `internal/embeddings/refresh.go:32` `Refresh(...)` — never reached from step 2. Only reachable via `hero scan` / `hero embeddings rebuild`.
5. `internal/embeddings/refresh.go:51-57` — builds the five corpus extractors; `refresh.go:60` skips any corpus not in `scope`.
6. `internal/embeddings/chunker.go:31` `ChunkSpecs` → `spec.Discover(heroDir)`, whole-corpus walk, one chunk per spec section.
7. `internal/embeddings/refresh.go:73-74` — `hash := textHash(tc.Text)` then `vec := model.Embed(tc.Text)`, **unconditionally, for every chunk**.
8. `internal/embeddings/storage.go:80-110` `Upsert` — *then* compares `text_hash`, and returns `changed=false` without writing when it matches. The vector computed at step 7 is discarded.
9. `internal/embeddings/refresh.go:117` → `storage.PruneCorpus(ext.name, keepIDs)` — deletes every chunk of that corpus not in `keepIDs`. With an empty `keepIDs` (`storage.go:123-131`) it deletes the whole corpus.
10. Read side: `internal/retrieval/retrieval.go:683-684` — `retrieveHybrid` embeds the query and calls `QuerySimilar` against whatever is in `vec_chunks`, with no freshness check, and fuses it with a *current* lexical result set at `retrieval.go:694`.
11. `internal/cli/check.go:223` `addRow(...)` — the health-report surface. No embeddings row is ever added, so the skew at step 10 is never reported.

---

## Key Files

### Embedding engine
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/embeddings/refresh.go` | 32–126 | `Refresh` — the function with no automatic caller. Lines 73–74 embed unconditionally; line 117 prunes. Lines 90–114 carry a 20-line comment explaining why `Added`/`Updated` are not distinguished. |
| `internal/embeddings/storage.go` | 80–110 | `Upsert` — the too-late hash check; one SELECT + one INSERT per chunk, each its own implicit transaction. |
| `internal/embeddings/storage.go` | 122–152 | `PruneCorpus` — deletes the entire corpus on empty `keepIDs`; builds an unbounded `NOT IN (?,…)`. |
| `internal/embeddings/storage.go` | 55–76 | `migrate()` — the `vec_chunks` schema. Columns: `chunk_id, corpus, source_id, section, text_hash, vector, embedded_at`. **No `text` column** — the stored `text_hash` is the only content fingerprint available, which is why the staleness guard keys off `embedded_at` / `source_id`. |
| `internal/embeddings/chunker.go` | 31–77, 137–166 | `ChunkSpecs` and `ChunkConventions` — both call `spec.Discover(heroDir)`, so the full corpus is walked and parsed twice per refresh. |
| `internal/embeddings/chunker.go` | 170–217 | `ChunkEvents` — extracts `$.body` / `$.subject`; live event nodes store their content under `$.text`. |
| `internal/embeddings/chunker.go` | 222–270 | `ChunkCodeSymbols` — reads `Symbol` nodes from `graph.db`, returns `(nil, nil)` when `graphDB` is nil. |
| `internal/embeddings/model.go` | 43–76 | `LoadModelFromConfig` — returns `(nil, nil)` when no model is available. Callers must nil-check; `Refresh` does not. |

### Trigger points
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/next_hooks.go` | 314–349 | `hookScript(kind)` — generates the managed hook block. The one place a harness-agnostic trigger can be added. |
| `internal/cli/next_hooks.go` | 260–267 | `installHooks` — writes `pre-commit` and `post-merge` only. |
| `internal/cli/next_hooks.go` | 206–212 | `preCommitHookStale` — byte-compares the installed block against `hookScript("pre-commit")`. Changing the template makes every installed hook report stale until `hero next install-hooks` / `hero upgrade` reruns. |
| `internal/cli/embeddings.go` | 39–42 | `init()` — registers `status` and `rebuild`; `refresh` is missing. |
| `internal/cli/embeddings.go` | 97–148 | `runEmbeddingsRebuild` — the closest existing template for a new `refresh` command, including the best-effort `graph.Open` at 132–136. |
| `internal/cli/scan.go` | 401–424 | The other `Refresh` call site, with the model nil-check and per-step report row. |

### Reporting surfaces
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/check.go` | 223–225 | `addRow(name, status, message)` — the extension point for the staleness row. |
| `internal/cli/check.go` | 261–273 | `stale-specs` row — the pattern the new row should follow (print block + `addRow`). |
| `internal/cli/check.go` | 62–72 | `healthJSONRow` / `healthJSONFile` — the `.hero/cache/health.json` contract the serve dashboard reads, so a new row propagates there for free. |
| `internal/index/index.go` | 79–93 | `busy_timeout(5000)` + WAL on `index.db` — the reason a contended write can block for 5s. |
| `internal/retrieval/retrieval.go` | 683–694 | `retrieveHybrid` — where a search-time freshness warning would go if we wanted one (we do not; see Boundaries). |

### Prior art
| File | Lines | Relevance |
|------|-------|-----------|
| `.hero/specs/embedded-inference/spec.md` | 224, 366, 427 | AC-9 text, the design note, and the Completion Ledger row marking AC-9 `PARTIAL` on a `completed` spec. |
| `.hero/planning/features/embeddings-index/spec.md` | 129, 212, 217 | Superseded predecessor. Line 129 specifies the pre-commit hook; 212 specifies the `hero check` staleness probe; 217 proposes inline refresh at search time (which this spec rejects — see Boundaries). |

---

## Design

### Decision: (a) incremental `Refresh` inline in the git pre-commit hook

Adopted, with two preconditions that make it safe: the no-op path must become
cheap (AC-1), and a delta pass must never prune (AC-4).

### 1. Trigger point

**Where:** the hero-managed block in `.git/hooks/pre-commit`, generated by
`hookScript("pre-commit")` at `internal/cli/next_hooks.go:314`. One new line,
immediately after the existing index sync:

```
  hero embeddings refresh --if-stale -q || true
```

**Why this satisfies the `harness-changes-cover-all-targets` tripwire:** the
git hook is part of the *repository*, not part of any harness. It fires
identically for `opencode`, `cursor`, `claude`, `copilot`, `codex`, and
`generic` — and for a plain human with no agent at all, and for CI.

**Why a session hook is not an acceptable alternative — explicitly
eliminated.** The obvious-looking answer is to hang the refresh off Claude
Code's `Stop` / `PreCompact` hook. Verified against the tree: hook wiring
exists for exactly two targets —

- `internal/install/claude_hooks.go:50` — `claudeHookEvents = []string{"Stop", "PreCompact", "SessionStart"}`
- `internal/install/codex_hooks.go:29` — `codexHookEvents = []string{"Stop", "SessionStart"}`

There is no `opencode_hooks.go`, `cursor_hooks.go`, `copilot_hooks.go`, or
`generic_hooks.go`. **Four of the six install targets have no session-hook
mechanism whatsoever.** A refresh hung off a session hook would keep the
index current for Claude and Codex users and silently leave everyone else
exactly as broken as today — the precise failure mode the tripwire exists to
prevent. (Note the tripwire's own wording calls Claude the sole outlier;
Codex has since grown a `Stop` hook. The conclusion is unchanged and in fact
stronger: the split is 2 / 4, not 1 / 5.)

**Why pre-commit and not post-commit (option b).** A detached post-commit
refresh has one real advantage — zero added git latency — and several
disqualifying costs:

- No post-commit hook exists today. `installHooks` writes `pre-commit` and
  `post-merge` only, so option (b) means a third managed hook file to
  install, keep in sync, staleness-check, and test.
- A detached process is *unsupervised*. Its failures are invisible. Invisible
  failure is precisely how the index rotted for nine days without anyone
  noticing; option (b) reproduces the original failure mode in a new place.
- Detached refreshes race each other. Rapid commits (an agent looping
  `deliver`) spawn overlapping writers against a `busy_timeout(5000)` WAL
  database.
- On many setups the detached child dies with its parent shell or agent
  session, so the refresh that was supposed to be free simply never runs.
- The existing precedent is pre-commit: `hero index --if-stale` already syncs
  the lexical half of the *same database* there. Putting the vector half in a
  different hook splits one invariant across two trigger points.

Option (b) is only compelling if the inline cost cannot be bounded. It can —
see the measurements and AC-1.

**Why not a watcher/daemon (option c).** It requires per-repo process
lifecycle management, dies silently, needs platform-specific filesystem
watching, and would have to be hosted somewhere (`hero serve` — which not
everyone runs). That is more machinery than the entire embeddings feature
contains, to solve a problem a single hook line solves. Rejected as
disproportionate.

**Cost budget.** Measured steady-state no-op refresh today is **575ms**. With
AC-1 (hash before embed) the no-op path drops to roughly the extraction cost
— ~49ms for the spec walk plus a single batched hash query per corpus — and a
real per-commit delta of tens of chunks adds single-digit milliseconds. The
`--deadline` guard (below) caps the worst case regardless.

### 2. Delta scoping

**What `scope` is today:** corpus-scoped. `refresh.go:41-44` builds a set from
`scope []string` and line 60 tests it against `ext.name` ∈ {`spec`,
`knowledge`, `convention`, `event`, `code`}. It is not file-scoped and no
extractor accepts a path filter.

**Recommendation: do not extend `scope` to file-level scoping.** Keep the
signature. Achieve the delta *inside* `Refresh` by moving the existing
`text_hash` comparison ahead of `model.Embed`. Reasons:

1. The hash cache already gives exact per-chunk delta semantics. It is
   applied one step too late — the vector is computed and thrown away
   (`refresh.go:73-74` then `storage.go:88`). Fixing the ordering is a
   ~10-line change with no API surface.
2. Whole-corpus *extraction* is cheap and does not need avoiding: 49ms
   measured for the full `spec.Discover` walk of 589 specs; the code corpus
   is a single SQL query.
3. File-scoped refresh is actively unsafe with `PruneCorpus`. A pass whose
   `keepIDs` contains only the changed files' chunks would delete every other
   chunk in the corpus (`storage.go:143`, `NOT IN`). Making it safe requires
   a prune-suppression mode — two code paths, and a new bug class (orphans
   that are never pruned). Not worth it for 49ms.
4. Determining changed files from git in the hook (`git diff --cached
   --name-only`) then mapping paths → chunk IDs is a mapping layer that does
   not exist and would need to handle spec sections, knowledge relpaths, and
   symbol keys differently per corpus.

**Corpus set for the hook: `spec`, `knowledge`, `convention` — the
file-backed corpora.** The `code` and `event` corpora are excluded from the
hook pass, deliberately:

- **`code` must not be refreshed by the hook, because codescan ingest must
  run first.** Confirmed (C6): `ChunkCodeSymbols` reads `Symbol` nodes from
  `graph.db`, which only `hero scan` writes. Embedding without re-ingesting
  would re-embed stale symbols — motion without progress. Wiring codescan
  into pre-commit is a materially larger change (the graph-ingest portion of
  `hero scan` is most of its 2.8s steady-state cost) and belongs in its own
  spec. **The answer to "must codescan ingest also be triggered?" is yes for
  code coverage, and no for this fix's scope** — instead the staleness guard
  reports the code corpus separately and tells the user to run `hero scan`
  (AC-8).
- **`event`** is excluded because it needs `graph.db`, and opening a second
  database inside the hook adds lock-contention risk for a corpus that is
  currently empty and broken anyway (Secondary Defect 3).

The hook scope is a hard-coded constant in the new command, not a config
knob. No new configuration.

### 3. Failure and timeout behavior

Governing contract: **hooks must never block or hang git.**

| Failure | Behavior |
|---|---|
| **Slow model load** | Not a real risk — measured ~23ms warm for the whole process, ~0.6s once per boot for cold binary page-in. Covered by the deadline regardless. |
| **Missing model** | `LoadModelFromConfig` returns `(nil, nil)`. The new command nil-checks, prints nothing under `-q`, exits 0. `Refresh` itself also gains a nil guard (AC-6) so no future caller can reach the `model.Embed` nil deref at `refresh.go:74`. |
| **Locked / contended DB** | `index.db` uses `busy_timeout(5000)`, so a single contended statement can block 5s and today there are ~10,700 statements. Two mitigations: (i) wrap the whole write batch in one transaction so the lock is acquired once instead of per chunk (AC-7); (ii) the deadline is checked between chunks and covers wall time including lock waits, so a jammed DB aborts rather than accumulating 5s stalls. |
| **Hanging embedding backend** | The current model is pure in-process Go with no I/O, so it cannot hang. The deadline covers a future pluggable backend. |
| **Any error at all** | Under `-q` the command exits **0** unconditionally. Errors go to stderr only when not quiet. Belt-and-braces: the hook line still carries `|| true`, matching every other call in the managed block. |
| **Deadline expiry** | Stop cleanly after the in-flight chunk, commit the partial transaction (partial progress is safe — every chunk is independently valid, and the next run picks up the rest), **skip the prune**, exit 0. |

The deadline is enforced **inside the Go command** (`--deadline`, default
`2s`), not by a shell `timeout`. `timeout(1)` is not present by default on
macOS (it ships as `gtimeout` with coreutils), so a shell-level guard would
be a silent no-op on the primary development platform.

### 4. The staleness guard

**Where:** a new `hero check` row via `addRow` at `internal/cli/check.go:223`,
following the `stale-specs` pattern at 261–273. Because `addRow` feeds
`healthJSONFile` → `.hero/cache/health.json` (`check.go:62-84`), the serve
dashboard picks it up with no extra work. Additionally, `hero embeddings
status` gains a `Last embedded` line with the lag (AC-9) — the natural place
to look once `hero check` has pointed you there.

**Signal:** `MAX(embedded_at)` over `vec_chunks` versus the committer
timestamp of HEAD (`git log -1 --format=%cI`).

**The false-positive trap, and how the guard avoids it.** `Upsert` only
rewrites `embedded_at` when the text hash changes (`storage.go:88`). In a
repository where commits touch only code and no specs, `MAX(embedded_at)`
legitimately stays old while the index is perfectly current. A bare
`max(embedded_at) < HEAD time` test would cry wolf. So the row uses a
two-part signal:

1. **Lag:** `HEAD committer time − MAX(embedded_at)`.
2. **Coverage:** the count of embeddable source files whose mtime is newer
   than `MAX(embedded_at)` — i.e. sources that have actually changed since
   the last embed — plus the count of discovered sources with no
   corresponding `source_id` in `vec_chunks` at all (never embedded).

**Thresholds:**

- `pass` — coverage count is 0 (nothing changed since the last embed), *or*
  lag ≤ 24h and coverage count is 0.
- `warn` — coverage count > 0. Message names the number of unembedded/changed
  sources and the lag in days, e.g.
  `47 source(s) changed since last embed (index 9d behind HEAD); run 'hero scan'`.
- Never `fail`. A stale vector index degrades ranking; it does not break the
  workspace, and `hero check` failing on it would be noise.

**Second row for the code corpus** (AC-8), because the hook does not cover
it: `MAX(ingested_at)` over `Symbol` nodes in `graph.db` versus HEAD time,
with the same coverage disambiguation over `.go`/source files. This is the
row that would have caught the reported symptom directly — "0 symbols for
files created today".

**Not doing:** inline refresh at search time. The superseded
`embeddings-index` spec proposed it (line 217: *"`Retriever.Search` checks
last-refresh mtime … and triggers a quick `--if-stale` refresh inline"*).
Rejected — it makes a read path mutate shared state and take an unbounded
write lock, on the hottest path in the product. The check row plus the
pre-commit trigger cover the same ground without that. See Boundaries.

---

## Acceptance Criteria

- AC-1: WHEN `Refresh` processes a chunk whose stored `text_hash` already matches the chunk's current text THE SYSTEM SHALL skip the `Embed` call entirely and count the chunk as skipped.
- AC-2: WHEN `hero embeddings refresh --if-stale` runs on a workspace with no source changes THE SYSTEM SHALL complete in under 200ms on the reference corpus and report `added=0 updated=0 pruned=0`.
- AC-3: WHEN the hero-managed pre-commit block is generated THE SYSTEM SHALL include a best-effort `hero embeddings refresh --if-stale -q` invocation, and SHALL generate the identical block for every install target because the trigger lives in the git hook rather than in any harness surface.
- AC-4: WHEN `hero embeddings refresh` runs in delta mode THE SYSTEM SHALL NOT prune any corpus, so that a partial or deadline-truncated pass can never delete chunks it merely did not visit.
- AC-5: IF `hero embeddings refresh` exceeds its `--deadline` (default 2s), OR the embedding model is unavailable, OR the index database is locked, OR extraction returns an error, THEN THE SYSTEM SHALL exit 0 without writing to stderr under `-q`, so that git neither stalls nor fails the commit.
- AC-6: IF `Refresh` is called with a nil model THEN THE SYSTEM SHALL return a descriptive error rather than dereferencing it.
- AC-7: WHEN `Refresh` writes changed chunks THE SYSTEM SHALL perform the writes for a corpus inside a single transaction, so that a contended database costs one lock acquisition rather than one per chunk.
- AC-8: WHEN `hero check` runs THE SYSTEM SHALL emit an `embeddings-stale` row and a `code-symbols-stale` row reporting the lag between the newest embedded/ingested timestamp and HEAD, warning only when at least one source has changed since that timestamp.
- AC-9: WHEN `hero embeddings status` runs THE SYSTEM SHALL print the newest `embedded_at` per corpus and its lag behind HEAD.
- AC-10: WHEN an extractor cannot reach its source (for example `graph.db` is unavailable and `ChunkCodeSymbols` yields nothing) THE SYSTEM SHALL skip that corpus entirely rather than treating the empty result as authoritative and pruning every chunk in it.
- AC-11: WHEN a commit is made in a repository with the refreshed hook installed THE SYSTEM SHALL leave `MAX(embedded_at)` for the `spec` corpus no older than the commit, verified by an integration test that edits a spec, commits, and asserts the chunk was re-embedded.

---

## Suggested Fix Approach

### 1. `internal/embeddings/refresh.go` — hash before embed, transactional writes, no prune in delta mode, nil guard (AC-1, AC-4, AC-6, AC-7, AC-10)

**Why:** this is the change everything else rests on. Today the vector is
computed for every chunk and then discarded if the hash matched — ~375ms of
pure waste per no-op pass on the spec corpus alone. Moving the comparison
ahead of `Embed` turns the no-op path into an extraction walk plus one
batched query per corpus. It also makes the existing "incremental" claim in
`embedded-inference` AC-4 actually true.

**Before** (`refresh.go:69-115`, abridged to the load-bearing lines):

```go
		keepIDs := make([]string, 0, len(chunks))
		for _, tc := range chunks {
			keepIDs = append(keepIDs, tc.ID)

			hash := textHash(tc.Text)
			vec := model.Embed(tc.Text)

			chunk := Chunk{
				ID:       tc.ID,
				Corpus:   tc.Corpus,
				SourceID: tc.SourceID,
				Section:  tc.Section,
				TextHash: hash,
				Vector:   vec,
			}

			changed, err := storage.Upsert(chunk)
			if err != nil {
				return nil, fmt.Errorf("upserting chunk %q: %w", tc.ID, err)
			}

			if changed {
				stats.Added++
			} else {
				stats.Skipped++
			}
		}
```

**After** (shape; the `Options` struct carries `Prune bool` and
`Deadline time.Time`):

```go
		// One query instead of one SELECT per chunk. Lets us decide
		// "needs re-embedding" without ever calling model.Embed.
		known, err := storage.HashesForCorpus(ext.name)
		if err != nil {
			return nil, fmt.Errorf("loading %s hashes: %w", ext.name, err)
		}

		keepIDs := make([]string, 0, len(chunks))
		pending := make([]Chunk, 0, 64)
		for _, tc := range chunks {
			keepIDs = append(keepIDs, tc.ID)

			hash := textHash(tc.Text)
			if prior, ok := known[tc.ID]; ok && prior == hash {
				stats.Skipped++
				continue // unchanged: no Embed, no write
			}

			pending = append(pending, Chunk{
				ID:       tc.ID,
				Corpus:   tc.Corpus,
				SourceID: tc.SourceID,
				Section:  tc.Section,
				TextHash: hash,
				Vector:   model.Embed(tc.Text),
			})

			if !opts.Deadline.IsZero() && time.Now().After(opts.Deadline) {
				stats.TimedOut = true
				break
			}
		}

		added, updated, err := storage.UpsertBatch(pending, known)
		if err != nil {
			return nil, fmt.Errorf("upserting %s chunks: %w", ext.name, err)
		}
		stats.Added += added
		stats.Updated += updated
```

and the prune becomes conditional:

**Before** (`refresh.go:117-121`):

```go
		pruned, err := storage.PruneCorpus(ext.name, keepIDs)
		if err != nil {
			return nil, fmt.Errorf("pruning %s corpus: %w", ext.name, err)
		}
		stats.Pruned += pruned
```

**After:**

```go
		// Never prune from a pass that may not have seen the whole
		// corpus: a delta or deadline-truncated run's keepIDs is not
		// authoritative, and PruneCorpus deletes everything not in it.
		if opts.Prune && !stats.TimedOut {
			pruned, err := storage.PruneCorpus(ext.name, keepIDs)
			if err != nil {
				return nil, fmt.Errorf("pruning %s corpus: %w", ext.name, err)
			}
			stats.Pruned += pruned
		}
```

Add at the top of `Refresh` (AC-6) — currently there is no guard and
`model.Embed` at line 74 is a nil deref waiting for a caller that forgets:

```go
	if model == nil {
		return nil, fmt.Errorf("embeddings: refresh requires a loaded model")
	}
```

And in the extractor loop, distinguish "source unavailable" from "genuinely
empty" (AC-10) so a nil `graphDB` can never wipe 2,927 code chunks. The
cleanest form is for `corpusExtractor.extract` to return an
`available bool`, with `ChunkEvents` / `ChunkCodeSymbols` reporting
`available=false` when `graphDB == nil`; an unavailable corpus is `continue`d
before both the upsert loop and the prune.

`Updated` becomes real for the first time: `UpsertBatch` knows whether each
ID was present in `known`, so the 20-line comment at `refresh.go:90-110`
explaining why the distinction is not made can be deleted along with the
behaviour it describes (Secondary Defect 9).

### 2. `internal/embeddings/storage.go` — batched hash read, batched write (AC-1, AC-7)

**Why:** `Upsert` does one `SELECT` and one `INSERT` per chunk, each in its
own implicit transaction — ~10,700 round trips against a database with a
5-second busy timeout. One `SELECT` per corpus and one transaction per batch
removes both the round trips and the repeated lock acquisition.

New:

```go
// HashesForCorpus returns chunk_id -> text_hash for every chunk in the
// corpus, so a refresh can decide what changed without a query per chunk.
func (s *Storage) HashesForCorpus(corpus string) (map[string]string, error)

// UpsertBatch writes all chunks in a single transaction and reports how
// many were new (absent from known) versus updated.
func (s *Storage) UpsertBatch(chunks []Chunk, known map[string]string) (added, updated int, err error)
```

Keep `Upsert` — `storage_test.go` exercises it and it is the natural
single-chunk API — implemented as a one-element `UpsertBatch` so the hash
semantics stay in one place.

### 3. `internal/cli/embeddings.go` — the missing `refresh` subcommand (AC-2, AC-4, AC-5)

**Why:** `embedded-inference` AC-9 names a command that was never built. The
hook cannot call what does not exist.

**Before** (`embeddings.go:39-42`):

```go
func init() {
	embeddingsCmd.AddCommand(embeddingsStatusCmd)
	embeddingsCmd.AddCommand(embeddingsRebuildCmd)
}
```

**After:**

```go
func init() {
	embeddingsCmd.AddCommand(embeddingsStatusCmd)
	embeddingsCmd.AddCommand(embeddingsRebuildCmd)
	embeddingsCmd.AddCommand(embeddingsRefreshCmd)

	embeddingsRefreshCmd.Flags().BoolVarP(&embeddingsRefreshIfStale, "if-stale", "s", false,
		"only re-embed chunks whose text changed (the hook path)")
	embeddingsRefreshCmd.Flags().DurationVar(&embeddingsRefreshDeadline, "deadline", 2*time.Second,
		"abort cleanly after this much wall time so git is never blocked")
	embeddingsRefreshCmd.Flags().BoolVarP(&embeddingsRefreshQuiet, "quiet", "q", false,
		"suppress all output and always exit 0 (for hooks)")
}
```

`runEmbeddingsRefresh` follows `runEmbeddingsRebuild`
(`embeddings.go:97-148`) with four differences: it does **not** `DELETE FROM
vec_chunks`; it passes `Prune: false` and the computed deadline; it uses the
hard-coded hook corpus set `[]string{"spec", "knowledge", "convention"}`
rather than `cfg.EmbeddingsScope()`; and every error path under `-q` returns
nil after writing nothing.

The quiet contract is the load-bearing part — the whole point is that a
broken embeddings subsystem cannot break `git commit`:

```go
	if embeddingsRefreshQuiet {
		// Nothing about embeddings is worth failing a commit over.
		if err := doRefresh(); err != nil {
			return nil
		}
		return nil
	}
```

### 4. `internal/cli/next_hooks.go` — the trigger (AC-3)

**Why:** this is the actual reported bug. One line, in the one place that is
harness-agnostic.

**Before** (`next_hooks.go:318-322`):

```go
	if kind == "pre-commit" {
		// Index sync first so the queue write reads from a current
		// index. All three calls are best-effort.
		indexRefresh = `
  hero index --if-stale -q || true`
```

**After:**

```go
	if kind == "pre-commit" {
		// Index sync first so the queue write reads from a current
		// index, then the vector half of the same index.db. The
		// embeddings pass self-enforces a wall-clock deadline and
		// always exits 0 — hooks must never block or fail git. This
		// lives in the *git* hook, not a harness session hook,
		// because four of the six install targets (opencode, cursor,
		// copilot, generic) have no session-hook mechanism at all.
		// Spec: embeddings-never-refresh-on-commit.
		indexRefresh = `
  hero index --if-stale -q || true
  hero embeddings refresh --if-stale -q || true`
```

Note the knock-on: `preCommitHookStale` (`next_hooks.go:206-212`) byte-compares
the installed block against `hookScript("pre-commit")`, so every existing
checkout reports a stale hook until `hero upgrade` /
`hero next install-hooks` reruns. `refreshHooksIfPresent`
(`next_hooks.go:222`) already handles this for `hero upgrade`, and
`hero check` already prints the "Pre-commit hook is stale" warning at
`check.go:452-459`. Confirm both in delivery; no new code expected.

### 5. `internal/cli/check.go` — the staleness rows (AC-8)

**Why:** the index rotted for nine days and nothing said a word. The trigger
stops the rot; this makes any future rot loud.

Insert after the `stale-specs` block (`check.go:261-273`), following its
shape exactly:

```go
	// Embeddings staleness — the vector half of index.db has its own
	// trigger (pre-commit) and can drift independently of FTS5.
	if emb, err := embeddingsStaleness(heroDir, projectRoot); err == nil {
		if emb.ChangedSources > 0 {
			issues++
			fmt.Printf("Embedding index is behind:\n")
			fmt.Printf("  %d source(s) changed since last embed (%s behind HEAD)\n",
				emb.ChangedSources, emb.Lag.Round(time.Hour))
			fmt.Println("  Run 'hero scan' to catch up.")
			fmt.Println()
			addRow("embeddings-stale", "warn", fmt.Sprintf(
				"%d source(s) changed since last embed (%s behind HEAD)",
				emb.ChangedSources, emb.Lag.Round(time.Hour)))
		} else {
			addRow("embeddings-stale", "pass", "embedding index is current")
		}
	}
```

plus the sibling `code-symbols-stale` row driven by `MAX(ingested_at)` over
`Symbol` nodes, since the hook deliberately does not refresh that corpus.

`embeddingsStaleness` is the only genuinely new logic: read
`MAX(embedded_at)` from `vec_chunks` (the schema has no `text` column, so
`embedded_at` and `source_id` are the only handles available), read HEAD's
committer time via `git log -1 --format=%cI`, and count sources whose mtime
exceeds the embed stamp or whose `source_id` is absent from `vec_chunks`
entirely. The coverage count is what prevents a quiet repository from
warning forever.

### 6. `internal/cli/embeddings.go` — surface the stamp in `status` (AC-9)

`runEmbeddingsStatus` (`embeddings.go:88-92`) prints chunk counts with no
timestamp. Add per-corpus `MAX(embedded_at)` and the lag, so the command
`hero check` points at actually answers the question. Backed by a new
`Storage.Stats()` field rather than a second query.

---

## Test Plan

### 1. Existing test review

| File | What it already covers | Gap |
|---|---|---|
| `internal/embeddings/refresh_test.go` (376 lines) | Idempotency and "changed-only re-embed" — but the assertions are on `RefreshStats.Skipped`, which today only means *the write was skipped*. The suite passes against code that embeds every chunk. | No test asserts `Embed` was not *called*. This is exactly the falsification gap that let Secondary Defect 1 ship as `AC-4 | DONE`. |
| `internal/embeddings/storage_test.go` (515 lines) | `Upsert` round-trip, hash-match skip, `PruneCorpus`, `QuerySimilar`, vector encode/decode. | Nothing covers `PruneCorpus` being reached with an empty `keepIDs` because the *source* was unavailable — the corpus-wipe hazard. |
| `internal/cli/next_hooks_test.go:170` | Asserts the generated pre-commit block contains `hero index --if-stale -q`. | Exact template to extend for the new line. |
| `internal/cli/check_test.go`, `check_json_test.go` | Row emission and the `health.json` schema. | Extend with the two new rows. |
| `internal/embeddings/model_test.go`, `model_real_test.go` | Embedding correctness; real-model tests skip unless `hero-embed-v1` is installed under `~/.hero/models/`. | Not affected. |
| `internal/cli/embeddings.go` | **No test file exists** for the CLI surface. | The new `refresh` command needs one from scratch. |

### 2. Test changes needed

**Falsification first.** Per this project's standing rule, each new test must
be proven to fail against the current code before the fix lands:

- **AC-1 (`refresh_test.go`)** — inject a counting model (wrap `*Model` behind
  a small interface, or count via a test-only hook) and assert that a second
  `Refresh` over unchanged content records **zero** `Embed` calls. *Falsify:*
  this must fail today with a call count equal to the chunk count. This is
  the single most important new test — it is the assertion whose absence let
  the defect ship.
- **AC-2 (`refresh_test.go`)** — a no-op refresh over a seeded corpus returns
  `Added==0 && Updated==0 && Pruned==0`. Assert on counters, not wall time;
  keep the 200ms budget as a documented manual check in the Completion
  Ledger rather than a flaky timing assertion in CI.
- **AC-4 / AC-10 (`refresh_test.go`, `storage_test.go`)** — with `Prune:
  false`, chunks absent from `keepIDs` survive. With `graphDB == nil`, a
  pre-populated `code` corpus is left **fully intact**. *Falsify:* today the
  second case deletes every code chunk.
- **AC-5 (new `internal/cli/embeddings_refresh_test.go`)** — table-driven over
  the failure matrix: nil model, deadline of `1ns`, unwritable index. Each
  case asserts exit code 0 and empty stdout/stderr under `-q`.
- **AC-6 (`refresh_test.go`)** — `Refresh(dir, nil, …)` returns an error and
  does not panic. *Falsify:* today it panics at `refresh.go:74`.
- **AC-7 (`storage_test.go`)** — `UpsertBatch` of N chunks where the (N-1)th
  fails leaves the table unchanged (transactional), and `added`/`updated`
  split correctly against the `known` map.
- **AC-3 (`next_hooks_test.go`)** — extend the existing assertion at line 170:
  the generated `pre-commit` block contains
  `hero embeddings refresh --if-stale -q`, it appears *after* the index line,
  it carries `|| true`, and the `post-merge` block does **not** contain it.
- **AC-8 (`check_test.go`)** — three fixtures: (i) sources newer than
  `MAX(embedded_at)` → `warn`; (ii) sources all older → `pass` even when HEAD
  is days newer (the quiet-repo false-positive case — this is the assertion
  that keeps the row trustworthy); (iii) a source with no row in `vec_chunks`
  at all → `warn`.
- **AC-9 (new CLI test)** — `hero embeddings status` output contains a
  per-corpus timestamp line.
- **AC-11 (integration)** — in a temp git repo with hooks installed: edit a
  spec, `git commit`, assert the affected chunk's `embedded_at` advanced and
  that unrelated chunks' stamps did **not**. Guard with the same skip the
  real-model tests use if `hero` is not resolvable on PATH.

### 3. Regression scope

- **Retrieval ranking** (`internal/retrieval/`) — a corpus that has been
  frozen for nine days will move substantially on first refresh. Hybrid RRF
  fusion tests using synthetic fixtures are unaffected, but
  `TestRetrieveHybrid_WithEmbeddedModel` and any expectation calibrated
  against the current (stale) live index should be re-confirmed.
- **`hero scan` timing** — `Refresh` gets faster, not slower; the scan report
  line `embeddings added=… skipped=…` keeps its shape but `updated` becomes
  non-zero for the first time. Any test or doc asserting `updated=0` will
  break, correctly.
- **Commit latency** — the user-visible risk. Measure the delta on a real
  commit before and after; the Completion Ledger must carry the number.
- **Hook staleness cascade** — changing `hookScript` makes every installed
  pre-commit hook compare unequal. Confirm `hero check` reports it,
  `hero upgrade` repairs it via `refreshHooksIfPresent`, and that a user who
  deliberately removed the managed block is still left alone
  (`next_hooks.go:227`).
- **`.hero/cache/health.json` consumers** — two new rows flow to
  `internal/serve/projectpage/data/health.go`. Confirm the dashboard tolerates
  unknown row names (it should; the contract is a flat list).
- **Concurrency** — the hook now opens `index.db` for writing while
  `hero index --if-stale` has just closed it and `hero queue write` is about
  to open it. Sequential within the hook, but a `hero serve` daemon holding
  the DB is the realistic contention case. The single-transaction batch plus
  the deadline are the mitigations; exercise them with a deliberately held
  lock.

---

## Secondary Defects

Found while tracing the flow. Numbers 1, 2 and 4 are entangled with the fix
and are addressed by the ACs above; the rest are separable.

1. **`Refresh` embeds every chunk on every pass** — `refresh.go:73-74` calls
   `model.Embed` before `Upsert` (`storage.go:88`) does the hash comparison,
   so the vector for an unchanged chunk is computed and discarded. Measured
   cost: ~375ms of the 424ms spec-only pass. `embedded-inference` records
   `AC-4 | re-embed changed chunks only | DONE` — that claim is false as
   written. **Addressed by AC-1.**

2. **An empty extraction result silently wipes a whole corpus** —
   `refresh.go:117` always calls `PruneCorpus`, and `storage.go:122-131`
   treats empty `keepIDs` as "delete everything in this corpus". Both
   `ChunkEvents` and `ChunkCodeSymbols` return `(nil, nil)` when `graphDB` is
   nil, which is indistinguishable from a genuinely empty corpus.
   `hero embeddings rebuild` opens `graph.db` best-effort
   (`embeddings.go:132-136`), so a locked or missing graph during a rebuild
   deletes all 2,927 code chunks with no error and no warning.
   **Addressed by AC-10.**

3. **The `event` corpus is permanently empty because the chunker reads the
   wrong props keys.** `ChunkEvents` (`chunker.go:176-182`) extracts
   `json_extract(props,'$.body')` and `'$.subject'`. Every live
   `UserAsk` / `SessionReflection` / `NextSuggestion` node stores its content
   under `$.text`:

   ```
   NextSuggestion | {"rationale":"","session_id":"","text":"Deliver graph-why-resolution…"}
   SessionReflection | {"session_id":"","tags":null,"text":"mkdocs-material's theme feature…"}
   ```

   `body` and `subject` are both absent, so `text` is empty, every chunk is
   skipped at `chunker.go:200`, and `PruneCorpus` then removes anything left
   over. `event` is in the default scope (`config.go:236`) and has **0 rows**
   in `vec_chunks`. An entire configured corpus contributes nothing to
   semantic search, silently. Directly against the mission — session
   reflections are exactly the "stuff nobody told the model" Hero exists to
   surface. Not fixed here (the hook scope excludes `event`); **worth its own
   spec.**

4. **`Refresh` has no nil-model guard** — `refresh.go:74` dereferences
   `model`, and `LoadModelFromConfig` returns `(nil, nil)` when no model is
   available (`model.go:75`). Both current callers nil-check externally
   (`embeddings.go:114`, `scan.go:406`), so the panic is latent — until a new
   caller inside a git hook forgets. **Addressed by AC-6.**

5. **`spec.Discover` runs twice per refresh** — `ChunkSpecs`
   (`chunker.go:32`) and `ChunkConventions` (`chunker.go:138`) each call it
   and each filter the result. Measured 49ms per walk of 589 specs, so ~49ms
   is thrown away on every refresh. Easy win: discover once and pass the
   slice to both. Not required by any AC; fold in if convenient.

6. **`Upsert` is one SELECT + one INSERT per chunk, each its own implicit
   transaction** (`storage.go:80-110`) — ~10,700 statement round trips and
   ~10,700 lock acquisitions against a WAL database with
   `busy_timeout(5000)`. **Addressed by AC-7.**

7. **`PruneCorpus` builds an unbounded parameter list** —
   `storage.go:133-146` emits one `?` per kept ID inside `NOT IN (…)`: 4,527
   parameters for this repo's spec corpus today. Under the modern SQLite
   default `SQLITE_MAX_VARIABLE_NUMBER` of 32,766 this works, but it is
   within one order of magnitude and scales with the corpus. A temp table or
   a `DELETE … WHERE corpus=? AND chunk_id NOT IN (SELECT …)` avoids the
   cliff. Not urgent; note it.

8. **Chunk-ID collision, unexplained.** A full rebuild reported
   `added=7512` while the resulting table held 7,502 rows — ~10 IDs upserted
   twice. Duplicate spec slugs ruled out (0 duplicates across
   `.hero/planning` and `.hero/specs`); `Symbol` keys ruled out (2,927 of
   2,927 distinct). Remaining candidates are sibling/satellite spec ingest
   and knowledge relpath collisions. **Hypothesis, not confirmed** —
   deliberately not folded into this fix.

9. **`RefreshStats.Updated` is dead code.** `refresh.go:90-114` counts every
   changed chunk as `Added` and carries a 20-line comment explaining why the
   insert/update distinction is not worth an extra round-trip. With the
   batched hash map from AC-1 the distinction is free, so both the comment
   and the always-zero field can go. Until then `hero scan` reports
   `updated=0` unconditionally, which is misleading.

---

## Boundaries

**In scope**

- Making the incremental `Refresh` path cheap enough to run per commit.
- A `hero embeddings refresh --if-stale` command with a self-enforced deadline
  and an exit-0-always quiet mode.
- One line in the managed **git** pre-commit block.
- Two `hero check` staleness rows and a timestamp line in
  `hero embeddings status`.
- The prune-on-empty and nil-model hazards that the new trigger would
  otherwise expose (Secondary Defects 2 and 4).

**Explicitly out of scope**

- **Any harness session hook.** Not a fallback, not an enhancement, not
  "additionally". Four of six targets cannot host one. See Design §1.
- **A watcher or daemon (option c).** Rejected as disproportionate; recorded
  in Design §1 so it is not re-proposed.
- **A detached post-commit refresh (option b).** Rejected; recorded with
  reasoning in Design §1.
- **Running codescan ingest from the hook.** The `code` corpus stays stale
  until `hero scan`. This is a real, acknowledged limitation of this fix —
  the bug report's "new .go files have 0 chunks" symptom is only *reported*
  by this spec (AC-8), not *fixed*. Wiring codescan into pre-commit needs its
  own cost analysis and its own spec.
- **The `event` corpus wrong-key bug (Secondary Defect 3).** Separable, and
  the hook scope excludes `event`. Own spec.
- **Inline refresh at search time.** The superseded `embeddings-index` spec
  proposed it; rejected here because it makes a read path take a write lock.
- **File-level scoping of `Refresh`'s `scope []string`.** The signature stays
  corpus-scoped. Rationale in Design §2.
- **New configuration.** The hook corpus set and the default deadline are
  constants. No `embeddings.hook_scope` knob.
- **Backfilling this repository's nine-day gap.** A one-off `hero scan` does
  that; it is not a code change.
- **Fixing `hero spec verify` so a spec cannot close with a PARTIAL AC.**
  This is the process cause that produced the bug, and it is the more
  valuable fix — but it is a governance change to the verify gate, not an
  embeddings change. See Notes.

---

## Risks

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| **Commit latency becomes noticeable.** Every `git commit` gains the refresh. | medium | medium | AC-1 removes the dominant cost (measured 575ms → extraction-bound); AC-5's 2s deadline caps the tail. Measure a real commit before/after and record the number in the Completion Ledger. If it still bites, the fallback is option (b), not a session hook. |
| **A partial (deadline-truncated) pass leaves the index in a mixed state.** | high — by design | low | Every chunk is independently valid; a truncated pass is simply "less caught up", never inconsistent. AC-4 guarantees it can never *delete*. The next commit continues. |
| **The prune change lets orphaned chunks accumulate**, since the hook path never prunes. | medium | low | `hero scan` still prunes (`Prune: true`). Orphans cost storage and can surface deleted specs in search. Bound it: the `hero check` row should also report orphan count, or `hero scan` frequency covers it. Call this out during delivery — it is the real cost of AC-4. |
| **Lock contention with a running `hero serve`.** `index.db` has `busy_timeout(5000)`; a held write lock could burn the whole deadline. | low | medium | AC-7 collapses ~10,700 lock acquisitions into one per corpus; the deadline covers wall time including lock waits; AC-5 guarantees exit 0. Test with a deliberately held lock. |
| **Hook-template change makes every installed hook report stale**, generating a wave of `hero check` warnings. | high — certain | low | Expected and already handled: `refreshHooksIfPresent` repairs on `hero upgrade`, and `check.go:452-459` already tells the user to run `hero next install-hooks`. Mention it in the release note. |
| **Users who removed the managed block get no refresh** and stay silently broken. | medium | medium | Exactly what the AC-8 staleness row is for — it works regardless of whether the hook is installed. |
| **First refresh after the fix is a large one.** Nine days of drift means the first post-fix commit re-embeds thousands of chunks and will hit the deadline. | high | low | Deadline-truncated passes are safe (AC-4) and each subsequent commit catches up further. Recommend a one-off `hero scan` at delivery to zero the backlog. |
| **Non-git checkouts get nothing.** The trigger is a git hook; a workspace used outside git has no automatic refresh. | low | low | Accepted. `hero scan` remains the manual path and the AC-8 row still reports staleness. |
| **The counting-model test for AC-1 requires touching the `Model` type** (interface or test hook) to be observable. | medium | low | Prefer a narrow package-internal seam over exporting an interface. If it turns invasive, assert on a new `RefreshStats.Embedded` counter instead — weaker, but still falsifiable. |

---

## Notes

**The process cause deserves its own attention.** The mechanical fix here is
small. The interesting finding is that `embedded-inference` closed as
`completed` while its own Completion Ledger said AC-9 was `PARTIAL` and
described the exact gap in accurate detail. The system knew. It wrote it
down. It shipped anyway, and the predecessor spec that carried the same
requirement was marked `superseded`, so the requirement had nowhere left to
live. Nine days of silent index rot followed — and the only reason it
surfaced at all is that someone went looking.

Two ledger rows in that spec were `PARTIAL` (AC-9 and AC-10). If
`hero spec verify` Gate 1 does not block completion on a self-reported
PARTIAL row without sign-off, this class of bug will recur — and it is worth
noting that a *sibling* open bug,
`ledger-signoff-substring-match-fails-open`, describes the sign-off gate
failing open by substring match. These two are plausibly the same governance
hole seen from two angles. Recommend raising a spec against the verify gate;
deliberately not folded in here.

**On the premise that the `text_hash` cache makes unchanged chunks free.**
It does not. It makes unchanged *writes* free. The `Embed` call at
`refresh.go:74` happens first, unconditionally, and its output is discarded
at `storage.go:88`. This was worth checking rather than trusting: it is the
difference between "the hook is already cheap, just call it" and "the hook is
only cheap after a ten-line reordering", and it accounts for roughly 80% of
the measured no-op refresh time.

**On the model-load worry.** It was the right thing to be suspicious of and
it turned out to be a non-issue: ~23ms warm for the entire process. The 24MB
weight blob is `//go:embed`-ed into the binary, so there is no file read at
all — just a 6.07M-element `Float32frombits` decode. The only cold cost is
the OS paging in a 58MB binary once per boot (~0.6s), which the 2s deadline
absorbs.

---

## Recap

Hero's semantic index only refreshes when a human types `hero scan` — the
pre-commit trigger was designed in `embedded-inference` AC-9, marked PARTIAL
in that spec's own ledger, and shipped as `completed` anyway, so neither the
`hero embeddings refresh` command nor the hook line was ever built. In this
repository the result is a vector index frozen at 2026-07-16 while HEAD is
2026-07-25 — 76 commits and 615 files invisible to semantic search, with no
warning anywhere. The fix is to make the no-op refresh genuinely cheap (the
hash check currently runs *after* the embedding, wasting ~80% of the pass),
add a deadline-bounded `hero embeddings refresh --if-stale`, call it from the
harness-agnostic **git** pre-commit hook, and add a `hero check` row so this
can never rot invisibly again.
