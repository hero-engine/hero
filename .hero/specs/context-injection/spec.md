---
title: Context Injection — Relevant Conventions and Decisions Before Every Delivery
slug: context-injection
type: feature
status: completed
tags: [context, conventions, delivery, agents]
created: 2026-04-12
relations:
  - target: hero-v2-system-design
    kind: parent
horizon: now
completed_at: 2026-05-18T19:25:38Z
---

## Goal

Before an agent writes code, Hero searches the spec corpus and injects relevant context: conventions that apply to the files being touched, past specs in the same area, active decisions, and known bug patterns. This is the highest-impact v2 feature — it ensures agents never work blind.

## What Was Built

**`hero context` CLI** — given a spec path or list of files, produces a structured context block: matching conventions, past specs in those files, applicable decisions, known risks.

**`hero context imports --files`** — file-based lookup without a spec. Used by delivery leads before delegating to the engineer agent.

**`context-injection` skill** — loaded by delivery leads, documents how to use `hero context` output and feed it into specialist agent instructions.

**Engineer agent updated** — `agents/engineer.md` now expects a context block and follows conventions listed in it.

**Delivery leads updated** — `feature-delivery-lead` and `platform-delivery-lead` run context injection before delegating.

## Changes

- `internal/cli/context.go` — `hero context` CLI command
- `internal/index/index.go` — file-based convention scope matching, files_touched lookup
- `skills/context-injection.md` — context injection skill for delivery leads
- `agents/engineer.md` — updated to follow context injection block
- `agents/feature-delivery-lead.md` — runs `hero context` before delegating
- `agents/platform-delivery-lead.md` — runs `hero context` before delegating
