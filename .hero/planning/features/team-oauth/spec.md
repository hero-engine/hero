---
title: "Team OAuth — GitHub/Google SSO for Team Server Authentication"
type: feature
status: delivering
received_from:
  peer_id: 5770cae7-b233-45c0-8e5d-765338a6058c
  peer_alias_display: hero
  originator_slug: team-oauth
  handed_off_at: 2026-06-24T22:17:29Z
  at_commit: 923152b
  reason: "All changes are hero-side code (internal/serve/, internal/cli/, internal/config/). Needs OAuth flow, JWT issuance, JWTAuthMiddleware, hero connect team command."
---

# Team OAuth — GitHub/Google SSO for Team Server Authentication

## Provenance

Handed off from peer `hero` (peer_id `5770cae7-b233-45c0-8e5d-765338a6058c`).
Originator spec: `team-oauth`.

**Reason:** All changes are hero-side code (internal/serve/, internal/cli/, internal/config/). Needs OAuth flow, JWT issuance, JWTAuthMiddleware, hero connect team command.

## Context

_Scaffolded by `hero handoff`. Flesh out goal, design, and acceptance criteria before delivering._

## Handoff Trail

- 2026-06-24T22:17:29Z — in ← hero-cloud (peer_id: 5770cae7-b233-45c0-8e5d-765338a6058c)
  mode: async-drop
  originating_spec: team-oauth
  peer_spec: hero/team-oauth
  at_commit: 923152b
  reason: "All changes are hero-side code (internal/serve/, internal/cli/, internal/config/). Needs OAuth flow, JWT issuance, JWTAuthMiddleware, hero connect team command."

## Completion Ledger

### Acceptance Criteria

- [x] **AC-1: OAuth login endpoint** — `GET /auth/oauth/login?provider=github&redirect_uri=...` redirects to GitHub/Google OAuth consent URL with CSRF state parameter
- [x] **AC-2: OAuth callback endpoint** — `GET /auth/oauth/callback` exchanges code for access token, fetches user info, enforces org/hosted-domain restriction, issues JWT, redirects to CLI localhost callback
- [x] **AC-3: OAuth config endpoint** — `GET /auth/oauth/config` returns `{"enabled": true, "provider": "github"}` when OAuth is configured
- [x] **AC-4: Token refresh endpoint** — `POST /auth/refresh` validates existing JWT, issues a new one if within 7 days of expiry
- [x] **AC-5: CLI OAuth flow** — `hero connect team <url>` (without `--token`) checks OAuth config, opens browser, starts localhost callback server, receives JWT, stores connection
- [x] **AC-6: State parameter security** — 32-byte random state with 10-minute TTL, validated and consumed on callback
- [x] **AC-7: GitHub org restriction** — When `HERO_OAUTH_ORG` is set, callback verifies user membership via `/user/orgs`
- [x] **AC-8: Google hosted domain restriction** — When `HERO_OAUTH_HOSTED_DOMAIN` is set, callback verifies `hd` claim from userinfo
- [x] **AC-9: Server wiring** — `server.go` reads `HERO_OAUTH_CLIENT_ID`, `HERO_OAUTH_CLIENT_SECRET`, `HERO_OAUTH_PROVIDER` env vars and wires OAuth endpoints
- [x] **AC-10: Tests pass** — 12 tests covering state management, endpoint routing, org restriction logic, token refresh (valid/not-eligible/no-auth)

### Changes

| File | Action | Description |
|---|---|---|
| `internal/serve/oauth.go` | Created | OAuthConfig, OAuthHandler (state management, login redirect, callback, config endpoint), GitHub+Google provider support, RegisterOAuthAPI |
| `internal/serve/oauth_test.go` | Created | 12 tests: state generate/validate/expiry, config endpoint, login redirect (GitHub+Google), callback validation, org restriction, token refresh (valid/not-eligible/no-auth) |
| `internal/serve/api_auth.go` | Modified | Added `POST /auth/refresh` endpoint with 7-day eligibility window; added `time` import |
| `internal/serve/server.go` | Modified | Reads OAuth env vars, constructs OAuthConfig, calls RegisterOAuthAPI when configured |
| `internal/cli/connect_team.go` | Modified | Added OAuth flow: checks `/auth/oauth/config`, starts localhost callback server, opens browser, receives JWT token; refactored into `connectWithToken` and `connectWithOAuth` |
