# Delivery audit — cli-sync-resilience

**Audited:** unstaged working-tree changes against `ca0eec7` (HEAD of main)
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria

- [✓] AC1: 401 + valid refresh token → refresh once, persist, retry, succeed — `sync_graph.go` `authTransport.RoundTrip` does a single refresh-and-retry on 401 via `refreshIfStale` (mutex-guarded, compares the failed token to avoid double-refresh); `tryRefresh` persists new creds. Covered by `TestAuthTransport_RefreshesOn401AndRetries` (asserts exactly 1 refresh + 2 protected calls + rotated token) and `TestAuthTransport_ReplaysPostBodyAfterRefresh` (POST body rebuilt via GetBody, sent intact on retry).
- [✓] AC2: refresh fails → clear "run `hero login`" error, not a bare 401 — `augmentAuthError` appends the hint to any 401-bearing push/pull error; `TestAugmentAuthError` checks hint-on-401 / no-hint-on-500 / nil-passthrough. The no-refresh-token loop guard (`TestAuthTransport_NoRefreshTokenSurfaces401`) confirms a single request and a surfaced 401.
- [✓] AC3: token already expired before sync → proactive refresh via `LoadCloudToken` — `loadRefreshedCredentials` (`cloud_auth.go`) refreshes an expired-at-rest token before the first request; wired into both `setupGraphSync` and the opportunistic scan path. `TestLoadRefreshedCredentials_ProactiveRefreshWhenExpired` (expired → refreshed) and `_KeepsValidToken` (valid → no refresh call).
- [✓] AC4: unreachable during opportunistic scan → skip with reason, scan completes — `runOpportunisticTeamSync` returns `Skipped + Reason` (pre-existing) and now also proactive-refreshes. `TestRunOpportunisticTeamSync_SkipsWhenUnreachable` asserts no error returned, `Skipped=true`, reason mentions the failure.
- [✓] AC5: unreachable during explicit `hero sync graph push`/`pull` → non-nil error — `Store.Push`/`Pull` surface the dial error, propagated by the command handlers. `TestPush_UnreachableServerReturnsErr` (graph package) pushes to a closed server and asserts a non-nil error.

## Changes

- [✓] `internal/cli/cloud_auth.go` — Added `loadRefreshedCredentials` (proactive refresh-at-load). Reuses existing `tryRefresh`/`saveCredentials`; no protocol change.
- [✓] `internal/cli/sync_graph.go` — Replaced the static `authTransport` with a refreshing one (`newAuthTransport`, `currentToken`, `refreshIfStale`, refresh-and-retry `RoundTrip`); added `augmentAuthError`; `setupGraphSync` now loads via `loadRefreshedCredentials` and guards a nil creds (prior code would nil-deref when logged out).
- [✓] `internal/cli/scan_team_sync.go` — Opportunistic path uses `loadRefreshedCredentials` + `newAuthTransport`.
- [✓] `internal/cli/sync_graph_auth_test.go` — New file, 7 tests (refresh/retry, POST body replay, no-refresh loop guard, proactive refresh, valid-token no-op, opportunistic skip, error hint).
- [✓] `internal/graph/sync_test.go` — Added `TestPush_UnreachableServerReturnsErr`.

## Open items

- `hero sync cloud` (spec sync) was not audited for the same static-token transport — left as a documented Open Question in the spec; this delivery scoped to graph sync where the cloud-cli-verify audit located the gap.

## Audit notes

- **Tests executed and green.** `go build ./...`, `go vet ./internal/cli ./internal/graph`, and `go test ./internal/cli ./internal/graph` all pass; the 8 new tests pass individually. Credential-writing tests isolate `HOME` to a temp dir so the real `~/.hero/credentials.json` is never touched.
- **Unrelated pre-existing build break.** `internal/cli` only compiles with the uncommitted `connect_team.go` WIP (team-connect) set aside — it adds a duplicate `openBrowser` with a clashing signature that breaks the package independently of this work. Verification was run against a tree with that single file temporarily stashed; nothing in this delivery touches it. Flagged to the repo owner.
- **RoundTrip body-replay constraint.** Retry only fires when the request body can be rebuilt (`GetBody` set, true for the bytes/strings readers the sync client uses); a streamed body with no `GetBody` surfaces the 401 rather than risking a half-consumed replay. Acceptable for the current call sites.
