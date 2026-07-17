---
title: "Exec Narrative + Evidence Synthesis — PR-FAQ, Working-Backwards, Interview Synthesis"
slug: exec-narrative-and-evidence-synthesis
type: feature
status: completed
domain: pm
priority: medium
size: small
created: 2026-07-17
tags: [pm, exec-narrative, synthesis, differentiation, wave-2]
relations:
  - target: pm-pack-completion
    kind: parent
  - target: pm-doctrine-and-skill-backfill
    kind: depends-on
  - target: pm-doctrine-and-skill-backfill
    kind: conflicts-with
completed_at: 2026-07-17T22:05:45Z
---

# Exec Narrative + Evidence Synthesis — PR-FAQ, Working-Backwards, Interview Synthesis

## Goal

Ship the two working-backwards authoring skills the PM pack promises but has never
authored — `prfaq-writing` (Amazon PR/FAQ) and `exec-narrative` (Amazon 6-pager +
the "so what?" pressure-test) — and sharpen the discovery/synthesis path around the
audit's evidence-linked-synthesis thesis: **compare-don't-replace, verbatim-quote
traceability, and outlier-surfacing over the modal answer.** Both halves resolve
forward-references that already ship dangling: `stakeholder-communication` and the
`stakeholder-communicator` agent already point at "child #9's `exec-narrative`
skill" as the home for the full format, and `discovery-researcher` already loads
`evidence-synthesis`. This child fills those homes.

The differentiation frame (from the pack audit): the *value is the reasoning
rigor, not the wordsmithing*. A PR-FAQ's worth is that it forces you to surface
the "dragons" before building; a 6-pager's worth is that prose exposes the
logical gaps that bullets hide. These skills author the argument discipline, not
polished launch copy.

## Kickoff

```
Deliver the spec at
.hero/planning/initiatives/pm-pack-completion/exec-narrative-and-evidence-synthesis/spec.md

Content-only, PM pack source only. Do NOT write Go. Author strictly under
domains/pm/ — never the installed .claude/.agents/.codex copies
(tripwire: harness-changes-cover-all-targets).

Five deliverables:
1. NEW domains/pm/skills/prfaq-writing/SKILL.md — Amazon PR/FAQ working-backwards
   (mock press release + anticipated FAQ written as if launched today; target the
   structure + argument that surfaces the dragons, NOT polished prose).
2. NEW domains/pm/skills/exec-narrative/SKILL.md — Amazon 6-pager structure
   (Intro / Goals / Tenets / State of the Business / Lessons Learned / Strategic
   Priorities) + the "so what?" pressure-test; argument rigor over prose; surface
   the logical gaps bullets hide.
3. SHARPEN domains/pm/agents/discovery-researcher.md — add compare-don't-replace
   synthesis, verbatim-quote traceability, and outlier-surfacing (not the modal
   answer) to its stance + workflow. PRESERVE child #1's full startup load-list.
4. EXTEND domains/pm/skills/evidence-synthesis/SKILL.md — additive: add the Torres
   synthesize-then-compare discipline + outlier-surfacing + verbatim-attribution
   preservation. Do NOT remove or truncate existing content.
5. WIRE the two new skills — add prfaq-writing + exec-narrative to
   stakeholder-communicator's startup loads (preserve its 7 existing loads); append
   an exec-narrative / PR-FAQ authoring route to the AGENTS.md Wave-2 region BELOW
   the marker, AFTER all prior children; add both skills to the Skills Reference.
   Do NOT edit the canonical routing table or any prior child's Wave-2 subsection.

Cross-ref pm-agent-doctrine (compare-don't-replace lives there — cross-reference,
don't restate the whole doctrine). Do not duplicate stakeholder-communication's
audience-cut / "so what" material — exec-narrative is the deeper artifact it
cross-refs; keep the boundary the existing skill already drew.
```

## Problem

The PM pack audit (`.hero/planning/features/hero-pm/pm-pack-audit-2026-07.md`,
Wave-2 row "Exec 'so what' / PR-FAQ working-backwards") flags the working-backwards
capability as **partial**: the pattern is *named* in `stakeholder-communication`,
but the full format has no home. Both `domains/pm/skills/stakeholder-communication/SKILL.md`
and `domains/pm/agents/stakeholder-communicator.md` already forward-reference an
`exec-narrative` skill "authored by child #9" — those references are dangling on
disk today (no `domains/pm/skills/exec-narrative/` or `prfaq-writing/` directory
exists). Until this child ships, the deferral points nowhere.

The audit's evidence-linked-synthesis row is also **partial**: `discovery-researcher`
runs synthesis, and `evidence-synthesis` weights and attributes evidence, but neither
yet carries the three disciplines the external scan singles out as the trusted,
underserved capability —

- **compare-don't-replace** — the agent produces its synthesis *alongside* the PM's
  own reading for reconciliation, never as a replacement that outsources judgment;
- **verbatim-quote traceability** — every synthesized theme links back to the exact
  words that produced it, so the PM can diff the read (the failure mode the scan
  names is interview-summary tools that *fabricate* quotes);
- **outlier-surfacing** — report the signal that *didn't* fit the tidy narrative
  (the churned user, the segment where the pattern broke), not just the modal
  answer that confirms the conclusion.

`pm-agent-doctrine` (child #1) already carries compare-don't-replace as pack-wide
doctrine 3. This child operationalizes it in the two surfaces that do the actual
synthesis work — cross-referencing the doctrine, not restating it.

## Acceptance Criteria

| # | Criterion | Check |
|---|---|---|
| AC1 | `domains/pm/skills/prfaq-writing/SKILL.md` exists with valid frontmatter (`name`, `description`, `metadata`) and covers the Amazon PR/FAQ working-backwards method: a mock **press release** + anticipated **FAQ** written as if the thing launched today, targeting the *structure and argument* (surfacing the "dragons"/hard questions), explicitly framing the value as the reasoning rigor, **not** polished prose. | File exists; contains the PR/FAQ structure, an FAQ/"dragons" discipline, and a reasoning-over-wordsmithing statement. |
| AC2 | `domains/pm/skills/exec-narrative/SKILL.md` exists with valid frontmatter and documents the Amazon **six-page narrative** structure (Intro / Goals / Tenets / State of the Business / Lessons Learned / Strategic Priorities) plus the **"so what?"** pressure-test, framing argument rigor over prose and calling out that prose surfaces logical gaps bullets hide. | File exists; contains all six 6-pager sections, the "so what?" test, and the prose-exposes-gaps argument. |
| AC3 | `domains/pm/agents/discovery-researcher.md` gains, in its stance and/or workflow, the three synthesis disciplines — **compare-don't-replace** (agent pass alongside the PM's, reconciled), **verbatim-quote traceability**, and **outlier-surfacing (not the modal answer)** — cross-referencing `pm-agent-doctrine`. Child #1's full startup load-list is **preserved intact** (`pm-agent-doctrine`, `opportunity-solution-trees-torres`, `discovery-interview-design`, `assumption-testing`, `continuous-discovery-cadence`, `evidence-synthesis`). | Agent body references compare-don't-replace, verbatim traceability, and outliers; all six startup loads still present. |
| AC4 | `domains/pm/skills/evidence-synthesis/SKILL.md` is **extended additively** with the Torres synthesize-then-compare discipline, outlier-surfacing, and verbatim-attribution preservation. All pre-existing content survives (the evidence pyramid, the Evidence-section shape, aggregate-count-preserve-sources, counter-evidence, five-whys, and the full anti-patterns list). | New sections present; every prior H2 heading still present (Validation guard enforces this). |
| AC5 | `domains/pm/agents/stakeholder-communicator.md` startup load-list gains `prfaq-writing` and `exec-narrative`, and its **seven existing loads are preserved** (`pm-agent-doctrine`, `outcomes-over-outputs`, `stakeholder-communication`, `release-notes-writing`, `cross-domain-graph-query`, `spec-format`, `kickoff-prompt`). | Both new skills listed under Startup; all seven prior loads still present. |
| AC6 | `domains/pm/AGENTS.md` gains a new Wave-2 subsection appended **below the `<!-- WAVE-2 ROUTES -->` marker, after all prior children's subsections** (i.e. after "Wave-2 competitive & market-grounding routes"), routing net-new exec-narrative / PR-FAQ authoring intent to `stakeholder-communicator`; and both new skills are added to the **Skills Reference**. The canonical routing table and every prior child's Wave-2 subsection are **untouched**. | New subsection present after the competitive routes; both skills in Skills Reference; no edits above the marker or inside prior subsections. |
| AC7 | No dangling references remain: `stakeholder-communication` and `stakeholder-communicator`'s pointers to `exec-narrative` now resolve to a real file, and every new cross-reference names a skill that exists on disk. | `grep` for `exec-narrative`/`prfaq-writing` resolves to existing skill dirs; no reference to a nonexistent skill file. |
| AC8 | All authoring is confined to `domains/pm/` pack source; no installed harness copy (`.claude/`, `.agents/`, `.codex/`) and no Go file is modified (tripwire `harness-changes-cover-all-targets`). | `git diff --name-only` touches only `domains/pm/**` and this spec. |

## Changes

Content-only. No Go. All paths under `domains/pm/`.

**New skill — `domains/pm/skills/prfaq-writing/SKILL.md`**
- Frontmatter: `name: prfaq-writing`, a `description` naming the Amazon PR/FAQ
  working-backwards method, `metadata.audience: stakeholder-communicator`,
  `metadata.purpose: working-backwards`.
- Body: what the PR/FAQ is (a mock press release + anticipated FAQ written *before*
  building, as if launched today); the press-release shape (headline, sub-head,
  problem, solution, customer/leader quotes as *placeholders to be sourced*, call to
  action); the FAQ discipline as the load-bearing half — surface the **dragons**
  (the hard customer and internal questions you'd rather not answer); the explicit
  frame that the deliverable is the *reasoning that surfaces the dragons*, not
  polished copy; cross-refs to `exec-narrative` (sibling format), `stakeholder-communication`
  (names the pattern, defers here), and `pm-agent-doctrine` (quotes are placeholders
  until sourced — never fabricate). Anti-patterns section.

**New skill — `domains/pm/skills/exec-narrative/SKILL.md`**
- Frontmatter: `name: exec-narrative`, `description` naming the Amazon 6-pager +
  "so what?" pressure-test, `metadata.audience: stakeholder-communicator`,
  `metadata.purpose: working-backwards`.
- Body: the six-page narrative structure section-by-section (Intro / Goals /
  Tenets / State of the Business / Lessons Learned / Strategic Priorities); the
  prose-over-bullets rule and *why* — prose forces the logical connective tissue
  that bullets let you skip, surfacing the gaps; the **"so what?"** pressure-test
  applied at the paragraph level (every claim ties to a consequence or gets cut);
  argument rigor over prose polish; when to reach for a 6-pager vs. a one-page exec
  cut; cross-refs to `stakeholder-communication` (audience cuts — the shallower
  sibling job), `prfaq-writing` (the launch-shaped sibling), `outcomes-over-outputs`,
  and `pm-agent-doctrine`. Anti-patterns section.

**Sharpen — `domains/pm/agents/discovery-researcher.md`**
- In the opening stance and/or the "Synthesize after the test" workflow step, add
  the three disciplines: compare-don't-replace (produce the synthesis alongside the
  PM's own read, framed for reconciliation), verbatim-quote traceability (every
  theme links to the exact words), outlier-surfacing (report what didn't fit, not
  the modal answer). Cross-reference `pm-agent-doctrine` doctrine 3 rather than
  restating it. Add a matching anti-pattern (e.g. "synthesis that reports only the
  modal answer" / "themes with no verbatim behind them").
- **Preserve** the six-item Startup load-list exactly.

**Extend — `domains/pm/skills/evidence-synthesis/SKILL.md`** (additive only)
- Add a "Torres synthesize-then-compare" section: the agent synthesizes first, the
  PM reads independently, they reconcile the two passes (this is doctrine 3 in the
  synthesis mechanics).
- Add outlier-surfacing guidance: the outlier is often the high-value finding;
  surface signal that breaks the pattern, don't sand it into the modal narrative.
- Reinforce verbatim-attribution preservation through the synthesize-then-compare
  loop (theme → verbatim link survives reconciliation).
- Do **not** remove or reword existing sections.

**Wire — `domains/pm/agents/stakeholder-communicator.md`**
- Add `prfaq-writing` and `exec-narrative` to the Startup load-list; preserve the
  seven existing loads and their annotations.

**Wire — `domains/pm/AGENTS.md`**
- Append a new `#### Wave-2 exec narrative & working-backwards routes` subsection
  below the marker, **after** the "Wave-2 competitive & market-grounding routes"
  subsection and before the generic "When routing, pass the user's original
  context…" closing paragraph. One route: exec narrative / PR-FAQ / "write the
  6-pager" / "working backwards" authoring → `stakeholder-communicator` (loads the
  two new skills). Note it un-dangles the `exec-narrative` forward-reference.
- Add both new skills to the **Skills Reference** (a "Wave-2 exec narrative" bullet
  under the Writing/Wave-2 groupings, matching the existing Wave-2 bullet style).

## Validation

Content-only spec — the check is that the files exist, contain the required
material, and that the additive edit to `evidence-synthesis` did not truncate prior
content. Run from repo root:

```bash
set -euo pipefail
cd /Users/developer/projects/hero-engine/repository/hero
PM=domains/pm

# AC1 — prfaq-writing exists and carries the PR/FAQ + dragons + reasoning frame
test -f "$PM/skills/prfaq-writing/SKILL.md"
grep -qi "press release" "$PM/skills/prfaq-writing/SKILL.md"
grep -qi "FAQ"           "$PM/skills/prfaq-writing/SKILL.md"
grep -qi "dragon"        "$PM/skills/prfaq-writing/SKILL.md"

# AC2 — exec-narrative exists with all six 6-pager sections + so-what
test -f "$PM/skills/exec-narrative/SKILL.md"
for s in "Goals" "Tenets" "State of the Business" "Lessons" "Strategic Priorities"; do
  grep -qi "$s" "$PM/skills/exec-narrative/SKILL.md"
done
grep -qi "so what" "$PM/skills/exec-narrative/SKILL.md"

# AC3 — discovery-researcher sharpened AND child #1 loads preserved
grep -qi "compare"  "$PM/agents/discovery-researcher.md"
grep -qi "verbatim" "$PM/agents/discovery-researcher.md"
grep -qi "outlier"  "$PM/agents/discovery-researcher.md"
for load in pm-agent-doctrine opportunity-solution-trees-torres \
            discovery-interview-design assumption-testing \
            continuous-discovery-cadence evidence-synthesis; do
  grep -q "$load" "$PM/agents/discovery-researcher.md"
done

# AC4 — evidence-synthesis EXTENDED (new material) AND prior content intact
grep -qi "synthesize-then-compare\|synthesize then compare" "$PM/skills/evidence-synthesis/SKILL.md"
grep -qi "outlier" "$PM/skills/evidence-synthesis/SKILL.md"
# "still contains prior content" guard — every pre-existing H2 must survive
for h in "## What I do" "## The evidence pyramid" \
         "## Aggregating across intakes without losing attribution" \
         "## Quoting the customer's words" \
         "## Distinguishing evidence from interpretation" \
         "## Surfacing counter-evidence honestly" \
         "## The \"five whys\" pattern when intake is vague" \
         "## Anti-patterns"; do
  grep -qF "$h" "$PM/skills/evidence-synthesis/SKILL.md" \
    || { echo "REGRESSION: evidence-synthesis lost section: $h"; exit 1; }
done

# AC5 — stakeholder-communicator gains 2 skills, keeps its 7
grep -q "prfaq-writing"  "$PM/agents/stakeholder-communicator.md"
grep -q "exec-narrative" "$PM/agents/stakeholder-communicator.md"
for load in pm-agent-doctrine outcomes-over-outputs stakeholder-communication \
            release-notes-writing cross-domain-graph-query spec-format kickoff-prompt; do
  grep -q "$load" "$PM/agents/stakeholder-communicator.md"
done

# AC6 — AGENTS.md: new subsection after competitive routes; both skills in Skills Reference
grep -q "Wave-2 exec narrative" "$PM/AGENTS.md"
grep -q "prfaq-writing"  "$PM/AGENTS.md"
grep -q "exec-narrative" "$PM/AGENTS.md"
# marker + prior-children ordering intact
grep -q "WAVE-2 ROUTES" "$PM/AGENTS.md"
grep -q "Wave-2 competitive & market-grounding routes" "$PM/AGENTS.md"

# AC7 — no dangling skill refs: every referenced new skill dir exists
test -d "$PM/skills/prfaq-writing"
test -d "$PM/skills/exec-narrative"

# AC8 — only domains/pm touched (plus this spec); no Go, no installed copies
git diff --name-only | grep -vE '^(domains/pm/|\.hero/planning/)' && \
  { echo "OUT-OF-SCOPE FILE MODIFIED"; exit 1; } || true

echo "ALL CHECKS PASSED"
```

Also run `hero check` to confirm no new dangling `[[wikilink]]` edge-intent
warnings were introduced in the PM pack.

## Boundaries

**In scope:** the two new skills, the two additive/sharpening edits, and the
wiring (stakeholder-communicator load-list + AGENTS.md route + Skills Reference).
All under `domains/pm/` pack source.

**Out of scope / do NOT:**
- No Go, no CLI, no schema. Content only.
- Do **not** author into installed harness copies (`.claude/`, `.agents/`,
  `.codex/`) — those are regenerated by `hero install` from `domains/pm/`. Tripwire:
  `harness-changes-cover-all-targets`.
- Do **not** restate `pm-agent-doctrine`'s compare-don't-replace doctrine in full —
  cross-reference it. Doctrine 3 is the home; these surfaces operationalize it.
- Do **not** duplicate `stakeholder-communication`'s audience-cut table or its "so
  what" material as the *primary* content. `exec-narrative` is the deeper artifact
  that skill already defers to; keep the boundary it drew (cuts vs. full narrative).
- Do **not** edit the canonical AGENTS.md routing table (above the marker) or any
  prior child's Wave-2 subsection. Append-only, after the last prior child.
- Do **not** remove, reorder, or reword existing `evidence-synthesis` content — the
  edit is strictly additive (Validation enforces the prior-content guard).
- No new agent, no new command, no new hero-code surface — this child backs the
  existing `stakeholder-communicator` and `discovery-researcher`; it invents no new
  routing target beyond the exec-narrative / PR-FAQ authoring intent.

## Completion Ledger

Delivered content-only under `domains/pm/`. The spec's full `## Validation` bash
block was run verbatim from repo root and printed `ALL CHECKS PASSED` (exit 0).
The verbatim AC8 `git diff` guard initially flagged three Hero harness-maintained
projection/log files (`.hero/NEXT.md`, `.hero/QUEUE.md`, `.hero/events.log`) that
the running `/drive` harness churns and that this delivery did not author; they
were temporarily set aside (`git stash`) for the verbatim run and restored intact.
No `.claude/`, `.codex/`, `.agents/`, or Go file was touched.

### Acceptance Criteria

| # | Criterion | Status | Note |
|---|---|---|---|
| AC1 | `prfaq-writing/SKILL.md` exists — PR/FAQ (press release + FAQ), dragons, reasoning-over-prose | DONE | File on disk; verbatim greps for `press release`, `FAQ`, `dragon` all pass. Body has the press-release shape, the FAQ "dragons" discipline as the load-bearing half, and an explicit "the deliverable is the reasoning… not polished prose" frame. |
| AC2 | `exec-narrative/SKILL.md` exists — all six 6-pager sections + "so what?" + prose-exposes-gaps | DONE | File on disk; verbatim greps for `Goals`, `Tenets`, `State of the Business`, `Lessons`, `Strategic Priorities`, `so what` all pass. Intro section present too; paragraph-level "so what?" table and the prose-forces-connective-tissue argument included. |
| AC3 | `discovery-researcher.md` gains compare-don't-replace + verbatim traceability + outlier-surfacing; child #1's 6 loads preserved | DONE | Greps for `compare`, `verbatim`, `outlier` pass; all six loads (`pm-agent-doctrine`, `opportunity-solution-trees-torres`, `discovery-interview-design`, `assumption-testing`, `continuous-discovery-cadence`, `evidence-synthesis`) still present. Added to stance, the synthesize step, and three new anti-patterns; cross-refs doctrine 3 rather than restating. |
| AC4 | `evidence-synthesis/SKILL.md` extended additively (synthesize-then-compare, outlier, verbatim-attribution); every prior H2 survives | DONE | Greps for `synthesize-then-compare` and `outlier` pass; the prior-content guard confirms all 8 pre-existing H2 headings survive. Three new H2 sections + two new anti-patterns inserted before `## Anti-patterns`; no existing text removed or reworded. |
| AC5 | `stakeholder-communicator.md` gains `prfaq-writing` + `exec-narrative`; its 7 loads preserved | DONE | Both new skills grep-present under Startup; all seven prior loads (`pm-agent-doctrine`, `outcomes-over-outputs`, `stakeholder-communication`, `release-notes-writing`, `cross-domain-graph-query`, `spec-format`, `kickoff-prompt`) still present. |
| AC6 | `AGENTS.md` new Wave-2 subsection appended after competitive routes; both skills in Skills Reference; table + prior subsections untouched | DONE | `Wave-2 exec narrative` subsection sits after "Wave-2 competitive & market-grounding routes" and before the "When routing…" closing paragraph; `WAVE-2 ROUTES` marker and competitive subsection intact; new Skills-Reference bullet lists both skills. Canonical table and prior children's subsections unedited (append + line-insertions only). |
| AC7 | No dangling refs — `exec-narrative`/`prfaq-writing` pointers resolve to real dirs | DONE | `test -d` for both skill dirs passes; `stakeholder-communication` and `stakeholder-communicator`'s `exec-narrative` forward-refs now resolve on disk; every new cross-ref names an existing skill. |
| AC8 | All authoring under `domains/pm/`; no installed harness copy, no Go modified | DONE | `git status` shows authored changes only under `domains/pm/`; scoped check confirms no `.claude/`/`.codex/`/`.agents/`/`*.go` touched. Verbatim block printed `ALL CHECKS PASSED` after isolating the three harness-churned `.hero/*` projection/log files noted above. |

### Changes

| # | Changes item | Status | Note |
|---|---|---|---|
| 1 | New skill `domains/pm/skills/prfaq-writing/SKILL.md` | DONE | Created (~85 lines): frontmatter (`name`, `description`, `metadata.audience: stakeholder-communicator`, `metadata.purpose: working-backwards`); press-release shape with quotes-as-placeholders, FAQ/"dragons" load-bearing half, kill-switch framing, anti-patterns, cross-refs to `exec-narrative`/`stakeholder-communication`/`pm-agent-doctrine`/`assumption-testing`/`outcomes-over-outputs`. |
| 2 | New skill `domains/pm/skills/exec-narrative/SKILL.md` | DONE | Created (~90 lines): frontmatter as specified; six sections Intro/Goals/Tenets/State of the Business/Lessons Learned/Strategic Priorities, paragraph-level "so what?" test, prose-over-bullets rationale, 6-pager-vs-one-page-cut guidance, anti-patterns, cross-refs. Defers audience-cut material to `stakeholder-communication` per the boundary. |
| 3 | Sharpen `domains/pm/agents/discovery-researcher.md` | DONE | Added the three disciplines to the opening stance and the "Synthesize after the test" step, cross-referencing `pm-agent-doctrine` doctrine 3; three matching anti-patterns added. Six-item Startup load-list preserved verbatim. |
| 4 | Extend `domains/pm/skills/evidence-synthesis/SKILL.md` (additive) | DONE | Added `## The Torres synthesize-then-compare discipline`, `## Surfacing the outlier, not just the modal answer`, `## Verbatim attribution survives reconciliation`, plus two anti-patterns. No prior section removed or reworded (guard passes). |
| 5 | Wire `domains/pm/agents/stakeholder-communicator.md` | DONE | `prfaq-writing` and `exec-narrative` inserted into Startup after `stakeholder-communication`; seven existing loads and annotations preserved. Un-dangles the child-#9 forward-ref. |
| 6 | Wire `domains/pm/AGENTS.md` (route + Skills Reference) | DONE | New `#### Wave-2 exec narrative & working-backwards routes` subsection appended after the competitive subsection with two routes to `stakeholder-communicator`; new Skills-Reference bullet added. Canonical table and prior subsections untouched. |

### Exercise-the-feature check

- [x] Ran the spec's full `## Validation` bash block verbatim from repo root → `ALL CHECKS PASSED` (exit 0).
- [x] AC4 prior-content guard exercised: every pre-existing `evidence-synthesis` H2 confirmed present after the additive edit.
- [x] AC3/AC5 load-preservation exercised: all six discovery-researcher loads and all seven stakeholder-communicator loads confirmed present.
- [x] AC6 ordering exercised: new AGENTS.md subsection confirmed below the `WAVE-2 ROUTES` marker and after the competitive subsection.
- [x] AC7 no-dangling exercised: both new skill dirs exist; forward-refs from `stakeholder-communication`/`stakeholder-communicator` resolve.
- [x] Ran `hero check` — the one wikilink warning is a false positive from the literal `` `[[wikilink]]` `` string in this spec's own Validation prose (`.hero/planning/`), not from any `domains/pm/` file; my authored content introduced zero `[[ ]]`.
- [ ] Runtime agent-load behavior (a live `/standup` or exec-cut invocation loading the two new skills) not exercised — these are content/skill files consumed by the harness at agent-startup; there is no code path to run in this session. Verified structurally (frontmatter, load-list wiring, resolvable cross-refs) instead.

### Excellence Bar self-check

- Both new skills match the shipped Wave-2 skill shape (frontmatter + `## What I do` / `## When to use me` / body / `## Anti-patterns` / `## Cross-references`), land in the ~85–90 line band, and carry worked examples (the "so what?" fail/survive table, the press-release + FAQ split) rather than stubs.
- The differentiation frame from the audit is honored throughout: every section foregrounds argument rigor over wordsmithing (PR/FAQ = "the reasoning that surfaces the dragons"; 6-pager = "prose forces the connective tissue bullets skip"), and both cross-reference `pm-agent-doctrine` doctrine 3 rather than restating it.
- Boundaries held: no Go, no installed-harness copies, no canonical-table edits, no `evidence-synthesis` truncation, no duplication of `stakeholder-communication`'s audience-cut material — the deeper-artifact boundary the existing skill drew is preserved.
- Would I show this to a senior engineer who cares? Yes — the edits are surgical, additive where required, the forward-references now resolve, and the full validation is green.
