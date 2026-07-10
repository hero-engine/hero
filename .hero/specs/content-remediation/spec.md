---
title: Content Remediation — Audit Follow-Through Across Packs, Gates, and Harnesses
slug: content-remediation
type: initiative
status: completed
priority: P1
size: x-large
domain: engineering
tags: [content, audit, remediation, initiative, packs, harness]
created: 2026-07-06
relations:
  - target: hero-content-audit
    kind: builds-on
  - target: content-dedup-resync
    kind: builds-on
  - target: core-vertical-layering
    kind: related
child:
  - pm-pack-phantom-surfaces
  - sales-pack-reality-sync
  - core-commands-domain-neutral
  - delivery-gate-consistency
  - harness-agnosticism-sweep
  - routing-file-completeness
  - token-efficiency-pass
  - chat-pack-disposition
mission_alignment: |
  The audit proved the shipped content lies to agents: phantom commands,
  fictional CLI, contradictory delivery gates, Claude-only assumptions on
  six-target installs. Every lie makes a session start dumber. This
  initiative retires every confirmed finding so the injected surface is
  true, minimal, and works on every harness for every domain.
completed_at: 2026-07-10T01:26:06Z
---

# Content Remediation — Audit Follow-Through Across Packs, Gates, and Harnesses

## Context

The [[hero-content-audit]] (completed 2026-07-05) triaged all 227 shipped
content files and produced 120 verified findings (31 S1) collapsing into
five themes: drifted duplication (T1), phantom surfaces (T2), delivery-gate
contradictions (T3), install-reality mismatches (T4), and token waste /
invisible content (T5). The report proposed ten sized follow-ups.

State at initiative creation:

- **Follow-up #1 shipped**: [[content-dedup-resync]] (commit `177e8a1`)
  collapsed the 34 core↔engineering duplicates to single masters in core
  and added the `content_parity_test.go` CI gate. Every child of this
  initiative edits each file exactly once as a result.
- **Follow-up #9 is in flight externally**: the `installFlat` README
  exclusion and the cursor zero-skills bug are running as background
  fix sessions (spawned 2026-07-06), alongside a `hero install --json`
  exit-code fix. Not children here; their landing is a cross-check in
  Validation.
- The remaining eight follow-ups are this initiative's children, each
  fully designed in its own folder beside this spec.

## Goal

Every confirmed audit finding is retired by a delivered child spec: no
shipped content references a surface that doesn't exist on its install;
the delivery doctrine is stated once and gate-consistent everywhere;
content works on all six install targets; the named verbosity cuts are
applied without losing a rule; and the chat pack's status is deliberate
instead of accidental. `hero spec verify` passes for all eight children.

## Kickoff

Initiative wrapping the eight remaining audit follow-ups as fully-designed
children (pm/sales/core truth fixes, gate consistency, harness sweep,
routing completeness, token cuts, chat disposition).

**Status:** planning — all eight children designed, none delivering.

**Pick up at:** Wave 1 — `/deliver pm-pack-phantom-surfaces` (or `/drive
content-remediation` to run the whole initiative; wave order below).

→ `.hero/planning/initiatives/content-remediation/spec.md`

## Children and delivery order

Waves minimize same-file conflicts. Within a wave, children are
independent and parallelizable; across waves, order matters.

| Wave | Child | Type | Size | Why this position |
|---|---|---|---|---|
| 1 | [[pm-pack-phantom-surfaces]] | bug | medium | Independent; highest S1 density (every PM session) |
| 1 | [[sales-pack-reality-sync]] | bug | medium | Independent; sales session-start fails today |
| 1 | [[core-commands-domain-neutral]] | bug | medium | Independent post-dedup; blast radius = pm/sales installs |
| 1 | [[delivery-gate-consistency]] | enhancement | medium | Independent; extracts the ledger contract others will cite |
| 1 | [[chat-pack-disposition]] | decision | — | Independent; unblocks nothing but closes T4's dead-pack row |
| 2 | [[harness-agnosticism-sweep]] | enhancement | medium | Touches pm/sales files wave 1 edits; owns parity-table + scoping content inside AGENTS.md |
| 2 | [[routing-file-completeness]] | enhancement | small | After harness sweep — skeleton/heading alignment formats final AGENTS.md content; both edit the Go fallback in lockstep, serialize 2a→2b. Must also follow core-commands-domain-neutral: its roster tables count command files, and that child retires `/prime` and moves `drive.md` into the engineering pack |
| 3 | [[token-efficiency-pass]] | enhancement | medium | Last — cuts against post-fix text; depends on gate-consistency's extraction to avoid double-moving the ledger contract |

### Wave 1 progress

All five wave-1 children delivered as of 2026-07-08: pm-pack-phantom-surfaces,
sales-pack-reality-sync, core-commands-domain-neutral, delivery-gate-consistency,
chat-pack-disposition. Wave 2 (harness-agnosticism-sweep,
routing-file-completeness) is now unblocked.

chat-pack-disposition's delivery also closes three audit findings directly:
F9 (findings-commands.md, chat pack graded dead — corrected: it's a live
client-embedded pack, consumed by hero-code's build.rs, not installable
via `hero install`; the exclusion is now documented in `content.go` and
enforced by `TestDomainsDirectory_AllEntriesAccounted`), F29
(findings-skills.md / commands, unscoped client internals in
`ask-corpus.md` / `space.md` — fixed), and the routing S3 finding
(findings-routing.md, engineering-fallback misroute risk — preempted by
`domains/chat/AGENTS.md`).

### Wave 2 / Wave 3 — all children delivered

`harness-agnosticism-sweep` and `routing-file-completeness` (wave 2)
landed, followed by `token-efficiency-pass` (wave 3, the last child) on
2026-07-09 — `hero spec verify` passed and it's archived to
`.hero/specs/token-efficiency-pass/`. All 8 declared children now show
`status: completed`. Note for the record: this initiative's own
`status:` frontmatter was still `planning` immediately after
`token-efficiency-pass`'s verify ran and after a `hero check
--reconcile` pass — the auto-complete-parent check in `hero spec
verify` (`internal/cli/verify.go: autoCompleteParentIfReady`) did not
flip it, for reasons not diagnosed here (not hand-edited per this
child's delivery instructions — a human or a follow-up session should
inspect why the check didn't fire despite all 8 declared `child:`
entries resolving to `completed` specs).

## Cross-cutting concerns

- **Dual-edit lockstep.** Engineering AGENTS.md and
  `generateEngineeringAgentsMdBody` (internal/install/agents_md.go) are
  test-enforced identical. Waves 2a/2b both touch them — every such
  change lands in both places in the same commit.
- **The parity gate governs new overrides.** Any child adding a
  domain-pack file that shadows a core path must annotate `core_fork:`
  or CI fails (content_parity_test.go).
- **Findings cite pre-dedup paths.** Every child was designed against
  the post-`177e8a1` tree, but delivery agents must still verify paths
  before editing — the audit evidence predates the dedup.
- **Single-owner rule.** Gate doctrine → delivery-gate-consistency's
  extracted skill; verbosity cuts → token-efficiency-pass; AGENTS.md
  structure → routing-file-completeness; AGENTS.md truth per pack →
  that pack's wave-1 child. A finding is fixed in exactly one child.
- **In-flight overlap watch.** `agent-safety-conventions` (planning) and
  the two external engine-fix sessions may touch adjacent files — each
  child's delivery starts with `hero status` + a conflict check.

## Boundaries

- **Engine code** stays out except where a child explicitly names it
  (Go fallback lockstep, spec-type loader for sales' deal type). The
  external fix sessions own installFlat/cursor/--json.
- **Repo layout** (where content directories live) remains
  [[core-vertical-layering]]'s.
- **New capabilities** (new agents, new domains, roster expansion) are
  roadmap work, not remediation.
- **hero-code-side work** (context-driven domain activation) proceeds
  independently in the peer repo.

## Risks

- **Cross-child file conflicts** if waves are ignored — the wave table
  is the mitigation; `/drive` should respect it.
- **Verbosity cuts losing rules** — token-efficiency-pass carries a
  per-file rule-inventory requirement; its auditor must diff rules, not
  just words.
- **AGENTS.md/Go-fallback drift** — the identity test catches it, but
  only if `go test` runs before commit (goreleaser hook does).
- **Scope creep from freshly-found issues** — new findings discovered
  during delivery go to `hero note` / new specs, not into a child's
  scope mid-flight.

## Acceptance Criteria

- WHEN all wave-1 children are completed THE SYSTEM SHALL contain no content file instructing a CLI invocation, slash command, agent, or skill that does not exist on the installs that receive it (per each child's validation).
- WHEN delivery-gate-consistency completes THE SYSTEM SHALL state the Completion Ledger contract and completion path (`hero spec verify`) in exactly one owning skill, with all delivery surfaces citing it.
- WHEN the initiative completes THE SYSTEM SHALL pass `hero spec verify` for all eight children and `go test ./...` including the parity and AGENTS.md-identity tests.
- THE SYSTEM SHALL record the chat pack's installability status as a deliberate, tested decision rather than an accidental omission.

## Validation

- `hero list --status completed` shows all eight children at the end.
- Re-run the audit's per-surface freshness spot-checks (each child's
  Validation section carries its own) — zero repeat findings.
- Cross-check: the three external fix sessions (installFlat README,
  cursor skills, install --json) landed or are re-filed.
