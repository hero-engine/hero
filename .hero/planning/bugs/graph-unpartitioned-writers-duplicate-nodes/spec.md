---
title: "Ten graph writers omit Repo, so repo-scoped identity lets '' and stamped nodes coexist forever"
slug: graph-unpartitioned-writers-duplicate-nodes
type: bug
status: planning
domain: engineering
priority: medium
severity: medium
root_cause_class: code
tags: [graph, substrate, repoKey, partition, duplication, ingest]
created: 2026-07-25
depends-on: [graph-node-identity-repo-scoped]
---

# Ten graph writers omit Repo, so repo-scoped identity lets `''` and stamped nodes coexist forever

## Kickoff

Paste into a fresh session to start delivery:

> Deliver `graph-unpartitioned-writers-duplicate-nodes`. Ten production
> `UpsertNode` call sites build their node without a `Repo` field, so they
> write into the unpartitioned (`repo = ''`) bucket. Since
> `graph-node-identity-repo-scoped` made identity `(type, key, repo)` and gave
> the write path `repoWriteScope` (an unpartitioned write matches only
> `repo = ''`, so it can never tombstone a stamped node), an unpartitioned
> writer and a stamped writer targeting the same key now leave two live rows
> that neither will ever retire. Stamp `Repo: repoKey` at each of the ten
> sites listed in **Key Files**, then decide what to do with the rows already
> in the `''` bucket. Start by reading **Key Files**, then work the Acceptance
> Criteria in order. Close with the cold delivery audit and `hero spec verify`.

## Summary

`graph-node-identity-repo-scoped` deliberately traded a correctness bug for a
duplication one: an unpartitioned write used to *tombstone* a repo-stamped
node (the team-oauth failure); now it stays in its own partition and both rows
remain live. That is the right trade — a duplicate is recoverable, a tombstone
is not — but it leaves a residue that should be cleaned up at the source.

## Issue

Confirmed by cold audit of `graph-node-identity-repo-scoped`: after an
unpartitioned write, a stamped write, then another unpartitioned write for the
same `(type, key)`, neither writer retires the other and `ListNodesByType`
returns both.

**Reachability today is low but real.** The seven unscoped `ListNodesByType`
callers read only `Task` and `Criterion`, neither of which is among the ten
unpartitioned writers, and this repo's live corpus has **0** such collisions
(16 nodes sit in the `''` bucket, none colliding with a stamped key).
`Person` is the genuine collision surface: it is written unpartitioned by
`tracker/graph_ingest.go` and `gitutil/graph_ingest.go`, but stamped by
`tasks/record.go`'s `upsertPerson`.

## Key Files

The ten unpartitioned `UpsertNode` sites:

| File:line | Node type | Key |
|---|---|---|
| `internal/tracker/graph_ingest.go:54` | Sprint | `info.ID` |
| `internal/tracker/graph_ingest.go:104` | Issue | `item.ID` |
| `internal/tracker/graph_ingest.go:131` | Person | assignee |
| `internal/tracker/graph_ingest.go:166` | Issue | `item.EpicID` |
| `internal/tracker/graph_ingest.go:267` | Issue | `item.ID` |
| `internal/tracker/graph_ingest.go:281` | Person | assignee |
| `internal/gitutil/graph_ingest.go:109` | Person | author email |
| `internal/gitutil/graph_ingest.go:188` | Issue | `ref.key` |
| `internal/extract/decisions.go:166` | Concept | term |
| `internal/acceptance/record.go:227` | Commit | sha |

Related: `internal/graph/node.go` (`repoWriteScope`, `repoPredicate`),
`internal/tasks/record.go:175` (`upsertPerson` — the stamped Person writer
these collide with).

## Goal

Every graph writer stamps a partition, so the `''` bucket holds only genuinely
global rows (or nothing), and no key can have both an unpartitioned and a
stamped live row.

## Suggested Fix Approach

1. **Stamp `Repo: repoKey` at each of the ten sites.** Most already have a
   repoKey in scope or are called from a function that does; thread it where
   not.
2. **Decide the rule for global node types.** `Person` and `Repo` are in
   `globalNodeTypes` and carry an empty `Domain` by design. Decide explicitly
   whether they should also be repo-global (`repo = ''` everywhere, including
   in `tasks/record.go`'s `upsertPerson`, which currently stamps) or
   per-repo — and make every writer agree. Today they disagree, which is the
   only real collision surface.
3. **Reconcile the existing `''` rows.** A one-shot migration or a documented
   `hero graph reingest` note. `graph.db` is regenerable, so a rebuild is a
   legitimate answer.
4. **Add an invariant test** asserting no `(type, key)` has both a live
   `repo = ''` row and a live stamped row.

## Acceptance Criteria

- **AC-1:** THE SYSTEM SHALL stamp a repo partition at every production
  `UpsertNode` call site, verified by a source-level guard test so a new
  unpartitioned writer cannot be added silently.
- **AC-2:** THE SYSTEM SHALL apply one consistent partitioning rule to global
  node types (`Person`, `Repo`) across every writer.
- **AC-3:** THE SYSTEM SHALL NOT hold both a live `repo = ''` row and a live
  stamped row for the same `(type, key)`. Asserted on existence rather than on
  what a query returns — the stronger and more directly testable invariant.
- **AC-4:** THE SYSTEM SHALL provide a path to reconcile pre-existing rows in
  the `''` bucket without manual SQL.

## Boundaries

- **In scope:** the ten writer sites, the global-node-type rule, the
  reconcile path, and the guard test.
- **Out of scope:** node identity itself (delivered in
  `graph-node-identity-repo-scoped`); changing `repoWriteScope`'s rule — the
  duplication is the intended safe side of that trade, not a defect in it.

## Risks

- **Changing Person keying could split existing person nodes** across
  partitions, affecting `hero resume` attribution and assignee joins.
  Mitigation: decide AC-2 before touching the writers, and reconcile in the
  same change.
