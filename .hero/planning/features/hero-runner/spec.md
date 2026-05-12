---
title: Hero Runner — Headless Agent Execution via Claude API
type: feature
status: planning
priority: P0
tags: [runner, headless, agent, automation, platform]
created: 2026-04-23
relations:
  - target: hero-platform
    kind: parent
horizon: next
smoke: deferred
---

## Goal

Add `hero run` — a new mode that drives agent work by calling the Claude
API directly, without a chat UI. Same specs, same tools, same context —
but the loop runner is Hero itself, not Claude Code or Cursor.

## Problem

Today an agent can only do work inside an interactive session. There's no
way to say "fix this bug" and walk away. The Anthropic SDK and Claude API
support tool-using agent loops, but Hero has no entry point that uses them.
Every other mode (MCP, serve, CLI) exists — the headless execution mode is
the gap.

## Design

### `hero run` command

```bash
hero run deliver csv-export                    # deliver a spec headlessly
hero run deliver csv-export --autopilot        # suppress confirmations
hero run diagnose login-crash                  # diagnose a bug
hero run "<natural language request>"          # NL routing, same as /do
hero run --model claude-sonnet-4-6             # override model
hero run --max-turns 50                        # limit agent loop iterations
hero run --budget 5.00                         # cost cap in dollars
hero run --dry-run                             # show what would execute
```

### Agent loop

The core loop is straightforward:

```go
func RunAgent(cfg RunConfig) error {
    // 1. Build system prompt from agent definition + AGENTS.md
    systemPrompt := buildSystemPrompt(cfg)

    // 2. Build initial user message from spec/command
    userMessage := buildUserMessage(cfg)

    // 3. Register Hero tools (same functions as MCP, but in-process)
    tools := registerTools(cfg)

    // 4. Loop: call Claude API → execute tool calls → send results
    messages := []Message{{Role: "user", Content: userMessage}}
    for turn := 0; turn < cfg.MaxTurns; turn++ {
        response := callClaude(cfg.Model, systemPrompt, messages, tools)

        if response.StopReason == "end_turn" {
            break // agent is done
        }

        // Execute each tool call
        for _, toolCall := range response.ToolUse {
            result := executeToolInProcess(toolCall)
            messages = append(messages, toolResultMessage(toolCall.ID, result))
        }

        // Check budget
        if cfg.Budget > 0 && totalCost > cfg.Budget {
            break
        }
    }

    return commitResults(cfg)
}
```

### Tool execution — in-process, not MCP

When Hero runs headlessly, tools are Go function calls in the same binary.
No JSON-RPC, no stdin/stdout serialization. The `toolContext()`,
`toolSearch()`, `toolDrift()` etc. methods on `MCPServer` already exist —
`hero run` calls them directly (or refactors them into a shared `tools`
package that both MCP and runner use).

The agent also needs code-writing tools:
- **File read/write** — `os.ReadFile` / `os.WriteFile`
- **Shell execution** — `exec.Command` for `go build`, `go test`, `git commit`
- **Git operations** — commit, diff, branch, push

These are standard tool definitions the Claude API understands.

### Output

```
hero run deliver csv-export

Running: deliver csv-export (autopilot mode)
Model: claude-sonnet-4-6
Budget: unlimited
Max turns: 100

[turn 1] Reading spec...
[turn 2] Running hero_context...
[turn 3] Writing internal/export/csv.go...
...
[turn 18] Running go test ./internal/export/...
[turn 19] All tests pass. Running hero drift csv-export...
[turn 20] No drift. Running hero complete...

✓ Delivered csv-export in 20 turns ($0.84)
  Commit: abc1234 "feat: CSV export with streaming for large files"
  Files: 3 created, 1 modified
  Tests: 4 added, all passing
```

### Multi-provider support

The runner is model-agnostic via a provider interface:

```go
type LLMProvider interface {
    Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
    Name() string
}
```

Built-in providers:

| Provider | Flag | Env var | Models |
|---|---|---|---|
| Anthropic | `--provider anthropic` (default) | `ANTHROPIC_API_KEY` | claude-sonnet-4-6, claude-opus-4-6 |
| OpenAI | `--provider openai` | `OPENAI_API_KEY` | gpt-4o, o3 |
| Azure OpenAI | `--provider azure` | `AZURE_OPENAI_KEY` + `AZURE_OPENAI_ENDPOINT` | gpt-4o (Azure-hosted) |

```bash
hero run deliver csv-export                          # default: anthropic
hero run deliver csv-export --provider openai        # use OpenAI
hero run deliver csv-export --provider azure         # use Azure OpenAI
hero run deliver csv-export --model gpt-4o           # auto-detects provider from model name
```

Each provider translates Hero's tool definitions into the provider's
native format (Claude tool_use vs OpenAI function_calling). The
translation is mechanical — same JSON Schema, different wire format.

### API key management

```bash
hero login                     # stores key in OS keychain / ~/.hero/credentials
ANTHROPIC_API_KEY=sk-...       # env var override
hero run --api-key sk-...      # flag override (not recommended)
```

`hero login` already exists for cloud auth. Extend it to also store
API keys for runner use. When running against a team server, the server
holds the org API key — individual developers don't need one.

Key resolution order:
1. `--api-key` flag (not recommended, visible in process list)
2. Provider-specific env var (`ANTHROPIC_API_KEY`, `OPENAI_API_KEY`)
3. Team server credential (if connected via `hero connect team`)
4. Stored credential from `hero login`

### Job tracking

Each `hero run` invocation creates a job record in `.hero/jobs/<id>.json`
with: start time, spec slug, model, status (running/completed/failed),
turns used, cost, commit hash. `hero jobs` lists recent jobs.
`hero jobs <id>` shows the full log.

This is the foundation the team server's job queue builds on.

### Safety

- `--max-turns` defaults to 100 — prevents runaway loops
- `--budget` caps API cost — stops when exceeded
- File writes are sandboxed to the project directory
- Shell commands run with the same permissions as the user
- `--dry-run` shows the execution plan without doing anything
- All git operations are on a branch, never force-push to main
- The agent uses the same AGENTS.md rules, conventions, and boundaries
  as interactive sessions — no special permissions

## Changes

- `internal/runner/runner.go` — agent loop, tool registry, job tracking
- `internal/runner/provider.go` — `LLMProvider` interface and provider registry
- `internal/runner/anthropic.go` — Anthropic Claude API provider
- `internal/runner/openai.go` — OpenAI / Azure OpenAI provider
- `internal/runner/tools.go` — in-process tool implementations (file I/O, shell, git)
- `internal/runner/runner_test.go` — unit tests for tool dispatch, budget tracking, turn limiting, provider selection
- `internal/cli/run.go` — `hero run` command with flags
- `internal/cli/jobs.go` — `hero jobs` command for listing/inspecting job history
- `internal/cli/root.go` — register `runCmd`, `jobsCmd`

## Acceptance Criteria

- WHEN `hero run deliver <slug>` is called with a valid API key THE SYSTEM SHALL execute the delivery agent loop headlessly and produce commits
- WHEN `hero run` exceeds `--max-turns` THE SYSTEM SHALL halt the agent loop and report partial progress
- WHEN `hero run` exceeds `--budget` THE SYSTEM SHALL halt the agent loop and report cost consumed
- WHEN `hero run --dry-run` is called THE SYSTEM SHALL display the execution plan without making any API calls or file changes
- WHEN `hero run` completes THE SYSTEM SHALL write a job record to `.hero/jobs/` with status, turns, cost, and commit hash
- WHEN `hero jobs` runs THE SYSTEM SHALL list recent job records with their status and summary
- WHEN `hero run` is called without an API key THE SYSTEM SHALL exit with a clear error message pointing to `hero login` or the provider's env var
- WHEN `hero run --provider openai` is called THE SYSTEM SHALL use the OpenAI API with function_calling format for tool invocations
- WHEN `hero run --provider azure` is called THE SYSTEM SHALL use the Azure OpenAI endpoint with the configured deployment
- WHEN `hero run --model gpt-4o` is called without `--provider` THE SYSTEM SHALL auto-detect the provider as OpenAI from the model name
- THE SYSTEM SHALL execute all Hero tools in-process without requiring an MCP server
- THE SYSTEM SHALL respect the same AGENTS.md rules, conventions, and boundaries as interactive sessions

## Boundaries

- Does **not** provide its own LLM — calls the Claude API (or compatible endpoint)
- Does **not** require Hero Cloud — works locally with an API key
- Does **not** manage git branches automatically — the agent uses git commands like in interactive mode
- Does **not** support parallel job execution in this feature — that's hero-team-server
- Does **not** expose a web UI — that's hero-dashboard-v2
