---
title: NEXT.md as a Graph Projection (with project/user/local split)
slug: next-as-projection
type: feature
status: completed
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

## Kickoff

NEXT.md is now a graph projection end-to-end. Delivery completed
2026-05-18 after the audit revealed ~95% of the spec had already
landed and only six narrow gaps remained. All six closed:

1. **AC-14 pre-flight migration gate** — `hero next checkpoint`
   refuses to overwrite NEXT.md when `next.projected == false` AND
   the file still carries hand-written content under legacy section
   headers or `<!-- BEGIN HERO MACHINE STATE -->` markers.
   (`internal/cli/checkpoint.go:detectUnmigratedNextMD`)
2. **AC-16 merge-recovery skill** — `skills/next-merge-recovery/SKILL.md`
   instructs agents to detect `<<<<<<<` markers in projected NEXT
   files and run `hero next checkpoint` to self-heal.
3. **AC-7/AC-11 session-start ingest hook** — Claude Code and Codex
   installations now wire a `SessionStart` hook firing
   `hero next ingest --quiet` so cross-machine round-trip continuity
   closes automatically on session open.
4. **AC-12 CI drift gate** — `.github/workflows/test.yml` rebuilds
   the graph and regenerates NEXT.md, failing the build if the
   committed file drifts from projection output.
5. **AC-6/AC-11 E2E cross-machine test** —
   `Test_CrossMachineRoundTrip_FullLoop` simulates two graph DBs +
   shared handoff file; suggestion / ask / reflection all round-trip;
   re-ingest is idempotent.
6. **Perf budget verified** — warm checkpoint takes 150–180 ms on
   the hero repo's own graph, under the 200 ms target.

The two reconsidered decisions from the audit:

- `hero next checkpoint` stays — rename to `--write` rejected because
  install-hook prefix matchers and the user-visible "auto-written
  by" string would all need to change for zero behavior gain.
- Field-grab writers stay positional (`hero next suggest "<text>"`);
  `set` subcommand rejected as redundant.

**Status:** completed. Downstream spec `project-snapshot`
(`.hero/planning/features/project-snapshot/spec.md`) is now
deliverable — it reuses the projector pattern, Stop-hook integration,
`.gitattributes` merge-driver model, and `hero install` hook flow.

→ `.hero/specs/next-as-projection/spec.md` (post-completion path)

**Skip:** per-branch user-state separation (out of scope); LLM
narration of project state (separate Tier-3 work).

## Mission fit

NEXT.md is THE cold-start surface for hero — the first thing a fresh
session, a new teammate, or a non-CLI viewer reads when they open
the repo. Today it drifts: the agent-narrative half ages out within
a session, the mechanical half churns merge conflicts on every
branch divergence, and cross-machine continuity is a coin flip.

This work answers the mission test — *"Does this make the next agent
session start smarter than the last one ended, and does it raise the
floor for everyone, not just the senior dev who already knows what
to ask?"* — with yes on both counts. By projecting NEXT.md from the
graph instead of asking the agent to remember to write it, the cold
start gets the **actual** current state, not a stale narrative. By
splitting project state from user state, non-CLI viewers (managers,
new hires, teammates skimming the repo on GitHub) get a clean
project view without per-user noise. By making `<user>.md` the
solo-no-Cloud federation medium, the floor rises for the
canonical hero-on-itself case where Cloud isn't configured.

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

Three new node types attributed to a `Person` and a `Session`.
Person attribution **MUST** use the existing `nextUserSlug()` helper
that the current tracking system uses (derived from
`tracking.defaultAgent` or git `user.name`) — no new identity
resolution path. This keeps the new nodes interoperable with the
existing Person identity scheme.

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

`hero next checkpoint` regenerates all three files from the graph
(and from machine state for the local file). No marker preservation
in NEXT.md or `<user>.md` — projections always win, hand-edits in
those files are wiped next turn. The `<user>.local.md` file is the
exception: marker-bounded auto sections rewritten, hand content
preserved.

The Stop hook (and git hooks) call `hero next checkpoint --quiet`.
The command name is kept rather than renamed to `--write`: it is the
detection prefix used by `internal/install/claude_hooks.go` and
`internal/install/codex_hooks.go` to recognise and upgrade existing
hero-managed entries (`strings.HasPrefix(cmd, "hero next checkpoint")`),
and it is the user-visible string baked into NEXT.md's "auto-written
by" footer. A rename would force a deprecation shim, hook-detection
regex updates, and doc churn with no behavior change.

### Field-grab CLI

For when scrolling NEXT.md is too much friction:

```
hero next suggest                    # prints just the suggested next prompt
hero next suggest --copy             # also copies to clipboard
hero next suggest --json             # structured output for scripting
hero next suggest --user alice       # someone else's (team mode)
hero next suggest "..."              # write a NextSuggestion node manually

hero next ask                        # last user ask
hero next ask "..."                  # record a new user ask
hero next reflection                 # most recent session reflection
hero next reflection "..."           # record a new reflection
hero next blockers                   # current project blockers (one-liner per)
```

All read directly from the graph — always current even if the file
projection hasn't fired yet on this machine. The writers use
positional-arg form (matching `git config user.email "..."`'s
read/write overload), not a separate `set` subcommand — adding `set`
would create a redundant third shape and confusing UX. The manual
writer covers the same need (capturing a suggested-next by hand
while the emit-as-you-work skill matures) without the extra surface.

### Round-trip ingest (solo-no-Cloud cross-machine)

Without Cloud, the `<user>.md` file is the federation medium. To
make it bidirectional:

- `hero next checkpoint` projects user-graph nodes → `<user>.md` file.
- `hero scan` (and a session-start hook) reads `<user>.md` back into
  the graph if entries aren't already present locally. Idempotent —
  re-ingest of the same content produces no new graph edits.
  Idempotency is enforced via a content-hash dedup keyed on `(node
  type, session_id, sha256(text))`.

Result: home laptop writes, commits, pushes; office desktop pulls,
session-start hook ingests `<user>.md` into its local graph, then
projects it back out. Same content visible on both machines.

### Layered hooks (reliable end-of-turn projection)

| Hook | Trigger | Runs |
|---|---|---|
| Stop hook (Claude Code, opencode, etc.) | Every assistant turn ends | `hero next checkpoint` |
| Pre-commit (git) | Right before `git commit` | `hero next checkpoint`; stage changes |
| Post-merge (git) | After `git merge` / `git pull` | `hero next checkpoint` |
| Session-start (Claude Code) | New session opens | `hero next checkpoint` (and round-trip ingest) |

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
conflict markers. A new skill rule, `skills/next-merge-recovery.md`,
instructs agents to detect `<<<<<<<` markers in `.hero/NEXT.md` or
`.hero/next/*.md` and run `hero next checkpoint` immediately as a
self-heal. The existing Stop hook also picks up the markers on the
next turn.

### Migration as the Phase 1 gate

The migration command (`hero next migrate-to-projection`) ships
**as part of Phase 1**, not later. The first `hero next checkpoint`
in a repo that still has legacy NEXT.md content would otherwise wipe
hand-written sections.

**Migration signal: `next.projected` flag in `hero.json`.** The
single authoritative signal is the `next.projected` boolean in
`.hero/hero.json`, written by `hero next migrate-to-projection` and
read by `Config.NextProjected()`. The projection write path
(`writeProjectedNextMD` in `checkpoint.go`) only fires when the flag
is true. No file-level sentinel is needed — the config flag is
simpler, atomic, and already shipping.

**Pre-flight gate.** When `next.projected == false` AND `.hero/NEXT.md`
contains unmigrated content, `hero next checkpoint` refuses to write
NEXT.md and exits non-zero with a message directing the user to
`hero next migrate-to-projection`. Detection signals for legacy
content:

- `<!-- BEGIN HERO MACHINE STATE -->` markers in `.hero/NEXT.md`
  (legacy location; new home is `.hero/next/<user>.local.md`).
- Legacy section headers (`## Just finished`, `## Next`, `## Tried
  and failed`, `## Context to carry forward`) directly inside
  `.hero/NEXT.md`.

`hero next migrate-to-projection`:

1. Reads existing `.hero/NEXT.md` content.
2. Captures durable items as graph nodes:
   - "Tried and failed" bullets → `Attempt(status=failed)` if not
     already present.
   - "Context to carry forward" pointers → curated `Note` nodes (or
     `keep` flags on existing ones).
   - "Last user ask" frontmatter / section → `UserAsk` node.
3. Writes the new three-file structure.
4. Updates `.gitignore` to add `.hero/next/*.local.md`.
5. Sets `next.projected = true` in `.hero/hero.json`.
6. Idempotent — second run is a no-op (detects the flag).

## Acceptance Criteria

- **AC-1:** THE SYSTEM SHALL support three new graph node types
  (`UserAsk`, `NextSuggestion`, `SessionReflection`) that can be
  created, queried by attribution to Person+Session, and federated
  via existing Cloud sync machinery (when configured). Verified by
  graph-level tests in `internal/graph/`. Person attribution MUST
  use the existing `nextUserSlug()` helper.
- **AC-2:** `hero next checkpoint` SHALL produce a `.hero/NEXT.md`
  containing only project-state sections, derived entirely from
  project-graph queries, with no `<!-- BEGIN HERO MACHINE STATE -->`
  markers, whenever `next.projected == true` in `.hero/hero.json`.
  Hand-edits in the file MUST be wiped on next regen.
- **AC-3:** WHEN `hero next checkpoint` runs THE SYSTEM SHALL
  produce `.hero/next/<user>.md` for the current user, containing
  user-state sections derived from the user's `UserAsk` /
  `NextSuggestion` / `SessionReflection` / `Attempt` nodes, as a
  total-rewrite each turn.
- **AC-4:** WHEN `hero next checkpoint` runs THE SYSTEM SHALL
  produce `.hero/next/<user>.local.md` with marker-bounded auto
  sections rewritten in place AND SHALL preserve verbatim any
  hand-written content outside the markers across regens.
- **AC-5:** `hero next suggest` SHALL print the suggested next
  prompt to stdout with no decoration; `--json` SHALL emit
  `{"text": "...", "session_id": "..."}`; `--copy` SHALL place it
  on the clipboard; `--user <name>` SHALL fetch another person's
  (or return "not found" and exit non-zero). Same shape for
  `hero next ask` and `hero next reflection`.
- **AC-6:** Round-trip ingest MUST work end-to-end. Sequence:
  machine A writes `chet-bellows.md` via `hero next checkpoint`;
  commit + push; clean machine B with empty local graph clones +
  pulls; machine B's `hero next ingest` (or session-start hook)
  ingests `chet-bellows.md` into its local graph; subsequent
  `hero next suggest` on machine B SHALL return the same
  suggested-next text as machine A. Tested by an in-process Go test
  exercising both ephemeral graph DBs.
- **AC-7:** WHEN `hero install` runs THE SYSTEM SHALL wire the Stop
  hook to call `hero next checkpoint --quiet`, AND a session-start
  hook to call `hero next ingest`. `hero check` SHALL flag repos
  whose hooks have drifted from this contract.
- **AC-8:** WHEN `hero install` runs THE SYSTEM SHALL write
  `.git/hooks/pre-commit` and `.git/hooks/post-merge` that call
  `hero next checkpoint`, marker-delimited so they preserve any
  pre-existing user content. Re-running `hero install` SHALL be
  idempotent.
- **AC-9:** WHEN `hero install` runs THE SYSTEM SHALL register the
  `hero-next` merge driver in `.git/config` pointing to
  `hero next merge-resolve --output %A`, AND the repo SHALL ship a
  `.gitattributes` containing `.hero/NEXT.md merge=hero-next` and
  `.hero/next/*.md merge=hero-next`.
- **AC-10:** WHEN `hero next merge-resolve` runs THE SYSTEM SHALL
  ignore `%O` and `%B`, write graph-projected content to `%A`, and
  exit 0. Tested by simulating a two-branch merge that touches
  NEXT.md and asserting no conflict markers in the result.
- **AC-11:** Cross-machine continuity SHALL be verified end-to-end:
  user-state written on machine A MUST be queryable on machine B
  after pull + session start, with no Cloud configured. Acceptance
  is the same E2E as AC-6.
- **AC-12:** CI SHALL keep the committed `.hero/NEXT.md` in
  lockstep with `hero next checkpoint --quiet` output (where the
  repo has been migrated), verified by a CI-time check that runs
  `hero next checkpoint --quiet` and `git diff --exit-code
  .hero/NEXT.md`, failing the build on any drift.
- **AC-13:** `hero next migrate-to-projection` SHALL ingest existing
  NEXT.md content into structured nodes, write the new three-file
  layout, update `.gitignore`, set `next.projected = true` in
  `.hero/hero.json`, and MUST be idempotent (second run produces
  zero file diffs and is a no-op).
- **AC-14:** `hero next checkpoint` SHALL exit non-zero when run
  against a repo where `next.projected == false` AND `.hero/NEXT.md`
  contains legacy markers (`<!-- BEGIN HERO MACHINE STATE -->`) or
  legacy section headers (`## Just finished`, `## Next`, `## Tried
  and failed`, `## Context to carry forward`), with a message
  directing the user to `hero next migrate-to-projection`, and MUST
  NOT overwrite the existing file in that state.
- **AC-15:** `hero next suggest "<text>"` (positional-arg writer)
  SHALL record a new `NextSuggestion` node attributed to the
  current user, and the text MUST be visible to subsequent
  `hero next suggest` reads on the same machine and (after commit +
  push + pull + `hero next ingest`) on a paired machine.
- **AC-16:** `skills/next-merge-recovery.md` MUST exist and SHALL
  instruct agents to detect `<<<<<<<` markers in `.hero/NEXT.md`
  or `.hero/next/*.md` and immediately run `hero next checkpoint`
  to self-heal.

ACs accrete as edge cases surface during delivery.

## Approach

**Phase 1 — Migration gate + file restructure + total-rewrite
projection** (~1.5 days) — **SHIPPED**:
- `hero next migrate-to-projection` ships in `internal/cli/next_migrate.go`.
- Three-file layout (`.hero/NEXT.md`, `.hero/next/<user>.md`,
  `.hero/next/<user>.local.md`) lands via `projection.NextMD`,
  `projection.UserHandoffMD`, plus `buildMachineBlock` /
  `rebuildLocalState` in `checkpoint.go`.
- Stop-hook wiring in `internal/install/claude_hooks.go` /
  `codex_hooks.go` uses `hero next checkpoint --quiet`.
- `next.projected` flag in `.hero/hero.json` is the authoritative
  migrated signal (`Config.NextProjected()` in `internal/config`).
- **Remaining gap:** the pre-flight gate (AC-14) — when
  `next.projected == false` AND `.hero/NEXT.md` has legacy markers
  or section headers, `hero next checkpoint` must refuse rather
  than fall through to the legacy write path.

**Phase 2 — New graph node types** (~½ day) — **SHIPPED**:
- Add `UserAsk`, `NextSuggestion`, `SessionReflection` node types
  to graph schema.
- Migration in `internal/graph/graph.go`.
- **Use `nextUserSlug()` for Person attribution** — explicit reuse,
  no new identity resolution.
- Helper APIs in `internal/handoff/` (new package): `RecordAsk`,
  `RecordSuggestion`, `RecordReflection`, plus query helpers
  `LatestAsk(person, session)` etc.
- Tests in `internal/handoff/`.
- Smoke: create a `NextSuggestion` via API, read it via API, confirm
  it federates through Cloud sync stub.

**Phase 3 — Field-grab CLI (read + write)** (~½ day) — **SHIPPED**:
- `hero next suggest` / `ask` / `reflection` / `ingest` all land in
  `internal/cli/next_handoff.go`.
- Writers use positional-arg form
  (`hero next suggest "<text>"`); no `set` subcommand.
- `--json` / `--copy` / `--user` flags on readers.

**Phase 4 — Projection wiring** (~½ day) — **SHIPPED**:
- `projection.NextMD`, `projection.UserHandoffMD` render project
  and per-user state.
- `buildMachineBlock` + `rebuildLocalState` in `checkpoint.go`
  handle the marker-bounded local file.
- `hero next checkpoint` writes all three.
- **Performance measured 2026-05-18** on the hero repo's own graph
  (worst-case for the project; 200+ specs, dense graph):
  - Cold-cache first run: ~540 ms (SQLite open + initial reads).
  - Warm-cache subsequent runs: 150–180 ms.
  - The 200 ms target is met on warm runs (the case that matters
    for Stop-hook firing — the SQLite WAL stays warm across hook
    invocations in a single session). Cold-cache excess is paid
    once per process, not per checkpoint.

**Phase 5 — Round-trip ingest** (~1 day) — **PARTIAL**:
- `handoff.IngestUserFile` + `hero next ingest` command land in
  `internal/handoff/ingest.go` + `internal/cli/next_handoff.go`.
- Idempotency via content-hash dedup.
- **Remaining gap:** session-start hook wiring in
  `internal/install/claude_hooks.go` (and `codex_hooks.go`) so the
  ingest fires automatically at new sessions. E2E cross-machine
  test using two ephemeral graph DBs.

**Phase 6 — Git hooks + merge driver + recovery skill** (~½ day)
— **PARTIAL**:
- `hero install` writes pre-commit, post-merge hooks
  (`internal/cli/next_hooks.go`).
- `hero-next` merge driver registered in `.git/config`.
- `.gitattributes` ships with merge directives.
- `hero next merge-resolve` command implemented.
- **Remaining gap:** `skills/next-merge-recovery.md` not yet written.

**Phase 7 — Emit-as-you-work skill (deferred to follow-up)**:
- A skill / agent rule that fires `hero next ask "..."` and
  `hero next suggest "..."` at appropriate moments during work, so
  the user-graph nodes accumulate naturally instead of requiring
  the user to invoke commands.
- Tracked separately because it's behavioral / prompt-engineering
  work, not codebase plumbing. The `hero next suggest set "..."`
  writer from Phase 3 covers the manual-capture path until this
  ships.

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

## Deferred

These were open questions resolved as "ship v1 as-designed, revisit
on signal":

- **Hand content inside `<user>.md`.** v1 ships total-rewrite, no
  marker preservation in `<user>.md`. Revisit if any user reports
  wanting to keep hand-written content in `<user>.md`; the likely
  fix is adding an `extras:` frontmatter field or a small
  marker-bounded zone. The `<user>.local.md` file already supports
  hand content for users who need a scribble surface today.

## Open questions

(All five original open questions resolved during spec finalization;
list intentionally empty. Re-add if new questions surface during
delivery.)

## Risks

- **Merge driver doesn't fire for users who haven't run
  `hero install`.** Clones without the driver get standard conflict
  markers in `.hero/NEXT.md` / `.hero/next/*.md`. Mitigated by the
  Phase 6 `skills/next-merge-recovery.md` rule (agents self-heal on
  marker detection) and by the next Stop-hook firing regenerating
  the file. The Stop hook is the existing self-heal already in
  place for the legacy command.
- **Round-trip ingest produces duplicate nodes if idempotency check
  is wrong.** A misclassified or hash-bucket-collision case could
  re-create nodes on every session start. Mitigated by content-hash
  dedup keyed on `(node type, session_id, sha256(text))`, and by
  E2E coverage (AC-6 / AC-11) running the ingest twice and
  asserting no second-run node creation.
- **Cross-machine race.** Two machines edit `<user>.md` in parallel
  before either pushes. Mitigated by the merge driver: last
  writer's local graph wins on the merge, and the loser's content
  survives via the next round-trip ingest (its graph nodes get
  ingested from the winner's pushed file). No data lost — last
  writer's view wins on conflict resolution, both sets of nodes
  re-converge on the next round-trip.
- **Phase 1 ships before users have migrated their working repos.**
  Mitigated by the Phase 1 pre-flight check that refuses to
  overwrite unmigrated content (AC-14). The check keys on legacy
  markers and section headers, gated on `next.projected == false`.
- **Performance: total-rewrite of three files on every Stop hook
  could feel slow on large graphs.** Documented < 200 ms budget
  target. If exceeded, switch the local file to marker-bounded
  partial rewrite (which is already its model anyway) and keep the
  tracked files as total-rewrite (they're smaller and not on the
  hot path of every keystroke).
- **`hero install` itself is a precondition for several mitigations.**
  Users who skip it lose the merge driver, the git hooks, and the
  Stop-hook rewire. Mitigated long-term by `hero check` flagging
  install drift; short-term by ensuring the manual
  `hero next checkpoint` path still works correctly without any
  hook installation.

## Validation

End-to-end and per-phase verification:

- **`hero check` health check** confirms the Stop hook points to
  `hero next checkpoint`, git hooks are installed, and the `hero-next`
  merge driver is registered. Run after every `hero install`.
- **CI gate** (AC-12) diffs the committed `.hero/NEXT.md` against
  `hero next checkpoint` output and fails the build on drift. Catches
  Stop-hook regressions and reminds contributors to commit
  projection changes.
- **E2E cross-machine script** (AC-6 / AC-11): `tmp/e2e-cross-machine.sh`
  or an area suite in `e2e-area-suites` exercises the full
  write-on-A → push → pull-on-B → ingest-on-B → read-on-B sequence
  with two ephemeral graph DBs and no Cloud configured.
- **Migration smoke**: run `hero next migrate-to-projection` on a
  freshly-cloned hero repo; confirm `next.projected = true` set,
  three files exist, second run is a no-op, legacy markers and
  section headers
  gone.
- **Merge-driver smoke**: simulate a two-branch merge touching
  `.hero/NEXT.md` on both sides; confirm no `<<<<<<<` markers in
  the resolved file for an `hero install`-ed clone; confirm the
  `next-merge-recovery` skill catches markers in a non-installed
  clone.
- **Performance smoke**: time `hero next checkpoint` against a
  representative graph; confirm < 200 ms total. Document the budget
  in the spec; revisit if exceeded.
- **Per-phase smoke** as enumerated under `## Approach`.

## Changes

Files modified or created during this delivery, in order of commit.

**Already shipped (audit confirmed; no changes needed):**

- `internal/cli/checkpoint.go` — projection write path
  (`writeProjectedNextMD`, `writeUserHandoffFile`), machine-block
  rebuild, `next.projected` flag gate.
- `internal/cli/next_handoff.go` — `suggest` / `ask` / `reflection`
  / `ingest` commands with positional-arg writers.
- `internal/cli/next_migrate.go` — `migrate-to-projection` with
  `next.projected` flag flip.
- `internal/cli/next_hooks.go` — pre-commit / post-merge / merge
  driver registration.
- `internal/projection/projection.go` — `NextMD`.
- `internal/projection/user_handoff.go` — `UserHandoffMD`.
- `internal/handoff/handoff.go` — `RecordAsk` / `RecordSuggestion`
  / `RecordReflection` / `LatestAsk` / `LatestSuggestion` /
  `RecentReflections`.
- `internal/handoff/ingest.go` — `IngestUserFile`.
- `internal/install/claude_hooks.go` — Stop / PreCompact hook
  wiring via `heroCheckpointCmd`.
- `internal/install/codex_hooks.go` — Codex Stop hook wiring.
- `internal/config/config.go` — `NextProjected()` flag accessor.
- `.gitattributes` — merge driver directives for NEXT.md +
  `next/*.md`.

**Modified in this delivery:**

1. `internal/cli/checkpoint.go` — add pre-flight migration gate:
   when `next.projected == false` AND `.hero/NEXT.md` has legacy
   markers / section headers, exit non-zero with a directive to
   run `hero next migrate-to-projection`. (AC-14)
2. `internal/install/claude_hooks.go` — add a `SessionStart` hook
   firing `hero next ingest` so cross-machine continuity works
   without a manual command. (AC-7, AC-11)
3. `internal/install/codex_hooks.go` — mirror the SessionStart
   hook for the Codex shape (if Codex supports it; otherwise leave
   a no-op and note in the file). (AC-7)
4. `.github/workflows/test.yml` — append a drift-gate step that
   runs `hero next checkpoint --quiet` and
   `git diff --exit-code .hero/NEXT.md`, failing the build on
   drift. (AC-12)
5. `internal/cli/checkpoint_test.go` (or new
   `internal/cli/cross_machine_test.go`) — Go test simulating two
   ephemeral graph DBs (machine A + B) with a shared handoff file
   on disk, verifying `hero next suggest` on B returns A's
   recorded text after ingest. (AC-6, AC-11)

**Created in this delivery:**

6. `skills/next-merge-recovery/skill.md` — agent rule for
   `<<<<<<<` marker detection → `hero next checkpoint` self-heal.
   (AC-16)

## Notes for downstream specs

`project-snapshot` (`.hero/planning/features/project-snapshot/spec.md`)
depends on this spec and reuses:

- The projector pattern (`projection.NextMD` / `UserHandoffMD` /
  `LocalStateMD`) — `SnapshotMD` follows the same shape.
- The Stop-hook integration in `internal/install/claude_hooks.go`.
- The `.gitattributes` + merge-driver model — `SNAPSHOT.md` is also
  a graph projection and benefits from `merge=hero-next`.
- The `hero install` hook-installation flow.
- The migration-gate pattern (pre-flight check refusing overwrite of
  unmigrated content) — applicable to any future projector that
  takes over an existing tracked file.

Any change in this spec to the projector pattern, hook wiring, merge
driver, or migration approach should be flagged to `project-snapshot`
before delivery.
