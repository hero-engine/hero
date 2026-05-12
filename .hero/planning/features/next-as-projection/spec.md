---
title: NEXT.md as a Graph Projection (with project/user/local split)
type: feature
status: planning
priority: P0
tags: [next-md, projection, graph, handoff, merge-conflicts, v2-recovery]
created: 2026-04-28
relations:
  - target: get-back-on-track
    kind: parent
  - target: graph-memory
    kind: completes
  - target: master-ingest-restore
    kind: depends-on
  - target: acceptance-criteria-graph
    kind: depends-on
  - target: traversal-queries
    kind: depends-on
mission_alignment: |
  NEXT.md is the cross-session handoff artifact — the load-bearing
  surface for principle #3 ("sessions start omniscient"). Today its
  agent-written half drifts (last session shipped 7 commits past its
  last NEXT.md update, including the very work it had proposed) and
  its mechanical half churns every turn (rewriting a tracked file on
  every Stop hook firing makes branch divergence a guaranteed merge
  conflict). The fix is to finish what `2158bd2` started: NEXT.md is
  a projection of the graph, regenerated on demand, never hand-merged.
  With Cloud federation absent (the canonical hero-on-itself case),
  cross-machine continuity rides on a small tracked per-user artifact
  that doubles as the federation medium for non-CLI viewers.
principles_check: |
  Serves #2 (natural language is the interface) — the user shouldn't
  remember what file to open or section to find; `hero next suggest`
  prints the suggested next prompt for them. Serves #3 (sessions
  start omniscient) — the agent half stops drifting because it isn't
  written; it's projected. Serves #5 (zero-buy-in benefit) — non-CLI
  teammates open NEXT.md and get current project context for free.
  Risks #4 (don't make people read docs) — mitigated by keeping
  NEXT.md the visible, GitHub-rendered top-level entrypoint plus
  field-grab CLI for those who'd rather not scroll.
horizon: now
smoke: deferred
---

## Goal

Finish making NEXT.md a graph projection (declared in `2158bd2`,
unfinished) and split the on-disk surface into three files that
serve three different audiences:

- `.hero/NEXT.md` — **project state**, tracked, projected, the
  entrypoint non-CLI viewers see when they open the repo.
- `.hero/next/<user>.md` — **your durable user state** (last ask,
  suggested next prompt, session reflections), tracked, projected.
  Doubles as the cross-machine federation medium when no Cloud is
  configured.
- `.hero/next/<user>.local.md` — **your scratch + machine state**
  (dirty tree, hot files, hand-written reminders), gitignored.
  Marker-bounded auto sections coexist with hand content.

All three files are total-rewrite from graph on every Stop hook.
Merge conflicts on the tracked files auto-resolve via a `hero-next`
git merge driver that ignores both sides and regenerates from the
local graph.

## Why this is mission-critical

Three observed failures the current design cannot prevent:

**F1 — Agent narrative drift.** The "Just finished / Next" sections
rely on agent discipline to refresh. Demonstrated 2026-04-28: a
session shipped 7 commits past its last NEXT.md update, including
the very work it had just proposed. The next session opened to a
NEXT.md proposing already-completed work. *Audit grade: invisible
without checking against `git log`.*

**F2 — Mechanical churn → merge conflicts.** The Stop hook rewrites
the bottom of NEXT.md on every assistant turn. Every branch
divergence guarantees a merge conflict on auto-content; `git status`
is always dirty; `git log -- NEXT.md` is unreadable noise.

**F3 — Cross-machine and cross-branch aliasing.** Same branch on two
machines: NEXT.md drifts because each writes its own machine state.
Switch branches: the file reflects the wrong branch's snapshot.

The graph substrate built in graph-memory phases 1–4 was supposed to
prevent all three. Phase 4 (`2158bd2`) declared NEXT.md a projection
("hero writes NEXT.md, humans don't") but the agent-half projection
never shipped. This spec finishes that work and adds the file-shape
needed to handle solo-no-Cloud cross-machine continuity through git.

## Design

### Three files, three audiences

| File | Tracked? | Audience | Updates |
|---|---|---|---|
| `.hero/NEXT.md` | yes | Anyone who opens the repo (incl. non-CLI viewers) | Total-rewrite each turn from project graph |
| `.hero/next/<user>.md` | yes | You, across machines / your teammates if any | Total-rewrite from your user-graph nodes |
| `.hero/next/<user>.local.md` | **no** | You, on this one machine | Auto sections rewritten between markers; hand content preserved |

### What goes in `.hero/NEXT.md` (project state)

Author-agnostic, branch-current, derived entirely from project graph:

- Mission summary (one-liner from `Mission` node)
- Recent commits (project-wide, last N, deduplicated by message)
- Recent AC status flips (any author)
- Open Features / Initiatives by priority
- Project blockers (`hero blocked` output: dep-blocked + failing-AC)
- Recent decisions (`Decision` nodes, last N)
- In-flight specs (status: planning, delivering)

No "Last user ask" or "Suggested next" — those are user state, not
project state.

### What goes in `.hero/next/<user>.md` (user durable state)

Per-user, projected from user-graph nodes attributed to this Person:

- Last user ask (paraphrased; `UserAsk` node, latest)
- Suggested next prompt (`NextSuggestion` node, latest)
- In-flight session reflections (`SessionReflection` nodes since
  last commit boundary)
- Your recent activity (commits authored by you, AC flips you
  caused)
- Your tried-and-failed in this session (`Attempt(status=failed)`
  attributed to you in current Session)

Tracked because: in solo-no-Cloud, this file IS the cross-machine
federation medium. Drive home → office: pull → the file is there →
session-start hook re-ingests its entries into the local graph
(round-trip), then projects them again. Idempotent.

### What goes in `.hero/next/<user>.local.md` (scratch + machine)

Gitignored. Two zones:

- **Auto zone** (between `<!-- BEGIN HERO MACHINE STATE -->` /
  `<!-- END -->` markers): branch, dirty working tree, hot files,
  activity since *your* last checkpoint *on this machine*.
  Rewritten in place every Stop hook.
- **Hand zone** (outside markers): your scratch reminders, ad-hoc
  thoughts, anything you typed by hand. Survives every regen.

Why both in one file: discoverability. One place to glance at
"what's my situation right now," not two.

### New graph node types

Three new node types attributed to a `Person` and a `Session`:

- **`UserAsk`** — paraphrased or verbatim text of the user's most
  recent prompt. Props: `text`, `paraphrased` (bool), `session_id`.
  Latest per Person is the projected "Last user ask."
- **`NextSuggestion`** — the agent's recommended next prompt for
  this user. Props: `text`, `rationale` (optional), `session_id`.
  Supersedes prior. Latest per Person is the projected "Suggested
  next."
- **`SessionReflection`** — mid-session lesson or observation worth
  surfacing. Props: `text`, `tags`, `session_id`. Latest N per
  Person within current Session are projected.

All three federate via Cloud sync (when present) and round-trip
through the tracked `<user>.md` file (when not). Bitemporal storage
gives "what was suggested-next on Tuesday at home?" for free.

### Total-rewrite projection

`hero next --write` regenerates all three files from the graph (and
from machine state for the local file). No marker preservation in
NEXT.md or `<user>.md` — projections always win, hand-edits in those
files are wiped next turn. The `<user>.local.md` file is the
exception: marker-bounded auto sections rewritten, hand content
preserved.

`hero next --write` replaces the existing `hero next checkpoint`.
The Stop hook (and new git hooks) call the new command.

### Field-grab CLI

For when scrolling NEXT.md is too much friction:

```
hero next suggest                    # prints just the suggested next prompt
hero next suggest --copy             # also copies to clipboard
hero next suggest --json             # structured output for scripting
hero next suggest --user alice       # someone else's (team mode)

hero next ask                        # last user ask
hero next reflection                 # most recent session reflection
hero next blockers                   # current project blockers (one-liner per)
```

All read directly from the graph — always current even if the file
projection hasn't fired yet on this machine.

### Round-trip ingest (solo-no-Cloud cross-machine)

Without Cloud, the `<user>.md` file is the federation medium. To
make it bidirectional:

- `hero next --write` projects user-graph nodes → `<user>.md` file.
- `hero scan` (and a session-start hook) reads `<user>.md` back into
  the graph if entries aren't already present locally. Idempotent —
  re-ingest of the same content produces no new graph edits.

Result: home laptop writes, commits, pushes; office desktop pulls,
session-start hook ingests `<user>.md` into its local graph, then
projects it back out. Same content visible on both machines.

### Layered hooks (reliable end-of-turn projection)

| Hook | Trigger | Runs |
|---|---|---|
| Stop hook (Claude Code, opencode, etc.) | Every assistant turn ends | `hero next --write` |
| Pre-commit (git) | Right before `git commit` | `hero next --write`; stage changes |
| Post-merge (git) | After `git merge` / `git pull` | `hero next --write` |
| Session-start (Claude Code) | New session opens | `hero next --write` (and round-trip ingest) |

`hero install` extends to write the git hooks (idempotent, marker-
delimited so user content is preserved).

### Merge driver: regenerate, don't merge

`.gitattributes` (committed):

```
.hero/NEXT.md          merge=hero-next
.hero/next/*.md        merge=hero-next
```

`.git/config` (registered by `hero install`):

```ini
[merge "hero-next"]
    name = "Hero — regenerate NEXT.md from graph on conflict"
    driver = "hero next merge-resolve --output %A"
```

`hero next merge-resolve` ignores `%O` (base) and `%B` (theirs),
regenerates from the local graph, writes to `%A`. Result: no
conflict markers ever land in these files for users who've run
`hero install`.

For clones that haven't run `hero install`, git produces normal
conflict markers. The next Stop hook regenerates and self-heals;
agents that see `<<<<<<<` markers in `.hero/NEXT.md` or
`.hero/next/*.md` should run `hero next --write` immediately.

### Migration

One-time `hero next migrate-to-projection` command:

1. Reads existing `.hero/NEXT.md` content.
2. Captures durable items as graph nodes:
   - "Tried and failed" bullets → `Attempt(status=failed)` if not
     already present.
   - "Context to carry forward" pointers → curated `Note` nodes (or
     `keep` flags on existing ones).
   - "Last user ask" frontmatter / section → `UserAsk` node.
3. Writes the new three-file structure.
4. Updates `.gitignore` to add `.hero/next/*.local.md`.
5. Idempotent — second run is a no-op.

## Acceptance criteria

**AC-1:** Three new graph node types exist (`UserAsk`,
`NextSuggestion`, `SessionReflection`), can be created, queried by
attribution to Person+Session, and federate via existing Cloud sync
machinery (when configured). Verified by graph-level tests in
`internal/graph/`.

**AC-2:** `hero next --write` produces a `.hero/NEXT.md` containing
only project-state sections, derived entirely from project-graph
queries. Hand-edits in the file are wiped on next regen. No
`<!-- BEGIN HERO MACHINE STATE -->` markers appear in `NEXT.md`.

**AC-3:** `hero next --write` produces `.hero/next/<user>.md` for
the current user, containing user-state sections derived from the
user's `UserAsk` / `NextSuggestion` / `SessionReflection` /
`Attempt` nodes. File is total-rewrite each turn.

**AC-4:** `hero next --write` produces `.hero/next/<user>.local.md`
with marker-bounded auto sections rewritten in place; any hand-
written content outside the markers is preserved verbatim across
regens.

**AC-5:** Field-grab CLI works:
`hero next suggest` prints the suggested next prompt to stdout (no
decoration); `--json` emits `{"text": "...", "session_id": "..."}`;
`--copy` puts it on the clipboard; `--user <name>` fetches another
person's (or returns "not found"). Same shape for `hero next ask`
and `hero next reflection`.

**AC-6:** Round-trip ingest works end-to-end. Sequence: machine A
writes `chet-bellows.md` via `hero next --write`; commit + push; clean
machine B with empty local graph clones + pulls; machine B's
`hero scan` (or session-start) ingests `chet-bellows.md` into its local
graph; subsequent `hero next suggest` on machine B returns the same
suggested-next text as machine A. Tested by an E2E script in
`tmp/e2e-cross-machine.sh` (or an area suite in `e2e-area-suites`).

**AC-7:** Stop hook (Claude Code's `~/.claude/settings.json` Stop
hook) calls `hero next --write` (not the retired `checkpoint`
command). Verified by `hero install` writing the new command and a
`hero check` health check that flags repos still wired to the old
command.

**AC-8:** `hero install` writes `.git/hooks/pre-commit` and
`.git/hooks/post-merge` that run `hero next --write`. Hooks are
marker-delimited so they preserve any pre-existing user content.
Re-running `hero install` is idempotent.

**AC-9:** `.gitattributes` ships with `.hero/NEXT.md merge=hero-next`
and `.hero/next/*.md merge=hero-next`. `hero install` registers the
`hero-next` driver in `.git/config` pointing to
`hero next merge-resolve --output %A`.

**AC-10:** `hero next merge-resolve` ignores `%O` and `%B`, writes
graph-projected content to `%A`, exits 0. Tested by simulating a
two-branch merge that touches NEXT.md and asserting no conflict
markers in the result.

**AC-11:** Cross-machine continuity verified end-to-end: user-state
written on machine A is queryable on machine B after pull + session
start, with no Cloud configured. Acceptance is the same E2E as AC-6.

**AC-12:** Non-CLI viewer scenario verified: a fresh clone of the
repo, *without* running `hero install`, contains a current
`.hero/NEXT.md` reflecting the project state at the most recent
commit. Asserted by a CI-time check that diffs the committed
`NEXT.md` against `hero next --write` output and fails on drift.

**AC-13:** `hero next migrate-to-projection` ingests existing
NEXT.md content into structured nodes, writes the new three-file
layout, updates `.gitignore`, and is idempotent. One-shot migration
verified on the canonical hero repo.

ACs accrete as edge cases surface during delivery.

## Approach

**Phase 1 — File restructure + total-rewrite projection** (~1 day):
- Move existing machine-state out of `NEXT.md` into
  `.hero/next/<user>.local.md` (marker-bounded zone).
- Create skeleton `.hero/next/<user>.md` projection (initially
  carries the agent-narrative content from current NEXT.md).
- Replace `hero next checkpoint` with `hero next --write`.
- Update Stop hook wire-up in `internal/install/claude_hooks.go`.
- Update `.gitignore` for `.hero/next/*.local.md`.
- Existing `internal/projection/projection.go:NextMD` rewritten or
  joined by `UserHandoffMD` and `LocalStateMD`.
- Ships visible benefit: machine state stops churning NEXT.md.

**Phase 2 — New graph node types** (~½ day):
- Add `UserAsk`, `NextSuggestion`, `SessionReflection` node types
  to graph schema.
- Migration in `internal/graph/graph.go`.
- Helper APIs in `internal/handoff/` (new package): `RecordAsk`,
  `RecordSuggestion`, `RecordReflection`, plus query helpers
  `LatestAsk(person, session)` etc.
- Tests in `internal/handoff/`.

**Phase 3 — Field-grab CLI** (~½ day):
- `hero next suggest` / `ask` / `reflection` / `blockers`.
- `--json` / `--copy` / `--user` flags.
- Tests using the in-process graph store.

**Phase 4 — Projection wiring** (~½ day):
- `projection.NextMD` renders project state only.
- `projection.UserHandoffMD` renders user durable state from new
  node types.
- `projection.LocalStateMD` renders machine state + preserves hand
  content.
- `hero next --write` writes all three.

**Phase 5 — Round-trip ingest** (~1 day):
- `hero scan` reads `.hero/next/<user>.md` if present, ingests
  entries into graph if not already present locally.
- Session-start hook for Claude Code calls `hero scan` (or just the
  ingest portion).
- E2E test `tmp/e2e-cross-machine.sh` validates two-machine
  scenario (uses two ephemeral graph DBs simulating two machines).

**Phase 6 — Git hooks + merge driver** (~½ day):
- `hero install` writes pre-commit, post-merge, session-start hooks.
- `hero install` registers `hero-next` merge driver in `.git/config`.
- `.gitattributes` shipped in repo with merge directives.
- `hero next merge-resolve` command implementation.

**Phase 7 — Migration command** (~½ day):
- `hero next migrate-to-projection` ingests existing NEXT.md content
  into structured graph nodes, writes new layout, updates
  `.gitignore`. Idempotent.

**Phase 8 — Emit-as-you-work skill (deferred)**:
- A skill / agent rule that fires `hero next ask "..."` and
  `hero next suggest "..."` at appropriate moments during work, so
  the user-graph nodes accumulate naturally instead of requiring
  the user to invoke commands.
- Tracked separately because it's behavioral / prompt-engineering
  work, not codebase plumbing.

## Out of scope

- LLM-narrated synthesis of project state into prose ("write me a
  paragraph summarizing where we are"). Separate Tier-3 work.
- Multi-user merge of `<user>.md` files (e.g., showing both Alice
  and Bob's suggested-nexts in one view). Single-user view per
  file; aggregation is `hero next team` (already exists).
- Visualization / web UI. Cloud dashboard work.
- Per-branch user-state separation. The `<user>.md` reflects the
  user's current intent regardless of branch; if you context-switch
  branches mid-work, the file reflects the latest. Refinement for
  later if the simple model bites.

## Open questions

- **Suggested next prompt — how does it get there in the first
  place?** Phase 8 (emit-as-you-work skill) addresses, but we may
  want a manual `hero next suggest set "..."` for explicit capture
  while the skill matures.
- **Local file marker preservation in `<user>.md`:** the design
  says `<user>.md` is total-rewrite (no hand content), but a user
  might reasonably want to scribble in there. Keep total-rewrite
  for v1; add an `extras:` frontmatter field if demand surfaces.
- **What to do when `hero install` hasn't run and a real merge
  conflict lands** — agents should auto-detect markers in NEXT.md
  and run `hero next --write`, but that's a skill rule we need to
  ship alongside Phase 6.
- **Person identity:** today `nextUserSlug()` derives from
  `tracking.defaultAgent` or git user.name. The new `UserAsk` etc.
  attribution should match — verify alignment in Phase 2.
- **Migration timing:** can we ship Phase 1 before Phase 7's
  migration command? Phase 1's projection will collide with the
  existing hand-written content on first run. Plan: Phase 1 ships
  with a pre-flight check that bails if it detects unmigrated
  content; Phase 7's `migrate-to-projection` is the gate. Or just
  ship migration as part of Phase 1.
