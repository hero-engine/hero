# Hero QA — Evidence-Driven Quality Assurance

The QA pack turns requirements into inspectable coverage and release evidence. It
works locally and offline: external test-management systems may decorate evidence,
but they never define whether a workflow is available.

## Operating rules

1. Trace every authored case to an acceptance criterion, risk, or named charter.
2. Distinguish coverage from test count: uncovered behavior stays visible.
3. Treat flaky behavior as work with an owner and verdict, never background noise.
4. Make release gates policy-based and evidence-backed; name every waiver.
5. Propose cross-domain changes. Humans confirm story rejection, bug creation,
   regression promotion, and release decisions.
6. Prefer local specifications and recorded evidence. Integrations are optional.

## Natural-language routing

- Coverage strategy, risk, or scope → `qa-strategist`.
- Cases from requirements → `test-author`; Gherkin, decision tables, and charters
  route to their format specialist.
- Test plans → `plan-author`.
- Failed tests or defects → `test-issue-triager`.
- Intermittent failures → `qa-flake-curator`.
- Regression membership → `regression-curator`.
- Release readiness → `release-readiness-strategist`, then
  `release-gate-reviewer`.
- Missing testability seam → `seam-requester`.
- QA artifact review → `qa-reviewer`.
- Workspace hygiene → the QA scrubbers through `/scrub-qa`.

Use `qa-delivery-lead` when the request crosses more than one of these concerns.

