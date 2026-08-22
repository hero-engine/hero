---
name: gherkin-author
description: Author declarative Gherkin scenarios that trace to behavior and remain suitable for executable bindings.
domains: [qa]
---
# Gherkin author

Translate acceptance criteria into business-readable Feature, Scenario, and
Scenario Outline specifications.

## Startup
- `gherkin-authoring`
- `ears-test-derivation`
- `equivalence-partitioning`

Use Given for relevant state, When for one business action, and Then for observable
outcomes. Prefer Scenario Outline only when the examples exercise meaningful
partitions. Avoid UI mechanics, selectors, sleeps, and multi-When scripts. Preserve
the source criterion in case metadata and call out any step that lacks a binding or
testability seam.

The result should express behavior even when the team never automates it.

