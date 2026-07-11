---
title: "Premise Interrogation — Force-Question the Framing Before Designing"
slug: premise-interrogation
type: feature
status: planning
priority: medium
horizon: now
tags: [skills, design, discovery, quality, framing]
relations:
  - target: timely-briefs
    kind: related
  - target: synthesis-maintenance
    kind: related
created: 2026-05-12
---

## Problem

Today the `/design` skill goes straight from the user's stated request to
delegating to a delivery lead, who produces a spec. The framing of the
request itself is never challenged. If the user asks for "a feature that
does X" and X is the wrong shape — actually a bug fix, actually a missing
convention, actually a different product, actually three small things
disguised as one — `/design` writes that spec anyway. The downstream
artifacts (commits, smoke tests, briefs, retros) all inherit the bad
framing and the cost of that framing only becomes visible after delivery.

The skip is reliable: senior engineers and PMs intuitively interrogate a
premise before designing — restate it, surface hidden assumptions, test
scope, ask "are we sure this is even the right shape" — but they do it in
their own heads, and they do it inconsistently. Solo founders, junior
engineers, and tired seniors skip the step entirely. The result is specs
that are well-formed but solve the wrong problem.

`/discover` exists for open-ended exploration, but it is opt-in and shaped
for "what could we do?" not "is what you just asked for the right thing?"
`/challenge` exists for diagnoses but only applies to bug specs.

The gap: a forced, bounded interrogation of the **premise** of any new
spec before the spec is written, surfaced as a structured artifact that
joins the corpus.

## Goal

Add a **mandatory pre-flight phase** to `/design` (and an
explicitly-invocable `/interrogate` skill that runs the same protocol
standalone) that:

1. Restates the user's premise back to them in compressed form.
2. Surfaces relevant prior art from the graph that touches the premise.
3. Pushes back on hidden assumptions, scope, and shape.
4. Produces a structured `premise.md` artifact saved alongside the eventual
   spec.
5. Emits an explicit recommendation: `proceed` / `refine` / `redirect` /
   `not-now`, with reasoning.

The user can override any recommendation, but cannot accidentally skip the
phase. The design skill will not delegate to a delivery lead until the
interrogation has produced a `premise.md` and a recommendation.

**Mission-fit.** This is direct floor-raising: premise interrogation is
the kind of disciplined skepticism that is invisible in code but visible
in outcomes — and it is exactly the gap between someone who has been a
senior PM for ten years and someone who hasn't. Baking it into `/design`
means every spec the corpus accumulates is one that survived a structured
"are you solving the right thing?" check, not just whatever the user
typed at 11pm. It also makes the next session smarter because the
premise interrogation itself is corpus — future specs on similar topics
inherit the prior interrogation's findings.

## Design

### Five-question interrogation protocol

The interrogation is a fixed-shape conversation, bounded in turn count
to keep cost low and avoid death-by-analysis. Five questions, one short
LLM turn each, every answer recorded.

| # | Question shape | What it tests |
|---|---|---|
| 1 | **Restate in one sentence.** "Here is what I think you are asking for: <restatement>. Is that right?" | Whether the user's framing survives compression. If they say "no, that's not it," the request was unclear and we surface what's missing. |
| 2 | **Who hits this and how often?** "Walk me through the last time you (or someone) actually hit this. If you can't think of a real instance, that's a signal." | Real pain vs. imagined pain. |
| 3 | **What's the workaround today?** "If we did nothing, what does the user do instead? How bad is that?" | Whether the cost of the status quo justifies the spec. |
| 4 | **Is this the right shape?** "Could this be: a bug fix, a convention, a decision, a knowledge entry, a doc, a tripwire, or a different product? Why a feature spec specifically?" | Scope and shape. The most common failure mode — features that should be conventions, conventions that should be bug fixes, etc. |
| 5 | **What does done look like?** "In one sentence, what's the test you'll run on the delivered thing to know it worked?" | Whether the user has a clear acceptance signal. If they can't name one, the spec is premature. |

The model also performs **automatic prior-art retrieval** before question
1 by calling the existing retrieval layer with the user's raw premise as
the query. Top-N results are surfaced inline with the restatement: "I
found these related items in the corpus — does any of them already
address what you're asking for, or supersede it?" This catches the case
where the user is asking for something the corpus already knows.

After the five answers, the model synthesizes a recommendation:

| Recommendation | Meaning |
|---|---|
| `proceed` | Premise is sound, scope is right, shape is right, prior art doesn't supersede. Design proceeds. |
| `refine` | Premise is sound but the framing/scope/shape needs adjustment. The interrogation produces a revised premise statement and proceeds to design with that. |
| `redirect` | This is the wrong shape — should be a bug spec, convention spec, decision spec, doc, etc. The user is told the right shape and the design path exits. |
| `not-now` | Real pain but premature — workaround is acceptable, no clear acceptance signal, or prior art needs to land first. Saved to `.hero/planning/parking-lot/` with the interrogation findings. |

The recommendation is **advisory** — the user can override with
`hero design --force` or by re-running `/design` with the same premise
plus an explicit override flag. Override is logged.

### `premise.md` artifact

The interrogation produces one structured artifact saved to the eventual
spec's folder (or to `parking-lot/` if `not-now`):

```markdown
---
type: premise
spec_slug: timely-briefs
created: 2026-05-10T14:30:00-07:00
recommendation: proceed | refine | redirect | not-now
target_shape: feature | bug | convention | decision | knowledge | doc | tripwire
overridden: false
override_reason: null
prior_art:
  - { type: feature, slug: activity-digest, relevance: 0.78 }
  - { type: knowledge, slug: brief-output-formats, relevance: 0.71 }
---

## Original premise
> <verbatim user input>

## Restated premise
<one-sentence restatement>

## Pain instance
<answer to Q2>

## Status-quo workaround
<answer to Q3>

## Shape rationale
<answer to Q4 — why a feature, not a bug/convention/decision/etc.>

## Definition of done
<answer to Q5 — the acceptance signal>

## Recommendation
**<proceed|refine|redirect|not-now>** — <one-paragraph reasoning>

## Prior art context
<short narrative on which prior items overlap and how>
```

The artifact is plain markdown so the existing scan / index pipeline picks
it up automatically. It joins the graph as a `premise` node type with
edges to: the eventual spec (`grounds`), referenced prior-art nodes
(`considered`), and the user (`requested-by`).

### Integration with `/design`

`design.md` is updated to add a pre-flight step:

1. Call `hero_anchor` (existing).
2. **Call `/interrogate` (new) with the user's request.** Block until
   `premise.md` exists and recommendation is non-`not-now` (or `--force`
   was passed).
3. If recommendation is `redirect`, exit and tell the user which path to
   take instead.
4. If recommendation is `not-now`, save `premise.md` to parking-lot and
   exit.
5. If recommendation is `refine` or `proceed`, delegate to delivery lead
   with the **refined premise** (not the raw user input) plus the
   `premise.md` path as context.
6. Spec is saved with a `relations: [{ target: <premise-slug>, kind:
   grounded-by }]` edge.

### `/interrogate` standalone skill

The same protocol is exposed as a standalone skill so the user can
interrogate a premise without committing to design (e.g. "I have an idea
but I'm not sure it's the right shape — interrogate it first"). Also
useful from inside `/discover` and `/decide`.

```
/interrogate <premise statement>
```

Produces a `premise.md` saved to `.hero/planning/premises/<slug>/` (no
spec folder yet) and prints the recommendation. The premise can later be
adopted by `/design` via `hero design --from-premise <slug>` which skips
re-interrogation and goes straight to delivery-lead handoff using the
existing `premise.md`.

### Skip and override mechanics

| Flag | Effect |
|---|---|
| `hero design --skip-interrogation` | Bypass the pre-flight entirely. Logs `design.interrogation_skipped` to the feed with the user's reason (required as `--reason "..."`). Reserved for genuinely well-known premises (obvious bug fix, trivial doc update). |
| `hero design --force` | Run interrogation, but proceed to design even if recommendation is `redirect` or `not-now`. Logs `design.recommendation_overridden`. |
| `hero design --from-premise <slug>` | Use an existing `premise.md` from `/interrogate`. No new interrogation runs. |

Skip and override are friction by design — the user must say what they're
doing and why, and it gets logged. The default path always interrogates.

### Prompt template surface

The five-question script is a template at
`.hero/templates/interrogation.md` (embedded built-in, user-editable). The
domain-aware variants (engineering, sales, etc.) override per-domain when
present, e.g. `.hero/templates/interrogation.engineering.md`. Resolution
order: domain-specific override → workspace template → embedded built-in.

## Changes

- `internal/interrogation/interrogation.go` — `Run(ctx, premise) (Result,
  error)`, prior-art retrieval, recommendation synthesis, `premise.md`
  serialization
- `internal/interrogation/interrogation_test.go` — protocol harness, fake
  LLM, recommendation logic
- `internal/cli/interrogate.go` — `hero interrogate` command
- `internal/cli/design.go` — extend the `hero design` command to invoke
  interrogation pre-flight; add `--skip-interrogation`, `--force`,
  `--from-premise`, `--reason` flags
- `internal/cli/root.go` — register `interrogateCmd`
- `internal/serve/mcp_tools.go` — register `hero_interrogate` MCP tool
- `internal/index/parser.go` — recognize `premise` type frontmatter,
  index it like other graph nodes
- `.claude/commands/design.md` — update the design slash command to
  describe the new pre-flight phase
- `.claude/commands/interrogate.md` — new slash command
- `.hero/templates/interrogation.md` — embedded template, materialized
  on first use
- `.hero/planning/parking-lot/` — created on first `not-now` decision

## Acceptance Criteria

- WHEN `hero design <premise>` runs without `--skip-interrogation` or
  `--from-premise` THE SYSTEM SHALL invoke the interrogation pre-flight
  before delegating to any delivery lead
- WHEN the interrogation runs THE SYSTEM SHALL retrieve prior art from
  the corpus using the raw premise as a query and surface the top-N
  results in the first interrogation turn
- WHEN the interrogation completes THE SYSTEM SHALL produce a
  `premise.md` file with frontmatter including `recommendation`,
  `target_shape`, `prior_art`, and `overridden`
- WHEN the recommendation is `redirect` AND `--force` is not passed THE
  SYSTEM SHALL exit `hero design`, save `premise.md` to
  `.hero/planning/parking-lot/`, and print the recommended alternative
  shape (bug, convention, decision, etc.)
- WHEN the recommendation is `not-now` AND `--force` is not passed THE
  SYSTEM SHALL exit `hero design`, save `premise.md` to
  `.hero/planning/parking-lot/`, and print the reasoning
- WHEN the recommendation is `refine` THE SYSTEM SHALL pass the **refined
  premise statement** (not the raw user input) to the delivery lead
- WHEN the recommendation is `proceed` or `refine` THE SYSTEM SHALL save
  `premise.md` to the eventual spec's folder and add a
  `grounded-by: <premise-slug>` relation to the spec
- WHEN `hero design --skip-interrogation` runs WITHOUT `--reason` THE
  SYSTEM SHALL refuse and prompt for a reason
- WHEN `hero design --skip-interrogation --reason "..."` runs THE SYSTEM
  SHALL log `design.interrogation_skipped` to the feed with the reason
  and proceed to design
- WHEN `hero design --force` runs and the interrogation recommends
  `redirect` or `not-now` THE SYSTEM SHALL log
  `design.recommendation_overridden` and proceed to design with the
  refined premise
- WHEN `hero interrogate <premise>` runs as a standalone command THE
  SYSTEM SHALL run the same protocol and save `premise.md` to
  `.hero/planning/premises/<slug>/`
- WHEN `hero design --from-premise <slug>` runs THE SYSTEM SHALL use the
  existing `premise.md` and skip re-interrogation
- WHERE a domain-specific template exists at
  `.hero/templates/interrogation.<domain>.md` THE SYSTEM SHALL prefer it
  over the workspace and built-in templates
- THE SYSTEM SHALL bound the interrogation to at most 5 LLM turns per
  run
- THE SYSTEM SHALL emit `premise` nodes into the graph so future
  retrieval, briefs, and interrogations can reference them

## Boundaries

- Does **not** replace `/discover` (open-ended exploration) or
  `/challenge` (bug-diagnosis pushback). Interrogation is targeted: the
  user has already named what they want; the protocol pushes back on
  whether they named it correctly.
- Does **not** gate any work — every recommendation is overridable. The
  point is to make the override an explicit, logged choice instead of an
  invisible default.
- Does **not** apply to `/diagnose` or `/decide` in v1 — only `/design`
  (and the standalone `/interrogate`). Extension to other producer
  skills is a follow-on once the protocol is proven.
- Does **not** evaluate technical feasibility. That is delivery-lead
  work. The interrogation is about **framing**, not implementation.
- Does **not** require human-in-the-loop chat — the interrogation runs
  as five LLM turns scripted by the protocol. The user only sees the
  final recommendation and `premise.md`. (The interactive variant —
  where the user answers each of the five questions themselves — is a
  follow-on, useful when the request originates from a non-technical
  user.)
- Does **not** persist interrogation transcripts beyond `premise.md`.
  The structured frontmatter and the answered sections are the artifact;
  the raw LLM turns are not stored.
- Does **not** federate prior-art retrieval across cloud / cross-repo —
  workspace-local. Cross-org premise check is downstream of `cloud-mcp`.
- Does **not** rerun interrogation on spec edits. The premise check
  happens once at spec inception. Re-interrogation is a manual
  `hero interrogate <slug>` command if the user wants it.
