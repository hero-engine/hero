---
title: "`hero install --target claude` emits both CLAUDE.md and AGENTS.md"
type: bug
status: planning
priority: medium
severity: low
---

## Goal

`hero install --target claude` should emit `CLAUDE.md` only (creating it if absent, updating its managed block if present). `AGENTS.md` should be reserved for harnesses that don't have a native conventions file — emitted when targeting cursor / codex / opencode / aider / generic, or as an explicit opt-in. Today (observed during hero-cloud workspace bring-up on 2026-05-15), running `--target claude` produces both files side-by-side.

## Kickoff

Reproduce: cd into a clean repo with no `CLAUDE.md` or `AGENTS.md`, run `hero install --target claude`. Expected: only `CLAUDE.md` lands. Observed: both `CLAUDE.md` and `AGENTS.md` are emitted with the same managed-block content. Fix likely lives in the install target dispatch in the hero CLI — read the install command source, find where both files get written, and gate `AGENTS.md` emission on the target not being `claude` (or on a generic/fallback target). Update tests to cover each target's expected file set.

## Problem

Confusion + duplication. Two files with the same managed block means future edits drift between them; readers don't know which is authoritative; the AGENTS.md vs. CLAUDE.md distinction (a deliberate "harness-native vs. fallback" design) collapses. Discovered in hero-cloud during Phase 1 workspace setup; user expected `--target claude` to skip AGENTS.md.

## Suggested Fix Approach

1. Locate the install target handling (probably `cmd/hero/install*.go` or `internal/install/...`)
2. Audit the file-emission step: which targets emit which files
3. Make the rule explicit: `claude` → `CLAUDE.md` only; everything else (including default/unspecified) → `AGENTS.md`; optional `--with-agents-md` flag if a user wants both intentionally
4. Add a test per target asserting the emitted file set
5. Update install docs / `--help` to reflect the rule

## Acceptance Criteria

- WHEN `hero install --target claude` runs in a clean repo, THE SYSTEM SHALL emit `CLAUDE.md` and SHALL NOT emit `AGENTS.md`
- WHEN `hero install --target <other>` runs in a clean repo (cursor, codex, opencode, aider, generic), THE SYSTEM SHALL emit `AGENTS.md` and SHALL NOT emit `CLAUDE.md`
- WHEN `hero install` runs with no target flag, THE SYSTEM SHALL emit `AGENTS.md` as the harness-agnostic fallback
- WHEN `hero install --target claude` runs against a repo that already has `AGENTS.md`, THE SYSTEM SHALL leave `AGENTS.md` untouched (no orphaned file removal; user can delete deliberately)
- WHERE both files exist in the same repo, `hero check` SHALL flag the duplication as a warning

## Out of Scope

- Migrating existing repos that already have both files
- Auto-deleting orphaned AGENTS.md on subsequent installs (separate decision)
- Cross-harness portability of the managed block content

## Open Questions

- Should `hero install --target all` exist for multi-harness teams, or is "run install once per harness" the model?
- For hero and hero-cloud which currently have both files committed — leave as-is, or run a one-shot cleanup once the fix lands?
