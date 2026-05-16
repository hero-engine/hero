# Hero QA — Agent / Skill / Command Pack Design

Audience: the agent that will eventually `/deliver hero-qa` (and the
human reviewing this before delivery). This document specifies *what
exists* in the QA domain pack at the agent / skill / command layer —
not how each agent is prompted (those files get drafted at delivery
time and live under `domains/qa/agents/`, `domains/qa/skills/`,
`domains/qa/commands/`).

Sibling files set the stage and must be read first:
- `spec.md` — strategic frame, locks (32), artifact types, gate-style
  preset, cross-pack interactions, dashboard views.
- `research-brief.md` — competitive UX research (steal/leave matrix)
  across TestRail, Xray, mabl, Functionize, and methodology sources;
  plus six design-original Hero QA contributions.
- (Pending) `mockup-brief.md` — eight killer screens.

The locked UX from `hero-pm` (IDE-style left-nav singletons + center
tabbed openables + bottom strip = verbs + opt-in right rail with
inline-proposed outputs) is taken as given (L1 in spec). Every
agent / command / button in this design must fit that UX.

The locked design voice from the spec is **opinionated defaults +
dialed configurability**. Agents ship strong defaults; presets carry
the variance.

---

## A) Prior-art appendix

This section synthesizes the prior art that **specifically informs
the QA agent designs** below. The full competitive treatment with
steal/leave matrices lives in `research-brief.md`; this section is
the bridge from research to agent shape.

### A.1 Carries forward from PM's prior art

PM's `agent-pack-design.md` studied wshobson/agents, MetaGPT,
BMad-Method, and ChatPRD. Those studies apply to QA too because the
shape conventions (tight role scoping, single-output-artifact
discipline, Action → Watch pattern, role/profile/goal/constraints
triple) are domain-agnostic. The QA pack reuses PM's agent shape
spec without restating it.

What's QA-specific from those studies:

- **BMad-Method's QA persona.** Sequential pipeline with explicit
  handoff payload (PRD + AC + branch + history); verdict-shaped
  output (pass with notes / fail with findings / request
  clarification). Hero QA's `qa-delivery-lead` adopts this handoff
  payload shape.
- **wshobson's QA cluster.** Multiple specialized QA agents
  (test-engineer / qa-engineer / e2e-testing-specialist /
  test-automation-architect) outperform one mega-agent. Hero QA
  splits along authoring / strategy / curation / triage axes.
- **No prior agent system in the public domain ships an
  AC-grounded test-derivation engine.** Functionize / mabl
  generate from recordings or NL; wshobson generates from
  requirement docs. Hero QA's `test-author` + `ears-test-derivation`
  skill is design-original.

### A.2 New for QA — test-tooling AI

The `research-brief` covered mabl / Functionize / Testim / Applitools
in depth. The specific agent-design takeaways:

- **mabl's flaky-run verdict classification** → `qa-flake-curator`
  produces a verdict per flaky case (test-issue / environment /
  true-bug), matching test-issue triage L30.
- **mabl's plain-English step library** → `test-author` generates
  plain-English step output by default (L19 step format).
- **Functionize's NL-to-test composition** → `test-author` accepts
  natural-language prompts as one input mode (alongside AC-grounded
  generation).
- **Testim's stability scoring** → `regression-curator` consumes
  per-case stability score in promotion ranking (L22).
- **QA Wolf's coverage % commitment** → `coverage-strategist`
  surfaces quantified coverage per story and per release on the
  Story queue and release-gate views.

### A.3 Methodology grounding

The agent designs explicitly load from ISTQB foundation, Bach/Bolton
context-driven testing, Hendrickson risk-based testing, and
Gregory/Crispin agile-testing-quadrants. Skills are the carriers
(see D.3 below). Agents that need methodology coaching load these
skills on demand rather than hard-coding philosophy. This mirrors
PM's pattern (`continuous-discovery-cadence` is a skill, not a
"discovery coach agent").

### A.4 Synthesis — what makes a QA agent good

From the combined study:

- **Verdict-shaped output.** A QA agent's output is judgable —
  pass/fail/needs-info — not free-form.
- **Traceability to AC.** Anything the agent produces (a case, a
  rejection finding, a regression flag) ties back to specific
  acceptance criteria via graph edge or content reference.
- **Format-aware authoring.** The same case agent renders four
  formats (step / Gherkin / decision-table / data-driven) without
  changing its derivation logic.
- **Surface, don't decide.** Triage agents surface options
  (the four-action composer for test-issue triage; the three-action
  composer for story rejection). The human picks. Agents that
  silently decide route human work into dead ends.
- **Active backlogs, not passive lists.** `qa-flake-curator` and
  `regression-curator` drive metrics toward a goal (flake → zero,
  regression coverage → 90%). Agents that just list are
  insufficient.

---

## B) PM reference summary

The QA pack inherits structural conventions from the PM pack. This
section names them so this document doesn't re-derive shape.

PM ships **27 agents, 32 skills, 22 commands**. The agent shape,
skill shape, and command shape are all reused — see PM's
`agent-pack-design.md` §B for the canonical specs. New conventions
this pack adds:

### What transfers directly to QA

- Frontmatter + permission maps.
- "You are X" + numbered process + rules + default-output.
- Skill template (what / when / core approach / anti-patterns).
- Command-as-router pattern.
- Inline-proposed output mode (PM-introduced for `propose-inline`
  on authoring agents — used heavily here for `Author cases`).
- Contextual-button invocation pattern (button fires command with
  `--inline-propose` flag).
- Methodology preset awareness — `qa-preset-detection` skill reads
  `hero.json` `qa.presets` (parallel to PM's `pm-preset-detection`).
- Cross-domain handoff via graph edges + `handoff-coordinator`
  agent (overload with QA-specific edge kinds).

### New for QA (beyond PM)

- **Lifecycle-overlay awareness.** QA agents that read or write
  `story` state must know about `qa-ready` and `qa-rejected`
  overlay states. Skill: `lifecycle-overlay-awareness`.
- **Cross-pack body augmentation.** `qa-reject-story` writes a
  collapsible `## QA Findings` block into a PM story's body
  with a QA gutter icon for authorship clarity (L15). New skill:
  `cross-pack-body-augmentation`.
- **Verdict-shaped output discipline.** QA agents output verdicts
  more often than PM agents do; see A.4.
- **Run-state coupling.** QA agents read run state from the
  integration (TestRail/Xray) when configured. Skill:
  `integration-run-state-reader`.
- **Normalization layer awareness.** Inbox-rendering agents read
  source fields and normalize via the standard schema (L23). Skill:
  `normalization-mapping`.

### Quantitative target

QA's pack should match PM's depth without exceeding it. Target
shape:

| | Engineering | PM | QA target |
|---|---|---|---|
| Agents | 34 | 27 | 23 |
| Skills | 45 | 32 | 26 |
| Commands | 27 | 22 | 18 |

QA is slightly leaner than PM because:
- No discovery / strategic tier as deep as PM's (no QA equivalent
  of `product-strategist` / `discovery-researcher` — those belong
  to PM and QA reuses them via cross-domain command routing).
- Authoring tier is denser (5 agents) because format-specific
  authoring is QA's main muscle.

---

## C) Agent roster (23 agents)

Each agent is documented at the same depth as PM: role, when
invoked, what produced, skills, delegation graph, prompt sketch,
prior-art attribution. P0/P1/P2 priority calls are at the
section level.

### C.1 Coordination tier (1)

#### `qa-delivery-lead` — P0
- **Role:** The pack's orchestrator. Coordinates specialists for
  any QA-shaped task — coverage planning, case authoring, triage,
  release-gate evaluation. Reads the active `gate_style` preset
  and routes accordingly.
- **When invoked:** `/qa` natural-language dispatch; bottom-strip
  buttons that don't have a single specialist owner; any time
  the user asks "what should we do here" inside a QA artifact.
- **Produces:** routing decisions, multi-agent coordination plans,
  pack-level summaries. Does not write artifacts directly —
  delegates to authoring/triage/curation agents.
- **Skills:** `qa-preset-detection`, `lifecycle-overlay-awareness`,
  `verdict-output`, `kickoff-prompt`, `context-injection`.
- **Delegates to:** every C.2–C.8 agent below.
- **Prompt sketch:** "You are the QA pack's delivery lead. You
  read the user's request, the active artifact's state, and the
  team's active QA presets, and you route to the right specialist.
  You produce verdicts and coordination plans, not artifacts.
  When the user's intent is ambiguous, you ask one focused
  question instead of guessing."
- **Prior art:** PM's `pm-delivery-lead`; engineering's
  `feature-delivery-lead`; BMad's Orchestrator.

### C.2 Strategic tier (3)

#### `qa-strategist` — P0
- **Role:** Frame test strategy for a release or sprint — risk
  areas, coverage gaps, integration vs E2E vs manual split,
  what to defer. Owns "what should we test and why."
- **When invoked:** `/plan-coverage`; release-gate creation;
  start of a sprint; "what's our test plan for X" natural language.
- **Produces:** test-strategy section on a `test-plan` spec;
  risk-and-coverage notes on roadmap-items / epics; sprint
  coverage budgets.
- **Skills:** `risk-based-testing`, `agile-testing-quadrants`,
  `coverage-budgeting`, `release-readiness-framing`,
  `qa-preset-detection`.
- **Delegates to:** `coverage-strategist`, `test-author` (downstream).
- **Prompt sketch:** "You are a senior QA strategist in the
  Hendrickson / Gregory tradition. You frame test strategy in
  terms of risk and outcome, not coverage percent. You say what
  *not* to test as clearly as what to test. You make the
  cost-of-defect-escape visible per area."
- **Prior art:** Hendrickson on risk-based testing;
  Gregory/Crispin's agile testing quadrants; PM's
  `product-strategist` shape.

#### `coverage-strategist` — P0
- **Role:** Quantify and surface coverage at the per-story,
  per-sprint, per-release level. Drives the "is this story
  covered" gate signal.
- **When invoked:** Story view ambient ("story has 5 AC, 0
  cases"); Story queue rendering (coverage signal per row);
  release-gate aggregation; `/coverage` command.
- **Produces:** coverage budget reports, ambient coverage cards
  on stories without cases, coverage % on release-gate.
- **Skills:** `coverage-budgeting`, `coverage-gap-detection`,
  `quadrant-tagging`.
- **Delegates to:** `test-author` (for actual case generation),
  `regression-curator` (for "what's covered by existing
  regression suite").
- **Prompt sketch:** "You are a senior coverage analyst. You
  quantify coverage as a function of acceptance criteria
  satisfied by passing cases — not test count. You distinguish
  feature coverage from regression coverage. You name the
  uncovered cases as specifically as the covered ones."
- **Prior art:** QA Wolf's coverage-commitment dashboards;
  Hendrickson's risk-coverage matrix.

#### `release-readiness-strategist` — P1
- **Role:** Aggregate sprint coverage, regression suite state,
  open blockers into a Go / No-Go verdict per release.
- **When invoked:** release-gate spec creation/refresh; "are
  we ready to ship" natural language; `/release-gate` command.
- **Produces:** release-gate spec body, Go/No-Go state, named
  blocker list, deferral recommendations.
- **Skills:** `release-readiness-framing`, `blocker-policy-evaluation`,
  `verdict-output`.
- **Delegates to:** `coverage-strategist`, `regression-curator`,
  `qa-investigator`.
- **Prompt sketch:** "You are a senior release readiness analyst.
  You produce a Go / No-Go verdict against the team's configured
  blocker policy. You name the blockers specifically. You
  distinguish ideal-state from acceptable-state, and you say
  which blockers warrant a release-hold versus a punt-to-next-
  cycle."
- **Prior art:** qTest milestone roll-ups; TestRail milestone
  view; Hero's release-engineer agent.

### C.3 Authoring tier (5)

#### `test-author` — P0
- **Role:** The case-authoring engine. Generates cases from AC
  (default), from natural-language prompts (Functionize-style),
  or from recorded flows (future). Output is *spec-shaped*, not
  code-shaped.
- **When invoked:** `Author cases` button on a story; `/author-cases`
  command; `Plan sprint coverage` follow-on; ambient
  "story has no cases" card acceptance.
- **Produces:** `test-case` specs (one per generated case);
  inline-proposed cases in the story view (with accept/edit/reject
  per case).
- **Skills:** `ears-test-derivation`, `step-by-step-authoring`,
  `gherkin-authoring`, `decision-table-authoring`,
  `data-driven-authoring`, `quadrant-tagging`,
  `existing-coverage-lookup`.
- **Delegates to:** `duplicate-detector` (overload from PM) for
  "is this case redundant with existing regression suite";
  `seam-requester` when a case can't be exercised without
  engineering support.
- **Prompt sketch:** "You are a senior test author. You derive
  cases from acceptance criteria using ISTQB techniques —
  equivalence partitioning, boundary value analysis, decision
  tables, state transitions. Every case traces back to a
  specific AC. You produce one case per AC by default, or
  exhaustive happy+edge+negative when asked. You never
  hallucinate cases that don't tie to an AC; if AC is too vague,
  you flag rather than invent."
- **Prior art:** ISTQB foundation techniques; mabl's plain-English
  step library; Functionize's NL composition; design-original on
  AC-grounded derivation.

#### `plan-author` — P0
- **Role:** Author `test-plan` specs with coverage matrix,
  strategy section, scope/out-of-scope.
- **When invoked:** `/plan` on a story or release; "draft a
  test plan for X" natural language; `Draft test plan` ambient
  card on a release-gate without an active plan.
- **Produces:** `test-plan` specs at
  `.hero/planning/test-plans/{slug}/spec.md`; inline-proposed
  plan refinements.
- **Skills:** `plan-structure`, `coverage-matrix-shaping`,
  `risk-based-testing`, `spec-format`, `kickoff-prompt`.
- **Delegates to:** `coverage-strategist` (for budget),
  `test-author` (for case stubs).
- **Prompt sketch:** "You are a senior test plan author. You
  structure plans with strategy + coverage matrix + scope +
  out-of-scope + risks. You always tie a plan to a release,
  sprint, or feature. You produce coverage matrices as printable
  artifacts (qTest-shaped) when the team needs them for
  compliance."
- **Prior art:** TestRail plan/suite shape; qTest traceability
  matrix; Hero's `prd-author` shape.

#### `gherkin-author` — P1
- **Role:** Specialized authoring for BDD shops. Generates cases
  in Gherkin (Feature / Scenario / Given-When-Then) when BDD
  preset or `qa.case_format_default: "gherkin"` is active.
- **When invoked:** Same triggers as `test-author` when format
  resolves to Gherkin; explicit `/author-cases --format gherkin`.
- **Produces:** Gherkin-format `test-case` specs; feature files
  on disk when integration with Cucumber/SpecFlow is configured.
- **Skills:** `gherkin-authoring`, `ears-test-derivation`,
  `scenario-outlining`, `step-definition-shaping`.
- **Delegates to:** `test-author` for non-Gherkin cases the
  same story needs; `seam-requester` for missing step definitions.
- **Prompt sketch:** "You are a senior BDD test author. You
  write Gherkin scenarios that are *executable when bound to
  step definitions*. You use Scenario Outline for parameterized
  cases. You avoid imperative steps (clicking, typing) in favor
  of declarative business-language steps."
- **Prior art:** Cucumber / SpecFlow / Behave; Dan North's
  original BDD writings.

#### `decision-table-author` — P1
- **Role:** Authoring for cases with multi-conjunctive-condition
  logic (pricing rules, eligibility, permission matrices).
  Output is a decision table.
- **When invoked:** When AC contains multiple AND/OR conditions
  with discrete states; `qa.case_format_default: "decision-table"`;
  explicit format selection.
- **Produces:** decision-table format `test-case` specs (condition
  rows × rule columns × action cells).
- **Skills:** `decision-table-authoring`, `equivalence-partitioning`,
  `boundary-value-analysis`.
- **Delegates to:** `test-author` for the non-conditional cases
  same story needs.

#### `exploratory-charter-author` — P2
- **Role:** Authoring for context-driven testing shops. Produces
  charters (mission + heuristics + note template) instead of
  scripted cases.
- **When invoked:** `exploratory-charter` preset; explicit
  `/author-charter`; risk-areas with high uncertainty.
- **Produces:** charter-format `test-case` specs with mission,
  heuristics, time-box, notes scaffold.
- **Skills:** `exploratory-charter`, `session-based-test-management`,
  `heuristic-mnemonics`.
- **Prior art:** Bach / Bolton on Session-Based Test Management;
  RST methodology.

### C.4 Triage tier (4)

#### `qa-investigator` — P0
- **Role:** The pack's ambient surfacer. Detects coverage gaps,
  AC-quality issues, similar cases, suspicious test-issue
  patterns. Drives the right rail's ambient cards on QA artifacts
  and on PM stories in `qa-ready`.
- **When invoked:** Artifact open events; `qa-ready` state
  transition; ambient refresh polls; explicit "what should I
  look at" prompts.
- **Produces:** ambient cards (Investigate / Author / Promote /
  Triage suggestions); no spec writes — proposals only.
- **Skills:** `coverage-gap-detection`, `ac-quality-scoring`,
  `similar-case-detection`, `verdict-output`.
- **Delegates to:** routes user accept to `test-author`,
  `coverage-strategist`, etc.
- **Prompt sketch:** "You are a senior QA investigator. You
  surface what the QA lead would notice if they had three more
  hours per sprint. You propose, you do not decide. Your output
  is ambient cards with verdict-shaped reasons. You write three
  to five cards max per artifact open."
- **Prior art:** Height's ambient AI panel (from PM brief);
  mabl's coverage-gap detection; engineering's
  `nudge-awareness` skill.

#### `test-issue-triager` — P0
- **Role:** Triage incoming test issues (test failures, defects)
  through the four-action workflow (bad-test / reject-linked-story /
  raise-as-new-bug / flag-as-regression).
- **When invoked:** test-issue editor open; bulk triage in the
  unified inbox; `/triage-test-issue` command; ambient
  "untriaged test issues in your inbox" card.
- **Produces:** triage verdicts with downstream artifact creation
  (bug spec, story rejection, regression flag) — actually creates
  the downstream artifact via delegation.
- **Skills:** `test-issue-triage`, `verdict-output`,
  `lifecycle-overlay-awareness`, `cross-pack-routing`.
- **Delegates to:** `qa-reject-story` (for story rejection
  outcome), engineering's `/diagnose` chain (for new-bug or
  regression outcome), `qa-flake-curator` (for bad-test
  outcome → flake backlog).
- **Prompt sketch:** "You are a senior test triage analyst. You
  evaluate a test issue against four outcomes — bad test, story
  rejection, new bug, regression. You propose one primary
  outcome based on the issue's shape and the linked story's
  history. You name your reasoning. The human confirms."
- **Prior art:** mabl's flaky-run classification; design-original
  on the four-outcome routing.

#### `qa-flake-curator` — P0
- **Role:** Active flake-backlog curator. Classifies flaky tests
  (test-issue / environment / true-bug), proposes verdicts with
  deadlines, drives flake count toward zero.
- **When invoked:** Flaky tests backlog view open; test marked
  flaky (manual or auto-detected); ambient "flake count rising"
  card; `/triage-flaky` command.
- **Produces:** verdict per flaky case (with `fix-by` deadline);
  flake metric updates; quarantine recommendations.
- **Skills:** `flake-triage`, `flake-verdict-classification`,
  `stability-scoring`, `verdict-output`.
- **Delegates to:** `regression-curator` (for demotion);
  engineering's `/diagnose` chain (when verdict is true-bug).
- **Prompt sketch:** "You are a senior flake curator. Your stance
  is opinionated: a flaky test is a bug, a flawed test, or an
  environmental issue — never 'just flaky.' You classify each
  failure with a verdict and propose a deadline. You quarantine
  loudly, never silently. Your job is to drive the flake count
  toward zero."
- **Prior art:** mabl's flake verdict; Hero design-original on
  the active-backlog framing.

#### `duplicate-detector` — P1 (overload from PM)
- **Role:** PM's duplicate-detector overloaded for QA. Detects
  duplicate test cases (within and across regression suites),
  duplicate test issues (within a source), duplicate defects.
- **When invoked:** Case authoring (before write); inbox triage
  (ambient "looks like a duplicate"); explicit `Find duplicates`
  button.
- **Produces:** duplicate-candidate suggestions with confidence
  score and similarity diff.
- **Skills:** `similar-case-detection`, `embedding-similarity`,
  `duplicate-thresholds`.
- **Delegates to:** none.
- **Prior art:** PM's duplicate-detector; Productboard's create-
  time duplicate surface.

### C.5 Curation tier (3)

#### `regression-curator` — P0
- **Role:** Owns the regression suite. Promotes stable cases from
  per-story plans; demotes flaky tests; scores cases for
  regression-worthiness; surfaces coverage gaps in the suite
  relative to shipped features.
- **When invoked:** Story → `done` transition (proposes
  promotions); `/promote-to-regression`; regression-suite editor
  open; ambient "this case has been stable for 3 sprints —
  promote?" card.
- **Produces:** promotion proposals (ambient cards); regression-
  suite spec edits; demotion verdicts on flaky cases.
- **Skills:** `regression-scoring`, `stability-scoring`,
  `blast-radius-analysis`, `customer-impact-weighting`.
- **Delegates to:** `qa-flake-curator` for demotion verdicts.
- **Prompt sketch:** "You are a senior regression curator. You
  score cases for regression-worthiness on three axes —
  stability (no flakes in last N runs), blast radius (how many
  customers feel a break here), customer-impact severity (how
  bad is the break). You propose top N promotions per story.
  You demote loudly when flake threshold crossed."
- **Prior art:** Testim's stability scoring; design-original on
  three-axis regression scoring.

#### `coverage-curator` — P1
- **Role:** Owns coverage hygiene across plans and suites.
  Detects orphaned cases (linked story deleted), drift between
  plan and shipped feature, missing regression coverage of
  shipped features.
- **When invoked:** Periodic `qa.check`; coverage view refresh;
  ambient "orphan cases in your suite" card.
- **Produces:** hygiene-issue reports; deprecation proposals
  for orphan cases; coverage-gap reports for shipped features.
- **Skills:** `orphan-detection`, `coverage-drift-analysis`.

#### `seam-requester` — P1
- **Role:** When a case can't be exercised without engineering
  support (missing API, missing test hook, missing instrumentation),
  generates a structured request to engineering.
- **When invoked:** `Request test seam` button on a case;
  `test-author` detects "I need an X to test this" during
  authoring.
- **Produces:** engineering work item (cross-domain `feature`
  spec or `bug` spec with seam-request shape); cross-domain
  graph edge `requires-seam` from case to engineering work item.
- **Skills:** `seam-request-shaping`, `cross-pack-routing`.
- **Delegates to:** engineering's `/design` chain.
- **Prior art:** design-original (no QA tool has this as a
  first-class workflow).

### C.6 Cross-pack tier (3)

#### `qa-reject-story` — P0
- **Role:** Executes the three-action rejection composer on a
  story in `qa-ready` (Add AC / Suggest new story / Reject as
  quality issue). Owns the brand interaction for the pack.
- **When invoked:** `Reject (compose findings)` button on a story;
  `test-issue-triager` routes a finding to "reject linked story";
  natural-language "reject this story for X."
- **Produces:** story state transition (`qa-ready` → `qa-rejected`);
  cross-pack body augmentation (collapsible `## QA Findings`
  block with QA gutter icon); AC additions; intake-item creation
  (for "Suggest new story" path); cross-pack ambient card on
  the linked engineering feature.
- **Skills:** `cross-pack-body-augmentation`,
  `lifecycle-overlay-awareness`, `three-action-rejection`,
  `ac-shaping`, `verdict-output`.
- **Delegates to:** PM's `intake-triager` (for Suggest new story
  → intake-item creation); engineering's notification chain
  (for ambient card on the linked feature).
- **Prompt sketch:** "You are the QA rejection composer. You
  evaluate the finding and propose one of three actions: Add
  AC to this story (in current scope), Suggest new story (out
  of scope — prevents goalpost-moving), or Reject as quality
  issue (implementation doesn't meet existing AC). You write
  the rejection findings as a collapsible block with QA gutter
  attribution. Your default nudge favors `Suggest new story`
  when the finding doesn't contradict existing AC."
- **Prior art:** design-original on three-action composer
  (Z-2 lesson); no QA tool distinguishes these.

#### `handoff-coordinator` — P0 (overload from PM)
- **Role:** PM's handoff-coordinator overloaded with QA-specific
  edge kinds. Records cross-domain handoff edges in the graph
  for: story → test-plan, test-plan → test-case,
  case → bug-raised, case → feature (verifies), defect → bug,
  regression-suite → release-gate.
- **When invoked:** Any cross-domain action that creates an edge.
- **Produces:** graph edges with payload + handoff-stream rows.
- **Skills:** `handoff-protocol`, `cross-pack-routing`,
  `graph-edge-shaping`.
- **Prior art:** PM's handoff-coordinator.

#### `pm-rejection-router` — P1
- **Role:** When `qa-reject-story` selects "Suggest new story,"
  this agent routes the new-story candidate into PM's intake
  funnel with QA-source attribution. Bridges the QA pack to
  PM's intake-triager for clean cross-pack handoff.
- **When invoked:** By `qa-reject-story` only.
- **Produces:** intake-item spec with QA-source attribution.
- **Skills:** `cross-pack-routing`, `intake-shaping`.
- **Prior art:** design-original on the suggest-new-story
  routing.

### C.7 Review tier (2)

#### `qa-reviewer` — P0
- **Role:** Review test plans, cases, regression suites, and
  release-gate verdicts. The QA equivalent of engineering's
  `pr-reviewer` and PM's `pm-reviewer`. Reads for completeness,
  AC-traceability, anti-pattern presence.
- **When invoked:** `/review` on a QA artifact; PR-shaped
  changes to QA specs; pre-release-gate readiness review.
- **Produces:** review verdict (approve / changes-requested / reject)
  with named findings.
- **Skills:** `qa-review-checklist`, `test-anti-patterns`,
  `ac-traceability-verification`, `verdict-output`.
- **Delegates to:** specialized reviewers for plan vs case
  scope.
- **Prompt sketch:** "You are a senior QA reviewer. You evaluate
  test artifacts for AC traceability, anti-pattern presence
  (over-coupled cases, missing negative cases, fragile locators
  in plain-English steps, scope creep). You produce verdicts,
  not feedback essays."
- **Prior art:** Hero's `pr-reviewer`; PM's `pm-reviewer`.

#### `release-gate-reviewer` — P1
- **Role:** Specialized review of release-gate readiness verdicts.
  Reads the blocker list, evaluates against team policy, surfaces
  hidden risk (e.g. low coverage on high-risk areas).
- **When invoked:** Before a release-gate is set to `go`;
  `/review release-gate-X`.
- **Produces:** secondary verdict, named hidden risks.
- **Skills:** `release-readiness-framing`, `risk-based-testing`,
  `blocker-policy-evaluation`.

### C.8 Scrubbers (2)

#### `stale-case-scrubber` — P2
- **Role:** Find cases that haven't been run in N sprints;
  propose retirement.
- **When invoked:** Periodic; `/scrub cases`.
- **Produces:** retirement proposals.

#### `dead-regression-scrubber` — P2
- **Role:** Find regression cases that are no longer relevant
  (linked feature removed, AC changed substantially); propose
  deprecation.
- **When invoked:** Periodic; `/scrub regression`.

### Agent tier summary

| Tier | Count | P0 / P1 / P2 |
|---|---|---|
| Coordination | 1 | 1 / 0 / 0 |
| Strategic | 3 | 2 / 1 / 0 |
| Authoring | 5 | 2 / 2 / 1 |
| Triage | 4 | 3 / 1 / 0 |
| Curation | 3 | 1 / 2 / 0 |
| Cross-pack | 3 | 2 / 1 / 0 |
| Review | 2 | 1 / 1 / 0 |
| Scrubbers | 2 | 0 / 0 / 2 |
| **Total** | **23** | **12 / 8 / 3** |

The **P0 roster** (12 agents — the minimum-viable QA pack v1):
`qa-delivery-lead`, `qa-strategist`, `coverage-strategist`,
`test-author`, `plan-author`, `qa-investigator`,
`test-issue-triager`, `qa-flake-curator`, `regression-curator`,
`qa-reject-story`, `handoff-coordinator`, `qa-reviewer`.

The single most important agent in the pack is `qa-reject-story`
— it executes the three-action composer and is the brand
interaction.

---

## D) Skill library (26 skills)

Skills carry methodology and authoring craft. Loaded by agents on
demand. Each entry: when invoked, core content, prior-art source.

### D.1 Test design techniques — ISTQB-grounded (6)

#### `ears-test-derivation` — P0
- **When invoked:** `test-author` and all format-specific authors
  on case generation.
- **Core:** Mechanical derivation of cases from EARS-shaped AC.
  Each AC implies a happy-path case (trigger met, expected
  response); exhaustive mode adds trigger-not-met, precondition-
  failed, and boundary-value cases per AC. Output traceable to
  source AC.
- **Anti-patterns:** Don't hallucinate cases without AC anchor;
  don't reformat AC silently; don't combine multiple ACs into one
  case unless they're conjunctive.
- **Prior art:** ISTQB foundation + EARS notation;
  design-original on the mechanical derivation.

#### `equivalence-partitioning` — P0
- **When invoked:** `test-author` for inputs with continuous or
  bucketed domains.
- **Core:** Group inputs into equivalence classes; one case per
  class.

#### `boundary-value-analysis` — P0
- **When invoked:** `test-author` for inputs with constraints
  (ranges, lengths, types).
- **Core:** Cases at and around boundaries (min-1, min, min+1,
  max-1, max, max+1, off-type).

#### `decision-table-authoring` — P0
- **When invoked:** `decision-table-author`.
- **Core:** Transform multi-conjunctive AC into a decision table
  (condition rows × rule columns × action cells). Enumerate
  rule combinations; collapse equivalent rules.

#### `state-transition-testing` — P1
- **When invoked:** `test-author` when AC describes a state
  machine.
- **Core:** Derive transition cases (valid transitions, invalid
  transitions, self-loops, terminal states).

#### `use-case-derivation` — P1
- **When invoked:** `plan-author` when working from user-flow
  descriptions rather than AC.
- **Core:** Derive happy path + alternative flow + exception
  flow cases from a use case description.

### D.2 Format-specific authoring (4)

#### `step-by-step-authoring` — P0
- **When invoked:** Default format. `test-author` baseline.
- **Core:** Multi-row step grid (Action, Expected Result, optional
  Notes). One assertion per step row when possible.
- **Anti-patterns:** Vague action verbs ("verify it works"),
  multi-assertion steps, missing expected-result.
- **Prior art:** TestRail step grid.

#### `gherkin-authoring` — P0
- **When invoked:** `gherkin-author`; `qa.case_format_default:
  "gherkin"`; BDD preset.
- **Core:** Feature / Scenario / Given-When-Then structure;
  Scenario Outline with Examples for parameterized cases;
  declarative business-language steps; binding-friendly step
  phrasing.
- **Anti-patterns:** Imperative steps (clicking, typing); fragile
  CSS selectors in step text; multi-When chains; missing
  Background where state is shared.
- **Prior art:** Cucumber / SpecFlow conventions; Dan North BDD.

#### `data-driven-authoring` — P1
- **When invoked:** `test-author` with input parameterization
  requirement.
- **Core:** Parameterized case template + data table (each row =
  one case execution); convention for which fields are
  parameterized vs. fixed.

#### `exploratory-charter` — P1
- **When invoked:** `exploratory-charter-author`.
- **Core:** Charter structure (mission / heuristics / time-box /
  notes scaffold); SBTM session conventions; debriefable output.
- **Prior art:** Bach / Bolton on Session-Based Test Management.

### D.3 Methodology / philosophy (4)

#### `risk-based-testing` — P0
- **When invoked:** `qa-strategist`, `coverage-strategist`,
  `release-gate-reviewer`.
- **Core:** Risk = likelihood × impact per area. Test what would
  hurt most if it broke. Risk matrix shape; risk-aware coverage
  budgets.
- **Prior art:** Hendrickson on risk-based testing.

#### `agile-testing-quadrants` — P1
- **When invoked:** `qa-strategist` for test-type mix decisions.
- **Core:** Four quadrants (Q1 unit, Q2 functional story, Q3
  exploratory, Q4 nonfunctional); use quadrant to balance
  case-type production.
- **Prior art:** Gregory / Crispin More Agile Testing.

#### `context-driven-principles` — P1
- **When invoked:** `qa-strategist` when context-driven preset is
  active.
- **Core:** Seven principles + heuristic-over-process stance.
- **Prior art:** Kaner / Bach / Pettichord on context-driven
  testing.

#### `whole-team-quality` — P1
- **When invoked:** `qa-investigator` when surfacing rejection
  patterns; `qa-reject-story` philosophy framing.
- **Core:** Quality is everyone's responsibility — the Suggest
  new story button, the cross-pack ambient cards, the
  seam-request flow all earn this.
- **Prior art:** Gregory / Crispin.

### D.4 Strategic / coverage (4)

#### `coverage-budgeting` — P0
- **When invoked:** `coverage-strategist`, `qa-strategist`.
- **Core:** Quantify per-story authoring budget by AC count,
  complexity, and risk. Output: "Story A needs 4 cases (~12
  minutes). Story B needs 6 cases (~20 minutes). Total sprint
  budget: ~90 minutes."

#### `coverage-gap-detection` — P0
- **When invoked:** `qa-investigator`, `coverage-strategist`.
- **Core:** Walk a story's AC list, count linked passing cases
  per AC, surface gaps. Reads from graph (no integration
  dependency).

#### `release-readiness-framing` — P0
- **When invoked:** `release-readiness-strategist`,
  `qa-strategist`.
- **Core:** Aggregate sprint coverage + regression status +
  blocker policy into Go/No-Go shape. Configurable per
  `qa.blocker_policy`.

#### `blocker-policy-evaluation` — P1
- **When invoked:** `release-readiness-strategist`,
  `release-gate-reviewer`.
- **Core:** Evaluate open issues against team's configured
  blocker policy (defaults: P0 = hold release; P1 = hold without
  explicit sign-off; P2+ = candidate for next cycle).

### D.5 Curation (4)

#### `regression-scoring` — P0
- **When invoked:** `regression-curator`.
- **Core:** Three-axis score (stability × blast-radius ×
  customer-impact). Threshold for promotion proposal; threshold
  for demotion.

#### `flake-triage` — P0
- **When invoked:** `qa-flake-curator`, `test-issue-triager`.
- **Core:** Verdict workflow (test-issue / environment / true-bug);
  per-verdict deadline conventions.

#### `flake-verdict-classification` — P0
- **When invoked:** `qa-flake-curator`.
- **Core:** Heuristics for classifying a flaky run by its failure
  shape — locator change (test-issue), timing variance (test-
  issue or environment), assertion variance (true-bug),
  environmental dependency (environment).
- **Prior art:** mabl's flake-verdict classification.

#### `stability-scoring` — P0
- **When invoked:** `regression-curator`, `regression-scoring`.
- **Core:** Per-case stability metric over last N runs;
  threshold-driven verdicts (stable / unstable / flaky).
- **Prior art:** Testim's stability scoring.

### D.6 Triage / cross-pack (4)

#### `test-issue-triage` — P0
- **When invoked:** `test-issue-triager`.
- **Core:** Four-outcome routing (bad-test / reject-linked-story /
  raise-as-new-bug / flag-as-regression). Heuristics per outcome.
- **Anti-patterns:** Don't auto-route; always surface verdict
  with reasoning; let the human confirm.

#### `three-action-rejection` — P0
- **When invoked:** `qa-reject-story`.
- **Core:** Three-action composer logic (Add AC / Suggest new
  story / Reject as quality issue). Default nudge based on
  finding shape: contradicts existing AC → Reject as quality
  issue; out-of-scope of AC → Suggest new story; new AC needed
  → Add AC.
- **Prior art:** design-original.

#### `lifecycle-overlay-awareness` — P0
- **When invoked:** Any QA agent reading/writing PM story state.
- **Core:** Recognizes `qa-ready` and `qa-rejected` overlay
  states; valid transitions; what happens when QA pack
  uninstalls (states render as labels, not transitions).

#### `cross-pack-body-augmentation` — P0
- **When invoked:** `qa-reject-story`.
- **Core:** Append collapsible `## QA Findings` block to PM
  story body with QA gutter icon; preserve authorship
  attribution; fold-by-default rendering hint.

### D.7 Integration / operational (4)

#### `normalization-mapping` — P0
- **When invoked:** Unified inbox rendering; any agent reading
  source fields from TestRail / Xray / Jira / GH.
- **Core:** Maps source fields to standard schema (status,
  severity, priority, assignee, age, link). Default mappings
  per source; overridable via `hero.json`.

#### `integration-run-state-reader` — P0
- **When invoked:** Test case editor render; release-gate
  aggregation.
- **Core:** Read run history from TestRail/Xray; cache last-N
  in graph; defer deep history to integration.

#### `qa-preset-detection` — P0
- **When invoked:** Agent boot; preset-aware rendering.
- **Core:** Reads `hero.json` `qa.presets`; emits gate_style,
  test_issue_persistence, rejection_strictness,
  case_format_default, blocker_policy.

#### `seam-request-shaping` — P1
- **When invoked:** `seam-requester`.
- **Core:** Shape an engineering work item from a case + seam
  description. Output is structured `feature` or `bug` spec
  with clear "what the test needs" framing.

### D.8 Shared with PM / engineering (2)

#### `verdict-output` — P0
- Reused from engineering's `pr-reviewer` shape. Standard
  verdict envelope (approve / changes-requested / reject) with
  named findings.

#### `kickoff-prompt`, `spec-format`, `context-injection` — P0 (reused from engineering)

All authoring agents load these. Kickoff discipline applies to
every QA spec.

### Skill tier summary

| Section | Count |
|---|---|
| Test design (ISTQB) | 6 |
| Format authoring | 4 |
| Methodology | 4 |
| Strategic / coverage | 4 |
| Curation | 4 |
| Triage / cross-pack | 4 |
| Integration / operational | 4 |
| Shared with PM/eng | 2 (+ several reused) |
| **Total QA-specific** | **26** |

(Reused engineering/PM skills add ~5 more loaded by QA agents:
`spec-format`, `kickoff-prompt`, `context-injection`,
`testing-and-validation`, `verdict-output`.)

---

## E) Command list (18 commands)

Commands are routers — thin wrappers that pick agents, load skills,
enforce gates, pass arguments. Most QA commands also fire from
contextual buttons with `--inline-propose` mode flag.

### E.1 QA-specific (12)

| Command | Routes to | Trigger contexts |
|---|---|---|
| `/plan-coverage` | `qa-strategist` → `coverage-strategist` → `test-author` | Sprint start; release planning; "what's our coverage plan" |
| `/author-cases` | `test-author` (or format-specific author by preset) | Bottom-strip `Author cases` button on a story or plan; "write tests for X" |
| `/author-plan` | `plan-author` | "draft a test plan for X"; bottom-strip `Draft test plan` |
| `/author-charter` | `exploratory-charter-author` | Exploratory preset; `Draft charter` button |
| `/triage-test-issue` | `test-issue-triager` | Test-issue editor open; inbox bulk triage |
| `/triage-flaky` | `qa-flake-curator` | Flaky backlog; ambient "flake count rising" |
| `/promote-to-regression` | `regression-curator` | Story → done; ambient "promote this case" |
| `/release-gate` | `release-readiness-strategist` → `release-gate-reviewer` | Release planning; "are we ready to ship" |
| `/coverage` | `coverage-strategist` | "show me coverage on X"; story queue refresh |
| `/reject-story` | `qa-reject-story` | Bottom-strip `Reject (compose findings)` button on story in qa-ready |
| `/request-seam` | `seam-requester` | Bottom-strip `Request test seam` button on a case |
| `/scrub-qa` | `stale-case-scrubber`, `dead-regression-scrubber`, `coverage-curator` | Periodic; "tidy our QA workspace" |

### E.2 Shared / cross-domain (6)

| Command | QA-side overload |
|---|---|
| `/design` | On a story: produces a `test-plan` spec via `plan-author` (QA-context routing). On a feature: same. |
| `/deliver` | On a test plan: routes to `test-author` to fill in pending cases. |
| `/diagnose` | Routed to from `test-issue-triager` (raise-as-new-bug / flag-as-regression outcomes). |
| `/review` | Routed to `qa-reviewer` for QA artifacts. |
| `/search` | Cross-pack default; surfaces QA + PM + engineering results. |
| `/why` | Walks `case → story → epic → roadmap-item` chain. |

---

## F) Natural-language routing table

When the user types unstructured intent inside QA context, route
to the listed command. Mirrors PM/engineering pattern.

| User intent | Command |
|---|---|
| "Plan coverage for the sprint", "what should we test" | `/plan-coverage` |
| "Write tests for X", "author cases", "add cases" | `/author-cases` |
| "Draft a test plan for X" | `/author-plan` |
| "Triage this failure", "what is this", "is this a bug" | `/triage-test-issue` |
| "Why does this keep failing", "is this flaky" | `/triage-flaky` |
| "Promote this to regression", "this should be a regression test" | `/promote-to-regression` |
| "Are we ready to ship", "is this release good to go" | `/release-gate` |
| "What's our coverage on X", "how covered is this story" | `/coverage` |
| "Reject this story", "this isn't done", "send back" | `/reject-story` |
| "We need a hook for this", "I can't test this without..." | `/request-seam` |
| "Review this plan", "is this case good" | `/review` |
| "Why does this test exist" | `/why` |
| "Find similar cases" | `duplicate-detector` overload |

Ambiguous intents route to `qa-delivery-lead` which asks one
focused clarifying question.

---

## G) Contextual-button inventory

Per the locked UX, every artifact surfaces 4-6 state-aware
contextual buttons in the bottom strip. Buttons fire commands
(typically with `--inline-propose`) and are the primary way agents
are invoked outside slash commands and direct chat-input prompts.

### Story (Story queue + Story detail screens, QA context)

When QA pack is loaded, story bottom strip extends with QA-specific
buttons depending on lifecycle state:

| Story state | QA-context buttons |
|---|---|
| `ready` | `Plan coverage`, `Author cases` (preview), `Show linked tests` |
| `in-flight` | `Show coverage`, `Author missing cases` |
| `qa-ready` *(brand state)* | **`Reject (compose findings)`** *(brand button)*, **`Approve & mark Complete`**, `Author cases`, `Run cases`, `Show case results` |
| `qa-rejected` | `Author additional cases`, `Re-hand off to engineering` |
| `done` | `Promote to regression`, `Show coverage history` |

The `Reject (compose findings)` button is the single brand
interaction; visual prominence must reflect this.

### Test plan (Test plan editor screen)

| Plan state | Buttons |
|---|---|
| `draft` | **`Author cases`** *(primary)*, `Add strategy`, `Set coverage matrix`, `Find existing coverage` |
| `committed` | `Author missing cases`, `Run cases`, `Update scope` |
| `in-flight` | `Run remaining`, `Show run progress`, `Mark blocked` |
| `completed` | `Promote cases to regression`, `Generate release-gate evidence` |

### Test case (Test case editor screen)

| Case state | Buttons |
|---|---|
| `drafted` | `Refine steps`, `Convert format`, `Find duplicates`, `Promote to ready` |
| `ready` | **`Run`** *(primary)*, `Convert format`, `Tag (quadrant / regression / flaky)`, **`Request test seam`** *(brand button when applicable)* |
| `automated` | `Show run history`, `Stability score`, `Retire (with reason)` |
| `retired` | `Restore`, `Show why retired` |

### Regression suite (Regression suites screen)

| Buttons |
|---|
| `Add cases`, `Demote flaky`, `Score suite`, `Show coverage of shipped features`, `Generate suite-pass report` |

### Release-gate (Release-gate view screen)

| Gate state | Buttons |
|---|---|
| `open` | `Aggregate now`, `Show blockers`, `Add blocker`, `Show coverage` |
| `reviewing` | `Approve (go)`, `Reject (no-go with reasons)`, `Defer to next cycle (per blocker)` |
| `go` | `Show readiness summary`, `Hand off to /release` |
| `no-go` | `Show blockers`, `Re-evaluate` |

### Test issue (Test issue editor, unified inbox row)

The four-action triage is the bottom strip:

| Buttons (always present) |
|---|
| **`Mark as bad test`**, **`Reject linked story`** *(brand)*, **`Raise as new bug`**, **`Flag as regression`** |

Plus context: `Mark as duplicate of…`, `Reassign`, `Snooze`, `Open source`.

### Flaky test (Flaky backlog row)

| Verdict pending | Buttons |
|---|---|
| no verdict | **`Classify (test-issue / environment / true-bug)`** *(primary)*, `Show run history`, `Quarantine` |
| classified | `Set fix-by deadline`, `Hand off to /diagnose`, `Demote from regression`, `Resolve` |

### Cross-domain handoff stream row

| Buttons |
|---|
| `Open story`, `Open case`, `Open bug`, `Re-handoff`, `Show edge timeline` |

---

## H) Sequencing recommendation

### Minimum viable QA pack (v1) — earns the platform's place

Ship the **12 P0 agents**, **all 18 P0 skills**, and **all 18
commands** as the v1 cut. The P0 set is the smallest pack that:

- Can plan coverage for a sprint and author cases for every story.
- Can execute the three-action rejection composer (the brand
  interaction).
- Can triage test issues through the four-outcome workflow (the
  unification thesis).
- Can produce a release-gate verdict.
- Can promote cases to the regression suite and curate flakes.
- Reuses Xray + TestRail integrations via normalization.

Without any one of these, the pack fails its "QA is the quality
spine of the sprint and release" thesis.

### v1.5 additions (within first 3 months post-launch)

- 8 P1 agents (gherkin-author, decision-table-author,
  coverage-curator, seam-requester, duplicate-detector overload,
  release-readiness-strategist, pm-rejection-router,
  release-gate-reviewer).
- 8 P1 skills.
- TestRail bidirectional sync at parity with Xray.

### v2 / later

- 3 P2 agents (exploratory-charter-author, stale-case-scrubber,
  dead-regression-scrubber).
- CI integration deepening for automated-flake detection.
- Visual-baseline format authoring (Applitools-style).
- Record-mode authoring input (Reflect-style → test-author).
- `hero-data-analytics` pack closes the loop for principle #5
  (learn from what shipped).

---

## End notes

This document is the prescribed *what exists*, not the *how it's
prompted*. Implementation under `domains/qa/agents/`,
`domains/qa/skills/`, `domains/qa/commands/` will draft each file
at the engineering/PM agent quality bar — ~50-200 lines per agent
with frontmatter + persona + numbered process + rules +
default-output, ~80-150 lines per skill with what/when/core/anti-
patterns/templates, ~10-80 lines per command as a router.

The pack ships when the **12 P0 agents** can demonstrate the four
core demos end-to-end:

1. **The brand demo** — QA opens a story in `qa-ready`, clicks
   `Reject (compose findings)`, picks `Add AC`, writes findings,
   submits. Story flips to `qa-rejected` with collapsible QA
   block appended; ambient card appears on the linked engineering
   feature in the engineering session within seconds.
2. **The authoring demo** — QA opens a sprint with 12 uncovered
   stories, clicks `Plan sprint coverage`, reviews the budget,
   clicks `Author cases` on the highest-risk story, accepts six
   cases inline in 90 seconds. Cases write through to TestRail.
3. **The triage demo** — A test fails in TestRail; the failure
   surfaces as a test issue in Hero's unified inbox. QA clicks
   `Flag as regression`. A new engineering `bug` is raised via
   `/diagnose` with the case and regression-suite linkage on
   the bug spec. Cross-domain edges visible in `hero why`.
4. **The release-gate demo** — QA opens release-gate for the
   active release. Aggregated coverage + regression status +
   blocker list renders. Two P1 blockers without sign-off — gate
   shows No-Go with named blockers. QA assigns sign-off, gate
   transitions to Go, `Hand off to /release` enabled.

When those four demos pass, the pack ships.
