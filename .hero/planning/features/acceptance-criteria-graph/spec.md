---
title: Acceptance Criteria as Graph Nodes
slug: acceptance-criteria-graph
type: feature
status: completed
status_verified: "2026-04-29 by hero ac record: 7/7 ACs passing"
priority: P0
tags: [graph, acceptance-criteria, contract, injection, dogfood]
created: 2026-04-28
relations:
  - target: get-back-on-track
    kind: parent
  - target: project-charter
    kind: sibling
  - target: spec-status-integrity
    kind: sibling
  - target: e2e-area-suites
    kind: enables
  - target: graph-memory
    kind: extends
mission_alignment: |
  Acceptance criteria are the most testable signal a project produces:
  what does "done" mean, did we get there. Today they're bullet points
  in markdown — invisible to traversal, invisible to injection, drift-
  prone. Making them first-class graph nodes turns them into structured
  context the model sees during delivery, the criterion the system
  grades against, and the signal that flips status truthfully. This is
  context-engineering applied to the most actionable knowledge a project
  has.
principles_check: |
  Serves #3 (sessions start omniscient — open ACs become part of the
  resume context), #4 (sessions end making everyone smarter — every AC
  pass/regression is captured automatically as graph state). Risks #5
  if surfaced as raw query API to humans; mitigated by injection-first
  design — ACs reach the model via context bundles, not via a new CLI
  surface humans must learn.
horizon: now
smoke:
  script: scripts/smoke/acceptance-criteria-graph.sh
  expects: [acceptance-criteria-graph:AC-1, acceptance-criteria-graph:AC-2, acceptance-criteria-graph:AC-3, acceptance-criteria-graph:AC-4, acceptance-criteria-graph:AC-6]
  runs_on: [commit-touches:internal/spec/acceptance*.go, commit-touches:internal/cli/ac*.go, nightly]
---

## Goal

Make acceptance criteria first-class graph citizens — `Criterion`
nodes with status, edges to features/scripts/commits/files, automatic
status updates from runs and commits, and injection into every
delivery-relevant context bundle. End the era of "AC-3 was a bullet
point that may or may not have ever passed."

## Why now

The recovery audit catalogued multiple ways spec status drifts from
reality (lying frontmatter, lying commit messages, phased-plan
checkmark fraud). The root cause is consistent: there is no
machine-readable, machine-verifiable contract for what each spec
promised. ACs in markdown are prose; nothing checks them.

Concurrently, the v2 graph thesis requires showcase queries to justify
its complexity tax. Modeling ACs as nodes immediately enables several:

- *"What ACs am I being graded on right now?"* — `hero deliver` injection
- *"What ACs broke when this commit landed?"* — `breaks` edge
- *"Which open features have failing ACs?"* — `hero blocked` joins
- *"Why does this AC exist?"* — `hero why AC-X` traverses to origin

## Schema additions

```sql
-- New node type — Criterion
-- key: <feature-slug>:AC-N (e.g. "e2e-onboarding:AC-3")
-- props: { statement, status, last_run_at, last_pass_at, severity }
-- source: { spec_path, spec_line }

-- New edge types — all Tier-1 deterministic, no LLM:
-- Criterion --belongs_to-->     Feature
-- Criterion --verified_by-->    Script | TestFile
-- Criterion --satisfied_by-->   Commit
-- Commit    --breaks-->         Criterion       (regression)
-- File      --participates_in--> Criterion      (join: file touched
--                                                by satisfying commit)
-- Criterion --derived_from-->   Attempt         (when a failed run
--                                                earned this AC)
```

Status enum: `proposed | passing | failing | regressed | retired`.

Bitemporal: status changes don't update — they invalidate the prior
row (`valid_to = now`) and insert a new one. Every status flip is a
permanent history record. *"When did AC-3 first pass? When did it
regress?"* are queries, not lost data.

## Ingest

**Tier-1 deterministic — no LLM required:**

- **Spec parser:** read `## Acceptance criteria` block from every
  feature/initiative/bug spec. Extract `AC-N: <statement>` patterns.
  Upsert `Criterion` nodes + `belongs_to` edges. Runs on `hero scan`
  and on file-watcher events.
- **Verifier annotations:** parse `verified_by:` annotations
  (already-existing format from `living-contract` spec) — link
  Criterion to the script/test that validates it.
- **Run results:** scripts (and `hero test`, `hero contract check`)
  emit a JSON payload mapping AC-id → pass|fail|skip. Hero ingests it
  and flips status (with `valid_to` invalidation). Commit SHA at run
  time becomes the `satisfied_by` / `breaks` edge target.
- **Diff-based file mapping:** for each `satisfied_by` commit,
  enumerate touched files and create `participates_in` edges
  (File → Criterion).

**Tier-2 (later, optional):** propose AC candidates from spec prose
that doesn't have an explicit AC block. Surface as suggestions, not
auto-add.

## Injection — the "guiding light" surfaces

The point of nodes-not-bullets is what they enable. Every command
listed below gets an AC-aware block in its output:

| Command | What gets injected |
|---|---|
| `hero deliver <spec>` | Open ACs printed as the success bar — the engineer agent's prompt explicitly includes "you are being graded on these N criteria" |
| `hero relevant <file>` | ACs touching this file via `participates_in` — *"changing this file affects AC-3 in onboarding and AC-1 in projection"* |
| `hero next` / `hero resume` | "Since last session: AC-X went green, AC-Y regressed" — concrete progress signal |
| `hero blocked` | Open features whose ACs are failing — joined automatically |
| `hero impact <file>` | Add at-risk-ACs column |
| `hero why <AC-id>` | Origin story — which bug/run created this criterion |
| `hero status` / `hero check` | AC-pass-rate per spec; specs with `status: completed` and any non-passing AC surface as truthfulness violations |
| Project-context-builder agent / delivery leads | Open ACs become a section in every context bundle the agent receives |

Two principles are explicitly served: #3 (sessions start omniscient —
the model sees what was passing/failing without anyone telling it) and
#4 (sessions end making everyone smarter — AC state from a delivery
session becomes context for the next one).

## Acceptance criteria (build-out-as-we-go set)

**AC-1:** ✅ **passing** (commit `91e4e69`, 2026-04-28). Spec parser
at `internal/spec/acceptance.go` extracts `## Acceptance criteria`
block (case-insensitive, accepts heading suffixes like
`Acceptance criteria (build-out-as-we-go set)`) and pulls
`AC-N: <text>` entries in both bullet (`- **AC-N:** …`) and paragraph
(`**AC-N:** …`) form. Continuation lines join with a single space.
Tests in `acceptance_test.go` cover bold, plain, bullet, paragraph,
and missing-section variants. Verified on get-back-on-track child
specs: 75 Criterion nodes ingested across 9 specs (5–9 per spec).

**AC-2:** ✅ **passing** (commit `91e4e69`, 2026-04-28). `hero scan`
upserts `Criterion` nodes (key: `<spec-slug>:AC-N`, props: ac_id,
statement, status=unknown, parent) and `belongs_to` edges to the
parent Feature/Initiative/Bug node. `hero graph stats` shows
`Criterion 75` after scan; SQL spot-check confirms 75 belongs_to
edges from Criterion → parent.

**AC-3:** ✅ **passing** (commit `03333d7`, 2026-04-28). Run-result
JSON (format: `[{"ac": "<spec-slug>:AC-N", "status": "pass|fail",
"ts": "...", "sha": "..."}]`) is consumed by `hero ac record
<run.json>`. Status flips are bitemporal — the prior row is
invalidated and a new current row is inserted. Pass-after-fail
promotes to "passing"; fail-after-pass promotes to "regressed".
Verified end-to-end on the 5 master-ingest-restore ACs that landed
this session: 5 status flips applied, 4 satisfied_by edges to
referenced commits, scan idempotency preserved.

**AC-4:** ✅ **passing** (commit `72643b3`, 2026-04-28). `hero deliver
<spec>` (and `hero spec deliver <spec>`) prints an "Acceptance criteria
— graded on N/M open" block before the manual/async dispatch. Each AC
gets a glyph (✅ / ❌ / ⚠️ / ⊘ / ◯) plus a one-line summary. Verified
end-to-end on master-ingest-restore: 5 passing + 3 proposed shown
correctly.

**AC-5:** ⚠️ deferred — Phase 4 dependency. The `participates_in`
File→Criterion edge isn't computed until Phase 4 (file participation
join from satisfied_by commit diffs). Until then, `hero relevant`
remains AC-blind. Phase 3 ships without it.

**AC-6:** ✅ **passing** (commit `03333d7`, 2026-04-28). New
`Store.GetNodeAt(typ, key, at)` returns the row that was current for
(type, key) at the given RFC3339 timestamp — `valid_from ≤ t < valid_to`.
Test: seed Criterion at past timestamp, flip status now, query at a
mid-interval timestamp returns the seed; query current returns flipped.
Verified manually on master-ingest-restore:AC-2: graph history shows
three rows (unknown → proposed → passing) with non-overlapping
[valid_from, valid_to) intervals.

**AC-7:** `hero why <feature:AC-N>` returns the chain: AC →
parent feature → originating note/decision.

ACs accrete as integrations surface gaps.

## Approach

**Phase 1 — schema + parser** (~1 day): `Criterion` node type, spec
parser, ingest path in `hero scan`. Tests on get-back-on-track specs.

**Phase 2 — run-result ingest** (~1 day): JSON schema for run results,
`hero ac record` command, bitemporal status flip, `satisfied_by` /
`breaks` edge wiring. Round-trip test.

**Phase 3 — injection** (~2 days): Wire AC blocks into `hero deliver`,
`hero relevant`, `hero next`, `hero blocked`, `hero impact`. Update
project-context-builder agent and delivery-lead context bundles.

**Phase 4 — file participation join** (~½ day): Compute
File→Criterion edges from `satisfied_by` commit diffs. Enables
`hero impact` AC column.

**Phase 5 — query surface** (~½ day): `hero why <AC-id>`,
`hero ac status [--feature X]`, `hero ac history <id>`. Lightweight;
internal-power-user surface, behind a sub-verb to honor principle #5.

## Out of scope

- AC clustering across features ("all 3 specs have a 'non-empty
  output' AC, suggest factoring") — Tier-3, later
- LLM-proposed AC drafts — out of scope for v1
- AC linking to external trackers (Jira custom fields) — open question

## Open questions

- Should `verified_by:` annotations live in the spec body or in a
  side file (`tests.md` in the three-file layout)? Lean: spec body for
  v1; matches existing `living-contract` shape.
- When a spec is `retired`, do its ACs auto-retire? Lean: yes, with
  cascade from feature retirement.
- Should `Criterion` nodes have their own `scope` (local/team/public)
  or inherit from parent feature? Lean: inherit. Simpler.
