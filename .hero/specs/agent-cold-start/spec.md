---
title: Agent Cold-Start — `.hero/NEXT.md` Briefing for Fresh Sessions
slug: agent-cold-start
type: feature
status: completed
tags: [agents, prime, cold-start, dx, skill]
created: 2026-04-22
relations:
  - target: competitor-parity
    kind: related
horizon: now
---

## Goal

Make a fresh agent session — or a session resuming after compaction — pick
up exactly where the last one left off, without re-reading the chat. Ship
this as agent-instructions-and-a-skill, not as new infrastructure: the
agent reads and writes `.hero/NEXT.md` directly per AGENTS.md and a new
`next-md` skill. A small `hero next show` command exists for humans to
peek from the terminal.

## Problem

After a few weeks of dogfooding, the biggest agent ergonomics gap is
continuity. A new session has no idea what was happening yesterday. Either
the user re-explains, or the agent re-derives state from git log + open
files + recent specs — and even then misses non-obvious decisions ("we
agreed not to touch X", "EARS criteria are preferred but freeform is
fine"). Compaction makes it worse: mid-session memory snaps to a summary
that loses load-bearing detail.

## Why this is mostly markdown, not code

A full Go implementation (CLI for set/clear, schema flags, auto-injection
into `hero context` and `hero pulse`, staleness state) was prototyped and
discarded. Findings:

- **A noob-prompter user never calls `hero next set`** — Hero exists to
  make non-experts productive with agents, not to give agents more CLI
  surface. The user types "fix this bug"; the agent does the work.
- **Agent direct file I/O is more reliable than CLI plumbing** — no shell
  quoting, no missing flags, no schema drift between flags and what the
  agent actually wants to write.
- **AGENTS.md and skills auto-load already** — every supported host tool
  reads them at session start. That's the auto-pickup mechanism. No code
  required.
- **Auto-injection in `hero context` / `hero pulse` was speculative** —
  agents that need NEXT will just read the file at session start (per
  AGENTS.md). Humans peek with `hero next show` or `cat`.

The complex version was Opus engineering for itself. The simple version
matches Hero's actual user.

## Design

### Agent contract — added to `AGENTS.md`

A short section:

> **Keep `.hero/NEXT.md` current.**
>
> *At session start:* if `.hero/NEXT.md` exists, read it before doing
> anything else and surface the contents to the user.
>
> *At end of a turn where meaningful work happened* — finished a spec
> section, landed a code change, made a design decision, or chose what to
> do next — overwrite `.hero/NEXT.md` with a fresh briefing. Always
> overwrite, never append. Skip when the turn was purely conversational.
>
> See the `next-md` skill for the format and quality bar.

That's the entire instruction surface. Auto-pickup is via AGENTS.md being
auto-loaded by Claude Code, OpenCode, and Cursor at session start.

### Format spec — `skills/next-md.md`

A new skill that owns the format definition. Three sections:

- **Just finished** — what shipped, where artifacts live, uncommitted
  state. Lets the fresh session avoid redoing work.
- **Next** — one short paragraph plus a `→ <pointer>` line. Pointer is a
  runnable command, spec path, or shell command.
- **Context to carry forward** — non-obvious decisions, user preferences,
  in-flight initiatives, gotchas. Omitted entirely when there's nothing
  meaningful to add.

Frontmatter: `updated` (RFC3339 UTC), `session` (host-tool/model-id),
`branch` (git branch). Target 15–40 lines, hard cap 60. Always overwrite.

### Human-facing CLI — `hero next show`

One command. Prints `.hero/NEXT.md` to stdout. If the file doesn't exist,
prints a one-line hint pointing at the skill. That's it — no `set`, no
`clear`, no flags. Humans don't write NEXT.md; the agent does.

### Prime command

`commands/prime.md` is updated so `/prime` reads `.hero/NEXT.md` first and
surfaces it to the user before any other priming output.

### What's explicitly NOT shipping

- No `hero next set` — the agent uses its existing file-write tool
- No `hero next clear` — `rm .hero/NEXT.md` works fine
- No `internal/heronext/` Go package — file I/O doesn't need a library
- No `NextConfig` in `hero.json` — there's nothing to configure
- No staleness logic in code — the agent eyeballs the `updated:` field
- No auto-injection into `hero context` or `hero pulse` — speculative;
  the agent reads NEXT.md directly per AGENTS.md

## Changes

- `AGENTS.md` — append the "Keep `.hero/NEXT.md` current" section
- `skills/next-md.md` — new skill defining the format and quality bar
- `commands/prime.md` — read `.hero/NEXT.md` first
- `internal/cli/next.go` — minimal `hero next` (alias for `hero next show`) printing the file

## Acceptance Criteria

- WHEN `hero next show` runs and `.hero/NEXT.md` exists THE SYSTEM SHALL print the file contents to stdout
- WHEN `hero next show` runs and `.hero/NEXT.md` does not exist THE SYSTEM SHALL print a one-line hint pointing at the `next-md` skill and exit 0
- THE SYSTEM SHALL ship a `next-md` skill in `skills/` that defines the three-section format, frontmatter fields, and quality bar
- THE SYSTEM SHALL update `AGENTS.md` with explicit instructions for reading `.hero/NEXT.md` at session start and overwriting it at end of meaningful turns
- THE SYSTEM SHALL update `commands/prime.md` so `/prime` surfaces `.hero/NEXT.md` ahead of other priming output
- THE SYSTEM SHALL not add any new `hero.json` configuration keys for this feature
- THE SYSTEM SHALL not introduce a new internal Go package — agent direct file I/O is the contract

## Boundaries

- Does **not** add `hero next set` or `hero next clear` — agent file I/O suffices
- Does **not** auto-inject NEXT.md into `hero context` or `hero pulse` output
- Does **not** define staleness or hide-after windows in code
- Does **not** persist NEXT history — git log + `hero session` cover history
- Does **not** add a hook, daemon, or watcher — capture is by agent convention
