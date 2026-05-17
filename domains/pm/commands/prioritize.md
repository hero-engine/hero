---
description: Rank a set of initiatives or specs using a prioritization framework (value-vs-effort, RICE, ICE, WSJF).
---
Route this prioritization request to the `prioritization-strategist` agent. Loads the `prioritization-frameworks` skill.

## Framework

- **Default**: `value-vs-effort` (simple 2x2; fastest signal).
- **`--framework rice`** — Reach × Impact × Confidence / Effort.
- **`--framework ice`** — Impact × Confidence × Ease.
- **`--framework wsjf`** — Weighted Shortest Job First (cost of delay / job size).

The strategist shows the math for whichever framework is in use. It also calls out soft inputs the score doesn't capture (strategic bets, sequencing dependencies, customer commitments) so the PM can override with eyes open.

## Target scope

Determine what's being ranked:
- No argument → all `initiative` specs on the `Now` horizon.
- `--theme <theme>` → items tagged with that theme.
- `--horizon <now|next|later>` → items on the named horizon.
- A list of slugs → rank exactly that set.
- An epic slug → rank child specs within the epic.

## Output

- Each item's frontmatter is updated with the framework's score fields (e.g. `rice_reach`, `rice_impact`, `rice_confidence`, `rice_effort`, `rice_score`) and the resulting `priority_rank` within the scope.
- On the Roadmap board view, an inline-proposed ordering change is surfaced (accept/reject) rather than auto-applied.
- A ranked table is logged to chat with scores and the strategist's soft-input commentary.

After ranking lands, log `hero event decision_made` with the framework used and the top-of-list rationale so the next session sees the priority call.

Request: $ARGUMENTS
