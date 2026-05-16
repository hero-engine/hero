# Hero QA — Competitive UX Research Brief

Audience: the ui-designer agent and downstream design pass on
`hero-qa`. The strategic frame (Hero QA as the quality spine of sprint
and release; layered presets; coverage-as-story-gate; AI authoring at
sprint speed; standalone-capable + seamless hybrid; unified inbox via
normalization + cross-kind linking) is taken as given — this brief
turns it into concrete reference material a designer can study without
having to wander the QA-tool landscape unguided.

For each tool: what they do well, what we steal, what to leave behind,
which Hero screens it influences. Treat "Screens to study" as a
shopping list — pull reference images, vendor docs, and walkthroughs
before drafting the corresponding Hero mockup.

Two notes on positioning:
- Hero QA reuses PM's UX grammar wholesale (L1 in spec). This brief
  does **not** re-justify the IDE-style layout, the bottom strip,
  the ambient right rail, or the chat surface. Those are settled.
  The brief focuses on what is **QA-unique**: coverage signals,
  test-case editing, run history, release-gating, triage flows,
  and AI authoring.
- The Hero QA design dialog (2026-05-16) locked 32 decisions. Where
  research and locks disagree, locks win. The brief surfaces tension
  explicitly when it exists.

The eight Hero QA screens this brief feeds:

1. Story queue with coverage signals (default landing)
2. Test plan editor (coverage matrix)
3. Test case editor (format-aware: step / Gherkin / decision-table / data-driven)
4. Story view in `qa-ready` (three-action rejection composer)
5. Release-gate view (Go / No-Go with blockers)
6. Flaky tests backlog (active queue, verdict workflow)
7. Unified inbox (kind tabs: All / Bugs / Test issues)
8. Cross-domain handoff stream (QA-side)

---

# Part I — Test-management tools (the "house" QA lives in today)

These are the tools Hero QA must respect, replace, or sit alongside.
They define what QA leads expect to see and where their workflow
muscle memory lives.

---

## TestRail

**Influences:** Test plan editor (primary); Test case editor (primary);
Regression suites (secondary).

TestRail is the dominant non-Jira test-management tool — case-shaped,
manual-test heavy, and the default choice for QA-led orgs that don't
live inside Jira. It's the one a Hero QA pack absolutely cannot
ignore.

### What they actually do well

- **Test case as a first-class document.** Each case has Preconditions,
  Steps (multi-row), Expected Result, Type, Priority, Estimate,
  References. The grid-of-steps with separate Expected per step is a
  small but important affordance — it lets reviewers spot ambiguity
  per step.
- **Suites as containers.** A test run pulls cases from one or more
  suites and creates a snapshot. Cases live in suites; results live
  in runs. Clean separation between "what we test" and "what we did
  this round."
- **Templates per project.** Cases default to a project-specific
  template (manual, exploratory, BDD). Changing template doesn't
  rewrite existing cases — just shapes new ones.
- **Run dashboard with progress and assignment.** Per-run
  pass/fail/blocked/untested counters with assignees per case.
  The "who's stuck" surface is genuinely useful in a live test cycle.
- **Stable, well-shaped API.** Hero's integration plumbing should
  not be hard to write against. Cases, sections, suites, milestones,
  runs, results, plans are all clean REST resources.

### What we should steal verbatim

- **Multi-row step grid with per-step Expected** → Hero Test case
  editor (default step format). Each step row has Action and
  Expected Result columns. Importantly: ship this as the *default*
  format. Gherkin, decision-table, and data-driven are toggles on
  top.
- **Cases live in containers; results live in runs** → Hero
  Test plan = a run + the snapshot of cases. Regression suite =
  a longer-lived container of cases promoted from runs. This
  matches L21–L22 (per-story coverage authoring + regression
  promotion).
- **Template-defaulting per project, not per case** → Hero loads
  the default case format from `qa.case_format_default`. Switching
  defaults doesn't rewrite history.
- **Run progress dashboard density** → Hero Test plan editor's run
  status panel: pass/fail/blocked/untested counters, assignee
  filter, per-case status.

### What to leave behind

- **Project-trapped silos.** TestRail projects don't talk to each
  other. Hero's cross-domain graph is the whole point.
- **No AI authoring.** Cases are written by hand or migrated. Hero's
  `test-author` agent is the entire point of difference.
- **Outdated visual density.** TestRail is enterprise-class spacing
  in ~2010 visual language. Hero matches PM's Linear-class density
  (~32px rows, ~12px body).
- **No story / feature awareness.** TestRail has "references" as a
  text field. Hero has real graph edges to PM stories and
  engineering features.
- **Heavy run-setup ceremony.** Creating a run involves picking
  suites, sections, cases, milestones, assignees, configurations.
  Hero's per-story / per-sprint authoring flow obviates 80% of
  this for the common case.

### Screens to study

- The TestRail test-case edit view (Preconditions + Steps grid +
  Expected per step + Priority/Estimate sidebar). Note the row
  density and how Expected sits inline with each step.
- The TestRail run view with the case list + status filter strip +
  per-case status update side panel. Note the test-execution
  rhythm: tester moves down the list pressing P (pass) / F (fail).
- A TestRail milestone view showing multiple runs rolling up.
  Compare against Hero's release-gate aggregation.

---

## Xray (Jira app)

**Influences:** Test plan editor (secondary); Unified inbox
test-issue rendering (primary); QA-rejection-via-story flow
(secondary).

Xray is the Jira-native test-management tool. It's a top three
choice in Jira shops because it doesn't require leaving Jira's
issue model. Test cases and defects are Jira issues with extra
fields.

### What they actually do well

- **Test as a Jira issue type.** Cases inherit Jira's workflow,
  permission, comment, attachment, and notification surfaces.
  Everything that works for Jira issues works for cases.
- **Native linking to requirements / stories.** Because everything
  is a Jira issue, the link between a test and the story it covers
  is the same kind of link as any other Jira issue link. Reporting
  on coverage is just JQL.
- **Test execution as an issue.** A test run produces issues; the
  pass/fail history of a test is queryable as the history of its
  execution-issue lifecycle.
- **Gherkin support inline.** Step definitions can be Gherkin-shaped
  for BDD workflows, with automated execution hooks.

### What we should steal verbatim

- **Story-test edge as a first-class graph relationship** → Hero
  already has this from L8 (story lifecycle overlay) + L27 (cross-
  kind relationships). Xray validates the shape — make sure Hero's
  edge model is at least as expressive as Xray's "test covers
  issue" link.
- **Treat test executions as queryable events, not transient runs**
  → Hero's hybrid storage (L17): last-N runs cached in graph; full
  history in integration. Xray's model says yes, executions are
  events.
- **JQL-style coverage queries** → Hero must support "find all
  stories in this sprint without coverage" as a first-class graph
  query. Probably belongs to a new skill like `coverage-query`.

### What to leave behind

- **Configuration heaviness.** Xray ships with ~20 issue types
  and a multi-layer hierarchy (Test Plan → Test Set → Test → Test
  Execution → Test Run). For most teams this is too much. Hero's
  L24 (kinds, not configs) and L25 (one inbox) cut against the
  Xray instinct to multiply types.
- **Jira-bound everything.** Xray is unusable without Jira. Hero
  must be standalone-capable (L28).
- **No standalone reporting outside Jira.** Xray's dashboards are
  Jira gadgets. Hero ships dashboard views via the view registry,
  no host system required.

### Screens to study

- A Jira story with an attached Xray test panel — showing linked
  tests, coverage state, execution status. Note how the panel
  *embeds* in the issue without leaving the Jira frame.
- The Xray test execution view (run a series of cases, mark each).
  Compare execution rhythm against TestRail.
- The Xray report widget on a Jira dashboard (coverage by epic /
  release). This is the closest prior-art to Hero's release-gate
  aggregation.

---

## Zephyr Scale (and Zephyr Squad)

**Influences:** Test case editor (secondary); Run history (secondary).

Sister tool to Xray, also Jira-app, different vendor and slightly
different shape. Zephyr Scale (formerly Adaptavist) is the more
common variant in mid-to-large orgs. Worth studying briefly because
it's often the *alternative* Jira shops chose over Xray, and the
reasons reveal real preferences.

### What they actually do well

- **Cases live in their own tree separate from Jira issues.** Unlike
  Xray (where tests *are* issues), Zephyr Scale keeps a case
  hierarchy separate from Jira issues. Coverage links are explicit.
  Some teams prefer this because tests don't pollute the issue
  backlog.
- **Versioning per case.** A case can have multiple versions; runs
  pin to a specific version. Audit trail is intact.
- **Bulk operations.** Multi-select with edit/move/delete in the
  case tree. Important affordance Hero's unified inbox needs to
  match.

### What we should steal verbatim

- **Case versioning with run pinning** → Hero Test case editor
  should preserve version history; runs reference a specific
  version. Important for regulated industries and any team that
  wants to know "did the case change between this passing run and
  this failing one?"
- **Bulk operations in the case tree** → Hero unified inbox + Test
  plan editor: multi-select, bulk edit / move / delete / tag.

### What to leave behind

- **Separate tree forces context switching.** Tests are off in their
  own world; you don't see them while looking at the story. Hero's
  embedded inbox panels (L29) explicitly fix this.

### Screens to study

- The Zephyr Scale case tree view inside Jira. Note the
  multi-select density.
- A Zephyr Scale test cycle view (their equivalent of a TestRail run).
  Compare nomenclature.

---

## qTest (Tricentis)

**Influences:** Compliance / audit angle (light); Bulk operations
(secondary).

qTest is enterprise / regulated-industry leaning. Healthcare, finance,
defense use it for the audit trail and traceability matrices. Less
relevant to mid-market Hero buyers, but worth understanding for the
compliance shape.

### What they actually do well

- **Traceability matrix.** A real matrix view (rows = requirements,
  columns = tests, cells = coverage state) that prints to a
  compliance-shaped report.
- **Audit log immutability.** Every action is logged forever;
  reports show who changed what when.
- **Defect-test-requirement triple linking.** First-class
  relationships across all three, with reporting that walks the
  chain.

### What we should steal verbatim

- **Traceability matrix as a view** → Hero release-gate view's
  expansion: a coverage matrix showing story × case × pass-state.
  This is what audit-conscious teams print.

### What to leave behind

- **Enterprise UI density and ceremony.** qTest's workflow editor,
  custom field builder, and approval gates are over-engineered for
  Hero's target buyer.
- **Compliance-as-feature.** Hero gets traceability for free via
  the graph; we don't ship audit dashboards in v1.

### Screens to study

- A qTest traceability matrix view. Note that the rows/columns/
  cells layout is what makes it printable.

---

## PractiTest

**Influences:** Filter/saved-view UX (secondary).

Smaller market share than the above but distinctive UX worth a
specific note. PractiTest's filter system is unusually strong and
worth borrowing.

### What they actually do well

- **Hierarchical filters with shareable URLs.** A filter is a
  composable expression; URL-encodable; shareable; nestable. Saved
  filters behave like views.
- **Customizable list views per role.** A QA lead sees different
  columns than an automation engineer; same data.

### What we should steal verbatim

- **Filter-as-URL composability** → Hero unified inbox + Story
  queue: filter state encoded in URL, shareable, savable as a
  named view in the left nav.

### Screens to study

- The PractiTest filter sidebar with multiple nested expressions
  applied to the case grid.

---

# Part II — AI-native QA players (where the puck is going)

These are the tools shaping QA expectations in 2026. Hero QA must
*at minimum* match their AI authoring + execution quality where it
overlaps. Where Hero is structurally different (spec-shaped,
graph-backed, cross-domain), the AI-native players validate the
direction without replicating the model.

> **Verification note:** the AI-native QA landscape evolves fast.
> Treat specific feature claims below as a snapshot. The designer
> should re-verify current state before deep mockup-design from
> any single vendor.

---

## mabl

**Influences:** Test authoring UX (primary); Auto-healing concept (secondary); Flaky tests backlog (primary).

mabl is the dominant AI-first end-to-end testing tool. Their
authoring is "record a flow, the AI watches, generates a test that
heals itself." For Hero's purposes, mabl is the gold standard on
"AI authoring at sprint speed."

### What they actually do well

- **Auto-healing locators.** When a UI element moves or renames,
  the test still passes because the AI re-finds the element by
  semantic / visual signal. Reduces flake at the locator layer
  dramatically.
- **Plain-English step library.** "Click the submit button" — not
  CSS selectors. Cases read like a QA tester wrote them, because
  one did.
- **Auto-suggested cases from user flows.** Watching real user
  sessions, mabl proposes "you don't have a test for this flow."
  Coverage-gap detection through behavioral data.
- **Flaky test classification.** Each flaky run is classified
  (environment / locator / data / true bug) automatically. The
  "verdict" workflow Hero locked in L11 is exactly this.

### What we should steal verbatim

- **Plain-English step authoring as the default** → Hero `test-author`
  generates cases in plain English (matching L19's step-by-step
  default). Locators and code are an implementation detail; the
  case-as-document is what humans review.
- **Flaky-run classification with verdict** → Hero `qa-flake-curator`
  agent should produce a verdict per flaky case (test-issue /
  environment / true-bug — matching L30's triage outcomes for
  test issues). This is more than a "list of flaky tests."
- **Coverage-gap detection from upstream signal** → Hero `qa-investigator`
  surfaces "story has no test cases" ambient cards (already
  designed). Extend: when test execution data exists, also surface
  "this user flow has no covering case."

### What to leave behind

- **End-to-end UI testing only.** mabl is browser-flow shaped. Hero
  QA is *any* test type (manual, automated, API, exploratory). Our
  authoring agent must support all formats (L19).
- **Black-box AI.** mabl's heuristics are opaque. Hero's `test-author`
  must show *why* a case was generated (which AC it derives from)
  so QA can audit and trust.
- **No spec-shape.** mabl tests live in mabl's database. Hero tests
  are markdown specs with frontmatter, graph-linked, version-
  controllable. That's the Hero difference.

### Screens to study

- The mabl test-editor view showing a recorded flow with plain-English
  steps. Note the readability of the step list and how each step
  shows the screenshot / element it acts on.
- The mabl flaky-test dashboard with verdict labels per failure.
- The mabl coverage-suggestions view (if accessible) showing
  proposed tests from user flows.

---

## Functionize

**Influences:** NLP-driven test authoring (primary).

Functionize is the "write your test in English" pioneer. You type
"Log in as admin, navigate to settings, change theme to dark,
verify it persists across reload" and it produces an executable
test.

### What they actually do well

- **Natural-language test generation.** A short paragraph becomes
  a multi-step test with assertions. The author doesn't write any
  syntax.
- **Smart maintenance.** When the app changes, the AI re-interprets
  the natural-language test against the new UI and patches itself.

### What we should steal verbatim

- **Natural-language test generation as one input mode** → Hero
  `test-author` accepts plain-English prompts ("write a test for
  the password reset flow") in addition to AC-grounded generation.
  This is the "ad-hoc test" path for cases that aren't tied to a
  story.

### What to leave behind

- **Generation without traceability.** Functionize generates from
  natural language, not from acceptance criteria. The case
  doesn't trace back to *why* it exists. Hero's AC-grounded
  generation (`test-author` reading EARS-shaped AC) is the
  defensible alternative.

### Screens to study

- The Functionize "write a test in English" composer. Note the
  density of the resulting case after generation.

---

## Testim (now part of Tricentis)

**Influences:** Stability focus / locator quality (secondary).

Testim's pitch is stability through AI locators. Less about
authoring, more about runs not breaking.

### What they actually do well

- **Layered locators per element.** Each interactive element has
  multiple AI-derived locators (ID, semantic role, visual signature,
  DOM neighborhood). A test passes if any locator resolves.

### What we should steal verbatim

- **Stability scoring on individual cases** → Hero `regression-curator`
  weights stability in regression promotion scoring (L22). A case
  that flakes often is a bad regression candidate.

### What to leave behind

- **Code-shaped output.** Testim produces JavaScript. Hero outputs
  spec-shaped markdown cases.

### Screens to study

- The Testim stability score widget on a test card.

---

## Applitools

**Influences:** Visual diff as an assertion type (secondary).

Applitools is the visual-AI play. "The button looks different than
last run" is a kind of test assertion they invented as a category.

### What they actually do well

- **Visual diff with semantic tolerance.** Pixel-perfect diff is
  noisy; Applitools' AI ignores anti-aliasing, scrollbar variation,
  ad placement — only flags meaningful visual change.
- **Region-of-interest masking.** Mark regions as "ignore" — useful
  for dynamic content.

### What we should steal verbatim

- **Visual assertion as a case format** → Hero Test case editor's
  format list extends with `visual-baseline` (L19 lists 4 formats;
  this could be a v2 addition). Format selection per case is
  granular.

### What to leave behind

- **Visual testing as the whole product.** Hero is broader; visual
  is one assertion type.

### Screens to study

- An Applitools visual-diff result view showing baseline vs
  current with regions highlighted.

---

## QA Wolf

**Influences:** Outsourced-QA-as-platform thinking (light); coverage
gap detection (secondary).

QA Wolf is a hybrid: AI plus humans-in-the-loop. Customers pay for
"100% E2E coverage" with QA Wolf running the authoring + maintenance.
Relevant because it answers "what would QA look like if AI did
80% and humans did 20%?"

### What they actually do well

- **Coverage commitment.** They quantify "we cover this percent of
  your user flows" and report against it.
- **Per-customer coverage dashboards.** Buyer sees what's tested,
  what's flaky, what's planned.

### What we should steal verbatim

- **Coverage as a *quantified, visible* commitment, not a vibe** →
  Hero release-gate view shows coverage % per story and per release.
  Story queue with coverage signals (default landing) shows the
  per-story bar at a glance.

### What to leave behind

- **The "we run it for you" model.** Hero is a product, not a
  service. Customers operate it.

### Screens to study

- The QA Wolf customer dashboard showing coverage % per area and
  per release.

---

## Reflect / TestSigma / Katalon Studio

**Influences:** Codeless authoring UX (light).

These three are codeless-authoring tools with varying AI depth.
Reflect is browser-recorder + cloud. TestSigma is more API-broad.
Katalon Studio is a desktop IDE with AI bolt-ons. None of them
reshape the Hero design materially; all of them validate that
plain-English / record-replay authoring is the mainstream
expectation for new QA tools.

### What we should steal verbatim

- **Record-mode as a future authoring input** → Hero v2 could
  accept browser-recording sessions as input to `test-author`
  (output: spec-shaped cases derived from the recording).
  Out of v1 scope.

---

# Part III — Agent / framework prior art

PM's `agent-pack-design.md` already studied wshobson/agents,
MetaGPT, BMad-Method, and ChatPRD. The QA pack inherits those
studies. This section captures **what is QA-specific** that PM's
study didn't cover.

---

## BMad-Method — QA persona (deep)

PM's brief covered BMad's six-persona pipeline (BA → PM → PO →
Architect → Developer → QA) and the Orchestrator. The QA persona
is the *last* in the pipeline and least documented. Worth a
dedicated dive.

### What the BMad QA persona does

- Receives a story handoff with: PRD, acceptance criteria,
  implementation summary, branch reference.
- Produces: test plan (acceptance test, edge case, negative
  case, regression risk), executes mentally or via tooling,
  produces verdict (pass / fail with findings).
- Handoff: back to Developer if fail, to release-coordinator if
  pass.

### Patterns to adopt

- **Sequential pipeline with explicit handoff payload** → Hero
  QA's `qa-delivery-lead` receives a story handoff payload that
  includes PRD, AC, linked feature spec, branch reference (if
  set), and prior QA history for the story.
- **Verdict-shaped output** → QA agents produce verdicts (pass
  with notes / fail with three-action composer state /
  request-clarification) rather than free-form output.

### Patterns to leave

- **Verbatim handoff prompts as strings.** BMad encodes handoffs
  as prompt strings; Hero encodes them as graph edges + structured
  payload. Same idea, better mechanism.

---

## wshobson/agents — QA cluster

PM's brief noted wshobson has 185 agents and *no* dedicated PM
agent. The QA picture is more populated.

### Notable QA-ish agents in wshobson

- `test-engineer` — general-purpose; writes unit tests, integration
  tests; code-shaped output.
- `qa-engineer` — heavier on test plan generation; reads requirement
  docs.
- `e2e-testing-specialist` — browser-flow shaped, similar to mabl's
  approach.
- `test-automation-architect` — strategy-level, picks frameworks
  and patterns.

### Pattern to adopt

- **Multiple specialized QA agents > one mega-agent.** The
  catalog's instinct to split test-engineer / qa-engineer /
  e2e-testing-specialist / test-automation-architect is right.
  Mirror in Hero with `test-author` / `qa-strategist` /
  `regression-curator` / `flake-curator` rather than one
  `qa-engineer` that does everything.

### Pattern to leave

- **Code-shaped output by default.** wshobson agents tend to write
  test code. Hero's primary output is *spec-shaped cases* —
  documents that humans review. Code generation is downstream.

---

## QualityAgent (open-source mini-projects)

Several smaller open-source projects (search GitHub for
"qa-agent", "test-author-agent", "test-generation-agent") implement
LLM-as-test-author. Common patterns:

- Read source code + requirement doc, output Gherkin or
  step-by-step test cases.
- Use embedding similarity for "find existing test that covers
  this requirement."
- Verdict-shaped review of generated test against the requirement.

### Pattern to adopt

- **Embedding-based existing-coverage lookup** → Hero's
  `coverage-budgeting` skill: before generating new cases for a
  story, embed-search the existing regression suite for cases
  that already cover similar AC. Reduces redundant generation.

---

## Methodology sources

These are not products. They are bodies of knowledge that shape
**how `test-author` generates cases** and **how `qa-investigator`
surfaces gaps**. Skills load from these.

### ISTQB foundation — test design techniques

The standardized vocabulary for case derivation:

- **Equivalence partitioning** — group inputs into equivalence
  classes; one case per class.
- **Boundary value analysis** — test at and around boundaries
  (off-by-one, min/max, edge of ranges).
- **Decision table testing** — for cases with multiple input
  conditions; matrix combinations.
- **State transition testing** — for stateful flows; transitions
  and invalid transitions.
- **Use-case-based testing** — derive cases from user-flow
  descriptions.

These map directly to **`ears-test-derivation` skill** (the AC →
case generation engine):

- Each EARS AC implies an equivalence class (the trigger condition).
- Boundary value cases come from constraints in the AC ("must be
  positive integer" → cases at 0, 1, max).
- Decision-table format is auto-selected when AC has multiple
  conjunctive conditions.
- State-transition format is auto-selected when AC describes a
  state machine.

### Heuristic test strategy (Bach / Bolton / Bolton — context-driven testing)

The opposite of ISTQB-shaped formal derivation: heuristic
exploratory testing.

- **Charter-driven exploration.** Time-boxed, mission-shaped
  exploration with notes (Session-Based Test Management).
- **Heuristic mnemonics.** SFDIPOT (Structure / Function / Data /
  Interfaces / Platform / Operations / Time), CRUSSPIC STMPL,
  and friends.

These shape **`exploratory-charter` skill** loaded under the
`exploratory-charter` test methodology preset. `test-author`
under this preset generates *charters* (mission + heuristics +
note template), not cases.

### Risk-based testing (Hendrickson)

- **Coverage prioritization by risk.** Test what would hurt most
  if it broke.
- **Risk matrices** — likelihood × impact per area.

Shapes **`risk-scoring` skill** loaded under `risk-based` preset.
`coverage-budgeting` consults this skill to weight per-story
authoring budget by risk.

### Agile testing quadrants (Gregory / Crispin)

The four quadrants:
- Q1: unit / component tests, technology-facing, supporting
  development.
- Q2: functional / story tests, business-facing, supporting
  development.
- Q3: exploratory / usability, business-facing, critique.
- Q4: performance / security / reliability, technology-facing,
  critique.

Shapes the **case-type taxonomy** Hero `test-author` produces.
Each case has an implicit quadrant tag. Coverage dashboards can
filter by quadrant.

### Janet Gregory / Lisa Crispin — More Agile Testing

The "whole team approach" to quality — every role contributes to
test thinking, not just QA. This is the philosophical anchor for
**Hero QA's silo-tearing patterns** and the **`Suggest new story`
button** (engineer-PM-QA loop, not isolated handoffs).

---

# Consolidated QA design philosophy

Distilled from the research, the Hero QA UX worldview is:

- **Coverage is a delivery gate, not a parallel workstream.**
  Pivotal/Linear treat QA as a state in the work pipeline (Deliver
  → Accept). Hero QA *strengthens* that by inserting `qa-ready`
  and `qa-rejected` states with substance (real coverage signal,
  real rejection composer). Coverage isn't optional decoration.

- **AI authors. Humans shape.** mabl / Functionize / Reflect prove
  the market expects AI to do the typing. Hero's job is to ensure
  what the AI types is *defensible* — derived from acceptance
  criteria, attributed to the AC, reviewable case-by-case. Generation
  without traceability is what we leave on the table.

- **Three case-format defaults, one mental model.** Step-by-step
  (TestRail), Gherkin (Xray/BDD shops), decision-table (input-heavy
  domains), data-driven (parameterized). Same underlying case
  spec; format is render-layer. Don't force a single format on
  every team.

- **Run state belongs in the integration. Authoring belongs in
  Hero.** TestRail / Xray are excellent execution systems. Hero
  doesn't try to be one. Hero is the spec layer + the synthesis
  + the AI authoring; the integration is the system of record for
  what actually ran when.

- **Test issues triage to four real outcomes.** Bad-test /
  reject-linked-story / raise-as-new-bug / flag-as-regression.
  Every QA tool today asks "is this a bug?" Hero asks the four
  questions and routes accordingly. This is structural improvement.

- **Flakiness is an opinionated backlog, not a list.** mabl's
  flaky verdict is right. Tracking flake without driving it
  toward zero is what every legacy tool does. Hero's stance:
  every flaky test has a verdict and a deadline.

- **Cross-kind links are the unification.** TestRail and Jira and
  Xray and GitHub Issues live in different houses. Hero doesn't
  merge them into one schema (Z-1 lesson); Hero links them by
  relationship and renders both faces. "Same lens, source
  fidelity preserved."

- **Local-first means the pack is usable on day one.** No team
  should hit "install QA pack" and see an empty inbox because
  they haven't wired TestRail yet. Engineering bugs from the
  active workspace render. Native QA defects (opt-in) render.
  Integration is decoration.

- **The standalone case is the design test.** When designing any
  QA screen, draw it twice: once with TestRail/Xray wired in
  (decorated), once without (pure). If the no-integration version
  looks empty or broken, the design is wrong.

---

# Anti-pattern list

Things to explicitly *not* do, with the tool that warns us off
each:

1. **Project-trapped silos.** (TestRail, Xray) Tests in one project
   can't reference stories in another. Hero's graph crosses
   projects and packs natively.
2. **Workflow editor as a feature.** (qTest, Jira) Ship presets,
   not a state-machine designer. Configurability through
   `qa.gate_style`, `qa.test_issue_persistence`, `qa.rejection_strictness`,
   not through a workflow UI.
3. **Forced full taxonomy.** (Xray, qTest) Test Plan → Test Set →
   Test → Test Execution → Test Run is too many levels. Hero ships
   Test plan + Test case + Regression suite + Release gate. Four
   types, real lifecycles.
4. **Generation without traceability.** (Functionize, ChatGPT-as-author)
   Cases that don't trace back to *why they exist* rot. Hero's
   `test-author` ties every case to an AC + story via graph edges.
5. **Code-shaped output by default.** (wshobson test-engineer,
   Testim) The case-as-document is the reviewable artifact. Code
   is a downstream concern.
6. **End-to-end-only thinking.** (mabl, Reflect) Browser flow is
   one test type. Hero ships step / Gherkin / decision-table /
   data-driven and treats them as peers.
7. **Opaque AI heuristics.** (mabl auto-heal, Testim locators)
   When a case is generated, the user sees *which AC* the case
   derives from. When a test is healed, the user sees *what
   changed*. No black boxes.
8. **Run-setup ceremony.** (TestRail run creation, qTest cycles)
   In the common case (per-story per-sprint authoring) the user
   never picks suites, sections, configurations. They click
   `Author cases` and review.
9. **Defect/bug as a forced split.** (Every test-management tool)
   By default, QA-found problems are coverage gaps on the story
   or engineering bugs via `/diagnose`. Separate defect lifecycle
   is opt-in (L6).
10. **Compliance UI as a v1 feature.** (qTest) Hero gets traceability
    for free via the graph. Audit dashboards wait for buyers who
    actually ask.
11. **Configurability through field editors.** (qTest, PractiTest)
    Custom fields per project lead to chaos. Hero's fields are
    spec-type-registry declared; preset config is the
    configurability surface.
12. **Forced AI-authoring on every case.** Even when team prefers
    to write manually, manual must be a first-class entry. AI is
    the *fast path*, not the only path.

---

# Screen-influence map

| Hero QA screen | Primary influences | Secondary influences |
|---|---|---|
| Story queue with coverage signals (default landing) | Pivotal Tracker (single-list flow, in-card state buttons from PM brief), Linear (J/K, density) | QA Wolf (coverage % visible), Xray (story-test edge rendered inline) |
| Test plan editor (coverage matrix) | TestRail (case grid + step rows), qTest (traceability matrix as a printable view) | Xray (story-link panel embedded), Functionize (NL-input composer for "ad-hoc" cases) |
| Test case editor (format-aware) | TestRail (default step grid), Xray (Gherkin support inline), Cucumber/Specflow (Gherkin canonical), Decision-table format conventions (ISTQB) | mabl (plain-English step library), Testim (stability score badge) |
| Story view in `qa-ready` (three-action rejection composer) | Pivotal Tracker (Reject as a first-class state action), Linear (right-rail action density, status pills) | Height (ambient AI panel for finding categorization), BMad QA persona (verdict-shaped output) |
| Release-gate view | qTest (traceability matrix), QA Wolf (coverage commitment dashboard), TestRail (milestone roll-up) | Linear (cycle view aggregation), Xray (JQL-style coverage queries) |
| Flaky tests backlog | mabl (verdict classification per flaky test) | Testim (stability score), GitHub Issues triage UX (verdict-shaped queue) |
| Unified inbox (kind tabs) | Linear (Triage view shape, command palette, source pills from PM brief), Productboard (source-tagged inbox from PM brief) | PractiTest (filter-as-URL composability), Zephyr (bulk operations) |
| Cross-domain handoff stream (QA-side) | Linear (cycle view timeline from PM brief), Kanbanize (cycle-time histogram from PM brief) | Aha! (strategic-context strip from PM brief), Xray (test-execution events as graph nodes) |

---

# What is genuinely new in Hero QA

To call out what does **not** appear in any prior art studied, and
is therefore Hero QA's design-original contribution:

1. **Cross-pack lifecycle overlays.** No test-management tool extends
   another tool's story lifecycle with QA states. Xray comes closest
   by living inside Jira, but the states are Jira's, not Xray's.
   Hero's `qa-ready` / `qa-rejected` states are *injected* into PM's
   story type at runtime. (L8, L12, L16.)
2. **Three-action rejection composer.** No QA tool I found
   distinguishes "Add AC" / "Suggest new story" / "Reject as quality
   issue" as primary actions at rejection time. Every tool collapses
   these into a single rejection workflow that bleeds scope through
   the gate. (L9, L18.)
3. **AC-grounded case derivation.** mabl / Functionize generate from
   recordings or natural language. wshobson generates from
   requirement docs. None mechanically derive cases from EARS-shaped
   acceptance criteria with one-case-per-AC defaults. (L21, ISTQB
   skill loading.)
4. **Test-issue triage with four primary actions.** mabl classifies
   flaky runs (closest prior art) but doesn't route to bug-vs-story-
   rejection-vs-regression. Hero's four-outcome triage is novel.
   (L30.)
5. **Coverage-as-story-gate (inline mode default).** Every QA tool
   surfaces coverage; none make it a story-completion gate by
   default. (L5, L13.)
6. **Local-first with seamless hybrid escalation.** Most QA tools
   require their backend; the AI-native ones require their cloud.
   Hero ships a pack that works on a laptop with no integration and
   gains decoration when one is wired. (L3, L28.)

These six are the Hero QA design-original contributions worth
naming in launch positioning and demo storytelling.
