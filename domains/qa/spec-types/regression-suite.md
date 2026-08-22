---
title: Regression Suite
type: regression-suite
domain: qa
category: work
bucket: regression-suites
location: .hero/planning/regression-suites/{slug}/spec.md
lifecycle:
  states: [active, under-review, deprecated]
  initial: active
  terminal: [deprecated]
frontmatter:
  required:
    - { name: title, type: string, classification: content }
    - { name: type, type: enum, values: [regression-suite], classification: content }
    - { name: status, type: enum, values: [active, under-review, deprecated], classification: org-state }
    - { name: scope, type: string, classification: content }
  optional:
    - { name: owner, type: string, classification: org-state }
    - { name: cases, type: "list[ref(test-case)]", classification: content }
    - { name: evidence_as_of, type: date, classification: org-state }
sections:
  required: [Purpose, Protected Behavior, Membership Policy, Cases]
  optional: [Execution Policy, Environments, Evidence, Curation History]
default_agents:
  authoring: regression-curator
  review: qa-reviewer
---
# Regression suite spec type

A regression suite is long-lived protection for shipped behavior. Membership is
intentional: each case explains the behavior it protects, its stability, and why
its value exceeds execution and maintenance cost. Deprecation preserves curation
history and identifies replacement protection or accepted risk.

