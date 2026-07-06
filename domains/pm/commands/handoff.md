---
core_fork: PM handoff is an owner-flip to engineering on the same artifact, intentionally replacing core's session-handoff /handoff
description: Hand a refined spec off to engineering — flips `owner: pm → engineering` on the same artifact. No new spec is created.
---
Route this handoff to the `handoff-coordinator` agent. Loads the `handoff-protocol` skill.

**This is the brand command.** Under the unified type model, the PM → engineering handoff is an *owner flip on the same spec* — not a fresh spec creation, not a `/design` call on the engineering side, not a separate graph edge. The cross-domain transition is recorded bitemporally in the spec's `owner_history`, and that history *is* the cross-domain edge.

## Required argument

A spec slug. Without it, the command asks which spec to hand off (don't guess from session context — the boundary crossing is too consequential for a wrong guess).

## Pre-flight

Before the coordinator flips owner, verify the spec:

1. **`status: in-review`.** Specs at `planning` aren't shippable (per the lifecycle table in `pm-preset-detection`). → If not ready, ask the user to run `/refine` first and stop.
2. **`owner: pm` currently.** If `owner` is already `engineering`, the handoff already happened — surface and stop. (Use `hero why <slug>` to inspect the `owner_history`.)
3. **Acceptance criteria use EARS patterns** (or have a documented reason not to). → If AC is missing or freeform-only without rationale, ask the user to run `/refine --section ac` first and stop.
4. **`Out-of-Scope` section is non-empty.** → If empty, ask the user to fill it via `/refine --section out-of-scope` first and stop.
5. **A linked PRD exists.** → If absent, **warn the user but do not block.** Some specs ship without a parent PRD (small tweaks, defect-driven changes). The warning is recorded in the event log so engineering sees the gap.
6. **A linked `initiative` exists with a rationale.** → If absent, **warn but do not block.** Same logic — surface, don't gate. Engineering-originated specs (`chore`, etc.) legitimately lack strategic framing.

## Workflow

The `handoff-coordinator`:

1. **Reads the spec** — resolve the slug (`hero_read_spec` MCP or `hero search --list`); pm specs live under `.hero/planning/{features,bugs,epics,prds,intake}/<slug>/spec.md` — and runs the pre-flight above.
2. **Flips the `owner` field** via `hero spec set-owner <slug> engineering` (`pm → engineering`). The command appends the bitemporal `owner_history` row atomically — a raw frontmatter edit records no history. **This is the load-bearing step.**
3. **Verifies the history landed** — reads back the `owner_history`; if the new row isn't present, that's a hard error (see Failure modes).
4. **Logs the transition** via `hero agent events spec_updated "owner flipped pm → engineering" --slug <spec-slug>` (or the `hero_event` MCP tool) so the Cross-domain Handoff stream picks it up.
5. **Verifies engineering pickup** — within a short window, re-read the spec (`owner: engineering` on disk); after engineering claims (`/deliver <spec-slug>`, engineering pack), the spec's `status` flips `in-review → delivering` (`hero list --status delivering` as the sweep). If pickup doesn't happen, surface as a finding (engineering may be offline; user can invoke `/deliver` manually).

What does **not** happen:

- No call to `/design` on the engineering side.
- No new engineering spec is created — the same spec carries through.
- No separate `kind: handoff` graph edge is written — the `owner_history` is the edge.
- No "linked engineering feature" rail is populated on the spec detail page — there is nothing to link to (it's the same spec).
- No handoff packet is paraphrased — the spec itself is the packet.

## Failure modes

- **Pre-flight fails (status, owner, AC, Out-of-Scope)** → surface what's missing, ask the user to refine, stop. Do not flip with an incomplete spec.
- **Spec is already engineering-owned** → surface `owner_history`; if the most recent transition is `pm → engineering`, the handoff already happened; if it's `engineering → pm` with a `handed_back_reason`, address the reason via `/refine` before re-flipping.
- **Frontmatter write fails** → hard error. The handoff did not happen. Surface and stop. Do not retry silently.
- **`owner_history` row does not appear after write** → hard error. The store invariant is broken; do not proceed.
- **Engineering doesn't claim within the expected window** → surface as a finding (engineering may be offline). The owner flip succeeded; the missing pickup is a workflow gap, not a handoff failure. User can invoke `/deliver <slug>` (engineering pack) manually to force the claim.

## Output

- The same spec (resolve the slug; pm specs live under `.hero/planning/{features,bugs,epics,prds,intake}/<slug>/spec.md`), now with `owner: engineering` and a new `owner_history` row.
- A one-line log: `handoff <slug>: owner flipped pm → engineering @ <timestamp>; engineering queue updated.`

## Contextual invocation

This command also fires from the **Hand off to engineering** button on a spec detail view. Button invocation always uses inline confirmation — never silent flip. The pre-flight check results are shown before the owner field is mutated.

After a successful handoff, if `knowledge.auto_capture` is on, evaluate whether the session produced a learning worth persisting (a recurring spec-shape issue worth a convention, a pre-flight failure pattern worth surfacing) and write it to `.hero/knowledge/notes/`.

Request: $ARGUMENTS
