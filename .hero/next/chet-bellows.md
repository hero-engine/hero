---
user: chet-bellows
updated: 2026-07-17T16:14:40Z
repo: hero-engine/hero
---

# chet-bellows's handoff

## Last user ask

> hero code is now embedding us - the hero app - and is serving our agents / commands and skills out of our binary live and dynamically depending on the active domain - eng / code, PM, QA etc.. and it said the PM pack is very light - so we need to audit what we have for when a product manager is active - and make the inventory based off research of what commands / skills and agents we need (probably…

## Suggested next prompt

> doctor-install-target-table delivered + verified on branch feat/doctor-install-target-table (5 commits ahead of origin/main, not merged). Next: merge to main + release if desired (prior pattern: --no-ff merge, tag patch bump v0.26.5, push tag triggers goreleaser). Housekeeping: .build/ and .hero/index.db-shm/-wal are untracked but NOT gitignored.

## Recent reflections

- Integration config should key connections by stable IDs, not provider names: provider-keyed objects cannot represent two Jira projects, while explicit default/roles separate definition from selection.
- Jira import already supports tracker.token in hero.local.json; the token_env-only error is false, and local connect can copy the resolved token into committed hero.json.
- mkdocs-material's theme feature for integrated left-nav TOC is toc.integrate, not navigation.integrate (that flag doesn't exist in 9.7.x) — check the compiled template partials before trusting a spec's exact feature-flag name

## Tried and failed (this session)

Nothing this session.

## Your recent activity

Run `git log --oneline --author=<you> -10` for recent commits.

