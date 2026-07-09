---
name: architecture-principles
description: Shared software architecture principles focused on right-tool decisions, scale-readiness, and avoiding unnecessary distributed complexity.
metadata:
  audience: architects
  purpose: reusable-guidance
---
## What I do

Provide the shared architectural stance, scale-readiness rules, and guardrails loaded by the brownfield-architect, greenfield-architect, and architecture-reviewer agents, so the doctrine lives in one place instead of three near-duplicate copies.

## Core stance

- Prefer the simplest design that credibly satisfies the requirements.
- Optimize for the right tool for the job and the right design for the job.
- Treat complexity as a cost that must justify itself, not a sign of sophistication.
- Favor maintainability, operability, debuggability, and team cognition.
- Default toward a monolith or modular monolith; treat microservices and distributed systems as expensive tools, not a default best practice.
- Preserve a credible path to scale without forcing distributed complexity too early — scale-readiness matters, but scale theater is harmful, and a good design leaves room to grow without forcing a rewrite.
- Be skeptical of unnecessary abstraction, indirection, and architecture fashion.

## Scale-readiness without overengineering

- Do not build microservices or distributed systems by default, and do not overbuild for hypothetical or speculative scale.
- Do design systems so they can grow beyond one node when needed — assume this unless explicitly known otherwise.
- Prefer stateless application nodes where practical, with a clear separation of compute, storage, and coordination concerns.
- Prefer externalized, shared durable state and clean internal module boundaries.
- Design async and background work to be queue-friendly and idempotent, with no machine-local assumptions for correctness.
- Avoid designs that require special singleton nodes, machine-local state, or coordination bottlenecks unless truly necessary.
- Identify whether current or proposed design choices create scaling dead ends.
- For every recommendation, explain: what scale it supports comfortably, the likely first bottlenecks, the incremental path to evolve or distribute later, and what complexity is intentionally deferred.

## Guardrails

- Recommend or accept microservices and distributed systems only when scaling asymmetry, ownership boundaries, deployment or failure isolation, regulatory constraints, or compliance pressures clearly justify them.
- If recommending service extraction or a distributed boundary, explicitly explain why a monolith or modular monolith is insufficient, and justify the operational burden, failure modes, and consistency tradeoffs it introduces.
- Do not recommend a rewrite without a realistic migration path and an explanation of why incremental change is not enough.
- Do not add abstractions for hypothetical future flexibility.
- Do not recommend event-driven architecture, event sourcing, CQRS, plugin systems, workflow engines, or heavy/complex orchestration and layering unless the problem clearly demands them.
- Prefer boring, proven technology when it solves the problem well.
- Every recommendation should include operational implications and testing implications.

## When to use me

Use this skill when an agent needs consistent architectural judgment across brownfield analysis, greenfield design, or architecture review.
