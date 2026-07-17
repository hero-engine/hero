# Delivery audit — adversarial-critics-bundle

**Audited:** working tree (uncommitted) vs. `HEAD` — `git diff HEAD` + untracked files
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria
- [✓] AC-1 roadmap-reviewer is an adversarial drift critic — names outcome-vs-output drift, ~60/30/10, stale-item flagging, honest-roadmap review (`roadmap-reviewer.md:17,40-44,89-94`); not a passive gate
- [✓] AC-2 roadmap-reviewer Startup loads `pm-agent-doctrine` + `outcomes-over-outputs`; both dirs resolve (`roadmap-reviewer.md:24-25`, validation PASS)
- [✓] AC-3 `skills/outcome-drift/SKILL.md` valid frontmatter `name: outcome-drift` + ratio tally + 4-type stale taxonomy (`outcome-drift/SKILL.md:2,24-46`)
- [✓] AC-4 prioritization-challenger is an anti-gaming critic — RICE/ICE/WSJF challenge, Confidence→50%, inflated Reach/Impact (`prioritization-challenger.md:17,40-44`)
- [✓] AC-5 prioritization-challenger Startup loads `pm-agent-doctrine` (resolves) (`prioritization-challenger.md:24`)
- [✓] AC-6 `skills/evidence-forcing/SKILL.md` — default-to-neutral, inflated-input detection, confidence-pumping, show-the-math (`evidence-forcing/SKILL.md:24-66`)
- [✓] AC-7 pm-reviewer sharpened to adversarial doc critic (premortem + "5 reasons") WHILE preserving both child-#1 Startup loads — Startup block unchanged in diff, both loads present (`pm-reviewer.md:24-25` Startup; diff adds Workflow step 7 CPO pass + output block)
- [✓] AC-8 experiment-readout-reviewer — SRM, no peeking, guardrails, multiple-comparisons, practical-vs-statistical significance; loads `pm-agent-doctrine` (`experiment-readout-reviewer.md:24,42-47`)
- [✓] AC-9 forward-references `experiment-design`/pre-registered brief in a `## Forward dependency` prose section (`experiment-readout-reviewer.md:49-58`) and explicitly does NOT load it in Startup (`:28`); `skills/experiment-design/` does not exist (validation PASS)
- [✓] AC-10 four Wave-2 critic routes below the marker (`AGENTS.md` diff hunk @ line 65-79)
- [✓] AC-11 `outcome-drift` + `evidence-forcing` in Skills Reference (`AGENTS.md` diff hunk @ line 188)
- [✓] AC-12 canonical routing table above marker (line 62) byte-unchanged — full `git diff` of AGENTS.md is additions only, zero deletions; first inserted hunk starts at line 64, below the marker
- [✓] AC-13 all edits confined to `domains/pm/` + `.hero/` bookkeeping; no `.claude/`, `.codex/`, `.agents/`, no Go
- [✓] AC-14 every Startup-loaded skill across the 4 critics resolves to a `SKILL.md` dir (validation PASS: pm-agent-doctrine, outcomes-over-outputs, outcome-drift, risk-surfacing, evidence-forcing, prioritization-frameworks, assumption-testing)

## Changes
- [✓] `agents/roadmap-reviewer.md` (new) — drift critic; loads doctrine + outcomes-over-outputs + outcome-drift + risk-surfacing
- [✓] `skills/outcome-drift/SKILL.md` (new) — ratio tally vs ~60/30/10 + 4-type stale taxonomy table
- [✓] `agents/prioritization-challenger.md` (new) — anti-gaming input critic; loads doctrine + evidence-forcing + prioritization-frameworks
- [✓] `skills/evidence-forcing/SKILL.md` (new) — default-to-neutral rule + worked recompute + confidence-pumping detection
- [✓] `agents/pm-reviewer.md` (modified) — CPO adversarial pass added as first-class Workflow step 7; child-#1 loads/permissions/delegation preserved
- [✓] `agents/experiment-readout-reviewer.md` (new) — adversarial readout critic; graceful-degrade forward-dep prose for child #7
- [✓] `domains/pm/AGENTS.md` (modified) — Wave-2 routes (4) + agent roster + Skills Reference lines; additions-only

## Substance check (the differentiation thesis)
All four are genuinely adversarial, corpus-grounded, and decision-gated — not passive reviewers dressed up:
- **prioritization-challenger** — forces named evidence for every soft input; defaults unsupported Confidence to 50% and *recomputes the score to show the swing* (`:40-42`); hunts confidence-pumping across revisions (`:44`); anti-pattern "objections with no corpus number behind them" (`:89`). Real teeth.
- **roadmap-reviewer** — computes the outcome/output/input ratio tally against ~60/30/10 with localization (`:40`), 4-type stale taxonomy with cited observables (`:42`), honest-roadmap review (`:43`); anti-pattern "ungrounded drift claims" (`:92`). Real drift computation.
- **experiment-readout-reviewer** — full SRM / peeking / guardrail / multiple-comparisons / significance / novelty checklist grounded in the readout's own numbers (`:42-47`); do-not-act verdict on SRM/peek/guardrail; anti-pattern "objections not grounded in the readout's own numbers" (`:103`). Real result-laundering defense.
- **pm-reviewer (sharpened)** — adds premortem (routed through `risk-surfacing`) + "5 reasons this won't work" inversion as first-class Workflow step 7 with a matching output block, WITHOUT dropping child #1's unconditional doctrine loads (Startup block untouched in the diff). Two-part bar (schema gate AND CPO critique). Real premortem teeth, doctrine loads intact.

Each names its inversion move explicitly, makes corpus-grounding a hard rule, lists "objections with no corpus anchor" as an anti-pattern, and is suggest-verdict (never auto-edits/reorders/ships).

## Open items
None. No PARTIAL / SKIPPED / BLOCKED rows.

## Audit notes
- Full `## Validation` bash block ran verbatim from repo root: **34/34 PASS, exit 0**.
- AC-12 verified hard: `git diff HEAD -- domains/pm/AGENTS.md` contains zero deletion lines (only `---` file header); marker is at line 62, first insertion hunk at line 64 (below it). Canonical table intact.
- AC-9 verified: `experiment-design` appears only in prose (`## Forward dependency` + one anti-pattern), never in a Startup load list; `domains/pm/skills/experiment-design/` does not exist — no dangling reference introduced.
- Scope: the spec deliverable is confined precisely to `domains/pm/`. Working tree also carries `.hero/` drive-run bookkeeping (`.hero/QUEUE.md`, `.hero/events.log`, `.hero/drive/pm-pack-completion.json`, `.hero/drive/trust/`) and the spec's own flat-file→directory conversion (`adversarial-critics-bundle.md` deleted, `adversarial-critics-bundle/` dir added). All under `.hero/` — permitted by the AC-13 tripwire and normal drive-session state, not scope drift in the deliverable. No Go, no installed harness dirs (`.claude/`/`.codex/`/`.agents/`), no sibling-child spec files touched.
