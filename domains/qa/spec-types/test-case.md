---
title: Test Case
type: test-case
domain: qa
category: work
bucket: test-cases
location: .hero/planning/test-cases/{slug}/spec.md
lifecycle:
  states: [drafted, ready, automated, retired]
  initial: drafted
  terminal: [retired]
frontmatter:
  required:
    - { name: title, type: string, classification: content }
    - { name: type, type: enum, values: [test-case], classification: content }
    - { name: status, type: enum, values: [drafted, ready, automated, retired], classification: org-state }
    - { name: format, type: enum, values: [step, gherkin, decision-table, data-driven, charter], classification: content }
  optional:
    - { name: verifies, type: "ref(feature)", classification: content }
    - { name: test_plan, type: "ref(test-plan)", classification: content }
    - { name: risk, type: string, classification: content }
    - { name: evidence_as_of, type: date, classification: org-state }
sections:
  required: [Purpose, Traceability, Preconditions, Procedure, Expected Results]
  optional: [Data, Environment, Automation, Evidence, Cleanup]
default_agents:
  authoring: test-author
  review: qa-reviewer
---
# Test case spec type

A test case is a durable behavioral specification. It traces to an acceptance
criterion, risk, or charter mission and states enough setup, procedure, and expected
result for another practitioner to execute it. Run results are evidence linked to
the case; they do not overwrite its authored purpose.

