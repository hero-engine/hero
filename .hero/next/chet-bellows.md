---
user: chet-bellows
updated: 2026-07-30T16:02:01Z
repo: hero-engine/hero
---

# chet-bellows's handoff

## Session goal

> Session opened with — so quick question on our cli here - for github connect - sample say - hero connect github \
>   --integration-id github-code-host \
>   --role code-host \
>   --project OWNER/REPOSITORY \
>   --token-stdin

_possibly stale — 2 commit(s) since, last set 1d 14h ago_

## Last user ask

> Spec and deliver the Jira attachment redirect fix from handoff mail_3ab437af53c66997e94a0268

## Suggested next prompt

> let's tackle Core / Vertical Layering — Make the Conceptual Split Physical

_Rationale: highest-priority open feature: Core / Vertical Layering — Make the Conceptual Split Physical (`core-vertical-layering`)_

_Source: auto-derived from open feature — `hero next suggest "..."` to override._

## Recent reflections

- Release rehearsal exposed Unix-only file locks after native tests passed. Added a shared build-tagged lock primitive, Windows cross-build CI, direct failure-path coverage, and completed a clean cold audit plus six-target snapshot verification.
- Tracker activity now has one provider-owned consumer contract: tracker_updated_at; import time, refresh time, and spec mtime are never substitutes.
- Tracker-backed diagnosis postback was never a hidden hook; it was an explicit agent closing step. Hero Code's 2026-07-15 hardening blocked all generic sync and exposed only import, making attach/comment unreachable while hero_diagnose also omitted the publish contract.
- Long session: shipped basic-chat + dormant research seed (chat-slim-to-basic-research-seed, on main), then root-caused and fixed the recurring 'hero why no node' bug (two-DB split + no graph.db self-heal + filepath.Base repoKey drift) — PR #2 open. Key stance to remember: core reliability (spec CRUD / install-copy / search) breaks weekly and needs a single self-healing store + write choke point + invariant tests. Chat/research/code content is Hero's call, not hero-code's; hero-code only consumes.
- Integration config should key connections by stable IDs, not provider names: provider-keyed objects cannot represent two Jira projects, while explicit default/roles separate definition from selection.

## Tried and failed (this session)

Nothing this session.

## Your recent activity

Run `git log --oneline --author=<you> -10` for recent commits.

