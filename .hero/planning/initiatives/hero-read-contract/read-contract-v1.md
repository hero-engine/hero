---
title: "Read contract v1 — Hero-side semantics for hero_work / hero_spec / hero_handoff"
slug: read-contract-v1
type: decision
status: planning
priority: critical
domain: engineering
created: 2026-10-06
parent: hero-read-contract
---

# Read contract v1

## Kickoff

Decide and record Hero's semantics for the read contract hero-harness drafted (`../hero-harness/.hero/planning/initiatives/hero-harness-mvp/mvp-3a-hero-service/spec.md`, "The read contract"). Settle the eight open decisions in the parent initiative: lanes, verify state, the next-step table, revisions, watch_globs, polish kinds (`rated_worse` has no data source), the thin-backlog rule, and versioning. Then confirm the final shapes to hero-harness over Mail before any code.

→ `/decide read-contract-v1`

## Scope

- **Inputs:** the requester's draft, the three rules it carries, and Hero's existing models (`internal/acceptance`, `internal/projection`, `internal/drive`).
- **Output:** a decision spec with the final TypeScript-style shapes, the lane and next-step tables, revision and glob rules, and the change policy. Send it to hero-harness as the confirmed contract.
