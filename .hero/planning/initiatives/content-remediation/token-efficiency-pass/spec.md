---
title: "Token Efficiency Pass — apply the content audit's named verbosity cuts"
slug: token-efficiency-pass
type: enhancement
status: planning
priority: P2
size: medium
domain: engineering
created: 2026-07-06
tags: [content-audit, token-efficiency, dedup, commands, agents, skills]
relations:
  - {target: content-remediation, kind: parent}
  - {target: hero-content-audit, kind: related}
  - {target: delivery-gate-consistency, kind: follows}
---

# Token Efficiency Pass — apply the content audit's named verbosity cuts

## Context

The hero-content-audit (`.hero/specs/hero-content-audit/findings-commands.md`,
`findings-agents.md`, `findings-skills.md`, `findings-routing.md`, base SHA
`bc86ad9`) flagged a recurring waste pattern across the shipped content packs:
command and agent files restate content that a skill they already load owns,
sibling files restate each other, and boilerplate repeats per-file. Every cut
in this spec was **individually named by the audit** — F12 (deliver.md, 5× the
p90 command length), F13 (peer.md), F17 (diagnose.md), F27 (resume.md), the
agents-pass token-efficiency findings (feature-delivery-lead, the architecture
trio, buyer-researcher, story-writer, the investigator doctrine, scrubber
boilerplate), and the skills/routing-pass duplication findings (spec-sizing,
the pm pitch/betting/roadmap/OST clusters, delivery-audit's double legend, the
grounding paragraph, sales boilerplate, PM AGENTS.md re-teaching). This spec
operationalizes them; it identifies nothing new.

Since the audit, commit `177e8a1` (content-dedup-resync) made core/ the single
master for previously duplicated core↔engineering content: `resume.md`,
`agent-reliability`, `spec-format`, and their siblings are **core-only** now,
and a CI parity gate protects the split. All paths and word counts below were
re-verified against the current tree (`wc -w`, 2026-07-06); they are the
before-baseline for validation. Files still in the engineering pack:
`deliver.md`, `diagnose.md`, `peer.md`, `feature-delivery-lead.md`,
`spec-sizing`.

Sibling coordination: `delivery-gate-consistency` (this spec `follows` it)
owns the Completion-Ledger extraction-to-skill and the
feature/platform-delivery-lead merge. `routing-file-completeness` owns
AGENTS.md structural fixes. The sales/pm content-truth (phantom-CLI,
phantom-agent) fixes are other siblings' scope. This spec is verbosity only.

## Goal

Every file named in Changes is at or under its target word count, with each
displaced rule living in exactly one named home (single-owner rule) and a
pointer left behind — no rule lost, verified by a per-file before/after rule
inventory. Aggregate: roughly 7,500–8,000 words cut from always-loaded and
per-invocation content, offset by ~1,400 words of new/expanded skill homes
(batch-discipline, buyer-research, architecture-principles), for a net corpus
reduction of ~6,000–6,500 words and a substantially larger per-session context
saving (most cuts land in files loaded on every `/deliver`, `/diagnose`, or PM
session).

## Kickoff

Applies the content audit's named verbosity cuts: ~20 command/agent/skill
files trimmed to target word counts, with every displaced rule moved to a
single owning skill and replaced by a pointer.

**Status:** planning — spec authored from audit findings; no cuts made yet.

**Pick up at:** confirm `delivery-gate-consistency` has landed (ledger home
may have moved from engineer.md to a skill), then execute Changes in order,
starting with the two new skill homes (items 1–2) so pointers have targets.

→ `.hero/planning/initiatives/content-remediation/token-efficiency-pass/spec.md`

**Files:** domains/engineering/commands/deliver.md,
domains/engineering/agents/feature-delivery-lead.md,
.hero/specs/hero-content-audit/findings-commands.md
**Skip:** platform-delivery-lead merge and ledger extraction — sibling
delivery-gate-consistency owns both.

## Approach

Four design principles govern every item:

1. **Single-owner rule.** Each piece of doctrine lives in exactly ONE skill or
   agent file; every other surface that needs it carries a one-line pointer
   (rule summary + "see `<skill>`"). Owners are named per item below. When a
   command already mandates loading the owning skill (peer.md, the scrubbers,
   story-writer), the pointer is nearly free.
2. **New homes before cuts.** Two new skills are created first
   (`batch-discipline`, `buyer-research`) and one existing skill is expanded
   (`architecture-principles`) so that no cut ever leaves a rule homeless.
   Content moves verbatim-or-tightened into the home in the same commit as
   the cut that references it.
3. **No rule lost.** Before editing each file, enumerate its rules/constraints
   (the audit's per-finding evidence is the starting checklist); after
   editing, every rule must be either (a) still present, (b) present in its
   named single home with a pointer, or (c) an exact duplicate of a rule
   already in the home. Record the inventory in the delivery notes per file.
4. **Measure.** `wc -w` before/after per file against the targets below. The
   before numbers in this spec are the re-verified baseline, not the audit's.

Sequencing: item order below is execution order — homes first, then the two
big engineering files, then per-pack sweeps. Where
`delivery-gate-consistency` has relocated the ledger contract out of
`engineer.md`, pointers target the new home; if it hasn't landed, pointers
target `engineer.md` "Closing output" (the current owner) and the sibling
updates them later — either way this spec never copies ledger content.

## Changes

Ordered by dependency, then by file. Each item: file (before → target words),
what to cut, and where displaced content lives.

1. **Create `domains/engineering/skills/batch-discipline/SKILL.md`** (new,
   ~400–500 words). Single home for generic multi-spec/multi-agent batch
   protocol: agent-count/concurrency discipline, per-item isolation, the
   "after all agents complete" verification loop, and the summary-table
   format — sourced from diagnose.md's "Parallel batch mode" section (audit
   F17) and the overlapping ~70% of deliver.md's Batch/Queue modes (F12).
   Standard skill frontmatter (name, description, metadata); no
   `compatibility:` key (post-v0.8.0 convention).

2. **Create `domains/sales/skills/buyer-research/SKILL.md`** (new, ~700
   words). Single home for buyer-researcher's "Company research" dimension
   catalog (subsections 1–8: tech-stack signal sources, news taxonomies,
   trigger tables). Content moves, not rewritten. Loaded by buyer-researcher.

3. **Expand `domains/engineering/skills/architecture-principles/SKILL.md`**
   (275 → ~500–550 words). Absorb the shared architectural stance,
   scale-readiness rules, and strict rules currently triplicated (~700 words
   total) across brownfield-architect, greenfield-architect, and
   architecture-reviewer — one canonical copy of the
   monolith-first / no-premature-distribution / no-CQRS doctrine. This is
   the home all three agents already claim it lives in.

4. **`domains/engineering/commands/diagnose.md`** (898 → ~500). Replace the
   ~400-word "Parallel batch mode" + "After all agents complete" protocol
   with the batch-selection rules (when to batch, how to pick items) plus a
   pointer to `batch-discipline`. Single-bug path untouched. Displaced
   content: item 1.

5. **`domains/engineering/commands/deliver.md`** (2,673 → ~1,400–1,500).
   Four cuts per audit F12:
   - Definition of done stated once (keep the `## Definition of done`
     section); delete its restatements in the `--supervised` mode row and
     the "MUST run `hero spec verify`" block (~200 words). Home: the
     section itself.
   - Cold-audit pass (step 6, ~600 words) reduced to invocation + verdict
     routing (~150 words). Block formats (AUDIT_VERDICT/HEADLINE/HIGHLIGHTS),
     display policy, signal-preservation rule: home is the
     `delivery-audit` skill, which already carries them — verify each cut
     rule exists there before deleting; add to the skill any that don't.
   - Completion-Ledger validation (~350 words across step 5 and batch
     step 4) reduced to the DONE/PARTIAL/SKIPPED routing rules stated once
     + pointer to the ledger contract's single home (engineer.md "Closing
     output" today; wherever `delivery-gate-consistency` puts it). Do NOT
     move or copy the contract itself — sibling's scope.
   - Merge `## Batch mode` and `## Queue mode` (~70% shared) into one
     multi-spec mode with a queue-ordering variant, deferring loop
     discipline to `batch-discipline` (item 1).
   Estimated recovery ~1,100–1,300 words.

6. **`domains/engineering/commands/peer.md`** (979 → ~580). Cut the
   prompt-composition guidance and budget defaults (steps 3–4), the "What
   NOT to do" list, and the example dispatch block — all restate
   `cross-repo-peering`, which the command already mandates loading (audit
   F13, ~400 words). Keep: sub-action detection table, pre-flight
   `hero peer list`, trail-discipline checklist. Home: `cross-repo-peering`
   skill (verify each cut rule is present there; port any that aren't).

7. **`core/commands/resume.md`** (554 → ~400; core-only since `177e8a1`).
   Per audit F27: trim the 18-phrase "What to say" trigger list to ~6
   representative triggers; cut "Why this matters" (~80 words of pep talk)
   to one line; delete the closing "run unconditionally / ≥99% useful"
   paragraph that repeats When-to-use bullet 1. No displaced home needed —
   this is pure redundancy within the file.

8. **`domains/engineering/agents/feature-delivery-lead.md`** (2,685 →
   ~1,600, the audit's ~40%). Four compressions, each to one pointer
   sentence + the non-negotiable rule:
   - Step 4d sizing-nudge walkthrough (~190 words) → pointer to
     `spec-sizing`, which carries the paste-ready phrasing the step already
     tells the lead to quote.
   - Step 17 Completion-Ledger validation (~260 words) → keep the halt
     condition + pointer to the ledger contract's home (same rule as
     item 5: never copy the contract; coordinate with
     `delivery-gate-consistency`).
   - Step 18 cold-audit mechanics → keep invocation + verdict routing;
     home: `delivery-audit`.
   - "Challenge handling" section → keep the trigger + halt rule; mode
     detection and response protocol home: `challenge-diagnosis`.
   Also merge the duplicated nudge-precedence explanation in design-phase
   paragraphs 2–3. Do NOT touch platform-delivery-lead or merge the two
   leads — sibling's scope.

9. **Architecture trio** — cut the triplicated stance moved in item 3,
   keeping 2–3 agent-specific bullets each plus the existing "load
   `architecture-principles`" instruction:
   - `domains/engineering/agents/brownfield-architect.md` (613 → ~420)
   - `domains/engineering/agents/greenfield-architect.md` (605 → ~410)
   - `domains/engineering/agents/architecture-reviewer.md` (341 → ~260)

10. **`domains/sales/agents/buyer-researcher.md`** (1,310 → ~600). Replace
    the "Company research" subsections 1–8 with a load instruction for
    `buyer-research` (item 2) + a one-line summary of the dimensions. Keep
    the ground-in-hero-first rule, workflow, output formats, and rules.

11. **`domains/pm/agents/story-writer.md`** (1,468 → ~1,250). Step 5: keep
    one EARS example; pattern set home: `acceptance-criteria-ears` (loaded
    at startup already). Drop the v1 filename-note block to a README/comment.

12. **Investigator doctrine** — `domains/pm/agents/pm-investigator.md`
    (1,183 → ~1,080) and `domains/engineering/agents/debug-investigator.md`
    (1,598 → ~1,500). The shared dead-end/persistence doctrine ("you won't
    always find the answer", worst-outcome framing) is compressed to a
    pointer in both; home: core `agent-reliability` (post-`177e8a1` the
    core single master already carries the persistence/honesty rules).
    Verify each cut sentence's rule exists in `agent-reliability` first;
    port any missing sentence there.

13. **`domains/engineering/skills/debugging-investigation/SKILL.md`** (687 →
    ~570). Delete the verbatim ~110-word "Ground before you guess"
    paragraph (line 20); home: core `agent-reliability` line 26 carries the
    identical text (verified 2026-07-06). Leave a one-line pointer.

14. **`domains/engineering/skills/spec-sizing/SKILL.md`** (2,120 → ~1,880).
    Merge `## Drift handling` (line 163) and `## What to do on drift`
    (line 260) into one section (~230 duplicated words). In-file dedup; no
    external home.

15. **`domains/engineering/skills/delivery-audit/SKILL.md`** (1,720 →
    ~1,690). Delete the second verbatim `✓/✗/~` legend (the copy after
    "Surface decision rules", ~line 177); the first copy (~line 102) is the
    home.

16. **PM pack skill dedup** — single owner per cluster, others keep a
    one-line cross-ref:
    - Betting-table/cooldown/appetite mechanics: owner
      `cycle-planning` (2,028 → ~2,000);
      `pitch-writing-shape-up` (2,078 → ~1,830) keeps author-facing
      implications only.
    - Pitch quality bars (Appetite/Rabbit-Holes/No-Gos examples,
      five-adjective test): owner `pitch-writing-shape-up`;
      `prd-structure` (1,731 → ~1,560) and `prd-anti-patterns`
      (1,923 → ~1,760) point at it.
    - Weekly roadmap reconciliation: `roadmap-framing` (1,818 → ~1,660)
      merges its two internal tellings and owns the policy;
      `cross-domain-graph-query` keeps only the query mechanics and
      cross-refs the policy.
    - OST ↔ discovery-cadence mutual restatement:
      `opportunity-solution-trees-torres` (1,708 → ~1,590) owns tree
      structure/levels; `continuous-discovery-cadence` (1,721 → ~1,600)
      owns the cadence; each replaces its restated section with a
      cross-ref.
    - PM "mission — principle #N" closers (~10 files): drop the closing
      line pack-wide, or keep it only where the file gives the mission
      document a resolvable path (audit S3 — the path is never given).
      In-body principle references that carry an argument stay.

17. **Sales pack dedup**:
    - Delete the "What this skill covers" description-restating boilerplate
      from all 7 `domains/sales/skills/*/SKILL.md` (~40–60 words each,
      ~350 total). Home: the frontmatter `description:` each restates.
    - `deal-qualification` (1,721 → ~1,400): per-dimension question banks
      → one exemplar question per dimension + pointer to
      `discovery-questioning` (owner); "Signs of a true champion" →
      pointer to `deal-strategy`'s champion definition (owner).

18. **`domains/pm/AGENTS.md`** (2,247 → ~1,750). Compress the four
    re-teaching sections to 2–3-line pointers at their owners: "The handoff
    is an owner flip" → `handoff-protocol` skill + `/handoff` command;
    "Methodology presets" → `pm-preset-detection` (the section already
    defers to it); the second vocabulary-routing table → fold its four rows
    into the main table; "Capture execution plans" → core plan-capture
    guidance. Content compression only — no structural/roster changes
    (sibling `routing-file-completeness` owns structure), no phantom-route
    fixes (content-truth siblings own those).

19. **Optional, lowest priority — scrubber startup boilerplate**
    (`domains/engineering/agents/*-scrubber.md`, 7 files): move the
    identical 3-step `## Startup` block into `code-scrub` (owner); each
    scrubber keeps one line ("Startup: follow the `code-scrub` startup
    sequence"). ~120 words net. Skip without ceremony if the pass runs
    long.

## Boundaries

- **`delivery-gate-consistency` (sibling, this spec follows it)** owns the
  Completion-Ledger extraction-to-skill and the
  feature/platform-delivery-lead merge. This spec never moves, copies, or
  rewrites the ledger contract, and never edits
  `platform-delivery-lead.md` — items 5 and 8 only compress restatements
  down to pointers at whatever home that spec establishes.
- **`routing-file-completeness` (sibling)** owns AGENTS.md structure,
  rosters, heading depth, and missing routing rows. Item 18 compresses
  prose within existing sections only. The dual-edit constraint
  (engineering AGENTS.md ↔ Go fallback) is its problem, not ours — this
  spec does not touch `domains/engineering/AGENTS.md`.
- **Sales/PM content-truth fixes are siblings' scope**: phantom CLI
  invocations (`hero event`, `hero new feature`, search-type fabrications),
  phantom agents/skills, wrong paths, hero.json schema claims. If a
  sentence this spec would cut also contains a truth bug, cut it here and
  note it; do not fix truth bugs in kept text.
- **No core↔engineering dedup work** — `177e8a1` already did it and CI
  guards it. This spec must not create new same-named core/domain pairs.
- **No behavior changes**: no new rules, no rule rewording that changes
  meaning, no roster or frontmatter-schema changes (the `compatibility:`
  strip and `role:` cleanup are separate audit items).
- Chat pack files (dead content, F9) are out of scope entirely.

## Risks

- **Losing a rule during a cut** — the central risk. A "restatement" may
  contain a nuance its supposed home lacks (e.g. deliver.md's ledger text
  could carry a routing rule engineer.md doesn't). Mitigation: the
  per-file before/after rule inventory in Approach #3 — enumerate rules
  before editing, diff against the home, port any orphan into the home
  before deleting the copy. This inventory is a deliverable per file, not
  optional.
- **Sibling collision on shared files.** `delivery-gate-consistency`
  touches deliver.md/feature-delivery-lead's ledger sections and may move
  the contract's home mid-flight. Mitigation: `follows` relation — execute
  after it lands; re-verify the ledger home before items 5 and 8.
- **Baseline drift.** Word counts were re-measured 2026-07-06 at the
  current tree; if other remediation siblings land first, re-run `wc -w`
  and adjust targets proportionally rather than cutting to stale numbers.
- **Pointer rot.** A pointer to a skill that a later pass renames or
  merges dangles silently. Mitigation: pointers name skills (not file
  paths) per the audit's cross-link finding; `hero check` / drift tooling
  catches renames.
- **Over-cutting command files below usability** — a command must still
  work for a cold agent without loading every referenced skill.
  Mitigation: each command keeps the decision rules (when/what) and defers
  only the how (formats, protocols) to skills.

## Acceptance Criteria

- THE SYSTEM SHALL keep every file listed in Changes at or under its target word count, measured by `wc -w` (±10% tolerance).
- THE SYSTEM SHALL provide exactly one owning file for each displaced rule, with every former location carrying a pointer naming that owner.
- WHEN a rule is removed from a file THE SYSTEM SHALL record it in that file's before/after rule inventory with its new home (or the home's pre-existing duplicate).
- THE SYSTEM SHALL ship `domains/engineering/skills/batch-discipline/SKILL.md` containing the multi-spec batch protocol, referenced by both deliver.md's multi-spec mode and diagnose.md's batch path.
- THE SYSTEM SHALL ship `domains/sales/skills/buyer-research/SKILL.md` containing the research-dimension catalog, loaded by buyer-researcher.
- THE SYSTEM SHALL state the architectural stance / scale-readiness / strict rules only in `architecture-principles`, with the three architecture agents each keeping at most 3 agent-specific bullets.
- THE SYSTEM SHALL state deliver.md's definition of done exactly once within that file.
- THE SYSTEM SHALL contain exactly one `✓/✗/~` legend block in `delivery-audit/SKILL.md` and exactly one drift-handling section in `spec-sizing/SKILL.md`.
- THE SYSTEM SHALL contain the "Ground before you guess" paragraph only in core `agent-reliability`, with `debugging-investigation` pointing at it.
- THE SYSTEM SHALL NOT modify `platform-delivery-lead.md`, `domains/engineering/AGENTS.md`, the ledger contract's owning text, or any chat-pack file.
- IF a cut sentence contains a rule absent from its designated home THEN THE SYSTEM SHALL add the rule to the home in the same commit as the cut.
- WHEN the pass completes THE SYSTEM SHALL report per-file before/after word counts against this spec's baseline table.

## Validation

1. **Word-count gate**: script the before/after table —
   `wc -w` on every file in Changes; each at/under target (±10%). Baseline
   numbers are the "before" figures in Changes (re-verified 2026-07-06).
2. **Rule-inventory review**: for each edited file, the delivery notes list
   every removed rule and its single home; spot-check the five largest cuts
   (deliver.md, feature-delivery-lead, buyer-researcher, PM AGENTS.md,
   deal-qualification) by reading home files and confirming each inventoried
   rule is present.
3. **Pointer resolution**: grep every added pointer for its named skill/agent
   and confirm the target exists in the same install surface (engineering
   pointers → engineering+core; pm → pm+core; sales → sales+core). No
   pointer may target a skill the loading pack doesn't ship.
4. **Duplication greps**: `grep -c "Definition of done"`-style checks per
   Acceptance Criteria — one DoD in deliver.md; one legend in delivery-audit;
   one drift section in spec-sizing; "Ground before you guess" only in
   core/skills/agent-reliability; "What this skill covers" absent from
   domains/sales/skills.
5. **CI parity gate**: `go test ./...` (includes the `177e8a1` core/domain
   parity check and `TestEngineeringPackBodyMatchesGoFallback`) passes —
   proving the pass created no new core/domain divergence and never touched
   the engineering AGENTS.md pair.
6. **Cold-read check**: for deliver.md, diagnose.md, and peer.md, read the
   trimmed command top-to-bottom and confirm a cold agent can determine
   *when* to do each step from the command alone, deferring only *how*
   details to the named skills.
7. `hero check` and `hero spec lint token-efficiency-pass` clean.
