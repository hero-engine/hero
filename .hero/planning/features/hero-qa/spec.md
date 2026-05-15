---
title: Hero QA — Quality Assurance Domain Pack
type: feature
status: planning
priority: P1
tags: [platform, domains, qa, testing, content-pack]
created: 2026-05-15
relations:
  - target: hero-domains
    kind: parent
  - target: hero-pm
    kind: sequenced-after
depends-on:
  - domain-plugin-architecture
  - spec-type-registry
  - domain-routing-and-agents
  - dashboard-view-registry
  - scan-pluggability
  - domain-scoped-knowledge-graph
horizon: next
smoke: deferred
---

> **Status: more sketch than spec.** This stub is a `/design`-ready
> brief, not a complete design. Implementation is blocked on the full
> platform primitives (items 1–6 of the parent initiative) and benefits
> from lessons learned during `hero-pm` (item 7) delivery. Many
> unknowns are listed explicitly — `/design hero-qa` will need a real
> discovery pass before producing the design.

## Kickoff

Second non-engineering Hero domain pack: QA. Test plans, regression suites, bug intake, agents for test design and regression curation, dashboard views for coverage and flaky tests. Integrations are explicitly undecided — TestRail, Xray, Zephyr, qTest, and "team's spreadsheet" are all real choices. Bugs from QA already flow into engineering's `/diagnose`; the QA pack makes raising one ergonomic. Parent initiative names this as the proof that Hero is a platform, not "engineering with synonyms."

**Status:** planning — sketch-grade brief written 2026-05-15. Blocked on platform primitives 1–6 and `hero-pm` lessons.

**Pick up at:** Run `/design hero-qa`. Before any artifact-type table, do a real domain discovery pass — pick one or two QA tooling targets (TestRail or Xray most likely), resolve the integration-shape question, and confirm the artifact list with someone who actually runs a QA org. Many unknowns; expect this design to take more passes than `hero-pm`.

→ `/design hero-qa`

**Files:** .hero/planning/features/hero-qa/spec.md, .hero/planning/initiatives/hero-domains/spec.md, .hero/planning/features/hero-pm/spec.md
**Skip:** Generic "supports every test-management tool" integration in v1. Building QA agents before primitives 1–6 ship. Treating QA as a thin variation of engineering — that's the failure mode the parent initiative is trying to avoid.

## Goal

Ship the second non-engineering Hero domain pack: Quality Assurance.
The QA pack provides QA-shaped spec types (test plan, regression
suite, possibly QA-owned bug records), agents for test strategy and
regression curation, QA-specific dashboard views, and an integration
story for at least one major test-management tool. Success means a QA
lead can plan a release's test coverage, track regression status, and
raise a defect that flows into engineering's `/diagnose` workflow — all
inside Hero — with the cross-domain handoff visible in the graph.

QA is also the parent initiative's *narrative* deliverable: it proves
that Hero absorbs a meaningfully different content shape (test plans,
defect lifecycles) without regressing PM. PM-first risks feeling like
"engineering with different words"; QA visibly differentiates.

## Why now / why second

PM ships first because its artifacts are spec-shaped, its integrations
reuse existing trackers, and the engineering handoff is clean. QA is
deliberately second:

- **Different artifact shape.** Test plans and regression suites
  aren't just "stories with different names" — they have run state,
  pass/fail history, flakiness, and coverage relationships to the
  features they test. The spec-type registry's flexibility gets a
  real workout here.
- **Fragmented integration landscape.** TestRail, Xray, Zephyr,
  qTest, and "an aging spreadsheet" all exist in real orgs.
  PM v1 sidestepped new integrations; QA v1 cannot.
- **Cross-domain hook already exists.** Bugs raised by QA today flow
  into engineering's `/diagnose`. The QA pack just needs to make
  raising one ergonomic and ensure the graph records the handoff.

The parent initiative flags the cross-cutting risk: don't slip QA
delivery. The multi-domain story holds up only if QA actually ships
on a real cadence after PM.

## Artifact types (sketch — confirm during design)

The pack will declare types via the spec-type registry. Sketch only;
real shape comes from the design pass.

| Type | Purpose | Lifecycle (sketch) | Notes |
|---|---|---|---|
| `test-plan` | Release- or feature-scoped test coverage plan | draft → committed → in-flight → completed | Largest QA artifact |
| `regression-suite` | Named suite of regression tests, with history | active → deprecated | Long-lived, not per-release |
| `test-case` _(maybe)_ | Individual test specification | drafted → ready → automated → retired | Or delegated to external tool entirely |
| `defect` _(maybe)_ | QA-owned bug record before handoff to engineering | reported → triaged → handed-off → closed | Or QA just raises engineering's existing `bug` type |
| `flaky` _(maybe)_ | Tracked flaky test with history | known → mitigated → resolved | Or modeled as a view, not a type |

The `(maybe)` rows are explicit unknowns — see below.

## Workflows (sketch — confirm during design)

1. **Test design** — turn a feature spec or PRD into a test plan.
   Cross-domain consumption of PM or engineering specs.
2. **Test execution** — record pass/fail; surface failed cases.
3. **Regression curation** — promote stable tests into a regression
   suite; demote flaky tests; track regression coverage of shipped
   features.
4. **Defect raising / escalation** — a failed test produces a defect
   that hands off into engineering's `/diagnose`. The cross-domain
   edge appears in the graph.
5. **Coverage analysis** — for a given feature, what's tested and
   what isn't.

## Agents (sketch — confirm during design)

- **qa-strategist** — test plan framing, coverage analysis,
  release-readiness.
- **test-author** — produces test cases from feature specs.
- **regression-curator** — maintains the regression suite,
  promotes/demotes tests, surfaces flakiness.
- **defect-raiser** _(maybe)_ — turns a failed test into a defect
  ready for engineering's `/diagnose`. Could also be a skill on
  `test-author` rather than a separate agent.

## Dashboard views (sketch — confirm during design)

- **Test coverage** — coverage by feature, by release, by domain.
- **Regression status** — pass/fail trend per regression suite.
- **Flaky tests** — list with history and mitigation status.
- **Handoff stream** — recent defects raised into engineering, with
  delivery status (mirror of PM's handoff stream).

## Integrations (genuinely undecided)

This is the largest unknown. Real options:

- **TestRail** — common, API exists, test-case-shaped.
- **Xray (Jira app)** — common in Jira shops, integrates with
  existing tracker.
- **Zephyr (Jira app)** — similar profile to Xray, different vendor.
- **qTest** — enterprise, API exists.
- **Spreadsheet (Google Sheets or Excel)** — depressingly common; an
  import-only integration may be more useful than a sync.
- **Native-only** — Hero hosts the test plans and suites; no external
  integration v1.

The right v1 answer is probably one well-supported provider plus a
spreadsheet importer for the long tail. Pick during design.

## Unknowns for design pass

1. **Artifact set.** Which of `test-case`, `defect`, `flaky` are
   first-class spec types vs views vs delegated to external tools?
   Strong opinion: `test-plan` and `regression-suite` are clearly
   Hero-shaped. The rest is real design work.
2. **Integration target(s) for v1.** Pick one provider and one
   importer — or commit to native-only and integrate later. This is
   the largest scope swing in the design.
3. **Defect raising — new type or reuse engineering `bug`?** Reusing
   `bug` makes the cross-domain handoff trivial. A QA-owned `defect`
   type captures pre-handoff state (triage, dedup) but doubles the
   surface. Strong opinion likely: reuse `bug`, capture pre-handoff
   state in the QA workflow without a new type.
4. **Test execution state ownership.** Test pass/fail history is
   noisy and time-series-shaped. Does it live in Hero's graph, in
   the integration, or both?
5. **Coverage relationship to feature specs.** Cross-domain edges:
   a test plan covers a feature. The `domain-scoped-knowledge-graph`
   primitive must make this navigable — confirm during design that
   primitive #6 actually delivered what QA needs.
6. **Flakiness modeling.** Is "flaky" a type, a tag on a regression
   test, a view, or a derived metric? Multiple right answers.
7. **What `/design` and `/deliver` mean here.** Probably `/design`
   on a `test-plan` produces test cases the same way `/design` on a
   PM story produces engineering features. Confirm.

## Boundaries

- **Not** designing PM artifacts or workflows — those land in `hero-pm`.
- **Not** committing to all major test-management tools in v1. Pick
  one or two.
- **Not** modeling test automation results from CI pipelines as a
  Hero data stream in v1 — possibly a follow-up.
- **Not** designing role-based access for QA artifacts — that's
  `cloud-admin`.
- **Not** absorbing engineering's existing `bug` workflow. QA
  triggers it; `/diagnose` stays where it is.

## Risks

- **Scope is genuinely large.** This is more sketch than spec. The
  design pass may need to split this into smaller specs once the
  artifact set and integration target are pinned down.
- **QA tooling is fragmented.** Picking the wrong integration target
  shrinks the addressable user base. Pick by user research, not by
  API quality alone.
- **PM-as-first-domain may have hidden assumptions.** Lessons from
  `hero-pm` delivery should feed back into the platform primitives;
  if QA discovers the primitives are too PM-shaped, that's a
  primitives bug, not a QA bug. Plan for some back-pressure.
- **Cross-domain handoff is the demo.** If the QA → engineering
  `/diagnose` handoff doesn't feel native, the multi-domain narrative
  weakens. Treat that path as the killer demo, mirror of PM's
  story → feature handoff.
