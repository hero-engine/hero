---
name: spec-composition
description: When work would produce 2+ related specs, prefer initiative-first composition over `/design` × N siblings. Carries the canonical "multiple related specs" phrasing that both interactive triage (`/roadmap-review`) and design-time routing (`/design`) quote.
compatibility: opencode
metadata:
  audience: any-agent
  purpose: composition-discipline
---

## What I do

Hold the canonical doctrine — and one specific paste-ready sentence —
for the **"multiple related specs"** condition: when the body of work
in front of the user is two or more topically related specs with no
shared initiative parent.

Two surfaces consume me:

- **`/roadmap-review`** (interactive triage) — surfaces orphan
  clusters that already exist in the planning corpus and offers to
  `/compose` them into an initiative.
- **`/design`** (multi-spec routing, sibling spec
  `multi-spec-design-routing`) — catches the condition *at design
  time*, before the second or third related spec lands, and routes
  the user toward `/compose` first.

Both surfaces quote the same canonical sentence from me. I exist so
the wording stays consistent across the two moments and neither
surface re-invents it.

## Goal

When two or more related specs would be created (or already exist
unparented), prefer **initiative-first composition** — `/compose` into
a parent with phased children — over scaffolding N flat sibling
features. Flat orphans become discoverable only by accident; an
initiative parent makes the work legible as a unit.

This is composition discipline, not sizing. A `medium` cluster of
three orphans is a composition problem; the size ladder doesn't model
it. That's why this skill exists separately from `spec-sizing`.

## Canonical phrasing

The sentence both surfaces quote verbatim, substituting the slugs:

> `<slug-a>`, `<slug-b>`, and `<slug-c>` look topically related and
> none of them have an initiative parent. If they're one body of
> work, `/compose` lifts them into a shared initiative. If they're
> genuinely independent, no action needed. Want me to `/compose`
> them?

Do not paraphrase. Quoting keeps the user's mental model of the nudge
stable across the design-time and triage-time surfaces.

## User always wins

The recommendation is advisory. If the user says "they're
independent," accept and proceed — don't re-ask in the same session,
don't escalate. The pattern mirrors the `spec-sizing` stance: the
nudge is a loud linter warning, not a gate.

If the user `/compose`s, the parent initiative scaffolds and the
existing children re-parent (or, at design time, the new specs land
under the new parent). If the user declines, the agent records the
choice and moves on. Either way, the system advances.

## Triggers

> Detection heuristics — what counts as "topically related" with
> enough confidence to fire this nudge — are owned by sibling spec
> `multi-spec-design-routing`. That delivery will populate this
> section with the concrete signals (tag overlap, title similarity,
> session co-creation window, etc.) and the threshold for firing.
> Until then, the `roadmap-review` skill's priority-4 rule ("2+
> related orphans without an initiative parent") is the operating
> definition.

When `multi-spec-design-routing` ships, it extends this skill with
`## Triggers`, `## Phrasing` refinements (if any), `## Stance`,
`## Precedence`, and `## Suppression` sections. This is cooperative
ownership — the v1 body lives here, the design-time-routing details
land via the sibling spec.

## Cross-skill references

- **`spec-sizing`** — the size ladder. This skill does not duplicate
  the per-type bands or the tier definitions; sizing remains the
  source of truth for "how big is this." Composition discipline is a
  separate axis.
- **`roadmap-review`** — interactive triage that fires the canonical
  phrasing above when priority-4 findings (orphan clusters) surface.
