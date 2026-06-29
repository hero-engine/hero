---
title: "Intake Capture Loop — Silently Capture Intent-Bearing Loose Asks, Manual Promote Gate"
type: feature
status: planning
priority: P2
domain: engineering
size: small
horizon: now
tags: [hero-core, intake, workflow, capture, claude-md, auto-capture]
relations:
  - { kind: follows, target_type: feature, target: hero-idea-primitive-core }
---

# Intake Capture Loop — Silently Capture Intent-Bearing Loose Asks, Manual Promote Gate

## Goal

When a loose one-off change carries intent a cold session would want — a *why*, a
small decision, a workaround — capture it as an `intake` so it doesn't evaporate.
Do it the way `auto_capture` already captures knowledge: **retroactively and
silently, only when it clears a threshold** — never gating the edit, never
capturing pure mechanical noise. The intent lands in a triageable surface; the
human stays the only gate (manual promote). This is the deliberately-light
workflow-loop follow-on to [hero-idea-primitive-core](../../intake/hero-idea-primitive-core/spec.md),
which already shipped the `intake` primitive (capture/triage/promote/reject,
pre-commitment exclusion). **Commit↔intake provenance is explicitly out of scope
here** (see Follow-ons) — the value is "the ask was written down," not graph
plumbing.

## Kickoff

Lightweight loop on the shipped `intake` primitive. Treat it as a sibling of
`auto_capture`: after a loose change lands, *if* the ask carried intent (a reason,
a decision, a workaround — not a typo/rename/format, not work already under a
spec), silently fire `hero intake "<ask + one-line why>"`. Reuse the existing
auto-capture threshold machinery and `hero intake` CLI — no new verbs, no graph
edges, no new hooks. Manual promote stays the gate. Land the trigger + threshold
on **all six install targets** (opencode/cursor/claude/copilot/codex/generic) via
the shared `.hero/knowledge/` convention rendered into AGENTS.md (+ CLAUDE.md for
Claude); make AGENTS.md guidance self-contained since only Claude has an
end-of-session hook. Decisions locked: retroactive + threshold-gated (NOT
capture-then-edit), capture auto / promote manual.

## Problem

The `intake` primitive can capture and triage, but **nothing fires it on the
inline path**, so the *intent* behind a small ask ("switch retry to backoff
because the tracker 429s") evaporates the moment the edit lands — exactly the
"stuff nobody told the next session" the mission exists to capture. The original
follow-on sketch overshot: "always capture before every edit" plus commit→intake
provenance edges is a burden (ceremony on every typo) and heavy (graph plumbing)
for a problem whose core is simply *the why wasn't recorded*. The right shape is
the one `auto_capture` already proves: silent, retroactive, threshold-gated.

## Approach

**Locked decisions:** capture **auto**, promote **manual**, capture is
**retroactive + threshold-gated** (reverses the earlier capture-then-edit idea —
always-capture is the burden the user explicitly wants to avoid). No altitude
classifier, no provenance edges, no new CLI in v1.

1. **Reuse the auto-capture pattern.** `auto_capture` (on by default in
   `hero.json`) already silently reviews a session for novel learnings and writes
   them to `.hero/knowledge/`. Add a sibling pass that, on the same trigger,
   recognizes an intent-bearing loose ask and writes it as an `intake` instead.
   Same silence, same "don't prompt," same threshold discipline — different sink
   (`intake` = triageable/promotable vs `knowledge` = reference).

2. **Threshold (the heart of "lightweight").** Capture only when the ask carries
   a *why* worth a future session knowing:
   - **Capture:** behavior/config change with a reason; a small decision/tradeoff
     ("X over Y here"); a workaround or gotcha; a deliberate "do it this way for
     now" scope cut.
   - **Skip (just edit):** typo, formatting, comment, mechanical rename; work
     already covered by an open spec/delivery; any change whose diff fully
     explains itself with no *why* beyond the code.

3. **Mechanism.** After the change lands, fire `hero intake "<verbatim ask +
   one-line rationale>"` (existing CLI, no new surface). Status `planning` until
   manual triage. Inherits pre-commitment exclusion, so freely-captured intakes
   never pollute queue/status/velocity/snapshot.

4. **Manual triage stays the gate.** `hero intake list` surfaces captured
   intakes; `promote`/`reject`/`merge` are manual and already shipped. No
   auto-promotion.

5. **Land it on every harness, not just Claude.** The trigger + threshold are
   authored once as a harness-agnostic `.hero/knowledge/` convention, then
   surfaced through Hero's existing install propagation so **all six supported
   targets** (`opencode | cursor | claude | copilot | codex | generic`) see it:
   - The guidance is rendered into the **canonical instruction surface**
     (`AGENTS.md` at root) — natively read by opencode, cursor, copilot, codex,
     and generic (and most of the ~9 AGENTS.md-native harnesses). See
     [[harness-instruction-file-survey]].
   - Claude Code (the AGENTS.md holdout) receives the same content via the
     `CLAUDE.md → AGENTS.md` symlink / `@AGENTS.md` import that `hero install`
     already wires. The one CLAUDE.md routing row is just the Claude-specific
     *view* of the shared convention, not a parallel source.
   - **Trigger asymmetry (must be designed for, not assumed away):** Claude Code
     fires the auto-capture pass reliably via its Stop/PreCompact hooks
     (`internal/cli/install.go` notes Claude "gets it for free"). The other five
     targets have **no equivalent end-of-session hook**, so for them the loop is
     driven purely by AGENTS.md instruction-file guidance — the model
     self-triggers the retroactive capture. Both paths consult the *same*
     threshold convention; only the trigger differs. The spec must not assume a
     hook exists outside Claude.

## Design / Data & State

- **No new spec type, no new CLI, no graph changes.** Rides entirely on the
  shipped `intake` type and `hero intake` verbs.
- **Capture content.** Intake body = verbatim ask + one-line rationale; status
  `planning`.
- **Rollup behavior unchanged.** Captured intakes inherit `IsPreCommitment`
  exclusion. The only new behavior is *when* `hero intake` fires (silently, on
  the auto-capture trigger, gated by the threshold).
- **Config.** Gate the behavior behind the existing `knowledge.auto_capture`
  switch (or a sibling `intake.auto_capture` defaulting to its value) so it's
  one toggle to turn the whole loop off.

### Harness coverage (delivery must satisfy all targets)

`hero install --target` supports `opencode | cursor | claude | copilot | codex |
generic`. The capture loop is delivered correctly only when every target can act
on it. The convention is authored once in `.hero/knowledge/`; propagation differs
by target:

| Target | Reads guidance via | End-of-session trigger |
|---|---|---|
| claude | `CLAUDE.md` (symlink/`@AGENTS.md` import) | **Hook** — Stop/PreCompact (reliable) |
| opencode | `AGENTS.md` (+ `opencode.json` `instructions` glob to `.hero/`) | Instruction-file self-trigger |
| cursor | `AGENTS.md` (Agent mode) | Instruction-file self-trigger |
| copilot | `AGENTS.md` | Instruction-file self-trigger |
| codex | `AGENTS.md` (+ `project_doc_fallback_filenames`) | Instruction-file self-trigger |
| generic | `AGENTS.md` | Instruction-file self-trigger |

Implication: the threshold guidance in `AGENTS.md` must be **self-contained and
imperative** ("after a change lands, if it carried intent, run `hero intake ...`")
so a harness with no hook still performs the capture from instruction alone. The
Claude hook is an enhancement, not a prerequisite. Do not gate any capture logic
on Claude-only mechanisms.

## Acceptance Criteria

- WHEN a session applies a loose change whose ask carries intent (a reason,
  decision, workaround, or deliberate scope cut) and that change is not already
  covered by an open spec THE SYSTEM SHALL silently capture an `intake` recording
  the ask and a one-line rationale, without prompting.
- IF a change is purely mechanical (typo, formatting, rename) or its diff is
  self-explanatory THEN THE SYSTEM SHALL NOT capture an intake.
- WHEN intake auto-capture fires THE SYSTEM SHALL NOT gate or delay the edit —
  capture is retroactive.
- THE SYSTEM SHALL NOT auto-promote a captured intake; promotion SHALL remain a
  manual `hero intake promote` action.
- WHILE a captured intake is un-promoted THE SYSTEM SHALL keep it out of every
  committed-work rollup (queue, status work buckets, velocity, snapshot) —
  inherited from the primitive.
- WHERE `knowledge.auto_capture` (or its intake sibling) is disabled THE SYSTEM
  SHALL NOT capture intakes automatically.
- WHEN `hero intake list` runs THE SYSTEM SHALL present captured-but-un-triaged
  intakes for manual promote/reject/merge.
- WHERE Hero is installed for any supported target (`opencode | cursor | claude |
  copilot | codex | generic`) THE SYSTEM SHALL deliver the capture trigger +
  threshold guidance to that harness via its instruction surface (`AGENTS.md`,
  and `CLAUDE.md` for Claude) — the convention SHALL NOT be Claude-only.
- WHERE a target has no end-of-session hook (every target except Claude Code)
  THE SYSTEM SHALL still drive capture from self-contained `AGENTS.md`
  instruction guidance; no capture logic SHALL depend on Claude-only Stop/
  PreCompact hooks.

## Test Plan

- **Threshold (capture)** — an intent-bearing loose-ask scenario fires one
  `hero intake` with the ask + rationale; assert the intake exists and `Discover`
  returns it as `TypeIntake`.
- **Threshold (skip)** — a typo/format/rename scenario, and a "covered by open
  spec" scenario, produce **no** intake.
- **Non-gating** — capture happens after the edit; the edit is never blocked
  (prompt-level / convention check).
- **Toggle** — with auto-capture disabled, no intake is created.
- **No-leak** — captured intake (status `planning`) absent from `hero queue`,
  `hero status` work buckets, snapshot, velocity; present in `hero intake list`
  (inherited; re-assert no regression).
- **Regression** — `intake` capture/promote/reject and all feature/bug/knowledge
  rollups unchanged; status/queue/snapshot goldens byte-identical.

## Changes

| File / area | Change | Est. |
|---|---|---|
| `.hero/knowledge/` (new convention) | Harness-agnostic "intake capture loop" trigger + threshold — the authoritative source, authored once | S |
| Canonical instruction surface (`AGENTS.md` render + the `.hero/` source it renders from) | Self-contained, imperative capture guidance so every AGENTS.md-native target (opencode/cursor/copilot/codex/generic) self-triggers capture without a hook | S |
| `CLAUDE.md` (routing table) | One row mirroring the shared convention — the Claude-specific view, fed by the `CLAUDE.md → AGENTS.md` link `hero install` already wires (not a parallel source) | S |
| `auto-knowledge-capture` skill / auto-capture pass | Extend the existing silent post-session pass to also recognize intent-bearing asks and route them to `hero intake`; honor the threshold. On Claude this rides the Stop/PreCompact hook; elsewhere it is the instruction-driven path | M |
| Install propagation (`internal/cli/install.go`) | Confirm the new guidance flows to **all six targets** via existing AGENTS.md + CLAUDE.md propagation; no target left without the convention. (Relates to bug `install-target-emits-both-claude-and-agents-md`.) | S |
| `hero.json` config | Reuse `knowledge.auto_capture` (or add sibling `intake.auto_capture` defaulting to it) as the single off switch | S |
| Tests | threshold capture/skip + toggle + non-gating + no-leak + regression + **per-target propagation** (guidance present in each target's instruction surface) | M |

## Risks

1. **Threshold too loose → intake spam.** *Mitigation:* default conservative
   (capture only with a clear *why*); pre-commitment exclusion keeps spam out of
   all work rollups; batch-reject in triage is cheap; the convention errs toward
   skip.
2. **Threshold too tight → misses real intent.** *Mitigation:* the bar is
   "would a cold session want this *why*?"; tune the convention from real usage.
   Retroactive-distill (Follow-ons) can backfill later.
3. **Harness adherence / trigger asymmetry.** Only Claude Code has a reliable
   end-of-session hook (Stop/PreCompact); the other five targets self-trigger
   from `AGENTS.md` guidance, which a model may skip. *Mitigation:* author the
   AGENTS.md guidance as self-contained and imperative so non-hook harnesses can
   act on it alone; the `.hero/knowledge/` convention is the shared backstop;
   verify per-target propagation in tests. Accept that non-Claude capture is
   best-effort — retroactive distill (Follow-ons) can backfill misses.
6. **Claude-only assumptions leak into core.** Building capture on the
   Stop/PreCompact hook would silently exclude five of six targets. *Mitigation:*
   the AC forbids gating capture logic on Claude-only hooks; the hook is an
   enhancement layered over the instruction-driven path, not the mechanism.
4. **Double-capture on promote.** Promote must reuse the captured intake.
   *Mitigation:* promote mutates the existing intake in place (inherited);
   regression test asserts no duplicate.

## Follow-ons (not in this spec)

- **Commit↔intake provenance** — link captured intakes to the commits they caused
  so `hero why <sha>` traces back even un-promoted. Deliberately deferred: heavy
  (graph edges, `hero_why` traversal, pre-commit stamping) and a nice-to-have, not
  the core "the ask was written down" value. Turn on only if the traceback is
  actually wanted.
- **Altitude classifier / auto-promote** — deferred by decision (manual promote
  chosen). Revisit only if triage volume proves unsustainable.
- **Retroactive distill from diff** — synthesize the intake from the diff for
  sessions where the ask wasn't explicit. Complementary backfill path.
