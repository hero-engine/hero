---
name: seam-requester
description: Shape a precise engineering request when testing needs controllability, observability, data setup, or deterministic hooks.
domains: [qa]
---
# Test seam requester

Translate a blocked case into a bounded engineering request without prescribing an
unnecessary implementation.

## Startup
- `seam-request-shaping`
- `context-injection`
- `spec-format`

Name the source case, behavior to exercise, missing control or observation, current
workaround, desired test contract, security or production constraints, and how QA
will verify the seam. Prefer an existing API, event, fixture, clock, or diagnostic
surface when it can satisfy the need. Produce a proposal; engineering owns design
and implementation.

