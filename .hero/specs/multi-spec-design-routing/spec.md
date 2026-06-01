---
title: Multi-Spec Design Routing — Nudge from /design ×N to /compose
slug: multi-spec-design-routing
type: feature
domain: engineering
status: completed
size: small
priority: P2
tags: [design, compose, routing, ambient-context]
created: 2026-06-01
relations:
  - target: roadmap-shape
    kind: parent
  - target: roadmap-review
    kind: relates-to
completed_at: 2026-06-01T21:21:30Z
---

## Goal

When a `/design` conversation surfaces 2+ topically related deliverables
that would each warrant their own spec, the delivery lead pauses,
quotes the canonical multi-spec routing phrasing from the
`spec-composition` skill, and recommends `/compose` (initiative-first)
instead of scaffolding N flat siblings. The user can decline and the
lead proceeds with the requested sibling; the nudge is advisory, never
a block. This catches the dangling-spec anti-pattern at the moment of
creation — before four orphan features land and have to be reparented
later.

## Context

This is child #4 in the [`roadmap-shape`](../../initiatives/roadmap-shape/spec.md)
initiative. It rides on top of decisions already locked by sibling #1
([`roadmap-review`](../roadmap-review/spec.md)):

- **`spec-composition` is the canonical home** for the "multiple related
  specs" detection condition and the paste-ready conversational
  phrasing (Option C). Sibling #1 creates the skill file; this spec
  contributes a small Triggers section if #1 didn't land it.
- **The detection condition** ("2+ specs are topically related and have
  no initiative parent") is the same one #1's roadmap-reviewer agent
  uses on a workspace-wide sweep. This spec fires the same condition
  *inline during `/design`*, at the moment of spec creation, before
  the orphan cluster exists.

The two surfaces (this one, plus #1's interactive triage) are
complementary: this one **prevents** the orphan cluster from forming;
#1 **resolves** clusters that already exist. They should not
double-fire in a single session — see the suppression rule in
Approach.

The nudge is *routing-only*. It does not redesign `/compose`'s UX, it
does not add new initiative-detection heuristics beyond the two
documented triggers, and it does not change how `/design` scaffolds
an individual spec when the user declines.

## Approach

### Where the detection runs

Inline inside the `/design` command flow, *before* the delivery lead
writes any spec file. The lead loads `spec-composition` as part of
its design-time skill set (alongside `spec-format` and `spec-sizing`),
checks the trigger conditions against the active `/design`
conversation, and — if any trigger fires — quotes the routing phrasing
from the skill before proceeding.

No new MCP tools, no new CLI surface. The lead already has session
context; the trigger check is a small reasoning step at the top of
the design flow.

### The detection triggers (any-of)

The nudge fires when **any** of the following is true. Single trigger
is enough — they are deliberately not AND-ed, because each one
independently signals a composition moment:

1. **Explicit multi-deliverable phrasing in the user request.** The
   `/design` invocation names ≥ 2 distinct deliverables:
   - "design X, Y, and Z" / "spec out three things" / "I need
     specs for A and B" / "design these four features"
   - "and also" / "plus" / "along with" patterns when each side
     names a discrete deliverable
   The lead reads the request as written; no NLP, no learned model
   — pattern recognition the agent already does.

2. **Lead-identified sub-deliverables during conversation.** While
   clarifying the request, the lead identifies that the work has
   ≥ 2 independent sub-deliverables that would each warrant their
   own spec (different surfaces, different acceptance criteria,
   could ship independently). This is the lead's design judgment,
   not a mechanical rule.

3. **Rolled-up size ≥ `large`.** If the lead has enough information
   to estimate the deliverables' sizes individually, and the
   midpoint-sum of those sizes rolls up to `large` or higher on the
   shared ladder, the cluster wants an initiative parent regardless
   of phrasing. Reuses the same midpoint-sum mechanic from sibling
   spec `size-drift-actionable-output` (slice 3 of the parent
   initiative).

Any-of. One trigger is sufficient. The lead does not need all three.

### The conversational phrasing (lives in `spec-composition`)

The phrasing is canonical and owned by the `spec-composition` skill.
This spec does not re-author it — the lead quotes from the skill,
the same way it quotes the size-promotion nudges from `spec-sizing`.

Paste-ready text the skill will carry (contributed here if the skill
does not yet have a Triggers + Phrasing section when this spec
delivers):

> *"What you described would produce N related specs (`<slug-a>`,
> `<slug-b>`, `<slug-c>`). I'd recommend `/compose` instead — that
> creates an initiative with N child stubs, each ready for its own
> `/design` pass when you're ready to flesh it out. The alternative
> is N flat sibling specs with no parent, which usually gets
> reparented later anyway. Want me to `/compose` this, or proceed
> with the siblings here?"*

For the size-rollup trigger when phrasing didn't surface the
deliverables explicitly:

> *"This is shaping up as ≥ `large` rolled up across multiple
> deliverables. That's initiative territory — `/compose` will
> scaffold the parent and let you sequence the children. Want to
> switch, or stay in `/design` and ship this as a single spec?"*

### "User always wins" — match `spec-sizing`'s stance

The nudge is **advisory**, never a block. Same language and stance
as the sizing nudge:

- The lead surfaces the recommendation once per `/design` invocation.
- The user can answer "no, just do them as siblings" / "proceed" /
  "stay in design" / "ship it" — any clear decline — and the lead
  resumes the design flow as if the nudge had not fired.
- The lead records the user's choice in the design conversation so
  the rest of the session does not re-litigate it. Specifically: if
  the user declines and asks for N siblings, the lead writes N specs
  in that session and does not re-fire on each one.
- The nudge is **friction with an off-ramp**, not a gate. Document
  this stance explicitly in `spec-composition` so it matches
  `spec-sizing` word-for-word on the override.

### What the lead does on "yes, compose"

The lead pivots cleanly into the `/compose` workflow with the
user's original `/design` request as the initiative input. From the
user's perspective, it's the same outcome they'd have gotten by
typing `/compose <request>` directly:

1. The lead acknowledges the pivot: *"Switching to `/compose`. I'll
   scaffold an initiative parent with N child stubs, then we can
   `/design` each child individually when you're ready."*
2. The lead invokes the `/compose` flow (delegates to the compose
   command's existing logic; does not simulate or reimplement it).
3. `/compose` runs and produces the initiative + child stubs.
4. The design session ends; the user re-enters `/design <child-slug>`
   when they want to flesh out an individual child.

The lead does **not** simulate `/compose`. It hands off cleanly,
the same way sibling #1's roadmap-reviewer agent hands off
`/compose` / `/split` on confirm.

### Double-fire suppression

Three suppression rules prevent the nudge from re-firing at the
wrong moments:

1. **Already in `/compose`.** If the `/design` invocation was itself
   launched from a `/compose` flow (e.g., the user is now designing
   a specific child stub of a freshly-scaffolded initiative), the
   nudge does not fire. The composition decision has already been
   made.

2. **User explicitly opted into siblings.** If the user has already
   declined the nudge in this session (responded "no, siblings is
   fine" / "ship as siblings" / similar to a prior nudge), the lead
   suppresses re-firing for the remainder of the session. One
   decline covers the whole session.

3. **Don't double-fire with the sizing nudge.** The two nudges
   describe different moments:
   - **Sizing nudge** (`spec-sizing`): *"This individual spec is
     `large`/`x-large`/`giant` — consider `/split` or `/compose` for
     this spec specifically."*
   - **Routing nudge** (this spec): *"Your request describes ≥ 2
     specs — consider `/compose` to scaffold the cluster as an
     initiative."*

   They can both apply to the same conversation (e.g., the user
   describes three big deliverables — each is `x-large`, *and* the
   cluster is multi-spec). When both apply, the **routing nudge
   wins**: it fires first, and if the user accepts and pivots to
   `/compose`, the sizing nudges resurface naturally inside the
   per-child `/design` passes. If the user declines the routing
   nudge and proceeds with N siblings, the sizing nudge fires
   per-sibling as usual.

   Document the precedence in `spec-composition`: routing nudge
   precedes sizing nudge when both apply to the same `/design`
   request. The two stances stay consistent ("user always wins")
   but they don't both pile onto the user at once.

### Cross-reference shape

The skill `spec-composition` is owned by sibling #1
(`roadmap-review`). This spec's job is the routing-trigger
contribution + cross-references from the design surfaces. Concretely:

- **Cooperative ownership of the skill.** If sibling #1 has already
  added a `## Triggers` section to `spec-composition/SKILL.md` when
  this spec delivers, this spec extends that section with the
  any-of triggers above plus the conversational phrasing variants.
  If sibling #1 has *not* added the section, this spec creates it
  from scratch — the skill file exists either way (sibling #1
  creates the file), but the Triggers section is fair game for
  this spec to author.
- **The two delivery-lead agents** each get one line in their
  design-phase pre-flight directing them to load
  `spec-composition` alongside `spec-format` and `spec-sizing`,
  and to check the triggers before writing any individual spec.
- **`commands/design.md`** gets one short paragraph linking to
  `spec-composition`, parallel to its existing paragraph linking
  to `spec-sizing`.
- **`skills/spec-sizing/SKILL.md`** gets a small "see also" line
  in its "Composing with related skills" section pointing at
  `spec-composition` for the multi-spec case, so the two nudges
  are discoverable from each other.

## Acceptance Criteria

- WHEN a `/design` request explicitly names ≥ 2 distinct
  deliverables THE SYSTEM SHALL fire the multi-spec routing nudge
  before writing any spec file.
- WHEN the delivery lead identifies ≥ 2 independent sub-deliverables
  during a `/design` conversation THE SYSTEM SHALL fire the
  multi-spec routing nudge before writing any spec file.
- WHEN the rolled-up midpoint-sum size across the deliverables in a
  `/design` request reaches `large` or higher THE SYSTEM SHALL fire
  the multi-spec routing nudge before writing any spec file.
- WHEN the nudge fires THE SYSTEM SHALL quote the canonical
  conversational phrasing from the `spec-composition` skill rather
  than improvising wording.
- WHEN the user confirms `/compose` in response to the nudge THE
  SYSTEM SHALL pivot into the existing `/compose` flow with the
  user's original `/design` request as the initiative input, and
  SHALL NOT scaffold any flat sibling spec.
- WHEN the user declines the nudge THE SYSTEM SHALL proceed with the
  requested sibling specs in the same session, and SHALL suppress
  the multi-spec routing nudge for the remainder of the session.
- WHEN a `/design` invocation was launched from a parent `/compose`
  flow THE SYSTEM SHALL suppress the multi-spec routing nudge for
  that invocation.
- WHEN both the multi-spec routing nudge and a per-spec sizing nudge
  would fire on the same `/design` request THE SYSTEM SHALL fire the
  routing nudge first, and SHALL fire sizing nudges per-child only
  if the user declines routing and proceeds with siblings.
- IF the user explicitly opted into siblings in this session THEN
  THE SYSTEM SHALL NOT re-fire the multi-spec routing nudge on
  subsequent specs in the same session.
- IF the `spec-composition` skill does not yet contain a
  `## Triggers` section when this spec delivers THEN THE SYSTEM SHALL
  author it as part of delivery (cooperative ownership with sibling
  #1, which establishes the skill file).
- THE SYSTEM SHALL document the routing nudge as advisory in the
  `spec-composition` skill body, matching the "user always wins"
  stance used in `spec-sizing`.
- THE SYSTEM SHALL document, in `spec-composition`, the precedence
  rule that the routing nudge fires before the sizing nudge when
  both apply to the same `/design` request.
- THE SYSTEM SHALL update `domains/engineering/agents/feature-delivery-lead.md`
  and `domains/engineering/agents/platform-delivery-lead.md` to load
  the `spec-composition` skill in the design-phase pre-flight,
  alongside `spec-format` and `spec-sizing`.
- THE SYSTEM SHALL update `domains/engineering/commands/design.md`
  with a short paragraph linking to `spec-composition` for the
  multi-spec routing trigger, parallel to the existing
  `spec-sizing` reference.
- THE SYSTEM SHALL add a "see also" cross-reference from
  `domains/engineering/skills/spec-sizing/SKILL.md` to
  `spec-composition` so the two nudges are discoverable from each
  other.

## Changes

Canonical paths under `domains/engineering/`; harness directories
(`.claude/`) are views and update via existing sync.

1. **`domains/engineering/skills/spec-composition/SKILL.md`**
   (cooperative ownership — sibling #1 creates the file; this spec
   contributes the Triggers + Phrasing section if not already present)
   - Add (or extend) a `## Triggers` section listing the three
     any-of triggers from Approach: explicit multi-deliverable
     phrasing, lead-identified sub-deliverables, rolled-up size ≥
     `large`. Any-of, not AND.
   - Add the paste-ready conversational phrasing (both variants:
     deliverable-named and size-rollup) under a `## Phrasing`
     subsection.
   - Add a `## Stance` paragraph mirroring `spec-sizing`'s "user
     always wins" stance: advisory, never blocking, one decline
     covers the session.
   - Add a `## Precedence` paragraph documenting that the routing
     nudge fires before the sizing nudge when both apply to the
     same `/design` request, and that declining routing causes
     per-child sizing nudges to fire naturally inside the child
     `/design` passes.
   - Add a `## Suppression` paragraph documenting the three
     suppression rules: in-`/compose` invocations, post-decline
     session state, and the don't-double-fire-with-sizing precedence.

2. **`domains/engineering/agents/feature-delivery-lead.md`**
   - In the design-phase pre-flight (around the existing line
     loading `spec-format` and `spec-sizing`), add one line:
     "Also load the `spec-composition` skill — if the request
     matches the multi-spec trigger conditions, surface the
     routing nudge from the skill before writing any individual
     spec."
   - One-line addition; no restructure of the existing pre-flight.

3. **`domains/engineering/agents/platform-delivery-lead.md`**
   - Same one-line addition in its design-phase pre-flight.
     Keeps the two leads in sync.

4. **`domains/engineering/commands/design.md`**
   - Add one short paragraph (parallel to the existing
     `spec-sizing` paragraph) directing the design flow to load
     `spec-composition` and fire its routing nudge if the request
     names multiple deliverables or rolls up to ≥ `large`. Link to
     the skill.

5. **`domains/engineering/skills/spec-sizing/SKILL.md`** (small edit)
   - In the existing "Composing with related skills" section, add a
     bullet: "**`spec-composition`** — for the multi-spec case
     (user request names ≥ 2 related deliverables, or rolled-up
     size ≥ `large`), the routing nudge from `spec-composition`
     fires first; sizing nudges fire per-child after the user
     either pivots to `/compose` or declines and proceeds with
     siblings."

No code changes outside the canonical agent/command/skill markdown
files. No new MCP tools. No new CLI commands. The lead's reasoning
is the implementation surface.

## Boundaries

Explicitly out of scope:

- **Auto-running `/compose` without confirmation.** The nudge always
  asks. The user always wins.
- **Detecting orphan related-spec clusters retroactively.** That's
  sibling #1's (`roadmap-review`) job — it sweeps the planning
  corpus and surfaces clusters that already exist. This spec
  prevents clusters from forming at creation time; it does not
  scan for ones that already did.
- **Redesigning `/compose` UX.** The pivot hands off to the
  existing `/compose` flow as a black box.
- **New initiative-naming heuristics beyond what `/compose`
  already does.** When the lead pivots, it passes the user's
  original `/design` request as the initiative input and lets
  `/compose` handle naming and child-stub scaffolding.
- **Learned classifier / cross-session model for the triggers.**
  The triggers are explicit, mechanical, and in-session only.
  No training data, no embeddings, no cross-session signals.
- **Ambient surfacing of routing concerns between sessions.** That's
  sibling #2's (`roadmap-review-ambient-surfacing`) territory — it
  surfaces shape concerns in NEXT.md, hero_pulse, and the
  delivery-lead pre-flight. This spec fires inline during one
  `/design` call.
- **Routing from `/diagnose` to `/compose`.** Bug clusters are
  handled differently (they usually share a root cause, not a
  composition gap). This spec is `/design` only.

## Risks

- **Nudge fatigue.** If the trigger is too sensitive (especially
  trigger #2, lead-identified sub-deliverables), users will mute
  the channel. *Mitigations:* (a) the explicit-phrasing trigger
  (#1) and size-rollup trigger (#3) are mechanical and well-defined
  — they should not over-fire; (b) trigger #2 is design judgment,
  which the lead should apply conservatively (lean toward not
  firing if uncertain); (c) one-decline-covers-the-session
  suppression bounds the worst case; (d) the wording is short and
  paste-ready, so a fired nudge is a 5-second decision, not a
  conversation.

- **Double-firing with the sizing nudge.** Two related-but-distinct
  nudges on the same request. *Mitigation:* explicit precedence
  rule (routing first, sizing per-child after decline), documented
  in `spec-composition` and cross-referenced from `spec-sizing`.
  When both could fire, exactly one fires per moment.

- **Cooperative ownership of `spec-composition`.** Sibling #1 owns
  the file but this spec contributes the Triggers section. If
  delivery sequencing is misordered (this spec delivers before #1
  ships the skill file), the change list here covers creating the
  file from scratch — but the prioritization rules / canonical
  phrasing #1 carries would need to be re-stubbed. *Mitigation:*
  parent initiative's recommended delivery order ships #1 before
  this spec; cooperative cases are explicit in the change list.

- **User says "yes, compose" but `/compose` has a UX gap.** Hand-off
  failures are not this spec's problem to fix, but they'll feel
  like one to users. *Mitigation:* this spec hands off cleanly;
  any `/compose` issues that surface are tracked separately
  (the initiative explicitly excludes `/compose` UX redesign).

- **Trigger #3 requires size estimates before specs are written.**
  Estimating size for not-yet-written deliverables is rougher than
  estimating for an existing spec. *Mitigation:* the lead uses the
  midpoint-sum logic only when it has enough information to
  estimate each deliverable individually; if it can't, it falls
  back to triggers #1 and #2. Don't force a size-rollup when the
  data isn't there.

## Validation

- A fresh `/design` request that explicitly names three deliverables
  (e.g., *"design A, B, and C"*) fires the routing nudge, quotes the
  `spec-composition` phrasing, and offers `/compose`. On confirm, the
  lead pivots to `/compose` and does not scaffold flat siblings.
- The same scenario with the user declining ("no, siblings is fine")
  produces three flat sibling specs in the same session, and the
  routing nudge does not re-fire for subsequent specs in the
  session.
- A `/design` invocation launched from inside a `/compose` flow
  (designing a freshly-scaffolded child stub) does not fire the
  routing nudge.
- A `/design` request that names only one deliverable but the lead
  estimates rolled-up scope at `large` (because the single
  deliverable is wider than it looked) fires the routing nudge via
  trigger #3.
- A `/design` request where both routing and sizing nudges would
  apply fires routing first; the sizing nudges fire per-child only
  if the user declines routing.
- The two delivery-lead agents and `commands/design.md` reference
  `spec-composition` in their pre-flight; a fresh session running
  `/design` loads the skill before writing any spec.
- `skills/spec-sizing/SKILL.md` cross-references `spec-composition`
  in its "Composing with related skills" section.
- Cross-spec check: sibling #1's `roadmap-review` skill cross-
  references the same `spec-composition` triggers, and the two
  surfaces describe the same condition without divergent wording.

## Kickoff

`multi-spec-design-routing` — DELIVERED. 15 ACs DONE; SHIP / clean
audit. `spec-composition` skill extended cooperatively (Goal +
Canonical phrasing from #1 intact; Triggers placeholder replaced with
the real 3 any-of triggers; Stance consolidated in-place to avoid
duplication; Precedence + Suppression sections added). Both delivery
leads load `spec-composition` at design phase and fire the routing
nudge before writing any individual spec; `/design` command carries
the matching paragraph. `spec-sizing`'s "see also" entry refined to
document the precedence rule explicitly from the sizing side. Both
skills now agree word-for-word on routing-first / sizing-per-child.
Closes the prevention half of the roadmap-shape initiative.

→ Next: `/deliver roadmap-review-ambient-surfacing` — only #2 left.
