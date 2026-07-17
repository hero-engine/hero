---
title: "Story Detail + Intake Scrubber Backing — Dependency Mapper, Duplicate Intake Scrubber"
slug: story-detail-and-intake-scrubber-backing
type: feature
status: completed
domain: pm
priority: high
size: small
created: 2026-07-17
tags: [pm, story-detail, intake, scrub, wave-1]
relations:
  - target: pm-pack-completion
    kind: parent
  - target: pm-doctrine-and-skill-backfill
    kind: depends-on
  - target: remaining-roles-scrubbers-and-launch
    kind: conflicts-with
completed_at: 2026-07-17T21:37:02Z
---

# Story Detail + Intake Scrubber Backing — Dependency Mapper, Duplicate Intake Scrubber

## Goal

Two live hero-code surfaces currently draw buttons with no backing agent:
Story Detail's **"Show dependencies"** and Intake Funnel's **"Cluster recent"**.
This spec authors the two agents that back them — `dependency-mapper` and
`duplicate-intake-scrubber` — and scaffolds the shared `/scrub <concern>`
command with its first concern (`intake`) routing to the scrubber. "Done" means
both agent files exist and load `pm-agent-doctrine` (plus their domain skills),
`scrub.md` exists with a clearly-marked, append-only concern-dispatch region so
Child #11 can add `roadmap` + `stories` concerns without touching the intake
block, and AGENTS.md registers the new routes/agents/command **additively** —
no canonical rows or prior children's routes edited, no dangling skill refs.
Content-only, entirely within `domains/pm/`; no Go.

## Kickoff

Backs Story Detail "Show dependencies" (`dependency-mapper`) and Intake
"Cluster recent" (`duplicate-intake-scrubber`), and scaffolds the shared
`/scrub` command with the intake concern.

**Status:** planning — spec materialized from the Child #5 stub; no files
authored yet. Depends on Child #1 (`pm-doctrine-and-skill-backfill`) for the
doctrine spine.

**Pick up at:** author `domains/pm/agents/dependency-mapper.md` first (mirror
`duplicate-detector.md` frontmatter shape; load the four skills), then the
scrubber, then scaffold `scrub.md` with the `<!-- SCRUB CONCERNS -->` marker.

→ `/deliver story-detail-and-intake-scrubber-backing`

**Files:** `domains/pm/agents/duplicate-detector.md` (shape ref),
`domains/pm/skills/dependency-mapping/SKILL.md`,
`domains/pm/commands/triage.md` (command shape ref),
`domains/pm/AGENTS.md:62` (Wave-2 append marker)
**Skip:** don't edit the canonical AGENTS.md routing table or prior children's
Wave-2 subsections — additions only.

## Problem

The 2026-07 PM pack audit (`pm-pack-audit-2026-07.md`, §Surface coverage matrix)
flags two partial surfaces:

- **Story Detail** ships `story-writer`, `handoff-coordinator`, `pm-reviewer`,
  `duplicate-detector` — but its **"Show dependencies"** button has no
  `dependency-mapper` behind it. A PM clicking it gets nothing.
- **Intake Funnel** ships `intake-triager` + `duplicate-detector` (the
  write-time live detector) — but its **"Cluster recent"** button has no
  `duplicate-intake-scrubber`. The live detector runs one-item-at-a-time at
  write time (Height pattern); it cannot catch dups that only become visible
  when you cluster a *batch* of recent intake whose vocabulary drifted apart.

Both agents were designed in `agent-pack-design.md` (§C.4 `dependency-mapper`,
§C.9 `duplicate-intake-scrubber`) as P1 and deferred out of the v1 minimum
pack. The shipped foundation they need is now in place: `dependency-mapping`,
`duplicate-detection`, and `pm-agent-doctrine` skills all exist on disk, and
`cross-domain-graph-query` + `risk-surfacing` skills ship. The agents just
need authoring.

The audit also calls for a shared `/scrub <concern>` command (Wave 1 command
row). This child scaffolds it with the **intake** concern only; Child #11
(`remaining-roles-scrubbers-and-launch`) later extends the *same file* with
`roadmap` (`stale-roadmap-scrubber`) and `stories` (`ambiguous-story-scrubber`)
concerns. This is the **5↔11 same-file seam** (initiative spec, "Intake
scrubber ↔ launch roles same-file seam"): the reciprocal `conflicts-with` is on
both children, and this child must structure `scrub.md` so #11 only *appends* —
never edits the intake block.

## Acceptance Criteria

- **AC-1:** THE SYSTEM SHALL provide `domains/pm/agents/dependency-mapper.md`,
  `domains/pm/agents/duplicate-intake-scrubber.md`, and
  `domains/pm/commands/scrub.md` as files on disk.
- **AC-2:** THE SYSTEM SHALL have both new agents load `pm-agent-doctrine`
  (the shared corpus-grounding / decision-gate / compare-don't-replace
  doctrine) in their skill-load step.
- **AC-3:** THE SYSTEM SHALL have `dependency-mapper.md` load
  `dependency-mapping`, `cross-domain-graph-query`, and `risk-surfacing`, and
  surface dependencies read-only (walk the graph and report; propose, never
  auto-edit graph state).
- **AC-4:** THE SYSTEM SHALL have `duplicate-intake-scrubber.md` load
  `duplicate-detection` and operate as a **batch/cluster** pass that is
  report-only — it recommends merges but performs **no auto-merge** — and
  states how it differs from the write-time `duplicate-detector`.
- **AC-5:** THE SYSTEM SHALL structure `scrub.md` with a clearly-marked
  append-only concern-dispatch region (a `<!-- SCRUB CONCERNS: ... -->`
  comment naming intake as this child's concern and roadmap + stories as
  Child #11's) such that Child #11 can add concerns without editing the
  intake block.
- **AC-6:** WHEN `/scrub intake` is invoked THE SYSTEM SHALL route to
  `duplicate-intake-scrubber`, and `scrub.md` SHALL document the dispatch of
  the concern argument to a concern-specific agent.
- **AC-7:** THE SYSTEM SHALL append the new routes to `domains/pm/AGENTS.md`
  **below** the `<!-- WAVE-2 ROUTES -->` marker and **after** prior children's
  subsections, leaving the canonical routing table (lines above the marker)
  and every prior child's Wave-2 subsection byte-for-byte unchanged.
- **AC-8:** THE SYSTEM SHALL list `dependency-mapper` and
  `duplicate-intake-scrubber` in the AGENTS.md **Agents Reference** and `/scrub`
  in the AGENTS.md **Commands Reference** roster.
- **AC-9:** THE SYSTEM SHALL introduce no dangling references — every skill
  named in either agent's load list resolves to an existing
  `domains/pm/skills/<name>/SKILL.md`, and every agent named in `scrub.md` or
  the new AGENTS.md routes resolves to an existing `domains/pm/agents/<name>.md`.

## Changes

Author only under `domains/pm/` (pack source). Do not touch any installed
harness copy (`.claude/`, `.agents/`, `.codex/`) — those regenerate on
`hero install`. This honors the `harness-changes-cover-all-targets` tripwire:
the single source edit is the pack source.

1. **Create `domains/pm/agents/dependency-mapper.md`** — the Story Detail
   "Show dependencies" backing agent.
   - Frontmatter mirroring `duplicate-detector.md`: `name: dependency-mapper`,
     a one-line `description` (surface dependencies across items/epics/stories
     incl. cross-domain to engineering features; report-only), `mode: subagent`,
     `temperature: 0.1`, `color: secondary`, `permission:` with `edit: deny`
     (it walks and reports; it does not author specs or mutate graph state),
     `task: {"*": deny}`, `skill: {"*": allow}`, `webfetch: allow`.
   - Persona opener: "You are a dependency mapper."
   - **Skill-load step** naming exactly: `pm-agent-doctrine`,
     `dependency-mapping`, `cross-domain-graph-query`, `risk-surfacing`.
   - **When invoked:** contextual "Show dependencies" button on a Story Detail
     / feature; `/prioritize` (so ranking sees real start dates); `/handoff`
     (so the packet carries upstream dependency context); natural-language
     "what's blocking X" / "what does Y unblock".
   - **Workflow:** read the target artifact; forward walk ("what does this
     block?") and backward walk ("what blocks this?") per `dependency-mapping`;
     traverse cross-domain edges to engineering features via
     `cross-domain-graph-query`; distinguish hard blocker / soft sequencing /
     coupling; surface transitive chains to terminal state; for a `delivering`
     hard blocker surface "blocker in progress, ETA" not binary blocked; flag
     at-risk chains for `roadmap-curator`. Use `hero_why` for backward walks.
   - **Doctrine binding:** every surfaced dependency is a *proposal* the human
     reads (decision-gate); no auto-edits to graph edges; ground each chain in
     actual graph edges, not inferred coupling (corpus-grounding).
   - **Produces:** a read-only dependency report (forward + backward chains,
     blocker types, cross-domain crossings, ETAs). Does not write to any spec.
   - **Anti-patterns:** one-hop-only analysis; mislabeling coupling as
     dependency; treating hard blockers as binary; trusting tracker over graph;
     auto-recording edges.
   - **Prior art:** `agent-pack-design.md` §C.4 dependency-mapper prompt sketch.

2. **Create `domains/pm/agents/duplicate-intake-scrubber.md`** — the Intake
   Funnel "Cluster recent" backing agent; the batch/cluster variant of the
   shipped write-time `duplicate-detector`.
   - Frontmatter mirroring `duplicate-detector.md`: `name:
     duplicate-intake-scrubber`, one-line `description` (batch/cluster recent
     intake to surface dups the live detector missed; report-only, no
     auto-merge), `mode: subagent`, `temperature: 0.1`, `color: secondary`,
     `permission:` with `edit: deny`, `task: {"*": deny}`,
     `skill: {"*": allow}`, `webfetch: allow`.
   - Persona opener: "You are a duplicate intake scrubber."
   - **Skill-load step** naming exactly: `pm-agent-doctrine`,
     `duplicate-detection`.
   - **Differentiation section (required):** explicitly contrast with
     `duplicate-detector` — the live detector runs synchronously on a *single*
     item at write time (Height pattern, recall-first, one candidate list);
     the scrubber runs a **background/batch sweep over a window of recent
     intake**, clustering N items to catch dups that only surface after
     vocabulary drift, cross-segment paraphrase, or accumulated volume — the
     misses the write-time single-item pass structurally cannot see. Cite
     `duplicate-detection`'s "Running duplicate detection only at intake
     write-time" anti-pattern (background sweeps are the complement).
   - **When invoked:** `/scrub intake`; contextual "Cluster recent" button on
     the Intake Funnel; a weekly/cron sweep.
   - **Workflow:** enumerate recent intake (window: default last N days or the
     `new`/untriaged queue); cluster by the `duplicate-detection` overlap
     signals (lexical → theme → segment → source-cluster → cross-domain);
     produce a **cluster report** with per-cluster confidence and the specific
     field overlap; recommend a canonical survivor per cluster.
   - **Report-only / no auto-merge (hard rule):** recommends merges; the merge
     decision is human-confirmed (decision-gate doctrine + `duplicate-detection`
     "Never auto-merge"). Preserve source attribution.
   - **Produces:** a cluster report (ranked clusters, overlap evidence, merge
     recommendations); an explicit "no clusters found" when the sweep is clean.
   - **Anti-patterns:** auto-merging; opaque similarity scores with no field
     overlap; ignoring cross-domain dups; collapsing source attribution;
     duplicating the write-time detector's job instead of complementing it.
   - **Prior art:** `agent-pack-design.md` §C.9 duplicate-intake-scrubber.

3. **Create `domains/pm/commands/scrub.md`** — SCAFFOLD the shared
   `/scrub <concern>` command with the **intake** concern, structured for
   append-only extension by Child #11 (the 5↔11 seam).
   - Frontmatter `description:` for a concern-dispatched scrub command.
   - A short pre-flight: read the `<concern>` argument; if absent, list the
     available concerns and ask.
   - **Concern-dispatch region** — a clearly-marked block that maps each
     concern to its agent. It MUST open with an HTML comment marker that names
     the ownership boundary, e.g.:
     ```
     <!-- SCRUB CONCERNS: intake (this child, story-detail-and-intake-scrubber-backing);
          roadmap + stories appended BELOW by remaining-roles-scrubbers-and-launch (#11).
          #11 APPENDS new concern blocks only — do not edit the intake block above. -->
     ```
   - The **intake** concern block (above/before that append point): route
     `/scrub intake` to `duplicate-intake-scrubber` — cluster recent intake,
     surface missed near-duplicates, emit the cluster report + merge
     recommendations; no auto-merge.
   - Structure so #11's `roadmap` (→ `stale-roadmap-scrubber`) and `stories`
     (→ `ambiguous-story-scrubber`) concern blocks slot in *after* the marker
     without editing the intake block. Do not pre-author those two concerns —
     reference them only in the marker comment as #11's to add.
   - **Report-only doctrine note:** scrub concerns surface findings and
     recommended actions; they never auto-apply state changes (decision-gate).
   - Pass `$ARGUMENTS` through, mirroring `triage.md`'s shape.

4. **Append to `domains/pm/AGENTS.md`** — additions only.
   - **Wave-2 routes region** (below the `<!-- WAVE-2 ROUTES -->` marker at
     ~line 62, **after** the existing `#### Wave-2 ...` subsections for prior
     children): add a new subsection, e.g. `#### Wave-1 backing routes
     (story-detail-and-intake-scrubber-backing)`, with a routing table:
     - "Show dependencies", "what's blocking X", "what does Y unblock",
       cross-domain dependency chain → invoke `dependency-mapper` directly
       (agent — backs the Story Detail "Show dependencies" button; no `/scrub`
       needed).
     - "Cluster recent", "scrub intake", "find dups the detector missed",
       batch dedup sweep → `/scrub intake` → `duplicate-intake-scrubber`
       (complements the write-time `duplicate-detector`).
     - Do NOT modify the canonical table above the marker, the marker comment
       itself, or any prior child's subsection.
   - **Agents Reference** roster: add a "PM Wave-1 Story-Detail / Intake
     backing" bullet (or extend the existing PM roster line) naming
     `dependency-mapper` (Story Detail "Show dependencies") and
     `duplicate-intake-scrubber` (Intake "Cluster recent"; batch complement to
     `duplicate-detector`).
   - **Commands Reference** roster (`### Commands Reference`): add `/scrub` to
     the PM commands list with a one-line gloss (concern-dispatched workspace
     scrub; intake concern ships now, roadmap + stories via #11).

## Validation

Content-only; verify by static checks that the files exist, load the right
skills, preserve the append-only seam, and introduce no dangling references.

```bash
cd /Users/bwheeler/projects/hero-engine/repository/hero

# AC-1: the three files exist
for f in domains/pm/agents/dependency-mapper.md \
         domains/pm/agents/duplicate-intake-scrubber.md \
         domains/pm/commands/scrub.md; do
  test -f "$f" && echo "OK exists: $f" || echo "MISSING: $f"
done

# AC-2: both agents load pm-agent-doctrine
grep -q pm-agent-doctrine domains/pm/agents/dependency-mapper.md && echo "OK doctrine: dependency-mapper"
grep -q pm-agent-doctrine domains/pm/agents/duplicate-intake-scrubber.md && echo "OK doctrine: duplicate-intake-scrubber"

# AC-3: dependency-mapper loads its three domain skills + edit:deny
for s in dependency-mapping cross-domain-graph-query risk-surfacing; do
  grep -q "$s" domains/pm/agents/dependency-mapper.md && echo "OK skill $s" || echo "MISSING skill $s"
done
grep -q 'edit: deny' domains/pm/agents/dependency-mapper.md && echo "OK read-only: dependency-mapper"

# AC-4: scrubber loads duplicate-detection, is no-auto-merge, edit:deny
grep -q duplicate-detection domains/pm/agents/duplicate-intake-scrubber.md && echo "OK skill duplicate-detection"
grep -qiE 'no auto-merge|never (auto-)?merge|report-only' domains/pm/agents/duplicate-intake-scrubber.md && echo "OK no-auto-merge stated"
grep -q 'edit: deny' domains/pm/agents/duplicate-intake-scrubber.md && echo "OK read-only: scrubber"

# AC-5 + AC-6: scrub.md has the append-only marker + intake→scrubber route
grep -q 'SCRUB CONCERNS' domains/pm/commands/scrub.md && echo "OK concern marker present"
grep -q duplicate-intake-scrubber domains/pm/commands/scrub.md && echo "OK intake routes to scrubber"

# AC-7: canonical AGENTS.md table + marker unchanged; new routes below marker.
# Confirm the marker still precedes any new subsection and the canonical rows are intact.
grep -n 'WAVE-2 ROUTES' domains/pm/AGENTS.md && echo "OK marker present"
git diff --unified=0 domains/pm/AGENTS.md | grep '^-' | grep -v '^---' \
  && echo "REVIEW: AGENTS.md removed/changed lines above — must be empty for additions-only" \
  || echo "OK AGENTS.md additions-only (no removed lines)"

# AC-8: new agents + command in the AGENTS.md rosters
grep -q dependency-mapper domains/pm/AGENTS.md && echo "OK ref: dependency-mapper"
grep -q duplicate-intake-scrubber domains/pm/AGENTS.md && echo "OK ref: duplicate-intake-scrubber"
grep -qE '/scrub' domains/pm/AGENTS.md && echo "OK ref: /scrub"

# AC-9: no dangling refs — every loaded skill + routed agent resolves on disk
for s in pm-agent-doctrine dependency-mapping cross-domain-graph-query risk-surfacing duplicate-detection; do
  test -f "domains/pm/skills/$s/SKILL.md" && echo "OK skill-file $s" || echo "DANGLING skill $s"
done
for a in dependency-mapper duplicate-intake-scrubber duplicate-detector; do
  test -f "domains/pm/agents/$a.md" && echo "OK agent-file $a" || echo "DANGLING agent $a"
done
```

All `OK` lines present and no `MISSING` / `DANGLING` / `REVIEW` failures =
delivered. The `git diff` additions-only check is the guard for the
5↔11 seam and the AGENTS.md canonical-table invariant.

## Boundaries

- **No Go, no engine changes.** Content-only, entirely within `domains/pm/`.
  Do not touch `internal/`, `cmd/`, or any consumer (hero-code) code.
- **Do not author the roadmap or stories scrub concerns.** Those are Child
  #11's (`stale-roadmap-scrubber`, `ambiguous-story-scrubber`). This child
  only scaffolds `scrub.md` + the intake concern and leaves the marked
  append point for #11. Editing the intake block from #11's side, or
  pre-authoring #11's concerns here, both break the 5↔11 seam.
- **Do not edit the AGENTS.md canonical routing table or any prior child's
  Wave-2 subsection.** Additions only, below the marker, after prior children.
- **Do not modify the shipped `duplicate-detector` agent or the
  `duplicate-detection` / `dependency-mapping` / `pm-agent-doctrine` /
  `cross-domain-graph-query` / `risk-surfacing` skills.** They are the
  foundation this child consumes; changing them is out of scope.
- **No auto-merge / no auto-state-change behavior.** Both agents are
  report-only per pack doctrine; adding write/merge capability is a separate,
  explicitly-out-of-scope decision.
- **No installed-harness edits.** Author pack source only; `.claude/` etc.
  regenerate on `hero install`.

## Completion Ledger

### Acceptance Criteria

| # | Criterion | Status | Note |
|---|---|---|---|
| AC-1 | Three files on disk (dependency-mapper.md, duplicate-intake-scrubber.md, scrub.md) | DONE | `test -f` passes for all three: `domains/pm/agents/dependency-mapper.md`, `domains/pm/agents/duplicate-intake-scrubber.md`, `domains/pm/commands/scrub.md`. |
| AC-2 | Both agents load `pm-agent-doctrine` | DONE | `grep -q pm-agent-doctrine` passes on both agent files; each lists it as the first bullet in its `## Startup` step. |
| AC-3 | dependency-mapper loads `dependency-mapping` + `cross-domain-graph-query` + `risk-surfacing`, read-only | DONE | All three skill names present in `## Startup`; `edit: deny` in frontmatter; workflow walks the graph and reports only (no edits, no state flips). |
| AC-4 | duplicate-intake-scrubber loads `duplicate-detection`, batch/cluster, report-only no auto-merge, differentiates from `duplicate-detector` | DONE | `duplicate-detection` in `## Startup`; `edit: deny`; "Report-only / no auto-merge (hard rule)" section; dedicated "How you differ from `duplicate-detector`" table (when/scope/pattern/catches). |
| AC-5 | scrub.md has a clearly-marked append-only concern-dispatch region naming intake (this child) + roadmap/stories (#11) | DONE | `<!-- SCRUB CONCERNS: intake (this child, story-detail-and-intake-scrubber-backing); roadmap + stories appended BELOW by remaining-roles-scrubbers-and-launch (#11) ... -->` plus a `#11 APPEND POINT` marker below the intake block. |
| AC-6 | `/scrub intake` routes to duplicate-intake-scrubber; dispatch of concern arg documented | DONE | `### Concern: intake` block routes `/scrub intake → duplicate-intake-scrubber`; `## Dispatch` section documents dispatching the concern argument to its concern-specific agent; `$ARGUMENTS` passed through. |
| AC-7 | New routes appended below the WAVE-2 marker, after prior children; canonical table + prior subsections byte-for-byte unchanged | DONE | `git diff --unified=0` shows zero removed lines (additions-only guard passes); new `#### Wave-1 backing routes` subsection sits after PRD Editor & comms block, before "When routing"; marker at line 62 intact. |
| AC-8 | dependency-mapper + duplicate-intake-scrubber in Agents Reference; `/scrub` in Commands Reference | DONE | New "PM Wave-1 Story-Detail / Intake backing" bullet in Agents Reference; new "PM Wave-1 backing" bullet with `/scrub` in Commands Reference; both greps pass. |
| AC-9 | No dangling refs — every loaded skill + routed agent resolves on disk | DONE | All 5 skill files (`pm-agent-doctrine`, `dependency-mapping`, `cross-domain-graph-query`, `risk-surfacing`, `duplicate-detection`) and all 3 agent files (`dependency-mapper`, `duplicate-intake-scrubber`, `duplicate-detector`) resolve via `test -f`. |

### Changes

| # | Changes item | Status | Note |
|---|---|---|---|
| 1 | Create `domains/pm/agents/dependency-mapper.md` | DONE | Created. Frontmatter mirrors `duplicate-detector.md` (`mode: subagent`, `temperature: 0.1`, `color: secondary`, `edit: deny`, `task {"*": deny}`, `skill {"*": allow}`, `webfetch: allow`). Persona "You are a dependency mapper." `## Startup` loads the four skills; When-invoked (Show dependencies / `/prioritize` / `/handoff` / NL); forward+backward walk workflow with `hero_why`, cross-domain traversal, hard/soft/coupling classification, transitive chains, "blocker in progress, ETA"; doctrine binding; produces read-only report; anti-patterns; prior-art cite §C.4. |
| 2 | Create `domains/pm/agents/duplicate-intake-scrubber.md` | DONE | Created. Frontmatter mirrors `duplicate-detector.md` with `edit: deny`. Persona "You are a duplicate intake scrubber." `## Startup` loads `pm-agent-doctrine` + `duplicate-detection`; required differentiation table vs. write-time `duplicate-detector` + cites the "only at intake write-time" anti-pattern; When-invoked (`/scrub intake` / Cluster recent / cron); cluster workflow with the overlap-signal ladder; "no auto-merge (hard rule)"; produces cluster report incl. "no clusters found"; anti-patterns; prior-art cite §C.9. |
| 3 | Create `domains/pm/commands/scrub.md` | DONE | Created. `description:`-only frontmatter (matches `triage.md`). Pre-flight reads `<concern>`, lists concerns if absent. `<!-- SCRUB CONCERNS ... -->` marker + `#11 APPEND POINT` marker delimit the append-only region; `### Concern: intake` block routes to `duplicate-intake-scrubber`; report-only doctrine note; `## Dispatch`; `$ARGUMENTS` passthrough. roadmap/stories NOT pre-authored (left to #11). |
| 4 | Append to `domains/pm/AGENTS.md` — additions only | DONE | New `#### Wave-1 backing routes (story-detail-and-intake-scrubber-backing)` subsection below the WAVE-2 marker, after prior children's subsections, with the two routes (dependency-mapper direct; `/scrub intake → duplicate-intake-scrubber`). New Agents Reference bullet (both agents) and new Commands Reference bullet (`/scrub`). All 4 edits are pure line-insertions — `git diff` shows no removed lines. Canonical table and prior children's Wave-2 subsections untouched. |

### Exercise-the-feature check

- [x] Ran the spec's full `## Validation` bash block verbatim from repo root — every line prints `OK`, zero `MISSING` / `DANGLING` / `REVIEW` failures.
- [x] AC-7 additions-only guard (`git diff --unified=0 | grep '^-'`) returns empty → verified by inspecting the full `git diff domains/pm/AGENTS.md` (4 additive hunks, no removed lines, prior subsections byte-for-byte intact).
- [x] Frontmatter of all three new files mirrors shipped shape (agents match `duplicate-detector.md` key/nesting; `scrub.md` matches `triage.md` description-only) — the shape the `hero install` loader already parses.
- [ ] No live runtime to exercise: this is content-only pack markdown (agents/commands are prompt docs consumed at `hero install` / dispatch time). The "feature" is routing + dispatch + no-dangling-refs, which the static Validation block exercises fully. There is no server/CLI/endpoint to hit for this change.

### Excellence Bar self-check

- Both agents are corpus-grounded and decision-gated (report-only) per `pm-agent-doctrine`: each carries an explicit doctrine-binding / no-auto-mutation section, not just a skill-load line. dependency-mapper grounds chains in real graph edges (not inferred coupling) and proposes missing edges rather than recording them; duplicate-intake-scrubber recommends survivors and preserves source attribution, never merging.
- The scrubber's differentiation from the shipped `duplicate-detector` is a first-class comparison table (when / scope / pattern / catches) plus prose naming the three structural blind spots (vocabulary drift, cross-segment paraphrase, accumulated volume) and citing the exact `duplicate-detection` anti-pattern — not a one-line "this is different."
- The 5↔11 seam is protected by two markers (ownership comment + explicit `#11 APPEND POINT`) so #11 appends without touching the intake block; roadmap/stories are referenced only as #11's to add, never pre-authored.
- AGENTS.md changes are strictly additive (zero removed lines), honoring the canonical-table invariant and every prior child's Wave-2 subsection. The single source edit is pack source only — no installed-harness (`.claude/`/`.codex/`/`.agents/`) files touched — honoring the `harness-changes-cover-all-targets` tripwire and the Boundaries.
