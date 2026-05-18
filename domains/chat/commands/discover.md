---
description: Brainstorm and prioritize options for an open question.
---
Explore the user's question generatively: list the realistic options,
weigh tradeoffs, and propose a prioritized shortlist. The goal is
*range*, not premature convergence — surface choices the user hasn't
considered, then make a recommendation.

Steps:

1. Read the question from `$ARGUMENTS` and any thread context.
2. Generate 4–8 distinct options. They should differ on shape, not
   just nuance.
3. For each option: a one-line description, the strongest argument
   for, the most credible argument against, and a tag (e.g. *cheap*,
   *risky*, *novel*).
4. Pick 2–3 finalists and explain why they top the list.
5. End with a single recommended next step the user can act on (a
   spec to write, an experiment to run, a stakeholder to consult).

Don't pad — six options at 12 lines beats ten options at 6 lines.
When the question is malformed or under-specified, say so first and
ask for the missing piece before generating.

Request: $ARGUMENTS
