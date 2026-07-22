---
title: Spec Kit and Swappable Workflow Providers
slug: spec-kit-and-swappable-workflow-providers
type: note
created: 2026-07-22
tags: [spec-kit, workflow-providers, interoperability, hero-strategy, research]
---
# Spec Kit and Swappable Workflow Providers

## Context

This note captures a deep comparison between Hero and GitHub Spec Kit, research
into whether Spec Kit's adoption is durable, and the resulting idea that Hero
could support replaceable workflow providers rather than owning every workflow.

## Discussion

**Human:** Compare Hero deeply with GitHub Spec Kit. Does Spec Kit make Hero
unnecessary, or should it become Hero's middle workflow layer?

**Assistant:** Spec Kit has evolved well beyond its original four prompts. Its
core engineering lifecycle includes constitution, specify, clarify, plan,
checklist, tasks, analyze, implement, and converge. It also has a persisted YAML
workflow runtime with gates, conditions, loops, pause/resume, and fan-out/fan-in;
35 agent integrations; presets; extensions; bundles; and organization catalogs.

Spec Kit's strongest areas are:

- Simple public mental model: specification → plan → tasks → implementation.
- Harness integration breadth.
- Artifact clarification and cross-artifact analysis.
- Workflow automation and packaging.
- A large public contributor and extension ecosystem.

Hero's categorically different strengths are:

- A project-wide, temporal corpus rather than isolated feature folders.
- Ranked cold-start context across all relevant project knowledge.
- Failed attempts as expensive-to-rediscover first-class memory.
- Relational traversal: why, blockers, decisions, dependencies, people, code,
  acceptance evidence, and supersession.
- Continuity across sessions, tools, machines, people, and repositories.
- Capture after a workflow so the next workflow starts smarter.

The concise distinction reached was:

> Spec Kit standardizes the work. Hero remembers the work.

**Human:** Could Hero integrate Spec Kit as the middle workflow layer and make
that layer totally swappable for other specification/process systems?

**Assistant:** Yes in principle, but Spec Kit should be a first-class provider,
not a required foundation. Hero Core would own memory, context, expectations,
evidence, provenance, and continuity. Providers could include Hero Code, Spec
Kit, OpenSpec, BMAD, Jira-driven processes, company RFC systems, or a direct/no-
workflow mode.

The provider boundary should normalize only what Hero genuinely needs:

```text
context in
artifacts out
lifecycle events
decisions and failed attempts
evidence and completion outcome
status, pause, resume, and cancellation
```

It should not standardize provider-native file layouts, internal phases, task
formats, prompts, branching strategy, or terminology.

**Human:** Before pursuing that architecture, determine whether Spec Kit is
actually durable or merely a viral fad.

**Assistant:** Research found that Spec Kit is genuinely substantial but not a
proven universal standard:

- It reports more than 121,000 GitHub stars, 9,500 forks, 240 contributors, 35
  integrations, 138 extensions, and 25 presets.
- It remains under extremely active development, with official releases and a
  trusted PyPI package continuing through July 2026.
- It has expanded into workflow execution, packaging, enterprise catalogs,
  air-gapped support, security hardening, and non-engineering workflows.
- A broader spec-driven category has formed around Spec Kit, OpenSpec, BMAD,
  GSD, Kiro, Agent OS, and others.

However, important uncertainties remain:

- Stars measure interest, not retained production use.
- Public data does not reveal 30/90-day project retention, repeat workflow use,
  completion rates, or team adoption.
- Extension count measures supply, not installation or maintenance quality.
- Users report ceremony, excessive generated text, difficulty evolving living
  specs, artifact drift, and weak multi-repo/multi-team fit.
- The category remains fragmented and most systems rely on prose artifacts that
  may not independently prove implementation correctness.

The durable bet is structured intent, decomposition, coordination, and
verification—not necessarily Spec Kit's current artifact model.

## Proposed Architecture

```text
Human and harness
        ↓
Hero Core
memory · context · expectations · provenance · evidence
        ↓
Minimal workflow-provider boundary
        ├── Hero Code
        ├── Spec Kit
        ├── organization-specific process
        └── direct / none
        ↓
Agent execution and business systems
        ↓
Hero captures outcomes and compounds knowledge
```

Potential provider events:

```text
workflow.started
phase.started
artifact.created
artifact.updated
assumption.recorded
decision.recorded
attempt.failed
approval.requested
verification.recorded
workflow.completed
workflow.failed
```

Provider-reported completion must remain distinct from Hero-verified evidence.

## Recommended Experiment

Do not publish a universal protocol or reorganize Hero around Spec Kit yet.

Build a narrow adapter for one real lifecycle:

```text
Hero context
    → Spec Kit specify / clarify / plan / tasks
    → Hero indexes artifacts
    → Spec Kit implement / converge
    → Hero captures decisions, evidence, knowledge, and handoff
```

Use it for 5–10 substantial changes and compare against Hero Code using:

- User correction count.
- Artifact review time.
- Delivery quality.
- Token cost.
- Workflow intervention burden.
- Cold-session resume quality.
- Durable knowledge produced.
- State duplication or disagreement.
- Adapter maintenance burden.
- Whether the user voluntarily chooses it again.

Extract and publish the provider contract only after at least two meaningfully
different implementations prove the same boundary.

## Key Takeaways

- Spec Kit is too active and substantial to dismiss, but not proven enough to
  become a required Hero dependency.
- Hero should be able to use Spec Kit without depending on it.
- Hero should own continuity and project understanding; providers may own phase
  sequencing and native artifacts.
- The provider interface is useful only if it stays small and emerges from real
  implementations.
- Do not hollow out Hero Code or build a speculative interoperability platform
  before dogfooding proves replacement value.
- Spec Kit's market success validates demand for structured AI work and may
  educate users who later discover the continuity problem Hero solves.
- Possible graduation: a design/decision spec after the two-provider experiment.
