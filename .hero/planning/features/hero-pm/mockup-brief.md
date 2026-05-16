# Hero PM — Mockup Brief for ui-designer

Audience: `ui-designer` agent (and any human designer reviewing the
mockups). This brief is a self-contained set of instructions for
producing **six killer-screen HTML mockups** for the Hero PM domain
pack. Each screen brief stands alone — a fresh ui-designer session
can act on any one without re-reading the others, though the preamble
applies to all six.

Read `research-brief.md` (sibling file) first if you want the deep
competitive analysis behind these screens. This file gives you the
*what to build*; the research brief gives you the *why these patterns*.

---

## Preamble — shared design grammar

### Headline UX target

> *Pivotal Tracker's flow-first philosophy + Linear's speed + Shape
> Up's pitches + Notion's doc fidelity — all consumable by engineering
> without leaving engineering's tools.*

### Layout grammar

Every Hero PM screen uses the same shell:

```
┌─────────────┬───────────────────────────────────────┬─────────────┐
│ Left nav    │ Content pane                          │ Right rail  │
│ (~220px)    │ (flex: 1)                             │ (~320px,    │
│             │                                       │  optional,  │
│             │                                       │  collapses) │
└─────────────┴───────────────────────────────────────┴─────────────┘
```

- **Left nav (~220px):** sticky, always visible. Sections:
  - Top: workspace name + domain pill (`PM`) with a dropdown to
    switch to `Engineering`.
  - Six PM view links: Roadmap, Story queue, PRDs, Intake, Handoff
    stream. (Story detail and PRD editor are nested under their list
    views; not top-level entries.)
  - "Active sessions" mini-list near the bottom (shared chrome).
  - Domain switcher / settings at the very bottom.
- **Content pane:** the screen's primary surface. Variable density;
  Linear-class.
- **Right rail (~320px, collapsible):** context for selected item.
  Holds the ambient-AI panel, related items, activity, metadata.
  Collapses to a 32px gutter when not needed.

### Top chrome

A 48px header above the three-column shell:

- Left: breadcrumbs (Workspace › View › Item if applicable).
- Center: global search affordance (placeholder "Search or jump to…
  ⌘K"). Cmd-K triggers the command palette.
- Right: preset indicator chip (e.g. "Cycle · 6w" or "Sprint · 2w"
  or "Flow"), notification bell, user avatar.

### Color / density target

```css
:root {
    --bg: #ffffff;
    --bg-secondary: #f7f8fa;        /* sidebar */
    --bg-tertiary: #eef0f4;
    --bg-hover: #f0f2f6;
    --text: #1a1d23;                 /* near-black, not pure */
    --text-secondary: #5d646e;
    --text-muted: #8a929c;
    --border: #e4e7eb;
    --border-strong: #d0d4da;
    --primary: #5e6ad2;              /* Linear-ish indigo */
    --primary-hover: #4a55c4;
    --success: #16a249;
    --warning: #d97706;
    --danger: #dc2626;
    --info: #0284c7;
    --accent-handoff: #7c3aed;       /* cross-domain handoff edge color */
}

body {
    font-size: 13px;                 /* Linear-class */
    line-height: 1.45;
    font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
}
```

- Row height: **32px** for list rows (Story queue, Intake funnel,
  Handoff stream). 40px for board cards (Roadmap).
- Section padding: 16–24px. Don't go above 32px between sections —
  the screen should feel dense, not spacious.
- Border radius: 6px on most affordances, 8–12px on cards.
- Shadows: subtle. Linear uses almost none; follow suit.

### Methodology preset

Every screen must accommodate the **active preset** specified via a
chip in the top-right of the chrome. Mockups should ship with one
canonical preset shown clearly in the chrome, but the brief calls out
preset variations per screen. Default canonical preset for mockup
delivery: **Horizon roadmap + Cycle delivery (6w)** — it exercises
the most preset-driven UI.

### Realistic sample copy

Use Hero-flavored, plausible PM content. The product is a B2B AI
agent platform; sample roadmap items, stories, and PRDs should
reflect that. **No lorem ipsum, no "Foo / Bar / Baz."** Suggested
sample identities for assignees: Sarah Chen (Engineering Lead),
Marcus Johnson (PM), Aisha Patel (Design), Diego Ramirez (Engineering),
Priya Shah (PM).

### File output convention

Each screen ships as one self-contained HTML file under
`.hero/mocks/hero-pm/`. Filenames:

- `01-story-detail.html`
- `02-roadmap-board.html`
- `03-story-queue.html`
- `04-prd-editor.html`
- `05-intake-funnel.html`
- `06-handoff-stream.html`

Follow the `html-mockup-generation` skill conventions: single file,
inline `<style>` and `<script>`, no CDNs, no external assets. SVG
inline for any icons. Realistic data; no lorem.

---

## Screen 1 — Story detail with "Hand off to /design"

> **The platform thesis. Most important screen.** Earns principles
> #2 (define), #4 (align). Influences: Linear (right rail) + Notion
> (description) + Shape Up (hill chart) + Height (ambient AI panel).

### Purpose

Show one PM story in full fidelity, with the cross-domain edge to
engineering made *visible* and the handoff interaction made
prominent.

### Layout

- **Left nav:** shared shell. "Story queue" entry highlighted.
- **Content pane:**
  - Breadcrumb / context strip at the top:
    `Roadmap-item: AI agent observability ›  Epic: Telemetry & traces ›  Story: Story-2847`
    (each segment is a link).
  - Story title as h1: `As an SRE I need agent runs to emit
    structured trace spans so I can correlate them with downstream
    service latency.`
  - A row of metadata pills below the title: type `story`, status
    `ready`, priority `high`, points `5` (sprint preset) OR
    appetite-chip `2w` (cycle preset), assignee avatar + name.
  - **Primary action bar** (sticky just below the title):
    - `Hand off to /design` — large solid button, indigo, with an
      arrow icon. Tooltip: "Create an engineering feature spec
      from this story."
    - `Refine` (secondary outline button — runs `/refine` skill).
    - `…` overflow menu (archive, duplicate, link to PRD…).
  - **Description block** — Notion-shaped editor (read state with
    headings, paragraphs, a checklist). Show 2–3 paragraphs of
    plausible content describing the user need and constraints.
  - **Acceptance Criteria** section — bullet list in EARS format
    where they fit. Example:
    - `WHEN an agent run starts THE SYSTEM SHALL emit a trace
      span with run-id and parent-session-id.`
    - `WHILE a run is active THE SYSTEM SHALL emit child spans
      for each tool call.`
    - `IF a span fails to flush THEN THE SYSTEM SHALL retry with
      backoff before dropping.`
    - Each bullet has a hover-state checkbox to mark accepted.
  - **Hill chart** (cycle preset only) — a small 240×100 SVG S-curve
    with two dots labeled "Trace schema (downhill, executing)" and
    "Storage backend choice (top of hill, deciding)." If the
    canonical preset is cycle-based, show this. Otherwise omit.
  - **Linked engineering feature** rail (this is THE killer
    surface):
    - Heading: "Engineering — handoff to `/design`"
    - One card showing the linked engineering feature spec:
      `feature: agent-trace-spans` with status `delivering`,
      assignee `Diego Ramirez`, "3 of 7 acceptance criteria
      complete," last commit `2h ago: wire span emitter into
      runner`, and a small `→ View in engineering` link.
    - Use the `--accent-handoff` color (purple) on the card
      border to visually mark this as a cross-domain link.
    - Empty state (when not yet handed off): a dashed-border card
      saying "Not yet handed off. Click *Hand off to /design* to
      create the engineering spec."
- **Right rail:**
  - **Ambient AI panel** (collapsible).
    - Actions: "Suggest acceptance criteria," "Find similar
      stories (3)," "Draft handoff context."
    - Recent suggestion: "This looks like Story-2691 (resolved
      4mo ago) — want to compare?"
  - **Activity stream** below the AI panel:
    - "Aisha Patel changed status to *ready* — 2h ago"
    - "Marcus Johnson refined acceptance criteria — yesterday"
    - "Sarah Chen linked from PRD: agent-observability — 3d ago"
  - **Sidebar metadata** (collapsible toggle): tracker_id, created,
    parent epic, parent roadmap-item, attached PRDs.

### Interactions (be explicit)

- **`Hand off to /design`** (primary) — clicking it should
  visually animate: button morphs into a progress chip ("Creating
  feature spec…"), then the "Linked engineering feature" rail
  transitions from empty state to populated. Mockup should show
  the *populated* state by default and link an inert "what
  clicking would do" tooltip on the button.
- **Description editor** — clicking a paragraph reveals Notion-style
  drag-handle on the left margin. (Show the handle in a hover
  state on one paragraph.)
- **Cmd-K** triggers global palette (no need to implement —
  reference the chrome's search placeholder).
- **Linked engineering feature card** is clickable; hover shows the
  feature spec's tooltip preview (mockup can show this hover state
  on one element).
- **EARS acceptance criteria checkboxes** are hover-revealable —
  the row is plain text until hover, then a checkbox slides in
  from the left.

### Must NOT do

- Don't render the handoff button as a tiny secondary action — it
  must be the most prominent button on the screen.
- Don't bury the linked engineering feature in a sidebar or modal
  — it lives in the content pane.
- Don't show progress percentages on the hill chart. Hill position
  is about *unknowns remaining*, not progress.
- Don't use a status dropdown buried in the right rail. Status is
  a pill near the title with an inline-edit affordance.
- Don't add a "Comments" tab as a separate surface. Comments are
  part of the activity stream in the right rail.

### Preset variations

- **Sprint preset:** show `points: 5` pill instead of `appetite`,
  show "fits sprint 14" indicator, hide hill chart, replace with
  a "Sprint progress" mini bar in the right rail.
- **Kanban preset:** hide both points and hill chart. Show a WIP
  age indicator ("4 days in `in-flight`") near the status pill.
- **Phased preset:** add a `Release: 2026-Q3` pill near status.
  Linked engineering feature card includes the release context.

---

## Screen 2 — Roadmap board (Now/Next/Later default)

> **Default landing page.** Earns principles #1 (decide), #3
> (trade-offs), #4 (align). Influences: ProductPlan/Roadmunk
> (drag UX) + Shape Up (betting table) + Productboard (evidence
> counts) + Aha (presentation overlay).

### Purpose

Land surface for the PM. Shows the active roadmap with cross-domain
delivery state baked in.

### Layout

- **Left nav:** shared shell. "Roadmap" highlighted.
- **Top chrome additions:** view-toggle segmented control on the
  left of the content pane:
  - `Now/Next/Later` (active)
  - `Quarters`
  - `RICE` (table)
  - `Value vs Effort` (scatter)
  - `Betting table` (visible only under cycle preset)
- **Content pane (Now/Next/Later default):**
  - Three columns side-by-side: **Now**, **Next**, **Later**.
    Each column is full-height, scrollable, with a header row
    showing column name + roadmap-item count + total appetite
    rollup.
  - Each roadmap-item is a 40px card. Card contents:
    - Title (one line, truncated).
    - Theme chip (e.g. `Observability`, `Agents`, `Onboarding`)
      with a color dot.
    - Appetite chip (cycle preset) or quarter (horizon preset).
    - Evidence count badge: `🔗 23` (number of linked
      intake-items) with a small bar showing customer-segment
      breakdown.
    - Delivery rollup pill: `2/5 stories delivering`. Pulled
      live across the domain boundary.
    - Assignee avatar.
  - Cards are drag-reorderable within and between columns.
    Indicate this with a small drag-handle that appears on
    hover.
  - A "Deferred" zone collapsed at the bottom — show 3 items
    in it, each with a reason chip (`deprioritized: Q4 capacity`,
    `merged into Now-3`, `rejected: not aligned with strategy`).
- **Right rail:** when a card is selected, show:
  - Roadmap-item title, full description, theme, owner.
  - Linked intake-items count + top 3 quotes from feedback
    ("Enterprise customers keep asking for…" with source pill).
  - Linked epics list with delivery state.
  - Action buttons: `Promote to active`, `Open detail`, `Add
    intake link`.

### Sample data

Theme: Agent Observability. Now items: "Agent trace spans"
(`2/5 stories delivering`, 23 linked feedback items, 2w
appetite), "Tool-call audit log" (`5/8 stories ready`, 12
linked feedback, 6w appetite). Next items: "Cross-session
memory dedupe," "Run replay UI." Later items: "Cost attribution
per agent," "Multi-tenant rate limits."

### Interactions

- Drag a card up/down within column → reorders priority.
- Drag a card from Next to Now → marks it `committed`; cursor
  changes to indicate the state transition.
- Click a card → right rail populates.
- Click a theme chip → filters the whole board to that theme.
- Lane configuration: a "Group by" dropdown above the columns
  lets the user re-lane by theme / owner / customer segment.
- A "Share view" button copies a URL that encodes the active
  view + filters + grouping. (Mockup shows the button; tooltip
  explains.)

### Must NOT do

- No timeline / Gantt view in v1.
- No date-bound bars stretched across a calendar.
- Don't make the "Deferred" zone hidden — trade-offs must be
  visible.
- Don't crowd cards with metadata. The five chips above are the
  ceiling.

### Preset variations

- **Cycle preset:** add a "Betting table" view-toggle option. In
  that view, the layout splits into "Candidate pitches" (left),
  "Bets for next cycle" (right), and "Recently bet" (bottom strip).
  Drag-to-bet interaction.
- **Sprint preset:** add a "Quarters" or "Months" alternate view
  but keep Now/Next/Later default.
- **Phased preset:** group by `target_release` instead of horizon
  by default.
- **Kanban preset:** identical to default (kanban is a delivery
  preset, not roadmap — horizon still applies).

---

## Screen 3 — Story queue (Pivotal-influenced)

> Earns principles #1 (decide), #2 (define). Influences: Pivotal
> Tracker (single-list flow) + Linear (keyboard, density) +
> Kanbanize (WIP aging).

### Purpose

The PM's working surface for prioritizing and refining stories.
Keyboard-fast, drag-to-reorder, single list with bands.

### Layout

- **Left nav:** shared shell. "Story queue" highlighted.
- **Top chrome additions:** filter chips above the queue ("Epic:
  all," "Owner: all," "Status: not done," "Type: any") + a
  velocity/capacity indicator (sprint preset) or cycle-fit
  indicator (cycle preset).
- **Content pane:**
  - Single vertical scrolling list, ~32px per row, with four
    horizontal band headers separating the list into:
    - **Icebox** (collapsed by default; click to expand) — N items.
    - **Backlog** — N items.
    - **Current** (sprint or cycle's active scope) — N items.
    - **Done** — most recent ~10 items, collapsed by default.
  - Each row shows: type icon (feature/bug/chore color-coded
    pixel) → story title → epic chip → owner avatar →
    points or appetite chip → status pill → small WIP-age
    indicator (only if `in-flight` > median cycle time).
  - **Velocity cut line** (sprint preset only): a horizontal
    indigo line cutting through the Backlog band at the velocity
    boundary. Above the line: "fits this sprint." Below: "doesn't
    fit." The line moves when estimates change.
  - **Cycle cut marker** (cycle preset only): a similar marker
    showing "fits this cycle's appetite."
  - Aging cue: rows in `in-flight` for > 7 days have a subtle
    amber left border; > 14 days have red.
- **Right rail:** when a row is selected, show a compact preview
  of the selected story (similar to Linear). Click the story title
  in the preview to open the full Story detail screen.

### Sample data (~15–20 rows across bands)

- Icebox: "Per-agent cost dashboard," "Slack source for intake."
- Backlog: "Agent trace spans" (5pt / 2w, ready), "Tool-call
  audit log export" (3pt, refined), "Run replay UI" (8pt, drafted),
  + 5 more.
- Current: "Cross-session memory dedupe" (5pt, in-flight, 4d age),
  "Hero install on Windows ARM64" (3pt, in-flight, 9d age, amber),
  "MCP rate limiting" (2pt, in-review).
- Done: "Tripwire system guardrails," "Spec status integrity v2."

### Interactions

- **J/K** navigates rows; **Enter** opens story detail; **Esc**
  closes preview; **Cmd-K** opens command palette.
- **Drag** to reorder within or across bands.
- **Shift-click** selects a range.
- **`R`** opens `/refine` on selected story (ambient skill
  invocation). **`T`** opens type-change menu. **`A`** opens
  assignee picker.
- Clicking the WIP-age indicator opens a tooltip with
  "in `in-flight` since YYYY-MM-DD."

### Must NOT do

- Don't render as four kanban columns. The single list is the
  feature.
- Don't force estimates under kanban or cycle preset.
- Don't hide Done — keep the last ~10 visible (collapsed but
  one-click expand).
- No nested swim lanes inside bands.

### Preset variations

- **Kanban:** hide velocity cut line, show WIP limits per band
  with a count vs limit (`Current 7/10`).
- **Sprint:** show velocity cut line at the active sprint's
  capacity; sprint name in chrome.
- **Cycle:** show cycle-fit marker; cooldown cycle is a separate
  band in Done state.
- **Phased:** add a `Release` filter chip; group within band by
  release if filter active.

---

## Screen 4 — PRD editor (Notion-fidelity)

> Earns principle #2 (define). Influences: Notion (block editing) +
> Shape Up (pitch sections) + Aha (goal context strip).

### Purpose

Long-form PRD authoring surface. Block-based, structured-but-flexible,
linked.

### Layout

- **Left nav:** shared shell. "PRDs" highlighted; a tree of recent
  PRDs visible below.
- **Content pane:**
  - **Goal context strip** at the top: a horizontal breadcrumb
    showing `Roadmap-item: Agent Observability → PRD: Agent Trace
    Spans`. Each segment links.
  - **Title:** "PRD — Agent Trace Spans" (large, inline-editable).
  - **Metadata row** below title: status pill (`draft / review /
    approved / delivered`), author avatar, last updated, "12
    linked stories" chip.
  - **Pitch-shaped section list** (default template, editable):
    1. **Problem** — ~2 paragraphs. Sample: "SREs running agent
       fleets cannot correlate agent activity with downstream
       service incidents because runs don't emit trace spans…"
    2. **Appetite** — explicit constraint chip + 1 paragraph.
       "2 weeks. We commit to span emission and a default
       backend integration; further dashboards are deferred."
    3. **Solution** — bullets + an embedded toggle showing a
       diagram-block placeholder.
    4. **Rabbit Holes** — bullets, each starting with "Avoid:
       …" Sample: "Avoid: building a generic tracing SDK from
       scratch. Use OpenTelemetry."
    5. **No-Gos** — explicit out-of-scope bullets.
    6. **Linked stories** — an embedded filtered list of all
       `story` specs that point at this PRD. Each row: title,
       status, owner, points/appetite. Click → opens Story
       detail.
    7. **Risks & open questions** — toggle list, each toggle
       expanding to a paragraph.
  - **Slash-command menu** affordance: clicking on an empty line
    reveals a "+" button on the left; clicking "+" or typing `/`
    opens a block-insertion menu (heading, table, toggle, embed,
    code block, story-list-embed).
- **Right rail:**
  - **Review state panel**: a small reviewer roster (3 avatars)
    with check / pending status, and an "Approve PRD" button
    (disabled until all reviewers check).
  - **Backlinks**: list of specs that `@`-mention this PRD.
    Sample: "Story-2847 references this PRD."
  - **Ambient AI panel**: "Suggest acceptance criteria from
    Solution section," "Find related decisions," "Summarize for
    standup."

### Interactions

- **Block drag** — hovering reveals a drag handle on the left
  margin of each block. Drag reorders.
- **`@` autocomplete** — typing `@` opens a fuzzy picker over
  Hero specs (PM and engineering — cross-domain). Selecting one
  inserts a live-linked reference; the reference renders as a
  chip with the spec's type icon.
- **`/` slash menu** — block insertion.
- **Inline-edit title** — click the h1, edit in place.
- **"Approve PRD"** button → state flips to `approved`, all
  reviewers' checks turn solid.

### Must NOT do

- Don't ship a freeform document with no template. The default
  template (pitch shape) is the opinion.
- Don't allow restructuring of the top-level section names
  freely — the section names are part of the template and
  enforced; *content* within sections is fully free.
- Don't separate PRD from its linked stories. The embedded
  story list is non-negotiable.
- No version history sidebar in v1; defer.
- Don't use a WYSIWYG toolbar pinned to the top. Slash-commands
  and inline shortcuts only — like Notion / Linear projects.

### Preset variations

- **Cycle preset:** Appetite section is mandatory (lint warns
  if empty); Rabbit Holes section is mandatory.
- **Sprint preset:** Appetite section becomes optional; "Target
  sprint" field surfaces in metadata row.
- **Kanban / Phased:** Appetite is optional; section list
  identical otherwise.
- **Horizon:** no impact on PRD editor; horizon lives on
  roadmap-item, not PRD.

---

## Screen 5 — Intake funnel (Productboard-influenced)

> Earns principles #1 (decide), #3 (trade-offs). Influences:
> Productboard (source-tagged inbox + highlight-to-link) +
> Linear (Triage view shape) + Height (duplicate detection).

### Purpose

The inbound funnel. Customer feedback, sales notes, support
escalations, competitive signals — all land here, get triaged,
and either promote to roadmap-items, merge with existing items,
or get rejected with a recorded reason.

### Layout

- **Left nav:** shared shell. "Intake" highlighted, with an
  unread-count badge (`23`).
- **Top chrome additions:** a yellow accent strip along the
  content pane top (echoing Linear's Triage view treatment).
  Filter chips: "Source: all," "Segment: all," "Status: new,"
  "Age: any."
- **Content pane:** split horizontally.
  - **Left list (~40%):** table of intake-items, ~32px rows.
    Each row:
    - Source icon (Intercom logo, Slack logo, customer email
      icon, internal-note icon, sales-CRM icon).
    - Excerpt — first line of feedback, truncated.
    - Source attribution: "from Acme Corp · Enterprise · Mar 14"
      OR "from #pm-feedback · @paula · 2h ago."
    - Status pill: `new` (yellow), `triaged`, `linked`,
      `rejected`.
    - Age (relative time).
    - Duplicate hint (if AI detects a near-duplicate): a small
      "⊟ 3 similar" chip.
  - **Right detail pane (~60%):** the currently-selected
    intake-item.
    - Full quote / message, with source metadata above.
    - Customer/source context block.
    - **Highlight-to-link interaction:** the user can select
      any range of the quote and a floating action bar appears:
      "Link to roadmap-item ▸" / "Promote to roadmap-item" /
      "Highlight for evidence." (Mockup should show the
      floating bar active over a selected text range.)
    - **Triage action bar** below: three primary buttons:
      - `Link to existing` (opens roadmap-item picker)
      - `Promote to new roadmap-item` (opens a quick-create
        modal with the highlighted text pre-filled as the
        item description)
      - `Reject with reason` (opens a reason-picker:
        "duplicate," "out of scope," "low signal," "wrong
        product area," other-text).
    - **Suggested matches** panel: AI-suggested existing
      roadmap-items the intake might belong to, with confidence
      and a one-click "Link to this" action.
- **Right rail (collapsed by default; expands on demand):**
  ambient AI panel with: "Find duplicates of this item,"
  "Cluster recent intake," "Summarize this week's inbound."

### Sample data

- Source: Intercom / "We really need to see which agents made
  which API calls — it's currently impossible to audit." / Acme
  Corp · Enterprise · 2h ago / `new`.
- Source: Sales CRM note / "Prospect (Vortex Industries) asked
  about cost attribution per agent before they'd sign." / 1d
  ago / `new`.
- Source: Slack #pm-feedback / "Customer in onboarding got
  confused by the cycle vs sprint preset choice — copy was
  unclear." / 3h ago / `triaged`.
- Source: Support ticket / "Spans are dropping during long
  runs (>30 min). Workaround: increase flush interval." / 4d
  ago / `linked` to roadmap-item "Agent trace spans."
- Source: Competitive scan (internal) / "Linear shipped
  multi-cycle planning today." / 1d ago / `rejected`,
  reason: "noted; not a feature ask."

### Interactions

- Click a row → right detail pane updates.
- Select text in the quote → floating action bar appears
  (highlight-to-link is the killer interaction here).
- Click "Promote to new roadmap-item" → modal slides in from
  the right with title + description pre-filled.
- Click "⊟ 3 similar" chip on a row → filters list to the
  3 near-duplicates.
- Keyboard: `J/K` for row nav, `L` for link, `P` for
  promote, `R` for reject, `M` for merge.

### Must NOT do

- Don't hide the source. Source attribution is the trust
  signal that makes prioritization defensible.
- Don't let intake-items rot. Items > N days old should show
  an aging indicator (subtle red age).
- Don't force a reason category list; allow free-text reason
  on reject.
- Don't separate "merge" and "link" — merging is just
  "link + close." One action.

### Preset variations

- No preset-driven variations. Intake is universal.

---

## Screen 6 — Cross-domain handoff stream

> **The silo-tear made visible.** Earns principles #4 (align),
> #5 (learn). Influences: Linear (cycle timeline) + Kanbanize
> (cycle-time histogram) + a custom Hero-shaped delivery rail.

### Purpose

Timeline showing every PM story handed off to engineering, with
each row exposing the *current engineering delivery state* pulled
live across the domain boundary. This is the proof the platform
is not two silos.

### Layout

- **Left nav:** shared shell. "Handoff stream" highlighted.
- **Top chrome additions:** date range filter ("Last 30 days"
  default), "Status: any," "Owner: any," "Cycle/Sprint: current."
- **Content pane:**
  - **Header strip:** small cycle-time histogram (Kanbanize
    influence). X-axis: days from handoff to ship. Y-axis:
    count. Show recent distribution (e.g. "p50: 8d, p90: 17d,
    p99: 31d"). One-line caption: "Recent handoffs ship in
    ~8 days, half take longer."
  - **Timeline list:** vertical, grouped by week. Each row is
    ~56px (taller than queue rows because it carries cross-
    domain data).
    - **Left side of row:** PM story info.
      - Story title (truncated).
      - Owner avatar + name.
      - Roadmap-item chip + epic chip.
      - "Handed off: 4d ago" timestamp.
    - **Center column:** the handoff edge made visible — a
      small horizontal arrow in `--accent-handoff` (purple)
      connecting the PM side (left) to the engineering side
      (right). When hovered, tooltip: "story-2847 → feature
      agent-trace-spans (handoff)."
    - **Right side of row:** engineering delivery state.
      - Feature title (truncated).
      - Status pill: `planning` / `delivering` / `in-review` /
        `completed` (matches engineering's lifecycle colors).
      - Acceptance criteria progress: "5 of 7 ✓" with a tiny
        progress bar.
      - Last activity: "2h ago: wire span emitter" (commit-
        message style).
      - Engineering owner avatar + name.
    - **Far right hover affordance:** an "Open in engineering"
      icon link.
  - **Row state variants:** rows have visual treatments by
    health:
    - Green: shipped within p50 expected.
    - Neutral: in progress.
    - Amber: handed off but engineering hasn't started after
      N days.
    - Red: engineering rolled back / spec abandoned / drift
      detected.
  - **"Shipped" rows** include a "shipped → roadmap-item" link
    that closes the learning loop (principle #5): "Shipped
    contribution to Agent Observability roadmap-item."
- **Right rail:** when a row is selected, show:
  - The PM story preview.
  - The engineering feature preview.
  - The handoff edge metadata: created, created by, original
    handoff context (the diff between the story description and
    what `/design` produced).
  - Action buttons: `Open story`, `Open feature`,
    `Re-handoff` (if engineering rejected / spec abandoned).

### Sample data (a week of handoffs)

- Story-2847 "Agent trace spans" → feature `agent-trace-spans`
  / `delivering` / 5 of 7 criteria / Diego Ramirez — handed off
  4d ago, healthy.
- Story-2851 "Cross-session memory dedupe" → feature `memory-
  dedupe-v2` / `delivering` / 2 of 6 criteria / Sarah Chen —
  handed off 9d ago, AMBER (no commit in 5 days).
- Story-2862 "MCP rate limiting" → feature `mcp-rate-limit` /
  `completed` / 4 of 4 criteria / Diego — shipped 2d ago,
  contributes to roadmap-item "Reliability & limits."
- Story-2871 "Tool-call audit log" → feature `tool-audit-log`
  / `planning` / 0 of 5 / unassigned — handed off 1d ago.
- Story-2710 "Hero install on Windows ARM64" → feature `win-
  arm64-install` / `in-review` / 6 of 6 / Aisha — shipping
  imminent.

### Interactions

- Click row → right rail populates.
- Click the engineering side of any row → cross-domain
  navigation: routes to the engineering view of that feature
  spec (domain switcher in left nav flips automatically;
  breadcrumb shows "PM › Handoff stream › Engineering ›
  feature").
- Histogram bars are clickable — click p90 bar to filter list
  to "took > 14 days."
- An empty state (no handoffs yet) shows: "No handoffs yet —
  from a Story detail page, click *Hand off to /design* to
  send work to engineering."

### Must NOT do

- Don't render this as a static report. Engineering delivery
  state must be live (within session) — that's the whole point.
- Don't hide the handoff edge. The purple arrow in the center
  is the brand mark of this screen.
- Don't duplicate the Story queue or Engineering board. This
  is the *interface* between them, not a replica of either.
- Don't omit shipped rows. The principle-#5 loop (learn from
  what shipped) lives here.
- Don't gate cross-domain navigation. Click anywhere on the
  engineering side and you're there.

### Preset variations

- **Cycle preset:** group rows by cycle (not just by week).
  Histogram x-axis covers one cycle's window by default.
- **Sprint preset:** group by sprint. Sprint name in chrome.
- **Kanban / Continuous flow:** group by week. Histogram shows
  rolling 30-day distribution.
- **Phased preset:** group by `target_release`.

---

## Mockup delivery checklist

When the ui-designer agent finishes a mockup, the deliverable should:

- [ ] Be a single self-contained HTML file under
      `.hero/mocks/hero-pm/`.
- [ ] Inline all CSS and JS; no CDN.
- [ ] Use the shared design grammar (color tokens, density, layout
      shell) consistently across all six mockups.
- [ ] Show the canonical preset (Horizon + Cycle 6w) in the chrome,
      with a brief code-comment near the top noting what would
      change under other presets.
- [ ] Use realistic Hero-flavored PM content (no lorem, no foo/bar).
- [ ] Include the killer interaction for each screen as a clearly
      visible affordance (hover-state, active state, or annotated
      with a small dotted callout).
- [ ] Be keyboard-navigable for primary actions (use semantic
      buttons / links — no div-only click targets).
- [ ] Stay under ~800 lines of HTML per file; the goal is a
      review prototype, not production code.
- [ ] Open each mockup in a browser cold and confirm it renders
      with no console errors.
