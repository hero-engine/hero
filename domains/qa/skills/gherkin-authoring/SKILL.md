---
name: gherkin-authoring
description: Express business behavior in declarative Gherkin scenarios that remain readable and binding-friendly.
---
# Gherkin authoring

Use Feature for the capability, Rule when several policies need grouping, Scenario
for one behavioral example, and Scenario Outline for meaningful parameter sets.
Given establishes relevant state, When names one business action, and Then states
observable outcomes. Use And only within the same semantic phase. Avoid selectors,
click sequences, waits, and implementation details. Reuse phrasing deliberately so
automation can bind stable steps without making scenarios generic.

