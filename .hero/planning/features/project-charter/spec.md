---
title: Project Charter — Mission, Principles, and Auto-Injection
type: feature
status: planning
priority: P0
tags: [mission, principles, charter, dogfood, foundational, prevention]
created: 2026-04-28
relations:
  - target: get-back-on-track
    kind: parent
  - target: acceptance-criteria-graph
    kind: sibling
  - target: spec-status-integrity
    kind: sibling
mission_alignment: |
  Captures the project's mission and experience principles as a single,
  versioned, auto-injected artifact — so every agent session starts
  knowing what the project is for and how it should feel. This is the
  prevention layer that makes "right context at the right moment"
  recursively true: the mission itself becomes context for every model
  call.
principles_check: |
  Serves #1 (it just works — wizard proposes draft from existing
  artifacts, user doesn't stare at blank file), #3 (sessions start
  omniscient — mission is the first thing the model sees), #5 (two
  audiences — the file is human-authored once; agents inject it
  automatically forever).
horizon: next
smoke: deferred
---

## Goal

Make the mission and experience principles a load-bearing artifact in
every agent session and every spec authored — not prose buried in
six different specs that nobody reads.

## Why now

The recovery audit established that the v2 drift was caused, mechanically,
by the absence of an enforced charter. Mission lived as quotes across
six specs. Principles lived in nobody's head. New work was never asked
*"does this serve the mission?"* because there was nowhere for the
question to be checkable. Every drift mode the audit catalogued is a
violation of the principles being captured here.

Crucially, this is not a Hero-internal artifact — it's a missing layer
in *every* AI engineering tool. Cursor rules, CLAUDE.md, project
templates: all live at the style/conventions layer. Nobody captures
the **teleological layer** (what is this project for, how should it
feel) and pumps it into every agent session. This feature ships as a
real Hero feature for every project, dogfooded on Hero itself.

## Surface

### `.hero/mission.md` — the artifact

Single hand-authored markdown file. ≤200 lines. Five sections:

```markdown
---
title: <Project> Charter
locked_at: 2026-04-28
locked_by: <person>
version: 1
---

## Mission

<the 12-word version on line 1, then 1-2 paragraphs maximum>

## Principles

1. <name>. <one-line statement.>
2. ...

## Vocabulary

- Preferred: <term> — <what it means, why we use it>
- Banned: <term> — <why we don't use it>

## Anti-patterns

- <thing the project must never become>

## Mission-fit test

<one question every new spec, commit, and PR must answer in one sentence>
```

### `hero init` — the wizard

Principle #1 demands magic here: never make the user stare at a blank
file. The wizard:

1. Reads `README.md`, recent commits (last 30 days), any existing
   high-status specs, the project's CLAUDE.md / AGENTS.md
2. Synthesizes a draft mission and 3–5 principles
3. Presents the draft to the user with one-line edit affordance per
   section
4. On accept, writes `.hero/mission.md` and locks it as a graph node

For an existing project running `hero init` over a populated workspace
(the Hero recovery case), the wizard mines the same sources.

### Auto-injection

Mission file content is the **highest-priority block** in every
context bundle:

- `hero context <files>` — mission first, then file-scoped context
- `hero relevant <file>` — mission first, then relevant ACs/decisions
- `hero resume` / `hero next` — mission first, then session state
- Every MCP context tool (`hero_context`, `hero_relevant`,
  `hero_resume`, `hero_recap`, `hero_recall`) — mission first
- Spec-template scaffold — mission rendered into the docstring header
  of every new spec (commented-out, for the author to read while
  writing)

### Spec frontmatter — required fields

After charter ships, every new spec frontmatter requires:

```yaml
mission_alignment: |
  <one paragraph: how this spec serves the mission>
principles_check: |
  <which principles this spec serves; which it might violate; mitigation>
```

`hero check` rejects specs without these fields. `hero spec new`
template includes them empty (with prompts).

### `hero check mission` — drift scoring

Reads recent specs (last 30 days) and recent commits (last 30 days),
checks their `mission_alignment` declarations against the locked
charter, surfaces:

- Specs whose alignment claim is vague or contradicts a principle
- Commits whose changes touch areas with no current mission-aligned
  spec
- Vocabulary drift (use of banned terms)
- Anti-pattern matches in commit diffs

Output is advisory in v1 (a report). Enforcement (block-on-violation
in pre-commit hook) is opt-in via `hero.json` setting.

## Graph representation

Mission is a `Mission` node, bitemporal:

```
Mission { key: "<project-slug>", version, statement, principles, vocab,
          anti_patterns, locked_at, locked_by, valid_from, valid_to }
```

Edits create new rows with new `valid_from`; old rows get `valid_to =
now`. So:

- `hero why <decision>` traverses to the version-of-mission that was
  active when the decision was made — proves whether the decision was
  mission-aligned at *its* time, not today's
- Mission revisions are auditable — when did principle N change and
  why
- Bitemporality stays load-bearing for the most important node type

## Acceptance criteria (build-out-as-we-go set)

**AC-1:** ✅ **passing** (commit `0c5904d`, 2026-04-28).
`.hero/mission.md` is parsed by `internal/mission` and upserted as a
`Mission` graph node keyed by scope (`core` for the project charter)
on `hero scan`. Idempotent re-ingest is a no-op via content-hashed
upsert. Verified end-to-end: scan prints
`Graph mission: core v2 (5 principles, 6 anti-patterns)`; SQLite
shows `Mission core` in the graph. Round-trip + parser tests cover
real-charter parsing, vocab bullets, numbered principle headings,
and idempotency.

**AC-2:** `hero init` on a populated repo (using the hero repo itself
as the test case) produces a draft mission file the user accepts with
≤5 edits.

**AC-3:** ✅ **partial passing** (commit `1d891e0`, 2026-04-28).
`hero resume` gains a top-level Mission section in its digest
(`internal/digest/digest.go missionSection`) — title, statement,
principles list, and mission-fit test all rendered as the first
block. `hero relevant` and `hero deliver` print a one-line
`Mission — <statement>` preamble before their primary output via
`mission.Preamble`. **Deferred:** `hero context` (no current
top-level CLI verb) and the 7 MCP context tools — these gain the
preamble in a follow-up since the surface is shared infrastructure
(MCP server) and doesn't share the digest path.

**AC-4:** `hero spec new <slug>` scaffolds a spec with
`mission_alignment:` and `principles_check:` fields populated with
prompts.

**AC-5:** `hero check` exits non-zero on any spec missing
`mission_alignment` or `principles_check`. Verified by intentionally
malformed test fixture.

**AC-6:** `hero check mission` runs against the hero repo's last 30
days and produces a report with at least the known violations from
the v2 audit (the speculative marketing/distribution specs surface as
mission-drift candidates).

**AC-7:** Mission edits create a new graph row with new `valid_from`;
old row gets `valid_to`; `hero why <node>` traverses to the
mission-version active at the node's `valid_from`.

ACs accrete as the recovery surfaces edge cases. Adding a new AC
follows the rule: write it into this spec first, then the script.

## Approach

**Phase 1 — file format + ingest** (~1 day): Define the markdown
structure, write the parser, add `Mission` node type, wire ingest to
`hero scan`. Round-trip test against the canonical hero charter file.

**Phase 2 — injection** (~1 day): Add mission-block prepend to all
context-emitting commands and MCP tools. Snapshot tests.

**Phase 3 — wizard** (~2 days): The hard part. Read README + commits +
existing specs, synthesize a starter draft. Probably uses Tier-2 LLM
(Haiku 4.5) since extraction quality matters. Behind feature flag for
the first ship.

**Phase 4 — spec-template + check** (~1 day): Required frontmatter
fields, scaffold updates, `hero check` validation, drift scoring.

**Phase 5 — Hero's own charter** (~½ day, immediate): Hand-author
`.hero/mission.md` for the hero project from the locked language in
[recovery-strategy-conversation](../../../knowledge/notes/recovery-strategy-conversation/spec.md)
and [hero-positioning](../../features/hero-positioning/spec.md). This
happens *before* the engineering work so we dogfood the artifact even
while building the support for it.

## Out of scope

- Multi-project federation of charters (one team's mission seen by
  another team's repos) — defer until single-project loop is proven
- Charter version diff visualization in dashboard — nice-to-have
- LLM-based mission-fit scoring beyond keyword/principle matching —
  later

## Open questions

- Should `mission_alignment` and `principles_check` be required on
  *all* spec types (notes, decisions) or only features/initiatives?
  Lean: features/initiatives only; notes/decisions are explicitly
  exploratory.
- Does the wizard always re-run on `hero init`, or only the first
  time? Lean: prompts to re-run if the charter is older than 90 days.
- Can multiple repos in a `repos:` config share a mission? Lean: yes
  via `mission: <path>` override in `hero.json`.
