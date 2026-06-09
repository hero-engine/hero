# Handoff prompt for hero-code

Paste this into a hero-code session:

---

Design and build native support for Hero commands, skills, and agents — the same way Claude Code handles them with its Skill tool. Right now hero-code's model sees the Hero routing table in the system prompt ("bug → /diagnose", "feature → /design") but has no tool to invoke those workflows. The model literally cannot execute Hero workflows. This is the single biggest gap in hero-code's Hero integration.

## What Claude Code does

Claude Code has a built-in `Skill` tool. When the model calls `Skill(skill: "diagnose", args: "checkout is broken")`, the tool:
1. Looks up "diagnose" in the command registry (`.claude/commands/diagnose.md`)
2. Reads the file body, replaces `$ARGUMENTS` with the args
3. Returns the expanded content as a tool result
4. The model follows those instructions using its other tools (Read, Edit, Bash, etc.)

It also loads skills (`.claude/skills/<name>/SKILL.md`) the same way — domain knowledge the model needs during a workflow.

## What hero-code needs

A `skill_run` native tool (or whatever you want to name it) that does the same thing. The model calls it by name, gets back workflow instructions or domain knowledge, and follows them.

It should handle all three content types:
- **Commands** (`.claude/commands/<name>.md`) — workflow entry points like design, deliver, diagnose. These have `$ARGUMENTS` that get expanded.
- **Skills** (`.claude/skills/<name>/SKILL.md`) — domain knowledge like spec-format, debugging-investigation, testing-and-validation. Static content, no argument expansion.
- **Agents** (`.claude/agents/<name>.md`) — role definitions like debug-investigator, feature-delivery-lead. These have system prompts and metadata.

## Architecture — what already exists

You already have all three registries built and loaded:

### Registries (all `@MainActor @Observable`)

**CommandRegistry** (`Engine/CommandRegistry.swift`, 133 lines)
- Scans `.hero/commands/` and `.claude/commands/`
- `expand(name:, arguments:) -> String?` — returns body with `$ARGUMENTS` replaced
- `allCommands` — lists all loaded commands
- Already loaded at startup via `AppState` (line 200)

**SkillRegistry** (`Engine/SkillRegistry.swift`, 87 lines)
- Scans `.hero/skills/` and `.claude/skills/` subdirectories for `SKILL.md`
- `skill(named:) -> SkillDefinition?` — returns skill by name
- `allSkills` — lists all loaded skills
- Already loaded at startup via `AppState` (line 202)

**AgentRegistry** (`Engine/AgentRegistry.swift`, 129 lines)
- Scans `.hero/agents/` and `.claude/agents/`
- `agent(named:) -> AgentDefinition?` — returns agent by name (has systemPrompt, model, temperature)
- `allAgents` — lists all loaded agents
- Already loaded at startup via `AppState` (line 201)

### Tool dispatch chain (`Engine/AgentLoop.swift`, line ~420-555)

The tool call routing in `AgentLoop.runLoop()` works by name matching:
1. `mcp__*` prefix → `MCPManager.callTool()` (line ~463)
2. `ask_user_question` → `onAskUser` callback (line ~483)
3. `request_plan_approval` → `onPlanApproval` callback (line ~495)
4. `spawn_agent` → `runSubAgent()` (line ~518)
5. Everything else → `toolExecutor.execute()` (line ~534)

The new `skill_run` tool should be intercepted at the AgentLoop level (between spawn_agent and toolExecutor), NOT added to ToolExecutor. Reason: the registries are `@MainActor` and AgentLoop is already `@MainActor`, so it can access them directly. ToolExecutor is a separate `actor` and can't access `@MainActor` properties without async bridging.

### Tool spec catalog (`Engine/AgentLoop.swift`, line ~1258)

`nativeToolSpecs()` is a static method returning an array of `ToolSpec`. Currently has 30 `spec()` calls. Add the new tool here with proper description and parameters.

### Wiring points (`State/AppState.swift`)

- Line 200-202: Registries are created
- Line 694: `agentLoop.mcpManager = mcpManager`
- Line 955: `agentLoop.toolExecutor = ToolExecutor(root: root)`
- Line 740, 1783: `loop.toolExecutor = agentLoop.toolExecutor` (sub-agent loops)

You'll need to add `commandRegistry`, `skillRegistry`, and `agentRegistry` properties to AgentLoop and wire them from AppState at the same points where toolExecutor is wired (including sub-agent loops at lines 740 and 1783).

## Design guidance

**Lookup order:** commands first, skills second, agents third. Commands and skills don't collide today (commands are verbs like `design`; skills are noun-phrases like `spec-format`), but commands should win if they ever do — they're the workflow entry points.

**Permission:** Auto-allow. This is a read-only content-loading operation. The actual work happens when the model follows the returned instructions using Write, Edit, Bash — which have their own permission gates.

**Error on miss:** Return available options so the model can self-correct:
```
Unknown command, skill, or agent 'foo'.
Available commands: blocked, capture, challenge, check, compose, ...
Available skills: agent-reliability, debugging-investigation, ...
Available agents: debug-investigator, feature-delivery-lead, ...
```

**Tool description matters a lot.** The description should make clear that the result is *instructions to follow*, not documentation to summarize. Something like: "Load a Hero workflow command, skill, or agent by name. Returns instructions to follow using your other tools."

**There's already an MCP tool `hero_skill_run`** on the Hero MCP server that handles skills. But it depends on the MCP server being connected (which has reliability issues on first turn — `ensureReady` race). The native tool is the reliable, always-available path. Both can coexist.

## What success looks like

1. User says "this checkout flow is broken"
2. Model sees AGENTS.md routing: "bug → /diagnose"
3. Model calls `skill_run(skill: "diagnose", args: "checkout flow is broken")`
4. Tool returns the diagnose.md workflow instructions with $ARGUMENTS expanded
5. Model follows the workflow: loads debug-investigator agent, investigates, writes spec
6. Hero workflows work in hero-code just like they do in Claude Code

## Also fix: AGENTS.md vs CLAUDE.md

`ProjectContext.swift` loads `CLAUDE.md` with higher priority than `AGENTS.md` (line ~192: `CLAUDE.md > AGENTS.md > .hero/hero.md`). The CLAUDE.md in this repo has Claude Code-specific instructions (ToolSearch, Explore agent, deferred tool loading) that don't work in hero-code. Either:
- Make ProjectContext prefer AGENTS.md over CLAUDE.md for hero-code
- Or fix AGENTS.md to be the authoritative source with hero-code-appropriate instructions (reference `skill_run` instead of Claude Code's `Skill` tool)
