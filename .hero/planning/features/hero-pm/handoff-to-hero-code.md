---
title: Hero PM — Handoff to hero-code for implementation kickoff
type: reference
status: handoff
created: 2026-05-15
relations:
  - target: hero-pm
    kind: companion
  - target: hero-domains
    kind: companion
audience: hero-code session starting PM domain implementation
---

## What this is

A self-contained handoff doc for a fresh Claude Code session running
in the `hero-code` repository. The PM domain design is complete enough
to start building. This doc points to every artifact, surfaces the
locked decisions, and lists what's still open. Read it end-to-end
before touching any code.

The design lives in the sibling `hero` repository at `../hero/`.
Implementation lives in `hero-code` (this repo).

## Start here

Read these four files in order. They are load-bearing — don't
short-circuit:

1. `../hero/.hero/planning/initiatives/hero-domains/spec.md` — the
   parent initiative. Sets the multi-domain expansion frame, the
   PM-first sequencing, the 6 platform primitives that must land
   before PM, and the deferred domain backlog.
2. `../hero/.hero/planning/features/hero-pm/spec.md` — the PM
   domain pack spec. Artifact types (prd / story / epic /
   roadmap-item / intake-item), methodology layers (layered
   presets, not modes), guiding principles, silo-tearing patterns,
   dashboard views, agent-pack summary.
3. `../hero/.hero/planning/features/hero-pm/agent-pack-design.md` —
   1,663-line design covering 27 agents, 32 skills, 22 commands,
   the natural-language routing table, the contextual-button
   inventory, and prior-art research from MetaGPT, BMad-Method,
   ChatPRD, and wshobson/agents.
4. `../hero/.hero/planning/features/hero-pm/mockups/index.html` —
   the 9 self-contained HTML mockups. Open these in a browser to
   see the locked UX model in action. Treat them as design source
   of truth for layout, density, and visual fidelity.

Supporting context (read as needed):

- `../hero/.hero/planning/features/hero-pm/research-brief.md` —
  per-tool UX research (Pivotal Tracker, Linear, Jira, Shape Up,
  Productboard, Aha, Notion, etc.) with what to take and what
  to leave per tool. Design philosophy distilled at the bottom.
- `../hero/.hero/planning/features/hero-pm/mockup-brief.md` —
  the screen-by-screen mockup brief that drove the HTML pass.
  Useful when the mockups need to be updated.

## The 9 mockups

All under `../hero/.hero/planning/features/hero-pm/mockups/`:

| # | File | Purpose |
|---|---|---|
| 0 | `index.html` | Landing page with previews and the locked UX model |
| 1 | `01-story-detail.html` | Story detail with contextual buttons + cross-domain handoff to engineering feature (the platform thesis screen) |
| 2 | `02-roadmap-board.html` | Now/Next/Later roadmap with Shape Up-influenced betting table cues, saved-view nesting |
| 3 | `03-story-queue.html` | Pivotal Tracker-influenced single-list queue with icebox/backlog/current/done |
| 4 | `04-prd-editor.html` | Notion-fidelity PRD authoring with markdown-with-slash-commands editing (NOT a block editor) |
| 5 | `05-intake-funnel.html` | Productboard-influenced triage. Demonstrates parameterized-singleton pattern (two Intake tabs open: master + "Mobile bugs" saved view) |
| 6 | `06-handoff-stream.html` | The silo-tear made visible. Live engineering delivery status pulled across the domain boundary |
| 7 | `07-chat-tab.html` | The Chat tab — design-mode surface. Full center pane, hero-code chrome (provider/model picker, context, attachments). Single rolling conversation. |
| 8 | `08-inline-proposal.html` | Story detail with 3 acceptance criteria proposed inline by `story-writer` (dotted-border, accept/edit/reject) |

## Locked design decisions

These are settled. Apply them, don't relitigate them:

### Layout grammar (IDE-style)

**Guiding principle**: PMs work in two modes. **Designing** (chat is
the work — drafting PRDs from scratch, strategic dialogue, discovery
exploration). **Housekeeping** (artifact is the work — refining,
triaging, hand-offs, accepting agent proposals). Mode-switching is
tab-switching. Chat doesn't compete with artifacts for screen space.

#### Left nav (~220–240px)

Inventory of openables. Includes:

- Roadmap, Library (Stories / PRDs / Epics), Intake, Story Queue,
  Handoff stream, Tracker mirrors
- **Chat** — a prominent entry near the top with the bolt icon.
  Single-click access; no hotkey required. Power users get cmd-K
  as an alternative path.
- **Saved views as parameterized singletons** nested under their
  parent (e.g. "Mobile bugs" under Intake; "Q3 bets" under Roadmap).
  User-named.

#### Center pane (tabbed openables, VS Code-style)

- Singleton tabs (one ever): Roadmap, Story Queue, Intake, Handoff
  stream, Search, **Chat** (when opened from left nav)
- Per-item tabs (one per spec): Story detail, PRD editor, Epic detail,
  Roadmap-item detail, Intake-item detail
- Opening an already-open thing flips to its tab; never duplicates
- **Chat tab** = the design-mode surface. Full center pane, full
  hero-code visual fidelity (provider/model picker, context fullness,
  attachments, mentions, slash, reasoning, single rolling conversation).
  When you're here, you're *in* the conversation.

#### Artifact body (the thing itself)

- **Header chips**: identity/state metadata — status, owner, priority,
  sprint/cycle, points, tracker link, tags. Inline at the top of the
  artifact.
- **Body**: the artifact content + load-bearing inline elements.
  Linked engineering feature **card with the hand-off button** lives
  inline (not in a rail) — it's the killer-demo affordance and earns
  prominent in-body placement. Hill chart inline (cycle preset). AC
  list, PRD sections, roadmap card content — all in the body.

#### Bottom strip (~70–90px, anchored to bottom of center pane, artifact tabs only)

The artifact-mode chat-and-action surface. Three elements:

- **State-aware contextual button row** — 4-6 verb-shaped buttons,
  varies by artifact and its state. Examples:
  - Drafted story: Refine, Draft AC, Split, Find duplicates, Discuss
  - Ready story: Hand off to /design, Find similar, Audit, Discuss
  - PRD: Add section, Suggest AC, Draft Risks, Audit completeness,
    Discuss
  - Roadmap card: Bump up, Bump down, Audit, Find similar, Discuss
- **Chat input** below the buttons — single line by default, expands
  on focus. Slash, mentions, attachments accessible inline.
- **Minimal chrome** — model chip (collapsed; expands on click for
  provider/model picker), send affordance.
- **Bottom strip is NOT present on the Chat tab** — when Chat is the
  active tab, you're in design mode and the chat IS the surface.
  Bottom strip only appears under artifact / workspace tabs.

#### Right panel (the chat panel for artifact mode, toggleable)

When open: shows the conversation for housekeeping mode. Structure:

- **Sticky pinned top** (~60–120px, often less, sometimes empty):
  the ambient region. Smarts about the active artifact.
  - **Hard relationships** (when they exist): parent epic chip,
    linked engineering feature reference + delivery status, related
    decisions
  - **Dynamic agent suggestions** (when relevant): "Similar to Story
    #234 (78%) — merge?", "Possible parent: 'Auth Refresh' epic
    (87%)", "Missing: description, AC" — each with inline accept /
    dismiss / customize affordances. Max 3-4 visible; dismissals
    persist per-artifact until state changes.
  - **Activity rollup** (when any exist): "3 events →" expandable
  - **Starter helpers** (only for new/empty artifacts): template
    pickers, quick-draft offers
  - **Empty state renders nothing** — no "no related items"
    placeholder; the chat scroll just starts at the top of the panel.
- **Chat scroll below** — the single rolling conversation, hero-code
  visual fidelity: bubbles 14/10px padding, 12px radius, 13/21px type,
  user bubble right-aligned, agent bubble left-aligned. "via /command"
  breadcrumb when slash-originated. Tool calls as expandable cards.
  Reasoning as collapsible "Thinking" pills. Meaningful system events
  (links, hand-offs, status flips, agent actions accepted/rejected)
  render in-thread as system messages mixed with conversational turns.

Properties:

- **Width**: ~360–400px default. Resizable boundary on the left edge.
- **Toggleable**: hide entirely for full-bleed artifact mode; re-open
  via chrome chevron. Default state on first run: open.
- **Single rolling chat for v1** — same conversation thread as the
  Chat tab. View it slim in the right panel, full in the Chat tab.
  Multi-session deferred to v2.
- **Switching artifacts updates the sticky pinned top** (new ambient
  for new artifact); chat scroll does NOT reset (conversation
  continues; agent context awareness shifts to the new artifact).
- **The sticky pinned top is structurally part of the chat scroll**
  — not a separately togglable region. It comes and goes with the
  panel.

#### Surface roles (the bedrock distinction)

| Surface | Word type | Role |
|---|---|---|
| **Bottom strip buttons** | **Verbs** | Concrete actions the user takes on the artifact (user-initiated) |
| **Ambient (sticky chat top)** | **Nouns / smarts** | Info, suggestions, helpers — what the agent knows or surfaces about the artifact (agent-initiated) |
| **Artifact header chips** | **Identity / state** | What this thing IS — status, owner, priority |
| **Artifact body** | **Content** | The thing's own essence + load-bearing inline elements |
| **Chat scroll** | **Time-ordered** | Conversation + meaningful system events |

The cleanest test: *verb vs noun*. *User-initiated vs agent-initiated*.
*Transient result (inline annotation on artifact) vs stable
relationship (ambient)*.

### Inline-proposed agent outputs

Agent-drafted content appears **in the artifact, in place**, marked
proposed with a dotted/dashed border, a "proposed by `<agent-name>`"
badge, and accept / edit / reject affordances. **The chat does NOT
show the proposed content** — only a log line ("`story-writer` drafted
4 AC → 3 accepted, 1 edited"). The artifact is source of truth; the
chat is the trace.

### Contextual buttons

On each artifact (story header, AC list, PRD section, roadmap card),
inline buttons trigger agents directly. Pulls from `agent-pack-design.md`
§G. The "Hand off to /design" button on Story detail is the most
visually prominent action on the page — it's the platform thesis
in one click.

### Tracker fronting & local-first (Hero-wide principle)

Hero is the working surface. Trackers (Jira / Linear / GitHub) and
Hero Cloud are backing stores, not front doors. Three operating modes,
one UX:

1. **Standalone** — `.hero/` filesystem is source of truth.
2. **Hero Cloud-backed** — Cloud is the sync layer for multi-user.
3. **Tracker-fronted** — tracker is the org system of record; Hero
   fronts it transparently. Local-first writes; async propagation;
   no syncing spinners; tracker changes flow back via webhook + poll.

The UX is identical in all three modes. Artifacts are round-trippable
markdown specs in `.hero/`. Conflict policy: Hero wins for content,
tracker wins for org-state fields (assignee, sprint, workflow status).

This principle applies to all domain packs, not just PM. It is
**not yet captured in a dedicated decision doc** — see open task
below.

### Brand

Hero brand is the **lightning bolt** (path: `M52 8 L22 46 L40 46 L34 82
L68 42 L50 42 Z`, viewBox 0 0 90 90) and the **light-blue palette**:

- `--hero-blue-300: #9bc1e6` — logo fill, light accents
- `--hero-blue-500: #6cb6ff` — mid, brighter dark-mode accent
- `--hero-blue-700: #2a6cb5` — deep, primary accent on light bg
- `--hero-ink: #14181e` — dark chrome

**Do not use** Linear's indigo (#5e6ad2) or purple (#7c3aed) as the
Hero brand. **Do not use** an "H" letterform as the logo. Purple is
reserved exclusively as the **cross-domain accent** — visible only
on the handoff edge (Story detail's linked engineering feature card,
Handoff stream's per-row domain bars).

Source: `../hero/web/docs/src/assets/logo.svg` and
`../hero/web/docs/src/stylesheets/brand.css`.

## Open questions — resolved 2026-05-16

### 1. Tracker-fronting decision doc — DONE

Written as `.hero/knowledge/decisions/tracker-fronting-and-local-first.md`.
Cross-referenced from `hero-domains/spec.md` (Hero-wide principles
section) and `hero-pm/spec.md` (Integrations section). The conflict
policy (Hero wins on content, tracker wins on org-state) is now the
canonical reference for the spec-type-registry primitive's
content-vs-org-state field declaration.

### 2. Inline-propose output mode primitive (#4b) — DONE

Spun up as a sibling spec at
`.hero/planning/features/inline-propose-output-mode/spec.md`. Added
to the `hero-domains` initiative's children table as #4b and to its
sequencing section between #4 and #5. Added to `hero-pm`'s
`depends-on` list. `/design inline-propose-output-mode` will resolve
the remaining open questions in that spec (storage default,
idempotency under repeat-propose, edit fidelity).

## Sequencing — what to build first

The parent initiative's sequence is binding. Don't jump ahead.

| # | Slug | What it does | Why first |
|---|---|---|---|
| 1 | domain-plugin-architecture | Refactor existing engineering content into `domains/engineering/`; add `hero init --domain`, `hero domain switch`, `hero.json` domain field | Pure refactor. Foundation. Every later item assumes packs exist. |
| 2 | spec-type-registry | Domain-declared spec types and lifecycles | Blocking for PM (PRD / story / epic / roadmap-item / intake-item don't exist as types today) |
| 3 | domain-routing-and-agents | Active-domain AGENTS.md and agent loader | Without this, model routes to engineering agents in PM project |
| 4 | dashboard-view-registry | Pluggable dashboard pages + entity detail views per domain | PM ships its own views |
| 4b | inline-propose output mode | Agents propose into artifact pane; accept / edit / reject UI | Required for the locked inline-proposed pattern. To be added. |
| 5 | scan-pluggability | Per-domain scanners | PM onboarding (import roadmap docs, tracker epics) needs this |
| 6 | domain-scoped-knowledge-graph | Namespace tags on graph nodes | P0 because the killer demo (story → feature handoff) requires cross-domain graph from day one |
| 7 | hero-pm | The PM domain pack itself | Validates the platform end-to-end |
| 8 | hero-qa | The QA domain pack | Second domain — closes downstream loop |

Each child spec is at `../hero/.hero/planning/features/<slug>/spec.md`.
Open each one when you're ready to deliver it.

## Where implementation lives in hero-code

The parent initiative envisions content packs at `domains/<name>/` —
agents, commands, skills, integrations, AGENTS.md per pack. The PM
pack would live at `domains/pm/` after `domain-plugin-architecture`
lands.

Until that primitive lands, no PM-specific implementation should be
drafted in `domains/pm/`. The right first move in hero-code is to
deliver primitive #1 and migrate engineering content into
`domains/engineering/` as part of the refactor.

If `/deliver hero-pm` needs prompt-shape proofs to de-risk the design
ahead of time, the agent-pack design (§C) recommends drafting 2-3
representative agent prompts as exemplars only — `handoff-coordinator`,
`story-writer`, `prd-author` are the strongest candidates. These would
land in a scratch location and migrate into `domains/pm/agents/` once
primitive #1 ships.

## Cross-repo conventions

Hero supports cross-repo peering. The PM specs live in the `hero` repo;
hero-code is the implementation repo. Reference specs by relative path
(`../hero/.hero/planning/...`). Hero's import mechanism may already
handle this — check `../hero/.hero/knowledge/decisions/cross-repo-peering-local-first.md`
for the conventions.

If hero-code's `.hero/planning/` workspace needs its own copies of
relevant specs (mirror for status tracking), follow the cross-repo
peering pattern — don't fork the content.

## Recommended kickoff prompt for the hero-code session

Paste this into a fresh Claude Code session in the `hero-code` repo:

> Read `../hero/.hero/planning/features/hero-pm/handoff-to-hero-code.md`
> end-to-end. Then read the four "Start here" files it points to.
> Don't begin any implementation yet — first confirm the two remaining
> open questions (tracker-fronting decision doc, inline-propose
> primitive #4b) with the user. Once those are resolved, the
> highest-leverage next move is to `/deliver domain-plugin-architecture`
> — the foundation primitive that unblocks the rest of the roadmap.

## What NOT to do in hero-code

- Don't start building PM domain content (`domains/pm/agents/`, etc.)
  before `domain-plugin-architecture` lands. The directory layout
  doesn't exist yet.
- Don't redesign the UX layout. The mockups are the design source of
  truth. If you find something that doesn't work in implementation,
  surface it back to the user and update the mockups — don't quietly
  diverge.
- Don't use Linear's indigo / purple as the Hero brand. Don't use
  an "H" letterform. The lightning bolt is the logo.
- Don't put chat in the right rail. The right rail is for spec-relevant
  ambient context only.
- Don't add new artifact types beyond the locked five (prd, story,
  epic, roadmap-item, intake-item).
