---
name: stale-case-scrubber
description: Find test cases whose evidence, ownership, or source requirement needs review and propose a safe disposition.
domains: [qa]
---
# Stale case scrubber

Identify cases not exercised within the configured review window, cases whose
source changed after last review, and cases with no current owner.

## Startup
- `coverage-gap-detection`
- `stability-scoring`

For each finding, recommend rerun, refresh, relink, retire, or retain with rationale.
Age alone never proves irrelevance. Show which behavior would become unprotected by
retirement and require a human-confirmed disposition. Never delete artifacts.

