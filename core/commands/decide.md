---
description: Evaluate an architectural decision with structured analysis and produce a decision spec.
---
**Before creating any file**, check whether the user is working in a sub-folder workspace. If so, preserve that workspace's `subproject:` frontmatter and write the decision under the workspace's `.hero/` root; otherwise write under the project `.hero/` root.

Run the structured evaluation directly as the session agent, or delegate to
the install's reviewer agent if one is installed (engineering:
`architecture-reviewer`; pm: `pm-reviewer`).

The evaluation:
1. Call `hero_anchor` with the decision context to load project mission and tripwires (forbidden options). Eliminate any option that appears in the tripwire list before evaluating — do not present it as an option with caveats.
2. Clarify the decision and its constraints
3. Evaluate the options with tradeoff analysis
4. Consider architectural fit, operational burden, team impact, and reversibility
5. Produce a decision spec with recommendation, rationale, and consequences
6. Save it to `.hero/knowledge/decisions/<slug>.md` using the decision (ADR) template from the `spec-format` skill

On engineering installs, also consult `brownfield-architect` for
existing-system concerns or `greenfield-architect` for new subsystems.

Decision to evaluate: $ARGUMENTS
