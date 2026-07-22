---
title: Hero Differentiated Identity
slug: hero-differentiated-identity
type: note
created: 2026-07-22
tags: [hero-strategy, positioning, project-memory, continuity, product-direction]
---
# Hero Differentiated Identity

## Context

This note captures a strategic conversation prompted by inspecting Hero's
feature set and implementation, comparing the experience of working with and
without Hero, and asking whether Hero remains worth pursuing after the rise of
large spec-driven toolkits.

## Discussion

**Human:** Hero makes AI-assisted work substantially easier. With Hero, it is
clearer what is expected, there are fewer surprises, and less time is spent
re-explaining, correcting, and recovering from work that went sideways. Without
Hero, supervision becomes tedious. How valuable does Hero look from the model's
side, and what could make it indispensable?

**Assistant:** Hero's value is real and visible from a cold agent session. A
single `hero resume` supplied mission, user identity, prior intent, suggested
next action, active work, recent changes, blockers, acceptance changes, and
nearby initiatives. That changes the agent's starting question from “what is
this repository and what probably matters?” to “given known intent, history,
constraints, and evidence, what is the best next move?”

The implementation also contains substantive machinery rather than prompt
theater: a temporal graph, bounded and ranked context digestion, per-user
handoffs, retrieval, acceptance-criterion state, traversal, active-session
tracking, cross-repo peering, and workflow/server integrations.

**Human:** Hero is not merely useful—it feels impossible to work without it.

**Assistant:** That is the strongest founder signal in the conversation. The
dependency does not appear to come mainly from generating specifications. It
comes from changing the quality of the human-machine relationship:

- Intent does not need to be reconstructed every session.
- Decisions, failures, and constraints survive.
- The next agent receives context the human would otherwise have to repeat.
- Completion is disciplined and tied to evidence.
- The project becomes easier to work on as experience accumulates.

Hero should optimize for **supervision burden**, not merely coding speed. Useful
measures include repeated explanations, corrective prompts, repeated rejected
approaches, time to useful work, scope drift, unsupported completion claims,
manual context pasted, and cold-resume quality.

**Human:** Hero has not been documented, marketed, or open-sourced well, so the
first-wave spec-tool opportunity may have been missed. It needs a unique fit.

**Assistant:** Hero likely did miss the easiest opportunity to launch as
“another spec-driven AI development framework.” That does not mean its strongest
opportunity was missed. Spec templates, agent rosters, planning commands, and
workflow sequencing are becoming commodities. Hero's differentiated product is
the continuity and project-intelligence layer around those workflows.

The strongest positioning developed in the conversation was:

> Hero gives every AI agent the memory of your project.

and, more emotionally:

> Correct your AI once. Hero makes sure you never have to explain it again.

The indispensable loop is:

```text
understand project
    → structure real work
    → capture what happens
    → preserve corrections, decisions, attempts, and evidence
    → resume intelligently
    → make the next session better
```

Hero should not merely remember statements. It should preserve provenance and
distinguish user-stated intent, inference, verified fact, stale knowledge,
superseded knowledge, contradictions, and evidence-backed completion.

**Human:** There may instead be secret sauce in using Hero to build
domain-specific AI workflows for businesses.

**Assistant:** That fits Hero's architecture: vocabulary, artifact types,
specialist roles, skills, workflow routing, integrations, memory, approvals,
evidence, and evaluation. But it would be premature to declare a broad domain
platform before proving that Hero can reproduce its “cannot work without it”
effect outside engineering. Domain packs should initially be treated as a
creative and empirical medium, not as a large platform promise.

## Product Thesis

Hero is not best understood as a proprietary specification methodology.

Hero is a durable intelligence layer that binds:

- Human intent and expectations.
- Project and organizational knowledge.
- Workflows and systems of record.
- Agent actions and artifacts.
- Decisions, attempts, corrections, and evidence.
- Continuity across sessions, tools, people, machines, and repositories.

The workflow is the work surface. The compounding corpus is the asset.

## Must-Win Experience

The canonical demonstration should use two different tools:

1. In the first tool, investigate substantial work, reject an approach, discover
   a constraint, make partial progress, and stop.
2. Open a cold session in another tool and say “continue.”
3. The second agent should know the goal, rejected approach, discovered
   constraint, current state, residual risk, and next action without manual
   re-priming.

The activation metric should be the percentage of second cold sessions that
resume accurately without manual context reconstruction—not first specs created
or commands invoked.

## Strategic Discipline

Classify Hero work as:

- **Essential:** directly causes the cannot-work-without-it experience.
- **Supporting:** makes essential behavior reliable and trustworthy.
- **Experimental:** potentially valuable but not yet proven through use.
- **Distraction:** broadens Hero without strengthening user dependence.

Installation, capture, persistence, cross-machine continuity, graph/index
self-healing, truthful guidance, knowledge visibility, and evidence-backed
completion are trust-critical. Failures there are mission-level defects, not
ordinary rough edges.

## Key Takeaways

- Hero's value is demonstrated through sustained personal use, not merely an
  abstract market thesis.
- The unique advantage is accumulated project understanding and reduced human
  supervision, not the number of commands, agents, or templates.
- Hero should not compete primarily as a larger spec framework.
- “I only have to correct the machine once” is a strong product promise.
- A trustworthy memory system must self-heal and must never silently lose or
  misrepresent context.
- Open source may provide trust and distribution, but the product must first be
  understandable and independently usable.
- The next external validation is whether a small cohort of serious users also
  becomes unwilling to work without Hero.
- Possible future graduation: formalize this thesis as a positioning decision
  after external dogfooding validates it.
