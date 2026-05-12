---
title: Spec Prioritization — Now / Next / Someday / Parking-Lot
type: feature
status: planning
priority: P0
tags: [prioritization, signal-to-noise, anti-drowning, spec-lifecycle]
created: 2026-04-28
relations:
  - target: get-back-on-track
    kind: parent
  - target: spec-status-integrity
    kind: complements
mission_alignment: |
  The corpus must surface what's actionable now without losing what's
  captured for later. If `hero status` shows 78 specs and 60 of them
  are speculative future work (marketing, distribution, launch,
  positioning, hypothetical features), the now-actionable signal
  drowns. The mission requires sessions start *smart* — that means
  surfacing the right slice, not the whole archive. This feature is
  the prioritization mechanism that lets us "capture everything" and
  also "see only what's relevant now."
principles_check: |
  Serves #3 (sessions start omniscient on what matters now, not
  buried in someday-noise). Serves #1 (default views show now-work
  without flags). Risks #5 if the temporal segmentation requires
  manual upkeep on every spec; mitigated by sane defaults
  (`horizon: now` for newly-written specs; auto-demote when status
  changes to `parked` or after N days of inactivity).
horizon: next
smoke:
  script: scripts/smoke/spec-prioritization.sh
  expects: [spec-prioritization:AC-1, spec-prioritization:AC-2, spec-prioritization:AC-3]
  runs_on: [commit-touches:internal/spec/spec*.go, commit-touches:internal/cli/status*.go, commit-touches:internal/cli/validate*.go, nightly]
---

## Goal

Distinguish *actionable-now* from *captured-for-later* so future
plans don't drown current signal. Without losing the capture
discipline that makes sessions end smarter.

## Why now

The recovery audit and conversation surfaced the tension directly:

> *"we have captured marketing stuff - we have to think about it
> someday - but we don't want to imply its more important to work on
> now - do we need a mechanism to prioritize? its good to not lose
> stuff - but its bad when future plans become noise to the now"* —
> user, 2026-04-28 conversation

The hero workspace has 78 specs across `planning/features/` and
`planning/initiatives/`. Many are speculative — `hero-marketing`,
`hero-content-engine`, `hero-launch-playbook`, `hero-positioning`,
`hero-sales`, `hero-community`, `hero-distribution`,
`hero-landing-page`. They were captured (correctly) so the thinking
isn't lost. They show up in `hero status`, in dashboards, in
sprint-planning views — drowning the actionable now-work.

`status` (planning / delivering / completed / blocked) and `priority`
(P0/P1/P2) don't solve this. *Priority* is "how urgent within the
now"; *status* is "where in lifecycle." Neither answers *"is this
work for now, or is it captured for later?"*.

## The mechanism

A new frontmatter field: `horizon`. Four values:

| Horizon | Meaning | Default view |
|---|---|---|
| `now` | Active or imminent work. The set you'd talk about in standup. | Shown |
| `next` | Queued for the next phase/sprint. Concrete enough to commit to soon. | Shown |
| `someday` | Captured because the thinking is worth keeping. Not committing to time. | Hidden |
| `parking` | Explicitly deferred (e.g., dependent on a future capability or business condition). Reasoning preserved. | Hidden |

Default for new specs (created via `hero spec new`): `horizon: now`.

`status` and `priority` remain orthogonal — a spec can be `now` +
`P0` + `planning`, or `someday` + `P0` + `draft` (this would be a
killer-tier idea we're not ready to act on yet).

## Surface

### Default views are now-only

- `hero status` defaults to `horizon ∈ {now, next}` — the actionable
  set. One-line summary: `12 active, 47 someday/parking (hidden)`.
- `hero status --all` shows everything (escape hatch — principle #5).
- `hero status --horizon someday` shows only the parking lot.
- Dashboard "Open work" panels filter by `horizon ∈ {now, next}`.
- `hero blocked` (per `traversal-queries`) only includes
  now/next-horizon work — *blocked someday* isn't actionable.
- `hero suggest` and `hero relevant` weight now-horizon specs higher
  than someday/parking when ranking.

### Promotion / demotion

- `hero spec promote <slug>` — bumps horizon up (someday → next →
  now). Records who/when in the spec frontmatter.
- `hero spec park <slug> [--reason "..."]` — demotes to `parking`
  with required reason. Surfaced in the parking-lot view.
- Auto-demote after staleness: a `now`/`next` spec untouched for N
  days (default 60) prompts a demotion review during `hero check`.
  Doesn't auto-demote — surfaces the question.

### `hero check` enforcement

- New specs without `horizon:` field are rejected (per `hero check`).
  Default-fill on `hero spec new` makes this invisible to the user.
- Mismatch between `status: completed` and any horizon other than
  `now` flagged as inconsistent (a completed spec was, by
  definition, now-work).

### `hero status` parking-lot summary

```
Hero workspace
─────────────────────────
Now: 8 features, 2 initiatives  (3 delivering, 7 planning)
Next: 4 features                 (next-up after current)
Someday: 23                      (use --horizon someday to view)
Parking: 12                      (use --horizon parking to view)
```

Without this, our existing 78-spec workspace shows as "78 specs"
which is paralyzing. With it: "12 things to think about, 47 captured
for later."

## Acceptance criteria

**AC-1:** Adding `horizon: now|next|someday|parking` to spec
frontmatter parses cleanly. `hero scan` ingests the field. Verified
on a fresh spec.

**AC-2:** `hero status` default output shows only `now` + `next`
specs by default. Hidden ones are summarized but not enumerated.
Verified on the current hero workspace after horizon-tagging the
existing speculative specs.

**AC-3:** `hero status --all` and `hero status --horizon someday`
return the complete and parking views. Verified.

**AC-4:** `hero spec new <slug>` defaults to `horizon: now`.
Verified by inspecting the scaffold output.

**AC-5:** `hero check` rejects specs without `horizon:`. Verified
with a fixture missing the field.

**AC-6:** `hero spec promote` and `hero spec park` change horizon
with audit trail (who, when, optional reason in frontmatter).
Verified by running and re-reading.

**AC-7:** `hero blocked` only includes now/next-horizon work by
default. Verified.

**AC-8:** Bulk-tag-existing-specs migration (one-shot): `hero spec
horizon-migrate` proposes a horizon for every existing spec based on
heuristics (frontmatter status + age + tags) and writes a diff for
review. The audit's "speculative marketing/distribution" set lands
as `someday`; recovery features land as `now`; everything else
defaults to a reviewable inbox.

ACs accrete as edge cases surface (e.g., parent/child horizon
inheritance — does an initiative being `someday` automatically demote
its children?).

## Approach

**Phase 1 — frontmatter field + parser** (~½ day): Add `horizon` to
the spec parser, defaults, validation. Update spec template.

**Phase 2 — default views** (~½ day): `hero status`, `hero check`,
`hero blocked`, dashboard panel filters.

**Phase 3 — promote/park commands** (~½ day):
`hero spec promote|park` with audit trail.

**Phase 4 — staleness check** (~½ day): `hero check` surfaces
`now`/`next` specs untouched for >60 days, asks for demotion review.

**Phase 5 — bulk migration** (~½ day): One-shot
`hero spec horizon-migrate` for the existing corpus.

## Open questions

- Does `next` need its own staleness threshold, or share `now`'s?
  Lean: same threshold (60 days), same review prompt.
- Should `parking` require a `parked_until: <condition>` field?
  Lean: optional. A condition makes auto-promotion possible later
  (e.g., *"unparked when v2 corpus delivers"*) but adds friction.
- Inheritance: if an initiative is `someday`, are its children
  automatically `someday`? Lean: yes, with override. Initiative
  horizon caps child horizon.
- How does `horizon` interact with `mission_alignment` and
  `principles_check` fields (per `project-charter`)? Lean: they're
  required regardless; horizon is "when," alignment is "why this
  serves the mission at all."

## Out of scope

- LLM-proposed horizon classification — defer to manual + heuristics
  for v1
- Horizon-based dashboards (timeline view, parking-lot grid) — UI
  work, later
- Automatic promotion based on dependency completion ("X depended on
  Y; Y just shipped, auto-promote X to now") — interesting but defer
