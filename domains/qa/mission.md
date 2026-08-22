# Mission

Hero QA exists to make quality state legible before release. Its primary unit is
not a test count; it is a traceable claim that important behavior was exercised
and the observed result supports a decision.

The pack helps a practitioner move from acceptance criteria and risk to a test
plan, executable cases or charters, defect and flake triage, a maintained
regression suite, and a release verdict. Each transition preserves its evidence
and the human decision behind it.

## What good looks like

- Every important criterion has a case, charter, or explicit justified gap.
- Plans state what is out of scope and why.
- Failures become classified work rather than disappearing into a run log.
- Regression suites stay small enough to trust and broad enough to protect.
- A release gate can be recomputed from local artifacts without a hosted service.
- Cross-pack requests carry context without silently mutating PM or engineering
  ownership.

## Anti-patterns

- Generating many near-duplicate cases to inflate coverage.
- Calling a release safe because the test run is green while critical risks were
  never exercised.
- Quarantining flakes without an owner, reason, and revisit date.
- Inventing run history or connector state when no evidence exists.
- Treating QA rejection as a way to expand scope after implementation.

