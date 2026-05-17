---
title: Hero QA — Handoff to hero-code for implementation kickoff
type: reference
status: handoff
created: 2026-05-17
relations:
  - target: hero-qa
    kind: companion
  - target: hero-domains
    kind: companion
  - target: hero-pm
    kind: sequenced-after
audience: hero-code session starting QA domain UI implementation
---

## What this is

A self-contained handoff doc for a fresh Claude Code session running
in the `hero-code` repository. The QA domain design is complete enough
to start building the UI surfaces. This doc points to every artifact,
surfaces the locked decisions, calls out dependencies on PM-side work,
and lists what's still open. Read it end-to-end before touching code.

The design lives in the sibling `hero` repository at `../hero/`.
Implementation lives in `hero-code` (this repo). The hero-code-side
planning entry for this work is
`.hero/planning/features/hero-qa-ui/spec.md` — open that next.

QA is **sequenced after PM** in the initiative roadmap. Some
platform primitives must land before either domain pack ships;
others are shared between PM and QA. Where QA adds new requirements
beyond what PM defined, this doc calls them out as **amendments**.

## Start here

Read these four files in order. They are load-bearing — don't
short-circuit:

1. `../hero/.hero/planning/initiatives/hero-domains/spec.md` — the
   parent initiative. Multi-domain expansion frame, sequencing,
   the 6 platform primitives + #4b (inline-propose), the deferred
   domain backlog. **Updated 2026-05-17** with the two new
   primitive amendments QA forces (lifecycle overlays on #2,
   cross-pack ambient on #4).
2. `../hero/.hero/planning/features/hero-qa/spec.md` — the QA
   domain pack spec. 32 locked decisions. Artifact types
   (test-plan, regression-suite, test-case, release-gate, opt-in
   defect), gate_style presets (inline / parallel / post-release),
   cross-pack lifecycle overlays, three-action rejection composer,
   four-action test-issue triage, normalization layer for the
   unified inbox, local-first promise.
3. `../hero/.hero/planning/features/hero-qa/agent-pack-design.md` —
   1,119-line design covering 23 agents, 26 skills, 18 commands,
   the natural-language routing table, the contextual-button
   inventory per artifact, P0/P1/P2 priority calls, and prior-art
   research synthesizing PM's wshobson/MetaGPT/BMad-Method study
   plus QA-specific deep-dives on mabl, Functionize, TestRail,
   Xray, ISTQB methodology.
4. `../hero/.hero/planning/features/hero-qa/mockups/index.html` —
   the 9 self-contained HTML mockups including the new Screen 09
   (AI authoring mid-flight — the brand demo for inline-propose).
   Open in a browser to see the locked UX model. Treat as design
   source of truth.

Supporting context (read as needed):

- `../hero/.hero/planning/features/hero-qa/research-brief.md` —
  887-line competitive UX research (TestRail, Xray, Zephyr, qTest,
  mabl, Functionize, Testim, Applitools, QA Wolf, + methodology
  sources). Steal/leave matrix per source. Names six design-original
  Hero-QA contributions worth quoting in launch positioning.
- `../hero/.hero/planning/features/hero-qa/mockup-brief.md` —
  the screen-by-screen mockup brief that drove the HTML pass.

You should also read the **PM handoff doc** at
`../hero/.hero/planning/features/hero-pm/handoff-to-hero-code.md`
end-to-end — it carries the locked UX grammar, brand rules, and
the inline-propose pattern that QA reuses wholesale. QA's
mockup-brief explicitly defers to PM for those.

## The 9 mockups

All under `../hero/.hero/planning/features/hero-qa/mockups/`:

| # | File | Purpose |
|---|---|---|
| 0 | `index.html` | Landing page with previews and the AI-workflow framing |
| 1 | `01-story-queue-coverage.html` | Default landing — sprint stories with per-row coverage signals, In-QA / In-flight / Ready / Done bands, AI-suggested next action under selected row |
| 2 | `02-test-plan-editor.html` | Coverage matrix (5 AC × 5 cases + 6th proposed column with pulsing PROPOSED state from `test-author`); gap callout above matrix |
| 3 | `03-test-case-editor.html` | Format-aware case editor with AI provenance chip on header ("Authored by `test-author` · accepted by Jamie · derived from AC#3 via boundary-value + state-transition"); Step ↔ Gherkin toggle is interactive; full authoring-history chat scroll |
| 4 | `04-story-qa-ready-rejection.html` | **Brand interaction.** PM story in `qa-ready`, three-action composer in bottom strip mid-composition. The findings textarea is a `.proposed-block.edited` from `qa-investigator` — not a blank field. Cross-pack preview tile shows what Diego will see in his engineering session on submit |
| 5 | `05-release-gate.html` | Go/No-Go verdict banner with collapsible reasoning proposed-block from `release-gate-reviewer` (92% confidence) above; named blockers with direct actions; chat thread answers "what would flip this to Go today?" |
| 6 | `06-flaky-backlog.html` | Active queue driving flake count toward zero. Classify split-button visually anchored on `test-issue` (qa-flake-curator's 87%-confidence suggestion). Inline proposed-block with ranked alternatives under RC-441 |
| 7 | `07-unified-inbox.html` | Kind tabs (All / Bugs / Test issues) + normalized rendering across Jira/TestRail/Xray. Four-action triage panel **ranked by `test-issue-triager` confidence** (Reject linked story 87 / Raise as new bug 8 / Flag as regression 4 / Mark as bad test 1) |
| 8 | `08-handoff-stream-qa.html` | Cross-domain events as a graph timeline with edge-type chip vocabulary; `qa-investigator` pattern-detection proposed-block ("Story-2847 has 2 rejections — route to PM via `pm-rejection-router`?") |
| 9 | `09-ai-authoring-inflight.html` | **AI brand demo.** Jamie clicked Author cases 90s ago; 5 cases land inline as proposed-blocks — 2 accepted (green), 1 edited (amber), 2 pending (blue, pulsing). Chat input pre-filled with Jamie's next refinement. QA equivalent of PM's `08-inline-proposal.html` |

## Locked design decisions

The 32 locks live in `hero-qa/spec.md` (table at the bottom). The
highest-leverage subset for implementation:

### UX grammar — INHERITED from PM (do not redesign)

L1: same IDE-style left nav · tabbed center · bottom strip (verb row
+ chat input row) · right panel (sticky ambient + chat scroll).
Bolt logo, Hero-blue palette, Inter font, ~32px row density.
Reference PM's handoff doc §Layout grammar for the full spec; QA
adds nothing here.

### QA-specific UX

- **Chat tab** is a pinned `.nav-link.chat-entry` at the top of the
  left nav (bolt icon, hero-blue-700 text). Same conversation visible
  in the right rail.
- **Bottom strip** is two rows on artifact tabs: state-aware verb
  buttons (the per-artifact contextual button inventory lives in
  `agent-pack-design.md` §G) on top + `.bottom-input` row underneath
  (model chip "Sonnet 4.6" + input + `/` and `@` tool affordances +
  send button).
- **Right rail** structure: `.right-header` ("Chat · {artifact-id}") →
  `.ambient` section (relationships + actionable `.ambient-suggest`
  cards with pulse + agent name + Accept/Dismiss/Override) →
  `.chat-scroll` (`.bubble-wrap.user` with `.via /command` and
  `.bubble.user`; `.bubble-wrap.agent` with `.bubble.agent`;
  `.sys-event.inline-event` rows for state transitions).
- **Inline-propose blocks** are the heart of every AI-authoring
  moment: `.proposed-block` containing `.proposed-badge`
  ("proposed by `{agent}` · {time}" with pulsing dot), `.proposed-text`
  (content), `.proposed-actions` (Accept primary / Edit / Reject /
  hint "⏎ to accept"). Border color shifts on state:
  `.proposed-block` (blue, pending) → `.proposed-block.edited`
  (amber, edited by user) → `.proposed-block.accepted` (green,
  actions hidden).

### QA-specific color tokens (additions to PM palette)

```css
--accent-coverage-full: #16a249;     /* covered ✓ */
--accent-coverage-partial: #d97706;  /* partial ⚠ */
--accent-coverage-none: #dc2626;     /* uncovered ! */
--accent-qa-gutter: #0891b2;         /* cyan-teal — QA-authored
                                        cross-pack content (gutter
                                        icon on QA Findings block) */
--accent-qa-ready: #0284c7;          /* qa-ready story state chip */
--accent-qa-rejected: #d97706;       /* qa-rejected story state chip */
--accent-flaky: #a855f7;             /* flake-violet */
```

### Cross-pack lifecycle overlays (the architectural amendment)

L8 + L12 + L16: QA pack registers a **lifecycle overlay** on PM's
`story` type — adds `qa-ready` and `qa-rejected` states without
modifying the PM-owned type. The spec-type registry (primitive #2)
must accept overlays from non-owning packs in addition to
methodology-layered-field declarations PM already needs. Graceful
degradation: when QA pack is uninstalled, stored extended states
render as labels (not transitionable). **This is a primitive #2
amendment beyond what PM's handoff specified.**

### Cross-pack ambient population (the second architectural amendment)

When QA rejects a story, an ambient card appears on the linked
engineering `feature` in the engineering pack's view. This means
the dashboard view registry (primitive #4) must support cross-pack
ambient producers — a pack registers an ambient-card producer that
fires on artifacts owned by a different pack. **This is a primitive
#4 amendment beyond what PM's handoff specified.**

### The brand interaction

L9 + L18: the three-action rejection composer (Add AC / Suggest
new story / Reject as quality issue) on stories in `qa-ready`.
Anti-goalpost-moving by design — the `Suggest new story` path
prevents QA from inflating stories by reinterpreting AC. Default
nudge surfaces based on finding shape; configurable strictness
(`block` / `advise`).

### The four-action test-issue triage

L30: every test issue triages to one of four outcomes — bad-test /
reject-linked-story (uses the three-action composer above) /
raise-as-new-bug (routes to engineering `/diagnose`) /
flag-as-regression (raises bug + flags regression suite). The
triage panel surfaces actions **ranked by `test-issue-triager`
confidence** so the user sees which outcome the AI thinks is right
without losing access to the others.

### The unified inbox normalization

L23 + L24 + L25: bugs and test issues are **distinct artifact
kinds** that link via relationships, not duplicate copies. One
inbox with kind tabs (All / Bugs / Test issues). Each integration
provides a mapper to a standard schema (status, severity, priority,
assignee, age, link); source-specific fields decorate rows for
fidelity. No auto-merge across sources — same-source duplicate
merging is a manual `Mark as duplicate of...` action only.

### Local-first promise

L28: every inbox view works fully off local specs when no
integration is configured. Engineering pack's local `bug` specs
render in the bugs tab on day one. Native QA `defect` records
(opt-in via `qa.defect_lifecycle: "owned"`) render in the test
issues tab. Integration is **additive decoration**, never a
prerequisite. The tracker-fronting-and-local-first decision doc
(`../hero/.hero/knowledge/decisions/tracker-fronting-and-local-first.md`)
is the canonical articulation.

### Configurable defaults — the pack design voice

Recurring pattern across 32 locks: opinionated defaults + dialed
configurability. Configurations to honor:

- `qa.gate_style: "inline" | "parallel" | "post-release"` (default: inline)
- `qa.defect_lifecycle: "funnel" | "owned"` (default: funnel)
- `qa.test_issue_persistence: "persistent-link" | "triage-and-close"` (default: persistent-link)
- `qa.rejection_strictness: "block" | "advise"` (default: block)
- `qa.feature_lifecycle_on_rejection: "informational" | "auto-revert"` (default: informational)
- `qa.case_format_default: "step" | "gherkin" | "decision-table" | "data-driven"` (default: step)
- `qa.blocker_policy: <custom rules>` (default: P0 hold, P1 hold without sign-off, P2+ candidate for next cycle)

## Dependencies on PM work and platform primitives

The QA UI cannot ship until these are in place:

| # | Slug | Why QA needs it |
|---|---|---|
| 1 | domain-plugin-architecture | The QA pack lives at `domains/qa/`. Can't exist before this primitive |
| 2 | spec-type-registry + **lifecycle overlay amendment** | QA declares test-plan / test-case / regression-suite / release-gate. AND injects qa-ready/qa-rejected states into PM's story type |
| 3 | domain-routing-and-agents | Active-domain AGENTS.md routes natural-language to QA agents inside QA workspace |
| 4 | dashboard-view-registry + **cross-pack ambient amendment** | QA registers 9 views. AND can populate ambient cards on PM-owned and engineering-owned artifacts |
| 4b | inline-propose output mode | Every QA authoring agent (`test-author`, `plan-author`, `qa-investigator`-as-composer) needs this. Already specced as a primitive |
| 5 | scan-pluggability | QA onboarding (TestRail / Xray imports + native test-failure ingest) |
| 6 | domain-scoped-knowledge-graph | Cross-pack edges (`raised-bug-from-failure`, `regression-of`, `verifies`, `qa-rejected`) need namespace tags |
| 7 | hero-pm | QA reuses PM's UX grammar wholesale and depends on PM's `story` type existing for the overlay to attach to |

**Soft dependency on PM delivery itself:** QA design can be implemented
in parallel with PM delivery if needed, but the user-facing pack
should not ship before PM ships so the multi-domain narrative holds
together (per the parent initiative's risk note).

## Sequencing — what to build first in hero-code

The hero-code-side feature spec at
`.hero/planning/features/hero-qa-ui/spec.md` carries the full
Rust-side implementation plan. Summary order:

1. **Don't start QA-specific UI implementation yet.** First the
   platform primitives (#1, #2 with lifecycle-overlay amendment,
   #3, #4 with cross-pack ambient amendment, #4b, #5, #6) must
   land. The hero-code session should `/deliver
   domain-plugin-architecture` first.
2. **`base-hero-ui` is the foundation.** The pluggable chat-first
   shell at `hero-code/.hero/planning/features/base-hero-ui/spec.md`
   carries the abstractions QA needs (UI profile, pluggable sidebar
   middle pane, pluggable center-pane document viewer, markdown
   in chat). Land this in parallel with the primitives.
3. **QA-specific surfaces ride on top:**
   - 9 dashboard views per `agent-pack-design.md` and mockups
   - The 23 agent prompt files at `domains/qa/agents/*.md`
   - The 26 skill files at `domains/qa/skills/*/SKILL.md`
   - The 18 command routers at `domains/qa/commands/*.md`
   - The 4 demos must pass end-to-end (see `agent-pack-design.md`
     §End notes for the four-demo bar).

## Where implementation lives in hero-code

Per the architecture in `hero-domains/spec.md`:

```
domains/
  engineering/         # existing (post-primitive-#1 refactor)
  pm/                  # PM pack (after primitive #1 + PM delivery)
  qa/                  # QA pack (after primitive #1 + QA delivery)
    agents/            # 23 agents from agent-pack-design.md §C
    skills/            # 26 skills from §D
    commands/          # 18 commands from §E
    spec-types.json    # test-plan, test-case, regression-suite,
                       # release-gate, (opt-in) defect
    spec-type-overlays.json   # qa-ready, qa-rejected on PM story
    views/             # 9 dashboard view manifests
    integrations.json  # xray, testrail (first); jira/gh shared
    AGENTS.md          # QA routing table
```

Rust-side, the UI surfaces map onto hero-code's existing patterns:

- The 9 views become center-pane tab implementations alongside
  `chat_view.rs`, `editor.rs`, etc.
- The pluggable sidebar profile from `base-hero-ui` carries the QA
  left-nav layout.
- The `.proposed-block` and surrounding inline-propose mechanics
  attach to the artifact body rendering path (whatever lands in
  the `inline-propose-output-mode` primitive).

## Cross-repo conventions

- All design assets stay in `hero/` (specs, briefs, mockups,
  knowledge). Don't copy them into hero-code; reference via
  `../hero/.hero/planning/...` relative paths.
- The hero-code-side feature spec
  (`hero-code/.hero/planning/features/hero-qa-ui/spec.md`) tracks
  the IMPLEMENTATION work — what Rust changes, what surfaces, what
  sequencing, what's still open on the code side. Status flips on
  this spec as work moves; design status flips happen on the
  hero-side QA spec.

## Open questions for the hero-code session

Resolved before delivery starts:

1. **Lifecycle-overlay storage** — how does the spec-type-registry
   represent overlay state on disk and at the API boundary? Likely
   resolved during `/design spec-type-registry` with the
   lifecycle-overlay amendment.
2. **Cross-pack ambient producer API** — how do packs register
   ambient producers that fire on foreign-pack artifacts? Likely
   resolved during `/design dashboard-view-registry` with the
   cross-pack ambient amendment.
3. **Inline-propose persistence** — when a user closes a session
   mid-flight with pending proposed-blocks unaccepted, do they
   survive the restart? Likely default: yes, stored in artifact
   frontmatter as pending. To be resolved during `/design
   inline-propose-output-mode`.
4. **`base-hero-ui` profile naming for QA** — does the QA profile
   sit alongside engineering/marketing/CS as a named UI profile,
   or does it parameterize a generic "domain" profile? Implementer
   call.

## Recommended kickoff prompt for the hero-code session

Paste this into a fresh Claude Code session in the `hero-code` repo:

> Open the QA UI implementation work.
>
> Read `../hero/.hero/planning/features/hero-qa/handoff-to-hero-code.md`
> end-to-end. Then read the four "Start here" files it points to.
> Also read `../hero/.hero/planning/features/hero-pm/handoff-to-hero-code.md`
> for the UX grammar QA inherits wholesale, and
> `.hero/planning/features/base-hero-ui/spec.md` for the pluggable
> shell QA-specific surfaces ride on top of.
>
> Don't begin implementation yet. First:
> 1. Confirm the two new platform-primitive amendments QA forces
>    (lifecycle overlays on primitive #2, cross-pack ambient
>    population on primitive #4) are reflected in those primitives'
>    specs. If they're not, surface that as a blocker.
> 2. Open `.hero/planning/features/hero-qa-ui/spec.md` (this repo)
>    and align on the Rust-side sequencing — what surfaces hit what
>    files, where they slot relative to `base-hero-ui`.
> 3. Confirm which primitive `/deliver` happens first — likely
>    `domain-plugin-architecture` per the parent initiative's
>    sequencing.
>
> Once those are clear, the highest-leverage next move is to
> `/deliver` whichever primitive is at the head of the queue.
> QA-specific UI implementation does not start until primitives
> #1, #2 (with lifecycle-overlay amendment), #3, #4 (with cross-pack
> ambient amendment), #4b, #5, #6 all land plus `base-hero-ui` and
> PM ships.

## What NOT to do in hero-code

- Don't start building QA domain content (`domains/qa/agents/`,
  `domains/qa/views/`, etc.) before `domain-plugin-architecture`
  lands. The directory layout doesn't exist yet.
- Don't redesign the UX. The mockups are the design source of truth.
  If something doesn't work in implementation, surface it back to
  the user and update the mockups — don't quietly diverge.
- Don't redesign the inline-propose pattern. It's specced as
  primitive #4b and consumed by both PM and QA. Implementation
  changes go in the primitive spec, not in QA-specific code.
- Don't fork the brand. Bolt logo + Hero blue + QA-cyan gutter +
  flake-violet are the QA color additions. No Linear indigo. No
  "H" letterform.
- Don't ship QA before PM. The multi-domain narrative requires
  both, in order.
- Don't add artifact types beyond the locked five (test-plan,
  test-case, regression-suite, release-gate, opt-in defect).
- Don't auto-merge across bug/test-issue sources. Same-source
  duplicate handling is manual only (`Mark as duplicate of...`).
- Don't treat ambient suggestions as decoration. They're actionable
  AI proposals with Accept/Override/Dismiss. The PM mockups got
  this wrong in their first pass; the QA mockups specifically
  retrofitted to fix it. Honor the live-AI feel in implementation.
