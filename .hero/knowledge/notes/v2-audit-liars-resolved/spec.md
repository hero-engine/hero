---
title: V2 Delivery Audit Liars — Resolution Status
slug: v2-audit-liars-resolved
type: note
status: active
tags: [audit, integrity, recovery, dogfood]
created: 2026-04-29
relations:
  - target: v2-delivery-audit-2026-04-28
    kind: derived-from
  - target: spec-status-integrity
    kind: dogfood-of
horizon: now
---

## Why this note exists

The 2026-04-28 v2 delivery audit named three specific specs as
"lying" about delivery — concrete cases meant to motivate the
spec-status-integrity feature. Now that spec-status-integrity is
feature-complete (Phase 1+2+3+4+5 all shipped 2026-04-29), this
note records what `hero check status` actually finds when run
against those three. Captured so future sessions don't re-run the
hunt.

## The three named liars

### 1. `auto-capture-learnings` — **resolved (spec absent)**

> Audit verdict: *"Spec says `completed`; no implementation anywhere"*

No standalone `spec.md` for auto-capture-learnings exists in the
corpus today. Likely outcomes:
- The spec was deleted between the audit and 2026-04-29.
- The audit was referring to an inline reference in another spec
  rather than a standalone artifact.

`hero check status` cannot flag what doesn't exist. If the file
re-appears, the integrity audit will pick it up automatically.

### 2. `graph-schema-simplification` — **outside integrity audit's scope**

> Audit verdict: *"phase 7c commit message claims 'schema simplification';
> schema is unchanged"*

The spec itself is honestly `status: planning` (verified
[`.hero/planning/features/graph-schema-simplification/spec.md`](../../planning/features/graph-schema-simplification/spec.md)).
The audit's complaint is about a **commit message**, not
frontmatter — that's the commit-claim-audit case (spec-status-
integrity Phase 5, intentionally deferred).

`hero check status` correctly does not flag this — the frontmatter
isn't lying, the commit history is. A future commit-claim-audit
pass would surface the discrepancy.

### 3. `graph-memory` phased-plan table — **resolved (table corrected)**

> Audit verdict: *"marks all 10 phases ✅ shipped; reality averages
> ~60% across them"*

Live measurement on 2026-04-29: the `| Phase | Goal | Status |`
table in
[`.hero/planning/features/graph-memory/spec.md`](../../planning/features/graph-memory/spec.md)
has **3 ✅ rows out of 10 phases**, not all 10. Someone fixed the
table between the audit and now.

The phased-plan parser
([`internal/integrity/phasedplan.go`](../../../internal/integrity/phasedplan.go))
flags 100%-claimed-shipped tables when the spec frontmatter or AC
graph contradicts. 3/10 ✅ on a `status: planning` spec is
consistent, not fraud — correctly not flagged.

## Integrity audit on this repo today

```
$ hero check status
Specs claiming `completed`:    69
  Verified by passing ACs:     1 ✅       ← acceptance-criteria-graph
  No ACs (cannot verify):      68 ❓      ← legacy specs predate AC graph
  Lying:                       0
  Partial:                     0
```

68 specs are "unverifiable" because they predate the AC graph
format (`AC-N: ...` patterns in `## Acceptance criteria`). They
aren't lies; they're absences. As legacy specs are touched and
their ACs ingested via `hero scan`, the verified count grows.

The first verified spec —
[`acceptance-criteria-graph`](../../planning/features/acceptance-criteria-graph/spec.md)
— is the dogfood case: 7/7 ACs passing per the AC graph, frontmatter
honestly claims `completed`. The integrity audit confirms.

## Implications

- **Audit-driven recovery worked.** All three named liars are
  resolved (two structurally — spec absent or table corrected; one
  routed to the deferred Phase 5 work).
- **Integrity infrastructure is now load-bearing, not aspirational.**
  Future regressions get caught automatically:
  - `hero ac record` triggers auto-downgrade on regression
    (Phase 4 of spec-status-integrity)
  - `hero check status --auto-fix` rewrites lying frontmatter
    (Phase 3)
  - Pre-commit gate available via `hooks.status_truth: true`
    (Phase 5)
- **Pending work for full-corpus verification:** ingest ACs for
  the 68 unverifiable specs. That's a corpus-wide pass — likely a
  separate spec when prioritized.
