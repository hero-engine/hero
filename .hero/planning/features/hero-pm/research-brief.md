# Hero PM — Competitive UX Research Brief

Audience: ui-designer agent and downstream design pass on `hero-pm`. The
strategic frame (layered methodology presets, five PM principles, six
killer screens, silo-tearing thesis) is taken as given — this brief
turns it into concrete reference material a designer can study without
having to wander the PM-tool landscape unguided.

For each tool: what to study, what to steal, what to leave behind,
which Hero screens it influences. Treat the "Screens to study" lines
as a shopping list — find reference images of those specific views
(product screenshots, vendor docs, YouTube walkthroughs are fine) and
absorb the layout grammar before mocking the corresponding Hero screen.

The six Hero screens this brief feeds:

1. Story detail (with `Hand off to /design`)
2. Roadmap board (Now/Next/Later default)
3. Story queue (Pivotal-influenced)
4. PRD editor (Notion-fidelity)
5. Intake funnel (Productboard-influenced)
6. Cross-domain handoff stream

---

## Pivotal Tracker

**Influences:** Story queue (primary); Story detail (secondary).

### What they actually do well

- **The single-list queue.** One vertical column — Icebox, Backlog,
  Current, Done — with stories flowing top-down as priorities are
  set. There is no per-status board; priority *is* the y-coordinate.
  Drag a story up; everything below shifts. The single-list mental
  model is the most underrated PM-UX idea ever shipped: it makes
  prioritization a *physical act* rather than a meeting outcome.
- **Auto-prioritizing current iteration.** Once you set a team
  velocity, Pivotal cuts a horizontal line in the queue at "this is
  what fits this week." The line moves as you change estimates. No
  separate sprint-planning meeting required.
- **Three-kind typing.** Feature (point-estimated), Bug
  (un-estimated), Chore (un-estimated). The point system is opinionated
  but escapable — bugs and chores don't get points because they're
  not capacity work. Engineering teams who hate estimating still
  benefit from the queue.
- **State machine inside a card.** Start → Finish → Deliver → Accept
  → Reject. The card has the workflow baked in; you don't drag it
  between columns, you click the button. Reduces noise.

### What we should steal verbatim

- **Single-list flow queue** → Hero Story queue. Icebox / Backlog /
  Current / Done as vertical bands within one scrolling list, not
  four kanban columns. Drag-reorder up and down.
- **Velocity cut line** → Hero Story queue under sprint preset. A
  horizontal "this fits this sprint" line that moves with estimates.
  Under kanban preset, hide the line; under cycle preset, replace
  with a "fits this cycle" marker.
- **Feature/Bug/Chore typing** → already in Hero's spec types but
  worth honoring inside the Story queue's visual language (a small
  type chip on each row).
- **In-card state advance buttons** → Hero Story detail. The state
  machine should advance via a single primary button, not a status
  dropdown buried in a sidebar.

### What to leave behind

- **Forced point estimation.** Pivotal makes points mandatory for
  features. We support estimation when sprint preset is on, hide
  estimation under kanban/cycle, and never make it a wall.
- **Auto-prioritize-everything dogma.** Pivotal's auto-iteration
  cutoff assumes velocity is accurate. For new teams, this lies.
  Hero shows the cut line as advisory, not gospel.
- **No long-form doc surface.** Pivotal stories are short cards.
  We need a PRD editor for the big artifacts.

### Screens to study

- The Pivotal Tracker project view (the main 3-column layout with
  Current/Backlog/Icebox stacked left). Focus on the density of the
  story rows, the inline state buttons (Start/Finish/Deliver), and
  the velocity cut line at the top of Current.
- A single story card expanded inline (story title, description,
  tasks, activity, comments). Focus on how the card grows in place
  rather than opening a modal.

---

## Linear

**Influences:** All six screens. Linear is the speed/density target
for the entire UI.

### What they actually do well

- **Command palette (Cmd-K).** Fuzzy-searches issues, projects,
  cycles, views, users, and *actions* (assign, prioritize, move,
  comment). Primary action is navigation; hold Shift to switch the
  default to assignment or status-change. Every action a user
  performs is reachable from the palette in under 3 keystrokes.
- **Keyboard-first issue navigation.** `J`/`K` to move row-to-row,
  `Enter` to open, `Esc` to return. Power users never touch the
  mouse. The animations are tuned to be slightly under the speed
  of human visual feedback — the UI feels faster than the network
  it's running on.
- **Cycles.** Time-boxed iterations (1–4 weeks, configurable) with
  carry-over rules: incomplete work either rolls to the next cycle
  or returns to backlog. Cleaner than scrum sprints because the
  ceremony is implicit (no separate "sprint planning" mode).
- **Triage view.** New issues land in a Triage queue, not the
  backlog. Untriaged work is visually distinct. Forces the team
  to actively accept or reject inbound.
- **Status pills with a tight vocabulary.** Backlog, Todo, In
  Progress, In Review, Done, Canceled, Duplicate. The list is
  short, the colors are tuned, the icons are 12px and unambiguous.
  Resist the urge to add new statuses; Linear's discipline is the
  feature.
- **Inline mentions and references.** `@user`, `@project`, `#issue`
  all autocomplete inside comments and descriptions, producing
  live-linked references. Linear treats the graph as text.

### What we should steal verbatim

- **Cmd-K command palette** → Hero PM global. Already a Hero
  primitive (`hero search`) — the dashboard's Cmd-K must surface
  PM specs, engineering specs, dashboard views, and actions
  (handoff, refine, triage) in one fuzzy list.
- **J/K navigation in Story queue** → Hero Story queue.
- **Cycle preset** → Hero methodology presets (cycle-based).
  Carry-over rules: incomplete → next cycle by default,
  user-overridable.
- **Triage as the inbound landing zone** → Hero Intake funnel.
  Triaged vs untriaged is the most important distinction; the
  Intake screen should feel like Linear's Triage view, not a
  generic table.
- **Inline @-mentions and #-references in PRDs and stories** →
  Hero PRD editor and Story detail. Should produce live graph
  edges, not just text.
- **Density.** ~32px row height for the Story queue. ~12px body
  text. Don't pad like enterprise tools.

### What to leave behind

- **Rigidity of cycles.** Linear strongly assumes cycles. Teams on
  pure kanban feel friction. We support cycles as one preset, not
  the default.
- **Few escape hatches.** Linear's opinions are excellent for
  their target user (modern startups). For Shape Up teams or
  phased-delivery teams, the model breaks. Hero presets are the
  escape hatch.
- **No long-form docs.** Linear's project documents got better
  recently but are still note-shaped, not PRD-shaped. We need
  Notion-class doc fidelity for PRDs.
- **Customer feedback handling.** Linear has a feedback inbox now
  but it's bolted on. Productboard handles this better.

### Screens to study

- The Linear project view with the cycle picker at the top, status
  groupings (Backlog / Todo / In Progress / In Review / Done), and
  the issue rows. Focus on the row density and the leading icons.
- Cmd-K palette opened over the project view. Focus on the result
  groupings (Issues, Projects, Cycles, Members, Views, Actions)
  and the action-default behavior.
- The Triage view. Focus on the visual differentiation from the
  main project view — usually a yellow accent color and a "needs
  triage" badge.
- A single issue detail page. Focus on the right rail (Status,
  Priority, Assignee, Project, Cycle, Labels) and the primary
  comment thread layout.

---

## Jira

**Influences:** none directly (we are running *away* from Jira's
UX) — but Workflow Flexibility informs how methodology presets are
configured.

### What they actually do well

- **Workflow flexibility.** Custom states, transitions, and gates
  per project. Whatever your methodology, Jira can model it. This
  is the single capability that has kept Jira alive despite the
  UI being what it is.
- **JQL.** Querying issues with a real query language is genuinely
  powerful for ops-heavy PMs and program managers.
- **Roles + permissions per project.** Enterprise-grade; not a
  v1 concern for Hero but worth knowing the shape exists.

### What we should steal verbatim

- **The principle behind workflow flexibility, not the UX.**
  Methodology presets are Hero's answer: ship 4–5 pre-configured
  workflow shapes, let teams pick. Don't ship a workflow editor.
  (That is exactly the trap Jira fell into.)
- **JQL-shaped power-user query** as a future affordance, surfaced
  through `hero search` rather than a separate query builder.

### What to leave behind

- **Configuration depth as a feature.** Jira's flexibility is its
  competitive moat *and* its UX cancer. Hero takes the flexibility
  and hides the configuration — presets, not a settings tree.
- **Slow page loads.** Hero's density and speed target is Linear,
  not Jira.
- **Custom-field-itis.** Jira projects accumulate dozens of fields
  that are mostly empty. Hero artifacts have a tight schema; no
  per-team field editor in v1.
- **Boards and backlogs as different surfaces.** Jira separates
  "board" from "backlog" and they get out of sync. Hero's Story
  queue collapses these.

### Screens to study

- A configured Jira board with swim lanes and a backlog drawer.
  Look at it not to copy, but to be reminded what we are
  consciously not building.
- The workflow editor (Settings → Workflows). Same — look at it
  to understand the trap we are sidestepping.

---

## Shape Up / Basecamp

**Influences:** Roadmap board (betting table mode); PRD editor
(pitch shape); Story detail (hill-chart status concept).

### What they actually do well

- **The Pitch document.** A structured long-form doc with five
  required sections: Problem, Appetite, Solution, Rabbit Holes,
  No-Gos. Sized so the team can read it in one sitting and decide
  yes/no. The shape *enforces* PM discipline: appetite must be
  explicit, no-gos must be named.
- **The Betting Table.** Once every 6 weeks, leadership sits with
  the list of pitches and "bets" on which cycle to do which.
  This is roadmap prioritization made explicit and small. Not a
  giant quarterly planning offsite — a 90-minute decision meeting
  with a fixed input list.
- **Hill chart.** Every scope of work plotted on an S-curve:
  uphill (figuring out), top (apex), downhill (executing). Visual
  representation of *unknowns remaining*, not progress percentage.
  This is the best status-of-risky-work UX ever shipped.
- **No estimates.** Appetite (the budget — 2 weeks vs 6 weeks)
  replaces estimates. The scope flexes inside the appetite.
- **Cooldown weeks.** Two-week breathing room after each 6-week
  cycle. Explicitly not roadmap work. Eliminates the "what do
  engineers work on between projects" question.

### What we should steal verbatim

- **Pitch-shaped PRD template** → Hero PRD editor. Default PRD
  template should have Problem / Appetite / Solution / Rabbit
  Holes / No-Gos as named sections (alongside Hero's existing
  EARS-shaped Acceptance Criteria). Teams can edit the template
  per-project.
- **Betting table view** → Hero Roadmap board (cycle preset).
  When cycle preset is active, Roadmap shows a betting-table
  layout: candidate pitches on the left, the upcoming cycle's
  bets on the right, and a "decided / deferred" zone in between.
- **Hill chart on Story detail** → Hero Story detail (cycle and
  sprint preset). For in-flight stories, an inline hill chart
  showing "still figuring out" vs "executing." This is the
  cleanest answer to the "what's the actual risk left" question.
- **Cooldown as a first-class state** in the cycle preset.

### What to leave behind

- **Six-week mandate.** Shape Up assumes 6 weeks; we make the
  cycle length configurable.
- **No-estimates dogma.** Some teams want story points. The
  sprint preset honors that; the cycle preset doesn't.
- **No backlog.** Shape Up explicitly rejects a backlog ("pitches
  expire"). For most teams this is unrealistic — a "later" or
  "icebox" zone is non-negotiable. Hero keeps it.

### Screens to study

- The Basecamp Betting Table screenshot (public, from the Shape
  Up book) — focus on the visual hierarchy of "pitches under
  consideration" vs "decisions made."
- A pitch document in Basecamp Docs — focus on how the five
  required sections render and how cross-links inside a pitch
  surface.
- A hill chart with multiple scopes plotted — focus on the dot
  shape, the labels, and how "stuck" is represented (a stalled
  dot at the apex).

---

## Productboard

**Influences:** Intake funnel (primary); Roadmap board (prioritization
framework concept).

### What they actually do well

- **Feedback inbox with source attribution.** Every piece of
  inbound (Intercom message, customer email, sales note, support
  ticket) lands in a structured inbox with the source preserved.
  PMs can highlight a sentence and link it to a feature, building
  a real evidence trail.
- **Insights → Features → Releases pipeline.** A single inbound
  comment can be linked to multiple feature candidates; a feature
  shows the *count and weight* of supporting feedback. Roadmap
  prioritization is *defensible* because the data is right there.
- **Built-in prioritization frameworks.** RICE, value-vs-effort
  scatter, weighted-shortest-job-first. Each is a configurable
  view that scores features against capacity. Optional, not
  forced.
- **Customer segment tagging.** Inbound is tagged by customer
  segment (Enterprise, SMB, etc.) so prioritization can be
  segment-weighted.

### What we should steal verbatim

- **Inbound inbox with source preserved** → Hero Intake funnel.
  Every intake-item shows the source (customer, sales, support,
  internal) and the verbatim quote where it came from.
- **Highlight → link interaction** → Hero Intake funnel. Select
  text in an intake item, action: "Link to roadmap-item" or
  "Promote to roadmap-item." This is the killer interaction.
- **Evidence counts on roadmap items** → Hero Roadmap board. Each
  roadmap-item shows the count of linked intake-items and a
  rolled-up "weight" (segment, recency, repeat customers).
- **Prioritization framework as a view, not a mode** → Hero
  Roadmap board. A view toggle: Now/Next/Later (default),
  Value-vs-Effort scatter, RICE table. Same underlying data,
  three lenses.

### What to leave behind

- **Enterprise pricing / gated features.** Not a UX concern but
  worth noting Productboard's roadmap export and stakeholder
  views are paywalled. Hero ships them inline.
- **Heavy "Insights" terminology.** Productboard's vocabulary is
  branded; Hero uses plain words (intake-item, evidence, link).
- **Separation from delivery.** Productboard stops at the
  roadmap; engineering work is in another tool. Hero's whole
  point is that the boundary doesn't exist.

### Screens to study

- The Productboard Insights inbox — focus on the source-tagged
  inbound rows and the highlight-to-link interaction.
- A feature detail with linked Insights aggregated below — focus
  on how the evidence count and per-segment breakdown surfaces.
- A prioritization view (e.g. RICE table or value/effort
  scatter) — focus on the toggle UX between roadmap layouts.

---

## Aha!

**Influences:** PRD editor (goal/initiative/feature hierarchy);
Roadmap board (strategic-context overlay).

### What they actually do well

- **Hierarchy clarity.** Goal → Initiative → Feature → Release →
  Requirement. Each level has a purpose and a different rendering.
  When done well, you can navigate from "company goal" down to
  "specific story" in three clicks and never lose context.
- **Goal-to-feature traceability.** A feature page shows which
  initiative it serves, which goal that initiative ladders to,
  and the strategic context for why this work matters. Aha's best
  feature is making strategy *contiguous* with delivery.
- **Roadmap presentation views.** Polished, stakeholder-ready
  exports (PDF, slide-shaped) of the roadmap. Built into the
  product, not a bolt-on.

### What we should steal verbatim

- **Goal context strip on every artifact** → Hero PRD editor and
  Story detail. A breadcrumb-shaped strip at the top: roadmap-item
  → epic → story. Click any link to navigate up. (Hero already has
  the relations model; this is just surfacing it consistently.)
- **Stakeholder-ready roadmap view** → Hero Roadmap board, as an
  alternate render. A "presentation mode" toggle that hides team
  chrome and shows a cleaner Now/Next/Later or quarterly grid.

### What to leave behind

- **Five-level hierarchy.** Aha's depth (Goal → Initiative →
  Feature → Release → Requirement) is too much. Hero's three
  levels (roadmap-item → epic → story) plus PRD as a horizontal
  doc is enough.
- **Enterprise feel.** Aha looks like enterprise software because
  it is. We are running toward Linear-density, not Aha-bloat.
- **Roadmap-first siloing.** Aha is mostly a roadmap-only tool;
  engineering happens elsewhere. Same trap as Productboard.

### Screens to study

- An Aha! feature detail page — focus on the strategic-context
  strip at the top and how the hierarchy is exposed.
- A roadmap presentation view (the "Now/Next/Later" or "Quarter"
  variant) — focus on what's hidden in presentation mode vs the
  team view.

---

## ProductPlan / Roadmunk

**Influences:** Roadmap board (drag UX, horizon presentation).

### What they actually do well

- **Drag-to-reorder roadmap bars.** Roadmap items are bars on a
  timeline or rows in a Now/Next/Later grid. Drag to reorder,
  drag to span a different horizon, drag to a different lane.
  The whole interaction model is direct manipulation.
- **Lane configuration.** Lanes by team, theme, product area,
  customer segment, or strategic goal. Configurable per view,
  shareable as a URL.
- **Presentation views.** Stakeholder-ready URL-shareable views
  that hide internal team chrome and expose only the
  decision-relevant slice.

### What we should steal verbatim

- **Drag-to-reorder roadmap-items in horizon view** → Hero
  Roadmap board (Now/Next/Later default).
- **Lane configuration** → Hero Roadmap board. Group by theme /
  owner / customer segment / quarter. Saved as a view.
- **Shareable URL view** → Hero Roadmap board. The exact filter
  + grouping is in the URL so a PM can paste a link to a
  stakeholder and they see what the PM is seeing.

### What to leave behind

- **Roadmap-as-only-thing.** Same siloing trap. ProductPlan
  doesn't know about delivery; Hero does.
- **Glossy presentation veneer.** Don't over-design the
  presentation view; Hero density wins.

### Screens to study

- A ProductPlan or Roadmunk Now/Next/Later board with multiple
  swim lanes — focus on the drag interaction and the lane
  configuration affordance.

---

## Notion / Coda

**Influences:** PRD editor (primary).

### What they actually do well

- **Block-based document editing.** Every paragraph, heading,
  table, embed, and toggle is a block. Drag, rearrange, nest.
  The document feels like a malleable surface.
- **Slash commands for block insertion.** `/heading`, `/table`,
  `/embed`, `/toggle`. Fast block insertion without leaving the
  keyboard.
- **Linked-database embeds.** A document can embed a filtered
  view of a structured database — table, board, calendar, gallery.
  Same underlying data, multiple views.
- **`@`-references and backlinks.** Mention any page anywhere;
  backlinks roll up automatically. Cross-references become a
  navigation surface.

### What we should steal verbatim

- **Block-based editing + slash commands** → Hero PRD editor.
  PRDs are long-form docs; the editing surface must feel like
  Notion, not like a textarea.
- **`@`-references with autocomplete** → Hero PRD editor (and
  Story detail comments). Typing `@story-` should fuzzy-find
  Hero stories and produce a live-linked reference. Same for
  `@feature-`, `@epic-`, `@convention-`.
- **Embedded view of stories inside a PRD** → Hero PRD editor.
  A PRD can embed "stories belonging to this PRD" as an
  inline filtered list. Same data as Story queue; different
  context.
- **Toggle blocks** for risks, FAQs, decision-log items inside
  a PRD.

### What to leave behind

- **Free-form chaos.** Notion is too flexible; teams end up with
  inconsistent doc shapes. Hero PRDs have an opinion (Problem /
  Solution / Scope / Metrics / Risks, pitch-shaped — see Shape
  Up). The flexibility is in *how* you fill the sections, not
  in *which* sections exist.
- **Wikis as a thing.** PRDs are docs, not a wiki. No
  free-floating pages, no parent-child page tree without a
  spec type.
- **Performance.** Notion is slow. Hero's editor should feel
  Linear-fast even on long PRDs.

### Screens to study

- A Notion document with mixed content (headings, an inline
  table, a toggle list, a database view embed) — focus on the
  block boundary affordances (the drag handle on hover, the
  `+` to add a block).
- The slash-command menu open mid-paragraph — focus on result
  grouping and keyboard navigation.

---

## Trello

**Influences:** Kanban preset variant of Story queue (light influence).

### What they actually do well

- **The lightest possible kanban.** Columns, cards, drag. Nothing
  else. For small teams or simple workflows, this is enough.
- **The empty-state shape.** A new Trello board has 3 columns
  (To Do / Doing / Done) and the user can start immediately.

### What we should steal verbatim

- **Empty-state defaults** for the kanban preset of Story queue:
  Icebox / Backlog / Current / Done preconfigured, no setup wizard.

### What to leave behind

- **No hierarchy.** Trello has cards and columns; that's it. No
  epics, no PRDs, no roadmap. Hero has all of these as first-class
  spec types.
- **Power-ups model.** Adding any structure to Trello requires
  a plugin marketplace. Hero ships the structure.

### Screens to study

- A simple Trello board (3 columns, ~5 cards each) — confirm what
  *not* to build. The kanban preset of Hero's Story queue should
  feel as approachable as Trello while still being Pivotal-shaped
  underneath.

---

## Kanbanize / Businessmap

**Influences:** Story queue (flow analytics overlay); Cross-domain
handoff stream (cycle-time charts).

### What they actually do well

- **Flow analytics built in.** Cycle time, lead time, throughput,
  WIP aging, cumulative flow diagrams. Every kanban board has a
  "metrics" view alongside the board view, same data, different
  lens.
- **WIP aging visualization.** Cards that have sat in a column
  for too long visibly age — typically a yellow → red gradient
  on the card border. The board *tells you* what's stuck.
- **Cycle time + lead time histograms.** Per-card-type
  distributions, not just averages. Helps a team see "95% of
  stories complete in under 10 days" which is more useful than
  "average is 5 days."

### What we should steal verbatim

- **WIP aging on Story queue rows** → Hero Story queue. Stories
  that have been in `in-flight` for longer than the team's median
  cycle time get a subtle visual treatment.
- **Cycle-time histogram on Cross-domain handoff stream** → Hero
  Handoff stream. A small chart showing the distribution of
  "time from handoff to ship" across recent stories. Surfaces
  whether handoffs are getting stuck.
- **Cumulative flow diagram** as an optional alternate view of
  Story queue under kanban preset — defer to v2.

### What to leave behind

- **Industrial UI feel.** Kanbanize looks like SAP, not Linear.
  Adopt the analytics, not the visual language.
- **WIP limit enforcement walls.** Kanbanize hard-blocks moving
  cards past WIP limits. Hero shows a warning, doesn't block.

### Screens to study

- A Kanbanize board with cumulative flow diagram below — focus
  on the diagram's color bands and how column transitions read.
- A WIP aging view — focus on the visual treatment of stale
  cards (border color, age badge).

---

## Height

**Influences:** Story queue, Intake funnel, PRD editor (AI patterns).

### What they actually do well

- **Ambient AI.** A "Copilot" panel offers context-aware actions:
  triage this issue, suggest assignee, draft acceptance criteria.
  Available, not mandatory. The model doesn't shove itself into
  the primary flow.
- **AI-suggested duplicate detection** during issue creation:
  "this looks like #341, merge?" The model intercepts at write
  time, not after-the-fact.
- **AI-suggested refinement** on stories with thin descriptions.
  A subtle "fill in details" affordance, not a forced prompt.

### What we should steal verbatim

- **Ambient AI panel** → Hero PM globally. A right-rail panel
  (collapsible) on Story detail, Intake funnel, and PRD editor
  with context-specific actions:
  - On Story detail: "Hand off to /design," "Refine via /refine,"
    "Find similar stories."
  - On Intake funnel: "Triage this," "Find duplicates," "Suggest
    roadmap-item to link."
  - On PRD editor: "Suggest acceptance criteria," "Find related
    decisions."
- **Duplicate detection at write time** → Hero Intake funnel.
  When pasting/creating an intake-item, surface near-duplicates
  immediately.

### What to leave behind

- **AI as a wrapper around mediocre PM.** Height's underlying
  PM model is thin; the AI is doing a lot of work to make up
  for it. Hero's underlying model is strong (specs, graph,
  presets) — AI is a force multiplier, not a crutch.
- **Pop-up AI prompts.** Never surprise the user with a modal
  AI suggestion. The panel is the only surface.

### Screens to study

- The Height workspace with the Copilot panel open on the right —
  focus on the action list and how context-awareness reads
  (the actions change based on what's selected).

---

## Consolidated design philosophy

Distilled from the research, the Hero PM UX worldview is:

- **Flow over ceremony.** A PM's primary action is moving work
  through the queue, not running meetings about the queue. The
  UI rewards continuous prioritization (drag, J/K, Cmd-K) and
  hides the ceremony that traditional tools wrap around the same
  underlying actions. Pivotal's queue, Linear's keyboard, Shape
  Up's appetite all instances of the same idea.
- **Layered presets, not modes.** Methodology is a thin overlay
  on top of universal artifacts. Switching presets changes what
  the dashboard *shows* (cycles vs sprints vs kanban) without
  changing the underlying spec types. Teams that hybridize
  (almost everyone) compose their own setup.
- **Long-form docs are first-class.** A PRD is not a "description
  field on a parent ticket." It's a Notion-grade document with
  embedded story lists, live cross-references, and a structured
  template (pitch-shaped by default). PM thinking happens in
  long-form; the tool must honor that.
- **Evidence makes prioritization defensible.** Roadmap items
  show the inbound signal that produced them. Decisions are
  re-readable. `hero why` works on roadmap-items as well as
  features.
- **The boundary with engineering is a hyperlink, not a wall.**
  The single highest-leverage interaction in the product is
  "Hand off this story to /design." From that click, a feature
  spec exists, an edge in the graph exists, and the Handoff
  stream reflects it within the same session. No copy-paste,
  no Jira-export, no cross-tool reconciliation.
- **Density is respect.** Linear's row density (~32px) and
  body type (~12px) is the target. Enterprise spacing wastes
  the user's screen and signals that the tool is for
  stakeholders, not operators.
- **AI is ambient, never modal.** A right-rail panel offers
  context actions. The model doesn't interrupt; the user
  invites it. Hero's existing agent skills (`/refine`,
  `/triage`, `/design`, `/diagnose`) are the action surface.
- **No new vocabulary the user wouldn't say aloud.** "Story,"
  "epic," "PRD," "roadmap." Not "insight," "objective key
  result group," "release cycle." If a PM doesn't already say
  it, Hero doesn't introduce it.

---

## Anti-pattern list

Things to explicitly *not* do, with the tool that warns us off
each:

1. **Forced point estimation in all presets.** (Pivotal) Make
   estimation opt-in via the sprint preset; never block work
   without a number.
2. **Mandatory cycle/sprint cadence.** (Linear, Shape Up) The
   "no preset" team — pure kanban / continuous flow — must be
   first-class, not a degraded mode.
3. **Workflow editor as a feature.** (Jira) Ship presets, not a
   workflow-builder UI. Configurability through presets, not
   through a state-machine designer.
4. **Custom-field accumulation.** (Jira) Fields are part of the
   spec-type schema (registry-defined). No per-team field
   editor in v1.
5. **Boards and backlogs as separate surfaces that drift apart.**
   (Jira) Story queue is one surface; methodology preset
   changes what's emphasized, not what's there.
6. **Roadmap as a silo with no link to delivery.** (Productboard,
   Aha, ProductPlan) Every roadmap-item shows live delivery
   state for child stories. The Handoff stream is the silo-tear
   made visible.
7. **Modal AI prompts and forced AI features.** (Height in its
   worst moments) AI lives in the right rail; the user invites
   it.
8. **Five-level hierarchy.** (Aha) Three levels (roadmap-item →
   epic → story) plus PRD as a horizontal doc. Stop.
9. **Free-form doc chaos.** (Notion) PRDs have a default template
   (pitch-shaped) and lint against structural drift. Flexibility
   is in content, not in section names.
10. **Enterprise visual density.** (Jira, Aha, Kanbanize) ~32px
    rows, ~12px body. Linear-class, not stakeholder-class.

---

## Screen-influence map

| Hero screen | Primary influences | Secondary influences |
|---|---|---|
| Story detail (Hand off button) | Linear (right rail, status pills), Notion (description editing) | Height (ambient AI panel), Shape Up (hill chart) |
| Roadmap board (Now/Next/Later) | ProductPlan/Roadmunk (drag, lanes, shareable URL), Shape Up (betting table for cycle preset) | Aha (presentation view, hierarchy strip), Productboard (evidence counts) |
| Story queue | Pivotal Tracker (single-list flow, in-card state), Linear (J/K, density, command palette) | Kanbanize (WIP aging), Trello (kanban-preset empty state) |
| PRD editor | Notion (block editing, slash commands, `@`-refs), Shape Up (pitch sections) | Linear (inline mentions producing graph edges), Aha (goal context strip) |
| Intake funnel | Productboard (source-tagged inbox, highlight-to-link), Linear (Triage view shape) | Height (duplicate detection on create) |
| Cross-domain handoff stream | Linear (cycle view timeline), Kanbanize (cycle-time histogram overlay) | Aha (strategic-context strip), Pivotal (deliver state) |
