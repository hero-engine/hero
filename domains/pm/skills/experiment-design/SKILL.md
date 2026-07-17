---
name: experiment-design
description: The pre-registered experiment brief — the artifact that fixes, before data, the single-variable hypothesis, primary metric + MDE, intended split, guardrails, and decision/stop rule. The brief the readout-reviewer reads back.
metadata:
  audience: experiment-designer, experiment-readout-reviewer
  purpose: framework-guidance
---

## What I do

Define the **pre-registered experiment brief**: the artifact an experiment is designed *from*, and the artifact its readout is judged *against*. The brief fixes every load-bearing term — the hypothesis, the primary metric, the minimum detectable effect (MDE), the intended split, the guardrail metrics, and the decision/stop rule — **before any data comes in**. Once launched, every registered term is frozen; changing one after the fact is the single most common way a "win" turns out to be a false positive.

This skill is one half of a contract. `experiment-designer` writes the brief; `experiment-readout-reviewer` (child #6) reads it back check-for-check at readout time. Every term named here is a term the reviewer looks for — the field names below and the reviewer's checklist are the same vocabulary on purpose.

## When to use me

- Designing an A/B test, holdout, or phased rollout before launch.
- Answering "what should we test / how would we know if this worked."
- Writing the `## Experiment Brief` section on a PRD, initiative, or feature spec.
- Any time a team is about to ship a change *to learn from it* rather than just ship it.

If a readout arrives with no brief on file, that is itself the reviewer's first finding — an experiment with no pre-registration can only be caveated, never fully cleared. Author the brief up front so the readout has something to be held to.

## Pre-registration — the whole point

Pre-registration is the discipline of committing the plan before the data. It is the only defense against the classic false-positive failures, each of which is a registered term being quietly changed after launch:

- **Peeked stop** — the stop rule was "stop when it looks significant" instead of a fixed duration.
- **Swapped metric** — the headline reports a metric that wasn't the registered primary.
- **Post-hoc MDE** — the bar for "worth shipping" was lowered after the effect came in small.
- **Dropped guardrail** — a protected metric regressed and the readout just doesn't mention it.

The rule: **a registered term changed after launch is a top finding at readout.** If the plan genuinely needs to change, that ends the current experiment and starts a new pre-registration — you do not edit the brief mid-flight.

## The five locked terms

Every brief fixes these five before launch. They are locked — registered, not adjustable once data is flowing.

### 1. Primary metric

Exactly **one** primary metric — the single number the ship/kill decision hangs on. It is the registered primary; a readout that decides on any other metric has swapped the primary and that is a finding. Define it to `metrics-design` standards: observable, leading, outcome-tied, with a named baseline (baseline before target). Secondary metrics may be *watched*, but they do not get a vote in the primary decision.

### 2. Primary metric MDE (minimum detectable effect)

The **MDE** is the smallest effect worth detecting — the lift below which you would not act even if it were real. It is registered before launch and **never lowered post-hoc**. The MDE drives the math: together with baseline variance and the chosen power (conventionally 80%) and significance (conventionally 5%), the MDE sets the sample size N and therefore the **duration**. A test powered for a 2% MDE cannot honestly claim a 0.3% win; a delta below the registered MDE is noise you agreed in advance not to chase.

### 3. Intended split (SRM baseline)

The **intended split** is the registered allocation between arms (e.g. 50/50, or 90/10 for a holdout). It is the baseline the readout's *observed* allocation is checked against — a **sample-ratio mismatch (SRM)** is when the observed split departs significantly from the intended split, which means randomization or logging is broken and the entire comparison is invalid regardless of how clean the lift looks. Register the split so the reviewer has a number to test the observed allocation against.

### 4. Guardrail metrics

**Guardrail metrics** are the protected metrics that must not regress while the primary moves — latency, error rate, revenue-per-user, retention, unsubscribe rate. A primary lift bought with a guardrail regression is not a win; it is often a net loss the headline hides. Name each guardrail *before* launch, with the regression threshold that would block ship even if the primary wins. Frame each as a scenario/indicator/response per `risk-surfacing`: e.g. "if p95 latency rises >10% (indicator), the primary lift does not justify ship (response)."

### 5. Pre-registered decision rule + stop rule

The **decision rule** is the ship/kill criterion stated in advance: "ship if the primary clears its MDE at p<0.05 **and** no guardrail regresses past threshold; otherwise kill or iterate." The **stop rule** is the fixed duration (or pre-committed sequential-testing plan) — the test runs to its registered end, covering at least one full behavioral cycle (usually a week) to survive novelty/primacy effects. **No early stopping, no peeking**: deciding to stop when the result first looks significant inflates the false-positive rate and is a peeked stop at readout.

## Single-variable hypothesis

One change, one primary metric, one falsifiable claim. The hypothesis names the single variable being changed and the directional effect expected on the primary: "Changing X will increase [primary metric] by at least [MDE]." If you cannot state it as one sentence with one variable, the experiment tests too much to attribute the result — split it into separate experiments. A multi-variable change that "wins" tells you nothing about *which* change won.

## Brief template — the pre-registered artifact

This is the copy-paste `## Experiment Brief` block the designer writes onto the spec and the reviewer reads back. The field names are the term-parity contract: each line here is a line the reviewer's checklist checks.

```markdown
## Experiment Brief: <experiment name>

**Hypothesis (single variable):** Changing <one change> will move <primary metric> by ≥ <MDE>, direction <up/down>.

**Primary metric:** <name> — baseline <current value, source>. The one metric the decision hangs on.
**MDE (minimum detectable effect):** <smallest effect worth acting on> — registered, not lowered post-hoc.
**Intended split (SRM baseline):** <e.g. 50/50> — observed allocation is checked against this for sample-ratio mismatch.
**Guardrail metrics:** <metric: threshold that blocks ship>; <metric: threshold>; … — must not regress even if the primary wins.
**Duration / stop rule:** <fixed duration, ≥ one full cycle> — runs to the end; no early stopping / peeking.
**Decision rule:** ship if <primary clears MDE at p<0.05 AND no guardrail regresses>; else <kill / iterate>.

**Corpus grounding:** <baseline source, prior experiment, intake driving this — doctrine 1>.
```

Everything in the brief is registered *before* launch. At readout, `experiment-readout-reviewer` holds the reported result to exactly these fields — registered primary metric (not a swapped one), registered MDE (not a lowered bar), intended split (SRM check), registered guardrails, registered stop rule (not a peeked stop). A readout that changed any registered term is the reviewer's top finding.

## Anti-patterns

- **Multiple primary metrics.** Two primaries means you decide on whichever one won — a swap waiting to happen. Register exactly one.
- **MDE set (or lowered) after the data.** Post-hoc MDE turns "too small to matter" into "significant." The MDE is registered before launch, full stop.
- **Peeking / early stopping baked into the plan.** "Stop when significant" is a coin flip dressed as a result. Register a fixed duration.
- **Guardrails omitted.** A brief with no guardrails invites a primary lift bought with a latency or revenue regression. Name them before launch.
- **Multi-variable hypothesis.** Change two things and a win is unattributable. One variable per experiment.
- **Editing the brief mid-flight.** A registered term that needs to change ends this experiment and starts a new pre-registration — it does not get quietly rewritten.
- **Deciding the ship call in the brief.** The designer designs; the team runs the test and reads the result and decides (doctrine 2). The brief fixes the criteria; it does not pre-ordain the outcome.

## Cross-references

- `metrics-design` — the primary metric and its baseline are defined to these standards (observable, leading, baseline-before-target).
- `risk-surfacing` — each guardrail is a scenario/indicator/response, not a vague worry.
- `assumption-testing` — pre-registration is assumption-testing discipline: fix the hypothesis and the bar before the data.
- `pm-agent-doctrine` — the brief's grounding line cites the corpus (doctrine 1); the designer suggests the design, the team decides the ship (doctrine 2).
- `experiment-readout-reviewer` (agent, child #6) — the downstream consumer: it reads this brief back check-for-check at readout time.
</content>
</invoke>
