---
title: "Handoff-file staging is opt-in and lives in only one of two hook installers"
slug: next-unconditional-commit-staging
type: bug
status: completed
severity: high
priority: high
size: small
domain: engineering
created: 2026-06-03
origin: session
root_cause_class: design
relates-to:
  - next-as-projection
  - handoff-one-call-simplification
  - next-auto-emit-user-ask
  - pre-commit-auto-stage-next
completed_at: 2026-06-04T13:05:07Z
---

# Handoff-file staging is opt-in and lives in only one of two hook installers

## Summary

### Categorization
| Attribute | Assessment |
|-----------|------------|
| **Criticality** | high — recurring felt pain ("end of turn — NEXT.md isn't part of my commit, ignored"). The mission-critical promise ("next session starts as smart as the last one ended") silently fails: handoff state strands on the local machine and never reaches teammates or other machines. |
| **Ease of Fix** | moderate — the staging logic already exists; the work is making it unconditional (single install path) and extending the `hero check` detector + staged-file list. Mostly consolidation, not new mechanism. |
| **Caused by our codebase?** | Yes — two independent hook installers, only one of which stages, plus an opt-in install path. |
| **Needs more research?** | No — root cause confirmed against source at file:line for both installers and the staging gap. |

### Background
Hero projects handoff files (`.hero/NEXT.md`, `.hero/next/<user>.md`, `.hero/SNAPSHOT.md`, `.hero/QUEUE.md`) from the graph so a fresh session starts informed. For those files to reach another machine or teammate they must be **staged into the commit**. Hero has **two separate git-hook installers** that write to the same `.git/hooks/` files but with different managed-block markers and different behavior. Only ONE of them stages the projected files. Which installer ran depends on how the repo was set up, so a repo can end up **projecting handoff files on every commit but never staging them** — exactly the user's recurring complaint.

### Analysis
- The staging `git add` lives **only** in the `next`-hooks installer's `pre-commit` script (`internal/cli/next_hooks.go:268-272`), reachable via `hero next install-hooks`, `hero init` (default), and `hero upgrade` refresh.
- The **generic** hook installer (`internal/hooks/install.go`, command `hero hooks install`) wires `pre-commit` and `post-commit` to call `hero hook <event>`. The generic `post-commit` handler **projects** NEXT.md (`writeCheckpoint()`, `internal/cli/hook.go:104`) but issues **no `git add`**. The generic `pre-commit` handler (`internal/cli/hook.go:111-128`) only runs the opt-in status-truth gate — no checkpoint, no staging.
- Result: a repo set up via `hero hooks install` (not `hero next install-hooks`/`hero init`) regenerates handoff files on every commit and never stages them. The drift is invisible until the user notices NEXT.md isn't in the commit.
- Even on the "right" path, staging is **opt-in**: `hero next install-hooks` must have been run (or `hero init` with hooks not skipped). A repo that opted out, predates the feature, or used the generic installer has no staging.

### Root Cause
**Design split (root_cause_class: design):** handoff-file staging is implemented as a feature of *one* of two installers rather than as an invariant of "hooks are installed." There is no single default install path that guarantees `hooks installed ⇒ handoff files travel`. The existence of the standing manual-staging backstop rule in `CLAUDE.md`/`AGENTS.md` (`internal/install/agents_md.go:452`) is itself evidence the hook is not reliably installed — we ask the agent to hand-stage as a fallback.

### Source
- `internal/cli/next_hooks.go:257-283` — `hookScript("pre-commit")`; the ONLY `git add` of projected files (line 272).
- `internal/hooks/install.go:18-116` — generic installer; wires `pre-commit`/`post-commit` to `hero hook`, never stages.
- `internal/cli/hook.go:94-128` — generic dispatcher; `post-commit` projects but doesn't stage; `pre-commit` is gate-only.
- `internal/cli/check.go:251-283` — pre-commit-hook health check; keyed to the next-hooks marker only.
- `internal/install/agents_md.go:452` / `CLAUDE.md:116` — the manual-staging backstop rule.

### Fix Direction
Make handoff-file staging an **unconditional** part of a **single** default hook install path, so "hooks installed" always implies "handoff files travel." Extend `hero check` to flag a repo where projected files are tracked but no staging-capable hook is wired, and align the staged-file list with the actually-projected tracked files (add `.hero/SNAPSHOT.md`; keep `.hero/QUEUE.md` best-effort; never stage gitignored files).

---

## Problem Statement

The user's #2 felt pain: at end of turn, NEXT.md is regenerated but is not part of the commit, so it gets ignored / stranded. Reproduction depends on which installer configured the repo:

**Repro A — generic-installer-only repo (projects but never stages):**
1. In a git repo, run `hero hooks install` (the generic installer) but NOT `hero next install-hooks`.
2. Make any change and `git add <code> && git commit`.
3. The generic `post-commit` hook fires `hero hook post-commit` → `writeCheckpoint()` regenerates `.hero/NEXT.md` (and SNAPSHOT.md) **after** the commit.
4. The regenerated files are left **unstaged** — there is no `git add`. They show as working-tree drift, never travel with the commit. Next session on another machine starts cold.

**Repro B — no staging hook at all (opt-out / legacy / pre-feature):**
1. Repo where `hero next install-hooks` was never run (e.g. `hero init --no-hooks`, or older setup).
2. Same commit flow — handoff files are regenerated by some other path (or by the agent) but never staged.
3. `CLAUDE.md:116` tells the agent to hand-stage as a backstop; when the agent forgets, the files strand. This is the literal user complaint.

The blast radius is every repo that didn't go through `hero init` (hooks on) or `hero next install-hooks`. Because the two installers write the same files with different markers, "I installed Hero's hooks" does not reliably mean "staging is wired."

## Environment Details
Local git repos using Hero. Relevant config:
- `hooks.status_truth` (off by default) gates the generic `pre-commit`.
- `.hero/.no-hooks` sentinel and `--no-hooks` opt out of auto-install.
- `.gitattributes` (written by next-hooks installer, `next_hooks.go:547-560`) declares `merge=union` for `.hero/NEXT.md`, `.hero/next/*.md`, `.hero/QUEUE.md`, `.hero/SNAPSHOT.md` — note SNAPSHOT.md is covered here for merge but NOT in the staging `git add` list.

---

## Root Cause Analysis

Confirmed findings (each `read` against source this session):

1. **Single staging path (read).** `internal/cli/next_hooks.go:268-272` is the only place projected files are staged:
   ```
   git add -- .hero/NEXT.md .hero/next/*.md .hero/QUEUE.md 2>/dev/null || true
   ```
   It runs only inside `hookScript("pre-commit")`, installed by `runNextInstallHooks` / `installNextHooksQuiet` (`next_hooks.go:57,198`), reachable from `hero next install-hooks`, `hero init` default (`init.go:164-174`), and `hero upgrade` refresh (`next_hooks.go:165-193`).

2. **Generic installer does not stage (read).** `internal/hooks/install.go:66-116` installs `pre-commit`, `post-commit`, `post-merge`, `post-checkout`, `prepare-commit-msg`, each calling `hero hook <event>`. None of these scripts stage.

3. **Generic dispatcher projects-without-staging (read).** `internal/cli/hook.go:94-104` — `post-commit` runs `writeCheckpoint()` (regenerates NEXT.md, and `projectSnapshot` regenerates SNAPSHOT.md via `checkpoint.go:206-210`) but issues no `git add`. `hook.go:111-128` — `pre-commit` only runs the opt-in status-truth gate; no checkpoint, no staging.

4. **Two markers, same files (read).** Generic installer uses marker `# Hero git hook` (`install.go:43`). Next-hooks installer uses `# >>> hero next hooks (managed) >>>` (`next_hooks.go:19`). They coexist in `.git/hooks/pre-commit` without conflict, but "a Hero pre-commit hook exists" does not imply staging.

5. **Check detector keyed to next-hooks marker only (read).** `internal/cli/check.go:256-282` calls `preCommitHookInstalled` → `currentPreCommitManagedBlock` (`next_hooks.go:91-142`), which searches for the **next-hooks** marker. A generic-only repo reports "not installed" (correct outcome) but the remediation text only names `hero next install-hooks`, and the check has no awareness of the staging invariant per se — it checks for the specific managed block, not "is staging wired."

6. **SNAPSHOT.md staging gap (read).** SNAPSHOT.md is tracked (`git ls-files` confirms), projected on every checkpoint (`checkpoint.go:206-210`), and declared in `.gitattributes` for merge (`next_hooks.go:555`), but is **absent** from the staging `git add` list (`next_hooks.go:272`). So even on the correct install path, SNAPSHOT.md changes do not travel with the commit.

7. **Gitignored files correctly excluded (read).** `.hero/next/*.local.md`, `.hero/graph.db*` are gitignored (`.gitignore:14-16,25`). The staging glob uses `.hero/next/*.md` (not `*.local.md`), so it does not stage local-only files — preserve this.

Hypothesis (not blocking, flagged): the standing CLAUDE.md/AGENTS.md backstop rule was added precisely because staging is unreliable. Confirmed the rule exists (`agents_md.go:452`); its existence is corroborating evidence, not proof of frequency.

---

## Code Flow (End to End)

How a repo ends up projecting-but-not-staging:

1. `internal/cli/hooks.go:76` — user runs `hero hooks install`; calls `hooks.Install(gitDir)`.
2. `internal/hooks/install.go:66-116` — installs `post-commit` (and `pre-commit`) scripts that call `hero hook <event>`. No `git add` is written into any script.
3. User stages code and commits: `git add <code> && git commit`.
4. `internal/cli/hook.go:111-128` — `pre-commit` fires `hero hook pre-commit`; with `status_truth` off (default) it returns immediately. No staging.
5. Commit is created with only the user's code (handoff files not staged).
6. `internal/cli/hook.go:94-104` — `post-commit` fires `hero hook post-commit`; `writeCheckpoint()` regenerates `.hero/NEXT.md`.
7. `internal/cli/checkpoint.go:206-210` — `projectSnapshot` regenerates `.hero/SNAPSHOT.md`.
8. The regenerated files are now **unstaged working-tree drift** — never `git add`ed. They miss this commit and (absent later staging) all future commits. Handoff state strands locally.

Contrast — the working path (`hero next install-hooks` / `hero init`):

1. `internal/cli/init.go:164-174` or `hero next install-hooks` → `installNextHooksQuiet` (`next_hooks.go:198`).
2. `next_hooks.go:204` writes `hookScript("pre-commit")`.
3. On commit, the pre-commit block (`next_hooks.go:264-272`) runs `hero next checkpoint -q`, `hero index --if-stale`, `hero queue write`, then `git add -- .hero/NEXT.md .hero/next/*.md .hero/QUEUE.md`. Files travel — **but SNAPSHOT.md is omitted** from the `git add`.

---

## Key Files

### Hook installers
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/next_hooks.go` | 257-283 | `hookScript("pre-commit")` — the only staging `git add`; omits SNAPSHOT.md |
| `internal/cli/next_hooks.go` | 57-85, 198-215 | `runNextInstallHooks` / `installNextHooksQuiet` — installs the staging hook |
| `internal/cli/next_hooks.go` | 547-560 | `updateGitAttributes` — merge=union list (includes SNAPSHOT.md) |
| `internal/hooks/install.go` | 18-116 | Generic installer; wires hooks to `hero hook`, never stages |
| `internal/cli/hooks.go` | 60-107 | `hero hooks install` command surface → `hooks.Install` |

### Hook dispatcher
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/hook.go` | 94-104 | Generic `post-commit`: projects NEXT.md, no staging |
| `internal/cli/hook.go` | 111-128 | Generic `pre-commit`: status-truth gate only |
| `internal/cli/checkpoint.go` | 90-213 | `writeCheckpoint` → projects NEXT.md + SNAPSHOT.md |

### Detection / messaging
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/check.go` | 251-283 | Pre-commit-hook health check; keyed to next-hooks marker |
| `internal/cli/next_hooks.go` | 91-155 | `preCommitHookInstalled` / `preCommitHookStale` — marker-scoped |
| `internal/install/agents_md.go` | 452 | The manual-staging backstop rule (evidence) |
| `internal/cli/init.go` | 160-186 | Default install path (next-hooks) |

---

## Secondary Defects

1. **SNAPSHOT.md never staged (confirmed).** Tracked + projected + merge-declared, but absent from the staging `git add` (`next_hooks.go:272`). Even on the correct hook, SNAPSHOT.md changes strand. Fix in the same change.
2. **Stale `.gitattributes` vs. staging-list divergence.** `.gitattributes` (merge=union) lists SNAPSHOT.md; the staging list does not. These two lists should be derived from one source so they cannot drift apart again.
3. **`hero check` remediation is single-installer-blind.** It names only `hero next install-hooks`; a generic-only repo gets a correct warn but no signal that staging specifically is the missing invariant.

---

## Notes
- `.hero/QUEUE.md` may be dropped as a file by a sibling effort. The staging glob already uses `2>/dev/null || true`, so a missing QUEUE.md is a safe no-op. Do not make staging depend on QUEUE.md's existence — stage what exists.
- Do NOT stage gitignored handoff files: `.hero/next/*.local.md`, `.hero/graph.db*`. The current `.hero/next/*.md` glob already excludes `*.local.md`; preserve that exactly.
- Two installers writing the same hook files is the structural defect. The cleanest fix removes the *possibility* of hooks-without-staging, not just patches one path.

---

## Recap
Handoff-file staging lives in only one of Hero's two git-hook installers and is opt-in, so a repo configured via the generic `hero hooks install` (or one that skipped `hero next install-hooks`) regenerates `.hero/NEXT.md`/`SNAPSHOT.md` on every commit but never stages them — the user's recurring "NEXT.md isn't in my commit" pain. Root cause is a design split (two installers, one staging path); severity is high because it silently breaks Hero's core cross-session/cross-machine handoff promise. The fix makes staging an unconditional invariant of a single default install path, extends `hero check` to flag projecting-but-not-staging repos, and aligns the staged-file list with the actually-projected tracked files (adding SNAPSHOT.md).

---

## Goal
"Hooks installed" always implies "projected handoff files travel with the commit." There is no reachable configuration where Hero projects handoff files on commit but fails to stage them. The staged-file list matches the actually-projected tracked files (`.hero/NEXT.md`, `.hero/next/*.md`, `.hero/SNAPSHOT.md`; `.hero/QUEUE.md` if present), never stages gitignored files, and `hero check` loudly flags a repo where projected files are tracked but staging isn't wired.

## Acceptance Criteria

- WHEN a repo has Hero hooks installed via the default install path THE SYSTEM SHALL stage all projected tracked handoff files (`.hero/NEXT.md`, `.hero/next/*.md`, `.hero/SNAPSHOT.md`, and `.hero/QUEUE.md` when present) into the commit being made.
- THE SYSTEM SHALL expose no reachable hook configuration that projects handoff files on commit without also staging them.
- WHEN `hero check` runs in a git repo where projected handoff files are tracked but no staging-capable pre-commit hook is wired THE SYSTEM SHALL emit a warning identifying the missing staging invariant and the single command to fix it.
- THE SYSTEM SHALL never stage gitignored handoff files (`.hero/next/*.local.md`, `.hero/graph.db*`).
- IF `.hero/QUEUE.md` does not exist THEN THE SYSTEM SHALL stage the remaining handoff files without error.
- THE SYSTEM SHALL derive the staging-file list and the `.gitattributes` merge=union list from a single source so they cannot diverge.

## Suggested Fix Approach

**Recommended: Option (b) — consolidate to one installer, staging unconditional, plus (a) as the consolidation mechanic.**

Make the next-hooks installer (the one that already stages) the single source of truth, and fold the generic installer's non-staging hook events into it OR have the generic `post-commit`/`pre-commit` delegate to the same staging logic. The goal is structural: no code path installs a Hero pre-commit hook that lacks staging.

1. **Single staged-file list (`internal/cli/next_hooks.go`).** Extract the handoff-file list into one package-level slice and use it for both the staging `git add` and the `.gitattributes` merge=union block.

   **Before** (`next_hooks.go:268-272`, staging):
   ```go
   stage = `
     # Re-stage projected files so they travel with the commit. ...
     git add -- .hero/NEXT.md .hero/next/*.md .hero/QUEUE.md 2>/dev/null || true`
   ```
   **After:**
   ```go
   stage = `
     # Re-stage projected files so they travel with the commit. ...
     git add -- .hero/NEXT.md .hero/next/*.md .hero/SNAPSHOT.md .hero/QUEUE.md 2>/dev/null || true`
   ```
   **Why:** adds the missing SNAPSHOT.md (tracked + projected + merge-declared but never staged). The `2>/dev/null || true` already makes missing QUEUE.md a safe no-op. Keep the `.hero/next/*.md` glob (excludes `*.local.md`). Ideally both this list and the `.gitattributes` list (`next_hooks.go:551-556`) are generated from one constant so they cannot drift (Secondary Defect 2).

2. **Eliminate the projecting-but-not-staging path (`internal/cli/hook.go` / `internal/hooks/install.go`).** Choose one:
   - **(b) Consolidate installers:** make `hero hooks install` (`internal/cli/hooks.go:76`) also install the next-hooks staging block (call `installNextHooksQuiet`), so any "install Hero hooks" action wires staging. Remove the divergence where generic-only hooks project-without-staging.
   - **(a) Make generic post-commit stage:** if the two installers must stay separate, the generic `post-commit` handler (`internal/cli/hook.go:104`) must `git add` the projected files after `writeCheckpoint()`. Note this stages them for the *next* commit (post-commit runs after commit creation) — a pre-commit stage is strictly better, so prefer (b).

   **Recommendation:** (b). One installer, staging always wired, no second code path to keep in sync. The generic installer's other events (post-checkout status flips, prepare-commit-msg) remain; only the pre-commit/staging concern is unified.

3. **Extend `hero check` staging detector (`internal/cli/check.go:251-283`).** Today it warns when the next-hooks managed block is absent. Strengthen the message and detection so a repo with *some* Hero pre-commit hook but *no staging block* is flagged distinctly: "Hero pre-commit hook present but projected handoff files are not staged — they won't travel with commits." Detect by checking whether the installed pre-commit content contains the staging `git add` of handoff files (or the next-hooks managed marker), not merely whether any Hero hook exists.

4. **Drop / re-scope the backstop rule (`internal/install/agents_md.go:452`).** Once staging is unconditional, the manual hand-staging instruction is redundant for correctly-installed repos. Keep a one-line note that the staging hook is what guarantees travel and `hero check` flags when it's missing; remove the framing that hand-staging is the routine expectation. (Optional, low-risk; coordinate with `next-as-projection` doc owners.)

## Delivery

Implemented per Option (b) + the SNAPSHOT/single-list fix. Files changed:

- `internal/cli/next_hooks.go` — added package-level `handoffFilePaths` single source of truth; `hookScript("pre-commit")` and `updateGitAttributes` both derive from it (adds `.hero/SNAPSHOT.md`). Staging rewritten as a **per-path loop** (`for p in …; do git add -- "$p" 2>/dev/null || true; done`) because a single combined `git add -- a b c` aborts the whole add and stages **nothing** when any pathspec matches no file (dropped QUEUE.md / empty `next/*.md` glob) — the latent defect that broke the missing-QUEUE AC. Added `preCommitHasHeroHookButNoStaging` detector.
- `internal/cli/hooks.go` — `runHooksInstall` now also calls `installNextHooksQuiet(projectRoot)` so the generic `hero hooks install` surface can never produce hooks-without-staging. The two managed blocks coexist; uninstall already strips both.
- `internal/cli/check.go` — added a distinct `hero check` warning for "Hero pre-commit hook present but handoff-file staging not wired," naming the invariant and the single fix command.
- `internal/install/agents_md.go` — re-scoped the manual-staging backstop rule (Change 4, done): staging is now automatic on every install path; hand-staging is a backstop only when `hero check` flags it.
- `domains/engineering/AGENTS.md` — regenerated golden pack file to match the re-scoped backstop rule (via `HERO_REGEN_PACK_AGENTS=1`).
- Tests: `internal/cli/next_hooks_test.go` (SNAPSHOT staging + single-list invariant), `internal/cli/check_test.go` (misconfig flag), `internal/cli/hooks_staging_integration_test.go` (NEW — default-install staging, **generic-install staging regression**, gitignored-never-staged, missing-QUEUE no-op).

## Test Plan

### Existing test review
- `internal/cli/next_hooks_test.go` — covers next-hooks install/uninstall and the managed-block/`.gitattributes` content. Extend to assert the staging `git add` line includes `.hero/SNAPSHOT.md`.
- `internal/cli/checkpoint_test.go` — covers NEXT.md/SNAPSHOT.md projection. No staging assertions today.
- `internal/hooks/install.go` has install/uninstall/status; verify there is a `*_test.go` covering generic install behavior and extend for the consolidation.

### Test changes needed
1. **Staging includes SNAPSHOT.md (unit).** Assert `hookScript("pre-commit")` contains `git add` of `.hero/NEXT.md`, `.hero/next/*.md`, `.hero/SNAPSHOT.md`, `.hero/QUEUE.md`.
2. **Single-list invariant (unit).** Assert the staging list and the `.gitattributes` merge=union list are derived from the same source (same set of paths).
3. **Install ⇒ staging integration (integration).** In a temp git repo: run the default install path, create a tracked code change, regenerate handoff files, `git commit`, then assert `.hero/NEXT.md` and `.hero/SNAPSHOT.md` are part of the commit tree (not unstaged drift).
4. **No hooks-without-staging (integration).** Run `hero hooks install` (the generic surface) in a temp repo; assert the resulting pre-commit configuration stages handoff files on commit (proving consolidation closed the gap).
5. **Gitignored never staged (integration).** Create `.hero/next/foo.local.md` and a `.hero/graph.db`; commit; assert neither is staged.
6. **Missing QUEUE.md is a no-op (integration).** Remove `.hero/QUEUE.md`; commit; assert no error and the remaining files stage.
7. **`hero check` flags misconfig (integration).** Configure a repo with a Hero pre-commit hook lacking the staging block; run `hero check`; assert a warning naming the staging invariant and a single fix command.

### Regression scope
- `hero hooks uninstall` / `uninstallNextHooks` must still cleanly strip the consolidated hook (both markers) — verify uninstall round-trips.
- `hero upgrade` hook-refresh (`refreshHooksIfPresent`, `next_hooks.go:165-193`) compares installed block to `hookScript("pre-commit")`; changing the staged-file list will mark existing hooks stale and trigger a refresh — confirm that's the intended (good) behavior and the stale-detector test is updated.
- Pre-commit performance: staging adds one `git add`; already best-effort. No measurable regression expected.
- Ensure adding SNAPSHOT.md to staging doesn't create commit noise loops (post-commit projection re-touching SNAPSHOT.md after a pre-commit stage). Pre-commit staging captures the pre-commit-projected state; verify the post-commit checkpoint doesn't produce a perpetually-dirty SNAPSHOT.md.

## Boundaries
- Not changing what the checkpoint projects or how NEXT.md/SNAPSHOT.md are rendered — only whether/where they are staged.
- Not addressing the sibling effort that may drop `.hero/QUEUE.md` as a file; this spec only makes staging tolerant of its absence.
- Not redesigning the status-truth `pre-commit` gate or the post-checkout status-flip behavior.
- Not introducing a new hook event; the work consolidates existing pre-commit/post-commit behavior.
- Not changing `.gitignore`; gitignored handoff files stay excluded.

## Risks
- **Stale-hook churn on upgrade:** changing the staged-file list flips every installed hook to "stale," so `hero upgrade` / `hero check` will prompt refresh across repos. Intended, but communicate it.
- **Double-staging / dirty-tree loops:** if both a consolidated pre-commit stage and the generic post-commit projection touch SNAPSHOT.md, ensure no perpetual dirty state. Prefer pre-commit staging as the single staging point.
- **Uninstall completeness:** consolidation means uninstall must strip the unified block; an incomplete strip leaves a dead hook. Covered by regression tests.
- **User-customized hooks:** the marker-bounded merge (`mergeMarkerBlock`) preserves out-of-marker user content; confirm consolidation keeps that guarantee.

## Kickoff

Makes Hero always stage projected handoff files (NEXT.md, SNAPSHOT.md, next/*.md) into commits — fixes "NEXT.md isn't in my commit" by closing the two-installer split where one installer projects but never stages.

**Status:** planning — diagnosis complete, root cause confirmed at file:line. No code yet.

**Pick up at:** consolidate so there's one install path that always stages. Start by (1) adding `.hero/SNAPSHOT.md` to the staging `git add` in `hookScript` and deriving both the staging list and the `.gitattributes` list from one constant, then (2) make `hero hooks install` route through `installNextHooksQuiet` so the generic installer can't produce hooks-without-staging.

→ `.hero/planning/bugs/next-unconditional-commit-staging/spec.md`

**Files:** `internal/cli/next_hooks.go:257-283`, `internal/cli/hook.go:94-128`, `internal/hooks/install.go:66-116`, `internal/cli/check.go:251-283`
**Skip:** patching only the generic post-commit to stage (post-commit stages for the *next* commit — pre-commit consolidation is correct); depending on QUEUE.md existing (a sibling effort may drop it).
