---
title: Per-Feature Smoke Coverage — Continuous Real-World Verification
type: feature
status: delivering
priority: P0
tags: [smoke, testing, continuous, anti-bigbang, regression-prevention, dogfood]
created: 2026-04-28
relations:
  - target: get-back-on-track
    kind: parent
  - target: e2e-area-suites
    kind: companion
  - target: acceptance-criteria-graph
    kind: depends-on
  - target: spec-status-integrity
    kind: complements
mission_alignment: |
  Every Hero feature, every Hero command must have a real-world smoke
  test that runs continuously — on every commit that touches it — not
  big-bang at the end. This is the structural fix for the v2 drift
  pattern where features regressed silently for weeks before anyone
  noticed. The mission requires sessions start smart on a corpus that
  works; if commands silently break between scans, the corpus is
  lying. Continuous smoke is how we make "the system works" a
  verifiable, ongoing claim, not a hope.
principles_check: |
  Serves #1 (it just works — but only because we verify continuously
  that it actually does), #4 (sessions end making everyone smarter,
  including the system itself — every smoke run flips AC status,
  enriching the corpus). Risks #5 if smoke output drowns the
  practitioner CLI; mitigated by smokes running silently in the
  background by default, surfacing only on failure or via explicit
  `hero smoke status`.
horizon: now
---

## Goal

Every Hero feature and every Hero command has a real-world smoke
test that exercises its happy path against a real workspace. Smokes
run continuously — on every commit that touches the relevant code,
on every PR, on a nightly schedule — flipping acceptance-criterion
status in the graph. Big-bang verification at the end of a sprint
becomes structurally impossible because regression is caught the day
it happens.

## Why now

The v2 audit established the cost of NOT doing this. The monolithic
`e2e_smoke.sh` ran *once* and surfaced ~9 issues; phases 4–10 of v2
went weeks without any targeted exercise; "delivered" features
silently regressed; the recovery work exists because the drift
compounded undetected. User after seeing the audits: *"we should
capture that we need to actually make a real world automated smoke
test on everything we do and run and verify it for every hero
command and feature as we rework them all - instead of a big bang at
the end where we realized we were so far off track."*

The fix is not a bigger smoke test. The fix is **per-unit**
verification that runs whenever the unit changes.

## The three layers of smoke

| Layer | Scope | Frequency | Owner |
|---|---|---|---|
| **Per-feature** | One feature spec → one script | Every commit touching feature's files | This spec |
| **Per-area** | Eight area suites covering related features | Nightly + on PR | [`e2e-area-suites`](../e2e-area-suites/spec.md) |
| **Full e2e** | Everything end-to-end | Weekly + on release tag | Existing [`scripts/e2e_smoke.sh`](../../../../scripts/e2e_smoke.sh) |

Per-feature is the new layer. The area suites and the full e2e
remain — they catch integration-level breakage that per-feature
smokes can't.

## Surface

### `smoke:` field on every spec

Every feature/initiative/bug spec gains a frontmatter field:

```yaml
smoke:
  script: scripts/smoke/<feature-slug>.sh
  expects: [feature-slug:AC-1, feature-slug:AC-2, ...]
  runs_on: [commit-touches:<glob>, pr, nightly]
```

`hero check` rejects new specs without `smoke:` (or `smoke: deferred`
with a written reason — escape hatch with friction).

### `scripts/smoke/<feature-slug>.sh`

One script per feature. Same AC-runner harness as area suites
(per `e2e-area-suites`):

- Reads ACs from the spec
- Executes the steps each AC requires
- Emits `results.json` mapping AC-id → pass/fail with timing + evidence
- `hero ac record` ingests the JSON, flips graph status

Per-feature smokes are *fast* (target: <30s each). They exercise
the happy path on a realistic but minimal workspace, not the full
real-world bake the area suites do.

### Built-in `--smoke` mode for every command

Every `hero <cmd>` gains a `--smoke` flag that runs the command's
own happy-path verification and exits 0/1 with a one-line result.
This is the default invocation per-feature smoke scripts use for
single-command features.

```bash
hero scan --smoke      # runs a smoke pass and exits
hero why --smoke       # runs a smoke pass and exits
hero blocked --smoke   # runs a smoke pass and exits
```

This means even a command without an explicit per-feature smoke
script has *something* runnable. The command knows how to verify
itself.

### `hero smoke` orchestration

```
hero smoke <feature-slug>     # run one feature's smoke
hero smoke --area onboarding  # run all features in an area
hero smoke --all              # run every per-feature smoke
hero smoke --since HEAD~5     # run smokes for features touched
                              #   in the last 5 commits
hero smoke status             # show last-run status of every smoke
```

`hero smoke --since` is the killer use: pre-commit / pre-push hook
runs only the smokes affected by the diff. Fast, targeted, never
wasteful.

### CI integration

A GitHub Action (or any CI) runs `hero smoke --since <merge-base>`
on every PR. Failed smokes block the PR. Same pattern for the
nightly area suites and the weekly full e2e.

### Failure handling

When a smoke fails:
- AC status flips to `failing` or `regressed` in the graph
- `hero status` surfaces the failure in its default output
- `hero blocked` (per `traversal-queries`) includes features whose
  smokes are red
- A graph event fires; `hero recap` shows the regression in the
  next session's startup context (per principle #3 — sessions start
  knowing what regressed)

## Acceptance criteria

**AC-1:** Every feature spec in `.hero/planning/features/` has a
`smoke:` field. Specs without one are flagged by `hero check`.
Verified by running `hero check` on the current workspace.
✅ **passing** — 8a9616b (Phase 2): `hero check validate` flags work specs missing
`smoke:`; 57 existing specs backfilled with `smoke: deferred`; `hero check validate`
now returns 0 missing-smoke issues on the hero repo.

**AC-2:** A per-feature smoke script runs against the hero repo,
emits `results.json`, and `hero ac record` ingests it. Verified
on `master-ingest-restore` smoke (the first one we'll write).

**AC-3:** `hero smoke --since <ref>` runs only smokes for features
whose files were touched between `<ref>` and HEAD. Verified by
making a small change, running, and confirming only the affected
smoke ran.
✅ **passing** — 7f38c4f (Phase 3): `git diff <ref>..HEAD` drives file set;
`smokeTriggeredBy` matches via `commit-touches:` globs (falling back to spec dir).
`TestSmokeCmd_SinceNoChanges` verifies HEAD..HEAD triggers nothing.

**AC-4:** `hero <cmd> --smoke` flag exists for every CLI command.
Single-command features can use it directly without a script wrapper.
Verified by `for cmd in $(hero --help | grep ...); do hero $cmd --smoke; done`.
✅ **passing** — c9d7d07 (Phase 1): persistent `--smoke` flag on rootCmd; every
top-level command with a RunE is wrapped by `smokeInterceptor`; `hero status --smoke`
and `hero scan --smoke` wired as POC with real smoke fns.

**AC-5:** A failing smoke causes the AC's graph status to flip to
`failing`/`regressed` (per `acceptance-criteria-graph`). Verified by
intentionally breaking a feature, running its smoke, and querying
graph status.

**AC-6:** `hero status` surfaces failed smokes in its default output
(one line summary; full list behind `hero smoke status`).

**AC-7:** `hero blocked` includes features whose smokes are red
(joined via failing ACs).

**AC-8:** Pre-commit hook (opt-in) runs `hero smoke --since
HEAD` and blocks commit on failure. Verified.

**AC-9:** CI integration: a GitHub Action runs `hero smoke --since
<base>` on every PR. Failed smokes show in the PR check.
✅ **passing** — Phase 5: `.github/workflows/smoke.yml` runs `hero smoke --since
<base-sha>` on PR + push, `hero smoke --all` nightly at 07:00 UTC, and supports
manual dispatch. Failed runs upload `tmp/e2e/**` + `.hero/smoke/last-run.json`
as 14-day artifacts. `make smoke` / `make smoke-all` mirror locally.

ACs accrete as edge cases surface (flaky smokes, slow smokes,
external-dependency smokes that can't always run).

## Approach

**Phase 1 — `--smoke` flag scaffolding** (~1 day): Add `--smoke` to
the cobra root command setup, with a default implementation that
runs the command's own help + a no-op execute. Each command then
overrides with its real smoke logic incrementally.
✅ **shipped** c9d7d07 — `internal/cli/smoke.go` (registry + interceptor),
persistent flag on rootCmd, `hero status --smoke` and `hero scan --smoke`
wired as POC. 5 tests in `smoke_test.go`.

**Phase 2 — `smoke:` frontmatter field + parser** (~½ day): Spec
parser update, `hero check` validation rule.
✅ **shipped** 8a9616b — `SmokeConfig` struct + `Smoke *SmokeConfig` on `Spec`;
`parseSmokeBlock` (mirrors `parseRelationsBlock`); `validateSpec` flags missing
`smoke:` on work specs; 57-spec backfill with `smoke: deferred`; 9 new tests.

**Phase 3 — `hero smoke` CLI** (~1 day): The orchestration commands
(`<slug>`, `--area`, `--all`, `--since`, `status`).
✅ **shipped** 7f38c4f — `internal/cli/smoke_cmd.go`; all five dispatch paths;
results stored in `.hero/smoke/last-run.json`; 11 tests in `smoke_cmd_test.go`.

**Phase 4 — first per-feature smoke scripts** (~ongoing): Write
smoke scripts for the recovery features as they ship:
`master-ingest-restore`, `traversal-queries`, `project-charter`,
`acceptance-criteria-graph`, `spec-status-integrity`,
`spec-prioritization`, `core-vertical-layering`. Each ships
alongside the feature itself.
✅ **shipped** (this commit) — 5 scripts written:
`scripts/smoke/acceptance-criteria-graph.sh` (ACs 1–4, 6),
`scripts/smoke/master-ingest-restore.sh` (ACs 1–8),
`scripts/smoke/traversal-queries.sh` (ACs 1, 3–6, 8–9),
`scripts/smoke/spec-status-integrity.sh` (ACs 1, 4),
`scripts/smoke/spec-prioritization.sh` (ACs 1–5).
Each spec's `smoke: deferred` upgraded to a full `smoke:` block.
Deferred: `project-charter` and `core-vertical-layering` (not yet
sufficiently delivered to write honest smokes against).

**Phase 5 — CI integration** (~½ day): GitHub Action template +
documentation.
✅ **shipped** — `.github/workflows/smoke.yml` (PR + push: `--since <base>`;
nightly: `--all`; workflow_dispatch with mode/ref inputs); artifact upload of
`tmp/e2e/**` + `.hero/smoke/last-run.json` on every run; `make smoke` /
`make smoke-all` Makefile targets; CI integration documented in
`docs/cli/testing-and-demos.md`.

**Phase 6 — backfill smokes for existing features** (~ongoing):
Walk existing specs marked `status: completed` (post
`spec-status-integrity` cleanup); write smokes for any that lack
them. This is the recovery's continuous-verification phase.

## Out of scope

- Performance benchmarks (separate concern, not smoke)
- Visual UI testing (Playwright work — separate)
- Cross-vertical smoke testing (a sales-vertical command's effect
  on engineering corpus) — defer until multi-vertical workspaces
  exist

## Open questions

- How does this relate to `living-contract`'s `verified_by:` test
  links? Lean: `living-contract` is unit/integration tests *of code*;
  smoke is end-to-end tests *of the user-facing command*. Both feed
  AC status.
- Should smokes run on a sandboxed copy of the workspace, or in-
  place? Lean: sandboxed (each smoke gets a tmp dir / fixture
  workspace). In-place is faster but pollutes user's hero workspace
  with test artifacts.
- What's the budget cap for a per-feature smoke? Lean: <30s for
  per-feature, <2min for an area suite, <10min for full e2e. Smokes
  exceeding budget get a warning; above 2× budget, they fail.
- How do we handle smokes that depend on external services (Jira,
  GitHub, team server)? Lean: opt-in via env var
  (`HERO_SMOKE_EXTERNAL=1`); default skips them with a "skipped:
  external dep" status that doesn't fail the run.
