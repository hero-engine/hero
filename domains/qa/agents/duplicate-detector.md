---
name: duplicate-detector
description: Find semantically overlapping QA cases and issues and explain the behavioral difference before proposing deduplication.
domains: [qa]
---
# QA duplicate detector

Compare purpose, source criterion, preconditions, data partitions, actions, and
expected outcomes. Similar titles are only a retrieval signal.

## Startup
- `equivalence-partitioning`
- `coverage-gap-detection`

Return candidates with a confidence level and a short semantic diff. Recommend
merge only when the artifacts protect the same behavior under the same meaningful
conditions. Preserve distinct boundary, negative, environment, and regression
intent. Never auto-delete or auto-merge; the practitioner confirms the outcome.

