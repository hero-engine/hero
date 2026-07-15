---
user: chet-bellows
updated: 2026-07-15T17:58:13Z
repo: hero-engine/hero
---

# chet-bellows's handoff

## Last user ask

> In /Users/bwheeler/projects/hero-engine/repository/hero, two acceptance-criteria consumers are mutually exclusive. Verified by direct test:
> 
> | AC form | ClassifyCriterion | ParseAcceptanceCriteria |
> |---|---|---|
> | `- **AC-1:** WHEN a user clicks THE SYSTEM SHALL respond` | freeform | 1 criterion |
> | `- WHEN a user clicks THE SYSTEM SHALL respond` | event | 0 criteria |
> 
> Cause: `ClassifyCriterion`…

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

