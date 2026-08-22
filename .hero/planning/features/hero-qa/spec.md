---
title: Hero QA — Quality Assurance Domain Pack
slug: hero-qa
type: feature
status: planning
priority: P1
tags: [platform, domains, qa, testing, content-pack]
created: 2026-05-15
updated: 2026-08-22
relations:
  - target: hero-domains
    kind: parent
  - target: hero-pm
    kind: sequenced-after
  - target: dual-mode-pm-qa-capability-packs
    kind: related
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

> **Status: design pass complete.** All three clusters (X — QA-aware
> story lifecycle; Y — coverage-authoring AI workflow; Z — unified
> defect/test-issue view via normalization + cross-kind linking)
> locked during the 2026-05-16 design dialog. 32 design decisions
> recorded. Mockups, agent pack design, and handoff doc still owed
> (parallel to PM's sibling artifacts). Implementation gated on
> platform primitives 1–6 plus the lifecycle-overlay amendment
> to primitive #2 and cross-pack ambient amendment to primitive #4
> named below.

## Composition amendment (2026-08-22)

The full QA pack is dual-mode. It can be enabled as a bounded extension
of an engineering workspace or selected as the primary pack in a
dedicated QA workspace. Engineering retains lightweight quality
essentials for ordinary coding; the full QA pack adds first-class test
artifacts, coverage operations, integrations, specialist agents, and QA
views. These are activation roles for one QA package, not separate
`qa-lite` and `qa-full` content forks. The composition and collision
contract is owned by `dual-mode-pm-qa-capability-packs`.

## Public pack delivery boundary (2026-08-22)

The offline practitioner content is delivered independently by
`qa-public-pack`: the locked 23-agent roster, QA workflow commands,
testing-method skills, and QA-owned artifact definitions live in the public Hero
repository and install without Hero Code, Hero Cloud, TestRail, or Xray. This broad
spec remains open for the proprietary application views, lifecycle-overlay UI,
hosted run history, and connector implementations described below. Those private
surfaces may consume the public pack contract, but the public QA workflows cannot
depend on them.

## Kickoff

Second non-engineering Hero domain pack: QA. The thesis: **Hero QA is the quality spine of the sprint and release, not a TestRail clone with chat.** Coverage is a story-completion gate (engineer hands off to QA in-flow); regression status is a release-readiness signal; defects exist when teams want them but the primary loop is fix-before-ship via story rejection. AI authors 40-80 test cases per sprint in minutes, not days. Integration to TestRail and Xray is seamless write-through so teams never duplicate-enter. Brand interaction: QA Reject augments the story with new acceptance criteria and bounces it back, without spawning orphan defect tickets.

**Status:** planning — design pass advanced 2026-05-16. Three siblings landed in single-day dialog: research-brief (887 lines — 12 tools + agent prior art + methodology grounding + six design-original Hero-QA contributions), agent-pack-design (1,119 lines — 23 agents / 26 skills / 18 commands in eight tiers with P0/P1/P2 priorities and contextual-button inventory per artifact), mockup-brief (1,010 lines — eight killer screens with layout, interactions, anti-patterns, and preset variations). Total QA design depth ~3,670 lines vs PM's ~3,630 — at parity. Implementation gated on platform primitives 1-6 plus two amendments (primitive #2 lifecycle overlays; primitive #4 cross-pack ambient population). HTML mockups + handoff-to-hero-code still owed.

**Pick up at:** Produce the eight HTML mockups under `.hero/mocks/hero-qa/` per `mockup-brief.md` (suggested order: Screen 1 → 4 → 2 → 3 → 7 → 5 → 6 → 8). Then write `handoff-to-hero-code.md` summarizing locked design + sibling docs + primitive amendments for a fresh hero-code session to pick up.

→ `/design hero-qa`

**Files:** .hero/planning/features/hero-qa/spec.md, .hero/planning/initiatives/hero-domains/spec.md, .hero/planning/features/hero-pm/spec.md, .hero/planning/features/hero-pm/agent-pack-design.md
**Skip:** Native-only-no-integration v1 (we ship Xray and TestRail). A standalone `defect` type by default (opt-in only). Treating QA as a thin variation of engineering. Letting QA reject stories without distinguishing AC-gap from scope-expansion.

## Goal

Ship the second non-engineering Hero domain pack: Quality Assurance.
The QA pack provides QA-shaped spec types (test plan, regression
suite, test case, release-gate, opt-in defect), agents for test
authoring and regression curation, QA-specific dashboard views, and
write-through integration with at least two major test-management
tools (Xray first, TestRail fast-follow). Success means a QA lead
can plan a sprint's test coverage in minutes (AI authoring), reject
or accept stories in-flow with QA states extending the PM story
lifecycle, drive release readiness via aggregated coverage and
regression signals, and raise a defect or new-story-candidate that
flows cleanly into engineering or back into PM — all inside Hero —
with cross-domain edges in the graph.

QA is also the parent initiative's *narrative* deliverable: it proves
that Hero absorbs a meaningfully different content shape (test plans,
defect lifecycles, time-series flakiness, lifecycle overlays across
packs) without regressing PM.

## Why now / why second

PM ships first because its artifacts are spec-shaped, its integrations
reuse existing trackers, and the engineering handoff is clean. QA is
deliberately second:

- **Different artifact shape.** Test plans, cases, and regression
  suites have run state, pass/fail history, flakiness, and coverage
  relationships to the features they test. The spec-type registry's
  flexibility gets a real workout — including the *lifecycle overlay*
  capability this design adds to primitive #2 (see Cross-pack
  interactions below).
- **Fragmented integration landscape.** TestRail, Xray, Zephyr, qTest,
  and "an aging spreadsheet" all exist in real orgs. PM v1 sidestepped
  new integrations; QA v1 cannot.
- **Cross-domain hook already exists.** Bugs raised by QA today flow
  into engineering's `/diagnose`. The QA pack just needs to make
  raising one ergonomic and ensure the graph records the handoff.

The parent initiative flags the cross-cutting risk: don't slip QA
delivery. The multi-domain story holds up only if QA actually ships
on a real cadence after PM.

## Strategic frame

Hero QA is shaped by one positioning move that everything else flows
from:

> **Hero QA is the quality spine of the sprint and release, not a
> test-management tool with chat.**

The pack's job is to make coverage a *first-class delivery gate*, drive
regression confidence into release planning, and unify defects/bugs
from wherever they live (TestRail, Xray, Jira, native) into one
queryable view. Existing tools own the *data*; Hero owns the
*synthesis*. Customers don't migrate; they get a better lens.

Implications:
- Coverage is a story-completion gate, not a parallel workstream.
- Release readiness is a named, aggregated state.
- AI authors the bulk of test cases; humans review and shape.
- Defect raising is a *secondary* path — primary is fix-before-ship
  via story rejection.

## Guiding principles

| # | Principle | Earned by |
|---|---|---|
| 1 | **Cover every story** — no delivery without test coverage. | Coverage rail on every story; `qa-investigator` ambient surfacing; coverage-as-story-gate (Cluster X). |
| 2 | **Author at sprint speed** — AI generates cases; humans shape. | Three motions of coverage authoring (Cluster Y); EARS-grounded `test-author` agent; inline-propose accept/edit/reject UX. |
| 3 | **Block before ship, not after** — quality issues stop delivery. | Story `qa-rejected` state with AC augmentation; release-gate aggregation with blocker policy. |
| 4 | **Unify without migrating** — bring all defect/bug sources into one view; never duplicate-enter. | Hero-as-authoring-surface + integration-as-system-of-record write-through pattern; merged defect/bug view (Cluster Z, pending). |
| 5 | **Drive flakiness to zero** — flaky tests are an opinionated backlog. | Active flake queue with verdict workflow (`qa-flake-curator` agent); trend dashboard with intent to reduce. |
| 6 | **Prevent scope-creep through the gate** — QA rejection distinguishes "missing AC" from "new story idea." | Three-action rejection composer: Add AC / Suggest new story / Reject as quality issue. Default friction nudges toward the right action. |

Principles 4 and 5 are partially deferred in early v1 (Cluster Z and
flakiness depth wait until pack v1.1). Principles 1, 2, 3, 6 ship
day one.

## Pack-wide design voice

A recurring decision pattern across the design dialog: **opinionated
defaults + dialed configurability for shops that don't fit the
default.** Examples:

- `qa.defect_lifecycle: "funnel" | "owned"` (L6)
- `qa.gate_style: "inline" | "parallel" | "post-release"` (L13)
- `qa.feature_lifecycle_on_rejection: "informational" | "auto-revert"` (L14)
- `qa.rejection_strictness: "block" | "advise"` (L18)
- `qa.case_format_default: "step" | "gherkin" | "decision-table" | "data-driven"` (L19)

This is consistent with how PM handles methodology presets and should
be named explicitly as the pack's design voice. Hero QA does not
impose process; it ships strong defaults and supports the variance
real teams run.

## Artifact types

| Type | Purpose | Lifecycle | Notes |
|---|---|---|---|
| `test-plan` | Release- or feature-scoped test coverage plan | draft → committed → in-flight → completed | Largest QA artifact. Container for cases. Has coverage matrix per linked story/feature AC. |
| `regression-suite` | Named long-lived suite of regression tests with history | active → deprecated | Not per-release. Promoted-into from per-story plans by `regression-curator`. |
| `test-case` | Individual test specification | drafted → ready → automated → retired | **First-class spec type (L2).** Authored by `test-author` from story ACs. Written through to TestRail/Xray when configured (L17). |
| `release-gate` | Aggregated release-readiness state | open → reviewing → go → no-go | **First-class artifact in v1.** Configurable blocker policy. Reads story states + regression suite state. |
| `defect` *(opt-in)* | QA-owned bug record before handoff to engineering | reported → triaged → handed-off → closed | **Off by default.** Enabled via `qa.defect_lifecycle: "owned"` for shops that need pre-handoff state. Default flow uses engineering's `bug` directly via `/diagnose`. |
| `flaky` *(view, not type)* | Tracked flaky test | n/a | Modeled as a derived view + active backlog with verdict workflow, not a separate spec type (L11). |

## Methodology presets

Parallel to PM's layered-preset model. Teams pick presets per
dimension; preset choice lives in `hero.json` under `qa.presets`.

### `gate_style` preset (primary)

Determines how QA interacts with story lifecycle.

| Preset | Flow | Story states (with QA pack loaded) |
|---|---|---|
| **`inline`** (default) | QA gates story completion directly. Engineer hands off; QA tests; QA accepts or rejects. | `... → in-flight → qa-ready → done` with `qa-rejected → in-flight` arc |
| **`parallel`** | QA runs in its own track. Story completes; verification state is a chip. | Story states unchanged; `qa_state` chip renders |
| **`post-release`** | QA happens after the story ships. Verification feeds release-gate only. | Story states unchanged; verification feeds release-gate aggregator |

All three ship in v1 (L13).

### Test methodology overlay

Determines how cases render and how `test-author` generates them.
Multi-select. Presets: `step-by-step` (default), `gherkin` (BDD),
`risk-based`, `exploratory-charter`. Format selection per case is
also possible — preset is the default.

### Strictness presets

- `qa.rejection_strictness: "block"` (default) — quality-issue
  rejections block story completion.
- `qa.rejection_strictness: "advise"` — rejections raise warnings
  but don't block.

- `qa.feature_lifecycle_on_rejection: "informational"` (default) —
  engineering's linked feature stays in its current state; ambient
  card surfaces the rejection.
- `qa.feature_lifecycle_on_rejection: "auto-revert"` — engineering
  feature reverts from `delivered` to `in-progress` automatically.

## Cross-pack interactions

The QA pack interacts with the PM pack (and engineering pack) at the
artifact-type level. Three patterns:

### Lifecycle overlays (Cluster X — locked)

The QA pack does not modify PM's `story` type. Instead it registers
a **lifecycle overlay** that adds the `qa-ready` and `qa-rejected`
states to the story type when QA pack is loaded. Architecturally:

- PM owns the `story` type and its base lifecycle.
- QA registers an overlay declaring extended states, valid
  transitions, entry/exit conditions, and overlay-owning pack.
- The spec-type registry composes overlays at runtime; conflicts
  surface as install-time errors.
- When QA pack is uninstalled, stored extended states render as
  *labels* (not as transitionable states). No data loss.

This requires an amendment to **platform primitive #2
(`spec-type-registry`)** to support lifecycle overlays from
non-owning packs, in addition to the methodology-layered-field
declarations PM already needs. See *Primitive amendments needed*
below.

### Cross-pack ambient population

QA-pack agents can populate ambient cards on artifacts owned by
other packs. Specifically: when QA rejects a story, an ambient card
appears on the linked engineering `feature` saying "Your linked
story was QA-rejected. N new acceptance criteria added. Open story
→". This means the dashboard view registry (primitive #4) must
support cross-pack ambient registration.

### Cross-pack body augmentation

On rejection, QA appends a collapsible `## QA Findings` block to the
PM story body, with a QA gutter icon for authorship clarity (L15).
This is the most invasive cross-pack write. Editor surfaces must
render foreign-pack content blocks with author attribution and a
fold-by-default state.

## Silo-tearing patterns

Following PM's playbook with QA-specific edges:

### Cross-domain hooks

- **`hero search` returns QA + PM + engineering results.** Searching
  for "csv export" from a QA session returns the test-plan, the PM
  story, and the engineering feature.
- **`hero why test-plan-X` walks plan → story → epic → roadmap-item.**
  QA can trace why a test plan exists end-to-end across packs.
- **`/diagnose` started from a failed test case carries case context.**
  The resulting engineering `bug` has a graph edge back to the test
  case that surfaced it.
- **Engineering feature ambient shows linked-story QA state.** When
  the QA pack is loaded, engineering's feature view surfaces the
  qa_state of its linked story without needing a separate query.

### Brand interactions

Three brand interactions in QA, mirroring PM's `Hand off to /design`:

1. **`Reject (compose findings)`** on a story in `qa-ready`. Opens the
   three-action composer (Add AC / Suggest new story / Reject as
   quality issue). Brand moment because it's the *primary loop* the
   pack exists to make ergonomic.
2. **`Author cases`** on a story without coverage. Opens inline
   `test-author` proposal with accept/edit/reject. Brand moment for
   the "AI authors at sprint speed" thesis.
3. **`Request test seam`** on a case that can't be exercised without
   engineering support. Creates an engineering work item via cross-
   domain edge. Brand moment for the QA→engineering upstream flow.

## Workflows

1. **Test design** — turn a story or feature into a test plan. AI
   authors the cases from acceptance criteria (Cluster Y).
2. **Story acceptance** — engineer hands story to QA; QA tests; QA
   accepts (story → done) or rejects (compose findings, three-action
   composer). Story bounces between qa-ready and qa-rejected as
   needed (Cluster X).
3. **Regression curation** — promote stable per-story cases into the
   regression suite; demote flaky tests; track regression coverage
   of shipped features.
4. **Release-gate evaluation** — aggregate sprint coverage + regression
   status + blockers into a Go / No-Go state for the release.
5. **Defect / bug unification** — surface bugs (Jira/GitHub) and
   test issues (TestRail/Xray) in one inbox via a normalization
   layer; link across kinds via first-class relationships; triage
   test issues via four primary actions (bad-test / story-rejection
   / new-bug / regression).
6. **Flakiness reduction** — active backlog with verdict workflow
   (test-issue / environment-issue / real-bug); drives flake count
   toward zero.

## Coverage authoring — the three motions (Cluster Y)

### Motion 1 — Per-story authoring (everyday loop)

Trigger: story transitions to `qa-ready` without coverage, or QA
opens a `qa-ready` story.

Ambient prompt (`qa-investigator`): proposes `Author cases` with
two density modes.

- **One-per-AC** — fast default. 5 cases for 5 AC, ~30 seconds.
- **Exhaustive** — happy + edge + negative per AC. 15-25 cases per
  story, ~90 seconds. More review burden.

Cases land in an inline pane (story view), accept/edit/reject per
case. AC-grounded generation — `test-author` mechanically derives
cases from EARS-shaped AC structure rather than hallucinating.

### Motion 2 — Per-sprint authoring (planning loop)

Trigger: QA opens Story queue at sprint start.

Bottom-strip verb: `Plan sprint coverage`. Runs `test-author` across
uncovered stories and produces a *coverage budget* — "Story A needs
4 cases. Story B needs 6. Story C largely covered by regression
suite (3 deltas). Total: ~38 cases, ~90 minutes of authoring."

Gives QA a paced plan, not just a list. Then Motion 1 fills it in
per story.

### Motion 3 — Per-feature authoring (continuous-flow shops)

Same engine as Motions 1 + 2 but triggered against "uncovered
features" or "new features since last release" for shops without
sprints.

### AC quality gating (L18)

`test-author` quality hard-depends on AC quality. The three-action
rejection composer addresses scope-creep risk:

| Action | Effect |
|---|---|
| **Add AC to this story** | In current scope. Engineer addresses before completion. |
| **Suggest new story** | Out of current scope. Creates an `intake-item` (PM pack) with QA as source. Does NOT block current story. The anti-goalpost-moving path. |
| **Reject as quality issue** | Current AC isn't met by implementation. Engineer fixes implementation, not AC. |

`qa-investigator` analyzes the finding's shape and surfaces a primary
suggestion among the three. When the finding doesn't contradict any
existing AC, the nudge is `Suggest new story`. Configurable
strictness (`block` / `advise`) sits on top.

### Regression promotion (L22)

After story → `done`, `regression-curator` scores cases for
regression-worthiness (stability of tested feature, blast radius,
customer-impact severity). Surfaces top N (default 3-5 per story)
as "Promote to regression suite?" ambient cards. One-click promotion
writes the case into the active regression-suite spec with a
`promoted-from-story` edge.

## Integrations

### Mode — standalone + seamless hybrid (L3)

Hero hosts plans, cases, suites, defects natively. When TestRail or
Xray is configured, write-through pattern applies:

- **Hero is the authoring surface.** Cases are created and edited
  in Hero.
- **The integration is the system of record for execution.** Run
  state, pass/fail history, environment tags flow from the
  integration back into Hero for display and aggregation.
- **Cases are pushed to the integration on save.** No duplicate
  entry.
- **When someone edits a case directly in the integration**, a
  conflict resolution surface appears (Cluster Z covers this in
  more depth).

### v1 providers (L4)

- **Xray first** — reuses existing Jira plumbing. Native to Jira-shop
  workflows.
- **TestRail second within v1** — broader install base outside Jira
  shops. Clean API.
- **CSV/spreadsheet importer** — for the long tail of orgs that run
  on sheets. Import-only; not bidirectional.

The `TestManagementIntegration` interface must be shaped to absorb
both Xray and TestRail from day one — avoid the "v1 supports one,
v2 retrofits the second" trap.

### Defect/bug unification (Cluster Z — locked)

The unification thesis: **Hero owns the synthesis across sources;
the source systems keep their data.** Customers don't migrate. They
get a better lens. The original "auto-merge duplicates across systems"
model was wrong — bugs and test issues are **distinct artifact kinds
that link via relationships**, not duplicate copies of one kind (L24).

#### Normalization layer (L23, L32)

Each integration provides a mapper to a **standard schema**: `status`,
`severity`, `priority`, `assignee`, `age`, `link`. Hero renders
normalized fields for sorting / filtering / aggregation. Source-
specific fields *decorate* rows for fidelity (a Jira pill, the
original priority string in tooltip, etc.). **No field is overwritten
in the source system by normalization** — write-through only fires
when the user takes an explicit action (close, reassign).

Sensible defaults ship for Jira / GitHub Issues / TestRail / Xray
out of the box; teams override mappings in `hero.json` when their
workflows are unusual.

#### Two artifact kinds, one inbox (L24, L25)

- **Bugs** — engineering-side artifact. Sourced from Jira, GitHub
  Issues, or native engineering `bug` specs. One bug source per
  shop is the norm.
- **Test issues** — QA-side artifact. Sourced from TestRail, Xray,
  or native QA `defect` records (when opt-in lifecycle is on).
  One test-management source per shop is the norm.

These are presented in **one inbox with kind tabs** (`All` / `Bugs` /
`Test issues`) — less left-nav clutter than two singletons, same
underlying engine, cross-kind queries trivial via the `All` tab.

#### Cross-kind relationships (L27)

Where unification actually pays off — not in collapsing duplicate
rows, but in **traversing the graph across kinds**. First-class
relationship edges:

- `raised-bug-from-failure` — test issue → bug it produced.
- `regression-of` — bug → regression test that should have caught it.
- `verifies` — case → story it covers (already established in Cluster X).

Each artifact renders a **Related items** rail showing linked items
across kinds. `hero why` traverses the full chain. Unified inbox
pivots on relationships: "show me bugs that came from test failures
last sprint" is a single query.

#### Test-issue triage workflow (L30, L31)

When a test fails, the resulting test issue is triaged with one of
four primary actions (bottom strip verbs on the test-issue editor):

| Action | Downstream artifact |
|---|---|
| **Mark as bad test / faulty test** | Close the test issue; flag the test for fix or send to flaky backlog. |
| **Reject linked story** | Use the Cluster X three-action rejection composer on the linked story. Story → `qa-rejected`. |
| **Raise as new bug** | New engineering `bug` via `/diagnose`, with `raised-bug-from-failure` edge back to test issue. |
| **Flag as regression** | New engineering `bug` + mark against the regression suite (this test was supposed to protect that area). Higher visibility; `regression-of` edge written. |

After triage, two persistence flavors exist (configurable):

- **`persistent-link`** (default) — test issue stays open, linked
  to its downstream artifact. Closes when downstream closes. Serves
  shops that treat TestRail/Xray as their primary view and "track
  their own test status like their own home."
- **`triage-and-close`** — test issue closes after triage; only
  downstream carries forward. Serves shops where TestRail/Xray is
  "just a way to make Jira reflect the truth."

Selected via `qa.test_issue_persistence`. Default = `persistent-link`
because QA-centric shops are the primary user of this pack.

#### Same-source duplicate handling (L26)

The rare case of two genuine duplicates in one system (two Jira bugs
for the same issue, or two TestRail defects for the same failure) is
handled by a **manual `Mark as duplicate of...` action** that writes
a `duplicate-of` edge. No auto-merge. No edit-propagation policy
machinery. The PM-pack `duplicate-detector` agent can suggest
candidates ambiently but the action is user-confirmed and follows
source-system semantics for what duplication does.

#### Local-first non-negotiable (L28)

All inbox views work fully off local specs when no integration is
configured:

- Bugs inbox renders engineering pack's local `bug` specs on day
  one without integration.
- Test issues inbox renders QA pack's native `defect` records (opt-in
  lifecycle on) or native test-failure events on day one without
  integration.
- "Add Jira integration" CTA in an empty inbox is a *suggestion*,
  not a wall.
- Integration is **additive decoration**, never a prerequisite for
  the view to function.

This generalizes the standalone-capable promise from L3 to every
inbox surface.

#### Embedded inbox panels (L29)

Same engine, filtered context, rendered inline on:

- **Release-gate view** — "Blockers" panel showing bugs and test
  issues currently blocking release readiness.
- **Story view** — "Open against this story" panel showing bugs
  and test issues linked to the story (directly or via cross-kind
  edges).

Bulk actions and triage primary actions are available in embedded
panels at the same depth as in the standalone inbox.

## Dashboard layout

Reuses PM's IDE-style grammar wholesale (L1). Same left nav + tabbed
center pane + bottom strip (verbs) + right panel (sticky ambient +
chat). See `hero-pm/spec.md#dashboard-layout` for the grammar
specification; do not redesign in this pack.

## Dashboard views

The pack registers the following views via the dashboard view
registry. Default landing is Story queue (coverage-centric view).

| View | Tab kind | Default landing? | Earns principle | Notes |
|---|---|---|---|---|
| **Story queue with coverage signals** | singleton | yes | 1, 2, 3 | Sprint or continuous-flow view. Each story shows coverage state (covered / partial / uncovered) and qa_state chip. |
| **Test plan editor** | per-item | — | 1, 2 | The PRD-equivalent for QA. Coverage matrix per linked AC. Cases as rows or as linked spec children depending on volume. |
| **Test case editor** | per-item | — | 2 | Spec-shaped editor with format-aware rendering (step / gherkin / decision-table / data-driven). Run-history rail from integration. |
| **Regression suites** | singleton | — | 1, 3, 5 | List of active suites with pass/fail trend per suite. |
| **Flaky tests backlog** | singleton | — | 5 | Active queue with verdict workflow (test-issue / environment / real-bug). Trend toward zero. |
| **Release-gate** | per-item | — | 3, 4 | Aggregated Go / No-Go view per release. Configurable blocker policy. |
| **Unified inbox** (kind tabs: All / Bugs / Test issues) | singleton | — | 4 | Normalized rendering across sources; cross-kind relationship rail; test-issue triage with 4 primary actions; works fully off local specs without integration. |
| **Cross-domain handoff stream** | singleton | — | 4 | Mirror of PM's stream. Recent QA rejections, defects raised, seam requests. |
| **Chat** | singleton | — | 1, 2 | Same as PM's Chat tab. |

Methodology preset variance:
- Story queue: shows different coverage signals under inline vs
  parallel vs post-release gate_style.
- Test case editor: format defaults per `qa.case_format_default`.
- Release-gate: blocker policy rules per team config.

## Open questions for design pass

### Pending agent-pack-design

- Full agent roster (sketch: `qa-strategist`, `test-author`,
  `regression-curator`, `qa-investigator`, `qa-flake-curator`,
  `seam-requester`, `duplicate-detector` overload). Detailed
  agent files at the depth of PM's 27-agent design.
- Skill roster (sketch: `ears-test-derivation`, `gherkin-authoring`,
  `regression-scoring`, `flake-triage`, `coverage-budgeting`,
  `seam-request-shaping`).
- Command roster (sketch: `/plan-coverage`, `/author-cases`,
  `/promote-to-regression`, `/release-gate`, `/triage-flaky`,
  `/unify-defects`).
- Contextual button inventory per artifact.

### Pending mockup-brief

- Eight or nine killer screens (parallel to PM):
  - Story queue with coverage signals (default landing)
  - Test plan editor with coverage matrix
  - Test case editor with format toggle and run history
  - Story view in `qa-ready` (showing the three-action rejection composer)
  - Release-gate view with Go / No-Go
  - Flaky tests backlog with verdict workflow
  - Unified defect/bug inbox (after Cluster Z)
  - Cross-domain handoff stream (QA-side)

## Primitive amendments needed

This design generates two new requirements for already-named
platform primitives:

1. **Primitive #2 (`spec-type-registry`) — add lifecycle overlays.**
   Already needed: layered-field declarations from preset config (per
   PM). Add: lifecycle overlays from non-owning packs (per QA's
   `qa-ready`/`qa-rejected` extension of PM's `story` type). Overlays
   declare extended states, valid transitions, entry/exit conditions,
   and owning pack. Graceful degradation when overlay's pack is
   uninstalled — stored states render as labels.
2. **Primitive #4 (`dashboard-view-registry`) — add cross-pack ambient
   population.** A pack must be able to register an ambient-card
   producer that runs on artifacts owned by a different pack (e.g.
   QA-pack agent populates ambient cards on engineering `feature`
   view). Requires a clean registration API and pack-load-order
   sequencing.

Both amendments should be added to the primitive specs before the
primitives ship, not retrofitted after QA design begins delivery.

## Boundaries

- **Not** designing PM artifacts or workflows — those land in `hero-pm`.
- **Not** committing to Zephyr, qTest, or test-management tools other
  than Xray and TestRail in v1.
- **Not** modeling test automation results from CI pipelines as a
  Hero data stream in v1 — possibly a follow-up; flakiness view works
  on manual marking until then.
- **Not** designing role-based access for QA artifacts — that's
  `cloud-admin`.
- **Not** absorbing engineering's existing `bug` workflow. QA
  triggers it; `/diagnose` stays where it is.
- **Not** building a QA-only Hero binary — same `hero` binary, domain
  selected via `hero.json`.

## Risks

- **Lifecycle overlay surface is new platform work.** The amendment
  to primitive #2 is real engineering. If underestimated, QA design
  outruns the platform's ability to support it.
- **Cross-pack body augmentation is invasive.** QA writing into the
  PM story body needs careful editor support and clear authorship
  attribution. Getting this wrong creates muddled artifacts.
- **Three gate_style presets is real scope.** Building all three in
  v1 costs ~30% more dashboard work than shipping inline only. Trade
  was made because real teams run all three flows; revisit if
  delivery is at risk.
- **TestRail + Xray both in v1 is real scope.** Both providers need
  end-to-end auth + sync + write-through + conflict-resolution. The
  `TestManagementIntegration` interface must absorb both from day
  one to avoid v2 retrofit pain.
- **Scope-creep through the QA gate is a real failure mode.** The
  three-action rejection composer addresses it via product design;
  if the `Suggest new story` path is awkward, QA will fall back to
  `Add AC` and inflate stories silently. UX detail matters here.
- **PM-as-first-domain may have hidden assumptions.** Lessons from
  `hero-pm` delivery should feed back into the platform primitives;
  if QA discovers the primitives are too PM-shaped, that's a
  primitives bug, not a QA bug. Plan for some back-pressure.

## Locked decisions (design dialog 2026-05-16)

For traceability. Each entry is a design lock established in dialog
and reflected in the body above.

| # | Lock |
|---|---|
| L1 | UX grammar matches PM (IDE-style left nav, tabbed center, bottom strip = verbs, right panel = ambient + chat) |
| L2 | Test cases are first-class spec types |
| L3 | Standalone + seamless hybrid integration mode |
| L4 | TestRail and Xray both in v1 (Xray first via existing Jira plumbing) |
| L5 | Coverage-as-story-gate is the brand interaction |
| L6 | Default: no defect type, problems funnel through story rejection or engineering bug. Opt-in `qa.defect_lifecycle: "owned"` for shops that want it |
| L7 | External defects/bugs from TestRail/Xray/Jira merged + categorized in unified view; funnel-together default, split-flow opt-in |
| L8 | Story lifecycle gains `qa-ready` and `qa-rejected` states when QA pack is active |
| L9 | QA Reject button augments story with new AC/tasks rather than spawning a defect |
| L10 | `release-gate` is a v1 first-class artifact with configurable blocker policy |
| L11 | Flakiness ships in v1 as an active backlog with verdict workflow, not a passive list. CI integration deepens it later |
| L12 | Lifecycle-overlay architecture (Option B) — QA pack overlays states onto PM's `story` type without modifying it |
| L13 | Three `gate_style` presets in v1: `inline` (default), `parallel`, `post-release` |
| L14 | Engineering feature lifecycle effect on QA rejection is configurable per team; default = informational ambient card, not auto-revert |
| L15 | QA-appended content in story body = collapsible blocks with QA gutter icon |
| L16 | Primitive #2 amendment required: lifecycle overlays from non-owning packs |
| L17 | Hero is authoring surface; integration (TestRail/Xray) is system of record for execution. Write-through on save |
| L18 | AC-quality rejection is configurable (`block` / `advise`) AND offers three actions: Add AC / Suggest new story / Reject as quality issue. Prevents scope-creep through the gate |
| L19 | All four test formats in v1: step-by-step (default), Gherkin (BDD preset), decision-table, data-driven |
| L20 | First-class `Request test seam` button on cases — creates engineering work via cross-domain edge. New `seam-requester` agent |
| L21 | Coverage authoring has three motions: per-story (everyday), per-sprint (planning), per-feature (continuous-flow shops). All driven by `test-author` at different scopes |
| L22 | Regression promotion is a deliberate ritual after story → done; `regression-curator` proposes top N cases based on stability + blast-radius + customer-impact |
| L23 | Normalization layer maps source fields to standard schema; source decoration preserved; no overwriting of source-system fields via normalization |
| L24 | Bugs and test issues are distinct artifact kinds that link via relationships, not duplicate copies. The original auto-merge-across-sources design is removed |
| L25 | One unified inbox with kind tabs (`All` / `Bugs` / `Test issues`) |
| L26 | Same-source duplicate handling is a manual `Mark as duplicate of...` action; no auto-merge or edit-propagation policy machinery |
| L27 | Cross-kind relationships are first-class: `raised-bug-from-failure`, `regression-of`, `verifies`. Renders as Related items rail; queryable via `hero why` |
| L28 | Local-first non-negotiable. All inbox views work fully off local specs when no integration configured. Integration is additive decoration |
| L29 | Embedded inbox panels ship in v1 on release-gate ("blockers") and story ("open against this story") |
| L30 | Test-issue triage is a first-class workflow with 4 primary actions: bad-test / reject-linked-story / raise-as-new-bug / flag-as-regression |
| L31 | `qa.test_issue_persistence: "persistent-link" \| "triage-and-close"`. Default = `persistent-link` (serves QA-centric shops) |
| L32 | Normalization shipped with sensible defaults for well-known systems (Jira / GH / TR / Xray); overridable via `hero.json` per shop |
