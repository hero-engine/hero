---
title: "Knowledge Context Injection — push captured knowledge into model context at edit time"
slug: knowledge-context-injection
type: feature
status: completed
priority: P2
size: medium
domain: engineering
created: 2026-07-06
completed_at: 2026-07-07T00:00:00Z
delivery_method: drive
tags: [knowledge, context-injection, hero-context, convention-scopes, all-domains]
relations:
  - target: knowledge-surfacing
    kind: parent
  - target: knowledge-content-retrieval
    kind: depends-on
  - target: unified-retrieval-layer
    kind: related
  - target: knowledge-retrieved-through-unified-corpus
    kind: decided-in
delivery_method: manual
---

# Knowledge Context Injection — push captured knowledge into model context at edit time

Phase 2 of the [[knowledge-surfacing]] initiative, and the higher-value half.
Phase 1 ([[knowledge-content-retrieval]]) makes flat knowledge *pullable*
(`hero ask`/`search`). This phase makes code-scoped knowledge *push* — injected
automatically at the moment the model touches the files it governs. **Depends on
P1's ingest.**

## Goal

WHERE a hand-authored knowledge entry declares a code `scope:` (conventions,
decisions, rules), THE SYSTEM SHALL inject it into the model's context when a
matching file is being worked — regardless of whether that entry is authored
flat or as `<slug>/spec.md`. This closes the "capture → drive better info for
models" loop that pull alone doesn't: the model shouldn't have to know to *ask*
for the convention that governs the file it's editing.

## Kickoff

Every ambient-injection surface reads `specs` / `convention_scopes`, which flat
knowledge never enters — so a flat `conventions/contracts-import-discipline.md`
that governs `internal/**` never injects when the model edits there. P1 lands
flat knowledge in the corpus; this phase feeds code-scoped knowledge's `scope:`
globs into `convention_scopes` so the *existing* matchers light up with no
per-surface rewrite: `BuildContext` (`hero_context`), `BuildNudge`
(`hero relevant`), `drift`, `impact` all ride `FindConventionsForFiles`.
Free-form knowledge (battlecards/playbooks) has no scope and stays pull-only, so
injection stays signal, not noise. Start:
`internal/index/index.go` (`IndexSpec` scope population, `FindConventionsForFiles`,
`BuildContext`, `BuildNudge`), `internal/drift/drift.go`,
`internal/impact/impact.go`, `internal/cli/anchor.go`.

## Where it must surface (from the initiative map)

| Moment | Surface | Today | This phase |
|---|---|---|---|
| File-scoped edit | `hero_context` / `BuildContext` | flat knowledge invisible | inject matching flat conventions/decisions/rules |
| Pre-edit nudge | `hero relevant` / `BuildNudge` | invisible | nudge on them at parity with spec.md conventions |
| Drift check | `drift.go` → `FindConventionsForFiles` | invisible | consider flat code-scoped knowledge |
| Impact analysis | `impact.go` → `FindConventionsForFiles` | invisible | same |
| Re-anchor | `hero anchor` (tripwires) | only spec.md-shaped | surface flat tripwires too, if any |

The unlock is single-seam: populate `convention_scopes` from flat code-scoped
knowledge during ingest. Everything above already queries that table, so they
light up together rather than one integration at a time.

## Scope

**In scope (delivered)**
- Knowledge ingest captures `scope:` globs (isolated `knowledge_scopes` table,
  keyed by knowledge slug — no FK to `specs`, preserving isolation). Only
  code-scoped kinds (`convention`, `rule`) carry them into injection;
  `FindKnowledgeForFiles` matches globs to files.
- `BuildContext` / `hero_context` inject matching flat code-scoped knowledge into
  the Conventions / Rules blocks — the primary edit-time model surface.
- `BuildNudge` / `hero relevant` nudge on them at parity with spec.md conventions.
- Free-form knowledge with no `scope:` never injects (pull-only via P1).

**Out of scope**
- **`drift` / `impact` inclusion** and **`hero anchor` flat-tripwire parity** —
  follow-on. These are analysis/anchor surfaces, not edit-time model-context
  injection; wiring them means merging flat knowledge into the shared
  `FindConventionsForFiles` seam (its consumers include drift/impact), a
  low-risk but separate change. Tracked as a follow-on, not claimed here.
- **Flat-decision file-scoped injection** — deliberately excluded for parity:
  spec.md decisions have no file-scope matcher, so flat decisions don't inject
  either. Decisions surface via `hero ask` (P1). (BuildContext routes a scoped
  decision to `ctx.Decisions` if one ever declares `scope:`, but nothing scopes
  decisions today.)
- File-scoped injection of free-form knowledge — pull-only by design.
- Auto-inferring a `scope:` — an entry injects only if its author scoped it.
- Cold-start / session-start knowledge digest — follow-on.

## Acceptance Criteria

- WHERE a flat `.hero/knowledge/conventions/*.md` (or `decisions`/`rules`)
  declares a `scope:` glob matching a path, WHEN that path is passed to
  `hero_context` / `hero context`, THE SYSTEM SHALL include that entry in the
  injected context — verified against `contracts-import-discipline.md` injecting
  for `internal/**` (today: it does not).
- THE SYSTEM SHALL inject flat and `<slug>/spec.md` code-scoped knowledge
  identically — layout must not change whether an entry injects.
- WHEN `hero relevant <files>` runs, THE SYSTEM SHALL nudge on matching flat
  code-scoped knowledge at parity with spec.md-shaped conventions.
- IF a knowledge entry declares no `scope:`, THEN THE SYSTEM SHALL NOT inject it
  into file-scoped context (it stays pull-only via P1).
- THE SYSTEM SHALL NOT inject free-form knowledge kinds (battlecards, playbooks)
  into file-scoped context even when P1 has made them pullable.

(Follow-on, not gating this spec: `drift`/`impact` inclusion and `hero anchor`
flat-tripwire parity — see Out of scope.)

## Design notes / open questions

- **Scope source.** `convention_scopes` is populated in `IndexSpec` from the
  spec's `scope:` frontmatter. The knowledge ingest must run the same population
  for code-scoped knowledge. Cleanest if P1's ingest reuses `IndexSpec`'s scope
  path rather than duplicating the glob-storage logic.
- **Ranking / precedence.** When a flat convention and a spec.md convention both
  match a file, injection order/dedup must be defined — prefer dedup by slug/path,
  then existing convention ordering.
- **Noise budget.** Injected context is a scarce resource. Confirm the existing
  `BuildContext` caps (top-N conventions) apply to the merged set so flat
  knowledge can't flood the block.
- **Anchor tripwire path.** `FindAllTripwires` reads `type='tripwire'` from the
  specs table; flat tripwires reach it only once P1 gives them a `category`/`type`
  row. Confirm the anchor query includes knowledge-category tripwires.

## Validation

- Repro-first: assert a flat scoped convention does **not** inject today, then
  that it does after the change (same file, `hero context`).
- Parity fixture: one flat and one `<slug>/spec.md` convention with identical
  `scope:` — both inject identically.
- Negative: an unscoped battlecard is pullable (P1) but never appears in
  `hero context` output.
- `go test ./...` green; existing `BuildContext`/`drift`/`impact` tests still pass
  with knowledge merged in.
