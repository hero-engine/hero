---
title: Competitor Parity — Borrow the Best Ideas from Competing Tools
slug: kiro-parity
type: initiative
status: completed
tags: [specs, drift, hooks, ears, parity, competitor]
created: 2026-04-22
child:
  - ears-acceptance-criteria
  - spec-three-file-split
  - spec-drift-detection
  - nl-event-hooks
  - delivery-mode-flags
horizon: now
---

## Goal

Selectively adopt the strongest ideas from competitors's spec-driven IDE without
abandoning Hero's tool-agnostic, multi-agent, tracker-integrated stance. Hero
already leads on orchestration, code-health, sprint/tracker integration, and
multi-tool support. competitor leads on three things worth importing: a tighter spec
shape (EARS criteria, three-file split), live spec↔code drift awareness, and
event-driven natural-language hooks.

## Why now

The competitor/Hero comparison conducted on 2026-04-22 surfaced five borrowable ideas.
Two are high-ROI because they compose with existing Hero strengths:

1. **EARS** plugs directly into `hero test generate` — heuristic mapping gets
   dramatically more reliable when criteria use a fixed grammar.
2. **`hero drift`** plays to the SQLite FTS5 + spec corpus we already maintain
   and gives Hero a story competitor can't quite tell: drift detection across an
   *initiative*, not just one file.

The other three (three-file split, NL hooks, delivery mode flags) are smaller
quality-of-life wins worth doing in the same sweep so the spec shape and
delivery surface stay coherent.

## Non-goals

- Building a Hero IDE — Hero stays a CLI + MCP server that augments OpenCode,
  Cursor, and Claude Code.
- In-editor diagnostics or chat UI — host tools own that loop.
- Bedrock or AWS-specific integrations.

## Children

| Spec | Summary | Priority |
|---|---|---|
| `ears-acceptance-criteria` | EARS grammar option in spec template; parser support; `hero test generate` upgrade | P0 |
| `spec-drift-detection` | `hero drift` command + MCP tool that flags code-vs-spec divergence | P0 |
| `spec-three-file-split` | Optional `requirements.md` / `design.md` / `tasks.md` layout per spec | P1 |
| `nl-event-hooks` | `.hero/hooks/` dir of natural-language hooks installed into host tool's hook system | P1 |
| `delivery-mode-flags` | `--autopilot` / `--supervised` / `--dry-run` flags on `/deliver` | P2 |

## Sequencing

1. `ears-acceptance-criteria` first — unblocks better drift heuristics and
   better test generation. Small surface area.
2. `spec-drift-detection` second — biggest user-visible win, depends on the
   tighter criteria grammar.
3. `spec-three-file-split` third — cosmetic until 1+2 land; reorganizing too
   early would force re-work.
4. `nl-event-hooks` and `delivery-mode-flags` in parallel after that — they
   touch different surfaces (host-tool hooks vs CLI flag plumbing).

## Boundaries

- Backwards-compatible: existing single-file specs, freeform criteria, and
  current `/deliver` behavior all keep working without any flags.
- No new external dependencies.
- No LLM calls in `hero drift` — heuristic + structural comparison only,
  matching the same posture as `hero ask` and `hero test generate`.
