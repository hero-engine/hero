---
title: "Add skill_run Tool to hero-code Native Tool Catalog"
slug: hihcp-skill-run-tool
type: feature
status: handed_off
domain: engineering
size: medium
priority: critical
created: 2026-06-09
tags: [hero-code, swift, tool-catalog, skill, workflow, p0]
parent: hero-in-hero-code-parity
---

# Add skill_run Tool to hero-code Native Tool Catalog

## Goal

The model can invoke any Hero workflow command or skill via a native
`skill_run` tool — no MCP server dependency, no filesystem reads, no
guessing. One tool call returns the workflow instructions; the model
follows them using its existing tools.

## Kickoff

Adds a `skill_run` native tool to hero-code so the model can invoke
Hero workflows (/design, /deliver, /diagnose, etc.).

**Status:** planning — full design complete, no code yet.

**Pick up at:** Add the tool spec to `nativeToolSpecs()` in AgentLoop,
then the dispatch intercept and registry wiring. Three files, one new
`else if` branch in the tool dispatch chain.

→ `/deliver hihcp-skill-run-tool`

**Files:** `Engine/AgentLoop.swift:1258` (nativeToolSpecs), `Engine/AgentLoop.swift:518` (dispatch chain), `State/AppState.swift:955` (registry wiring)
**Skip:** ToolExecutor dispatch (registries are @MainActor, ToolExecutor is an actor — use AgentLoop intercept instead)

## Problem

hero-code's system prompt tells the model to use Hero slash commands
(`/design`, `/deliver`, `/diagnose`), but the model has no tool to
invoke them. `AgentLoop.nativeToolSpecs()` defines 27 tools — none is
a skill or command invocation tool. The model sees routing instructions
it cannot follow, falls back to unstructured investigation, and never
produces structured spec output. This is the single biggest blocker for
Hero workflow integration in the desktop app.

### Why not just use the MCP `hero_skill_run` tool?

A `hero_skill_run` MCP tool already exists on the Hero server. It
handles skills but not commands, and depends on the MCP server being
connected — the very thing items 3-4 of this initiative fix. On first
turns (before MCP readiness), and during server disconnects, the MCP
tool is unavailable. The native tool is the reliable, always-available
path.

### Prior art: Codex bridge

The Codex harness solved the same problem differently (commit
`afe9553`). Codex's `SlashCommand` is a built-in Rust enum
(`codex-rs/tui/src/slash_command.rs`) — you can't define new slash
commands, though Codex does support skills, agents, hooks, and
AGENTS.md. The workaround: materialize each Hero command as a Codex
skill file (`.agents/skills/command-<name>/SKILL.md`) and teach
AGENTS.md to tell the model to Read and follow those files.

hero-code has a programmable tool catalog (`nativeToolSpecs()`), so the
proper solution is a native tool — the model calls `skill_run` and
gets the content back as a tool result, with no filesystem round-trip
and no risk of the model treating instructions as documentation.

## Design

### Tool spec

Add to `AgentLoop.nativeToolSpecs()`:

```swift
spec("skill_run",
     "Load a Hero workflow command or skill by name and return its instructions. "
     + "Use this to run Hero workflows like /design, /deliver, /diagnose. "
     + "The returned text contains step-by-step instructions to follow using your other tools.",
     props: [
        "skill": strProp("Name of the command or skill (e.g. 'design', 'deliver', 'diagnose')"),
        "args": strProp("Optional context or arguments to pass to the workflow"),
     ],
     required: ["skill"])
```

Key description choices:
- Names the primary workflows explicitly so the model knows what values to pass
- States that the result is *instructions to follow*, not documentation
- Uses `skill` as the parameter name (not `command`) since the model's
  system prompt uses "skill" vocabulary from the Hero surface layer

### Dispatch — AgentLoop intercept

Route `skill_run` in AgentLoop's tool dispatch chain, alongside the
other intercepted tools (`ask_user_question`, `request_plan_approval`,
`spawn_agent`). This is the right layer because:

1. **Registry access.** `CommandRegistry` and `SkillRegistry` are
   `@MainActor @Observable`. `AgentLoop` is already `@MainActor`, so
   it can read them directly. `ToolExecutor` is a separate `actor` and
   cannot access `@MainActor` properties without async bridging.

2. **Precedent.** All tools that need app-layer resources (UI handlers,
   sub-agent spawning) are intercepted in AgentLoop before reaching
   ToolExecutor. `skill_run` needs registry resources.

3. **No ToolExecutor changes.** The existing ToolExecutor `default:`
   case already throws `ToolError.unknownTool` for unrecognized names.
   Adding `skill_run` to ToolExecutor would require threading the
   registries through actor boundaries.

Insert the `skill_run` branch **after** `spawn_agent` and **before**
the `toolExecutor` fallthrough (line ~534):

```swift
} else if call.name == "skill_run" {
    let argsDict = (try? JSONSerialization.jsonObject(
        with: Data(call.arguments.utf8)
    ) as? [String: Any]) ?? [:]
    let skillName = argsDict["skill"] as? String ?? ""
    let skillArgs = argsDict["args"] as? String ?? ""
    result = resolveSkillRun(name: skillName, arguments: skillArgs)
```

### Lookup logic — `resolveSkillRun`

New private method on `AgentLoop`:

```swift
private func resolveSkillRun(name: String, arguments: String) -> String {
    // 1. Try commands first — commands are full workflows (/design, /deliver)
    if let body = commandRegistry?.expand(name: name, arguments: arguments) {
        return body
    }

    // 2. Try skills — skills are domain knowledge (spec-format, testing-and-validation)
    if let skill = skillRegistry?.skill(named: name) {
        // Skills don't have $ARGUMENTS expansion, just return content
        return skill.content
    }

    // 3. Name miss — return available options
    let cmdNames = commandRegistry?.allCommands.map(\.name).sorted() ?? []
    let skillNames = skillRegistry?.allSkills.map(\.name).sorted() ?? []
    var msg = "Unknown command or skill '\(name)'."
    if !cmdNames.isEmpty {
        msg += "\n\nAvailable commands: \(cmdNames.joined(separator: ", "))"
    }
    if !skillNames.isEmpty {
        msg += "\n\nAvailable skills: \(skillNames.joined(separator: ", "))"
    }
    return msg
}
```

**Resolution order: commands first, skills second.** Rationale:
- Commands are full workflows (`/design`, `/deliver`, `/diagnose`)
  that the model invokes from the routing table. They're the primary
  use case.
- Skills are domain knowledge (`spec-format`, `testing-and-validation`)
  loaded for reference during a workflow.
- No name collision exists today (commands are verbs like `design`;
  skills are noun-phrases like `spec-format`). If a collision were
  introduced, the command should win because workflow invocation is
  the tool's primary purpose.

### Registry wiring

Add two optional properties to `AgentLoop`:

```swift
var commandRegistry: CommandRegistry?
var skillRegistry: SkillRegistry?
```

Wire them in `AppState` alongside the existing `toolExecutor` assignment
(around line 955):

```swift
agentLoop.commandRegistry = commandRegistry
agentLoop.skillRegistry = skillRegistry
```

Also wire them for sub-agent loops — check `runSubAgent` and any place
a child `AgentLoop` is constructed (the child needs skill_run too):

```swift
loop.commandRegistry = commandRegistry
loop.skillRegistry = skillRegistry
```

### $ARGUMENTS expansion

`CommandRegistry.expand(name:, arguments:)` already replaces
`$ARGUMENTS` in the command body with the provided arguments string.
This is exactly what we need — when the model calls
`skill_run(skill: "diagnose", args: "user sees 500 on checkout")`,
the `args` value replaces `$ARGUMENTS` in `diagnose.md`'s body.

Skills don't use `$ARGUMENTS` — their content is static domain
knowledge. The `args` parameter is ignored for skills (the content
is returned as-is).

### Permission classification

`skill_run` should be classified as **auto-allow** in the permission
system. It's a read-only content-loading operation — no filesystem
writes, no shell execution, no network calls. The actual work happens
when the model follows the returned instructions using Write, Edit,
Bash, etc., which have their own permission gates.

In `PermissionTrustStore` or the trust classification logic, add
`skill_run` to the auto-allow set alongside `Read`, `list_directory`,
`git_status`, etc.

## Changes

### `AgentLoop.swift`

1. **`nativeToolSpecs()`** (line ~1258) — Add `skill_run` spec as the
   28th tool, placed in a new `// ── Hero ──` section after the
   Phase 3 tools.

2. **Tool dispatch chain** (line ~534) — Add `else if call.name ==
   "skill_run"` branch after `spawn_agent`, before the `toolExecutor`
   fallthrough.

3. **`resolveSkillRun(name:arguments:)`** — New private method
   implementing the lookup logic described above.

4. **Properties** — Add `var commandRegistry: CommandRegistry?` and
   `var skillRegistry: SkillRegistry?`.

### `AppState.swift`

1. **Registry wiring** (~line 955) — After `agentLoop.toolExecutor =
   ToolExecutor(root: root)`, add:
   ```swift
   agentLoop.commandRegistry = commandRegistry
   agentLoop.skillRegistry = skillRegistry
   ```

2. **Sub-agent wiring** — Anywhere a child AgentLoop is constructed
   (search for `loop.toolExecutor = agentLoop.toolExecutor`), add:
   ```swift
   loop.commandRegistry = commandRegistry
   loop.skillRegistry = skillRegistry
   ```

### No changes to:

- **`ToolExecutor.swift`** — skill_run is intercepted in AgentLoop,
  never reaches ToolExecutor.
- **`CommandRegistry.swift`** — Already has `expand(name:, arguments:)`
  and `allCommands`. No new methods needed.
- **`SkillRegistry.swift`** — Already has `skill(named:)` and
  `allSkills`. No new methods needed.
- **`hero/registry.rs`** (Rust) — The Swift registries scan the same
  filesystem directories. No FFI needed.
- **MCP tools** — `hero_skill_run` MCP tool remains as a secondary path;
  this spec adds the native primary path.

## Acceptance Criteria

WHEN the model calls `skill_run` with a valid command name THE SYSTEM
SHALL return the command body with `$ARGUMENTS` expanded and the model
SHALL receive it as a tool result.

WHEN the model calls `skill_run` with a valid skill name THE SYSTEM
SHALL return the skill content as a tool result.

WHEN the model calls `skill_run` with a name that matches both a
command and a skill THE SYSTEM SHALL return the command (commands take
precedence).

WHEN the model calls `skill_run` with an unknown name THE SYSTEM SHALL
return an error message listing all available commands and skills.

WHEN the model calls `skill_run` with `args` and a command name THE
SYSTEM SHALL replace `$ARGUMENTS` in the command body with the provided
args value.

THE SYSTEM SHALL classify `skill_run` as auto-allow in the permission
system (no user approval prompt).

WHEN a sub-agent loop is spawned THE SYSTEM SHALL wire `commandRegistry`
and `skillRegistry` to the child loop so sub-agents can also invoke
`skill_run`.

## Test Plan

1. **Unit: resolveSkillRun** — Test command lookup, skill lookup,
   command precedence over skill on collision, name-miss error with
   available options listing, $ARGUMENTS expansion.

2. **Unit: nativeToolSpecs** — Verify `skill_run` appears in the tool
   catalog with correct parameter schema.

3. **Integration: tool dispatch** — Simulate a tool call with
   `name: "skill_run"` and verify it reaches the AgentLoop intercept
   (not ToolExecutor).

4. **Integration: sub-agent** — Verify child AgentLoop inherits
   registry references.

5. **E2E: manual smoke** — In a hero-code session, say "diagnose a bug"
   and verify: (a) model calls `skill_run(skill: "diagnose", ...)`,
   (b) receives the diagnose workflow, (c) follows it to produce a spec.

## Boundaries

- No sub-agent isolation in v1 — content-loading only. The workflow
  text is returned as a tool result and the model follows it inline.
  Upgrade to sub-agent isolation if context-window pressure demands it.
- No content caching between calls — stateless per-invocation. The
  registries are already loaded in memory; lookup is O(n) on a list
  of ~30 commands and ~50 skills.
- No changes to the Rust ContentRegistry or FFI layer.
- No changes to MCP tools — this is purely a native tool addition.

## Risks

- **Tool description quality.** If the description doesn't clearly
  convey that the result is *instructions to follow*, the model may
  treat it as documentation and summarize rather than execute. Mitigate
  by including "follow using your other tools" in the description.
- **Large command bodies.** `deliver.md` is 278 lines (~10KB). Verify
  this fits comfortably in the tool result without truncation. The
  existing tool result pipeline handles up to 200KB (ToolExecutor's
  readFile truncation limit), so this is fine.
- **Registry not loaded yet.** If `skill_run` is called before the
  registries finish their async scan, `commandRegistry?.expand()` returns
  nil. The name-miss path handles this gracefully (lists empty sets).
  In practice, registry scans complete in <100ms and the model's first
  turn hasn't started tool calls yet.

## Handoff Trail

- 2026-06-09T21:23:18Z — out → hero-code (peer_id: cd8dd06d-3df1-4878-a88f-24593dcbb4b3)
  mode: async-drop
  originating_spec: hihcp-skill-run-tool
  peer_spec: hero-code/hihcp-skill-run-tool
  at_commit: 3e792fb
  reason: "hero-code needs native skill_run tool to invoke Hero workflows — commands, skills, and agents. All registries already exist (CommandRegistry, SkillRegistry, AgentRegistry). Need to add tool spec to nativeToolSpecs(), dispatch intercept in AgentLoop, and wire registries from AppState. Full architecture details and line numbers in the spec."

