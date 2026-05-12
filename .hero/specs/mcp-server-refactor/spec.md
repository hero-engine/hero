---
title: MCP Server Refactor — Split the 3774-Line Monolith into Domain Files with Table-Driven Dispatch
type: feature
status: completed
tags: [mcp, refactor, quality, internal, no-behavior-change, testing]
created: 2026-05-01
relations:
  - target: two-tier-mcp-responses
    kind: related
horizon: now
---

## Kickoff

`internal/serve/mcp.go` is 3774 lines holding protocol types, lifecycle, dispatch, tool definitions, and ~40 tool implementations in one file. Split it into domain-scoped files with a table-driven dispatcher so adding or evolving a tool is a single registration, not an edit to a switch statement at line 800-something.

**Status:** planning — refactor lands immediately after `two-tier-mcp-responses` Phase 1, on top of those changes.

**Pick up at:** create the new files, move tool funcs, swap the switch to a handler map, run full test suite to confirm zero behavior change.

## Goal

Make the MCP server quality without changing what it does. After this refactor:

- Adding a tool = one file, one registration line
- Reviewing a tool = open one focused file, not scroll through 3700+ lines
- Test surface is unchanged; every existing test passes byte-identically against the new layout
- Code coverage on dispatcher and tool result wrapping improves because the small extracted helpers are easier to test in isolation

## Problem

`internal/serve/mcp.go` is a monolith:

- Protocol types, server struct, initialize/lifecycle handler
- `tools/list` and `tools/call` dispatchers
- `toolDefinitions()` — a 200+ line giant slice literal
- A switch statement dispatching ~40 tools
- ~40 tool function implementations covering reads, mutations, and analyses
- Result-wrapping plumbing duplicated at error and success paths

The size has real costs:
- Cognitive load: any change requires scrolling past unrelated code
- Diff noise: a tool change shows up alongside protocol work in `git blame`
- Test focus: `mcp_test.go` covers the whole surface; small helpers can't be tested in isolation
- Onboarding: new contributors face a wall of unfamiliar code in one file

## Design

### File split (no behavior change)

```
internal/serve/
├── mcp.go               (kept, slim — public surface, NewMCPServer, exports)
├── mcp_protocol.go      protocol types, JSON-RPC structs, capability declarations
├── mcp_lifecycle.go     initialize, instructions, capabilities negotiation
├── mcp_dispatch.go      tools/list, tools/call, handler map, error/success wrap
├── mcp_tools_def.go     toolDefinitions() — the giant slice literal
├── mcp_tools_read.go    read-side tools (context, search, status, list, read_spec, ask, recap, why, blocked, feed, knowledge, anchor, pulse, plan, brief, contract, impact)
├── mcp_tools_mutate.go  mutation tools (claim, event, kickoff, queue, skill_run, test_generate, demo_record, code, enrich)
├── mcp_tools_analyze.go analysis tools (drift, diagnose, diagnose_batch, score, error_pattern, velocity, check, nudge)
└── mcp_test.go          unchanged
```

Allocation rules:
- **Read** = returns a query result, no state change
- **Mutate** = writes to graph, index, files, or external systems
- **Analyze** = computes a derived view (may cache, but no semantic state change)

If a tool straddles categories, place it where its primary action sits. Document any judgment calls in a comment at top of the destination file.

### Table-driven dispatch

Replace the switch at line ~800 of current `mcp.go`:

```go
switch params.Name {
case "hero_context":
    result, toolErr = s.toolContext(params.Arguments)
case "hero_search":
    result, toolErr = s.toolSearch(params.Arguments)
// ... ~40 cases
}
```

With a registered handler map:

```go
type toolHandler func(args map[string]interface{}) (string, error)

func (s *MCPServer) toolHandlers() map[string]toolHandler {
    return map[string]toolHandler{
        "hero_context":       s.toolContext,
        "hero_search":        s.toolSearch,
        "hero_read_spec":     s.toolReadSpec,
        "hero_expand":        s.toolExpand,  // from two-tier work
        // ... one line per tool
    }
}

func (s *MCPServer) dispatchToolCall(req *MCPRequest, params ToolCallParams) {
    handlers := s.toolHandlers()
    handler, ok := handlers[params.Name]
    if !ok {
        s.sendError(req.ID, ErrCodeInvalidParams, fmt.Sprintf("Unknown tool: %s", params.Name))
        return
    }
    // ... existing filter check, call, wrap result
}
```

Adding a new tool becomes: write the handler in the appropriate `mcp_tools_*.go` file, add one line to the handler map, add the definition to `toolDefinitions()`. No more switch edits.

### Result-wrapping helper

Extract the duplicated success/error result wrapping:

```go
func (s *MCPServer) finishToolCall(req *MCPRequest, result string, err error) {
    if err != nil {
        s.sendResult(req.ID, ToolCallResult{
            Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Error: %v", err)}},
            IsError: true,
        })
        return
    }
    s.sendResult(req.ID, ToolCallResult{
        Content: []ToolContent{{Type: "text", Text: result}},
    })
}
```

Small thing, but the duplicated wrap appears in multiple call sites today.

### What does NOT change

- Public API: `NewMCPServer`, `NewMCPServerWithFilter`, `MCPServer` struct, exported types
- Tool names, signatures, return shapes
- Tool definitions content (descriptions, schemas) — exact text preserved
- Tool filter behavior
- Error codes, error messages
- Initialization sequence
- Test file structure (one file, `mcp_test.go`, may grow but tests don't move)

This is a pure refactor. Anything that changes behavior is out of scope and is a follow-up.

## Acceptance Criteria

- THE SYSTEM SHALL preserve every public symbol exported from `internal/serve/mcp.go` after the refactor
- THE SYSTEM SHALL preserve every tool name, signature, and response shape
- WHEN the existing test suite runs against the refactored code THE SYSTEM SHALL pass every test that passed before, with zero modification to test files
- WHEN a new tool is added THE SYSTEM SHALL require exactly one file edit (the appropriate `mcp_tools_*.go`) plus one entry in the handler map plus one entry in `toolDefinitions()`
- WHEN `tools/call` receives an unknown tool name THE SYSTEM SHALL return the same error code and message as before refactor
- WHEN `tools/list` is called THE SYSTEM SHALL return the identical tool list (same ordering, same descriptions, same schemas)
- WHILE the refactor is in progress THE SYSTEM SHALL keep `mcp.go` compilable at every commit (no broken intermediate states)
- IF a tool's category is ambiguous (read/mutate/analyze) THEN THE SYSTEM SHALL place it where its primary action sits and document the choice in a top-of-file comment
- THE SYSTEM SHALL extract a `finishToolCall` helper that consolidates success/error result wrapping
- THE SYSTEM SHALL replace the dispatch switch with a handler map keyed by tool name
- THE SYSTEM SHALL keep `mcp.go` itself under 500 lines after the refactor (sanity floor — exact target depends on what stays as the public-surface shim)

## Verification Plan

1. **Before refactor:** capture full test suite output to a baseline file
2. **After refactor:** run the same tests, diff against baseline → must be byte-identical pass set
3. **Tool inventory check:** run `tools/list` before and after, JSON-diff the output → must be identical
4. **Smoke test each domain:** invoke one read, one mutate, one analyze tool and confirm response shape unchanged
5. **Coverage delta:** confirm coverage on `internal/serve` does not drop. Stretch: improve coverage on extracted helpers.

## Out of Scope

- Changing tool descriptions or schemas (separate spec if needed)
- Adding new tools (this is a refactor)
- Changing tool filter logic or profile semantics
- Restructuring tool implementations internally
- Splitting `mcp_test.go` (test file restructure is its own decision)
- Renaming exported types or functions
- Performance optimization

## Risks

- **Regression risk:** mitigated by full test suite run + baseline diff. The whole point of the refactor is to be invisible to tests.
- **Merge conflict risk if other PRs touch `mcp.go`:** mitigate by landing immediately after `two-tier-mcp-responses` Phase 1, in a single atomic commit, on a clean branch.
- **Hidden coupling:** if a tool function reaches into private state we didn't realize was scoped to `mcp.go`, the move surfaces it. Resolve case-by-case during the move.
- **Reviewer fatigue:** the diff will look large. Mitigate with a clear commit message: "refactor only — no behavior change. Verified via full test suite + tools/list diff."

## Changes

- New files (split out of `mcp.go`):
  - `internal/serve/mcp_protocol.go`
  - `internal/serve/mcp_lifecycle.go`
  - `internal/serve/mcp_dispatch.go`
  - `internal/serve/mcp_tools_def.go`
  - `internal/serve/mcp_tools_read.go`
  - `internal/serve/mcp_tools_mutate.go`
  - `internal/serve/mcp_tools_analyze.go`
- Modified: `internal/serve/mcp.go` (slimmed to public-surface shim)
- Possibly modified: `internal/serve/mcp_test.go` (only if test imports need adjustment — prefer keeping test file untouched)
