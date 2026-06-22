---
title: Spec slug-resolution is fragmented across CLI commands
type: decision
created: 2026-06-21
tags: [architecture, cli, slug-resolution, verify, reuse]
---

# Spec slug-resolution is fragmented across CLI commands

Hero's slug space is **flat**: `spec.Discover(heroDir)` walks the whole `.hero`
tree via `filepath.Walk`, picking up any `spec.md` (or three-file dir) at any
depth, and `slugFromPath` slugs from the parent dir name (frontmatter `slug:`
overrides). A spec being a child of an initiative is irrelevant to resolution —
any materialized child `spec.md` is findable by its slug. There is **no
"initiative-child slug" concept** that any command rejects.

## The fragmentation

Multiple commands resolve a spec from a CLI arg, but each does it differently —
no shared helper:

- `hero spec verify` (`verify.go:86-95`) — resolves by **flat slug** against
  `Discover`, exact + case-sensitive match, then bare `spec %q not found`.
- `hero complete` (`complete.go:45-67`) — **path-based**, NOT slug-based. Takes
  a `<spec-path>`, `ParseFile`s it, and redirects work specs to
  `hero spec verify`. Do not "harden its slug resolution" — it has none by
  contract.
- `size.go:loadSpecBySlug`, `drift.go:findSpecBySlugOrPath`,
  `claim.go:findSpecBySlug` — each hand-rolls its own slug resolution.

## Why it matters

A bare exact-match-or-"not found" gives agents zero diagnostic signal. In a real
incident, a downstream project's model passed an unmaterialized initiative-child
slug (`R-01` — existed only as a `## Children` table row, never designed into a
spec), got "not found", and **confabulated a phantom tool limitation** ("verify
doesn't support initiative-child slugs") to rationalize skipping the lifecycle.
The fix (spec `verify-slug-resolution-hints`) makes the not-found branch
helpful: case-insensitive resolve → initiative-child detection → fuzzy "did you
mean" → unchanged fallback.

The natural future consolidation is a shared `internal/spec` resolve-or-hint
helper adopted by the genuinely slug-based commands above — but NOT `complete`,
whose path-based contract is deliberate. Relates to
[[spec-lifecycle-last-mile-gap]].
