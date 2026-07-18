---
name: research-workflow
description: The orchestration doctrine behind `/research` — plan first and pause for approval, work from a bounded source set, search in rounds against explicit stopping criteria, and emit a fixed checkpoint sequence any client can render and interrupt. This skill is the authoritative definition of the checkpoint/interrupt contract.
metadata:
  audience: researcher
  purpose: research-orchestration
---

## What I do

I define how a rigorous research run is *structured* — the phases, the order,
and the observable checkpoints — so the `researcher` agent produces a reviewable,
interruptible investigation instead of a single opaque answer. The rigor of
evaluating and citing sources lives in `source-evaluation` and
`evidence-and-citation`; I own the loop that drives them.

I am **client-agnostic**. I cannot draw a UI. What I can do is define a fixed
checkpoint vocabulary and ordering that any client — a Swift chat app, a GPUI
editor, a terminal — can render into a plan-approval prompt, a progress view, and
a stop control. Describe capabilities abstractly ("the session's web-search
capability", "the session's file-read capability"); name a specific client only
as an optional aside.

## When to use me

Load me at the start of any `/research` run, before producing the plan. The
`researcher` agent loads me automatically.

## The seven properties, and where each lives

A research run must be reviewable, source-bounded, iterative, source-evaluated,
evidence-synthesized, cited, and interruptible. Three of those are mine (plan,
rounds, checkpoints/interrupt); the other four are delegated:

| Property | Owner |
|---|---|
| Reviewable planning | this skill |
| Controlled sources | this skill (declared in the plan) |
| Iterative search | this skill (the round loop) |
| Source evaluation | `source-evaluation` |
| Evidence synthesis | `evidence-and-citation` |
| Citations | `evidence-and-citation` |
| Progress + interruptibility | this skill (the checkpoint contract) |

## Phase 1 — Plan, then pause

Before running a single search, produce a **research plan** and stop for the
user to review, edit, or approve it. The plan is a checkpoint the client renders
for approval; searching before approval defeats the point.

A plan contains:

- **Question** — the user's question restated precisely, with any scoping the
  restatement makes explicit.
- **Sub-questions** — the 3–6 decomposed questions whose answers add up to the
  whole. If you cannot decompose it, the question is either trivial (answer it
  conversationally, no `/research` needed) or under-specified (ask first).
- **Source set** — the bounded, explicit set of sources you will draw on (see
  Phase 2). Name it; do not leave it open.
- **Stopping criteria** — what "done" looks like: every sub-question answered
  with corroborated evidence, or a stated round ceiling reached, whichever comes
  first. Vague stopping criteria produce runaway searches.

Emit the plan as the `plan` checkpoint and **wait**. Do not proceed to Phase 3
until the user approves. If the user edits the plan, re-emit and wait again.

## Phase 2 — Controlled sources

The plan declares one bounded source set, chosen deliberately:

- **corpus-only** — the workspace knowledge corpus and files the user points at;
  no open web.
- **web-allowed** — the session's web-search capability plus the corpus.
- **named allowlist** — a specific set of sources or domains the user named.

You do **not** silently widen scope mid-run. If a round reveals you need a source
class outside the declared set, stop, propose an **amended plan** naming the new
source class and why, and pause for approval — a fresh `plan` checkpoint. Scope
creep is the quiet way a controlled investigation turns into an unbounded one.

## Phase 3 — Search in rounds

Search proceeds in numbered **rounds**, not one undifferentiated sweep. Each
round:

1. Runs a focused batch of queries aimed at the currently least-answered
   sub-questions.
2. Triages what came back through `source-evaluation` before using any of it.
3. Emits a `round` checkpoint: a short summary of *queries run*, *what was
   found*, and *what is still missing*.
4. Decides against the plan's stopping criteria whether another round is
   warranted. If every sub-question is answered with corroborated evidence, stop.
   If the round ceiling is hit with gaps remaining, stop and report the gaps
   honestly — do not manufacture coverage.

A round that finds nothing new is a signal: either the source set is exhausted
(stop, report what is unanswerable within scope) or the queries need reframing
(one reframed retry, then stop).

## Phase 4 — Evaluation and synthesis

Once rounds close, the retained, evaluated sources feed `evidence-and-citation`:
assemble claims from evaluated evidence, surface contradictions between sources
rather than silently resolving them, and attach an inline citation to every
non-obvious claim. Emit the `evaluation` checkpoint (the triaged source set with
its credibility read) and then the `synthesis` checkpoint (the assembled,
cited claims) before writing the final report.

## Phase 5 — Report

Emit the `report` checkpoint: the cited synthesis as a readable report, with a
`Sources:` register (see `evidence-and-citation`) and an explicit note of any
sub-question that could not be answered within the declared source set.

## The checkpoint contract (client-renderable)

Every run emits this fixed sequence. Clients render each however they like; the
*names and ordering* are the contract they build against:

```
plan  →  round (×N)  →  evaluation  →  synthesis  →  report
```

Each phase carries content for the user:

- `plan` — the restated question, sub-questions, source set, stopping criteria.
  **Blocks on user approval.**
- `round` — the round number, queries run, findings, remaining gaps. Emitted once
  per round.
- `evaluation` — the retained source set with each source's credibility triage.
- `synthesis` — the assembled claims with their citations and any surfaced
  contradictions.
- `report` — the final cited report and the `Sources:` register.

### Machine-readable emission (the parseable signal)

Prose alone is not a contract a client can bind to — wording drift would silently
break a client that sniffs for phrases. So at **each** transition you MUST print,
**on its own line, before that phase's human-readable content**, a single
sentinel of this exact fixed-prefix form:

```
<hero:checkpoint kind="KIND" ...attributes>
```

The line is a machine signal, not prose; a client matches the literal prefix
`<hero:checkpoint` and reads the attributes. The one per phase:

| Emit at | Sentinel |
|---|---|
| Plan ready, before any search | `<hero:checkpoint kind="plan" status="awaiting-approval">` |
| Start of round K | `<hero:checkpoint kind="round" n="K">` |
| Source evaluation | `<hero:checkpoint kind="evaluation">` |
| Synthesis | `<hero:checkpoint kind="synthesis">` |
| Final report (run completed) | `<hero:checkpoint kind="report" status="complete">` |
| Final report (interrupted) | `<hero:checkpoint kind="report" status="incomplete" stopped-after-round="K">` |

Rules that make it parseable:

- **One sentinel per transition, on its own line, at the transition** — before the
  content it announces. Never inline inside a sentence.
- **Fixed prefix, quoted attributes.** `kind` is always present; `n`,
  `status`, and `stopped-after-round` appear where the table shows them.
- **The `plan` sentinel's `status="awaiting-approval"` is the pause signal.** The
  client renders the plan for approval on seeing it and must not proceed until the
  user approves. You do not search past a `plan` sentinel on your own.
- The sentinels are the authoritative machine signal; the surrounding prose is for
  the human. Keep both — clients that ignore the sentinels still get readable
  output, and clients that bind to them get a schema.

### What the client owns (not enforceable here)

This contract is emission only. Two guarantees depend on the client's loop and
cannot be enforced from content — state them so the boundary is explicit:

- **Plan-approval is a hard gate the client enforces.** The sentinel signals
  "await approval"; actually withholding tool/search execution until the user
  approves is the client's job. Treat the pause as advisory-until-the-client-gates.
- **Graceful stop.** The partial-report guarantee below holds only if the client's
  stop control lets you emit one final turn (the `report` sentinel + partial),
  rather than hard-killing the stream mid-token.

## Interrupt safety

A client's stop control can fire at any point. On interrupt, **never drop the
turn**. Checkpoint whatever findings exist and produce a *usable partial report*:

- Emit the interrupted `report` sentinel —
  `<hero:checkpoint kind="report" status="incomplete" stopped-after-round="K">` —
  then, as the report's **first human-readable line**, the banner
  **"Incomplete — stopped after round K."** (The sentinel is the machine form; the
  banner is the human form; both carry the same K.)
- Include the sub-questions answered so far with their citations, and list the
  sub-questions still open.
- Keep the `Sources:` register for whatever was actually used.

A partial report is a real deliverable, not an apology. The user stopped because
they had enough or changed direction; give them the value already gathered.

## Anti-patterns

- **Searching before the plan is approved.** The pause is the feature; skipping
  it turns `/research` into an ordinary answer with extra steps.
- **One giant search instead of rounds.** Without rounds there is no visible
  progress and no principled stopping point.
- **Silent scope widening.** Reaching for a source class the plan didn't declare,
  without an amended plan, breaks the "controlled sources" guarantee.
- **Dropping the turn on interrupt.** A stopped run must still yield the partial
  report. Anything else loses the user's work.
- **Manufacturing coverage at the ceiling.** If the round ceiling hits with gaps,
  name the gaps. Do not pad the report to look complete.
