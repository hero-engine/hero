---
description: Audit QA artifacts for stale cases, dead regression entries, orphaned links, and coverage drift.
---
# Scrub QA corpus

Route stale-case findings to `stale-case-scrubber`, regression findings to
`dead-regression-scrubber`, and relationship or coverage findings to
`coverage-curator`. Return a categorized report with evidence and recommended
repair, relink, rerun, merge, or retirement. Do not delete, retire, or rewrite
artifacts automatically; every destructive disposition needs human confirmation.

Request: $ARGUMENTS

