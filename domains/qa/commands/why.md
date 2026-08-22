---
core_fork: QA graph tracing includes cases, plans, suites, defects, and release evidence rather than Core's generic lineage view
description: Trace why a QA artifact exists through its requirement, plan, suite, failure, and release relationships.
---
# QA why router

Walk declared graph relationships instead of inferring genealogy from body links.
For a case, show its criterion and plan; for a regression member, show the shipped
behavior it protects; for a defect, show the failure and case that raised it; for a
release gate, show blockers and contributing evidence. Label missing edges and
historical or foreign-pack artifacts rather than inventing a chain.

Request: $ARGUMENTS
