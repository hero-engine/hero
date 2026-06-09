---
title: "Harden Agent Loop Error Recovery (Stuck Turn Prevention)"
slug: hihcp-agent-loop-error-recovery
type: bug
status: planning
domain: engineering
size: medium
priority: high
created: 2026-06-09
tags: [hero-code, swift, agent-loop, error-recovery, stuck-turn, p1]
parent: hero-in-hero-code-parity
depends-on:
  - hihcp-mcp-first-turn-readiness
  - hihcp-mcp-auto-reconnect
---

# Harden Agent Loop Error Recovery (Stuck Turn Prevention)

## Issue

15+ `assistant_turn_errored` events with `loop_phase_before_clear:
"runningTools"` were observed in a 3-day audit period. Tool execution failures
crash turns without producing a tool result, leaving the agent loop stuck in the
`runningTools` phase. Users are forced to type "stuck?" or restart the session to
recover.

Parent initiative: `hero-in-hero-code-parity`.
Depends on: `hihcp-mcp-first-turn-readiness` and `hihcp-mcp-auto-reconnect`
(some stuck turns are MCP-caused; fix root causes first, then address residual).

## Scope -- design inputs for `/design`

Three complementary fixes:

1. **Top-level catch in tool execution.** Every tool call must produce a tool
   result, even on unexpected exceptions. Wrap the tool execution path in
   `AgentLoop.swift` with a catch-all that converts unhandled errors into a
   structured error tool result the model can read and act on.

2. **Watchdog timer for `runningTools` phase.** If the agent loop stays in
   `runningTools` for longer than a configurable timeout (e.g., 120 seconds),
   force a graceful transition: produce an error tool result for the stuck call
   and allow the model to continue. This is the safety net for cases where the
   tool call itself hangs (e.g., blocked I/O, deadlock).

3. **Clean error surfacing.** When a tool execution fails, surface the error in
   both the tool result (so the model can self-correct) and the UI (so the user
   sees what happened). The user should never need to type "stuck?" to discover
   that a turn failed.

**Files to touch:**
- `Engine/AgentLoop.swift` -- tool execution error handling, watchdog timer
- `ChatLoop/ChatLoopReducer.swift` -- phase transition logic for stuck recovery
- `ChatLoop/ChatLoopViewModel.swift` -- error surfacing to the UI

## Boundaries

- Do not change MCP server management (items 3-4 handle that)
- Do not retry failed tool calls automatically -- report the error and let the
  model decide
- Do not change the happy path -- only add safety nets around the failure path

## Risks

- Watchdog timer too aggressive: kills legitimate long-running tools (e.g., large
  `rg` search). The 120s timeout should be conservative enough.
- Error tool result format: must match Claude's expected tool_result schema or the
  model will be confused. Verify against the API spec.
- Phase transition safety: forcing a transition out of `runningTools` must not
  leave the state machine in an inconsistent state.

## Validation

- No turn hangs indefinitely; every tool failure produces a result
- Users never need to type "stuck?" to unblock the agent loop
- The model receives a structured error it can understand and act on
- The UI shows what went wrong when a tool call fails
- Legitimate long-running tools (under 120s) are not interrupted
