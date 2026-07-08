---
name: handoff-protocol
description: The formal protocol for handing a spec off from PM to engineering — pre-flight gates, the `owner: pm → engineering` flip on the same artifact, the bitemporal ownership history that is the cross-domain edge, and the hand-back path.
metadata:
  audience: handoff-coordinator, pm-reviewer, pm-delivery-lead
  purpose: cross-domain
---

## What I do

Provide the formal protocol for the PM → engineering handoff under the
**unified type model**. Under the unified model, a handoff is *not* a
tracker copy, *not* a vibes pass, and *not* a fresh engineering-spec
creation. It is an **owner flip on the same artifact**: the spec PM
authored and refined carries through into engineering's delivery,
unchanged in identity, with `owner` flipped from `pm` to `engineering`
and the transition recorded bitemporally.

This skill defines the pre-flight gates that prevent premature handoff,
the owner-flip mechanics, the bitemporal ownership history that serves
as the cross-domain edge, the failure modes when engineering can't
accept, and the hand-back semantics for revisions.

## When to use me

Load this skill when:

- `/handoff` is invoked on a spec (`handoff-coordinator`)
- "Hand off to engineering" contextual button fires on a Spec detail
- `pm-reviewer` is auditing a spec for handoff-readiness
- `pm-delivery-lead` is orchestrating a refinement → handoff sequence
- a hand-back is observed (engineering flipped `owner` back to `pm`
  with `handed_back_reason`)

## What changed from legacy

Old workflow (deprecated):

1. PM authors a "story" type.
2. `handoff-coordinator` shapes a packet, calls `/design` on the
   engineering side.
3. `feature-delivery-lead` authors a separate `feature` spec.
4. A `kind: handoff` graph edge is written linking the two specs.
5. The story detail's "linked engineering feature" rail is populated.

New workflow (current):

1. PM authors a feature (`type: feature`).
2. `handoff-coordinator` pre-flights the spec and flips `owner: pm →
   engineering` on the same spec.
3. Engineering's `engineer` agent picks up the spec via `/deliver
   <slug>` (engineering pack).
4. The `owner_history` (bitemporal) is the cross-domain edge — no
   separate `kind: handoff` edge is written.
5. There is no "linked engineering feature" rail — it's the same spec.

If you find this skill referring to `/design`, `feature-delivery-lead`,
or a `kind: handoff` graph edge, the reference is legacy and should be
updated. The owner flip *is* the handoff.

## Pre-flight checks

The owner flip aborts (with a specific error per failure) if any
**blocking** pre-flight check fails. Warnings (missing PRD, missing
initiative rationale) surface but do not block — some specs ship
without strategic framing legitimately.

### Blocking gates

1. **Status is `in-review`.** Not `planning` (per the lifecycle table
   in `pm-preset-detection`). The `pm-reviewer`
   gate must have passed. An `in-review` status means INVEST was checked,
   AC was written in EARS, the preset's required fields are populated,
   and Out-of-Scope is explicit.

2. **`owner` is currently `pm`.** If `owner` is already `engineering`,
   the handoff has already happened. Read `owner_history`; if the
   most recent transition is `pm → engineering`, surface and stop. If
   the engineering side handed back (`engineering → pm` with a
   `handed_back_reason`), the spec is yours again — but address the
   hand-back reason before flipping again.

3. **Acceptance Criteria exists in EARS format.** Empty AC is
   undeliverable. AC in freeform prose without EARS structure is
   acceptable only when the criteria genuinely don't fit any pattern;
   the default is EARS (per `acceptance-criteria-ears`).

4. **Out-of-Scope is populated.** An empty Out-of-Scope after `/refine`
   usually means scope is implicit and engineering will guess. The
   spec needs explicit no-gos.

### Warning gates (surface, don't block)

5. **Linked PRD context where applicable.** If the spec is part of a
   larger PRD-scoped effort, the `prd:` frontmatter ref should be set.
   Absence is acceptable for small bugs and defect-driven changes —
   warn but don't block.

6. **Linked initiative with rationale.** For specs that map to a
   strategic bet, the `initiative:` ref or parent epic should reach
   an initiative with populated `rationale`. Engineering benefits
   from knowing the strategic context to make sensible implementation
   tradeoffs. Absence is acceptable for engineering-originated work
   (e.g. `chore`).

If any blocking check fails, `handoff-coordinator` halts and emits an
actionable error pointing at the missing field. Do not proceed with a
malformed spec "to get unblocked"; the spec is the artifact engineering
delivers from, and ambiguity at the flip becomes drift downstream.

## The owner flip

This is the load-bearing step. `handoff-coordinator` flips owner with
`hero spec set-owner <slug> engineering`:

```
hero spec set-owner <slug> engineering
# owner: pm → engineering; status stays in-review until engineering's
# claim flips it to `delivering` (per the lifecycle table in
# `pm-preset-detection`)
```

`set-owner` appends the `owner_history` row atomically; a raw
frontmatter edit records **no** history, so always use the command.
The history entry looks like:

```yaml
owner_history:
  - { from: null,         to: pm,          at: 2026-05-01T09:12:00Z }
  - { from: pm,           to: engineering, at: 2026-05-17T14:33:00Z }
```

The bitemporal history *is* the cross-domain edge. The Cross-domain
Handoff stream queries `Spec` nodes for `owner_history` rows where the
transition crosses domains; the stream row is sourced from the history,
not from a dedicated edge kind.

## What does NOT happen at the flip

The unified model removes several legacy steps. None of these run:

- **No `/design` call** on the engineering side.
- **No separate engineering `feature` spec** is created — there is
  one `feature` artifact, and it is shared across the owner flip.
- **No `kind: handoff` graph edge** is written. The `owner_history`
  bitemporal rows serve the same purpose.
- **No "linked engineering feature" rail** appears on the spec detail
  page — there is no second artifact to link to.
- **No handoff packet** is shaped or paraphrased — the spec itself is
  the packet. Engineering reads the same Description, AC, Out-of-Scope,
  linked PRD, and linked initiative that PM authored.

## Verifying the cross-domain transition

The owner flip is the handoff. `handoff-coordinator` verifies, in order:

1. The frontmatter write succeeded — `owner: engineering` is on disk.
2. The bitemporal `owner_history` row landed — `from: pm, to:
   engineering, at: <ts>` is present.
3. The Cross-domain Handoff stream picks up the transition (the event
   log row from `hero agent events spec_updated ...` is what the dashboard polls;
   the bitemporal history is the source of truth).
4. A read-back of the spec shows `owner: engineering` on disk (`hero
   list --status in-review` as the sweep before engineering claims).
5. Within a short window, engineering's `engineer` agent claims the
   spec via `/deliver <slug>` (engineering pack); status flips `in-review → delivering`.

If verification step 5 doesn't happen within the expected window,
surface as a finding — engineering may be offline; the user can
manually invoke `/deliver` (engineering pack) to push the pickup. Do not retry the owner
flip; it's already done.

## Hand-back path

Engineering can flip `owner` back to `pm` with a frontmatter
`handed_back_reason:` field set, when refinement reveals an
under-specified requirement. When this happens:

1. The spec's `owner:` is `pm` again (read it back).
2. `handoff-coordinator` reads the spec on next pre-flight; sees
   `owner: pm` and a `handed_back_reason:`; routes to `pm-reviewer`
   or `story-writer` to address the gap.
3. After the gap is addressed and `pm-reviewer` passes again, the
   flip runs again. The `owner_history` records both transitions:

```yaml
owner_history:
  - { from: null,         to: pm,          at: 2026-05-01T09:12:00Z }
  - { from: pm,           to: engineering, at: 2026-05-17T14:33:00Z }
  - { from: engineering,  to: pm,          at: 2026-05-17T16:02:00Z,
      handed_back_reason: "AC ambiguous on retry semantics — pick one." }
  - { from: pm,           to: engineering, at: 2026-05-18T10:14:00Z }
```

The history preserves the back-and-forth; the cross-domain stream
shows both directions; nothing is overwritten.

## Failure modes

### Engineering can't claim the spec

The owner flip succeeded but engineering doesn't pick the spec up
within the expected window. Possible causes:

- **Engineering is offline.** Surface; the user can invoke `/deliver
  <slug>` (engineering pack) manually to force the pickup.
- **No engineer agent available.** Same resolution.
- **Workspace doesn't have an engineering domain configured.** This is
  a PM-only workspace — the owner-flip workflow doesn't make sense.
  Surface the configuration issue.

The owner flip itself succeeded — the spec is engineering-owned. The
follow-on claim is asynchronous; missing claim is a workflow gap, not
a handoff failure.

### Engineering hands back with `handed_back_reason`

See the Hand-back path above. The spec returns to PM with a reason;
PM addresses the reason and re-flips.

### Frontmatter write or history append fails

If the disk write fails, halt and surface. The handoff did not happen
— the user needs to know. Do not silently retry.

## Anti-patterns

- **Calling `/design` on the engineering side.** Legacy workflow. No
  longer applies.
- **Creating a new engineering spec.** No second spec exists under the
  unified type model. Wrong shape.
- **Writing a `kind: handoff` graph edge.** The `owner_history`
  bitemporal rows are the edge. A separate edge duplicates state.
- **Paraphrasing the spec into a "packet".** The spec is the packet.
  Engineering reads the same Description, AC, Out-of-Scope that PM
  authored.
- **Flipping owner without the pre-flight.** Premature handoff
  produces a delivery the engineer can't act on cleanly.
- **Flipping owner on a spec that's already engineering-owned.**
  Either the handoff already happened (surface and stop) or the spec
  was never PM-led to begin with (engineering-originated; no PM
  handoff needed).
- **Silently retrying on flip failure.** Halt and surface. The user
  needs to know.
- **Deleting the original `owner_history` row on re-handoff.** History
  matters. The store is append-only; the bitemporal rows preserve the
  full sequence of transitions.
- **Treating tracker status as the handoff signal.** The owner flip
  in Hero is the handoff; the tracker is org-state propagation.

## Relationship to engineering's engineer agent

The boundary is hard, but lighter than before:

- `handoff-coordinator` does PM-side work: pre-flight, flip, verify.
  It does **not** author engineering content, does **not** invoke
  engineering's agents directly, does **not** edit the spec's
  implementation sections.
- Engineering's `engineer` (or `feature-delivery-lead`) authors
  `plan.md` as a companion artifact in the *same spec folder*, owns
  implementation framing, runs engineering review gates, and
  ultimately decides whether the spec is acceptable as scoped.

If `handoff-coordinator` finds itself writing into the spec's
implementation sections or `plan.md`, the boundary has been crossed.
Stop, let engineering own its side. The spec is shared; the
companion artifacts are domain-scoped.
