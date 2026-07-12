---
user: chet-bellows
updated: 2026-07-12T19:59:52Z
repo: hero-engine/hero
---

# chet-bellows's handoff

## Last user ask

> Investigate and fix a data-loss bug in the incremental code-scan knowledge writer.
> 
> ## Symptom
> After an incremental `hero scan`, `.hero/knowledge/code/` contains ONLY the packages whose files changed since the previous scan — every unchanged package's `spec.md` directory has been deleted. On this repo, `ls .hero/knowledge/code/` shows just `index/`, `internal-projection/`, `internal-serve-opsrun…

## Suggested next prompt

> Harness-native install/upgrade is delivered + verified (bug resolved). Next: commit the batch (install code + governance spec edits) to main; then re-send the hero-code peer advisory when auth works; optionally decide if hero init should skip AGENTS.md for claude-only repos. Then back to remaining pre-release bugs.

## Recent reflections

_(none yet)_

## Tried and failed (this session)

Nothing this session.

## Your recent activity

Run `git log --oneline --author=<you> -10` for recent commits.

