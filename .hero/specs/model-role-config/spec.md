---
title: Model Role Config — Loose Role-Based Model Assignment
slug: model-role-config
type: feature
status: completed
milestone: v0.2
tags: [ai-models, config, roles, routing, multi-model]
created: 2026-04-12
relations:
  - target: second-model-review
    kind: dependency-of
  - target: mcp-tool-filtering
    kind: related
horizon: now
completed_at: 2026-05-18T19:25:38Z
---

## Goal

Hero agents and commands have different computational needs: a spec generator needs deep reasoning and long context; a code writer needs speed and cost efficiency; a design reviewer needs independence from the primary reasoning chain. Today there's no way to express these preferences — all agents run on whatever model the user's AI tool happens to be using.

Model role config is a lightweight, optional system that lets teams declare which model to use for which purpose, and lets agent definitions tag themselves with the role they need. The Hero harness resolves `agent.role → config.models[role] → model ID` at invocation time, with zero config required to get started.

## Context

This feature is sourced from the `second-model-design-review` knowledge note, which identified model role config as the infrastructure needed to support the second-model-review feature and any future multi-model workflow. It's designed to be additive: zero config = still works, with no role optimization. Full config = purpose-built models for each workload class.

## Design

### Config Shape

In `hero.json` or a separate `hero.config.yaml`:

```yaml
# hero.config.yaml (optional — not committed, or committed as team default)
models:
  design: anthropic/claude-opus-4      # deep reasoning, long context, slow ok
  execution: anthropic/claude-sonnet-4  # fast, cost-effective, code-optimized
  review: openai/gpt-4o                 # independent reviewer, different provider
  research: anthropic/claude-haiku      # fast retrieval and summarization
  # fallback: use whatever the harness default is
```

Or in `hero.json` under a `models` key:

```json
{
  "models": {
    "design": "anthropic/claude-opus-4",
    "execution": "anthropic/claude-sonnet-4",
    "review": "openai/gpt-4o",
    "research": "anthropic/claude-haiku"
  }
}
```

The config is fully optional. Any unset role falls back to the harness default (whatever the AI tool — OpenCode, Cursor, Claude Code — has configured). Partial config is valid: you can set only `review` to get independent review routing without touching the others.

### Agent Role Tags

Each agent or command definition declares its role in frontmatter:

```yaml
# agents/greenfield-architect.md
---
name: greenfield-architect
role: design
---
```

```yaml
# agents/engineer.md
---
name: engineer
role: execution
---
```

```yaml
# agents/design-reviewer.md
---
name: design-reviewer
role: review
---
```

```yaml
# agents/codebase-explorer.md
---
name: codebase-explorer
role: research
---
```

At invocation, the Hero harness resolves: `agent.role → config.models[role] → model ID`. No per-command wiring. Agents that don't declare a role use the `default` (harness default).

### Role Taxonomy

| Role | Purpose | Optimal Characteristics |
|---|---|---|
| `design` | Architectural thinking, spec generation, planning | High reasoning, long context window, slow is ok |
| `execution` | Code generation, file editing, implementation | Fast, cost-effective, code-optimized |
| `review` | Critique, adversarial analysis, second opinions | Independent provider preferred, strong reasoning |
| `research` | Codebase exploration, investigation, summarization | Good at retrieval and synthesis, fast |
| `default` | Fallback for untagged agents/commands | Whatever the harness default is |

The `review` role is special: its value is maximized when it's a *different provider* than `design`, avoiding correlated blind spots. The config schema allows this naturally — there's no constraint requiring same-provider models.

### Harness Integration

Hero is used as a context layer by AI tools (OpenCode, Cursor, Claude Code) — it does not execute model calls itself. Model routing must be surfaced to the harness/tool layer.

The integration mechanism depends on the AI tool:

**OpenCode**: Model selection can be expressed in the agent definition's YAML frontmatter. Hero would emit role-resolved model IDs into agent files at install/update time:
```yaml
# agents/design-reviewer.md (after role resolution)
---
name: design-reviewer
model: openai/gpt-4o   # resolved from models.review config
---
```

**Cursor**: `.cursor/rules/` agent definitions support model selection in newer versions. Hero writes resolved model IDs into agent rule files.

**Claude Code**: Agent system prompts can include model preference hints, though Claude Code controls the actual model selection.

**Fallback**: If the harness doesn't support model-per-agent selection, `hero context` prepends a model recommendation comment to the context block: `<!-- Recommended model for this agent: review → openai/gpt-4o -->`. The user sees the recommendation even if it can't be auto-applied.

### `hero models` Command

A new CLI subcommand to inspect the current model role configuration:

```bash
hero models
# Output:
# Model Role Configuration
# design:    anthropic/claude-opus-4   (configured)
# execution: anthropic/claude-sonnet-4 (configured)
# review:    openai/gpt-4o             (configured)
# research:  anthropic/claude-haiku    (configured)
# default:   (harness default)

hero models --check
# Validates that configured model IDs are reachable for each role
# (pings the relevant provider API)

hero models apply
# Writes resolved model IDs into all agent definition files
# (the mechanism depends on the installed harness)
```

### Agent Role Index

When `hero install` runs or `hero models apply` runs, Hero scans all agent definitions, reads their `role` frontmatter, and writes a role index to `.hero/models.json`:

```json
{
  "roles": {
    "design": {
      "model": "anthropic/claude-opus-4",
      "agents": ["greenfield-architect", "spec-writer", "sprint-planner"]
    },
    "execution": {
      "model": "anthropic/claude-sonnet-4",
      "agents": ["engineer", "refactorer", "test-writer"]
    },
    "review": {
      "model": "openai/gpt-4o",
      "agents": ["design-reviewer", "architecture-reviewer", "security-reviewer"]
    }
  }
}
```

This index is used by `hero models`, by the MCP tool filtering layer (for role-based tool access), and by any future harness integration that needs to resolve agent → model.

## Changes

- `internal/config/config.go` — `models` config section with role taxonomy
- `internal/models/roles.go` — role resolution logic (config lookup, fallback chain)
- `internal/models/index.go` — agent role index builder (scans agents, writes `.hero/models.json`)
- `internal/cli/models.go` — `hero models` command with `--check` and `apply` subcommands
- `internal/cli/install.go` — `hero install` triggers role index build
- Agent definitions — add `role:` frontmatter to all agents that have a natural role

## Acceptance Criteria

- `hero.json` `models` section accepts role → model ID mappings
- Agents with `role:` frontmatter are resolved to the configured model at invocation
- Undefined roles fall back to the harness default (no error, no degraded behavior)
- `hero models` displays current role configuration with configured/default distinction
- `hero models apply` writes resolved model IDs into agent definition files
- `hero install` rebuilds the role index when agent definitions change
- Partial config (e.g., only `review` role set) works without requiring all roles to be configured
- The `review` role defaults to a note recommending a different provider in the generated context

## Boundaries

- Does **not** make model API calls itself — Hero is a context layer, not a model orchestrator
- Does **not** enforce model selection — the harness/tool controls actual model routing; Hero provides the configuration and hints
- Does **not** support per-command model overrides at runtime (config-time only)
- Role taxonomy is fixed for v0.2 — custom roles are a future extension
