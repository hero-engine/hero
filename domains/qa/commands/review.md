---
description: Review a QA artifact or release gate for traceability, rigor, and evidence quality.
---
# QA review router

Route plans, cases, suites, and triage artifacts to `qa-reviewer`. Route release
gates to `release-gate-reviewer`. The reviewer reads the artifact cold, cites named
findings, and returns an explicit verdict without rewriting the subject. Check
scope, criteria traceability, meaningful technique coverage, executability,
evidence freshness, unsupported assumptions, and policy application.

Request: $ARGUMENTS

