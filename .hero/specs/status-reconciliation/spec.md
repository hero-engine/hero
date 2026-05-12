---
title: "Git-Derived Status Reconciliation"
type: feature
status: completed
created: 2026-04-13
milestone: v0.2
tags: [check, git, status, reconciliation, automation]
depends-on: []
horizon: now
---

# Git-Derived Status Reconciliation

## Problem

Spec status fields drift out of sync with reality. An agent finishes implementing
a feature but forgets to update `status: planning` → `status: delivering`. A PR
merges but the spec stays at `delivering` instead of moving to `completed`. The
README, dashboard, and `hero status` all show stale information because they read
from spec frontmatter that nobody updated.

This is a systemic issue — relying on humans or agents to remember manual status
updates will always be lossy.

## Approach

Two complementary strategies:

### 1. Git-derived reconciliation (after the fact)

`hero check` gains the ability to compare each spec's declared `status:` against
git evidence and report (or fix) mismatches.

**Evidence signals:**

| Signal | Implies |
|--------|---------|
| Spec's `FilesTouched` have commits on a non-default branch since spec creation | At least `delivering` |
| Spec's branch is merged to default branch | Candidate for `completed` |
| Spec status is `planning` but `FilesTouched` have uncommitted changes | Work started, should be `delivering` |
| Spec has `claimed_by` set | At least `delivering` (someone is working on it) |

**Rules:**
- Only promote forward: `planning` → `delivering` → `completed`. Never demote.
- Never auto-complete — too destructive. Only suggest.
- `hero check` reports mismatches as warnings by default.
- `hero check --reconcile` auto-fixes `planning` → `delivering` when evidence is clear.
- Specs without a `## Changes` section (no `FilesTouched`) are skipped for git checks.

**Implementation:**

New package `internal/gitutil/` with helpers:
- `DefaultBranch(projectRoot)` — detect main/master
- `FilesChangedSinceBranch(projectRoot, base)` — files with commits on current branch vs base
- `FilesChangedUncommitted(projectRoot)` — staged + unstaged + untracked
- `IsMergedToDefault(projectRoot, branch)` — check if branch is merged
- `CommitsSince(projectRoot, path, since)` — commits touching a path after a date

New function `internal/reconcile/reconcile.go`:
- `Reconcile(heroDir, projectRoot)` → `[]Finding`
- Each `Finding` has: spec slug, current status, suggested status, evidence string
- Pure logic — does not write. Callers decide what to do with findings.

Integration in `check.go`:
- After existing stale/unclaimed checks, run reconciliation
- Print findings as a new section
- With `--reconcile`, also write updated frontmatter

### 2. Agent instruction updates (during implementation)

Update `feature-delivery-lead.md` and `engineer.md` to include explicit
session-end instructions:

> When you finish implementing a spec or making significant progress:
> 1. Update the spec's `status:` frontmatter to `delivering`
> 2. Update the spec's `## Changes` section with files you touched
> 3. Run `hero index` to refresh the search index

This is a documentation-only change — no code needed, just better instructions.

## Changes

- `internal/gitutil/gitutil.go` — New package: git helper functions
- `internal/gitutil/gitutil_test.go` — Tests for git helpers
- `internal/reconcile/reconcile.go` — Reconciliation logic
- `internal/reconcile/reconcile_test.go` — Tests for reconciliation
- `internal/cli/check.go` — Add `--reconcile` flag, integrate reconciliation
- `internal/cli/check_test.go` — Tests for new check behavior
- `agents/feature-delivery-lead.md` — Add session-end spec update instructions
- `agents/engineer.md` — Add session-end spec update instructions

## Acceptance Criteria

1. `hero check` shows a "Status Drift" section listing specs where git evidence
   contradicts declared status
2. `hero check --reconcile` auto-promotes `planning` → `delivering` when files
   in the spec's Changes section have been modified
3. Agent instructions include session-end spec update guidance
4. All existing tests continue to pass
5. Works in non-git directories (gracefully skips git checks)
