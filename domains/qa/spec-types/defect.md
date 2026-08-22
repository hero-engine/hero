---
title: QA Defect
type: defect
domain: qa
category: work
bucket: defects
location: .hero/planning/defects/{slug}/spec.md
lifecycle:
  states: [reported, triaging, accepted, handed-off, resolved, closed]
  initial: reported
  terminal: [resolved, closed]
frontmatter:
  required:
    - { name: title, type: string, classification: content }
    - { name: type, type: enum, values: [defect], classification: content }
    - { name: status, type: enum, values: [reported, triaging, accepted, handed-off, resolved, closed], classification: org-state }
    - { name: severity, type: enum, values: [critical, high, medium, low], classification: content }
  optional:
    - { name: case, type: "ref(test-case)", classification: content }
    - { name: feature, type: "ref(feature)", classification: content }
    - { name: environment, type: string, classification: content }
    - { name: owner, type: string, classification: org-state }
sections:
  required: [Observed Behavior, Expected Behavior, Reproduction, Evidence, Triage]
  optional: [Environment, Impact, Workaround, Related Items, Resolution]
default_agents:
  authoring: test-issue-triager
  review: qa-reviewer
---
# QA defect spec type

A defect is an optional QA-owned record used before engineering handoff. Teams that
do not need this funnel create a Core bug directly through diagnosis. Handoff must
preserve the case, requirement, reproduction, evidence, and QA defect relationship
rather than copying an untraceable summary.

