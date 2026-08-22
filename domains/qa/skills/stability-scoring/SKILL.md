---
name: stability-scoring
description: Compute an explainable confidence signal from recent executions without hiding sparse or biased evidence.
---
# Stability scoring

Choose a declared window and report total runs, passes, deterministic failures,
intermittent failures, skipped runs, environments, and recency. A ratio without
sample size is misleading, so show both. Separate product failures from invalid
test or environment runs. Label insufficient data and environment bias. Use the
score to prioritize investigation or regression promotion, never to erase the raw
history.

