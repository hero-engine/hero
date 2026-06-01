---
title: Size-Drift Actionable Output — Inline Next-Step and Dedupe Duplicate Error
slug: size-drift-actionable-output
type: feature
domain: engineering
status: planning
size: trivial
priority: P2
tags: [size, drift, cli, polish]
created: 2026-06-01
relations:
  - target: roadmap-shape
    kind: parent
---

## Goal

Make `hero size --check` drift output actionable on first read: each
drift row gets the inline next-step command appended ("declared `medium`,
computed `large` → run `hero size foo large`"), and the duplicate
"size drift" error that surfaces twice in the `hero_warnings` MCP path
gets deduped.

## Context

This is the polish/trivial child in the [`roadmap-shape`](../../initiatives/roadmap-shape/spec.md)
initiative. It ships in phase 2 alongside `multi-spec-design-routing`
because it's small and independent of #1, but it should land before
the surfacing child (#2) so the `hero size --check` row format is
stable if #2 ever references row text.

## Kickoff

Tiny polish: inline the fix command on `hero size --check` drift rows
and dedupe the duplicate `hero_warnings` error.

**Status:** planning — stub only; needs full `/design` pass.

**Pick up at:** run `/design size-drift-actionable-output` to produce
the full spec. Two surgical changes; no schema work.

→ `/design size-drift-actionable-output`

**Files:** `internal/cli/size.go`, `internal/mcp/warnings.go`
**Skip:** new check categories, restructuring `--check` output schema.

## Scope-creep watch

Two surgical changes only: **inline next-step command on drift rows**
and **dedupe the duplicate `hero_warnings` error**. Resist adding new
check categories, restructuring the `--check` output schema, changing
the JSON contract for `hero size --check`, or "while we're in here"
refactors of nearby drift-detection code. If the design conversation
starts proposing schema changes, that's a different spec.

## Notes for design

Decisions already made in the composition session that `/design`
should honor:

- **Two changes, no more.** Inline next-step on rows; dedupe duplicate
  error. Don't expand.
- **No schema changes to `hero size --check` output.** Row format may
  grow a column, but the JSON contract stays compatible — `hero_warnings`
  and any future ambient-surfacing consumer must keep working.
- **Inline command format** should match the phrasing used in the
  `spec-sizing` skill's "what to do on drift" section ("`hero size <slug>
  <new-tier>`"). Don't reinvent the wording.
- **The duplicate error in `hero_warnings`** is the same drift surfaced
  via two code paths joining at the warnings consumer. Fix at the
  collation point, not by silencing one of the producers.
- **Surfacing layer (#2) may consume this output.** Keep the row format
  stable and concise; #2's recommended posture is count-only, but if it
  ever pulls a row excerpt, the text needs to read well in a NEXT.md
  briefing.
