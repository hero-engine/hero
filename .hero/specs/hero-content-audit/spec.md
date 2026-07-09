---
title: Hero Content Audit — Agents, Commands, and Skills Quality Pass
slug: hero-content-audit
type: enhancement
status: completed
priority: P1
size: large
domain: engineering
tags: [content, agents, skills, commands, audit, quality, dedup]
created: 2026-07-05
relations:
  - target: core-vertical-layering
    kind: related
  - target: domain-plugin-architecture
    kind: related
  - target: v2-agents-and-skills
    kind: follows
mission_alignment: |
  The agents, commands, and skills ARE the surface Hero injects into
  every harness session. Every wasted word in them is context-window
  tax paid on every session by every user; every stale reference or
  drifted duplicate makes the next session start dumber, not smarter.
  Auditing this content is directly raising the floor.
completed_at: 2026-07-06T04:21:40Z
---

# Hero Content Audit — Agents, Commands, and Skills Quality Pass

## Context

Hero ships ~224 markdown content files (~168k words) that `hero install`
renders into user harnesses: 58 agents, 72 commands, and 94 skills across
`core/` and the four domain packs (`domains/engineering`, `domains/pm`,
`domains/sales`, `domains/chat`), plus the per-domain `AGENTS.md` routing
files. This content has grown accretively across v1, v2, and the domain-pack
refactor without a holistic quality pass.

A shallow recon already surfaced concrete smells:

- **13 skills exist in both `core/skills/` and `domains/engineering/skills/`.**
  8 are byte-identical (pure duplication: `auto-knowledge-capture`,
  `convention-writing`, `documentation-practices`, `executive-report`,
  `knowledge-flywheel`, `note-capture`, `nudge-awareness`,
  `project-context-generation`) and 5 have silently diverged
  (`agent-reliability`, `context-injection`, `next-handoff-emit`, `next-md`,
  `spec-format`). Divergence is worse than duplication — which copy wins is
  ambiguous, and fixes land in one and not the other.
- **Heavy files with no token budget.** The largest content files run
  2,000–2,700 words each (`feature-delivery-lead.md`, `deliver.md`,
  `spec-format/SKILL.md`, `domains/pm/AGENTS.md`). This text lands in agent
  context windows; verbosity is a per-session cost.
- **Frontmatter inconsistency.** e.g. skills carry `compatibility: opencode`
  even though Hero installs to six targets — unclear if the field is stale,
  wrong, or load-bearing.

Related but distinct: [[core-vertical-layering]] makes the core/domain split
physical in the repo layout; this audit judges the *content* — what each file
says, whether it earns its tokens, and whether the roster is right. Structural
findings feed that spec rather than duplicating it.

## Goal

A complete, evidence-backed audit report covering every shipped content file,
scoring each against a shared rubric, with a ranked findings list and sized
follow-up spec proposals. Every finding cites file paths (and diffs for drift).
No content edits happen in this spec — it is findings-only; remediation is
spun into follow-up specs.

## Kickoff

Findings-only audit of all shipped Hero content — 58 agents, 72 commands,
94 skills, and domain AGENTS.md files across `core/` and `domains/*`
(227 files, ~168k words).

**Status:** delivering — audit complete: 227/227 files triaged, 118 findings
(33 S1), 5 themes, 10 sized follow-up specs proposed. Awaiting verify gate.

**Pick up at:** read `audit-report.md` beside this spec, then spin the
follow-up specs — `content-dedup-resync` first (removes the double-edit
hazard every other fix would trip on).

→ `.hero/planning/features/hero-content-audit/audit-report.md`

## Approach

Audit in phases, one artifact per phase, all saved beside this spec so the
folder bundles the evidence. Delivery can fan phases 3a–3d out to parallel
subagents since each surface is independently auditable against the shared
rubric.

**Phase 1 — Inventory.** Machine-generated table of every content file:
path, surface (agent/command/skill/routing), domain, word count, frontmatter
fields present. This is the coverage checklist the report is verified against.

**Phase 2 — Rubric.** Short scoring rubric applied to every file. Dimensions:

1. **Earns its place** — does this agent/command/skill do a distinct job, or
   does it overlap another entry in the roster? (35 engineering agents is the
   roster most likely to have overlap.)
2. **Token efficiency** — signal density; flag files where the same guidance
   could land in materially fewer words, and boilerplate repeated across files
   that belongs in one shared skill.
3. **Actionability** — can a cold agent act on this, or is it aspirational
   prose? Concrete file paths, commands, and decision rules beat vibes.
4. **Freshness** — references to commands, tools, flags, paths, and other
   content files that no longer exist or have been renamed.
5. **Triggering** — do descriptions/frontmatter make the right sessions load
   the right content? (Skill descriptions are load-bearing for auto-selection.)
6. **Harness-agnosticism** — per the `harness-changes-cover-all-targets`
   tripwire: no Claude-only assumptions; guidance must work on all six install
   targets, and any harness-specific text must be explicitly scoped.
7. **Format consistency** — frontmatter schema, section structure, and voice
   consistent within each surface.

**Phase 3 — Per-surface passes.** Apply the rubric file-by-file:
- **3a Agents** (58): role clarity, roster overlap map, delegation guidance.
- **3b Commands** (72): routing-table alignment (do the CLAUDE.md/AGENTS.md
  routing tables, the slash/CLI parity table, and the actual command files
  agree?), argument handling, cross-references to skills that must exist.
- **3c Skills** (94): the dedup/drift matrix (core vs engineering vs pm vs
  sales — including cross-domain copies, not just the 13 known core/engineering
  pairs), frontmatter audit, size outliers.
- **3d Routing files**: per-domain `AGENTS.md` files and `core/commands/hero.md`
  — accuracy of what they claim the packs contain.

**Phase 4 — Synthesis.** `audit-report.md`: ranked findings (severity ×
blast-radius), the dedup/drift matrix, roster recommendations
(merge/cut/rewrite candidates), and a proposed set of sized follow-up specs.

## Boundaries

- **No content edits.** Not even byte-identical dedup — every fix, however
  trivial, goes in the follow-up proposals. This keeps the audit reviewable
  as a pure findings artifact.
- **Repo structure is out of scope.** Where files should live (core vs domain,
  overlay mechanics in `content.go`) belongs to [[core-vertical-layering]];
  this audit may hand it evidence but not redesign the layout.
- **New capabilities are out of scope.** "We should add an agent for X" is
  roadmap material (`/discover`), not an audit finding — except where a gap is
  the flip side of a roster overlap being flagged.
- **Engine code is out of scope.** How content is loaded, rendered, or
  installed is not audited; only the markdown content itself.
- **Web docs (`web/docs/src/`) are out of scope** — they mirror content but
  are not what agents consume.

## Risks

- **Rubric subjectivity.** "Too verbose" without a bar becomes taste. Mitigate:
  rubric defines per-surface word-count reference bands from the current
  distribution, and every verbosity finding must name what to cut.
- **Moving target.** Active specs (`agent-safety-conventions`,
  `roadmap-review`, `v2-agents-and-skills` follow-ons) are adding content.
  The inventory is stamped at a git SHA; files changed after the stamp are
  noted, not re-audited.
- **Drifted duplicates may be intentional.** A core/domain pair that differs
  might be deliberate specialization, not drift. The audit classifies each
  diverged pair as intentional/accidental with evidence (git history), and
  marks it "unclear" rather than guessing.
- **Finding volume.** 224 files can produce hundreds of findings. The report
  must rank ruthlessly; anything below the cut lands in an appendix, not the
  action list.

## Acceptance Criteria

- THE SYSTEM SHALL produce `inventory.md`, `rubric.md`, and `audit-report.md`
  in `.hero/planning/features/hero-content-audit/`.
- WHEN the audit completes THE SYSTEM SHALL have triaged every file in the
  Phase 1 inventory, with the report's coverage table accounting for all of
  them (audited, or explicitly skipped with reason).
- THE SYSTEM SHALL include a dedup/drift matrix covering all cross-surface
  duplicate content (core↔engineering and any cross-domain pairs), classifying
  each pair as identical, intentional divergence, accidental drift, or unclear.
- IF a finding claims staleness (dead reference, renamed command, missing
  skill) THEN THE SYSTEM SHALL cite the referencing file and the evidence the
  target is gone.
- THE SYSTEM SHALL rank findings by severity × blast-radius and propose
  follow-up specs for remediation, each with a title, one-line scope, and
  size on the 6-tier ladder.
- THE SYSTEM SHALL make zero edits to files under `core/` and `domains/`
  during the audit.

## Completion Ledger

| # | AC / Change item | Status | Evidence |
|---|---|---|---|
| AC1 | `inventory.md`, `rubric.md`, `audit-report.md` produced in the spec folder | DONE | all three exist beside this spec (plus dup-map.txt and four findings files) |
| AC2 | Every Phase-1 inventory file triaged; coverage table accounts for all | DONE | 227/227 — cold auditor diffed inventory rows against per-file verdicts programmatically, zero gaps |
| AC3 | Dedup/drift matrix, every pair classified | DONE | findings-skills.md §(b) 13-pair matrix w/ git evidence + OverlayFS precedence; findings-commands.md §D 17-pair table; agent pairs in findings-agents.md; none left unclear |
| AC4 | Staleness findings cite referencing file + evidence target gone | DONE | all S1s verified against the `hero` binary and `internal/` source; auditor re-verified 11 samples — 10 held, 1 false S1 withdrawn with citation (`internal/cli/pulse.go:33`) |
| AC5 | Findings ranked severity × blast-radius; follow-ups w/ title, scope, size | DONE | audit-report.md 14-row ranked S1 table + 10 sized follow-up proposals |
| AC6 | Zero edits under `core/` and `domains/` | DONE | `git status --porcelain core/ domains/` empty — re-verified by cold auditor after fix pass |
| P1 | Phase 1 — inventory | DONE | inventory.md, 227 rows, SHA `bc86ad9` |
| P2 | Phase 2 — rubric | DONE | rubric.md, 7 dimensions, empirical bands |
| P3 | Phase 3 — per-surface passes | DONE | findings-{agents,commands,skills,routing}.md (58/58, 72/72, 94/94, 3+1+6) |
| P4 | Phase 4 — synthesis | DONE | audit-report.md, 120 findings (31 S1 · 54 S2 · 35 S3), 5 themes |

Exercise-the-feature check: n/a — documents-only delivery; the cold audit
(delivery-audit.md, verdict SHIP/clean on re-audit) is the independent
evidence pass.

## Validation

- `ls .hero/planning/features/hero-content-audit/` shows the three artifacts.
- Spot-check 5 random inventory rows against the report's coverage table.
- `git status core/ domains/` is clean after the audit run.
- Each proposed follow-up spec is actionable enough to hand to `/design`
  or `/deliver` without re-reading the full report.
