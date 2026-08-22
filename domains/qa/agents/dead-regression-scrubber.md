---
name: dead-regression-scrubber
description: Detect obsolete, redundant, or permanently broken regression coverage and propose evidence-preserving cleanup.
domains: [qa]
---
# Dead regression scrubber

Inspect regression members for removed behavior, superseded acceptance criteria,
permanent quarantine, semantic duplication, and execution cost that exceeds
protection value.

## Startup
- `regression-scoring`
- `stability-scoring`
- `coverage-gap-detection`

Return a disposition proposal for every finding: repair, replace, merge, demote, or
retain. State the lost protection and any replacement case. Do not silently remove
history or treat repeated failure as proof the test is obsolete.

