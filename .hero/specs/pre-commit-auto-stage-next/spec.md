---
title: Pre-Commit Auto-Stage — Projected NEXT Files Travel With Every Commit
type: feature
status: completed
milestone: v0.4
tags: [git, hooks, next, projection, workspace, handoff]
created: 2026-04-29
relations:
  - target: git-hook-integration
    kind: extends
  - target: agent-cold-start
    kind: related
  - target: client-id-user-scoping
    kind: related
horizon: now
---

## Goal

Make `.hero/NEXT.md` and `.hero/next/<user>.md` travel with every commit, automatically, regardless of which harness (Claude Code, Cursor, Codex, plain human) is making the commit. When the next session starts on another machine, the workspace state is already there — not stranded as unstaged drift on the previous laptop.

## Problem

Hero already does the hard parts:

- `hero next install-hooks` writes a `pre-commit` hook that calls `hero next checkpoint -q`, which regenerates `.hero/NEXT.md` and `.hero/next/<user>.md` from the graph.
- A `merge=hero-next` driver regenerates these files on merge conflicts so they never mark up.

But there's a hole in the middle. AI harnesses commit by listing the files they touched: `git add internal/foo.go internal/bar.go && git commit`. The pre-commit hook fires, regenerates the projected files in the working tree — but never stages them. The commit lands without them. The agent pushes. The regenerated NEXT state stays as unstaged modifications on that laptop. Pull from another machine and it's missing. Two agents on two machines drift the same file in parallel and the merge driver papers over it, but the *state* that should have been carried forward never made it onto the branch.

This is a direct hit on Hero's mission promise: every session starts as smart as the last one ended. Today, that promise quietly fails any time an agent commits narrowly — which is most of the time.

## Design

### Where the fix goes

Inside the existing managed block in `.git/hooks/pre-commit` (the one written by `hero next install-hooks`). After `hero next checkpoint -q` regenerates the projected files, the same block calls `git add` for them. One file, one extension, no new commands.

The hook is already marker-delimited and idempotent — re-running `hero next install-hooks` after this change replaces the managed block in place. Users who already installed the hook get the new behavior on the next install/upgrade.

### Hook script (after change)

```sh
# >>> hero next hooks (managed) >>>
# Refresh projected NEXT files from the graph so the commit reflects
# current state, then stage them so they travel with the commit.
# Best-effort: a hero failure shouldn't block git operations.
if command -v hero >/dev/null 2>&1; then
  hero next checkpoint -q || true
  # Stage the projected files. -- guard makes git no-op cleanly if
  # any path doesn't exist yet (e.g. solo mode, no per-user file).
  git add -- .hero/NEXT.md .hero/next/*.md 2>/dev/null || true
fi
# <<< hero next hooks (managed) <<<
```

Notes on the script:

- **`git add` outside `--` glob brittleness.** `.hero/next/*.md` expands in-shell. If no file matches (fresh repo), the literal pattern is passed and `git add` errors — `2>/dev/null || true` swallows that. Same for `NEXT.md` if it doesn't exist.
- **Local-state file stays out.** `.hero/next/<user>.local.md` is gitignored — `git add` on a gitignored path is a no-op. No need to filter explicitly.
- **No staging of untracked file types.** The glob is intentionally narrow (`.hero/NEXT.md`, `.hero/next/*.md`). It does *not* stage `.hero/knowledge/`, `.hero/specs/`, `.hero/version.json`, or anything else. Those are user-edited files; auto-staging them would surprise people.

### What about other hero workspace files?

Three categories of files under `.hero/`. Only the first gets auto-staged.

| Path | Behavior | Rationale |
|---|---|---|
| `.hero/NEXT.md`, `.hero/next/*.md` | Auto-staged by hook | Projected from graph every checkpoint. Total-rewrite. The whole point of this spec. |
| `.hero/knowledge/*.md`, `.hero/specs/**`, `.hero/mission.md` | Not auto-staged | User-or-skill-authored. Intentional `git add`. |
| `.hero/version.json`, `.hero/automations/**` | Not auto-staged | Config / state files. Change on intentional actions, not per-turn churn. |
| `.hero/graph.db`, `.hero/index.db`, `.hero/events.log`, `.hero/reports/**`, `.hero/smoke/**`, `.hero/next/*.local.md` | Gitignored | Per-machine derived state. Already handled. |

If future projections produce more files-that-must-travel, extend the glob — but keep the rule narrow. The default position is "Hero does not stage things on your behalf." Projected NEXT files are the principled exception because (a) they are deterministic regenerations from a shared graph and (b) Hero's mission collapses if they don't travel.

### Edge cases

- **`git commit -o <pathspec>` (only-this-pathspec mode).** Git uses a partial index built from the pathspec — modifications staged into the main index by the hook do not affect the partial commit. The hook's `git add` succeeds but the projected files don't land in *that* commit. They remain staged in the main index and are picked up by the next normal commit. This is acceptable: `-o` is an explicit "I want only these paths" instruction, and the next commit catches up.
- **`--no-verify`.** Skips the hook entirely. Standard git escape hatch, intentionally preserved. Document it.
- **`--allow-empty`.** Hook runs, files get staged if modified; otherwise no-op. Behaves the same as without this change.
- **Detached HEAD / rebase-in-progress.** `hero next checkpoint -q` runs as before; staging is best-effort. Not a regression.
- **`core.hooksPath` set elsewhere.** If the user has redirected hooks to a different directory, our install writes to `.git/hooks/` and is ignored. Pre-existing limitation of the install command — out of scope; flagged as a follow-up in `Boundaries`.
- **Hook expansion / no-`bash`.** Existing hook header is `#!/usr/bin/env bash`. The new line uses POSIX `git add -- pathspec`. No bashisms introduced.

### Installation discoverability (the "new clone" problem)

Today: a user clones the repo, hero workspace files start to drift, they never run `hero next install-hooks`, and the bug recurs. Three layers, weakest-to-strongest:

1. **Surface in `hero check`.** `hero check` already runs workspace-health checks. Add a check: "pre-commit hook installed? managed block present?". On failure, suggest `hero next install-hooks`. Cheap, non-invasive.
2. **Auto-install on first `hero scan`.** When `hero scan` runs in a repo that has `.git/` but no managed block in `.git/hooks/pre-commit`, install the hook automatically and print one line: `installed pre-commit hook (run 'hero next uninstall-hooks' to remove)`. Idempotent — no-ops on re-run because the marker is present.
3. **CLAUDE.md backstop.** Add a snippet to the Hero-managed CLAUDE.md guidance (where applicable) telling agents: "When committing, include any modified files under `.hero/NEXT.md` and `.hero/next/`." Soft. Last-resort safety net for repos that haven't installed the hook.

Layers 1 and 2 are in scope for this spec. Layer 3 is in scope but lighter-weight (one-liner addition).

### Why not `core.hooksPath` + `.githooks/`?

Considered. Tradeoffs:

- **Pro:** The hook script lives in the repo, tracked, no per-clone install step beyond `git config core.hooksPath .githooks`.
- **Con:** Conflicts with the existing model. `git-hook-integration` and `next install-hooks` already write into `.git/hooks/` with marker blocks that preserve user content. Switching to `core.hooksPath` would either (a) fork two installation models or (b) require migrating the existing approach. Both are bigger than the problem warrants.
- **Decision:** Stay with `.git/hooks/` + marker-block model. Improve installation discoverability instead (the three layers above). Revisit if/when a future spec consolidates Hero's git integration story.

## Changes

- `internal/cli/next_hooks.go` — extend `hookScript("pre-commit")` to add the `git add -- .hero/NEXT.md .hero/next/*.md` line inside the managed block. Update the `Long` description for `nextInstallHooksCmd` to mention auto-staging. Update the `pre-commit` description in the docstring.
- `internal/cli/checkpoint.go` — no change required. The Go-side `writeCheckpoint` doesn't need to call `git add`; that's the hook's job. Keeps the command pure for non-hook callers.
- `internal/cli/check.go` (or wherever `hero check` registers checks) — add a `pre-commit hook installed` check that inspects `.git/hooks/pre-commit` for `hookMarkerStart`. On miss, recommend `hero next install-hooks`.
- `internal/cli/scan.go` (or the equivalent first-run path) — auto-install the pre-commit hook if missing and the repo has a `.git/`. One-line stdout message. Skip if `--no-hooks` flag passed (add the flag).
- `internal/cli/next_hooks_test.go` — add a test that asserts the generated managed block contains the `git add` line and that the file glob is correct.
- `CLAUDE.md` (project-root, the one Hero writes/manages) — add one line under commit guidance: "Include any modified `.hero/NEXT.md` and `.hero/next/*.md` files in commits if the pre-commit hook isn't installed."
- `.hero/specs/git-hook-integration/spec.md` — append a note pointing to this spec as the auto-staging extension. (Or rely on the `relations.extends` link in the new spec; pick one — the link is enough.)

## Acceptance Criteria

- WHEN a user (or agent) runs `git commit` in a repo with the Hero pre-commit hook installed THE SYSTEM SHALL stage any modified `.hero/NEXT.md` and `.hero/next/*.md` files into the index before the commit is recorded.
- WHEN `hero next install-hooks` is run THE SYSTEM SHALL write a `pre-commit` hook whose managed block calls `hero next checkpoint -q` followed by `git add -- .hero/NEXT.md .hero/next/*.md`.
- WHEN `hero next install-hooks` is re-run after this spec ships THE SYSTEM SHALL replace the prior managed block in place without disturbing user content outside the markers.
- IF the user passes `--no-verify` to `git commit` THEN THE SYSTEM SHALL skip auto-staging (standard git behavior, hook does not run).
- IF the user passes `git commit -o <pathspec>` THEN THE SYSTEM SHALL leave the projected files staged in the main index for the next normal commit (partial-commit pathspec mode is documented as out of scope).
- IF `.hero/NEXT.md` or `.hero/next/*.md` does not exist (fresh repo, solo mode) THEN THE SYSTEM SHALL not error — `git add` failures are swallowed and git proceeds.
- WHEN `hero check` runs THE SYSTEM SHALL report whether the pre-commit hook's managed block is present and recommend `hero next install-hooks` if not.
- WHEN `hero scan` runs in a git repo without the pre-commit managed block THE SYSTEM SHALL install the hook automatically and print a one-line confirmation, unless `--no-hooks` is passed.
- THE SYSTEM SHALL complete the pre-commit hook in under 200ms on a typical commit (existing checkpoint already meets this; the added `git add` is microseconds).
- THE SYSTEM SHALL preserve `--no-verify`, `--allow-empty`, and other standard git flags as escape hatches without modification.
- THE SYSTEM SHALL include CLAUDE.md guidance telling agents to include hero workspace files in commits as a backstop for repos without the hook.

## Boundaries

- Does **not** change `hero next checkpoint` behavior. Checkpoint still writes the files; only the hook's surrounding shell additionally stages them.
- Does **not** auto-stage anything outside `.hero/NEXT.md` and `.hero/next/*.md`. Knowledge entries, specs, version.json, automations, and other `.hero/**` files remain manually staged.
- Does **not** add a `post-commit` or `post-merge` auto-stage. `post-merge` already calls checkpoint to regen; staging would happen in the *next* commit and is fine.
- Does **not** migrate to `core.hooksPath` / `.githooks/`. Stays with `.git/hooks/` + marker block model. Revisit in a future consolidation spec.
- Does **not** touch the `hero hooks install` (status-truth) command from `git-hook-integration`. Different concern, different managed block, different lifecycle. The two coexist in `.git/hooks/pre-commit` because the marker-block merger preserves both.
- Does **not** auto-install hooks on every `hero` invocation — only on `hero scan` (the explicit setup path) and surfaces in `hero check` (the explicit health path). Avoids astonishment.
- Does **not** address bare repos, submodules, or worktrees beyond what the existing hook install already supports. Same caveat as `git-hook-integration`.

## Mission Fit

> "Does this make the next agent session start smarter than the last one ended — and does it raise the floor for everyone, not just the senior dev who already knows what to ask?"

Yes, directly. Hero's foundational promise is that the next session starts where the last one ended. Today that promise leaks every time an agent commits narrowly — the regenerated handoff state stays on the previous machine, invisible to the next session. This spec closes the leak with a one-line hook change, gated by the install-discoverability improvements so it actually ships to clones in the wild. Floor-raising: works the same for a senior who knows the workflow and a brand-new clone whose owner has never heard of `hero next install-hooks`.
