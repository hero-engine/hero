---
title: Roadmap Item
type: roadmap-item
domain: pm
category: work
bucket: roadmap
location: .hero/planning/roadmap/{slug}/spec.md
lifecycle:
  states: [candidate, committed, shipped, dropped]
  initial: candidate
  terminal: [shipped, dropped]
frontmatter:
  required:
    - { name: title, type: string, classification: content }
    - { name: type, type: enum, values: [roadmap-item], classification: content }
    - { name: status, type: enum, values: [candidate, committed, shipped, dropped], classification: org-state }
    - { name: horizon, type: enum, values: [now, next, later], classification: org-state }
  optional:
    - { name: initiative, type: "ref(initiative)", classification: content }
    - { name: evidence, type: "list[ref(intake)]", classification: content }
    - { name: features, type: "list[ref(feature)]", classification: content }
    - { name: appetite, type: enum, values: [1w, 2w, 6w], classification: content }
    - { name: target_release, type: string, classification: org-state }
sections:
  required: [Outcome, Evidence, Why Now]
  optional: [Appetite, Non-goals, Measures, Linked delivery]
default_agents:
  authoring: roadmap-curator
  review: roadmap-reviewer
---

# Roadmap item spec type

A roadmap item is a PM-owned statement of intended outcome and timing. It is
not a disguised feature list. Evidence explains why the outcome matters;
linked initiatives, epics, and features explain how the organization chose to
pursue it.

`candidate` means the opportunity is being weighed. `committed` means the
organization has intentionally reserved attention or capacity. `shipped`
requires delivery evidence from linked work, and `dropped` requires a recorded
reason so the decision remains inspectable.

Horizon is the default planning axis. A cycle preset may render the item as a
Pitch and add appetite; a phased preset may add a target release. The artifact
remains a canonical `roadmap-item` in every preset.
