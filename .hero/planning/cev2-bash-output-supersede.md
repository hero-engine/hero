---
title: "Bash output supersede"
slug: cev2-bash-output-supersede
type: feature
status: superseded
superseded_by: bash-output-supersede
superseded_by_peer:
  peer_id: cd8dd06d-3df1-4878-a88f-24593dcbb4b3
  peer_alias_display: hero-code
  peer_slug: bash-output-supersede
  successor_initiative: context-engine-v3
  reason: "Scope moved to hero-code's context-engine-v3 initiative; implementation is native Swift and lives in the peer workspace."
priority: high
size: medium
parent: context-engine-v2
created: 2026-06-09
relations:
  - target: bash-output-supersede
    kind: superseded-by
tags: [hero-code, swift, context-engine, curator, supersede, bash, token-efficiency]
---

# Bash output supersede

## Context

The curator's dedup/supersede pass (`ContextCurator.swift` curate function,
lines 183-199+) only processes tool outputs whose tool name is in
`pathWhitelist`:

```swift
static let pathWhitelist: Set<String> = ["Read", "Write", "Edit", "grep", "glob"]
```

These tools have a natural supersede key: the file path. When the same file
is read twice, the later read supersedes the earlier one.

Bash tool outputs -- which dominate real coding sessions -- are not in the
whitelist and never participate in supersede. In the analyzed ~50K token
conversation, Bash outputs accounted for a significant fraction of retained
tokens despite many being stale (e.g., re-running the same test suite,
re-reading the same git log, re-checking the same build output).

## Goal

Bash tool outputs participate in supersede, keyed on the command string
rather than a file path. When the same or sufficiently similar Bash command
runs again, the older output is superseded by the newer one. This
eliminates the largest category of token waste from tool outputs that
accumulate without bound.

## Approach

Bash commands don't have a clean "file path" to key on, but they do have the
command string itself in the `arguments` field of the preceding assistant
message's `tool_use` block. The approach:

1. Extract the Bash command from the `tool_use` arguments.
2. Normalize it (trim whitespace, collapse runs of whitespace).
3. Use the normalized command as the supersede key, analogous to how
   `extractToolPath()` provides the key for Read/Write/Edit/grep/glob.
4. When two Bash tool results have the same normalized command key, the
   later one supersedes the earlier one.

Start conservative: exact normalized command match only. Future work could
add fuzzy matching (e.g., `cat foo.txt` and `cat -n foo.txt` are "similar
enough") but that introduces false-positive risk.

## Changes

All files are in `../hero-code/apps/hero-desktop-mac/Sources/HeroDesktop/Engine/`.

1. **Add Bash command extraction function** (`ContextCurator.swift`)
   - Add a static function `extractBashCommand(_ msg: LLMMessage, messages: [LLMMessage], index: Int) -> String?`
     (or similar) that:
     - Checks if `msg.role == "tool"` and `msg.name == "Bash"`.
     - Looks backward in `messages` for the preceding assistant message
       that issued this tool call (matching by `tool_call_id`).
     - Extracts the `command` field from the assistant's `toolCalls`
       arguments JSON.
     - Normalizes: trim whitespace, collapse internal whitespace runs.
     - Returns the normalized command string, or nil if extraction fails.
   - Alternative approach: extract the command from the tool result content
     itself. Many Bash tool results echo the command in the first line
     (e.g., `$ git status\n...`). This is fragile but avoids the backward
     lookup. Recommend the arguments-based approach for reliability.

2. **Integrate Bash supersede into the dedup/supersede loop**
   (`ContextCurator.swift` curate function, dedup/supersede section)
   - In the existing `for idx in 0..<count` loop over `.toolOutput`
     messages:
     - After the existing `pathWhitelist` supersede check, add a second
       check for Bash outputs.
     - Maintain a `latestBashCmd: [String: Int]` dictionary mapping
       normalized command strings to the latest index.
     - If the same command key appears again, mark the earlier index as
       `.superseded` with reason `"superseded by later Bash run of same command"`.
   - Verbatim-window messages are already protected from supersede. The
     existing `if verbatim[idx] { continue }` guard (or equivalent)
     applies to Bash outputs too -- no special handling needed.

3. **Add escape hatch for commands with legitimately different outputs**
   (`ContextCurator.swift`)
   - Some commands produce legitimately different outputs each time they
     run (e.g., `date`, `git log` after commits, `ls` after file changes).
     The model may need both outputs.
   - Phase 1: No escape hatch. Exact command match supersedes. This is
     correct for the majority case (re-running tests, re-reading files).
     The model can always re-run a command if it needs the output again.
   - Phase 2 (future, out of scope): Add a `bashSupersedeBlacklist`
     in `CurationOptions` for commands that should never supersede
     (e.g., commands containing `date`, `git log`).

4. **Update port-fidelity ledger** (`ContextCurator.swift` lines 17-22)
   - Add a fidelity note:
     `//  - [cev2] Bash outputs supersede by normalized command key (v1 skips non-whitelisted tools).`

## Boundaries

- Do NOT add fuzzy command matching in this spec. Start with exact
  normalized match. Fuzzy matching (e.g., treating `cat foo` and
  `cat -n foo` as equivalent) is a separate future enhancement.
- Do NOT modify the existing `pathWhitelist` supersede logic. Bash
  supersede is additive -- a parallel key space.
- Do NOT supersede across different tool names. Only Bash-to-Bash
  supersede is in scope. A `Read` of a file and a `cat` of the same file
  in Bash are different tool outputs with different semantic properties.
- Do NOT introduce SwiftUI/AppKit dependencies.

## Risks

- **False positives on stateful commands.** `git status` after `git add`
  produces different output. Superseding the earlier `git status` loses
  the pre-add state. Mitigation: this is acceptable -- the model cares
  about current state, not historical state. The older output is stale
  by definition.
- **Command extraction reliability.** Parsing the command from the
  `arguments` JSON requires knowing the argument schema. If the Bash tool
  spec changes (e.g., the field is renamed from `command` to `cmd`), the
  extraction breaks silently (returns nil, no supersede -- safe failure).
- **Backward lookup cost.** Finding the assistant message that issued a
  tool call requires scanning backward. In long conversations this could
  be slow. Mitigation: the scan stops at the first matching assistant
  message. In practice, tool results immediately follow their assistant
  turn, so the scan is 1-2 messages.
- **Arguments JSON parsing.** The `arguments` field on `LLMToolCall` is a
  raw JSON string. Parsing it to extract the command requires
  `JSONSerialization` or similar. This is a Foundation API, no external
  dependencies needed.

## Validation

- **Unit test -- exact match supersede:** Two Bash tool results with the
  same command string. Verify the earlier is marked `.superseded`.
- **Unit test -- different commands preserved:** Two Bash tool results with
  different commands. Verify both are `.kept`.
- **Unit test -- verbatim protection:** A Bash tool result inside the
  verbatim window with the same command as an earlier result. Verify the
  verbatim one is `.kept` and the earlier one is `.superseded`.
- **Unit test -- whitespace normalization:** `"  git status  "` and
  `"git status"` should supersede each other.
- **Unit test -- non-Bash tools unaffected:** Verify that Read/Write/Edit
  tools continue to use path-based supersede, not command-based.
- **Existing tests must still pass.** The `test_nonWhitelistedToolDoesNotSupersede`
  test verifies that Bash does NOT supersede -- this test must be UPDATED
  to reflect the new behavior (Bash now supersedes by command key).
- **Token reduction measurement:** Run the curator on a representative
  conversation with repeated Bash commands and measure token reduction.
  Expect 10-20% reduction in Bash-heavy sessions.
