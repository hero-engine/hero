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

The first initiative passed many child gates while its orchestration text stayed
stale and its branch kept accumulating unrelated work. This child exists to
make integration truth and scope provenance explicit before merge.

## Goal

Prove the complete original outcome on the clean successor branch, reject any
untraceable production change, run an independent cold audit, and close the
initiative only when its recorded status matches verified reality.

## Design brief

The `/design` pass must define a non-feature gate that:

1. Maps every production diff hunk to a parent acceptance criterion and rejects
   every item named in the parent's Boundaries section.
2. Runs the initiative-wide prompt policy, flag compatibility, connect resolver,
   six-target parity, and selector behavior matrices.
3. Runs the full suite, affected-package race tests, vet, build, Windows runtime
   or platform-seam validation, spec linting, and diff hygiene checks.
4. Runs a fresh cold delivery audit against the curated branch.
5. Reconciles initiative progress and evidence, then runs
   `hero spec verify interactive-cli-input-scoped-completion`.

## Boundaries

This child introduces no production feature and performs no opportunistic
repair. Any failure routes back to `prompt-and-tty-contract-closure`,
`interactive-setup-and-connect-closure`, or `corpus-selector-closure`. All
parent initiative exclusions apply verbatim.

## Acceptance targets

- No production change lacks an original-scope acceptance criterion.
- All automated and platform validation passes on the clean successor branch.
- The cold audit reports no unresolved correctness, regression, or scope blocker.
- Progress text, child state, completion ledger, audit, and code agree.
- Hero verification passes before the initiative is marked completed.

## Kickoff

Design the non-feature closing gate after both adoption children verify. Build
one scope-provenance map and one initiative-wide validation matrix, then require
full tests, race, vet, build, Windows evidence, cold audit, truthful progress,
and Hero verification. Do not fix production code here or absorb excluded work;
route failures back to the child that owns the violated original contract.

→ `/design interactive-cli-acceptance-and-merge-gate`
