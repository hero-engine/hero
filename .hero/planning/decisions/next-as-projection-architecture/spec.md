---
type: decision
status: accepted
title: NEXT-as-Projection Architecture — Three-File Split, Merge Driver, Migration Gate, Drift CI
created: 2026-05-18
shipped-in: v0.10.0
shipped-commits:
  - 522e4a9
  - e7d9594
  - 97c85bd
  - 07a0403
  - 7807117
  - 61efffb
  - ae6da6f
  - 885ed35
  - 0cfe403
relates-to:
  - snapshot-architecture
  - next-as-projection
tags: [decision, post-hoc, architecture, next-md, projection, handoff, merge-conflicts, migration]
---

# NEXT-as-Projection Architecture — Three-File Split, Merge Driver, Migration Gate, Drift CI

## Kickoff

Post-hoc decision record for the v0.10.0 NEXT.md cutover from
hand-maintained handoff document to graph projection. Captures *why* the
subsystem is shaped the way it is — three-file split
(project / per-user durable / per-machine local), Stop-hook total
rewrite, merge driver, migration gate, SessionStart ingest hook, CI
drift gate. The original delivery spec
(`.hero/specs/next-as-projection/spec.md`) carries the full design
narrative; this record exists so a future maintainer extending or
modifying the handoff plumbing doesn't have to reverse-engineer the
choices.

**Status:** accepted — design shipped across the v0.9.2→v0.10.0 commit
window, completed 2026-05-18 (commit `ae6da6f`).

**Pick up at:** read this record before touching the projector entry
points, the merge driver, the SessionStart hook, the CI drift gate, or
the three-file file-shape contract. Cross-check the original delivery
spec for sub-decision rationale (e.g. why `checkpoint` was kept as the
command name, why writers stay positional vs `set` subcommands).

→ `.hero/planning/decisions/next-as-projection-architecture/spec.md`

**Files:** `internal/cli/checkpoint.go`,
`internal/cli/next.go`,
`internal/handoff/handoff.go`,
`internal/install/claude_hooks.go`,
`internal/install/codex_hooks.go`,
`.gitattributes`,
`.github/workflows/test.yml` (drift gate step),
`skills/next-merge-recovery/SKILL.md`,
`.hero/specs/next-as-projection/spec.md`

**Skip:** per-branch user-state separation (out of scope by design); LLM
narration of project state (separate Tier-3 work); a `set` subcommand
form for the field-grab writers (positional form wins by symmetry with
`git config user.email "..."`); renaming `hero next checkpoint` to
`--write` (install-hook prefix matchers and the user-visible
"auto-written by" string would all break for zero behavior gain);
marker-preservation inside `.hero/next/<user>.md` (total-rewrite stays
v1; revisit only on user request).

## Context

NEXT.md was the cross-session handoff artifact — the load-bearing
surface for principle #3 ("sessions start omniscient"). Three observed
failures the pre-projection design could not prevent:

- **F1 — Agent narrative drift.** The "Just finished / Next" sections
  relied on agent discipline to refresh. Demonstrated 2026-04-28: a
  session shipped 7 commits past its last NEXT.md update, including
  the very work it had just proposed. The next session opened to a
  NEXT.md proposing already-completed work. Audit-invisible without
  checking against `git log`.
- **F2 — Mechanical churn → merge conflicts.** The Stop hook
  rewrote the bottom of NEXT.md on every assistant turn. Every branch
  divergence guaranteed a merge conflict on auto-content; `git status`
  was always dirty; `git log -- NEXT.md` was unreadable noise.
- **F3 — Cross-machine and cross-branch aliasing.** Same branch on two
  machines: NEXT.md drifted because each machine wrote its own machine
  state. Switch branches: the file reflected the wrong branch's
  snapshot.

The graph substrate built in graph-memory phases 1–4 was supposed to
prevent all three. Phase 4 (commit `2158bd2`, pre-v0.10) declared
NEXT.md a projection ("hero writes NEXT.md, humans don't") but the
agent-half projection never shipped. This work finished that cutover
and added the file-shape needed to handle solo-no-Cloud cross-machine
continuity through git.

Constraints in play:

- **No Cloud federation in the canonical hero-on-itself case.** The
  shared-graph story everyone else gets via federation had to ride on
  something that traveled in git — meaning a tracked artifact had to be
  the federation medium when no Cloud was configured.
- **Stop hook already existed and was load-bearing.** Replacing it was
  off-limits; the projector had to slot in where the existing
  end-of-turn write fired.
- **Existing users had hand-authored NEXT.md content.** A cutover that
  silently wiped their files on first `hero next checkpoint` would
  destroy work and erode trust in the projection model.
- **Merge conflicts on a Stop-hook-written file are guaranteed across
  branches.** Any solution had to handle them or live with the same
  failure mode F2 it was trying to fix.

## Decision

NEXT.md and its peers become *graph projections* — derived state,
regenerated on every Stop hook, never hand-merged. The design has eight
load-bearing pieces.

### 1. Three files, three audiences

| File | Tracked | Audience | Updates |
|---|---|---|---|
| `.hero/NEXT.md` | yes | Anyone who opens the repo (incl. non-CLI viewers, GitHub) | Total-rewrite each turn from **project** graph |
| `.hero/next/<user>.md` | yes | You, across machines; teammates if any | Total-rewrite from **user-graph** nodes |
| `.hero/next/<user>.local.md` | **no** (gitignored) | You, on this one machine | Marker-bounded auto sections rewritten; hand content preserved outside markers |

Audiences are deliberately split. Project state has no "Last user ask"
or "Suggested next" — those are user state. User state has no machine
specifics — those are local. Local state mixes auto and hand content
because the user wants one place to glance at "what's my situation
right now," not two.

### 2. Three new graph node types

`UserAsk`, `NextSuggestion`, `SessionReflection` — each attributed to a
`Person` and a `Session`. Person attribution **reuses the existing
`nextUserSlug()` helper** (derived from `tracking.defaultAgent` or git
`user.name`) — no new identity resolution path, interoperable with the
existing Person identity scheme. All three federate via Cloud sync
(when present) and round-trip through the tracked `<user>.md` file
(when not). Bitemporal storage gives "what was suggested-next on
Tuesday at home?" for free.

### 3. Total-rewrite projection on every Stop hook

`hero next checkpoint` regenerates all three files from the graph (and
from machine state for the local file). No marker preservation in
NEXT.md or `<user>.md` — projections always win, hand-edits in those
files are wiped next turn. The `<user>.local.md` file is the
exception: marker-bounded auto sections rewritten in place, hand
content preserved verbatim. **Command name stays `hero next checkpoint`**
— renaming to `--write` was considered and rejected because the install
detection prefix in `internal/install/claude_hooks.go` and
`codex_hooks.go` (`strings.HasPrefix(cmd, "hero next checkpoint")`) and
the user-visible "auto-written by" string would all need to change for
zero behavior gain.

### 4. The `next.projected` flag is the single authoritative migration signal

`next.projected` boolean in `.hero/hero.json`, written by
`hero next migrate-to-projection` and read by `Config.NextProjected()`.
The projection write path (`writeProjectedNextMD` in `checkpoint.go`)
only fires when the flag is true. No file-level sentinel needed — the
config flag is simpler, atomic, and lives where every other config
already lives.

### 5. Pre-flight migration gate

When `next.projected == false` AND `.hero/NEXT.md` contains unmigrated
content, `hero next checkpoint` *refuses* to write NEXT.md and exits
non-zero with a message directing the user to
`hero next migrate-to-projection`. Detection signals
(`detectUnmigratedNextMD` in `internal/cli/checkpoint.go`):

- `<!-- BEGIN HERO MACHINE STATE -->` markers in `.hero/NEXT.md` (legacy
  location for machine state; new home is `.hero/next/<user>.local.md`).
- Legacy section headers (`## Just finished`, `## Next`, `## Tried and
  failed`, `## Context to carry forward`) containing real content.
  Italic placeholder lines (the `nextPlaceholder()` "agent fills this
  in" convention) don't count as real content.

`hero next migrate-to-projection` reads existing NEXT.md, captures
durable items as graph nodes ("Tried and failed" → failed `Attempt`,
"Context to carry forward" → curated `Note`, "Last user ask" →
`UserAsk`), writes the new three-file structure, updates `.gitignore`
to add `.hero/next/*.local.md`, sets `next.projected = true`. Idempotent
— second run is a no-op.

### 6. `hero-next` git merge driver

`.gitattributes` (committed):

```
.hero/next/*.md merge=hero-next
.hero/NEXT.md merge=hero-next
.hero/QUEUE.md merge=hero-next
```

`.git/config` (registered by `hero install`):

```ini
[merge "hero-next"]
    name = "Hero — regenerate NEXT.md from graph on conflict"
    driver = "hero next merge-resolve --output %A"
```

`hero next merge-resolve` *ignores* `%O` (base) and `%B` (theirs),
regenerates from the local graph, writes to `%A`. Result: no conflict
markers ever land in these files for users who've run `hero install`.

Clones that haven't run `hero install` (fresh checkouts, CI, teammates
picking up the repo) get standard conflict markers. The next Stop hook
catches them and regenerates, and the `next-merge-recovery` skill
(`skills/next-merge-recovery/SKILL.md`) instructs agents to detect
`<<<<<<<` in these paths and run `hero next checkpoint` immediately —
don't hand-resolve.

### 7. SessionStart `hero next ingest` hook for cross-machine round-trip

Without Cloud, `<user>.md` is the federation medium. Round-trip is
bidirectional:

- `hero next checkpoint` projects user-graph nodes → `<user>.md` file.
- `hero next ingest` reads `<user>.md` back into the graph if entries
  aren't already present locally. Idempotent via content-hash dedup
  keyed on `(node type, session_id, sha256(text))`.

`hero install` wires four hook surfaces:

| Hook | Trigger | Runs |
|---|---|---|
| Stop hook (Claude Code, Codex, opencode) | Every assistant turn ends | `hero next checkpoint --quiet` |
| Pre-commit (git) | Right before `git commit` | `hero next checkpoint`; stage changes |
| Post-merge (git) | After `git merge` / `git pull` | `hero next checkpoint` |
| SessionStart (Claude Code + Codex) | New session opens | `hero next ingest --quiet` (round-trip ingest) |

Result: home laptop writes, commits, pushes; office desktop pulls,
session-start hook ingests `<user>.md` into its local graph, then
projects it back out on the next Stop hook. Same content visible on
both machines. The detection prefix
(`internal/install/claude_hooks.go` `heroCmdPrefixes`) recognises both
`hero next checkpoint` and `hero next ingest` so re-running
`hero install` upgrades older entries idempotently.

### 8. CI drift gate

`.github/workflows/test.yml` rebuilds the graph and regenerates NEXT.md,
failing the build on any drift between the committed file and the
projection output:

```yaml
- name: NEXT.md projection drift gate
  run: |
    set -e
    go build -o ./hero ./cmd/hero
    ./hero scan -q || true
    ./hero next checkpoint --quiet
    if ! git diff --exit-code -- .hero/NEXT.md; then
      echo "::error::.hero/NEXT.md drifted..."
      exit 1
    fi
```

Catches Stop-hook regressions, contributors who bypass the pre-commit
hook, and projector logic changes that don't carry the regenerated
file. The graph DB is gitignored so it gets rebuilt via `hero scan` on
each CI run — same path a fresh clone follows after `hero install`.

### 9. Field-grab CLI for friction-free read/write

For when scrolling NEXT.md is too much friction:

```
hero next suggest                    # prints just the suggested next prompt
hero next suggest --copy             # also copies to clipboard
hero next suggest --json             # structured output for scripting
hero next suggest --user alice       # someone else's (team mode)
hero next suggest "..."              # write a NextSuggestion node manually

hero next ask        / hero next ask "..."
hero next reflection / hero next reflection "..."
hero next blockers
```

All reads go *directly to the graph* — always current even if the file
projection hasn't fired yet on this machine. Writers use positional-arg
form (matching `git config user.email "..."`'s read/write overload),
not a separate `set` subcommand — adding `set` would create a redundant
third shape and confusing UX.

## Alternatives considered

### A1 — Keep NEXT.md hand-authored, ask agents to refresh harder (rejected — empirical, F1 demonstrates failure)

Pre-projection, NEXT.md *was* hand-authored by agents. The 2026-04-28
audit (cited in the delivery spec) demonstrated a session shipping 7
commits past its last NEXT.md update, including the very work it had
just proposed. Agent discipline does not fix this; it's the same
"two-truth-sync" anti-pattern that motivates projection elsewhere.
Rejected on evidence.

### A2 — Single tracked NEXT.md with both project and user state (rejected — explicit in the delivery spec)

Keep everything in one file. Pro: single artifact to read. Con: every
machine writes its own machine state into a tracked file, guaranteeing
F3 drift every time two machines work on the same branch. Splitting
project state (tracked, projected, shared) from user durable state
(tracked, per-user, projected) from machine state (gitignored, per-
machine, marker-bounded) is the only file shape that resolves all
three failures without trading one for another.

### A3 — Marker-bounded partial rewrite of tracked files (rejected — explicit in the delivery spec)

Preserve hand-written content in `<user>.md` and `NEXT.md` between
markers; rewrite only the auto zones. Pro: hand-edits survive
regen. Con: the user-graph nodes are the source of truth; an in-file
edit creates a fourth surface to reconcile. The shipped design ships
total-rewrite for v1 with explicit "revisit if any user reports
wanting to keep hand-written content in `<user>.md`" — the deferred
fix is a small `extras:` frontmatter field or a small marker-bounded
zone. The `<user>.local.md` file already provides a hand-content
surface for users who need one today.

### A4 — Per-branch user-state separation (rejected — explicit in delivery spec "Out of scope")

Make `<user>.md` carry per-branch suggested-nexts so context-switching
branches doesn't show stale intent. Pro: surgical context preservation
across branch boundaries. Con: doubles the file count (per-user × per-
branch), introduces a stale-on-branch-delete failure mode, and the
common case is users whose last intent is "what I was doing across the
project" regardless of which branch they were on when. Rejected for
v1; deferred until the simple model bites.

### A5 — Rename `hero next checkpoint` to `hero next write` or similar (rejected — explicit in the delivery spec)

A rename was considered. Rejected because install-hook prefix matchers
(`strings.HasPrefix(cmd, "hero next checkpoint")` in both
`claude_hooks.go` and `codex_hooks.go`) and the user-visible
"auto-written by" string baked into NEXT.md's footer would all need to
change. Deprecation shim + hook-detection regex updates + doc churn
for zero behavior change.

### A6 — `set` subcommand instead of positional-arg writers (rejected — explicit in the delivery spec)

`hero next suggest set "<text>"` would have made the read/write
distinction explicit. Rejected as redundant: positional-arg form
(`hero next suggest "<text>"`) matches the read/write overload of
`git config user.email "..."` and avoids a third shape. The manual
writer covers the capture-by-hand need without the extra surface.

### A7 — Cloud-only federation, no tracked `<user>.md` (rejected — implicit; the canonical hero-on-itself case)

Rely on Cloud sync for cross-machine continuity; skip the tracked
`<user>.md` file. Rejected because the canonical hero-on-itself case
has no Cloud configured. Solo developers with two machines (home /
office) are the demonstrated use case, and they need round-trip via
git. Once Cloud is present, both paths coexist — the tracked file
becomes a no-cost backup; the Cloud-mediated sync is the hot path.

### A8 — File-level sentinel instead of `next.projected` config flag (inferred — not explicit in the delivery spec)

A sentinel comment at the top of `.hero/NEXT.md`
(`<!-- HERO PROJECTED -->`) could have signaled "this file is
projection output, safe to overwrite." Inferred from the design's
structure: the delivery spec says "no file-level sentinel is needed —
the config flag is simpler, atomic, and already shipping." The flag
wins on (a) atomicity (config is a single value, the file is bytes),
(b) location (config holds every other knob), and (c) the existing
write path was already gated on it before the cutover finished.

### A9 — Keep merge conflicts; ask users to resolve by hand (rejected — empirical, F2 demonstrates the cost)

Pre-projection this was the lived experience and the source of F2.
The merge driver eliminates the entire class of conflicts for users
who run `hero install`; the recovery skill catches the remainder. The
two-layer solution (driver for installed clones, skill+Stop-hook for
the rest) covers both populations.

### A10 — Skip the CI drift gate (rejected — implicit; AC-12 makes it explicit)

The pre-commit hook is supposed to keep `.hero/NEXT.md` current.
Without the drift gate, a contributor who bypasses the hook
(`git commit --no-verify`, or a contributor who hasn't run
`hero install`) silently lands stale projections. The CI gate is the
backstop. The trade-off — CI builds a graph DB and runs the projector
on every PR — is paid once per build and catches a class of subtle
errors at PR time, before they merge.

## Consequences

### What this enables

- **NEXT.md is now true.** Cold-start agents read what's actually in
  the graph, not a stale narrative. The "sessions start omniscient"
  principle is enforceable rather than aspirational.
- **Solo cross-machine continuity without Cloud.** Home laptop writes
  → push → pull → office desktop ingests → same state. No federation
  layer required; git is the medium.
- **Non-CLI viewers get current project state from GitHub.** The
  tracked, projected `.hero/NEXT.md` renders cleanly on GitHub and
  shows the actual current state of the repo to anyone browsing.
- **Branch divergence stops producing merge conflicts on NEXT.md.**
  The driver resolves cleanly; the recovery skill catches edge cases.
- **A projector framework now exists for other tracked-but-projected
  artifacts.** `project-snapshot` (shipped one commit later) reuses
  the entire shape: total-rewrite, merge driver, Stop-hook integration,
  `hero install` hook flow, migration-gate pattern. The marginal cost
  of the next projector is small.
- **Field-grab CLI lets agents and users skip the file entirely.**
  `hero next suggest` reads directly from the graph, always current.

### What this locks in

- **`hero next checkpoint` runs on every Stop hook on every assistant
  turn.** Combined with `project-snapshot` (which now also fires from
  this same checkpoint), the budget is ~190ms warm against a 250ms
  target. Anything else that wants to share the Stop hook needs to fit
  inside the same window or accept the cost.
- **Three-file layout is a public contract.** Any tool reading
  `.hero/NEXT.md` (GitHub renderers, IDE integrations, executive
  reports) sees project state only; tools reading `.hero/next/<user>.md`
  see one user's durable state; `<user>.local.md` is private and
  gitignored. Restructuring is a breaking change for every consumer.
- **`nextUserSlug()` is the identity primitive.** All three new node
  types attribute via this helper. Any future per-user node should
  use it too; introducing a parallel identity resolution would
  fragment the Person graph.
- **The migration gate must keep firing for unmigrated repos
  indefinitely.** Removing it would silently wipe hand-authored
  content the first time `hero next checkpoint` ran on a repo where
  someone skipped `hero next migrate-to-projection`. The check is
  cheap and the cost of removing it is unbounded.
- **The CI drift gate is part of the workflow.** Anyone who removes
  the workflow step removes the contract that committed NEXT.md
  matches projector output. Removal should be loudly noticed.
- **The `hero-next` merge driver is now also used by `.hero/SNAPSHOT.md`
  and `.hero/QUEUE.md`.** Generalization happened post-shipping; the
  driver name no longer reflects its scope. Future projected files
  inherit by adding a `merge=hero-next` line in `.gitattributes`.

### Operational properties future maintainers need to know

- **Stop-hook failure is observable, not silent.** A failed
  `hero next checkpoint` writes to stderr (the user sees it on next
  turn) and leaves the previous file in place. The legacy fallback
  path inside `writeCheckpoint` exists for the case where projection
  fails — it produces *something* rather than nothing.
- **Content-hash skip applies to all three files.** A graph that
  hasn't moved produces zero writes per Stop hook. `git status` stays
  clean across quiet turns.
- **`hero next ingest` is idempotent.** Content-hash dedup on
  `(node type, session_id, sha256(text))` means re-running ingest on
  the same `<user>.md` produces no new nodes. Safe to fire from
  SessionStart on every session open without growing the graph.
- **Cross-machine race resolves last-writer-wins with no data loss.**
  Two machines edit `<user>.md` in parallel before either pushes; the
  merge driver picks the local graph on each side; on next pull, the
  loser's content survives via the next round-trip ingest. Both sets
  of nodes re-converge; eventual consistency.
- **Hand-content backup before discard.** When the local file has
  non-empty hand-content outside the markers,
  `backupHandContentIfNeeded` writes a `.bak` once before discard —
  defense against accidental loss during the first cutover.
- **Performance budget is empirical and load-bearing.** Measured
  150–180ms warm on the hero repo's own graph (200ms target). Cold-
  cache first run is ~540ms (SQLite open + initial reads); this is
  paid once per process, not per checkpoint. Profile-and-revisit if
  graphs grow materially.
- **`hero check` is the validation surface.** Confirms the Stop hook
  points to `hero next checkpoint`, git hooks are installed, the
  `hero-next` merge driver is registered. Run after every
  `hero install`.
- **The recovery skill (`skills/next-merge-recovery/SKILL.md`) is part
  of the user-visible contract.** Removing it would force agents to
  reinvent the "regenerate, don't merge" instinct on conflict
  markers. Update it if the file shape or commands change.

## Open follow-ups

Noticed during the reverse-engineer; not blockers, worth recording so
they aren't lost.

1. **Hand-content support in `<user>.md` is deferred.** v1 ships
   total-rewrite, no marker preservation. Revisit if any user reports
   wanting to keep hand-written content there; the likely fix is an
   `extras:` frontmatter field or a small marker-bounded zone.
2. **Per-branch user-state separation is out of scope.** The
   `<user>.md` reflects the user's current intent regardless of branch;
   if you context-switch branches mid-work, the file reflects the
   latest. Refinement deferred until the simple model bites.
3. **The "emit-as-you-work skill" (Phase 7) is deferred behavioral
   work.** A skill/agent rule that fires `hero next ask "..."` and
   `hero next suggest "..."` at appropriate moments so the user-graph
   nodes accumulate naturally instead of requiring manual command
   invocation. Tracked separately because it's prompt engineering,
   not codebase plumbing. Until it lands, the manual writer path is
   the capture mechanism.
4. **Codex SessionStart support is best-effort.** The delivery spec
   notes Codex documents support for both Stop and SessionStart, but
   if a Codex client ignores SessionStart the hook is benign. Verify
   on real Codex usage; document the gap if it manifests.
5. **The drift gate runs after `hero scan -q`.** This means schema
   changes that affect projection output need a corresponding scan
   run on CI. If `hero scan` ever becomes non-idempotent or slow on
   the CI runner, the gate logic will need rethinking.
6. **A bug-spec exists for narrative accumulation in `.local.md`**
   (commit `0cfe403`: "stop .local.md narrative accumulation + repo-
   scope handoff reads"). The fix was applied late in the cycle;
   future contributors editing `rebuildLocalState` should re-read the
   commit message to avoid regressing the same path.
7. **A separate bug-spec covers cross-repo handoff pollution** (commit
   `cdbdc6e`: "archive next-checkpoint-cross-repo-pollution → specs/").
   The next-checkpoint write path needs to stay scoped to the current
   repo's graph; cross-repo work should land via the cross-repo-peering
   surface, not as a side-effect of next-checkpoint.

## Provenance and gaps

Sources mined for this record:

- `.hero/specs/next-as-projection/spec.md` — the delivery spec
  (724 lines); unusually thorough on rationale (each design choice
  has a paragraph explaining why alternative shapes were rejected).
- Commits in the v0.9.2..v0.10.0 window matching `next-as-projection`
  / `next-merge-recovery` / `next-projection` / `checkpoint`:
  `522e4a9` (SessionStart hook), `e7d9594` (CI drift gate),
  `97c85bd` (recovery skill), `07a0403` (migration gate),
  `7807117` (audit correction), `61efffb` (spec completion),
  `ae6da6f` (spec archive), `885ed35` (post-fix sweep),
  `0cfe403` (.local.md narrative accumulation fix).
- `.gitattributes` (merge driver entry, now generalized across NEXT,
  QUEUE, SNAPSHOT, and `next/*.md`).
- `.github/workflows/test.yml` (drift gate step).
- `internal/cli/checkpoint.go` (the projector entry point including
  the pre-flight migration gate `detectUnmigratedNextMD` and the
  projector-side snapshot integration).
- `internal/cli/next.go` (CLI surface).
- `internal/install/claude_hooks.go` and
  `internal/install/codex_hooks.go` (Stop + SessionStart + PreCompact
  hook wiring, prefix matchers for idempotent re-install).
- `skills/next-merge-recovery/SKILL.md` (recovery protocol).

**Rationale gaps to flag:**

- **A8 (file-level sentinel vs config flag) is inferred.** The
  delivery spec confirms the config flag was chosen and explains why
  ("simpler, atomic, and already shipping") but does not explicitly
  enumerate the file-level sentinel alternative. The framing here is
  inferred from the design's structure; the rationale is sourced from
  the delivery spec.
- **A7 (Cloud-only federation) is inferred.** The delivery spec
  speaks throughout about "solo-no-Cloud cross-machine continuity"
  but does not stage Cloud-only as an explicit rejected alternative.
  Inferred from the framing of `<user>.md` as "the cross-machine
  federation medium when no Cloud is configured."
- Otherwise the alternatives are quoted or paraphrased directly from
  the delivery spec; no fabricated history.
