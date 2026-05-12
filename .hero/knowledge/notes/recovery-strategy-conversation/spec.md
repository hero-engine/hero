---
title: Recovery Strategy Conversation — 2026-04-28
type: note
status: active
tags: [strategy, recovery, mission, principles, v2-audit, meta]
created: 2026-04-28
relations:
  - target: get-back-on-track
    kind: originated
horizon: now
---

# Recovery Strategy Conversation — 2026-04-28

A back-and-forth session that diagnosed the post-v2 drift, surfaced the
mission and principles that should have been guarding it, audited what
shipped vs. what was promised, and produced the `get-back-on-track`
initiative. Captured here verbatim-where-possible because *"a lot of
back and forth with a really smart session got lost — impossible to
recapture the magic"* (user, this session). Don't let it happen again.

## The trigger

User opened with the diagnosis: v1 (markdown corpus + code scans + skill
files for AI prompt context) worked great. v2 (graph DB substitute,
tooling to feed it on demand) shipped over ~3 weeks. Now feels like *"a
mess and big step back"*. Need to go back to the core mission and
actually deliver what v2 was supposed to do, with end-to-end testing on
each area.

## Eight strategic moves the session converged on

The session unfolded as a sequence of sharp, brief meta-observations
from the user, each refining the recovery plan. Recorded in order so
the reasoning is reconstructible.

### 1. Carve recovery into 8 e2e areas, not one monolithic smoke test

The existing `scripts/e2e_smoke.sh` runs every workflow back-to-back. It
caught 3 bugs and 6 UX issues on first run, but only at the surface
layer. Phases 4–10 of the v2 graph plan have no targeted exercise.

Areas: Onboarding · Discovery · Planning · Traversal · Delivery ·
Ingest · Projection · Federation. Each gets its own spec, its own
script, its own observation log against `go-task/task` and the hero
repo itself. Sequencing prioritizes Onboarding/Discovery/Projection
first (foundation), then Traversal (the actual v2 unlock), then the
rest.

### 2. Each area starts with 3–5 acceptance criteria — built out as we go

User: *"focus on a core set of acceptance criteria to be built out as
we go"*. Don't try to enumerate every test upfront. Start with the
minimum that proves the area isn't lying. New criteria accrete when
runs surface new bugs. The spec is the durable record; the script
asserts against the spec's current AC list.

### 3. Acceptance criteria themselves should be graph nodes

User: *"we also never really thought about graphing the acceptance
criteria for your own projects and injecting that properly as a guiding
light"*. The whole graph thesis was about turning project knowledge
into a queryable graph — but ACs, the most testable signal a project
produces, are buried as bullet points in markdown.

Make them first-class:
- `Criterion` node, key `<feature-slug>:AC-N`
- Edges: `belongs_to → Feature`, `verified_by → Script`, `satisfied_by → Commit`, `breaks` (Commit→Criterion regression), `participates_in` (File→Criterion via join), `derived_from → Attempt`
- All Tier-1 deterministic — no LLM
- Bitemporal: each AC status change is a row, not an overwrite

Injection points (the "guiding light"):
- `hero deliver <spec>` — open ACs printed as the success bar; engineer agent grades against them
- `hero relevant <file>` — ACs touching this file via `participates_in`
- `hero next` / `hero resume` — "since last session: AC-X went green, AC-Y regressed"
- `hero blocked` — features with failing ACs join naturally
- `hero why <AC>` — origin story

### 4. `hero scan` was supposed to be master ingest — it's not anymore

User: *"wasn't hero scan supposed to scan way more than just code? that
got lost in the shuffle"*.

Audit confirmed: v2 spec said `hero scan` is "master ingest: code,
planning, notes, raw, git, tracker, sync." Today it does code +
planning + git + raw + sessions only. **Missing:**
- notes-as-Note-nodes (treated as specs, not extracted as Note nodes)
- tracker pull (separate `hero sync pull` / `hero import`)
- team-server sync (separate `hero sync cloud`)
- memory files (`~/.claude/.../memory/` — never wired)
- Tier-2 LLM extraction (separate `hero extract`)

Each became its own verb — exactly the sub-verb sprawl the v2 spec
*explicitly forbade*: *"Every command is a verb that does one thing.
No sub-verb sprawl."*

### 5. Audit all the v2 specs against what was delivered — gaps everywhere

User: *"we need to review all these specs from the last week against
what was delivered - there are big gaps"*.

Three parallel Explore-agent audits ran. See companion note
`v2-delivery-audit-2026-04-28` for full reports. Headline findings:

- **`hero why` / `hero blocked` don't exist** — the v2 traversal
  showcase queries were never built
- **`graph-schema-simplification`** marked as part of phase 7c commit
  message; **schema unchanged** — fake delivery
- **`auto-capture learnings`** spec marked **status: completed**; **no
  implementation exists anywhere**
- **`cross-spec-awareness`, `institutional-memory`,
  `architectural-drift-detection`, `cross-org-intelligence`** — specs
  exist, zero code
- **`hero scan` ~60% of master-ingest promise**
- **Tier-2 extraction opt-in only** — never automatic
- **OAuth missing** — password-only auth
- **Dashboard a shell** — endpoints don't exist
- **Automations engine loads YAML; no triggers fire**
- **Notifications framework wired; never invoked from `ApproveJob`**
- **Sub-verb sprawl** — 35 MCP tools, 27 commands

Pattern across all three audits: *foundations ship; the surface that
delivers the mission doesn't*.

### 6. Find and lock the mission as a guiding light

User: *"dig out the actual main goal of hero that we iterated on - its
not tools and calls and shit - its well specd and discussed. id rather
not try to repeat it - so dig it up"*.

Mission has been articulated repeatedly across at least 6 specs in
remarkably consistent language:

- `buddy-model-architecture` (deepest): *"Big models like Claude are
  stateless. Every request starts cold. Hero is fundamentally a context
  engineering system: it structures project knowledge so the right
  information lands in the window at the right time."*
- `hero-v2-system-design`: *"Hero v2 transforms Hero from a solo
  productivity tool into the institutional memory layer for an entire
  engineering team. Every AI agent session benefits from collective
  project knowledge."*
- `hero-serve-daemon`: *"The model starts cold every session and only
  gets knowledge that someone thinks to inject upfront."*
- `graph-memory`: *"For hero — which is fundamentally a
  context-engineering system across sessions, tools, and teams — graph
  is the right substrate."*
- `context-injection`: *"Before an agent writes code, Hero searches the
  spec corpus and injects relevant context… it ensures agents never
  work blind."*
- `AGENTS.md`: *"ephemeral Q&A becomes permanent institutional memory."*

Synthesis: **Hero is a context-engineering system for AI coding tools.
Its job is to make sure the AI working in your editor has the right
project knowledge in its window at the right moment — including the
stuff nobody told it. The spec-driven workflow is the surface; the
corpus that compounds across sessions, people, and time is the
substance. v2's ambition is to make that corpus a team-shared
institutional memory layer, so Dev A's hard-won lesson becomes Dev B's
starting context.**

What it explicitly is NOT:
- Not an AI assistant or copilot (no LLM in the binary — hard rule)
- Not a code generator (the agent tool does that; Hero feeds it)
- Not a wiki (machine-readable, agent-consumable, queryable)
- Not a config file like cursorrules (a corpus + workflow)
- Not "agentic platform" / "AI ops" (banned terms in positioning)

User confirmed the read with: *"i think the mission is pretty well
captured"*.

### 7. Add experience principles as a second-tier guiding layer

User: *"the it has to just work - it has to work for the smart AI
person - the model - i has to make the less experienced AI person
magically do the right things - it cant' be a do this command then
that command etc.. it has to be natural language triggers the right
thing. tools are for the hard core user. sessions have to magically
start up as though they know all and were watching the whole time.
sessions have to end enhancing everything everyone is doing."*

Vocabulary: this layer is **principles** (or *tenets* in
Amazon-speak). Mission says *what*; principles say *how it must feel*;
ACs say *whether a feature delivered*.

Five principles (drafted, locked in get-back-on-track):

1. **It just works.** Zero ceremony. No command sequences.
2. **Natural language is the interface; tools are the escape hatch.**
3. **Sessions start omniscient.** No re-priming.
4. **Sessions end making everyone smarter.** Zero ephemera; the corpus
   always grows.
5. **Two audiences, one product: the model and the human.** Magic for
   the model and less-experienced human; tools for the practitioner.

Critical realization: **every gap in the audits violates at least one
principle.**

### 8. The mission + principles capture is itself a Hero feature

User: *"so we are doing this - but also probably crafting an important
feature about laying the mission and guiding principles for your
project that will probably make using hero work better yeah? we should
capture that"*.

This is a missing layer in every existing AI-engineering tool. Cursor
rules, CLAUDE.md, project-template scripts: all live at the
style/conventions layer. Nobody captures the **teleological layer** —
what is this project for, how should it feel — and pumps it into every
agent session.

Becomes feature `project-charter`:
- Hand-authored `.hero/mission.md` (mission + principles + locked
  vocabulary + anti-patterns)
- `hero init` wizard proposes a draft synthesized from README + recent
  commits + first conversation (principle #1 demands magic here)
- Required spec frontmatter fields: `mission_alignment:` and
  `principles_check:`
- Auto-injected into every agent context bundle as highest-priority
  context
- `hero check` rejects specs missing alignment fields
- `hero check mission` scores recent specs/commits against alignment;
  surfaces drift
- Charter stored as `Mission` graph node, bitemporal — edits preserve
  history; `hero why <decision>` traverses to the version-of-mission
  active when the decision was made

Hero gets its own prevention layer immediately. We dogfood the feature
on our own recovery — if our drift is uncatchable from this point, that
is the demo for every customer.

### 9. The mission framing the user landed on (after the specs were written)

While the six feature specs were being written, three short user
messages locked the mission in its sharpest form yet. Captured
verbatim because the framing is much better than my synthesis:

- *"i could start from scratch on a mission - but i think it would be
  loose and sloppy off the top of my head - and you did dig up a lot
  of past stuff here and did a good job teasing it out to something i
  struggled to convey."* → consent to draft the mission file from the
  quoted-source synthesis.

- *"ideally were fully on the same page - use the harness - interact -
  build, design, create, execute, fix, plan blah blah - do stuff - and
  hero is the side kick brain that makes us all smarter together -
  growing with every turn, and raising our floor together."* → the
  mission, in human language. Key terms locked: **harness** (the
  user's AI coding tool — Claude Code, Cursor, etc.), **sidekick
  brain** (what Hero is to the user/model team), **growing with every
  turn** (the compounding mechanic), **raising our floor together**
  (lift everyone, not just the senior dev).

- *"we should never start another session having to explain what we
  are trying to build - and having to re-iterate on our goals and
  plans to get back to where we left off."* → principle #3 ("sessions
  start omniscient") in its sharpest form. Re-priming is a system
  failure.

These three messages became the body of [`.hero/mission.md`](../../../mission.md)
— quoted verbatim where possible. The mission file is now the
authoritative artifact; this note records the conversation that
produced it.

### 10. Two layers, one manifesto: core Hero vs. verticals

User: *"we always need to be thinking in terms of 'core hero' - the
engine and mechanisms to always capture the knowledge and decisions
and changes and trials and failures and fixes and everything possible
you would just insert into anyone to make them understand everything
that has gone on - and - what here is a vertical application of hero
with a surrounding tool box to make it happen - we have 'hero code'
here - a blend of project management, product management, engineering,
testing, release readiness etc.. but that is a specific vertical set
of skills to guide proper flow of the core. we could make new
verticals that have nothing to do with coding - and it should have the
same manifesto right."*

Critical architectural framing locked here, not new but never
articulated this cleanly before. Two layers:

- **Core Hero** — the engine. Capture, structure, project, inject,
  sync, compound. Domain-agnostic. The substrate.
- **Verticals** — domain-specific toolkits riding on the core. Each
  adds spec types, agents, skills, commands, vocabulary tailored to
  one kind of work. Hero Code is one. Hero Sales is the next. Hero
  Legal / Research / Support / Design are open territory.

The manifesto (mission + principles) lives at *core*. Every vertical
inherits it unchanged.

This wasn't a new idea — `domains/engineering/`,
`multi-domain-core`, `hero-sales`, `domain-plugin-architecture`
specs already existed. But the layering wasn't load-bearing in any
artifact, so v2 work blended core + Code-vertical concerns
indiscriminately. Part of why the drift looked random.

The user's framing of *"everything you'd want to insert into a new
person to make them understand everything that has gone on"* is the
sharpest core-mission statement to date — and is now in
[.hero/mission.md](../../mission.md) verbatim.

### 12. Make the layering physical + scaffold a second vertical

User: *"would it make sense to plan out a new structure? does core
live at the root - do we put verticals (right word?) as a sub folder?
do we rethink the commands, agents, etc all we have as what is core
- what is engineering - which just happens to be the first vertical
we tackled as we built her?"* and *"how does missions lay out when
there are multiple. i'm sure you have good ideas. maybe we should
prepare for a hero-sales vertical so we have 2 sitting here to help
guide when there's questions on what lives in core or not."*

Captured as new feature spec
[core-vertical-layering](../../planning/features/core-vertical-layering/spec.md)
which extends the in-flight `domain-plugin-architecture` work
(audit said it was 80% done — `hero domain` CLI missing).

Key triangulation insight: with only one vertical (engineering),
it's structurally impossible to tell what's *engineering-specific*
from what's *core but happens to look engineering-shaped because
it's the only example we have*. Scaffolding Hero Sales now (just
the mission file + skeleton dirs) gives us a second specimen.
Suddenly "would this serve sales too?" becomes a checkable question.

Done in this turn:
- [`domains/sales/mission.md`](../../../domains/sales/mission.md)
  written as scaffold (mirrors Hero Code structure with sales
  vocabulary)
- [`domains/sales/AGENTS.md`](../../../domains/sales/AGENTS.md)
  placeholder
- `domains/sales/{agents,commands,skills,spec-types}/` empty dirs
- `core-vertical-layering` spec includes a triangulation table
  walking every existing root-level agent / command / skill and
  classifying as core / engineering / ambiguous

The proposed structure (per the spec):
- `core/` at root for universal agents/commands/skills
- `domains/<name>/` per vertical (engineering, sales, future)
- `internal/`, `cmd/`, `cloud/` stay where they are (engine
  implementation)
- One open question flagged: `internal/codescan/` is
  engineering-specific despite living in `internal/` — resolve in
  triangulation review

### 14. Continuous per-feature smoke testing — never big-bang again

User: *"we should capture that we need to actually make a real world
automated smoke test on everything we do and run and verify it for
every hero command and feature as we rework them all - instead of a
big bang at the end where we realized we were so far off track."*

Direct lesson from the v2 audit. The monolithic
[`scripts/e2e_smoke.sh`](../../../scripts/e2e_smoke.sh) ran once,
surfaced ~9 issues; phases 4–10 went weeks without targeted
exercise; "delivered" features silently regressed for weeks.

Captured as new feature spec
[per-feature-smoke-coverage](../../planning/features/per-feature-smoke-coverage/spec.md)
which defines three layers of smoke:

| Layer | Scope | Frequency |
|---|---|---|
| Per-feature | One spec → one script | Every commit touching it |
| Per-area | 8 area suites | Nightly + PR |
| Full e2e | Everything | Weekly + release tag |

Key mechanisms:
- New spec frontmatter field `smoke:` — required by `hero check`
- One script per feature in `scripts/smoke/<slug>.sh`
- Built-in `--smoke` flag on every CLI command (so even unscripted
  features have *something* runnable)
- `hero smoke --since <ref>` runs only smokes whose features were
  touched in the diff (perfect for pre-commit / PR CI)
- Failing smokes flip AC status in the graph; `hero status` and
  `hero blocked` surface them
- Pre-commit + PR-CI integration

This is the structural fix for big-bang drift. Combined with
`acceptance-criteria-graph` and `spec-status-integrity`, regression
becomes detectable the day it lands, not the month after.

### 13. Verticals can have dedicated UIs — Hero is a platform, verticals are products

User: *"i think we'll have finance, accounting?, marketing, management,
down the road. each with their own desktop app or web app."*

Sharpens the core/vertical framing significantly. A vertical is not
just a content pack (agents + commands + skills + vocabulary). It can
also include its **own dedicated user interface** — desktop or web
app tailored to the domain.

- **Hero Code's UI today is implicit:** the harness (Claude Code,
  Cursor, opencode) + the Hero dashboard
- **Future verticals will have dedicated apps:** Hero Sales for reps
  during calls, Hero Finance for accounting workflows, Hero Marketing
  for campaigns, Hero Management for org/planning views

This means: **Hero is a platform at the core layer; each vertical is
a product built on it.** Closer to how Office ships Word/Excel/
PowerPoint as distinct products on shared infrastructure than how a
plugin extends a host application.

Implications captured in
[core-vertical-layering](../../planning/features/core-vertical-layering/spec.md):
- Vertical directory layout grows a `ui/` subtree (web + desktop +
  shared) alongside agents/commands/skills when the vertical has its
  own app
- Vertical UIs consume core via the same APIs as the harness — MCP
  tools, HTTP API, SSE — meaning the core engine doesn't know or
  care which UI is reading
- Cross-vertical context flows through the shared corpus
  automatically (a sales conversation that informs an engineering
  decision; a finance constraint that shapes marketing) — that's
  core's job
- Building any specific vertical UI is per-vertical-spec work; this
  spec just makes the directory accommodate them

The vertical roster in [.hero/mission.md](../../mission.md) was
updated to include finance, accounting, marketing, management on the
horizon (with `someday` semantics — captured but not now-actionable).

### 11. Mission feedback round (after v2 mission file)

User reviewed the first mission draft and gave point-by-point
feedback that produced v2 of the file plus the engineering vertical
charter. Key changes:

- **Broaden core mission.** First draft said *"sidekick brain for
  AI-driven engineering"*. User: *"its for AI agent collaboration?
  deeper than that?"* — broadened to *"sidekick brain for
  AI-augmented knowledge work"*. Engineering framing moved to the
  Code-vertical charter.
- **Three modes added.** User: *"works amazing by yourself in local
  mode... team mode - we all are a hive mind made possible by hero -
  and thats where hero cloud comes in... the 3rd mode is really just
  running your own hero cloud locally if you don't want to or aren't
  allowed to leverage the cloud service."* — captured as a top-level
  mission section.
- **Anti-pattern reordering.** User: *"we might want to add something
  about doing work without a spec is bad too - before the spec /
  acceptance criteria"* — added "Work without a spec" as the first
  anti-pattern.
- **Sub-verb sprawl reframed.** User: *"i don't think i get the
  sub-verb sprawl"* — reframed as "Required command sequencing":
  the bad thing isn't sub-verbs (which are fine when they organize
  related but independent actions), it's requiring users to chain
  multiple commands to accomplish one logical task.
- **LLM rule softened.** User: *"could we forsee a future of a
  fronting or growing local tiny model that would be a much more
  awesome way to achieve what were trying to do with markdown docs
  and a graph db?"* — reframed as "Hero competing with the harness's
  brain": the rule is "don't take the harness's job," not "no LLMs
  ever." Local models for substrate work (extraction, projection,
  entity resolution) are fair game.
- **Future-plans-as-noise anti-pattern.** User: *"we have captured
  marketing stuff - we have to think about it someday - but we don't
  want to imply its more important to work on now - do we need a
  mechanism to prioritize? its good to not lose stuff - but its bad
  when future plans become noise to the now"* — added "Future plans
  drowning the now" anti-pattern AND spawned a new feature spec:
  [spec-prioritization](../../planning/features/spec-prioritization/spec.md)
  with a `horizon: now|next|someday|parking` mechanism.

User reaction to the round: *"overall - amazing. better than i could
have done alone."* Locking the mission file at version 2.

## Final user instruction

*"yes capture it all - but also lets make a get back on track
initiative and capture this entire convo into a well crafted set of
specs to make it a reality"*.

Result: `get-back-on-track` initiative + 6 child feature specs +
companion audit-findings note + the locked mission file. This file
is the conversation capture.

## What this note is for

When the next session resumes work on the recovery, read this **first**
— before the initiative spec, before the feature specs. The plan is
only fully comprehensible with the meta-reasoning that produced it,
which the strict-form specs lose.

The pattern this session also proved out: **a single magic
back-and-forth produces 8 strategic moves in 90 minutes that would have
taken weeks to converge on through PR review or async docs**. Hero's
mission of "preserve session magic as institutional memory" applies
recursively to itself. This note is the proof-of-concept.
