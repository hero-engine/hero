---
description: Draft a new PRD or refine an existing one — preset-aware (pitch shape under cycle, ten-section under sprint/flow).
---
Route this PRD request to the `prd-author` agent. Loads the `prd-structure` skill.

## Modes

- **No slug** → draft a new PRD. Author pulls context from the current session: an active intake cluster, an `initiative` promoted recently, customer evidence from discovery notes, or whatever the user passes as $ARGUMENTS.
- **Slug provided** → refine the existing PRD at that path.

## Pre-flight

1. Load the active methodology preset via `pm-preset-detection`:
   - `delivery: cycle` → use the Shape Up pitch template (Problem, Appetite, Solution sketch, Rabbit holes, No-Gos).
   - Otherwise → use the ten-section template (Context, Problem, Goals/Non-goals, Users, Solution, AC, Open questions, Metrics, Risks, Rollout).
2. **If drafting and no linked `initiative` exists**, ask the user which initiative this PRD belongs to before generating. A PRD without a roadmap parent is a PRD without strategic context — don't paper over that. If the user says "no initiative yet," offer to spin one up via `/refine` on an initiative draft first.

## Flags

- `--section <name>` — refine only the named section (e.g. `--section metrics` to author the Metrics section in isolation; pairs well with `/metrics`).
- `--inline-propose` — land content as proposed-content in the artifact pane (default when invoked from a contextual button).

## Output

- The PRD spec file at `.hero/planning/prds/<slug>/spec.md` is created or updated.
- A one-line log to chat naming what was drafted or refined.
- After authoring, if `knowledge.auto_capture` is on, capture any novel decisions made during shape (tradeoffs taken, alternatives ruled out, customer-evidence patterns) to `.hero/knowledge/notes/`. See the `auto-knowledge-capture` skill.

Request: $ARGUMENTS
