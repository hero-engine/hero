---
title: Harness Walk-Up Asymmetry — What Walks Up, What Doesn't
type: note
status: active
tags: [harness-integration, install, monorepo, claude-code, codex, opencode]
created: 2026-05-05
---

# Harness Walk-Up Asymmetry — What Walks Up, What Doesn't

When a chat (Claude Code, Codex, opencode, etc.) is opened in a subfolder of a project, the harness does **not** treat all configuration the same way. This asymmetry is the load-bearing constraint behind the monorepo satellite-installs design and is worth knowing for any future install/integration work.

## What walks up

Files the harness searches for by walking from cwd toward the filesystem root:

- `CLAUDE.md` (Claude Code), `AGENTS.md`, equivalent harness-specific top-level instruction files
- `.mcp.json` (and similarly-named MCP server registration files)

For these, opening a chat in `repo/sub/sub2/` will pick up `repo/CLAUDE.md` and `repo/.mcp.json` automatically.

## What does NOT walk up

Files and directories the harness loads from cwd only:

- `.claude/agents/` — subagent definitions
- `.claude/commands/` — slash commands
- `.claude/skills/` — skill definitions
- `.claude/settings.json`, `.claude/settings.local.json`
- The equivalent directories under `.codex/`, `.opencode/`, `.cursor/`, etc.

For these, opening a chat in `repo/sub/sub2/` will see *nothing* — even if `repo/.claude/agents/` is fully populated. The chat starts with whatever happens to live in `repo/sub/sub2/.claude/`, and if nothing lives there, the chat has no slash commands, no subagents, no skills.

## Why this matters

This asymmetry is the entire reason the satellite-installs spec exists. If everything walked up, a single root install would Just Work for any subfolder chat, and the only design question would be where specs land. Because agents/commands/skills don't walk up, opening a chat in a subfolder of a Hero workspace produces a **silently degraded session** — MCP works, but the rich harness layer is missing.

The satellite design responds to exactly this: symlink the directories that don't walk up; rely on walk-up for the ones that do. Specifically:

- Subdirectory symlinks for `agents/`, `commands/`, `skills/` (the don't-walk-up set, minus settings).
- No `.mcp.json` in satellites — walk-up handles it.
- Optional small `CLAUDE.md` marker in satellites — walk-up handles the *real* one at root, but the local marker tells the user inside the chat that they're in a satellite of a larger workspace.

## Don't-trust-without-checking

This was confirmed for Claude Code as of 2026-05. If a future harness changes its discovery rules (e.g., Codex starts walking up for agents too), the satellite design becomes partially redundant for that target — the implementation should detect this per-target rather than assume the asymmetry is universal forever.

The split between what-walks-up and what-doesn't is also a useful diagnostic when troubleshooting "my chat isn't using Hero":

1. Ask the user where they opened the chat (cwd vs. workspace root).
2. If cwd ≠ root: do they have a satellite there? (`.hero-satellite` file present?)
3. Does `walk_up(.mcp.json)` resolve? (cheap test: `ls $(git rev-parse --show-toplevel)/.mcp.json`)
4. Does `<cwd>/.claude/agents/` resolve to a populated directory or symlink?

If 3 succeeds and 4 fails, the user is in a degraded subfolder chat — that's a satellite-missing problem, not a Hero-broken problem.
