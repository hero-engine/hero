---
title: "Adversarial Critics Bundle — Drift, Prioritization, Doc, and Readout Critics"
slug: adversarial-critics-bundle
type: feature
status: completed
domain: pm
priority: high
size: large
created: 2026-07-17
tags: [pm, critics, differentiation, wave-2]
relations:
  - target: pm-pack-completion
    kind: parent
  - target: pm-doctrine-and-skill-backfill
    kind: depends-on
  - target: pm-doctrine-and-skill-backfill
    kind: conflicts-with
  - target: experiment-stage-and-metric-rca
    kind: conflicts-with
completed_at: 2026-07-17T20:56:55Z
---

# Adversarial Critics Bundle — Drift, Prioritization, Doc, and Readout Critics

## Goal

Author the four **adversarial critic agents** that are the PM pack's core
differentiation — critics over generators. When delivery completes, four files
exist and stand up under a cold audit: a roadmap **drift critic**
(`roadmap-reviewer`), an anti-gaming **prioritization critic**
(`prioritization-challenger`), a sharpened adversarial **doc critic**
(`pm-reviewer`), and an adversarial **experiment-readout critic**
(`experiment-readout-reviewer`). Each critic loads `pm-agent-doctrine` so its
stance is corpus-grounded, decision-gated, and compare-don't-replace. Two new
skills back the net-new critics (`outcome-drift`, `evidence-forcing`), and the
`domains/pm/AGENTS.md` Wave-2 region routes to them. All authoring is in
`domains/pm/` pack source only; install propagates to all six targets.

## Kickoff

Deliver `adversarial-critics-bundle` (Wave-2 child #4 of `pm-pack-completion`).
Author four adversarial PM critic agents in `domains/pm/agents/`
(`roadmap-reviewer`, `prioritization-challenger`, sharpened `pm-reviewer`,
`experiment-readout-reviewer`) plus two skills (`outcome-drift`,
`evidence-forcing`). Every critic loads `pm-agent-doctrine`; the drift critic
also loads `outcomes-over-outputs`. Append net-new routes to the AGENTS.md
Wave-2 region **below** the marker only — leave the canonical table byte-for-byte
unchanged. Content-only, no Go. Do not undo child #1's load-list additions to
`pm-reviewer`. Run the `## Validation` block before flipping status.

## Problem

Wave-1 (`pm-doctrine-and-skill-backfill`) landed the doctrine spine —
`pm-agent-doctrine`, `outcomes-over-outputs`, `risk-surfacing`,
`assumption-testing` — and added the doctrine loads to the shipped authoring and
review agents. But the PM pack audit (`pm-pack-audit-2026-07.md`) is emphatic
that the **differentiated, trusted** value of an AI PM pack is not generation:

> Generators are commoditized and *distrusted*. The underserved, high-leverage
> capabilities are the "keep it honest" ones: roadmap-drift detection,
> anti-gaming prioritization critique, adversarial experiment readouts, doc
> critique as a CPO would.

Today the pack has one review agent (`pm-reviewer`), framed by the design
(`agent-pack-design.md` §C.7) as a **passive quality gate** — "has this artifact
earned its next transition?" That is necessary but not the differentiator. The
audit's Wave-2 reframe is explicit: **elevate the critics into first-class
adversarial agents.** A passive gate asks "is this good enough to advance?" An
adversarial critic asks "what's wrong with this, what's the strongest case
against it, and what evidence is missing?" — and it grounds every objection in
the team's own corpus so the critique is checkable rather than contrarian.

Four capabilities are missing or under-sharpened:

1. **Roadmap-drift critic** — no agent audits a whole roadmap for
   output-vs-outcome drift, the ~60/30/10 ratio, or stale items. `roadmap-curator`
   reconciles delivery *state*; it does not critique *framing*. The
   `outcomes-over-outputs` skill already carries the drift lens (it was authored
   in Wave-1 anticipating a "drift critic" loader) but nothing loads it as a
   critic. (Note: an engineering-domain `roadmap-reviewer` exists as a separate
   pack's roadmap-shape triage agent; this is the **pm-domain** critic — a
   distinct file under `domains/pm/agents/`.)
2. **Anti-gaming prioritization critic** — nothing challenges RICE/ICE/WSJF
   inputs. `prioritization-strategist` *applies* frameworks; no agent forces the
   inputs to be defensible ("a Confidence rating needs named evidence or it's
   50%", inflated Reach/Impact detection). The `prioritization-frameworks` skill
   names the abuse modes but no agent hunts them.
3. **Adversarial doc critic** — `pm-reviewer` is a passive gate. The audit wants
   "review as CPO": premortem, "5 reasons this won't work," the strongest case
   against the doc — not just schema compliance.
4. **Adversarial experiment-readout critic** — no agent critiques an experiment
   readout for the classic result-laundering failures (sample-ratio mismatch,
   peeking / early stopping, guardrail regressions ignored, multiple-comparisons
   inflation, statistical-vs-practical significance conflation). This is the
   whole trusted core of experimentation tooling and the pack has none of it.

This spec authors all four as adversarial critics with a consistent doctrine
stance, so the discipline is authored once across the bundle rather than
drifting per agent.

## Approach

**Shape.** Every critic mirrors the shipped agent shape (YAML frontmatter with
`name`/`description`/`mode: subagent`/`temperature`/`color`/`permission`;
"You are a …" persona opener; `## Startup` skill-load list; numbered `##
Workflow`; `## Produces`; `## Output format`; `## Anti-patterns`; closing
discipline). Use `pm-reviewer.md` and `product-strategist.md` as the reference
templates already on disk.

**Doctrine is the shared spine.** Every critic's `## Startup` loads
`pm-agent-doctrine` **unconditionally** — that is where the adversarial,
corpus-grounded, decision-gated, compare-don't-replace stance lives. A critic's
objections are only trusted if they are grounded (doctrine 1): "this Reach is
inflated" must cite the corpus number it contradicts, not free-associate. And a
critic *suggests, never decides* (doctrine 2): it produces findings and a
verdict, never auto-edits the artifact or auto-reorders a board. This keeps the
critics from becoming contrarian noise — an adversarial critic that can't ground
its objection is exactly the distrusted free-association the doctrine forbids.

**Critic vs. author boundary.** These agents review; they do not author. They
carry `task: "*": deny` (no delegation) and write findings into a `## Review` /
`## Critique` section on the spec — they do not rewrite the artifact. That is the
authoring agent's job; the critic's verdict routes the PM back to the author.

**Adversarial framing, concretely.** The move that distinguishes an adversarial
critic from a passive gate is **inversion**: instead of checking a checklist,
the critic argues the *strongest case against* the artifact and then tests
whether the artifact survives it. Premortem ("assume it failed — why?"),
"5 reasons this won't work," "what would a skeptical CPO kill this for in
review?" Each critic file names its inversion move explicitly.

**Reviewers, not gates, in the routing.** The routes append to the AGENTS.md
Wave-2 region below the marker. The design (`agent-pack-design.md` §F) ships no
`/review` slash command in pm, so — consistent with the existing pm-reviewer
row — the routes point at **invoking the agent directly**, not a command shim.

**Forward dependency on child #7 (soft coupling).** `experiment-readout-reviewer`
critiques an experiment **readout** against the **pre-registered brief** that
child #7 (`experiment-stage-and-metric-rca`) will define via its
`experiment-design` skill. Child #7 is not yet delivered. Therefore this spec:
(a) has the reviewer **reference** the pre-registered brief format as a forward
dependency in a dedicated `## Forward dependency` prose section, stating it reads
the pre-registered brief and critiques the readout against it; (b) does **not**
put `experiment-design` in the reviewer's unconditional Startup load list (it
would be a dangling reference until #7 lands, and this spec's own no-dangling-refs
gate would fail); (c) does **not** author the `experiment-design` skill here —
that is #7's scope. The reviewer degrades gracefully: with no brief on file it
critiques against the general pre-registration discipline it carries inline and
flags the missing brief as its first finding.

## Acceptance Criteria

- **AC-1:** THE SYSTEM SHALL provide `domains/pm/agents/roadmap-reviewer.md`
  authored as an adversarial drift critic — its body names outcome-vs-output
  drift detection, the ~60/30/10 outcome/output/input ratio check, stale-item
  flagging, and honest-roadmap review — not a passive gate.
- **AC-2:** WHEN `roadmap-reviewer.md` runs its Startup THE SYSTEM SHALL load
  both `pm-agent-doctrine` and `outcomes-over-outputs`, and both skill
  directories SHALL resolve to an existing `SKILL.md`.
- **AC-3:** THE SYSTEM SHALL provide `domains/pm/skills/outcome-drift/SKILL.md`
  with valid frontmatter (`name: outcome-drift`, `description`, `metadata`) whose
  body carries the drift-signal method (ratio tally, stale-item taxonomy) that
  `roadmap-reviewer` applies.
- **AC-4:** THE SYSTEM SHALL provide
  `domains/pm/agents/prioritization-challenger.md` authored as an anti-gaming
  critic — its body names challenging RICE/ICE/WSJF inputs, the rule "a Confidence
  rating needs named evidence or it defaults to 50%", and detection of inflated
  Reach/Impact.
- **AC-5:** WHEN `prioritization-challenger.md` runs its Startup THE SYSTEM SHALL
  load `pm-agent-doctrine`, and the load SHALL resolve to an existing `SKILL.md`.
- **AC-6:** THE SYSTEM SHALL provide
  `domains/pm/skills/evidence-forcing/SKILL.md` with valid frontmatter whose body
  carries the evidence-forcing discipline (unnamed-evidence-defaults-to-neutral,
  inflated-input detection, show-the-math) that `prioritization-challenger`
  applies.
- **AC-7:** THE SYSTEM SHALL sharpen `domains/pm/agents/pm-reviewer.md` into an
  adversarial doc critic — its body adds a "review as CPO" premortem / "5 reasons
  this won't work" inversion stance — WHILE preserving child #1's unconditional
  Startup loads of `pm-agent-doctrine` and `outcomes-over-outputs` (neither is
  removed).
- **AC-8:** THE SYSTEM SHALL provide
  `domains/pm/agents/experiment-readout-reviewer.md` authored as an adversarial
  readout critic — its body names SRM (sample-ratio-mismatch) check, no
  early-stopping / peeking, guardrail-regression check, multiple-comparisons
  correction, and practical-vs-statistical significance — and loads
  `pm-agent-doctrine` in Startup (resolving).
- **AC-9:** THE SYSTEM SHALL have `experiment-readout-reviewer.md` reference the
  pre-registered experiment brief and the `experiment-design` skill as a forward
  dependency on child #7 (`experiment-stage-and-metric-rca`) in a prose section,
  WITHOUT listing `experiment-design` in any unconditional Startup skill-load
  list (so no dangling reference is introduced).
- **AC-10:** WHERE the `domains/pm/AGENTS.md` Wave-2 region below the marker line
  is concerned THE SYSTEM SHALL contain four net-new routes — reviewing a roadmap
  for drift → `roadmap-reviewer`; challenging a prioritization →
  `prioritization-challenger`; adversarial PRD/spec review → `pm-reviewer`
  (pm-critic mode); reviewing an experiment readout → `experiment-readout-reviewer`.
- **AC-11:** THE SYSTEM SHALL add `outcome-drift` and `evidence-forcing` to the
  `domains/pm/AGENTS.md` Skills Reference.
- **AC-12:** IF any content above the Wave-2 marker line in `domains/pm/AGENTS.md`
  is altered THEN the change is rejected — the canonical routing table above the
  marker SHALL be byte-for-byte unchanged (the diff shows additions only, zero
  deletions).
- **AC-13:** THE SYSTEM SHALL confine all edits to `domains/pm/` pack source —
  no edits to any per-target or installed harness directory (`.claude/`,
  `.codex/`, `.agents/`, or any other install target) — satisfying the
  `harness-changes-cover-all-targets` tripwire.
- **AC-14:** THE SYSTEM SHALL introduce no dangling skill reference — every skill
  named in any critic's unconditional Startup load list SHALL resolve to an
  existing `domains/pm/skills/<name>/SKILL.md`.

## Changes

1. **`domains/pm/agents/roadmap-reviewer.md`** (new) — pm-domain roadmap **drift
   critic**. Model the frontmatter on `pm-reviewer.md`
   (`mode: subagent`, `temperature: 0.1`, `color: warning`, `permission.edit:
   allow`, `task."*": deny`, `skill."*": allow`, `webfetch: allow`).
   - Persona opener: "You are a senior roadmap drift critic." State the stance:
     you do not reconcile delivery state (that is `roadmap-curator`) — you
     interrogate *framing*: is the roadmap a set of outcome bets or a build queue
     wearing quarterly labels?
   - `## Startup` (unconditional): load `pm-agent-doctrine` (adversarial stance,
     ground every drift finding in the corpus) and `outcomes-over-outputs` (the
     outcome ladder + 60/30/10 ratio — the lens for the whole pass). Also load
     `outcome-drift` (this spec, change 2) and `risk-surfacing` (for aging-bet
     findings).
   - `## Workflow`: (a) tally every roadmap item by ladder rung
     (input/output/outcome/impact) per `outcomes-over-outputs`; compute the
     realized ratio and compare against ~60/30/10; (b) flag output-framed **bets**
     — items that can only be *shipped*, not *measured* — and demand the outcome
     each is betting to move; (c) flag **stale items** per the `outcome-drift`
     taxonomy (no movement in N cycles, shipped-but-still-active "lying-shipped"
     items, "Later" items older than the planning horizon); (d) run the
     honest-roadmap review — "what does this roadmap claim that reality
     contradicts?" reading live delivery state where available.
   - `## Produces`: a `## Roadmap Critique` section (or `.hero/knowledge/` note
     for a whole-board pass) with a drift verdict, the ratio tally, and per-item
     findings — each naming the specific reframe or action. Decision-gated:
     surfaces `needs-attention` recommendations; never auto-reassigns horizons or
     auto-drops items (doctrine 2).
   - `## Anti-patterns`: demanding every output be reframed (maintenance/compliance
     work is legitimately output-shaped); contrarian drift findings with no corpus
     grounding; conflating state reconciliation with framing critique.

2. **`domains/pm/skills/outcome-drift/SKILL.md`** (new) — the drift-detection
   method `roadmap-reviewer` applies. Frontmatter: `name: outcome-drift`,
   `description` (one sentence on detecting roadmap drift toward output-framing
   and staleness), `metadata.audience` (`roadmap-reviewer`, and the deferred
   `stale-roadmap-scrubber`), `metadata.purpose: curation`.
   - Body sections: **What I do** (turn a roadmap into a drift signal — ratio
     tally + stale taxonomy); **When to use me** (whole-roadmap critique passes);
     **The ratio tally** (how to bucket items by rung and read the realized
     input/output/outcome shape against ~60/30/10 — this operationalizes
     `outcomes-over-outputs`, cross-reference it, don't restate the ladder);
     **The stale-item taxonomy** (no-movement, lying-shipped, over-horizon,
     orphan-output-with-no-outcome — each with the observable that triggers it and
     the recommended action: refresh / drop-with-reason / archive / re-hang under
     an outcome); **Anti-patterns** (drift theater — flagging every output as
     drift; staleness by calendar alone without checking the graph;
     recommendations with no action). Cross-reference `outcomes-over-outputs`,
     `roadmap-framing`, `risk-surfacing`, `pm-agent-doctrine`.

3. **`domains/pm/agents/prioritization-challenger.md`** (new) — anti-gaming
   **prioritization critic**. Frontmatter modeled on `pm-reviewer.md`.
   - Persona opener: "You are a senior prioritization challenger." Stance: you do
     not rank (that is `prioritization-strategist`) — you **stress-test the
     inputs** of an existing ranking so a soft score can't masquerade as data.
   - `## Startup` (unconditional): load `pm-agent-doctrine` (the anti-gaming
     corollary of doctrine 2 lives here — "agents must not quietly tune inputs to
     steer a decision"; the critic hunts exactly that), `evidence-forcing` (this
     spec, change 4), and `prioritization-frameworks` (the framework mechanics +
     catalogued abuse modes it checks against).
   - `## Workflow`: for each scored item, (a) demand a named-evidence citation for
     every soft input — **a Confidence rating with no named evidence defaults to
     50%**, and the critic recomputes the score at the defaulted value to show the
     ranking's sensitivity; (b) detect **inflated Reach/Impact** — a Reach that
     exceeds the addressable segment size in the corpus, an Impact rating with no
     outcome tie; (c) detect **confidence-pumping** — the same pet item whose
     Confidence keeps rising across revisions with no new evidence; (d) show the
     recomputed math so the team can challenge it.
   - `## Produces`: a `## Prioritization Critique` section — per-input findings,
     the recomputed/ defaulted score, and a verdict (defensible / soft-inputs /
     gamed). Decision-gated: it proposes the corrected ranking as a suggestion,
     never rewrites the board (doctrine 2).
   - `## Anti-patterns`: challenging the *output order* rather than the *inputs*
     (the human owns the order; the critic owns input integrity); treating a
     framework score as truth to defend rather than a claim to interrogate;
     objections with no corpus number behind them.

4. **`domains/pm/skills/evidence-forcing/SKILL.md`** (new) — the evidence-forcing
   discipline `prioritization-challenger` applies. Frontmatter:
   `name: evidence-forcing`, `description` (one sentence: force every
   prioritization input to name its evidence or default to neutral),
   `metadata.audience` (`prioritization-challenger`, and the deferred
   `prioritization-strategist` critic-mode), `metadata.purpose: framework-guidance`.
   - Body sections: **What I do** (make soft inputs honest — the line between
     RICE-as-data and RICE-theater); **When to use me** (challenging any framework
     score); **The default-to-neutral rule** ("a Confidence with no named evidence
     is 50%, not the optimistic value"; the same principle for any unsupported
     multiplier — recompute at the neutral default and show the delta); **Inflated-
     input detection** (Reach vs. addressable-segment size from the corpus, Impact
     with no outcome tie, Effort that ignores known dependencies); **Confidence-
     pumping detection** (track an input across revisions; a rising Confidence with
     no new evidence is the tell); **Show-the-math discipline** (every challenge
     re-states the score at defensible inputs so the team can see the swing).
     **Anti-patterns**: challenging order not inputs; demanding certainty (the
     goal is *named evidence or an honest default*, not perfect data); black-box
     "this feels inflated" with no corpus anchor. Cross-reference
     `prioritization-frameworks`, `pm-agent-doctrine`, `outcomes-over-outputs`,
     `evidence-synthesis`.

5. **`domains/pm/agents/pm-reviewer.md`** (sharpen in place — do NOT rewrite from
   scratch, do NOT touch the Wave-1 Startup loads). Extend the existing passive
   gate into an adversarial doc critic:
   - **Preserve exactly** the two unconditional Startup loads child #1 added:
     `pm-agent-doctrine` and `outcomes-over-outputs`. Do not remove or reorder
     them out of the load list.
   - Add an adversarial stance to the persona/opening and Workflow: alongside the
     existing schema/principle gate, add a **"review as a skeptical CPO would"**
     pass — run a **premortem** on the artifact (assume the bet shipped and
     failed; enumerate why) and force a **"5 reasons this won't work"** listing
     before the verdict. Frame it as inversion: the artifact must survive the
     strongest case against it, not merely pass a checklist. Cross-reference
     `risk-surfacing`'s premortem procedure (already conditionally loaded for
     PRDs/initiatives) and make the premortem a first-class step, not an optional
     extra.
   - Keep the passive-gate function intact (the pre-owner-flip and pre-promotion
     gates are load-bearing) — the adversarial pass is **additive**: the doc must
     both earn its transition *and* survive the CPO critique. Update the
     `## Output format` verdict block to carry the "reasons this won't work" list
     and the premortem findings.
   - Do not change the frontmatter permission map or the delegation rules.

6. **`domains/pm/agents/experiment-readout-reviewer.md`** (new) — adversarial
   **experiment-readout critic**. Frontmatter modeled on `pm-reviewer.md`.
   - Persona opener: "You are a senior experiment-readout critic." Stance: an
     experiment readout is where results get laundered — you argue the strongest
     case that the reported result is a false positive before anyone acts on it.
   - `## Startup` (unconditional): load `pm-agent-doctrine` (ground every objection
     — cite the readout's own numbers), `risk-surfacing`, and `assumption-testing`
     (pre-registration discipline — the reviewer holds the readout to the
     pre-registered brief). Do **NOT** load `experiment-design` here (forward
     dependency; see below).
   - `## Workflow` — the adversarial readout checklist: (a) **SRM check** — does
     the actual allocation match the intended split; a sample-ratio mismatch
     invalidates the whole readout; (b) **no early stopping / peeking** — was the
     stop time pre-registered, or did the team stop when it looked significant;
     (c) **guardrail regressions** — did any protected metric (latency, error
     rate, revenue, retention) move the wrong way while the primary "won"; (d)
     **multiple comparisons** — how many metrics/segments were tested, and was the
     significance threshold corrected; a "win" found by slicing is a false
     positive; (e) **practical vs. statistical significance** — is the effect size
     large enough to matter, or merely p<0.05 on a trivial delta; (f)
     **novelty/primacy and duration** — did the effect hold past the novelty
     window.
   - `## Forward dependency` (prose section): this reviewer critiques the readout
     against the **pre-registered experiment brief** — the artifact defined by the
     `experiment-design` skill in child #7 (`experiment-stage-and-metric-rca`),
     **not yet delivered**. When a pre-registered brief exists, the reviewer reads
     it and holds the readout to it (registered primary metric, registered MDE,
     registered stop rule). When no brief exists yet, the reviewer critiques
     against the general pre-registration discipline it carries inline and flags
     the missing brief as its first finding. State plainly that `experiment-design`
     is a forward dependency and is intentionally **not** in the Startup load list
     until #7 lands.
   - `## Produces`: an `## Experiment Readout Critique` section — per-check
     findings, an overall verdict (trustworthy / caveated / do-not-act), and the
     specific re-analysis or re-run needed to resolve each red flag. Decision-gated
     (doctrine 2): it recommends; the team decides whether to ship the variant.
   - `## Anti-patterns`: accepting a headline lift without checking SRM/guardrails;
     treating p<0.05 as sufficient; flagging methodology nits with no bearing on
     the decision; objections not grounded in the readout's own reported numbers.

7. **`domains/pm/AGENTS.md`** — append to the **Wave-2 region only** (below the
   `<!-- WAVE-2 ROUTES … -->` marker on line 62; the region already carries the
   "Children #4 … append net-new agent routes BELOW this line only" instruction).
   **Do not edit any line above the marker** — the canonical routing table is
   owned by child #1 and must stay byte-for-byte unchanged.
   - Add a Wave-2 routes subsection (a small markdown table matching the canonical
     table's `User intent | … | shipped surface` columns) with four rows:
     - "Review a roadmap for drift, outcome-vs-output check, ~60/30/10, stale
       items" → invoke `roadmap-reviewer` directly (agent) — no `/review` command
       ships in pm.
     - "Challenge a prioritization, is this RICE/ICE/WSJF gamed, are the inputs
       defensible" → invoke `prioritization-challenger` directly (agent).
     - "Adversarial PRD/spec review, review as a CPO, premortem, 5 reasons this
       won't work" → invoke `pm-reviewer` directly (pm-critic mode) — sharpens the
       existing reviewer row's agent, no new command.
     - "Review an experiment readout, is this result real, SRM / peeking /
       guardrails" → invoke `experiment-readout-reviewer` directly (agent).
   - In the **Skills Reference** section, add `outcome-drift` and
     `evidence-forcing`. Place them under an appropriate existing grouping (e.g.
     append `outcome-drift` to the Curation line and `evidence-forcing` to the
     Frameworks line), or add a short "Wave-2 critics" grouping — either is
     acceptable as long as both skills appear and the line is below or within a
     section the Wave-2 region owns. Prefer extending the existing Skills Reference
     lists in place since that section sits below the marker's ownership note only
     if it does; if the Skills Reference is above the marker, add the two skill
     names to their existing grouping lines (this is an addition to a list, not an
     edit to the canonical *routing table* the byte-unchanged constraint protects —
     AC-12 protects the routing table region above the marker; the Skills Reference
     list further down the file may receive the two additions). **Confirm with a
     diff that no existing routing-table line changed.**

## Validation

A cold auditor runs this from the repo root
(`/Users/developer/projects/hero-engine/repository/hero`). All checks must pass.

```bash
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
PASS=1; ok(){ echo "PASS: $1"; }; bad(){ echo "FAIL: $1"; PASS=0; }

AG=domains/pm/agents
SK=domains/pm/skills
AGENTS=domains/pm/AGENTS.md

# --- AC-1/AC-4/AC-8: the three net-new critic files exist ---
for f in roadmap-reviewer prioritization-challenger experiment-readout-reviewer; do
  [ -f "$AG/$f.md" ] && ok "$f.md exists" || bad "$AG/$f.md missing"
done

# --- AC-3/AC-6: the two new skills exist with SKILL.md ---
for s in outcome-drift evidence-forcing; do
  [ -f "$SK/$s/SKILL.md" ] && ok "$s SKILL.md exists" || bad "$SK/$s/SKILL.md missing"
  grep -q "^name: $s" "$SK/$s/SKILL.md" && ok "$s frontmatter name" || bad "$s frontmatter name"
done

# --- AC-2/AC-5/AC-8: every critic loads pm-agent-doctrine ---
for f in roadmap-reviewer prioritization-challenger pm-reviewer experiment-readout-reviewer; do
  grep -q "pm-agent-doctrine" "$AG/$f.md" && ok "$f loads pm-agent-doctrine" || bad "$f missing pm-agent-doctrine"
done

# --- AC-2: drift critic also loads outcomes-over-outputs ---
grep -q "outcomes-over-outputs" "$AG/roadmap-reviewer.md" && ok "roadmap-reviewer loads outcomes-over-outputs" || bad "roadmap-reviewer missing outcomes-over-outputs"

# --- AC-7: pm-reviewer keeps BOTH child-#1 doctrine loads (not undone) ---
grep -q "pm-agent-doctrine" "$AG/pm-reviewer.md" && grep -q "outcomes-over-outputs" "$AG/pm-reviewer.md" \
  && ok "pm-reviewer preserves child-#1 loads" || bad "pm-reviewer dropped a child-#1 load"

# --- AC-1: roadmap-reviewer carries the drift-critic content ---
grep -qi "60/30/10" "$AG/roadmap-reviewer.md" && grep -qiE "stale" "$AG/roadmap-reviewer.md" \
  && ok "roadmap-reviewer has drift-critic content" || bad "roadmap-reviewer missing drift content"

# --- AC-4: prioritization-challenger carries anti-gaming content ---
grep -qiE "RICE|ICE|WSJF" "$AG/prioritization-challenger.md" \
  && grep -qiE "confidence" "$AG/prioritization-challenger.md" \
  && ok "prioritization-challenger has anti-gaming content" || bad "prioritization-challenger missing anti-gaming content"

# --- AC-7: pm-reviewer has adversarial CPO/premortem stance ---
grep -qiE "premortem|won't work|CPO" "$AG/pm-reviewer.md" \
  && ok "pm-reviewer sharpened to adversarial" || bad "pm-reviewer not sharpened"

# --- AC-8: experiment-readout-reviewer adversarial checklist ---
grep -qiE "SRM|sample.ratio" "$AG/experiment-readout-reviewer.md" \
  && grep -qiE "early.stopping|peek" "$AG/experiment-readout-reviewer.md" \
  && grep -qiE "guardrail" "$AG/experiment-readout-reviewer.md" \
  && ok "experiment-readout-reviewer has adversarial checklist" || bad "experiment-readout-reviewer missing checklist"

# --- AC-9: forward dep referenced but NOT hard-loaded ---
grep -qi "experiment-design" "$AG/experiment-readout-reviewer.md" \
  && ok "experiment-readout-reviewer references experiment-design (forward dep)" \
  || bad "experiment-readout-reviewer missing forward-dep reference"
[ ! -d "$SK/experiment-design" ] && ok "experiment-design skill not authored here (correct — child #7)" \
  || bad "experiment-design skill authored here — out of scope"

# --- AC-14: no dangling skill refs — every Startup-loaded skill resolves ---
# For each critic, the skills it names in its load list must exist as dirs.
for skill in pm-agent-doctrine outcomes-over-outputs outcome-drift risk-surfacing \
             evidence-forcing prioritization-frameworks assumption-testing; do
  [ -d "$SK/$skill" ] && ok "skill dir resolves: $skill" || bad "dangling skill ref: $skill"
done

# --- AC-10/AC-11: Wave-2 region has the four routes + both skills ---
MARK='WAVE-2 ROUTES'
BELOW=$(awk "/$MARK/{f=1} f" "$AGENTS")
for a in roadmap-reviewer prioritization-challenger experiment-readout-reviewer; do
  echo "$BELOW" | grep -q "$a" && ok "Wave-2 route present: $a" || bad "Wave-2 route missing: $a"
done
echo "$BELOW" | grep -qi "pm-critic\|pm-reviewer" && ok "Wave-2 route present: pm-reviewer critic" || bad "Wave-2 pm-reviewer route missing"
grep -q "outcome-drift" "$AGENTS" && grep -q "evidence-forcing" "$AGENTS" \
  && ok "AGENTS.md Skills Reference lists both new skills" || bad "AGENTS.md missing a new skill in Skills Reference"

# --- AC-12: canonical routing table above the marker is byte-unchanged ---
# The full diff for AGENTS.md must show additions only (zero real deletions).
DEL=$(git diff -- "$AGENTS" | grep -E '^-' | grep -vE '^---' || true)
[ -z "$DEL" ] && ok "AGENTS.md: additions only, no deletions (canonical table intact)" \
  || { echo "$DEL"; bad "AGENTS.md has deletions — canonical table changed"; }

# --- AC-13: tripwire — only domains/pm/ touched, no installed targets ---
OUT=$(git status --porcelain | awk '{print $2}' | grep -vE '^(domains/pm/|\.hero/)' || true)
[ -z "$OUT" ] && ok "only domains/pm/ + .hero touched (no installed targets)" \
  || { echo "$OUT"; bad "edits outside domains/pm/ — tripwire violated"; }

[ "$PASS" = 1 ] && echo "ALL VALIDATION PASSED" || { echo "VALIDATION FAILED"; exit 1; }
```

Additionally: `hero check` should surface no new dangling-reference or
broken-skill warnings for the pm domain after delivery.

## Boundaries

- **No Go / no engine code.** Content-only. No changes under `cmd/`, `internal/`,
  or any `.go` file.
- **`domains/pm/` pack source only.** Never edit installed/per-target copies
  (`.claude/`, `.codex/`, `.agents/`, or any harness output dir). Install
  propagates to all six targets from pack source (tripwire
  `harness-changes-cover-all-targets`).
- **Do not author the `experiment-design` skill.** The pre-registered experiment
  brief format is child #7's (`experiment-stage-and-metric-rca`) deliverable.
  This spec only *references* it as a forward dependency.
- **Do not undo child #1's work.** `pm-reviewer`'s Wave-1 Startup loads
  (`pm-agent-doctrine`, `outcomes-over-outputs`) stay. The canonical AGENTS.md
  routing table above the marker stays byte-for-byte unchanged.
- **Do not edit the canonical routing table.** All AGENTS.md additions go in the
  Wave-2 region below the marker (routes) and the Skills Reference list additions
  only.
- **No new commands.** Consistent with the design (§F ships no `/review` slash
  command in pm), routes point at invoking the agents directly. Do not author a
  `/critic`, `/review`, or `/experiment` command here.
- **Not the metric-RCA agent, not the experiment-designer agent.** Those are
  child #7 scope. This bundle is the four critics + their two skills only.
- **`roadmap-curator` is out of scope.** The drift critic reads framing; it does
  not take over state reconciliation from the shipped `roadmap-curator`.

## Risks

- **Overlap seam with child #1 (`conflicts-with`).** Both children edit
  `domains/pm/AGENTS.md`. Child #1 owns the canonical table and opened the Wave-2
  region; this child appends below the marker only. If delivered concurrently, an
  edit collision is possible — the `conflicts-with` relation is the soft mutex.
  Do not deliver this while child #1 is mid-flight touching AGENTS.md. AC-12's
  additions-only diff check is the guard.
- **Soft coupling with child #7 (`conflicts-with`).** `experiment-readout-reviewer`
  forward-references `experiment-design` (child #7). Delivered here, the reference
  is intentionally prose-only so no dangling load is introduced. When #7 lands, a
  follow-up may promote `experiment-design` into the reviewer's Startup load list
  — that is #7's or a later reconciliation's job, not this spec's. The audit open
  question (c) about hardening this to a `depends-on` is noted but not resolved
  here.
- **Adversarial-without-grounding failure mode.** The whole value of a critic is
  that its objections are checkable. An ungrounded "this feels inflated" is the
  exact distrusted free-association `pm-agent-doctrine` forbids. Each critic file
  must make corpus-grounding of objections a hard rule, not a suggestion — the
  reviewer during delivery should challenge any critic file whose Anti-patterns
  don't include "objections with no corpus anchor."
- **Skills Reference placement.** If the AGENTS.md Skills Reference list sits
  above the marker, adding the two skill names is an addition to a list (not a
  routing-table edit) and is permitted — but the delivering engineer must confirm
  via diff that no existing routing-table line changed (AC-12). If in doubt,
  surface the placement decision rather than guessing.

## Completion Ledger

Content-only delivery in `domains/pm/` pack source. 3 net-new critic agents + 1
sharpened agent + 2 new skills + AGENTS.md Wave-2 additions. Validation: ran the
spec's full `## Validation` bash block verbatim from repo root — 34/34 checks
PASS, exit 0. `hero check` surfaces no new dangling-ref/broken-skill warning.
Cold audit + verify run this same turn.

### Acceptance Criteria

| AC | Status | Evidence |
|---|---|---|
| AC-1 | DONE | `agents/roadmap-reviewer.md` drift critic — ratio tally + stale taxonomy + honest-roadmap; grep PASS (`60/30/10` + `stale`) |
| AC-2 | DONE | Startup loads `pm-agent-doctrine` + `outcomes-over-outputs`, both resolve |
| AC-3 | DONE | `skills/outcome-drift/SKILL.md` — `name:` frontmatter + ratio tally + 4-type stale taxonomy |
| AC-4 | DONE | `agents/prioritization-challenger.md` — RICE/ICE/WSJF challenge, Confidence→50%, inflated-input detection |
| AC-5 | DONE | challenger Startup loads `pm-agent-doctrine` (resolves) |
| AC-6 | DONE | `skills/evidence-forcing/SKILL.md` — named-evidence-or-neutral, show-the-math |
| AC-7 | DONE | `agents/pm-reviewer.md` sharpened (CPO premortem + "5 reasons") WHILE child-#1 Startup loads preserved |
| AC-8 | DONE | `agents/experiment-readout-reviewer.md` — SRM/peeking/guardrails/multiple-comparisons/significance; loads doctrine |
| AC-9 | DONE | forward-refs `experiment-design` in prose only; Startup explicitly does not load it — no dangling ref |
| AC-10 | DONE | four Wave-2 critic routes below the marker |
| AC-11 | DONE | `outcome-drift` + `evidence-forcing` in Skills Reference |
| AC-12 | DONE | `git diff domains/pm/AGENTS.md` = 17 insertions / 0 deletions; canonical table above marker byte-unchanged |
| AC-13 | DONE | change set confined to `domains/pm/`; no installed targets |
| AC-14 | DONE | every Startup-loaded skill across the 4 critics resolves — no dangling refs |

### Changes

| # | Changes item | Status | Note |
|---|---|---|---|
| 1 | Create `agents/roadmap-reviewer.md` | DONE | drift critic; loads doctrine + outcomes-over-outputs + outcome-drift + risk-surfacing |
| 2 | Create `skills/outcome-drift/SKILL.md` | DONE | ratio tally vs ~60/30/10 + 4-type stale taxonomy |
| 3 | Create `agents/prioritization-challenger.md` | DONE | anti-gaming input critic; loads doctrine + evidence-forcing + prioritization-frameworks |
| 4 | Create `skills/evidence-forcing/SKILL.md` | DONE | evidence-forcing discipline |
| 5 | Sharpen `agents/pm-reviewer.md` | DONE | adversarial CPO doc-critic; child-#1 loads/permissions/delegation preserved |
| 6 | Create `agents/experiment-readout-reviewer.md` | DONE | adversarial readout critic; forward-dep prose for child #7 |
| 7 | Append Wave-2 routes to `domains/pm/AGENTS.md` | DONE | Wave-2 routes (4) + reference rosters; additions-only (17/0) |

### Exercise-the-feature check

- [x] Full `## Validation` block run verbatim from repo root — 34/34 PASS, exit 0 (file existence, frontmatter, Startup load resolution, forward-dep-without-load, additions-only AGENTS.md diff, tripwire scope). `hero check` shows no new pm dangling-ref warning.
- [ ] Not exercised: live critic-agent invocation against a real roadmap/ranking/readout and `hero install` propagation — both require an installed harness session outside this content-authoring scope.

### Excellence Bar self-check

Yes. The four critics share one doctrine spine, each names its inversion move, every critic makes corpus-grounding a hard rule (and lists "objections with no corpus anchor" as an anti-pattern), and every critic is decision-gated (suggest-verdict, never auto-edit/reorder/ship). AGENTS.md kept surgical and additions-only.
