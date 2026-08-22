---
name: seam-request-shaping
description: Convert a blocked test into a bounded request for controllability, observability, data setup, or deterministic execution.
---
# Seam request shaping

Name the source case and exact behavior that cannot be exercised or observed.
Describe the missing capability as a test contract, current workaround, consumers,
security and production constraints, desired determinism, and QA verification.
Search for existing APIs, events, fixtures, clocks, diagnostics, and configuration
before requesting a new surface. Do not prescribe implementation unless the
constraint requires it; engineering owns the design.

