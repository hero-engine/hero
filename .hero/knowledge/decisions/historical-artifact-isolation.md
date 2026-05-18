---
title: Historical Artifacts Must Be Isolated From Default Discovery
type: decision
status: proposed
created: 2026-05-18
tags: [archive, history, search, isolation, principle, invariant]
relations:
  - target: project-snapshot
    kind: decided-in
---

# Historical Artifacts Must Be Isolated From Default Discovery

## Decision

Any Hero artifact that captures **point-in-time state** (snapshot
archives, frozen pulse rollups, milestone reports — anything
intentionally dated and immutable) must be **strictly isolated**
from default-discovery surfaces. Default-discovery surfaces include
`hero search` (without an explicit history flag), `/resume`,
`/prime`, `auto-knowledge-capture`, `note-capture`,
auto-memory, and default serve listings / homepages.

Historical content is reachable only through **explicit
history-querying surfaces**: a `history` subcommand, an explicit
MCP flag like `at: <date>` or `history: true`, a dedicated serve
route like `/project/snapshots/<date>`, or an opt-in
`--include-history` flag on search.

## Context

Snapshot archives were the first artifact that surfaced this risk.
A six-month-old archive row claiming `hero-serve: building`
appearing in `hero search` results would let the model treat
historical state as current — confidently telling the user the
wrong thing about where the project is.

This is the worst-case failure mode of any feature that captures
trajectory data. The discovery channels meant to surface relevant
context become the channels that poison context.

## Rationale

Two facts make isolation the right default:

1. **Default-discovery cannot distinguish historical from live by
   inspection.** A markdown file in a corpus looks like a markdown
   file. Unless the system explicitly filters, archives will
   surface alongside specs.
2. **Historical state is almost always wrong as a substitute for
   live state.** Users querying default-discovery surfaces want
   "what is true now" — that's the only safe interpretation.
   Historical state is useful but only when *explicitly* requested.

## Invariants

For every historical artifact:

1. **Search exclusion.** The artifact's directory is excluded from
   the default search index. Opt-in via an explicit flag
   (`--include-history`) is the only path that returns archive
   bodies.
2. **Frontmatter flags.** Every file carries `historical: true` and
   `not_current: true` so any future indexer / tool can filter
   explicitly without inspecting paths.
3. **In-body banner.** Every file body opens with a banner line
   declaring the historical date and pointing at the live
   equivalent. The writer refuses to emit the artifact without
   the banner.
4. **Capture-subsystem skip.** `auto-knowledge-capture`,
   `note-capture`, and auto-memory consult the same
   `exclude_paths` config and skip the historical directory.
5. **Cold-start skip.** `/resume` and `/prime` do not include
   historical content in cold-start bundles.
6. **Serve isolation.** Historical bodies render only on dedicated
   routes (`/<surface>/snapshots/<date>`) — never in listings,
   homepages, or relevance feeds.

These are *each* a verifiable invariant; the test plan asserts
each one independently.

## Tripwires

- A future skill imports archive bodies into a knowledge entry,
  promotion, or convention. Push back hard — knowledge captures
  *patterns*, not point-in-time observations.
- A future search feature ranks archive bodies into default
  results. Reject; require the `--include-history` opt-in.
- A "smart" cold-start skill auto-includes the most recent archive
  to "give context on recent change." Reject; the live snapshot
  already shows current state — historical comparison is a
  separate explicit query.

## Application beyond snapshot archives

The same isolation contract should apply to any future
point-in-time artifact:

- Pulse rollups frozen at week boundaries.
- Release-time state captures.
- Migration trajectory captures.
- Anything dated, immutable, and intentionally redundant with the
  live equivalent.

When designing such a feature, the spec must declare the isolation
invariants explicitly and list the explicit history-querying
surfaces.
