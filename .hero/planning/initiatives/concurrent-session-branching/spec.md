---
title: "Concurrent-Session Branching & Worktree Isolation"
slug: concurrent-session-branching
type: initiative
status: planning
domain: engineering
size: x-large
priority: high
created: 2026-06-04
tags: [git, worktree, concurrency, claims, isolation, branching]
child:
  - csb-phase0-git-primitives-async-retrofit
  - csb-phase1-interactive-worktree-isolation
  - csb-phase2-claim-gated-branch-creation
  - csb-phase3-active-integration-target
  - csb-phase4-lifecycle-and-visibility
  - csb-phase5-midflight-target-shift
  - csb-phase6-stacked-branches
---

# Concurrent-Session Branching & Worktree Isolation

## Vision

Multiple Hero sessions — human-driven, agent-driven, and the async runner —
can work the same repository at the same time without ever destroying each
other's uncommitted work. Each unit of in-flight work lives on its own git
branch in its own worktree, anchored to a `hero claim`, resolving to one
shared `.hero/` graph. Branches are never anonymous: every branch has a
reason (a claim or a labeled `scratch/`), and the steady-state branch count
equals the work-in-flight count because merge tears the branch down. Hero
becomes safe to run N-at-a-time on one checkout.

## Goal

Eliminate the concurrent-session **clobbering** class of failure — one session's
`git checkout` destroying another session's uncommitted work on a shared working
tree — by isolating every active unit of work in its own git worktree, anchored
to a `hero claim`, while leaving **content conflicts** (two committed changes
that disagree) to surface naturally through git and `hero conflicts`. The
initiative delivers net-new git *mutation* on top of today's read-only
`gitutil`, generalizes the primitives the async runner already needs, and
guarantees that nothing Hero does ever silently rewrites, rebases, or deletes
work — orphaned and stale state is *surfaced*, never auto-removed.

## Kickoff

Make Hero safe to run in many concurrent sessions on one checkout: every
claimed spec gets its own git worktree + branch, all resolving to one shared
`.hero/`. Spec lives at
`.hero/planning/initiatives/concurrent-session-branching/spec.md`.

**Status:** planning — initiative spec landed, 7 child stubs sequenced, no code.

**Pick up at:** `/design csb-phase0-git-primitives-async-retrofit` — add write
ops to `internal/gitutil/gitutil.go` (currently READ-ONLY) and retrofit
`internal/async/runner.go` to run each `runDeliver` job in an isolated
worktree. That closes the live clobbering bug and proves the primitives.

→ `.hero/planning/initiatives/concurrent-session-branching/spec.md`

**Files:** `internal/gitutil/gitutil.go`, `internal/async/runner.go:142`, `internal/async/jobs.go:38`, `internal/workspace/locate.go:85`, `internal/cli/claim.go:72`
**Skip:** reusing graph-conflict-detection for content conflicts (wrong layer — use `hero conflicts`); `/release` owning the integration target (net-new state, prefer a `hero target set` verb); auto-managing per-worktree build state in v1.

## Problem

Hero is increasingly run by multiple concurrent actors against a single
checkout: a human in their editor, one or more agent sessions, and the async
delivery runner. They all share one working tree. The moment two of them touch
git branch state — `git checkout -b`, `git checkout <base>`, commit — one can
silently destroy the other's uncommitted work. This is happening **today**:
`internal/async/runner.go` `runDeliver` does `git checkout -b` / commit / push /
`git checkout <base>` directly against the shared `projectDir` (the runner's
project directory). If a human has uncommitted edits when the runner flips
branches underneath them, those edits are clobbered.

**Key distinction — this initiative is built on it:**

- **Clobbering** is when one session destroys another session's *uncommitted*
  work by mutating the shared working tree out from under it. Clobbering is a
  defect of *shared mutable state*. It is fully **PREVENTABLE via isolation** —
  give each unit of work its own worktree and no session ever touches another's
  tree.
- **Content conflicts** are when two *committed* changes disagree (the same
  lines edited two ways). Conflicts are **INHERENT to parallel work** — they are
  not a bug, they are the cost of two people changing the same thing. The right
  response is to **surface them early** (via `hero conflicts` on overlapping
  in-flight specs, and via git's own merge machinery) and **never try to
  prevent** them by serializing or auto-merging.

The whole initiative attacks clobbering with isolation and leaves conflicts to
git. Conflating the two is the most common way designs in this space go wrong.

## Guiding principles

1. **Every branch has a reason.** A branch is either claim-anchored (created
   because a spec was claimed) or labeled scratch (`scratch/<identity>/<label>`
   for explicit ad-hoc work). There are **no anonymous branches**. If you can't
   say why a branch exists, Hero shouldn't have created it.
2. **Merge tears down.** When work merges, its branch and worktree are torn
   down. Steady-state **branch count == work-in-flight count**. A pile of stale
   branches is a smell; the system should drive toward zero idle branches.
3. **Nothing auto-deletes or silently rewrites work.** Orphaned worktrees,
   stale locks, and abandoned branches are **SURFACED** (in `hero check`, in the
   branch inventory, in reconcile) — never silently removed. In-flight work is
   **never silently rebased**. The one place the design intentionally rewrites
   history (Phase 6 stacked rebase-cascade) is strictly opt-in and announced.

## Grounding facts

These are verified against the current codebase. Child specs must build on
these realities, not re-investigate from scratch.

- **`internal/gitutil/gitutil.go` is READ-ONLY today.** Its exported surface is
  all inspection: `IsRepo`, `DefaultBranch`, `CurrentBranch`,
  `FilesChangedOnBranch`, `FilesChangedUncommitted`, `AllChangedFiles`,
  `NormalizeFilePath`, `RepoKey`, `UserName`. There is **no** branch/worktree
  *mutation* anywhere in this package. The entire initiative is net-new git
  mutation layered on top.
- **`internal/async/runner.go` `runDeliver` is a LIVE instance of the clobbering
  bug.** It runs `git checkout` / commit / push / `git checkout <base>` against
  the shared `projectDir` (`runner.go:142`, `:149`, `:226`, `:310`). It already
  has the two primitives to generalize: `async.BranchName(slug) ->
  hero/deliver/<safe-slug>` (`runner.go:321`) and `job.BaseBranch`
  (`internal/async/jobs.go:38`, on the `Job` struct). There is **zero** worktree
  usage in `internal/async/` today — confirmed.
- **A claim is pure metadata across THREE stores that must stay consistent.**
  `runClaimNew` (`internal/cli/claim.go:72`) writes: (1) spec frontmatter
  `claimed_by` via `tracking.UpdateSpecFrontmatter`, (2) the FTS5 index via
  `idx.Claim` (`internal/index`), and (3) `events.log` via
  `tracking.AppendEvent(..., tracking.ClaimEvent{...})`. Adding a worktree
  identity to a claim makes it a **FOURTH coordinated state**. This is a
  consistency hazard (R1) — reconciliation across the four stores must route
  through `internal/reconcile` (`reconcile.go`), the existing home for
  cross-store reconciliation.
- **`workspace.Locate` + `WithStopAt` (`internal/workspace/locate.go:85`,
  `:66`) governs `.hero/` discovery by upward directory walk.** A git worktree
  lives in a **sibling directory**, so a naive upward walk from inside a worktree
  will NOT find the main checkout's `.hero/`. Every worktree **must** resolve to
  the **SAME canonical `.hero/`** (one shared `graph.db` / `index.db` /
  `events.log`), or claims/index/events fork per-worktree and coordination
  collapses. There are **6** `workspace.Locate` callers to audit:
  `internal/cli/root.go`, `context.go`, `status.go`, `note.go`, `install.go`,
  `install_satellites.go`. This is the hidden hard core of the isolation phase
  (R2).
- **`/release` is ADVISORY-ONLY today.** The command (`.claude/commands/release.md`)
  routes to `release-engineer` / `devops-engineer` and runs `hero docs check`.
  It owns **no state**. Making it set the integration target would be **net-new
  stateful behavior**, not an extension of what it does. A dedicated `hero target
  set` CLI verb is the cleaner owner — recommended.
- **Config override is HAND-WRITTEN field-by-field.** `config.Load`
  (`internal/config/config.go:1251`) -> `LoadLocal` (`:1367`) -> `MergeLocal`
  (`:1408`) merges `hero.local.json` field by field, by hand. There are **THREE
  active in-flight bugs** on exactly this path:
  `hero-local-merge-missing-dialect-fields`, `spec-types-cache-frontmatter-empty`,
  and `vocabulary-resolve-misses-methodology-derivation` (all present in
  `.hero/planning/bugs/`, all `delivering`). Adding an integration-target config
  block means extending `MergeLocal` — **HARD SCHEDULING CONSTRAINT (R3): those
  three bugs must close before Phase 3 lands.**
- **Content-conflict surfacing must reuse `hero conflicts` /
  `idx.FindConflicts`.** `runConflicts` (`internal/cli/conflicts.go:26`) calls
  `idx.FindConflicts(slug)` (`internal/index/index.go:963`), which does
  **file-overlap detection on in-flight specs** — exactly the content-conflict
  surface we want. Do **NOT** use graph-conflict-detection
  (`graph-conflict-detection` spec / `internal/triage/conflict.go`
  `FindConflicts`): that detects **graph-metadata divergence** and *explicitly
  defers content merge to git*. Wiring the wrong module here is a known trap —
  stated so nobody does it.

## Specs

Seven sequenced child specs. Each stub below is actionable as input to a future
`/design` pass. Sizes use the shared 6-tier ladder; the initiative as a whole is
`x-large` and one child (Phase 6) is itself `x-large`.

---

### Phase 0 — Git mutation primitives + async-runner retrofit
**Slug:** `csb-phase0-git-primitives-async-retrofit` · **Size:** medium ·
**Deps:** none (foundation) · **SHIPS FIRST**

One-line: add the net-new git write primitives to `gitutil`, a trunk-lock
primitive, and retrofit the async runner to deliver each job in an isolated
worktree — closing the live clobbering bug and proving the primitives with a
real consumer.

**Scope — in:**
- New write ops in `internal/gitutil/gitutil.go`: `AddWorktree`,
  `RemoveWorktree`, `ListWorktrees` (parse `git worktree list --porcelain`),
  `CreateBranch`, `BranchExists`.
- A trunk-lock primitive: advisory `.hero/trunk.lock` carrying holder identity +
  timestamp, with staleness detection and crash recovery (a dead holder's lock
  is reported stale and reclaimable, never silently stolen mid-operation).
- Retrofit `internal/async/runner.go` `runDeliver` to create an isolated
  worktree per job, run checkout/commit/push inside it, and tear it down on
  completion — instead of mutating the shared `projectDir`.

**Scope — out:** interactive-session isolation (Phase 1); claim coupling
(Phase 2); anything user-facing beyond the runner.

**Modules/files:** `internal/gitutil/gitutil.go`, `internal/gitutil/gitutil_test.go`,
`internal/async/runner.go` (lines 142/149/226/310/321), `internal/async/jobs.go`
(`Job.BaseBranch`), new lock helper (likely `internal/gitutil` or a small
`internal/worktree` package — design decision for the `/design` pass).

**Dependencies:** hard: none. This is the foundation everything else consumes.

**Acceptance criteria:**
- THE SYSTEM SHALL expose `AddWorktree`, `RemoveWorktree`, `ListWorktrees`,
  `CreateBranch`, and `BranchExists` from `internal/gitutil` with explicit
  error returns.
- WHEN `ListWorktrees` runs THE SYSTEM SHALL parse `git worktree list
  --porcelain` and return one record per worktree including its path and branch.
- WHEN the async runner delivers a job THE SYSTEM SHALL perform all branch and
  commit operations inside a job-specific worktree and SHALL NOT run `git
  checkout` against the shared project directory.
- WHEN a delivery job completes THE SYSTEM SHALL tear down its worktree.
- IF a trunk-lock holder is no longer alive THEN THE SYSTEM SHALL report the
  lock as stale and reclaimable rather than blocking indefinitely or silently
  stealing it.
- WHILE a worktree-isolated job runs THE SYSTEM SHALL leave any uncommitted
  changes in the shared working tree untouched.

---

### Phase 1 — Interactive-session worktree isolation + workspace resolution + trunk lock
**Slug:** `csb-phase1-interactive-worktree-isolation` · **Size:** large ·
**Deps:** Phase 0 (hard)

One-line: give interactive (human/agent) sessions worktree-per-claimed-spec
isolation, and make every worktree resolve to one canonical `.hero/` — the
hidden hard core of the whole initiative.

**Scope — in:**
- Worktree-per-claimed-spec for interactive sessions, consuming the Phase 0
  primitives.
- **Workspace resolution across worktrees:** every worktree (a sibling
  directory) must resolve to the **one** canonical `.hero/` with the shared
  `graph.db` / `index.db` / `events.log`. Touches `internal/workspace/locate.go`
  (`Locate`, `WithStopAt`), `internal/cli/root.go`, and audits the 6 `Locate`
  callers: `root.go`, `context.go`, `status.go`, `note.go`, `install.go`,
  `install_satellites.go`.
- Trunk-lock integration for interactive sessions (acquire/report on shared-tree
  operations).

**Scope — out:** claim-gated *creation* (Phase 2 — Phase 1 isolates a session
that already has a claim/worktree; Phase 2 mints them); branch naming strategies
(Phase 2). Per-worktree build state is explicitly **surface-only** here.

**Per-worktree build state (node_modules / .env / local DB):** v1 **SURFACES
missing state and offers a documented opt-in init hook only**. Do NOT
auto-manage build state in v1 — a worktree with no `node_modules` is reported,
not silently populated.

**Modules/files:** `internal/workspace/locate.go`, `internal/cli/root.go`,
`internal/cli/{context,status,note,install,install_satellites}.go`,
`internal/gitutil` (Phase 0 primitives), docs for the opt-in init hook.

**Dependencies:** hard: Phase 0.

**Acceptance criteria:**
- WHEN any Hero command runs inside a git worktree THE SYSTEM SHALL resolve to
  the same canonical `.hero/` directory used by the main checkout.
- THE SYSTEM SHALL share a single `graph.db`, `index.db`, and `events.log`
  across all worktrees of the same repository.
- IF a worktree's upward walk would resolve to a different or missing `.hero/`
  THEN THE SYSTEM SHALL redirect to the canonical workspace rather than create
  or use a per-worktree one.
- WHILE an interactive session works in its own worktree THE SYSTEM SHALL leave
  other sessions' worktrees and the shared tree's uncommitted work untouched.
- WHERE a worktree is missing build state (node_modules / .env / local DB) THE
  SYSTEM SHALL surface the gap and SHALL NOT auto-populate it; an opt-in init
  hook MAY populate it when explicitly invoked.

---

### Phase 2 — Claim-gated branch creation + naming + attach + scratch + easy strategies
**Slug:** `csb-phase2-claim-gated-branch-creation` · **Size:** medium ·
**Deps:** Phase 0 (hard), Phase 1 (hard)

One-line: branches are minted only through a claim (or explicit scratch), via a
single claim→worktree path, with deterministic naming and attach-on-re-claim.

**Scope — in:**
- Gate branch+worktree creation on `hero claim` by extending `runClaimNew`
  (`internal/cli/claim.go:72`) — the **single** claim→worktree path the whole
  initiative routes through.
- Generalize `async.BranchName` into a shared deterministic namer used by both
  the runner and interactive claims.
- **Attach-on-second-claim:** the claim carries its worktree path. Re-claiming
  an already-claimed-by-me spec re-attaches the existing worktree rather than
  minting a new one. Extends `ClaimEvent`, spec frontmatter, and the index —
  **this is where the fourth coordinated state lands (R1)**; watch tri-store →
  quad-store consistency.
- **Scratch branches** `scratch/<identity>/<label>` for explicit ad-hoc work
  with no spec — labeled, never anonymous.
- **Fold in spec-scoped and trunk-based strategies as config knobs**, not as
  separate phases. They are naming/base-branch configuration
  (`hero/deliver/<slug>` per-spec vs. a shared trunk base), expressed as config,
  not new machinery.

**Scope — out:** active integration target config block (Phase 3); lifecycle /
prune / inventory (Phase 4); stacked bases (Phase 6).

**Modules/files:** `internal/cli/claim.go` (`runClaimNew`), `internal/async`
(`BranchName` generalization), `internal/tracking` (`ClaimEvent` extension),
`internal/index` (`Claim` extension to carry worktree path),
`internal/config` (read-only: strategy knobs — note Phase 3 owns the *new* block).

**Dependencies:** hard: Phase 0, Phase 1.

**Acceptance criteria:**
- WHEN a user runs `hero claim <slug>` THE SYSTEM SHALL mint a branch and
  worktree for that spec through a single claim→worktree code path.
- WHEN a spec already claimed by the same identity is claimed again THE SYSTEM
  SHALL re-attach the existing worktree rather than create a second one.
- THE SYSTEM SHALL record the worktree path consistently across all four claim
  stores (frontmatter, index, events.log, and the worktree itself) or fail the
  claim atomically rather than leave them divergent.
- WHEN a user creates a scratch branch THE SYSTEM SHALL name it
  `scratch/<identity>/<label>` and SHALL refuse to create an unlabeled
  (anonymous) branch.
- WHERE a branch-naming or base strategy is configured (spec-scoped vs.
  trunk-based) THE SYSTEM SHALL apply it as configuration without requiring new
  branching machinery.

---

### Phase 3 — Active integration target (config + pin-at-claim + owner)
**Slug:** `csb-phase3-active-integration-target` · **Size:** medium ·
**Deps:** Phase 2 (hard) · **SCHEDULING CONSTRAINT (R3): the three MergeLocal
bugs must close first.**

One-line: introduce a configurable integration target, pinned into claim
metadata at claim time and never re-read live from HEAD.

**Scope — in:**
- A new config block for the integration target; extend `MergeLocal`
  (`internal/config/config.go:1408`) to merge it — **only after**
  `hero-local-merge-missing-dialect-fields`, `spec-types-cache-frontmatter-empty`,
  and `vocabulary-resolve-misses-methodology-derivation` are closed.
- **Pin-at-claim:** at claim time, snapshot the integration target with
  precedence **explicit-config → current branch → main**, written into claim
  metadata. The pinned value is **NEVER re-read live from HEAD** afterward.
- **Decide the owner.** Recommend a dedicated **`hero target set`** CLI verb that
  owns this state, rather than making `/release` stateful (it is advisory-only
  today; see grounding facts).

**Scope — out:** mid-flight target *shift* detection and `hero retarget`
(Phase 5); lifecycle/inventory (Phase 4).

**Modules/files:** `internal/config/config.go` (`MergeLocal`, config struct),
`internal/cli/claim.go` (write pinned target into claim metadata), new
`internal/cli/target.go` (`hero target set`), `internal/tracking` (claim
metadata field).

**Dependencies:** hard: Phase 2. Scheduling: the three MergeLocal bugs.

**Acceptance criteria:**
- THE SYSTEM SHALL resolve the integration target by precedence: explicit config,
  then the current branch, then `main`.
- WHEN a spec is claimed THE SYSTEM SHALL snapshot the resolved integration
  target into the claim's metadata.
- THE SYSTEM SHALL use the pinned target for the life of the claim and SHALL NOT
  re-read the live HEAD to recompute it.
- WHEN a user runs `hero target set <branch>` THE SYSTEM SHALL persist the
  explicit integration target as configuration.
- IF the three MergeLocal bugs are not yet closed THEN THE SYSTEM SHALL NOT land
  the new config-block merge logic (scheduling gate).

---

### Phase 4 — Lifecycle + visibility
**Slug:** `csb-phase4-lifecycle-and-visibility` · **Size:** large ·
**Deps:** Phase 1 (hard), Phase 2 (hard) · *(brought BEFORE stacked branches)*

One-line: a spec-aware branch inventory, one-step resume, safe surfacing-only
prune, worktree-aware content-conflict surfacing, and `hero check` coverage for
the new failure modes.

**Scope — in:**
- **Spec-aware projected branch inventory:** join claims × worktrees × `gh pr`
  state — **NOT** a raw `git branch` dump. Shows why each branch exists and where
  it stands.
- **One-step resume:** re-attach a worktree for a claimed spec in a single step.
- **Safe prune (surfacing-first, never auto-delete):** merged → teardown;
  orphaned → route to reconcile; stale → surface; `scratch/*` → age-out nag.
  **NEVER auto-deletes work.** Model on `internal/reconcile` (`reconcile.go`) +
  `tracking.StaleClaims` (`internal/tracking/tracking.go:94`).
- **Worktree-aware content-conflict surfacing:** extend `hero conflicts` /
  `idx.FindConflicts` (`internal/cli/conflicts.go:26`,
  `internal/index/index.go:963`) to be worktree-aware. Reuse the **file-overlap**
  surface — do NOT use graph-conflict-detection.
- **`hero check` coverage** for new failure modes: orphan worktree, stale trunk
  lock, claim-without-worktree, worktree-without-claim.

**Scope — out:** mid-flight target shift (Phase 5); stacked bases (Phase 6).

**Modules/files:** `internal/cli/conflicts.go`, `internal/index/index.go`
(`FindConflicts`), `internal/reconcile/reconcile.go`,
`internal/tracking/tracking.go` (`StaleClaims`), `internal/cli` (new inventory +
resume + prune commands, `check` extensions), `gh` integration for PR state.

**Dependencies:** hard: Phase 1, Phase 2.

**Acceptance criteria:**
- WHEN a user lists the branch inventory THE SYSTEM SHALL show a spec-aware view
  joining claims, worktrees, and PR state — not a raw git branch list.
- WHEN a user resumes a claimed spec THE SYSTEM SHALL re-attach its worktree in a
  single step.
- WHEN prune runs THE SYSTEM SHALL tear down merged branches, route orphaned
  worktrees to reconcile, surface stale ones, and nag aging scratch branches —
  and SHALL NOT delete any branch with un-merged work.
- WHEN two in-flight specs touch overlapping files across worktrees THE SYSTEM
  SHALL surface the content conflict via `hero conflicts`.
- IF an orphan worktree, stale trunk lock, claim-without-worktree, or
  worktree-without-claim exists THEN `hero check` SHALL report it.

---

### Phase 5 — Mid-flight target shift + re-target command
**Slug:** `csb-phase5-midflight-target-shift` · **Size:** medium ·
**Deps:** Phase 3 (hard), Phase 4 (hard)

One-line: detect when the integration target moves while branches are in flight,
warn and leave them on the pinned target, and offer an explicit `hero retarget`.

**Scope — in:**
- Detect that the integration target has moved while branches reference the old
  pinned target.
- **WARN and leave** the branch on its pinned target — never silently rebase.
- Explicit, opt-in `hero retarget` for the user to move a branch to the new
  target deliberately.

**Scope — out:** stacked bases / rebase-cascade (Phase 6 — different mechanism).

**Modules/files:** `internal/cli/target.go` (extend), new `internal/cli` retarget
command, `internal/cli/check.go` (surface the drift), claim metadata (pinned vs.
current target comparison).

**Dependencies:** hard: Phase 3 (pin-at-claim), Phase 4 (inventory/visibility to
display the drift).

**Acceptance criteria:**
- WHEN the integration target moves while a branch is in flight THE SYSTEM SHALL
  warn that the branch's pinned target differs from the current target.
- THE SYSTEM SHALL leave the in-flight branch on its pinned target and SHALL NOT
  silently rebase it.
- WHEN a user runs `hero retarget <slug>` THE SYSTEM SHALL move the branch to the
  new target as an explicit, opt-in action.
- WHILE a target drift is unresolved THE SYSTEM SHALL keep surfacing it in the
  inventory and `hero check`.

---

### Phase 6 — Stacked branches
**Slug:** `csb-phase6-stacked-branches` · **Size:** x-large · **Deps:** Phase 2
(hard), Phase 3 (hard), Phase 4 (soft — to display the stack) · **LAST**

One-line: derive branch bases from the `hero blocked` dependency graph and
cascade rebases when a parent merges — the one place the design intentionally
rewrites in-flight work, strictly opt-in and announced.

**Scope — in:**
- **Base-from-dependency-graph:** a spec that `depends-on` another bases its
  branch on the dependency's branch, derived from the `hero_blocked` graph.
- **Rebase-cascade-on-parent-merge:** when a parent branch merges, cascade-rebase
  the dependent stack onto the new base.
- **Guardrails for "never silently rebase":** the cascade is **strictly opt-in
  and announced** — the user is told exactly what will be rewritten before it
  happens. This is the deliberate, explicit exception to principle 3.

**Scope — out:** the trivial naming strategies (those are Phase 2 config — do NOT
ship the rebase-cascade alongside them); mid-flight target shift (Phase 5,
different mechanism).

**Modules/files:** `internal/index` / graph (`hero_blocked` / `depends-on`
traversal), `internal/gitutil` (rebase op — net-new, guarded),
`internal/cli` (stacked-branch + cascade commands), Phase 4 inventory (display
the stack).

**Dependencies:** hard: Phase 2 (claim→worktree path), Phase 3 (pinned target);
soft: Phase 4 (stack display).

**Acceptance criteria:**
- WHERE stacked branches are enabled THE SYSTEM SHALL base a dependent spec's
  branch on its dependency's branch, derived from the `hero blocked` graph.
- WHEN a parent branch merges THE SYSTEM SHALL offer to cascade-rebase the
  dependent stack onto the new base.
- THE SYSTEM SHALL rebase in-flight work ONLY when the user has explicitly opted
  in, and SHALL announce exactly which branches will be rewritten before doing
  so.
- IF a cascade rebase encounters a content conflict THEN THE SYSTEM SHALL halt
  and surface it rather than auto-resolve.

## Dependencies

```
Phase 0 (foundation, ships first)
  └─> Phase 1 (interactive isolation + workspace resolution)
        └─> Phase 2 (claim-gated creation + naming + scratch + strategies)
              ├─> Phase 3 (integration target)   [gate: MergeLocal bugs closed]
              │     └─> Phase 5 (mid-flight shift + retarget)  [also needs Phase 4]
              ├─> Phase 4 (lifecycle + visibility)   [needs Phase 1 + Phase 2]
              │     └─> Phase 5
              └─> Phase 6 (stacked branches)  [needs Phase 2 + Phase 3, soft Phase 4]
```

Hard scheduling gate: **Phase 3 must not land until
`hero-local-merge-missing-dialect-fields`, `spec-types-cache-frontmatter-empty`,
and `vocabulary-resolve-misses-methodology-derivation` are closed** (they all
touch `MergeLocal`, which Phase 3 extends).

## Cross-cutting concerns & shared risks

**R1 — Multi-state claim consistency.** A claim already spans three stores
(frontmatter `claimed_by`, the FTS5 index, `events.log`). Phase 2 adds worktree
identity as a **fourth** coordinated state. Divergence between the four (e.g. a
frontmatter claim with no worktree, or a worktree with a stale index entry) is
the core integrity risk. **Mitigation:** route all reconciliation through
`internal/reconcile`; claim mutation should be atomic across the four stores or
fail cleanly. `hero check` (Phase 4) must detect each divergence shape.

**R2 — Workspace identity under worktrees.** A worktree is a sibling directory;
`workspace.Locate`'s upward walk will not naturally find the main `.hero/`. If
this is wrong, every per-worktree session forks its own graph/index/events and
coordination collapses silently. **Mitigation:** Phase 1 makes canonical
resolution the explicit hard core; all 6 `Locate` callers are audited; tests
prove a command run inside a worktree hits the one shared `.hero/`.

**R3 — Landing config changes during active MergeLocal bugs.** `MergeLocal` is
hand-written field-by-field and currently has three open bugs. Extending it for
the integration-target block while those are in flight risks merge regressions
and rework. **Mitigation:** hard scheduling gate — Phase 3 lands only after all
three close.

**R4 — Stacked rebase vs. "never silently rebase."** Phase 6's cascade rebase is
the one place Hero rewrites in-flight history. It directly tensions principle 3.
**Mitigation:** strictly opt-in, fully announced (name every branch to be
rewritten before acting), halt-and-surface on conflict, and never shipped
alongside the trivial Phase 2 strategies.

**The claim-release / handoff / peering lifecycle RULE.** State it verbatim and
honor it everywhere:

> A worktree is bound to a LOCAL claim. Any event that **releases** the claim
> (release / complete / cross-repo `hero handoff`) surfaces the worktree as
> **orphaned-pending-teardown** — surfaced, never auto-deleted, because the
> handoff may bounce back via `hero handoff accept`. Any event that **creates** a
> claim (`hero claim` / `hero handoff accept`) **mints-or-attaches** a worktree
> via the SINGLE Phase 2 claim→worktree path.

Corollaries:
- **Cross-repo handoff** (`internal/peering/handoff.go`) crosses **no shared git
  state** — isolation is naturally per-repo; the peer mints its own branch under
  its own `.hero/`. Do not try to share worktrees across repos.
- **NEXT.md projection** (`internal/nextdoc`, `internal/handoff`) is
  **SESSION-context handoff, NOT git-branch handoff**. It must learn exactly
  **ONE new fact** — which worktree/branch a session was on — so a resuming
  session re-attaches the right worktree. Do **NOT** entangle the two "handoff"
  verbs; they share a word and nothing else.

## Infeasible / fights-the-architecture — explicit warnings

- **Do NOT reuse graph-conflict-detection for content conflicts.** Wrong layer.
  Graph-conflict-detection (`internal/triage/conflict.go`) detects graph-metadata
  divergence and explicitly defers content merge to git. Content conflicts use
  `hero conflicts` / `idx.FindConflicts` (file-overlap).
- **`/release owns the target` is net-new stateful behavior, not an extension.**
  `/release` is advisory-only today. Size it as net-new, or (recommended) use a
  dedicated `hero target set` verb.
- **Do NOT auto-manage per-worktree build state in v1.** Provide a documented
  opt-in init hook and surface missing state — nothing more.
- **Stacked rebase-cascade must be strictly opt-in and announced.** Never ship it
  alongside the trivial Phase 2 strategies; never rebase in-flight work silently.

## Recommended delivery order

Deliver **Phase 0 first** — async-runner-first. Rationale: the async runner is a
**live, reproducible instance of the clobbering bug** and a real consumer of
exactly the primitives the whole initiative needs (`AddWorktree`/`RemoveWorktree`/
`ListWorktrees`, `CreateBranch`, `BranchExists`, the trunk lock). Forcing the
primitives into existence behind a real consumer — with the runner's existing
test surface to prove them — means they ship hardened and exercised, not
speculative. It also delivers immediate value (closes the live bug) before any
user-facing surface area exists.

Then proceed strictly down the dependency graph: Phase 1 (the hard workspace-
resolution core) → Phase 2 (the single claim→worktree path everything routes
through) → Phase 4 (lifecycle/visibility, brought before stacked branches so the
system is observable and prunable before it gets more complex) and Phase 3
(integration target, gated on the MergeLocal bugs) in parallel where the gate
allows → Phase 5 (mid-flight shift, needs both 3 and 4) → **Phase 6 last**
(stacked branches — the largest, riskiest, and the only intentional history
rewrite).

## Progress

- 2026-06-04 — Initiative spec authored from product-ideator grounding analysis.
  Seven child stubs sequenced; no code yet. Grounding facts verified against the
  current codebase (gitutil read-only surface, async runner clobbering site,
  tri-store claim, `workspace.Locate` callers, `MergeLocal` + its three open
  bugs, `hero conflicts` vs. graph-conflict-detection, advisory `/release`).
  Next: `/design csb-phase0-git-primitives-async-retrofit`.
