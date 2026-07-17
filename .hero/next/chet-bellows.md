---
user: chet-bellows
updated: 2026-07-17T00:39:30Z
repo: hero-engine/hero
---

# chet-bellows's handoff

## Session goal

> Session opened with — In /Users/developer/projects/hero-engine/repository/hero, `.agents/skills/` is committed to git but is stale and drifting from its source. Decide how to resolve it. Do NOT assume gitignoring is the answer — see the policy distinction below.
> 
> FACTS (verified):
> - `.agents/skills/<name>/SKILL.md` is genuinely where the Codex CLI loads repo-scoped skills (verified against openai/codex `codex-rs/core-…

_possibly stale — 3 commit(s) since, last set 1h 34m ago_

## Last user ask

> Deliver codex-agents-wholesale-wipe fix: scope removeLegacyDir with a *.md-only predicate so the .codex/agents cleanup removes only legacy .md dead-bytes, leaving user files and Hero .toml (owned by render + pruneStaleFiles). Other call sites pass nil (allow-all, unchanged).

_possibly stale — 1 commit(s) since, last set 15m ago_

## Suggested next prompt

> let's tackle Core / Vertical Layering — Make the Conceptual Split Physical

_Rationale: highest-priority open feature: Core / Vertical Layering — Make the Conceptual Split Physical (`core-vertical-layering`)_

_Source: auto-derived from open feature — `hero next suggest "..."` to override._

## Recent reflections

- Integration config should key connections by stable IDs, not provider names: provider-keyed objects cannot represent two Jira projects, while explicit default/roles separate definition from selection.
- Jira import already supports tracker.token in hero.local.json; the token_env-only error is false, and local connect can copy the resolved token into committed hero.json.
- mkdocs-material's theme feature for integrated left-nav TOC is toc.integrate, not navigation.integrate (that flag doesn't exist in 9.7.x) — check the compiled template partials before trusting a spec's exact feature-flag name

## Tried and failed (this session)

Nothing this session.

## Your recent activity

Run `git log --oneline --author=<you> -10` for recent commits.

