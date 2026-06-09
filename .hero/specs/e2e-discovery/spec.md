---
title: E2E Discovery Suite — The Model Can Find What's There
slug: e2e-discovery
type: feature
status: completed
priority: P0
tags: [e2e, discovery, smoke, ac-backed]
created: 2026-04-29
relations:
  - target: e2e-area-suites
    kind: parent
  - target: acceptance-criteria-graph
    kind: depends-on
mission_alignment: |
  Discovery is the everyday surface — `hero search`, `hero ask`,
  `hero recap`, `hero next`, `hero resume` are how a session lands
  with the right context. The mission ("AI gets the right context at
  the right moment") fails closed if any of these silently no-op.
  The original e2e smoke caught `hero ask` returning 0 bytes pre-
  polish; this suite makes that class of regression a one-AC failure
  instead of a buried smoke-pass false positive.
principles_check: |
  Serves #1 (each AC is "verb runs and produces non-trivial output")
  and #4 (every run leaves observation residue). Catches the silent-
  no-op class directly via byte-count assertions on stdout.
horizon: now
smoke: deferred
completed_at: 2026-06-09T19:23:50Z
---

## Goal

Smoke the five discovery verbs: `search`, `ask`, `recap`, `next`,
`resume`. Each AC is a single command-level smoke that catches both
hard failures (non-zero exit) and the silent-no-op class (zero exit,
empty output). Mirrors the
[e2e-traversal](../e2e-traversal/spec.md) suite's shape — runs
against the outer hero repo because discovery in an empty graph is
uninformative.

## Why now

`hero ask` was returning 0 bytes pre-polish; that escaped the
monolithic smoke because nothing asserted on output volume. With
this suite a regression in any of the five verbs surfaces as a
single failing AC instead of operator-eyeball noise.

## Suite shape

- Script: `scripts/e2e/discovery.sh` (sources `scripts/e2e/lib.sh`)
- Target repo: outer hero repo (REPO_ROOT). No sandbox — discovery
  needs the populated graph from everyday work.
- Each step exercises one verb with a known-good input.
- Records `results.json` per the
  [`acceptance-criteria-graph`](../acceptance-criteria-graph/spec.md)
  schema; pass `--record` to ingest into the graph.

## Acceptance criteria

**AC-1:** ✅ **passing** (2026-04-29). `hero search <term>` exits 0
and emits >100 bytes for a known-populated term. Catches the
silent-empty regression and the "FTS5 fallback never fires" class.
Test target: `graph` (returns ~146 matches today). Verified: 3327
bytes returned on first run.

**AC-2:** ❌ **failing** (2026-04-29). `hero ask <question>` exits
0 and emits >100 bytes — the canonical pre-polish 0-byte
regression. Test target: `"what is hero"`. Discovery-suite first
run caught a real silent-no-op: post-`hero scan`, ask returns "No
knowledge found" (38 bytes) for *every* query. Root cause appears
to be in `internal/cli/ask.go:99-103`: passage extraction requires
`r.Path`, but the unified retrieval layer (Phase B) now returns
graph-node results without `Path` set, so all results get filtered
out before passage scoring. Tracked separately — this AC stays red
until the ask path is fixed to either use graph-node body content
or fall through to FTS5 when graph results carry no Path.

**AC-3:** ✅ **passing** (2026-04-29). `hero recap --since 7d`
exits 0 with non-zero stdout. Catches the case where the recap
walker returns 0 commits silently. Verified: 26108 bytes.

**AC-4:** ✅ **passing** (2026-04-29). `hero next` exits 0 with
non-zero stdout. Verifies the NEXT.md projection is readable
end-to-end. Verified: 4008 bytes.

**AC-5:** ✅ **passing** (2026-04-29). `hero resume --budget 500`
exits 0 with non-zero stdout. Smokes the brief generator with a
small budget so the run stays fast. Verified: 3504 bytes.

**AC-6:** ✅ **passing** (2026-04-29). `hero ac list e2e-discovery
--json` returns a JSON array that includes `AC-1`. Proves this
spec's own ACs were ingested by `hero scan`. Verified live — first
`--record` run flipped 5/6 ACs to `passing` and emitted 5
satisfied_by edges to the run's commit SHA.

ACs accrete as runs surface "the verb worked but the result was
useless" gaps.

## Completion Ledger

| # | Item | Status | Evidence |
|---|------|--------|----------|
| AC-1 | `hero search <term>` exits 0 and emits >100 bytes for known-populated term | DONE | `scripts/e2e/discovery.sh:AC-1` — runs `hero search graph`, asserts exit=0 and >100 bytes. Verified: 3327 bytes on first run. |
| AC-2 | `hero ask <question>` exits 0 and emits >100 bytes | DONE | `scripts/e2e/discovery.sh:AC-2` — check correctly implemented. Currently **fails at runtime** (known bug: unified retrieval returns graph-node results without `Path` set, so passage extraction returns 0 bytes). Root cause documented in spec; tracked separately. The AC correctly surfaces this regression. |
| AC-3 | `hero recap --since 7d` exits 0 with non-zero stdout | DONE | `scripts/e2e/discovery.sh:AC-3` — asserts exit=0 and >0 bytes. Verified: 26108 bytes. |
| AC-4 | `hero next` exits 0 with non-zero stdout | DONE | `scripts/e2e/discovery.sh:AC-4` — asserts exit=0 and >0 bytes. Verified: 4008 bytes. |
| AC-5 | `hero resume --budget 500` exits 0 with non-zero stdout | DONE | `scripts/e2e/discovery.sh:AC-5` — asserts exit=0 and >0 bytes. Verified: 3504 bytes. |
| AC-6 | `hero ac list e2e-discovery --json` returns array including AC-1 | DONE | `scripts/e2e/discovery.sh:AC-6` — greps for `"ac_id":"AC-1"`. Verified: 5/6 ACs flipped to passing on first `--record` run. |

### Exercise-the-feature check

- [x] Exercised: `scripts/e2e/discovery.sh` exists and implements all 6 ACs. First run verified: ACs 1/3/4/5/6 passing, AC-2 correctly failing (catching real bug). `go build ./...` clean.

## Out of scope

- Quality of `ask` answers (LLM-narrated semantic relevance) —
  separate eval harness, not e2e.
- `search --cross-repo` and federation paths — covered by
  area 8 (Federation) suite when it ships.
- `recap --format json` schema regressions — add as a dedicated AC
  once the framework proves stable.

## Open questions

- Should AC-1 also smoke `hero search --json` to catch JSON-format
  regressions? Lean: add as AC-7 once the framework proves stable.
- Test target for AC-1 (`graph`) is a high-recall term today; if
  the graph schema renames it bit-rots. Mitigation: a comment in
  the script explaining what to re-pin if it ever returns 0
  matches.
