---
user: chet-bellows
updated: 2026-07-16T23:36:58Z
repo: hero-engine/hero
---

# chet-bellows's handoff

## Session goal

> Session opened with — In /Users/bwheeler/projects/hero-engine/repository/hero, `.agents/skills/` is committed to git but is stale and drifting from its source. Decide how to resolve it. Do NOT assume gitignoring is the answer — see the policy distinction below.
> 
> FACTS (verified):
> - `.agents/skills/<name>/SKILL.md` is genuinely where the Codex CLI loads repo-scoped skills (verified against openai/codex `codex-rs/core-…

## Last user ask

> Deliver manifest-driven-prune: record a per-target file manifest (TargetState.Files in install-state.json) from the canonical render set on install/upgrade; prune files in the prior manifest but absent from the new render (product-removed), never touching non-manifest user files; nil manifest = no-op (covers fresh clone + first-run). Generalizes prune.go's skill-dir provenance model to agent/command/skill files across all six targets.

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

