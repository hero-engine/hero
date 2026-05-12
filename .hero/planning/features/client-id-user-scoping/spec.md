---
title: Client ID User Scoping — Fix Conflict False-Positives
type: feature
status: planning
priority: P0
tags: [graph-memory, federation, conflicts, identity]
created: 2026-04-27
relations:
  - target: graph-conflict-detection
    kind: sibling
  - target: pre-launch-hardening
    kind: child
horizon: next
smoke: deferred
---

## Problem

Phase 7c verification surfaced 114 "conflicts" on a normal Bob push,
when only **one** spec (`conflict-test`) had actually diverged. Every
other shared spec showed up as a conflict because Alice and Bob are
the same human, the same repo, the same content — but two workspaces
on disk with two different machine-generated `client_id`s.

```
Feature graph-memory — overwrote version from client d54856e743f01d18
Feature cloud-dashboard — overwrote version from client d54856e743f01d18
... 112 more ...
Feature conflict-test — overwrote version from client d54856e743f01d18
```

The signal we want is buried in noise. A real teammate-vs-teammate
conflict is invisible when 113 false positives accompany it.

## Root cause

`client_id` is currently generated **per workspace** — a random hash
written to `.hero/graph.db` the first time the workspace is initialized.
That made sense when "client" meant "this graph.db file", but the user
mental model is different: a developer is one identity regardless of
how many checkouts of a repo they have.

Two checkouts of the same repo by the same person look like two
"clients" colliding, when they're really one person reconciling their
own work.

## Resolution

Bind `client_id` to the **authenticated user**, not the workspace. The
JWT subject (`claims.UserID`, the org-member UUID) is the natural
identity — it's already on the server side of every push.

### Server-side change

In `cloud/api/graph.go:handlePush`, replace `req.ClientID` (which the
client sends) with `claims.UserID` from the JWT context. The server
ignores whatever the client sends — the authenticated identity wins.

```go
claims := middleware.GetClaims(r.Context())
clientID := claims.UserID  // was: req.ClientID
// ... use clientID throughout, not req.ClientID
```

This means two workspaces by the same user push as the same `client_id`
and never conflict with themselves. Two workspaces by different users
of the same org push as different `client_id`s and conflict correctly.

### Client-side change (optional cleanup)

The client can stop generating and storing a workspace-scoped client_id
since the server ignores it. For backward compat, leave the field on
the wire but stop reading from the local `meta` table. Eventually
remove the column from the SQLite schema.

## What this fixes

- Bob pushing his copy of `graph-memory` doesn't conflict with Alice's
  identical copy — same hash, same user → idempotent
- The 114-conflict noise on the verification test drops to 1
  (the deliberately-divergent `conflict-test`)
- Real conflicts (Alice and Bob, two different humans, edited the same
  spec) still surface correctly

## Files

| File | Change |
|---|---|
| `cloud/api/graph.go` | Use `claims.UserID` instead of `req.ClientID` |
| `internal/graph/sync.go` | Stop hydrating `client_id` from local meta (optional) |
| `internal/graph/store.go` | Mark client_id meta column as deprecated (optional) |

## Success criteria

- Two-workspace same-user test (current verification setup): zero
  conflicts on push
- Two-user test (alice@x.com and bob@x.com both touch `conflict-test`):
  exactly one conflict reported, attributed to the correct prior user
- `hero check conflicts` shows `overwrote version from <user-uuid>`
  rather than an opaque hash, so the user can match it to a real teammate

## Out of scope

- Pretty-printing user UUIDs as emails / display names in the conflict
  message (cosmetic; needs a join against `users` on the server). File
  separately as `conflict-attribution-display`.
- Per-machine identity for non-graph telemetry purposes (telemetry has
  its own install-id, unrelated to graph sync)
