# Hero QA — Mockup Brief for ui-designer

Audience: `ui-designer` agent (and any human designer reviewing the
mockups). This brief is a self-contained set of instructions for
producing **eight killer-screen HTML mockups** for the Hero QA domain
pack. Each screen brief stands alone — a fresh ui-designer session
can act on any one without re-reading the others, though the preamble
applies to all eight.

Read `research-brief.md` (sibling file) first if you want the deep
competitive analysis behind these screens. This file gives you the
*what to build*; the research brief gives you the *why these patterns*.
The QA pack reuses PM's UX grammar wholesale (L1 in spec.md); refer
to `../hero-pm/mockup-brief.md` for the shared layout/density/copy
conventions if anything below is ambiguous.

---

## Preamble — shared design grammar

### Headline UX target

> *TestRail's case-as-document fidelity + Xray's story-test edge
> visibility + mabl's AI-authored-at-sprint-speed + QA Wolf's coverage
> commitment — all rendered inside Linear-class UI, with cross-pack
> lifecycle states (qa-ready / qa-rejected) where no QA tool today
> can reach.*

### Layout grammar — inherited from PM

Same three-column shell as PM, same chrome rules, same density
target. **Do not redesign.** Differences for QA:

- **Left nav** sections (in order):
  - Top: workspace name + domain pill (`QA`) with dropdown to
    switch to `Engineering` or `PM`.
  - QA view links: **Story queue** (default, coverage signals),
    **Test plans**, **Test cases**, **Regression suites**,
    **Flaky tests**, **Release gates**, **Unified inbox**,
    **Handoff stream**.
  - Active sessions mini-list near the bottom.
  - Domain switcher / settings at the very bottom.
- **Content pane:** screen-specific.
- **Right rail (~320px, collapsible):** ambient QA panel + activity
  + metadata. Always has the QA-specific ambient cards described
  per screen.
- **Bottom strip** (~70-90px tall, anchored to artifact tabs only —
  not present on Story queue / Unified inbox / Handoff stream
  singletons): state-aware contextual buttons (verbs) + chat input
  + model chip. Follow `agent-pack-design.md` §G for per-artifact
  button inventory.

### Color additions

Inherit PM's palette. Add QA-specific accent colors:

```css
:root {
    /* ...inherit PM tokens... */
    --accent-coverage-full: #16a249;     /* green for full coverage */
    --accent-coverage-partial: #d97706;  /* amber for partial */
    --accent-coverage-none: #dc2626;     /* red for uncovered */
    --accent-qa-gutter: #0891b2;         /* cyan teal for QA-authored content gutter */
    --accent-qa-ready: #0284c7;          /* blue chip for qa-ready state */
    --accent-qa-rejected: #d97706;       /* amber chip for qa-rejected */
    --accent-flaky: #a855f7;             /* violet for flaky indicator */
    --accent-pass: #16a249;              /* green for pass */
    --accent-fail: #dc2626;              /* red for fail */
    --accent-blocked: #6b7280;           /* gray for blocked */
}
```

### Methodology preset

Every screen must accommodate the active QA preset combination
specified via chrome chips. Canonical preset for mockup delivery:
**`gate_style: inline` + `case_format_default: step` + `rejection_strictness: block`
+ `test_issue_persistence: persistent-link`** — this is the
opinionated default and exercises the most preset-driven UI.

Surface preset chips in top-right chrome: `Inline gate · Step-by-step · Persistent-link`.

### Realistic sample copy

Use Hero-flavored, plausible QA content. Sample identities reused
from PM: Sarah Chen (Engineering Lead), Marcus Johnson (PM), Aisha
Patel (Design), Diego Ramirez (Engineering), Priya Shah (PM). Add:
**Jamie Liu** (QA Lead), **Tomás Reyes** (QA Engineer), **Yuki Tanaka**
(QA Automation). No lorem ipsum, no "Foo / Bar / Baz."

Sample story content reused from PM (agent telemetry, observability,
trace spans, etc.) — this lets the cross-domain demos read naturally
when QA mockups reference the same PM stories.

### File output convention

Each screen ships as one self-contained HTML file under
`.hero/mocks/hero-qa/`. Filenames:

- `01-story-queue-coverage.html`
- `02-test-plan-editor.html`
- `03-test-case-editor.html`
- `04-story-qa-ready-rejection.html`
- `05-release-gate.html`
- `06-flaky-backlog.html`
- `07-unified-inbox.html`
- `08-handoff-stream-qa.html`

Follow `html-mockup-generation` skill conventions: single file,
inline `<style>` and `<script>`, no CDNs, no external assets. SVG
inline for any icons. Realistic data; no lorem.

---

## Screen 1 — Story queue with coverage signals (default landing)

> **The default landing page.** Earns principles #1 (cover every
> story), #2 (author at sprint speed), #3 (block before ship).
> Influences: Pivotal Tracker (single-list flow), Linear (J/K +
> density), QA Wolf (coverage % visible), Xray (story-test edges).

### Purpose

The QA lead's home. A single scrolling list of all sprint stories
with **coverage signal per row** and quick-access to authoring +
triage actions. The lead can scan coverage state at a glance and
drill into any story.

### Layout

- **Left nav:** shared shell. "Story queue" highlighted.
- **Top chrome additions:** view-toggle segmented control:
  - `Sprint 23 (active)` ← default, shows current sprint stories
  - `Backlog`
  - `All in-flight`
  - `My triage queue`
- **Content pane:**
  - Sprint summary strip at top — single row showing:
    - Sprint name + dates: "Sprint 23 · May 13 – May 26"
    - Coverage rollup: 8/12 stories fully covered, 3 partial, 1
      uncovered (color-coded mini bar).
    - Estimated authoring budget: "12 cases left to author · ~35 min."
    - Action: `Plan sprint coverage` button (right side, primary).
  - Single-list flow of story rows (Pivotal-shaped). Each row 32px:
    - **Left:** drag handle (8px, hover-visible only).
    - **Type chip + state chip:** `story · qa-ready` (qa-ready in
      blue accent chip).
    - **Coverage chip** *(QA-unique)*: small icon + count.
      - Full (green): `✓ 5/5`
      - Partial (amber): `⚠ 3/5`
      - None (red): `! 0/5`
      - Tooltip: "5 acceptance criteria, 5/3/0 covered by passing
        cases."
    - **Title:** truncated to one line.
    - **Right cluster:** assignee avatar (8px), Jira/tracker chip
      `STORY-2847`, last-activity time ("2h ago").
  - Group stories under bands:
    - **In QA** (top, brand-relevant — stories in `qa-ready` or
      `qa-rejected`) — show 2-3 stories here.
    - **In flight** (engineering working) — show 4-5 stories.
    - **Ready** (waiting for engineering pickup) — show 3 stories.
    - **Done this sprint** (already signed off) — show 2 stories,
      collapsed-by-default.
  - **Selected row** (one story highlighted in the In-QA band) —
    pre-select Story-2847 (the trace-spans story from PM mockups
    for cross-mockup consistency).
- **Right rail:**
  - **Ambient QA panel** for the selected story:
    - Card 1: "Coverage: 5 AC, 3 cases drafted, 2 missing for
      AC#3 and AC#4." → `Author missing cases` button.
    - Card 2: "Engineering hand-off summary: Diego shipped on
      branch `agent-trace-spans` 2h ago. Ready for QA." →
      `View linked feature` button.
    - Card 3: "Similar story shipped 4mo ago: Story-2691. 8 cases
      then; only 4 still in regression suite. Worth comparing?"
  - **Quick actions** strip below ambient: `Author cases`,
    `Reject (compose findings)`, `Approve & mark Complete`.
    These echo the bottom strip when the story is opened in
    detail, but are accessible here for fast triage.
  - **Activity stream** below.
- **Bottom strip** is NOT present on the queue (it's a singleton
  view). The right-rail Quick actions stand in. When the user
  opens a story (Screen 4), the bottom strip lights up there.

### Interactions

- **Coverage chip** hover shows breakdown popover: which AC
  covered by which case, with case ID + last-run status.
- **`Plan sprint coverage`** button (top right) opens a modal
  showing the per-story budget (Story A: 4 cases, Story B: 6
  cases…) with `Accept & start authoring` action.
- **J/K** moves between rows (vim-style); **Enter** opens story
  detail (Screen 4).
- **Drag handle** lets QA reorder *within a band* (priority
  within QA's queue, separate from sprint priority).
- **Coverage chip → click** opens Test plan editor (Screen 2)
  for the story.

### Must NOT do

- Don't make coverage feel optional. Red `!` for uncovered should
  be impossible to miss.
- Don't bury authoring actions in a right-click menu. `Author
  cases` is one click from the queue.
- Don't show flake count as a story-row chip — flakes live on
  the case level, not the story level.
- Don't render bands as kanban columns. Single vertical list
  with band headers is the Pivotal-shaped mental model.

### Preset variations

- **Parallel gate_style:** The "In QA" band disappears (QA doesn't
  gate); instead a `qa_state` chip renders inline on each row
  showing the QA verification state without affecting story
  position.
- **Continuous flow preset** (kanban delivery layer from PM):
  No sprint summary strip; replace with "Coverage by area" rollup.
  Bands become Icebox / Backlog / Current / Done with WIP-age
  indicators.

---

## Screen 2 — Test plan editor (coverage matrix)

> **The PRD-equivalent for QA.** Earns principles #1 (cover every
> story), #2 (author at sprint speed). Influences: TestRail (plan
> shape) + qTest (traceability matrix) + Xray (story-link panel)
> + Functionize (NL composer for ad-hoc cases).

### Purpose

Edit a test plan with a coverage matrix that maps story AC to
test cases. The matrix is the heart of the screen — it's the QA
artifact that proves coverage is real.

### Layout

- **Left nav:** shared shell. "Test plans" highlighted.
- **Content pane:**
  - Breadcrumb: `Sprint 23 ›  Story-2847 ›  Plan: agent-trace-coverage`
  - Plan title as h1: "Coverage plan for `Story-2847: agent run
    trace spans`"
  - Metadata pills below title: type `test-plan`, state
    `in-flight`, owner `Jamie Liu`, linked story `STORY-2847`,
    linked release `2026-05-26`.
  - Tab strip below pills: `Strategy` · `Coverage matrix (active)` ·
    `Cases (5)` · `Runs (2)` · `History`.
  - **Coverage matrix** (the centerpiece, when active tab):
    - A table with AC rows × Case columns. Each cell is empty
      (uncovered), filled with a chip (covered), or marked
      `?` (partial — case touches AC but doesn't fully verify it).
    - Example matrix (5 AC × 5 cases):
      ```
                              C1   C2   C3   C4   C5
      AC1: trace span emit    ✓    ·    ·    ·    ·
      AC2: child spans/tool   ·    ✓    ·    ·    ·
      AC3: retry on flush     ·    ·    ✓    ✓    ·
      AC4: parent-session-id  ✓    ✓    ·    ·    ·
      AC5: backoff & drop     ·    ·    ✓    ·    ✓
      ```
    - Coverage rollup at the bottom: "5/5 AC covered · 5 cases · 12 last-run results · 11 pass · 1 fail."
    - A `Print as compliance report` button at the top-right of
      the matrix (qTest-shaped traceability export).
  - **Coverage gaps callout** when partial — show as a yellow strip
    above the matrix: "AC#4 partially covered — happy path only,
    no negative case." with `Author negative case` button.
- **Right rail:**
  - **Ambient panel:**
    - Card 1: "AC#3 has 2 cases (C3, C4). Possible redundancy —
      `regression-curator` suggests demoting C4 once tests pass.
      Promote one to regression?"
    - Card 2: "Story marked `qa-ready` 2h ago. Authoring budget
      remaining: 0 (you're done!)"
    - Card 3: "5 cases × 4 AC × 1 last-run = good plan integrity."
  - **Plan metadata** (collapsible): created, scope, out-of-scope,
    risk areas.
  - **Activity:** "Tomás added Case-C5 — 1h ago"; "Jamie ran cases
    C1-C5 against build #1284 — 1h ago: 4 pass / 1 fail."
- **Bottom strip:**
  - State `in-flight` buttons (from agent-pack §G):
    `Run remaining`, `Show run progress`, `Mark blocked`,
    `Author missing cases`, `Promote cases to regression`.
  - Chat input below buttons.

### Interactions

- **Matrix cells** are clickable:
  - Empty cell → "Author a case to cover AC#X" inline-propose flow.
  - Filled cell → opens the case popover with last-run status.
- **Row hover** highlights the AC + all linked cases in the
  Cases tab.
- **Print report** opens a print-styled view of the matrix
  suitable for compliance binders.

### Must NOT do

- Don't render the matrix as a wall of equally-sized cells
  that scrolls horizontally forever — the canonical case-set
  for a story is 4-8 cases; design for that.
- Don't hide the coverage gap callout if partial — it's the
  primary signal the QA lead acts on.
- Don't make print-report a v2 feature in the mockup — show
  the button.

### Preset variations

- **Gherkin format default:** the Cases tab shows `Scenario:` /
  `Feature:` blocks instead of step rows. The matrix shape is
  unchanged (Gherkin scenarios are still cases that cover AC).
- **Decision-table format:** cells in the matrix expand to show
  small decision-table previews on hover.

---

## Screen 3 — Test case editor (format-aware)

> **The case-as-document fidelity demo.** Earns principle #2
> (author at sprint speed). Influences: TestRail (step grid) +
> mabl (plain-English step library) + Testim (stability score)
> + Gherkin (Cucumber/SpecFlow).

### Purpose

Show a single test case in full fidelity with the format toggle,
run history, and stability score. This is where the AI-authored
case proves its quality.

### Layout

- **Left nav:** shared shell. "Test cases" highlighted.
- **Content pane:**
  - Breadcrumb: `Plan: agent-trace-coverage ›  Case-C3`.
  - Case title as h1: "Verify retry-with-backoff when initial
    flush fails."
  - Metadata pills: type `test-case`, state `ready`, format
    `step-by-step (default)`, last-run `1h ago — passed`,
    stability `94% over last 30 runs`, linked AC `Story-2847 AC#3`,
    quadrant `Q2 functional`.
  - **Format toggle** segmented control (small, near title):
    `Step-by-step` (active) · `Gherkin` · `Decision table` ·
    `Data-driven`. Toggling re-renders the same case content
    in the alternate format.
  - **Preconditions block** — 1-2 sentences. Editable. Example:
    "Agent runtime in test environment. Spans backend reachable
    on port 4317."
  - **Steps grid** (default step-by-step format):
    - 4 rows, columns: `#`, `Action`, `Expected Result`, `Notes`.
    - Step 1: "Start agent run with id `test-run-001`" →
      "Trace span emitted with run-id and parent-session-id."
    - Step 2: "Inject network fault on span flush endpoint" →
      "Flush returns error; agent enters retry state."
    - Step 3: "Allow 3 retry cycles to complete" →
      "Retries logged with backoff (1s, 2s, 4s)."
    - Step 4: "Observe span queue after 4th attempt fails" →
      "Span dropped; drop event logged to operational telemetry."
    - Each row has hover-visible drag handle + Notes affordance.
  - **AC traceability strip** below steps: "This case covers AC#3
    of `Story-2847: retry on flush`" — clickable to open story.
  - **Run history** panel below traceability:
    - Last 10 runs as small chips: `✓ ✓ ✓ ✓ ✓ ✗ ✓ ✓ ✓ ✓` color-coded.
    - Click any chip to see run detail (timestamp, build, env,
      result, log link to integration).
    - Stability score: `94%` with sparkline (last 30 days).
- **Right rail:**
  - **Ambient panel:**
    - Card 1: "Generated from AC#3 by `test-author` on May 12 —
      derived via boundary-value-analysis + state-transition.
      Trace: AC#3 → derivation skill → step 1-4."
    - Card 2: "Similar case in `regression-suite-agent-runtime`:
      `RC-441`. `regression-curator` suggests promotion when
      stability ≥ 95%."
    - Card 3 (only if flaky): "Stability dipped to 86% last
      week — `qa-flake-curator` classified two failures as
      environment-issue. View verdict →" (hide in canonical
      mockup since stability is 94%).
  - **Sidebar metadata:** linked plans (1), linked AC (1),
    quadrant tag, format, version (v3 of 3).
- **Bottom strip:**
  - State `ready` buttons (from agent-pack §G):
    **`Run`** *(primary)*, `Convert format`, `Tag (quadrant /
    regression / flaky)`, **`Request test seam`** *(brand if
    applicable; show this case has no seam request — button is
    secondary)*.

### Interactions

- **Format toggle** — clicking `Gherkin` re-renders the steps as:
  ```
  Feature: Trace span resilience
    Scenario: Retry with backoff when initial flush fails
      Given the agent runtime in test environment
       And spans backend reachable on port 4317
       When agent run "test-run-001" is started
        And network fault is injected on span flush endpoint
        And 3 retry cycles complete
       Then retries are logged with backoff (1s, 2s, 4s)
        And span is dropped after the 4th attempt
        And drop event is logged to operational telemetry
  ```
  Same case, different render. Mockup should show step-by-step
  view as canonical, with the toggle prominent.
- **Run chip click** opens a small popover with run detail.
- **Drag step rows** to reorder (hover-visible drag handle).
- **`Request test seam`** — when clicked (mockup may show
  hover tooltip): opens a composer to describe what engineering
  needs to add for testability.

### Must NOT do

- Don't render Gherkin as the default. Step-by-step is the
  default per `qa.case_format_default: step` (canonical preset).
- Don't bury run history below the fold. The run chips and
  stability score live above the right rail.
- Don't show the AC traceability as a small ID link. It should
  be a contextual strip that reminds the reader *why this case
  exists*.

### Preset variations

- **Gherkin preset default:** swap the canonical rendering to
  Gherkin. Step-by-step becomes the alternate toggle.
- **Decision-table format active:** replace steps grid with a
  condition × rule × action table (only if the AC's shape
  warrants it — for Case-C3 it doesn't, so this preset variation
  is best illustrated by a *different* case as the canonical
  example).

---

## Screen 4 — Story view in `qa-ready` (three-action rejection composer)

> **THE brand interaction.** Earns principles #3 (block before
> ship), #6 (prevent scope-creep through the gate). Influences:
> Pivotal Tracker (Reject as state action) + Linear (right rail
> density) + Hero design-original on the three-action composer.

### Purpose

Show a PM story in `qa-ready` state from the QA pack's perspective.
The bottom strip's `Reject (compose findings)` button is open mid-
composition, showing the three-action composer in action. This is
the screen demos open on.

### Layout

- **Left nav:** shared shell. "Story queue" highlighted.
  Right above the left-nav body, show a domain pill `QA →` and
  next to it a small chip `Viewing PM story` indicating cross-
  domain context.
- **Content pane:**
  - Breadcrumb (cross-pack): `PM: Roadmap-item: agent observability
    ›  Story-2847 (PM-owned, QA viewing)`.
  - Story title as h1 — same as PM Screen 1.
  - Metadata pills: type `story`, state **`qa-ready`** (blue
    accent chip — the brand state), priority `high`, points `5`,
    assignee `Diego Ramirez`, linked feature `feature: agent-
    trace-spans (delivered)`.
  - **Coverage rail** (above description, QA-unique):
    - 5 AC × 5 cases pass-state strip. 4 green, 1 red ("AC#3:
      retry on flush — 1 of 2 cases failing").
    - `Open test plan` link.
  - **Description block** (read-only here — QA doesn't edit PM
    body except via QA Findings).
  - **Acceptance Criteria** section — same EARS bullets as PM
    Screen 1.
  - **QA Findings — Round 2** *(collapsible, currently empty —
    we're composing now)*:
    - Empty state with placeholder: "QA Findings appear here
      after rejection."
- **Bottom strip — IN COMPOSITION:**
  - Top of bottom strip shows the **three-action composer**:
    ```
    ┌────────────────────────────────────────────────────────────┐
    │ Compose finding for AC#3 (retry on flush)                  │
    │                                                            │
    │ ◉ Add AC to this story  (in current scope)                 │
    │ ○ Suggest new story    (out of scope — prevents creep)     │
    │ ○ Reject as quality issue  (implementation doesn't meet AC)│
    │                                                            │
    │ Findings: [textarea, partially filled]                     │
    │ "Retry succeeded but backoff sequence was 0.5/1/2 instead │
    │  of 1/2/4 specified in AC. Implementation appears to use  │
    │  a 0.5s base — likely a config default..."                │
    │                                                            │
    │ qa-investigator suggests: this contradicts existing AC →  │
    │ "Reject as quality issue" (default nudge).                │
    │                                                            │
    │ [Cancel] [Submit rejection]                                │
    └────────────────────────────────────────────────────────────┘
    ```
  - **`Submit rejection`** is the primary button (right, indigo).
- **Right rail:**
  - **Ambient QA panel:**
    - Card 1 (high priority): "AC#3 is failing in 1 of 2 cases.
      Suggest classifying as `Reject as quality issue`."
    - Card 2: "Story has been in `qa-ready` for 3h. Engineering
      session is open in Diego's editor — he'll see the rejection
      ambient card as soon as you submit."
    - Card 3: "Previous rejection of Story-2847 (round 1, 4d ago)
      → 'Add AC' for retry semantics. Engineer addressed in
      commit `f3a912`."
  - **Engineering feature rail** (cross-domain decoration):
    - Card: `feature: agent-trace-spans` · state `delivered` ·
      assignee `Diego Ramirez` · last commit `f3a912 (Diego):
      add retry backoff sequence`.
    - Show preview of the ambient card that **will appear** in
      Diego's engineering session after rejection: "Your linked
      story Story-2847 was QA-rejected (round 2). Findings: retry
      backoff sequence doesn't match AC#3. Open story →"

### Interactions

- **Three-action composer** is the centerpiece. Radio buttons
  drive different default findings templates.
  - **Add AC** → expands a `New AC (EARS-shaped)` editor below
    the findings textarea.
  - **Suggest new story** → expands a `New story title` + `Why
    out of scope` editor; on submit, creates intake-item.
  - **Reject as quality issue** → expands findings textarea
    only; on submit, augments story with QA Findings block.
- **Default selection** comes from `qa-investigator`'s nudge —
  shown as a hint below the radios.
- **Submit** animates: composer closes; story state pill
  transitions `qa-ready` → `qa-rejected` with a brief flash;
  the `QA Findings — Round 2` block expands to show the just-
  added content with QA-cyan gutter icon; cross-pack ambient
  card prepares (toast says "Diego notified in engineering
  session").

### Must NOT do

- Don't show the three-action composer as a modal — it lives
  inline in the bottom strip where the brand interaction belongs.
- Don't hide the "Suggest new story" option. It's the anti-
  goalpost-moving path; it must look as primary as the others.
- Don't show the rejection animation as a destructive red.
  `qa-rejected` is part of normal lifecycle, not error state.
- Don't omit the cross-pack ambient preview on the right rail.
  Showing what Diego will see *before* submit is the silo-tearing
  proof.

### Preset variations

- **Parallel gate_style:** the composer's `Reject as quality issue`
  outcome doesn't transition story state (story stays in its main
  lifecycle); instead the `qa_state` chip flips to `rejected`. Bottom
  strip wording adjusts: "Mark verification: rejected."
- **Rejection strictness `advise`:** the rejection raises a warning
  on the story but doesn't block transition to `done`. The submit
  button label changes to `Submit advisory rejection`.

---

## Screen 5 — Release-gate view (Go / No-Go)

> **The release-readiness destination.** Earns principles #3
> (block before ship), #4 (unify without migrating). Influences:
> qTest (traceability matrix) + QA Wolf (coverage commitment) +
> TestRail (milestone roll-up).

### Purpose

The QA destination for release planning. Aggregates sprint
coverage + regression state + blockers into a Go/No-Go verdict.
This is what QA shows leadership before a release.

### Layout

- **Left nav:** shared shell. "Release gates" highlighted.
- **Content pane:**
  - Breadcrumb: `Release gates ›  2026-05-26 (Sprint 23 cut)`.
  - Release title as h1: "Release readiness: 2026-05-26"
  - Metadata pills: type `release-gate`, state `reviewing`,
    owner `Jamie Liu`, sprint `23`, linked release
    `release-2026-05-26`.
  - **The verdict banner** (top of content pane, the centerpiece):
    ```
    ┌─────────────────────────────────────────────────────────────┐
    │   ⚠ NO-GO — 2 blockers, 1 awaiting sign-off                 │
    │                                                             │
    │   Coverage: 11 of 12 stories fully covered ✓               │
    │   Regression: 142/148 passing (96%) ⚠                       │
    │   Blockers: 2 P0 open, 1 P1 awaiting sign-off               │
    │                                                             │
    │   [Re-aggregate] [Override to Go] [Defer to next cycle]    │
    └─────────────────────────────────────────────────────────────┘
    ```
    Verdict colored amber (No-Go but recoverable, not catastrophic).
  - **Blockers section:** named list of 3 items:
    - **B1 (P0):** `bug: agent-trace-flush-loops` — Diego, blocking
      retry mechanism. Open since 2d. `Open bug →`
    - **B2 (P0):** `bug: span-id-collision-under-load` — Aisha
      assigned, in `delivering`. ETA tomorrow. `Open bug →`
    - **B3 (P1):** Story-2853 — coverage 0/4. Awaiting QA
      sign-off override. `Open story →` `Sign off →`
  - **Coverage breakdown** (collapsible):
    - 12 stories, 11 fully covered, 1 partial (Story-2853).
    - Mini per-story coverage bars.
  - **Regression snapshot** (collapsible):
    - 148 cases in active regression suites, 142 passing, 6
      failing, 3 flaky (verdict pending).
    - Per-suite breakdown: `agent-runtime-regression` 89/89,
      `telemetry-regression` 28/30, `webhooks-regression` 25/29.
  - **Sign-off log** (collapsible): "Jamie signed off Sprint 23
    coverage on May 25"; "Marcus signed off PM scope on May 25";
    "Sarah pending engineering sign-off."
- **Right rail:**
  - **Ambient panel:**
    - Card 1: "B1 has been open 2 days. `release-gate-reviewer`
      recommends gate stays No-Go until resolved."
    - Card 2: "Sign-off pending from Sarah (engineering). Hero's
      `handoff-coordinator` is notifying her session."
    - Card 3: "If B1 + B2 close today, gate flips to Go pending
      B3 sign-off."
  - **Blocker policy** (collapsible): your team's configured
    rules (defaults: P0 hold, P1 hold without sign-off, P2+
    candidate for next cycle).
- **Bottom strip:**
  - State `reviewing` buttons (from agent-pack §G):
    `Approve (go)` *(primary, disabled while blockers open)*,
    `Reject (no-go with reasons)`, `Defer to next cycle (per
    blocker)`.

### Interactions

- **Re-aggregate** button refreshes the verdict against current
  state — useful after a blocker closes.
- **Override to Go** is destructive — requires confirmation modal
  with reason capture (logged on the gate spec).
- **Defer blocker → next cycle** moves a blocker out of the
  current release-gate's scope; written to the next gate's open
  list.
- **Per-blocker `Sign off` button** is the fast-path resolution
  for P1+ blockers that have explicit sign-off authority.

### Must NOT do

- Don't render the verdict as a status pill. It's a banner —
  the most prominent thing on the screen.
- Don't allow override-to-Go without reason capture.
- Don't hide the sign-off log. Audit traceability is part of
  the gate's value.
- Don't render blockers as a wall of text. Named list with
  P0/P1 distinction and direct actions.

### Preset variations

- **Strict blocker policy** (`block` on P0 and P1): no override
  affordance.
- **Permissive policy:** override available with reason capture
  and additional sign-off requirement.
- **Phased release preset:** verdict applies per phase; blockers
  attach to phases.

---

## Screen 6 — Flaky tests backlog (active queue with verdict workflow)

> **The opinionated zero-flake drive.** Earns principle #5 (drive
> flakiness to zero). Influences: mabl (verdict classification)
> + Testim (stability score) + GitHub Issues triage UX.

### Purpose

Active backlog of flaky tests with verdict classification per
failure. Designed to drive flake count *toward zero*, not just
track it.

### Layout

- **Left nav:** shared shell. "Flaky tests" highlighted.
- **Content pane:**
  - Top strip with the leaderboard: "Flake count: **8** (down
    from 14 last week). Goal: 0 by end of sprint."
    - Mini sparkline (last 30 days) showing trend.
    - `Re-classify all` button.
  - View toggle: `Unclassified (3)` (default) · `In progress (4)` ·
    `Resolved this sprint (12)` · `All`.
  - Single-list of flaky cases (32px rows):
    - **Row 1 (unclassified, default selected):**
      - Stability chip: `47%` (red).
      - Case title: `RC-441 — span emission under concurrent runs`.
      - Last 10 runs: `✓ ✗ ✓ ✗ ✗ ✓ ✓ ✗ ✓ ✗`.
      - Verdict pending pill.
      - Linked regression-suite: `agent-runtime-regression`.
    - **Row 2 (unclassified):**
      - Stability: `71%`.
      - Case: `RC-388 — webhook retry timing`.
      - Linked: `webhooks-regression`.
    - **Row 3 (unclassified):**
      - Stability: `58%`.
      - Case: `RC-512 — trace span queue depth`.
    - **Row 4 (in progress, classified test-issue):**
      - Stability: `66%`.
      - Verdict: `test-issue · fix-by 2026-05-22 · Tomás`.
    - **Row 5 (in progress, classified environment):**
      - Stability: `81%`.
      - Verdict: `environment · fix-by 2026-05-20 · Yuki`.
- **Right rail (for selected RC-441):**
  - **Ambient panel:**
    - Card 1: "`qa-flake-curator` classification: **test-issue**
      (locator instability under concurrent runs). Confidence: 87%."
    - Card 2: "Failures cluster around build #1280-1284. Inspect
      env? View run logs →"
    - Card 3: "If true-bug verdict: route to `/diagnose` chain;
      will create engineering `bug` linked to RC-441."
  - **Case details strip:** linked AC, linked plan, last 30
    failures grouped by error signature (top 3 signatures shown).
- **Bottom strip:**
  - Verdict-pending buttons:
    **`Classify (test-issue / environment / true-bug)`** *(primary
    — opens a sub-menu)*, `Show run history`, `Quarantine`.
  - On classification:
    - test-issue → opens fix-by deadline picker.
    - environment → opens "what env factor" tag picker.
    - true-bug → hands off to `/diagnose` (creates engineering
      bug); confirmation modal first.

### Interactions

- **Classify menu** is a quick split-button: hover shows three
  verdict options with explanations.
- **`Quarantine`** moves the case to a separate `quarantined`
  state (out of regression suite) until verdict reached.
- **`Re-classify all`** re-runs `qa-flake-curator` heuristics
  on every unclassified case — useful after new failure data.

### Must NOT do

- Don't render flake count as a passive metric. It's a
  leaderboard with a goal.
- Don't allow "ignore" as a verdict — that's the failure mode
  the pack opposes. Quarantine is the only "don't engage now"
  action, and it requires a verdict eventually.
- Don't bury fix-by deadlines. Each classified case shows the
  deadline on its row.
- Don't auto-classify silently. The agent's classification is
  always a suggestion; the human confirms.

### Preset variations

- **No CI integration:** flakes are manually marked rather than
  auto-detected from run history. The screen looks the same;
  the data flow upstream is different.
- **Permissive flake stance** (opt-out of the opinionated
  zero-flake stance): a `Snooze indefinitely` button appears
  alongside Quarantine. Don't ship this in v1 default mockup —
  it's an anti-pattern hint, but the platform supports it.

---

## Screen 7 — Unified inbox (kind tabs: All / Bugs / Test issues)

> **The unification thesis made visible.** Earns principle #4
> (unify without migrating). Influences: Linear (Triage view shape,
> command palette) + Productboard (source-tagged inbox) +
> PractiTest (filter-as-URL composability).

### Purpose

One queue for bugs (from Jira / GitHub) and test issues (from
TestRail / Xray / native), normalized via the standard schema,
decorated by source. The destination for cross-source triage.

### Layout

- **Left nav:** shared shell. "Unified inbox" highlighted.
- **Top chrome additions:**
  - Kind tabs in the content pane header: `All (47)` *(active)* ·
    `Bugs (28)` · `Test issues (19)`.
  - Filter strip below tabs: source pills (Jira `28` · TestRail
    `12` · Xray `7` · GitHub `0` · Native `0`), status filters,
    priority filters. Each filter is URL-encodable.
- **Content pane:**
  - Single-list flow of inbox items (32px rows):
    - **Row 1 (test issue, TestRail):**
      - Kind chip: `test-issue` + source pill `TestRail`.
      - Title: `RC-441 failed on build #1284 — assertion mismatch`.
      - Normalized: priority `high`, severity `medium`, age `2h`.
      - Source decoration (hover): TR-specific priority "P2", custom
        severity label "regression".
      - Linked: `case RC-441`, `regression-suite agent-runtime`.
      - Triage pending.
    - **Row 2 (bug, Jira):**
      - Kind chip: `bug` + source pill `Jira`.
      - Title: `BUG-9912 — agent trace span ordering broken under
        load`.
      - Normalized: priority `high`, severity `high`, age `4h`,
        assignee `Diego Ramirez`.
      - Source decoration: jira_priority "Critical", jira_status
        "In Progress".
      - Linked: `feature: agent-trace-spans`.
    - **Row 3 (test issue, Xray, classified flake):**
      - Kind chip: `test-issue` + source pill `Xray`.
      - Title: `XRAY-3382 — span queue depth flake`.
      - Verdict: `environment · fix-by 2026-05-20`.
    - **Row 4 (bug, Jira, regression):**
      - Kind chip: `bug` + source pill `Jira` + regression badge.
      - Title: `BUG-9889 — webhook retries reset on restart
        (regression of 2026-04 fix)`.
    - …continue with 6-8 more rows mixing bugs and test issues.
  - **Selected row (Row 1)** in highlight state.
- **Right rail (for selected Row 1):**
  - **Triage panel (the brand surface):**
    - Four primary actions:
      - **`Mark as bad test`** — close test issue; flag RC-441
        for fix.
      - **`Reject linked story`** — open rejection composer
        against Story-2847.
      - **`Raise as new bug`** — open `/diagnose` chain with
        case context pre-loaded.
      - **`Flag as regression`** — open `/diagnose` + flag
        regression suite.
    - `qa-investigator` nudge: "Assertion mismatch on third
      retry attempt. This contradicts AC#3 of Story-2847. Suggest
      **Reject linked story**."
  - **Cross-kind relationships rail:**
    - "Linked case: RC-441" → opens Test case editor.
    - "Possibly related: BUG-9912 (same build, retry-mechanism
      area). Link as `raised-bug-from-failure`?"
  - **Source decoration:** TestRail field values rendered in a
    "From source" collapsible: TR priority, TR custom fields,
    TR comments preview, link to TestRail.
- **Bottom strip:**
  - Singleton view — bottom strip not present. Triage panel in
    right rail carries the four-action verbs.
- **Local-first decoration** (when no integration configured):
  - Empty state strip at top: "No integrations configured. You're
    seeing local bugs (8) and native test failures (4)."
  - Source pills show `Local` for engineering pack bugs and
    native test failures. CTAs to "Add Jira integration" /
    "Add TestRail integration" present but non-blocking.

### Interactions

- **Kind tab click** filters rows by kind (`Bugs` / `Test issues`).
- **Source pill click** toggles source filter (clicking `TestRail`
  shows only TestRail rows).
- **Filter expression** is URL-encoded — sharing the URL shares
  the filter state.
- **Multi-select** (shift-click or checkbox column toggle in
  header) enables bulk actions: bulk close, bulk reassign, bulk
  triage with confirmation.
- **Triage actions** (4 buttons) each open a confirmation /
  composer flow:
  - `Mark as bad test` → confirm modal "Close this test issue and
    send RC-441 to flake backlog with verdict `test-issue`?"
  - `Reject linked story` → opens Screen 4's three-action composer.
  - `Raise as new bug` → opens `/diagnose` with case context.
  - `Flag as regression` → opens `/diagnose` + regression flag
    composer.
- **`Mark as duplicate of...`** (same-source manual action) is
  available from the row's `…` overflow.

### Must NOT do

- Don't render bugs and test issues as separate columns or
  separate views. The kind tabs are filters, not separate inboxes.
- Don't auto-merge. Same-source duplicate is *manual only*.
- Don't hide source pills. Provenance is part of fidelity.
- Don't make the local-first empty state apologetic. It's a
  feature — Hero works without integration.
- Don't let bulk-close cross sources without an explicit
  confirmation showing the source breakdown.

### Preset variations

- **No integration mode (purely local):** empty state strip is
  prominent; source pills uniform `Local`; triage actions still
  fully functional against local specs.
- **TestRail-only or Xray-only:** filter strip shows only the
  configured source pill.

---

## Screen 8 — Cross-domain handoff stream (QA-side)

> **The cross-pack visibility play.** Earns principle #4 (unify
> without migrating). Influences: Linear (cycle view timeline) +
> Kanbanize (cycle-time histogram) + Aha (strategic-context strip)
> + Hero-shaped graph traversal.

### Purpose

Recent QA-driven cross-domain events: rejections sent back to
engineering, bugs raised from failures, seam requests, regression
promotions. The QA-side mirror of PM's handoff stream.

### Layout

- **Left nav:** shared shell. "Handoff stream" highlighted.
- **Top chrome additions:**
  - Time toggle: `Last 24h` · `Last 7d` *(active)* · `Last 30d` ·
    `All`.
  - Direction filter: `From QA` *(active)* · `To QA` · `Both`.
- **Content pane:**
  - Histogram strip at top: rejections / bugs raised / seams
    requested / promotions per day for the active window.
  - Single-list of handoff events (32px rows, newest first):
    - **Row 1:** "QA rejected Story-2847 (round 2) → Diego, 12m
      ago. Findings: backoff sequence ≠ AC#3."
      - Direction icon: ← (QA → engineering).
      - Cross-domain edge type chip: `qa-rejected`.
      - State: engineer notified.
    - **Row 2:** "QA raised `BUG-9921` from RC-441 failure →
      `/diagnose` chain, 1h ago."
      - Edge chip: `raised-bug-from-failure`.
      - State: bug in `delivering` (Diego).
    - **Row 3:** "Seam request from Case-C7 → engineering feature
      `agent-test-hooks`, 3h ago."
      - Edge chip: `requires-seam`.
      - State: feature in `planning` (Sarah).
    - **Row 4:** "Promoted Case-C3 to `agent-runtime-regression`,
      4h ago." (within-pack event but stream-worthy)
      - Edge chip: `promoted-from-story`.
    - **Row 5:** "QA approved Story-2841 → marked complete, 1d ago."
      - Edge chip: `qa-approved`.
    - …continue with 4-5 more events.
  - **Selected row** (Row 1, default).
- **Right rail (for selected Row 1):**
  - **Edge timeline:** the full lineage from this rejection:
    - 4d ago: Story-2847 entered `qa-ready` (round 1)
    - 4d ago: QA rejected → Add AC for retry semantics
    - 3d ago: Engineer addressed (commit f3a912)
    - 2d ago: Engineering marked story `qa-ready` again (round 2)
    - 12m ago: QA rejected → Reject as quality issue (backoff)
    - now: Engineer notified (ambient card surfaced)
  - **Ambient panel:**
    - Card 1: "Story-2847 has had 2 rejection rounds. If this
      pattern continues, `qa-investigator` recommends pulling
      the linked AC quality into PM via `pm-rejection-router`."
    - Card 2: "Diego's engineering session is open. ETA on
      rework: surface to ask."
- **Bottom strip:** singleton view — not present.

### Interactions

- **Row click** opens the relevant source artifact (story / bug /
  case) and brings the edge into focus.
- **Timeline** is scrollable; hovering an event shows the actor
  and any payload.
- **`Re-handoff` action** on a stalled handoff row prompts a
  re-engage flow.

### Must NOT do

- Don't make this a notification center. It's a graph view of
  cross-pack events, not "things to read."
- Don't render edges as decorative; the edge type chip carries
  real meaning (`qa-rejected` vs `raised-bug-from-failure`
  vs `requires-seam` is the platform's vocabulary).
- Don't omit within-pack promotions. The regression-promotion
  event is a stream-worthy moment.

### Preset variations

- **Cross-team filter:** filter by which team owns the receiving
  artifact (useful in multi-team workspaces).

---

## Mockup delivery checklist

For each of the eight screens, verify before declaring done:

- [ ] Single self-contained HTML file under `.hero/mocks/hero-qa/`.
- [ ] Inline `<style>` and `<script>`; no external dependencies.
- [ ] SVG icons inline; no icon fonts.
- [ ] Linear-class density (~32px rows, ~13px body type).
- [ ] Hero brand bolt logo and Hero blue palette in chrome (see
      `feedback_hero_brand_assets.md` in user memory — bolt SVG,
      not "H" letterform; light blue palette, not Linear indigo).
- [ ] Realistic sample data (no lorem ipsum); identities Jamie,
      Tomás, Yuki for QA roles; Sarah / Marcus / Diego / Aisha /
      Priya carried from PM mockups for cross-screen continuity.
- [ ] Canonical preset chips visible in top-right chrome.
- [ ] One brand interaction visible per screen (Reject composer,
      Coverage chip, Verdict classify, etc.) — not all hidden.
- [ ] Empty-state and no-integration paths shown where called out
      in the brief (Screen 7 especially).
- [ ] Cross-pack ambient cards visible on screens that have them
      (Screen 4 and Screen 5).
- [ ] Bottom strip rendered with state-appropriate verbs (Screens
      2, 3, 4, 5, 6). Singleton views (1, 7, 8) explicitly omit
      the bottom strip.
- [ ] Accessible color contrast — verify the QA-cyan gutter, the
      qa-ready blue chip, the flaky violet against the chrome.
- [ ] Index file (`index.html`) listing all eight mockups with
      one-line descriptions, mirroring PM's mockup index.

### Sequencing

If shipping in parts, ship in this order — earlier screens are
load-bearing for later screens' realism:

1. Screen 1 (Story queue) — establishes the queue + sample data
   identities.
2. Screen 4 (Story qa-ready rejection) — the brand interaction;
   the screen demos open on.
3. Screen 2 (Test plan editor) — the matrix that proves coverage
   is real.
4. Screen 3 (Test case editor) — the case-as-document fidelity.
5. Screen 7 (Unified inbox) — the unification thesis visible.
6. Screen 5 (Release-gate) — the destination view.
7. Screen 6 (Flaky backlog) — the opinionated zero-flake stance.
8. Screen 8 (Handoff stream) — the graph traversal payoff.

Build this sequence assuming each screen will be reviewed before
the next is started, so per-screen iteration time can drive the
priority order.
