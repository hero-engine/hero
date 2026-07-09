# Delivery audit — team-oauth

**Audited:** `git show 33a8473` (core impl) + on-disk `internal/serve/server.go`, `internal/cli/connect_team.go` (later touches)
**Verdict:** SHIP
**Surface:** noteworthy

Framing: this is an authorized "complete now, live-verify deferred" delivery. The 12 tests
are unit tests with mocked/absent providers. ACs describing a real external round-trip
(AC-1 redirect target, AC-2 code exchange + user fetch, AC-7 org check, AC-8 hosted domain)
are assessed as **unit-verified, live-verification deferred** — an accepted deferral, not a
defect. Live GitHub/Google verification is still pending human-registered OAuth app credentials.

## Acceptance criteria

- [✓] **AC-1: OAuth login endpoint** — `HandleLogin` (oauth.go:107) requires `redirect_uri`, generates state, redirects (302) to GitHub/Google authorize URL. Tested: `TestOAuthLoginRequiresRedirectURI`, `TestOAuthLoginRedirectsGitHub`, `TestOAuthLoginRedirectsGoogle` assert 302 + URL prefix + client_id + scopes/hd param. Redirect *construction* unit-verified; live provider consent flow deferred.
- [✓] **AC-2: OAuth callback endpoint** — `HandleCallback` (oauth.go:150) validates code+state, dispatches to `githubCallback`/`googleCallback` (exchange → fetch user → org/hd check → `FindOrCreateOAuthUser` → `IssueJWT` 30d → redirect to CLI localhost with token/email/name). Real HTTP calls to provider APIs present (oauth.go:236, 258, 384, 405). Tested only for param/state validation (`TestOAuthCallbackMissingParams`, `TestOAuthCallbackInvalidState`); real code exchange + user fetch is live-deferred (no mocked-provider round-trip test).
- [✓] **AC-3: OAuth config endpoint** — `HandleConfig` (oauth.go:224) returns `{"enabled":true,"provider":...}`. Tested: `TestOAuthConfigEndpoint` asserts 200 + enabled=true + provider=github.
- [✓] **AC-4: Token refresh endpoint** — `handleRefresh` (api_auth.go:175) validates JWT, rejects if >7 days from expiry, else issues new 30d token. Registered at `POST /auth/refresh` (api_auth.go:36). Tested: `TestRefreshTokenValid` (200 + new≠old token), `TestRefreshTokenNotEligible` (400 for 20d token), `TestRefreshTokenNoAuth` (401). Real, asserting.
- [✓] **AC-5: CLI OAuth flow** — `runConnectTeam` → `connectWithOAuth` (connect_team.go:144): checks `/auth/oauth/config`, starts localhost listener on random port, opens browser to login URL, receives token via `/callback`, saves `TeamConnection`, 2-min timeout. Logic present on disk. No unit test for the CLI flow (interactive/browser-driven) — consistent with the deferral.
- [✓] **AC-6: State parameter security** — 32-byte `crypto/rand` state, base64url, stored with createdAt; `validateState` (oauth.go:76) enforces 10-min TTL and consumes via `LoadAndDelete`; background `cleanStates` sweep. Tested: `TestOAuthStateGenerateAndValidate` (valid → URI, second validate fails = consumed), `TestOAuthStateExpiry` (11-min-old → invalid). Real, asserting.
- [~] **AC-7: GitHub org restriction** — Code is real: `githubCheckOrg` (oauth.go:290) GETs `/user/orgs` and case-insensitively matches `HERO_OAUTH_ORG`, else returns "not a member" error; invoked from `githubCallback` when `cfg.Org != ""`. HOWEVER the claimed test `TestOAuthGitHubOrgRestriction` (oauth_test.go:170) does **not** call `githubCheckOrg` — it spins up a mock GitHub server that is never used, then re-implements the matching loop inline against a local slice and asserts on the copy. The production function is not exercised. Code present; org-check logic **live-deferred** and the unit test for it is performative (asserts on a re-implementation, not the real code path).
- [✓] **AC-8: Google hosted domain restriction** — `googleCallback` (oauth.go:360) checks `userInfo["hd"]` against `cfg.HostedDomain`, rejects mismatch; `googleAuthURL` also sets `hd=` param. `hd` param presence tested via `TestOAuthLoginRedirectsGoogle`. The callback-side `hd` claim enforcement is live-deferred (no mocked-userinfo test). Code present.
- [✓] **AC-9: Server wiring** — server.go:404-419 reads `HERO_OAUTH_CLIENT_ID/SECRET/PROVIDER` (all three required to enable), `HERO_OAUTH_ORG`, `HERO_OAUTH_HOSTED_DOMAIN`, `HERO_OAUTH_REDIRECT_URI`; constructs `OAuthConfig`; calls `RegisterOAuthAPI`; logs "OAuth enabled". Verified on disk. No unit test (env-gated wiring); low risk, readable.
- [✓] **AC-10: Tests pass** — `go test ./internal/serve/ -run 'OAuth|Auth|Refresh|State' -count=1` → ok. 12 named OAuth/Auth/Refresh/State tests all PASS; build clean. Test bodies read: most assert on real behavior (state consume/expiry, config JSON, redirect URL contents, refresh eligibility windows). Exception: `TestOAuthGitHubOrgRestriction` asserts on an inline re-implementation, not `githubCheckOrg` (see AC-7).

## Changes

- [✓] `internal/serve/oauth.go` — Created (457 lines). OAuthConfig, OAuthHandler, state mgmt, login/callback/config handlers, GitHub + Google exchange/fetch/restriction helpers, RegisterOAuthAPI. Matches ledger.
- [✓] `internal/serve/oauth_test.go` — Created (324 lines). 12 test funcs as described. Matches ledger, with the AC-7 test caveat noted above.
- [✓] `internal/serve/api_auth.go` — Modified. Added `POST /auth/refresh` route + `handleRefresh` with 7-day window; added `time` import. Matches ledger.
- [✓] `internal/serve/server.go` — Modified. OAuth env wiring + RegisterOAuthAPI call. Matches ledger.
- [✓] `internal/cli/connect_team.go` — Modified. `connectWithOAuth`/`connectWithToken` split, `/auth/oauth/config` probe, localhost callback server, browser open, JWT store. Matches ledger.

Dependency `FindOrCreateOAuthUser` exists at `internal/serve/users.go:92` (not in the ledger's
Changes table — pre-existing or added elsewhere; not claimed here, so no defect).

## Open items

- **AC-7 org restriction** — test does not exercise the real `githubCheckOrg` code path; it asserts on a re-implemented loop while the mock server it creates goes unused. The production code is present and looks correct, but has zero real coverage. Assessment: **soft test** — real code, performative test. Recommend a follow-up test that points `githubCheckOrg` at an httptest server (currently hardcoded to `https://api.github.com/user/orgs`, so this would require URL injection) OR explicit acknowledgment that org enforcement rides on live verification only.
- **Live verification pending (AC-1/AC-2/AC-7/AC-8)** — the real GitHub/Google round-trip (redirect consent, code→token exchange, `/user` + `/user/orgs` + userinfo fetch, `hd` claim) has not run against a human-registered OAuth app. Accepted deferral per delivery framing; flagged so the user/peer know live GitHub/Google verification with real client credentials is still outstanding.

## Audit notes

- Provider API base URLs are hardcoded (`api.github.com`, `oauth2.googleapis.com`, etc.), which is why the org-restriction unit test couldn't hit a mock and fell back to re-implementing the loop. Not a shipping blocker, but it's the reason AC-7 has no genuine coverage — worth knowing if live verification surfaces an org-check bug.
- Not downgrading any AC to `✗`: every claimed endpoint and function exists and does what the AC says. The only performative artifact is the *test* for AC-7, not the code — so this is a `~` (soft coverage) under an authorized-deferral delivery, not a HOLD. Verdict stays SHIP with the two open items surfaced.
- Diff is well-scoped to the spec's 5 named files (plus the spec file itself). No scope drift.
