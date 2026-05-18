---
title: Agent End-of-Turn Recap and Spin-Off Suggestions — Consistent Closing Behavior, Scope-Scaled
slug: agent-end-of-turn-recap
type: feature
status: planning
tags: [agents, agents-md, recap, handoff, behavior, spin-off]
created: 2026-04-30
relations:
  - target: agent-cold-start
    kind: extends
  - target: kickoff-prompts-queue
    kind: related
  - target: async-delivery
    kind: related
horizon: now
---

## Kickoff

Codify how the agent ends a turn so multi-step / spec-touching work always closes with a short status + concrete next-ask, while trivial Q&A stays terse. AGENTS.md gets the contract; a new `end-of-turn-recap` skill carries the format and worked examples.

**Status:** planning — design just finalized, no code changes yet.

**Pick up at:** add the "End-of-turn recap" section to `AGENTS.md` with the L0/L1/L2 rules, then the matching skill in `skills/end-of-turn-recap.md`.

→ `.hero/specs/agent-end-of-turn-recap/spec.md`

**Files:** [AGENTS.md](AGENTS.md), [skills/next-md.md](skills/next-md.md) (mirror format), [skills/kickoff-prompt.md](skills/kickoff-prompt.md) (sibling rule)

## Goal

Make every meaningful turn end with a status + concrete next-ask the user can act on, *without* turning trivial turns into checklist theater. Verbosity scales with scope: a one-line edit gets a one-line close; a multi-step spec-touching session gets a short recap with a runnable next move (and an optional spin-off offer if the next chunk wants its own session).

The user shouldn't have to ask *"where are we?"* or *"what's next?"* after a chunk of work. They also shouldn't have to wade through a green-checkbox table after asking a one-line question.

## Problem

Observed inconsistency in current behavior:

- **Multi-step turns sometimes end with zero status info.** A 6-tool-call turn that touches three specs and lands a commit can end with just *"Done."* — the user has to ask what shipped and what's next, burning a turn.
- **Trivial turns sometimes get a checkbox table.** A one-line clarification answer is followed by a ✓✓✓ summary and a "ready for me to do X next?" offer that doesn't fit the scope.
- **Recap shape is inconsistent.** Sometimes a markdown table, sometimes prose, sometimes a one-liner. The user's mental overhead reading the close is non-zero each time.
- **Verbosity doesn't scale to scope.** No principled rule for "how much recap" — it's whatever shape happens that turn.
- **No spin-off offer.** When the next concrete chunk would clearly benefit from a new session (different scope, heavy context, parallel track), the assistant rarely names that — the user has to know to ask *"should I spin a new session?"*.

These are all symptoms of *no rule*. AGENTS.md is the right place to codify one. The existing `next-md` skill already does this for *file* handoffs; this spec does the same thing for *response-message* handoffs.

## Design

### Three verbosity levels — L0 / L1 / L2

The agent picks one level per turn based on what happened, applies the matching format, and closes.

| Level | When | Format |
|---|---|---|
| **L0 — none** | Conversational turns, trivial Q&A, single-file reads, single short answer | State the result inline. No recap section, no checkboxes, no next-ask. |
| **L1 — one-liner** | Single tool call, narrow edit, focused fix | One sentence: what changed + (optional) the next obvious move in 5–10 words. |
| **L2 — short recap** | Multi-step (≥3 substantive tool calls), multi-file, spec-touching, status-flip, or chunk-landing turns | Two short blocks separated by a blank line:<br>**✓ Shipped:** 1–3 bullets, what materially changed, file paths as markdown links.<br>**⏭ Next:** 1–2 lines, concrete and runnable. Optional one-line spin-off offer. |

L2 is **hard-capped**. Never a wall of text. If the work needs more recap than fits, the answer is to break it across turns or offer a spin-off — not to write a longer recap.

### Trigger heuristics — picking the level

Applied in order; first match wins:

1. **Was the user's prompt a clarifying follow-up to a turn that already ended L2?** → L0 (anti-redundancy: don't recap the same state twice in a row).
2. **Did the turn produce 0 file mutations and ≤1 read-only tool calls?** → L0.
3. **Was a `spec.md` written/edited, a status field flipped, or a commit made?** → L2.
4. **Did the turn touch ≥3 files OR run ≥4 substantive tool calls?** → L2.
5. **Single tool call, single edit, single fix?** → L1.
6. **Default** when ambiguous → L1 (small over big).

"Substantive" excludes pure Read calls used for orientation. A turn that read 5 files and wrote 1 is L1, not L2 — the read calls were instrumental, not deliverables.

### Spin-off offer rule (L2 only)

At L2, the agent considers a spin-off offer. Add **at most one** offer line when ≥1 of these is true:

- **Different scope:** the next concrete chunk is parallel to (not a continuation of) the current work.
- **Heavy context:** the next chunk would benefit from a fresh slate (different files, different mental model, large surface).
- **Autonomous-friendly:** the next chunk has a complete spec and well-defined acceptance criteria — it can run headless without live oversight.
- **Recurring / scheduled:** the next chunk is a routine sweep or "do X in N weeks/days" follow-up.

The offer points at the right Hero-native target for the chunk's shape:

| Chunk shape | Offer target | Phrasing |
|---|---|---|
| Spec-tied, autonomous-friendly | `hero agent run deliver <slug>` (or `diagnose <slug>`) | `→ Want me to fire this headless via `hero agent run deliver <slug>`?` |
| Interactive resume in a new session | `hero_kickoff <slug>` from `kickoff-prompts-queue` | `→ Want me to drop a kickoff for <name> so you can spin a new session?` |
| Event-driven recurring | `hero agent automate` | `→ Worth wiring as a `hero agent automate` rule — say the word.` |
| Recurring on a clock | `/schedule` (only when harness supports it) | `→ Want me to `/schedule` this one?` (Claude Code only; falls back to `hero agent automate` otherwise) |

Notes:

- **Hero-native first.** `hero agent run` is harness-agnostic and ties directly into the spec corpus — prefer it over harness-specific scheduling for spec-tied work.
- **`/schedule` is harness-conditional.** Only offer when running under a harness that exposes it (Claude Code's scheduled-tasks). In other harnesses (opencode, plain), name `hero agent automate` instead.
- **Compose with `kickoff-prompts-queue`.** Interactive resume offers reference `hero_kickoff <slug>` so the user gets a real artifact, not a dumped chat prompt.
- **One offer per turn.** If multiple chunks could be spun off, name the highest-leverage one and stop.
- **Don't dump prompts inline.** Point at the artifact (`hero agent run …`, `hero_kickoff …`) — the user can run it themselves or ask you to.

### Anti-redundancy rule

If the prior assistant turn ended L2 and the current turn:

- only answered a clarifying question, OR
- only confirmed/declined an offer from the prior turn,

then close L0. The user already has the recap from one turn ago; repeating it is noise.

### Format examples

**L0 (correct):**

> The `pinned` field defaults to `false` if omitted. It's a regular YAML boolean.

**L1 (correct):**

> Renamed `kickoff_tuned` → removed entirely; spec now has `pinned` only. Want me to verify the parser doesn't choke on the old field?

**L2 (correct):**

> **✓ Shipped:**
> - Added `## Kickoff` section to [spec template](commands/design.md)
> - Updated [skills/kickoff-prompt.md](skills/kickoff-prompt.md) with format + worked examples
> - Re-indexed: 156 specs tracked
>
> **⏭ Next:** wire `hero queue` to read kickoff bodies — the selector is in place, just needs the renderer.
>
> → Want me to fire this headless via `hero agent run deliver kickoff-prompts-queue`?

**L2 anti-pattern (avoid):**

> ## Summary
>
> | Task | Status |
> |---|---|
> | Read spec | ✅ |
> | Update field | ✅ |
> | Run index | ✅ |
> | Verify result | ✅ |
>
> All four tasks have been successfully completed. The spec has been updated, the index has been rebuilt, and the changes are now live. The system is ready for the next phase of work. Please let me know how you would like to proceed.

The bad version is verbose, uses tables for low-information data, and ends in passive deference instead of a concrete next move.

### What's explicitly NOT shipping

- **No code-level enforcement.** This is a behavioral contract. The agent reads AGENTS.md + skill at session start; that's the entire delivery mechanism. No hook, no runtime check, no telemetry pipeline that gates responses.
- **No automatic level detection from prior conversation history.** The level is a function of *this turn's* work — heuristics are applied in-the-moment, not retroactively.
- **No mandatory spin-off offer.** L2 *may* offer; doesn't *must*. Many L2 turns have a natural inline next-step that doesn't warrant spinning off.
- **No new MCP tool, no CLI command.** Pure prose contract.

### Optional measurement layer (stretch goal)

If session transcripts are available (from `hero session` or harness logs), `hero check` could surface an advisory: *"in the last N sessions, M turns met L2 triggers but ended L0/L1 — recap rule is firing inconsistently."* Measurement only, no enforcement. Out of scope for v1; flag in Boundaries as a follow-up.

## Changes

- `AGENTS.md` — new top-level section "End-of-turn recap" containing: the three levels (L0/L1/L2), the trigger heuristics, the spin-off offer rule, the anti-redundancy rule, and one short example per level. Aim for 30–50 lines of AGENTS.md.
- `skills/end-of-turn-recap.md` — new skill mirroring `skills/next-md.md`. Contains: full format spec, expanded worked examples (good and anti-pattern for each level), quality bar, and the precise trigger checklist. The skill is the deep reference; AGENTS.md is the always-loaded summary.
- `skills/next-md.md` — add a one-line cross-reference: "For closing the turn message itself (vs. updating the file), see `end-of-turn-recap`."
- `commands/prime.md`, `commands/resume.md` — add a one-line note: "When you produce a recap at the end of priming, follow the `end-of-turn-recap` skill."
- `agents/feature-delivery-lead.md`, `agents/platform-delivery-lead.md` (if present) — add a one-line note: "Close the spec-authoring turn at L2 per `end-of-turn-recap`."

No Go changes. No CLI surface. No MCP tool.

## Acceptance Criteria

### AGENTS.md contract

- THE SYSTEM SHALL ship an "End-of-turn recap" section in `AGENTS.md` defining levels L0, L1, and L2 with one example each.
- THE SYSTEM SHALL ship a `skills/end-of-turn-recap.md` skill containing the full format spec, expanded examples, and the trigger checklist.
- THE SYSTEM SHALL keep the AGENTS.md section under 50 lines (deep reference lives in the skill).

### Level selection

- WHEN a turn produces zero file mutations and ≤1 read-only tool calls THE SYSTEM SHALL close at L0.
- WHEN a turn writes or edits a `spec.md`, flips a status field, or makes a commit THE SYSTEM SHALL close at L2.
- WHEN a turn touches ≥3 files OR runs ≥4 substantive tool calls THE SYSTEM SHALL close at L2.
- WHEN a turn does a single substantive edit or single fix THE SYSTEM SHALL close at L1.
- WHEN the prior assistant turn closed at L2 AND the current turn only answers a clarifying question or confirms/declines a prior offer THE SYSTEM SHALL close at L0 regardless of other signals.
- IF the level is ambiguous after applying the heuristics THEN THE SYSTEM SHALL prefer L1 over L2 (small over big).

### L2 format

- WHEN the level is L2 THE SYSTEM SHALL emit a "✓ Shipped" block with 1–3 bullets and a "⏭ Next" block with 1–2 concrete runnable lines.
- THE SYSTEM SHALL hard-cap an L2 recap at 12 lines of total user-facing text (excluding any spin-off offer line).
- THE SYSTEM SHALL render file references in L2 recaps as markdown links.
- IF an L2 turn would require more than the cap to recap THEN THE SYSTEM SHALL split the work across turns or offer a spin-off rather than expand the recap.

### Spin-off offer

- WHERE a turn closes at L2 AND the next concrete chunk fits a spin-off shape (different scope, heavy context, autonomous-friendly, or recurring) THE SYSTEM SHALL append at most one spin-off offer line.
- WHEN a spin-off offer targets autonomous spec execution THE SYSTEM SHALL reference `hero agent run deliver <slug>` (or `hero agent run diagnose <slug>` for bugs) as the suggested mechanism.
- WHEN a spin-off offer targets interactive resume in a new session THE SYSTEM SHALL reference `hero_kickoff <slug>` from `kickoff-prompts-queue`.
- WHEN a spin-off offer targets event-driven or recurring work THE SYSTEM SHALL prefer `hero agent automate` over harness-specific scheduling unless the harness's scheduler is the explicit fit.
- THE SYSTEM SHALL not stack multiple spin-off offers in a single turn.
- THE SYSTEM SHALL not dump long prompts inline as part of a spin-off offer — the offer references an artifact or command the user can run.

### Composition with existing rules

- THE SYSTEM SHALL not modify the NEXT.md update rule from `agent-cold-start` / `next-md` skill — those govern the *file* handoff; this spec governs the *message* close.
- THE SYSTEM SHALL not introduce a hook, runtime check, or code-level enforcement of recap behavior.

## Boundaries

- Does **not** add code, CLI commands, or MCP tools. Behavioral contract only.
- Does **not** retroactively recap state from prior turns — the level is a function of *this* turn's work.
- Does **not** mandate a spin-off offer at L2 — many L2 turns have a clean inline next-step.
- Does **not** modify NEXT.md update rules. File handoffs and message closes are independent surfaces.
- Does **not** ship a measurement / enforcement layer in v1. The "did the agent follow the rule?" question is observable manually; an advisory check in `hero check` is a follow-up if signal warrants it.
- Does **not** prescribe a tone or persona for recap content beyond format. Voice stays consistent with whatever the harness configuration sets.
- Does **not** apply to system-internal subagent invocations (Agent tool calls return raw tool output; closing-message rules apply only to user-facing turns).

## Mission Fit

> "Does this make the next agent session start smarter than the last one ended — and does it raise the floor for everyone, not just the senior dev who already knows what to ask?"

Yes — directly on the floor-raising axis.

A senior dogfooder *can* always ask *"what's next?"*. A non-expert often won't — they get a *"Done."* response and don't know whether to follow up, switch tracks, or call the work complete. The current inconsistency punishes them. Codifying L2 means *every* meaningful turn closes with a status + next-ask the user can act on without asking.

Smarter starts axis: the spin-off offer ties into `kickoff-prompts-queue`. When the agent says *"→ Want me to drop a kickoff for X?"*, it's actively producing the artifact that makes the next session start where this one ended — instead of leaving the user to guess what prompt to paste.
