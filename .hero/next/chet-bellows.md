---
user: chet-bellows
updated: 2026-08-23T19:27:12Z
repo: hero-engine/hero
---

# chet-bellows's handoff

## Session goal

> Ship a verified Hero release containing the completed MCP tool metadata contract for hero-code while keeping the superseded interactive CLI donor branch evidence-only.

_possibly stale — 20 commit(s) since, last set 9d 3h ago_

## Last user ask

> Deliver all v0.34 public-release prep: refresh docs and site around Hero's memory-first two-system story, ready the repository and release candidate, and stop before final license, visibility, and publication mutations.

_possibly stale — 4 commit(s) since, last set 3h 39m ago_

## Suggested next prompt

> let's tackle Core / Vertical Layering — Make the Conceptual Split Physical

_Rationale: highest-priority open feature: Core / Vertical Layering — Make the Conceptual Split Physical (`core-vertical-layering`)_

_Source: auto-derived from open feature — `hero next suggest "..."` to override._

## Recent reflections

- Repo-scoped graph identity must propagate into every derived index; node_index kept the old two-column uniqueness after graph v5 and made valid sibling keys fatal.
- Grok Build 1.0.4 inspect --json normalizes AGENTS.md as Agents.md; qualify native loading by source type and .grok path suffix, not display casing.
- Release rehearsal exposed Unix-only file locks after native tests passed. Added a shared build-tagged lock primitive, Windows cross-build CI, direct failure-path coverage, and completed a clean cold audit plus six-target snapshot verification.
- Tracker activity now has one provider-owned consumer contract: tracker_updated_at; import time, refresh time, and spec mtime are never substitutes.
- Tracker-backed diagnosis postback was never a hidden hook; it was an explicit agent closing step. Hero Code's 2026-07-15 hardening blocked all generic sync and exposed only import, making attach/comment unreachable while hero_diagnose also omitted the publish contract.

## Tried and failed (this session)

Nothing this session.

## Your recent activity

Run `git log --oneline --author=<you> -10` for recent commits.

