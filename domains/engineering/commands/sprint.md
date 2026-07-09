---
description: Plan a sprint by selecting and sequencing specs from a backlog or initiative.
---
Route this sprint planning request to the `feature-delivery-lead`.

The delivery lead will:
1. Read the initiative or backlog context provided
2. If a specific initiative slug is provided, load it and its child specs
3. If no initiative is specified, run `hero check` and `hero dashboard` to understand the current workspace state
4. Identify specs that are ready for the next sprint (dependencies met, not blocked)
5. Suggest a sprint scope based on:
   - Team capacity (if known from conventions/context)
   - Spec complexity and dependencies
   - Priority and business impact
6. For each selected spec, verify it has:
   - Clear acceptance criteria
   - A Changes section listing files to modify
   - An assigned owner (suggest one if unclaimed)
7. Produce a sprint plan document as a note in the knowledge base at `.hero/knowledge/notes/sprint-{date}/spec.md` containing:
   - Sprint goal (1 sentence)
   - Selected specs with delivery order
   - Dependency graph (if specs depend on each other)
   - Risk items and blockers
   - Capacity allocation (which specs go to whom)

**Sprint planning principles:**
- A sprint should be achievable in 1-2 weeks
- Include a mix of high-value features and smaller wins
- Don't overcommit — leave 20% buffer for unplanned work
- Prioritize specs that unblock other specs
- If a spec is too large for a sprint, suggest running `/split` first

## Executing the sprint

`/sprint` plans; it does not run a bespoke execution loop. Route based on shape:

- **Initiative-shaped** (the sprint maps to an initiative's child specs) →
  `/drive <initiative>`. Autonomous, with deterministic `needs_me` pauses;
  strictly supersedes any hand-rolled autopilot loop here.
- **Ad-hoc list of ready specs** (no single initiative) → `/deliver` queue
  mode: "deliver these while I'm away."

Both paths already run `hero spec verify` as their own closing gate — no
separate `hero spec complete` step is needed or correct here.

Initiative or context: $ARGUMENTS
