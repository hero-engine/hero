---
title: AGENTS.md Project Structure section lies about content-path locations
slug: hero-workspace-not-self-describing
type: bug
status: planning
severity: high
created: 2026-05-12
tags: [install, agents-md, symlink-refactor, opencode, multi-harness]
---
# AGENTS.md Project Structure section lies about content-path locations

## Problem

After `hero install` on a fresh project (any non-Claude-Code harness — opencode, codex, cursor, generic), the AGENTS.md file Hero writes points the host model at `commands/`, `agents/`, `skills/` directories that **do not exist** in the project root. The v0.8 single-source-install refactor moved the canonical content tree under `.hero/{agents,commands,skills}/` (with `.opencode/agents → ../.hero/agents` etc. symlinks), but the AGENTS.md managed-region body was not updated.

Reporter saw this in an opencode session on a Groovy/Spock + Vitest/Playwright project: after `hero scan` completed, the model started reasoning from first principles about what `.hero/`, `agents/`, `commands/`, AGENTS.md "managed sections" meant — bouncing between tangents like "Investigating glob issues" and "Need update AGENTS nonmanaged top maybe user asked enrich". The model couldn't find the agents/commands/skills directories AGENTS.md described, so it improvised against the `.hero/` content it could `ls`.

In Claude Code this regression is hidden because the harness auto-loads `.claude/agents/`, `.claude/commands/`, `.claude/skills/` directly — the AGENTS.md description is decorative, not load-bearing. In opencode (and codex/cursor/generic) AGENTS.md is the primary routing surface and the wrong paths actually mislead the model.

## Steps to Reproduce

```
mkdir /tmp/probe && cd /tmp/probe
hero init
hero install project . --target opencode
grep -A 8 "### Project Structure" AGENTS.md
# Before fix: lists commands/, agents/, skills/ at project root (don't exist).
# After fix: lists .hero/commands/, .hero/agents/, .hero/skills/ (the canonical paths).
```

## Expected Behavior

The Project Structure section must reflect the actual canonical content paths configured for the project. For a default install: `.hero/agents/`, `.hero/commands/`, `.hero/skills/`. For the hero-on-hero dogfood case (where `content.<kind>_path` overrides point at the source tree): `agents/`, `commands/`, `skills/`.

## Root Cause

`generateAgentsMdBody()` in `internal/install/agents_md.go` hard-coded `commands/`, `agents/`, `skills/` — the wording was tuned to the dogfood case where Hero's own repo uses `content.<kind>_path` overrides. When the single-source-install refactor (v0.8) made `.hero/{agents,commands,skills}/` the default canonical, the body text wasn't updated to match.

This is a regression of the v0.8 install refactor that wasn't caught because:
1. No test pinned the Project Structure wording against actual install output.
2. The repo's own dogfood install (the only one developers exercise locally) overrides to `agents/` and made the text look correct.
3. Claude Code's harness-level directory loading hides the wrong paths.

## Fix

Make the body resolve the actual canonical paths from the project's `hero.json` config:

- New helper `resolveContentPathsForBody(opts)` that calls existing `CanonicalDirs(targetDir, cfg)` and returns project-relative paths.
- `generateAgentsMdBody(paths contentPathsForBody)` takes the resolved paths and renders them in the Project Structure section.
- Both `installAgentsMd` and `installClaudeMd` resolve and pass the paths.
- Added a clarifying line: "Your harness may expose the agent/command/skill directories under its own prefix (`.claude/`, `.opencode/`, `.cursor/`, etc.) as symlinks back to the canonical paths above. Edit only the canonical files — harness directories are views."
- Also corrected `hero.json` → `.hero/hero.json` in the same list (it lives under `.hero/`, not at root).

## Changes

- `internal/install/agents_md.go`: added `contentPathsForBody` struct, `resolveContentPathsForBody`/`relForBody` helpers, new `generateAgentsMdBody(paths)` signature, updated Project Structure section.
- `internal/install/claude_md.go`: updated `installClaudeMd` to pass resolved paths.

## Verification

- `go test ./internal/install/...` passes.
- Fresh `hero init && hero install project . --target opencode` writes Project Structure with `.hero/agents/` etc.
- Hero-on-hero install (with `content.<kind>_path` overrides) writes Project Structure with `agents/` etc. (unchanged from previous behavior).

## Kickoff

Resume work on the AGENTS.md project-structure regression. Read this spec, the v0.8 install refactor commits (`git log --oneline | head -20`), and `internal/install/agents_md.go`. The fix is already in place; remaining work is (a) optional unit test pinning the resolved paths against fresh-install output, (b) follow-up for non-AGENTS.md surfaces that may carry the same hardcoded layout description (search for `commands/. — Slash` and similar wording across the repo).
