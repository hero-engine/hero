---
title: Release Gate
type: release-gate
domain: qa
category: work
bucket: release-gates
location: .hero/planning/release-gates/{slug}/spec.md
lifecycle:
  states: [open, reviewing, go, conditional-go, no-go, closed]
  initial: open
  terminal: [go, conditional-go, no-go, closed]
frontmatter:
  required:
    - { name: title, type: string, classification: content }
    - { name: type, type: enum, values: [release-gate], classification: content }
    - { name: status, type: enum, values: [open, reviewing, go, conditional-go, no-go, closed], classification: org-state }
    - { name: release, type: "ref(release)", classification: content }
  optional:
    - { name: policy, type: string, classification: content }
    - { name: decision_owner, type: string, classification: org-state }
    - { name: evidence_as_of, type: date, classification: org-state }
sections:
  required: [Candidate, Policy, Scope, Coverage, Regression Evidence, Open Blockers, Recommendation]
  optional: [Environment Limits, Flake Signal, Waivers, Rollback Readiness, Decision]
default_agents:
  authoring: release-readiness-strategist
  review: release-gate-reviewer
---
# Release gate spec type

A release gate records the evidence and policy behind a release recommendation.
Go, Conditional Go, and No-Go are explainable states, not automatic deployment
actions. Every waiver names an owner, rationale, affected risk, and follow-up; the
human release owner records the final decision.

