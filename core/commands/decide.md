---
description: Evaluate an architectural decision with structured analysis and produce a decision spec.
---
Route this architectural decision to `architecture-reviewer` for structured evaluation.

The reviewer will:
1. Clarify the decision and its constraints
2. Evaluate the options with tradeoff analysis
3. Consider architectural fit, operational burden, team impact, and reversibility
4. Produce a decision spec with recommendation, rationale, and consequences
5. Save it to `.hero/decisions/{slug}/spec.md`

For decisions that involve a specific technology domain, also involve the relevant architect:
- Existing system concerns → also consult `brownfield-architect`
- New system or subsystem design → also consult `greenfield-architect`

Decision to evaluate: $ARGUMENTS
