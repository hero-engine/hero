---
title: "Progressive Design AC Recap — Coverage and Delta Before Full Tables"
slug: progressive-design-ac-recap
type: enhancement
status: completed
domain: engineering
size: small
size_ack: small
priority: medium
created: 2026-07-20
tags: [design, acceptance-criteria, workflow, progressive-disclosure, token-efficiency, harness-agnostic]
relations:
  - target: agent-end-of-turn-recap
    kind: related
claimed_by: codex
claimed_at: 2026-07-20T14:16:14-06:00
delivery_method: manual
completed_at: 2026-07-20T20:22:34Z
---

# Progressive Design AC Recap — Coverage and Delta Before Full Tables

## Context

The engineering `/design` workflow currently requires every closing message to
render a one-row-per-acceptance-criterion table. That corrected an earlier failure
mode where a designer reported only a count, but larger specs now duplicate a long
contract that already lives in the linked spec. The result pushes the useful
design summary and next action off screen, especially after iterative refinement.

The canonical instruction is
`domains/engineering/commands/design.md`; `hero install` embeds and renders that
one source into the native command surface for all six targets (`opencode`,
`cursor`, `claude`, `copilot`, `codex`, and `generic`). The active mission
tripwire therefore requires one harness-agnostic rule plus propagation coverage,
not client-specific foldout markup.

## Goal

Make ordinary `/design` closings concise without losing contract visibility.
The default recap reports the criterion count, 2–5 thematic coverage areas, and
the criteria changed during the current design loop. It calls out criteria that
need human attention, while the complete contract remains in the linked spec.
The full one-row-per-criterion table appears only when the contract is short, the
user requests it, or the user explicitly enters a formal acceptance-criteria
review or approval step.

## Kickoff

Replace `/design`'s unconditional one-row-per-AC closing table with a bounded,
progressive-disclosure recap shared by every harness.

**Status:** completed — implementation, six-target validation, cold audit, and Hero verification all passed.

**Pick up at:** use the completed contract when refining future `/design`
closings; any broader recap changes should start as a separate related spec.

→ `.hero/specs/progressive-design-ac-recap/spec.md`

**Files:** `domains/engineering/commands/design.md`, `internal/install/harness_smoke_test.go`
**Skip:** client-specific `<details>` or foldout UI; installed commands must remain harness-agnostic.

## Problem

The current closing contract has only two states: useless count-only reporting or
complete tabular duplication. It lacks a bounded middle form that answers the
questions a reviewer usually has after a design loop:

- How large is the contract?
- What behavior does it cover?
- What changed in this iteration?
- Is anything ambiguous, deferred, conflicting, or otherwise in need of attention?
- Where can I inspect the exact wording?

A foldable full table would reduce visible height in clients that support it, but
would preserve the duplicated payload, depend on renderer behavior, and still be
noisy in transcripts and context windows. Progressive disclosure must therefore
be expressed as content selection, with the spec as the durable full-detail
surface.

## Approach

Replace the unconditional table instruction in the canonical engineering design
command with a deterministic closing policy:

1. **Default recap:** render `Acceptance criteria: N total`, followed by 2–5
   short thematic coverage labels and a `Changed this loop` line.
2. **Bound the delta:** name changed AC IDs with short behavior phrases when five
   or fewer changed. For more than five, report the changed count, summarize the
   affected themes, and point to the spec rather than recreating another long
   list. On initial creation, say `initial contract created`; do not treat every
   new criterion as a delta that must be enumerated.
3. **Surface exceptions:** add an `Attention` line only when a criterion is
   ambiguous, not independently testable, deferred, conflicting, or carries an
   unresolved decision. Name the affected IDs and issue. Omit the line when
   there are no exceptions.
4. **Expand selectively:** retain the existing compact `AC | Pattern | Behavior`
   table when there are five or fewer total criteria, when the user explicitly
   asks to see all criteria, or when the user explicitly asks to review or
   approve the acceptance-criteria contract. An ordinary post-design closing is
   not, by itself, a formal AC approval request.
5. **Keep the source visible:** always link the spec path and state that the full
   criteria live there. Preserve the existing score and next-step recommendation.

Representative default shape:

```markdown
**Acceptance criteria:** 14 total
**Coverage:** transcript spacing · hover stability · tool hierarchy · migration/parity
**Changed this loop:** AC-4 hover actions, AC-5 metadata reservation, AC-12 default migration
**Full contract:** [spec.md](...)
```

This is an instruction-only change. No client disclosure widget, Markdown
`<details>` contract, CLI flag, configuration field, or spec schema change is
needed.

## Changes

1. Update `domains/engineering/commands/design.md`.
   - Replace the unconditional one-row-per-AC closing table requirement with the
     default count + themes + current-loop delta contract.
   - Define the five-item bounds for short contracts and changed-criterion lists.
   - Require attention-worthy exceptions to remain visible and require the full
     spec link in every closing.
   - Preserve the compact table format as the selective expanded representation.
2. Extend `internal/install/harness_smoke_test.go` (or a focused test beside it).
   - Install the engineering domain for all six targets.
   - Assert each target's native design-command surface contains a stable phrase
     from the progressive-disclosure contract.
   - Assert the obsolete unconditional-table phrase is absent from every target.
   - Cover Codex's command-as-skill output as well as the five direct command
     renderers, preventing a single-harness fix.

## Boundaries

- Do not change acceptance-criteria authoring, parsing, EARS syntax, scoring,
  verification, or delivery gates.
- Do not shorten or remove the full `## Acceptance Criteria` section in specs.
- Do not add client UI, foldout components, HTML `<details>`, configuration, or a
  new CLI/MCP surface.
- Do not change other workflow closings (`/diagnose`, `/deliver`, PM story
  authoring) in this enhancement; they may adopt the pattern separately after
  their approval semantics are examined.
- Do not edit generated installed copies such as
  `.agents/skills/command-design/SKILL.md`; propagation comes from the canonical
  embedded source through `hero install`.

## Risks

- Theme summaries are model-authored and could become vague. The instruction must
  bound them to concrete behavior areas rather than generic labels such as
  “functionality” or “quality.”
- “Formal approval” can be over-read as every design completion. The rule must
  tie expansion to an explicit user request to review or approve the AC contract.
- A large refinement delta can recreate the original wall of text. The five-item
  delta bound and `+N more in spec` behavior prevent that regression.
- Exact prose assertions can make install tests brittle. Use one stable contract
  phrase and one obsolete-phrase absence check rather than snapshotting the full
  command.

## Acceptance Criteria

- AC-1: WHEN `/design` closes with more than five acceptance criteria and no explicit full-contract review request, THE SYSTEM SHALL summarize the contract with its total count and 2–5 concrete coverage themes instead of rendering one row per criterion.
- AC-2: WHEN acceptance criteria changed during the current design loop, THE SYSTEM SHALL identify up to five changed AC IDs with short behavior phrases, or summarize the count and affected themes when more than five changed.
- AC-3: WHEN `/design` creates an initial contract, THE SYSTEM SHALL label it as an initial contract rather than enumerate every new criterion as the current-loop delta.
- AC-4: WHEN any criterion is ambiguous, not independently testable, deferred, conflicting, or has an unresolved decision, THE SYSTEM SHALL surface an attention line naming the affected criterion and issue.
- AC-5: WHEN a contract has five or fewer criteria, the user asks to see all criteria, or the user explicitly requests review or approval of the AC contract, THE SYSTEM SHALL render the compact `AC | Pattern | Behavior` table.
- AC-6: THE SYSTEM SHALL link the spec containing the complete acceptance-criteria contract in every `/design` closing and preserve the score and next-step recommendation.
- AC-7: WHEN Hero installs the engineering domain for any supported target, THE SYSTEM SHALL propagate the same progressive-disclosure closing contract to that target's native design-command surface, including Codex's command-as-skill representation.
- AC-8: THE SYSTEM SHALL NOT require client-specific foldout markup, UI support, configuration, or changes to acceptance-criteria persistence and verification.

## Validation

- Run the focused install test covering `opencode`, `cursor`, `claude`,
  `copilot`, `codex`, and `generic` command destinations.
- Run `go test ./internal/install` to catch target rendering and contract
  regressions.
- Inspect one direct command output and Codex's generated
  `.agents/skills/command-design/SKILL.md` in test fixtures to confirm the same
  semantic instruction is present and the obsolete unconditional table mandate
  is absent.
- Run `hero spec score progressive-design-ac-recap` and address structural
  warnings before delivery.

## Completion Ledger

Implemented the progressive `/design` closing contract in the canonical engineering command and verified that the real embedded engineering domain propagates it to every supported harness. Loaded `agent-reliability`, `implementation-principles`, `testing-and-validation`, `go-stack`, and `completion-ledger`. Validation passed: focused six-target install test, full `internal/install` package, practical generic and Codex CLI installs, direct output inspection, and spec scoring (88/100, deliverable; no structural warning).

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Large contracts use count and 2–5 concrete themes | DONE | `domains/engineering/commands/design.md:38` — defines the bounded default recap instead of one row per AC. |
| 2 | Current-loop delta names up to five ACs or summarizes larger deltas | DONE | `domains/engineering/commands/design.md:38` — defines both bounded delta forms. |
| 3 | Initial contracts are labeled rather than enumerated as deltas | DONE | `domains/engineering/commands/design.md:38` — requires `initial contract created`. |
| 4 | Attention-worthy exceptions name the affected AC and issue | DONE | `domains/engineering/commands/design.md:40` — requires a conditional `Attention` line and enumerates qualifying issues. |
| 5 | Short or explicitly requested contracts retain the compact table | DONE | `domains/engineering/commands/design.md:42` — defines all three expansion triggers and retains the table at lines 44–48. |
| 6 | Every closing links the full spec and preserves score and next step | DONE | `domains/engineering/commands/design.md:40` — explicitly requires the full-contract link, score, and recommendation. |
| 7 | Contract propagates to all six native command surfaces | DONE | `internal/install/harness_smoke_test.go:182` — real-domain install coverage asserts the stable contract and obsolete-rule absence for all six targets, including Codex. |
| 8 | No client foldout, configuration, or AC persistence changes | DONE | Diff is limited to canonical workflow prose, propagation coverage, and this delivery ledger; practical installs confirm content-only propagation. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Update canonical `design.md` closing contract | DONE | `domains/engineering/commands/design.md:38` — adds count, themes, bounded delta, exception visibility, selective expansion, and full-spec-link rules. |
| 2 | Add all-six-target propagation coverage | DONE | `internal/install/harness_smoke_test.go:182` — installs the real engineering overlay and checks every native design-command destination. |

### Exercise-the-feature check

- [x] User-visible behavior was exercised end-to-end: `go run ./cmd/hero install project /private/tmp/hero-ac-recap-{generic,codex}-20260720 --target <target> --only-target --domain engineering --force --no-hooks`; both installs completed with 120 files, and `rg` confirmed the progressive contract plus selective-expansion rule in generic `.ai/commands/design.md` and Codex `.agents/skills/command-design/SKILL.md`, with no obsolete unconditional-table phrase.

### Excellence Bar self-check

Honest answer to “would a senior engineer who cares about this codebase be proud to ship this?” — yes. The instruction is deterministic and bounded, the test exercises real embedded content across every supported renderer rather than a synthetic fixture, and package-level regression coverage passes.
