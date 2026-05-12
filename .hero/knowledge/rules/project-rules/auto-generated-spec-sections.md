---
title: Skill-authored content beats projected content in source files
type: rule
tags: [architecture, specs, skills, hooks, projection]
created: 2026-04-30
---

## Rule

When deciding how to populate content in a source file (a `spec.md`, a config, a doc), prefer **skill-authored content** over **heuristic projection** unless the content is purely derived state.

| Approach | When to use | Example |
|---|---|---|
| **Skill-authored, inline** | Content needs judgment, prose, or model-quality framing. Authored once during a workflow (`/design`, `/deliver`), edited inline by the user, lives in the file. | `## Kickoff` sections in specs. AC bullets. Prose Goal/Problem statements. |
| **Projected derived state** | Content is a deterministic function of other state, has no opinion to express, and would be the same whether a model or a script wrote it. Lives in a separate file or a clearly-marked machine block. | NEXT.md machine half (`<!-- BEGIN HERO MACHINE STATE -->`). `.hero/QUEUE.md` snapshot. |

The failure mode to avoid: **heuristic projection of content that wants judgment.** A hook that stitches together a "kickoff prompt" from `## Goal` first sentence + `## Changes` first three lines produces lossy slop next to what a `/design` skill writes with the full spec context. Once you have a hook doing it, you also need a `tuned` lock flag, a force-override, a drift detector, a diff command — all infrastructure to paper over the fact that the hook is a worse author than the skill.

**Decision rule:** if you reach for a `tuned: true` lock to protect hand-authored content from a projector, the projector shouldn't exist. Let the skill or human author once and trust the file as source of truth.

## Related rule: hooks must stay heuristic

Pre-commit hooks, post-merge hooks, and any synchronous projection path **must not depend on model calls**. Reasons:

- Latency budget — hooks should run in milliseconds.
- Reliability — no API key, no internet, no daemon → broken commits.
- Predictability — a hook that sometimes calls Opus and sometimes doesn't is impossible to reason about.

This rule and the rule above are paired. Together they say: hooks project derived state (cheap, deterministic, no judgment); skills author content (judgment, prose, model in the loop). Don't blur the boundary.

## Concrete applications

- `kickoff-prompts-queue` spec — `## Kickoff` is skill-authored (during `/design` and `/deliver`). `.hero/QUEUE.md` is a hook-projected snapshot of pure DB state. No projector touches the spec body.
- `pre-commit-auto-stage-next` — projects NEXT.md and stages it. Pure derived state, no model call.
- `agent-cold-start` (the legacy NEXT.md design) — explicitly chose agent direct file I/O over a CLI projector for the agent half. Same rule in different shape.
