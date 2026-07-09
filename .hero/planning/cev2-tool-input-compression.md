---
title: "Tool input compression"
slug: cev2-tool-input-compression
type: feature
status: superseded
superseded_by: tool-input-compression
superseded_by_peer:
  peer_id: cd8dd06d-3df1-4878-a88f-24593dcbb4b3
  peer_alias_display: hero-code
  peer_slug: tool-input-compression
  successor_initiative: context-engine-v3
  reason: "Scope moved to hero-code's context-engine-v3 initiative."
priority: medium
size: small
parent: context-engine-v2
created: 2026-06-09
relations:
  - target: tool-input-compression
    kind: superseded-by
tags: [hero-code, swift, context-engine, curator, compression, tool-use, token-efficiency]
---

# Tool input compression

## Context

When the model issues a tool call, the `tool_use` block in the assistant
message contains the full `arguments` field -- the complete JSON payload
sent to the tool. For Bash tool calls, this includes the entire script
or command. For Edit tool calls, this includes the full `old_string` and
`new_string`. For Write tool calls, this includes the entire file content.

After `toLLMMessages()` expands assistant turns, these `arguments` fields
persist at full size in every subsequent model call. Once the tool result
is back, the arguments' informational value drops near zero -- the result
tells the model what happened, and the arguments are just a record of what
was asked. But the token cost remains.

In the analyzed ~50K token conversation, tool input arguments accounted
for a significant fraction of wasted tokens -- hundreds of lines of Bash
scripts, file contents, and edit payloads preserved verbatim across every
subsequent turn.

## Goal

Compress the `arguments` field on non-verbatim `tool_use` blocks to a
compact summary (tool name + first line of command/content), reducing
token cost while preserving the structural fields required by the API
contract (tool_call id, function name).

## Approach

The compression runs in the curator's assembly pass (after tagging,
dedup/supersede, and budget pruning, but before final assembly). It
operates on assistant messages that:
- Are NOT in the verbatim window (recent tool calls need full arguments
  for the model to understand its latest work).
- Contain `toolCalls` with non-empty `arguments`.
- Have a matching `tool` result message later in the conversation
  (confirming the tool completed -- compressing an in-flight tool call
  would lose the request context).

The compression replaces the `arguments` string with a one-line summary:
`"{tool_name}: {first_line_of_args_truncated_to_80_chars}"`. The `id` and
`function.name` fields are preserved unchanged.

## Changes

All files are in `../hero-code/apps/hero-desktop-mac/Sources/HeroDesktop/Engine/`.

1. **Add tool input compression to the curation pipeline**
   (`ContextCurator.swift`)
   - Add a new pass after dedup/supersede and before budget pruning (or
     during the assembly pass -- both work). This pass iterates over
     `mutableMessages` and, for each assistant message with `toolCalls`:
     - Skip if the message index is in the verbatim window.
     - For each `LLMToolCall` in `toolCalls`:
       - Check that a matching tool result exists later in the
         conversation (scan forward for `role == "tool"` with matching
         `toolCallId`).
       - If match found, replace `arguments` with a compressed summary:
         `"{function.name}: {first_meaningful_line(arguments)}"`.
       - `first_meaningful_line()`: parse `arguments` as JSON, extract
         the most informative field (`command` for Bash, `file_path` for
         Read/Write/Edit, `query` for grep/glob), take the first 80
         characters.
     - Track the token savings in the `AnnotatedContextPlan` (add a
       `compressedToolInputTokens` field or similar).

2. **Preserve API contract integrity**
   - The Anthropic API requires `tool_use` blocks to have: `id` (string),
     `type: "tool_use"`, `name` (string), `input` (object). The `input`
     field maps to `arguments` in `LLMToolCall.function.arguments`.
   - The compressed arguments must be valid JSON. Wrap the summary string
     in a JSON object: `{"_compressed": "{tool_name}: {summary}"}`.
   - Alternative: keep `arguments` as a raw string (it's already a string
     in `LLMFunctionCall`). The API sends it as-is. Verify that the
     provider correctly handles a short string in `arguments` without
     error.

3. **Add compression toggle to `CurationOptions`**
   - Add `var compressToolInputs: Bool = true` to `CurationOptions`.
   - Wire it in `AgentLoop.outgoing()` alongside the existing options.
   - Default on. Can be disabled for debugging or if the API rejects
     compressed arguments.

4. **Update port-fidelity ledger** (`ContextCurator.swift` lines 17-22)
   - Add a fidelity note:
     `//  - [cev2] Tool input arguments compressed on non-verbatim turns (v1 has no tool input compression).`

## Boundaries

- Do NOT compress tool results (the `content` field on `role: "tool"`
  messages). That is what dedup/supersede handles. This spec only
  compresses the `arguments` field on the assistant's `tool_use` block.
- Do NOT compress verbatim-window messages. Recent tool calls need full
  arguments for the model to reason about its latest work.
- Do NOT compress tool calls that lack a matching result. If the tool
  call is in-flight (no result yet), the full arguments are needed for
  the model to understand the pending request.
- Do NOT introduce SwiftUI/AppKit dependencies.

## Risks

- **API contract violation.** If the provider expects `arguments` to be
  parseable as the original tool input schema, compressed arguments may
  cause an API error. Mitigation: test with real provider calls. The
  Anthropic API sends `tool_use` blocks in the request, and the model
  reads them back. The model is robust to seeing compressed prior tool
  inputs -- it has the result and doesn't need to re-parse the arguments.
- **tool_use/tool_result ID pairing.** The `id` field must be preserved
  unchanged. The compression only touches `arguments`, never `id` or
  `function.name`. Verify with a test that the pairing survives.
- **Loss of debugging information.** Engineers inspecting the context
  plan won't see the original arguments. Mitigation: the original
  arguments are preserved in the session's stored transcript
  (the curator operates on a derived copy, not the persisted conversation).
  The context inspector UI can always show the original.

## Validation

- **Unit test -- arguments compressed:** Create a conversation with an
  assistant tool call + tool result, outside the verbatim window. Verify
  the curated output has compressed arguments (short string) and preserved
  `id` and `function.name`.
- **Unit test -- verbatim protection:** Same setup but within the verbatim
  window. Verify arguments are NOT compressed.
- **Unit test -- no matching result:** An assistant tool call without a
  following tool result. Verify arguments are NOT compressed (the tool
  call may be in-flight).
- **Unit test -- token savings:** Measure the token estimate before and
  after compression on a message with a large `arguments` field (e.g.,
  1000-character Bash script). Verify meaningful reduction.
- **Unit test -- API contract:** Verify the compressed `LLMMessage`
  preserves all structural fields required by the provider: `id`,
  `function.name`, `arguments` (non-empty string).
- **Existing tests must still pass.** The `test_tokenEstimateIncludesToolCalls`
  test uses explicit character counts for arguments -- it operates on raw
  messages before curation, so it should not be affected by the compression
  pass.
