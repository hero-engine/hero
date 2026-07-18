---
user: chet-bellows
updated: 2026-07-18T03:31:06Z
repo: hero-engine/hero
---

# chet-bellows's handoff

## Last user ask

> hero code is now embedding us - the hero app - and is serving our agents / commands and skills out of our binary live and dynamically depending on the active domain - eng / code, PM, QA etc.. and it said the PM pack is very light - so we need to audit what we have for when a product manager is active - and make the inventory based off research of what commands / skills and agents we need (probably…

_possibly stale — 1 commit(s) since, last set 20m ago_

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

