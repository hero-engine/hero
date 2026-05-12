---
title: Pre-Launch Hardening — Federation Polish, Security, Observability
type: initiative
status: delivered
priority: P0
tags: [sprint, pre-launch, federation, security]
created: 2026-04-27
relations:
  - target: hero-cloud
    kind: child
  - target: graph-memory-federation
    kind: related
  - target: launch-readiness
    kind: sibling
horizon: someday
---

## Goal

Ship the federation knowledge graph to the first real team without
leaving any of the obvious foot-guns we've already identified. After
this sprint, Hero Cloud is one onboarding away from being usable by
people who don't work here.

## Sprint contents

| # | Spec | Status |
|---|---|---|
| 1 | [`client-id-user-scoping`](../../features/client-id-user-scoping/spec.md) | ✅ delivered — 114 false-positive conflicts → 0 |
| 2 | [`tenant-isolation-rls`](../../features/tenant-isolation-rls/spec.md) | ✅ delivered — RLS infra + every tenant handler bound |
| 3 | [`unified-search`](../../features/unified-search/spec.md) | ✅ delivered — sibling-repo scan, repo-tagged results |
| ~~4~~ | ~~`hero-telemetry`~~ | Deferred to [`launch-readiness`](../launch-readiness/spec.md) |
| ~~5~~ | ~~Production deploy~~ | Deferred to [`launch-readiness`](../launch-readiness/spec.md) |

Telemetry and production deploy were moved to the
[`launch-readiness`](../launch-readiness/spec.md) initiative on
2026-04-27 — the call was to keep polishing the core product before
deploying / instrumenting it. Pick those back up when the polish work
is in a place we'd be proud to demo without caveats.

## Order rationale

- **(1) before (2)**: `client-id-user-scoping` was a one-session fix
  and unblocked honest conflict-detection demos.
- **(2) before public exposure**: RLS lands before *any* external
  team's data is allowed in the cluster. Hard gate.
- **(3) was parallel-safe** with the others.

## Out of sprint

Explicitly **not** in this sprint, even though they're tempting:

- **Marketing track** (`hero-landing-page`, `hero-docs-site`,
  `hero-launch-playbook`, `hero-positioning`, etc.) — separate
  initiative, separate skillset, runs in parallel with this work.
  Belongs to [`hero-marketing`](../hero-marketing/spec.md).
- **Hero v2 / agents / killer features** — feature work, not
  hardening. Belongs to [`hero-killer-features`](../hero-killer-features/spec.md).
- **Conflict attribution display** (showing email instead of UUID)
  — cosmetic; ship after launch when there are real teams to test it.
- **Field-level merge** in conflicts — graph-conflict-detection v2,
  out of scope per its own spec.
- **Graph database migration** — current schema works; revisit only
  when we need multi-hop traversal we can't do in SQL.

## Definition of done — met

- ✅ All three in-scope features delivered
- ✅ `hero sync graph push` from a fresh workspace returns 0 conflicts
  for same-user, 1+ for genuine cross-user divergence (verified)
- ✅ Direct SQL on the cluster cannot read another tenant's rows
  with no `WHERE` filter (verified at SQL level)
- ✅ `hero search foo` returns federation pulls + on-disk sibling
  specs in one merged result list, with the source repo tagged

## Notes from delivery

- **RLS performance**: CockroachDB v26.1 doesn't support subqueries
  in policy expressions, so we denormalized `org_id` onto
  `specs`/`knowledge`/`conventions`/`pr_checks` rather than reaching
  through a foreign key. Side benefit: faster reads.
- **Schema simplification**: Mid-sprint we replaced the bitemporal
  `valid_to` graph schema with a simple upsert + history-table model.
  Push went from 17,000 SQL statements to 3 per batch and no longer
  hits CockroachDB serialization errors.
- **Same-user conflict detection**: Binding `client_id` to the JWT
  subject (not a per-workspace random) eliminated the noise. One
  human across two checkouts no longer "conflicts with themselves."
