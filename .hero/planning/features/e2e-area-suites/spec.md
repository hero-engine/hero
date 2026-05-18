---
title: E2E Area Suites — Per-Area Tests with AC-Backed Verification
slug: e2e-area-suites
type: feature
status: planning
priority: P0
tags: [e2e, testing, ac-backed, regression-prevention, dogfood]
created: 2026-04-28
relations:
  - target: get-back-on-track
    kind: parent
  - target: e2e-validation
    kind: extends
  - target: acceptance-criteria-graph
    kind: depends-on
  - target: project-charter
    kind: depends-on
mission_alignment: |
  Ongoing verification that the corpus → injection loop actually
  works for users. The mission is "AI gets right context at right
  moment" — these suites prove that's true on real repos, on a
  schedule, with regressions caught at boundary, not at PR review.
  Each suite asserts against ACs from `acceptance-criteria-graph`,
  closing the loop where contract → verification → status are one
  artifact.
principles_check: |
  Serves #1 (assertions surface failure cleanly, no triage required),
  #4 (every run produces residue — observation logs, AC status
  flips). Risks none directly; #5 risk if scripts become a maze of
  sub-flags is mitigated by keeping each script focused on one area.
horizon: next
smoke: deferred
---

## Goal

Split the existing monolithic [`scripts/e2e_smoke.sh`](../../../../scripts/e2e_smoke.sh)
into 8 area-specific suites, each asserting against ACs from
`acceptance-criteria-graph`. Each suite is a standalone, repeatable,
fast-feedback proof that one area of Hero is delivering its mission.
The original smoke script remains as the integration-level pass.

## Why now

The existing e2e smoke ran the full workflow once, surfaced ~9 issues
on first run, and three got patched in `polish round 1`. The other
six are in the audit findings. The rest — every regression in phases
4–10 — slipped past because there's no targeted exercise for any
single area.

Per the recovery-strategy conversation: each area gets a small number
of acceptance criteria built out as we go. ACs become the contract;
the script becomes the executor; AC-graph status becomes the result.

## The eight areas

| # | Area | Mission promise tested | Commands exercised |
|---|---|---|---|
| 1 | **Onboarding** | Fresh project becomes a populated, productive workspace | `init`, `scan`, `status`, `dashboard` |
| 2 | **Discovery** | The model can find what's there | `search`, `ask`, `recap`, `next`, `resume` |
| 3 | **Planning** | Spec-first loop intact end-to-end | `design`, `compose`, `note`, `decide`, `convention` |
| 4 | **Traversal** ← *the v2 unlock* | Graph queries beat grep | `why`, `blocked`, `impact`, `relevant`, `suggest`, `check conflicts` |
| 5 | **Delivery** | Spec → code closes; ACs flip green | `deliver`, `diff`, `drift`, `coverage`, `test`, `contract` |
| 6 | **Ingest tiers** | Tier-1 deterministic + Tier-2 LLM both populate | `import`, `extract`, `scan` (per `master-ingest-restore`) |
| 7 | **Projection** | Graph → md round-trips, no drift | NEXT.md, MEMORY.md, `code/<pkg>/spec.md`, `publish pages` |
| 8 | **Federation** | Team sync works; RLS holds; conflicts surface | `sync push/pull`, conflict detection, multi-machine |

## Suite shape (uniform)

Each suite at `scripts/e2e/<area>.sh`:

1. Loads its own area spec at `.hero/planning/features/e2e-<area>/spec.md`
2. Reads the spec's `## Acceptance criteria` block (parsed via
   AC-graph)
3. Runs the steps required to exercise each AC
4. Records each AC as `pass | fail | skip` with timing and evidence
5. Emits a JSON run-result file `tmp/e2e/<area>-<ts>/results.json`
6. `hero ac record <results.json>` (per `acceptance-criteria-graph`)
   ingests results, flips graph status

Plus a markdown observation log per run (matches existing
`e2e_smoke.sh` format), under `tmp/e2e/<area>-<ts>/observations.md`,
for the operator's eyeball pass.

## Top-level orchestration

`scripts/e2e/all.sh` runs all 8 area suites in dependency order
(per the initiative's sequencing rationale). Honored by CI:

```
1, 2, 7  →  4  →  6  →  3, 5  →  8
```

## Per-area spec scaffolds

Each area gets `.hero/planning/features/e2e-<area>/spec.md` with the
common shape: goal, mission_alignment, principles_check, ACs (3–5
to start), script path, target repos, observation-log template. The
ACs grow as new bugs surface — log entry per AC addition.

This feature spec doesn't enumerate each area's ACs (that's the per-
area spec's job). It defines the framework, the uniform shape, and
the orchestration.

## Acceptance criteria (build-out-as-we-go set)

**AC-1:** ⚠️ **Phase 1+2+3 (3/8 areas)** (2026-04-29).
Onboarding (`29db555`), Traversal, and Discovery suites at
`scripts/e2e/{onboarding,traversal,discovery}.sh` exist, are
executable, run to completion. Discovery surfaced a real `hero ask`
regression (passage extraction requires `Path`, graph-node results
don't carry one — tracked separately). Other 5 areas (Planning,
Delivery, Ingest tiers, Projection, Federation) ship in later
phases.

**AC-2:** ✅ **passing** (commit `29db555`, 2026-04-28). Onboarding
script discovers ACs from the graph via
`hero ac list e2e-onboarding --json` (exposed by Phase-1 of this
work). No hard-coded AC list inside the script. Verified: AC-4 of
the onboarding suite itself proves this — calling `hero ac list`
returns the spec's ACs; the harness uses the same call.

**AC-3:** ✅ **passing** (commit `29db555`, 2026-04-28). Onboarding
script emits `results.json` in the
`acceptance-criteria-graph` schema (`{ac, status, ts, sha,
duration_ms, run_id}`). `hero ac record results.json` consumes it
without error — verified end-to-end: `--record` flag flips all 5
e2e-onboarding ACs to `passing` and emits 5 satisfied_by edges to
the run's commit SHA.

**AC-4:** ✅ **passing** (commit `29db555`, 2026-04-28). AC graph
status flips on every `--record` run. Passing ACs become `passing`;
the existing `acceptance.Record` Phase 2 logic also handles
fail-after-pass → `regressed`. Verified live: 5 e2e-onboarding
Criterion nodes graduated from `proposed` to `passing` after the
first `--record` run.

**AC-5:** `scripts/e2e/all.sh` runs all 8 in dependency order, fails
fast on a blocker, and produces an aggregate report.

**AC-6:** Each area's first AC is a smoke check — *"the area's
primary entry point exits 0 with substantive output."* Catches the
"silent no-op" regressions (the `hero ask` pre-polish bug, the `hero
relevant` 0-byte bug).

**AC-7:** Observation log template includes the operator-fill
sections from existing `e2e_smoke.sh` (ease score, friction notes,
polish targets, things that worked).

**AC-8:** A run that adds a new AC to a spec follows the rule:
spec edited first, then script updated. PR description references
the spec change.

ACs accrete as runs surface gaps in the framework.

## Approach

**Phase 1 — framework + area 1** (~1 day): Build the shared
script-runner harness (parses AC list from spec, executes steps,
emits `results.json`). Stand up `e2e-onboarding` spec with 5 ACs +
script. Run against `go-task/task` and hero repo. Iterate until
passing or until ACs accrete to honesty.

**Phase 2 — areas 2 + 7** (~1 day each): Discovery + Projection.
These complete the foundation slice.

**Phase 3 — area 4 (Traversal)** (~1 day): Depends on
`traversal-queries` shipping `hero why` / `hero blocked`. Pivotal
suite — proves v2 paid off.

**Phase 4 — area 6 (Ingest)** (~1 day): Depends on
`master-ingest-restore`. Confirms `hero scan` covers all 7 sources.

**Phase 5 — areas 3 + 5** (~1 day each): Planning + Delivery. The
spec-first loop end-to-end.

**Phase 6 — area 8 (Federation)** (~2 days): Multi-machine setup.
Re-uses scaffolding from
[graph-memory-7c-live-test](../../planning/features/graph-memory-7c-live-test/spec.md).

**Phase 7 — `all.sh` orchestration + CI** (~½ day).

## Target repos

Default: `go-task/task` (existing baseline) + the hero repo itself
(self-test).

Cross-language extensions (deferred until Go suite is green):
- Python: `httpie/httpie`
- TypeScript: `vadimdemedes/ink`

## Out of scope

- Performance/load testing (separate effort, not e2e)
- Cloud/team-server deployment validation (covered by `launch-
  readiness`, deferred per initiative scope)
- Visual UI testing of dashboard (Playwright work — separate)

## Open questions

- Should each area suite assume `hero` is on PATH or take
  `HERO_BIN=` as the existing smoke does? Lean: same as existing —
  `HERO_BIN` env, default `hero`.
- Should runs auto-trigger on commit (CI) or only on demand? Lean:
  on-demand for v1; CI integration deferred until all 8 suites are
  passing locally.
- Should observation logs be committed to the repo (alongside the
  spec) or stay in `tmp/`? Lean: `tmp/` for runs; one canonical
  "last-known-good" log per area committed under the area spec dir
  for reference.
