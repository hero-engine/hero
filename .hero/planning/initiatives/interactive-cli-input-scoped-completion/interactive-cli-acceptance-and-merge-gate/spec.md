---
title: "Interactive CLI Acceptance and Merge Gate"
slug: interactive-cli-acceptance-and-merge-gate
type: feature
status: delivering
created: 2026-08-03
domain: engineering
size: small
priority: high
parent: interactive-cli-input-scoped-completion
depends-on:
  - interactive-setup-and-connect-closure
  - corpus-selector-closure
tags: [cli, validation, audit, merge-readiness]
---

# Interactive CLI Acceptance and Merge Gate

## Context

The donor initiative passed many child gates while its orchestration text stayed
stale and its branch accumulated unrelated work. Package tests passing was not
enough to expose Windows secret failure, live-pipe blocking, or selectors that
refused Hero-sized corpora. This child makes integration truth, donor-change
disposition, and scope provenance explicit before merge.

## Goal

Prove the complete original outcome on the clean successor branch, reject any
untraceable production change, run an independent cold audit, and close the
initiative only when its recorded status matches verified reality.

## Approach

This is a non-feature closing gate. It may add or strengthen tests, validation
scripts, the Completion Ledger, delivery audit, and progress evidence. It may
not change production behavior. If validation finds a production defect, reopen
the owning child, repair it there, re-verify that child, and restart this gate.

The gate uses three evidence maps:

1. **Scope provenance:** every production diff hunk maps to a parent AC and one
   owning child.
2. **Donor disposition:** every donor commit/change is marked ported, extracted
   to named follow-up work, already present on current `main`, or deliberately
   dropped with a reason in `donor-branch-disposition.md`.
3. **Acceptance coverage:** every parent and child AC maps to an executable
   test or a named platform/manual check with captured evidence.

## Changes

1. Complete the scope-provenance and donor-disposition maps beside this spec.
   - Compare `main...HEAD` by file and hunk, not only by commit subject.
   - Reject index, guided init, alias, global invocation guard, timeout policy,
     graph/spec, unrelated uninstall/Codex, new-selector, and form-engine work.
   - Do not delete the donor branch until every salvageable change has a named
     destination.
2. Build an initiative-wide test ledger covering:
   - prompt-site TTY/closed/live-pipe/JSON/NEVER-PROMPT behavior;
   - the two prompt-policy corrections and two connect corrections;
   - flag-driven golden compatibility;
   - secure secret platform seams, including Windows runtime evidence;
   - connect role/capability/default equivalence and resolver acceptance;
   - six install/uninstall targets;
   - every selector target under empty, single, 25, 26+, cancellation,
     invalid, explicit, non-TTY, and JSON conditions.
3. Run repository validation on the clean successor branch.
   - `go test -count=1 -timeout 10m ./...`
   - affected-package race tests
   - `go vet ./...` and `go build ./...`
   - Windows cross-build plus the platform-specific secret runtime/seam tests
   - `git diff --check`, Hero spec lint/score, and drift checks
4. Produce the Completion Ledger and commission a fresh cold delivery audit
   using only the on-disk spec, diff, ledger, and test evidence.
5. Reconcile the parent Progress section and all child evidence, then run
   `hero spec verify interactive-cli-acceptance-and-merge-gate` followed by
   `hero spec verify interactive-cli-input-scoped-completion`.

## Boundaries

This child introduces no production feature and performs no opportunistic
repair. Any failure routes back to `prompt-and-tty-contract-closure`,
`interactive-setup-and-connect-closure`, or `corpus-selector-closure`. All
parent initiative exclusions apply verbatim. It does not deliver the extracted
side-quest follow-ups; it only proves they have not been lost.

## Risks

1. A green full suite can still miss terminal runtime semantics. Require the
   explicit live-pipe and platform evidence rather than inferring it.
2. The donor ledger can become performative if groups are marked "separate"
   without a resolvable spec or explicit drop rationale.
3. Audit findings can restart scope growth. Route only original-contract
   failures back into the initiative.
4. Generated Hero projections can make the diff look broad. Separate generated
   artifacts from production provenance while still committing required
   projections.

## Acceptance Criteria

- **AC-1:** THE SYSTEM SHALL map every production diff hunk to a parent acceptance criterion and one owning child.
- **AC-2:** IF a production change falls within a parent boundary exclusion THEN THE SYSTEM SHALL remove it from the successor branch before merge.
- **AC-3:** THE SYSTEM SHALL classify every donor-branch change as ported, extracted to a named follow-up, already present on `main`, or deliberately dropped with a recorded rationale.
- **AC-4:** THE SYSTEM SHALL NOT delete the donor branch while any salvageable change lacks a durable destination.
- **AC-5:** WHEN the prompt-policy matrix runs THE SYSTEM SHALL prove TTY, closed-input, live-pipe, JSON, NEVER-PROMPT, and secure-secret behavior for every owned path.
- **AC-6:** WHEN connect validation runs THE SYSTEM SHALL prove interactive and flag-driven role/capability/default equivalence through the effective resolver.
- **AC-7:** WHEN target-parity validation runs THE SYSTEM SHALL prove install and uninstall behavior for all six harness targets.
- **AC-8:** WHEN selector validation runs THE SYSTEM SHALL cover every named target across empty, single, large, cancellation, invalid, explicit, non-TTY, and JSON cases.
- **AC-9:** THE SYSTEM SHALL pass the full suite, affected race tests, vet, build, diff hygiene, Hero lint/score/drift, and Windows platform evidence on the clean successor branch.
- **AC-10:** THE SYSTEM SHALL produce a fully `DONE` Completion Ledger and a fresh cold-audit `SHIP` verdict before either verification command runs.
- **AC-11:** IF this gate discovers a required production repair THEN THE SYSTEM SHALL return it to the owning child and SHALL NOT implement it inside the gate.
- **AC-12:** WHEN the initiative completes THE SYSTEM SHALL keep parent progress, child status, test evidence, audit evidence, and code state consistent.

## Validation

The Changes list is the validation procedure. Store command outputs or concise
evidence references beside the Completion Ledger so the cold auditor can
reproduce claims. A compile-only Windows result does not satisfy secure-secret
runtime evidence. A closed `strings.Reader` does not satisfy live-pipe liveness.
The gate fails on either substitution.

## Kickoff

Closes the interactive CLI initiative with a hunk map, donor disposition, and
integrated validation evidence; it makes no production change.

**Status:** delivering — the first cold audit found artifact-only truth gaps;
the bounded evidence repair is complete and awaits a fresh audit.

**Pick up at:** run a fresh cold audit against the repaired evidence commit,
then run both Hero verification gates only if it reports SHIP.

→ `hero spec verify interactive-cli-acceptance-and-merge-gate --skip-tests`

**Files:** `.hero/planning/initiatives/interactive-cli-input-scoped-completion/interactive-cli-acceptance-and-merge-gate/scope-provenance.md`, `.hero/planning/initiatives/interactive-cli-input-scoped-completion/donor-branch-disposition.md`, `.hero/planning/initiatives/interactive-cli-input-scoped-completion/interactive-cli-acceptance-and-merge-gate/acceptance-evidence.md`
**Skip:** production repairs and donor branch deletion; route either to its owning child or retained donor destination.

## Completion Ledger

Stack detected: Go. This non-production gate loaded the delivery, Go,
reliability, validation, and ledger guidance. It compared zero-context
`main...HEAD` hunks, reconciled every donor commit group, and records the
integrated command matrix in the evidence artifacts.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Every production hunk has a parent AC and owner | DONE | scope-provenance.md records 229 production hunks by file, hunk range, parent AC, and final child owner; four portable PTY hunks are separately classified as test-only infrastructure. |
| 2 | Boundary-excluded production work is absent before merge | DONE | The provenance boundary scan reports no index, init, alias, invocation-guard, timeout, graph/spec-terminal, form-engine, or new-selector production hunk. |
| 3 | Every donor change has a durable disposition | DONE | donor-branch-disposition.md reconciles every group in main..design/interactive-cli-input as Ported, Extracted donor-retained, Already present, or deliberately dropped with a reason. |
| 4 | Donor stays while salvageable work lacks a local destination | DONE | The disposition map names exact retained-donor paths and commits for every absent local follow-up and expressly prohibits donor deletion. |
| 5 | Prompt policy matrix proves terminal, pipe, JSON, NEVER-PROMPT, and secrets | DONE | acceptance-evidence.md maps prompt package, stream, baseline, JSON, policy, live-pipe, and protected-terminal seam tests. |
| 6 | Connect equivalence reaches the effective resolver | DONE | Resolver, paired persistence, role routing, JSON, and live-pipe tests are mapped in acceptance-evidence.md. |
| 7 | All six targets have install/uninstall proof | DONE | Six-target picker parity plus four existing uninstall paths and real Copilot/generic manifest round trips are mapped in acceptance-evidence.md. |
| 8 | Every selector target has the required interaction matrix | DONE | selector_test covers frozen targets, empty/single/25/26/250, cancellation, invalid, explicit, non-TTY, and JSON paths. |
| 9 | Repository, platform, and Hero validation pass | DONE | Full suite, affected CLI race test, vet, native and Windows builds, portable Windows secret seam, live pipe, diff check, lint, score, and drift are recorded as passing in this delivery. |
| 10 | Fully DONE ledger and fresh SHIP audit precede verification | BLOCKED | Per parent-owned closing scope, the fresh cold audit and both verification commands must be run by the root agent after this evidence commit; they were intentionally not run here. |
| 11 | Production defects return to their owning child | DONE | The hunk and test review found no production defect; this gate made no production-code edit. |
| 12 | Parent/child progress, tests, audit, and code agree | DONE | Parent Progress, this kickoff, archived child audits, provenance, donor map, and acceptance evidence now identify the same successor state. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Complete scope-provenance and donor-disposition maps | DONE | scope-provenance.md covers the final diff by hunk; donor-branch-disposition.md reconciles every donor group and retained destination. |
| 2 | Build initiative-wide test ledger | DONE | acceptance-evidence.md maps the prompt, connect, target-parity, selector, secret-platform, and liveness evidence. |
| 3 | Run clean-successor validation matrix | DONE | Full normal/race/vet/build/Windows/platform-seam/live-pipe/diff/Hero checks passed; commands are recorded in this delivery's evidence. |
| 4 | Produce ledger and commission a cold audit | DONE | The independent on-disk delivery-audit.md records that the cold audit was commissioned and returned HOLD for artifact-only truth corrections. |
| 5 | Reconcile progress and run both Hero verify commands | BLOCKED | Progress is reconciled by this repair; root must obtain a fresh SHIP audit and run both verification commands after this commit. |

### Exercise-the-feature check

- [x] User-visible behavior was exercised end-to-end: the full CLI suite plus
  focused prompt live-pipe, protected-secret seam, setup/connect, uninstall,
  and selector matrices exercised the built command paths and their persisted
  state; the results are recorded in acceptance-evidence.md.

### Excellence Bar self-check

Honest answer to "would a senior engineer who cares about this codebase be
proud to ship this?" — yes for this gate's owned evidence work: it makes scope,
donor retention, platform limits, and the remaining root-owned gate explicit
instead of treating passing package tests as merge proof.

### Delivered artifacts

- `.hero/planning/initiatives/interactive-cli-input-scoped-completion/interactive-cli-acceptance-and-merge-gate/spec.md` — final-gate ledger, status, and refreshed kickoff.
- `.hero/planning/initiatives/interactive-cli-input-scoped-completion/interactive-cli-acceptance-and-merge-gate/scope-provenance.md` — production hunk-to-AC/owner map.
- `.hero/planning/initiatives/interactive-cli-input-scoped-completion/interactive-cli-acceptance-and-merge-gate/acceptance-evidence.md` — parent and gate acceptance-to-test map.
- `.hero/planning/initiatives/interactive-cli-input-scoped-completion/donor-branch-disposition.md` — final donor reconciliation and retention constraints.
- `.hero/planning/initiatives/interactive-cli-input-scoped-completion/spec.md` — reconciled initiative progress.
