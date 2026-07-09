---
title: "System prompt curator"
slug: cev2-system-prompt-curator
type: feature
status: superseded
superseded_by: system-prompt-curation
superseded_by_peer:
  peer_id: cd8dd06d-3df1-4878-a88f-24593dcbb4b3
  peer_alias_display: hero-code
  peer_slug: system-prompt-curation
  successor_initiative: context-engine-v3
  reason: "Scope moved to hero-code's context-engine-v3 initiative; successor slug renamed to system-prompt-curation."
priority: medium
size: large
parent: context-engine-v2
depends-on:
  - cev2-context-engine-test-harness
created: 2026-06-09
relations:
  - target: system-prompt-curation
    kind: superseded-by
tags: [hero-code, swift, context-engine, system-prompt, curation, token-efficiency, high-risk]
---

# System prompt curator

## Context

The context curator (`ContextCurator.swift`) only operates on conversation
messages. The system prompt -- assembled by `HeroContextAssembler` and
injected via `AgentLoop.runTurn()` as the `systemPrompt` parameter to
`provider.stream()` -- passes through uncurated at full size every turn.

In the analyzed ~50K token conversation, the system prompt was ~24K tokens.
Of those, approximately 14K tokens were wasted on:

- **Unused tool schemas** -- tool definitions for tools the model never
  called in the session (e.g., computer-use tools in a pure coding session,
  file-upload tools when no files were uploaded).
- **Irrelevant skill lists** -- the full `<system-reminder>` skill list
  including dozens of skills not relevant to the current task.
- **Computer-use instructions** -- multi-paragraph behavioral instructions
  for browser automation, teach mode, and screen interaction in sessions
  that never touch a screen.
- **Stale cached file reads** -- file contents cached in the system prompt
  that have since been superseded by explicit `Read` tool calls in the
  conversation.

The system prompt is the single largest token consumer. Curating it has the
highest absolute token savings potential -- but also the highest risk of
behavioral regression if important instructions are pruned.

## Goal

Reduce the system prompt from ~24K tokens to ~10K tokens by removing
content that is demonstrably irrelevant to the current session. The
reduction is gated by a feature flag and rolled out incrementally. The model
retains all instructions relevant to its current task and available tools.

## Approach

This spec requires a design phase before implementation. The key design
questions are:

### What sections are safe to prune?

The system prompt has distinct sections with different safety profiles:

| Section | Safe to prune? | Signal |
|---------|---------------|--------|
| Tool schemas for unused tools | Yes, with care | Tool not in active tool catalog |
| Skill list entries | Probably | Skill not loaded in current session |
| Computer-use behavioral instructions | Yes, when no screen tools | No computer-use tools in catalog |
| Chrome MCP instructions | Yes, when no Chrome tools | No Chrome MCP tools in catalog |
| Cached file contents (CLAUDE.md, AGENTS.md) | Partially | File re-read in conversation supersedes |
| Core behavioral instructions | NO | Always needed |
| Safety/privacy rules | NO | Always needed |
| Memory/preferences | NO | Always needed |

### How to detect relevance?

Two approaches, combinable:

1. **Static analysis at assembly time.** When `HeroContextAssembler` builds
   the system prompt, it knows which tools are in the catalog, which skills
   are loaded, and which domain is active. It can omit sections that are
   provably irrelevant. This is the safest approach -- no risk of pruning
   something that's needed.

2. **Dynamic curation per turn.** Like the conversation curator, inspect
   the system prompt content and prune sections based on the conversation
   so far (e.g., if no computer-use tools have been called in 10 turns,
   prune the computer-use instructions). This is riskier -- the model might
   need those instructions in a future turn.

Recommend starting with approach 1 (static analysis at assembly time)
and deferring approach 2.

### Feature-flag gated rollout

Given the behavioral regression risk, the system prompt curator must be
behind a feature flag with a gradual rollout:

1. **Phase A: Instrument.** Add token counting to the system prompt
   sections. Log which sections are present and their token costs. No
   pruning yet.
2. **Phase B: Tool schema pruning.** Remove schemas for tools not in
   the active catalog. Feature-flagged off by default.
3. **Phase C: Instruction section pruning.** Remove behavioral instruction
   blocks (computer-use, Chrome MCP, teach mode) when the corresponding
   tools are absent. Feature-flagged off by default.
4. **Phase D: Stale cache pruning.** Remove cached file contents that have
   been superseded by explicit reads in the conversation. Feature-flagged
   off by default.

Each phase ships independently with its own flag. Flags are enabled in
sequence after validation that the prior phase causes no regressions.

## Changes

Files are in `../hero-code/apps/hero-desktop-mac/Sources/HeroDesktop/`.

1. **Add system prompt section tracking to `HeroContextAssembler`**
   (`Engine/HeroContextAssembler.swift` or equivalent)
   - Break the system prompt into named sections with boundaries
     (start/end markers or an array of section structs).
   - Track each section's token cost using `ContextCurator.estimateMessageTokens()`.
   - Log section inventory to the context snapshot (`onContextSnapshot`).

2. **Add tool catalog awareness to the assembler**
   - At assembly time, receive the active tool catalog (the list of
     tool specs available for this turn).
   - Omit tool schema sections for tools not in the catalog.
   - Omit behavioral instruction sections whose corresponding tools
     are absent (e.g., computer-use instructions when no computer-use
     tools are registered).

3. **Add stale cache detection**
   - Compare cached file contents in the system prompt against file paths
     that have been `Read` in the conversation.
   - If a cached file has been re-read, the conversation copy is more
     recent -- remove the cached version from the system prompt.
   - This requires access to the conversation history at assembly time
     (the assembler needs to see the message list, not just the tool
     catalog).

4. **Add feature flags**
   - `systemPromptToolSchemaPruning: Bool` (default: false)
   - `systemPromptInstructionPruning: Bool` (default: false)
   - `systemPromptStaleCachePruning: Bool` (default: false)
   - Wire through `SettingsStore` -> `AgentLoop` -> `HeroContextAssembler`.

5. **Update port-fidelity ledger** (`ContextCurator.swift` lines 17-22)
   - Add a fidelity note:
     `//  - [cev2] System prompt curation added (v1 has no system prompt curation).`

## Boundaries

- Do NOT prune core behavioral instructions, safety/privacy rules, or
  memory/preferences. These are always needed regardless of context.
- Do NOT implement dynamic per-turn curation of the system prompt in this
  spec. Start with static analysis at assembly time only.
- Do NOT modify the conversation curator. System prompt curation is a
  parallel concern with its own code path.
- Do NOT change the `LLMMessage` struct or the `ChatProvider` interface.
  The system prompt is passed as a separate `systemPrompt: String`
  parameter to `provider.stream()`.

## Risks

- **Behavioral regression.** The highest risk in the entire initiative.
  Pruning system prompt content can change model behavior in subtle ways
  that are hard to detect with unit tests. Mitigation: feature flags with
  gradual rollout, and a comprehensive regression test suite comparing
  model behavior with and without pruning on a set of representative
  tasks.
- **Section boundary fragility.** The system prompt is assembled from
  multiple sources (CLAUDE.md, skill content, tool schemas, MCP server
  instructions). Identifying section boundaries reliably requires
  structured assembly (not string concatenation). If the assembler uses
  string concatenation today, it must be refactored to section-based
  assembly first.
- **Tool catalog completeness.** If the tool catalog is incomplete at
  assembly time (e.g., MCP tools haven't loaded yet due to the first-turn
  readiness race), the assembler might prune schemas for tools that will
  become available on the next turn. Mitigation: only prune tools that
  are provably absent (not in any registered MCP server's manifest), not
  tools that are merely slow to load.
- **Stale cache false positives.** A `Read` of a file in the conversation
  doesn't necessarily mean the system prompt's cached version is stale --
  the conversation Read might be from a different point in time, or the
  system prompt version might be more recent. Mitigation: always prefer
  the conversation version (it's what the model has seen most recently).

## Validation

- **Instrumentation validation:** With Phase A deployed, verify that
  section inventory is logged correctly and token counts match
  `estimateMessageTokens()` output.
- **Tool schema pruning validation:** With Phase B enabled, verify that:
  - Tool schemas for active tools are present in the system prompt.
  - Tool schemas for absent tools are removed.
  - The model can still invoke all active tools correctly.
  - Token count is reduced by the expected amount.
- **Instruction pruning validation:** With Phase C enabled, verify that:
  - Computer-use instructions are absent when no computer-use tools exist.
  - The model does not attempt to use computer-use actions.
  - Coding behavior is unchanged in pure coding sessions.
- **Stale cache pruning validation:** With Phase D enabled, verify that:
  - A file cached in the system prompt and later `Read` in the conversation
    is removed from the system prompt.
  - The model references the conversation version, not the stale cache.
- **End-to-end regression suite:** Run a set of 5-10 representative tasks
  (coding, debugging, file manipulation, tool-heavy, tool-light) with all
  pruning flags enabled and verify equivalent model behavior vs. baseline.
- **Token reduction target:** System prompt drops from ~24K to ~10K tokens
  in a pure coding session (no computer-use, no Chrome MCP). Measure and
  report.
