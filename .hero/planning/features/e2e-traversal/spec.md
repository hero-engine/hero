---
title: E2E Traversal Suite — Graph Queries Beat Grep
type: feature
status: delivering
priority: P0
tags: [e2e, traversal, smoke, ac-backed, v2-showcase]
created: 2026-04-29
relations:
  - target: e2e-area-suites
    kind: parent
  - target: acceptance-criteria-graph
    kind: depends-on
  - target: traversal-queries
    kind: depends-on
mission_alignment: |
  Traversal is the v2 unlock — the entire reason the graph substrate
  exists. `hero why`, `hero blocked`, `hero impact`, `hero relevant`
  are the showcase queries that beat grep + git-log archaeology. If
  any of them silently no-op or return broken output, the v2 promise
  collapses on first use. This suite catches regressions before they
  reach a session.
principles_check: |
  Serves #1 (verbs work without ceremony — every AC is "command runs,
  output is non-trivial") and #3 (graph queries return omniscient
  context). Catches the silent-no-op class directly — a `hero why`
  that exits 0 with empty stdout passes a naïve smoke but fails the
  byte-count assertion below.
horizon: now
smoke: deferred
---

## Goal

Smoke the six traversal verbs: `why`, `blocked`, `impact`, `relevant`,
`suggest`, `check conflicts`. Each AC is a single command-level smoke
that catches both hard failures (non-zero exit) and the silent-no-op
class (zero exit, empty output).

Runs against the outer hero repo itself — the graph has substantial
content (200+ commits, 75+ Criterion nodes, real depends_on edges)
so the queries have meaningful surface to traverse. Onboarding-style
sandbox doesn't apply here: traversal in an empty graph is uninformative.

## Why now

The traversal-queries spec shipped phases 1–5 in this recovery sweep,
but until now nothing was watching them for drift. A regression in
`hero why` would land silently between manual exercises. This suite
runs every CI pass (when wired) and on demand.

## Suite shape

- Script: `scripts/e2e/traversal.sh` (sources `scripts/e2e/lib.sh`)
- Target repo: outer hero repo (REPO_ROOT). No sandbox — needs the
  graph populated by everyday work.
- Each step exercises one verb with a known-good input.
- Records `results.json` per the
  [`acceptance-criteria-graph`](../../features/acceptance-criteria-graph/spec.md)
  schema; pass `--record` to ingest into the graph.

## Acceptance criteria

**AC-1:** `hero why <feature-slug>` on a known feature returns the
origin chain non-empty. Verifies the `traversal.Why` recursive CTE
walks origin edges. Test target: `next-as-projection` (recently
shipped, has Feature → Initiative parent edges).

**AC-2:** `hero why <feature:AC-N>` on a known AC key returns the
chain showing the parent feature plus any `satisfied_by` commit.
Verifies AC-id resolution + the participation join. Test target:
`acceptance-criteria-graph:AC-3` (we just ingested its
`satisfied_by` edge in this session).

**AC-3:** `hero blocked` exits 0 with structured output (the literal
"Nothing." or one or more `← waiting on` rows). Catches the
silent-empty regression where `hero blocked` runs but emits 0 bytes.

**AC-4:** `hero relevant <file>` runs, exits 0, prints non-trivial
output. Test target: a file we know is wired into specs via
`participates_in` edges (e.g. `internal/cli/checkpoint.go`).

**AC-5:** `hero impact <file>` runs without error. Coverage smoke
only — depth of analysis verified by separate impact-specific tests.
Same test target as AC-4.

**AC-6:** `hero suggest` runs without error and emits non-zero output
(or the canonical "no high-churn files" message when the heuristic
finds nothing).

**AC-7:** `hero check conflicts` runs without error.

**AC-8:** `hero ac list e2e-traversal --json` returns a JSON array
that includes `AC-1` — proves this spec's own ACs were ingested by
`hero scan` (mirrors onboarding's AC-4 self-check).

ACs accrete as runs surface "the verb worked but the result was
useless" gaps.

## Out of scope

- Performance budgets (covered by `TestWhy_DepthFourUnder200ms` in
  `internal/traversal`)
- Cross-repo traversal (separate suite when `unified-retrieval-layer`
  ships)
- LLM-narrated explanations of why-chains (Tier-3, future)

## Open questions

- Should AC-3 also smoke `hero blocked --json` to catch JSON-format
  regressions? Lean: add as AC-9 once the framework proves stable.
- Test targets in AC-1/AC-2 are recent specs; if the recency heuristic
  bit-rots (e.g. spec gets renamed), the suite breaks brittlely.
  Mitigation: hard-pinned targets with a comment explaining what to
  re-pin if they ever go away.
