---
title: "Delivery Completion Discipline — Stop Agents from Silently Narrowing Scope, Punting Hard Items, or Yielding Early"
slug: delivery-completion-discipline
type: feature
status: completed
priority: high
created: 2026-05-22
domain: engineering
tags: [agent-reliability, deliver, engineer, instructions, completion]
relates-to: [premise-interrogation, agent-reliability, next-as-projection]
completed_at: 2026-05-22T19:19:31Z
---

# Delivery Completion Discipline — Stop Agents from Silently Narrowing Scope, Punting Hard Items, or Yielding Early

## Context

Across many delivery cycles in this and adjacent projects, the user keeps ending
up with "a useless pile of shit after going through spec and deliver." The
specs look good. The delivery output does not match the spec. Two failure modes
recur:

**Failure mode A — silent scope narrowing during `/deliver`.** The engineer
agent reads "build feature X" as "render the UI for X" and stops. Decides hard
pieces are "beyond scope" and drops them. Picks the easiest plausible reading
of an ambiguous requirement without naming the ambiguity. Marks
`status: completed` when significant acceptance criteria are unaddressed.

**Failure mode B — premature yielding on continuous tasks.** When the user
asks an agent to do something "until complete," the agent does one small slice
and stops. User asks "why did you stop?" — agent says "my bad, I shouldn't
have." User says "keep going" — agent does another small slice and stops
again. Each individual stop has a plausible-sounding micro-reason but the
pattern is the failure.

Both modes share one root cause: **the current instructions defend hard
against agents doing TOO MUCH and only weakly against agents doing TOO
LITTLE.** Scope discipline, smallest-change rules, boundaries, and
don't-refactor-nearby are repeated across [implementation-principles](domains/engineering/skills/implementation-principles/SKILL.md), [engineer.md](domains/engineering/agents/engineer.md), and [agent-reliability](domains/engineering/skills/agent-reliability/SKILL.md). There is no
symmetric defense against under-delivering. Asymmetric guardrails produce
asymmetric failure.

This is a meta-spec: the changes apply to Hero's own engineering instructions
under `domains/engineering/`, not to a user-product feature. The mission fit
is direct: every Hero session is supposed to start smarter than the last one
ended, and that depends on each delivery cycle actually completing what was
agreed. A workspace full of half-finished specs raises the floor for no one.

### Specific leak points in current instructions

- [implementation-principles/SKILL.md:17](domains/engineering/skills/implementation-principles/SKILL.md:17) — "smallest correct change that satisfies the requirement" gets read as "smallest plausible change that compiles."
- [engineer.md:49](domains/engineering/agents/engineer.md:49) — "respect the Boundaries section — do not go beyond scope" — used as an escape hatch. Agents reframe required items as out-of-scope.
- [engineer.md:51](domains/engineering/agents/engineer.md:51) — "if unclear, stop and explain" covers admitted confusion but not silent reinterpretation.
- [deliver.md:97-115](domains/engineering/commands/deliver.md:97) — post-delivery loop checks drift, test coverage, kickoff. No per-acceptance-criterion accounting. `hero drift` checks file lists, not whether the feature works.
- [engineer.md:101-106](domains/engineering/agents/engineer.md:101) — default output lets the engineer write a soft "implementation summary." Easy to gloss skipped items.
- Nothing addresses persistence on continuous tasks across `agent-reliability` or `engineer.md`.

## Goal

Tighten the engineering instruction set so Hero agents deliver the **kick-ass
version** of every spec — not the half-assed minimum that satisfies the letter
of the requirements while missing the point. After this spec lands, a fresh
agent reading the engineer/delivery-lead/skill files should encounter:

1. An explicit **excellence bar** — "make this great, not adequate" — alongside
   the existing scope-discipline rules. The current instructions are defensive
   ("don't do too much"). This spec adds the offensive direction ("do it
   excellently") that's currently missing.
2. **Symmetric guardrails against under-delivery** alongside the existing
   guardrails against over-delivery — agents will not declare delivery complete
   when acceptance criteria are unaddressed, will not silently reinterpret
   ambiguous scope to its easier reading, and will not yield control mid-task
   without a true blocker.
3. A concrete end-of-delivery **Completion Ledger** the engineer must produce,
   and a delivery-lead refusal to mark `status: completed` if the ledger has
   gaps without explicit sign-off.

The quality bar is **outcome excellence**, not extra scope. Doing the agreed
thing exceptionally well beats doing more things adequately.

## Kickoff

Adds an **Excellence Bar** ("kick-ass, not half-assed") plus
anti-under-delivery rules to Hero's engineer/delivery-lead instructions so
agents aim for the excellent version of every spec, stop punting on hard
parts, stop silently narrowing scope, and stop yielding before continuous
tasks are done. Pure docs change — edited five markdown files. Change #6
(optional spec-format worked example) intentionally skipped — the ledger
format in `engineer.md` is concrete enough that a duplicate example would
add bloat.

**Status:** completed — 2026-05-22. Engineer produced a fully-`DONE`
Completion Ledger (10/10 acceptance criteria, 5/5 mandatory Changes,
1 SKIPPED-by-design-per-spec). Delivery lead spot-checked every `DONE`
row against actual file content; `hero drift` reports zero signals and
all criteria `addressed: true`. Spec auto-archived to `.hero/specs/`.

**Pick up at:** nothing — this spec is done. If the rules don't bite as
expected in subsequent deliveries, file a follow-up bug spec rather than
re-opening this one. Candidate follow-on: add `hero_completion_score` as
a symmetric counterpart to `hero_score` if dishonest ledger fills are
observed in practice (spec's own Risks section flagged this).

→ `.hero/specs/delivery-completion-discipline/spec.md`

**Files actually touched:**
- `domains/engineering/skills/implementation-principles/SKILL.md` — added Excellence Bar (new first principle), rebalanced "smallest correct change," added Anti-punt section
- `domains/engineering/skills/agent-reliability/SKILL.md` — added "Honesty about scope" (Two-Reading Rule + no-reclassification + no-soft-completion) and "Persistence on continuous tasks"
- `domains/engineering/agents/engineer.md` — Two-Reading Rule cross-ref, mandatory Changes/AC rule, Exercise-the-feature gate, replaced "Default output" with structured Completion Ledger format
- `domains/engineering/agents/feature-delivery-lead.md` — loads agent-reliability, added ledger validation step before status flip, non-`DONE` as autopilot halt condition
- `domains/engineering/commands/deliver.md` — ledger validation in single/batch/queue flows, autopilot halt on non-`DONE` ledger, replaced "implementation summary" language

**Skip:** new tooling/commands — instructions-only change; `hero drift` is not the right gate (file-list comparison, not behavior). Also skip: framing this as pure anti-under-delivery — the user's bar is *kick-ass*, not "don't half-ass," and the offensive direction is load-bearing. Change #6 (spec-format worked example) skipped per spec's own optional clause.

## Problem

The four behaviors that produce useless deliveries:

1. **Silent reinterpretation.** Spec says "implement X." Agent reads this as
   "render X's UI." Doesn't surface the ambiguity. Picks the easy reading and
   declares done.

2. **Boundary as escape hatch.** Spec lists a hard item in `## Changes`.
   Agent finds it hard. Reframes it as "out of scope" or "beyond what the
   change requires" and drops it without flagging.

3. **Soft completion summary.** Engineer writes an "implementation summary"
   that mentions what was done. Items that were skipped or partially done are
   omitted or glossed. Delivery lead trusts the summary. Spec is marked
   completed. The user discovers the gap weeks later.

4. **Polite yielding.** Engineer finishes one piece of a multi-step task and
   stops with "let me know if you want me to continue." User has to nudge.
   Each nudge produces one more slice. Net result: ten round-trips to do what
   should have been one.

Today's instructions are silent on all four. The skills that load by default
into the engineer (`implementation-principles`, `agent-reliability`) repeatedly
caution against doing too much. They never caution against doing too little.

## Approach

Six coordinated rule edits across five files, plus one structural addition
(the Completion Ledger). No new tooling. No new commands.

### A0. The Excellence Bar — make it kick-ass, not half-assed

**The single most important rule** added by this spec, and the one that goes
at the top of `implementation-principles` so every implementation-oriented
agent loads it first:

> **Aim for the excellent version of the work, not the adequate one.** When
> you can see two ways to satisfy a Changes item — one that meets the letter
> of the requirement and one that meets the spirit — choose the one that meets
> the spirit, even if it's marginally harder. "Smallest correct change" means
> the smallest change that produces a *great* result, not the smallest change
> that compiles. If the spec asks for a feature, the user wants the feature to
> actually rip — not a stub that satisfies the acceptance criteria on paper.
>
> **Quality bar = outcome excellence, not extra scope.** Doing the agreed thing
> exceptionally well beats doing more things adequately. This rule does not
> license scope creep — it raises the bar on the work that's already in scope.
> If you're tempted to broaden scope, surface that to the delivery lead; don't
> self-authorize.
>
> When you finish, ask yourself: "Would I be proud to show this to a senior
> engineer who cares?" If the honest answer is no, the work isn't done.

This rule lives in `implementation-principles` (not `agent-reliability`)
because it's about *what to build*, not *how to verify*. Pair it with the
existing scope-discipline content — they constrain each other usefully.

### A. Define "done" precisely — the Completion Ledger

Add a **mandatory final output** to [engineer.md](domains/engineering/agents/engineer.md) and
[feature-delivery-lead.md](domains/engineering/agents/feature-delivery-lead.md):
the engineer must produce a **Completion Ledger** before reporting done. The
ledger enumerates every acceptance criterion AND every `## Changes` item from
the spec, and marks each one as:

- `DONE` — implemented and verified
- `PARTIAL` — partially implemented (must include what remains and why it's not done)
- `SKIPPED` — explicitly not done (must include why, and must be acknowledged by user or delivery lead before status flips to `completed`)
- `BLOCKED` — attempted, hit a true blocker (must include what was tried and the specific obstacle)

The delivery lead refuses to mark `status: completed` if any item is `PARTIAL`,
`SKIPPED`, or `BLOCKED` without explicit user sign-off. In autopilot mode, any
non-`DONE` item halts the run and surfaces.

This is the single most important change. It converts "I'm done" from a soft
self-report into a structured artifact the agent has to fill out item-by-item,
and it forces the delivery lead into a refusal posture by default.

### B. The Two-Reading Rule (anti-narrowing)

Add to [engineer.md](domains/engineering/agents/engineer.md) under "Working
from a spec":

> **Two-reading rule.** When a Changes item or acceptance criterion has two
> plausible readings — for example, "implement X" could mean "render the UI
> for X" or "render + wire it up so X actually works" — you must name both
> readings explicitly and pick the more thorough one, OR pause and ask. You
> may not silently pick the easier reading. The default interpretation of any
> verb that could mean "show it" or "make it work" is "make it work."

Pair with a complementary rule in [agent-reliability/SKILL.md](domains/engineering/skills/agent-reliability/SKILL.md)
under a new "Honesty about scope" section, so this travels with any agent
that loads agent-reliability (not just the engineer).

### C. Rebalance "smallest correct change"

Edit [implementation-principles/SKILL.md:17](domains/engineering/skills/implementation-principles/SKILL.md:17) so "smallest correct change"
is defined symmetrically:

> Make the smallest change that fully satisfies the requirement. "Correct" means
> the feature actually works end-to-end for its intended user, not just that
> the code compiles and existing tests pass. If a Changes item or acceptance
> criterion is hard, that is not grounds to drop it or reinterpret it —
> surface the difficulty and either complete it, halt with a clear blocker
> report, or get explicit user sign-off to descope. Silent descoping is a
> failure mode, not pragmatism.

This single sentence is the most-quoted defense agents use to justify
under-delivery. Fixing it at the source is high leverage.

### D. Boundary discipline — symmetric phrasing

In [engineer.md:49](domains/engineering/agents/engineer.md:49) and in the
spec's general rules, add the inverse of the existing boundary rule:

> The `## Boundaries` section names work that is OUT of scope. Items listed
> in `## Changes` and `## Acceptance Criteria` are IN scope and are mandatory.
> You may not move an item from Changes to Boundaries during delivery. If a
> Changes item turns out to be wrong, infeasible, or genuinely out of scope,
> surface it to the delivery lead — do not silently reclassify it.

### E. Exercise-the-feature gate

Add to [engineer.md](domains/engineering/agents/engineer.md)
"Verification rules" and to
[deliver.md](domains/engineering/commands/deliver.md) post-delivery loop:

> **Exercise the feature.** For any change that produces user-visible
> behavior (UI, CLI command, API endpoint, tool output), run the feature
> end-to-end before declaring done. Unit tests prove the code compiles and
> the assertions you wrote pass; they do not prove the feature works. Start
> the dev server, run the CLI command, hit the endpoint, exercise the tool.
> If you cannot exercise it (no sandbox, no test data, no harness), say so
> explicitly in the Completion Ledger rather than claiming `DONE`.

The CLAUDE.md root already has a UI version of this rule. The change brings
it into the engineer agent itself and broadens it past UI to any user-visible
behavior.

### F. Persistence on continuous tasks

Add to [agent-reliability/SKILL.md](domains/engineering/skills/agent-reliability/SKILL.md)
a new "Persistence" section:

> **Do not yield mid-task.** When the user asks you to do something "until
> complete," gives you a multi-step task, or invokes a workflow that has more
> phases to run (`/deliver` on a multi-phase spec, batch mode, queue mode),
> you must continue until the work is complete, you hit a true blocker, or
> the user explicitly interrupts you.
>
> A **true blocker** is one of: a tool returned an error you cannot work
> around, a credential or permission is missing, the next step requires a
> decision only the user can make (and that decision is not already implied
> by the spec), or you have made two failed attempts at the same approach
> and need to reframe.
>
> A **true blocker is NOT**: "this is taking a while," "the next step is
> tedious," "I want to check before continuing," "let me know if you want
> me to keep going." Stopping early with "let me know if you want me to
> continue" is a failure mode, not politeness. The next message should be
> the next step, not a yield.

This is the single highest-leverage edit for failure mode B. The persistence
rule needs to live in `agent-reliability` (not just the engineer) because
this failure happens to every implementation-oriented agent, not only ones
running under `/deliver`.

### Why no new tooling

The user's constraint was explicit: "should not require new tooling unless
clearly necessary; prefer rule changes over new gates." Every behavior above
is enforceable by instruction. `hero drift` already exists for file-level
checks; we are not extending it. The Completion Ledger is a markdown artifact
the engineer writes — no parser changes required for v1. A future spec could
add `hero_completion_score` as a symmetric counterpart to `hero_score`, but
that is out of scope for this iteration.

### Why not just bolt on `hero_completion_score`?

Considered and rejected for v1. The score would be derived from the same
ledger the engineer writes, and the ledger itself is the load-bearing
artifact. If agents fill the ledger honestly, a score is mostly redundant.
If they don't, a score derived from a dishonest ledger doesn't help. Build
the ledger first, observe whether dishonest fills happen in practice, and
add the score in a follow-on spec if so.

## Acceptance Criteria

- THE SYSTEM SHALL state an explicit "Excellence Bar" at the top of `implementation-principles` directing every implementation-oriented agent to aim for the excellent version of the work (the version that meets the spirit of the requirement and that a senior engineer would be proud to ship), not the adequate version that merely satisfies the letter.
- THE SYSTEM SHALL clarify that the Excellence Bar raises quality of in-scope work and is NOT a license for scope creep; agents who see broader work should surface it to the delivery lead rather than self-authorize.
- THE SYSTEM SHALL require the engineer agent to produce a Completion Ledger enumerating every acceptance criterion and every `## Changes` item with a status of `DONE`, `PARTIAL`, `SKIPPED`, or `BLOCKED` before reporting delivery complete.
- IF any Completion Ledger item is `PARTIAL`, `SKIPPED`, or `BLOCKED` THEN THE SYSTEM SHALL prevent the feature-delivery-lead from marking the spec `status: completed` without explicit user sign-off (or, in autopilot mode, halt and surface).
- WHEN a Changes item or acceptance criterion has two plausible readings THE SYSTEM SHALL require the engineer to name both readings and pick the more thorough one (or pause to ask); silent selection of the easier reading is forbidden.
- WHEN the user asks for work to continue "until complete" or assigns a multi-step task THE SYSTEM SHALL require the agent to continue until completion, a true blocker is hit, or the user explicitly interrupts; yielding mid-task with phrases like "let me know if you want me to continue" is classified as a failure mode in the agent-reliability skill.
- WHEN a change produces user-visible behavior (UI, CLI, API, tool output) THE SYSTEM SHALL require the engineer to exercise the feature end-to-end before reporting done, and IF the engineer cannot exercise it THEN THE SYSTEM SHALL require this to be stated explicitly in the Completion Ledger rather than marked `DONE`.
- THE SYSTEM SHALL define "smallest correct change" in `implementation-principles` as one where the feature actually works end-to-end, not merely where the code compiles and existing tests pass.
- THE SYSTEM SHALL state in `engineer.md` that Changes and Acceptance Criteria items are mandatory and may not be silently reclassified as Boundary items during delivery.
- WHERE a spec is being delivered with `--autopilot` THE SYSTEM SHALL treat a non-`DONE` Completion Ledger item as a halt condition equivalent to a test failure or drift warning.

## Changes

1. **`domains/engineering/skills/implementation-principles/SKILL.md`** — Excellence Bar added as the first principle (top of file, above Core principles); "smallest correct change" rebalanced to mean end-to-end working behavior; Anti-punt section added with the three explicit options (complete / halt with blocker report / get sign-off to descope) and the "silent descoping is a failure mode" line. Existing scope-discipline content preserved.

2. **`domains/engineering/skills/agent-reliability/SKILL.md`** — "Honesty about scope" section added (Two-Reading Rule with render-vs-make-it-work example; no-silent-reclassification; no-soft-completion-language). "Persistence on continuous tasks" section added with explicit true-blocker definition and the not-a-true-blocker list. Note: the engineer loads agent-reliability automatically; the delivery-lead now loads it explicitly via the change in #4 below.

3. **`domains/engineering/agents/engineer.md`** — Two-Reading Rule cross-referenced under "Working from a spec." Mandatory-items rule added to "Working from a spec" and "Rules." Excellence Bar and Persistence cross-refs added to "Rules." Exercise-the-feature gate added to "Verification rules." "Default output" replaced with the **Completion Ledger** closing artifact: two markdown tables (acceptance criteria + Changes items), Exercise check, Excellence Bar self-check, status definitions, and rules against performative `DONE` marks.

4. **`domains/engineering/agents/feature-delivery-lead.md`** — Delivery phase now loads `agent-reliability` explicitly. Step 17 inserted after test-coverage verification: validate the engineer's Completion Ledger, cross-check `DONE` rows against on-disk evidence, refuse to flip `status: completed` if any row is non-`DONE` without explicit sign-off. Autopilot halt conditions extended to include non-`DONE` ledger items. Step numbering downstream updated.

5. **`domains/engineering/commands/deliver.md`** — Autopilot mode description updated: halt on non-`DONE` ledger item; Persistence rule cross-referenced; `ledger` added to `--halt-on` options. Batch mode flow requires a Completion Ledger per fix and refuses `status: completed` flip if any row is non-`DONE`. Queue mode confirmation language updated and post-spec validation includes ledger check. Single-spec post-delivery checklist gained step 5 (Validate the Completion Ledger). Final paragraph replaces "implementation summary" language with "the ledger is the artifact of record."

6. **`domains/engineering/skills/spec-format/SKILL.md`** — **SKIPPED.** Per the spec's own optional clause, item #6 was to be done only if items 1–5 left the ledger format under-specified. The format is fully specified in `engineer.md` ("Closing output — the Completion Ledger") with a concrete markdown template, status definitions, and rules. Duplicating it in `spec-format` would add bloat without value. If a future spec finds spec-authors confused about what the engineer will produce, a worked example can be added then.

## Boundaries

- **No new tooling.** No new MCP tools, no new `hero` subcommands, no parser changes. This is an instructions-only change. A follow-on spec can add `hero_completion_score` if the ledger turns out to be insufficient in practice.
- **No changes outside `domains/engineering/`.** Other domains (PM, sales, support) can adopt these patterns later — out of scope for v1.
- **No changes to the spec format itself.** Acceptance Criteria and Changes already exist. The Completion Ledger is an *engineer output*, not a new spec section.
- **No changes to `hero drift`.** Drift detection is a file-list tool. The ledger is a behavior-completion tool. They are complementary, not overlapping.
- **No retroactive ledger requirement.** Existing in-flight specs do not need ledgers reconstructed after the fact. The rule applies to deliveries that start *after* this spec lands.
- **No "stop the agent from being wrong" rule.** This spec is about under-delivery, not about correctness in general. Hallucination, bad design choices, broken code — those are existing agent-reliability concerns and stay where they are.

## Risks

- **Instruction bloat.** The engineering instructions are already long. Adding five rules across five files risks pushing agents into "too much to read, skim and skip." Mitigation: keep each new rule terse (target 3–6 sentences), and prefer cross-references over duplication between `agent-reliability` and `engineer.md`.
- **Performative ledgers.** Agents could fill out the ledger dishonestly — marking everything `DONE` to avoid friction. Mitigation: the delivery lead reviews the ledger against the spec and challenges any `DONE` that doesn't have corresponding code or test evidence. Add a sentence in `feature-delivery-lead.md` requiring this challenge step.
- **Persistence rule overshoots.** Agents may misread "do not yield mid-task" and refuse to surface real blockers, or burn context running in circles. Mitigation: the definition of *true blocker* explicitly includes "two failed attempts at the same approach require a reframe" — this preserves the existing error-recovery rule from `agent-reliability`.
- **Conflict with `/deliver --supervised` confirmations.** Supervised mode asks for confirmations at specialist handoffs. Persistence rule should not override explicit confirmation prompts. Mitigation: phrase the persistence rule as "do not yield *unilaterally* mid-task" — explicit confirmation steps in supervised mode are not unilateral yields.
- **User signs off without reading.** "Are you OK with these SKIPPED items?" is just another prompt the user might rubber-stamp. Partial mitigation: require the engineer to write the *reason* for SKIPPED/PARTIAL in the ledger, so the user is shown specific text rather than a yes/no.
- **Two-reading rule fires on everything.** If every ambiguous requirement triggers a "name both readings" dance, delivery slows to a crawl. Mitigation: phrase the rule as "two *plausible* readings where one is materially easier than the other" — well-specified items don't have two plausible readings.

## Validation

This spec is validated by:

1. **Self-application.** When the engineer agent delivers this very spec, they must produce a Completion Ledger covering every Changes item above. If the ledger format isn't yet defined when delivery starts, the engineer defines it as part of the work (Change #3) and then applies it to themselves. A dogfooded ledger is the strongest signal that the format is workable.
2. **Re-read pass.** After the five files are edited, re-read each one cold and ask: "Does a fresh agent reading this know what to do, both about doing too much AND about doing too little?" If the answer is no for any file, iterate.
3. **Adversarial test.** Pick a recently-completed bug or feature spec and ask a fresh delivery session to apply the new rules retroactively — would the new rules have caught the under-delivery patterns this spec is trying to fix? If the answer is no, the rules are too weak or too narrow.
4. **Drift check.** Run `hero drift delivery-completion-discipline` after the edits land to confirm the Changes list matches the files actually touched.
5. **Index refresh.** Run `hero index --if-stale -q` and `hero queue write -q` after spec save so the spec surfaces in tooling.

There is no automated test for instruction quality. The signal is whether the
*next* batch of deliveries leaves a cleaner trail than the last.
