---
description: Design a customer interview guide — open, story-based questions about specific past experiences, non-leading framing, sample size, and a synthesis plan.
---
Route to `discovery-researcher`, loading the `discovery-interview-design` skill.

## Scope

- **A topic or outcome** (e.g. "export workflow", "onboarding friction") → design an interview guide to reduce uncertainty about it.
- **`--count <n>`** → target sample size for the segment (5 is the unit; diminishing returns after that per segment).
- **No arguments** → ask which outcome or assumption the interviews should de-risk; don't guess.

## What lands

An interview guide the team can run, following Torres-tradition discipline (per `discovery-interview-design`):

- **Story-based, not opinion-based** — "Tell me about the last time you exported data" beats "Would you use a CSV export?"
- **Past behavior over speculation** — what they did, not what they would do.
- **Avoid leading the witness** — no "Don't you think…" / "Wouldn't it be helpful if…".
- **Sample size** — 5 users is the unit per segment; mix in churned / never-converted / competitor users to avoid confirmation bias.
- **Synthesis plan** — how findings will be extracted (assumptions confirmed / disconfirmed, new opportunities) and where they'll be written.

Write the guide as an artifact section or, if reusable, a note in `.hero/knowledge/notes/`.

## Output

- The interview guide (questions, sample, synthesis plan).
- A one-line log naming the outcome being de-risked and the target sample.

Request: $ARGUMENTS
