---
title: Discoverable, Not Pushed — Heavy Artifacts Are Pulled, Never Injected
type: decision
status: proposed
created: 2026-05-18
tags: [context, injection, discoverability, artifacts, principle]
relations:
  - target: project-snapshot
    kind: decided-in
  - target: next-as-projection
    kind: relates-to
---

# Discoverable, Not Pushed — Heavy Artifacts Are Pulled, Never Injected

## Decision

Heavy Hero artifacts (project snapshot, knowledge rollups, archive
trajectories, anything substantially larger than a session-handoff
summary) are **discoverable** rather than **pushed**. The projector
writes a single-line pointer in NEXT.md / AGENTS.md that says where
the artifact lives. Sessions, skills, and tools *pull* the artifact
when they decide it's relevant. The projector never injects artifact
bytes into session context.

## Context

`NEXT.md` is light enough that every session reads it on every turn
— so it's pushed. That worked, and the natural impulse on the next
artifact (project-snapshot) was to repeat the pattern: opt-in
auto-injection with a v2 flip to default-on.

The user pushed back: snapshot is structurally different. It's
heavier; most sessions don't need the full surface rollup; and
forcing it into context burns tokens on most sessions to serve a
minority of cases.

## Rationale

Three things make "discoverable, not pushed" the right pattern for
heavy artifacts:

1. **Cost asymmetry.** Push pays the read+inject cost on every
   session. Pull pays it only when the artifact is actually needed.
   For artifacts most sessions don't need, push is the wrong
   default — and "opt in to push" is a knob users don't tune.
2. **Discovery is cheap.** A one-line pointer in a file the session
   already reads (NEXT.md, AGENTS.md) gives the model the artifact's
   existence and location at near-zero token cost. The model can
   then choose to read it.
3. **Explicit-pull moments exist.** `/resume`, `/prime`, MCP tool
   calls, direct file opens are all explicit moments the user or
   model can opt into. The artifact stays one call away — that's
   the whole pull surface.

## Implementation

For project-snapshot:

- The projector writes
  `Project shape: see [SNAPSHOT.md](.hero/SNAPSHOT.md).`
  into NEXT.md and AGENTS.md (idempotent, inserted once).
- No `auto_context.include_snapshot` config field.
- `/resume` and `/prime` *may optionally* pull the snapshot into
  their cold-start bundles; default behavior unchanged.
- `hero_snapshot` MCP tool is the programmatic pull.

The general rule: **if an artifact is heavier than ~1KB and most
sessions don't need it, the projector writes a pointer; consumers
pull.** Reserve push for artifacts every session needs (Mission,
NEXT).

## Tripwires

- A future feature proposes auto-injecting any artifact other than
  Mission / NEXT into session context by default. Push back: ask
  whether the pointer pattern covers it.
- A feature adds an `include_X: true` config to inject artifact X.
  Push back: opt-in injection is a knob users don't tune; build the
  pull path instead.

## Open Questions

- Whether the pointer line should aggregate ("Project shape:
  SNAPSHOT.md · Pulse: PULSE.md · …") or stay one-artifact-per-line.
  Currently one line per artifact; revisit once we have a second
  pointer to write.
