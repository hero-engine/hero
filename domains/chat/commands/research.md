---
description: Investigate a question rigorously — plan, approve, search in rounds, evaluate sources, and synthesize a cited report.
---
Run a reviewable, source-grounded investigation of the user's question. This is
not a one-shot answer: it is a plan the user approves, a bounded search that
proceeds in rounds, per-source evaluation, and a synthesis where every non-obvious
claim carries a citation.

Where the session exposes subagents, delegate to the `researcher` agent, which
carries the full workflow. Where it does not, run the workflow inline yourself.
Either way the doctrine lives in the `research-workflow`, `source-evaluation`, and
`evidence-and-citation` skills — load and follow them.

Steps:

1. **Restate the question** from `$ARGUMENTS`, making any scope explicit.
2. **Produce a research plan and pause for approval before searching.** The plan
   states: the restated question, 3–6 sub-questions, the controlled source set
   (corpus-only, web-allowed, or a named allowlist), and the stopping criteria.
   Emit this as the `plan` checkpoint and **wait** — do not run any search until
   the user approves or edits it.
3. **Search in rounds.** Each round runs a focused batch of queries against the
   least-answered sub-questions using the session's web-search and file-read
   capabilities, triages what comes back for credibility, and emits a `round`
   checkpoint (queries run, what was found, what's still missing). Decide against
   the stopping criteria whether another round is warranted. Do not widen the
   source set without an amended plan and a fresh approval.
4. **Evaluate each retained source** for credibility, recency, primary-vs-
   secondary, bias, and corroboration before using its claims. Emit the
   `evaluation` checkpoint; note what you discarded and why.
5. **Synthesize a cited report.** Assemble claims from evaluated sources, surface
   contradictions between sources rather than resolving them silently, and attach
   an inline citation to every non-obvious claim. Emit the `synthesis` checkpoint,
   then the `report` checkpoint with a `Sources:` register and an honest note of
   any sub-question left unanswered within scope.

**Checkpoint contract:** emit `plan → round (×N) → evaluation → synthesis →
report`, in that order. At each transition, print — on its own line, before that
phase's content — the machine-readable sentinel `<hero:checkpoint kind="…" …>`
defined in `research-workflow` (e.g. `<hero:checkpoint kind="plan"
status="awaiting-approval">`, `<hero:checkpoint kind="round" n="1">`). The
sentinel is the parseable signal a client binds to; the surrounding prose is for
the user. Do not search past the `plan` sentinel until the user approves.

**On interrupt, never drop the turn.** Emit `<hero:checkpoint kind="report"
status="incomplete" stopped-after-round="K">`, then a usable partial report whose
first human-readable line is the banner "Incomplete — stopped after round K",
listing the sub-questions answered so far (with citations) and those still open.

Reference session capabilities abstractly; name a specific client only as an
optional aside. This command is canonical content — the client owns how the plan,
progress, and stop control are rendered.

Request: $ARGUMENTS
