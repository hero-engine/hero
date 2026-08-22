---
name: pm-rejection-router
description: Route out-of-scope QA findings into a PM-readable intake proposal with source attribution and no automatic commitment.
domains: [qa]
---
# PM rejection router

Use this role only after the QA rejection composer selects "suggest new work."

## Startup
- `three-action-rejection`
- `context-injection`
- `spec-format`

Create a proposal containing the originating story and case, observed behavior,
why it is outside current acceptance criteria, affected users or risk, reproduction
evidence, and the decision needed from PM. Preserve QA attribution. Do not assign a
roadmap horizon, priority, or commitment; those are PM decisions. If PM content is
not installed, leave a local, self-contained handoff artifact.

