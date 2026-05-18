---
title: E2E Onboarding Suite — Fresh Repo to Productive Workspace
slug: e2e-onboarding
type: feature
status: planning
priority: P0
tags: [e2e, onboarding, smoke, ac-backed]
created: 2026-04-28
relations:
  - target: e2e-area-suites
    kind: parent
  - target: acceptance-criteria-graph
    kind: depends-on
mission_alignment: |
  The first thing a new user does with Hero is `hero init` followed by
  `hero scan`. If those don't produce a populated, queryable workspace
  on their own, every later mission promise (sessions start omniscient,
  the corpus compounds) collapses at the first step. This suite is the
  gate that proves a fresh repo becomes a productive workspace without
  the user reading documentation.
principles_check: |
  Serves #1 directly — every AC is "did the verb succeed without the
  user knowing what to expect?" Catches the silent-no-op regressions
  the v2 audit found (e.g. `hero ask` returning 0 bytes pre-polish).
horizon: next
smoke: deferred
---

## Goal

Smoke the onboarding flow on a clean repo: `init` creates the
workspace, `scan` populates the graph, `status` summarizes what's
there, and `ac list` proves Criterion ingest works. Each AC is a
single command-level smoke; together they answer "does a fresh user
get a working workspace by running the canonical commands once?"

## Why now

`scripts/e2e_smoke.sh` exercises a much larger flow (init → scan →
ask → relevant → suggest → design …). That's still useful as the
integration pass, but it doesn't isolate which step regressed when
something breaks. This suite isolates onboarding so any regression
in `init` or `scan` shows up as a single failing AC, not a "smoke
passed/failed" boolean.

## Suite shape

- Script: `scripts/e2e/onboarding.sh` (sources `scripts/e2e/lib.sh`)
- Target repo: `tmp/e2e/onboarding-<ts>/sandbox/` — a freshly
  initialized empty git repo so we control all state. Self-test
  against this repo itself stays a separate pass.
- Each step runs from inside the sandbox.
- Records `results.json` per the
  [`acceptance-criteria-graph`](../../features/acceptance-criteria-graph/spec.md)
  schema; pass `--record` to ingest into the graph.

## Acceptance criteria

**AC-1:** ✅ **passing** (commit `29db555`, 2026-04-28).
`hero init` on a clean repo exits 0 and creates `.hero/`
with `hero.json` plus an `AGENTS.md` at repo root. (`hero init`
intentionally does NOT install the agent file pack — that's
`hero install`'s job; this AC stays scoped to the workspace
bootstrap.)

**AC-2:** ✅ **passing** (commit `29db555`, 2026-04-28). `hero scan` on the initialized repo runs to completion
(exit 0) and emits a "Graph ingest summary" block — verifiable by
grepping the captured stdout for the literal string.

**AC-3:** ✅ **passing** (commit `29db555`, 2026-04-28). `hero status` on the scanned repo prints something —
non-zero stdout output. Catches the "silent no-op" regression
class.

**AC-4:** ✅ **passing** (commit `29db555`, 2026-04-28). `hero ac list e2e-onboarding --json` returns a JSON array
with at least 1 entry whose `ac_id` matches `AC-1`. Proves the
scan-time AC ingest is wired (this spec was scanned, its ACs
became Criterion nodes).

**AC-5:** ✅ **passing** (commit `29db555`, 2026-04-28). Re-running `hero scan` produces no new graph nodes
(idempotency smoke). Captured by comparing `hero graph stats` node
totals before/after.

ACs accrete as runs surface "the flow worked but the result wasn't
useful" gaps.

## Out of scope

- Code-scan depth tuning / language-specific behavior (covered by
  e2e-ingest area suite — Phase 4)
- Multi-repo `repos:` config testing — separate
- Tier-2 LLM extraction smoke (needs API key — separate suite)

## Open questions

- Should the sandbox be a git-init'd empty dir, or a small canned
  test repo committed under `tmp/e2e/fixtures/`? Lean: empty dir
  for now; canned fixture if AC-3 / AC-4 need real content.
- Should AC-5 also assert the "Graph ingest summary" block reports
  identical totals across runs (stronger idempotency check)? Lean:
  yes once the framework proves stable.
