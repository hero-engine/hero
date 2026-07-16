---
user: chet-bellows
updated: 2026-07-16T07:36:50Z
repo: hero-engine/hero
---

# chet-bellows's handoff

## Session goal

> Session opened with — In /Users/developer/projects/hero-engine/repository/hero, `.agents/skills/` is committed to git but is stale and drifting from its source. Decide how to resolve it. Do NOT assume gitignoring is the answer — see the policy distinction below.
> 
> FACTS (verified):
> - `.agents/skills/<name>/SKILL.md` is genuinely where the Codex CLI loads repo-scoped skills (verified against openai/codex `codex-rs/core-…

_possibly stale — 5 commit(s) since, last set 10h 56m ago_

## Last user ask

> Deliver codex-mcp-binary-path-resolution: findHeroBinary → os.Executable with surfaced errors; Codex MCP block moves to ~/.codex/config.toml User layer on all installs; migration strips Hero's managed block from project .codex/config.toml without touching user content.

## Suggested next prompt

> Next up: deliver codex-mcp-binary-path-resolution — direction is decided in the spec (MCP block to ~/.codex/config.toml User layer, os.Executable resolver, remove Hero's block from project config.toml). Last open item from today's install-integrity thread. Also two small bug specs worth filing when convenient: detectOrphanInstructionFiles fresh-clone false positive (chip task_6c7a7f41 already pending), and copilot being un-inferable on fresh clones (legacy .github/copilot probe in targetLayouts).

## Recent reflections

- Integration config should key connections by stable IDs, not provider names: provider-keyed objects cannot represent two Jira projects, while explicit default/roles separate definition from selection.
- Jira import already supports tracker.token in hero.local.json; the token_env-only error is false, and local connect can copy the resolved token into committed hero.json.
- mkdocs-material's theme feature for integrated left-nav TOC is toc.integrate, not navigation.integrate (that flag doesn't exist in 9.7.x) — check the compiled template partials before trusting a spec's exact feature-flag name

## Tried and failed (this session)

Nothing this session.

## Your recent activity

Run `git log --oneline --author=<you> -10` for recent commits.

