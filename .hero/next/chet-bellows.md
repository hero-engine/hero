---
user: chet-bellows
updated: 2026-08-23T15:47:20Z
repo: hero-engine/hero
---

# chet-bellows's handoff

## Session goal

> Ship a verified Hero release containing the completed MCP tool metadata contract for hero-code while keeping the superseded interactive CLI donor branch evidence-only.

_possibly stale — 14 commit(s) since, last set 8d 23h ago_

## Last user ask

> Deliver the public PM pack, public QA pack, and dual-mode setup in order, then run a full test build and smoke-test PM/QA project initialization.

_possibly stale — 2 commit(s) since, last set 17h 40m ago_

## Suggested next prompt

> Refresh public docs and setup guidance for dual-mode PM/QA packs

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

