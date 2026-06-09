# hero-code: skill_run dispatches at AgentLoop, not ToolExecutor

**Date:** 2026-06-09
**Spec:** hihcp-skill-run-tool
**Context:** Designing where to intercept the `skill_run` native tool in hero-code's dispatch chain.

## Decision

`skill_run` is intercepted in `AgentLoop` (alongside `spawn_agent`, `ask_user_question`, `request_plan_approval`) — NOT dispatched through `ToolExecutor.execute()`.

## Rationale

1. `CommandRegistry` and `SkillRegistry` are `@MainActor @Observable`. `AgentLoop` is `@MainActor` and can access them directly. `ToolExecutor` is a separate `actor` — accessing `@MainActor` properties requires async bridging.

2. Precedent: all tools that need app-layer resources are intercepted in AgentLoop. `ToolExecutor` handles pure filesystem/shell/git operations.

3. Minimality: no changes to ToolExecutor, no new actor isolation bridging, no new protocol requirements.

## Pattern

The hero-code tool dispatch chain (in `AgentLoop.runLoop`) routes by name:
1. `mcp__*` prefix → `MCPManager.callTool()`
2. `ask_user_question` → `onAskUser` callback
3. `request_plan_approval` → `onPlanApproval` callback
4. `spawn_agent` → `runSubAgent()`
5. **`skill_run` → `resolveSkillRun()` (NEW)**
6. Everything else → `ToolExecutor.execute()`

Tools in positions 2-5 need app-layer resources (UI handlers, registries) that ToolExecutor doesn't have.
