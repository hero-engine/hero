---
title: Second Model Design Review — Automated Second Set of Eyes
type: note
created: 2026-04-12
milestone: v0.2
tags: [ai-models, design-review, architecture, automation, hero-features, v0.2]
horizon: next
---

# Second Model Design Review — Automated Second Set of Eyes

## Context

Idea captured from conversation: when an agent produces a design or approach, automatically send it to a second model for review and advisory feedback before proceeding. A structured, automated "second set of eyes" on design decisions.

## The Idea

After a primary model (e.g. Claude) produces a design proposal, architectural approach, or significant plan, Hero could automatically route that output to a second model for independent review. The second model critiques, surfaces blind spots, identifies risks, and advises — without having participated in the original reasoning chain.

This mirrors how engineering teams use design reviews: the author is too close to the work; a reviewer with fresh eyes catches what the author misses.

## Why This Matters

- Models have confirmation bias toward their own reasoning — a second model breaks that loop
- The reviewer hasn't been anchored by the first model's framing or assumptions
- Catches logical errors, missing edge cases, unconsidered alternatives
- Surfaces concerns the primary model rationalized away
- Adds a quality gate without requiring human review for every decision

## How It Could Work

```
primary model → produces design/approach
     ↓
hero review --approach <file-or-context>
     ↓
second model (same or different provider) receives:
  - the proposed design
  - relevant project context (conventions, decisions, existing architecture)
  - a review prompt: "Critique this. What's missing? What are the risks? What would you do differently?"
     ↓
review output → surfaced to user (or appended to spec as "Review Notes")
```

The second model could be:
- A different model from a different provider (e.g. GPT-4 reviewing a Claude design)
- The same model with a adversarial/critic role prompt
- A specialized reviewer agent (Hero already has `architecture-reviewer`, `security-reviewer`, etc.)

## Integration Points

- Could be a step in `/design` — after spec is generated, auto-trigger a review pass
- Could be an explicit `hero review --design <slug>` command
- Could be a flag: `hero design --with-review`
- The reviewer output could be written into the spec as a `## Review Notes` section

## Key Takeaways

- This is a natural extension of Hero's existing agent architecture (`architecture-reviewer`, `security-reviewer` agents already exist)
- The value is in the independence: reviewer hasn't seen the primary model's reasoning chain
- Works best with a different model/provider to avoid correlated blind spots
- Low implementation friction — Hero already routes context to agents; this is another routing pattern
- Could be optional/configurable: auto-review for designs above a certain complexity threshold

## Open Questions

- Should the review be blocking (must acknowledge before proceeding) or advisory (appended, non-blocking)?
- What's the right reviewer prompt to elicit genuine critique rather than validation?
- Should the primary model see the review and respond, creating a multi-turn design dialogue?
- Which Hero agents are the right reviewers for which design types?

---

## Follow-up: Model Role Config — Loose Harness-Level Assignment (2026-04-12)

The second model review idea extends naturally into a broader model role configuration system. Rather than hardcoding which model does what, Hero could support a loose, optional config where users declare roles for their models — and agent/command definitions tag themselves with the role they need. The harness auto-picks the right model at invocation time.

### The Config Shape

Fully optional. If not set, Hero falls back to whatever the user's AI tool (OpenCode, Cursor, etc.) has as its default.

```yaml
# .hero/config.yaml or hero.config.yaml
models:
  design: anthropic/claude-opus-4     # deep reasoning, long context
  execution: anthropic/claude-sonnet  # fast, cheap, good at code
  review: openai/gpt-4o               # independent reviewer, different provider
  # fallback: use whatever the harness default is
```

Or via OpenCode config, Cursor rules, etc. — Hero reads from wherever the user has already configured their models. The config is a layer on top of existing tool config, not a replacement.

### Agent/Command Definitions Tag Their Role

Each agent or command definition declares what model role it requires:

```yaml
# agents/architecture-reviewer.md frontmatter
---
name: architecture-reviewer
role: review          # ← Hero harness uses `models.review` for this agent
---
```

```yaml
# agents/greenfield-architect.md frontmatter
---
name: greenfield-architect
role: design          # ← Hero harness uses `models.design` for this agent
---
```

```yaml
# agents/engineer.md frontmatter
---
name: engineer
role: execution       # ← Hero harness uses `models.execution` for this agent
---
```

At invocation, the harness resolves: `agent.role → config.models[role] → model ID`. No manual wiring per-command.

### Why This Works

- **Loose coupling**: the config is optional; undefined roles fall back gracefully
- **Provider diversity built-in**: review model from a different provider than design model — correlated blind spots avoided by default if the user sets it up that way
- **One place to change**: swap your review model once, all reviewer agents pick it up
- **Portable**: the role tags live in the agent definitions (in the repo), the model assignments live in user config (not committed, or committed as a team default)
- **Composable with existing tool config**: OpenCode already has model config; Hero just needs to read it and layer role assignments on top

### Role Taxonomy (initial cut)

| Role | Purpose | Characteristics |
|---|---|---|
| `design` | Architectural thinking, spec generation, planning | High reasoning, long context, slow ok |
| `execution` | Code generation, file editing, implementation | Fast, cost-effective, code-optimized |
| `review` | Critique, adversarial analysis, second opinions | Independent provider preferred, strong reasoning |
| `research` | Investigation, codebase exploration, summarization | Good at retrieval and synthesis |
| `default` | Fallback for untagged agents | Whatever the harness default is |

### Key Takeaways

- Model role config should be **optional and additive** — zero config = still works, just no role optimization
- Agent definitions should declare their role in frontmatter — simple, readable, portable
- The harness resolves role → model at invocation time — no per-command wiring
- `review` role defaults to a different provider if possible — makes the independence guarantee structural rather than advisory
- This is infrastructure for the second-model-review feature AND a general capability for any multi-model workflow Hero supports in the future
