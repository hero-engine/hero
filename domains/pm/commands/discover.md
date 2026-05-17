---
description: Continuous-discovery research kickoff or check-in — opportunity solution trees, interview design, assumption tests.
---
Route this discovery request to the `discovery-researcher` agent. Loads `opportunity-solution-trees-torres`, `continuous-discovery-cadence`, and `evidence-synthesis` skills.

## Modes

- **No arguments** → open the active OST (opportunity solution tree) for the current initiative in session context. If none, ask which initiative to research.
- **`--outcome <slug>`** → design an OST starting from the named outcome. Use this for an initiative that has a desired outcome but no opportunity framing yet.
- **In context of an unframed initiative** (no OST present) → design the OST: outcome → opportunities → solutions → assumption tests.
- **`--interview <count>`** → design `<count>` interview guides plus recruitment screener questions. Pulls segment / persona context from the initiative or linked intake.
- **`--assumption <statement>`** → design an assumption test (smallest experiment that would disprove the assumption).

## Output

Discovery artifacts land inside the linked initiative or PRD — never as standalone files:

- OST → written to a `## Opportunity Solution Tree` section on the initiative.
- Interview guides → written to `.hero/planning/roadmap/<slug>/research/interview-<n>.md` and linked from the initiative.
- Assumption tests → added to a `## Assumption Tests` section with hypothesis, test design, success criteria, and a status field.
- Synthesis notes (after interviews / tests run) → written to `.hero/knowledge/notes/research/<topic>.md` so future sessions inherit the evidence base.

The agent shows a one-line log to chat for each artifact written. The artifact is the deliverable.

After discovery work, evaluate whether the session produced a customer insight worth persisting beyond the initiative (a recurring segment pattern, a falsified assumption, a strong quote). If so, file it to `.hero/knowledge/notes/research/`. See `auto-knowledge-capture`.

Request: $ARGUMENTS
