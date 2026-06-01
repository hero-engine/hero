---
title: Move LLM Judgment to CLI When Option Failure Costs Are Asymmetric
type: decision
status: proposed
created: 2026-05-31
tags: [agent-reliability, prompt-design, cli, renderer-selection, structural-fix, principle]
relations:
  - target: mockup-renderer-selection-swiftui-bias
    kind: drove
---

# Move LLM Judgment to CLI When Option Failure Costs Are Asymmetric

## Decision

When Hero asks an agent to pick between options whose **failure costs
are asymmetric** (one option always succeeds; another can fail noisily),
the choice MUST be made by deterministic Go code and emitted as a
single line of JSON for the agent to read verbatim — not left as
prompt-interpreted instructions.

Examples of asymmetric-cost choices:

- **Mockup renderer:** HTML never fails (no toolchain, always emits
  `index.html`); SwiftUI requires `swiftc`, compile success, and
  successful screenshot capture.
- **Stack-specific test runner:** `make test` may not exist; `go test
  ./...` always at least returns an exit code.
- **Spec-vs-tracker source of truth:** local search always returns;
  tracker call requires network and credentials.

Under any prompt ambiguity, the agent will **systematically drift
toward the lower-risk option** even when it's the wrong answer for the
project, because the cost of being wrong is asymmetric (silent wrong
artifact vs. visible error). Prompt counter-pressure cannot fix this
structurally — the bias is built into how LLMs weigh risk.

The structural fix is to remove the judgment from the LLM entirely:
add a `hero <feature> detect` CLI command that runs the algorithm in
Go, emits one line of JSON with `{choice, reason, signals, conflict}`,
and replace the prompt's selection block with "call this; use the
output verbatim."

## Why

The triggering case: `/mock` kept selecting HTML on Swift projects
despite a clear four-step auto-detect algorithm in
`domains/engineering/commands/mock.md`. Re-wording the prompt didn't
fix it across attempts. Root cause was structural: free-form glob
probing is exactly what LLMs do badly, and the safer option always
wins under ambiguity.

The same pattern will recur in any decision where:

1. The agent has to interpret multiple project signals (files, paths,
   configs) before choosing.
2. One option has lower visible failure cost than another.
3. We add prompt rules to counter the bias and they stop working a
   release or two later.

Catching this once and naming the pattern ("asymmetric-failure choice
→ move to CLI") lets us short-circuit the prompt-tuning death spiral
in future cases.

## Implementation pattern

```
1. hero <feature> detect [--<flag>=value]
   - Walks the repo (reuse internal/snapshot/detect.ScanRepo when
     possible to share monorepo semantics)
   - Applies the algorithm in Go: explicit flag → config override →
     auto-detect → toolchain gate
   - Emits one line of JSON:
       { "choice": "...", "reason": "...", "signals": [...],
         "toolchain_ok": bool, "conflict": "..." | null }
   - Conflict field populated when user-passed flag opposes detected
     stack — agent must halt and confirm

2. Prompt block in /command and agent shrinks to:
   "Run `hero <feature> detect [--<flag>]`. Use the `choice` field
    of its JSON output verbatim. If `conflict` is non-null, halt and
    surface the message; do not proceed until the user confirms."

3. Mandatory announce step before generation:
   "Choice: X  —  reason: Y  —  toolchain: <path|unavailable>"
   Surfaces wrong picks immediately, gives the user a same-turn
   correction window.
```

## Anti-patterns this replaces

- **"Just reword the prompt more clearly."** Three rounds of
  rewording the renderer-selection block did not stop the HTML bias.
  Prompt tuning treats the symptom; the bias is structural.
- **"Add a stronger 'MUST' to the rule."** Same problem — the LLM
  still weighs visible failure cost above stated rules under
  ambiguity.
- **"Duplicate the algorithm in the agent definition AND the slash
  command."** Two prompt copies of the same algorithm just gives the
  LLM two free-form interpretations to drift between. The pattern
  here has one Go implementation; the prompts reference it.

## When NOT to apply this

- The choice has **symmetric failure cost** (both options can succeed
  or fail equally) — prompt rules work fine, the bias driver isn't
  present.
- The choice depends on **runtime context the CLI can't see** (user
  intent, conversation history, in-flight session state). LLM judgment
  is the right tool there.
- The choice is **one-shot and unique** to a particular spec. Don't
  build a CLI command for a decision that fires once.

## Relations

- Drove `.hero/planning/bugs/mockup-renderer-selection-swiftui-bias/spec.md`.
- Likely candidates for the same treatment (not yet specced):
  test-runner detection in `/deliver`, peer-call mode selection in
  `hero peer call`, spec-type classification on `/diagnose` vs
  `/design`.
