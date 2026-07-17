---
title: "PRD Editor + Comms Backing — Pitch Author, Stakeholder Communicator"
slug: prd-editor-comms-backing
type: feature
status: completed
domain: pm
priority: high
size: medium
created: 2026-07-17
tags: [pm, prd, comms, wave-1]
relations:
  - target: pm-pack-completion
    kind: parent
  - target: pm-doctrine-and-skill-backfill
    kind: depends-on
  - target: pm-doctrine-and-skill-backfill
    kind: conflicts-with
completed_at: 2026-07-17T21:24:59Z
---

# PRD Editor + Comms Backing — Pitch Author, Stakeholder Communicator

## Goal

Two shipped hero-code PRD Editor buttons draw with nothing behind them, and two
shipped commands route to placeholders. This child authors the two agents and
two skills that back them, and repoints the commands. When done: the PRD Editor
"Convert to pitch" action resolves to a real `pitch-author` agent; "Summarize for
standup" resolves to a real `stakeholder-communicator` agent; `/pitch` and
`/release-notes` route to those real agents instead of v1.5 placeholders; and two
net-new commands — `/standup` and `/interview` — exist and route to their backing
agents. All work is content-only under `domains/pm/` (agent/skill/command
markdown + `AGENTS.md` registration). No Go, no consumer-side code. `prd-author`
is preserved intact — it keeps owning PRD authoring; `pitch-author` is split out
as the dedicated pitch specialist that the `/pitch` surface now points to.

## Kickoff

Deliver `prd-editor-comms-backing` (initiative `pm-pack-completion`). Content-only,
`domains/pm/` pack source only — no Go. Author two agents (`pitch-author`,
`stakeholder-communicator`), two skills (`stakeholder-communication`,
`release-notes-writing`), two commands (`/standup`, `/interview`); repoint shipped
`/pitch` → `pitch-author` and `/release-notes` → `stakeholder-communicator`
(strip v1.5 deferral language); append four routes to the `AGENTS.md` Wave-2
region below the marker, after the #6/#7 blocks, and register the new agents/skills
in the Reference sections. Both new agents load `pm-agent-doctrine`. Keep
`prd-author.md` intact. Reference only skills/agents that exist on disk. Tripwire:
`harness-changes-cover-all-targets` — author in `domains/pm/` pack source only.

## Problem

The 2026-07 PM pack audit (`.hero/planning/features/hero-pm/pm-pack-audit-2026-07.md`,
§Surface coverage) flags the PRD Editor as `⚠️ partial`: `prd-author` ships, but
`stakeholder-communicator` and `pitch-author` are `❌` missing. hero-code already
draws the "Summarize for standup" and "Convert to pitch" buttons — they invoke
agents that don't exist.

Two shipped commands are backed weakly:

- `domains/pm/commands/pitch.md` routes to `prd-author` with the note
  "`pitch-author` takes over in v1.5".
- `domains/pm/commands/release-notes.md` routes to `pm-delivery-lead` with the
  note "stakeholder-communicator + skill ship v1.5".

Two commands the audit's Wave-1 list calls for — `/standup` and `/interview` —
have no command file at all (`domains/pm/commands/` has neither), and the
`AGENTS.md` canonical table records both as "no v1 surface (ships v1.5)".

Two skills the new agents need — `stakeholder-communication` and
`release-notes-writing` — are named in the locked design (`agent-pack-design.md`
§D.1) but were never authored (absent from `domains/pm/skills/`).

The design (`agent-pack-design.md` §C.3, §C.6) already specifies both agents; the
audit's external-scan reframe adds the exec "so what" pressure-test and
Amazon-6-pager / PR-FAQ awareness that `stakeholder-communication` should carry
(cross-referencing, not duplicating, child #9's `exec-narrative`).

## Acceptance Criteria

- **AC-1:** THE SYSTEM SHALL provide `domains/pm/agents/pitch-author.md` defining a Shape Up pitch specialist whose Startup load list includes `pm-agent-doctrine`, `pitch-writing-shape-up`, and `prd-structure`, and which refuses to ship a pitch with an empty Appetite or No-Gos.
- **AC-2:** THE SYSTEM SHALL provide `domains/pm/agents/stakeholder-communicator.md` defining an agent whose Startup load list includes `pm-agent-doctrine`, `outcomes-over-outputs`, `stakeholder-communication`, and `release-notes-writing`, and which produces audience-shaped cuts (exec / customer / internal) of PM artifacts.
- **AC-3:** THE SYSTEM SHALL provide `domains/pm/skills/stakeholder-communication/SKILL.md` covering audience-shaped messaging (exec / customer / eng / sales), the "so what" pressure-test, and Amazon-6-pager / PR-FAQ awareness that cross-references `exec-narrative` rather than duplicating it.
- **AC-4:** THE SYSTEM SHALL provide `domains/pm/skills/release-notes-writing/SKILL.md` covering both the customer-facing and internal release-note shapes.
- **AC-5:** WHEN a user runs `/standup` THE SYSTEM SHALL route to `stakeholder-communicator` to produce a standup update composed from intra-cycle graph changes.
- **AC-6:** WHEN a user runs `/interview` THE SYSTEM SHALL route to `discovery-researcher`, loading the `discovery-interview-design` skill, to design a customer interview guide.
- **AC-7:** WHEN a user runs `/pitch` THE SYSTEM SHALL route to `pitch-author`, and `domains/pm/commands/pitch.md` SHALL contain no "v1.5"/"takes over" deferral language.
- **AC-8:** WHEN a user runs `/release-notes` THE SYSTEM SHALL route to `stakeholder-communicator`, and `domains/pm/commands/release-notes.md` SHALL contain no "v1.5"/"ship v1.5" deferral language.
- **AC-9:** THE SYSTEM SHALL append the four new routes (standup, interview, pitch-author, stakeholder-communicator) to the `AGENTS.md` Wave-2 region below the `<!-- WAVE-2 ROUTES -->` marker and after the existing #6/#7 subsections, leaving the canonical routing table (rows above the marker) and the #6/#7 route blocks unchanged.
- **AC-10:** THE SYSTEM SHALL additively register the new surfaces in `AGENTS.md`: `stakeholder-communication` and `release-notes-writing` in the Skills Reference; `pitch-author` and `stakeholder-communicator` in the Agents Reference; `/standup` and `/interview` in the Commands Reference PM list.
- **AC-11:** THE SYSTEM SHALL leave `domains/pm/agents/prd-author.md` intact — its Startup load list and PRD templates (pitch-shaped and ten-section) are unchanged, so `prd-author` continues to own PRD authoring.
- **AC-12:** IF a new agent, skill, or command references a skill or agent id THEN THE SYSTEM SHALL reference only ids that exist on disk under `domains/pm/` (no dangling references).

## Changes

All paths are under `/Users/developer/projects/hero-engine/repository/hero/domains/pm/`.

1. **Create `agents/pitch-author.md`** — the Shape Up pitch specialist split out of
   `prd-author`. Backs the PRD Editor "Convert to pitch" action.
   - YAML frontmatter matching the shipped agent shape (see `prd-author.md`):
     `name: pitch-author`, `description:` (Shape Up pitch specialist; appetite as
     budget, rabbit holes as named traps, no-gos as scope defense), `mode:
     subagent`, `temperature: 0.1`, `color`, `permission` with `edit: allow`,
     `task: {"*": deny}`, `skill: {"*": allow}`, `webfetch: allow`.
   - "You are a senior Shape Up pitch author" persona opener.
   - **Startup load list** (only shipped ids): `pm-agent-doctrine`,
     `pitch-writing-shape-up`, `prd-structure`, `pm-preset-detection`,
     `acceptance-criteria-ears`, `spec-format`, `kickoff-prompt`. Do NOT reference
     `shape-up-cadence` (not shipped) — that would dangle.
   - "When invoked": `/pitch` slash, "shape this as a pitch" natural language, the
     cycle-preset "Convert to pitch" / "Write Pitch" contextual button on a PRD /
     initiative. Note it is the cycle-preset specialist; under a non-cycle preset it
     asks for a conscious override (mirror `pitch.md`'s pre-flight).
   - Enforces the five pitch sections (Problem / Appetite / Solution / Rabbit Holes /
     No-Gos) per `pitch-writing-shape-up`; **refuses to flip a pitch to review-ready
     with an empty Appetite or empty No-Gos**. Writes pitch specs to
     `.hero/planning/prds/<slug>/spec.md` (pitches are pitch-shaped PRDs), sets
     `prd_template: pitch`.
   - Persona/permission/anti-pattern/default-output shape mirrors `prd-author.md`;
     keep the doctrine posture (corpus-grounded, suggest-don't-decide,
     compare-don't-replace) from `pm-agent-doctrine`.

2. **Create `agents/stakeholder-communicator.md`** — translates PM artifacts into
   exec / customer / internal cuts. Backs the PRD Editor "Summarize for standup"
   action.
   - Frontmatter same shape as above; `name: stakeholder-communicator`.
   - "You are a stakeholder communicator" persona: same artifact, different cut per
     audience — executives want outcomes and tradeoffs, customers want capability and
     timing, engineers want context and AC, sales want talking points; shape the
     message to the audience **without distorting the truth**.
   - **Startup load list** (only shipped ids): `pm-agent-doctrine`,
     `outcomes-over-outputs`, `stakeholder-communication`, `release-notes-writing`,
     plus `cross-domain-graph-query` (for the `/standup` intra-cycle graph read) and
     `spec-format`, `kickoff-prompt`. All confirmed present under `domains/pm/skills/`
     and core.
   - "When invoked": `/standup`, `/release-notes`, presentation-mode toggle on the
     Roadmap board, "summarize for the leadership review" / shipped-story
     announcements, the "Summarize for standup" contextual button.
   - Output discipline: writes to `.hero/knowledge/notes/` or the release-note
     artifact paths that `release-notes.md` already names
     (`.hero/planning/release-notes/<window>/customer.md` / `internal.md`); honors
     doctrine 1 (no fabricated quotes/metrics) and doctrine 3 (compare-don't-replace)
     when summarizing.

3. **Create `skills/stakeholder-communication/SKILL.md`** — audience-shaped
   messaging skill.
   - Frontmatter (`name`, `description`, `metadata.audience:
     stakeholder-communicator`, `metadata.purpose`) matching the shipped skill shape
     (see `pitch-writing-shape-up/SKILL.md`, `pm-agent-doctrine/SKILL.md`).
   - Body sections: **What I do / When to use me / The four audience cuts** (exec,
     customer, eng, sales — each: what they want, what to lead with, what to omit) /
     **The "so what" pressure-test** (every exec-facing line must survive "so what?"
     — tie the statement to an outcome or cut it) / **Working-backwards awareness**
     (Amazon 6-pager / PR-FAQ) that **cross-references `exec-narrative`** (child #9's
     skill) as the home for the full PR-FAQ / narrative mechanics — this skill names
     the pattern and when to reach for it, it does NOT duplicate the format /
     **Anti-patterns** (one-cut-for-all messages; sandbagging timing for one audience
     while quoting an earlier date to another; marketing-flavor everything).
   - Cross-references: `outcomes-over-outputs`, `release-notes-writing`,
     `pm-agent-doctrine`, and `exec-narrative` (note it is authored by child #9; keep
     the cross-ref even though the target may not exist on disk yet — it is a
     forward-reference in prose, not a load-bearing skill-id load).

4. **Create `skills/release-notes-writing/SKILL.md`** — customer + internal
   release-note shapes.
   - Same shipped skill frontmatter shape; `metadata.audience:
     stakeholder-communicator, pm-delivery-lead`.
   - Body: **What I do / When to use me / Customer-facing shape** (lead with the user
     benefit not the feature name; call out behavior changes affecting existing
     workflows; group by theme not by spec; link to docs) / **Internal update shape**
     (spec slugs, owners, links back to originating PRDs/initiatives for traceability)
     / **Anti-patterns** (changelog dumps with no narrative; release notes that read
     like internal commit messages; marketing-flavor everything).
   - Content should track `agent-pack-design.md` §D.1 `release-notes-writing`.

5. **Create `commands/standup.md`** — `/standup` → `stakeholder-communicator`.
   - Thin router, matching the shipped command shape (see `metrics.md`,
     `release-notes.md`): frontmatter `description:`; body routes to
     `stakeholder-communicator`, loading `stakeholder-communication` +
     `cross-domain-graph-query`.
   - Behavior: compose a standup update from **intra-cycle graph changes** — what
     moved since the last standup (specs advanced, handoffs, hill-chart movement,
     blockers hit) read from the cross-domain graph (`hero feed` / graph events), not
     from a hand-maintained list. Audience is the internal team cut. End with
     `Request: $ARGUMENTS`.

6. **Create `commands/interview.md`** — `/interview` → `discovery-researcher`.
   - Thin router matching shipped shape; routes to `discovery-researcher`, loading
     the shipped `discovery-interview-design` skill (present under
     `domains/pm/skills/`). Purpose: design a customer interview guide (open
     questions about specific past experiences, avoid-leading-the-witness framing,
     sample size, synthesis plan). End with `Request: $ARGUMENTS`.

7. **Repoint `commands/pitch.md`** — change the routing line from `prd-author` (with
   "`pitch-author` takes over in v1.5") to route to **`pitch-author`**. Remove the
   v1.5 deferral parenthetical. Keep the rest (pre-flight preset check, pitch shape,
   enforcement, output) unchanged — `pitch-author` honors the same enforcement.

8. **Repoint `commands/release-notes.md`** — change the routing line from
   `pm-delivery-lead` (with "stakeholder-communicator + skill ship v1.5") to route to
   **`stakeholder-communicator`**. Remove the v1.5 deferral language. Keep the Scope /
   shipped-status / Output sections unchanged.

9. **Append routes to `AGENTS.md` Wave-2 region** — below the `<!-- WAVE-2 ROUTES -->`
   marker, **after** the existing "Wave-2 adversarial critic routes" (#6) and
   "Wave-2 experiment & metrics routes" (#7) subsections. Add a new subsection, e.g.
   `#### Wave-2 PRD Editor & comms routes`, with four rows following the established
   three-column format (User intent | Vocabulary-variant phrasing | Command (shipped
   surface)):
   - `/standup` → `stakeholder-communicator` (standup update from intra-cycle graph
     changes) — note it supersedes the canonical "no v1 surface (ships v1.5)"
     annotation, per the pattern the #7 `/metrics` row uses.
   - `/interview` → `discovery-researcher` (design a customer interview guide) —
     supersedes the canonical "`/discover --interview` … no v1 surface" annotation.
   - "Convert this to a pitch", "shape as a pitch" → `/pitch` → `pitch-author` (now a
     real backing agent; was `prd-author`).
   - "Summarize for standup", "cut this for exec / customer" → `stakeholder-communicator`
     (backs PRD Editor "Summarize for standup"; `/release-notes` also resolves here).
   - **Do NOT edit the canonical table above the marker, nor the #6/#7 subsections.**
     Additions-only, below and after.

10. **Register in `AGENTS.md` Reference sections** (additive edits, not
    reflows):
    - **Skills Reference:** add `stakeholder-communication` and
      `release-notes-writing` (Writing group is the natural home).
    - **Agents Reference:** add a bullet for the two new agents (e.g. "PM Wave-2 PRD
      Editor & comms: `pitch-author` (Shape Up pitch specialist, split from
      `prd-author`; backs Convert-to-pitch), `stakeholder-communicator` (audience-shaped
      cuts; backs Summarize-for-standup and `/release-notes`)").
    - **Commands Reference:** add `/standup` and `/interview` to the PM commands
      bullet.

11. **Leave `agents/prd-author.md` untouched.** It is read-only context for this
    child — it keeps owning PRD authoring (both templates). The split is additive:
    `pitch-author` becomes the dedicated pitch specialist the `/pitch` surface points
    to; `prd-author`'s latent pitch capability is not removed.

## Validation

Run from the repo root (`/Users/developer/projects/hero-engine/repository/hero`):

```bash
set -e
cd /Users/developer/projects/hero-engine/repository/hero
PM=domains/pm

# AC-1..4: new files exist
for f in \
  $PM/agents/pitch-author.md \
  $PM/agents/stakeholder-communicator.md \
  $PM/skills/stakeholder-communication/SKILL.md \
  $PM/skills/release-notes-writing/SKILL.md \
  $PM/commands/standup.md \
  $PM/commands/interview.md ; do
  test -f "$f" || { echo "MISSING: $f"; exit 1; }
done

# AC-1: pitch-author loads the three required skills
for s in pm-agent-doctrine pitch-writing-shape-up prd-structure ; do
  grep -q "$s" $PM/agents/pitch-author.md || { echo "pitch-author missing load: $s"; exit 1; }
done

# AC-2: stakeholder-communicator loads the four required skills
for s in pm-agent-doctrine outcomes-over-outputs stakeholder-communication release-notes-writing ; do
  grep -q "$s" $PM/agents/stakeholder-communicator.md || { echo "stakeholder-communicator missing load: $s"; exit 1; }
done

# AC-3: stakeholder-communication skill covers the audiences, so-what test, PR-FAQ cross-ref
grep -qi "so what" $PM/skills/stakeholder-communication/SKILL.md || { echo "so-what test missing"; exit 1; }
grep -qiE "PR-FAQ|6-pager|working.backwards" $PM/skills/stakeholder-communication/SKILL.md || { echo "PR-FAQ awareness missing"; exit 1; }
grep -q "exec-narrative" $PM/skills/stakeholder-communication/SKILL.md || { echo "exec-narrative cross-ref missing"; exit 1; }

# AC-4: release-notes-writing covers customer + internal shapes
grep -qi "customer" $PM/skills/release-notes-writing/SKILL.md || { echo "customer shape missing"; exit 1; }
grep -qi "internal" $PM/skills/release-notes-writing/SKILL.md || { echo "internal shape missing"; exit 1; }

# AC-5,6: new commands route to the right agents / skill
grep -q "stakeholder-communicator" $PM/commands/standup.md || { echo "/standup routing wrong"; exit 1; }
grep -q "discovery-researcher" $PM/commands/interview.md || { echo "/interview routing wrong"; exit 1; }
grep -q "discovery-interview-design" $PM/commands/interview.md || { echo "/interview skill missing"; exit 1; }

# AC-7: /pitch repointed, no deferral language
grep -q "pitch-author" $PM/commands/pitch.md || { echo "/pitch not repointed"; exit 1; }
grep -qi "v1.5" $PM/commands/pitch.md && { echo "/pitch still defers to v1.5"; exit 1; }

# AC-8: /release-notes repointed, no deferral language
grep -q "stakeholder-communicator" $PM/commands/release-notes.md || { echo "/release-notes not repointed"; exit 1; }
grep -qi "v1.5" $PM/commands/release-notes.md && { echo "/release-notes still defers to v1.5"; exit 1; }

# AC-9: Wave-2 additions present, appended AFTER #6/#7 subsections
grep -q "Wave-2 PRD Editor & comms routes" $PM/AGENTS.md || { echo "new Wave-2 subsection missing"; exit 1; }
awk '/WAVE-2 ROUTES/{m=1} m && /Wave-2 experiment & metrics routes/{e=NR} m && /Wave-2 PRD Editor & comms routes/{p=NR} END{ if(!(e>0 && p>e)) exit 1 }' $PM/AGENTS.md || { echo "new subsection not after #7"; exit 1; }

# AC-9: canonical table + #6/#7 route blocks unchanged (diff only touches lines below/after)
git diff -- $PM/AGENTS.md | grep -E '^\+' | grep -qiE "Wave-2 adversarial critic routes|prioritization-challenger|experiment-designer" \
  && { echo "edited a #6/#7 line — must be additions-only below"; exit 1; } || true

# AC-10: Reference-section registrations
grep -q "stakeholder-communication" $PM/AGENTS.md || { echo "skill not registered"; exit 1; }
grep -q "release-notes-writing" $PM/AGENTS.md || { echo "skill not registered"; exit 1; }
grep -q "pitch-author" $PM/AGENTS.md || { echo "agent not registered"; exit 1; }
grep -q "stakeholder-communicator" $PM/AGENTS.md || { echo "agent not registered"; exit 1; }

# AC-11: prd-author intact (unchanged)
git diff --quiet -- $PM/agents/prd-author.md || { echo "prd-author.md was modified — must stay intact"; exit 1; }

# AC-12: no dangling skill/agent references from the new agents
for s in $(grep -oE '\b[a-z][a-z0-9-]+\b' $PM/agents/pitch-author.md $PM/agents/stakeholder-communicator.md \
           | sort -u | grep -E 'shape-up-cadence|prfaq-writing|market-sizing|okr-design'); do
  echo "SUSPECT unshipped skill referenced: $s"; exit 1;
done

echo "ALL CHECKS PASSED"
```

Manual verification:

- Read `pitch-author.md` and `stakeholder-communicator.md`: confirm the persona,
  when-invoked triggers, and doctrine posture read like the shipped agents
  (`prd-author.md` is the template), and that each references only ids present under
  `domains/pm/skills/`.
- Read the new `AGENTS.md` Wave-2 subsection: confirm it sits after the experiment &
  metrics subsection, uses the three-column format, and notes the supersession of the
  canonical "no v1 surface" annotations (mirroring the #7 `/metrics` row style).
- Confirm `stakeholder-communication` names `exec-narrative` as the PR-FAQ home
  rather than reproducing the format.

## Boundaries

- **No Go, no engine code.** Content-only under `domains/pm/`. Tripwire
  `harness-changes-cover-all-targets` applies — author in pack source only; do not
  hand-edit any installed `.claude/` / `.codex/` copies.
- **No consumer-side (hero-code) changes.** hero-code already draws the PRD Editor
  buttons; wiring the manifest is a separate consumer-side concern (Wave-0
  `PMManifest.swift` repoint), out of scope here.
- **Do not edit the `AGENTS.md` canonical routing table** (rows above the
  `<!-- WAVE-2 ROUTES -->` marker) or the #6/#7 Wave-2 route subsections. This child
  is additions-only, below and after them.
- **Do not modify `prd-author.md`.** The split is additive; removing prd-author's PRD
  authoring is explicitly out of scope.
- **Do not author `exec-narrative`** or any other child #9 skill — cross-reference it,
  don't create it. Do not author Wave-1 siblings' agents/skills (`dependency-mapper`,
  `capacity-planner`, `cycle-planner`, `duplicate-intake-scrubber`, etc.).
- **No new skill ids beyond the two named.** `stakeholder-communication` and
  `release-notes-writing` only; do not invent `prfaq-writing` / `market-sizing` /
  `shape-up-cadence` here.
- **`/standup` and `/interview` are the only new commands.** `/capacity`,
  `/plan-{cycle,sprint,iteration}`, `/scrub` remain deferred to their owning children.

## Risks

- **AGENTS.md contradiction.** The canonical table still records `/standup` and
  `/interview` as "no v1 surface (ships v1.5)". Per the additions-only constraint we
  do not edit those rows; the appended Wave-2 subsection must explicitly note it
  supersedes them (the established pattern the #7 `/metrics` row uses) so the doc
  reads coherently rather than self-contradictory.
- **Dangling-ref temptation.** The design (`agent-pack-design.md` §C.3) lists
  `shape-up-cadence` for `pitch-author`, but that skill is not shipped. Referencing it
  would dangle. Stick to the shipped load list. Same caution for `exec-narrative`
  (child #9) — allowed only as a prose cross-reference, never a load-bearing skill
  load, until #9 ships.
- **Overlap seam with `pm-doctrine-and-skill-backfill` (#1).** Reciprocal
  `conflicts-with` is declared because both children touch the `AGENTS.md` Wave-2
  region / Reference sections. Deliver serially, not concurrently, to avoid a
  same-file collision; #1 is also a `depends-on` (it authored `pm-agent-doctrine` and
  `outcomes-over-outputs`, which both new agents load).

## Completion Ledger

### Acceptance Criteria

| # | Criterion | Status | Note |
|---|---|---|---|
| AC-1 | `pitch-author.md` Shape Up specialist; Startup loads `pm-agent-doctrine`, `pitch-writing-shape-up`, `prd-structure`; refuses empty Appetite/No-Gos | DONE | `domains/pm/agents/pitch-author.md` — Startup list has all three (plus `pm-preset-detection`, `acceptance-criteria-ears`, `spec-format`, `kickoff-prompt`); §"The enforcement gate" refuses `review` with empty Appetite or No-Gos. Validation loop confirms the three loads. |
| AC-2 | `stakeholder-communicator.md` Startup loads `pm-agent-doctrine`, `outcomes-over-outputs`, `stakeholder-communication`, `release-notes-writing`; produces exec/customer/internal cuts | DONE | `domains/pm/agents/stakeholder-communicator.md` — Startup list has all four (plus `cross-domain-graph-query`, `spec-format`, `kickoff-prompt`); §"Cut for the audience" produces exec/customer/internal cuts. Validation confirms the four loads. |
| AC-3 | `stakeholder-communication/SKILL.md` covers exec/customer/eng/sales, "so what" test, PR-FAQ/6-pager cross-ref to `exec-narrative` (not duplicated) | DONE | `domains/pm/skills/stakeholder-communication/SKILL.md` — four-audience table, §"The 'so what' pressure-test", §"Working-backwards awareness" names PR-FAQ/6-pager and defers the format to `exec-narrative`. grep gates for `so what`, `PR-FAQ\|6-pager`, `exec-narrative` all pass. |
| AC-4 | `release-notes-writing/SKILL.md` covers customer-facing + internal shapes | DONE | `domains/pm/skills/release-notes-writing/SKILL.md` — §"Customer-facing shape" and §"Internal update shape". grep gates for `customer`/`internal` pass. |
| AC-5 | `/standup` routes to `stakeholder-communicator`, standup from intra-cycle graph changes | DONE | `domains/pm/commands/standup.md` — routes to `stakeholder-communicator` loading `stakeholder-communication` + `cross-domain-graph-query`; §"What lands" composes from graph changes. grep gate passes. |
| AC-6 | `/interview` routes to `discovery-researcher`, loads `discovery-interview-design` | DONE | `domains/pm/commands/interview.md` — routes to `discovery-researcher` loading `discovery-interview-design`; designs a customer interview guide. Both grep gates pass. |
| AC-7 | `/pitch` routes to `pitch-author`, no v1.5/"takes over" deferral | DONE | `domains/pm/commands/pitch.md` — routing line now `Route to \`pitch-author\``; the v1.5 parenthetical removed. grep for `pitch-author` passes, grep for `v1.5` finds nothing. |
| AC-8 | `/release-notes` routes to `stakeholder-communicator`, no v1.5 deferral | DONE | `domains/pm/commands/release-notes.md` — routing line now `Route to \`stakeholder-communicator\``; "ship v1.5" removed; Scope/shipped-status/Output unchanged. grep for `stakeholder-communicator` passes, grep for `v1.5` finds nothing. |
| AC-9 | Four routes appended to Wave-2 region below the marker, after #6/#7; canonical table + #6/#7 blocks unchanged | DONE | `domains/pm/AGENTS.md` — new "#### Wave-2 PRD Editor & comms routes" subsection sits after the experiment & metrics (#7) subsection with four rows. awk order gate passes; diff of the canonical table and #6/#7 blocks is empty (removed-lines check shows only the two additive Reference bullets). |
| AC-10 | Register new surfaces in `AGENTS.md`: 2 skills (Skills Ref), 2 agents (Agents Ref), `/standup`+`/interview` (Commands Ref) | DONE | `domains/pm/AGENTS.md` — Writing skills line adds `stakeholder-communication`, `release-notes-writing`; new "PM Wave-2 PRD Editor & comms" agents bullet adds both agents; PM commands bullet adds `/interview` and `/standup`. All four grep registration gates pass. |
| AC-11 | `prd-author.md` left intact — Startup list and templates unchanged | DONE | `git diff --quiet -- domains/pm/agents/prd-author.md` returns clean (UNCHANGED). No edit was made to the file. |
| AC-12 | New agents/skills/commands reference only on-disk ids (no dangling refs) | DONE | All referenced skill ids (`pm-agent-doctrine`, `pitch-writing-shape-up`, `prd-structure`, `pm-preset-detection`, `acceptance-criteria-ears`, `outcomes-over-outputs`, `stakeholder-communication`, `release-notes-writing`, `cross-domain-graph-query`, `discovery-interview-design`) confirmed present via `ls domains/pm/skills/`; `spec-format`/`kickoff-prompt` are core. `discovery-researcher` agent present. AC-12 suspect-id grep (`shape-up-cadence`/`prfaq-writing`/`market-sizing`/`okr-design`) finds none. `exec-narrative` appears only as a prose cross-ref, never a load. |

### Changes

| # | Changes item | Status | Note |
|---|---|---|---|
| 1 | Create `agents/pitch-author.md` | DONE | Shape Up pitch specialist; frontmatter matches shipped shape (`mode: subagent`, `temperature: 0.1`, permission block); Startup load list of shipped ids only; enforcement gate refuses empty Appetite/No-Gos; writes to `.hero/planning/prds/<slug>/spec.md` with `prd_template: pitch`. |
| 2 | Create `agents/stakeholder-communicator.md` | DONE | Audience-cut agent; same frontmatter shape; Startup loads the four required + `cross-domain-graph-query`/`spec-format`/`kickoff-prompt`; exec/customer/internal cuts; doctrine 1 + 3 posture. |
| 3 | Create `skills/stakeholder-communication/SKILL.md` | DONE | Frontmatter `metadata.audience: stakeholder-communicator`; four-audience cuts, "so what" pressure-test, working-backwards awareness cross-referencing `exec-narrative` (not duplicated), anti-patterns, cross-refs. |
| 4 | Create `skills/release-notes-writing/SKILL.md` | DONE | Frontmatter `metadata.audience: stakeholder-communicator, pm-delivery-lead`; customer-facing + internal shapes, shipped-status-from-graph, anti-patterns, cross-refs. |
| 5 | Create `commands/standup.md` | DONE | Thin router → `stakeholder-communicator` loading `stakeholder-communication` + `cross-domain-graph-query`; intra-cycle graph changes; ends `Request: $ARGUMENTS`. |
| 6 | Create `commands/interview.md` | DONE | Thin router → `discovery-researcher` loading `discovery-interview-design`; designs a customer interview guide; ends `Request: $ARGUMENTS`. |
| 7 | Repoint `commands/pitch.md` → `pitch-author` | DONE | Routing line changed to `Route to \`pitch-author\``; v1.5 parenthetical removed; pre-flight / shape / enforcement / output unchanged. |
| 8 | Repoint `commands/release-notes.md` → `stakeholder-communicator` | DONE | Routing line changed to `Route to \`stakeholder-communicator\`, loading \`release-notes-writing\``; "ship v1.5" removed; Scope/shipped-status/Output unchanged. |
| 9 | Append routes to `AGENTS.md` Wave-2 region | DONE | New "#### Wave-2 PRD Editor & comms routes" subsection with the four rows, after #6/#7, below the marker; notes supersession of the canonical "no v1 surface" annotations per the #7 `/metrics` pattern. Additions-only. |
| 10 | Register in `AGENTS.md` Reference sections | DONE | Skills Reference (Writing), Agents Reference (new PM Wave-2 bullet), Commands Reference (PM list) all updated additively. |
| 11 | Leave `agents/prd-author.md` untouched | DONE | No edit made; `git diff --quiet` clean. |

### Exercise-the-feature check

- [x] Full Validation bash block run verbatim from repo root → `ALL CHECKS PASSED`.
- [x] `/pitch` resolves: `commands/pitch.md` routes to `pitch-author`, which exists on disk with a matching enforcement gate (AC-7).
- [x] `/release-notes` resolves: `commands/release-notes.md` routes to `stakeholder-communicator`, which exists on disk and loads `release-notes-writing` (AC-8).
- [x] `/standup` and `/interview` resolve to on-disk agents (`stakeholder-communicator`, `discovery-researcher`) with the named skills present (AC-5, AC-6).
- [x] AGENTS.md additions-only confirmed: removed-lines check shows only the two additive Reference bullets; canonical table and #6/#7 blocks untouched (AC-9).
- [x] No-dangling-refs confirmed: every referenced skill/agent id present under `domains/pm/`; suspect-id grep empty (AC-12).
- [ ] Runtime agent invocation not exercised — these are harness markdown definitions consumed by the installed harness at agent-dispatch time; there is no compile/run surface in this repo to dispatch them. Verification is by on-disk shape-match against the shipped `prd-author.md` / `discovery-researcher.md` templates and the Validation gates, which is the appropriate check for content-only pack source.

### Excellence Bar self-check

- Both new agents match the shipped agent shape (frontmatter block, persona opener, Startup load list, When-invoked triggers, Workflow, doctrine posture, Anti-patterns, Default output) modeled on `prd-author.md` and `discovery-researcher.md`.
- Every agent stance is corpus-grounded and decision-gated per `pm-agent-doctrine` (explicit doctrine sections in both agents; "no fabricated quotes/metrics", "suggest-don't-decide", "compare-don't-replace").
- Skills are full-length (no stubs), each with What-I-do / When-to-use / body sections / Anti-patterns / Cross-references, matching `pitch-writing-shape-up` / `pm-agent-doctrine` depth.
- Dangling-ref discipline honored: `shape-up-cadence` deliberately omitted from `pitch-author`'s load list (unshipped); `exec-narrative` kept as a prose forward-reference only.
- The AGENTS.md contradiction risk (canonical "no v1 surface" rows) is handled by an explicit supersession note in the new subsection rather than editing protected rows, so the doc reads coherently.
