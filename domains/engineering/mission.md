---
title: Hero Code — Engineering Vertical Charter
type: mission
scope: vertical
vertical: engineering
inherits: ../../.hero/mission.md
locked_at: 2026-04-28
locked_by: chet-bellows
version: 1
---

## Mission

**Hero Code is the engineering vertical of Hero — the sidekick brain
for AI-driven software engineering.**

It rides on [Core Hero](../../.hero/mission.md) and adds the spec
types, agents, skills, commands, and vocabulary that turn the
core engine into a complete project-management + product-management +
engineering + testing + release-readiness toolkit.

The core mission applies unchanged: the model in the editor starts
cold; Hero captures everything that happens during the work and
injects it back automatically; sessions start smart, end smarter, the
floor rises for everyone.

What this vertical adds: the **shape** of the work, in engineering
terms. *Spec-first* (design before you build, diagnose before you
fix). *Specs as living contracts*. *Acceptance criteria that flip
green or red as code moves*. *Conventions that travel with the
codebase*. *Cross-spec, cross-repo, cross-team awareness as the team
scales*.

## What this vertical brings

| Layer | What's in it |
|---|---|
| **Spec types** | feature, bug, initiative, decision, convention, sprint, retro, demo, scratch — each with its own lifecycle |
| **Agents** | engineer, brownfield/greenfield architects, debug-investigator, delivery leads, scrubbers (×7), reviewers (×7), product-ideator, ui-designer — see [`agents/`](agents/) |
| **Skills** | language stacks (Go, Python, JS, TS, React, Rust, Java, Groovy), workflow skills (api-design, db-stack, security-review, debug-investigation, performance, migration, etc.) — see [`skills/`](skills/) |
| **Commands** | `/discover`, `/design`, `/diagnose`, `/deliver`, `/decide`, `/compose`, `/split`, `/review`, `/scrub`, `/release`, `/sprint`, `/retro`, `/check`, `/scan`, `/import`, `/note`, `/docs`, etc. — see [`commands/`](commands/) |
| **Lifecycle** | discover → design → deliver and diagnose → fix-spec → deliver. Plus sprints, retros, releases. |
| **Trackers** | Jira, GitHub Issues, Linear — read/write integration |
| **Wiki targets** | GitHub Pages (default), GitHub Wiki, Confluence |

## How it specializes the core

The core mission test asks: *"Does this make the next agent session
start smarter than the last one ended?"* For Hero Code, "next agent
session" means *the next engineering session — picking up a spec,
debugging a bug, reviewing a PR, planning a sprint, shipping a
release.* Every Hero Code feature must answer: does an engineering
session start with the right context loaded — open specs in flight,
recent decisions, conventions for the files they're touching, ACs
they're being graded on, what teammates are doing nearby, what
blocked them last time?

## Vocabulary additions (engineering-specific)

These extend (never override) the core vocabulary.

- **spec** — design artifact for engineering work; types: feature,
  bug, decision, convention, sprint, etc.
- **delivery** — the implementation phase against an approved spec
- **diagnosis** — the investigation phase that produces a fix spec
- **drift** — divergence between a spec and the code that was
  supposed to implement it
- **acceptance criterion** — a single testable claim a spec makes;
  ACs flip green/red as code moves
- **scrub** — targeted code-quality pass (deadcode, dedup, types,
  defensive, legacy, deps, comments)
- **handoff** / **resume** — session-state checkpoint and restore
  for cross-tool, cross-day continuity

## Anti-patterns specific to this vertical

In addition to core anti-patterns, Hero Code must never become:

- **A spec system that lives separately from the code.** Specs and
  code live in the same repo, in the same PR review, with the same
  blame. If specs become Confluence-with-extra-steps, we failed.
- **A code generator.** The harness writes code; we feed it context.
  We don't write code in the binary.
- **A linter or build tool.** Convention enforcement is advisory —
  agents are instructed to follow conventions; the binary does not
  block builds. (Hard rule, inherited from v2 charter.)

## Other verticals (for reference)

- **Hero Sales** — next vertical (in planning, see [`hero-sales`](../../.hero/planning/features/hero-sales/spec.md))
- **Hero Legal**, **Hero Research**, **Hero Support**, **Hero
  Design** — open territory; same manifesto, different vocabulary

## Inheritance discipline

When [`.hero/mission.md`](../../.hero/mission.md) (the core charter)
changes, this vertical charter is reviewed for compatibility within
the same PR. Vertical charter cannot weaken or contradict the core;
it can only specialize.
