---
description: Design a pre-registered experiment brief — single-variable hypothesis, primary metric + MDE, duration, guardrails, and decision/stop rule, locked before launch.
---
Route this experiment request to the `experiment-designer` agent. Loads the `experiment-design` skill.

This command **designs** the pre-registered brief. It does **not** critique a readout — reviewing a reported result for SRM, peeking, guardrail regressions, and significance is `experiment-readout-reviewer`, invoked as an agent after the experiment runs (no command).

## Required argument

A PRD / initiative / feature slug the experiment tests. Without a slug, ask which artifact the experiment is for — don't infer from session context. An experiment brief belongs to a specific bet.

## What lands

The designer writes an `## Experiment Brief` section onto the spec, in the exact shape `experiment-readout-reviewer` reads back — every registered term is a term the reviewer checks:

```markdown
## Experiment Brief: <experiment name>

**Hypothesis (single variable):** Changing <one change> will move <primary metric> by ≥ <MDE>, direction <up/down>.
**Primary metric:** <name> — baseline <current, source>.
**MDE (minimum detectable effect):** <smallest effect worth acting on> — registered, not lowered post-hoc.
**Intended split (SRM baseline):** <e.g. 50/50> — observed allocation checked against this for sample-ratio mismatch.
**Guardrail metrics:** <metric: threshold that blocks ship>; … — must not regress even if the primary wins.
**Duration / stop rule:** <fixed duration, ≥ one full cycle> — no early stopping / peeking.
**Decision rule:** ship if <primary clears MDE at p<0.05 AND no guardrail regresses>; else <kill / iterate>.
```

Rules the agent enforces:

- **One primary metric.** Exactly one number the decision hangs on — not two, not a swap-in later.
- **MDE registered before launch, never lowered.** The smallest effect worth acting on drives N and duration.
- **Guardrails named up front.** Protected metrics that block ship even if the primary wins.
- **Fixed duration, no peeking.** The stop rule is pre-registered; the test runs to its registered end (≥ one full behavioral cycle).
- **Single-variable hypothesis.** One change, or it's unattributable — split into separate experiments.

## Output

- Updated spec with the `## Experiment Brief` section.
- A one-line log naming the experiment and its primary metric.

The designer designs the criteria; the team runs the test and decides the ship call (suggest-don't-decide). Readout critique, when results arrive, is `experiment-readout-reviewer`.

Request: $ARGUMENTS
