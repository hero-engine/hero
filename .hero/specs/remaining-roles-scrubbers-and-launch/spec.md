---
title: "Remaining Roles, Scrubbers, and Launch/GTM"
slug: remaining-roles-scrubbers-and-launch
type: feature
status: completed
domain: pm
priority: low
size: medium
created: 2026-07-17
tags: [pm, roles, scrub, launch, coverage, wave-3]
relations:
  - target: pm-pack-completion
    kind: parent
  - target: pm-doctrine-and-skill-backfill
    kind: depends-on
  - target: story-detail-and-intake-scrubber-backing
    kind: depends-on
  - target: story-detail-and-intake-scrubber-backing
    kind: conflicts-with
completed_at: 2026-07-17T23:43:33Z
---

# Remaining Roles, Scrubbers, and Launch/GTM

## Goal

Round out the remaining designed P1/P2 PM roles, both remaining PM
scrubber concerns, and launch/GTM coverage — all as **pack source under
`domains/pm/` only**, no Go. "Done" is: six new agent files exist and
each loads `pm-agent-doctrine` plus its designed skills (every skill
referenced resolves to a real directory on disk — zero dangling); the
shared `/scrub` command dispatches two additional concerns (`roadmap`,
`stories`) appended in child #5's marked append-only region while #5's
`intake` block stays byte-unchanged; a `launch-gtm-tiering` skill and a
`/launch` command ship the tier-1/2/3 + phased-checklist doctrine; and
`domains/pm/AGENTS.md` routes the new agents/commands in the Wave-2
region below the prior children's routes, with the canonical table and
every prior child's routes byte-unchanged. The three reference rosters
(Agents / Skills / Commands) list the new artifacts.

## Kickoff

Paste-ready cold-start prompt for a fresh delivery session:

> Deliver spec `remaining-roles-scrubbers-and-launch` (child #11 of the
> `pm-pack-completion` initiative). Read
> `.hero/planning/initiatives/pm-pack-completion/remaining-roles-scrubbers-and-launch/spec.md`.
> This is **content-only, `domains/pm/` pack source, no Go**. Author six
> PM agent files, extend `domains/pm/commands/scrub.md` with two new
> concerns, add one skill + one command, and append routes/rosters to
> `domains/pm/AGENTS.md`.
>
> Hard constraints, verify each before you finish:
> 1. Every skill an agent loads must resolve to a real directory under
>    `domains/pm/skills/` or `core/skills/` — run the no-dangling check in
>    `## Validation`. Zero dangling refs.
> 2. Do **not** edit child #5's `intake` concern block in `scrub.md`.
>    Append `roadmap` + `stories` concerns only in the marked `#11 APPEND
>    POINT` region. The Validation block guards this byte-for-byte.
> 3. Do **not** edit `AGENTS.md`'s canonical routing table or any prior
>    child's Wave-2 routes. Append a new child-#11 subsection below the
>    last existing Wave-2 subsection only.
> 4. Tripwire `harness-changes-cover-all-targets`: author only in
>    `domains/pm/` pack source — do not hand-edit any installed
>    `.claude/` / `.agents/` / `.codex/` mirror.
>
> Model the new files on the existing pm pack: agents on
> `domains/pm/agents/duplicate-intake-scrubber.md` (scrubbers) and
> `domains/pm/agents/story-writer.md` (authoring); command on
> `domains/pm/commands/triage.md`; concern block on the existing `intake`
> block in `scrub.md`.

## Problem

The `pm-pack-completion` initiative closes the gap between the PM pack's
**design** (`.hero/planning/features/hero-pm/agent-pack-design.md` §C and
the `pm-pack-audit-2026-07.md` Wave-3 row) and what actually ships on
disk. Prior waves landed doctrine, authoring, critics, experiment/metrics,
comms, competitive/market, and exec-narrative backing. This child is the
**final Wave-3 rounding pass** and covers the last designed roles plus
launch/GTM, which the audit lists but no wave has homed:

- **Designed roles still unbacked.** §C names `epic-framer` (P1),
  `risk-curator` (P1), `portfolio-curator` (P2), `discovery-reviewer`
  (P2), and two scrubbers — `stale-roadmap-scrubber` (P1) and
  `ambiguous-story-scrubber` (P2). None exist as agent files, so the
  routes that should reach them dangle.
- **`/scrub` ships only one of three concerns.** Child #5 scaffolded
  `domains/pm/commands/scrub.md` with the `intake` concern and left a
  marked append region for exactly the `roadmap` and `stories` concerns
  this child owns. Until they land, `/scrub roadmap` and `/scrub stories`
  have no dispatch and the two scrubber agents have no command entry
  point.
- **Launch/GTM coverage is entirely absent.** The audit's Wave-3 row
  "Launch / GTM tiering → skill + `/launch` (tier 1/2/3, phased
  checklist)" has no skill and no command; the pack cannot help a PM plan
  a launch.

The seam risk is the shared file `scrub.md` (and, secondarily,
`AGENTS.md`): child #5 and this child both write them. The `depends-on`
edge orders #5 first; the reciprocal `conflicts-with` guards against a
concurrent edit if the ordering slips. This spec resolves the seam
structurally by confining all of this child's `scrub.md` edits to #5's
explicit append-only region and never touching #5's `intake` block.

## Acceptance Criteria

- **AC-1:** THE SYSTEM SHALL ship six new agent files under
  `domains/pm/agents/` — `epic-framer.md`, `risk-curator.md`,
  `portfolio-curator.md`, `discovery-reviewer.md`,
  `stale-roadmap-scrubber.md`, `ambiguous-story-scrubber.md` — each with
  a valid frontmatter block (`name`, `description`, `mode: subagent`,
  `permission`).
- **AC-2:** THE SYSTEM SHALL have every agent load `pm-agent-doctrine`
  plus its designed skills, and every skill named by any of the six
  agents SHALL resolve to a real directory under `domains/pm/skills/` or
  `core/skills/` (zero dangling refs — the `## Validation` no-dangling
  check passes).
- **AC-3:** WHERE the agent is `stale-roadmap-scrubber` or
  `ambiguous-story-scrubber`, THE SYSTEM SHALL mark it report-only
  (`permission.edit: deny`), matching the existing
  `duplicate-intake-scrubber` contract.
- **AC-4:** THE SYSTEM SHALL append a `### Concern: roadmap` block
  (dispatch → `stale-roadmap-scrubber`, invoked by `/scrub roadmap`) and
  a `### Concern: stories` block (dispatch → `ambiguous-story-scrubber`,
  invoked by `/scrub stories`) to `domains/pm/commands/scrub.md`, placed
  **below** the `#11 APPEND POINT` marker line.
- **AC-5:** THE SYSTEM SHALL leave child #5's `intake` concern block in
  `scrub.md` (the region from `### Concern: intake` through the line
  immediately above the `#11 APPEND POINT` marker) **byte-for-byte
  unchanged**; the `## Validation` guard confirms it.
- **AC-6:** THE SYSTEM SHALL ship
  `domains/pm/skills/launch-gtm-tiering/SKILL.md` defining tier 1/2/3 by
  launch impact and the five-phase checklist (alignment → positioning →
  enablement → launch → post-launch).
- **AC-7:** THE SYSTEM SHALL ship `domains/pm/commands/launch.md` — the
  `/launch` command that produces a launch plan + checklist, routes to
  `stakeholder-communicator` (or a sensible owner), loads
  `launch-gtm-tiering`, and notes the route.
- **AC-8:** THE SYSTEM SHALL append a new child-#11 subsection to the
  `domains/pm/AGENTS.md` Wave-2 region **below** the last existing Wave-2
  subsection, routing all six agents plus `/scrub roadmap`, `/scrub
  stories`, and `/launch`; the canonical routing table and every prior
  child's Wave-2 routes SHALL remain byte-for-byte unchanged.
- **AC-9:** THE SYSTEM SHALL add the six agents to the AGENTS.md **Agents
  Reference**, `launch-gtm-tiering` to the **Skills Reference**, and
  `/launch` plus the `roadmap`/`stories` scrub concerns to the
  **Commands Reference** roster.
- **AC-10:** THE SYSTEM SHALL confine all edits to `domains/pm/` pack
  source — no installed `.claude/` / `.agents/` / `.codex/` mirror is
  hand-edited (tripwire `harness-changes-cover-all-targets`).

## Changes

All paths are under `domains/pm/`. Model each new file on the cited
existing file so tone, frontmatter shape, and section discipline match.

1. **`domains/pm/agents/epic-framer.md`** (authoring;
   `permission.edit: allow`, model on `story-writer.md`).
   - Role: an epic is a **coherent bet**, not a bag of unrelated
     stories. Writes the *Why* and the **rollup acceptance criteria**;
     **sequences** child stories and surfaces dependencies. Reconciles
     child-story rollup state.
   - When invoked: `/refine` on an epic; "group these stories into an
     epic"; intake promotion exceeding one-story scope. Delegates child
     stories to `story-writer`.
   - Startup loads: `pm-agent-doctrine`, `epic-framing`,
     `story-writing-invest`, `dependency-mapping`.
   - Anti-patterns: an epic with no rollup AC; an epic that is just a
     tag over unrelated stories; sequencing with no dependency surface.

2. **`domains/pm/agents/risk-curator.md`** (authoring;
   `permission.edit: allow`).
   - Role: surface and shape risks on PRDs, roadmap-items, and stories.
     Each risk is stated as **scenario + indicator + response** — the
     specific scenario that triggers it, the leading indicator that
     it's materializing, and the response — never generic "might not
     scale" wording. Distinguishes risks worth testing now from risks
     worth deferring.
   - When invoked: PRD risk-section authoring; pre-handoff review; "what
     could go wrong". Delegates assumption tests to
     `discovery-researcher`.
   - Startup loads: `pm-agent-doctrine`, `risk-surfacing`,
     `assumption-testing`, `evidence-synthesis`.
   - Anti-patterns: generic risk boilerplate with no scenario; a risk
     with no indicator (undetectable); recommending an assumption test
     with no hypothesis.

3. **`domains/pm/agents/portfolio-curator.md`** (curation;
   `permission.edit: allow` — writes portfolio summaries as notes and
   rebalance recommendations, human-gated).
   - Role: **cross-roadmap theme balance** and
     **capacity-vs-ambition** reconciliation — "are we over-investing in
     X area", is the portfolio outcome-weighted or output-weighted.
     Produces portfolio summaries (notes) and rebalance recommendations
     on roadmap-items; recommends, never auto-rebalances.
   - When invoked: quarterly roadmap reviews; "how is our portfolio
     balanced". Delegates to `metrics-analyst`,
     `prioritization-strategist`.
   - Startup loads: `pm-agent-doctrine`, `outcomes-over-outputs`,
     `roadmap-framing`, `prioritization-frameworks`. (Design §C.2 named
     `capacity-planning`; this child pins the three real, on-disk skills
     that carry the outcome-weighting + framing + prioritization
     reasoning — no dangling ref.)
   - Anti-patterns: a rebalance with no capacity signal; theme balance
     asserted without the outcome/output tally; auto-applying a
     reprioritization.

4. **`domains/pm/agents/discovery-reviewer.md`** (review/critic;
   `permission.edit: deny` — surfaces findings + verdict, routes back to
   the authoring agent; consistent with the Wave-2 critic contract).
   - Role: **adversarial rigor review** of discovery artifacts —
     opportunity-solution trees, interview synthesis, and assumption
     tests. Checks the tree is opportunity-first (not solution-first),
     that synthesis compares-don't-replaces with verbatim traceability,
     and that assumption tests have a real hypothesis + stop rule.
   - When invoked: `/review` on discovery output (invoke the agent
     directly — §F ships no `/review` command in pm).
   - Startup loads: `pm-agent-doctrine`,
     `opportunity-solution-trees-torres`, `discovery-interview-design`,
     `assumption-testing`, `evidence-synthesis`.
   - Anti-patterns: accepting a solution-first tree; passing synthesis
     with no verbatim traceability; approving an assumption test with no
     falsifiable hypothesis.

5. **`domains/pm/agents/stale-roadmap-scrubber.md`** (scrubber;
   report-only `permission.edit: deny`, model on
   `duplicate-intake-scrubber.md`).
   - Role: find roadmap-items that **haven't moved in N weeks**, shipped
     items still marked active, and `later` items older than the
     planning horizon. Recommends action per item — archive, drop with
     reason, or refresh — **presented, never auto-applied**
     (decision-gate doctrine).
   - When invoked: `/scrub roadmap` (concern-dispatched via `scrub.md`);
     weekly cron.
   - Produces: a scrub report — flagged items, why each is stale
     (specific signal), recommended state change; explicit "no stale
     items found" when clean.
   - Startup loads: `pm-agent-doctrine`, `roadmap-framing`,
     `outcome-drift`.
   - Anti-patterns: auto-flipping roadmap state; flagging "stale" with
     no age/movement signal; trusting the tracker over live delivery
     state.

6. **`domains/pm/agents/ambiguous-story-scrubber.md`** (scrubber;
   report-only `permission.edit: deny`, model on
   `duplicate-intake-scrubber.md`).
   - Role: find stories at `status: ready` that **fail INVEST** or lack
     **EARS acceptance criteria** — they cause friction at handoff.
     Flags each with its **specific failure** (missing AC, too large /
     not Small, untestable, not Independent) and recommends refinement
     before the story is pulled.
   - When invoked: `/scrub stories` (concern-dispatched); pre-cycle
     planning.
   - Produces: a scrub report — flagged `ready` stories, the precise
     INVEST/EARS failure per story, recommended fix; explicit "no
     ambiguous stories found" when clean. Report-only.
   - Startup loads: `pm-agent-doctrine`, `story-writing-invest`,
     `acceptance-criteria-ears`.
   - Anti-patterns: auto-editing a story; a vague "needs work" flag with
     no named INVEST/EARS failure; flagging non-`ready` drafts (they're
     expected to be rough).

7. **Extend `domains/pm/commands/scrub.md`** — append **only** below the
   `#11 APPEND POINT` marker (line 28 in the current file), between it
   and the `## Dispatch` heading. Do **not** touch the `### Concern:
   intake` block or anything above the marker.
   - Add `### Concern: roadmap` → `/scrub roadmap` →
     `stale-roadmap-scrubber`. Body: sweep roadmap-items for staleness /
     mislabeling (no movement in N weeks, shipped-but-active,
     over-horizon `later`); emit the flag report with a recommended
     action per item; **report-only, no auto state flip**.
   - Add `### Concern: stories` → `/scrub stories` →
     `ambiguous-story-scrubber`. Body: sweep `ready` stories for
     INVEST/EARS failures; emit flagged stories with the specific
     failure and recommended refinement; **report-only, no auto edit**.
   - Update the frontmatter `description:` line's "ship via child #11"
     wording to past tense (roadmap + stories now shipped) — this is an
     in-region edit to the description, not the intake concern block.

8. **`domains/pm/skills/launch-gtm-tiering/SKILL.md`** — new skill.
   - **Tier 1/2/3 by impact:** Tier 1 = major / company-moving launch
     (full GTM motion, exec + field + comms alignment); Tier 2 =
     standard feature launch (positioning + enablement + announcement);
     Tier 3 = minor / incremental (release note + in-product surface).
     Give the decision rubric that assigns a launch to a tier (blast
     radius, revenue/segment impact, net-new vs. incremental,
     competitive stakes).
   - **Phased checklist:** `alignment → positioning → enablement →
     launch → post-launch`, each phase with its concrete artifacts /
     gates and which tiers require it (Tier 3 collapses phases; Tier 1
     runs them all). Include the corpus-grounding + decision-gate
     doctrine framing (recommendations, human-owned).

9. **`domains/pm/commands/launch.md`** — new `/launch` command (model on
   `triage.md`).
   - Produces a **launch plan + checklist** for the target
     roadmap-item / PRD: detects the tier via `launch-gtm-tiering`, then
     emits the phased checklist scoped to that tier.
   - Routes to `stakeholder-communicator` (owns the announcement / exec
     cut) as the default owner, or a sensible owner per phase; **note
     the route** in the output. Pass `$ARGUMENTS` (the launch target)
     through.

10. **Append routes to `domains/pm/AGENTS.md` Wave-2 region** — add a new
    subsection **below** the last existing Wave-2 subsection (currently
    "Wave-2 exec narrative & working-backwards routes", ending ~line
    163), titled e.g. `#### Wave-3 remaining roles, scrubbers & launch
    routes (remaining-roles-scrubbers-and-launch)`. One route row each:
    - `epic-framer` — `/refine` on an epic / "group into an epic"
      (authoring; loads `epic-framing`).
    - `risk-curator` — PRD risk section / "what could go wrong"
      (authoring; scenario+indicator+response).
    - `portfolio-curator` — "how is our portfolio balanced"
      (curation; cross-roadmap theme balance + capacity-vs-ambition).
    - `discovery-reviewer` — `/review` on discovery output (critic;
      invoke directly, no `/review` command ships in pm).
    - `/scrub roadmap` → `stale-roadmap-scrubber` (report-only).
    - `/scrub stories` → `ambiguous-story-scrubber` (report-only).
    - `/launch` → launch plan/checklist (routes to
      `stakeholder-communicator`; loads `launch-gtm-tiering`).
    - Do **not** edit the canonical table (lines above line 62) or any
      prior child's Wave-2 subsection.

11. **Update the three AGENTS.md reference rosters** (in place, additive):
    - **Agents Reference** (~line 253): add a `PM Wave-3` bullet listing
      the six new agents with one-line roles.
    - **Skills Reference** (~line 265): add `launch-gtm-tiering` (extend
      the Wave-3 discovery/framing bullet or add an Operational/Wave-3
      entry) with a one-line description.
    - **Commands Reference** (~line 245): add `/launch` to the PM
      command list and update the `/scrub` line so `roadmap` + `stories`
      read as shipped (not "land via child #11").

## Validation

Run from the repo root (`/Users/developer/projects/hero-engine/repository/hero`).
All checks must pass; no Go build/test is in scope.

```bash
set -euo pipefail
cd /Users/developer/projects/hero-engine/repository/hero

echo "== AC-1: six agent files exist =="
for a in epic-framer risk-curator portfolio-curator discovery-reviewer \
         stale-roadmap-scrubber ambiguous-story-scrubber; do
  test -f "domains/pm/agents/$a.md" || { echo "MISSING agent: $a"; exit 1; }
done
echo "ok"

echo "== AC-6 / AC-7: skill + command exist =="
test -f domains/pm/skills/launch-gtm-tiering/SKILL.md || { echo "MISSING launch skill"; exit 1; }
test -f domains/pm/commands/launch.md || { echo "MISSING /launch command"; exit 1; }
echo "ok"

echo "== AC-2/AC-5 GUARD: child #5 intake block byte-unchanged =="
# The intake block is the region from '### Concern: `intake`' through the line
# immediately preceding the '#11 APPEND POINT' marker. It must hash to the
# value baked in below (captured from #5's scaffold). If this fails, #11
# edited #5's territory — revert and re-append only below the marker.
INTAKE_BLOCK=$(awk '/^### Concern: `intake`/{f=1} /#11 APPEND POINT/{f=0} f' \
  domains/pm/commands/scrub.md | shasum -a 256 | cut -d" " -f1)
EXPECTED_INTAKE=37b041541c4cfe34b660cec83400a5d5b0a48ea7bca9cb006b5bb32385d9c4c7
if [ "$INTAKE_BLOCK" != "$EXPECTED_INTAKE" ]; then
  echo "WARN: intake-block hash changed — verify #11 did NOT edit #5's block."
  echo "  If the design intentionally re-baselined it, update EXPECTED_INTAKE."
  echo "  (Placeholder hash: replace with the real pre-edit value at delivery"
  echo "   time via: awk '/^### Concern: \`intake\`/{f=1} /#11 APPEND POINT/{f=0} f'"
  echo "   domains/pm/commands/scrub.md | shasum -a 256)"
fi
# Structural guard that always holds regardless of the hash placeholder:
grep -q '### Concern: `intake`'  domains/pm/commands/scrub.md || { echo "intake concern vanished"; exit 1; }
grep -q '#11 APPEND POINT'       domains/pm/commands/scrub.md || { echo "append marker vanished"; exit 1; }

echo "== AC-4: roadmap + stories concerns appended BELOW the marker =="
MARKER=$(grep -n '#11 APPEND POINT' domains/pm/commands/scrub.md | head -1 | cut -d: -f1)
ROADMAP=$(grep -n '### Concern: `roadmap`' domains/pm/commands/scrub.md | head -1 | cut -d: -f1)
STORIES=$(grep -n '### Concern: `stories`' domains/pm/commands/scrub.md | head -1 | cut -d: -f1)
[ -n "$ROADMAP" ] && [ "$ROADMAP" -gt "$MARKER" ] || { echo "roadmap concern missing/above marker"; exit 1; }
[ -n "$STORIES" ] && [ "$STORIES" -gt "$MARKER" ] || { echo "stories concern missing/above marker"; exit 1; }
echo "ok"

echo "== AC-3: scrubbers are report-only (edit: deny) =="
for s in stale-roadmap-scrubber ambiguous-story-scrubber; do
  grep -qE 'edit:\s*deny' "domains/pm/agents/$s.md" || { echo "$s not edit:deny"; exit 1; }
done
echo "ok"

echo "== AC-2: no dangling skill refs — every skill an agent loads exists on disk =="
DANGLING=0
for a in epic-framer risk-curator portfolio-curator discovery-reviewer \
         stale-roadmap-scrubber ambiguous-story-scrubber; do
  # pull backtick-quoted skill tokens from the Startup section; check each resolves
  for sk in $(grep -oE '`[a-z0-9-]+`' "domains/pm/agents/$a.md" | tr -d '`' | sort -u); do
    if [ -d "domains/pm/skills/$sk" ] || [ -d "core/skills/$sk" ]; then :; else
      # only fail for tokens that are actually referenced as skills to load
      if grep -qE "load|Startup|skill" "domains/pm/agents/$a.md" && \
         grep -q "\`$sk\`" "domains/pm/agents/$a.md"; then
        # heuristic: is this token one of the known designed skills?
        case "$sk" in
          pm-agent-doctrine|epic-framing|story-writing-invest|dependency-mapping|\
          risk-surfacing|assumption-testing|evidence-synthesis|outcomes-over-outputs|\
          roadmap-framing|prioritization-frameworks|opportunity-solution-trees-torres|\
          discovery-interview-design|outcome-drift|acceptance-criteria-ears|launch-gtm-tiering)
            echo "DANGLING skill ref in $a: $sk"; DANGLING=1;;
        esac
      fi
    fi
  done
done
[ "$DANGLING" -eq 0 ] || exit 1
echo "ok — all designed skill refs resolve"

echo "== AC-8: AGENTS.md canonical table + prior Wave-2 routes unchanged; child-#11 subsection present =="
grep -q 'WAVE-2 ROUTES' domains/pm/AGENTS.md || { echo "Wave-2 marker vanished"; exit 1; }
grep -qi 'remaining-roles-scrubbers-and-launch' domains/pm/AGENTS.md || { echo "child-#11 subsection missing"; exit 1; }
# canonical table sentinel rows (must survive)
grep -q 'This table is the \*\*canonical\*\* PM routing table' domains/pm/AGENTS.md || { echo "canonical intro changed"; exit 1; }
echo "ok"

echo "== AC-9: rosters updated =="
grep -q 'launch-gtm-tiering' domains/pm/AGENTS.md || { echo "skill not in roster"; exit 1; }
grep -q '/launch' domains/pm/AGENTS.md || { echo "/launch not in roster"; exit 1; }
echo "ok"

echo "== AC-10: no installed harness mirror hand-edited =="
if git status --porcelain | grep -E '^\s*[AM].*(\.claude/|\.agents/|\.codex/)' ; then
  echo "FAIL: an installed mirror was edited — author only in domains/pm/"; exit 1;
fi
echo "ok"

echo "ALL VALIDATION CHECKS PASSED"
```

**Delivery note on the intake-block guard.** `EXPECTED_INTAKE` above is a
placeholder. At the **start** of delivery (before editing `scrub.md`),
capture the real pre-edit hash and substitute it, so the guard becomes a
hard byte-for-byte assertion rather than a warning:

```bash
awk '/^### Concern: `intake`/{f=1} /#11 APPEND POINT/{f=0} f' \
  domains/pm/commands/scrub.md | shasum -a 256
```

The structural `grep` guards (intake concern present, marker present,
new concerns strictly below the marker) hold unconditionally regardless
of the hash.

## Boundaries

- **No Go, no engine changes.** Author only markdown pack source under
  `domains/pm/`. This child adds zero CLI subcommands, zero graph
  schema, zero recognition logic.
- **Do not edit installed mirrors.** `.claude/`, `.agents/`, `.codex/`
  are regenerated by `hero install` from `domains/pm/` — hand-edits are
  overwritten and violate tripwire `harness-changes-cover-all-targets`.
- **Do not touch child #5's `intake` concern block** in `scrub.md`, nor
  its Wave-1 backing subsection in `AGENTS.md`. Append-only, in the
  marked regions, below prior children.
- **Do not edit the AGENTS.md canonical routing table** or any prior
  child's Wave-2 subsection (#3, #4, #5, #6, #7, #8, #9).
- **No `okr-design` skill** — the audit parks it for a possible future
  `strategy` domain, not PM; out of scope here.
- **No new scrubber concerns beyond `roadmap` and `stories`.** `intake`
  is #5's; those two are this child's; nothing else.
- **`portfolio-curator` does not auto-rebalance** and the two scrubbers
  do not auto-apply state changes — report-only, decision-gate doctrine.
  Building any auto-apply path is out of scope.
- **No `/review` command for `discovery-reviewer`.** Consistent with the
  pack design (§F ships no `/review` in pm), it is invoked directly.

## Completion Ledger

Delivered content-only under `domains/pm/` (no Go). Full `## Validation`
bash block run verbatim from repo root — **ALL VALIDATION CHECKS PASSED**;
the intake-block hash guard matched the real pre-edit value
(`37b041541c4cfe34b660cec83400a5d5b0a48ea7bca9cb006b5bb32385d9c4c7`,
substituted into `EXPECTED_INTAKE` per the delivery note) with no WARN
emitted, so AC-5 is a hard byte-for-byte assertion, not a warning.

### Acceptance Criteria

| # | Criterion | Status | Note |
|---|---|---|---|
| AC-1 | Six agent files exist with valid frontmatter (`name`, `description`, `mode: subagent`, `permission`) | DONE | `domains/pm/agents/{epic-framer,risk-curator,portfolio-curator,discovery-reviewer,stale-roadmap-scrubber,ambiguous-story-scrubber}.md`; AC-1 loop green |
| AC-2 | Every agent loads `pm-agent-doctrine` + designed skills; every skill ref resolves on disk (zero dangling) | DONE | Each Startup loads `pm-agent-doctrine` first; no-dangling check prints "ok — all designed skill refs resolve" |
| AC-3 | Both scrubbers report-only (`permission.edit: deny`) | DONE | `edit: deny` in both scrubber frontmatters; AC-3 check green |
| AC-4 | `### Concern: roadmap` + `### Concern: stories` appended below the `#11 APPEND POINT` marker | DONE | Both concern line numbers `> MARKER`; roadmap → `stale-roadmap-scrubber`, stories → `ambiguous-story-scrubber` |
| AC-5 | Child #5 `intake` block byte-for-byte unchanged | DONE | Post-edit intake-block hash == pre-edit `37b0415…`; no WARN; `git diff` on scrub.md removes only the frontmatter description line (above the block) |
| AC-6 | `launch-gtm-tiering/SKILL.md` — tier 1/2/3 + five-phase checklist | DONE | `domains/pm/skills/launch-gtm-tiering/SKILL.md`; rubric table + phase-coverage table (alignment → positioning → enablement → launch → post-launch) |
| AC-7 | `commands/launch.md` — `/launch` plan+checklist, routes to `stakeholder-communicator`, loads `launch-gtm-tiering`, notes route | DONE | `domains/pm/commands/launch.md`; tier detection + tier-scoped checklist + explicit route note |
| AC-8 | New child-#11 subsection below last Wave-2 subsection; canonical table + prior Wave-2 routes byte-unchanged | DONE | `#### Wave-3 remaining roles, scrubbers & launch routes …` added after the exec-narrative subsection; AGENTS.md diff removes only the 2 Commands-Reference roster lines (in-scope #11), no table/subsection lines touched |
| AC-9 | Six agents in Agents Reference; `launch-gtm-tiering` in Skills Reference; `/launch` + scrub concerns in Commands Reference | DONE | PM Wave-3 agents bullet; Launch/GTM skills bullet; `/launch` in PM list + rewritten `/scrub` concerns roster line |
| AC-10 | All edits confined to `domains/pm/`; no installed mirror hand-edited | DONE | AC-10 check green; `git status` shows no `.claude/`/`.agents/`/`.codex/` edits |

### Changes

| # | Changes item | Status | Note |
|---|---|---|---|
| 1 | `domains/pm/agents/epic-framer.md` (authoring, edit: allow) | DONE | Coherent-bet framing, rollup AC, sequencing+deps; loads `pm-agent-doctrine`, `epic-framing`, `story-writing-invest`, `dependency-mapping`; delegates bodies to `story-writer` |
| 2 | `domains/pm/agents/risk-curator.md` (authoring, edit: allow) | DONE | Scenario+indicator+response; test-now vs. defer; loads `pm-agent-doctrine`, `risk-surfacing`, `assumption-testing`, `evidence-synthesis`; delegates tests to `discovery-researcher` |
| 3 | `domains/pm/agents/portfolio-curator.md` (curation, edit: allow) | DONE | Cross-roadmap theme balance + capacity-vs-ambition; recommends, never auto-rebalances; loads `pm-agent-doctrine`, `outcomes-over-outputs`, `roadmap-framing`, `prioritization-frameworks` (NOT `capacity-planning`) |
| 4 | `domains/pm/agents/discovery-reviewer.md` (critic, edit: deny) | DONE | Adversarial rigor review; loads `pm-agent-doctrine`, `opportunity-solution-trees-torres`, `discovery-interview-design`, `assumption-testing`, `evidence-synthesis`; routes back to `discovery-researcher` |
| 5 | `domains/pm/agents/stale-roadmap-scrubber.md` (scrubber, edit: deny) | DONE | Report-only `/scrub roadmap`; loads `pm-agent-doctrine`, `roadmap-framing`, `outcome-drift` |
| 6 | `domains/pm/agents/ambiguous-story-scrubber.md` (scrubber, edit: deny) | DONE | Report-only `/scrub stories`; loads `pm-agent-doctrine`, `story-writing-invest`, `acceptance-criteria-ears` |
| 7 | Extend `domains/pm/commands/scrub.md` — append `roadmap` + `stories` concerns below marker; description → past tense | DONE | Two concern blocks appended in the `#11 APPEND POINT` region only; frontmatter description updated (above the intake block); intake block byte-unchanged |
| 8 | `domains/pm/skills/launch-gtm-tiering/SKILL.md` | DONE | Tier rubric (blast radius / revenue / net-new / competitive) + five-phase checklist with per-tier coverage; corpus-grounded + decision-gated framing |
| 9 | `domains/pm/commands/launch.md` | DONE | `/launch` → tier detection → tier-scoped phased checklist; routes to `stakeholder-communicator`, notes per-phase owners; passes `$ARGUMENTS` |
| 10 | Append routes to `domains/pm/AGENTS.md` Wave-2 region below last subsection | DONE | Wave-3 subsection with one route row each for the 6 agents + `/scrub roadmap` + `/scrub stories` + `/launch`; canonical table + prior subsections unchanged |
| 11 | Update the three AGENTS.md reference rosters (Agents / Skills / Commands) | DONE | PM Wave-3 agents bullet; Launch/GTM skills bullet; `/launch` added + `/scrub` roster line rewritten to shipped state |

### Exercise-the-feature check

Content-only markdown pack source — no runtime surface to drive (no Go, no CLI subcommand, no server). "Exercising the feature" here is the structural + hash validation the spec defines, all of which passed:

- [x] Full `## Validation` bash block run verbatim → `ALL VALIDATION CHECKS PASSED`
- [x] AC-5 intake-block hash guard is a hard assertion (real pre-edit hash substituted), matched with no WARN
- [x] AC-8 verified additions-only via `git diff --numstat` + removed-line inspection (only the 2 in-scope roster lines removed)
- [x] AC-2 no-dangling: every skill token each of the six agents loads resolves to a real dir under `domains/pm/skills/` or `core/skills/`
- [x] AC-10: `git status` shows zero edits under `.claude/` / `.agents/` / `.codex/`
- [ ] Live `/launch` / `/scrub roadmap` / `/scrub stories` dispatch in a harness session — NOT run here; these are prompt-driven agent routes exercised at install time (would require `hero install` into a harness), out of scope for this content-only delivery. Routing correctness verified by inspection: `scrub.md` maps both new concerns to their agents, `launch.md` routes to `stakeholder-communicator`, and AGENTS.md carries every route row.

### Excellence Bar self-check

- **Corpus-grounded + decision-gated doctrine, per role.** Every agent loads `pm-agent-doctrine` first and carries its own anti-patterns section; authoring agents surface proposals, the critic and both scrubbers are report-only with explicit "no findings" outputs, `portfolio-curator` recommends-never-rebalances. Matches the shipped Wave-1/2 tone and section discipline (Startup / When invoked / Workflow / Produces / Delegation / Anti-patterns).
- **Seam discipline held structurally.** All `scrub.md` edits confined to the marked append region; the byte-for-byte intake guard is now a hard hash assertion, not a placeholder warning. AGENTS.md canonical table and all seven prior children's subsections are untouched.
- **The skill carries real reasoning, not a checklist.** `launch-gtm-tiering` gives a scored rubric (highest-hitting dimension pulls the tier up), a worked SSO example that inverts a naive "one segment = Tier 2" read, tier→phase collapse rules, and grounded/gated framing — usable, not decorative.
- **Would I show this to a senior PM-pack author?** Yes — the six agents, skill, command, and routes read as one coherent Wave-3 rounding pass consistent with the pack that shipped before them.

