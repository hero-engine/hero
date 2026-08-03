---
title: "Interactive CLI Input — Scoped Completion"
slug: interactive-cli-input-scoped-completion
type: initiative
status: planning
created: 2026-08-03
domain: engineering
size: x-large
priority: critical
autonomy: guided
tags: [cli, refactor, prompt, tty, interactive, recovery]
child:
  - prompt-and-tty-contract-closure
  - interactive-setup-and-connect-closure
  - corpus-selector-closure
  - interactive-cli-acceptance-and-merge-gate
relates-to: [cli-surface-consolidation]
---

# Interactive CLI Input — Scoped Completion

## Vision

Hero's human-facing CLI should ask for missing values when a person is at a
terminal, while flag-driven, piped, JSON, and agent-facing invocations remain
deterministic and never block. One shared prompt layer should own interactive
input and terminal classification. Setup commands should guide people through
the values they would otherwise have to look up, and selector commands should
offer useful choices from the local corpus even in a real Hero-sized workspace.

This is the original `interactive-cli-input` outcome, not a reduced recovery
scope. The successor exists because the first initiative was marked complete
while acceptance gaps remained and its branch accumulated unrelated repairs.
The archived initiative is historical evidence; this spec is the current source
of progress truth.

## Goal

Produce one curated, merge-ready implementation whose production changes all
trace to the original interactive-input outcome. The completed result has one
cross-platform prompt authority, safe TTY and secret handling, additive setup
prompts, selectors that work at actual corpus scale, unchanged machine paths,
and evidence that no affected invocation can prompt or hang unexpectedly.

## Starting point

`design/interactive-cli-input` is an implementation candidate and evidence
source, not the merge base. It is 52 commits ahead of `main` and continued well
beyond the original selector delivery into index, uninstall/config, `init`,
alias, timeout, and invocation-lint work.

Delivery starts from a clean branch based on `main`. Each child selectively
ports only its owned original-scope changes, then closes the corresponding
acceptance gaps. If a scoped patch needs a helper introduced by a side quest,
port the minimum helper rather than importing the side quest.

Valid side discoveries are preserved without entering this merge through
[`donor-branch-disposition.md`](donor-branch-disposition.md). The donor branch
stays intact until the final gate confirms every change was ported, extracted to
named follow-up work, already present on `main`, or deliberately dropped with a
reason.

### Current truth

| Original outcome | Candidate state | Acceptance state |
|---|---|---|
| Six-target uninstall parity | Implemented | Candidate only; validate real removal for all six targets |
| Shared prompt package and prompt-site adoption | Mostly implemented | Incomplete: Windows secret input, remaining reader ownership, and liveness proof |
| Connect writer and `code-host` role | Implemented | Incomplete: non-TTY interactive path can prompt or block |
| PROMPT / SELECTOR / NEVER-PROMPT classification | Active convention | Delivered input to the successor |
| Guided setup prompts | Implemented | Incomplete until every site obeys missing-value + TTY gating |
| Corpus selectors | Implemented | Incomplete: the 25-item ceiling makes them inert in this repository |
| Connect help | Implemented | Candidate only; validate against the curated implementation |

The branch is functionally advanced, but the initiative is not complete and the
branch is not merge-ready.

## Specs

| Phase | Slug | Outcome | Size | Priority | Depends on | Status |
|---|---|---|---|---|---|---|
| 1 | `prompt-and-tty-contract-closure` | Curate and close the shared prompt, stream, TTY, secret, and machine-path contract | `large` | `critical` | — | planning |
| 2A | `interactive-setup-and-connect-closure` | Curate and close the original setup prompts, connect semantics, target parity, and help | `large` | `high` | prompt/TTY closure | planning |
| 2B | `corpus-selector-closure` | Curate the original selector targets and make them useful beyond 25 candidates | `medium` | `high` | prompt/TTY closure | planning |
| 3 | `interactive-cli-acceptance-and-merge-gate` | Prove the complete outcome, cold-audit the curated diff, and make the merge decision | `small` | `high` | setup/connect + selectors | planning |

## Delivery path

### Phase 1 — foundation

Deliver `prompt-and-tty-contract-closure` first. It owns the prompt package and
all original prompt-site migrations, so no adoption work proceeds against a
half-stable contract.

### Phase 2 — bounded adoption

After the foundation verifies, deliver `interactive-setup-and-connect-closure`
and `corpus-selector-closure`. They may proceed in parallel only while ownership
stays strict:

- setup/connect does not grow the prompt package or selector infrastructure;
- selectors do not modify connect, setup, uninstall, or prompt primitives.

If either design introduces a shared-file seam between the two, add reciprocal
`conflicts-with` relations to both child specs and record the seam here before
delivery. There is no speculative mutex today.

### Phase 3 — truth gate

`interactive-cli-acceptance-and-merge-gate` introduces no production feature.
It validates the integrated result, rejects scope drift, runs the cold audit,
reconciles progress claims, and closes only after Hero verification passes. A
production defect found here returns to its owning child; the gate does not
quietly absorb another repair initiative.

## Scope rule

Classify every audit finding before acting:

1. If it violates the original interactive CLI contract, it stays.
2. If it exists only because of an unrelated branch addition, exclude that
   addition from the curated branch.
3. If it is pre-existing and unrelated, leave it in separate backlog work.

No audit finding is automatic permission to expand this initiative.

## Boundaries

The following do not belong in this initiative or its merge branch:

- CLI surface consolidation or command-tree reshaping
- alias flag/argument parity, alias derivation, or alias message work
- global Go/Markdown invocation-string guards or the dead `hero verify` audit trail
- guided `hero init` or first-run setup expansion
- index repartition, retrieval, or search repair
- convention, decision, graph, or spec-terminal side quests
- uninstall managed-block, shared-root, or Codex line-welding repairs beyond
  the removal behavior required for six-target parity
- test timeout or package-headroom programs
- exact-slug retrieval changes
- new selector targets beyond the original committed target set
- a generalized form engine, field-type registry, schema registry, or plugin API
- network-backed selectors

`cli-surface-consolidation`, `alias-pair-derivation`,
`alias-parity-message-assertion`, and `cli-test-package-headroom` remain separate
planning work and do not gate this initiative.

## Acceptance Criteria

- **AC-1:** THE SYSTEM SHALL use `internal/cli/prompt` as the sole authority for interactive input and terminal classification within `internal/cli`, with no hardcoded or mutable package-global interactive reader remaining.
- **AC-2:** THE SYSTEM SHALL keep input-terminal and output-terminal checks semantically distinct and SHALL reject ordinary character devices such as `/dev/null` as proof of an interactive terminal.
- **AC-3:** WHEN required human input is absent and stdin is closed or is a live non-TTY pipe THE SYSTEM SHALL fail promptly with no prompt and no wait for EOF.
- **AC-4:** WHEN a secret is requested THE SYSTEM SHALL use a secure platform-specific terminal implementation and SHALL never fall back to echoed input.
- **AC-5:** WHEN a fully supplied flag- or argument-driven invocation runs THE SYSTEM SHALL preserve its baseline stdout, stderr, and exit status except for the explicitly enumerated prompt-policy and connect corrections.
- **AC-6:** WHERE `--json` IS ENABLED THE SYSTEM SHALL NOT prompt.
- **AC-7:** THE SYSTEM SHALL NOT prompt on any command classified NEVER-PROMPT.
- **AC-8:** WHEN interactive and flag-driven connect paths receive equivalent values THE SYSTEM SHALL persist equivalent role, capability, and default state, and a `code-host` connection SHALL resolve successfully.
- **AC-9:** THE SYSTEM SHALL expose the same six targets through install and uninstall and SHALL provide a working removal path for each target.
- **AC-10:** WHEN any original selector target has more than 25 local candidates THE SYSTEM SHALL allow the user to reach the full candidate set through bounded interaction without silent truncation.
- **AC-11:** IF a selector is cancelled THEN THE SYSTEM SHALL exit non-zero without mutation.
- **AC-12:** WHEN a selector target runs with an explicit argument, non-TTY input, or a machine-facing mode THE SYSTEM SHALL preserve the non-interactive path and SHALL NOT show a picker.
- **AC-13:** THE SYSTEM SHALL keep the curated production diff traceable to these acceptance criteria and SHALL contain none of the work listed in Boundaries.
- **AC-14:** WHEN the final child completes THE SYSTEM SHALL have passed the full test suite, affected-package race tests, vet, build, Windows runtime/platform evidence, a fresh cold delivery audit, and `hero spec verify interactive-cli-input-scoped-completion`.

## Sanctioned behavior corrections

These are the finite compatibility exceptions. Do not create an open-ended
allowance for additional behavior changes.

1. `hero install project` without a TTY and without `--target` fails instead of
   silently selecting `opencode`.
2. Password entry without a TTY refuses echoed input and directs automation to
   the supported non-interactive mechanism.
3. `hero connect ... --role code-host` follows the supplied role and writes the
   corresponding capability/default state instead of falling into a writer that
   ignores the role.
4. Missing required connect fields on non-TTY input fail promptly rather than
   consuming or waiting on a pipe.

## Validation strategy

- Reuse the original golden prompt fixtures only after checking that each
  fixture represents the pre-change baseline and one of the four corrections
  above is not being normalized away.
- Test every prompt site through Cobra's configured streams.
- Use both closed readers and a live, unanswered non-TTY pipe with a bounded
  completion assertion; EOF-only tests do not prove liveness.
- Exercise secure secret input at the platform seam. A Windows cross-compile is
  necessary but is not runtime evidence.
- Verify connect through the effective resolver, not by inspecting JSON alone.
- Exercise every named selector with empty, single-item, large, cancellation,
  invalid-choice, explicit-argument, non-TTY, and machine-mode cases.
- Run `go test -count=1 -timeout 10m ./...`, affected-package race tests,
  `go vet ./...`, `go build ./...`, Windows validation, spec linting, a cold
  audit, and the Hero verification gate on the clean successor branch.

## Risks

1. Selective porting can accidentally import a side quest through a shared
   commit. Review the resulting diff by behavior and file ownership, not by
   commit label.
2. A half-ported prompt layer recreates the original fork problem. Phase 1 must
   port and verify the complete original prompt-site set before adoption starts.
3. Terminal behavior differs at runtime across Unix and Windows. Compile-only
   evidence can hide a broken secret path.
4. Selector scale can invite a TUI rewrite. The requirement is bounded access
   to the full local corpus, not a rendering framework.
5. The final gate can become another catch-all audit. It must route production
   defects back to an owner and reject unrelated work.

## Progress

- Original implementation and audit evidence exist on
  `design/interactive-cli-input`.
- Original input classification exists as active convention
  `cli-input-classification`.
- The prompt/TTY, setup/connect, and selector children are archived as completed
  after independent SHIP audits and their verification gates.
- The final gate is delivering: scope provenance maps the 233 production hunks;
  donor-only follow-ups are retained at explicit donor paths rather than falsely
  claimed as local successor specs.
- Integrated validation evidence is recorded by
  interactive-cli-acceptance-and-merge-gate. Its remaining actions are a fresh
  cold audit and both Hero verification gates; the donor branch must remain.

## Kickoff

Finish the original interactive CLI outcome without carrying forward its branch
drift. Treat `design/interactive-cli-input` as a selective donor only. Start on
clean `main` with `prompt-and-tty-contract-closure`; port the complete shared
prompt foundation, close cross-platform TTY and liveness gaps, and verify it
before setup or selector adoption. Do not import index, init, alias, timeout,
invocation-lint, graph, or unrelated uninstall work.

→ `/drive interactive-cli-input-scoped-completion`
