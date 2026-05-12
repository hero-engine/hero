---
title: "Hand Delivery Workflow — Developer Autonomy"
type: feature
status: completed
priority: high
tags: [core, philosophy]
delivery_method: manual
horizon: now
---

# Hand Delivery Workflow — Developer Autonomy

## Principle
Hero must always support the workflow: take the spec, implement it yourself. This is not optional. Losing this story loses mindshare with senior engineers.

## Why this matters:
- Senior engineers who think they can code better than any AI should be able to prove it — using the same spec, same acceptance criteria, same governance
- The spec is the contract, not the delivery mechanism
- The governance layer doesn't care HOW a spec was delivered — only that it traces back to a spec
- A developer should be able to: read the spec, implement by hand, run the acceptance criteria, mark delivered
- Hero's value to them: the spec itself (clear requirements), the knowledge base (conventions, architecture), the governance (their PR passes the same checks)

## Implementation:
- `hero deliver --manual <slug>` or simply reading the spec and coding
- Spec status transitions: draft → approved → in-progress → delivered (regardless of agent vs human)
- Same quality gates apply: tests pass, conventions followed, spec scope respected
- Dashboard tracks delivery method (agent vs manual) without penalizing either

## Batch workflows respect both paths:
The batch pipeline (`import → diagnose → deliver`) generates specs at every stage. A developer can:
- Let the agent batch-diagnose 30 bugs, then hand-deliver the fix specs they care about
- Cherry-pick from a batch: "agent delivers these 20, I'll take the hard 10"
- Use agent-generated diagnosis specs as starting points, refine them, deliver manually
- Mix freely: some specs agent-delivered, some hand-delivered, same governance, same dashboard

The spec is always the handoff point. Whether you hand it to an agent or keep it is your call.

## Cultural note:
The grumpy engineer who hand-delivers specs and beats the AI agent on quality is VALUABLE. They're the ones who write good conventions and catch bad specs. Don't alienate them.
