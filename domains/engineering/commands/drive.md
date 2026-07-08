---
description: Run a whole initiative autonomously — execute its child specs, pausing only when a decision genuinely needs you.
---
Run an initiative on autopilot via the `drive` skill. Load the `drive` skill before starting.

`/drive <initiative>` arms an autonomous run over the initiative's child
specs: design → deliver → verify → advance to the next, pausing only at a
`needs_me` boundary (a real design fork, ambiguity, an irreversible action, a
stuck gate, or a hard cap). Natural-language synonyms route here too:
"autopilot this initiative", "put X on autopilot", "keep working the whole
initiative".

**Hero does not drive the loop** — the harness `/goal` does. This command
arms the run: it ensures the initiative has a `## Goal` run-opener, confirms
on first arm (showing the condition, the `autonomy:` mode, and the
guardrails), emits the condition with `hero goal <init> --emit`, and wires
the Stop hook that calls `hero goal <init> --check` each turn. On a `pause`
verdict it surfaces the question and waits; on `done` it reports completion.

**This is not `/deliver`.** `/deliver` runs one spec, one step, you at the
boundary. `/drive` runs the whole initiative autonomously. If the target is a
single (non-initiative) spec, decline and point to `/deliver`.

Initiative to drive: $ARGUMENTS
