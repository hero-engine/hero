---
title: "Tracker Fixtures — GitLab Parity and Offline Mock Server for Round-Trip Validation"
slug: tracker-fixtures
type: initiative
status: completed
priority: high
horizon: next
tags: [tracker, gitlab, mock-server, fixtures, round-trip, validation]
created: 2026-06-27
received_from:
  peer_id: cd8dd06d-3df1-4878-a88f-24593dcbb4b3
  peer_alias_display: hero-code
  originator_slug: pm-workbench-tracker-validation
  call_id: 18bd0775ec35d74042d241cd94a2fd37
  mode: spec-out
  at: 2026-06-27T19:40:30Z
  at_commit: f1d4c1b
  reason: "hero-code's pm-workbench-tracker-validation harness has no real or fake tracker to exercise. The Go tracker engine (GitLab client + mock-tracker-server) belongs in hero, not the Swift app."
relations:
  - target: gitlab-tracker-support
    kind: child
  - target: mock-tracker-server
    kind: child
completed_at: 2026-07-02T02:37:27Z
---

# Tracker Fixtures — GitLab Parity and Offline Mock Server for Round-Trip Validation

## Provenance

Designed in response to `hero peer call --mode=spec-out` from peer
`hero-code` (peer_id `cd8dd06d-3df1-4878-a88f-24593dcbb4b3`),
related to originator spec `pm-workbench-tracker-validation`.
The originator harness asserts `hero sync import`, `hero sync push --field`,
`hero sync pull --field`, and `hero spec set-owner` round-trip cleanly
against every supported tracker — but it has never been run against a
real or fake tracker. This initiative produces the two missing pieces
that live on the hero side: a fourth real tracker (GitLab) and an
offline server that speaks the subset of every tracker API we touch.

## Goal

Make Hero's tracker round-trip (import → push → pull → set-owner)
exercisable two ways:

1. **Live**: against a real GitLab project, joining the existing
   github/jira/linear round-trip parity.
2. **Offline**: against a local `mock-tracker-server` binary that speaks
   the GitHub, Jira, Linear, and GitLab API subset our adapters
   actually call. Same harness, same assertions, no network, no
   rate-limit risk, deterministic.

Together the two children unblock the originator-side live-validation
harness: hero-code can point at a real GitLab for one CI lane and at
`mock-tracker-server` for the other.

## Why This Exists

Today the field-level push/pull paths are believed to round-trip but
have never been exercised end-to-end against a tracker we control.
Mocks at the unit-test layer test our HTTP request shape; they do not
test that pagination terminates, that 429 backoff matches reality, or
that a server-side field mutation actually surfaces as drift on the
next pull. The Swift originator harness was built to be the missing
end-to-end test, but the Go side never gave it anything to talk to.

Adding GitLab also closes a long-standing parity gap (we ship
github/jira/linear; GitLab is the next-most-requested) while
deliberately exercising the abstraction — any place where the
`Tracker` interface leaks tracker-specific assumptions will surface
during the GitLab port.

## Children

| Slug | Title | Phase | Status |
|---|---|---|---|
| gitlab-tracker-support | GitLab Tracker Support — Issues/Epics/Milestones/Iterations Round-Trip | 1 | Planning |
| mock-tracker-server | Mock Tracker Server — Offline Multi-Tracker HTTP Fake with Drift Injection | 1 | Planning |

## Sequencing

```
Phase 1 (parallel — independent):
  ┌── gitlab-tracker-support ───┐
  │                              │
  └── mock-tracker-server ──────┘
              │
Phase 2:      ▼
   originator harness runs offline (mock server) + live (real GitLab)
   in hero-code's pm-workbench-tracker-validation CI lane
```

The two children are independent and can be built in parallel. They
join at the mock server's `gitlab` mode, which mock-tracker-server
must implement *because* GitLab is now a supported tracker — that's
the only ordering constraint, and it falls naturally out of building
gitlab.go first or shipping mock-tracker-server's GitLab handler with
a one-line guarded `TODO: depends on gitlab-tracker-support`.

## Boundary With the Originator

The Swift app (hero-code) owns:
- `GitLabImportShape.swift` and shapes for the other trackers
- The Acme Checkout dataset and per-tracker seeders that materialize
  it into each tracker's wire shape
- The live-validation harness that drives `hero sync …` and asserts
  round-trip equality
- CI orchestration: pick "live GitLab" vs "mock-tracker-server" mode

The hero side (this initiative) owns:
- The Go GitLab adapter (real GitLab REST client)
- The mock-tracker-server binary and its in-memory state model
- A small default seed bundled into the binary so hero's own
  integration tests can run the server without depending on hero-code

The contract between the two sides is the **mock server's seed
format** (a tracker-neutral JSON document) plus the **published API
surface** the mock server claims to support. Both are versioned and
documented in the mock-tracker-server feature spec.

## Acceptance Criteria

1. **AC-1: GitLab round-trip parity** — All `gitlab-tracker-support`
   acceptance criteria pass. `hero sync import`, `hero sync push
   --field`, `hero sync pull --field`, `hero spec set-owner` work
   against a real GitLab project with the same semantics as
   github/jira/linear.
2. **AC-2: Offline harness** — `mock-tracker-server` runs as a single
   binary, accepts a seed file, serves the four tracker API surfaces,
   honors pagination to exhaustion, emits 429 with `Retry-After` on a
   configurable trigger, and supports admin endpoints to mutate
   server-side fields (induce drift) and rotate API IDs.
3. **AC-3: End-to-end** — A scripted scenario seeds the mock with the
   Acme Checkout fixture (provided by hero-code), runs `hero sync
   import` against each of the four tracker modes, runs `hero sync
   push --field` on a modified spec, mutates the server-side field
   out-of-band, runs `hero sync pull --field`, and observes the
   expected drift. This scenario is the smoke test for the initiative.
4. **AC-4: No leakage of test-only types** — `mock-tracker-server`
   lives in its own `cmd/mock-tracker-server/` and
   `internal/mocktracker/` packages; nothing in `internal/tracker/`,
   `internal/cli/`, or any production code path imports it. CI builds
   the binary, but `go build ./...` for hero doesn't pull it into the
   main binary.

## Risks and Mitigations

| Risk | Mitigation |
|---|---|
| GitLab Epics and Iterations are GitLab Premium features — open-source GitLab instances may not expose them | Adapter gracefully degrades: epics and iterations are read with `?include_subscribed=false` and skipped on 403/404 with a one-line "tracker tier does not expose epics" notice. Tests cover both paths. |
| Mock server diverges from real APIs over time | Mock server contract documented in its spec; live-validation lane catches divergence by definition (same harness runs against both). |
| Acme dataset shape evolves on the hero-code side | Seed format versioned (`schema_version: 1`). Mock server rejects unknown versions with a clear error rather than silently mis-interpreting. |
| GitLab self-hosted base URL handling | GitLab adapter requires `base_url` in `TrackerConfig` (like Jira) and rejects empty values at construction. |

## Out of Scope

- GitLab Issue Boards as a separate concept (boards expose issues via
  the issues endpoint already covered).
- GitLab Merge Request integration (this initiative covers Issues,
  Epics, Milestones, Iterations only — MR support is a separate
  request if it ever comes).
- A persistent-storage mode for the mock server (in-memory only;
  resets on restart — that's the point).
- Mocking provider OAuth flows. The mock server accepts any
  non-empty `Authorization` header.
- Reproducing every GitHub Projects v2 GraphQL field. The existing
  github adapter skips points/priority with a warning; the mock
  preserves that behavior verbatim rather than implementing v2.

## Handoff Trail

- 2026-07-02T02:40:21Z — out → hero-code (peer_id: cd8dd06d-3df1-4878-a88f-24593dcbb4b3)
  mode: advisory
  originating_spec: tracker-fixtures
  at_commit: 98731d6
  result_ref: .hero/peer-calls/18be5891189bad88b4c345e6262a1c8a.md
  reason: "Closing the loop: the tracker engine hero-code spec-out'd (pm-workbench-tracker-validation, call 18bd0775) is delivered and released in hero v0.23.3."

