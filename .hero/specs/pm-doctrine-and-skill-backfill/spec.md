---
title: "PM Doctrine Spine + Referenced-Skill Backfill + Canonical AGENTS.md"
slug: pm-doctrine-and-skill-backfill
type: feature
status: completed
domain: pm
priority: critical
size: large
created: 2026-07-17
tags: [pm, doctrine, skills, routing, wave-0, wave-1]
relations:
  - target: pm-pack-completion
    kind: parent
  - target: prd-editor-comms-backing
    kind: conflicts-with
  - target: adversarial-critics-bundle
    kind: conflicts-with
  - target: experiment-stage-and-metric-rca
    kind: conflicts-with
  - target: competitive-and-market-grounding
    kind: conflicts-with
  - target: exec-narrative-and-evidence-synthesis
    kind: conflicts-with
completed_at: 2026-07-17T20:02:52Z
---

# PM Doctrine Spine + Referenced-Skill Backfill + Canonical AGENTS.md

Root anchor of the `pm-pack-completion` initiative — its sole `critical`
child. Every other child carries a `depends-on` edge to this one because it
delivers two shared foundations they all build on: the **pack-wide doctrine
spine** (`pm-agent-doctrine` + `outcomes-over-outputs`) and the **canonical
`domains/pm/AGENTS.md` routing table** that Wave-2 children register their
net-new agents into. Content only — no Go code.

## Goal

Close the three foundation gaps the 2026-07-17 PM pack audit surfaced, so the
PM pack stops referencing skills that don't exist and every agent shares one
doctrine:

1. **Author the doctrine spine** — `pm-agent-doctrine` (corpus-grounding
   contract / suggest-don't-decide gates / compare-don't-replace synthesis)
   and `outcomes-over-outputs` (the external scan's #1 load-bearing framework,
   loaded by the strategy/review spine).
2. **Backfill the 9 referenced-but-unauthored skills** so no PM agent
   load-list (shipped or Wave-2-designed) dangles.
3. **Replace the `domains/pm/AGENTS.md` scaffold routing table** with the full
   canonical table (`agent-pack-design.md` §F), reconciled to shipped reality,
   with a marked Wave-2 appendable region this child owns.
4. **Retrofit** the doctrine spine + newly-authored skills into the 8 shipped
   agents' load-lists per the design's §C.
5. **Reconcile the `dashboard.md` orphan** (Wave-0 item) against the live tree.

All authoring happens in the **`domains/pm/` pack source**. `hero install`
renders that source to all six harness targets (claude → `CLAUDE.md`; the
others → `AGENTS.md`). This spec touches **no per-target file and no installed
`CLAUDE.md`** — see the tripwire note under Boundaries.

## Kickoff

Deliver the PM doctrine spine and skill backfill, content-only, in
`domains/pm/` pack source. Author `skills/pm-agent-doctrine` and
`skills/outcomes-over-outputs` (match shipped skill frontmatter shape), plus 9
backfill skills. Retrofit the spine + new skills into the 8 shipped agents'
`## Startup` load-lists per `agent-pack-design.md` §C. Replace the
`domains/pm/AGENTS.md` scaffold routing table with the canonical §F table +
the `<!-- WAVE-2 ROUTES ... -->` appendable region. Verify no dangling skill
refs remain. Do NOT edit any installed `.claude/`/`CLAUDE.md`. Run the
Validation greps before flipping status.

## Problem

The PM pack shipped a deliberate v1 minimum-viable set, but hero-code now
embeds the engine and exercises surfaces v1 deferred, exposing three
foundation gaps (audit `pm-pack-audit-2026-07.md`):

- **Doctrine is an afterthought, not a shared skill.** The external best-
  practice scan is unambiguous: generic PM generators are commoditized and
  *distrusted* (Notion AI PRDs ~30% generic/hallucinated; fabricated interview
  quotes). The trusted, high-leverage capability is *rigor* — corpus-grounded,
  decision-gated, compare-don't-replace. Today that discipline lives in prose
  scattered across agents, if at all. It must become one skill every
  authoring/critic agent loads: `pm-agent-doctrine`.
- **Referenced skills don't exist.** The design's §C/§D agent load-lists and
  the audit name skills with no backing file. `outcomes-over-outputs` is the
  worst offender — the spine framework the design threads through the strategy
  and review agents — and 8 further skills (`competitive-research`,
  `feature-comparison-framing`, `epic-framing`, `horizon-assignment`,
  `customer-segment-weighting`, `risk-surfacing`, `assumption-testing`,
  `discovery-interview-design`, `acceptance-criteria-gherkin`) are named by
  shipped or Wave-2-designed agents. Every dangling ref is a silent no-op at
  load time.
- **`AGENTS.md` is a scaffold, and it's the initiative's hottest file.**
  Wave-2 children (#4, #6, #7, #8, #9) each add agents whose routes must land
  in this one table. Without a canonical base table + a marked append region
  owned here, those children collide on it. (The reciprocal `conflicts-with`
  edges in this spec's frontmatter serialize them against this child.)

### Design tension to resolve at delivery (surfaced, not silently decided)

`agent-pack-design.md` §F is the *design-era* routing table. The **current**
`domains/pm/AGENTS.md` scaffold is in several ways *more* correct than §F: it
carries a vocabulary-variant column, encodes the owner-flip handoff model
(not §F's "produce a new engineering spec"), and honestly annotates which §F
commands do **not** ship a v1 surface (`/capacity`, `/plan-*`, `/standup`,
`/interview` → `/discover --interview`, `/review` → direct `pm-reviewer`
invoke). A blind overwrite with §F would *regress* those truths.

**Resolution (baked into AC-9/AC-10):** the canonical table must have **§F's
full intent coverage** (a row for every §F intent) **while preserving the
shipped-reality annotations and the vocabulary-variant column** the current
scaffold already got right. Coverage from §F; accuracy from the current file.
This is a reconcile, not a paste.

## Acceptance Criteria

EARS clauses used where they fit; file-existence and grep-gate criteria are
stated as concrete checks a cold auditor runs (see Validation for exact
commands).

| ID | EARS / pattern | Behavior |
|---|---|---|
| AC-1 | Ubiquitous | `domains/pm/skills/pm-agent-doctrine/SKILL.md` exists with `name`/`description`/`metadata: {audience, purpose}` frontmatter matching shipped skill shape, and body sections covering all three doctrines: **corpus-grounding contract**, **suggest-don't-decide decision gates**, **compare-don't-replace synthesis**. |
| AC-2 | Ubiquitous | `domains/pm/skills/outcomes-over-outputs/SKILL.md` exists with matching frontmatter and body covering the **outcome ladder**, **outcome vs output vs input** distinction, and the **~60/30/10** outcome/output/input ratio. |
| AC-3 | Ubiquitous | All 9 backfill skills exist as `domains/pm/skills/<name>/SKILL.md` with matching frontmatter: `competitive-research`, `feature-comparison-framing`, `epic-framing`, `horizon-assignment`, `customer-segment-weighting`, `risk-surfacing`, `assumption-testing`, `discovery-interview-design`, `acceptance-criteria-gherkin`. |
| AC-4 | Event-driven | WHEN `hero install` renders the pack, THE doctrine spine and backfill skills SHALL render from `domains/pm/` source into every target (no per-target authored copy exists). |
| AC-5 | Ubiquitous | `pm-agent-doctrine` appears in the `## Startup` load-list of **all 8** retrofit agents (product-strategist, pm-reviewer, roadmap-curator, intake-triager, prioritization-strategist, story-writer, prd-author, discovery-researcher). |
| AC-6 | Ubiquitous | `outcomes-over-outputs` appears in the load-list of **exactly** the agents §C designates among the 8: `product-strategist` and `pm-reviewer`. (The other 4 of the "spine 6" — metrics-analyst, portfolio-curator, roadmap-reviewer, stakeholder-communicator — are unshipped; they load it when authored in Waves 2–3. This is recorded, not an omission.) |
| AC-7 | Ubiquitous | Each backfill skill is wired into the load-list of the shipped agent(s) §C designates: `risk-surfacing` → product-strategist + roadmap-curator (+ available to pm-reviewer for risk review); `horizon-assignment` → roadmap-curator; `customer-segment-weighting` → intake-triager + prioritization-strategist; `discovery-interview-design` + `assumption-testing` → discovery-researcher; `acceptance-criteria-gherkin` → story-writer. |
| AC-8 | Unwanted behavior | IF a backfill skill has no shipped-agent consumer (`competitive-research`, `feature-comparison-framing`, `epic-framing`), THEN the spec SHALL document it as **forward-authored** for its named downstream agent (competitive-analyst → competitive-research + feature-comparison-framing; epic-framer → epic-framing), so it is a recorded forward reference, not an orphan. |
| AC-9 | Event-driven | WHEN a PM natural-language intent matches any §F row, THE canonical `domains/pm/AGENTS.md` table SHALL route it — i.e. every one of the 26 §F intents has a corresponding row (verified by the command-token grep in Validation). |
| AC-10 | State-driven | WHILE preserving §F coverage, the canonical table SHALL retain the vocabulary-variant column and the shipped-reality command annotations from the current scaffold (no §F row may claim a v1 surface the pack doesn't ship). |
| AC-11 | Ubiquitous | `domains/pm/AGENTS.md` contains exactly one appendable region opened by the literal marker `<!-- WAVE-2 ROUTES: appended by adversarial-critics-bundle / experiment-stage-and-metric-rca -->` (also serving children #4/#8/#9), placed **after** the canonical rows; the region is empty on delivery of this child. |
| AC-12 | Unwanted behavior | IF any skill name appears in a PM agent load-list, THEN a real `domains/pm/skills/<name>/SKILL.md` SHALL resolve it — **no dangling skill refs remain** (Validation's dangling-ref scan returns empty). |
| AC-13 | Ubiquitous | `story-writer.md`'s prior "acceptance-criteria-gherkin is planned for v1.5 — stay in EARS" prose is updated to reflect that the skill now ships (offer Gherkin on request), so the file no longer describes an unshipped skill. |
| AC-14 | Unwanted behavior | IF the tree is checked for `domains/pm/commands/dashboard.md`, THEN it SHALL be confirmed **absent** (Wave-0 orphan resolved by drop), and no `AGENTS.md`/command references a `/dashboard` command that would dangle. |

## Changes

Every path is under the `domains/pm/` **pack source** (never an installed
harness dir). Skill bodies follow the shipped shape seen in
`skills/roadmap-framing/SKILL.md` and `skills/prioritization-frameworks/SKILL.md`
(frontmatter `name` / `description` / `metadata.{audience,purpose}`; body:
*What I do* / *When to use me* / core stance / anti-patterns / cross-references).

### New skills — doctrine spine (2)

- `domains/pm/skills/pm-agent-doctrine/SKILL.md` — pack-wide doctrine. Body:
  - **Corpus-grounding contract** — every PM suggestion cites the team's own
    corpus (linked intake / calls / tracker / analytics / research notes); no
    free-association or model-memory claims. Uncited assertion = flag the gap,
    don't ship the claim. Source for this section: audit "Cross-cutting
    doctrine" §1.
  - **Suggest-don't-decide decision gates** — suggestions are **marked**,
    **reversible**, **explainable**, **human-accountable**. Never auto-decide
    prioritization or strategy; the human owns the call, the agent owns the
    audit trail. Source: audit §2.
  - **Compare-don't-replace synthesis** — the agent does its pass, the PM does
    theirs, they reconcile; protects against outsourcing judgment. Source:
    audit §3.
  - `metadata.audience`: the authoring/critic agents (all 8 retrofit targets +
    the Wave-2 critics that will depend on it). `metadata.purpose: doctrine`.
- `domains/pm/skills/outcomes-over-outputs/SKILL.md` — the spine framework
  (Cagan/SVPG). Body: outcome ladder (input → output → outcome → impact);
  the outcome-vs-output-vs-input distinction with a pass/fail table (mirror
  the `roadmap-framing` output/outcome table style); the ~**60/30/10**
  outcome/output/input framing ratio and how to apply it when reviewing a
  roadmap/PRD; anti-patterns (output-framed bets, vanity outputs quoted as
  outcomes). `metadata.audience`: product-strategist, pm-reviewer (+ the 4
  unshipped spine-loaders, noted). Cross-ref `roadmap-framing`,
  `metrics-design`.

### New skills — backfill (9)

Each `domains/pm/skills/<name>/SKILL.md`, content grounded in `agent-pack-design.md`
§C/§D where the design specced it, else authored to the shipped skill bar:

- `risk-surfacing` — concrete risk naming (scenario + indicator + response);
  "might not scale" is not a risk. (§D.4)
- `assumption-testing` — Torres-style fast tests: desirability/viability/
  feasibility/usability; pre-registered pass/fail; resolve in days. (§D.6)
- `discovery-interview-design` — non-leading questions about specific past
  experience; 5–10/week cadence; structured synthesis. (§D.2)
- `acceptance-criteria-gherkin` — Given/When/Then as the alt AC shape; data
  tables; scenario outlines; anti-pattern "Gherkin novels". (§D.1)
- `horizon-assignment` — now/next/later (and quarter) assignment discipline;
  what makes something honestly "now". (extracted/aligned with the horizon
  section already in `roadmap-framing`; cross-ref it, don't duplicate).
- `customer-segment-weighting` — weight reach/impact by segment economics;
  team-decided weights recorded once; disclose weighting in notes.
  (cross-ref `prioritization-frameworks`' segment-weighting section).
- `competitive-research` — **retrieval-augmented, never model-memory**;
  teardown of what competitors actually ship; parity vs differentiation.
  Forward-authored for `competitive-analyst` (Wave-2 child #8).
- `feature-comparison-framing` — feature-matrix framing; must-match parity vs
  optional differentiation vs white space. Forward-authored for
  `competitive-analyst` (child #8).
- `epic-framing` — an epic is a coherent bet, not a bag of stories; Why +
  rollup AC + sequenced children. Forward-authored for `epic-framer`
  (Wave-3 child #11).

### Agent load-list retrofits (8 files)

Edit only the `## Startup` "Load before substantial work" bullets (and, for
`pm-reviewer`, its step-1 conditional load block). No prompt rewrites. `skill:`
permission is already `"*": allow` on all 8, so no permission edits.

| Agent | Add to load-list |
|---|---|
| `agents/product-strategist.md` | `pm-agent-doctrine`, `outcomes-over-outputs`, `risk-surfacing` |
| `agents/pm-reviewer.md` | `pm-agent-doctrine`, `outcomes-over-outputs` (unconditional); `risk-surfacing` available in the initiative/PRD conditional-load branch |
| `agents/roadmap-curator.md` | `pm-agent-doctrine`, `horizon-assignment`, `risk-surfacing` |
| `agents/intake-triager.md` | `pm-agent-doctrine`, `customer-segment-weighting` |
| `agents/prioritization-strategist.md` | `pm-agent-doctrine`, `customer-segment-weighting` |
| `agents/story-writer.md` | `pm-agent-doctrine`, `acceptance-criteria-gherkin` (+ update the "planned for v1.5" prose per AC-13) |
| `agents/prd-author.md` | `pm-agent-doctrine` |
| `agents/discovery-researcher.md` | `pm-agent-doctrine`, `discovery-interview-design`, `assumption-testing` |

### `domains/pm/AGENTS.md` — canonical routing table (1 file)

- Replace the current scaffold routing table (the table under **Natural
  Language Routing**, current lines ~21–48) with the **reconciled canonical
  table**: one row for each of the 26 `agent-pack-design.md` §F intents,
  **keeping** the current file's vocabulary-variant column and its shipped-
  reality command annotations (per AC-9/AC-10). Where §F names a command with
  no v1 surface, carry the current scaffold's honest annotation rather than
  §F's aspirational command.
- Immediately after the canonical rows, insert the appendable region:

  ```
  <!-- WAVE-2 ROUTES: appended by adversarial-critics-bundle / experiment-stage-and-metric-rca -->
  <!-- Children #4, #6, #7, #8, #9 append net-new agent routes BELOW this line only.
       Do NOT edit the canonical rows above. This region is owned by
       pm-doctrine-and-skill-backfill; downstream children only append. -->
  ```

  The region is **empty** on this child's delivery (AC-11). This is the
  `<!-- wave-2 additions -->` region the initiative's "AGENTS.md routing-table
  hotspot" section refers to — the literal marker text above is authoritative.
- Leave the surrounding AGENTS.md sections (Vocabulary-aware routing,
  Methodology presets, Agents/Skills Reference, etc.) intact except: the
  **Skills Reference** list should gain the 11 newly-authored skills so the
  reference stays truthful (add under Writing/Frameworks/Curation/Operational
  as fits; `pm-agent-doctrine` + `outcomes-over-outputs` under a doctrine line).

### Orphan reconcile (0 files — confirm absence)

- `domains/pm/commands/dashboard.md` is **absent** as of the design pass
  (verified: `ls` returns no such file). Wave-0 resolution = **drop** (there is
  nothing to remove). Record in the delivery ledger that the orphan is already
  gone; assert absence in Validation (AC-14) and confirm no `/dashboard`
  route dangles anywhere in the pack.

### Delivered (actual — 2026-07-17)

New skill files (11), each `domains/pm/skills/<name>/SKILL.md`:
- `pm-agent-doctrine` — corpus-grounding / suggest-don't-decide / compare-don't-replace, worked pass + quick-check.
- `outcomes-over-outputs` — outcome ladder, outcome/output/input pass-fail table, 60/30/10 ratio, worked roadmap audit, leading-vs-lagging.
- `risk-surfacing`, `assumption-testing`, `discovery-interview-design`, `acceptance-criteria-gherkin`, `horizon-assignment`, `customer-segment-weighting`, `competitive-research`, `feature-comparison-framing`, `epic-framing`.

Agent load-list retrofits (8), `domains/pm/agents/*.md`:
- `product-strategist` — Startup: +`pm-agent-doctrine`, +`outcomes-over-outputs`, +`risk-surfacing`.
- `pm-reviewer` — new `## Startup` with unconditional `pm-agent-doctrine` + `outcomes-over-outputs`; step-1 conditional load: +`risk-surfacing` on PRDs/Initiatives.
- `roadmap-curator` — step-1 load: +`pm-agent-doctrine`, +`horizon-assignment`, +`risk-surfacing`.
- `intake-triager` — step-1 load: +`pm-agent-doctrine`, +`customer-segment-weighting`.
- `prioritization-strategist` — step-1 load: +`pm-agent-doctrine`, +`customer-segment-weighting`.
- `story-writer` — Startup: +`pm-agent-doctrine`, +`acceptance-criteria-gherkin`; AC-13 gherkin prose updated (no longer "planned for v1.5").
- `prd-author` — Startup: +`pm-agent-doctrine`.
- `discovery-researcher` — Startup: +`pm-agent-doctrine`, +`discovery-interview-design`, +`assumption-testing`.

Routing table + reference, `domains/pm/AGENTS.md`:
- Scaffold routing table replaced with the reconciled canonical table (26 §F intents, vocab-variant column + shipped-reality annotations preserved); Wave-2 marker + empty append region inserted after the canonical rows; Skills Reference gained a Doctrine line + the 11 new skills.

Orphan: `domains/pm/commands/dashboard.md` confirmed absent (drop = nothing to remove); no `/dashboard` route in the pack.

**AC-12 note (resolved by delivery lead):** the original scan compared *all*
backticked prose tokens against `skills/`, which over-matched agent / spec-type /
status / core-skill cross-references (`discovery-researcher`, `epic`, `planning`,
`context-injection`, …) — none of which are dangling *skill* refs. The gate was
corrected (above) to measure the actual invariant: skills listed in an agent's
"Load before substantial work:" block must resolve in the pm pack **or** the core
overlay (`core/skills/`, merged at install). The corrected scan returns empty —
every loaded skill resolves. The delivery closed the one true dangle
(`acceptance-criteria-gherkin`); the earlier prose over-matches were never skill
dangles. No agent prose was de-backticked (Boundaries preserved).

## Validation

Exact commands a cold auditor runs from the repo root
(`/Users/bwheeler/projects/hero-engine/repository/hero`). All must pass before
`status` flips to `completed`.

```bash
cd /Users/bwheeler/projects/hero-engine/repository/hero/domains/pm

# AC-1/2/3 — all 11 new skill files exist with a name: frontmatter line
for s in pm-agent-doctrine outcomes-over-outputs risk-surfacing assumption-testing \
         discovery-interview-design acceptance-criteria-gherkin horizon-assignment \
         customer-segment-weighting competitive-research feature-comparison-framing epic-framing; do
  test -f "skills/$s/SKILL.md" && grep -q "^name: $s" "skills/$s/SKILL.md" \
    && echo "OK  $s" || echo "FAIL $s"
done

# AC-1 — doctrine covers all three doctrines (section presence, case-insensitive)
grep -qi "corpus-ground"        skills/pm-agent-doctrine/SKILL.md && \
grep -qiE "decision gate|suggest.*(don.?t|not).*decide" skills/pm-agent-doctrine/SKILL.md && \
grep -qi "compare-don.?t-replace\|compare, not replace\|compare don" skills/pm-agent-doctrine/SKILL.md \
  && echo "OK doctrine-sections" || echo "FAIL doctrine-sections"

# AC-2 — outcomes skill covers ladder + ratio
grep -qi "outcome ladder" skills/outcomes-over-outputs/SKILL.md && \
grep -qE "60/30/10|60 */ *30 */ *10" skills/outcomes-over-outputs/SKILL.md \
  && echo "OK outcomes-sections" || echo "FAIL outcomes-sections"

# AC-5 — pm-agent-doctrine in ALL 8 agent load-lists
for a in product-strategist pm-reviewer roadmap-curator intake-triager \
         prioritization-strategist story-writer prd-author discovery-researcher; do
  grep -q "pm-agent-doctrine" agents/$a.md && echo "OK  doctrine@$a" || echo "FAIL doctrine@$a"
done

# AC-6 — outcomes-over-outputs in EXACTLY product-strategist + pm-reviewer (and no other of the 8)
grep -l "outcomes-over-outputs" agents/*.md | sort   # expect exactly product-strategist.md, pm-reviewer.md

# AC-7 — backfill skills wired where §C designates
grep -q "risk-surfacing"              agents/product-strategist.md
grep -q "risk-surfacing"              agents/roadmap-curator.md
grep -q "horizon-assignment"          agents/roadmap-curator.md
grep -q "customer-segment-weighting"  agents/intake-triager.md
grep -q "customer-segment-weighting"  agents/prioritization-strategist.md
grep -q "discovery-interview-design"  agents/discovery-researcher.md
grep -q "assumption-testing"          agents/discovery-researcher.md
grep -q "acceptance-criteria-gherkin" agents/story-writer.md

# AC-9 — every one of the 26 §F command tokens appears in AGENTS.md canonical region
for c in /triage /refine /prioritize /handoff /prd /pitch /roadmap /discover /metrics \
         /interview /release-notes /capacity /plan- /standup "/scrub roadmap" \
         "/scrub intake" "/scrub stories" /diagnose /design /search /why /blocked \
         /note /decide /review /retro; do
  grep -q -- "$c" AGENTS.md && echo "OK  route $c" || echo "MISS route $c"
done

# AC-11 — exactly one canonical Wave-2 marker, region empty (no route rows between marker and next section)
grep -c "WAVE-2 ROUTES: appended by adversarial-critics-bundle" AGENTS.md   # expect 1

# AC-12 — DANGLING-REF SCAN: every skill listed in an agent's "Load before
#   substantial work:" block resolves to a real skill dir — in EITHER the pm pack
#   (skills/) OR the core overlay (../../core/skills/), since install merges both.
#   Scoped to the load block (not all backticked prose): a skill is the leading
#   `token` of a "- `skill` — ..." bullet inside the load block. This measures the
#   actual invariant; scanning all backticked prose over-matches agent/type/status
#   names (discovery-researcher, epic, planning, …) that are not skills.
{ ls skills/; ls ../../core/skills/; } | sort -u > /tmp/pm_all_skills.txt
for a in agents/*.md; do
  awk '/Load before substantial work:/{f=1;next} f&&/^$/{f=0} f&&/^- `/{print}' "$a"
done | grep -oE '^- `[a-z][a-z0-9-]+`' | grep -oE '`[a-z][a-z0-9-]+`' | tr -d '`' | sort -u > /tmp/pm_loaded.txt
comm -23 /tmp/pm_loaded.txt /tmp/pm_all_skills.txt | tee /tmp/pm_dangling.txt
test ! -s /tmp/pm_dangling.txt && echo "OK no-dangling-refs" || echo "FAIL dangling refs listed above"

# AC-13 — story-writer no longer describes gherkin as unshipped/v1.5
! grep -qi "planned for v1.5" agents/story-writer.md && echo "OK gherkin-prose-updated" || echo "FAIL stale gherkin prose"

# AC-14 — dashboard orphan absent; no /dashboard route dangles
test ! -f commands/dashboard.md && echo "OK no-dashboard-file" || echo "FAIL dashboard file present"
! grep -rq "/dashboard" AGENTS.md commands/ && echo "OK no-dashboard-route" || echo "FAIL /dashboard route dangles"
```

The AC-12 dangling-ref scan is the load-bearing gate: it is the machine proof
that the backfill closed every referenced-but-unauthored skill. It must print
`OK no-dangling-refs` with an empty `/tmp/pm_dangling.txt`.

## Boundaries (not in scope)

- **No Go code.** This is a content-only child (skills, agent load-lists,
  AGENTS.md prose).
- **Tripwire: harness-changes-cover-all-targets.** All authoring is in
  `domains/pm/` pack source, which `hero install` renders to all six targets
  (claude → `CLAUDE.md`; others → `AGENTS.md`). Do **not** edit any installed
  `.claude/`, installed `CLAUDE.md`, or any per-target copy. The routing table
  is authored in `domains/pm/AGENTS.md` (pack source) only.
- **No new Wave-2 agent routes.** This child authors the canonical base table
  and the empty append region only. `pitch-author`, `stakeholder-communicator`,
  the critics, `experiment-*`, `metrics-analyst`, `competitive-analyst`, and
  their routes belong to Wave-2 children #4/#6/#7/#8/#9 (which append into the
  region this child opens).
- **No prompt rewrites.** Agent edits are load-list line additions + the one
  gherkin prose fix (AC-13), nothing more.
- **Deferred spine-loaders not authored here.** metrics-analyst,
  portfolio-curator, roadmap-reviewer, stakeholder-communicator are unshipped;
  they load `outcomes-over-outputs`/`pm-agent-doctrine` when authored in later
  waves. Not this child's job.
- **No sizing/compose nudge.** `size: large` is already composed as a child of
  `pm-pack-completion`; splitting further would fragment the shared foundation
  every sibling depends on.

## Completion Ledger

Content-only delivery in the `domains/pm/` pack source (Hero PM domain pack).
Skills loaded: `completion-ledger`, `context-injection`. Validation: ran the
spec's full `## Validation` bash block verbatim from repo root — all gates OK,
including the corrected AC-12 dangling-ref scan (`OK no-dangling-refs`). Cold
audit: SHIP (report at `delivery-audit.md`, noteworthy — AC-12 gate rewrite
independently re-verified as legitimate). No Go, no installed harness files.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | `pm-agent-doctrine` exists; 3 doctrines covered | DONE | `skills/pm-agent-doctrine/SKILL.md` (135 lines) — corpus-grounding / suggest-don't-decide / compare-don't-replace; grep `OK doctrine-sections` |
| 2 | `outcomes-over-outputs` exists; ladder + 60/30/10 | DONE | `skills/outcomes-over-outputs/SKILL.md` (122 lines); grep `OK outcomes-sections` |
| 3 | 9 backfill skills exist w/ shipped-shape frontmatter | DONE | 9 dirs; `OK ×9` on `^name:` lines |
| 4 | Renders from pack source to all targets; no per-target copy | DONE | Authored only in `domains/pm/`; scope-clean confirmed; render structurally guaranteed by source-only authoring |
| 5 | `pm-agent-doctrine` in all 8 retrofit agents | DONE | `OK doctrine@` ×8 |
| 6 | `outcomes-over-outputs` in exactly product-strategist + pm-reviewer | DONE | `grep -l` returns exactly those two |
| 7 | Backfill skills wired per §C | DONE | `ok1…ok8` (risk-surfacing, horizon-assignment, customer-segment-weighting, discovery-interview-design, assumption-testing, acceptance-criteria-gherkin) |
| 8 | Consumerless skills documented as forward-authored | DONE | competitive-research / feature-comparison-framing → `audience: competitive-analyst (Wave-2)`; epic-framing → `epic-framer (Wave-3)` |
| 9 | All 26 §F intents route | DONE | `OK route` ×26 in `AGENTS.md` |
| 10 | Vocab-variant column + shipped-reality annotations preserved | DONE | Canonical table keeps 3rd column; every non-shipping §F command annotated, none claims an unshipped v1 surface |
| 11 | Exactly one Wave-2 marker, region empty | DONE | `grep -c` = 1; region empty before next section |
| 12 | No dangling skill refs (scan empty) | DONE | Gate corrected to load-block scope resolving against pm+core overlay; `OK no-dangling-refs`; the one true dangle (`acceptance-criteria-gherkin`) closed; cold auditor independently re-verified |
| 13 | story-writer gherkin prose de-staled | DONE | `OK gherkin-prose-updated` |
| 14 | `dashboard.md` absent; no `/dashboard` route | DONE | `OK no-dashboard-file`, `OK no-dashboard-route` |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | New skills — doctrine spine (2) | DONE | `pm-agent-doctrine`, `outcomes-over-outputs` at load-bearing depth |
| 2 | New skills — backfill (9) | DONE | grounded in §D/audit where specced; spot-checked by cold audit — no stubs |
| 3 | Agent load-list retrofits (8 files) | DONE | per the retrofit table; only load-list bullets + AC-13 prose touched |
| 4 | `AGENTS.md` canonical routing table reconcile | DONE | §F-covering reconciled table + empty Wave-2 region + Skills Reference updated |
| 5 | Orphan reconcile (`dashboard.md`) | DONE | confirmed absent (drop = nothing to remove) |

### Exercise-the-feature check

- [x] Exercised end-to-end via the spec's full Validation bash block run verbatim from repo root — 26 routing tokens resolve, marker count = 1, region empty, all skill/frontmatter/load-list gates green, corrected dangling scan empty, gherkin prose + dashboard absence confirmed. Independently re-run by the cold auditor.
- [ ] Not exercisable: `hero install` render into harness targets (would write installed dirs this spec forbids) — render is structurally guaranteed by authoring only in pack source.

### Excellence Bar self-check

Yes — the doctrine spine and backfill skills are specific, example-rich, and grounded in the audit/design sources (not stubs; cold audit concurred). The AGENTS.md reconcile preserved shipped-reality accuracy rather than blindly pasting §F. The one mid-delivery judgment — correcting the AC-12 gate to measure its true invariant rather than de-backticking prose (which Boundaries forbid) — was surfaced explicitly and independently re-verified by a cold auditor.
