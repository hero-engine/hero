---
title: "CLI Sync Resilience — Unreachable Fallback & Mid-Sync Token Refresh"
slug: cli-sync-resilience
type: feature
status: planning
created: 2026-06-24
tags: [cli, cloud, sync, auth, resilience]
parent: cloud-cli-verify
---
# CLI Sync Resilience — Unreachable Fallback & Mid-Sync Token Refresh

## Goal

Close the two CLI-side acceptance criteria that hero-cloud's `cloud-cli-verify`
delivery audit SKIPPED (it cannot exercise CLI behavior from its own repo):

- **AC #5** — server unreachable → CLI falls back to local operation with a warning.
- **AC #6** — auth token expires mid-sync → CLI refreshes the token and retries.

These are separate from the graph-sync URL fix already shipped in v0.21.1.

## Background

Surfaced via an advisory `hero peer call` to hero-cloud on 2026-06-24
(`private-peer-result-excluded`) confirming the URL fix and
flagging #5/#6 as the remaining CLI-side items.

## Investigation (what's actually true today)

### AC #5 — unreachable fallback: PARTIALLY satisfied (test gap only)

- **Opportunistic scan path** (`internal/cli/scan_team_sync.go` →
  `internal/cli/scan.go:708-712`): on any push/pull error, `runOpportunisticTeamSync`
  returns `Skipped: true, Reason: "...failed: <err>"`; the scan reports a skipped
  `team-server` step and **continues locally**. This *is* "fall back to local
  operation with a warning" — satisfied behaviorally.
- **Direct commands** `hero sync graph push` / `pull`
  (`internal/cli/sync_graph.go:156,221`): hard-error on failure. Defensible — an
  explicit push to a down server *should* fail loudly, not silently no-op.
- **Gap:** no regression test pins the graceful-degradation behavior, and the
  intended split (best-effort scan vs. loud explicit command) is undocumented.

### AC #6 — mid-sync token refresh: NOT satisfied (real gap)

- Refresh machinery exists — `LoadCloudToken()` and `tryRefresh()`
  (`internal/cli/cloud_auth.go:205-264`) — but the graph-sync path does not use it:
  - `setupGraphSync` (`sync_graph.go:82`) loads the token via `loadCredentials()`
    (raw read), **not** `LoadCloudToken()`, so it skips even the proactive
    "refresh if already expired at load" path.
  - `authTransport.RoundTrip` (`sync_graph.go:142-145`) injects a **static** bearer
    token captured at setup. There is no 401 interception and no refresh-and-retry.
- Result: a token that expires before or during a sync yields a `server 401` error
  from `Store.Push`/`Pull` with no recovery.

## Design

### AC #6 — refreshing auth transport (the substantive work)

Replace the static `authTransport` with one that can refresh once on `401`:

- Hold credentials (access + refresh token + expiry), not just a bare string.
- **Proactive:** route token loading through `LoadCloudToken()` so an already-expired
  token is refreshed before the first request (wire it into `setupGraphSync` and the
  opportunistic path).
- **Reactive:** in `RoundTrip`, on a `401` response, call `tryRefresh` once, persist
  the new creds, replay the request with the new token. Guard against infinite loops
  (single retry) and concurrent refreshes (mutex / singleflight) since push and pull
  may share the client.
- Keep `tryRefresh` as-is; it already persists via `saveCredentials`.

### AC #5 — pin behavior + document the split

- Add tests: opportunistic sync with an unreachable server → scan completes, step
  marked skipped with reason (no error returned). Direct `push`/`pull` against an
  unreachable server → non-nil error.
- Document the intentional contract: implicit/opportunistic sync is best-effort and
  degrades; explicit `hero sync graph push|pull` fails loudly.

## Changes

- `internal/cli/sync_graph.go` — refreshing transport; load via `LoadCloudToken`.
- `internal/cli/cloud_auth.go` — expose creds/refresh hook to the transport as needed
  (minimal surface; reuse `tryRefresh`).
- `internal/cli/scan_team_sync.go` — ensure the opportunistic path also benefits from
  proactive refresh.
- Tests: a 401-then-200 fake server asserting one refresh + successful retry; an
  unreachable-server case for both the scan path (skip+continue) and the direct
  command (error).

## Acceptance Criteria

- WHEN a sync request returns 401 due to an expired access token AND a valid refresh
  token exists THE CLI SHALL refresh once, persist new credentials, retry the request,
  and succeed.
- WHEN refresh fails (no or invalid refresh token) THE CLI SHALL surface a clear
  "run `hero login`" error rather than a bare 401.
- WHEN the access token is already expired before a sync starts THE CLI SHALL refresh
  proactively via `LoadCloudToken` before the first request.
- WHEN the server is unreachable during an opportunistic scan sync THE CLI SHALL skip
  with a visible reason and complete the scan locally (regression test).
- WHEN the server is unreachable during an explicit `hero sync graph push|pull` THE
  CLI SHALL return a non-nil error (regression test).

## Boundaries

- Does NOT change the wire/auth protocol — uses the existing
  `/api/v1/auth/refresh` endpoint and credential file.
- Does NOT add background/automatic token renewal — refresh is on-demand
  (proactive-at-load + reactive-on-401).
- Single retry only; repeated 401s after a refresh surface the error.

## Open Questions

- Does `hero sync cloud` (spec sync) share the same static-token transport and need
  the same treatment, or is it already routed through `LoadCloudToken`? Verify during
  implementation and fold in if the gap is identical.
