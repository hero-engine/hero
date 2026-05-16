# Hero PM — Agent / Skill / Command Pack Design

Audience: the agent that will eventually `/deliver hero-pm` (and the
human reviewing this before delivery). This document specifies *what
exists* in the PM domain pack at the agent / skill / command layer —
not how each agent is prompted (those files get drafted at delivery
time and live under `domains/pm/agents/`, `domains/pm/skills/`,
`domains/pm/commands/`).

Sibling files set the stage and must be read first:
- `spec.md` — strategic frame, principles, artifact types, methodology
  layers, dashboard views.
- `research-brief.md` — competitive UX research (steal/leave matrix).
- `mockup-brief.md` — six killer screens with locked UX grammar.

The locked UX (IDE-style left-nav singletons + center tabbed openables
+ bottom IDE-terminal chat bar + opt-in right rail with inline-proposed
outputs) is taken as given. Every agent / command / button in this
design must fit that UX.

---

## A) Prior-art appendix

### A.1 wshobson/agents (~185 Claude Code subagents)

**Studied:** the public agents.md reference and the marketplace plugin
list. Searched the "business" cluster (4 agents) and the broader
catalog for PM roles.

**Finding (mildly surprising):** despite scale, the catalog has **no
dedicated PM agent**. The business cluster covers `business-analyst`
(metrics/KPI), `quant-analyst`, `risk-manager`, plus marketing/sales
execution roles. PRD authoring, story refinement, roadmap curation,
and intake triage are conspicuously absent. This validates the
opportunity: nobody in the public agent ecosystem has shipped a
serious PM pack yet.

**Patterns to adopt:**
- **Tight role scoping** — wshobson agents are narrow and composable
  (one job, one prompt, ~30–80 lines). Borrow this shape; resist
  building monolithic "PM agent" that does everything.
- **Stack/role-detection skill pattern** — the catalog leans heavily
  on a stack-detection skill that decides which language-specific
  skills to load. We mirror this with a `pm-preset-detection` skill
  that reads `hero.json` and loads the active methodology overlay.

**Patterns to leave:**
- The catalog's surface-area-first philosophy (185 agents, many
  redundant). Hero engineering's ~34-agent density is the target;
  PM should match, not exceed.

### A.2 MetaGPT — ProductManager role ("Alice")

**Studied:** `metagpt/roles/product_manager.py` and the surrounding
roles directory. The role is a clean reference for "single-PM-agent
producing PRD" but is monolithic.

**Structure:**
- `name`: Alice, `profile`: "Product Manager".
- `goal`: "Create a Product Requirement Document or market
  research/competitive product research."
- `constraints`: "utilize the same language as the user requirements
  for seamless communication."
- Two `Action` classes: `PrepareDocuments`, `WritePRD`.
- Watches `UserRequirement` and `PrepareDocuments`; produces PRD as
  the sole artifact; hands off to Architect downstream.

**Patterns to adopt:**
- The **role/profile/goal/constraints triple** as a header pattern
  for our agent files (already partially present in engineering
  agents via the `description` frontmatter — extend for PM).
- The **single-output-artifact discipline** — Alice produces a PRD,
  full stop. Mirror with our `prd-author` producing a `prd` spec,
  `story-writer` producing a `story` spec, `pitch-author` producing
  a pitch-shaped `prd`. No agent owns multiple artifact types.
- The **Action → Watch** pattern translates to Hero's "when invoked"
  criteria — what events trigger this agent.

**Patterns to leave:**
- The monolithic ProductManager handling discovery + PRD + handoff.
  Split across discovery / authoring / handoff tiers as engineering
  splits feature-delivery-lead vs engineer vs reviewer.
- Hardcoded handoff to a fixed "Architect" downstream. Hero's
  cross-domain handoff is explicit and graph-recorded
  (`story → feature` edge), not implicit role passing.

### A.3 BMad-Method (bmad-code-org)

**Studied:** the public docs (docs.bmad-method.org) and the workflow
reference on deepwiki. BMad is the closest published prior art to a
multi-agent PM-through-eng pipeline.

**Structure:**
- Six canonical personas: **Business Analyst → Product Manager →
  Product Owner → Architect → Developer → QA**, plus an
  **Orchestrator** that coordinates.
- Each persona has its own system prompt, distinct responsibilities,
  and **explicit handoff prompts** that travel from one persona to
  the next (e.g. "Project brief is complete. Save it as
  `docs/project-brief.md`. Then create the PRD.").
- Artifacts are file-shaped (briefs, PRDs, architecture docs, test
  plans) — same idea as Hero specs.
- Workflow files act as project plans, sequencing personas with
  dependencies and handoff points.

**Patterns to adopt:**
- The **Analyst → PM → PO split.** Hero collapses some of this but
  the lesson holds: separate the "what's the problem" discovery
  agent from the "what's the spec" authoring agent from the
  "shape it for delivery" refinement agent. Maps to our
  `discovery-researcher` → `prd-author` → `story-writer` →
  `handoff-coordinator` chain.
- **Explicit handoff prompts as first-class artifacts.** A handoff
  isn't a vibes pass — it's a structured message from agent N to
  agent N+1. Hero PM's `handoff-protocol` skill should formalize this.
- **Orchestrator as a coordinating role.** Maps to our
  `pm-delivery-lead` (analog of `feature-delivery-lead`).

**Patterns to leave:**
- Six-role mandate. Hero's domain boundary (PM vs eng) is the bigger
  split; sub-splits inside PM should be lighter than BMad's.
- Sequential-only flow. Hero's graph model lets a `roadmap-item`
  spawn multiple stories that move in parallel; BMad's linear
  pipeline doesn't capture that.

### A.4 ChatPRD (Steve Daniels' productized PRD agent)

**Studied:** chatprd.ai's published PRD template and "Using AI to
write a PRD" guide.

**Canonical 10-section PRD template:**
1. Overview / Problem Statement
2. Goals & Success Metrics
3. User Stories & Use Cases
4. Functional Requirements
5. Non-Functional Requirements
6. Design Considerations
7. Technical Considerations
8. Timeline & Milestones
9. Open Questions & Risks
10. Appendix

Their "complete" PRD criterion: **clarity, structure, flexibility,
actionability, stakeholder focus** — balancing technical specs with
business objectives.

**Patterns to adopt:**
- A **canonical PRD template** is non-negotiable. Hero PM's default
  template is pitch-shaped (Shape Up — see spec.md) but the
  ChatPRD ten-section shape becomes an alternate template for
  teams that want it. The `prd-structure` skill carries both.
- **PRD completeness as a lintable quality bar.** Their five
  adjectives (clarity / structure / flexibility / actionability /
  stakeholder focus) translate to `prd-anti-patterns` skill content
  and a `hero score` check on prd specs.

**Patterns to leave:**
- Single-doc-for-everything. ChatPRD bundles user stories *into* the
  PRD; Hero separates them into linked `story` specs (cleaner
  handoff to engineering). The PRD references stories; it doesn't
  contain them.
- Technical Considerations as a PM-authored section. In Hero, that
  belongs in the engineering `feature` spec produced by `/design`
  on a story — keep PM-side PRD focused on what/why, not how.

### A.5 Methodology grounding — synthesis

**Teresa Torres — Continuous Discovery / Opportunity Solution Tree.**
The OST has four layers: **Outcome** (the business need at the root)
→ **Opportunity space** (customer needs/pains) → **Solutions** (ways
to address opportunities) → **Assumption tests** (hypotheses
validated in days, not weeks). Continuous, not phased. Powers our
`opportunity-solution-trees-torres` skill and `discovery-researcher`
agent.

**Shape Up (Basecamp / Ryan Singer).** Pitch shape: **Problem,
Appetite, Solution, Rabbit Holes, No-Gos**. Betting table is a
fixed-input prioritization meeting every cycle. Hill chart visualizes
*unknowns remaining* (uphill = figuring out, downhill = executing) —
not progress percent. Cooldown is a first-class state. Powers our
`pitch-writing-shape-up`, `hill-chart-reasoning`, and the cycle preset
skills, plus the cycle preset's roadmap variant.

**Marty Cagan / SVPG.** Outcomes-over-outputs. Discovery vs delivery
as parallel ongoing tracks. Product-design-engineering trio as the
"empowered team" archetype. Informs `product-strategist` agent POV
and `outcomes-over-outputs` skill.

**EARS (Easy Approach to Requirements Syntax).** Five clause shapes:
ubiquitous (`THE SYSTEM SHALL`), event-driven (`WHEN…SHALL`),
state-driven (`WHILE…SHALL`), unwanted-behavior (`IF…THEN…SHALL`),
optional-feature (`WHERE…IS ENABLED…SHALL`). Already used in
engineering specs and mockup brief. Powers `acceptance-criteria-ears`.

---

## B) Engineering reference summary

The engineering pack ships **34 agents, 45 skills, 27 commands**. The
shape and depth that make it work:

### Agent shape

Engineering agents are YAML-frontmattered markdown files, ~50–200
lines. The strongest examples (`feature-delivery-lead.md`,
`debug-investigator.md`, `engineer.md`) share:

1. **Frontmatter declares operational reality.** `name`, `description`,
   `mode: subagent`, `temperature`, `color`, and crucially
   **`permission`** — what the agent can edit, which other agents it
   can call (`task:`), which skills it can load (`skill:`), webfetch
   access. The delivery-lead's permission map (calls 21 named
   specialists) is the orchestration shape.
2. **"You are X" persona opener.** Tight role definition (e.g. "You
   are a senior detective software engineer specializing in…").
3. **Explicit load list** of skills before substantial work.
4. **Numbered process sections.** Pre-flight, investigation, output,
   verification. Each step is a concrete imperative.
5. **Rules section** with hard constraints (e.g. "Never edit source
   code — only spec files").
6. **Default-output section** describing the artifact shape the agent
   produces.
7. **Anti-loop awareness** (debug-investigator's "you won't always
   find the answer" — partial findings are valuable).

### Skill shape

Engineering skills live under `skills/<name>/SKILL.md` (sometimes
with supplementary files). Frontmatter has `name`, `description`,
`compatibility`, `metadata.audience`, `metadata.purpose`. Body is:

1. **What I do** — one-paragraph statement of purpose.
2. **When to use me** — invocation triggers (which agents, which
   stages).
3. **Core approach** — 3–6 bullets of stance.
4. **Practical guidance** — operational rules.
5. **Anti-patterns** or **Guardrails** — specific behaviors to
   avoid (the strongest skills make this concrete, e.g.
   `architecture-principles` lists "Do not recommend CQRS, event
   sourcing, plugin systems, or heavy orchestration unless the
   problem clearly demands them").
6. **Templates / report shapes** when applicable (e.g.
   `debugging-investigation` carries the full investigation-report
   markdown template).

### Command shape

Engineering commands are thin: ~10–80 lines of YAML+markdown that
**route** to one or more agents. Their job is to:

1. Detect the workspace / sub-folder context.
2. Choose the right agent (or ask if ambiguous).
3. Load required skills, call required platform tools
   (`hero_anchor`, `hero next ask`, `hero index --if-stale`).
4. Pass the user's arguments through.
5. Enforce gates (status checks, dependency checks, kickoff
   inclusion).

Commands rarely contain logic; they orchestrate agents.

### What transfers directly to PM, what needs adaptation

**Transfers directly:**
- Frontmatter shape + permission maps.
- "You are X" + numbered process + rules + default-output.
- Skill template (what / when / core approach / anti-patterns).
- Command-as-router pattern.
- Spec-as-deliverable discipline (the spec file on disk is the
  artifact; chat output is the trace).
- Anchor checks (tripwires) before proposing direction.
- Kickoff-section discipline on every spec.

**Needs adaptation:**
- **Inline-proposed outputs.** Engineering agents write specs to
  disk; PM agents must *also* support inline-proposed artifact
  updates (Draft AC, Refine, Prioritize) that appear in the
  artifact pane with accept/edit/reject, per the locked UX. New
  pattern: a `propose-inline` output mode on PM authoring agents.
- **Contextual-button invocations.** Engineering commands run from
  slash input only. PM commands also fire from contextual buttons
  on artifacts, which inject the artifact's content as `$ARGUMENTS`
  plus an "inline-propose" mode flag.
- **Methodology preset awareness.** No engineering equivalent. PM
  agents must read `hero.json`'s `pm.presets` and adjust authoring
  (e.g. `story-writer` prompts for `points` under sprint preset,
  `appetite` under cycle preset).
- **Cross-domain handoff is a real edge, not a vibes pass.** The
  `handoff-coordinator` agent + `handoff-protocol` skill formalize
  what BMad does with handoff prompts — but Hero records the edge
  in the graph (domain-scoped-knowledge-graph primitive).
- **Live engineering state in PM surfaces.** PM agents querying
  "what's the delivery state of this story's linked feature" must
  hit the cross-domain graph, not the tracker. Skill:
  `cross-domain-graph-query`.

---

## C) Agent roster (27 agents)

Tier counts: **Coordination 2, Strategic 5, Authoring 5, Triage 3,
Prioritization 3, Coordination-Delivery 3, Review 3, Scrubbers 3
= 27.** Methodology coaching is delivered as skills (see §C.8),
not standalone agents.

Priority distribution: **13 P0 / 9 P1 / 5 P2.**

Naming follows engineering's kebab-case convention. Where an agent
mirrors an engineering analog, the analog is named explicitly.

### C.1 Coordination tier (2)

#### `pm-delivery-lead` — P0
- **Role:** Orchestrate PM agents to refine, prioritize, hand off,
  and ship PM work. Analog to `feature-delivery-lead`.
- **When invoked:** `/design` (when target is a `story`, `prd`,
  `epic`, or `roadmap-item`); `/refine`; `/handoff`;
  natural-language "shape this for delivery" / "make this ready".
- **Produces:** updated `prd` / `story` / `epic` / `roadmap-item`
  specs on disk; orchestrated multi-agent outputs.
- **Skills:** `spec-format`, `kickoff-prompt`, `context-injection`,
  `pm-preset-detection`, `handoff-protocol`.
- **Delegates to:** all authoring, triage, prioritization, review,
  and methodology-coach agents below. Cannot edit source code —
  edits specs only.
- **Prompt sketch:** "You are a senior PM lead. You orchestrate
  PM specialists to refine artifacts and prepare them for
  engineering handoff. You do not author content directly — you
  delegate to the right authoring agent. You enforce the five PM
  principles and the methodology preset's authoring rules. The
  artifact file on disk is the deliverable; chat is the trace."
- **Prior art:** structurally mirrors `feature-delivery-lead`;
  borrows BMad's Orchestrator coordination role.

#### `pm-investigator` — P0
- **Role:** Investigate ambiguous intake / customer signals /
  vague feature asks to understand what's *really* being requested
  before authoring anything. Analog to `debug-investigator`.
- **When invoked:** `/triage` on an unclear intake-item; `/discover`
  on a fuzzy problem; "what's actually being asked here?" natural
  language.
- **Produces:** an investigation report written into the intake-item
  or roadmap-item spec (problem statement, evidence summary,
  hypothesized root opportunity, recommended next agent).
- **Skills:** `intake-classification`, `discovery-interview-design`,
  `opportunity-solution-trees-torres`, `evidence-synthesis`.
- **Delegates to:** none (it's an investigator). Hands off
  conclusions to `intake-triager`, `discovery-researcher`, or
  `prd-author` via `pm-delivery-lead`.
- **Prompt sketch:** "You are a senior product detective. You
  investigate customer signals, support escalations, vague asks,
  and competitive intel to identify the underlying opportunity —
  not the surface-level request. You separate evidence from
  hypothesis. Partial findings are valuable; saying 'we need to
  interview three users to resolve this' is a complete output."
- **Prior art:** debug-investigator's investigation discipline
  applied to product signals.

### C.2 Strategic tier (5)

#### `product-strategist` — P0
- **Role:** Frame roadmap-level bets in terms of outcomes,
  opportunities, and tradeoffs. Owns "why this and not that".
- **When invoked:** `/discover`; `/roadmap`; "what should we
  bet on next quarter"; roadmap-item creation.
- **Produces:** roadmap-item descriptions, tradeoff narratives,
  rejected-with-reason annotations, strategic-context strips
  on PRDs/stories.
- **Skills:** `outcomes-over-outputs`, `prioritization-frameworks`,
  `risk-surfacing`, `roadmap-framing`, `opportunity-solution-trees-torres`.
- **Delegates to:** `competitive-analyst`, `metrics-analyst`,
  `discovery-researcher` for inputs; hands off shaped
  roadmap-items to `prd-author`.
- **Prompt sketch:** "You are a senior product strategist in the
  Marty Cagan tradition. You think in outcomes, not outputs.
  You make tradeoffs visible — what's being deferred and why
  it costs to change course. You challenge whether a proposed
  bet ladders to a stated outcome."
- **Prior art:** Cagan's SVPG philosophy; closest engineering
  analog is `greenfield-architect` (high-level framer).

#### `discovery-researcher` — P0
- **Role:** Design and synthesize customer/user research to populate
  the opportunity space of the OST. Continuous discovery cadence.
- **When invoked:** `/discover`; `/interview`; "we need to
  understand X before building"; high-uncertainty roadmap-items.
- **Produces:** discovery notes, interview guides, opportunity
  trees (as embedded sections in PRD or roadmap-item), assumption
  tests with pass/fail criteria.
- **Skills:** `opportunity-solution-trees-torres`,
  `discovery-interview-design`, `assumption-testing`,
  `continuous-discovery-cadence`, `evidence-synthesis`.
- **Delegates to:** none (it's a researcher). Outputs feed
  `product-strategist` and `prd-author`.
- **Prompt sketch:** "You are a senior continuous-discovery
  researcher in the Teresa Torres tradition. You map outcomes →
  opportunities → solutions → assumption tests. You design
  assumption tests that resolve in days, not weeks. You
  surface unstated assumptions before they become silent bets."
- **Prior art:** Teresa Torres' OST + continuous discovery habits
  framework.

#### `competitive-analyst` — P1
- **Role:** Track competitive product moves, pull comparable
  feature shapes, surface what competitors do that we don't.
- **When invoked:** `/discover` with a "what are competitors
  doing about X" angle; intake-items from sales/competitive
  scans; "should we match feature X" asks.
- **Produces:** competitive snapshots as notes in `.hero/knowledge/`,
  attached evidence on roadmap-items.
- **Skills:** `competitive-research`, `evidence-synthesis`,
  `feature-comparison-framing`.
- **Delegates to:** none.
- **Prompt sketch:** "You are a senior competitive analyst. You
  describe what competitors actually ship, not what their
  marketing claims. You distinguish must-match parity from
  optional differentiation. You surface gaps and white space."

#### `metrics-analyst` — P1
- **Role:** Define and interpret success metrics; tie roadmap-items
  and stories to measurable outcomes. (Light v1 — full metric
  specs deferred to `hero-data-analytics`.)
- **When invoked:** `/metrics`; PRD's `Goals & Success Metrics`
  section authoring; principle-#5 ("learn from what shipped")
  retrospective hooks.
- **Produces:** metric definitions as frontmatter on PRDs;
  shipped-vs-expected annotations on completed stories.
- **Skills:** `metrics-design`, `outcomes-over-outputs`,
  `evidence-synthesis`.
- **Delegates to:** none.
- **Prompt sketch:** "You are a senior product analyst. You
  define metrics that are leading, observable, and tied to
  outcomes — not vanity counters. You name the baseline before
  you propose the target."

#### `portfolio-curator` — P2
- **Role:** Cross-roadmap-item curation — theme balance,
  capacity-vs-ambition reconciliation, "are we over-investing in
  X area" surfacing.
- **When invoked:** quarterly roadmap reviews; "how is our
  portfolio balanced" asks.
- **Produces:** portfolio summaries (notes), rebalance
  recommendations on roadmap-items.
- **Skills:** `capacity-planning`, `risk-surfacing`,
  `outcomes-over-outputs`.
- **Delegates to:** `metrics-analyst`, `prioritization-strategist`.

### C.3 Authoring tier (5)

#### `prd-author` — P0
- **Role:** Produce and refine `prd` specs in pitch-shaped or
  ChatPRD-shaped templates. Owns the PRD editor's authoring loop.
- **When invoked:** `/prd`; `/pitch`; `/design` on a roadmap-item;
  "draft a PRD for X" natural language; contextual "Draft PRD"
  button on a roadmap-item.
- **Produces:** `prd` spec at `.hero/planning/prds/{slug}/spec.md`;
  inline-proposed section refinements on existing PRDs.
- **Skills:** `prd-structure`, `prd-anti-patterns`,
  `pitch-writing-shape-up`, `acceptance-criteria-ears`,
  `pm-preset-detection`, `spec-format`, `kickoff-prompt`.
- **Delegates to:** none directly; receives input from
  `product-strategist` / `discovery-researcher` upstream.
- **Prompt sketch:** "You are a senior PRD author. You write
  PRDs that pass the five-adjective test: clarity, structure,
  flexibility, actionability, stakeholder focus. Default template
  is pitch-shaped (Problem / Appetite / Solution / Rabbit Holes /
  No-Gos / Linked stories / Risks). Under non-cycle presets you
  may fall back to the ten-section template. You never invent
  technical implementation — that belongs to engineering."
- **Prior art:** MetaGPT's Alice + ChatPRD's template +
  Shape Up's pitch shape.

#### `story-writer` — P0
- **Role:** Produce and refine `story` specs to INVEST shape with
  EARS-format acceptance criteria. The single highest-volume
  authoring agent.
- **When invoked:** `/refine` on a story; `/refine` on a PRD
  ("split into stories"); contextual "Draft AC" / "Refine"
  button on a story; "make this story ready" natural language.
- **Produces:** `story` spec; inline-proposed AC bullets;
  story-typing chip changes (feature/bug/chore).
- **Skills:** `story-writing-invest`, `acceptance-criteria-ears`,
  `acceptance-criteria-gherkin` (alt format), `pm-preset-detection`
  (points vs appetite vs no-estimate), `spec-format`.
- **Delegates to:** none directly.
- **Prompt sketch:** "You are a senior story writer. INVEST
  (Independent / Negotiable / Valuable / Estimable / Small /
  Testable) is the bar. Acceptance criteria default to EARS
  (WHEN/WHILE/IF/WHERE/THE SYSTEM SHALL); Gherkin available on
  request. You leave the *how* to engineering — your job is the
  *what* and the *done* line. You read the active preset and
  populate the right delivery fields (points / appetite / phase /
  cycle)."
- **Prior art:** INVEST framework + EARS + Pivotal Tracker
  feature/bug/chore typing.

#### `pitch-author` — P1
- **Role:** Author Shape Up pitches specifically — used when the
  active delivery preset is cycle-based and the team wants a
  proper pitch (not a generic PRD). Specialized variant of
  `prd-author`.
- **When invoked:** `/pitch` slash; "shape this as a pitch"
  natural language; cycle preset's contextual "Write Pitch" button
  on a roadmap-item.
- **Produces:** `prd` spec with pitch template applied; appetite
  + rabbit-holes + no-gos enforced as required sections.
- **Skills:** `pitch-writing-shape-up`, `shape-up-cadence`,
  `prd-structure`.
- **Delegates to:** none.
- **Prompt sketch:** "You are a senior Shape Up pitch author.
  Appetite is the budget, not the estimate. Rabbit Holes name
  specific traps to avoid. No-Gos exclude work that won't fit
  the appetite. You refuse to ship a pitch with empty
  Appetite or No-Gos."

#### `epic-framer` — P1
- **Role:** Author and curate `epic` specs — frame the container
  story for a related cluster of stories; reconcile child story
  states.
- **When invoked:** `/refine` on an epic; "group these stories
  into an epic" natural language; intake promotion that exceeds
  one-story scope.
- **Produces:** `epic` spec; epic-to-story rollup state on
  dashboard.
- **Skills:** `epic-framing`, `story-writing-invest`,
  `dependency-mapping`, `spec-format`.
- **Delegates to:** `story-writer` for child stories.
- **Prompt sketch:** "You are a senior epic author. An epic is a
  coherent bet, not a bag of unrelated stories. You write the
  Why and the rollup acceptance criteria; you sequence child
  stories and surface dependencies."

#### `roadmap-curator` — P0
- **Role:** Maintain the roadmap board view — horizon assignments,
  delivery-state reconciliation, stale-item surfacing, lane
  configuration suggestions.
- **When invoked:** `/roadmap`; default-landing-page interactions;
  "reconcile the roadmap with what shipped"; cron-shaped weekly
  sweeps.
- **Produces:** roadmap-item state changes (now/next/later
  reassignments, `shipped` transitions, `dropped` with reason);
  rollup pills on roadmap cards.
- **Skills:** `roadmap-framing`, `horizon-assignment`,
  `cross-domain-graph-query`, `dependency-mapping`,
  `risk-surfacing`.
- **Delegates to:** none.
- **Prompt sketch:** "You are a senior roadmap curator. You keep
  the roadmap honest — what's actually now vs aspirationally
  now, what shipped vs what we still claim, what's deferred
  with a reason. You read live engineering delivery state from
  the cross-domain graph; you don't trust the tracker alone."

### C.4 Triage / curation tier (3)

#### `intake-triager` — P0
- **Role:** Process inbound (customer feedback, sales notes,
  support escalations, competitive signals) into triaged
  intake-items linked to roadmap-items or rejected with reason.
- **When invoked:** `/triage`; new intake-item creation;
  contextual "Triage" button on an Intake row; high-frequency
  natural language ("triage my inbox", "what's new in intake").
- **Produces:** intake-item status changes, link edges to
  roadmap-items, rejection annotations.
- **Skills:** `intake-classification`, `duplicate-detection`,
  `evidence-synthesis`, `customer-segment-weighting`.
- **Delegates to:** `duplicate-detector` (for borderline
  duplicates); `pm-investigator` (for ambiguous signals).
- **Prompt sketch:** "You are a senior intake triager. Every
  inbound item gets a status within 24 hours: linked, merged,
  promoted, or rejected with reason. You preserve source
  attribution — the customer quote is the trust signal. You
  cluster duplicates aggressively."

#### `duplicate-detector` — P0
- **Role:** Detect near-duplicate intake-items, roadmap-items, and
  stories at write-time (Height pattern). Surface candidates with
  confidence + one-click merge.
- **When invoked:** automatically during intake-item creation;
  story creation; roadmap-item promotion; contextual "Find
  duplicates" button.
- **Produces:** duplicate candidate list with confidence scores;
  no autonomous merges.
- **Skills:** `duplicate-detection`, `cross-domain-graph-query`
  (so stories surface duplicate engineering features too),
  `evidence-synthesis`.
- **Delegates to:** none.
- **Prompt sketch:** "You are a duplicate detector. Speed and
  recall matter; precision is the human's job (they confirm
  before merging). Return ranked candidates with the specific
  field overlap that triggered the match — never a black-box
  similarity score alone."

#### `dependency-mapper` — P1
- **Role:** Surface dependencies between roadmap-items, epics,
  and stories — including cross-domain dependencies on engineering
  features.
- **When invoked:** `/prioritize`; `/handoff` (so the handoff
  carries upstream dependency context); contextual "Show
  dependencies" button.
- **Produces:** dependency edges in the graph; "blocked by"
  annotations on cards.
- **Skills:** `dependency-mapping`, `cross-domain-graph-query`,
  `risk-surfacing`.
- **Delegates to:** none.
- **Prompt sketch:** "You are a dependency mapper. You walk the
  graph forward and backward from a given node. You distinguish
  hard blockers from soft sequencing. You surface chains that
  cross the PM/engineering boundary."

### C.5 Prioritization tier (3)

#### `prioritization-strategist` — P0
- **Role:** Apply prioritization frameworks (RICE / ICE /
  value-vs-effort / WSJF) to roadmap-items and stories. Power
  the Roadmap board's RICE / Value-vs-Effort view toggles.
- **When invoked:** `/prioritize`; framework view toggle on
  Roadmap board; "what's first" / "rank these" natural language.
- **Produces:** ranked lists with framework scores as frontmatter;
  inline-proposed ordering changes on the roadmap board.
- **Skills:** `prioritization-frameworks`, `customer-segment-weighting`,
  `evidence-synthesis`, `capacity-planning`.
- **Delegates to:** `metrics-analyst` for impact inputs;
  `capacity-planner` for effort inputs.
- **Prompt sketch:** "You are a senior prioritization strategist.
  Frameworks are tools, not truth — you call out when a score
  rests on a soft input. You show the math so the team can
  challenge it."

#### `capacity-planner` — P1
- **Role:** Reconcile committed work against team capacity —
  velocity (sprint), appetite (cycle), WIP limits (kanban),
  release scope (phased).
- **When invoked:** `/capacity`; `/plan-sprint`; `/plan-cycle`;
  "can we fit X this cycle".
- **Produces:** capacity rollups on cycle/sprint pickers;
  cut-line annotations on Story queue; warning surfaces when
  commitments exceed capacity.
- **Skills:** `capacity-planning`, `sprint-planning`,
  `cycle-planning`, `iteration-planning`, `pm-preset-detection`.
- **Delegates to:** none.
- **Prompt sketch:** "You are a capacity planner. Velocity is
  noisy and not always honest — you show the recent distribution,
  not just the mean. Under cycle preset, appetite is the
  constraint; you flag scope creep against appetite, not
  estimates."

#### `risk-curator` — P1
- **Role:** Surface and shape risks on PRDs, roadmap-items, and
  stories — what could go wrong, what we'd discover too late,
  what assumptions are still untested.
- **When invoked:** PRD risk-section authoring; pre-handoff
  review; "what could go wrong" natural language.
- **Produces:** risk bullets on PRDs and stories; assumption-test
  recommendations.
- **Skills:** `risk-surfacing`, `assumption-testing`,
  `evidence-synthesis`.
- **Delegates to:** `discovery-researcher` (for assumption tests).
- **Prompt sketch:** "You are a risk curator. You name risks
  with the specific scenario that would trigger them, not
  generic 'might not scale' wording. You distinguish risks
  worth testing now from risks worth deferring until we know
  more."

### C.6 Coordination-delivery tier (3)

#### `handoff-coordinator` — P0
- **Role:** Execute the PM → engineering handoff. **The single
  most important agent** because it powers the platform thesis.
- **When invoked:** `/handoff`; "Hand off to /design" contextual
  button on a story; "send to engineering" natural language.
- **Produces:**
  1. Calls `/design` (engineering domain) with the story content
     + handoff context.
  2. Writes a cross-domain `story → feature` edge in the graph
     (`kind: handoff`).
  3. Surfaces the new feature on the Story detail's "Linked
     engineering feature" rail.
  4. Adds a row to the Cross-domain handoff stream.
- **Skills:** `handoff-protocol`, `cross-domain-graph-query`,
  `acceptance-criteria-ears`, `context-injection`.
- **Delegates to:** the engineering `feature-delivery-lead` via
  `/design` invocation. Does **not** generate the engineering
  spec itself — that's `feature-delivery-lead`'s job. Handoff
  is the boundary.
- **Prompt sketch:** "You are the handoff coordinator. You shape
  the story's content into a handoff packet that
  `feature-delivery-lead` can act on: tightened EARS criteria,
  any explicit no-gos, linked PRD context, the originating
  roadmap-item, customer evidence. You then call `/design` on
  the engineering side and verify the cross-domain edge landed.
  This is the brand interaction of the platform — make it work."
- **Prior art:** BMad's explicit handoff prompts, formalized
  into a graph edge.

#### `stakeholder-communicator` — P1
- **Role:** Translate PM artifacts into stakeholder-shaped
  outputs — exec summaries, customer-facing announcements,
  internal updates. Powers Roadmap presentation view.
- **When invoked:** `/release-notes`; presentation-mode toggle on
  Roadmap board; "summarize for the leadership review";
  shipped-story announcements.
- **Produces:** notes (`.hero/knowledge/notes/`) or
  release-note artifacts; presentation-mode roadmap variant.
- **Skills:** `stakeholder-communication`, `release-notes-writing`,
  `outcomes-over-outputs`.
- **Delegates to:** none.
- **Prompt sketch:** "You are a stakeholder communicator. You
  shape the message to the audience without distorting the
  truth. Executives want outcomes and tradeoffs; customers want
  capability and timing; engineers want context and acceptance
  criteria. Same artifact, different cuts."

#### `cycle-planner` — P1 (single agent — see notes)
- **Role:** Plan an upcoming iteration (sprint / cycle / phase
  release). Pulls from the prioritized queue, applies capacity,
  surfaces a recommended commit.
- **When invoked:** `/plan-sprint`; `/plan-cycle`;
  `/plan-iteration`; "plan next cycle" natural language.
- **Produces:** updated `sprint` / `cycle` / `phase` assignment
  fields on stories; recommended commit list; betting-table
  output (cycle preset).
- **Skills:** `sprint-planning`, `cycle-planning`,
  `iteration-planning`, `capacity-planning`, `pm-preset-detection`,
  `shape-up-cadence`.
- **Delegates to:** `capacity-planner`, `prioritization-strategist`.
- **Prompt sketch:** "You are a cycle / sprint / phase planner.
  You read the active preset and apply the right model. You
  don't autonomously commit work — you recommend, and the team
  decides. You surface what gets cut and why."
- **Note:** intentionally one agent across the three presets.
  Preset detection switches its behavior; splitting into
  three agents would create three near-identical files.

### C.7 Review tier (3)

#### `pm-reviewer` — P0
- **Role:** Review PM artifacts (PRDs, stories, epics,
  roadmap-items, intake-items) for quality before they advance.
  Analog to `design-reviewer` / `pr-reviewer`.
- **When invoked:** `/review` on a PM spec; pre-handoff gate
  (before `handoff-coordinator` fires); contextual "Review"
  button.
- **Produces:** review findings written to the spec
  (`## Review` section) or as inline-proposed annotations.
- **Skills:** `prd-anti-patterns`, `story-writing-invest`,
  `acceptance-criteria-ears`, `outcomes-over-outputs`.
- **Delegates to:** none.
- **Prompt sketch:** "You are a senior PM reviewer. Findings
  first, summary later. Severity-ordered. You distinguish
  blocking from advisory. You cite the section of the artifact
  for each finding."

#### `roadmap-reviewer` — P1
- **Role:** Review the roadmap as a whole — balance, alignment
  with stated outcomes, dependency health, stale-item burden.
- **When invoked:** `/review` on the roadmap; quarterly review
  prep.
- **Produces:** roadmap review note in `.hero/knowledge/`;
  flagged roadmap-items with `needs-attention` annotation.
- **Skills:** `roadmap-framing`, `outcomes-over-outputs`,
  `risk-surfacing`.

#### `discovery-reviewer` — P2
- **Role:** Review discovery artifacts — opportunity trees,
  interview synthesis, assumption tests — for rigor.
- **When invoked:** `/review` on discovery output.
- **Produces:** review findings on discovery artifacts.
- **Skills:** `opportunity-solution-trees-torres`,
  `discovery-interview-design`, `assumption-testing`,
  `evidence-synthesis`.

### C.8 Methodology coaching — skills, not agents

Methodology guidance (Continuous Discovery, Shape Up, etc.) is
delivered as **skills**, not as standalone coaching agents. They
package the methodology's knowledge and cadence — `pm-delivery-lead`
and authoring agents (`prd-author`, `story-writer`, `discovery-researcher`)
load these skills on demand when a user asks for methodology
guidance, when a relevant preset is enabled, or when an authoring
agent needs framework grounding.

Skills covering this responsibility:
- `opportunity-solution-trees-torres` (D.2) — OST construction, assumption tests, discovery cadence
- `continuous-discovery-cadence` (D.3) — interview rhythm and weekly touchpoints
- `discovery-interview-design` (D.2) — interview question shape and avoid-leading-the-witness patterns
- `pitch-writing-shape-up` (D.1) — pitch structure, appetite, rabbit holes, no-gos
- `shape-up-cadence` (D.3) — 6-week cycle + cooldown norms, betting-table operation
- `hill-chart-reasoning` (D.3) — uphill (unknowns) vs downhill (execution)

When a user types "how do we do continuous discovery" or "we want
to try Shape Up", the bottom-bar router invokes `pm-delivery-lead`
with the relevant skills loaded; `pm-delivery-lead` produces
coaching notes, cadence configs, and methodology-shaped artifacts
(populated OST templates, cycle-cadence configs, betting-table prep)
without needing dedicated coach agents.

Saves two agent slots and keeps the methodology-vs-action boundary
clean: skills teach, agents act.

### C.9 Scrubbers (3)

Engineering scrubbers (`deadcode-scrubber`, `dedup-scrubber`,
`type-scrubber`, …) clean code-quality issues. PM scrubbers clean
*workspace* quality issues — stale artifacts, ambiguous specs,
duplicate intake.

#### `stale-roadmap-scrubber` — P1
- **Role:** Find roadmap-items that haven't moved in N weeks,
  shipped roadmap-items still marked active, "later" items
  older than the planning horizon. Recommends action: archive,
  drop with reason, or refresh.
- **When invoked:** `/scrub roadmap`; weekly cron.
- **Produces:** scrub report; recommended state changes
  (presented to user, not auto-applied).

#### `duplicate-intake-scrubber` — P1
- **Role:** Cluster recent intake-items, surface near-duplicates
  the live `duplicate-detector` missed at write time, merge
  recommendations.
- **When invoked:** `/scrub intake`; weekly cron.
- **Produces:** cluster report; recommended merges.

#### `ambiguous-story-scrubber` — P2
- **Role:** Find stories that are `ready` but fail INVEST or
  lack EARS acceptance criteria — they'll cause friction at
  handoff. Recommend refinement before they get pulled.
- **When invoked:** `/scrub stories`; pre-cycle planning.
- **Produces:** scrub report; flagged stories with specific
  failure (missing AC, too large, untestable).

---

## D) Skill library (32 skills)

Skill counts by domain: **Writing/authoring 9, Frameworks 5,
Process/methodology 6, Curation 6, Cross-domain 3, Operational 3
= 32.**

### D.1 Writing / authoring (9)

#### `story-writing-invest`
- **Description:** INVEST shape for user stories (Independent /
  Negotiable / Valuable / Estimable / Small / Testable).
- **When invoked:** `story-writer`, `pm-reviewer`,
  `ambiguous-story-scrubber`.
- **Core content:** the six INVEST adjectives with concrete
  examples; how to split stories that fail Small; how to write
  valuable stories that survive negotiation.
- **Anti-patterns:** technical-task masquerading as a story;
  stories that span multiple cycles ("epic disguised");
  stories without a user/role; stories whose value statement is
  "because the spec says so."

#### `acceptance-criteria-ears`
- **Description:** EARS clause patterns for acceptance criteria.
- **When invoked:** `story-writer`, `prd-author`, `pm-reviewer`,
  `handoff-coordinator`.
- **Core content:** five EARS clause shapes (ubiquitous, event-
  driven, state-driven, unwanted, optional-feature); when each
  fits; how to mix freeform bullets with EARS without forcing.
- **Anti-patterns:** forcing every criterion into EARS when it
  doesn't fit; vague predicates ("when appropriate"); criteria
  that test implementation rather than behavior.
- **Prior art:** EARS spec (Alistair Mavin); already used
  engineering-side in `/design`.

#### `acceptance-criteria-gherkin`
- **Description:** Given/When/Then format as an alternate AC shape.
- **When invoked:** `story-writer` (when team prefers Gherkin
  over EARS), `pm-reviewer`.
- **Core content:** Gherkin clause shapes; data tables; scenario
  outlines; tag conventions for cross-cutting concerns.
- **Anti-patterns:** Gherkin novels (10+ steps per scenario);
  test-implementation language ("click the button") in criteria.

#### `prd-structure`
- **Description:** Canonical PRD section list and ordering. Two
  templates: pitch-shaped (default) and ChatPRD-shaped (alt).
- **When invoked:** `prd-author`, `pitch-author`, `pm-reviewer`.
- **Core content:**
  - Default pitch template (Problem / Appetite / Solution /
    Rabbit Holes / No-Gos / Linked stories / Risks).
  - Alt ChatPRD-shaped template (10 sections).
  - When to choose each; how to switch between them without
    rewriting from scratch.
- **Anti-patterns:** PRDs that bundle stories inline instead of
  linking; PRDs that author implementation; freeform PRDs with
  no template (the Notion-chaos trap).
- **Prior art:** Shape Up pitch + ChatPRD template.

#### `prd-anti-patterns`
- **Description:** Concrete failure modes PRDs fall into.
- **When invoked:** `prd-author`, `pm-reviewer`.
- **Core content:** the five-adjective bar (clarity, structure,
  flexibility, actionability, stakeholder focus); ten specific
  PRD smells with examples and fixes.
- **Anti-patterns:** the catalog *is* the anti-patterns — vague
  goals, missing No-Gos, success metrics with no baseline,
  Rabbit Holes that read like reassurance, "stakeholder
  references" without actual signal.

#### `pitch-writing-shape-up`
- **Description:** Shape Up pitch shape — appetite, rabbit holes,
  no-gos as required sections; how to actually fill them.
- **When invoked:** `pitch-author`,
  `prd-author` (when cycle preset is active).
- **Core content:** appetite as budget (not estimate); rabbit
  holes as specific traps (not generic risks); no-gos as
  explicit scope exclusions tied to appetite.
- **Anti-patterns:** empty Appetite section; Rabbit Holes that
  read like risks (they're *traps with specific avoidance
  decisions*, not "might be hard"); No-Gos that copy the
  Boundaries section of an engineering spec; six-week
  mandate.
- **Prior art:** Ryan Singer, Shape Up.

#### `roadmap-framing`
- **Description:** How to write a roadmap-item that earns its
  place — outcome ladder, evidence, appetite/horizon, named
  trade-off.
- **When invoked:** `roadmap-curator`, `product-strategist`,
  `pm-reviewer`.
- **Core content:** every roadmap-item should answer "what
  outcome does this serve," "what evidence supports it,"
  "what are we deferring to do this."
- **Anti-patterns:** roadmap-items that are project names with
  no outcome; "Q3 priorities" lists that never explicitly
  defer anything; roadmap-items that are themes (themes are
  lanes, not items).

#### `release-notes-writing`
- **Description:** Customer-facing and internal release-note
  shapes; what to say, what to omit.
- **When invoked:** `stakeholder-communicator`,
  `pm-delivery-lead` on shipped stories.
- **Core content:** lead with the user benefit, not the
  feature name; mention behavior changes that affect existing
  workflows; link to docs.
- **Anti-patterns:** marketing-flavor everything; changelog
  dumps with no narrative; release notes that read like
  internal commit messages.

#### `stakeholder-communication`
- **Description:** Audience-shaped messaging — exec, customer,
  internal eng, sales — without distorting the underlying truth.
- **When invoked:** `stakeholder-communicator`.
- **Core content:** the same artifact has a different cut for
  each audience; outcomes for exec, capability + timing for
  customer, context + AC for engineering, talking points for
  sales.
- **Anti-patterns:** one-cut-for-all messages; sandbagging timing
  for one audience while quoting an earlier date to another.

### D.2 Frameworks (5)

#### `prioritization-frameworks`
- **Description:** RICE / ICE / value-vs-effort / WSJF —
  mechanics, inputs, when each fits, common abuse modes.
- **When invoked:** `prioritization-strategist`,
  `portfolio-curator`.
- **Core content:** input definitions (Reach, Impact,
  Confidence, Effort for RICE; weighted-shortest-job-first
  math); when soft inputs make the score meaningless.
- **Anti-patterns:** treating a single framework's score as
  ground truth; abusing Confidence to make a pet project win;
  RICE-theater where the inputs are guesses but the output is
  quoted as data.

#### `metrics-design`
- **Description:** How to design measurable success metrics for
  a product change — leading vs lagging, baseline before
  target, observability.
- **When invoked:** `metrics-analyst`, `prd-author` (Goals &
  Success Metrics section).
- **Core content:** a good metric is observable, leading,
  outcome-tied, has a baseline, has a target with rationale.
- **Anti-patterns:** vanity counters; "engagement" with no
  definition; targets without baselines; metrics that can only
  be measured retrospectively.

#### `opportunity-solution-trees-torres`
- **Description:** Teresa Torres' OST framework — outcome →
  opportunities → solutions → assumption tests.
- **When invoked:** `discovery-researcher`,
  `product-strategist`,
  `pm-investigator`.
- **Core content:** the four layers and what belongs at each;
  how to break a solution into testable assumptions; assumption
  tests that resolve in days, not weeks; the discipline of
  validating before scaling.
- **Anti-patterns:** trees that jump straight from outcome to
  solution (skipping opportunity space); assumption tests that
  take a quarter to run; OSTs that ossify into roadmaps.

#### `discovery-interview-design`
- **Description:** Designing customer/user interviews that
  generate opportunity-space signal — sample size, question
  framing, synthesis.
- **When invoked:** `discovery-researcher`,
  `pm-delivery-lead`.
- **Core content:** open questions about specific past
  experiences; avoid leading; cadence (5–10 interviews / week
  for a discovery-active team); structured synthesis.
- **Anti-patterns:** "would you use it" hypothetical questions;
  interviewing only happy customers; single-interview-then-build.

#### `okr-design` — P2
- **Description:** Objectives + Key Results — outcome-shaped
  objectives, measurable KRs, cadence.
- **When invoked:** deferred — OKRs are out of v1 scope per
  spec.md. Skill stays in the catalog as P2 so it's ready
  when the `strategy` domain or `hero-pm` v2 brings OKRs in.

### D.3 Process / methodology (6)

#### `sprint-planning`
- **Description:** Scrum/scrumban sprint planning — velocity,
  story-point sizing, sprint goals, cut-line decisions.
- **When invoked:** `capacity-planner`, `cycle-planner`
  (sprint preset).
- **Core content:** velocity as a noisy distribution, not a
  point estimate; sprint-goal-first commitment; carry-over
  rules.
- **Anti-patterns:** rolling-average velocity treated as
  ground truth; sprint goals that are just "finish the
  stories"; planning-poker theater.

#### `cycle-planning`
- **Description:** Shape Up cycle planning — betting table,
  appetite vs estimate, cooldown.
- **When invoked:** `capacity-planner`, `cycle-planner`
  (cycle preset).
- **Core content:** the betting table is a 90-minute decision
  meeting with fixed inputs (the pitches); appetite caps
  scope; cooldown is non-negotiable.
- **Anti-patterns:** estimating inside cycles; skipping
  cooldown to "catch up"; rolling cycles into each other
  without an explicit betting decision.

#### `iteration-planning`
- **Description:** Generic iteration shape — used under kanban
  and phased presets; rolling commitment, WIP awareness.
- **When invoked:** `cycle-planner` (kanban / phased presets).
- **Core content:** WIP limits as a tool, not a wall;
  rolling commitment; phase-gate semantics for phased preset.

#### `continuous-discovery-cadence`
- **Description:** Weekly cadence for continuous discovery —
  interview slots, synthesis sessions, OST refresh.
- **When invoked:** `discovery-researcher`,
  `pm-delivery-lead`.
- **Core content:** the weekly habit (Torres recommends 5–10
  interviews / week per discovery-active team); OST refresh
  at the same cadence; assumption-test scheduling.
- **Anti-patterns:** episodic discovery ("we'll do interviews
  before Q3"); discovery without synthesis; tests that
  block forward motion.

#### `shape-up-cadence`
- **Description:** Six-week cycle + two-week cooldown rhythm;
  betting-table timing; hill-chart updates.
- **When invoked:** `cycle-planner`
  (cycle preset).
- **Core content:** the cycle is the cadence; betting once
  per cycle, not continuously; hill-chart updates at least
  twice / cycle.
- **Anti-patterns:** ad-hoc cycle starts; betting outside the
  betting table; hill charts as progress bars.

#### `hill-chart-reasoning`
- **Description:** Hill chart as an unknowns-remaining
  visualization, not a progress percentage.
- **When invoked:** `pm-delivery-lead`
  (cycle preset).
- **Core content:** uphill = figuring out, top = apex, downhill
  = executing; stuck dots at the apex are the signal that
  matters; movement (or absence of movement) is the data.
- **Anti-patterns:** hill charts converted to percentages;
  static dots that never move; hill charts on bug fixes (use
  status for those).

### D.4 Curation (6)

#### `intake-classification`
- **Description:** Classifying inbound — source, segment,
  signal-strength, opportunity hint.
- **When invoked:** `intake-triager`, `pm-investigator`.
- **Core content:** preserve source attribution as the trust
  signal; classify by signal strength (single-customer ask,
  repeated across segment, blocking deal) and proposed action
  (link / promote / merge / reject).
- **Anti-patterns:** discarding source on classification;
  treating one loud customer as a pattern; rejecting without
  a recorded reason.

#### `duplicate-detection`
- **Description:** Strategies for detecting near-duplicate PM
  artifacts at write time and via background sweeps.
- **When invoked:** `duplicate-detector`,
  `duplicate-intake-scrubber`.
- **Core content:** title overlap, content shingles, topic
  embeddings, linked-roadmap-item overlap; ranked candidates
  with field-specific match reasons.
- **Anti-patterns:** opaque similarity scores; auto-merging;
  duplicate detection that ignores cross-domain (a story
  duplicating an engineering feature is a real case).

#### `dependency-mapping`
- **Description:** Cross-artifact dependency graph walking —
  forward, backward, and across domain boundaries.
- **When invoked:** `dependency-mapper`, `roadmap-curator`,
  `prioritization-strategist`.
- **Core content:** hard blockers vs soft sequencing;
  transitive dependencies; cross-domain edges
  (`story → feature → bug` chains).
- **Anti-patterns:** one-hop-only dependency analysis;
  ignoring soft sequencing that becomes hard at scale.

#### `capacity-planning`
- **Description:** Reading team capacity under each preset —
  velocity distributions (sprint), appetite math (cycle),
  WIP and aging (kanban), release scope (phased).
- **When invoked:** `capacity-planner`, `cycle-planner`.
- **Core content:** preset-specific math; honest velocity
  reporting; the difference between commit and forecast.
- **Anti-patterns:** sandbag-then-overcommit; using last
  sprint's velocity as next sprint's commit without
  uncertainty bands.

#### `risk-surfacing`
- **Description:** Naming risks concretely — specific scenario,
  who/what/when, recommended response.
- **When invoked:** `risk-curator`, `pm-reviewer`,
  `roadmap-reviewer`.
- **Core content:** good risks have a scenario, an indicator,
  and a response; "might not scale" is not a risk, "if usage
  exceeds 10×, the cron-based digest will miss its window" is.
- **Anti-patterns:** generic risk catalogs; risks without
  indicators; risks named after technical-debt-of-the-week.

#### `domain-glossary-maintenance`
- **Description:** Maintain the PM domain's shared vocabulary
  (in `.hero/knowledge/`) so terms stay consistent across
  artifacts.
- **When invoked:** `pm-reviewer` (when new terminology
  appears), `convention-author` (cross-domain).
- **Core content:** glossary entries with definition, scope,
  cross-references; conflict detection across artifacts.
- **Anti-patterns:** glossary that documents every word ever
  used; PM-only glossary that ignores engineering's terms
  (the silo-tear breaks if vocabulary doesn't span domains).

### D.5 Cross-domain (3)

#### `handoff-protocol`
- **Description:** The formal protocol for handing a `story`
  off to engineering as a `feature`. The platform thesis as
  a skill.
- **When invoked:** `handoff-coordinator`,
  `pm-delivery-lead`.
- **Core content:**
  - Handoff packet shape: story content + AC + PRD linkage +
    roadmap-item linkage + customer evidence + No-Gos.
  - The graph edge: `story → feature`, `kind: handoff`,
    `created_by: handoff-coordinator`, `created_at:`,
    `handoff_context:` (the diff between story content and
    what `/design` produced — captured later by
    `feature-delivery-lead`).
  - Verification: confirm the edge landed in the graph;
    confirm the engineering spec exists; confirm the Story
    detail's "Linked engineering feature" rail updated.
  - Re-handoff semantics: when engineering rejects or the
    spec is abandoned, the original handoff edge stays but
    a new edge is created (history preserved).
- **Anti-patterns:** copy-paste handoffs without context;
  handoffs that skip the graph edge (the rail won't update);
  handoffs before the story is `ready` (premature handoff).
- **Prior art:** BMad's handoff prompts, generalized to a
  graph edge.

#### `cross-domain-graph-query`
- **Description:** Querying the cross-domain knowledge graph
  from a PM session — pulling live engineering delivery state,
  walking `story → feature → bug` chains, finding cross-domain
  duplicates.
- **When invoked:** `roadmap-curator`, `dependency-mapper`,
  `duplicate-detector`, `handoff-coordinator`.
- **Core content:** how to express cross-namespace queries
  (depends on primitive #6); ranking active-domain vs
  cross-domain results; respecting domain boundaries in
  output rendering.
- **Anti-patterns:** silently merging cross-domain results
  into active-domain output (lose the boundary signal);
  blindly trusting tracker state over graph state when they
  disagree (graph wins for in-session views).

#### `product-vision-writing`
- **Description:** Crafting a one-page product vision that
  ladders strategy to roadmap. Spans PM and engineering —
  the vision is the root of the OST and the framing for
  engineering's architecture decisions.
- **When invoked:** `product-strategist`,
  `portfolio-curator`.
- **Core content:** target user, problem, why-now,
  outcome ladder, what we're not (the explicit deferrals).
- **Anti-patterns:** vision docs that read like marketing
  taglines; visions without explicit deferrals (everything
  is in scope); visions that ignore engineering reality.

### D.6 Operational (3)

#### `pm-preset-detection`
- **Description:** Read `hero.json`'s `pm.presets` config and
  decide which preset-specific behavior applies. Analog to
  engineering's `stack-detection`.
- **When invoked:** every authoring agent
  (`prd-author`, `story-writer`, `epic-framer`,
  `roadmap-curator`); `capacity-planner`, `cycle-planner`.
- **Core content:** how to read the preset config; what each
  preset implies for fields (`points` vs `appetite` vs none),
  rollups, and dashboard variants; fallback when preset is
  unset.
- **Anti-patterns:** hardcoding sprint assumptions in agents;
  forcing estimation when the active preset doesn't require it.

#### `evidence-synthesis`
- **Description:** Pulling intake-items, support tickets,
  competitive notes, and interview synthesis into a coherent
  evidence trail attached to a roadmap-item or PRD.
- **When invoked:** `discovery-researcher`,
  `prioritization-strategist`, `risk-curator`,
  `metrics-analyst`, `competitive-analyst`, `pm-investigator`.
- **Core content:** group evidence by source kind; weight by
  recency, segment, and customer pain intensity; preserve
  quote attribution.
- **Anti-patterns:** "100 customers asked for this" with no
  underlying data; cherry-picking one customer quote into a
  trend; synthesis that strips source attribution.

#### `assumption-testing`
- **Description:** Designing fast assumption tests (Torres-style)
  — what to test, how to design the test, pass/fail criteria,
  what the result changes.
- **When invoked:** `discovery-researcher`, `risk-curator`,
  `pm-delivery-lead`.
- **Core content:** identify desirability / viability /
  feasibility / usability assumptions; design tests that
  resolve in days; pre-register pass/fail.
- **Anti-patterns:** "let's build an MVP to test it" (an MVP
  is a delivery, not a test); tests with no pre-registered
  pass/fail; tests that take longer than the work they're
  testing.

---

## E) Command list (22 commands)

PM-specific (new): 14. Shared (reused or routed): 8.

### E.1 PM-specific (14)

#### `/refine`
- **Description:** Refine a story or PRD — sharpen AC, fill
  gaps, propose splits, apply INVEST.
- **When invoked:** slash; natural language "make this ready",
  "tighten the AC", "split this story"; contextual "Refine"
  button on Story / PRD.
- **Workflow:** routes to `pm-delivery-lead` → delegates to
  `story-writer` or `prd-author` based on artifact type;
  loads `story-writing-invest`, `acceptance-criteria-ears`,
  `pm-preset-detection`.
- **Output:** inline-proposed AC bullets / section refinements
  on the open artifact; spec on disk is updated when accepted.
- **Gates:** spec must be in editable state (`drafted` /
  `refined`); not on `done` / `shipped` artifacts.

#### `/triage`
- **Description:** Triage one or many intake-items.
- **When invoked:** slash; natural language "triage my inbox",
  "what's new in intake"; contextual "Triage" button on an
  Intake row.
- **Workflow:** routes to `intake-triager` → calls
  `duplicate-detector` on each; `pm-investigator` for
  ambiguous items.
- **Output:** intake-item state changes (linked / merged /
  promoted / rejected with reason); right-detail-pane
  populated with suggested matches.
- **Gates:** none.

#### `/roadmap`
- **Description:** Open / curate / reconcile the roadmap.
- **When invoked:** slash; natural language "show the roadmap",
  "reconcile the roadmap", "what shipped this quarter".
- **Workflow:** routes to `roadmap-curator`; for reconciliation,
  pulls live engineering delivery state via
  `cross-domain-graph-query`.
- **Output:** updated roadmap-items; opens Roadmap board tab if
  not already open.

#### `/prioritize`
- **Description:** Apply a prioritization framework to a set
  of roadmap-items or stories.
- **When invoked:** slash with optional framework
  (`/prioritize --rice`, `/prioritize --wsjf`); natural
  language "rank these", "what's first"; framework view
  toggle on Roadmap board.
- **Workflow:** routes to `prioritization-strategist` →
  delegates effort/impact inputs to `capacity-planner` /
  `metrics-analyst`.
- **Output:** ranked list with framework scores as
  frontmatter on roadmap-items; inline-proposed ordering on
  the Roadmap board.

#### `/pitch`
- **Description:** Author a Shape Up pitch from a
  roadmap-item.
- **When invoked:** slash; contextual "Write Pitch" button on
  a roadmap-item (visible under cycle preset only).
- **Workflow:** routes to `pitch-author`; loads
  `pitch-writing-shape-up`.
- **Output:** `prd` spec with pitch template, written to
  `.hero/planning/prds/{slug}/spec.md`.
- **Gates:** cycle preset active (warn otherwise); roadmap-item
  must exist.

#### `/prd`
- **Description:** Author a PRD from a roadmap-item (default
  template: pitch-shaped, or ChatPRD-shaped if requested).
- **When invoked:** slash; contextual "Draft PRD" button on a
  roadmap-item.
- **Workflow:** routes to `prd-author`; loads `prd-structure`,
  `prd-anti-patterns`, `pm-preset-detection`.
- **Output:** `prd` spec on disk; inline-proposed in the PRD
  editor.

#### `/handoff`
- **Description:** Hand a `story` off to engineering as a
  `feature`. **The platform-thesis command.**
- **When invoked:** slash; "Hand off to /design" contextual
  button on Story detail; natural language "send to
  engineering", "make this an engineering feature".
- **Workflow:** routes to `handoff-coordinator` → calls
  engineering's `/design` with the story content + handoff
  packet → writes the cross-domain edge → verifies the
  Story detail rail.
- **Output:** engineering `feature` spec exists; graph edge
  `story → feature` with `kind: handoff` exists; Story
  detail's "Linked engineering feature" rail populated;
  new row in Cross-domain handoff stream.
- **Gates:** story must be `ready`; PRD link recommended;
  AC must be non-empty.

#### `/discover`
- **Description:** PM-flavored discovery — opportunity
  framing, customer signal exploration, OST construction.
- **When invoked:** slash; natural language "explore X",
  "we don't know enough about Y".
- **Workflow:** routes to `product-strategist` →
  `discovery-researcher` → `competitive-analyst` as needed.
- **Output:** discovery notes; OST embedded in roadmap-item
  or `.hero/knowledge/`; recommended next agent.
- **Note on naming:** engineering already ships `/discover`
  (product-ideator). The domain-routing primitive (#3) makes
  this a non-issue — `/discover` resolves to the active
  domain's binding. PM in PM, engineering in engineering.
  No alias needed, no `/discover-pm` ever ships. Only one
  binding is live at any time because only one domain is
  active. Same pattern applies to any future name collision
  across domain packs.

#### `/metrics`
- **Description:** Define / update / interpret success metrics
  on a PRD or roadmap-item.
- **When invoked:** slash; contextual "Define metrics" on PRD;
  natural language "what's the success metric for X".
- **Workflow:** routes to `metrics-analyst`.
- **Output:** metric definitions as frontmatter; baseline +
  target + observability path.

#### `/interview`
- **Description:** Design a customer/user interview guide.
- **When invoked:** slash; "draft an interview guide".
- **Workflow:** routes to `discovery-researcher`; loads
  `discovery-interview-design`.
- **Output:** interview guide written to
  `.hero/knowledge/interviews/{slug}.md`.

#### `/release-notes`
- **Description:** Produce release notes for shipped
  stories / epics / roadmap-items.
- **When invoked:** slash; natural language "draft release
  notes for X"; contextual "Draft release notes" on a
  shipped artifact.
- **Workflow:** routes to `stakeholder-communicator`; loads
  `release-notes-writing`.
- **Output:** release-note artifact; optionally pushed to
  tracker comment / external channel.

#### `/capacity`
- **Description:** Show / reconcile team capacity for the
  active cadence.
- **When invoked:** slash; cycle/sprint picker on Story queue.
- **Workflow:** routes to `capacity-planner`.
- **Output:** capacity rollup; cut-line on Story queue.

#### `/plan-cycle`, `/plan-sprint`, `/plan-iteration`
- **Description:** Plan the next iteration under the active
  preset (cycle / sprint / phased or kanban respectively).
- **When invoked:** slash; "plan next cycle / sprint";
  pre-cycle planning automation.
- **Workflow:** routes to `cycle-planner` which adapts to
  the active preset.
- **Output:** field assignments on stories; recommended
  commit list; betting-table output (cycle preset).
- **Note:** three aliases backed by one agent — preset
  detection picks the right behavior. Surfacing three
  commands matches engineering's `/sprint` convention and
  matches the user's mental model (you don't ask to "plan
  an iteration" if you run sprints).

#### `/scrub <concern>`
- **Description:** Hygiene sweeps for PM artifacts. Concerns:
  `roadmap` (stale roadmap-items), `intake` (duplicate
  intake), `stories` (ambiguous stories). Matches engineering's
  `/scrub <concern>` pattern.
- **When invoked:** slash with concern argument; weekly cron;
  "clean up the roadmap", "duplicate intake", "ambiguous
  stories".
- **Workflow:** routes to the matching scrubber agent based
  on the concern argument.
- **Output:** scrub report with recommended actions; no
  auto-apply.

#### `/standup`
- **Description:** Generate a stand-up update from
  intra-cycle changes — what moved, what's blocked, what
  shipped.
- **When invoked:** slash; "draft my standup"; daily cron.
- **Workflow:** routes to `stakeholder-communicator` →
  pulls graph state for the active cycle.
- **Output:** standup note for the PM, optionally pushed to
  Slack / posted as a tracker comment.

### E.2 Shared with engineering (8)

These commands already exist in engineering; the PM pack
adapts them to accept PM artifact types.

#### `/design`
- **PM behavior:** accepts `roadmap-item` (produces a PRD via
  `prd-author`) and `story` (produces an engineering `feature`
  via cross-domain routing — same as the killer-demo
  handoff). When invoked on a story, routes through
  `handoff-coordinator` so the cross-domain edge is recorded.
- **Gate:** the cross-story handoff variant requires AC to
  be non-empty.

#### `/deliver`
- **PM behavior:** for PRDs, `/deliver` means "advance the
  PRD through review → approved." For stories,
  `/deliver` typically routes to engineering (via
  `/handoff`). PM-side `/deliver` on a story is rare and
  warns.

#### `/diagnose`
- **PM behavior:** investigates ambiguous customer
  asks / vague intake — analog to engineering bug
  investigation. Routes to `pm-investigator` (not
  `debug-investigator`).

#### `/search`, `/why`, `/blocked`, `/note`, `/decide`
- **PM behavior:** unchanged surface, cross-domain results
  (PM + engineering). `/why` on a roadmap-item walks down
  through epic → story → feature → commit; on a feature
  walks up to story → roadmap-item.

---

## F) Natural-language routing table

Mirrors engineering's CLAUDE.md routing table — for the PM
domain. This table goes into `domains/pm/AGENTS.md` when
the domain-plugin-architecture primitive lands.

| User intent | Command |
|---|---|
| New feedback, customer ask, support escalation, sales note, "this came in" | `/triage` |
| Refine, tighten, "make this ready", "draft AC", INVEST, EARS | `/refine` |
| Prioritize, rank, RICE, ICE, WSJF, value-vs-effort, "what's first" | `/prioritize` |
| Hand off, send to engineering, "ready for dev", "make this an engineering feature" | `/handoff` |
| Draft PRD, write requirements, product doc, "spec this out" | `/prd` |
| Pitch, Shape Up, "shape this", appetite, betting table | `/pitch` |
| Roadmap, "what's coming", reconcile roadmap, "show the roadmap" | `/roadmap` |
| Discover, explore, "we don't know enough about X", customer research | `/discover` (resolves to PM discovery via domain routing) |
| Metric, success measure, KPI, "how do we measure this" | `/metrics` |
| Interview, customer call, user research, "design an interview" | `/interview` |
| Release notes, announce, "what shipped this week" | `/release-notes` |
| Capacity, "can we fit X this cycle", velocity, appetite room | `/capacity` |
| Plan next cycle, plan next sprint, plan iteration, "what should we commit to" | `/plan-cycle` / `/plan-sprint` / `/plan-iteration` |
| Standup, daily update, "what's new this week" | `/standup` |
| Stale roadmap, "clean up the roadmap", "what's been dropped" | `/scrub roadmap` |
| Duplicate intake, cluster feedback, "is this a duplicate" | `/scrub intake` |
| Ambiguous stories, "stories that won't deliver cleanly" | `/scrub stories` |
| Bug in a customer ask, "this signal is confusing", "what's actually being asked" | `/diagnose` (PM-flavored, routes to `pm-investigator`) |
| Design a feature spec from a roadmap-item or story | `/design` |
| Search across PRDs, stories, intake, roadmap | `/search` |
| Why does this exist, "trace this back" | `/why` |
| What's stuck, "blocked items", dependencies | `/blocked` |
| Note, capture, remember this conversation | `/note` |
| Decision, tradeoff, choose between options | `/decide` |
| Review this PRD / story / roadmap | `/review` (routed to `pm-reviewer`) |
| Retro, postmortem, lessons learned on a shipped item | `/retro` |

---

## G) Contextual-button inventory

Each PM artifact surfaces a small set of contextual buttons
inline (per the locked UX). Buttons fire commands (often with
an `--inline-propose` flag so the output appears in the
artifact pane, not a new tab).

### Story (Story detail screen)
- **Hand off to /design** → `/handoff` → `handoff-coordinator`.
  **The brand button.**
- **Refine** → `/refine` → `pm-delivery-lead` → `story-writer`.
- **Draft AC** → `/refine --section ac --inline-propose` →
  `story-writer` with `acceptance-criteria-ears`.
- **Find duplicates** → `duplicate-detector` (inline panel).
- **Find similar stories** → contextual
  `cross-domain-graph-query` invocation.
- **Show dependencies** → `dependency-mapper` (inline panel).
- **Review** → `/review` → `pm-reviewer`.

### PRD (PRD editor screen)
- **Approve** → state-flip action (requires reviewer checks).
- **Suggest AC** → contextual `story-writer` invocation that
  drafts AC into a child story.
- **Find related decisions** → cross-domain `/why` walk.
- **Summarize for standup** → `stakeholder-communicator`.
- **Refine section** (per-section) → `prd-author`
  `--section` flag.
- **Convert to pitch** (Shape Up template) → `pitch-author`.

### Epic (Epic detail)
- **Split into stories** → `epic-framer` → `story-writer`.
- **Show rollup** → `epic-framer` rollup query.
- **Refine** → `/refine` → `epic-framer`.

### Roadmap-item (Roadmap board card right-rail / detail)
- **Promote to active** → state flip.
- **Draft PRD** → `/prd` → `prd-author`.
- **Write Pitch** (cycle preset) → `/pitch` → `pitch-author`.
- **Add intake link** → manual link selector.
- **Show evidence** → `evidence-synthesis` (inline panel).
- **Prioritize** → `/prioritize` → `prioritization-strategist`.
- **Reject with reason** → state flip with reason picker.

### Intake-item (Intake funnel right-detail pane)
- **Link to existing** → roadmap-item picker.
- **Promote to roadmap-item** → quick-create modal,
  `intake-triager`.
- **Reject with reason** → reason picker.
- **Find duplicates** → `duplicate-detector`.
- **Triage** → `/triage` on this single item →
  `intake-triager`.
- **Cluster recent** → `duplicate-intake-scrubber` on
  surrounding items.

### Handoff-stream row (Cross-domain handoff stream)
- **Open story** → navigate to Story detail.
- **Open feature** → cross-domain navigate to engineering
  feature.
- **Re-handoff** → `handoff-coordinator` with previous-edge
  history.

---

## H) Sequencing recommendation

### Minimum viable PM pack (v1) — earns the platform's place

The smallest pack that makes the killer demo work and the
five PM principles real, with everything else deferred.

**Agents (13 P0):**
1. `pm-delivery-lead`
2. `pm-investigator`
3. `product-strategist`
4. `discovery-researcher`
5. `prd-author`
6. `story-writer`
7. `roadmap-curator`
8. `intake-triager`
9. `duplicate-detector`
10. `prioritization-strategist`
11. `handoff-coordinator` *(non-negotiable — this is the
    killer demo)*
12. `pm-reviewer`
13. *(none from coaches / scrubbers / phase planners in v1)*

**Skills (v1 set — ~18 of 32):**
- Writing: `story-writing-invest`, `acceptance-criteria-ears`,
  `prd-structure`, `prd-anti-patterns`,
  `pitch-writing-shape-up`, `roadmap-framing`
- Frameworks: `prioritization-frameworks`,
  `opportunity-solution-trees-torres`, `metrics-design`
- Process: `continuous-discovery-cadence`, `sprint-planning`,
  `cycle-planning` *(load lightly; cycle-planner agent is P1)*
- Curation: `intake-classification`, `duplicate-detection`,
  `dependency-mapping`, `evidence-synthesis`
- Cross-domain: `handoff-protocol`, `cross-domain-graph-query`
- Operational: `pm-preset-detection`

**Commands (v1 set — 12):**
- New: `/refine`, `/triage`, `/roadmap`, `/prioritize`,
  `/prd`, `/pitch`, `/handoff`, `/discover`, `/metrics`,
  `/release-notes`
- Reused: `/design` (accepting PM artifact types), `/why`
  (cross-domain), `/search`, `/note`

### v1.5 additions

P1 agents, especially those that unlock day-to-day cadence:
- `capacity-planner`, `cycle-planner` (covers sprint / cycle /
  iteration variants)
- `pitch-author` (split out from `prd-author` when cycle teams
  adopt Hero in numbers)
- `epic-framer`, `dependency-mapper`, `risk-curator`
- `stakeholder-communicator`
- `competitive-analyst`, `metrics-analyst`
- `roadmap-reviewer`, `stale-roadmap-scrubber`,
  `duplicate-intake-scrubber`

Plus the P1 commands (`/capacity`, `/plan-*`, `/standup`,
`/scrub <concern>`, `/interview`) and the P1 skills,
including the methodology skills now packaged as
`opportunity-solution-trees-torres`, `continuous-discovery-cadence`,
`pitch-writing-shape-up`, `shape-up-cadence`, `hill-chart-reasoning`,
loaded by `pm-delivery-lead` and authoring agents on demand.

### v2 / later

- `okr-design` skill + a future `okr-author` agent (if OKRs
  land in PM rather than spinning off to a `strategy` domain
  per spec.md unknown #1).
- `portfolio-curator` (multi-product / multi-team).
- `discovery-reviewer` (rigor pass on discovery output —
  rare workflow).
- `ambiguous-story-scrubber` (the other two scrubbers cover
  most of the surface; this is polish).

---

## End notes

This design intentionally does **not**:
- Write the agent prompts (those land at `/deliver hero-pm`
  time under `domains/pm/agents/`).
- Invent new artifact types beyond what `spec.md` declares
  (`prd`, `story`, `epic`, `roadmap-item`, `intake-item`).
- Reshape the UX (the IDE-style layout is locked).
- Pre-commit to OKR support, cross-tracker handoff, or
  domain coexistence model — see `spec.md` unknowns 1–5.
