---
title: "Cloud CLI Verify — Cross-Repo CLI ↔ Server Integration (hero slice)"
type: feature
status: completed
completed_at: 2026-06-24T20:55:00Z
received_from:
  peer_id: 5770cae7-b233-45c0-8e5d-765338a6058c
  peer_alias_display: hero
  originator_slug: cloud-cli-verify
  handed_off_at: 2026-06-24T18:09:11Z
  at_commit: ea7ed71
---

# Cloud CLI Verify — Cross-Repo CLI ↔ Server Integration (hero slice)

## Provenance

Handed off from peer `hero-cloud` (peer_id `5770cae7-b233-45c0-8e5d-765338a6058c`)
as an async-drop. hero-cloud completed the full `cloud-cli-verify` spec on their
side (integration tests, wire-format docs, governance filtering). This repo owns
**one slice**: the graph-sync URL double-nesting fix called out in their AC #7 and
Changes table. Originating spec: `hero-cloud/.hero/specs/cloud-cli-verify/spec.md`.

## Goal

Fix the graph-sync URL construction in the hero CLI so it matches the hero-cloud
server's org-scoped route contract, and add a regression test that locks the
correct URL shape.

## Diagnosis

- **Server contract** (hero-cloud): graph endpoints are org-scoped at
  `…/api/v1/orgs/<org>/graph/push` and `…/graph/pull`.
- **Call sites** correctly set `SyncClient.ServerURL = {cloudURL}/api/v1/orgs/<org>`
  (`internal/cli/sync_graph.go:120`, `internal/cli/scan_team_sync.go:60`).
- **Bug:** `Store.Push` / `Store.Pull` appended `/api/v1/graph/push` and
  `/api/v1/graph/pull` onto that already-prefixed ServerURL
  (`internal/graph/sync.go`), producing a doubled prefix:
  `…/api/v1/orgs/<org>/api/v1/graph/push`.
- **Why it shipped:** the existing tests passed a *bare* `ts.URL` as ServerURL, so
  `bare + /api/v1/graph/push` resolved correctly and never exercised the prefixed
  form the real CLI uses.

## Fix

- `internal/graph/sync.go` — Push/Pull now append only `/graph/push` and
  `/graph/pull`; the `/api/v1/orgs/<org>` prefix lives in `ServerURL`. Doc comment
  updated to describe ServerURL as the org-scoped base.
- `internal/graph/sync_test.go` — tests now build ServerURL via an `orgURL` helper
  that mirrors the real call sites; the fake server's routes moved to the
  org-scoped paths. Added `TestSyncEndpoints_DoNotDoubleAPIPrefix` asserting the
  exact push/pull paths and a single `/api/v1/` segment.

## Acceptance Criteria

- WHEN `hero sync graph push` runs THE CLI SHALL POST to
  `…/api/v1/orgs/<org>/graph/push` with no doubled `/api/v1/` segment — DONE
- WHEN `hero sync graph pull` runs THE CLI SHALL GET
  `…/api/v1/orgs/<org>/graph/pull` with no doubled `/api/v1/` segment — DONE
- A regression test fails if the doubled-prefix bug is reintroduced — DONE
  (verified: reintroducing the bug fails `TestSyncEndpoints_DoNotDoubleAPIPrefix`)

## Boundaries

- Only this repo's URL construction + its regression test. The integration tests,
  wire docs, and governance filtering are hero-cloud's, already delivered.
- Does not change the wire protocol — aligns the CLI to hero-cloud's existing route.

## Completion Ledger

| # | Item | Status | Evidence |
|---|------|--------|----------|
| 1 | Fix URL double-nesting in `internal/graph/sync.go` | DONE | Push `sync.go:273`, Pull `sync.go:312` append `/graph/...` only |
| 2 | Regression test for correct URL | DONE | `TestSyncEndpoints_DoNotDoubleAPIPrefix`; proven to fail on reintroduced bug |
| 3 | Existing sync tests realigned to real ServerURL shape | DONE | `orgURL` helper + org-scoped fake routes; all graph tests pass |
| 4 | Build + vet + package tests green | DONE | `go build ./...`, `go vet`, `go test ./internal/graph ./internal/cli` all pass |

## Notes

- Notified hero-cloud via `hero peer call --mode=advisory` on 2026-06-24
  (artifact `.hero/peer-calls/18bc2898432b8f308b4b2e4d358b6714.md`; trail entry
  below). hero-cloud CONFIRMED the wire contract matches byte-for-byte — server
  routes `POST/GET /api/v1/orgs/{org}/graph/push|pull` (`cloud/api/graph.go:36-37`)
  — and that their `TestIntegration_GraphURLContract` is the symmetric server-side
  guard to our `TestSyncEndpoints_DoNotDoubleAPIPrefix` (a contract-pinning pair).
- hero-cloud flagged two CLI-side acceptance criteria they SKIPPED (cannot test
  from their repo) as possible follow-up. These are NOT part of the URL-fix slice:
  - AC #5: server unreachable → CLI falls back to local operation with a warning
  - AC #6: auth token expires mid-sync → CLI refreshes the token and retries

## Handoff Trail

- 2026-06-24T18:09:11Z — in ← hero-cloud (peer_id: 5770cae7-b233-45c0-8e5d-765338a6058c)
  mode: async-drop
  originating_spec: cloud-cli-verify
  peer_spec: hero/cloud-cli-verify
  at_commit: ea7ed71

- 2026-06-24T23:37:59Z — out → hero-cloud (peer_id: 5770cae7-b233-45c0-8e5d-765338a6058c)
  mode: advisory
  originating_spec: cloud-cli-verify
  at_commit: ce338c5
  result_ref: .hero/peer-calls/18bc2898432b8f308b4b2e4d358b6714.md
  reason: "Notify hero-cloud that the hero-repo slice of cloud-cli-verify is delivered (graph-sync URL fix), shipped in v0.21.1."

