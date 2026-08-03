---
title: "Interactive CLI Acceptance and Merge Gate"
slug: interactive-cli-acceptance-and-merge-gate
type: feature
status: planning
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

Design the non-feature closing gate after both adoption children verify. Build
the scope, donor-disposition, and acceptance maps; then require full tests,
race, vet, build, Windows and live-pipe evidence, cold audit, truthful progress,
and both Hero verification gates. Do not fix production code here or lose valid
side fixes—route original-contract failures back and extracted work to its named
follow-up.

→ `/design interactive-cli-acceptance-and-merge-gate`
