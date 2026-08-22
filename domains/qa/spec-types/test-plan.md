---
title: Test Plan
type: test-plan
domain: qa
category: work
bucket: test-plans
location: .hero/planning/test-plans/{slug}/spec.md
lifecycle:
  states: [draft, committed, in-flight, completed, superseded]
  initial: draft
  terminal: [completed, superseded]
frontmatter:
  required:
    - { name: title, type: string, classification: content }
    - { name: type, type: enum, values: [test-plan], classification: content }
    - { name: status, type: enum, values: [draft, committed, in-flight, completed, superseded], classification: org-state }
    - { name: scope, type: string, classification: content }
  optional:
    - { name: release, type: "ref(release)", classification: content }
    - { name: features, type: "list[ref(feature)]", classification: content }
    - { name: owner, type: string, classification: org-state }
    - { name: evidence_as_of, type: date, classification: org-state }
sections:
  required: [Objective, Scope, Risks, Coverage Matrix, Environments and Data, Exit Conditions]
  optional: [Out of Scope, Responsibilities, Schedule, Evidence, Test Seams]
default_agents:
  authoring: plan-author
  review: qa-reviewer
---
# Test plan spec type

A test plan defines the quality evidence a feature, sprint, or release needs. Its
coverage matrix maps acceptance criteria and risks to cases, charters, suites, or
explicit gaps. `completed` means the planned evidence was reconciled and the exit
conditions were evaluated; it does not imply every test passed.

