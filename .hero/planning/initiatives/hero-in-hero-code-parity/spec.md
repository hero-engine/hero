---
title: "Hero-in-Hero-Code Parity — Fix Hero Workflow Integration in the Desktop App"
slug: hero-in-hero-code-parity
type: initiative
status: planning
domain: engineering
size: x-large
priority: critical
created: 2026-06-09
tags: [hero-code, swift, desktop, mcp, tool-catalog, reliability, dx]
child:
  - hihcp-skill-run-tool
  - hihcp-agents-md-harness-agnostic
  - hihcp-mcp-first-turn-readiness
  - hihcp-mcp-auto-reconnect
  - hihcp-agent-loop-error-recovery
  - hihcp-rgignore
  - hihcp-fuzzy-path-resolution
  - hihcp-permission-bridge-validation
---

# Hero-in-Hero-Code Parity — Fix Hero Workflow Integration in the Desktop App

## Vision

Hero workflows (`/design`, `/deliver`, `/diagnose`, etc.) work identically in
hero-code as they do in Claude Code. The model can invoke workflows, Hero MCP
tools are available from the first turn, the agent loop recovers from errors
without hanging, and common DX irritants (rg timeouts, broken path resolution,
crash-on-malformed-permission) are eliminated. When all eight items ship, a user
opening hero-code on a Hero-managed repo has the same workflow experience they
get in Claude Code.

## Problem

A deep audit of hero-code desktop app sessions (2026-06-06 through 2026-06-08)
revealed that Hero integration is broken at multiple levels:

1. **Workflows cannot execute.** The model's system prompt tells it to use Hero
   slash commands via a `Skill` tool, but hero-code's native tool catalog
   (`AgentLoop.nativeToolSpecs()`) has no such tool. The model literally cannot
   invoke `/design`, `/deliver`, or `/diagnose`. This is the P0 blocker.

2. **System prompt references the wrong harness.** `CLAUDE.md` contains
   Claude Code-specific concepts (ToolSearch, deferred tool loading, Explore
   agent) that do not exist in hero-code. The model follows instructions it
   cannot satisfy, producing confused output and wasted turns.

3. **MCP race condition.** Hero MCP tools (`hero_search`, `hero_list`,
   `hero_read_spec`, etc.) are missing on the first turn because `ensureReady`
   is only called from `callTool()`, not during tool catalog construction. The
   model's first turn sees no Hero tools and falls back to `grep` against
   `.hero/`.

4. **No MCP auto-reconnect.** When the Hero MCP server dies mid-session,
   `handleDisconnect` sets state to `.errored` but never attempts recovery. The
   session is permanently degraded.

5. **Stuck turns.** 15+ `assistant_turn_errored` events with
   `loop_phase_before_clear: "runningTools"` in the audit period. Tool execution
   failures crash turns without producing a tool result, leaving the agent loop
   stuck until the user manually intervenes.

6. **rg timeouts.** `rg` searches against the workspace scan `.build/`,
   `DerivedData/`, and other heavyweight directories, producing 10-second
   timeouts. No `.rgignore` exists.

7. **Broken path resolution.** The model passes bare filenames
   (`AgentLoop.swift`) that `resolvePath()` cannot locate because it expects
   absolute or workspace-relative paths. Every `Read` call on a bare filename
   fails.

8. **Permission bridge crashes.** Malformed permission payloads crash the agent
   loop instead of producing a denial with explanation.

## Goal

Close all eight gaps so that Hero workflows in hero-code reach functional parity
with Claude Code. "Parity" means: the model can invoke any Hero workflow, Hero
MCP tools are available from turn one, the session is resilient to MCP
disconnects and tool failures, and common DX operations (search, file reads,
permissions) work without friction.

## Architecture reference

The hero-code desktop app (Swift + Rust):

- **Swift app:** `apps/hero-desktop-mac/Sources/HeroDesktop/`
  - `Engine/AgentLoop.swift` -- native tool specs, tool execution, streaming
  - `Engine/HeroContextAssembler.swift` -- Hero system prompt layers
  - `Engine/ProjectContext.swift` -- loads AGENTS.md/CLAUDE.md
  - `Engine/ToolExecutor.swift` -- dispatches tool calls
  - `ChatLoop/` -- chat loop state machine (ChatLoopReducer, ChatLoopViewModel)
- **Rust:** `crates/hero-core/src/`
  - `hero/embedded.rs` -- embedded Hero content at build time
  - `hero/registry.rs` -- ContentRegistry (embedded + user items)
  - `session_runtime.rs` -- streaming, tools, sub-agents
- **Content:** `domains/<id>/{agents,skills,commands}/` -- embedded at build time
- **Config:** `~/.config/hero-code/` -- app state, providers, sessions

The hero-code repo lives at `/Users/developer/projects/hero-engine/repository/hero-code/`.

## Specs

Eight sequenced child specs across three waves. Wave 1 is blocking -- without it,
Hero workflows literally cannot run. Wave 2 addresses reliability. Wave 3 is DX
polish.

---

### Wave 1 -- Hero workflows can execute (P0, blocking)

These two items unblock all Hero workflow usage. Item 2 depends on item 1.

#### 1. Add `skill_run` tool to the native tool catalog
**Slug:** `hihcp-skill-run-tool` | **Type:** feature | **Size:** medium |
**Deps:** none (foundation) | **SHIPS FIRST**

The model needs a tool to invoke Hero slash command workflows. Currently
`AgentLoop.nativeToolSpecs()` has 27 tools but no Skill/skill_run tool. Add one
with params `skill: string` (required), `args: string` (optional). On call, look
up the skill/command in the registries (CommandRegistry, SkillRegistry), load its
content, return it as a tool result. The model then follows the loaded workflow
instructions using its existing tools (Read, Edit, Bash, Hero MCP tools).

**Files:** `Engine/AgentLoop.swift` (add tool spec to `nativeToolSpecs()`),
`Engine/ToolExecutor.swift` (add dispatch case), CommandRegistry + SkillRegistry
(need accessor methods to look up by name and return content).

**Open question:** Should `skill_run` load commands, skills, or both? Recommend
both, preferring commands on name collision (commands are full workflows, skills
are domain knowledge).

**Open question:** Does `skill_run` need sub-agent isolation or is
content-loading sufficient? Start with content-loading (return the expanded
workflow text as a tool result, let the model follow it inline). Upgrade to
sub-agent isolation later if context-window pressure demands it.

---

#### 2. Produce a harness-agnostic AGENTS.md, demote CLAUDE.md
**Slug:** `hihcp-agents-md-harness-agnostic` | **Type:** bug | **Size:** small |
**Deps:** item 1 (AGENTS.md should reference the tool that now exists)

`CLAUDE.md` has Claude Code-specific instructions (ToolSearch, Explore agent,
deferred tool loading) that do not work in hero-code. `AGENTS.md` is a skeleton.
Need `AGENTS.md` to be the authoritative source with harness-agnostic routing
that references `skill_run` tool instead of Claude Code's `Skill` tool.

**Files:** hero-code `AGENTS.md` (rewrite with harness-agnostic instructions),
`CLAUDE.md` (strip or delete hero:managed block). Also in the hero repo: teach
the snapshot emitter to write to `AGENTS.md` when detecting a non-Claude-Code
harness.

**Cross-repo note:** This item touches both hero-code (the `AGENTS.md` rewrite)
and the hero repo (snapshot emitter change). The hero repo change ensures
future `hero install` runs emit the right file for the detected harness.

---

### Wave 2 -- MCP reliability (P1)

These three items can be delivered in parallel after Wave 1 ships. Items 3 and 4
may partially resolve item 5 (stuck turns often correlate with MCP failures).

#### 3. Gate first turn on Hero MCP readiness
**Slug:** `hihcp-mcp-first-turn-readiness` | **Type:** bug | **Size:** small |
**Deps:** none (independent of Wave 1, but lower priority)

`ensureReady` is only called from `callTool()`, so the first turn's tool catalog
may miss all Hero MCP tools. Gate tool catalog construction on MCP readiness with
a short timeout (e.g., 3 seconds). On timeout, proceed with the tools that are
available and log a warning.

**Files:** `Engine/AgentLoop.swift` (tool catalog construction),
`Engine/MCPManager.swift` (readiness check).

---

#### 4. Auto-recover from MCP server disconnect mid-session
**Slug:** `hihcp-mcp-auto-reconnect` | **Type:** bug | **Size:** small |
**Deps:** none

`handleDisconnect` sets state to `.errored` but does not auto-respawn. Add
proactive reconnection on unexpected disconnect with exponential backoff. Three
consecutive failures within a window trigger a cooldown period; surface the
degradation to the user.

**Files:** `Engine/MCPManager.swift`.

---

#### 5. Harden agent loop error recovery (stuck turn prevention)
**Slug:** `hihcp-agent-loop-error-recovery` | **Type:** bug | **Size:** medium |
**Deps:** items 3-4 (may partially fix this; deliver after to measure residual)

15+ `assistant_turn_errored` with `loop_phase_before_clear: "runningTools"` in 3
days. Tool execution failures crash turns without clean recovery. Need: (a)
top-level catch in tool execution that always produces a tool result (even on
unexpected exceptions), (b) watchdog timer for the `runningTools` phase that
forces a graceful error result after a configurable timeout, (c) clean error
surfacing so the user sees what failed without needing to type "stuck?" to
unblock.

**Files:** `Engine/AgentLoop.swift` (tool execution error handling),
`ChatLoop/ChatLoopReducer.swift` (phase transitions),
`ChatLoop/ChatLoopViewModel.swift` (error surfacing to UI).

**Open question:** Are stuck turns correlated with MCP failures? Items 3-4 may
partially fix item 5. Deliver 3-4 first, measure residual stuck-turn rate, then
address remaining cases in item 5.

---

### Wave 3 -- Performance and DX (P2)

These three items are independent of each other and of earlier waves. They can be
delivered in any order, or in parallel.

#### 6. Add `.rgignore` to hero-code repo
**Slug:** `hihcp-rgignore` | **Type:** bug | **Size:** small |
**Deps:** none

Create `.rgignore` at the repo root excluding `.build/`, `build/`,
`DerivedData/`, `.hero/cache/`, `.hero/sessions/`, and any other heavyweight
generated directories. This is a single-file change.

**Files:** `.rgignore` (new file at repo root).

---

#### 7. Add workspace-relative path fuzzy resolution
**Slug:** `hihcp-fuzzy-path-resolution` | **Type:** feature | **Size:** small |
**Deps:** none

Enhance `resolvePath()` in `ToolExecutor.swift` to detect bare filenames and
search the workspace tree for a unique match. If multiple matches exist, return a
disambiguation error listing the candidates. This fixes the common case where the
model passes `AgentLoop.swift` instead of the full path.

**Files:** `Engine/ToolExecutor.swift` (`resolvePath()` enhancement).

---

#### 8. Harden permission bridge payload validation
**Slug:** `hihcp-permission-bridge-validation` | **Type:** bug | **Size:** small |
**Deps:** none

Validate permission payloads defensively in the agent loop's permission bridge
methods. Malformed payloads should produce a denial with a human-readable
explanation, not a crash. Add guard clauses and structured error returns.

**Files:** `Engine/AgentLoop.swift` (permission bridge methods, 1-2 call sites).

## Dependencies

```
Wave 1 (P0, blocking)
  1. hihcp-skill-run-tool         (foundation, SHIPS FIRST)
  └─> 2. hihcp-agents-md-harness-agnostic  (references the tool from item 1)

Wave 2 (P1, reliability)  — all three can start after Wave 1
  3. hihcp-mcp-first-turn-readiness   (independent)
  4. hihcp-mcp-auto-reconnect         (independent)
  └─> 5. hihcp-agent-loop-error-recovery  (deliver after 3+4 to measure residual)

Wave 3 (P2, DX)  — independent of each other and earlier waves
  6. hihcp-rgignore                    (independent)
  7. hihcp-fuzzy-path-resolution       (independent)
  8. hihcp-permission-bridge-validation (independent)
```

No hard external scheduling gates. Wave 2 items 3-4 may partially resolve item
5, so the recommended sequence is 3 and 4 first, then measure, then 5. Wave 3
items are trivial-to-small and can be interleaved as quick wins during Wave 1 or
Wave 2 delivery.

## Cross-cutting concerns and shared risks

**R1 -- Cross-repo coordination (hero + hero-code).** Item 2
(AGENTS.md) touches both the hero-code repo and the hero repo (snapshot
emitter). The hero repo change must be designed so that `hero install` emits
the correct file (AGENTS.md vs. CLAUDE.md) based on detected harness. Coordinate
the two changes so neither lands in isolation -- a hero-code AGENTS.md that
references `skill_run` is useless if the hero repo still emits CLAUDE.md for
hero-code installs.

**R2 -- Tool catalog contract stability.** Adding `skill_run` (item 1) changes
the tool catalog the model sees. If the tool spec is wrong (bad parameter
schema, missing description), the model will misuse it on every turn. Validate
the tool spec against Claude's tool-use documentation before shipping. Test
with real workflows (`/design`, `/diagnose`, `/deliver`) end-to-end.

**R3 -- MCP reliability compounding.** Items 3, 4, and 5 are partially
overlapping failure modes. A first-turn MCP miss (item 3) can cascade into a
tool execution failure that hangs the turn (item 5). Similarly, an MCP disconnect
(item 4) without recovery can produce the same stuck-turn symptom. The risk is
fixing item 5 symptomatically (e.g., adding a watchdog) without closing the
root causes in items 3-4. Recommended sequence: fix 3 and 4 first, then measure
the residual stuck-turn rate before investing in item 5.

**R4 -- AGENTS.md drift.** Once AGENTS.md becomes the authoritative source (item
2), it must be maintained as the hero repo evolves. If new workflows, tools, or
routing rules are added to hero, the AGENTS.md emitter must pick them up. Risk:
AGENTS.md becomes stale and the hero-code model falls behind. Mitigation: the
snapshot emitter in the hero repo generates AGENTS.md from the same source data
it uses for CLAUDE.md -- same data, different rendering.

**R5 -- Regression in Claude Code.** Item 2 demotes CLAUDE.md. If the hero repo
change is not carefully scoped, it could regress Claude Code users who depend on
CLAUDE.md's current content. Mitigation: CLAUDE.md is not deleted, only stripped
of the hero:managed block. Claude Code-specific features (ToolSearch, deferred
tools, Explore agent) remain in CLAUDE.md. The harness-agnostic content moves to
AGENTS.md, which both harnesses read.

## Open questions

1. **Should `skill_run` load commands, skills, or both?** Recommend: both, prefer
   commands on name collision. Commands are full workflows; skills are domain
   knowledge. The model should be able to load either.

2. **What happens to CLAUDE.md's hero:managed block?** The hero repo needs to
   emit to AGENTS.md for non-Claude harnesses. The hero:managed block in
   CLAUDE.md should be stripped for hero-code; Claude Code keeps its version.

3. **Are stuck turns correlated with MCP failures?** Items 3-4 may partially fix
   item 5. Deliver in sequence and measure.

4. **Does `skill_run` need sub-agent isolation or is content-loading sufficient?**
   Start with content-loading (simpler, proven pattern in Claude Code's Skill
   tool). Upgrade to sub-agent isolation later if context-window pressure demands
   it.

## Recommended delivery order

1. **Item 1 (skill_run tool) -- deliver first, unblocks everything.** This is the
   single highest-leverage change. Without it, Hero workflows are impossible.
   Medium effort, 3-4 files, clear acceptance criteria.

2. **Item 2 (AGENTS.md) -- deliver second, immediately after item 1.** Completes
   the Wave 1 pair. The model can now invoke workflows AND receives instructions
   it can actually follow.

3. **Item 6 (rgignore) -- quick win, interleave anytime.** Single file, trivial
   effort, immediate DX improvement. Ship it as a warm-up or between waves.

4. **Items 3+4 (MCP readiness + auto-reconnect) -- deliver together or in rapid
   sequence.** Both are small, both improve the same subsystem (MCPManager), and
   fixing them first informs item 5.

5. **Item 5 (error recovery) -- deliver after 3+4, measure first.** Some stuck
   turns may be MCP-caused. Fix the root causes, measure the residual, then
   harden the agent loop for the remaining cases.

6. **Items 7+8 (fuzzy paths + permission validation) -- deliver last or
   interleave as quick wins.** Both are small, independent, and lower priority.
   Good candidates for interleaving during Wave 2 delivery or for a cleanup pass
   after the main reliability work.

## Progress

- 2026-06-09 -- Initiative spec authored from desktop app session audit findings.
  Eight child stubs sequenced across three waves. No code yet. Next: `/design
  hihcp-skill-run-tool` to flesh out the P0 feature spec.
