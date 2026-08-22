---
name: flake-verdict-classification
description: Use failure-shape heuristics to distinguish test, environment, and product sources of intermittent behavior.
---
# Flake verdict classification

Locator drift, order dependence, leaked state, and assertion races point toward the
test. Resource exhaustion, unavailable dependencies, clock skew, and infrastructure
variance point toward the environment. Deterministic invariant violations,
concurrency races, and data-dependent behavior may indicate a product defect.
These are hypotheses, not verdicts: cite the observations, disconfirm competing
causes, and state what experiment would raise confidence.

