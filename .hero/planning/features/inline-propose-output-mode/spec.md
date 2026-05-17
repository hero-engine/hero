---
title: Inline-Propose Output Mode — Agents Propose into the Artifact Pane
type: feature
status: planning
priority: P0
tags: [platform, domains, agents, dashboard, ui, registry]
created: 2026-05-16
relations:
  - target: hero-domains
    kind: parent
depends-on:
  - domain-plugin-architecture
  - domain-routing-and-agents
  - dashboard-view-registry
horizon: next
smoke: deferred
---

> **Status: awaiting primitives #1, #3, #4.** This stub is a
> `/design`-ready brief, not a complete design. The agent-side
> output-mode contract attaches to the active-domain agent loader
> (`domain-routing-and-agents`); the view-side accept/edit/reject
> control attaches to the view registry (`dashboard-view-registry`).
> Neither side can land until those primitives exist.

## Kickoff

Add a new agent output mode — `--inline-propose` — that targets the open artifact pane instead of writing to disk. The view layer renders the proposed content inline on the artifact (dotted/dashed border, "proposed by `<agent>`" badge) with accept / edit / reject affordances. Required by the locked Hero PM UX pattern (mockup `08-inline-proposal.html`); load-bearing for every PM authoring agent (`story-writer`, `prd-author`, `roadmap-curator`, etc.). Slotted as primitive #4b in the `hero-domains` initiative — sequenced after `dashboard-view-registry` but before `hero-pm` can ship.

**Status:** planning — stub written 2026-05-16. Blocked on primitives #1, #3, #4.

**Pick up at:** Run `/design inline-propose-output-mode`. First decision: where the proposed content lives between propose and accept — in the spec file as a marked draft section, in a sidecar file, or in transient view state only. Then the wire shape: what an agent emits when `--inline-propose` is active vs. the default write-to-disk mode.

→ `/design inline-propose-output-mode`

**Files:** .hero/planning/features/inline-propose-output-mode/spec.md, .hero/planning/initiatives/hero-domains/spec.md, .hero/planning/features/hero-pm/spec.md, .hero/planning/features/hero-pm/mockups/08-inline-proposal.html
**Skip:** Multi-author concurrent proposals on one artifact (single-author v1). Proposal persistence across sessions when not accepted (transient v1; revisit if PM users want a "saved drafts" tray). Cross-artifact proposals (proposal targets exactly one artifact).

## Goal

Provide a platform-level contract for agents to propose changes into
the artifact UI rather than write directly to the spec file. The
contract has two halves:

1. **Agent side** — a `--inline-propose` output mode that any agent
   can target. When active, the agent emits a structured proposal
   (target artifact, section/anchor, proposed content, optional
   rationale) instead of a file write.
2. **View side** — every artifact view in the dashboard view registry
   knows how to render proposals on the artifact pane (dotted-border
   block, badge, accept / edit / reject controls) and to commit or
   discard the change on user action.

The proposal is the trace; the artifact file on disk is the source
of truth. Only on **accept** does the proposed content land on disk;
**edit** opens the proposed content in the inline editor before
accept; **reject** drops it. The chat log records the lifecycle line
("`story-writer` drafted 4 AC → 3 accepted, 1 edited, 0 rejected")
but never carries the proposed content itself.

## Why now

The Hero PM domain pack (`hero-pm/spec.md`) locks an inline-propose
UX pattern as the canonical agent-output shape — agent-drafted AC
bullets, refined PRD sections, prioritization order changes, and
template proposals all appear *in the artifact*, marked proposed,
with accept/edit/reject. This is the locked design (see mockup
`08-inline-proposal.html`).

Engineering agents today only support write-to-disk. The PM agent
roster (~13 P0 agents per `agent-pack-design.md` §H) cannot ship
without this primitive — `story-writer`'s "Draft AC" button,
`prd-author`'s "Refine section" button, `prioritization-strategist`'s
"Bump up/down" buttons all assume inline-propose.

Inline-propose is also the right shape for engineering agents going
forward (e.g. `pr-reviewer` suggesting an AC fix inline on a feature
spec), so this primitive earns its place across both domains, not
just PM.

## Scope outline

The design pass should cover:

1. **Proposal wire shape.** The structured payload an agent emits
   under `--inline-propose`: target spec id, target section/anchor
   (frontmatter field, heading, list item, free position), proposed
   content (markdown or structured), rationale (optional), origin
   agent + skill chain. Versioned so future fields don't break
   parsers.

2. **Proposal storage between emit and accept.** Three candidates:
   - **Sidecar file** (e.g. `<spec>.proposals.json` next to the
     spec) — survives session restart, easy to clear.
   - **Marked draft section in the spec itself** — the proposed
     content lives in the file but inside a fenced "proposed by
     X" block that the parser knows to render distinctly and exclude
     from artifact-content reads.
   - **Transient session state** — proposals live in memory and the
     dashboard view's in-process state; lost on session restart.
   
   Pick one default; document the tradeoffs. v1 strawman: transient
   session state for simplicity, sidecar deferred until users want
   "saved drafts."

3. **Agent output contract.** How agents declare they support
   `--inline-propose`; what shape the agent's prompt produces (so a
   reviewer agent can also propose without bespoke wiring); how
   contextual buttons (per `agent-pack-design.md` §G) pass the
   `--inline-propose` flag through the command router.

4. **View-side rendering contract.** What artifact views must
   implement to surface proposals — the dotted-border block, the
   "proposed by `<agent>`" badge, the three affordances
   (accept / edit / reject), and the lifecycle log line that lands
   in the right-panel chat scroll. Goes into the view registry
   primitive (#4) as a required widget contract.

5. **Accept / edit / reject semantics.** What each action commits
   to disk, what the resulting spec edit looks like (idempotent
   apply; safe under concurrent reads), and what the chat log
   line says in each case.

6. **Multiple-proposal coordination.** When `story-writer` drafts
   4 AC bullets at once, the user can accept some and reject others.
   Each proposed bullet is an independent proposal; bulk accept /
   bulk reject affordances exist for ergonomics.

## Touchpoints (sketch — confirm during design)

- Agent loader / runner — where `--inline-propose` becomes a flag
  the agent honors. Likely lives in the same code path that
  domain-routing-and-agents (#3) refactors.
- Dashboard view registry — every PM and engineering artifact view
  gains a "proposal rendering" contract. Shared widget so individual
  views don't re-implement it.
- Spec parser — must tolerate the chosen storage shape (sidecar
  ignored, draft-block parsed-but-excluded, or no parser change if
  transient).
- Chat log / right-panel sticky region — the lifecycle log line
  format must align with the existing system-event style in
  `hero-pm/spec.md`'s "Surface roles" table.
- Command router — contextual buttons (per agent-pack-design §G)
  invoke commands with the inline-propose flag; the router must
  thread it through.

## Unknowns for design pass

1. **Storage default.** Sidecar vs marked-draft-block vs transient
   session state. The choice determines whether proposals survive a
   session restart, whether the spec parser changes, and whether
   `hero score` and lint see proposed content.
2. **Idempotency under repeat-propose.** If `story-writer` runs twice
   in a session and proposes overlapping AC, does the second run
   replace the first set of pending proposals or stack them? Likely
   replace per-section, but the rule needs to be explicit.
3. **Edit affordance fidelity.** "Edit" opens the proposed content
   for in-place editing before accept. Decide whether this is a
   single-line input, a markdown editor, or a passthrough to the
   artifact's regular editor with the proposed content pre-filled.
4. **Cross-domain propose.** Can an engineering agent propose into a
   PM artifact (e.g. `pr-reviewer` proposing a clarification on a
   PM story it's about to deliver)? Out of v1 scope likely, but the
   contract should not preclude it.
5. **Persistence of dismissed proposals.** Dismissed/rejected
   proposals do not re-appear the next time the user opens the
   artifact. Decide whether the dismissal is recorded (anywhere) or
   simply forgotten — affects the agent loop ("did I already propose
   this and get rejected?").
6. **Lifecycle log line format.** Aligning with the existing
   system-event style ("Status: drafted → refined") and the example
   in `hero-pm/spec.md` ("`story-writer` drafted 4 AC → 3 accepted,
   1 edited"). Pin the exact wording.

## Boundaries

- **Not** a generic "diff and review" surface for arbitrary file
  edits. Inline-propose targets *artifact specs* (`prd`, `story`,
  `epic`, `roadmap-item`, `intake-item`, engineering `feature`,
  etc.), not arbitrary source files.
- **Not** a code-review tool. Inline-propose on a PR is out of
  scope; `pr-reviewer` continues to write to disk for code edits.
- **Not** a multi-user collaboration surface. v1 assumes one author
  per session; concurrent author conflicts are deferred to a future
  Hero Cloud workflow.
- **Not** a replacement for write-to-disk mode. Agents continue to
  support write-to-disk as the default for non-interactive flows
  (`/deliver`, cron-driven scrubs, batch operations).

## Risks

- **Storage choice paints us into a corner.** If we ship transient
  state in v1 and users want saved drafts, retrofitting persistence
  means adding a sidecar after the fact. Acceptable, but call it out
  during design so the wire shape can absorb a `persisted: true`
  flag without breaking parsers.
- **View contract is per-artifact-view, not central.** Each artifact
  view in the registry must implement proposal rendering. If we
  scatter the implementation, behavior will drift (one view
  rejects-on-escape, another doesn't). Centralize in a shared
  widget; require all views to use it.
- **The accept action must be atomic and idempotent.** A proposal
  applied twice should not double-apply; a proposal applied while
  the user is editing the same section should not silently overwrite.
  Define the edit-conflict semantics in the design pass.
- **Contextual-button proliferation.** Every PM artifact gets 4–6
  inline-propose buttons (per agent-pack-design §G). The view
  registry must accept a button manifest without becoming a
  configuration swamp.
