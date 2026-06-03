---
title: Retro — pre-commit-auto-stage-next
type: retro
spec: pre-commit-auto-stage-next
status: complete
created: 2026-05-12
relations:
  - target: pre-commit-auto-stage-next
    kind: retrospects
  - target: spec-status-integrity
    kind: motivates
---

## Summary

The spec was marked `status: completed` in v0.4, but one acceptance criterion (AC #11, the CLAUDE.md/AGENTS.md agent-side backstop) never landed. The shortfall surfaced today (2026-05-12) when an agent in a fresh worktree — without the pre-commit hook installed — defaulted to treating `.hero/NEXT.md` and `.hero/next/<user>.md` as "session scratch" and proposed skipping them on commit. That is precisely the failure mode the Layer 3 backstop was meant to prevent.

The fix landed today in commit `da246f4`: the rule is now in `internal/install/agents_md.go` (single template source for both `CLAUDE.md` and `AGENTS.md` managed blocks), plus refreshed renderings in this repo's `CLAUDE.md`, `AGENTS.md`, and `domains/engineering/AGENTS.md`.

## What matched the plan

- **Hook side shipped clean.** `hero next install-hooks` writes a managed block that calls `hero next checkpoint -q` followed by `git add -- .hero/NEXT.md .hero/next/*.md .hero/QUEUE.md` ([internal/cli/next_hooks.go:361](../../../internal/cli/next_hooks.go:361)). Re-install is idempotent via the marker block.
- **Install discoverability shipped.** `hero check` reports a missing managed block (refs `hookMarkerStart` in [internal/cli/check_test.go:80](../../../internal/cli/check_test.go:80)). Auto-install on first-run paths is wired.
- **Git escape hatches preserved.** `--no-verify`, `--allow-empty`, `-o pathspec` all behave per the spec.
- **No-error on missing files.** `2>/dev/null || true` swallows the fresh-repo case.

## What deviated

- **AC #11 (CLAUDE.md/AGENTS.md backstop) never shipped.** The spec's "Changes" section explicitly listed it as a one-line edit; AC made it normative. Yet `internal/install/agents_md.go` — the canonical single source for the Hero-managed block on every harness — was never touched. Spec was marked completed anyway.
- **Bonus extension found in delivery:** the hook stages `.hero/QUEUE.md` as well as the two files named in the spec. Pragmatic addition, but undocumented relative to the spec text.

## What was harder than expected

- Not "harder" but **easier-to-miss than expected:** the shipped pieces are the substantial engineering (hook plumbing, marker-block lifecycle, `hero check` integration, scan auto-install). The unshipped piece was a literal one-line addition to a string-builder. The smallest item on the Changes list became the load-bearing failure mode.

## What was easier or unnecessary

- The hook's `git add` glob worked across all expected edge cases without iteration — the `2>/dev/null || true` swallow plus `--` separator handled every documented scenario.
- The `core.hooksPath` vs `.git/hooks/` debate (boundaries section) settled on the existing model and never required revisiting.

## Learnings

### 1. Convention: cross-harness agent rules belong in `agents_md.go`

When Hero introduces a behavioral rule that agents on multiple harnesses (Claude, Codex, Cursor, OpenCode, Copilot, generic) must follow, the canonical insertion point is `generateAgentsMdBody()` in [internal/install/agents_md.go](../../../internal/install/agents_md.go). That function is the single source for the managed block written into both project-root `CLAUDE.md` and project-root `AGENTS.md`. Domain-pack-specific rules go in `domains/<pack>/AGENTS.md`.

Editing only `CLAUDE.md` — which I almost did mid-fix — leaves every non-Claude agent without the rule. Captured as a rule in `.hero/knowledge/rules/project-rules/cross-harness-agent-rules.md`.

### 2. Decision evidence: spec-status integrity

The completed `pre-commit-auto-stage-next` is now a concrete data point for the in-flight [spec-status-integrity](../../../.hero/planning/spec-status-integrity/) feature. A spec with 11 ACs shipped 10 and was marked done; the missed AC was small, late in the Changes list, and invisible to anyone who only checked "does the headline behavior work?" If `spec-status-integrity` had been live, the gap would have been called out at status transition. Filed as a note in `.hero/knowledge/notes/spec-status-integrity-motivation.md`.

### 3. Estimation calibration

Small terminal items on a Changes list need an explicit verification checkpoint, not "we'll do it, it's trivial." The risk of slippage is inversely correlated with item size — trivial items don't get tracked, don't get tested, don't get noticed missing.

## Knowledge captured

- `.hero/knowledge/rules/project-rules/cross-harness-agent-rules.md` — where to add agent rules so every harness sees them
- `.hero/knowledge/notes/spec-status-integrity-motivation.md` — evidence point for the in-flight feature
