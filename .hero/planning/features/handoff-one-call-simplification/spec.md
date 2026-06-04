---
title: Handoff Simplification — One Persist, One Load, Fewest Files
slug: handoff-one-call-simplification
type: feature
status: planning
priority: P1
domain: engineering
size: large
created: 2026-06-03
origin: session
tags: [next-md, projection, handoff, snapshot, queue, simplification, architecture, decision]
relations:
  - target: next-as-projection
    kind: revises-decision-of
  - target: snapshot-architecture
    kind: revises-decision-of
  - target: next-auto-emit-user-ask
    kind: parent-of
  - target: next-unconditional-commit-staging
    kind: parent-of
---

# Handoff Simplification — One Persist, One Load, Fewest Files

> Session-originated design spec. Backed by a four-angle deep-dive (end-of-turn write
> machinery, read/consumption audit, SNAPSHOT/QUEUE archaeology, target architecture).
> No `tracker_id`.

## The job, restated

Hero's entire handoff job is two sentences: **at the end of a turn, persist what a future
session needs; at the start of a turn (maybe on another machine), load it.** Plus git
plumbing so it travels. Everything else is accretion.

The maintainer's felt experience after living with it: "constant drift, end-of-turn 'NEXT.md
isn't part of my commit — ignoring it,' or it's stale — such a simple thing with so much
drama." This spec is the response: name what's load-bearing, name what's ceremony, and
collapse toward the simplest design with the fewest moving parts.

## The two things that actually hurt (and why)

The deep-dive found the felt pain has exactly two root causes, and **both are fixable now by
re-wiring code that already exists — no new infrastructure:**

1. **Drift / staleness = nothing auto-emits to the graph.** Projection was supposed to kill
   drift: regenerate NEXT from the graph every turn so it can't fall behind. But the graph
   only gets fresh handoff content (last ask, suggested next, reflections) when the agent
   *remembers* to run `hero next ask/suggest/reflection`. `writeCheckpoint` projects but never
   emits — it is a pure projector that performs zero graph mutations. So you get a perfectly
   re-rendered projection of a **stale graph**. → Fixed by **[next-auto-emit-user-ask](next-auto-emit-user-ask)**:
   fold auto-emit of `UserAsk` into the Stop-hook checkpoint, reusing the transcript-parsing
   machinery that already exists in `next_compact_handoff.go` (`resolveSessionContext` +
   `firstUserAskFromTranscript`). Pure re-wire.

2. **"NEXT.md isn't in my commit" = staging lives in only one of two hook installers.** There
   are two installers — the generic `hero hook` dispatcher (`internal/hooks/install.go`,
   post-commit runs `writeCheckpoint` but stages nothing) and `hero next install-hooks`
   (`next_hooks.go`, the only path that `git add`s the projected files). A repo set up the
   first way projects handoff files but never stages them. → Fixed by
   **[next-unconditional-commit-staging](next-unconditional-commit-staging)**: make staging an
   invariant of a single default install path, so "hooks installed" ⟹ "handoff travels."

**These two are Phase 1.** They address both felt pains and neither needs new capability —
only consolidation of code already in the tree. Everything below is de-bloating that makes
the system match the "fewest moving parts" mental model, but is not load-bearing for the pain
relief.

## How we ended up with SNAPSHOT and QUEUE (the archaeology)

The three project-level files are **not a designed-together trio.** They arrived in three
waves, each justified independently against "why not just extend NEXT.md":

- **NEXT.md** came first — the original cross-session handoff, the single-session thread
  ("just finished / next / blocked / tried-and-failed / context"). Re-architected from a
  hand-maintained doc into a graph projection in the v0.9.2→v0.10.0 window (`next-as-projection`).
- That **projector framework** is what made the next two cheap. Both landed 2026-05-18:
  - **QUEUE.md** (`kickoff-prompts-queue`) — the full ranked backlog of *every* ready spec with
    a paste-ready kickoff each. NEXT holds *one* path forward; QUEUE holds *all 99*. The
    `.hero/QUEUE.md` *file* (65 KB, committed, hook-regenerated) exists for one concrete reason:
    some harnesses (Claude Code) can't shell out at session start, so the file is a
    staleness-tolerant cache of the `hero queue` command.
  - **SNAPSHOT.md** (`project-snapshot`, decision `snapshot-architecture`) — the project-shape
    rollup along the **surface × lifecycle** axis (core/serve/mcp/docs/…, each with its stage,
    blockers, aged bugs). This is the one genuinely new axis nothing else carried.

Verdict: the split is **accretive in arrival, deliberate in gating** — each file passed a real
"NEXT can't absorb this" test, but no single decision ever blessed "three files" as a unit.

## What's actually useful vs ceremony (the consumption audit)

The honest finding: **only one file is load-bearing for code, and `/resume` already bypasses
all of them.**

| Artifact | Read back by code? | Verdict |
|---|---|---|
| `graph.db` + `events.log` | Yes — source of truth, feed/replay/tracking | **Core** |
| `.hero/next/<user>.md` | **Yes** — `hero next ingest` (SessionStart hook) re-hydrates the graph from it across machines | **The one load-bearing file** — cross-machine federation medium |
| `.hero/NEXT.md` | Only `session:` frontmatter + "Tried and failed" re-ingest; body read by nobody programmatically | Human/GitHub display; `/resume` ignores the body |
| `.hero/SNAPSHOT.md` | **No** — every consumer (CLI, `hero_snapshot` MCP) rebuilds from the graph; the file is never read back | Display-only; committed churn |
| `.hero/QUEUE.md` | **No** Go reader; `hero_queue`/graph reproduce it | Redundant cache for a harness limitation |
| `.hero/next/<user>.local.md` | Only by its own writer (timestamp) | Local-only; recomputable from git |

Two structural facts fall out:

- **There are two competing session-start philosophies.** `hero resume` (the now-primary,
  auto-fired path) builds its brief entirely from the **graph digest** and reads none of these
  file bodies. `/prime` (older, secondary) reads NEXT.md/QUEUE.md as files. So the rich prose
  in NEXT.md — the part that costs the most to keep fresh — is the part `/resume` doesn't read.
- **`graph.db` makes the committed files pure cache.** Every committed handoff file is
  reprojectable from the graph. They are committed so a cold/cross-machine checkout has warm
  markdown without the (gitignored) `graph.db`. That's a legitimate choice — but it means the
  whole 9-artifact fan-out is redundant by construction, so collapsing it is low-risk.

(One concrete inconsistency to fix along the way: the `hero_snapshot` MCP tool description
claims it "Reads `.hero/SNAPSHOT.md` when fresh" — the default code path rebuilds from the
graph and never reads the file. Stale description.)

## The moving-part count today

Roughly **18 moving parts** produce **9 file artifacts**: 5 hooks (Claude Stop, Claude
PreCompact, Codex Stop, git pre-commit, git post-commit) + the reverse SessionStart/ingest
pair + post-merge; 4 end-of-turn commands (checkpoint, index, queue write, git add); 5
projector functions inside `writeCheckpoint`; 3 handoff node types + AC-flip. But the headline
is encouraging: **`writeCheckpoint` is already ~80% of "the one persist method"** — a single
function projecting 6 artifacts from graph+git. The N-looking surface is mostly *trigger*
multiplicity (5 hooks all calling the same checkpoint), not 5 independent systems. The real
split is **projection vs. emission**: checkpoint projects but never emits.

## Target architecture (the minimal design)

**End-of-turn — one call.** `hero next checkpoint` (Stop hook, stdin-fed):
- read the Stop payload (`session_id`, `transcript_path`) → **auto-emit `UserAsk`** from the
  last user message (reuse `firstUserAskFromTranscript`);
- `NextSuggestion`: mechanical `PickUserSuggestion` floor (top open feature) + the agent's
  optional `hero next suggest` as the quality ceiling — never a blank, never a discipline tax;
- project the per-user file (the one tracked surface); idempotent, total-rewrite,
  semantic-change-gated (no timestamp churn).

**Start-of-turn — one call.** `hero resume` (SessionStart hook):
- ingest every `.hero/next/*.md` into the local graph (cross-machine federation);
- emit the briefing (last ask, suggested-next, reflections, **live** machine state computed on
  the fly, top open features). Absorbs today's separate `ingest` / `resume` / `prime` /
  compact-handoff-kickoff into one.

**Travel — one unconditional pre-commit hook,** folded into the single default installer:
`git add` the projected tracked files so "hooks installed" ⟹ "handoff travels."

**File set: 5 → 2 tracked + 0 persisted-local.**
- `.hero/next/<user>.md` — **KEEP** (the federation medium and primary surface).
- `.hero/NEXT.md` — **MERGE/DROP**: fold the project-state header into the per-user surface, or
  keep a thin generated pointer. Resolves the current two-writers race in team mode.
- `.hero/next/<user>.local.md` — **DROP the file**; recompute machine state live in `resume`.
  Removes the `.bak.<timestamp>` backup dance (~90 lines of safety for a disposable file).
- `.hero/SNAPSHOT.md` — **DROP as a committed handoff file**; keep the projector available
  on-demand (`hero snapshot` / `hero_snapshot` rebuild from the graph already). Its unique
  surface×lifecycle view stays a *command*, not committed churn.
- `.hero/QUEUE.md` — **DROP as a committed handoff file**; it's a query, not a handoff.
  `hero queue` / `hero_queue` already serve it from the graph. Emit a top-N summary inline in
  `resume` if wanted at cold start.

**Hook set: 3 (Stop → checkpoint, SessionStart → resume, pre-commit → stage), one installer.**
**Node set: 3 (`UserAsk`, `NextSuggestion`, `SessionReflection`) — unchanged; already minimal.**

## What's irreducible vs accreted

**Irreducible:** graph as source of truth (`graph.db` from `events.log`); one tracked per-user
file as the cross-machine medium; git staging for travel; idempotent/total-rewrite writes; the
three node types.

**Accreted (safe to remove/merge):** the 5-file split → 2 surfaces; `.local.md` as a file + its
backup dance; the migration gate / legacy-detection (`detectUnmigratedNextMD`,
`legacyNextHeaders`) — given the tiny same-tool user base, a clean break beats keeping the
bridge; the dual hook installers → one; manual emit as the *primary* freshness path → auto-emit.

## Honest hard parts

- **Auto-deriving a *good* suggested-next is not deterministic** — it needs model judgment.
  Keep the mechanical floor + optional agent ceiling. Trying to make the deterministic path
  "understand the turn" is the trap that re-bloats this system. Do not over-invest.
- **Stdin is harness-specific.** Claude provides the transcript; cursor/codex/generic don't.
  Auto-emit is a no-op without a payload (fall back to the existing graph state) — must never
  error or hang; preserve the bounded-read (64 KB/1000-line) discipline.
- **SNAPSHOT carries a real unique axis** (surface×lifecycle). "Drop as a handoff file" ≠ "delete
  the projector" — the view survives as a command. Confirm no consumer reads the *file* before
  removing it (the audit found none; the MCP tool rebuilds from the graph).
- **Team mode** is the one place "one project surface" meets real tension (shared roster vs
  personal file). For this user base, bias to the simple solo shape and let team mode *embed* the
  shared header in the per-user file rather than maintain a separate `NEXT.md`.

## Phasing

**Phase 1 — kills both felt pains, pure re-wire (do first):**
- [next-auto-emit-user-ask](next-auto-emit-user-ask) — auto-emit `UserAsk` at Stop. Kills drift.
- [next-unconditional-commit-staging](next-unconditional-commit-staging) — staging as an invariant
  of one installer. Kills "not in my commit."

**Phase 2 — fewer files:**
- Drop `.hero/QUEUE.md` and `.hero/SNAPSHOT.md` as committed handoff files (keep the commands);
  fix the stale `hero_snapshot` description.
- Drop `.hero/next/<user>.local.md` as a file; compute machine state live in `resume`; remove the
  backup dance.
- Collapse `NEXT.md` into the per-user surface (or a thin pointer); resolve the two-writers race.

**Phase 3 — collapse commands + clean break:**
- Merge `ingest` + `resume` + `prime` + compact-kickoff into one `hero resume`.
- Delete the migration gate and legacy-detection machinery (clean break; ping the two other
  users to re-run install once).

After Phase 1 the two felt problems are gone. Phases 2–3 are de-bloating to match the
fewest-moving-parts mental model.

## Acceptance Criteria (design-level — children carry the testable ACs)

- The end-of-turn path emits handoff content automatically; freshness no longer depends on the
  agent running `hero next ask` (owned by [next-auto-emit-user-ask](next-auto-emit-user-ask)).
- Handoff files travel with commits under a single default hook install path; no
  hooks-without-staging configuration is reachable (owned by
  [next-unconditional-commit-staging](next-unconditional-commit-staging)).
- The committed handoff surface is reduced to the federation-medium file (+ optional thin
  pointer); SNAPSHOT/QUEUE survive as on-demand commands, not committed churn (Phase 2).
- Start-of-turn is a single `hero resume` that ingests then emits (Phase 3).
- No regression to cross-machine continuity: project → commit → push → pull → ingest → re-project
  stays idempotent throughout.

## Boundaries

- This spec is the **umbrella**; the testable, delivery-ready work lives in the child specs.
  Phase 2/3 items should be split into their own specs when the user commits to them — do not
  bulk-delete SNAPSHOT/QUEUE files without their own spec confirming no consumer reads them.
- Not changing the graph schema or the three node types.
- Not building team-mode roster rendering here (separate deferred feature).

## Kickoff

You're picking up the umbrella simplification of Hero's handoff subsystem. Read this spec, then
the two Phase-1 children. The thesis: the whole subsystem is "persist at end of turn, load at
start of turn, travel via git," and it accreted into ~18 moving parts and 9 files where ~2 files
and one persist/one load call would do. The two things the maintainer actually feels — drift and
"not in my commit" — are **Phase 1** and are pure re-wires of existing code.

**Pick up at:** deliver [next-auto-emit-user-ask](next-auto-emit-user-ask) and
[next-unconditional-commit-staging](next-unconditional-commit-staging) — both are diagnosed,
delivery-ready, and independent. Auto-emit reuses `resolveSessionContext` /
`firstUserAskFromTranscript` from `internal/cli/next_compact_handoff.go`; staging consolidates
the two installers in `internal/hooks/install.go` + `internal/cli/next_hooks.go`. After Phase 1,
revisit Phase 2 (drop SNAPSHOT/QUEUE/local files) with fresh per-file specs.

→ `internal/cli/checkpoint.go`, `internal/cli/next_compact_handoff.go`, `internal/cli/next_hooks.go`, `internal/hooks/install.go`, `internal/projection/user_handoff.go`

## Recap

Hero's handoff subsystem does a simple job — persist at end of turn, load at start, travel via
git — but accreted into ~18 moving parts and 9 files (NEXT/per-user/local/SNAPSHOT/QUEUE/…),
most of them graph-derived caches no code reads back. The maintainer's two felt pains, drift and
"not in my commit," have one root cause each: nothing auto-emits to the graph (so projections
re-render stale data), and staging lives in only one of two hook installers (so files project but
don't travel). Both are Phase-1 re-wires of existing code. The target collapses to one end-of-turn
`checkpoint` (now also auto-emitting), one start-of-turn `resume` (ingest + emit), one
unconditional staging hook, and two tracked files — keeping SNAPSHOT/QUEUE as on-demand commands
rather than committed churn. SNAPSHOT/QUEUE arrived as independently-justified accretions on the
projection framework, not a designed trio; only the per-user federation file is load-bearing for
code. Clean break on legacy machinery given the tiny user base.
