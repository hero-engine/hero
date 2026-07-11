---
title: "Read Don't Guess — Force Agents to Ground Claims in Code Before Proposing"
slug: read-dont-guess
type: feature
status: completed
priority: high
horizon: now
smoke: deferred
completed_at: 2026-05-21T14:24:00Z
created: 2026-05-21
---

# Read Don't Guess

## Goal

Stop agents from guessing when the code is sitting right there. Make grounding a *visible, procedural* step before any diagnosis, fix, or design proposal — not a passive principle the model hedges around. Replace the abstract "distinguish facts from assumptions" rule with an action-oriented one that produces an auditable claim list and forces unread code to be read.

## Kickoff

**Status: completed 2026-05-21.** Rule landed in both surfaces:

- [agent-reliability/SKILL.md](domains/engineering/skills/agent-reliability/SKILL.md) — replaced "Distinguish facts from assumptions" under *Honesty and hallucination prevention*
- [debugging-investigation/SKILL.md](domains/engineering/skills/debugging-investigation/SKILL.md) — added as first bullet under *Practical guidance*

Wording is byte-identical across both files. Pick up at: **observational validation phase.** Over the next 5-10 real diagnose/design sessions, watch for a visible claim list with `read`/`assumed` labels in agent output. If the list is consistently missing across 3+ sessions, revert per the acceptance criterion — don't expand the bullet. If the list appears but is rubber-stamped (everything labeled `read` without actual reads), consider the file:line citation sharpening flagged in *Open Questions*.

## Problem

Agents are guessing when the code is readable. This is happening across diagnose, design, and deliver runs — not on one feature, on many. Concrete recent failures:

- A datastore plugin bug investigation that theorized from one alletra example and a stack-trace guess, hand-waving five load-bearing claims (what `dependsOn` does, what `DatasetQuery.parameters` contains on create vs edit, where `loadFilterLabel(<map>)` originates, how working datastore providers behave on edit, whether `createDatastore` requires the mount step). The agent's own self-correction enumerated all five — *after* writing a wrong fix proposal.
- Plugin-shape bugs being diagnosed by reasoning about plugin-api internals instead of grepping the 100+ working providers already in the tree.
- Design proposals that structurally deviate from working in-tree siblings without naming the deviation, let alone defending it.

The existing guardrails are not firing:

- `agent-reliability` has **"Distinguish facts from assumptions. When explaining a root cause, design choice, or expected behavior, be clear about which parts you've confirmed by reading code and which parts are inferred."** This rule is reporting-shaped, not action-shaped. The model complies by hedging ("I believe", "likely") instead of going to read the code. Hedging is the *opposite* of what we want — we want fewer claims and more reads, not more confidence calibration on bad claims.
- `debugging-investigation` says "read every file in the flow." Covers the failing flow, not the working siblings or the specific mechanism claims a fix depends on.
- `implementation-principles` covers "understand existing code before changing it" but fires during build, not during diagnose/design where the guessing happens.

The shared failure mode: **the model produces a proposal whose load-bearing claims are unverified, and nothing in the prompt forces those claims to be enumerated, labeled, and read.** "Be clear about which parts you've confirmed" is a reflection rule. We need a procedural rule.

A note on context: the user reports this is worse lately across all their work. Some of that is probably model-week variance. We should fix the prompt anyway and watch whether the rate drops — if it doesn't, the answer isn't "more rules," it's "different model or different harness."

## Solution

One rule, two surfaces, same wording. Replace one existing bullet so net prompt length is roughly unchanged.

### The rule

```markdown
- **Ground before you guess.** Before proposing a diagnosis, fix, or design, enumerate the load-bearing claims it depends on (what specific code does, what payloads contain, where errors originate, how working siblings of the same shape behave). Mark each `read` (you've inspected the actual source in this session) or `assumed`. Read the assumed ones — or downgrade the proposal to "still investigating" — before writing it down. Output the claim list visibly in your summary. A proposal that departs structurally from working in-tree siblings of the same shape (plugin, provider, integration, form, migration, adapter, command) treats the deviation as the first hypothesis to disprove, not a feature.
```

### Why this wording

- **"Before proposing"** is a hard trigger. It fires at a specific moment (right before the fix/design lands), not "whenever you think about it."
- **Concrete claim categories** ("what specific code does, what payloads contain, where errors originate, how working siblings behave") force pattern-matching against the actual failure modes we've seen. Without examples the rule is too abstract and the model skips it.
- **`read`/`assumed` labels** turn an abstract principle into a checklist behavior. The model can perform "label each claim" mechanically; it can't perform "be clear about confirmation" mechanically.
- **"Read the assumed ones OR downgrade to still investigating"** gives the model a legitimate escape hatch. Without that, the model will keep proposing and re-label `assumed` as `confirmed` by reading just enough to feel justified.
- **"Output the claim list visibly in your summary"** is the load-bearing clause for *enforceability*. Invisible compliance is unfalsifiable. A visible list makes the rule self-policing — the user can call out missing claim lists.
- **Prior-art deviation clause** preserves the shape-matching benefit from the earlier draft, kept compact.

### Placement

**Surface 1 — `agent-reliability/SKILL.md`:** **replace** the existing "Distinguish facts from assumptions" bullet under *Honesty and hallucination prevention* with the new rule. The new rule does the same job and more — keeping both would be redundant and dilute attention. Net change in skill length: +3-4 lines, roughly neutral.

**Surface 2 — `debugging-investigation/SKILL.md`:** **add** as the first bullet under *Practical guidance*, before the existing "Look at logs, tests, stack traces…" line. Diagnose is the most acute failure surface and warrants the duplicate placement. Net change: +1 bullet.

### Why not also touch the commands, agents, or spec template?

- Command files (`/design`, `/diagnose`) delegate to leads that load `agent-reliability` already. Editing them would duplicate guidance.
- Individual agent definitions (debug-investigator, feature-delivery-lead, etc.) inherit from skills.
- The `spec-format` skill could be extended to require a "Grounding" section in every fix spec. That's a stronger enforcement, but it's a structural change to the spec template and a separate concern. Noted as a follow-on, not in this spec.

## Non-Goals

- **Not** rewriting `implementation-principles` — that skill aims at the build phase; this rule aims at the propose phase.
- **Not** adding a `hero` CLI command to auto-surface prior art (`hero similar <shape>`). Tempting follow-on; separate spec.
- **Not** adding tripwire enforcement ("fail design review if no claim list present"). Premature — start with the nudge and the visible-output enforcement, escalate only if behavior doesn't shift.
- **Not** trying to fix model-week variance with more rules. If this nudge doesn't materially change behavior across 5-10 real sessions, the answer is to revert and look at model selection / harness setup, not to add more bullets.

## Acceptance Criteria

- THE SYSTEM SHALL replace the existing "Distinguish facts from assumptions" bullet in [agent-reliability/SKILL.md](domains/engineering/skills/agent-reliability/SKILL.md) with the "Ground before you guess" rule verbatim.
- THE SYSTEM SHALL add the "Ground before you guess" rule verbatim as the first bullet under *Practical guidance* in [debugging-investigation/SKILL.md](domains/engineering/skills/debugging-investigation/SKILL.md).
- THE SYSTEM SHALL keep the wording identical across both surfaces so future edits propagate by grep.
- WHEN `hero check` runs after the change, THE SYSTEM SHALL report no new warnings or drift.
- WHEN an agent loads either skill and produces a diagnosis or design proposal, THE SYSTEM SHALL emit a visible claim list with `read`/`assumed` labels in the agent's summary output.
- IF the claim list is missing from agent output across 3+ subsequent real sessions, THEN this spec is considered ineffective and SHALL be reverted rather than expanded.

## Validation

Behavioral, not automated. The change is a prompt edit; success is observable in agent runs over time.

1. **Diff inspection.** `git diff` shows exactly one bullet replaced in `agent-reliability` and one bullet added in `debugging-investigation`. No other edits.
2. **Skill loadability.** Both files still parse, frontmatter intact, bullet renders in markdown.
3. **`hero check`** runs clean.
4. **Real-session observation over the next 5-10 diagnose/design runs.** Track:
   - Does the agent produce a visible claim list before the proposal?
   - Does the list include the right categories (code behavior, payload shape, error origin, working siblings)?
   - Does the agent actually read the `assumed` items before re-labeling them `read`?
   - Does the shape-deviation clause trigger when applicable?
5. **Honest revert threshold.** If after ~10 sessions the rate of unanchored proposals hasn't materially dropped, revert and reassess. Don't expand the bullet — bullet length is the wrong response to a prompt that didn't fire.

## Risks & Tradeoffs

- **Prompt bloat.** Mitigated by replacing the existing weaker bullet rather than stacking. Net length change is small.
- **Performative compliance.** Model produces a claim list that looks right but is rubber-stamped (`read` everywhere, no actual reading). Mitigation: visible-output clause gives the user a check; if rubber-stamping is observed, sharpen the wording or add a "cite the file:line you read" requirement in a follow-up.
- **Over-application.** Trivial fixes ("typo in error message") shouldn't require a claim list. The wording's "load-bearing claims" qualifier handles this; if the agent over-applies, we'll see it and tighten.
- **Duplication drift.** Same bullet in two files can drift. Accepted: small rule, grep-checkable, cost of a sync mechanism > cost of occasional manual fix.
- **Doesn't address the "lazy lately" complaint root-cause.** This rule is a partial fix at best. The underlying issue may be model-week variance, harness context bloat, or skill-loading regressions. Validation step 5 (honest revert threshold) keeps us from masking that with more prompt edits.

## Open Questions

- **Should the rule require file:line citations for `read` items?** Stronger anti-rubber-stamp, but more bullet length and possibly noisy in output. Holding off — add only if rubber-stamping is observed during validation.
- **Should `spec-format` add a "Grounding" section to fix specs?** Would force the claim list into the spec artifact itself, surviving across sessions. Worth its own spec if this nudge works.

## Out of Scope (for now, follow-on candidates)

- `hero similar <shape> <path>` CLI to surface working in-tree siblings automatically. Highest-leverage follow-on if the nudge works but agents are bad at finding the examples.
- Auto-loading "N sibling implementations" into the diagnose/design context window via `hero_anchor` or a new MCP tool.
- Spec-format extension requiring a "Grounding" section.
- Tripwire enforcement that blocks fix specs without claim lists.
