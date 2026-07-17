---
description: Draft a Shape Up pitch — Problem, Appetite, Solution sketch, Rabbit holes, No-Gos.
---
Route to `pitch-author`. Loads the `pitch-writing-shape-up` skill.

## Pre-flight

1. Load the active methodology preset via `pm-preset-detection`.
2. **If the active delivery preset is not `cycle`**, ask the user: "The active preset is `<preset>`, not `cycle`. Do you want to apply pitch shape anyway (override preset for this artifact only)?" Pitches without a cycle context are valid — some teams pitch ideas without committing to Shape Up — but the override should be conscious.
3. If an `initiative` slug is in scope, link the pitch to it.

## Pitch shape

The pitch lands with five sections:

- **Problem** — the raw signal and why it matters now.
- **Appetite** — the time budget (small batch / big batch). Non-empty is required to mark `review`-ready.
- **Solution** — fat-marker sketch; not engineering-detailed.
- **Rabbit holes** — known traps and how the pitch avoids them.
- **No-Gos** — explicitly out of scope. Non-empty is required to mark `review`-ready.

## Enforcement

Before the agent flips the pitch to `status: review`:
- `appetite` field must be set (e.g. "6 weeks", "2 weeks").
- `no_gos` section must contain at least one bullet.

If either is empty, the agent leaves the pitch at `status: draft` and surfaces what's missing.

## Output

- The pitch spec file at `.hero/planning/prds/<slug>/spec.md` (pitches are PRDs with pitch-shaped content) is created or updated.
- A one-line log naming the appetite and the headline of the solution sketch.

Request: $ARGUMENTS
